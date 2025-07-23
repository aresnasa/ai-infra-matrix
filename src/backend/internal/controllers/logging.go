package controllers

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingController 处理日志级别相关的API
type LoggingController struct {
	mu sync.RWMutex
}

// NewLoggingController 创建新的日志控制器
func NewLoggingController() *LoggingController {
	return &LoggingController{}
}

// LogLevelRequest 日志级别设置请求
type LogLevelRequest struct {
	Level string `json:"level" binding:"required" example:"debug"`
}

// LogLevelResponse 日志级别响应
type LogLevelResponse struct {
	CurrentLevel string   `json:"current_level" example:"info"`
	ValidLevels  []string `json:"valid_levels"`
	Message      string   `json:"message"`
}

// GetLogLevel 获取当前日志级别
// @Summary 获取当前日志级别
// @Description 返回当前系统的日志级别和所有可用级别
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} LogLevelResponse
// @Router /admin/logging/level [get]
func (lc *LoggingController) GetLogLevel(c *gin.Context) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	currentLevel := logrus.GetLevel()
	validLevels := []string{"panic", "fatal", "error", "warn", "info", "debug", "trace"}

	response := LogLevelResponse{
		CurrentLevel: currentLevel.String(),
		ValidLevels:  validLevels,
		Message:      "Current log level retrieved successfully",
	}

	logrus.WithField("current_level", currentLevel.String()).Debug("Log level requested")
	c.JSON(http.StatusOK, response)
}

// SetLogLevel 设置日志级别
// @Summary 设置日志级别
// @Description 动态修改系统的日志级别（需要管理员权限）
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body LogLevelRequest true "日志级别设置"
// @Success 200 {object} LogLevelResponse
// @Failure 400 {object} gin.H "无效请求"
// @Failure 401 {object} gin.H "未授权"
// @Failure 403 {object} gin.H "权限不足"
// @Router /admin/logging/level [post]
func (lc *LoggingController) SetLogLevel(c *gin.Context) {
	var req LogLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Invalid log level request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 验证日志级别
	newLevel, err := logrus.ParseLevel(req.Level)
	if err != nil {
		logrus.WithField("requested_level", req.Level).WithError(err).Error("Invalid log level requested")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid log level",
			"valid_levels": []string{"panic", "fatal", "error", "warn", "info", "debug", "trace"},
			"details": err.Error(),
		})
		return
	}

	// 记录原来的级别
	oldLevel := logrus.GetLevel()

	// 设置新的日志级别
	logrus.SetLevel(newLevel)

	// 记录级别变更
	logrus.WithFields(logrus.Fields{
		"old_level": oldLevel.String(),
		"new_level": newLevel.String(),
		"changed_by": c.GetString("username"),
	}).Info("Log level changed")

	response := LogLevelResponse{
		CurrentLevel: newLevel.String(),
		ValidLevels:  []string{"panic", "fatal", "error", "warn", "info", "debug", "trace"},
		Message:      "Log level updated successfully",
	}

	c.JSON(http.StatusOK, response)
}

// TestLogLevels 测试不同日志级别的输出
// @Summary 测试日志级别
// @Description 输出所有级别的测试日志，用于验证当前日志级别设置
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H "未授权"
// @Failure 403 {object} gin.H "权限不足"
// @Router /admin/logging/test [post]
func (lc *LoggingController) TestLogLevels(c *gin.Context) {
	username := c.GetString("username")
	
	// 测试所有日志级别
	logrus.WithField("test_user", username).Trace("This is a TRACE level message")
	logrus.WithField("test_user", username).Debug("This is a DEBUG level message")
	logrus.WithField("test_user", username).Info("This is an INFO level message")
	logrus.WithField("test_user", username).Warn("This is a WARN level message")
	logrus.WithField("test_user", username).Error("This is an ERROR level message")
	// 注意：不测试Fatal和Panic级别，因为它们会导致程序退出

	currentLevel := logrus.GetLevel()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Test log messages sent at all levels",
		"current_level": currentLevel.String(),
		"note": "Check the application logs to see which messages are actually output based on current log level",
		"levels_tested": []string{"trace", "debug", "info", "warn", "error"},
	})
}

// GetLoggingInfo 获取日志配置信息
// @Summary 获取日志配置信息
// @Description 返回当前日志系统的详细配置信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /admin/logging/info [get]
func (lc *LoggingController) GetLoggingInfo(c *gin.Context) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	currentLevel := logrus.GetLevel()
	
	// 获取格式化器信息
	formatter := logrus.StandardLogger().Formatter
	var formatterType string
	switch formatter.(type) {
	case *logrus.JSONFormatter:
		formatterType = "JSON"
	case *logrus.TextFormatter:
		formatterType = "Text"
	default:
		formatterType = "Unknown"
	}

	info := gin.H{
		"current_level": currentLevel.String(),
		"formatter": formatterType,
		"valid_levels": []string{"panic", "fatal", "error", "warn", "info", "debug", "trace"},
		"level_descriptions": gin.H{
			"panic": "Highest level of severity. Logs and then calls panic with the message passed in",
			"fatal": "Logs and then calls os.Exit(1). Cannot be caught",
			"error": "Error conditions that should definitely be noted",
			"warn":  "Warning conditions that should be noted",
			"info":  "General operational entries about what's happening inside the application",
			"debug": "Usually only enabled when debugging. Very verbose logging",
			"trace": "Designates finer-grained informational events than the Debug",
		},
		"current_level_numeric": int(currentLevel),
		"notes": []string{
			"Lower numeric values represent higher severity",
			"Setting a level enables that level and all higher severity levels",
			"For example, setting 'warn' will show warn, error, fatal, and panic messages",
		},
	}

	c.JSON(http.StatusOK, info)
}
