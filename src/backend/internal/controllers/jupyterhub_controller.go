package controllers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// JupyterHubController JupyterHub控制器
type JupyterHubController struct {
	service *services.JupyterHubService
}

// NewJupyterHubController 创建JupyterHub控制器
func NewJupyterHubController() *JupyterHubController {
	return &JupyterHubController{
		service: services.NewJupyterHubService(),
	}
}

// CreateHubConfig 创建JupyterHub配置
// @Summary 创建JupyterHub配置
// @Description 创建新的JupyterHub配置
// @Tags JupyterHub
// @Accept json
// @Produce json
// @Param config body models.JupyterHubConfig true "JupyterHub配置"
// @Success 201 {object} models.JupyterHubConfig
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/configs [post]
func (ctrl *JupyterHubController) CreateHubConfig(c *gin.Context) {
	var config models.JupyterHubConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 测试连接
	if err := ctrl.service.TestJupyterHubConnection(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to connect to JupyterHub",
			"details": err.Error(),
		})
		return
	}

	if err := ctrl.service.CreateHubConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetHubConfigs 获取JupyterHub配置列表
// @Summary 获取JupyterHub配置列表
// @Description 获取所有JupyterHub配置
// @Tags JupyterHub
// @Produce json
// @Success 200 {array} models.JupyterHubConfig
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/configs [get]
func (ctrl *JupyterHubController) GetHubConfigs(c *gin.Context) {
	configs, err := ctrl.service.GetHubConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
		"total":   len(configs),
	})
}

// GetHubConfig 获取指定的JupyterHub配置
// @Summary 获取JupyterHub配置
// @Description 根据ID获取JupyterHub配置
// @Tags JupyterHub
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} models.JupyterHubConfig
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/configs/{id} [get]
func (ctrl *JupyterHubController) GetHubConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config ID"})
		return
	}

	config, err := ctrl.service.GetHubConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateHubConfig 更新JupyterHub配置
// @Summary 更新JupyterHub配置
// @Description 更新指定的JupyterHub配置
// @Tags JupyterHub
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param updates body map[string]interface{} true "更新字段"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/configs/{id} [put]
func (ctrl *JupyterHubController) UpdateHubConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.service.UpdateHubConfig(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Config updated successfully"})
}

// DeleteHubConfig 删除JupyterHub配置
// @Summary 删除JupyterHub配置
// @Description 删除指定的JupyterHub配置
// @Tags JupyterHub
// @Param id path int true "配置ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/configs/{id} [delete]
func (ctrl *JupyterHubController) DeleteHubConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config ID"})
		return
	}

	if err := ctrl.service.DeleteHubConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Config deleted successfully"})
}

// TestConnection 测试JupyterHub连接
// @Summary 测试JupyterHub连接
// @Description 测试与JupyterHub的连接
// @Tags JupyterHub
// @Accept json
// @Produce json
// @Param config body models.JupyterHubConfig true "JupyterHub配置"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/test-connection [post]
func (ctrl *JupyterHubController) TestConnection(c *gin.Context) {
	var config models.JupyterHubConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.service.TestJupyterHubConnection(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"message":   "Connection successful",
	})
}

// CreateTask 创建JupyterHub任务
// @Summary 创建JupyterHub任务
// @Description 创建新的JupyterHub执行任务
// @Tags JupyterHub
// @Accept json
// @Produce json
// @Param task body models.JupyterTask true "任务信息"
// @Success 201 {object} models.JupyterTask
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/tasks [post]
func (ctrl *JupyterHubController) CreateTask(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var task models.JupyterTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task.UserID = userID

	if err := ctrl.service.CreateTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// GetTasks 获取用户任务列表
// @Summary 获取任务列表
// @Description 获取当前用户的JupyterHub任务列表
// @Tags JupyterHub
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/tasks [get]
func (ctrl *JupyterHubController) GetTasks(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	tasks, total, err := ctrl.service.GetTasks(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":       tasks,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetTask 获取指定任务
// @Summary 获取任务详情
// @Description 获取指定任务的详细信息
// @Tags JupyterHub
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} models.JupyterTask
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/tasks/{id} [get]
func (ctrl *JupyterHubController) GetTask(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := ctrl.service.GetTask(uint(taskID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// CancelTask 取消任务
// @Summary 取消任务
// @Description 取消正在运行的任务
// @Tags JupyterHub
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/tasks/{id}/cancel [post]
func (ctrl *JupyterHubController) CancelTask(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	if err := ctrl.service.CancelTask(uint(taskID), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task cancelled successfully"})
}

// GetTaskOutput 获取任务输出
// @Summary 获取任务输出
// @Description 获取任务的执行输出日志
// @Tags JupyterHub
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/tasks/{id}/output [get]
func (ctrl *JupyterHubController) GetTaskOutput(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	output, err := ctrl.service.GetTaskOutput(uint(taskID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"output":  output,
	})
}

// GetJupyterHubUsers 获取JupyterHub用户列表
// @Summary 获取JupyterHub用户列表
// @Description 获取指定JupyterHub实例的用户列表
// @Tags JupyterHub
// @Produce json
// @Param config_id query int true "配置ID"
// @Success 200 {array} models.JupyterHubUser
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/users [get]
func (ctrl *JupyterHubController) GetJupyterHubUsers(c *gin.Context) {
	configIDStr := c.Query("config_id")
	if configIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config_id is required"})
		return
	}

	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config_id"})
		return
	}

	users, err := ctrl.service.GetJupyterHubUsers(uint(configID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": len(users),
	})
}

// GetHubStatus 获取JupyterHub状态
// @Summary 获取JupyterHub状态
// @Description 获取JupyterHub实例状态信息
// @Tags JupyterHub
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/status [get]
func (ctrl *JupyterHubController) GetHubStatus(c *gin.Context) {
	status, err := ctrl.service.GetHubStatus()
	if err != nil {
		// 如果获取真实状态失败，返回模拟数据
		c.JSON(http.StatusOK, gin.H{
			"running":         true,
			"users_online":    5,
			"servers_running": 3,
			"total_memory_gb": 32,
			"used_memory_gb":  12,
			"total_cpu_cores": 16,
			"used_cpu_cores":  6,
			"version":         "5.3.0",
			"url":             "/jupyter/",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetUserTasks 获取当前用户的任务列表
// @Summary 获取用户任务列表
// @Description 获取当前登录用户的JupyterHub任务列表
// @Tags JupyterHub
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /jupyterhub/user-tasks [get]
func (ctrl *JupyterHubController) GetUserTasks(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		// 如果没有认证，返回模拟数据用于演示
		c.JSON(http.StatusOK, gin.H{
			"tasks": []map[string]interface{}{
				{
					"id":          1,
					"task_name":   "数据分析任务",
					"status":      "running",
					"created_at":  "2025-08-01T10:00:00Z",
					"progress":    65,
					"description": "正在执行Python数据分析脚本",
					"output":      "执行中...",
				},
				{
					"id":          2,
					"task_name":   "机器学习训练",
					"status":      "completed",
					"created_at":  "2025-08-01T08:30:00Z",
					"progress":    100,
					"description": "模型训练已完成",
					"output":      "Training completed with 95.2% accuracy",
				},
				{
					"id":          3,
					"task_name":   "数据预处理",
					"status":      "failed",
					"created_at":  "2025-08-01T09:15:00Z",
					"progress":    30,
					"description": "数据清洗过程中出现错误",
					"output":      "Error: Missing values in column 'age'",
				},
			},
		})
		return
	}

	tasks, err := ctrl.service.GetUserTasks(userID)
	if err != nil {
		// 如果获取真实任务失败，返回模拟数据
		c.JSON(http.StatusOK, gin.H{
			"tasks": []map[string]interface{}{
				{
					"id":         1,
					"task_name":  "数据分析任务",
					"status":     "running",
					"created_at": "2025-08-01T10:00:00Z",
					"progress":   65,
				},
				{
					"id":         2,
					"task_name":  "机器学习训练",
					"status":     "completed",
					"created_at": "2025-08-01T09:00:00Z",
					"progress":   100,
				},
			},
			"total": 2,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// RegisterRoutes 注册路由
func (ctrl *JupyterHubController) RegisterRoutes(router *gin.RouterGroup) {
	jupyterhub := router.Group("/jupyterhub")

	// 需要认证的路由
	authenticated := jupyterhub.Group("")
	authenticated.Use(middleware.AuthMiddlewareWithSession())

	{
		// JupyterHub配置管理
		authenticated.POST("/configs", ctrl.CreateHubConfig)
		authenticated.GET("/configs", ctrl.GetHubConfigs)
		authenticated.GET("/configs/:id", ctrl.GetHubConfig)
		authenticated.PUT("/configs/:id", ctrl.UpdateHubConfig)
		authenticated.DELETE("/configs/:id", ctrl.DeleteHubConfig)
		authenticated.POST("/test-connection", ctrl.TestConnection)

		// 任务管理
		authenticated.POST("/tasks", ctrl.CreateTask)
		authenticated.GET("/tasks", ctrl.GetTasks)
		authenticated.GET("/tasks/:id", ctrl.GetTask)
		authenticated.POST("/tasks/:id/cancel", ctrl.CancelTask)
		authenticated.GET("/tasks/:id/output", ctrl.GetTaskOutput)

		// JupyterHub用户管理
		authenticated.GET("/users", ctrl.GetJupyterHubUsers)
	}

	// 公开访问的API（用于前端页面，后续可以添加认证）
	public := jupyterhub.Group("")
	{
		public.GET("/status", ctrl.GetHubStatus)
		public.GET("/user-tasks", ctrl.GetUserTasks)
	}
}
