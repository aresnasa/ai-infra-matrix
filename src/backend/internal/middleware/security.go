package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// SQL注入检测正则表达式
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|delete\s+from|update\s+.*\s+set|drop\s+table|create\s+table)`),
	regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|script|javascript:|<script)`),
	regexp.MustCompile(`(?i)(--|\#|\/\*|\*\/|;|'|"|\||&|\$)`),
	regexp.MustCompile(`(?i)(0x[0-9a-f]+|char\(|concat\(|load_file\()`),
}

// XSS检测正则表达式
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`),
	regexp.MustCompile(`(?i)<iframe[^>]*>`),
	regexp.MustCompile(`(?i)<object[^>]*>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
}

// 路径遍历检测
var pathTraversalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.\./`),
	regexp.MustCompile(`\.\.\`),
	regexp.MustCompile(`%2e%2e/`),
	regexp.MustCompile(`%2e%2e%5c`),
}

// SQLInjectionDefense 中间件防御SQL注入攻击
func SQLInjectionDefense() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查所有查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if isSQLInjection(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "Invalid input detected",
						"message": "Suspicious SQL pattern detected in query parameters",
						"param":   key,
					})
					c.Abort()
					return
				}
			}
		}

		// 检查POST/PUT请求体中的JSON字段
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			// 对于application/json，需要在后续处理中进行检查
			// 这里我们可以检查原始body，但更好的方式是在绑定后检查
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "application/x-www-form-urlencoded") {
				if err := c.Request.ParseForm(); err == nil {
					for key, values := range c.Request.PostForm {
						for _, value := range values {
							if isSQLInjection(value) {
								c.JSON(http.StatusBadRequest, gin.H{
									"error":   "Invalid input detected",
									"message": "Suspicious SQL pattern detected in form data",
									"param":   key,
								})
								c.Abort()
								return
							}
						}
					}
				}
			}
		}

		c.Next()
	}
}

// XSSDefense 中间件防御XSS攻击
func XSSDefense() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查所有查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if isXSS(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "Invalid input detected",
						"message": "Suspicious XSS pattern detected in query parameters",
						"param":   key,
					})
					c.Abort()
					return
				}
			}
		}

		// 添加安全响应头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';")

		c.Next()
	}
}

// PathTraversalDefense 防御路径遍历攻击
func PathTraversalDefense() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		for _, pattern := range pathTraversalPatterns {
			if pattern.MatchString(path) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid path detected",
					"message": "Path traversal attempt detected",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(requestsPerSecond float64, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// IPRateLimitMiddleware IP级别的速率限制
func IPRateLimitMiddleware(requestsPerMinute float64) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	clients := make(map[string]*client)

	// 清理过期的客户端（每分钟清理一次）
	go func() {
		for {
			time.Sleep(time.Minute)
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if _, exists := clients[ip]; !exists {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(requestsPerMinute/60.0), 1),
			}
		}

		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from this IP address",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecureHeaders 添加安全响应头
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestSizeLimit 限制请求体大小
func RequestSizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// SanitizeLogMiddleware 日志脱敏中间件
func SanitizeLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 移除敏感请求头
		sensitiveHeaders := []string{"Authorization", "Cookie", "X-Auth-Token", "Api-Key"}

		sanitizedHeaders := make(map[string]string)
		for k, v := range c.Request.Header {
			isSensitive := false
			for _, sh := range sensitiveHeaders {
				if strings.EqualFold(k, sh) {
					sanitizedHeaders[k] = "[REDACTED]"
					isSensitive = true
					break
				}
			}
			if !isSensitive {
				sanitizedHeaders[k] = strings.Join(v, ",")
			}
		}

		// 在日志中使用脱敏后的头信息
		c.Set("sanitized_headers", sanitizedHeaders)

		c.Next()
	}
}

// Helper functions

func isSQLInjection(input string) bool {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func isXSS(input string) bool {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ValidateInput 通用输入验证函数
func ValidateInput(input string, maxLength int) error {
	// 检查长度
	if len(input) > maxLength {
		return fmt.Errorf("input too long: max %d characters", maxLength)
	}

	// 检查SQL注入
	if isSQLInjection(input) {
		return fmt.Errorf("invalid input - SQL injection pattern detected")
	}

	// 检查XSS
	if isXSS(input) {
		return fmt.Errorf("invalid input - XSS pattern detected")
	}

	return nil
}
