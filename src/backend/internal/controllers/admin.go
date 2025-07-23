package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/sirupsen/logrus"
)

type AdminController struct {
	db              *gorm.DB
	rbacService     *services.RBACService
	ldapService     *services.LDAPService
	ldapSyncService *services.LDAPSyncService
}

func NewAdminController(db *gorm.DB) *AdminController {
	ldapService := services.NewLDAPService(db)
	userService := services.NewUserService()
	rbacService := services.NewRBACService(db)
	
	return &AdminController{
		db:              db,
		rbacService:     rbacService,
		ldapService:     ldapService,
		ldapSyncService: services.NewLDAPSyncService(db, ldapService, userService, rbacService),
	}
}

// GetAllUsers 获取所有用户（管理员专用）
// @Summary 获取所有用户
// @Description 管理员获取系统中的所有用户
// @Tags Admin
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Router /admin/users [get]
func (c *AdminController) GetAllUsers(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var users []models.User
	var total int64

	// 获取总数
	if err := c.db.Model(&models.User{}).Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取用户列表，预加载角色和用户组
	if err := c.db.Preload("Roles").Preload("UserGroups").
		Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUserDetail 获取用户详情（管理员专用）
// @Summary 获取用户详情
// @Description 管理员获取指定用户的详细信息
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} models.User
// @Router /admin/users/{id} [get]
func (c *AdminController) GetUserDetail(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var user models.User
	if err := c.db.Preload("Roles").Preload("UserGroups").Preload("Projects").
		First(&user, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// GetUserWithAuthSource 获取用户详情包含认证来源信息
// @Summary 获取用户详情（包含认证来源）
// @Description 管理员获取指定用户的详细信息，包括认证来源
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{}
// @Router /admin/users/{id}/details [get]
func (c *AdminController) GetUserWithAuthSource(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var user models.User
	if err := c.db.Preload("Roles").Preload("UserGroups").
		First(&user, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 判断用户认证来源
	authSource := "local"
	isAdmin := false
	
	// 检查是否为管理员
	for _, role := range user.Roles {
		if role.Name == "admin" {
			isAdmin = true
			break
		}
	}
	
	// 检查是否为LDAP用户（没有本地密码或密码为空）
	if user.Password == "" {
		authSource = "ldap"
	}

	response := gin.H{
		"user":        user,
		"auth_source": authSource,
		"is_admin":    isAdmin,
		"can_disable": authSource != "local" || !isAdmin, // 本地管理员不能被禁用
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateUserStatus 更新用户状态（管理员专用）
// @Summary 更新用户状态
// @Description 管理员启用或禁用用户
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body map[string]bool true "状态更新请求"
// @Success 200 {object} models.User
// @Router /admin/users/{id}/status [put]
func (c *AdminController) UpdateUserStatus(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := c.db.First(&user, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.IsActive = req.IsActive
	if err := c.db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// UpdateUserStatusEnhanced 更新用户状态（增强版）
// @Summary 更新用户状态
// @Description 管理员启用或禁用用户，保护本地管理员账户
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body map[string]interface{} true "状态更新请求"
// @Success 200 {object} map[string]interface{}
// @Router /admin/users/{id}/status [put]
func (c *AdminController) UpdateUserStatusEnhanced(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req struct {
		IsActive bool   `json:"is_active"`
		Reason   string `json:"reason,omitempty"` // 操作原因
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := c.db.Preload("Roles").First(&user, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否为本地管理员用户
	isLocalAdmin := user.Password != "" // 有本地密码
	for _, role := range user.Roles {
		if role.Name == "admin" {
			isLocalAdmin = isLocalAdmin && true
			break
		}
	}

	// 保护本地管理员账户不被禁用
	if isLocalAdmin && !req.IsActive {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "不能禁用本地管理员账户，这是为了防止失去系统访问权限的安全措施",
			"suggestion": "如需禁用此管理员，请先确保有其他活跃的管理员账户",
		})
		return
	}

	// 记录操作日志
	logrus.WithFields(logrus.Fields{
		"operator_id":   userID,
		"target_user":   user.Username,
		"action":        "update_status",
		"new_status":    req.IsActive,
		"reason":        req.Reason,
		"is_local_admin": isLocalAdmin,
	}).Info("User status update")

	user.IsActive = req.IsActive
	if err := c.db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user":    user,
		"message": "用户状态更新成功",
	})
}

// DeleteUser 删除用户（管理员专用）
// @Summary 删除用户
// @Description 管理员删除指定用户
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 204
// @Router /admin/users/{id} [delete]
func (c *AdminController) DeleteUser(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "delete", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 不能删除自己
	if uint(id) == userID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}

	var user models.User
	if err := c.db.First(&user, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := c.db.Delete(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetAllProjects 获取所有项目（管理员专用）
// @Summary 获取所有项目
// @Description 管理员获取系统中的所有项目
// @Tags Admin
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param user_id query int false "按用户ID筛选"
// @Success 200 {object} map[string]interface{}
// @Router /admin/projects [get]
func (c *AdminController) GetAllProjects(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// 筛选参数
	filterUserID := ctx.Query("user_id")

	var projects []models.Project
	var total int64

	query := c.db.Model(&models.Project{})
	if filterUserID != "" {
		query = query.Where("user_id = ?", filterUserID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取项目列表，预加载用户信息
	if err := query.Preload("User").
		Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetProjectDetail 获取项目详情（管理员专用）
// @Summary 获取项目详情
// @Description 管理员获取指定项目的详细信息
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} models.Project
// @Router /admin/projects/{id} [get]
func (c *AdminController) GetProjectDetail(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var project models.Project
	if err := c.db.Preload("User").Preload("Hosts").Preload("Variables").Preload("Tasks").
		First(&project, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, project)
}

// TransferProject 转移项目所有权（管理员专用）
// @Summary 转移项目所有权
// @Description 管理员将项目转移给其他用户
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Param request body map[string]uint true "转移请求"
// @Success 200 {object} models.Project
// @Router /admin/projects/{id}/transfer [put]
func (c *AdminController) TransferProject(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req struct {
		NewUserID uint `json:"new_user_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查新用户是否存在
	var newUser models.User
	if err := c.db.First(&newUser, req.NewUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "目标用户不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var project models.Project
	if err := c.db.First(&project, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	project.UserID = req.NewUserID
	if err := c.db.Save(&project).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载项目信息
	if err := c.db.Preload("User").First(&project, project.ID).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, project)
}

// GetSystemStats 获取系统统计信息（管理员专用）
// @Summary 获取系统统计信息
// @Description 管理员获取系统使用统计
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/stats [get]
func (c *AdminController) GetSystemStats(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "*", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var userCount, projectCount, roleCount, groupCount int64

	// 统计用户数量
	if err := c.db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计项目数量
	if err := c.db.Model(&models.Project{}).Count(&projectCount).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计角色数量
	if err := c.db.Model(&models.Role{}).Count(&roleCount).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计用户组数量
	if err := c.db.Model(&models.UserGroup{}).Count(&groupCount).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取最近活跃用户
	var activeUsers []models.User
	if err := c.db.Where("last_login IS NOT NULL").
		Order("last_login DESC").Limit(5).Find(&activeUsers).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取最近创建的项目
	var recentProjects []models.Project
	if err := c.db.Preload("User").
		Order("created_at DESC").Limit(5).Find(&recentProjects).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"user_count":    userCount,
			"project_count": projectCount,
			"role_count":    roleCount,
			"group_count":   groupCount,
		},
		"active_users":     activeUsers,
		"recent_projects": recentProjects,
	})
}

// InitializeRBAC 初始化RBAC系统（管理员专用）
// @Summary 初始化RBAC系统
// @Description 初始化默认的角色和权限
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {string} string "初始化成功"
// @Router /admin/rbac/initialize [post]
func (c *AdminController) InitializeRBAC(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "*", "*", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	if err := c.rbacService.InitializeDefaultRBAC(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "RBAC系统初始化成功"})
}

// GetLDAPConfig 获取LDAP配置（管理员专用）
// @Summary 获取LDAP配置
// @Description 管理员获取当前LDAP配置
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} models.LDAPConfig
// @Router /admin/ldap/config [get]
func (c *AdminController) GetLDAPConfig(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "system", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	config, err := c.ldapService.GetLDAPConfig()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 隐藏敏感信息
	config.BindPassword = "***"

	ctx.JSON(http.StatusOK, config)
}

// UpdateLDAPConfig 更新LDAP配置（管理员专用）
// @Summary 更新LDAP配置
// @Description 管理员更新LDAP配置
// @Tags Admin
// @Accept json
// @Produce json
// @Param config body models.LDAPConfigRequest true "LDAP配置"
// @Success 200 {object} models.LDAPConfig
// @Router /admin/ldap/config [put]
func (c *AdminController) UpdateLDAPConfig(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "system", "write", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.LDAPConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := c.ldapService.UpdateLDAPConfig(&req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 隐藏敏感信息
	config.BindPassword = "***"

	ctx.JSON(http.StatusOK, config)
}

// TestLDAPConnection 测试LDAP连接（管理员专用）
// @Summary 测试LDAP连接
// @Description 管理员测试LDAP服务器连接
// @Tags Admin
// @Accept json
// @Produce json
// @Param config body models.LDAPTestRequest true "LDAP测试配置"
// @Success 200 {object} map[string]string
// @Router /admin/ldap/test [post]
func (c *AdminController) TestLDAPConnection(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "system", "write", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.LDAPTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.ldapService.TestLDAPConnection(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "LDAP连接测试成功"})
}

// SyncLDAPUsers 同步LDAP用户和用户组（管理员专用）
// @Summary 同步LDAP用户和用户组
// @Description 管理员触发LDAP用户和用户组同步
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/ldap/sync [post]
func (c *AdminController) SyncLDAPUsers(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "system", "write", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 检查LDAP是否启用
	config, err := c.ldapService.GetLDAPConfig()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取LDAP配置失败: " + err.Error()})
		return
	}

	if !config.IsEnabled {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "LDAP认证未启用"})
		return
	}

	// 触发同步
	syncID := c.ldapSyncService.TriggerSync()
	
	ctx.JSON(http.StatusOK, gin.H{
		"message": "LDAP同步已启动",
		"sync_id": syncID,
	})
}

// GetLDAPSyncStatus 获取LDAP同步状态（管理员专用）
// @Summary 获取LDAP同步状态
// @Description 管理员获取LDAP同步操作的状态
// @Tags Admin
// @Accept json
// @Produce json
// @Param sync_id path string true "同步任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /admin/ldap/sync/{sync_id}/status [get]
func (c *AdminController) GetLDAPSyncStatus(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "system", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	syncID := ctx.Param("sync_id")
	if syncID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少同步任务ID"})
		return
	}

	status := c.ldapSyncService.GetSyncStatus(syncID)
	if status == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "同步任务不存在"})
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// GetLDAPSyncHistory 获取LDAP同步历史（管理员专用）
// @Summary 获取LDAP同步历史
// @Description 管理员获取最近的LDAP同步历史记录
// @Tags Admin
// @Accept json
// @Produce json
// @Param limit query int false "记录数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Router /admin/ldap/sync/history [get]
func (c *AdminController) GetLDAPSyncHistory(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "system", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	history := c.ldapSyncService.GetSyncHistory(limit)
	
	ctx.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}

// GetProjectsTrash 获取回收站中的项目（管理员专用）
// @Summary 获取回收站项目
// @Description 管理员获取回收站中的所有项目
// @Tags Admin
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Router /admin/projects/trash [get]
func (c *AdminController) GetProjectsTrash(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var projects []models.Project
	var total int64

	// 获取软删除的项目（包含已删除的记录）
	query := c.db.Unscoped().Where("deleted_at IS NOT NULL").Preload("User")

	// 获取总数
	if err := query.Model(&models.Project{}).Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取分页数据
	if err := query.Offset(offset).Limit(limit).Order("deleted_at DESC").Find(&projects).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// RestoreProject 恢复项目（管理员专用）
// @Summary 恢复项目
// @Description 管理员从回收站恢复项目
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]string
// @Router /admin/projects/{id}/restore [patch]
func (c *AdminController) RestoreProject(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "write", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	projectID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	// 恢复项目（取消软删除）
	result := c.db.Unscoped().Model(&models.Project{}).Where("id = ? AND deleted_at IS NOT NULL", projectID).Update("deleted_at", nil)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "项目不存在或未删除"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "项目恢复成功"})
}

// ForceDeleteProject 永久删除项目（管理员专用）
// @Summary 永久删除项目
// @Description 管理员永久删除回收站中的项目
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]string
// @Router /admin/projects/{id}/force-delete [delete]
func (c *AdminController) ForceDeleteProject(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "delete", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	projectID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	// 永久删除项目及其关联数据
	err = c.db.Transaction(func(tx *gorm.DB) error {
		// 删除关联的主机
		if err := tx.Unscoped().Where("project_id = ?", projectID).Delete(&models.Host{}).Error; err != nil {
			return err
		}

		// 删除关联的变量
		if err := tx.Unscoped().Where("project_id = ?", projectID).Delete(&models.Variable{}).Error; err != nil {
			return err
		}

		// 删除关联的任务
		if err := tx.Unscoped().Where("project_id = ?", projectID).Delete(&models.Task{}).Error; err != nil {
			return err
		}

		// 删除项目
		if err := tx.Unscoped().Where("id = ?", projectID).Delete(&models.Project{}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "项目永久删除成功"})
}

// ClearTrash 清空回收站（管理员专用）
// @Summary 清空回收站
// @Description 管理员清空回收站中的所有项目
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/projects/trash/clear [delete]
func (c *AdminController) ClearTrash(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "projects", "delete", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var deletedCount int64

	// 统计要删除的项目数量
	err := c.db.Unscoped().Model(&models.Project{}).Where("deleted_at IS NOT NULL").Count(&deletedCount).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if deletedCount == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "回收站已空",
			"deleted_count": 0,
		})
		return
	}

	// 永久删除所有回收站中的项目
	err = c.db.Transaction(func(tx *gorm.DB) error {
		// 获取所有要删除的项目ID
		var projectIDs []uint
		if err := tx.Unscoped().Model(&models.Project{}).
			Where("deleted_at IS NOT NULL").
			Pluck("id", &projectIDs).Error; err != nil {
			return err
		}

		if len(projectIDs) > 0 {
			// 删除关联的主机
			if err := tx.Unscoped().Where("project_id IN ?", projectIDs).Delete(&models.Host{}).Error; err != nil {
				return err
			}

			// 删除关联的变量
			if err := tx.Unscoped().Where("project_id IN ?", projectIDs).Delete(&models.Variable{}).Error; err != nil {
				return err
			}

			// 删除关联的任务
			if err := tx.Unscoped().Where("project_id IN ?", projectIDs).Delete(&models.Task{}).Error; err != nil {
				return err
			}

			// 删除项目
			if err := tx.Unscoped().Where("id IN ?", projectIDs).Delete(&models.Project{}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "回收站清空成功",
		"deleted_count": deletedCount,
	})
}
