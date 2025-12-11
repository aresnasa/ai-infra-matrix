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
	"sync/atomic"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/scripts"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NodeMetricsSyncService 节点指标同步服务
// 定期使用 Salt 命令获取所有节点的 CPU/内存/GPU/IB 等指标，并同步到 Redis 和数据库
// 设计原则：
// 1. 每次采集都异步写入 Redis（实时性）
// 2. 数据库批量更新，降低写入频率（每 N 次同步周期才写一次数据库）
// 3. 监控数据允许丢失，不影响核心业务
type NodeMetricsSyncService struct {
	db              *gorm.DB
	redisClient     *redis.Client
	saltMasterURL   string
	saltAPIUsername string
	saltAPIPassword string
	saltAPIEauth    string
	stopChan        chan struct{}
	syncInterval    time.Duration // Redis 同步间隔，默认 60 秒
	dbSyncRatio     int           // 数据库同步比率：每 N 次 Redis 同步才写一次数据库
	syncCounter     int64         // 同步计数器（原子操作）
	mu              sync.RWMutex
	running         bool

	// 异步写入缓冲区
	metricsBuffer     map[string]*NodeMetricsData
	metricsBufferLock sync.RWMutex

	// 并发控制
	maxConcurrent int           // 最大并发数
	semaphore     chan struct{} // 信号量
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

	// 数据库同步比率：默认每 5 次 Redis 同步才写一次数据库（即 5 分钟）
	dbSyncRatio := 5
	if envRatio := os.Getenv("NODE_METRICS_DB_SYNC_RATIO"); envRatio != "" {
		if ratio, err := strconv.Atoi(envRatio); err == nil && ratio > 0 {
			dbSyncRatio = ratio
		}
	}

	// 最大并发数
	maxConcurrent := 10
	if envConcurrent := os.Getenv("NODE_METRICS_MAX_CONCURRENT"); envConcurrent != "" {
		if concurrent, err := strconv.Atoi(envConcurrent); err == nil && concurrent > 0 {
			maxConcurrent = concurrent
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
		dbSyncRatio:     dbSyncRatio,
		syncCounter:     0,
		metricsBuffer:   make(map[string]*NodeMetricsData),
		maxConcurrent:   maxConcurrent,
		semaphore:       make(chan struct{}, maxConcurrent),
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

	log.Printf("[NodeMetricsSyncService] 启动节点指标同步服务，Redis同步间隔: %v, 数据库同步比率: 1/%d",
		s.syncInterval, s.dbSyncRatio)

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

	// 递增同步计数器
	counter := atomic.AddInt64(&s.syncCounter, 1)

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

	// 异步保存到 Redis（每次都执行）
	var wg sync.WaitGroup
	redisSuccessCount := int32(0)

	for minionID, metrics := range metricsMap {
		wg.Add(1)
		go func(id string, m *NodeMetricsData) {
			defer wg.Done()

			// 使用信号量控制并发
			s.semaphore <- struct{}{}
			defer func() { <-s.semaphore }()

			// 保存到 Redis（允许失败，不阻塞）
			if s.redisClient != nil {
				s.saveMetricsToRedisAsync(id, m)
				atomic.AddInt32(&redisSuccessCount, 1)
			}

			// 更新缓冲区（用于后续数据库批量写入）
			s.metricsBufferLock.Lock()
			s.metricsBuffer[id] = m
			s.metricsBufferLock.Unlock()
		}(minionID, metrics)
	}

	wg.Wait()

	log.Printf("[NodeMetricsSyncService] Redis 同步完成: %d/%d 成功", redisSuccessCount, len(metricsMap))

	// 判断是否需要写入数据库（按比率控制）
	if counter%int64(s.dbSyncRatio) == 0 {
		s.flushMetricsToDatabase()
	}
}

// saveMetricsToRedisAsync 异步保存指标到 Redis（允许失败）
func (s *NodeMetricsSyncService) saveMetricsToRedisAsync(minionID string, data *NodeMetricsData) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

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
		// 序列化失败，静默忽略
		return
	}

	// 设置过期时间为同步间隔的 3 倍
	expiration := s.syncInterval * 3
	if err := s.redisClient.Set(ctx, key, metricsJSON, expiration).Err(); err != nil {
		// Redis 写入失败，静默忽略（监控数据允许丢失）
		log.Printf("[NodeMetricsSyncService] Redis 写入失败 (允许): %v", err)
	}
}

// flushMetricsToDatabase 批量刷新指标到数据库
func (s *NodeMetricsSyncService) flushMetricsToDatabase() {
	s.metricsBufferLock.Lock()
	buffer := s.metricsBuffer
	s.metricsBuffer = make(map[string]*NodeMetricsData) // 重置缓冲区
	s.metricsBufferLock.Unlock()

	if len(buffer) == 0 {
		return
	}

	log.Printf("[NodeMetricsSyncService] 开始批量写入数据库，共 %d 条记录...", len(buffer))

	// 准备批量 upsert 数据
	records := make([]models.NodeMetricsLatest, 0, len(buffer))
	now := time.Now()

	for minionID, data := range buffer {
		records = append(records, models.NodeMetricsLatest{
			MinionID:           minionID,
			Timestamp:          now,
			CPUUsagePercent:    data.CPUUsagePercent,
			CPULoadAvg:         data.LoadAvg,
			MemoryTotalGB:      data.MemoryTotalGB,
			MemoryUsedGB:       data.MemoryUsedGB,
			MemoryAvailableGB:  data.MemoryAvailableGB,
			MemoryUsagePercent: data.MemoryUsagePercent,
		})
	}

	// 使用批量 upsert（ON CONFLICT DO UPDATE）减少数据库操作次数
	// 分批处理，每批最多 100 条
	batchSize := 100
	successCount := 0
	failCount := 0

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		// 使用 GORM 的 Clauses 进行 upsert
		err := s.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "minion_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"timestamp",
				"cpu_usage_percent",
				"cpu_load_avg",
				"memory_total_gb",
				"memory_used_gb",
				"memory_available_gb",
				"memory_usage_percent",
			}),
		}).CreateInBatches(batch, batchSize).Error

		if err != nil {
			log.Printf("[NodeMetricsSyncService] 批量写入数据库失败 (batch %d-%d): %v", i, end, err)
			failCount += len(batch)
		} else {
			successCount += len(batch)
		}
	}

	log.Printf("[NodeMetricsSyncService] 数据库批量写入完成: 成功 %d, 失败 %d", successCount, failCount)
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

// ============================================================================
// Redis 读取接口（供外部调用获取实时指标）
// ============================================================================

// GetMetricsFromRedis 从 Redis 获取单个节点的指标（优先使用）
func (s *NodeMetricsSyncService) GetMetricsFromRedis(minionID string) (*NodeMetricsData, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("Redis 未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("node_metrics:%s", minionID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	metrics := &NodeMetricsData{}
	if v, ok := result["cpu_usage_percent"].(float64); ok {
		metrics.CPUUsagePercent = v
	}
	if v, ok := result["memory_total_gb"].(float64); ok {
		metrics.MemoryTotalGB = v
	}
	if v, ok := result["memory_used_gb"].(float64); ok {
		metrics.MemoryUsedGB = v
	}
	if v, ok := result["memory_available_gb"].(float64); ok {
		metrics.MemoryAvailableGB = v
	}
	if v, ok := result["memory_usage_percent"].(float64); ok {
		metrics.MemoryUsagePercent = v
	}
	if v, ok := result["cpu_load_avg"].(string); ok {
		metrics.LoadAvg = v
	}

	return metrics, nil
}

// GetAllMetricsFromRedis 从 Redis 获取所有节点的指标
func (s *NodeMetricsSyncService) GetAllMetricsFromRedis() (map[string]*NodeMetricsData, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("Redis 未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用 SCAN 命令获取所有 node_metrics:* 键
	result := make(map[string]*NodeMetricsData)
	iter := s.redisClient.Scan(ctx, 0, "node_metrics:*", 1000).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		minionID := strings.TrimPrefix(key, "node_metrics:")

		data, err := s.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var metricsMap map[string]interface{}
		if err := json.Unmarshal([]byte(data), &metricsMap); err != nil {
			continue
		}

		metrics := &NodeMetricsData{}
		if v, ok := metricsMap["cpu_usage_percent"].(float64); ok {
			metrics.CPUUsagePercent = v
		}
		if v, ok := metricsMap["memory_total_gb"].(float64); ok {
			metrics.MemoryTotalGB = v
		}
		if v, ok := metricsMap["memory_used_gb"].(float64); ok {
			metrics.MemoryUsedGB = v
		}
		if v, ok := metricsMap["memory_available_gb"].(float64); ok {
			metrics.MemoryAvailableGB = v
		}
		if v, ok := metricsMap["memory_usage_percent"].(float64); ok {
			metrics.MemoryUsagePercent = v
		}
		if v, ok := metricsMap["cpu_load_avg"].(string); ok {
			metrics.LoadAvg = v
		}

		result[minionID] = metrics
	}

	if err := iter.Err(); err != nil {
		return result, err
	}

	return result, nil
}

// GetSyncStatus 获取同步服务状态
func (s *NodeMetricsSyncService) GetSyncStatus() map[string]interface{} {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	s.metricsBufferLock.RLock()
	bufferSize := len(s.metricsBuffer)
	s.metricsBufferLock.RUnlock()

	return map[string]interface{}{
		"running":         running,
		"sync_interval":   s.syncInterval.String(),
		"db_sync_ratio":   s.dbSyncRatio,
		"sync_counter":    atomic.LoadInt64(&s.syncCounter),
		"buffer_size":     bufferSize,
		"max_concurrent":  s.maxConcurrent,
		"next_db_sync_in": s.dbSyncRatio - int(atomic.LoadInt64(&s.syncCounter)%int64(s.dbSyncRatio)),
	}
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

// GetNodeMetricsFromRedis 从 Redis 获取节点指标（全局便捷函数）
func GetNodeMetricsFromRedis(minionID string) (*NodeMetricsData, error) {
	return GetNodeMetricsSyncService().GetMetricsFromRedis(minionID)
}

// GetAllNodeMetricsFromRedis 从 Redis 获取所有节点指标（全局便捷函数）
func GetAllNodeMetricsFromRedis() (map[string]*NodeMetricsData, error) {
	return GetNodeMetricsSyncService().GetAllMetricsFromRedis()
}
