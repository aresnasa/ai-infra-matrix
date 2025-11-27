package services

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

//go:embed scripts/*.sh scripts/**/*.sh
var embeddedScripts embed.FS

// ScriptLoader 脚本加载器服务
// 优先从文件系统加载脚本，如果不存在则从嵌入的脚本中加载
type ScriptLoader struct {
	scriptsDir    string
	templateCache map[string]*template.Template
	cacheMutex    sync.RWMutex
}

// ScriptParams 脚本参数基础结构
type ScriptParams map[string]interface{}

// SaltInstallParams Salt Minion 安装参数
type SaltInstallParams struct {
	AppHubURL  string
	MasterHost string
	MinionID   string
	Version    string
	Arch       string
	RpmArch    string
	SudoPrefix string
	OS         string
	OSVersion  string
}

// SaltUninstallParams Salt Minion 卸载参数
type SaltUninstallParams struct {
	SudoPrefix string
	OS         string
}

// SSHTestParams SSH 测试参数
type SSHTestParams struct {
	SudoPass string
}

// OSDetectParams 操作系统检测参数
type OSDetectParams struct{}

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

		logrus.Infof("[ScriptLoader] 初始化，脚本目录: %s", scriptsDir)
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
	// 1. 优先从文件系统加载
	fsPath := filepath.Join(s.scriptsDir, scriptName)
	if content, err := os.ReadFile(fsPath); err == nil {
		logrus.Debugf("[ScriptLoader] 从文件系统加载脚本: %s", fsPath)
		return string(content), nil
	}

	// 2. 从嵌入的脚本中加载
	embeddedPath := "scripts/" + scriptName
	content, err := embeddedScripts.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("脚本未找到: %s (尝试路径: %s, %s)", scriptName, fsPath, embeddedPath)
	}

	logrus.Debugf("[ScriptLoader] 从嵌入资源加载脚本: %s", embeddedPath)
	return string(content), nil
}

// GetScriptTemplate 获取脚本模板
func (s *ScriptLoader) GetScriptTemplate(scriptName string) (*template.Template, error) {
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
		return nil, fmt.Errorf("解析脚本模板失败: %v", err)
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
		return "", fmt.Errorf("渲染脚本失败: %v", err)
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
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".sh") {
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
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sh") {
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

// ClearCache 清除模板缓存
func (s *ScriptLoader) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.templateCache = make(map[string]*template.Template)
}

// ========================= 预定义脚本生成函数 =========================

// GenerateSaltInstallScript 生成 Salt Minion 安装脚本
func (s *ScriptLoader) GenerateSaltInstallScript(params SaltInstallParams) (string, error) {
	// 如果有模板脚本，优先使用模板
	if script, err := s.RenderScript("templates/salt-install.sh.tmpl", params); err == nil {
		return script, nil
	}

	// 回退到内置脚本生成
	return s.generateSaltInstallScriptInline(params)
}

// generateSaltInstallScriptInline 生成内联 Salt 安装脚本（向后兼容）
func (s *ScriptLoader) generateSaltInstallScriptInline(params SaltInstallParams) (string, error) {
	switch params.OS {
	case "ubuntu", "debian":
		return s.generateDebianSaltInstallScript(params), nil
	case "centos", "rhel", "rocky", "almalinux", "fedora":
		return s.generateRhelSaltInstallScript(params), nil
	default:
		return s.generateGenericSaltInstallScript(params), nil
	}
}

func (s *ScriptLoader) generateDebianSaltInstallScript(p SaltInstallParams) string {
	return fmt.Sprintf(`
set -e
echo "=== Starting Salt Minion Installation ==="

APPHUB_SUCCESS=0

# Create temp directory
cd /tmp
rm -rf salt-install && mkdir -p salt-install && cd salt-install

echo "=== Trying to download Salt packages from AppHub ==="
# Try to download salt packages from AppHub
if curl -fsSL --connect-timeout 10 "%s/pkgs/saltstack-deb/salt-common_%s_%s.deb" -o salt-common.deb 2>/dev/null && \
   curl -fsSL --connect-timeout 10 "%s/pkgs/saltstack-deb/salt-minion_%s_%s.deb" -o salt-minion.deb 2>/dev/null; then
    echo "=== Downloaded packages from AppHub ==="
    
    echo "=== Installing dependencies ==="
    %sapt-get update -qq || true
    %sapt-get install -y -qq python3 python3-pip python3-setuptools || true
    
    echo "=== Installing Salt packages from AppHub ==="
    %sdpkg -i salt-common.deb || %sapt-get install -f -y -qq
    %sdpkg -i salt-minion.deb || %sapt-get install -f -y -qq
    APPHUB_SUCCESS=1
else
    echo "=== AppHub download failed, falling back to online installation ==="
fi

# If AppHub failed, use Salt Bootstrap script
if [ "$APPHUB_SUCCESS" -eq 0 ]; then
    echo "=== Using Salt Bootstrap script for online installation ==="
    cd /tmp
    rm -f bootstrap-salt.sh
    
    # Download Salt Bootstrap script
    curl -fsSL -o bootstrap-salt.sh https://bootstrap.saltproject.io || \
    curl -fsSL -o bootstrap-salt.sh https://raw.githubusercontent.com/saltstack/salt-bootstrap/stable/bootstrap-salt.sh || \
    { echo "Failed to download Salt Bootstrap script"; exit 1; }
    
    # Make executable and run
    chmod +x bootstrap-salt.sh
    %sbash bootstrap-salt.sh -x python3 stable || { echo "Bootstrap failed"; exit 1; }
    
    echo "=== Salt Bootstrap installation completed ==="
fi

echo "=== Configuring Salt Minion ==="
%smkdir -p /etc/salt
cat << 'SALTCONF' | %stee /etc/salt/minion
master: %s
id: %s
mine_enabled: true
mine_return_job: true
mine_interval: 60
SALTCONF

echo "=== Starting Salt Minion service ==="
%ssystemctl daemon-reload || true
%ssystemctl enable salt-minion || true
%ssystemctl restart salt-minion || true
sleep 2
%ssystemctl status salt-minion --no-pager || true

echo "=== Cleaning up ==="
cd /tmp && rm -rf salt-install bootstrap-salt.sh

echo "=== Salt Minion Installation Complete ==="
`, p.AppHubURL, p.Version, p.Arch,
		p.AppHubURL, p.Version, p.Arch,
		p.SudoPrefix, p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.MasterHost, p.MinionID,
		p.SudoPrefix, p.SudoPrefix, p.SudoPrefix, p.SudoPrefix)
}

func (s *ScriptLoader) generateRhelSaltInstallScript(p SaltInstallParams) string {
	return fmt.Sprintf(`
set -e
echo "=== Starting Salt Minion Installation ==="

APPHUB_SUCCESS=0

# Create temp directory
cd /tmp
rm -rf salt-install && mkdir -p salt-install && cd salt-install

echo "=== Trying to download Salt packages from AppHub ==="
# Try to download salt packages from AppHub
if curl -fsSL --connect-timeout 10 "%s/pkgs/saltstack-rpm/salt-%s-0.%s.rpm" -o salt.rpm 2>/dev/null && \
   curl -fsSL --connect-timeout 10 "%s/pkgs/saltstack-rpm/salt-minion-%s-0.%s.rpm" -o salt-minion.rpm 2>/dev/null; then
    echo "=== Downloaded packages from AppHub ==="
    
    echo "=== Installing dependencies ==="
    %syum install -y python3 python3-pip || %sdnf install -y python3 python3-pip || true
    
    echo "=== Installing Salt packages from AppHub ==="
    %srpm -Uvh --replacepkgs salt.rpm || %syum localinstall -y salt.rpm || true
    %srpm -Uvh --replacepkgs salt-minion.rpm || %syum localinstall -y salt-minion.rpm || true
    APPHUB_SUCCESS=1
else
    echo "=== AppHub download failed, falling back to online installation ==="
fi

# If AppHub failed, use Salt Bootstrap script
if [ "$APPHUB_SUCCESS" -eq 0 ]; then
    echo "=== Using Salt Bootstrap script for online installation ==="
    cd /tmp
    rm -f bootstrap-salt.sh
    
    # Download Salt Bootstrap script
    curl -fsSL -o bootstrap-salt.sh https://bootstrap.saltproject.io || \
    curl -fsSL -o bootstrap-salt.sh https://raw.githubusercontent.com/saltstack/salt-bootstrap/stable/bootstrap-salt.sh || \
    { echo "Failed to download Salt Bootstrap script"; exit 1; }
    
    # Make executable and run
    chmod +x bootstrap-salt.sh
    %sbash bootstrap-salt.sh -x python3 stable || { echo "Bootstrap failed"; exit 1; }
    
    echo "=== Salt Bootstrap installation completed ==="
fi

echo "=== Configuring Salt Minion ==="
%smkdir -p /etc/salt
cat << 'SALTCONF' | %stee /etc/salt/minion
master: %s
id: %s
mine_enabled: true
mine_return_job: true
mine_interval: 60
SALTCONF

echo "=== Starting Salt Minion service ==="
%ssystemctl daemon-reload || true
%ssystemctl enable salt-minion || true
%ssystemctl restart salt-minion || true
sleep 2
%ssystemctl status salt-minion --no-pager || true

echo "=== Cleaning up ==="
cd /tmp && rm -rf salt-install bootstrap-salt.sh

echo "=== Salt Minion Installation Complete ==="
`, p.AppHubURL, p.Version, p.RpmArch,
		p.AppHubURL, p.Version, p.RpmArch,
		p.SudoPrefix, p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.MasterHost, p.MinionID,
		p.SudoPrefix, p.SudoPrefix, p.SudoPrefix, p.SudoPrefix)
}

func (s *ScriptLoader) generateGenericSaltInstallScript(p SaltInstallParams) string {
	return fmt.Sprintf(`
set -e
echo "=== Starting Salt Minion Installation (Bootstrap) ==="
echo "=== Detected OS: %s %s ==="

cd /tmp
rm -f bootstrap-salt.sh

echo "=== Downloading Salt Bootstrap script ==="
curl -fsSL -o bootstrap-salt.sh https://bootstrap.saltproject.io || \
curl -fsSL -o bootstrap-salt.sh https://raw.githubusercontent.com/saltstack/salt-bootstrap/stable/bootstrap-salt.sh || \
{ echo "Failed to download Salt Bootstrap script"; exit 1; }

chmod +x bootstrap-salt.sh
%sbash bootstrap-salt.sh -x python3 stable || { echo "Bootstrap failed"; exit 1; }

echo "=== Configuring Salt Minion ==="
%smkdir -p /etc/salt
cat << 'SALTCONF' | %stee /etc/salt/minion
master: %s
id: %s
mine_enabled: true
mine_return_job: true
mine_interval: 60
SALTCONF

echo "=== Starting Salt Minion service ==="
%ssystemctl daemon-reload || true
%ssystemctl enable salt-minion || true
%ssystemctl restart salt-minion || true
sleep 2
%ssystemctl status salt-minion --no-pager || true

echo "=== Cleaning up ==="
rm -f /tmp/bootstrap-salt.sh

echo "=== Salt Minion Installation Complete ==="
`, p.OS, p.OSVersion,
		p.SudoPrefix,
		p.SudoPrefix, p.SudoPrefix,
		p.MasterHost, p.MinionID,
		p.SudoPrefix, p.SudoPrefix, p.SudoPrefix, p.SudoPrefix)
}

// GenerateSaltUninstallScript 生成 Salt Minion 卸载脚本
func (s *ScriptLoader) GenerateSaltUninstallScript(params SaltUninstallParams) (string, error) {
	switch params.OS {
	case "ubuntu", "debian":
		return fmt.Sprintf(`
set -e
echo "=== Uninstalling Salt Minion ==="
%ssystemctl stop salt-minion || true
%ssystemctl disable salt-minion || true
%sapt-get remove -y salt-minion salt-common || true
%sapt-get autoremove -y || true
%srm -rf /etc/salt /var/cache/salt /var/log/salt /var/run/salt
echo "=== Salt Minion Uninstall Complete ==="
`, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix), nil
	case "centos", "rhel", "rocky", "almalinux", "fedora":
		return fmt.Sprintf(`
set -e
echo "=== Uninstalling Salt Minion ==="
%ssystemctl stop salt-minion || true
%ssystemctl disable salt-minion || true
%syum remove -y salt-minion salt || %sdnf remove -y salt-minion salt || true
%srm -rf /etc/salt /var/cache/salt /var/log/salt /var/run/salt
echo "=== Salt Minion Uninstall Complete ==="
`, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix), nil
	default:
		return fmt.Sprintf(`
set -e
echo "=== Uninstalling Salt Minion ==="
%ssystemctl stop salt-minion || true
%ssystemctl disable salt-minion || true
%srm -rf /etc/salt /var/cache/salt /var/log/salt /var/run/salt
echo "=== Salt Minion Uninstall Complete (manual package removal may be needed) ==="
`, params.SudoPrefix, params.SudoPrefix, params.SudoPrefix), nil
	}
}

// GenerateOSDetectScript 生成操作系统检测脚本
func (s *ScriptLoader) GenerateOSDetectScript() string {
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

// GenerateSSHTestScript 生成 SSH 连接测试脚本
func (s *ScriptLoader) GenerateSSHTestScript(sudoPass string) string {
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
