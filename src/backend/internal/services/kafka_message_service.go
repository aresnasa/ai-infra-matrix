package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// KafkaMessageService Kafka消息服务
type KafkaMessageService struct {
	producer sarama.SyncProducer
	consumer sarama.ConsumerGroup
	topics   []string
	ctx      context.Context
	cancel   context.CancelFunc
}

// KafkaMessage Kafka消息结构
type KafkaMessage struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	UserID         uint                   `json:"user_id"`
	ConversationID *uint                  `json:"conversation_id,omitempty"`
	Content        string                 `json:"content"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Priority       string                 `json:"priority"`
	RetryCount     int                    `json:"retry_count"`
}

// NewKafkaMessageService 创建Kafka消息服务
func NewKafkaMessageService(brokers []string) (*KafkaMessageService, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	config.Version = sarama.V2_8_0_0

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaMessageService{
		producer: producer,
		ctx:      ctx,
		cancel:   cancel,
		topics:   []string{"ai-chat-messages", "ai-message-events", "ai-message-cache"},
	}, nil
}

// SendMessage 发送消息到Kafka
func (k *KafkaMessageService) SendMessage(topic string, message *KafkaMessage) error {
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(fmt.Sprintf("%d", message.UserID)),
		Value: sarama.StringEncoder(messageData),
		Timestamp: time.Now(),
	}

	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	logrus.Infof("Message sent to topic %s partition %d offset %d", topic, partition, offset)
	return nil
}

// SendChatMessage 发送聊天消息
func (k *KafkaMessageService) SendChatMessage(userID uint, conversationID *uint, content string, context map[string]interface{}) error {
	message := &KafkaMessage{
		ID:             fmt.Sprintf("chat_%d_%d", userID, time.Now().UnixNano()),
		Type:           "chat_message",
		UserID:         userID,
		ConversationID: conversationID,
		Content:        content,
		Context:        context,
		Timestamp:      time.Now(),
		Priority:       "normal",
	}

	return k.SendMessage("ai-chat-messages", message)
}

// SendMessageEvent 发送消息事件
func (k *KafkaMessageService) SendMessageEvent(eventType string, messageID string, userID uint, data map[string]interface{}) error {
	event := &KafkaMessage{
		ID:        fmt.Sprintf("event_%s_%d", eventType, time.Now().UnixNano()),
		Type:      eventType,
		UserID:   userID,
		Content:  messageID,
		Context:  data,
		Timestamp: time.Now(),
	}

	return k.SendMessage("ai-message-events", event)
}

// CacheMessage 缓存消息
func (k *KafkaMessageService) CacheMessage(conversationID uint, message *models.AIMessage) error {
	cacheData := map[string]interface{}{
		"conversation_id": conversationID,
		"message":         message,
		"action":          "cache",
	}

	event := &KafkaMessage{
		ID:        fmt.Sprintf("cache_%d_%d", conversationID, time.Now().UnixNano()),
		Type:      "cache_message",
		UserID:   0, // 系统消息
		Content:  fmt.Sprintf("Cache message for conversation %d", conversationID),
		Context:  cacheData,
		Timestamp: time.Now(),
	}

	return k.SendMessage("ai-message-cache", event)
}

// StartConsumer 启动消费者
func (k *KafkaMessageService) StartConsumer(groupID string, handler KafkaMessageHandler) error {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V2_8_0_0

	consumer, err := sarama.NewConsumerGroup([]string{"localhost:9092"}, groupID, config)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %v", err)
	}

	k.consumer = consumer

	go func() {
		for {
			select {
			case <-k.ctx.Done():
				return
			default:
				err := consumer.Consume(k.ctx, k.topics, &kafkaConsumerHandler{handler: handler})
				if err != nil {
					logrus.Errorf("Error consuming messages: %v", err)
				}
			}
		}
	}()

	return nil
}

// KafkaMessageHandler Kafka消息处理器接口
type KafkaMessageHandler interface {
	HandleMessage(message *KafkaMessage) error
}

// kafkaConsumerHandler Kafka消费者处理器
type kafkaConsumerHandler struct {
	handler KafkaMessageHandler
}

func (h *kafkaConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *kafkaConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *kafkaConsumerHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var kafkaMsg KafkaMessage
		if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
			logrus.Errorf("Failed to unmarshal message: %v", err)
			continue
		}

		if err := h.handler.HandleMessage(&kafkaMsg); err != nil {
			logrus.Errorf("Failed to handle message: %v", err)
			continue
		}

		sess.MarkMessage(msg, "")
	}
	return nil
}

// Close 关闭服务
func (k *KafkaMessageService) Close() error {
	k.cancel()
	if k.consumer != nil {
		k.consumer.Close()
	}
	return k.producer.Close()
}

// HealthCheck 健康检查
func (k *KafkaMessageService) HealthCheck() error {
	// 检查producer连接
	if k.producer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}
	return nil
}
