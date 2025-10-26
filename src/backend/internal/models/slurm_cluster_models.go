package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SlurmCluster SLURM集群模型
type SlurmCluster struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;size:100"`
	Description string         `json:"description" gorm:"size:500"`
	Status      string         `json:"status" gorm:"default:'pending';size:50"` // pending, deploying, running, scaling, failed, stopped
	MasterHost  string         `json:"master_host" gorm:"size:255"`
	MasterPort  int            `json:"master_port" gorm:"default:22"`
	SaltMaster  string         `json:"salt_master" gorm:"size:255"` // SaltStack Master地址
	Config      ClusterConfig  `json:"config" gorm:"type:json"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User        User                `json:"user,omitempty" gorm:"foreignKey:CreatedBy"`
	Nodes       []SlurmNode         `json:"nodes,omitempty" gorm:"foreignKey:ClusterID"`
	Deployments []ClusterDeployment `json:"deployments,omitempty" gorm:"foreignKey:ClusterID"`
}

// SlurmNode SLURM节点模型
type SlurmNode struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	ClusterID    uint           `json:"cluster_id" gorm:"not null"`
	NodeName     string         `json:"node_name" gorm:"not null;size:100"`
	NodeType     string         `json:"node_type" gorm:"not null;size:50"` // master, compute, login
	Host         string         `json:"host" gorm:"not null;size:255"`
	Port         int            `json:"port" gorm:"default:22"`
	Username     string         `json:"username" gorm:"not null;size:100"`
	AuthType     string         `json:"auth_type" gorm:"default:'password';size:20"` // password, key
	Password     string         `json:"password,omitempty" gorm:"size:255"`
	KeyPath      string         `json:"key_path,omitempty" gorm:"size:500"`
	Status       string         `json:"status" gorm:"default:'pending';size:50"` // pending, connecting, installing, configuring, active, failed, removing
	SaltMinionID string         `json:"salt_minion_id" gorm:"size:100"`
	CPUs         int            `json:"cpus" gorm:"default:1"`
	Memory       int            `json:"memory" gorm:"default:1024"` // MB
	Storage      int            `json:"storage" gorm:"default:10"`  // GB
	GPUs         int            `json:"gpus" gorm:"default:0"`
	NodeConfig   NodeConfig     `json:"node_config" gorm:"type:json"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Cluster      SlurmCluster      `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	InstallTasks []NodeInstallTask `json:"install_tasks,omitempty" gorm:"foreignKey:NodeID"`
	SSHLogs      []SSHExecutionLog `json:"ssh_logs,omitempty" gorm:"foreignKey:NodeID"`
}

// ClusterDeployment 集群部署记录
type ClusterDeployment struct {
	ID           uint             `json:"id" gorm:"primaryKey"`
	ClusterID    uint             `json:"cluster_id" gorm:"not null"`
	DeploymentID string           `json:"deployment_id" gorm:"not null;uniqueIndex;size:100"`
	Action       string           `json:"action" gorm:"not null;size:50"`          // deploy, scale-up, scale-down, update, destroy
	Status       string           `json:"status" gorm:"default:'pending';size:50"` // pending, running, completed, failed, cancelled
	Progress     int              `json:"progress" gorm:"default:0"`               // 0-100
	StartedAt    *time.Time       `json:"started_at,omitempty"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty" gorm:"type:text"`
	Config       DeploymentConfig `json:"config" gorm:"type:json"`
	Result       DeploymentResult `json:"result" gorm:"type:json"`
	CreatedBy    uint             `json:"created_by"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`

	// 关联关系
	Cluster      SlurmCluster      `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	User         User              `json:"user,omitempty" gorm:"foreignKey:CreatedBy"`
	Steps        []DeploymentStep  `json:"steps,omitempty" gorm:"foreignKey:DeploymentID"`
	InstallTasks []NodeInstallTask `json:"install_tasks,omitempty" gorm:"foreignKey:DeploymentID"`
}

// DeploymentStep 部署步骤记录
type DeploymentStep struct {
	ID           uint        `json:"id" gorm:"primaryKey"`
	DeploymentID uint        `json:"deployment_id" gorm:"not null"`
	StepName     string      `json:"step_name" gorm:"not null;size:100"`
	StepType     string      `json:"step_type" gorm:"not null;size:50"`       // ssh, salt, slurm, validation
	Status       string      `json:"status" gorm:"default:'pending';size:50"` // pending, running, completed, failed, skipped
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	Duration     int         `json:"duration" gorm:"default:0"` // 秒
	Command      string      `json:"command,omitempty" gorm:"type:text"`
	Output       string      `json:"output,omitempty" gorm:"type:text"`
	ErrorMessage string      `json:"error_message,omitempty" gorm:"type:text"`
	NodeTargets  StringArray `json:"node_targets" gorm:"type:json"` // 目标节点列表
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`

	// 关联关系 - 完全移除外键约束，只保留逻辑关系
	Deployment ClusterDeployment `json:"deployment,omitempty" gorm:"-"`
	SSHLogs    []SSHExecutionLog `json:"ssh_logs,omitempty" gorm:"-"`
}

// NodeInstallTask 节点安装任务记录
type NodeInstallTask struct {
	ID            uint              `json:"id" gorm:"primaryKey"`
	TaskID        string            `json:"task_id" gorm:"not null;uniqueIndex;size:100"`
	NodeID        uint              `json:"node_id" gorm:"not null"`
	DeploymentID  uint              `json:"deployment_id"`
	TaskType      string            `json:"task_type" gorm:"not null;size:50"`       // salt-minion, slurm-node, slurm-master, slurm-login
	Status        string            `json:"status" gorm:"default:'pending';size:50"` // pending, running, completed, failed, cancelled
	Progress      int               `json:"progress" gorm:"default:0"`               // 0-100
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty" gorm:"type:text"`
	InstallConfig InstallTaskConfig `json:"install_config" gorm:"type:json"`
	Result        InstallTaskResult `json:"result" gorm:"type:json"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`

	// 关联关系
	// 提示：为避免 AutoMigrate 在 PostgreSQL 下内联创建外键约束（导致创建顺序问题），
	// 仅在需要的方向上保留关系元数据，其余在 schema 层面禁用（gorm:"-")，
	// 外键约束由 database.addForeignKeys() 统一补齐。
	Node       SlurmNode         `json:"node,omitempty" gorm:"foreignKey:NodeID"`
	Deployment ClusterDeployment `json:"deployment,omitempty" gorm:"-"`
	Steps      []InstallStep     `json:"steps,omitempty" gorm:"foreignKey:TaskID"`
	SSHLogs    []SSHExecutionLog `json:"ssh_logs,omitempty" gorm:"foreignKey:TaskID"`
}

// InstallStep 安装步骤记录
type InstallStep struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	TaskID       uint       `json:"task_id" gorm:"not null"`
	StepName     string     `json:"step_name" gorm:"not null;size:100"`
	StepType     string     `json:"step_type" gorm:"not null;size:50"`       // ssh-connect, download, install, configure, start, validate
	Status       string     `json:"status" gorm:"default:'pending';size:50"` // pending, running, completed, failed, skipped
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Duration     int        `json:"duration" gorm:"default:0"` // 秒
	Command      string     `json:"command,omitempty" gorm:"type:text"`
	Output       string     `json:"output,omitempty" gorm:"type:text"`
	ErrorMessage string     `json:"error_message,omitempty" gorm:"type:text"`
	RetryCount   int        `json:"retry_count" gorm:"default:0"`
	MaxRetries   int        `json:"max_retries" gorm:"default:3"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联关系
	// 注意：为防止 GORM 在创建 install_steps 表时内联生成外键约束，禁用反向引用字段
	// 以避免出现 "relation \"node_install_tasks\" does not exist" 的 42P01 错误。
	// 该外键由 database.addForeignKeys() 在所有表创建完成后补充。
	Task NodeInstallTask `json:"task,omitempty" gorm:"-"`
}

// SSHExecutionLog SSH执行日志记录
type SSHExecutionLog struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	SessionID   string     `json:"session_id" gorm:"not null;size:100"`
	NodeID      uint       `json:"node_id"`
	TaskID      uint       `json:"task_id"`
	StepID      uint       `json:"step_id"`
	Host        string     `json:"host" gorm:"not null;size:255"`
	Port        int        `json:"port" gorm:"default:22"`
	Username    string     `json:"username" gorm:"not null;size:100"`
	Command     string     `json:"command" gorm:"not null;type:text"`
	ExitCode    int        `json:"exit_code"`
	Output      string     `json:"output,omitempty" gorm:"type:text"`
	ErrorOutput string     `json:"error_output,omitempty" gorm:"type:text"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int        `json:"duration" gorm:"default:0"` // 毫秒
	Success     bool       `json:"success" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at"`

	// 关联关系
	// 为避免内联外键，全部禁用 schema 级关联，通过 addForeignKeys() 统一处理外键
	Node SlurmNode       `json:"node,omitempty" gorm:"-"`
	Task NodeInstallTask `json:"task,omitempty" gorm:"-"`
	Step DeploymentStep  `json:"step,omitempty" gorm:"-"`
}

// NodeTemplate 节点模板模型
type NodeTemplate struct {
	ID          string         `json:"id" gorm:"primaryKey;size:100"`
	Name        string         `json:"name" gorm:"not null;size:100"`
	Description string         `json:"description" gorm:"size:500"`
	Config      NodeConfig     `json:"config" gorm:"type:json"`
	Tags        StringArray    `json:"tags" gorm:"type:json"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:CreatedBy"`
}

// JSON字段类型定义
type ClusterConfig struct {
	SlurmVersion   string            `json:"slurm_version"`
	SaltVersion    string            `json:"salt_version"`
	AccountingDB   string            `json:"accounting_db"`
	Partitions     []PartitionConfig `json:"partitions"`
	GlobalSettings map[string]string `json:"global_settings"`
	CustomPackages []string          `json:"custom_packages"`
}

type NodeConfig struct {
	Partitions     []string          `json:"partitions"`
	Features       []string          `json:"features"`
	CustomSettings map[string]string `json:"custom_settings"`
	Mounts         []MountConfig     `json:"mounts"`
}

type PartitionConfig struct {
	Name        string   `json:"name"`
	Nodes       []string `json:"nodes"`
	MaxTime     string   `json:"max_time"`
	DefaultTime string   `json:"default_time"`
	State       string   `json:"state"`
	Priority    int      `json:"priority"`
}

type MountConfig struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Type        string `json:"type"`
	Options     string `json:"options"`
}

type DeploymentConfig struct {
	Action         string            `json:"action"`
	TargetNodes    []string          `json:"target_nodes"`
	Parallel       int               `json:"parallel"`
	Timeout        int               `json:"timeout"`
	RetryCount     int               `json:"retry_count"`
	CustomSettings map[string]string `json:"custom_settings"`
}

type DeploymentResult struct {
	TotalNodes    int               `json:"total_nodes"`
	SuccessNodes  int               `json:"success_nodes"`
	FailedNodes   int               `json:"failed_nodes"`
	SkippedNodes  int               `json:"skipped_nodes"`
	TotalDuration int               `json:"total_duration"`
	NodeResults   map[string]string `json:"node_results"`
	FinalStatus   string            `json:"final_status"`
}

type InstallTaskConfig struct {
	PackageSource  string            `json:"package_source"`
	Version        string            `json:"version"`
	InstallType    string            `json:"install_type"` // package, binary, compile
	Dependencies   []string          `json:"dependencies"`
	ConfigTemplate string            `json:"config_template"`
	CustomSettings map[string]string `json:"custom_settings"`
}

type InstallTaskResult struct {
	InstalledVersion string          `json:"installed_version"`
	InstalledPath    string          `json:"installed_path"`
	ConfigFiles      []string        `json:"config_files"`
	Services         []string        `json:"services"`
	Ports            []int           `json:"ports"`
	ValidationTests  map[string]bool `json:"validation_tests"`
}

// 自定义JSON字段类型
type StringArray []string

func (sa StringArray) Value() (driver.Value, error) {
	return json.Marshal(sa)
}

func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan StringArray: unsupported type %T", value)
	}

	return json.Unmarshal(bytes, sa)
}

// API请求结构体
type CreateClusterRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	MasterHost  string              `json:"master_host" binding:"required"`
	MasterPort  int                 `json:"master_port"`
	SaltMaster  string              `json:"salt_master" binding:"required"`
	Config      ClusterConfig       `json:"config"`
	Nodes       []CreateNodeRequest `json:"nodes" binding:"required,min=1"`
}

type CreateNodeRequest struct {
	NodeName   string     `json:"node_name" binding:"required"`
	NodeType   string     `json:"node_type" binding:"required,oneof=master compute login"`
	Host       string     `json:"host" binding:"required"`
	Port       int        `json:"port"`
	Username   string     `json:"username" binding:"required"`
	AuthType   string     `json:"auth_type" binding:"oneof=password key"`
	Password   string     `json:"password"`
	KeyPath    string     `json:"key_path"`
	CPUs       int        `json:"cpus"`
	Memory     int        `json:"memory"`
	Storage    int        `json:"storage"`
	GPUs       int        `json:"gpus"`
	NodeConfig NodeConfig `json:"node_config"`
}

type ScaleClusterRequest struct {
	ClusterID   uint                `json:"cluster_id" binding:"required"`
	Action      string              `json:"action" binding:"required,oneof=scale-up scale-down"`
	Nodes       []CreateNodeRequest `json:"nodes" binding:"required_if=Action scale-up"`
	RemoveNodes []string            `json:"remove_nodes" binding:"required_if=Action scale-down"`
	Config      DeploymentConfig    `json:"config"`
}

type DeployClusterRequest struct {
	ClusterID uint             `json:"cluster_id" binding:"required"`
	Action    string           `json:"action" binding:"required,oneof=deploy update restart stop"`
	Config    DeploymentConfig `json:"config"`
}
