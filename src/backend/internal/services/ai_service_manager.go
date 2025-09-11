package services

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/sirupsen/logrus"
)

// AIServiceManager AI服务管理器
type AIServiceManager struct {
	messageProcessor    *AIMessageProcessor
	kafkaService        *KafkaMessageService
	messagePersistence  *MessagePersistenceService
	messageRetrieval    *MessageRetrievalService
	aiService           AIService
	messageQueueService MessageQueueService
	cacheService        CacheService
}

// NewAIServiceManager 创建AI服务管理器
func NewAIServiceManager() *AIServiceManager {
	// 初始化基础服务
	aiSvc := NewAIService()
	messageQueueSvc := NewMessageQueueService()
	cacheSvc := NewCacheService()

	// 初始化Kafka服务（如果配置了）
	var kafkaSvc *KafkaMessageService
	if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
		brokers := strings.Split(kafkaBrokers, ",")
		var err error
		kafkaSvc, err = NewKafkaMessageService(brokers)
		if err != nil {
			logrus.Warnf("Failed to initialize Kafka service: %v", err)
		} else {
			logrus.Info("Kafka service initialized successfully")
		}
	}

	// 初始化消息持久化服务
	messagePersistenceSvc := NewMessagePersistenceService(
		database.DB,
		kafkaSvc,
		messageQueueSvc,
	)

	// 初始化消息检索服务
	messageRetrievalSvc := NewMessageRetrievalService(
		messagePersistenceSvc,
		kafkaSvc,
		cacheSvc,
		messageQueueSvc,
	)

	// 使用带有依赖的AI服务
	if kafkaSvc != nil {
		aiSvc = NewAIServiceWithDependencies(
			database.DB,
			database.CryptoService,
			messagePersistenceSvc,
			messageRetrievalSvc,
			kafkaSvc,
		)
	}

	// 初始化消息处理器
	messageProcessor := NewAIMessageProcessorWithDependencies(
		aiSvc,
		messageQueueSvc,
		messagePersistenceSvc,
		kafkaSvc,
	)

	return &AIServiceManager{
		messageProcessor:    messageProcessor,
		kafkaService:        kafkaSvc,
		messagePersistence:  messagePersistenceSvc,
		messageRetrieval:    messageRetrievalSvc,
		aiService:           aiSvc,
		messageQueueService: messageQueueSvc,
		cacheService:        cacheSvc,
	}
}

// Start 启动所有AI服务
func (m *AIServiceManager) Start() error {
	logrus.Info("Starting AI Service Manager...")

	// 启动消息处理器
	if err := m.messageProcessor.Start(); err != nil {
		return fmt.Errorf("failed to start message processor: %v", err)
	}
	logrus.Info("Message processor started successfully")

	// 启动Kafka消费者（如果有）
	if m.kafkaService != nil {
		kafkaHandler := &AIServiceKafkaHandler{
			serviceManager: m,
		}

		if err := m.kafkaService.StartConsumer("ai-service-consumer", kafkaHandler); err != nil {
			logrus.Warnf("Failed to start Kafka consumer: %v", err)
		} else {
			logrus.Info("Kafka consumer started successfully")
		}
	}

	// 启动定期清理任务
	go m.startPeriodicTasks()

	logrus.Info("AI Service Manager started successfully")
	return nil
}

// Stop 停止所有AI服务
func (m *AIServiceManager) Stop() error {
	logrus.Info("Stopping AI Service Manager...")

	if m.kafkaService != nil {
		if err := m.kafkaService.Close(); err != nil {
			logrus.Warnf("Error closing Kafka service: %v", err)
		}
	}

	logrus.Info("AI Service Manager stopped")
	return nil
}

// GetMessageProcessor 获取消息处理器
func (m *AIServiceManager) GetMessageProcessor() *AIMessageProcessor {
	return m.messageProcessor
}

// GetMessagePersistence 获取消息持久化服务
func (m *AIServiceManager) GetMessagePersistence() *MessagePersistenceService {
	return m.messagePersistence
}

// GetMessageRetrieval 获取消息检索服务
func (m *AIServiceManager) GetMessageRetrieval() *MessageRetrievalService {
	return m.messageRetrieval
}

// GetAIService 获取AI服务
func (m *AIServiceManager) GetAIService() AIService {
	return m.aiService
}

// GetKafkaService 获取Kafka服务
func (m *AIServiceManager) GetKafkaService() *KafkaMessageService {
	return m.kafkaService
}

// startPeriodicTasks 启动定期任务
func (m *AIServiceManager) startPeriodicTasks() {
	// 每小时清理一次孤立消息
	// go func() {
	// 	ticker := time.NewTicker(1 * time.Hour)
	// 	defer ticker.Stop()

	// 	for range ticker.C {
	// 		if err := m.messagePersistence.CleanupOrphanedMessages(); err != nil {
	// 			logrus.Errorf("Failed to cleanup orphaned messages: %v", err)
	// 		}
	// 	}
	// }()

	// 每天归档一次旧消息
	// go func() {
	// 	ticker := time.NewTicker(24 * time.Hour)
	// 	defer ticker.Stop()

	// 	for range ticker.C {
	// 		if err := m.messagePersistence.ArchiveOldMessages(90); err != nil {
	// 			logrus.Errorf("Failed to archive old messages: %v", err)
	// 		}
	// 	}
	// }()
}

// AIServiceKafkaHandler AI服务Kafka处理器
type AIServiceKafkaHandler struct {
	serviceManager *AIServiceManager
}

func (h *AIServiceKafkaHandler) HandleMessage(message *KafkaMessage) error {
	logrus.Infof("Processing Kafka message: %s, type: %s", message.ID, message.Type)

	switch message.Type {
	case "cache_message":
		return h.handleCacheMessage(message)
	case "message_retrieved":
		return h.handleMessageRetrieved(message)
	case "cache_invalidated":
		return h.handleCacheInvalidated(message)
	default:
		logrus.Warnf("Unknown message type: %s", message.Type)
		return nil
	}
}

func (h *AIServiceKafkaHandler) handleCacheMessage(message *KafkaMessage) error {
	// 处理缓存消息事件
	logrus.Debugf("Handling cache message event: %s", message.ID)
	return nil
}

func (h *AIServiceKafkaHandler) handleMessageRetrieved(message *KafkaMessage) error {
	// 处理消息检索事件
	logrus.Debugf("Handling message retrieved event: %s", message.ID)
	return nil
}

func (h *AIServiceKafkaHandler) handleCacheInvalidated(message *KafkaMessage) error {
	// 处理缓存失效事件
	if conversationID, ok := message.Context["conversation_id"].(float64); ok {
		logrus.Infof("Invalidating cache for conversation %d", int(conversationID))
		h.serviceManager.messageRetrieval.InvalidateCache(uint(conversationID))
	}
	return nil
}

// HealthCheck 健康检查
func (m *AIServiceManager) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"timestamp": time.Now(),
		"services":  map[string]interface{}{},
	}

	// 检查消息队列健康状态
	if err := m.messageQueueService.HealthCheck(); err != nil {
		health["services"].(map[string]interface{})["message_queue"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["services"].(map[string]interface{})["message_queue"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// 检查缓存健康状态
	if err := m.cacheService.HealthCheck(); err != nil {
		health["services"].(map[string]interface{})["cache"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["services"].(map[string]interface{})["cache"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// 检查Kafka健康状态
	if m.kafkaService != nil {
		if err := m.kafkaService.HealthCheck(); err != nil {
			health["services"].(map[string]interface{})["kafka"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			health["services"].(map[string]interface{})["kafka"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	// 计算整体健康状态
	allHealthy := true
	for _, service := range health["services"].(map[string]interface{}) {
		if service.(map[string]interface{})["status"] != "healthy" {
			allHealthy = false
			break
		}
	}

	health["overall_status"] = "healthy"
	if !allHealthy {
		health["overall_status"] = "degraded"
	}

	return health
}
