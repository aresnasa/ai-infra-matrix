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

// LDAP配置和同步相关模型

// LDAPConfig LDAP配置
type LDAPConfig struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	Server           string    `json:"server" gorm:"not null;size:255"`
	Port             int       `json:"port" gorm:"default:389"`
	UseSSL           bool      `json:"use_ssl" gorm:"default:false"`
	BaseDN           string    `json:"base_dn" gorm:"not null;size:500"`
	BindDN           string    `json:"bind_dn" gorm:"size:500"`
	BindPassword     string    `json:"bind_password" gorm:"size:255"`
	UserFilter       string    `json:"user_filter" gorm:"size:500"`
	GroupFilter      string    `json:"group_filter" gorm:"size:500"`
	UsernameAttr     string    `json:"username_attr" gorm:"default:'uid';size:100"`
	EmailAttr        string    `json:"email_attr" gorm:"default:'mail';size:100"`
	DisplayNameAttr  string    `json:"display_name_attr" gorm:"default:'cn';size:100"`
	GroupNameAttr    string    `json:"group_name_attr" gorm:"default:'cn';size:100"`
	MemberAttr       string    `json:"member_attr" gorm:"default:'member';size:100"`
	SyncEnabled      bool      `json:"sync_enabled" gorm:"default:false"`
	AutoCreateUser   bool      `json:"auto_create_user" gorm:"default:true"`
	AutoCreateGroup  bool      `json:"auto_create_group" gorm:"default:true"`
	DefaultRole      string    `json:"default_role" gorm:"default:'user';size:50"`
	LastSync         *time.Time `json:"last_sync,omitempty"`
	SyncStatus       string    `json:"sync_status" gorm:"default:'never';size:50"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// LDAPSyncResult LDAP同步结果
type LDAPSyncResult struct {
	Created   int                     `json:"created"`
	Updated   int                     `json:"updated"`
	Skipped   int                     `json:"skipped"`
	Errors    int                     `json:"errors"`
	Details   []LDAPSyncDetail        `json:"details"`
	StartTime time.Time               `json:"start_time"`
	EndTime   time.Time               `json:"end_time"`
	Duration  string                  `json:"duration"`
}

// LDAPSyncDetail LDAP同步详情
type LDAPSyncDetail struct {
	Action   string `json:"action"`   // created, updated, skipped, error
	Username string `json:"username"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}

// LDAPUser LDAP用户信息
type LDAPUser struct {
	DN          string   `json:"dn"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Groups      []string `json:"groups"`
}

// LDAPGroup LDAP组信息
type LDAPGroup struct {
	DN      string   `json:"dn"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// LDAPTestRequest LDAP连接测试请求
type LDAPTestRequest struct {
	Server       string `json:"server" binding:"required"`
	Port         int    `json:"port" binding:"required"`
	UseSSL       bool   `json:"use_ssl"`
	BaseDN       string `json:"base_dn" binding:"required"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
}

// LDAPTestResponse LDAP连接测试响应
type LDAPTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// LDAPSyncOptions LDAP同步选项
type LDAPSyncOptions struct {
	ForceUpdate     bool     `json:"force_update"`      // 强制更新已存在用户
	SyncGroups      bool     `json:"sync_groups"`       // 同步用户组
	DeleteMissing   bool     `json:"delete_missing"`    // 删除LDAP中不存在的用户
	DryRun          bool     `json:"dry_run"`           // 仅模拟，不实际创建
	UserFilter      string   `json:"user_filter"`       // 自定义用户过滤器
	SelectedUsers   []string `json:"selected_users"`    // 仅同步指定用户
}

// 用户组权限关联表
type UserGroupRole struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserGroupID uint      `json:"user_group_id" gorm:"not null;index"`
	RoleID      uint      `json:"role_id" gorm:"not null;index"`
	CreatedAt   time.Time `json:"created_at"`
	
	// 关联关系
	UserGroup UserGroup `json:"user_group,omitempty" gorm:"foreignKey:UserGroupID"`
	Role      Role      `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// 确保表名正确
func (UserGroupRole) TableName() string {
	return "user_group_roles"
}
