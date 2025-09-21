package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/handlers"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/controllers"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	_ "github.com/aresnasa/ai-infra-matrix/src/backend/docs"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
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
		
		// 设置role_template字段
		if existingAdmin.RoleTemplate == "" {
			existingAdmin.RoleTemplate = "admin"
			if err := db.Save(&existingAdmin).Error; err != nil {
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

	// 连接数据库
	if err := database.Connect(cfg); err != nil {
		logrus.Fatal("Failed to connect to database:", err)
	}

	// 运行数据库迁移
	if err := database.Migrate(); err != nil {
		logrus.Fatal("Failed to migrate database:", err)
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
	slurmService := services.NewSlurmService()
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
			"path": c.Request.URL.Path,
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
			"method": method,
			"origin": origin,
			"path": c.Request.URL.Path,
			"user_agent": c.GetHeader("User-Agent"),
		}).Info("CORS Debug: Request received")
		
		c.Next()
		
		logrus.WithFields(logrus.Fields{
			"method": method,
			"origin": origin,
			"path": c.Request.URL.Path,
			"status": c.Writer.Status(),
		}).Info("CORS Debug: Response sent")
	})

	// 手动CORS中间件替代gin-contrib/cors
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method
		
		logrus.WithFields(logrus.Fields{
			"method": method,
			"origin": origin,
			"path": c.Request.URL.Path,
			"user_agent": c.GetHeader("User-Agent"),
		}).Info("Manual CORS: Request received")
		
		// 设置CORS头
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Authorization, X-Requested-With, Accept, Access-Control-Request-Method, Access-Control-Request-Headers")
		c.Header("Access-Control-Max-Age", "86400")
		
		// 处理预检请求
		if method == "OPTIONS" {
			logrus.WithFields(logrus.Fields{
				"origin": origin,
				"path": c.Request.URL.Path,
			}).Info("Manual CORS: Handling OPTIONS preflight request")
			c.AbortWithStatus(200)
			return
		}
		
		c.Next()
		
		logrus.WithFields(logrus.Fields{
			"method": method,
			"origin": origin,
			"path": c.Request.URL.Path,
			"status": c.Writer.Status(),
		}).Info("Manual CORS: Response sent")
	})

	// Swagger文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 设置 API 路由
	setupAPIRoutes(r, cfg, jobService, sshService)

	// 优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logrus.Info("Shutting down server...")
		
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
				"status": "error",
				"message": "Failed to get database instance",
				"error": err.Error(),
			})
			return
		}
		
		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{
				"status": "error",
				"message": "Database connection failed",
				"error": err.Error(),
			})
			return
		}
		
		// 检查Redis连接
		if err := cache.RDB.Ping(c.Request.Context()).Err(); err != nil {
			c.JSON(500, gin.H{
				"status": "error", 
				"message": "Redis connection failed",
				"error": err.Error(),
			})
			return
		}
		
		c.JSON(200, gin.H{
			"status": "healthy",
			"message": "All services are running",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 鉴权路由（公开）
	userHandler := handlers.NewUserHandler(database.DB)
	// JupyterHub认证处理器（在多个地方使用）
	jupyterHubAuthHandler := handlers.NewJupyterHubAuthHandler(database.DB, cfg, cache.RDB)

	auth := api.Group("/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/validate-ldap", userHandler.ValidateLDAP)
		auth.POST("/login", userHandler.Login)
		auth.POST("/logout", middleware.AuthMiddlewareWithSession(), userHandler.Logout)
		auth.POST("/refresh", userHandler.RefreshToken)
		// 兼容前端/SSO刷新端点
		auth.POST("/refresh-token", userHandler.RefreshToken)
		auth.GET("/profile", middleware.AuthMiddlewareWithSession(), userHandler.GetProfile)
		auth.GET("/me", middleware.AuthMiddlewareWithSession(), userHandler.GetProfile)
		auth.PUT("/profile", middleware.AuthMiddlewareWithSession(), userHandler.UpdateProfile)
		auth.PUT("/change-password", middleware.AuthMiddlewareWithSession(), userHandler.ChangePassword)
		// JupyterHub单点登录令牌生成
		auth.POST("/jupyterhub-token", middleware.AuthMiddlewareWithSession(), userHandler.GenerateJupyterHubToken)
		// JWT令牌验证（用于JupyterHub认证器）
		auth.POST("/verify-token", userHandler.VerifyJWT)
		// 简单令牌验证（用于SSO认证）
		auth.GET("/verify", middleware.AuthMiddleware(), userHandler.VerifyTokenSimple)
		
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
			users.PUT("/:id/reset-password", userHandler.AdminResetPassword)
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
			admin.GET("/ldap/sync/history", adminController.GetLDAPSyncHistory)			// 系统统计
			admin.GET("/stats", adminController.GetSystemStats)
			admin.GET("/user-stats", adminController.GetUserStatistics)
			
			// RBAC初始化
			admin.POST("/rbac/initialize", adminController.InitializeRBAC)
			
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

		// Slurm 路由（需要认证）
		slurmController := controllers.NewSlurmController()
		slurm := api.Group("/slurm")
		slurm.Use(middleware.AuthMiddlewareWithSession())
		{
			slurm.GET("/summary", slurmController.GetSummary)
			slurm.GET("/nodes", slurmController.GetNodes)
			slurm.GET("/jobs", slurmController.GetJobs)

			// 扩缩容相关路由
			slurm.GET("/scaling/status", slurmController.GetScalingStatus)
			slurm.POST("/scaling/scale-up", slurmController.ScaleUp)
			slurm.POST("/scaling/scale-down", slurmController.ScaleDown)

			// 节点模板管理
			slurm.GET("/node-templates", slurmController.GetNodeTemplates)
			slurm.POST("/node-templates", slurmController.CreateNodeTemplate)
			slurm.PUT("/node-templates/:id", slurmController.UpdateNodeTemplate)
			slurm.DELETE("/node-templates/:id", slurmController.DeleteNodeTemplate)

			// SaltStack集成
			slurm.GET("/saltstack/integration", slurmController.GetSaltStackIntegration)
			slurm.POST("/saltstack/deploy-minion", slurmController.DeploySaltMinion)
			slurm.POST("/saltstack/execute", slurmController.ExecuteSaltCommand)
			slurm.GET("/saltstack/jobs", slurmController.GetSaltJobs)
		}

		// 作业管理路由（需要认证）
		jobController := controllers.NewJobController(jobService)
		jobs := api.Group("/jobs")
		jobs.Use(middleware.AuthMiddlewareWithSession())
		{
			jobs.GET("", jobController.ListJobs)
			jobs.POST("", jobController.SubmitJob)
			jobs.GET("/:jobId", jobController.GetJobDetail)
			jobs.POST("/:jobId/cancel", jobController.CancelJob)
			jobs.GET("/:jobId/output", jobController.GetJobOutput)
			jobs.GET("/clusters", jobController.ListClusters)
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

}
