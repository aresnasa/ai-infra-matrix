package services

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/sirupsen/logrus"
)

//go:embed scripts/*.sh scripts/**/*.sh scripts/**/*.tmpl
var embeddedScripts embed.FS

// ScriptLoader 脚本加载器服务
// 优先从文件系统加载脚本，如果不存在则从嵌入的脚本中加载
// 设计说明：
//   - 脚本模板使用 Go text/template 语法
//   - 优先从 SCRIPTS_DIR 环境变量指定的目录加载（便于运维修改）
//   - 文件系统找不到时回退到嵌入资源（确保程序可独立运行）
//   - 模板文件使用 .tmpl 后缀
type ScriptLoader struct {
	scriptsDir    string
	templateCache map[string]*template.Template
	cacheMutex    sync.RWMutex
}

// ScriptParams 脚本参数基础结构
type ScriptParams map[string]interface{}

// SaltInstallParams Salt Minion 安装参数
type SaltInstallParams struct {
	AppHubURL    string // AppHub 服务地址，用于下载安装包
	MasterHost   string // Salt Master 主机地址
	MinionID     string // Minion 标识符
	Version      string // Salt 版本号
	Arch         string // DEB 包架构 (amd64, arm64)
	RpmArch      string // RPM 包架构 (x86_64, aarch64)
	SudoPrefix   string // sudo 前缀 (空或 "sudo ")
	OS           string // 操作系统类型 (ubuntu, debian, centos, rhel, etc.)
	OSVersion    string // 操作系统版本
	MasterPubURL string // Master 公钥下载 URL（一次性令牌）
}

// SaltUninstallParams Salt Minion 卸载参数
type SaltUninstallParams struct {
	SudoPrefix string // sudo 前缀 (空或 "sudo ")
	OS         string // 操作系统类型
}

// SSHTestParams SSH 测试参数
type SSHTestParams struct {
	SudoPass string // sudo 密码（用于测试 sudo 权限）
}

// OSDetectParams 操作系统检测参数
type OSDetectParams struct{}

// NodeMetricsDeployParams 节点指标采集部署参数
type NodeMetricsDeployParams struct {
	CallbackURL     string // 指标回调 URL
	CollectInterval string // 采集间隔（分钟）
	APIToken        string // API 认证令牌
	MinionID        string // Minion 标识符
	CollectScript   string // 采集脚本内容
}

// 模板文件映射 - 定义各类脚本对应的模板文件
var templateFiles = map[string]map[string]string{
	"salt-install": {
		"ubuntu":    "templates/salt-install-debian.sh.tmpl",
		"debian":    "templates/salt-install-debian.sh.tmpl",
		"centos":    "templates/salt-install-rhel.sh.tmpl",
		"rhel":      "templates/salt-install-rhel.sh.tmpl",
		"rocky":     "templates/salt-install-rhel.sh.tmpl",
		"almalinux": "templates/salt-install-rhel.sh.tmpl",
		"fedora":    "templates/salt-install-rhel.sh.tmpl",
		"default":   "templates/salt-install-generic.sh.tmpl",
	},
	"salt-uninstall": {
		"ubuntu":    "templates/salt-uninstall-debian.sh.tmpl",
		"debian":    "templates/salt-uninstall-debian.sh.tmpl",
		"centos":    "templates/salt-uninstall-rhel.sh.tmpl",
		"rhel":      "templates/salt-uninstall-rhel.sh.tmpl",
		"rocky":     "templates/salt-uninstall-rhel.sh.tmpl",
		"almalinux": "templates/salt-uninstall-rhel.sh.tmpl",
		"fedora":    "templates/salt-uninstall-rhel.sh.tmpl",
		"default":   "templates/salt-uninstall-generic.sh.tmpl",
	},
	"categraf-install": {
		"ubuntu":    "templates/categraf-install-debian.sh.tmpl",
		"debian":    "templates/categraf-install-debian.sh.tmpl",
		"centos":    "templates/categraf-install-rhel.sh.tmpl",
		"rhel":      "templates/categraf-install-rhel.sh.tmpl",
		"rocky":     "templates/categraf-install-rhel.sh.tmpl",
		"almalinux": "templates/categraf-install-rhel.sh.tmpl",
		"fedora":    "templates/categraf-install-rhel.sh.tmpl",
		"default":   "templates/categraf-install-debian.sh.tmpl",
	},
	"node-metrics-deploy": {
		"default": "templates/node-metrics-deploy.sh.tmpl",
	},
}

var (
	scriptLoaderInstance *ScriptLoader
	scriptLoaderOnce     sync.Once
)

// GetScriptLoader 获取脚本加载器单例
func GetScriptLoader() *ScriptLoader {
	scriptLoaderOnce.Do(func() {
		scriptsDir := os.Getenv("SCRIPTS_DIR")
		if scriptsDir == "" {
			// 默认在可执行文件同目录的 scripts 文件夹
			execPath, _ := os.Executable()
			scriptsDir = filepath.Join(filepath.Dir(execPath), "scripts")
		}

		scriptLoaderInstance = &ScriptLoader{
			scriptsDir:    scriptsDir,
			templateCache: make(map[string]*template.Template),
		}

		logrus.Infof("[ScriptLoader] 初始化完成")
		logrus.Infof("[ScriptLoader] 外部脚本目录: %s", scriptsDir)
		logrus.Infof("[ScriptLoader] 运维人员可修改外部脚本目录中的模板文件，无需重新编译程序")
	})
	return scriptLoaderInstance
}

// NewScriptLoader 创建脚本加载器（用于测试）
func NewScriptLoader(scriptsDir string) *ScriptLoader {
	return &ScriptLoader{
		scriptsDir:    scriptsDir,
		templateCache: make(map[string]*template.Template),
	}
}

// GetScript 获取脚本内容
// scriptName: 脚本名称，如 "install-salt-minion.sh" 或 "salt-minion/01-install-salt-minion.sh"
func (s *ScriptLoader) GetScript(scriptName string) (string, error) {
	// 1. 优先从文件系统加载（便于运维修改）
	fsPath := filepath.Join(s.scriptsDir, scriptName)
	if content, err := os.ReadFile(fsPath); err == nil {
		logrus.Debugf("[ScriptLoader] 从文件系统加载脚本: %s", fsPath)
		return string(content), nil
	}

	// 2. 从嵌入的脚本中加载（确保程序可独立运行）
	embeddedPath := "scripts/" + scriptName
	content, err := embeddedScripts.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("脚本未找到: %s (已尝试路径: %s, embedded:%s)", scriptName, fsPath, embeddedPath)
	}

	logrus.Debugf("[ScriptLoader] 从嵌入资源加载脚本: %s", embeddedPath)
	return string(content), nil
}

// GetScriptTemplate 获取脚本模板
func (s *ScriptLoader) GetScriptTemplate(scriptName string) (*template.Template, error) {
	// 检查缓存
	s.cacheMutex.RLock()
	if tmpl, ok := s.templateCache[scriptName]; ok {
		s.cacheMutex.RUnlock()
		return tmpl, nil
	}
	s.cacheMutex.RUnlock()

	// 加载脚本内容
	content, err := s.GetScript(scriptName)
	if err != nil {
		return nil, err
	}

	// 解析为模板
	tmpl, err := template.New(scriptName).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("解析脚本模板失败 [%s]: %v", scriptName, err)
	}

	// 缓存模板
	s.cacheMutex.Lock()
	s.templateCache[scriptName] = tmpl
	s.cacheMutex.Unlock()

	return tmpl, nil
}

// RenderScript 渲染脚本（使用模板参数）
func (s *ScriptLoader) RenderScript(scriptName string, params interface{}) (string, error) {
	tmpl, err := s.GetScriptTemplate(scriptName)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("渲染脚本失败 [%s]: %v", scriptName, err)
	}

	return buf.String(), nil
}

// ListScripts 列出所有可用脚本
func (s *ScriptLoader) ListScripts() ([]string, error) {
	var scripts []string
	seen := make(map[string]bool)

	// 1. 从文件系统加载
	if _, err := os.Stat(s.scriptsDir); err == nil {
		err := filepath.Walk(s.scriptsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 忽略错误，继续遍历
			}
			if !info.IsDir() && (strings.HasSuffix(info.Name(), ".sh") || strings.HasSuffix(info.Name(), ".tmpl")) {
				relPath, _ := filepath.Rel(s.scriptsDir, path)
				scripts = append(scripts, relPath)
				seen[relPath] = true
			}
			return nil
		})
		if err != nil {
			logrus.Warnf("[ScriptLoader] 遍历脚本目录失败: %v", err)
		}
	}

	// 2. 从嵌入资源加载
	err := fs.WalkDir(embeddedScripts, "scripts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".sh") || strings.HasSuffix(d.Name(), ".tmpl")) {
			relPath := strings.TrimPrefix(path, "scripts/")
			if !seen[relPath] {
				scripts = append(scripts, relPath)
			}
		}
		return nil
	})
	if err != nil {
		logrus.Warnf("[ScriptLoader] 遍历嵌入脚本失败: %v", err)
	}

	return scripts, nil
}

// ClearCache 清除模板缓存（用于热更新脚本后生效）
func (s *ScriptLoader) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.templateCache = make(map[string]*template.Template)
	logrus.Info("[ScriptLoader] 模板缓存已清除")
}

// ReloadScript 重新加载指定脚本的模板（从缓存中移除）
func (s *ScriptLoader) ReloadScript(scriptName string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	delete(s.templateCache, scriptName)
	logrus.Infof("[ScriptLoader] 脚本模板已从缓存移除: %s", scriptName)
}

// GetScriptsDir 获取外部脚本目录路径
func (s *ScriptLoader) GetScriptsDir() string {
	return s.scriptsDir
}

// ========================= 脚本生成函数 =========================
// 以下函数基于外部模板文件生成脚本
// 模板文件位于 scripts/templates/ 目录
// 运维人员可直接修改模板文件，无需重新编译程序

// getTemplateForOS 根据操作系统获取对应的模板文件名
func getTemplateForOS(scriptType, osName string) string {
	osMap, ok := templateFiles[scriptType]
	if !ok {
		return ""
	}
	if tmplName, ok := osMap[osName]; ok {
		return tmplName
	}
	return osMap["default"]
}

// GenerateSaltInstallScript 生成 Salt Minion 安装脚本
func (s *ScriptLoader) GenerateSaltInstallScript(params SaltInstallParams) (string, error) {
	// 获取对应操作系统的模板
	templateName := getTemplateForOS("salt-install", params.OS)
	if templateName == "" {
		templateName = "templates/salt-install-generic.sh.tmpl"
	}

	script, err := s.RenderScript(templateName, params)
	if err != nil {
		logrus.Warnf("[ScriptLoader] 加载模板 %s 失败: %v，将使用通用模板", templateName, err)
		// 尝试通用模板
		script, err = s.RenderScript("templates/salt-install-generic.sh.tmpl", params)
		if err != nil {
			return "", fmt.Errorf("无法生成 Salt 安装脚本: %v", err)
		}
	}

	return script, nil
}

// GenerateSaltUninstallScript 生成 Salt Minion 卸载脚本
func (s *ScriptLoader) GenerateSaltUninstallScript(params SaltUninstallParams) (string, error) {
	// 获取对应操作系统的模板
	templateName := getTemplateForOS("salt-uninstall", params.OS)
	if templateName == "" {
		templateName = "templates/salt-uninstall-generic.sh.tmpl"
	}

	script, err := s.RenderScript(templateName, params)
	if err != nil {
		logrus.Warnf("[ScriptLoader] 加载模板 %s 失败: %v，将使用通用模板", templateName, err)
		// 尝试通用模板
		script, err = s.RenderScript("templates/salt-uninstall-generic.sh.tmpl", params)
		if err != nil {
			return "", fmt.Errorf("无法生成 Salt 卸载脚本: %v", err)
		}
	}

	return script, nil
}

// GenerateOSDetectScript 生成操作系统检测脚本
func (s *ScriptLoader) GenerateOSDetectScript() string {
	script, err := s.GetScript("templates/os-detect.sh.tmpl")
	if err != nil {
		logrus.Warnf("[ScriptLoader] 无法加载 OS 检测脚本模板: %v，使用内置脚本", err)
		// 回退到内置脚本（确保基本功能可用）
		return `
if [ -f /etc/os-release ]; then
    . /etc/os-release
    echo "OS:$ID"
    echo "VERSION:$VERSION_ID"
elif [ -f /etc/redhat-release ]; then
    echo "OS:rhel"
    echo "VERSION:$(cat /etc/redhat-release | grep -oE '[0-9]+' | head -1)"
else
    echo "OS:unknown"
    echo "VERSION:unknown"
fi
echo "ARCH:$(uname -m)"
`
	}
	return script
}

// GenerateSSHTestScript 生成 SSH 连接测试脚本
func (s *ScriptLoader) GenerateSSHTestScript(sudoPass string) string {
	params := SSHTestParams{SudoPass: sudoPass}
	script, err := s.RenderScript("templates/ssh-test.sh.tmpl", params)
	if err != nil {
		logrus.Warnf("[ScriptLoader] 无法加载 SSH 测试脚本模板: %v，使用内置脚本", err)
		// 回退到内置脚本
		if sudoPass != "" {
			return fmt.Sprintf(`
export SUDO_ASKPASS=/dev/null
echo '%s' | sudo -S -v 2>/dev/null && echo "SUDO:yes:nopassword" || {
    if echo '%s' | sudo -S true 2>/dev/null; then
        echo "SUDO:yes:password"
    else
        echo "SUDO:no"
    fi
}
hostname
uname -a
`, sudoPass, sudoPass)
		}
		return `hostname && uname -a`
	}
	return script
}

// GenerateCategrafInstallScript 生成 Categraf 安装脚本
func (s *ScriptLoader) GenerateCategrafInstallScript(params map[string]string) (string, error) {
	// 获取对应操作系统的模板
	osType := params["OS"]
	templateName := getTemplateForOS("categraf-install", osType)
	if templateName == "" {
		templateName = "templates/categraf-install-debian.sh.tmpl"
	}

	script, err := s.RenderScript(templateName, params)
	if err != nil {
		logrus.Warnf("[ScriptLoader] 加载模板 %s 失败: %v，将使用通用模板", templateName, err)
		// 尝试 Debian 模板作为通用模板
		script, err = s.RenderScript("templates/categraf-install-debian.sh.tmpl", params)
		if err != nil {
			return "", fmt.Errorf("无法生成 Categraf 安装脚本: %v", err)
		}
	}

	return script, nil
}

// GenerateNodeMetricsDeployScript 生成节点指标采集部署脚本
func (s *ScriptLoader) GenerateNodeMetricsDeployScript(params NodeMetricsDeployParams) (string, error) {
	// 首先加载采集脚本内容
	collectScript, err := s.GetScript("node-metrics/collect-node-metrics.sh")
	if err != nil {
		return "", fmt.Errorf("无法加载节点指标采集脚本: %v", err)
	}
	params.CollectScript = collectScript

	// 渲染部署模板
	script, err := s.RenderScript("templates/node-metrics-deploy.sh.tmpl", params)
	if err != nil {
		return "", fmt.Errorf("无法生成节点指标部署脚本: %v", err)
	}

	return script, nil
}
