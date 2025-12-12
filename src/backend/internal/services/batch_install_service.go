package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// 定义 minions 缓存键常量
const saltMinionsKey = "saltstack:minions"

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
	Hosts           []HostInstallConfig `json:"hosts"`
	Parallel        int                 `json:"parallel"`         // 并发数，默认3
	MasterHost      string              `json:"master_host"`      // Salt Master 地址
	InstallType     string              `json:"install_type"`     // saltstack, slurm
	UseSudo         bool                `json:"use_sudo"`         // 是否使用 sudo
	SudoPass        string              `json:"sudo_pass"`        // sudo 密码（如果不同于登录密码）
	AutoAccept      bool                `json:"auto_accept"`      // 自动接受 minion key
	Version         string              `json:"version"`          // 安装版本
	InstallCategraf bool                `json:"install_categraf"` // 是否同时安装 Categraf 监控代理
	N9EHost         string              `json:"n9e_host"`         // Nightingale 服务器地址
	N9EPort         string              `json:"n9e_port"`         // Nightingale 端口（默认 17000）
	CategrafVersion string              `json:"categraf_version"` // Categraf 版本
}

// HostInstallConfig 单主机安装配置
type HostInstallConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	KeyPath         string `json:"key_path,omitempty"`
	MinionID        string `json:"minion_id,omitempty"`        // 可选，默认使用 hostname
	UseSudo         bool   `json:"use_sudo,omitempty"`         // 覆盖全局设置
	SudoPass        string `json:"sudo_pass,omitempty"`        // 覆盖全局设置
	Group           string `json:"group,omitempty"`            // 分组名称
	InstallCategraf bool   `json:"install_categraf,omitempty"` // 是否安装 Categraf
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
	Host     string   `json:"host"`
	MinionID string   `json:"minion_id,omitempty"` // 安装的 minion ID
	Status   string   `json:"status"`              // success, failed, partial
	Message  string   `json:"message"`
	Duration int64    `json:"duration"` // 耗时（毫秒）
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"` // 警告信息列表
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

// CalculateDynamicParallel 计算动态并行度
// 根据节点数量动态计算并行度，遵循指数递减规律：
//   - <= 20 台: 100% 并发（直接使用节点数）
//   - 21-50 台: 60% 并发
//   - 51-100 台: 50% 并发
//   - 101-500 台: 20% 并发
//   - 501-1000 台: 10% 并发
//   - 1001-5000 台: 3% 并发
//   - 5001-10000 台: 1% 并发
//   - > 10000 台: 0.1% 并发
//
// 最小并发数为 1，最大并发数默认为 100（可通过 maxParallel 参数调整）
func CalculateDynamicParallel(hostCount int, maxParallel int) int {
	if hostCount <= 0 {
		return 1
	}
	if maxParallel <= 0 {
		maxParallel = 100 // 默认最大并发数
	}

	var parallel int

	switch {
	case hostCount <= 20:
		// 小规模：100% 并发（直接使用节点数）
		parallel = hostCount
	case hostCount <= 50:
		// 小规模：60% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.6))
	case hostCount <= 100:
		// 中小规模：50% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.5))
	case hostCount <= 500:
		// 中规模：20% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.2))
	case hostCount <= 1000:
		// 中大规模：10% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.1))
	case hostCount <= 5000:
		// 大规模：3% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.03))
	case hostCount <= 10000:
		// 超大规模：1% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.01))
	default:
		// 超超大规模：0.1% 并发
		parallel = int(math.Ceil(float64(hostCount) * 0.001))
	}

	// 确保最小并发数为 1
	if parallel < 1 {
		parallel = 1
	}

	// 确保不超过最大并发数
	if parallel > maxParallel {
		parallel = maxParallel
	}

	return parallel
}

// GetParallelConfig 获取并行度配置信息（用于 API 返回）
type ParallelConfig struct {
	HostCount       int     `json:"host_count"`
	Parallel        int     `json:"parallel"`
	Percentage      float64 `json:"percentage"`
	MaxParallel     int     `json:"max_parallel"`
	IsAutoCalculate bool    `json:"is_auto_calculate"`
}

// GetParallelInfo 获取并行度详细信息
func GetParallelInfo(hostCount int, requestedParallel int, maxParallel int) ParallelConfig {
	if maxParallel <= 0 {
		maxParallel = 100
	}

	config := ParallelConfig{
		HostCount:   hostCount,
		MaxParallel: maxParallel,
	}

	// 如果请求中指定了并行度且大于0，使用指定值
	if requestedParallel > 0 {
		config.Parallel = requestedParallel
		if config.Parallel > maxParallel {
			config.Parallel = maxParallel
		}
		config.IsAutoCalculate = false
	} else {
		// 自动计算
		config.Parallel = CalculateDynamicParallel(hostCount, maxParallel)
		config.IsAutoCalculate = true
	}

	if hostCount > 0 {
		config.Percentage = float64(config.Parallel) / float64(hostCount) * 100
	}

	return config
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

	// 计算动态并行度
	// 如果请求中指定了 Parallel > 0，使用指定值；否则自动计算
	parallelInfo := GetParallelInfo(len(req.Hosts), req.Parallel, 100)
	req.Parallel = parallelInfo.Parallel

	logrus.WithFields(logrus.Fields{
		"task_id":           taskID,
		"host_count":        parallelInfo.HostCount,
		"parallel":          parallelInfo.Parallel,
		"percentage":        fmt.Sprintf("%.1f%%", parallelInfo.Percentage),
		"is_auto_calculate": parallelInfo.IsAutoCalculate,
	}).Info("Calculated parallel workers for batch install")

	// 处理 MasterHost - 如果是容器名称（如 "salt", "saltstack"），替换为实际的外部 IP
	externalHost := os.Getenv("EXTERNAL_HOST")
	if req.MasterHost == "" || req.MasterHost == "salt" || req.MasterHost == "saltstack" {
		if externalHost != "" {
			req.MasterHost = externalHost
			logrus.Infof("[BatchInstall] Using EXTERNAL_HOST for master: %s", req.MasterHost)
		} else {
			// 如果没有设置 EXTERNAL_HOST，保持原值（可能是主机名）
			if req.MasterHost == "" {
				req.MasterHost = "saltstack"
			}
			logrus.Warnf("[BatchInstall] EXTERNAL_HOST not set, using: %s (minion may not be able to connect)", req.MasterHost)
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
	// 存储脱敏后的配置到数据库（保护敏感信息）
	maskedReq := s.maskBatchInstallRequest(req)
	task.SetConfig(maskedReq)

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
			// 更新任务状态为失败
			if database.DB != nil && task.ID > 0 {
				task.Status = "failed"
				now := time.Now()
				task.EndTime = &now
				database.DB.Save(task)
			}
		}
		// 延迟关闭 SSE 通道，让客户端有时间接收最后的消息
		time.Sleep(2 * time.Second)
		s.CloseSSEChannel(taskID)
	}()

	s.sendEvent(taskID, SSEEvent{
		Type:    "progress",
		Message: fmt.Sprintf("Starting batch installation for %d hosts with %d parallel workers (%.1f%% concurrency)", len(req.Hosts), req.Parallel, float64(req.Parallel)/float64(len(req.Hosts))*100),
		Data: map[string]interface{}{
			"completed":  0,
			"total":      len(req.Hosts),
			"success":    0,
			"failed":     0,
			"progress":   0,
			"parallel":   req.Parallel,
			"percentage": float64(req.Parallel) / float64(len(req.Hosts)) * 100,
		},
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
	totalHosts := len(req.Hosts)

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

			// 实时更新任务统计 - 直接设置计数，不调用 UpdateHostStats
			// 因为 UpdateHostStats 会从 task.HostResults 重新计算，但我们的 HostResults 未加载
			database.DB.Model(task).Updates(map[string]interface{}{
				"success_hosts": successCount,
				"failed_hosts":  failedCount,
			})
		}

		// 计算进度
		completedCount := successCount + failedCount
		progress := 0
		if totalHosts > 0 {
			progress = completedCount * 100 / totalHosts
		}

		// 发送进度事件
		s.sendEvent(taskID, SSEEvent{
			Type:    "progress",
			Host:    result.Host,
			Message: fmt.Sprintf("Completed: %d/%d (Success: %d, Failed: %d)", completedCount, totalHosts, successCount, failedCount),
			Data: map[string]interface{}{
				"completed": completedCount,
				"total":     totalHosts,
				"success":   successCount,
				"failed":    failedCount,
				"progress":  progress,
				"host_result": map[string]interface{}{
					"host":     result.Host,
					"status":   result.Status,
					"error":    result.Error,
					"duration": result.Duration,
				},
			},
		})
	}

	// 计算最终状态
	finalStatus := "completed"
	if failedCount > 0 && successCount == 0 {
		finalStatus = "failed"
	} else if failedCount > 0 {
		finalStatus = "partial" // 部分成功
	}

	// 更新任务最终状态
	if database.DB != nil && task.ID > 0 {
		now := time.Now()
		duration := now.Sub(task.StartTime).Seconds()
		durationInt := int64(duration)

		// 使用 Updates 直接更新，避免 UpdateHostStats 重置值
		database.DB.Model(task).Updates(map[string]interface{}{
			"status":        finalStatus,
			"success_hosts": successCount,
			"failed_hosts":  failedCount,
			"end_time":      now,
			"duration":      durationInt,
		})
	}

	// 清除 minions 缓存，确保前端能立即获取最新的 minion 列表
	if successCount > 0 {
		s.invalidateMinionsCache()
	}

	// 发送完成事件
	s.sendEvent(taskID, SSEEvent{
		Type:    "complete",
		Message: fmt.Sprintf("Batch installation %s. Success: %d, Failed: %d", finalStatus, successCount, failedCount),
		Data: map[string]interface{}{
			"task_id":       taskID,
			"total_hosts":   totalHosts,
			"success_hosts": successCount,
			"failed_hosts":  failedCount,
			"status":        finalStatus,
			"progress":      100,
			"host_results":  hostResults,
		},
	})

	logrus.WithFields(logrus.Fields{
		"task_id": taskID,
		"total":   totalHosts,
		"success": successCount,
		"failed":  failedCount,
		"status":  finalStatus,
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

	// 构建 sudo 前缀 - 使用标准 sudo 格式，密码通过环境变量传递
	// 注意：非 root 用户需要 sudo，root 用户不需要
	sudoPrefix := ""
	if hostConfig.Username != "root" && useSudo {
		sudoPrefix = "sudo "
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

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    hostConfig.Host,
		Message: "Salt Minion installed, waiting for key registration...",
	})

	// 自动接受 Minion Key（如果启用）
	keyAccepted := false
	if req.AutoAccept {
		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    hostConfig.Host,
			Message: fmt.Sprintf("Auto-accepting minion key for %s...", minionID),
		})

		// 等待 minion 密钥出现并接受
		acceptErr := s.waitAndAcceptMinionKey(ctx, minionID, taskID, hostConfig.Host)
		if acceptErr != nil {
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    hostConfig.Host,
				Message: fmt.Sprintf("Warning: Failed to auto-accept minion key: %v", acceptErr),
			})
			s.logToDatabase(taskID, "warn", hostConfig.Host, fmt.Sprintf("Minion installed but key not auto-accepted: %v", acceptErr))
			// 不标记为失败，因为 minion 已安装，只是 key 未被接受
		} else {
			keyAccepted = true
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    hostConfig.Host,
				Message: fmt.Sprintf("Minion key accepted for %s", minionID),
			})
		}
	}

	// 验证 minion 是否能响应 test.ping（如果 key 已接受）
	if keyAccepted {
		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    hostConfig.Host,
			Message: fmt.Sprintf("Verifying minion %s is responding to master...", minionID),
		})

		pingErr := s.verifyMinionPing(ctx, minionID, taskID, hostConfig.Host)
		if pingErr != nil {
			result.Status = "partial"
			result.Error = fmt.Sprintf("Minion installed but not responding: %v", pingErr)
			result.Message = result.Error
			result.Duration = time.Since(startTime).Milliseconds()
			s.sendEvent(taskID, SSEEvent{
				Type:    "warning",
				Host:    hostConfig.Host,
				Message: result.Error,
			})
			s.logToDatabase(taskID, "warn", hostConfig.Host, result.Error)
			return result
		}

		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    hostConfig.Host,
			Message: fmt.Sprintf("Minion %s verified - responding to test.ping", minionID),
		})
	}

	// 安装 Categraf 监控代理（如果启用）
	// 检查全局设置或单主机设置
	shouldInstallCategraf := req.InstallCategraf || hostConfig.InstallCategraf
	if shouldInstallCategraf {
		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    hostConfig.Host,
			Message: "Installing Categraf monitoring agent...",
		})
		s.logToDatabase(taskID, "info", hostConfig.Host, "Installing Categraf monitoring agent...")

		categrafErr := s.installCategraf(client, osInfo, req, sudoPrefix, taskID, hostConfig.Host, minionID)
		if categrafErr != nil {
			s.sendEvent(taskID, SSEEvent{
				Type:    "warning",
				Host:    hostConfig.Host,
				Message: fmt.Sprintf("Categraf installation failed: %v (Salt Minion was installed successfully)", categrafErr),
			})
			s.logToDatabase(taskID, "warn", hostConfig.Host, fmt.Sprintf("Categraf installation failed: %v", categrafErr))
			// 不标记为失败，因为 Salt Minion 已安装成功
		} else {
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    hostConfig.Host,
				Message: "Categraf monitoring agent installed successfully",
			})
			s.logToDatabase(taskID, "info", hostConfig.Host, "Categraf monitoring agent installed successfully")
		}
	}

	// 设置 Minion 分组（如果指定了分组）
	if hostConfig.Group != "" && minionID != "" {
		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    hostConfig.Host,
			Message: fmt.Sprintf("Setting minion group to: %s", hostConfig.Group),
		})
		s.logToDatabase(taskID, "info", hostConfig.Host, fmt.Sprintf("Setting minion group to: %s", hostConfig.Group))

		if err := s.setMinionGroup(minionID, hostConfig.Group); err != nil {
			s.sendEvent(taskID, SSEEvent{
				Type:    "warning",
				Host:    hostConfig.Host,
				Message: fmt.Sprintf("Failed to set minion group: %v", err),
			})
			s.logToDatabase(taskID, "warn", hostConfig.Host, fmt.Sprintf("Failed to set minion group: %v", err))
		} else {
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    hostConfig.Host,
				Message: fmt.Sprintf("Minion group set to: %s", hostConfig.Group),
			})
			s.logToDatabase(taskID, "info", hostConfig.Host, fmt.Sprintf("Minion group set to: %s", hostConfig.Group))
		}
	}

	// 部署节点指标采集（GPU/IB 检测）
	if keyAccepted && minionID != "" {
		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    hostConfig.Host,
			Message: "Deploying node metrics collection (GPU/IB detection)...",
		})
		s.logToDatabase(taskID, "info", hostConfig.Host, "Deploying node metrics collection...")

		if err := s.deployNodeMetrics(minionID, req.MasterHost); err != nil {
			s.sendEvent(taskID, SSEEvent{
				Type:    "warning",
				Host:    hostConfig.Host,
				Message: fmt.Sprintf("Failed to deploy node metrics: %v", err),
			})
			s.logToDatabase(taskID, "warn", hostConfig.Host, fmt.Sprintf("Failed to deploy node metrics: %v", err))
		} else {
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    hostConfig.Host,
				Message: "Node metrics collection deployed successfully",
			})
			s.logToDatabase(taskID, "info", hostConfig.Host, "Node metrics collection deployed successfully")

			// 触发立即采集一次，同步 minion 数据到数据库
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    hostConfig.Host,
				Message: "Triggering initial metrics collection...",
			})
			if err := s.triggerImmediateMetricsCollection(minionID); err != nil {
				s.sendEvent(taskID, SSEEvent{
					Type:    "warning",
					Host:    hostConfig.Host,
					Message: fmt.Sprintf("Failed to trigger initial metrics collection: %v", err),
				})
				s.logToDatabase(taskID, "warn", hostConfig.Host, fmt.Sprintf("Failed to trigger initial metrics: %v", err))
			} else {
				s.sendEvent(taskID, SSEEvent{
					Type:    "log",
					Host:    hostConfig.Host,
					Message: "Initial metrics collection triggered, data will sync shortly",
				})
				s.logToDatabase(taskID, "info", hostConfig.Host, "Initial metrics collection triggered")
			}
		}
	}

	result.Status = "success"
	result.MinionID = minionID // 记录 minion ID
	result.Message = "Salt Minion installed and started successfully"
	if keyAccepted {
		result.Message = "Salt Minion installed, key accepted, and verified responding"
	}
	if req.InstallCategraf {
		result.Message += " (with Categraf monitoring)"
	}
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
		logrus.WithField("host", config.Host).Debug("[connectSSH] Password authentication enabled")
	}

	// 密钥认证
	if config.KeyPath != "" {
		key, err := os.ReadFile(config.KeyPath)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"host":     config.Host,
				"key_path": config.KeyPath,
				"error":    err.Error(),
			}).Warn("[connectSSH] Failed to read key file")
		} else {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"host":     config.Host,
					"key_path": config.KeyPath,
					"error":    err.Error(),
				}).Warn("[connectSSH] Failed to parse private key")
			} else {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				logrus.WithField("host", config.Host).Debug("[connectSSH] Key authentication enabled")
			}
		}
	}

	if len(authMethods) == 0 {
		logrus.WithField("host", config.Host).Error("[connectSSH] No authentication method available")
		return nil, fmt.Errorf("no authentication method available")
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	logrus.WithFields(logrus.Fields{
		"host":     config.Host,
		"port":     config.Port,
		"username": config.Username,
		"addr":     addr,
	}).Debug("[connectSSH] Attempting SSH connection")

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"host":  config.Host,
			"addr":  addr,
			"error": err.Error(),
		}).Error("[connectSSH] SSH dial failed")
		return nil, err
	}

	return client, nil
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

	// 使用 ScriptLoader 获取操作系统检测脚本
	scriptLoader := GetScriptLoader()
	cmd := scriptLoader.GenerateOSDetectScript()

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
// 优先尝试从 AppHub 下载安装包，如果失败则回退到在线安装（使用 Salt Bootstrap 脚本）
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

	// RPM 架构名称
	rpmArch := osInfo.Arch
	if rpmArch == "amd64" {
		rpmArch = "x86_64"
	} else if rpmArch == "arm64" {
		rpmArch = "aarch64"
	}

	// 生成 Master 公钥获取 URL（使用一次性令牌）
	masterPubURL := ""
	if saltKeyHandler := GetSaltKeyHandler(); saltKeyHandler != nil {
		_, url, err := saltKeyHandler.GenerateInstallTokenForBatch(minionID, 600) // 10分钟有效期
		if err == nil {
			masterPubURL = url
			logrus.WithFields(logrus.Fields{
				"minion_id":      minionID,
				"master_pub_url": url[:50] + "...",
			}).Info("Generated install token for master pub key")
		} else {
			logrus.WithError(err).Warn("Failed to generate install token, master pub key will not be pre-synced")
		}
	}

	// 使用 ScriptLoader 生成安装脚本
	scriptLoader := GetScriptLoader()
	installCmd, err := scriptLoader.GenerateSaltInstallScript(SaltInstallParams{
		AppHubURL:    appHubURL,
		MasterHost:   masterHost,
		MinionID:     minionID,
		Version:      version,
		Arch:         arch,
		RpmArch:      rpmArch,
		SudoPrefix:   sudoPrefix,
		OS:           osInfo.OS,
		OSVersion:    osInfo.Version,
		MasterPubURL: masterPubURL,
	})
	if err != nil {
		return fmt.Errorf("failed to generate install script: %v", err)
	}

	// 执行安装命令
	return s.runCommandWithLogging(client, installCmd, taskID, host)
}

// runCommandWithLogging 执行 SSH 命令并记录输出到数据库
func (s *BatchInstallService) runCommandWithLogging(client *ssh.Client, cmd, taskID, host string) error {
	startTime := time.Now()
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

	// 用于收集完整输出
	var stdoutBuf, stderrBuf strings.Builder
	var stdoutMu, stderrMu sync.Mutex

	// 启动命令
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	// 读取输出并发送 SSE 事件，同时收集完整输出
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.streamOutputWithBuffer(stdout, taskID, host, "stdout", &stdoutBuf, &stdoutMu)
	}()
	go func() {
		defer wg.Done()
		s.streamOutputWithBuffer(stderr, taskID, host, "stderr", &stderrBuf, &stderrMu)
	}()

	// 等待输出读取完成
	wg.Wait()

	// 等待命令完成
	cmdErr := session.Wait()
	duration := time.Since(startTime).Milliseconds()

	// 获取完整输出
	stdoutMu.Lock()
	fullStdout := stdoutBuf.String()
	stdoutMu.Unlock()

	stderrMu.Lock()
	fullStderr := stderrBuf.String()
	stderrMu.Unlock()

	// 记录 SSH 命令执行日志
	exitCode := 0
	status := "success"
	if cmdErr != nil {
		status = "failed"
		// 尝试获取退出码
		if exitErr, ok := cmdErr.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			exitCode = -1
		}
	}

	// 保存到 SSHLog 表
	s.logSSHCommand(taskID, host, "[Salt Minion Installation Script]", fullStdout, fullStderr, exitCode, duration, status)

	// 同时保存摘要到 TaskLog 表
	outputSummary := fullStdout
	if len(outputSummary) > 5000 {
		outputSummary = outputSummary[:5000] + "\n... [truncated]"
	}
	s.logToDatabaseWithCategory(taskID, "info", host, "install", "Salt Minion installation script executed", outputSummary)

	if cmdErr != nil {
		s.logToDatabaseWithCategory(taskID, "error", host, "install", fmt.Sprintf("Installation script failed with exit code %d", exitCode), fullStderr)
		return fmt.Errorf("command failed: %v", cmdErr)
	}

	return nil
}

// installCategraf 安装 Categraf 监控代理
func (s *BatchInstallService) installCategraf(client *ssh.Client, osInfo *models.OSInfo, req BatchInstallRequest, sudoPrefix, taskID, host, hostname string) error {
	scriptLoader := GetScriptLoader()

	// 设置默认值
	n9eHost := req.N9EHost
	if n9eHost == "" {
		n9eHost = os.Getenv("N9E_HOST")
		if n9eHost == "" {
			n9eHost = os.Getenv("EXTERNAL_HOST")
		}
	}

	n9ePort := req.N9EPort
	if n9ePort == "" {
		n9ePort = os.Getenv("N9E_PORT")
		if n9ePort == "" {
			n9ePort = "17000"
		}
	}

	categrafVersion := req.CategrafVersion
	if categrafVersion == "" {
		categrafVersion = os.Getenv("CATEGRAF_VERSION")
		if categrafVersion == "" {
			categrafVersion = "v0.4.25"
		}
	}
	// Ensure version starts with 'v'
	if categrafVersion != "" && categrafVersion[0] != 'v' {
		categrafVersion = "v" + categrafVersion
	}

	appHubURL := os.Getenv("APPHUB_URL")
	if appHubURL == "" {
		externalHost := os.Getenv("EXTERNAL_HOST")
		apphubPort := os.Getenv("APPHUB_PORT")
		if externalHost != "" && apphubPort != "" {
			appHubURL = fmt.Sprintf("http://%s:%s", externalHost, apphubPort)
		}
	}

	// 模板数据
	data := map[string]string{
		"SudoPrefix":      sudoPrefix,
		"N9EHost":         n9eHost,
		"N9EPort":         n9ePort,
		"CategrafVersion": categrafVersion,
		"AppHubURL":       appHubURL,
		"Hostname":        hostname,
		"HostIP":          host,
		"OS":              strings.ToLower(osInfo.OS),
	}

	// 生成安装脚本
	script, err := scriptLoader.GenerateCategrafInstallScript(data)
	if err != nil {
		return fmt.Errorf("failed to generate Categraf install script: %v", err)
	}

	// 执行安装脚本
	logrus.WithFields(logrus.Fields{
		"task_id":  taskID,
		"host":     host,
		"n9e_host": n9eHost,
		"n9e_port": n9ePort,
		"version":  categrafVersion,
	}).Info("[BatchInstall] Installing Categraf monitoring agent")

	return s.runCommandWithLogging(client, script, taskID, host)
}

// runCommand 执行 SSH 命令并记录输出 (保留旧函数以兼容)
func (s *BatchInstallService) runCommand(client *ssh.Client, cmd, taskID, host string) error {
	return s.runCommandWithLogging(client, cmd, taskID, host)
}

// streamOutputWithBuffer 流式输出并收集到 buffer
func (s *BatchInstallService) streamOutputWithBuffer(reader io.Reader, taskID, host, streamType string, buf *strings.Builder, mu *sync.Mutex) {
	scanner := make([]byte, 4096)
	for {
		n, err := reader.Read(scanner)
		if n > 0 {
			output := string(scanner[:n])

			// 收集到 buffer
			mu.Lock()
			buf.WriteString(output)
			mu.Unlock()

			// 按行发送 SSE 事件
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					// 检测 Warning 行并发送 warning 类型事件
					eventType := "log"
					lineLC := strings.ToLower(line)
					if strings.Contains(lineLC, "warning:") || strings.Contains(lineLC, "warn:") {
						eventType = "warning"
					} else if strings.Contains(lineLC, "error:") || strings.Contains(lineLC, "failed:") {
						eventType = "error"
					}

					s.sendEvent(taskID, SSEEvent{
						Type:    eventType,
						Host:    host,
						Message: line,
						Data: map[string]string{
							"stream": streamType,
						},
					})
					// 同时记录每行到数据库（仅重要的行）
					if strings.HasPrefix(line, "===") || strings.Contains(lineLC, "error") || strings.Contains(lineLC, "failed") || strings.Contains(lineLC, "warning") {
						logLevel := "info"
						if eventType == "warning" {
							logLevel = "warn"
						} else if eventType == "error" {
							logLevel = "error"
						}
						s.logToDatabaseWithCategory(taskID, logLevel, host, "output", line, "")
					}
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
	s.logToDatabaseWithCategory(taskID, level, host, "general", message, "")
}

// logToDatabaseWithCategory 记录带分类的日志到数据库
// 自动脱敏日志中的敏感信息（密码、密钥、Token等）
func (s *BatchInstallService) logToDatabaseWithCategory(taskID, level, host, category, message, output string) {
	if database.DB == nil {
		return
	}

	// 脱敏日志消息和输出中的敏感信息
	maskedMessage := utils.MaskLogMessage(message)
	maskedOutput := utils.MaskLogMessage(output)

	log := &models.TaskLog{
		TaskID:    taskID,
		Host:      host,
		LogLevel:  level,
		Category:  category,
		Message:   maskedMessage,
		Output:    maskedOutput,
		Timestamp: time.Now(),
	}

	if err := database.DB.Create(log).Error; err != nil {
		logrus.WithError(err).Warn("Failed to save task log to database")
	}
}

// logSSHCommand 记录 SSH 命令执行日志
// 自动脱敏命令和输出中的敏感信息
func (s *BatchInstallService) logSSHCommand(taskID, host, command, output, errorOutput string, exitCode int, duration int64, status string) {
	if database.DB == nil {
		return
	}

	// 脱敏命令和输出中的敏感信息
	maskedCommand := utils.MaskLogMessage(command)
	maskedOutput := utils.MaskLogMessage(output)
	maskedErrorOutput := utils.MaskLogMessage(errorOutput)

	sshLog := &models.SSHLog{
		TaskID:      taskID,
		Host:        host,
		Port:        22,
		User:        "", // 可以在调用时传入
		Command:     maskedCommand,
		Output:      maskedOutput,
		ErrorOutput: maskedErrorOutput,
		ExitCode:    exitCode,
		Duration:    duration,
		Status:      status,
		StartTime:   time.Now().Add(-time.Duration(duration) * time.Millisecond),
		EndTime:     time.Now(),
	}

	if err := database.DB.Create(sshLog).Error; err != nil {
		logrus.WithError(err).Warn("Failed to save SSH log to database")
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

// GetTaskLogsFiltered 获取带过滤条件的任务日志
func (s *BatchInstallService) GetTaskLogsFiltered(taskID, host, level, category string, limit, offset int) ([]models.TaskLog, int64, error) {
	if database.DB == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	query := database.DB.Where("task_id = ?", taskID)

	if host != "" {
		query = query.Where("host = ?", host)
	}
	if level != "" {
		query = query.Where("log_level = ?", level)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var total int64
	if err := query.Model(&models.TaskLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.TaskLog
	err := query.Order("timestamp ASC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

// GetSSHLogs 获取 SSH 执行日志
func (s *BatchInstallService) GetSSHLogs(taskID, host string) ([]models.SSHLog, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := database.DB.Where("task_id = ?", taskID)

	if host != "" {
		query = query.Where("host = ?", host)
	}

	var logs []models.SSHLog
	err := query.Order("start_time ASC").Find(&logs).Error
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

	// 使用动态并行度计算（SSH 测试最大并发 50）
	parallelInfo := GetParallelInfo(len(req.Hosts), req.Parallel, 50)
	parallel := parallelInfo.Parallel

	logrus.WithFields(logrus.Fields{
		"host_count":        parallelInfo.HostCount,
		"parallel":          parallel,
		"percentage":        fmt.Sprintf("%.1f%%", parallelInfo.Percentage),
		"is_auto_calculate": parallelInfo.IsAutoCalculate,
	}).Info("Calculated parallel workers for SSH batch test")

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
	logrus.WithFields(logrus.Fields{
		"host":     config.Host,
		"port":     config.Port,
		"username": config.Username,
		"use_sudo": config.UseSudo,
	}).Info("[UninstallMinion] Starting uninstall process")

	// 建立 SSH 连接
	client, err := s.connectSSH(config)
	if err != nil {
		logrus.WithError(err).WithField("host", config.Host).Error("[UninstallMinion] SSH connection failed")
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	defer client.Close()
	logrus.WithField("host", config.Host).Info("[UninstallMinion] SSH connection established")

	// 检测操作系统
	osInfo, err := s.detectOS(client)
	if err != nil {
		logrus.WithError(err).WithField("host", config.Host).Error("[UninstallMinion] OS detection failed")
		return fmt.Errorf("OS detection failed: %v", err)
	}
	logrus.WithFields(logrus.Fields{
		"host":    config.Host,
		"os":      osInfo.OS,
		"version": osInfo.Version,
	}).Info("[UninstallMinion] OS detected")

	// 确定 sudo 前缀 - 使用标准 sudo 格式
	sudoPrefix := ""
	if config.UseSudo && config.Username != "root" {
		sudoPrefix = "sudo "
	}

	// 使用 ScriptLoader 生成卸载脚本
	scriptLoader := GetScriptLoader()
	uninstallCmd, err := scriptLoader.GenerateSaltUninstallScript(SaltUninstallParams{
		SudoPrefix: sudoPrefix,
		OS:         osInfo.OS,
	})
	if err != nil {
		logrus.WithError(err).WithField("host", config.Host).Error("[UninstallMinion] Script generation failed")
		return fmt.Errorf("failed to generate uninstall script: %v", err)
	}
	logrus.WithField("host", config.Host).Info("[UninstallMinion] Uninstall script generated")

	// 执行卸载命令
	session, err := client.NewSession()
	if err != nil {
		logrus.WithError(err).WithField("host", config.Host).Error("[UninstallMinion] Failed to create SSH session")
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var output bytes.Buffer
	session.Stdout = &output
	session.Stderr = &output

	logrus.WithField("host", config.Host).Info("[UninstallMinion] Executing uninstall script...")
	if err := session.Run(uninstallCmd); err != nil {
		logrus.WithFields(logrus.Fields{
			"host":   config.Host,
			"error":  err.Error(),
			"output": output.String(),
		}).Error("[UninstallMinion] Script execution failed")
		return fmt.Errorf("uninstall failed: %v, output: %s", err, output.String())
	}

	logrus.WithFields(logrus.Fields{
		"host":   config.Host,
		"output": output.String(),
	}).Info("[UninstallMinion] Salt Minion uninstalled successfully")
	return nil
}

// waitAndAcceptMinionKey 等待并接受 Minion 密钥
// 注意：此函数使用独立的 context，不受调用方 context 取消的影响
// 优化：增加随机延迟和指数退避，避免大量并发请求同时访问 Salt Master
func (s *BatchInstallService) waitAndAcceptMinionKey(parentCtx context.Context, minionID, taskID, host string) error {
	// 创建独立的 context，设置 90 秒超时（增加超时时间）
	// 不使用 parentCtx 是因为它可能在 HTTP 请求结束后被取消
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 创建 SaltStack 服务实例
	saltService := NewSaltStackService()
	if saltService == nil {
		return fmt.Errorf("SaltStack service not available")
	}

	// 添加随机延迟（0-3秒），避免多个节点同时请求
	jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
	time.Sleep(jitter)

	// 初始轮询间隔和最大间隔
	basePollInterval := 2 * time.Second
	maxPollInterval := 10 * time.Second
	currentInterval := basePollInterval

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    host,
		Message: "Waiting for minion key to appear (max 90s)...",
	})

	retryCount := 0
	maxRetries := 30 // 最多重试 30 次

	for retryCount < maxRetries {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for minion key")
		default:
		}

		retryCount++

		// 获取 Salt 状态（包含密钥信息）
		status, err := saltService.GetStatus(ctx)
		if err != nil {
			logrus.WithError(err).Warn("Failed to get salt status, retrying...")
			// 使用指数退避
			time.Sleep(currentInterval)
			if currentInterval < maxPollInterval {
				currentInterval = time.Duration(float64(currentInterval) * 1.5)
				if currentInterval > maxPollInterval {
					currentInterval = maxPollInterval
				}
			}
			continue
		}

		// 重置间隔（成功获取状态）
		currentInterval = basePollInterval

		// 检查是否在 unaccepted 列表中
		for _, k := range status.UnacceptedKeys {
			if k == minionID {
				s.sendEvent(taskID, SSEEvent{
					Type:    "log",
					Host:    host,
					Message: fmt.Sprintf("Found unaccepted key for %s, accepting...", minionID),
				})

				// 添加小延迟再接受，避免 Salt Master 负载过高
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

				// 接受密钥，带重试
				acceptRetries := 3
				var acceptErr error
				for i := 0; i < acceptRetries; i++ {
					acceptErr = saltService.AcceptMinion(ctx, minionID)
					if acceptErr == nil {
						break
					}
					logrus.WithError(acceptErr).Warnf("Accept minion key attempt %d failed, retrying...", i+1)
					time.Sleep(time.Duration(i+1) * time.Second)
				}
				if acceptErr != nil {
					return fmt.Errorf("failed to accept minion key after %d attempts: %v", acceptRetries, acceptErr)
				}

				// 等待一小段时间让 master 处理
				time.Sleep(3 * time.Second)

				s.sendEvent(taskID, SSEEvent{
					Type:    "log",
					Host:    host,
					Message: fmt.Sprintf("Minion key for %s accepted successfully", minionID),
				})
				return nil
			}
		}

		// 检查是否已经在 accepted 列表中
		for _, k := range status.AcceptedKeys {
			if k == minionID {
				s.sendEvent(taskID, SSEEvent{
					Type:    "log",
					Host:    host,
					Message: fmt.Sprintf("Minion key for %s is already accepted", minionID),
				})
				return nil
			}
		}

		time.Sleep(currentInterval)
	}

	return fmt.Errorf("timeout waiting for minion key after %d retries", maxRetries)
}

// verifyMinionPing 验证 Minion 能否响应 test.ping
// 使用独立的 context，设置更长超时，最多重试 15 次
// 优化：增加随机延迟和指数退避，处理大并发场景下 Salt Master 负载问题
func (s *BatchInstallService) verifyMinionPing(parentCtx context.Context, minionID, taskID, host string) error {
	// 创建独立的 context，设置 60 秒超时（增加超时时间）
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 创建 SaltStack 服务实例
	saltService := NewSaltStackService()
	if saltService == nil {
		return fmt.Errorf("SaltStack service not available")
	}

	// 添加随机延迟（0-2秒），避免多个节点同时 ping
	jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
	time.Sleep(jitter)

	maxRetries := 15 // 增加重试次数
	basePollInterval := 2 * time.Second
	maxPollInterval := 8 * time.Second
	currentInterval := basePollInterval

	s.sendEvent(taskID, SSEEvent{
		Type:    "log",
		Host:    host,
		Message: fmt.Sprintf("Verifying minion %s can respond to test.ping (max %d attempts)...", minionID, maxRetries),
	})

	for retry := 1; retry <= maxRetries; retry++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for minion ping response")
		default:
		}

		// 执行 test.ping
		online, err := saltService.Ping(ctx, minionID)
		if err != nil {
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    host,
				Message: fmt.Sprintf("Ping attempt %d/%d failed: %v", retry, maxRetries, err),
			})
			if retry < maxRetries {
				// 使用指数退避
				time.Sleep(currentInterval)
				if currentInterval < maxPollInterval {
					currentInterval = time.Duration(float64(currentInterval) * 1.3)
					if currentInterval > maxPollInterval {
						currentInterval = maxPollInterval
					}
				}
			}
			continue
		}

		if online {
			s.sendEvent(taskID, SSEEvent{
				Type:    "log",
				Host:    host,
				Message: fmt.Sprintf("Minion %s responded to test.ping successfully (attempt %d/%d)", minionID, retry, maxRetries),
			})
			return nil
		}

		s.sendEvent(taskID, SSEEvent{
			Type:    "log",
			Host:    host,
			Message: fmt.Sprintf("Ping attempt %d/%d: no response from minion %s", retry, maxRetries, minionID),
		})

		if retry < maxRetries {
			// 使用指数退避
			time.Sleep(currentInterval)
			if currentInterval < maxPollInterval {
				currentInterval = time.Duration(float64(currentInterval) * 1.3)
				if currentInterval > maxPollInterval {
					currentInterval = maxPollInterval
				}
			}
		}
	}

	return fmt.Errorf("minion %s did not respond to test.ping after %d attempts", minionID, maxRetries)
}

// maskBatchInstallRequest 创建脱敏后的请求副本用于存储到数据库
// 保留主机、端口、用户名的前几位，密码和密钥完全脱敏
func (s *BatchInstallService) maskBatchInstallRequest(req BatchInstallRequest) BatchInstallRequest {
	maskedReq := BatchInstallRequest{
		Hosts:       make([]HostInstallConfig, len(req.Hosts)),
		Parallel:    req.Parallel,
		MasterHost:  req.MasterHost,
		InstallType: req.InstallType,
		UseSudo:     req.UseSudo,
		SudoPass:    utils.MaskPassword(req.SudoPass),
		AutoAccept:  req.AutoAccept,
		Version:     req.Version,
	}

	for i, host := range req.Hosts {
		maskedReq.Hosts[i] = HostInstallConfig{
			Host:     host.Host,
			Port:     host.Port,
			Username: host.Username, // 保留用户名用于问题排查
			Password: utils.MaskPassword(host.Password),
			KeyPath:  host.KeyPath,
			MinionID: host.MinionID,
			UseSudo:  host.UseSudo,
			SudoPass: utils.MaskPassword(host.SudoPass),
		}
	}

	return maskedReq
}

// triggerImmediateMetricsCollection 触发 minion 立即执行一次指标采集
// 在安装完成后调用，确保 minion 数据能立即同步到数据库
func (s *BatchInstallService) triggerImmediateMetricsCollection(minionID string) error {
	if minionID == "" {
		return fmt.Errorf("minion ID is required")
	}

	// 获取 Salt API 配置
	saltAPIURL := os.Getenv("SALT_API_URL")
	saltAPIUser := os.Getenv("SALT_API_USERNAME")
	if saltAPIUser == "" {
		saltAPIUser = os.Getenv("SALT_API_USER")
	}
	if saltAPIUser == "" {
		saltAPIUser = "saltapi"
	}
	saltAPIPass := os.Getenv("SALT_API_PASSWORD")
	saltAPIEauth := os.Getenv("SALT_API_EAUTH")
	if saltAPIEauth == "" {
		saltAPIEauth = "file"
	}

	if saltAPIURL == "" {
		return fmt.Errorf("SALT_API_URL not configured")
	}

	if saltAPIPass == "" {
		return fmt.Errorf("salt API password not configured")
	}

	// 创建 HTTP 客户端（跳过 TLS 验证）
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}

	// 构建触发采集的命令 - 直接执行采集脚本
	triggerCmd := "/opt/ai-infra/node-metrics/collect-node-metrics.sh 2>/dev/null || true"

	// 构建请求 payload
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{triggerCmd},
		"kwarg": map[string]interface{}{
			"timeout":      30,
			"shell":        "/bin/bash",
			"python_shell": true,
		},
		"username": saltAPIUser,
		"password": saltAPIPass,
		"eauth":    saltAPIEauth,
	}

	// 发送 HTTP 请求
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", saltAPIURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("salt API error: %d %s", resp.StatusCode, string(body))
	}

	logrus.Infof("[BatchInstall] Triggered immediate metrics collection for minion %s", minionID)
	return nil
}

// invalidateMinionsCache 使 minions 缓存失效
// 在批量安装完成后调用，确保前端能立即获取最新的 minion 列表
func (s *BatchInstallService) invalidateMinionsCache() {
	if cache.RDB == nil {
		logrus.Warn("[BatchInstall] Redis client not available, skip cache invalidation")
		return
	}

	err := cache.Delete(saltMinionsKey)
	if err != nil {
		logrus.WithError(err).Warn("[BatchInstall] Failed to invalidate minions cache")
	} else {
		logrus.Info("[BatchInstall] Minions cache invalidated, frontend will fetch fresh data")
	}
}

// setMinionGroup 设置 Minion 的分组
// 如果分组不存在则自动创建
func (s *BatchInstallService) setMinionGroup(minionID string, groupName string) error {
	if minionID == "" || groupName == "" {
		return nil
	}

	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// 先删除该 Minion 的所有分组关系
	if err := db.Where("minion_id = ?", minionID).Delete(&models.MinionGroupMembership{}).Error; err != nil {
		logrus.WithError(err).Warnf("[BatchInstall] Failed to clear existing group for minion %s", minionID)
	}

	// 查找或创建分组
	var group models.MinionGroup
	err := db.Where("name = ?", groupName).First(&group).Error
	if err != nil {
		// 分组不存在，创建新分组
		group = models.MinionGroup{
			Name:        groupName,
			DisplayName: groupName,
			Description: "Auto-created during batch installation",
		}
		if err := db.Create(&group).Error; err != nil {
			return fmt.Errorf("failed to create group: %v", err)
		}
		logrus.Infof("[BatchInstall] Created new group: %s", groupName)
	}

	// 添加成员关系
	membership := models.MinionGroupMembership{
		MinionID: minionID,
		GroupID:  group.ID,
	}
	if err := db.Create(&membership).Error; err != nil {
		return fmt.Errorf("failed to add minion to group: %v", err)
	}

	logrus.Infof("[BatchInstall] Minion %s added to group %s", minionID, groupName)
	return nil
}

// deployNodeMetrics 部署节点指标采集（GPU/IB 检测脚本）
// 通过 Salt API 直接部署采集脚本到节点
// 不依赖 Salt State 文件，直接通过 cmd.run 部署
func (s *BatchInstallService) deployNodeMetrics(minionID string, masterHost string) error {
	if minionID == "" {
		return fmt.Errorf("minion ID is required")
	}

	// 获取 Salt API 配置
	saltAPIURL := os.Getenv("SALT_API_URL")
	saltAPIUser := os.Getenv("SALT_API_USERNAME")
	if saltAPIUser == "" {
		saltAPIUser = os.Getenv("SALT_API_USER")
	}
	if saltAPIUser == "" {
		saltAPIUser = "saltapi"
	}
	saltAPIPass := os.Getenv("SALT_API_PASSWORD")
	saltAPIEauth := os.Getenv("SALT_API_EAUTH")
	if saltAPIEauth == "" {
		saltAPIEauth = "file"
	}

	if saltAPIURL == "" {
		// 尝试从 master host 构建 URL
		if masterHost != "" {
			saltAPIURL = fmt.Sprintf("https://%s:8000", masterHost)
		} else {
			return fmt.Errorf("SALT_API_URL not configured")
		}
	}

	if saltAPIPass == "" {
		return fmt.Errorf("salt API password not configured")
	}

	// 获取回调 URL
	callbackURL := os.Getenv("NODE_METRICS_CALLBACK_URL")
	if callbackURL == "" {
		backendURL := os.Getenv("BACKEND_URL")
		if backendURL == "" {
			backendURL = "http://localhost:8080"
		}
		callbackURL = backendURL + "/api/saltstack/node-metrics/callback"
	}

	// 获取 API Token
	apiToken := os.Getenv("NODE_METRICS_API_TOKEN")

	// 采集间隔（分钟）
	collectInterval := os.Getenv("NODE_METRICS_COLLECT_INTERVAL")
	if collectInterval == "" {
		collectInterval = "3"
	}

	// 创建 HTTP 客户端（跳过 TLS 验证）
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   120 * time.Second,
	}

	// 构建部署脚本（使用 ScriptLoader 加载外部脚本）
	deployScript, err := GetScriptLoader().GenerateNodeMetricsDeployScript(NodeMetricsDeployParams{
		CallbackURL:     callbackURL,
		CollectInterval: collectInterval,
		APIToken:        apiToken,
		MinionID:        minionID,
	})
	if err != nil {
		return fmt.Errorf("failed to generate node metrics deploy script: %v", err)
	}

	// 构建请求 payload - 使用 cmd.run 执行部署脚本
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{deployScript},
		"kwarg": map[string]interface{}{
			"timeout":      60,
			"shell":        "/bin/bash",
			"python_shell": true,
		},
		"username": saltAPIUser,
		"password": saltAPIPass,
		"eauth":    saltAPIEauth,
	}

	// 发送 HTTP 请求
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", saltAPIURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("salt API error: %d %s", resp.StatusCode, string(body))
	}

	// 解析响应检查执行结果
	body, _ := io.ReadAll(resp.Body)
	logrus.Infof("[BatchInstall] Deployed node-metrics to minion %s, response: %s", minionID, string(body))
	return nil
}
