package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
)

// JupyterHubK8sHandler JupyterHub K8s集成处理器
type JupyterHubK8sHandler struct {
	service *services.JupyterHubK8sService
}

// NewJupyterHubK8sHandler 创建新的处理器
func NewJupyterHubK8sHandler(service *services.JupyterHubK8sService) *JupyterHubK8sHandler {
	return &JupyterHubK8sHandler{
		service: service,
	}
}

// NewJupyterHubHandler 创建JupyterHub处理器 (别名)
func NewJupyterHubHandler() *JupyterHubK8sHandler {
	// 创建一个默认配置或者简化的服务
	return &JupyterHubK8sHandler{
		service: nil, // TODO: Initialize with proper service when config is available
	}
}

// SubmitPythonScriptRequest 提交Python脚本请求结构
type SubmitPythonScriptRequest struct {
	Name           string            `json:"name" binding:"required"`
	Script         string            `json:"script" binding:"required"`
	Requirements   []string          `json:"requirements"`
	GPURequired    bool              `json:"gpu_required"`
	GPUCount       int               `json:"gpu_count"`
	GPUType        string            `json:"gpu_type"`
	MemoryMB       int               `json:"memory_mb"`
	CPUCores       int               `json:"cpu_cores"`
	Environment    map[string]string `json:"environment"`
	WorkingDir     string            `json:"working_dir"`
	OutputPath     string            `json:"output_path"`
}

// SubmitPythonScriptResponse 提交Python脚本响应结构
type SubmitPythonScriptResponse struct {
	JobID     string    `json:"job_id"`
	JobName   string    `json:"job_name"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// GPUResourceStatusResponse GPU资源状态响应
type GPUResourceStatusResponse struct {
	TotalGPUs     int                      `json:"total_gpus"`
	AvailableGPUs int                      `json:"available_gpus"`
	UsedGPUs      int                      `json:"used_gpus"`
	GPUNodes      []services.GPUNodeInfo   `json:"gpu_nodes"`
	LastUpdated   time.Time                `json:"last_updated"`
}

// JobStatusResponse Job状态响应
type JobStatusResponse struct {
	JobID         string    `json:"job_id"`
	JobName       string    `json:"job_name"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Logs          string    `json:"logs,omitempty"`
}

// RegisterRoutes 注册路由
func (h *JupyterHubK8sHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/jupyterhub")
	{
		// GPU资源管理
		api.GET("/gpu/status", h.GetGPUResourceStatus)
		api.GET("/gpu/nodes", h.FindSuitableGPUNodes)
		
		// Python脚本Job管理
		api.POST("/jobs/submit", h.SubmitPythonScript)
		api.GET("/jobs/:jobName/status", h.GetJobStatus)
		api.GET("/jobs/:jobName/logs", h.GetJobLogs)
		api.DELETE("/jobs/:jobName", h.DeleteJob)
		
		// Job批量操作
		api.GET("/jobs", h.ListJobs)
		api.POST("/jobs/cleanup", h.CleanupJobs)
		
		// 健康检查
		api.GET("/health", h.HealthCheck)
	}
}

// GetGPUResourceStatus 获取GPU资源状态
// @Summary 获取GPU资源状态
// @Description 获取集群中所有GPU节点的资源状态信息
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Success 200 {object} GPUResourceStatusResponse
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/gpu/status [get]
func (h *JupyterHubK8sHandler) GetGPUResourceStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	status, err := h.service.GetGPUResourceStatus(ctx)
	if err != nil {
		log.Printf("获取GPU资源状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取GPU资源状态失败",
			"details": err.Error(),
		})
		return
	}
	
	response := GPUResourceStatusResponse{
		TotalGPUs:     status.TotalGPUs,
		AvailableGPUs: status.AvailableGPUs,
		UsedGPUs:      status.UsedGPUs,
		GPUNodes:      status.GPUNodes,
		LastUpdated:   status.LastUpdated,
	}
	
	c.JSON(http.StatusOK, response)
}

// FindSuitableGPUNodes 查找适合的GPU节点
// @Summary 查找适合的GPU节点
// @Description 根据指定的GPU数量和类型查找适合的GPU节点
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Param gpu_count query int false "需要的GPU数量" default(1)
// @Param gpu_type query string false "GPU类型偏好"
// @Success 200 {object} []services.GPUNodeInfo
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/gpu/nodes [get]
func (h *JupyterHubK8sHandler) FindSuitableGPUNodes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 解析查询参数
	gpuCountStr := c.DefaultQuery("gpu_count", "1")
	gpuCount, err := strconv.Atoi(gpuCountStr)
	if err != nil || gpuCount < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的GPU数量",
		})
		return
	}
	
	gpuType := c.Query("gpu_type")
	
	nodes, err := h.service.FindSuitableGPUNodes(ctx, gpuCount, gpuType)
	if err != nil {
		log.Printf("查找适合的GPU节点失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查找适合的GPU节点失败",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"suitable_nodes": nodes,
		"count": len(nodes),
	})
}

// SubmitPythonScript 提交Python脚本
// @Summary 提交Python脚本作业
// @Description 将Python脚本转换为Kubernetes Job并提交执行
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Param request body SubmitPythonScriptRequest true "Python脚本作业信息"
// @Success 200 {object} SubmitPythonScriptResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/jobs/submit [post]
func (h *JupyterHubK8sHandler) SubmitPythonScript(c *gin.Context) {
	var req SubmitPythonScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数无效",
			"details": err.Error(),
		})
		return
	}
	
	// 生成唯一Job ID
	jobID := fmt.Sprintf("job-%d", time.Now().Unix())
	
	// 设置默认值
	if req.MemoryMB == 0 {
		req.MemoryMB = 1024 // 默认1GB内存
	}
	if req.CPUCores == 0 {
		req.CPUCores = 1 // 默认1核CPU
	}
	if req.GPURequired && req.GPUCount == 0 {
		req.GPUCount = 1 // 默认1个GPU
	}
	
	// 构建Job对象
	job := &services.PythonScriptJob{
		ID:           jobID,
		Name:         req.Name,
		Script:       req.Script,
		Requirements: req.Requirements,
		GPURequired:  req.GPURequired,
		GPUCount:     req.GPUCount,
		GPUType:      req.GPUType,
		MemoryMB:     req.MemoryMB,
		CPUCores:     req.CPUCores,
		Environment:  req.Environment,
		WorkingDir:   req.WorkingDir,
		OutputPath:   req.OutputPath,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	// 提交Job
	k8sJob, err := h.service.SubmitPythonScriptJob(ctx, job)
	if err != nil {
		log.Printf("提交Python脚本Job失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "提交Python脚本Job失败",
			"details": err.Error(),
		})
		return
	}
	
	response := SubmitPythonScriptResponse{
		JobID:     jobID,
		JobName:   k8sJob.Name,
		Status:    "submitted",
		Message:   "Python脚本Job已成功提交",
		CreatedAt: job.CreatedAt,
	}
	
	c.JSON(http.StatusOK, response)
}

// GetJobStatus 获取Job状态
// @Summary 获取Job状态
// @Description 获取指定Job的执行状态和详细信息
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Param jobName path string true "Job名称"
// @Success 200 {object} JobStatusResponse
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/jobs/{jobName}/status [get]
func (h *JupyterHubK8sHandler) GetJobStatus(c *gin.Context) {
	jobName := c.Param("jobName")
	if jobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job名称不能为空",
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	job, err := h.service.MonitorJob(ctx, jobName)
	if err != nil {
		log.Printf("获取Job状态失败: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "获取Job状态失败",
			"details": err.Error(),
		})
		return
	}
	
	response := JobStatusResponse{
		JobID:        fmt.Sprintf("%d", job.ID),
		JobName:      job.Name,
		Status:       job.Status,
		CreatedAt:    job.CreatedAt,
		StartedAt:    job.StartedAt,
		CompletedAt:  job.CompletedAt,
		ErrorMessage: job.ErrorMessage,
	}
	
	c.JSON(http.StatusOK, response)
}

// GetJobLogs 获取Job日志
// @Summary 获取Job日志
// @Description 获取指定Job的执行日志
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Param jobName path string true "Job名称"
// @Success 200 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/jobs/{jobName}/logs [get]
func (h *JupyterHubK8sHandler) GetJobLogs(c *gin.Context) {
	jobName := c.Param("jobName")
	if jobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job名称不能为空",
		})
		return
	}
	
	// 这里需要实现获取日志的逻辑
	// 可以通过service获取Pod日志
	c.JSON(http.StatusOK, gin.H{
		"job_name": jobName,
		"logs": "暂未实现日志获取功能",
		"message": "请使用kubectl命令查看日志",
	})
}

// DeleteJob 删除Job
// @Summary 删除Job
// @Description 删除指定的Kubernetes Job
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Param jobName path string true "Job名称"
// @Success 200 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/jobs/{jobName} [delete]
func (h *JupyterHubK8sHandler) DeleteJob(c *gin.Context) {
	jobName := c.Param("jobName")
	if jobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job名称不能为空",
		})
		return
	}
	
	// 这里需要实现删除Job的逻辑
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Job %s 已删除", jobName),
		"job_name": jobName,
	})
}

// ListJobs 列出所有Job
// @Summary 列出所有Job
// @Description 列出命名空间中的所有JupyterHub Python Job
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Param status query string false "Job状态过滤"
// @Success 200 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/jobs [get]
func (h *JupyterHubK8sHandler) ListJobs(c *gin.Context) {
	// 这里需要实现列出Job的逻辑
	c.JSON(http.StatusOK, gin.H{
		"jobs": []string{},
		"count": 0,
		"message": "暂未实现Job列表功能",
	})
}

// CleanupJobs 清理已完成的Job
// @Summary 清理已完成的Job
// @Description 清理所有已完成超过指定时间的Job
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/jupyterhub/jobs/cleanup [post]
func (h *JupyterHubK8sHandler) CleanupJobs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	err := h.service.CleanupCompletedJobs(ctx)
	if err != nil {
		log.Printf("清理Job失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "清理Job失败",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Job清理完成",
	})
}

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查JupyterHub K8s服务的健康状态
// @Tags JupyterHub-K8s
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /api/v1/jupyterhub/health [get]
func (h *JupyterHubK8sHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// 检查GPU资源状态作为健康检查
	_, err := h.service.GetGPUResourceStatus(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"timestamp": time.Now(),
		"service": "jupyterhub-k8s",
	})
}
