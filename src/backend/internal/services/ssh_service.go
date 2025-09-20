package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
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
	DefaultUser     string
	DefaultKeyPath  string
	ConnectTimeout  time.Duration
	CommandTimeout  time.Duration
	MaxConcurrency  int
}

// SSHConnection SSH连接信息
type SSHConnection struct {
	Host     string
	Port     int
	User     string
	KeyPath  string
	Password string
}

// DeploymentResult 部署结果
type DeploymentResult struct {
	Host    string
	Success bool
	Output  string
	Error   string
	Duration time.Duration
}

// SaltStackDeploymentConfig SaltStack部署配置
type SaltStackDeploymentConfig struct {
	MasterHost string
	MasterPort int
	MinionID   string
	AutoAccept bool
}

// NewSSHService 创建新的SSH服务
func NewSSHService() *SSHService {
	return &SSHService{
		config: &SSHConfig{
			DefaultUser:     "root",
			DefaultKeyPath:  "/root/.ssh/id_rsa",
			ConnectTimeout:  10 * time.Second,
			CommandTimeout:  30 * time.Second,
			MaxConcurrency:  10,
		},
	}
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
		User: conn.User,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         s.config.ConnectTimeout,
	}

	// 添加认证方法
	if conn.KeyPath != "" {
		key, err := s.loadPrivateKey(conn.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("加载私钥失败: %v", err)
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

	return fmt.Sprintf(`
cat > /etc/salt/minion << EOF
master: %s
id: %s
log_level: info
EOF

# 如果需要自动接受密钥
if [ "%t" = "true" ]; then
    echo "auto_accept: True" >> /etc/salt/minion
fi
`,
		net.JoinHostPort(config.MasterHost, fmt.Sprintf("%d", config.MasterPort)),
		minionID,
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