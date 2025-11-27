package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SaltStackClientController 处理SaltStack客户端安装相关的HTTP请求
type SaltStackClientController struct {
	service             *services.SaltStackClientService
	batchInstallService *services.BatchInstallService
}

func NewSaltStackClientController() *SaltStackClientController {
	return &SaltStackClientController{
		service:             services.NewSaltStackClientService(),
		batchInstallService: services.NewBatchInstallService(),
	}
}

// InstallSaltMinionAsync 异步安装SaltStack Minion客户端
// @Summary 异步安装SaltStack Minion客户端
// @Description 在指定的主机上异步安装SaltStack Minion客户端
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body services.InstallRequest true "安装请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saltstack/install [post]
func (c *SaltStackClientController) InstallSaltMinionAsync(ctx *gin.Context) {
	var req services.InstallRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind install request")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// 验证请求参数
	if len(req.Hosts) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "No hosts specified",
			"message": "At least one host must be specified for installation",
		})
		return
	}

	// 设置默认并发数
	if req.Parallel <= 0 {
		req.Parallel = 3
	}

	taskIDs, err := c.service.InstallSaltMinionAsync(ctx.Request.Context(), req)
	if err != nil {
		logrus.WithError(err).Error("Failed to start salt minion installation")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start installation",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Installation tasks created successfully",
		"task_ids": taskIDs,
		"count":    len(taskIDs),
	})

	logrus.WithField("task_count", len(taskIDs)).Info("SaltStack Minion installation tasks created")
}

// GetInstallTask 获取安装任务状态
// @Summary 获取安装任务状态
// @Description 获取指定安装任务的详细状态信息
// @Tags SaltStack
// @Produce json
// @Param taskId path string true "任务ID"
// @Success 200 {object} services.InstallTask
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saltstack/install/{taskId} [get]
func (c *SaltStackClientController) GetInstallTask(ctx *gin.Context) {
	taskID := ctx.Param("taskId")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Task ID required",
			"message": "Task ID parameter is required",
		})
		return
	}

	task, err := c.service.GetInstallTask(taskID)
	if err != nil {
		logrus.WithError(err).WithField("task_id", taskID).Error("Failed to get install task")
		ctx.JSON(http.StatusNotFound, gin.H{
			"error":   "Task not found",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, task)
}

// ListInstallTasks 列出所有安装任务
// @Summary 列出所有安装任务
// @Description 获取所有SaltStack Minion安装任务的列表
// @Tags SaltStack
// @Produce json
// @Param status query string false "任务状态过滤" Enums(pending,running,success,failed)
// @Param limit query int false "限制返回数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/install [get]
func (c *SaltStackClientController) ListInstallTasks(ctx *gin.Context) {
	// 获取查询参数
	statusFilter := ctx.Query("status")
	limitStr := ctx.DefaultQuery("limit", "50")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	tasks := c.service.ListInstallTasks()

	// 过滤任务
	var filteredTasks []*services.InstallTask
	for _, task := range tasks {
		if statusFilter == "" || task.Status == statusFilter {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// 分页
	total := len(filteredTasks)
	start := offset
	if start > total {
		start = total
	}

	end := start + limit
	if end > total {
		end = total
	}

	paginatedTasks := filteredTasks[start:end]

	ctx.JSON(http.StatusOK, gin.H{
		"tasks":  paginatedTasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})

	logrus.WithFields(logrus.Fields{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"status": statusFilter,
	}).Debug("Listed SaltStack install tasks")
}

// GetTestHosts 获取测试主机列表
// @Summary 获取测试主机列表
// @Description 获取可用的SSH测试主机列表
// @Tags SaltStack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/test-hosts [get]
func (c *SaltStackClientController) GetTestHosts(ctx *gin.Context) {
	testHosts := []map[string]interface{}{
		{
			"name":          "test-ssh01",
			"host":          "test-ssh01",
			"port":          22,
			"external_port": 2201,
			"username":      "testuser",
			"password":      "testpass123",
			"description":   "SSH测试容器1",
		},
		{
			"name":          "test-ssh02",
			"host":          "test-ssh02",
			"port":          22,
			"external_port": 2202,
			"username":      "testuser",
			"password":      "testpass123",
			"description":   "SSH测试容器2",
		},
		{
			"name":          "test-ssh03",
			"host":          "test-ssh03",
			"port":          22,
			"external_port": 2203,
			"username":      "testuser",
			"password":      "testpass123",
			"description":   "SSH测试容器3",
		},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"hosts": testHosts,
		"count": len(testHosts),
	})
}

// RegisterRoutes 注册SaltStack客户端相关路由
func (c *SaltStackClientController) RegisterRoutes(api *gin.RouterGroup) {
	saltstack := api.Group("/saltstack")
	{
		// 客户端安装
		saltstack.POST("/install", c.InstallSaltMinionAsync)
		saltstack.GET("/install", c.ListInstallTasks)
		saltstack.GET("/install/:taskId", c.GetInstallTask)

		// 批量安装
		saltstack.POST("/batch-install", c.BatchInstallMinion)
		saltstack.GET("/batch-install/:taskId", c.GetBatchInstallTask)
		saltstack.GET("/batch-install/:taskId/stream", c.StreamBatchInstallProgress)
		saltstack.GET("/batch-install", c.ListBatchInstallTasks)

		// SSH 测试（含 sudo 权限检查）
		saltstack.POST("/ssh/test", c.TestSSHConnection)
		saltstack.POST("/ssh/test-batch", c.BatchTestSSHConnections)

		// Minion 管理
		saltstack.DELETE("/minion/:minionId", c.DeleteMinion)
		saltstack.POST("/minion/batch-delete", c.BatchDeleteMinions)
		saltstack.POST("/minion/:minionId/uninstall", c.UninstallMinion)

		// 主机模板
		saltstack.GET("/host-templates", c.ListHostTemplates)
		saltstack.POST("/host-templates", c.CreateHostTemplate)
		saltstack.GET("/host-templates/:id", c.GetHostTemplate)
		saltstack.DELETE("/host-templates/:id", c.DeleteHostTemplate)
		saltstack.GET("/host-templates/:id/hosts", c.GetHostTemplateHosts)
		saltstack.GET("/host-templates/download/:format", c.DownloadHostTemplate)
		saltstack.POST("/hosts/parse", c.ParseHostFile)

		// 测试主机
		saltstack.GET("/test-hosts", c.GetTestHosts)
	}
}

// BatchInstallMinion 批量安装 Salt Minion
// @Summary 批量安装Salt Minion
// @Description 批量并发安装Salt Minion到多个节点，支持sudo和root两种账号
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body services.BatchInstallRequest true "批量安装请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saltstack/batch-install [post]
func (c *SaltStackClientController) BatchInstallMinion(ctx *gin.Context) {
	var req services.BatchInstallRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind batch install request")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// 验证请求参数
	if len(req.Hosts) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No hosts specified",
			"message": "At least one host must be specified for installation",
		})
		return
	}

	// 设置默认值
	if req.Parallel <= 0 {
		req.Parallel = 3
	}
	if req.MasterHost == "" {
		req.MasterHost = "salt"
	}
	if req.InstallType == "" {
		req.InstallType = "saltstack"
	}

	logrus.WithFields(logrus.Fields{
		"host_count":   len(req.Hosts),
		"parallel":     req.Parallel,
		"master_host":  req.MasterHost,
		"install_type": req.InstallType,
		"use_sudo":     req.UseSudo,
	}).Info("Starting batch install")

	// 启动批量安装任务
	// TODO: 从JWT中获取真实用户ID
	var userID uint = 1
	taskID, err := c.batchInstallService.BatchInstallSaltMinion(ctx.Request.Context(), req, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to start batch install")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start batch installation",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Batch installation started",
		"task_id":    taskID,
		"host_count": len(req.Hosts),
		"stream_url": fmt.Sprintf("/api/saltstack/batch-install/%s/stream", taskID),
	})
}

// GetBatchInstallTask 获取批量安装任务状态
// @Summary 获取批量安装任务状态
// @Description 获取指定批量安装任务的详细状态信息
// @Tags SaltStack
// @Produce json
// @Param taskId path string true "任务ID"
// @Success 200 {object} services.BatchInstallResult
// @Failure 404 {object} map[string]interface{}
// @Router /api/saltstack/batch-install/{taskId} [get]
func (c *SaltStackClientController) GetBatchInstallTask(ctx *gin.Context) {
	taskID := ctx.Param("taskId")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Task ID required",
		})
		return
	}

	task, err := c.batchInstallService.GetTask(taskID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Task not found",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

// ListBatchInstallTasks 列出批量安装任务
// @Summary 列出批量安装任务
// @Description 获取所有批量安装任务的列表
// @Tags SaltStack
// @Produce json
// @Param limit query int false "限制返回数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/batch-install [get]
func (c *SaltStackClientController) ListBatchInstallTasks(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "50")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	tasks, total, err := c.batchInstallService.ListTasks("saltstack", limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list tasks",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks":  tasks,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// StreamBatchInstallProgress SSE流式输出批量安装进度
// @Summary SSE流式输出批量安装进度
// @Description 通过Server-Sent Events实时推送批量安装进度
// @Tags SaltStack
// @Produce text/event-stream
// @Param taskId path string true "任务ID"
// @Success 200 {string} string "SSE stream"
// @Router /api/saltstack/batch-install/{taskId}/stream [get]
func (c *SaltStackClientController) StreamBatchInstallProgress(ctx *gin.Context) {
	taskID := ctx.Param("taskId")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Task ID required",
		})
		return
	}

	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.Header("X-Accel-Buffering", "no")

	// 获取 SSE 通道
	eventChan := c.batchInstallService.GetSSEChannel(taskID)
	if eventChan == nil {
		// 任务可能已经完成或不存在，尝试获取结果
		task, err := c.batchInstallService.GetTask(taskID)
		if err == nil {
			// 发送最终结果
			data, _ := json.Marshal(services.SSEEvent{
				Type:    "complete",
				Message: "Task completed",
				Data:    task,
			})
			ctx.SSEvent("message", string(data))
			ctx.Writer.Flush()
		} else {
			// 任务不存在
			data, _ := json.Marshal(services.SSEEvent{
				Type:    "error",
				Message: "Task not found",
			})
			ctx.SSEvent("message", string(data))
			ctx.Writer.Flush()
		}
		return
	}

	logrus.WithField("task_id", taskID).Info("SSE stream started for batch install")

	// 发送初始连接成功消息
	initData, _ := json.Marshal(services.SSEEvent{
		Type:    "connected",
		Message: "Connected to batch install progress stream",
	})
	ctx.SSEvent("message", string(initData))
	ctx.Writer.Flush()

	// 监听事件
	clientGone := ctx.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			logrus.WithField("task_id", taskID).Info("SSE client disconnected")
			return
		case event, ok := <-eventChan:
			if !ok {
				// 通道已关闭，发送完成事件
				completeData, _ := json.Marshal(services.SSEEvent{
					Type:    "closed",
					Message: "Stream closed",
				})
				ctx.SSEvent("message", string(completeData))
				ctx.Writer.Flush()
				return
			}

			// 发送事件
			data, err := json.Marshal(event)
			if err != nil {
				logrus.WithError(err).Error("Failed to marshal SSE event")
				continue
			}
			ctx.SSEvent("message", string(data))
			ctx.Writer.Flush()

			// 如果是完成或错误事件，关闭连接
			if event.Type == "complete" || event.Type == "error" {
				return
			}
		}
	}
}

// TestSSHConnection 测试单个 SSH 连接（包含 sudo 权限检查）
// @Summary 测试SSH连接
// @Description 测试SSH连接是否成功，检查sudo权限
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body services.HostInstallConfig true "SSH连接配置"
// @Success 200 {object} services.SSHTestResult
// @Router /api/saltstack/ssh/test [post]
func (c *SaltStackClientController) TestSSHConnection(ctx *gin.Context) {
	var req services.HostInstallConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	if req.Host == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Host is required",
		})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}

	result := c.batchInstallService.TestSSHConnection(ctx.Request.Context(), req)

	ctx.JSON(http.StatusOK, gin.H{
		"success": result.Connected,
		"data":    result,
	})
}

// BatchTestSSHConnections 批量测试 SSH 连接
// @Summary 批量测试SSH连接
// @Description 批量测试多个主机的SSH连接，检查sudo权限
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body services.SSHTestRequest true "批量SSH测试请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/ssh/test-batch [post]
func (c *SaltStackClientController) BatchTestSSHConnections(ctx *gin.Context) {
	var req services.SSHTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	if len(req.Hosts) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least one host is required",
		})
		return
	}

	results, err := c.batchInstallService.BatchTestSSHConnections(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 统计结果
	connectedCount := 0
	sudoCount := 0
	for _, r := range results {
		if r.Connected {
			connectedCount++
		}
		if r.HasSudo {
			sudoCount++
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"results":         results,
			"total":           len(results),
			"connected_count": connectedCount,
			"sudo_count":      sudoCount,
		},
	})
}

// DeleteMinion 删除 Minion（从 Salt Master 中移除密钥）
// @Summary 删除Minion
// @Description 从Salt Master中删除Minion密钥
// @Tags SaltStack
// @Produce json
// @Param minionId path string true "Minion ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/{minionId} [delete]
func (c *SaltStackClientController) DeleteMinion(ctx *gin.Context) {
	minionID := ctx.Param("minionId")
	if minionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minion ID is required",
		})
		return
	}

	saltSvc := services.NewSaltStackService()
	err := saltSvc.DeleteMinion(ctx.Request.Context(), minionID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to delete minion: %v", err),
		})
		return
	}

	logrus.WithField("minion_id", minionID).Info("Minion deleted successfully")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Minion %s deleted successfully", minionID),
	})
}

// BatchDeleteMinionsRequest 批量删除 Minion 请求
type BatchDeleteMinionsRequest struct {
	MinionIDs []string `json:"minion_ids" binding:"required,min=1"`
}

// BatchDeleteMinions 批量删除 Minion
// @Summary 批量删除Minion
// @Description 从Salt Master中批量删除Minion密钥
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body BatchDeleteMinionsRequest true "批量删除请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/batch-delete [post]
func (c *SaltStackClientController) BatchDeleteMinions(ctx *gin.Context) {
	var req BatchDeleteMinionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	saltSvc := services.NewSaltStackService()
	results, _ := saltSvc.DeleteMinionBatch(ctx.Request.Context(), req.MinionIDs)

	successCount := 0
	failedCount := 0
	details := make([]gin.H, 0, len(results))

	for minionID, err := range results {
		if err == nil {
			successCount++
			details = append(details, gin.H{
				"minion_id": minionID,
				"success":   true,
			})
		} else {
			failedCount++
			details = append(details, gin.H{
				"minion_id": minionID,
				"success":   false,
				"error":     err.Error(),
			})
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":       failedCount == 0,
		"message":       fmt.Sprintf("Deleted %d minions, %d failed", successCount, failedCount),
		"success_count": successCount,
		"failed_count":  failedCount,
		"details":       details,
	})
}

// UninstallMinionRequest 卸载 Minion 请求
type UninstallMinionRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password"`
	KeyPath  string `json:"key_path"`
	UseSudo  bool   `json:"use_sudo"`
	SudoPass string `json:"sudo_pass"`
}

// UninstallMinion 卸载 Minion（通过 SSH 远程卸载）
// @Summary 卸载Minion
// @Description 通过SSH连接远程卸载Salt Minion
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param minionId path string true "Minion ID"
// @Param request body UninstallMinionRequest true "SSH连接配置"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/{minionId}/uninstall [post]
func (c *SaltStackClientController) UninstallMinion(ctx *gin.Context) {
	minionID := ctx.Param("minionId")
	if minionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minion ID is required",
		})
		return
	}

	var req UninstallMinionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}

	// 通过 SSH 卸载 Salt Minion
	config := services.HostInstallConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		KeyPath:  req.KeyPath,
		UseSudo:  req.UseSudo,
		SudoPass: req.SudoPass,
	}

	err := c.batchInstallService.UninstallSaltMinion(ctx.Request.Context(), config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to uninstall minion: %v", err),
		})
		return
	}

	// 同时从 Salt Master 删除密钥
	saltSvc := services.NewSaltStackService()
	_ = saltSvc.DeleteMinion(ctx.Request.Context(), minionID)

	logrus.WithFields(logrus.Fields{
		"minion_id": minionID,
		"host":      req.Host,
	}).Info("Minion uninstalled and removed from master")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Minion %s uninstalled and removed from master", minionID),
	})
}

// ========================= 主机模板相关 API =========================

// ListHostTemplates 列出所有主机模板
// @Summary 列出主机模板
// @Description 获取用户的所有主机配置模板
// @Tags SaltStack
// @Produce json
// @Param limit query int false "限制返回数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/host-templates [get]
func (c *SaltStackClientController) ListHostTemplates(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "50")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	db := database.GetDB()
	var templates []models.HostTemplate
	var total int64

	db.Model(&models.HostTemplate{}).Count(&total)
	db.Order("created_at desc").Limit(limit).Offset(offset).Find(&templates)

	// 转换为响应格式（不包含主机详情）
	responses := make([]*models.HostTemplateResponse, 0, len(templates))
	for _, t := range templates {
		resp, err := t.ToResponse(false, true)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"templates": responses,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// CreateHostTemplate 创建主机模板
// @Summary 创建主机模板
// @Description 创建新的主机配置模板（主机数据加密存储）
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body models.HostTemplateCreateRequest true "创建请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/host-templates [post]
func (c *SaltStackClientController) CreateHostTemplate(ctx *gin.Context) {
	var req models.HostTemplateCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	if len(req.Hosts) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least one host is required",
		})
		return
	}

	// 创建模板
	template := &models.HostTemplate{
		Name:        req.Name,
		Description: req.Description,
		Format:      req.Format,
		CreatedBy:   1, // TODO: 从 JWT 获取用户 ID
	}

	// 加密存储主机数据
	if err := template.SetHosts(req.Hosts); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to encrypt host data",
			"message": err.Error(),
		})
		return
	}

	db := database.GetDB()
	if err := db.Create(template).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create template",
			"message": err.Error(),
		})
		return
	}

	resp, _ := template.ToResponse(false, true)

	logrus.WithFields(logrus.Fields{
		"template_id": template.ID,
		"name":        template.Name,
		"host_count":  template.HostCount,
	}).Info("Host template created")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Host template created successfully",
		"data":    resp,
	})
}

// GetHostTemplate 获取主机模板详情
// @Summary 获取主机模板详情
// @Description 获取指定主机模板的详细信息
// @Tags SaltStack
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/host-templates/{id} [get]
func (c *SaltStackClientController) GetHostTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid template ID",
		})
		return
	}

	db := database.GetDB()
	var template models.HostTemplate
	if err := db.First(&template, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Template not found",
		})
		return
	}

	// 返回包含主机列表（密码脱敏）
	resp, err := template.ToResponse(true, true)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get template data",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// GetHostTemplateHosts 获取主机模板的主机列表（包含密码）
// @Summary 获取主机模板主机列表
// @Description 获取指定主机模板的完整主机列表（包含密码，用于批量安装）
// @Tags SaltStack
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/host-templates/{id}/hosts [get]
func (c *SaltStackClientController) GetHostTemplateHosts(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid template ID",
		})
		return
	}

	db := database.GetDB()
	var template models.HostTemplate
	if err := db.First(&template, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Template not found",
		})
		return
	}

	// 获取完整主机列表（包含密码）
	hosts, err := template.GetHosts()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to decrypt host data",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"template_id":   template.ID,
			"template_name": template.Name,
			"hosts":         hosts,
			"count":         len(hosts),
		},
	})
}

// DeleteHostTemplate 删除主机模板
// @Summary 删除主机模板
// @Description 删除指定的主机配置模板
// @Tags SaltStack
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/host-templates/{id} [delete]
func (c *SaltStackClientController) DeleteHostTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid template ID",
		})
		return
	}

	db := database.GetDB()
	result := db.Delete(&models.HostTemplate{}, id)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete template",
			"message": result.Error.Error(),
		})
		return
	}

	if result.RowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Template not found",
		})
		return
	}

	logrus.WithField("template_id", id).Info("Host template deleted")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Template deleted successfully",
	})
}

// DownloadHostTemplate 下载主机模板示例文件
// @Summary 下载主机模板
// @Description 下载指定格式的主机配置模板示例文件
// @Tags SaltStack
// @Produce text/plain
// @Param format path string true "格式" Enums(csv, json, yaml, ini)
// @Success 200 {string} string "模板内容"
// @Router /api/saltstack/host-templates/download/{format} [get]
func (c *SaltStackClientController) DownloadHostTemplate(ctx *gin.Context) {
	format := strings.ToLower(ctx.Param("format"))

	parser := services.NewHostParserService()
	var content, filename, contentType string

	switch format {
	case "csv":
		content = parser.GenerateCSVTemplate()
		filename = "hosts_template.csv"
		contentType = "text/csv"
	case "json":
		content = parser.GenerateJSONTemplate()
		filename = "hosts_template.json"
		contentType = "application/json"
	case "yaml", "yml":
		content = parser.GenerateYAMLTemplate()
		filename = "hosts_template.yaml"
		contentType = "application/x-yaml"
	case "ini":
		content = parser.GenerateAnsibleINITemplate()
		filename = "hosts_template.ini"
		contentType = "text/plain"
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid format. Supported: csv, json, yaml, ini",
		})
		return
	}

	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Header("Content-Type", contentType)
	ctx.String(http.StatusOK, content)
}

// ParseHostFileRequest 解析主机文件请求
type ParseHostFileRequest struct {
	Content  string `json:"content" binding:"required"`
	Format   string `json:"format"`   // 可选，如果提供则使用指定格式
	Filename string `json:"filename"` // 文件名，用于自动检测格式
}

// ParseHostFile 解析上传的主机文件
// @Summary 解析主机文件
// @Description 解析上传的主机配置文件（CSV/JSON/YAML/INI格式）
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body ParseHostFileRequest true "文件内容"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/hosts/parse [post]
func (c *SaltStackClientController) ParseHostFile(ctx *gin.Context) {
	var req ParseHostFileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	parser := services.NewHostParserService()
	var hosts []services.HostConfig
	var err error

	// 确定解析格式
	format := strings.ToLower(req.Format)
	if format == "" && req.Filename != "" {
		// 从文件名自动检测格式
		if strings.HasSuffix(strings.ToLower(req.Filename), ".csv") {
			format = "csv"
		} else if strings.HasSuffix(strings.ToLower(req.Filename), ".json") {
			format = "json"
		} else if strings.HasSuffix(strings.ToLower(req.Filename), ".yaml") || strings.HasSuffix(strings.ToLower(req.Filename), ".yml") {
			format = "yaml"
		} else if strings.HasSuffix(strings.ToLower(req.Filename), ".ini") {
			format = "ini"
		}
	}

	// 根据格式解析
	contentBytes := []byte(req.Content)
	switch format {
	case "csv":
		hosts, err = parser.ParseCSV(contentBytes)
	case "json":
		hosts, err = parser.ParseJSON(contentBytes)
	case "yaml", "yml":
		hosts, err = parser.ParseYAML(contentBytes)
	case "ini":
		hosts, err = parser.ParseAnsibleINI(contentBytes)
	default:
		// 尝试自动检测格式
		hosts, err = parser.AutoDetectAndParse(contentBytes)
	}

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to parse file",
			"message": err.Error(),
		})
		return
	}

	// 转换为前端需要的格式
	hostList := make([]models.HostTemplateHost, 0, len(hosts))
	for _, h := range hosts {
		hostList = append(hostList, models.HostTemplateHost{
			Host:     h.Host,
			Port:     h.Port,
			Username: h.Username,
			Password: h.Password,
			UseSudo:  h.UseSudo,
			MinionID: h.MinionID,
			Group:    h.Group,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"hosts":  hostList,
			"count":  len(hostList),
			"format": format,
		},
	})
}
