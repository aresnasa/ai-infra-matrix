package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ==================== 审计日志常量定义 ====================

// AuditCategory 审计类别
type AuditCategory string

const (
	// 基础设施操作类别
	AuditCategoryAnsible      AuditCategory = "ansible"       // Ansible 相关操作
	AuditCategorySlurm        AuditCategory = "slurm"         // SLURM 相关操作
	AuditCategorySaltstack    AuditCategory = "saltstack"     // SaltStack 相关操作
	AuditCategoryRoleTemplate AuditCategory = "role_template" // 角色模板相关操作
	AuditCategoryKubernetes   AuditCategory = "kubernetes"    // Kubernetes 相关操作
	AuditCategoryMonitor      AuditCategory = "monitor"       // 监控相关操作
	AuditCategoryAdmin        AuditCategory = "admin"         // 管理员相关操作
	AuditCategorySecurity     AuditCategory = "security"      // 安全相关操作
	AuditCategorySystem       AuditCategory = "system"        // 系统相关操作
)

// AuditAction 审计动作类型
type AuditAction string

const (
	// 通用动作
	AuditActionCreate  AuditAction = "create"
	AuditActionUpdate  AuditAction = "update"
	AuditActionDelete  AuditAction = "delete"
	AuditActionRead    AuditAction = "read"
	AuditActionList    AuditAction = "list"
	AuditActionExecute AuditAction = "execute"
	AuditActionApprove AuditAction = "approve"
	AuditActionReject  AuditAction = "reject"
	AuditActionEnable  AuditAction = "enable"
	AuditActionDisable AuditAction = "disable"
	AuditActionImport  AuditAction = "import"
	AuditActionExport  AuditAction = "export"
	AuditActionSync    AuditAction = "sync"
	AuditActionDeploy  AuditAction = "deploy"
	AuditActionScale   AuditAction = "scale"
	AuditActionRestart AuditAction = "restart"
	AuditActionStop    AuditAction = "stop"
	AuditActionStart   AuditAction = "start"

	// Ansible 特定动作
	AuditActionPlaybookRun     AuditAction = "playbook_run"
	AuditActionPlaybookDryRun  AuditAction = "playbook_dry_run"
	AuditActionInventoryUpdate AuditAction = "inventory_update"

	// SaltStack 特定动作
	AuditActionSaltExecute      AuditAction = "salt_execute"
	AuditActionSaltStateApply   AuditAction = "salt_state_apply"
	AuditActionSaltPillarUpdate AuditAction = "salt_pillar_update"
	AuditActionSaltKeyAccept    AuditAction = "salt_key_accept"
	AuditActionSaltKeyReject    AuditAction = "salt_key_reject"
	AuditActionSaltKeyDelete    AuditAction = "salt_key_delete"

	// SLURM 特定动作
	AuditActionJobSubmit       AuditAction = "job_submit"
	AuditActionJobCancel       AuditAction = "job_cancel"
	AuditActionNodeAdd         AuditAction = "node_add"
	AuditActionNodeRemove      AuditAction = "node_remove"
	AuditActionNodeDrain       AuditAction = "node_drain"
	AuditActionNodeResume      AuditAction = "node_resume"
	AuditActionPartitionCreate AuditAction = "partition_create"
	AuditActionPartitionUpdate AuditAction = "partition_update"
	AuditActionClusterDeploy   AuditAction = "cluster_deploy"

	// Kubernetes 特定动作
	AuditActionK8sResourceCreate  AuditAction = "k8s_resource_create"
	AuditActionK8sResourceUpdate  AuditAction = "k8s_resource_update"
	AuditActionK8sResourceDelete  AuditAction = "k8s_resource_delete"
	AuditActionK8sNamespaceCreate AuditAction = "k8s_namespace_create"
	AuditActionK8sHelmInstall     AuditAction = "k8s_helm_install"
	AuditActionK8sHelmUpgrade     AuditAction = "k8s_helm_upgrade"
	AuditActionK8sHelmUninstall   AuditAction = "k8s_helm_uninstall"

	// 角色模板特定动作
	AuditActionRoleAssign       AuditAction = "role_assign"
	AuditActionRoleRevoke       AuditAction = "role_revoke"
	AuditActionPermissionGrant  AuditAction = "permission_grant"
	AuditActionPermissionRevoke AuditAction = "permission_revoke"

	// 监控特定动作
	AuditActionAlertCreate     AuditAction = "alert_create"
	AuditActionAlertAck        AuditAction = "alert_ack"
	AuditActionAlertResolve    AuditAction = "alert_resolve"
	AuditActionDashboardCreate AuditAction = "dashboard_create"
	AuditActionDashboardUpdate AuditAction = "dashboard_update"

	// Admin 特定动作
	AuditActionUserCreate    AuditAction = "user_create"
	AuditActionUserUpdate    AuditAction = "user_update"
	AuditActionUserDelete    AuditAction = "user_delete"
	AuditActionUserLock      AuditAction = "user_lock"
	AuditActionUserUnlock    AuditAction = "user_unlock"
	AuditActionPasswordReset AuditAction = "password_reset"
	AuditActionConfigUpdate  AuditAction = "config_update"
	AuditActionBackupCreate  AuditAction = "backup_create"
	AuditActionBackupRestore AuditAction = "backup_restore"
)

// AuditStatus 审计状态
type AuditStatus string

const (
	AuditStatusSuccess   AuditStatus = "success"
	AuditStatusFailed    AuditStatus = "failed"
	AuditStatusPending   AuditStatus = "pending"
	AuditStatusCancelled AuditStatus = "cancelled"
)

// AuditSeverity 审计严重程度
type AuditSeverity string

const (
	AuditSeverityInfo     AuditSeverity = "info"     // 信息级别，常规操作
	AuditSeverityWarning  AuditSeverity = "warning"  // 警告级别，需要注意
	AuditSeverityCritical AuditSeverity = "critical" // 关键级别，重要变更
	AuditSeverityAlert    AuditSeverity = "alert"    // 告警级别，需要立即关注
)

// ==================== 审计日志模型 ====================

// InfraAuditLog 基础设施审计日志（核心表）
type InfraAuditLog struct {
	ID       uint          `json:"id" gorm:"primaryKey"`
	TraceID  string        `json:"trace_id" gorm:"size:64;index"`                 // 追踪ID，用于关联多个审计记录
	Category AuditCategory `json:"category" gorm:"size:32;index;not null"`        // 审计类别
	Action   AuditAction   `json:"action" gorm:"size:64;index;not null"`          // 操作动作
	Status   AuditStatus   `json:"status" gorm:"size:32;index;default:'success'"` // 操作状态
	Severity AuditSeverity `json:"severity" gorm:"size:16;index;default:'info'"`  // 严重程度

	// 操作者信息
	UserID   uint   `json:"user_id" gorm:"index"`           // 操作用户ID
	Username string `json:"username" gorm:"size:100;index"` // 操作用户名
	UserRole string `json:"user_role" gorm:"size:50"`       // 用户角色

	// 资源信息
	ResourceType string `json:"resource_type" gorm:"size:100;index"` // 资源类型（如 cluster, node, job, role）
	ResourceID   string `json:"resource_id" gorm:"size:100;index"`   // 资源ID
	ResourceName string `json:"resource_name" gorm:"size:255"`       // 资源名称

	// 请求信息
	RequestMethod string `json:"request_method" gorm:"size:16"`   // HTTP方法
	RequestPath   string `json:"request_path" gorm:"size:500"`    // 请求路径
	RequestParams string `json:"request_params" gorm:"type:text"` // 请求参数（JSON）

	// 变更详情
	OldValue      string `json:"old_value" gorm:"type:text"`     // 变更前的值（JSON）
	NewValue      string `json:"new_value" gorm:"type:text"`     // 变更后的值（JSON）
	ChangeSummary string `json:"change_summary" gorm:"size:500"` // 变更摘要

	// 客户端信息
	ClientIP  string `json:"client_ip" gorm:"size:64;index"` // 客户端IP
	UserAgent string `json:"user_agent" gorm:"size:256"`     // User-Agent
	SessionID string `json:"session_id" gorm:"size:64"`      // 会话ID

	// 执行信息
	ExecutionTime int64  `json:"execution_time"`                 // 执行耗时（毫秒）
	ErrorMessage  string `json:"error_message" gorm:"type:text"` // 错误信息
	StackTrace    string `json:"stack_trace" gorm:"type:text"`   // 堆栈追踪（失败时）

	// 环境信息
	Environment string `json:"environment" gorm:"size:32"` // 环境（dev/test/prod）
	HostName    string `json:"host_name" gorm:"size:100"`  // 服务器主机名

	// 额外信息
	Metadata string `json:"metadata" gorm:"type:text"` // 额外元数据（JSON）
	Tags     string `json:"tags" gorm:"size:500"`      // 标签（逗号分隔）
	Notes    string `json:"notes" gorm:"type:text"`    // 备注

	// 时间戳
	CreatedAt time.Time `json:"created_at" gorm:"index"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (InfraAuditLog) TableName() string {
	return "infra_audit_logs"
}

// AuditChangeDetail 审计变更明细
type AuditChangeDetail struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	AuditLogID uint      `json:"audit_log_id" gorm:"index;not null"`  // 关联审计日志ID
	FieldName  string    `json:"field_name" gorm:"size:100;not null"` // 变更字段名
	FieldPath  string    `json:"field_path" gorm:"size:255"`          // 字段路径（嵌套结构）
	OldValue   string    `json:"old_value" gorm:"type:text"`          // 变更前的值
	NewValue   string    `json:"new_value" gorm:"type:text"`          // 变更后的值
	ChangeType string    `json:"change_type" gorm:"size:32"`          // 变更类型：add, modify, delete
	CreatedAt  time.Time `json:"created_at"`
}

// TableName 指定表名
func (AuditChangeDetail) TableName() string {
	return "audit_change_details"
}

// AuditArchive 审计日志归档
type AuditArchive struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	ArchiveDate   time.Time      `json:"archive_date" gorm:"index"`     // 归档日期
	Category      AuditCategory  `json:"category" gorm:"size:32;index"` // 审计类别
	RecordCount   int64          `json:"record_count"`                  // 记录数量
	FilePath      string         `json:"file_path" gorm:"size:500"`     // 归档文件路径
	FileSize      int64          `json:"file_size"`                     // 文件大小（字节）
	Checksum      string         `json:"checksum" gorm:"size:64"`       // 文件校验和
	RetentionDays int            `json:"retention_days"`                // 保留天数
	ExpiresAt     time.Time      `json:"expires_at" gorm:"index"`       // 过期时间
	Status        string         `json:"status" gorm:"size:32"`         // 状态：pending, completed, failed
	CreatedBy     uint           `json:"created_by"`                    // 创建人ID
	CreatedAt     time.Time      `json:"created_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (AuditArchive) TableName() string {
	return "audit_archives"
}

// AuditConfig 审计配置
type AuditConfig struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	Category           AuditCategory  `json:"category" gorm:"size:32;uniqueIndex;not null"` // 审计类别
	Enabled            bool           `json:"enabled" gorm:"default:true"`                  // 是否启用
	LogLevel           string         `json:"log_level" gorm:"size:16;default:'info'"`      // 日志级别
	RetentionDays      int            `json:"retention_days" gorm:"default:90"`             // 保留天数
	ArchiveEnabled     bool           `json:"archive_enabled" gorm:"default:true"`          // 是否启用归档
	ArchiveAfterDays   int            `json:"archive_after_days" gorm:"default:30"`         // 多少天后归档
	NotifyEnabled      bool           `json:"notify_enabled" gorm:"default:false"`          // 是否启用通知
	NotifyOn           string         `json:"notify_on" gorm:"size:100"`                    // 触发通知的动作（逗号分隔）
	NotifyChannels     string         `json:"notify_channels" gorm:"size:100"`              // 通知渠道（email,webhook,slack）
	NotifyWebhookURL   string         `json:"notify_webhook_url" gorm:"size:500"`           // Webhook URL
	ExcludeActions     string         `json:"exclude_actions" gorm:"type:text"`             // 排除的动作（逗号分隔）
	ExcludeUsers       string         `json:"exclude_users" gorm:"type:text"`               // 排除的用户（逗号分隔）
	SensitiveFieldMask string         `json:"sensitive_field_mask" gorm:"type:text"`        // 敏感字段掩码规则
	Description        string         `json:"description" gorm:"size:500"`                  // 描述
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (AuditConfig) TableName() string {
	return "audit_configs"
}

// AuditExportRequest 审计导出请求
type AuditExportRequest struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	RequestedBy   uint           `json:"requested_by" gorm:"index"`       // 请求用户ID
	Category      AuditCategory  `json:"category" gorm:"size:32"`         // 审计类别（空表示全部）
	StartDate     time.Time      `json:"start_date"`                      // 开始日期
	EndDate       time.Time      `json:"end_date"`                        // 结束日期
	Format        string         `json:"format" gorm:"size:16"`           // 导出格式：csv, json, xlsx
	Status        string         `json:"status" gorm:"size:32;index"`     // 状态：pending, processing, completed, failed
	FilePath      string         `json:"file_path" gorm:"size:500"`       // 生成的文件路径
	FileSize      int64          `json:"file_size"`                       // 文件大小
	RecordCount   int64          `json:"record_count"`                    // 导出记录数
	ErrorMessage  string         `json:"error_message" gorm:"type:text"`  // 错误信息
	ExpiresAt     time.Time      `json:"expires_at"`                      // 文件过期时间
	DownloadCount int            `json:"download_count" gorm:"default:0"` // 下载次数
	CreatedAt     time.Time      `json:"created_at"`
	CompletedAt   *time.Time     `json:"completed_at"` // 完成时间
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (AuditExportRequest) TableName() string {
	return "audit_export_requests"
}

// ==================== 请求/响应结构体 ====================

// CreateAuditLogRequest 创建审计日志请求
type CreateAuditLogRequest struct {
	Category      AuditCategory `json:"category" binding:"required"`
	Action        AuditAction   `json:"action" binding:"required"`
	Status        AuditStatus   `json:"status"`
	Severity      AuditSeverity `json:"severity"`
	ResourceType  string        `json:"resource_type"`
	ResourceID    string        `json:"resource_id"`
	ResourceName  string        `json:"resource_name"`
	OldValue      interface{}   `json:"old_value"`
	NewValue      interface{}   `json:"new_value"`
	ChangeSummary string        `json:"change_summary"`
	ErrorMessage  string        `json:"error_message"`
	Metadata      interface{}   `json:"metadata"`
	Tags          []string      `json:"tags"`
	Notes         string        `json:"notes"`
}

// AuditLogQueryRequest 审计日志查询请求
type AuditLogQueryRequest struct {
	Category     string    `form:"category"`      // 审计类别
	Action       string    `form:"action"`        // 操作动作
	Status       string    `form:"status"`        // 状态
	Severity     string    `form:"severity"`      // 严重程度
	UserID       uint      `form:"user_id"`       // 用户ID
	Username     string    `form:"username"`      // 用户名
	ResourceType string    `form:"resource_type"` // 资源类型
	ResourceID   string    `form:"resource_id"`   // 资源ID
	ClientIP     string    `form:"client_ip"`     // 客户端IP
	StartDate    time.Time `form:"start_date" time_format:"2006-01-02"`
	EndDate      time.Time `form:"end_date" time_format:"2006-01-02"`
	Keywords     string    `form:"keywords"` // 关键词搜索
	Page         int       `form:"page" binding:"omitempty,min=1"`
	PageSize     int       `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy       string    `form:"sort_by"`    // 排序字段
	SortOrder    string    `form:"sort_order"` // 排序方向 asc/desc
}

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
	Data       []InfraAuditLog `json:"data"`
}

// AuditStatisticsResponse 审计统计响应
type AuditStatisticsResponse struct {
	TotalLogs     int64              `json:"total_logs"`
	TodayLogs     int64              `json:"today_logs"`
	SuccessCount  int64              `json:"success_count"`
	FailedCount   int64              `json:"failed_count"`
	CategoryStats []CategoryStatItem `json:"category_stats"`
	ActionStats   []ActionStatItem   `json:"action_stats"`
	UserStats     []UserStatItem     `json:"user_stats"`
	TrendData     []TrendDataItem    `json:"trend_data"`
}

// CategoryStatItem 类别统计项
type CategoryStatItem struct {
	Category AuditCategory `json:"category"`
	Count    int64         `json:"count"`
}

// ActionStatItem 动作统计项
type ActionStatItem struct {
	Action AuditAction `json:"action"`
	Count  int64       `json:"count"`
}

// UserStatItem 用户统计项
type UserStatItem struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Count    int64  `json:"count"`
}

// TrendDataItem 趋势数据项
type TrendDataItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// AuditExportRequestCreate 创建导出请求
type AuditExportRequestCreate struct {
	Category  AuditCategory `json:"category"`
	StartDate time.Time     `json:"start_date" binding:"required"`
	EndDate   time.Time     `json:"end_date" binding:"required"`
	Format    string        `json:"format" binding:"required,oneof=csv json xlsx"`
}

// ==================== 辅助方法 ====================

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// GetCategoryDisplayName 获取类别的显示名称
func (c AuditCategory) GetDisplayName() string {
	names := map[AuditCategory]string{
		AuditCategoryAnsible:      "Ansible",
		AuditCategorySlurm:        "SLURM",
		AuditCategorySaltstack:    "SaltStack",
		AuditCategoryRoleTemplate: "角色模板",
		AuditCategoryKubernetes:   "Kubernetes",
		AuditCategoryMonitor:      "监控",
		AuditCategoryAdmin:        "管理员操作",
		AuditCategorySecurity:     "安全",
		AuditCategorySystem:       "系统",
	}
	if name, ok := names[c]; ok {
		return name
	}
	return string(c)
}

// GetActionDisplayName 获取动作的显示名称
func (a AuditAction) GetDisplayName() string {
	names := map[AuditAction]string{
		AuditActionCreate:           "创建",
		AuditActionUpdate:           "更新",
		AuditActionDelete:           "删除",
		AuditActionRead:             "查看",
		AuditActionList:             "列表",
		AuditActionExecute:          "执行",
		AuditActionPlaybookRun:      "运行Playbook",
		AuditActionPlaybookDryRun:   "Playbook预演",
		AuditActionSaltExecute:      "Salt命令执行",
		AuditActionSaltStateApply:   "Salt状态应用",
		AuditActionSaltKeyAccept:    "接受Salt Key",
		AuditActionSaltKeyReject:    "拒绝Salt Key",
		AuditActionSaltKeyDelete:    "删除Salt Key",
		AuditActionJobSubmit:        "提交作业",
		AuditActionJobCancel:        "取消作业",
		AuditActionNodeAdd:          "添加节点",
		AuditActionNodeRemove:       "移除节点",
		AuditActionRoleAssign:       "分配角色",
		AuditActionRoleRevoke:       "撤销角色",
		AuditActionPermissionGrant:  "授予权限",
		AuditActionPermissionRevoke: "撤销权限",
		AuditActionUserCreate:       "创建用户",
		AuditActionUserUpdate:       "更新用户",
		AuditActionUserDelete:       "删除用户",
		AuditActionPasswordReset:    "重置密码",
		AuditActionConfigUpdate:     "更新配置",
	}
	if name, ok := names[a]; ok {
		return name
	}
	return string(a)
}

// GetSeverityLevel 获取严重程度级别数字（用于排序）
func (s AuditSeverity) GetLevel() int {
	levels := map[AuditSeverity]int{
		AuditSeverityInfo:     1,
		AuditSeverityWarning:  2,
		AuditSeverityCritical: 3,
		AuditSeverityAlert:    4,
	}
	if level, ok := levels[s]; ok {
		return level
	}
	return 0
}

// GetDefaultAuditConfigs 获取默认审计配置
func GetDefaultAuditConfigs() []AuditConfig {
	return []AuditConfig{
		{
			Category:         AuditCategoryAnsible,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    90,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 30,
			NotifyEnabled:    false,
			Description:      "Ansible 自动化操作审计",
		},
		{
			Category:         AuditCategorySlurm,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    90,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 30,
			NotifyEnabled:    false,
			Description:      "SLURM 集群操作审计",
		},
		{
			Category:         AuditCategorySaltstack,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    90,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 30,
			NotifyEnabled:    true,
			NotifyOn:         "salt_state_apply,salt_key_delete",
			Description:      "SaltStack 配置管理审计",
		},
		{
			Category:         AuditCategoryRoleTemplate,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    180,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 60,
			NotifyEnabled:    true,
			NotifyOn:         "role_assign,role_revoke,permission_grant,permission_revoke",
			Description:      "角色模板和权限变更审计",
		},
		{
			Category:         AuditCategoryKubernetes,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    90,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 30,
			NotifyEnabled:    false,
			Description:      "Kubernetes 资源操作审计",
		},
		{
			Category:         AuditCategoryMonitor,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    60,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 30,
			NotifyEnabled:    false,
			Description:      "监控配置操作审计",
		},
		{
			Category:         AuditCategoryAdmin,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    365,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 90,
			NotifyEnabled:    true,
			NotifyOn:         "user_create,user_delete,password_reset,config_update",
			Description:      "管理员操作审计（保留时间最长）",
		},
		{
			Category:         AuditCategorySecurity,
			Enabled:          true,
			LogLevel:         "info",
			RetentionDays:    365,
			ArchiveEnabled:   true,
			ArchiveAfterDays: 90,
			NotifyEnabled:    true,
			NotifyOn:         "*",
			Description:      "安全相关操作审计",
		},
	}
}
