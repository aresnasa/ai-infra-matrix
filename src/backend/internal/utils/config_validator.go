package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ConfigValidator 配置文件验证器
// 提供通用的安全验证功能，可被各种配置文件解析服务使用
type ConfigValidator struct {
	MaxFileSize    int64    // 最大文件大小（字节）
	MaxLineCount   int      // 最大行数
	MaxLineLength  int      // 最大单行长度
	MaxFieldLength int      // 最大字段长度
	MaxDepth       int      // 最大嵌套深度
	AllowedFormats []string // 允许的文件格式
}

// ValidationOptions 验证选项
type ValidationOptions struct {
	CheckEncoding     bool // 检查编码
	CheckSize         bool // 检查大小
	CheckLineCount    bool // 检查行数
	CheckLineLength   bool // 检查行长度
	CheckDangerous    bool // 检查危险内容
	CheckDepth        bool // 检查嵌套深度
	AllowLocalAddress bool // 允许本地地址
	AllowInternalIPs  bool // 允许内网 IP
}

// DefaultValidationOptions 默认验证选项（开启所有检查）
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		CheckEncoding:     true,
		CheckSize:         true,
		CheckLineCount:    true,
		CheckLineLength:   true,
		CheckDangerous:    true,
		CheckDepth:        true,
		AllowLocalAddress: false,
		AllowInternalIPs:  true,
	}
}

// NewConfigValidator 创建配置验证器（使用默认限制）
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		MaxFileSize:    1024 * 1024, // 1MB
		MaxLineCount:   10000,
		MaxLineLength:  4096,
		MaxFieldLength: 1024,
		MaxDepth:       10,
		AllowedFormats: []string{"csv", "json", "yaml", "yml", "ini", "toml", "conf"},
	}
}

// NewConfigValidatorWithLimits 创建自定义限制的配置验证器
func NewConfigValidatorWithLimits(maxFileSize int64, maxLineCount, maxLineLength, maxFieldLength, maxDepth int) *ConfigValidator {
	return &ConfigValidator{
		MaxFileSize:    maxFileSize,
		MaxLineCount:   maxLineCount,
		MaxLineLength:  maxLineLength,
		MaxFieldLength: maxFieldLength,
		MaxDepth:       maxDepth,
		AllowedFormats: []string{"csv", "json", "yaml", "yml", "ini", "toml", "conf"},
	}
}

// ================== 危险模式定义 ==================

// DangerousPatterns 通用危险模式
var DangerousPatterns = []*regexp.Regexp{
	// Shell 命令执行
	regexp.MustCompile(`(?i)\$\([^)]+\)`),                                    // $(command)
	regexp.MustCompile("(?i)`[^`]+`"),                                        // `command`
	regexp.MustCompile(`(?i)\|\s*(bash|sh|zsh|ksh|csh|tcsh|fish)`),           // | bash
	regexp.MustCompile(`(?i)(bash|sh|zsh)\s+-c\s+`),                          // bash -c
	regexp.MustCompile(`(?i)(curl|wget)\s+.*(http|ftp).*\|\s*(bash|sh|zsh)`), // curl | bash

	// 危险命令
	regexp.MustCompile(`(?i)\b(rm\s+-rf|dd\s+if=|mkfs|fdisk|parted)\b`),
	regexp.MustCompile(`(?i)\b(chmod\s+777|chown\s+-R|sudo\s+su)\b`),

	// 脚本注入
	regexp.MustCompile(`(?i)<script[^>]*>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']+`),

	// 代码执行
	regexp.MustCompile(`(?i)(python|python3|ruby|perl|node)\s+-e\s+`),
	regexp.MustCompile(`(?i)__import__\s*\(`),
	regexp.MustCompile(`(?i)\beval\s*\(`),
	regexp.MustCompile(`(?i)\bexec\s*\(`),

	// 网络攻击
	regexp.MustCompile(`(?i)nc\s+-[elp]`),
	regexp.MustCompile(`(?i)/dev/tcp/`),

	// 危险文件操作
	regexp.MustCompile(`(?i)>\s*/etc/`),
	regexp.MustCompile(`(?i)>\s*/root/`),
	regexp.MustCompile(`(?i)>\s*/bin/`),
	regexp.MustCompile(`(?i)>\s*/sbin/`),
	regexp.MustCompile(`(?i)>\s*/usr/`),

	// YAML/JSON 特殊攻击
	regexp.MustCompile(`(?i)!!python/`),
	regexp.MustCompile(`(?i)!!ruby/`),
	regexp.MustCompile(`(?i)tag:yaml\.org,\d+:python/`),
	regexp.MustCompile(`(?i)!<tag:yaml\.org`),

	// SQL 注入
	regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|insert\s+into|update\s+.+\s+set)`),
	regexp.MustCompile(`(?i)(;\s*--)|(--\s*$)|(/\*.*\*/)`),

	// 路径遍历
	regexp.MustCompile(`(?i)\.\.\/`),
	regexp.MustCompile(`(?i)\.\.\\`),
	regexp.MustCompile(`(?i)%2e%2e[/\\]`),

	// 环境变量注入
	regexp.MustCompile(`(?i)\$\{[^}]+\}`),
	regexp.MustCompile(`(?i)%[A-Z_]+%`),
}

// ================== 验证方法 ==================

// Validate 执行完整验证
func (v *ConfigValidator) Validate(data []byte, format string, opts ValidationOptions) error {
	// 检查是否为空
	if len(data) == 0 {
		return fmt.Errorf("文件内容为空")
	}

	// 检查编码
	if opts.CheckEncoding {
		if err := v.ValidateEncoding(data); err != nil {
			return err
		}
	}

	// 检查大小
	if opts.CheckSize {
		if err := v.ValidateSize(data); err != nil {
			return err
		}
	}

	// 检查行数
	if opts.CheckLineCount {
		if err := v.ValidateLineCount(data); err != nil {
			return err
		}
	}

	// 检查行长度
	if opts.CheckLineLength {
		if err := v.ValidateLineLength(data); err != nil {
			return err
		}
	}

	// 检查格式
	if format != "" {
		if err := v.ValidateFormat(format); err != nil {
			return err
		}
	}

	// 检查危险内容
	if opts.CheckDangerous {
		if err := v.ValidateDangerousContent(data); err != nil {
			return err
		}
	}

	return nil
}

// ValidateEncoding 验证文件编码
func (v *ConfigValidator) ValidateEncoding(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("文件编码无效，请使用 UTF-8 编码")
	}
	return nil
}

// ValidateSize 验证文件大小
func (v *ConfigValidator) ValidateSize(data []byte) error {
	if int64(len(data)) > v.MaxFileSize {
		return fmt.Errorf("文件大小超过限制: %d bytes (最大 %d bytes)", len(data), v.MaxFileSize)
	}
	return nil
}

// ValidateLineCount 验证行数
func (v *ConfigValidator) ValidateLineCount(data []byte) error {
	lineCount := strings.Count(string(data), "\n") + 1
	if lineCount > v.MaxLineCount {
		return fmt.Errorf("文件行数超过限制: %d 行 (最大 %d 行)", lineCount, v.MaxLineCount)
	}
	return nil
}

// ValidateLineLength 验证单行长度
func (v *ConfigValidator) ValidateLineLength(data []byte) error {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if len(line) > v.MaxLineLength {
			return fmt.Errorf("第 %d 行超过最大长度限制 (%d 字符，最大 %d)", i+1, len(line), v.MaxLineLength)
		}
	}
	return nil
}

// ValidateFormat 验证文件格式
func (v *ConfigValidator) ValidateFormat(format string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	for _, allowed := range v.AllowedFormats {
		if format == allowed {
			return nil
		}
	}
	return fmt.Errorf("不支持的文件格式: %s，仅支持 %s", format, strings.Join(v.AllowedFormats, ", "))
}

// ValidateDangerousContent 检测危险内容
func (v *ConfigValidator) ValidateDangerousContent(data []byte) error {
	content := string(data)

	for _, pattern := range DangerousPatterns {
		if pattern.MatchString(content) {
			match := pattern.FindString(content)
			// 截断匹配内容，避免泄露太多信息
			if len(match) > 50 {
				match = match[:50] + "..."
			}
			return fmt.Errorf("检测到危险内容，请检查文件是否包含恶意代码: %s", match)
		}
	}

	return nil
}

// ValidateFieldValue 验证字段值
func (v *ConfigValidator) ValidateFieldValue(fieldName, value string, isSensitive bool) error {
	// 检查字段长度
	if len(value) > v.MaxFieldLength {
		return fmt.Errorf("字段 %s 值过长 (%d 字符，最大 %d)", fieldName, len(value), v.MaxFieldLength)
	}

	// 非敏感字段检查危险内容
	if !isSensitive {
		for _, pattern := range DangerousPatterns {
			if pattern.MatchString(value) {
				return fmt.Errorf("字段 %s 包含危险内容", fieldName)
			}
		}
	}

	return nil
}

// ValidateYAMLDepth 验证 YAML 嵌套深度
func (v *ConfigValidator) ValidateYAMLDepth(data []byte) error {
	lines := strings.Split(string(data), "\n")
	maxIndent := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		depth := indent / 2
		if depth > maxIndent {
			maxIndent = depth
		}
	}
	if maxIndent > v.MaxDepth {
		return fmt.Errorf("YAML 嵌套深度超过限制: %d (最大 %d)", maxIndent, v.MaxDepth)
	}
	return nil
}

// ValidateIPAddress 验证 IP 地址安全性
func (v *ConfigValidator) ValidateIPAddress(ip string, opts ValidationOptions) error {
	ipLower := strings.ToLower(ip)

	// 检查本地地址
	if !opts.AllowLocalAddress {
		if ipLower == "localhost" || ipLower == "127.0.0.1" || ipLower == "::1" {
			return fmt.Errorf("不允许使用本地地址: %s", ip)
		}
	}

	// 检查内网地址
	if !opts.AllowInternalIPs {
		if strings.HasPrefix(ip, "10.") ||
			strings.HasPrefix(ip, "172.16.") || strings.HasPrefix(ip, "172.17.") ||
			strings.HasPrefix(ip, "172.18.") || strings.HasPrefix(ip, "172.19.") ||
			strings.HasPrefix(ip, "172.20.") || strings.HasPrefix(ip, "172.21.") ||
			strings.HasPrefix(ip, "172.22.") || strings.HasPrefix(ip, "172.23.") ||
			strings.HasPrefix(ip, "172.24.") || strings.HasPrefix(ip, "172.25.") ||
			strings.HasPrefix(ip, "172.26.") || strings.HasPrefix(ip, "172.27.") ||
			strings.HasPrefix(ip, "172.28.") || strings.HasPrefix(ip, "172.29.") ||
			strings.HasPrefix(ip, "172.30.") || strings.HasPrefix(ip, "172.31.") ||
			strings.HasPrefix(ip, "192.168.") {
			return fmt.Errorf("不允许使用内网地址: %s", ip)
		}
	}

	return nil
}

// ValidateUsername 验证用户名安全性
func (v *ConfigValidator) ValidateUsername(username string) error {
	// 用户名不能为空
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("用户名不能为空")
	}

	// 用户名长度限制
	if len(username) > 64 {
		return fmt.Errorf("用户名过长 (%d 字符，最大 64)", len(username))
	}

	// 用户名不能包含危险字符
	if strings.ContainsAny(username, ";|&$`\"'<>(){}[]\\") {
		return fmt.Errorf("用户名包含非法字符")
	}

	return nil
}

// ValidateIdentifier 验证标识符（如 minion_id, group 等）
func (v *ConfigValidator) ValidateIdentifier(name, value string) error {
	if value == "" {
		return nil // 空值是允许的
	}

	// 长度限制
	if len(value) > 128 {
		return fmt.Errorf("%s 过长 (%d 字符，最大 128)", name, len(value))
	}

	// 只能包含字母、数字、连字符、下划线和点
	identifierRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !identifierRegex.MatchString(value) {
		return fmt.Errorf("%s 格式无效: %s (只能包含字母、数字、连字符、下划线和点)", name, value)
	}

	return nil
}

// SanitizeForLog 对敏感信息进行脱敏（用于日志记录）
func SanitizeForLog(content string) string {
	// 脱敏密码字段
	passwordPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|passwd|pass|secret|token|key|auth)\s*[:=]\s*["']?([^"'\s]+)["']?`),
		regexp.MustCompile(`(?i)(password|passwd|pass|secret|token|key|auth)\s*[:=]\s*(.+?)[\s,}\]"]`),
	}

	sanitized := content
	for _, pattern := range passwordPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "$1=***REDACTED***")
	}

	return sanitized
}
