package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// BatchInstallService 批量安装服务
type BatchInstallService struct {
	sseChannels   map[string]chan SSEEvent
	channelsMutex sync.RWMutex
}

// SSEEvent SSE事件
type SSEEvent struct {
	Type      string      `json:"type"` // progress, log, complete, error
	Host      string      `json:"host,omitempty"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"ts"`
}

// BatchInstallRequest 批量安装请求
type BatchInstallRequest struct {
	Hosts       []HostInstallConfig `json:"hosts"`
	Parallel    int                 `json:"parallel"`     // 并发数，默认3
	MasterHost  string              `json:"master_host"`  // Salt Master 地址
	InstallType string              `json:"install_type"` // saltstack, slurm
	UseSudo     bool                `json:"use_sudo"`     // 是否使用 sudo
	SudoPass    string              `json:"sudo_pass"`    // sudo 密码（如果不同于登录密码）
	AutoAccept  bool                `json:"auto_accept"`  // 自动接受 minion key
	Version     string              `json:"version"`      // 安装版本
}

// HostInstallConfig 单主机安装配置
type HostInstallConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	KeyPath  string `json:"key_path,omitempty"`
	MinionID string `json:"minion_id,omitempty"` // 可选，默认使用 hostname
	UseSudo  bool   `json:"use_sudo,omitempty"`  // 覆盖全局设置
	SudoPass string `json:"sudo_pass,omitempty"` // 覆盖全局设置
}

// BatchInstallResult 批量安装结果
type BatchInstallResult struct {
	TaskID       string              `json:"task_id"`
	TotalHosts   int                 `json:"total_hosts"`
	SuccessHosts int                 `json:"success_hosts"`
	FailedHosts  int                 `json:"failed_hosts"`
	Duration     int64               `json:"duration"` // 总耗时（毫秒）
	HostResults  []HostInstallResult `json:"host_results"`
}

// HostInstallResult 单主机安装结果
type HostInstallResult struct {
	Host     string `json:"host"`
	Status   string `json:"status"` // success, failed
	Message  string `json:"message"`
	Duration int64  `json:"duration"` // 耗时（毫秒）
	Error    string `json:"error,omitempty"`
}

// SSHTestRequest SSH 测试请求
type SSHTestRequest struct {
	Hosts    []HostInstallConfig `json:"hosts"`
	Parallel int                 `json:"parallel"` // 并发数
}

// SSHTestResult SSH 测试结果
type SSHTestResult struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Connected      bool   `json:"connected"`        // SSH 是否连接成功
	AuthMethod     string `json:"auth_method"`      // 认证方式: password, key
	HasSudo        bool   `json:"has_sudo"`         // 是否有 sudo 权限
	SudoNoPassword bool   `json:"sudo_no_password"` // sudo 是否需要密码
	OSInfo         string `json:"os_info"`          // 操作系统信息
	Hostname       string `json:"hostname"`         // 主机名
	Error          string `json:"error,omitempty"`  // 错误信息
	Duration       int64  `json:"duration"`         // 测试耗时（毫秒）
}

// NewBatchInstallService 创建批量安装服务
func NewBatchInstallService() *BatchInstallService {
	return &BatchInstallService{
		sseChannels: make(map[string]chan SSEEvent),
	}
}

// GetSSEChannel 获取或创建 SSE 通道
func (s *BatchInstallService) GetSSEChannel(taskID string) chan SSEEvent {
	s.channelsMutex.Lock()
	defer s.channelsMutex.Unlock()

	if ch, exists := s.sseChannels[taskID]; exists {
		return ch
	}

	ch := make(chan SSEEvent, 100)
	s.sseChannels[taskID] = ch
	return ch
}

// CloseSSEChannel 关闭 SSE 通道
func (s *BatchInstallService) CloseSSEChannel(taskID string) {
	s.channelsMutex.Lock()
	defer s.channelsMutex.Unlock()

	if ch, exists := s.sseChannels[taskID]; exists {
		close(ch)
		delete(s.sseChannels, taskID)
	}
}

// sendEvent 发送 SSE 事件
func (s *BatchInstallService) sendEvent(taskID string, event SSEEvent) {
	s.channelsMutex.RLock()
	ch, exists := s.sseChannels[taskID]
	s.channelsMutex.RUnlock()

	if exists {
		event.Timestamp = time.Now()
		select {
		case ch <- event:
		default:
			// 通道已满，丢弃事件
			logrus.WithField("task_id", taskID).Warn("SSE channel full, dropping event")
		}
	}
}

// BatchInstallSaltMinion 批量安装 Salt Minion
func (s *BatchInstallService) BatchInstallSaltMinion(ctx context.Context, req BatchInstallRequest, userID uint) (string, error) {
	// 生成任务ID
	taskID := fmt.Sprintf("batch-salt-%s", uuid.New().String()[:8])

	// 设置默认值
	if req.Parallel <= 0 {
		req.Parallel = 3
	}
	if req.MasterHost == "" {
		// 尝试从环境变量获取
		req.MasterHost = os.Getenv("EXTERNAL_HOST")
		if req.MasterHost == "" {
			req.MasterHost = "saltstack"
		}
	}
	if req.Version == "" {
		req.Version = os.Getenv("SALTSTACK_VERSION")
		if req.Version == "" {
			req.Version = "3007.8"
		}
	}

	// 创建数据库任务记录
	task := &models.InstallationTask{
		TaskName:   fmt.Sprintf("Batch Salt Minion Install - %s", taskID),
		TaskType:   "saltstack",
		Status:     "running",
		TotalHosts: len(req.Hosts),
		StartTime:  time.Now(),
	}
	task.SetConfig(req)

	if database.DB != nil {
		if err := database.DB.Create(task).Error; err != nil {
			logrus.WithError(err).Error("Failed to create installation task in database")
		}
	}

	// 创建 SSE 通道
	s.GetSSEChannel(taskID)

	// 启动异步安装
	go s.runBatchInstall(ctx, taskID, task, req, userID)

	logrus.WithFields(logrus.Fields{
		"task_id":     taskID,
		"total_hosts": len(req.Hosts),
		"parallel":    req.Parallel,
	}).Info("Batch Salt Minion installation started")

	return taskID, nil
}

// runBatchInstall 执行批量安装
func (s *BatchInstallService) runBatchInstall(ctx context.Context, taskID string, task *models.InstallationTask, req BatchInstallRequest, userID uint) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"task_id": taskID,
				"panic":   r,
			}).Error("Batch install panicked")
			s.sendEvent(taskID, SSEEvent{
				Type:    "error",
				Message: fmt.Sprintf("Installation failed with panic: %v", r),
			})
		}
		// 延迟关闭 SSE 通道，让客户端有时间接收最后的消息
		time.Sleep(2 * time.Second)
		s.CloseSSEChannel(taskID)
	}()

	s.sendEvent(taskID, SSEEvent{
		Type:    "progress",
		Message: fmt.Sprintf("Starting batch installation for %d hosts with %d parallel workers", len(req.Hosts), req.Parallel),
	})

	// 创建工作通道
	jobs := make(chan HostInstallConfig, len(req.Hosts))
	results := make(chan HostInstallResult, len(req.Hosts))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < req.Parallel; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for hostConfig := range jobs {
				result := s.installSingleHost(ctx, taskID, hostConfig, req)
				results <- result
			}
		}(i)
	}

	// 发送任务
	for _, host := range req.Hosts {
		// 继承全局设置
		if !host.UseSudo && req.UseSudo {
			host.UseSudo = req.UseSudo
		}
		if host.SudoPass == "" && req.SudoPass != "" {
			host.SudoPass = req.SudoPass
		}
		if host.Port == 0 {
			host.Port = 22
		}
		jobs <- host
	}
	close(jobs)

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var hostResults []HostInstallResult
	successCount := 0
	failedCount := 0

	for result := range results {
		hostResults = append(hostResults, result)
		if result.Status == "success" {
			successCount++
		} else {
			failedCount++
		}

		// 保存主机结果到数据库
		if database.DB != nil && task.ID > 0 {
			hostResult := &models.InstallationHostResult{
				TaskID:   task.ID,
				Host:     result.Host,
				Status:   result.Status,
				Error:    result.Error,
				Duration: result.Duration,
				Output:   result.Message,
			}
			database.DB.Create(hostResult)
		}

		// 发送进度事件
		s.sendEvent(taskID, SSEEvent{
			Type:    "progress",
			Host:    result.Host,
			Message: fmt.Sprintf("Completed: %d/%d (Success: %d, Failed: %d)", successCount+failedCount, len(req.Hosts), successCount, failedCount),
			Data: map[string]interface{}{
				"completed": successCount + failedCount,
				"total":     len(req.Hosts),
				"success":   successCount,
				"failed":    failedCount,
			},
		})
	}

	// 更新任务状态
	if database.DB != nil && task.ID > 0 {
		task.SuccessHosts = successCount
		task.FailedHosts = failedCount
		task.UpdateHostStats()
		database.DB.Save(task)
	}

	// 发送完成事件
	s.sendEvent(taskID, SSEEvent{
		Type:    "complete",
		Message: fmt.Sprintf("Batch installation completed. Success: %d, Failed: %d", successCount, failedCount),
		Data: BatchInstallResult{
			TaskID:       taskID,
			TotalHosts:   len(req.Hosts),
			SuccessHosts: successCount,
			FailedHosts:  failedCount,
			HostResults:  hostResults,
		},
	})

	logrus.WithFields(logrus.Fields{
		"task_id": taskID,
		"total":   len(req.Hosts),
		"success": successCount,
		"failed":  failedCount,
	}).Info("Batch Salt Minion installation completed")
}

// installSingleHost 安装单个主机
func (s *BatchInstallService) installSingleHost(ctx context.Context, taskID string, hostConfig HostInstallConfig, req BatchInstallRequest) HostInstallResult {
	startTime := time.Now()
	result := HostInstallResult{
		Host:   hostConfig.Host,
		Status: "failed",
	}

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    hostConfig.Host,
		Message: fmt.Sprintf("Starting installation on %s", hostConfig.Host),
	})

	// 记录日志到数据库
	s.logToDatabase(taskID, "info", hostConfig.Host, fmt.Sprintf("Starting Salt Minion installation on %s:%d", hostConfig.Host, hostConfig.Port))

	// 建立 SSH 连接
	client, err := s.connectSSH(hostConfig)
	if err != nil {
		result.Error = fmt.Sprintf("SSH connection failed: %v", err)
		result.Message = result.Error
		result.Duration = time.Since(startTime).Milliseconds()
		s.sendEvent(taskID, SSEEvent{
			Type:    "error",
			Host:    hostConfig.Host,
			Message: result.Error,
		})
		s.logToDatabase(taskID, "error", hostConfig.Host, result.Error)
		return result
	}
	defer client.Close()

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    hostConfig.Host,
		Message: "SSH connection established",
	})

	// 检测操作系统
	osInfo, err := s.detectOS(client)
	if err != nil {
		result.Error = fmt.Sprintf("OS detection failed: %v", err)
		result.Message = result.Error
		result.Duration = time.Since(startTime).Milliseconds()
		s.sendEvent(taskID, SSEEvent{
			Type:    "error",
			Host:    hostConfig.Host,
			Message: result.Error,
		})
		s.logToDatabase(taskID, "error", hostConfig.Host, result.Error)
		return result
	}

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    hostConfig.Host,
		Message: fmt.Sprintf("Detected OS: %s %s (%s)", osInfo.OS, osInfo.Version, osInfo.Arch),
	})

	// 确定是否使用 sudo
	useSudo := hostConfig.UseSudo || req.UseSudo
	sudoPass := hostConfig.SudoPass
	if sudoPass == "" {
		sudoPass = req.SudoPass
	}
	if sudoPass == "" {
		sudoPass = hostConfig.Password // 默认使用登录密码
	}

	// 构建安装命令
	sudoPrefix := ""
	if useSudo && hostConfig.Username != "root" {
		sudoPrefix = fmt.Sprintf("echo '%s' | sudo -S ", sudoPass)
	}

	// 获取 minion ID
	minionID := hostConfig.MinionID
	if minionID == "" {
		minionID = hostConfig.Host
	}

	// 安装 Salt Minion
	err = s.installSaltMinion(client, osInfo, req.MasterHost, minionID, req.Version, sudoPrefix, taskID, hostConfig.Host)
	if err != nil {
		result.Error = fmt.Sprintf("Installation failed: %v", err)
		result.Message = result.Error
		result.Duration = time.Since(startTime).Milliseconds()
		s.sendEvent(taskID, SSEEvent{
			Type:    "error",
			Host:    hostConfig.Host,
			Message: result.Error,
		})
		s.logToDatabase(taskID, "error", hostConfig.Host, result.Error)
		return result
	}

	result.Status = "success"
	result.Message = "Salt Minion installed and started successfully"
	result.Duration = time.Since(startTime).Milliseconds()

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    hostConfig.Host,
		Message: fmt.Sprintf("Installation completed successfully in %.2fs", float64(result.Duration)/1000),
	})
	s.logToDatabase(taskID, "info", hostConfig.Host, result.Message)

	return result
}

// connectSSH 建立 SSH 连接
func (s *BatchInstallService) connectSSH(config HostInstallConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// 密码认证
	if config.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	// 密钥认证
	if config.KeyPath != "" {
		key, err := os.ReadFile(config.KeyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available")
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	return ssh.Dial("tcp", addr, sshConfig)
}

// detectOS 检测操作系统
func (s *BatchInstallService) detectOS(client *ssh.Client) (*models.OSInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

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

	osInfo := &models.OSInfo{}
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

// installSaltMinion 安装 Salt Minion
func (s *BatchInstallService) installSaltMinion(client *ssh.Client, osInfo *models.OSInfo, masterHost, minionID, version, sudoPrefix, taskID, host string) error {
	// 构建 AppHub URL
	appHubHost := os.Getenv("EXTERNAL_HOST")
	if appHubHost == "" {
		appHubHost = "localhost"
	}
	appHubPort := os.Getenv("APPHUB_PORT")
	if appHubPort == "" {
		appHubPort = "28080"
	}
	appHubURL := fmt.Sprintf("http://%s:%s", appHubHost, appHubPort)

	// 获取架构
	arch := osInfo.Arch
	if arch == "x86_64" {
		arch = "amd64"
	} else if arch == "aarch64" {
		arch = "arm64"
	}

	// 根据操作系统构建安装命令
	var installCmd string

	switch osInfo.OS {
	case "ubuntu", "debian":
		installCmd = fmt.Sprintf(`
set -e
echo "=== Starting Salt Minion Installation ==="

# Create temp directory
cd /tmp
rm -rf salt-install && mkdir -p salt-install && cd salt-install

echo "=== Downloading Salt packages from AppHub ==="
# Download salt packages
curl -fsSL "%s/pkgs/saltstack-deb/salt-common_%s_%s.deb" -o salt-common.deb || { echo "Failed to download salt-common"; exit 1; }
curl -fsSL "%s/pkgs/saltstack-deb/salt-minion_%s_%s.deb" -o salt-minion.deb || { echo "Failed to download salt-minion"; exit 1; }

echo "=== Installing dependencies ==="
%sapt-get update -qq || true
%sapt-get install -y -qq python3 python3-pip python3-setuptools || true

echo "=== Installing Salt packages ==="
%sdpkg -i salt-common.deb || %sapt-get install -f -y -qq
%sdpkg -i salt-minion.deb || %sapt-get install -f -y -qq

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
cd /tmp && rm -rf salt-install

echo "=== Salt Minion Installation Complete ==="
`, appHubURL, version, arch,
			appHubURL, version, arch,
			sudoPrefix, sudoPrefix,
			sudoPrefix, sudoPrefix,
			sudoPrefix, sudoPrefix,
			sudoPrefix, sudoPrefix,
			masterHost, minionID,
			sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix)

	case "centos", "rhel", "rocky", "almalinux", "fedora":
		rpmArch := osInfo.Arch
		if rpmArch == "amd64" {
			rpmArch = "x86_64"
		} else if rpmArch == "arm64" {
			rpmArch = "aarch64"
		}

		installCmd = fmt.Sprintf(`
set -e
echo "=== Starting Salt Minion Installation ==="

# Create temp directory
cd /tmp
rm -rf salt-install && mkdir -p salt-install && cd salt-install

echo "=== Downloading Salt packages from AppHub ==="
# Download salt packages
curl -fsSL "%s/pkgs/saltstack-rpm/salt-%s-0.%s.rpm" -o salt.rpm || { echo "Failed to download salt"; exit 1; }
curl -fsSL "%s/pkgs/saltstack-rpm/salt-minion-%s-0.%s.rpm" -o salt-minion.rpm || { echo "Failed to download salt-minion"; exit 1; }

echo "=== Installing dependencies ==="
%syum install -y python3 python3-pip || %sdnf install -y python3 python3-pip || true

echo "=== Installing Salt packages ==="
%srpm -Uvh --replacepkgs salt.rpm || %syum localinstall -y salt.rpm || true
%srpm -Uvh --replacepkgs salt-minion.rpm || %syum localinstall -y salt-minion.rpm || true

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
cd /tmp && rm -rf salt-install

echo "=== Salt Minion Installation Complete ==="
`, appHubURL, version, rpmArch,
			appHubURL, version, rpmArch,
			sudoPrefix, sudoPrefix,
			sudoPrefix, sudoPrefix,
			sudoPrefix, sudoPrefix,
			sudoPrefix, sudoPrefix,
			masterHost, minionID,
			sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix)

	default:
		return fmt.Errorf("unsupported operating system: %s", osInfo.OS)
	}

	// 执行安装命令
	return s.runCommand(client, installCmd, taskID, host)
}

// runCommand 执行 SSH 命令并记录输出
func (s *BatchInstallService) runCommand(client *ssh.Client, cmd, taskID, host string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// 创建管道读取输出
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// 启动命令
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	// 读取输出并发送 SSE 事件
	go func() {
		s.streamOutput(stdout, taskID, host, "stdout")
	}()
	go func() {
		s.streamOutput(stderr, taskID, host, "stderr")
	}()

	// 等待命令完成
	if err := session.Wait(); err != nil {
		return fmt.Errorf("command failed: %v", err)
	}

	return nil
}

// streamOutput 流式输出
func (s *BatchInstallService) streamOutput(reader io.Reader, taskID, host, streamType string) {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			output := string(buf[:n])
			// 按行发送
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					s.sendEvent(taskID, SSEEvent{
						Type:    "log",
						Host:    host,
						Message: line,
						Data: map[string]string{
							"stream": streamType,
						},
					})
				}
			}
		}
		if err != nil {
			break
		}
	}
}

// logToDatabase 记录日志到数据库
func (s *BatchInstallService) logToDatabase(taskID, level, host, message string) {
	if database.DB == nil {
		return
	}

	log := &models.TaskLog{
		TaskID:    taskID,
		LogLevel:  level,
		Message:   fmt.Sprintf("[%s] %s", host, message),
		Timestamp: time.Now(),
	}

	if err := database.DB.Create(log).Error; err != nil {
		logrus.WithError(err).Warn("Failed to save task log to database")
	}
}

// GetTaskLogs 获取任务日志
func (s *BatchInstallService) GetTaskLogs(taskID string) ([]models.TaskLog, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var logs []models.TaskLog
	err := database.DB.Where("task_id = ?", taskID).Order("timestamp ASC").Find(&logs).Error
	return logs, err
}

// GetTask 获取任务详情
func (s *BatchInstallService) GetTask(taskID string) (*models.InstallationTask, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var task models.InstallationTask
	err := database.DB.Preload("HostResults").Where("task_name LIKE ?", "%"+taskID+"%").First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks 列出任务
func (s *BatchInstallService) ListTasks(taskType string, limit, offset int) ([]models.InstallationTask, int64, error) {
	if database.DB == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	var tasks []models.InstallationTask
	var total int64

	query := database.DB.Model(&models.InstallationTask{})
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}

	query.Count(&total)
	err := query.Preload("HostResults").Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks).Error

	return tasks, total, err
}

// MarshalSSEEvent 将 SSE 事件序列化为 JSON
func MarshalSSEEvent(event SSEEvent) ([]byte, error) {
	return json.Marshal(event)
}

// TestSSHConnection 测试单个 SSH 连接（包含 sudo 权限检查）
func (s *BatchInstallService) TestSSHConnection(ctx context.Context, config HostInstallConfig) SSHTestResult {
	startTime := time.Now()
	result := SSHTestResult{
		Host:     config.Host,
		Port:     config.Port,
		Username: config.Username,
	}

	// 建立 SSH 连接
	client, err := s.connectSSH(config)
	if err != nil {
		result.Error = fmt.Sprintf("SSH connection failed: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}
	defer client.Close()

	result.Connected = true

	// 检测认证方式
	if config.Password != "" {
		result.AuthMethod = "password"
	} else if config.KeyPath != "" {
		result.AuthMethod = "key"
	}

	// 获取主机名和操作系统信息
	session, err := client.NewSession()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create session: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	var output bytes.Buffer
	session.Stdout = &output
	cmd := `hostname && uname -a`
	if err := session.Run(cmd); err == nil {
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) >= 1 {
			result.Hostname = strings.TrimSpace(lines[0])
		}
		if len(lines) >= 2 {
			result.OSInfo = strings.TrimSpace(lines[1])
		}
	}
	session.Close()

	// 检查 sudo 权限
	if config.Username != "root" {
		// 首先尝试不需要密码的 sudo
		session2, err := client.NewSession()
		if err == nil {
			var sudoOutput bytes.Buffer
			session2.Stdout = &sudoOutput
			session2.Stderr = &sudoOutput

			// 使用 sudo -n 测试不需要密码的 sudo
			if err := session2.Run("sudo -n whoami 2>/dev/null"); err == nil {
				output := strings.TrimSpace(sudoOutput.String())
				if output == "root" {
					result.HasSudo = true
					result.SudoNoPassword = true
				}
			}
			session2.Close()
		}

		// 如果不需要密码的 sudo 失败，尝试带密码的 sudo
		if !result.HasSudo && config.Password != "" {
			session3, err := client.NewSession()
			if err == nil {
				var sudoOutput2 bytes.Buffer
				session3.Stdout = &sudoOutput2
				session3.Stderr = &sudoOutput2

				sudoPass := config.SudoPass
				if sudoPass == "" {
					sudoPass = config.Password
				}

				// 使用 echo password | sudo -S 测试带密码的 sudo
				sudoCmd := fmt.Sprintf("echo '%s' | sudo -S whoami 2>/dev/null", sudoPass)
				if err := session3.Run(sudoCmd); err == nil {
					output := strings.TrimSpace(sudoOutput2.String())
					if strings.Contains(output, "root") {
						result.HasSudo = true
						result.SudoNoPassword = false
					}
				}
				session3.Close()
			}
		}
	} else {
		// root 用户默认有 sudo 权限
		result.HasSudo = true
		result.SudoNoPassword = true
	}

	result.Duration = time.Since(startTime).Milliseconds()
	return result
}

// BatchTestSSHConnections 批量测试 SSH 连接
func (s *BatchInstallService) BatchTestSSHConnections(ctx context.Context, req SSHTestRequest) ([]SSHTestResult, error) {
	if len(req.Hosts) == 0 {
		return nil, fmt.Errorf("no hosts specified")
	}

	// 设置默认并发数
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = 5
	}
	if parallel > 20 {
		parallel = 20
	}

	// 创建工作通道
	jobs := make(chan HostInstallConfig, len(req.Hosts))
	results := make(chan SSHTestResult, len(req.Hosts))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for hostConfig := range jobs {
				result := s.TestSSHConnection(ctx, hostConfig)
				results <- result
			}
		}()
	}

	// 发送任务
	for _, host := range req.Hosts {
		if host.Port == 0 {
			host.Port = 22
		}
		jobs <- host
	}
	close(jobs)

	// 等待完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var testResults []SSHTestResult
	for result := range results {
		testResults = append(testResults, result)
	}

	return testResults, nil
}

// UninstallSaltMinion 卸载 Salt Minion
func (s *BatchInstallService) UninstallSaltMinion(ctx context.Context, config HostInstallConfig) error {
	// 建立 SSH 连接
	client, err := s.connectSSH(config)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	defer client.Close()

	// 检测操作系统
	osInfo, err := s.detectOS(client)
	if err != nil {
		return fmt.Errorf("OS detection failed: %v", err)
	}

	// 确定 sudo 前缀
	sudoPrefix := ""
	if config.UseSudo && config.Username != "root" {
		sudoPass := config.SudoPass
		if sudoPass == "" {
			sudoPass = config.Password
		}
		sudoPrefix = fmt.Sprintf("echo '%s' | sudo -S ", sudoPass)
	}

	// 构建卸载命令
	var uninstallCmd string
	switch osInfo.OS {
	case "ubuntu", "debian":
		uninstallCmd = fmt.Sprintf(`
set -e
echo "=== Stopping Salt Minion service ==="
%ssystemctl stop salt-minion || true
%ssystemctl disable salt-minion || true

echo "=== Removing Salt packages ==="
%sapt-get remove -y salt-minion salt-common || true
%sapt-get purge -y salt-minion salt-common || true
%sapt-get autoremove -y || true

echo "=== Cleaning up configuration ==="
%srm -rf /etc/salt /var/cache/salt /var/log/salt /var/run/salt

echo "=== Salt Minion uninstallation complete ==="
`, sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix)

	case "centos", "rhel", "rocky", "almalinux", "fedora":
		uninstallCmd = fmt.Sprintf(`
set -e
echo "=== Stopping Salt Minion service ==="
%ssystemctl stop salt-minion || true
%ssystemctl disable salt-minion || true

echo "=== Removing Salt packages ==="
%syum remove -y salt-minion salt || %sdnf remove -y salt-minion salt || true

echo "=== Cleaning up configuration ==="
%srm -rf /etc/salt /var/cache/salt /var/log/salt /var/run/salt

echo "=== Salt Minion uninstallation complete ==="
`, sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix, sudoPrefix)

	default:
		return fmt.Errorf("unsupported operating system: %s", osInfo.OS)
	}

	// 执行卸载命令
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var output bytes.Buffer
	session.Stdout = &output
	session.Stderr = &output

	if err := session.Run(uninstallCmd); err != nil {
		return fmt.Errorf("uninstall failed: %v, output: %s", err, output.String())
	}

	logrus.WithField("host", config.Host).Info("Salt Minion uninstalled successfully")
	return nil
}
