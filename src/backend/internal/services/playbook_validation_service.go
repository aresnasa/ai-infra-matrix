package services

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ================== 安全配置常量 ==================

// Playbook 最大文件大小限制 (512KB)
const MaxPlaybookSize = 512 * 1024

// Playbook 最大行数限制
const MaxPlaybookLines = 5000

// Playbook 最大任务数限制
const MaxPlaybookTasks = 1000

// Playbook YAML 最大递归深度
const MaxPlaybookYAMLDepth = 15

// Playbook 危险模式检测正则表达式
var playbookDangerousPatterns = []*regexp.Regexp{
	// YAML 特殊攻击标签
	regexp.MustCompile(`(?i)!!python/`),
	regexp.MustCompile(`(?i)!!ruby/`),
	regexp.MustCompile(`(?i)tag:yaml\.org,\d+:python/`),
	regexp.MustCompile(`(?i)!<tag:yaml\.org`),
	// 危险的 shell 命令
	regexp.MustCompile(`(?i)\brm\s+-rf\s+/\s*$`),
	regexp.MustCompile(`(?i)\bdd\s+if=.*of=/dev/`),
	regexp.MustCompile(`(?i)\bmkfs\.[a-z]+\s+/dev/`),
	// 反弹 shell
	regexp.MustCompile(`(?i)nc\s+-[elp].*\d+\s*$`),
	regexp.MustCompile(`(?i)/dev/tcp/`),
	regexp.MustCompile(`(?i)bash\s+-i\s+>&\s*/dev/tcp/`),
	// 环境变量泄露
	regexp.MustCompile(`(?i)env\s*\|\s*(curl|wget|nc)`),
	regexp.MustCompile(`(?i)printenv\s*\|\s*(curl|wget|nc)`),
}

// ValidationResult 校验结果
type ValidationResult struct {
	IsValid  bool              `json:"is_valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Score    int               `json:"score"` // 0-100 分数
}

// ValidationError 校验错误
type ValidationError struct {
	Type       string `json:"type"`     // syntax, structure, module, compatibility
	Severity   string `json:"severity"` // error, warning, info
	Message    string `json:"message"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Field      string `json:"field,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// AnsibleVersionCompatibility Ansible版本兼容性
type AnsibleVersionCompatibility struct {
	Version     string   `json:"version"`
	Supported   bool     `json:"supported"`
	Issues      []string `json:"issues,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// PlaybookValidationService Playbook校验服务
type PlaybookValidationService struct {
	// 支持的Ansible版本
	supportedVersions []string
	// 模块兼容性映射
	moduleCompatibility map[string][]string
}

// NewPlaybookValidationService 创建新的校验服务
func NewPlaybookValidationService() *PlaybookValidationService {
	return &PlaybookValidationService{
		supportedVersions:   []string{"2.9", "2.10", "2.11", "2.12", "2.13", "2.14", "2.15", "2.16", "3.0", "4.0", "5.0", "6.0", "7.0", "8.0", "9.0"},
		moduleCompatibility: initModuleCompatibility(),
	}
}

// ValidatePlaybook 校验Playbook内容
func (s *PlaybookValidationService) ValidatePlaybook(yamlContent []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
		Score:    100,
	}

	// 0. 安全检查
	if err := s.validatePlaybookSecurity(yamlContent, result); err != nil {
		return result, err
	}

	// 1. YAML语法校验
	if err := s.validateYAMLSyntax(yamlContent, result); err != nil {
		logrus.WithError(err).Error("Failed to validate YAML syntax")
		return result, err
	}

	// 2. Ansible结构校验
	var playbooks []interface{}
	if err := yaml.Unmarshal(yamlContent, &playbooks); err == nil {
		s.validateAnsibleStructure(playbooks, result)
		s.validateModules(playbooks, result)
		s.validateBestPractices(playbooks, result)
	}

	// 计算最终分数
	s.calculateScore(result)

	return result, nil
}

// validatePlaybookSecurity 校验 Playbook 安全性
func (s *PlaybookValidationService) validatePlaybookSecurity(yamlContent []byte, result *ValidationResult) error {
	// 1. 检查文件大小
	if len(yamlContent) > MaxPlaybookSize {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "security",
			Severity: "error",
			Message:  fmt.Sprintf("Playbook 文件大小超过限制 (%d bytes，最大 %d bytes)", len(yamlContent), MaxPlaybookSize),
		})
		result.IsValid = false
		return fmt.Errorf("playbook file size exceeds limit")
	}

	// 2. 检查文件编码
	if !utf8.Valid(yamlContent) {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "security",
			Severity: "error",
			Message:  "Playbook 文件编码无效，请使用 UTF-8 编码",
		})
		result.IsValid = false
		return fmt.Errorf("invalid file encoding")
	}

	// 3. 检查行数
	lineCount := strings.Count(string(yamlContent), "\n") + 1
	if lineCount > MaxPlaybookLines {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "security",
			Severity: "error",
			Message:  fmt.Sprintf("Playbook 行数超过限制 (%d 行，最大 %d 行)", lineCount, MaxPlaybookLines),
		})
		result.IsValid = false
		return fmt.Errorf("playbook line count exceeds limit")
	}

	// 4. 检查危险内容
	content := string(yamlContent)
	for _, pattern := range playbookDangerousPatterns {
		if pattern.MatchString(content) {
			match := pattern.FindString(content)
			if len(match) > 50 {
				match = match[:50] + "..."
			}
			result.Errors = append(result.Errors, ValidationError{
				Type:       "security",
				Severity:   "error",
				Message:    fmt.Sprintf("检测到危险内容: %s", match),
				Suggestion: "请检查 Playbook 是否包含恶意代码或危险命令",
			})
			result.IsValid = false
			return fmt.Errorf("dangerous content detected")
		}
	}

	// 5. 检查 YAML 嵌套深度
	if err := s.validatePlaybookYAMLDepth(yamlContent); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "security",
			Severity: "error",
			Message:  err.Error(),
		})
		result.IsValid = false
		return err
	}

	return nil
}

// validatePlaybookYAMLDepth 验证 Playbook YAML 嵌套深度
func (s *PlaybookValidationService) validatePlaybookYAMLDepth(data []byte) error {
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
	if maxIndent > MaxPlaybookYAMLDepth {
		return fmt.Errorf("YAML 嵌套深度超过限制: %d (最大 %d)", maxIndent, MaxPlaybookYAMLDepth)
	}
	return nil
}

// ValidateCompatibility 校验版本兼容性
func (s *PlaybookValidationService) ValidateCompatibility(yamlContent []byte, targetVersions []string) (map[string]*AnsibleVersionCompatibility, error) {
	compatibility := make(map[string]*AnsibleVersionCompatibility)

	var playbooks []interface{}
	if err := yaml.Unmarshal(yamlContent, &playbooks); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	for _, version := range targetVersions {
		compat := &AnsibleVersionCompatibility{
			Version:     version,
			Supported:   true,
			Issues:      []string{},
			Suggestions: []string{},
		}

		s.checkVersionCompatibility(playbooks, version, compat)
		compatibility[version] = compat
	}

	return compatibility, nil
}

// validateYAMLSyntax 校验YAML语法
func (s *PlaybookValidationService) validateYAMLSyntax(yamlContent []byte, result *ValidationResult) error {
	var temp interface{}
	if err := yaml.Unmarshal(yamlContent, &temp); err != nil {
		if yamlErr, ok := err.(*yaml.TypeError); ok {
			for _, errMsg := range yamlErr.Errors {
				result.Errors = append(result.Errors, ValidationError{
					Type:     "syntax",
					Severity: "error",
					Message:  fmt.Sprintf("YAML syntax error: %s", errMsg),
				})
			}
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "syntax",
				Severity: "error",
				Message:  fmt.Sprintf("YAML parsing failed: %s", err.Error()),
			})
		}
		result.IsValid = false
		return nil
	}
	return nil
}

// validateAnsibleStructure 校验Ansible结构
func (s *PlaybookValidationService) validateAnsibleStructure(playbooks []interface{}, result *ValidationResult) {
	if len(playbooks) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Type:       "structure",
			Severity:   "error",
			Message:    "Playbook must contain at least one play",
			Suggestion: "Add at least one play to the playbook",
		})
		result.IsValid = false
		return
	}

	for i, playbook := range playbooks {
		if playMap, ok := playbook.(map[string]interface{}); ok {
			s.validatePlay(playMap, i+1, result)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "structure",
				Severity: "error",
				Message:  fmt.Sprintf("Play %d is not a valid dictionary", i+1),
			})
			result.IsValid = false
		}
	}
}

// validatePlay 校验单个Play
func (s *PlaybookValidationService) validatePlay(play map[string]interface{}, playIndex int, result *ValidationResult) {
	// 检查必需字段
	requiredFields := []string{"hosts"}
	for _, field := range requiredFields {
		if _, exists := play[field]; !exists {
			result.Errors = append(result.Errors, ValidationError{
				Type:       "structure",
				Severity:   "error",
				Field:      field,
				Message:    fmt.Sprintf("Play %d missing required field: %s", playIndex, field),
				Suggestion: fmt.Sprintf("Add '%s' field to the play", field),
			})
			result.IsValid = false
		}
	}

	// 检查hosts字段
	if hosts, exists := play["hosts"]; exists {
		if hostsStr, ok := hosts.(string); ok {
			if strings.TrimSpace(hostsStr) == "" {
				result.Errors = append(result.Errors, ValidationError{
					Type:       "structure",
					Severity:   "error",
					Field:      "hosts",
					Message:    fmt.Sprintf("Play %d has empty hosts field", playIndex),
					Suggestion: "Specify target hosts or use 'all' for all hosts",
				})
				result.IsValid = false
			}
		}
	}

	// 检查tasks
	if tasks, exists := play["tasks"]; exists {
		if tasksList, ok := tasks.([]interface{}); ok {
			s.validateTasks(tasksList, playIndex, result)
		}
	}

	// 检查vars
	if vars, exists := play["vars"]; exists {
		s.validateVars(vars, playIndex, result)
	}
}

// validateTasks 校验任务列表
func (s *PlaybookValidationService) validateTasks(tasks []interface{}, playIndex int, result *ValidationResult) {
	if len(tasks) == 0 {
		result.Warnings = append(result.Warnings, ValidationError{
			Type:       "structure",
			Severity:   "warning",
			Message:    fmt.Sprintf("Play %d has no tasks", playIndex),
			Suggestion: "Add tasks to make the play functional",
		})
	}

	for i, task := range tasks {
		if taskMap, ok := task.(map[string]interface{}); ok {
			s.validateTask(taskMap, playIndex, i+1, result)
		}
	}
}

// validateTask 校验单个任务
func (s *PlaybookValidationService) validateTask(task map[string]interface{}, playIndex, taskIndex int, result *ValidationResult) {
	// 检查是否有name
	if _, hasName := task["name"]; !hasName {
		result.Warnings = append(result.Warnings, ValidationError{
			Type:       "best_practice",
			Severity:   "warning",
			Message:    fmt.Sprintf("Task %d in play %d has no name", taskIndex, playIndex),
			Suggestion: "Add a descriptive name to the task",
		})
	}

	// 检查是否有可执行的模块
	hasModule := false
	for key := range task {
		if key != "name" && key != "when" && key != "tags" && key != "vars" &&
			key != "register" && key != "become" && key != "become_user" &&
			key != "ignore_errors" && key != "changed_when" && key != "failed_when" {
			hasModule = true
			break
		}
	}

	if !hasModule {
		result.Errors = append(result.Errors, ValidationError{
			Type:       "structure",
			Severity:   "error",
			Message:    fmt.Sprintf("Task %d in play %d has no executable module", taskIndex, playIndex),
			Suggestion: "Add a module to the task (e.g., shell, command, copy, etc.)",
		})
		result.IsValid = false
	}
}

// validateVars 校验变量
func (s *PlaybookValidationService) validateVars(vars interface{}, playIndex int, result *ValidationResult) {
	if varsMap, ok := vars.(map[string]interface{}); ok {
		for varName, varValue := range varsMap {
			// 检查变量名格式
			if !isValidVariableName(varName) {
				result.Warnings = append(result.Warnings, ValidationError{
					Type:       "best_practice",
					Severity:   "warning",
					Field:      varName,
					Message:    fmt.Sprintf("Variable name '%s' in play %d should follow snake_case convention", varName, playIndex),
					Suggestion: "Use snake_case for variable names (e.g., my_variable)",
				})
			}

			// 检查变量值
			s.validateVariableValue(varName, varValue, playIndex, result)
		}
	}
}

// validateVariableValue 校验变量值
func (s *PlaybookValidationService) validateVariableValue(name string, value interface{}, playIndex int, result *ValidationResult) {
	if valueStr, ok := value.(string); ok {
		// 检查是否包含未转义的特殊字符
		if strings.Contains(valueStr, "{{") && !strings.Contains(valueStr, "}}") {
			result.Warnings = append(result.Warnings, ValidationError{
				Type:       "template",
				Severity:   "warning",
				Field:      name,
				Message:    fmt.Sprintf("Variable '%s' in play %d may have malformed Jinja2 template", name, playIndex),
				Suggestion: "Ensure Jinja2 templates are properly closed with }}",
			})
		}
	}
}

// validateModules 校验模块使用
func (s *PlaybookValidationService) validateModules(playbooks []interface{}, result *ValidationResult) {
	for playIndex, playbook := range playbooks {
		if playMap, ok := playbook.(map[string]interface{}); ok {
			if tasks, exists := playMap["tasks"]; exists {
				if tasksList, ok := tasks.([]interface{}); ok {
					s.validateTaskModules(tasksList, playIndex+1, result)
				}
			}
		}
	}
}

// validateTaskModules 校验任务模块
func (s *PlaybookValidationService) validateTaskModules(tasks []interface{}, playIndex int, result *ValidationResult) {
	for taskIndex, task := range tasks {
		if taskMap, ok := task.(map[string]interface{}); ok {
			for moduleName := range taskMap {
				if moduleName != "name" && moduleName != "when" && moduleName != "tags" &&
					moduleName != "vars" && moduleName != "register" && moduleName != "become" &&
					moduleName != "become_user" && moduleName != "ignore_errors" &&
					moduleName != "changed_when" && moduleName != "failed_when" {
					s.validateModule(moduleName, playIndex, taskIndex+1, result)
					break // 只检查第一个模块
				}
			}
		}
	}
}

// validateModule 校验单个模块
func (s *PlaybookValidationService) validateModule(moduleName string, playIndex, taskIndex int, result *ValidationResult) {
	// 检查是否是已知的模块
	knownModules := []string{
		"shell", "command", "copy", "file", "template", "lineinfile", "replace",
		"service", "systemd", "package", "yum", "apt", "pip", "git", "unarchive",
		"get_url", "uri", "debug", "ping", "setup", "set_fact", "include_tasks",
		"import_tasks", "include_role", "import_role", "block", "meta", "fail",
		"assert", "pause", "wait_for", "cron", "user", "group", "mount", "filesystem",
	}

	isKnown := false
	for _, known := range knownModules {
		if moduleName == known {
			isKnown = true
			break
		}
	}

	if !isKnown {
		result.Warnings = append(result.Warnings, ValidationError{
			Type:       "module",
			Severity:   "warning",
			Message:    fmt.Sprintf("Unknown or uncommon module '%s' in task %d of play %d", moduleName, taskIndex, playIndex),
			Suggestion: "Verify module name and ensure it's available in your Ansible installation",
		})
	}

	// 检查已废弃的模块
	deprecatedModules := map[string]string{
		"raw":     "Consider using shell or command module instead",
		"include": "Use include_tasks or import_tasks instead",
	}

	if suggestion, isDeprecated := deprecatedModules[moduleName]; isDeprecated {
		result.Warnings = append(result.Warnings, ValidationError{
			Type:       "compatibility",
			Severity:   "warning",
			Message:    fmt.Sprintf("Module '%s' in task %d of play %d is deprecated", moduleName, taskIndex, playIndex),
			Suggestion: suggestion,
		})
	}
}

// validateBestPractices 校验最佳实践
func (s *PlaybookValidationService) validateBestPractices(playbooks []interface{}, result *ValidationResult) {
	for playIndex, playbook := range playbooks {
		if playMap, ok := playbook.(map[string]interface{}); ok {
			// 检查是否有name
			if _, hasName := playMap["name"]; !hasName {
				result.Warnings = append(result.Warnings, ValidationError{
					Type:       "best_practice",
					Severity:   "warning",
					Message:    fmt.Sprintf("Play %d has no name", playIndex+1),
					Suggestion: "Add a descriptive name to the play",
				})
			}

			// 检查是否使用become
			if become, exists := playMap["become"]; exists {
				if becomeVal, ok := become.(bool); ok && becomeVal {
					// 检查是否指定了become_user
					if _, hasBecomeUser := playMap["become_user"]; !hasBecomeUser {
						result.Warnings = append(result.Warnings, ValidationError{
							Type:       "security",
							Severity:   "warning",
							Message:    fmt.Sprintf("Play %d uses become without specifying become_user", playIndex+1),
							Suggestion: "Explicitly specify become_user for security clarity",
						})
					}
				}
			}
		}
	}
}

// checkVersionCompatibility 检查版本兼容性
func (s *PlaybookValidationService) checkVersionCompatibility(playbooks []interface{}, version string, compat *AnsibleVersionCompatibility) {
	// 基于版本检查特定功能兼容性
	versionFloat := parseVersion(version)

	for _, playbook := range playbooks {
		if playMap, ok := playbook.(map[string]interface{}); ok {
			// 检查collections字段 (Ansible 2.9+)
			if _, hasCollections := playMap["collections"]; hasCollections && versionFloat < 2.9 {
				compat.Issues = append(compat.Issues, "collections field requires Ansible 2.9+")
				compat.Supported = false
			}

			// 检查任务中的新功能
			if tasks, exists := playMap["tasks"]; exists {
				if tasksList, ok := tasks.([]interface{}); ok {
					s.checkTasksCompatibility(tasksList, version, versionFloat, compat)
				}
			}
		}
	}
}

// checkTasksCompatibility 检查任务兼容性
func (s *PlaybookValidationService) checkTasksCompatibility(tasks []interface{}, version string, versionFloat float64, compat *AnsibleVersionCompatibility) {
	for _, task := range tasks {
		if taskMap, ok := task.(map[string]interface{}); ok {
			// 检查特定模块的版本要求
			for moduleName := range taskMap {
				if moduleVersions, exists := s.moduleCompatibility[moduleName]; exists {
					supported := false
					for _, supportedVersion := range moduleVersions {
						if parseVersion(supportedVersion) <= versionFloat {
							supported = true
							break
						}
					}
					if !supported {
						compat.Issues = append(compat.Issues,
							fmt.Sprintf("Module '%s' may not be available in Ansible %s", moduleName, version))
						compat.Suggestions = append(compat.Suggestions,
							fmt.Sprintf("Consider upgrading Ansible or using alternative module for '%s'", moduleName))
					}
				}
			}
		}
	}
}

// calculateScore 计算校验分数
func (s *PlaybookValidationService) calculateScore(result *ValidationResult) {
	score := 100

	// 错误扣分更多
	for _, err := range result.Errors {
		switch err.Severity {
		case "error":
			score -= 15
		case "warning":
			score -= 5
		}
	}

	// 警告扣分较少
	for _, warning := range result.Warnings {
		switch warning.Severity {
		case "warning":
			score -= 3
		case "info":
			score -= 1
		}
	}

	if score < 0 {
		score = 0
	}

	result.Score = score
}

// 辅助函数

// isValidVariableName 检查变量名是否有效
func isValidVariableName(name string) bool {
	// Ansible变量名应该是snake_case，包含字母、数字和下划线
	matched, _ := regexp.MatchString("^[a-z][a-z0-9_]*$", name)
	return matched
}

// parseVersion 解析版本号
func parseVersion(version string) float64 {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		major := 0
		minor := 0
		fmt.Sscanf(parts[0], "%d", &major)
		fmt.Sscanf(parts[1], "%d", &minor)
		return float64(major) + float64(minor)/10.0
	}
	return 0.0
}

// initModuleCompatibility 初始化模块兼容性映射
func initModuleCompatibility() map[string][]string {
	return map[string][]string{
		"collections":                          {"2.9"},
		"podman_image":                         {"2.10"},
		"podman_container":                     {"2.10"},
		"community.general.telegram":           {"2.9"},
		"ansible.posix.synchronize":            {"2.9"},
		"community.crypto.openssl_certificate": {"2.9"},
	}
}
