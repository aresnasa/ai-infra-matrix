package database

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		"host":     cfg.Database.Host,
		"port":     cfg.Database.Port,
		"database": cfg.Database.DBName,
		"user":     cfg.Database.User,
	}).Info("Connecting to database")

	// 根据应用日志级别设置GORM日志级别（OceanBase 和 Postgres 复用）
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

	// OceanBase 优先（如果启用）
	if cfg.OceanBase.Enabled {
		obDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
			cfg.OceanBase.User,
			cfg.OceanBase.Password,
			cfg.OceanBase.Host,
			cfg.OceanBase.Port,
			cfg.OceanBase.DBName,
			cfg.OceanBase.Params,
		)
		logrus.WithFields(logrus.Fields{
			"host":     cfg.OceanBase.Host,
			"port":     cfg.OceanBase.Port,
			"database": cfg.OceanBase.DBName,
			"user":     cfg.OceanBase.User,
		}).Info("Connecting to OceanBase (MySQL protocol)")

		var err error
		DB, err = gorm.Open(mysql.Open(obDSN), &gorm.Config{
			Logger:  logger.Default.LogMode(gormLogLevel),
			NowFunc: func() time.Time { return time.Now().Local() },
		})
		if err != nil {
			logrus.WithError(err).Error("Failed to connect to OceanBase")
			return fmt.Errorf("failed to connect to OceanBase: %w", err)
		}

		// 配置连接池
		sqlDB, err := DB.DB()
		if err != nil {
			logrus.WithError(err).Error("Failed to get OceanBase database instance")
			return fmt.Errorf("failed to get OceanBase database instance: %w", err)
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)

		logrus.WithFields(logrus.Fields{
			"max_idle_conns":    10,
			"max_open_conns":    100,
			"conn_max_lifetime": "1h",
		}).Debug("OceanBase connection pool configured")

		logrus.Info("OceanBase connected successfully")
		return nil
	}

	// 默认使用 Postgres
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
		// 先禁用外键自动创建，待所有表创建后再手动添加，避免创建顺序导致的引用错误
		DisableForeignKeyConstraintWhenMigrating: true,
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
		"max_idle_conns":    10,
		"max_open_conns":    100,
		"conn_max_lifetime": "1h",
	}).Debug("Database connection pool configured")

	logrus.Info("PostgreSQL connected successfully")
	return nil
}

// fixSlurmTasksTableSchema 修复 slurm_tasks 表的 task_id 字段类型
// 确保 task_id 是 VARCHAR(36) 而不是 bigint
func fixSlurmTasksTableSchema() error {
	// 检查表是否存在
	var exists bool
	err := DB.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'slurm_tasks')").Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("check table existence: %w", err)
	}

	if !exists {
		logrus.Info("slurm_tasks table does not exist yet, skipping schema fix")
		return nil
	}

	// 检查 task_id 字段的当前类型
	var dataType string
	err = DB.Raw(`
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_schema = CURRENT_SCHEMA() 
		  AND table_name = 'slurm_tasks' 
		  AND column_name = 'task_id'
	`).Scan(&dataType).Error

	if err != nil {
		return fmt.Errorf("check task_id column type: %w", err)
	}

	logrus.Infof("Current task_id column type: %s", dataType)

	// 如果是 bigint，需要修改为 varchar(36)
	if dataType == "bigint" {
		logrus.Info("Fixing task_id column type from bigint to varchar(36)...")

		// 清空表数据（因为类型不兼容）
		if err := DB.Exec("TRUNCATE TABLE slurm_task_events CASCADE").Error; err != nil {
			return fmt.Errorf("truncate slurm_task_events: %w", err)
		}
		if err := DB.Exec("TRUNCATE TABLE slurm_tasks CASCADE").Error; err != nil {
			return fmt.Errorf("truncate slurm_tasks: %w", err)
		}

		// 删除旧索引
		if err := DB.Exec("DROP INDEX IF EXISTS idx_slurm_tasks_task_id").Error; err != nil {
			return fmt.Errorf("drop old index: %w", err)
		}

		// 修改字段类型
		if err := DB.Exec("ALTER TABLE slurm_tasks ALTER COLUMN task_id TYPE VARCHAR(36)").Error; err != nil {
			return fmt.Errorf("alter column type: %w", err)
		}

		// 重建索引
		if err := DB.Exec("CREATE UNIQUE INDEX idx_slurm_tasks_task_id ON slurm_tasks(task_id)").Error; err != nil {
			return fmt.Errorf("create unique index: %w", err)
		}

		logrus.Info("✓ Successfully fixed task_id column type to varchar(36)")
	} else if dataType == "character varying" {
		logrus.Info("✓ task_id column type is already varchar, no fix needed")
	}

	return nil
}

func Migrate() error {
	logrus.Info("Starting database migration...")

	// 修复 slurm_tasks 表的 task_id 字段类型（如果需要）
	if err := fixSlurmTasksTableSchema(); err != nil {
		logrus.WithError(err).Warn("Failed to fix slurm_tasks table schema, continuing with migration...")
	}

	// 一次性迁移所有表，让GORM自动处理表创建顺序
	logrus.Info("Migrating all tables...")
	if err := DB.AutoMigrate(
		// 基础表
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
		// SLURM集群相关模型 - 先创建主表
		&models.SlurmCluster{},
		&models.SlurmNode{},
		&models.ClusterDeployment{},
		&models.NodeInstallTask{},
		&models.DeploymentStep{},
		&models.InstallStep{},
		&models.SSHExecutionLog{},
		&models.NodeTemplate{},
		// 安装任务相关表
		&models.InstallationTask{},
		&models.InstallationHostResult{},
		// 作业管理相关表
		&models.Cluster{},
		&models.Job{},
		&models.JobTemplate{},
		// SLURM任务管理相关表
		&models.SlurmTask{},
		&models.SlurmTaskEvent{},
		&models.SlurmTaskStatistics{},
		// 对象存储相关表
		&models.ObjectStorageConfig{},
		&models.ObjectStorageLog{},
	); err != nil {
		logrus.WithError(err).Error("Database migration failed")
		return fmt.Errorf("database migration failed: %w", err)
	}

	// AutoMigrate 完成后，再次修复 slurm_tasks 表的 task_id 字段类型
	// 这是为了确保即使 GORM 错误地创建了 bigint 类型，我们也能修复它
	logrus.Info("Post-migration: fixing slurm_tasks table schema...")
	if err := fixSlurmTasksTableSchema(); err != nil {
		logrus.WithError(err).Error("Failed to fix slurm_tasks table schema after migration")
		return fmt.Errorf("failed to fix slurm_tasks table schema: %w", err)
	}

	// 所有表创建完成后，按顺序补充外键约束（PostgreSQL）
	// 这样可以避免 AutoMigrate 在创建子表时引用父表尚未存在导致的 42P01 错误
	if err := addForeignKeys(); err != nil {
		logrus.WithError(err).Error("Failed to add foreign key constraints after migration")
		return err
	}

	// 执行自定义迁移（添加新字段等）
	if err := runCustomMigrations(); err != nil {
		logrus.WithError(err).Error("Failed to run custom migrations")
		return err
	}

	// 注册GORM钩子用于自动加密/解密
	if err := registerEncryptionHooks(); err != nil {
		logrus.WithError(err).Error("Failed to register encryption hooks")
		return fmt.Errorf("failed to register encryption hooks: %w", err)
	}

	logrus.Info("Database migration completed successfully")
	return nil
}

// addForeignKeys 在表全部创建后补充必要的外键约束（幂等）
func addForeignKeys() error {
	// 仅在 PostgreSQL 下执行外键补充，其他方言（如 OceanBase/MySQL）跳过
	if DB == nil || DB.Dialector == nil || DB.Dialector.Name() != "postgres" {
		logrus.Debug("Skipping addForeignKeys: not a PostgreSQL dialect")
		return nil
	}

	// install_steps.task_id -> node_install_tasks.id（删除任务时级联删除步骤）
	if err := addForeignKeyIfNotExists(
		"install_steps", "task_id", "node_install_tasks", "id",
		"fk_install_steps_task", "CASCADE", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_install_steps_task: %w", err)
	}

	// node_install_tasks.node_id -> slurm_nodes.id（删除节点时级联删除其任务）
	if err := addForeignKeyIfNotExists(
		"node_install_tasks", "node_id", "slurm_nodes", "id",
		"fk_node_install_tasks_node", "CASCADE", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_node_install_tasks_node: %w", err)
	}

	// node_install_tasks.deployment_id -> cluster_deployments.id（删除部署时将引用置空）
	if err := addForeignKeyIfNotExists(
		"node_install_tasks", "deployment_id", "cluster_deployments", "id",
		"fk_node_install_tasks_deployment", "SET NULL", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_node_install_tasks_deployment: %w", err)
	}

	// deployment_steps.deployment_id -> cluster_deployments.id（删除部署时级联删除步骤）
	if err := addForeignKeyIfNotExists(
		"deployment_steps", "deployment_id", "cluster_deployments", "id",
		"fk_deployment_steps_deployment", "CASCADE", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_deployment_steps_deployment: %w", err)
	}

	// ssh_execution_logs.node_id -> slurm_nodes.id（节点删除时置空日志引用）
	if err := addForeignKeyIfNotExists(
		"ssh_execution_logs", "node_id", "slurm_nodes", "id",
		"fk_ssh_logs_node", "SET NULL", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_ssh_logs_node: %w", err)
	}

	// ssh_execution_logs.task_id -> node_install_tasks.id（任务删除时置空日志引用）
	if err := addForeignKeyIfNotExists(
		"ssh_execution_logs", "task_id", "node_install_tasks", "id",
		"fk_ssh_logs_task", "SET NULL", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_ssh_logs_task: %w", err)
	}

	// ssh_execution_logs.step_id -> deployment_steps.id（步骤删除时置空日志引用）
	if err := addForeignKeyIfNotExists(
		"ssh_execution_logs", "step_id", "deployment_steps", "id",
		"fk_ssh_logs_step", "SET NULL", "CASCADE",
	); err != nil {
		return fmt.Errorf("add fk_ssh_logs_step: %w", err)
	}

	logrus.Info("Foreign key constraints ensured successfully")
	return nil
}

// addForeignKeyIfNotExists 仅当约束不存在时添加外键（PostgreSQL）
func addForeignKeyIfNotExists(table, column, refTable, refColumn, constraint, onDelete, onUpdate string) error {
	// 使用 DO $$ 块检查 pg_constraint 中是否已存在该约束，避免重复创建
	sql := fmt.Sprintf(`
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = '%s'
	) THEN
		ALTER TABLE "%s" ADD CONSTRAINT "%s"
		FOREIGN KEY ("%s") REFERENCES "%s"("%s")
		ON UPDATE %s ON DELETE %s;
	END IF;
END
$$;`, constraint, table, constraint, column, refTable, refColumn, onUpdate, onDelete)

	if err := DB.Exec(sql).Error; err != nil {
		return err
	}
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

	// 注册 ObjectStorageConfig 的加密钩子
	err = DB.Callback().Create().Before("gorm:create").Register("encrypt_object_storage_config", func(db *gorm.DB) {
		if config, ok := db.Statement.Dest.(*models.ObjectStorageConfig); ok {
			if CryptoService != nil {
				if config.AccessKey != "" {
					encrypted, encErr := CryptoService.EncryptIfNeeded(config.AccessKey)
					if encErr != nil {
						logrus.WithError(encErr).Error("Failed to encrypt ObjectStorage AccessKey during create")
						db.AddError(encErr)
						return
					}
					config.AccessKey = encrypted
				}
				if config.SecretKey != "" {
					encrypted, encErr := CryptoService.EncryptIfNeeded(config.SecretKey)
					if encErr != nil {
						logrus.WithError(encErr).Error("Failed to encrypt ObjectStorage SecretKey during create")
						db.AddError(encErr)
						return
					}
					config.SecretKey = encrypted
				}
				logrus.Debug("ObjectStorage credentials encrypted during create operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register create hook for ObjectStorageConfig: %w", err)
	}

	err = DB.Callback().Update().Before("gorm:update").Register("encrypt_object_storage_config_update", func(db *gorm.DB) {
		if config, ok := db.Statement.Dest.(*models.ObjectStorageConfig); ok {
			if CryptoService != nil {
				if config.AccessKey != "" {
					encrypted, encErr := CryptoService.EncryptIfNeeded(config.AccessKey)
					if encErr != nil {
						logrus.WithError(encErr).Error("Failed to encrypt ObjectStorage AccessKey during update")
						db.AddError(encErr)
						return
					}
					config.AccessKey = encrypted
				}
				if config.SecretKey != "" {
					encrypted, encErr := CryptoService.EncryptIfNeeded(config.SecretKey)
					if encErr != nil {
						logrus.WithError(encErr).Error("Failed to encrypt ObjectStorage SecretKey during update")
						db.AddError(encErr)
						return
					}
					config.SecretKey = encrypted
				}
				logrus.Debug("ObjectStorage credentials encrypted during update operation")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register update hook for ObjectStorageConfig: %w", err)
	}

	err = DB.Callback().Query().After("gorm:after_query").Register("decrypt_object_storage_config", func(db *gorm.DB) {
		// 处理单个对象
		if config, ok := db.Statement.Dest.(*models.ObjectStorageConfig); ok {
			if CryptoService != nil {
				if config.AccessKey != "" {
					decrypted := CryptoService.DecryptSafely(config.AccessKey)
					config.AccessKey = decrypted
				}
				if config.SecretKey != "" {
					decrypted := CryptoService.DecryptSafely(config.SecretKey)
					config.SecretKey = decrypted
				}
				logrus.Debug("ObjectStorage credentials decrypted during query operation")
			}
		}

		// 处理切片对象
		if configs, ok := db.Statement.Dest.(*[]models.ObjectStorageConfig); ok {
			for i := range *configs {
				config := &(*configs)[i]
				if CryptoService != nil {
					if config.AccessKey != "" {
						decrypted := CryptoService.DecryptSafely(config.AccessKey)
						config.AccessKey = decrypted
					}
					if config.SecretKey != "" {
						decrypted := CryptoService.DecryptSafely(config.SecretKey)
						config.SecretKey = decrypted
					}
				}
			}
			logrus.Debug("ObjectStorage credentials decrypted during query operation (slice)")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register query hook for ObjectStorageConfig: %w", err)
	}

	logrus.Info("Encryption hooks registered successfully")
	return nil
}

// runCustomMigrations 执行自定义数据库迁移脚本
func runCustomMigrations() error {
	logrus.Info("Running custom database migrations...")

	// 检查数据库类型
	dialector := DB.Dialector.Name()

	// 迁移1: 为 slurm_clusters 表添加新字段
	if err := addSlurmClusterFields(dialector); err != nil {
		return fmt.Errorf("failed to add slurm cluster fields: %w", err)
	}

	logrus.Info("Custom database migrations completed successfully")
	return nil
}

// addSlurmClusterFields 为 slurm_clusters 表添加 cluster_type 和 master_ssh 字段
func addSlurmClusterFields(dialector string) error {
	logrus.Info("Adding cluster_type and master_ssh fields to slurm_clusters table...")

	// 检查字段是否已存在
	var columnExists bool

	if dialector == "postgres" {
		// PostgreSQL 语法
		err := DB.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'slurm_clusters' 
				AND column_name = 'cluster_type'
			)
		`).Scan(&columnExists).Error
		if err != nil {
			return fmt.Errorf("failed to check if cluster_type column exists: %w", err)
		}

		if !columnExists {
			// 添加 cluster_type 字段
			if err := DB.Exec(`
				ALTER TABLE slurm_clusters 
				ADD COLUMN cluster_type VARCHAR(50) DEFAULT 'managed'
			`).Error; err != nil {
				return fmt.Errorf("failed to add cluster_type column: %w", err)
			}
			logrus.Info("✓ Added cluster_type column to slurm_clusters")
		} else {
			logrus.Info("✓ cluster_type column already exists")
		}

		// 检查 master_ssh 字段
		err = DB.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'slurm_clusters' 
				AND column_name = 'master_ssh'
			)
		`).Scan(&columnExists).Error
		if err != nil {
			return fmt.Errorf("failed to check if master_ssh column exists: %w", err)
		}

		if !columnExists {
			// 添加 master_ssh 字段 (JSON 类型)
			if err := DB.Exec(`
				ALTER TABLE slurm_clusters 
				ADD COLUMN master_ssh JSONB
			`).Error; err != nil {
				return fmt.Errorf("failed to add master_ssh column: %w", err)
			}
			logrus.Info("✓ Added master_ssh column to slurm_clusters")
		} else {
			logrus.Info("✓ master_ssh column already exists")
		}

	} else {
		// MySQL/MariaDB/OceanBase 语法
		err := DB.Raw(`
			SELECT COUNT(*) > 0 FROM information_schema.columns 
			WHERE table_schema = DATABASE() 
			AND table_name = 'slurm_clusters' 
			AND column_name = 'cluster_type'
		`).Scan(&columnExists).Error
		if err != nil {
			return fmt.Errorf("failed to check if cluster_type column exists: %w", err)
		}

		if !columnExists {
			// 添加 cluster_type 字段
			if err := DB.Exec(`
				ALTER TABLE slurm_clusters 
				ADD COLUMN cluster_type VARCHAR(50) DEFAULT 'managed'
			`).Error; err != nil {
				return fmt.Errorf("failed to add cluster_type column: %w", err)
			}
			logrus.Info("✓ Added cluster_type column to slurm_clusters")
		} else {
			logrus.Info("✓ cluster_type column already exists")
		}

		// 检查 master_ssh 字段
		err = DB.Raw(`
			SELECT COUNT(*) > 0 FROM information_schema.columns 
			WHERE table_schema = DATABASE() 
			AND table_name = 'slurm_clusters' 
			AND column_name = 'master_ssh'
		`).Scan(&columnExists).Error
		if err != nil {
			return fmt.Errorf("failed to check if master_ssh column exists: %w", err)
		}

		if !columnExists {
			// 添加 master_ssh 字段 (JSON 类型)
			if err := DB.Exec(`
				ALTER TABLE slurm_clusters 
				ADD COLUMN master_ssh JSON
			`).Error; err != nil {
				return fmt.Errorf("failed to add master_ssh column: %w", err)
			}
			logrus.Info("✓ Added master_ssh column to slurm_clusters")
		} else {
			logrus.Info("✓ master_ssh column already exists")
		}
	}

	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
