package models

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
	"gorm.io/gorm"
)

// Response 通用API响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 角色常量
const (
	RoleAdmin      = "admin"
	RoleUser       = "user"
	RoleViewer     = "viewer"
	RoleSuperAdmin = "super-admin"
)

// User 用户表
type User struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Username      string         `json:"username" gorm:"uniqueIndex;not null;size:100"`
	Email         string         `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Name          string         `json:"name" gorm:"size:255"` // 显示名称
	Password      string         `json:"-" gorm:"not null;size:255"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	AuthSource    string         `json:"auth_source" gorm:"default:'local';size:50"` // 认证来源: local, ldap
	LDAPDn        string         `json:"ldap_dn,omitempty" gorm:"size:500"`          // LDAP用户的DN
	DashboardRole string         `json:"dashboard_role" gorm:"size:50"`              // 仪表板角色
	RoleTemplate  string         `json:"role_template" gorm:"size:50"`               // 角色模板
	LastLogin     *time.Time     `json:"last_login,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系 - 用户拥有的项目
	Projects []Project `json:"projects,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	// RBAC 关联关系
	Roles      []Role      `json:"roles,omitempty" gorm:"many2many:user_roles"`
	UserGroups []UserGroup `json:"user_groups,omitempty" gorm:"many2many:user_group_memberships"`
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求结构
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	// Optional: department/team for LDAP group and K8s namespace mapping
	Department string `json:"department" binding:"omitempty"`
	// Optional desired role; maps to RBAC and K8s ClusterRole (viewer|user|admin)
	Role string `json:"role" binding:"omitempty,oneof=viewer user admin"`
	// Role template for predefined role assignments
	RoleTemplate string `json:"role_template" binding:"omitempty,oneof=admin data-developer model-developer sre engineer"`
	// Registration requires admin approval
	RequiresApproval bool `json:"requires_approval" binding:"omitempty"`
}

// ChangePasswordRequest 修改密码请求结构
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateUserProfileRequest 更新用户个人信息请求结构
type UpdateUserProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// AdminResetPasswordRequest 管理员重置用户密码请求结构
type AdminResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateUserGroupsRequest 管理员更新用户组请求结构
type UpdateUserGroupsRequest struct {
	UserGroupIDs []uint `json:"user_group_ids" binding:"required"`
}

// AssignRoleRequest 角色分配请求结构
type AssignRoleRequest struct {
	UserID     uint   `json:"user_id" binding:"required"`
	RoleName   string `json:"role_name" binding:"required"`
	AssignedBy string `json:"assigned_by"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Token     string `json:"token"`
	User      User   `json:"user"`
	ExpiresAt int64  `json:"expires_at"`
}

// Project 项目表
type Project struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"` // 添加用户关联
	Name        string         `json:"name" gorm:"not null;size:255"`
	Description string         `json:"description" gorm:"size:1000"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User      User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Hosts     []Host     `json:"hosts,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Variables []Variable `json:"variables,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Tasks     []Task     `json:"tasks,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

// Host 主机表
type Host struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ProjectID uint           `json:"project_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"not null;size:255"`
	IP        string         `json:"ip" gorm:"not null;size:45"`
	Port      int            `json:"port" gorm:"default:22"`
	User      string         `json:"user" gorm:"not null;size:100"`
	Group     string         `json:"group" gorm:"size:100"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Variable 变量表
type Variable struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ProjectID uint           `json:"project_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"not null;size:255"`
	Value     string         `json:"value" gorm:"type:text"`
	Type      string         `json:"type" gorm:"size:50;default:'string'"` // string, number, boolean, list
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Task 任务表
type Task struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ProjectID uint           `json:"project_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"not null;size:255"`
	Module    string         `json:"module" gorm:"not null;size:100"`
	Args      string         `json:"args" gorm:"type:text"`
	OrderNum  int            `json:"order" gorm:"default:0"`
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// PlaybookGeneration 生成记录表
type PlaybookGeneration struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ProjectID uint      `json:"project_id" gorm:"not null;index"`
	FileName  string    `json:"file_name" gorm:"not null;size:255"`
	FilePath  string    `json:"file_path" gorm:"not null;size:500"`
	Status    string    `json:"status" gorm:"size:50;default:'success'"` // success, failed
	Error     string    `json:"error,omitempty" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

// Kubernetes 集群管理模型

// KubernetesCluster Kubernetes集群表
type KubernetesCluster struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Name           string         `json:"name" gorm:"not null;size:255"`
	Description    string         `json:"description" gorm:"size:1000"`
	APIServer      string         `json:"api_server" gorm:"not null;size:500"`
	KubeConfig     string         `json:"kube_config" gorm:"type:text"`     // Kubeconfig内容（自动加密）
	KubeConfigPath string         `json:"kube_config_path" gorm:"size:500"` // Kubeconfig文件路径
	Namespace      string         `json:"namespace" gorm:"size:255;default:'default'"`
	Status         string         `json:"status" gorm:"size:50;default:'unknown'"` // connected, disconnected, error, unknown
	Version        string         `json:"version" gorm:"size:100"`
	UserID         uint           `json:"user_id" gorm:"not null;index"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	LastCheckAt    *time.Time     `json:"last_check_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// KubernetesClusterCreateRequest 创建Kubernetes集群请求
type KubernetesClusterCreateRequest struct {
	Name           string `json:"name" binding:"required,min=2,max=255"`
	Description    string `json:"description"`
	APIServer      string `json:"api_server" binding:"required"`
	KubeConfig     string `json:"kube_config"`
	KubeConfigPath string `json:"kube_config_path"`
	Namespace      string `json:"namespace"`
}

// KubernetesClusterUpdateRequest 更新Kubernetes集群请求
type KubernetesClusterUpdateRequest struct {
	Name           string `json:"name" binding:"omitempty,min=2,max=255"`
	Description    string `json:"description"`
	APIServer      string `json:"api_server"`
	KubeConfig     string `json:"kube_config"`
	KubeConfigPath string `json:"kube_config_path"`
	Namespace      string `json:"namespace"`
	IsActive       *bool  `json:"is_active"`
}

// Ansible执行历史模型

// AnsibleExecution Ansible执行历史表
type AnsibleExecution struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	ProjectID     uint           `json:"project_id" gorm:"not null;index"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`
	ExecutionType string         `json:"execution_type" gorm:"not null;size:50"` // dry-run, execute
	Environment   string         `json:"environment" gorm:"size:100"`            // dev, test, prod
	PlaybookPath  string         `json:"playbook_path" gorm:"not null;size:500"`
	InventoryPath string         `json:"inventory_path" gorm:"size:500"`
	ExtraVars     string         `json:"extra_vars" gorm:"type:text"`             // JSON格式的额外变量
	Status        string         `json:"status" gorm:"size:50;default:'pending'"` // pending, running, success, failed, cancelled
	StartTime     *time.Time     `json:"start_time,omitempty"`
	EndTime       *time.Time     `json:"end_time,omitempty"`
	Duration      int            `json:"duration"`                      // 执行时长（秒）
	Output        string         `json:"output" gorm:"type:text"`       // 执行输出
	ErrorOutput   string         `json:"error_output" gorm:"type:text"` // 错误输出
	ExitCode      int            `json:"exit_code"`
	PID           int            `json:"pid"` // 进程ID
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Project Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User    User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// AnsibleExecutionRequest Ansible执行请求
type AnsibleExecutionRequest struct {
	ProjectID     uint   `json:"project_id" binding:"required"`
	ExecutionType string `json:"execution_type" binding:"required,oneof=dry-run execute"`
	Environment   string `json:"environment" binding:"required"`
	ExtraVars     string `json:"extra_vars"`
	Inventory     string `json:"inventory"` // 自定义inventory内容
}

// AnsibleExecutionResponse Ansible执行响应
type AnsibleExecutionResponse struct {
	ID           uint   `json:"id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	ExecutionURL string `json:"execution_url"` // 查看执行详情的URL
}

// RBAC 相关模型

// UserGroup 用户组表
type UserGroup struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	Name              string         `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Description       string         `json:"description" gorm:"size:500"`
	DashboardTemplate string         `json:"dashboard_template" gorm:"type:text"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Users []User `json:"users,omitempty" gorm:"many2many:user_group_memberships"`
}

// Permission 权限表
type Permission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Resource    string    `json:"resource" gorm:"not null;size:100"` // projects, users, roles, etc.
	Verb        string    `json:"verb" gorm:"not null;size:50"`      // create, read, update, delete, list
	Scope       string    `json:"scope" gorm:"size:100;default:'*'"` // * for all, specific resource ID
	Description string    `json:"description" gorm:"size:500"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	Roles []Role `json:"roles,omitempty" gorm:"many2many:role_permissions"`
}

// GetPermissionKey 获取权限键
func (p Permission) GetPermissionKey() string {
	return fmt.Sprintf("%s:%s:%s", p.Resource, p.Verb, p.Scope)
}

// Role 角色表
type Role struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Description string         `json:"description" gorm:"size:500"`
	IsSystem    bool           `json:"is_system" gorm:"default:false"` // 系统角色不可删除
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions"`
	Users       []User       `json:"users,omitempty" gorm:"many2many:user_roles"`
	UserGroups  []UserGroup  `json:"user_groups,omitempty" gorm:"many2many:user_group_roles"`
}

// RoleBinding 角色绑定表 (类似K8s RoleBinding)
type RoleBinding struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Name      string `json:"name" gorm:"not null;size:100"`
	RoleID    uint   `json:"role_id" gorm:"not null;index"`
	Namespace string `json:"namespace" gorm:"size:100;default:'default'"` // 命名空间概念

	// Subject 可以是用户或用户组
	SubjectType string `json:"subject_type" gorm:"not null;size:50"` // user, group
	SubjectID   uint   `json:"subject_id" gorm:"not null;index"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Role Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// UserGroupMembership 用户组成员关系表
type UserGroupMembership struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"not null;index"`
	UserGroupID uint      `json:"user_group_id" gorm:"not null;index"`
	CreatedAt   time.Time `json:"created_at"`

	// 关联关系
	User      User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	UserGroup UserGroup `json:"user_group,omitempty" gorm:"foreignKey:UserGroupID"`
}

// UserRole 用户角色关系表
type UserRole struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	RoleID    uint      `json:"role_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`

	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// UserGroupRole 用户组角色关系表
type UserGroupRole struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserGroupID uint      `json:"user_group_id" gorm:"not null;index"`
	RoleID      uint      `json:"role_id" gorm:"not null;index"`
	CreatedAt   time.Time `json:"created_at"`

	// 关联关系
	UserGroup UserGroup `json:"user_group,omitempty" gorm:"foreignKey:UserGroupID"`
	Role      Role      `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

// RolePermission 角色权限关系表
type RolePermission struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RoleID       uint      `json:"role_id" gorm:"not null;index"`
	PermissionID uint      `json:"permission_id" gorm:"not null;index"`
	CreatedAt    time.Time `json:"created_at"`

	// 关联关系
	Role       Role       `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Permission Permission `json:"permission,omitempty" gorm:"foreignKey:PermissionID"`
}

// LDAP相关模型

// LDAPConfig LDAP配置表
type LDAPConfig struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Server          string     `json:"server" gorm:"not null;size:255"`
	Port            int        `json:"port" gorm:"default:389"`
	BindDN          string     `json:"bind_dn" gorm:"not null;size:500"`
	BindPassword    string     `json:"bind_password" gorm:"not null;size:255"`
	BaseDN          string     `json:"base_dn" gorm:"not null;size:500"`
	UserFilter      string     `json:"user_filter" gorm:"not null;size:500"`
	UsernameAttr    string     `json:"username_attr" gorm:"default:'uid';size:100"`
	EmailAttr       string     `json:"email_attr" gorm:"default:'mail';size:100"`
	NameAttr        string     `json:"name_attr" gorm:"default:'cn';size:100"`
	DisplayNameAttr string     `json:"display_name_attr" gorm:"default:'displayName';size:100"`
	GroupNameAttr   string     `json:"group_name_attr" gorm:"default:'cn';size:100"`
	MemberAttr      string     `json:"member_attr" gorm:"default:'member';size:100"`
	UseSSL          bool       `json:"use_ssl" gorm:"default:false"`
	SkipVerify      bool       `json:"skip_verify" gorm:"default:false"`
	IsEnabled       bool       `json:"is_enabled" gorm:"default:false"`
	SyncEnabled     bool       `json:"sync_enabled" gorm:"default:false"`
	AutoCreateUser  bool       `json:"auto_create_user" gorm:"default:true"`
	AutoCreateGroup bool       `json:"auto_create_group" gorm:"default:true"`
	DefaultRole     string     `json:"default_role" gorm:"default:'user';size:50"`
	SyncStatus      string     `json:"sync_status" gorm:"default:'never';size:50"`
	LastSync        *time.Time `json:"last_sync,omitempty"`
	// Optional OUs for user and groups management
	UsersOU  string `json:"users_ou" gorm:"size:255"`
	GroupsOU string `json:"groups_ou" gorm:"size:255"`
	// Optional group DN for admins and the member attribute name (e.g., member or memberUid)
	AdminGroupDN    string    `json:"admin_group_dn" gorm:"size:500"`
	GroupMemberAttr string    `json:"group_member_attr" gorm:"size:100"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// LDAPUser LDAP用户信息
type LDAPUser struct {
	DN          string   `json:"dn"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Groups      []string `json:"groups,omitempty"` // 添加用户组信息
}

// LDAPGroup LDAP组信息
type LDAPGroup struct {
	DN      string   `json:"dn"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// LDAPSyncOptions LDAP同步选项
type LDAPSyncOptions struct {
	DryRun        bool     `json:"dry_run"`
	BatchSize     int      `json:"batch_size"`
	ForceUpdate   bool     `json:"force_update"`   // 强制更新已存在用户
	SyncGroups    bool     `json:"sync_groups"`    // 同步用户组
	DeleteMissing bool     `json:"delete_missing"` // 删除LDAP中不存在的用户
	UserFilter    string   `json:"user_filter"`    // 自定义用户过滤器
	SelectedUsers []string `json:"selected_users"` // 仅同步指定用户
}

// LDAPSyncResult LDAP同步结果
type LDAPSyncResult struct {
	Created   int              `json:"created"`
	Updated   int              `json:"updated"`
	Skipped   int              `json:"skipped"`
	Errors    int              `json:"errors"`
	Details   []LDAPSyncDetail `json:"details"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	Duration  string           `json:"duration"`
}

// LDAPSyncDetail LDAP同步详情
type LDAPSyncDetail struct {
	Action   string `json:"action"` // created, updated, skipped, error
	Username string `json:"username"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}

// LDAPTestResponse LDAP连接测试响应
type LDAPTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// LDAPConfigRequest LDAP配置请求
type LDAPConfigRequest struct {
	Server          string `json:"server" binding:"required"`
	Port            int    `json:"port" binding:"required,min=1,max=65535"`
	BindDN          string `json:"bind_dn" binding:"required"`
	BindPassword    string `json:"bind_password" binding:"required"`
	BaseDN          string `json:"base_dn" binding:"required"`
	UserFilter      string `json:"user_filter" binding:"required"`
	UsernameAttr    string `json:"username_attr"`
	EmailAttr       string `json:"email_attr"`
	NameAttr        string `json:"name_attr"`
	UseSSL          bool   `json:"use_ssl"`
	SkipVerify      bool   `json:"skip_verify"`
	IsEnabled       bool   `json:"is_enabled"`
	UsersOU         string `json:"users_ou"`
	GroupsOU        string `json:"groups_ou"`
	AdminGroupDN    string `json:"admin_group_dn"`
	GroupMemberAttr string `json:"group_member_attr"`
}

// LDAPTestRequest LDAP测试连接请求
type LDAPTestRequest struct {
	Server       string `json:"server" validate:"required"`
	Port         int    `json:"port" validate:"required,min=1,max=65535"`
	BindDN       string `json:"bind_dn" validate:"required"`
	BindPassword string `json:"bind_password" validate:"required"`
	BaseDN       string `json:"base_dn" validate:"required"`
	UserFilter   string `json:"user_filter"`
	UseSSL       bool   `json:"use_ssl"`
	SkipVerify   bool   `json:"skip_verify"`
	// Windows兼容性字段
	Timeout        int `json:"timeout,omitempty"`         // 连接超时时间(秒)
	MaxConnections int `json:"max_connections,omitempty"` // 最大连接数
}
type PermissionCheckRequest struct {
	Resource  string `json:"resource" binding:"required"`
	Verb      string `json:"verb" binding:"required"`
	Scope     string `json:"scope,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// 权限检查响应
type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// 创建角色请求
type CreateRoleRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=100"`
	Description   string `json:"description,omitempty"`
	PermissionIDs []uint `json:"permission_ids,omitempty"`
}

// 创建用户组请求
type CreateUserGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description,omitempty"`
}

// 角色分配请求
type RoleAssignmentRequest struct {
	SubjectType string `json:"subject_type" binding:"required,oneof=user group"`
	SubjectID   uint   `json:"subject_id" binding:"required"`
	RoleID      uint   `json:"role_id" binding:"required"`
}

// 创建权限请求
type CreatePermissionRequest struct {
	Resource    string `json:"resource" binding:"required"`
	Verb        string `json:"verb" binding:"required"`
	Scope       string `json:"scope,omitempty"`
	Description string `json:"description,omitempty"`
}

// TableName 自定义表名
func (User) TableName() string {
	return "users"
}

func (Project) TableName() string {
	return "projects"
}

func (Host) TableName() string {
	return "hosts"
}

func (Variable) TableName() string {
	return "variables"
}

func (Task) TableName() string {
	return "tasks"
}

func (PlaybookGeneration) TableName() string {
	return "playbook_generations"
}

func (KubernetesCluster) TableName() string {
	return "kubernetes_clusters"
}

// BeforeCreate GORM钩子：创建前加密敏感数据
func (kc *KubernetesCluster) BeforeCreate(tx *gorm.DB) error {
	return kc.encryptSensitiveData()
}

// BeforeUpdate GORM钩子：更新前加密敏感数据
func (kc *KubernetesCluster) BeforeUpdate(tx *gorm.DB) error {
	return kc.encryptSensitiveData()
}

// BeforeSave GORM钩子：保存前加密敏感数据
func (kc *KubernetesCluster) BeforeSave(tx *gorm.DB) error {
	return kc.encryptSensitiveData()
}

// AfterFind GORM钩子：查询后解密敏感数据
func (kc *KubernetesCluster) AfterFind(tx *gorm.DB) error {
	return kc.decryptSensitiveData()
}

// encryptSensitiveData 加密敏感数据
func (kc *KubernetesCluster) encryptSensitiveData() error {
	if kc.KubeConfig != "" {
		// 引入加密服务
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		cryptoService := utils.NewCryptoService(cfg.EncryptionKey)

		// 检查是否已经加密
		if !cryptoService.IsEncrypted(kc.KubeConfig) {
			encrypted, err := cryptoService.Encrypt(kc.KubeConfig)
			if err != nil {
				return fmt.Errorf("failed to encrypt KubeConfig: %w", err)
			}
			kc.KubeConfig = encrypted
		}
	}
	return nil
}

// decryptSensitiveData 解密敏感数据
func (kc *KubernetesCluster) decryptSensitiveData() error {
	if kc.KubeConfig != "" {
		// 引入加密服务
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		cryptoService := utils.NewCryptoService(cfg.EncryptionKey)

		// 检查是否已经加密，如果是则解密
		if cryptoService.IsEncrypted(kc.KubeConfig) {
			decrypted, err := cryptoService.Decrypt(kc.KubeConfig)
			if err != nil {
				return fmt.Errorf("failed to decrypt KubeConfig: %w", err)
			}
			kc.KubeConfig = decrypted
		}
	}
	return nil
}

func (AnsibleExecution) TableName() string {
	return "ansible_executions"
}

func (UserGroup) TableName() string {
	return "user_groups"
}

func (Permission) TableName() string {
	return "permissions"
}

func (Role) TableName() string {
	return "roles"
}

func (RoleBinding) TableName() string {
	return "role_bindings"
}

func (UserGroupMembership) TableName() string {
	return "user_group_memberships"
}

func (UserRole) TableName() string {
	return "user_roles"
}

func (UserGroupRole) TableName() string {
	return "user_group_roles"
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

func (LDAPConfig) TableName() string {
	return "ldap_configs"
}

// UserNavigationConfig 用户导航配置模型
type UserNavigationConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Config    string    `json:"config" gorm:"type:text"` // JSON格式的导航配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (UserNavigationConfig) TableName() string {
	return "user_navigation_configs"
}

// RegistrationApproval 注册审批表
type RegistrationApproval struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	Username     string     `json:"username" gorm:"not null;size:50"`
	Email        string     `json:"email" gorm:"not null;size:100"`
	Department   string     `json:"department" gorm:"size:100"`
	RoleTemplate string     `json:"role_template" gorm:"size:50"`
	Status       string     `json:"status" gorm:"not null;default:'pending';size:20"` // pending, approved, rejected
	ApprovedBy   *uint      `json:"approved_by,omitempty" gorm:"index"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	RejectedBy   *uint      `json:"rejected_by,omitempty" gorm:"index"`
	RejectedAt   *time.Time `json:"rejected_at,omitempty"`
	RejectReason string     `json:"reject_reason" gorm:"size:500"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联关系
	User     User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Approver User `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
	Rejector User `json:"rejector,omitempty" gorm:"foreignKey:RejectedBy"`
}

func (RegistrationApproval) TableName() string {
	return "registration_approvals"
}

// ==================== 作业管理相关模型 ====================

// Job 作业表
type Job struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"not null;index"`
	ClusterID  string     `json:"cluster_id" gorm:"not null;size:100;index"`
	JobID      uint32     `json:"job_id" gorm:"not null;index"` // SLURM作业ID
	Name       string     `json:"name" gorm:"not null;size:255"`
	Command    string     `json:"command" gorm:"type:text"`
	WorkingDir string     `json:"working_dir" gorm:"size:500"`
	Status     string     `json:"status" gorm:"not null;size:50;index"` // PENDING, RUNNING, COMPLETED, FAILED, CANCELLED
	ExitCode   *int       `json:"exit_code,omitempty"`
	SubmitTime time.Time  `json:"submit_time"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Partition  string     `json:"partition" gorm:"size:100"`
	Nodes      int        `json:"nodes" gorm:"default:1"`
	CPUs       int        `json:"cpus" gorm:"default:1"`
	Memory     string     `json:"memory" gorm:"size:50"`     // e.g., "4G", "8000M"
	TimeLimit  string     `json:"time_limit" gorm:"size:50"` // e.g., "01:00:00"
	StdOut     string     `json:"std_out" gorm:"size:500"`
	StdErr     string     `json:"std_err" gorm:"size:500"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 关联关系
	User    User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Cluster Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
}

func (Job) TableName() string {
	return "jobs"
}

// JobStatus 作业状态信息
type JobStatus struct {
	JobID uint32 `json:"job_id"`
	State string `json:"state"` // PENDING, RUNNING, COMPLETED, FAILED, CANCELLED, etc.
}

// Cluster 集群表
type Cluster struct {
	ID          string    `json:"id" gorm:"primaryKey;size:100"`
	Name        string    `json:"name" gorm:"not null;size:255"`
	Description string    `json:"description" gorm:"type:text"`
	Host        string    `json:"host" gorm:"not null;size:255"`
	Port        int       `json:"port" gorm:"default:22"`
	Username    string    `json:"username" gorm:"size:100"`                        // SSH用户名
	Password    string    `json:"password,omitempty" gorm:"size:255"`              // SSH密码
	Status      string    `json:"status" gorm:"not null;default:'active';size:20"` // active, inactive, maintenance
	Config      string    `json:"config" gorm:"type:json"`                         // 集群配置JSON
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	Jobs []Job `json:"jobs,omitempty" gorm:"foreignKey:ClusterID"`
}

func (Cluster) TableName() string {
	return "clusters"
}

// JobTemplate 作业模板表
type JobTemplate struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null;size:255"`
	Description string    `json:"description" gorm:"type:text"`
	Script      string    `json:"script" gorm:"type:text"`  // 完整的sbatch脚本内容
	Command     string    `json:"command" gorm:"type:text"` // 主要执行命令
	Partition   string    `json:"partition" gorm:"size:100"`
	Nodes       int       `json:"nodes" gorm:"default:1"`
	CPUs        int       `json:"cpus" gorm:"default:1"`
	Memory      string    `json:"memory" gorm:"size:50"`
	TimeLimit   string    `json:"time_limit" gorm:"size:50"`
	WorkingDir  string    `json:"working_dir" gorm:"size:500"`
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	Category    string    `json:"category" gorm:"size:100"` // 模板分类：计算、深度学习、数据处理等
	Tags        string    `json:"tags" gorm:"type:text"`    // 标签，JSON 格式
	UserID      uint      `json:"user_id" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (JobTemplate) TableName() string {
	return "job_templates"
}

// ==================== API 请求/响应模型 ====================

// SubmitJobRequest 提交作业请求
type SubmitJobRequest struct {
	UserID     string `json:"user_id"`
	ClusterID  string `json:"cluster_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Command    string `json:"command" binding:"required"`
	WorkingDir string `json:"working_dir"`
	Partition  string `json:"partition"`
	Nodes      int    `json:"nodes"`
	CPUs       int    `json:"cpus"`
	Memory     string `json:"memory"`
	TimeLimit  string `json:"time_limit"`
	TemplateID uint   `json:"template_id"` // 可选：从模板创建作业
}

// CreateJobTemplateRequest 创建作业模板请求
type CreateJobTemplateRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Script      string   `json:"script" binding:"required"`
	Command     string   `json:"command"`
	Partition   string   `json:"partition"`
	Nodes       int      `json:"nodes"`
	CPUs        int      `json:"cpus"`
	Memory      string   `json:"memory"`
	TimeLimit   string   `json:"time_limit"`
	WorkingDir  string   `json:"working_dir"`
	IsPublic    bool     `json:"is_public"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// UpdateJobTemplateRequest 更新作业模板请求
type UpdateJobTemplateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Script      string   `json:"script"`
	Command     string   `json:"command"`
	Partition   string   `json:"partition"`
	Nodes       int      `json:"nodes"`
	CPUs        int      `json:"cpus"`
	Memory      string   `json:"memory"`
	TimeLimit   string   `json:"time_limit"`
	WorkingDir  string   `json:"working_dir"`
	IsPublic    bool     `json:"is_public"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// JobTemplateListResponse 作业模板列表响应
type JobTemplateListResponse struct {
	Templates []JobTemplate `json:"templates"`
	Total     int64         `json:"total"`
	Page      int           `json:"page"`
	PageSize  int           `json:"page_size"`
}

// JobListResponse 作业列表响应
type JobListResponse struct {
	Jobs     []Job `json:"jobs"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// JobOutput 作业输出
type JobOutput struct {
	JobID    uint32 `json:"job_id"`
	StdOut   string `json:"stdout"`
	StdErr   string `json:"stderr"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Partitions  []PartitionInfo `json:"partitions"`
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	Name      string `json:"name"`
	Nodes     int    `json:"nodes"`
	State     string `json:"state"`
	CPUs      int    `json:"cpus"`
	Memory    string `json:"memory"`
	TimeLimit string `json:"time_limit"`
}

// FileInfo 文件信息
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
	Mode    string    `json:"mode"`
}

// JobDashboardStats 作业仪表板统计
type JobDashboardStats struct {
	TotalJobs      int64 `json:"total_jobs"`
	RunningJobs    int64 `json:"running_jobs"`
	PendingJobs    int64 `json:"pending_jobs"`
	CompletedJobs  int64 `json:"completed_jobs"`
	FailedJobs     int64 `json:"failed_jobs"`
	TotalClusters  int64 `json:"total_clusters"`
	ActiveClusters int64 `json:"active_clusters"`
}

// SaltStackTask SaltStack任务表
type SaltStackTask struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	TaskID     string     `json:"task_id" gorm:"uniqueIndex;not null;size:100"`
	TaskType   string     `json:"task_type" gorm:"not null;size:50"` // install, execute, deploy
	Status     string     `json:"status" gorm:"not null;size:20"`    // pending, running, success, failed
	TargetHost string     `json:"target_host" gorm:"not null;size:255"`
	Command    string     `json:"command" gorm:"type:text"`
	Output     string     `json:"output" gorm:"type:text"`
	ErrorMsg   string     `json:"error_msg" gorm:"type:text"`
	Progress   int        `json:"progress" gorm:"default:0"` // 0-100
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	Duration   int64      `json:"duration"` // 执行时长（毫秒）
	UserID     uint       `json:"user_id" gorm:"index"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 关联关系
	User     User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	TaskLogs []TaskLog `json:"task_logs,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
	SSHLogs  []SSHLog  `json:"ssh_logs,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
}

// TaskLog 任务日志表
type TaskLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id" gorm:"index;not null;size:100"`
	LogLevel  string    `json:"log_level" gorm:"not null;size:20"` // info, warn, error, debug
	Message   string    `json:"message" gorm:"type:text"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}

// SSHLog SSH连接和执行日志表
type SSHLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      string    `json:"task_id" gorm:"index;not null;size:100"`
	Host        string    `json:"host" gorm:"not null;size:255"`
	Port        int       `json:"port" gorm:"not null;default:22"`
	User        string    `json:"user" gorm:"not null;size:100"`
	Command     string    `json:"command" gorm:"type:text"`
	Output      string    `json:"output" gorm:"type:text"`
	ErrorOutput string    `json:"error_output" gorm:"type:text"`
	ExitCode    int       `json:"exit_code"`
	Duration    int64     `json:"duration"`                       // 执行时长（毫秒）
	Status      string    `json:"status" gorm:"not null;size:20"` // success, failed, timeout
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
}

// SlurmJob SLURM作业表
type SlurmJob struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	JobID       string     `json:"job_id" gorm:"uniqueIndex;not null;size:100"`
	JobName     string     `json:"job_name" gorm:"not null;size:255"`
	Status      string     `json:"status" gorm:"not null;size:20"` // pending, running, completed, failed, cancelled
	Queue       string     `json:"queue" gorm:"size:100"`
	Nodes       int        `json:"nodes" gorm:"default:1"`
	CPUs        int        `json:"cpus" gorm:"default:1"`
	Memory      string     `json:"memory" gorm:"size:50"`     // 如 "4G", "1024M"
	TimeLimit   string     `json:"time_limit" gorm:"size:50"` // 如 "01:00:00"
	WorkDir     string     `json:"work_dir" gorm:"type:text"`
	Command     string     `json:"command" gorm:"type:text"`
	Output      string     `json:"output" gorm:"type:text"`
	ErrorOutput string     `json:"error_output" gorm:"type:text"`
	SubmitTime  time.Time  `json:"submit_time"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	UserID      uint       `json:"user_id" gorm:"index"`
	ClusterID   uint       `json:"cluster_id" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 关联关系
	User    User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Cluster SlurmCluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
}

func (SlurmJob) TableName() string {
	return "slurm_jobs"
}

func (SaltStackTask) TableName() string {
	return "saltstack_tasks"
}

func (TaskLog) TableName() string {
	return "task_logs"
}

func (SSHLog) TableName() string {
	return "ssh_logs"
}

// OSInfo 操作系统信息
type OSInfo struct {
	OS      string `json:"os"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}
