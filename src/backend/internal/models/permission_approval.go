package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// PermissionApprovalStatus 权限审批状态
type PermissionApprovalStatus string

const (
	ApprovalStatusPending  PermissionApprovalStatus = "pending"  // 待审批
	ApprovalStatusApproved PermissionApprovalStatus = "approved" // 已批准
	ApprovalStatusRejected PermissionApprovalStatus = "rejected" // 已拒绝
	ApprovalStatusCanceled PermissionApprovalStatus = "canceled" // 已取消
	ApprovalStatusExpired  PermissionApprovalStatus = "expired"  // 已过期
)

// PermissionModuleType 权限模块类型
type PermissionModuleType string

const (
	ModuleSaltStack     PermissionModuleType = "saltstack"
	ModuleAnsible       PermissionModuleType = "ansible"
	ModuleKubernetes    PermissionModuleType = "kubernetes"
	ModuleJupyterHub    PermissionModuleType = "jupyterhub"
	ModuleSlurm         PermissionModuleType = "slurm"
	ModuleGitea         PermissionModuleType = "gitea"
	ModuleKafkaUI       PermissionModuleType = "kafka-ui"
	ModuleNightingale   PermissionModuleType = "nightingale"
	ModuleProjects      PermissionModuleType = "projects"
	ModuleHosts         PermissionModuleType = "hosts"
	ModuleUsers         PermissionModuleType = "users"
	ModuleRoles         PermissionModuleType = "roles"
	ModuleObjectStorage PermissionModuleType = "object-storage"
	ModuleAuditLogs     PermissionModuleType = "audit-logs"
	ModuleAIChat        PermissionModuleType = "ai-chat"
)

// PermissionRequest 权限申请记录表
type PermissionRequest struct {
	ID               uint                     `json:"id" gorm:"primaryKey"`
	RequesterID      uint                     `json:"requester_id" gorm:"not null;index"`      // 申请人ID
	TargetUserID     uint                     `json:"target_user_id" gorm:"index"`             // 目标用户ID（可为空，默认为申请人自己）
	RoleTemplateName string                   `json:"role_template_name" gorm:"size:100"`      // 申请的角色模板名称
	RequestedModules json.RawMessage          `json:"requested_modules" gorm:"type:jsonb"`     // 申请的模块列表 (JSON数组)
	RequestedVerbs   json.RawMessage          `json:"requested_verbs" gorm:"type:jsonb"`       // 申请的操作权限 (JSON数组)
	Reason           string                   `json:"reason" gorm:"type:text"`                 // 申请理由
	Status           PermissionApprovalStatus `json:"status" gorm:"size:20;default:'pending'"` // 审批状态
	Priority         int                      `json:"priority" gorm:"default:0"`               // 优先级 (0=普通, 1=紧急, 2=非常紧急)
	ExpiresAt        *time.Time               `json:"expires_at,omitempty"`                    // 权限过期时间（可选）
	ValidDays        int                      `json:"valid_days" gorm:"default:0"`             // 权限有效天数（0=永久）
	ApproverID       *uint                    `json:"approver_id,omitempty" gorm:"index"`      // 审批人ID
	ApproveComment   string                   `json:"approve_comment" gorm:"type:text"`        // 审批意见
	ApprovedAt       *time.Time               `json:"approved_at,omitempty"`                   // 审批时间
	AutoApprove      bool                     `json:"auto_approve" gorm:"default:false"`       // 是否自动审批
	NotifyEmail      bool                     `json:"notify_email" gorm:"default:true"`        // 是否邮件通知
	NotifyDingTalk   bool                     `json:"notify_dingtalk" gorm:"default:false"`    // 是否钉钉通知
	RelatedTicket    string                   `json:"related_ticket" gorm:"size:100"`          // 关联工单号
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	DeletedAt        gorm.DeletedAt           `json:"-" gorm:"index"`

	// 关联关系
	Requester  User  `json:"requester,omitempty" gorm:"foreignKey:RequesterID"`
	TargetUser *User `json:"target_user,omitempty" gorm:"foreignKey:TargetUserID"`
	Approver   *User `json:"approver,omitempty" gorm:"foreignKey:ApproverID"`
}

// PermissionApprovalLog 权限审批日志表
type PermissionApprovalLog struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	PermissionRequestID uint      `json:"permission_request_id" gorm:"not null;index"` // 关联的申请记录ID
	OperatorID          uint      `json:"operator_id" gorm:"not null;index"`           // 操作人ID
	Action              string    `json:"action" gorm:"size:50;not null"`              // 操作类型: submit, approve, reject, cancel, expire
	OldStatus           string    `json:"old_status" gorm:"size:20"`                   // 原状态
	NewStatus           string    `json:"new_status" gorm:"size:20"`                   // 新状态
	Comment             string    `json:"comment" gorm:"type:text"`                    // 操作备注
	IPAddress           string    `json:"ip_address" gorm:"size:45"`                   // 操作人IP
	UserAgent           string    `json:"user_agent" gorm:"size:500"`                  // 浏览器UA
	CreatedAt           time.Time `json:"created_at"`

	// 关联关系
	PermissionRequest PermissionRequest `json:"permission_request,omitempty" gorm:"foreignKey:PermissionRequestID"`
	Operator          User              `json:"operator,omitempty" gorm:"foreignKey:OperatorID"`
}

// PermissionApprovalRule 权限审批规则表（自动化审批配置）
type PermissionApprovalRule struct {
	ID                uint            `json:"id" gorm:"primaryKey"`
	Name              string          `json:"name" gorm:"not null;size:100"`        // 规则名称
	Description       string          `json:"description" gorm:"size:500"`          // 规则描述
	IsActive          bool            `json:"is_active" gorm:"default:true"`        // 是否启用
	Priority          int             `json:"priority" gorm:"default:0"`            // 规则优先级（数字越大优先级越高）
	ConditionType     string          `json:"condition_type" gorm:"size:50"`        // 条件类型: role_template, module, user_group
	ConditionValue    json.RawMessage `json:"condition_value" gorm:"type:jsonb"`    // 条件值（JSON格式）
	AutoApprove       bool            `json:"auto_approve" gorm:"default:false"`    // 是否自动审批
	RequiredApprovers json.RawMessage `json:"required_approvers" gorm:"type:jsonb"` // 必须的审批人ID列表（JSON数组）
	MinApprovals      int             `json:"min_approvals" gorm:"default:1"`       // 最少需要的审批数
	MaxValidDays      int             `json:"max_valid_days" gorm:"default:0"`      // 最大有效天数（0=无限制）
	AllowedModules    json.RawMessage `json:"allowed_modules" gorm:"type:jsonb"`    // 允许的模块列表
	NotifyAdmins      bool            `json:"notify_admins" gorm:"default:true"`    // 是否通知管理员
	CreatedBy         uint            `json:"created_by" gorm:"not null"`           // 创建人ID
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `json:"-" gorm:"index"`

	// 关联关系
	Creator User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// PermissionGrant 已授权权限记录表（跟踪已批准的权限）
type PermissionGrant struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	UserID              uint           `json:"user_id" gorm:"not null;index"`                // 被授权用户ID
	PermissionRequestID *uint          `json:"permission_request_id" gorm:"index"`           // 关联的申请记录ID（可为空，手动授权时为空）
	Module              string         `json:"module" gorm:"not null;size:50"`               // 模块名称
	Resource            string         `json:"resource" gorm:"size:100"`                     // 资源名称
	Verb                string         `json:"verb" gorm:"not null;size:50"`                 // 操作动词
	Scope               string         `json:"scope" gorm:"size:100;default:'*'"`            // 作用域
	GrantType           string         `json:"grant_type" gorm:"size:20;default:'approval'"` // 授权类型: approval, manual, auto
	GrantedBy           uint           `json:"granted_by" gorm:"not null;index"`             // 授权人ID
	Reason              string         `json:"reason" gorm:"size:500"`                       // 授权理由
	ExpiresAt           *time.Time     `json:"expires_at,omitempty"`                         // 过期时间
	IsActive            bool           `json:"is_active" gorm:"default:true"`                // 是否有效
	RevokedAt           *time.Time     `json:"revoked_at,omitempty"`                         // 撤销时间
	RevokedBy           *uint          `json:"revoked_by,omitempty"`                         // 撤销人ID
	RevokeReason        string         `json:"revoke_reason" gorm:"size:500"`                // 撤销原因
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User              User               `json:"user,omitempty" gorm:"foreignKey:UserID"`
	PermissionRequest *PermissionRequest `json:"permission_request,omitempty" gorm:"foreignKey:PermissionRequestID"`
	Grantor           User               `json:"grantor,omitempty" gorm:"foreignKey:GrantedBy"`
}

// ====================
// 请求和响应结构
// ====================

// CreatePermissionRequestInput 创建权限申请请求
type CreatePermissionRequestInput struct {
	TargetUserID     uint     `json:"target_user_id"`                             // 目标用户ID（可选，默认为当前用户）
	RoleTemplateName string   `json:"role_template_name" binding:"omitempty"`     // 角色模板名称
	RequestedModules []string `json:"requested_modules" binding:"required,min=1"` // 申请的模块列表
	RequestedVerbs   []string `json:"requested_verbs" binding:"required,min=1"`   // 申请的操作权限
	Reason           string   `json:"reason" binding:"required,min=10,max=1000"`  // 申请理由
	Priority         int      `json:"priority" binding:"omitempty,min=0,max=2"`   // 优先级
	ValidDays        int      `json:"valid_days" binding:"omitempty,min=0"`       // 有效天数
	NotifyEmail      bool     `json:"notify_email"`                               // 邮件通知
	NotifyDingTalk   bool     `json:"notify_dingtalk"`                            // 钉钉通知
	RelatedTicket    string   `json:"related_ticket" binding:"omitempty,max=100"` // 关联工单号
}

// ApprovePermissionRequestInput 审批权限申请请求
type ApprovePermissionRequestInput struct {
	Approved bool   `json:"approved" binding:"required"`         // 是否批准
	Comment  string `json:"comment" binding:"omitempty,max=500"` // 审批意见
}

// PermissionRequestQuery 权限申请查询参数
type PermissionRequestQuery struct {
	Status      string `form:"status" binding:"omitempty"`          // 状态筛选
	RequesterID uint   `form:"requester_id" binding:"omitempty"`    // 申请人筛选
	ApproverID  uint   `form:"approver_id" binding:"omitempty"`     // 审批人筛选
	Module      string `form:"module" binding:"omitempty"`          // 模块筛选
	Priority    int    `form:"priority" binding:"omitempty"`        // 优先级筛选
	StartDate   string `form:"start_date" binding:"omitempty"`      // 开始日期
	EndDate     string `form:"end_date" binding:"omitempty"`        // 结束日期
	Page        int    `form:"page" binding:"omitempty,min=1"`      // 页码
	PageSize    int    `form:"page_size" binding:"omitempty,min=1"` // 每页数量
	SortBy      string `form:"sort_by" binding:"omitempty"`         // 排序字段
	SortOrder   string `form:"sort_order" binding:"omitempty"`      // 排序方向
	OnlyPending bool   `form:"only_pending" binding:"omitempty"`    // 只查询待审批
}

// PermissionRequestListResponse 权限申请列表响应
type PermissionRequestListResponse struct {
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Items    []PermissionRequest `json:"items"`
}

// CreateApprovalRuleInput 创建审批规则请求
type CreateApprovalRuleInput struct {
	Name              string   `json:"name" binding:"required,min=2,max=100"`
	Description       string   `json:"description" binding:"omitempty,max=500"`
	IsActive          bool     `json:"is_active"`
	Priority          int      `json:"priority" binding:"omitempty,min=0"`
	ConditionType     string   `json:"condition_type" binding:"required,oneof=role_template module user_group"`
	ConditionValue    []string `json:"condition_value" binding:"required,min=1"`
	AutoApprove       bool     `json:"auto_approve"`
	RequiredApprovers []uint   `json:"required_approvers" binding:"omitempty"`
	MinApprovals      int      `json:"min_approvals" binding:"omitempty,min=1"`
	MaxValidDays      int      `json:"max_valid_days" binding:"omitempty,min=0"`
	AllowedModules    []string `json:"allowed_modules" binding:"omitempty"`
	NotifyAdmins      bool     `json:"notify_admins"`
}

// GrantPermissionInput 手动授权权限请求
type GrantPermissionInput struct {
	UserID    uint     `json:"user_id" binding:"required"`              // 目标用户ID
	Modules   []string `json:"modules" binding:"required,min=1"`        // 模块列表
	Verbs     []string `json:"verbs" binding:"required,min=1"`          // 操作权限
	Reason    string   `json:"reason" binding:"required,min=5,max=500"` // 授权理由
	ValidDays int      `json:"valid_days" binding:"omitempty,min=0"`    // 有效天数（0=永久）
}

// RevokePermissionInput 撤销权限请求
type RevokePermissionInput struct {
	UserID uint   `json:"user_id" binding:"required"`              // 目标用户ID
	Module string `json:"module" binding:"required"`               // 模块名称
	Reason string `json:"reason" binding:"required,min=5,max=500"` // 撤销理由
}

// PermissionModuleInfo 权限模块信息
type PermissionModuleInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Verbs       []string `json:"verbs"`
}

// PermissionStats 权限统计信息
type PermissionStats struct {
	TotalRequests    int64 `json:"total_requests"`
	PendingRequests  int64 `json:"pending_requests"`
	ApprovedRequests int64 `json:"approved_requests"`
	RejectedRequests int64 `json:"rejected_requests"`
	TotalGrants      int64 `json:"total_grants"`
	ActiveGrants     int64 `json:"active_grants"`
	ExpiredGrants    int64 `json:"expired_grants"`
}

// ====================
// 辅助方法
// ====================

// GetRequestedModules 获取申请的模块列表
func (p *PermissionRequest) GetRequestedModules() []string {
	var modules []string
	if p.RequestedModules != nil {
		json.Unmarshal(p.RequestedModules, &modules)
	}
	return modules
}

// GetRequestedVerbs 获取申请的操作权限
func (p *PermissionRequest) GetRequestedVerbs() []string {
	var verbs []string
	if p.RequestedVerbs != nil {
		json.Unmarshal(p.RequestedVerbs, &verbs)
	}
	return verbs
}

// SetRequestedModules 设置申请的模块列表
func (p *PermissionRequest) SetRequestedModules(modules []string) error {
	data, err := json.Marshal(modules)
	if err != nil {
		return err
	}
	p.RequestedModules = data
	return nil
}

// SetRequestedVerbs 设置申请的操作权限
func (p *PermissionRequest) SetRequestedVerbs(verbs []string) error {
	data, err := json.Marshal(verbs)
	if err != nil {
		return err
	}
	p.RequestedVerbs = data
	return nil
}

// IsPending 是否待审批
func (p *PermissionRequest) IsPending() bool {
	return p.Status == ApprovalStatusPending
}

// IsApproved 是否已批准
func (p *PermissionRequest) IsApproved() bool {
	return p.Status == ApprovalStatusApproved
}

// IsRejected 是否已拒绝
func (p *PermissionRequest) IsRejected() bool {
	return p.Status == ApprovalStatusRejected
}

// CanBeApproved 是否可以被审批
func (p *PermissionRequest) CanBeApproved() bool {
	return p.Status == ApprovalStatusPending
}

// IsExpired 检查权限是否已过期
func (g *PermissionGrant) IsExpired() bool {
	if g.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*g.ExpiresAt)
}

// GetConditionValue 获取条件值
func (r *PermissionApprovalRule) GetConditionValue() []string {
	var values []string
	if r.ConditionValue != nil {
		json.Unmarshal(r.ConditionValue, &values)
	}
	return values
}

// GetRequiredApprovers 获取必须的审批人
func (r *PermissionApprovalRule) GetRequiredApprovers() []uint {
	var approvers []uint
	if r.RequiredApprovers != nil {
		json.Unmarshal(r.RequiredApprovers, &approvers)
	}
	return approvers
}

// GetAllowedModules 获取允许的模块
func (r *PermissionApprovalRule) GetAllowedModules() []string {
	var modules []string
	if r.AllowedModules != nil {
		json.Unmarshal(r.AllowedModules, &modules)
	}
	return modules
}
