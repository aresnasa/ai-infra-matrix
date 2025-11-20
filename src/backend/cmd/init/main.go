package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"net/http"
	"os"

	"github.com/go-ldap/ldap/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// quoteIdentifier safely quotes PostgreSQL identifiers to prevent SQL injection
// This follows PostgreSQL's identifier quoting rules
func quoteIdentifier(name string) string {
	// Replace double quotes with escaped double quotes
	escaped := strings.ReplaceAll(name, "\"", "\"\"")
	// Wrap in double quotes
	return fmt.Sprintf("\"%s\"", escaped)
}

// quoteLiteral safely quotes PostgreSQL string literals to prevent SQL injection
// This follows PostgreSQL's string literal quoting rules
func quoteLiteral(value string) string {
	// Replace single quotes with escaped single quotes
	escaped := strings.ReplaceAll(value, "'", "''")
	// Wrap in single quotes
	return fmt.Sprintf("'%s'", escaped)
}

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 检查数据库是否存在，如果存在则备份并重建
	if err := handleDatabaseReset(cfg); err != nil {
		log.Fatal("Failed to handle database reset:", err)
	}

	// 连接数据库
	if err := database.Connect(cfg); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 运行数据库迁移
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// 创建JupyterHub数据库
	if err := createJupyterHubDatabase(cfg); err != nil {
		log.Fatal("Failed to create JupyterHub database:", err)
	}

	// 创建Gitea数据库与用户（按照当前 .env 配置）
	if err := createGiteaDatabase(cfg); err != nil {
		log.Fatal("Failed to create Gitea database:", err)
	}

	// 创建SLURM数据库与用户
	if err := createSLURMDatabase(cfg); err != nil {
		log.Fatal("Failed to create SLURM database:", err)
	}

	// 创建Nightingale数据库与用户
	if err := createNightingaleDatabase(cfg); err != nil {
		log.Fatal("Failed to create Nightingale database:", err)
	}

	// 初始化RBAC系统
	if err := initializeRBAC(); err != nil {
		log.Fatal("Failed to initialize RBAC:", err)
	}

	// 创建默认管理员用户
	createDefaultAdmin()

	// 初始化默认AI配置
	initializeDefaultAIConfigs()

	// 初始化LDAP用户（如果LDAP服务可用）
	initializeLDAPUsers(cfg)

	// 初始化并同步 Gitea 用户（如果启用）
	initializeGiteaUsers(cfg)

	log.Println("Initialization completed successfully!")
}

func handleDatabaseReset(cfg *config.Config) error {
	// 连接到 postgres 系统数据库来检查目标数据库是否存在
	systemDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to system database: %w", err)
	}

	// 检查目标数据库是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = ?)"
	if err := systemDB.Raw(query, cfg.Database.DBName).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		log.Printf("Database '%s' already exists", cfg.Database.DBName)

		// 创建备份数据库名称
		backupDBName := fmt.Sprintf("%s_backup_%s", cfg.Database.DBName, time.Now().Format("20060102_150405"))

		// 备份现有数据库 - Use quoted identifiers to prevent SQL injection
		log.Printf("Creating backup database: %s", backupDBName)
		backupQuery := fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s",
			quoteIdentifier(backupDBName), quoteIdentifier(cfg.Database.DBName))
		if err := systemDB.Exec(backupQuery).Error; err != nil {
			log.Printf("Warning: Failed to create backup database: %v", err)
		} else {
			log.Printf("Backup database created successfully: %s", backupDBName)
		}

		// 终止所有连接到目标数据库的连接
		log.Printf("Terminating connections to database: %s", cfg.Database.DBName)
		terminateQuery := `
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = ? AND pid <> pg_backend_pid()
		`
		if err := systemDB.Exec(terminateQuery, cfg.Database.DBName).Error; err != nil {
			log.Printf("Warning: Failed to terminate connections: %v", err)
		}

		// 删除现有数据库 - Use quoted identifier
		log.Printf("Dropping existing database: %s", cfg.Database.DBName)
		dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdentifier(cfg.Database.DBName))
		if err := systemDB.Exec(dropQuery).Error; err != nil {
			return fmt.Errorf("failed to drop existing database: %w", err)
		}

		log.Printf("Database '%s' dropped successfully", cfg.Database.DBName)
	}

	// 创建新数据库 - Use quoted identifier to prevent SQL injection
	log.Printf("Creating new database: %s", cfg.Database.DBName)
	createQuery := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(cfg.Database.DBName))
	if err := systemDB.Exec(createQuery).Error; err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	log.Printf("Database '%s' created successfully", cfg.Database.DBName)

	// 关闭系统数据库连接
	sqlDB, _ := systemDB.DB()
	sqlDB.Close()

	return nil
}

func initializeRBAC() error {
	log.Println("Initializing RBAC system...")

	rbacService := services.NewRBACService(database.DB)
	if err := rbacService.InitializeDefaultRBAC(); err != nil {
		return fmt.Errorf("failed to initialize RBAC: %w", err)
	}

	log.Println("RBAC system initialized successfully!")
	return nil
}

func createDefaultAdmin() {
	db := database.DB
	rbacService := services.NewRBACService(db)

	log.Println("Creating default admin user...")

	// 创建默认管理员用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	admin := &models.User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: string(hashedPassword),
		IsActive: true,
	}

	if err := db.Create(admin).Error; err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	// 为管理员分配超级管理员角色
	var superAdminRole models.Role
	if err := db.Where("name = ?", "super-admin").First(&superAdminRole).Error; err != nil {
		log.Fatal("Failed to find super-admin role:", err)
	}

	if err := rbacService.AssignRoleToUser(admin.ID, superAdminRole.ID); err != nil {
		log.Printf("Warning: Failed to assign super-admin role to admin user: %v", err)
	} else {
		log.Println("Super-admin role assigned to admin user successfully!")
	}

	log.Println("Default admin user created successfully!")
	log.Println("Username: admin")
	log.Println("Password: admin123")
	log.Printf("Password hash: %s", string(hashedPassword))
	log.Println("Please change the password after first login!")
}

// initializeLDAPUsers 初始化LDAP用户账户
func initializeLDAPUsers(cfg *config.Config) {
	log.Println("Initializing LDAP users...")

	// 检查是否应该初始化LDAP（通过配置控制）
	if !shouldInitializeLDAP(cfg) {
		log.Println("LDAP initialization skipped (INIT_LDAP not set to true)")
		return
	}

	// 等待LDAP服务启动
	if !waitForLDAP(cfg) {
		log.Println("LDAP server not available, skipping LDAP user initialization")
		return
	}

	// 连接到LDAP服务器
	conn, err := connectToLDAP(cfg)
	if err != nil {
		log.Printf("Failed to connect to LDAP: %v", err)
		return
	}
	defer conn.Close()

	// 创建LDAP用户
	if err := createLDAPUsers(conn, cfg); err != nil {
		log.Printf("Failed to create LDAP users: %v", err)
		return
	}

	log.Println("LDAP users initialized successfully!")
	log.Printf("Created users:")
	log.Printf("  - %s (password: %s)", cfg.LDAPInit.AdminUser.UID, cfg.LDAPInit.AdminUser.Password)
	log.Printf("  - %s (password: %s)", cfg.LDAPInit.RegularUser.UID, cfg.LDAPInit.RegularUser.Password)

	// 输出LDAP配置信息
	printLDAPConfigInfo(cfg)
}

// shouldInitializeLDAP 检查是否应该初始化LDAP
func shouldInitializeLDAP(cfg *config.Config) bool {
	return cfg.LDAPInit.InitLDAP
}

// waitForLDAP 等待LDAP服务启动
func waitForLDAP(cfg *config.Config) bool {
	maxRetries := cfg.LDAPInit.RetryCount
	retryInterval := time.Duration(cfg.LDAPInit.RetryInterval) * time.Second

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			log.Printf("Waiting for LDAP server (attempt %d/%d)...", i+1, maxRetries)
			time.Sleep(retryInterval)
		}

		// 尝试连接LDAP服务器
		conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", cfg.LDAP.Server, cfg.LDAP.Port))
		if err == nil {
			conn.Close()
			log.Println("LDAP server is ready")
			return true
		}

		log.Printf("LDAP server not ready: %v", err)
	}

	log.Printf("LDAP server not available after %d attempts", maxRetries)
	return false
}

// connectToLDAP 连接到LDAP服务器
func connectToLDAP(cfg *config.Config) (*ldap.Conn, error) {
	// 连接到LDAP服务器 - 不要包含协议前缀
	address := fmt.Sprintf("%s:%d", cfg.LDAP.Server, cfg.LDAP.Port)

	log.Printf("Attempting to connect to LDAP server at %s", address)
	log.Printf("Base DN: %s", cfg.LDAP.BaseDN)

	var conn *ldap.Conn
	var err error

	// 检查是否使用SSL
	if cfg.LDAP.UseSSL {
		log.Printf("Using SSL/TLS connection")
		conn, err = ldap.DialTLS("tcp", address, &tls.Config{InsecureSkipVerify: true})
	} else {
		log.Printf("Using plain connection")
		conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server %s: %w", address, err)
	}

	// 使用管理员账户绑定
	log.Printf("Attempting to bind as: %s", cfg.LDAP.BindDN)

	err = conn.Bind(cfg.LDAP.BindDN, cfg.LDAP.BindPassword)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind to LDAP as admin (%s): %w", cfg.LDAP.BindDN, err)
	}

	log.Printf("Successfully connected and bound to LDAP server at %s", address)
	return conn, nil
}

// createLDAPUsers 创建LDAP用户
func createLDAPUsers(conn *ldap.Conn, cfg *config.Config) error {
	baseDN := cfg.LDAP.BaseDN
	peopleDN := fmt.Sprintf("ou=%s,%s", cfg.LDAPInit.PeopleOU, baseDN)
	groupsDN := fmt.Sprintf("ou=%s,%s", cfg.LDAPInit.GroupsOU, baseDN)

	log.Printf("Creating LDAP users with base DN: %s", baseDN)
	log.Printf("People DN: %s", peopleDN)
	log.Printf("Groups DN: %s", groupsDN)

	// 确保组织单位存在
	if err := ensureOrgUnitsExist(conn, cfg); err != nil {
		return fmt.Errorf("failed to create organizational units: %w", err)
	}

	// 创建用户组
	if err := ensureGroupsExist(conn, cfg, groupsDN, peopleDN); err != nil {
		return fmt.Errorf("failed to create groups: %w", err)
	}

	// 创建用户
	users := []LDAPUser{
		{
			UID:           cfg.LDAPInit.AdminUser.UID,
			CN:            cfg.LDAPInit.AdminUser.CN,
			SN:            cfg.LDAPInit.AdminUser.SN,
			GivenName:     cfg.LDAPInit.AdminUser.GivenName,
			Mail:          cfg.LDAPInit.AdminUser.Email,
			Password:      cfg.LDAPInit.AdminUser.Password,
			UIDNumber:     cfg.LDAPInit.AdminUser.UIDNumber,
			GIDNumber:     cfg.LDAPInit.AdminUser.GIDNumber,
			HomeDirectory: cfg.LDAPInit.AdminUser.HomeDirectory,
		},
		{
			UID:           cfg.LDAPInit.RegularUser.UID,
			CN:            cfg.LDAPInit.RegularUser.CN,
			SN:            cfg.LDAPInit.RegularUser.SN,
			GivenName:     cfg.LDAPInit.RegularUser.GivenName,
			Mail:          cfg.LDAPInit.RegularUser.Email,
			Password:      cfg.LDAPInit.RegularUser.Password,
			UIDNumber:     cfg.LDAPInit.RegularUser.UIDNumber,
			GIDNumber:     cfg.LDAPInit.RegularUser.GIDNumber,
			HomeDirectory: cfg.LDAPInit.RegularUser.HomeDirectory,
		},
	}

	for _, user := range users {
		if err := createLDAPUser(conn, user, peopleDN); err != nil {
			log.Printf("Warning: Failed to create user %s: %v", user.UID, err)
		} else {
			log.Printf("Created LDAP user: %s", user.UID)
		}
	}

	return nil
}

// LDAPUser LDAP用户结构
type LDAPUser struct {
	UID           string
	CN            string
	SN            string
	GivenName     string
	Mail          string
	Password      string
	UIDNumber     int
	GIDNumber     int
	HomeDirectory string
}

// ensureOrgUnitsExist 确保组织单位存在
func ensureOrgUnitsExist(conn *ldap.Conn, cfg *config.Config) error {
	baseDN := cfg.LDAP.BaseDN
	orgUnits := []struct {
		DN string
		OU string
	}{
		{fmt.Sprintf("ou=%s,%s", cfg.LDAPInit.PeopleOU, baseDN), cfg.LDAPInit.PeopleOU},
		{fmt.Sprintf("ou=%s,%s", cfg.LDAPInit.GroupsOU, baseDN), cfg.LDAPInit.GroupsOU},
	}

	for _, unit := range orgUnits {
		log.Printf("Creating organizational unit: %s", unit.DN)
		addReq := ldap.NewAddRequest(unit.DN, nil)
		addReq.Attribute("objectClass", []string{"organizationalUnit"})
		addReq.Attribute("ou", []string{unit.OU})

		err := conn.Add(addReq)
		if err != nil && !isLDAPError(err, ldap.LDAPResultEntryAlreadyExists) {
			return fmt.Errorf("failed to create OU %s: %w", unit.OU, err)
		}
		if err == nil {
			log.Printf("Successfully created OU: %s", unit.OU)
		} else {
			log.Printf("OU already exists: %s", unit.OU)
		}
	}

	return nil
}

// ensureGroupsExist 确保用户组存在
func ensureGroupsExist(conn *ldap.Conn, cfg *config.Config, groupsDN, peopleDN string) error {
	groups := []struct {
		DN      string
		CN      string
		Members []string
	}{
		{
			DN:      fmt.Sprintf("cn=%s,%s", cfg.LDAPInit.AdminGroupCN, groupsDN),
			CN:      cfg.LDAPInit.AdminGroupCN,
			Members: []string{fmt.Sprintf("uid=%s,%s", cfg.LDAPInit.AdminUser.UID, peopleDN)},
		},
		{
			DN:      fmt.Sprintf("cn=%s,%s", cfg.LDAPInit.UserGroupCN, groupsDN),
			CN:      cfg.LDAPInit.UserGroupCN,
			Members: []string{fmt.Sprintf("uid=%s,%s", cfg.LDAPInit.RegularUser.UID, peopleDN)},
		},
	}

	for _, group := range groups {
		addReq := ldap.NewAddRequest(group.DN, nil)
		addReq.Attribute("objectClass", []string{"groupOfNames"})
		addReq.Attribute("cn", []string{group.CN})
		addReq.Attribute("member", group.Members)

		err := conn.Add(addReq)
		if err != nil && !isLDAPError(err, ldap.LDAPResultEntryAlreadyExists) {
			return fmt.Errorf("failed to create group %s: %w", group.CN, err)
		}
	}

	return nil
}

// createLDAPUser 创建LDAP用户
func createLDAPUser(conn *ldap.Conn, user LDAPUser, peopleDN string) error {
	userDN := fmt.Sprintf("uid=%s,%s", user.UID, peopleDN)

	// 生成密码哈希 (SSHA)
	passwordHash, err := generateSSHAPassword(user.Password)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	addReq := ldap.NewAddRequest(userDN, nil)
	addReq.Attribute("objectClass", []string{"inetOrgPerson", "posixAccount"})
	addReq.Attribute("uid", []string{user.UID})
	addReq.Attribute("cn", []string{user.CN})
	addReq.Attribute("sn", []string{user.SN})
	addReq.Attribute("givenName", []string{user.GivenName})
	addReq.Attribute("mail", []string{user.Mail})
	addReq.Attribute("userPassword", []string{passwordHash})
	addReq.Attribute("uidNumber", []string{fmt.Sprintf("%d", user.UIDNumber)})
	addReq.Attribute("gidNumber", []string{fmt.Sprintf("%d", user.GIDNumber)})
	addReq.Attribute("homeDirectory", []string{user.HomeDirectory})

	err = conn.Add(addReq)
	if err != nil && !isLDAPError(err, ldap.LDAPResultEntryAlreadyExists) {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

// generateSSHAPassword 生成SSHA密码哈希
func generateSSHAPassword(password string) (string, error) {
	// 对于简单起见，这里使用明文密码
	// 在生产环境中应该使用更安全的哈希方法
	return password, nil
}

// isLDAPError 检查是否为特定的LDAP错误
func isLDAPError(err error, code uint16) bool {
	if ldapErr, ok := err.(*ldap.Error); ok {
		return ldapErr.ResultCode == code
	}
	return false
}

// printLDAPConfigInfo 输出LDAP配置信息
func printLDAPConfigInfo(cfg *config.Config) {
	log.Println("")
	log.Println("=== LDAP Configuration Information ===")
	log.Println("Please use the following configuration in your backend service:")
	log.Println("")
	log.Printf("LDAP Server: %s", cfg.LDAP.Server)
	log.Printf("LDAP Port: %d", cfg.LDAP.Port)
	log.Printf("Use SSL: %t", cfg.LDAP.UseSSL)
	log.Printf("Base DN: %s", cfg.LDAP.BaseDN)
	log.Printf("Admin DN: %s", cfg.LDAP.BindDN)
	log.Printf("Admin Password: %s", cfg.LDAP.BindPassword)
	log.Println("")
	log.Println("=== Organizational Units ===")
	log.Printf("People OU: ou=%s,%s", cfg.LDAPInit.PeopleOU, cfg.LDAP.BaseDN)
	log.Printf("Groups OU: ou=%s,%s", cfg.LDAPInit.GroupsOU, cfg.LDAP.BaseDN)
	log.Println("")
	log.Println("=== Groups ===")
	log.Printf("Admins Group: cn=%s,ou=%s,%s", cfg.LDAPInit.AdminGroupCN, cfg.LDAPInit.GroupsOU, cfg.LDAP.BaseDN)
	log.Printf("Users Group: cn=%s,ou=%s,%s", cfg.LDAPInit.UserGroupCN, cfg.LDAPInit.GroupsOU, cfg.LDAP.BaseDN)
	log.Println("")
	log.Println("=== User Accounts ===")
	log.Printf("Admin User DN: uid=%s,ou=%s,%s", cfg.LDAPInit.AdminUser.UID, cfg.LDAPInit.PeopleOU, cfg.LDAP.BaseDN)
	log.Printf("  - Username: %s", cfg.LDAPInit.AdminUser.UID)
	log.Printf("  - Password: %s", cfg.LDAPInit.AdminUser.Password)
	log.Printf("  - Email: %s", cfg.LDAPInit.AdminUser.Email)
	log.Printf("  - UID Number: %d", cfg.LDAPInit.AdminUser.UIDNumber)
	log.Printf("  - GID Number: %d", cfg.LDAPInit.AdminUser.GIDNumber)
	log.Printf("  - Home Directory: %s", cfg.LDAPInit.AdminUser.HomeDirectory)
	log.Println("")
	log.Printf("Regular User DN: uid=%s,ou=%s,%s", cfg.LDAPInit.RegularUser.UID, cfg.LDAPInit.PeopleOU, cfg.LDAP.BaseDN)
	log.Printf("  - Username: %s", cfg.LDAPInit.RegularUser.UID)
	log.Printf("  - Password: %s", cfg.LDAPInit.RegularUser.Password)
	log.Printf("  - Email: %s", cfg.LDAPInit.RegularUser.Email)
	log.Printf("  - UID Number: %d", cfg.LDAPInit.RegularUser.UIDNumber)
	log.Printf("  - GID Number: %d", cfg.LDAPInit.RegularUser.GIDNumber)
	log.Printf("  - Home Directory: %s", cfg.LDAPInit.RegularUser.HomeDirectory)
	log.Println("")
	log.Println("=== Search Filters (for backend configuration) ===")
	log.Printf("User Search Base: ou=%s,%s", cfg.LDAPInit.PeopleOU, cfg.LDAP.BaseDN)
	log.Println("User Search Filter: (uid={username})")
	log.Println("User Object Class: inetOrgPerson")
	log.Printf("Group Search Base: ou=%s,%s", cfg.LDAPInit.GroupsOU, cfg.LDAP.BaseDN)
	log.Println("Group Search Filter: (member={userdn})")
	log.Println("Group Object Class: groupOfNames")
	log.Println("")
	log.Println("=== Common LDAP Attributes ===")
	log.Println("Username Attribute: uid")
	log.Println("Email Attribute: mail")
	log.Println("Display Name Attribute: cn")
	log.Println("First Name Attribute: givenName")
	log.Println("Last Name Attribute: sn")
	log.Println("Group Name Attribute: cn")
	log.Println("Group Member Attribute: member")
	log.Println("")
	log.Println("=== Example Backend Configuration (JSON) ===")
	log.Printf(`{
  "ldap": {
    "enabled": true,
    "server": "%s",
    "port": %d,
    "use_ssl": %t,
    "base_dn": "%s",
    "bind_dn": "%s",
    "bind_password": "%s",
    "user_search_base": "ou=%s,%s",
    "user_search_filter": "(uid={username})",
    "user_attributes": {
      "username": "uid",
      "email": "mail",
      "display_name": "cn",
      "first_name": "givenName",
      "last_name": "sn"
    },
    "group_search_base": "ou=%s,%s",
    "group_search_filter": "(member={userdn})",
    "group_attributes": {
      "name": "cn"
    },
    "admin_group": "cn=%s,ou=%s,%s"
  }
}`, cfg.LDAP.Server, cfg.LDAP.Port, cfg.LDAP.UseSSL, cfg.LDAP.BaseDN, cfg.LDAP.BindDN, cfg.LDAP.BindPassword,
		cfg.LDAPInit.PeopleOU, cfg.LDAP.BaseDN, cfg.LDAPInit.GroupsOU, cfg.LDAP.BaseDN,
		cfg.LDAPInit.AdminGroupCN, cfg.LDAPInit.GroupsOU, cfg.LDAP.BaseDN)
	log.Println("")
	log.Println("=== Test Commands ===")
	log.Println("You can test LDAP connectivity using these commands:")
	log.Printf("ldapsearch -x -H ldap://%s:%d -D '%s' -w '%s' -b '%s' '(objectClass=*)'",
		cfg.LDAP.Server, cfg.LDAP.Port, cfg.LDAP.BindDN, cfg.LDAP.BindPassword, cfg.LDAP.BaseDN)
	log.Println("")
	log.Printf("ldapsearch -x -H ldap://%s:%d -D '%s' -w '%s' -b 'ou=%s,%s' '(uid=%s)'",
		cfg.LDAP.Server, cfg.LDAP.Port, cfg.LDAP.BindDN, cfg.LDAP.BindPassword,
		cfg.LDAPInit.PeopleOU, cfg.LDAP.BaseDN, cfg.LDAPInit.AdminUser.UID)
	log.Println("")
	log.Println("=== LDAP Configuration Complete ===")
}

// initializeDefaultAIConfigs 初始化默认AI配置
func initializeDefaultAIConfigs() {
	log.Println("=== Initializing Default AI Configurations ===")

	// 检查是否强制重新初始化
	forceReinit := getEnvCompat("FORCE_AI_REINIT", "false") == "true"

	// 检查是否已存在AI配置
	var count int64
	database.DB.Model(&models.AIAssistantConfig{}).Count(&count)

	if count > 0 && !forceReinit {
		log.Printf("AI configurations already exist (%d configs found), use FORCE_AI_REINIT=true to reinitialize", count)
		return
	}

	if forceReinit && count > 0 {
		log.Printf("Force reinitializing AI configs, clearing existing %d configurations...", count)
		// 清理现有配置
		database.DB.Exec("DELETE FROM ai_assistant_configs")
		database.DB.Exec("DELETE FROM ai_conversations")
		database.DB.Exec("DELETE FROM ai_messages")
		database.DB.Exec("DELETE FROM ai_usage_stats")
		log.Println("✓ Existing AI configurations cleared")
	}

	createdConfigs := 0

	// 从环境变量读取配置
	systemPrompt := getEnvCompat("AI_ASSISTANT_DEFAULT_SYSTEM_PROMPT", "你是一个智能的AI助手，请提供准确、有用的回答。")

	// 创建OpenAI配置
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey != "" && openaiAPIKey != "sk-test-demo-key-replace-with-real-api-key" {
		openaiConfig := &models.AIAssistantConfig{
			Name:         "默认 OpenAI GPT-4",
			Provider:     models.ProviderOpenAI,
			ModelType:    models.ModelTypeChat,
			APIKey:       openaiAPIKey,
			APIEndpoint:  getEnvCompat("OPENAI_BASE_URL", "https://api.openai.com/v1/chat/completions"),
			Model:        getEnvCompat("OPENAI_DEFAULT_MODEL", "gpt-4"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: systemPrompt,
			IsEnabled:    true,
			IsDefault:    true,
			Description:  "默认的OpenAI GPT-4模型配置",
			Category:     "通用对话",
		}

		if err := database.DB.Create(openaiConfig).Error; err != nil {
			log.Printf("Warning: Failed to create OpenAI config: %v", err)
		} else {
			log.Println("✓ Created OpenAI configuration with API key")
			createdConfigs++
		}
	} else {
		log.Println("⚠ OPENAI_API_KEY not provided or is demo key, skipping OpenAI config")
	}

	// 创建Claude配置
	claudeAPIKey := os.Getenv("CLAUDE_API_KEY")
	if claudeAPIKey != "" {
		claudeConfig := &models.AIAssistantConfig{
			Name:         "默认 Claude 3.5 Sonnet",
			Provider:     models.ProviderClaude,
			ModelType:    models.ModelTypeChat,
			APIKey:       claudeAPIKey,
			APIEndpoint:  getEnvCompat("CLAUDE_BASE_URL", "https://api.anthropic.com"),
			Model:        getEnvCompat("CLAUDE_DEFAULT_MODEL", "claude-3-5-sonnet-20241022"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是Claude，一个由Anthropic开发的AI助手。请提供有帮助、准确和诚实的回答。",
			IsEnabled:    true,
			IsDefault:    (createdConfigs == 0), // 如果没有OpenAI配置，则Claude设为默认
			Description:  "默认的Claude 3.5 Sonnet模型配置",
			Category:     "通用对话",
		}

		if err := database.DB.Create(claudeConfig).Error; err != nil {
			log.Printf("Warning: Failed to create Claude config: %v", err)
		} else {
			log.Println("✓ Created Claude configuration with API key")
			createdConfigs++
		}
	} else {
		log.Println("⚠ CLAUDE_API_KEY not provided, skipping Claude config")
	}

	// 创建其他提供商配置
	createOtherProviderConfigs(&createdConfigs, systemPrompt)

	// 初始化Backend服务相关配置
	initializeBackendConfigs()

	// 初始化SLURM服务相关配置
	initializeSlurmConfigs()

	// 初始化SaltStack服务相关配置
	initializeSaltStackConfigs()

	if createdConfigs > 0 {
		log.Printf("=== AI Configurations Initialized Successfully ===")
		log.Printf("✓ Created %d AI provider configurations", createdConfigs)
		log.Println("🌐 Access the AI Assistant Management at: /admin/ai-assistant")
	} else {
		log.Println("⚠ No AI configurations created. Please set API keys in environment variables:")
		log.Println("  - OPENAI_API_KEY for OpenAI")
		log.Println("  - CLAUDE_API_KEY for Claude")
		log.Println("  - DEEPSEEK_API_KEY for DeepSeek")
		log.Println("  - GLM_API_KEY for GLM")
		log.Println("  - QWEN_API_KEY for Qwen")
	}
}

// createOtherProviderConfigs 创建其他AI提供商的配置
func createOtherProviderConfigs(createdConfigs *int, systemPrompt string) {
	// 创建DeepSeek配置
	// 检查是否配置了 DEEPSEEK_API_KEY 环境变量
	deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY")

	// 只有配置了 DEEPSEEK_API_KEY 才创建 DeepSeek 配置
	if deepseekAPIKey != "" && deepseekAPIKey != "sk-test-demo-key-replace-with-real-api-key" {

		baseURL := getEnvCompat("DEEPSEEK_BASE_URL", "https://api.deepseek.com")

		// 创建 DeepSeek Chat 配置（非思考模式）
		chatModel := getEnvCompat("DEEPSEEK_CHAT_MODEL", "deepseek-chat")
		deepseekChatConfig := &models.AIAssistantConfig{
			Name:         "DeepSeek-V3.2-Exp (Chat)",
			Provider:     models.ProviderDeepSeek,
			ModelType:    models.ModelTypeChat,
			APIKey:       deepseekAPIKey,
			APIEndpoint:  baseURL,
			Model:        chatModel,
			MaxTokens:    8192,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是DeepSeek助手，基于DeepSeek-V3.2-Exp模型。请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "DeepSeek-V3.2-Exp 非思考模式，适合快速对话和一般任务",
			Category:     "通用对话",
		}

		if err := database.DB.Create(deepseekChatConfig).Error; err != nil {
			log.Printf("Warning: Failed to create DeepSeek Chat config: %v", err)
		} else {
			log.Println("✓ Created DeepSeek Chat (V3.2-Exp) configuration")
			*createdConfigs++
		}

		// 创建 DeepSeek Reasoner 配置（思考模式）
		reasonerModel := getEnvCompat("DEEPSEEK_REASONER_MODEL", "deepseek-reasoner")
		deepseekReasonerConfig := &models.AIAssistantConfig{
			Name:         "DeepSeek-V3.2-Exp (Reasoner)",
			Provider:     models.ProviderDeepSeek,
			ModelType:    models.ModelTypeChat,
			APIKey:       deepseekAPIKey,
			APIEndpoint:  baseURL,
			Model:        reasonerModel,
			MaxTokens:    8192,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是DeepSeek推理助手，基于DeepSeek-V3.2-Exp模型的思考模式。你会深入分析问题并提供详细的推理过程。",
			IsEnabled:    true,
			IsDefault:    false,
			Description:  "DeepSeek-V3.2-Exp 思考模式，适合复杂推理、数学问题和深度分析",
			Category:     "深度推理",
		}

		if err := database.DB.Create(deepseekReasonerConfig).Error; err != nil {
			log.Printf("Warning: Failed to create DeepSeek Reasoner config: %v", err)
		} else {
			log.Println("✓ Created DeepSeek Reasoner (V3.2-Exp) configuration")
			*createdConfigs++
		}
	} else {
		log.Println("⚠ DEEPSEEK_API_KEY not provided or is demo key, skipping DeepSeek config")
	}

	// 创建GLM配置
	if glmAPIKey := os.Getenv("GLM_API_KEY"); glmAPIKey != "" {
		glmConfig := &models.AIAssistantConfig{
			Name:         "默认 GLM-4",
			Provider:     models.ProviderCustom,
			ModelType:    models.ModelTypeChat,
			APIKey:       glmAPIKey,
			APIEndpoint:  getEnvCompat("GLM_BASE_URL", "https://open.bigmodel.cn/api/paas/v4"),
			Model:        getEnvCompat("GLM_DEFAULT_MODEL", "glm-4"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是智谱AI的GLM助手，请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "默认的智谱AI GLM-4模型配置",
			Category:     "通用对话",
		}

		if err := database.DB.Create(glmConfig).Error; err != nil {
			log.Printf("Warning: Failed to create GLM config: %v", err)
		} else {
			log.Println("✓ Created GLM configuration")
			*createdConfigs++
		}
	}

	// 创建通义千问配置
	if qwenAPIKey := os.Getenv("QWEN_API_KEY"); qwenAPIKey != "" {
		qwenConfig := &models.AIAssistantConfig{
			Name:         "默认 通义千问",
			Provider:     models.ProviderCustom,
			ModelType:    models.ModelTypeChat,
			APIKey:       qwenAPIKey,
			APIEndpoint:  getEnvCompat("QWEN_BASE_URL", "https://dashscope.aliyuncs.com/api/v1"),
			Model:        getEnvCompat("QWEN_DEFAULT_MODEL", "qwen-turbo"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是通义千问助手，请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "默认的阿里云通义千问模型配置",
			Category:     "通用对话",
		}

		if err := database.DB.Create(qwenConfig).Error; err != nil {
			log.Printf("Warning: Failed to create Qwen config: %v", err)
		} else {
			log.Println("✓ Created Qwen configuration")
			*createdConfigs++
		}
	}

	// 创建本地AI配置
	if localAIEnabled := getEnvCompat("LOCAL_AI_ENABLED", "false"); localAIEnabled == "true" {
		localConfig := &models.AIAssistantConfig{
			Name:         "本地 AI 模型",
			Provider:     models.ProviderLocal,
			ModelType:    models.ModelTypeChat,
			APIEndpoint:  getEnvCompat("LOCAL_AI_BASE_URL", "http://localhost:8080/v1"),
			Model:        getEnvCompat("LOCAL_AI_DEFAULT_MODEL", "llama2"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是一个本地部署的AI助手，请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "本地部署的AI模型配置",
			Category:     "通用对话",
		}

		if err := database.DB.Create(localConfig).Error; err != nil {
			log.Printf("Warning: Failed to create Local AI config: %v", err)
		} else {
			log.Println("✓ Created Local AI configuration")
			*createdConfigs++
		}
	}
}

// initializeBackendConfigs 初始化Backend服务配置
func initializeBackendConfigs() {
	log.Println("=== Initializing Backend Service Configurations ===")

	// 这里可以添加Backend服务特定的初始化逻辑
	// 例如：初始化缓存配置、消息队列配置等

	log.Println("✓ Backend service configurations initialized")
}

// initializeSlurmConfigs 初始化SLURM服务配置
func initializeSlurmConfigs() {
	log.Println("=== Initializing SLURM Service Configurations ===")

	// 这里可以添加SLURM服务特定的初始化逻辑
	// 例如：初始化SLURM集群配置、节点配置等

	slurmEnabled := getEnvCompat("SLURM_ENABLED", "true")
	if slurmEnabled == "true" {
		slurmCluster := getEnvCompat("SLURM_CLUSTER_NAME", "ai-infra-cluster")
		slurmController := getEnvCompat("SLURM_CONTROLLER_HOST", "slurm-master")

		log.Printf("✓ SLURM cluster: %s", slurmCluster)
		log.Printf("✓ SLURM controller: %s", slurmController)
		log.Println("✓ SLURM service configurations initialized")
	} else {
		log.Println("⚠ SLURM service disabled")
	}
}

// initializeSaltStackConfigs 初始化SaltStack服务配置
func initializeSaltStackConfigs() {
	log.Println("=== Initializing SaltStack Service Configurations ===")

	// 这里可以添加SaltStack服务特定的初始化逻辑
	// 例如：初始化Salt Master配置、Minion配置等

	saltEnabled := getEnvCompat("SALTSTACK_ENABLED", "true")
	if saltEnabled == "true" {
		saltMaster := getEnvCompat("SALTSTACK_MASTER_HOST", "saltstack")
		// 优先使用完整URL，其次按协议/主机/端口拼装，避免写死默认URL
		saltAPI := strings.TrimSpace(os.Getenv("SALTSTACK_MASTER_URL"))
		if saltAPI == "" {
			scheme := getEnvCompat("SALT_API_SCHEME", "http")
			host := getEnvCompat("SALT_MASTER_HOST", saltMaster)
			port := getEnvCompat("SALT_API_PORT", "8002")
			saltAPI = fmt.Sprintf("%s://%s:%s", scheme, host, port)
		}

		log.Printf("✓ SaltStack master: %s", saltMaster)
		log.Printf("✓ SaltStack API: %s", saltAPI)
		log.Println("✓ SaltStack service configurations initialized")
	} else {
		log.Println("⚠ SaltStack service disabled")
	}
}

// createJupyterHubDatabase 创建JupyterHub专用数据库
func createJupyterHubDatabase(cfg *config.Config) error {
	log.Println("Creating JupyterHub database...")

	// 连接到 postgres 系统数据库
	systemDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to system database: %w", err)
	}
	defer func() {
		sqlDB, _ := systemDB.DB()
		sqlDB.Close()
	}()

	// 检查JupyterHub数据库是否存在
	var exists bool
	jupyterhubDBName := "jupyterhub_db"
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = ?)"
	if err := systemDB.Raw(query, jupyterhubDBName).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check JupyterHub database existence: %w", err)
	}

	if !exists {
		// 创建JupyterHub数据库 - Use quoted identifier to prevent SQL injection
		log.Printf("Creating JupyterHub database: %s", jupyterhubDBName)
		createQuery := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(jupyterhubDBName))
		if err := systemDB.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create JupyterHub database: %w", err)
		}
		log.Printf("JupyterHub database '%s' created successfully", jupyterhubDBName)
	} else {
		log.Printf("JupyterHub database '%s' already exists", jupyterhubDBName)
	}

	return nil
}

// createGiteaDatabase ensures the Gitea role and database exist with the configured credentials
func createGiteaDatabase(cfg *config.Config) error {
	log.Println("Creating Gitea database and role...")

	// Read Gitea DB settings from env (compose passes into container)
	gUser := getEnvCompat("GITEA_DB_USER", "gitea")
	gPass := getEnvCompat("GITEA_DB_PASSWD", "gitea-password")
	gDB := getEnvCompat("GITEA_DB_NAME", "gitea")

	// Connect to system DB
	systemDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to system database: %w", err)
	}
	defer func() {
		sqlDB, _ := systemDB.DB()
		sqlDB.Close()
	}()

	// Create role if missing - simplified approach without DO block
	// First check if role exists
	var roleExists bool
	checkRoleSQL := "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = ?)"
	if err := systemDB.Raw(checkRoleSQL, gUser).Scan(&roleExists).Error; err != nil {
		return fmt.Errorf("failed to check Gitea role existence: %w", err)
	}

	if !roleExists {
		// Create role - use quoteLiteral for password
		// Note: CREATE USER doesn't support parameterized passwords
		createUserSQL := fmt.Sprintf("CREATE USER %s WITH LOGIN PASSWORD %s",
			quoteIdentifier(gUser), quoteLiteral(gPass))
		if err := systemDB.Exec(createUserSQL).Error; err != nil {
			return fmt.Errorf("failed to create Gitea role: %w", err)
		}
		log.Printf("✓ Gitea user '%s' created successfully", gUser)
	} else {
		log.Printf("✓ Gitea user '%s' already exists", gUser)
	}

	// Create DB if missing and grant
	var exists bool
	if err := systemDB.Raw("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = ?)", gDB).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check Gitea DB existence: %w", err)
	}
	if !exists {
		// Use format with %I for identifier quoting to prevent SQL injection
		createDatabaseSQL := fmt.Sprintf("CREATE DATABASE %s OWNER %s",
			quoteIdentifier(gDB), quoteIdentifier(gUser))
		if err := systemDB.Exec(createDatabaseSQL).Error; err != nil {
			return fmt.Errorf("failed to create Gitea database: %w", err)
		}
	}
	// Use quoted identifiers for grant statement
	grantSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s",
		quoteIdentifier(gDB), quoteIdentifier(gUser))
	if err := systemDB.Exec(grantSQL).Error; err != nil {
		log.Printf("Warning: failed to grant privileges on %s to %s: %v", gDB, gUser, err)
	}

	log.Println("Gitea database initialization done")
	return nil
}

// getEnvCompat reads from process env; used by init which runs inside container
func getEnvCompat(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// initializeGiteaUsers 在系统初始化阶段，确保后端用户同步到 Gitea
func initializeGiteaUsers(cfg *config.Config) {
	if !cfg.Gitea.Enabled {
		log.Println("Gitea integration disabled, skipping Gitea user initialization")
		return
	}
	if cfg.Gitea.AdminToken == "" {
		log.Println("Gitea admin token not configured, skipping Gitea user initialization")
		return
	}

	log.Println("Initializing Gitea users...")

	// 等待 Gitea HTTP 服务就绪
	if !waitForGitea(cfg, 30, 2*time.Second) {
		log.Println("Gitea not available, skip initializing Gitea users")
		return
	}

	// 调用后台服务进行一次全量同步（幂等）
	giteaSvc := services.NewGiteaService(cfg)
	created, updated, skipped, err := giteaSvc.SyncAllUsers()
	if err != nil {
		log.Printf("Warning: Gitea user sync failed: %v (created=%d updated=%d skipped=%d)", err, created, updated, skipped)
		return
	}

	log.Printf("Gitea users initialized: created=%d updated=%d skipped=%d", created, updated, skipped)
}

// waitForGitea 简单等待 Gitea 健康（GET /api/v1/version 200 即认为可用）
func waitForGitea(cfg *config.Config, maxRetries int, interval time.Duration) bool {
	base := cfg.Gitea.BaseURL
	// 去掉末尾斜杠，拼接 API 路径
	url := fmt.Sprintf("%s/api/v1/version", strings.TrimRight(base, "/"))
	client := &http.Client{Timeout: 3 * time.Second}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(interval)
		}
		req, _ := http.NewRequest("GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			// 只在前3次和每5次打印日志，避免日志刷屏
			if i < 3 || (i+1)%5 == 0 {
				log.Printf("Waiting for Gitea... (%d/%d): %v", i+1, maxRetries, err)
			}
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Println("Gitea is ready")
			return true
		}
		lastErr = fmt.Errorf("status=%d", resp.StatusCode)
		log.Printf("Gitea not ready, status=%d (%d/%d)", resp.StatusCode, i+1, maxRetries)
	}
	log.Printf("Gitea unavailable after %d retries, last error: %v", maxRetries, lastErr)
	return false
}

// createSLURMDatabase creates SLURM database and user for accounting
func createSLURMDatabase(cfg *config.Config) error {
	log.Println("Creating SLURM database and role...")

	// Read SLURM DB settings from env
	slurmUser := getEnvCompat("SLURM_DB_USER", "slurm")
	slurmPass := getEnvCompat("SLURM_DB_PASSWORD", "slurm123")
	slurmDB := getEnvCompat("SLURM_DB_NAME", "slurm_acct_db")

	log.Printf("SLURM DB settings - User: %s, DB: %s", slurmUser, slurmDB)

	// Connect to system DB
	systemDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to system database: %w", err)
	}
	defer func() {
		sqlDB, _ := systemDB.DB()
		sqlDB.Close()
	}()

	// Create SLURM role if missing - simplified approach
	log.Printf("Creating SLURM user: %s", slurmUser)
	var roleExists bool
	checkRoleSQL := "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = ?)"
	if err := systemDB.Raw(checkRoleSQL, slurmUser).Scan(&roleExists).Error; err != nil {
		return fmt.Errorf("failed to check SLURM role existence: %w", err)
	}

	if !roleExists {
		// Create role - use quoteLiteral for password
		// Note: CREATE USER doesn't support parameterized passwords
		createUserSQL := fmt.Sprintf("CREATE USER %s WITH LOGIN PASSWORD %s",
			quoteIdentifier(slurmUser), quoteLiteral(slurmPass))
		if err := systemDB.Exec(createUserSQL).Error; err != nil {
			return fmt.Errorf("failed to create SLURM role: %w", err)
		}
		log.Printf("✓ SLURM user '%s' created successfully", slurmUser)
	} else {
		log.Printf("✓ SLURM user '%s' already exists", slurmUser)
	}

	// Create SLURM DB if missing
	var exists bool
	if err := systemDB.Raw("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = ?)", slurmDB).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check SLURM DB existence: %w", err)
	}

	if !exists {
		log.Printf("Creating SLURM database: %s", slurmDB)
		// Use quoted identifiers to prevent SQL injection
		createDatabaseSQL := fmt.Sprintf("CREATE DATABASE %s OWNER %s",
			quoteIdentifier(slurmDB), quoteIdentifier(slurmUser))
		if err := systemDB.Exec(createDatabaseSQL).Error; err != nil {
			return fmt.Errorf("failed to create SLURM database: %w", err)
		}
		log.Printf("✓ SLURM database '%s' created successfully", slurmDB)
	} else {
		log.Printf("✓ SLURM database '%s' already exists", slurmDB)
	}

	// Grant all privileges to SLURM user - use quoted identifiers
	grantSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s",
		quoteIdentifier(slurmDB), quoteIdentifier(slurmUser))
	if err := systemDB.Exec(grantSQL).Error; err != nil {
		log.Printf("Warning: failed to grant privileges on %s to %s: %v", slurmDB, slurmUser, err)
	} else {
		log.Printf("✓ Granted all privileges on '%s' to '%s'", slurmDB, slurmUser)
	}

	log.Println("✓ SLURM database initialization completed!")
	return nil
}

// createNightingaleDatabase creates Nightingale database and initializes admin account using GORM
func createNightingaleDatabase(cfg *config.Config) error {
	log.Println("=== Creating Nightingale Database ===")

	// Read Nightingale DB settings from env
	nightingaleDB := getEnvCompat("NIGHTINGALE_DB_NAME", "nightingale")

	log.Printf("Nightingale DB settings - DB: %s", nightingaleDB)

	// Connect to system DB
	systemDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to system database: %w", err)
	}
	defer func() {
		sqlDB, _ := systemDB.DB()
		sqlDB.Close()
	}()

	// Check if Nightingale DB exists
	var exists bool
	if err := systemDB.Raw("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = ?)", nightingaleDB).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check Nightingale DB existence: %w", err)
	}

	var shouldInitSchema bool

	if !exists {
		log.Printf("Creating Nightingale database: %s", nightingaleDB)
		// Use quoted identifier to prevent SQL injection
		createDatabaseSQL := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(nightingaleDB))
		if err := systemDB.Exec(createDatabaseSQL).Error; err != nil {
			return fmt.Errorf("failed to create Nightingale database: %w", err)
		}
		log.Printf("✓ Nightingale database '%s' created successfully", nightingaleDB)
		shouldInitSchema = true
	} else {
		log.Printf("✓ Nightingale database '%s' already exists", nightingaleDB)

		// Check if it is in partial state (e.g. missing builtin_payloads table)
		nightingaleDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
			cfg.Database.Host,
			cfg.Database.User,
			cfg.Database.Password,
			nightingaleDB,
			cfg.Database.Port,
			cfg.Database.SSLMode,
		)
		tempDB, err := gorm.Open(postgres.Open(nightingaleDSN), &gorm.Config{})
		if err == nil {
			var tableExists bool
			// Check for builtin_payloads which is not created by GORM
			tempDB.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'builtin_payloads')").Scan(&tableExists)

			sqlDB, _ := tempDB.DB()
			sqlDB.Close()

			if !tableExists {
				log.Println("⚠ Database exists but 'builtin_payloads' table is missing. Database might be in partial state.")
				log.Println("Recreating database to ensure clean state...")

				// Terminate connections
				terminateQuery := `
					SELECT pg_terminate_backend(pid)
					FROM pg_stat_activity
					WHERE datname = ? AND pid <> pg_backend_pid()
				`
				systemDB.Exec(terminateQuery, nightingaleDB)

				// Drop and Recreate
				dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdentifier(nightingaleDB))
				if err := systemDB.Exec(dropQuery).Error; err != nil {
					return fmt.Errorf("failed to drop partial database: %w", err)
				}

				createDatabaseSQL := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(nightingaleDB))
				if err := systemDB.Exec(createDatabaseSQL).Error; err != nil {
					return fmt.Errorf("failed to recreate Nightingale database: %w", err)
				}
				shouldInitSchema = true
			} else {
				log.Println("✓ Database schema appears complete")
			}
		}
	}

	// Connect to Nightingale database
	nightingaleDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		nightingaleDB,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	nightingaleDB_conn, err := gorm.Open(postgres.Open(nightingaleDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to Nightingale database: %w", err)
	}
	defer func() {
		sqlDB, _ := nightingaleDB_conn.DB()
		sqlDB.Close()
	}()

	if shouldInitSchema {
		log.Println("Initializing Nightingale schema from SQL file...")
		if err := executeSQLFile(nightingaleDB_conn, "n9e_postgres.sql"); err != nil {
			log.Printf("Warning: Failed to execute SQL file: %v. Falling back to AutoMigrate.", err)
			// Fallback to AutoMigrate if SQL fails
			nightingaleModels := models.InitNightingaleModels()
			if err := nightingaleDB_conn.AutoMigrate(nightingaleModels...); err != nil {
				return fmt.Errorf("failed to auto migrate Nightingale models: %w", err)
			}
		} else {
			log.Println("✓ Nightingale schema initialized from SQL")
		}
	}

	// Initialize default roles
	if err := initializeNightingaleRoles(nightingaleDB_conn); err != nil {
		log.Printf("Warning: Failed to initialize Nightingale roles: %v", err)
	}

	// Initialize admin account synced from main system
	if err := initializeNightingaleAdmin(nightingaleDB_conn); err != nil {
		log.Printf("Warning: Failed to initialize Nightingale admin account: %v", err)
	}

	// Create default business group
	if err := initializeNightingaleBusiGroup(nightingaleDB_conn); err != nil {
		log.Printf("Warning: Failed to initialize Nightingale business group: %v", err)
	}

	log.Println("✓ Nightingale database initialization completed!")
	return nil
} // initializeNightingaleRoles initializes default roles in Nightingale using GORM
func initializeNightingaleRoles(db *gorm.DB) error {
	log.Println("Initializing Nightingale roles...")

	// Check if Admin role exists
	var count int64
	if err := db.Model(&models.NightingaleRole{}).Where("name = ?", "Admin").Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check role existence: %w", err)
	}

	if count == 0 {
		adminRole := &models.NightingaleRole{
			Name: "Admin",
			Note: "Administrator role with full permissions",
		}
		if err := db.Create(adminRole).Error; err != nil {
			return fmt.Errorf("failed to create Admin role: %w", err)
		}
		log.Println("✓ Admin role created")
	} else {
		log.Println("✓ Admin role already exists")
	}

	// Create Standard role
	if err := db.Model(&models.NightingaleRole{}).Where("name = ?", "Standard").Count(&count).Error; err == nil && count == 0 {
		standardRole := &models.NightingaleRole{
			Name: "Standard",
			Note: "Standard user role",
		}
		db.Create(standardRole)
		log.Println("✓ Standard role created")
	}

	// Create Guest role
	if err := db.Model(&models.NightingaleRole{}).Where("name = ?", "Guest").Count(&count).Error; err == nil && count == 0 {
		guestRole := &models.NightingaleRole{
			Name: "Guest",
			Note: "Guest user role with read-only permissions",
		}
		db.Create(guestRole)
		log.Println("✓ Guest role created")
	}

	return nil
}

// initializeNightingaleAdmin syncs admin account from main system to Nightingale using GORM
func initializeNightingaleAdmin(db *gorm.DB) error {
	log.Println("Syncing admin account from main system to Nightingale...")

	// Get admin user from main system
	var mainAdmin models.User
	if err := database.DB.Where("username = ?", "admin").First(&mainAdmin).Error; err != nil {
		log.Printf("Warning: Could not find admin user in main system: %v", err)
		log.Println("Nightingale admin will need to be created manually")
		return nil
	}

	// Check if admin already exists in Nightingale
	var nightingaleAdmin models.NightingaleUser
	result := db.Where("username = ?", mainAdmin.Username).First(&nightingaleAdmin)

	currentTime := time.Now().Unix()

	if result.Error == gorm.ErrRecordNotFound {
		// Create new admin user
		newAdmin := &models.NightingaleUser{
			Username:   mainAdmin.Username,
			Nickname:   mainAdmin.Name,
			Password:   mainAdmin.Password, // Use bcrypt hash from main system
			Email:      mainAdmin.Email,
			Roles:      "Admin",
			Contacts:   "{}",
			Maintainer: 1,
			CreateAt:   currentTime,
			CreateBy:   "system",
			UpdateAt:   currentTime,
			UpdateBy:   "system",
		}

		if err := db.Create(newAdmin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Printf("✓ Admin user '%s' created in Nightingale", mainAdmin.Username)
		log.Printf("  Email: %s", mainAdmin.Email)
		log.Println("  Password synced with main system")

		// Create default user group for admin
		if err := createNightingaleAdminGroup(db, newAdmin.ID, mainAdmin.Username); err != nil {
			log.Printf("Warning: Failed to create admin group: %v", err)
		}

	} else if result.Error != nil {
		return fmt.Errorf("failed to query admin user: %w", result.Error)
	} else {
		// Update existing admin user
		nightingaleAdmin.Password = mainAdmin.Password
		nightingaleAdmin.Email = mainAdmin.Email
		nightingaleAdmin.Nickname = mainAdmin.Name
		nightingaleAdmin.UpdateAt = currentTime
		nightingaleAdmin.UpdateBy = "system"

		if err := db.Save(&nightingaleAdmin).Error; err != nil {
			return fmt.Errorf("failed to update admin user: %w", err)
		}

		log.Printf("✓ Admin user '%s' updated in Nightingale", mainAdmin.Username)
		log.Println("  Password synced with main system")
	}

	return nil
}

// createNightingaleAdminGroup creates a default user group for admin
func createNightingaleAdminGroup(db *gorm.DB, adminUserID uint, adminUsername string) error {
	currentTime := time.Now().Unix()

	// Check if admin group already exists
	var existingGroup models.NightingaleUserGroup
	result := db.Where("name = ?", "admin-group").First(&existingGroup)

	var groupID uint
	if result.Error == gorm.ErrRecordNotFound {
		// Create admin group
		adminGroup := &models.NightingaleUserGroup{
			Name:     "admin-group",
			Note:     "Administrator group",
			CreateAt: currentTime,
			CreateBy: adminUsername,
			UpdateAt: currentTime,
			UpdateBy: adminUsername,
		}

		if err := db.Create(adminGroup).Error; err != nil {
			return fmt.Errorf("failed to create admin group: %w", err)
		}
		groupID = adminGroup.ID
		log.Println("✓ Admin group created")
	} else if result.Error != nil {
		return fmt.Errorf("failed to query admin group: %w", result.Error)
	} else {
		groupID = existingGroup.ID
		log.Println("✓ Admin group already exists")
	}

	// Check if admin is already a member
	var memberCount int64
	db.Model(&models.NightingaleUserGroupMember{}).Where("group_id = ? AND user_id = ?", groupID, adminUserID).Count(&memberCount)

	if memberCount == 0 {
		// Add admin to group
		member := &models.NightingaleUserGroupMember{
			GroupID: int64(groupID),
			UserID:  int64(adminUserID),
		}

		if err := db.Create(member).Error; err != nil {
			return fmt.Errorf("failed to add admin to group: %w", err)
		}
		log.Println("✓ Admin added to admin group")
	}

	return nil
}

// initializeNightingaleBusiGroup creates default business group
func initializeNightingaleBusiGroup(db *gorm.DB) error {
	log.Println("Initializing default business group...")

	currentTime := time.Now().Unix()

	// Check if default business group exists
	var existingGroup models.NightingaleBusiGroup
	result := db.Where("name = ?", "Default Group").First(&existingGroup)

	var groupID uint
	if result.Error == gorm.ErrRecordNotFound {
		// Create default business group
		defaultGroup := &models.NightingaleBusiGroup{
			Name:        "Default Group",
			LabelEnable: 0,
			LabelValue:  "",
			CreateAt:    currentTime,
			CreateBy:    "system",
			UpdateAt:    currentTime,
			UpdateBy:    "system",
		}

		if err := db.Create(defaultGroup).Error; err != nil {
			return fmt.Errorf("failed to create default business group: %w", err)
		}
		groupID = defaultGroup.ID
		log.Println("✓ Default business group created")
	} else if result.Error != nil {
		return fmt.Errorf("failed to query default business group: %w", result.Error)
	} else {
		groupID = existingGroup.ID
		log.Println("✓ Default business group already exists")
	}

	// Link admin group to business group with rw permission
	var adminUserGroup models.NightingaleUserGroup
	if err := db.Where("name = ?", "admin-group").First(&adminUserGroup).Error; err == nil {
		var memberCount int64
		db.Model(&models.NightingaleBusiGroupMember{}).Where("busi_group_id = ? AND user_group_id = ?", groupID, adminUserGroup.ID).Count(&memberCount)

		if memberCount == 0 {
			member := &models.NightingaleBusiGroupMember{
				BusiGroupID: int64(groupID),
				UserGroupID: int64(adminUserGroup.ID),
				PermFlag:    "rw", // read-write permission
			}

			if err := db.Create(member).Error; err != nil {
				log.Printf("Warning: Failed to link admin group to business group: %v", err)
			} else {
				log.Println("✓ Admin group linked to default business group with rw permission")
			}
		}
	}

	return nil
}

// executeSQLFile executes a SQL file
func executeSQLFile(db *gorm.DB, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	// Execute the SQL
	if err := db.Exec(string(content)).Error; err != nil {
		return err
	}
	return nil
}
