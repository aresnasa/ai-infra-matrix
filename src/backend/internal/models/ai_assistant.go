package models

import (
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
	"gorm.io/gorm"
)

// AIProvider AI服务提供商类型
type AIProvider string

const (
	ProviderOpenAI   AIProvider = "openai"
	ProviderClaude   AIProvider = "claude"
	ProviderLocal    AIProvider = "local"
	ProviderMCP      AIProvider = "mcp"
	ProviderCustom   AIProvider = "custom"
	ProviderDeepSeek AIProvider = "deepseek"
	ProviderGLM      AIProvider = "glm"
	ProviderQwen     AIProvider = "qwen"
)

// AIModelType AI模型类型
type AIModelType string

const (
	ModelTypeChat       AIModelType = "chat"
	ModelTypeCompletion AIModelType = "completion"
	ModelTypeEmbedding  AIModelType = "embedding"
	ModelTypeImage      AIModelType = "image"
)

// AIAssistantConfig AI助手配置
type AIAssistantConfig struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	Name             string         `json:"name" gorm:"not null;size:100"`
	Provider         AIProvider     `json:"provider" gorm:"not null"`
	ModelType        AIModelType    `json:"model_type" gorm:"default:'chat'"`
	APIKey           string         `json:"-" gorm:"type:text"` // 加密存储，不在JSON中暴露
	APISecret        string         `json:"-" gorm:"type:text"` // 加密存储，不在JSON中暴露
	APIEndpoint      string         `json:"api_endpoint" gorm:"size:500"`
	Model            string         `json:"model" gorm:"size:100"`
	MaxTokens        int            `json:"max_tokens" gorm:"default:4096"`
	Temperature      float32        `json:"temperature" gorm:"default:0.7"`
	TopP             float32        `json:"top_p" gorm:"default:1.0"`
	FrequencyPenalty float32        `json:"frequency_penalty" gorm:"default:0.0"`
	PresencePenalty  float32        `json:"presence_penalty" gorm:"default:0.0"`
	SystemPrompt     string         `json:"system_prompt" gorm:"type:text"`
	IsEnabled        bool           `json:"is_enabled" gorm:"default:true"`
	IsDefault        bool           `json:"is_default" gorm:"default:false"`
	MCPConfig        *MCPConfig     `json:"mcp_config,omitempty" gorm:"type:text"` // JSON存储MCP配置
	RateLimitPerHour int            `json:"rate_limit_per_hour" gorm:"default:100"`
	RateLimitPerDay  int            `json:"rate_limit_per_day" gorm:"default:1000"`
	TimeoutSeconds   int            `json:"timeout_seconds" gorm:"default:60"`
	RetryAttempts    int            `json:"retry_attempts" gorm:"default:3"`
	Headers          string         `json:"headers" gorm:"type:text"`    // JSON格式的额外请求头
	Parameters       string         `json:"parameters" gorm:"type:text"` // JSON格式的额外参数
	Description      string         `json:"description" gorm:"size:500"`
	IconURL          string         `json:"icon_url" gorm:"size:500"`
	Category         string         `json:"category" gorm:"size:50"` // 模型分类，如：通用对话、代码生成、专业领域等
	Tags             string         `json:"tags" gorm:"type:text"`   // JSON格式的标签数组
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	// HasAPIKey 用于前端显示是否已配置密钥
	HasAPIKey    bool `json:"has_api_key" gorm:"-"`
	HasAPISecret bool `json:"has_api_secret" gorm:"-"`
}

// MCPConfig Model Context Protocol配置
type MCPConfig struct {
	ServerURL      string            `json:"server_url"`
	Capabilities   []string          `json:"capabilities"`
	Tools          []MCPTool         `json:"tools"`
	Authentication map[string]string `json:"authentication,omitempty"`
}

// MCPTool MCP工具定义
type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Schema      interface{} `json:"schema"`
}

// AIConversation AI对话记录
type AIConversation struct {
	ID         uint               `json:"id" gorm:"primaryKey"`
	UserID     uint               `json:"user_id" gorm:"not null;index"`
	User       *User              `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Title      string             `json:"title" gorm:"size:200"`
	ConfigID   uint               `json:"config_id" gorm:"not null;index"`
	Config     *AIAssistantConfig `json:"config,omitempty" gorm:"foreignKey:ConfigID"`
	Messages   []AIMessage        `json:"messages,omitempty" gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
	Context    string             `json:"context" gorm:"type:text"` // 项目上下文等
	IsActive   bool               `json:"is_active" gorm:"default:true"`
	TokensUsed int                `json:"tokens_used" gorm:"default:0"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	DeletedAt  gorm.DeletedAt     `json:"deleted_at,omitempty" gorm:"index"`
}

// AIMessage AI对话消息
type AIMessage struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	ConversationID uint            `json:"conversation_id" gorm:"not null;index"`
	Conversation   *AIConversation `json:"conversation,omitempty" gorm:"foreignKey:ConversationID"`
	Role           string          `json:"role" gorm:"not null"` // user, assistant, system
	Content        string          `json:"content" gorm:"type:text;not null"`
	Metadata       string          `json:"metadata" gorm:"type:text"` // JSON格式的元数据
	TokensUsed     int             `json:"tokens_used" gorm:"default:0"`
	ResponseTime   int             `json:"response_time"` // 响应时间（毫秒）
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      gorm.DeletedAt  `json:"deleted_at,omitempty" gorm:"index"`
}

// AIUsageStats AI使用统计
type AIUsageStats struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"not null;index"`
	ConfigID        uint      `json:"config_id" gorm:"not null;index"`
	Date            time.Time `json:"date" gorm:"not null;index"`
	RequestCount    int       `json:"request_count" gorm:"default:0"`
	TokensUsed      int       `json:"tokens_used" gorm:"default:0"`
	SuccessCount    int       `json:"success_count" gorm:"default:0"`
	ErrorCount      int       `json:"error_count" gorm:"default:0"`
	AverageResponse int       `json:"average_response" gorm:"default:0"` // 平均响应时间
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定表名
func (AIAssistantConfig) TableName() string {
	return "ai_assistant_configs"
}

func (AIConversation) TableName() string {
	return "ai_conversations"
}

func (AIMessage) TableName() string {
	return "ai_messages"
}

func (AIUsageStats) TableName() string {
	return "ai_usage_stats"
}

// BeforeCreate 创建前的钩子
func (c *AIAssistantConfig) BeforeCreate(tx *gorm.DB) error {
	// 如果设置为默认，取消其他默认配置
	if c.IsDefault {
		tx.Model(&AIAssistantConfig{}).Where("is_default = ?", true).Update("is_default", false)
	}
	// 加密敏感字段
	c.encryptSensitiveFields()
	return nil
}

func (c *AIAssistantConfig) BeforeUpdate(tx *gorm.DB) error {
	// 如果设置为默认，取消其他默认配置
	if c.IsDefault {
		tx.Model(&AIAssistantConfig{}).Where("id != ? AND is_default = ?", c.ID, true).Update("is_default", false)
	}
	// 加密敏感字段
	c.encryptSensitiveFields()
	return nil
}

// BeforeSave 保存前加密敏感字段
func (c *AIAssistantConfig) BeforeSave(tx *gorm.DB) error {
	c.encryptSensitiveFields()
	return nil
}

// AfterFind 查询后解密敏感字段并设置标志
func (c *AIAssistantConfig) AfterFind(tx *gorm.DB) error {
	// 在解密前记录是否有密钥
	c.HasAPIKey = c.APIKey != ""
	c.HasAPISecret = c.APISecret != ""
	// 解密敏感字段（但JSON中不会暴露，只有内部使用）
	c.decryptSensitiveFields()
	return nil
}

// encryptSensitiveFields 加密敏感字段
func (c *AIAssistantConfig) encryptSensitiveFields() {
	// 加密 API Key
	if c.APIKey != "" {
		c.APIKey = utils.EncryptSensitiveField(c.APIKey)
	}
	// 加密 API Secret
	if c.APISecret != "" {
		c.APISecret = utils.EncryptSensitiveField(c.APISecret)
	}
}

// decryptSensitiveFields 解密敏感字段
func (c *AIAssistantConfig) decryptSensitiveFields() {
	// 解密 API Key
	if c.APIKey != "" {
		c.APIKey = utils.DecryptSensitiveField(c.APIKey)
	}
	// 解密 API Secret
	if c.APISecret != "" {
		c.APISecret = utils.DecryptSensitiveField(c.APISecret)
	}
}
