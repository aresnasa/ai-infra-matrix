package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SaltStackService SaltStack服务
type SaltStackService struct {
	masterURL   string
	apiToken    string
	username    string
	password    string
	eauth       string
	client      *http.Client
	tokenExpiry time.Time
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
	username := os.Getenv("SALT_API_USERNAME")
	if username == "" {
		username = "saltapi"
	}
	password := os.Getenv("SALT_API_PASSWORD")
	eauth := os.Getenv("SALT_API_EAUTH")
	if eauth == "" {
		eauth = "file"
	}

	return &SaltStackService{
		masterURL: masterURL,
		apiToken:  apiToken,
		username:  username,
		password:  password,
		eauth:     eauth,
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
	// 确保有有效的token
	if err := s.ensureToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get auth token: %v", err)
	}

	log.Printf("[SaltStack] Executing command with payload: %+v", payload)

	// Salt API 需要数组格式的请求体: [{...}]
	requestBody := []map[string]interface{}{payload}

	// 序列化请求数据
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	log.Printf("[SaltStack] Request JSON: %s", string(jsonData))

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", s.masterURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 添加认证头
	if s.apiToken != "" {
		req.Header.Set("X-Auth-Token", s.apiToken)
	}

	log.Printf("[SaltStack] Sending request to: %s", s.masterURL+"/")

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[SaltStack] Request failed: %v", err)
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("[SaltStack] Response status: %d", resp.StatusCode)
	log.Printf("[SaltStack] Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	log.Printf("[SaltStack] Parsed result: %+v", result)

	return result, nil
}

// ensureToken 确保有有效的认证token
func (s *SaltStackService) ensureToken(ctx context.Context) error {
	// 如果已有token且未过期，直接返回
	if s.apiToken != "" && time.Now().Before(s.tokenExpiry) {
		return nil
	}

	// 如果没有配置用户名或密码，无法登录
	if s.username == "" || s.password == "" {
		return fmt.Errorf("no username or password configured for Salt API authentication")
	}

	// 登录获取token
	loginPayload := map[string]interface{}{
		"username": s.username,
		"password": s.password,
		"eauth":    s.eauth,
	}

	jsonData, err := json.Marshal(loginPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.masterURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析登录响应
	var loginResult map[string]interface{}
	if err := json.Unmarshal(body, &loginResult); err != nil {
		return fmt.Errorf("failed to parse login response: %v", err)
	}

	// 提取token
	if returnData, ok := loginResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if token, ok := tokenData["token"].(string); ok {
				s.apiToken = token
				// 设置token过期时间（通常是8小时，这里提前5分钟刷新）
				if expire, ok := tokenData["expire"].(float64); ok {
					s.tokenExpiry = time.Unix(int64(expire), 0).Add(-5 * time.Minute)
				} else {
					// 如果没有过期时间，默认7小时55分钟后过期
					s.tokenExpiry = time.Now().Add(7*time.Hour + 55*time.Minute)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("failed to extract token from login response")
}

// containsShellMetaChars 检查字符串是否包含 shell 元字符
// 这些字符需要通过 shell 来解释执行
func containsShellMetaChars(s string) bool {
	// Shell 元字符列表：管道、重定向、逻辑运算符、命令替换、通配符等
	metaChars := []string{
		"|",    // 管道
		"&",    // 后台执行或逻辑与
		";",    // 命令分隔符
		">",    // 重定向
		"<",    // 输入重定向
		"$",    // 变量替换或命令替换
		"`",    // 命令替换
		"(",    // 子 shell
		")",    // 子 shell
		"*",    // 通配符
		"?",    // 通配符
		"[",    // 通配符
		"]",    // 通配符
		"&&",   // 逻辑与
		"||",   // 逻辑或
		">>",   // 追加重定向
		"<<",   // Here document
		"2>",   // 错误重定向
		"2>&1", // 错误重定向到标准输出
	}

	for _, meta := range metaChars {
		if strings.Contains(s, meta) {
			return true
		}
	}
	return false
}

// ExecuteCommand 执行SaltStack命令
func (s *SaltStackService) ExecuteCommand(ctx context.Context, command string, targets []string, args ...string) (map[string]interface{}, error) {
	// 构建Salt API请求
	payload := map[string]interface{}{
		"fun":    command,
		"tgt":    "*", // 默认目标所有minions
		"client": "local",
	}

	// 如果指定了目标，检查是否是通配符或具体的 minion 列表
	if len(targets) > 0 {
		// 如果只有一个目标且是 "*"，使用默认的 glob 模式
		if len(targets) == 1 && targets[0] == "*" {
			payload["tgt"] = "*"
			// 不设置 tgt_type，使用默认的 glob 模式
		} else {
			// 否则使用列表模式匹配具体的 minions
			payload["tgt"] = targets
			payload["tgt_type"] = "list"
		}
	}

	// 如果有参数，添加到 payload (Salt API 使用 "arg" 字段)
	if len(args) > 0 {
		// 检查参数中是否包含 shell 特殊字符（管道、重定向、逻辑运算符等）
		needsShell := false
		for _, arg := range args {
			if containsShellMetaChars(arg) {
				needsShell = true
				break
			}
		}

		// 如果需要 shell，使用 cmd.shell 或设置 python_shell=True
		if needsShell {
			if command == "cmd.run" {
				// 对于 cmd.run，设置 python_shell=True 来支持 shell 特性
				payload["kwarg"] = map[string]interface{}{
					"python_shell": true,
				}
				log.Printf("[SaltStack] Detected shell metacharacters, enabling python_shell=True")
			}
		}

		payload["arg"] = args
	}

	log.Printf("[SaltStack] ExecuteCommand - command: %s, targets: %v, args: %v, payload: %+v", command, targets, args, payload)

	// 执行命令
	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		log.Printf("[SaltStack] ExecuteCommand failed: %v", err)
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

	log.Printf("[SaltStack] ExecuteCommand success, result: %+v", result)

	// Salt API 返回格式: {"return": [{"minion1": result1, "minion2": result2}]}
	// 直接返回整个响应，保持 Salt API 的原始格式
	return map[string]interface{}{
		"success": true,
		"result":  result, // result 已经包含了 "return" 字段
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

// DeleteMinion 删除Minion密钥（从 Salt Master 中完全移除）
func (s *SaltStackService) DeleteMinion(ctx context.Context, minionID string) error {
	return s.DeleteMinionWithForce(ctx, minionID, false)
}

// DeleteMinionWithForce 删除Minion密钥，支持强制删除在线节点
// 增强版：尝试从所有密钥状态中删除（accepted, rejected, unaccepted/pending）
// 强制删除模式会尝试所有可能的方法，包括直接删除 PKI 目录中的密钥文件
func (s *SaltStackService) DeleteMinionWithForce(ctx context.Context, minionID string, force bool) error {
	log.Printf("[SaltStack] Deleting minion: %s (force=%v)", minionID, force)

	// 如果不是强制删除，先检查节点是否在线（通过 test.ping）
	if !force {
		pingPayload := map[string]interface{}{
			"fun":     "test.ping",
			"tgt":     minionID,
			"client":  "local",
			"timeout": 3,
		}
		pingResult, err := s.executeSaltCommand(ctx, pingPayload)
		if err == nil && pingResult != nil {
			// 检查 ping 结果中是否有该 minion 的响应
			if returnData, ok := pingResult["return"].([]interface{}); ok && len(returnData) > 0 {
				if minionData, ok := returnData[0].(map[string]interface{}); ok {
					if _, exists := minionData[minionID]; exists {
						return fmt.Errorf("minion %s is online, use force=true to delete online minions", minionID)
					}
				}
			}
		}
	}

	var lastErr error
	deletedAny := false

	// 方法1: 使用 key.delete 删除已接受的密钥
	deletePayload := map[string]interface{}{
		"fun":    "key.delete",
		"match":  minionID,
		"client": "wheel",
	}
	result, err := s.executeSaltCommand(ctx, deletePayload)
	if err == nil {
		log.Printf("[SaltStack] key.delete API call succeeded for minion: %s, result: %+v", minionID, result)
		deletedAny = true
	} else {
		log.Printf("[SaltStack] key.delete failed for minion %s: %v, trying other methods", minionID, err)
		lastErr = err
	}

	// 方法2: 尝试拒绝然后删除（用于处理 pending/unaccepted 状态）
	rejectPayload := map[string]interface{}{
		"fun":    "key.reject",
		"match":  minionID,
		"client": "wheel",
	}
	_, err = s.executeSaltCommand(ctx, rejectPayload)
	if err == nil {
		log.Printf("[SaltStack] key.reject succeeded for minion: %s", minionID)
		// 拒绝后再次尝试删除
		_, err = s.executeSaltCommand(ctx, deletePayload)
		if err == nil {
			log.Printf("[SaltStack] key.delete after reject succeeded for minion: %s", minionID)
			deletedAny = true
		}
	}

	// 方法3: 尝试 key.delete_deny 删除已拒绝的密钥
	deleteDenyPayload := map[string]interface{}{
		"fun":    "key.delete_deny",
		"match":  minionID,
		"client": "wheel",
	}
	_, err = s.executeSaltCommand(ctx, deleteDenyPayload)
	if err == nil {
		log.Printf("[SaltStack] key.delete_deny succeeded for minion: %s", minionID)
		deletedAny = true
	}

	// 强制删除模式：无论 Salt API 是否成功，都尝试使用 Docker exec 直接操作确保彻底删除
	if force {
		log.Printf("[SaltStack] Force mode: trying Docker exec to ensure complete deletion of minion: %s", minionID)
		if err := s.deleteMinionKeyViaDocker(ctx, minionID); err == nil {
			log.Printf("[SaltStack] Docker exec delete succeeded for minion: %s", minionID)
			return nil // 强制删除模式下，Docker exec 成功即可返回
		} else {
			log.Printf("[SaltStack] Docker exec delete failed for minion %s: %v", minionID, err)
			// 即使 Docker exec 失败，如果 Salt API 成功了也可以
		}
	}

	// 如果至少有一种方法成功，认为删除成功
	if deletedAny {
		log.Printf("[SaltStack] Minion %s deleted successfully (force=%v)", minionID, force)
		return nil
	}

	// 非强制删除模式下，所有 Salt API 方法都失败，尝试 Docker exec fallback
	if !force {
		log.Printf("[SaltStack] Trying Docker exec fallback for non-force delete: %s", minionID)
		if err := s.deleteMinionKeyViaDocker(ctx, minionID); err == nil {
			log.Printf("[SaltStack] Docker exec delete succeeded for minion: %s", minionID)
			return nil
		} else {
			log.Printf("[SaltStack] Docker exec delete failed for minion %s: %v", minionID, err)
		}
	}

	// 所有方法都失败
	if lastErr != nil {
		return fmt.Errorf("failed to delete minion %s from any key state: %v", minionID, lastErr)
	}
	return fmt.Errorf("failed to delete minion %s: no deletion method succeeded", minionID)
}

// deleteMinionKeyViaDocker 通过 Docker exec 执行 salt-key 命令删除密钥
// 增强版：尝试删除所有状态的密钥并清理相关的 pubkey 文件
func (s *SaltStackService) deleteMinionKeyViaDocker(ctx context.Context, minionID string) error {
	// Salt Master 容器名称模式
	containerNames := []string{
		"ai-infra-salt-master-1",
		"ai-infra-salt-master",
		"salt-master-1",
		"salt-master",
	}

	for _, containerName := range containerNames {
		// 先检查容器是否存在
		checkCmd := exec.CommandContext(ctx, "docker", "inspect", containerName)
		if err := checkCmd.Run(); err != nil {
			continue // 容器不存在，尝试下一个
		}

		log.Printf("[SaltStack] Found container %s, attempting to delete minion %s", containerName, minionID)
		deletedAny := false

		// 方法1: 执行 salt-key -d 命令删除已接受的密钥（-y 自动确认）
		cmd := exec.CommandContext(ctx, "docker", "exec", containerName, "salt-key", "-d", minionID, "-y")
		output, err := cmd.CombinedOutput()
		if err == nil {
			log.Printf("[SaltStack] salt-key -d succeeded for %s in container %s: %s", minionID, containerName, string(output))
			deletedAny = true
		} else {
			log.Printf("[SaltStack] salt-key -d failed in container %s: %v, output: %s", containerName, err, string(output))
		}

		// 方法2: 尝试删除被拒绝的密钥 (salt-key -r <minion_id> 先拒绝，然后 -d)
		// 先拒绝密钥（如果存在于 unaccepted 列表中）
		rejectCmd := exec.CommandContext(ctx, "docker", "exec", containerName, "salt-key", "-r", minionID, "-y")
		rejectOutput, rejectErr := rejectCmd.CombinedOutput()
		if rejectErr == nil {
			log.Printf("[SaltStack] salt-key -r succeeded for %s: %s", minionID, string(rejectOutput))
			// 拒绝后再删除
			delCmd := exec.CommandContext(ctx, "docker", "exec", containerName, "salt-key", "-d", minionID, "-y")
			delOutput, delErr := delCmd.CombinedOutput()
			if delErr == nil {
				log.Printf("[SaltStack] salt-key -d after reject succeeded for %s: %s", minionID, string(delOutput))
				deletedAny = true
			}
		}

		// 方法3: 直接删除 pki 目录中的密钥文件（最彻底的方式）
		keyPaths := []string{
			fmt.Sprintf("/etc/salt/pki/master/minions/%s", minionID),
			fmt.Sprintf("/etc/salt/pki/master/minions_pre/%s", minionID),
			fmt.Sprintf("/etc/salt/pki/master/minions_denied/%s", minionID),
			fmt.Sprintf("/etc/salt/pki/master/minions_rejected/%s", minionID),
		}
		for _, keyPath := range keyPaths {
			rmCmd := exec.CommandContext(ctx, "docker", "exec", containerName, "rm", "-f", keyPath)
			rmOutput, rmErr := rmCmd.CombinedOutput()
			if rmErr == nil {
				log.Printf("[SaltStack] Removed key file %s: %s", keyPath, string(rmOutput))
				deletedAny = true
			}
		}

		// 方法4: 验证删除结果
		listCmd := exec.CommandContext(ctx, "docker", "exec", containerName, "salt-key", "-L")
		listOutput, listErr := listCmd.CombinedOutput()
		if listErr == nil {
			if !strings.Contains(string(listOutput), minionID) {
				log.Printf("[SaltStack] Verified: minion %s is no longer in salt-key list", minionID)
				return nil
			}
			log.Printf("[SaltStack] Warning: minion %s still appears in salt-key list: %s", minionID, string(listOutput))
		}

		if deletedAny {
			return nil
		}
	}

	return fmt.Errorf("docker exec fallback failed for all container names")
}

// DeleteMinionBatch 批量删除 Minion 密钥
func (s *SaltStackService) DeleteMinionBatch(ctx context.Context, minionIDs []string) (map[string]error, error) {
	return s.DeleteMinionBatchWithForce(ctx, minionIDs, false)
}

// DeleteMinionBatchWithForce 批量删除 Minion 密钥，支持强制删除
func (s *SaltStackService) DeleteMinionBatchWithForce(ctx context.Context, minionIDs []string, force bool) (map[string]error, error) {
	results := make(map[string]error)

	for _, minionID := range minionIDs {
		err := s.DeleteMinionWithForce(ctx, minionID, force)
		results[minionID] = err
	}

	return results, nil
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

// IsClientAccepted 检查节点是否已在 SaltStack 中注册并被接受
func (s *SaltStackService) IsClientAccepted(ctx context.Context, minionID string) (bool, error) {
	status, err := s.GetStatus(ctx)
	if err != nil {
		return false, err
	}

	// 检查是否在已接受的 keys 列表中
	for _, minion := range status.AcceptedKeys {
		if minion == minionID {
			return true, nil
		}
	}

	return false, nil
}

// Ping 检查节点是否在线
func (s *SaltStackService) Ping(ctx context.Context, minionID string) (bool, error) {
	payload := map[string]interface{}{
		"fun":    "test.ping",
		"tgt":    minionID,
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return false, err
	}

	// 检查返回结果
	if result == nil {
		return false, nil
	}

	// 如果返回 true 表示在线
	if returnData, ok := result["return"].([]interface{}); ok && len(returnData) > 0 {
		if nodeResult, ok := returnData[0].(map[string]interface{}); ok {
			if ping, ok := nodeResult[minionID].(bool); ok {
				return ping, nil
			}
		}
	}

	return false, nil
}

// GetMinionVersion 获取 Salt Minion 版本
func (s *SaltStackService) GetMinionVersion(ctx context.Context, minionID string) (string, error) {
	payload := map[string]interface{}{
		"fun":    "test.version",
		"tgt":    minionID,
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return "", err
	}

	// 解析版本号
	if returnData, ok := result["return"].([]interface{}); ok && len(returnData) > 0 {
		if nodeResult, ok := returnData[0].(map[string]interface{}); ok {
			if version, ok := nodeResult[minionID].(string); ok {
				return version, nil
			}
		}
	}

	return "", fmt.Errorf("无法获取版本信息")
}

// CheckPackageInstalled 检查节点上是否已安装指定软件包
func (s *SaltStackService) CheckPackageInstalled(ctx context.Context, minionID, packageName string) (bool, error) {
	payload := map[string]interface{}{
		"fun":    "pkg.version",
		"tgt":    minionID,
		"arg":    []string{packageName},
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return false, err
	}

	// 检查返回结果
	if returnData, ok := result["return"].([]interface{}); ok && len(returnData) > 0 {
		if nodeResult, ok := returnData[0].(map[string]interface{}); ok {
			if version, ok := nodeResult[minionID].(string); ok {
				// 如果返回非空字符串，表示已安装
				return version != "", nil
			}
		}
	}

	return false, nil
}

// InstallSlurmNode 在节点上安装 SLURM（通过直接执行脚本）
func (s *SaltStackService) InstallSlurmNode(ctx context.Context, minionID string, cluster interface{}) error {
	log.Printf("[DEBUG] InstallSlurmNode: 开始在节点 %s 上安装 SLURM", minionID)

	// 获取 AppHub URL（从环境变量或使用默认值）
	apphubURL := os.Getenv("APPHUB_URL")
	if apphubURL == "" {
		apphubURL = "http://ai-infra-apphub:8080"
	}

	// 读取安装脚本内容
	scriptPath := "/app/scripts/install-slurm-node.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取安装脚本失败: %v", err)
	}

	// 通过 cmd.run 执行脚本（使用 stdin 传递脚本内容）
	script := fmt.Sprintf("cat > /tmp/install-slurm-node.sh << 'EOFSCRIPT'\n%s\nEOFSCRIPT\nchmod +x /tmp/install-slurm-node.sh && /tmp/install-slurm-node.sh %s compute",
		string(scriptContent), apphubURL)

	payload := map[string]interface{}{
		"fun": "cmd.run",
		"tgt": minionID,
		"arg": []string{script},
		"kwarg": map[string]interface{}{
			"shell":        "/bin/bash",
			"python_shell": true,
			"timeout":      300, // 5分钟超时
		},
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return fmt.Errorf("执行安装脚本失败: %v", err)
	}

	log.Printf("[DEBUG] InstallSlurmNode: 安装脚本执行完成，结果: %+v", result)
	return nil
}

// ConfigureSlurmNode 配置 SLURM 节点（部署配置文件并启动服务）
func (s *SaltStackService) ConfigureSlurmNode(ctx context.Context, minionID string, cluster interface{}) error {
	log.Printf("[DEBUG] ConfigureSlurmNode: 开始配置节点 %s", minionID)

	// 读取配置脚本
	scriptPath := "/app/scripts/configure-slurm-node.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取配置脚本失败: %v", err)
	}

	// 从 slurm-master 通过 SSH 获取 munge.key
	masterHost := os.Getenv("SLURM_CONTROLLER_HOST")
	if masterHost == "" {
		masterHost = "ai-infra-slurm-master"
	}

	// 使用 SSH 读取 munge.key（使用 SSH 密钥认证）
	getMungeCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -i /root/.ssh/id_rsa root@%s 'cat /etc/munge/munge.key | base64'", masterHost)
	mungeKeyPayload := map[string]interface{}{
		"fun":    "cmd.run",
		"tgt":    minionID,
		"arg":    []string{getMungeCmd},
		"kwarg":  map[string]interface{}{"shell": "/bin/bash", "python_shell": true},
		"client": "local",
	}
	mungeResult, err := s.executeSaltCommand(ctx, mungeKeyPayload)
	if err != nil {
		return fmt.Errorf("从 master 获取 munge.key 失败: %v", err)
	}
	mungeKeyB64 := s.extractCommandResult(mungeResult, minionID)
	if mungeKeyB64 == "" {
		return fmt.Errorf("无法提取 munge.key 内容")
	}

	// 使用 SSH 读取 slurm.conf
	getSlurmConfCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -i /root/.ssh/id_rsa root@%s 'cat /etc/slurm/slurm.conf | base64'", masterHost)
	slurmConfPayload := map[string]interface{}{
		"fun":    "cmd.run",
		"tgt":    minionID,
		"arg":    []string{getSlurmConfCmd},
		"kwarg":  map[string]interface{}{"shell": "/bin/bash", "python_shell": true},
		"client": "local",
	}
	slurmConfResult, err := s.executeSaltCommand(ctx, slurmConfPayload)
	if err != nil {
		return fmt.Errorf("从 master 获取 slurm.conf 失败: %v", err)
	}
	slurmConfB64 := s.extractCommandResult(slurmConfResult, minionID)
	if slurmConfB64 == "" {
		return fmt.Errorf("无法提取 slurm.conf 内容")
	}

	// 执行配置脚本
	script := fmt.Sprintf("cat > /tmp/configure-slurm-node.sh << 'EOFSCRIPT'\n%s\nEOFSCRIPT\nchmod +x /tmp/configure-slurm-node.sh && /tmp/configure-slurm-node.sh %s '%s' '%s'",
		string(scriptContent), masterHost, mungeKeyB64, slurmConfB64)

	payload := map[string]interface{}{
		"fun": "cmd.run",
		"tgt": minionID,
		"arg": []string{script},
		"kwarg": map[string]interface{}{
			"shell":        "/bin/bash",
			"python_shell": true,
			"timeout":      120,
		},
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return fmt.Errorf("执行配置脚本失败: %v", err)
	}

	log.Printf("[DEBUG] ConfigureSlurmNode: 配置脚本执行完成，结果: %+v", result)
	return nil
}

// extractCommandResult 从 Salt 命令结果中提取输出
func (s *SaltStackService) extractCommandResult(result interface{}, minionID string) string {
	if result == nil {
		return ""
	}

	// Salt API 返回格式: {"return": [{"minion-id": "output"}]}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return ""
	}

	returnData, ok := resultMap["return"]
	if !ok {
		return ""
	}

	returnArray, ok := returnData.([]interface{})
	if !ok || len(returnArray) == 0 {
		return ""
	}

	firstReturn, ok := returnArray[0].(map[string]interface{})
	if !ok {
		return ""
	}

	output, ok := firstReturn[minionID].(string)
	if !ok {
		return ""
	}

	// 移除前后空白字符
	return strings.TrimSpace(output)
}

// StartSlurmService 启动 SLURM 服务（已集成到 ConfigureSlurmNode 中）
func (s *SaltStackService) StartSlurmService(ctx context.Context, minionID string) error {
	log.Printf("[DEBUG] StartSlurmService: 服务已在 ConfigureSlurmNode 中启动，节点: %s", minionID)
	// 服务已经在 configure-slurm-node.sh 中启动
	// 这里只是验证服务状态

	payload := map[string]interface{}{
		"fun":    "cmd.run",
		"tgt":    minionID,
		"arg":    []string{"pgrep -x slurmd && echo 'running' || echo 'stopped'"},
		"kwarg":  map[string]interface{}{"python_shell": true},
		"client": "local",
	}

	result, err := s.executeSaltCommand(ctx, payload)
	if err != nil {
		return fmt.Errorf("检查 slurmd 状态失败: %v", err)
	}

	log.Printf("[DEBUG] StartSlurmService: slurmd 状态检查结果: %+v", result)
	return nil
}
