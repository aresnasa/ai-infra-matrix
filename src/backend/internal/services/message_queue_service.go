package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeChatRequest    MessageType = "chat_request"
	MessageTypeClusterOp      MessageType = "cluster_operation"
	MessageTypeNotification   MessageType = "notification"
	MessageTypeStatusUpdate   MessageType = "status_update"
)

// Message 消息结构
type Message struct {
	ID             string                 `json:"id"`
	Type           MessageType            `json:"type"`
	UserID         uint                   `json:"user_id"`
	ConversationID *uint                  `json:"conversation_id,omitempty"`
	Content        string                 `json:"content"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Priority       string                 `json:"priority"`
	Timestamp      time.Time              `json:"timestamp"`
	RetryCount     int                    `json:"retry_count"`
	MaxRetries     int                    `json:"max_retries"`
}

// MessageStatus 消息状态
type MessageStatus struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"` // pending, processing, completed, failed
	Result      string    `json:"result,omitempty"`
	Error       string    `json:"error,omitempty"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// MessageQueueService 消息队列服务接口
type MessageQueueService interface {
	// 消息发送
	SendMessage(streamName string, message *Message) error
	SendChatRequest(userID uint, conversationID *uint, content string, context map[string]interface{}) (string, error)
	SendClusterOperation(userID uint, operation string, params map[string]interface{}) (string, error)
	
	// 消息消费
	StartConsumer(streamName, consumerGroup, consumerName string, handler MessageHandler) error
	StopConsumer(streamName, consumerGroup string) error
	
	// 状态管理
	SetMessageStatus(messageID string, status *MessageStatus) error
	GetMessageStatus(messageID string) (*MessageStatus, error)
	
	// 健康检查
	HealthCheck() error
}

// MessageHandler 消息处理器
type MessageHandler func(message *Message) error

// messageQueueServiceImpl 消息队列服务实现
type messageQueueServiceImpl struct {
	redis    *redis.Client
	ctx      context.Context
	stopChan chan struct{}
}

// NewMessageQueueService 创建消息队列服务
func NewMessageQueueService() MessageQueueService {
	return &messageQueueServiceImpl{
		redis:    cache.RDB,
		ctx:      context.Background(),
		stopChan: make(chan struct{}),
	}
}

// SendMessage 发送消息到指定流
func (s *messageQueueServiceImpl) SendMessage(streamName string, message *Message) error {
	// 序列化消息
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// 发送到Redis Stream
	args := &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data": messageData,
		},
	}

	_, err = s.redis.XAdd(s.ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to send message to stream %s: %v", streamName, err)
	}

	// 设置初始状态
	status := &MessageStatus{
		ID:        message.ID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	s.SetMessageStatus(message.ID, status)

	logrus.Infof("Message sent to stream %s: %s", streamName, message.ID)
	return nil
}

// SendChatRequest 发送聊天请求
func (s *messageQueueServiceImpl) SendChatRequest(userID uint, conversationID *uint, content string, context map[string]interface{}) (string, error) {
	messageID := fmt.Sprintf("chat_%d_%d", userID, time.Now().UnixNano())
	
	message := &Message{
		ID:             messageID,
		Type:           MessageTypeChatRequest,
		UserID:         userID,
		ConversationID: conversationID,
		Content:        content,
		Context:        context,
		Priority:       "normal",
		Timestamp:      time.Now(),
		MaxRetries:     3,
	}

	err := s.SendMessage("ai:chat:requests", message)
	return messageID, err
}

// SendClusterOperation 发送集群操作请求
func (s *messageQueueServiceImpl) SendClusterOperation(userID uint, operation string, params map[string]interface{}) (string, error) {
	messageID := fmt.Sprintf("cluster_%d_%d", userID, time.Now().UnixNano())
	
	message := &Message{
		ID:         messageID,
		Type:       MessageTypeClusterOp,
		UserID:     userID,
		Content:    operation,
		Context:    params,
		Priority:   "high",
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	err := s.SendMessage("ai:cluster:operations", message)
	return messageID, err
}

// StartConsumer 启动消费者
func (s *messageQueueServiceImpl) StartConsumer(streamName, consumerGroup, consumerName string, handler MessageHandler) error {
	// 创建消费者组（如果不存在）
	err := s.redis.XGroupCreateMkStream(s.ctx, streamName, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %v", err)
	}

	go func() {
		logrus.Infof("Starting consumer %s in group %s for stream %s", consumerName, consumerGroup, streamName)
		
		for {
			select {
			case <-s.stopChan:
				logrus.Infof("Stopping consumer %s", consumerName)
				return
			default:
				// 读取消息
				args := &redis.XReadGroupArgs{
					Group:    consumerGroup,
					Consumer: consumerName,
					Streams:  []string{streamName, ">"},
					Count:    1,
					Block:    time.Second * 5,
				}

				streams, err := s.redis.XReadGroup(s.ctx, args).Result()
				if err != nil {
					if err != redis.Nil {
						logrus.Errorf("Error reading from stream %s: %v", streamName, err)
					}
					continue
				}

				// 处理消息
				for _, stream := range streams {
					for _, msg := range stream.Messages {
						s.processMessage(streamName, consumerGroup, msg, handler)
					}
				}
			}
		}
	}()

	return nil
}

// processMessage 处理单个消息
func (s *messageQueueServiceImpl) processMessage(streamName, consumerGroup string, msg redis.XMessage, handler MessageHandler) {
	messageData, ok := msg.Values["data"].(string)
	if !ok {
		logrus.Errorf("Invalid message format in stream %s", streamName)
		return
	}

	var message Message
	if err := json.Unmarshal([]byte(messageData), &message); err != nil {
		logrus.Errorf("Failed to unmarshal message: %v", err)
		return
	}

	// 更新状态为处理中
	status := &MessageStatus{
		ID:     message.ID,
		Status: "processing",
	}
	s.SetMessageStatus(message.ID, status)

	// 处理消息
	err := handler(&message)
	
	if err != nil {
		logrus.Errorf("Failed to process message %s: %v", message.ID, err)
		
		// 检查是否需要重试
		if message.RetryCount < message.MaxRetries {
			message.RetryCount++
			s.retryMessage(streamName, &message)
		} else {
			// 标记为失败
			status.Status = "failed"
			status.Error = err.Error()
			status.ProcessedAt = time.Now()
			s.SetMessageStatus(message.ID, status)
		}
	} else {
		// 标记为成功
		status.Status = "completed"
		status.ProcessedAt = time.Now()
		s.SetMessageStatus(message.ID, status)
	}

	// 确认消息处理完成
	s.redis.XAck(s.ctx, streamName, consumerGroup, msg.ID)
}

// retryMessage 重试消息
func (s *messageQueueServiceImpl) retryMessage(streamName string, message *Message) {
	// 指数退避
	delay := time.Duration(message.RetryCount*message.RetryCount) * time.Second
	
	go func() {
		time.Sleep(delay)
		s.SendMessage(streamName, message)
	}()
}

// StopConsumer 停止消费者
func (s *messageQueueServiceImpl) StopConsumer(streamName, consumerGroup string) error {
	close(s.stopChan)
	return nil
}

// SetMessageStatus 设置消息状态
func (s *messageQueueServiceImpl) SetMessageStatus(messageID string, status *MessageStatus) error {
	statusData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("ai:status:%s", messageID)
	return s.redis.Set(s.ctx, key, statusData, 5*time.Minute).Err()
}

// GetMessageStatus 获取消息状态
func (s *messageQueueServiceImpl) GetMessageStatus(messageID string) (*MessageStatus, error) {
	key := fmt.Sprintf("ai:status:%s", messageID)
	data, err := s.redis.Get(s.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("message status not found")
		}
		return nil, err
	}

	var status MessageStatus
	err = json.Unmarshal([]byte(data), &status)
	return &status, err
}

// HealthCheck 健康检查
func (s *messageQueueServiceImpl) HealthCheck() error {
	// 检查Redis连接
	_, err := s.redis.Ping(s.ctx).Result()
	if err != nil {
		return fmt.Errorf("redis connection failed: %v", err)
	}

	// 检查流是否存在
	streams := []string{"ai:chat:requests", "ai:cluster:operations", "ai:notifications"}
	for _, stream := range streams {
		_, err := s.redis.XLen(s.ctx, stream).Result()
		if err != nil {
			logrus.Warnf("Stream %s not accessible: %v", stream, err)
		}
	}

	return nil
}
