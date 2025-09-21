package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// SaltStackClientService 负责SaltStack客户端的远程安装和管理
type SaltStackClientService struct {
	sshService    *SSHService
	appHubService *AppHubService
	installTasks  map[string]*InstallTask
	taskMutex     sync.RWMutex
}

// InstallTask 表示一个安装任务
type InstallTask struct {
	ID       string                  `json:"id"`
	Host     string                  `json:"host"`
	Status   string                  `json:"status"` // pending, running, success, failed
	Progress int                     `json:"progress"`
	Message  string                  `json:"message"`
	StartAt  time.Time               `json:"start_at"`
	EndAt    *time.Time              `json:"end_at,omitempty"`
	Steps    []InstallStep           `json:"steps"`
	Config   SaltMinionInstallConfig `json:"config"`
}

type InstallStep struct {
	Name    string     `json:"name"`
	Status  string     `json:"status"`
	Message string     `json:"message"`
	StartAt time.Time  `json:"start_at"`
	EndAt   *time.Time `json:"end_at,omitempty"`
}

// SaltMinionInstallConfig SaltStack Minion安装配置
type SaltMinionInstallConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	KeyPath     string `json:"key_path,omitempty"`
	MasterHost  string `json:"master_host"`
	MinionID    string `json:"minion_id"`
	InstallType string `json:"install_type"` // binary, package
	Version     string `json:"version"`
	AutoAccept  bool   `json:"auto_accept"`
}

// InstallRequest 安装请求
type InstallRequest struct {
	Hosts    []SaltMinionInstallConfig `json:"hosts"`
	Parallel int                       `json:"parallel"` // 并发安装数量
}

// AppHubBinary AppHub中的二进制包信息
type AppHubBinary struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Platform    string `json:"platform"`
	Arch        string `json:"arch"`
	DownloadURL string `json:"download_url"`
	CheckSum    string `json:"checksum"`
	Size        int64  `json:"size"`
}

func NewSaltStackClientService() *SaltStackClientService {
	return &SaltStackClientService{
		sshService:    NewSSHService(),
		appHubService: NewAppHubService(),
		installTasks:  make(map[string]*InstallTask),
	}
}

// InstallSaltMinionAsync 异步安装SaltStack Minion客户端
func (s *SaltStackClientService) InstallSaltMinionAsync(ctx context.Context, req InstallRequest) ([]string, error) {
	var taskIDs []string

	for _, config := range req.Hosts {
		taskID := fmt.Sprintf("salt-install-%s-%d", config.Host, time.Now().Unix())

		task := &InstallTask{
			ID:       taskID,
			Host:     config.Host,
			Status:   "pending",
			Progress: 0,
			Message:  "Installation queued",
			StartAt:  time.Now(),
			Config:   config,
			Steps:    []InstallStep{},
		}

		s.taskMutex.Lock()
		s.installTasks[taskID] = task
		s.taskMutex.Unlock()

		// 启动异步安装
		go s.runInstallTask(ctx, task)

		taskIDs = append(taskIDs, taskID)

		logrus.WithFields(logrus.Fields{
			"task_id": taskID,
			"host":    config.Host,
		}).Info("SaltStack Minion installation task created")
	}

	return taskIDs, nil
}

// runInstallTask 执行安装任务
func (s *SaltStackClientService) runInstallTask(ctx context.Context, task *InstallTask) {
	defer func() {
		if r := recover(); r != nil {
			task.Status = "failed"
			task.Message = fmt.Sprintf("Installation failed with panic: %v", r)
			now := time.Now()
			task.EndAt = &now
			logrus.WithFields(logrus.Fields{
				"task_id": task.ID,
				"host":    task.Host,
				"error":   r,
			}).Error("SaltStack Minion installation task panicked")
		}
	}()

	task.Status = "running"
	task.Message = "Starting installation"
	task.Progress = 10

	logrus.WithFields(logrus.Fields{
		"task_id": task.ID,
		"host":    task.Host,
	}).Info("Starting SaltStack Minion installation")

	// 步骤1: 建立SSH连接
	if err := s.addStep(task, "connect", "Connecting to host"); err != nil {
		s.failTask(task, fmt.Sprintf("Failed to add connect step: %v", err))
		return
	}

	sshConfig := &ssh.ClientConfig{
		User: task.Config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(task.Config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	if task.Config.KeyPath != "" {
		// TODO: 支持SSH密钥认证
		logrus.WithField("key_path", task.Config.KeyPath).Debug("SSH key authentication not implemented yet")
	}

	addr := fmt.Sprintf("%s:%d", task.Config.Host, task.Config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		s.failStep(task, "connect", fmt.Sprintf("SSH connection failed: %v", err))
		s.failTask(task, "SSH connection failed")
		return
	}
	defer client.Close()

	s.completeStep(task, "connect", "SSH connection established")
	task.Progress = 20

	// 步骤2: 检测系统信息
	if err := s.addStep(task, "detect", "Detecting system information"); err != nil {
		s.failTask(task, fmt.Sprintf("Failed to add detect step: %v", err))
		return
	}

	osInfo, err := s.detectOSInfo(client)
	if err != nil {
		s.failStep(task, "detect", fmt.Sprintf("OS detection failed: %v", err))
		s.failTask(task, "System detection failed")
		return
	}

	s.completeStep(task, "detect", fmt.Sprintf("Detected OS: %s %s %s", osInfo.OS, osInfo.Version, osInfo.Arch))
	task.Progress = 30

	// 步骤3: 获取二进制包
	if err := s.addStep(task, "download", "Downloading SaltStack binary"); err != nil {
		s.failTask(task, fmt.Sprintf("Failed to add download step: %v", err))
		return
	}

	binary, err := s.getSaltStackBinary(osInfo, task.Config.Version)
	if err != nil {
		s.failStep(task, "download", fmt.Sprintf("Failed to get binary: %v", err))
		s.failTask(task, "Failed to get SaltStack binary")
		return
	}

	s.completeStep(task, "download", fmt.Sprintf("Binary located: %s", binary.Name))
	task.Progress = 50

	// 步骤4: 安装SaltStack
	if err := s.addStep(task, "install", "Installing SaltStack Minion"); err != nil {
		s.failTask(task, fmt.Sprintf("Failed to add install step: %v", err))
		return
	}

	if err := s.installSaltStackMinion(client, binary, task.Config); err != nil {
		s.failStep(task, "install", fmt.Sprintf("Installation failed: %v", err))
		s.failTask(task, "SaltStack installation failed")
		return
	}

	s.completeStep(task, "install", "SaltStack Minion installed successfully")
	task.Progress = 80

	// 步骤5: 配置和启动服务
	if err := s.addStep(task, "configure", "Configuring and starting service"); err != nil {
		s.failTask(task, fmt.Sprintf("Failed to add configure step: %v", err))
		return
	}

	if err := s.configureSaltMinion(client, task.Config); err != nil {
		s.failStep(task, "configure", fmt.Sprintf("Configuration failed: %v", err))
		s.failTask(task, "Service configuration failed")
		return
	}

	s.completeStep(task, "configure", "Service configured and started")
	task.Progress = 100

	// 完成任务
	task.Status = "success"
	task.Message = "SaltStack Minion installed successfully"
	now := time.Now()
	task.EndAt = &now

	logrus.WithFields(logrus.Fields{
		"task_id": task.ID,
		"host":    task.Host,
	}).Info("SaltStack Minion installation completed successfully")
}

// detectOSInfo 检测操作系统信息
func (s *SaltStackClientService) detectOSInfo(client *ssh.Client) (*OSInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// 检测操作系统
	var output bytes.Buffer
	session.Stdout = &output

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

	if err := session.Run(cmd); err != nil {
		return nil, fmt.Errorf("failed to run detection command: %v", err)
	}

	osInfo := &OSInfo{}
	lines := strings.Split(output.String(), "\n")
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

// getSaltStackBinary 从AppHub获取SaltStack二进制包
func (s *SaltStackClientService) getSaltStackBinary(osInfo *OSInfo, version string) (*AppHubBinary, error) {
	// 这里应该调用AppHub服务获取二进制包
	// 暂时返回模拟数据
	return &AppHubBinary{
		Name:        "salt-minion",
		Version:     version,
		Platform:    osInfo.OS,
		Arch:        osInfo.Arch,
		DownloadURL: fmt.Sprintf("https://apphub.example.com/salt-minion/%s/%s/%s", version, osInfo.OS, osInfo.Arch),
		CheckSum:    "sha256:example",
		Size:        1024 * 1024 * 50, // 50MB
	}, nil
}

// installSaltStackMinion 安装SaltStack Minion
func (s *SaltStackClientService) installSaltStackMinion(client *ssh.Client, binary *AppHubBinary, config SaltMinionInstallConfig) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// 根据安装类型执行不同的安装命令
	var installCmd string

	switch config.InstallType {
	case "binary":
		installCmd = fmt.Sprintf(`
			set -e
			cd /tmp
			curl -L -o salt-minion.tar.gz "%s"
			tar -xzf salt-minion.tar.gz
			sudo mv salt-minion /usr/local/bin/
			sudo chmod +x /usr/local/bin/salt-minion
			sudo mkdir -p /etc/salt
		`, binary.DownloadURL)
	case "package":
		// 根据操作系统使用包管理器安装
		switch binary.Platform {
		case "ubuntu", "debian":
			installCmd = `
				set -e
				curl -fsSL https://repo.saltproject.io/py3/ubuntu/20.04/amd64/latest/salt-archive-keyring.gpg | sudo apt-key add -
				echo "deb https://repo.saltproject.io/py3/ubuntu/20.04/amd64/latest focal main" | sudo tee /etc/apt/sources.list.d/salt.list
				sudo apt-get update
				sudo apt-get install -y salt-minion
			`
		case "centos", "rhel":
			installCmd = `
				set -e
				sudo yum install -y https://repo.saltproject.io/py3/redhat/salt-py3-repo-latest.el8.noarch.rpm
				sudo yum install -y salt-minion
			`
		default:
			return fmt.Errorf("unsupported platform: %s", binary.Platform)
		}
	default:
		return fmt.Errorf("unsupported install type: %s", config.InstallType)
	}

	if err := session.Run(installCmd); err != nil {
		return fmt.Errorf("installation command failed: %v", err)
	}

	return nil
}

// configureSaltMinion 配置Salt Minion
func (s *SaltStackClientService) configureSaltMinion(client *ssh.Client, config SaltMinionInstallConfig) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// 创建Salt Minion配置文件
	configContent := fmt.Sprintf(`
master: %s
id: %s
mine_enabled: true
mine_return_job: true
mine_interval: 60
`, config.MasterHost, config.MinionID)

	configCmd := fmt.Sprintf(`
		set -e
		sudo mkdir -p /etc/salt
		echo '%s' | sudo tee /etc/salt/minion
		sudo systemctl enable salt-minion
		sudo systemctl start salt-minion
		sudo systemctl status salt-minion
	`, configContent)

	if err := session.Run(configCmd); err != nil {
		return fmt.Errorf("configuration command failed: %v", err)
	}

	return nil
}

// GetInstallTask 获取安装任务状态
func (s *SaltStackClientService) GetInstallTask(taskID string) (*InstallTask, error) {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	task, exists := s.installTasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// ListInstallTasks 列出所有安装任务
func (s *SaltStackClientService) ListInstallTasks() []*InstallTask {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	tasks := make([]*InstallTask, 0, len(s.installTasks))
	for _, task := range s.installTasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// 辅助方法
func (s *SaltStackClientService) addStep(task *InstallTask, name, message string) error {
	step := InstallStep{
		Name:    name,
		Status:  "running",
		Message: message,
		StartAt: time.Now(),
	}
	task.Steps = append(task.Steps, step)
	return nil
}

func (s *SaltStackClientService) completeStep(task *InstallTask, name, message string) {
	for i := range task.Steps {
		if task.Steps[i].Name == name {
			task.Steps[i].Status = "success"
			task.Steps[i].Message = message
			now := time.Now()
			task.Steps[i].EndAt = &now
			break
		}
	}
}

func (s *SaltStackClientService) failStep(task *InstallTask, name, message string) {
	for i := range task.Steps {
		if task.Steps[i].Name == name {
			task.Steps[i].Status = "failed"
			task.Steps[i].Message = message
			now := time.Now()
			task.Steps[i].EndAt = &now
			break
		}
	}
}

func (s *SaltStackClientService) failTask(task *InstallTask, message string) {
	task.Status = "failed"
	task.Message = message
	now := time.Now()
	task.EndAt = &now
}

// AppHubService AppHub服务接口
type AppHubService struct {
	baseURL string
	client  *http.Client
}

func NewAppHubService() *AppHubService {
	return &AppHubService{
		baseURL: "http://apphub:8080", // 假设AppHub服务地址
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAvailableBinaries 获取可用的二进制包列表
func (a *AppHubService) GetAvailableBinaries(name, platform, arch string) ([]AppHubBinary, error) {
	url := fmt.Sprintf("%s/api/binaries?name=%s&platform=%s&arch=%s", a.baseURL, name, platform, arch)

	resp, err := a.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to request AppHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AppHub returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var binaries []AppHubBinary
	if err := json.Unmarshal(body, &binaries); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return binaries, nil
}
