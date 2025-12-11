package models

// NightingaleConfig 配置表
type NightingaleConfig struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Ckey      string `gorm:"uniqueIndex;not null;size:191"`
	Cval      string `gorm:"not null;default:'';type:text"`
	Note      string `gorm:"not null;default:'';size:1024"`
	External  int    `gorm:"not null;default:0"`
	Encrypted int    `gorm:"not null;default:0"`
	CreateAt  int64  `gorm:"not null;default:0"`
	CreateBy  string `gorm:"not null;default:'';size:64"`
	UpdateAt  int64  `gorm:"not null;default:0"`
	UpdateBy  string `gorm:"not null;default:'';size:64"`
}

func (NightingaleConfig) TableName() string {
	return "configs"
}

// NightingaleRoleOperation 角色操作权限表
type NightingaleRoleOperation struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	RoleName  string `gorm:"not null;size:128;index:idx_role_operation_role_name"`
	Operation string `gorm:"not null;size:191;index:idx_role_operation_operation"`
}

func (NightingaleRoleOperation) TableName() string {
	return "role_operation"
}

// NightingaleBoard 仪表盘表
type NightingaleBoard struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	GroupID    int64  `gorm:"not null;default:0;uniqueIndex:idx_board_group_name"`
	Name       string `gorm:"not null;size:191;uniqueIndex:idx_board_group_name"`
	Ident      string `gorm:"not null;default:'';size:200;index:idx_board_ident"`
	Tags       string `gorm:"not null;size:255"`
	Public     int    `gorm:"not null;default:0"`
	BuiltIn    int    `gorm:"not null;default:0"`
	Hide       int    `gorm:"not null;default:0"`
	PublicCate int    `gorm:"not null;default:0"`
	CreateAt   int64  `gorm:"not null;default:0"`
	CreateBy   string `gorm:"not null;default:'';size:64"`
	UpdateAt   int64  `gorm:"not null;default:0"`
	UpdateBy   string `gorm:"not null;default:'';size:64"`
	Note       string `gorm:"not null;default:'';size:1024"`
}

func (NightingaleBoard) TableName() string {
	return "board"
}

// NightingaleBoardPayload 仪表盘内容表
type NightingaleBoardPayload struct {
	ID      uint   `gorm:"primaryKey;autoIncrement:false"` // Same as Board ID
	Payload string `gorm:"not null;type:text"`
}

func (NightingaleBoardPayload) TableName() string {
	return "board_payload"
}

// NightingaleChartShare 图表分享表
type NightingaleChartShare struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	Cluster      string `gorm:"not null;size:128"`
	DatasourceID int64  `gorm:"not null;default:0"`
	Configs      string `gorm:"type:text"`
	CreateAt     int64  `gorm:"not null;default:0;index:idx_chart_share_create_at"`
	CreateBy     string `gorm:"not null;default:'';size:64"`
}

func (NightingaleChartShare) TableName() string {
	return "chart_share"
}

// NightingaleAlertRule 告警规则表
type NightingaleAlertRule struct {
	ID               uint   `gorm:"primaryKey;autoIncrement"`
	GroupID          int64  `gorm:"not null;default:0;index:idx_alert_rule_group_id"`
	Cate             string `gorm:"not null;size:128"`
	DatasourceIDs    string `gorm:"not null;default:'';size:255"`
	Cluster          string `gorm:"not null;size:128"`
	Name             string `gorm:"not null;size:255"`
	Note             string `gorm:"not null;default:'';size:1024"`
	Prod             string `gorm:"not null;default:'';size:255"`
	Algorithm        string `gorm:"not null;default:'';size:255"`
	AlgoParams       string `gorm:"size:255"`
	Delay            int    `gorm:"not null;default:0"`
	Severity         int    `gorm:"not null"`
	Disabled         int    `gorm:"not null"`
	PromForDuration  int    `gorm:"not null"`
	RuleConfig       string `gorm:"not null;type:text"`
	PromQL           string `gorm:"not null;type:text"`
	PromEvalInterval int    `gorm:"not null"`
	EnableStime      string `gorm:"not null;default:'00:00';size:255"`
	EnableEtime      string `gorm:"not null;default:'23:59';size:255"`
	EnableDaysOfWeek string `gorm:"not null;default:'';size:255"`
	EnableInBg       int    `gorm:"not null;default:0"`
	NotifyRecovered  int    `gorm:"not null"`
	NotifyChannels   string `gorm:"not null;default:'';size:255"`
	NotifyGroups     string `gorm:"not null;default:'';size:255"`
	NotifyRepeatStep int    `gorm:"not null;default:0"`
	NotifyMaxNumber  int    `gorm:"not null;default:0"`
	RecoverDuration  int    `gorm:"not null;default:0"`
	Callbacks        string `gorm:"not null;default:'';size:255"`
	RunbookURL       string `gorm:"size:255"`
	AppendTags       string `gorm:"not null;default:'';size:255"`
	Annotations      string `gorm:"not null;type:text"`
	ExtraConfig      string `gorm:"not null;type:text"`
	CreateAt         int64  `gorm:"not null;default:0"`
	CreateBy         string `gorm:"not null;default:'';size:64"`
	UpdateAt         int64  `gorm:"not null;default:0;index:idx_alert_rule_update_at"`
	UpdateBy         string `gorm:"not null;default:'';size:64"`
}

func (NightingaleAlertRule) TableName() string {
	return "alert_rule"
}

// NightingaleAlertMute 告警屏蔽表
type NightingaleAlertMute struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	GroupID       int64  `gorm:"not null;default:0;index:idx_alert_mute_group_id"`
	Prod          string `gorm:"not null;default:'';size:255"`
	Note          string `gorm:"not null;default:'';size:1024"`
	Cate          string `gorm:"not null;size:128"`
	Cluster       string `gorm:"not null;size:128"`
	DatasourceIDs string `gorm:"not null;default:'';size:255"`
	Tags          string `gorm:"not null;type:jsonb;default:'[]'"` // Postgres specific
	Cause         string `gorm:"not null;default:'';size:255"`
	Btime         int64  `gorm:"not null;default:0"`
	Etime         int64  `gorm:"not null;default:0"`
	Disabled      int    `gorm:"not null;default:0"`
	MuteTimeType  int    `gorm:"not null;default:0"`
	PeriodicMutes string `gorm:"not null;default:'';size:4096"`
	Severities    string `gorm:"not null;default:'';size:32"`
	CreateAt      int64  `gorm:"not null;default:0"`
	CreateBy      string `gorm:"not null;default:'';size:64"`
	UpdateAt      int64  `gorm:"not null;default:0;index:idx_alert_mute_update_at"`
	UpdateBy      string `gorm:"not null;default:'';size:64"`
}

func (NightingaleAlertMute) TableName() string {
	return "alert_mute"
}

// NightingaleAlertSubscribe 告警订阅表
type NightingaleAlertSubscribe struct {
	ID               uint   `gorm:"primaryKey;autoIncrement"`
	Name             string `gorm:"not null;default:'';size:255"`
	Disabled         int    `gorm:"not null;default:0"`
	GroupID          int64  `gorm:"not null;default:0;index:idx_alert_subscribe_group_id"`
	Prod             string `gorm:"not null;default:'';size:255"`
	Cate             string `gorm:"not null;size:128"`
	DatasourceIDs    string `gorm:"not null;default:'';size:255"`
	Cluster          string `gorm:"not null;size:128"`
	RuleID           int64  `gorm:"not null;default:0"`
	Severities       string `gorm:"not null;default:'';size:32"`
	Tags             string `gorm:"not null;default:'[]';size:4096"`
	RedefineSeverity int    `gorm:"default:0"`
	NewSeverity      int    `gorm:"not null"`
	RedefineChannels int    `gorm:"default:0"`
	NewChannels      string `gorm:"not null;default:'';size:255"`
	UserGroupIDs     string `gorm:"not null;size:250"`
	BusiGroups       string `gorm:"not null;default:'[]';size:4096"`
	Note             string `gorm:"default:'';size:1024"`
	RuleIDs          string `gorm:"default:'';size:1024"`
	Webhooks         string `gorm:"not null;type:text"`
	ExtraConfig      string `gorm:"not null;type:text"`
	RedefineWebhooks int    `gorm:"default:0"`
	ForDuration      int64  `gorm:"not null;default:0"`
	CreateAt         int64  `gorm:"not null;default:0"`
	CreateBy         string `gorm:"not null;default:'';size:64"`
	UpdateAt         int64  `gorm:"not null;default:0;index:idx_alert_subscribe_update_at"`
	UpdateBy         string `gorm:"not null;default:'';size:64"`
}

func (NightingaleAlertSubscribe) TableName() string {
	return "alert_subscribe"
}

// NightingaleMetricView 视图表
type NightingaleMetricView struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"not null;default:'';size:191"`
	Cate     int    `gorm:"not null"`
	Configs  string `gorm:"not null;default:'';size:8192"`
	CreateAt int64  `gorm:"not null;default:0"`
	CreateBy int64  `gorm:"not null;default:0;index:idx_metric_view_create_by"`
	UpdateAt int64  `gorm:"not null;default:0"`
}

func (NightingaleMetricView) TableName() string {
	return "metric_view"
}

// NightingaleRecordingRule 记录规则表
type NightingaleRecordingRule struct {
	ID               uint   `gorm:"primaryKey;autoIncrement"`
	GroupID          int64  `gorm:"not null;default:0;index:idx_recording_rule_group_id"`
	DatasourceIDs    string `gorm:"not null;default:'';size:255"`
	Cluster          string `gorm:"not null;size:128"`
	Name             string `gorm:"not null;size:255"`
	Note             string `gorm:"not null;size:255"`
	Disabled         int    `gorm:"not null;default:0"`
	PromQL           string `gorm:"not null;size:8192"`
	PromEvalInterval int    `gorm:"not null"`
	AppendTags       string `gorm:"default:'';size:255"`
	QueryConfigs     string `gorm:"not null;type:text"`
	CreateAt         int64  `gorm:"default:0"`
	CreateBy         string `gorm:"default:'';size:64"`
	UpdateAt         int64  `gorm:"default:0;index:idx_recording_rule_update_at"`
	UpdateBy         string `gorm:"default:'';size:64"`
}

func (NightingaleRecordingRule) TableName() string {
	return "recording_rule"
}

// NightingaleAlertAggrView 告警聚合视图
type NightingaleAlertAggrView struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"not null;default:'';size:191"`
	Rule     string `gorm:"not null;default:'';size:2048"`
	Cate     int    `gorm:"not null"`
	CreateAt int64  `gorm:"not null;default:0"`
	CreateBy int64  `gorm:"not null;default:0;index:idx_alert_aggr_view_create_by"`
	UpdateAt int64  `gorm:"not null;default:0"`
}

func (NightingaleAlertAggrView) TableName() string {
	return "alert_aggr_view"
}

// NightingaleAlertCurEvent 当前告警事件
type NightingaleAlertCurEvent struct {
	ID               int64  `gorm:"primaryKey;autoIncrement:false"`
	Cate             string `gorm:"not null;size:128"`
	DatasourceID     int64  `gorm:"not null;default:0"`
	Cluster          string `gorm:"not null;size:128"`
	GroupID          int64  `gorm:"not null;index:idx_alert_cur_event_tg_idx"`
	GroupName        string `gorm:"not null;default:'';size:255"`
	Hash             string `gorm:"not null;size:64;index:idx_alert_cur_event_hash_idx"`
	RuleID           int64  `gorm:"not null;index:idx_alert_cur_event_rule_id_idx"`
	RuleName         string `gorm:"not null;size:255"`
	RuleNote         string `gorm:"not null;size:2048"`
	RuleProd         string `gorm:"not null;default:'';size:255"`
	RuleAlgo         string `gorm:"not null;default:'';size:255"`
	Severity         int    `gorm:"not null"`
	PromForDuration  int    `gorm:"not null"`
	PromQL           string `gorm:"not null;size:8192"`
	PromEvalInterval int    `gorm:"not null"`
	Callbacks        string `gorm:"not null;default:'';size:255"`
	RunbookURL       string `gorm:"size:255"`
	NotifyRecovered  int    `gorm:"not null"`
	NotifyChannels   string `gorm:"not null;default:'';size:255"`
	NotifyGroups     string `gorm:"not null;default:'';size:255"`
	NotifyRepeatNext int64  `gorm:"not null;default:0;index:idx_alert_cur_event_nrn_idx"`
	NotifyCurNumber  int    `gorm:"not null;default:0"`
	TargetIdent      string `gorm:"not null;default:'';size:191"`
	TargetNote       string `gorm:"not null;default:'';size:191"`
	FirstTriggerTime int64
	TriggerTime      int64  `gorm:"not null;index:idx_alert_cur_event_tg_idx"`
	TriggerValue     string `gorm:"not null;size:2048"`
	Annotations      string `gorm:"not null;type:text"`
	RuleConfig       string `gorm:"not null;type:text"`
	Tags             string `gorm:"not null;default:'';size:1024"`
}

func (NightingaleAlertCurEvent) TableName() string {
	return "alert_cur_event"
}

// NightingaleAlertHisEvent 历史告警事件
type NightingaleAlertHisEvent struct {
	ID               uint   `gorm:"primaryKey;autoIncrement"`
	IsRecovered      int    `gorm:"not null"`
	Cate             string `gorm:"not null;size:128"`
	DatasourceID     int64  `gorm:"not null;default:0"`
	Cluster          string `gorm:"not null;size:128"`
	GroupID          int64  `gorm:"not null;index:idx_alert_his_event_tg_idx"`
	GroupName        string `gorm:"not null;default:'';size:255"`
	Hash             string `gorm:"not null;size:64;index:idx_alert_his_event_hash_idx"`
	RuleID           int64  `gorm:"not null;index:idx_alert_his_event_rule_id_idx"`
	RuleName         string `gorm:"not null;size:255"`
	RuleNote         string `gorm:"not null;default:'alert rule note';size:2048"`
	RuleProd         string `gorm:"not null;default:'';size:255"`
	RuleAlgo         string `gorm:"not null;default:'';size:255"`
	Severity         int    `gorm:"not null"`
	PromForDuration  int    `gorm:"not null"`
	PromQL           string `gorm:"not null;size:8192"`
	PromEvalInterval int    `gorm:"not null"`
	Callbacks        string `gorm:"not null;default:'';size:255"`
	RunbookURL       string `gorm:"size:255"`
	NotifyRecovered  int    `gorm:"not null"`
	NotifyChannels   string `gorm:"not null;default:'';size:255"`
	NotifyGroups     string `gorm:"not null;default:'';size:255"`
	NotifyCurNumber  int    `gorm:"not null;default:0"`
	TargetIdent      string `gorm:"not null;default:'';size:191"`
	TargetNote       string `gorm:"not null;default:'';size:191"`
	FirstTriggerTime int64
	TriggerTime      int64  `gorm:"not null;index:idx_alert_his_event_tg_idx"`
	TriggerValue     string `gorm:"not null;size:2048"`
	RecoverTime      int64  `gorm:"not null;default:0"`
	LastEvalTime     int64  `gorm:"not null;default:0;index:idx_alert_his_event_nrn_idx"`
	Tags             string `gorm:"not null;default:'';size:1024"`
	Annotations      string `gorm:"not null;type:text"`
	RuleConfig       string `gorm:"not null;type:text"`
}

func (NightingaleAlertHisEvent) TableName() string {
	return "alert_his_event"
}

// NightingaleTaskTpl 任务模板
type NightingaleTaskTpl struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	GroupID   int    `gorm:"not null;index:idx_task_tpl_group_id"`
	Title     string `gorm:"not null;default:'';size:255"`
	Account   string `gorm:"not null;size:64"`
	Batch     int    `gorm:"not null;default:0"`
	Tolerance int    `gorm:"not null;default:0"`
	Timeout   int    `gorm:"not null;default:0"`
	Pause     string `gorm:"not null;default:'';size:255"`
	Script    string `gorm:"not null;type:text"`
	Args      string `gorm:"not null;default:'';size:512"`
	Tags      string `gorm:"not null;default:'';size:255"`
	CreateAt  int64  `gorm:"not null;default:0"`
	CreateBy  string `gorm:"not null;default:'';size:64"`
	UpdateAt  int64  `gorm:"not null;default:0"`
	UpdateBy  string `gorm:"not null;default:'';size:64"`
}

func (NightingaleTaskTpl) TableName() string {
	return "task_tpl"
}

// NightingaleTaskTplHost 任务模板主机
type NightingaleTaskTplHost struct {
	II   uint   `gorm:"primaryKey;autoIncrement"`
	ID   int    `gorm:"not null;index:idx_task_tpl_host_id_host"`
	Host string `gorm:"not null;size:128;index:idx_task_tpl_host_id_host"`
}

func (NightingaleTaskTplHost) TableName() string {
	return "task_tpl_host"
}

// NightingaleTaskRecord 任务记录
type NightingaleTaskRecord struct {
	ID           int64  `gorm:"primaryKey;autoIncrement:false"`
	EventID      int64  `gorm:"not null;default:0;index:idx_task_record_event_id"`
	GroupID      int64  `gorm:"not null;index:idx_task_record_cg_idx"`
	IbexAddress  string `gorm:"not null;size:128"`
	IbexAuthUser string `gorm:"not null;default:'';size:128"`
	IbexAuthPass string `gorm:"not null;default:'';size:128"`
	Title        string `gorm:"not null;default:'';size:255"`
	Account      string `gorm:"not null;size:64"`
	Batch        int    `gorm:"not null;default:0"`
	Tolerance    int    `gorm:"not null;default:0"`
	Timeout      int    `gorm:"not null;default:0"`
	Pause        string `gorm:"not null;default:'';size:255"`
	Script       string `gorm:"not null;type:text"`
	Args         string `gorm:"not null;default:'';size:512"`
	CreateAt     int64  `gorm:"not null;default:0;index:idx_task_record_cg_idx"`
	CreateBy     string `gorm:"not null;default:'';size:64;index:idx_task_record_create_by"`
}

func (NightingaleTaskRecord) TableName() string {
	return "task_record"
}

// NightingaleAlertingEngine 告警引擎
type NightingaleAlertingEngine struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	Instance      string `gorm:"not null;default:'';size:128"`
	DatasourceID  int64  `gorm:"not null;default:0"`
	EngineCluster string `gorm:"not null;default:'';size:128"`
	Clock         int64  `gorm:"not null"`
}

func (NightingaleAlertingEngine) TableName() string {
	return "alerting_engines"
}

// NightingaleDatasource 数据源
type NightingaleDatasource struct {
	ID             uint   `gorm:"primaryKey;autoIncrement"`
	Name           string `gorm:"uniqueIndex;not null;default:'';size:191"`
	Identifier     string `gorm:"not null;default:'';size:255"`
	Description    string `gorm:"not null;default:'';size:255"`
	Category       string `gorm:"not null;default:'';size:255"`
	PluginID       int    `gorm:"not null;default:0"`
	PluginType     string `gorm:"not null;default:'';size:255"`
	PluginTypeName string `gorm:"not null;default:'';size:255"`
	ClusterName    string `gorm:"not null;default:'';size:255"`
	Settings       string `gorm:"not null;type:text"`
	Status         string `gorm:"not null;default:'';size:255"`
	HTTP           string `gorm:"not null;default:'';size:4096"`
	Auth           string `gorm:"not null;default:'';size:8192"`
	IsDefault      bool   `gorm:"not null;default:false"`
	CreatedAt      int64  `gorm:"not null;default:0"`
	CreatedBy      string `gorm:"not null;default:'';size:64"`
	UpdatedAt      int64  `gorm:"not null;default:0"`
	UpdatedBy      string `gorm:"not null;default:'';size:64"`
}

func (NightingaleDatasource) TableName() string {
	return "datasource"
}

// NightingaleBuiltinCate 内置分类
type NightingaleBuiltinCate struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	Name   string `gorm:"not null;size:191"`
	UserID int64  `gorm:"not null;default:0"`
}

func (NightingaleBuiltinCate) TableName() string {
	return "builtin_cate"
}

// NightingaleNotifyTpl 通知模板
type NightingaleNotifyTpl struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Channel  string `gorm:"uniqueIndex;not null;size:32"`
	Name     string `gorm:"not null;size:255"`
	Content  string `gorm:"not null;type:text"`
	CreateAt int64  `gorm:"not null;default:0"`
	CreateBy string `gorm:"not null;default:'';size:64"`
	UpdateAt int64  `gorm:"not null;default:0"`
	UpdateBy string `gorm:"not null;default:'';size:64"`
}

func (NightingaleNotifyTpl) TableName() string {
	return "notify_tpl"
}

// NightingaleSsoConfig SSO配置
type NightingaleSsoConfig struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"uniqueIndex;not null;size:191"`
	Content  string `gorm:"not null;type:text"`
	UpdateAt int64  `gorm:"not null;default:0"`
}

func (NightingaleSsoConfig) TableName() string {
	return "sso_config"
}

// NightingaleEsIndexPattern ES索引模式
type NightingaleEsIndexPattern struct {
	ID                     uint   `gorm:"primaryKey;autoIncrement"`
	DatasourceID           int64  `gorm:"not null;default:0;uniqueIndex:idx_es_index_pattern_ds_name"`
	Name                   string `gorm:"not null;size:191;uniqueIndex:idx_es_index_pattern_ds_name"`
	TimeField              string `gorm:"not null;default:'@timestamp';size:128"`
	AllowHideSystemIndices int    `gorm:"not null;default:0"`
	FieldsFormat           string `gorm:"not null;default:'';size:4096"`
	CrossClusterEnabled    int    `gorm:"not null;default:0"`
	CreateAt               int64  `gorm:"default:0"`
	CreateBy               string `gorm:"default:'';size:64"`
	UpdateAt               int64  `gorm:"default:0"`
	UpdateBy               string `gorm:"default:'';size:64"`
	Note                   string `gorm:"not null;default:'';size:4096"`
}

func (NightingaleEsIndexPattern) TableName() string {
	return "es_index_pattern"
}

// NightingaleBuiltinMetric 内置指标
type NightingaleBuiltinMetric struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	Collector  string `gorm:"not null;size:191;index:idx_collector;uniqueIndex:idx_builtin_metric_unique"`
	Typ        string `gorm:"not null;size:191;index:idx_typ;uniqueIndex:idx_builtin_metric_unique"`
	Name       string `gorm:"not null;size:191;index:idx_name;uniqueIndex:idx_builtin_metric_unique"`
	Unit       string `gorm:"not null;size:191"`
	Lang       string `gorm:"not null;default:'';size:191;index:idx_lang;uniqueIndex:idx_builtin_metric_unique"`
	Note       string `gorm:"not null;size:4096"`
	Expression string `gorm:"not null;size:4096"`
	CreatedAt  int64  `gorm:"not null;default:0"`
	CreatedBy  string `gorm:"not null;default:'';size:191"`
	UpdatedAt  int64  `gorm:"not null;default:0"`
	UpdatedBy  string `gorm:"not null;default:'';size:191"`
	UUID       int64  `gorm:"not null;default:0"`
}

func (NightingaleBuiltinMetric) TableName() string {
	return "builtin_metrics"
}

// NightingaleMetricFilter 指标过滤
type NightingaleMetricFilter struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	Name       string `gorm:"not null;size:191;index:idx_metric_filter_name"`
	Configs    string `gorm:"not null;size:4096"`
	GroupsPerm string `gorm:"type:text"`
	CreateAt   int64  `gorm:"not null;default:0"`
	CreateBy   string `gorm:"not null;default:'';size:191"`
	UpdateAt   int64  `gorm:"not null;default:0"`
	UpdateBy   string `gorm:"not null;default:'';size:191"`
}

func (NightingaleMetricFilter) TableName() string {
	return "metric_filter"
}

// NightingaleBoardBusiGroup 仪表盘业务组关联
type NightingaleBoardBusiGroup struct {
	BusiGroupID int64 `gorm:"primaryKey;not null;default:0"`
	BoardID     int64 `gorm:"primaryKey;not null;default:0"`
}

func (NightingaleBoardBusiGroup) TableName() string {
	return "board_busigroup"
}

// NightingaleBuiltinComponent 内置组件
type NightingaleBuiltinComponent struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Ident     string `gorm:"not null;size:191;index:idx_ident"`
	Logo      string `gorm:"not null;size:191"`
	Readme    string `gorm:"not null;type:text"`
	Disabled  int    `gorm:"not null;default:0"`
	CreatedAt int64  `gorm:"not null;default:0"`
	CreatedBy string `gorm:"not null;default:'';size:191"`
	UpdatedAt int64  `gorm:"not null;default:0"`
	UpdatedBy string `gorm:"not null;default:'';size:191"`
}

func (NightingaleBuiltinComponent) TableName() string {
	return "builtin_components"
}

// NightingaleBuiltinPayload 内置Payload
type NightingaleBuiltinPayload struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Type      string `gorm:"not null;size:191;index:idx_type"`
	UUID      int64  `gorm:"not null;default:0"`
	Component string `gorm:"not null;size:191;index:idx_component"`
	Cate      string `gorm:"not null;size:191;index:idx_cate"`
	Name      string `gorm:"not null;size:191;index:idx_builtin_payloads_name"`
	Tags      string `gorm:"not null;default:'';size:191"`
	Content   string `gorm:"not null;type:text"`
	Note      string `gorm:"not null;default:'';size:1024"`
	CreatedAt int64  `gorm:"not null;default:0"`
	CreatedBy string `gorm:"not null;default:'';size:191"`
	UpdatedAt int64  `gorm:"not null;default:0"`
	UpdatedBy string `gorm:"not null;default:'';size:191"`
}

func (NightingaleBuiltinPayload) TableName() string {
	return "builtin_payloads"
}

// NightingaleDashAnnotation 仪表盘注解
type NightingaleDashAnnotation struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	DashboardID int64  `gorm:"not null"`
	PanelID     string `gorm:"not null;size:191"`
	Tags        string `gorm:"type:text"`
	Description string `gorm:"type:text"`
	Config      string `gorm:"type:text"`
	TimeStart   int64  `gorm:"not null;default:0"`
	TimeEnd     int64  `gorm:"not null;default:0"`
	CreateAt    int64  `gorm:"not null;default:0"`
	CreateBy    string `gorm:"not null;default:'';size:64"`
	UpdateAt    int64  `gorm:"not null;default:0"`
	UpdateBy    string `gorm:"not null;default:'';size:64"`
}

func (NightingaleDashAnnotation) TableName() string {
	return "dash_annotation"
}

// NightingaleSourceToken 来源Token
type NightingaleSourceToken struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	SourceType string `gorm:"not null;default:'';size:64;index:idx_source_token_type_id_token"`
	SourceID   string `gorm:"not null;default:'';size:255;index:idx_source_token_type_id_token"`
	Token      string `gorm:"not null;default:'';size:255;index:idx_source_token_type_id_token"`
	ExpireAt   int64  `gorm:"not null;default:0"`
	CreateAt   int64  `gorm:"not null;default:0"`
	CreateBy   string `gorm:"not null;default:'';size:64"`
}

func (NightingaleSourceToken) TableName() string {
	return "source_token"
}

// NightingaleNotificationRecord 通知记录
type NightingaleNotificationRecord struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	NotifyRuleID int64  `gorm:"not null;default:0"`
	EventID      int64  `gorm:"not null;index:idx_evt"`
	SubID        *int64 `gorm:"default:null"`
	Channel      string `gorm:"not null;size:255"`
	Status       *int64 `gorm:"default:null"`
	Target       string `gorm:"not null;size:1024"`
	Details      string `gorm:"default:'';size:2048"`
	CreatedAt    int64  `gorm:"not null"`
}

func (NightingaleNotificationRecord) TableName() string {
	return "notification_record"
}

// NightingaleTargetBusiGroup 目标业务组关联
type NightingaleTargetBusiGroup struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	TargetIdent string `gorm:"not null;size:191;uniqueIndex:idx_target_group"`
	GroupID     int64  `gorm:"not null;uniqueIndex:idx_target_group"`
	UpdateAt    int64  `gorm:"not null"`
}

func (NightingaleTargetBusiGroup) TableName() string {
	return "target_busi_group"
}

// NightingaleUserToken 用户Token
type NightingaleUserToken struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Username  string `gorm:"not null;default:'';size:255"`
	TokenName string `gorm:"not null;default:'';size:255"`
	Token     string `gorm:"not null;default:'';size:255"`
	CreateAt  int64  `gorm:"not null;default:0"`
	LastUsed  int64  `gorm:"not null;default:0"`
}

func (NightingaleUserToken) TableName() string {
	return "user_token"
}

// NightingaleNotifyRule 通知规则
type NightingaleNotifyRule struct {
	ID              uint   `gorm:"primaryKey;autoIncrement"`
	Name            string `gorm:"not null;size:255"`
	Description     string `gorm:"type:text"`
	Enable          bool   `gorm:"default:false"`
	UserGroupIDs    string `gorm:"not null;default:'';size:255"`
	NotifyConfigs   string `gorm:"type:text"`
	PipelineConfigs string `gorm:"type:text"`
	CreateAt        int64  `gorm:"not null;default:0"`
	CreateBy        string `gorm:"not null;default:'';size:64"`
	UpdateAt        int64  `gorm:"not null;default:0"`
	UpdateBy        string `gorm:"not null;default:'';size:64"`
}

func (NightingaleNotifyRule) TableName() string {
	return "notify_rule"
}

// NightingaleNotifyChannel 通知渠道
type NightingaleNotifyChannel struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	Name          string `gorm:"not null;size:255"`
	Ident         string `gorm:"not null;size:255"`
	Description   string `gorm:"type:text"`
	Enable        bool   `gorm:"default:false"`
	ParamConfig   string `gorm:"type:text"`
	RequestType   string `gorm:"not null;size:50"`
	RequestConfig string `gorm:"type:text"`
	Weight        int    `gorm:"not null;default:0"`
	CreateAt      int64  `gorm:"not null;default:0"`
	CreateBy      string `gorm:"not null;default:'';size:64"`
	UpdateAt      int64  `gorm:"not null;default:0"`
	UpdateBy      string `gorm:"not null;default:'';size:64"`
}

func (NightingaleNotifyChannel) TableName() string {
	return "notify_channel"
}

// NightingaleMessageTemplate 消息模板
type NightingaleMessageTemplate struct {
	ID                 uint   `gorm:"primaryKey;autoIncrement"`
	Name               string `gorm:"not null;size:64"`
	Ident              string `gorm:"not null;size:64"`
	Content            string `gorm:"type:text"`
	UserGroupIDs       string `gorm:"size:64"`
	NotifyChannelIdent string `gorm:"not null;default:'';size:64"`
	Private            int    `gorm:"not null;default:0"`
	Weight             int    `gorm:"not null;default:0"`
	CreateAt           int64  `gorm:"not null;default:0"`
	CreateBy           string `gorm:"not null;default:'';size:64"`
	UpdateAt           int64  `gorm:"not null;default:0"`
	UpdateBy           string `gorm:"not null;default:'';size:64"`
}

func (NightingaleMessageTemplate) TableName() string {
	return "message_template"
}

// NightingaleEventPipeline 事件处理管道
type NightingaleEventPipeline struct {
	ID               uint   `gorm:"primaryKey;autoIncrement"`
	Name             string `gorm:"not null;size:128"`
	TeamIDs          string `gorm:"type:text"`
	Description      string `gorm:"not null;default:'';size:255"`
	FilterEnable     int    `gorm:"not null;default:0"`
	LabelFilters     string `gorm:"type:text"`
	AttributeFilters string `gorm:"type:text"`
	Processors       string `gorm:"type:text"`
	CreateAt         int64  `gorm:"not null;default:0"`
	CreateBy         string `gorm:"not null;default:'';size:64"`
	UpdateAt         int64  `gorm:"not null;default:0"`
	UpdateBy         string `gorm:"not null;default:'';size:64"`
}

func (NightingaleEventPipeline) TableName() string {
	return "event_pipeline"
}

// NightingaleEmbeddedProduct 嵌入式产品
type NightingaleEmbeddedProduct struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:255"`
	URL       string `gorm:"size:255"`
	IsPrivate bool   `gorm:"default:null"`
	TeamIDs   string `gorm:"size:255"`
	CreateAt  int64  `gorm:"not null;default:0"`
	CreateBy  string `gorm:"not null;default:'';size:64"`
	UpdateAt  int64  `gorm:"not null;default:0"`
	UpdateBy  string `gorm:"not null;default:'';size:64"`
}

func (NightingaleEmbeddedProduct) TableName() string {
	return "embedded_product"
}
