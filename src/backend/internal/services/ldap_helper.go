package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/go-ldap/ldap/v3"
)

// LDAPConnectionHelper LDAP连接助手，提供Windows兼容性支持
type LDAPConnectionHelper struct{}

// NewLDAPConnectionHelper 创建LDAP连接助手
func NewLDAPConnectionHelper() *LDAPConnectionHelper {
	return &LDAPConnectionHelper{}
}

// TestConnectionWithRetry 带重试的连接测试
func (h *LDAPConnectionHelper) TestConnectionWithRetry(config *models.LDAPConfig, maxRetries int) *models.LDAPTestResponse {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		result := h.testConnectionOnce(config)
		if result.Success {
			if i > 0 {
				result.Details += fmt.Sprintf(" (成功于第%d次重试)", i+1)
			}
			return result
		}
		lastErr = fmt.Errorf(result.Details)
		
		// 短暂等待后重试
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	
	return &models.LDAPTestResponse{
		Success: false,
		Message: fmt.Sprintf("连接失败，已重试%d次", maxRetries),
		Details: fmt.Sprintf("最后错误: %v", lastErr),
	}
}

// testConnectionOnce 单次连接测试
func (h *LDAPConnectionHelper) testConnectionOnce(config *models.LDAPConfig) *models.LDAPTestResponse {
	// Windows特殊处理
	if runtime.GOOS == "windows" {
		return h.testConnectionWindows(config)
	}
	
	// Unix/Linux处理
	return h.testConnectionUnix(config)
}

// testConnectionWindows Windows专用连接测试
func (h *LDAPConnectionHelper) testConnectionWindows(config *models.LDAPConfig) *models.LDAPTestResponse {
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)
	
	// Windows环境下使用更长的超时时间
	timeout := time.Duration(15) * time.Second
	
	var conn *ldap.Conn
	
	if config.UseSSL {
		// Windows AD通常需要更宽松的TLS设置
		tlsConfig := &tls.Config{
			ServerName:         config.Server,
			InsecureSkipVerify: true, // Windows AD环境经常需要跳过证书验证
			MinVersion:         tls.VersionTLS10, // 兼容旧版本
		}
		
		dialer := &net.Dialer{
			Timeout: timeout,
		}
		
		netConn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if dialErr != nil {
			return &models.LDAPTestResponse{
				Success: false,
				Message: "SSL连接失败",
				Details: fmt.Sprintf("Windows SSL连接错误: %v", dialErr),
			}
		}
		
		conn = ldap.NewConn(netConn, true)
		conn.Start()
	} else {
		// 普通连接，使用超时控制
		dialer := &net.Dialer{
			Timeout: timeout,
		}
		
		netConn, dialErr := dialer.Dial("tcp", addr)
		if dialErr != nil {
			return &models.LDAPTestResponse{
				Success: false,
				Message: "网络连接失败",
				Details: fmt.Sprintf("Windows网络连接错误: %v (请检查防火墙设置)", dialErr),
			}
		}
		
		conn = ldap.NewConn(netConn, false)
		conn.Start()
	}
	
	defer conn.Close()
	conn.SetTimeout(timeout)
	
	// Windows AD绑定测试
	return h.testBindAndSearch(conn, config, "Windows环境")
}

// testConnectionUnix Unix/Linux专用连接测试
func (h *LDAPConnectionHelper) testConnectionUnix(config *models.LDAPConfig) *models.LDAPTestResponse {
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)
	timeout := time.Duration(10) * time.Second
	
	var conn *ldap.Conn
	var err error
	
	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			ServerName:         config.Server,
			InsecureSkipVerify: false, // Unix环境通常证书验证更严格
		})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}
	
	if err != nil {
		return &models.LDAPTestResponse{
			Success: false,
			Message: "连接失败",
			Details: fmt.Sprintf("Unix连接错误: %v", err),
		}
	}
	
	defer conn.Close()
	conn.SetTimeout(timeout)
	
	return h.testBindAndSearch(conn, config, "Unix/Linux环境")
}

// testBindAndSearch 测试绑定和搜索
func (h *LDAPConnectionHelper) testBindAndSearch(conn *ldap.Conn, config *models.LDAPConfig, env string) *models.LDAPTestResponse {
	// 测试绑定
	if config.BindDN != "" {
		err := conn.Bind(config.BindDN, config.BindPassword)
		if err != nil {
			// 特殊处理Windows AD常见错误
			if strings.Contains(err.Error(), "49") {
				return &models.LDAPTestResponse{
					Success: false,
					Message: "身份验证失败",
					Details: fmt.Sprintf("%s - 用户名或密码错误 (错误码49): %v", env, err),
				}
			}
			return &models.LDAPTestResponse{
				Success: false,
				Message: "绑定失败",
				Details: fmt.Sprintf("%s - 绑定错误: %v", env, err),
			}
		}
	}
	
	// 测试BaseDN搜索
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		10, // 10秒超时
		false,
		"(objectClass=*)",
		[]string{"dn", "objectClass"},
		nil,
	)
	
	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return &models.LDAPTestResponse{
			Success: false,
			Message: "搜索测试失败",
			Details: fmt.Sprintf("%s - BaseDN搜索错误: %v (请检查BaseDN格式)", env, err),
		}
	}
	
	// 成功响应
	details := fmt.Sprintf("%s - 连接成功到 %s:%d", env, config.Server, config.Port)
	if config.UseSSL {
		details += " (SSL/TLS)"
	}
	if len(searchResult.Entries) > 0 {
		details += fmt.Sprintf(", BaseDN有效: %s", config.BaseDN)
	}
	
	return &models.LDAPTestResponse{
		Success: true,
		Message: "连接测试成功",
		Details: details,
	}
}

// GetRecommendedSettings 获取推荐的Windows AD设置
func (h *LDAPConnectionHelper) GetRecommendedSettings(serverType string) map[string]interface{} {
	settings := make(map[string]interface{})
	
	switch strings.ToLower(serverType) {
	case "windows", "ad", "activedirectory":
		settings["port"] = 389
		settings["use_ssl"] = false
		settings["ssl_port"] = 636
		settings["timeout"] = 15
		settings["base_dn_example"] = "dc=example,dc=com"
		settings["bind_dn_example"] = "cn=ldapuser,cn=Users,dc=example,dc=com"
		settings["user_filter"] = "(&(objectClass=user)(sAMAccountName=%s))"
		settings["group_filter"] = "(&(objectClass=group)(cn=%s))"
		
	case "openldap":
		settings["port"] = 389
		settings["use_ssl"] = false
		settings["ssl_port"] = 636
		settings["timeout"] = 10
		settings["base_dn_example"] = "dc=example,dc=org"
		settings["bind_dn_example"] = "cn=admin,dc=example,dc=org"
		settings["user_filter"] = "(&(objectClass=person)(uid=%s))"
		settings["group_filter"] = "(&(objectClass=groupOfNames)(cn=%s))"
		
	default:
		settings["port"] = 389
		settings["use_ssl"] = false
		settings["timeout"] = 10
	}
	
	return settings
}