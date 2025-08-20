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
