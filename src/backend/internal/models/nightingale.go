package models

// NightingaleUser Nightingale 用户表
type NightingaleUser struct {
	ID             uint   `gorm:"primaryKey;autoIncrement"`
	Username       string `gorm:"uniqueIndex;not null;size:64"`
	Nickname       string `gorm:"not null;size:64"`
	Password       string `gorm:"not null;default:'';size:128"`
	Phone          string `gorm:"not null;default:'';size:16"`
	Email          string `gorm:"not null;default:'';size:64"`
	Portrait       string `gorm:"not null;default:'';size:255"`
	Roles          string `gorm:"not null;size:255"`  // Admin, Standard, Guest
	Contacts       string `gorm:"type:text"`          // JSON format
	Maintainer     int    `gorm:"not null;default:0"` // 是否为维护者
	Belong         string `gorm:"not null;default:'';size:16"`
	LastActiveTime int64  `gorm:"not null;default:0"`
	CreateAt       int64  `gorm:"not null;default:0"`
	CreateBy       string `gorm:"not null;default:'';size:64"`
	UpdateAt       int64  `gorm:"not null;default:0"`
	UpdateBy       string `gorm:"not null;default:'';size:64"`
}

func (NightingaleUser) TableName() string {
	return "users"
}

// NightingaleUserGroup Nightingale 用户组表
type NightingaleUserGroup struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"not null;default:'';size:128"`
	Note     string `gorm:"not null;default:'';size:255"`
	CreateAt int64  `gorm:"not null;default:0;index:idx_user_group_create_at"`
	CreateBy string `gorm:"not null;default:'';size:64;index:idx_user_group_create_by"`
	UpdateAt int64  `gorm:"not null;default:0;index:idx_user_group_update_at"`
	UpdateBy string `gorm:"not null;default:'';size:64"`
}

func (NightingaleUserGroup) TableName() string {
	return "user_group"
}

// NightingaleUserGroupMember Nightingale 用户组成员表
type NightingaleUserGroupMember struct {
	ID      uint  `gorm:"primaryKey;autoIncrement"`
	GroupID int64 `gorm:"not null;index:idx_user_group_member_group_id"`
	UserID  int64 `gorm:"not null;index:idx_user_group_member_user_id"`
}

func (NightingaleUserGroupMember) TableName() string {
	return "user_group_member"
}

// NightingaleRole Nightingale 角色表
type NightingaleRole struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"uniqueIndex;not null;default:'';size:191"`
	Note string `gorm:"not null;default:'';size:255"`
}

func (NightingaleRole) TableName() string {
	return "role"
}

// NightingaleTarget Nightingale 监控目标（主机）表
type NightingaleTarget struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Ident    string `gorm:"uniqueIndex;not null;size:191"` // 主机标识符（通常是 hostname 或 IP）
	Note     string `gorm:"not null;default:'';size:255"`
	Tags     string `gorm:"not null;default:'';size:512"` // 标签，用于分组和过滤
	UpdateAt int64  `gorm:"not null;default:0"`
}

func (NightingaleTarget) TableName() string {
	return "target"
}

// NightingaleBusiGroup Nightingale 业务组表
type NightingaleBusiGroup struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"not null;size:191"`
	LabelEnable int    `gorm:"not null;default:0"`
	LabelValue  string `gorm:"not null;default:'';size:191"`
	CreateAt    int64  `gorm:"not null;default:0"`
	CreateBy    string `gorm:"not null;default:'';size:64"`
	UpdateAt    int64  `gorm:"not null;default:0"`
	UpdateBy    string `gorm:"not null;default:'';size:64"`
}

func (NightingaleBusiGroup) TableName() string {
	return "busi_group"
}

// NightingaleBusiGroupMember Nightingale 业务组成员表
type NightingaleBusiGroupMember struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	BusiGroupID int64  `gorm:"not null;index:idx_busi_group_member_busi_group_id"`
	UserGroupID int64  `gorm:"not null;index:idx_busi_group_member_user_group_id"`
	PermFlag    string `gorm:"not null;size:2"` // ro: read-only, rw: read-write
}

func (NightingaleBusiGroupMember) TableName() string {
	return "busi_group_member"
}

// InitNightingaleModels 返回所有需要迁移的 Nightingale 模型
func InitNightingaleModels() []interface{} {
	return []interface{}{
		&NightingaleUser{},
		&NightingaleUserGroup{},
		&NightingaleUserGroupMember{},
		&NightingaleRole{},
		&NightingaleTarget{},
		&NightingaleBusiGroup{},
		&NightingaleBusiGroupMember{},
		&NightingaleConfig{},
		&NightingaleRoleOperation{},
		&NightingaleBoard{},
		&NightingaleBoardPayload{},
		&NightingaleChartShare{},
		&NightingaleAlertRule{},
		&NightingaleAlertMute{},
		&NightingaleAlertSubscribe{},
		&NightingaleMetricView{},
		&NightingaleRecordingRule{},
		&NightingaleAlertAggrView{},
		&NightingaleAlertCurEvent{},
		&NightingaleAlertHisEvent{},
		&NightingaleTaskTpl{},
		&NightingaleTaskTplHost{},
		&NightingaleTaskRecord{},
		&NightingaleAlertingEngine{},
		&NightingaleDatasource{},
		&NightingaleBuiltinCate{},
		&NightingaleNotifyTpl{},
		&NightingaleSsoConfig{},
		&NightingaleEsIndexPattern{},
		&NightingaleBuiltinMetric{},
		&NightingaleMetricFilter{},
		&NightingaleBoardBusiGroup{},
		&NightingaleBuiltinComponent{},
		&NightingaleBuiltinPayload{},
		&NightingaleDashAnnotation{},
		&NightingaleSourceToken{},
		&NightingaleNotificationRecord{},
		&NightingaleTargetBusiGroup{},
		&NightingaleUserToken{},
		&NightingaleNotifyRule{},
		&NightingaleNotifyChannel{},
		&NightingaleMessageTemplate{},
		&NightingaleEventPipeline{},
		&NightingaleEmbeddedProduct{},
	}
}

// NightingaleMonitoringAgent 监控客户端配置
type NightingaleMonitoringAgent struct {
	Hostname   string
	IP         string
	Tags       []string
	BusiGroup  string
	Collectors []string // 需要启用的采集器列表
}
