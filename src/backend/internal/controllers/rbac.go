package controllers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RBACController struct {
	db          *gorm.DB
	rbacService *services.RBACService
}

func NewRBACController(db *gorm.DB) *RBACController {
	return &RBACController{
		db:          db,
		rbacService: services.NewRBACService(db),
	}
}

// CheckPermission 检查权限
// @Summary 检查用户权限
// @Description 检查用户是否有权限执行特定操作
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.PermissionCheckRequest true "权限检查请求"
// @Success 200 {object} models.PermissionCheckResponse
// @Router /rbac/check-permission [post]
func (c *RBACController) CheckPermission(ctx *gin.Context) {
	var req models.PermissionCheckRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用中间件函数获取用户ID
	userID, exists := middleware.GetCurrentUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	allowed := c.rbacService.CheckPermission(userID, req.Resource, req.Verb, req.Scope, req.Namespace)

	response := models.PermissionCheckResponse{
		Allowed: allowed,
	}

	if !allowed {
		response.Reason = "权限不足"
	}

	ctx.JSON(http.StatusOK, response)
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建新的角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.CreateRoleRequest true "创建角色请求"
// @Success 201 {object} models.Role
// @Router /rbac/roles [post]
func (c *RBACController) CreateRole(ctx *gin.Context) {
	// 检查权限
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "create", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.CreateRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role, err := c.rbacService.CreateRole(req.Name, req.Description, req.PermissionIDs, false)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, role)
}

// GetRoles 获取角色列表
// @Summary 获取角色列表
// @Description 获取所有角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Success 200 {array} models.Role
// @Router /rbac/roles [get]
func (c *RBACController) GetRoles(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var roles []models.Role
	if err := c.db.Preload("Permissions").Find(&roles).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, roles)
}

// GetRole 获取角色详情
// @Summary 获取角色详情
// @Description 根据ID获取角色详情
// @Tags RBAC
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} models.Role
// @Router /rbac/roles/{id} [get]
func (c *RBACController) GetRole(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := c.db.Preload("Permissions").First(&role, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, role)
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Description 更新角色信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Param request body models.CreateRoleRequest true "更新角色请求"
// @Success 200 {object} models.Role
// @Router /rbac/roles/{id} [put]
func (c *RBACController) UpdateRole(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var req models.CreateRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var role models.Role
	if err := c.db.First(&role, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否是系统角色
	if role.IsSystem {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "系统角色不能修改"})
		return
	}

	// 开始事务
	tx := c.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新角色基本信息
	role.Name = req.Name
	role.Description = req.Description
	if err := tx.Save(&role).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除旧的权限关联
	if err := tx.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 添加新的权限关联
	for _, permissionID := range req.PermissionIDs {
		rolePermission := models.RolePermission{
			RoleID:       role.ID,
			PermissionID: permissionID,
		}
		if err := tx.Create(&rolePermission).Error; err != nil {
			tx.Rollback()
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载角色信息
	if err := c.db.Preload("Permissions").First(&role, role.ID).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, role)
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 删除角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Success 204
// @Router /rbac/roles/{id} [delete]
func (c *RBACController) DeleteRole(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "delete", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := c.db.First(&role, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否是系统角色
	if role.IsSystem {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "系统角色不能删除"})
		return
	}

	if err := c.db.Delete(&role).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CreateUserGroup 创建用户组
// @Summary 创建用户组
// @Description 创建新的用户组
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.CreateUserGroupRequest true "创建用户组请求"
// @Success 201 {object} models.UserGroup
// @Router /rbac/groups [post]
func (c *RBACController) CreateUserGroup(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "groups", "create", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.CreateUserGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userGroup, err := c.rbacService.CreateUserGroup(req.Name, req.Description)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, userGroup)
}

// GetUserGroups 获取用户组列表
// @Summary 获取用户组列表
// @Description 获取所有用户组
// @Tags RBAC
// @Accept json
// @Produce json
// @Success 200 {array} models.UserGroup
// @Router /rbac/groups [get]
func (c *RBACController) GetUserGroups(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "groups", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var userGroups []models.UserGroup
	if err := c.db.Find(&userGroups).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, userGroups)
}

// AssignRole 分配角色
// @Summary 分配角色
// @Description 为用户或用户组分配角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.RoleAssignmentRequest true "角色分配请求"
// @Success 200 {string} string "分配成功"
// @Router /rbac/assign-role [post]
func (c *RBACController) AssignRole(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "assign", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.RoleAssignmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if req.SubjectType == "user" {
		err = c.rbacService.AssignRoleToUser(req.SubjectID, req.RoleID)
	} else if req.SubjectType == "group" {
		err = c.rbacService.AssignRoleToUserGroup(req.SubjectID, req.RoleID)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的主体类型"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "角色分配成功"})
}

// RevokeRole 撤销角色
// @Summary 撤销角色
// @Description 撤销用户或用户组的角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.RoleAssignmentRequest true "角色撤销请求"
// @Success 200 {string} string "撤销成功"
// @Router /rbac/revoke-role [delete]
func (c *RBACController) RevokeRole(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "roles", "revoke", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.RoleAssignmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if req.SubjectType == "user" {
		err = c.rbacService.RevokeRoleFromUser(req.SubjectID, req.RoleID)
	} else if req.SubjectType == "group" {
		err = c.rbacService.RevokeRoleFromUserGroup(req.SubjectID, req.RoleID)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的主体类型"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "角色撤销成功"})
}

// GetPermissions 获取权限列表
// @Summary 获取权限列表
// @Description 获取所有权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Success 200 {array} models.Permission
// @Router /rbac/permissions [get]
func (c *RBACController) GetPermissions(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "permissions", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var permissions []models.Permission
	if err := c.db.Find(&permissions).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, permissions)
}

// CreatePermission 创建权限
// @Summary 创建权限
// @Description 创建新的权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.CreatePermissionRequest true "创建权限请求"
// @Success 201 {object} models.Permission
// @Router /rbac/permissions [post]
func (c *RBACController) CreatePermission(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "permissions", "create", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.CreatePermissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permission, err := c.rbacService.CreatePermission(req.Resource, req.Verb, req.Scope, req.Description)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, permission)
}

// AddUserToGroup 将用户添加到用户组
// @Summary 将用户添加到用户组
// @Description 将用户添加到指定用户组
// @Tags RBAC
// @Accept json
// @Produce json
// @Param group_id path int true "用户组ID"
// @Param user_id path int true "用户ID"
// @Success 200 {string} string "添加成功"
// @Router /rbac/groups/{group_id}/users/{user_id} [post]
func (c *RBACController) AddUserToGroup(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "groups", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	groupID, err := strconv.ParseUint(ctx.Param("group_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户组ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(ctx.Param("user_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	if err := c.rbacService.AddUserToGroup(uint(targetUserID), uint(groupID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "用户添加到用户组成功"})
}

// RemoveUserFromGroup 从用户组中移除用户
// @Summary 从用户组中移除用户
// @Description 从指定用户组中移除用户
// @Tags RBAC
// @Accept json
// @Produce json
// @Param group_id path int true "用户组ID"
// @Param user_id path int true "用户ID"
// @Success 200 {string} string "移除成功"
// @Router /rbac/groups/{group_id}/users/{user_id} [delete]
func (c *RBACController) RemoveUserFromGroup(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "groups", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	groupID, err := strconv.ParseUint(ctx.Param("group_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户组ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(ctx.Param("user_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	if err := c.rbacService.RemoveUserFromGroup(uint(targetUserID), uint(groupID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "用户从用户组移除成功"})
}

// GetUserPermissions 获取用户权限
// @Summary 获取用户权限
// @Description 获取指定用户的所有权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {array} models.Permission
// @Router /rbac/users/{user_id}/permissions [get]
func (c *RBACController) GetUserPermissions(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "users", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	targetUserID, err := strconv.ParseUint(ctx.Param("user_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	permissions, err := c.rbacService.GetUserPermissions(uint(targetUserID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, permissions)
}

// ==================== 角色模板管理 ====================

// ListRoleTemplates 获取角色模板列表
// @Summary 获取角色模板列表
// @Description 获取所有角色模板
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Param active_only query bool false "只获取启用的模板"
// @Success 200 {array} models.RoleTemplate
// @Router /rbac/role-templates [get]
func (c *RBACController) ListRoleTemplates(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "role_templates", "list", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	activeOnly := ctx.Query("active_only") == "true"
	templates, err := c.rbacService.ListRoleTemplates(activeOnly)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, templates)
}

// GetRoleTemplate 获取角色模板详情
// @Summary 获取角色模板详情
// @Description 根据ID获取角色模板详情
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Param id path int true "角色模板ID"
// @Success 200 {object} models.RoleTemplate
// @Router /rbac/role-templates/{id} [get]
func (c *RBACController) GetRoleTemplate(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "role_templates", "read", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色模板ID"})
		return
	}

	template, err := c.rbacService.GetRoleTemplateByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, template)
}

// CreateRoleTemplate 创建角色模板
// @Summary 创建角色模板
// @Description 创建新的角色模板
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Param request body models.CreateRoleTemplateRequest true "创建角色模板请求"
// @Success 201 {object} models.RoleTemplate
// @Router /rbac/role-templates [post]
func (c *RBACController) CreateRoleTemplate(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "role_templates", "create", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	var req models.CreateRoleTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := c.rbacService.CreateRoleTemplateFromRequest(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, template)
}

// UpdateRoleTemplate 更新角色模板
// @Summary 更新角色模板
// @Description 更新角色模板信息
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Param id path int true "角色模板ID"
// @Param request body models.UpdateRoleTemplateRequest true "更新角色模板请求"
// @Success 200 {object} models.RoleTemplate
// @Router /rbac/role-templates/{id} [put]
func (c *RBACController) UpdateRoleTemplate(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "role_templates", "update", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色模板ID"})
		return
	}

	var req models.UpdateRoleTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := c.rbacService.UpdateRoleTemplate(uint(id), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, template)
}

// DeleteRoleTemplate 删除角色模板
// @Summary 删除角色模板
// @Description 删除角色模板
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Param id path int true "角色模板ID"
// @Success 204
// @Router /rbac/role-templates/{id} [delete]
func (c *RBACController) DeleteRoleTemplate(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "role_templates", "delete", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色模板ID"})
		return
	}

	if err := c.rbacService.DeleteRoleTemplate(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetAvailableResources 获取可配置的资源列表
// @Summary 获取可配置的资源列表
// @Description 获取可用于配置权限的资源列表
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Router /rbac/resources [get]
func (c *RBACController) GetAvailableResources(ctx *gin.Context) {
	resources := c.rbacService.GetAvailableResources()
	ctx.JSON(http.StatusOK, resources)
}

// GetAvailableVerbs 获取可配置的操作列表
// @Summary 获取可配置的操作列表
// @Description 获取可用于配置权限的操作动词列表
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Router /rbac/verbs [get]
func (c *RBACController) GetAvailableVerbs(ctx *gin.Context) {
	verbs := c.rbacService.GetAvailableVerbs()
	ctx.JSON(http.StatusOK, verbs)
}

// SyncRoleTemplates 同步角色模板到角色
// @Summary 同步角色模板到角色
// @Description 将角色模板同步为实际角色
// @Tags RBAC - Role Templates
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /rbac/role-templates/sync [post]
func (c *RBACController) SyncRoleTemplates(ctx *gin.Context) {
	userID := ctx.GetUint("user_id")
	if !c.rbacService.CheckPermission(userID, "role_templates", "admin", "*", "") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 重新初始化 RBAC 会同步模板
	if err := c.rbacService.InitializeDefaultRBAC(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "角色模板同步成功"})
}
