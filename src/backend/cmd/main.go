package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/aresnasa/ai-infra-matrix/src/backend/docs"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/controllers"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/handlers"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// initializeDefaultAdmin 初始化默认admin用户并关联到admin角色
func initializeDefaultAdmin(userService *services.UserService, rbacService *services.RBACService) error {
	db := database.DB

	// 检查admin用户是否已存在
	var existingAdmin models.User
	err := db.Where("username = ?", "admin").First(&existingAdmin).Error
	if err == nil {
		// admin用户已存在，检查是否已关联admin角色
		var adminRole models.Role
		if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
			logrus.WithError(err).Error("Admin role not found")
			return err
		}

		// 检查用户是否已有admin角色
		var userRole models.UserRole
		err = db.Where("user_id = ? AND role_id = ?", existingAdmin.ID, adminRole.ID).First(&userRole).Error
		if err == gorm.ErrRecordNotFound {
			// 用户没有admin角色，分配admin角色
			if err := rbacService.AssignRoleToUser(existingAdmin.ID, adminRole.ID); err != nil {
				logrus.WithError(err).Error("Failed to assign admin role to existing admin user")
				return err
			}
			logrus.Info("Admin role assigned to existing admin user")
		}

		// 设置role_template字段（只更新该字段，避免覆盖其他字段如SecondaryPassword）
		if existingAdmin.RoleTemplate == "" {
			if err := db.Model(&existingAdmin).Update("role_template", "admin").Error; err != nil {
				logrus.WithError(err).Error("Failed to update admin user role_template")
			}
		}

		logrus.Info("Admin user already exists and is properly configured")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		logrus.WithError(err).Error("Failed to check for existing admin user")
		return err
	}

	// 创建默认admin用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		logrus.WithError(err).Error("Failed to hash admin password")
		return err
	}

	admin := &models.User{
		Username:      "admin",
		Email:         "admin@example.com",
		Password:      string(hashedPassword),
		IsActive:      true,
		AuthSource:    "local",
		DashboardRole: "admin",
		RoleTemplate:  "admin",
	}

	if err := db.Create(admin).Error; err != nil {
		logrus.WithError(err).Error("Failed to create admin user")
		return err
	}

	// 获取admin角色
	var adminRole models.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		logrus.WithError(err).Error("Admin role not found")
		return err
	}

	// 为admin用户分配admin角色
	if err := rbacService.AssignRoleToUser(admin.ID, adminRole.ID); err != nil {
		logrus.WithError(err).Error("Failed to assign admin role to admin user")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"username": admin.Username,
		"email":    admin.Email,
	}).Info("Default admin user created successfully")
	logrus.Info("Default admin credentials - Username: admin, Password: admin123")
	logrus.Info("Please change the admin password after first login!")

	return nil
}

// @title AI-Infra-Matrix API
// @version 1.0
// @description 用于生成Ansible Playbook的REST API服务
// @host localhost:8082
// @BasePath /api
func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithField("requested_level", cfg.LogLevel).WithError(err).Warn("Invalid log level, falling back to info")
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// 记录日志级别设置
	logrus.WithField("log_level", level.String()).Info("Log level configured")

	// 初始化加密服务
	if err := utils.InitEncryptionService(cfg.EncryptionKey); err != nil {
		logrus.WithError(err).Fatal("Failed to initialize encryption service")
	}
	logrus.Info("Encryption service initialized successfully")

	// 连接数据库
	if err := database.Connect(cfg); err != nil {
		logrus.Fatal("Failed to connect to database:", err)
	}

	// 初始化二次认证模块
	utils.InitSecondaryAuth(database.DB)
	logrus.Info("Secondary authentication module initialized")

	// 运行数据库迁移
	if err := database.Migrate(); err != nil {
		logrus.Fatal("Failed to migrate database:", err)
	}

	// 自动迁移敏感数据（加密未加密的凭据）
	if err := database.MigrateSensitiveData(); err != nil {
		logrus.WithError(err).Warn("Failed to migrate sensitive data, continuing...")
	}

	// 初始化默认数据
	if err := database.SeedDefaultData(); err != nil {
		logrus.WithError(err).Warn("Failed to seed default data, continuing...")
	}

	// 初始化RBAC系统
	rbacService := services.NewRBACService(database.DB)
	if err := rbacService.InitializeDefaultRBAC(); err != nil {
		logrus.WithError(err).Error("Failed to initialize RBAC system")
	} else {
		logrus.Info("RBAC system initialized successfully")
	}

	// 初始化默认admin用户
	userService := services.NewUserService()
	if err := initializeDefaultAdmin(userService, rbacService); err != nil {
		logrus.WithError(err).Error("Failed to initialize default admin user")
	} else {
		logrus.Info("Default admin user initialized successfully")
	}

	// 连接Redis
	if err := cache.Connect(cfg); err != nil {
		logrus.Fatal("Failed to connect to Redis:", err)
	}

	// 初始化AI网关服务
	if err := services.InitializeAIGateway(); err != nil {
		logrus.WithError(err).Error("Failed to initialize AI Gateway")
	} else {
		logrus.Info("AI Gateway initialized successfully")
	}

	// 初始化AI默认配置
	aiService := services.NewAIService()
	if err := aiService.InitDefaultConfigs(); err != nil {
		logrus.WithError(err).Error("Failed to initialize default AI configurations")
	} else {
		logrus.Info("Default AI configurations initialized successfully")
	}

	// 初始化作业管理服务
	slurmDB := database.GetSlurmDB()
	primaryDB := database.DB
	if slurmDB == nil {
		slurmDB = primaryDB
	}
	if primaryDB == nil {
		primaryDB = slurmDB
	}

	slurmService := services.NewSlurmServiceWithStores(slurmDB, primaryDB)
	slurmService.StartAutoRegisterLoop()
	sshService := services.NewSSHService()
	cacheService := services.NewCacheService()
	jobService := services.NewJobService(database.DB, slurmService, sshService, cacheService)

	// 设置JWT密钥
	jwt.SetSecret(cfg.JWTSecret)

	// 启动Gitea后台同步（如果启用）
	if cfg.Gitea.Enabled && cfg.GiteaSync.Enabled {
		interval := time.Duration(cfg.GiteaSync.IntervalSeconds) * time.Second
		giteaSvc := services.NewGiteaService(cfg)
		logrus.WithFields(logrus.Fields{"interval": interval}).Info("Starting background Gitea sync loop")
		go func() {
			// 启动延迟，等待依赖稳定
			time.Sleep(5 * time.Second)
			for {
				if !cfg.Gitea.Enabled || !cfg.GiteaSync.Enabled {
					return
				}
				if _, _, _, err := giteaSvc.SyncAllUsers(); err != nil {
					logrus.WithError(err).Warn("Background Gitea sync failed")
				} else {
					logrus.Info("Background Gitea sync completed")
				}
				time.Sleep(interval)
			}
		}()
	}
	// duplicate import removed

	// 启动后台LDAP同步（如果启用）
	if cfg.LDAP.Enabled && cfg.LDAPSync.Enabled {
		interval := time.Duration(cfg.LDAPSync.IntervalSeconds) * time.Second
		ldapService := services.NewLDAPService(database.DB)
		ldapSync := services.NewLDAPSyncService(database.DB, ldapService, services.NewUserService(), rbacService)
		logrus.WithFields(logrus.Fields{"interval": interval}).Info("Starting background LDAP sync loop")
		go func() {
			// 启动延迟，等待依赖稳定
			time.Sleep(5 * time.Second)
			for {
				if !cfg.LDAP.Enabled || !cfg.LDAPSync.Enabled {
					return
				}
				if _, err := ldapSync.SyncLDAPUsersAndGroups(); err != nil {
					logrus.WithError(err).Warn("Background LDAP sync failed")
				} else {
					logrus.Info("Background LDAP sync completed")
				}
				time.Sleep(interval)
			}
		}()
	}

	// 创建必要的目录
	os.MkdirAll("outputs", 0755)
	os.MkdirAll("uploads", 0755)

	// 初始化Gin路由
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 最早期调试中间件 - 必须能看到这个
	r.Use(func(c *gin.Context) {
		logrus.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"origin": c.GetHeader("Origin"),
		}).Info("EARLY DEBUG: Request entering")
		c.Next()
	})

	// 添加日志中间件
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggingMiddleware())

	// 添加CORS调试中间件
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method

		logrus.WithFields(logrus.Fields{
			"method":     method,
			"origin":     origin,
			"path":       c.Request.URL.Path,
			"user_agent": c.GetHeader("User-Agent"),
		}).Info("CORS Debug: Request received")

		c.Next()

		logrus.WithFields(logrus.Fields{
			"method": method,
			"origin": origin,
			"path":   c.Request.URL.Path,
			"status": c.Writer.Status(),
		}).Info("CORS Debug: Response sent")
	})

	// 手动CORS中间件替代gin-contrib/cors
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method

		logrus.WithFields(logrus.Fields{
			"method":     method,
			"origin":     origin,
			"path":       c.Request.URL.Path,
			"user_agent": c.GetHeader("User-Agent"),
		}).Info("Manual CORS: Request received")

		// 安全的CORS配置 - 只允许特定的来源
		allowedOrigins := []string{
			"https://ai-infra-matrix.top",
			"https://www.ai-infra-matrix.top",
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		}

		isAllowedOrigin := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowedOrigin = true
				break
			}
		}

		// 设置CORS头 - 只对允许的来源设置
		if isAllowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			// 同源请求或无Origin头的请求
			c.Header("Access-Control-Allow-Origin", "https://ai-infra-matrix.top")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Authorization, X-Requested-With, Accept, Access-Control-Request-Method, Access-Control-Request-Headers, X-External-Host")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if method == "OPTIONS" {
			logrus.WithFields(logrus.Fields{
				"origin": origin,
				"path":   c.Request.URL.Path,
			}).Info("Manual CORS: Handling OPTIONS preflight request")
			c.AbortWithStatus(200)
			return
		}

		c.Next()

		logrus.WithFields(logrus.Fields{
			"method": method,
			"origin": origin,
			"path":   c.Request.URL.Path,
			"status": c.Writer.Status(),
		}).Info("Manual CORS: Response sent")
	})

	// Swagger文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 添加审计中间件（在所有API路由之前，用于记录基础设施操作）
	r.Use(middleware.AuditMiddleware(middleware.GetDefaultAuditConfig()))

	// 设置 API 路由
	setupAPIRoutes(r, cfg, jobService, sshService)

	// 启动节点指标同步服务（定期使用 Salt 命令采集 CPU/内存等指标）
	services.StartNodeMetricsSync()
	logrus.Info("NodeMetricsSync service started")

	// 启动对象存储健康检查服务（定期检查 SeaweedFS 等存储服务连接状态）
	services.StartObjectStorageHealthCheck(database.DB)
	logrus.Info("ObjectStorageHealthCheck service started")

	// 优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logrus.Info("Shutting down server...")

		// 停止节点指标同步服务
		services.StopNodeMetricsSync()
		logrus.Info("NodeMetricsSync service stopped")

		// 停止对象存储健康检查服务
		services.StopObjectStorageHealthCheck()
		logrus.Info("ObjectStorageHealthCheck service stopped")

		// 关闭AI网关服务
		if err := services.ShutdownAIGateway(); err != nil {
			logrus.Error("Error shutting down AI Gateway:", err)
		}

		// 关闭数据库连接
		if err := database.Close(); err != nil {
			logrus.Error("Error closing database:", err)
		}

		// 关闭Redis连接
		if err := cache.Close(); err != nil {
			logrus.Error("Error closing Redis:", err)
		}

		os.Exit(0)
	}()

	// 启动服务器
	logrus.WithField("port", cfg.Port).Info("Server starting...")
	logrus.WithField("port", cfg.Port).Info("Swagger documentation available at: http://localhost:" + cfg.Port + "/swagger/index.html")

	if err := r.Run(":" + cfg.Port); err != nil {
		logrus.Fatal("Failed to start server:", err)
	}
}

// setupAPIRoutes 注册所有 API 路由，保持 main 简洁并避免花括号错配
func setupAPIRoutes(r *gin.Engine, cfg *config.Config, jobService *services.JobService, sshService *services.SSHService) {
	// API路由组
	api := r.Group("/api")

	// 健康检查端点
	api.GET("/health", func(c *gin.Context) {
		// 检查数据库连接
		sqlDB, err := database.DB.DB()
		if err != nil {
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "Failed to get database instance",
				"error":   err.Error(),
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "Database connection failed",
				"error":   err.Error(),
			})
			return
		}

		// 检查Redis连接
		if err := cache.RDB.Ping(c.Request.Context()).Err(); err != nil {
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "Redis connection failed",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"status":    "healthy",
			"message":   "All services are running",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// removed early unauthenticated slurm routes; see authenticated group below
	// 鉴权路由（公开）
	userHandler := handlers.NewUserHandler(database.DB)
	// JupyterHub认证处理器（在多个地方使用）
	jupyterHubAuthHandler := handlers.NewJupyterHubAuthHandler(database.DB, cfg, cache.RDB)
	// 邀请码验证处理器（公开API）
	invitationCodePublicHandler := handlers.NewInvitationCodeHandler()

	auth := api.Group("/auth")
	{
		auth.GET("/registration-config", userHandler.GetRegistrationConfig) // 公开API：获取注册配置
		auth.POST("/register", userHandler.Register)
		auth.POST("/validate-ldap", userHandler.ValidateLDAP)
		auth.GET("/validate-invitation-code", invitationCodePublicHandler.ValidateInvitationCode) // 公开API：验证邀请码
		auth.POST("/login", userHandler.Login)
		auth.POST("/verify-2fa", userHandler.Verify2FALogin) // 2FA验证登录
		auth.POST("/logout", middleware.AuthMiddlewareWithSession(), userHandler.Logout)
		auth.POST("/refresh", userHandler.RefreshToken)
		// 兼容前端/SSO刷新端点
		auth.POST("/refresh-token", userHandler.RefreshToken)
		auth.GET("/profile", middleware.AuthMiddlewareWithSession(), userHandler.GetProfile)
		auth.GET("/me", middleware.AuthMiddlewareWithSession(), userHandler.GetProfile)
		auth.PUT("/profile", middleware.AuthMiddlewareWithSession(), userHandler.UpdateProfile)
		auth.POST("/change-password", middleware.AuthMiddlewareWithSession(), userHandler.ChangePassword)
		// 二次密码管理
		auth.POST("/secondary-password", middleware.AuthMiddlewareWithSession(), userHandler.SetSecondaryPassword)
		auth.PUT("/secondary-password", middleware.AuthMiddlewareWithSession(), userHandler.ChangeSecondaryPassword)
		auth.GET("/secondary-password/status", middleware.AuthMiddlewareWithSession(), userHandler.GetSecondaryPasswordStatus)
		// JupyterHub单点登录令牌生成
		auth.POST("/jupyterhub-token", middleware.AuthMiddlewareWithSession(), userHandler.GenerateJupyterHubToken)
		// JWT令牌验证（用于JupyterHub认证器）
		auth.POST("/verify-token", userHandler.VerifyJWT)
		// 简单令牌验证（用于SSO认证）- 使用 AuthMiddlewareWithSession 支持 Cookie 认证
		auth.GET("/verify", middleware.AuthMiddlewareWithSession(), userHandler.VerifyTokenSimple)

		// JupyterHub认证路由
		// JupyterHub令牌生成和验证
		auth.POST("/jupyterhub-login", middleware.AuthMiddlewareWithSession(), jupyterHubAuthHandler.GenerateJupyterHubLoginToken)
		auth.POST("/verify-jupyterhub-token", jupyterHubAuthHandler.VerifyJupyterHubToken)

		// JupyterHub会话管理
		auth.GET("/verify-jupyterhub-session", middleware.AuthMiddlewareWithSession(), jupyterHubAuthHandler.VerifyJupyterHubSession)
		auth.POST("/refresh-jupyterhub-token", middleware.AuthMiddlewareWithSession(), jupyterHubAuthHandler.RefreshJupyterHubToken)
	}

	// JupyterHub前端访问路由
	jupyter := api.Group("/jupyter")
	{
		// 需要认证的路由
		jupyterAuth := jupyter.Group("")
		jupyterAuth.Use(middleware.AuthMiddlewareWithSession())
		{
			jupyterAuth.POST("/access", jupyterHubAuthHandler.HandleJupyterHubAccess)
			jupyterAuth.GET("/status", jupyterHubAuthHandler.GetJupyterHubAccessStatus)
		}

		// 公共路由（用于检查服务状态）
		jupyter.GET("/health", jupyterHubAuthHandler.GetJupyterHubStatus)
	}

	// 用户管理路由（管理员）
	users := api.Group("/users")
	users.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminMiddleware())
	{
		users.GET("", userHandler.GetUsers)
		users.DELETE("/:id", userHandler.DeleteUser)
		users.POST("/:id/reset-password", userHandler.AdminResetPassword)
		users.PUT("/:id/role-template", userHandler.AdminUpdateRoleTemplate)
		users.PUT("/:id/groups", userHandler.AdminUpdateUserGroups)
		users.PUT("/:id/status", userHandler.ToggleUserStatus)
	}

	// 增强用户管理路由（管理员）
	enhancedUserController := controllers.NewEnhancedUserController(database.DB)
	enhancedUsers := api.Group("/admin/enhanced-users")
	enhancedUsers.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminMiddleware())
	{
		enhancedUsers.GET("", enhancedUserController.GetUsers)
		enhancedUsers.POST("", enhancedUserController.CreateUser)
		enhancedUsers.POST("/:id/reset-password", enhancedUserController.ResetPassword)
	}

	// 用户个人信息路由（用户自己可访问）
	userProfile := api.Group("/users")
	userProfile.Use(middleware.AuthMiddlewareWithSession())
	{
		userProfile.GET("/profile", userHandler.GetProfile)
		userProfile.PUT("/profile", userHandler.UpdateProfile)
	}

	// 用户组管理路由（管理员）
	userGroups := api.Group("/user-groups")
	userGroups.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminMiddleware())
	{
		userGroups.GET("", enhancedUserController.GetUserGroups)
		userGroups.POST("", enhancedUserController.CreateUserGroup)
		userGroups.PUT("/:id", enhancedUserController.UpdateUserGroup)
		userGroups.DELETE("/:id", enhancedUserController.DeleteUserGroup)
	}

	// 用户组成员管理路由（管理员）- 使用不同的前缀避免冲突
	groupMembers := api.Group("/group-members")
	groupMembers.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminMiddleware())
	{
		groupMembers.POST("/:groupId/:userId", enhancedUserController.AddUserToGroup)
		groupMembers.DELETE("/:groupId/:userId", enhancedUserController.RemoveUserFromGroup)
	}

	// 角色管理路由（管理员）
	roles := api.Group("/roles")
	roles.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminMiddleware())
	{
		roles.GET("", enhancedUserController.GetRoles)
	}

	// 项目路由（需要认证和RBAC权限）
	projectHandler := handlers.NewProjectHandler(database.DB)
	projects := api.Group("/projects")
	projects.Use(middleware.AuthMiddlewareWithSession())
	{
		projects.POST("", middleware.RBACMiddleware(database.DB, "projects", "create"), projectHandler.CreateProject)
		projects.GET("", middleware.RBACMiddleware(database.DB, "projects", "list"), projectHandler.GetProjects)
		projects.GET("/:id", middleware.RBACMiddleware(database.DB, "projects", "read"), projectHandler.GetProject)
		projects.PUT("/:id", middleware.RBACMiddleware(database.DB, "projects", "update"), projectHandler.UpdateProject)
		projects.DELETE("/:id", middleware.RBACMiddleware(database.DB, "projects", "delete"), projectHandler.DeleteProject)

		// 垃圾箱相关路由
		projects.PATCH("/:id/soft-delete", middleware.RBACMiddleware(database.DB, "projects", "delete"), projectHandler.SoftDeleteProject)
		projects.GET("/trash", middleware.RBACMiddleware(database.DB, "projects", "list"), projectHandler.GetDeletedProjects)
		projects.PATCH("/:id/restore", middleware.RBACMiddleware(database.DB, "projects", "update"), projectHandler.RestoreProject)
		projects.DELETE("/:id/force", middleware.AdminOnlyMiddleware(database.DB), projectHandler.ForceDeleteProject)
	}

	// 主机路由（需要认证和RBAC权限）
	hostHandler := handlers.NewHostHandler()
	hosts := api.Group("/hosts")
	hosts.Use(middleware.AuthMiddlewareWithSession())
	{
		hosts.POST("", middleware.RBACMiddleware(database.DB, "hosts", "create"), hostHandler.CreateHost)
		hosts.GET("", middleware.RBACMiddleware(database.DB, "hosts", "list"), hostHandler.GetHosts)
		hosts.PUT("/:id", middleware.RBACMiddleware(database.DB, "hosts", "update"), hostHandler.UpdateHost)
		hosts.DELETE("/:id", middleware.RBACMiddleware(database.DB, "hosts", "delete"), hostHandler.DeleteHost)
	}

	// 变量路由（需要认证和RBAC权限）
	variableHandler := handlers.NewVariableHandler()
	variables := api.Group("/variables")
	variables.Use(middleware.AuthMiddlewareWithSession())
	{
		variables.POST("", middleware.RBACMiddleware(database.DB, "variables", "create"), variableHandler.CreateVariable)
		variables.GET("", middleware.RBACMiddleware(database.DB, "variables", "list"), variableHandler.GetVariables)
		variables.PUT("/:id", middleware.RBACMiddleware(database.DB, "variables", "update"), variableHandler.UpdateVariable)
		variables.DELETE("/:id", middleware.RBACMiddleware(database.DB, "variables", "delete"), variableHandler.DeleteVariable)
	}

	// 任务路由（需要认证和RBAC权限）
	taskHandler := handlers.NewTaskHandler()
	tasks := api.Group("/tasks")
	tasks.Use(middleware.AuthMiddlewareWithSession())
	{
		tasks.POST("", middleware.RBACMiddleware(database.DB, "tasks", "create"), taskHandler.CreateTask)
		tasks.GET("", middleware.RBACMiddleware(database.DB, "tasks", "list"), taskHandler.GetTasks)
		tasks.PUT("/:id", middleware.RBACMiddleware(database.DB, "tasks", "update"), taskHandler.UpdateTask)
		tasks.DELETE("/:id", middleware.RBACMiddleware(database.DB, "tasks", "delete"), taskHandler.DeleteTask)
	}

	// Playbook路由（需要认证和RBAC权限）
	playbookHandler := handlers.NewPlaybookHandler()
	playbook := api.Group("/playbook")
	playbook.Use(middleware.AuthMiddlewareWithSession())
	{
		playbook.POST("/generate", middleware.RBACMiddleware(database.DB, "playbooks", "create"), playbookHandler.GeneratePlaybook)
		playbook.GET("/download/:id", middleware.RBACMiddleware(database.DB, "playbooks", "read"), playbookHandler.DownloadPlaybook)
		playbook.POST("/preview", middleware.RBACMiddleware(database.DB, "playbooks", "read"), playbookHandler.PreviewPlaybook)
		playbook.POST("/validate", middleware.RBACMiddleware(database.DB, "playbooks", "read"), playbookHandler.ValidatePlaybook)
		playbook.POST("/compatibility", middleware.RBACMiddleware(database.DB, "playbooks", "read"), playbookHandler.CheckCompatibility)
		playbook.POST("/package", middleware.RBACMiddleware(database.DB, "playbooks", "create"), playbookHandler.GeneratePackage)
		playbook.GET("/download-zip/*path", middleware.RBACMiddleware(database.DB, "playbooks", "read"), playbookHandler.DownloadPackage)
	}

	// RBAC路由（需要认证）
	rbacController := controllers.NewRBACController(database.DB)
	rbac := api.Group("/rbac")
	rbac.Use(middleware.AuthMiddlewareWithSession())
	{
		// 权限检查
		rbac.POST("/check-permission", rbacController.CheckPermission)

		// 角色管理
		rbac.POST("/roles", rbacController.CreateRole)
		rbac.GET("/roles", rbacController.GetRoles)
		rbac.GET("/roles/:id", rbacController.GetRole)
		rbac.PUT("/roles/:id", rbacController.UpdateRole)
		rbac.DELETE("/roles/:id", rbacController.DeleteRole)

		// 用户组管理
		rbac.POST("/groups", rbacController.CreateUserGroup)
		rbac.GET("/groups", rbacController.GetUserGroups)
		rbac.POST("/groups/:group_id/users/:user_id", rbacController.AddUserToGroup)
		rbac.DELETE("/groups/:group_id/users/:user_id", rbacController.RemoveUserFromGroup)

		// 权限管理
		rbac.POST("/permissions", rbacController.CreatePermission)
		rbac.GET("/permissions", rbacController.GetPermissions)

		// 角色分配
		rbac.POST("/assign-role", rbacController.AssignRole)
		rbac.DELETE("/revoke-role", rbacController.RevokeRole)

		// 角色模板管理
		rbac.GET("/role-templates", rbacController.ListRoleTemplates)
		rbac.GET("/role-templates/:id", rbacController.GetRoleTemplate)
		rbac.POST("/role-templates", rbacController.CreateRoleTemplate)
		rbac.PUT("/role-templates/:id", rbacController.UpdateRoleTemplate)
		rbac.DELETE("/role-templates/:id", rbacController.DeleteRoleTemplate)
		rbac.POST("/role-templates/sync", rbacController.SyncRoleTemplates)

		// 资源和操作列表（用于前端配置）
		rbac.GET("/resources", rbacController.GetAvailableResources)
		rbac.GET("/verbs", rbacController.GetAvailableVerbs)
	}

	// 权限审批路由（需要认证）
	approvalRbacService := services.NewRBACService(database.DB)
	approvalService := services.NewPermissionApprovalService(database.DB, approvalRbacService)
	approvalController := controllers.NewPermissionApprovalController(approvalService)
	approvals := api.Group("/approvals")
	approvals.Use(middleware.AuthMiddlewareWithSession())
	{
		// 模块和权限信息
		approvals.GET("/modules", approvalController.GetAvailableModules)
		approvals.GET("/verbs", approvalController.GetAvailableVerbs)

		// 权限申请管理
		approvals.POST("/requests", approvalController.CreatePermissionRequest)
		approvals.GET("/requests", approvalController.ListPermissionRequests)
		approvals.GET("/requests/:id", approvalController.GetPermissionRequest)
		approvals.POST("/requests/:id/approve", approvalController.ApprovePermissionRequest)
		approvals.POST("/requests/:id/cancel", approvalController.CancelPermissionRequest)

		// 权限授权管理
		approvals.POST("/grants", approvalController.GrantPermission)
		approvals.POST("/grants/revoke", approvalController.RevokePermission)
		approvals.GET("/grants", approvalController.GetUserGrants)
		approvals.GET("/my-grants", approvalController.GetMyGrants)

		// 审批规则管理
		approvals.POST("/rules", approvalController.CreateApprovalRule)
		approvals.GET("/rules", approvalController.GetApprovalRules)
		approvals.PUT("/rules/:id", approvalController.UpdateApprovalRule)
		approvals.DELETE("/rules/:id", approvalController.DeleteApprovalRule)

		// 统计和检查
		approvals.GET("/stats", approvalController.GetStats)
		approvals.GET("/check", approvalController.CheckModulePermission)
	}

	// 管理员路由（需要管理员权限）
	adminController := controllers.NewAdminController(database.DB)
	loggingController := controllers.NewLoggingController()
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminOnlyMiddleware(database.DB))
	{
		// 手动触发 Gitea 用户同步
		admin.POST("/sync-gitea-users", func(c *gin.Context) {
			if !cfg.Gitea.Enabled {
				c.JSON(400, gin.H{"status": "disabled", "message": "Gitea integration disabled"})
				return
			}
			giteaSvc := services.NewGiteaService(cfg)
			created, updated, skipped, err := giteaSvc.SyncAllUsers()
			if err != nil {
				c.JSON(500, gin.H{"status": "error", "error": err.Error(), "created": created, "updated": updated, "skipped": skipped})
				return
			}
			c.JSON(200, gin.H{"status": "ok", "created": created, "updated": updated, "skipped": skipped})
		})
		// 用户管理
		admin.GET("/users", adminController.GetAllUsers)
		admin.GET("/users/:id", adminController.GetUserDetail)
		admin.GET("/users/:id/auth-source", adminController.GetUserWithAuthSource)
		admin.PUT("/users/:id/status", adminController.UpdateUserStatus)
		admin.PUT("/users/:id/status-enhanced", adminController.UpdateUserStatusEnhanced)
		admin.DELETE("/users/:id", adminController.DeleteUser)

		// 注册审批管理
		admin.GET("/approvals/pending", userHandler.GetPendingApprovals)
		admin.POST("/approvals/:id/approve", userHandler.ApproveRegistration)
		admin.POST("/approvals/:id/reject", userHandler.RejectRegistration)

		// 用户模块权限管理
		admin.POST("/users/:id/modules", userHandler.GrantUserModules)

		// 项目管理
		admin.GET("/projects", adminController.GetAllProjects)
		admin.GET("/projects/:id", adminController.GetProjectDetail)
		admin.PUT("/projects/:id/transfer", adminController.TransferProject)

		// 回收站管理
		admin.GET("/projects/trash", adminController.GetProjectsTrash)
		admin.PATCH("/projects/:id/restore", adminController.RestoreProject)
		admin.DELETE("/projects/:id/force-delete", adminController.ForceDeleteProject)
		admin.DELETE("/projects/trash/clear", adminController.ClearTrash)

		// LDAP管理相关接口
		admin.GET("/ldap/config", adminController.GetLDAPConfig)
		admin.PUT("/ldap/config", adminController.UpdateLDAPConfig)
		admin.POST("/ldap/test", adminController.TestLDAPConnection)
		admin.GET("/ldap/users", adminController.GetLDAPUsers)

		// LDAP同步相关接口
		admin.POST("/ldap/sync", adminController.SyncLDAPUsers)
		admin.GET("/ldap/sync/:sync_id/status", adminController.GetLDAPSyncStatus)
		admin.GET("/ldap/sync/history", adminController.GetLDAPSyncHistory) // 系统统计
		admin.GET("/stats", adminController.GetSystemStats)
		admin.GET("/user-stats", adminController.GetUserStatistics)

		// RBAC初始化
		admin.POST("/rbac/initialize", adminController.InitializeRBAC)

		// 邀请码管理
		invitationCodeHandler := handlers.NewInvitationCodeHandler()
		invitationCodes := admin.Group("/invitation-codes")
		{
			invitationCodes.POST("", invitationCodeHandler.CreateInvitationCode)
			invitationCodes.GET("", invitationCodeHandler.ListInvitationCodes)
			invitationCodes.GET("/statistics", invitationCodeHandler.GetInvitationCodeStatistics)
			invitationCodes.GET("/:id", invitationCodeHandler.GetInvitationCode)
			invitationCodes.POST("/:id/disable", invitationCodeHandler.DisableInvitationCode)
			invitationCodes.POST("/:id/enable", invitationCodeHandler.EnableInvitationCode)
			invitationCodes.DELETE("/:id", invitationCodeHandler.DeleteInvitationCode)
		}

		// 日志级别管理
		logging := admin.Group("/logging")
		{
			logging.GET("/level", loggingController.GetLogLevel)
			logging.POST("/level", loggingController.SetLogLevel)
			logging.POST("/test", loggingController.TestLogLevels)
			logging.GET("/info", loggingController.GetLoggingInfo)
		}
	}

	// Kubernetes 集群管理路由（需要认证和RBAC权限）
	k8sController := controllers.NewKubernetesController()
	k8s := api.Group("/kubernetes")
	k8s.Use(middleware.AuthMiddlewareWithSession())
	{
		k8s.GET("/clusters", k8sController.ListClusters)
		k8s.POST("/clusters", k8sController.CreateCluster)
		k8s.PUT("/clusters/:id", k8sController.UpdateCluster)
		k8s.DELETE("/clusters/:id", k8sController.DeleteCluster)
		k8s.POST("/clusters/:id/test", k8sController.TestConnection)
		k8s.GET("/clusters/:id/info", k8sController.GetClusterInfo)

		// 通用资源发现与CRUD接口
		kres := controllers.NewKubernetesResourcesController()
		// 资源发现与命名空间列表
		k8s.GET("/clusters/:id/discovery", kres.DiscoverResources)
		k8s.GET("/clusters/:id/version", kres.GetClusterVersion)            // 新增：获取集群版本
		k8s.GET("/clusters/:id/enhanced-discovery", kres.EnhancedDiscovery) // 新增：增强发现（含CRD）
		k8s.GET("/clusters/:id/namespaces", kres.ListNamespaces)
		// 命名空间内资源
		k8s.GET("/clusters/:id/namespaces/:namespace/resources/:resource", kres.ListResources)
		k8s.GET("/clusters/:id/namespaces/:namespace/resources/:resource/:name", kres.GetResource)
		k8s.POST("/clusters/:id/namespaces/:namespace/resources/:resource", kres.CreateResource)
		k8s.PUT("/clusters/:id/namespaces/:namespace/resources/:resource/:name", kres.UpdateResource)
		k8s.PATCH("/clusters/:id/namespaces/:namespace/resources/:resource/:name", kres.PatchResource)
		k8s.DELETE("/clusters/:id/namespaces/:namespace/resources/:resource/:name", kres.DeleteResource)
		// 集群级资源
		k8s.GET("/clusters/:id/cluster-resources/:resource", kres.ListClusterResources)
		k8s.GET("/clusters/:id/cluster-resources/:resource/:name", kres.GetClusterResource)
		k8s.POST("/clusters/:id/cluster-resources/:resource", kres.CreateClusterResource)
		k8s.PUT("/clusters/:id/cluster-resources/:resource/:name", kres.UpdateClusterResource)
		k8s.PATCH("/clusters/:id/cluster-resources/:resource/:name", kres.PatchClusterResource)
		k8s.DELETE("/clusters/:id/cluster-resources/:resource/:name", kres.DeleteClusterResource)
		// 批量并发查询
		k8s.GET("/clusters/:id/namespaces/:namespace/resources:batch", kres.BatchListResources)
	}

	// Ansible 执行管理路由（需要认证和RBAC权限）
	ansibleController := controllers.NewAnsibleController()
	ansible := api.Group("/ansible")
	ansible.Use(middleware.AuthMiddlewareWithSession())
	{
		ansible.POST("/execute", middleware.RBACMiddleware(database.DB, "projects", "execute"), ansibleController.ExecutePlaybook)
		ansible.POST("/dry-run", middleware.RBACMiddleware(database.DB, "projects", "execute"), ansibleController.DryRunPlaybook)
		ansible.GET("/execution/:id/status", middleware.RBACMiddleware(database.DB, "projects", "read"), ansibleController.GetExecutionStatus)
		ansible.GET("/execution/:id/logs", middleware.RBACMiddleware(database.DB, "projects", "read"), ansibleController.GetExecutionLogs)
		ansible.POST("/execution/:id/cancel", middleware.RBACMiddleware(database.DB, "projects", "execute"), ansibleController.CancelExecution)
		ansible.GET("/executions", middleware.RBACMiddleware(database.DB, "projects", "list"), ansibleController.ListExecutions)
	}

	// JupyterHub 路由（需要认证和RBAC权限）
	jupyterHubController := controllers.NewJupyterHubController()
	jupyterHubController.RegisterRoutes(api)

	// JupyterLab 模板路由（需要认证）
	jupyterLabTemplateController := controllers.NewJupyterLabTemplateController()
	jupyterLabTemplateController.RegisterRoutes(api)

	// 添加JupyterHub管理页面的静态路由（需要认证）
	r.GET("/admin/jupyterhub", middleware.AuthMiddlewareWithSession(), func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JupyterHub 管理中心</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { border-bottom: 1px solid #eee; padding-bottom: 20px; margin-bottom: 20px; }
        .status { display: flex; gap: 20px; margin-bottom: 20px; }
        .status-card { flex: 1; padding: 15px; background: #f8f9fa; border-radius: 6px; text-align: center; }
        .iframe-container { height: 600px; border: 1px solid #ddd; border-radius: 6px; overflow: hidden; }
        iframe { width: 100%; height: 100%; border: none; }
        .error { color: #dc3545; padding: 10px; background: #f8d7da; border: 1px solid #f5c6cb; border-radius: 4px; margin-bottom: 20px; }
        .loading { text-align: center; padding: 50px; color: #666; }
        .btn { display: inline-block; padding: 8px 16px; margin: 5px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; }
        .btn:hover { background: #0056b3; }
        .btn-secondary { background: #6c757d; }
        .btn-secondary:hover { background: #545b62; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>JupyterHub 管理中心</h1>
            <p>管理和监控JupyterHub实例</p>
            <div>
                <button class="btn" onclick="refreshStatus()">刷新状态</button>
                <button class="btn btn-secondary" onclick="toggleIframe()">切换显示模式</button>
                <a href="/jupyter/" target="_blank" class="btn">访问JupyterHub</a>
            </div>
        </div>
        
        <div class="status">
            <div class="status-card">
                <h3>服务状态</h3>
                <div id="service-status">检查中...</div>
            </div>
            <div class="status-card">
                <h3>在线用户</h3>
                <div id="online-users">--</div>
            </div>
            <div class="status-card">
                <h3>运行中服务器</h3>
                <div id="running-servers">--</div>
            </div>
            <div class="status-card">
                <h3>内存使用</h3>
                <div id="memory-usage">--</div>
            </div>
        </div>
        
        <div id="error-message" class="error" style="display: none;"></div>
        
        <div class="iframe-container" id="iframe-container">
            <div class="loading" id="loading">正在加载JupyterHub管理界面...</div>
            <iframe id="jupyterhub-iframe" style="display: none;"></iframe>
        </div>
        
        <div style="margin-top: 20px; padding: 15px; background: #e9ecef; border-radius: 6px; font-size: 14px;">
            <h4>使用说明：</h4>
            <ul>
                <li>如果iframe无法加载，请点击"访问JupyterHub"按钮在新窗口中打开</li>
                <li>确保JupyterHub服务正在运行并且网络连接正常</li>
                <li>管理员功能需要在JupyterHub中配置相应的权限</li>
                <li>页面每30秒自动刷新状态信息</li>
            </ul>
        </div>
    </div>
    
    <script>
        let iframeVisible = true;
        
        // 检查JupyterHub状态
        async function checkStatus() {
            try {
                const response = await fetch('/api/jupyterhub/status');
                const data = await response.json();
                
                document.getElementById('service-status').innerHTML = 
                    data.running ? '<span style="color: green;">✅ 运行中</span>' : '<span style="color: red;">❌ 已停止</span>';
                document.getElementById('online-users').textContent = data.users_online || '0';
                document.getElementById('running-servers').textContent = data.servers_running || '0';
                
                if (data.used_memory_gb && data.total_memory_gb) {
                    const memUsage = Math.round((data.used_memory_gb / data.total_memory_gb) * 100);
                    document.getElementById('memory-usage').innerHTML = 
                        data.used_memory_gb + 'GB/' + data.total_memory_gb + 'GB (' + memUsage + '%)';
                } else {
                    document.getElementById('memory-usage').textContent = '--';
                }
                
                if (data.running && iframeVisible) {
                    loadJupyterHub();
                } else if (!data.running) {
                    showError('JupyterHub服务未运行，请检查服务状态');
                }
            } catch (error) {
                console.error('获取状态失败:', error);
                document.getElementById('service-status').innerHTML = '<span style="color: orange;">⚠️ 连接失败</span>';
                // 即使状态检查失败，也尝试加载界面
                if (iframeVisible) {
                    loadJupyterHub();
                }
            }
        }
        
        function loadJupyterHub() {
            const iframe = document.getElementById('jupyterhub-iframe');
            const loading = document.getElementById('loading');
            
            if (!iframe.src) {
                iframe.onload = function() {
                    loading.style.display = 'none';
                    iframe.style.display = 'block';
                };
                
                iframe.onerror = function() {
                    showError('无法加载JupyterHub管理界面，请确认JupyterHub服务正常运行');
                };
                
                // 尝试多个可能的路径
                const paths = ['/hub/admin', '/jupyter/hub/admin', '/admin', '/hub', '/jupyter/'];
                let currentPathIndex = 0;
                
                function tryNextPath() {
                    if (currentPathIndex < paths.length) {
                        console.log('尝试路径:', paths[currentPathIndex]);
                        iframe.src = paths[currentPathIndex];
                        currentPathIndex++;
                        
                        // 3秒后尝试下一个路径（如果当前路径加载失败）
                        setTimeout(() => {
                            if (iframe.style.display === 'none') {
                                tryNextPath();
                            }
                        }, 3000);
                    } else {
                        showError('尝试了所有可能的JupyterHub路径，但都无法访问。请检查JupyterHub配置，或点击"访问JupyterHub"按钮在新窗口中访问。');
                    }
                }
                
                tryNextPath();
            }
        }
        
        function showError(message) {
            const errorDiv = document.getElementById('error-message');
            const loading = document.getElementById('loading');
            
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
            loading.style.display = 'none';
        }
        
        function refreshStatus() {
            document.getElementById('error-message').style.display = 'none';
            checkStatus();
        }
        
        function toggleIframe() {
            const container = document.getElementById('iframe-container');
            iframeVisible = !iframeVisible;
            
            if (iframeVisible) {
                container.style.display = 'block';
                loadJupyterHub();
            } else {
                container.style.display = 'none';
            }
        }
        
        // 页面加载时检查状态
        checkStatus();
        
        // 每30秒更新一次状态
        setInterval(checkStatus, 30000);
    </script>
</body>
</html>
		`)
	})

	// Slurm 路由（需要认证）
	slurmController := controllers.NewSlurmController()
	slurm := api.Group("/slurm")
	slurm.Use(middleware.AuthMiddlewareWithSession())
	{
		// 基础信息
		slurm.GET("/summary", slurmController.GetSummary)
		slurm.GET("/nodes", slurmController.GetNodes)
		slurm.GET("/jobs", slurmController.GetJobs)
		slurm.GET("/partitions", slurmController.GetPartitions)

		// 节点管理
		slurm.POST("/nodes/manage", slurmController.ManageNodes)
		slurm.POST("/nodes/health-check", slurmController.HealthCheckNode) // 节点健康检测

		// 作业管理
		slurm.POST("/jobs/manage", slurmController.ManageJobs)

		// SLURM运维命令
		slurm.POST("/exec", slurmController.ExecuteSlurmCommand)
		slurm.GET("/diagnostics", slurmController.GetSlurmDiagnostics)

		// 仓库与节点初始化
		slurm.POST("/repo/setup", slurmController.SetupRepo)
		slurm.POST("/init-nodes", slurmController.InitNodes)
		// 异步版本 + 进度流
		slurm.POST("/repo/setup/async", slurmController.SetupRepoAsync)
		slurm.POST("/init-nodes/async", slurmController.InitNodesAsync)
		slurm.GET("/progress/:opId", slurmController.GetProgress)
		slurm.GET("/progress/:opId/stream", slurmController.StreamProgress)
		slurm.GET("/tasks", slurmController.GetTasks)
		slurm.GET("/tasks/statistics", slurmController.GetTaskStatistics)
		slurm.GET("/tasks/:task_id", slurmController.GetTaskDetail)
		slurm.POST("/tasks/:task_id/cancel", slurmController.CancelTask)
		slurm.POST("/tasks/:task_id/retry", slurmController.RetryTask)
		slurm.DELETE("/tasks/:task_id", slurmController.DeleteTask)

		// 扩缩容相关路由
		slurm.GET("/scaling/status", slurmController.GetScalingStatus)
		slurm.POST("/scaling/scale-up", slurmController.ScaleUp)
		slurm.POST("/scaling/scale-up/async", slurmController.ScaleUpAsync)
		slurm.POST("/scaling/scale-down", slurmController.ScaleDown)

		// 基于REST API的扩缩容路由
		slurm.POST("/scaling/scale-up-api", slurmController.ScaleUpViaAPI)
		slurm.POST("/scaling/scale-down-api", slurmController.ScaleDownViaAPI)
		slurm.POST("/reload-config", slurmController.ReloadSlurmConfig)

		// 节点安装路由
		slurm.POST("/nodes/install", slurmController.InstallSlurmNode)
		slurm.POST("/nodes/batch-install", slurmController.BatchInstallSlurmNodes)

		// 节点配置管理路由
		slurm.POST("/nodes/:nodeName/validate-config", slurmController.ValidateNodeConfig)
		slurm.POST("/nodes/:nodeName/sync-config", slurmController.SyncNodeConfig)
		slurm.POST("/nodes/sync-all-configs", slurmController.SyncAllNodesConfig)

		// SaltStack 集成路由
		slurm.GET("/saltstack/integration", slurmController.GetSaltStackIntegration)
		slurm.POST("/saltstack/deploy-minion", slurmController.DeploySaltMinion)
		slurm.POST("/ssh/test-connection", slurmController.TestSSHConnection)
		slurm.POST("/ssh/test-batch", slurmController.TestBatchSSHConnection) // 批量测试
		slurm.POST("/hosts/initialize", slurmController.InitializeHosts)
		slurm.POST("/saltstack/execute/async", slurmController.ExecuteSaltCommandAsync)
		slurm.POST("/saltstack/execute", slurmController.ExecuteSaltCommand)
		slurm.GET("/saltstack/jobs", slurmController.GetSaltJobs)

		// 节点模板路由
		slurm.GET("/node-templates", slurmController.GetNodeTemplates)
		slurm.POST("/node-templates", slurmController.CreateNodeTemplate)
		slurm.PUT("/node-templates/:id", slurmController.UpdateNodeTemplate)
		slurm.DELETE("/node-templates/:id", slurmController.DeleteNodeTemplate)

		// 安装包相关路由
		slurm.POST("/install-packages", slurmController.InstallPackages)
		slurm.POST("/install-test-nodes", slurmController.InstallTestNodes)
		slurm.GET("/installation-tasks", slurmController.GetInstallationTasks)
		slurm.GET("/installation-tasks/:id", slurmController.GetInstallationTask)
	}

	// SaltStack 客户端管理路由（需要认证）
	saltStackClientController := controllers.NewSaltStackClientController()
	saltStackClientController.RegisterRoutes(api)

	// Salt Master 公钥安全分发路由（部分端点无需认证，使用一次性令牌）
	saltKeyHandler := handlers.NewSaltKeyHandler()
	saltKeyHandler.RegisterRoutes(api)

	// SLURM 集群管理路由（需要认证）
	slurmClusterController := controllers.NewSlurmClusterController(database.GetSlurmDB())
	slurmClusterController.RegisterRoutes(api)

	// SLURM 节点扩容管理路由（需要认证）
	slurmClusterService := services.NewSlurmClusterService(database.GetSlurmDB())
	saltStackService := services.NewSaltStackService()
	slurmNodeScaleController := controllers.NewSlurmNodeScaleController(slurmClusterService, saltStackService)
	slurmNodeScaleController.RegisterRoutes(api)

	// 作业管理路由（需要认证）
	jobController := controllers.NewJobController(jobService)
	jobs := api.Group("/jobs")
	jobs.Use(middleware.AuthMiddlewareWithSession())
	{
		jobs.GET("", jobController.ListJobs)
		jobs.POST("", jobController.SubmitJob)
		jobs.POST("/async", jobController.SubmitJobAsync)
		jobs.GET("/:jobId", jobController.GetJobDetail)
		jobs.GET("/:jobId/status", jobController.GetJobStatus)
		jobs.POST("/:jobId/cancel", jobController.CancelJob)
		jobs.GET("/:jobId/output", jobController.GetJobOutput)
		jobs.GET("/clusters", jobController.ListClusters)
	}

	// 初始化 SaltStack 作业持久化服务
	saltJobService := services.NewSaltJobService(database.DB, cache.RDB)
	_ = saltJobService // 服务会自动注册为单例，供 handler 使用

	// SaltStack 管理路由（需要认证）
	saltStackHandler := handlers.NewSaltStackHandler(cfg, cache.RDB)

	// 节点指标回调 API（使用 API Token 认证，允许节点直接上报）
	// 这个路由不使用 session 认证，而是使用 X-API-Token 头进行认证
	api.POST("/saltstack/node-metrics/callback", saltStackHandler.NodeMetricsCallback)

	saltstack := api.Group("/saltstack")
	saltstack.Use(middleware.AuthMiddlewareWithSession())
	{
		saltstack.GET("/status", saltStackHandler.GetSaltStackStatus)
		saltstack.GET("/minions", saltStackHandler.GetSaltMinions)
		saltstack.GET("/minions/:minionId/details", saltStackHandler.GetMinionDetails)
		saltstack.GET("/jobs", saltStackHandler.GetSaltJobs)
		saltstack.GET("/jobs/:jid", saltStackHandler.GetSaltJobDetail) // 获取单个作业详情（优先数据库，回退Salt API）
		saltstack.POST("/execute", saltStackHandler.ExecuteSaltCommand)
		// 自定义脚本执行（异步）+ 进度
		saltstack.POST("/execute-custom/async", saltStackHandler.ExecuteCustomCommandAsync)
		saltstack.GET("/progress/:opId", saltStackHandler.GetProgress)
		saltstack.GET("/progress/:opId/stream", saltStackHandler.StreamProgress)
		// 连接性调试端点（仅限已登录用户调用，用于排查Salt API问题）
		saltstack.GET("/_debug", saltStackHandler.DebugSaltConnectivity)

		// Minion 分组管理
		saltstack.GET("/groups", saltStackHandler.ListMinionGroups)
		saltstack.POST("/groups", saltStackHandler.CreateMinionGroup)
		saltstack.PUT("/groups/:id", saltStackHandler.UpdateMinionGroup)
		saltstack.DELETE("/groups/:id", saltStackHandler.DeleteMinionGroup)
		saltstack.GET("/groups/:id/minions", saltStackHandler.GetGroupMinions)
		saltstack.POST("/minions/set-group", saltStackHandler.SetMinionGroup)
		saltstack.POST("/minions/batch-set-groups", saltStackHandler.BatchSetMinionGroups)
		// 批量为 Minion 安装 Categraf
		saltstack.POST("/minions/install-categraf", saltStackHandler.InstallCategrafOnMinions)
		saltstack.GET("/minions/install-categraf/:task_id/stream", saltStackHandler.CategrafInstallStream)
		// 节点指标采集（管理接口需要认证，回调接口在上面单独注册无需认证）
		saltstack.GET("/node-metrics", saltStackHandler.GetNodeMetrics)
		saltstack.GET("/node-metrics/summary", saltStackHandler.GetNodeMetricsSummary)
		saltstack.POST("/node-metrics/deploy", saltStackHandler.DeployNodeMetricsState)
		saltstack.POST("/node-metrics/trigger", saltStackHandler.TriggerMetricsCollection)
		// IB 端口忽略管理和告警
		saltstack.GET("/ib-ignores", saltStackHandler.GetIBPortIgnores)
		saltstack.POST("/ib-ignores", saltStackHandler.AddIBPortIgnore)
		saltstack.DELETE("/ib-ignores/:minion_id/:port_name", saltStackHandler.RemoveIBPortIgnore)
		saltstack.GET("/ib-alerts", saltStackHandler.GetIBPortAlerts)
		// 作业历史配置和管理（持久化到数据库）
		saltstack.GET("/jobs/config", saltStackHandler.GetSaltJobConfig)
		saltstack.PUT("/jobs/config", saltStackHandler.UpdateSaltJobConfig)
		saltstack.GET("/jobs/history", saltStackHandler.GetSaltJobHistory)
		saltstack.GET("/jobs/by-task/:task_id", saltStackHandler.GetSaltJobByTaskID)
		saltstack.POST("/jobs/cleanup", saltStackHandler.TriggerJobCleanup)
	}

	// 仪表板统计路由（需要认证）
	dashboard := api.Group("/dashboard")
	dashboard.Use(middleware.AuthMiddlewareWithSession())
	{
		dashboard.GET("/stats", jobController.GetDashboardStats)
	}

	// 文件浏览与传输（需要认证）
	filesSvc := services.NewFilesService(database.DB, sshService)
	filesCtrl := controllers.NewFilesController(filesSvc)
	files := api.Group("/files")
	files.Use(middleware.AuthMiddlewareWithSession())
	{
		// 列表: GET /api/files?cluster=...&path=/path
		files.GET("", filesCtrl.List)
		// 下载: GET /api/files/download?cluster=...&path=/file
		files.GET("/download", filesCtrl.Download)
		// 上传: POST multipart /api/files/upload (cluster, path, file)
		files.POST("/upload", filesCtrl.Upload)
	}

	// 作业模板管理路由（需要认证）
	jobTemplateController := controllers.NewJobTemplateController(database.DB)
	jobTemplateController.RegisterRoutes(api)

	// AI助手管理路由（需要认证）
	aiAssistantController := controllers.NewAIAssistantController()
	ai := api.Group("/ai")
	ai.Use(middleware.AuthMiddlewareWithSession())
	{
		// AI配置管理
		ai.POST("/configs", aiAssistantController.CreateConfig)
		ai.GET("/configs", aiAssistantController.ListConfigs)
		ai.GET("/configs/:id", aiAssistantController.GetConfig)
		ai.PUT("/configs/:id", aiAssistantController.UpdateConfig)
		ai.DELETE("/configs/:id", aiAssistantController.DeleteConfig)

		// 对话管理
		ai.POST("/conversations", aiAssistantController.CreateConversation)
		ai.GET("/conversations", aiAssistantController.ListConversations)
		ai.GET("/conversations/:id", aiAssistantController.GetConversation)
		ai.DELETE("/conversations/:id", aiAssistantController.DeleteConversation)
		ai.POST("/conversations/:id/stop", aiAssistantController.StopConversation)
		ai.POST("/conversations/:id/resume", aiAssistantController.ResumeConversation)

		// 消息管理
		ai.POST("/conversations/:id/messages", aiAssistantController.SendMessage)
		ai.GET("/conversations/:id/messages", aiAssistantController.GetMessages)
		ai.GET("/messages/:id/status", aiAssistantController.GetMessageStatus)
		ai.PATCH("/messages/:id/stop", aiAssistantController.StopMessage)

		// 集群操作
		ai.POST("/cluster-operations", aiAssistantController.SubmitClusterOperation)
		ai.GET("/cluster-operations/:id/status", aiAssistantController.GetOperationStatus)

		// 系统监控
		ai.GET("/system/health", aiAssistantController.GetSystemHealth)
		ai.GET("/system/usage", aiAssistantController.GetUsageStats)
		ai.GET("/usage-stats", aiAssistantController.GetUsageStats) // 别名，兼容前端调用

		// 连接测试
		ai.POST("/test-connection", aiAssistantController.TestBotConnection)
		ai.GET("/models", aiAssistantController.GetBotModels)

		// 快速聊天
		ai.POST("/quick-chat", aiAssistantController.QuickChat)
	}

	// 对象存储管理路由（需要认证）
	objectStorageController := controllers.NewObjectStorageController(database.DB)
	objectStorage := api.Group("/object-storage")
	objectStorage.Use(middleware.AuthMiddlewareWithSession())
	{
		// 配置管理
		objectStorage.GET("/configs", objectStorageController.GetConfigs)
		objectStorage.GET("/configs/:id", objectStorageController.GetConfig)
		objectStorage.POST("/configs", objectStorageController.CreateConfig)
		objectStorage.PUT("/configs/:id", objectStorageController.UpdateConfig)
		objectStorage.DELETE("/configs/:id", objectStorageController.DeleteConfig)
		objectStorage.POST("/configs/:id/activate", objectStorageController.SetActiveConfig)

		// 连接测试和状态检查
		objectStorage.POST("/test-connection", objectStorageController.TestConnection)
		objectStorage.GET("/configs/:id/status", objectStorageController.CheckConnectionStatus)

		// 统计信息
		objectStorage.GET("/configs/:id/statistics", objectStorageController.GetStatistics)
	}

	// 导航配置管理路由（需要认证）
	navigationController := controllers.NewNavigationController()
	navigation := api.Group("/navigation")
	navigation.Use(middleware.AuthMiddlewareWithSession())
	{
		navigation.GET("/config", navigationController.GetNavigationConfig)
		navigation.POST("/config", navigationController.SaveNavigationConfig)
		navigation.PUT("/config", navigationController.SaveNavigationConfig) // 支持 PUT 方法保存配置
		navigation.DELETE("/config", navigationController.ResetNavigationConfig)
		navigation.GET("/default", navigationController.GetDefaultNavigationConfig)
	}

	// KeyVault 密钥保管库路由
	keyVaultHandler := handlers.NewKeyVaultHandler()
	// 自动迁移 KeyVault 数据库表
	if err := keyVaultHandler.AutoMigrate(); err != nil {
		log.Printf("Warning: Failed to migrate KeyVault tables: %v", err)
	}

	keyvault := api.Group("/keyvault")
	{
		// 公开端点：健康检查
		keyvault.GET("/health", keyVaultHandler.HealthCheck)

		// 同步端点：使用一次性令牌，不需要 JWT（令牌本身包含鉴权信息）
		keyvault.POST("/sync", keyVaultHandler.SyncKey)
		keyvault.POST("/sync/batch", keyVaultHandler.SyncMultipleKeys)
		keyvault.POST("/sync/store", keyVaultHandler.StoreKeyWithToken)

		// 需要 JWT 认证的端点
		keyvaultAuth := keyvault.Group("")
		keyvaultAuth.Use(middleware.AuthMiddlewareWithSession())
		{
			// 生成同步令牌
			keyvaultAuth.POST("/sync-token", keyVaultHandler.GenerateSyncToken)

			// 密钥管理（需要认证）
			keyvaultAuth.GET("/keys", keyVaultHandler.ListKeys)
			keyvaultAuth.GET("/keys/:name", keyVaultHandler.GetKey)
			keyvaultAuth.POST("/keys", keyVaultHandler.StoreKey)
			keyvaultAuth.DELETE("/keys/:name", keyVaultHandler.DeleteKey)

			// 访问日志（需要认证）
			keyvaultAuth.GET("/logs", keyVaultHandler.GetAccessLogs)
		}
	}

	// 安全管理路由（需要认证）
	securityHandler := handlers.NewSecurityHandler()
	// 自动迁移安全相关数据库表
	if err := securityHandler.AutoMigrate(); err != nil {
		log.Printf("Warning: Failed to migrate security tables: %v", err)
	}

	security := api.Group("/security")
	security.Use(middleware.AuthMiddlewareWithSession())
	{
		// IP 黑名单管理
		security.GET("/ip-blacklist", securityHandler.ListIPBlacklist)
		security.POST("/ip-blacklist", securityHandler.AddIPBlacklist)
		security.PUT("/ip-blacklist/:id", securityHandler.UpdateIPBlacklist)
		security.DELETE("/ip-blacklist/:id", securityHandler.DeleteIPBlacklist)
		security.POST("/ip-blacklist/batch-delete", securityHandler.BatchDeleteIPBlacklist)

		// IP 白名单管理
		security.GET("/ip-whitelist", securityHandler.ListIPWhitelist)
		security.POST("/ip-whitelist", securityHandler.AddIPWhitelist)
		security.DELETE("/ip-whitelist/:id", securityHandler.DeleteIPWhitelist)

		// 二次认证（2FA）管理
		security.GET("/2fa/status", securityHandler.Get2FAStatus)
		security.POST("/2fa/setup", securityHandler.Setup2FA)
		security.POST("/2fa/enable", securityHandler.Enable2FA)
		security.POST("/2fa/disable", securityHandler.Disable2FA)
		security.POST("/2fa/verify", securityHandler.Verify2FA)
		security.POST("/2fa/recovery-codes", securityHandler.RegenerateRecoveryCodes)

		// 管理员2FA管理（为其他用户管理2FA）
		security.GET("/admin/2fa/:user_id/status", securityHandler.AdminGet2FAStatus)
		security.POST("/admin/2fa/:user_id/enable", securityHandler.AdminEnable2FA)
		security.POST("/admin/2fa/:user_id/disable", securityHandler.AdminDisable2FA)

		// OAuth 第三方登录配置
		security.GET("/oauth/providers", securityHandler.ListOAuthProviders)
		security.GET("/oauth/providers/:id", securityHandler.GetOAuthProvider)
		security.PUT("/oauth/providers/:id", securityHandler.UpdateOAuthProvider)

		// 全局安全配置 - 只允许管理员访问
		security.GET("/config", middleware.AdminMiddleware(), securityHandler.GetSecurityConfig)
		security.PUT("/config", middleware.AdminMiddleware(), securityHandler.UpdateSecurityConfig)

		// 安全审计日志
		security.GET("/audit-logs", securityHandler.ListAuditLogs)

		// 登录保护管理
		security.GET("/locked-accounts", securityHandler.GetLockedAccounts)
		security.POST("/accounts/:username/unlock", securityHandler.UnlockAccount)
		security.GET("/blocked-ips", securityHandler.GetBlockedIPsFromProtection)
		security.POST("/block-ip", securityHandler.BlockIPManually)
		security.POST("/ips/:ip/unblock", securityHandler.UnblockIP)
		security.GET("/login-attempts", securityHandler.GetLoginAttempts)
		security.GET("/ip-stats", securityHandler.GetIPLoginStats)
		security.GET("/ip-stats/:ip", securityHandler.GetIPStatsDetail)
		security.GET("/login-stats/summary", securityHandler.GetLoginStatsSummary)
		security.POST("/login-records/cleanup", securityHandler.CleanupLoginRecords)

		// 客户端信息和GeoIP查询
		security.GET("/client-info", securityHandler.GetClientInfo)
		security.GET("/geoip/:ip", securityHandler.LookupIPGeoInfo)
		security.POST("/geoip/batch", securityHandler.BatchLookupIPGeoInfo)
		security.GET("/geoip/stats", securityHandler.GetGeoIPCacheStats)
	}

	// ArgoCD GitOps 管理路由（需要认证）
	argoCDHandler := handlers.NewArgoCDHandler()
	argocd := api.Group("/argocd")
	argocd.Use(middleware.AuthMiddlewareWithSession())
	{
		// ArgoCD 服务状态和配置
		argocd.GET("/status", argoCDHandler.GetArgoCDStatus)
		argocd.POST("/status/refresh", argoCDHandler.RefreshArgoCDAvailability)
		argocd.GET("/version", argoCDHandler.GetVersion)
		argocd.GET("/settings", argoCDHandler.GetSettings)

		// 应用管理
		argocd.GET("/applications", argoCDHandler.ListApplications)
		argocd.GET("/applications/:name", argoCDHandler.GetApplication)
		argocd.POST("/applications", argoCDHandler.CreateApplication)
		argocd.DELETE("/applications/:name", argoCDHandler.DeleteApplication)
		argocd.POST("/applications/:name/sync", argoCDHandler.SyncApplication)
		argocd.POST("/applications/:name/refresh", argoCDHandler.RefreshApplication)
		argocd.GET("/applications/:name/resource-tree", argoCDHandler.GetApplicationResourceTree)

		// 仓库管理
		argocd.GET("/repositories", argoCDHandler.ListRepositories)
		argocd.POST("/repositories", argoCDHandler.CreateRepository)
		argocd.DELETE("/repositories/*repo", argoCDHandler.DeleteRepository)

		// 集群管理 (ArgoCD 内部集群)
		argocd.GET("/clusters", argoCDHandler.ListClusters)
		argocd.GET("/clusters/managed", argoCDHandler.ListArgoCDManagedClusters)
		argocd.POST("/clusters/sync-all", argoCDHandler.SyncAllClusters)

		// 项目管理
		argocd.GET("/projects", argoCDHandler.ListProjects)
		argocd.GET("/projects/:name", argoCDHandler.GetProject)
		argocd.POST("/projects", argoCDHandler.CreateProject)
	}

	// Keycloak 身份认证管理路由（需要认证）
	keycloakHandler := handlers.NewKeycloakHandler()
	keycloak := api.Group("/keycloak")
	keycloak.Use(middleware.AuthMiddlewareWithSession())
	{
		// 服务器信息
		keycloak.GET("/server-info", keycloakHandler.GetServerInfo)

		// Realm 管理
		keycloak.GET("/realms", keycloakHandler.ListRealms)
		keycloak.GET("/realms/:realm", keycloakHandler.GetRealm)

		// 用户管理
		keycloak.GET("/realms/:realm/users", keycloakHandler.ListUsers)
		keycloak.GET("/realms/:realm/users/:userId", keycloakHandler.GetUser)
		keycloak.POST("/realms/:realm/users", keycloakHandler.CreateUser)
		keycloak.PUT("/realms/:realm/users/:userId", keycloakHandler.UpdateUser)
		keycloak.DELETE("/realms/:realm/users/:userId", keycloakHandler.DeleteUser)
		keycloak.PUT("/realms/:realm/users/:userId/reset-password", keycloakHandler.ResetPassword)
		keycloak.GET("/realms/:realm/users/:userId/role-mappings", keycloakHandler.GetUserRoles)
		keycloak.GET("/realms/:realm/users/:userId/sessions", keycloakHandler.GetUserSessions)

		// 客户端管理
		keycloak.GET("/realms/:realm/clients", keycloakHandler.ListClients)
		keycloak.GET("/realms/:realm/clients/:clientId", keycloakHandler.GetClient)

		// 用户组管理
		keycloak.GET("/realms/:realm/groups", keycloakHandler.ListGroups)
		keycloak.GET("/realms/:realm/groups/:groupId", keycloakHandler.GetGroup)

		// 角色管理
		keycloak.GET("/realms/:realm/roles", keycloakHandler.ListRoles)

		// 会话管理
		keycloak.GET("/realms/:realm/sessions", keycloakHandler.ListSessions)
	}

	// 基础设施审计日志路由（需要认证）
	auditHandler := handlers.NewAuditHandler()
	// 初始化审计配置
	services.GetAuditService().InitializeDefaultConfigs()

	audit := api.Group("/audit")
	audit.Use(middleware.AuthMiddlewareWithSession())
	{
		// 审计日志查询
		audit.GET("/logs", auditHandler.ListAuditLogs)
		audit.GET("/logs/:id", auditHandler.GetAuditLog)
		audit.GET("/my-logs", auditHandler.GetUserAuditLogs)
		audit.GET("/resources/:resource_type/:resource_id", auditHandler.GetResourceAuditLogs)

		// 审计统计
		audit.GET("/statistics", auditHandler.GetAuditStatistics)

		// 审计元数据
		audit.GET("/categories", auditHandler.GetAuditCategories)
		audit.GET("/actions", auditHandler.GetAuditActions)

		// 审计配置管理（仅管理员）
		auditAdmin := audit.Group("/configs")
		auditAdmin.Use(middleware.AdminMiddleware())
		{
			auditAdmin.GET("", auditHandler.ListAuditConfigs)
			auditAdmin.GET("/:category", auditHandler.GetAuditConfig)
			auditAdmin.PUT("/:category", auditHandler.UpdateAuditConfig)
			auditAdmin.POST("/init", auditHandler.InitializeAuditConfigs)
		}

		// 审计日志导出（仅管理员）
		audit.POST("/export", middleware.AdminMiddleware(), auditHandler.ExportAuditLogs)

		// 审计日志清理（仅管理员）
		audit.POST("/cleanup", middleware.AdminMiddleware(), auditHandler.CleanupAuditLogs)
	}
}
