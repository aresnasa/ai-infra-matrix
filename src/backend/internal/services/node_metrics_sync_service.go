package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/scripts"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
	"gorm.io/gorm"
)

// NodeMetricsSyncService 节点指标同步服务
// 定期使用 Salt 命令获取所有节点的 CPU/内存/GPU/IB 等指标，并同步到 Redis 和数据库
type NodeMetricsSyncService struct {
	db              *gorm.DB
	redisClient     *redis.Client
	saltMasterURL   string
	saltAPIUsername string
	saltAPIPassword string
	saltAPIEauth    string
	stopChan        chan struct{}
	syncInterval    time.Duration // 同步间隔，默认 60 秒
	mu              sync.RWMutex
	running         bool
}

// syncSaltAPIClient 内部 Salt API 客户端（避免与 handler 中的冲突）
type syncSaltAPIClient struct {
	baseURL  string
	token    string
	client   *http.Client
	username string
	password string
	eauth    string
}

// NewNodeMetricsSyncService 创建节点指标同步服务
func NewNodeMetricsSyncService() *NodeMetricsSyncService {
	syncInterval := 60 * time.Second // 默认 60 秒
	if envInterval := os.Getenv("NODE_METRICS_SYNC_INTERVAL"); envInterval != "" {
		if seconds, err := strconv.Atoi(envInterval); err == nil && seconds > 0 {
			syncInterval = time.Duration(seconds) * time.Second
		}
	}

	return &NodeMetricsSyncService{
		db:              database.DB,
		redisClient:     cache.RDB,
		saltMasterURL:   getEnvDefaultSync("SALT_API_URL", "http://ai-infra-salt-master-1:8000"),
		saltAPIUsername: getEnvDefaultSync("SALT_API_USERNAME", "saltapi"),
		saltAPIPassword: os.Getenv("SALT_API_PASSWORD"),
		saltAPIEauth:    getEnvDefaultSync("SALT_API_EAUTH", "file"),
		stopChan:        make(chan struct{}),
		syncInterval:    syncInterval,
	}
}

// getEnvDefaultSync 获取环境变量，如果不存在则返回默认值
func getEnvDefaultSync(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Start 启动同步服务
func (s *NodeMetricsSyncService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Printf("[NodeMetricsSyncService] 启动节点指标同步服务，同步间隔: %v", s.syncInterval)

	go s.syncWorker()
}

// Stop 停止同步服务
func (s *NodeMetricsSyncService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopChan)
	log.Printf("[NodeMetricsSyncService] 已停止节点指标同步服务")
}

// syncWorker 同步工作器
func (s *NodeMetricsSyncService) syncWorker() {
	// 立即执行一次同步
	s.syncAllNodeMetrics()

	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			log.Printf("[NodeMetricsSyncService] 同步工作器已停止")
			return
		case <-ticker.C:
			s.syncAllNodeMetrics()
		}
	}
}

// syncAllNodeMetrics 同步所有节点的指标数据
func (s *NodeMetricsSyncService) syncAllNodeMetrics() {
	log.Printf("[NodeMetricsSyncService] 开始同步所有节点指标...")

	// 创建 Salt API 客户端
	client := s.newSaltAPIClient()
	if err := client.authenticate(); err != nil {
		log.Printf("[NodeMetricsSyncService] Salt API 认证失败: %v", err)
		return
	}

	// 获取所有在线的 Minion 列表
	upMinions, err := s.getUpMinions(client)
	if err != nil {
		log.Printf("[NodeMetricsSyncService] 获取在线 Minion 列表失败: %v", err)
		return
	}

	if len(upMinions) == 0 {
		log.Printf("[NodeMetricsSyncService] 没有在线的 Minion")
		return
	}

	log.Printf("[NodeMetricsSyncService] 发现 %d 个在线 Minion，开始采集指标...", len(upMinions))

	// 批量获取所有节点的 CPU/内存指标
	metricsMap := s.batchGetCPUMemoryMetrics(client, upMinions)

	// 保存到数据库和 Redis
	savedCount := 0
	for minionID, metrics := range metricsMap {
		if err := s.saveMetrics(minionID, metrics); err != nil {
			log.Printf("[NodeMetricsSyncService] 保存 %s 的指标失败: %v", minionID, err)
		} else {
			savedCount++
		}
	}

	log.Printf("[NodeMetricsSyncService] 同步完成，成功保存 %d/%d 个节点的指标", savedCount, len(upMinions))
}

// getUpMinions 获取所有在线的 Minion ID 列表
func (s *NodeMetricsSyncService) getUpMinions(client *syncSaltAPIClient) ([]string, error) {
	resp, err := client.makeRunner("manage.status", map[string]interface{}{
		"timeout":            5,
		"gather_job_timeout": 3,
	})
	if err != nil {
		return nil, err
	}

	var up []string
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if u, ok := m["up"].([]interface{}); ok {
				for _, v := range u {
					if s, ok := v.(string); ok {
						up = append(up, s)
					}
				}
			}
		}
	}
	return up, nil
}

// batchGetCPUMemoryMetrics 批量获取所有节点的 CPU/内存指标
func (s *NodeMetricsSyncService) batchGetCPUMemoryMetrics(client *syncSaltAPIClient, minionIDs []string) map[string]*NodeMetricsData {
	metricsMap := make(map[string]*NodeMetricsData)

	// 使用 Salt 的批量执行功能一次性获取所有节点的数据
	// 从嵌入脚本文件读取，避免硬编码
	script, err := scripts.GetCPUMemoryLoadAvgScript()
	if err != nil {
		log.Printf("[NodeMetricsSyncService] 读取脚本文件失败: %v", err)
		return metricsMap
	}

	payload := map[string]interface{}{
		"client": "local",
		"tgt":    "*", // 目标所有节点
		"fun":    "cmd.run",
		"arg":    []interface{}{script},
		"kwarg": map[string]interface{}{
			"timeout":      30,
			"python_shell": true, // 必须启用 shell 模式以支持管道和复杂命令
		},
	}

	resp, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		log.Printf("[NodeMetricsSyncService] 批量执行命令失败: %v", err)
		return metricsMap
	}

	// 解析响应
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			for minionID, output := range m {
				outputStr, ok := output.(string)
				if !ok || outputStr == "" {
					continue
				}

				metrics := s.parseCPUMemoryOutput(outputStr)
				if metrics != nil {
					metricsMap[minionID] = metrics
				}
			}
		}
	}

	return metricsMap
}

// NodeMetricsData 节点指标数据
type NodeMetricsData struct {
	CPUUsagePercent    float64
	MemoryTotalGB      float64
	MemoryUsedGB       float64
	MemoryAvailableGB  float64
	MemoryUsagePercent float64
	LoadAvg            string
}

// parseCPUMemoryOutput 解析 CPU/内存输出
func (s *NodeMetricsSyncService) parseCPUMemoryOutput(output string) *NodeMetricsData {
	output = strings.TrimSpace(output)
	// 处理多行输出，只取最后一行
	lines := strings.Split(output, "\n")
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// 格式: cpu_percent|mem_total_kb|mem_avail_kb|load_avg
	parts := strings.Split(lastLine, "|")
	if len(parts) < 4 {
		return nil
	}

	metrics := &NodeMetricsData{}

	// CPU 使用率
	if cpuPercent, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
		metrics.CPUUsagePercent = cpuPercent
	}

	// 内存总量 (KB -> GB)
	var memTotalKB float64
	if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil && val > 0 {
		memTotalKB = val
		metrics.MemoryTotalGB = memTotalKB / 1024 / 1024
	}

	// 内存可用量 (KB -> GB) 及使用率计算
	if memAvailKB, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil && memTotalKB > 0 {
		metrics.MemoryAvailableGB = memAvailKB / 1024 / 1024
		memUsedKB := memTotalKB - memAvailKB
		metrics.MemoryUsedGB = memUsedKB / 1024 / 1024
		metrics.MemoryUsagePercent = (memUsedKB / memTotalKB) * 100
	}

	// 负载平均值
	if len(parts) >= 4 {
		metrics.LoadAvg = strings.TrimSpace(parts[3])
	}

	return metrics
}

// saveMetrics 保存指标到数据库和 Redis
func (s *NodeMetricsSyncService) saveMetrics(minionID string, data *NodeMetricsData) error {
	if s.db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 更新或创建 NodeMetricsLatest 记录
	var existing models.NodeMetricsLatest
	result := s.db.Where("minion_id = ?", minionID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		newMetrics := models.NodeMetricsLatest{
			MinionID:           minionID,
			Timestamp:          time.Now(),
			CPUUsagePercent:    data.CPUUsagePercent,
			CPULoadAvg:         data.LoadAvg,
			MemoryTotalGB:      data.MemoryTotalGB,
			MemoryUsedGB:       data.MemoryUsedGB,
			MemoryAvailableGB:  data.MemoryAvailableGB,
			MemoryUsagePercent: data.MemoryUsagePercent,
		}
		if err := s.db.Create(&newMetrics).Error; err != nil {
			return fmt.Errorf("创建指标记录失败: %v", err)
		}
	} else if result.Error != nil {
		return fmt.Errorf("查询指标记录失败: %v", result.Error)
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"timestamp":            time.Now(),
			"cpu_usage_percent":    data.CPUUsagePercent,
			"cpu_load_avg":         data.LoadAvg,
			"memory_total_gb":      data.MemoryTotalGB,
			"memory_used_gb":       data.MemoryUsedGB,
			"memory_available_gb":  data.MemoryAvailableGB,
			"memory_usage_percent": data.MemoryUsagePercent,
		}
		if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新指标记录失败: %v", err)
		}
	}

	// 同步到 Redis（如果 Redis 可用）
	if s.redisClient != nil {
		s.saveMetricsToRedis(minionID, data)
	}

	return nil
}

// saveMetricsToRedis 保存指标到 Redis
func (s *NodeMetricsSyncService) saveMetricsToRedis(minionID string, data *NodeMetricsData) {
	ctx := context.Background()
	key := fmt.Sprintf("node_metrics:%s", minionID)

	metricsJSON, err := json.Marshal(map[string]interface{}{
		"minion_id":            minionID,
		"cpu_usage_percent":    data.CPUUsagePercent,
		"cpu_load_avg":         data.LoadAvg,
		"memory_total_gb":      data.MemoryTotalGB,
		"memory_used_gb":       data.MemoryUsedGB,
		"memory_available_gb":  data.MemoryAvailableGB,
		"memory_usage_percent": data.MemoryUsagePercent,
		"timestamp":            time.Now().Format(time.RFC3339),
	})
	if err != nil {
		log.Printf("[NodeMetricsSyncService] 序列化 Redis 数据失败: %v", err)
		return
	}

	// 设置过期时间为同步间隔的 3 倍，确保数据不会因为同步延迟而丢失
	expiration := s.syncInterval * 3
	if err := s.redisClient.Set(ctx, key, metricsJSON, expiration).Err(); err != nil {
		log.Printf("[NodeMetricsSyncService] 写入 Redis 失败: %v", err)
	}
}

// newSaltAPIClient 创建 Salt API 客户端
func (s *NodeMetricsSyncService) newSaltAPIClient() *syncSaltAPIClient {
	return &syncSaltAPIClient{
		baseURL:  s.saltMasterURL,
		username: s.saltAPIUsername,
		password: s.saltAPIPassword,
		eauth:    s.saltAPIEauth,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

// authenticate Salt API 认证
func (c *syncSaltAPIClient) authenticate() error {
	payload := map[string]interface{}{
		"username": c.username,
		"password": c.password,
		"eauth":    c.eauth,
	}

	resp, err := c.makeRequest("/login", "POST", payload)
	if err != nil {
		return err
	}

	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if token, ok := m["token"].(string); ok && token != "" {
				c.token = token
				return nil
			}
		}
	}

	return fmt.Errorf("认证失败：无法获取 token")
}

// makeRequest 发起 HTTP 请求
func (c *syncSaltAPIClient) makeRequest(endpoint, method string, payload map[string]interface{}) (map[string]interface{}, error) {
	url := c.baseURL + endpoint

	var req *http.Request
	var err error

	if payload != nil {
		body, _ := json.Marshal(payload)
		req, err = http.NewRequest(method, url, strings.NewReader(string(body)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("X-Auth-Token", c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// makeRunner 执行 Salt Runner
func (c *syncSaltAPIClient) makeRunner(fun string, kwarg map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"client": "runner",
		"fun":    fun,
	}
	if kwarg != nil {
		payload["kwarg"] = kwarg
	}
	return c.makeRequest("/", "POST", payload)
}

// 全局实例
var (
	nodeMetricsSyncService *NodeMetricsSyncService
	nodeMetricsSyncOnce    sync.Once
)

// GetNodeMetricsSyncService 获取节点指标同步服务单例
func GetNodeMetricsSyncService() *NodeMetricsSyncService {
	nodeMetricsSyncOnce.Do(func() {
		nodeMetricsSyncService = NewNodeMetricsSyncService()
	})
	return nodeMetricsSyncService
}

// StartNodeMetricsSync 启动节点指标同步（便捷函数）
func StartNodeMetricsSync() {
	GetNodeMetricsSyncService().Start()
}

// StopNodeMetricsSync 停止节点指标同步（便捷函数）
func StopNodeMetricsSync() {
	if nodeMetricsSyncService != nil {
		nodeMetricsSyncService.Stop()
	}
}
