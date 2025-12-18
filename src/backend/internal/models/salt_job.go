package models

import (
	"encoding/json"
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
	ID                    uint      `gorm:"primaryKey" json:"id"`
	MaxRetentionDays      int       `gorm:"default:30" json:"max_retention_days"`               // 最大保留天数
	MaxRecords            int       `gorm:"default:10000" json:"max_records"`                   // 最大记录数
	CleanupEnabled        bool      `gorm:"default:true" json:"cleanup_enabled"`                // 是否启用自动清理
	CleanupIntervalHour   int       `gorm:"default:24" json:"cleanup_interval_hour"`            // 清理间隔（转换后的小时数）
	CleanupIntervalValue  int       `gorm:"default:1" json:"cleanup_interval_value"`            // 清理间隔值
	CleanupIntervalUnit   string    `gorm:"size:16;default:'day'" json:"cleanup_interval_unit"` // 清理间隔单位: hour, day, month, year
	LastCleanupTime       time.Time `json:"last_cleanup_time"`                                  // 上次清理时间
	DangerousCommandsJSON string    `gorm:"type:text;column:dangerous_commands" json:"-"`       // 危险命令黑名单（JSON存储）
	BlacklistEnabled      bool      `gorm:"default:true" json:"blacklist_enabled"`              // 是否启用黑名单检查
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// CalculateCleanupIntervalHour 根据值和单位计算小时数
func (c *SaltJobConfig) CalculateCleanupIntervalHour() int {
	value := c.CleanupIntervalValue
	if value < 1 {
		value = 1
	}
	switch c.CleanupIntervalUnit {
	case "hour":
		return value
	case "day":
		return value * 24
	case "month":
		return value * 24 * 30 // 近似30天
	case "year":
		return value * 24 * 365 // 近似365天
	default:
		return value * 24 // 默认按天
	}
}

// DangerousCommand 危险命令配置
type DangerousCommand struct {
	Pattern     string `json:"pattern"`     // 命令模式（支持正则表达式）
	IsRegex     bool   `json:"is_regex"`    // 是否为正则表达式
	Description string `json:"description"` // 描述说明
	Severity    string `json:"severity"`    // 危险等级: critical, high, medium, low
	Enabled     bool   `json:"enabled"`     // 是否启用
}

// GetDangerousCommands 获取危险命令列表
func (c *SaltJobConfig) GetDangerousCommands() []DangerousCommand {
	if c.DangerousCommandsJSON == "" {
		return GetDefaultDangerousCommands()
	}
	var commands []DangerousCommand
	if err := json.Unmarshal([]byte(c.DangerousCommandsJSON), &commands); err != nil {
		return GetDefaultDangerousCommands()
	}
	return commands
}

// SetDangerousCommands 设置危险命令列表
func (c *SaltJobConfig) SetDangerousCommands(commands []DangerousCommand) error {
	data, err := json.Marshal(commands)
	if err != nil {
		return err
	}
	c.DangerousCommandsJSON = string(data)
	return nil
}

// GetDefaultDangerousCommands 获取默认危险命令列表
func GetDefaultDangerousCommands() []DangerousCommand {
	return []DangerousCommand{
		{Pattern: "rm -rf /", IsRegex: false, Description: "删除根目录所有文件", Severity: "critical", Enabled: true},
		{Pattern: "rm -rf /*", IsRegex: false, Description: "删除根目录所有文件", Severity: "critical", Enabled: true},
		{Pattern: `rm\s+(-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*|-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*)\s+/\s*$`, IsRegex: true, Description: "递归强制删除根目录", Severity: "critical", Enabled: true},
		{Pattern: "dd if=/dev/zero of=/dev/sd", IsRegex: false, Description: "清空磁盘数据", Severity: "critical", Enabled: true},
		{Pattern: "mkfs", IsRegex: false, Description: "格式化文件系统", Severity: "critical", Enabled: true},
		{Pattern: ":(){:|:&};:", IsRegex: false, Description: "Fork炸弹", Severity: "critical", Enabled: true},
		{Pattern: "chmod -R 777 /", IsRegex: false, Description: "修改根目录权限", Severity: "high", Enabled: true},
		{Pattern: "chown -R", IsRegex: false, Description: "递归修改所有者", Severity: "high", Enabled: true},
		{Pattern: "> /dev/sda", IsRegex: false, Description: "清空磁盘设备", Severity: "critical", Enabled: true},
		{Pattern: "mv /* /dev/null", IsRegex: false, Description: "移动所有文件到空设备", Severity: "critical", Enabled: true},
		{Pattern: "wget.*|.*sh", IsRegex: true, Description: "下载并执行脚本（需审核）", Severity: "medium", Enabled: false},
		{Pattern: "curl.*|.*sh", IsRegex: true, Description: "下载并执行脚本（需审核）", Severity: "medium", Enabled: false},
		{Pattern: "shutdown", IsRegex: false, Description: "关机命令", Severity: "high", Enabled: true},
		{Pattern: "reboot", IsRegex: false, Description: "重启命令", Severity: "high", Enabled: true},
		{Pattern: "halt", IsRegex: false, Description: "停止系统", Severity: "high", Enabled: true},
		{Pattern: "init 0", IsRegex: false, Description: "关机", Severity: "high", Enabled: true},
		{Pattern: "init 6", IsRegex: false, Description: "重启", Severity: "high", Enabled: true},
		{Pattern: "systemctl poweroff", IsRegex: false, Description: "关机", Severity: "high", Enabled: true},
		{Pattern: "systemctl reboot", IsRegex: false, Description: "重启", Severity: "high", Enabled: true},
		{Pattern: `echo\s+.*\s*>\s*/etc/passwd`, IsRegex: true, Description: "覆盖密码文件", Severity: "critical", Enabled: true},
		{Pattern: `echo\s+.*\s*>\s*/etc/shadow`, IsRegex: true, Description: "覆盖密码文件", Severity: "critical", Enabled: true},
		{Pattern: "userdel root", IsRegex: false, Description: "删除root用户", Severity: "critical", Enabled: true},
		{Pattern: "passwd -d root", IsRegex: false, Description: "删除root密码", Severity: "critical", Enabled: true},
	}
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
	UserOnly bool   `form:"user_only,default=true"` // 默认只返回用户任务（有 task_id 的）
}

// SaltJobListResponse 作业列表响应
type SaltJobListResponse struct {
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
	Data  []SaltJobHistory `json:"data"`
}
