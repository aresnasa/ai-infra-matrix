package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/scripts"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SaltStackHandler 处理SaltStack相关的API请求
type SaltStackHandler struct {
	config             *config.Config
	cache              *redis.Client
	metricsService     *services.MetricsService
	masterPool         *SaltMasterPool              // 多 Master 连接池
	minionGroupService *services.MinionGroupService // Minion 分组服务
}

// SaltMasterPool 管理多个 Salt Master 的连接池
type SaltMasterPool struct {
	masters       []SaltMasterConfig
	healthStatus  map[string]bool      // Master URL -> 健康状态
	lastCheck     map[string]time.Time // Master URL -> 上次检查时间
	mu            sync.RWMutex
	checkInterval time.Duration
}

// SaltMasterConfig 单个 Salt Master 配置
type SaltMasterConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Eauth    string `json:"eauth"`
	Priority int    `json:"priority"` // 优先级，数字越小优先级越高
}

// NewSaltMasterPool 创建 Master 连接池
func NewSaltMasterPool() *SaltMasterPool {
	pool := &SaltMasterPool{
		masters:       make([]SaltMasterConfig, 0),
		healthStatus:  make(map[string]bool),
		lastCheck:     make(map[string]time.Time),
		checkInterval: 180 * time.Second, // Master 健康检查间隔: 3分钟
	}
	pool.loadMastersFromEnv()
	return pool
}

// loadMastersFromEnv 从环境变量加载 Master 配置
// 支持两种配置方式：
// 1. 单 Master: SALTSTACK_MASTER_URL 或 SALT_MASTER_HOST
// 2. 多 Master: SALT_MASTERS_CONFIG (JSON 格式)
func (p *SaltMasterPool) loadMastersFromEnv() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 方式1: 从 JSON 配置加载多 Master
	if configJSON := os.Getenv("SALT_MASTERS_CONFIG"); configJSON != "" {
		var configs []SaltMasterConfig
		if err := json.Unmarshal([]byte(configJSON), &configs); err == nil && len(configs) > 0 {
			p.masters = configs
			// 按优先级排序
			sort.Slice(p.masters, func(i, j int) bool {
				return p.masters[i].Priority < p.masters[j].Priority
			})
			log.Printf("[SaltMasterPool] 从 SALT_MASTERS_CONFIG 加载了 %d 个 Master", len(p.masters))
			return
		}
	}

	// 方式2: 从逗号分隔的 URL 列表加载
	if urlList := os.Getenv("SALT_MASTER_URLS"); urlList != "" {
		urls := strings.Split(urlList, ",")
		username := getEnv("SALT_API_USERNAME", "saltapi")
		password := os.Getenv("SALT_API_PASSWORD")
		eauth := getEnv("SALT_API_EAUTH", "file")

		for i, u := range urls {
			u = strings.TrimSpace(u)
			if u != "" {
				p.masters = append(p.masters, SaltMasterConfig{
					URL:      u,
					Username: username,
					Password: password,
					Eauth:    eauth,
					Priority: i,
				})
			}
		}
		if len(p.masters) > 0 {
			log.Printf("[SaltMasterPool] 从 SALT_MASTER_URLS 加载了 %d 个 Master", len(p.masters))
			return
		}
	}

	// 方式3: 单 Master 兼容模式
	var masterURL string
	if base := strings.TrimSpace(os.Getenv("SALTSTACK_MASTER_URL")); base != "" {
		if parsed, err := url.Parse(base); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			masterURL = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		} else {
			masterURL = base
		}
	} else {
		scheme := getEnv("SALT_API_SCHEME", "http")
		host := getEnv("SALT_MASTER_HOST", "saltstack")
		port := getEnv("SALT_API_PORT", "8002")
		masterURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	p.masters = []SaltMasterConfig{{
		URL:      masterURL,
		Username: getEnv("SALT_API_USERNAME", "saltapi"),
		Password: os.Getenv("SALT_API_PASSWORD"),
		Eauth:    getEnv("SALT_API_EAUTH", "file"),
		Priority: 0,
	}}
	log.Printf("[SaltMasterPool] 使用单 Master 模式: %s", masterURL)
}

// GetHealthyMaster 获取一个健康的 Master，支持故障转移
func (p *SaltMasterPool) GetHealthyMaster() (*SaltMasterConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.masters) == 0 {
		return nil, errors.New("no salt masters configured")
	}

	// 按优先级顺序尝试每个 Master
	for i := range p.masters {
		master := &p.masters[i]
		// 如果有健康状态记录且健康，直接返回
		if healthy, exists := p.healthStatus[master.URL]; exists && healthy {
			return master, nil
		}
		// 如果没有记录或上次检查时间过久，尝试这个 Master
		if lastCheck, exists := p.lastCheck[master.URL]; !exists || time.Since(lastCheck) > p.checkInterval {
			return master, nil
		}
	}

	// 所有 Master 都不健康，返回第一个（让调用方重试）
	return &p.masters[0], nil
}

// GetAllMasters 获取所有配置的 Master（用于并行探测）
func (p *SaltMasterPool) GetAllMasters() []SaltMasterConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]SaltMasterConfig, len(p.masters))
	copy(result, p.masters)
	return result
}

// UpdateHealth 更新 Master 健康状态
func (p *SaltMasterPool) UpdateHealth(url string, healthy bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthStatus[url] = healthy
	p.lastCheck[url] = time.Now()
	if healthy {
		log.Printf("[SaltMasterPool] Master %s 健康检查通过", url)
	} else {
		log.Printf("[SaltMasterPool] Master %s 健康检查失败", url)
	}
}

// GetMasterCount 获取配置的 Master 数量
func (p *SaltMasterPool) GetMasterCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.masters)
}

// GetHealthyCount 获取健康的 Master 数量
func (p *SaltMasterPool) GetHealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	count := 0
	for _, healthy := range p.healthStatus {
		if healthy {
			count++
		}
	}
	return count
}

// NewSaltStackHandler 创建新的SaltStack处理器
func NewSaltStackHandler(cfg *config.Config, cache *redis.Client) *SaltStackHandler {
	handler := &SaltStackHandler{
		config:             cfg,
		cache:              cache,
		metricsService:     services.NewMetricsService(),
		masterPool:         NewSaltMasterPool(),
		minionGroupService: services.NewMinionGroupService(),
	}
	// 启动后台健康检查
	go handler.startHealthCheck()
	return handler
}

// startHealthCheck 启动后台健康检查
func (h *SaltStackHandler) startHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 首次立即检查
	h.checkAllMastersHealth()

	for range ticker.C {
		h.checkAllMastersHealth()
	}
}

// checkAllMastersHealth 检查所有 Master 的健康状态
func (h *SaltStackHandler) checkAllMastersHealth() {
	masters := h.masterPool.GetAllMasters()
	if len(masters) <= 1 {
		return // 单 Master 模式不需要健康检查
	}

	for _, master := range masters {
		go func(m SaltMasterConfig) {
			client := h.newSaltAPIClientForMaster(&m)
			// 尝试简单的 GET / 请求来检查连通性
			_, err := client.makeRequest("/", "GET", nil)
			h.masterPool.UpdateHealth(m.URL, err == nil)
		}(master)
	}
}

// SaltStackStatus SaltStack状态信息
type SaltStackStatus struct {
	Status           string            `json:"status"`
	MasterVersion    string            `json:"master_version"`
	APIVersion       string            `json:"api_version"`
	Uptime           int64             `json:"uptime"`
	ConnectedMinions int               `json:"connected_minions"`
	AcceptedKeys     []string          `json:"accepted_keys"`
	UnacceptedKeys   []string          `json:"unaccepted_keys"`
	RejectedKeys     []string          `json:"rejected_keys"`
	Services         map[string]string `json:"services"`
	LastUpdated      time.Time         `json:"last_updated"`
	Demo             bool              `json:"demo,omitempty"`
	// 前端兼容字段
	MasterStatus      string  `json:"master_status,omitempty"`
	APIStatus         string  `json:"api_status,omitempty"`
	MinionsUp         int     `json:"minions_up,omitempty"`
	MinionsDown       int     `json:"minions_down,omitempty"`
	SaltVersion       string  `json:"salt_version,omitempty"`
	ConfigFile        string  `json:"config_file,omitempty"`
	LogLevel          string  `json:"log_level,omitempty"`
	CPUUsage          float64 `json:"cpu_usage"`          // 不使用 omitempty，确保返回 0
	MemoryUsage       int     `json:"memory_usage"`       // 不使用 omitempty，确保返回 0
	ActiveConnections int     `json:"active_connections"` // 不使用 omitempty，确保返回 0
	NetworkBandwidth  float64 `json:"network_bandwidth"`  // 不使用 omitempty，确保返回 0 (Mbps)
	MetricsSource     string  `json:"metrics_source"`     // 监控数据来源：victoriametrics/docker/salt/none
	// 多 Master 高可用字段
	ActiveMasterURL  string             `json:"active_master_url,omitempty"`
	MasterCount      int                `json:"master_count,omitempty"`
	HealthyMasters   int                `json:"healthy_masters,omitempty"`
	MasterHealthInfo []MasterHealthInfo `json:"master_health_info,omitempty"`
}

// MasterHealthInfo 单个 Master 的健康信息
type MasterHealthInfo struct {
	URL       string    `json:"url"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check,omitempty"`
	Priority  int       `json:"priority"`
}

// SaltMinion Salt Minion信息
type SaltMinion struct {
	ID               string                 `json:"id"`
	Status           string                 `json:"status"`
	OS               string                 `json:"os"`
	OSVersion        string                 `json:"os_version"`
	Architecture     string                 `json:"architecture"`
	Arch             string                 `json:"arch,omitempty"`
	SaltVersion      string                 `json:"salt_version,omitempty"`
	LastSeen         time.Time              `json:"last_seen"`
	Grains           map[string]interface{} `json:"grains"`
	Pillar           map[string]interface{} `json:"pillar,omitempty"`
	Group            string                 `json:"group,omitempty"` // 分组名称
	KernelVersion    string                 `json:"kernel_version,omitempty"`
	GPUDriverVersion string                 `json:"gpu_driver_version,omitempty"`
	CUDAVersion      string                 `json:"cuda_version,omitempty"`
	GPUCount         int                    `json:"gpu_count,omitempty"`
	GPUModel         string                 `json:"gpu_model,omitempty"`
	NPUVersion       string                 `json:"npu_version,omitempty"`
	NPUCount         int                    `json:"npu_count,omitempty"`
	NPUModel         string                 `json:"npu_model,omitempty"`
	// IB (InfiniBand) 信息
	IBStatus string `json:"ib_status,omitempty"` // active, inactive, not_installed
	IBCount  int    `json:"ib_count,omitempty"`
	IBRate   string `json:"ib_rate,omitempty"` // 如 "200 Gb/sec (4X HDR)"
	// CPU/内存使用率信息（实时采集）
	CPUUsagePercent    float64 `json:"cpu_usage_percent,omitempty"`
	MemoryUsagePercent float64 `json:"memory_usage_percent,omitempty"`
	MemoryTotalGB      float64 `json:"memory_total_gb,omitempty"`
	MemoryUsedGB       float64 `json:"memory_used_gb,omitempty"`
	// 数据来源标识：realtime（实时从Salt获取）, cached（从数据库读取）, unavailable（无法获取）
	DataSource string `json:"data_source,omitempty"`
}

// SaltJob Salt作业信息
type SaltJob struct {
	JID          string                 `json:"jid"`
	Function     string                 `json:"function"`
	Arguments    []string               `json:"arguments"`
	Target       string                 `json:"target"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Status       string                 `json:"status"`
	Result       map[string]interface{} `json:"result,omitempty"`
	User         string                 `json:"user"`
	TaskID       string                 `json:"task_id,omitempty"` // 前端生成的任务ID，用于用户追踪
	SuccessCount int                    `json:"success_count"`     // 成功节点数量
	FailedCount  int                    `json:"failed_count"`      // 失败节点数量
}

// saltAPIClient SaltStack API客户端
type saltAPIClient struct {
	baseURL  string
	token    string
	client   *http.Client
	username string
	password string
	eauth    string
}

// newSaltAPIClient 创建Salt API客户端（使用多 Master 故障转移）
func (h *SaltStackHandler) newSaltAPIClient() *saltAPIClient {
	master, err := h.masterPool.GetHealthyMaster()
	if err != nil {
		log.Printf("[SaltStack] 获取健康 Master 失败: %v, 使用默认配置", err)
		return h.newSaltAPIClientDefault()
	}
	return h.newSaltAPIClientForMaster(master)
}

// newSaltAPIClientDefault 使用默认配置创建客户端（兼容旧逻辑）
func (h *SaltStackHandler) newSaltAPIClientDefault() *saltAPIClient {
	timeoutSec := h.getAPITimeout()
	return &saltAPIClient{
		baseURL:  h.getSaltAPIURL(),
		username: getEnv("SALT_API_USERNAME", "saltapi"),
		password: os.Getenv("SALT_API_PASSWORD"),
		eauth:    getEnv("SALT_API_EAUTH", "file"),
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

// newSaltAPIClientForMaster 为指定 Master 创建客户端
func (h *SaltStackHandler) newSaltAPIClientForMaster(master *SaltMasterConfig) *saltAPIClient {
	timeoutSec := h.getAPITimeout()
	return &saltAPIClient{
		baseURL:  master.URL,
		username: master.Username,
		password: master.Password,
		eauth:    master.Eauth,
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

// getAPITimeout 获取 API 超时配置
func (h *SaltStackHandler) getAPITimeout() int {
	timeoutSec := 30
	if t := os.Getenv("SALT_API_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil && d > 0 {
			timeoutSec = int(d.Seconds())
		} else if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			timeoutSec = parsed
		}
	}
	return timeoutSec
}

// newSaltAPIClientWithTimeout 创建带自定义超时的Salt API客户端
func (h *SaltStackHandler) newSaltAPIClientWithTimeout(timeout time.Duration) *saltAPIClient {
	master, _ := h.masterPool.GetHealthyMaster()
	baseURL := h.getSaltAPIURL()
	username := getEnv("SALT_API_USERNAME", "saltapi")
	password := os.Getenv("SALT_API_PASSWORD")
	eauth := getEnv("SALT_API_EAUTH", "file")

	if master != nil {
		baseURL = master.URL
		username = master.Username
		password = master.Password
		eauth = master.Eauth
	}

	return &saltAPIClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		eauth:    eauth,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

// newSaltAPIClientWithFailover 带故障转移的客户端创建（尝试所有 Master）
func (h *SaltStackHandler) newSaltAPIClientWithFailover() (*saltAPIClient, error) {
	masters := h.masterPool.GetAllMasters()
	if len(masters) == 0 {
		return nil, errors.New("no salt masters configured")
	}

	timeoutSec := h.getAPITimeout()

	// 尝试每个 Master
	for _, master := range masters {
		client := &saltAPIClient{
			baseURL:  master.URL,
			username: master.Username,
			password: master.Password,
			eauth:    master.Eauth,
			client: &http.Client{
				Timeout: time.Duration(timeoutSec) * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
					DisableKeepAlives:   false,
					DialContext: (&net.Dialer{
						Timeout:   5 * time.Second, // 故障转移时使用较短连接超时
						KeepAlive: 30 * time.Second,
					}).DialContext,
				},
			},
		}

		// 尝试认证
		if err := client.authenticate(); err == nil {
			h.masterPool.UpdateHealth(master.URL, true)
			return client, nil
		} else {
			h.masterPool.UpdateHealth(master.URL, false)
			log.Printf("[SaltStack] Master %s 认证失败: %v, 尝试下一个", master.URL, err)
		}
	}

	return nil, errors.New("all salt masters are unavailable")
}

// getSaltAPIURL 获取Salt API URL
func (h *SaltStackHandler) getSaltAPIURL() string {
	// 优先使用完整URL
	if base := strings.TrimSpace(os.Getenv("SALTSTACK_MASTER_URL")); base != "" {
		// 仅保留协议+主机(含端口)，剥离可能误配的路径（例如 '/app'）
		if parsed, err := url.Parse(base); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			// 注意：如果 Host 为空但包含在 Opaque 中（罕见），则回退原值
			return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		}
		return base
	}
	// 否则按协议/主机/端口组合
	scheme := getEnv("SALT_API_SCHEME", "http")
	host := getEnv("SALT_MASTER_HOST", "saltstack")
	port := getEnv("SALT_API_PORT", "8002")
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// authenticate 向Salt API认证
func (c *saltAPIClient) authenticate() error {
	// 优先使用客户端存储的凭证，否则从环境变量获取（兼容旧代码）
	username := c.username
	password := c.password
	eauth := c.eauth

	if username == "" {
		username = getEnv("SALT_API_USERNAME", "saltapi")
	}
	if password == "" {
		password = os.Getenv("SALT_API_PASSWORD")
	}
	if eauth == "" {
		eauth = getEnv("SALT_API_EAUTH", "file")
	}

	// 如果未配置密码，则尝试无认证直接使用
	if password == "" {
		return nil
	}

	// Try JSON login first
	payload := map[string]string{
		"username": username,
		"password": password,
		"eauth":    eauth,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.baseURL+"/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if arr, ok := result["return"].([]interface{}); ok && len(arr) > 0 {
				if m, ok := arr[0].(map[string]interface{}); ok {
					if t, ok := m["token"].(string); ok && t != "" {
						c.token = t
						return nil
					}
				}
			}
		}
		// fallthrough to form mode if no token
	} else {
		// Close body before retry
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Fallback: some deployments expect form-encoded login
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set("eauth", eauth)
	req2, err := http.NewRequest("POST", c.baseURL+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp2, err := c.client.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return fmt.Errorf("salt api login failed: status %d", resp2.StatusCode)
	}
	result = map[string]interface{}{}
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
		return err
	}
	// 解析 token: 形如 {"return": [{"token": "...", "expire": 1234567890}]}
	if arr, ok := result["return"].([]interface{}); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]interface{}); ok {
			if t, ok := m["token"].(string); ok && t != "" {
				c.token = t
				return nil
			}
		}
	}
	return fmt.Errorf("salt api login: token not found")
}

// makeRequest 发送请求到Salt API
func (c *saltAPIClient) makeRequest(endpoint string, method string, data interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Auth-Token", c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("salt api error: %d %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// makeRunner 调用 runner 客户端
func (c *saltAPIClient) makeRunner(fun string, kwarg map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"client": "runner",
		"fun":    fun,
	}
	if kwarg != nil {
		payload["kwarg"] = kwarg
	}
	// 优先尝试标准根路径
	res, err := c.makeRequest("/", "POST", payload)
	if err != nil {
		// 某些rest_cherrypy配置使用 /run 作为执行入口，针对404进行回退
		if strings.Contains(strings.ToLower(err.Error()), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return c.makeRequest("/run", "POST", payload)
		}
	}
	return res, err
}

// makeWheel 调用 wheel 客户端（如 key.list_all）
func (c *saltAPIClient) makeWheel(fun string, kwarg map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"client": "wheel",
		"fun":    fun,
	}
	if kwarg != nil {
		payload["kwarg"] = kwarg
	}
	// 优先尝试标准根路径
	res, err := c.makeRequest("/", "POST", payload)
	if err != nil {
		// 针对404进行 /run 回退
		if strings.Contains(strings.ToLower(err.Error()), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return c.makeRequest("/run", "POST", payload)
		}
	}
	return res, err
}

// makeLocal 调用 local 客户端（执行到 minion 上）
func (c *saltAPIClient) makeLocal(fun string, arg []interface{}, kwarg map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"client": "local",
		"fun":    fun,
	}
	if len(arg) > 0 {
		payload["arg"] = arg
	}
	if kwarg != nil {
		payload["kwarg"] = kwarg
	}
	// 优先尝试标准根路径
	res, err := c.makeRequest("/", "POST", payload)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return c.makeRequest("/run", "POST", payload)
		}
	}
	return res, err
}

// GetSaltStackStatus 获取SaltStack状态
func (h *SaltStackHandler) GetSaltStackStatus(c *gin.Context) {
	ctx := c.Request.Context()
	cacheKey := "saltstack:status"

	// 尝试从缓存获取
	if cached, err := h.cache.Get(ctx, cacheKey).Result(); err == nil {
		var status SaltStackStatus
		if err := json.Unmarshal([]byte(cached), &status); err == nil {
			c.JSON(http.StatusOK, gin.H{"data": status, "cached": true})
			return
		}
	}

	// 创建带超时控制的 context
	timeoutSec := 60 // 状态检查使用 60 秒超时
	if t := os.Getenv("SALT_API_STATUS_TIMEOUT"); t != "" {
		// 支持带单位的时间值（如 "60s", "2m"）或纯数字（秒）
		if d, err := time.ParseDuration(t); err == nil && d > 0 {
			timeoutSec = int(d.Seconds())
		} else if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			timeoutSec = parsed
		}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// 创建API客户端
	client := h.newSaltAPIClient()

	// 使用 goroutine 执行，以便响应 context 取消
	type result struct {
		status SaltStackStatus
		err    error
	}
	resultChan := make(chan result, 1)

	go func() {
		// 连接Salt API
		if err := client.authenticate(); err != nil {
			resultChan <- result{err: fmt.Errorf("salt API 认证失败: %v", err)}
			return
		}

		// 获取实际状态
		status, err := h.getRealSaltStackStatus(client)
		if err != nil {
			resultChan <- result{err: fmt.Errorf("获取 salt 状态失败: %v", err)}
			return
		}
		resultChan <- result{status: status}
	}()

	// 等待结果或超时
	select {
	case <-ctx.Done():
		// 超时，尝试返回旧缓存
		if cached, err := h.cache.Get(context.Background(), cacheKey).Result(); err == nil {
			var status SaltStackStatus
			if err := json.Unmarshal([]byte(cached), &status); err == nil {
				c.JSON(http.StatusOK, gin.H{
					"data":    status,
					"cached":  true,
					"warning": "请求超时，返回缓存数据",
				})
				return
			}
		}
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "获取 Salt 状态超时，请检查网络连接"})
		return

	case res := <-resultChan:
		if res.err != nil {
			// 尝试返回旧缓存
			if cached, err := h.cache.Get(context.Background(), cacheKey).Result(); err == nil {
				var status SaltStackStatus
				if err := json.Unmarshal([]byte(cached), &status); err == nil {
					c.JSON(http.StatusOK, gin.H{
						"data":    status,
						"cached":  true,
						"warning": res.err.Error(),
					})
					return
				}
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": res.err.Error()})
			return
		}

		// 缓存状态（2分钟）
		h.cacheStatus(&res.status, 120)
		c.JSON(http.StatusOK, gin.H{"data": res.status})
	}
}

// SaltDebugInfo 用于输出调试信息，便于快速定位集成问题
type SaltDebugInfo struct {
	BaseURL      string                 `json:"base_url"`
	Env          map[string]string      `json:"env"`
	TCPDial      map[string]interface{} `json:"tcp_dial"`
	RootGET      map[string]interface{} `json:"root_get"`
	Login        map[string]interface{} `json:"login"`
	ManageStatus map[string]interface{} `json:"manage_status"`
	KeyListAll   map[string]interface{} `json:"key_list_all"`
	Timestamp    time.Time              `json:"timestamp"`
}

// DebugSaltConnectivity 输出Salt API的连通性与基本调用结果
func (h *SaltStackHandler) DebugSaltConnectivity(c *gin.Context) {
	client := h.newSaltAPIClient()
	res := &SaltDebugInfo{
		BaseURL: client.baseURL,
		Env: map[string]string{
			"SALTSTACK_MASTER_URL": strings.TrimSpace(os.Getenv("SALTSTACK_MASTER_URL")),
			"SALT_API_SCHEME":      getEnv("SALT_API_SCHEME", "http"),
			"SALT_MASTER_HOST":     getEnv("SALT_MASTER_HOST", "saltstack"),
			"SALT_API_PORT":        getEnv("SALT_API_PORT", "8002"),
			"SALT_API_USERNAME":    getEnv("SALT_API_USERNAME", "saltapi"),
			// 不回显明文密码，只提示是否设置
			"SALT_API_PASSWORD_SET": func() string {
				if os.Getenv("SALT_API_PASSWORD") != "" {
					return "true"
				}
				return "false"
			}(),
			"SALT_API_EAUTH": getEnv("SALT_API_EAUTH", "file"),
		},
		TCPDial:      map[string]interface{}{},
		RootGET:      map[string]interface{}{},
		Login:        map[string]interface{}{},
		ManageStatus: map[string]interface{}{},
		KeyListAll:   map[string]interface{}{},
		Timestamp:    time.Now(),
	}

	// 解析 host:port 进行TCP连通性测试
	hostPort := ""
	if strings.HasPrefix(client.baseURL, "http://") {
		hostPort = strings.TrimPrefix(client.baseURL, "http://")
	} else if strings.HasPrefix(client.baseURL, "https://") {
		hostPort = strings.TrimPrefix(client.baseURL, "https://")
	} else {
		hostPort = client.baseURL
	}
	if idx := strings.Index(hostPort, "/"); idx >= 0 { // 去掉路径
		hostPort = hostPort[:idx]
	}
	// 如果没有端口，补全默认端口
	if !strings.Contains(hostPort, ":") {
		hostPort = hostPort + ":80"
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", hostPort, 3*time.Second)
	if err != nil {
		res.TCPDial["ok"] = false
		res.TCPDial["error"] = err.Error()
	} else {
		_ = conn.Close()
		res.TCPDial["ok"] = true
	}
	res.TCPDial["duration_ms"] = time.Since(start).Milliseconds()
	res.TCPDial["target"] = hostPort

	// GET /
	{
		start := time.Now()
		info, err := client.makeRequest("/", "GET", nil)
		res.RootGET["duration_ms"] = time.Since(start).Milliseconds()
		if err != nil {
			res.RootGET["ok"] = false
			res.RootGET["error"] = err.Error()
		} else {
			res.RootGET["ok"] = true
			// 尝试提取版本
			res.RootGET["api_version"] = h.extractAPIVersion(info)
			res.RootGET["salt_version_hint"] = h.extractAPISaltVersion(info)
		}
	}

	// 登录
	{
		start := time.Now()
		if err := client.authenticate(); err != nil {
			res.Login["ok"] = false
			res.Login["error"] = err.Error()
		} else {
			res.Login["ok"] = true
		}
		res.Login["duration_ms"] = time.Since(start).Milliseconds()
		res.Login["token_set"] = client.token != ""
	}

	// runner.manage.status
	{
		start := time.Now()
		r, err := client.makeRunner("manage.status", map[string]interface{}{
			"timeout":            5,
			"gather_job_timeout": 3,
		})
		res.ManageStatus["duration_ms"] = time.Since(start).Milliseconds()
		if err != nil {
			res.ManageStatus["ok"] = false
			res.ManageStatus["error"] = err.Error()
		} else {
			up, down := h.parseManageStatus(r)
			res.ManageStatus["ok"] = true
			res.ManageStatus["up_count"] = len(up)
			res.ManageStatus["down_count"] = len(down)
			res.ManageStatus["sample_up"] = up
			res.ManageStatus["sample_down"] = down
		}
	}

	// wheel.key.list_all
	{
		start := time.Now()
		r, err := client.makeWheel("key.list_all", nil)
		res.KeyListAll["duration_ms"] = time.Since(start).Milliseconds()
		if err != nil {
			res.KeyListAll["ok"] = false
			res.KeyListAll["error"] = err.Error()
		} else {
			minions, pre, rejected := h.parseWheelKeys(r)
			res.KeyListAll["ok"] = true
			res.KeyListAll["accepted"] = minions
			res.KeyListAll["pending"] = pre
			res.KeyListAll["rejected"] = rejected
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

// getRealSaltStackStatus 获取真实的SaltStack状态
func (h *SaltStackHandler) getRealSaltStackStatus(client *saltAPIClient) (SaltStackStatus, error) {
	// 使用并行请求获取状态，提高响应速度
	type apiResult struct {
		apiInfo      map[string]interface{}
		manageStatus map[string]interface{}
		keysResp     map[string]interface{}
		err          error
	}

	// 并行获取 API 信息、管理状态和密钥信息
	resultChan := make(chan apiResult, 1)
	go func() {
		var result apiResult

		// 获取 API 根信息（用于APIVersion）- 这个通常很快
		result.apiInfo, _ = client.makeRequest("/", "GET", nil)

		// 获取 up/down 状态 - 这是最重要的
		// 设置较短的超时参数，避免等待离线 minion 过久
		// timeout: 等待 minion 响应的超时时间 (秒)
		// gather_job_timeout: 收集作业结果的超时时间 (秒)
		result.manageStatus, result.err = client.makeRunner("manage.status", map[string]interface{}{
			"timeout":            5, // 5 秒等待 minion 响应
			"gather_job_timeout": 3, // 3 秒收集结果
		})
		if result.err != nil {
			resultChan <- result
			return
		}

		// 获取 keys
		result.keysResp, result.err = client.makeWheel("key.list_all", nil)
		resultChan <- result
	}()

	// 等待基础状态获取完成（最多 30 秒）
	var apiInfo, manageStatus, keysResp map[string]interface{}
	select {
	case result := <-resultChan:
		if result.err != nil {
			return SaltStackStatus{}, result.err
		}
		apiInfo = result.apiInfo
		manageStatus = result.manageStatus
		keysResp = result.keysResp
	case <-time.After(30 * time.Second):
		return SaltStackStatus{}, fmt.Errorf("获取 Salt 基础状态超时")
	}

	up, down := h.parseManageStatus(manageStatus)
	minions, pre, rejected := h.parseWheelKeys(keysResp)

	// 性能指标在单独的 goroutine 中获取，设置较短超时，失败不影响主要状态
	cpuUsage, memoryUsage, activeConnections := 0, 0, 0
	type metricsResult struct {
		cpu, mem, conn int
		bw             float64
		source         string
	}
	metricsChan := make(chan metricsResult, 1)
	go func() {
		log.Printf("[SaltStack] 开始获取性能指标...")
		cpu, mem, conn, bw, src := h.getPerformanceMetrics(client)
		log.Printf("[SaltStack] 性能指标获取完成: CPU=%d%%, Memory=%d%%, Connections=%d, Bandwidth=%.2f Mbps, Source=%s", cpu, mem, conn, bw, src)
		metricsChan <- metricsResult{cpu, mem, conn, bw, src}
	}()

	// 等待性能指标，最多 10 秒
	var networkBandwidth float64
	var metricsSource string = "none"
	select {
	case metrics := <-metricsChan:
		cpuUsage, memoryUsage, activeConnections = metrics.cpu, metrics.mem, metrics.conn
		networkBandwidth = metrics.bw
		metricsSource = metrics.source
		log.Printf("[SaltStack] 使用获取到的性能指标: CPU=%d%%, Memory=%d%%, Connections=%d, Bandwidth=%.2f Mbps, Source=%s", cpuUsage, memoryUsage, activeConnections, networkBandwidth, metricsSource)
	case <-time.After(10 * time.Second):
		log.Printf("[SaltStack] 获取性能指标超时（10秒），使用默认值")
		metricsSource = "timeout"
	}

	// 构造状态
	status := SaltStackStatus{
		Status:           "connected",
		MasterVersion:    h.extractAPISaltVersion(apiInfo),
		APIVersion:       h.extractAPIVersion(apiInfo),
		Uptime:           0,
		ConnectedMinions: len(up),
		AcceptedKeys:     minions,
		UnacceptedKeys:   pre,
		RejectedKeys:     rejected,
		Services: map[string]string{
			"salt-master": "running",
			"salt-api":    "running",
		},
		LastUpdated: time.Now(),
		Demo:        false,
		// 兼容字段
		MasterStatus:      "running",
		APIStatus:         "running",
		MinionsUp:         len(up),
		MinionsDown:       len(down),
		SaltVersion:       h.extractAPISaltVersion(apiInfo),
		ConfigFile:        "/etc/salt/master",
		LogLevel:          "info",
		CPUUsage:          float64(cpuUsage),
		MemoryUsage:       memoryUsage,
		ActiveConnections: activeConnections,
		NetworkBandwidth:  networkBandwidth, // 从 Docker API 或 VictoriaMetrics 获取
		MetricsSource:     metricsSource,    // 监控数据来源
		// 多 Master 高可用信息
		ActiveMasterURL:  client.baseURL,
		MasterCount:      h.masterPool.GetMasterCount(),
		HealthyMasters:   h.masterPool.GetHealthyCount(),
		MasterHealthInfo: h.getMasterHealthInfo(),
	}
	_ = down // 可用于前端显示 down 数量
	return status, nil
}

// getMasterHealthInfo 获取所有 Master 的健康信息
func (h *SaltStackHandler) getMasterHealthInfo() []MasterHealthInfo {
	masters := h.masterPool.GetAllMasters()
	result := make([]MasterHealthInfo, 0, len(masters))

	h.masterPool.mu.RLock()
	defer h.masterPool.mu.RUnlock()

	for _, m := range masters {
		info := MasterHealthInfo{
			URL:      m.URL,
			Priority: m.Priority,
			Healthy:  h.masterPool.healthStatus[m.URL],
		}
		if lastCheck, exists := h.masterPool.lastCheck[m.URL]; exists {
			info.LastCheck = lastCheck
		}
		result = append(result, info)
	}
	return result
}

// getPerformanceMetrics 获取Salt Master性能指标（CPU、内存、活跃连接数、网络带宽）
// 优先级：1. VictoriaMetrics 2. Docker API (Salt Master容器) 3. Salt API
// 注意：不使用本地系统采集，因为那会采集到 backend 容器的指标，而不是 Salt Master 的指标
// 返回值：cpu%, memory%, connections, bandwidth(Mbps), source(数据来源)
func (h *SaltStackHandler) getPerformanceMetrics(client *saltAPIClient) (int, int, int, float64, string) {
	// 优先从 VictoriaMetrics 获取 Salt Master 监控数据（需要 Categraf 已部署）
	cpuUsage, memoryUsage, activeConnections, err := h.metricsService.GetSaltStackMetrics()
	if err != nil {
		log.Printf("[MetricsService] 从VictoriaMetrics获取监控数据失败: %v", err)
	}

	// 如果从 VictoriaMetrics 获取到有效数据，直接返回
	if cpuUsage > 0 || memoryUsage > 0 {
		log.Printf("[MetricsService] 从VictoriaMetrics获取到Salt Master监控数据: CPU=%d%%, Memory=%d%%, Connections=%d",
			cpuUsage, memoryUsage, activeConnections)
		// 从 VictoriaMetrics 获取网络带宽
		networkBw := h.getNetworkBandwidthFromMetrics()
		return cpuUsage, memoryUsage, activeConnections, networkBw, "victoriametrics"
	}

	// 回退：尝试通过 Docker API 获取 Salt Master 容器指标
	dockerCPU, dockerMem, dockerConns, dockerBw := h.getSaltMasterContainerMetrics()
	if dockerCPU > 0 || dockerMem > 0 {
		log.Printf("[MetricsService] 从Docker API获取到Salt Master容器指标: CPU=%d%%, Memory=%d%%, Connections=%d, Bandwidth=%.2f Mbps",
			dockerCPU, dockerMem, dockerConns, dockerBw)
		return dockerCPU, dockerMem, dockerConns, dockerBw, "docker"
	}

	// 最后回退：Salt API 方式获取
	log.Printf("[MetricsService] VictoriaMetrics和Docker均无数据，回退到Salt API方式获取性能指标")
	cpu, mem, conn := h.getPerformanceMetricsFromSalt(client)
	if cpu > 0 || mem > 0 {
		return cpu, mem, conn, 0, "salt"
	}
	return cpu, mem, conn, 0, "none"
}

// getNetworkBandwidthFromMetrics 从 VictoriaMetrics 获取网络带宽
func (h *SaltStackHandler) getNetworkBandwidthFromMetrics() float64 {
	// 尝试查询网络带宽指标
	// 返回 Mbps
	if h.metricsService == nil {
		return 0
	}

	// 查询网络接收速率
	rxQuery := `sum(rate(net_bytes_recv{ident=~".*salt.*|saltstack|salt-master.*"}[1m]))`
	txQuery := `sum(rate(net_bytes_sent{ident=~".*salt.*|saltstack|salt-master.*"}[1m]))`

	var totalBw float64
	if result, err := h.metricsService.Query(rxQuery); err == nil {
		totalBw += h.metricsService.ExtractValue(result)
	}
	if result, err := h.metricsService.Query(txQuery); err == nil {
		totalBw += h.metricsService.ExtractValue(result)
	}

	// 转换为 Mbps (bytes/s -> Mbps)
	return totalBw * 8 / 1000000
}

// getSaltMasterContainerMetrics 通过 Docker API 获取 Salt Master 容器指标
func (h *SaltStackHandler) getSaltMasterContainerMetrics() (cpuUsage, memoryUsage, activeConnections int, networkBandwidth float64) {
	// 尝试连接 Docker socket
	dockerClient, err := services.NewDockerMetricsClient()
	if err != nil {
		log.Printf("[DockerMetrics] 无法连接Docker API: %v", err)
		return 0, 0, 0, 0
	}
	defer dockerClient.Close()

	// Salt Master 容器名称模式
	containerNames := []string{
		"ai-infra-salt-master-1",
		"ai-infra-salt-master",
		"salt-master-1",
		"salt-master",
	}

	for _, name := range containerNames {
		metrics, err := dockerClient.GetContainerMetrics(name)
		if err == nil && metrics != nil {
			cpuUsage = int(metrics.CPUPercent)
			memoryUsage = int(metrics.MemoryPercent)
			activeConnections = metrics.NetworkConnections
			// 计算网络带宽 (rx + tx) bytes/s -> Mbps
			networkBandwidth = float64(metrics.NetworkRxBytes+metrics.NetworkTxBytes) * 8 / 1000000
			if cpuUsage > 0 || memoryUsage > 0 {
				log.Printf("[DockerMetrics] 从容器 %s 获取到指标", name)
				return cpuUsage, memoryUsage, activeConnections, networkBandwidth
			}
		}
	}

	return 0, 0, 0, 0
}

// getPerformanceMetricsFromSalt 从Salt API获取性能指标（回退方式）
// 使用 Salt 内置的 status 模块获取系统指标
func (h *SaltStackHandler) getPerformanceMetricsFromSalt(client *saltAPIClient) (int, int, int) {
	// 默认值
	cpuUsage := 0
	memoryUsage := 0
	activeConnections := 0

	// 获取任意一个在线 minion 的指标作为集群代表
	// 因为 Salt Master 自身通常不作为 minion 运行
	targetMinion := "*"

	log.Printf("[SaltMetrics] 开始从 Salt API 获取性能指标，目标: %s", targetMinion)

	// 使用 status.cpustats 获取更准确的 CPU 使用率
	// 优先尝试 status.cpustats，它返回详细的 CPU 使用统计
	cpuPayload := map[string]interface{}{
		"client": "local",
		"tgt":    targetMinion,
		"fun":    "status.cpustats",
		"kwarg": map[string]interface{}{
			"timeout": 5,
		},
	}
	cpuResp, err := client.makeRequest("/", "POST", cpuPayload)
	if err != nil {
		log.Printf("[SaltMetrics] status.cpustats 请求失败: %v", err)
	} else {
		cpuUsage = h.extractCPUUsageFromStats(cpuResp)
		log.Printf("[SaltMetrics] status.cpustats 返回 CPU 使用率: %d%%", cpuUsage)
	}

	// 如果 cpustats 没有数据，回退到 cpuload（负载平均值）
	if cpuUsage == 0 {
		log.Printf("[SaltMetrics] cpustats 无数据，尝试 cpuload...")
		cpuPayload["fun"] = "status.cpuload"
		cpuResp, err = client.makeRequest("/", "POST", cpuPayload)
		if err != nil {
			log.Printf("[SaltMetrics] status.cpuload 请求失败: %v", err)
		} else {
			cpuUsage = h.extractFirstSaltCPULoad(cpuResp)
			log.Printf("[SaltMetrics] status.cpuload 返回 CPU 负载转换: %d%%", cpuUsage)
		}
	}

	// 使用 status.meminfo 获取内存信息
	memPayload := map[string]interface{}{
		"client": "local",
		"tgt":    targetMinion,
		"fun":    "status.meminfo",
		"kwarg": map[string]interface{}{
			"timeout": 5,
		},
	}
	memResp, err := client.makeRequest("/", "POST", memPayload)
	if err != nil {
		log.Printf("[SaltMetrics] status.meminfo 请求失败: %v", err)
	} else {
		memoryUsage = h.extractFirstSaltMemoryUsage(memResp)
		log.Printf("[SaltMetrics] status.meminfo 返回内存使用率: %d%%", memoryUsage)
	}

	// 使用 runner 获取活跃连接数（已连接的 minion 数量）
	connPayload := map[string]interface{}{
		"client": "runner",
		"fun":    "manage.status",
	}
	connResp, err := client.makeRequest("/", "POST", connPayload)
	if err != nil {
		log.Printf("[SaltMetrics] manage.status 请求失败: %v", err)
	} else {
		activeConnections = h.extractActiveMinions(connResp)
		log.Printf("[SaltMetrics] manage.status 返回活跃连接数: %d", activeConnections)
	}

	log.Printf("[SaltMetrics] 最终获取到的指标: CPU=%d%%, Memory=%d%%, Connections=%d",
		cpuUsage, memoryUsage, activeConnections)

	return cpuUsage, memoryUsage, activeConnections
}

// extractSaltCPULoad 从 status.cpuload 响应中提取 CPU 使用率
// cpuload 返回格式: {"return": [{"minion_id": {"1-min": 0.5, "5-min": 0.3, "15-min": 0.2}}]}
// 我们取 1 分钟平均负载并转换为百分比（假设单核 100% 负载）
func (h *SaltStackHandler) extractSaltCPULoad(resp map[string]interface{}, minionID string) int {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if retMap, ok := ret[0].(map[string]interface{}); ok {
			if minionData, ok := retMap[minionID]; ok {
				if loadMap, ok := minionData.(map[string]interface{}); ok {
					// 取 1 分钟负载
					if load1, ok := loadMap["1-min"].(float64); ok {
						// 将负载转换为百分比（假设 1.0 = 100%）
						// 对于多核系统，可能需要除以核心数
						cpuPercent := int(load1 * 100)
						if cpuPercent > 100 {
							cpuPercent = 100
						}
						return cpuPercent
					}
				}
			}
		}
	}
	return 0
}

// extractFirstSaltCPULoad 从任意 minion 提取 CPU 负载
func (h *SaltStackHandler) extractFirstSaltCPULoad(resp map[string]interface{}) int {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if retMap, ok := ret[0].(map[string]interface{}); ok {
			for _, minionData := range retMap {
				if loadMap, ok := minionData.(map[string]interface{}); ok {
					if load1, ok := loadMap["1-min"].(float64); ok {
						cpuPercent := int(load1 * 100)
						if cpuPercent > 100 {
							cpuPercent = 100
						}
						return cpuPercent
					}
				}
			}
		}
	}
	return 0
}

// extractCPUUsageFromStats 从 status.cpustats 响应中提取真正的 CPU 使用率
// cpustats 返回格式: {"return": [{"minion_id": {"user": 1.5, "system": 0.8, "idle": 97.7, ...}}]}
// CPU 使用率 = 100 - idle
func (h *SaltStackHandler) extractCPUUsageFromStats(resp map[string]interface{}) int {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if retMap, ok := ret[0].(map[string]interface{}); ok {
			for _, minionData := range retMap {
				if statsMap, ok := minionData.(map[string]interface{}); ok {
					// 尝试从 idle 计算使用率
					if idle, ok := statsMap["idle"].(float64); ok {
						cpuUsage := int(100 - idle)
						if cpuUsage < 0 {
							cpuUsage = 0
						}
						if cpuUsage > 100 {
							cpuUsage = 100
						}
						return cpuUsage
					}
					// 备用：累加 user + system + nice + iowait 等
					var totalUsage float64
					for key, value := range statsMap {
						if key != "idle" && key != "steal" {
							if v, ok := value.(float64); ok {
								totalUsage += v
							}
						}
					}
					if totalUsage > 0 {
						cpuUsage := int(totalUsage)
						if cpuUsage > 100 {
							cpuUsage = 100
						}
						return cpuUsage
					}
				}
			}
		}
	}
	return 0
}

// extractSaltMemoryUsage 从 status.meminfo 响应中提取内存使用率
// meminfo 返回格式: {"return": [{"minion_id": {"MemTotal": {"value": "8094236", "unit": "kB"}, ...}}]}
func (h *SaltStackHandler) extractSaltMemoryUsage(resp map[string]interface{}, minionID string) int {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if retMap, ok := ret[0].(map[string]interface{}); ok {
			if minionData, ok := retMap[minionID]; ok {
				return h.calculateMemoryUsage(minionData)
			}
		}
	}
	return 0
}

// extractFirstSaltMemoryUsage 从任意 minion 提取内存使用率
func (h *SaltStackHandler) extractFirstSaltMemoryUsage(resp map[string]interface{}) int {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if retMap, ok := ret[0].(map[string]interface{}); ok {
			for _, minionData := range retMap {
				if usage := h.calculateMemoryUsage(minionData); usage > 0 {
					return usage
				}
			}
		}
	}
	return 0
}

// calculateMemoryUsage 计算内存使用率百分比
func (h *SaltStackHandler) calculateMemoryUsage(minionData interface{}) int {
	memMap, ok := minionData.(map[string]interface{})
	if !ok {
		return 0
	}

	// 提取 MemTotal 和 MemAvailable
	var memTotal, memAvailable float64

	if memTotalData, ok := memMap["MemTotal"].(map[string]interface{}); ok {
		if valStr, ok := memTotalData["value"].(string); ok {
			memTotal, _ = strconv.ParseFloat(valStr, 64)
		}
	}

	if memAvailData, ok := memMap["MemAvailable"].(map[string]interface{}); ok {
		if valStr, ok := memAvailData["value"].(string); ok {
			memAvailable, _ = strconv.ParseFloat(valStr, 64)
		}
	}

	// 计算内存使用率
	if memTotal > 0 {
		memUsed := memTotal - memAvailable
		memPercent := int((memUsed / memTotal) * 100)
		if memPercent < 0 {
			memPercent = 0
		}
		if memPercent > 100 {
			memPercent = 100
		}
		return memPercent
	}

	return 0
}

// extractActiveMinions 从 runner manage.status 响应中提取活跃 minion 数量
// 返回格式: {"return": [{"up": ["minion1", "minion2"], "down": []}]}
func (h *SaltStackHandler) extractActiveMinions(resp map[string]interface{}) int {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if statusMap, ok := ret[0].(map[string]interface{}); ok {
			if upList, ok := statusMap["up"].([]interface{}); ok {
				return len(upList)
			}
		}
	}
	return 0
}

// getDemoSaltStackStatus 获取演示用的SaltStack状态
func (h *SaltStackHandler) getDemoSaltStackStatus() SaltStackStatus {
	return SaltStackStatus{
		Status:           "demo",
		MasterVersion:    "3006.4",
		APIVersion:       "3000.3",
		Uptime:           7200, // 2小时
		ConnectedMinions: 3,
		AcceptedKeys:     []string{"ai-infra-web-01", "ai-infra-db-01", "ai-infra-compute-01"},
		UnacceptedKeys:   []string{"ai-infra-new-node"},
		RejectedKeys:     []string{},
		Services: map[string]string{
			"salt-master": "running",
			"salt-api":    "running",
			"salt-syndic": "stopped",
		},
		LastUpdated: time.Now(),
		Demo:        true,
		// 兼容字段
		MasterStatus:      "running",
		APIStatus:         "running",
		MinionsUp:         2,
		MinionsDown:       1,
		SaltVersion:       "3006.4",
		ConfigFile:        "/etc/salt/master",
		LogLevel:          "info",
		CPUUsage:          12.5,
		MemoryUsage:       23,
		ActiveConnections: 2,
		NetworkBandwidth:  15.8,
	}
}

// GetSaltMinions 获取Salt Minion列表
func (h *SaltStackHandler) GetSaltMinions(c *gin.Context) {
	ctx := c.Request.Context()
	cacheKey := "saltstack:minions"

	// 检查是否强制刷新（支持 ?refresh=true 或 ?force=true）
	forceRefresh := c.Query("refresh") == "true" || c.Query("force") == "true"

	// 尝试从缓存获取（除非强制刷新）
	if !forceRefresh {
		if cached, err := h.cache.Get(ctx, cacheKey).Result(); err == nil {
			var minions []SaltMinion
			if err := json.Unmarshal([]byte(cached), &minions); err == nil {
				// 从数据库获取最新的分组信息并更新到缓存的 Minion 数据中
				groupMap, _ := h.minionGroupService.GetAllMinionGroupMap()
				for i := range minions {
					if group, ok := groupMap[minions[i].ID]; ok {
						minions[i].Group = group
					} else {
						minions[i].Group = "" // 清除旧分组信息
					}
				}
				c.JSON(http.StatusOK, gin.H{"data": minions, "cached": true})
				return
			}
		}
	}

	// 创建带超时控制的 context
	timeoutSec := 60 // 默认60秒总超时
	if t := os.Getenv("SALT_API_REQUEST_TIMEOUT"); t != "" {
		// 支持带单位的时间值（如 "60s", "2m"）或纯数字（秒）
		if d, err := time.ParseDuration(t); err == nil && d > 0 {
			timeoutSec = int(d.Seconds())
		} else if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			timeoutSec = parsed
		}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// 创建API客户端
	client := h.newSaltAPIClient()

	// 使用 goroutine 执行认证和获取数据，以便可以响应 context 取消
	type result struct {
		minions []SaltMinion
		err     error
	}
	resultChan := make(chan result, 1)

	go func() {
		// 认证
		if err := client.authenticate(); err != nil {
			resultChan <- result{err: fmt.Errorf("salt API authentication failed: %v", err)}
			return
		}

		// 获取真实数据
		minions, err := h.getRealMinions(client)
		if err != nil {
			resultChan <- result{err: fmt.Errorf("failed to get minions: %v", err)}
			return
		}
		resultChan <- result{minions: minions}
	}()

	// 等待结果或超时
	select {
	case <-ctx.Done():
		// 超时，尝试返回旧缓存
		if cached, err := h.cache.Get(context.Background(), cacheKey).Result(); err == nil {
			var minions []SaltMinion
			if err := json.Unmarshal([]byte(cached), &minions); err == nil {
				// 从数据库获取最新的分组信息
				groupMap, _ := h.minionGroupService.GetAllMinionGroupMap()
				for i := range minions {
					if group, ok := groupMap[minions[i].ID]; ok {
						minions[i].Group = group
					} else {
						minions[i].Group = ""
					}
				}
				c.JSON(http.StatusOK, gin.H{
					"data":    minions,
					"cached":  true,
					"warning": "请求超时，返回缓存数据",
				})
				return
			}
		}
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "获取 Minions 超时，请检查网络连接"})
		return

	case res := <-resultChan:
		if res.err != nil {
			// 尝试返回旧缓存
			if cached, err := h.cache.Get(context.Background(), cacheKey).Result(); err == nil {
				var minions []SaltMinion
				if err := json.Unmarshal([]byte(cached), &minions); err == nil {
					// 从数据库获取最新的分组信息
					groupMap, _ := h.minionGroupService.GetAllMinionGroupMap()
					for i := range minions {
						if group, ok := groupMap[minions[i].ID]; ok {
							minions[i].Group = group
						} else {
							minions[i].Group = ""
						}
					}
					c.JSON(http.StatusOK, gin.H{
						"data":    minions,
						"cached":  true,
						"warning": res.err.Error(),
					})
					return
				}
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": res.err.Error()})
			return
		}

		// 为每个 Minion 添加分组信息
		groupMap, _ := h.minionGroupService.GetAllMinionGroupMap()
		for i := range res.minions {
			if group, ok := groupMap[res.minions[i].ID]; ok {
				res.minions[i].Group = group
			}
		}

		// 缓存数据（缓存时间延长到 5 分钟）
		if data, err := json.Marshal(res.minions); err == nil {
			h.cache.Set(context.Background(), cacheKey, string(data), 300*time.Second)
		}

		c.JSON(http.StatusOK, gin.H{"data": res.minions, "demo": false})
	}
}

// getDemoMinions 获取演示用的Minion数据
func (h *SaltStackHandler) getDemoMinions() []SaltMinion {
	return []SaltMinion{
		{
			ID:           "ai-infra-web-01",
			Status:       "up",
			OS:           "Ubuntu",
			OSVersion:    "22.04",
			Architecture: "x86_64",
			LastSeen:     time.Now().Add(-5 * time.Minute),
			Grains: map[string]interface{}{
				"roles":     []string{"web", "nginx"},
				"cpu_cores": 4,
				"memory":    8192,
			},
		},
		{
			ID:           "ai-infra-db-01",
			Status:       "up",
			OS:           "Ubuntu",
			OSVersion:    "22.04",
			Architecture: "x86_64",
			LastSeen:     time.Now().Add(-2 * time.Minute),
			Grains: map[string]interface{}{
				"roles":     []string{"database", "postgres"},
				"cpu_cores": 8,
				"memory":    16384,
			},
		},
		{
			ID:           "ai-infra-compute-01",
			Status:       "down",
			OS:           "Ubuntu",
			OSVersion:    "22.04",
			Architecture: "x86_64",
			LastSeen:     time.Now().Add(-30 * time.Minute),
			Grains: map[string]interface{}{
				"roles":     []string{"compute", "gpu"},
				"cpu_cores": 16,
				"memory":    32768,
				"gpu":       "NVIDIA RTX 4090",
			},
		},
	}
}

// GetSaltJobs 获取Salt作业列表
func (h *SaltStackHandler) GetSaltJobs(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	var jobs []SaltJob

	// 创建API客户端并认证
	client := h.newSaltAPIClient()
	if err := client.authenticate(); err != nil {
		log.Printf("[SaltStack] Salt API 认证失败: %v，返回空作业列表", err)
		// 认证失败时返回空列表而不是错误，避免前端一直转圈
		c.JSON(http.StatusOK, gin.H{"data": jobs, "demo": false, "total": 0, "error": fmt.Sprintf("Salt API 认证失败: %v", err)})
		return
	}

	// 获取真实作业数据
	var err error
	jobs, err = h.getRealJobs(client, limit)
	if err != nil {
		log.Printf("[SaltStack] 获取Jobs失败: %v，返回空作业列表", err)
		// 获取失败时返回空列表而不是错误
		c.JSON(http.StatusOK, gin.H{"data": []SaltJob{}, "demo": false, "total": 0, "error": fmt.Sprintf("获取Jobs失败: %v", err)})
		return
	}

	// 创建 JID 集合用于快速查找
	existingJIDs := make(map[string]bool)
	for _, job := range jobs {
		existingJIDs[job.JID] = true
	}

	// 从 Redis 获取每个作业的 TaskID
	if h.cache != nil {
		for i := range jobs {
			jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", jobs[i].JID)
			if taskID, err := h.cache.Get(context.Background(), jidToTaskKey).Result(); err == nil && taskID != "" {
				jobs[i].TaskID = taskID
			}
		}

		// 从 Redis 获取最近执行的作业（补充 Salt API 没有返回的）
		recentJIDs, err := h.cache.LRange(context.Background(), "saltstack:recent_jobs", 0, 49).Result()
		if err == nil && len(recentJIDs) > 0 {
			// 监控相关函数的黑名单 - 这些任务不应该展示给用户
			monitoringFunctions := map[string]bool{
				"status.cpuload":        true,
				"runner.manage.status":  true,
				"test.ping":             true,
				"grains.items":          true,
				"saltutil.sync_all":     true,
				"saltutil.sync_grains":  true,
				"saltutil.sync_modules": true,
				"status.meminfo":        true,
				"status.cpuinfo":        true,
				"status.diskusage":      true,
				"status.netstats":       true,
				"status.uptime":         true,
				"status.loadavg":        true,
			}

			for _, jid := range recentJIDs {
				// 如果这个 JID 已经在 Salt API 结果中，跳过
				if existingJIDs[jid] {
					continue
				}

				// 从 Redis 获取作业详情
				jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
				jobInfoJSON, err := h.cache.Get(context.Background(), jobDetailKey).Result()
				if err != nil {
					continue
				}

				var jobInfo map[string]interface{}
				if err := json.Unmarshal([]byte(jobInfoJSON), &jobInfo); err != nil {
					continue
				}

				// 构建 SaltJob 对象
				newJob := SaltJob{
					JID:      jid,
					Function: getStringFromMap(jobInfo, "function"),
					Target:   getStringFromMap(jobInfo, "target"),
					Status:   getStringFromMap(jobInfo, "status"),
					User:     getStringFromMap(jobInfo, "user"),
					TaskID:   getStringFromMap(jobInfo, "task_id"),
				}

				// 过滤监控相关的任务
				if monitoringFunctions[newJob.Function] {
					continue
				}
				// 过滤以 "status." 或 "runner." 开头的任务
				if strings.HasPrefix(newJob.Function, "status.") || strings.HasPrefix(newJob.Function, "runner.") {
					continue
				}

				// 解析成功/失败数量
				if sc, ok := jobInfo["success_count"].(float64); ok {
					newJob.SuccessCount = int(sc)
				}
				if fc, ok := jobInfo["failed_count"].(float64); ok {
					newJob.FailedCount = int(fc)
				}

				// 解析结果
				if result, ok := jobInfo["result"].(map[string]interface{}); ok {
					newJob.Result = result
				}

				// 解析参数
				if args, ok := jobInfo["arguments"].([]interface{}); ok {
					for _, arg := range args {
						newJob.Arguments = append(newJob.Arguments, fmt.Sprint(arg))
					}
				}

				// 解析开始时间
				if startTimeStr, ok := jobInfo["start_time"].(string); ok {
					if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
						newJob.StartTime = t
					} else {
						newJob.StartTime = time.Now()
					}
				} else {
					newJob.StartTime = time.Now()
				}

				// 解析结束时间
				if endTimeStr, ok := jobInfo["end_time"].(string); ok {
					if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
						newJob.EndTime = &t
					}
				}

				// 添加到作业列表
				jobs = append(jobs, newJob)
				existingJIDs[jid] = true
			}
		}
	}

	// 按时间排序
	if len(jobs) > 1 {
		sort.Slice(jobs, func(i, j int) bool { return jobs[i].StartTime.After(jobs[j].StartTime) })
	}

	// 限制返回数量
	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}

	// 缓存数据
	cacheKey := fmt.Sprintf("saltstack:jobs:%d", limit)
	if data, err := json.Marshal(jobs); err == nil {
		h.cache.Set(context.Background(), cacheKey, string(data), 300*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{"data": jobs, "demo": false, "total": len(jobs)})
}

// getStringFromMap 从 map 中安全获取字符串值
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// GetSaltJobDetail 获取单个Salt作业的详细结果
// 优先从数据库查询已持久化的作业，只有数据库没有时才从 Salt API 查询
// 如果从 Salt API 获取到数据，会自动同步到数据库
func (h *SaltStackHandler) GetSaltJobDetail(c *gin.Context) {
	jid := c.Param("jid")
	if jid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少作业ID (jid)"})
		return
	}

	ctx := context.Background()

	// Step 1: 优先从数据库查询（最可靠的数据源）
	saltJobService := services.GetSaltJobService()
	if saltJobService != nil {
		dbJob, err := saltJobService.GetJobByJID(ctx, jid)
		if err == nil && dbJob != nil {
			log.Printf("[GetSaltJobDetail] 从数据库获取作业: JID=%s", jid)

			// 解析结果 JSON
			var resultData map[string]interface{}
			if dbJob.Result != "" {
				json.Unmarshal([]byte(dbJob.Result), &resultData)
			}

			// 解析参数
			var args []string
			if dbJob.Arguments != "" {
				json.Unmarshal([]byte(dbJob.Arguments), &args)
			}

			c.JSON(http.StatusOK, gin.H{
				"jid":    jid,
				"source": "database",
				"info": map[string]interface{}{
					"Function":    dbJob.Function,
					"Target":      dbJob.Target,
					"Arguments":   args,
					"User":        dbJob.User,
					"StartTime":   dbJob.StartTime.Format(time.RFC3339),
					"Status":      dbJob.Status,
					"task_id":     dbJob.TaskID,
					"return_code": dbJob.ReturnCode,
				},
				"result":        resultData,
				"status":        dbJob.Status,
				"task_id":       dbJob.TaskID,
				"user":          dbJob.User,
				"start_time":    dbJob.StartTime,
				"end_time":      dbJob.EndTime,
				"duration":      dbJob.Duration,
				"success_count": dbJob.SuccessCount,
				"failed_count":  dbJob.FailedCount,
				"error_message": dbJob.ErrorMessage,
			})
			return
		}
	}

	// Step 2: 数据库没有，尝试从 Redis 缓存获取
	if h.cache != nil {
		jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
		jobInfoJSON, err := h.cache.Get(ctx, jobDetailKey).Result()
		if err == nil && jobInfoJSON != "" {
			log.Printf("[GetSaltJobDetail] 从 Redis 获取作业: JID=%s", jid)

			var cachedJob map[string]interface{}
			if err := json.Unmarshal([]byte(jobInfoJSON), &cachedJob); err == nil {
				c.JSON(http.StatusOK, gin.H{
					"jid":    jid,
					"source": "redis",
					"info":   cachedJob,
					"result": cachedJob["result"],
				})
				return
			}
		}
	}

	// Step 3: 都没有，从 Salt API 查询
	log.Printf("[GetSaltJobDetail] 从 Salt API 查询作业: JID=%s", jid)

	client := h.newSaltAPIClient()
	if err := client.authenticate(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Salt API 认证失败: %v", err)})
		return
	}

	resp, err := client.makeRequest("/jobs/"+jid, "GET", nil)
	if err != nil {
		// Salt API 也查不到，返回未找到
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "作业不存在或已过期",
			"jid":     jid,
			"message": "该作业可能已被 Salt Master 清理，建议检查作业历史配置中的保留时间",
		})
		return
	}

	// 解析 Salt API 返回结果
	var jobInfo map[string]interface{}
	var jobResult map[string]interface{}

	if info, ok := resp["info"].([]interface{}); ok && len(info) > 0 {
		if infoMap, ok := info[0].(map[string]interface{}); ok {
			jobInfo = infoMap
		}
	}

	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if retMap, ok := ret[0].(map[string]interface{}); ok {
			jobResult = retMap
		}
	}

	// Step 4: 从 Salt API 获取成功，同步到数据库（兜底记录）
	if saltJobService != nil && jobInfo != nil {
		function := getStringFromMap(jobInfo, "Function")
		target := getStringFromMap(jobInfo, "Target")
		user := getStringFromMap(jobInfo, "User")

		// 解析参数
		var argsStr string
		if args, ok := jobInfo["Arguments"].([]interface{}); ok {
			argsJSON, _ := json.Marshal(args)
			argsStr = string(argsJSON)
		}

		// 解析结果
		var resultStr string
		if jobResult != nil {
			resultJSON, _ := json.Marshal(jobResult)
			resultStr = string(resultJSON)
		}

		// 确定状态
		status := "completed"
		if len(jobResult) == 0 {
			status = "unknown"
		}

		// 创建或更新数据库记录
		dbJob := &models.SaltJobHistory{
			JID:       jid,
			Function:  function,
			Target:    target,
			Arguments: argsStr,
			Result:    resultStr,
			User:      user,
			Status:    status,
			StartTime: time.Now(), // Salt API 可能不返回精确时间
		}

		if err := saltJobService.CreateJob(ctx, dbJob); err != nil {
			log.Printf("[GetSaltJobDetail] 同步作业到数据库失败: %v", err)
		} else {
			log.Printf("[GetSaltJobDetail] 已同步作业到数据库: JID=%s", jid)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"jid":    jid,
		"source": "salt_api",
		"info":   jobInfo,
		"result": jobResult,
		"raw":    resp,
	})
}

// getRealJobs 通过Salt NetAPI获取真实作业列表
func (h *SaltStackHandler) getRealJobs(client *saltAPIClient, limit int) ([]SaltJob, error) {
	// 尝试 GET /jobs
	resp, err := client.makeRequest("/jobs", "GET", nil)
	if err != nil {
		return nil, err
	}
	var jobs []SaltJob

	// 监控相关函数的黑名单 - 这些任务不应该展示给用户
	monitoringFunctions := map[string]bool{
		"status.cpuload":        true, // CPU 负载监控
		"runner.manage.status":  true, // Salt Master 状态
		"test.ping":             true, // 心跳检测
		"grains.items":          true, // 节点信息采集
		"saltutil.sync_all":     true, // 模块同步
		"saltutil.sync_grains":  true, // Grains 同步
		"saltutil.sync_modules": true, // 模块同步
		"status.meminfo":        true, // 内存信息
		"status.cpuinfo":        true, // CPU 信息
		"status.diskusage":      true, // 磁盘使用
		"status.netstats":       true, // 网络统计
		"status.uptime":         true, // 系统运行时间
		"status.loadavg":        true, // 系统负载
	}

	// 解析常见结构: {"return":[{"jobs": {"<jid>": {..}, ...}}]}
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			// 某些环境直接返回 map[jid]info
			var jobsMap map[string]interface{}
			if jm, ok := m["jobs"].(map[string]interface{}); ok {
				jobsMap = jm
			} else {
				// 兼容直接是 jid->info 的情况
				jobsMap = m
			}
			for jid, v := range jobsMap {
				if info, ok := v.(map[string]interface{}); ok {
					j := SaltJob{JID: jid}
					if f, ok := info["Function"].(string); ok {
						j.Function = f
					}

					// 过滤监控相关的任务
					if monitoringFunctions[j.Function] {
						continue
					}
					// 过滤以 "status." 或 "runner." 开头的任务（监控类）
					if strings.HasPrefix(j.Function, "status.") || strings.HasPrefix(j.Function, "runner.") {
						continue
					}

					if t, ok := info["Target"].(string); ok {
						j.Target = t
					}
					if u, ok := info["User"].(string); ok {
						j.User = u
					}
					if args, ok := info["Arguments"].([]interface{}); ok {
						for _, a := range args {
							j.Arguments = append(j.Arguments, fmt.Sprint(a))
						}
					}
					// 尝试解析开始时间
					if st, ok := info["StartTime"].(string); ok {
						if ts, err := time.Parse(time.RFC3339, st); err == nil {
							j.StartTime = ts
						} else {
							j.StartTime = time.Now()
						}
					} else {
						j.StartTime = time.Now()
					}
					// 解析作业结果来判断状态
					if result, ok := info["Result"].(map[string]interface{}); ok && len(result) > 0 {
						j.Result = result
						j.Status = "completed"
						// 设置结束时间为开始时间后几秒（估计值）
						endTime := j.StartTime.Add(time.Second * 5)
						j.EndTime = &endTime
					} else {
						// 没有结果，根据时间判断状态
						// 如果作业开始超过5分钟还没结果，可能是失败或超时
						if time.Since(j.StartTime) > 5*time.Minute {
							j.Status = "timeout"
						} else {
							// 默认已完成（Salt的/jobs接口只返回已完成的作业）
							j.Status = "completed"
						}
					}
					jobs = append(jobs, j)
				}
			}
		}
	}
	// 按时间排序并裁剪
	if len(jobs) > 1 {
		sort.Slice(jobs, func(i, j int) bool { return jobs[i].StartTime.After(jobs[j].StartTime) })
	}
	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs, nil
}

// getDemoJobs 获取演示用的作业数据
func (h *SaltStackHandler) getDemoJobs(limit int) []SaltJob {
	jobs := []SaltJob{
		{
			JID:       "20250820135001",
			Function:  "test.ping",
			Arguments: []string{},
			Target:    "*",
			StartTime: time.Now().Add(-10 * time.Minute),
			Status:    "completed",
			Result: map[string]interface{}{
				"ai-infra-web-01": true,
				"ai-infra-db-01":  true,
			},
			User: "admin",
		},
		{
			JID:       "20250820134501",
			Function:  "state.apply",
			Arguments: []string{"common"},
			Target:    "ai-infra-*",
			StartTime: time.Now().Add(-25 * time.Minute),
			Status:    "running",
			User:      "admin",
		},
		{
			JID:       "20250820134001",
			Function:  "cmd.run",
			Arguments: []string{"uptime"},
			Target:    "ai-infra-web-01",
			StartTime: time.Now().Add(-35 * time.Minute),
			Status:    "completed",
			Result: map[string]interface{}{
				"ai-infra-web-01": "13:40:01 up 2 days, 4:30, 1 user, load average: 0.15, 0.05, 0.01",
			},
			User: "admin",
		},
	}

	// 限制返回数量
	if limit < len(jobs) {
		jobs = jobs[:limit]
	}

	// 为结束的作业设置结束时间
	for i := range jobs {
		if jobs[i].Status == "completed" {
			endTime := jobs[i].StartTime.Add(time.Duration(i+1) * time.Minute)
			jobs[i].EndTime = &endTime
		}
	}

	return jobs
}

// 辅助方法
func (h *SaltStackHandler) cacheStatus(status *SaltStackStatus, seconds int) {
	if data, err := json.Marshal(status); err == nil {
		h.cache.Set(context.Background(), "saltstack:status", string(data), time.Duration(seconds)*time.Second)
	}
}

func (h *SaltStackHandler) extractVersion(resp map[string]interface{}) string {
	return h.extractAPISaltVersion(resp)
}

func (h *SaltStackHandler) countMinions(resp map[string]interface{}) int {
	// 兼容多种返回格式
	if data, ok := resp["data"].([]interface{}); ok {
		return len(data)
	}
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		switch v := ret[0].(type) {
		case []interface{}:
			return len(v)
		case map[string]interface{}:
			if minions, ok := v["minions"].([]interface{}); ok {
				return len(minions)
			}
		}
	}
	return 0
}

func (h *SaltStackHandler) extractKeys(resp map[string]interface{}, keyType string) []string {
	if data, ok := resp["data"].(map[string]interface{}); ok {
		if keys, ok := data[keyType].([]interface{}); ok {
			var result []string
			for _, key := range keys {
				if str, ok := key.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			// wheel key.list_all 返回嵌套 data.return
			if data, ok := m["data"].(map[string]interface{}); ok {
				if r, ok := data["return"].(map[string]interface{}); ok {
					if keys, ok := r[keyType].([]interface{}); ok {
						var result []string
						for _, key := range keys {
							if str, ok := key.(string); ok {
								result = append(result, str)
							}
						}
						return result
					}
				}
			}
		}
	}
	return []string{}
}

// getRecentlyDeletedMinionIDs 获取最近已完成删除的 Minion ID 列表
// 返回最近 5 分钟内完成删除的 Minion，用于过滤 Salt Master 可能还返回的残留数据
func (h *SaltStackHandler) getRecentlyDeletedMinionIDs() map[string]bool {
	deletedIDs := make(map[string]bool)

	deleteSvc := services.GetMinionDeleteService()
	if deleteSvc == nil {
		return deletedIDs
	}

	// 获取最近完成删除的任务
	recentlyDeleted, err := deleteSvc.GetRecentlyCompletedMinionIDs(5 * time.Minute)
	if err != nil {
		return deletedIDs
	}

	for _, id := range recentlyDeleted {
		deletedIDs[id] = true
	}

	return deletedIDs
}

// filterDeletedMinions 从 Minion ID 列表中过滤掉已删除的
func (h *SaltStackHandler) filterDeletedMinions(minionIDs []string, deletedIDs map[string]bool) []string {
	if len(deletedIDs) == 0 {
		return minionIDs
	}

	filtered := make([]string, 0, len(minionIDs))
	for _, id := range minionIDs {
		if !deletedIDs[id] {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func (h *SaltStackHandler) getRealMinions(client *saltAPIClient) ([]SaltMinion, error) {
	// 使用 runner manage.status 获取 up/down 列表
	// 设置较短超时，避免等待离线 minion 过久
	statusResp, err := client.makeRunner("manage.status", map[string]interface{}{
		"timeout":            5,
		"gather_job_timeout": 3,
	})
	if err != nil {
		return nil, err
	}
	up, down := h.parseManageStatus(statusResp)

	// 获取最近已完成删除的 Minion ID 列表（用于过滤）
	deletedMinionIDs := h.getRecentlyDeletedMinionIDs()

	// 过滤掉已删除的 Minion
	up = h.filterDeletedMinions(up, deletedMinionIDs)
	down = h.filterDeletedMinions(down, deletedMinionIDs)

	// 并发获取 up 节点的 grains 信息
	var minions []SaltMinion

	if len(up) > 0 {
		// 使用 channel 收集结果
		type minionResult struct {
			minion SaltMinion
			err    error
		}
		resultChan := make(chan minionResult, len(up))

		// 限制并发数，避免过多请求
		maxConcurrent := 5
		if envMax := os.Getenv("SALT_API_MAX_CONCURRENT"); envMax != "" {
			if parsed, err := strconv.Atoi(envMax); err == nil && parsed > 0 {
				maxConcurrent = parsed
			}
		}
		semaphore := make(chan struct{}, maxConcurrent)

		// 为并发请求创建带短超时的客户端
		concurrentClient := h.newSaltAPIClientWithTimeout(15 * time.Second)
		concurrentClient.token = client.token // 复用已认证的 token

		for _, id := range up {
			go func(minionID string) {
				semaphore <- struct{}{}        // 获取信号量
				defer func() { <-semaphore }() // 释放信号量

				// 尝试获取 grains
				r, err := concurrentClient.makeRequest("/minions/"+minionID, "GET", nil)
				if err != nil {
					// 如果获取失败，返回基本信息，标记数据来源为不可用
					resultChan <- minionResult{
						minion: SaltMinion{ID: minionID, Status: "up", LastSeen: time.Now(), DataSource: "unavailable"},
						err:    nil, // 不视为致命错误
					}
					return
				}

				grains := h.parseMinionGrains(r, minionID)
				m := SaltMinion{
					ID:            minionID,
					Status:        "up",
					OS:            fmt.Sprintf("%v", grains["os"]),
					OSVersion:     fmt.Sprintf("%v", grains["osrelease"]),
					Architecture:  fmt.Sprintf("%v", grains["osarch"]),
					Arch:          fmt.Sprintf("%v", grains["osarch"]),
					SaltVersion:   fmt.Sprintf("%v", grains["saltversion"]),
					LastSeen:      time.Now(),
					Grains:        grains,
					KernelVersion: h.extractKernelVersion(grains),
					DataSource:    "realtime", // 默认标记为实时数据
				}

				// 尝试实时获取 GPU 信息（不阻塞主流程）
				gpuInfo := h.getGPUInfo(concurrentClient, minionID)
				if gpuInfo.DriverVersion != "" {
					m.GPUDriverVersion = gpuInfo.DriverVersion
					m.CUDAVersion = gpuInfo.CUDAVersion
					m.GPUCount = gpuInfo.GPUCount
					m.GPUModel = gpuInfo.GPUModel
				}

				// 尝试实时获取 IB 信息
				ibInfo := h.getIBInfo(concurrentClient, minionID)
				m.IBStatus = ibInfo.Status
				m.IBCount = ibInfo.Count
				m.IBRate = ibInfo.Rate

				// 尝试实时获取 CPU/内存使用率信息
				cpuMemInfo := h.getCPUMemoryInfo(concurrentClient, minionID)
				if cpuMemInfo.CPUUsagePercent > 0 || cpuMemInfo.MemoryUsagePercent > 0 {
					m.CPUUsagePercent = cpuMemInfo.CPUUsagePercent
					m.MemoryUsagePercent = cpuMemInfo.MemoryUsagePercent
					m.MemoryTotalGB = cpuMemInfo.MemoryTotalGB
					m.MemoryUsedGB = cpuMemInfo.MemoryUsedGB
				}

				resultChan <- minionResult{minion: m, err: nil}
			}(id)
		}

		// 收集所有结果
		for i := 0; i < len(up); i++ {
			result := <-resultChan
			minions = append(minions, result.minion)
		}
	}

	// 将 down 的节点也加入列表以便页面展示
	for _, id := range down {
		minions = append(minions, SaltMinion{ID: id, Status: "down", DataSource: "unavailable"})
	}

	// 从数据库读取节点指标数据作为兜底（只在实时数据为空时才使用）
	h.enrichMinionsWithMetrics(minions)

	return minions, nil
}

// enrichMinionsWithMetrics 填充节点指标数据
// 优先级: 1. Redis 实时数据 2. 数据库缓存数据
// 只有在实时数据为空时才使用数据库数据，并将数据来源标记为 "cached"
func (h *SaltStackHandler) enrichMinionsWithMetrics(minions []SaltMinion) {
	if len(minions) == 0 {
		return
	}

	// 优先从 Redis 获取实时指标
	redisMetrics, redisErr := services.GetAllNodeMetricsFromRedis()
	if redisErr == nil && len(redisMetrics) > 0 {
		log.Printf("[enrichMinionsWithMetrics] 从 Redis 获取到 %d 个节点的实时指标", len(redisMetrics))
		for i := range minions {
			if metrics, ok := redisMetrics[minions[i].ID]; ok {
				if minions[i].CPUUsagePercent == 0 && metrics.CPUUsagePercent > 0 {
					minions[i].CPUUsagePercent = metrics.CPUUsagePercent
				}
				if minions[i].MemoryUsagePercent == 0 && metrics.MemoryUsagePercent > 0 {
					minions[i].MemoryUsagePercent = metrics.MemoryUsagePercent
				}
				if minions[i].MemoryTotalGB == 0 && metrics.MemoryTotalGB > 0 {
					minions[i].MemoryTotalGB = metrics.MemoryTotalGB
				}
				if minions[i].MemoryUsedGB == 0 && metrics.MemoryUsedGB > 0 {
					minions[i].MemoryUsedGB = metrics.MemoryUsedGB
				}
				// Redis 数据标记为实时
				if minions[i].DataSource == "" {
					minions[i].DataSource = "redis"
				}
			}
		}
	}

	// 从数据库获取所有节点的最新指标（作为 Redis 的兜底）
	metricsService := services.NewNodeMetricsService()
	allMetrics, err := metricsService.GetAllLatestMetrics()
	if err != nil {
		return // 静默失败，不影响基本 Minion 信息返回
	}

	// 构建 minionID -> metrics 的映射
	metricsMap := make(map[string]*models.NodeMetricsLatest)
	for i := range allMetrics {
		metricsMap[allMetrics[i].MinionID] = &allMetrics[i]
	}

	// 填充指标数据到 Minion（仅在实时数据为空时才使用数据库数据作为兜底）
	for i := range minions {
		if metrics, ok := metricsMap[minions[i].ID]; ok {
			usedCachedData := false

			// CPU 使用率兜底
			if minions[i].CPUUsagePercent == 0 && metrics.CPUUsagePercent > 0 {
				minions[i].CPUUsagePercent = metrics.CPUUsagePercent
				usedCachedData = true
			}
			// 内存使用率兜底
			if minions[i].MemoryUsagePercent == 0 && metrics.MemoryUsagePercent > 0 {
				minions[i].MemoryUsagePercent = metrics.MemoryUsagePercent
				usedCachedData = true
			}
			// 内存总量兜底
			if minions[i].MemoryTotalGB == 0 && metrics.MemoryTotalGB > 0 {
				minions[i].MemoryTotalGB = metrics.MemoryTotalGB
				usedCachedData = true
			}
			// 内存已用兜底
			if minions[i].MemoryUsedGB == 0 && metrics.MemoryUsedGB > 0 {
				minions[i].MemoryUsedGB = metrics.MemoryUsedGB
				usedCachedData = true
			}

			// GPU 驱动版本兜底
			if minions[i].GPUDriverVersion == "" && metrics.GPUDriverVersion != "" {
				minions[i].GPUDriverVersion = metrics.GPUDriverVersion
				usedCachedData = true
			}
			// CUDA 版本兜底
			if minions[i].CUDAVersion == "" && metrics.CUDAVersion != "" {
				minions[i].CUDAVersion = metrics.CUDAVersion
				usedCachedData = true
			}
			// GPU 数量兜底
			if minions[i].GPUCount == 0 && metrics.GPUCount > 0 {
				minions[i].GPUCount = metrics.GPUCount
				usedCachedData = true
			}
			// GPU 型号兜底
			if minions[i].GPUModel == "" && metrics.GPUModel != "" {
				minions[i].GPUModel = metrics.GPUModel
				usedCachedData = true
			}
			// IB 状态兜底（根据数据库中的活跃/总数计算状态）
			if minions[i].IBStatus == "" || minions[i].IBStatus == "not_installed" {
				if metrics.IBTotalCount > 0 {
					if metrics.IBActiveCount > 0 {
						minions[i].IBStatus = "active"
					} else {
						minions[i].IBStatus = "inactive"
					}
					usedCachedData = true
				}
			}
			// IB 数量兜底
			if minions[i].IBCount == 0 && metrics.IBTotalCount > 0 {
				minions[i].IBCount = metrics.IBTotalCount
				usedCachedData = true
			}
			// IB 速率兜底（从 IBPortsInfo 中解析，如果可用）
			if minions[i].IBRate == "" && metrics.IBPortsInfo != "" {
				// 尝试从 IBPortsInfo 中提取速率信息
				// IBPortsInfo 格式通常为 JSON，可能包含 rate 字段
				// 这里简单处理，如果需要复杂解析可以扩展
				usedCachedData = true
			}

			// 如果使用了缓存数据，且当前数据来源不是 realtime，则标记为 cached
			if usedCachedData && minions[i].DataSource != "realtime" {
				minions[i].DataSource = "cached"
			}
		}
	}
}

// parseManageStatus 解析 manage.status 的返回，得到 up/down 列表
func (h *SaltStackHandler) parseManageStatus(resp map[string]interface{}) (up []string, down []string) {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if u, ok := m["up"].([]interface{}); ok {
				for _, v := range u {
					if s, ok := v.(string); ok {
						up = append(up, s)
					}
				}
			}
			if d, ok := m["down"].([]interface{}); ok {
				for _, v := range d {
					if s, ok := v.(string); ok {
						down = append(down, s)
					}
				}
			}
		}
	}
	return
}

// parseWheelKeys 解析 key.list_all 的返回
func (h *SaltStackHandler) parseWheelKeys(resp map[string]interface{}) (minions, pre, rejected []string) {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if data, ok := m["data"].(map[string]interface{}); ok {
				if r, ok := data["return"].(map[string]interface{}); ok {
					toStrings := func(v interface{}) []string {
						res := []string{}
						if arr, ok := v.([]interface{}); ok {
							for _, it := range arr {
								if s, ok := it.(string); ok {
									res = append(res, s)
								}
							}
						}
						return res
					}
					minions = toStrings(r["minions"])
					pre = toStrings(r["minions_pre"])
					rejected = toStrings(r["minions_rejected"])
				}
			}
		}
	}
	return
}

// parseMinionGrains 从 GET /minions/{id} 的结果中提取 grains
func (h *SaltStackHandler) parseMinionGrains(resp map[string]interface{}, id string) map[string]interface{} {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if g, ok := m[id].(map[string]interface{}); ok {
				return g
			}
		}
	}
	return map[string]interface{}{}
}

// extractAPISaltVersion 从 GET / 的返回中尽量提取Salt版本
func (h *SaltStackHandler) extractAPISaltVersion(resp map[string]interface{}) string {
	// NetAPI 通常返回 {"return":[{"clients":[...]}]}
	// 版本信息不稳定，返回空或默认
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if v, ok := m["version"].(string); ok && v != "" {
				return v
			}
		}
	}
	return ""
}

// extractAPIVersion 尝试提取API版本（如无则为空）
func (h *SaltStackHandler) extractAPIVersion(resp map[string]interface{}) string {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if v, ok := m["api_version"].(string); ok && v != "" {
				return v
			}
		}
	}
	return ""
}

// ExecuteSaltCommand 执行Salt命令
func (h *SaltStackHandler) ExecuteSaltCommand(c *gin.Context) {
	var request struct {
		Target    string        `json:"target" binding:"required"`
		Function  string        `json:"function"`  // 支持 function 字段
		Fun       string        `json:"fun"`       // 兼容 fun 字段（前端常用）
		Arguments string        `json:"arguments"` // 支持字符串参数
		Arg       []interface{} `json:"arg"`       // 兼容 arg 数组格式
		TgtType   string        `json:"tgt_type"`  // 目标类型: glob, list, grain 等
		TaskID    string        `json:"task_id"`   // 前端传递的任务ID，用于关联作业历史
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 兼容处理: fun 和 function 字段，优先使用 function
	function := request.Function
	if function == "" {
		function = request.Fun
	}
	if function == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "function 或 fun 字段是必需的"})
		return
	}

	// 兼容处理: arguments 和 arg 字段
	var args []interface{}
	if request.Arguments != "" {
		args = []interface{}{request.Arguments}
	} else if len(request.Arg) > 0 {
		args = request.Arg
	}

	// 使用已有的 saltAPIClient 进行认证和请求
	client := h.newSaltAPIClient()
	if err := client.authenticate(); err != nil {
		log.Printf("[ERROR] Salt API 认证失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Salt API 认证失败: %v", err),
		})
		return
	}

	// 构建 payload，使用 local_async 模式获取 JID
	// 这样前端可以通过 JID 追踪作业历史
	payload := map[string]interface{}{
		"client": "local_async",
		"tgt":    request.Target,
		"fun":    function,
		"arg":    args,
	}
	if request.TgtType != "" {
		payload["tgt_type"] = request.TgtType
	}

	// 对 cmd.run 命令始终启用 python_shell
	if function == "cmd.run" {
		payload["kwarg"] = map[string]interface{}{
			"python_shell": true,
		}
	}

	// 调试：打印请求信息
	log.Printf("[DEBUG] Salt API 请求 (async): Payload=%+v", payload)

	// 使用 client 发送请求
	result, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		// 回退到 /run 端点
		if strings.Contains(strings.ToLower(err.Error()), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			result, err = client.makeRequest("/run", "POST", payload)
		}
	}
	if err != nil {
		log.Printf("[ERROR] Salt API 请求失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Salt API 请求失败: %v", err),
		})
		return
	}

	// 从异步响应中提取 JID
	// local_async 返回格式: {"return": [{"jid": "20231201...", "minions": ["minion1", ...]}]}
	var jid string
	var minions []string
	if ret, ok := result["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if j, ok := m["jid"].(string); ok {
				jid = j
			}
			if mins, ok := m["minions"].([]interface{}); ok {
				for _, min := range mins {
					if s, ok := min.(string); ok {
						minions = append(minions, s)
					}
				}
			}
		}
	}

	if jid == "" {
		log.Printf("[WARNING] Salt API 异步执行未返回 JID")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"result":  result,
		})
		return
	}

	log.Printf("[DEBUG] Salt 异步作业已提交: JID=%s, 目标minions=%v", jid, minions)

	// 从认证上下文获取用户名（安全审计关键字段）
	username := "unknown"
	if u, exists := c.Get("username"); exists {
		if s, ok := u.(string); ok && s != "" {
			username = s
		}
	}
	log.Printf("[AUDIT] Salt 命令执行: User=%s, JID=%s, Function=%s, Target=%s", username, jid, function, request.Target)

	// 保存作业信息到数据库和 Redis
	ctx := context.Background()
	argsJSON, _ := json.Marshal(args)

	// 通过服务层持久化到数据库
	saltJobService := services.GetSaltJobService()
	if saltJobService != nil {
		jobHistory := &models.SaltJobHistory{
			JID:       jid,
			TaskID:    request.TaskID,
			Function:  function,
			Arguments: string(argsJSON),
			Target:    request.Target,
			Status:    "running",
			User:      username, // 从认证上下文获取
			StartTime: time.Now(),
		}
		if err := saltJobService.CreateJob(ctx, jobHistory); err != nil {
			log.Printf("[WARNING] 保存作业到数据库失败: %v", err)
		}
	}

	// 同时保存到 Redis（用于快速查询和兼容性）
	if h.cache != nil {
		// 创建作业信息 JSON
		jobInfo := map[string]interface{}{
			"jid":        jid,
			"function":   function,
			"target":     request.Target,
			"arguments":  args,
			"start_time": time.Now().Format(time.RFC3339),
			"status":     "running",
			"user":       username, // 从认证上下文获取
		}
		if request.TaskID != "" {
			jobInfo["task_id"] = request.TaskID
		}
		jobInfoJSON, _ := json.Marshal(jobInfo)

		// 保存作业详情（以 JID 为 key）
		jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
		if err := h.cache.Set(ctx, jobDetailKey, string(jobInfoJSON), 7*24*time.Hour).Err(); err != nil {
			log.Printf("[WARNING] 保存作业详情到 Redis 失败: %v", err)
		}

		// 将 JID 添加到最近作业列表（用于补充 Salt API 返回的作业列表）
		recentJobsKey := "saltstack:recent_jobs"
		if err := h.cache.LPush(ctx, recentJobsKey, jid).Err(); err != nil {
			log.Printf("[WARNING] 添加 JID 到最近作业列表失败: %v", err)
		}
		// 保持最近作业列表不超过 100 条
		h.cache.LTrim(ctx, recentJobsKey, 0, 99)

		// 如果前端传递了 TaskID，保存双向映射
		if request.TaskID != "" {
			// 保存 JID -> TaskID 映射（用于通过JID查询TaskID）
			jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", jid)
			if err := h.cache.Set(ctx, jidToTaskKey, request.TaskID, 7*24*time.Hour).Err(); err != nil {
				log.Printf("[WARNING] 保存 JID->TaskID 映射到 Redis 失败: %v", err)
			} else {
				log.Printf("[DEBUG] 已保存 JID->TaskID 映射: %s -> %s", jid, request.TaskID)
			}

			// 保存 TaskID -> JID 映射（用于通过TaskID查询JID）
			taskToJidKey := fmt.Sprintf("saltstack:task_to_jid:%s", request.TaskID)
			if err := h.cache.Set(ctx, taskToJidKey, jid, 7*24*time.Hour).Err(); err != nil {
				log.Printf("[WARNING] 保存 TaskID->JID 映射到 Redis 失败: %v", err)
			} else {
				log.Printf("[DEBUG] 已保存 TaskID->JID 映射: %s -> %s", request.TaskID, jid)
			}
		}
	}

	// 轮询等待结果（最多等待 90 秒）
	maxWaitTime := 90 * time.Second
	pollInterval := 2 * time.Second
	startTime := time.Now()

	var finalResult map[string]interface{}
	for {
		if time.Since(startTime) > maxWaitTime {
			log.Printf("[WARNING] 等待作业 %s 结果超时", jid)
			break
		}

		// 查询作业结果
		lookupPayload := map[string]interface{}{
			"client": "runner",
			"fun":    "jobs.lookup_jid",
			"kwarg": map[string]interface{}{
				"jid": jid,
			},
		}

		jobResult, err := client.makeRequest("/", "POST", lookupPayload)
		if err != nil {
			log.Printf("[ERROR] 查询作业 %s 失败: %v", jid, err)
			time.Sleep(pollInterval)
			continue
		}

		// 解析结果: {"return": [{"minion1": result1, "minion2": result2}]}
		if ret, ok := jobResult["return"].([]interface{}); ok && len(ret) > 0 {
			if m, ok := ret[0].(map[string]interface{}); ok && len(m) > 0 {
				finalResult = m
				log.Printf("[DEBUG] 作业 %s 完成，收到 %d 个节点的结果", jid, len(m))

				// 统计成功和失败数量
				successCount := 0
				failedCount := 0
				for _, v := range m {
					if vMap, ok := v.(map[string]interface{}); ok {
						if retcode, ok := vMap["retcode"].(float64); ok && retcode != 0 {
							failedCount++
						} else {
							successCount++
						}
					} else {
						successCount++ // 简单结果视为成功
					}
				}

				// 更新数据库中的作业状态为 completed
				if saltJobService != nil {
					if err := saltJobService.CompleteJob(ctx, jid, m, successCount, failedCount); err != nil {
						log.Printf("[WARNING] 更新数据库作业状态失败: %v", err)
					}
				}

				// 更新 Redis 中的作业状态为 completed
				if h.cache != nil {
					jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
					jobInfoJSON, err := h.cache.Get(ctx, jobDetailKey).Result()
					if err == nil {
						var jobInfo map[string]interface{}
						if json.Unmarshal([]byte(jobInfoJSON), &jobInfo) == nil {
							jobInfo["status"] = "completed"
							jobInfo["end_time"] = time.Now().Format(time.RFC3339)
							jobInfo["result"] = m
							jobInfo["success_count"] = successCount
							jobInfo["failed_count"] = failedCount
							if updatedJSON, err := json.Marshal(jobInfo); err == nil {
								h.cache.Set(ctx, jobDetailKey, string(updatedJSON), 7*24*time.Hour)
								log.Printf("[DEBUG] 已更新作业 %s 状态为 completed", jid)
							}
						}
					}
				}

				break
			}
		}

		time.Sleep(pollInterval)
	}

	// 如果轮询超时，更新状态为 timeout
	if finalResult == nil {
		// 更新数据库中的作业状态为 timeout
		if saltJobService != nil {
			if err := saltJobService.TimeoutJob(ctx, jid); err != nil {
				log.Printf("[WARNING] 更新数据库作业超时状态失败: %v", err)
			}
		}

		// 更新 Redis 中的作业状态为 timeout
		if h.cache != nil {
			jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
			jobInfoJSON, err := h.cache.Get(ctx, jobDetailKey).Result()
			if err == nil {
				var jobInfo map[string]interface{}
				if json.Unmarshal([]byte(jobInfoJSON), &jobInfo) == nil {
					jobInfo["status"] = "timeout"
					jobInfo["end_time"] = time.Now().Format(time.RFC3339)
					if updatedJSON, err := json.Marshal(jobInfo); err == nil {
						h.cache.Set(ctx, jobDetailKey, string(updatedJSON), 7*24*time.Hour)
						log.Printf("[DEBUG] 已更新作业 %s 状态为 timeout", jid)
					}
				}
			}
		}
	}

	// 返回执行结果，包含 JID 和 TaskID 用于前端追踪
	log.Printf("[DEBUG] 成功返回 Salt 执行结果，JID=%s, TaskID=%s", jid, request.TaskID)
	response := gin.H{
		"success": true,
		"jid":     jid,
		"result": map[string]interface{}{
			"return": []interface{}{finalResult},
		},
	}
	// 如果有 TaskID，也返回给前端
	if request.TaskID != "" {
		response["task_id"] = request.TaskID
	}
	c.JSON(http.StatusOK, response)
}

// getSaltAuthToken 获取 Salt API 认证 token
func (h *SaltStackHandler) getSaltAuthToken(ctx context.Context) string {
	// 尝试从 Redis 缓存获取
	cacheKey := "salt:auth:token"
	if h.cache != nil {
		if token, err := h.cache.Get(ctx, cacheKey).Result(); err == nil && token != "" {
			return token
		}
	}

	// 从环境变量获取认证信息
	saltMaster := os.Getenv("SALTSTACK_MASTER_HOST")
	if saltMaster == "" {
		saltMaster = "saltstack"
	}
	saltAPIPort := os.Getenv("SALT_API_PORT")
	if saltAPIPort == "" {
		saltAPIPort = "8002"
	}
	saltAPIScheme := os.Getenv("SALT_API_SCHEME")
	if saltAPIScheme == "" {
		saltAPIScheme = "http"
	}
	username := os.Getenv("SALT_API_USERNAME")
	if username == "" {
		username = "saltapi"
	}
	password := os.Getenv("SALT_API_PASSWORD")
	if password == "" {
		return "" // 没有密码无法登录
	}
	eauth := os.Getenv("SALT_API_EAUTH")
	if eauth == "" {
		eauth = "file"
	}

	// 登录获取新 token
	loginURL := fmt.Sprintf("%s://%s:%s/login", saltAPIScheme, saltMaster, saltAPIPort)
	loginPayload := map[string]interface{}{
		"username": username,
		"password": password,
		"eauth":    eauth,
	}

	payloadBytes, _ := json.Marshal(loginPayload)
	resp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var loginResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return ""
	}

	// 提取 token
	if returnData, ok := loginResp["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if token, ok := tokenData["token"].(string); ok && token != "" {
				// 缓存 token（12小时）
				if h.cache != nil {
					h.cache.Set(ctx, cacheKey, token, 12*time.Hour)
				}
				return token
			}
		}
	}

	return ""
}

// ExecuteCustomCommandAsync 异步执行自定义 Bash/Python 命令（通过 Salt cmd.run 下发）
// 请求: { target: string, language: "bash"|"python", code: string, timeout?: int }
// 返回: { opId: string }
func (h *SaltStackHandler) ExecuteCustomCommandAsync(c *gin.Context) {
	var req struct {
		Target   string `json:"target"`
		Language string `json:"language"`
		Code     string `json:"code"`
		Timeout  int    `json:"timeout"`
		User     string `json:"user,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 基础校验
	if req.Target == "" {
		req.Target = "*"
	}
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if req.Language != "bash" && req.Language != "python" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language must be 'bash' or 'python'"})
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	if len(code) > 20000 { // 保护性限制
		c.JSON(http.StatusBadRequest, gin.H{"error": "code too long (max 20000 chars)"})
		return
	}

	// 轻量格式校验（前端也会做一次）
	if err := validateScriptFormat(req.Language, code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("format check failed: %v", err)})
		return
	}

	pm := services.GetProgressManager()
	op := pm.Start("salt:execute-custom", "开始下发自定义命令")

	// Avoid anonymous-struct type mismatch by capturing the bound request
	r := req
	go func(opID string) {
		failed := false
		defer func() { pm.Complete(opID, failed, "命令执行完成") }()

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "prepare", Message: "准备连接 Salt API"})

		client := h.newSaltAPIClient()
		if err := client.authenticate(); err != nil {
			failed = true
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "auth", Message: fmt.Sprintf("Salt API 认证失败: %v", err)})
			return
		}

		// 构造 cmd.run 参数
		var cmd string
		if r.Language == "bash" {
			cmd = "bash -s"
		} else {
			cmd = "python3 -"
		}

		kwarg := map[string]interface{}{
			"stdin":        r.Code,
			"python_shell": true,
		}
		if r.Timeout > 0 {
			kwarg["timeout"] = r.Timeout
		}

		payload := map[string]interface{}{
			"client": "local",
			"tgt":    r.Target,
			"fun":    "cmd.run",
			"arg":    []interface{}{cmd},
			"kwarg":  kwarg,
		}

		pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "dispatch", Message: fmt.Sprintf("下发到目标: %s", r.Target)})

		// 通过 NetAPI 执行
		start := time.Now()
		res, err := client.makeRequest("/", "POST", payload)
		if err != nil {
			// 回退到 /run
			if strings.Contains(strings.ToLower(err.Error()), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
				res, err = client.makeRequest("/run", "POST", payload)
			}
		}
		duration := time.Since(start)
		if err != nil {
			failed = true
			pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "execute", Message: fmt.Sprintf("执行失败: %v", err)})
			return
		}

		// 解析返回，并按 minion 逐项记录
		results := extractLocalResults(res)
		if len(results) == 0 {
			pm.Emit(opID, services.ProgressEvent{Type: "step-log", Step: "result", Message: "无返回或解析失败", Data: res})
		} else {
			per := 0.0
			step := 0
			for minion, output := range results {
				step++
				per = float64(step) / float64(len(results))
				pm.Emit(opID, services.ProgressEvent{Type: "step-log", Step: "result", Host: minion, Progress: per, Message: "命令输出", Data: map[string]interface{}{"stdout": output}})
			}
		}
		pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "dispatch", Message: fmt.Sprintf("执行完成，用时 %dms", duration.Milliseconds()), Data: res})
	}(op.ID)

	c.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// GetProgress 返回异步操作的快照
func (h *SaltStackHandler) GetProgress(c *gin.Context) {
	opID := c.Param("opId")
	pm := services.GetProgressManager()
	snap, ok := pm.Snapshot(opID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": snap})
}

// StreamProgress 以SSE流式输出进度
func (h *SaltStackHandler) StreamProgress(c *gin.Context) {
	opID := c.Param("opId")
	pm := services.GetProgressManager()
	ch, ok := pm.Subscribe(opID)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 初始快照
	if snap, ok := pm.Snapshot(opID); ok {
		b, _ := json.Marshal(snap)
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
		flusher.Flush()
	}

	notify := c.Writer.CloseNotify()
	for {
		select {
		case <-notify:
			return
		case ev, more := <-ch:
			if !more {
				return
			}
			b, _ := json.Marshal(ev)
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
			flusher.Flush()
		}
	}
}

// --- 辅助函数 ---

func validateScriptFormat(lang, code string) error {
	// 基础防御性校验：控制字符与大体结构
	if strings.ContainsRune(code, '\u0000') {
		return errors.New("code contains NUL byte")
	}
	// 简单括号与引号平衡检查（尽力而为，不保证完全准确）
	single := 0
	double := 0
	for i := 0; i < len(code); i++ {
		ch := code[i]
		if ch == '\'' {
			single ^= 1
		} else if ch == '"' {
			double ^= 1
		}
	}
	if single != 0 || double != 0 {
		return errors.New("unbalanced quotes detected")
	}
	// 针对 python：检查缩进混用的一些明显问题
	if lang == "python" {
		lines := strings.Split(code, "\n")
		for _, ln := range lines {
			if strings.HasPrefix(ln, "\t") && strings.HasPrefix(strings.TrimLeft(ln, "\t"), " ") {
				return errors.New("mixed tabs and spaces in indentation")
			}
		}
	}
	return nil
}

// extractLocalResults 尽力从 local 执行的返回中解析出 minion->输出 的映射
func extractLocalResults(resp map[string]interface{}) map[string]string {
	out := map[string]string{}
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			for k, v := range m {
				switch vv := v.(type) {
				case string:
					out[k] = vv
				case map[string]interface{}:
					// 某些模块会返回结构化结果，尝试提取 stdout/ret
					if s, ok := vv["stdout"].(string); ok {
						out[k] = s
					} else if s, ok := vv["ret"].(string); ok {
						out[k] = s
					} else {
						b, _ := json.Marshal(vv)
						out[k] = string(b)
					}
				default:
					b, _ := json.Marshal(v)
					out[k] = string(b)
				}
			}
		}
	}
	return out
}

// ==================== Minion 分组管理 API ====================

// ListMinionGroups 获取所有分组
func (h *SaltStackHandler) ListMinionGroups(c *gin.Context) {
	groups, err := h.minionGroupService.ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": groups})
}

// CreateMinionGroup 创建分组
func (h *SaltStackHandler) CreateMinionGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Priority    int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	group := &models.MinionGroup{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Color:       req.Color,
		Priority:    req.Priority,
	}
	if group.DisplayName == "" {
		group.DisplayName = group.Name
	}
	if group.Color == "" {
		group.Color = "blue"
	}

	if err := h.minionGroupService.CreateGroup(group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": group})
}

// UpdateMinionGroup 更新分组
func (h *SaltStackHandler) UpdateMinionGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid group id"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 过滤允许更新的字段
	updates := make(map[string]interface{})
	allowedFields := []string{"display_name", "description", "color", "priority", "name"}
	for _, field := range allowedFields {
		if val, ok := req[field]; ok {
			updates[field] = val
		}
	}

	if err := h.minionGroupService.UpdateGroup(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "updated"})
}

// DeleteMinionGroup 删除分组
func (h *SaltStackHandler) DeleteMinionGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid group id"})
		return
	}

	if err := h.minionGroupService.DeleteGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "deleted"})
}

// SetMinionGroup 设置 Minion 的分组
func (h *SaltStackHandler) SetMinionGroup(c *gin.Context) {
	var req struct {
		MinionID  string `json:"minion_id" binding:"required"`
		GroupName string `json:"group_name"` // 空字符串表示移除分组
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.minionGroupService.SetMinionGroup(req.MinionID, req.GroupName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 清除 minions 缓存，使分组更新立即生效
	h.cache.Del(context.Background(), "saltstack:minions")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "group set successfully"})
}

// BatchSetMinionGroups 批量设置 Minion 分组
func (h *SaltStackHandler) BatchSetMinionGroups(c *gin.Context) {
	var req struct {
		MinionGroups map[string]string `json:"minion_groups" binding:"required"` // minionID -> groupName
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.minionGroupService.BatchSetMinionGroups(req.MinionGroups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 清除 minions 缓存
	h.cache.Del(context.Background(), "saltstack:minions")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "groups set successfully", "count": len(req.MinionGroups)})
}

// GetGroupMinions 获取分组内的 Minion 列表
func (h *SaltStackHandler) GetGroupMinions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid group id"})
		return
	}

	minionIDs, err := h.minionGroupService.GetGroupMinions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": minionIDs})
}

// GetMinionDetails 获取 Minion 详细信息（包含 GPU/NPU 信息）
// @Summary 获取Minion详细信息
// @Description 获取指定 Minion 的详细信息，包括内核版本、GPU/NPU 驱动信息
// @Tags SaltStack
// @Produce json
// @Param minionId path string true "Minion ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minions/{minionId}/details [get]
func (h *SaltStackHandler) GetMinionDetails(c *gin.Context) {
	minionID := c.Param("minionId")
	if minionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Minion ID is required"})
		return
	}

	// 创建 API 客户端
	client := h.newSaltAPIClient()
	if err := client.authenticate(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": fmt.Sprintf("Salt API 认证失败: %v", err)})
		return
	}

	// 获取 grains 信息
	grainsResp, err := client.makeRequest("/minions/"+minionID, "GET", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": fmt.Sprintf("获取 Minion 信息失败: %v", err)})
		return
	}
	grains := h.parseMinionGrains(grainsResp, minionID)

	// 构建基本 Minion 信息
	minion := SaltMinion{
		ID:            minionID,
		Status:        "up",
		OS:            fmt.Sprintf("%v", grains["os"]),
		OSVersion:     fmt.Sprintf("%v", grains["osrelease"]),
		Architecture:  fmt.Sprintf("%v", grains["osarch"]),
		Arch:          fmt.Sprintf("%v", grains["osarch"]),
		SaltVersion:   fmt.Sprintf("%v", grains["saltversion"]),
		LastSeen:      time.Now(),
		Grains:        grains,
		KernelVersion: h.extractKernelVersion(grains),
	}

	// 并行获取 GPU 和 NPU 信息
	var wg sync.WaitGroup
	var gpuInfo GPUInfo
	var npuInfo NPUInfo

	wg.Add(2)

	// 获取 GPU 信息 (nvidia-smi)
	go func() {
		defer wg.Done()
		gpuInfo = h.getGPUInfo(client, minionID)
	}()

	// 获取 NPU 信息 (npu-smi)
	go func() {
		defer wg.Done()
		npuInfo = h.getNPUInfo(client, minionID)
	}()

	wg.Wait()

	// 填充 GPU/NPU 信息
	if gpuInfo.DriverVersion != "" {
		minion.GPUDriverVersion = gpuInfo.DriverVersion
		minion.CUDAVersion = gpuInfo.CUDAVersion
		minion.GPUCount = gpuInfo.GPUCount
		minion.GPUModel = gpuInfo.GPUModel
	}

	if npuInfo.Version != "" {
		minion.NPUVersion = npuInfo.Version
		minion.NPUCount = npuInfo.NPUCount
		minion.NPUModel = npuInfo.NPUModel
	}

	// 获取分组信息
	if groupName := h.minionGroupService.GetMinionGroupName(minionID); groupName != "" {
		minion.Group = groupName
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": minion})
}

// GPUInfo NVIDIA GPU 信息结构
type GPUInfo struct {
	DriverVersion string `json:"driver_version"`
	CUDAVersion   string `json:"cuda_version"`
	GPUCount      int    `json:"gpu_count"`
	GPUModel      string `json:"gpu_model"`
}

// NPUInfo NPU 信息结构 (华为昇腾、寒武纪等)
type NPUInfo struct {
	Vendor   string `json:"vendor"`    // huawei, cambricon, iluvatar
	Version  string `json:"version"`   // 驱动/SMI 版本
	NPUCount int    `json:"npu_count"` // NPU 数量
	NPUModel string `json:"npu_model"` // NPU 型号
}

// TPUInfo TPU 或其他 AI 加速器信息结构
type TPUInfo struct {
	Vendor   string `json:"vendor"`    // google, 或其他厂商
	Version  string `json:"version"`   // 驱动版本
	TPUCount int    `json:"tpu_count"` // TPU 数量
	TPUModel string `json:"tpu_model"` // TPU 型号
}

// AcceleratorInfo 综合加速器信息结构
type AcceleratorInfo struct {
	GPU *GPUInfo `json:"gpu,omitempty"`
	NPU *NPUInfo `json:"npu,omitempty"`
	TPU *TPUInfo `json:"tpu,omitempty"`
}

// extractKernelVersion 从 grains 中提取内核版本
func (h *SaltStackHandler) extractKernelVersion(grains map[string]interface{}) string {
	// 优先使用 kernelrelease
	if kr, ok := grains["kernelrelease"].(string); ok && kr != "" {
		return kr
	}
	// 回退到 kernel
	if k, ok := grains["kernel"].(string); ok && k != "" {
		return k
	}
	return ""
}

// getGPUInfo 通过 nvidia-smi 获取 GPU 信息
func (h *SaltStackHandler) getGPUInfo(client *saltAPIClient, minionID string) GPUInfo {
	info := GPUInfo{}

	// 执行 nvidia-smi 命令获取驱动版本和 CUDA 版本
	// nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits
	// nvidia-smi --query-gpu=name,count --format=csv,noheader
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits 2>/dev/null | head -1"},
		"kwarg": map[string]interface{}{
			"timeout":      10,
			"python_shell": true,
		},
	}

	resp, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		return info
	}

	// 解析驱动版本
	if output := h.extractCmdOutput(resp, minionID); output != "" && !strings.Contains(strings.ToLower(output), "not found") && !strings.Contains(strings.ToLower(output), "error") {
		info.DriverVersion = strings.TrimSpace(output)
	}

	// 获取 CUDA 版本
	// nvidia-smi 的输出头部包含 CUDA Version
	// 例如: CUDA Version: 13.0
	cudaPayload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \\K[0-9.]+' | head -1"},
		"kwarg": map[string]interface{}{
			"timeout":      10,
			"python_shell": true,
		},
	}
	cudaResp, err := client.makeRequest("/", "POST", cudaPayload)
	if err == nil {
		if cudaOutput := h.extractCmdOutput(cudaResp, minionID); cudaOutput != "" {
			info.CUDAVersion = strings.TrimSpace(cudaOutput)
		}
	}

	// 获取 GPU 数量和型号
	countPayload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"nvidia-smi --query-gpu=name,count --format=csv,noheader 2>/dev/null | head -1"},
		"kwarg": map[string]interface{}{
			"timeout":      10,
			"python_shell": true,
		},
	}
	countResp, err := client.makeRequest("/", "POST", countPayload)
	if err == nil {
		if countOutput := h.extractCmdOutput(countResp, minionID); countOutput != "" {
			// 输出格式: "NVIDIA H100 80GB HBM3, 8"
			parts := strings.Split(countOutput, ",")
			if len(parts) >= 1 {
				info.GPUModel = strings.TrimSpace(parts[0])
			}
		}
	}

	// 获取 GPU 数量
	gpuCountPayload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"nvidia-smi -L 2>/dev/null | wc -l"},
		"kwarg": map[string]interface{}{
			"timeout":      10,
			"python_shell": true,
		},
	}
	gpuCountResp, err := client.makeRequest("/", "POST", gpuCountPayload)
	if err == nil {
		if gpuCountOutput := h.extractCmdOutput(gpuCountResp, minionID); gpuCountOutput != "" {
			if count, parseErr := strconv.Atoi(strings.TrimSpace(gpuCountOutput)); parseErr == nil {
				info.GPUCount = count
			}
		}
	}

	return info
}

// getNPUInfo 通过 npu-smi 获取华为 NPU 信息
func (h *SaltStackHandler) getNPUInfo(client *saltAPIClient, minionID string) NPUInfo {
	info := NPUInfo{}

	// 先检测华为昇腾 NPU (npu-smi)
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"npu-smi info 2>/dev/null"},
		"kwarg": map[string]interface{}{
			"timeout": 10,
		},
	}

	resp, err := client.makeRequest("/", "POST", payload)
	if err == nil {
		output := h.extractCmdOutput(resp, minionID)
		if output != "" && !strings.Contains(strings.ToLower(output), "not found") && !strings.Contains(strings.ToLower(output), "command not found") {
			info.Vendor = "huawei"
			// 解析 npu-smi 输出
			// 格式: | npu-smi 24.1.1                   Version: 24.1.1 |
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				// 解析版本号
				if strings.Contains(line, "npu-smi") && strings.Contains(line, "Version:") {
					if idx := strings.Index(line, "Version:"); idx != -1 {
						versionPart := strings.TrimSpace(line[idx+len("Version:"):])
						versionPart = strings.TrimSuffix(strings.TrimSpace(versionPart), "|")
						info.Version = strings.TrimSpace(versionPart)
					}
				}
				// 解析 NPU 型号和数量
				// 格式: | 0     910B3     OK ...
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "|") && !strings.Contains(line, "NPU") && !strings.Contains(line, "npu-smi") && !strings.Contains(line, "---") && !strings.Contains(line, "Chip") {
					fields := strings.Fields(strings.Trim(line, "|"))
					if len(fields) >= 2 {
						if _, parseErr := strconv.Atoi(fields[0]); parseErr == nil {
							info.NPUCount++
							if info.NPUModel == "" {
								info.NPUModel = fields[1]
							}
						}
					}
				}
			}
			if info.NPUCount > 0 {
				return info
			}
		}
	}

	// 检测寒武纪 MLU (cnmon)
	cnmonPayload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"cnmon info 2>/dev/null"},
		"kwarg": map[string]interface{}{
			"timeout": 10,
		},
	}

	cnmonResp, err := client.makeRequest("/", "POST", cnmonPayload)
	if err == nil {
		output := h.extractCmdOutput(cnmonResp, minionID)
		if output != "" && !strings.Contains(strings.ToLower(output), "not found") && !strings.Contains(strings.ToLower(output), "command not found") {
			info.Vendor = "cambricon"
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "driver version") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						info.Version = strings.TrimSpace(parts[1])
					}
				}
				if strings.Contains(line, "MLU") && !strings.Contains(strings.ToLower(line), "driver") {
					info.NPUCount++
				}
				if strings.Contains(strings.ToLower(line), "product name") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 && info.NPUModel == "" {
						info.NPUModel = strings.TrimSpace(parts[1])
					}
				}
			}
			if info.NPUCount > 0 {
				return info
			}
		}
	}

	// 检测天数智芯 GPU (ixsmi)
	ixsmiPayload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"ixsmi -L 2>/dev/null"},
		"kwarg": map[string]interface{}{
			"timeout": 10,
		},
	}

	ixsmiResp, err := client.makeRequest("/", "POST", ixsmiPayload)
	if err == nil {
		output := h.extractCmdOutput(ixsmiResp, minionID)
		if output != "" && !strings.Contains(strings.ToLower(output), "not found") && !strings.Contains(strings.ToLower(output), "command not found") {
			info.Vendor = "iluvatar"
			// 统计输出行数作为 GPU 数量
			lines := strings.Split(strings.TrimSpace(output), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					info.NPUCount++
				}
			}
			if info.NPUCount > 0 {
				return info
			}
		}
	}

	return info
}

// IBInfo InfiniBand 网络信息结构
type IBInfo struct {
	Status string `json:"status"` // active, inactive, not_installed
	Count  int    `json:"count"`
	Rate   string `json:"rate"` // 如 "200 Gb/sec (4X HDR)"
}

// getIBInfo 通过 ibstat 获取 InfiniBand 信息
func (h *SaltStackHandler) getIBInfo(client *saltAPIClient, minionID string) IBInfo {
	info := IBInfo{Status: "not_installed"}

	// 检查 ibstat 命令是否存在
	checkPayload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"which ibstat 2>/dev/null || command -v ibstat 2>/dev/null"},
		"kwarg": map[string]interface{}{
			"timeout":      10,
			"python_shell": true,
		},
	}

	checkResp, err := client.makeRequest("/", "POST", checkPayload)
	if err != nil {
		return info
	}

	checkOutput := h.extractCmdOutput(checkResp, minionID)
	if checkOutput == "" || strings.Contains(strings.ToLower(checkOutput), "not found") {
		// ibstat 不存在，表示未安装 InfiniBand
		return info
	}

	// 执行 ibstat 获取 IB 状态
	// 输出格式:
	// CA 'mlx5_0'
	//     CA type: MT4123
	//     Number of ports: 1
	//     ...
	//     Port 1:
	//         State: Active
	//         Physical state: LinkUp
	//         Rate: 200
	//         Base lid: 0
	//         ...
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{"ibstat 2>/dev/null"},
		"kwarg": map[string]interface{}{
			"timeout":      15,
			"python_shell": true,
		},
	}

	resp, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		info.Status = "unavailable"
		return info
	}

	output := h.extractCmdOutput(resp, minionID)
	if output == "" || strings.Contains(strings.ToLower(output), "error") {
		info.Status = "unavailable"
		return info
	}

	// 解析 ibstat 输出
	lines := strings.Split(output, "\n")
	activeCount := 0
	totalCount := 0
	var rate string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 统计 CA 数量
		if strings.HasPrefix(line, "CA '") {
			totalCount++
		}

		// 检查端口状态
		if strings.HasPrefix(line, "State:") {
			state := strings.TrimSpace(strings.TrimPrefix(line, "State:"))
			if strings.ToLower(state) == "active" {
				activeCount++
			}
		}

		// 获取速率
		if strings.HasPrefix(line, "Rate:") && rate == "" {
			rateValue := strings.TrimSpace(strings.TrimPrefix(line, "Rate:"))
			if rateValue != "" && rateValue != "0" {
				rate = rateValue + " Gb/sec"
			}
		}
	}

	info.Count = totalCount
	if totalCount > 0 {
		if activeCount > 0 {
			info.Status = "active"
		} else {
			info.Status = "inactive"
		}
	}
	info.Rate = rate

	return info
}

// CPUMemoryInfo CPU和内存使用率信息结构
type CPUMemoryInfo struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	MemoryTotalGB      float64 `json:"memory_total_gb"`
	MemoryUsedGB       float64 `json:"memory_used_gb"`
}

// getCPUMemoryInfo 通过 Salt 命令获取 CPU 和内存使用率信息
func (h *SaltStackHandler) getCPUMemoryInfo(client *saltAPIClient, minionID string) CPUMemoryInfo {
	info := CPUMemoryInfo{}

	// 从嵌入脚本文件读取，避免硬编码
	script, err := scripts.GetCPUMemoryScript()
	if err != nil {
		log.Printf("[SaltStack] 读取脚本文件失败: %v", err)
		return info
	}

	payload := map[string]interface{}{
		"client": "local",
		"tgt":    minionID,
		"fun":    "cmd.run",
		"arg":    []interface{}{script},
		"kwarg": map[string]interface{}{
			"timeout":      10,
			"python_shell": true, // 必须启用 shell 模式以支持管道和复杂命令
		},
	}

	resp, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		return info
	}

	output := h.extractCmdOutput(resp, minionID)
	if output == "" || strings.Contains(strings.ToLower(output), "error") {
		return info
	}

	// 解析输出: cpu_percent|mem_total_kb|mem_available_kb
	output = strings.TrimSpace(output)
	// 处理多行输出（脚本可能有额外输出），只取最后一行
	lines := strings.Split(output, "\n")
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	parts := strings.Split(lastLine, "|")
	if len(parts) >= 3 {
		// CPU 使用率
		if cpuPercent, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			info.CPUUsagePercent = cpuPercent
		}

		// 内存总量 (KB -> GB)
		var memTotalKB float64
		if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil && val > 0 {
			memTotalKB = val
			info.MemoryTotalGB = memTotalKB / 1024 / 1024
		}

		// 内存可用量 (KB -> GB) 及使用率计算
		if memAvailKB, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil && memTotalKB > 0 {
			memUsedKB := memTotalKB - memAvailKB
			info.MemoryUsedGB = memUsedKB / 1024 / 1024
			info.MemoryUsagePercent = (memUsedKB / memTotalKB) * 100
		}
	}

	return info
}

// extractCmdOutput 从 cmd.run 的响应中提取输出
func (h *SaltStackHandler) extractCmdOutput(resp map[string]interface{}, minionID string) string {
	if ret, ok := resp["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			if output, ok := m[minionID].(string); ok {
				return output
			}
		}
	}
	return ""
}

// ================== 批量为 Minion 安装 Categraf ==================

// categrafInstallTask 存储 Categraf 安装任务信息
type categrafInstallTask struct {
	TaskID    string              `json:"task_id"`
	MinionIDs []string            `json:"minion_ids"`
	Status    string              `json:"status"` // pending, running, completed, failed
	Events    []map[string]string `json:"events"`
	StartTime time.Time           `json:"start_time"`
}

var categrafInstallTasks = make(map[string]*categrafInstallTask)
var categrafInstallTasksMutex sync.RWMutex

// InstallCategrafOnMinions 批量为 Minion 安装 Categraf
// @Summary 批量为 Minion 安装 Categraf
// @Description 通过 Salt State 在指定的 Minion 上安装 Categraf 监控代理
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param body body object true "安装请求" example({"minion_ids": ["minion1", "minion2"]})
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/minions/install-categraf [post]
func (h *SaltStackHandler) InstallCategrafOnMinions(c *gin.Context) {
	var req struct {
		MinionIDs []string `json:"minion_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if len(req.MinionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "minion_ids cannot be empty"})
		return
	}

	// 生成任务 ID
	taskID := fmt.Sprintf("categraf-%d", time.Now().UnixNano())

	// 创建任务记录
	task := &categrafInstallTask{
		TaskID:    taskID,
		MinionIDs: req.MinionIDs,
		Status:    "pending",
		Events:    make([]map[string]string, 0),
		StartTime: time.Now(),
	}

	categrafInstallTasksMutex.Lock()
	categrafInstallTasks[taskID] = task
	categrafInstallTasksMutex.Unlock()

	// 异步执行安装
	go h.executeCategrafInstall(taskID, req.MinionIDs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Categraf installation task created",
		"data": gin.H{
			"task_id":    taskID,
			"minion_ids": req.MinionIDs,
		},
	})
}

// executeCategrafInstall 异步执行 Categraf 安装
func (h *SaltStackHandler) executeCategrafInstall(taskID string, minionIDs []string) {
	categrafInstallTasksMutex.Lock()
	task := categrafInstallTasks[taskID]
	if task == nil {
		categrafInstallTasksMutex.Unlock()
		return
	}
	task.Status = "running"
	categrafInstallTasksMutex.Unlock()

	defer func() {
		categrafInstallTasksMutex.Lock()
		if task.Status == "running" {
			task.Status = "completed"
		}
		categrafInstallTasksMutex.Unlock()
	}()

	// 添加开始事件
	h.addCategrafEvent(taskID, map[string]string{
		"type":      "start",
		"message":   fmt.Sprintf("Starting Categraf installation on %d minions", len(minionIDs)),
		"timestamp": time.Now().Format(time.RFC3339),
	})

	for _, minionID := range minionIDs {
		h.addCategrafEvent(taskID, map[string]string{
			"type":      "running",
			"status":    "running",
			"minion_id": minionID,
			"message":   fmt.Sprintf("Installing Categraf on %s...", minionID),
			"timestamp": time.Now().Format(time.RFC3339),
		})

		// 通过 Salt State 安装 Categraf
		// 使用 state.apply 命令执行 categraf state
		payload := map[string]interface{}{
			"client": "local",
			"tgt":    minionID,
			"fun":    "state.apply",
			"arg":    []string{"categraf"},
		}

		client := h.newSaltAPIClient()
		resp, err := client.makeRequest("/", "POST", payload)
		if err != nil {
			h.addCategrafEvent(taskID, map[string]string{
				"type":      "error",
				"status":    "error",
				"minion_id": minionID,
				"message":   fmt.Sprintf("Failed to install Categraf on %s: %v", minionID, err),
				"timestamp": time.Now().Format(time.RFC3339),
			})
			continue
		}

		// 解析响应，检查是否成功
		success, message := h.checkStateApplyResult(resp, minionID)
		if success {
			h.addCategrafEvent(taskID, map[string]string{
				"type":      "success",
				"status":    "success",
				"minion_id": minionID,
				"message":   fmt.Sprintf("Categraf installed successfully on %s: %s", minionID, message),
				"timestamp": time.Now().Format(time.RFC3339),
			})
		} else {
			h.addCategrafEvent(taskID, map[string]string{
				"type":      "error",
				"status":    "error",
				"minion_id": minionID,
				"message":   fmt.Sprintf("Categraf installation failed on %s: %s", minionID, message),
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}
	}

	// 添加完成事件
	h.addCategrafEvent(taskID, map[string]string{
		"type":      "complete",
		"message":   "Categraf installation completed",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// addCategrafEvent 添加 Categraf 安装事件
func (h *SaltStackHandler) addCategrafEvent(taskID string, event map[string]string) {
	categrafInstallTasksMutex.Lock()
	defer categrafInstallTasksMutex.Unlock()

	if task, ok := categrafInstallTasks[taskID]; ok {
		task.Events = append(task.Events, event)
	}
}

// checkStateApplyResult 检查 state.apply 的结果
func (h *SaltStackHandler) checkStateApplyResult(resp map[string]interface{}, minionID string) (bool, string) {
	ret, ok := resp["return"].([]interface{})
	if !ok || len(ret) == 0 {
		return false, "empty response"
	}

	minionResp, ok := ret[0].(map[string]interface{})
	if !ok {
		return false, "invalid response format"
	}

	minionResult, ok := minionResp[minionID]
	if !ok {
		return false, "no result for minion"
	}

	// 检查是否所有 state 都成功
	switch result := minionResult.(type) {
	case map[string]interface{}:
		allSuccess := true
		var messages []string
		for _, stateResult := range result {
			if sr, ok := stateResult.(map[string]interface{}); ok {
				if success, ok := sr["result"].(bool); ok && !success {
					allSuccess = false
					if comment, ok := sr["comment"].(string); ok {
						messages = append(messages, comment)
					}
				}
			}
		}
		if allSuccess {
			return true, "all states applied successfully"
		}
		return false, strings.Join(messages, "; ")
	case string:
		// 可能是错误消息
		return false, result
	default:
		return false, "unknown response type"
	}
}

// CategrafInstallStream SSE 流式返回 Categraf 安装进度
// @Summary Categraf 安装进度流
// @Description 通过 SSE 流式返回 Categraf 安装任务的进度
// @Tags SaltStack
// @Produce text/event-stream
// @Param task_id path string true "任务 ID"
// @Success 200 {string} string "event-stream"
// @Router /api/saltstack/minions/install-categraf/{task_id}/stream [get]
func (h *SaltStackHandler) CategrafInstallStream(c *gin.Context) {
	taskID := c.Param("task_id")

	categrafInstallTasksMutex.RLock()
	task := categrafInstallTasks[taskID]
	categrafInstallTasksMutex.RUnlock()

	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "task not found"})
		return
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 发送已有事件
	sentIndex := 0
	for {
		categrafInstallTasksMutex.RLock()
		task = categrafInstallTasks[taskID]
		if task == nil {
			categrafInstallTasksMutex.RUnlock()
			break
		}
		events := task.Events
		status := task.Status
		categrafInstallTasksMutex.RUnlock()

		// 发送新事件
		for i := sentIndex; i < len(events); i++ {
			eventData, _ := json.Marshal(events[i])
			c.SSEvent("message", string(eventData))
			c.Writer.Flush()
			sentIndex = i + 1
		}

		// 任务完成则退出
		if status == "completed" || status == "failed" {
			break
		}

		// 等待新事件
		time.Sleep(200 * time.Millisecond)
	}

	// 发送关闭事件
	closeEvent, _ := json.Marshal(map[string]string{
		"type":      "closed",
		"message":   "Stream closed",
		"timestamp": time.Now().Format(time.RFC3339),
	})
	c.SSEvent("message", string(closeEvent))
	c.Writer.Flush()
}

// ==================== 节点指标采集相关 ====================

// NodeMetricsCallback 接收节点指标回调
// @Summary 接收节点指标回调
// @Description 接收从 Salt Minion 定期采集的 CPU/内存/网络/GPU/IB/RoCE 等硬件信息。支持可选的 API Token 认证。
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param X-API-Token header string false "API Token（可选，如配置 NODE_METRICS_API_TOKEN 环境变量则必须提供）"
// @Param request body models.NodeMetricsCallbackRequest true "节点指标数据"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/node-metrics/callback [post]
func (h *SaltStackHandler) NodeMetricsCallback(c *gin.Context) {
	// 可选的 API Token 认证
	expectedToken := os.Getenv("NODE_METRICS_API_TOKEN")
	if expectedToken != "" {
		providedToken := c.GetHeader("X-API-Token")
		if providedToken != expectedToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or missing API token",
			})
			return
		}
	}

	var req models.NodeMetricsCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	if req.MinionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "minion_id is required",
		})
		return
	}

	// 解析时间戳
	var timestamp time.Time
	if req.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			timestamp = time.Now()
		} else {
			timestamp = parsed
		}
	} else {
		timestamp = time.Now()
	}

	// 构建 NodeMetrics 记录
	metrics := &models.NodeMetrics{
		MinionID:  req.MinionID,
		Timestamp: timestamp,
	}

	// 处理 CPU 信息
	if req.CPU != nil {
		metrics.CPUCores = req.CPU.Cores
		metrics.CPUModel = req.CPU.Model
		metrics.CPUUsagePercent = req.CPU.UsagePercent
		metrics.CPULoadAvg = req.CPU.LoadAvg
	}

	// 处理内存信息
	if req.Memory != nil {
		metrics.MemoryTotalGB = req.Memory.TotalGB
		metrics.MemoryUsedGB = req.Memory.UsedGB
		metrics.MemoryAvailableGB = req.Memory.AvailableGB
		metrics.MemoryUsagePercent = req.Memory.UsagePercent
	}

	// 处理网络信息
	if req.Network != nil {
		metrics.ActiveConnections = req.Network.ActiveConnections
		if req.Network.Interfaces != nil {
			networkJSON, _ := json.Marshal(req.Network.Interfaces)
			metrics.NetworkInfo = string(networkJSON)
		}
	}

	// 处理 GPU 信息
	if req.GPU != nil {
		metrics.GPUDriverVersion = req.GPU.DriverVersion
		metrics.CUDAVersion = req.GPU.CUDAVersion
		metrics.GPUCount = req.GPU.Count
		metrics.GPUModel = req.GPU.Model
		metrics.GPUMemoryTotal = req.GPU.MemoryTotal
		metrics.GPUAvgUtilization = req.GPU.AvgUtilization
		metrics.GPUMemoryUsedMB = req.GPU.MemoryUsedMB
		metrics.GPUMemoryTotalMB = req.GPU.MemoryTotalMB
		metrics.GPUMemoryUsagePercent = req.GPU.MemoryUsagePercent
		if req.GPU.GPUs != nil {
			gpuInfoJSON, _ := json.Marshal(req.GPU.GPUs)
			metrics.GPUInfo = string(gpuInfoJSON)
		}
	}

	// 处理 IB 信息
	if req.IB != nil {
		metrics.IBActiveCount = req.IB.ActiveCount
		metrics.IBDownCount = req.IB.DownCount
		metrics.IBTotalCount = req.IB.TotalCount
		if req.IB.Ports != nil {
			ibPortsJSON, _ := json.Marshal(req.IB.Ports)
			metrics.IBPortsInfo = string(ibPortsJSON)
		}
	}

	// 处理 RoCE 信息
	if req.RoCE != nil {
		roceJSON, _ := json.Marshal(req.RoCE)
		metrics.RoCEInfo = string(roceJSON)
	}

	// 处理系统信息
	if req.System != nil {
		metrics.KernelVersion = req.System.KernelVersion
		metrics.OSVersion = req.System.OSVersion
		metrics.UptimeSeconds = req.System.UptimeSeconds
	}

	// 保存原始数据
	rawDataJSON, _ := json.Marshal(req)
	metrics.RawData = string(rawDataJSON)

	// 调用服务保存指标
	metricsService := services.NewNodeMetricsService()
	if err := metricsService.SaveNodeMetrics(metrics); err != nil {
		log.Printf("[NodeMetricsCallback] Failed to save metrics for minion %s: %v", req.MinionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save metrics: " + err.Error(),
		})
		return
	}

	log.Printf("[NodeMetricsCallback] Received metrics from minion: %s, CPU: %.1f%%, Mem: %.1f%%, GPU count: %d (util: %.1f%%), IB: %d active, %d down",
		req.MinionID, metrics.CPUUsagePercent, metrics.MemoryUsagePercent,
		metrics.GPUCount, metrics.GPUAvgUtilization,
		metrics.IBActiveCount, metrics.IBDownCount)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Metrics received and saved",
		"minion":  req.MinionID,
	})
}

// GetNodeMetrics 获取节点指标
// @Summary 获取节点指标
// @Description 获取指定节点的最新硬件指标（GPU/IB）
// @Tags SaltStack
// @Produce json
// @Param minion_id query string false "Minion ID，留空获取所有节点"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/node-metrics [get]
func (h *SaltStackHandler) GetNodeMetrics(c *gin.Context) {
	minionID := c.Query("minion_id")

	metricsService := services.NewNodeMetricsService()

	if minionID != "" {
		// 获取单个节点的最新指标
		metrics, err := metricsService.GetLatestMetrics(minionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Metrics not found for minion: " + minionID,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    h.formatNodeMetricsResponse(metrics),
		})
		return
	}

	// 获取所有节点的最新指标
	allMetrics, err := metricsService.GetAllLatestMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get metrics: " + err.Error(),
		})
		return
	}

	// 格式化响应
	var results []models.NodeMetricsResponse
	for _, m := range allMetrics {
		results = append(results, h.formatNodeMetricsResponse(&m))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"total":   len(results),
	})
}

// formatNodeMetricsResponse 格式化节点指标响应
func (h *SaltStackHandler) formatNodeMetricsResponse(m *models.NodeMetricsLatest) models.NodeMetricsResponse {
	resp := models.NodeMetricsResponse{
		MinionID:    m.MinionID,
		CollectedAt: m.Timestamp,
	}

	// CPU 信息 - 始终返回，即使值为 0
	resp.CPU = &models.NodeCPUMetrics{
		Cores:        m.CPUCores,
		Model:        m.CPUModel,
		UsagePercent: m.CPUUsagePercent,
		Usage:        m.CPUUsagePercent, // 兼容前端期望的 usage 字段
		LoadAvg:      m.CPULoadAvg,
	}

	// 内存信息 - 始终返回，即使值为 0
	resp.Memory = &models.NodeMemoryMetrics{
		TotalGB:      m.MemoryTotalGB,
		UsedGB:       m.MemoryUsedGB,
		AvailableGB:  m.MemoryAvailableGB,
		UsagePercent: m.MemoryUsagePercent,
	}

	// 网络信息 - 始终返回
	resp.Network = &models.NodeNetworkMetrics{
		ActiveConnections: m.ActiveConnections,
	}
	if m.NetworkInfo != "" {
		var interfaces []models.NodeNetworkInterface
		if err := json.Unmarshal([]byte(m.NetworkInfo), &interfaces); err == nil {
			resp.Network.Interfaces = interfaces
		}
	}

	// GPU 信息 - 始终返回，即使没有 GPU
	resp.GPU = &models.NodeGPUMetrics{
		DriverVersion:      m.GPUDriverVersion,
		CUDAVersion:        m.CUDAVersion,
		Count:              m.GPUCount,
		Model:              m.GPUModel,
		MemoryTotal:        m.GPUMemoryTotal,
		AvgUtilization:     m.GPUAvgUtilization,
		MemoryUsedMB:       m.GPUMemoryUsedMB,
		MemoryTotalMB:      m.GPUMemoryTotalMB,
		MemoryUsagePercent: m.GPUMemoryUsagePercent,
	}
	// 解析详细 GPU 信息
	if m.GPUInfo != "" {
		var gpus []models.NodeGPUDetailInfo
		if err := json.Unmarshal([]byte(m.GPUInfo), &gpus); err == nil {
			resp.GPU.GPUs = gpus
		}
	}

	// IB 信息 - 始终返回，即使没有 IB 设备
	resp.IB = &models.NodeIBMetrics{
		ActiveCount: m.IBActiveCount,
		DownCount:   m.IBDownCount,
		TotalCount:  m.IBTotalCount,
	}
	// 解析 IB 端口信息
	if m.IBPortsInfo != "" {
		var ports []models.NodeIBPortInfo
		if err := json.Unmarshal([]byte(m.IBPortsInfo), &ports); err == nil {
			resp.IB.Ports = ports
		}
	}

	// RoCE 信息
	if m.RoCEInfo != "" {
		var roce models.NodeRoCEMetrics
		if err := json.Unmarshal([]byte(m.RoCEInfo), &roce); err == nil {
			resp.RoCE = &roce
		}
	}

	// 系统信息
	if m.KernelVersion != "" || m.OSVersion != "" {
		resp.System = &models.NodeSystemMetrics{
			KernelVersion: m.KernelVersion,
			OSVersion:     m.OSVersion,
			UptimeSeconds: m.UptimeSeconds,
		}
	}

	return resp
}

// DeployNodeMetricsState 部署节点指标采集 State
// @Summary 部署节点指标采集
// @Description 向指定 Minion 部署节点指标采集脚本和定时任务
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "部署请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/node-metrics/deploy [post]
func (h *SaltStackHandler) DeployNodeMetricsState(c *gin.Context) {
	var req struct {
		Target   string `json:"target" binding:"required"` // Minion ID 或通配符
		Interval int    `json:"interval"`                  // 采集间隔（分钟），默认 3
		APIToken string `json:"api_token"`                 // 可选的 API Token
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Interval <= 0 {
		req.Interval = 3
	}

	// 获取回调 URL
	callbackURL := h.getNodeMetricsCallbackURL(c)

	// 获取 API Token（优先使用请求中的，否则使用环境变量）
	apiToken := req.APIToken
	if apiToken == "" {
		apiToken = os.Getenv("NODE_METRICS_API_TOKEN")
	}

	// 构建 pillar 数据
	pillarData := map[string]interface{}{
		"node_metrics": map[string]interface{}{
			"callback_url":     callbackURL,
			"collect_interval": req.Interval,
			"api_token":        apiToken,
		},
	}

	// 获取 Salt API 客户端
	client := h.newSaltAPIClient()
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get Salt API client",
		})
		return
	}

	// 执行 state.apply
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    req.Target,
		"fun":    "state.apply",
		"arg":    []string{"node-metrics"},
		"pillar": pillarData,
	}

	result, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		log.Printf("[DeployNodeMetricsState] Salt API error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Salt API error: " + err.Error(),
		})
		return
	}

	log.Printf("[DeployNodeMetricsState] Deployed node-metrics state to target: %s", req.Target)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Node metrics state deployed",
		"target":      req.Target,
		"interval":    req.Interval,
		"callbackURL": callbackURL,
		"result":      result,
	})
}

// getNodeMetricsCallbackURL 获取节点指标回调 URL
func (h *SaltStackHandler) getNodeMetricsCallbackURL(c *gin.Context) string {
	// 优先从环境变量获取
	if callbackURL := os.Getenv("NODE_METRICS_CALLBACK_URL"); callbackURL != "" {
		return callbackURL
	}

	// 从配置获取后端地址
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	// 尝试获取真实主机地址
	host := c.Request.Host
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	return fmt.Sprintf("%s://%s/api/saltstack/node-metrics/callback", scheme, host)
}

// TriggerMetricsCollection 触发节点指标采集
// @Summary 触发节点指标采集
// @Description 通过 Salt API 触发指定节点立即执行指标采集脚本
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "触发请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/node-metrics/trigger [post]
func (h *SaltStackHandler) TriggerMetricsCollection(c *gin.Context) {
	var req struct {
		Target string `json:"target"` // Minion ID 或通配符，默认 "*"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体，使用默认值
		req.Target = "*"
	}

	if req.Target == "" {
		req.Target = "*"
	}

	// 获取 Salt API 客户端
	client := h.newSaltAPIClient()
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get Salt API client",
		})
		return
	}

	// 执行采集脚本
	collectCmd := "/opt/ai-infra/scripts/collect-node-metrics.sh"
	payload := map[string]interface{}{
		"client": "local",
		"tgt":    req.Target,
		"fun":    "cmd.run",
		"arg":    []interface{}{collectCmd},
		"kwarg": map[string]interface{}{
			"timeout":      30,
			"shell":        "/bin/bash",
			"python_shell": true,
		},
	}

	result, err := client.makeRequest("/", "POST", payload)
	if err != nil {
		log.Printf("[TriggerMetricsCollection] Salt API error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Salt API error: " + err.Error(),
		})
		return
	}

	log.Printf("[TriggerMetricsCollection] Triggered metrics collection for target: %s", req.Target)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Metrics collection triggered",
		"target":  req.Target,
		"result":  result,
	})
}

// GetNodeMetricsSummary 获取节点指标汇总统计
// @Summary 获取节点指标汇总统计
// @Description 获取所有节点的 GPU/IB 等硬件指标汇总
// @Tags SaltStack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/node-metrics/summary [get]
func (h *SaltStackHandler) GetNodeMetricsSummary(c *gin.Context) {
	metricsService := services.NewNodeMetricsService()
	summary, err := metricsService.GetMetricsSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get metrics summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetIBPortIgnores 获取指定 minion 的 IB 端口忽略列表
// @Summary 获取 IB 端口忽略列表
// @Description 获取指定 minion 已忽略的 IB 端口列表
// @Tags SaltStack
// @Produce json
// @Param minion_id query string false "Minion ID（不指定则返回所有）"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/ib-ignores [get]
func (h *SaltStackHandler) GetIBPortIgnores(c *gin.Context) {
	minionID := c.Query("minion_id")

	metricsService := services.NewNodeMetricsService()
	ignores, err := metricsService.GetIBPortIgnores(minionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get IB port ignores: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    ignores,
	})
}

// AddIBPortIgnore 添加 IB 端口忽略
// @Summary 添加 IB 端口忽略
// @Description 将指定 minion 的 IB 端口加入忽略列表（因物理未接线等原因）
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "忽略请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/ib-ignores [post]
func (h *SaltStackHandler) AddIBPortIgnore(c *gin.Context) {
	var req struct {
		MinionID string `json:"minion_id" binding:"required"`
		PortName string `json:"port_name" binding:"required"`
		PortNum  int    `json:"port_num"`
		Reason   string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// 默认端口号为 1
	if req.PortNum <= 0 {
		req.PortNum = 1
	}

	// 获取当前用户
	createdBy := "system"
	if user, exists := c.Get("username"); exists {
		createdBy = user.(string)
	}

	metricsService := services.NewNodeMetricsService()
	if err := metricsService.AddIBPortIgnore(req.MinionID, req.PortName, req.PortNum, req.Reason, createdBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add IB port ignore: " + err.Error(),
		})
		return
	}

	log.Printf("[AddIBPortIgnore] Added ignore for minion=%s port=%s portNum=%d by=%s reason=%s",
		req.MinionID, req.PortName, req.PortNum, createdBy, req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "IB port ignore added",
		"minion_id": req.MinionID,
		"port_name": req.PortName,
		"port_num":  req.PortNum,
		"reason":    req.Reason,
	})
}

// RemoveIBPortIgnore 移除 IB 端口忽略
// @Summary 移除 IB 端口忽略
// @Description 将指定 minion 的 IB 端口从忽略列表中移除
// @Tags SaltStack
// @Produce json
// @Param minion_id path string true "Minion ID"
// @Param port_name path string true "Port Name"
// @Param port_num query int false "Port Number (不指定则删除所有匹配的端口)"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/ib-ignores/{minion_id}/{port_name} [delete]
func (h *SaltStackHandler) RemoveIBPortIgnore(c *gin.Context) {
	minionID := c.Param("minion_id")
	portName := c.Param("port_name")

	if minionID == "" || portName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "minion_id and port_name are required",
		})
		return
	}

	// 从查询参数获取 port_num，默认为 0（删除所有匹配的端口号）
	portNum := 0
	if portNumStr := c.Query("port_num"); portNumStr != "" {
		if pn, err := strconv.Atoi(portNumStr); err == nil {
			portNum = pn
		}
	}

	metricsService := services.NewNodeMetricsService()
	if err := metricsService.RemoveIBPortIgnore(minionID, portName, portNum); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to remove IB port ignore: " + err.Error(),
		})
		return
	}

	log.Printf("[RemoveIBPortIgnore] Removed ignore for minion=%s port=%s portNum=%d", minionID, portName, portNum)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "IB port ignore removed",
	})
}

// GetIBPortAlerts 获取 IB 端口告警列表
// @Summary 获取 IB 端口告警列表
// @Description 获取所有 Down 状态但未被忽略的 IB 端口告警
// @Tags SaltStack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/ib-alerts [get]
func (h *SaltStackHandler) GetIBPortAlerts(c *gin.Context) {
	metricsService := services.NewNodeMetricsService()
	alerts, err := metricsService.GetIBPortAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get IB port alerts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    alerts,
		"total":   len(alerts),
	})
}

// GetSaltJobConfig 获取作业配置
// @Summary 获取作业配置
// @Description 获取 Salt 作业的保存配置（最大保留天数、最大记录数等）
// @Tags SaltStack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/jobs/config [get]
func (h *SaltStackHandler) GetSaltJobConfig(c *gin.Context) {
	saltJobService := services.GetSaltJobService()
	if saltJobService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "SaltJobService not initialized",
		})
		return
	}

	config := saltJobService.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// UpdateSaltJobConfig 更新作业配置
// @Summary 更新作业配置
// @Description 更新 Salt 作业的保存配置（最大保留天数、最大记录数等）
// @Tags SaltStack
// @Accept json
// @Produce json
// @Param config body models.SaltJobConfig true "配置信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/jobs/config [put]
func (h *SaltStackHandler) UpdateSaltJobConfig(c *gin.Context) {
	saltJobService := services.GetSaltJobService()
	if saltJobService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "SaltJobService not initialized",
		})
		return
	}

	var config models.SaltJobConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 参数校验
	if config.MaxRetentionDays < 1 {
		config.MaxRetentionDays = 1
	} else if config.MaxRetentionDays > 365 {
		config.MaxRetentionDays = 365
	}
	if config.MaxRecords < 100 {
		config.MaxRecords = 100
	} else if config.MaxRecords > 100000 {
		config.MaxRecords = 100000
	}
	if config.CleanupIntervalHour < 1 {
		config.CleanupIntervalHour = 1
	} else if config.CleanupIntervalHour > 168 {
		config.CleanupIntervalHour = 168
	}

	// 保留现有配置的 ID
	existingConfig := saltJobService.GetConfig()
	if existingConfig != nil {
		config.ID = existingConfig.ID
		config.LastCleanupTime = existingConfig.LastCleanupTime
	}

	if err := saltJobService.UpdateConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated",
		"data":    config,
	})
}

// GetSaltJobHistory 获取作业历史列表（从数据库）
// @Summary 获取作业历史列表
// @Description 从数据库获取分页的 Salt 作业历史列表
// @Tags SaltStack
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param task_id query string false "任务ID过滤"
// @Param jid query string false "JID过滤"
// @Param function query string false "函数名过滤"
// @Param target query string false "目标过滤"
// @Param status query string false "状态过滤"
// @Param user query string false "用户过滤"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/jobs/history [get]
func (h *SaltStackHandler) GetSaltJobHistory(c *gin.Context) {
	saltJobService := services.GetSaltJobService()
	if saltJobService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "SaltJobService not initialized",
		})
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort_by", "start_time")
	sortDesc := c.DefaultQuery("sort_desc", "true") == "true"

	params := &models.SaltJobQueryParams{
		TaskID:   c.Query("task_id"),
		JID:      c.Query("jid"),
		Function: c.Query("function"),
		Target:   c.Query("target"),
		Status:   c.Query("status"),
		User:     c.Query("user"),
		Page:     page,
		PageSize: pageSize,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	result, err := saltJobService.ListJobs(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"total":   result.Total,
		"page":    result.Page,
		"size":    result.Size,
	})
}

// GetSaltJobByTaskID 通过 TaskID 获取作业详情
// @Summary 通过 TaskID 获取作业
// @Description 通过前端传递的 TaskID 获取作业详情
// @Tags SaltStack
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/jobs/by-task/{task_id} [get]
func (h *SaltStackHandler) GetSaltJobByTaskID(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "task_id is required",
		})
		return
	}

	saltJobService := services.GetSaltJobService()
	if saltJobService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "SaltJobService not initialized",
		})
		return
	}

	job, err := saltJobService.GetJobByTaskID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Job not found: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

// TriggerJobCleanup 手动触发作业清理
// @Summary 手动触发作业清理
// @Description 立即执行一次作业历史清理
// @Tags SaltStack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/saltstack/jobs/cleanup [post]
func (h *SaltStackHandler) TriggerJobCleanup(c *gin.Context) {
	saltJobService := services.GetSaltJobService()
	if saltJobService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "SaltJobService not initialized",
		})
		return
	}

	// 在后台执行清理
	go func() {
		saltJobService.CheckAndUpdateStaleJobs(context.Background(), 30)
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cleanup triggered",
	})
}
