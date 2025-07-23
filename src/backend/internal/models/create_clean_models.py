#!/usr/bin/env python3
"""
重新创建一个干净的 models.go 文件
"""

clean_content = '''package models

import (
	"fmt"
	"time"
	"gorm.io/gorm"
)

// 角色常量
const (
	RoleAdmin    = "admin"
	RoleUser     = "user"
	RoleViewer   = "viewer"
	RoleSuperAdmin = "super-admin"
)

// User 用户表
type User struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Username   string    `json:"username" gorm:"uniqueIndex;not null;size:100"`
	Email      string    `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Password   string    `json:"-" gorm:"not null;size:255"`
	IsActive   bool      `json:"is_active" gorm:"default:true"`
	AuthSource string    `json:"auth_source" gorm:"default:'local';size:50"` // 认证来源: local, ldap
	LDAPDn     string    `json:"ldap_dn,omitempty" gorm:"size:500"`          // LDAP用户的DN
	LastLogin  *time.Time `json:"last_login,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	
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
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"not null;index"` // 添加用户关联
	Name        string    `json:"name" gorm:"not null;size:255"`
	Description string    `json:"description" gorm:"size:1000"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联关系
	User      User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Hosts     []Host     `json:"hosts,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Variables []Variable `json:"variables,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Tasks     []Task     `json:"tasks,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

// Host 主机表
type Host struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null;index"`
	Name      string `json:"name" gorm:"not null;size:255"`
	IP        string `json:"ip" gorm:"not null;size:45"`
	Port      int    `json:"port" gorm:"default:22"`
	User      string `json:"user" gorm:"not null;size:100"`
	Group     string `json:"group" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Variable 变量表
type Variable struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null;index"`
	Name      string `json:"name" gorm:"not null;size:255"`
	Value     string `json:"value" gorm:"type:text"`
	Type      string `json:"type" gorm:"size:50;default:'string'"` // string, number, boolean, list
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Task 任务表
type Task struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null;index"`
	Name      string `json:"name" gorm:"not null;size:255"`
	Module    string `json:"module" gorm:"not null;size:100"`
	Args      string `json:"args" gorm:"type:text"`
	OrderNum  int    `json:"order" gorm:"default:0"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// PlaybookGeneration 生成记录表
type PlaybookGeneration struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ProjectID  uint      `json:"project_id" gorm:"not null;index"`
	FileName   string    `json:"file_name" gorm:"not null;size:255"`
	FilePath   string    `json:"file_path" gorm:"not null;size:500"`
	Status     string    `json:"status" gorm:"size:50;default:'success'"` // success, failed
	Error      string    `json:"error,omitempty" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

// Kubernetes 集群管理模型

// KubernetesCluster Kubernetes集群表
type KubernetesCluster struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Name           string    `json:"name" gorm:"not null;size:255"`
	Description    string    `json:"description" gorm:"size:1000"`
	APIServer      string    `json:"api_server" gorm:"not null;size:500"`
	KubeConfig     string    `json:"kube_config" gorm:"type:text"`          // Kubeconfig内容
	KubeConfigPath string    `json:"kube_config_path" gorm:"size:500"`      // Kubeconfig文件路径
	Namespace      string    `json:"namespace" gorm:"size:255;default:'default'"`
	Status         string    `json:"status" gorm:"size:50;default:'unknown'"` // connected, disconnected, error, unknown
	Version        string    `json:"version" gorm:"size:100"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	IsActive       bool      `json:"is_active" gorm:"default:true"`
	LastCheckAt    *time.Time `json:"last_check_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	ID           uint      `json:"id" gorm:"primaryKey"`
	ProjectID    uint      `json:"project_id" gorm:"not null;index"`
	UserID       uint      `json:"user_id" gorm:"not null;index"`
	ExecutionType string   `json:"execution_type" gorm:"not null;size:50"` // dry-run, execute
	Environment   string   `json:"environment" gorm:"size:100"`            // dev, test, prod
	PlaybookPath  string   `json:"playbook_path" gorm:"not null;size:500"`
	InventoryPath string   `json:"inventory_path" gorm:"size:500"`
	ExtraVars     string   `json:"extra_vars" gorm:"type:text"`           // JSON格式的额外变量
	Status        string   `json:"status" gorm:"size:50;default:'pending'"` // pending, running, success, failed, cancelled
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Duration      int       `json:"duration"` // 执行时长（秒）
	Output        string    `json:"output" gorm:"type:text"` // 执行输出
	ErrorOutput   string    `json:"error_output" gorm:"type:text"` // 错误输出
	ExitCode      int       `json:"exit_code"`
	PID           int       `json:"pid"` // 进程ID
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Description string    `json:"description" gorm:"size:500"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联关系
	Users []User `json:"users,omitempty" gorm:"many2many:user_group_memberships"`
}

// Permission 权限表
type Permission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Resource    string    `json:"resource" gorm:"not null;size:100"` // projects, users, roles, etc.
	Verb        string    `json:"verb" gorm:"not null;size:50"`       // create, read, update, delete, list
	Scope       string    `json:"scope" gorm:"size:100;default:'*'"`  // * for all, specific resource ID
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
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Description string    `json:"description" gorm:"size:500"`
	IsSystem    bool      `json:"is_system" gorm:"default:false"` // 系统角色不可删除
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联关系
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions"`
	Users       []User       `json:"users,omitempty" gorm:"many2many:user_roles"`
	UserGroups  []UserGroup  `json:"user_groups,omitempty" gorm:"many2many:user_group_roles"`
}

// RoleBinding 角色绑定表 (类似K8s RoleBinding)
type RoleBinding struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null;size:100"`
	RoleID    uint      `json:"role_id" gorm:"not null;index"`
	Namespace string    `json:"namespace" gorm:"size:100;default:'default'"` // 命名空间概念
	
	// Subject 可以是用户或用户组
	SubjectType string `json:"subject_type" gorm:"not null;size:50"` // user, group
	SubjectID   uint   `json:"subject_id" gorm:"not null;index"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID             uint      `json:"id" gorm:"primaryKey"`
	Server         string    `json:"server" gorm:"not null;size:255"`
	Port           int       `json:"port" gorm:"default:389"`
	BindDN         string    `json:"bind_dn" gorm:"not null;size:500"`
	BindPassword   string    `json:"bind_password" gorm:"not null;size:255"`
	BaseDN         string    `json:"base_dn" gorm:"not null;size:500"`
	UserFilter     string    `json:"user_filter" gorm:"not null;size:500"`
	UsernameAttr   string    `json:"username_attr" gorm:"default:'uid';size:100"`
	EmailAttr      string    `json:"email_attr" gorm:"default:'mail';size:100"`
	NameAttr       string    `json:"name_attr" gorm:"default:'cn';size:100"`
	UseSSL         bool      `json:"use_ssl" gorm:"default:false"`
	SkipVerify     bool      `json:"skip_verify" gorm:"default:false"`
	IsEnabled      bool      `json:"is_enabled" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LDAPUser LDAP用户信息
type LDAPUser struct {
	DN          string `json:"dn"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// LDAPConfigRequest LDAP配置请求
type LDAPConfigRequest struct {
	Server       string `json:"server" binding:"required"`
	Port         int    `json:"port" binding:"required,min=1,max=65535"`
	BindDN       string `json:"bind_dn" binding:"required"`
	BindPassword string `json:"bind_password" binding:"required"`
	BaseDN       string `json:"base_dn" binding:"required"`
	UserFilter   string `json:"user_filter" binding:"required"`
	UsernameAttr string `json:"username_attr"`
	EmailAttr    string `json:"email_attr"`
	NameAttr     string `json:"name_attr"`
	UseSSL       bool   `json:"use_ssl"`
	SkipVerify   bool   `json:"skip_verify"`
	IsEnabled    bool   `json:"is_enabled"`
}

// LDAPTestRequest LDAP测试连接请求
type LDAPTestRequest struct {
	Server       string `json:"server" binding:"required"`
	Port         int    `json:"port" binding:"required"`
	BindDN       string `json:"bind_dn" binding:"required"`
	BindPassword string `json:"bind_password" binding:"required"`
	BaseDN       string `json:"base_dn" binding:"required"`
	UserFilter   string `json:"user_filter" binding:"required"`
	UseSSL       bool   `json:"use_ssl"`
	SkipVerify   bool   `json:"skip_verify"`
}

// 权限检查请求结构
type PermissionCheckRequest struct {
	Resource  string `json:"resource"`
	Verb      string `json:"verb"`
	Scope     string `json:"scope,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// 权限检查响应结构
type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// 角色分配请求结构
type RoleAssignmentRequest struct {
	SubjectType string `json:"subject_type" binding:"required"` // user, group
	SubjectID   uint   `json:"subject_id" binding:"required"`
	RoleID      uint   `json:"role_id" binding:"required"`
	Namespace   string `json:"namespace,omitempty"`
}

// 用户组创建请求结构
type CreateUserGroupRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description"`
}

// 角色创建请求结构
type CreateRoleRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=100"`
	Description  string `json:"description"`
	PermissionIDs []uint `json:"permission_ids"`
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
'''

file_path = '/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/apisChecker/deploybot/ansible-playbook-generator/web-v2/backend/internal/models/models.go'

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(clean_content)

print("干净的 models.go 文件已创建完成")
