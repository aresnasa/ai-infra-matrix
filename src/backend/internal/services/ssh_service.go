package services

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	AppHubConfig       AppHubConfig
	SaltMasterHost     string
	SaltMasterPort     int
	MinionID           string
	SlurmRole          string // controller|compute
	EnableSaltMinion   bool
	EnableSlurmClient  bool
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
	Name        string
	Success     bool
	Output      string
	Error       string
	Duration    time.Duration
	Timestamp   time.Time
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

// SaltStackDeploymentConfig SaltStack部署配置
type SaltStackDeploymentConfig struct {
	MasterHost string
	MasterPort int
	MinionID   string
	AutoAccept bool
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

	// 生成安装步骤
	steps := s.generateInstallationSteps(config, conn.Host)

	// 执行所有步骤
	allSuccess := true
	for _, step := range steps {
		stepResult := s.executeInstallationStep(client, step)
		result.Steps = append(result.Steps, stepResult)

		if !stepResult.Success && step.Critical {
			allSuccess = false
			result.Error = fmt.Sprintf("关键步骤失败: %s", step.Name)
			break
		}
	}

	result.Success = allSuccess
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
				"systemctl enable salt-minion",
				"systemctl start salt-minion",
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
    apt-get update -y
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
if command -v apt-get >/dev/null 2>&1; then
    # Ubuntu/Debian
    export DEBIAN_FRONTEND=noninteractive
    curl -fsSL https://packages.broadcom.com/artifactory/api/security/keypair/SaltProjectKey/public | apt-key add -
    echo "deb https://packages.broadcom.com/artifactory/saltproject-deb/ stable main" > /etc/apt/sources.list.d/saltstack.list
    apt-get update -y
    apt-get install -y salt-minion
elif command -v yum >/dev/null 2>&1; then
    # CentOS/RHEL
    yum install -y https://packages.broadcom.com/artifactory/saltproject-rpm/salt-3007-1.el8.noarch.rpm
    yum install -y salt-minion
elif command -v dnf >/dev/null 2>&1; then
    # Fedora
    dnf install -y https://packages.broadcom.com/artifactory/saltproject-rpm/salt-3007-1.fc36.noarch.rpm
    dnf install -y salt-minion
else
    echo "暂不支持的系统类型，尝试通用安装方法"
    curl -L https://bootstrap.saltproject.io -o install_salt.sh
    sh install_salt.sh -M -A 192.168.0.200
fi
`
}

// getConfigureSaltMinionCommand 获取配置SaltStack Minion的命令
func (s *SSHService) getConfigureSaltMinionCommand(masterHost string, masterPort int, minionID string) string {
	if masterPort == 0 {
		masterPort = 4506
	}

	return fmt.Sprintf(`
mkdir -p /etc/salt
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

# 创建日志目录
mkdir -p /var/log/salt
chown -R root:root /etc/salt /var/log/salt
chmod -R 644 /etc/salt/minion
`, masterHost, masterPort, minionID)
}

// getInstallSlurmClientCommand 获取安装SLURM客户端的命令
func (s *SSHService) getInstallSlurmClientCommand(appHubURL string) string {
	return fmt.Sprintf(`
if command -v apt-get >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
    # 从AppHub安装SLURM包
    apt-get install -y slurm-smd-client slurm-smd-slurmd || {
        echo "从APT源安装失败，尝试直接下载安装"
        cd /tmp
        wget -q %s/pkgs/slurm-deb/slurm-smd-client_25.05.3-1_arm64.deb
        wget -q %s/pkgs/slurm-deb/slurm-smd-slurmd_25.05.3-1_arm64.deb
        dpkg -i slurm-smd-client_25.05.3-1_arm64.deb slurm-smd-slurmd_25.05.3-1_arm64.deb || true
        apt-get install -f -y
    }
elif command -v yum >/dev/null 2>&1; then
    yum install -y slurm slurm-slurmd
elif command -v dnf >/dev/null 2>&1; then
    dnf install -y slurm slurm-slurmd
else
    echo "不支持的包管理器，跳过SLURM客户端安装"
    exit 1
fi
`, appHubURL, appHubURL)
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

# 节点配置
NodeName=%s CPUs=2 Sockets=1 CoresPerSocket=2 ThreadsPerCore=1 RealMemory=1000 State=UNKNOWN
PartitionName=compute Nodes=%s Default=YES MaxTime=INFINITE State=UP
EOF

# 设置权限
chown -R slurm:slurm /var/log/slurm /var/spool/slurm /etc/slurm 2>/dev/null || true
chmod 644 /etc/slurm/slurm.conf

# 根据角色启用相应服务
if [ "%s" = "controller" ]; then
    systemctl enable slurmctld 2>/dev/null || true
else
    systemctl enable slurmd 2>/dev/null || true
fi
`, hostname, hostname, role)
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
func (s *SSHService) executeDeploymentSteps(client *ssh.Client, config SaltStackDeploymentConfig) (string, error) {
	var output strings.Builder

	steps := []struct {
		name    string
		command string
	}{
		{"检查系统", "cat /etc/os-release"},
		{"安装SaltStack", s.getInstallCommand()},
		{"配置Minion", s.getMinionConfigCommand(config)},
		{"启动服务", "systemctl enable salt-minion && systemctl start salt-minion"},
		{"检查状态", "systemctl status salt-minion --no-pager"},
	}

	for _, step := range steps {
		fmt.Fprintf(&output, "\n=== %s ===\n", step.name)

		stepOutput, err := s.executeCommand(client, step.command)
		fmt.Fprintf(&output, "%s", stepOutput)

		if err != nil {
			return output.String(), fmt.Errorf("步骤 '%s' 失败: %v", step.name, err)
		}
	}

	return output.String(), nil
}

// getInstallCommand 根据系统获取安装命令
func (s *SSHService) getInstallCommand() string {
	return `
if command -v apt-get >/dev/null 2>&1; then
    apt-get update && apt-get install -y salt-minion
elif command -v yum >/dev/null 2>&1; then
    yum install -y salt-minion
elif command -v dnf >/dev/null 2>&1; then
    dnf install -y salt-minion
elif command -v zypper >/dev/null 2>&1; then
    zypper install -y salt-minion
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
		masterHost,
		minionID,
		config.MasterPort,
		config.AutoAccept)
}

// executeCommand 执行SSH命令
func (s *SSHService) executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// 设置命令超时
	ctx, cancel := context.WithTimeout(context.Background(), s.config.CommandTimeout)
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
	
	// 启动测试容器
	start := time.Now()
	
	// 构建docker-compose命令来启动指定的服务
	services := strings.Join(testHosts, " ")
	cmd := fmt.Sprintf("cd /workspaces && docker-compose -f docker-compose.test.yml up -d %s", services)
	
	// 执行命令启动容器
	output, err := s.executeDockerCommand(ctx, cmd)
	duration := time.Since(start)
	
	if err != nil {
		// 启动失败，所有测试容器标记为失败
		for _, host := range testHosts {
			results = append(results, DeploymentResult{
				Host:     host,
				Success:  false,
				Output:   output,
				Error:    fmt.Sprintf("启动容器失败: %v", err),
				Duration: duration,
			})
		}
		// 其他主机标记为就绪
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
	
	// 等待容器完全启动并可接受SSH连接
	for _, host := range testHosts {
		hostResult := s.waitForSSHReady(ctx, host, 22, "root", "rootpass123")
		results = append(results, hostResult)
	}
	
	// 处理非测试容器
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
