package controllers

import (
	"context"
	"net/http"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesController 提供集群管理API

type KubernetesController struct {
	service       *services.KubernetesService
	argoCDService *services.ArgoCDService
}

func NewKubernetesController() *KubernetesController {
	return &KubernetesController{
		service:       services.NewKubernetesService(),
		argoCDService: services.GetArgoCDService(),
	}
}

// ListClusters 获取所有集群
func (ctl *KubernetesController) ListClusters(c *gin.Context) {
	var clusters []models.KubernetesCluster
	db := database.DB
	if err := db.Find(&clusters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, clusters)
}

// CreateCluster 新建集群
func (ctl *KubernetesController) CreateCluster(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.KubernetesClusterCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cluster := models.KubernetesCluster{
		Name:           req.Name,
		Description:    req.Description,
		APIServer:      req.APIServer,
		KubeConfig:     req.KubeConfig,
		KubeConfigPath: req.KubeConfigPath,
		Namespace:      req.Namespace,
		Status:         "unknown",
		UserID:         userID, // 设置用户ID
	}

	// 如果提供了kubeconfig，尝试自动测试连接
	if req.KubeConfig != "" {
		if clientset, err := ctl.service.ConnectToCluster(req.KubeConfig); err == nil {
			// 尝试获取集群版本验证连接
			if version, err := clientset.Discovery().ServerVersion(); err == nil {
				cluster.Status = "connected"
				cluster.Version = version.String()
			} else {
				cluster.Status = "disconnected"
			}
		} else {
			cluster.Status = "disconnected"
		}
	}

	db := database.DB
	if err := db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果集群连接成功，同步到 ArgoCD
	if cluster.Status == "connected" {
		go func() {
			if err := ctl.argoCDService.SyncClusterToArgoCD(&cluster); err != nil {
				logrus.WithError(err).WithField("cluster_id", cluster.ID).Warn("Failed to sync cluster to ArgoCD")
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":       cluster,
		"argocd_synced": cluster.Status == "connected" && ctl.argoCDService.IsEnabled(),
	})
}

// UpdateCluster 更新集群
func (ctl *KubernetesController) UpdateCluster(c *gin.Context) {
	id := c.Param("id")
	var req models.KubernetesClusterUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := database.DB
	var cluster models.KubernetesCluster
	if err := db.First(&cluster, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if req.Name != "" {
		cluster.Name = req.Name
	}
	if req.Description != "" {
		cluster.Description = req.Description
	}
	if req.APIServer != "" {
		cluster.APIServer = req.APIServer
	}
	if req.KubeConfig != "" {
		cluster.KubeConfig = req.KubeConfig
	}
	if req.KubeConfigPath != "" {
		cluster.KubeConfigPath = req.KubeConfigPath
	}
	if req.Namespace != "" {
		cluster.Namespace = req.Namespace
	}
	if req.IsActive != nil {
		cluster.IsActive = *req.IsActive
	}
	if err := db.Save(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cluster)
}

// DeleteCluster 删除集群
func (ctl *KubernetesController) DeleteCluster(c *gin.Context) {
	id := c.Param("id")
	db := database.DB

	// 先获取集群信息，用于从 ArgoCD 移除
	var cluster models.KubernetesCluster
	if err := db.First(&cluster, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}

	// 从 ArgoCD 移除集群
	go func() {
		if err := ctl.argoCDService.RemoveClusterFromArgoCD(&cluster); err != nil {
			logrus.WithError(err).WithField("cluster_id", cluster.ID).Warn("Failed to remove cluster from ArgoCD")
		}
	}()

	// 删除数据库记录
	if err := db.Delete(&models.KubernetesCluster{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// TestConnection 测试集群连接
func (ctl *KubernetesController) TestConnection(c *gin.Context) {
	id := c.Param("id")
	db := database.DB
	var cluster models.KubernetesCluster
	if err := db.First(&cluster, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}

	// 测试连接
	clientset, err := ctl.service.ConnectToCluster(cluster.KubeConfig)
	if err != nil {
		// 更新状态为disconnected
		cluster.Status = "disconnected"
		db.Save(&cluster)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "连接失败: " + err.Error(),
			"status": "disconnected",
		})
		return
	}

	// 获取集群信息验证连接
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		cluster.Status = "disconnected"
		db.Save(&cluster)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "无法获取集群版本: " + err.Error(),
			"status": "disconnected",
		})
		return
	}

	// 更新状态为connected，同时保存版本信息
	cluster.Status = "connected"
	cluster.Version = version.String()
	db.Save(&cluster)

	// 尝试获取更多集群信息
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	nodeCount := 0
	if err == nil {
		nodeCount = len(nodes.Items)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	namespaceCount := 0
	if err == nil {
		namespaceCount = len(namespaces.Items)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "连接成功",
		"status":          "connected",
		"cluster_version": version.String(),
		"node_count":      nodeCount,
		"namespace_count": namespaceCount,
		"cluster_info": gin.H{
			"version":    version,
			"nodes":      nodeCount,
			"namespaces": namespaceCount,
		},
	})
}

// GetClusterInfo 获取集群详细信息
func (ctl *KubernetesController) GetClusterInfo(c *gin.Context) {
	id := c.Param("id")
	db := database.DB
	var cluster models.KubernetesCluster
	if err := db.First(&cluster, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}

	// 连接到集群
	clientset, err := ctl.service.ConnectToCluster(cluster.KubeConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "连接集群失败: " + err.Error()})
		return
	}

	// 获取节点信息
	nodes, err := clientset.CoreV1().Nodes().List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点信息失败: " + err.Error()})
		return
	}

	// 获取命名空间信息
	namespaces, err := clientset.CoreV1().Namespaces().List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取命名空间信息失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":          cluster,
		"nodes_count":      len(nodes.Items),
		"namespaces_count": len(namespaces.Items),
		"nodes":            nodes.Items,
		"namespaces":       namespaces.Items,
	})
}
