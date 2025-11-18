package services

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// MessageRetrievalService 消息检索服务
type MessageRetrievalService struct {
	persistenceService *MessagePersistenceService
	kafkaService       *KafkaMessageService
	cacheService       CacheService
	messageQueue       MessageQueueService
}

// NewMessageRetrievalService 创建消息检索服务
func NewMessageRetrievalService(
	persistenceService *MessagePersistenceService,
	kafkaService *KafkaMessageService,
	cacheService CacheService,
	messageQueue MessageQueueService,
) *MessageRetrievalService {
	return &MessageRetrievalService{
		persistenceService: persistenceService,
		kafkaService:       kafkaService,
		cacheService:       cacheService,
		messageQueue:       messageQueue,
	}
}

// GetMessagesWithOptimization 优化的消息获取
func (s *MessageRetrievalService) GetMessagesWithOptimization(conversationID uint, userID uint, options *MessageQueryOptions) ([]*models.AIMessage, error) {
	// 验证权限
	if err := s.validateConversationAccess(conversationID, userID); err != nil {
		return nil, err
	}

	// ===== 临时禁用缓存读取，确保实时性 =====
	// 在快速连续的消息处理场景下，缓存可能导致读取到旧数据
	// 直接从数据库获取最新消息，确保数据一致性
	logrus.Debugf("Fetching messages from database for conversation %d (cache disabled for real-time consistency)", conversationID)

	// 从数据库获取
	messages, err := s.getFromDatabase(conversationID, options)
	if err != nil {
		return nil, err
	}

	// 仍然缓存结果（为了兼容性和未来可能的优化）
	// 但读取时不使用缓存
	cacheKey := s.buildCacheKey(conversationID, options)
	s.cacheMessages(cacheKey, messages, options)

	// 发送到Kafka进行流处理
	if s.kafkaService != nil {
		s.sendRetrievalEvent(conversationID, userID, len(messages))
	}

	return messages, nil
}

// MessageQueryOptions 消息查询选项
type MessageQueryOptions struct {
	Limit           int        `json:"limit,omitempty"`
	Offset          int        `json:"offset,omitempty"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	Keyword         string     `json:"keyword,omitempty"`
	Role            string     `json:"role,omitempty"`       // user, assistant, system
	SortBy          string     `json:"sort_by,omitempty"`    // created_at, tokens_used
	SortOrder       string     `json:"sort_order,omitempty"` // asc, desc
	IncludeMetadata bool       `json:"include_metadata,omitempty"`
}

// GetMessagesByTimeRange 按时间范围获取消息
func (s *MessageRetrievalService) GetMessagesByTimeRange(conversationID uint, userID uint, startTime, endTime time.Time) ([]*models.AIMessage, error) {
	if err := s.validateConversationAccess(conversationID, userID); err != nil {
		return nil, err
	}

	options := &MessageQueryOptions{
		StartDate: &startTime,
		EndDate:   &endTime,
		SortBy:    "created_at",
		SortOrder: "asc",
	}

	return s.GetMessagesWithOptimization(conversationID, userID, options)
}

// SearchMessagesInConversation 在对话中搜索消息
func (s *MessageRetrievalService) SearchMessagesInConversation(conversationID uint, userID uint, keyword string, limit int) ([]*models.AIMessage, error) {
	if err := s.validateConversationAccess(conversationID, userID); err != nil {
		return nil, err
	}

	// 使用数据库搜索
	messages, err := s.persistenceService.SearchMessages(conversationID, keyword, limit)
	if err != nil {
		return nil, err
	}

	// 发送搜索事件到Kafka
	if s.kafkaService != nil {
		s.sendSearchEvent(conversationID, userID, keyword, len(messages))
	}

	return messages, nil
}

// GetRecentMessages 获取最近的消息
func (s *MessageRetrievalService) GetRecentMessages(conversationID uint, userID uint, limit int) ([]*models.AIMessage, error) {
	options := &MessageQueryOptions{
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	return s.GetMessagesWithOptimization(conversationID, userID, options)
}

// GetMessageByID 根据ID获取消息
func (s *MessageRetrievalService) GetMessageByID(messageID uint, userID uint) (*models.AIMessage, error) {
	// 先从缓存获取
	cacheKey := fmt.Sprintf("message:%d", messageID)
	if cachedMessage := s.cacheService.GetMessage(cacheKey); cachedMessage != nil {
		// 验证权限
		if err := s.validateMessageAccess(cachedMessage, userID); err != nil {
			return nil, err
		}
		return cachedMessage, nil
	}

	// 从数据库获取
	var message models.AIMessage
	if err := s.persistenceService.db.Where("id = ?", messageID).First(&message).Error; err != nil {
		return nil, fmt.Errorf("message not found: %v", err)
	}

	// 验证权限
	if err := s.validateMessageAccess(&message, userID); err != nil {
		return nil, err
	}

	// 缓存消息
	s.cacheService.SetMessage(cacheKey, &message, 1*time.Hour)

	return &message, nil
}

// StreamMessages 流式获取消息（用于实时更新）
func (s *MessageRetrievalService) StreamMessages(conversationID uint, userID uint, lastMessageID uint) (<-chan *models.AIMessage, error) {
	if err := s.validateConversationAccess(conversationID, userID); err != nil {
		return nil, err
	}

	messageChan := make(chan *models.AIMessage, 100)

	go func() {
		defer close(messageChan)

		// 获取自上次消息ID之后的新消息
		var messages []*models.AIMessage
		query := s.persistenceService.db.Where("conversation_id = ? AND id > ?", conversationID, lastMessageID).
			Order("created_at ASC")

		if err := query.Find(&messages).Error; err != nil {
			logrus.Errorf("Failed to stream messages: %v", err)
			return
		}

		// 发送消息到通道
		for _, message := range messages {
			select {
			case messageChan <- message:
			case <-time.After(30 * time.Second):
				logrus.Warn("Message streaming timeout")
				return
			}
		}
	}()

	return messageChan, nil
}

// PreloadMessages 预加载消息到缓存
func (s *MessageRetrievalService) PreloadMessages(conversationID uint, userID uint) error {
	if err := s.validateConversationAccess(conversationID, userID); err != nil {
		return err
	}

	// 获取最近的消息
	messages, err := s.GetRecentMessages(conversationID, userID, 50)
	if err != nil {
		return err
	}

	// 预热缓存
	cacheKey := s.buildCacheKey(conversationID, &MessageQueryOptions{Limit: 50, SortBy: "created_at", SortOrder: "desc"})
	s.cacheMessages(cacheKey, messages, &MessageQueryOptions{Limit: 50})

	logrus.Infof("Preloaded %d messages for conversation %d", len(messages), conversationID)
	return nil
}

// InvalidateCache 使缓存失效
func (s *MessageRetrievalService) InvalidateCache(conversationID uint) {
	cacheKey := fmt.Sprintf("messages:%d", conversationID)
	s.cacheService.Delete(cacheKey)

	// 发送缓存失效事件到Kafka
	if s.kafkaService != nil {
		s.sendCacheInvalidationEvent(conversationID)
	}

	logrus.Infof("Invalidated cache for conversation %d", conversationID)
}

// 私有方法

func (s *MessageRetrievalService) validateConversationAccess(conversationID uint, userID uint) error {
	// 这里应该检查用户是否有权限访问对话
	// 为了简化，这里只是检查对话是否存在
	var conversation models.AIConversation
	if err := s.persistenceService.db.Where("id = ? AND user_id = ?", conversationID, userID).First(&conversation).Error; err != nil {
		return fmt.Errorf("conversation not found or access denied: %v", err)
	}
	return nil
}

func (s *MessageRetrievalService) validateMessageAccess(message *models.AIMessage, userID uint) error {
	return s.validateConversationAccess(message.ConversationID, userID)
}

func (s *MessageRetrievalService) buildCacheKey(conversationID uint, options *MessageQueryOptions) string {
	key := fmt.Sprintf("messages:%d", conversationID)
	if options != nil {
		if options.Limit > 0 {
			key += fmt.Sprintf(":limit_%d", options.Limit)
		}
		if options.Offset > 0 {
			key += fmt.Sprintf(":offset_%d", options.Offset)
		}
		if options.Keyword != "" {
			key += fmt.Sprintf(":keyword_%s", options.Keyword)
		}
		if options.SortBy != "" {
			key += fmt.Sprintf(":sort_%s_%s", options.SortBy, options.SortOrder)
		}
	}
	return key
}

func (s *MessageRetrievalService) getFromCache(cacheKey string) []*models.AIMessage {
	messages := s.cacheService.GetMessages(cacheKey)
	if messages == nil {
		return nil
	}

	// 转换 []models.AIMessage 到 []*models.AIMessage
	result := make([]*models.AIMessage, len(messages))
	for i := range messages {
		result[i] = &messages[i]
	}
	return result
}

func (s *MessageRetrievalService) getFromDatabase(conversationID uint, options *MessageQueryOptions) ([]*models.AIMessage, error) {
	return s.persistenceService.GetMessageHistory(conversationID, options.Limit, options.Offset)
}

func (s *MessageRetrievalService) cacheMessages(cacheKey string, messages []*models.AIMessage, options *MessageQueryOptions) {
	// 根据查询类型设置不同的缓存时间
	var cacheDuration time.Duration
	if options != nil && options.Keyword != "" {
		// 搜索结果缓存时间较短
		cacheDuration = 15 * time.Minute
	} else {
		// 普通查询缓存时间较长
		cacheDuration = 1 * time.Hour
	}

	// 转换 []*models.AIMessage 到 []models.AIMessage
	messageValues := make([]models.AIMessage, len(messages))
	for i, msg := range messages {
		messageValues[i] = *msg
	}

	s.cacheService.SetMessages(cacheKey, messageValues, cacheDuration)
}

func (s *MessageRetrievalService) sendRetrievalEvent(conversationID uint, userID uint, messageCount int) {
	eventData := map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         userID,
		"message_count":   messageCount,
		"timestamp":       time.Now(),
	}

	if err := s.kafkaService.SendMessageEvent("message_retrieved", fmt.Sprintf("conv_%d", conversationID), userID, eventData); err != nil {
		logrus.Warnf("Failed to send retrieval event: %v", err)
	}
}

func (s *MessageRetrievalService) sendSearchEvent(conversationID uint, userID uint, keyword string, resultCount int) {
	eventData := map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         userID,
		"keyword":         keyword,
		"result_count":    resultCount,
		"timestamp":       time.Now(),
	}

	if err := s.kafkaService.SendMessageEvent("message_searched", fmt.Sprintf("search_%d", conversationID), userID, eventData); err != nil {
		logrus.Warnf("Failed to send search event: %v", err)
	}
}

func (s *MessageRetrievalService) sendCacheInvalidationEvent(conversationID uint) {
	eventData := map[string]interface{}{
		"conversation_id": conversationID,
		"timestamp":       time.Now(),
	}

	if err := s.kafkaService.SendMessageEvent("cache_invalidated", fmt.Sprintf("cache_%d", conversationID), 0, eventData); err != nil {
		logrus.Warnf("Failed to send cache invalidation event: %v", err)
	}
}
