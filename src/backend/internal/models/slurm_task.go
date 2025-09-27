package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
	"gorm.io/gorm"
)

// SlurmTask SLURM任务记录模型
type SlurmTask struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	TaskID        string         `json:"task_id" gorm:"uniqueIndex;size:36;not null"` // UUID
	Name          string         `json:"name" gorm:"size:255;not null"`               // 任务名称
	Type          string         `json:"type" gorm:"size:50;not null"`                // 任务类型：scale-up, scale-down, deploy-minion等
	Status        string         `json:"status" gorm:"size:20;not null;default:'pending'"` // pending, running, completed, failed, cancelled
	UserID        uint           `json:"user_id" gorm:"not null"`                     // 执行任务的用户ID
	ClusterID     *uint          `json:"cluster_id"`                                  // 关联的集群ID（可选）
	
	// 任务执行信息
	StartedAt     *time.Time     `json:"started_at"`                                  // 开始时间
	CompletedAt   *time.Time     `json:"completed_at"`                                // 完成时间
	Duration      int64          `json:"duration"`                                    // 执行时长（秒）
	Progress      float64        `json:"progress" gorm:"default:0"`                   // 进度 0-1
	
	// 任务参数和结果
	Parameters    JSON           `json:"parameters" gorm:"type:jsonb"`                // 任务参数
	Results       JSON           `json:"results" gorm:"type:jsonb"`                   // 执行结果
	ErrorMessage  string         `json:"error_message" gorm:"type:text"`              // 错误信息
	
	// 执行详情
	StepsCurrent  string         `json:"steps_current" gorm:"size:100"`               // 当前执行步骤
	StepsTotal    int            `json:"steps_total" gorm:"default:0"`                // 总步骤数
	StepsCount    int            `json:"steps_count" gorm:"default:0"`                // 已完成步骤数
	
	// 资源信息
	TargetNodes   StringArray    `json:"target_nodes" gorm:"type:text"`               // 目标节点列表
	NodesTotal    int            `json:"nodes_total" gorm:"default:0"`                // 总节点数
	NodesSuccess  int            `json:"nodes_success" gorm:"default:0"`              // 成功节点数
	NodesFailed   int            `json:"nodes_failed" gorm:"default:0"`               // 失败节点数
	
	// 日志和事件
	ExecutionLogs string         `json:"execution_logs" gorm:"type:text"`             // 执行日志
	EventHistory  JSON           `json:"event_history" gorm:"type:jsonb"`             // 事件历史记录
	
	// 元数据
	Tags          StringArray    `json:"tags" gorm:"type:text"`                       // 任务标签
	Priority      int            `json:"priority" gorm:"default:0"`                   // 任务优先级
	RetryCount    int            `json:"retry_count" gorm:"default:0"`                // 重试次数
	MaxRetries    int            `json:"max_retries" gorm:"default:3"`                // 最大重试次数
	
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联
	User          *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Cluster       *Cluster       `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
}

// SlurmTaskEvent 任务事件记录
type SlurmTaskEvent struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	TaskID      uint           `json:"task_id" gorm:"not null"`                      // 关联的任务ID
	EventType   string         `json:"event_type" gorm:"size:50;not null"`           // 事件类型：start, step-start, step-done, error, complete等
	Step        string         `json:"step" gorm:"size:100"`                         // 步骤名称
	Message     string         `json:"message" gorm:"type:text"`                     // 事件消息
	Host        string         `json:"host" gorm:"size:255"`                         // 相关主机
	Progress    float64        `json:"progress" gorm:"default:0"`                    // 当前进度
	Data        JSON           `json:"data" gorm:"type:jsonb"`                       // 附加数据
	Timestamp   time.Time      `json:"timestamp" gorm:"not null"`                    // 事件时间戳
	CreatedAt   time.Time      `json:"created_at"`
	
	// 关联
	Task        *SlurmTask     `json:"task,omitempty" gorm:"foreignKey:TaskID"`
}

// SlurmTaskStatistics 任务统计信息
type SlurmTaskStatistics struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Date              time.Time `json:"date" gorm:"uniqueIndex;not null"`             // 统计日期
	TasksTotal        int       `json:"tasks_total" gorm:"default:0"`                 // 总任务数
	TasksCompleted    int       `json:"tasks_completed" gorm:"default:0"`             // 完成任务数
	TasksFailed       int       `json:"tasks_failed" gorm:"default:0"`                // 失败任务数
	TasksCancelled    int       `json:"tasks_cancelled" gorm:"default:0"`             // 取消任务数
	NodesDeployed     int       `json:"nodes_deployed" gorm:"default:0"`              // 部署节点数
	NodesRemoved      int       `json:"nodes_removed" gorm:"default:0"`               // 移除节点数
	AvgExecutionTime  float64   `json:"avg_execution_time" gorm:"default:0"`          // 平均执行时间（秒）
	TotalExecutionTime float64  `json:"total_execution_time" gorm:"default:0"`        // 总执行时间（秒）
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SlurmTask) TableName() string {
	return "slurm_tasks"
}

func (SlurmTaskEvent) TableName() string {
	return "slurm_task_events"
}

func (SlurmTaskStatistics) TableName() string {
	return "slurm_task_statistics"
}

// BeforeCreate 创建前回调
func (st *SlurmTask) BeforeCreate(tx *gorm.DB) error {
	if st.StartedAt == nil {
		now := time.Now()
		st.StartedAt = &now
	}
	return nil
}

// BeforeUpdate 更新前回调
func (st *SlurmTask) BeforeUpdate(tx *gorm.DB) error {
	// 计算执行时长
	if st.CompletedAt != nil && st.StartedAt != nil {
		st.Duration = int64(st.CompletedAt.Sub(*st.StartedAt).Seconds())
	}
	
	// 更新节点统计
	if st.TargetNodes != nil {
		st.NodesTotal = len(st.TargetNodes)
	}
	
	return nil
}

// GetFormattedDuration 获取格式化的执行时长
func (st *SlurmTask) GetFormattedDuration() string {
	if st.Duration == 0 {
		if st.StartedAt != nil {
			duration := time.Since(*st.StartedAt)
			return duration.Truncate(time.Second).String()
		}
		return "0s"
	}
	
	duration := time.Duration(st.Duration) * time.Second
	return duration.String()
}

// IsCompleted 检查任务是否已完成
func (st *SlurmTask) IsCompleted() bool {
	return st.Status == "completed" || st.Status == "failed" || st.Status == "cancelled"
}

// GetSuccessRate 获取成功率
func (st *SlurmTask) GetSuccessRate() float64 {
	if st.NodesTotal == 0 {
		return 0
	}
	return float64(st.NodesSuccess) / float64(st.NodesTotal) * 100
}

// AddEvent 添加任务事件
func (st *SlurmTask) AddEvent(db *gorm.DB, eventType, step, message, host string, progress float64, data interface{}) error {
	var dataJSON JSON
	if data != nil {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		dataJSON = JSON(dataBytes)
	}
	
	event := SlurmTaskEvent{
		TaskID:    st.ID,
		EventType: eventType,
		Step:      step,
		Message:   message,
		Host:      host,
		Progress:  progress,
		Data:      dataJSON,
		Timestamp: time.Now(),
	}
	
	return db.Create(&event).Error
}

// UpdateProgress 更新任务进度
func (st *SlurmTask) UpdateProgress(db *gorm.DB, progress float64, currentStep string) error {
	st.Progress = progress
	st.StepsCurrent = currentStep
	return db.Model(st).Updates(map[string]interface{}{
		"progress":      progress,
		"steps_current": currentStep,
	}).Error
}

// Complete 完成任务
func (st *SlurmTask) Complete(db *gorm.DB, status string, errorMsg ...string) error {
	now := time.Now()
	st.CompletedAt = &now
	st.Status = status
	st.Progress = 1.0
	
	updates := map[string]interface{}{
		"completed_at": now,
		"status":       status,
		"progress":     1.0,
	}
	
	if len(errorMsg) > 0 && errorMsg[0] != "" {
		st.ErrorMessage = errorMsg[0]
		updates["error_message"] = errorMsg[0]
	}
	
	return db.Model(st).Updates(updates).Error
}

// JSON 自定义JSON字段类型
type JSON []byte

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = make(JSON, len(v))
		copy(*j, v)
	case string:
		*j = JSON(v)
	default:
		return fmt.Errorf("cannot scan %T into JSON", value)
	}

	return nil
}