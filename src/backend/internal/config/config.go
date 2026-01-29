package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Port string

	// 数据库配置
	Database DatabaseConfig

	// OceanBase 可选配置（MySQL 协议）
	OceanBase OceanBaseConfig

	// Redis配置
	Redis RedisConfig

	// JWT配置
	JWTSecret string

	// 加密配置
	EncryptionKey string

	// LDAP配置
	LDAP LDAPConfig

	// LDAP初始化配置
	LDAPInit LDAPInitConfig
	// LDAP后台同步（可选）
	LDAPSync LDAPSyncRuntime

	// Gitea 集成
	Gitea GiteaConfig
	// Gitea 后台同步
	GiteaSync GiteaSyncRuntime

	// AppHub 包仓库配置
	AppHub AppHubConfig

	// Keycloak IAM 配置
	Keycloak KeycloakConfig

	// Nightingale 监控配置
	Nightingale NightingaleConfig

	// ArgoCD GitOps 配置
	ArgoCD ArgoCDConfig

	// 日志级别 (trace, debug, info, warn, error, fatal, panic)
	LogLevel string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// OceanBaseConfig 兼容 MySQL 协议
type OceanBaseConfig struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	// 额外参数，例如 charset、parseTime
	Params string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type LDAPConfig struct {
	Enabled      bool   `json:"enabled"`
	Server       string `json:"server"`
	Port         int    `json:"port"`
	UseSSL       bool   `json:"use_ssl"`
	StartTLS     bool   `json:"start_tls"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
	BaseDN       string `json:"base_dn"`
	UserFilter   string `json:"user_filter"`
	GroupFilter  string `json:"group_filter"`
	UserIDAttr   string `json:"user_id_attr"`
	UserNameAttr string `json:"user_name_attr"`
	EmailAttr    string `json:"email_attr"`
	AdminGroups  string `json:"admin_groups"`
}

type LDAPInitConfig struct {
	// 初始化控制
	InitLDAP      bool `json:"init_ldap"`
	RetryCount    int  `json:"retry_count"`
	RetryInterval int  `json:"retry_interval"`

	// 组织单位配置
	PeopleOU string `json:"people_ou"`
	GroupsOU string `json:"groups_ou"`

	// 管理员用户配置
	AdminUser AdminUserConfig `json:"admin_user"`

	// 普通用户配置
	RegularUser RegularUserConfig `json:"regular_user"`

	// 组配置
	AdminGroupCN string `json:"admin_group_cn"`
	UserGroupCN  string `json:"user_group_cn"`
}

// LDAPSyncRuntime 运行时同步配置
type LDAPSyncRuntime struct {
	Enabled bool `json:"enabled"`
	// 同步间隔，单位秒（默认900秒=15分钟）
	IntervalSeconds int `json:"interval_seconds"`
}

// GiteaConfig Gitea 集成配置
type GiteaConfig struct {
	Enabled    bool   `json:"enabled"`
	BaseURL    string `json:"base_url"`
	AdminToken string `json:"admin_token"`
	AutoCreate bool   `json:"auto_create"`
	AutoUpdate bool   `json:"auto_update"`
	// AliasAdminTo maps the reserved backend username "admin" to a concrete Gitea account name.
	// Gitea reserves the name "admin", so provisioning that username will fail with 422.
	// When set (non-empty), any "admin" user will be provisioned/updated as this target username.
	// Example: "test" (an existing Gitea admin user).
	AliasAdminTo string `json:"alias_admin_to"`
}

// GiteaSyncRuntime Gitea 同步配置
type GiteaSyncRuntime struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"interval_seconds"`
}

// AppHubConfig AppHub 包仓库配置
type AppHubConfig struct {
	Host   string `json:"host"`   // AppHub 主机地址，默认使用 EXTERNAL_HOST
	Port   string `json:"port"`   // AppHub 端口，默认 28080
	Scheme string `json:"scheme"` // 协议，默认 http
}

// KeycloakConfig Keycloak IAM 配置
type KeycloakConfig struct {
	Enabled      bool   `json:"enabled"`
	BaseURL      string `json:"base_url"`       // Keycloak 服务地址，如 http://keycloak:8080/auth
	Realm        string `json:"realm"`          // Realm 名称，默认 ai-infra
	ClientID     string `json:"client_id"`      // 客户端 ID
	ClientSecret string `json:"client_secret"`  // 客户端密钥
	AdminUser    string `json:"admin_user"`     // 管理员用户名
	AdminPass    string `json:"admin_password"` // 管理员密码
	// SSO 配置
	SSOEnabled     bool   `json:"sso_enabled"`      // 是否启用 SSO 登录
	SSORedirectURL string `json:"sso_redirect_url"` // SSO 回调地址
}

// NightingaleConfig Nightingale 监控配置
type NightingaleConfig struct {
	Enabled     bool   `json:"enabled"`
	BaseURL     string `json:"base_url"`     // Nightingale 服务地址，如 http://nightingale:17000
	APIUsername string `json:"api_username"` // API 认证用户名
	APIPassword string `json:"api_password"` // API 认证密码 (MD5 哈希)
	// 用户同步配置
	UserSyncEnabled         bool `json:"user_sync_enabled"`
	UserSyncIntervalSeconds int  `json:"user_sync_interval_seconds"`
}

// ArgoCDConfig ArgoCD GitOps 配置
type ArgoCDConfig struct {
	Enabled    bool   `json:"enabled"`
	BaseURL    string `json:"base_url"`    // ArgoCD 服务地址
	AdminToken string `json:"admin_token"` // ArgoCD API Token
	// Gitea 集成配置
	GiteaRepoURL string `json:"gitea_repo_url"` // Gitea 仓库 URL
}

// GetAppHubURL 获取 AppHub 完整 URL
func (c *AppHubConfig) GetAppHubURL() string {
	return c.Scheme + "://" + c.Host + ":" + c.Port
}

type AdminUserConfig struct {
	UID           string `json:"uid"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	CN            string `json:"cn"`
	SN            string `json:"sn"`
	GivenName     string `json:"given_name"`
	UIDNumber     int    `json:"uid_number"`
	GIDNumber     int    `json:"gid_number"`
	HomeDirectory string `json:"home_directory"`
}

type RegularUserConfig struct {
	UID           string `json:"uid"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	CN            string `json:"cn"`
	SN            string `json:"sn"`
	GivenName     string `json:"given_name"`
	UIDNumber     int    `json:"uid_number"`
	GIDNumber     int    `json:"gid_number"`
	HomeDirectory string `json:"home_directory"`
}

func Load() (*Config, error) {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	obPort, _ := strconv.Atoi(getEnv("OB_PORT", "2881"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	config := &Config{
		Port: getEnv("PORT", "8082"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "ai_infra_matrix"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     redisPort,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		OceanBase: OceanBaseConfig{
			Enabled:  getEnv("OB_ENABLED", "false") == "true",
			Host:     getEnv("OB_HOST", "oceanbase"),
			Port:     obPort,
			User:     getEnv("OB_USER", "root@sys"),
			Password: getEnv("OB_PASSWORD", ""),
			DBName:   getEnv("OB_DB", "aimatrix"),
			Params:   getEnv("OB_PARAMS", "charset=utf8mb4&parseTime=True&loc=Local"),
		},
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "your-encryption-key-change-in-production-32-bytes"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LDAP: LDAPConfig{
			Enabled:      getEnv("LDAP_ENABLED", "false") == "true",
			Server:       getEnv("LDAP_SERVER", "localhost"),
			Port:         getEnvAsInt("LDAP_PORT", 389),
			UseSSL:       getEnv("LDAP_USE_SSL", "false") == "true",
			StartTLS:     getEnv("LDAP_START_TLS", "false") == "true",
			BindDN:       getEnv("LDAP_BIND_DN", ""),
			BindPassword: getEnv("LDAP_BIND_PASSWORD", ""),
			BaseDN:       getEnv("LDAP_BASE_DN", ""),
			UserFilter:   getEnv("LDAP_USER_FILTER", "(sAMAccountName=%s)"),
			GroupFilter:  getEnv("LDAP_GROUP_FILTER", "(member=%s)"),
			UserIDAttr:   getEnv("LDAP_USER_ID_ATTR", "sAMAccountName"),
			UserNameAttr: getEnv("LDAP_USER_NAME_ATTR", "cn"),
			EmailAttr:    getEnv("LDAP_EMAIL_ATTR", "mail"),
			AdminGroups:  getEnv("LDAP_ADMIN_GROUPS", ""),
		},
		LDAPInit: LDAPInitConfig{
			InitLDAP:      getEnv("INIT_LDAP", "false") == "true",
			RetryCount:    getEnvAsInt("LDAP_INIT_RETRY_COUNT", 30),
			RetryInterval: getEnvAsInt("LDAP_INIT_RETRY_INTERVAL", 2),
			PeopleOU:      getEnv("LDAP_PEOPLE_OU", "test"),
			GroupsOU:      getEnv("LDAP_GROUPS_OU", "groups"),
			AdminUser: AdminUserConfig{
				UID:           getEnv("LDAP_ADMIN_USER_UID", "ldap-admin"),
				Password:      getEnv("LDAP_ADMIN_USER_PASSWORD", "admin123"),
				Email:         getEnv("LDAP_ADMIN_USER_EMAIL", "ldap-admin@test.com"),
				CN:            getEnv("LDAP_ADMIN_USER_CN", "LDAP Administrator"),
				SN:            getEnv("LDAP_ADMIN_USER_SN", "Administrator"),
				GivenName:     getEnv("LDAP_ADMIN_USER_GIVEN_NAME", "LDAP"),
				UIDNumber:     getEnvAsInt("LDAP_ADMIN_USER_UID_NUMBER", 1001),
				GIDNumber:     getEnvAsInt("LDAP_ADMIN_USER_GID_NUMBER", 1001),
				HomeDirectory: getEnv("LDAP_ADMIN_USER_HOME_DIR", "/home/ldap-admin"),
			},
			RegularUser: RegularUserConfig{
				UID:           getEnv("LDAP_REGULAR_USER_UID", "ldap-user"),
				Password:      getEnv("LDAP_REGULAR_USER_PASSWORD", "user123"),
				Email:         getEnv("LDAP_REGULAR_USER_EMAIL", "ldap-user@test.com"),
				CN:            getEnv("LDAP_REGULAR_USER_CN", "LDAP User"),
				SN:            getEnv("LDAP_REGULAR_USER_SN", "User"),
				GivenName:     getEnv("LDAP_REGULAR_USER_GIVEN_NAME", "LDAP"),
				UIDNumber:     getEnvAsInt("LDAP_REGULAR_USER_UID_NUMBER", 1002),
				GIDNumber:     getEnvAsInt("LDAP_REGULAR_USER_GID_NUMBER", 1002),
				HomeDirectory: getEnv("LDAP_REGULAR_USER_HOME_DIR", "/home/ldap-user"),
			},
			AdminGroupCN: getEnv("LDAP_ADMIN_GROUP_CN", "admins"),
			UserGroupCN:  getEnv("LDAP_USER_GROUP_CN", "users"),
		},
		LDAPSync: LDAPSyncRuntime{
			Enabled:         getEnv("LDAP_SYNC_ENABLED", "false") == "true",
			IntervalSeconds: getEnvAsInt("LDAP_SYNC_INTERVAL_SECONDS", 900),
		},
		Gitea: GiteaConfig{
			Enabled: getEnv("GITEA_ENABLED", "false") == "true",
			// IMPORTANT: Use the internal service base URL WITHOUT the web SUBURL (/gitea)
			// Admin API is always rooted at /api/v1 on the service, regardless of SUBURL.
			BaseURL:    getEnv("GITEA_BASE_URL", "http://gitea:3000"),
			AdminToken: getEnv("GITEA_ADMIN_TOKEN", ""),
			AutoCreate: getEnv("GITEA_AUTO_CREATE", "true") == "true",
			AutoUpdate: getEnv("GITEA_AUTO_UPDATE", "true") == "true",
			// Default alias maps backend "admin" to Gitea user "test". Override via env if needed.
			AliasAdminTo: getEnv("GITEA_ALIAS_ADMIN_TO", "test"),
		},
		GiteaSync: GiteaSyncRuntime{
			Enabled:         getEnv("GITEA_SYNC_ENABLED", "false") == "true",
			IntervalSeconds: getEnvAsInt("GITEA_SYNC_INTERVAL_SECONDS", 600),
		},
		AppHub: AppHubConfig{
			Host:   getEnv("EXTERNAL_HOST", "localhost"),
			Port:   getEnv("APPHUB_PORT", "28080"),
			Scheme: getEnv("EXTERNAL_SCHEME", "http"),
		},
		Keycloak: KeycloakConfig{
			Enabled:        getEnv("KEYCLOAK_ENABLED", "false") == "true",
			BaseURL:        getEnv("KEYCLOAK_BASE_URL", "http://keycloak:8080/auth"),
			Realm:          getEnv("KEYCLOAK_REALM", "ai-infra"),
			ClientID:       getEnv("KEYCLOAK_CLIENT_ID", "ai-infra-backend"),
			ClientSecret:   getEnv("KEYCLOAK_BACKEND_CLIENT_SECRET", ""),
			AdminUser:      getEnv("KEYCLOAK_ADMIN", "admin"),
			AdminPass:      getEnv("KEYCLOAK_ADMIN_PASSWORD", "admin"),
			SSOEnabled:     getEnv("KEYCLOAK_SSO_ENABLED", "false") == "true",
			SSORedirectURL: getEnv("KEYCLOAK_SSO_REDIRECT_URL", ""),
		},
		Nightingale: NightingaleConfig{
			Enabled:                 getEnv("NIGHTINGALE_ENABLED", "true") == "true",
			BaseURL:                 getEnv("NIGHTINGALE_BASE_URL", "http://nightingale:17000"),
			APIUsername:             getEnv("NIGHTINGALE_API_USERNAME", "n9e-api"),
			APIPassword:             getEnv("NIGHTINGALE_API_PASSWORD", "e10adc3949ba59abbe56e057f20f883e"),
			UserSyncEnabled:         getEnv("NIGHTINGALE_USER_SYNC_ENABLED", "false") == "true",
			UserSyncIntervalSeconds: getEnvAsInt("NIGHTINGALE_USER_SYNC_INTERVAL_SECONDS", 600),
		},
		ArgoCD: ArgoCDConfig{
			Enabled:      getEnv("ARGOCD_ENABLED", "false") == "true",
			BaseURL:      getEnv("ARGOCD_BASE_URL", "http://argocd-server:8080"),
			AdminToken:   getEnv("ARGOCD_ADMIN_TOKEN", ""),
			GiteaRepoURL: getEnv("ARGOCD_GITEA_URL", "http://gitea:3000"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
