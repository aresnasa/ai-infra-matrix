package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// PreviewContent 预览内容
type PreviewContent struct {
	PlaybookYAML    string `json:"playbook_yaml"`
	InventoryINI    string `json:"inventory_ini"`
	ReadmeContent   string `json:"readme_content"`
	ValidationScore int    `json:"validation_score"`
	IsValid         bool   `json:"is_valid"`
}

// DownloadPackage 下载包信息
type DownloadPackage struct {
	ZipPath     string   `json:"zip_path"`
	Files       []string `json:"files"`
	TotalSize   int64    `json:"total_size"`
	GeneratedAt string   `json:"generated_at"`
}

// PlaybookPreviewService Playbook预览服务
type PlaybookPreviewService struct {
	playbookService   *PlaybookService
	validationService *PlaybookValidationService
}

// NewPlaybookPreviewService 创建新的预览服务
func NewPlaybookPreviewService() *PlaybookPreviewService {
	return &PlaybookPreviewService{
		playbookService:   NewPlaybookService(),
		validationService: NewPlaybookValidationService(),
	}
}

// GeneratePreview 生成预览内容
func (s *PlaybookPreviewService) GeneratePreview(projectID uint, userID uint) (*PreviewContent, error) {
	// 获取项目信息
	projectService := NewProjectService()
	rbacService := NewRBACService(database.DB)
	project, err := projectService.GetProject(projectID, userID, rbacService)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// 生成playbook内容
	playbookContent, err := s.playbookService.buildPlaybookContent(project)
	if err != nil {
		return nil, fmt.Errorf("failed to build playbook content: %w", err)
	}

	// 转换为YAML字符串
	yamlBytes, err := yaml.Marshal(playbookContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal playbook to YAML: %w", err)
	}

	// 添加文件头注释
	yamlString := s.addPlaybookHeader(project, string(yamlBytes))

	// 生成inventory内容
	inventoryContent := s.generateInventoryPreview(project.Hosts)

	// 生成README内容
	readmeContent := s.generateReadmeContent(project)

	// 校验playbook
	validationResult, err := s.validationService.ValidatePlaybook([]byte(yamlString))
	if err != nil {
		logrus.WithError(err).Warn("Failed to validate playbook during preview")
		validationResult = &ValidationResult{IsValid: false, Score: 0}
	}

	return &PreviewContent{
		PlaybookYAML:    yamlString,
		InventoryINI:    inventoryContent,
		ReadmeContent:   readmeContent,
		ValidationScore: validationResult.Score,
		IsValid:         validationResult.IsValid,
	}, nil
}

// GenerateDownloadPackage 生成下载包
func (s *PlaybookPreviewService) GenerateDownloadPackage(projectID uint, userID uint) (*DownloadPackage, error) {
	// 获取项目信息
	projectService := NewProjectService()
	rbacService := NewRBACService(database.DB)
	project, err := projectService.GetProject(projectID, userID, rbacService)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// 创建临时目录，使用绝对路径
	timestamp := time.Now().Unix()
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	tempDir := filepath.Join(currentDir, "outputs", "packages", fmt.Sprintf("package-%s-%d",
		strings.ReplaceAll(project.Name, " ", "-"), timestamp))

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// 生成文件内容
	preview, err := s.GeneratePreview(projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview: %w", err)
	}

	// 创建文件
	files := []string{}

	// 1. 创建playbook文件
	playbookFile := filepath.Join(tempDir, "playbook.yml")
	if err := s.writeFile(playbookFile, preview.PlaybookYAML); err != nil {
		return nil, fmt.Errorf("failed to write playbook file: %w", err)
	}
	files = append(files, "playbook.yml")

	// 2. 创建inventory文件
	inventoryFile := filepath.Join(tempDir, "inventory.ini")
	if err := s.writeFile(inventoryFile, preview.InventoryINI); err != nil {
		return nil, fmt.Errorf("failed to write inventory file: %w", err)
	}
	files = append(files, "inventory.ini")

	// 3. 创建README文件
	readmeFile := filepath.Join(tempDir, "README.md")
	if err := s.writeFile(readmeFile, preview.ReadmeContent); err != nil {
		return nil, fmt.Errorf("failed to write README file: %w", err)
	}
	files = append(files, "README.md")

	// 4. 创建ansible.cfg配置文件
	ansibleCfgFile := filepath.Join(tempDir, "ansible.cfg")
	ansibleCfgContent := s.generateAnsibleCfg(project)
	if err := s.writeFile(ansibleCfgFile, ansibleCfgContent); err != nil {
		return nil, fmt.Errorf("failed to write ansible.cfg file: %w", err)
	}
	files = append(files, "ansible.cfg")

	// 5. 创建执行脚本
	runScriptFile := filepath.Join(tempDir, "run.sh")
	runScriptContent := s.generateRunScript(project)
	if err := s.writeFile(runScriptFile, runScriptContent); err != nil {
		return nil, fmt.Errorf("failed to write run script: %w", err)
	}
	// 设置执行权限
	if err := os.Chmod(runScriptFile, 0755); err != nil {
		logrus.WithError(err).Warn("Failed to set execute permission on run script")
	}
	files = append(files, "run.sh")

	// 6. 如果有变量文件，创建group_vars目录
	if len(project.Variables) > 0 {
		groupVarsDir := filepath.Join(tempDir, "group_vars")
		if err := os.MkdirAll(groupVarsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create group_vars directory: %w", err)
		}

		allVarsFile := filepath.Join(groupVarsDir, "all.yml")
		varsContent := s.generateGroupVars(project.Variables)
		if err := s.writeFile(allVarsFile, varsContent); err != nil {
			return nil, fmt.Errorf("failed to write group vars file: %w", err)
		}
		files = append(files, "group_vars/all.yml")
	}

	// 创建ZIP包
	zipFileName := fmt.Sprintf("ansible-playbook-%s-%d.zip",
		strings.ReplaceAll(project.Name, " ", "-"), timestamp)
	zipPath := filepath.Join(currentDir, "outputs", "packages", zipFileName)

	if err := s.createZipArchive(tempDir, zipPath, files); err != nil {
		return nil, fmt.Errorf("failed to create ZIP archive: %w", err)
	}

	// 获取文件大小
	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get ZIP file info: %w", err)
	}

	// 清理临时目录
	if err := os.RemoveAll(tempDir); err != nil {
		logrus.WithError(err).Warn("Failed to clean up temp directory")
	}

	return &DownloadPackage{
		ZipPath:     zipFileName, // 只返回文件名而不是完整路径
		Files:       files,
		TotalSize:   zipInfo.Size(),
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// ValidatePreview 校验预览内容
func (s *PlaybookPreviewService) ValidatePreview(projectID uint, userID uint) (*ValidationResult, error) {
	preview, err := s.GeneratePreview(projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview: %w", err)
	}

	return s.validationService.ValidatePlaybook([]byte(preview.PlaybookYAML))
}

// addPlaybookHeader 添加playbook文件头
func (s *PlaybookPreviewService) addPlaybookHeader(project *models.Project, yamlContent string) string {
	header := fmt.Sprintf(`---
# Ansible Playbook: %s
# 生成时间: %s
# 描述: %s
# 
# 使用方法:
#   ansible-playbook -i inventory.ini playbook.yml
# 
# 注意事项:
#   - 请确保目标主机已配置SSH密钥认证
#   - 检查目标主机的sudo权限配置
#   - 根据实际需求调整变量值
#

`, project.Name, time.Now().Format("2006-01-02 15:04:05"), project.Description)

	return header + yamlContent
}

// generateInventoryPreview 生成inventory预览
func (s *PlaybookPreviewService) generateInventoryPreview(hosts []models.Host) string {
	if len(hosts) == 0 {
		return `# Ansible Inventory
# 没有配置主机

[all]
# 请添加主机配置
# example.com ansible_host=192.168.1.100 ansible_user=root
`
	}

	var buf bytes.Buffer
	buf.WriteString("# Ansible Inventory\n")
	buf.WriteString(fmt.Sprintf("# 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 按组分类主机
	groups := make(map[string][]models.Host)
	for _, host := range hosts {
		group := host.Group
		if group == "" {
			group = "ungrouped"
		}
		groups[group] = append(groups[group], host)
	}

	// 写入各组的主机配置
	for group, groupHosts := range groups {
		buf.WriteString(fmt.Sprintf("[%s]\n", group))
		for _, host := range groupHosts {
			buf.WriteString(fmt.Sprintf("%s ansible_host=%s ansible_port=%d ansible_user=%s\n",
				host.Name, host.IP, host.Port, host.User))
		}
		buf.WriteString("\n")
	}

	// 添加所有主机组
	if len(groups) > 1 {
		buf.WriteString("[all:children]\n")
		for group := range groups {
			if group != "ungrouped" {
				buf.WriteString(fmt.Sprintf("%s\n", group))
			}
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// generateReadmeContent 生成README内容
func (s *PlaybookPreviewService) generateReadmeContent(project *models.Project) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# %s\n\n", project.Name))
	buf.WriteString(fmt.Sprintf("**描述**: %s\n\n", project.Description))
	buf.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	buf.WriteString("## 文件说明\n\n")
	buf.WriteString("- `playbook.yml` - 主要的Ansible Playbook文件\n")
	buf.WriteString("- `inventory.ini` - 主机清单文件\n")
	buf.WriteString("- `ansible.cfg` - Ansible配置文件\n")
	buf.WriteString("- `run.sh` - 执行脚本\n")
	if len(project.Variables) > 0 {
		buf.WriteString("- `group_vars/all.yml` - 全局变量文件\n")
	}
	buf.WriteString("\n")

	buf.WriteString("## 使用方法\n\n")
	buf.WriteString("### 1. 准备环境\n\n")
	buf.WriteString("确保已安装Ansible:\n")
	buf.WriteString("```bash\n")
	buf.WriteString("# Ubuntu/Debian\n")
	buf.WriteString("sudo apt update && sudo apt install ansible\n\n")
	buf.WriteString("# CentOS/RHEL\n")
	buf.WriteString("sudo yum install ansible\n\n")
	buf.WriteString("# macOS\n")
	buf.WriteString("brew install ansible\n")
	buf.WriteString("```\n\n")

	buf.WriteString("### 2. 配置SSH访问\n\n")
	buf.WriteString("确保可以SSH连接到目标主机:\n")
	buf.WriteString("```bash\n")
	buf.WriteString("# 测试连接\n")
	buf.WriteString("ssh user@target-host\n\n")
	buf.WriteString("# 或配置SSH密钥\n")
	buf.WriteString("ssh-copy-id user@target-host\n")
	buf.WriteString("```\n\n")

	buf.WriteString("### 3. 执行Playbook\n\n")
	buf.WriteString("#### 方法一：使用提供的脚本\n")
	buf.WriteString("```bash\n")
	buf.WriteString("chmod +x run.sh\n")
	buf.WriteString("./run.sh\n")
	buf.WriteString("```\n\n")

	buf.WriteString("#### 方法二：直接执行\n")
	buf.WriteString("```bash\n")
	buf.WriteString("# 检查语法\n")
	buf.WriteString("ansible-playbook --syntax-check -i inventory.ini playbook.yml\n\n")
	buf.WriteString("# 干运行（不实际执行）\n")
	buf.WriteString("ansible-playbook --check -i inventory.ini playbook.yml\n\n")
	buf.WriteString("# 正式执行\n")
	buf.WriteString("ansible-playbook -i inventory.ini playbook.yml\n")
	buf.WriteString("```\n\n")

	buf.WriteString("### 4. 高级选项\n\n")
	buf.WriteString("```bash\n")
	buf.WriteString("# 限制执行的主机\n")
	buf.WriteString("ansible-playbook -i inventory.ini playbook.yml --limit \"web_servers\"\n\n")
	buf.WriteString("# 指定用户\n")
	buf.WriteString("ansible-playbook -i inventory.ini playbook.yml --user myuser\n\n")
	buf.WriteString("# 使用sudo\n")
	buf.WriteString("ansible-playbook -i inventory.ini playbook.yml --become\n\n")
	buf.WriteString("# 详细输出\n")
	buf.WriteString("ansible-playbook -i inventory.ini playbook.yml -vvv\n")
	buf.WriteString("```\n\n")

	if len(project.Variables) > 0 {
		buf.WriteString("## 变量说明\n\n")
		buf.WriteString("以下变量在 `group_vars/all.yml` 中定义，您可以根据需要修改:\n\n")
		for _, variable := range project.Variables {
			buf.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", variable.Name, variable.Type, variable.Value))
		}
		buf.WriteString("\n")
	}

	if len(project.Hosts) > 0 {
		buf.WriteString("## 主机清单\n\n")
		buf.WriteString("| 主机名 | IP地址 | 端口 | 用户 | 组 |\n")
		buf.WriteString("|--------|--------|------|------|----|\n")
		for _, host := range project.Hosts {
			group := host.Group
			if group == "" {
				group = "ungrouped"
			}
			buf.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s |\n",
				host.Name, host.IP, host.Port, host.User, group))
		}
		buf.WriteString("\n")
	}

	if len(project.Tasks) > 0 {
		buf.WriteString("## 任务说明\n\n")
		for i, task := range project.Tasks {
			if task.Enabled {
				buf.WriteString(fmt.Sprintf("%d. **%s** - 使用 `%s` 模块\n", i+1, task.Name, task.Module))
				if task.Args != "" {
					buf.WriteString(fmt.Sprintf("   - 参数: `%s`\n", task.Args))
				}
			}
		}
		buf.WriteString("\n")
	}

	buf.WriteString("## 故障排除\n\n")
	buf.WriteString("### 常见问题\n\n")
	buf.WriteString("1. **SSH连接失败**\n")
	buf.WriteString("   - 检查SSH密钥配置\n")
	buf.WriteString("   - 确认目标主机防火墙设置\n")
	buf.WriteString("   - 验证用户名和端口配置\n\n")

	buf.WriteString("2. **权限不足**\n")
	buf.WriteString("   - 使用 `--become` 参数获取sudo权限\n")
	buf.WriteString("   - 检查目标主机的sudo配置\n")
	buf.WriteString("   - 确认用户在sudoers文件中\n\n")

	buf.WriteString("3. **模块未找到**\n")
	buf.WriteString("   - 检查Ansible版本兼容性\n")
	buf.WriteString("   - 更新Ansible到最新版本\n")
	buf.WriteString("   - 安装相关的Ansible集合\n\n")

	buf.WriteString("### 日志和调试\n\n")
	buf.WriteString("```bash\n")
	buf.WriteString("# 启用详细日志\n")
	buf.WriteString("export ANSIBLE_LOG_PATH=./ansible.log\n")
	buf.WriteString("ansible-playbook -i inventory.ini playbook.yml -vvv\n\n")
	buf.WriteString("# 检查连接\n")
	buf.WriteString("ansible all -i inventory.ini -m ping\n")
	buf.WriteString("```\n\n")

	buf.WriteString("## 版本兼容性\n\n")
	buf.WriteString("此Playbook已在以下版本测试:\n")
	buf.WriteString("- Ansible 2.9+\n")
	buf.WriteString("- Python 3.6+\n\n")

	buf.WriteString("---\n")
	buf.WriteString("*此文档由Ansible Playbook生成器自动生成*\n")

	return buf.String()
}

// generateAnsibleCfg 生成ansible.cfg配置
func (s *PlaybookPreviewService) generateAnsibleCfg(project *models.Project) string {
	return `[defaults]
# 主机清单文件
inventory = inventory.ini

# 禁用主机密钥检查（仅用于测试环境）
host_key_checking = False

# 设置并发执行数
forks = 5

# 超时设置
timeout = 30

# 日志路径（可选）
# log_path = ansible.log

# 重试文件设置
retry_files_enabled = False

# 角色路径
roles_path = roles

# 收集facts超时时间
gather_timeout = 30

[ssh_connection]
# SSH连接超时
ssh_timeout = 30

# SSH参数
ssh_args = -o ControlMaster=auto -o ControlPersist=60s

# 禁用SSH管道
pipelining = True

[privilege_escalation]
# 默认不使用sudo（在playbook中按需指定）
become = False
`
}

// generateRunScript 生成执行脚本
func (s *PlaybookPreviewService) generateRunScript(project *models.Project) string {
	return fmt.Sprintf(`#!/bin/bash
# Ansible Playbook 执行脚本
# 项目: %s
# 生成时间: %s

set -e

echo "=== Ansible Playbook 执行脚本 ==="
echo "项目: %s"
echo "描述: %s"
echo ""

# 检查Ansible是否安装
if ! command -v ansible-playbook &> /dev/null; then
    echo "错误: 未找到ansible-playbook命令"
    echo "请先安装Ansible:"
    echo "  Ubuntu/Debian: sudo apt install ansible"
    echo "  CentOS/RHEL:   sudo yum install ansible"
    echo "  macOS:         brew install ansible"
    exit 1
fi

# 检查必要文件
if [ ! -f "playbook.yml" ]; then
    echo "错误: 未找到playbook.yml文件"
    exit 1
fi

if [ ! -f "inventory.ini" ]; then
    echo "错误: 未找到inventory.ini文件"
    exit 1
fi

echo "1. 检查语法..."
if ! ansible-playbook --syntax-check -i inventory.ini playbook.yml; then
    echo "错误: Playbook语法检查失败"
    exit 1
fi
echo "   语法检查通过 ✓"

echo ""
echo "2. 测试连接..."
if ! ansible all -i inventory.ini -m ping; then
    echo "警告: 部分主机连接失败"
    read -p "是否继续执行? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "取消执行"
        exit 1
    fi
else
    echo "   连接测试通过 ✓"
fi

echo ""
echo "3. 选择执行模式:"
echo "   1) 干运行（检查模式，不实际执行）"
echo "   2) 正常执行"
echo "   3) 详细模式执行"
read -p "请选择 (1-3): " -n 1 -r
echo

case $REPLY in
    1)
        echo "执行干运行..."
        ansible-playbook --check -i inventory.ini playbook.yml
        ;;
    2)
        echo "正常执行..."
        ansible-playbook -i inventory.ini playbook.yml
        ;;
    3)
        echo "详细模式执行..."
        ansible-playbook -i inventory.ini playbook.yml -vvv
        ;;
    *)
        echo "无效选择，默认使用干运行模式"
        ansible-playbook --check -i inventory.ini playbook.yml
        ;;
esac

echo ""
echo "=== 执行完成 ==="
`, project.Name, time.Now().Format("2006-01-02 15:04:05"), project.Name, project.Description)
}

// generateGroupVars 生成group_vars文件内容
func (s *PlaybookPreviewService) generateGroupVars(variables []models.Variable) string {
	var buf bytes.Buffer

	buf.WriteString("---\n")
	buf.WriteString("# 全局变量配置\n")
	buf.WriteString(fmt.Sprintf("# 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString("# 可以根据实际需求修改以下变量值\n\n")

	for _, variable := range variables {
		buf.WriteString(fmt.Sprintf("# %s (%s)\n", variable.Name, variable.Type))

		// 根据类型格式化变量值
		switch variable.Type {
		case "number":
			if strings.Contains(variable.Value, ".") {
				buf.WriteString(fmt.Sprintf("%s: %s\n", variable.Name, variable.Value))
			} else {
				buf.WriteString(fmt.Sprintf("%s: %s\n", variable.Name, variable.Value))
			}
		case "boolean":
			value := strings.ToLower(variable.Value) == "true"
			buf.WriteString(fmt.Sprintf("%s: %t\n", variable.Name, value))
		case "list":
			buf.WriteString(fmt.Sprintf("%s:\n", variable.Name))
			items := strings.Split(variable.Value, ",")
			for _, item := range items {
				buf.WriteString(fmt.Sprintf("  - %s\n", strings.TrimSpace(item)))
			}
		default:
			buf.WriteString(fmt.Sprintf("%s: \"%s\"\n", variable.Name, variable.Value))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// writeFile 写入文件内容
func (s *PlaybookPreviewService) writeFile(filePath, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// createZipArchive 创建ZIP压缩包
func (s *PlaybookPreviewService) createZipArchive(sourceDir, zipPath string, files []string) error {
	// 创建ZIP文件
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// 创建ZIP写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 添加文件到ZIP
	for _, file := range files {
		sourceFile := filepath.Join(sourceDir, file)

		// 读取文件内容
		fileContent, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// 创建ZIP文件条目
		zipEntry, err := zipWriter.Create(file)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", file, err)
		}

		// 写入文件内容
		if _, err := io.Copy(zipEntry, bytes.NewReader(fileContent)); err != nil {
			return fmt.Errorf("failed to write file %s to zip: %w", file, err)
		}
	}

	return nil
}
