package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
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
	// 应用认证中间件 - 所有 saltstack 客户端 API 都需要认证
	saltstack.Use(middleware.AuthMiddlewareWithSession())
	{
		// 客户端安装
		saltstack.POST("/install", c.InstallSaltMinionAsync)
		saltstack.GET("/install", c.ListInstallTasks)
		saltstack.GET("/install/:taskId", c.GetInstallTask)

		// 批量安装
		saltstack.POST("/batch-install", c.BatchInstallMinion)
		saltstack.GET("/batch-install/calculate-parallel", c.CalculateParallel) // 计算动态并行度
		saltstack.GET("/batch-install/:taskId", c.GetBatchInstallTask)
		saltstack.GET("/batch-install/:taskId/stream", c.StreamBatchInstallProgress)
		saltstack.GET("/batch-install/:taskId/logs", c.GetBatchInstallTaskLogs)
		saltstack.GET("/batch-install/:taskId/ssh-logs", c.GetBatchInstallSSHLogs)
		saltstack.GET("/batch-install", c.ListBatchInstallTasks)

		// SSH 测试（含 sudo 权限检查）
		saltstack.POST("/ssh/test", c.TestSSHConnection)
		saltstack.POST("/ssh/test-batch", c.BatchTestSSHConnections)

		// Minion 管理
		saltstack.DELETE("/minion/:minionId", c.DeleteMinion)
		saltstack.POST("/minion/batch-delete", c.BatchDeleteMinions)
		saltstack.POST("/minion/:minionId/uninstall", c.UninstallMinion)
		// 删除任务管理
		saltstack.GET("/minion/delete-tasks", c.ListDeleteTasks)
		saltstack.GET("/minion/delete-tasks/:minionId", c.GetDeleteTaskStatus)
		saltstack.POST("/minion/delete-tasks/:minionId/cancel", c.CancelDeleteTask)
		saltstack.POST("/minion/delete-tasks/:minionId/retry", c.RetryDeleteTask)
		saltstack.GET("/minion/pending-deletes", c.GetPendingDeleteMinions)

		// 主机模板
		saltstack.GET("/host-templates", c.ListHostTemplates)
		saltstack.POST("/host-templates", c.CreateHostTemplate)
		saltstack.GET("/host-templates/:id", c.GetHostTemplate)
		saltstack.DELETE("/host-templates/:id", c.DeleteHostTemplate)
		saltstack.GET("/host-templates/:id/hosts", c.GetHostTemplateHosts)
		saltstack.GET("/host-templates/download/:format", c.DownloadHostTemplate)
		saltstack.POST("/hosts/parse", c.ParseHostFile)
		saltstack.POST("/hosts/parse/debug", c.ParseHostFileDebug) // 调试接口

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

	// 设置默认值（并行度会在 BatchInstallSaltMinion 中自动计算）
	// req.Parallel 为 0 时会自动使用动态并行度
	if req.MasterHost == "" {
		req.MasterHost = "salt"
	}
	if req.InstallType == "" {
		req.InstallType = "saltstack"
	}

	// 获取并行度配置信息（用于日志和返回）
	parallelInfo := services.GetParallelInfo(len(req.Hosts), req.Parallel, 100)

	logrus.WithFields(logrus.Fields{
		"host_count":        len(req.Hosts),
		"parallel":          parallelInfo.Parallel,
		"percentage":        fmt.Sprintf("%.1f%%", parallelInfo.Percentage),
		"is_auto_calculate": parallelInfo.IsAutoCalculate,
		"master_host":       req.MasterHost,
		"install_type":      req.InstallType,
		"use_sudo":          req.UseSudo,
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
		"success":          true,
		"message":          "Batch installation started",
		"task_id":          taskID,
		"host_count":       len(req.Hosts),
		"parallel":         parallelInfo.Parallel,
		"percentage":       parallelInfo.Percentage,
		"is_auto_parallel": parallelInfo.IsAutoCalculate,
		"stream_url":       fmt.Sprintf("/api/saltstack/batch-install/%s/stream", taskID),
	})
}

// CalculateParallel 计算动态并行度
// @Summary 计算动态并行度
// @Description 根据节点数量计算推荐的并行度
// @Tags SaltStack
// @Produce json
// @Param host_count query int true "节点数量"
// @Param max_parallel query int false "最大并行度（默认100）"
// @Success 200 {object} services.ParallelConfig
// @Router /api/saltstack/batch-install/calculate-parallel [get]
func (c *SaltStackClientController) CalculateParallel(ctx *gin.Context) {
	hostCountStr := ctx.Query("host_count")
	if hostCountStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "host_count parameter is required",
		})
		return
	}

	hostCount, err := strconv.Atoi(hostCountStr)
	if err != nil || hostCount < 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid host_count parameter",
		})
		return
	}

	maxParallel := 100
	if maxStr := ctx.Query("max_parallel"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			maxParallel = max
		}
	}

	parallelInfo := services.GetParallelInfo(hostCount, 0, maxParallel)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    parallelInfo,
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

// GetBatchInstallTaskLogs 获取批量安装任务日志
// @Summary 获取批量安装任务日志
// @Description 获取指定批量安装任务的详细日志记录
// @Tags SaltStack
// @Produce json
// @Param taskId path string true "任务ID"
// @Param host query string false "按主机过滤"
// @Param level query string false "按日志级别过滤 (info, warn, error, debug)"
// @Param category query string false "按分类过滤 (ssh, install, config, key, output)"
// @Param limit query int false "限制返回数量" default(100)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/saltstack/batch-install/{taskId}/logs [get]
func (c *SaltStackClientController) GetBatchInstallTaskLogs(ctx *gin.Context) {
	taskID := ctx.Param("taskId")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Task ID required",
		})
		return
	}

	// 查询参数
	host := ctx.Query("host")
	level := ctx.Query("level")
	category := ctx.Query("category")
	limitStr := ctx.DefaultQuery("limit", "100")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	logs, total, err := c.batchInstallService.GetTaskLogsFiltered(taskID, host, level, category, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get task logs",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":   logs,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetBatchInstallSSHLogs 获取批量安装任务的 SSH 执行日志
// @Summary 获取SSH执行日志
// @Description 获取指定批量安装任务的SSH命令执行详细日志
// @Tags SaltStack
// @Produce json
// @Param taskId path string true "任务ID"
// @Param host query string false "按主机过滤"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/saltstack/batch-install/{taskId}/ssh-logs [get]
func (c *SaltStackClientController) GetBatchInstallSSHLogs(ctx *gin.Context) {
	taskID := ctx.Param("taskId")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Task ID required",
		})
		return
	}

	host := ctx.Query("host")

	logs, err := c.batchInstallService.GetSSHLogs(taskID, host)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get SSH logs",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
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

// DeleteMinion 删除 Minion（软删除 + 后台异步真实删除）
// @Summary 删除Minion
// @Description 软删除Minion并在后台异步执行真实删除，立即返回结果提升用户体验
// @Tags SaltStack
// @Produce json
// @Param minionId path string true "Minion ID"
// @Param force query bool false "强制删除（包括在线节点）"
// @Param sync query bool false "同步删除（等待真实删除完成，默认false使用软删除）"
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

	// 检查是否强制删除
	forceDelete := ctx.Query("force") == "true"
	// 检查是否同步删除（兼容旧逻辑）
	syncDelete := ctx.Query("sync") == "true"

	// 获取当前用户
	createdBy := ""
	if user, exists := ctx.Get("username"); exists {
		createdBy = user.(string)
	}

	// 如果是同步删除，使用旧逻辑
	if syncDelete {
		saltSvc := services.NewSaltStackService()
		err := saltSvc.DeleteMinionWithForce(ctx.Request.Context(), minionID, forceDelete)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to delete minion: %v", err),
			})
			return
		}
		logrus.WithFields(logrus.Fields{
			"minion_id": minionID,
			"force":     forceDelete,
			"sync":      true,
		}).Info("Minion deleted successfully (sync)")
		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("Minion %s deleted successfully", minionID),
			"force":   forceDelete,
			"sync":    true,
		})
		return
	}

	// 默认使用软删除（立即返回，后台异步执行真实删除）
	deleteSvc := services.GetMinionDeleteService()
	task, err := deleteSvc.SoftDelete(ctx.Request.Context(), minionID, forceDelete, createdBy)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to create delete task: %v", err),
		})
		return
	}

	logrus.WithFields(logrus.Fields{
		"minion_id": minionID,
		"task_id":   task.ID,
		"force":     forceDelete,
	}).Info("Minion soft-delete task created")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Minion %s 已标记删除，后台正在执行真实删除", minionID),
		"task_id": task.ID,
		"status":  task.Status,
		"force":   forceDelete,
		"async":   true,
	})
}

// BatchDeleteMinionsRequest 批量删除 Minion 请求
type BatchDeleteMinionsRequest struct {
	MinionIDs   []string          `json:"minion_ids" binding:"required,min=1"`
	Force       bool              `json:"force"`        // 强制删除（包括在线节点）
	Uninstall   bool              `json:"uninstall"`    // 是否通过 SSH 卸载远程 salt-minion
	SSHPort     int               `json:"ssh_port"`     // SSH 端口（默认 22）
	SSHUsername string            `json:"ssh_username"` // SSH 用户名
	SSHPassword string            `json:"ssh_password"` // SSH 密码
	SSHKeyPath  string            `json:"ssh_key_path"` // SSH 私钥路径
	UseSudo     bool              `json:"use_sudo"`     // 是否使用 sudo
	HostMapping map[string]string `json:"host_mapping"` // minion_id -> ssh_host 映射（可选）
}

// BatchDeleteMinions 批量删除 Minion（软删除 + 后台异步真实删除）
// @Summary 批量删除Minion
// @Description 批量软删除Minion并在后台异步执行真实删除，支持通过SSH远程卸载salt-minion
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

	// 获取当前用户
	createdBy := ""
	if user, exists := ctx.Get("username"); exists {
		createdBy = user.(string)
	}

	// 设置默认 SSH 端口
	sshPort := req.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	// 使用软删除服务
	deleteSvc := services.GetMinionDeleteService()
	tasks := make([]*models.MinionDeleteTask, 0, len(req.MinionIDs))

	for _, minionID := range req.MinionIDs {
		// 确定 SSH 主机地址
		sshHost := minionID // 默认使用 minionID 作为主机地址
		if req.HostMapping != nil {
			if host, ok := req.HostMapping[minionID]; ok && host != "" {
				sshHost = host
			}
		}

		opts := services.SoftDeleteOptions{
			Force:       req.Force,
			Uninstall:   req.Uninstall,
			SSHHost:     sshHost,
			SSHPort:     sshPort,
			SSHUsername: req.SSHUsername,
			SSHPassword: req.SSHPassword,
			SSHKeyPath:  req.SSHKeyPath,
			UseSudo:     req.UseSudo,
		}

		task, err := deleteSvc.SoftDeleteWithOptions(ctx.Request.Context(), minionID, opts, createdBy)
		if err != nil {
			logrus.WithError(err).WithField("minion_id", minionID).Warn("创建批量删除任务失败")
			continue
		}
		tasks = append(tasks, task)
	}

	details := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		details = append(details, gin.H{
			"minion_id": task.MinionID,
			"task_id":   task.ID,
			"status":    task.Status,
			"success":   true,
		})
	}

	logrus.WithFields(logrus.Fields{
		"count": len(tasks),
		"force": req.Force,
	}).Info("Batch soft-delete tasks created")

	ctx.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       fmt.Sprintf("%d 个 Minion 已标记删除，后台正在执行真实删除", len(tasks)),
		"success_count": len(tasks),
		"failed_count":  len(req.MinionIDs) - len(tasks),
		"details":       details,
		"async":         true,
	})
}

// ListDeleteTasks 列出删除任务
// @Summary 列出删除任务
// @Description 获取 Minion 删除任务列表
// @Tags SaltStack
// @Produce json
// @Param status query string false "任务状态过滤 (pending, deleting, completed, failed, cancelled)"
// @Param limit query int false "返回数量限制"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/delete-tasks [get]
func (c *SaltStackClientController) ListDeleteTasks(ctx *gin.Context) {
	status := ctx.Query("status")
	limitStr := ctx.Query("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	deleteSvc := services.GetMinionDeleteService()
	tasks, err := deleteSvc.ListDeleteTasks(status, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to list delete tasks: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tasks,
		"count":   len(tasks),
	})
}

// GetDeleteTaskStatus 获取删除任务状态
// @Summary 获取删除任务状态
// @Description 根据 Minion ID 获取最新的删除任务状态
// @Tags SaltStack
// @Produce json
// @Param minionId path string true "Minion ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/delete-tasks/{minionId} [get]
func (c *SaltStackClientController) GetDeleteTaskStatus(ctx *gin.Context) {
	minionID := ctx.Param("minionId")
	if minionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minion ID is required",
		})
		return
	}

	deleteSvc := services.GetMinionDeleteService()
	task, err := deleteSvc.GetDeleteTaskByMinionID(minionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   fmt.Sprintf("No delete task found for minion %s", minionID),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

// CancelDeleteTask 取消删除任务
// @Summary 取消删除任务
// @Description 取消待处理或失败的删除任务
// @Tags SaltStack
// @Produce json
// @Param minionId path string true "Minion ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/delete-tasks/{minionId}/cancel [post]
func (c *SaltStackClientController) CancelDeleteTask(ctx *gin.Context) {
	minionID := ctx.Param("minionId")
	if minionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minion ID is required",
		})
		return
	}

	deleteSvc := services.GetMinionDeleteService()
	if err := deleteSvc.CancelDelete(minionID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Delete task for minion %s cancelled", minionID),
	})
}

// RetryDeleteTask 重试删除任务
// @Summary 重试删除任务
// @Description 重试失败的删除任务
// @Tags SaltStack
// @Produce json
// @Param minionId path string true "Minion ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/delete-tasks/{minionId}/retry [post]
func (c *SaltStackClientController) RetryDeleteTask(ctx *gin.Context) {
	minionID := ctx.Param("minionId")
	if minionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minion ID is required",
		})
		return
	}

	deleteSvc := services.GetMinionDeleteService()
	if err := deleteSvc.RetryDelete(minionID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Delete task for minion %s retried", minionID),
	})
}

// GetPendingDeleteMinions 获取待删除的 Minion 列表
// @Summary 获取待删除的 Minion 列表
// @Description 获取所有状态为 pending 或 deleting 的 Minion ID
// @Tags SaltStack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minion/pending-deletes [get]
func (c *SaltStackClientController) GetPendingDeleteMinions(ctx *gin.Context) {
	deleteSvc := services.GetMinionDeleteService()
	minionIDs, err := deleteSvc.GetPendingDeleteMinionIDs()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get pending deletes: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":    true,
		"minion_ids": minionIDs,
		"count":      len(minionIDs),
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

	// 使用安全解析方法（包含文件类型、大小、恶意内容检查）
	contentBytes := []byte(req.Content)
	hosts, err := parser.ValidateAndParse(contentBytes, format)
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
			Host:            h.Host,
			Port:            h.Port,
			Username:        h.Username,
			Password:        h.Password,
			UseSudo:         h.UseSudo,
			MinionID:        h.MinionID,
			Group:           h.Group,
			InstallCategraf: h.InstallCategraf,
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

// ParseHostFileDebugRequest 调试解析主机文件请求
type ParseHostFileDebugRequest struct {
	Content  string `json:"content" binding:"required"`
	Format   string `json:"format"`   // 可选，如果提供则使用指定格式
	Filename string `json:"filename"` // 文件名，用于自动检测格式
}

// ParseHostFileDebug 调试解析上传的主机文件（返回详细解析过程）
// @Summary 调试解析主机文件
// @Description 调试解析上传的主机配置文件，返回详细的解析过程和验证结果
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body ParseHostFileDebugRequest true "文件内容"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/hosts/parse/debug [post]
func (c *SaltStackClientController) ParseHostFileDebug(ctx *gin.Context) {
	var req ParseHostFileDebugRequest
	debugInfo := make(map[string]interface{})
	debugInfo["timestamp"] = time.Now().Format(time.RFC3339)
	debugInfo["steps"] = []map[string]interface{}{}

	addStep := func(name string, status string, details interface{}) {
		steps := debugInfo["steps"].([]map[string]interface{})
		debugInfo["steps"] = append(steps, map[string]interface{}{
			"step":    len(steps) + 1,
			"name":    name,
			"status":  status,
			"details": details,
		})
	}

	// 1. 解析请求
	if err := ctx.ShouldBindJSON(&req); err != nil {
		addStep("解析请求体", "failed", map[string]interface{}{
			"error": err.Error(),
		})
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
			"debug":   debugInfo,
		})
		return
	}

	addStep("解析请求体", "success", map[string]interface{}{
		"filename":       req.Filename,
		"format":         req.Format,
		"content_length": len(req.Content),
		"content_preview": func() string {
			if len(req.Content) > 500 {
				return req.Content[:500] + "..."
			}
			return req.Content
		}(),
	})

	parser := services.NewHostParserService()
	contentBytes := []byte(req.Content)

	// 2. 检测文件格式
	format := strings.ToLower(req.Format)
	if format == "" && req.Filename != "" {
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
	addStep("检测文件格式", "success", map[string]interface{}{
		"detected_format": format,
		"from_filename":   req.Filename,
		"from_param":      req.Format,
	})

	// 3. 验证文件编码
	if err := parser.ValidateEncoding(contentBytes); err != nil {
		addStep("验证文件编码", "failed", map[string]interface{}{
			"error": err.Error(),
		})
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Encoding validation failed",
			"message": err.Error(),
			"debug":   debugInfo,
		})
		return
	}
	addStep("验证文件编码", "success", map[string]interface{}{
		"encoding": "UTF-8",
	})

	// 4. 验证文件大小
	if err := parser.ValidateFileSize(contentBytes); err != nil {
		addStep("验证文件大小", "failed", map[string]interface{}{
			"error":     err.Error(),
			"file_size": len(contentBytes),
			"max_size":  1024 * 1024,
		})
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "File size validation failed",
			"message": err.Error(),
			"debug":   debugInfo,
		})
		return
	}
	addStep("验证文件大小", "success", map[string]interface{}{
		"file_size":    len(contentBytes),
		"max_size":     1024 * 1024,
		"size_percent": fmt.Sprintf("%.2f%%", float64(len(contentBytes))/float64(1024*1024)*100),
	})

	// 5. 验证行数
	lineCount := strings.Count(req.Content, "\n") + 1
	if err := parser.ValidateLineCount(contentBytes); err != nil {
		addStep("验证行数", "failed", map[string]interface{}{
			"error":      err.Error(),
			"line_count": lineCount,
			"max_lines":  10000,
		})
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Line count validation failed",
			"message": err.Error(),
			"debug":   debugInfo,
		})
		return
	}
	addStep("验证行数", "success", map[string]interface{}{
		"line_count": lineCount,
		"max_lines":  10000,
	})

	// 6. 检测危险内容
	if err := parser.DetectDangerousContent(contentBytes); err != nil {
		addStep("安全检查", "failed", map[string]interface{}{
			"error": err.Error(),
		})
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Security check failed",
			"message": err.Error(),
			"debug":   debugInfo,
		})
		return
	}
	addStep("安全检查", "success", map[string]interface{}{
		"dangerous_patterns_checked": true,
	})

	// 7. 解析文件内容
	hosts, err := parser.ParseHosts(contentBytes, format)
	if err != nil {
		addStep("解析文件内容", "failed", map[string]interface{}{
			"error":  err.Error(),
			"format": format,
		})
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to parse file",
			"message": err.Error(),
			"debug":   debugInfo,
		})
		return
	}

	// 解析每个主机的详细信息
	hostsDebug := make([]map[string]interface{}, 0, len(hosts))
	for i, h := range hosts {
		hostDebug := map[string]interface{}{
			"index":     i + 1,
			"host":      h.Host,
			"port":      h.Port,
			"username":  h.Username,
			"password":  strings.Repeat("*", len(h.Password)), // 脱敏
			"use_sudo":  h.UseSudo,
			"minion_id": h.MinionID,
			"group":     h.Group,
		}

		// 验证每个主机配置
		if err := parser.ValidateHostConfig(&h); err != nil {
			hostDebug["validation"] = "failed"
			hostDebug["validation_error"] = err.Error()
		} else {
			hostDebug["validation"] = "passed"
		}

		hostsDebug = append(hostsDebug, hostDebug)
	}

	addStep("解析文件内容", "success", map[string]interface{}{
		"format":      format,
		"hosts_count": len(hosts),
		"hosts":       hostsDebug,
	})

	// 8. 验证主机配置安全性
	validHosts := 0
	invalidHosts := 0
	validationErrors := []string{}
	for i, host := range hosts {
		if err := parser.ValidateHostConfig(&host); err != nil {
			invalidHosts++
			validationErrors = append(validationErrors, fmt.Sprintf("主机 %d (%s): %v", i+1, host.Host, err))
		} else {
			validHosts++
		}
	}

	addStep("验证主机配置安全性", func() string {
		if invalidHosts > 0 {
			return "warning"
		}
		return "success"
	}(), map[string]interface{}{
		"valid_hosts":       validHosts,
		"invalid_hosts":     invalidHosts,
		"validation_errors": validationErrors,
	})

	// 转换为前端需要的格式
	hostList := make([]models.HostTemplateHost, 0, len(hosts))
	for _, h := range hosts {
		hostList = append(hostList, models.HostTemplateHost{
			Host:            h.Host,
			Port:            h.Port,
			Username:        h.Username,
			Password:        h.Password,
			UseSudo:         h.UseSudo,
			MinionID:        h.MinionID,
			Group:           h.Group,
			InstallCategraf: h.InstallCategraf,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"hosts":  hostList,
			"count":  len(hostList),
			"format": format,
		},
		"debug": debugInfo,
	})
}
