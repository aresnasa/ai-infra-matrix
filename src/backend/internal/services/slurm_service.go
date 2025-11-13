package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Slurm service provides lightweight access to Slurm cluster metrics by shelling out
// to common CLI tools (sinfo, squeue). This implementation is strict: it requires
// real tools/data and no longer falls back to demo data.

type SlurmService struct {
	db            *gorm.DB
	restAPIURL    string
	restAPIToken  string
	httpClient    *http.Client
	useSlurmrestd bool // 是否使用 slurmrestd API (true) 还是 SSH (false)
}

func NewSlurmService() *SlurmService {
	restAPIURL := os.Getenv("SLURM_REST_API_URL")
	if restAPIURL == "" {
		restAPIURL = "http://slurm-master:6820" // 默认URL
	}

	// 从环境变量读取是否使用 slurmrestd
	useSlurmrestd := os.Getenv("USE_SLURMRESTD") == "true"

	return &SlurmService{
		restAPIURL:    restAPIURL,
		useSlurmrestd: useSlurmrestd,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewSlurmServiceWithDB(db *gorm.DB) *SlurmService {
	restAPIURL := os.Getenv("SLURM_REST_API_URL")
	if restAPIURL == "" {
		restAPIURL = "http://slurm-master:6820" // 默认URL
	}

	// 从环境变量读取是否使用 slurmrestd
	useSlurmrestd := os.Getenv("USE_SLURMRESTD") == "true"

	return &SlurmService{
		db:            db,
		restAPIURL:    restAPIURL,
		useSlurmrestd: useSlurmrestd,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetUseSlurmRestd 返回是否启用 slurmrestd REST API
func (s *SlurmService) GetUseSlurmRestd() bool {
	return s.useSlurmrestd
}

func (s *SlurmService) GetSummary(ctx context.Context) (*SlurmSummary, error) {
	// Gather from sinfo/squeue; if tools are unavailable, return empty summary
	nodesTotal, nodesIdle, nodesAlloc, partitions, demo1 := s.getNodeStats(ctx)
	jobsRun, jobsPend, jobsOther, demo2 := s.getJobStats(ctx)

	// Return empty summary instead of error when tools are unavailable
	return &SlurmSummary{
		NodesTotal:  nodesTotal,
		NodesIdle:   nodesIdle,
		NodesAlloc:  nodesAlloc,
		Partitions:  partitions,
		JobsRunning: jobsRun,
		JobsPending: jobsPend,
		JobsOther:   jobsOther,
		Demo:        demo1 || demo2,
		GeneratedAt: time.Now(),
	}, nil
}

func (s *SlurmService) GetNodes(ctx context.Context) ([]SlurmNode, bool, error) {
	// 通过SSH执行sinfo命令获取节点信息
	// sinfo format: NodeName|State|CPUS(A/I/O/T)|Memory|Partition
	output, err := s.executeSlurmCommand(ctx, "sinfo -N -o '%N|%T|%C|%m|%P'")
	if err != nil {
		// Try to get data from database first
		if s.db != nil {
			var dbNodes []models.SlurmNode
			if err := s.db.Find(&dbNodes).Error; err == nil && len(dbNodes) > 0 {
				// Convert database nodes to service nodes
				var nodes []SlurmNode
				for _, dbNode := range dbNodes {
					nodes = append(nodes, SlurmNode{
						Name:      dbNode.NodeName,
						State:     dbNode.Status,
						CPUs:      fmt.Sprintf("%d", dbNode.CPUs),
						MemoryMB:  fmt.Sprintf("%d", dbNode.Memory),
						Partition: "compute", // 默认分区
					})
				}
				return nodes, false, nil
			}
		}
		// No SSH connection and no DB fallback: return empty list with demo flag
		return []SlurmNode{}, true, nil
	}

	var nodes []SlurmNode
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// skip header if present
		if first && strings.Contains(line, "NODELIST") {
			first = false
			continue
		}
		first = false
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			// try to be lenient
			for len(parts) < 5 {
				parts = append(parts, "")
			}
		}
		nodes = append(nodes, SlurmNode{
			Name:      parts[0],
			State:     parts[1],
			CPUs:      parseCPUs(parts[2]),
			MemoryMB:  parts[3],
			Partition: parts[4],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}
	return nodes, false, nil
}

func (s *SlurmService) GetJobs(ctx context.Context) ([]SlurmJob, bool, error) {
	// squeue format: JobID|User|State|Elapsed|Nodes|Reason|Name|Partition
	// 通过SSH执行squeue命令
	output, err := s.executeSlurmCommand(ctx, "squeue -o '%i|%u|%T|%M|%D|%R|%j|%P'")
	if err != nil {
		// SSH命令失败: 返回空列表和demo标记
		return []SlurmJob{}, true, nil
	}

	var jobs []SlurmJob
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if first && (strings.Contains(line, "JOBID") || strings.Contains(line, "JOBID PARTITION")) {
			first = false
			continue
		}
		first = false
		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			for len(parts) < 8 {
				parts = append(parts, "")
			}
		}
		jobs = append(jobs, SlurmJob{
			ID:        parts[0],
			User:      parts[1],
			State:     parts[2],
			Elapsed:   parts[3],
			Nodes:     parts[4],
			Reason:    parts[5],
			Name:      parts[6],
			Partition: parts[7],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}
	return jobs, false, nil
}

// Helpers
func (s *SlurmService) getNodeStats(ctx context.Context) (total, idle, alloc, partitions int, demo bool) {
	// 通过SSH执行sinfo命令获取节点统计信息
	// sinfo summarized counts
	// partitions count via: sinfo -h -o %P | sort -u | wc -l (approx) – we'll parse simply
	output, err := s.executeSlurmCommand(ctx, "sinfo -h -o '%T|%P'")
	if err != nil {
		// Return zeros instead of mock data when SLURM is unavailable
		return 0, 0, 0, 0, true
	}
	seenPartitions := map[string]struct{}{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		state := parts[0]
		_ = state
		part := ""
		if len(parts) > 1 {
			part = parts[1]
		}
		if part != "" {
			seenPartitions[part] = struct{}{}
		}
		total++
		// Rough classification: lines correspond to partition-view; we'll refine with -N below
	}
	// Try better node-level stats
	out2, err2 := s.executeSlurmCommand(ctx, "sinfo -h -N -o '%T'")
	if err2 == nil {
		total = 0
		idle = 0
		alloc = 0
		scanner2 := bufio.NewScanner(strings.NewReader(out2))
		for scanner2.Scan() {
			total++
			st := strings.TrimSpace(scanner2.Text())
			up := strings.ToUpper(st)
			if strings.Contains(up, "IDLE") {
				idle++
			}
			if strings.Contains(up, "ALLOC") || up == "MIXED" {
				alloc++
			}
		}
	}
	partitions = len(seenPartitions)
	if partitions == 0 {
		// Try to derive partitions another way
		out3, err3 := s.executeSlurmCommand(ctx, "sinfo -h -o '%P'")
		if err3 == nil {
			set := map[string]struct{}{}
			scanner3 := bufio.NewScanner(strings.NewReader(out3))
			for scanner3.Scan() {
				set[strings.TrimSpace(scanner3.Text())] = struct{}{}
			}
			partitions = len(set)
		}
	}
	return total, idle, alloc, partitions, false
}

func (s *SlurmService) getJobStats(ctx context.Context) (running, pending, other int, demo bool) {
	// 通过SSH执行squeue命令获取作业统计信息
	output, err := s.executeSlurmCommand(ctx, "squeue -h -o '%T'")
	if err != nil {
		// Return zeros instead of mock data when SLURM is unavailable
		return 0, 0, 0, true
	}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		st := strings.ToUpper(strings.TrimSpace(scanner.Text()))
		switch st {
		case "RUNNING":
			running++
		case "PENDING":
			pending++
		default:
			other++
		}
	}
	return running, pending, other, false
}

func parseCPUs(c string) string {
	// CPUS format like A/I/O/T; return T if parsable
	if strings.Contains(c, "/") {
		parts := strings.Split(c, "/")
		return parts[len(parts)-1]
	}
	// Or a number
	if _, err := strconv.Atoi(c); err == nil {
		return c
	}
	return ""
}

// utility to detect missing binaries
func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

var ErrNotAvailable = errors.New("slurm tools not available")

// SlurmSummary SLURM集群摘要信息
type SlurmSummary struct {
	NodesTotal  int       `json:"nodes_total"`
	NodesIdle   int       `json:"nodes_idle"`
	NodesAlloc  int       `json:"nodes_alloc"`
	Partitions  int       `json:"partitions"`
	JobsRunning int       `json:"jobs_running"`
	JobsPending int       `json:"jobs_pending"`
	JobsOther   int       `json:"jobs_other"`
	Demo        bool      `json:"demo"`
	GeneratedAt time.Time `json:"generated_at"`
}

// SlurmNode SLURM节点信息
type SlurmNode struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	CPUs      string `json:"cpus"`
	MemoryMB  string `json:"memory_mb"`
	Partition string `json:"partition"`
}

// SlurmJob SLURM作业信息
type SlurmJob struct {
	ID        string `json:"id"`
	User      string `json:"user"`
	State     string `json:"state"`
	Elapsed   string `json:"elapsed"`
	Nodes     string `json:"nodes"`
	Reason    string `json:"reason"`
	Name      string `json:"name"`
	Partition string `json:"partition"`
}

// ScalingStatus 扩缩容状态
type ScalingStatus struct {
	ActiveOperations []ScalingOperation `json:"active_operations"`
	RecentOperations []ScalingOperation `json:"recent_operations"`
	NodeTemplates    []NodeTemplate     `json:"node_templates"`
	// 前端期望的字段
	Active       bool `json:"active"`        // 是否有活跃任务
	ActiveTasks  int  `json:"active_tasks"`  // 活跃任务数
	SuccessNodes int  `json:"success_nodes"` // 成功节点数
	FailedNodes  int  `json:"failed_nodes"`  // 失败节点数
	Progress     int  `json:"progress"`      // 进度百分比
}

// ScalingOperation 扩缩容操作
type ScalingOperation struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"` // "scale-up" or "scale-down"
	Status      string     `json:"status"`
	Nodes       []string   `json:"nodes"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// NodeConfig 节点配置
// NodeConfig 节点配置
type NodeConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	KeyPath    string `json:"key_path"`
	PrivateKey string `json:"private_key"` // 新增：内联私钥内容
	Password   string `json:"password"`
	MinionID   string `json:"minion_id"`

	// 硬件配置
	CPUs           int    `json:"cpus"`             // CPU 核心数
	Memory         int    `json:"memory"`           // 内存大小 (MB)
	Storage        int    `json:"storage"`          // 存储大小 (GB)
	GPUs           int    `json:"gpus"`             // GPU 数量
	XPUs           int    `json:"xpus"`             // XPU (昆仑芯) 数量
	Sockets        int    `json:"sockets"`          // CPU 插槽数
	CoresPerSocket int    `json:"cores_per_socket"` // 每个插槽的核心数
	ThreadsPerCore int    `json:"threads_per_core"` // 每个核心的线程数
	Features       string `json:"features"`         // 节点特性标签
}

// NodeTemplate 节点模板
type NodeTemplate struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Config      NodeConfig `json:"config"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ScalingResult 扩缩容结果
type ScalingResult struct {
	OperationID string              `json:"operation_id"`
	Success     bool                `json:"success"`
	Results     []NodeScalingResult `json:"results"`
}

// NodeScalingResult 节点扩缩容结果
type NodeScalingResult struct {
	NodeID  string `json:"node_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetScalingStatus 获取扩缩容状态
func (s *SlurmService) GetScalingStatus(ctx context.Context) (*ScalingStatus, error) {
	// 从数据库获取实际的任务状态
	var runningTasks int64
	var completedTasks int64
	var failedTasks int64

	// 查询 NodeInstallTask 表中的扩缩容相关任务数量
	// 查询最近24小时内的任务
	recentTime := time.Now().Add(-24 * time.Hour)

	s.db.Model(&models.NodeInstallTask{}).
		Where("created_at > ?", recentTime).
		Where("status IN ?", []string{"running", "in_progress", "processing", "pending"}).
		Count(&runningTasks)

	s.db.Model(&models.NodeInstallTask{}).
		Where("created_at > ?", recentTime).
		Where("status IN ?", []string{"completed", "success"}).
		Count(&completedTasks)

	s.db.Model(&models.NodeInstallTask{}).
		Where("created_at > ?", recentTime).
		Where("status IN ?", []string{"failed", "error"}).
		Count(&failedTasks)

	// 计算总体进度
	totalTasks := runningTasks + completedTasks + failedTasks
	progress := 0
	if totalTasks > 0 {
		progress = int((completedTasks * 100) / totalTasks)
	}

	// 获取最近的安装任务及其详细进度
	var recentTasks []models.NodeInstallTask
	s.db.Where("created_at > ?", recentTime).
		Order("created_at DESC").
		Limit(10).
		Find(&recentTasks)

	// 如果有正在运行的任务，计算加权平均进度
	if runningTasks > 0 {
		var totalProgress int64
		var taskCount int64
		for _, task := range recentTasks {
			if task.Status == "running" || task.Status == "in_progress" || task.Status == "processing" {
				totalProgress += int64(task.Progress)
				taskCount++
			}
		}
		// 使用任务的实际进度来计算更准确的总体进度
		if taskCount > 0 {
			runningAvgProgress := totalProgress / taskCount
			// 综合考虑完成任务和正在运行任务的进度
			if totalTasks > 0 {
				progress = int((completedTasks*100 + runningTasks*runningAvgProgress) / totalTasks)
			}
		}
	}

	// 构建最近操作列表
	recentOperations := []ScalingOperation{}
	for _, task := range recentTasks {
		if task.Status == "completed" || task.Status == "failed" {
			op := ScalingOperation{
				ID:        task.TaskID,
				Type:      task.TaskType,
				Status:    task.Status,
				Nodes:     []string{fmt.Sprintf("node-%d", task.NodeID)},
				StartedAt: task.CreatedAt,
			}
			if task.CompletedAt != nil {
				op.CompletedAt = task.CompletedAt
			}
			if task.ErrorMessage != "" {
				op.Error = task.ErrorMessage
			}
			recentOperations = append(recentOperations, op)
			if len(recentOperations) >= 5 {
				break
			}
		}
	}

	return &ScalingStatus{
		ActiveOperations: []ScalingOperation{},
		RecentOperations: recentOperations,
		NodeTemplates:    []NodeTemplate{},
		Active:           runningTasks > 0,
		ActiveTasks:      int(runningTasks),
		SuccessNodes:     int(completedTasks),
		FailedNodes:      int(failedTasks),
		Progress:         progress,
	}, nil
}

// ScaleUp 执行扩容操作
func (s *SlurmService) ScaleUp(ctx context.Context, nodes []NodeConfig) (*ScalingResult, error) {
	log.Printf("[DEBUG] ScaleUp: 开始扩容操作，节点数量: %d", len(nodes))

	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	// 1. 更新数据库中的节点状态为active
	for _, node := range nodes {
		// 查找或创建节点记录
		var dbNode models.SlurmNode
		err := s.db.Where("host = ? OR node_name = ?", node.Host, node.MinionID).First(&dbNode).Error

		if err == gorm.ErrRecordNotFound {
			// 节点不存在，跳过（应该在之前的步骤中创建）
			log.Printf("[WARN] 节点 %s 在数据库中不存在，跳过", node.Host)
			result.Results = append(result.Results, NodeScalingResult{
				NodeID:  node.Host,
				Success: false,
				Message: "节点在数据库中不存在",
			})
			continue
		} else if err != nil {
			log.Printf("[ERROR] 查询节点 %s 失败: %v", node.Host, err)
			result.Results = append(result.Results, NodeScalingResult{
				NodeID:  node.Host,
				Success: false,
				Message: fmt.Sprintf("数据库错误: %v", err),
			})
			continue
		}

		// 更新节点状态和硬件配置为active
		updateData := map[string]interface{}{
			"status":         "active",
			"salt_minion_id": node.MinionID,
		}

		// 如果提供了硬件配置，更新到数据库
		if node.CPUs > 0 {
			updateData["cpus"] = node.CPUs
		}
		if node.Memory > 0 {
			updateData["memory"] = node.Memory
		}
		if node.Storage > 0 {
			updateData["storage"] = node.Storage
		}
		if node.GPUs > 0 {
			updateData["gpus"] = node.GPUs
		}
		if node.XPUs > 0 {
			updateData["xpus"] = node.XPUs
		}
		if node.Sockets > 0 {
			updateData["sockets"] = node.Sockets
		}
		if node.CoresPerSocket > 0 {
			updateData["cores_per_socket"] = node.CoresPerSocket
		}
		if node.ThreadsPerCore > 0 {
			updateData["threads_per_core"] = node.ThreadsPerCore
		}

		if err := s.db.Model(&dbNode).Updates(updateData).Error; err != nil {
			log.Printf("[ERROR] 更新节点 %s 状态失败: %v", node.Host, err)
			result.Results = append(result.Results, NodeScalingResult{
				NodeID:  node.Host,
				Success: false,
				Message: fmt.Sprintf("更新状态失败: %v", err),
			})
			continue
		}

		log.Printf("[DEBUG] 节点 %s 状态已更新为 active", node.Host)
		result.Results = append(result.Results, NodeScalingResult{
			NodeID:  node.Host,
			Success: true,
			Message: "节点已成功添加到SLURM集群",
		})
	}

	// 2. 在节点上并发安装slurmd服务
	log.Printf("[DEBUG] ScaleUp: 开始在 %d 个节点上并发安装slurmd服务", len(nodes))

	var installWg sync.WaitGroup
	var installMu sync.Mutex
	installResults := make(map[string]*InstallSlurmNodeResponse)

	// 限制并发数为 5，避免过多并发
	maxConcurrency := 5
	semaphore := make(chan struct{}, maxConcurrency)

	for _, node := range nodes {
		installWg.Add(1)
		go func(n NodeConfig) {
			defer installWg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检测操作系统类型
			osType := s.detectNodeOSType(ctx, n.Host)
			if osType == "" {
				log.Printf("[WARN] 无法检测节点 %s 的操作系统类型，尝试rocky", n.Host)
				osType = "rocky" // 默认使用rocky
			}

			log.Printf("[INFO] 在节点 %s 上安装slurmd (OS: %s)", n.Host, osType)

			// 调用安装方法
			installReq := InstallSlurmNodeRequest{
				NodeName: n.Host,
				OSType:   osType,
			}

			installResp, err := s.InstallSlurmNode(ctx, installReq)
			if err != nil {
				installResp = &InstallSlurmNodeResponse{
					Success: false,
					Message: fmt.Sprintf("安装失败: %v", err),
					Logs:    "",
				}
			}

			// 保存结果
			installMu.Lock()
			installResults[n.Host] = installResp
			installMu.Unlock()

			if installResp.Success {
				log.Printf("[SUCCESS] 节点 %s slurmd安装成功", n.Host)
			} else {
				log.Printf("[ERROR] 节点 %s slurmd安装失败: %s", n.Host, installResp.Message)
			}
		}(node)
	}

	// 等待所有安装任务完成
	installWg.Wait()
	log.Printf("[DEBUG] ScaleUp: 所有节点安装任务已完成")

	// 更新结果消息
	for i := range result.Results {
		nodeID := result.Results[i].NodeID
		if installResp, ok := installResults[nodeID]; ok {
			if installResp.Success {
				result.Results[i].Message = "节点已成功添加到SLURM集群并安装slurmd服务"
			} else {
				result.Results[i].Success = false
				result.Results[i].Message = fmt.Sprintf("节点添加成功但slurmd安装失败: %s", installResp.Message)
			}
		}
	}

	// 3. 生成并更新SLURM配置
	log.Printf("[DEBUG] ScaleUp: 开始更新SLURM配置")
	if err := s.updateSlurmConfigAndReload(ctx); err != nil {
		log.Printf("[ERROR] 更新SLURM配置失败: %v", err)
		// 不返回错误，因为节点状态已更新，可以手动重新加载配置
		result.Success = false
		for i := range result.Results {
			if result.Results[i].Success {
				result.Results[i].Message += " (SLURM配置更新失败，需要手动reload)"
			}
		}
	} else {
		log.Printf("[DEBUG] ScaleUp: SLURM配置更新成功")
	}

	return result, nil
}

// updateSlurmConfigAndReload 更新SLURM配置并重新加载
func (s *SlurmService) updateSlurmConfigAndReload(ctx context.Context) error {
	// 获取所有active状态的节点
	var nodes []models.SlurmNode
	if err := s.db.Where("status = ?", "active").Find(&nodes).Error; err != nil {
		return fmt.Errorf("获取active节点失败: %w", err)
	}

	log.Printf("[DEBUG] 找到 %d 个active节点", len(nodes))

	if len(nodes) == 0 {
		return fmt.Errorf("没有active状态的节点")
	}

	// 生成SLURM配置
	config := s.generateSlurmConfig(nodes)
	log.Printf("[DEBUG] 生成的SLURM配置:\n%s", config)

	// 通过Salt API更新SLURM master的配置文件
	if err := s.updateSlurmMasterConfig(ctx, config); err != nil {
		return fmt.Errorf("更新SLURM master配置失败: %w", err)
	}

	// 重新加载SLURM配置
	if err := s.reloadSlurmConfig(ctx); err != nil {
		return fmt.Errorf("重新加载SLURM配置失败: %w", err)
	}

	// 等待一秒让配置生效
	time.Sleep(1 * time.Second)

	// 将所有新节点设置为 DOWN 状态，并检测节点健康状态
	log.Printf("[DEBUG] 开始设置新添加节点的初始状态并进行健康检测...")
	for _, node := range nodes {
		if node.NodeType == "compute" || node.NodeType == "node" {
			// 将节点设置为 DOWN 状态
			downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason=\"新添加节点，正在检测状态\"", node.NodeName)
			output, err := s.ExecuteSlurmCommand(ctx, downCmd)
			if err != nil {
				log.Printf("[WARNING] 设置节点 %s 为 DOWN 状态失败: %v, output: %s", node.NodeName, err, output)
				continue
			}
			log.Printf("[INFO] 节点 %s 已设置为 DOWN 状态，开始健康检测...", node.NodeName)

			// 尝试检测和修复节点状态（最多3次）
			if err := s.DetectAndFixNodeState(ctx, node.NodeName, 3); err != nil {
				log.Printf("[ERROR] 节点 %s 健康检测失败: %v", node.NodeName, err)
			}
		}
	}

	log.Printf("[INFO] 所有新节点已完成初始化和健康检测")

	return nil
}

// DetectAndFixNodeState 检测节点状态并尝试修复异常（导出方法）
// maxRetries: 最大重试次数
// 返回 nil 表示节点正常或已成功修复，返回 error 表示修复失败
func (s *SlurmService) DetectAndFixNodeState(ctx context.Context, nodeName string, maxRetries int) error {
	log.Printf("[DEBUG] 开始检测节点 %s 的健康状态（最多重试 %d 次）", nodeName, maxRetries)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 等待节点状态稳定
		if attempt > 1 {
			waitTime := time.Duration(attempt) * 2 * time.Second
			log.Printf("[DEBUG] 第 %d 次检测前等待 %v...", attempt, waitTime)
			time.Sleep(waitTime)
		} else {
			time.Sleep(2 * time.Second)
		}

		// 检查节点状态
		checkCmd := fmt.Sprintf("scontrol show node %s", nodeName)
		output, err := s.ExecuteSlurmCommand(ctx, checkCmd)
		if err != nil {
			log.Printf("[WARNING] 第 %d/%d 次：无法检查节点 %s 状态: %v", attempt, maxRetries, nodeName, err)
			continue
		}

		// 分析节点状态
		if strings.Contains(output, "State=IDLE") || strings.Contains(output, "State=ALLOCATED") || strings.Contains(output, "State=MIXED") {
			// 节点状态正常
			log.Printf("[SUCCESS] 节点 %s 状态正常: IDLE/ALLOCATED/MIXED", nodeName)
			return nil
		}

		if strings.Contains(output, "State=DOWN") && !strings.Contains(output, "NOT_RESPONDING") {
			// 节点处于 DOWN 状态但可以响应，尝试激活
			log.Printf("[INFO] 第 %d/%d 次：节点 %s 处于 DOWN 状态，尝试激活...", attempt, maxRetries, nodeName)
			resumeCmd := fmt.Sprintf("scontrol update NodeName=%s State=RESUME", nodeName)
			if _, err := s.ExecuteSlurmCommand(ctx, resumeCmd); err != nil {
				log.Printf("[WARNING] 第 %d/%d 次：激活节点 %s 失败: %v", attempt, maxRetries, nodeName, err)
				continue
			}
			log.Printf("[INFO] 第 %d/%d 次：节点 %s 激活命令已执行", attempt, maxRetries, nodeName)
			continue // 下一次循环检查激活是否成功
		}

		if strings.Contains(output, "NOT_RESPONDING") || strings.Contains(output, "State=UNKNOWN") {
			// 节点无响应或状态未知
			log.Printf("[WARNING] 第 %d/%d 次：节点 %s 未响应或状态未知", attempt, maxRetries, nodeName)

			// 尝试将节点设置为 DOWN 状态（避免 UNKNOWN）
			downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason=\"节点未响应，第%d次检测失败\"", nodeName, attempt)
			if _, err := s.ExecuteSlurmCommand(ctx, downCmd); err != nil {
				log.Printf("[WARNING] 第 %d/%d 次：设置节点 %s 为 DOWN 失败: %v", attempt, maxRetries, nodeName, err)
			}

			// 如果不是最后一次尝试，继续重试
			if attempt < maxRetries {
				continue
			}

			// 最后一次尝试失败，返回错误
			return fmt.Errorf("节点未响应或状态未知，可能原因：slurmd未运行、网络不可达或配置错误。请执行：1) 检查节点连接性 2) 确认slurmd服务状态 3) 手动执行: scontrol update NodeName=%s State=RESUME", nodeName)
		}

		// 其他未预期的状态
		log.Printf("[INFO] 第 %d/%d 次：节点 %s 状态: %s", attempt, maxRetries, nodeName, output)
	}

	// 所有重试都失败
	return fmt.Errorf("节点状态检测失败，已重试 %d 次。请手动检查节点状态并执行: scontrol update NodeName=%s State=RESUME", maxRetries, nodeName)
}

// updateSlurmMasterConfig 通过SSH动态更新SLURM master的配置文件
func (s *SlurmService) updateSlurmMasterConfig(ctx context.Context, config string) error {
	log.Printf("[DEBUG] 开始通过SSH更新SLURM配置")

	// 获取SLURM master的SSH连接信息
	slurmMasterHost := os.Getenv("SLURM_MASTER_HOST")
	if slurmMasterHost == "" {
		slurmMasterHost = "ai-infra-slurm-master" // 容器名，Docker网络可以解析
	}

	slurmMasterUser := os.Getenv("SLURM_MASTER_USER")
	if slurmMasterUser == "" {
		slurmMasterUser = "root"
	}

	slurmMasterPassword := os.Getenv("SLURM_MASTER_PASSWORD")
	if slurmMasterPassword == "" {
		slurmMasterPassword = "root" // 默认密码
	}

	// 1. 首先读取当前的基础配置（保留非节点相关的配置）
	log.Printf("[DEBUG] 读取SLURM master现有配置...")
	readCmd := "cat /etc/slurm/slurm.conf"
	currentConfig, err := s.executeSSHCommand(slurmMasterHost, 22, slurmMasterUser, slurmMasterPassword, readCmd)
	if err != nil {
		log.Printf("[WARNING] 无法读取现有配置，将使用新配置: %v", err)
	}

	// 2. 生成完整的配置文件
	// 如果能读取到现有配置，则提取基础部分并合并节点配置
	finalConfig := s.mergeConfigs(currentConfig, config)

	// 3. 通过SSH写入新配置
	log.Printf("[DEBUG] 通过SSH写入新的SLURM配置（长度: %d 字节）", len(finalConfig))

	// 使用here-doc方式写入，避免特殊字符问题
	writeCmd := fmt.Sprintf("cat > /etc/slurm/slurm.conf << 'SLURM_CONFIG_EOF'\n%s\nSLURM_CONFIG_EOF", finalConfig)
	_, err = s.executeSSHCommand(slurmMasterHost, 22, slurmMasterUser, slurmMasterPassword, writeCmd)
	if err != nil {
		return fmt.Errorf("SSH写入配置文件失败: %w", err)
	}

	// 4. 验证配置文件已正确写入
	verifyCmd := "wc -l /etc/slurm/slurm.conf"
	verifyOutput, err := s.executeSSHCommand(slurmMasterHost, 22, slurmMasterUser, slurmMasterPassword, verifyCmd)
	if err != nil {
		log.Printf("[WARNING] 无法验证配置文件: %v", err)
	} else {
		log.Printf("[DEBUG] 配置文件已验证: %s", strings.TrimSpace(verifyOutput))
	}

	log.Printf("[DEBUG] SLURM配置文件已通过SSH成功更新")
	return nil
}

// mergeConfigs 合并基础配置和节点配置
func (s *SlurmService) mergeConfigs(currentConfig, newNodesConfig string) string {
	// 如果无法读取当前配置，直接使用新生成的配置
	if currentConfig == "" {
		return newNodesConfig
	}

	// 提取当前配置中的基础部分（非NodeName和PartitionName行）
	var baseLines []string
	lines := strings.Split(currentConfig, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过节点和分区定义行
		if strings.HasPrefix(trimmed, "NodeName=") ||
			strings.HasPrefix(trimmed, "PartitionName=") {
			continue
		}
		baseLines = append(baseLines, line)
	}

	// 提取新配置中的节点和分区定义
	var nodeLines []string
	newLines := strings.Split(newNodesConfig, "\n")
	inNodeSection := false

	for _, line := range newLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# 节点配置") {
			inNodeSection = true
		}
		if inNodeSection && (strings.HasPrefix(trimmed, "NodeName=") ||
			strings.HasPrefix(trimmed, "PartitionName=")) {
			nodeLines = append(nodeLines, line)
		}
	}

	// 合并：基础配置 + 节点配置注释 + 节点定义
	result := strings.Join(baseLines, "\n")
	if len(nodeLines) > 0 {
		result += "\n\n# 节点配置（动态生成）\n"
		result += strings.Join(nodeLines, "\n")
	}

	return result
}

// reloadSlurmConfig 通过SSH重新加载SLURM配置
func (s *SlurmService) reloadSlurmConfig(ctx context.Context) error {
	log.Printf("[DEBUG] 开始通过SSH重新加载SLURM配置")

	slurmMasterHost := os.Getenv("SLURM_MASTER_HOST")
	if slurmMasterHost == "" {
		slurmMasterHost = "ai-infra-slurm-master"
	}

	slurmMasterUser := os.Getenv("SLURM_MASTER_USER")
	if slurmMasterUser == "" {
		slurmMasterUser = "root"
	}

	slurmMasterPassword := os.Getenv("SLURM_MASTER_PASSWORD")
	if slurmMasterPassword == "" {
		slurmMasterPassword = "root"
	}

	// 执行scontrol reconfigure
	reconfigCmd := "scontrol reconfigure"
	output, err := s.executeSSHCommand(slurmMasterHost, 22, slurmMasterUser, slurmMasterPassword, reconfigCmd)
	if err != nil {
		// 如果错误是"Zero Bytes were transmitted"，这实际上是成功的（命令执行了但没有输出）
		if strings.Contains(output, "Zero Bytes were transmitted") || strings.TrimSpace(output) == "" {
			log.Printf("[DEBUG] scontrol reconfigure执行成功（无输出）")
		} else {
			return fmt.Errorf("SSH执行scontrol reconfigure失败: %w, output: %s", err, output)
		}
	} else {
		log.Printf("[DEBUG] scontrol reconfigure成功: %s", strings.TrimSpace(output))
	}

	// 验证配置重新加载成功
	verifyCmd := "scontrol ping"
	verifyOutput, err := s.executeSSHCommand(slurmMasterHost, 22, slurmMasterUser, slurmMasterPassword, verifyCmd)
	if err != nil {
		log.Printf("[WARNING] 无法验证slurmctld状态: %v", err)
	} else {
		log.Printf("[DEBUG] slurmctld状态: %s", strings.TrimSpace(verifyOutput))
	}

	return nil
}

// executeSlurmCommand 执行SLURM命令的辅助函数（通过SSH连接到SLURM master）
func (s *SlurmService) executeSlurmCommand(ctx context.Context, command string) (string, error) {
	// 获取SLURM master的SSH连接信息
	slurmMasterHost := os.Getenv("SLURM_MASTER_HOST")
	if slurmMasterHost == "" {
		slurmMasterHost = "ai-infra-slurm-master"
	}

	slurmMasterUser := os.Getenv("SLURM_MASTER_USER")
	if slurmMasterUser == "" {
		slurmMasterUser = "root"
	}

	slurmMasterPassword := os.Getenv("SLURM_MASTER_PASSWORD")
	if slurmMasterPassword == "" {
		slurmMasterPassword = "root"
	}

	return s.executeSSHCommand(slurmMasterHost, 22, slurmMasterUser, slurmMasterPassword, command)
}

// ExecuteSlurmCommand 公开的SLURM命令执行方法（供Controller调用）
func (s *SlurmService) ExecuteSlurmCommand(ctx context.Context, command string) (string, error) {
	return s.executeSlurmCommand(ctx, command)
}

// executeSSHCommand 执行SSH命令的辅助函数（支持密码和密钥认证）
func (s *SlurmService) executeSSHCommand(host string, port int, user, password, command string) (string, error) {
	return s.executeSSHCommandWithKey(host, port, user, password, "", command)
}

// executeSSHCommandWithKey 执行SSH命令（支持密码和密钥认证）
func (s *SlurmService) executeSSHCommandWithKey(host string, port int, user, password, privateKey, command string) (string, error) {
	// 创建SSH客户端配置
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应使用正确的host key验证
		Timeout:         30 * time.Second,
	}

	// 设置认证方法
	var authMethods []ssh.AuthMethod

	// 优先使用私钥认证
	if privateKey != "" {
		var signer ssh.Signer
		var err error

		// 判断是文件路径还是密钥内容
		if strings.HasPrefix(privateKey, "-----BEGIN") {
			// 直接使用密钥内容
			signer, err = ssh.ParsePrivateKey([]byte(privateKey))
		} else {
			// 从文件读取密钥
			keyContent, readErr := os.ReadFile(privateKey)
			if readErr != nil {
				return "", fmt.Errorf("读取私钥文件失败: %v", readErr)
			}
			signer, err = ssh.ParsePrivateKey(keyContent)
		}

		if err != nil {
			return "", fmt.Errorf("解析私钥失败: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 如果提供了密码，添加密码认证
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return "", fmt.Errorf("未提供任何认证方式（密码或私钥）")
	}

	config.Auth = authMethods

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 执行命令
	output, err := session.CombinedOutput(command)
	outputStr := string(output)

	if err != nil {
		// 检查是否是退出码错误
		if exitErr, ok := err.(*ssh.ExitError); ok {
			// 返回更详细的错误信息，包含输出内容
			if outputStr != "" {
				return outputStr, fmt.Errorf("SSH命令执行失败 (退出码 %d): %s", exitErr.ExitStatus(), outputStr)
			}
			return outputStr, fmt.Errorf("SSH命令执行失败 (退出码 %d)", exitErr.ExitStatus())
		}
		// 其他SSH错误
		return outputStr, fmt.Errorf("SSH命令执行失败: %w (输出: %s)", err, outputStr)
	}

	return outputStr, nil
}

// getSaltAPIToken 获取Salt API认证token
func (s *SlurmService) getSaltAPIToken(ctx context.Context) (string, error) {
	saltAPIURL := os.Getenv("SALTSTACK_MASTER_URL")
	if saltAPIURL == "" {
		saltAPIURL = "http://saltstack:8002"
	}

	username := os.Getenv("SALT_API_USERNAME")
	if username == "" {
		username = "saltapi"
	}
	password := os.Getenv("SALT_API_PASSWORD")
	if password == "" {
		password = "your-salt-api-password"
	}
	eauth := os.Getenv("SALT_API_EAUTH")
	if eauth == "" {
		eauth = "file"
	}

	// 认证请求
	authPayload := map[string]string{
		"username": username,
		"password": password,
		"eauth":    eauth,
	}

	jsonData, err := json.Marshal(authPayload)
	if err != nil {
		return "", fmt.Errorf("序列化认证数据失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", saltAPIURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建认证请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("认证失败 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var authResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&authResult); err != nil {
		return "", fmt.Errorf("解析认证响应失败: %w", err)
	}

	// 提取token
	var token string
	if returnData, ok := authResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if t, ok := tokenData["token"].(string); ok {
				token = t
			}
		}
	}

	if token == "" {
		return "", fmt.Errorf("未能从认证响应中获取token")
	}

	return token, nil
}

// executeSaltCommand 执行Salt命令
func (s *SlurmService) executeSaltCommand(ctx context.Context, token string, cmd map[string]interface{}) (map[string]interface{}, error) {
	saltAPIURL := os.Getenv("SALTSTACK_MASTER_URL")
	if saltAPIURL == "" {
		saltAPIURL = "http://saltstack:8002"
	}

	jsonData, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("序列化命令数据失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", saltAPIURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result, nil
} // UpdateSlurmConfig 更新SLURM配置文件并重新加载
func (s *SlurmService) UpdateSlurmConfig(ctx context.Context, sshSvc SSHServiceInterface) error {
	// 获取当前所有活跃节点
	var nodes []models.SlurmNode
	if err := s.db.Where("status = ?", "active").Find(&nodes).Error; err != nil {
		return fmt.Errorf("failed to get active nodes: %w", err)
	}

	// 生成新的slurm.conf内容
	config := s.generateSlurmConfig(nodes)

	// 获取SLURM控制器信息
	controllerHost, controllerPort, err := s.getSlurmControllerInfo()
	if err != nil {
		return fmt.Errorf("failed to get SLURM controller info: %w", err)
	}

	// 上传新的配置文件到控制器
	configPath := "/etc/slurm/slurm.conf"
	if err := sshSvc.UploadFile(controllerHost, controllerPort, "root", "", []byte(config), configPath); err != nil {
		return fmt.Errorf("failed to upload slurm.conf: %w", err)
	}

	// 重新加载SLURM配置
	reloadCmd := "scontrol reconfigure"
	if _, err := sshSvc.ExecuteCommand(controllerHost, controllerPort, "root", "", reloadCmd); err != nil {
		return fmt.Errorf("failed to reload SLURM config: %w", err)
	}

	return nil
}

// generateSlurmConfig 生成SLURM配置文件内容
func (s *SlurmService) generateSlurmConfig(nodes []models.SlurmNode) string {
	// 只生成节点配置部分，不包含基础配置
	// 基础配置保留在SLURM master的原始配置文件中
	config := "# 节点配置（动态生成）\n"

	// 添加节点定义
	computeNodes := []string{}
	log.Printf("[DEBUG] generateSlurmConfig: 处理 %d 个节点", len(nodes))
	for i, node := range nodes {
		log.Printf("[DEBUG] 节点 #%d: NodeName=%s, NodeType=%s, Host=%s", i, node.NodeName, node.NodeType, node.Host)
		if node.NodeType == "compute" || node.NodeType == "node" {
			// 使用节点配置的硬件参数，如果为0则使用默认值
			cpus := node.CPUs
			if cpus == 0 {
				cpus = 2 // 默认2核
			}

			memory := node.Memory
			if memory == 0 {
				memory = 1000 // 默认1000MB
			}

			sockets := node.Sockets
			if sockets == 0 {
				sockets = 1 // 默认1个插槽
			}

			coresPerSocket := node.CoresPerSocket
			if coresPerSocket == 0 {
				coresPerSocket = cpus / sockets // 自动计算
				if coresPerSocket == 0 {
					coresPerSocket = 1
				}
			}

			threadsPerCore := node.ThreadsPerCore
			if threadsPerCore == 0 {
				threadsPerCore = 1 // 默认1个线程
			}

			// 构建节点配置字符串
			nodeConfig := fmt.Sprintf("NodeName=%s NodeAddr=%s CPUs=%d Sockets=%d CoresPerSocket=%d ThreadsPerCore=%d RealMemory=%d",
				node.NodeName, node.Host, cpus, sockets, coresPerSocket, threadsPerCore, memory)

			// 添加 GPU 配置（如果有）
			if node.GPUs > 0 {
				nodeConfig += fmt.Sprintf(" Gres=gpu:%d", node.GPUs)
			}

			// 添加 XPU 配置（如果有）
			if node.XPUs > 0 {
				if node.GPUs > 0 {
					// 如果已经有GPU配置，追加XPU
					nodeConfig += fmt.Sprintf(",xpu:%d", node.XPUs)
				} else {
					nodeConfig += fmt.Sprintf(" Gres=xpu:%d", node.XPUs)
				}
			}

			config += nodeConfig + "\n"
			computeNodes = append(computeNodes, node.NodeName)
			log.Printf("[DEBUG] 已添加计算节点: %s (地址: %s, CPU:%d, 内存:%dMB, GPU:%d, XPU:%d)",
				node.NodeName, node.Host, cpus, memory, node.GPUs, node.XPUs)
		} else {
			log.Printf("[WARNING] 跳过节点 %s，类型不匹配: %s", node.NodeName, node.NodeType)
		}
	}

	// 添加分区配置
	log.Printf("[DEBUG] 计算节点列表: %v", computeNodes)
	if len(computeNodes) > 0 {
		partitionConfig := fmt.Sprintf("PartitionName=compute Nodes=%s Default=YES MaxTime=INFINITE State=UP",
			strings.Join(computeNodes, ","))
		config += partitionConfig + "\n"
		log.Printf("[DEBUG] 已添加分区配置: %s", partitionConfig)
	} else {
		log.Printf("[WARNING] 没有计算节点，跳过分区配置")
	}

	return config
}

// getSlurmControllerInfo 获取SLURM控制器连接信息
func (s *SlurmService) getSlurmControllerInfo() (string, int, error) {
	// 从环境变量或配置中获取SLURM控制器地址
	// 这里假设使用Docker Compose中的服务名
	return "slurm-master", 22, nil
} // SSHServiceInterface 定义SSH服务接口以便测试
type SSHServiceInterface interface {
	UploadFile(host string, port int, user, password string, content []byte, remotePath string) error
	ExecuteCommand(host string, port int, user, password, command string) (string, error)
}

// ScaleDown 执行缩容操作
func (s *SlurmService) ScaleDown(ctx context.Context, nodeIDs []string) (*ScalingResult, error) {
	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	// 获取SLURM Master的连接信息
	masterHost := os.Getenv("SLURM_MASTER_HOST")
	if masterHost == "" {
		masterHost = "slurm-master" // 默认使用docker-compose服务名
	}

	masterPortStr := os.Getenv("SLURM_MASTER_PORT")
	masterPort := 22
	if masterPortStr != "" {
		if port, err := strconv.Atoi(masterPortStr); err == nil {
			masterPort = port
		}
	}

	masterUser := os.Getenv("SLURM_MASTER_USER")
	if masterUser == "" {
		masterUser = "root"
	}

	masterPassword := os.Getenv("SLURM_MASTER_PASSWORD")
	if masterPassword == "" {
		return nil, fmt.Errorf("未配置SLURM_MASTER_PASSWORD环境变量，无法连接SLURM Master")
	}

	// 建立SSH连接
	sshConfig := &ssh.ClientConfig{
		User: masterUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(masterPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", masterHost, masterPort), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("连接SLURM Master失败: %v", err)
	}
	defer client.Close()

	// 对每个节点执行缩容操作
	for _, nodeID := range nodeIDs {
		nodeResult := NodeScalingResult{
			NodeID:  nodeID,
			Success: false,
			Message: "",
		}

		// 步骤1: 将节点状态设置为DOWN
		downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason='缩容移除节点_%s'",
			nodeID, time.Now().Format("20060102_150405"))

		session, err := client.NewSession()
		if err != nil {
			nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		output, err := session.CombinedOutput(downCmd)
		session.Close()

		if err != nil {
			nodeResult.Message = fmt.Sprintf("设置节点DOWN状态失败: %v, 输出: %s", err, string(output))
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 步骤2: 从slurm.conf中移除节点
		configPath := "/etc/slurm/slurm.conf"

		// 读取配置文件
		session, err = client.NewSession()
		if err != nil {
			nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		configData, err := session.CombinedOutput(fmt.Sprintf("cat %s", configPath))
		session.Close()

		if err != nil {
			nodeResult.Message = fmt.Sprintf("读取slurm.conf失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 移除包含该节点的行
		lines := strings.Split(string(configData), "\n")
		var newLines []string
		removed := false
		for _, line := range lines {
			// 跳过包含该节点名称的NodeName行
			if strings.Contains(line, "NodeName="+nodeID) ||
				(strings.HasPrefix(line, "NodeName=") && strings.Contains(line, nodeID)) {
				removed = true
				continue
			}
			newLines = append(newLines, line)
		}

		if removed {
			// 写回配置文件
			newConfig := strings.Join(newLines, "\n")

			session, err = client.NewSession()
			if err != nil {
				nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}

			// 使用临时文件并移动的方式更新配置
			tmpPath := "/tmp/slurm.conf.tmp"
			writeCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF\nmv %s %s", tmpPath, newConfig, tmpPath, configPath)
			output, err = session.CombinedOutput(writeCmd)
			session.Close()

			if err != nil {
				nodeResult.Message = fmt.Sprintf("更新slurm.conf失败: %v, 输出: %s", err, string(output))
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}

			// 步骤3: 重新加载SLURM配置
			session, err = client.NewSession()
			if err != nil {
				nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}

			output, err = session.CombinedOutput("scontrol reconfigure")
			session.Close()

			if err != nil {
				nodeResult.Message = fmt.Sprintf("重新加载SLURM配置失败: %v, 输出: %s", err, string(output))
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}
		} else {
			nodeResult.Message = fmt.Sprintf("在slurm.conf中未找到节点 %s", nodeID)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 成功
		nodeResult.Success = true
		nodeResult.Message = "节点已成功从SLURM集群中移除"
		result.Results = append(result.Results, nodeResult)
	}

	// 如果所有操作都失败，整体标记为失败
	allFailed := true
	for _, r := range result.Results {
		if r.Success {
			allFailed = false
			break
		}
	}
	if allFailed {
		result.Success = false
	}

	return result, nil
}

// GetNodeTemplates 获取节点模板列表
func (s *SlurmService) GetNodeTemplates(ctx context.Context) ([]NodeTemplate, error) {
	// 如果数据库没有数据，返回一些默认模板
	var templates []models.NodeTemplate
	if err := s.db.Find(&templates).Error; err != nil {
		// 如果数据库查询失败，返回默认模板
		defaultTemplates := []NodeTemplate{
			{
				ID:          "small",
				Name:        "小型计算节点",
				Description: "2核4GB内存，适合轻量级计算任务",
				Config: NodeConfig{
					Host:     "compute-small",
					Port:     22,
					User:     "root",
					KeyPath:  "",
					Password: "",
					MinionID: "",
				},
				Tags:      []string{"small", "compute"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "medium",
				Name:        "中型计算节点",
				Description: "4核8GB内存，适合中等规模计算任务",
				Config: NodeConfig{
					Host:     "compute-medium",
					Port:     22,
					User:     "root",
					KeyPath:  "",
					Password: "",
					MinionID: "",
				},
				Tags:      []string{"medium", "compute"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "large",
				Name:        "大型计算节点",
				Description: "8核16GB内存，适合大规模计算任务",
				Config: NodeConfig{
					Host:     "compute-large",
					Port:     22,
					User:     "root",
					KeyPath:  "",
					Password: "",
					MinionID: "",
				},
				Tags:      []string{"large", "compute", "gpu"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		return defaultTemplates, nil
	}

	// 转换数据库中的模板为服务层结构
	result := make([]NodeTemplate, len(templates))
	for i, t := range templates {
		// 安全地转换模型层的 NodeConfig
		serviceConfig := NodeConfig{
			Host:     "localhost", // 默认主机
			Port:     22,          // 默认SSH端口
			User:     "root",      // 默认用户
			KeyPath:  "",
			Password: "",
			MinionID: "",
		}

		// 如果配置中有分区信息，使用第一个作为主机名
		if len(t.Config.Partitions) > 0 {
			serviceConfig.Host = t.Config.Partitions[0]
		}

		result[i] = NodeTemplate{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Config:      serviceConfig,
			Tags:        t.Tags,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		}
	}
	return result, nil
}

// CreateNodeTemplate 创建节点模板
func (s *SlurmService) CreateNodeTemplate(ctx context.Context, template *NodeTemplate) error {
	// 获取当前用户ID（需要从context中获取，这里暂时使用默认值）
	userID := uint(1) // TODO: 从JWT中获取用户ID

	// 将服务层的 NodeConfig 转换为模型层的 NodeConfig
	modelConfig := models.NodeConfig{
		Partitions:     []string{template.Config.Host}, // 使用主机名作为分区
		Features:       []string{},
		CustomSettings: map[string]string{},
		Mounts:         []models.MountConfig{},
	}

	model := &models.NodeTemplate{
		ID:          generateTemplateID(),
		Name:        template.Name,
		Description: template.Description,
		Config:      modelConfig,
		Tags:        template.Tags,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	template.ID = model.ID
	template.CreatedAt = model.CreatedAt
	template.UpdatedAt = model.UpdatedAt
	return nil
}

// UpdateNodeTemplate 更新节点模板
func (s *SlurmService) UpdateNodeTemplate(ctx context.Context, id string, template *NodeTemplate) error {
	var model models.NodeTemplate
	if err := s.db.Where("id = ?", id).First(&model).Error; err != nil {
		return err
	}

	// 将服务层的 NodeConfig 转换为模型层的 NodeConfig
	modelConfig := models.NodeConfig{
		Partitions:     []string{template.Config.Host}, // 使用主机名作为分区
		Features:       []string{},
		CustomSettings: map[string]string{},
		Mounts:         []models.MountConfig{},
	}

	model.Name = template.Name
	model.Description = template.Description
	model.Config = modelConfig
	model.Tags = template.Tags
	model.UpdatedAt = time.Now()

	return s.db.Save(&model).Error
}

// DeleteNodeTemplate 删除节点模板
func (s *SlurmService) DeleteNodeTemplate(ctx context.Context, id string) error {
	return s.db.Where("id = ?", id).Delete(&models.NodeTemplate{}).Error
}

// generateOperationID 生成操作ID
func generateOperationID() string {
	return fmt.Sprintf("op-%d", time.Now().Unix())
}

// SLURM REST API 相关结构体和方法

// SlurmNodeSpec REST API节点规格
type SlurmNodeSpec struct {
	NodeName string `json:"name"`
	CPUs     int    `json:"cpus"`
	Memory   int    `json:"real_memory"` // MB
	Features string `json:"features,omitempty"`
	Gres     string `json:"gres,omitempty"`
	State    string `json:"state,omitempty"`
}

// SlurmAPIResponse REST API响应
type SlurmAPIResponse struct {
	Meta   map[string]interface{} `json:"meta,omitempty"`
	Errors []SlurmAPIError        `json:"errors,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
}

// SlurmAPIError REST API错误
type SlurmAPIError struct {
	Error       string `json:"error"`
	ErrorNumber int    `json:"error_number"`
	Source      string `json:"source,omitempty"`
}

// SlurmNodeUpdate 节点更新请求
type SlurmNodeUpdate struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// getJWTToken 获取SLURM JWT token
func (s *SlurmService) getJWTToken(ctx context.Context) (string, error) {
	// 尝试从环境变量获取预设的token
	if token := os.Getenv("SLURM_JWT_TOKEN"); token != "" {
		return token, nil
	}

	// 通过SSH连接获取token
	masterHost := os.Getenv("SLURM_MASTER_HOST")
	if masterHost == "" {
		masterHost = "slurm-master"
	}

	masterPortStr := os.Getenv("SLURM_MASTER_PORT")
	masterPort := 22
	if masterPortStr != "" {
		if port, err := strconv.Atoi(masterPortStr); err == nil {
			masterPort = port
		}
	}

	masterUser := os.Getenv("SLURM_MASTER_USER")
	if masterUser == "" {
		masterUser = "root"
	}

	masterPassword := os.Getenv("SLURM_MASTER_PASSWORD")
	if masterPassword == "" {
		return "", fmt.Errorf("未配置SLURM_MASTER_PASSWORD环境变量")
	}

	sshConfig := &ssh.ClientConfig{
		User: masterUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(masterPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", masterHost, masterPort), sshConfig)
	if err != nil {
		return "", fmt.Errorf("连接SLURM Master失败: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	// 获取token
	output, err := session.CombinedOutput("scontrol token lifespan=3600")
	if err != nil {
		return "", fmt.Errorf("获取JWT token失败: %v, 输出: %s", err, string(output))
	}

	// 解析token输出 (格式: SLURM_JWT=xxxxx)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SLURM_JWT=") {
			return strings.TrimPrefix(line, "SLURM_JWT="), nil
		}
	}

	return "", fmt.Errorf("未找到有效的JWT token")
}

// callSlurmAPI 调用SLURM REST API
func (s *SlurmService) callSlurmAPI(ctx context.Context, method, endpoint string, body interface{}) (*SlurmAPIResponse, error) {
	// 获取JWT token
	token, err := s.getJWTToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取JWT token失败: %v", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/slurm/v0.0.41%s", s.restAPIURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("X-SLURM-USER-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp SlurmAPIResponse
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		// 如果不是JSON，返回原始响应
		log.Printf("[WARN] 解析JSON响应失败: %v, 响应: %s", err, string(respData))
		return &SlurmAPIResponse{
			Data: string(respData),
		}, nil
	}

	// 检查API错误
	if len(apiResp.Errors) > 0 {
		return &apiResp, fmt.Errorf("SLURM API错误: %s", apiResp.Errors[0].Error)
	}

	return &apiResp, nil
}

// AddNodeViaAPI 通过REST API添加节点
func (s *SlurmService) AddNodeViaAPI(ctx context.Context, nodeSpec SlurmNodeSpec) error {
	log.Printf("[DEBUG] AddNodeViaAPI: 添加节点 %s", nodeSpec.NodeName)

	// 调用API添加节点
	resp, err := s.callSlurmAPI(ctx, "POST", "/nodes", map[string]interface{}{
		"nodes": []SlurmNodeSpec{nodeSpec},
	})

	if err != nil {
		return fmt.Errorf("添加节点失败: %v", err)
	}

	log.Printf("[DEBUG] AddNodeViaAPI: 节点 %s 添加成功, 响应: %+v", nodeSpec.NodeName, resp)
	return nil
}

// UpdateNodeViaAPI 通过REST API更新节点状态
func (s *SlurmService) UpdateNodeViaAPI(ctx context.Context, nodeName string, update SlurmNodeUpdate) error {
	log.Printf("[DEBUG] UpdateNodeViaAPI: 更新节点 %s 状态为 %s", nodeName, update.State)

	// 调用API更新节点
	resp, err := s.callSlurmAPI(ctx, "POST", fmt.Sprintf("/node/%s", nodeName), update)
	if err != nil {
		return fmt.Errorf("更新节点状态失败: %v", err)
	}

	log.Printf("[DEBUG] UpdateNodeViaAPI: 节点 %s 状态更新成功, 响应: %+v", nodeName, resp)
	return nil
}

// CancelJobViaAPI 通过REST API取消作业
func (s *SlurmService) CancelJobViaAPI(ctx context.Context, jobID string, signal string) error {
	log.Printf("[DEBUG] CancelJobViaAPI: 取消作业 %s (信号: %s)", jobID, signal)

	payload := map[string]interface{}{
		"job_id": jobID,
	}
	if signal != "" {
		payload["signal"] = signal
	}

	resp, err := s.callSlurmAPI(ctx, "DELETE", fmt.Sprintf("/job/%s", jobID), payload)
	if err != nil {
		return fmt.Errorf("取消作业失败: %v", err)
	}

	log.Printf("[DEBUG] CancelJobViaAPI: 作业 %s 取消成功, 响应: %+v", jobID, resp)
	return nil
}

// UpdateJobViaAPI 通过REST API更新作业状态
func (s *SlurmService) UpdateJobViaAPI(ctx context.Context, jobID string, action string) error {
	log.Printf("[DEBUG] UpdateJobViaAPI: 对作业 %s 执行 %s 操作", jobID, action)

	payload := map[string]interface{}{
		"job_id": jobID,
		"action": action,
	}

	resp, err := s.callSlurmAPI(ctx, "POST", fmt.Sprintf("/job/%s", jobID), payload)
	if err != nil {
		return fmt.Errorf("更新作业状态失败: %v", err)
	}

	log.Printf("[DEBUG] UpdateJobViaAPI: 作业 %s 操作 %s 成功, 响应: %+v", jobID, action, resp)
	return nil
}

// DeleteNodeViaAPI 通过REST API删除节点
func (s *SlurmService) DeleteNodeViaAPI(ctx context.Context, nodeName string) error {
	log.Printf("[DEBUG] DeleteNodeViaAPI: 删除节点 %s", nodeName)

	// 先将节点设置为DOWN状态
	err := s.UpdateNodeViaAPI(ctx, nodeName, SlurmNodeUpdate{
		State:  "DOWN",
		Reason: fmt.Sprintf("节点删除_%s", time.Now().Format("20060102_150405")),
	})
	if err != nil {
		return fmt.Errorf("设置节点DOWN状态失败: %v", err)
	}

	// 调用API删除节点
	resp, err := s.callSlurmAPI(ctx, "DELETE", fmt.Sprintf("/node/%s", nodeName), nil)
	if err != nil {
		return fmt.Errorf("删除节点失败: %v", err)
	}

	log.Printf("[DEBUG] DeleteNodeViaAPI: 节点 %s 删除成功, 响应: %+v", nodeName, resp)
	return nil
}

// ScaleUpViaAPI 通过REST API扩容
func (s *SlurmService) ScaleUpViaAPI(ctx context.Context, nodes []NodeConfig) (*ScalingResult, error) {
	log.Printf("[DEBUG] ScaleUpViaAPI: 开始基于REST API的扩容操作，节点数量: %d", len(nodes))

	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	for _, node := range nodes {
		nodeResult := NodeScalingResult{
			NodeID:  node.Host,
			Success: false,
			Message: "",
		}

		// 创建节点规格 (从配置读取，提供默认值)
		cpus := node.CPUs
		if cpus == 0 {
			cpus = 4 // 默认4核
		}

		memory := node.Memory
		if memory == 0 {
			memory = 8192 // 默认8GB内存 (MB)
		}

		features := node.Features
		if features == "" {
			features = "compute" // 默认特性
		}

		nodeSpec := SlurmNodeSpec{
			NodeName: node.Host,
			CPUs:     cpus,
			Memory:   memory,
			Features: features,
			State:    "DOWN", // 新节点初始状态设置为 DOWN，等待手动激活
		}

		log.Printf("[DEBUG] ScaleUpViaAPI: 节点 %s 配置: CPUs=%d, Memory=%d MB, Features=%s, State=DOWN (需手动激活)",
			node.Host, cpus, memory, features)

		// 通过API添加节点
		if err := s.AddNodeViaAPI(ctx, nodeSpec); err != nil {
			nodeResult.Message = fmt.Sprintf("API添加节点失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 更新数据库状态
		if s.db != nil {
			var dbNode models.SlurmNode
			err := s.db.Where("host = ? OR node_name = ?", node.Host, node.MinionID).First(&dbNode).Error

			if err == gorm.ErrRecordNotFound {
				// 创建新节点记录
				dbNode = models.SlurmNode{
					NodeName:     node.Host,
					Host:         node.Host,
					CPUs:         4,    // 默认4核
					Memory:       8192, // 默认8GB内存
					Status:       "active",
					SaltMinionID: node.MinionID,
				}
				if err := s.db.Create(&dbNode).Error; err != nil {
					log.Printf("[WARN] 创建节点 %s 数据库记录失败: %v", node.Host, err)
				}
			} else if err == nil {
				// 更新现有记录
				if err := s.db.Model(&dbNode).Updates(map[string]interface{}{
					"status":         "active",
					"salt_minion_id": node.MinionID,
				}).Error; err != nil {
					log.Printf("[WARN] 更新节点 %s 数据库记录失败: %v", node.Host, err)
				}
			}
		}

		nodeResult.Success = true
		nodeResult.Message = "节点已添加到SLURM集群(状态: DOWN)，请确认slurmd运行后执行: scontrol update NodeName=" + node.Host + " State=RESUME"
		result.Results = append(result.Results, nodeResult)

		log.Printf("[DEBUG] 节点 %s 通过REST API扩容成功", node.Host)
	}

	log.Printf("[INFO] ScaleUpViaAPI 完成，所有节点已添加为 DOWN 状态，需手动激活")

	return result, nil
}

// ScaleDownViaAPI 通过REST API缩容
func (s *SlurmService) ScaleDownViaAPI(ctx context.Context, nodeIDs []string) (*ScalingResult, error) {
	log.Printf("[DEBUG] ScaleDownViaAPI: 开始基于REST API的缩容操作，节点数量: %d", len(nodeIDs))

	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	for _, nodeID := range nodeIDs {
		nodeResult := NodeScalingResult{
			NodeID:  nodeID,
			Success: false,
			Message: "",
		}

		// 通过API删除节点
		if err := s.DeleteNodeViaAPI(ctx, nodeID); err != nil {
			nodeResult.Message = fmt.Sprintf("API删除节点失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 更新数据库状态
		if s.db != nil {
			if err := s.db.Model(&models.SlurmNode{}).Where("node_name = ? OR host = ?", nodeID, nodeID).
				Updates(map[string]interface{}{
					"status": "removed",
				}).Error; err != nil {
				log.Printf("[WARN] 更新节点 %s 数据库状态失败: %v", nodeID, err)
			}
		}

		nodeResult.Success = true
		nodeResult.Message = "节点成功从SLURM集群移除"
		result.Results = append(result.Results, nodeResult)

		log.Printf("[DEBUG] 节点 %s 通过REST API缩容成功", nodeID)
	}

	return result, nil
}

// ReloadSlurmConfig 重新加载SLURM配置
func (s *SlurmService) ReloadSlurmConfig(ctx context.Context) error {
	log.Printf("[DEBUG] ReloadSlurmConfig: 重新加载SLURM配置")

	// 调用API重新加载配置
	resp, err := s.callSlurmAPI(ctx, "POST", "/reconfigure", nil)
	if err != nil {
		return fmt.Errorf("重新加载SLURM配置失败: %v", err)
	}

	log.Printf("[DEBUG] SLURM配置重新加载成功, 响应: %+v", resp)
	return nil
}

// generateTemplateID 生成模板ID
func generateTemplateID() string {
	return fmt.Sprintf("tmpl-%d", time.Now().Unix())
}

// InstallSlurmNodeRequest 安装SLURM节点请求
type InstallSlurmNodeRequest struct {
	NodeName string `json:"node_name"` // 节点名称（容器名）
	OSType   string `json:"os_type"`   // 操作系统类型：rocky 或 ubuntu
}

// InstallSlurmNodeResponse 安装SLURM节点响应
type InstallSlurmNodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Logs    string `json:"logs"` // 安装日志
}

// InstallSlurmNode 在指定节点上安装SLURM客户端和slurmd
func (s *SlurmService) InstallSlurmNode(ctx context.Context, req InstallSlurmNodeRequest) (*InstallSlurmNodeResponse, error) {
	log.Printf("[INFO] 开始在节点 %s 上安装SLURM (OS: %s)", req.NodeName, req.OSType)

	var logBuffer bytes.Buffer
	logWriter := io.MultiWriter(os.Stdout, &logBuffer)

	// 1. 获取slurm.conf和munge.key
	log.Printf("[INFO] 从slurm-master获取配置文件...")
	slurmConf, err := s.getSlurmMasterConfig(ctx)
	if err != nil {
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("获取slurm.conf失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	mungeKey, err := s.getMungeKey(ctx)
	if err != nil {
		log.Printf("[WARN] 获取munge.key失败: %v，将继续安装", err)
		mungeKey = nil
	}

	// 2. 根据OS类型安装SLURM
	if err := s.installSlurmPackages(ctx, req.NodeName, req.OSType, logWriter); err != nil {
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("安装SLURM包失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	// 3. 配置文件
	if err := s.configureSlurmNode(ctx, req.NodeName, req.OSType, slurmConf, mungeKey, logWriter); err != nil {
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("配置节点失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	// 4. 启动服务
	if err := s.startSlurmServices(ctx, req.NodeName, req.OSType, logWriter); err != nil {
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("启动服务失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	log.Printf("[INFO] 节点 %s SLURM安装完成", req.NodeName)
	return &InstallSlurmNodeResponse{
		Success: true,
		Message: "SLURM安装成功",
		Logs:    logBuffer.String(),
	}, nil
}

// getSlurmMasterConfig 从slurm-master获取slurm.conf
func (s *SlurmService) getSlurmMasterConfig(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", "exec", "ai-infra-slurm-master", "cat", "/etc/slurm/slurm.conf")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("读取slurm.conf失败: %v", err)
	}
	return output, nil
}

// getMungeKey 从slurm-master获取munge.key
func (s *SlurmService) getMungeKey(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", "exec", "ai-infra-slurm-master", "cat", "/etc/munge/munge.key")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

// installSlurmPackages 安装SLURM包（使用 SaltStack 客户端执行）
func (s *SlurmService) installSlurmPackages(ctx context.Context, nodeName, osType string, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 在 %s 上使用 Salt 客户端安装SLURM包 (OS: %s)\n", nodeName, osType)

	// 读取安装脚本（路径根据 Dockerfile 中的 COPY 命令）
	scriptPath := "/root/scripts/install-slurm-node.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取安装脚本失败 %s: %w", scriptPath, err)
	}

	// 获取apphub URL
	apphubURL := os.Getenv("APPHUB_URL")
	if apphubURL == "" {
		apphubURL = "http://ai-infra-apphub" // 默认URL（AppHub 使用端口 80）
	}

	// Salt master 主机（从环境变量获取，默认为 ai-infra-saltstack）
	saltMaster := os.Getenv("SALT_MASTER_HOST")
	if saltMaster == "" {
		saltMaster = "ai-infra-saltstack" // 实际的 Salt Master 容器名
	}

	fmt.Fprintf(logWriter, "[INFO] 使用 Salt Master: %s\n", saltMaster)
	fmt.Fprintf(logWriter, "[INFO] AppHub URL: %s\n", apphubURL)

	// 1. 将脚本内容写入目标节点的临时文件
	tmpScriptPath := fmt.Sprintf("/tmp/install-slurm-%s.sh", nodeName)
	writeScriptCmd := fmt.Sprintf(`cat > %s << 'SCRIPT_EOF'
%s
SCRIPT_EOF
chmod +x %s`, tmpScriptPath, string(scriptContent), tmpScriptPath)

	// 使用 salt 命令将脚本写入节点
	saltWriteCmd := exec.CommandContext(ctx, "docker", "exec", saltMaster,
		"salt", nodeName, "cmd.run", writeScriptCmd)
	saltWriteCmd.Stdout = logWriter
	saltWriteCmd.Stderr = logWriter

	fmt.Fprintf(logWriter, "[INFO] 上传安装脚本到节点 %s...\n", nodeName)
	if err := saltWriteCmd.Run(); err != nil {
		return fmt.Errorf("上传脚本到节点失败: %v", err)
	}

	// 2. 使用 salt 命令执行安装脚本
	executeScriptCmd := fmt.Sprintf("%s %s compute", tmpScriptPath, apphubURL)

	saltExecCmd := exec.CommandContext(ctx, "docker", "exec", saltMaster,
		"salt", nodeName, "cmd.run", executeScriptCmd,
		"timeout=600") // 10分钟超时

	saltExecCmd.Stdout = logWriter
	saltExecCmd.Stderr = logWriter

	fmt.Fprintf(logWriter, "[INFO] 通过 Salt 执行安装脚本...\n")
	if err := saltExecCmd.Run(); err != nil {
		return fmt.Errorf("通过 Salt 执行安装脚本失败: %v", err)
	}

	// 3. 清理临时脚本
	saltCleanCmd := exec.CommandContext(ctx, "docker", "exec", saltMaster,
		"salt", nodeName, "cmd.run", fmt.Sprintf("rm -f %s", tmpScriptPath))
	saltCleanCmd.Run() // 忽略清理错误

	fmt.Fprintf(logWriter, "[INFO] ✓ SLURM包安装成功（通过 Salt）\n")
	return nil
}

// configureSlurmNode 配置SLURM节点
func (s *SlurmService) configureSlurmNode(ctx context.Context, nodeName, osType string, slurmConf, mungeKey []byte, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 配置 %s 节点\n", nodeName)

	// 确定配置文件路径
	slurmConfPath := "/etc/slurm/slurm.conf"
	if osType == "ubuntu" || osType == "debian" {
		slurmConfPath = "/etc/slurm-llnl/slurm.conf"
	}

	// 写入临时文件
	tmpSlurmConf := "/tmp/slurm.conf." + nodeName
	if err := os.WriteFile(tmpSlurmConf, slurmConf, 0644); err != nil {
		return fmt.Errorf("写入临时slurm.conf失败: %v", err)
	}
	defer os.Remove(tmpSlurmConf)

	// 复制slurm.conf到容器
	cmd := exec.CommandContext(ctx, "docker", "cp", tmpSlurmConf, nodeName+":"+slurmConfPath)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("复制slurm.conf失败: %v", err)
	}
	fmt.Fprintf(logWriter, "[INFO] slurm.conf复制成功 -> %s\n", slurmConfPath)

	// 如果有munge.key，也复制
	if mungeKey != nil && len(mungeKey) > 0 {
		tmpMungeKey := "/tmp/munge.key." + nodeName
		if err := os.WriteFile(tmpMungeKey, mungeKey, 0400); err != nil {
			return fmt.Errorf("写入临时munge.key失败: %v", err)
		}
		defer os.Remove(tmpMungeKey)

		cmd = exec.CommandContext(ctx, "docker", "cp", tmpMungeKey, nodeName+":/etc/munge/munge.key")
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		if err := cmd.Run(); err != nil {
			log.Printf("[WARN] 复制munge.key失败: %v", err)
		} else {
			// 设置权限
			cmd = exec.CommandContext(ctx, "docker", "exec", nodeName, "chown", "munge:munge", "/etc/munge/munge.key")
			cmd.Run()
			cmd = exec.CommandContext(ctx, "docker", "exec", nodeName, "chmod", "400", "/etc/munge/munge.key")
			cmd.Run()
			fmt.Fprintf(logWriter, "[INFO] munge.key配置成功\n")
		}
	}

	return nil
}

// startSlurmServices 启动SLURM服务（通过SSH远程执行脚本）
func (s *SlurmService) startSlurmServices(ctx context.Context, nodeName, osType string, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 启动 %s 上的服务\n", nodeName)

	// 1. 读取启动脚本
	scriptPath := "src/backend/scripts/start-slurmd.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取启动脚本失败: %v", err)
	}

	fmt.Fprintf(logWriter, "[INFO] 使用脚本启动 slurmd...\n")

	// 2. 通过 docker exec 执行脚本
	cmd := exec.CommandContext(ctx, "docker", "exec", nodeName, "bash", "-c", string(scriptContent))
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] 执行启动脚本失败: %v", err)
		fmt.Fprintf(logWriter, "[ERROR] 执行启动脚本失败: %v\n", err)
		return fmt.Errorf("启动服务失败: %v", err)
	}

	fmt.Fprintf(logWriter, "[SUCCESS] 服务启动完成\n")

	// 3. 验证服务状态
	time.Sleep(2 * time.Second)

	checkCmd := "pgrep -x slurmd >/dev/null && echo 'slurmd running' || echo 'slurmd not running'"
	cmd = exec.CommandContext(ctx, "docker", "exec", nodeName, "bash", "-c", checkCmd)
	output, _ := cmd.Output()
	fmt.Fprintf(logWriter, "[INFO] 验证结果: %s\n", strings.TrimSpace(string(output)))

	return nil
}

// executeScriptViaSSH 通过SSH执行脚本（支持密码和密钥认证）
func (s *SlurmService) executeScriptViaSSH(ctx context.Context, host string, port int, user, password, privateKey, scriptPath string) (string, error) {
	// 读取脚本内容
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("读取脚本失败: %v", err)
	}

	// 通过SSH执行脚本
	return s.executeSSHCommandWithKey(host, port, user, password, privateKey, string(scriptContent))
}

// startSlurmServicesViaSSH 通过真实SSH启动SLURM服务（非docker exec）
func (s *SlurmService) startSlurmServicesViaSSH(ctx context.Context, host string, port int, user, password, privateKey string, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 通过SSH启动 %s 上的服务\n", host)

	// 执行启动脚本
	scriptPath := "src/backend/scripts/start-slurmd.sh"
	output, err := s.executeScriptViaSSH(ctx, host, port, user, password, privateKey, scriptPath)

	if err != nil {
		fmt.Fprintf(logWriter, "[ERROR] 执行启动脚本失败: %v\n", err)
		fmt.Fprintf(logWriter, "[ERROR] 输出: %s\n", output)
		return fmt.Errorf("启动服务失败: %v", err)
	}

	fmt.Fprintf(logWriter, "[SUCCESS] 脚本执行输出:\n%s\n", output)
	return nil
}

// checkSlurmServicesViaSSH 通过SSH检查SLURM服务状态
func (s *SlurmService) checkSlurmServicesViaSSH(ctx context.Context, host string, port int, user, password, privateKey string) (string, error) {
	scriptPath := "src/backend/scripts/check-slurmd.sh"
	return s.executeScriptViaSSH(ctx, host, port, user, password, privateKey, scriptPath)
}

// stopSlurmServicesViaSSH 通过SSH停止SLURM服务
func (s *SlurmService) stopSlurmServicesViaSSH(ctx context.Context, host string, port int, user, password, privateKey string) (string, error) {
	scriptPath := "src/backend/scripts/stop-slurmd.sh"
	return s.executeScriptViaSSH(ctx, host, port, user, password, privateKey, scriptPath)
}

// detectNodeOSType 检测节点的操作系统类型
func (s *SlurmService) detectNodeOSType(ctx context.Context, nodeName string) string {
	// 尝试检测操作系统类型
	cmd := exec.CommandContext(ctx, "docker", "exec", nodeName, "cat", "/etc/os-release")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[WARN] 检测节点 %s 操作系统失败: %v", nodeName, err)
		return ""
	}

	outputStr := string(output)
	// 检查是否是 Rocky Linux
	if strings.Contains(outputStr, "Rocky") || strings.Contains(outputStr, "rocky") {
		return "rocky"
	}
	// 检查是否是 CentOS
	if strings.Contains(outputStr, "CentOS") || strings.Contains(outputStr, "centos") {
		return "centos"
	}
	// 检查是否是 Ubuntu
	if strings.Contains(outputStr, "Ubuntu") || strings.Contains(outputStr, "ubuntu") {
		return "ubuntu"
	}
	// 检查是否是 Debian
	if strings.Contains(outputStr, "Debian") || strings.Contains(outputStr, "debian") {
		return "debian"
	}

	log.Printf("[WARN] 无法识别节点 %s 的操作系统类型", nodeName)
	return ""
}

// BatchInstallSlurmNodes 批量安装SLURM节点（并发）
func (s *SlurmService) BatchInstallSlurmNodes(ctx context.Context, nodes []InstallSlurmNodeRequest) (map[string]*InstallSlurmNodeResponse, error) {
	results := make(map[string]*InstallSlurmNodeResponse)
	var mu sync.Mutex // 保护 results map
	var wg sync.WaitGroup

	// 使用 channel 来限制并发数，避免过多并发导致系统负载过高
	maxConcurrency := 5
	semaphore := make(chan struct{}, maxConcurrency)

	for _, node := range nodes {
		wg.Add(1)
		go func(n InstallSlurmNodeRequest) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resp, err := s.InstallSlurmNode(ctx, n)
			if err != nil {
				resp = &InstallSlurmNodeResponse{
					Success: false,
					Message: err.Error(),
				}
			}

			// 写入结果时加锁
			mu.Lock()
			results[n.NodeName] = resp
			mu.Unlock()
		}(node)
	}

	// 等待所有安装任务完成
	wg.Wait()

	return results, nil
}
