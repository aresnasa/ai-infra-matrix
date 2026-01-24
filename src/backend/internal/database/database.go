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

var (
	DB      *gorm.DB
	SlurmDB *gorm.DB
)

// GetDB 返回数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

// GetSlurmDB 返回SLURM专用数据库连接（MySQL/OceanBase）
func GetSlurmDB() *gorm.DB {
	return SlurmDB
}

func Connect(cfg *config.Config) error {
	// 初始化加密服务
	InitCrypto(cfg)

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

	if err := connectTaskStore(cfg, gormLogLevel); err != nil {
		return err
	}

	if err := connectSlurmStore(cfg, gormLogLevel); err != nil {
		return err
	}

	return nil
}

func connectTaskStore(cfg *config.Config, gormLogLevel logger.LogLevel) error {
	logrus.WithFields(logrus.Fields{
		"host":     cfg.Database.Host,
		"port":     cfg.Database.Port,
		"database": cfg.Database.DBName,
		"user":     cfg.Database.User,
	}).Info("Connecting to PostgreSQL task store")

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
		logrus.WithError(err).Error("Failed to connect to PostgreSQL task store")
		return fmt.Errorf("failed to connect to PostgreSQL task store: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logrus.WithError(err).Error("Failed to get PostgreSQL database instance")
		return fmt.Errorf("failed to get PostgreSQL database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logrus.WithFields(logrus.Fields{
		"max_idle_conns":    10,
		"max_open_conns":    100,
		"conn_max_lifetime": "1h",
	}).Debug("PostgreSQL connection pool configured")

	logrus.Info("PostgreSQL task store connected successfully")
	return nil
}

func connectSlurmStore(cfg *config.Config, gormLogLevel logger.LogLevel) error {
	if !cfg.OceanBase.Enabled {
		// 未单独配置时，让 SLURM 复用主库，至少保证功能可用
		SlurmDB = DB
		logrus.Warn("OceanBase/MySQL for SLURM not configured; falling back to primary PostgreSQL store")
		return nil
	}

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
	}).Info("Connecting to SLURM OceanBase/MySQL store")

	var err error
	SlurmDB, err = gorm.Open(mysql.Open(obDSN), &gorm.Config{
		Logger:  logger.Default.LogMode(gormLogLevel),
		NowFunc: func() time.Time { return time.Now().Local() },
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to OceanBase for SLURM")
		return fmt.Errorf("failed to connect to OceanBase for SLURM: %w", err)
	}

	sqlDB, err := SlurmDB.DB()
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
	}).Debug("OceanBase connection pool configured for SLURM")

	logrus.Info("SLURM OceanBase/MySQL store connected successfully")
	return nil
}

// fixSlurmTasksTableSchema 修复 slurm_tasks 表的 task_id 字段类型
// 确保 task_id 是 VARCHAR(36) 而不是 bigint
func fixSlurmTasksTableSchema() error {
	if SlurmDB == nil {
		logrus.Debug("Skipping slurm_tasks schema fix: SLURM database not initialized")
		return nil
	}

	if SlurmDB.Dialector == nil || SlurmDB.Dialector.Name() != "postgres" {
		logrus.Debug("Skipping slurm_tasks schema fix: only required for PostgreSQL")
		return nil
	}

	// 检查表是否存在
	var exists bool
	err := SlurmDB.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'slurm_tasks')").Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("check table existence: %w", err)
	}

	if !exists {
		logrus.Info("slurm_tasks table does not exist yet, skipping schema fix")
		return nil
	}

	// 检查 task_id 字段的当前类型
	var dataType string
	err = SlurmDB.Raw(`
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
		if err := SlurmDB.Exec("TRUNCATE TABLE slurm_task_events CASCADE").Error; err != nil {
			return fmt.Errorf("truncate slurm_task_events: %w", err)
		}
		if err := SlurmDB.Exec("TRUNCATE TABLE slurm_tasks CASCADE").Error; err != nil {
			return fmt.Errorf("truncate slurm_tasks: %w", err)
		}

		// 删除旧索引
		if err := SlurmDB.Exec("DROP INDEX IF EXISTS idx_slurm_tasks_task_id").Error; err != nil {
			return fmt.Errorf("drop old index: %w", err)
		}

		// 修改字段类型
		if err := SlurmDB.Exec("ALTER TABLE slurm_tasks ALTER COLUMN task_id TYPE VARCHAR(36)").Error; err != nil {
			return fmt.Errorf("alter column type: %w", err)
		}

		// 重建索引
		if err := SlurmDB.Exec("CREATE UNIQUE INDEX idx_slurm_tasks_task_id ON slurm_tasks(task_id)").Error; err != nil {
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

	if err := migrateTaskSchema(); err != nil {
		return err
	}

	if err := migrateSlurmSchema(); err != nil {
		return err
	}

	// 所有表创建完成后，按顺序补充外键约束（仅当运行在同一个PostgreSQL库时）
	if err := addForeignKeys(); err != nil {
		logrus.WithError(err).Error("Failed to add foreign key constraints after migration")
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

func migrateTaskSchema() error {
	if DB == nil {
		return fmt.Errorf("primary database connection is not initialized")
	}

	logrus.Info("Migrating PostgreSQL task store tables...")
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Host{},
		&models.Variable{},
		&models.Task{},
		&models.PlaybookGeneration{},
		&models.UserGroup{},
		&models.Permission{},
		&models.Role{},
		&models.RoleBinding{},
		&models.UserGroupMembership{},
		&models.UserRole{},
		&models.UserGroupRole{},
		&models.RolePermission{},
		&models.RoleTemplate{},
		&models.RoleTemplatePermission{},
		&models.UserNavigationConfig{},
		&models.LDAPConfig{},
		&models.KubernetesCluster{},
		&models.AnsibleExecution{},
		&models.AIAssistantConfig{},
		&models.AIConversation{},
		&models.AIMessage{},
		&models.AIUsageStats{},
		&models.JupyterHubConfig{},
		&models.JupyterTask{},
		&models.JupyterLabTemplate{},
		&models.JupyterLabResourceQuota{},
		&models.JupyterLabInstance{},
		&models.InstallationTask{},
		&models.InstallationHostResult{},
		&models.Cluster{},
		&models.Job{},
		&models.JobTemplate{},
		&models.ObjectStorageConfig{},
		&models.ObjectStorageLog{},
		&models.HostTemplate{},
		// SaltStack 任务日志表
		&models.SaltStackTask{},
		&models.TaskLog{},
		&models.SSHLog{},
		// Minion 软删除任务表
		&models.MinionDeleteTask{},
		&models.MinionDeleteLog{},
		// Minion 分组表
		&models.MinionGroup{},
		&models.MinionGroupMembership{},
		// 节点指标表
		&models.NodeMetrics{},
		&models.NodeMetricsLatest{},
		// IB 端口忽略表
		&models.IBPortIgnore{},
		// Salt 作业历史表（持久化用户任务）
		&models.SaltJobHistory{},
		&models.SaltJobConfig{},
		// 安全管理表
		&models.IPBlacklist{},
		&models.IPWhitelist{},
		&models.LoginAttempt{},
		&models.TwoFactorConfig{},
		&models.TwoFactorGlobalConfig{},
		&models.OAuthProvider{},
		&models.UserOAuthBinding{},
		&models.SecurityAuditLog{},
		&models.SecurityConfig{},
		// SLURM task tracking tables live in PostgreSQL for consistency with global task store
		&models.SlurmTask{},
		&models.SlurmTaskEvent{},
		&models.SlurmTaskStatistics{},
		// 基础设施审计日志表
		&models.InfraAuditLog{},
		&models.AuditChangeDetail{},
		&models.AuditArchive{},
		&models.AuditConfig{},
		&models.AuditExportRequest{},
		// 权限审批工作流表
		&models.PermissionRequest{},
		&models.PermissionApprovalLog{},
		&models.PermissionApprovalRule{},
		&models.PermissionGrant{},
	); err != nil {
		logrus.WithError(err).Error("Task store migration failed")
		return fmt.Errorf("task store migration failed: %w", err)
	}
	return nil
}

func migrateSlurmSchema() error {
	if SlurmDB == nil {
		logrus.Warn("SLURM database connection is not initialized; skipping SLURM migrations")
		return nil
	}

	logrus.Info("Migrating SLURM MySQL tables...")
	if err := SlurmDB.AutoMigrate(
		&models.SlurmCluster{},
		&models.SlurmNode{},
		&models.ClusterDeployment{},
		&models.NodeInstallTask{},
		&models.DeploymentStep{},
		&models.InstallStep{},
		&models.SSHExecutionLog{},
		&models.NodeTemplate{},
	); err != nil {
		logrus.WithError(err).Error("SLURM store migration failed")
		return fmt.Errorf("slurm store migration failed: %w", err)
	}

	logrus.Info("Ensuring slurm_tasks schema correctness...")
	if err := fixSlurmTasksTableSchema(); err != nil {
		return err
	}

	if err := runSlurmCustomMigrations(); err != nil {
		return err
	}

	return nil
}

// addForeignKeys 在表全部创建后补充必要的外键约束（幂等）
func addForeignKeys() error {
	// 仅在 PostgreSQL 下执行外键补充，其他方言（如 OceanBase/MySQL）跳过
	if DB == nil || DB.Dialector == nil || DB.Dialector.Name() != "postgres" {
		logrus.Debug("Skipping addForeignKeys: not a PostgreSQL dialect")
		return nil
	}

	if SlurmDB == nil || SlurmDB != DB {
		logrus.Debug("Skipping addForeignKeys: SLURM tables stored separately")
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

// runSlurmCustomMigrations 执行 SLURM 相关的定制迁移
func runSlurmCustomMigrations() error {
	if SlurmDB == nil {
		logrus.Debug("Skipping SLURM custom migrations: database not initialized")
		return nil
	}

	logrus.Info("Running custom SLURM database migrations...")

	// 检查数据库类型
	dialector := SlurmDB.Dialector.Name()

	// 迁移1: 为 slurm_clusters 表添加新字段
	if err := addSlurmClusterFields(SlurmDB, dialector); err != nil {
		return fmt.Errorf("failed to add slurm cluster fields: %w", err)
	}

	logrus.Info("Custom SLURM database migrations completed successfully")
	return nil
}

// addSlurmClusterFields 为 slurm_clusters 表添加 cluster_type 和 master_ssh 字段
func addSlurmClusterFields(db *gorm.DB, dialector string) error {
	logrus.Info("Adding cluster_type and master_ssh fields to slurm_clusters table...")

	// 检查字段是否已存在
	var columnExists bool

	if dialector == "postgres" {
		// PostgreSQL 语法
		err := db.Raw(`
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
			if err := db.Exec(`
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
		err = db.Raw(`
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
			if err := db.Exec(`
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
		err := db.Raw(`
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
			if err := db.Exec(`
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
		err = db.Raw(`
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
			if err := db.Exec(`
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
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Close(); err != nil {
			return err
		}
	}

	if SlurmDB != nil && SlurmDB != DB {
		sqlDB, err := SlurmDB.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Close(); err != nil {
			return err
		}
	}

	return nil
}

// MigrateSensitiveData 自动检测并加密未加密的敏感数据
// 该函数在系统启动时自动调用，确保所有敏感数据都已加密
func MigrateSensitiveData() error {
	if CryptoService == nil {
		logrus.Warn("Crypto service not initialized, skipping sensitive data migration")
		return nil
	}

	logrus.Info("Checking and migrating unencrypted sensitive data...")

	// 1. 迁移 ObjectStorageConfig 的 access_key 和 secret_key
	if err := migrateObjectStorageConfigs(); err != nil {
		logrus.WithError(err).Warn("Failed to migrate object_storage_configs")
	}

	// 2. 迁移 SlurmNode 的 username 和 password
	if err := migrateSlurmNodeCredentials(); err != nil {
		logrus.WithError(err).Warn("Failed to migrate slurm_nodes credentials")
	}

	// 3. 迁移 AIAssistantConfig 的 api_key 和 api_secret
	if err := migrateAIAssistantConfigs(); err != nil {
		logrus.WithError(err).Warn("Failed to migrate ai_assistant_configs")
	}

	logrus.Info("Sensitive data migration check completed")
	return nil
}

// migrateObjectStorageConfigs 加密 object_storage_configs 表中未加密的敏感数据
func migrateObjectStorageConfigs() error {
	type ObjectStorageConfigRaw struct {
		ID        uint
		AccessKey string `gorm:"column:access_key"`
		SecretKey string `gorm:"column:secret_key"`
	}

	var configs []ObjectStorageConfigRaw
	if err := DB.Table("object_storage_configs").Select("id, access_key, secret_key").Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to query object_storage_configs: %w", err)
	}

	migratedCount := 0
	for _, cfg := range configs {
		needUpdate := false
		updates := make(map[string]interface{})

		// 检查并加密 access_key
		if cfg.AccessKey != "" && !CryptoService.IsEncrypted(cfg.AccessKey) {
			encrypted, err := CryptoService.Encrypt(cfg.AccessKey)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to encrypt access_key for config %d", cfg.ID)
				continue
			}
			updates["access_key"] = encrypted
			needUpdate = true
		}

		// 检查并加密 secret_key
		if cfg.SecretKey != "" && !CryptoService.IsEncrypted(cfg.SecretKey) {
			encrypted, err := CryptoService.Encrypt(cfg.SecretKey)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to encrypt secret_key for config %d", cfg.ID)
				continue
			}
			updates["secret_key"] = encrypted
			needUpdate = true
		}

		if needUpdate {
			if err := DB.Table("object_storage_configs").Where("id = ?", cfg.ID).Updates(updates).Error; err != nil {
				logrus.WithError(err).Warnf("Failed to update object_storage_config %d", cfg.ID)
				continue
			}
			migratedCount++
		}
	}

	if migratedCount > 0 {
		logrus.Infof("Migrated %d object_storage_configs records (encrypted credentials)", migratedCount)
	}
	return nil
}

// migrateSlurmNodeCredentials 加密 slurm_nodes 表中未加密的敏感数据
func migrateSlurmNodeCredentials() error {
	type SlurmNodeRaw struct {
		ID       uint
		Username string
		Password string
	}

	var nodes []SlurmNodeRaw
	if err := DB.Table("slurm_nodes").Select("id, username, password").Find(&nodes).Error; err != nil {
		return fmt.Errorf("failed to query slurm_nodes: %w", err)
	}

	migratedCount := 0
	for _, node := range nodes {
		needUpdate := false
		updates := make(map[string]interface{})

		if node.Username != "" && !CryptoService.IsEncrypted(node.Username) {
			encrypted, err := CryptoService.Encrypt(node.Username)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to encrypt username for node %d", node.ID)
				continue
			}
			updates["username"] = encrypted
			needUpdate = true
		}

		if node.Password != "" && !CryptoService.IsEncrypted(node.Password) {
			encrypted, err := CryptoService.Encrypt(node.Password)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to encrypt password for node %d", node.ID)
				continue
			}
			updates["password"] = encrypted
			needUpdate = true
		}

		if needUpdate {
			if err := DB.Table("slurm_nodes").Where("id = ?", node.ID).Updates(updates).Error; err != nil {
				logrus.WithError(err).Warnf("Failed to update slurm_node %d", node.ID)
				continue
			}
			migratedCount++
		}
	}

	if migratedCount > 0 {
		logrus.Infof("Migrated %d slurm_nodes records (encrypted credentials)", migratedCount)
	}
	return nil
}

// migrateAIAssistantConfigs 加密 ai_assistant_configs 表中未加密的敏感数据
func migrateAIAssistantConfigs() error {
	type AIConfigRaw struct {
		ID        uint
		APIKey    string `gorm:"column:api_key"`
		APISecret string `gorm:"column:api_secret"`
	}

	var configs []AIConfigRaw
	if err := DB.Table("ai_assistant_configs").Select("id, api_key, api_secret").Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to query ai_assistant_configs: %w", err)
	}

	migratedCount := 0
	for _, cfg := range configs {
		needUpdate := false
		updates := make(map[string]interface{})

		if cfg.APIKey != "" && !CryptoService.IsEncrypted(cfg.APIKey) {
			encrypted, err := CryptoService.Encrypt(cfg.APIKey)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to encrypt api_key for config %d", cfg.ID)
				continue
			}
			updates["api_key"] = encrypted
			needUpdate = true
		}

		if cfg.APISecret != "" && !CryptoService.IsEncrypted(cfg.APISecret) {
			encrypted, err := CryptoService.Encrypt(cfg.APISecret)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to encrypt api_secret for config %d", cfg.ID)
				continue
			}
			updates["api_secret"] = encrypted
			needUpdate = true
		}

		if needUpdate {
			if err := DB.Table("ai_assistant_configs").Where("id = ?", cfg.ID).Updates(updates).Error; err != nil {
				logrus.WithError(err).Warnf("Failed to update ai_assistant_config %d", cfg.ID)
				continue
			}
			migratedCount++
		}
	}

	if migratedCount > 0 {
		logrus.Infof("Migrated %d ai_assistant_configs records (encrypted credentials)", migratedCount)
	}
	return nil
}
