package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardController struct {
	db *gorm.DB
}

func NewDashboardController(db *gorm.DB) *DashboardController {
	return &DashboardController{db: db}
}

// GetUserDashboard 获取用户仪表板配置
func (dc *DashboardController) GetUserDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var dashboard models.Dashboard
	err := dc.db.Where("user_id = ?", userID).First(&dashboard).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			defaultConfig := models.DashboardConfig{
				Widgets: []models.DashboardWidget{
					{
						ID:       "widget-1",
						Type:     "JUPYTERHUB",
						Title:    "JupyterHub",
						URL:      "/jupyter",
						Size:     models.DashboardSize{Width: 12, Height: 600},
						Position: 0,
						Visible:  true,
						Settings: make(map[string]interface{}),
					},
					{
						ID:       "widget-2",
						Type:     "GITEA",
						Title:    "Gitea",
						URL:      "/gitea",
						Size:     models.DashboardSize{Width: 12, Height: 600},
						Position: 1,
						Visible:  true,
						Settings: make(map[string]interface{}),
					},
				},
			}
			c.JSON(http.StatusOK, gin.H{"widgets": defaultConfig.Widgets})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取仪表板配置失败"})
		return
	}

	var config models.DashboardConfig
	if err := json.Unmarshal([]byte(dashboard.Config), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析仪表板配置失败"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateDashboard 更新用户仪表板配置
func (dc *DashboardController) UpdateDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var req models.DashboardUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	config := models.DashboardConfig{
		Widgets: req.Widgets,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "序列化配置失败"})
		return
	}

	// 查找或创建仪表板配置
	var dashboard models.Dashboard
	err = dc.db.Where("user_id = ?", userID).First(&dashboard).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		dashboard = models.Dashboard{
			UserID: userID.(uint),
			Config: string(configJSON),
		}
		if err := dc.db.Create(&dashboard).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建仪表板配置失败"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询仪表板配置失败"})
		return
	} else {
		// 更新已有配置
		dashboard.Config = string(configJSON)
		if err := dc.db.Save(&dashboard).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新仪表板配置失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "仪表板配置更新成功"})
}

// ResetDashboard 重置用户仪表板配置
func (dc *DashboardController) ResetDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 删除用户的仪表板配置
	if err := dc.db.Where("user_id = ?", userID).Delete(&models.Dashboard{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置仪表板配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "仪表板配置已重置"})
}

// 增强用户管理控制器
type EnhancedUserController struct {
	db *gorm.DB
}

func NewEnhancedUserController(db *gorm.DB) *EnhancedUserController {
	return &EnhancedUserController{db: db}
}

// GetUsers 获取用户列表（支持分页和过滤）
func (uc *EnhancedUserController) GetUsers(c *gin.Context) {
	var users []models.User
	
	// 构建查询
	query := uc.db.Preload("Roles").Preload("UserGroups")
	
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	
	// 搜索过滤
	if search := c.Query("search"); search != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	
	// 角色过滤
	if role := c.Query("role"); role != "" {
		query = query.Joins("JOIN user_roles ON users.id = user_roles.user_id").
			Joins("JOIN roles ON user_roles.role_id = roles.id").
			Where("roles.name = ?", role)
	}
	
	// 认证来源过滤
	if authSource := c.Query("auth_source"); authSource != "" {
		query = query.Where("auth_source = ?", authSource)
	}
	
	// 状态过滤
	if isActive := c.Query("is_active"); isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}
	
	// 执行查询
	var total int64
	query.Model(&models.User{}).Count(&total)
	
	if err := query.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// CreateUser 创建用户（支持LDAP和本地）
func (uc *EnhancedUserController) CreateUser(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	// 检查用户是否已存在
	var existingUser models.User
	if err := uc.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名或邮箱已存在"})
		return
	}

	// 创建用户
	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		AuthSource: "local", // 默认本地认证
		IsActive:   true,
	}

	// 设置密码（仅本地用户）
	if req.Password != "" {
		// 这里应该使用密码哈希
		user.Password = req.Password // 注意：实际应用中需要哈希处理
	}

	if err := uc.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	// 分配默认角色
	if req.Role != "" {
		var role models.Role
		if err := uc.db.Where("name = ?", req.Role).First(&role).Error; err == nil {
			uc.db.Model(&user).Association("Roles").Append(&role)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "用户创建成功", "user": user})
}

// GetUserGroups 获取用户组列表
func (uc *EnhancedUserController) GetUserGroups(c *gin.Context) {
	var groups []models.UserGroup
	
	if err := uc.db.Preload("Users").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户组列表失败"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

// CreateUserGroup 创建用户组
func (uc *EnhancedUserController) CreateUserGroup(c *gin.Context) {
	var group models.UserGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	if err := uc.db.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户组失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "用户组创建成功", "group": group})
}

// GetRoles 获取角色列表
func (uc *EnhancedUserController) GetRoles(c *gin.Context) {
	var roles []models.Role
	
	if err := uc.db.Preload("Permissions").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取角色列表失败"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

// ResetPassword 重置用户密码
func (uc *EnhancedUserController) ResetPassword(c *gin.Context) {
	userID := c.Param("id")
	
	var user models.User
	if err := uc.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 生成随机密码
	newPassword := "TempPass123!" // 这里应该生成随机密码
	user.Password = newPassword   // 注意：实际应用中需要哈希处理

	if err := uc.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "密码重置成功",
		"password": newPassword,
	})
}
