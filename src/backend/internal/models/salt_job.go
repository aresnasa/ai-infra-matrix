package models

import (
	"time"

	"gorm.io/gorm"
)

// SaltJobHistory Salt作业历史记录（持久化到数据库）
type SaltJobHistory struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	JID          string         `gorm:"uniqueIndex;size:64;not null" json:"jid"`       // Salt Job ID
	TaskID       string         `gorm:"index;size:64" json:"task_id,omitempty"`        // 前端生成的任务ID
	Function     string         `gorm:"size:128" json:"function"`                      // 执行的函数如 cmd.run
	Arguments    string         `gorm:"type:text" json:"arguments"`                    // 参数（JSON格式）
	Target       string         `gorm:"size:256" json:"target"`                        // 目标节点
	TgtType      string         `gorm:"size:32;default:'glob'" json:"tgt_type"`        // 目标类型
	User         string         `gorm:"size:64" json:"user"`                           // 执行用户
	Status       string         `gorm:"size:32;index;default:'running'" json:"status"` // 状态：running, completed, failed, timeout
	ReturnCode   int            `gorm:"default:0" json:"return_code"`                  // 返回码
	SuccessCount int            `gorm:"default:0" json:"success_count"`                // 成功节点数
	FailedCount  int            `gorm:"default:0" json:"failed_count"`                 // 失败节点数
	Result       string         `gorm:"type:text" json:"result,omitempty"`             // 执行结果（JSON格式）
	ErrorMessage string         `gorm:"type:text" json:"error_message,omitempty"`      // 错误信息
	StartTime    time.Time      `gorm:"index" json:"start_time"`                       // 开始时间
	EndTime      *time.Time     `json:"end_time,omitempty"`                            // 结束时间
	Duration     int64          `json:"duration,omitempty"`                            // 持续时间（毫秒）
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (SaltJobHistory) TableName() string {
	return "salt_job_histories"
}

// SaltJobConfig Salt作业配置（存储清理策略等）
type SaltJobConfig struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	MaxRetentionDays    int       `gorm:"default:30" json:"max_retention_days"`    // 最大保留天数
	MaxRecords          int       `gorm:"default:10000" json:"max_records"`        // 最大记录数
	CleanupEnabled      bool      `gorm:"default:true" json:"cleanup_enabled"`     // 是否启用自动清理
	CleanupIntervalHour int       `gorm:"default:24" json:"cleanup_interval_hour"` // 清理间隔（小时）
	LastCleanupTime     time.Time `json:"last_cleanup_time"`                       // 上次清理时间
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SaltJobConfig) TableName() string {
	return "salt_job_configs"
}

// SaltJobCreateRequest 创建作业记录请求
type SaltJobCreateRequest struct {
	JID       string   `json:"jid" binding:"required"`
	TaskID    string   `json:"task_id"`
	Function  string   `json:"function" binding:"required"`
	Arguments []string `json:"arguments"`
	Target    string   `json:"target" binding:"required"`
	TgtType   string   `json:"tgt_type"`
	User      string   `json:"user"`
}

// SaltJobUpdateRequest 更新作业状态请求
type SaltJobUpdateRequest struct {
	Status       string                 `json:"status"`
	ReturnCode   int                    `json:"return_code"`
	SuccessCount int                    `json:"success_count"`
	FailedCount  int                    `json:"failed_count"`
	Result       map[string]interface{} `json:"result"`
	ErrorMessage string                 `json:"error_message"`
}

// SaltJobQueryParams 查询参数
type SaltJobQueryParams struct {
	TaskID   string `form:"task_id"`
	JID      string `form:"jid"`
	Function string `form:"function"`
	Target   string `form:"target"`
	Status   string `form:"status"`
	User     string `form:"user"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	SortBy   string `form:"sort_by,default=start_time"`
	SortDesc bool   `form:"sort_desc,default=true"`
}

// SaltJobListResponse 作业列表响应
type SaltJobListResponse struct {
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
	Data  []SaltJobHistory `json:"data"`
}
