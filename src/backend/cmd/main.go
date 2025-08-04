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
	_ "github.com/aresnasa/ai-infra-matrix/src/backend/docs"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

// @title Ansible Playbook Generator API
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

	// 设置JWT密钥
	jwt.SetSecret(cfg.JWTSecret)

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
	auth := api.Group("/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/login", userHandler.Login)
			auth.POST("/logout", middleware.AuthMiddlewareWithSession(), userHandler.Logout)
			auth.GET("/profile", middleware.AuthMiddlewareWithSession(), userHandler.GetProfile)
			auth.GET("/me", middleware.AuthMiddlewareWithSession(), userHandler.GetProfile)
			auth.PUT("/profile", middleware.AuthMiddlewareWithSession(), userHandler.UpdateProfile)
			auth.PUT("/change-password", middleware.AuthMiddlewareWithSession(), userHandler.ChangePassword)
			// JupyterHub单点登录令牌生成
			auth.POST("/jupyterhub-token", middleware.AuthMiddlewareWithSession(), userHandler.GenerateJupyterHubToken)
			// JWT令牌验证（用于JupyterHub认证器）
			auth.POST("/verify-token", userHandler.VerifyJWT)
		}

		// JupyterHub认证路由（独立处理）
		jupyterHubAuthHandler := handlers.NewJupyterHubAuthHandler(database.DB, cfg, cache.RDB)
		{
			// JupyterHub令牌生成和验证
			auth.POST("/jupyterhub-login", middleware.AuthMiddlewareWithSession(), jupyterHubAuthHandler.GenerateJupyterHubLoginToken)
			auth.POST("/verify-jupyterhub-token", jupyterHubAuthHandler.VerifyJupyterHubToken)
		}

		// 用户管理路由（管理员）
		users := api.Group("/users")
		users.Use(middleware.AuthMiddlewareWithSession(), middleware.AdminMiddleware())
		{
			users.GET("", userHandler.GetUsers)
			users.DELETE("/:id", userHandler.DeleteUser)
			users.PUT("/:id/reset-password", userHandler.AdminResetPassword)
			users.PUT("/:id/groups", userHandler.AdminUpdateUserGroups)
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
			// 用户管理
			admin.GET("/users", adminController.GetAllUsers)
			admin.GET("/users/:id", adminController.GetUserDetail)
			admin.GET("/users/:id/auth-source", adminController.GetUserWithAuthSource)
			admin.PUT("/users/:id/status", adminController.UpdateUserStatus)
			admin.PUT("/users/:id/status-enhanced", adminController.UpdateUserStatusEnhanced)
			admin.DELETE("/users/:id", adminController.DeleteUser)
			
			// 项目管理
			admin.GET("/projects", adminController.GetAllProjects)
			admin.GET("/projects/:id", adminController.GetProjectDetail)
			admin.PUT("/projects/:id/transfer", adminController.TransferProject)
			
			// 回收站管理
			admin.GET("/projects/trash", adminController.GetProjectsTrash)
			admin.PATCH("/projects/:id/restore", adminController.RestoreProject)
			admin.DELETE("/projects/:id/force-delete", adminController.ForceDeleteProject)
			admin.DELETE("/projects/trash/clear", adminController.ClearTrash)
			
			// LDAP配置管理
			admin.GET("/ldap/config", adminController.GetLDAPConfig)
			admin.PUT("/ldap/config", adminController.UpdateLDAPConfig)
			admin.POST("/ldap/test", adminController.TestLDAPConnection)
			
			// LDAP同步管理
			admin.POST("/ldap/sync", adminController.SyncLDAPUsers)
			admin.GET("/ldap/sync/:sync_id/status", adminController.GetLDAPSyncStatus)
			admin.GET("/ldap/sync/history", adminController.GetLDAPSyncHistory)
			
			// 系统统计
			admin.GET("/stats", adminController.GetSystemStats)
			
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

		// AI 助手路由（需要认证）
		aiController := controllers.NewAIAssistantController()
		ai := api.Group("/ai")
		ai.Use(middleware.AuthMiddlewareWithSession())
		{
			// 配置管理（管理员权限）
			ai.POST("/configs", middleware.AdminOnlyMiddleware(database.DB), aiController.CreateConfig)
			ai.GET("/configs", aiController.ListConfigs)
			ai.GET("/configs/:id", aiController.GetConfig)
			ai.PUT("/configs/:id", middleware.AdminOnlyMiddleware(database.DB), aiController.UpdateConfig)
			ai.DELETE("/configs/:id", middleware.AdminOnlyMiddleware(database.DB), aiController.DeleteConfig)

			// 对话管理
			ai.POST("/conversations", aiController.CreateConversation)
			ai.GET("/conversations", aiController.ListConversations)
			ai.GET("/conversations/:id", aiController.GetConversation)
			ai.DELETE("/conversations/:id", aiController.DeleteConversation)

			// 消息处理（异步版本）
			ai.POST("/conversations/:id/messages", aiController.SendMessage)
			ai.GET("/conversations/:id/messages", aiController.GetMessages)
			ai.GET("/messages/:message_id/status", aiController.GetMessageStatus)

			// 快速聊天（异步版本）
			ai.POST("/quick-chat", aiController.QuickChat)

			// 集群操作
			ai.POST("/cluster-operations", aiController.SubmitClusterOperation)
			ai.GET("/operations/:operation_id/status", aiController.GetOperationStatus)

			// 系统健康检查
			ai.GET("/health", aiController.GetSystemHealth)

			// 使用统计
			ai.GET("/usage-stats", aiController.GetUsageStats)

			// 异步API子路由组（为测试提供专用端点）
			async := ai.Group("/async")
			{
				// 系统健康检查
				async.GET("/health", aiController.GetSystemHealth)
				
				// 快速聊天（异步版本）
				async.POST("/quick-chat", aiController.QuickChat)
				
				// 消息状态检查
				async.GET("/messages/:message_id/status", aiController.GetMessageStatus)
				
				// 集群操作
				async.POST("/cluster-operations", aiController.SubmitClusterOperation)
				async.GET("/operations/:operation_id/status", aiController.GetOperationStatus)
				
				// 使用统计
				async.GET("/usage-stats", aiController.GetUsageStats)
			}
		}

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
