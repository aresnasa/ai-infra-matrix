package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SlurmClusterController SLURM集群管理控制器
type SlurmClusterController struct {
	db      *gorm.DB
	service *services.SlurmClusterService
}

func NewSlurmClusterController(db *gorm.DB) *SlurmClusterController {
	return &SlurmClusterController{
		db:      db,
		service: services.NewSlurmClusterService(db),
	}
}

// CreateCluster 创建SLURM集群
// @Summary 创建SLURM集群
// @Description 创建一个新的SLURM集群，包括节点配置和SaltStack集成
// @Tags SLURM Cluster
// @Accept json
// @Produce json
// @Param request body models.CreateClusterRequest true "集群创建请求"
// @Success 201 {object} models.SlurmCluster
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/slurm/clusters [post]
func (c *SlurmClusterController) CreateCluster(ctx *gin.Context) {
	var req models.CreateClusterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind cluster creation request")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 验证请求参数
	if len(req.Nodes) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "No nodes specified",
			"message": "At least one node must be specified for the cluster",
		})
		return
	}

	// 验证至少有一个master节点
	hasMaster := false
	for _, node := range req.Nodes {
		if node.NodeType == "master" {
			hasMaster = true
			break
		}
	}
	if !hasMaster {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "No master node specified",
			"message": "At least one master node is required",
		})
		return
	}

	cluster, err := c.service.CreateCluster(ctx.Request.Context(), req, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to create SLURM cluster")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create cluster",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Cluster created successfully",
		"data":    cluster,
	})

	logrus.WithFields(logrus.Fields{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"user_id":      userID,
		"node_count":   len(cluster.Nodes),
	}).Info("SLURM cluster created")
}

// DeployCluster 部署SLURM集群
// @Summary 部署SLURM集群
// @Description 异步部署SLURM集群，包括SaltStack安装和SLURM配置
// @Tags SLURM Cluster
// @Accept json
// @Produce json
// @Param request body models.DeployClusterRequest true "集群部署请求"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/slurm/clusters/deploy [post]
func (c *SlurmClusterController) DeployCluster(ctx *gin.Context) {
	var req models.DeployClusterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind cluster deployment request")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 验证集群是否属于当前用户
	var cluster models.SlurmCluster
	if err := c.db.Where("id = ? AND created_by = ?", req.ClusterID, userID).First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Cluster not found or access denied",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
		}
		return
	}

	deploymentID, err := c.service.DeployClusterAsync(ctx.Request.Context(), req, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to start cluster deployment")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start deployment",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"success":       true,
		"message":       "Deployment started successfully",
		"deployment_id": deploymentID,
		"status_url":    "/api/slurm/deployments/" + deploymentID + "/status",
	})

	logrus.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"cluster_id":    req.ClusterID,
		"action":        req.Action,
		"user_id":       userID,
	}).Info("SLURM cluster deployment started")
}

// GetDeploymentStatus 获取部署状态
// @Summary 获取部署状态
// @Description 获取集群部署的详细状态和进度信息
// @Tags SLURM Cluster
// @Produce json
// @Param deploymentId path string true "部署ID"
// @Success 200 {object} services.DeploymentTask
// @Failure 404 {object} map[string]interface{}
// @Router /api/slurm/deployments/{deploymentId}/status [get]
func (c *SlurmClusterController) GetDeploymentStatus(ctx *gin.Context) {
	deploymentID := ctx.Param("deploymentId")
	if deploymentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Deployment ID is required",
		})
		return
	}

	task, err := c.service.GetDeploymentStatus(deploymentID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error":   "Deployment not found",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"deployment_id": task.ID,
		"cluster_id":    task.ClusterID,
		"status":        task.Status,
		"progress":      task.Progress,
		"current_step":  task.CurrentStep,
		"started_at":    task.StartedAt,
		"updated_at":    task.UpdatedAt,
		"node_tasks":    task.NodeTasks,
	})
}

// StreamDeploymentProgress 流式获取部署进度
// @Summary 流式获取部署进度
// @Description 通过Server-Sent Events流式获取集群部署的实时进度
// @Tags SLURM Cluster
// @Produce text/event-stream
// @Param deploymentId path string true "部署ID"
// @Router /api/slurm/deployments/{deploymentId}/stream [get]
func (c *SlurmClusterController) StreamDeploymentProgress(ctx *gin.Context) {
	deploymentID := ctx.Param("deploymentId")
	if deploymentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Deployment ID is required",
		})
		return
	}

	// 设置SSE头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")

	// 创建取消上下文
	ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), "client", ctx.Writer))

	// 流式推送进度
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case <-ticker.C:
			task, err := c.service.GetDeploymentStatus(deploymentID)
			if err != nil {
				ctx.SSEvent("error", gin.H{"message": "Deployment not found"})
				return
			}

			ctx.SSEvent("progress", gin.H{
				"deployment_id": task.ID,
				"status":        task.Status,
				"progress":      task.Progress,
				"current_step":  task.CurrentStep,
				"node_tasks":    task.NodeTasks,
			})

			ctx.Writer.Flush()

			// 如果部署完成，停止流
			if task.Status == "completed" || task.Status == "failed" {
				return
			}
		}
	}
}

// ListClusters 列出集群
// @Summary 列出集群
// @Description 获取当前用户的所有SLURM集群列表
// @Tags SLURM Cluster
// @Produce json
// @Param status query string false "状态过滤" Enums(pending,deploying,running,scaling,failed,stopped)
// @Param limit query int false "限制返回数量" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/clusters [get]
func (c *SlurmClusterController) ListClusters(ctx *gin.Context) {
	// 获取当前用户ID
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 获取查询参数
	statusFilter := ctx.Query("status")
	limitStr := ctx.DefaultQuery("limit", "20")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	clusters, err := c.service.ListClusters(userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to list clusters")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve clusters",
		})
		return
	}

	// 过滤和分页
	var filteredClusters []models.SlurmCluster
	for _, cluster := range clusters {
		if statusFilter == "" || cluster.Status == statusFilter {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	total := len(filteredClusters)
	start := offset
	if start > total {
		start = total
	}

	end := start + limit
	if end > total {
		end = total
	}

	paginatedClusters := filteredClusters[start:end]

	ctx.JSON(http.StatusOK, gin.H{
		"clusters": paginatedClusters,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetCluster 获取集群详情
// @Summary 获取集群详情
// @Description 获取指定SLURM集群的详细信息
// @Tags SLURM Cluster
// @Produce json
// @Param clusterId path int true "集群ID"
// @Success 200 {object} models.SlurmCluster
// @Failure 404 {object} map[string]interface{}
// @Router /api/slurm/clusters/{clusterId} [get]
func (c *SlurmClusterController) GetCluster(ctx *gin.Context) {
	clusterIDStr := ctx.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cluster ID",
		})
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	cluster, err := c.service.GetCluster(uint(clusterID), userID.(uint))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Cluster not found or access denied",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve cluster",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, cluster)
}

// ScaleCluster 扩缩容集群
// @Summary 扩缩容集群
// @Description 对SLURM集群进行扩容或缩容操作
// @Tags SLURM Cluster
// @Accept json
// @Produce json
// @Param request body models.ScaleClusterRequest true "集群扩缩容请求"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/slurm/clusters/scale [post]
func (c *SlurmClusterController) ScaleCluster(ctx *gin.Context) {
	var req models.ScaleClusterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 转换为部署请求
	deployReq := models.DeployClusterRequest{
		ClusterID: req.ClusterID,
		Action:    req.Action,
		Config:    req.Config,
	}

	deploymentID, err := c.service.DeployClusterAsync(ctx.Request.Context(), deployReq, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to start cluster scaling")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start scaling",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"success":       true,
		"message":       "Scaling operation started successfully",
		"deployment_id": deploymentID,
		"status_url":    "/api/slurm/deployments/" + deploymentID + "/status",
	})
}

// GetDeploymentLogs 获取部署日志
// @Summary 获取部署日志
// @Description 获取集群部署的详细日志信息
// @Tags SLURM Cluster
// @Produce json
// @Param deploymentId path string true "部署ID"
// @Param nodeId query int false "节点ID过滤"
// @Param stepType query string false "步骤类型过滤" Enums(ssh,salt,slurm,validation)
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/deployments/{deploymentId}/logs [get]
func (c *SlurmClusterController) GetDeploymentLogs(ctx *gin.Context) {
	deploymentID := ctx.Param("deploymentId")
	if deploymentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Deployment ID is required",
		})
		return
	}

	// 获取查询参数
	nodeIDStr := ctx.Query("nodeId")
	stepType := ctx.Query("stepType")

	// 查询部署记录
	var deployment models.ClusterDeployment
	query := c.db.Where("deployment_id = ?", deploymentID).
		Preload("Steps").
		Preload("InstallTasks").
		Preload("InstallTasks.Steps").
		Preload("InstallTasks.SSHLogs")

	if err := query.First(&deployment).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Deployment not found",
		})
		return
	}

	// 查询SSH日志
	var sshLogs []models.SSHExecutionLog
	logQuery := c.db.Where("task_id IN (SELECT id FROM node_install_tasks WHERE deployment_id = ?)", deployment.ID)

	if nodeIDStr != "" {
		if nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32); err == nil {
			logQuery = logQuery.Where("node_id = ?", uint(nodeID))
		}
	}

	logQuery.Order("started_at ASC").Find(&sshLogs)

	// 过滤步骤类型
	var filteredSteps []models.DeploymentStep
	for _, step := range deployment.Steps {
		if stepType == "" || step.StepType == stepType {
			filteredSteps = append(filteredSteps, step)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"deployment":    deployment,
		"steps":         filteredSteps,
		"install_tasks": deployment.InstallTasks,
		"ssh_logs":      sshLogs,
	})
}

// RegisterRoutes 注册路由
func (c *SlurmClusterController) RegisterRoutes(api *gin.RouterGroup) {
	clusters := api.Group("/slurm/clusters")
	{
		clusters.POST("", c.CreateCluster)
		clusters.GET("", c.ListClusters)
		clusters.GET("/:clusterId", c.GetCluster)
		clusters.POST("/deploy", c.DeployCluster)
		clusters.POST("/scale", c.ScaleCluster)
	}

	deployments := api.Group("/slurm/deployments")
	{
		deployments.GET("/:deploymentId/status", c.GetDeploymentStatus)
		deployments.GET("/:deploymentId/stream", c.StreamDeploymentProgress)
		deployments.GET("/:deploymentId/logs", c.GetDeploymentLogs)
	}
}
