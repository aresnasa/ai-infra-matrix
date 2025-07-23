package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
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

	// 对话管理
	CreateConversation(userID uint, configID uint, title string, context string) (*models.AIConversation, error)
	GetConversation(id uint) (*models.AIConversation, error)
	ListUserConversations(userID uint) ([]models.AIConversation, error)
	UpdateConversation(conversation *models.AIConversation) error
	DeleteConversation(id uint) error

	// 消息处理
	SendMessage(conversationID uint, userMessage string) (*models.AIMessage, error)
	GetMessages(conversationID uint) ([]models.AIMessage, error)

	// 统计
	GetUsageStats(userID uint, startDate, endDate time.Time) ([]models.AIUsageStats, error)
	RecordUsage(userID, configID uint, tokenUsed, responseTime int, success bool) error
}

// aiServiceImpl AI服务实现
type aiServiceImpl struct {
	db *gorm.DB
	cryptoService *utils.CryptoService
}

// NewAIService 创建AI服务实例
func NewAIService() AIService {
	return &aiServiceImpl{
		db: database.DB,
		cryptoService: database.CryptoService,
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

	// 解密API密钥
	if config.APIKey != "" {
		decryptedKey, err := s.cryptoService.Decrypt(config.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %v", err)
		}
		config.APIKey = decryptedKey
	}

	return &config, nil
}

func (s *aiServiceImpl) GetDefaultConfig() (*models.AIAssistantConfig, error) {
	var config models.AIAssistantConfig
	err := s.db.Where("is_default = ? AND is_enabled = ?", true, true).First(&config).Error
	if err != nil {
		return nil, err
	}

	// 解密API密钥
	if config.APIKey != "" {
		decryptedKey, err := s.cryptoService.Decrypt(config.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %v", err)
		}
		config.APIKey = decryptedKey
	}

	return &config, nil
}

func (s *aiServiceImpl) UpdateConfig(config *models.AIAssistantConfig) error {
	// 加密API密钥
	if config.APIKey != "" {
		encryptedKey, err := s.cryptoService.Encrypt(config.APIKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %v", err)
		}
		config.APIKey = encryptedKey
	}

	return s.db.Save(config).Error
}

func (s *aiServiceImpl) DeleteConfig(id uint) error {
	return s.db.Delete(&models.AIAssistantConfig{}, id).Error
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

	// 保存用户消息
	userMsg := &models.AIMessage{
		ConversationID: conversationID,
		Role:           "user",
		Content:        userMessage,
	}

	if err := s.db.Create(userMsg).Error; err != nil {
		return nil, err
	}

	// 调用AI API
	startTime := time.Now()
	aiResponse, tokensUsed, err := s.callAIAPI(config, conversation, userMessage)
	responseTime := int(time.Since(startTime).Milliseconds())

	// 记录使用统计
	s.RecordUsage(conversation.UserID, config.ID, tokensUsed, responseTime, err == nil)

	if err != nil {
		return nil, err
	}

	// 保存AI回复
	aiMsg := &models.AIMessage{
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        aiResponse,
		TokensUsed:     tokensUsed,
		ResponseTime:   responseTime,
	}

	if err := s.db.Create(aiMsg).Error; err != nil {
		return nil, err
	}

	// 更新对话的tokens使用量
	conversation.TokensUsed += tokensUsed
	s.db.Save(conversation)

	return aiMsg, nil
}

func (s *aiServiceImpl) GetMessages(conversationID uint) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}

// callAIAPI 调用AI API
func (s *aiServiceImpl) callAIAPI(config *models.AIAssistantConfig, conversation *models.AIConversation, userMessage string) (string, int, error) {
	switch config.Provider {
	case models.ProviderOpenAI:
		return s.callOpenAI(config, conversation, userMessage)
	case models.ProviderClaude:
		return s.callClaude(config, conversation, userMessage)
	case models.ProviderMCP:
		return s.callMCP(config, conversation, userMessage)
	default:
		return "", 0, fmt.Errorf("unsupported AI provider: %s", config.Provider)
	}
}

// callOpenAI 调用OpenAI API
func (s *aiServiceImpl) callOpenAI(config *models.AIAssistantConfig, conversation *models.AIConversation, userMessage string) (string, int, error) {
	// 构建消息历史
	messages := []map[string]string{
		{"role": "system", "content": config.SystemPrompt},
	}

	// 添加历史消息（最近的20条）
	var historyMessages []models.AIMessage
	s.db.Where("conversation_id = ?", conversation.ID).
		Order("created_at DESC").
		Limit(20).
		Find(&historyMessages)

	for i := len(historyMessages) - 1; i >= 0; i-- {
		msg := historyMessages[i]
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// 添加当前用户消息
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userMessage,
	})

	// 构建请求体
	requestBody := map[string]interface{}{
		"model":       config.Model,
		"messages":    messages,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", 0, err
	}

	// 发送请求
	req, err := http.NewRequest("POST", config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, err
	}

	if len(response.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from AI")
	}

	return response.Choices[0].Message.Content, response.Usage.TotalTokens, nil
}

// callClaude 调用Claude API
func (s *aiServiceImpl) callClaude(config *models.AIAssistantConfig, conversation *models.AIConversation, userMessage string) (string, int, error) {
	// 构建消息历史
	messages := []map[string]string{}

	// 添加历史消息（最近的20条）
	var historyMessages []models.AIMessage
	s.db.Where("conversation_id = ?", conversation.ID).
		Order("created_at DESC").
		Limit(20).
		Find(&historyMessages)

	for i := len(historyMessages) - 1; i >= 0; i-- {
		msg := historyMessages[i]
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// 添加当前用户消息
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userMessage,
	})

	// 构建请求体
	requestBody := map[string]interface{}{
		"model":       config.Model,
		"max_tokens":  config.MaxTokens,
		"messages":    messages,
		"system":      config.SystemPrompt,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", 0, err
	}

	// 发送请求
	req, err := http.NewRequest("POST", config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, err
	}

	if len(response.Content) == 0 {
		return "", 0, fmt.Errorf("no response from AI")
	}

	totalTokens := response.Usage.InputTokens + response.Usage.OutputTokens
	return response.Content[0].Text, totalTokens, nil
}

// callMCP 调用MCP协议
func (s *aiServiceImpl) callMCP(config *models.AIAssistantConfig, conversation *models.AIConversation, userMessage string) (string, int, error) {
	// MCP协议实现（预留接口）
	// 这里可以实现Model Context Protocol的具体逻辑
	return "MCP功能正在开发中...", 0, nil
}

// 统计功能
func (s *aiServiceImpl) GetUsageStats(userID uint, startDate, endDate time.Time) ([]models.AIUsageStats, error) {
	var stats []models.AIUsageStats
	err := s.db.Where("user_id = ? AND date BETWEEN ? AND ?", userID, startDate, endDate).
		Find(&stats).Error

	return stats, err
}

func (s *aiServiceImpl) RecordUsage(userID, configID uint, tokensUsed, responseTime int, success bool) error {
	today := time.Now().Truncate(24 * time.Hour)

	var stat models.AIUsageStats
	err := s.db.Where("user_id = ? AND config_id = ? AND date = ?", userID, configID, today).
		First(&stat).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		stat = models.AIUsageStats{
			UserID:          userID,
			ConfigID:        configID,
			Date:            today,
			RequestCount:    1,
			TokensUsed:      tokensUsed,
			SuccessCount:    0,
			ErrorCount:      0,
			AverageResponse: responseTime,
		}

		if success {
			stat.SuccessCount = 1
		} else {
			stat.ErrorCount = 1
		}

		return s.db.Create(&stat).Error
	} else if err != nil {
		return err
	} else {
		// 更新现有记录
		stat.RequestCount++
		stat.TokensUsed += tokensUsed

		if success {
			stat.SuccessCount++
		} else {
			stat.ErrorCount++
		}

		// 计算平均响应时间
		stat.AverageResponse = (stat.AverageResponse*(stat.RequestCount-1) + responseTime) / stat.RequestCount

		return s.db.Save(&stat).Error
	}
}