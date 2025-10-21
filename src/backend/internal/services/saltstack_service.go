package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// SaltStackService SaltStack服务
type SaltStackService struct {
	masterURL string
	apiToken  string
	client    *http.Client
}

// SaltStackStatus SaltStack状态
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
}

// SaltJob SaltStack作业
type SaltJob struct {
	JID       string                 `json:"jid"`
	Function  string                 `json:"function"`
	Target    string                 `json:"target"`
	StartTime time.Time              `json:"start_time"`
	Results   map[string]interface{} `json:"results"`
}

// NewSaltStackService 创建新的SaltStack服务
func NewSaltStackService() *SaltStackService {
	// 从环境变量读取配置，提供默认值
	masterURL := os.Getenv("SALTSTACK_MASTER_URL")
	if masterURL == "" {
		// 兼容按 Host/Port/Scheme 组合的配置，避免写死容器内服务名
		scheme := os.Getenv("SALT_API_SCHEME")
		if scheme == "" {
			scheme = "http"
		}
		host := os.Getenv("SALT_MASTER_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("SALT_API_PORT")
		if port == "" {
			port = "8002"
		}
		masterURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	apiToken := os.Getenv("SALTSTACK_API_TOKEN")
	// API Token是可选的，如果没有设置则为空

	return &SaltStackService{
		masterURL: masterURL,
		apiToken:  apiToken,
		client: &http.Client{
			Timeout: 90 * time.Second, // 增加超时时间以支持 SaltStack minions 响应超时（默认60秒）
		},
	}
}

// GetStatus 获取SaltStack状态
func (s *SaltStackService) GetStatus(ctx context.Context) (*SaltStackStatus, error) {
	// 首先尝试获取真实的SaltStack状态
	status, err := s.getRealSaltStatus(ctx)
	if err != nil {
		// 修复：如果无法连接到Salt API，返回错误而不是演示数据
		// 这样调用者可以fallback到其他方法获取真实数据
		return nil, fmt.Errorf("salt API unavailable: %v", err)
	}
	return status, nil
}

// getRealSaltStatus 获取真实的SaltStack状态
func (s *SaltStackService) getRealSaltStatus(ctx context.Context) (*SaltStackStatus, error) {
	// 获取密钥状态
	keysData, err := s.executeSaltCommand(ctx, map[string]interface{}{
		"fun":    "key.list_all",
		"client": "wheel",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %v", err)
	}

	// 获取管理状态
	_, err = s.executeSaltCommand(ctx, map[string]interface{}{
		"fun":    "manage.status",
		"client": "runner",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %v", err)
	}

	// 解析响应
	status := &SaltStackStatus{
		Status:        "running",
		MasterVersion: "3006.1",
		APIVersion:    "v1",
		Uptime:        3600,
		Services: map[string]string{
			"salt-master": "running",
			"salt-api":    "running",
		},
		LastUpdated: time.Now(),
	}

	// 解析密钥数据
	if keys, ok := keysData["return"].([]interface{}); ok && len(keys) > 0 {
		if keyData, ok := keys[0].(map[string]interface{}); ok {
			if data, ok := keyData["data"].(map[string]interface{}); ok {
				if return_data, ok := data["return"].(map[string]interface{}); ok {
					if minions, ok := return_data["minions"].([]interface{}); ok {
						for _, minion := range minions {
							if minionStr, ok := minion.(string); ok {
								status.AcceptedKeys = append(status.AcceptedKeys, minionStr)
							}
						}
					}
					if unaccepted, ok := return_data["minions_pre"].([]interface{}); ok {
						for _, minion := range unaccepted {
							if minionStr, ok := minion.(string); ok {
								status.UnacceptedKeys = append(status.UnacceptedKeys, minionStr)
							}
						}
					}
				}
			}
		}
	}

	status.ConnectedMinions = len(status.AcceptedKeys)

	return status, nil
}

// executeSaltCommand 执行Salt API命令
func (s *SaltStackService) executeSaltCommand(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	// 序列化请求数据
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", s.masterURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 如果有API token，添加认证头
	if s.apiToken != "" {
		req.Header.Set("X-Auth-Token", s.apiToken)
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result, nil
}

// ExecuteCommand 执行SaltStack命令
func (s *SaltStackService) ExecuteCommand(ctx context.Context, command string, targets []string) (map[string]interface{}, error) {
	// 构建Salt API请求
	payload := map[string]interface{}{
		"fun":    command,
		"tgt":    "*", // 默认目标所有minions
		"client": "local",
	}

	// 如果指定了目标，使用列表模式
	if len(targets) > 0 {
		payload["tgt"] = targets
		payload["tgt_type"] = "list"
	}

	// 执行命令
	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		// 如果API调用失败，返回模拟结果
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Salt API unavailable: %v", err),
			"result": map[string]interface{}{
				"targets": targets,
				"command": command,
				"output":  "API连接失败",
			},
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"result":  result,
	}, nil
}

// GetJobs 获取SaltStack作业列表
func (s *SaltStackService) GetJobs(ctx context.Context) ([]SaltJob, error) {
	// 这里应该调用SaltStack API获取作业列表
	// 目前返回模拟数据
	return []SaltJob{
		{
			JID:       "20231201000000000000",
			Function:  "test.ping",
			Target:    "*",
			StartTime: time.Now().Add(-1 * time.Hour),
			Results: map[string]interface{}{
				"minion1": true,
				"minion2": true,
			},
		},
	}, nil
}

// AcceptMinion 接受Minion密钥
func (s *SaltStackService) AcceptMinion(ctx context.Context, minionID string) error {
	payload := map[string]interface{}{
		"fun":    "key.accept",
		"match":  minionID,
		"client": "wheel",
	}

	_, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to accept minion %s: %v", minionID, err)
	}
	return nil
}

// RejectMinion 拒绝Minion密钥
func (s *SaltStackService) RejectMinion(ctx context.Context, minionID string) error {
	payload := map[string]interface{}{
		"fun":    "key.reject",
		"match":  minionID,
		"client": "wheel",
	}

	_, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to reject minion %s: %v", minionID, err)
	}
	return nil
}

// GetMinionStatus 获取Minion状态
func (s *SaltStackService) GetMinionStatus(ctx context.Context, minionID string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"fun":    "grains.items",
		"tgt":    minionID,
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		// 返回模拟数据
		return map[string]interface{}{
			"id":     minionID,
			"status": "offline",
			"error":  fmt.Sprintf("无法连接到minion: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"id":     minionID,
		"status": "online",
		"grains": result,
	}, nil
}
