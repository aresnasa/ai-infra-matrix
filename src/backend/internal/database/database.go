package database

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/sirupsen/logrus"
)

var DB *gorm.DB

// GetDB 返回数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

func Connect(cfg *config.Config) error {
	// 初始化加密服务
	InitCrypto(cfg)
	
	logrus.WithFields(logrus.Fields{
		"host": cfg.Database.Host,
		"port": cfg.Database.Port,
		"database": cfg.Database.DBName,
		"user": cfg.Database.User,
	}).Info("Connecting to database")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	// 根据应用日志级别设置GORM日志级别
	var gormLogLevel logger.LogLevel
	appLogLevel := logrus.GetLevel()
	switch appLogLevel {
	case logrus.TraceLevel, logrus.DebugLevel:
		gormLogLevel = logger.Info // 显示所有SQL语句
		logrus.Debug("GORM logging set to Info level (shows all SQL)")
	case logrus.InfoLevel:
		gormLogLevel = logger.Warn // 只显示慢查询和错误
		logrus.Debug("GORM logging set to Warn level (slow queries and errors)")
	default:
		gormLogLevel = logger.Error // 只显示错误
		logrus.Debug("GORM logging set to Error level (errors only)")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to database")
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		logrus.WithError(err).Error("Failed to get database instance")
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logrus.WithFields(logrus.Fields{
		"max_idle_conns": 10,
		"max_open_conns": 100,
		"conn_max_lifetime": "1h",
	}).Debug("Database connection pool configured")

	logrus.Info("Database connected successfully")
	return nil
}

func Migrate() error {
	logrus.Info("Starting database migration...")
	
	err := DB.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Host{},
		&models.Variable{},
		&models.Task{},
		&models.PlaybookGeneration{},
		// RBAC 相关表
		&models.UserGroup{},
		&models.Permission{},
		&models.Role{},
		&models.RoleBinding{},
		&models.UserGroupMembership{},
		&models.UserRole{},
		&models.UserGroupRole{},
		&models.RolePermission{},
		// 用户导航配置表
		&models.UserNavigationConfig{},
		// LDAP 配置表
		&models.LDAPConfig{},
		// Kubernetes 和 Ansible 相关表
		&models.KubernetesCluster{},
		&models.AnsibleExecution{},
		// AI 助手相关表
		&models.AIAssistantConfig{},
		&models.AIConversation{},
		&models.AIMessage{},
		&models.AIUsageStats{},
		// JupyterHub 相关表
		&models.JupyterHubConfig{},
		&models.JupyterTask{},
		// JupyterLab 模板相关表
		&models.JupyterLabTemplate{},
		&models.JupyterLabResourceQuota{},
		&models.JupyterLabInstance{},
	)
	if err != nil {
		logrus.Errorf("Database migration failed: %v", err)
		return fmt.Errorf("database migration failed: %w", err)
	}

	// 注册GORM钩子用于自动加密/解密
	if err := registerEncryptionHooks(); err != nil {
		logrus.WithError(err).Error("Failed to register encryption hooks")
		return fmt.Errorf("failed to register encryption hooks: %w", err)
	}

	logrus.Info("Database migration completed")
	return nil
}

// registerEncryptionHooks 注册加密钩子
func registerEncryptionHooks() error {
	// 注册 KubernetesCluster 的加密钩子
	err := DB.Callback().Create().Before("gorm:create").Register("encrypt_kubernetes_cluster", func(db *gorm.DB) {
		if cluster, ok := db.Statement.Dest.(*models.KubernetesCluster); ok {
			if CryptoService != nil && cluster.KubeConfig != "" {
				encrypted, err := CryptoService.EncryptIfNeeded(cluster.KubeConfig)
				if err != nil {
					logrus.WithError(err).Error("Failed to encrypt KubeConfig during create")
					db.AddError(err)
					return
				}
				cluster.KubeConfig = encrypted
				logrus.Debug("KubeConfig encrypted during create operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register create hook for KubernetesCluster: %w", err)
	}

	err = DB.Callback().Update().Before("gorm:update").Register("encrypt_kubernetes_cluster_update", func(db *gorm.DB) {
		if cluster, ok := db.Statement.Dest.(*models.KubernetesCluster); ok {
			if CryptoService != nil && cluster.KubeConfig != "" {
				encrypted, err := CryptoService.EncryptIfNeeded(cluster.KubeConfig)
				if err != nil {
					logrus.WithError(err).Error("Failed to encrypt KubeConfig during update")
					db.AddError(err)
					return
				}
				cluster.KubeConfig = encrypted
				logrus.Debug("KubeConfig encrypted during update operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register update hook for KubernetesCluster: %w", err)
	}

	err = DB.Callback().Query().After("gorm:after_query").Register("decrypt_kubernetes_cluster", func(db *gorm.DB) {
		// 处理单个对象
		if cluster, ok := db.Statement.Dest.(*models.KubernetesCluster); ok {
			if CryptoService != nil && cluster.KubeConfig != "" {
				decrypted := CryptoService.DecryptSafely(cluster.KubeConfig)
				cluster.KubeConfig = decrypted
				logrus.Debug("KubeConfig decrypted during query operation")
			}
		}
		
		// 处理切片对象
		if clusters, ok := db.Statement.Dest.(*[]models.KubernetesCluster); ok {
			for i := range *clusters {
				cluster := &(*clusters)[i]
				if CryptoService != nil && cluster.KubeConfig != "" {
					decrypted := CryptoService.DecryptSafely(cluster.KubeConfig)
					cluster.KubeConfig = decrypted
				}
			}
			logrus.Debug("KubeConfig decrypted during query operation (slice)")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register query hook for KubernetesCluster: %w", err)
	}

	// 注册 LDAPConfig 的加密钩子
	err = DB.Callback().Create().Before("gorm:create").Register("encrypt_ldap_config", func(db *gorm.DB) {
		if config, ok := db.Statement.Dest.(*models.LDAPConfig); ok {
			if CryptoService != nil && config.BindPassword != "" {
				encrypted, err := CryptoService.EncryptIfNeeded(config.BindPassword)
				if err != nil {
					logrus.WithError(err).Error("Failed to encrypt LDAP BindPassword during create")
					db.AddError(err)
					return
				}
				config.BindPassword = encrypted
				logrus.Debug("LDAP BindPassword encrypted during create operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register create hook for LDAPConfig: %w", err)
	}

	err = DB.Callback().Update().Before("gorm:update").Register("encrypt_ldap_config_update", func(db *gorm.DB) {
		if config, ok := db.Statement.Dest.(*models.LDAPConfig); ok {
			if CryptoService != nil && config.BindPassword != "" {
				encrypted, err := CryptoService.EncryptIfNeeded(config.BindPassword)
				if err != nil {
					logrus.WithError(err).Error("Failed to encrypt LDAP BindPassword during update")
					db.AddError(err)
					return
				}
				config.BindPassword = encrypted
				logrus.Debug("LDAP BindPassword encrypted during update operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register update hook for LDAPConfig: %w", err)
	}

	err = DB.Callback().Query().After("gorm:after_query").Register("decrypt_ldap_config", func(db *gorm.DB) {
		// 处理单个对象
		if config, ok := db.Statement.Dest.(*models.LDAPConfig); ok {
			if CryptoService != nil && config.BindPassword != "" {
				decrypted := CryptoService.DecryptSafely(config.BindPassword)
				config.BindPassword = decrypted
				logrus.Debug("LDAP BindPassword decrypted during query operation")
			}
		}
		
		// 处理切片对象
		if configs, ok := db.Statement.Dest.(*[]models.LDAPConfig); ok {
			for i := range *configs {
				config := &(*configs)[i]
				if CryptoService != nil && config.BindPassword != "" {
					decrypted := CryptoService.DecryptSafely(config.BindPassword)
					config.BindPassword = decrypted
				}
			}
			logrus.Debug("LDAP BindPassword decrypted during query operation (slice)")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register query hook for LDAPConfig: %w", err)
	}

	// 注册 AIAssistantConfig 的加密钩子
	err = DB.Callback().Create().Before("gorm:create").Register("encrypt_ai_config", func(db *gorm.DB) {
		if config, ok := db.Statement.Dest.(*models.AIAssistantConfig); ok {
			if CryptoService != nil && config.APIKey != "" {
				encrypted, err := CryptoService.EncryptIfNeeded(config.APIKey)
				if err != nil {
					logrus.WithError(err).Error("Failed to encrypt AI API key during create")
					db.AddError(err)
					return
				}
				config.APIKey = encrypted
				logrus.Debug("AI API key encrypted during create operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register create hook for AIAssistantConfig: %w", err)
	}

	err = DB.Callback().Update().Before("gorm:update").Register("encrypt_ai_config_update", func(db *gorm.DB) {
		if config, ok := db.Statement.Dest.(*models.AIAssistantConfig); ok {
			if CryptoService != nil && config.APIKey != "" {
				encrypted, err := CryptoService.EncryptIfNeeded(config.APIKey)
				if err != nil {
					logrus.WithError(err).Error("Failed to encrypt AI API key during update")
					db.AddError(err)
					return
				}
				config.APIKey = encrypted
				logrus.Debug("AI API key encrypted during update operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register update hook for AIAssistantConfig: %w", err)
	}

	err = DB.Callback().Query().After("gorm:after_query").Register("decrypt_ai_config", func(db *gorm.DB) {
		// 处理单个对象
		if config, ok := db.Statement.Dest.(*models.AIAssistantConfig); ok {
			if CryptoService != nil && config.APIKey != "" {
				decrypted := CryptoService.DecryptSafely(config.APIKey)
				config.APIKey = decrypted
				logrus.Debug("AI API key decrypted during query operation")
			}
		}
		
		// 处理切片对象
		if configs, ok := db.Statement.Dest.(*[]models.AIAssistantConfig); ok {
			for i := range *configs {
				config := &(*configs)[i]
				if CryptoService != nil && config.APIKey != "" {
					decrypted := CryptoService.DecryptSafely(config.APIKey)
					config.APIKey = decrypted
				}
			}
			logrus.Debug("AI API key decrypted during query operation (slice)")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register query hook for AIAssistantConfig: %w", err)
	}

	logrus.Info("Encryption hooks registered successfully")
	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
