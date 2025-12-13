package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// MinionDeleteStatus 删除任务状态
const (
	MinionDeleteStatusPending   = "pending"   // 待删除（软删除状态，前端显示为删除中）
	MinionDeleteStatusDeleting  = "deleting"  // 正在执行删除
	MinionDeleteStatusCompleted = "completed" // 删除完成
	MinionDeleteStatusFailed    = "failed"    // 删除失败
	MinionDeleteStatusCancelled = "cancelled" // 已取消
)

// MinionDeleteTask Minion 删除任务记录
// 用于实现软删除 + 后台异步真实删除 + SSH 远程卸载
type MinionDeleteTask struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	MinionID     string         `json:"minion_id" gorm:"not null;index;size:255"`
	MasterURL    string         `json:"master_url" gorm:"size:500"`           // Salt Master URL
	Status       string         `json:"status" gorm:"not null;size:50;index"` // pending, deleting, completed, failed, cancelled
	Force        bool           `json:"force" gorm:"default:false"`           // 是否强制删除
	ErrorMessage string         `json:"error_message,omitempty" gorm:"type:text"`
	RetryCount   int            `json:"retry_count" gorm:"default:0"`
	MaxRetries   int            `json:"max_retries" gorm:"default:3"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	CreatedBy    string         `json:"created_by,omitempty" gorm:"size:100"`

	// SSH 远程卸载配置（可选，用于彻底卸载远程节点上的 salt-minion）
	SSHHost     string `json:"ssh_host,omitempty" gorm:"size:255"`     // SSH 主机地址
	SSHPort     int    `json:"ssh_port,omitempty" gorm:"default:22"`   // SSH 端口
	SSHUsername string `json:"ssh_username,omitempty" gorm:"size:100"` // SSH 用户名
	SSHPassword string `json:"-" gorm:"size:500"`                      // SSH 密码（加密存储，不返回给前端）
	SSHKeyPath  string `json:"ssh_key_path,omitempty" gorm:"size:500"` // SSH 私钥路径
	UseSudo     bool   `json:"use_sudo" gorm:"default:false"`          // 是否使用 sudo
	Uninstall   bool   `json:"uninstall" gorm:"default:false"`         // 是否执行远程卸载（不仅仅是删除密钥）

	// 详细步骤日志（JSON 格式存储）
	StepsJSON string `json:"steps_json,omitempty" gorm:"type:text"` // JSON encoded steps
	Duration  int64  `json:"duration,omitempty"`                    // 执行时长（毫秒）

	// 关联的详细日志
	Logs []MinionDeleteLog `json:"logs,omitempty" gorm:"foreignKey:TaskID"`
}

// MinionDeleteLog 删除任务详细日志
type MinionDeleteLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    uint      `json:"task_id" gorm:"not null;index"`
	Step      string    `json:"step" gorm:"not null;size:100"`     // 步骤名称
	Status    string    `json:"status" gorm:"size:50"`             // success, failed, running
	Message   string    `json:"message" gorm:"type:text"`          // 日志消息
	Output    string    `json:"output,omitempty" gorm:"type:text"` // 命令输出
	Error     string    `json:"error,omitempty" gorm:"type:text"`  // 错误信息
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (MinionDeleteLog) TableName() string {
	return "minion_delete_logs"
}

// DeleteStep 删除步骤记录（嵌套在Task中）
type DeleteStep struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // success, failed, skipped, running
	Output      string    `json:"output,omitempty"`
	Error       string    `json:"error,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Duration    int64     `json:"duration"` // Duration in milliseconds
}

// SetSteps 设置删除步骤（序列化为JSON）
func (m *MinionDeleteTask) SetSteps(steps []DeleteStep) error {
	stepsBytes, err := json.Marshal(steps)
	if err != nil {
		return err
	}
	m.StepsJSON = string(stepsBytes)
	return nil
}

// GetSteps 获取删除步骤（从JSON反序列化）
func (m *MinionDeleteTask) GetSteps() ([]DeleteStep, error) {
	if m.StepsJSON == "" {
		return []DeleteStep{}, nil
	}

	var steps []DeleteStep
	err := json.Unmarshal([]byte(m.StepsJSON), &steps)
	if err != nil {
		return nil, err
	}
	return steps, nil
}

// AddStep 添加单个步骤到步骤列表
func (m *MinionDeleteTask) AddStep(step DeleteStep) error {
	steps, err := m.GetSteps()
	if err != nil {
		steps = []DeleteStep{}
	}
	steps = append(steps, step)
	return m.SetSteps(steps)
}

// TableName 指定表名
func (MinionDeleteTask) TableName() string {
	return "minion_delete_tasks"
}

// IsPending 检查是否为待删除状态
func (m *MinionDeleteTask) IsPending() bool {
	return m.Status == MinionDeleteStatusPending || m.Status == MinionDeleteStatusDeleting
}

// IsCompleted 检查是否已完成
func (m *MinionDeleteTask) IsCompleted() bool {
	return m.Status == MinionDeleteStatusCompleted
}

// IsFailed 检查是否失败
func (m *MinionDeleteTask) IsFailed() bool {
	return m.Status == MinionDeleteStatusFailed
}

// CanRetry 检查是否可以重试
func (m *MinionDeleteTask) CanRetry() bool {
	return m.Status == MinionDeleteStatusFailed && m.RetryCount < m.MaxRetries
}

// MarkAsDeleting 标记为正在删除
func (m *MinionDeleteTask) MarkAsDeleting() {
	m.Status = MinionDeleteStatusDeleting
}

// MarkAsCompleted 标记为已完成
func (m *MinionDeleteTask) MarkAsCompleted() {
	m.Status = MinionDeleteStatusCompleted
	now := time.Now()
	m.CompletedAt = &now
}

// MarkAsFailed 标记为失败
func (m *MinionDeleteTask) MarkAsFailed(err string) {
	m.Status = MinionDeleteStatusFailed
	m.ErrorMessage = err
	m.RetryCount++
}

// MarkAsCancelled 标记为已取消
func (m *MinionDeleteTask) MarkAsCancelled() {
	m.Status = MinionDeleteStatusCancelled
}
