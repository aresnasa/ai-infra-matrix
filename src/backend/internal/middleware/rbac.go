package middleware

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RBACMiddleware RBAC权限检查中间件
func RBACMiddleware(db *gorm.DB, resource, verb string) gin.HandlerFunc {
	rbacService := services.NewRBACService(db)

	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
			c.Abort()
			return
		}

		// 构建作用域
		scope := "*"

		// 如果是项目相关的操作，检查项目所有权
		if resource == "projects" {
			projectIDParam := c.Param("id")
			if projectIDParam != "" {
				projectID, err := strconv.ParseUint(projectIDParam, 10, 32)
				if err == nil {
					scope = "project:" + strconv.FormatUint(projectID, 10)

					// 检查项目所有权
					if verb != "create" && verb != "list" {
						if !rbacService.HasRoleInProject(uid, uint(projectID), "owner") &&
							!rbacService.CheckPermission(uid, resource, verb, "*", "") {
							c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
							c.Abort()
							return
						}
					}
				}
			}
		}

		// 检查权限
		if !rbacService.CheckPermission(uid, resource, verb, scope, "") {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnlyMiddleware 仅管理员访问中间件
func AdminOnlyMiddleware(db *gorm.DB) gin.HandlerFunc {
	rbacService := services.NewRBACService(db)

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
			c.Abort()
			return
		}

		// 检查是否是管理员
		if !rbacService.IsAdmin(uid) {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ProjectOwnerMiddleware 项目所有者中间件
func ProjectOwnerMiddleware(db *gorm.DB) gin.HandlerFunc {
	rbacService := services.NewRBACService(db)

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
			c.Abort()
			return
		}

		// 获取项目ID
		projectIDParam := c.Param("id")
		if projectIDParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少项目ID"})
			c.Abort()
			return
		}

		projectID, err := strconv.ParseUint(projectIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
			c.Abort()
			return
		}

		// 检查是否是项目所有者或管理员
		if !rbacService.HasRoleInProject(uid, uint(projectID), "owner") &&
			!rbacService.IsAdmin(uid) {
			c.JSON(http.StatusForbidden, gin.H{"error": "只有项目所有者或管理员可以访问"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission 需要特定权限的中间件工厂函数
func RequirePermission(db *gorm.DB, resource, verb, scope string) gin.HandlerFunc {
	rbacService := services.NewRBACService(db)

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
			c.Abort()
			return
		}

		// 动态构建作用域
		finalScope := scope
		if scope == "{id}" {
			idParam := c.Param("id")
			if idParam != "" {
				finalScope = idParam
			}
		}

		// 检查权限
		if !rbacService.CheckPermission(uid, resource, verb, finalScope, "") {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}
