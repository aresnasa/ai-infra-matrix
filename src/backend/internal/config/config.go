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
	InitLDAP     bool `json:"init_ldap"`
	RetryCount   int  `json:"retry_count"`
	RetryInterval int `json:"retry_interval"`
	
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
	Enabled bool   `json:"enabled"`
	// 同步间隔，单位秒（默认900秒=15分钟）
	IntervalSeconds int `json:"interval_seconds"`
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
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	config := &Config{
		Port: getEnv("PORT", "8082"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "ansible_playbook_generator"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     redisPort,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "your-encryption-key-change-in-production-32-bytes"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
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
