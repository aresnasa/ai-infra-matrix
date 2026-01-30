package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 检查Bearer token格式
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]
		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

// AuthMiddlewareWithSession JWT认证中间件（支持Redis会话、Cookie和Keycloak Token）
func AuthMiddlewareWithSession() gin.HandlerFunc {
	sessionService := services.NewSessionService()

	return func(c *gin.Context) {
		var tokenString string

		// 首先尝试从Authorization头部获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// 检查Bearer token格式
			tokenParts := strings.SplitN(authHeader, " ", 2)
			if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
				tokenString = tokenParts[1]
			}
		}

		// 如果没有Authorization头部，尝试从cookie获取token
		if tokenString == "" {
			// 优先检查 ai_infra_token cookie
			if cookie, err := c.Cookie("ai_infra_token"); err == nil && cookie != "" {
				tokenString = cookie
			}
		}
		if tokenString == "" {
			if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
				tokenString = cookie
			}
		}
		if tokenString == "" {
			if cookie, err := c.Cookie("jwt_token"); err == nil && cookie != "" {
				tokenString = cookie
			}
		}

		// 如果既没有Authorization头部也没有cookie，返回401
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// 认证流程：Redis会话 -> 本地JWT -> Keycloak Token（JIT用户预配）
		authenticated := false

		// 1. 首先检查Redis会话
		session, err := sessionService.GetSession(tokenString)
		if err == nil && session != nil {
			// 使用Redis会话信息
			c.Set("user_id", session.UserID)
			c.Set("username", session.Username)
			c.Set("roles", session.Roles)
			c.Set("permissions", session.Permissions)
			// 更新会话活动时间
			sessionService.UpdateActivity(tokenString)
			authenticated = true
		}

		// 2. 如果Redis会话不存在，尝试本地JWT验证
		if !authenticated {
			claims, err := jwt.ParseToken(tokenString)
			if err == nil {
				// 将用户信息存储到上下文中
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("roles", claims.Roles)
				c.Set("permissions", claims.Permissions)
				authenticated = true
			}
		}

		// 3. 如果本地JWT验证失败，尝试Keycloak Token验证（JIT用户预配）
		if !authenticated {
			userID, username, roles, permissions, err := tryKeycloakAuth(tokenString)
			if err == nil {
				c.Set("user_id", userID)
				c.Set("username", username)
				c.Set("roles", roles)
				c.Set("permissions", permissions)
				c.Set("auth_source", "keycloak") // 标记来源为Keycloak
				authenticated = true
			}
		}

		if !authenticated {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// tryKeycloakAuth 尝试使用Keycloak Token进行认证，并实现JIT（Just-In-Time）用户预配
// 返回: userID, username, roles, permissions, error
func tryKeycloakAuth(token string) (uint, string, []string, []string, error) {
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Debug("Failed to load config for Keycloak auth")
		return 0, "", nil, nil, err
	}

	// 检查Keycloak是否启用
	if !cfg.Keycloak.Enabled {
		return 0, "", nil, nil, errors.New("keycloak is not enabled")
	}

	// 创建Keycloak服务并验证Token
	keycloakService := services.NewKeycloakService(cfg, database.DB)
	tokenInfo, err := keycloakService.ValidateToken(token)
	if err != nil {
		logrus.WithError(err).Debug("Keycloak token validation failed")
		return 0, "", nil, nil, err
	}

	if tokenInfo == nil || tokenInfo.Username == "" {
		return 0, "", nil, nil, errors.New("keycloak token validation returned empty user info")
	}

	username := tokenInfo.Username
	keycloakRoles := tokenInfo.Roles

	logrus.WithFields(logrus.Fields{
		"username":       username,
		"keycloak_roles": keycloakRoles,
	}).Debug("Keycloak token validated, performing JIT user provisioning")

	// JIT用户预配：检查用户是否存在于本地数据库
	var user models.User
	db := database.DB
	err = db.Where("username = ?", username).First(&user).Error

	if err != nil {
		// 用户不存在，创建新用户（JIT Provisioning）
		logrus.WithField("username", username).Info("Creating JIT user from Keycloak")

		user = models.User{
			Username:   username,
			Email:      username + "@keycloak.local", // 默认邮箱，后续可从Keycloak获取
			IsActive:   true,
			AuthSource: "keycloak",
		}

		// 尝试从Keycloak获取更多用户信息
		keycloakUser, err := keycloakService.GetUser(username)
		if err == nil && keycloakUser != nil {
			if keycloakUser.Email != "" {
				user.Email = keycloakUser.Email
			}
			if keycloakUser.FirstName != "" || keycloakUser.LastName != "" {
				// 可以设置全名
			}
		}

		// 创建用户
		if err := db.Create(&user).Error; err != nil {
			// 如果创建失败（可能是并发创建），尝试重新查询
			if err2 := db.Where("username = ?", username).First(&user).Error; err2 != nil {
				logrus.WithError(err).WithField("username", username).Error("Failed to create JIT user")
				return 0, "", nil, nil, errors.New("failed to create JIT user")
			}
		} else {
			logrus.WithFields(logrus.Fields{
				"user_id":  user.ID,
				"username": username,
			}).Info("JIT user created from Keycloak")

			// 异步同步到Gitea和其他组件
			go jitSyncUserToComponents(&user, cfg)
		}
	}

	// 映射Keycloak角色到本地角色
	localRoles := mapKeycloakRolesToLocal(keycloakRoles)
	permissions := getRolePermissions(db, localRoles)

	return user.ID, user.Username, localRoles, permissions, nil
}

// jitSyncUserToComponents 异步同步JIT用户到各个组件（Gitea等）
func jitSyncUserToComponents(user *models.User, cfg *config.Config) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("panic", r).Error("Panic in JIT user sync")
		}
	}()

	// 给一点延迟确保用户已经持久化
	time.Sleep(100 * time.Millisecond)

	// 同步到Gitea
	if cfg.Gitea.Enabled {
		giteaService := services.NewGiteaService(cfg)
		if err := giteaService.EnsureUser(*user); err != nil {
			logrus.WithError(err).WithField("username", user.Username).Warn("Failed to sync JIT user to Gitea")
		} else {
			logrus.WithField("username", user.Username).Info("JIT user synced to Gitea")
		}
	}
}

// mapKeycloakRolesToLocal 将Keycloak角色映射到本地角色
func mapKeycloakRolesToLocal(keycloakRoles []string) []string {
	roleMapping := map[string]string{
		"admin":          "admin",
		"realm-admin":    "admin",
		"sre_admin":      "admin",
		"developer":      "developer",
		"devops":         "devops",
		"viewer":         "viewer",
		"user":           "user",
		"default-roles-ai-infra": "user",
	}

	localRoles := make([]string, 0)
	roleSet := make(map[string]bool)

	for _, kcRole := range keycloakRoles {
		// 直接映射
		if localRole, ok := roleMapping[strings.ToLower(kcRole)]; ok {
			if !roleSet[localRole] {
				localRoles = append(localRoles, localRole)
				roleSet[localRole] = true
			}
		}
	}

	// 如果没有映射到任何角色，给一个默认的user角色
	if len(localRoles) == 0 {
		localRoles = append(localRoles, "user")
	}

	return localRoles
}

// getRolePermissions 获取角色对应的权限
func getRolePermissions(db *gorm.DB, roles []string) []string {
	permissions := make([]string, 0)
	permSet := make(map[string]bool)

	for _, roleName := range roles {
		var role models.Role
		if err := db.Where("name = ?", roleName).Preload("Permissions").First(&role).Error; err != nil {
			continue
		}
		for _, perm := range role.Permissions {
			key := perm.GetPermissionKey()
			if !permSet[key] {
				permissions = append(permissions, key)
				permSet[key] = true
			}
		}
	}

	return permissions
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No roles found"})
			c.Abort()
			return
		}

		roleList, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid roles format"})
			c.Abort()
			return
		}

		// 检查是否有管理员角色
		hasAdminRole := false
		for _, role := range roleList {
			if role == "admin" {
				hasAdminRole = true
				break
			}
		}

		if !hasAdminRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetCurrentUserID 获取当前用户ID
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetCurrentUserRoles 获取当前用户的角色列表
func GetCurrentUserRoles(c *gin.Context) ([]string, bool) {
	roles, exists := c.Get("roles")
	if !exists {
		return nil, false
	}
	roleList, ok := roles.([]string)
	return roleList, ok
}

// GetCurrentUserPermissions 获取当前用户的权限列表
func GetCurrentUserPermissions(c *gin.Context) ([]string, bool) {
	permissions, exists := c.Get("permissions")
	if !exists {
		return nil, false
	}
	permissionList, ok := permissions.([]string)
	return permissionList, ok
}

// HasRole 检查用户是否具有指定角色
func HasRole(c *gin.Context, roleName string) bool {
	roles, ok := GetCurrentUserRoles(c)
	if !ok {
		return false
	}

	for _, role := range roles {
		if role == roleName {
			return true
		}
	}
	return false
}

// HasPermission 检查用户是否具有指定权限
func HasPermission(c *gin.Context, permission string) bool {
	permissions, ok := GetCurrentUserPermissions(c)
	if !ok {
		return false
	}

	for _, perm := range permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// GetUserID 从上下文中获取用户ID
func GetUserID(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, errors.New("user ID not found")
	}

	// 处理不同类型的用户ID
	switch v := userID.(type) {
	case uint:
		return v, nil
	case int:
		return uint(v), nil
	case string:
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return uint(id), nil
	default:
		return 0, errors.New("invalid user ID type")
	}
}
