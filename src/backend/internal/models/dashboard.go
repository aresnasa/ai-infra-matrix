package models

import (
	"time"
	"gorm.io/gorm"
)

// Dashboard 用户仪表板配置
type Dashboard struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex"`
	Config    string    `json:"config" gorm:"type:text"` // JSON配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// DashboardWidget Widget配置结构
type DashboardWidget struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	URL      string            `json:"url"`
	Size     DashboardSize     `json:"size"`
	Position int               `json:"position"`
	Visible  bool              `json:"visible"`
	Settings map[string]interface{} `json:"settings"`
}

// DashboardSize Widget尺寸
type DashboardSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardConfig 仪表板配置
type DashboardConfig struct {
	Widgets []DashboardWidget `json:"widgets"`
}

// DashboardUpdateRequest 更新仪表板请求
type DashboardUpdateRequest struct {
	Widgets []DashboardWidget `json:"widgets"`
}

// DashboardStats 仪表板统计信息
type DashboardStats struct {
	TotalUsers     int `json:"total_users"`
	ActiveUsers    int `json:"active_users"`
	TotalWidgets   int `json:"total_widgets"`
	PopularWidgets []WidgetUsageStats `json:"popular_widgets"`
}

// WidgetUsageStats Widget使用统计
type WidgetUsageStats struct {
	WidgetType string `json:"widget_type"`
	UsageCount int    `json:"usage_count"`
}

// EnhancedDashboardResponse 增强仪表板响应
type EnhancedDashboardResponse struct {
	Config      DashboardConfig `json:"config"`
	Stats       DashboardStats  `json:"stats"`
	Permissions []string        `json:"permissions"`
	Templates   []DashboardTemplate `json:"templates"`
}

// DashboardTemplate 仪表板模板
type DashboardTemplate struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Role        string          `json:"role"`
	Config      DashboardConfig `json:"config"`
	IsDefault   bool            `json:"is_default"`
}

// DashboardImportRequest 导入仪表板请求
type DashboardImportRequest struct {
	Config    DashboardConfig `json:"config"`
	Overwrite bool           `json:"overwrite"`
}
