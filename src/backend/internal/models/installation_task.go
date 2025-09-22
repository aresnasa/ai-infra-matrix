package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// InstallationTask 安装任务记录
type InstallationTask struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskName    string    `json:"taskName" gorm:"not null"`
	TaskType    string    `json:"taskType" gorm:"not null"` // saltstack, slurm, combined
	Status      string    `json:"status" gorm:"not null"`   // pending, running, completed, failed
	TotalHosts  int       `json:"totalHosts"`
	SuccessHosts int      `json:"successHosts"`
	FailedHosts int       `json:"failedHosts"`
	StartTime   time.Time `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Duration    *int64    `json:"duration,omitempty"` // Duration in seconds
	Config      string    `json:"config" gorm:"type:text"` // JSON configuration
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	
	// 关联的主机结果
	HostResults []InstallationHostResult `json:"hostResults,omitempty" gorm:"foreignKey:TaskID"`
}

// InstallationHostResult 主机安装结果记录
type InstallationHostResult struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	TaskID   uint   `json:"taskId" gorm:"not null"`
	Host     string `json:"host" gorm:"not null"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Status   string `json:"status" gorm:"not null"` // success, failed
	Error    string `json:"error,omitempty"`
	Duration int64  `json:"duration"` // Duration in milliseconds
	Output   string `json:"output" gorm:"type:text"`
	StepsJSON string `json:"stepsJson,omitempty" gorm:"type:text"` // JSON encoded steps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// InstallationStep 安装步骤记录（嵌套在HostResult中）
type InstallationStep struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // success, failed, skipped
	Output      string    `json:"output"`
	Error       string    `json:"error,omitempty"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Duration    int64     `json:"duration"` // Duration in milliseconds
}

// SetSteps 设置安装步骤（序列化为JSON）
func (ihr *InstallationHostResult) SetSteps(steps []InstallationStep) error {
	stepsBytes, err := json.Marshal(steps)
	if err != nil {
		return err
	}
	ihr.StepsJSON = string(stepsBytes)
	return nil
}

// GetSteps 获取安装步骤（从JSON反序列化）
func (ihr *InstallationHostResult) GetSteps() ([]InstallationStep, error) {
	if ihr.StepsJSON == "" {
		return []InstallationStep{}, nil
	}
	
	var steps []InstallationStep
	err := json.Unmarshal([]byte(ihr.StepsJSON), &steps)
	if err != nil {
		return nil, err
	}
	return steps, nil
}

// SetConfig 设置任务配置（序列化为JSON）
func (it *InstallationTask) SetConfig(config interface{}) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	it.Config = string(configBytes)
	return nil
}

// GetConfig 获取任务配置（从JSON反序列化到指定类型）
func (it *InstallationTask) GetConfig(config interface{}) error {
	if it.Config == "" {
		return nil
	}
	return json.Unmarshal([]byte(it.Config), config)
}

// BeforeCreate GORM钩子
func (it *InstallationTask) BeforeCreate(tx *gorm.DB) error {
	it.CreatedAt = time.Now()
	it.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate GORM钩子
func (it *InstallationTask) BeforeUpdate(tx *gorm.DB) error {
	it.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate GORM钩子
func (ihr *InstallationHostResult) BeforeCreate(tx *gorm.DB) error {
	ihr.CreatedAt = time.Now()
	ihr.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate GORM钩子
func (ihr *InstallationHostResult) BeforeUpdate(tx *gorm.DB) error {
	ihr.UpdatedAt = time.Now()
	return nil
}

// MarkAsCompleted 标记任务为已完成
func (it *InstallationTask) MarkAsCompleted() {
	now := time.Now()
	it.EndTime = &now
	it.Status = "completed"
	if !it.StartTime.IsZero() {
		duration := now.Sub(it.StartTime).Milliseconds() / 1000 // Convert to seconds
		it.Duration = &duration
	}
}

// MarkAsFailed 标记任务为失败
func (it *InstallationTask) MarkAsFailed() {
	now := time.Now()
	it.EndTime = &now
	it.Status = "failed"
	if !it.StartTime.IsZero() {
		duration := now.Sub(it.StartTime).Milliseconds() / 1000 // Convert to seconds
		it.Duration = &duration
	}
}

// UpdateHostStats 更新主机统计信息
func (it *InstallationTask) UpdateHostStats() {
	it.SuccessHosts = 0
	it.FailedHosts = 0
	
	for _, result := range it.HostResults {
		if result.Status == "success" {
			it.SuccessHosts++
		} else {
			it.FailedHosts++
		}
	}
	
	// 如果所有主机都处理完成，更新任务状态
	if it.SuccessHosts+it.FailedHosts == it.TotalHosts {
		if it.FailedHosts == 0 {
			it.MarkAsCompleted()
		} else {
			it.Status = "completed" // 即使有失败也算完成，但可以通过FailedHosts判断
			now := time.Now()
			it.EndTime = &now
			if !it.StartTime.IsZero() {
				duration := now.Sub(it.StartTime).Milliseconds() / 1000
				it.Duration = &duration
			}
		}
	}
}