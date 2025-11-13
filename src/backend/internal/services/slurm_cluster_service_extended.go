package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"golang.org/x/crypto/ssh"
)

// 继续 slurm_cluster_service.go 的其余实现

// configureSalt 配置SaltStack
func (s *SlurmClusterService) configureSalt(task *DeploymentTask, cluster *models.SlurmCluster, deployment *models.ClusterDeployment) error {
	task.CurrentStep = "Configuring SaltStack"

	stepRecord := s.createDeploymentStep(task.DeploymentID, "salt-configuration", "salt", "Configuring SaltStack Master and Minions")

	// TODO: 配置Salt Master接受Minion密钥
	// 这里需要实现Salt Master的密钥接受逻辑

	s.completeDeploymentStep(stepRecord, "completed", "SaltStack configured successfully")
	return nil
}

// installSlurm 安装SLURM
func (s *SlurmClusterService) installSlurm(task *DeploymentTask, cluster *models.SlurmCluster, deployment *models.ClusterDeployment) error {
	task.CurrentStep = "Installing SLURM"

	stepRecord := s.createDeploymentStep(task.DeploymentID, "slurm-installation", "slurm", "Installing SLURM on all nodes")

	var wg sync.WaitGroup
	errChan := make(chan error, len(cluster.Nodes))

	// 并发安装SLURM
	for _, node := range cluster.Nodes {
		wg.Add(1)
		go func(n models.SlurmNode) {
			defer wg.Done()

			nodeTask := task.NodeTasks[n.ID]
			nodeTask.Status = "running"
			nodeTask.CurrentStep = "Installing SLURM"

			if err := s.installSlurmOnNode(&n, cluster); err != nil {
				nodeTask.Status = "failed"
				nodeTask.Error = err.Error()
				errChan <- fmt.Errorf("node %s: %v", n.NodeName, err)
			} else {
				nodeTask.Status = "completed"
				nodeTask.Progress = 100
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	// 检查错误
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		s.completeDeploymentStep(stepRecord, "failed", strings.Join(errors, "; "))
		return fmt.Errorf("SLURM installation failed: %s", strings.Join(errors, "; "))
	}

	s.completeDeploymentStep(stepRecord, "completed", "SLURM installed on all nodes")
	return nil
}

// startAndValidateSlurm 启动和验证SLURM服务
func (s *SlurmClusterService) startAndValidateSlurm(task *DeploymentTask, cluster *models.SlurmCluster, deployment *models.ClusterDeployment) error {
	task.CurrentStep = "Starting SLURM services"

	stepRecord := s.createDeploymentStep(task.DeploymentID, "slurm-startup", "slurm", "Starting and validating SLURM services")

	// 首先启动slurmctld（在master节点）
	for _, node := range cluster.Nodes {
		if node.NodeType == "master" {
			if err := s.startSlurmController(&node); err != nil {
				s.completeDeploymentStep(stepRecord, "failed", fmt.Sprintf("Failed to start slurmctld on %s: %v", node.NodeName, err))
				return err
			}
			break
		}
	}

	// 然后启动slurmd（在compute节点）
	for _, node := range cluster.Nodes {
		if node.NodeType == "compute" {
			if err := s.startSlurmDaemon(&node); err != nil {
				s.completeDeploymentStep(stepRecord, "failed", fmt.Sprintf("Failed to start slurmd on %s: %v", node.NodeName, err))
				return err
			}
		}
	}

	s.completeDeploymentStep(stepRecord, "completed", "SLURM services started successfully")
	return nil
}

// finalValidation 最终验证
func (s *SlurmClusterService) finalValidation(task *DeploymentTask, cluster *models.SlurmCluster, deployment *models.ClusterDeployment) error {
	task.CurrentStep = "Final validation"

	stepRecord := s.createDeploymentStep(task.DeploymentID, "final-validation", "validation", "Performing final cluster validation")

	// 验证集群状态
	for _, node := range cluster.Nodes {
		if node.NodeType == "master" {
			if err := s.validateSlurmCluster(&node); err != nil {
				s.completeDeploymentStep(stepRecord, "failed", fmt.Sprintf("Cluster validation failed: %v", err))
				return err
			}
			break
		}
	}

	s.completeDeploymentStep(stepRecord, "completed", "Final validation completed successfully")
	return nil
}

// 具体的SSH执行方法

// testNodeSSHConnection 测试节点SSH连接
func (s *SlurmClusterService) testNodeSSHConnection(node *models.SlurmNode) error {
	sshConfig := &ssh.ClientConfig{
		User: node.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(node.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", node.Host, node.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return err
	}
	defer client.Close()

	// 执行简单的测试命令
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if err := session.Run("echo 'SSH connection test successful'"); err != nil {
		return err
	}

	return nil
}

// detectOSInfo 检测操作系统信息
func (s *SlurmClusterService) detectOSInfo(client *ssh.Client, sessionID string, node *models.SlurmNode, installTask *models.NodeInstallTask, step *models.InstallStep) (*models.OSInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	cmd := `
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

	startTime := time.Now()
	sshLog := &models.SSHExecutionLog{
		SessionID: sessionID,
		NodeID:    node.ID,
		TaskID:    installTask.ID,
		Host:      node.Host,
		Port:      node.Port,
		Username:  node.Username,
		Command:   cmd,
		StartedAt: startTime,
	}

	output, err := session.Output(cmd)
	completedAt := time.Now()
	sshLog.CompletedAt = &completedAt
	sshLog.Duration = int(completedAt.Sub(startTime).Milliseconds())

	if err != nil {
		sshLog.Success = false
		sshLog.ErrorOutput = err.Error()
		sshLog.ExitCode = 1
		s.db.Create(sshLog)
		return nil, err
	}

	sshLog.Success = true
	sshLog.Output = string(output)
	sshLog.ExitCode = 0
	s.db.Create(sshLog)

	osInfo := &models.OSInfo{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "OS":
			osInfo.OS = value
		case "VERSION":
			osInfo.Version = value
		case "ARCH":
			osInfo.Arch = value
		}
	}

	return osInfo, nil
}

// installSaltRepository 安装SaltStack仓库
func (s *SlurmClusterService) installSaltRepository(client *ssh.Client, osInfo *models.OSInfo, sessionID string, node *models.SlurmNode, installTask *models.NodeInstallTask, step *models.InstallStep) error {
	var cmd string

	switch osInfo.OS {
	case "ubuntu", "debian":
		cmd = `
			set -e
			curl -fsSL https://repo.saltproject.io/py3/ubuntu/20.04/amd64/latest/salt-archive-keyring.gpg | sudo apt-key add -
			echo "deb https://repo.saltproject.io/py3/ubuntu/20.04/amd64/latest focal main" | sudo tee /etc/apt/sources.list.d/salt.list
			sudo apt-get update
		`
	case "centos", "rhel":
		cmd = `
			set -e
			sudo yum install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.el8.noarch.rpm
		`
	default:
		return fmt.Errorf("unsupported OS: %s", osInfo.OS)
	}

	return s.executeSSHCommand(client, cmd, sessionID, node, installTask, step)
}

// installSaltMinionPackage 安装Salt Minion包（从AppHub下载）
func (s *SlurmClusterService) installSaltMinionPackage(client *ssh.Client, osInfo *models.OSInfo, sessionID string, node *models.SlurmNode, installTask *models.NodeInstallTask, step *models.InstallStep) error {
	// 获取AppHub配置
	appHubURL := getAppHubBaseURL()

	// 使用统一的安装脚本（支持所有操作系统）
	scriptPath := "/root/scripts/salt-minion/01-install-salt-minion.sh"

	// 复制脚本到远程主机
	if err := s.copyScriptToRemote(client, scriptPath, "/tmp/install-salt-minion.sh"); err != nil {
		return fmt.Errorf("failed to copy install script: %v", err)
	}

	// 执行安装脚本（新脚本通过环境变量接收AppHub URL）
	cmd := fmt.Sprintf("export APPHUB_URL='%s' && bash /tmp/install-salt-minion.sh", appHubURL)
	if err := s.executeSSHCommand(client, cmd, sessionID, node, installTask, step); err != nil {
		return fmt.Errorf("failed to execute install script: %v", err)
	}

	// 清理远程脚本
	cleanupCmd := "rm -f /tmp/install-salt-minion.sh"
	_ = s.executeSSHCommand(client, cleanupCmd, sessionID, node, installTask, step)

	return nil
}

// copyScriptToRemote 复制脚本文件到远程主机（使用SSH而不是SFTP）
func (s *SlurmClusterService) copyScriptToRemote(client *ssh.Client, localPath, remotePath string) error {
	// 读取本地脚本内容
	scriptContent, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read script file %s: %v", localPath, err)
	}

	// 创建SSH会话
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// 使用 cat 命令将脚本内容写入远程文件
	// 使用 base64 编码避免特殊字符问题
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	// 启动命令：接收stdin并写入文件
	if err := session.Start(fmt.Sprintf("cat > %s && chmod +x %s", remotePath, remotePath)); err != nil {
		return fmt.Errorf("failed to start copy command: %v", err)
	}

	// 写入脚本内容
	if _, err := io.Copy(stdin, strings.NewReader(string(scriptContent))); err != nil {
		return fmt.Errorf("failed to write script content: %v", err)
	}
	stdin.Close()

	// 等待命令完成
	if err := session.Wait(); err != nil {
		return fmt.Errorf("failed to complete script copy: %v", err)
	}

	return nil
}

// getSaltStackVersion 获取SaltStack版本
func getSaltStackVersion() string {
	// 从环境变量读取版本，如果没有则使用默认值
	if version := os.Getenv("SALTSTACK_VERSION"); version != "" {
		// 移除可能的 'v' 前缀
		return strings.TrimPrefix(version, "v")
	}
	return "3007.8" // 默认版本
}

// getAppHubBaseURL 获取AppHub基础URL
func getAppHubBaseURL() string {
	// 从环境变量获取，或使用默认值
	if url := os.Getenv("APPHUB_URL"); url != "" {
		return strings.TrimSuffix(url, "/")
	}

	// 构建AppHub URL（与slurm_controller.go中的逻辑一致）
	externalHost := os.Getenv("EXTERNAL_HOST")
	if externalHost == "" {
		externalHost = "localhost"
	}

	appHubPort := os.Getenv("APPHUB_PORT")
	if appHubPort == "" {
		appHubPort = "53434" // 默认端口
	}

	scheme := os.Getenv("EXTERNAL_SCHEME")
	if scheme == "" {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s:%s", scheme, externalHost, appHubPort)
}

// configureSaltMinion 配置Salt Minion
func (s *SlurmClusterService) configureSaltMinion(client *ssh.Client, cluster *models.SlurmCluster, node *models.SlurmNode, sessionID string, installTask *models.NodeInstallTask, step *models.InstallStep) error {
	configContent := fmt.Sprintf(`
master: %s
id: %s
mine_enabled: true
mine_return_job: true
mine_interval: 60
`, cluster.SaltMaster, node.SaltMinionID)

	cmd := fmt.Sprintf(`
		set -e
		sudo mkdir -p /etc/salt
		echo '%s' | sudo tee /etc/salt/minion
		sudo systemctl enable salt-minion
	`, configContent)

	return s.executeSSHCommand(client, cmd, sessionID, node, installTask, step)
}

// startSaltMinionService 启动Salt Minion服务
func (s *SlurmClusterService) startSaltMinionService(client *ssh.Client, sessionID string, node *models.SlurmNode, installTask *models.NodeInstallTask, step *models.InstallStep) error {
	cmd := `
		set -e
		sudo systemctl start salt-minion
		sudo systemctl status salt-minion --no-pager
	`

	return s.executeSSHCommand(client, cmd, sessionID, node, installTask, step)
}

// installSlurmOnNode 在节点上安装SLURM
func (s *SlurmClusterService) installSlurmOnNode(node *models.SlurmNode, cluster *models.SlurmCluster) error {
	log.Printf("[INFO] 开始在节点 %s 上安装 SLURM", node.Host)

	// 创建SSH配置
	config := RemoteNodeInitConfig{
		Host:     node.Host,
		Port:     node.Port,
		Username: node.Username,
		Password: node.Password,
	}

	// 建立SSH连接
	client, err := s.createSSHClient(config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer client.Close()

	// 1. 检测操作系统类型
	osType, err := s.detectOSType(client)
	if err != nil {
		return fmt.Errorf("检测操作系统失败: %v", err)
	}
	log.Printf("[INFO] 检测到操作系统: %s", osType)

	// 2. 安装 SLURM 和 Munge 包
	if err := s.installSlurmPackages(client, osType); err != nil {
		return fmt.Errorf("安装SLURM包失败: %v", err)
	}

	// 3. 创建必要的目录和用户
	if err := s.setupSlurmDirectories(client, osType); err != nil {
		return fmt.Errorf("设置SLURM目录失败: %v", err)
	}

	// 4. 配置 Munge 密钥
	if err := s.configureMungeKey(client, osType); err != nil {
		return fmt.Errorf("配置Munge密钥失败: %v", err)
	}

	// 5. 配置 slurm.conf
	if err := s.configureSlurmConf(client, osType, cluster); err != nil {
		return fmt.Errorf("配置slurm.conf失败: %v", err)
	}

	// 6. 启动 Munge 服务
	if err := s.startMungeServiceDirect(client, osType); err != nil {
		return fmt.Errorf("启动Munge服务失败: %v", err)
	}

	// 7. 启动 SLURMD 服务
	if err := s.startSlurmdServiceDirect(client, osType); err != nil {
		return fmt.Errorf("启动SLURMD服务失败: %v", err)
	}

	log.Printf("[INFO] 节点 %s SLURM安装完成", node.Host)
	return nil
}

// startSlurmController 启动SLURM控制器
func (s *SlurmClusterService) startSlurmController(node *models.SlurmNode) error {
	// TODO: 通过SaltStack启动slurmctld服务
	return nil
}

// startSlurmDaemon 启动SLURM守护进程
func (s *SlurmClusterService) startSlurmDaemon(node *models.SlurmNode) error {
	// TODO: 通过SaltStack启动slurmd服务
	return nil
}

// validateSlurmCluster 验证SLURM集群
func (s *SlurmClusterService) validateSlurmCluster(node *models.SlurmNode) error {
	// TODO: 验证SLURM集群状态
	return nil
}

// executeSSHCommand 执行SSH命令并记录日志
func (s *SlurmClusterService) executeSSHCommand(client *ssh.Client, cmd, sessionID string, node *models.SlurmNode, installTask *models.NodeInstallTask, step *models.InstallStep) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	startTime := time.Now()
	sshLog := &models.SSHExecutionLog{
		SessionID: sessionID,
		NodeID:    node.ID,
		TaskID:    installTask.ID,
		Host:      node.Host,
		Port:      node.Port,
		Username:  node.Username,
		Command:   cmd,
		StartedAt: startTime,
	}

	output, err := session.CombinedOutput(cmd)
	completedAt := time.Now()
	sshLog.CompletedAt = &completedAt
	sshLog.Duration = int(completedAt.Sub(startTime).Milliseconds())

	if err != nil {
		sshLog.Success = false
		sshLog.ErrorOutput = string(output)
		sshLog.ExitCode = 1
		s.db.Create(sshLog)
		return fmt.Errorf("command failed: %v, output: %s", err, string(output))
	}

	sshLog.Success = true
	sshLog.Output = string(output)
	sshLog.ExitCode = 0
	s.db.Create(sshLog)

	return nil
}

// 数据库记录辅助方法
func (s *SlurmClusterService) createDeploymentStep(deploymentID uint, stepName, stepType, description string) *models.DeploymentStep {
	step := &models.DeploymentStep{
		DeploymentID: deploymentID,
		StepName:     stepName,
		StepType:     stepType,
		Status:       "running",
	}
	startedAt := time.Now()
	step.StartedAt = &startedAt
	s.db.Create(step)
	return step
}

func (s *SlurmClusterService) completeDeploymentStep(step *models.DeploymentStep, status, output string) {
	completedAt := time.Now()
	step.CompletedAt = &completedAt
	step.Status = status
	step.Duration = int(completedAt.Sub(*step.StartedAt).Seconds())
	if status == "failed" {
		step.ErrorMessage = output
	} else {
		step.Output = output
	}
	s.db.Save(step)
}

func (s *SlurmClusterService) createInstallStep(taskID uint, stepName, stepType, description string) *models.InstallStep {
	step := &models.InstallStep{
		TaskID:     taskID,
		StepName:   stepName,
		StepType:   stepType,
		Status:     "running",
		MaxRetries: 3,
	}
	startedAt := time.Now()
	step.StartedAt = &startedAt
	s.db.Create(step)
	return step
}

func (s *SlurmClusterService) completeInstallStep(step *models.InstallStep, status, message string) {
	completedAt := time.Now()
	step.CompletedAt = &completedAt
	step.Status = status
	step.Duration = int(completedAt.Sub(*step.StartedAt).Seconds())
	if status == "failed" {
		step.ErrorMessage = message
	} else {
		step.Output = message
	}
	s.db.Save(step)
}

// SLURM 节点安装辅助函数

// detectOSType 检测操作系统类型
func (s *SlurmClusterService) detectOSType(client *ssh.Client) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput("cat /etc/os-release")
	if err != nil {
		return "", fmt.Errorf("读取 os-release 失败: %v", err)
	}

	osInfo := string(output)
	if strings.Contains(osInfo, "ID=rocky") || strings.Contains(osInfo, "ID=\"rocky\"") {
		return "rocky", nil
	} else if strings.Contains(osInfo, "ID=centos") || strings.Contains(osInfo, "ID=\"centos\"") {
		return "centos", nil
	} else if strings.Contains(osInfo, "ID=ubuntu") || strings.Contains(osInfo, "ID=\"ubuntu\"") {
		return "ubuntu", nil
	} else if strings.Contains(osInfo, "ID=debian") || strings.Contains(osInfo, "ID=\"debian\"") {
		return "debian", nil
	} else if strings.Contains(osInfo, "ID=alpine") || strings.Contains(osInfo, "ID=\"alpine\"") {
		return "alpine", nil
	}

	return "unknown", nil
}

// installSlurmPackages 安装 SLURM 和 Munge 包
func (s *SlurmClusterService) installSlurmPackages(client *ssh.Client, osType string) error {
	var cmd string

	switch osType {
	case "rocky", "centos", "rhel", "almalinux":
		cmd = `
set -e
echo "[安装] 安装 EPEL 仓库..."
dnf install -y epel-release 2>/dev/null || yum install -y epel-release
dnf makecache --refresh 2>/dev/null || yum makecache

echo "[安装] 安装 Munge..."
dnf install -y munge munge-libs 2>/dev/null || yum install -y munge munge-libs

echo "[安装] 安装 SLURM..."
dnf install -y slurm slurm-slurmd slurm-contribs 2>/dev/null || yum install -y slurm slurm-slurmd slurm-contribs || {
    echo "警告: SLURM 从仓库安装失败，安装依赖..."
    dnf install -y munge-devel pam-devel perl perl-ExtUtils-MakeMaker readline-devel 2>/dev/null || true
}

echo "[完成] SLURM 和 Munge 安装完成"
`
	case "ubuntu", "debian":
		cmd = `
set -e
echo "[安装] 更新包索引..."
apt-get update -qq

echo "[安装] 安装 Munge..."
DEBIAN_FRONTEND=noninteractive apt-get install -y munge libmunge-dev

echo "[安装] 安装 SLURM..."
DEBIAN_FRONTEND=noninteractive apt-get install -y slurmd slurm-client

echo "[完成] SLURM 和 Munge 安装完成"
`
	case "alpine":
		cmd = `
set -e
echo "[安装] 安装 SLURM 和 Munge..."
apk add --no-cache slurm munge

echo "[完成] SLURM 和 Munge 安装完成"
`
	default:
		return fmt.Errorf("不支持的操作系统: %s", osType)
	}

	return s.executeSSHCmd(client, cmd)
}

// setupSlurmDirectories 创建 SLURM 和 Munge 目录及用户
func (s *SlurmClusterService) setupSlurmDirectories(client *ssh.Client, osType string) error {
	var cmd string

	if osType == "alpine" {
		cmd = `
set -e
echo "[设置] 创建 Munge 目录..."
mkdir -p /etc/munge /var/log/munge /var/lib/munge /run/munge

echo "[设置] 创建 SLURM 目录..."
mkdir -p /etc/slurm /var/log/slurm /var/spool/slurmd /var/run/slurm

echo "[设置] 创建用户..."
if ! id -u munge >/dev/null 2>&1; then
    adduser -D -s /bin/false munge
fi

if ! id -u slurm >/dev/null 2>&1; then
    adduser -D -s /bin/false slurm
fi

echo "[设置] 设置权限..."
chown -R munge:munge /etc/munge /var/log/munge /var/lib/munge /run/munge
chmod 700 /etc/munge
chmod 711 /var/lib/munge
chmod 755 /var/log/munge /run/munge

chown -R slurm:slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
chmod 755 /var/log/slurm /var/spool/slurmd /var/run/slurm

echo "[完成] 目录设置完成"
`
	} else {
		cmd = `
set -e
echo "[设置] 创建 Munge 目录..."
mkdir -p /etc/munge /var/log/munge /var/lib/munge /run/munge

echo "[设置] 创建 SLURM 目录..."
mkdir -p /etc/slurm /var/log/slurm /var/spool/slurmd /var/run/slurm

echo "[设置] 创建用户..."
if ! id -u munge >/dev/null 2>&1; then
    useradd -r -s /bin/false munge || true
fi

if ! id -u slurm >/dev/null 2>&1; then
    useradd -r -s /bin/false slurm || true
fi

echo "[设置] 设置权限..."
chown -R munge:munge /etc/munge /var/log/munge /var/lib/munge /run/munge
chmod 700 /etc/munge
chmod 711 /var/lib/munge
chmod 755 /var/log/munge /run/munge

chown -R slurm:slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
chmod 755 /var/log/slurm /var/spool/slurmd /var/run/slurm

echo "[完成] 目录设置完成"
`
	}

	return s.executeSSHCmd(client, cmd)
}

// configureMungeKey 配置 Munge 密钥
func (s *SlurmClusterService) configureMungeKey(client *ssh.Client, osType string) error {
	log.Printf("[配置] 开始配置 Munge 密钥...")

	// 从 SLURM Master 读取 munge.key
	mungeKey, err := s.getMungeKeyFromMaster()
	if err != nil {
		return fmt.Errorf("获取 Munge 密钥失败: %v", err)
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "munge.key.*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(mungeKey); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 通过 SFTP 上传文件
	if err := s.uploadFileViaSSH(client, tmpFile.Name(), "/etc/munge/munge.key"); err != nil {
		return fmt.Errorf("上传 munge.key 失败: %v", err)
	}

	// 设置权限
	cmd := `
chown munge:munge /etc/munge/munge.key
chmod 400 /etc/munge/munge.key
echo "[完成] Munge 密钥配置完成"
`
	return s.executeSSHCmd(client, cmd)
}

// configureSlurmConf 配置 slurm.conf
func (s *SlurmClusterService) configureSlurmConf(client *ssh.Client, osType string, cluster *models.SlurmCluster) error {
	log.Printf("[配置] 开始配置 slurm.conf...")

	// 从 SLURM Master 读取 slurm.conf
	slurmConf, err := s.getSlurmConfFromMaster()
	if err != nil {
		return fmt.Errorf("获取 slurm.conf 失败: %v", err)
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "slurm.conf.*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(slurmConf); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 确定目标路径
	targetPath := "/etc/slurm/slurm.conf"
	if osType == "ubuntu" || osType == "debian" {
		// Ubuntu 可能使用 /etc/slurm-llnl/slurm.conf
		checkCmd := "test -d /etc/slurm-llnl && echo 'llnl' || echo 'slurm'"
		session, err := client.NewSession()
		if err == nil {
			output, _ := session.CombinedOutput(checkCmd)
			session.Close()
			if strings.TrimSpace(string(output)) == "llnl" {
				targetPath = "/etc/slurm-llnl/slurm.conf"
				// 确保目录存在
				s.executeSSHCmd(client, "mkdir -p /etc/slurm-llnl")
			}
		}
	}

	// 通过 SFTP 上传文件
	if err := s.uploadFileViaSSH(client, tmpFile.Name(), targetPath); err != nil {
		return fmt.Errorf("上传 slurm.conf 失败: %v", err)
	}

	// 设置权限
	cmd := fmt.Sprintf(`
chmod 644 %s
echo "[完成] slurm.conf 配置完成"
`, targetPath)
	return s.executeSSHCmd(client, cmd)
}

// startMungeServiceDirect 直接启动 Munge 服务（不依赖脚本）
func (s *SlurmClusterService) startMungeServiceDirect(client *ssh.Client, osType string) error {
	log.Printf("[启动] 启动 Munge 服务...")

	var cmd string
	if osType == "alpine" {
		cmd = `
set -e
echo "[启动] 启动 Munge 服务..."
rc-service munge stop 2>/dev/null || true
rc-update add munge default 2>/dev/null || true
rc-service munge start || munged -f &

sleep 2

if munge -n | unmunge >/dev/null 2>&1; then
    echo "[成功] Munge 服务运行正常"
else
    echo "[错误] Munge 验证失败"
    exit 1
fi
`
	} else {
		cmd = `
set -e
echo "[启动] 启动 Munge 服务..."
systemctl stop munge 2>/dev/null || true
systemctl enable munge 2>/dev/null || true
systemctl start munge || munged -f &

sleep 2

if munge -n | unmunge >/dev/null 2>&1; then
    echo "[成功] Munge 服务运行正常"
else
    echo "[错误] Munge 验证失败"
    exit 1
fi
`
	}

	return s.executeSSHCmd(client, cmd)
}

// startSlurmdServiceDirect 直接启动 SLURMD 服务（不依赖脚本）
func (s *SlurmClusterService) startSlurmdServiceDirect(client *ssh.Client, osType string) error {
	log.Printf("[启动] 启动 SLURMD 服务...")

	var cmd string
	if osType == "alpine" {
		cmd = `
set -e
echo "[启动] 启动 SLURMD 服务..."
rc-service slurmd stop 2>/dev/null || true
rc-update add slurmd default 2>/dev/null || true
rc-service slurmd start || slurmd -D &

sleep 2

if pgrep -x slurmd >/dev/null; then
    echo "[成功] SLURMD 服务运行正常"
else
    echo "[错误] SLURMD 服务未运行"
    exit 1
fi
`
	} else {
		cmd = `
set -e
echo "[启动] 启动 SLURMD 服务..."
systemctl stop slurmd 2>/dev/null || true
systemctl enable slurmd 2>/dev/null || true
systemctl start slurmd || slurmd -D &

sleep 2

if pgrep -x slurmd >/dev/null; then
    echo "[成功] SLURMD 服务运行正常"
else
    echo "[错误] SLURMD 服务未运行"
    exit 1
fi
`
	}

	return s.executeSSHCmd(client, cmd)
}

// executeSSHCmd 执行 SSH 命令的辅助函数
func (s *SlurmClusterService) executeSSHCmd(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("命令执行失败: %v, 输出: %s", err, string(output))
	}

	log.Printf("[输出] %s", string(output))
	return nil
}

// getMungeKeyFromMaster 从 SLURM Master 获取 munge.key
func (s *SlurmClusterService) getMungeKeyFromMaster() ([]byte, error) {
	// 尝试从 Docker 容器读取
	config := RemoteNodeInitConfig{
		Host:     os.Getenv("SLURM_MASTER_HOST"),
		Port:     22,
		Username: "root",
		Password: os.Getenv("SLURM_MASTER_PASSWORD"),
	}

	if config.Host == "" {
		config.Host = "ai-infra-slurm-master"
	}
	if config.Password == "" {
		config.Password = "rootpass123" // 默认密码
	}

	client, err := s.createSSHClient(config)
	if err != nil {
		return nil, fmt.Errorf("连接 SLURM Master 失败: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput("cat /etc/munge/munge.key")
	if err != nil {
		return nil, fmt.Errorf("读取 munge.key 失败: %v", err)
	}

	return output, nil
}

// getSlurmConfFromMaster 从 SLURM Master 获取 slurm.conf
func (s *SlurmClusterService) getSlurmConfFromMaster() ([]byte, error) {
	config := RemoteNodeInitConfig{
		Host:     os.Getenv("SLURM_MASTER_HOST"),
		Port:     22,
		Username: "root",
		Password: os.Getenv("SLURM_MASTER_PASSWORD"),
	}

	if config.Host == "" {
		config.Host = "ai-infra-slurm-master"
	}
	if config.Password == "" {
		config.Password = "rootpass123"
	}

	client, err := s.createSSHClient(config)
	if err != nil {
		return nil, fmt.Errorf("连接 SLURM Master 失败: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput("cat /etc/slurm/slurm.conf")
	if err != nil {
		return nil, fmt.Errorf("读取 slurm.conf 失败: %v", err)
	}

	return output, nil
}

// uploadFileViaSSH 通过 SSH/SFTP 上传文件
func (s *SlurmClusterService) uploadFileViaSSH(client *ssh.Client, localPath, remotePath string) error {
	// 读取本地文件
	content, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}

	// 使用 SSH 写入远程文件
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 使用 cat 命令写入文件
	cmd := fmt.Sprintf("cat > %s <<'EOF'\n%s\nEOF", remotePath, string(content))
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("写入远程文件失败: %v, 输出: %s", err, string(output))
	}

	return nil
}
