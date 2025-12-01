package models

import (
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
// 用于实现软删除 + 后台异步真实删除
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
