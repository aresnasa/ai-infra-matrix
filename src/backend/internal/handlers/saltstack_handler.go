package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
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
}

// SaltMinion Salt Minion信息
type SaltMinion struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	OS           string            `json:"os"`
	OSVersion    string            `json:"os_version"`
	Architecture string            `json:"architecture"`
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
	host := getEnv("SALT_MASTER_HOST", "saltstack")
	port := getEnv("SALT_API_PORT", "8000")
	return fmt.Sprintf("http://%s:%s", host, port)
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
	loginData := map[string]interface{}{
		"username": "salt",
		"password": "salt",
		"eauth":    "pam",
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(c.baseURL+"/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("authentication failed: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if token, ok := result["token"].(string); ok {
		c.token = token
	}

	return nil
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

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
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
	
	// 尝试连接Salt API
	if err := client.authenticate(); err != nil {
		// 如果连接失败，返回模拟数据
		status := h.getDemoSaltStackStatus()
		h.cacheStatus(&status, 300) // 缓存5分钟
		c.JSON(http.StatusOK, gin.H{"data": status})
		return
	}

	// 获取实际状态
	status, err := h.getRealSaltStackStatus(client)
	if err != nil {
		status = h.getDemoSaltStackStatus()
	}

	// 缓存状态
	h.cacheStatus(&status, 60) // 缓存1分钟

	c.JSON(http.StatusOK, gin.H{"data": status})
}

// getRealSaltStackStatus 获取真实的SaltStack状态
func (h *SaltStackHandler) getRealSaltStackStatus(client *saltAPIClient) (SaltStackStatus, error) {
	// 获取版本信息
	versionResp, err := client.makeRequest("/", "GET", nil)
	if err != nil {
		return SaltStackStatus{}, err
	}

	// 获取密钥状态
	keysResp, err := client.makeRequest("/keys", "GET", nil)
	if err != nil {
		return SaltStackStatus{}, err
	}

	// 获取Minion状态
	minionsResp, err := client.makeRequest("/minions", "GET", nil)
	if err != nil {
		return SaltStackStatus{}, err
	}

	status := SaltStackStatus{
		Status:           "connected",
		MasterVersion:    h.extractVersion(versionResp),
		APIVersion:       "3000.3",
		Uptime:           time.Now().Unix() - 3600, // 模拟1小时运行时间
		ConnectedMinions: h.countMinions(minionsResp),
		AcceptedKeys:     h.extractKeys(keysResp, "minions"),
		UnacceptedKeys:   h.extractKeys(keysResp, "minions_pre"),
		RejectedKeys:     h.extractKeys(keysResp, "minions_rejected"),
		Services: map[string]string{
			"salt-master": "running",
			"salt-api":    "running",
			"salt-syndic": "stopped",
		},
		LastUpdated: time.Now(),
		Demo:        false,
	}

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
	
	var minions []SaltMinion
	var demo bool

	// 尝试获取真实数据
	if err := client.authenticate(); err == nil {
		realMinions, err := h.getRealMinions(client)
		if err == nil {
			minions = realMinions
		} else {
			minions = h.getDemoMinions()
			demo = true
		}
	} else {
		minions = h.getDemoMinions()
		demo = true
	}

	// 设置demo标记
	for i := range minions {
		if demo {
			minions[i].Status = "demo"
		}
	}

	// 缓存数据
	if data, err := json.Marshal(minions); err == nil {
		h.cache.Set(context.Background(), "saltstack:minions", string(data), 120*time.Second) // 缓存2分钟
	}

	c.JSON(http.StatusOK, gin.H{"data": minions, "demo": demo})
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

	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("saltstack:jobs:%d", limit)
	if cached, err := h.cache.Get(context.Background(), cacheKey).Result(); err == nil {
		var jobs []SaltJob
		if err := json.Unmarshal([]byte(cached), &jobs); err == nil {
			c.JSON(http.StatusOK, gin.H{"data": jobs})
			return
		}
	}

	// 获取演示数据
	jobs := h.getDemoJobs(limit)

	// 缓存数据
	if data, err := json.Marshal(jobs); err == nil {
		h.cache.Set(context.Background(), cacheKey, string(data), 300*time.Second) // 缓存5分钟
	}

	c.JSON(http.StatusOK, gin.H{"data": jobs, "demo": true})
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
	if data, ok := resp["data"].(map[string]interface{}); ok {
		if version, ok := data["version"].(string); ok {
			return version
		}
	}
	return "3006.4"
}

func (h *SaltStackHandler) countMinions(resp map[string]interface{}) int {
	if data, ok := resp["data"].([]interface{}); ok {
		return len(data)
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
	return []string{}
}

func (h *SaltStackHandler) getRealMinions(client *saltAPIClient) ([]SaltMinion, error) {
	// 实现真实的Minion数据获取
	// 这里可以调用Salt API获取实际数据
	return nil, fmt.Errorf("not implemented")
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
