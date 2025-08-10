package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserHandler struct {
	db             *gorm.DB
	userService    *services.UserService
	rbacService    *services.RBACService
	sessionService *services.SessionService
	ldapService    *services.LDAPService
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		db:             db,
		userService:    services.NewUserService(),
		rbacService:    services.NewRBACService(db),
		sessionService: services.NewSessionService(),
		ldapService:    services.NewLDAPService(db),
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "注册信息"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Register(&req)
	if err != nil {
		logrus.Error("Register error:", err)
		if err.Error() == "username or email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录并获取JWT token
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "登录信息"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	// 添加调试日志
	logrus.WithFields(logrus.Fields{
		"user_agent": c.GetHeader("User-Agent"),
		"origin":     c.GetHeader("Origin"),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
	}).Info("Login request received")

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithField("error", err).Error("Failed to bind login request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.WithField("username", req.Username).Info("Login attempt for user")

	var user *models.User
	var err error

	// 实现混合认证策略
	user, err = h.performHybridAuthentication(&req)
	if err != nil {
		logrus.Error("Authentication error:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 获取用户角色
	roles, err := h.rbacService.GetUserRoles(user.ID)
	if err != nil {
		logrus.Error("Get user roles error:", err)
		// 即使获取角色失败，也允许登录，但使用空角色列表
		roles = []models.Role{}
	}

	// 获取用户权限
	permissions, err := h.rbacService.GetUserPermissions(user.ID)
	if err != nil {
		logrus.Error("Get user permissions error:", err)
		// 即使获取权限失败，也允许登录，但使用空权限列表
		permissions = []models.Permission{}
	}

	// 提取角色名称和权限键
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	permissionKeys := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionKeys[i] = permission.GetPermissionKey()
	}

	// 生成JWT token
	token, expiresAt, err := jwt.GenerateToken(user.ID, user.Username, roleNames, permissionKeys)
	if err != nil {
		logrus.Error("Generate token error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// 创建Redis会话
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	if err := h.sessionService.CreateSession(user, token, clientIP, userAgent); err != nil {
		logrus.Error("Create session error:", err)
		// 会话创建失败不阻止登录，只记录错误
	}

	response := models.LoginResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}

	c.JSON(http.StatusOK, response)
}

// handleLDAPUser 处理LDAP认证用户，创建或更新本地用户账户
func (h *UserHandler) handleLDAPUser(ldapUser *models.LDAPUser) (*models.User, error) {
	// 添加调试日志
	logrus.Debugf("handleLDAPUser called with Username: %s, DisplayName: %s, Email: %s", 
		ldapUser.Username, ldapUser.DisplayName, ldapUser.Email)
	
	// 首先尝试通过用户名查找现有用户
	existingUser, err := h.userService.GetUserByUsername(ldapUser.Username)
	if err == nil {
		// 用户已存在，更新用户信息
		updates := map[string]interface{}{
			"email":       ldapUser.Email,
			"last_login":  gorm.Expr("NOW()"),
			"is_active":   true,
		}
		
		if err := h.userService.UpdateUser(existingUser.ID, updates); err != nil {
			return nil, err
		}
		
		// 检查并分配LDAP管理员角色
		if h.ldapService.IsUserAdmin(ldapUser) {
			if err := h.assignAdminRoleIfNeeded(existingUser.ID); err != nil {
				logrus.Error("Failed to assign admin role to LDAP user:", err)
			}
		}
		
		return existingUser, nil
	} else {
		// 用户不存在，创建新用户
		newUser := &models.User{
			Username:   ldapUser.Username,
			Email:      ldapUser.Email,
			Password:   "", // LDAP用户不设置本地密码
			IsActive:   true,
			AuthSource: "ldap", // 设置认证源为LDAP
			LDAPDn:     ldapUser.DN, // 设置LDAP DN
		}
		
		if err := h.userService.CreateUserDirectly(newUser); err != nil {
			return nil, err
		}
		
		// 检查并分配LDAP管理员角色
		if h.ldapService.IsUserAdmin(ldapUser) {
			if err := h.assignAdminRoleIfNeeded(newUser.ID); err != nil {
				logrus.Error("Failed to assign admin role to new LDAP user:", err)
			}
		}
		
		return newUser, nil
	}
}

// assignAdminRoleIfNeeded 为用户分配管理员角色（如果需要）
func (h *UserHandler) assignAdminRoleIfNeeded(userID uint) error {
	// 检查用户是否已经有管理员角色
	roles, err := h.rbacService.GetUserRoles(userID)
	if err != nil {
		return err
	}
	
	hasAdminRole := false
	for _, role := range roles {
		if role.Name == "admin" {
			hasAdminRole = true
			break
		}
	}
	
	if !hasAdminRole {
		// 查找管理员角色ID
		var adminRole models.Role
		if err := h.db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
			logrus.WithError(err).Error("Failed to find admin role")
			return err
		}
		
		// 分配管理员角色
		return h.rbacService.AssignRoleToUser(userID, adminRole.ID)
	}
	
	return nil
}

// GetUsers 获取用户列表（管理员功能）
// @Summary 获取用户列表
// @Description 获取系统中所有用户的列表（仅管理员）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := h.userService.GetUsers(page, pageSize)
	if err != nil {
		logrus.Error("Get users error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// DeleteUser 删除用户（管理员功能）
// @Summary 删除用户
// @Description 删除指定用户（仅管理员）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// 防止删除自己
	currentUserID, _ := middleware.GetCurrentUserID(c)
	if uint(userID) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	if err := h.userService.DeleteUser(uint(userID)); err != nil {
		logrus.Error("Delete user error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// Logout 用户退出登录
// @Summary 用户退出登录
// @Description 清除用户会话和token
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	// 获取Authorization header中的token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// 提取Bearer token
	tokenParts := strings.SplitN(authHeader, " ", 2)
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
		return
	}

	token := tokenParts[1]
	
	// 删除Redis会话
	if err := h.sessionService.DeleteSession(token); err != nil {
		logrus.Error("Delete session error:", err)
		// 即使删除会话失败，也返回成功，因为客户端会丢弃token
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ChangePassword 修改当前用户密码
// @Summary 修改当前用户密码
// @Description 用户修改自己的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "密码修改信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.ChangePassword(userID, &req); err != nil {
		logrus.Error("Change password error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// UpdateProfile 更新用户个人信息
// @Summary 更新用户个人信息
// @Description 用户更新自己的个人信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateUserProfileRequest true "个人信息更新"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var req models.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.UpdateUserProfile(userID, &req)
	if err != nil {
		logrus.Error("Update profile error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetProfile 获取用户个人信息
// @Summary 获取用户个人信息
// @Description 获取当前用户的详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	user, err := h.userService.GetUserWithDetails(userID)
	if err != nil {
		logrus.Error("Get profile error:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AdminResetPassword 管理员重置用户密码
// @Summary 管理员重置用户密码
// @Description 管理员重置指定用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param request body models.AdminResetPasswordRequest true "重置密码信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /admin/users/{id}/reset-password [post]
func (h *UserHandler) AdminResetPassword(c *gin.Context) {
	// 检查管理员权限
	currentUserID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	if !h.rbacService.CheckPermission(currentUserID, "users", "update", "*", "") {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 获取用户ID
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req models.AdminResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.AdminResetPassword(uint(userID), &req); err != nil {
		logrus.Error("Admin reset password error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功"})
}

// AdminUpdateUserGroups 管理员更新用户的用户组
// @Summary 管理员更新用户的用户组
// @Description 管理员修改指定用户的用户组归属
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param request body models.UpdateUserGroupsRequest true "用户组更新信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /admin/users/{id}/groups [put]
func (h *UserHandler) AdminUpdateUserGroups(c *gin.Context) {
	// 检查管理员权限
	currentUserID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	if !h.rbacService.CheckPermission(currentUserID, "groups", "update", "*", "") {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 获取用户ID
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req models.UpdateUserGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.AdminUpdateUserGroups(uint(userID), &req); err != nil {
		logrus.Error("Admin update user groups error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户组更新成功"})
}

// performHybridAuthentication 执行混合认证策略
func (h *UserHandler) performHybridAuthentication(req *models.LoginRequest) (*models.User, error) {
	// 首先检查LDAP配置是否启用
	ldapConfig, ldapErr := h.ldapService.GetLDAPConfig()
	
	// 检查是否为本地admin用户 - 如果是，优先尝试本地认证
	localUser, localErr := h.userService.GetUserByUsername(req.Username)
	isLocalAdmin := localErr == nil && h.isAdminUser(localUser)
	
	if ldapErr == nil && ldapConfig.IsEnabled {
		logrus.Debug("LDAP is enabled, attempting hybrid authentication")
		
		// 如果是本地admin用户，优先本地认证，除非被明确禁用
		if isLocalAdmin && localUser.IsActive {
			logrus.Debug("Attempting local authentication for admin user:", req.Username)
			if user, err := h.userService.Login(req); err == nil {
				logrus.Info("Local admin user authenticated successfully:", req.Username)
				return user, nil
			}
			logrus.Debug("Local authentication failed for admin user, trying LDAP")
		}
		
		// 尝试LDAP认证
		ldapUser, ldapAuthErr := h.ldapService.AuthenticateUser(req.Username, req.Password)
		if ldapAuthErr == nil {
			logrus.Info("LDAP authentication successful for user:", req.Username)
			// LDAP认证成功，检查或创建本地用户
			user, err := h.handleLDAPUser(ldapUser)
			if err != nil {
				logrus.Error("Failed to handle LDAP user:", err)
				return nil, errors.New("failed to process LDAP user")
			}
			return user, nil
		}
		
		logrus.Debug("LDAP authentication failed:", ldapAuthErr)
		
		// LDAP认证失败，尝试本地认证（除非是已经尝试过的本地admin用户）
		if !isLocalAdmin {
			logrus.Debug("Falling back to local authentication for user:", req.Username)
			user, err := h.userService.Login(req)
			if err != nil {
				return nil, errors.New("authentication failed")
			}
			logrus.Info("Local authentication successful for user:", req.Username)
			return user, nil
		}
		
		return nil, errors.New("authentication failed")
	} else {
		// LDAP未配置或未启用，使用本地认证
		logrus.Debug("LDAP not enabled, using local authentication")
		user, err := h.userService.Login(req)
		if err != nil {
			return nil, err
		}
		logrus.Info("Local authentication successful for user:", req.Username)
		return user, nil
	}
}

// isAdminUser 检查用户是否为管理员
func (h *UserHandler) isAdminUser(user *models.User) bool {
	roles, err := h.rbacService.GetUserRoles(user.ID)
	if err != nil {
		return false
	}
	
	for _, role := range roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

// GenerateJupyterHubToken 生成JupyterHub访问令牌
// @Summary 生成JupyterHub访问令牌
// @Description 为已认证用户生成JupyterHub访问令牌，用于单点登录到JupyterHub
// @Tags 用户管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/jupyterhub-token [post]
func (h *UserHandler) GenerateJupyterHubToken(c *gin.Context) {
	// 获取当前用户信息
	currentUserID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取用户详细信息
	currentUser, err := h.userService.GetUserByID(currentUserID)
	if err != nil {
		logrus.Error("Failed to get current user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	// 生成JupyterHub API token
	hubToken, err := h.generateJupyterHubAPIToken(currentUser)
	if err != nil {
		logrus.Error("Failed to generate JupyterHub token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "生成JupyterHub令牌失败",
			"details": err.Error(),
		})
		return
	}

	// 创建或更新JupyterHub用户
	if err := h.ensureJupyterHubUser(currentUser, hubToken); err != nil {
		logrus.Warn("Failed to ensure JupyterHub user, but continuing:", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"token":       hubToken,
		"username":    currentUser.Username,
		"expires_at":  time.Now().Add(24 * time.Hour).Unix(), // 24小时有效期
		"redirect_url": fmt.Sprintf("/hub/login?token=%s", hubToken),
	})
}

// generateJupyterHubAPIToken 生成JupyterHub API令牌
func (h *UserHandler) generateJupyterHubAPIToken(user *models.User) (string, error) {
	// 生成随机令牌
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	
	// 使用JWT生成令牌（复用现有的JWT基础设施）
	roles, _ := h.rbacService.GetUserRoles(user.ID)
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}
	
	permissions, _ := h.rbacService.GetUserPermissions(user.ID)
	permissionKeys := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionKeys[i] = permission.GetPermissionKey()
	}
	
	// 生成专用于JupyterHub的JWT令牌
	token, _, err := jwt.GenerateToken(user.ID, user.Username, roleNames, permissionKeys)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT token: %w", err)
	}
	
	// 在实际生产环境中，这里应该调用JupyterHub API创建令牌
	// 现在返回我们生成的JWT令牌
	return token, nil
}

// ensureJupyterHubUser 确保JupyterHub中存在该用户
func (h *UserHandler) ensureJupyterHubUser(user *models.User, token string) error {
	// 获取JupyterHub配置 - 修正端口配置
	jupyterHubURL := os.Getenv("JUPYTERHUB_URL")
	if jupyterHubURL == "" {
		jupyterHubURL = "http://localhost:8088" // 修正为实际端口8088
	}
	
	// 准备用户数据
	userData := map[string]interface{}{
		"username": user.Username,
		"admin":    h.isAdminUser(user),
	}
	
	jsonData, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}
	
	// 创建HTTP请求到JupyterHub API
	req, err := http.NewRequest("POST", 
		fmt.Sprintf("%s/hub/api/users/%s", jupyterHubURL, user.Username), 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	
	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to JupyterHub: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("JupyterHub API returned status: %d", resp.StatusCode)
	}
	
	logrus.Infof("Successfully ensured JupyterHub user: %s", user.Username)
	return nil
}

// VerifyJWT 验证JWT令牌
// @Summary 验证JWT令牌
// @Description 验证JWT令牌的有效性，用于JupyterHub认证
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body map[string]string true "包含token的请求体"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/verify-token [post]
func (h *UserHandler) VerifyJWT(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	token, exists := req["token"]
	if !exists || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	// 验证JWT token
	claims, err := jwt.ParseToken(token)
	if err != nil {
		logrus.Error("Token validation failed:", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "Invalid token",
		})
		return
	}

	// 获取用户信息
	userID := uint(claims.UserID)
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		logrus.Error("Failed to get user:", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "User not found",
		})
		return
	}

	// 检查用户是否活跃
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid":  false,
			"error":  "User account is inactive",
		})
		return
	}

	// 获取用户角色和权限
	roles, _ := h.rbacService.GetUserRoles(user.ID)
	permissions, _ := h.rbacService.GetUserPermissions(user.ID)

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	permissionKeys := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionKeys[i] = permission.GetPermissionKey()
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"is_active":   user.IsActive,
			"auth_source": user.AuthSource,
			"roles":       roleNames,
			"permissions": permissionKeys,
		},
		"expires_at": claims.ExpiresAt.Time.Format(time.RFC3339),
	})
}

// VerifyTokenSimple 简单的token验证（通过Authorization header）
// @Summary 简单验证JWT令牌
// @Description 通过Authorization header验证JWT令牌，返回用户信息
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/verify [get]
func (h *UserHandler) VerifyTokenSimple(c *gin.Context) {
	// 从context中获取用户信息（由AuthMiddleware设置）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	username, _ := c.Get("username")
	roles, _ := c.Get("roles")
	permissions, _ := c.Get("permissions")

	// 获取完整用户信息
	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		logrus.Error("Failed to get user:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":       true,
		"username":    username,
		"email":       user.Email,
		"roles":       roles,
		"permissions": permissions,
		"is_active":   user.IsActive,
		"user_id":     userID,
	})
}

// RefreshToken 刷新访问令牌
// @Summary 刷新访问令牌
// @Description 刷新用户的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} models.LoginResponse
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /auth/refresh [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// 检查Bearer token格式
	tokenParts := strings.SplitN(authHeader, " ", 2)
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
		return
	}

	tokenString := tokenParts[1]
	
	// 解析现有token
	claims, err := jwt.ParseToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// 获取用户信息
	var user models.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// 检查用户是否仍然活跃
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	// 获取用户角色和权限
	roles, err := h.rbacService.GetUserRoles(user.ID)
	if err != nil {
		logrus.Error("Get user roles error:", err)
		roles = []models.Role{}
	}

	permissions, err := h.rbacService.GetUserPermissions(user.ID)
	if err != nil {
		logrus.Error("Get user permissions error:", err)
		permissions = []models.Permission{}
	}

	// 提取角色名称和权限键
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	permissionKeys := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionKeys[i] = permission.GetPermissionKey()
	}

	// 生成新的JWT token
	newToken, expiresAt, err := jwt.GenerateToken(user.ID, user.Username, roleNames, permissionKeys)
	if err != nil {
		logrus.Error("Generate token error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// 更新Redis会话
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	if err := h.sessionService.CreateSession(&user, newToken, clientIP, userAgent); err != nil {
		logrus.Error("Create session error:", err)
		// 会话创建失败不阻止token刷新，只记录错误
	}

	// 删除旧的会话
	h.sessionService.DeleteSession(tokenString)

	response := models.LoginResponse{
		Token:     newToken,
		User:      user,
		ExpiresAt: expiresAt,
	}

	c.JSON(http.StatusOK, response)
}
