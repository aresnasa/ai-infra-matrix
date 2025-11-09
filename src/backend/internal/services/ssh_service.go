package services

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHService 处理SSH连接和并发部署
type SSHService struct {
	config *SSHConfig
}

// SSHConfig SSH服务配置
type SSHConfig struct {
	DefaultUser    string
	DefaultKeyPath string
	ConnectTimeout time.Duration
	CommandTimeout time.Duration
	MaxConcurrency int
}

// AppHubConfig AppHub包仓库配置
type AppHubConfig struct {
	BaseURL  string // AppHub的基础URL，如 http://192.168.0.200:8090
	Username string // 用户名（可选）
	Password string // 密码（可选）
}

// PackageInstallationConfig 包安装配置
type PackageInstallationConfig struct {
	AppHubConfig      AppHubConfig
	SaltMasterHost    string
	SaltMasterPort    int
	MinionID          string
	SlurmRole         string // controller|compute
	EnableSaltMinion  bool
	EnableSlurmClient bool
}

// InstallationStep 安装步骤
type InstallationStep struct {
	Name        string
	Description string
	Commands    []string
	Critical    bool // 是否为关键步骤，失败时停止安装
}

// InstallationResult 完整安装结果
type InstallationResult struct {
	Host     string
	Success  bool
	Steps    []StepResult
	Duration time.Duration
	Error    string
}

// StepResult 步骤执行结果
type StepResult struct {
	Name      string
	Success   bool
	Output    string
	Error     string
	Duration  time.Duration
	Timestamp time.Time
}

// SSHConnection SSH连接信息
type SSHConnection struct {
	Host       string
	Port       int
	User       string
	KeyPath    string
	PrivateKey string // 新增：内联私钥内容
	Password   string
}

// DeploymentResult 部署结果
type DeploymentResult struct {
	Host     string
	Success  bool
	Output   string
	Error    string
	Duration time.Duration
}

// ScriptSpec 描述一段可在远程主机上执行的脚本
// 支持两种来源：
//   - Content: 内联脚本内容
//   - URL: 在远程通过 curl/wget 下载的脚本地址
//
// 二者至少提供其一；同时提供时优先使用 Content
type ScriptSpec struct {
	Name        string            // 脚本名称，仅用于日志
	Content     string            // 内联脚本内容
	URL         string            // 远程脚本URL（http/https）
	Args        []string          // 传递给脚本的参数
	Env         map[string]string // 环境变量
	Interpreter string            // 解释器，例如 "/bin/bash"、"/bin/sh"，为空则直接执行脚本
	UseSudo     bool              // 是否使用sudo -E -n执行
	WorkDir     string            // 执行前切换的工作目录
	Timeout     time.Duration     // 单次执行超时时间；为空则使用默认
}

// SaltStackDeploymentConfig SaltStack部署配置
type SaltStackDeploymentConfig struct {
	MasterHost string
	MasterPort int
	MinionID   string
	AutoAccept bool
	// AppHubURL: 用于离线/内网安装salt-minion的包仓库，例如 http://<external_host>:<apphub_port>
	AppHubURL string
}

// SetupSimpleDebRepo 在远程主机上安装nginx与dpkg-dev并创建简单的deb仓库目录结构
func (s *SSHService) SetupSimpleDebRepo(host string, port int, user, password, basePath string, enableIndex bool) error {
	if port == 0 {
		port = 22
	}
	// 安装必要组件并创建目录
	cmds := []string{
		`/bin/sh -lc 'if command -v apt-get >/dev/null 2>&1; then export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get install -y nginx dpkg-dev && systemctl enable nginx || true && systemctl restart nginx || true; elif command -v yum >/dev/null 2>&1; then yum install -y nginx createrepo; systemctl enable nginx || true; systemctl restart nginx || true; elif command -v dnf >/dev/null 2>&1; then dnf install -y nginx createrepo; systemctl enable nginx || true; systemctl restart nginx || true; else echo "Unsupported distro"; exit 1; fi'`,
		"/bin/sh -lc 'mkdir -p " + basePath + "'",
	}
	for _, cmd := range cmds {
		if _, err := s.ExecuteCommand(host, port, user, password, cmd); err != nil {
			return err
		}
	}
	if enableIndex {
		// 生成Packages索引（仅Deb系）
		idx := "/bin/sh -lc 'if command -v apt-ftparchive >/dev/null 2>&1; then (cd " + basePath + " && apt-ftparchive packages . > Packages && gzip -f Packages); fi'"
		if _, err := s.ExecuteCommand(host, port, user, password, idx); err != nil {
			return err
		}
	}
	return nil
}

// ConfigureAptRepo 写入sources.list.d源并更新缓存
func (s *SSHService) ConfigureAptRepo(host string, port int, user, password, repoURL string) error {
	if port == 0 {
		port = 22
	}
	// 仅对Deb系系统执行
	cmd := `/bin/sh -lc 'if command -v apt-get >/dev/null 2>&1; then echo "deb [trusted=yes] ` + repoURL + ` ./" >/etc/apt/sources.list.d/ai-infra-slurm.list && apt-get update; else echo "Non Debian-based OS, skipping"; fi'`
	_, err := s.ExecuteCommand(host, port, user, password, cmd)
	return err
}

// InstallSlurm 安装并配置slurm组件，role: controller|node
func (s *SSHService) InstallSlurm(host string, port int, user, password, role string) (string, error) {
	if port == 0 {
		port = 22
	}
	pkgCmd := "/bin/sh -lc 'if command -v apt-get >/dev/null 2>&1; then export DEBIAN_FRONTEND=noninteractive; apt-get install -y slurmctld slurmd slurm-client; elif command -v yum >/dev/null 2>&1; then yum install -y slurm slurmctld slurmd; elif command -v dnf >/dev/null 2>&1; then dnf install -y slurm slurmctld slurmd; else echo Unsupported; exit 1; fi'"
	out, err := s.ExecuteCommand(host, port, user, password, pkgCmd)
	if err != nil {
		return out, err
	}

	// 根据角色启用服务
	var enable string
	if role == "controller" {
		enable = "/bin/sh -lc 'systemctl enable slurmctld || true; systemctl restart slurmctld || true'"
	} else {
		enable = "/bin/sh -lc 'systemctl enable slurmd || true; systemctl restart slurmd || true'"
	}
	out2, err := s.ExecuteCommand(host, port, user, password, enable)
	return out + "\n" + out2, err
}

// NewSSHService 创建新的SSH服务
func NewSSHService() *SSHService {
	return &SSHService{
		config: &SSHConfig{
			DefaultUser:    "root",
			DefaultKeyPath: "/root/.ssh/id_rsa",
			ConnectTimeout: 10 * time.Second,
			CommandTimeout: 30 * time.Second,
			MaxConcurrency: 10,
		},
	}
}

// InstallPackagesOnHosts 在多个主机上并发安装SaltStack Minion和SLURM客户端
func (s *SSHService) InstallPackagesOnHosts(ctx context.Context, connections []SSHConnection, config PackageInstallationConfig) ([]InstallationResult, error) {
	results := make([]InstallationResult, len(connections))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.MaxConcurrency)

	for i, conn := range connections {
		wg.Add(1)
		go func(index int, connection SSHConnection) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			startTime := time.Now()
			result := s.installPackagesOnSingleHost(ctx, connection, config)
			result.Duration = time.Since(startTime)
			results[index] = result
		}(i, conn)
	}

	wg.Wait()
	return results, nil
}

// installPackagesOnSingleHost 在单个主机上安装包
func (s *SSHService) installPackagesOnSingleHost(ctx context.Context, conn SSHConnection, config PackageInstallationConfig) InstallationResult {
	result := InstallationResult{
		Host:    conn.Host,
		Success: false,
		Steps:   []StepResult{},
	}

	// 建立SSH连接
	client, err := s.connectSSH(conn)
	if err != nil {
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer client.Close()

	// 步骤1: 上传salt-minion安装脚本
	if config.EnableSaltMinion {
		uploadStep := s.uploadSaltMinionScript(conn, config)
		result.Steps = append(result.Steps, uploadStep)

		if !uploadStep.Success {
			result.Error = "上传salt-minion安装脚本失败"
			return result
		}
	}

	// 步骤2: 执行salt-minion安装脚本
	if config.EnableSaltMinion {
		executeStep := s.executeSaltMinionScript(client, conn, config)
		result.Steps = append(result.Steps, executeStep)

		if !executeStep.Success {
			result.Error = "执行salt-minion安装脚本失败"
			return result
		}
	}

	// 步骤3: 配置salt-minion
	if config.EnableSaltMinion {
		configStep := s.configureSaltMinion(client, config, conn.Host)
		result.Steps = append(result.Steps, configStep)

		if !configStep.Success {
			result.Error = "配置salt-minion失败"
			return result
		}
	}

	// 步骤4: 启动salt-minion服务
	if config.EnableSaltMinion {
		startStep := s.startSaltMinion(client)
		result.Steps = append(result.Steps, startStep)

		if !startStep.Success {
			result.Error = "启动salt-minion服务失败"
			return result
		}
	}

	// 如果启用SLURM客户端，生成并执行SLURM相关步骤
	if config.EnableSlurmClient {
		slurmSteps := s.generateSlurmInstallationSteps(config, conn.Host)
		for _, step := range slurmSteps {
			stepResult := s.executeInstallationStep(client, step)
			result.Steps = append(result.Steps, stepResult)

			if !stepResult.Success && step.Critical {
				result.Error = fmt.Sprintf("SLURM安装失败: %s", step.Name)
				return result
			}
		}
	}

	result.Success = true
	return result
}

// generateInstallationSteps 生成安装步骤
func (s *SSHService) generateInstallationSteps(config PackageInstallationConfig, hostname string) []InstallationStep {
	var steps []InstallationStep

	// 1. 系统检查和初始化
	steps = append(steps, InstallationStep{
		Name:        "system_check",
		Description: "检查系统信息和网络连接",
		Critical:    true,
		Commands: []string{
			"uname -a",
			"cat /etc/os-release",
			"free -h",
			"df -h",
			"ping -c 3 " + s.extractHostFromURL(config.AppHubConfig.BaseURL),
		},
	})

	// 2. 配置APT源（仅Debian/Ubuntu系统）
	steps = append(steps, InstallationStep{
		Name:        "configure_apt_source",
		Description: "配置AppHub APT源",
		Critical:    false,
		Commands: []string{
			s.getConfigureAptSourceCommand(config.AppHubConfig.BaseURL),
		},
	})

	// 3. 更新包索引
	steps = append(steps, InstallationStep{
		Name:        "update_packages",
		Description: "更新系统包索引",
		Critical:    true,
		Commands: []string{
			s.getUpdatePackagesCommand(),
		},
	})

	// 4. 安装基础工具
	steps = append(steps, InstallationStep{
		Name:        "install_basic_tools",
		Description: "安装基础工具和依赖",
		Critical:    true,
		Commands: []string{
			s.getInstallBasicToolsCommand(),
		},
	})

	// 5. 安装SaltStack Minion（如果启用）
	if config.EnableSaltMinion {
		steps = append(steps, InstallationStep{
			Name:        "install_saltstack_minion",
			Description: "安装和配置SaltStack Minion",
			Critical:    true,
			Commands: []string{
				s.getInstallSaltMinionCommand(),
				s.getConfigureSaltMinionCommand(config.SaltMasterHost, config.SaltMasterPort, s.getMinionID(config.MinionID, hostname)),
				"systemctl enable salt-minion 2>/dev/null || true",
				"systemctl daemon-reload 2>/dev/null || true",
				"systemctl start salt-minion 2>/dev/null || service salt-minion start 2>/dev/null || salt-minion -d || true",
			},
		})
	}

	// 6. 安装SLURM客户端（如果启用）
	if config.EnableSlurmClient {
		steps = append(steps, InstallationStep{
			Name:        "install_slurm_client",
			Description: "安装SLURM客户端组件",
			Critical:    false,
			Commands: []string{
				s.getInstallSlurmClientCommand(config.AppHubConfig.BaseURL),
			},
		})

		// 7. 配置SLURM节点
		steps = append(steps, InstallationStep{
			Name:        "configure_slurm_node",
			Description: "配置SLURM节点",
			Critical:    false,
			Commands: []string{
				s.getConfigureSlurmNodeCommand(config.SlurmRole, hostname),
			},
		})
	}

	// 8. 最终验证
	steps = append(steps, InstallationStep{
		Name:        "final_verification",
		Description: "验证安装结果",
		Critical:    false,
		Commands:    s.getVerificationCommands(config.EnableSaltMinion, config.EnableSlurmClient),
	})

	return steps
}

// executeInstallationStep 执行安装步骤
func (s *SSHService) executeInstallationStep(client *ssh.Client, step InstallationStep) StepResult {
	result := StepResult{
		Name:      step.Name,
		Success:   true,
		Timestamp: time.Now(),
	}

	start := time.Now()
	var output strings.Builder

	for i, command := range step.Commands {
		fmt.Fprintf(&output, "\n--- 执行命令 %d/%d ---\n", i+1, len(step.Commands))
		fmt.Fprintf(&output, "命令: %s\n", command)

		cmdOutput, err := s.executeCommand(client, command)
		fmt.Fprintf(&output, "%s", cmdOutput)

		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("命令执行失败: %v", err)
			fmt.Fprintf(&output, "错误: %s\n", err.Error())
			break
		}
	}

	result.Output = output.String()
	result.Duration = time.Since(start)

	return result
}

// 以下是各种命令生成函数

// extractHostFromURL 从URL中提取主机名
func (s *SSHService) extractHostFromURL(url string) string {
	// 简单的URL解析，提取主机部分
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}

	if colonIndex := strings.Index(url, ":"); colonIndex != -1 {
		url = url[:colonIndex]
	}

	if slashIndex := strings.Index(url, "/"); slashIndex != -1 {
		url = url[:slashIndex]
	}

	return url
}

// getConfigureAptSourceCommand 获取配置APT源的命令
func (s *SSHService) getConfigureAptSourceCommand(appHubURL string) string {
	return fmt.Sprintf(`
if command -v apt-get >/dev/null 2>&1; then
    echo "deb [trusted=yes] %s/pkgs/slurm-deb ./" > /etc/apt/sources.list.d/ai-infra-matrix.list
    echo "配置了AI Infrastructure Matrix APT源"
else
    echo "非Debian/Ubuntu系统，跳过APT源配置"
fi
`, appHubURL)
}

// getUpdatePackagesCommand 获取更新包索引的命令
func (s *SSHService) getUpdatePackagesCommand() string {
	return `
if command -v apt-get >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
	if ! apt-get update -y; then
		echo "[Update] apt-get update failed, attempting to repair Salt repositories..."
		apt-get install -y --no-install-recommends curl wget gnupg2 ca-certificates lsb-release || true
		mkdir -p /usr/share/keyrings
		if [ ! -f /usr/share/keyrings/salt-archive-keyring.gpg ]; then
			curl -fsSL https://packages.broadcom.com/artifactory/api/security/keypair/SaltProjectKey/public -o /usr/share/keyrings/salt-archive-keyring.gpg || true
		fi
		echo "deb [signed-by=/usr/share/keyrings/salt-archive-keyring.gpg] https://packages.broadcom.com/artifactory/saltproject-deb/ stable main" > /etc/apt/sources.list.d/saltproject.list
		apt-get update -y \
			|| apt-get update -o Acquire::AllowInsecureRepositories=true -o Acquire::AllowDowngradeToInsecureRepositories=true -y \
			|| true
	fi
elif command -v yum >/dev/null 2>&1; then
    yum makecache -y
elif command -v dnf >/dev/null 2>&1; then
    dnf makecache -y
elif command -v zypper >/dev/null 2>&1; then
    zypper refresh
else
    echo "未识别的包管理器"
    exit 1
fi
`
}

// getInstallBasicToolsCommand 获取安装基础工具的命令
func (s *SSHService) getInstallBasicToolsCommand() string {
	return `
if command -v apt-get >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get install -y curl wget gnupg2 ca-certificates lsb-release systemd
elif command -v yum >/dev/null 2>&1; then
    yum install -y curl wget gnupg2 ca-certificates redhat-lsb-core systemd
elif command -v dnf >/dev/null 2>&1; then
    dnf install -y curl wget gnupg2 ca-certificates redhat-lsb systemd
elif command -v zypper >/dev/null 2>&1; then
    zypper install -y curl wget gpg2 ca-certificates lsb-release systemd
else
    echo "未识别的包管理器"
    exit 1
fi
`
}

// getInstallSaltMinionCommand 获取安装SaltStack Minion的命令
func (s *SSHService) getInstallSaltMinionCommand() string {
	return `
set -e
if command -v apt-get >/dev/null 2>&1; then
	export DEBIAN_FRONTEND=noninteractive
	apt-get update -y || true
	if ! apt-get install -y salt-minion; then
		echo "[Salt] apt 安装失败，尝试添加Broadcom Salt仓库 (keyring)..."
		apt-get install -y curl gnupg2 ca-certificates lsb-release || true
		mkdir -p /usr/share/keyrings
		curl -fsSL https://packages.broadcom.com/artifactory/api/security/keypair/SaltProjectKey/public -o /usr/share/keyrings/salt-archive-keyring.gpg || true
		echo "deb [signed-by=/usr/share/keyrings/salt-archive-keyring.gpg] https://packages.broadcom.com/artifactory/saltproject-deb/ stable main" > /etc/apt/sources.list.d/saltproject.list
		apt-get update -y || true
		if ! apt-get install -y salt-minion; then
			echo "[Salt] Broadcom仓库安装失败，尝试使用官方bootstrap脚本..."
			curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		fi
	fi
elif command -v yum >/dev/null 2>&1; then
	if ! yum install -y salt-minion; then
		echo "[Salt] yum 安装失败，尝试添加Broadcom Salt仓库..."
		yum install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.el8.noarch.rpm || true
		yum clean all || true
		yum makecache -y || true
		if ! yum install -y salt-minion; then
			echo "[Salt] Broadcom仓库安装失败，尝试bootstrap脚本"
			curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		fi
	fi
elif command -v dnf >/dev/null 2>&1; then
	if ! dnf install -y salt-minion; then
		echo "[Salt] dnf 安装失败，尝试添加Broadcom Salt仓库..."
		dnf install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.fc36.noarch.rpm || true
		dnf makecache -y || true
		if ! dnf install -y salt-minion; then
			echo "[Salt] Broadcom仓库安装失败，尝试bootstrap脚本"
			curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		fi
	fi
else
	echo "暂不支持的系统类型，尝试通用安装方法"
	curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
		|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
		|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
	sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
fi
`
}

// getConfigureSaltMinionCommand 获取配置SaltStack Minion的命令
func (s *SSHService) getConfigureSaltMinionCommand(masterHost string, masterPort int, minionID string) string {
	if masterPort == 0 {
		masterPort = 4506
	}

	return fmt.Sprintf(`
mkdir -p /etc/salt /var/log/salt
touch /var/log/salt/minion || true
chmod 644 /var/log/salt/minion || true
cat > /etc/salt/minion << 'EOF'
master: %s
master_port: %d
id: %s
log_level: info
log_file: /var/log/salt/minion

# 网络配置
master_alive_interval: 30
master_tries: 3
ping_interval: 0

# 安全配置
open_mode: False
auto_accept_grains: False

# 性能配置
multiprocessing: True
process_count_max: 4
EOF

# 权限设置
chown -R root:root /etc/salt /var/log/salt
chmod 644 /etc/salt/minion || true
`, masterHost, masterPort, minionID)
}

// getInstallSlurmClientCommand 获取安装SLURM客户端的命令
func (s *SSHService) getInstallSlurmClientCommand(appHubURL string) string {
	return fmt.Sprintf(`
set -e
APPHUB_URL="%s"

echo "=== 开始安装 SLURM 客户端（从 AppHub） ==="

# 确保 wget 可用
if ! command -v wget >/dev/null 2>&1; then
	echo "wget 未安装，尝试安装..."
	if command -v apt-get >/dev/null 2>&1; then
		apt-get update || true
		apt-get install -y wget || true
	elif command -v yum >/dev/null 2>&1; then
		yum install -y wget || true
	elif command -v dnf >/dev/null 2>&1; then
		dnf install -y wget || true
	fi
fi

# 优先使用 AppHub 提供的统一安装脚本（支持所有架构和发行版）
echo ">>> 从 AppHub 下载 SLURM 安装脚本..."
cd /tmp

if wget --timeout=30 --tries=3 -O install-slurm.sh "${APPHUB_URL}/packages/install-slurm.sh" 2>/dev/null; then
	echo "✓ 安装脚本下载成功"
	chmod +x install-slurm.sh
	
	# 执行安装脚本（会自动检测架构并下载对应的二进制文件）
	if APPHUB_URL="${APPHUB_URL}" ./install-slurm.sh; then
		echo "✓ SLURM 客户端安装成功（AppHub 二进制）"
		rm -f install-slurm.sh
		exit 0
	else
		echo "⚠️  AppHub 安装脚本执行失败，尝试备选方案..."
		rm -f install-slurm.sh
	fi
else
	echo "⚠️  无法从 AppHub 下载安装脚本，尝试备选方案..."
fi

# 备选方案1: 尝试通过系统包管理器安装
if command -v apt-get >/dev/null 2>&1; then
	export DEBIAN_FRONTEND=noninteractive
	
	echo "[备选] 尝试通过APT安装 SLURM 客户端..."
	if apt-get update && apt-get install -y slurm-smd-client slurm-smd-slurmd 2>/dev/null; then
		echo "✓ SLURM 客户端安装成功（APT）"
		exit 0
	fi
	
	# 备选方案2: 从 AppHub 下载 DEB 包
	echo "[备选] 尝试从 AppHub 下载 DEB 包..."
	ARCH=$(dpkg --print-architecture 2>/dev/null || echo amd64)
	CLIENT_PKG="slurm-smd-client_25.05.3-1_${ARCH}.deb"
	SLURMD_PKG="slurm-smd-slurmd_25.05.3-1_${ARCH}.deb"
	
	if wget -q --timeout=30 --tries=3 "${APPHUB_URL}/pkgs/slurm-deb/${CLIENT_PKG}" && \
	   wget -q --timeout=30 --tries=3 "${APPHUB_URL}/pkgs/slurm-deb/${SLURMD_PKG}"; then
		dpkg -i "${CLIENT_PKG}" "${SLURMD_PKG}" 2>/dev/null || true
		apt-get install -f -y 2>/dev/null || true
		rm -f "${CLIENT_PKG}" "${SLURMD_PKG}" 2>/dev/null || true
		
		if command -v slurmd >/dev/null 2>&1; then
			echo "✓ SLURM 客户端安装成功（AppHub DEB）"
			exit 0
		fi
	fi
	
	echo "❌ SLURM 客户端安装失败（所有 Debian 方案均失败）"
	exit 1
	
elif command -v yum >/dev/null 2>&1; then
	# 备选方案: 尝试通过YUM安装
	echo "[备选] 尝试通过YUM安装 SLURM 客户端..."
	if yum install -y slurm slurm-slurmd 2>/dev/null; then
		echo "✓ SLURM 客户端安装成功（YUM）"
		exit 0
	fi
	
	# 备选方案2: 从 AppHub 下载 RPM 包（如果可用）
	echo "[备选] 尝试从 AppHub 下载 RPM 包..."
	ARCH=$(uname -m)
	RPM_ARCH=$(echo $ARCH | sed 's/aarch64/aarch64/;s/x86_64/x86_64/')
	SLURM_RPM="slurm-25.05.3-1.el9.${RPM_ARCH}.rpm"
	
	if wget -q --timeout=30 --tries=3 "${APPHUB_URL}/pkgs/slurm-rpm/${SLURM_RPM}"; then
		yum install -y "./${SLURM_RPM}" 2>/dev/null || true
		rm -f "${SLURM_RPM}" 2>/dev/null || true
		
		if command -v slurmd >/dev/null 2>&1; then
			echo "✓ SLURM 客户端安装成功（AppHub RPM）"
			exit 0
		fi
	fi
	
	echo "❌ SLURM 客户端安装失败（所有 RHEL/CentOS 方案均失败）"
	exit 1
	
elif command -v dnf >/dev/null 2>&1; then
	# 备选方案: 尝试通过DNF安装
	echo "[备选] 尝试通过DNF安装 SLURM 客户端..."
	if dnf install -y slurm slurm-slurmd 2>/dev/null; then
		echo "✓ SLURM 客户端安装成功（DNF）"
		exit 0
	fi
	
	# 备选方案2: 从 AppHub 下载 RPM 包（如果可用）
	echo "[备选] 尝试从 AppHub 下载 RPM 包..."
	ARCH=$(uname -m)
	RPM_ARCH=$(echo $ARCH | sed 's/aarch64/aarch64/;s/x86_64/x86_64/')
	SLURM_RPM="slurm-25.05.3-1.el9.${RPM_ARCH}.rpm"
	
	if wget -q --timeout=30 --tries=3 "${APPHUB_URL}/pkgs/slurm-rpm/${SLURM_RPM}"; then
		dnf install -y "./${SLURM_RPM}" 2>/dev/null || true
		rm -f "${SLURM_RPM}" 2>/dev/null || true
		
		if command -v slurmd >/dev/null 2>&1; then
			echo "✓ SLURM 客户端安装成功（AppHub RPM）"
			exit 0
		fi
	fi
	
	echo "❌ SLURM 客户端安装失败（所有 Fedora/Rocky 方案均失败）"
	exit 1
else
	echo "❌ 不支持的包管理器"
	exit 1
fi
`, appHubURL)
}

// getConfigureSlurmNodeCommand 获取配置SLURM节点的命令
func (s *SSHService) getConfigureSlurmNodeCommand(role, hostname string) string {
	return fmt.Sprintf(`
# 创建SLURM配置目录
mkdir -p /etc/slurm /var/log/slurm /var/spool/slurm

# 创建基础的slurm.conf配置
cat > /etc/slurm/slurm.conf << 'EOF'
# SLURM配置文件
ClusterName=ai-infra-cluster
ControlMachine=slurm-controller
ControlAddr=slurm-controller

# 认证和安全
AuthType=auth/munge
CryptoType=crypto/munge

# 调度器配置
SchedulerType=sched/backfill
SelectType=select/cons_res
SelectTypeParameters=CR_Core

# 日志配置
SlurmdLogFile=/var/log/slurm/slurmd.log
SlurmctldLogFile=/var/log/slurm/slurmctld.log
SlurmdSpoolDir=/var/spool/slurm

# 节点配置 (不设置State，让SLURM自动管理)
NodeName=%s CPUs=2 Sockets=1 CoresPerSocket=2 ThreadsPerCore=1 RealMemory=1000
PartitionName=compute Nodes=%s Default=YES MaxTime=INFINITE State=UP
EOF

# 设置权限
chown -R slurm:slurm /var/log/slurm /var/spool/slurm /etc/slurm 2>/dev/null || true
chmod 644 /etc/slurm/slurm.conf

# 根据角色启用相应服务
if [ "%s" = "controller" ]; then
    systemctl enable slurmctld 2>/dev/null || true
    # 重载配置后激活节点
    sleep 2
    scontrol reconfigure 2>/dev/null || true
    sleep 1
    scontrol update NodeName=%s State=RESUME 2>/dev/null || echo "节点激活命令已执行"
else
    systemctl enable slurmd 2>/dev/null || true
fi
`, hostname, hostname, role, hostname)
}

// getMinionID 获取Minion ID
func (s *SSHService) getMinionID(configuredID, hostname string) string {
	if configuredID != "" {
		return configuredID
	}
	return hostname
}

// getVerificationCommands 获取验证命令
func (s *SSHService) getVerificationCommands(saltEnabled, slurmEnabled bool) []string {
	commands := []string{
		"echo '=== 系统状态 ==='",
		"uptime",
		"free -h",
	}

	if saltEnabled {
		commands = append(commands,
			"echo '=== SaltStack Minion 状态 ==='",
			"systemctl status salt-minion --no-pager -l",
			"salt-minion --version 2>/dev/null || echo 'salt-minion版本检查失败'",
		)
	}

	if slurmEnabled {
		commands = append(commands,
			"echo '=== SLURM 状态 ==='",
			"systemctl status slurmd --no-pager -l 2>/dev/null || echo 'slurmd未运行'",
			"sinfo --version 2>/dev/null || echo 'SLURM客户端工具未安装'",
		)
	}

	return commands
}

// DeploySaltMinion 并发部署SaltStack Minion到多个节点
func (s *SSHService) DeploySaltMinion(ctx context.Context, connections []SSHConnection, config SaltStackDeploymentConfig) ([]DeploymentResult, error) {
	results := make([]DeploymentResult, len(connections))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.MaxConcurrency)

	for i, conn := range connections {
		wg.Add(1)
		go func(index int, connection SSHConnection) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			startTime := time.Now()
			result := s.deploySingleMinion(ctx, connection, config)
			result.Duration = time.Since(startTime)
			results[index] = result
		}(i, conn)
	}

	wg.Wait()

	// 等待所有成功部署的 Minion 被 Master 接受
	if config.AutoAccept {
		successfulHosts := []string{}
		for i, result := range results {
			if result.Success {
				successfulHosts = append(successfulHosts, connections[i].Host)
			}
		}

		if len(successfulHosts) > 0 {
			// 等待 Minion 密钥被接受（最多等待5分钟）
			waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			acceptErrors := s.waitForMinionsAccepted(waitCtx, successfulHosts, config.MasterHost)

			// 更新结果中的错误信息
			for i, result := range results {
				if result.Success {
					host := connections[i].Host
					if err, exists := acceptErrors[host]; exists && err != nil {
						results[i].Success = false
						results[i].Error = fmt.Sprintf("Minion部署成功但未能加入集群: %v", err)
					}
				}
			}
		}
	}

	return results, nil
}

// deploySingleMinion 部署单个SaltStack Minion
func (s *SSHService) deploySingleMinion(ctx context.Context, conn SSHConnection, config SaltStackDeploymentConfig) DeploymentResult {
	result := DeploymentResult{
		Host:    conn.Host,
		Success: false,
	}

	// 建立SSH连接
	client, err := s.connectSSH(conn)
	if err != nil {
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer client.Close()

	// 执行部署步骤
	output, err := s.executeDeploymentSteps(client, config)
	if err != nil {
		result.Error = fmt.Sprintf("部署失败: %v", err)
		result.Output = output
		return result
	}

	result.Success = true
	result.Output = output
	return result
}

// connectSSH 建立SSH连接
func (s *SSHService) connectSSH(conn SSHConnection) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            conn.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         s.config.ConnectTimeout,
	}

	// 添加认证方法
	// 优先使用内联私钥
	if conn.PrivateKey != "" {
		key, err := s.parsePrivateKeyFromString(conn.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("解析内联私钥失败: %v", err)
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(key))
	} else if conn.KeyPath != "" {
		key, err := s.loadPrivateKey(conn.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("加载私钥文件失败: %v", err)
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(key))
	}

	if conn.Password != "" {
		config.Auth = append(config.Auth, ssh.Password(conn.Password))
	}

	if len(config.Auth) == 0 {
		return nil, fmt.Errorf("未提供有效的认证方法")
	}

	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	return ssh.Dial("tcp", addr, config)
}

// loadPrivateKey 加载私钥
func (s *SSHService) loadPrivateKey(keyPath string) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// parsePrivateKeyFromString 从字符串解析私钥
func (s *SSHService) parsePrivateKeyFromString(keyContent string) (ssh.Signer, error) {
	key, err := ssh.ParsePrivateKey([]byte(keyContent))
	if err != nil {
		return nil, err
	}
	return key, nil
}

// executeDeploymentSteps 执行部署步骤
// Go代码只负责调度脚本执行和收集输出，所有安装逻辑都在Bash脚本中
func (s *SSHService) executeDeploymentSteps(client *ssh.Client, config SaltStackDeploymentConfig) (string, error) {
	var output strings.Builder

	// 读取脚本目录中的所有脚本
	scriptsDir := "scripts/salt-minion"
	scripts, err := s.loadDeploymentScripts(scriptsDir)
	if err != nil {
		return output.String(), fmt.Errorf("加载部署脚本失败: %v", err)
	}

	// 准备环境变量
	envVars := map[string]string{
		"APPHUB_URL":       config.AppHubURL,
		"SALT_MASTER_HOST": config.MasterHost,
		"SALT_MINION_ID":   "", // 可选，留空使用主机名
	}

	// 构建环境变量设置
	var envExports strings.Builder
	for key, value := range envVars {
		if value != "" {
			envExports.WriteString(fmt.Sprintf("export %s='%s'\n", key, value))
		}
	}

	// 按顺序执行每个脚本
	for _, script := range scripts {
		fmt.Fprintf(&output, "\n=== 执行脚本: %s ===\n", script.Name)

		// 组合环境变量和脚本内容
		fullCommand := envExports.String() + script.Content

		// 执行脚本
		stepOutput, err := s.executeCommand(client, fullCommand)
		fmt.Fprintf(&output, "%s\n", stepOutput)

		// 脚本执行失败，直接返回错误
		// Bash脚本中的set -e和exit码会确保错误被正确传递
		if err != nil {
			return output.String(), fmt.Errorf("脚本 '%s' 执行失败: %v", script.Name, err)
		}

		fmt.Fprintf(&output, "[✓] 脚本 %s 执行成功\n", script.Name)
	}

	return output.String(), nil
}

// DeploymentScript 表示一个部署脚本
type DeploymentScript struct {
	Name    string
	Path    string
	Content string
	Order   int
}

// loadDeploymentScripts 加载部署脚本目录中的所有脚本
func (s *SSHService) loadDeploymentScripts(dir string) ([]DeploymentScript, error) {
	var scripts []DeploymentScript

	// 读取目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取脚本目录失败: %v", err)
	}

	// 遍历文件
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理 .sh 文件
		if !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}

		// 读取脚本内容
		scriptPath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			return nil, fmt.Errorf("读取脚本 %s 失败: %v", entry.Name(), err)
		}

		// 提取序号（假设文件名格式为 NN-name.sh）
		order := 999 // 默认序号
		if len(entry.Name()) >= 2 {
			if num, err := strconv.Atoi(entry.Name()[:2]); err == nil {
				order = num
			}
		}

		scripts = append(scripts, DeploymentScript{
			Name:    entry.Name(),
			Path:    scriptPath,
			Content: string(content),
			Order:   order,
		})
	}

	// 按序号排序
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Order < scripts[j].Order
	})

	return scripts, nil
}

// waitForMinionsAccepted 等待 Minion 密钥被 Master 接受
// 返回每个主机的错误信息（如果有）
func (s *SSHService) waitForMinionsAccepted(ctx context.Context, hosts []string, masterHost string) map[string]error {
	errors := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()

			// 为每个 Minion 设置独立的超时（最多等待 3 分钟）
			minionCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
			defer cancel()

			err := s.waitForSingleMinionAccepted(minionCtx, h, masterHost)
			if err != nil {
				mu.Lock()
				errors[h] = err
				mu.Unlock()
			}
		}(host)
	}

	wg.Wait()
	return errors
}

// waitForSingleMinionAccepted 等待单个 Minion 被接受
func (s *SSHService) waitForSingleMinionAccepted(ctx context.Context, host, masterHost string) error {
	// Minion ID 通常是主机名，但也可能是 FQDN
	// 我们需要检查可能的 Minion ID 格式
	possibleMinionIDs := []string{
		host,                        // 原始主机名/IP
		strings.Split(host, ".")[0], // 短主机名
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待超时：Minion未能在指定时间内加入集群")
		case <-ticker.C:
			// 检查 Minion 是否已被接受
			accepted, err := s.checkMinionAccepted(masterHost, possibleMinionIDs)
			if err != nil {
				// 继续重试，不立即返回错误
				continue
			}
			if accepted {
				return nil
			}
		}
	}
}

// checkMinionAccepted 检查 Minion 是否已被 Master 接受
func (s *SSHService) checkMinionAccepted(masterHost string, possibleMinionIDs []string) (bool, error) {
	// 使用 Salt API 检查 minion 状态（不依赖 docker CLI）
	saltAPIURL := os.Getenv("SALTSTACK_MASTER_URL")
	if saltAPIURL == "" {
		// 构建默认 URL
		scheme := os.Getenv("SALT_API_SCHEME")
		if scheme == "" {
			scheme = "http"
		}
		host := os.Getenv("SALT_MASTER_HOST")
		if host == "" {
			host = "saltstack"
		}
		port := os.Getenv("SALT_API_PORT")
		if port == "" {
			port = "8002"
		}
		saltAPIURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	username := os.Getenv("SALT_API_USERNAME")
	if username == "" {
		username = "saltapi"
	}
	password := os.Getenv("SALT_API_PASSWORD")
	if password == "" {
		password = "your-salt-api-password"
	}
	eauth := os.Getenv("SALT_API_EAUTH")
	if eauth == "" {
		eauth = "file"
	}

	// 创建 HTTP 客户端
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. 认证获取 token
	authPayload := fmt.Sprintf(`{"username":"%s","password":"%s","eauth":"%s"}`, username, password, eauth)
	authReq, err := http.NewRequest("POST", saltAPIURL+"/login", strings.NewReader(authPayload))
	if err != nil {
		return false, fmt.Errorf("创建认证请求失败: %v", err)
	}
	authReq.Header.Set("Content-Type", "application/json")

	authResp, err := client.Do(authReq)
	if err != nil {
		log.Printf("[DEBUG] Salt API 认证请求失败: %v", err)
		return false, fmt.Errorf("Salt API 认证请求失败: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(authResp.Body)
		log.Printf("[DEBUG] Salt API 认证失败 (状态码 %d): %s", authResp.StatusCode, string(body))
		return false, fmt.Errorf("Salt API 认证失败，状态码: %d", authResp.StatusCode)
	}

	var authResult map[string]interface{}
	if err := json.NewDecoder(authResp.Body).Decode(&authResult); err != nil {
		return false, fmt.Errorf("解析认证响应失败: %v", err)
	}

	// 提取 token
	var token string
	if returnData, ok := authResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if t, ok := tokenData["token"].(string); ok {
				token = t
			}
		}
	}

	if token == "" {
		return false, fmt.Errorf("未能从认证响应中获取 token")
	}

	// 2. 获取密钥列表
	keysReq, err := http.NewRequest("GET", saltAPIURL+"/keys", nil)
	if err != nil {
		return false, fmt.Errorf("创建密钥列表请求失败: %v", err)
	}
	keysReq.Header.Set("X-Auth-Token", token)

	keysResp, err := client.Do(keysReq)
	if err != nil {
		log.Printf("[DEBUG] 获取 Salt 密钥列表失败: %v", err)
		return false, fmt.Errorf("获取 Salt 密钥列表失败: %v", err)
	}
	defer keysResp.Body.Close()

	if keysResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(keysResp.Body)
		log.Printf("[DEBUG] 获取密钥列表失败 (状态码 %d): %s", keysResp.StatusCode, string(body))
		return false, fmt.Errorf("获取密钥列表失败，状态码: %d", keysResp.StatusCode)
	}

	var keysResult map[string]interface{}
	if err := json.NewDecoder(keysResp.Body).Decode(&keysResult); err != nil {
		return false, fmt.Errorf("解析密钥列表响应失败: %v", err)
	}

	// 调试：输出原始响应
	keysJSON, _ := json.Marshal(keysResult)
	log.Printf("[DEBUG] Salt API keys响应: %s", string(keysJSON))

	// 解析密钥数据
	var minionsAccepted []string
	var minionsPending []string

	// 检查是否直接返回了字典（而不是嵌套在return数组中）
	if minions, ok := keysResult["minions"].([]interface{}); ok {
		// 直接格式: {"minions": [...], "minions_pre": [...]}
		for _, m := range minions {
			if minionID, ok := m.(string); ok {
				minionsAccepted = append(minionsAccepted, minionID)
			}
		}
		if minions, ok := keysResult["minions_pre"].([]interface{}); ok {
			for _, m := range minions {
				if minionID, ok := m.(string); ok {
					minionsPending = append(minionsPending, minionID)
				}
			}
		}
	} else if returnData, ok := keysResult["return"].([]interface{}); ok && len(returnData) > 0 {
		// 嵌套格式: {"return": [{"minions": [...], "minions_pre": [...]}]}
		if data, ok := returnData[0].(map[string]interface{}); ok {
			if minions, ok := data["minions"].([]interface{}); ok {
				for _, m := range minions {
					if minionID, ok := m.(string); ok {
						minionsAccepted = append(minionsAccepted, minionID)
					}
				}
			}
			if minions, ok := data["minions_pre"].([]interface{}); ok {
				for _, m := range minions {
					if minionID, ok := m.(string); ok {
						minionsPending = append(minionsPending, minionID)
					}
				}
			}
		}
	} else if returnData, ok := keysResult["return"].(map[string]interface{}); ok {
		// 另一种格式: {"return": {"minions": [...], "minions_pre": [...]}}
		if minions, ok := returnData["minions"].([]interface{}); ok {
			for _, m := range minions {
				if minionID, ok := m.(string); ok {
					minionsAccepted = append(minionsAccepted, minionID)
				}
			}
		}
		if minions, ok := returnData["minions_pre"].([]interface{}); ok {
			for _, m := range minions {
				if minionID, ok := m.(string); ok {
					minionsPending = append(minionsPending, minionID)
				}
			}
		}
	}

	log.Printf("[DEBUG] 检查 Minion 接受状态:")
	log.Printf("[DEBUG]   可能的 Minion IDs: %v", possibleMinionIDs)
	log.Printf("[DEBUG]   已接受的 Minions: %v", minionsAccepted)
	log.Printf("[DEBUG]   等待中的 Minions: %v", minionsPending)

	// 检查是否有任何可能的 Minion ID 已被接受
	for _, minionID := range possibleMinionIDs {
		for _, accepted := range minionsAccepted {
			if accepted == minionID {
				log.Printf("[DEBUG] ✓ Minion %s 已被接受", minionID)
				return true, nil
			}
		}
	}

	// 如果在 pending 列表中，尝试自动接受
	for _, minionID := range possibleMinionIDs {
		for _, pending := range minionsPending {
			if pending == minionID {
				log.Printf("[DEBUG] Minion %s 在等待列表中，尝试自动接受...", minionID)

				// 使用 Salt API 接受密钥
				acceptPayload := fmt.Sprintf(`{"client":"wheel","fun":"key.accept","match":"%s"}`, minionID)
				acceptReq, err := http.NewRequest("POST", saltAPIURL+"/", strings.NewReader(acceptPayload))
				if err != nil {
					return false, fmt.Errorf("创建接受密钥请求失败: %v", err)
				}
				acceptReq.Header.Set("X-Auth-Token", token)
				acceptReq.Header.Set("Content-Type", "application/json")

				acceptResp, err := client.Do(acceptReq)
				if err != nil {
					log.Printf("[DEBUG] 自动接受密钥请求失败: %v", err)
					return false, fmt.Errorf("自动接受密钥请求失败: %v", err)
				}
				defer acceptResp.Body.Close()

				if acceptResp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(acceptResp.Body)
					log.Printf("[DEBUG] 自动接受密钥失败 (状态码 %d): %s", acceptResp.StatusCode, string(body))
					return false, fmt.Errorf("自动接受密钥失败，状态码: %d", acceptResp.StatusCode)
				}

				log.Printf("[DEBUG] ✓ 已自动接受 Minion %s", minionID)
				return true, nil
			}
		}
	}

	log.Printf("[DEBUG] ✗ Minion 未找到在已接受或等待列表中")
	return false, nil
}

// getSaltStackContainerName 获取 SaltStack 容器名称
func getSaltStackContainerName() (string, error) {
	// 优先使用环境变量
	if containerName := os.Getenv("SALT_CONTAINER_NAME"); containerName != "" {
		return containerName, nil
	}

	// 尝试常见的容器名
	possibleContainers := []string{"ai-infra-saltstack", "saltstack", "salt-master"}
	for _, name := range possibleContainers {
		testCmd := exec.Command("docker", "exec", name, "echo", "test")
		if err := testCmd.Run(); err == nil {
			return name, nil
		}
	}

	return "", fmt.Errorf("无法找到 SaltStack 容器，尝试了: %v", possibleContainers)
}

// getInstallCommand 根据系统获取安装命令
func (s *SSHService) getInstallCommand() string {
	return `
set -e
if command -v apt-get >/dev/null 2>&1; then
	export DEBIAN_FRONTEND=noninteractive
	apt-get update -y || true
	if ! apt-get install -y salt-minion; then
		echo "[Salt] apt 安装失败，尝试添加Broadcom Salt仓库 (keyring)..."
		apt-get install -y curl gnupg2 ca-certificates lsb-release || true
		mkdir -p /usr/share/keyrings
		curl -fsSL https://packages.broadcom.com/artifactory/api/security/keypair/SaltProjectKey/public -o /usr/share/keyrings/salt-archive-keyring.gpg || true
		echo "deb [signed-by=/usr/share/keyrings/salt-archive-keyring.gpg] https://packages.broadcom.com/artifactory/saltproject-deb/ stable main" > /etc/apt/sources.list.d/saltproject.list
		apt-get update -y || true
		if ! apt-get install -y salt-minion; then
			echo "[Salt] Broadcom仓库安装失败，尝试使用官方bootstrap脚本..."
			curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		fi
	fi
elif command -v yum >/dev/null 2>&1; then
	if ! yum install -y salt-minion; then
		echo "[Salt] yum 安装失败，尝试添加Broadcom Salt仓库..."
		yum install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.el8.noarch.rpm || true
		yum clean all || true
		yum makecache -y || true
		if ! yum install -y salt-minion; then
			echo "[Salt] Broadcom仓库安装失败，尝试bootstrap脚本"
			curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		fi
	fi
elif command -v dnf >/dev/null 2>&1; then
	if ! dnf install -y salt-minion; then
		echo "[Salt] dnf 安装失败，尝试添加Broadcom Salt仓库..."
		dnf install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.fc36.noarch.rpm || true
		dnf makecache -y || true
		if ! dnf install -y salt-minion; then
			echo "[Salt] Broadcom仓库安装失败，尝试bootstrap脚本"
			curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
				|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		fi
	fi
elif command -v zypper >/dev/null 2>&1; then
	zypper refresh || true
	if ! zypper install -y salt-minion; then
		echo "[Salt] zypper安装失败，尝试bootstrap脚本"
		curl -fsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
			|| curl -kfsSL https://bootstrap.saltproject.io -o /tmp/install_salt.sh \
			|| wget -q --no-check-certificate https://bootstrap.saltproject.io -O /tmp/install_salt.sh
		sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
	fi
else
	echo "不支持的包管理器"
	exit 1
fi
`
}

// getMinionConfigCommand 获取Minion配置命令
func (s *SSHService) getMinionConfigCommand(config SaltStackDeploymentConfig) string {
	minionID := config.MinionID
	if minionID == "" {
		minionID = "$(hostname)"
	}

	// 使用正确的master地址格式，不包含端口号在master字段中
	masterHost := config.MasterHost
	if config.MasterPort != 4506 && config.MasterPort != 0 {
		masterHost = fmt.Sprintf("%s:%d", config.MasterHost, config.MasterPort)
	}

	return fmt.Sprintf(`
mkdir -p /etc/salt /var/log/salt
touch /var/log/salt/minion || true
chmod 644 /var/log/salt/minion || true
cat > /etc/salt/minion << EOF
master: %s
id: %s
log_level: info
master_port: %d
EOF

# 如果需要自动接受密钥
if [ "%t" = "true" ]; then
    echo "auto_accept: True" >> /etc/salt/minion
fi
`,
		masterHost, minionID, config.MasterPort, config.AutoAccept)
}

// getInstallCommandWithAppHub 根据系统获取安装命令（优先使用AppHub离线仓库）
func (s *SSHService) getInstallCommandWithAppHub(cfg SaltStackDeploymentConfig) string {
	apphub := strings.TrimSpace(cfg.AppHubURL)
	// 如果未提供AppHubURL，则回退到原有逻辑
	if apphub == "" {
		return s.getInstallCommand()
	}

	// 生成包含AppHub优先安装的脚本
	// 目录约定：
	//  - APT:  ${APPHUB_URL}/pkgs/saltstack-deb/ (应提供Packages索引)
	//  - RPM:  ${APPHUB_URL}/pkgs/saltstack-rpm/ (应提供repodata/repomd.xml)
	script := fmt.Sprintf(`
set -e
APPHUB_URL='%s'
installed=0
if command -v apt-get >/dev/null 2>&1; then
	export DEBIAN_FRONTEND=noninteractive
	# 检测AppHub APT索引
	if timeout 8 wget -q --spider "$APPHUB_URL/pkgs/saltstack-deb/Packages"; then
		echo "[Salt] 使用AppHub APT仓库安装salt-minion: $APPHUB_URL/pkgs/saltstack-deb"
		echo "deb [trusted=yes] $APPHUB_URL/pkgs/saltstack-deb ./" > /etc/apt/sources.list.d/ai-infra-salt.list
		apt-get update -y || true
		if apt-get install -y salt-minion; then
			installed=1
		else
			echo "[Salt] 从AppHub APT安装失败，尝试系统仓库/官方渠道..."
		fi
	else
		echo "[Salt] 未检测到AppHub APT索引(Packages)，跳过AppHub APT"
	fi
	if [ "$installed" -eq 0 ]; then
		apt-get update -y || true
		if apt-get install -y salt-minion; then
			installed=1
		fi
	fi
	# 如果系统仓库也没有，尝试 Broadcom 仓库
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] 系统仓库无salt-minion，尝试添加Broadcom Salt仓库 (keyring)..."
		apt-get install -y curl gnupg2 ca-certificates lsb-release wget || true
		mkdir -p /usr/share/keyrings
		if curl -fsSL --connect-timeout 10 --max-time 30 https://packages.broadcom.com/artifactory/api/security/keypair/SaltProjectKey/public -o /usr/share/keyrings/salt-archive-keyring.gpg; then
			echo "deb [signed-by=/usr/share/keyrings/salt-archive-keyring.gpg] https://packages.broadcom.com/artifactory/saltproject-deb/ stable main" > /etc/apt/sources.list.d/saltproject.list
			apt-get update -y || true
			if apt-get install -y salt-minion; then
				installed=1
			fi
		fi
	fi
	# 最后尝试 bootstrap 脚本
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] 所有仓库失败，使用官方bootstrap脚本..."
		if curl -fsSL --connect-timeout 10 --max-time 60 https://bootstrap.saltproject.io -o /tmp/install_salt.sh; then
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		else
			echo "[Salt] bootstrap脚本下载失败"
			exit 1
		fi
	fi
elif command -v yum >/dev/null 2>&1; then
	# 检测AppHub RPM元数据
	if timeout 8 wget -q --spider "$APPHUB_URL/pkgs/saltstack-rpm/repodata/repomd.xml"; then
		echo "[Salt] 使用AppHub YUM仓库安装salt-minion: $APPHUB_URL/pkgs/saltstack-rpm"
		cat > /etc/yum.repos.d/ai-infra-salt.repo <<EOF
[ai-infra-salt]
name=AI Infra Salt RPMs
baseurl=$APPHUB_URL/pkgs/saltstack-rpm
enabled=1
gpgcheck=0
EOF
		yum clean all || true
		yum makecache -y || true
		if yum install -y salt-minion; then
			installed=1
		else
			echo "[Salt] 从AppHub YUM安装失败，尝试系统仓库/官方渠道..."
		fi
	else
		echo "[Salt] 未检测到AppHub YUM元数据，跳过AppHub YUM"
	fi
	if [ "$installed" -eq 0 ]; then
		if yum install -y salt-minion; then
			installed=1
		fi
	fi
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] 系统仓库无salt-minion，尝试添加Salt官方仓库..."
		if yum install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.el8.noarch.rpm; then
			yum clean all || true
			yum makecache -y || true
			if yum install -y salt-minion; then
				installed=1
			fi
		fi
	fi
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] 所有仓库失败，使用bootstrap脚本"
		if curl -fsSL --connect-timeout 10 --max-time 60 https://bootstrap.saltproject.io -o /tmp/install_salt.sh; then
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		else
			echo "[Salt] bootstrap脚本下载失败"
			exit 1
		fi
	fi
elif command -v dnf >/dev/null 2>&1; then
	# 检测AppHub RPM元数据
	if timeout 8 wget -q --spider "$APPHUB_URL/pkgs/saltstack-rpm/repodata/repomd.xml"; then
		echo "[Salt] 使用AppHub DNF仓库安装salt-minion: $APPHUB_URL/pkgs/saltstack-rpm"
		cat > /etc/yum.repos.d/ai-infra-salt.repo <<EOF
[ai-infra-salt]
name=AI Infra Salt RPMs
baseurl=$APPHUB_URL/pkgs/saltstack-rpm
enabled=1
gpgcheck=0
EOF
		dnf makecache -y || true
		if dnf install -y salt-minion; then
			installed=1
		else
			echo "[Salt] 从AppHub DNF安装失败，尝试系统仓库/官方渠道..."
		fi
	else
		echo "[Salt] 未检测到AppHub DNF元数据，跳过AppHub DNF"
	fi
	if [ "$installed" -eq 0 ]; then
		if dnf install -y salt-minion; then
			installed=1
		fi
	fi
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] 系统仓库无salt-minion，尝试添加Salt官方仓库..."
		if dnf install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.fc36.noarch.rpm; then
			dnf makecache -y || true
			if dnf install -y salt-minion; then
				installed=1
			fi
		fi
	fi
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] 所有仓库失败，使用bootstrap脚本"
		if curl -fsSL --connect-timeout 10 --max-time 60 https://bootstrap.saltproject.io -o /tmp/install_salt.sh; then
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		else
			echo "[Salt] bootstrap脚本下载失败"
			exit 1
		fi
	fi
elif command -v zypper >/dev/null 2>&1; then
	zypper refresh || true
	if zypper install -y salt-minion; then
		installed=1
	fi
	if [ "$installed" -eq 0 ]; then
		echo "[Salt] zypper安装失败，尝试bootstrap脚本"
		if curl -fsSL --connect-timeout 10 --max-time 60 https://bootstrap.saltproject.io -o /tmp/install_salt.sh; then
			sh /tmp/install_salt.sh -X || sh /tmp/install_salt.sh || true
		else
			echo "[Salt] bootstrap脚本下载失败"
			exit 1
		fi
	fi
else
	echo "不支持的包管理器"
	exit 1
fi
`, apphub)

	return script
}

// 注意：保留在服务级别，不再内嵌更大的具体安装脚本；建议未来迁移到通用脚本执行接口

// executeCommand 执行SSH命令
func (s *SSHService) executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// 设置命令超时 - 对于安装命令使用更长的超时时间
	timeout := s.config.CommandTimeout
	if strings.Contains(command, "apt-get") || strings.Contains(command, "yum") ||
		strings.Contains(command, "dnf") || strings.Contains(command, "zypper") ||
		strings.Contains(command, "salt-minion") {
		timeout = 20 * time.Minute // SaltStack安装使用20分钟超时（从外网下载可能较慢）
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 获取输出管道
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return "", err
	}

	// 启动命令
	if err := session.Start(command); err != nil {
		return "", err
	}

	// 读取输出
	var output strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.readOutput(stdout, &output)
	}()

	go func() {
		defer wg.Done()
		s.readOutput(stderr, &output)
	}()

	// 等待命令完成或超时
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case err := <-done:
		wg.Wait()
		return output.String(), err
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return output.String(), ctx.Err()
	}
}

// readOutput 读取命令输出
func (s *SSHService) readOutput(reader io.Reader, output *strings.Builder) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Fprintf(output, "%s\n", scanner.Text())
	}
}

// executeCommandWithTimeout 允许为单次执行覆盖超时时间
func (s *SSHService) executeCommandWithTimeout(client *ssh.Client, command string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		// 使用默认逻辑
		return s.executeCommand(client, command)
	}

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := session.Start(command); err != nil {
		return "", err
	}

	var output strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.readOutput(stdout, &output)
	}()

	go func() {
		defer wg.Done()
		s.readOutput(stderr, &output)
	}()

	done := make(chan error, 1)
	go func() { done <- session.Wait() }()

	select {
	case err := <-done:
		wg.Wait()
		return output.String(), err
	case <-time.After(timeout):
		_ = session.Signal(ssh.SIGKILL)
		return output.String(), fmt.Errorf("命令超时 (%v)", timeout)
	}
}

// ExecuteCommandOnHosts 在多个主机上并发执行命令
func (s *SSHService) ExecuteCommandOnHosts(ctx context.Context, connections []SSHConnection, command string) ([]DeploymentResult, error) {
	results := make([]DeploymentResult, len(connections))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.MaxConcurrency)

	for i, conn := range connections {
		wg.Add(1)
		go func(index int, connection SSHConnection) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			startTime := time.Now()
			result := s.executeCommandOnHost(connection, command)
			result.Duration = time.Since(startTime)
			results[index] = result
		}(i, conn)
	}

	wg.Wait()
	return results, nil
}

// executeCommandOnHost 在单个主机上执行命令
func (s *SSHService) executeCommandOnHost(conn SSHConnection, command string) DeploymentResult {
	result := DeploymentResult{
		Host:    conn.Host,
		Success: false,
	}

	client, err := s.connectSSH(conn)
	if err != nil {
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer client.Close()

	output, err := s.executeCommand(client, command)
	if err != nil {
		result.Error = fmt.Sprintf("命令执行失败: %v", err)
		result.Output = output
		return result
	}

	result.Success = true
	result.Output = output
	return result
}

// ExecuteScriptOnHosts 在多个主机上并发执行脚本（通用接口）
func (s *SSHService) ExecuteScriptOnHosts(ctx context.Context, connections []SSHConnection, script ScriptSpec) ([]DeploymentResult, error) {
	results := make([]DeploymentResult, len(connections))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.MaxConcurrency)

	for i, conn := range connections {
		wg.Add(1)
		go func(index int, connection SSHConnection) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			res := s.executeScriptOnHost(ctx, connection, script)
			res.Duration = time.Since(start)
			results[index] = res
		}(i, conn)
	}

	wg.Wait()
	return results, nil
}

// executeScriptOnHost 在单个主机上执行脚本（通用接口）
func (s *SSHService) executeScriptOnHost(ctx context.Context, conn SSHConnection, spec ScriptSpec) DeploymentResult {
	result := DeploymentResult{Host: conn.Host}

	client, err := s.connectSSH(conn)
	if err != nil {
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer client.Close()

	// 1) 准备脚本到远端临时路径
	remoteScript := "/tmp/aimatrix_script_" + fmt.Sprintf("%d.sh", time.Now().UnixNano())

	if strings.TrimSpace(spec.Content) != "" {
		// 直接上传内容
		if err := s.UploadBinaryFile(conn.Host, conn.Port, conn.User, conn.Password, []byte(spec.Content), remoteScript, true); err != nil {
			result.Error = fmt.Sprintf("上传脚本失败: %v", err)
			return result
		}
	} else if strings.TrimSpace(spec.URL) != "" {
		// 在远端下载脚本
		downloadCmd := fmt.Sprintf(`/bin/sh -lc 'curl -fsSL %s -o %s || wget -q %s -O %s; chmod +x %s'`,
			singleQuote(spec.URL), remoteScript, singleQuote(spec.URL), remoteScript, remoteScript)
		if out, err := s.executeCommand(client, downloadCmd); err != nil {
			result.Output = out
			result.Error = fmt.Sprintf("下载脚本失败: %v", err)
			return result
		} else {
			result.Output += out
		}
	} else {
		result.Error = "ScriptSpec 必须提供 Content 或 URL"
		return result
	}

	// 2) 组装执行命令
	cmd := s.buildScriptExecutionCommand(remoteScript, spec)

	// 3) 按需设置超时并执行
	out, err := s.executeCommandWithTimeout(client, cmd, spec.Timeout)
	result.Output += out
	if err != nil {
		result.Error = fmt.Sprintf("脚本执行失败: %v", err)
		return result
	}

	// 4) 清理临时脚本
	_, _ = s.executeCommand(client, fmt.Sprintf("/bin/sh -lc 'rm -f %s'", remoteScript))

	result.Success = true
	return result
}

// buildScriptExecutionCommand 根据ScriptSpec拼装执行命令
func (s *SSHService) buildScriptExecutionCommand(remoteScript string, spec ScriptSpec) string {
	var parts []string
	// 环境变量
	if len(spec.Env) > 0 {
		var exports []string
		for k, v := range spec.Env {
			exports = append(exports, fmt.Sprintf("%s=%s", k, singleQuote(v)))
		}
		parts = append(parts, strings.Join(exports, " "))
	}
	// 切换目录
	if strings.TrimSpace(spec.WorkDir) != "" {
		parts = append(parts, "cd "+singleQuote(spec.WorkDir)+" &&")
	}
	// 执行器
	runner := remoteScript
	if strings.TrimSpace(spec.Interpreter) != "" {
		runner = fmt.Sprintf("%s %s", spec.Interpreter, remoteScript)
	}
	// 参数
	if len(spec.Args) > 0 {
		// 简单拼接，调用方需保证参数已做适当转义
		runner = runner + " " + strings.Join(spec.Args, " ")
	}
	// sudo
	if spec.UseSudo {
		runner = "sudo -E -n " + runner
	}
	parts = append(parts, runner)

	return "/bin/sh -lc " + singleQuote(strings.Join(parts, " "))
}

// ExecuteCommand 在指定主机上执行命令（导出方法，便于其他服务直接调用）
func (s *SSHService) ExecuteCommand(host string, port int, user, password, command string) (string, error) {
	conn := SSHConnection{Host: host, Port: port, User: user, Password: password, KeyPath: s.config.DefaultKeyPath}
	client, err := s.connectSSH(conn)
	if err != nil {
		return "", err
	}
	defer client.Close()
	return s.executeCommand(client, command)
}

// UploadFile 将内容写入远程文件（通过heredoc创建，避免外部SFTP依赖）
func (s *SSHService) UploadFile(host string, port int, user, password string, content []byte, remotePath string) error {
	// 为了避免二进制内容被shell转义破坏，使用base64安全传输
	return s.UploadBinaryFile(host, port, user, password, content, remotePath, true)
}

// ReadFile 读取远程文件内容
func (s *SSHService) ReadFile(host string, port int, user, password, remotePath string) (string, error) {
	cmd := fmt.Sprintf("/bin/sh -c 'cat %s'", remotePath)
	return s.ExecuteCommand(host, port, user, password, cmd)
}

// UploadBinaryFile 以base64方式安全上传二进制到远程，并可选设置可执行权限
func (s *SSHService) UploadBinaryFile(host string, port int, user, password string, content []byte, remotePath string, makeExecutable bool) error {
	// 分块写入临时b64文件，避免命令长度限制
	b64 := base64.StdEncoding.EncodeToString(content)
	const chunkSize = 32 * 1024
	tmp := "/tmp/.aimatrix_upload.b64"
	// 清理旧文件
	_, _ = s.ExecuteCommand(host, port, user, password, fmt.Sprintf("/bin/sh -lc 'rm -f %s'", tmp))
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		// 逐块追加
		cmd := fmt.Sprintf("/bin/sh -lc 'printf %s >> %s'", singleQuote(chunk), tmp)
		if _, err := s.ExecuteCommand(host, port, user, password, cmd); err != nil {
			return err
		}
	}
	// 解码并写入目标
	finalize := fmt.Sprintf("/bin/sh -lc 'base64 -d %s > %s'", tmp, remotePath)
	if _, err := s.ExecuteCommand(host, port, user, password, finalize); err != nil {
		return err
	}
	// 设置权限
	if makeExecutable {
		if _, err := s.ExecuteCommand(host, port, user, password, fmt.Sprintf("/bin/sh -lc 'chmod +x %s'", remotePath)); err != nil {
			return err
		}
	}
	// 清理临时文件
	_, _ = s.ExecuteCommand(host, port, user, password, fmt.Sprintf("/bin/sh -lc 'rm -f %s'", tmp))
	return nil
}

// singleQuote 将字符串安全包装为单引号shell字面量
func singleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// InitializeTestHosts 初始化测试主机，确保Docker容器启动
func (s *SSHService) InitializeTestHosts(ctx context.Context, hosts []string) ([]DeploymentResult, error) {
	results := make([]DeploymentResult, 0, len(hosts))

	// 检查哪些主机是测试容器
	testHosts := []string{}
	for _, host := range hosts {
		if strings.HasPrefix(host, "test-ssh") {
			testHosts = append(testHosts, host)
		}
	}

	if len(testHosts) == 0 {
		// 没有测试容器，返回所有主机为已就绪
		for _, host := range hosts {
			results = append(results, DeploymentResult{
				Host:     host,
				Success:  true,
				Output:   "非测试容器，跳过初始化",
				Error:    "",
				Duration: 0,
			})
		}
		return results, nil
	}

	// 对于测试容器，假设它们已经在运行，只需要验证连接
	start := time.Now()

	for _, host := range testHosts {
		// 测试SSH连接到容器
		conn := SSHConnection{
			Host:     host,
			Port:     22,
			User:     "root",
			Password: "rootpass123",
		}

		output, err := s.TestSSHConnection(ctx, conn)
		duration := time.Since(start)

		if err != nil {
			results = append(results, DeploymentResult{
				Host:     host,
				Success:  false,
				Output:   output,
				Error:    fmt.Sprintf("SSH连接测试失败: %v", err),
				Duration: duration,
			})
		} else {
			results = append(results, DeploymentResult{
				Host:     host,
				Success:  true,
				Output:   "测试容器连接正常",
				Error:    "",
				Duration: duration,
			})
		}
	}

	// 处理其他非测试主机
	for _, host := range hosts {
		found := false
		for _, testHost := range testHosts {
			if host == testHost {
				found = true
				break
			}
		}
		if !found {
			results = append(results, DeploymentResult{
				Host:     host,
				Success:  true,
				Output:   "非测试容器，跳过初始化",
				Error:    "",
				Duration: 0,
			})
		}
	}

	return results, nil
}

// executeDockerCommand 执行Docker命令
func (s *SSHService) executeDockerCommand(ctx context.Context, cmd string) (string, error) {
	// 这里我们使用本地执行Docker命令
	// 在实际部署中，这可能需要根据环境调整
	return s.executeLocalCommand(ctx, cmd)
}

// executeLocalCommand 执行本地命令
func (s *SSHService) executeLocalCommand(ctx context.Context, cmd string) (string, error) {
	// 使用bash执行命令
	cmdExec := exec.CommandContext(ctx, "bash", "-c", cmd)
	output, err := cmdExec.CombinedOutput()
	return string(output), err
}

// waitForSSHReady 等待SSH服务就绪
func (s *SSHService) waitForSSHReady(ctx context.Context, host string, port int, user, password string) DeploymentResult {
	start := time.Now()
	maxRetries := 30 // 最多等待30秒

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return DeploymentResult{
				Host:     host,
				Success:  false,
				Output:   "",
				Error:    "等待SSH就绪时超时",
				Duration: time.Since(start),
			}
		default:
		}

		// 尝试SSH连接
		config := &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         3 * time.Second,
		}

		address := fmt.Sprintf("%s:%d", host, port)
		conn, err := ssh.Dial("tcp", address, config)
		if err == nil {
			conn.Close()
			return DeploymentResult{
				Host:     host,
				Success:  true,
				Output:   fmt.Sprintf("SSH连接就绪，耗时 %v", time.Since(start)),
				Error:    "",
				Duration: time.Since(start),
			}
		}

		// 等待1秒后重试
		time.Sleep(1 * time.Second)
	}

	return DeploymentResult{
		Host:     host,
		Success:  false,
		Output:   "",
		Error:    fmt.Sprintf("等待SSH就绪超时，最后错误: 连接超时"),
		Duration: time.Since(start),
	}
}

// TestSSHConnection 测试SSH连接
func (s *SSHService) TestSSHConnection(ctx context.Context, conn SSHConnection) (string, error) {
	client, err := s.connectSSH(conn)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer client.Close()

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	// 执行简单的测试命令
	output, err := session.CombinedOutput("echo 'SSH连接测试成功' && whoami && hostname")
	if err != nil {
		return string(output), fmt.Errorf("执行测试命令失败: %v", err)
	}

	return string(output), nil
}

// uploadSaltMinionScript 上传salt-minion安装脚本到远程主机
func (s *SSHService) uploadSaltMinionScript(conn SSHConnection, config PackageInstallationConfig) StepResult {
	startTime := time.Now()

	// 读取本地的salt-minion安装脚本
	scriptPath := "/root/scripts/salt-minion/01-install-salt-minion.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return StepResult{
			Name:      "upload_salt_minion_script",
			Success:   false,
			Output:    "",
			Error:     fmt.Sprintf("读取安装脚本失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}

	// 上传脚本到远程主机
	remotePath := "/tmp/install-salt-minion.sh"
	err = s.UploadBinaryFile(
		conn.Host,
		conn.Port,
		conn.User,
		conn.Password,
		scriptContent,
		remotePath,
		true, // 设置可执行权限
	)

	if err != nil {
		return StepResult{
			Name:      "upload_salt_minion_script",
			Success:   false,
			Output:    "",
			Error:     fmt.Sprintf("上传脚本失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}

	return StepResult{
		Name:      "upload_salt_minion_script",
		Success:   true,
		Output:    fmt.Sprintf("已上传脚本到 %s:%s", conn.Host, remotePath),
		Error:     "",
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}
}

// executeSaltMinionScript 执行salt-minion安装脚本
func (s *SSHService) executeSaltMinionScript(client *ssh.Client, conn SSHConnection, config PackageInstallationConfig) StepResult {
	startTime := time.Now()

	// 构建执行命令，传递AppHub URL环境变量
	remotePath := "/tmp/install-salt-minion.sh"
	cmd := fmt.Sprintf("export APPHUB_URL='%s' && bash %s", config.AppHubConfig.BaseURL, remotePath)

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return StepResult{
			Name:      "execute_salt_minion_script",
			Success:   false,
			Output:    "",
			Error:     fmt.Sprintf("创建SSH会话失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}
	defer session.Close()

	// 执行脚本
	output, err := session.CombinedOutput(cmd)
	outputStr := string(output)

	if err != nil {
		return StepResult{
			Name:      "execute_salt_minion_script",
			Success:   false,
			Output:    outputStr,
			Error:     fmt.Sprintf("执行脚本失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}

	return StepResult{
		Name:      "execute_salt_minion_script",
		Success:   true,
		Output:    outputStr,
		Error:     "",
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}
}

// configureSaltMinion 配置salt-minion连接到master
func (s *SSHService) configureSaltMinion(client *ssh.Client, config PackageInstallationConfig, hostname string) StepResult {
	startTime := time.Now()

	minionID := s.getMinionID(config.MinionID, hostname)

	// 配置minion文件
	configCmd := fmt.Sprintf(`
cat > /etc/salt/minion.d/99-master-address.conf <<EOF
master: %s
master_port: %d
id: %s
EOF
`, config.SaltMasterHost, config.SaltMasterPort, minionID)

	session, err := client.NewSession()
	if err != nil {
		return StepResult{
			Name:      "configure_salt_minion",
			Success:   false,
			Output:    "",
			Error:     fmt.Sprintf("创建SSH会话失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}
	defer session.Close()

	output, err := session.CombinedOutput(configCmd)
	outputStr := string(output)

	if err != nil {
		return StepResult{
			Name:      "configure_salt_minion",
			Success:   false,
			Output:    outputStr,
			Error:     fmt.Sprintf("配置salt-minion失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}

	return StepResult{
		Name:      "configure_salt_minion",
		Success:   true,
		Output:    fmt.Sprintf("已配置minion连接到 %s:%d，ID: %s\n%s", config.SaltMasterHost, config.SaltMasterPort, minionID, outputStr),
		Error:     "",
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}
}

// startSaltMinion 启动salt-minion服务
func (s *SSHService) startSaltMinion(client *ssh.Client) StepResult {
	startTime := time.Now()

	// 尝试多种启动方式
	startCmd := `
systemctl daemon-reload 2>/dev/null || true
systemctl enable salt-minion 2>/dev/null || true
systemctl start salt-minion 2>/dev/null || service salt-minion start 2>/dev/null || salt-minion -d || true
sleep 2
systemctl status salt-minion 2>/dev/null || service salt-minion status 2>/dev/null || ps aux | grep salt-minion | grep -v grep
`

	session, err := client.NewSession()
	if err != nil {
		return StepResult{
			Name:      "start_salt_minion",
			Success:   false,
			Output:    "",
			Error:     fmt.Sprintf("创建SSH会话失败: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}
	defer session.Close()

	output, err := session.CombinedOutput(startCmd)
	outputStr := string(output)

	// 不强制要求命令成功，只要能看到进程就算成功
	success := err == nil || strings.Contains(outputStr, "salt-minion") || strings.Contains(outputStr, "active")

	return StepResult{
		Name:      "start_salt_minion",
		Success:   success,
		Output:    outputStr,
		Error:     "",
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}
}

// generateSlurmInstallationSteps 生成SLURM安装步骤
func (s *SSHService) generateSlurmInstallationSteps(config PackageInstallationConfig, hostname string) []InstallationStep {
	var steps []InstallationStep

	// SLURM客户端安装
	steps = append(steps, InstallationStep{
		Name:        "install_slurm_client",
		Description: "安装SLURM客户端组件",
		Critical:    false,
		Commands: []string{
			s.getInstallSlurmClientCommand(config.AppHubConfig.BaseURL),
		},
	})

	// SLURM节点配置
	steps = append(steps, InstallationStep{
		Name:        "configure_slurm_node",
		Description: "配置SLURM节点",
		Critical:    false,
		Commands: []string{
			s.getConfigureSlurmNodeCommand(config.SlurmRole, hostname),
		},
	})

	return steps
}
