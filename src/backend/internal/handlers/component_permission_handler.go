package handlers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ComponentPermissionHandler 组件权限处理器
type ComponentPermissionHandler struct {
	componentService services.ComponentRegistrationService
}

// NewComponentPermissionHandler 创建组件权限处理器
func NewComponentPermissionHandler(componentService services.ComponentRegistrationService) *ComponentPermissionHandler {
	return &ComponentPermissionHandler{
		componentService: componentService,
	}
}

// GetUserComponentPermissions 获取用户的组件权限
// @Summary 获取用户组件权限
// @Description 获取指定用户在各组件上的权限配置
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {object} models.Response{data=[]models.UserComponentPermission}
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/users/{user_id}/component-permissions [get]
func (h *ComponentPermissionHandler) GetUserComponentPermissions(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	permissions, err := h.componentService.GetUserComponentPermissions(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取组件权限失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    permissions,
	})
}

// GetComponentSyncStatus 获取组件同步状态
// @Summary 获取组件同步状态
// @Description 获取指定用户在各组件上的同步状态
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {object} models.Response{data=[]models.ComponentSyncStatus}
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/users/{user_id}/component-sync-status [get]
func (h *ComponentPermissionHandler) GetComponentSyncStatus(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	statuses, err := h.componentService.GetComponentSyncStatus(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取同步状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    statuses,
	})
}

// UpdateUserComponentPermission 更新用户的组件权限
// @Summary 更新用户组件权限
// @Description 更新指定用户在指定组件上的权限配置
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Param component path string true "组件名称"
// @Param request body models.UpdateComponentPermissionRequest true "权限配置"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/users/{user_id}/component-permissions/{component} [put]
func (h *ComponentPermissionHandler) UpdateUserComponentPermission(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	component := c.Param("component")
	if component == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "组件名称不能为空",
		})
		return
	}

	var req models.UpdateComponentPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.componentService.UpdateUserComponentPermission(uint(userID), component, req); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "更新组件权限失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "组件权限更新成功",
	})
}

// RegisterUserToComponents 注册用户到组件
// @Summary 注册用户到组件
// @Description 将用户注册到指定的组件
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param request body models.ComponentRegistrationRequest true "注册配置"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/component-registration [post]
func (h *ComponentPermissionHandler) RegisterUserToComponents(c *gin.Context) {
	var req models.ComponentRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取用户信息
	var user models.User
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "用户不存在",
		})
		return
	}

	// 注册到组件
	if err := h.componentService.RegisterUserToComponents(user, "", req.Components); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "注册组件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "用户已注册到组件",
	})
}

// SyncUserToComponent 同步用户到单个组件
// @Summary 同步用户到组件
// @Description 将用户同步到指定的组件（用于重试失败的同步）
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Param component path string true "组件名称"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/users/{user_id}/component-sync/{component} [post]
func (h *ComponentPermissionHandler) SyncUserToComponent(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	component := c.Param("component")
	if component == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "组件名称不能为空",
		})
		return
	}

	// 获取用户信息
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "用户不存在",
		})
		return
	}

	// 获取用户的组件权限配置
	permissions, err := h.componentService.GetUserComponentPermissions(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取权限配置失败: " + err.Error(),
		})
		return
	}

	// 找到指定组件的权限配置
	var targetPermission *models.UserComponentPermission
	for _, p := range permissions {
		if p.Component == component {
			targetPermission = &p
			break
		}
	}

	if targetPermission == nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "用户未配置该组件权限",
		})
		return
	}

	// 同步到组件
	if err := h.componentService.SyncUserToComponent(user, "", component, *targetPermission); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "同步组件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "同步成功",
	})
}

// DeleteUserFromComponent 从组件删除用户
// @Summary 从组件删除用户
// @Description 将用户从指定组件中删除
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Param component path string true "组件名称"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/users/{user_id}/component-permissions/{component} [delete]
func (h *ComponentPermissionHandler) DeleteUserFromComponent(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	component := c.Param("component")
	if component == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "组件名称不能为空",
		})
		return
	}

	if err := h.componentService.DeleteUserFromComponent(uint(userID), component); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "用户已从组件删除",
	})
}

// GetRoleTemplateComponentPermissions 获取角色模板的组件权限配置
// @Summary 获取角色模板组件权限
// @Description 获取指定角色模板的组件权限配置
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param role_template path string true "角色模板名称"
// @Success 200 {object} models.Response{data=[]models.RoleTemplateComponentPermission}
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/role-templates/{role_template}/component-permissions [get]
func (h *ComponentPermissionHandler) GetRoleTemplateComponentPermissions(c *gin.Context) {
	roleTemplate := c.Param("role_template")
	if roleTemplate == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "角色模板名称不能为空",
		})
		return
	}

	permissions, err := h.componentService.GetRoleTemplateComponentPermissions(roleTemplate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取角色模板组件权限失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    permissions,
	})
}

// SetRoleTemplateComponentPermissions 设置角色模板的组件权限配置
// @Summary 设置角色模板组件权限
// @Description 设置指定角色模板的组件权限配置
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param role_template_id path int true "角色模板ID"
// @Param request body []models.RoleTemplateComponentPermission true "权限配置列表"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/role-templates/{role_template_id}/component-permissions [put]
func (h *ComponentPermissionHandler) SetRoleTemplateComponentPermissions(c *gin.Context) {
	roleTemplateIDStr := c.Param("role_template_id")
	roleTemplateID, err := strconv.ParseUint(roleTemplateIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的角色模板ID",
		})
		return
	}

	var permissions []models.RoleTemplateComponentPermission
	if err := c.ShouldBindJSON(&permissions); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := h.componentService.SetRoleTemplateComponentPermissions(uint(roleTemplateID), permissions); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "设置角色模板组件权限失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "角色模板组件权限设置成功",
	})
}

// RetryFailedSyncs 重试失败的同步任务
// @Summary 重试失败同步
// @Description 重试所有失败的组件同步任务
// @Tags 组件权限
// @Accept json
// @Produce json
// @Success 200 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/component-sync/retry-failed [post]
func (h *ComponentPermissionHandler) RetryFailedSyncs(c *gin.Context) {
	if err := h.componentService.RetryFailedSyncs(); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "重试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "重试任务已启动",
	})
}

// ListAvailableComponents 列出可用的组件
// @Summary 列出可用组件
// @Description 列出所有可用于权限配置的组件
// @Tags 组件权限
// @Accept json
// @Produce json
// @Success 200 {object} models.Response
// @Router /api/v1/components [get]
func (h *ComponentPermissionHandler) ListAvailableComponents(c *gin.Context) {
	components := []map[string]interface{}{
		{
			"name":         models.ComponentNightingale,
			"display_name": "Nightingale 监控",
			"description":  "统一监控告警平台",
			"icon":         "monitor",
		},
		{
			"name":         models.ComponentGitea,
			"display_name": "Gitea 代码仓库",
			"description":  "Git 代码托管平台",
			"icon":         "git",
		},
		{
			"name":         models.ComponentSeaweedFS,
			"display_name": "SeaweedFS 对象存储",
			"description":  "分布式对象存储系统",
			"icon":         "storage",
		},
		{
			"name":         models.ComponentJupyterHub,
			"display_name": "JupyterHub",
			"description":  "多用户 Jupyter 环境",
			"icon":         "jupyter",
		},
		{
			"name":         models.ComponentSlurm,
			"display_name": "SLURM 计算集群",
			"description":  "HPC 作业调度系统",
			"icon":         "cluster",
		},
		{
			"name":         models.ComponentKeycloak,
			"display_name": "Keycloak IAM",
			"description":  "身份认证管理",
			"icon":         "security",
		},
	}

	permissionLevels := []map[string]string{
		{"value": models.PermissionLevelNone, "label": "无权限"},
		{"value": models.PermissionLevelReadonly, "label": "只读"},
		{"value": models.PermissionLevelUser, "label": "普通用户"},
		{"value": models.PermissionLevelAdmin, "label": "管理员"},
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data: map[string]interface{}{
			"components":        components,
			"permission_levels": permissionLevels,
		},
	})
}

// BulkRegisterUsersToComponents 批量注册用户到组件
// @Summary 批量注册用户到组件
// @Description 将多个用户批量注册到指定组件
// @Tags 组件权限
// @Accept json
// @Produce json
// @Param request body BulkRegistrationRequest true "批量注册配置"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/component-registration/bulk [post]
func (h *ComponentPermissionHandler) BulkRegisterUsersToComponents(c *gin.Context) {
	var req BulkRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	results := make([]models.UserComponentSyncResult, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			results = append(results, models.UserComponentSyncResult{
				UserID:  userID,
				Success: false,
				Message: "用户不存在",
			})
			continue
		}

		err := h.componentService.RegisterUserToComponents(user, "", req.Components)
		result := models.UserComponentSyncResult{
			UserID:   userID,
			Username: user.Username,
			Success:  err == nil,
		}
		if err != nil {
			result.Message = err.Error()
		}

		// 获取同步状态
		statuses, _ := h.componentService.GetComponentSyncStatus(userID)
		result.Components = statuses

		results = append(results, result)
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "批量注册完成",
		Data:    results,
	})
}

// BulkRegistrationRequest 批量注册请求
type BulkRegistrationRequest struct {
	UserIDs    []uint                             `json:"user_ids" binding:"required"`
	Components []models.ComponentPermissionConfig `json:"components" binding:"required"`
}

// RegisterHandlers 注册路由
func (h *ComponentPermissionHandler) RegisterHandlers(r *gin.RouterGroup) {
	// 组件信息
	r.GET("/components", h.ListAvailableComponents)

	// 用户组件权限
	r.GET("/users/:user_id/component-permissions", h.GetUserComponentPermissions)
	r.PUT("/users/:user_id/component-permissions/:component", h.UpdateUserComponentPermission)
	r.DELETE("/users/:user_id/component-permissions/:component", h.DeleteUserFromComponent)

	// 同步状态
	r.GET("/users/:user_id/component-sync-status", h.GetComponentSyncStatus)
	r.POST("/users/:user_id/component-sync/:component", h.SyncUserToComponent)

	// 组件注册
	r.POST("/component-registration", h.RegisterUserToComponents)
	r.POST("/component-registration/bulk", h.BulkRegisterUsersToComponents)
	r.POST("/component-sync/retry-failed", h.RetryFailedSyncs)

	// 角色模板组件权限
	r.GET("/role-templates/:role_template/component-permissions", h.GetRoleTemplateComponentPermissions)
	r.PUT("/role-templates/:role_template_id/component-permissions", h.SetRoleTemplateComponentPermissions)

	logrus.Info("Component permission handlers registered")
}
