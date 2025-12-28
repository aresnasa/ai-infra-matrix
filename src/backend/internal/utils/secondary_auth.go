package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SecondaryAuthRequest 二次认证请求结构
type SecondaryAuthRequest struct {
	AuthPassword string `json:"auth_password"` // 二次认证密码
}

// SecondaryAuthResponse 二次认证响应结构
type SecondaryAuthResponse struct {
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	RequireAuth  bool   `json:"require_auth,omitempty"`
	AuthRequired string `json:"auth_required,omitempty"` // 标识需要认证的操作类型
}

// SecondaryAuthConfig 二次认证配置
type SecondaryAuthConfig struct {
	Enabled      bool   // 是否启用二次认证
	AuthType     string // 认证类型标识（用于前端显示）
	ErrorMessage string // 认证提示信息
}

// User 用户模型（仅用于密码验证，避免循环依赖）
type authUser struct {
	ID       uint   `gorm:"primarykey"`
	Username string `gorm:"uniqueIndex;size:100"`
	Password string `gorm:"size:255"`
	IsActive bool   `gorm:"default:true"`
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

// VerifyUserPassword 验证用户密码（公共函数，可复用）
// 参数:
//   - username: 用户名
//   - password: 待验证的密码
//
// 返回:
//   - bool: 验证是否成功
//   - error: 错误信息
func VerifyUserPassword(username, password string) (bool, error) {
	if username == "" || password == "" {
		return false, nil
	}

	if authDB == nil {
		return false, nil
	}

	var user authUser
	if err := authDB.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		return false, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return false, nil
	}

	return true, nil
}

// RequireSecondaryAuth 检查并要求二次认证
// 这是一个通用函数，可以在任何需要二次认证的地方调用
// 参数:
//   - c: Gin上下文
//   - config: 二次认证配置
//   - password: 用户提供的密码
//
// 返回:
//   - bool: 是否通过认证（true=通过，false=需要认证或认证失败）
//   - 如果返回false且需要认证，会自动返回JSON响应
func RequireSecondaryAuth(c *gin.Context, config SecondaryAuthConfig, password string) bool {
	// 如果未启用二次认证，直接通过
	if !config.Enabled {
		return true
	}

	// 如果没有提供密码，返回需要认证的响应
	if password == "" {
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

	// 获取当前用户
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success: false,
			Error:   "无法获取当前用户信息",
		})
		return false
	}

	// 验证密码
	verified, err := VerifyUserPassword(username, password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SecondaryAuthResponse{
			Success: false,
			Error:   "密码验证过程出错",
		})
		return false
	}

	if !verified {
		c.JSON(http.StatusUnauthorized, SecondaryAuthResponse{
			Success: false,
			Error:   "密码验证失败",
		})
		return false
	}

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
