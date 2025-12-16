package models

import (
	"time"

	"gorm.io/gorm"
)

// SaltJob Salt作业持久化记录
type SaltJob struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	JID       string     `json:"jid" gorm:"uniqueIndex;type:varchar(32);not null"` // Salt Job ID (YYYYMMDDHHMMSSxxxxxx)
	TaskID    string     `json:"task_id" gorm:"index;type:varchar(64)"`            // 前端生成的任务ID (EXEC-YYYYMMDD-HHMMSS-xxxx-xxxx)
	Function  string     `json:"function" gorm:"size:128;not null"`                // Salt函数 (cmd.run, state.apply等)
	Target    string     `json:"target" gorm:"size:255;not null"`                  // 目标 minion
	Arguments string     `json:"arguments" gorm:"type:text"`                       // 命令参数 (JSON数组字符串)
	User      string     `json:"user" gorm:"size:64"`                              // 执行用户
	Status    string     `json:"status" gorm:"size:20;not null;default:'running'"` // running, completed, failed, timeout
	Result    JSON       `json:"result" gorm:"type:jsonb"`                         // 执行结果 (JSON)
	StartTime time.Time  `json:"start_time" gorm:"not null"`                       // 开始时间
	EndTime   *time.Time `json:"end_time"`                                         // 结束时间
	Duration  int64      `json:"duration"`                                         // 执行时长（毫秒）

	// 扩展信息
	MinionCount  int    `json:"minion_count" gorm:"default:0"`  // 目标minion数量
	SuccessCount int    `json:"success_count" gorm:"default:0"` // 成功minion数量
	FailedCount  int    `json:"failed_count" gorm:"default:0"`  // 失败minion数量
	ErrorMessage string `json:"error_message" gorm:"type:text"` // 错误信息

	// 元数据
	Source string `json:"source" gorm:"size:32;default:'api'"` // 来源：api, scheduler, manual
	Tags   string `json:"tags" gorm:"type:text"`               // 标签 (JSON数组)

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (SaltJob) TableName() string {
	return "salt_jobs"
}

// SaltJobListRequest 作业列表查询请求
type SaltJobListRequest struct {
	Page      int    `form:"page" json:"page"`             // 页码，从1开始
	PageSize  int    `form:"page_size" json:"page_size"`   // 每页数量
	TaskID    string `form:"task_id" json:"task_id"`       // 按TaskID过滤
	JID       string `form:"jid" json:"jid"`               // 按JID过滤
	Function  string `form:"function" json:"function"`     // 按函数过滤
	Target    string `form:"target" json:"target"`         // 按目标过滤
	Status    string `form:"status" json:"status"`         // 按状态过滤
	User      string `form:"user" json:"user"`             // 按用户过滤
	StartFrom string `form:"start_from" json:"start_from"` // 开始时间范围起点
	StartTo   string `form:"start_to" json:"start_to"`     // 开始时间范围终点
	Keyword   string `form:"keyword" json:"keyword"`       // 关键词搜索（TaskID, JID, Function）
}

// SaltJobListResponse 作业列表响应
type SaltJobListResponse struct {
	Total int64     `json:"total"`
	Page  int       `json:"page"`
	Size  int       `json:"size"`
	Data  []SaltJob `json:"data"`
}
