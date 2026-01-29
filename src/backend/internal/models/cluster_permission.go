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
	PermissionTypeSlurmCluster      ClusterPermissionType = "slurm_cluster"
	PermissionTypeSlurmPartition    ClusterPermissionType = "slurm_partition"
	PermissionTypeSaltstackCluster  ClusterPermissionType = "saltstack_cluster"
	PermissionTypeSaltstackMinion   ClusterPermissionType = "saltstack_minion"
	PermissionTypeKubernetesCluster ClusterPermissionType = "kubernetes_cluster"
	PermissionTypeKubernetesNS      ClusterPermissionType = "kubernetes_namespace"
	PermissionTypeAnsibleProject    ClusterPermissionType = "ansible_project"
	PermissionTypeAnsibleInventory  ClusterPermissionType = "ansible_inventory"
	PermissionTypeJupyterHub        ClusterPermissionType = "jupyterhub"
	PermissionTypeGitea             ClusterPermissionType = "gitea"
	PermissionTypeMonitoring        ClusterPermissionType = "monitoring"
	PermissionTypeObjectStorage     ClusterPermissionType = "object_storage"
	PermissionTypeArgoCD            ClusterPermissionType = "argocd"
)

// ClusterPermissionVerb 集群权限动作
type ClusterPermissionVerb string

const (
	VerbView    ClusterPermissionVerb = "view"     // 查看
	VerbSubmit  ClusterPermissionVerb = "submit"   // 提交任务
	VerbManage  ClusterPermissionVerb = "manage"   // 管理（启动/停止/配置）
	VerbAdmin   ClusterPermissionVerb = "admin"    // 管理员（包含所有权限）
	VerbExecute ClusterPermissionVerb = "execute"  // 执行命令（SaltStack/Ansible）
	VerbMonitor ClusterPermissionVerb = "monitor"  // 监控
	VerbScale   ClusterPermissionVerb = "scale"    // 扩缩容
	VerbConnect ClusterPermissionVerb = "connect"  // 连接/SSH
	VerbDeploy  ClusterPermissionVerb = "deploy"   // 部署（K8s/ArgoCD）
	VerbDelete  ClusterPermissionVerb = "delete"   // 删除资源
	VerbExec    ClusterPermissionVerb = "exec"     // 执行Pod命令（K8s）
	VerbLogs    ClusterPermissionVerb = "logs"     // 查看日志
	VerbPort    ClusterPermissionVerb = "port"     // 端口转发（K8s）
	VerbCreate  ClusterPermissionVerb = "create"   // 创建资源
	VerbEdit    ClusterPermissionVerb = "edit"     // 编辑资源
	VerbRun     ClusterPermissionVerb = "run"      // 运行（Ansible Playbook）
	VerbSync    ClusterPermissionVerb = "sync"     // 同步（ArgoCD）
	VerbUpload  ClusterPermissionVerb = "upload"   // 上传文件
	VerbDownld  ClusterPermissionVerb = "download" // 下载文件
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
// Kubernetes 集群权限模型
// ========================================

// KubernetesClusterPermission Kubernetes集群权限
type KubernetesClusterPermission struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"user_id" gorm:"not null;index"`
	ClusterID        uint           `json:"cluster_id" gorm:"not null;index"`
	Verbs            StringArray    `json:"verbs" gorm:"type:json;not null"`         // 允许的操作 [view, deploy, delete, exec, logs, port, scale, admin]
	AllNamespaces    bool           `json:"all_namespaces" gorm:"default:false"`     // 是否有所有命名空间权限
	Namespaces       StringArray    `json:"namespaces" gorm:"type:json"`             // 允许访问的命名空间列表
	AllResources     bool           `json:"all_resources" gorm:"default:false"`      // 是否有所有资源类型权限
	ResourceTypes    StringArray    `json:"resource_types" gorm:"type:json"`         // 允许的资源类型 [pods, deployments, services, configmaps, secrets, etc.]
	AllowExec        bool           `json:"allow_exec" gorm:"default:false"`         // 是否允许exec进入Pod
	AllowPortForward bool           `json:"allow_port_forward" gorm:"default:false"` // 是否允许端口转发
	AllowLogs        bool           `json:"allow_logs" gorm:"default:true"`          // 是否允许查看日志
	MaxPods          int            `json:"max_pods" gorm:"default:0"`               // 最大Pod数(0=无限制)
	MaxCPU           string         `json:"max_cpu" gorm:"size:50"`                  // 最大CPU限制(如 "4000m")
	MaxMemory        string         `json:"max_memory" gorm:"size:50"`               // 最大内存限制(如 "8Gi")
	MaxStorage       string         `json:"max_storage" gorm:"size:50"`              // 最大存储限制(如 "100Gi")
	Constraints      K8sConstraints `json:"constraints" gorm:"type:json"`            // 其他约束
	GrantedBy        uint           `json:"granted_by" gorm:"not null"`              // 授权人ID
	GrantReason      string         `json:"grant_reason" gorm:"size:500"`            // 授权原因
	ExpiresAt        *time.Time     `json:"expires_at,omitempty"`                    // 过期时间
	IsActive         bool           `json:"is_active" gorm:"default:true"`           // 是否有效
	RevokedAt        *time.Time     `json:"revoked_at,omitempty"`                    // 撤销时间
	RevokedBy        *uint          `json:"revoked_by,omitempty"`                    // 撤销人ID
	RevokeReason     string         `json:"revoke_reason" gorm:"size:500"`           // 撤销原因
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User    User              `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Cluster KubernetesCluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	Grantor User              `json:"grantor,omitempty" gorm:"foreignKey:GrantedBy"`
}

// K8sConstraints Kubernetes约束配置
type K8sConstraints struct {
	AllowedLabels      map[string]string `json:"allowed_labels,omitempty"`       // 允许的标签选择器
	RequiredLabels     map[string]string `json:"required_labels,omitempty"`      // 必需的标签
	AllowedAnnotations map[string]string `json:"allowed_annotations,omitempty"`  // 允许的注解
	AllowPrivileged    bool              `json:"allow_privileged"`               // 是否允许特权容器
	AllowHostNetwork   bool              `json:"allow_host_network"`             // 是否允许主机网络
	AllowHostPID       bool              `json:"allow_host_pid"`                 // 是否允许主机PID
	AllowedImages      []string          `json:"allowed_images,omitempty"`       // 允许的镜像列表/模式
	DeniedImages       []string          `json:"denied_images,omitempty"`        // 禁止的镜像列表/模式
	ServiceAccountName string            `json:"service_account_name,omitempty"` // 指定的服务账户
	NodeSelector       map[string]string `json:"node_selector,omitempty"`        // 节点选择器限制
}

// Scan 实现 database/sql.Scanner 接口
func (kc *K8sConstraints) Scan(value interface{}) error {
	if value == nil {
		*kc = K8sConstraints{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan K8sConstraints: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, kc)
}

// Value 实现 database/sql/driver.Valuer 接口
func (kc K8sConstraints) Value() (driver.Value, error) {
	return json.Marshal(kc)
}

// TableName 指定表名
func (KubernetesClusterPermission) TableName() string {
	return "kubernetes_cluster_permissions"
}

// HasVerb 检查是否有指定权限
func (p *KubernetesClusterPermission) HasVerb(verb ClusterPermissionVerb) bool {
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

// HasNamespaceAccess 检查是否有命名空间访问权限
func (p *KubernetesClusterPermission) HasNamespaceAccess(namespace string) bool {
	if p.AllNamespaces {
		return true
	}
	if p.Namespaces == nil {
		return false
	}
	for _, ns := range p.Namespaces {
		if ns == namespace || ns == "*" {
			return true
		}
	}
	return false
}

// HasResourceAccess 检查是否有资源类型访问权限
func (p *KubernetesClusterPermission) HasResourceAccess(resourceType string) bool {
	if p.AllResources {
		return true
	}
	if p.ResourceTypes == nil {
		return false
	}
	for _, rt := range p.ResourceTypes {
		if rt == resourceType || rt == "*" {
			return true
		}
	}
	return false
}

// IsExpired 检查权限是否过期
func (p *KubernetesClusterPermission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsValid 检查权限是否有效
func (p *KubernetesClusterPermission) IsValid() bool {
	return p.IsActive && !p.IsExpired() && p.RevokedAt == nil
}

// ========================================
// Ansible 权限模型
// ========================================

// AnsiblePermission Ansible执行权限
type AnsiblePermission struct {
	ID             uint               `json:"id" gorm:"primaryKey"`
	UserID         uint               `json:"user_id" gorm:"not null;index"`
	ProjectID      uint               `json:"project_id" gorm:"index"`              // 项目ID(0=所有项目)
	Verbs          StringArray        `json:"verbs" gorm:"type:json;not null"`      // 允许的操作 [view, run, edit, delete, admin]
	AllProjects    bool               `json:"all_projects" gorm:"default:false"`    // 是否有所有项目权限
	AllInventories bool               `json:"all_inventories" gorm:"default:false"` // 是否有所有清单权限
	Inventories    StringArray        `json:"inventories" gorm:"type:json"`         // 允许访问的清单ID列表
	AllPlaybooks   bool               `json:"all_playbooks" gorm:"default:false"`   // 是否有所有Playbook权限
	Playbooks      StringArray        `json:"playbooks" gorm:"type:json"`           // 允许执行的Playbook路径列表
	AllowedHosts   StringArray        `json:"allowed_hosts" gorm:"type:json"`       // 允许操作的主机/组
	DeniedHosts    StringArray        `json:"denied_hosts" gorm:"type:json"`        // 禁止操作的主机/组
	AllowDryRun    bool               `json:"allow_dry_run" gorm:"default:true"`    // 是否允许dry-run模式
	AllowProdExec  bool               `json:"allow_prod_exec" gorm:"default:false"` // 是否允许生产环境执行
	MaxConcurrent  int                `json:"max_concurrent" gorm:"default:5"`      // 最大并发任务数
	Environments   StringArray        `json:"environments" gorm:"type:json"`        // 允许操作的环境 [dev, test, staging, prod]
	Constraints    AnsibleConstraints `json:"constraints" gorm:"type:json"`         // 其他约束
	GrantedBy      uint               `json:"granted_by" gorm:"not null"`           // 授权人ID
	GrantReason    string             `json:"grant_reason" gorm:"size:500"`         // 授权原因
	ExpiresAt      *time.Time         `json:"expires_at,omitempty"`                 // 过期时间
	IsActive       bool               `json:"is_active" gorm:"default:true"`        // 是否有效
	RevokedAt      *time.Time         `json:"revoked_at,omitempty"`                 // 撤销时间
	RevokedBy      *uint              `json:"revoked_by,omitempty"`                 // 撤销人ID
	RevokeReason   string             `json:"revoke_reason" gorm:"size:500"`        // 撤销原因
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	DeletedAt      gorm.DeletedAt     `json:"-" gorm:"index"`

	// 关联关系
	User    User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Grantor User `json:"grantor,omitempty" gorm:"foreignKey:GrantedBy"`
}

// AnsibleConstraints Ansible约束配置
type AnsibleConstraints struct {
	AllowedModules   []string            `json:"allowed_modules,omitempty"`    // 允许使用的模块
	DeniedModules    []string            `json:"denied_modules,omitempty"`     // 禁止使用的模块
	AllowedRoles     []string            `json:"allowed_roles,omitempty"`      // 允许使用的角色
	RequireApproval  bool                `json:"require_approval"`             // 是否需要审批
	ApproverRoles    []string            `json:"approver_roles,omitempty"`     // 审批人角色
	TimeRestriction  *TimeRestriction    `json:"time_restriction,omitempty"`   // 时间限制
	MaxExecTime      int                 `json:"max_exec_time,omitempty"`      // 最大执行时间(秒)
	ExtraVarsAllowed map[string][]string `json:"extra_vars_allowed,omitempty"` // 允许的额外变量
	ExtraVarsDenied  []string            `json:"extra_vars_denied,omitempty"`  // 禁止的额外变量
}

// Scan 实现 database/sql.Scanner 接口
func (ac *AnsibleConstraints) Scan(value interface{}) error {
	if value == nil {
		*ac = AnsibleConstraints{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan AnsibleConstraints: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, ac)
}

// Value 实现 database/sql/driver.Valuer 接口
func (ac AnsibleConstraints) Value() (driver.Value, error) {
	return json.Marshal(ac)
}

// TableName 指定表名
func (AnsiblePermission) TableName() string {
	return "ansible_permissions"
}

// HasVerb 检查是否有指定权限
func (p *AnsiblePermission) HasVerb(verb ClusterPermissionVerb) bool {
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

// HasHostAccess 检查是否有主机访问权限
func (p *AnsiblePermission) HasHostAccess(host string) bool {
	// 检查黑名单
	if p.DeniedHosts != nil {
		for _, denied := range p.DeniedHosts {
			if matchPattern(denied, host) {
				return false
			}
		}
	}
	// 如果没有白名单限制，则允许
	if p.AllowedHosts == nil || len(p.AllowedHosts) == 0 {
		return true
	}
	// 检查白名单
	for _, allowed := range p.AllowedHosts {
		if matchPattern(allowed, host) {
			return true
		}
	}
	return false
}

// HasPlaybookAccess 检查是否有Playbook执行权限
func (p *AnsiblePermission) HasPlaybookAccess(playbookPath string) bool {
	if p.AllPlaybooks {
		return true
	}
	if p.Playbooks == nil || len(p.Playbooks) == 0 {
		return true // 默认允许
	}
	for _, pb := range p.Playbooks {
		if matchPattern(pb, playbookPath) {
			return true
		}
	}
	return false
}

// HasEnvironmentAccess 检查是否有环境访问权限
func (p *AnsiblePermission) HasEnvironmentAccess(env string) bool {
	if p.Environments == nil || len(p.Environments) == 0 {
		return true // 默认允许所有环境
	}
	for _, e := range p.Environments {
		if e == env || e == "*" {
			return true
		}
	}
	return false
}

// IsExpired 检查权限是否过期
func (p *AnsiblePermission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsValid 检查权限是否有效
func (p *AnsiblePermission) IsValid() bool {
	return p.IsActive && !p.IsExpired() && p.RevokedAt == nil
}

// ========================================
// 组件通用权限模型
// ========================================

// ComponentPermission 组件通用权限（JupyterHub、Gitea、监控、对象存储、ArgoCD等）
type ComponentPermission struct {
	ID            uint                 `json:"id" gorm:"primaryKey"`
	UserID        uint                 `json:"user_id" gorm:"not null;index"`
	ComponentType string               `json:"component_type" gorm:"not null;size:50;index"` // jupyterhub, gitea, monitoring, object_storage, argocd
	ComponentID   string               `json:"component_id" gorm:"size:100;index"`           // 组件实例ID（可选）
	Verbs         StringArray          `json:"verbs" gorm:"type:json;not null"`              // 允许的操作
	Scope         StringArray          `json:"scope" gorm:"type:json"`                       // 权限范围（如特定仓库、项目等）
	Constraints   ComponentConstraints `json:"constraints" gorm:"type:json"`                 // 组件特定约束
	GrantedBy     uint                 `json:"granted_by" gorm:"not null"`                   // 授权人ID
	GrantReason   string               `json:"grant_reason" gorm:"size:500"`                 // 授权原因
	ExpiresAt     *time.Time           `json:"expires_at,omitempty"`                         // 过期时间
	IsActive      bool                 `json:"is_active" gorm:"default:true"`                // 是否有效
	RevokedAt     *time.Time           `json:"revoked_at,omitempty"`                         // 撤销时间
	RevokedBy     *uint                `json:"revoked_by,omitempty"`                         // 撤销人ID
	RevokeReason  string               `json:"revoke_reason" gorm:"size:500"`                // 撤销原因
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	DeletedAt     gorm.DeletedAt       `json:"-" gorm:"index"`

	// 关联关系
	User    User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Grantor User `json:"grantor,omitempty" gorm:"foreignKey:GrantedBy"`
}

// ComponentConstraints 组件约束配置
type ComponentConstraints struct {
	// JupyterHub特定
	MaxServers      int      `json:"max_servers,omitempty"`      // 最大服务器数
	AllowedImages   []string `json:"allowed_images,omitempty"`   // 允许的镜像
	MaxCPU          string   `json:"max_cpu,omitempty"`          // 最大CPU
	MaxMemory       string   `json:"max_memory,omitempty"`       // 最大内存
	MaxGPU          int      `json:"max_gpu,omitempty"`          // 最大GPU
	AllowedProfiles []string `json:"allowed_profiles,omitempty"` // 允许的配置文件

	// Gitea特定
	MaxRepos         int      `json:"max_repos,omitempty"`    // 最大仓库数
	AllowedOrgs      []string `json:"allowed_orgs,omitempty"` // 允许访问的组织
	AllowPrivateRepo bool     `json:"allow_private_repo"`     // 是否允许私有仓库
	AllowFork        bool     `json:"allow_fork"`             // 是否允许Fork

	// 监控特定
	AllowedDashboards []string `json:"allowed_dashboards,omitempty"` // 允许访问的仪表板
	AllowAlertConfig  bool     `json:"allow_alert_config"`           // 是否允许配置告警
	AllowSilence      bool     `json:"allow_silence"`                // 是否允许静默告警

	// 对象存储特定
	MaxBuckets       int      `json:"max_buckets,omitempty"`     // 最大桶数
	MaxStorageGB     int      `json:"max_storage_gb,omitempty"`  // 最大存储GB
	AllowedBuckets   []string `json:"allowed_buckets,omitempty"` // 允许访问的桶
	AllowPublicRead  bool     `json:"allow_public_read"`         // 是否允许公开读
	AllowPublicWrite bool     `json:"allow_public_write"`        // 是否允许公开写

	// ArgoCD特定
	AllowedApps      []string `json:"allowed_apps,omitempty"`     // 允许访问的应用
	AllowedProjects  []string `json:"allowed_projects,omitempty"` // 允许访问的项目
	AllowSync        bool     `json:"allow_sync"`                 // 是否允许同步
	AllowRollback    bool     `json:"allow_rollback"`             // 是否允许回滚
	AllowHardRefresh bool     `json:"allow_hard_refresh"`         // 是否允许强制刷新

	// 通用
	TimeRestriction *TimeRestriction `json:"time_restriction,omitempty"` // 时间限制
	IPWhitelist     []string         `json:"ip_whitelist,omitempty"`     // IP白名单
	RateLimit       int              `json:"rate_limit,omitempty"`       // 速率限制
}

// Scan 实现 database/sql.Scanner 接口
func (cc *ComponentConstraints) Scan(value interface{}) error {
	if value == nil {
		*cc = ComponentConstraints{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ComponentConstraints: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, cc)
}

// Value 实现 database/sql/driver.Valuer 接口
func (cc ComponentConstraints) Value() (driver.Value, error) {
	return json.Marshal(cc)
}

// TableName 指定表名
func (ComponentPermission) TableName() string {
	return "component_permissions"
}

// HasVerb 检查是否有指定权限
func (p *ComponentPermission) HasVerb(verb ClusterPermissionVerb) bool {
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

// HasScopeAccess 检查是否有范围访问权限
func (p *ComponentPermission) HasScopeAccess(scope string) bool {
	if p.Scope == nil || len(p.Scope) == 0 {
		return true // 无限制
	}
	for _, s := range p.Scope {
		if s == scope || s == "*" || matchPattern(s, scope) {
			return true
		}
	}
	return false
}

// IsExpired 检查权限是否过期
func (p *ComponentPermission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsValid 检查权限是否有效
func (p *ComponentPermission) IsValid() bool {
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
	UserID     uint                          `json:"user_id"`
	Username   string                        `json:"username"`
	Slurm      []SlurmClusterPermission      `json:"slurm"`
	Saltstack  []SaltstackClusterPermission  `json:"saltstack"`
	Kubernetes []KubernetesClusterPermission `json:"kubernetes"`
	Ansible    []AnsiblePermission           `json:"ansible"`
	Components []ComponentPermission         `json:"components"`
}

// GrantKubernetesPermissionInput 授予Kubernetes权限请求
type GrantKubernetesPermissionInput struct {
	UserID           uint     `json:"user_id" binding:"required"`
	ClusterID        uint     `json:"cluster_id" binding:"required"`
	Verbs            []string `json:"verbs" binding:"required,min=1"`
	AllNamespaces    bool     `json:"all_namespaces"`
	Namespaces       []string `json:"namespaces"`
	AllResources     bool     `json:"all_resources"`
	ResourceTypes    []string `json:"resource_types"`
	AllowExec        bool     `json:"allow_exec"`
	AllowPortForward bool     `json:"allow_port_forward"`
	AllowLogs        bool     `json:"allow_logs"`
	MaxPods          int      `json:"max_pods"`
	MaxCPU           string   `json:"max_cpu"`
	MaxMemory        string   `json:"max_memory"`
	MaxStorage       string   `json:"max_storage"`
	ValidDays        int      `json:"valid_days"`
	Reason           string   `json:"reason" binding:"required,min=5,max=500"`
}

// UpdateKubernetesPermissionInput 更新Kubernetes权限请求
type UpdateKubernetesPermissionInput struct {
	Verbs            []string `json:"verbs"`
	AllNamespaces    *bool    `json:"all_namespaces"`
	Namespaces       []string `json:"namespaces"`
	AllResources     *bool    `json:"all_resources"`
	ResourceTypes    []string `json:"resource_types"`
	AllowExec        *bool    `json:"allow_exec"`
	AllowPortForward *bool    `json:"allow_port_forward"`
	AllowLogs        *bool    `json:"allow_logs"`
	MaxPods          *int     `json:"max_pods"`
	MaxCPU           *string  `json:"max_cpu"`
	MaxMemory        *string  `json:"max_memory"`
	MaxStorage       *string  `json:"max_storage"`
	ExpiresAt        *string  `json:"expires_at"`
	Reason           string   `json:"reason" binding:"required,min=5,max=500"`
}

// GrantAnsiblePermissionInput 授予Ansible权限请求
type GrantAnsiblePermissionInput struct {
	UserID         uint     `json:"user_id" binding:"required"`
	ProjectID      uint     `json:"project_id"`
	Verbs          []string `json:"verbs" binding:"required,min=1"`
	AllProjects    bool     `json:"all_projects"`
	AllInventories bool     `json:"all_inventories"`
	Inventories    []string `json:"inventories"`
	AllPlaybooks   bool     `json:"all_playbooks"`
	Playbooks      []string `json:"playbooks"`
	AllowedHosts   []string `json:"allowed_hosts"`
	DeniedHosts    []string `json:"denied_hosts"`
	AllowDryRun    bool     `json:"allow_dry_run"`
	AllowProdExec  bool     `json:"allow_prod_exec"`
	MaxConcurrent  int      `json:"max_concurrent"`
	Environments   []string `json:"environments"`
	ValidDays      int      `json:"valid_days"`
	Reason         string   `json:"reason" binding:"required,min=5,max=500"`
}

// UpdateAnsiblePermissionInput 更新Ansible权限请求
type UpdateAnsiblePermissionInput struct {
	Verbs          []string `json:"verbs"`
	AllProjects    *bool    `json:"all_projects"`
	AllInventories *bool    `json:"all_inventories"`
	Inventories    []string `json:"inventories"`
	AllPlaybooks   *bool    `json:"all_playbooks"`
	Playbooks      []string `json:"playbooks"`
	AllowedHosts   []string `json:"allowed_hosts"`
	DeniedHosts    []string `json:"denied_hosts"`
	AllowDryRun    *bool    `json:"allow_dry_run"`
	AllowProdExec  *bool    `json:"allow_prod_exec"`
	MaxConcurrent  *int     `json:"max_concurrent"`
	Environments   []string `json:"environments"`
	ExpiresAt      *string  `json:"expires_at"`
	Reason         string   `json:"reason" binding:"required,min=5,max=500"`
}

// GrantComponentPermissionInput 授予组件权限请求
type GrantComponentPermissionInput struct {
	UserID        uint                 `json:"user_id" binding:"required"`
	ComponentType string               `json:"component_type" binding:"required"` // jupyterhub, gitea, monitoring, object_storage, argocd
	ComponentID   string               `json:"component_id"`
	Verbs         []string             `json:"verbs" binding:"required,min=1"`
	Scope         []string             `json:"scope"`
	Constraints   ComponentConstraints `json:"constraints"`
	ValidDays     int                  `json:"valid_days"`
	Reason        string               `json:"reason" binding:"required,min=5,max=500"`
}

// UpdateComponentPermissionInput 更新组件权限请求
type UpdateComponentPermissionInput struct {
	Verbs       []string              `json:"verbs"`
	Scope       []string              `json:"scope"`
	Constraints *ComponentConstraints `json:"constraints"`
	ExpiresAt   *string               `json:"expires_at"`
	Reason      string                `json:"reason" binding:"required,min=5,max=500"`
}

// KubernetesPermissionListResponse Kubernetes权限列表响应
type KubernetesPermissionListResponse struct {
	Total    int64                         `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"page_size"`
	Items    []KubernetesClusterPermission `json:"items"`
}

// AnsiblePermissionListResponse Ansible权限列表响应
type AnsiblePermissionListResponse struct {
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Items    []AnsiblePermission `json:"items"`
}

// ComponentPermissionListResponse 组件权限列表响应
type ComponentPermissionListResponse struct {
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Items    []ComponentPermission `json:"items"`
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
