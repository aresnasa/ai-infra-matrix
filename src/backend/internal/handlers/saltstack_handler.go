package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net"
	"net/url"
	"os"
	"strings"
	"strconv"
	"time"
	"sort"
	"errors"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SaltStackHandler 处理SaltStack相关的API请求
type SaltStackHandler struct {
	config *config.Config
	cache  *redis.Client
}

// NewSaltStackHandler 创建新的SaltStack处理器
func NewSaltStackHandler(cfg *config.Config, cache *redis.Client) *SaltStackHandler {
	return &SaltStackHandler{
		config: cfg,
		cache:  cache,
	}
}

// SaltStackStatus SaltStack状态信息
type SaltStackStatus struct {
	Status        string            `json:"status"`
	MasterVersion string            `json:"master_version"`
	APIVersion    string            `json:"api_version"`
	Uptime        int64             `json:"uptime"`
	ConnectedMinions int           `json:"connected_minions"`
	AcceptedKeys  []string          `json:"accepted_keys"`
	UnacceptedKeys []string         `json:"unaccepted_keys"`
	RejectedKeys  []string          `json:"rejected_keys"`
	Services      map[string]string `json:"services"`
	LastUpdated   time.Time         `json:"last_updated"`
	Demo          bool              `json:"demo,omitempty"`
	// 前端兼容字段
	MasterStatus     string `json:"master_status,omitempty"`
	APIStatus        string `json:"api_status,omitempty"`
	MinionsUp        int    `json:"minions_up,omitempty"`
	MinionsDown      int    `json:"minions_down,omitempty"`
	SaltVersion      string `json:"salt_version,omitempty"`
	ConfigFile       string `json:"config_file,omitempty"`
	LogLevel         string `json:"log_level,omitempty"`
	CPUUsage         int    `json:"cpu_usage,omitempty"`
	MemoryUsage      int    `json:"memory_usage,omitempty"`
	ActiveConnections int   `json:"active_connections,omitempty"`
}

// SaltMinion Salt Minion信息
type SaltMinion struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	OS           string            `json:"os"`
	OSVersion    string            `json:"os_version"`
	Architecture string            `json:"architecture"`
	Arch         string            `json:"arch,omitempty"`
	SaltVersion  string            `json:"salt_version,omitempty"`
	LastSeen     time.Time         `json:"last_seen"`
	Grains       map[string]interface{} `json:"grains"`
	Pillar       map[string]interface{} `json:"pillar,omitempty"`
}

// SaltJob Salt作业信息
type SaltJob struct {
	JID         string                 `json:"jid"`
	Function    string                 `json:"function"`
	Arguments   []string               `json:"arguments"`
	Target      string                 `json:"target"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     *time.Time            `json:"end_time,omitempty"`
	Status      string                `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	User        string                `json:"user"`
}

// saltAPIClient SaltStack API客户端
type saltAPIClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// newSaltAPIClient 创建Salt API客户端
func (h *SaltStackHandler) newSaltAPIClient() *saltAPIClient {
	return &saltAPIClient{
		baseURL: h.getSaltAPIURL(),
		client: &http.Client{
			Timeout: 10 * time.Second, // 设置较短超时以避免 SSH minions 连接超时阻塞整个请求
		},
	}
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

// authenticate 向Salt API认证 - 临时绕过认证
func (c *saltAPIClient) authenticate() error {
	username := getEnv("SALT_API_USERNAME", "saltapi")
	password := os.Getenv("SALT_API_PASSWORD")
	eauth := getEnv("SALT_API_EAUTH", "file")

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
	// 尝试从缓存获取
	if cached, err := h.cache.Get(context.Background(), "saltstack:status").Result(); err == nil {
		var status SaltStackStatus
		if err := json.Unmarshal([]byte(cached), &status); err == nil {
			c.JSON(http.StatusOK, gin.H{"data": status})
			return
		}
	}

	// 创建API客户端
	client := h.newSaltAPIClient()

	// 连接Salt API（要求真实集群，失败则返回错误，不再返回演示数据）
	if err := client.authenticate(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Salt API 认证失败: %v", err)})
		return
	}

	// 获取实际状态
	status, err := h.getRealSaltStackStatus(client)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("获取Salt状态失败: %v", err)})
		return
	}

	// 缓存状态（1分钟）
	h.cacheStatus(&status, 60)
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// SaltDebugInfo 用于输出调试信息，便于快速定位集成问题
type SaltDebugInfo struct {
	BaseURL       string                 `json:"base_url"`
	Env           map[string]string      `json:"env"`
	TCPDial       map[string]interface{} `json:"tcp_dial"`
	RootGET       map[string]interface{} `json:"root_get"`
	Login         map[string]interface{} `json:"login"`
	ManageStatus  map[string]interface{} `json:"manage_status"`
	KeyListAll    map[string]interface{} `json:"key_list_all"`
	Timestamp     time.Time              `json:"timestamp"`
}

// DebugSaltConnectivity 输出Salt API的连通性与基本调用结果
func (h *SaltStackHandler) DebugSaltConnectivity(c *gin.Context) {
	client := h.newSaltAPIClient()
	res := &SaltDebugInfo{
		BaseURL:   client.baseURL,
		Env: map[string]string{
			"SALTSTACK_MASTER_URL": strings.TrimSpace(os.Getenv("SALTSTACK_MASTER_URL")),
			"SALT_API_SCHEME":       getEnv("SALT_API_SCHEME", "http"),
			"SALT_MASTER_HOST":      getEnv("SALT_MASTER_HOST", "saltstack"),
			"SALT_API_PORT":         getEnv("SALT_API_PORT", "8002"),
			"SALT_API_USERNAME":     getEnv("SALT_API_USERNAME", "saltapi"),
			// 不回显明文密码，只提示是否设置
			"SALT_API_PASSWORD_SET": func() string { if os.Getenv("SALT_API_PASSWORD") != "" { return "true" }; return "false" }(),
			"SALT_API_EAUTH":        getEnv("SALT_API_EAUTH", "file"),
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
	{   start := time.Now()
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
	{   start := time.Now()
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
	{   start := time.Now()
		r, err := client.makeRunner("manage.status", nil)
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
	{   start := time.Now()
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
	// 获取 API 根信息（用于APIVersion）
	apiInfo, _ := client.makeRequest("/", "GET", nil)

	// 获取 up/down 状态
	manageStatus, err := client.makeRunner("manage.status", nil)
	if err != nil {
		return SaltStackStatus{}, err
	}
	up, down := h.parseManageStatus(manageStatus)

	// 获取 keys
	keysResp, err := client.makeWheel("key.list_all", nil)
	if err != nil {
		return SaltStackStatus{}, err
	}
	minions, pre, rejected := h.parseWheelKeys(keysResp)

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
		MasterStatus:  "running",
		APIStatus:     "running",
		MinionsUp:     len(up),
		MinionsDown:   len(down),
		SaltVersion:   h.extractAPISaltVersion(apiInfo),
		ConfigFile:    "/etc/salt/master",
		LogLevel:      "info",
		CPUUsage:      0,
		MemoryUsage:   0,
		ActiveConnections: 0,
	}
	_ = down // 可用于前端显示 down 数量
	return status, nil
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
		MasterStatus:  "running",
		APIStatus:     "running",
		MinionsUp:     2,
		MinionsDown:   1,
		SaltVersion:   "3006.4",
		ConfigFile:    "/etc/salt/master",
		LogLevel:      "info",
		CPUUsage:      12,
		MemoryUsage:   23,
		ActiveConnections: 2,
	}
}

// GetSaltMinions 获取Salt Minion列表
func (h *SaltStackHandler) GetSaltMinions(c *gin.Context) {
	// 尝试从缓存获取
	if cached, err := h.cache.Get(context.Background(), "saltstack:minions").Result(); err == nil {
		var minions []SaltMinion
		if err := json.Unmarshal([]byte(cached), &minions); err == nil {
			c.JSON(http.StatusOK, gin.H{"data": minions})
			return
		}
	}

	// 创建API客户端
	client := h.newSaltAPIClient()

	// 认证
	if err := client.authenticate(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Salt API 认证失败: %v", err)})
		return
	}

	// 获取真实数据
	minions, err := h.getRealMinions(client)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("获取Minions失败: %v", err)})
		return
	}

	// 缓存数据
	if data, err := json.Marshal(minions); err == nil {
		h.cache.Set(context.Background(), "saltstack:minions", string(data), 120*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{"data": minions, "demo": false})
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

	// 创建API客户端并认证
	client := h.newSaltAPIClient()
	if err := client.authenticate(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Salt API 认证失败: %v", err)})
		return
	}

	// 获取真实作业数据
	jobs, err := h.getRealJobs(client, limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("获取Jobs失败: %v", err)})
		return
	}

	// 缓存数据
	cacheKey := fmt.Sprintf("saltstack:jobs:%d", limit)
	if data, err := json.Marshal(jobs); err == nil {
		h.cache.Set(context.Background(), cacheKey, string(data), 300*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{"data": jobs, "demo": false})
}

// getRealJobs 通过Salt NetAPI获取真实作业列表
func (h *SaltStackHandler) getRealJobs(client *saltAPIClient, limit int) ([]SaltJob, error) {
	// 尝试 GET /jobs
	resp, err := client.makeRequest("/jobs", "GET", nil)
	if err != nil {
		return nil, err
	}
	var jobs []SaltJob
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
					if f, ok := info["Function"].(string); ok { j.Function = f }
					if t, ok := info["Target"].(string); ok { j.Target = t }
					if u, ok := info["User"].(string); ok { j.User = u }
					if args, ok := info["Arguments"].([]interface{}); ok {
						for _, a := range args { j.Arguments = append(j.Arguments, fmt.Sprint(a)) }
					}
					// 尝试解析开始时间
					if st, ok := info["StartTime"].(string); ok {
						if ts, err := time.Parse(time.RFC3339, st); err == nil { j.StartTime = ts } else { j.StartTime = time.Now() }
					} else {
						j.StartTime = time.Now()
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

func (h *SaltStackHandler) getRealMinions(client *saltAPIClient) ([]SaltMinion, error) {
	// 使用 runner manage.status 获取 up/down 列表
	statusResp, err := client.makeRunner("manage.status", nil)
	if err != nil {
		return nil, err
	}
	up, down := h.parseManageStatus(statusResp)

	// 将 up 的节点获取详细 grains
	var minions []SaltMinion
	for _, id := range up {
		// GET /minions/{id}
		r, err := client.makeRequest("/minions/"+id, "GET", nil)
		if err != nil {
			// 如果失败，至少添加一个基本项
			minions = append(minions, SaltMinion{ID: id, Status: "up"})
			continue
		}
		grains := h.parseMinionGrains(r, id)
		m := SaltMinion{
			ID:           id,
			Status:       "up",
			OS:           fmt.Sprintf("%v", grains["os"]),
			OSVersion:    fmt.Sprintf("%v", grains["osrelease"]),
			Architecture: fmt.Sprintf("%v", grains["osarch"]),
			Arch:         fmt.Sprintf("%v", grains["osarch"]),
			SaltVersion:  fmt.Sprintf("%v", grains["saltversion"]),
			LastSeen:     time.Now(),
			Grains:       grains,
		}
		minions = append(minions, m)
	}
	// 将 down 的节点也加入列表以便页面展示
	for _, id := range down {
		minions = append(minions, SaltMinion{ID: id, Status: "down"})
	}
	return minions, nil
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
		Target   string      `json:"target" binding:"required"`
		Function string      `json:"function" binding:"required"`
		Args     []string    `json:"args"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 实现命令执行逻辑
	// 这里需要与Salt API交互执行实际命令

	// 返回模拟结果
	result := SaltJob{
		JID:       fmt.Sprintf("%d", time.Now().Unix()),
		Function:  request.Function,
		Arguments: request.Args,
		Target:    request.Target,
		StartTime: time.Now(),
		Status:    "submitted",
		User:      "admin", // 从JWT token获取用户信息
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
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
			"stdin": r.Code,
			"python_shell": true,
		}
		if r.Timeout > 0 {
			kwarg["timeout"] = r.Timeout
		}

		payload := map[string]interface{}{
			"client":  "local",
			"tgt":     r.Target,
			"fun":     "cmd.run",
			"arg":     []interface{}{cmd},
			"kwarg":   kwarg,
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
