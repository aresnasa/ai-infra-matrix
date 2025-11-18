package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeChatRequest  MessageType = "chat_request"
	MessageTypeClusterOp    MessageType = "cluster_operation"
	MessageTypeNotification MessageType = "notification"
	MessageTypeStatusUpdate MessageType = "status_update"
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

	// 消息控制
	StopMessage(messageID string, userID uint) error
	CanUserStopMessage(messageID string, userID uint) (bool, error)
	IsMessageStopped(messageID string) bool

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
	// worker pool
	workerCount    int
	handlerTimeout time.Duration
}

// NewMessageQueueService 创建消息队列服务
func NewMessageQueueService() MessageQueueService {
	// configurable via environment variables for flexibility and horizontal scaling
	workerCount := 4
	if v := os.Getenv("MQ_WORKER_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workerCount = n
		}
	}

	handlerTimeout := 60 * time.Second
	if v := os.Getenv("MQ_HANDLER_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			handlerTimeout = time.Duration(n) * time.Second
		}
	}

	return &messageQueueServiceImpl{
		redis:          cache.RDB,
		ctx:            context.Background(),
		stopChan:       make(chan struct{}),
		workerCount:    workerCount,
		handlerTimeout: handlerTimeout,
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

	// 保存消息详情以供权限验证使用
	messageDetailsKey := fmt.Sprintf("message:details:%s", message.ID)
	err = s.redis.Set(s.ctx, messageDetailsKey, messageData, 30*time.Minute).Err() // 30分钟过期
	if err != nil {
		logrus.Errorf("保存消息详情失败: %v", err)
	}

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

	// start a buffered channel for delivering messages to workers
	msgCh := make(chan redis.XMessage, s.workerCount*2)

	// worker goroutines
	for i := 0; i < s.workerCount; i++ {
		workerName := fmt.Sprintf("%s-worker-%d", consumerName, i+1)
		go func(wname string) {
			logrus.Infof("Starting worker %s for stream %s", wname, streamName)
			for {
				select {
				case <-s.stopChan:
					logrus.Infof("Stopping worker %s", wname)
					return
				case msg := <-msgCh:
					// handle message with timeout context
					done := make(chan error, 1)
					go func(m redis.XMessage) {
						done <- s.processMessageWithHandler(streamName, consumerGroup, m, handler)
					}(msg)

					select {
					case err := <-done:
						if err != nil {
							logrus.Errorf("Worker %s: handler error for msg %s: %v", wname, msg.ID, err)
						}
						// ack after processing (success or failure) to avoid blocking the stream
						errAck := s.redis.XAck(s.ctx, streamName, consumerGroup, msg.ID).Err()
						if errAck != nil {
							logrus.Errorf("Failed to XAck message %s from stream %s: %v", msg.ID, streamName, errAck)
						}
					case <-time.After(s.handlerTimeout):
						logrus.Errorf("Worker %s: handler timeout for msg %s after %v", wname, msg.ID, s.handlerTimeout)
						// mark failed due to timeout
						mid := getMessageIDFromXMessage(msg)
						status := &MessageStatus{ID: mid, Status: "failed", Error: "handler timeout", ProcessedAt: time.Now()}
						s.SetMessageStatus(mid, status)
						// push to DLQ
						s.pushToDLQ(streamName, msg)
						// ack to avoid blocking
						errAck2 := s.redis.XAck(s.ctx, streamName, consumerGroup, msg.ID).Err()
						if errAck2 != nil {
							logrus.Errorf("Failed to XAck message %s from stream %s after timeout: %v", msg.ID, streamName, errAck2)
						}
					}
				}
			}
		}(workerName)
	}

	// reader loop: only XReadGroup and push to msgCh
	go func() {
		logrus.Infof("Starting consumer %s in group %s for stream %s", consumerName, consumerGroup, streamName)
		for {
			select {
			case <-s.stopChan:
				logrus.Infof("Stopping consumer %s", consumerName)
				close(msgCh)
				return
			default:
				args := &redis.XReadGroupArgs{
					Group:    consumerGroup,
					Consumer: consumerName,
					Streams:  []string{streamName, ">"},
					Count:    int64(s.workerCount),
					Block:    time.Second * 2,
				}

				streams, err := s.redis.XReadGroup(s.ctx, args).Result()
				if err != nil {
					if err != redis.Nil {
						// Check if it's a NOGROUP error (consumer group was deleted)
						if err.Error() == "NOGROUP No such key '"+streamName+"' or consumer group '"+consumerGroup+"' in XREADGROUP with GROUP option" ||
							(len(err.Error()) > 7 && err.Error()[:7] == "NOGROUP") {
							logrus.Warnf("Consumer group %s not found for stream %s, recreating...", consumerGroup, streamName)
							// Try to recreate the consumer group
							if recreateErr := s.redis.XGroupCreateMkStream(s.ctx, streamName, consumerGroup, "0").Err(); recreateErr != nil {
								if recreateErr.Error() != "BUSYGROUP Consumer Group name already exists" {
									logrus.Errorf("Failed to recreate consumer group %s: %v", consumerGroup, recreateErr)
								}
							} else {
								logrus.Infof("Successfully recreated consumer group %s for stream %s", consumerGroup, streamName)
							}
							time.Sleep(time.Second) // Brief pause before retrying
						} else {
							logrus.Errorf("Error reading from stream %s: %v", streamName, err)
						}
					}
					continue
				}

				for _, stream := range streams {
					for _, msg := range stream.Messages {
						select {
						case msgCh <- msg:
						default:
							// channel full, retry after a short sleep to apply backpressure
							logrus.Warnf("msgCh full, backpressure on stream %s", streamName)
							time.Sleep(200 * time.Millisecond)
							msgCh <- msg
						}
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

// processMessageWithHandler is a wrapper that returns an error for use with worker timeouts
func (s *messageQueueServiceImpl) processMessageWithHandler(streamName, consumerGroup string, msg redis.XMessage, handler MessageHandler) error {
	// reuse message extraction logic from processMessage but return errors instead of only logging
	messageData, ok := msg.Values["data"].(string)
	if !ok {
		return fmt.Errorf("Invalid message format in stream %s", streamName)
	}

	var message Message
	if err := json.Unmarshal([]byte(messageData), &message); err != nil {
		return fmt.Errorf("Failed to unmarshal message: %v", err)
	}

	// update status
	status := &MessageStatus{
		ID:     message.ID,
		Status: "processing",
	}
	if err := s.SetMessageStatus(message.ID, status); err != nil {
		logrus.Errorf("Failed to set message status: %v", err)
	}

	// handler
	if err := handler(&message); err != nil {
		// processing failed
		if message.RetryCount < message.MaxRetries {
			message.RetryCount++
			s.retryMessage(streamName, &message)
			return fmt.Errorf("handler error, scheduled retry: %v", err)
		}

		status.Status = "failed"
		status.Error = err.Error()
		status.ProcessedAt = time.Now()
		if err2 := s.SetMessageStatus(message.ID, status); err2 != nil {
			logrus.Errorf("Failed to set failed status: %v", err2)
		}
		return err
	}

	status.Status = "completed"
	status.ProcessedAt = time.Now()
	if err := s.SetMessageStatus(message.ID, status); err != nil {
		logrus.Errorf("Failed to set completed status: %v", err)
	}
	return nil
}

// getMessageIDFromXMessage extracts original message ID from XMessage payload
func getMessageIDFromXMessage(msg redis.XMessage) string {
	messageData, ok := msg.Values["data"].(string)
	if !ok {
		return ""
	}
	var message Message
	if err := json.Unmarshal([]byte(messageData), &message); err != nil {
		return ""
	}
	return message.ID
}

// pushToDLQ pushes the raw XMessage into a DLQ stream for later inspection/reprocessing
func (s *messageQueueServiceImpl) pushToDLQ(originalStream string, msg redis.XMessage) {
	dlqStream := os.Getenv("MQ_DLQ_STREAM")
	if dlqStream == "" {
		dlqStream = "ai:chat:dlq"
	}

	// preserve original fields
	messageData, _ := msg.Values["data"].(string)
	payload := map[string]interface{}{
		"original_stream": originalStream,
		"original_id":     msg.ID,
		"data":            messageData,
		"pushed_at":       time.Now().Format(time.RFC3339),
	}

	args := &redis.XAddArgs{
		Stream: dlqStream,
		Values: payload,
	}
	if _, err := s.redis.XAdd(s.ctx, args).Result(); err != nil {
		logrus.Errorf("Failed to push message %s to DLQ %s: %v", msg.ID, dlqStream, err)
	} else {
		logrus.Infof("Pushed message %s from stream %s to DLQ %s", msg.ID, originalStream, dlqStream)
	}
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

// StopMessage 停止消息处理
func (s *messageQueueServiceImpl) StopMessage(messageID string, userID uint) error {
	// 首先检查消息是否可以被停止
	canStop, err := s.CanUserStopMessage(messageID, userID)
	if err != nil {
		return fmt.Errorf("检查停止权限失败: %v", err)
	}

	if !canStop {
		return fmt.Errorf("用户无权停止此消息处理")
	}

	// 设置停止标志
	stopKey := fmt.Sprintf("message:stop:%s", messageID)
	err = s.redis.Set(s.ctx, stopKey, userID, 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("设置停止标志失败: %v", err)
	}

	// 更新消息状态为已停止
	status := &MessageStatus{
		ID:          messageID,
		Status:      "stopped",
		Error:       fmt.Sprintf("用户 %d 主动停止", userID),
		ProcessedAt: time.Now(),
	}

	err = s.SetMessageStatus(messageID, status)
	if err != nil {
		logrus.Errorf("更新消息状态失败: %v", err)
	}

	logrus.Infof("消息 %s 已被用户 %d 停止", messageID, userID)
	return nil
}

// CanUserStopMessage 检查用户是否可以停止指定消息
func (s *messageQueueServiceImpl) CanUserStopMessage(messageID string, userID uint) (bool, error) {
	// 获取消息状态
	status, err := s.GetMessageStatus(messageID)
	if err != nil {
		return false, fmt.Errorf("获取消息状态失败: %v", err)
	}

	if status == nil {
		return false, fmt.Errorf("消息不存在")
	}

	// 检查消息是否正在处理中
	if status.Status != "processing" && status.Status != "pending" {
		return false, fmt.Errorf("消息状态为 %s，无法停止", status.Status)
	}

	// 从Redis中获取消息详情以验证用户权限
	messageKey := fmt.Sprintf("message:details:%s", messageID)
	messageData, err := s.redis.Get(s.ctx, messageKey).Result()
	if err != nil {
		if err == redis.Nil {
			return false, fmt.Errorf("消息详情不存在")
		}
		return false, fmt.Errorf("获取消息详情失败: %v", err)
	}

	var message Message
	err = json.Unmarshal([]byte(messageData), &message)
	if err != nil {
		return false, fmt.Errorf("解析消息详情失败: %v", err)
	}

	// 检查用户权限 - 只有消息创建者可以停止
	if message.UserID != userID {
		return false, nil
	}

	return true, nil
}

// IsMessageStopped 检查消息是否被停止（供消息处理器使用）
func (s *messageQueueServiceImpl) IsMessageStopped(messageID string) bool {
	stopKey := fmt.Sprintf("message:stop:%s", messageID)
	exists, err := s.redis.Exists(s.ctx, stopKey).Result()
	if err != nil {
		logrus.Errorf("检查消息停止标志失败: %v", err)
		return false
	}
	return exists > 0
}
