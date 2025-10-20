package ai_providers

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

// DefaultProviderFactory 默认提供商工厂
type DefaultProviderFactory struct{}

// NewProviderFactory 创建提供商工厂
func NewProviderFactory() ProviderFactory {
	return &DefaultProviderFactory{}
}

// CreateProvider 根据配置创建提供商实例
func (f *DefaultProviderFactory) CreateProvider(config *models.AIAssistantConfig) (AIProvider, error) {
	switch config.Provider {
	case models.ProviderOpenAI:
		provider := NewOpenAIProvider(config)
		if err := provider.ValidateConfig(config); err != nil {
			return nil, fmt.Errorf("invalid OpenAI config: %v", err)
		}
		return provider, nil

	case models.ProviderClaude:
		provider := NewClaudeProvider(config)
		if err := provider.ValidateConfig(config); err != nil {
			return nil, fmt.Errorf("invalid Claude config: %v", err)
		}
		return provider, nil

	case models.ProviderDeepSeek:
		return f.createDeepSeekProvider(config)

	case models.ProviderGLM:
		return f.createGLMProvider(config)

	case models.ProviderQwen:
		return f.createQwenProvider(config)

	case models.ProviderLocal:
		return f.createLocalProvider(config)

	case models.ProviderMCP:
		return f.createMCPProvider(config)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// GetSupportedProviders 获取支持的提供商列表
func (f *DefaultProviderFactory) GetSupportedProviders() []string {
	return []string{
		string(models.ProviderOpenAI),
		string(models.ProviderClaude),
		string(models.ProviderDeepSeek),
		string(models.ProviderGLM),
		string(models.ProviderQwen),
		string(models.ProviderLocal),
		string(models.ProviderMCP),
	}
}

// createDeepSeekProvider 创建DeepSeek提供商（使用OpenAI兼容接口）
func (f *DefaultProviderFactory) createDeepSeekProvider(config *models.AIAssistantConfig) (AIProvider, error) {
	// DeepSeek使用OpenAI兼容的API
	deepSeekConfig := *config
	if deepSeekConfig.APIEndpoint == "" {
		deepSeekConfig.APIEndpoint = "https://api.deepseek.com/v1/chat/completions"
	}
	deepSeekConfig.APIEndpoint = normalizeDeepSeekEndpoint(deepSeekConfig.APIEndpoint)
	
	// 从环境变量读取默认模型，支持不同模式
	if deepSeekConfig.Model == "" {
		// 优先使用 DEEPSEEK_DEFAULT_MODEL，如果未设置则使用 deepseek-chat
		defaultModel := os.Getenv("DEEPSEEK_DEFAULT_MODEL")
		if defaultModel == "" {
			defaultModel = os.Getenv("DEEPSEEK_CHAT_MODEL")
		}
		if defaultModel == "" {
			defaultModel = "deepseek-chat" // 最终回退值
		}
		deepSeekConfig.Model = defaultModel
	}

	provider := NewOpenAIProvider(&deepSeekConfig)
	if err := provider.ValidateConfig(&deepSeekConfig); err != nil {
		return nil, fmt.Errorf("invalid DeepSeek config: %v", err)
	}

	// 将默认值写回原始配置，确保后续请求使用正确模型和端点
	config.APIEndpoint = deepSeekConfig.APIEndpoint
	config.Model = deepSeekConfig.Model
	return provider, nil
}

// normalizeDeepSeekEndpoint 确保 DeepSeek 端点包含正确的路径
func normalizeDeepSeekEndpoint(endpoint string) string {
	if endpoint == "" {
		return "https://api.deepseek.com/v1/chat/completions"
	}

	// 移除末尾的斜杠
	endpoint = strings.TrimSuffix(endpoint, "/")

	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		// 如果解析失败，返回默认端点
		return "https://api.deepseek.com/v1/chat/completions"
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	
	// 检查并补全路径
	switch {
	case path == "" || path == "/":
		// 没有路径或只有根路径，添加完整路径
		parsed.Path = "/v1/chat/completions"
	case path == "/v1":
		// 只有 /v1，添加 chat/completions
		parsed.Path = "/v1/chat/completions"
	case strings.HasSuffix(path, "/chat/completions"):
		// 已经有正确的端点
		// 不做修改
	case strings.Contains(path, "/chat/completions"):
		// 已经包含 chat/completions
		// 不做修改
	default:
		// 其他情况，在现有路径后添加 /v1/chat/completions
		if strings.HasSuffix(path, "/v1") {
			parsed.Path = path + "/chat/completions"
		} else {
			parsed.Path = "/v1/chat/completions"
		}
	}

	result := parsed.String()
	fmt.Printf("DEBUG: normalizeDeepSeekEndpoint: %s -> %s\n", endpoint, result)
	return result
}

// createGLMProvider 创建GLM提供商（智谱AI）
func (f *DefaultProviderFactory) createGLMProvider(config *models.AIAssistantConfig) (AIProvider, error) {
	// GLM使用自定义API格式，这里简化使用OpenAI兼容接口
	glmConfig := *config
	if glmConfig.APIEndpoint == "" {
		glmConfig.APIEndpoint = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	}
	if glmConfig.Model == "" {
		glmConfig.Model = "glm-4"
	}

	provider := NewOpenAIProvider(&glmConfig)
	if err := provider.ValidateConfig(&glmConfig); err != nil {
		return nil, fmt.Errorf("invalid GLM config: %v", err)
	}

	config.APIEndpoint = glmConfig.APIEndpoint
	config.Model = glmConfig.Model
	return provider, nil
}

// createQwenProvider 创建通义千问提供商
func (f *DefaultProviderFactory) createQwenProvider(config *models.AIAssistantConfig) (AIProvider, error) {
	// 通义千问使用自定义API格式，这里简化实现
	qwenConfig := *config
	if qwenConfig.APIEndpoint == "" {
		qwenConfig.APIEndpoint = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	}
	if qwenConfig.Model == "" {
		qwenConfig.Model = "qwen-turbo"
	}

	// 暂时使用OpenAI兼容的实现，实际应该实现专门的通义千问Provider
	provider := NewOpenAIProvider(&qwenConfig)
	if err := provider.ValidateConfig(&qwenConfig); err != nil {
		return nil, fmt.Errorf("invalid Qwen config: %v", err)
	}

	config.APIEndpoint = qwenConfig.APIEndpoint
	config.Model = qwenConfig.Model
	return provider, nil
}

// createLocalProvider 创建本地提供商
func (f *DefaultProviderFactory) createLocalProvider(config *models.AIAssistantConfig) (AIProvider, error) {
	// 本地提供商通常使用Ollama或其他本地API
	localConfig := *config
	if localConfig.APIEndpoint == "" {
		localConfig.APIEndpoint = "http://localhost:11434/v1/chat/completions" // Ollama默认端点
	}
	if localConfig.Model == "" {
		localConfig.Model = "llama2"
	}

	provider := NewOpenAIProvider(&localConfig)
	// 本地提供商不需要API密钥
	localConfig.APIKey = "local"
	if err := provider.ValidateConfig(&localConfig); err != nil {
		return nil, fmt.Errorf("invalid local config: %v", err)
	}

	config.APIEndpoint = localConfig.APIEndpoint
	config.Model = localConfig.Model
	return provider, nil
}

// createMCPProvider 创建MCP提供商
func (f *DefaultProviderFactory) createMCPProvider(config *models.AIAssistantConfig) (AIProvider, error) {
	// MCP (Model Context Protocol) 提供商是未来扩展，暂时返回错误
	return nil, fmt.Errorf("MCP provider not implemented yet")
}
