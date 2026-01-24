package models

import (
	"time"

	"gorm.io/gorm"
)

// IPBlacklist IP黑名单配置
type IPBlacklist struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	IP          string         `gorm:"size:64;uniqueIndex;not null" json:"ip"`        // IP地址或CIDR格式
	Reason      string         `gorm:"size:256" json:"reason"`                        // 封禁原因
	BlockType   string         `gorm:"size:32;default:'permanent'" json:"block_type"` // permanent永久, temporary临时
	ExpireAt    *time.Time     `json:"expire_at,omitempty"`                           // 临时封禁的过期时间
	CreatedBy   string         `gorm:"size:64" json:"created_by"`                     // 创建人
	HitCount    int            `gorm:"default:0" json:"hit_count"`                    // 命中次数
	LastHitAt   *time.Time     `json:"last_hit_at,omitempty"`                         // 最后命中时间
	Enabled     bool           `gorm:"default:true" json:"enabled"`                   // 是否启用
	Description string         `gorm:"type:text" json:"description,omitempty"`        // 详细描述
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (IPBlacklist) TableName() string {
	return "ip_blacklists"
}

// IPWhitelist IP白名单配置（优先级高于黑名单）
type IPWhitelist struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	IP          string         `gorm:"size:64;uniqueIndex;not null" json:"ip"` // IP地址或CIDR格式
	Description string         `gorm:"size:256" json:"description"`            // 描述
	CreatedBy   string         `gorm:"size:64" json:"created_by"`              // 创建人
	Enabled     bool           `gorm:"default:true" json:"enabled"`            // 是否启用
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (IPWhitelist) TableName() string {
	return "ip_whitelists"
}

// LoginAttempt 登录尝试记录（用于检测暴力破解和统计）
type LoginAttempt struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	IP          string    `gorm:"size:64;index;not null" json:"ip"`
	Username    string    `gorm:"size:64;index" json:"username"`
	UserID      uint      `gorm:"index" json:"user_id,omitempty"` // 关联用户ID（如果用户存在）
	Success     bool      `gorm:"default:false;index" json:"success"`
	UserAgent   string    `gorm:"size:512" json:"user_agent,omitempty"`
	Reason      string    `gorm:"size:256" json:"reason,omitempty"`            // 失败原因
	FailureType string    `gorm:"size:32;index" json:"failure_type,omitempty"` // 失败类型: invalid_password, account_locked, ip_blocked, user_not_found
	Country     string    `gorm:"size:64" json:"country,omitempty"`            // IP 所属国家（可选，需要 GeoIP）
	City        string    `gorm:"size:64" json:"city,omitempty"`               // IP 所属城市（可选）
	RequestID   string    `gorm:"size:64" json:"request_id,omitempty"`         // 请求追踪ID
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (LoginAttempt) TableName() string {
	return "login_attempts"
}

// LoginFailureType 登录失败类型常量
const (
	LoginFailureInvalidPassword = "invalid_password" // 密码错误
	LoginFailureAccountLocked   = "account_locked"   // 账号已锁定
	LoginFailureIPBlocked       = "ip_blocked"       // IP 已封禁
	LoginFailureUserNotFound    = "user_not_found"   // 用户不存在
	LoginFailureAccountDisabled = "account_disabled" // 账号已禁用
	LoginFailure2FAFailed       = "2fa_failed"       // 二次认证失败
	LoginFailureLDAPError       = "ldap_error"       // LDAP 认证错误
)

// IPLoginStats IP 登录统计（用于快速查询）
type IPLoginStats struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	IP               string     `gorm:"size:64;uniqueIndex;not null" json:"ip"`
	TotalAttempts    int        `gorm:"default:0" json:"total_attempts"`    // 总尝试次数
	SuccessCount     int        `gorm:"default:0" json:"success_count"`     // 成功次数
	FailureCount     int        `gorm:"default:0" json:"failure_count"`     // 失败次数
	ConsecutiveFails int        `gorm:"default:0" json:"consecutive_fails"` // 连续失败次数
	LastAttemptAt    *time.Time `json:"last_attempt_at,omitempty"`          // 最后尝试时间
	LastSuccessAt    *time.Time `json:"last_success_at,omitempty"`          // 最后成功时间
	LastFailureAt    *time.Time `json:"last_failure_at,omitempty"`          // 最后失败时间
	BlockedUntil     *time.Time `json:"blocked_until,omitempty"`            // 封禁到期时间
	BlockCount       int        `gorm:"default:0" json:"block_count"`       // 被封禁次数
	FirstSeenAt      time.Time  `json:"first_seen_at"`                      // 首次出现时间
	UniqueUsernames  int        `gorm:"default:0" json:"unique_usernames"`  // 尝试的不同用户名数量
	Country          string     `gorm:"size:64" json:"country,omitempty"`
	City             string     `gorm:"size:64" json:"city,omitempty"`
	RiskScore        int        `gorm:"default:0" json:"risk_score"` // 风险评分 (0-100)
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (IPLoginStats) TableName() string {
	return "ip_login_stats"
}

// IsBlocked 检查 IP 是否被封禁
func (s *IPLoginStats) IsBlocked() bool {
	if s.BlockedUntil == nil {
		return false
	}
	return time.Now().Before(*s.BlockedUntil)
}

// TwoFactorConfig 二次认证配置
type TwoFactorConfig struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"uniqueIndex;not null" json:"user_id"`  // 关联用户ID
	Enabled           bool           `gorm:"default:false" json:"enabled"`         // 是否启用2FA
	Type              string         `gorm:"size:32;default:'totp'" json:"type"`   // totp/sms/email
	Secret            string         `gorm:"size:256" json:"secret,omitempty"`     // TOTP密钥（加密存储）
	Phone             string         `gorm:"size:32" json:"phone,omitempty"`       // 短信验证手机号
	Email             string         `gorm:"size:128" json:"email,omitempty"`      // 邮箱验证地址
	RecoveryCodes     string         `gorm:"type:text" json:"-"`                   // 恢复码（JSON数组，加密存储）
	RecoveryUsedCount int            `gorm:"default:0" json:"recovery_used_count"` // 已使用的恢复码数量
	LastVerifiedAt    *time.Time     `json:"last_verified_at,omitempty"`           // 最后验证时间
	VerifyCount       int            `gorm:"default:0" json:"verify_count"`        // 验证次数
	FailedCount       int            `gorm:"default:0" json:"failed_count"`        // 连续失败次数
	LockedUntil       *time.Time     `json:"locked_until,omitempty"`               // 锁定到期时间
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (TwoFactorConfig) TableName() string {
	return "two_factor_configs"
}

// TwoFactorGlobalConfig 全局二次认证配置
type TwoFactorGlobalConfig struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Enabled           bool           `gorm:"default:false" json:"enabled"`                         // 是否全局启用2FA
	EnforceForAdmin   bool           `gorm:"default:true" json:"enforce_for_admin"`                // 是否强制管理员启用
	EnforceForAll     bool           `gorm:"default:false" json:"enforce_for_all"`                 // 是否强制所有用户启用
	AllowedTypes      string         `gorm:"size:128;default:'totp'" json:"allowed_types"`         // 允许的2FA类型，逗号分隔
	TOTPIssuer        string         `gorm:"size:64;default:'AI-Infra-Matrix'" json:"totp_issuer"` // TOTP发行者名称
	TOTPDigits        int            `gorm:"default:6" json:"totp_digits"`                         // TOTP码位数
	TOTPPeriod        int            `gorm:"default:30" json:"totp_period"`                        // TOTP码有效期（秒）
	RecoveryCodeCount int            `gorm:"default:10" json:"recovery_code_count"`                // 恢复码数量
	MaxFailedAttempts int            `gorm:"default:5" json:"max_failed_attempts"`                 // 最大失败尝试次数
	LockoutDuration   int            `gorm:"default:300" json:"lockout_duration"`                  // 锁定时长（秒）
	SMSEnabled        bool           `gorm:"default:false" json:"sms_enabled"`                     // 是否启用短信验证
	SMSProvider       string         `gorm:"size:32" json:"sms_provider,omitempty"`                // 短信提供商
	SMSConfig         string         `gorm:"type:text" json:"-"`                                   // 短信配置（JSON，加密存储）
	EmailEnabled      bool           `gorm:"default:false" json:"email_enabled"`                   // 是否启用邮箱验证
	EmailConfig       string         `gorm:"type:text" json:"-"`                                   // 邮箱配置（JSON，加密存储）
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (TwoFactorGlobalConfig) TableName() string {
	return "two_factor_global_configs"
}

// OAuthProvider OAuth/第三方登录提供商配置
type OAuthProvider struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"size:32;uniqueIndex;not null" json:"name"`   // 提供商名称：google, github, wechat, dingtalk, feishu, teams
	DisplayName  string         `gorm:"size:64" json:"display_name"`                // 显示名称
	Type         string         `gorm:"size:32;default:'oauth2'" json:"type"`       // oauth2, oidc, saml
	Enabled      bool           `gorm:"default:false" json:"enabled"`               // 是否启用
	ClientID     string         `gorm:"size:256" json:"client_id,omitempty"`        // 客户端ID（加密存储）
	ClientSecret string         `gorm:"size:512" json:"-"`                          // 客户端密钥（加密存储）
	AuthURL      string         `gorm:"size:512" json:"auth_url,omitempty"`         // 认证URL
	TokenURL     string         `gorm:"size:512" json:"token_url,omitempty"`        // Token URL
	UserInfoURL  string         `gorm:"size:512" json:"user_info_url,omitempty"`    // 用户信息URL
	RedirectURL  string         `gorm:"size:512" json:"redirect_url,omitempty"`     // 回调URL
	Scopes       string         `gorm:"size:256" json:"scopes,omitempty"`           // 权限范围
	IconURL      string         `gorm:"size:256" json:"icon_url,omitempty"`         // 图标URL
	SortOrder    int            `gorm:"default:0" json:"sort_order"`                // 排序顺序
	AutoRegister bool           `gorm:"default:true" json:"auto_register"`          // 是否自动注册新用户
	DefaultRole  string         `gorm:"size:32;default:'user'" json:"default_role"` // 自动注册时的默认角色
	ExtraConfig  string         `gorm:"type:text" json:"-"`                         // 额外配置（JSON，加密存储）
	Description  string         `gorm:"type:text" json:"description,omitempty"`     // 描述
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (OAuthProvider) TableName() string {
	return "oauth_providers"
}

// UserOAuthBinding 用户OAuth绑定关系
type UserOAuthBinding struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`               // 本系统用户ID
	ProviderName string         `gorm:"size:32;index;not null" json:"provider_name"` // 提供商名称
	ProviderUID  string         `gorm:"size:128;index;not null" json:"provider_uid"` // 第三方用户ID
	UnionID      string         `gorm:"size:128;index" json:"union_id,omitempty"`    // 统一ID（微信等）
	Nickname     string         `gorm:"size:64" json:"nickname,omitempty"`           // 第三方昵称
	Avatar       string         `gorm:"size:256" json:"avatar,omitempty"`            // 第三方头像
	Email        string         `gorm:"size:128" json:"email,omitempty"`             // 第三方邮箱
	Phone        string         `gorm:"size:32" json:"phone,omitempty"`              // 第三方手机号
	AccessToken  string         `gorm:"size:1024" json:"-"`                          // 访问令牌（加密存储）
	RefreshToken string         `gorm:"size:1024" json:"-"`                          // 刷新令牌（加密存储）
	TokenExpiry  *time.Time     `json:"token_expiry,omitempty"`                      // 令牌过期时间
	RawData      string         `gorm:"type:text" json:"-"`                          // 原始用户数据（JSON）
	LastLoginAt  *time.Time     `json:"last_login_at,omitempty"`                     // 最后登录时间
	LoginCount   int            `gorm:"default:0" json:"login_count"`                // 登录次数
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (UserOAuthBinding) TableName() string {
	return "user_oauth_bindings"
}

// 创建唯一约束
func (UserOAuthBinding) BeforeCreate(tx *gorm.DB) error {
	// provider_name + provider_uid 唯一
	return nil
}

// SecurityAuditLog 安全审计日志
type SecurityAuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id,omitempty"`
	Username   string    `gorm:"size:64;index" json:"username,omitempty"`
	Action     string    `gorm:"size:64;index;not null" json:"action"` // login, logout, 2fa_enable, 2fa_disable, password_change, ip_block, etc.
	Resource   string    `gorm:"size:64" json:"resource,omitempty"`    // 操作的资源类型
	ResourceID string    `gorm:"size:64" json:"resource_id,omitempty"` // 资源ID
	IP         string    `gorm:"size:64;index" json:"ip"`
	UserAgent  string    `gorm:"size:256" json:"user_agent,omitempty"`
	Status     string    `gorm:"size:32;default:'success'" json:"status"` // success, failed
	Details    string    `gorm:"type:text" json:"details,omitempty"`      // 详细信息（JSON）
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (SecurityAuditLog) TableName() string {
	return "security_audit_logs"
}

// SecurityConfig 安全配置（全局）
type SecurityConfig struct {
	ID                       uint           `gorm:"primaryKey" json:"id"`
	IPBlacklistEnabled       bool           `gorm:"default:true" json:"ip_blacklist_enabled"`       // 是否启用IP黑名单
	IPWhitelistEnabled       bool           `gorm:"default:false" json:"ip_whitelist_enabled"`      // 是否启用IP白名单（启用后只允许白名单IP访问）
	MaxLoginAttempts         int            `gorm:"default:5" json:"max_login_attempts"`            // 最大登录尝试次数
	LoginLockoutDuration     int            `gorm:"default:900" json:"login_lockout_duration"`      // 登录锁定时长（秒）
	AutoBlockEnabled         bool           `gorm:"default:true" json:"auto_block_enabled"`         // 是否自动封禁恶意IP
	AutoBlockThreshold       int            `gorm:"default:10" json:"auto_block_threshold"`         // 自动封禁阈值（失败次数）
	AutoBlockDuration        int            `gorm:"default:3600" json:"auto_block_duration"`        // 自动封禁时长（秒）
	SessionTimeout           int            `gorm:"default:3600" json:"session_timeout"`            // 会话超时时间（秒）
	MaxConcurrentSessions    int            `gorm:"default:5" json:"max_concurrent_sessions"`       // 最大并发会话数
	PasswordMinLength        int            `gorm:"default:8" json:"password_min_length"`           // 密码最小长度
	PasswordRequireUppercase bool           `gorm:"default:true" json:"password_require_uppercase"` // 密码需要大写字母
	PasswordRequireLowercase bool           `gorm:"default:true" json:"password_require_lowercase"` // 密码需要小写字母
	PasswordRequireNumber    bool           `gorm:"default:true" json:"password_require_number"`    // 密码需要数字
	PasswordRequireSpecial   bool           `gorm:"default:false" json:"password_require_special"`  // 密码需要特殊字符
	PasswordExpireDays       int            `gorm:"default:0" json:"password_expire_days"`          // 密码过期天数（0表示不过期）
	AuditLogEnabled          bool           `gorm:"default:true" json:"audit_log_enabled"`          // 是否启用审计日志
	AuditLogRetentionDays    int            `gorm:"default:90" json:"audit_log_retention_days"`     // 审计日志保留天数
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (SecurityConfig) TableName() string {
	return "security_configs"
}

// === 预定义的 OAuth 提供商配置 ===

// GetDefaultOAuthProviders 获取预定义的OAuth提供商配置模板
func GetDefaultOAuthProviders() []OAuthProvider {
	return []OAuthProvider{
		{
			Name:         "google",
			DisplayName:  "Google",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v3/userinfo",
			Scopes:       "openid,email,profile",
			IconURL:      "/icons/google.svg",
			SortOrder:    1,
			AutoRegister: true,
			DefaultRole:  "user",
		},
		{
			Name:         "github",
			DisplayName:  "GitHub",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       "user:email,read:user",
			IconURL:      "/icons/github.svg",
			SortOrder:    2,
			AutoRegister: true,
			DefaultRole:  "user",
		},
		{
			Name:         "wechat",
			DisplayName:  "微信",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://open.weixin.qq.com/connect/qrconnect",
			TokenURL:     "https://api.weixin.qq.com/sns/oauth2/access_token",
			UserInfoURL:  "https://api.weixin.qq.com/sns/userinfo",
			Scopes:       "snsapi_login",
			IconURL:      "/icons/wechat.svg",
			SortOrder:    3,
			AutoRegister: true,
			DefaultRole:  "user",
			Description:  "微信开放平台登录（网站应用）",
		},
		{
			Name:         "wechat_work",
			DisplayName:  "企业微信",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://open.work.weixin.qq.com/wwopen/sso/qrConnect",
			TokenURL:     "https://qyapi.weixin.qq.com/cgi-bin/gettoken",
			UserInfoURL:  "https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo",
			Scopes:       "snsapi_base",
			IconURL:      "/icons/wechat-work.svg",
			SortOrder:    4,
			AutoRegister: true,
			DefaultRole:  "user",
			Description:  "企业微信扫码登录",
		},
		{
			Name:         "dingtalk",
			DisplayName:  "钉钉",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://login.dingtalk.com/oauth2/auth",
			TokenURL:     "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
			UserInfoURL:  "https://api.dingtalk.com/v1.0/contact/users/me",
			Scopes:       "openid,corpid",
			IconURL:      "/icons/dingtalk.svg",
			SortOrder:    5,
			AutoRegister: true,
			DefaultRole:  "user",
			Description:  "钉钉扫码登录",
		},
		{
			Name:         "feishu",
			DisplayName:  "飞书",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://passport.feishu.cn/suite/passport/oauth/authorize",
			TokenURL:     "https://passport.feishu.cn/suite/passport/oauth/token",
			UserInfoURL:  "https://passport.feishu.cn/suite/passport/oauth/userinfo",
			Scopes:       "contact:user.email:readonly,contact:user.phone:readonly",
			IconURL:      "/icons/feishu.svg",
			SortOrder:    6,
			AutoRegister: true,
			DefaultRole:  "user",
			Description:  "飞书扫码登录",
		},
		{
			Name:         "teams",
			DisplayName:  "Microsoft Teams",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			UserInfoURL:  "https://graph.microsoft.com/v1.0/me",
			Scopes:       "openid,email,profile,User.Read",
			IconURL:      "/icons/teams.svg",
			SortOrder:    7,
			AutoRegister: true,
			DefaultRole:  "user",
			Description:  "Microsoft Teams / Azure AD 登录",
		},
		{
			Name:         "gitlab",
			DisplayName:  "GitLab",
			Type:         "oauth2",
			Enabled:      false,
			AuthURL:      "https://gitlab.com/oauth/authorize",
			TokenURL:     "https://gitlab.com/oauth/token",
			UserInfoURL:  "https://gitlab.com/api/v4/user",
			Scopes:       "read_user,openid,email",
			IconURL:      "/icons/gitlab.svg",
			SortOrder:    8,
			AutoRegister: true,
			DefaultRole:  "user",
			Description:  "GitLab OAuth 登录（可配置私有实例）",
		},
	}
}
