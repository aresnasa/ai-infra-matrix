package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

// NavigationController 导航配置控制器
type NavigationController struct{}

// NewNavigationController 创建新的导航控制器
func NewNavigationController() *NavigationController {
	return &NavigationController{}
}

// NavigationItem 导航项结构
type NavigationItem struct {
	ID      string   `json:"id"`
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Icon    string   `json:"icon"`
	Visible bool     `json:"visible"`
	Order   int      `json:"order"`
	Roles   []string `json:"roles"`
}

// NavigationConfig 导航配置请求结构
type NavigationConfig struct {
	Items []NavigationItem `json:"items"`
}

// DefaultNavigationItems 默认导航项
var DefaultNavigationItems = []NavigationItem{
	{
		ID:      "projects",
		Key:     "/projects",
		Label:   "项目管理",
		Icon:    "ProjectOutlined",
		Visible: true,
		Order:   0,
		Roles:   []string{"user", "admin", "super-admin"},
	},
	{
		ID:      "gitea",
		Key:     "/gitea",
		Label:   "Gitea",
		Icon:    "CodeOutlined",
		Visible: true,
		Order:   1,
		Roles:   []string{"user", "admin", "super-admin"},
	},
	{
		ID:      "kubernetes",
		Key:     "/kubernetes",
		Label:   "Kubernetes",
		Icon:    "CloudServerOutlined",
		Visible: true,
		Order:   2,
		Roles:   []string{"admin", "super-admin"},
	},
	{
		ID:      "ansible",
		Key:     "/ansible",
		Label:   "Ansible",
		Icon:    "FileTextOutlined",
		Visible: true,
		Order:   3,
		Roles:   []string{"admin", "super-admin"},
	},
	{
		ID:      "jupyterhub",
		Key:     "/jupyterhub",
		Label:   "JupyterHub",
		Icon:    "ExperimentTwoTone",
		Visible: true,
		Order:   4,
		Roles:   []string{"user", "admin", "super-admin"},
	},
	{
		ID:      "slurm",
		Key:     "/slurm",
		Label:   "Slurm",
		Icon:    "ClusterOutlined",
		Visible: true,
		Order:   5,
		Roles:   []string{"admin", "super-admin"},
	},
	{
		ID:      "saltstack",
		Key:     "/saltstack",
		Label:   "SaltStack",
		Icon:    "ControlOutlined",
		Visible: true,
		Order:   6,
		Roles:   []string{"admin", "super-admin"},
	},
}

// GetNavigationConfig 获取用户导航配置
func (nc *NavigationController) GetNavigationConfig(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查询用户的导航配置
	var navConfig models.UserNavigationConfig
	result := database.DB.Where("user_id = ?", userID).First(&navConfig)
	
	if result.Error != nil {
		// 如果没有找到配置，返回默认配置
		c.JSON(http.StatusOK, gin.H{"data": DefaultNavigationItems})
		return
	}

	// 解析JSON配置
	var items []NavigationItem
	if err := json.Unmarshal([]byte(navConfig.Config), &items); err != nil {
		c.JSON(http.StatusOK, gin.H{"data": DefaultNavigationItems})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

// SaveNavigationConfig 保存用户导航配置
func (nc *NavigationController) SaveNavigationConfig(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 解析请求体
	var req NavigationConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	// 验证items不为空
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导航配置不能为空"})
		return
	}

	// 将配置转换为JSON
	configJSON, err := json.Marshal(req.Items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置序列化失败: " + err.Error()})
		return
	}

	// 保存或更新用户配置
	var navConfig models.UserNavigationConfig
	result := database.DB.Where("user_id = ?", userID).First(&navConfig)
	
	if result.Error != nil {
		// 创建新配置
		navConfig = models.UserNavigationConfig{
			UserID: userID.(uint),
			Config: string(configJSON),
		}
		if err := database.DB.Create(&navConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
			return
		}
	} else {
		// 更新现有配置
		navConfig.Config = string(configJSON)
		if err := database.DB.Save(&navConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "导航配置已保存", "saved_items": len(req.Items)})
}

// ResetNavigationConfig 重置导航配置为默认值
func (nc *NavigationController) ResetNavigationConfig(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 删除用户的自定义配置
	if err := database.DB.Where("user_id = ?", userID).Delete(&models.UserNavigationConfig{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "导航配置已重置为默认值"})
}

// GetDefaultNavigationConfig 获取默认导航配置
func (nc *NavigationController) GetDefaultNavigationConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": DefaultNavigationItems})
}
