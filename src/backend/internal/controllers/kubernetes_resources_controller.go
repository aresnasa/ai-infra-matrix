package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// KubernetesResourcesController 提供对任意Kubernetes资源的通用CRUD接口
type KubernetesResourcesController struct {
	svc *services.KubernetesService
}

func NewKubernetesResourcesController() *KubernetesResourcesController {
	return &KubernetesResourcesController{svc: services.NewKubernetesService()}
}

// getClusterKubeConfig 根据集群ID获取kubeconfig
func (ctl *KubernetesResourcesController) getClusterKubeConfig(c *gin.Context) (string, *models.KubernetesCluster, bool) {
	id := c.Param("id")
	var cluster models.KubernetesCluster
	if err := database.DB.First(&cluster, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return "", nil, false
	}
	if cluster.KubeConfig == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster kubeconfig is empty"})
		return "", nil, false
	}
	return cluster.KubeConfig, &cluster, true
}

// DiscoverResources 返回API资源发现结果，供前端构建资源树
func (ctl *KubernetesResourcesController) DiscoverResources(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	disco, mapper, resources, err := ctl.svc.GetDiscoveryAndMapper(ctx, kubeConfig)
	_ = mapper // 暂未直接返回mapper
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 组装返回
	c.JSON(http.StatusOK, gin.H{
		"groups":    disco,     // 按组归类的简要信息
		"resources": resources, // 详细APIResource列表
	})
}

// GetClusterVersion 获取集群版本信息
func (ctl *KubernetesResourcesController) GetClusterVersion(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	versionInfo, err := ctl.svc.GetClusterVersion(ctx, kubeConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versionInfo)
}

// EnhancedDiscovery 增强的资源发现，包含 CRD 和版本信息
func (ctl *KubernetesResourcesController) EnhancedDiscovery(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := ctl.svc.GetEnhancedDiscovery(ctx, kubeConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListNamespaces 获取命名空间列表
func (ctl *KubernetesResourcesController) ListNamespaces(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	clientset, err := ctl.svc.ConnectToCluster(kubeConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	nss, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nss)
}

// ListResources 列出资源（命名空间内）
func (ctl *KubernetesResourcesController) ListResources(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	res := c.Param("resource")

	// 选择器与分页
	labelSelector := c.Query("labelSelector")
	fieldSelector := c.Query("fieldSelector")
	limitStr := c.DefaultQuery("limit", "0")
	cont := c.Query("continue")

	opts := metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector, Continue: cont}
	if limitStr != "0" {
		// 忽略转换错误，保持0为默认
		var l int64
		_, _ = fmt.Sscan(limitStr, &l)
		opts.Limit = l
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	list, err := ctl.svc.DynamicList(ctx, kubeConfig, res, namespace, opts)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ListClusterResources 列出集群范围资源（无命名空间）
func (ctl *KubernetesResourcesController) ListClusterResources(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	res := c.Param("resource")
	labelSelector := c.Query("labelSelector")
	fieldSelector := c.Query("fieldSelector")
	limitStr := c.DefaultQuery("limit", "0")
	cont := c.Query("continue")

	opts := metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector, Continue: cont}
	if limitStr != "0" {
		var l int64
		_, _ = fmt.Sscan(limitStr, &l)
		opts.Limit = l
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	list, err := ctl.svc.DynamicList(ctx, kubeConfig, res, "", opts)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetResource 获取资源
func (ctl *KubernetesResourcesController) GetResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	res := c.Param("resource")
	name := c.Param("name")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	obj, err := ctl.svc.DynamicGet(ctx, kubeConfig, res, namespace, name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, obj)
}

// GetClusterResource 获取集群级资源
func (ctl *KubernetesResourcesController) GetClusterResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	res := c.Param("resource")
	name := c.Param("name")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	obj, err := ctl.svc.DynamicGet(ctx, kubeConfig, res, "", name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, obj)
}

// CreateResource 创建资源（命名空间内）
func (ctl *KubernetesResourcesController) CreateResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	res := c.Param("resource")

	var obj unstructured.Unstructured
	if err := c.ShouldBindJSON(&obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	created, err := ctl.svc.DynamicCreate(ctx, kubeConfig, res, namespace, &obj)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, created)
}

// CreateClusterResource 创建集群级资源
func (ctl *KubernetesResourcesController) CreateClusterResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	res := c.Param("resource")

	var obj unstructured.Unstructured
	if err := c.ShouldBindJSON(&obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	created, err := ctl.svc.DynamicCreate(ctx, kubeConfig, res, "", &obj)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, created)
}

// UpdateResource 更新资源（命名空间内）
func (ctl *KubernetesResourcesController) UpdateResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	res := c.Param("resource")
	name := c.Param("name")

	var obj unstructured.Unstructured
	if err := c.ShouldBindJSON(&obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	updated, err := ctl.svc.DynamicUpdate(ctx, kubeConfig, res, namespace, name, &obj)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// UpdateClusterResource 更新集群级资源
func (ctl *KubernetesResourcesController) UpdateClusterResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	res := c.Param("resource")
	name := c.Param("name")

	var obj unstructured.Unstructured
	if err := c.ShouldBindJSON(&obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	updated, err := ctl.svc.DynamicUpdate(ctx, kubeConfig, res, "", name, &obj)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// PatchResource Patch资源
func (ctl *KubernetesResourcesController) PatchResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	res := c.Param("resource")
	name := c.Param("name")
	pt := types.PatchType(c.DefaultQuery("type", string(types.MergePatchType)))

	// 原始字节更可靠，这里直接读取Body
	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	patched, err := ctl.svc.DynamicPatch(ctx, kubeConfig, res, namespace, name, pt, raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, patched)
}

// PatchClusterResource Patch集群级资源
func (ctl *KubernetesResourcesController) PatchClusterResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	res := c.Param("resource")
	name := c.Param("name")
	pt := types.PatchType(c.DefaultQuery("type", string(types.MergePatchType)))

	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	patched, err := ctl.svc.DynamicPatch(ctx, kubeConfig, res, "", name, pt, raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, patched)
}

// DeleteResource 删除资源
func (ctl *KubernetesResourcesController) DeleteResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	res := c.Param("resource")
	name := c.Param("name")
	propagation := strings.ToLower(c.DefaultQuery("propagationPolicy", "Foreground"))
	var policy metav1.DeletionPropagation
	switch propagation {
	case "background":
		policy = metav1.DeletePropagationBackground
	case "orphan":
		policy = metav1.DeletePropagationOrphan
	default:
		policy = metav1.DeletePropagationForeground
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := ctl.svc.DynamicDelete(ctx, kubeConfig, res, namespace, name, metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// DeleteClusterResource 删除集群级资源
func (ctl *KubernetesResourcesController) DeleteClusterResource(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	res := c.Param("resource")
	name := c.Param("name")
	propagation := strings.ToLower(c.DefaultQuery("propagationPolicy", "Foreground"))
	var policy metav1.DeletionPropagation
	switch propagation {
	case "background":
		policy = metav1.DeletePropagationBackground
	case "orphan":
		policy = metav1.DeletePropagationOrphan
	default:
		policy = metav1.DeletePropagationForeground
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := ctl.svc.DynamicDelete(ctx, kubeConfig, res, "", name, metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// BatchListResources 并发批量列出多种资源（用于提升前端汇总性能）
func (ctl *KubernetesResourcesController) BatchListResources(c *gin.Context) {
	kubeConfig, _, ok := ctl.getClusterKubeConfig(c)
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	kindsParam := c.Query("kinds") // 逗号分隔，例如: pods,deployments,services
	if kindsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kinds is required"})
		return
	}
	kinds := strings.Split(kindsParam, ",")

	labelSelector := c.Query("labelSelector")
	fieldSelector := c.Query("fieldSelector")
	opts := metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 40*time.Second)
	defer cancel()

	results := make(map[string]interface{})
	eg, ctx2 := errgroup.WithContext(ctx)
	eg.SetLimit(6) // 限制并发，防止过载

	for _, k := range kinds {
		kind := strings.TrimSpace(k)
		if kind == "" {
			continue
		}
		k := strings.ToLower(kind)
		func(resName string) {
			eg.Go(func() error {
				list, err := ctl.svc.DynamicList(ctx2, kubeConfig, resName, namespace, opts)
				if err != nil {
					// 记录错误但不中断全部
					results[resName] = gin.H{"error": err.Error()}
					return nil
				}
				results[resName] = list
				return nil
			})
		}(k)
	}

	_ = eg.Wait()
	c.JSON(http.StatusOK, results)
}
