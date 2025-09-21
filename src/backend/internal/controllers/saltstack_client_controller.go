package controllers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SaltStackClientController 处理SaltStack客户端安装相关的HTTP请求
type SaltStackClientController struct {
	service *services.SaltStackClientService
}

func NewSaltStackClientController() *SaltStackClientController {
	return &SaltStackClientController{
		service: services.NewSaltStackClientService(),
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

		// 测试主机
		saltstack.GET("/test-hosts", c.GetTestHosts)
	}
}
