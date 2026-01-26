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

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db                     *gorm.DB
	userService            *services.UserService
	rbacService            *services.RBACService
	sessionService         *services.SessionService
	ldapService            *services.LDAPService
	loginProtectionService *services.LoginProtectionService
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		db:                     db,
		userService:            services.NewUserService(),
		rbacService:            services.NewRBACService(db),
		sessionService:         services.NewSessionService(),
		ldapService:            services.NewLDAPService(db),
		loginProtectionService: services.NewLoginProtectionService(),
	}
}

// ValidateLDAP LDAP验证
// @Summary LDAP账户验证
// @Description 验证用户在LDAP中的账户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "LDAP验证信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/validate-ldap [post]
func (h *UserHandler) ValidateLDAP(c *gin.Context) {
	// E2E test bypass: allow fake LDAP validation when explicitly enabled
	if os.Getenv("E2E_ALLOW_FAKE_LDAP") == "true" {
		c.JSON(http.StatusOK, gin.H{
			"message": "E2E LDAP validation bypass enabled",
			"valid":   true,
			"ldap_user": gin.H{
				"username": "e2e-bypass",
			},
		})
		return
	}

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "valid": false})
		return
	}

	// LDAP验证
	ldapUser, err := h.ldapService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		logrus.WithError(err).Error("LDAP validation failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "LDAP验证失败: 用户不存在或密码错误",
			"valid": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "LDAP验证成功",
		"valid":     true,
		"ldap_user": ldapUser,
	})
}

// GetRegistrationConfig 获取注册配置
// @Summary 获取注册配置
// @Description 获取当前系统的注册策略配置
// @Tags 用户管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/registration-config [get]
func (h *UserHandler) GetRegistrationConfig(c *gin.Context) {
	// 检查 REGISTRATION_REQUIRE_INVITATION_CODE 环境变量
	requireInvitationCode := true // 默认值
	val := strings.TrimSpace(strings.ToLower(os.Getenv("REGISTRATION_REQUIRE_INVITATION_CODE")))
	if val == "false" || val == "0" || val == "no" {
		requireInvitationCode = false
	}

	// 检查是否禁用注册
	disableRegistration := false
	disableVal := strings.TrimSpace(strings.ToLower(os.Getenv("DISABLE_REGISTRATION")))
	if disableVal == "true" || disableVal == "1" || disableVal == "yes" {
		disableRegistration = true
	}

	c.JSON(http.StatusOK, gin.H{
		"require_invitation_code": requireInvitationCode,
		"disable_registration":    disableRegistration,
		"allow_approval_mode":     !requireInvitationCode, // 允许审批模式（无邀请码注册需审批）
	})
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

	// E2E test bypass: when enabled AND no invitation code provided, skip LDAP and approval
	// If invitation code is provided, always use the normal flow to properly validate and use the code
	if os.Getenv("E2E_ALLOW_FAKE_LDAP") == "true" && strings.TrimSpace(req.InvitationCode) == "" {
		// Ensure username/email uniqueness via service call path
		user := &models.User{
			Username:      req.Username,
			Email:         req.Email,
			Password:      "", // no local password needed for LDAP-mode users; login uses hybrid/LDAP, but we will also allow local below
			IsActive:      true,
			AuthSource:    "ldap",
			DashboardRole: req.Role,
			RoleTemplate:  req.RoleTemplate,
		}
		// Attempt to set a bcrypt password matching provided value to allow local login in hybrid mode
		if strings.TrimSpace(req.Password) != "" {
			if hpw, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost); err == nil {
				user.Password = string(hpw)
				user.AuthSource = "local" // mark as local-capable for tests
			}
		}
		if err := h.userService.CreateUserDirectly(user); err != nil {
			logrus.WithError(err).Error("E2E bypass user creation failed")
			// 检查是否是用户名/邮箱已存在的错误
			if err.Error() == "username or email already exists" || strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique constraint") {
				c.JSON(http.StatusConflict, gin.H{"error": "用户名或邮箱已存在"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user (e2e)"})
			}
			return
		}
		// Assign role template if provided
		if req.RoleTemplate != "" {
			if err := h.rbacService.AssignRoleTemplateToUser(user.ID, req.RoleTemplate); err != nil {
				logrus.WithError(err).Warn("E2E bypass: assign role template failed")
			}
		}
		c.JSON(http.StatusCreated, user)
		return
	}

	// 获取客户端IP地址
	clientIP := c.ClientIP()

	user, err := h.userService.RegisterWithIP(&req, clientIP)
	if err != nil {
		logrus.Error("Register error:", err)
		errMsg := err.Error()
		if errMsg == "username or email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": errMsg})
		} else if strings.Contains(errMsg, "邀请码") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else if strings.Contains(errMsg, "待审批") {
			c.JSON(http.StatusConflict, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	// 根据用户激活状态返回不同的消息
	if user.IsActive {
		// 邀请码注册成功，用户已激活
		c.JSON(http.StatusCreated, gin.H{
			"message":   "注册成功！您可以直接登录系统。",
			"user":      user,
			"activated": true,
		})
	} else {
		// 普通注册，等待审批 - 返回 202 Accepted
		c.JSON(http.StatusAccepted, gin.H{
			"message":           "注册申请已提交！请等待管理员审批。",
			"requires_approval": true,
			"activated":         false,
		})
	}

	// After user created (if active), try to create corresponding LDAP entry and K8s SA/RBAC (best-effort)
	if user.IsActive {
		go func() {
			// 1) LDAP user provisioning
			if h.ldapService != nil {
				// displayName defaults to username
				if err := h.ldapService.CreateUser(req.Username, req.Password, req.Email, req.Username, req.Department); err != nil {
					logrus.WithError(err).Warn("LDAP user provisioning failed")
				} else {
					logrus.WithField("username", req.Username).Info("LDAP user provisioned")
				}
			}

			// 2) K8s SA/RBAC provisioning
			// Find default cluster from DB if exists (first enabled cluster)
			defer func() { recover() }()
			var cluster models.KubernetesCluster
			if err := database.DB.Where("enabled = ?", true).First(&cluster).Error; err == nil {
				ks := services.NewKubernetesService()
				clientset, cerr := ks.ConnectToCluster(cluster.KubeConfig)
				if cerr != nil {
					logrus.WithError(cerr).Warn("K8s connect for provisioning failed")
					return
				}
				// Namespace strategy: use department if provided, else "users"
				ns := strings.ToLower(strings.TrimSpace(req.Department))
				if ns == "" {
					ns = "users"
				}
				if err := ks.EnsureNamespace(clientset, ns); err != nil {
					logrus.WithError(err).Warn("Ensure namespace failed")
				}
				saName := fmt.Sprintf("user-%s", req.Username)
				if _, err := ks.EnsureServiceAccount(clientset, ns, saName); err != nil {
					logrus.WithError(err).Warn("Ensure ServiceAccount failed")
				}
				// Map role to ClusterRole
				role := strings.ToLower(strings.TrimSpace(req.Role))
				clusterRole := "view"
				if role == "user" {
					clusterRole = "edit"
				}
				if role == "admin" {
					clusterRole = "admin"
				}
				rbName := fmt.Sprintf("%s-%s-rb", saName, clusterRole)
				if _, err := ks.EnsureRoleBinding(clientset, ns, rbName, saName, clusterRole); err != nil {
					logrus.WithError(err).Warn("Ensure RoleBinding failed")
				}
				logrus.WithFields(logrus.Fields{"username": req.Username, "namespace": ns, "cluster_role": clusterRole}).Info("K8s SA/RBAC provisioned")
			}
		}()
	}
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
// @Failure 403 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	if os.Getenv("E2E_ALLOW_FAKE_LDAP") == "true" {
		logrus.Debug("E2E bypass enabled: hybrid authentication will allow local accounts first")
	}

	// 获取客户端IP和请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// 添加调试日志
	logrus.WithFields(logrus.Fields{
		"user_agent": userAgent,
		"origin":     c.GetHeader("Origin"),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"client_ip":  clientIP,
		"request_id": requestID,
	}).Info("Login request received")

	// 检查IP是否被封禁
	if isBlocked, reason, remainingSeconds, err := h.loginProtectionService.CheckIPBlocked(clientIP); err != nil {
		logrus.WithError(err).Error("Failed to check IP block status")
	} else if isBlocked {
		logrus.WithFields(logrus.Fields{
			"ip":                clientIP,
			"reason":            reason,
			"remaining_seconds": remainingSeconds,
		}).Warn("Login blocked due to IP ban")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":             "IP地址已被封禁",
			"reason":            reason,
			"remaining_seconds": remainingSeconds,
			"retry_after":       remainingSeconds,
		})
		return
	}

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithField("error", err).Error("Failed to bind login request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.WithField("username", req.Username).Info("Login attempt for user")

	// 检查账号是否被锁定
	if isLocked, remainingSeconds, err := h.loginProtectionService.CheckAccountLocked(req.Username); err != nil {
		logrus.WithError(err).Error("Failed to check account lock status")
	} else if isLocked {
		logrus.WithFields(logrus.Fields{
			"username":          req.Username,
			"remaining_seconds": remainingSeconds,
		}).Warn("Login blocked due to account lock")

		// 记录登录尝试（账号已锁定）
		h.loginProtectionService.RecordLoginAttempt(clientIP, req.Username, userAgent, false, models.LoginFailureAccountLocked, requestID)

		c.JSON(http.StatusForbidden, gin.H{
			"error":             "账号已被锁定",
			"remaining_seconds": remainingSeconds,
			"retry_after":       remainingSeconds,
		})
		return
	}

	var user *models.User
	var err error

	// 实现混合认证策略
	user, err = h.performHybridAuthentication(&req)
	if err != nil {
		logrus.Error("Authentication error:", err)

		// 记录登录失败
		failureType := models.LoginFailureInvalidPassword
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "不存在") {
			failureType = models.LoginFailureUserNotFound
		} else if strings.Contains(err.Error(), "LDAP") || strings.Contains(err.Error(), "ldap") {
			failureType = models.LoginFailureLDAPError
		} else if strings.Contains(err.Error(), "disabled") || strings.Contains(err.Error(), "禁用") {
			failureType = models.LoginFailureAccountDisabled
		}
		h.loginProtectionService.RecordLoginAttempt(clientIP, req.Username, userAgent, false, failureType, requestID)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 检查用户是否启用了2FA
	var twoFAConfig models.TwoFactorConfig
	if err := h.db.Where("user_id = ? AND enabled = ?", user.ID, true).First(&twoFAConfig).Error; err == nil {
		// 用户启用了2FA，需要进行二次验证
		logrus.WithField("user_id", user.ID).Info("User has 2FA enabled, requiring verification")

		// 生成临时token用于2FA验证
		tempToken := generateTempToken()

		// 存储临时认证信息到Redis或内存（这里简化为存储在内存map中）
		store2FAPendingAuth(tempToken, user.ID, time.Now().Add(5*time.Minute))

		c.JSON(http.StatusOK, gin.H{
			"requires_2fa": true,
			"temp_token":   tempToken,
			"user_id":      user.ID,
			"message":      "Please provide 2FA verification code",
		})
		return
	}

	// 记录登录成功
	h.loginProtectionService.RecordLoginAttempt(clientIP, req.Username, userAgent, true, "", requestID)

	// 没有启用2FA，直接完成登录
	h.completeLogin(c, user)
}

// completeLogin 完成登录流程（生成token等）
func (h *UserHandler) completeLogin(c *gin.Context, user *models.User) {
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

	// Set SSO cookies to simplify downstream auth (read by nginx auth_request)
	// Use SameSite=Lax so normal navigations carry the cookie; mark HttpOnly for security.
	// expiresAt is unix seconds (int64); convert to time.Time for time.Until
	maxAge := int(time.Until(time.Unix(expiresAt, 0)).Seconds())
	if maxAge <= 0 {
		maxAge = 3600
	}
	// Ensure SameSite before setting cookies
	c.SetSameSite(http.SameSiteLaxMode)
	// Primary cookie preferred by gateway
	c.SetCookie("ai_infra_token", token, maxAge, "/", "", false, true)
	// Backward/compat cookies some clients expect
	c.SetCookie("jwt_token", token, maxAge, "/", "", false, true)
	c.SetCookie("auth_token", token, maxAge, "/", "", false, true)

	c.JSON(http.StatusOK, response)
}

// 2FA临时认证存储（简化实现，生产环境应使用Redis）
var pending2FAAuth = make(map[string]pending2FAData)

type pending2FAData struct {
	UserID    uint
	ExpiresAt time.Time
}

func store2FAPendingAuth(token string, userID uint, expiresAt time.Time) {
	pending2FAAuth[token] = pending2FAData{
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
}

func get2FAPendingAuth(token string) (uint, bool) {
	data, exists := pending2FAAuth[token]
	if !exists {
		return 0, false
	}
	if time.Now().After(data.ExpiresAt) {
		delete(pending2FAAuth, token)
		return 0, false
	}
	return data.UserID, true
}

func delete2FAPendingAuth(token string) {
	delete(pending2FAAuth, token)
}

func generateTempToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// Verify2FALogin 验证2FA并完成登录
// @Summary 2FA验证登录
// @Description 验证双因素认证码并完成登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body map[string]string true "2FA验证信息"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/verify-2fa [post]
func (h *UserHandler) Verify2FALogin(c *gin.Context) {
	var req struct {
		TempToken string `json:"temp_token" binding:"required"`
		Code      string `json:"code" binding:"required"`
		Username  string `json:"username"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供临时token和验证码"})
		return
	}

	// 验证临时token
	userID, valid := get2FAPendingAuth(req.TempToken)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "验证已过期，请重新登录"})
		return
	}

	// 获取用户的2FA配置
	var twoFAConfig models.TwoFactorConfig
	if err := h.db.Where("user_id = ? AND enabled = ?", userID, true).First(&twoFAConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA配置不存在"})
		return
	}

	// 验证TOTP码
	valid = verifyTOTPCode(twoFAConfig.Secret, req.Code)
	if !valid {
		// 尝试恢复码验证
		valid = h.verifyRecoveryCode(&twoFAConfig, req.Code)
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "验证码无效"})
		return
	}

	// 删除临时token
	delete2FAPendingAuth(req.TempToken)

	// 更新2FA验证统计
	h.db.Model(&twoFAConfig).Updates(map[string]interface{}{
		"last_verified_at": time.Now(),
		"verify_count":     gorm.Expr("verify_count + 1"),
	})

	// 获取用户信息
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	// 完成登录
	h.completeLogin(c, user)
}

// verifyTOTPCode 验证TOTP码
func verifyTOTPCode(secret, code string) bool {
	// 使用totp库验证
	return totp.Validate(code, secret)
}

// verifyRecoveryCode 验证恢复码
func (h *UserHandler) verifyRecoveryCode(config *models.TwoFactorConfig, code string) bool {
	if config.RecoveryCodes == "" {
		return false
	}

	var codes []string
	if err := json.Unmarshal([]byte(config.RecoveryCodes), &codes); err != nil {
		return false
	}

	for i, c := range codes {
		if c == code {
			// 移除已使用的恢复码
			codes = append(codes[:i], codes[i+1:]...)
			newCodesJSON, _ := json.Marshal(codes)
			h.db.Model(config).Updates(map[string]interface{}{
				"recovery_codes":      string(newCodesJSON),
				"recovery_used_count": gorm.Expr("recovery_used_count + 1"),
			})
			return true
		}
	}
	return false
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
			"email":      ldapUser.Email,
			"last_login": gorm.Expr("NOW()"),
			"is_active":  true,
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
			AuthSource: "ldap",      // 设置认证源为LDAP
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
		if role.Name == "admin" || role.Name == "super-admin" {
			hasAdminRole = true
			break
		}
	}

	if !hasAdminRole {
		// 首先尝试查找super-admin角色
		var adminRole models.Role
		if err := h.db.Where("name = ?", "super-admin").First(&adminRole).Error; err != nil {
			// 如果找不到super-admin，尝试查找admin角色
			if err := h.db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
				logrus.WithError(err).Error("Failed to find admin or super-admin role")
				return err
			}
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
		"users":     users,
		"total":     total,
		"page":      page,
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

	// 检查是否为受保护用户
	if services.IsProtectedUserByID(uint(userID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete protected system user"})
		return
	}

	if err := h.userService.DeleteUser(uint(userID)); err != nil {
		logrus.Error("Delete user error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ToggleUserStatus 切换用户启用/禁用状态（管理员功能）
// @Summary 切换用户状态
// @Description 启用或禁用指定用户（仅管理员），不会影响LDAP数据
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param status body bool true "用户状态"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /users/{id}/status [put]
func (h *UserHandler) ToggleUserStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 防止禁用自己
	currentUserID, _ := middleware.GetCurrentUserID(c)
	if uint(userID) == currentUserID && !req.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot disable yourself"})
		return
	}

	// 检查是否为受保护用户（如 admin），禁止禁用
	if !req.IsActive && services.IsProtectedUserByID(uint(userID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot disable protected system user"})
		return
	}

	// 更新用户状态
	updates := map[string]interface{}{
		"is_active":  req.IsActive,
		"updated_at": time.Now(),
	}

	if err := h.userService.UpdateUser(uint(userID), updates); err != nil {
		logrus.Error("Toggle user status error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle user status"})
		return
	}

	status := "enabled"
	if !req.IsActive {
		status = "disabled"
	}

	logrus.Infof("Admin user %d %s user %d", currentUserID, status, userID)

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("User %s successfully", status),
		"user_id":   userID,
		"is_active": req.IsActive,
	})
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

// SetSecondaryPassword 设置二次密码
// @Summary 设置二次密码
// @Description 用户设置用于敏感操作验证的二次密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SetSecondaryPasswordRequest true "二次密码设置信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/secondary-password [post]
func (h *UserHandler) SetSecondaryPassword(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		CurrentPassword      string `json:"current_password" binding:"required"`             // 当前登录密码（用于验证身份）
		NewSecondaryPassword string `json:"new_secondary_password" binding:"required,min=6"` // 新二次密码
		ConfirmPassword      string `json:"confirm_password" binding:"required"`             // 确认二次密码
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查两次输入的密码是否一致
	if req.NewSecondaryPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "两次输入的密码不一致"})
		return
	}

	// 验证当前登录密码
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前密码验证失败"})
		return
	}

	// 加密并保存二次密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewSecondaryPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	if err := h.db.Model(&user).Update("secondary_password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存二次密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "二次密码设置成功"})
}

// ChangeSecondaryPassword 修改二次密码
// @Summary 修改二次密码
// @Description 用户修改二次密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangeSecondaryPasswordRequest true "二次密码修改信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/secondary-password [put]
func (h *UserHandler) ChangeSecondaryPassword(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		OldSecondaryPassword string `json:"old_secondary_password" binding:"required"`       // 旧二次密码
		NewSecondaryPassword string `json:"new_secondary_password" binding:"required,min=6"` // 新二次密码
		ConfirmPassword      string `json:"confirm_password" binding:"required"`             // 确认二次密码
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查两次输入的密码是否一致
	if req.NewSecondaryPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "两次输入的密码不一致"})
		return
	}

	// 获取用户
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 检查是否已设置二次密码
	if user.SecondaryPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您还未设置二次密码，请先设置"})
		return
	}

	// 验证旧二次密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.SecondaryPassword), []byte(req.OldSecondaryPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "旧二次密码验证失败"})
		return
	}

	// 加密并保存新二次密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewSecondaryPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	if err := h.db.Model(&user).Update("secondary_password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存二次密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "二次密码修改成功"})
}

// GetSecondaryPasswordStatus 获取二次密码状态
// @Summary 获取二次密码状态
// @Description 检查当前用户是否已设置二次密码
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/secondary-password/status [get]
func (h *UserHandler) GetSecondaryPasswordStatus(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"has_secondary_password": user.SecondaryPassword != "",
	})
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

	// Add headers for ProxyAuth integration (used by nginx auth_request)
	c.Header("X-User-Name", user.Username)
	c.Header("X-User-Email", user.Email)
	c.Header("X-User-ID", fmt.Sprintf("%d", user.ID))

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
	// 尝试绑定JSON，如果失败则生成随机密码
	if err := c.ShouldBindJSON(&req); err != nil || req.NewPassword == "" {
		// 生成8位随机密码
		req.NewPassword = generateRandomPassword(8)
	}

	if err := h.userService.AdminResetPassword(uint(userID), &req); err != nil {
		logrus.Error("Admin reset password error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "密码重置成功",
		"password": req.NewPassword,
	})
}

// AdminUpdateRoleTemplate 管理员更新用户的角色模板
// @Summary 管理员更新用户的角色模板
// @Description 管理员为指定用户设置角色模板，并自动分配对应的RBAC角色
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param request body map[string]string true "角色模板信息，如 {\"role_template\": \"data-developer\"}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /users/{id}/role-template [put]
func (h *UserHandler) AdminUpdateRoleTemplate(c *gin.Context) {
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

	var req struct {
		RoleTemplate string `json:"role_template"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.RoleTemplate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_template 不能为空"})
		return
	}

	// 更新用户role_template字段
	updates := map[string]interface{}{
		"role_template": req.RoleTemplate,
		"updated_at":    time.Now(),
	}
	if err := h.userService.UpdateUser(uint(userID), updates); err != nil {
		logrus.WithError(err).Error("Update user role_template failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新角色模板失败"})
		return
	}

	// 分配角色模板对应的RBAC角色（幂等）
	if err := h.rbacService.AssignRoleTemplateToUser(uint(userID), req.RoleTemplate); err != nil {
		logrus.WithError(err).Warn("Assign role template to user failed")
		// 不中断，尽量返回成功并提示
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "角色模板更新成功",
		"user_id":       userID,
		"role_template": req.RoleTemplate,
	})
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
		"success":      true,
		"token":        hubToken,
		"username":     currentUser.Username,
		"expires_at":   time.Now().Add(24 * time.Hour).Unix(), // 24小时有效期
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
	// 获取JupyterHub配置 - 使用环境变量配置
	jupyterHubURL := os.Getenv("JUPYTERHUB_URL")
	if jupyterHubURL == "" {
		jupyterHubURL = "http://ai-infra-matrix-jupyterhub:8000" // K8s服务名
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
			"valid": false,
			"error": "User account is inactive",
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

	// 在SSO校验通过时，后台异步触发统一预配（Gitea / 可选JupyterHub）
	go func(u *models.User) {
		defer func() { _ = recover() }()
		cfg, cfgErr := config.Load()
		if cfgErr != nil {
			logrus.WithError(cfgErr).Debug("Skip SSO provisioning: failed to load config")
			return
		}
		// Gitea 预配（幂等）
		if cfg.Gitea.Enabled {
			gsvc := services.NewGiteaService(cfg)
			if err := gsvc.EnsureUser(*u); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"username": u.Username,
				}).Warn("Gitea ensure user failed during SSO verify")
			} else {
				logrus.WithFields(logrus.Fields{
					"username": u.Username,
				}).Debug("Gitea user ensured during SSO verify")
			}
		}
		// 可选：JupyterHub 用户预配（通过环境开关控制，最佳努力）
		if os.Getenv("JUPYTERHUB_AUTO_PROVISION") == "true" {
			if token, err := h.generateJupyterHubAPIToken(u); err == nil {
				if err := h.ensureJupyterHubUser(u, token); err != nil {
					logrus.WithError(err).WithField("username", u.Username).Debug("JupyterHub ensure user failed (optional)")
				}
			} else {
				logrus.WithError(err).WithField("username", u.Username).Debug("Generate JupyterHub token failed (optional)")
			}
		}
	}(user)

	// 暴露简化的用户信息到响应头，便于反向代理认证（如Nginx auth_request）
	if uname, ok := username.(string); ok {
		c.Header("X-User", uname)
		// 兼容反向代理认证常用头，便于下游（如Gitea）直接复用
		c.Header("X-WEBAUTH-USER", uname)
	}
	if user.Email != "" {
		c.Header("X-Email", user.Email)
		// 兼容反向代理认证常用头
		c.Header("X-WEBAUTH-EMAIL", user.Email)
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

// GetPendingApprovals 获取待审批的注册申请
// @Summary 获取待审批的注册申请
// @Description 获取所有待审批的注册申请列表
// @Tags 管理员功能
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.RegistrationApproval
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/approvals/pending [get]
func (h *UserHandler) GetPendingApprovals(c *gin.Context) {
	approvals, err := h.userService.GetPendingApprovals()
	if err != nil {
		logrus.Error("Get pending approvals error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取待审批申请失败"})
		return
	}

	c.JSON(http.StatusOK, approvals)
}

// ApproveRegistration 批准注册申请
// @Summary 批准注册申请
// @Description 批准指定的注册申请
// @Tags 管理员功能
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "审批ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/approvals/{id}/approve [post]
func (h *UserHandler) ApproveRegistration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审批ID"})
		return
	}

	// 获取当前管理员用户ID
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	err = h.userService.ApproveRegistration(uint(id), adminID.(uint))
	if err != nil {
		logrus.Error("Approve registration error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册申请已批准"})
}

// RejectRegistration 拒绝注册申请
// @Summary 拒绝注册申请
// @Description 拒绝指定的注册申请
// @Tags 管理员功能
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "审批ID"
// @Param request body map[string]string true "拒绝原因"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/approvals/{id}/reject [post]
func (h *UserHandler) RejectRegistration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审批ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前管理员用户ID
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	err = h.userService.RejectRegistration(uint(id), adminID.(uint), req.Reason)
	if err != nil {
		logrus.Error("Reject registration error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册申请已拒绝"})
}

// GetAllUsers 获取所有用户（管理员功能）
// @Summary 获取所有用户
// @Description 分页获取所有用户列表（管理员功能）
// @Tags 管理员功能
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/users [get]
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := h.userService.GetUsers(page, pageSize)
	if err != nil {
		logrus.Error("Get all users error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// generateRandomPassword 生成指定长度的随机密码
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	password := make([]byte, length)
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// 如果随机生成失败，使用固定密码
		return "Reset123!"
	}
	for i := 0; i < length; i++ {
		password[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(password)
}

// GrantUserModulesRequest 授予用户模块权限的请求体
type GrantUserModulesRequest struct {
	Modules []string `json:"modules" binding:"required"` // 模块列表，如 ["saltstack", "ansible", "kubernetes"]
	Verbs   []string `json:"verbs"`                      // 操作权限，如 ["read", "create", "update", "delete", "list"]
}

// GrantUserModules 为用户授予模块权限
// @Summary 授予用户模块权限
// @Description 为指定用户授予特定模块的操作权限
// @Tags 管理员功能
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param request body GrantUserModulesRequest true "模块权限信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/users/{id}/modules [post]
func (h *UserHandler) GrantUserModules(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req GrantUserModulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
		return
	}

	// 默认操作权限
	if len(req.Verbs) == 0 {
		req.Verbs = []string{"read", "create", "update", "delete", "list"}
	}

	// 有效的模块列表
	validModules := map[string]bool{
		"saltstack":   true,
		"ansible":     true,
		"kubernetes":  true,
		"hosts":       true,
		"nightingale": true,
		"audit-logs":  true,
		"projects":    true,
		"variables":   true,
		"tasks":       true,
		"jupyterhub":  true,
		"kafka":       true,
		"kafka-ui":    true,
		"redis":       true,
	}

	// 验证模块
	for _, module := range req.Modules {
		if !validModules[module] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模块: " + module})
			return
		}
	}

	// 验证操作权限
	validVerbs := map[string]bool{
		"create": true,
		"read":   true,
		"update": true,
		"delete": true,
		"list":   true,
	}
	for _, verb := range req.Verbs {
		if !validVerbs[verb] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的操作权限: " + verb})
			return
		}
	}

	// 使用 RBAC 服务为用户授予模块权限
	err = h.rbacService.GrantUserModulePermissions(uint(userID), req.Modules, req.Verbs)
	if err != nil {
		logrus.Error("Grant user module permissions error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "授予模块权限失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "模块权限授予成功",
		"user_id": userID,
		"modules": req.Modules,
		"verbs":   req.Verbs,
	})
}
