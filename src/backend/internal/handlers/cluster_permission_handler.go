package handlers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ClusterPermissionHandler 集群权限处理器
type ClusterPermissionHandler struct {
	service *services.ClusterPermissionService
	log     *logrus.Logger
}

// NewClusterPermissionHandler 创建集群权限处理器
func NewClusterPermissionHandler(db *gorm.DB) *ClusterPermissionHandler {
	return &ClusterPermissionHandler{
		service: services.NewClusterPermissionService(db),
		log:     logrus.StandardLogger(),
	}
}

// RegisterRoutes 注册路由
func (h *ClusterPermissionHandler) RegisterRoutes(r *gin.RouterGroup) {
	// SLURM 集群权限
	slurm := r.Group("/slurm-permissions")
	{
		slurm.GET("", h.ListSlurmPermissions)
		slurm.GET("/:id", h.GetSlurmPermission)
		slurm.POST("", h.GrantSlurmPermission)
		slurm.PUT("/:id", h.UpdateSlurmPermission)
		slurm.DELETE("/:id", h.RevokeSlurmPermission)
		slurm.POST("/check", h.CheckSlurmAccess)
	}

	// SaltStack 集群权限
	salt := r.Group("/saltstack-permissions")
	{
		salt.GET("", h.ListSaltstackPermissions)
		salt.GET("/:id", h.GetSaltstackPermission)
		salt.POST("", h.GrantSaltstackPermission)
		salt.PUT("/:id", h.UpdateSaltstackPermission)
		salt.DELETE("/:id", h.RevokeSaltstackPermission)
		salt.POST("/check", h.CheckSaltstackAccess)
	}

	// 用户权限汇总
	r.GET("/user-permissions/:user_id", h.GetUserClusterPermissions)
	r.GET("/user-permissions/:user_id/access-list", h.GetClusterAccessList)
	r.GET("/my-permissions", h.GetMyPermissions)
	r.GET("/my-access-list", h.GetMyAccessList)

	// 权限日志
	r.GET("/permission-logs", h.GetPermissionLogs)
}

// ========================================
// SLURM 权限接口
// ========================================

// ListSlurmPermissions 列出SLURM权限
// @Summary 列出SLURM集群权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param user_id query int false "用户ID"
// @Param cluster_id query int false "集群ID"
// @Param is_active query bool false "是否有效"
// @Param include_expired query bool false "包含过期的"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} models.SlurmPermissionListResponse
// @Router /api/v1/cluster-permissions/slurm-permissions [get]
func (h *ClusterPermissionHandler) ListSlurmPermissions(c *gin.Context) {
	var query models.ClusterPermissionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ListSlurmPermissions(c.Request.Context(), &query)
	if err != nil {
		h.log.WithError(err).Error("Failed to list SLURM permissions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSlurmPermission 获取单个SLURM权限
// @Summary 获取SLURM权限详情
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Success 200 {object} models.SlurmClusterPermission
// @Router /api/v1/cluster-permissions/slurm-permissions/{id} [get]
func (h *ClusterPermissionHandler) GetSlurmPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	perm, err := h.service.GetSlurmPermission(c.Request.Context(), uint(id))
	if err != nil {
		h.log.WithError(err).Error("Failed to get SLURM permission")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perm)
}

// GrantSlurmPermission 授予SLURM权限
// @Summary 授予用户SLURM集群权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param body body models.GrantSlurmPermissionInput true "授权信息"
// @Success 201 {object} models.SlurmClusterPermission
// @Router /api/v1/cluster-permissions/slurm-permissions [post]
func (h *ClusterPermissionHandler) GrantSlurmPermission(c *gin.Context) {
	var input models.GrantSlurmPermissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	perm, err := h.service.GrantSlurmPermission(c.Request.Context(), &input, userID.(uint), c.ClientIP())
	if err != nil {
		h.log.WithError(err).Error("Failed to grant SLURM permission")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, perm)
}

// UpdateSlurmPermission 更新SLURM权限
// @Summary 更新SLURM权限配置
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Param body body models.UpdateSlurmPermissionInput true "更新信息"
// @Success 200 {object} models.SlurmClusterPermission
// @Router /api/v1/cluster-permissions/slurm-permissions/{id} [put]
func (h *ClusterPermissionHandler) UpdateSlurmPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var input models.UpdateSlurmPermissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	perm, err := h.service.UpdateSlurmPermission(c.Request.Context(), uint(id), &input, userID.(uint), c.ClientIP())
	if err != nil {
		h.log.WithError(err).Error("Failed to update SLURM permission")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perm)
}

// RevokeSlurmPermission 撤销SLURM权限
// @Summary 撤销用户SLURM权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Param body body models.RevokeClusterPermissionInput true "撤销原因"
// @Success 200 {object} map[string]string
// @Router /api/v1/cluster-permissions/slurm-permissions/{id} [delete]
func (h *ClusterPermissionHandler) RevokeSlurmPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var input models.RevokeClusterPermissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	if err := h.service.RevokeSlurmPermission(c.Request.Context(), uint(id), input.Reason, userID.(uint), c.ClientIP()); err != nil {
		h.log.WithError(err).Error("Failed to revoke SLURM permission")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission revoked successfully"})
}

// CheckSlurmAccess 检查SLURM访问权限
// @Summary 检查用户是否有SLURM集群访问权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param body body CheckSlurmAccessInput true "检查参数"
// @Success 200 {object} models.VerifyPermissionResult
// @Router /api/v1/cluster-permissions/slurm-permissions/check [post]
func (h *ClusterPermissionHandler) CheckSlurmAccess(c *gin.Context) {
	var input struct {
		UserID    uint   `json:"user_id" binding:"required"`
		ClusterID uint   `json:"cluster_id" binding:"required"`
		Verb      string `json:"verb" binding:"required"`
		Partition string `json:"partition"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CheckSlurmAccess(c.Request.Context(), input.UserID, input.ClusterID, models.ClusterPermissionVerb(input.Verb), input.Partition)
	if err != nil {
		h.log.WithError(err).Error("Failed to check SLURM access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ========================================
// SaltStack 权限接口
// ========================================

// ListSaltstackPermissions 列出SaltStack权限
// @Summary 列出SaltStack集群权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param user_id query int false "用户ID"
// @Param master_id query string false "Master ID"
// @Param is_active query bool false "是否有效"
// @Param include_expired query bool false "包含过期的"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} models.SaltstackPermissionListResponse
// @Router /api/v1/cluster-permissions/saltstack-permissions [get]
func (h *ClusterPermissionHandler) ListSaltstackPermissions(c *gin.Context) {
	var query models.ClusterPermissionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ListSaltstackPermissions(c.Request.Context(), &query)
	if err != nil {
		h.log.WithError(err).Error("Failed to list SaltStack permissions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSaltstackPermission 获取单个SaltStack权限
// @Summary 获取SaltStack权限详情
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Success 200 {object} models.SaltstackClusterPermission
// @Router /api/v1/cluster-permissions/saltstack-permissions/{id} [get]
func (h *ClusterPermissionHandler) GetSaltstackPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	perm, err := h.service.GetSaltstackPermission(c.Request.Context(), uint(id))
	if err != nil {
		h.log.WithError(err).Error("Failed to get SaltStack permission")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perm)
}

// GrantSaltstackPermission 授予SaltStack权限
// @Summary 授予用户SaltStack集群权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param body body models.GrantSaltstackPermissionInput true "授权信息"
// @Success 201 {object} models.SaltstackClusterPermission
// @Router /api/v1/cluster-permissions/saltstack-permissions [post]
func (h *ClusterPermissionHandler) GrantSaltstackPermission(c *gin.Context) {
	var input models.GrantSaltstackPermissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	perm, err := h.service.GrantSaltstackPermission(c.Request.Context(), &input, userID.(uint), c.ClientIP())
	if err != nil {
		h.log.WithError(err).Error("Failed to grant SaltStack permission")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, perm)
}

// UpdateSaltstackPermission 更新SaltStack权限
// @Summary 更新SaltStack权限配置
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Param body body models.UpdateSaltstackPermissionInput true "更新信息"
// @Success 200 {object} models.SaltstackClusterPermission
// @Router /api/v1/cluster-permissions/saltstack-permissions/{id} [put]
func (h *ClusterPermissionHandler) UpdateSaltstackPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var input models.UpdateSaltstackPermissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	perm, err := h.service.UpdateSaltstackPermission(c.Request.Context(), uint(id), &input, userID.(uint), c.ClientIP())
	if err != nil {
		h.log.WithError(err).Error("Failed to update SaltStack permission")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perm)
}

// RevokeSaltstackPermission 撤销SaltStack权限
// @Summary 撤销用户SaltStack权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Param body body models.RevokeClusterPermissionInput true "撤销原因"
// @Success 200 {object} map[string]string
// @Router /api/v1/cluster-permissions/saltstack-permissions/{id} [delete]
func (h *ClusterPermissionHandler) RevokeSaltstackPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var input models.RevokeClusterPermissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	if err := h.service.RevokeSaltstackPermission(c.Request.Context(), uint(id), input.Reason, userID.(uint), c.ClientIP()); err != nil {
		h.log.WithError(err).Error("Failed to revoke SaltStack permission")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission revoked successfully"})
}

// CheckSaltstackAccess 检查SaltStack访问权限
// @Summary 检查用户是否有SaltStack集群访问权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param body body CheckSaltstackAccessInput true "检查参数"
// @Success 200 {object} models.VerifyPermissionResult
// @Router /api/v1/cluster-permissions/saltstack-permissions/check [post]
func (h *ClusterPermissionHandler) CheckSaltstackAccess(c *gin.Context) {
	var input struct {
		UserID   uint   `json:"user_id" binding:"required"`
		MasterID string `json:"master_id" binding:"required"`
		Verb     string `json:"verb" binding:"required"`
		MinionID string `json:"minion_id"`
		Function string `json:"function"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CheckSaltstackAccess(c.Request.Context(), input.UserID, input.MasterID, models.ClusterPermissionVerb(input.Verb), input.MinionID, input.Function)
	if err != nil {
		h.log.WithError(err).Error("Failed to check SaltStack access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ========================================
// 用户权限汇总接口
// ========================================

// GetUserClusterPermissions 获取用户的所有集群权限
// @Summary 获取用户的所有集群权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {object} models.UserClusterPermissions
// @Router /api/v1/cluster-permissions/user-permissions/{user_id} [get]
func (h *ClusterPermissionHandler) GetUserClusterPermissions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	perms, err := h.service.GetUserClusterPermissions(c.Request.Context(), uint(userID))
	if err != nil {
		h.log.WithError(err).Error("Failed to get user cluster permissions")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perms)
}

// GetClusterAccessList 获取用户可访问的集群列表
// @Summary 获取用户可访问的集群列表
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {array} models.ClusterAccessInfo
// @Router /api/v1/cluster-permissions/user-permissions/{user_id}/access-list [get]
func (h *ClusterPermissionHandler) GetClusterAccessList(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	list, err := h.service.GetClusterAccessList(c.Request.Context(), uint(userID))
	if err != nil {
		h.log.WithError(err).Error("Failed to get cluster access list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access list"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// GetMyPermissions 获取当前用户的权限
// @Summary 获取当前登录用户的所有集群权限
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Success 200 {object} models.UserClusterPermissions
// @Router /api/v1/cluster-permissions/my-permissions [get]
func (h *ClusterPermissionHandler) GetMyPermissions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	perms, err := h.service.GetUserClusterPermissions(c.Request.Context(), userID.(uint))
	if err != nil {
		h.log.WithError(err).Error("Failed to get my cluster permissions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get permissions"})
		return
	}

	c.JSON(http.StatusOK, perms)
}

// GetMyAccessList 获取当前用户可访问的集群列表
// @Summary 获取当前登录用户可访问的集群列表
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Success 200 {array} models.ClusterAccessInfo
// @Router /api/v1/cluster-permissions/my-access-list [get]
func (h *ClusterPermissionHandler) GetMyAccessList(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	list, err := h.service.GetClusterAccessList(c.Request.Context(), userID.(uint))
	if err != nil {
		h.log.WithError(err).Error("Failed to get my access list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access list"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// ========================================
// 权限日志接口
// ========================================

// GetPermissionLogs 获取权限变更日志
// @Summary 获取权限变更日志
// @Tags ClusterPermissions
// @Accept json
// @Produce json
// @Param permission_type query string false "权限类型"
// @Param permission_id query int false "权限ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/cluster-permissions/permission-logs [get]
func (h *ClusterPermissionHandler) GetPermissionLogs(c *gin.Context) {
	permType := c.Query("permission_type")
	permID, _ := strconv.ParseUint(c.Query("permission_id"), 10, 32)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.service.GetPermissionLogs(c.Request.Context(), models.ClusterPermissionType(permType), uint(permID), page, pageSize)
	if err != nil {
		h.log.WithError(err).Error("Failed to get permission logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     logs,
	})
}
