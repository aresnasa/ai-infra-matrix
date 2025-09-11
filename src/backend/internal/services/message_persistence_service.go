package services

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MessagePersistenceService 消息持久化服务
type MessagePersistenceService struct {
	db               *gorm.DB
	kafkaService     *KafkaMessageService
	redisService     *messageQueueServiceImpl
	messageQueue     MessageQueueService
}

// NewMessagePersistenceService 创建消息持久化服务
func NewMessagePersistenceService(db *gorm.DB, kafkaService *KafkaMessageService, messageQueue MessageQueueService) *MessagePersistenceService {
	return &MessagePersistenceService{
		db:           db,
		kafkaService: kafkaService,
		messageQueue: messageQueue,
	}
}

// SaveMessage 保存消息到数据库
func (s *MessagePersistenceService) SaveMessage(message *models.AIMessage) error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 保存消息
	if err := tx.Create(message).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save message: %v", err)
	}

	// 更新对话的token使用量
	if err := tx.Model(&models.AIConversation{}).
		Where("id = ?", message.ConversationID).
		Update("tokens_used", gorm.Expr("tokens_used + ?", message.TokensUsed)).
		Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update conversation tokens: %v", err)
	}

	// 更新对话的最后活动时间
	if err := tx.Model(&models.AIConversation{}).
		Where("id = ?", message.ConversationID).
		Update("updated_at", time.Now()).
		Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update conversation timestamp: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// 发送到Kafka进行缓存和流处理
	if s.kafkaService != nil {
		if err := s.kafkaService.CacheMessage(message.ConversationID, message); err != nil {
			logrus.Warnf("Failed to send message to Kafka: %v", err)
			// 不影响主流程
		}
	}

	logrus.Infof("Message saved successfully: conversation_id=%d, message_id=%d", message.ConversationID, message.ID)
	return nil
}

// BatchSaveMessages 批量保存消息
func (s *MessagePersistenceService) BatchSaveMessages(messages []*models.AIMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 批量保存消息
	if err := tx.CreateInBatches(messages, 100).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to batch save messages: %v", err)
	}

	// 更新对话统计信息
	conversationStats := make(map[uint]int)
	for _, msg := range messages {
		conversationStats[msg.ConversationID] += msg.TokensUsed
	}

	for conversationID, totalTokens := range conversationStats {
		if err := tx.Model(&models.AIConversation{}).
			Where("id = ?", conversationID).
			Updates(map[string]interface{}{
				"tokens_used": gorm.Expr("tokens_used + ?", totalTokens),
				"updated_at":  time.Now(),
			}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update conversation stats: %v", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit batch transaction: %v", err)
	}

	// 发送到Kafka进行流处理
	if s.kafkaService != nil {
		for _, message := range messages {
			if err := s.kafkaService.CacheMessage(message.ConversationID, message); err != nil {
				logrus.Warnf("Failed to send batch message to Kafka: %v", err)
			}
		}
	}

	logrus.Infof("Batch saved %d messages successfully", len(messages))
	return nil
}

// GetMessageHistory 获取消息历史记录
func (s *MessagePersistenceService) GetMessageHistory(conversationID uint, limit int, offset int) ([]*models.AIMessage, error) {
	var messages []*models.AIMessage

	query := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get message history: %v", err)
	}

	// 反转消息顺序（从旧到新）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SearchMessages 搜索消息
func (s *MessagePersistenceService) SearchMessages(conversationID uint, keyword string, limit int) ([]*models.AIMessage, error) {
	var messages []*models.AIMessage

	query := s.db.Where("conversation_id = ? AND content LIKE ?", conversationID, "%"+keyword+"%").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to search messages: %v", err)
	}

	return messages, nil
}

// DeleteMessage 删除消息
func (s *MessagePersistenceService) DeleteMessage(messageID uint, userID uint) error {
	// 检查消息所有权
	var message models.AIMessage
	if err := s.db.Where("id = ?", messageID).First(&message).Error; err != nil {
		return fmt.Errorf("message not found: %v", err)
	}

	// 获取对话信息
	var conversation models.AIConversation
	if err := s.db.Where("id = ?", message.ConversationID).First(&conversation).Error; err != nil {
		return fmt.Errorf("conversation not found: %v", err)
	}

	// 检查权限
	if conversation.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	// 软删除消息
	if err := s.db.Delete(&message).Error; err != nil {
		return fmt.Errorf("failed to delete message: %v", err)
	}

	// 更新对话的token使用量
	if err := s.db.Model(&models.AIConversation{}).
		Where("id = ?", message.ConversationID).
		Update("tokens_used", gorm.Expr("tokens_used - ?", message.TokensUsed)).
		Error; err != nil {
		logrus.Warnf("Failed to update conversation tokens after message deletion: %v", err)
	}

	logrus.Infof("Message deleted successfully: message_id=%d", messageID)
	return nil
}

// ArchiveOldMessages 归档旧消息
func (s *MessagePersistenceService) ArchiveOldMessages(daysOld int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysOld)

	// 将旧消息标记为已归档（可以考虑移动到归档表）
	result := s.db.Model(&models.AIMessage{}).
		Where("created_at < ? AND deleted_at IS NULL", cutoffDate).
		Update("metadata", gorm.Expr("JSON_SET(COALESCE(metadata, '{}'), '$.archived', true)"))

	if result.Error != nil {
		return fmt.Errorf("failed to archive old messages: %v", result.Error)
	}

	logrus.Infof("Archived %d old messages", result.RowsAffected)
	return nil
}

// GetMessageStats 获取消息统计信息
func (s *MessagePersistenceService) GetMessageStats(userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	var stats struct {
		TotalMessages    int64 `json:"total_messages"`
		TotalTokens      int64 `json:"total_tokens"`
		ActiveConversations int64 `json:"active_conversations"`
		AvgResponseTime  float64 `json:"avg_response_time"`
	}

	// 统计消息数量和tokens
	s.db.Model(&models.AIMessage{}).
		Joins("JOIN ai_conversations ON ai_messages.conversation_id = ai_conversations.id").
		Where("ai_conversations.user_id = ? AND ai_messages.created_at BETWEEN ? AND ?", userID, startDate, endDate).
		Select("COUNT(*) as total_messages, COALESCE(SUM(tokens_used), 0) as total_tokens, AVG(response_time) as avg_response_time").
		Scan(&stats)

	// 统计活跃对话数
	s.db.Model(&models.AIConversation{}).
		Where("user_id = ? AND updated_at BETWEEN ? AND ?", userID, startDate, endDate).
		Count(&stats.ActiveConversations)

	return map[string]interface{}{
		"total_messages":       stats.TotalMessages,
		"total_tokens":         stats.TotalTokens,
		"active_conversations": stats.ActiveConversations,
		"avg_response_time":    stats.AvgResponseTime,
	}, nil
}

// CleanupOrphanedMessages 清理孤立消息
func (s *MessagePersistenceService) CleanupOrphanedMessages() error {
	// 删除没有对应对话的消息
	result := s.db.Where("conversation_id NOT IN (SELECT id FROM ai_conversations)").Delete(&models.AIMessage{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup orphaned messages: %v", result.Error)
	}

	if result.RowsAffected > 0 {
		logrus.Infof("Cleaned up %d orphaned messages", result.RowsAffected)
	}

	return nil
}
