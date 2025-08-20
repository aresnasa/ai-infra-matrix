package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EnhancedDashboardController struct {
	db *gorm.DB
}

func NewEnhancedDashboardController(db *gorm.DB) *EnhancedDashboardController {
	return &EnhancedDashboardController{db: db}
}

// DashboardStatsResponse 仪表板统计响应
type DashboardStatsResponse struct {
	TotalWidgets     int                    `json:"total_widgets"`
	ActiveWidgets    int                    `json:"active_widgets"`
	WidgetTypes      map[string]int         `json:"widget_types"`
	WidgetCategories map[string]int         `json:"widget_categories"`
	UserDashboards   int                    `json:"user_dashboards"`
	PopularWidgets   []PopularWidget        `json:"popular_widgets"`
	RecentActivity   []DashboardActivity    `json:"recent_activity"`
	UsageStats       DashboardUsageStats    `json:"usage_stats"`
}

type PopularWidget struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Count       int    `json:"count"`
	Icon        string `json:"icon"`
}

type DashboardActivity struct {
	UserID      uint      `json:"user_id"`
	Username    string    `json:"username"`
	Action      string    `json:"action"`
	WidgetType  string    `json:"widget_type,omitempty"`
	WidgetTitle string    `json:"widget_title,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

type DashboardUsageStats struct {
	TotalUsers        int                   `json:"total_users"`
	ActiveUsers       int                   `json:"active_users"`
	AvgWidgetsPerUser float64               `json:"avg_widgets_per_user"`
	TopUsers          []UserUsageInfo       `json:"top_users"`
	WidgetCategories  map[string]int        `json:"widget_categories"`
}

type UserUsageInfo struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	WidgetCount  int    `json:"widget_count"`
	LastActivity time.Time `json:"last_activity"`
}

// DashboardTemplate 仪表板模板
type DashboardTemplate struct {
	ID          uint                     `json:"id" gorm:"primaryKey"`
	Name        string                   `json:"name" gorm:"not null"`
	Description string                   `json:"description"`
	Category    string                   `json:"category"` // developer, admin, researcher, custom
	IsPublic    bool                     `json:"is_public" gorm:"default:false"`
	CreatedBy   uint                     `json:"created_by"`
	Creator     models.User              `json:"creator" gorm:"foreignKey:CreatedBy"`
	Config      string                   `json:"config" gorm:"type:text"` // JSON配置
	UsageCount  int                      `json:"usage_count" gorm:"default:0"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// GetDashboardStats 获取仪表板统计信息
func (edc *EnhancedDashboardController) GetDashboardStats(c *gin.Context) {
	var stats DashboardStatsResponse
	
	// 计算总widget数量和活跃widget数量
	var dashboards []models.Dashboard
	if err := edc.db.Find(&dashboards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取仪表板数据失败"})
		return
	}

	stats.UserDashboards = len(dashboards)
	stats.WidgetTypes = make(map[string]int)
	stats.WidgetCategories = make(map[string]int)
	
	totalWidgets := 0
	activeWidgets := 0
	
	// Widget类型映射到类别
	widgetCategories := map[string]string{
		"JUPYTERHUB":  "development",
		"GITEA":       "development", 
		"KUBERNETES":  "infrastructure",
		"ANSIBLE":     "automation",
		"SLURM":       "compute",
		"SALTSTACK":   "infrastructure",
		"MONITORING":  "monitoring",
		"CUSTOM":      "custom",
	}

	for _, dashboard := range dashboards {
		var config models.DashboardConfig
		if err := json.Unmarshal([]byte(dashboard.Config), &config); err != nil {
			continue
		}
		
		for _, widget := range config.Widgets {
			totalWidgets++
			stats.WidgetTypes[widget.Type]++
			
			if category, exists := widgetCategories[widget.Type]; exists {
				stats.WidgetCategories[category]++
			}
			
			if widget.Visible {
				activeWidgets++
			}
		}
	}
	
	stats.TotalWidgets = totalWidgets
	stats.ActiveWidgets = activeWidgets

	// 计算热门widget
	stats.PopularWidgets = edc.getPopularWidgets(stats.WidgetTypes)

	// 获取最近活动
	stats.RecentActivity = edc.getRecentActivity()

	// 计算使用统计
	stats.UsageStats = edc.getUsageStats(dashboards)

	c.JSON(http.StatusOK, stats)
}

// getPopularWidgets 获取热门widget
func (edc *EnhancedDashboardController) getPopularWidgets(widgetTypes map[string]int) []PopularWidget {
	// Widget类型到显示信息的映射
	widgetInfo := map[string]map[string]string{
		"JUPYTERHUB":  {"name": "JupyterHub", "icon": "🚀"},
		"GITEA":       {"name": "Gitea", "icon": "📚"},
		"KUBERNETES":  {"name": "Kubernetes", "icon": "☸️"},
		"ANSIBLE":     {"name": "Ansible", "icon": "🔧"},
		"SLURM":       {"name": "Slurm", "icon": "🖥️"},
		"SALTSTACK":   {"name": "SaltStack", "icon": "⚡"},
		"MONITORING":  {"name": "监控面板", "icon": "📊"},
		"CUSTOM":      {"name": "自定义", "icon": "🔗"},
	}

	var popular []PopularWidget
	for widgetType, count := range widgetTypes {
		info := widgetInfo[widgetType]
		if info == nil {
			info = map[string]string{"name": widgetType, "icon": "🔲"}
		}
		
		popular = append(popular, PopularWidget{
			Type:  widgetType,
			Name:  info["name"],
			Count: count,
			Icon:  info["icon"],
		})
	}

	// 按使用次数排序
	for i := 0; i < len(popular)-1; i++ {
		for j := i + 1; j < len(popular); j++ {
			if popular[i].Count < popular[j].Count {
				popular[i], popular[j] = popular[j], popular[i]
			}
		}
	}

	// 只返回前5个
	if len(popular) > 5 {
		popular = popular[:5]
	}

	return popular
}

// getRecentActivity 获取最近活动
func (edc *EnhancedDashboardController) getRecentActivity() []DashboardActivity {
	// 这里应该从活动日志表获取，简化起见直接返回模拟数据
	// 在实际实现中，应该有一个单独的活动日志表来记录用户操作
	var activities []DashboardActivity
	
	var recentDashboards []models.Dashboard
	if err := edc.db.Preload("User").
		Order("updated_at DESC").
		Limit(10).
		Find(&recentDashboards).Error; err != nil {
		return activities
	}

	for _, dashboard := range recentDashboards {
		activities = append(activities, DashboardActivity{
			UserID:    dashboard.UserID,
			Username:  dashboard.User.Username,
			Action:    "更新仪表板",
			Timestamp: dashboard.UpdatedAt,
		})
	}

	return activities
}

// getUsageStats 获取使用统计
func (edc *EnhancedDashboardController) getUsageStats(dashboards []models.Dashboard) DashboardUsageStats {
	var stats DashboardUsageStats
	
	// 计算总用户数
	var totalUsers int64
	edc.db.Model(&models.User{}).Count(&totalUsers)
	stats.TotalUsers = int(totalUsers)
	
	stats.ActiveUsers = len(dashboards)
	
	// 计算平均widget数
	if len(dashboards) > 0 {
		totalWidgets := 0
		userWidgetCounts := make(map[uint]int)
		
		for _, dashboard := range dashboards {
			var config models.DashboardConfig
			if err := json.Unmarshal([]byte(dashboard.Config), &config); err != nil {
				continue
			}
			
			widgetCount := len(config.Widgets)
			totalWidgets += widgetCount
			userWidgetCounts[dashboard.UserID] = widgetCount
		}
		
		stats.AvgWidgetsPerUser = float64(totalWidgets) / float64(len(dashboards))
		
		// 获取top用户
		type userWidgetInfo struct {
			userID      uint
			widgetCount int
		}
		
		var topUsers []userWidgetInfo
		for userID, count := range userWidgetCounts {
			topUsers = append(topUsers, userWidgetInfo{
				userID:      userID,
				widgetCount: count,
			})
		}
		
		// 排序
		for i := 0; i < len(topUsers)-1; i++ {
			for j := i + 1; j < len(topUsers); j++ {
				if topUsers[i].widgetCount < topUsers[j].widgetCount {
					topUsers[i], topUsers[j] = topUsers[j], topUsers[i]
				}
			}
		}
		
		// 获取用户信息
		if len(topUsers) > 5 {
			topUsers = topUsers[:5]
		}
		
		for _, userInfo := range topUsers {
			var user models.User
			if err := edc.db.First(&user, userInfo.userID).Error; err == nil {
				var dashboard models.Dashboard
				edc.db.Where("user_id = ?", userInfo.userID).First(&dashboard)
				
				stats.TopUsers = append(stats.TopUsers, UserUsageInfo{
					UserID:       userInfo.userID,
					Username:     user.Username,
					WidgetCount:  userInfo.widgetCount,
					LastActivity: dashboard.UpdatedAt,
				})
			}
		}
	}
	
	return stats
}

// GetUserDashboardEnhanced 获取增强的用户仪表板配置
func (edc *EnhancedDashboardController) GetUserDashboardEnhanced(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var dashboard models.Dashboard
	err := edc.db.Where("user_id = ?", userID).First(&dashboard).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 根据用户角色返回推荐的默认配置
			defaultConfig := edc.getDefaultConfigByUserRole(c)
			c.JSON(http.StatusOK, defaultConfig)
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

	// 添加增强信息
	response := gin.H{
		"widgets":      config.Widgets,
		"last_updated": dashboard.UpdatedAt,
		"created_at":   dashboard.CreatedAt,
		"meta": gin.H{
			"total_widgets":  len(config.Widgets),
			"visible_widgets": edc.countVisibleWidgets(config.Widgets),
			"categories":     edc.categorizeWidgets(config.Widgets),
		},
	}

	c.JSON(http.StatusOK, response)
}

// getDefaultConfigByUserRole 根据用户角色获取默认配置
func (edc *EnhancedDashboardController) getDefaultConfigByUserRole(c *gin.Context) gin.H {
	// 获取用户角色
	roles, exists := c.Get("roles")
	if !exists {
		roles = []string{"user"}
	}
	
	roleList := roles.([]string)
	
	// 根据角色确定默认配置
	var defaultWidgets []models.DashboardWidget
	
	isAdmin := edc.hasRole(roleList, "admin")
	isOperator := edc.hasRole(roleList, "operator")
	
	if isAdmin {
		// 管理员默认配置
		defaultWidgets = []models.DashboardWidget{
			{
				ID:       "widget-1",
				Type:     "KUBERNETES",
				Title:    "Kubernetes集群",
				URL:      "/kubernetes",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 0,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
			{
				ID:       "widget-2",
				Type:     "SALTSTACK",
				Title:    "SaltStack配置",
				URL:      "/saltstack",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 1,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
			{
				ID:       "widget-3",
				Type:     "MONITORING",
				Title:    "系统监控",
				URL:      "/grafana",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 2,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
		}
	} else if isOperator {
		// 运维人员默认配置
		defaultWidgets = []models.DashboardWidget{
			{
				ID:       "widget-1",
				Type:     "JUPYTERHUB",
				Title:    "Jupyter开发环境",
				URL:      "/jupyter",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 0,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
			{
				ID:       "widget-2",
				Type:     "GITEA",
				Title:    "Git代码仓库",
				URL:      "/gitea",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 1,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
			{
				ID:       "widget-3",
				Type:     "ANSIBLE",
				Title:    "Ansible自动化",
				URL:      "/ansible",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 2,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
		}
	} else {
		// 普通用户默认配置
		defaultWidgets = []models.DashboardWidget{
			{
				ID:       "widget-1",
				Type:     "JUPYTERHUB",
				Title:    "Jupyter研究环境",
				URL:      "/jupyter",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 0,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
			{
				ID:       "widget-2",
				Type:     "SLURM",
				Title:    "Slurm计算集群",
				URL:      "/slurm",
				Size:     models.DashboardSize{Width: 12, Height: 600},
				Position: 1,
				Visible:  true,
				Settings: make(map[string]interface{}),
			},
		}
	}

	return gin.H{
		"widgets": defaultWidgets,
		"meta": gin.H{
			"is_default":     true,
			"recommended_for": edc.getRoleDescription(roleList),
			"total_widgets":   len(defaultWidgets),
		},
	}
}

// hasRole 检查用户是否有指定角色
func (edc *EnhancedDashboardController) hasRole(roles []string, targetRole string) bool {
	for _, role := range roles {
		if role == targetRole {
			return true
		}
	}
	return false
}

// getRoleDescription 获取角色描述
func (edc *EnhancedDashboardController) getRoleDescription(roles []string) string {
	if edc.hasRole(roles, "admin") {
		return "系统管理员"
	}
	if edc.hasRole(roles, "operator") {
		return "运维人员"
	}
	return "普通用户"
}

// countVisibleWidgets 计算可见widget数量
func (edc *EnhancedDashboardController) countVisibleWidgets(widgets []models.DashboardWidget) int {
	count := 0
	for _, widget := range widgets {
		if widget.Visible {
			count++
		}
	}
	return count
}

// categorizeWidgets 对widget进行分类
func (edc *EnhancedDashboardController) categorizeWidgets(widgets []models.DashboardWidget) map[string]int {
	categories := make(map[string]int)
	
	categoryMap := map[string]string{
		"JUPYTERHUB":  "development",
		"GITEA":       "development",
		"KUBERNETES":  "infrastructure", 
		"ANSIBLE":     "automation",
		"SLURM":       "compute",
		"SALTSTACK":   "infrastructure",
		"MONITORING":  "monitoring",
		"CUSTOM":      "custom",
	}
	
	for _, widget := range widgets {
		if category, exists := categoryMap[widget.Type]; exists {
			categories[category]++
		} else {
			categories["other"]++
		}
	}
	
	return categories
}

// CloneDashboard 克隆仪表板配置
func (edc *EnhancedDashboardController) CloneDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	sourceUserID, err := strconv.ParseUint(c.Param("sourceUserId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的源用户ID"})
		return
	}

	// 获取源用户的仪表板配置
	var sourceDashboard models.Dashboard
	if err := edc.db.Where("user_id = ?", sourceUserID).First(&sourceDashboard).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "源仪表板不存在"})
		return
	}

	// 创建或更新当前用户的仪表板
	var userDashboard models.Dashboard
	err = edc.db.Where("user_id = ?", userID).First(&userDashboard).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新的仪表板
		userDashboard = models.Dashboard{
			UserID: userID.(uint),
			Config: sourceDashboard.Config,
		}
		if err := edc.db.Create(&userDashboard).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建仪表板失败"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询仪表板失败"})
		return
	} else {
		// 更新现有仪表板
		userDashboard.Config = sourceDashboard.Config
		if err := edc.db.Save(&userDashboard).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新仪表板失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "仪表板克隆成功",
		"dashboard_id": userDashboard.ID,
	})
}

// ExportDashboard 导出仪表板配置
func (edc *EnhancedDashboardController) ExportDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var dashboard models.Dashboard
	if err := edc.db.Where("user_id = ?", userID).First(&dashboard).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "仪表板不存在"})
		return
	}

	// 获取用户信息
	var user models.User
	edc.db.First(&user, userID)

	exportData := gin.H{
		"version":     "1.0",
		"export_time": time.Now(),
		"user":        user.Username,
		"config":      dashboard.Config,
		"metadata": gin.H{
			"created_at": dashboard.CreatedAt,
			"updated_at": dashboard.UpdatedAt,
		},
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=dashboard-export.json")
	c.JSON(http.StatusOK, exportData)
}

// ImportDashboard 导入仪表板配置
func (edc *EnhancedDashboardController) ImportDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var req struct {
		Config   string `json:"config" binding:"required"`
		Overwrite bool  `json:"overwrite"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	// 验证配置格式
	var config models.DashboardConfig
	if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置格式无效"})
		return
	}

	// 检查用户是否已有仪表板
	var dashboard models.Dashboard
	err := edc.db.Where("user_id = ?", userID).First(&dashboard).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新仪表板
		dashboard = models.Dashboard{
			UserID: userID.(uint),
			Config: req.Config,
		}
		if err := edc.db.Create(&dashboard).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存仪表板失败"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询仪表板失败"})
		return
	} else {
		// 已存在仪表板
		if !req.Overwrite {
			c.JSON(http.StatusConflict, gin.H{"error": "仪表板已存在，请选择覆盖选项"})
			return
		}
		
		dashboard.Config = req.Config
		if err := edc.db.Save(&dashboard).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新仪表板失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "仪表板导入成功",
		"dashboard_id": dashboard.ID,
		"widgets_count": len(config.Widgets),
	})
}
