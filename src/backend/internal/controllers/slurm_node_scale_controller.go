package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// SlurmNodeScaleController 处理 SLURM 节点扩容相关请求
type SlurmNodeScaleController struct {
	slurmService     *services.SlurmClusterService
	saltstackService *services.SaltStackService
}

// NewSlurmNodeScaleController 创建新的控制器
func NewSlurmNodeScaleController(
	slurmService *services.SlurmClusterService,
	saltstackService *services.SaltStackService,
) *SlurmNodeScaleController {
	return &SlurmNodeScaleController{
		slurmService:     slurmService,
		saltstackService: saltstackService,
	}
}

// ScaleNodeRequest 扩容节点请求
type ScaleNodeRequest struct {
	ClusterID uint     `json:"cluster_id" binding:"required"`
	NodeNames []string `json:"node_names" binding:"required,min=1"`
	SSHConfig *struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Port     int    `json:"port"`
	} `json:"ssh_config"`
}

// CheckSaltStackResponse SaltStack 检查响应
type CheckSaltStackResponse struct {
	NodeName        string `json:"node_name"`
	HasSaltClient   bool   `json:"has_salt_client"`
	ClientVersion   string `json:"client_version,omitempty"`
	IsOnline        bool   `json:"is_online"`
	CanInstallSlurm bool   `json:"can_install_slurm"`
	Message         string `json:"message,omitempty"`
}

// CheckSaltStackClients 检查节点是否具备 SaltStack 客户端
// @Summary 检查节点 SaltStack 客户端状态
// @Tags SLURM节点扩容
// @Accept json
// @Produce json
// @Param request body ScaleNodeRequest true "节点列表"
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/nodes/check-saltstack [post]
func (c *SlurmNodeScaleController) CheckSaltStackClients(ctx *gin.Context) {
	var req ScaleNodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 检查每个节点的 SaltStack 客户端状态
	results := make([]CheckSaltStackResponse, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		result := CheckSaltStackResponse{
			NodeName: nodeName,
		}

		// 1. 检查节点是否在 SaltStack 中
		isAccepted, err := c.saltstackService.IsClientAccepted(ctx, nodeName)
		if err != nil {
			result.HasSaltClient = false
			result.IsOnline = false
			result.CanInstallSlurm = false
			result.Message = fmt.Sprintf("无法连接到 SaltStack Master: %v", err)
			results = append(results, result)
			continue
		}

		if !isAccepted {
			result.HasSaltClient = false
			result.IsOnline = false
			result.CanInstallSlurm = false
			result.Message = "节点未在 SaltStack 中注册，请先安装 Salt Minion"
			results = append(results, result)
			continue
		}

		result.HasSaltClient = true

		// 2. 检查节点是否在线
		isOnline, err := c.saltstackService.Ping(ctx, nodeName)
		if err != nil || !isOnline {
			result.IsOnline = false
			result.CanInstallSlurm = false
			result.Message = "节点离线，无法执行安装"
			results = append(results, result)
			continue
		}

		result.IsOnline = true

		// 3. 获取 Salt Minion 版本
		version, err := c.saltstackService.GetMinionVersion(ctx, nodeName)
		if err == nil {
			result.ClientVersion = version
		}

		// 4. 检查是否已安装 SLURM
		hasSlurm, err := c.saltstackService.CheckPackageInstalled(ctx, nodeName, "slurm-wlm")
		if err == nil && hasSlurm {
			result.CanInstallSlurm = true
			result.Message = "节点已安装 SLURM，可以直接配置"
		} else {
			result.CanInstallSlurm = true
			result.Message = "节点准备就绪，可以安装 SLURM"
		}

		results = append(results, result)
	}

	// 统计
	totalNodes := len(results)
	readyNodes := 0
	for _, r := range results {
		if r.CanInstallSlurm {
			readyNodes++
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"nodes": results,
			"summary": gin.H{
				"total":     totalNodes,
				"ready":     readyNodes,
				"not_ready": totalNodes - readyNodes,
			},
		},
	})
}

// ScaleNodes 扩容节点 - 触发 SLURM 安装和配置
// @Summary 扩容 SLURM 集群节点
// @Tags SLURM节点扩容
// @Accept json
// @Produce json
// @Param request body ScaleNodeRequest true "扩容请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/slurm/nodes/scale [post]
func (c *SlurmNodeScaleController) ScaleNodes(ctx *gin.Context) {
	var req ScaleNodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 1. 先检查 SaltStack 客户端
	checkCtx := context.Background()
	allReady := true
	notReadyNodes := []string{}

	for _, nodeName := range req.NodeNames {
		isAccepted, err := c.saltstackService.IsClientAccepted(checkCtx, nodeName)
		if err != nil || !isAccepted {
			allReady = false
			notReadyNodes = append(notReadyNodes, nodeName)
			continue
		}

		isOnline, err := c.saltstackService.Ping(checkCtx, nodeName)
		if err != nil || !isOnline {
			allReady = false
			notReadyNodes = append(notReadyNodes, nodeName)
		}
	}

	if !allReady {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success":         false,
			"error":           fmt.Sprintf("以下节点未就绪: %s", strings.Join(notReadyNodes, ", ")),
			"not_ready_nodes": notReadyNodes,
		})
		return
	}

	// 2. 获取集群信息
	cluster, err := c.slurmService.GetClusterByID(ctx, req.ClusterID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "集群不存在",
		})
		return
	}

	// 3. 通过 SaltStack 安装和配置 SLURM
	taskID := fmt.Sprintf("slurm-scale-%d-%d", req.ClusterID, time.Now().Unix())

	// 异步执行安装
	go func() {
		installCtx := context.Background()

		for _, nodeName := range req.NodeNames {
			// 3.1 安装 SLURM 包
			err := c.saltstackService.InstallSlurmNode(installCtx, nodeName, cluster)
			if err != nil {
				// 记录错误但继续处理其他节点
				fmt.Printf("安装 SLURM 到节点 %s 失败: %v\n", nodeName, err)
				continue
			}

			// 3.2 配置 SLURM
			err = c.saltstackService.ConfigureSlurmNode(installCtx, nodeName, cluster)
			if err != nil {
				fmt.Printf("配置节点 %s 失败: %v\n", nodeName, err)
				continue
			}

			// 3.3 启动 slurmd 服务
			err = c.saltstackService.StartSlurmService(installCtx, nodeName)
			if err != nil {
				fmt.Printf("启动节点 %s 的 slurmd 服务失败: %v\n", nodeName, err)
				continue
			}

			// 3.4 创建节点记录
			node := &models.SlurmNode{
				ClusterID: req.ClusterID,
				NodeName:  nodeName,
				Status:    "running",
			}
			c.slurmService.CreateNode(installCtx, node)
		}
	}()

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "节点扩容任务已启动",
		"data": gin.H{
			"task_id":    taskID,
			"cluster_id": req.ClusterID,
			"node_count": len(req.NodeNames),
			"nodes":      req.NodeNames,
		},
	})
}

// RegisterRoutes 注册路由
func (c *SlurmNodeScaleController) RegisterRoutes(api *gin.RouterGroup) {
	nodes := api.Group("/slurm/nodes")
	{
		nodes.POST("/check-saltstack", c.CheckSaltStackClients)
		nodes.POST("/scale", c.ScaleNodes)
	}
}
