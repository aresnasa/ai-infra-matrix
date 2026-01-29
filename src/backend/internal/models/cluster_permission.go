package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ClusterPermissionType 集群权限类型
type ClusterPermissionType string

const (
	PermissionTypeSlurmCluster     ClusterPermissionType = "slurm_cluster"
	PermissionTypeSlurmPartition   ClusterPermissionType = "slurm_partition"
	PermissionTypeSaltstackCluster ClusterPermissionType = "saltstack_cluster"
	PermissionTypeSaltstackMinion  ClusterPermissionType = "saltstack_minion"
)

// ClusterPermissionVerb 集群权限动作
type ClusterPermissionVerb string

const (
	VerbView    ClusterPermissionVerb = "view"    // 查看
	VerbSubmit  ClusterPermissionVerb = "submit"  // 提交任务
	VerbManage  ClusterPermissionVerb = "manage"  // 管理（启动/停止/配置）
	VerbAdmin   ClusterPermissionVerb = "admin"   // 管理员（包含所有权限）
	VerbExecute ClusterPermissionVerb = "execute" // 执行命令（SaltStack）
	VerbMonitor ClusterPermissionVerb = "monitor" // 监控
	VerbScale   ClusterPermissionVerb = "scale"   // 扩缩容
	VerbConnect ClusterPermissionVerb = "connect" // 连接/SSH
)

// ========================================
// SLURM 集群权限模型
// ========================================

// SlurmClusterPermission SLURM集群权限
type SlurmClusterPermission struct {
	ID            uint             `json:"id" gorm:"primaryKey"`
	UserID        uint             `json:"user_id" gorm:"not null;index"`
	ClusterID     uint             `json:"cluster_id" gorm:"not null;index"`
	Verbs         StringArray      `json:"verbs" gorm:"type:json;not null"`     // 允许的操作 [view, submit, manage, admin]
	AllPartitions bool             `json:"all_partitions" gorm:"default:false"` // 是否有所有分区权限
	Partitions    StringArray      `json:"partitions" gorm:"type:json"`         // 允许访问的分区列表
	MaxJobs       int              `json:"max_jobs" gorm:"default:0"`           // 最大并发任务数(0=无限制)
	MaxCPUs       int              `json:"max_cpus" gorm:"default:0"`           // 最大CPU核心数(0=无限制)
	MaxGPUs       int              `json:"max_gpus" gorm:"default:0"`           // 最大GPU数量(0=无限制)
	MaxMemoryGB   int              `json:"max_memory_gb" gorm:"default:0"`      // 最大内存GB(0=无限制)
	MaxWalltime   string           `json:"max_walltime" gorm:"size:50"`         // 最大运行时间(如 "24:00:00")
	Priority      int              `json:"priority" gorm:"default:100"`         // 任务优先级调整
	QOS           string           `json:"qos" gorm:"size:100"`                 // Quality of Service
	Account       string           `json:"account" gorm:"size:100"`             // SLURM账户名
	Constraints   SlurmConstraints `json:"constraints" gorm:"type:json"`        // 其他约束
	GrantedBy     uint             `json:"granted_by" gorm:"not null"`          // 授权人ID
	GrantReason   string           `json:"grant_reason" gorm:"size:500"`        // 授权原因
	ExpiresAt     *time.Time       `json:"expires_at,omitempty"`                // 过期时间
	IsActive      bool             `json:"is_active" gorm:"default:true"`       // 是否有效
	RevokedAt     *time.Time       `json:"revoked_at,omitempty"`                // 撤销时间
	RevokedBy     *uint            `json:"revoked_by,omitempty"`                // 撤销人ID
	RevokeReason  string           `json:"revoke_reason" gorm:"size:500"`       // 撤销原因
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	DeletedAt     gorm.DeletedAt   `json:"-" gorm:"index"`

	// 关联关系
	User    User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Cluster SlurmCluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	Grantor User         `json:"grantor,omitempty" gorm:"foreignKey:GrantedBy"`
}

// SlurmConstraints SLURM约束配置
type SlurmConstraints struct {
	AllowedNodes    []string          `json:"allowed_nodes,omitempty"`    // 允许使用的节点
	ExcludedNodes   []string          `json:"excluded_nodes,omitempty"`   // 排除的节点
	AllowedFeatures []string          `json:"allowed_features,omitempty"` // 允许的特性
	RequiredGres    []string          `json:"required_gres,omitempty"`    // 必需的GRES
	CustomFlags     map[string]string `json:"custom_flags,omitempty"`     // 自定义标志
}

// Scan 实现 database/sql.Scanner 接口
func (sc *SlurmConstraints) Scan(value interface{}) error {
	if value == nil {
		*sc = SlurmConstraints{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan SlurmConstraints: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, sc)
}

// Value 实现 database/sql/driver.Valuer 接口
func (sc SlurmConstraints) Value() (driver.Value, error) {
	return json.Marshal(sc)
}

// TableName 指定表名
func (SlurmClusterPermission) TableName() string {
	return "slurm_cluster_permissions"
}

// HasVerb 检查是否有指定权限
func (p *SlurmClusterPermission) HasVerb(verb ClusterPermissionVerb) bool {
	if p.Verbs == nil {
		return false
	}
	// admin权限包含所有其他权限
	for _, v := range p.Verbs {
		if ClusterPermissionVerb(v) == VerbAdmin || ClusterPermissionVerb(v) == verb {
			return true
		}
	}
	return false
}

// HasPartitionAccess 检查是否有分区访问权限
func (p *SlurmClusterPermission) HasPartitionAccess(partition string) bool {
	if p.AllPartitions {
		return true
	}
	if p.Partitions == nil {
		return false
	}
	for _, part := range p.Partitions {
		if part == partition || part == "*" {
			return true
		}
	}
	return false
}

// IsExpired 检查权限是否过期
func (p *SlurmClusterPermission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsValid 检查权限是否有效
func (p *SlurmClusterPermission) IsValid() bool {
	return p.IsActive && !p.IsExpired() && p.RevokedAt == nil
}

// ========================================
// SaltStack 集群权限模型
// ========================================

// SaltstackClusterPermission SaltStack集群权限
type SaltstackClusterPermission struct {
	ID               uint            `json:"id" gorm:"primaryKey"`
	UserID           uint            `json:"user_id" gorm:"not null;index"`
	MasterID         string          `json:"master_id" gorm:"not null;index;size:100"` // Salt Master标识
	MasterAddress    string          `json:"master_address" gorm:"size:255"`           // Salt Master地址
	Verbs            StringArray     `json:"verbs" gorm:"type:json;not null"`          // 允许的操作 [view, execute, manage, admin]
	AllMinions       bool            `json:"all_minions" gorm:"default:false"`         // 是否有所有Minion权限
	MinionGroups     StringArray     `json:"minion_groups" gorm:"type:json"`           // 允许访问的Minion分组
	MinionPatterns   StringArray     `json:"minion_patterns" gorm:"type:json"`         // 允许访问的Minion模式(支持通配符)
	AllowedFunctions StringArray     `json:"allowed_functions" gorm:"type:json"`       // 允许执行的Salt函数
	DeniedFunctions  StringArray     `json:"denied_functions" gorm:"type:json"`        // 禁止执行的Salt函数
	AllowDangerous   bool            `json:"allow_dangerous" gorm:"default:false"`     // 是否允许执行危险命令
	MaxConcurrent    int             `json:"max_concurrent" gorm:"default:10"`         // 最大并发执行数
	RateLimit        int             `json:"rate_limit" gorm:"default:60"`             // 每分钟最大请求数
	Constraints      SaltConstraints `json:"constraints" gorm:"type:json"`             // 其他约束
	GrantedBy        uint            `json:"granted_by" gorm:"not null"`               // 授权人ID
	GrantReason      string          `json:"grant_reason" gorm:"size:500"`             // 授权原因
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"`                     // 过期时间
	IsActive         bool            `json:"is_active" gorm:"default:true"`            // 是否有效
	RevokedAt        *time.Time      `json:"revoked_at,omitempty"`                     // 撤销时间
	RevokedBy        *uint           `json:"revoked_by,omitempty"`                     // 撤销人ID
	RevokeReason     string          `json:"revoke_reason" gorm:"size:500"`            // 撤销原因
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `json:"-" gorm:"index"`

	// 关联关系
	User    User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Grantor User `json:"grantor,omitempty" gorm:"foreignKey:GrantedBy"`
}

// SaltConstraints SaltStack约束配置
type SaltConstraints struct {
	AllowedGrains   map[string][]string `json:"allowed_grains,omitempty"`   // 允许的Grains条件
	RequiredPillar  []string            `json:"required_pillar,omitempty"`  // 必需的Pillar数据
	AllowedModules  []string            `json:"allowed_modules,omitempty"`  // 允许使用的模块
	TimeRestriction *TimeRestriction    `json:"time_restriction,omitempty"` // 时间限制
	IPWhitelist     []string            `json:"ip_whitelist,omitempty"`     // IP白名单
}

// TimeRestriction 时间限制
type TimeRestriction struct {
	AllowedDays      []string `json:"allowed_days,omitempty"`       // 允许的星期几 ["monday", "tuesday"...]
	AllowedStartTime string   `json:"allowed_start_time,omitempty"` // 允许开始时间 "09:00"
	AllowedEndTime   string   `json:"allowed_end_time,omitempty"`   // 允许结束时间 "18:00"
	Timezone         string   `json:"timezone,omitempty"`           // 时区
}

// Scan 实现 database/sql.Scanner 接口
func (sc *SaltConstraints) Scan(value interface{}) error {
	if value == nil {
		*sc = SaltConstraints{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan SaltConstraints: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, sc)
}

// Value 实现 database/sql/driver.Valuer 接口
func (sc SaltConstraints) Value() (driver.Value, error) {
	return json.Marshal(sc)
}

// TableName 指定表名
func (SaltstackClusterPermission) TableName() string {
	return "saltstack_cluster_permissions"
}

// HasVerb 检查是否有指定权限
func (p *SaltstackClusterPermission) HasVerb(verb ClusterPermissionVerb) bool {
	if p.Verbs == nil {
		return false
	}
	for _, v := range p.Verbs {
		if ClusterPermissionVerb(v) == VerbAdmin || ClusterPermissionVerb(v) == verb {
			return true
		}
	}
	return false
}

// HasMinionAccess 检查是否有Minion访问权限
func (p *SaltstackClusterPermission) HasMinionAccess(minionID string) bool {
	if p.AllMinions {
		return true
	}
	// 检查精确匹配
	if p.MinionGroups != nil {
		for _, group := range p.MinionGroups {
			if group == minionID {
				return true
			}
		}
	}
	// 检查模式匹配
	if p.MinionPatterns != nil {
		for _, pattern := range p.MinionPatterns {
			if matchPattern(pattern, minionID) {
				return true
			}
		}
	}
	return false
}

// IsFunctionAllowed 检查Salt函数是否允许执行
func (p *SaltstackClusterPermission) IsFunctionAllowed(function string) bool {
	// 首先检查黑名单
	if p.DeniedFunctions != nil {
		for _, denied := range p.DeniedFunctions {
			if matchPattern(denied, function) {
				return false
			}
		}
	}
	// 如果没有白名单限制，则允许
	if p.AllowedFunctions == nil || len(p.AllowedFunctions) == 0 {
		return true
	}
	// 检查白名单
	for _, allowed := range p.AllowedFunctions {
		if matchPattern(allowed, function) {
			return true
		}
	}
	return false
}

// IsExpired 检查权限是否过期
func (p *SaltstackClusterPermission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsValid 检查权限是否有效
func (p *SaltstackClusterPermission) IsValid() bool {
	return p.IsActive && !p.IsExpired() && p.RevokedAt == nil
}

// ========================================
// 权限变更日志
// ========================================

// ClusterPermissionLog 集群权限变更日志
type ClusterPermissionLog struct {
	ID             uint                  `json:"id" gorm:"primaryKey"`
	PermissionType ClusterPermissionType `json:"permission_type" gorm:"not null;size:50;index"`
	PermissionID   uint                  `json:"permission_id" gorm:"not null;index"`
	UserID         uint                  `json:"user_id" gorm:"not null;index"`     // 被授权用户ID
	OperatorID     uint                  `json:"operator_id" gorm:"not null;index"` // 操作人ID
	Action         string                `json:"action" gorm:"not null;size:50"`    // grant, revoke, modify, expire
	OldValue       string                `json:"old_value" gorm:"type:text"`        // 变更前的值(JSON)
	NewValue       string                `json:"new_value" gorm:"type:text"`        // 变更后的值(JSON)
	Reason         string                `json:"reason" gorm:"size:500"`            // 操作原因
	IPAddress      string                `json:"ip_address" gorm:"size:45"`         // 操作人IP
	UserAgent      string                `json:"user_agent" gorm:"size:500"`        // 浏览器UA
	CreatedAt      time.Time             `json:"created_at"`

	// 关联关系
	User     User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Operator User `json:"operator,omitempty" gorm:"foreignKey:OperatorID"`
}

// TableName 指定表名
func (ClusterPermissionLog) TableName() string {
	return "cluster_permission_logs"
}

// ========================================
// 请求和响应结构
// ========================================

// GrantSlurmPermissionInput 授予SLURM权限请求
type GrantSlurmPermissionInput struct {
	UserID        uint     `json:"user_id" binding:"required"`
	ClusterID     uint     `json:"cluster_id" binding:"required"`
	Verbs         []string `json:"verbs" binding:"required,min=1"`
	AllPartitions bool     `json:"all_partitions"`
	Partitions    []string `json:"partitions"`
	MaxJobs       int      `json:"max_jobs"`
	MaxCPUs       int      `json:"max_cpus"`
	MaxGPUs       int      `json:"max_gpus"`
	MaxMemoryGB   int      `json:"max_memory_gb"`
	MaxWalltime   string   `json:"max_walltime"`
	Priority      int      `json:"priority"`
	QOS           string   `json:"qos"`
	Account       string   `json:"account"`
	ValidDays     int      `json:"valid_days"` // 有效天数(0=永久)
	Reason        string   `json:"reason" binding:"required,min=5,max=500"`
}

// GrantSaltstackPermissionInput 授予SaltStack权限请求
type GrantSaltstackPermissionInput struct {
	UserID           uint     `json:"user_id" binding:"required"`
	MasterID         string   `json:"master_id" binding:"required"`
	MasterAddress    string   `json:"master_address"`
	Verbs            []string `json:"verbs" binding:"required,min=1"`
	AllMinions       bool     `json:"all_minions"`
	MinionGroups     []string `json:"minion_groups"`
	MinionPatterns   []string `json:"minion_patterns"`
	AllowedFunctions []string `json:"allowed_functions"`
	DeniedFunctions  []string `json:"denied_functions"`
	AllowDangerous   bool     `json:"allow_dangerous"`
	MaxConcurrent    int      `json:"max_concurrent"`
	RateLimit        int      `json:"rate_limit"`
	ValidDays        int      `json:"valid_days"`
	Reason           string   `json:"reason" binding:"required,min=5,max=500"`
}

// UpdateSlurmPermissionInput 更新SLURM权限请求
type UpdateSlurmPermissionInput struct {
	Verbs         []string `json:"verbs"`
	AllPartitions *bool    `json:"all_partitions"`
	Partitions    []string `json:"partitions"`
	MaxJobs       *int     `json:"max_jobs"`
	MaxCPUs       *int     `json:"max_cpus"`
	MaxGPUs       *int     `json:"max_gpus"`
	MaxMemoryGB   *int     `json:"max_memory_gb"`
	MaxWalltime   *string  `json:"max_walltime"`
	Priority      *int     `json:"priority"`
	QOS           *string  `json:"qos"`
	Account       *string  `json:"account"`
	ExpiresAt     *string  `json:"expires_at"` // ISO8601格式
	Reason        string   `json:"reason" binding:"required,min=5,max=500"`
}

// UpdateSaltstackPermissionInput 更新SaltStack权限请求
type UpdateSaltstackPermissionInput struct {
	Verbs            []string `json:"verbs"`
	AllMinions       *bool    `json:"all_minions"`
	MinionGroups     []string `json:"minion_groups"`
	MinionPatterns   []string `json:"minion_patterns"`
	AllowedFunctions []string `json:"allowed_functions"`
	DeniedFunctions  []string `json:"denied_functions"`
	AllowDangerous   *bool    `json:"allow_dangerous"`
	MaxConcurrent    *int     `json:"max_concurrent"`
	RateLimit        *int     `json:"rate_limit"`
	ExpiresAt        *string  `json:"expires_at"`
	Reason           string   `json:"reason" binding:"required,min=5,max=500"`
}

// RevokeClusterPermissionInput 撤销权限请求
type RevokeClusterPermissionInput struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

// ClusterPermissionQuery 权限查询参数
type ClusterPermissionQuery struct {
	UserID         uint   `form:"user_id"`
	ClusterID      uint   `form:"cluster_id"`
	MasterID       string `form:"master_id"`
	PermissionType string `form:"permission_type"`
	IsActive       *bool  `form:"is_active"`
	IncludeExpired bool   `form:"include_expired"`
	Page           int    `form:"page" binding:"omitempty,min=1"`
	PageSize       int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// SlurmPermissionListResponse SLURM权限列表响应
type SlurmPermissionListResponse struct {
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
	Items    []SlurmClusterPermission `json:"items"`
}

// SaltstackPermissionListResponse SaltStack权限列表响应
type SaltstackPermissionListResponse struct {
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
	Items    []SaltstackClusterPermission `json:"items"`
}

// UserClusterPermissions 用户集群权限汇总
type UserClusterPermissions struct {
	UserID    uint                         `json:"user_id"`
	Username  string                       `json:"username"`
	Slurm     []SlurmClusterPermission     `json:"slurm"`
	Saltstack []SaltstackClusterPermission `json:"saltstack"`
}

// ClusterAccessInfo 集群访问信息
type ClusterAccessInfo struct {
	ClusterID      uint            `json:"cluster_id"`
	ClusterName    string          `json:"cluster_name"`
	ClusterType    string          `json:"cluster_type"`
	HasAccess      bool            `json:"has_access"`
	Verbs          []string        `json:"verbs"`
	Partitions     []string        `json:"partitions,omitempty"`    // SLURM特有
	MinionGroups   []string        `json:"minion_groups,omitempty"` // SaltStack特有
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty"`
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	MaxJobs     int    `json:"max_jobs,omitempty"`
	MaxCPUs     int    `json:"max_cpus,omitempty"`
	MaxGPUs     int    `json:"max_gpus,omitempty"`
	MaxMemoryGB int    `json:"max_memory_gb,omitempty"`
	MaxWalltime string `json:"max_walltime,omitempty"`
}

// ========================================
// 辅助函数
// ========================================

// matchPattern 简单的通配符匹配
func matchPattern(pattern, str string) bool {
	if pattern == "*" || pattern == str {
		return true
	}
	// 支持前缀匹配 (如 "cmd.*")
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}
	// 支持后缀匹配 (如 "*.run")
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
	}
	return false
}

// VerifyPermission 验证用户是否有集群权限
type VerifyPermissionResult struct {
	Allowed        bool            `json:"allowed"`
	Reason         string          `json:"reason"`
	MatchedVerbs   []string        `json:"matched_verbs,omitempty"`
	MissingVerbs   []string        `json:"missing_verbs,omitempty"`
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty"`
}
