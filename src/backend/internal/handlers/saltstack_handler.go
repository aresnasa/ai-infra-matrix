package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"strconv"
	"time"
	"sort"

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
			Timeout: 30 * time.Second,
		},
	}
}

// getSaltAPIURL 获取Salt API URL
func (h *SaltStackHandler) getSaltAPIURL() string {
	// 优先使用完整URL
	if base := strings.TrimSpace(os.Getenv("SALTSTACK_MASTER_URL")); base != "" {
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
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("salt api login failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
	return c.makeRequest("/", "POST", payload)
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
	return c.makeRequest("/", "POST", payload)
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
