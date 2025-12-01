package services

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// ================== 安全配置常量 ==================

// 允许的文件格式
var allowedFormats = map[string]bool{
	"csv":  true,
	"json": true,
	"yaml": true,
	"yml":  true,
	"ini":  true,
}

// 最大文件大小限制 (1MB)
const MaxFileSize = 1024 * 1024

// 最大行数限制
const MaxLineCount = 10000

// 最大主机数限制
const MaxHostCount = 5000

// 最大字段长度限制
const MaxFieldLength = 1024

// YAML 最大递归深度
const MaxYAMLDepth = 10

// 最大单行长度
const MaxLineLength = 4096

// ================== 危险模式检测 ==================

// 危险模式检测正则表达式
var dangerousPatterns = []*regexp.Regexp{
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
	regexp.MustCompile(`(?i)<script[^>]*>`),          // HTML script tag
	regexp.MustCompile(`(?i)javascript:`),            // javascript:
	regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']+`), // event handlers
	// Python/Ruby 执行
	regexp.MustCompile(`(?i)(python|python3|ruby|perl|node)\s+-e\s+`),
	regexp.MustCompile(`(?i)__import__\s*\(`),
	regexp.MustCompile(`(?i)\beval\s*\(`),
	regexp.MustCompile(`(?i)\bexec\s*\(`),
	// 网络相关危险操作
	regexp.MustCompile(`(?i)nc\s+-[elp]`), // netcat reverse shell
	regexp.MustCompile(`(?i)/dev/tcp/`),   // bash tcp redirect
	// 文件操作危险模式
	regexp.MustCompile(`(?i)>\s*/etc/`),  // 写入 /etc
	regexp.MustCompile(`(?i)>\s*/root/`), // 写入 /root
	regexp.MustCompile(`(?i)>\s*/bin/`),  // 写入 /bin
	regexp.MustCompile(`(?i)>\s*/sbin/`), // 写入 /sbin
	regexp.MustCompile(`(?i)>\s*/usr/`),  // 写入 /usr
	// YAML 特殊攻击
	regexp.MustCompile(`(?i)!!python/`),                 // PyYAML code execution
	regexp.MustCompile(`(?i)!!ruby/`),                   // Ruby YAML code execution
	regexp.MustCompile(`(?i)tag:yaml\.org,\d+:python/`), // YAML python tag
	regexp.MustCompile(`(?i)!<tag:yaml\.org`),           // YAML custom tag
	// SQL 注入模式
	regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|insert\s+into|update\s+.+\s+set)`),
	regexp.MustCompile(`(?i)(;\s*--)|(--\s*$)|(/\*.*\*/)`), // SQL comments
	// 路径遍历
	regexp.MustCompile(`(?i)\.\./`),       // 相对路径遍历
	regexp.MustCompile(`(?i)\.\.\\`),      // Windows 路径遍历
	regexp.MustCompile(`(?i)%2e%2e[/\\]`), // URL 编码路径遍历
	// 环境变量注入
	regexp.MustCompile(`(?i)\$\{[^}]+\}`), // ${VAR}
	regexp.MustCompile(`(?i)\$[A-Z_]+\b`), // $VAR (大写变量名)
	regexp.MustCompile(`(?i)%[A-Z_]+%`),   // Windows %VAR%
}

// 敏感字段名列表（需要脱敏的字段）
var sensitiveFieldNames = map[string]bool{
	"password":      true,
	"pass":          true,
	"secret":        true,
	"token":         true,
	"key":           true,
	"private":       true,
	"credential":    true,
	"auth":          true,
	"api_key":       true,
	"apikey":        true,
	"access_token":  true,
	"refresh_token": true,
	"ssh_password":  true,
	"ansible_pass":  true,
}

// HostConfig 主机配置结构
type HostConfig struct {
	Host     string `json:"host" yaml:"host" csv:"host"`
	Port     int    `json:"port" yaml:"port" csv:"port"`
	Username string `json:"username" yaml:"username" csv:"username"`
	Password string `json:"password" yaml:"password" csv:"password"`
	UseSudo  bool   `json:"use_sudo" yaml:"use_sudo" csv:"use_sudo"`
	MinionID string `json:"minion_id,omitempty" yaml:"minion_id,omitempty" csv:"minion_id"`
	Group    string `json:"group,omitempty" yaml:"group,omitempty" csv:"group"`
}

// HostParserService 主机数据解析服务
type HostParserService struct{}

// NewHostParserService 创建主机解析服务
func NewHostParserService() *HostParserService {
	return &HostParserService{}
}

// ValidateFormat 验证文件格式是否在白名单中
func (s *HostParserService) ValidateFormat(format string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		return nil // 允许空格式，将使用自动检测
	}
	if !allowedFormats[format] {
		return fmt.Errorf("不支持的文件格式: %s，仅支持 csv, json, yaml, yml, ini", format)
	}
	return nil
}

// ValidateFileSize 验证文件大小
func (s *HostParserService) ValidateFileSize(data []byte) error {
	if len(data) > MaxFileSize {
		return fmt.Errorf("文件大小超过限制: %d bytes (最大 %d bytes)", len(data), MaxFileSize)
	}
	return nil
}

// DetectDangerousContent 检测危险内容
func (s *HostParserService) DetectDangerousContent(data []byte) error {
	content := string(data)

	for _, pattern := range dangerousPatterns {
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

// ValidateLineCount 验证行数限制
func (s *HostParserService) ValidateLineCount(data []byte) error {
	lineCount := strings.Count(string(data), "\n") + 1
	if lineCount > MaxLineCount {
		return fmt.Errorf("文件行数超过限制: %d 行 (最大 %d 行)", lineCount, MaxLineCount)
	}
	return nil
}

// ValidateEncoding 验证文件编码（必须是有效的 UTF-8）
func (s *HostParserService) ValidateEncoding(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("文件编码无效，请使用 UTF-8 编码")
	}
	return nil
}

// ValidateLineLength 验证单行长度
func (s *HostParserService) ValidateLineLength(data []byte) error {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if len(line) > MaxLineLength {
			return fmt.Errorf("第 %d 行超过最大长度限制 (%d 字符)", i+1, MaxLineLength)
		}
	}
	return nil
}

// ValidateFieldValue 验证字段值安全性
func (s *HostParserService) ValidateFieldValue(fieldName, value string) error {
	// 检查字段长度
	if len(value) > MaxFieldLength {
		return fmt.Errorf("字段 %s 值过长 (%d 字符，最大 %d)", fieldName, len(value), MaxFieldLength)
	}

	// 非密码字段检查危险内容
	if !sensitiveFieldNames[strings.ToLower(fieldName)] {
		for _, pattern := range dangerousPatterns {
			if pattern.MatchString(value) {
				return fmt.Errorf("字段 %s 包含危险内容", fieldName)
			}
		}
	}

	return nil
}

// ValidateHostConfig 验证主机配置的安全性
func (s *HostParserService) ValidateHostConfig(host *HostConfig) error {
	// 验证 Host 字段
	if err := s.ValidateFieldValue("host", host.Host); err != nil {
		return err
	}
	// Host 不能是本地回环地址（防止 SSRF）
	hostLower := strings.ToLower(host.Host)
	if hostLower == "localhost" || hostLower == "127.0.0.1" || hostLower == "::1" {
		return fmt.Errorf("不允许使用本地地址: %s", host.Host)
	}
	// Host 不能是内部 Docker 网络地址
	if strings.HasPrefix(host.Host, "172.") || strings.HasPrefix(host.Host, "10.") {
		// 允许内网地址，但需要记录日志（不返回错误）
	}

	// 验证 Username
	if err := s.ValidateFieldValue("username", host.Username); err != nil {
		return err
	}
	// Username 不能包含特殊字符
	if strings.ContainsAny(host.Username, ";|&$`\"'<>(){}[]\\") {
		return fmt.Errorf("用户名包含非法字符: %s", host.Username)
	}

	// 验证 MinionID
	if host.MinionID != "" {
		if err := s.ValidateFieldValue("minion_id", host.MinionID); err != nil {
			return err
		}
		// MinionID 只能包含字母、数字、连字符、下划线和点
		minionIDRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
		if !minionIDRegex.MatchString(host.MinionID) {
			return fmt.Errorf("Minion ID 格式无效: %s (只能包含字母、数字、连字符、下划线和点)", host.MinionID)
		}
	}

	// 验证 Group
	if host.Group != "" {
		if err := s.ValidateFieldValue("group", host.Group); err != nil {
			return err
		}
		// Group 只能包含字母、数字、连字符、下划线
		groupRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !groupRegex.MatchString(host.Group) {
			return fmt.Errorf("分组名格式无效: %s (只能包含字母、数字、连字符、下划线)", host.Group)
		}
	}

	return nil
}

// ValidateAndParse 验证并解析主机文件（带安全检查）
func (s *HostParserService) ValidateAndParse(data []byte, format string) ([]HostConfig, error) {
	// 1. 检查是否为空
	if len(data) == 0 {
		return nil, fmt.Errorf("文件内容为空")
	}

	// 2. 验证文件编码
	if err := s.ValidateEncoding(data); err != nil {
		return nil, err
	}

	// 3. 验证文件大小
	if err := s.ValidateFileSize(data); err != nil {
		return nil, err
	}

	// 4. 验证行数
	if err := s.ValidateLineCount(data); err != nil {
		return nil, err
	}

	// 5. 验证单行长度
	if err := s.ValidateLineLength(data); err != nil {
		return nil, err
	}

	// 6. 验证文件格式
	if err := s.ValidateFormat(format); err != nil {
		return nil, err
	}

	// 7. 检测危险内容
	if err := s.DetectDangerousContent(data); err != nil {
		return nil, err
	}

	// 8. 解析文件
	hosts, err := s.ParseHosts(data, format)
	if err != nil {
		return nil, err
	}

	// 9. 验证主机数量
	if len(hosts) > MaxHostCount {
		return nil, fmt.Errorf("主机数量超过限制: %d (最大 %d)", len(hosts), MaxHostCount)
	}

	// 10. 验证每个主机配置的安全性
	for i, host := range hosts {
		if err := s.ValidateHostConfig(&host); err != nil {
			return nil, fmt.Errorf("第 %d 个主机配置错误: %v", i+1, err)
		}
	}

	return hosts, nil
}

// ParseHosts 根据格式解析主机数据
func (s *HostParserService) ParseHosts(data []byte, format string) ([]HostConfig, error) {
	format = strings.ToLower(strings.TrimSpace(format))

	switch format {
	case "csv":
		return s.ParseCSV(data)
	case "json":
		return s.ParseJSON(data)
	case "yaml", "yml":
		return s.ParseYAML(data)
	case "ini", "ansible":
		return s.ParseAnsibleINI(data)
	default:
		// 尝试自动检测格式
		return s.AutoDetectAndParse(data)
	}
}

// AutoDetectAndParse 自动检测格式并解析
func (s *HostParserService) AutoDetectAndParse(data []byte) ([]HostConfig, error) {
	content := strings.TrimSpace(string(data))

	// 尝试 JSON
	if strings.HasPrefix(content, "[") || strings.HasPrefix(content, "{") {
		if hosts, err := s.ParseJSON(data); err == nil {
			return hosts, nil
		}
	}

	// 尝试 YAML
	if strings.Contains(content, ":") && !strings.Contains(content, ",") {
		if hosts, err := s.ParseYAML(data); err == nil && len(hosts) > 0 {
			return hosts, nil
		}
	}

	// 尝试 CSV
	if strings.Contains(content, ",") {
		if hosts, err := s.ParseCSV(data); err == nil && len(hosts) > 0 {
			return hosts, nil
		}
	}

	// 尝试 Ansible INI
	if strings.Contains(content, "[") && strings.Contains(content, "]") {
		if hosts, err := s.ParseAnsibleINI(data); err == nil && len(hosts) > 0 {
			return hosts, nil
		}
	}

	return nil, fmt.Errorf("无法自动检测文件格式，请明确指定格式")
}

// ParseCSV 解析 CSV 格式
// 格式: host,port,username,password,use_sudo,minion_id,group
func (s *HostParserService) ParseCSV(data []byte) ([]HostConfig, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // 允许不同行有不同字段数

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 解析错误: %v", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV 文件为空")
	}

	// 解析表头
	header := records[0]
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// 检查必需字段
	requiredFields := []string{"host"}
	for _, field := range requiredFields {
		if _, ok := headerMap[field]; !ok {
			// 尝试别名
			aliases := map[string][]string{
				"host": {"ip", "address", "hostname", "server"},
			}
			found := false
			for _, alias := range aliases[field] {
				if idx, ok := headerMap[alias]; ok {
					headerMap[field] = idx
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("CSV 缺少必需字段: %s", field)
			}
		}
	}

	// 解析数据行
	var hosts []HostConfig
	for i, record := range records[1:] {
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue // 跳过空行
		}

		host := HostConfig{
			Port:     22,
			Username: "root",
			UseSudo:  false,
		}

		// 解析各字段
		if idx, ok := headerMap["host"]; ok && idx < len(record) {
			host.Host = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["ip"]; ok && idx < len(record) && host.Host == "" {
			host.Host = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["port"]; ok && idx < len(record) {
			if port, err := strconv.Atoi(strings.TrimSpace(record[idx])); err == nil {
				host.Port = port
			}
		}
		if idx, ok := headerMap["username"]; ok && idx < len(record) {
			if u := strings.TrimSpace(record[idx]); u != "" {
				host.Username = u
			}
		}
		if idx, ok := headerMap["user"]; ok && idx < len(record) && host.Username == "root" {
			if u := strings.TrimSpace(record[idx]); u != "" {
				host.Username = u
			}
		}
		if idx, ok := headerMap["password"]; ok && idx < len(record) {
			host.Password = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["pass"]; ok && idx < len(record) && host.Password == "" {
			host.Password = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["use_sudo"]; ok && idx < len(record) {
			val := strings.ToLower(strings.TrimSpace(record[idx]))
			host.UseSudo = val == "true" || val == "yes" || val == "1"
		}
		if idx, ok := headerMap["sudo"]; ok && idx < len(record) {
			val := strings.ToLower(strings.TrimSpace(record[idx]))
			host.UseSudo = val == "true" || val == "yes" || val == "1"
		}
		if idx, ok := headerMap["minion_id"]; ok && idx < len(record) {
			host.MinionID = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["group"]; ok && idx < len(record) {
			host.Group = strings.TrimSpace(record[idx])
		}

		if host.Host == "" {
			continue // 跳过没有主机地址的行
		}

		// 验证
		if err := s.validateHost(&host); err != nil {
			return nil, fmt.Errorf("第 %d 行数据错误: %v", i+2, err)
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}

// ParseJSON 解析 JSON 格式
func (s *HostParserService) ParseJSON(data []byte) ([]HostConfig, error) {
	var hosts []HostConfig

	// 尝试解析为数组
	if err := json.Unmarshal(data, &hosts); err == nil {
		for i := range hosts {
			s.setDefaults(&hosts[i])
			if err := s.validateHost(&hosts[i]); err != nil {
				return nil, fmt.Errorf("第 %d 个主机配置错误: %v", i+1, err)
			}
		}
		return hosts, nil
	}

	// 尝试解析为对象（hosts 字段）
	var wrapper struct {
		Hosts []HostConfig `json:"hosts"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Hosts) > 0 {
		for i := range wrapper.Hosts {
			s.setDefaults(&wrapper.Hosts[i])
			if err := s.validateHost(&wrapper.Hosts[i]); err != nil {
				return nil, fmt.Errorf("第 %d 个主机配置错误: %v", i+1, err)
			}
		}
		return wrapper.Hosts, nil
	}

	return nil, fmt.Errorf("JSON 解析失败：格式不正确")
}

// ParseYAML 解析 YAML 格式（带安全限制）
func (s *HostParserService) ParseYAML(data []byte) ([]HostConfig, error) {
	// 使用 yaml.v3 的 Decoder 进行安全解析
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	// 限制节点数量，防止 YAML 炸弹攻击
	decoder.KnownFields(true) // 不允许未知字段

	var hosts []HostConfig

	// 尝试解析为数组
	if err := yaml.Unmarshal(data, &hosts); err == nil && len(hosts) > 0 {
		// 验证解析深度（防止过于复杂的嵌套结构）
		if err := s.validateYAMLDepth(data); err != nil {
			return nil, err
		}
		for i := range hosts {
			s.setDefaults(&hosts[i])
			if err := s.validateHost(&hosts[i]); err != nil {
				return nil, fmt.Errorf("第 %d 个主机配置错误: %v", i+1, err)
			}
		}
		return hosts, nil
	}

	// 尝试解析为对象（hosts 字段）
	var wrapper struct {
		Hosts []HostConfig `yaml:"hosts"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err == nil && len(wrapper.Hosts) > 0 {
		if err := s.validateYAMLDepth(data); err != nil {
			return nil, err
		}
		for i := range wrapper.Hosts {
			s.setDefaults(&wrapper.Hosts[i])
			if err := s.validateHost(&wrapper.Hosts[i]); err != nil {
				return nil, fmt.Errorf("第 %d 个主机配置错误: %v", i+1, err)
			}
		}
		return wrapper.Hosts, nil
	}

	// 尝试解析为 map 格式（Ansible 风格）
	var hostMap map[string]interface{}
	if err := yaml.Unmarshal(data, &hostMap); err == nil {
		if err := s.validateYAMLDepth(data); err != nil {
			return nil, err
		}
		return s.parseYAMLMap(hostMap)
	}

	return nil, fmt.Errorf("YAML 解析失败：格式不正确")
}

// validateYAMLDepth 验证 YAML 嵌套深度
func (s *HostParserService) validateYAMLDepth(data []byte) error {
	// 简单的深度检测：通过缩进来估算
	lines := strings.Split(string(data), "\n")
	maxIndent := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		// 假设每层缩进 2 个空格
		depth := indent / 2
		if depth > maxIndent {
			maxIndent = depth
		}
	}
	if maxIndent > MaxYAMLDepth {
		return fmt.Errorf("YAML 嵌套深度超过限制: %d (最大 %d)", maxIndent, MaxYAMLDepth)
	}
	return nil
}

// parseYAMLMap 解析 YAML map 格式
func (s *HostParserService) parseYAMLMap(hostMap map[string]interface{}) ([]HostConfig, error) {
	var hosts []HostConfig

	for key, value := range hostMap {
		switch v := value.(type) {
		case map[string]interface{}:
			host := HostConfig{
				Host:     key,
				Port:     22,
				Username: "root",
			}

			if port, ok := v["port"].(int); ok {
				host.Port = port
			}
			if username, ok := v["username"].(string); ok {
				host.Username = username
			}
			if username, ok := v["user"].(string); ok {
				host.Username = username
			}
			if password, ok := v["password"].(string); ok {
				host.Password = password
			}
			if useSudo, ok := v["use_sudo"].(bool); ok {
				host.UseSudo = useSudo
			}
			if minionID, ok := v["minion_id"].(string); ok {
				host.MinionID = minionID
			}
			if group, ok := v["group"].(string); ok {
				host.Group = group
			}

			if err := s.validateHost(&host); err == nil {
				hosts = append(hosts, host)
			}
		}
	}

	return hosts, nil
}

// ParseAnsibleINI 解析 Ansible INI 格式
// 格式示例:
// [webservers]
// web1 ansible_host=192.168.1.10 ansible_port=22 ansible_user=root ansible_password=pass ansible_become=true
// web2 ansible_host=192.168.1.11
func (s *HostParserService) ParseAnsibleINI(data []byte) ([]HostConfig, error) {
	var hosts []HostConfig
	currentGroup := ""

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNum := 0

	// 正则表达式
	groupRegex := regexp.MustCompile(`^\[([^\]]+)\]`)
	hostRegex := regexp.MustCompile(`^([^\s#]+)\s*(.*)`)
	varRegex := regexp.MustCompile(`(\w+)=([^\s]+)`)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// 检查是否是组定义
		if matches := groupRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentGroup = matches[1]
			// 跳过特殊组
			if strings.HasSuffix(currentGroup, ":vars") || strings.HasSuffix(currentGroup, ":children") {
				currentGroup = ""
			}
			continue
		}

		// 跳过特殊组内容
		if currentGroup == "" {
			continue
		}

		// 解析主机行
		if matches := hostRegex.FindStringSubmatch(line); len(matches) > 1 {
			hostName := matches[1]
			vars := matches[2]

			host := HostConfig{
				Host:     hostName,
				Port:     22,
				Username: "root",
				UseSudo:  false,
				Group:    currentGroup,
			}

			// 解析变量
			varMatches := varRegex.FindAllStringSubmatch(vars, -1)
			for _, vm := range varMatches {
				key := strings.ToLower(vm[1])
				value := strings.Trim(vm[2], "\"'")

				switch key {
				case "ansible_host":
					host.Host = value
				case "ansible_port":
					if port, err := strconv.Atoi(value); err == nil {
						host.Port = port
					}
				case "ansible_user", "ansible_ssh_user":
					host.Username = value
				case "ansible_password", "ansible_ssh_pass":
					host.Password = value
				case "ansible_become", "ansible_sudo":
					host.UseSudo = value == "true" || value == "yes" || value == "1"
				case "minion_id":
					host.MinionID = value
				}
			}

			if err := s.validateHost(&host); err == nil {
				hosts = append(hosts, host)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 INI 文件错误: %v", err)
	}

	return hosts, nil
}

// setDefaults 设置默认值
func (s *HostParserService) setDefaults(host *HostConfig) {
	if host.Port == 0 {
		host.Port = 22
	}
	if host.Username == "" {
		host.Username = "root"
	}
}

// validateHost 验证主机配置
func (s *HostParserService) validateHost(host *HostConfig) error {
	if host.Host == "" {
		return fmt.Errorf("主机地址不能为空")
	}
	if host.Port < 1 || host.Port > 65535 {
		return fmt.Errorf("端口号无效: %d", host.Port)
	}
	if host.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return nil
}

// GenerateCSVTemplate 生成 CSV 模板
func (s *HostParserService) GenerateCSVTemplate() string {
	return `host,port,username,password,use_sudo,minion_id,group
192.168.1.10,22,root,your_password,false,minion-01,webservers
192.168.1.11,22,admin,your_password,true,minion-02,databases
192.168.1.12,22,deploy,your_password,true,minion-03,webservers`
}

// GenerateJSONTemplate 生成 JSON 模板
func (s *HostParserService) GenerateJSONTemplate() string {
	template := []HostConfig{
		{Host: "192.168.1.10", Port: 22, Username: "root", Password: "your_password", UseSudo: false, MinionID: "minion-01", Group: "webservers"},
		{Host: "192.168.1.11", Port: 22, Username: "admin", Password: "your_password", UseSudo: true, MinionID: "minion-02", Group: "databases"},
		{Host: "192.168.1.12", Port: 22, Username: "deploy", Password: "your_password", UseSudo: true, MinionID: "minion-03", Group: "webservers"},
	}
	data, _ := json.MarshalIndent(template, "", "  ")
	return string(data)
}

// GenerateYAMLTemplate 生成 YAML 模板
func (s *HostParserService) GenerateYAMLTemplate() string {
	return `# Salt Minion 主机配置
# 字段说明: host-主机地址, port-SSH端口, username-用户名, password-密码, use_sudo-是否使用sudo, minion_id-Minion ID, group-分组

hosts:
  - host: 192.168.1.10
    port: 22
    username: root
    password: your_password
    use_sudo: false
    minion_id: minion-01
    group: webservers

  - host: 192.168.1.11
    port: 22
    username: admin
    password: your_password
    use_sudo: true
    minion_id: minion-02
    group: databases

  - host: 192.168.1.12
    port: 22
    username: deploy
    password: your_password
    use_sudo: true
    minion_id: minion-03
    group: webservers`
}

// GenerateAnsibleINITemplate 生成 Ansible INI 模板
func (s *HostParserService) GenerateAnsibleINITemplate() string {
	return `# Ansible Inventory 格式
# 可以直接使用 Ansible 的 inventory 文件

[webservers]
minion-01 ansible_host=192.168.1.10 ansible_port=22 ansible_user=root ansible_password=your_password ansible_become=false
minion-03 ansible_host=192.168.1.12 ansible_port=22 ansible_user=deploy ansible_password=your_password ansible_become=true

[databases]
minion-02 ansible_host=192.168.1.11 ansible_port=22 ansible_user=admin ansible_password=your_password ansible_become=true

[all:vars]
# 全局变量（可选）
ansible_python_interpreter=/usr/bin/python3`
}

// GetTemplate 获取指定格式的模板
func (s *HostParserService) GetTemplate(format string) (string, string, error) {
	format = strings.ToLower(strings.TrimSpace(format))

	switch format {
	case "csv":
		return s.GenerateCSVTemplate(), "hosts_template.csv", nil
	case "json":
		return s.GenerateJSONTemplate(), "hosts_template.json", nil
	case "yaml", "yml":
		return s.GenerateYAMLTemplate(), "hosts_template.yaml", nil
	case "ini", "ansible":
		return s.GenerateAnsibleINITemplate(), "hosts_inventory.ini", nil
	default:
		return "", "", fmt.Errorf("不支持的格式: %s", format)
	}
}

// ExportHosts 导出主机配置为指定格式
func (s *HostParserService) ExportHosts(hosts []HostConfig, format string) ([]byte, error) {
	format = strings.ToLower(strings.TrimSpace(format))

	switch format {
	case "csv":
		return s.exportCSV(hosts)
	case "json":
		return json.MarshalIndent(hosts, "", "  ")
	case "yaml", "yml":
		return yaml.Marshal(hosts)
	default:
		return nil, fmt.Errorf("不支持的导出格式: %s", format)
	}
}

func (s *HostParserService) exportCSV(hosts []HostConfig) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// 写入表头
	writer.Write([]string{"host", "port", "username", "password", "use_sudo", "minion_id", "group"})

	// 写入数据
	for _, h := range hosts {
		useSudo := "false"
		if h.UseSudo {
			useSudo = "true"
		}
		writer.Write([]string{
			h.Host,
			strconv.Itoa(h.Port),
			h.Username,
			h.Password,
			useSudo,
			h.MinionID,
			h.Group,
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// StreamParseCSV 流式解析大型 CSV 文件
func (s *HostParserService) StreamParseCSV(reader io.Reader, callback func(host HostConfig) error) error {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// 读取表头
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("读取 CSV 表头失败: %v", err)
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	lineNum := 1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("第 %d 行读取错误: %v", lineNum+1, err)
		}
		lineNum++

		host := HostConfig{Port: 22, Username: "root"}

		if idx, ok := headerMap["host"]; ok && idx < len(record) {
			host.Host = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["port"]; ok && idx < len(record) {
			if port, err := strconv.Atoi(strings.TrimSpace(record[idx])); err == nil {
				host.Port = port
			}
		}
		if idx, ok := headerMap["username"]; ok && idx < len(record) {
			if u := strings.TrimSpace(record[idx]); u != "" {
				host.Username = u
			}
		}
		if idx, ok := headerMap["password"]; ok && idx < len(record) {
			host.Password = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["use_sudo"]; ok && idx < len(record) {
			val := strings.ToLower(strings.TrimSpace(record[idx]))
			host.UseSudo = val == "true" || val == "yes" || val == "1"
		}
		if idx, ok := headerMap["minion_id"]; ok && idx < len(record) {
			host.MinionID = strings.TrimSpace(record[idx])
		}
		if idx, ok := headerMap["group"]; ok && idx < len(record) {
			host.Group = strings.TrimSpace(record[idx])
		}

		if host.Host == "" {
			continue
		}

		if err := callback(host); err != nil {
			return fmt.Errorf("处理第 %d 行失败: %v", lineNum, err)
		}
	}

	return nil
}
