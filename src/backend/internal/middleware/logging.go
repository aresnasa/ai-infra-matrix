package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingMiddleware 增强的日志中间件，支持不同级别的日志记录
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// 根据状态码和响应时间确定日志级别
			fields := logrus.Fields{
				"method":     param.Method,
				"path":       param.Path,
				"status":     param.StatusCode,
				"latency":    param.Latency,
				"client_ip":  param.ClientIP,
				"user_agent": param.Request.UserAgent(),
			}

			// 根据状态码设置不同的日志级别
			switch {
			case param.StatusCode >= 500:
				logrus.WithFields(fields).Error("Server error response")
			case param.StatusCode >= 400:
				logrus.WithFields(fields).Warn("Client error response")
			case param.Latency > time.Second:
				logrus.WithFields(fields).Warn("Slow response")
			case param.StatusCode >= 300:
				logrus.WithFields(fields).Info("Redirect response")
			default:
				logrus.WithFields(fields).Debug("Successful response")
			}

			return ""
		},
		Output: gin.DefaultWriter,
	})
}

// RequestIDMiddleware 为每个请求添加唯一ID，便于日志追踪
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成简单的请求ID（实际使用中可以用UUID）
		requestID := time.Now().UnixNano()
		c.Set("request_id", requestID)
		
		logrus.WithField("request_id", requestID).Debug("Request started")
		
		c.Next()
		
		logrus.WithField("request_id", requestID).Debug("Request completed")
	}
}
