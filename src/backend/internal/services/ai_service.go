package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services/ai_providers"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// AIService AI服务接口
type AIService interface {
	// 配置管理
	CreateConfig(config *models.AIAssistantConfig) error
	GetConfig(id uint) (*models.AIAssistantConfig, error)
	GetDefaultConfig() (*models.AIAssistantConfig, error)
	UpdateConfig(config *models.AIAssistantConfig) error
	DeleteConfig(id uint) error
	ListConfigs() ([]models.AIAssistantConfig, error)
	InitDefaultConfigs() error

	// 对话管理
	CreateConversation(userID uint, configID uint, title string, context string) (*models.AIConversation, error)
	GetConversation(id uint) (*models.AIConversation, error)
	ListUserConversations(userID uint) ([]models.AIConversation, error)
	UpdateConversation(conversation *models.AIConversation) error
	DeleteConversation(id uint) error
	StopConversation(id uint, userID uint) error
	ResumeConversation(id uint, userID uint) error

	// 消息处理
	SendMessage(conversationID uint, userMessage string) (*models.AIMessage, error)
	GetMessages(conversationID uint) ([]models.AIMessage, error)

	// 统计
	GetUsageStats(userID uint, startDate, endDate time.Time) ([]models.AIUsageStats, error)
	RecordUsage(userID, configID uint, tokenUsed, responseTime int, success bool) error

	// 新增的机器人管理功能
	TestConnection(config *models.AIAssistantConfig) (map[string]interface{}, error)
	GetAvailableModels(provider models.AIProvider) ([]map[string]interface{}, error)
	GetMessageStatus(messageID uint, userID uint) (string, interface{}, error)
	CloneConfig(id uint, newName string) (*models.AIAssistantConfig, error)
	BatchUpdateConfigs(configIDs []uint, updates map[string]interface{}) error
	
	// 模型对比功能
	CompareModels(conversationID uint, userMessage string, configIDs []uint) (*ModelComparisonResult, error)
}

// ModelComparisonResult 模型对比结果
type ModelComparisonResult struct {
	MessageID string                    `json:"message_id"`
	UserMessage string                  `json:"user_message"`
	Results   []ModelComparisonResponse `json:"results"`
	CreatedAt time.Time                 `json:"created_at"`
}

// ModelComparisonResponse 单个模型的对比响应
type ModelComparisonResponse struct {
	ConfigID     uint                   `json:"config_id"`
	ConfigName   string                 `json:"config_name"`
	Provider     string                 `json:"provider"`
	Model        string                 `json:"model"`
	Content      string                 `json:"content"`
	TokensUsed   int                    `json:"tokens_used"`
	ResponseTime int                    `json:"response_time"`
	Status       string                 `json:"status"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// aiServiceImpl AI服务实现
type aiServiceImpl struct {
	db                 *gorm.DB
	cryptoService      *utils.CryptoService
	messagePersistence *MessagePersistenceService
	messageRetrieval   *MessageRetrievalService
	kafkaService       *KafkaMessageService
	providerFactory    ai_providers.ProviderFactory
}

// NewAIService 创建AI服务实例
func NewAIService() AIService {
	return &aiServiceImpl{
		db:              database.DB,
		cryptoService:   database.CryptoService,
		providerFactory: ai_providers.NewProviderFactory(),
	}
}

// NewAIServiceWithDependencies 创建带有依赖的AI服务实例
func NewAIServiceWithDependencies(
	db *gorm.DB,
	cryptoService *utils.CryptoService,
	messagePersistence *MessagePersistenceService,
	messageRetrieval *MessageRetrievalService,
	kafkaService *KafkaMessageService,
) AIService {
	return &aiServiceImpl{
		db:                 db,
		cryptoService:      cryptoService,
		messagePersistence: messagePersistence,
		messageRetrieval:   messageRetrieval,
		kafkaService:       kafkaService,
		providerFactory:    ai_providers.NewProviderFactory(),
	}
}

// 配置管理
func (s *aiServiceImpl) CreateConfig(config *models.AIAssistantConfig) error {
	// 加密API密钥
	if config.APIKey != "" {
		encryptedKey, err := s.cryptoService.Encrypt(config.APIKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %v", err)
		}
		config.APIKey = encryptedKey
	}

	return s.db.Create(config).Error
}

func (s *aiServiceImpl) GetConfig(id uint) (*models.AIAssistantConfig, error) {
	var config models.AIAssistantConfig
	err := s.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}

	// 安全解密API密钥（如果解密失败，会返回原数据）
	if config.APIKey != "" && config.APIKey != "***" {
		config.APIKey = s.cryptoService.DecryptSafely(config.APIKey)
	}

	return &config, nil
}

func (s *aiServiceImpl) GetDefaultConfig() (*models.AIAssistantConfig, error) {
	var config models.AIAssistantConfig
	err := s.db.Where("is_default = ? AND is_enabled = ?", true, true).First(&config).Error
	if err != nil {
		return nil, err
	}

	// 安全解密API密钥（如果解密失败，会返回原数据）
	if config.APIKey != "" && config.APIKey != "***" {
		config.APIKey = s.cryptoService.DecryptSafely(config.APIKey)
	}

	return &config, nil
}

func (s *aiServiceImpl) UpdateConfig(config *models.AIAssistantConfig) error {
	// 如果API密钥是占位符"***"，则从数据库获取原有密钥
	if config.APIKey == "***" {
		var existingConfig models.AIAssistantConfig
		if err := s.db.First(&existingConfig, config.ID).Error; err != nil {
			return fmt.Errorf("failed to get existing config: %v", err)
		}
		config.APIKey = existingConfig.APIKey
		logrus.Debug("API key preserved from existing config (was masked)")
	} else if config.APIKey != "" {
		// 只有当API密钥未加密时才加密
		if !s.cryptoService.IsEncrypted(config.APIKey) {
			encryptedKey, err := s.cryptoService.Encrypt(config.APIKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt API key: %v", err)
			}
			config.APIKey = encryptedKey
			logrus.Debug("API key encrypted during config update")
		} else {
			logrus.Debug("API key already encrypted, skipping encryption")
		}
	}

	return s.db.Save(config).Error
}

func (s *aiServiceImpl) DeleteConfig(id uint) error {
	// 检查配置是否存在
	var config models.AIAssistantConfig
	if err := s.db.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("配置不存在")
		}
		return fmt.Errorf("查询配置失败: %v", err)
	}

	// 检查是否有对话正在使用此配置
	var conversationCount int64
	if err := s.db.Model(&models.AIConversation{}).Where("config_id = ?", id).Count(&conversationCount).Error; err != nil {
		logrus.Warnf("检查关联对话失败: %v", err)
	}

	// 软删除配置
	if err := s.db.Delete(&config).Error; err != nil {
		return fmt.Errorf("删除配置失败: %v", err)
	}

	logrus.Infof("配置删除成功: ID=%d, Name=%s, 关联对话数=%d", id, config.Name, conversationCount)
	return nil
}

func (s *aiServiceImpl) ListConfigs() ([]models.AIAssistantConfig, error) {
	var configs []models.AIAssistantConfig
	err := s.db.Find(&configs).Error

	// 不返回解密的API密钥（出于安全考虑）
	for i := range configs {
		if configs[i].APIKey != "" {
			configs[i].APIKey = "***"
		}
	}

	return configs, err
}

// 对话管理
func (s *aiServiceImpl) CreateConversation(userID uint, configID uint, title string, context string) (*models.AIConversation, error) {
	conversation := &models.AIConversation{
		UserID:   userID,
		ConfigID: configID,
		Title:    title,
		Context:  context,
		IsActive: true,
	}

	err := s.db.Create(conversation).Error
	if err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *aiServiceImpl) GetConversation(id uint) (*models.AIConversation, error) {
	var conversation models.AIConversation
	err := s.db.Preload("Messages").Preload("Config").First(&conversation, id).Error
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (s *aiServiceImpl) ListUserConversations(userID uint) ([]models.AIConversation, error) {
	var conversations []models.AIConversation
	err := s.db.Where("user_id = ?", userID).
		Preload("Config").
		Order("updated_at DESC").
		Find(&conversations).Error

	return conversations, err
}

func (s *aiServiceImpl) UpdateConversation(conversation *models.AIConversation) error {
	return s.db.Save(conversation).Error
}

func (s *aiServiceImpl) DeleteConversation(id uint) error {
	return s.db.Delete(&models.AIConversation{}, id).Error
}

// StopConversation 停止对话
func (s *aiServiceImpl) StopConversation(id uint, userID uint) error {
	// 验证对话是否属于用户
	var conversation models.AIConversation
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("对话不存在或无权访问")
		}
		return fmt.Errorf("查询对话失败: %v", err)
	}

	// 更新对话状态为停止
	updates := map[string]interface{}{
		"status":     "stopped",
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&conversation).Updates(updates).Error; err != nil {
		return fmt.Errorf("停止对话失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"conversation_id": id,
		"user_id":         userID,
	}).Info("对话已停止")

	return nil
}

// ResumeConversation 恢复对话
func (s *aiServiceImpl) ResumeConversation(id uint, userID uint) error {
	// 验证对话是否属于用户
	var conversation models.AIConversation
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("对话不存在或无权访问")
		}
		return fmt.Errorf("查询对话失败: %v", err)
	}

	// 更新对话状态为活跃
	updates := map[string]interface{}{
		"status":     "active",
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&conversation).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复对话失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"conversation_id": id,
		"user_id":         userID,
	}).Info("对话已恢复")

	return nil
}

// 消息处理
func (s *aiServiceImpl) SendMessage(conversationID uint, userMessage string) (*models.AIMessage, error) {
	// 获取对话信息
	conversation, err := s.GetConversation(conversationID)
	if err != nil {
		return nil, err
	}

	// 获取配置信息
	config, err := s.GetConfig(conversation.ConfigID)
	if err != nil {
		return nil, err
	}

	// 安全解密API密钥（GetConfig已经解密过了，但这里再确认一次）
	// 注意：GetConfig 已经使用 DecryptSafely 解密了，这里通常不需要再次解密
	// 但为了兼容性保留此代码，使用安全解密避免重复解密导致错误
	if config.APIKey != "" && config.APIKey != "***" {
		// 检查是否已解密（如果看起来像加密数据则尝试解密）
		if s.cryptoService.IsEncrypted(config.APIKey) {
			config.APIKey = s.cryptoService.DecryptSafely(config.APIKey)
		}
	}

	// 保存用户消息
	userMsg := &models.AIMessage{
		ConversationID: conversationID,
		Role:           "user",
		Content:        userMessage,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 使用消息持久化服务保存用户消息
	if s.messagePersistence != nil {
		if err := s.messagePersistence.SaveMessage(userMsg); err != nil {
			return nil, fmt.Errorf("failed to save user message: %v", err)
		}
	} else {
		if err := s.db.Create(userMsg).Error; err != nil {
			return nil, err
		}
	}

	// 创建AI提供商
	provider, err := s.providerFactory.CreateProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI provider: %v", err)
	}

	// 获取历史消息
	var historyMessages []models.AIMessage
	s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(20). // 限制历史消息数量
		Find(&historyMessages)

	// 构建消息历史
	chatMessages := make([]ai_providers.ChatMessage, 0, len(historyMessages)+1)
	for _, msg := range historyMessages {
		chatMessages = append(chatMessages, ai_providers.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 添加当前用户消息
	chatMessages = append(chatMessages, ai_providers.ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// 构建请求
	request := ai_providers.ChatRequest{
		Model:        config.Model,
		Messages:     chatMessages,
		MaxTokens:    config.MaxTokens,
		Temperature:  config.Temperature,
		SystemPrompt: config.SystemPrompt,
	}

	// 调用AI API
	ctx := context.Background()
	startTime := time.Now()
	response, err := provider.Chat(ctx, request)
	responseTime := int(time.Since(startTime).Milliseconds())

	// 记录使用统计
	tokensUsed := 0
	if response != nil {
		tokensUsed = response.TokensUsed
	}
	s.RecordUsage(conversation.UserID, config.ID, tokensUsed, responseTime, err == nil)

	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %v", err)
	}

	// 保存AI回复
	aiMsg := &models.AIMessage{
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response.Content,
		TokensUsed:     response.TokensUsed,
		ResponseTime:   response.ResponseTime,
		Metadata:       "", // 可以存储response.Metadata的JSON
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 序列化metadata
	if response.Metadata != nil {
		if metadataBytes, err := json.Marshal(response.Metadata); err == nil {
			aiMsg.Metadata = string(metadataBytes)
		}
	}

	// 使用消息持久化服务保存AI回复
	if s.messagePersistence != nil {
		if err := s.messagePersistence.SaveMessage(aiMsg); err != nil {
			return nil, fmt.Errorf("failed to save AI message: %v", err)
		}
	} else {
		if err := s.db.Create(aiMsg).Error; err != nil {
			return nil, err
		}
		// 更新对话的tokens使用量
		conversation.TokensUsed += response.TokensUsed
		s.db.Save(conversation)
	}

	// 发送消息事件到Kafka
	if s.kafkaService != nil {
		if err := s.kafkaService.SendChatMessage(conversation.UserID, &conversationID, userMessage, map[string]interface{}{
			"ai_response": response.Content,
			"tokens_used": response.TokensUsed,
			"provider":    string(config.Provider),
			"model":       config.Model,
		}); err != nil {
			logrus.Errorf("Failed to send message to Kafka: %v", err)
		}
	}

	return aiMsg, nil
}

func (s *aiServiceImpl) GetMessages(conversationID uint) ([]models.AIMessage, error) {
	// 使用优化的消息检索服务
	if s.messageRetrieval != nil {
		messages, err := s.messageRetrieval.GetMessagesWithOptimization(conversationID, 0, &MessageQueryOptions{
			SortBy:    "created_at",
			SortOrder: "asc",
		})
		if err != nil {
			return nil, err
		}

		// 转换类型
		result := make([]models.AIMessage, len(messages))
		for i, msg := range messages {
			result[i] = *msg
		}
		return result, nil
	}

	// 回退到原始实现
	var messages []models.AIMessage
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}

// 统计功能
func (s *aiServiceImpl) GetUsageStats(userID uint, startDate, endDate time.Time) ([]models.AIUsageStats, error) {
	var stats []models.AIUsageStats
	err := s.db.Where("user_id = ? AND date BETWEEN ? AND ?", userID, startDate, endDate).
		Find(&stats).Error

	return stats, err
}

func (s *aiServiceImpl) RecordUsage(userID, configID uint, tokenUsed, responseTime int, success bool) error {
	stats := &models.AIUsageStats{
		UserID:          userID,
		ConfigID:        configID,
		Date:            time.Now().Truncate(24 * time.Hour),
		TokensUsed:      tokenUsed,
		RequestCount:    1,
		SuccessCount:    0,
		ErrorCount:      0,
		AverageResponse: responseTime,
	}

	if success {
		stats.SuccessCount = 1
	} else {
		stats.ErrorCount = 1
	}

	// 使用UPSERT语法更新或插入统计数据
	err := s.db.Raw(`
		INSERT INTO ai_usage_stats (user_id, config_id, date, tokens_used, request_count, success_count, error_count, average_response, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (user_id, config_id, date)
		DO UPDATE SET
			tokens_used = ai_usage_stats.tokens_used + EXCLUDED.tokens_used,
			request_count = ai_usage_stats.request_count + EXCLUDED.request_count,
			success_count = ai_usage_stats.success_count + EXCLUDED.success_count,
			error_count = ai_usage_stats.error_count + EXCLUDED.error_count,
			average_response = (ai_usage_stats.average_response + EXCLUDED.average_response) / 2,
			updated_at = NOW()
	`, userID, configID, stats.Date, tokenUsed, 1, stats.SuccessCount, stats.ErrorCount, responseTime).Error

	return err
}

// TestConnection 测试机器人连接
func (s *aiServiceImpl) TestConnection(config *models.AIAssistantConfig) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"status":        "unknown",
		"message":       "",
		"response_time": 0,
	}

	startTime := time.Now()

	// 安全解密API密钥
	if config.APIKey != "" && config.APIKey != "***" {
		config.APIKey = s.cryptoService.DecryptSafely(config.APIKey)
	}

	// 创建AI提供商
	provider, err := s.providerFactory.CreateProvider(config)
	if err != nil {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("Failed to create provider: %v", err)
		result["response_time"] = int(time.Since(startTime).Milliseconds())
		return result, err
	}

	// 测试连接
	ctx := context.Background()
	err = provider.TestConnection(ctx)

	result["response_time"] = int(time.Since(startTime).Milliseconds())

	if err != nil {
		result["status"] = "error"
		result["message"] = err.Error()
		return result, err
	}

	result["status"] = "success"
	result["message"] = "Connection test successful"
	result["provider"] = provider.GetName()
	result["capabilities"] = provider.GetSupportedCapabilities()

	return result, nil
}

// GetAvailableModels 获取指定提供商的可用模型列表
func (s *aiServiceImpl) GetAvailableModels(provider models.AIProvider) ([]map[string]interface{}, error) {
	// 创建一个临时配置来获取提供商
	tempConfig := &models.AIAssistantConfig{
		Provider: provider,
		APIKey:   "temp", // 临时密钥，仅用于创建提供商实例
	}

	aiProvider, err := s.providerFactory.CreateProvider(tempConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	models, err := aiProvider.GetAvailableModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %v", err)
	}

	// 转换为map格式以保持向后兼容
	result := make([]map[string]interface{}, 0, len(models))
	for _, model := range models {
		modelMap := map[string]interface{}{
			"id":           model.ID,
			"name":         model.Name,
			"provider":     model.Provider,
			"type":         model.Type,
			"max_tokens":   model.MaxTokens,
			"capabilities": model.Capabilities,
			"description":  fmt.Sprintf("%s - 最大tokens: %d", model.Name, model.MaxTokens),
		}

		// 添加成本信息（如果有）
		if model.Cost.InputTokenPrice > 0 || model.Cost.OutputTokenPrice > 0 {
			modelMap["cost"] = map[string]interface{}{
				"input_token_price":  model.Cost.InputTokenPrice,
				"output_token_price": model.Cost.OutputTokenPrice,
			}
		}

		result = append(result, modelMap)
	}

	return result, nil
}

// GetMessageStatus 获取消息处理状态
func (s *aiServiceImpl) GetMessageStatus(messageID uint, userID uint) (string, interface{}, error) {
	// 这里应该从消息队列服务或缓存中获取状态
	// 暂时返回模拟数据
	return "completed", map[string]interface{}{
		"content":     "这是模拟的AI回复",
		"tokens_used": 150,
	}, nil
}

// CloneConfig 克隆机器人配置
func (s *aiServiceImpl) CloneConfig(id uint, newName string) (*models.AIAssistantConfig, error) {
	original, err := s.GetConfig(id)
	if err != nil {
		return nil, err
	}

	// 创建克隆配置
	clone := *original
	clone.ID = 0 // 重置ID，让数据库生成新的
	clone.Name = newName
	clone.IsDefault = false // 克隆的配置默认不是默认配置
	clone.CreatedAt = time.Time{}
	clone.UpdatedAt = time.Time{}
	clone.DeletedAt = gorm.DeletedAt{}

	if err := s.db.Create(&clone).Error; err != nil {
		return nil, err
	}

	return &clone, nil
}

// BatchUpdateConfigs 批量更新机器人配置
func (s *aiServiceImpl) BatchUpdateConfigs(configIDs []uint, updates map[string]interface{}) error {
	return s.db.Model(&models.AIAssistantConfig{}).
		Where("id IN ?", configIDs).
		Updates(updates).Error
}

// InitDefaultConfigs 初始化默认的OpenAI和Claude配置
func (s *aiServiceImpl) InitDefaultConfigs() error {
	logrus.Info("正在初始化默认AI配置...")

	// 检查是否已存在基础配置
	var totalCount int64
	err := s.db.Model(&models.AIAssistantConfig{}).Count(&totalCount).Error
	if err != nil {
		return fmt.Errorf("检查现有配置失败: %v", err)
	}

	// 检查是否存在默认配置
	var defaultCount int64
	err = s.db.Model(&models.AIAssistantConfig{}).Where("is_default = ?", true).Count(&defaultCount).Error
	if err != nil {
		return fmt.Errorf("检查默认配置失败: %v", err)
	}

	// 如果没有任何配置，创建默认配置
	if totalCount == 0 {
		logrus.Info("数据库中没有AI配置，开始创建默认配置...")

		// 从环境变量读取配置
		openaiAPIKey := os.Getenv("OPENAI_API_KEY")
		openaiBaseURL := os.Getenv("OPENAI_BASE_URL")
		if openaiBaseURL == "" {
			openaiBaseURL = "https://api.openai.com/v1"
		}
		openaiModel := os.Getenv("OPENAI_DEFAULT_MODEL")
		if openaiModel == "" {
			openaiModel = "gpt-4"
		}

		claudeAPIKey := os.Getenv("CLAUDE_API_KEY")
		claudeBaseURL := os.Getenv("CLAUDE_BASE_URL")
		if claudeBaseURL == "" {
			claudeBaseURL = "https://api.anthropic.com"
		}
		claudeModel := os.Getenv("CLAUDE_DEFAULT_MODEL")
		if claudeModel == "" {
			claudeModel = "claude-3-5-sonnet-20241022"
		}

		systemPrompt := os.Getenv("AI_ASSISTANT_DEFAULT_SYSTEM_PROMPT")
		if systemPrompt == "" {
			systemPrompt = "你是一个智能的AI助手，请提供准确、有用的回答。"
		}

		createdConfigs := 0

		// 创建默认的OpenAI配置（如果提供了API密钥）
		if openaiAPIKey != "" {
			openaiConfig := &models.AIAssistantConfig{
				Name:         "默认 OpenAI GPT-4",
				Provider:     models.ProviderOpenAI,
				ModelType:    models.ModelTypeChat,
				APIKey:       openaiAPIKey,
				APIEndpoint:  openaiBaseURL,
				Model:        openaiModel,
				MaxTokens:    4096,
				Temperature:  0.7,
				TopP:         1.0,
				SystemPrompt: systemPrompt,
				IsEnabled:    true,
				IsDefault:    true, // 设为默认配置
				Description:  "默认的OpenAI GPT-4模型配置",
				Category:     "通用对话",
			}

			if err := s.CreateConfig(openaiConfig); err != nil {
				logrus.Errorf("创建OpenAI配置失败: %v", err)
			} else {
				logrus.Info("已创建默认OpenAI配置")
				createdConfigs++
			}
		} else {
			logrus.Warn("未提供OPENAI_API_KEY环境变量，跳过OpenAI配置创建")
		}

		// 创建默认的Claude配置（如果提供了API密钥）
		if claudeAPIKey != "" {
			claudeConfig := &models.AIAssistantConfig{
				Name:         "默认 Claude 3.5 Sonnet",
				Provider:     models.ProviderClaude,
				ModelType:    models.ModelTypeChat,
				APIKey:       claudeAPIKey,
				APIEndpoint:  claudeBaseURL,
				Model:        claudeModel,
				MaxTokens:    4096,
				Temperature:  0.7,
				TopP:         1.0,
				SystemPrompt: "你是Claude，一个由Anthropic开发的AI助手。请提供有帮助、准确和诚实的回答。",
				IsEnabled:    true,
				IsDefault:    (createdConfigs == 0), // 如果没有创建OpenAI配置，则Claude设为默认
				Description:  "默认的Claude 3.5 Sonnet模型配置",
				Category:     "通用对话",
			}

			if err := s.CreateConfig(claudeConfig); err != nil {
				logrus.Errorf("创建Claude配置失败: %v", err)
			} else {
				logrus.Info("已创建默认Claude配置")
				createdConfigs++
			}
		} else {
			logrus.Warn("未提供CLAUDE_API_KEY环境变量，跳过Claude配置创建")
		}

		// 创建其他提供商的配置
		s.createOtherProviderConfigs(&createdConfigs)

		if createdConfigs > 0 {
			logrus.Infof("默认AI配置初始化完成，创建了 %d 个基础配置", createdConfigs)
		} else {
			logrus.Warn("未创建任何AI配置，请检查环境变量设置")
		}
	} else if defaultCount == 0 {
		// 如果有配置但没有默认配置，将第一个启用的配置设为默认
		var firstConfig models.AIAssistantConfig
		err := s.db.Where("is_enabled = ?", true).First(&firstConfig).Error
		if err == nil {
			firstConfig.IsDefault = true
			if updateErr := s.db.Save(&firstConfig).Error; updateErr != nil {
				logrus.Warnf("设置默认配置失败: %v", updateErr)
			} else {
				logrus.Infof("将配置 '%s' 设为默认配置", firstConfig.Name)
			}
		}
	} else {
		logrus.Infof("检测到 %d 个配置，包含 %d 个默认配置，跳过初始化", totalCount, defaultCount)
	}

	return nil
}

// createOtherProviderConfigs 创建其他AI提供商的配置
func (s *aiServiceImpl) createOtherProviderConfigs(createdConfigs *int) {
	// 创建DeepSeek配置
	if deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY"); deepseekAPIKey != "" {
		baseURL := getEnvOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
		
		// 创建 DeepSeek Chat 配置（非思考模式）
		chatModel := getEnvOrDefault("DEEPSEEK_CHAT_MODEL", "deepseek-chat")
		deepseekChatConfig := &models.AIAssistantConfig{
			Name:         "DeepSeek-V3.2-Exp (Chat)",
			Provider:     models.ProviderDeepSeek,
			ModelType:    models.ModelTypeChat,
			APIKey:       deepseekAPIKey,
			APIEndpoint:  baseURL,
			Model:        chatModel,
			MaxTokens:    8192,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是DeepSeek助手，基于DeepSeek-V3.2-Exp模型。请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "DeepSeek-V3.2-Exp 非思考模式，适合快速对话和一般任务",
			Category:     "通用对话",
		}

		if err := s.CreateConfig(deepseekChatConfig); err != nil {
			logrus.Errorf("创建DeepSeek Chat配置失败: %v", err)
		} else {
			logrus.Info("已创建DeepSeek Chat (V3.2-Exp) 配置")
			*createdConfigs++
		}

		// 创建 DeepSeek Reasoner 配置（思考模式）
		reasonerModel := getEnvOrDefault("DEEPSEEK_REASONER_MODEL", "deepseek-reasoner")
		deepseekReasonerConfig := &models.AIAssistantConfig{
			Name:         "DeepSeek-V3.2-Exp (Reasoner)",
			Provider:     models.ProviderDeepSeek,
			ModelType:    models.ModelTypeChat,
			APIKey:       deepseekAPIKey,
			APIEndpoint:  baseURL,
			Model:        reasonerModel,
			MaxTokens:    8192,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是DeepSeek推理助手，基于DeepSeek-V3.2-Exp模型的思考模式。你会深入分析问题并提供详细的推理过程。",
			IsEnabled:    true,
			IsDefault:    false,
			Description:  "DeepSeek-V3.2-Exp 思考模式，适合复杂推理、数学问题和深度分析",
			Category:     "深度推理",
		}

		if err := s.CreateConfig(deepseekReasonerConfig); err != nil {
			logrus.Errorf("创建DeepSeek Reasoner配置失败: %v", err)
		} else {
			logrus.Info("已创建DeepSeek Reasoner (V3.2-Exp) 配置")
		}
	}

	// 创建GLM配置
	if glmAPIKey := os.Getenv("GLM_API_KEY"); glmAPIKey != "" {
		glmConfig := &models.AIAssistantConfig{
			Name:         "默认 GLM-4",
			Provider:     models.ProviderCustom,
			ModelType:    models.ModelTypeChat,
			APIKey:       glmAPIKey,
			APIEndpoint:  os.Getenv("GLM_BASE_URL"),
			Model:        getEnvOrDefault("GLM_DEFAULT_MODEL", "glm-4"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是智谱AI的GLM助手，请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "默认的智谱AI GLM-4模型配置",
			Category:     "通用对话",
		}

		if err := s.CreateConfig(glmConfig); err != nil {
			logrus.Errorf("创建GLM配置失败: %v", err)
		} else {
			logrus.Info("已创建默认GLM配置")
			*createdConfigs++
		}
	}

	// 创建通义千问配置
	if qwenAPIKey := os.Getenv("QWEN_API_KEY"); qwenAPIKey != "" {
		qwenConfig := &models.AIAssistantConfig{
			Name:         "默认 通义千问",
			Provider:     models.ProviderCustom,
			ModelType:    models.ModelTypeChat,
			APIKey:       qwenAPIKey,
			APIEndpoint:  os.Getenv("QWEN_BASE_URL"),
			Model:        getEnvOrDefault("QWEN_DEFAULT_MODEL", "qwen-turbo"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是通义千问助手，请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "默认的阿里云通义千问模型配置",
			Category:     "通用对话",
		}

		if err := s.CreateConfig(qwenConfig); err != nil {
			logrus.Errorf("创建通义千问配置失败: %v", err)
		} else {
			logrus.Info("已创建默认通义千问配置")
			*createdConfigs++
		}
	}

	// 创建本地AI配置
	if localAIEnabled := os.Getenv("LOCAL_AI_ENABLED"); localAIEnabled == "true" {
		localConfig := &models.AIAssistantConfig{
			Name:         "本地 AI 模型",
			Provider:     models.ProviderLocal,
			ModelType:    models.ModelTypeChat,
			APIEndpoint:  getEnvOrDefault("LOCAL_AI_BASE_URL", "http://localhost:8080/v1"),
			Model:        getEnvOrDefault("LOCAL_AI_DEFAULT_MODEL", "llama2"),
			MaxTokens:    4096,
			Temperature:  0.7,
			TopP:         1.0,
			SystemPrompt: "你是一个本地部署的AI助手，请提供准确、有用的回答。",
			IsEnabled:    true,
			IsDefault:    (*createdConfigs == 0),
			Description:  "本地部署的AI模型配置",
			Category:     "通用对话",
		}

		if err := s.CreateConfig(localConfig); err != nil {
			logrus.Errorf("创建本地AI配置失败: %v", err)
		} else {
			logrus.Info("已创建默认本地AI配置")
			*createdConfigs++
		}
	}
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CompareModels 对比多个模型的响应
func (s *aiServiceImpl) CompareModels(conversationID uint, userMessage string, configIDs []uint) (*ModelComparisonResult, error) {
	if len(configIDs) == 0 {
		return nil, fmt.Errorf("至少需要选择一个模型进行对比")
	}

	if len(configIDs) > 5 {
		return nil, fmt.Errorf("最多同时对比5个模型")
	}

	// 验证会话存在
	conversation, err := s.GetConversation(conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %v", err)
	}

	// 生成对比任务ID
	messageID := fmt.Sprintf("compare_%d_%d", conversationID, time.Now().UnixNano())
	
	logrus.WithFields(logrus.Fields{
		"message_id":       messageID,
		"conversation_id":  conversationID,
		"config_count":     len(configIDs),
	}).Info("开始模型对比")

	// 使用 channel 和 waitgroup 并发调用多个模型
	type modelResult struct {
		index    int
		response *ModelComparisonResponse
		err      error
	}

	resultChan := make(chan modelResult, len(configIDs))
	var wg sync.WaitGroup

	// 并发调用各个模型
	for i, configID := range configIDs {
		wg.Add(1)
		go func(index int, id uint) {
			defer wg.Done()

			result := &ModelComparisonResponse{
				ConfigID: id,
				Status:   "processing",
			}

			// 获取配置
			config, err := s.GetConfig(id)
			if err != nil {
				result.Status = "error"
				result.Error = fmt.Sprintf("获取配置失败: %v", err)
				resultChan <- modelResult{index: index, response: result, err: err}
				return
			}

			result.ConfigName = config.Name
			result.Provider = string(config.Provider)
			result.Model = config.Model

			// 安全解密API密钥
			if config.APIKey != "" && config.APIKey != "***" {
				if s.cryptoService.IsEncrypted(config.APIKey) {
					config.APIKey = s.cryptoService.DecryptSafely(config.APIKey)
				}
			}

			// 创建AI提供商
			provider, err := s.providerFactory.CreateProvider(config)
			if err != nil {
				result.Status = "error"
				result.Error = fmt.Sprintf("创建AI提供商失败: %v", err)
				resultChan <- modelResult{index: index, response: result, err: err}
				return
			}

			// 获取历史消息（用于上下文）
			var historyMessages []models.AIMessage
			s.db.Where("conversation_id = ?", conversationID).
				Order("created_at ASC").
				Limit(10). // 限制上下文长度
				Find(&historyMessages)

			// 构建消息历史
			chatMessages := make([]ai_providers.ChatMessage, 0, len(historyMessages)+1)
			for _, msg := range historyMessages {
				chatMessages = append(chatMessages, ai_providers.ChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}

			// 添加当前用户消息
			chatMessages = append(chatMessages, ai_providers.ChatMessage{
				Role:    "user",
				Content: userMessage,
			})

			// 构建请求
			request := ai_providers.ChatRequest{
				Model:        config.Model,
				Messages:     chatMessages,
				MaxTokens:    config.MaxTokens,
				Temperature:  config.Temperature,
				SystemPrompt: config.SystemPrompt,
			}

			// 调用AI API
			ctx := context.Background()
			startTime := time.Now()
			response, err := provider.Chat(ctx, request)
			responseTime := int(time.Since(startTime).Milliseconds())

			if err != nil {
				result.Status = "error"
				result.Error = fmt.Sprintf("AI API调用失败: %v", err)
				result.ResponseTime = responseTime
				resultChan <- modelResult{index: index, response: result, err: err}
				
				// 记录失败统计
				s.RecordUsage(conversation.UserID, config.ID, 0, responseTime, false)
				return
			}

			// 成功响应
			result.Status = "success"
			result.Content = response.Content
			result.TokensUsed = response.TokensUsed
			result.ResponseTime = response.ResponseTime
			result.Metadata = response.Metadata

			// 记录成功统计
			s.RecordUsage(conversation.UserID, config.ID, response.TokensUsed, responseTime, true)

			resultChan <- modelResult{index: index, response: result, err: nil}

		}(i, configID)
	}

	// 等待所有协程完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]*ModelComparisonResponse, len(configIDs))
	for result := range resultChan {
		results[result.index] = result.response
	}

	// 统计成功和失败的模型数
	successCount := 0
	errorCount := 0
	for _, result := range results {
		if result.Status == "success" {
			successCount++
		} else {
			errorCount++
		}
	}

	logrus.WithFields(logrus.Fields{
		"message_id":     messageID,
		"success_count":  successCount,
		"error_count":    errorCount,
		"total_count":    len(configIDs),
	}).Info("模型对比完成")

	// 构建对比结果（去掉指针）
	finalResults := make([]ModelComparisonResponse, len(results))
	for i, r := range results {
		if r != nil {
			finalResults[i] = *r
		}
	}

	comparisonResult := &ModelComparisonResult{
		MessageID:   messageID,
		UserMessage: userMessage,
		Results:     finalResults,
		CreatedAt:   time.Now(),
	}

	return comparisonResult, nil
}
