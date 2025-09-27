package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	
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

// AuthMiddlewareWithSession JWT认证中间件（支持Redis会话和Cookie）
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
			if cookie, err := c.Cookie("auth_token"); err == nil {
				tokenString = cookie
			}
		}
		
		// 如果既没有Authorization头部也没有cookie，返回401
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}
		
		// 首先检查Redis会话
		session, err := sessionService.GetSession(tokenString)
		if err != nil {
			// Redis会话检查失败，回退到JWT验证
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
		} else {
			// 使用Redis会话信息
			c.Set("user_id", session.UserID)
			c.Set("username", session.Username)
			c.Set("roles", session.Roles)
			c.Set("permissions", session.Permissions)
			
			// 更新会话活动时间
			sessionService.UpdateActivity(tokenString)
		}
		
		c.Next()
	}
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


