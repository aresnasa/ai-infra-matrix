package utils

import (
	"regexp"
	"strings"
)

// MaskConfig 脱敏配置
type MaskConfig struct {
	// 保留前几位
	KeepPrefix int
	// 保留后几位
	KeepSuffix int
	// 掩码字符
	MaskChar string
	// 最小长度（小于此长度全部脱敏）
	MinLength int
}

// DefaultPasswordMask 默认密码脱敏配置（保留前2后2位）
var DefaultPasswordMask = MaskConfig{
	KeepPrefix: 2,
	KeepSuffix: 2,
	MaskChar:   "*",
	MinLength:  6,
}

// DefaultUsernameMask 默认用户名脱敏配置（保留前3后1位）
var DefaultUsernameMask = MaskConfig{
	KeepPrefix: 3,
	KeepSuffix: 1,
	MaskChar:   "*",
	MinLength:  4,
}

// DefaultKeyMask 默认密钥脱敏配置（保留前8后8位）
var DefaultKeyMask = MaskConfig{
	KeepPrefix: 8,
	KeepSuffix: 8,
	MaskChar:   "*",
	MinLength:  20,
}

// DefaultTokenMask 默认Token脱敏配置（保留前4后4位）
var DefaultTokenMask = MaskConfig{
	KeepPrefix: 4,
	KeepSuffix: 4,
	MaskChar:   "*",
	MinLength:  12,
}

// MaskString 对字符串进行脱敏处理
// 示例：
//   - MaskString("password123", DefaultPasswordMask) => "pa*****23"
//   - MaskString("admin", DefaultUsernameMask) => "adm*n"
//   - MaskString("ssh-rsa AAAAB3...", DefaultKeyMask) => "ssh-rsa ********...后8位"
func MaskString(s string, config MaskConfig) string {
	if s == "" {
		return ""
	}

	length := len(s)

	// 如果字符串长度小于最小长度，全部脱敏
	if length < config.MinLength {
		return strings.Repeat(config.MaskChar, length)
	}

	// 如果保留的长度超过字符串长度，调整策略
	totalKeep := config.KeepPrefix + config.KeepSuffix
	if totalKeep >= length {
		// 保留前半部分，脱敏后半部分
		keep := length / 2
		if keep < 1 {
			keep = 1
		}
		return s[:keep] + strings.Repeat(config.MaskChar, length-keep)
	}

	// 正常脱敏：保留前后，中间用掩码
	maskLen := length - totalKeep
	return s[:config.KeepPrefix] + strings.Repeat(config.MaskChar, maskLen) + s[length-config.KeepSuffix:]
}

// MaskPassword 脱敏密码
func MaskPassword(password string) string {
	return MaskString(password, DefaultPasswordMask)
}

// MaskUsername 脱敏用户名
func MaskUsername(username string) string {
	return MaskString(username, DefaultUsernameMask)
}

// MaskPrivateKey 脱敏私钥
func MaskPrivateKey(key string) string {
	if key == "" {
		return ""
	}

	// 检测是否是完整的PEM格式私钥
	if strings.Contains(key, "-----BEGIN") {
		// 提取类型
		keyType := "PRIVATE KEY"
		if strings.Contains(key, "RSA") {
			keyType = "RSA PRIVATE KEY"
		} else if strings.Contains(key, "EC") {
			keyType = "EC PRIVATE KEY"
		} else if strings.Contains(key, "OPENSSH") {
			keyType = "OPENSSH PRIVATE KEY"
		}
		return "-----BEGIN " + keyType + "-----\n[MASKED]\n-----END " + keyType + "-----"
	}

	// 普通密钥字符串
	return MaskString(key, DefaultKeyMask)
}

// MaskToken 脱敏Token
func MaskToken(token string) string {
	return MaskString(token, DefaultTokenMask)
}

// MaskEmail 脱敏邮箱
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		// 不是有效邮箱格式
		return MaskString(email, DefaultUsernameMask)
	}

	// 脱敏用户名部分
	localPart := parts[0]
	domain := parts[1]

	if len(localPart) <= 2 {
		return localPart[:1] + "*@" + domain
	}

	// 保留首尾字符
	masked := localPart[:1] + strings.Repeat("*", len(localPart)-2) + localPart[len(localPart)-1:]
	return masked + "@" + domain
}

// MaskIP 脱敏IP地址（保留网段）
func MaskIP(ip string) string {
	if ip == "" {
		return ""
	}

	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		// IPv4: 保留前两段
		return parts[0] + "." + parts[1] + ".*.*"
	}

	// IPv6 或其他格式，保留前几位
	if len(ip) > 8 {
		return ip[:8] + "..."
	}
	return ip
}

// SensitiveFields 敏感字段列表
var SensitiveFields = []string{
	"password", "passwd", "pwd", "secret",
	"token", "auth", "key", "credential",
	"private_key", "privatekey", "ssh_key",
	"api_key", "apikey", "access_token",
	"refresh_token", "bearer", "authorization",
}

// MaskSensitiveFields 脱敏 map 中的敏感字段
func MaskSensitiveFields(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		lowerKey := strings.ToLower(key)
		isSensitive := false

		for _, sensitiveField := range SensitiveFields {
			if strings.Contains(lowerKey, sensitiveField) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			if strVal, ok := value.(string); ok {
				result[key] = MaskPassword(strVal)
			} else {
				result[key] = "[MASKED]"
			}
		} else {
			// 递归处理嵌套的 map
			if nestedMap, ok := value.(map[string]interface{}); ok {
				result[key] = MaskSensitiveFields(nestedMap)
			} else {
				result[key] = value
			}
		}
	}

	return result
}

// MaskLogMessage 脱敏日志消息中的敏感信息
// 使用正则表达式查找并替换常见的敏感信息模式
func MaskLogMessage(message string) string {
	if message == "" {
		return ""
	}

	result := message

	// 脱敏密码模式: password=xxx, passwd=xxx, pwd=xxx
	passwordPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*['"]?([^'"\s,}\]]+)['"]?`),
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*'([^']+)'`),
	}

	for _, pattern := range passwordPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// 提取密码值并脱敏
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				key := submatches[1]
				value := submatches[2]
				masked := MaskPassword(value)
				return key + "=" + masked
			}
			return match
		})
	}

	// 脱敏 token 模式
	tokenPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(token|auth|bearer|api_key|apikey)\s*[=:]\s*['"]?([^'"\s,}\]]+)['"]?`),
	}

	for _, pattern := range tokenPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				key := submatches[1]
				value := submatches[2]
				masked := MaskToken(value)
				return key + "=" + masked
			}
			return match
		})
	}

	// 脱敏 private key (多行)
	privateKeyPattern := regexp.MustCompile(`(?s)(-----BEGIN[^-]*PRIVATE KEY-----).*?(-----END[^-]*PRIVATE KEY-----)`)
	result = privateKeyPattern.ReplaceAllString(result, "$1\n[MASKED]\n$2")

	// 脱敏 JSON 中的敏感字段
	jsonPasswordPattern := regexp.MustCompile(`(?i)"(password|passwd|pwd|secret|token|key)":\s*"([^"]*)"`)
	result = jsonPasswordPattern.ReplaceAllStringFunc(result, func(match string) string {
		submatches := jsonPasswordPattern.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			key := submatches[1]
			value := submatches[2]
			masked := MaskPassword(value)
			return `"` + key + `":"` + masked + `"`
		}
		return match
	})

	return result
}

// MaskSSHCredentials 专门脱敏 SSH 凭据信息
type SSHCredentialsMasked struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

// MaskSSHCredentialsForLog 为日志脱敏 SSH 凭据
func MaskSSHCredentialsForLog(host string, port int, username, password, privateKey string) SSHCredentialsMasked {
	return SSHCredentialsMasked{
		Host:       host,
		Port:       port,
		Username:   MaskUsername(username),
		Password:   MaskPassword(password),
		PrivateKey: MaskPrivateKey(privateKey),
	}
}

// FormatMaskedCredentials 格式化脱敏后的凭据为日志字符串
func FormatMaskedCredentials(host string, port int, username, password string) string {
	maskedUser := MaskUsername(username)
	maskedPass := MaskPassword(password)
	return strings.Join([]string{
		"host=" + host,
		"port=" + string(rune(port)),
		"user=" + maskedUser,
		"pass=" + maskedPass,
	}, ", ")
}

// MaskHostConfig 脱敏主机配置结构
type MaskedHostConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	UseSudo    bool   `json:"use_sudo,omitempty"`
	SudoPass   string `json:"sudo_pass,omitempty"`
}

// MaskHostConfigForStorage 为数据库存储脱敏主机配置
func MaskHostConfigForStorage(host string, port int, username, password, privateKey string, useSudo bool, sudoPass string) MaskedHostConfig {
	return MaskedHostConfig{
		Host:       host,
		Port:       port,
		Username:   username, // 用户名可以保留用于识别
		Password:   MaskPassword(password),
		PrivateKey: MaskPrivateKey(privateKey),
		UseSudo:    useSudo,
		SudoPass:   MaskPassword(sudoPass),
	}
}
