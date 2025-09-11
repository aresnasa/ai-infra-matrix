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

// OpenAIProvider OpenAI提供商实现
type OpenAIProvider struct {
	config *models.AIAssistantConfig
	client *http.Client
}

// NewOpenAIProvider 创建OpenAI提供商
func NewOpenAIProvider(config *models.AIAssistantConfig) *OpenAIProvider {
	return &OpenAIProvider{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetName 获取提供商名称
func (p *OpenAIProvider) GetName() string {
	return "openai"
}

// ValidateConfig 验证配置
func (p *OpenAIProvider) ValidateConfig(config *models.AIAssistantConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	if config.APIEndpoint == "" {
		config.APIEndpoint = "https://api.openai.com/v1/chat/completions"
	}
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}
	return nil
}

// Chat 发送聊天请求
func (p *OpenAIProvider) Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()
	
	// 构建OpenAI请求格式
	messages := make([]map[string]string, 0, len(request.Messages))
	for _, msg := range request.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	
	requestBody := map[string]interface{}{
		"model":       request.Model,
		"messages":    messages,
		"max_tokens":  request.MaxTokens,
		"temperature": request.Temperature,
	}
	
	// 添加系统提示（如果有）
	if request.SystemPrompt != "" {
		systemMessage := map[string]string{
			"role":    "system",
			"content": request.SystemPrompt,
		}
		messages = append([]map[string]string{systemMessage}, messages...)
		requestBody["messages"] = messages
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
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	
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
	
	// 解析OpenAI响应
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	
	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}
	
	responseTime := int(time.Since(startTime).Milliseconds())
	
	return &ChatResponse{
		Content:      openaiResp.Choices[0].Message.Content,
		TokensUsed:   openaiResp.Usage.TotalTokens,
		ResponseTime: responseTime,
		Metadata: map[string]interface{}{
			"provider": "openai",
			"model":    request.Model,
		},
	}, nil
}

// GetAvailableModels 获取可用模型列表
func (p *OpenAIProvider) GetAvailableModels(ctx context.Context) ([]ModelInfo, error) {
	// OpenAI常用模型列表
	models := []ModelInfo{
		{
			ID:          "gpt-4o",
			Name:        "GPT-4o",
			Provider:    "openai",
			Type:        "chat",
			MaxTokens:   128000,
			Capabilities: []string{"chat", "function_calling", "vision"},
			Cost: struct {
				InputTokenPrice  float64 `json:"input_token_price"`
				OutputTokenPrice float64 `json:"output_token_price"`
			}{InputTokenPrice: 0.005, OutputTokenPrice: 0.015},
		},
		{
			ID:          "gpt-4o-mini",
			Name:        "GPT-4o Mini",
			Provider:    "openai",
			Type:        "chat",
			MaxTokens:   128000,
			Capabilities: []string{"chat", "function_calling"},
			Cost: struct {
				InputTokenPrice  float64 `json:"input_token_price"`
				OutputTokenPrice float64 `json:"output_token_price"`
			}{InputTokenPrice: 0.00015, OutputTokenPrice: 0.0006},
		},
		{
			ID:          "gpt-3.5-turbo",
			Name:        "GPT-3.5 Turbo",
			Provider:    "openai",
			Type:        "chat",
			MaxTokens:   16385,
			Capabilities: []string{"chat", "function_calling"},
			Cost: struct {
				InputTokenPrice  float64 `json:"input_token_price"`
				OutputTokenPrice float64 `json:"output_token_price"`
			}{InputTokenPrice: 0.0005, OutputTokenPrice: 0.0015},
		},
	}
	
	return models, nil
}

// TestConnection 测试连接
func (p *OpenAIProvider) TestConnection(ctx context.Context) error {
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
func (p *OpenAIProvider) GetSupportedCapabilities() []string {
	return []string{"chat", "function_calling", "streaming"}
}
