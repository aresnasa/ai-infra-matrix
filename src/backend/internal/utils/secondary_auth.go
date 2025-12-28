package utils

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SecondaryAuthRequest 二次认证请求结构
type SecondaryAuthRequest struct {
	AuthPassword string `json:"auth_password"` // 二次认证密码
	AuthCode     string `json:"auth_code"`     // 2FA验证码（TOTP）
}

// SecondaryAuthMethod 二次认证方式
type SecondaryAuthMethod string

const (
	// AuthMethodPassword 密码认证
	AuthMethodPassword SecondaryAuthMethod = "password"
	// AuthMethodTOTP TOTP（2FA）认证
	AuthMethodTOTP SecondaryAuthMethod = "totp"
)

// SecondaryAuthResponse 二次认证响应结构
type SecondaryAuthResponse struct {
	Success      bool                `json:"success"`
	Error        string              `json:"error,omitempty"`
	RequireAuth  bool                `json:"require_auth,omitempty"`
	AuthRequired string              `json:"auth_required,omitempty"` // 标识需要认证的操作类型
	AuthMethod   SecondaryAuthMethod `json:"auth_method,omitempty"`   // 认证方式：password/totp
}

// SecondaryAuthConfig 二次认证配置
type SecondaryAuthConfig struct {
	Enabled      bool                // 是否启用二次认证
	AuthType     string              // 认证类型标识（用于前端显示）
	AuthMethod   SecondaryAuthMethod // 认证方式：password/totp
	ErrorMessage string              // 认证提示信息
}

// User 用户模型（仅用于密码验证，避免循环依赖）
type authUser struct {
	ID                uint   `gorm:"primarykey"`
	Username          string `gorm:"uniqueIndex;size:100"`
	Password          string `gorm:"size:255"`
	SecondaryPassword string `gorm:"size:255"` // 二次认证密码
	IsActive          bool   `gorm:"default:true"`
}

func (authUser) TableName() string {
	return "users"
}

// 模块级变量，存储数据库连接
var authDB *gorm.DB

// InitSecondaryAuth 初始化二次认证模块
// 这个函数应该在应用启动时调用，传入数据库连接
func InitSecondaryAuth(db *gorm.DB) {
	authDB = db
}

// VerifyUserPassword 验证用户登录密码（公共函数，可复用）
// 参数:
//   - username: 用户名
//   - password: 待验证的密码
//
// 返回:
//   - bool: 验证是否成功
//   - error: 错误信息
func VerifyUserPassword(username, password string) (bool, error) {
	if username == "" || password == "" {
		log.Printf("[SecondaryAuth] VerifyUserPassword: username或password为空, username=%s, passwordLen=%d", username, len(password))
		return false, nil
	}

	if authDB == nil {
		log.Printf("[SecondaryAuth] VerifyUserPassword: authDB未初始化")
		return false, nil
	}

	var user authUser
	if err := authDB.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		log.Printf("[SecondaryAuth] VerifyUserPassword: 查询用户失败, username=%s, err=%v", username, err)
		return false, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Printf("[SecondaryAuth] VerifyUserPassword: 密码验证失败, username=%s, err=%v", username, err)
		return false, nil
	}

	log.Printf("[SecondaryAuth] VerifyUserPassword: 密码验证成功, username=%s", username)
	return true, nil
}

// VerifyUserSecondaryPassword 验证用户二次密码（用于敏感操作）
// 参数:
//   - username: 用户名
//   - password: 待验证的二次密码
//
// 返回:
//   - bool: 验证是否成功
//   - bool: 是否已设置二次密码
//   - error: 错误信息
func VerifyUserSecondaryPassword(username, password string) (bool, bool, error) {
	if username == "" || password == "" {
		log.Printf("[SecondaryAuth] VerifyUserSecondaryPassword: username或password为空, username=%s, passwordLen=%d", username, len(password))
		return false, false, nil
	}

	if authDB == nil {
		log.Printf("[SecondaryAuth] VerifyUserSecondaryPassword: authDB未初始化")
		return false, false, nil
	}

	var user authUser
	if err := authDB.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		log.Printf("[SecondaryAuth] VerifyUserSecondaryPassword: 查询用户失败, username=%s, err=%v", username, err)
		return false, false, err
	}

	// 检查是否已设置二次密码
	if user.SecondaryPassword == "" {
		log.Printf("[SecondaryAuth] VerifyUserSecondaryPassword: 用户未设置二次密码, username=%s", username)
		return false, false, nil
	}

	// 验证二次密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.SecondaryPassword), []byte(password)); err != nil {
		log.Printf("[SecondaryAuth] VerifyUserSecondaryPassword: 二次密码验证失败, username=%s, err=%v", username, err)
		return false, true, nil
	}

	log.Printf("[SecondaryAuth] VerifyUserSecondaryPassword: 二次密码验证成功, username=%s", username)
	return true, true, nil
}

// CheckUserHasSecondaryPassword 检查用户是否已设置二次密码
func CheckUserHasSecondaryPassword(username string) (bool, error) {
	if authDB == nil {
		return false, nil
	}

	var user authUser
	if err := authDB.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		return false, err
	}

	return user.SecondaryPassword != "", nil
}

// RequireSecondaryAuth 检查并要求二次认证（使用二次密码）
// 这是一个通用函数，可以在任何需要二次认证的地方调用
// 参数:
//   - c: Gin上下文
//   - config: 二次认证配置
//   - password: 用户提供的二次密码
//
// 返回:
//   - bool: 是否通过认证（true=通过，false=需要认证或认证失败）
//   - 如果返回false且需要认证，会自动返回JSON响应
func RequireSecondaryAuth(c *gin.Context, config SecondaryAuthConfig, password string) bool {
	// 如果未启用二次认证，直接通过
	if !config.Enabled {
		return true
	}

	// 获取当前用户
	username := c.GetString("username")
	if username == "" {
		log.Printf("[SecondaryAuth] RequireSecondaryAuth: 无法获取当前用户信息")
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success: false,
			Error:   "无法获取当前用户信息",
		})
		return false
	}

	// 检查用户是否已设置二次密码
	hasSecondaryPassword, err := CheckUserHasSecondaryPassword(username)
	if err != nil {
		log.Printf("[SecondaryAuth] RequireSecondaryAuth: 检查二次密码状态失败, err=%v", err)
		c.JSON(http.StatusInternalServerError, SecondaryAuthResponse{
			Success: false,
			Error:   "检查二次密码状态失败",
		})
		return false
	}

	if !hasSecondaryPassword {
		log.Printf("[SecondaryAuth] RequireSecondaryAuth: 用户未设置二次密码, username=%s", username)
		c.JSON(http.StatusForbidden, SecondaryAuthResponse{
			Success:      false,
			Error:        "请先在安全设置中设置二次密码，然后才能执行此操作",
			RequireAuth:  true,
			AuthRequired: "setup_secondary_password",
		})
		return false
	}

	// 如果没有提供密码，返回需要认证的响应
	if password == "" {
		log.Printf("[SecondaryAuth] RequireSecondaryAuth: 未提供二次密码，返回需要认证")
		errorMsg := config.ErrorMessage
		if errorMsg == "" {
			errorMsg = "此操作需要二次认证"
		}
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success:      false,
			Error:        errorMsg,
			RequireAuth:  true,
			AuthRequired: config.AuthType,
		})
		return false
	}

	log.Printf("[SecondaryAuth] RequireSecondaryAuth: 开始验证用户二次密码, username=%s, passwordLen=%d", username, len(password))

	// 验证二次密码
	verified, _, err := VerifyUserSecondaryPassword(username, password)
	if err != nil {
		log.Printf("[SecondaryAuth] RequireSecondaryAuth: 二次密码验证过程出错, err=%v", err)
		c.JSON(http.StatusInternalServerError, SecondaryAuthResponse{
			Success: false,
			Error:   "二次密码验证过程出错",
		})
		return false
	}

	if !verified {
		log.Printf("[SecondaryAuth] RequireSecondaryAuth: 二次密码验证失败, username=%s", username)
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success: false,
			Error:   "二次密码验证失败",
		})
		return false
	}

	log.Printf("[SecondaryAuth] RequireSecondaryAuth: 二次密码验证成功, username=%s", username)
	return true
}

// SecondaryAuthMiddleware 二次认证中间件
// 用于需要二次认证的路由组
// 参数:
//   - configFunc: 返回二次认证配置的函数（允许动态获取配置）
func SecondaryAuthMiddleware(configFunc func() SecondaryAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := configFunc()

		// 如果未启用二次认证，直接通过
		if !config.Enabled {
			c.Next()
			return
		}

		// 从请求头或请求体获取密码
		password := c.GetHeader("X-Auth-Password")
		if password == "" {
			// 尝试从请求体获取（需要在具体handler中处理）
			c.Set("secondary_auth_required", true)
			c.Set("secondary_auth_config", config)
		}

		c.Next()
	}
}

// authTwoFactorConfig 2FA配置模型（仅用于验证，避免循环依赖）
type authTwoFactorConfig struct {
	ID             uint       `gorm:"primaryKey"`
	UserID         uint       `gorm:"uniqueIndex;not null"`
	Enabled        bool       `gorm:"default:false"`
	Secret         string     `gorm:"size:256"`
	RecoveryCodes  string     `gorm:"type:text"`
	FailedCount    int        `gorm:"default:0"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
	LastVerifiedAt *time.Time
	VerifyCount    int `gorm:"default:0"`
}

func (authTwoFactorConfig) TableName() string {
	return "two_factor_configs"
}

// GetUserIDByUsername 根据用户名获取用户ID
func GetUserIDByUsername(username string) (uint, error) {
	if authDB == nil {
		return 0, nil
	}

	var user authUser
	if err := authDB.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		return 0, err
	}

	return user.ID, nil
}

// VerifyUser2FA 验证用户的2FA验证码
// 参数:
//   - userID: 用户ID
//   - code: TOTP验证码
//
// 返回:
//   - bool: 验证是否成功
//   - bool: 是否已启用2FA
//   - error: 错误信息
func VerifyUser2FA(userID uint, code string) (bool, bool, error) {
	if code == "" {
		return false, false, nil
	}

	if authDB == nil {
		return false, false, nil
	}

	var config authTwoFactorConfig
	if err := authDB.Where("user_id = ?", userID).First(&config).Error; err != nil {
		// 未找到2FA配置，表示用户未启用2FA
		return false, false, nil
	}

	// 检查是否已启用2FA
	if !config.Enabled {
		return false, false, nil
	}

	// 检查是否被锁定
	if config.LockedUntil != nil && config.LockedUntil.After(time.Now()) {
		return false, true, nil
	}

	// 验证TOTP码
	valid := totp.Validate(code, config.Secret)

	if valid {
		// 验证成功，重置失败计数
		now := time.Now()
		authDB.Model(&config).Updates(map[string]interface{}{
			"failed_count":     0,
			"last_verified_at": now,
			"verify_count":     gorm.Expr("verify_count + 1"),
			"locked_until":     nil,
		})
	} else {
		// 验证失败，增加失败计数
		newFailedCount := config.FailedCount + 1
		updates := map[string]interface{}{
			"failed_count": newFailedCount,
		}

		// 5次失败后锁定5分钟
		if newFailedCount >= 5 {
			lockUntil := time.Now().Add(5 * time.Minute)
			updates["locked_until"] = lockUntil
			updates["failed_count"] = 0
		}

		authDB.Model(&config).Updates(updates)
	}

	return valid, true, nil
}

// RequireSecondaryAuthWith2FA 检查并要求二次认证（支持2FA/TOTP）
// 这是一个增强版的二次认证函数，支持密码认证和2FA认证
// 参数:
//   - c: Gin上下文
//   - config: 二次认证配置
//   - password: 用户提供的密码（如果AuthMethod为password）
//   - authCode: 用户提供的2FA验证码（如果AuthMethod为totp）
//
// 返回:
//   - bool: 是否通过认证（true=通过，false=需要认证或认证失败）
//   - 如果返回false且需要认证，会自动返回JSON响应
func RequireSecondaryAuthWith2FA(c *gin.Context, config SecondaryAuthConfig, password, authCode string) bool {
	// 如果未启用二次认证，直接通过
	if !config.Enabled {
		return true
	}

	// 获取当前用户
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success: false,
			Error:   "无法获取当前用户信息",
		})
		return false
	}

	// 获取用户ID
	userID, err := GetUserIDByUsername(username)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success: false,
			Error:   "无法获取用户信息",
		})
		return false
	}

	// 根据配置的认证方式进行验证
	switch config.AuthMethod {
	case AuthMethodTOTP:
		// 使用2FA/TOTP验证
		return requireTOTPAuth(c, config, userID, authCode)
	case AuthMethodPassword:
		fallthrough
	default:
		// 使用密码验证（保持向后兼容）
		return RequireSecondaryAuth(c, config, password)
	}
}

// requireTOTPAuth 要求TOTP认证
func requireTOTPAuth(c *gin.Context, config SecondaryAuthConfig, userID uint, authCode string) bool {
	// 首先检查用户是否已启用2FA
	var tfConfig authTwoFactorConfig
	if err := authDB.Where("user_id = ?", userID).First(&tfConfig).Error; err != nil || !tfConfig.Enabled {
		// 用户未启用2FA，返回错误提示
		c.JSON(http.StatusForbidden, SecondaryAuthResponse{
			Success:      false,
			Error:        "此操作需要启用2FA（两步验证），请先在安全设置中启用2FA",
			RequireAuth:  true,
			AuthRequired: config.AuthType,
			AuthMethod:   AuthMethodTOTP,
		})
		return false
	}

	// 如果没有提供2FA验证码，返回需要认证的响应
	if authCode == "" {
		errorMsg := config.ErrorMessage
		if errorMsg == "" {
			errorMsg = "此操作需要2FA验证"
		}
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success:      false,
			Error:        errorMsg,
			RequireAuth:  true,
			AuthRequired: config.AuthType,
			AuthMethod:   AuthMethodTOTP,
		})
		return false
	}

	// 检查是否被锁定
	if tfConfig.LockedUntil != nil && tfConfig.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusTooManyRequests, SecondaryAuthResponse{
			Success: false,
			Error:   "2FA验证已被临时锁定，请稍后再试",
		})
		return false
	}

	// 验证2FA码
	verified, _, err := VerifyUser2FA(userID, authCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SecondaryAuthResponse{
			Success: false,
			Error:   "2FA验证过程出错",
		})
		return false
	}

	if !verified {
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success:      false,
			Error:        "2FA验证码无效",
			AuthMethod:   AuthMethodTOTP,
			AuthRequired: config.AuthType,
		})
		return false
	}

	return true
}

// Check2FAEnabled 检查用户是否已启用2FA
func Check2FAEnabled(userID uint) (bool, error) {
	if authDB == nil {
		return false, nil
	}

	var config authTwoFactorConfig
	if err := authDB.Where("user_id = ?", userID).First(&config).Error; err != nil {
		return false, nil
	}

	return config.Enabled, nil
}
