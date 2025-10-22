package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EnhancedUserController struct {
	db *gorm.DB
}

func NewEnhancedUserController(db *gorm.DB) *EnhancedUserController {
	return &EnhancedUserController{db: db}
}

// GetUsers 获取增强用户列表
func (euc *EnhancedUserController) GetUsers(c *gin.Context) {
	var users []models.User
	var total int64

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 搜索参数
	search := c.Query("search")
	role := c.Query("role")
	status := c.Query("status")

	query := euc.db.Model(&models.User{})

	// 构建查询条件
	if search != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if status != "" {
		query = query.Where("is_active = ?", status == "active")
	}

	// 获取总数
	query.Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	// 统计信息
	var activeCount, inactiveCount int64
	euc.db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeCount)
	euc.db.Model(&models.User{}).Where("is_active = ?", false).Count(&inactiveCount)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
		"statistics": gin.H{
			"total_users":    total,
			"active_users":   activeCount,
			"inactive_users": inactiveCount,
		},
	})
}

// CreateUser 创建增强用户
func (euc *EnhancedUserController) CreateUser(c *gin.Context) {
	var req struct {
		Username      string                 `json:"username" binding:"required"`
		Email         string                 `json:"email" binding:"required,email"`
		Name          string                 `json:"name"`
		Password      string                 `json:"password,omitempty"`
		Role          string                 `json:"role"`
		IsActive      bool                   `json:"is_active"`
		AuthSource    string                 `json:"auth_source"`
		Permissions   map[string]interface{} `json:"permissions"`
		Groups        []uint                 `json:"groups"`
		DashboardRole string                 `json:"dashboard_role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名和邮箱是否已存在
	var existingUser models.User
	if err := euc.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名或邮箱已存在"})
		return
	}

	// 创建用户
	user := models.User{
		Username:      req.Username,
		Email:         req.Email,
		Name:          req.Name,
		IsActive:      req.IsActive,
		AuthSource:    req.AuthSource,
		DashboardRole: req.DashboardRole,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 如果提供了密码，进行哈希处理
	if req.Password != "" {
		// 这里应该使用密码哈希算法，比如 bcrypt
		// 示例：user.Password = hashPassword(req.Password)
		user.Password = req.Password // 临时直接存储，实际应该哈希
	}

	if err := euc.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "用户创建成功",
		"user":    user,
	})
}

// ResetPassword 重置用户密码
func (euc *EnhancedUserController) ResetPassword(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		NewPassword string `json:"new_password" binding:"required"`
		SendEmail   bool   `json:"send_email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := euc.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		}
		return
	}

	// 更新密码（应该进行哈希处理）
	user.Password = req.NewPassword // 实际应该使用哈希
	user.UpdatedAt = time.Now()

	if err := euc.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	// 如果需要发送邮件通知
	if req.SendEmail {
		// 这里应该实现邮件发送逻辑
		// sendPasswordResetEmail(user.Email, req.NewPassword)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码重置成功",
		"user_id": user.ID,
	})
}

// GetUserGroups 获取用户组列表
func (euc *EnhancedUserController) GetUserGroups(c *gin.Context) {
	var groups []models.UserGroup

	err := euc.db.Preload("Users").Order("created_at DESC").Find(&groups).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户组失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"total":  len(groups),
	})
}

// CreateUserGroup 创建用户组
func (euc *EnhancedUserController) CreateUserGroup(c *gin.Context) {
	var req struct {
		Name              string                 `json:"name" binding:"required"`
		Description       string                 `json:"description"`
		Permissions       map[string]interface{} `json:"permissions"`
		DashboardTemplate string                 `json:"dashboard_template"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查组名是否已存在
	var existingGroup models.UserGroup
	if err := euc.db.Where("name = ?", req.Name).First(&existingGroup).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户组名已存在"})
		return
	}

	group := models.UserGroup{
		Name:              req.Name,
		Description:       req.Description,
		DashboardTemplate: req.DashboardTemplate,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := euc.db.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户组失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "用户组创建成功",
		"group":   group,
	})
}

// UpdateUserGroup 更新用户组
func (euc *EnhancedUserController) UpdateUserGroup(c *gin.Context) {
	groupID := c.Param("id")

	var group models.UserGroup
	if err := euc.db.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户组不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户组失败"})
		}
		return
	}

	var req struct {
		Name              string                 `json:"name"`
		Description       string                 `json:"description"`
		Permissions       map[string]interface{} `json:"permissions"`
		DashboardTemplate string                 `json:"dashboard_template"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新字段
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if req.DashboardTemplate != "" {
		group.DashboardTemplate = req.DashboardTemplate
	}
	group.UpdatedAt = time.Now()

	if err := euc.db.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户组失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户组更新成功",
		"group":   group,
	})
}

// DeleteUserGroup 删除用户组
func (euc *EnhancedUserController) DeleteUserGroup(c *gin.Context) {
	groupID := c.Param("id")

	var group models.UserGroup
	if err := euc.db.Preload("Users").First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户组不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户组失败"})
		}
		return
	}

	// 检查是否有用户属于该组（通过多对多关系）
	if len(group.Users) > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户组下还有用户，无法删除"})
		return
	}

	if err := euc.db.Delete(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户组失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户组删除成功"})
}

// AddUserToGroup 将用户添加到组
func (euc *EnhancedUserController) AddUserToGroup(c *gin.Context) {
	groupID := c.Param("groupId")
	userID := c.Param("userId")

	// 检查用户和组是否存在
	var user models.User
	var group models.UserGroup

	if err := euc.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if err := euc.db.First(&group, groupID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户组不存在"})
		return
	}

	// 使用多对多关系添加用户到组
	if err := euc.db.Model(&user).Association("UserGroups").Append(&group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加用户到组失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "用户已成功添加到组",
		"user_id":  user.ID,
		"group_id": group.ID,
	})
}

// RemoveUserFromGroup 从组中移除用户
func (euc *EnhancedUserController) RemoveUserFromGroup(c *gin.Context) {
	groupID := c.Param("groupId")
	userID := c.Param("userId")

	var user models.User
	var group models.UserGroup

	if err := euc.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if err := euc.db.First(&group, groupID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户组不存在"})
		return
	}

	// 使用多对多关系移除用户的组关联
	if err := euc.db.Model(&user).Association("UserGroups").Delete(&group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "用户已从组中移除",
		"user_id":  user.ID,
		"group_id": groupID,
	})
}

// GetRoles 获取角色列表
func (euc *EnhancedUserController) GetRoles(c *gin.Context) {
	// 静态角色列表，也可以从数据库获取
	roles := []gin.H{
		{"id": "admin", "name": "管理员", "description": "系统管理员，拥有所有权限"},
		{"id": "developer", "name": "开发者", "description": "开发者角色，可以管理项目和资源"},
		{"id": "researcher", "name": "研究员", "description": "研究员角色，可以访问研究相关功能"},
		{"id": "user", "name": "普通用户", "description": "普通用户，基础访问权限"},
		{"id": "guest", "name": "访客", "description": "访客用户，只读权限"},
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"total": len(roles),
	})
}

// 辅助函数
func parseUint(s string) uint {
	val, _ := strconv.ParseUint(s, 10, 32)
	return uint(val)
}
