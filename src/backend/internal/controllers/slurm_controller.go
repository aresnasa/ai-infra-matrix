package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SlurmController 提供Slurm集群初始化与仓库配置能力，并包含扩缩容与SaltStack集成
type SlurmController struct {
	slurmSvc   *services.SlurmService
	saltSvc    *services.SaltStackService
	sshSvc     *services.SSHService
	clusterSvc *services.SlurmClusterService
	taskSvc    *services.SlurmTaskService
	config     *config.Config
	db         *gorm.DB
}

func NewSlurmController() *SlurmController {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{} // 使用默认配置
	}

	return &SlurmController{
		slurmSvc:   services.NewSlurmServiceWithDB(database.DB),
		saltSvc:    services.NewSaltStackService(),
		sshSvc:     services.NewSSHService(),
		clusterSvc: services.NewSlurmClusterService(database.DB),
		taskSvc:    services.NewSlurmTaskService(database.DB),
		config:     cfg,
		db:         database.DB,
	}
}

// --- 请求与响应结构 ---

type SSHAuth struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type RepoSetupRequest struct {
	RepoHost    SSHAuth `json:"repoHost"`
	BasePath    string  `json:"basePath"` // e.g. /var/www/html/deb/slurm
	BaseURL     string  `json:"baseURL"`  // e.g. http://repo-host/deb/slurm
	EnableIndex bool    `json:"enableIndex"`
}

type RepoSetupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	BaseURL string `json:"baseURL"`
}

type InitNode struct {
	SSH  SSHAuth `json:"ssh"`
	Role string  `json:"role"` // controller | node
}

type InitNodesRequest struct {
	RepoURL string     `json:"repoURL"`
	Nodes   []InitNode `json:"nodes"`
}

type HostResult struct {
	Host    string `json:"host"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error"`
	TookMS  int64  `json:"tookMs"`
}

type InitNodesResponse struct {
	Success bool         `json:"success"`
	Results []HostResult `json:"results"`
}

// GET /api/slurm/summary
func (c *SlurmController) GetSummary(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 4*time.Second)
	defer cancel()
	sum, err := c.slurmSvc.GetSummary(ctxWithTimeout)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrNotAvailable) {
			status = http.StatusBadGateway
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": sum})
}

// GET /api/slurm/nodes
func (c *SlurmController) GetNodes(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()
	nodes, demo, err := c.slurmSvc.GetNodes(ctxWithTimeout)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrNotAvailable) {
			status = http.StatusBadGateway
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// 增强节点数据：添加 SaltStack minion 状态
	enrichedNodes := c.enrichNodesWithSaltStackStatus(ctx, nodes)

	ctx.JSON(http.StatusOK, gin.H{"data": enrichedNodes, "demo": demo})
}

// enrichNodesWithSaltStackStatus 为节点数据添加 SaltStack 状态信息
func (c *SlurmController) enrichNodesWithSaltStackStatus(ctx *gin.Context, nodes []services.SlurmNode) []map[string]interface{} {
	// 获取 SaltStack 状态
	saltStatus, err := c.saltSvc.GetStatus(ctx)
	if err != nil {
		// 记录错误日志
		logrus.WithFields(logrus.Fields{
			"error":       err.Error(),
			"nodes_count": len(nodes),
		}).Warn("无法获取 SaltStack 状态，所有节点将标记为 unknown")

		// 如果无法获取 Salt 状态，返回转换后的原始节点数据，标记为 API 失败
		result := make([]map[string]interface{}, len(nodes))
		for i, node := range nodes {
			result[i] = map[string]interface{}{
				"name":              node.Name,
				"state":             node.State,
				"cpus":              node.CPUs,
				"memory_mb":         node.MemoryMB,
				"partition":         node.Partition,
				"salt_status":       "unknown",
				"salt_enabled":      false,
				"salt_status_error": "SaltStack API 连接失败",
			}
		}
		return result
	}

	// 创建 minion 状态映射：minion_id -> status
	minionStatusMap := make(map[string]string)

	// 在线的 minions（accepted 且可以 ping 通）
	for _, minionID := range saltStatus.AcceptedKeys {
		// 默认为 accepted，实际在线状态需要 ping 验证
		minionStatusMap[minionID] = "accepted"
	}

	// 未接受的 minions
	for _, minionID := range saltStatus.UnacceptedKeys {
		minionStatusMap[minionID] = "pending"
	}

	// 被拒绝的 minions
	for _, minionID := range saltStatus.RejectedKeys {
		minionStatusMap[minionID] = "rejected"
	}

	// 为每个节点添加 SaltStack 状态
	enrichedNodes := make([]map[string]interface{}, len(nodes))
	for i, node := range nodes {
		// 转换原始节点数据
		enrichedNode := map[string]interface{}{
			"name":      node.Name,
			"state":     node.State,
			"cpus":      node.CPUs,
			"memory_mb": node.MemoryMB,
			"partition": node.Partition,
		}

		// 匹配 minion 状态
		// 尝试多种匹配方式：精确匹配、前缀匹配
		saltStatusStr := "unknown"
		saltMinionID := ""

		nodeName := node.Name
		if nodeName != "" {
			// 精确匹配
			if status, ok := minionStatusMap[nodeName]; ok {
				saltStatusStr = status
				saltMinionID = nodeName
			} else {
				// 尝试模糊匹配（移除域名后缀等）
				shortName := strings.Split(nodeName, ".")[0]
				if status, ok := minionStatusMap[shortName]; ok {
					saltStatusStr = status
					saltMinionID = shortName
				} else {
					// 反向匹配：检查是否有 minion ID 包含节点名
					for minionID, status := range minionStatusMap {
						if strings.Contains(minionID, nodeName) || strings.Contains(nodeName, minionID) {
							saltStatusStr = status
							saltMinionID = minionID
							break
						}
					}
				}
			}
		}

		// 如果仍然是 unknown，记录调试信息
		if saltStatusStr == "unknown" && nodeName != "" {
			logrus.WithFields(logrus.Fields{
				"node_name":    nodeName,
				"minion_count": len(minionStatusMap),
			}).Debug("节点未匹配到 SaltStack Minion ID")
		}

		// 添加 SaltStack 状态字段
		enrichedNode["salt_status"] = saltStatusStr
		enrichedNode["salt_minion_id"] = saltMinionID
		enrichedNode["salt_enabled"] = saltStatusStr != "unknown"

		enrichedNodes[i] = enrichedNode
	}

	return enrichedNodes
}

// GET /api/slurm/jobs
func (c *SlurmController) GetJobs(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()
	jobs, demo, err := c.slurmSvc.GetJobs(ctxWithTimeout)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrNotAvailable) {
			status = http.StatusBadGateway
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": jobs, "demo": demo})
}

// NodeManagementRequest 节点管理请求
type NodeManagementRequest struct {
	NodeNames []string `json:"node_names" binding:"required"` // 节点名称列表
	Action    string   `json:"action" binding:"required"`     // 操作：resume/drain/down/idle/power_down/power_up
	Reason    string   `json:"reason"`                        // 操作原因（可选）
}

// POST /api/slurm/nodes/manage
// 批量管理节点状态
func (c *SlurmController) ManageNodes(ctx *gin.Context) {
	var req NodeManagementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	// 验证操作类型
	validActions := map[string]string{
		"resume":     "RESUME",
		"drain":      "DRAIN",
		"down":       "DOWN",
		"idle":       "RESUME", // IDLE 状态通过 RESUME 实现
		"power_down": "POWER_DOWN",
		"power_up":   "POWER_UP",
	}

	slurmState, ok := validActions[req.Action]
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的操作: %s", req.Action)})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	// 检查是否使用 slurmrestd REST API
	if c.slurmSvc.GetUseSlurmRestd() {
		// 使用 REST API 方式
		var failedNodes []string
		var successCount int

		for _, nodeName := range req.NodeNames {
			update := services.SlurmNodeUpdate{
				State:  slurmState,
				Reason: req.Reason,
			}
			err := c.slurmSvc.UpdateNodeViaAPI(ctxWithTimeout, nodeName, update)
			if err != nil {
				failedNodes = append(failedNodes, nodeName)
			} else {
				successCount++
			}
		}

		if len(failedNodes) > 0 {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success":      false,
				"message":      fmt.Sprintf("部分节点操作失败: %d 成功, %d 失败", successCount, len(failedNodes)),
				"failed_nodes": failedNodes,
				"action":       req.Action,
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": fmt.Sprintf("成功对 %d 个节点执行 %s 操作", len(req.NodeNames), req.Action),
				"nodes":   req.NodeNames,
				"action":  req.Action,
			})
		}
		return
	}

	// 使用 SSH 方式 (原有逻辑)
	// 构建 scontrol 命令
	nodeList := strings.Join(req.NodeNames, ",")
	command := fmt.Sprintf("scontrol update NodeName=%s State=%s", nodeList, slurmState)

	// DOWN 和 DRAIN 操作必须提供 Reason
	if slurmState == "DOWN" || slurmState == "DRAIN" {
		reason := req.Reason
		if reason == "" {
			// 如果没有提供 reason，使用默认值
			reason = fmt.Sprintf("由管理员通过 Web 界面执行 %s 操作", slurmState)
		}
		command += fmt.Sprintf(" Reason='%s'", reason)
	} else if req.Reason != "" {
		// 其他操作：如果提供了 reason，也添加上
		command += fmt.Sprintf(" Reason='%s'", req.Reason)
	}

	// 执行命令
	output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, command)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("执行命令失败: %v", err),
			"output":  output,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功将 %d 个节点设置为 %s 状态", len(req.NodeNames), slurmState),
		"nodes":   req.NodeNames,
		"action":  req.Action,
		"output":  output,
	})
}

// JobManagementRequest 作业管理请求
type JobManagementRequest struct {
	JobIDs []string `json:"job_ids" binding:"required"` // 作业ID列表
	Action string   `json:"action" binding:"required"`  // 操作：cancel/hold/release/suspend/resume/requeue
	Signal string   `json:"signal"`                     // 信号类型（用于cancel）
}

// POST /api/slurm/jobs/manage
// 批量管理作业
func (c *SlurmController) ManageJobs(ctx *gin.Context) {
	var req JobManagementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	// 检查是否使用 slurmrestd REST API
	if c.slurmSvc.GetUseSlurmRestd() {
		// 使用 REST API 方式
		var failedJobs []string
		var successCount int

		for _, jobID := range req.JobIDs {
			var err error

			switch req.Action {
			case "cancel":
				err = c.slurmSvc.CancelJobViaAPI(ctxWithTimeout, jobID, req.Signal)
			case "hold":
				err = c.slurmSvc.UpdateJobViaAPI(ctxWithTimeout, jobID, "hold")
			case "release":
				err = c.slurmSvc.UpdateJobViaAPI(ctxWithTimeout, jobID, "release")
			case "suspend":
				err = c.slurmSvc.UpdateJobViaAPI(ctxWithTimeout, jobID, "suspend")
			case "resume":
				err = c.slurmSvc.UpdateJobViaAPI(ctxWithTimeout, jobID, "resume")
			case "requeue":
				err = c.slurmSvc.UpdateJobViaAPI(ctxWithTimeout, jobID, "requeue")
			default:
				ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的操作: %s", req.Action)})
				return
			}

			if err != nil {
				failedJobs = append(failedJobs, jobID)
			} else {
				successCount++
			}
		}

		if len(failedJobs) > 0 {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success":     false,
				"message":     fmt.Sprintf("部分作业操作失败: %d 成功, %d 失败", successCount, len(failedJobs)),
				"failed_jobs": failedJobs,
				"action":      req.Action,
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": fmt.Sprintf("成功对 %d 个作业执行 %s 操作", len(req.JobIDs), req.Action),
				"jobs":    req.JobIDs,
				"action":  req.Action,
			})
		}
		return
	}

	// 使用 SSH 方式 (原有逻辑)
	var command string
	jobList := strings.Join(req.JobIDs, ",")

	switch req.Action {
	case "cancel":
		command = fmt.Sprintf("scancel %s", jobList)
		if req.Signal != "" {
			command = fmt.Sprintf("scancel -s %s %s", req.Signal, jobList)
		}
	case "hold":
		command = fmt.Sprintf("scontrol hold %s", jobList)
	case "release":
		command = fmt.Sprintf("scontrol release %s", jobList)
	case "suspend":
		command = fmt.Sprintf("scontrol suspend %s", jobList)
	case "resume":
		command = fmt.Sprintf("scontrol resume %s", jobList)
	case "requeue":
		command = fmt.Sprintf("scontrol requeue %s", jobList)
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的操作: %s", req.Action)})
		return
	}

	output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, command)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("执行命令失败: %v", err),
			"output":  output,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功对 %d 个作业执行 %s 操作", len(req.JobIDs), req.Action),
		"jobs":    req.JobIDs,
		"action":  req.Action,
		"output":  output,
	})
}

// GET /api/slurm/partitions
// 获取分区列表和详细信息
func (c *SlurmController) GetPartitions(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, "sinfo -h -o '%P|%a|%l|%D|%T|%N'")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取分区信息失败: %v", err)})
		return
	}

	partitions := []gin.H{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 6 {
			partitions = append(partitions, gin.H{
				"name":       strings.TrimSuffix(parts[0], "*"),
				"avail":      parts[1],
				"timelimit":  parts[2],
				"nodes":      parts[3],
				"state":      parts[4],
				"nodelist":   parts[5],
				"is_default": strings.HasSuffix(parts[0], "*"),
			})
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"data": partitions})
}

// ScalingRequest 扩缩容请求
type ScalingRequest struct {
	Nodes []services.NodeConfig `json:"nodes" binding:"required"`
}

// ScaleDownRequest 缩容请求
type ScaleDownRequest struct {
	NodeIDs []string `json:"node_ids" binding:"required"`
}

// GET /api/slurm/scaling/status
func (c *SlurmController) GetScalingStatus(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	status, err := c.slurmSvc.GetScalingStatus(ctxWithTimeout)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": status})
}

// POST /api/slurm/scaling/scale-up/async
func (c *SlurmController) ScaleUpAsync(ctx *gin.Context) {
	var req ScalingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前用户信息
	userID, exists := middleware.GetCurrentUserID(ctx)
	if !exists {
		// 如果没有用户信息，使用系统用户ID 1（需要确保数据库中存在）
		userID = 1
		logrus.Warn("ScaleUpAsync: 未找到用户信息，使用系统用户ID 1")
	}

	// 构建目标节点列表
	targetNodes := make([]string, len(req.Nodes))
	for i, node := range req.Nodes {
		targetNodes[i] = node.Host
	}

	// 创建数据库任务记录
	taskReq := services.CreateTaskRequest{
		Name:        "SLURM集群扩容",
		Type:        "scale_up",
		UserID:      userID,
		TargetNodes: targetNodes,
		Parameters: map[string]interface{}{
			"nodes":       req.Nodes,
			"node_count":  len(req.Nodes),
			"salt_master": getSaltStackMasterHost(),
			"apphub_url":  getAppHubBaseURL(),
		},
		Tags:     []string{"scale-up", "slurm"},
		Priority: 5, // 普通优先级
	}

	logrus.WithFields(logrus.Fields{
		"user_id":      userID,
		"target_nodes": targetNodes,
		"node_count":   len(req.Nodes),
	}).Info("创建扩容任务记录")

	dbTask, err := c.taskSvc.CreateTask(ctx.Request.Context(), taskReq)
	if err != nil {
		logrus.WithError(err).Error("创建任务记录失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务记录失败: " + err.Error()})
		return
	}

	taskID := dbTask.TaskID

	pm := services.GetProgressManager()
	op := pm.Start(taskID, "开始扩容Slurm节点")

	go func(opID string, r ScalingRequest, dbTaskID string) {
		bgCtx := context.Background()
		failed := false
		var finalError string

		defer func() {
			// 完成内存任务
			pm.Complete(opID, failed, "扩容完成")

			// 更新数据库任务状态
			status := "completed"
			if failed {
				status = "failed"
			}
			if err := c.taskSvc.UpdateTaskStatus(bgCtx, dbTaskID, status, finalError); err != nil {
				fmt.Printf("更新任务状态失败: %v\n", err)
			}
		}()

		// 启动任务
		if err := c.taskSvc.StartTask(bgCtx, dbTaskID); err != nil {
			fmt.Printf("启动任务失败: %v\n", err)
		}

		// 转换节点配置
		connections := make([]services.SSHConnection, len(r.Nodes))
		for i, node := range r.Nodes {
			connections[i] = services.SSHConnection{
				Host:     node.Host,
				Port:     node.Port,
				User:     node.User,
				KeyPath:  node.KeyPath,
				Password: node.Password,
			}
		}

		// SaltStack部署配置
		saltConfig := services.SaltStackDeploymentConfig{
			MasterHost: getSaltStackMasterHost(), // 从环境变量读取
			MasterPort: 4506,
			AutoAccept: true,
			AppHubURL:  getAppHubBaseURL(), // 离线/内网安装支持
		}

		total := float64(len(connections))
		for i, conn := range connections {
			host := conn.Host
			progress := float64(i) / total * 100
			pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: fmt.Sprintf("deploy-minion-%s", host), Message: "部署SaltStack Minion", Host: host, Progress: progress})
			c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, progress, fmt.Sprintf("部署Minion到%s", host))
		}

		// 并发部署SaltStack Minion
		results, err := c.sshSvc.DeploySaltMinion(context.Background(), connections, saltConfig)
		if err != nil {
			failed = true
			finalError = err.Error()
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "deploy-minions", Message: err.Error()})
			c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "error", "deploy-minions", "部署Minion失败: "+err.Error(), "", 0, nil)
			return
		}

		successCount := 0
		failedCount := 0
		for i, result := range results {
			host := connections[i].Host
			progress := float64(i+1) / total * 100
			if result.Success {
				successCount++
				pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: fmt.Sprintf("deploy-minion-%s", host), Message: "Minion部署成功", Host: host, Progress: progress})
				c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "success", fmt.Sprintf("deploy-minion-%s", host), "Minion部署成功", host, progress, nil)
			} else {
				failed = true
				failedCount++
				finalError = result.Error
				pm.Emit(opID, services.ProgressEvent{Type: "error", Step: fmt.Sprintf("deploy-minion-%s", host), Message: result.Error, Host: host, Progress: progress})
				c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "error", fmt.Sprintf("deploy-minion-%s", host), "Minion部署失败: "+result.Error, host, progress, nil)
			}
		}

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "add-nodes-to-cluster", Message: "添加节点到SLURM集群数据库"})
		c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, 60, "添加节点到集群数据库")

		// 添加节点到数据库
		for _, node := range r.Nodes {
			if err := c.addNodeToCluster(node.Host, node.Port, node.User, "compute"); err != nil {
				failed = true
				finalError = err.Error()
				pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "add-nodes-to-cluster", Message: fmt.Sprintf("添加节点 %s 失败: %v", node.Host, err)})
				c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "error", "add-nodes", fmt.Sprintf("添加节点%s失败: %v", node.Host, err), node.Host, 60, nil)
				continue
			}
			pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: fmt.Sprintf("add-node-%s", node.Host), Message: fmt.Sprintf("节点 %s 已添加到集群", node.Host), Host: node.Host})
			c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "success", "add-node", fmt.Sprintf("节点%s已添加到集群", node.Host), node.Host, 60, nil)
		}

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "scale-up-slurm", Message: "执行Slurm扩容"})
		c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, 80, "执行SLURM扩容")

		// 执行扩容操作
		scaleResults, err := c.slurmSvc.ScaleUp(context.Background(), r.Nodes)
		if err != nil {
			failed = true
			finalError = err.Error()
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "scale-up-slurm", Message: err.Error()})
			c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "error", "scale-up", "SLURM扩容失败: "+err.Error(), "", 80, nil)
			return
		}

		pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "scale-up-slurm", Message: "Slurm扩容完成", Data: scaleResults})
		c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, 100, "扩容完成")
		c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "success", "scale-up", "SLURM扩容成功", "", 100, scaleResults)
	}(op.ID, req, taskID)

	ctx.JSON(http.StatusAccepted, gin.H{
		"opId":   op.ID,
		"taskId": taskID,
	})
}

// POST /api/slurm/scaling/scale-up
func (c *SlurmController) ScaleUp(ctx *gin.Context) {
	var req ScalingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 300*time.Second)
	defer cancel()

	// 转换节点配置
	connections := make([]services.SSHConnection, len(req.Nodes))
	for i, node := range req.Nodes {
		connections[i] = services.SSHConnection{
			Host:     node.Host,
			Port:     node.Port,
			User:     node.User,
			KeyPath:  node.KeyPath,
			Password: node.Password,
		}
	}

	// SaltStack部署配置
	saltConfig := services.SaltStackDeploymentConfig{
		MasterHost: getSaltStackMasterHost(), // 从环境变量读取
		MasterPort: 4506,
		AutoAccept: true,
		AppHubURL:  getAppHubBaseURL(), // 离线/内网安装支持
	}

	// 并发部署SaltStack Minion
	results, err := c.sshSvc.DeploySaltMinion(ctxWithTimeout, connections, saltConfig)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 执行扩容操作
	scaleResults, err := c.slurmSvc.ScaleUp(ctxWithTimeout, req.Nodes)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"saltstack_deployment": results,
		"scale_results":        scaleResults,
	})
}

// POST /api/slurm/scaling/scale-down
func (c *SlurmController) ScaleDown(ctx *gin.Context) {
	var req ScaleDownRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
	defer cancel()

	results, err := c.slurmSvc.ScaleDown(ctxWithTimeout, req.NodeIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": results})
}

// POST /api/slurm/nodes/health-check
// @Summary 检测并修复节点状态
// @Description 检测指定节点的健康状态，如果发现异常会尝试自动修复
// @Tags SLURM
// @Accept json
// @Produce json
// @Param request body object{node_name=string,max_retries=int} true "节点名称和最大重试次数"
// @Success 200 {object} object{success=bool,message=string,node_name=string,status=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string,details=string}
// @Router /api/slurm/nodes/health-check [post]
func (c *SlurmController) HealthCheckNode(ctx *gin.Context) {
	var req struct {
		NodeName   string `json:"node_name" binding:"required"`
		MaxRetries int    `json:"max_retries"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 默认重试3次
	if req.MaxRetries <= 0 {
		req.MaxRetries = 3
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), time.Duration(req.MaxRetries*10)*time.Second)
	defer cancel()

	// 执行健康检测
	err := c.slurmSvc.DetectAndFixNodeState(ctxWithTimeout, req.NodeName, req.MaxRetries)

	if err != nil {
		// 检测失败，返回错误信息和建议
		ctx.JSON(http.StatusOK, gin.H{
			"success":   false,
			"node_name": req.NodeName,
			"status":    "unhealthy",
			"message":   "节点健康检测失败",
			"details":   err.Error(),
			"suggestions": []string{
				"1. 检查节点网络连接: ping " + req.NodeName,
				"2. 确认slurmd服务状态: ssh " + req.NodeName + " systemctl status slurmd",
				"3. 检查slurm配置: ssh " + req.NodeName + " slurmd -C",
				"4. 手动激活节点: scontrol update NodeName=" + req.NodeName + " State=RESUME",
			},
		})
		return
	}

	// 检测成功
	ctx.JSON(http.StatusOK, gin.H{
		"success":   true,
		"node_name": req.NodeName,
		"status":    "healthy",
		"message":   "节点健康检测通过，状态正常",
	})
}

// GET /api/slurm/node-templates
func (c *SlurmController) GetNodeTemplates(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	templates, err := c.slurmSvc.GetNodeTemplates(ctxWithTimeout)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": templates})
}

// POST /api/slurm/node-templates
func (c *SlurmController) CreateNodeTemplate(ctx *gin.Context) {
	var template services.NodeTemplate
	if err := ctx.ShouldBindJSON(&template); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err := c.slurmSvc.CreateNodeTemplate(ctxWithTimeout, &template)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": template})
}

// PUT /api/slurm/node-templates/:id
func (c *SlurmController) UpdateNodeTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	var template services.NodeTemplate
	if err := ctx.ShouldBindJSON(&template); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err := c.slurmSvc.UpdateNodeTemplate(ctxWithTimeout, id, &template)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": template})
}

// DELETE /api/slurm/node-templates/:id
func (c *SlurmController) DeleteNodeTemplate(ctx *gin.Context) {
	id := ctx.Param("id")

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err := c.slurmSvc.DeleteNodeTemplate(ctxWithTimeout, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "模板删除成功"})
}

// GET /api/slurm/saltstack/integration
func (c *SlurmController) GetSaltStackIntegration(ctx *gin.Context) {
	// 使用 saltSvc 服务获取状态（更快，有缓存）
	status, err := c.saltSvc.GetStatus(ctx)
	if err != nil {
		// 如果连接失败，返回不可用状态
		ctx.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"enabled":       false,
				"master_status": "unavailable",
				"api_status":    "unavailable",
				"minions": map[string]interface{}{
					"total":   0,
					"online":  0,
					"offline": 0,
				},
				"minion_list":  []interface{}{},
				"recent_jobs":  0,
				"services":     map[string]string{"salt-api": "unavailable"},
				"last_updated": time.Now(),
				"error":        err.Error(),
			},
		})
		return
	}

	// 转换并返回真实数据
	result := c.convertSaltStatusToIntegration(status)
	ctx.JSON(http.StatusOK, gin.H{"data": result})
}

// getRealSaltStackStatus 直接获取 Salt API 状态（不使用 saltSvc）
func (c *SlurmController) getRealSaltStackStatus(ctx *gin.Context) (*services.SaltStackStatus, error) {
	// 从环境变量读取 Salt API 配置
	saltAPIURL := c.getSaltAPIURL()
	username := c.getEnv("SALT_API_USERNAME", "saltapi")
	password := c.getEnv("SALT_API_PASSWORD", "saltapi123")
	eauth := c.getEnv("SALT_API_EAUTH", "file")
	timeout := c.getEnvDuration("SALT_API_TIMEOUT", 8*time.Second)

	client := &http.Client{Timeout: timeout}

	// 1. 认证
	authReq, _ := http.NewRequest("POST", saltAPIURL+"/login", strings.NewReader(fmt.Sprintf(
		`{"username":"%s","password":"%s","eauth":"%s"}`,
		username, password, eauth,
	)))
	authReq.Header.Set("Content-Type", "application/json")

	authResp, err := client.Do(authReq)
	if err != nil {
		return nil, fmt.Errorf("auth request failed: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth failed with status: %d", authResp.StatusCode)
	}

	var authResult map[string]interface{}
	if err := json.NewDecoder(authResp.Body).Decode(&authResult); err != nil {
		return nil, fmt.Errorf("failed to decode auth response: %v", err)
	}

	// 提取 token
	var token string
	if returnData, ok := authResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if t, ok := tokenData["token"].(string); ok {
				token = t
			}
		}
	}

	if token == "" {
		return nil, fmt.Errorf("no token in auth response")
	}

	// 2. 获取 minions
	minionsReq, _ := http.NewRequest("GET", saltAPIURL+"/minions", nil)
	minionsReq.Header.Set("X-Auth-Token", token)

	minionsResp, err := client.Do(minionsReq)
	if err != nil {
		return nil, fmt.Errorf("minions request failed: %v", err)
	}
	defer minionsResp.Body.Close()

	var minionsResult map[string]interface{}
	json.NewDecoder(minionsResp.Body).Decode(&minionsResult)

	// 3. Ping minions
	pingReq, _ := http.NewRequest("POST", saltAPIURL+"/", strings.NewReader(
		`{"client":"local","tgt":"*","fun":"test.ping"}`,
	))
	pingReq.Header.Set("X-Auth-Token", token)
	pingReq.Header.Set("Content-Type", "application/json")

	pingResp, err := client.Do(pingReq)
	if err != nil {
		return nil, fmt.Errorf("ping request failed: %v", err)
	}
	defer pingResp.Body.Close()

	var pingResult map[string]interface{}
	json.NewDecoder(pingResp.Body).Decode(&pingResult)

	// 解析结果
	var acceptedKeys []string
	connectedCount := 0

	if returnData, ok := minionsResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if minions, ok := returnData[0].(map[string]interface{}); ok {
			for minionID := range minions {
				acceptedKeys = append(acceptedKeys, minionID)
			}
		}
	}

	if returnData, ok := pingResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if minions, ok := returnData[0].(map[string]interface{}); ok {
			connectedCount = len(minions)
		}
	}

	return &services.SaltStackStatus{
		Status:           "connected",
		ConnectedMinions: connectedCount,
		AcceptedKeys:     acceptedKeys,
		UnacceptedKeys:   []string{},
		RejectedKeys:     []string{},
		Services: map[string]string{
			"salt-master": "running",
			"salt-api":    "running",
		},
		LastUpdated: time.Now(),
		Demo:        false,
	}, nil
} // getSaltStackStatusFromHandler 从 SaltStack handler 获取真实状态
func (c *SlurmController) getSaltStackStatusFromHandler(ctx *gin.Context) map[string]interface{} {
	// 构造内部请求
	req, err := http.NewRequestWithContext(ctx.Request.Context(), "GET", "http://backend:8082/api/saltstack/status", nil)
	if err != nil {
		return nil
	}

	// 复制认证头
	if authHeader := ctx.GetHeader("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	// 提取 data 字段
	if data, ok := result["data"].(map[string]interface{}); ok {
		return c.convertToIntegrationFormat(data)
	}

	return nil
}

// convertToIntegrationFormat 转换 SaltStack 状态数据为集成格式
func (c *SlurmController) convertToIntegrationFormat(statusData map[string]interface{}) map[string]interface{} {
	// 提取字段
	masterStatus := getStringField(statusData, "master_status", "status", "unknown")
	apiStatus := getStringField(statusData, "api_status", "unknown")
	connectedMinions := getIntField(statusData, "connected_minions", "minions_up", 0)

	// 获取密钥列表
	acceptedKeys := getStringArrayField(statusData, "accepted_keys", []string{})
	unacceptedKeys := getStringArrayField(statusData, "unaccepted_keys", []string{})

	totalMinions := len(acceptedKeys) + len(unacceptedKeys)
	onlineMinions := connectedMinions
	if onlineMinions > totalMinions {
		onlineMinions = totalMinions
	}
	offlineMinions := totalMinions - onlineMinions

	// 构建 minion 列表
	minionList := []map[string]interface{}{}
	for _, minionID := range acceptedKeys {
		minionList = append(minionList, map[string]interface{}{
			"id":     minionID,
			"name":   minionID,
			"status": "online",
		})
	}
	for _, minionID := range unacceptedKeys {
		minionList = append(minionList, map[string]interface{}{
			"id":     minionID,
			"name":   minionID,
			"status": "pending",
		})
	}

	// 获取服务状态
	services := map[string]string{}
	if srvData, ok := statusData["services"].(map[string]interface{}); ok {
		for k, v := range srvData {
			if str, ok := v.(string); ok {
				services[k] = str
			}
		}
	}
	if len(services) == 0 {
		services = map[string]string{
			"salt-master": masterStatus,
			"salt-api":    apiStatus,
		}
	}

	return map[string]interface{}{
		"enabled":       masterStatus == "running" || masterStatus == "connected",
		"master_status": masterStatus,
		"api_status":    apiStatus,
		"minions": map[string]interface{}{
			"total":   totalMinions,
			"online":  onlineMinions,
			"offline": offlineMinions,
		},
		"minion_list":  minionList,
		"recent_jobs":  0,
		"services":     services,
		"last_updated": time.Now(),
		"demo":         false,
	}
}

// convertSaltStatusToIntegration 转换 SaltStackStatus 结构为集成格式
func (c *SlurmController) convertSaltStatusToIntegration(status *services.SaltStackStatus) map[string]interface{} {
	totalMinions := len(status.AcceptedKeys) + len(status.UnacceptedKeys)
	onlineMinions := status.ConnectedMinions
	if onlineMinions > totalMinions {
		onlineMinions = totalMinions
	}
	offlineMinions := totalMinions - onlineMinions

	minionList := []map[string]interface{}{}
	for _, minionID := range status.AcceptedKeys {
		minionList = append(minionList, map[string]interface{}{
			"id":     minionID,
			"name":   minionID,
			"status": "online",
		})
	}
	for _, minionID := range status.UnacceptedKeys {
		minionList = append(minionList, map[string]interface{}{
			"id":     minionID,
			"name":   minionID,
			"status": "pending",
		})
	}

	return map[string]interface{}{
		"enabled":       status.Status == "running" && !status.Demo,
		"master_status": status.Status,
		"api_status": func() string {
			if status.Demo {
				return "unavailable"
			}
			if status.Status == "running" {
				return "connected"
			}
			return "disconnected"
		}(),
		"minions": map[string]interface{}{
			"total":   totalMinions,
			"online":  onlineMinions,
			"offline": offlineMinions,
		},
		"minion_list":  minionList,
		"recent_jobs":  0,
		"services":     status.Services,
		"last_updated": status.LastUpdated,
		"demo":         status.Demo,
	}
}

// 辅助函数
func getStringField(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	if len(keys) > 0 {
		return keys[len(keys)-1] // 返回最后一个作为默认值
	}
	return "unknown"
}

func getIntField(data map[string]interface{}, keys ...interface{}) int {
	defaultVal := 0
	for i, key := range keys {
		// 最后一个参数如果是int，作为默认值
		if i == len(keys)-1 {
			if intVal, ok := key.(int); ok {
				defaultVal = intVal
				break
			}
		}

		if keyStr, ok := key.(string); ok {
			if val, ok := data[keyStr]; ok {
				switch v := val.(type) {
				case int:
					return v
				case float64:
					return int(v)
				case int64:
					return int(v)
				}
			}
		}
	}
	return defaultVal
}

func getStringArrayField(data map[string]interface{}, key string, defaultVal []string) []string {
	if val, ok := data[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if arr, ok := val.([]string); ok {
			return arr
		}
	}
	return defaultVal
}

// POST /api/slurm/saltstack/deploy-minion
func (c *SlurmController) DeploySaltMinion(ctx *gin.Context) {
	var req services.NodeConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 120*time.Second)
	defer cancel()

	connection := services.SSHConnection{
		Host:       req.Host,
		Port:       req.Port,
		User:       req.User,
		KeyPath:    req.KeyPath,
		PrivateKey: req.PrivateKey,
		Password:   req.Password,
	}

	saltConfig := services.SaltStackDeploymentConfig{
		MasterHost: getSaltStackMasterHost(), // 从环境变量读取
		MasterPort: 4506,
		MinionID:   req.MinionID,
		AutoAccept: true,
		AppHubURL:  getAppHubBaseURL(), // 离线/内网安装支持
	}

	results, err := c.sshSvc.DeploySaltMinion(ctxWithTimeout, []services.SSHConnection{connection}, saltConfig)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(results) > 0 {
		ctx.JSON(http.StatusOK, gin.H{"data": results[0]})
	} else {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "部署失败"})
	}
}

// POST /api/slurm/ssh/test-connection
func (c *SlurmController) TestSSHConnection(ctx *gin.Context) {
	var req services.NodeConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置合理的超时
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 15*time.Second)
	defer cancel()

	connection := services.SSHConnection{
		Host:       req.Host,
		Port:       req.Port,
		User:       req.User,
		KeyPath:    req.KeyPath,
		PrivateKey: req.PrivateKey,
		Password:   req.Password,
	}

	// 测试SSH连接并执行简单命令
	testCommand := "whoami && uname -a && echo 'SSH连接测试成功'"
	results, err := c.sshSvc.ExecuteCommandOnHosts(ctxWithTimeout, []services.SSHConnection{connection}, testCommand)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(results) == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "未获取到测试结果"})
		return
	}

	result := results[0]
	if !result.Success {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"error":    result.Error,
			"output":   result.Output,
			"host":     result.Host,
			"duration": result.Duration.Milliseconds(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "SSH连接测试成功",
		"output":   result.Output,
		"host":     result.Host,
		"duration": result.Duration.Milliseconds(),
	})
}

// BatchSSHTestRequest 批量SSH测试请求
type BatchSSHTestRequest struct {
	Nodes []services.NodeConfig `json:"nodes" binding:"required,min=1,max=100"`
}

// POST /api/slurm/ssh/test-batch
// 批量并发测试SSH连接（支持最多100个节点）
func (c *SlurmController) TestBatchSSHConnection(ctx *gin.Context) {
	var req BatchSSHTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置超时时间，给每个节点15秒，但总时间不超过2分钟
	timeout := time.Duration(min(len(req.Nodes)*15, 120)) * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), timeout)
	defer cancel()

	// 构建SSH连接列表
	connections := make([]services.SSHConnection, 0, len(req.Nodes))
	for _, node := range req.Nodes {
		connections = append(connections, services.SSHConnection{
			Host:       node.Host,
			Port:       node.Port,
			User:       node.User,
			KeyPath:    node.KeyPath,
			PrivateKey: node.PrivateKey,
			Password:   node.Password,
		})
	}

	// 并发执行SSH测试
	testCommand := "whoami && uname -a && echo 'SSH连接测试成功'"
	results, err := c.sshSvc.ExecuteCommandOnHosts(ctxWithTimeout, connections, testCommand)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"results": results,
		})
		return
	}

	// 统计结果
	successCount := 0
	failCount := 0
	resultDetails := make([]gin.H, 0, len(results))

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failCount++
		}

		resultDetails = append(resultDetails, gin.H{
			"host":     result.Host,
			"success":  result.Success,
			"message":  result.Error,
			"output":   result.Output,
			"duration": result.Duration.Milliseconds(),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":       successCount > 0 && failCount == 0,
		"total":         len(results),
		"success_count": successCount,
		"fail_count":    failCount,
		"results":       resultDetails,
	})
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// POST /api/slurm/saltstack/execute/async
func (c *SlurmController) ExecuteSaltCommandAsync(ctx *gin.Context) {
	var req struct {
		Command string   `json:"command" binding:"required"`
		Targets []string `json:"targets"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pm := services.GetProgressManager()
	op := pm.Start("slurm:execute-command", "开始执行SaltStack命令")

	go func(opID string, r struct {
		Command string   `json:"command" binding:"required"`
		Targets []string `json:"targets"`
	}) {
		failed := false
		defer func() {
			pm.Complete(opID, failed, "命令执行完成")
		}()

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "execute", Message: "执行命令: " + r.Command})
		result, err := c.saltSvc.ExecuteCommand(context.Background(), r.Command, r.Targets)
		if err != nil {
			failed = true
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "execute", Message: err.Error()})
			return
		}

		pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "execute", Message: "命令执行成功", Data: result})
	}(op.ID, req)

	ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// POST /api/slurm/saltstack/execute
func (c *SlurmController) ExecuteSaltCommand(ctx *gin.Context) {
	var req struct {
		// 兼容两种格式：老格式(command/targets) 和 新格式(target/function/arguments)
		Command   string   `json:"command"`
		Targets   []string `json:"targets"`
		Target    string   `json:"target"`
		Function  string   `json:"function"`
		Arguments string   `json:"arguments"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数解析失败: " + err.Error()})
		return
	}

	// 确定实际的命令和目标
	var command string
	var targets []string

	// 新格式优先：使用 function 字段
	if req.Function != "" {
		command = req.Function
		if req.Target != "" {
			targets = []string{req.Target}
		}
	} else if req.Command != "" {
		// 兼容老格式
		command = req.Command
		targets = req.Targets
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少必需参数: 请提供 function 或 command"})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
	defer cancel()

	result, err := c.saltSvc.ExecuteCommand(ctxWithTimeout, command, targets)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "执行命令失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": result})
}

// GET /api/slurm/saltstack/jobs
func (c *SlurmController) GetSaltJobs(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	jobs, err := c.saltSvc.GetJobs(ctxWithTimeout)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": jobs})
}

// --- 通用进度：SSE 与快照 ---

// GET /api/slurm/progress/:opId
func (c *SlurmController) GetProgress(ctx *gin.Context) {
	opID := ctx.Param("opId")
	pm := services.GetProgressManager()
	snap, ok := pm.Snapshot(opID)
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": snap})
}

// GET /api/slurm/tasks - 获取所有任务列表
func (c *SlurmController) GetTasks(ctx *gin.Context) {
	// 解析查询参数
	var req services.TaskListRequest
	req.Page = 1
	req.PageSize = 20

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("userID")
	if exists {
		if uid, ok := userID.(uint); ok {
			req.UserID = &uid
		}
	}

	// 查询数据库中的持久化任务记录
	dbTasks, total, err := c.taskSvc.ListTasks(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询任务列表失败: " + err.Error()})
		return
	}

	// 获取内存中的运行时任务
	pm := services.GetProgressManager()
	runtimeTasks := pm.ListOperations()

	// 合并并格式化任务数据
	taskList := make([]gin.H, 0, len(dbTasks)+len(runtimeTasks))

	// 添加数据库任务
	for _, task := range dbTasks {
		taskData := gin.H{
			"id":            task.TaskID,
			"name":          task.Name,
			"type":          task.Type,
			"status":        task.Status,
			"progress":      task.Progress,
			"user_id":       task.UserID,
			"user_name":     "",
			"target_nodes":  task.TargetNodes,
			"nodes_total":   task.NodesTotal,
			"nodes_success": task.NodesSuccess,
			"nodes_failed":  task.NodesFailed,
			"current_step":  task.StepsCurrent,
			"steps_total":   task.StepsTotal,
			"steps_count":   task.StepsCount,
			"error_message": task.ErrorMessage,
			"tags":          task.Tags,
			"priority":      task.Priority,
			"retry_count":   task.RetryCount,
			"max_retries":   task.MaxRetries,
			"created_at":    task.CreatedAt.Unix(),
			"started_at":    0,
			"completed_at":  0,
			"duration":      task.GetFormattedDuration(),
			"success_rate":  task.GetSuccessRate(),
			"source":        "database",
		}

		if task.User != nil {
			taskData["user_name"] = task.User.Username
		}

		if task.StartedAt != nil {
			taskData["started_at"] = task.StartedAt.Unix()
		}

		if task.CompletedAt != nil {
			taskData["completed_at"] = task.CompletedAt.Unix()
		}

		taskList = append(taskList, taskData)
	}

	// 添加运行时任务（内存中的）- 创建快照以获取正确的时间戳格式
	for _, task := range runtimeTasks {
		// 获取任务快照以获得正确的时间戳格式
		snap, _ := pm.Snapshot(task.ID)

		taskData := gin.H{
			"id":            task.ID,
			"name":          task.Name,
			"type":          "runtime",
			"status":        string(task.Status),
			"progress":      0.0,
			"user_id":       nil,
			"user_name":     "系统",
			"cluster_name":  "默认集群",
			"target_nodes":  0,
			"nodes_total":   0,
			"nodes_success": 0,
			"nodes_failed":  0,
			"current_step":  "",
			"steps_total":   0,
			"steps_count":   len(task.Events),
			"error_message": "",
			"tags":          nil,
			"priority":      0,
			"retry_count":   0,
			"max_retries":   0,
			"created_at":    snap.StartedAt / 1000, // 转换为秒
			"started_at":    snap.StartedAt / 1000, // 转换为秒
			"completed_at":  0,
			"duration":      time.Since(time.UnixMilli(snap.StartedAt)).Truncate(time.Second).String(),
			"success_rate":  0.0,
			"source":        "runtime",
		}

		if snap.CompletedAt > 0 {
			taskData["completed_at"] = snap.CompletedAt / 1000 // 转换为秒
			taskData["duration"] = time.UnixMilli(snap.CompletedAt).Sub(time.UnixMilli(snap.StartedAt)).Truncate(time.Second).String()
		}

		// 获取最新进度事件
		if len(task.Events) > 0 {
			latestEvent := task.Events[len(task.Events)-1]
			taskData["current_step"] = latestEvent.Step
			taskData["progress"] = latestEvent.Progress
			taskData["last_message"] = latestEvent.Message

			// 如果事件中有错误信息
			if latestEvent.Type == "error" {
				taskData["error_message"] = latestEvent.Message
			}
		}

		taskList = append(taskList, taskData)
	}

	// 返回结果 - 匹配前端期望的数据格式
	totalTasks := total + int64(len(runtimeTasks))
	response := gin.H{
		"data": gin.H{
			"tasks": taskList,
			"total": totalTasks,
			"pagination": gin.H{
				"page":        req.Page,
				"page_size":   req.PageSize,
				"total":       totalTasks,
				"total_pages": (totalTasks + int64(req.PageSize) - 1) / int64(req.PageSize),
			},
			"runtime_tasks_count": len(runtimeTasks),
			"db_tasks_count":      len(dbTasks),
		},
	}

	ctx.JSON(http.StatusOK, response)
}

// GET /api/slurm/progress/:opId/stream (SSE)
func (c *SlurmController) StreamProgress(ctx *gin.Context) {
	opID := ctx.Param("opId")
	pm := services.GetProgressManager()
	snap, ok := pm.Snapshot(opID)
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
		return
	}
	// Setup SSE headers
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	flusher, okf := ctx.Writer.(http.Flusher)
	if !okf {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "stream not supported"})
		return
	}
	// Send existing events first
	for _, ev := range snap.Events {
		b, _ := json.Marshal(ev)
		fmt.Fprintf(ctx.Writer, "data: %s\n\n", string(b))
	}
	flusher.Flush()

	// Subscribe to future events
	ch, ok := pm.Subscribe(opID)
	if !ok {
		return
	}
	defer func() {
		// channel will be closed by manager on completion; nothing else to do
	}()

	notify := ctx.Writer.CloseNotify()
	for {
		select {
		case <-notify:
			return
		case ev, more := <-ch:
			if !more {
				return
			}
			b, _ := json.Marshal(ev)
			fmt.Fprintf(ctx.Writer, "data: %s\n\n", string(b))
			flusher.Flush()
		}
	}
}

// --- 异步：仓库准备 ---

// POST /api/slurm/repo/setup/async
func (c *SlurmController) SetupRepoAsync(ctx *gin.Context) {
	var req RepoSetupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pm := services.GetProgressManager()
	op := pm.Start("slurm:setup-repo", "开始准备仓库")

	go func(opID string, r RepoSetupRequest) {
		failed := false
		defer func() {
			pm.Complete(opID, failed, "仓库准备完成")
		}()
		if r.RepoHost.Port == 0 {
			r.RepoHost.Port = 22
		}
		if r.BasePath == "" {
			r.BasePath = "/var/www/html/deb/slurm"
		}

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "connect", Message: "连接到仓库主机", Host: r.RepoHost.Host})
		// We assume connection happens within next call; no explicit check here.

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "install", Message: "安装Nginx与dpkg-dev"})
		if _, err := c.sshSvc.ExecuteCommand(r.RepoHost.Host, r.RepoHost.Port, r.RepoHost.User, r.RepoHost.Password, `/bin/sh -lc 'if command -v apt-get >/dev/null 2>&1; then export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get install -y nginx dpkg-dev && systemctl enable nginx || true && systemctl restart nginx || true; elif command -v yum >/dev/null 2>&1; then yum install -y nginx createrepo; systemctl enable nginx || true; systemctl restart nginx || true; elif command -v dnf >/dev/null 2>&1; then dnf install -y nginx createrepo; systemctl enable nginx || true; systemctl restart nginx || true; else echo "Unsupported distro"; exit 1; fi'`); err != nil {
			failed = true
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "install", Message: err.Error()})
			return
		}
		pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "install", Message: "组件安装完成"})

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "mkdir", Message: "创建仓库目录"})
		if _, err := c.sshSvc.ExecuteCommand(r.RepoHost.Host, r.RepoHost.Port, r.RepoHost.User, r.RepoHost.Password, "/bin/sh -lc 'mkdir -p "+r.BasePath+"'"); err != nil {
			failed = true
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "mkdir", Message: err.Error()})
			return
		}
		pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "mkdir", Message: "目录已创建"})

		if r.EnableIndex {
			pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "index", Message: "生成Packages索引（Deb系）"})
			if _, err := c.sshSvc.ExecuteCommand(r.RepoHost.Host, r.RepoHost.Port, r.RepoHost.User, r.RepoHost.Password, "/bin/sh -lc 'if command -v apt-ftparchive >/dev/null 2>&1; then (cd "+r.BasePath+" && apt-ftparchive packages . > Packages && gzip -f Packages); fi'"); err != nil {
				failed = true
				pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "index", Message: err.Error()})
				return
			}
			pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "index", Message: "索引生成完成"})
		}
	}(op.ID, req)

	ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// --- 异步：节点初始化 ---

// POST /api/slurm/init-nodes/async
func (c *SlurmController) InitNodesAsync(ctx *gin.Context) {
	var req InitNodesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Nodes) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "nodes is empty"})
		return
	}
	pm := services.GetProgressManager()
	op := pm.Start("slurm:init-nodes", "开始初始化节点")

	go func(opID string, r InitNodesRequest) {
		failed := false
		defer func() {
			pm.Complete(opID, failed, "节点初始化完成")
		}()
		total := float64(len(r.Nodes))
		for i, n := range r.Nodes {
			host := n.SSH.Host
			stepBase := fmt.Sprintf("node-%s", host)
			pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: stepBase + ":config-repo", Message: "配置APT仓库", Host: host, Progress: float64(i) / total})
			if err := c.sshSvc.ConfigureAptRepo(host, n.SSH.Port, n.SSH.User, n.SSH.Password, r.RepoURL); err != nil {
				failed = true
				pm.Emit(opID, services.ProgressEvent{Type: "error", Step: stepBase + ":config-repo", Message: err.Error(), Host: host, Progress: float64(i) / total})
				continue
			}
			pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: stepBase + ":config-repo", Message: "APT仓库配置完成", Host: host, Progress: float64(i) / total})

			pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: stepBase + ":install-slurm", Message: "安装Slurm组件", Host: host, Progress: float64(i) / total})
			if out, err := c.sshSvc.InstallSlurm(host, n.SSH.Port, n.SSH.User, n.SSH.Password, n.Role); err != nil {
				failed = true
				pm.Emit(opID, services.ProgressEvent{Type: "error", Step: stepBase + ":install-slurm", Message: err.Error(), Host: host, Data: out, Progress: float64(i) / total})
				continue
			} else {
				pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: stepBase + ":install-slurm", Message: "Slurm安装完成", Host: host, Progress: float64(i+1) / total})
			}
		}
	}(op.ID, req)

	ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// SetupRepo 在指定主机上安装nginx + dpkg-dev 并创建简易的 deb 仓库路径
func (c *SlurmController) SetupRepo(ctx *gin.Context) {
	var req RepoSetupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RepoHost.Port == 0 {
		req.RepoHost.Port = 22
	}
	if req.BasePath == "" {
		req.BasePath = "/var/www/html/deb/slurm"
	}

	if err := c.sshSvc.SetupSimpleDebRepo(
		req.RepoHost.Host, req.RepoHost.Port, req.RepoHost.User, req.RepoHost.Password,
		req.BasePath, req.EnableIndex,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, RepoSetupResponse{Success: false, Message: err.Error(), BaseURL: req.BaseURL})
		return
	}

	ctx.JSON(http.StatusOK, RepoSetupResponse{Success: true, Message: "Repo prepared", BaseURL: req.BaseURL})
}

// InitNodes 将所有节点配置为使用给定repo并安装slurm，依据角色安装 slurmctld/slurmd
func (c *SlurmController) InitNodes(ctx *gin.Context) {
	var req InitNodesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Nodes) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "nodes is empty"})
		return
	}

	results := make([]HostResult, 0, len(req.Nodes))
	for _, n := range req.Nodes {
		start := time.Now()
		host := n.SSH.Host
		port := n.SSH.Port
		if port == 0 {
			port = 22
		}
		// 配置APT源
		if err := c.sshSvc.ConfigureAptRepo(host, port, n.SSH.User, n.SSH.Password, req.RepoURL); err != nil {
			results = append(results, HostResult{Host: host, Success: false, Error: err.Error(), TookMS: time.Since(start).Milliseconds()})
			continue
		}
		// 安装Slurm
		out, err := c.sshSvc.InstallSlurm(host, port, n.SSH.User, n.SSH.Password, n.Role)
		if err != nil {
			results = append(results, HostResult{Host: host, Success: false, Output: out, Error: err.Error(), TookMS: time.Since(start).Milliseconds()})
			continue
		}

		// 成功安装后，将节点添加到SLURM集群数据库
		if err := c.addNodeToCluster(host, port, n.SSH.User, n.Role); err != nil {
			// 记录警告但不失败，因为SLURM已经安装成功
			out += fmt.Sprintf("\n警告: 添加节点到集群失败: %v", err)
		}

		results = append(results, HostResult{Host: host, Success: true, Output: out, TookMS: time.Since(start).Milliseconds()})
	}

	ok := true
	for _, r := range results {
		if !r.Success {
			ok = false
			break
		}
	}
	ctx.JSON(http.StatusOK, InitNodesResponse{Success: ok, Results: results})
}

// POST /api/slurm/hosts/initialize
func (c *SlurmController) InitializeHosts(ctx *gin.Context) {
	var req struct {
		Hosts []string `json:"hosts" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置超时
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
	defer cancel()

	// 检查并启动测试容器
	results, err := c.sshSvc.InitializeTestHosts(ctxWithTimeout, req.Hosts)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计成功和失败的数量
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":    successCount > 0,
		"total":      len(results),
		"successful": successCount,
		"failed":     len(results) - successCount,
		"results":    results,
	})
}

// getSaltStackMasterHost 从环境变量读取SaltStack Master主机地址
func getSaltStackMasterHost() string {
	masterHost := os.Getenv("SALTSTACK_MASTER_HOST")
	if masterHost == "" {
		masterHost = "saltstack" // 默认容器名
	}
	return masterHost
}

// getAppHubBaseURL 组合AppHub基础URL，基于EXTERNAL_HOST/APPHUB_PORT/EXTERNAL_SCHEME
func getAppHubBaseURL() string {
	scheme := os.Getenv("EXTERNAL_SCHEME")
	if scheme == "" {
		scheme = "http"
	}
	host := os.Getenv("EXTERNAL_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("APPHUB_PORT")
	if port == "" {
		// 与build.sh一致：APPHUB_PORT = EXTERNAL_PORT + 45354，缺省时退回常见端口
		port = "53434"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

// addNodeToCluster 添加节点到SLURM集群数据库
func (sc *SlurmController) addNodeToCluster(host string, port int, user, role string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	var nodeUpdated bool

	// 检查是否已存在该节点
	var existingNode models.SlurmNode
	err := db.Where("host = ?", host).First(&existingNode).Error
	if err == nil {
		// 节点已存在，更新状态
		err = db.Model(&existingNode).Updates(models.SlurmNode{
			Status:    "active",
			Port:      port,
			Username:  user,
			NodeType:  role,
			UpdatedAt: time.Now(),
		}).Error
		nodeUpdated = true
	} else {
		// 创建新节点记录 - 需要有一个默认的ClusterID
		// 这里使用ClusterID为1，在实际应用中应该根据业务逻辑确定
		node := models.SlurmNode{
			ClusterID: 1, // 默认集群ID
			NodeName:  host,
			NodeType:  role,
			Host:      host,
			Port:      port,
			Username:  user,
			Status:    "active",
			AuthType:  "password", // 默认认证方式
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = db.Create(&node).Error
		nodeUpdated = true
	}

	if err != nil {
		return fmt.Errorf("failed to save node to database: %w", err)
	}

	// 如果节点被添加或更新，重新生成SLURM配置
	if nodeUpdated {
		ctx := context.Background()
		if err := sc.slurmSvc.UpdateSlurmConfig(ctx, sc.sshSvc); err != nil {
			// 记录警告但不返回错误，因为节点已成功添加到数据库
			fmt.Printf("Warning: failed to update SLURM config after adding node %s: %v\n", host, err)
		}
	}

	return nil
}

// --- 新增：使用AppHub的自动安装API ---

// InstallPackagesRequest 安装包请求
type InstallPackagesRequest struct {
	Hosts             []HostConfig `json:"hosts" binding:"required"`
	AppHubURL         string       `json:"appHubURL" binding:"required"`
	SaltMasterHost    string       `json:"saltMasterHost"`
	SaltMasterPort    int          `json:"saltMasterPort"`
	EnableSaltMinion  bool         `json:"enableSaltMinion"`
	EnableSlurmClient bool         `json:"enableSlurmClient"`
	SlurmRole         string       `json:"slurmRole"` // controller|compute
}

// HostConfig 主机配置
type HostConfig struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	MinionID string `json:"minionId"` // 可选的Minion ID
}

// InstallPackagesResponse 安装包响应
type InstallPackagesResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Results []InstallationHostResult `json:"results"`
}

// InstallationHostResult 主机安装结果
type InstallationHostResult struct {
	Host     string                `json:"host"`
	Success  bool                  `json:"success"`
	Error    string                `json:"error"`
	Duration string                `json:"duration"`
	Steps    []services.StepResult `json:"steps"`
}

// POST /api/slurm/install-packages
func (c *SlurmController) InstallPackages(ctx *gin.Context) {
	var req InstallPackagesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	// 转换为SSH连接配置
	connections := make([]services.SSHConnection, len(req.Hosts))
	for i, host := range req.Hosts {
		port := host.Port
		if port == 0 {
			port = 22
		}
		connections[i] = services.SSHConnection{
			Host:     host.Host,
			Port:     port,
			User:     host.User,
			Password: host.Password,
		}
	}

	// 构建安装配置
	installConfig := services.PackageInstallationConfig{
		AppHubConfig: services.AppHubConfig{
			BaseURL: req.AppHubURL,
		},
		SaltMasterHost:    req.SaltMasterHost,
		SaltMasterPort:    req.SaltMasterPort,
		SlurmRole:         req.SlurmRole,
		EnableSaltMinion:  req.EnableSaltMinion,
		EnableSlurmClient: req.EnableSlurmClient,
	}

	// 设置默认值
	if installConfig.SaltMasterHost == "" {
		installConfig.SaltMasterHost = "saltstack" // 默认使用容器名
	}
	if installConfig.SaltMasterPort == 0 {
		installConfig.SaltMasterPort = 4506
	}
	if installConfig.SlurmRole == "" {
		installConfig.SlurmRole = "compute"
	}

	// 执行安装
	results, err := c.sshSvc.InstallPackagesOnHosts(context.Background(), connections, installConfig)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "安装过程中发生错误: " + err.Error(),
		})
		return
	}

	// 转换结果格式
	hostResults := make([]InstallationHostResult, len(results))
	allSuccess := true
	for i, result := range results {
		hostResults[i] = InstallationHostResult{
			Host:     result.Host,
			Success:  result.Success,
			Error:    result.Error,
			Duration: result.Duration.String(),
			Steps:    result.Steps,
		}
		if !result.Success {
			allSuccess = false
		}
	}

	// 记录安装任务到数据库
	go c.recordInstallationTask(req, results)

	message := "安装任务完成"
	if !allSuccess {
		message = "部分主机安装失败，请查看详细结果"
	}

	ctx.JSON(http.StatusOK, InstallPackagesResponse{
		Success: allSuccess,
		Message: message,
		Results: hostResults,
	})
}

// POST /api/slurm/install-test-nodes
// 专门用于测试节点的快速安装（test-ssh01, test-ssh02, test-ssh03）
func (c *SlurmController) InstallTestNodes(ctx *gin.Context) {
	type TestNodesInstallRequest struct {
		Nodes             []string `json:"nodes" binding:"required"`
		AppHubURL         string   `json:"appHubURL" binding:"required"`
		EnableSaltMinion  bool     `json:"enableSaltMinion"`
		EnableSlurmClient bool     `json:"enableSlurmClient"`
	}

	var req TestNodesInstallRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	// 首先初始化测试容器
	results, err := c.sshSvc.InitializeTestHosts(context.Background(), req.Nodes)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "初始化测试容器失败: " + err.Error(),
		})
		return
	}

	// 检查初始化结果
	var failedHosts []string
	for _, result := range results {
		if !result.Success {
			failedHosts = append(failedHosts, result.Host)
		}
	}

	if len(failedHosts) > 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("以下测试容器初始化失败: %v", failedHosts),
			"results": results,
		})
		return
	}

	// 构建SSH连接（测试容器使用固定的凭据）
	connections := make([]services.SSHConnection, len(req.Nodes))
	for i, host := range req.Nodes {
		connections[i] = services.SSHConnection{
			Host:     host,
			Port:     22,
			User:     "root",
			Password: "rootpass123", // 测试容器的默认密码
		}
	}

	// 构建安装配置
	installConfig := services.PackageInstallationConfig{
		AppHubConfig: services.AppHubConfig{
			BaseURL: req.AppHubURL,
		},
		SaltMasterHost:    "saltstack",
		SaltMasterPort:    4506,
		SlurmRole:         "compute",
		EnableSaltMinion:  req.EnableSaltMinion,
		EnableSlurmClient: req.EnableSlurmClient,
	}

	// 执行安装
	installResults, err := c.sshSvc.InstallPackagesOnHosts(context.Background(), connections, installConfig)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "安装过程中发生错误: " + err.Error(),
		})
		return
	}

	// 转换结果格式
	hostResults := make([]InstallationHostResult, len(installResults))
	allSuccess := true
	for i, result := range installResults {
		hostResults[i] = InstallationHostResult{
			Host:     result.Host,
			Success:  result.Success,
			Error:    result.Error,
			Duration: result.Duration.String(),
			Steps:    result.Steps,
		}
		if !result.Success {
			allSuccess = false
		}
	}

	message := "测试节点安装任务完成"
	if !allSuccess {
		message = "部分测试节点安装失败，请查看详细结果"
	}

	ctx.JSON(http.StatusOK, InstallPackagesResponse{
		Success: allSuccess,
		Message: message,
		Results: hostResults,
	})
}

// recordInstallationTask 记录安装任务到数据库
func (c *SlurmController) recordInstallationTask(req InstallPackagesRequest, results []services.InstallationResult) {
	db := database.DB

	// 创建安装任务记录
	task := models.InstallationTask{
		TaskName:     fmt.Sprintf("安装包到 %d 个主机", len(req.Hosts)),
		TaskType:     c.getTaskType(req.EnableSaltMinion, req.EnableSlurmClient),
		Status:       "completed",
		TotalHosts:   len(req.Hosts),
		SuccessHosts: 0,
		FailedHosts:  0,
		StartTime:    time.Now().Add(-c.calculateTotalDuration(results)),
	}

	// 设置配置信息
	if err := task.SetConfig(req); err != nil {
		fmt.Printf("设置任务配置失败: %v\n", err)
	}

	// 创建任务记录
	if err := db.Create(&task).Error; err != nil {
		fmt.Printf("创建任务记录失败: %v\n", err)
		return
	}

	// 创建主机结果记录
	for _, result := range results {
		hostResult := models.InstallationHostResult{
			TaskID:   task.ID,
			Host:     result.Host,
			Port:     22, // 默认端口
			User:     c.findUserForHost(req.Hosts, result.Host),
			Status:   c.getResultStatus(result.Success),
			Error:    result.Error,
			Duration: result.Duration.Milliseconds(),
			Output:   c.formatStepsOutput(result.Steps),
		}

		// 转换安装步骤
		installationSteps := c.convertToInstallationSteps(result.Steps)
		if err := hostResult.SetSteps(installationSteps); err != nil {
			fmt.Printf("设置主机步骤失败: %v\n", err)
		}

		// 创建主机结果记录
		if err := db.Create(&hostResult).Error; err != nil {
			fmt.Printf("创建主机结果记录失败: %v\n", err)
			continue
		}

		// 更新统计
		if result.Success {
			task.SuccessHosts++
		} else {
			task.FailedHosts++
		}
	}

	// 更新任务统计
	task.UpdateHostStats()
	if err := db.Save(&task).Error; err != nil {
		fmt.Printf("更新任务统计失败: %v\n", err)
	}

	fmt.Printf("安装任务记录完成 - 任务ID: %d, 成功: %d, 失败: %d\n",
		task.ID, task.SuccessHosts, task.FailedHosts)
}

// 辅助方法
func (c *SlurmController) getTaskType(saltEnabled, slurmEnabled bool) string {
	if saltEnabled && slurmEnabled {
		return "combined"
	} else if saltEnabled {
		return "saltstack"
	} else if slurmEnabled {
		return "slurm"
	}
	return "unknown"
}

func (c *SlurmController) calculateTotalDuration(results []services.InstallationResult) time.Duration {
	if len(results) == 0 {
		return 0
	}

	// 取最长的执行时间作为总时间（因为是并发执行）
	maxDuration := time.Duration(0)
	for _, result := range results {
		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}
	}
	return maxDuration
}

func (c *SlurmController) findUserForHost(hosts []HostConfig, targetHost string) string {
	for _, host := range hosts {
		if host.Host == targetHost {
			return host.User
		}
	}
	return "unknown"
}

func (c *SlurmController) getResultStatus(success bool) string {
	if success {
		return "success"
	}
	return "failed"
}

func (c *SlurmController) formatStepsOutput(steps []services.StepResult) string {
	var output strings.Builder
	for _, step := range steps {
		fmt.Fprintf(&output, "=== %s ===\n", step.Name)
		fmt.Fprintf(&output, "状态: %s\n", c.getResultStatus(step.Success))
		if step.Error != "" {
			fmt.Fprintf(&output, "错误: %s\n", step.Error)
		}
		fmt.Fprintf(&output, "耗时: %s\n", step.Duration.String())
		fmt.Fprintf(&output, "输出:\n%s\n\n", step.Output)
	}
	return output.String()
}

func (c *SlurmController) convertToInstallationSteps(serviceSteps []services.StepResult) []models.InstallationStep {
	steps := make([]models.InstallationStep, len(serviceSteps))
	for i, step := range serviceSteps {
		steps[i] = models.InstallationStep{
			Name:        step.Name,
			Description: step.Name, // 使用名称作为描述
			Status:      c.getResultStatus(step.Success),
			Output:      step.Output,
			Error:       step.Error,
			StartTime:   step.Timestamp,
			EndTime:     step.Timestamp.Add(step.Duration),
			Duration:    step.Duration.Milliseconds(),
		}
	}
	return steps
}

// GetTaskDetail 获取任务详情
// @Summary 获取任务详情
// @Description 获取SLURM任务的详细信息，包括执行日志和事件历史
// @Tags SLURM
// @Param task_id path string true "任务ID"
// @Success 200 {object} services.TaskDetailResponse
// @Router /api/slurm/tasks/{task_id} [get]
func (c *SlurmController) GetTaskDetail(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	// 首先尝试从数据库获取
	detail, err := c.taskSvc.GetTask(ctx.Request.Context(), taskID)
	if err != nil {
		// 如果数据库中没有，尝试从运行时获取
		pm := services.GetProgressManager()
		snap, ok := pm.Snapshot(taskID)
		if !ok {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
			return
		}

		// 转换运行时任务为响应格式
		runtimeDetail := gin.H{
			"task": gin.H{
				"task_id":    snap.ID,
				"name":       snap.Name,
				"type":       "runtime",
				"status":     string(snap.Status),
				"started_at": snap.StartedAt,
				"events":     snap.Events,
				"source":     "runtime",
			},
			"events": snap.Events,
			"statistics": gin.H{
				"total_events":       len(snap.Events),
				"formatted_duration": time.Since(time.UnixMilli(snap.StartedAt)).Truncate(time.Second).String(),
			},
			"can_retry":  false,
			"can_cancel": snap.Status == "running",
		}

		if snap.CompletedAt > 0 {
			runtimeDetail["task"].(gin.H)["completed_at"] = snap.CompletedAt
			runtimeDetail["statistics"].(gin.H)["formatted_duration"] = time.UnixMilli(snap.CompletedAt).Sub(time.UnixMilli(snap.StartedAt)).Truncate(time.Second).String()
		}

		ctx.JSON(http.StatusOK, gin.H{"data": runtimeDetail})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": detail})
}

// GetTaskStatistics 获取任务统计信息
// @Summary 获取任务统计信息
// @Description 获取指定时间范围内的任务统计信息
// @Tags SLURM
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/tasks/statistics [get]
func (c *SlurmController) GetTaskStatistics(ctx *gin.Context) {
	// 解析时间参数
	startDateStr := ctx.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDateStr := ctx.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式错误，请使用 YYYY-MM-DD 格式"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式错误，请使用 YYYY-MM-DD 格式"})
		return
	}

	// 设置时间范围为当天的结束时间
	endDate = endDate.Add(24*time.Hour - time.Second)

	statistics, err := c.taskSvc.GetTaskStatistics(ctx.Request.Context(), startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计信息失败: " + err.Error()})
		return
	}

	// 添加运行时任务统计
	pm := services.GetProgressManager()
	runtimeTasks := pm.ListOperations()
	runtimeStats := gin.H{
		"total_runtime_tasks": len(runtimeTasks),
		"runtime_by_status":   make(map[string]int),
	}

	for _, task := range runtimeTasks {
		status := string(task.Status)
		if count, exists := runtimeStats["runtime_by_status"].(map[string]int)[status]; exists {
			runtimeStats["runtime_by_status"].(map[string]int)[status] = count + 1
		} else {
			runtimeStats["runtime_by_status"].(map[string]int)[status] = 1
		}
	}

	statistics["runtime_stats"] = runtimeStats
	statistics["date_range"] = gin.H{
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}

	ctx.JSON(http.StatusOK, gin.H{"data": statistics})
}

// CancelTask 取消任务
// @Summary 取消任务
// @Description 取消正在执行或待执行的任务
// @Tags SLURM
// @Param task_id path string true "任务ID"
// @Param reason body object false "取消原因"
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/tasks/{task_id}/cancel [post]
func (c *SlurmController) CancelTask(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		req.Reason = "用户取消"
	}

	if req.Reason == "" {
		req.Reason = "用户取消"
	}

	// 尝试取消数据库任务
	err := c.taskSvc.CancelTask(ctx.Request.Context(), taskID, req.Reason)
	if err != nil {
		// 尝试取消运行时任务
		pm := services.GetProgressManager()
		if _, exists := pm.Get(taskID); exists {
			pm.Complete(taskID, true, "任务被用户取消: "+req.Reason)
			ctx.JSON(http.StatusOK, gin.H{
				"message": "运行时任务已取消",
				"task_id": taskID,
			})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": "取消任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "任务已取消",
		"task_id": taskID,
	})
}

// RetryTask 重试任务
// @Summary 重试任务
// @Description 重试失败的任务
// @Tags SLURM
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/tasks/{task_id}/retry [post]
func (c *SlurmController) RetryTask(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	newTask, err := c.taskSvc.RetryTask(ctx.Request.Context(), taskID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "重试任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "重试任务已创建",
		"original_id": taskID,
		"new_task_id": newTask.TaskID,
		"new_task":    newTask,
	})
}

// DeleteTask 删除任务
// @Summary 删除任务
// @Description 删除已完成或失败的任务记录
// @Tags SLURM
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/tasks/{task_id} [delete]
func (c *SlurmController) DeleteTask(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	err := c.taskSvc.DeleteTask(ctx.Request.Context(), taskID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "删除任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "任务已删除",
		"task_id": taskID,
	})
}

// GetInstallationTasks 获取安装任务列表
func (c *SlurmController) GetInstallationTasks(ctx *gin.Context) {
	db := database.DB

	var tasks []models.InstallationTask

	// 获取查询参数
	limit := 50 // 默认限制
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	taskType := ctx.Query("type")
	status := ctx.Query("status")

	// 构建查询
	query := db.Model(&models.InstallationTask{})

	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 预加载关联的主机结果
	if err := query.Preload("HostResults").Order("created_at DESC").Limit(limit).Find(&tasks).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取安装任务列表失败",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// GetInstallationTask 获取单个安装任务详情
func (c *SlurmController) GetInstallationTask(ctx *gin.Context) {
	db := database.DB

	taskIDStr := ctx.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的任务ID",
		})
		return
	}

	var task models.InstallationTask

	// 查找任务并预加载关联数据
	if err := db.Preload("HostResults").First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "任务不存在",
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取任务详情失败",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, task)
}

// getSaltAPIURL 获取 Salt API URL（从环境变量读取）
func (c *SlurmController) getSaltAPIURL() string {
	// 优先使用 SALTSTACK_MASTER_URL
	if url := os.Getenv("SALTSTACK_MASTER_URL"); url != "" {
		return url
	}

	// 否则组合 scheme + host + port
	scheme := c.getEnv("SALT_API_SCHEME", "http")
	host := c.getEnv("SALT_MASTER_HOST", "salt-master")
	port := c.getEnv("SALT_API_PORT", "8000")

	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func (c *SlurmController) getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration 获取环境变量并转换为 Duration
func (c *SlurmController) getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ExecuteSlurmCommand 执行SLURM运维命令
// POST /api/slurm/exec
func (c *SlurmController) ExecuteSlurmCommand(ctx *gin.Context) {
	var req struct {
		Command string `json:"command" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 验证命令是否为允许的SLURM命令
	allowedCommands := []string{"sinfo", "squeue", "scontrol", "sacct", "sstat", "srun", "sbatch", "scancel"}
	commandParts := strings.Fields(req.Command)
	if len(commandParts) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "命令不能为空",
		})
		return
	}

	baseCommand := commandParts[0]
	isAllowed := false
	for _, allowed := range allowedCommands {
		if baseCommand == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error":            fmt.Sprintf("不允许执行命令: %s", baseCommand),
			"allowed_commands": allowedCommands,
		})
		return
	}

	// 通过SSH执行命令
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, req.Command)
	if err != nil {
		logrus.Errorf("执行SLURM命令失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "命令执行失败",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"command": req.Command,
		"output":  output,
		"stdout":  output, // 兼容性
	})
}

// GetSlurmDiagnostics 获取SLURM诊断信息
// GET /api/slurm/diagnostics
func (c *SlurmController) GetSlurmDiagnostics(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	// 执行多个诊断命令
	diagnostics := make(map[string]interface{})

	// 1. sinfo - 节点信息
	if output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, "sinfo"); err == nil {
		diagnostics["sinfo"] = output
	} else {
		diagnostics["sinfo_error"] = err.Error()
	}

	// 2. sinfo -Nel - 详细节点列表
	if output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, "sinfo -Nel"); err == nil {
		diagnostics["sinfo_detail"] = output
	} else {
		diagnostics["sinfo_detail_error"] = err.Error()
	}

	// 3. squeue - 作业队列
	if output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, "squeue"); err == nil {
		diagnostics["squeue"] = output
	} else {
		diagnostics["squeue_error"] = err.Error()
	}

	// 4. scontrol show config - 配置信息
	if output, err := c.slurmSvc.ExecuteSlurmCommand(ctxWithTimeout, "scontrol show config"); err == nil {
		diagnostics["config"] = output
	} else {
		diagnostics["config_error"] = err.Error()
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":     true,
		"diagnostics": diagnostics,
		"timestamp":   time.Now().Format(time.RFC3339),
	})
}

// POST /api/slurm/scaling/scale-up-api - 基于REST API的扩容
func (c *SlurmController) ScaleUpViaAPI(ctx *gin.Context) {
	var req ScalingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 300*time.Second)
	defer cancel()

	// 执行基于REST API的扩容操作
	results, err := c.slurmSvc.ScaleUpViaAPI(ctxWithTimeout, req.Nodes)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": results})
}

// POST /api/slurm/scaling/scale-down-api - 基于REST API的缩容
func (c *SlurmController) ScaleDownViaAPI(ctx *gin.Context) {
	var req ScaleDownRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
	defer cancel()

	results, err := c.slurmSvc.ScaleDownViaAPI(ctxWithTimeout, req.NodeIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": results})
}

// POST /api/slurm/reload-config - 重新加载SLURM配置
func (c *SlurmController) ReloadSlurmConfig(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	err := c.slurmSvc.ReloadSlurmConfig(ctxWithTimeout)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "SLURM配置已重新加载",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
