package ai_providers

import (
	"context"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

// ChatMessage 统一的聊天消息格式
type ChatMessage struct {
	Role    string `json:"role"`    // user, assistant, system
	Content string `json:"content"`
}

// ChatRequest 统一的聊天请求格式
type ChatRequest struct {
	Model        string        `json:"model"`
	Messages     []ChatMessage `json:"messages"`
	MaxTokens    int           `json:"max_tokens,omitempty"`
	Temperature  float32       `json:"temperature,omitempty"`
	TopP         float32       `json:"top_p,omitempty"`
	SystemPrompt string        `json:"system_prompt,omitempty"`
	Stream       bool          `json:"stream,omitempty"`
}

// ChatResponse 统一的聊天响应格式
type ChatResponse struct {
	Content      string                 `json:"content"`
	TokensUsed   int                    `json:"tokens_used"`
	ResponseTime int                    `json:"response_time"` // 毫秒
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Error        error                  `json:"error,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Type        string   `json:"type"` // chat, completion, embedding, image
	MaxTokens   int      `json:"max_tokens"`
	Capabilities []string `json:"capabilities"`
	Cost        struct {
		InputTokenPrice  float64 `json:"input_token_price"`  // 每1K token价格
		OutputTokenPrice float64 `json:"output_token_price"` // 每1K token价格
	} `json:"cost"`
}

// AIProvider 统一的AI提供商接口
type AIProvider interface {
	// GetName 获取提供商名称
	GetName() string
	
	// Chat 发送聊天请求
	Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error)
	
	// GetAvailableModels 获取可用模型列表
	GetAvailableModels(ctx context.Context) ([]ModelInfo, error)
	
	// TestConnection 测试连接
	TestConnection(ctx context.Context) error
	
	// ValidateConfig 验证配置
	ValidateConfig(config *models.AIAssistantConfig) error
	
	// GetSupportedCapabilities 获取支持的功能
	GetSupportedCapabilities() []string
}

// ProviderFactory 提供商工厂接口
type ProviderFactory interface {
	// CreateProvider 根据配置创建提供商实例
	CreateProvider(config *models.AIAssistantConfig) (AIProvider, error)
	
	// GetSupportedProviders 获取支持的提供商列表
	GetSupportedProviders() []string
}

// StreamResponse 流式响应
type StreamResponse struct {
	Content   string                 `json:"content"`
	Done      bool                   `json:"done"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     error                  `json:"error,omitempty"`
}

// StreamingProvider 支持流式响应的提供商接口
type StreamingProvider interface {
	AIProvider
	
	// ChatStream 发送流式聊天请求
	ChatStream(ctx context.Context, request ChatRequest) (<-chan StreamResponse, error)
}
