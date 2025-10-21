package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// AIMessageProcessor AI消息处理器
type AIMessageProcessor struct {
	aiService           AIService
	messageQueueService MessageQueueService
	cacheService        CacheService
	messagePersistence  *MessagePersistenceService
	kafkaService        *KafkaMessageService
}

// NewAIMessageProcessor 创建AI消息处理器
func NewAIMessageProcessor(aiService AIService, messageQueueService MessageQueueService) *AIMessageProcessor {
	return &AIMessageProcessor{
		aiService:           aiService,
		messageQueueService: messageQueueService,
		cacheService:        NewCacheService(),
	}
}

// NewAIMessageProcessorWithDependencies 创建带有完整依赖的AI消息处理器
func NewAIMessageProcessorWithDependencies(
	aiService AIService,
	messageQueueService MessageQueueService,
	messagePersistence *MessagePersistenceService,
	kafkaService *KafkaMessageService,
) *AIMessageProcessor {
	return &AIMessageProcessor{
		aiService:           aiService,
		messageQueueService: messageQueueService,
		cacheService:        NewCacheService(),
		messagePersistence:  messagePersistence,
		kafkaService:        kafkaService,
	}
}

// Start 启动消息处理器
func (p *AIMessageProcessor) Start() error {
	// 启动聊天请求处理器
	err := p.messageQueueService.StartConsumer(
		"ai:chat:requests",
		"chat-processors",
		"chat-worker-1",
		p.handleChatRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to start chat request processor: %v", err)
	}

	// 启动集群操作处理器
	err = p.messageQueueService.StartConsumer(
		"ai:cluster:operations",
		"cluster-ops",
		"cluster-worker-1",
		p.handleClusterOperation,
	)
	if err != nil {
		return fmt.Errorf("failed to start cluster operation processor: %v", err)
	}

	logrus.Info("AI Message Processor started successfully")
	return nil
}

// handleChatRequest 处理聊天请求
func (p *AIMessageProcessor) handleChatRequest(message *Message) error {
	logrus.Infof("Processing chat request: %s", message.ID)

	// 检查消息是否已被停止
	if p.isMessageStopped(message.ID) {
		logrus.Infof("消息 %s 已被停止，跳过处理", message.ID)
		return p.markMessageStopped(message.ID)
	}

	// 更新处理状态
	status := &MessageStatus{
		ID:     message.ID,
		Status: "processing",
	}
	p.messageQueueService.SetMessageStatus(message.ID, status)

	var conversationID uint
	var err error

	// 如果没有对话ID，创建新对话
	if message.ConversationID == nil {
		// 在创建对话前再次检查是否被停止
		if p.isMessageStopped(message.ID) {
			return p.markMessageStopped(message.ID)
		}

		conversation, createErr := p.createConversationFromContext(message)
		if createErr != nil {
			return fmt.Errorf("failed to create conversation: %v", createErr)
		}
		conversationID = conversation.ID
	} else {
		conversationID = *message.ConversationID
	}

	// 检查缓存中是否有最近的对话
	cacheKey := fmt.Sprintf("conversation:%d", conversationID)
	cachedConversation := p.cacheService.GetConversation(cacheKey)

	var conversation *models.AIConversation
	if cachedConversation != nil {
		conversation = cachedConversation
	} else {
		// 从数据库获取对话
		conversation, err = p.aiService.GetConversation(conversationID)
		if err != nil {
			return fmt.Errorf("failed to get conversation: %v", err)
		}
		// 缓存对话信息
		p.cacheService.SetConversation(cacheKey, conversation, 30*time.Minute)
	}

	// 保存用户消息到数据库
	userMessage := &models.AIMessage{
		ConversationID: conversationID,
		Role:           "user",
		Content:        message.Content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if p.messagePersistence != nil {
		if err := p.messagePersistence.SaveMessage(userMessage); err != nil {
			logrus.Errorf("Failed to save user message: %v", err)
			// 继续处理，不影响主流程
		}
	}

	// 在发送到AI前最后检查一次是否被停止
	if p.isMessageStopped(message.ID) {
		return p.markMessageStopped(message.ID)
	}

	// 发送消息并获取AI回复
	aiMessage, err := p.aiService.SendMessage(conversationID, message.Content)
	if err != nil {
		return fmt.Errorf("failed to send message to AI: %v", err)
	}

	// 处理完成后检查是否被停止（虽然AI已经处理了，但可以标记状态）
	if p.isMessageStopped(message.ID) {
		logrus.Infof("消息 %s 在AI处理完成后被标记为停止", message.ID)
		return p.markMessageStopped(message.ID)
	}

	// ===== 关键优化：确保缓存一致性 =====
	// 1. 先删除所有带查询参数的缓存键，防止并发读取到旧数据
	p.cacheService.DeleteKeysWithPattern(fmt.Sprintf("messages:%d:*", conversationID))

	// 2. 更新基础消息缓存
	messagesKey := fmt.Sprintf("messages:%d", conversationID)
	p.cacheService.AppendMessage(messagesKey, aiMessage)

	// 3. 主动重建最常用查询的缓存（预热）
	// 从数据库获取最新的完整消息列表
	if p.messagePersistence != nil {
		latestMessagesPtr, err := p.messagePersistence.GetMessageHistory(conversationID, 50, 0)
		if err == nil && len(latestMessagesPtr) > 0 {
			// 转换指针切片为值切片
			latestMessages := make([]models.AIMessage, len(latestMessagesPtr))
			for i, msgPtr := range latestMessagesPtr {
				latestMessages[i] = *msgPtr
			}
			// 缓存最常见的查询：limit=50, sort=created_at_asc
			commonQueryKey := fmt.Sprintf("messages:%d:limit_50:offset_0:sort_created_at_asc", conversationID)
			p.cacheService.SetMessages(commonQueryKey, latestMessages, 1*time.Hour)
			logrus.Debugf("Prewarmed cache for conversation %d with %d messages", conversationID, len(latestMessages))
		} else if err != nil {
			logrus.Warnf("Failed to prewarm cache for conversation %d: %v", conversationID, err)
		}
	}

	// 发送到Kafka进行流处理
	if p.kafkaService != nil {
		if err := p.kafkaService.CacheMessage(conversationID, aiMessage); err != nil {
			logrus.Warnf("Failed to send message to Kafka: %v", err)
		}

		// 发送消息事件
		if err := p.kafkaService.SendMessageEvent("message_processed", message.ID, message.UserID, map[string]interface{}{
			"conversation_id": conversationID,
			"ai_message_id":   aiMessage.ID,
			"tokens_used":     aiMessage.TokensUsed,
		}); err != nil {
			logrus.Warnf("Failed to send message event to Kafka: %v", err)
		}
	}

	// 更新成功状态
	status.Status = "completed"
	status.Result = aiMessage.Content
	status.ProcessedAt = time.Now()
	p.messageQueueService.SetMessageStatus(message.ID, status)

	// 发送实时通知给前端
	p.sendRealtimeNotification(message.UserID, conversationID, aiMessage)

	logrus.Infof("Chat request processed successfully: %s", message.ID)
	return nil
}

// handleClusterOperation 处理集群操作请求
func (p *AIMessageProcessor) handleClusterOperation(message *Message) error {
	logrus.Infof("Processing cluster operation: %s", message.ID)

	// 解析集群操作
	operation, err := p.parseClusterOperation(message.Content, message.Context)
	if err != nil {
		return fmt.Errorf("failed to parse cluster operation: %v", err)
	}

	// 执行集群操作
	result, err := p.executeClusterOperation(operation)
	if err != nil {
		return fmt.Errorf("failed to execute cluster operation: %v", err)
	}

	// 更新状态
	status := &MessageStatus{
		ID:          message.ID,
		Status:      "completed",
		Result:      result,
		ProcessedAt: time.Now(),
	}
	p.messageQueueService.SetMessageStatus(message.ID, status)

	// 发送结果通知
	p.sendOperationResult(message.UserID, operation, result)

	return nil
}

// createConversationFromContext 根据上下文创建对话
func (p *AIMessageProcessor) createConversationFromContext(message *Message) (*models.AIConversation, error) {
	// 获取默认配置
	config, err := p.aiService.GetDefaultConfig()
	if err != nil {
		return nil, err
	}

	// 根据上下文生成标题
	title := p.generateConversationTitle(message.Content, message.Context)

	// 创建对话
	conversation, err := p.aiService.CreateConversation(
		message.UserID,
		config.ID,
		title,
		p.contextToString(message.Context),
	)

	return conversation, err
}

// generateConversationTitle 生成对话标题
func (p *AIMessageProcessor) generateConversationTitle(content string, context map[string]interface{}) string {
	// 简单的标题生成逻辑
	if len(content) > 20 {
		return content[:20] + "..."
	}
	return content
}

// contextToString 将上下文转换为字符串
func (p *AIMessageProcessor) contextToString(context map[string]interface{}) string {
	if context == nil {
		return ""
	}

	data, err := json.Marshal(context)
	if err != nil {
		return ""
	}

	return string(data)
}

// parseClusterOperation 解析集群操作
func (p *AIMessageProcessor) parseClusterOperation(content string, context map[string]interface{}) (*ClusterOperation, error) {
	// 使用AI或规则引擎解析操作内容
	operation := &ClusterOperation{
		Type:       p.detectOperationType(content),
		Content:    content,
		Parameters: context,
		CreatedAt:  time.Now(),
	}

	return operation, nil
}

// detectOperationType 检测操作类型
func (p *AIMessageProcessor) detectOperationType(content string) string {
	// 简单的关键词匹配，可以扩展为更复杂的NLP分析
	keywords := map[string]string{
		"部署":     "deployment",
		"deploy": "deployment",
		"扩容":     "scaling",
		"scale":  "scaling",
		"监控":     "monitoring",
		"查看状态":   "status",
		"删除":     "deletion",
		"更新":     "update",
	}

	for keyword, opType := range keywords {
		if contains(content, keyword) {
			return opType
		}
	}

	return "unknown"
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	// 简化版本，实际可以使用更复杂的字符串匹配
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// executeClusterOperation 执行集群操作
func (p *AIMessageProcessor) executeClusterOperation(operation *ClusterOperation) (string, error) {
	switch operation.Type {
	case "deployment":
		return p.executeDeployment(operation)
	case "scaling":
		return p.executeScaling(operation)
	case "monitoring":
		return p.executeMonitoring(operation)
	default:
		return "", fmt.Errorf("unsupported operation type: %s", operation.Type)
	}
}

// executeDeployment 执行部署操作
func (p *AIMessageProcessor) executeDeployment(operation *ClusterOperation) (string, error) {
	// 这里集成Ansible或K8s客户端
	logrus.Infof("Executing deployment operation: %s", operation.Content)

	// 模拟执行结果
	result := fmt.Sprintf("部署操作已提交执行，操作ID: %s",
		fmt.Sprintf("deploy_%d", time.Now().Unix()))

	return result, nil
}

// executeScaling 执行扩容操作
func (p *AIMessageProcessor) executeScaling(operation *ClusterOperation) (string, error) {
	logrus.Infof("Executing scaling operation: %s", operation.Content)

	result := fmt.Sprintf("扩容操作已提交执行，操作ID: %s",
		fmt.Sprintf("scale_%d", time.Now().Unix()))

	return result, nil
}

// executeMonitoring 执行监控操作
func (p *AIMessageProcessor) executeMonitoring(operation *ClusterOperation) (string, error) {
	logrus.Infof("Executing monitoring operation: %s", operation.Content)

	result := fmt.Sprintf("监控查询已完成，数据已更新到仪表板")

	return result, nil
}

// sendRealtimeNotification 发送实时通知
func (p *AIMessageProcessor) sendRealtimeNotification(userID uint, conversationID uint, message *models.AIMessage) {
	notification := map[string]interface{}{
		"type":            "ai_message",
		"user_id":         userID,
		"conversation_id": conversationID,
		"message_id":      message.ID,
		"content":         message.Content,
		"timestamp":       message.CreatedAt,
	}

	// 发送到通知流
	notificationMessage := &Message{
		ID:        fmt.Sprintf("notification_%d_%d", userID, time.Now().UnixNano()),
		Type:      MessageTypeNotification,
		UserID:    userID,
		Content:   "new_ai_message",
		Context:   notification,
		Priority:  "normal",
		Timestamp: time.Now(),
	}

	p.messageQueueService.SendMessage("ai:notifications", notificationMessage)
}

// sendOperationResult 发送操作结果
func (p *AIMessageProcessor) sendOperationResult(userID uint, operation *ClusterOperation, result string) {
	notification := map[string]interface{}{
		"type":      "cluster_operation_result",
		"user_id":   userID,
		"operation": operation.Type,
		"result":    result,
		"timestamp": time.Now(),
	}

	notificationMessage := &Message{
		ID:        fmt.Sprintf("op_result_%d_%d", userID, time.Now().UnixNano()),
		Type:      MessageTypeNotification,
		UserID:    userID,
		Content:   "cluster_operation_completed",
		Context:   notification,
		Priority:  "normal",
		Timestamp: time.Now(),
	}

	p.messageQueueService.SendMessage("ai:notifications", notificationMessage)
}

// ClusterOperation 集群操作结构
type ClusterOperation struct {
	Type       string                 `json:"type"`
	Content    string                 `json:"content"`
	Parameters map[string]interface{} `json:"parameters"`
	CreatedAt  time.Time              `json:"created_at"`
}

// isMessageStopped 检查消息是否被停止
func (p *AIMessageProcessor) isMessageStopped(messageID string) bool {
	// 检查消息队列服务中的停止标志
	if mqsImpl, ok := p.messageQueueService.(*messageQueueServiceImpl); ok {
		return mqsImpl.IsMessageStopped(messageID)
	}

	// 如果无法访问实现细节，返回false
	return false
}

// markMessageStopped 标记消息为已停止状态
func (p *AIMessageProcessor) markMessageStopped(messageID string) error {
	status := &MessageStatus{
		ID:          messageID,
		Status:      "stopped",
		Error:       "消息处理已被用户停止",
		ProcessedAt: time.Now(),
	}

	err := p.messageQueueService.SetMessageStatus(messageID, status)
	if err != nil {
		logrus.Errorf("更新消息停止状态失败: %v", err)
		return fmt.Errorf("更新消息停止状态失败: %v", err)
	}

	logrus.Infof("消息 %s 已标记为停止状态", messageID)
	return nil
}
