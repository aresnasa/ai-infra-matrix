package middleware

import (
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// IPWhitelistConfig IP白名单配置
type IPWhitelistConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedIPs       []string `json:"allowed_ips"`        // 允许的IP地址列表
	AllowedCIDRs     []string `json:"allowed_cidrs"`      // 允许的CIDR网段列表
	AdminRestricted  bool     `json:"admin_restricted"`   // 是否限制admin账号只能从白名单IP登录
	TrustProxyHeader bool     `json:"trust_proxy_header"` // 是否信任X-Forwarded-For等代理头
}

var (
	ipWhitelistConfig *IPWhitelistConfig
	ipWhitelistMu     sync.RWMutex
	parsedCIDRs       []*net.IPNet
)

// init 初始化IP白名单配置
func init() {
	loadIPWhitelistConfig()
}

// loadIPWhitelistConfig 从环境变量加载IP白名单配置
func loadIPWhitelistConfig() {
	ipWhitelistMu.Lock()
	defer ipWhitelistMu.Unlock()

	config := &IPWhitelistConfig{
		Enabled:          os.Getenv("ADMIN_IP_WHITELIST_ENABLED") == "true",
		AdminRestricted:  os.Getenv("ADMIN_IP_RESTRICTED") != "false", // 默认为true
		TrustProxyHeader: os.Getenv("TRUST_PROXY_HEADER") == "true",
	}

	// 解析允许的IP地址列表
	if ips := os.Getenv("ADMIN_ALLOWED_IPS"); ips != "" {
		config.AllowedIPs = strings.Split(ips, ",")
		for i, ip := range config.AllowedIPs {
			config.AllowedIPs[i] = strings.TrimSpace(ip)
		}
	}

	// 解析允许的CIDR网段列表
	if cidrs := os.Getenv("ADMIN_ALLOWED_CIDRS"); cidrs != "" {
		config.AllowedCIDRs = strings.Split(cidrs, ",")
		for i, cidr := range config.AllowedCIDRs {
			config.AllowedCIDRs[i] = strings.TrimSpace(cidr)
		}
	}

	// 默认添加常见内网网段
	defaultCIDRs := []string{
		"10.0.0.0/8",     // A类私有地址
		"172.16.0.0/12",  // B类私有地址
		"192.168.0.0/16", // C类私有地址
		"127.0.0.0/8",    // 本地回环
		"::1/128",        // IPv6本地回环
		"fc00::/7",       // IPv6唯一本地地址
		"fe80::/10",      // IPv6链路本地地址
	}

	// 如果没有配置CIDR，使用默认内网网段
	if len(config.AllowedCIDRs) == 0 {
		config.AllowedCIDRs = defaultCIDRs
	}

	// 预解析CIDR网段
	parsedCIDRs = make([]*net.IPNet, 0, len(config.AllowedCIDRs))
	for _, cidrStr := range config.AllowedCIDRs {
		_, cidr, err := net.ParseCIDR(cidrStr)
		if err != nil {
			logrus.Warnf("Invalid CIDR format: %s, error: %v", cidrStr, err)
			continue
		}
		parsedCIDRs = append(parsedCIDRs, cidr)
	}

	ipWhitelistConfig = config

	logrus.WithFields(logrus.Fields{
		"enabled":          config.Enabled,
		"admin_restricted": config.AdminRestricted,
		"allowed_ips":      config.AllowedIPs,
		"allowed_cidrs":    config.AllowedCIDRs,
	}).Info("IP whitelist configuration loaded")
}

// ReloadIPWhitelistConfig 重新加载IP白名单配置
func ReloadIPWhitelistConfig() {
	loadIPWhitelistConfig()
}

// GetIPWhitelistConfig 获取当前IP白名单配置
func GetIPWhitelistConfig() *IPWhitelistConfig {
	ipWhitelistMu.RLock()
	defer ipWhitelistMu.RUnlock()
	return ipWhitelistConfig
}

// isIPAllowed 检查IP是否在白名单中
func isIPAllowed(ip string) bool {
	ipWhitelistMu.RLock()
	defer ipWhitelistMu.RUnlock()

	// 解析IP地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		logrus.Warnf("Invalid IP address: %s", ip)
		return false
	}

	// 检查是否在允许的IP列表中
	for _, allowedIP := range ipWhitelistConfig.AllowedIPs {
		if ip == allowedIP {
			return true
		}
	}

	// 检查是否在允许的CIDR网段中
	for _, cidr := range parsedCIDRs {
		if cidr.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// getClientIP 获取客户端真实IP
func getClientIP(c *gin.Context) string {
	ipWhitelistMu.RLock()
	trustProxy := ipWhitelistConfig.TrustProxyHeader
	ipWhitelistMu.RUnlock()

	if trustProxy {
		// 按优先级检查代理头
		proxyHeaders := []string{
			"X-Real-IP",
			"X-Forwarded-For",
			"CF-Connecting-IP", // Cloudflare
			"True-Client-IP",   // Akamai
		}

		for _, header := range proxyHeaders {
			if ip := c.GetHeader(header); ip != "" {
				// X-Forwarded-For可能包含多个IP，取第一个
				if header == "X-Forwarded-For" {
					ips := strings.Split(ip, ",")
					if len(ips) > 0 {
						return strings.TrimSpace(ips[0])
					}
				}
				return strings.TrimSpace(ip)
			}
		}
	}

	return c.ClientIP()
}

// AdminIPWhitelistMiddleware admin账号IP白名单验证中间件
// 此中间件需要在AuthMiddlewareWithSession之后使用
func AdminIPWhitelistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ipWhitelistMu.RLock()
		enabled := ipWhitelistConfig.Enabled
		adminRestricted := ipWhitelistConfig.AdminRestricted
		ipWhitelistMu.RUnlock()

		// 如果未启用IP白名单，直接放行
		if !enabled {
			c.Next()
			return
		}

		// 获取当前用户的角色
		roles, exists := c.Get("roles")
		if !exists {
			c.Next()
			return
		}

		roleList, ok := roles.([]string)
		if !ok {
			c.Next()
			return
		}

		// 检查是否是admin角色
		isAdmin := false
		for _, role := range roleList {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		// 如果是admin角色且启用了admin限制，检查IP
		if isAdmin && adminRestricted {
			clientIP := getClientIP(c)
			if !isIPAllowed(clientIP) {
				logrus.WithFields(logrus.Fields{
					"client_ip": clientIP,
					"username":  c.GetString("username"),
					"path":      c.Request.URL.Path,
				}).Warn("Admin access denied: IP not in whitelist")

				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Admin access denied from this IP address",
					"code":    "IP_NOT_ALLOWED",
					"message": "管理员账号只能从内网IP登录",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// AdminLoginIPCheckMiddleware admin登录IP检查中间件
// 此中间件用于登录接口，在验证用户名密码之前检查IP
func AdminLoginIPCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ipWhitelistMu.RLock()
		enabled := ipWhitelistConfig.Enabled
		adminRestricted := ipWhitelistConfig.AdminRestricted
		ipWhitelistMu.RUnlock()

		// 如果未启用IP白名单，直接放行
		if !enabled || !adminRestricted {
			c.Next()
			return
		}

		// 尝试获取登录用户名
		var loginReq struct {
			Username string `json:"username"`
		}

		// 保存原始body，因为可能需要后续绑定
		bodyBytes, err := c.GetRawData()
		if err != nil {
			c.Next()
			return
		}

		// 复制一份供后续使用
		c.Request.Body = newReadCloser(bodyBytes)

		// 尝试解析用户名
		if err := c.ShouldBindJSON(&loginReq); err == nil && loginReq.Username == "admin" {
			clientIP := getClientIP(c)
			if !isIPAllowed(clientIP) {
				logrus.WithFields(logrus.Fields{
					"client_ip": clientIP,
					"username":  loginReq.Username,
					"path":      c.Request.URL.Path,
				}).Warn("Admin login denied: IP not in whitelist")

				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Admin login denied from this IP address",
					"code":    "IP_NOT_ALLOWED",
					"message": "管理员账号只能从内网IP登录",
				})
				c.Abort()
				return
			}
		}

		// 重置body供后续handler使用
		c.Request.Body = newReadCloser(bodyBytes)
		c.Next()
	}
}

// IPWhitelistCheckMiddleware 通用IP白名单检查中间件
// 用于保护敏感路由
func IPWhitelistCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ipWhitelistMu.RLock()
		enabled := ipWhitelistConfig.Enabled
		ipWhitelistMu.RUnlock()

		if !enabled {
			c.Next()
			return
		}

		clientIP := getClientIP(c)
		if !isIPAllowed(clientIP) {
			logrus.WithFields(logrus.Fields{
				"client_ip": clientIP,
				"path":      c.Request.URL.Path,
			}).Warn("Access denied: IP not in whitelist")

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied from this IP address",
				"code":    "IP_NOT_ALLOWED",
				"message": "该功能只允许从内网IP访问",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimitMiddleware IP访问频率限制中间件
// 防止暴力破解
type ipRateLimiter struct {
	attempts map[string]*rateLimitEntry
	mu       sync.RWMutex
}

type rateLimitEntry struct {
	count      int
	lastAccess time.Time
	blocked    bool
	blockedAt  time.Time
}

var (
	rateLimiter = &ipRateLimiter{
		attempts: make(map[string]*rateLimitEntry),
	}
	maxAttempts    = 5                // 最大尝试次数
	blockDuration  = 15 * time.Minute // 封锁时间
	windowDuration = 5 * time.Minute  // 时间窗口
)

func init() {
	// 从环境变量读取配置
	if val := os.Getenv("LOGIN_MAX_ATTEMPTS"); val != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && n > 0 {
			maxAttempts = n
		}
	}
	if val := os.Getenv("LOGIN_BLOCK_DURATION_MINUTES"); val != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && n > 0 {
			blockDuration = time.Duration(n) * time.Minute
		}
	}

	// 启动清理协程
	go rateLimiter.cleanupLoop()
}

func (rl *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

func (rl *ipRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, entry := range rl.attempts {
		// 清理过期的记录
		if entry.blocked && now.Sub(entry.blockedAt) > blockDuration {
			delete(rl.attempts, ip)
		} else if !entry.blocked && now.Sub(entry.lastAccess) > windowDuration {
			delete(rl.attempts, ip)
		}
	}
}

func (rl *ipRateLimiter) recordAttempt(ip string, success bool) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.attempts[ip]

	if !exists {
		entry = &rateLimitEntry{}
		rl.attempts[ip] = entry
	}

	// 检查是否仍在封锁期
	if entry.blocked {
		if now.Sub(entry.blockedAt) > blockDuration {
			// 封锁期已过，重置
			entry.blocked = false
			entry.count = 0
		} else {
			return true // 仍在封锁期
		}
	}

	// 检查时间窗口是否已过
	if now.Sub(entry.lastAccess) > windowDuration {
		entry.count = 0
	}

	entry.lastAccess = now

	if success {
		// 登录成功，重置计数
		entry.count = 0
		return false
	}

	// 登录失败，增加计数
	entry.count++
	if entry.count >= maxAttempts {
		entry.blocked = true
		entry.blockedAt = now
		return true
	}

	return false
}

func (rl *ipRateLimiter) isBlocked(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.attempts[ip]
	if !exists {
		return false
	}

	if entry.blocked && time.Since(entry.blockedAt) < blockDuration {
		return true
	}

	return false
}

// LoginRateLimitMiddleware 登录频率限制中间件
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)

		if rateLimiter.isBlocked(clientIP) {
			logrus.WithFields(logrus.Fields{
				"client_ip": clientIP,
			}).Warn("Login blocked: too many failed attempts")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many login attempts",
				"code":    "RATE_LIMITED",
				"message": "登录尝试次数过多，请稍后再试",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RecordLoginAttempt 记录登录尝试结果
func RecordLoginAttempt(c *gin.Context, success bool) {
	clientIP := getClientIP(c)
	rateLimiter.recordAttempt(clientIP, success)
}

// readCloser 用于重置request body
type readCloser struct {
	data []byte
	pos  int
}

func newReadCloser(data []byte) *readCloser {
	return &readCloser{data: data}
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	if rc.pos >= len(rc.data) {
		return 0, io.EOF
	}
	n = copy(p, rc.data[rc.pos:])
	rc.pos += n
	return n, nil
}

func (rc *readCloser) Close() error {
	return nil
}
