package ai_providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

// ClaudeProvider Claude提供商实现
type ClaudeProvider struct {
	config *models.AIAssistantConfig
	client *http.Client
}

// NewClaudeProvider 创建Claude提供商
func NewClaudeProvider(config *models.AIAssistantConfig) *ClaudeProvider {
	return &ClaudeProvider{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetName 获取提供商名称
func (p *ClaudeProvider) GetName() string {
	return "claude"
}

// ValidateConfig 验证配置
func (p *ClaudeProvider) ValidateConfig(config *models.AIAssistantConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Claude API key is required")
	}
	if config.APIEndpoint == "" {
		config.APIEndpoint = "https://api.anthropic.com/v1/messages"
	}
	if config.Model == "" {
		config.Model = "claude-3-haiku-20240307"
	}
	return nil
}

// Chat 发送聊天请求
func (p *ClaudeProvider) Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	// 构建Claude请求格式
	messages := make([]map[string]string, 0, len(request.Messages))
	for _, msg := range request.Messages {
		// Claude不支持system角色，将其转换为user消息
		role := msg.Role
		if role == "system" {
			role = "user"
		}
		messages = append(messages, map[string]string{
			"role":    role,
			"content": msg.Content,
		})
	}

	requestBody := map[string]interface{}{
		"model":      request.Model,
		"max_tokens": request.MaxTokens,
		"messages":   messages,
	}

	// Claude使用system参数而不是system消息
	if request.SystemPrompt != "" {
		requestBody["system"] = request.SystemPrompt
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析Claude响应
	var claudeResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Claude")
	}

	responseTime := int(time.Since(startTime).Milliseconds())
	totalTokens := claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens

	return &ChatResponse{
		Content:      claudeResp.Content[0].Text,
		TokensUsed:   totalTokens,
		ResponseTime: responseTime,
		Metadata: map[string]interface{}{
			"provider":      "claude",
			"model":         request.Model,
			"input_tokens":  claudeResp.Usage.InputTokens,
			"output_tokens": claudeResp.Usage.OutputTokens,
		},
	}, nil
}

// GetAvailableModels 获取可用模型列表
func (p *ClaudeProvider) GetAvailableModels(ctx context.Context) ([]ModelInfo, error) {
	// Claude模型列表
	models := []ModelInfo{
		{
			ID:           "claude-3-5-sonnet-20241022",
			Name:         "Claude 3.5 Sonnet",
			Provider:     "claude",
			Type:         "chat",
			MaxTokens:    200000,
			Capabilities: []string{"chat", "analysis", "coding", "vision"},
			Cost: struct {
				InputTokenPrice  float64 `json:"input_token_price"`
				OutputTokenPrice float64 `json:"output_token_price"`
			}{InputTokenPrice: 0.003, OutputTokenPrice: 0.015},
		},
		{
			ID:           "claude-3-haiku-20240307",
			Name:         "Claude 3 Haiku",
			Provider:     "claude",
			Type:         "chat",
			MaxTokens:    200000,
			Capabilities: []string{"chat", "fast_response"},
			Cost: struct {
				InputTokenPrice  float64 `json:"input_token_price"`
				OutputTokenPrice float64 `json:"output_token_price"`
			}{InputTokenPrice: 0.00025, OutputTokenPrice: 0.00125},
		},
		{
			ID:           "claude-3-opus-20240229",
			Name:         "Claude 3 Opus",
			Provider:     "claude",
			Type:         "chat",
			MaxTokens:    200000,
			Capabilities: []string{"chat", "analysis", "complex_reasoning"},
			Cost: struct {
				InputTokenPrice  float64 `json:"input_token_price"`
				OutputTokenPrice float64 `json:"output_token_price"`
			}{InputTokenPrice: 0.015, OutputTokenPrice: 0.075},
		},
	}

	return models, nil
}

// TestConnection 测试连接
func (p *ClaudeProvider) TestConnection(ctx context.Context) error {
	testRequest := ChatRequest{
		Model:       p.config.Model,
		Messages:    []ChatMessage{{Role: "user", Content: "Hello"}},
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := p.Chat(ctx, testRequest)
	return err
}

// GetSupportedCapabilities 获取支持的功能
func (p *ClaudeProvider) GetSupportedCapabilities() []string {
	return []string{"chat", "analysis", "long_context"}
}
