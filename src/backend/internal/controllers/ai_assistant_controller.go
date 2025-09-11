package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AIAssistantController AI助手控制器
type AIAssistantController struct {
	aiService             services.AIService
	messageQueueService   services.MessageQueueService
	cacheService          services.CacheService
	messagePersistence    *services.MessagePersistenceService
	messageRetrieval      *services.MessageRetrievalService
	kafkaService          *services.KafkaMessageService
}

// NewAIAssistantController 创建AI助手控制器
func NewAIAssistantController() *AIAssistantController {
	// 初始化基础服务
	aiSvc := services.NewAIService()
	messageQueueSvc := services.NewMessageQueueService()
	cacheSvc := services.NewCacheService()

	// 初始化Kafka服务（如果配置了）
	var kafkaSvc *services.KafkaMessageService
	if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
		brokers := strings.Split(kafkaBrokers, ",")
		var err error
		kafkaSvc, err = services.NewKafkaMessageService(brokers)
		if err != nil {
			logrus.Warnf("Failed to initialize Kafka service: %v", err)
		}
	}

	// 初始化消息持久化服务
	messagePersistenceSvc := services.NewMessagePersistenceService(
		database.DB,
		kafkaSvc,
		messageQueueSvc,
	)

	// 初始化消息检索服务
	messageRetrievalSvc := services.NewMessageRetrievalService(
		messagePersistenceSvc,
		kafkaSvc,
		cacheSvc,
		messageQueueSvc,
	)

	// 使用带有依赖的AI服务
	if kafkaSvc != nil {
		aiSvc = services.NewAIServiceWithDependencies(
			database.DB,
			database.CryptoService,
			messagePersistenceSvc,
			messageRetrievalSvc,
			kafkaSvc,
		)
	}

	return &AIAssistantController{
		aiService:           aiSvc,
		messageQueueService: messageQueueSvc,
		cacheService:        cacheSvc,
		messagePersistence:  messagePersistenceSvc,
		messageRetrieval:    messageRetrievalSvc,
		kafkaService:        kafkaSvc,
	}
}

// CreateConfig 创建AI配置
func (ctrl *AIAssistantController) CreateConfig(c *gin.Context) {
	var req models.AIAssistantConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.aiService.CreateConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "AI配置创建成功", "data": req})
}

// GetConfig 获取AI配置
func (ctrl *AIAssistantController) GetConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}

	config, err := ctrl.aiService.GetConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	// 隐藏API密钥
	config.APIKey = "***"
	c.JSON(http.StatusOK, gin.H{"data": config})
}

// ListConfigs 获取AI配置列表
func (ctrl *AIAssistantController) ListConfigs(c *gin.Context) {
	configs, err := ctrl.aiService.ListConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": configs})
}

// UpdateConfig 更新AI配置
func (ctrl *AIAssistantController) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}

	var req models.AIAssistantConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uint(id)
	if err := ctrl.aiService.UpdateConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI配置更新成功"})
}

// DeleteConfig 删除AI配置
func (ctrl *AIAssistantController) DeleteConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}

	if err := ctrl.aiService.DeleteConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI配置删除成功"})
}

// CreateConversation 创建对话
func (ctrl *AIAssistantController) CreateConversation(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req struct {
		ConfigID *uint  `json:"config_id"` // 使ConfigID可选
		Title    string `json:"title"`
		Context  string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果没有提供ConfigID，使用默认配置
	var configID uint
	if req.ConfigID != nil {
		configID = *req.ConfigID
	} else {
		// 获取默认配置
		defaultConfig, err := ctrl.aiService.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取默认AI配置: " + err.Error()})
			return
		}
		configID = defaultConfig.ID
	}

	if req.Title == "" {
		req.Title = "新对话"
	}

	conversation, err := ctrl.aiService.CreateConversation(userID, configID, req.Title, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "对话创建成功", "data": conversation})
}

// ListConversations 获取用户对话列表（增强缓存版本）
func (ctrl *AIAssistantController) ListConversations(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 先从缓存获取
	cacheKey := fmt.Sprintf("user_conversations:%d", userID)
	cachedConversations := ctrl.cacheService.GetMessages(cacheKey)
	
	if cachedConversations != nil {
		c.JSON(http.StatusOK, gin.H{"data": cachedConversations, "from_cache": true})
		return
	}

	// 缓存未命中，从数据库获取
	conversations, err := ctrl.aiService.ListUserConversations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 缓存结果
	// 注意：这里为了简化，复用了Messages的缓存方法，实际应该为Conversations创建专门的缓存方法
	// ctrl.cacheService.SetConversations(cacheKey, conversations, 15*time.Minute)

	c.JSON(http.StatusOK, gin.H{"data": conversations, "from_cache": false})
}

// GetConversation 获取对话详情
func (ctrl *AIAssistantController) GetConversation(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	conversation, err := ctrl.aiService.GetConversation(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "对话不存在"})
		return
	}

	// 检查权限
	if conversation.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此对话"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": conversation})
}

// DeleteConversation 删除对话
func (ctrl *AIAssistantController) DeleteConversation(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	conversation, err := ctrl.aiService.GetConversation(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "对话不存在"})
		return
	}

	// 检查权限
	if conversation.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此对话"})
		return
	}

	if err := ctrl.aiService.DeleteConversation(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "对话删除成功"})
}

// SendMessage 发送消息（异步版本）
func (ctrl *AIAssistantController) SendMessage(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查对话权限
	conversation, err := ctrl.aiService.GetConversation(uint(conversationID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "对话不存在"})
		return
	}

	if conversation.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此对话"})
		return
	}

	// 异步发送消息到队列
	messageID, err := ctrl.messageQueueService.SendChatRequest(
		userID, 
		&[]uint{uint(conversationID)}[0], 
		req.Message, 
		map[string]interface{}{
			"page": c.GetHeader("Referer"),
			"conversation_id": conversationID,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "消息发送失败"})
		return
	}

	// 立即返回消息ID，前端可以用来查询状态
	c.JSON(http.StatusAccepted, gin.H{
		"message_id": messageID,
		"status": "pending",
		"message": "消息已提交处理",
	})
}

// GetMessages 获取对话消息（增强缓存版本）
func (ctrl *AIAssistantController) GetMessages(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	// 检查对话权限
	conversation, err := ctrl.aiService.GetConversation(uint(conversationID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "对话不存在"})
		return
	}

	if conversation.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此对话"})
		return
	}

	// 解析查询参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	keyword := c.Query("keyword")

	options := &services.MessageQueryOptions{
		Limit:   limit,
		Offset:  offset,
		Keyword: keyword,
		SortBy:  "created_at",
		SortOrder: "asc",
	}

	// 使用优化的消息检索服务
	messages, err := ctrl.messageRetrieval.GetMessagesWithOptimization(uint(conversationID), userID, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换消息格式
	result := make([]models.AIMessage, len(messages))
	for i, msg := range messages {
		result[i] = *msg
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
		"from_cache": c.GetBool("from_cache"),
		"total": len(result),
	})
}

// SubmitClusterOperation 提交集群操作请求
func (ctrl *AIAssistantController) SubmitClusterOperation(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req struct {
		Operation   string                 `json:"operation" binding:"required"`
		Parameters  map[string]interface{} `json:"parameters"`
		ClusterID   *uint                  `json:"cluster_id"`
		Description string                 `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 构建操作上下文
	context := req.Parameters
	if context == nil {
		context = make(map[string]interface{})
	}
	context["cluster_id"] = req.ClusterID
	context["description"] = req.Description
	context["source"] = "manual_operation"

	// 发送到集群操作队列
	operationID, err := ctrl.messageQueueService.SendClusterOperation(
		userID, 
		req.Operation, 
		context,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "集群操作提交失败"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"operation_id": operationID,
		"status": "pending",
		"message": "集群操作已提交处理",
	})
}

// GetOperationStatus 获取集群操作状态
func (ctrl *AIAssistantController) GetOperationStatus(c *gin.Context) {
	operationID := c.Param("operation_id")
	if operationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作ID不能为空"})
		return
	}

	status, err := ctrl.messageQueueService.GetMessageStatus(operationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "操作状态不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}

// GetSystemHealth 获取AI系统健康状态
func (ctrl *AIAssistantController) GetSystemHealth(c *gin.Context) {
	health := map[string]interface{}{
		"timestamp": time.Now(),
		"services": map[string]interface{}{},
	}

	// 检查消息队列健康状态
	if err := ctrl.messageQueueService.HealthCheck(); err != nil {
		health["services"].(map[string]interface{})["message_queue"] = map[string]interface{}{
			"status": "unhealthy",
			"error": err.Error(),
		}
	} else {
		health["services"].(map[string]interface{})["message_queue"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// 检查缓存健康状态
	if err := ctrl.cacheService.HealthCheck(); err != nil {
		health["services"].(map[string]interface{})["cache"] = map[string]interface{}{
			"status": "unhealthy",
			"error": err.Error(),
		}
	} else {
		health["services"].(map[string]interface{})["cache"] = map[string]interface{}{
			"status": "healthy",
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

	c.JSON(http.StatusOK, gin.H{"data": health})
}

// GetUsageStats 获取AI系统使用统计
func (ctrl *AIAssistantController) GetUsageStats(c *gin.Context) {
	// 从缓存或数据库获取使用统计
	stats := map[string]interface{}{
		"timestamp": time.Now(),
		"total_messages": 0,
		"total_operations": 0,
		"active_conversations": 0,
		"queue_status": map[string]interface{}{},
		"cache_stats": map[string]interface{}{},
	}

	// 获取队列统计
	queueStats := map[string]interface{}{}

	// 从Redis获取队列长度（如果可用）
	// TODO: 实现实际的统计收集逻辑
	queueStats["chat_requests_pending"] = 0
	queueStats["cluster_operations_pending"] = 0
	queueStats["notifications_pending"] = 0

	stats["queue_status"] = queueStats

	// 获取缓存统计
	cacheStats := map[string]interface{}{}
	cacheStats["hit_rate"] = 85.5
	cacheStats["total_keys"] = 100
	cacheStats["memory_usage"] = "10MB"

	stats["cache_stats"] = cacheStats

	// 模拟一些基础统计数据
	stats["total_messages"] = 1000
	stats["total_operations"] = 150
	stats["active_conversations"] = 25

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// TestBotConnection 测试机器人连接
func (ctrl *AIAssistantController) TestBotConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}

	config, err := ctrl.aiService.GetConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	// 测试连接
	testResult, err := ctrl.aiService.TestConnection(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("连接测试失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "连接测试完成",
		"data": testResult,
	})
}

// GetBotModels 获取机器人支持的模型列表
func (ctrl *AIAssistantController) GetBotModels(c *gin.Context) {
	provider := c.Query("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定提供商"})
		return
	}

	models, err := ctrl.aiService.GetAvailableModels(models.AIProvider(provider))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取模型列表失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}

// QuickChat 快速聊天接口（无需创建对话）
func (ctrl *AIAssistantController) QuickChat(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req struct {
		ConfigID *uint  `json:"config_id"` // 使ConfigID可选
		Message  string `json:"message" binding:"required"`
		Context  string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果没有提供ConfigID，使用默认配置
	var configID uint
	if req.ConfigID != nil {
		configID = *req.ConfigID
	} else {
		// 获取默认配置
		defaultConfig, err := ctrl.aiService.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取默认AI配置: " + err.Error()})
			return
		}
		configID = defaultConfig.ID
	}

	// 异步处理快速聊天
	messageID, err := ctrl.messageQueueService.SendChatRequest(
		userID,
		nil, // 无对话ID，自动创建新对话
		req.Message,
		map[string]interface{}{
			"page": c.GetHeader("Referer"),
			"type": "quick_chat",
			"config_id": configID,
			"context": req.Context,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "快速聊天请求失败"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message_id": messageID,
		"status": "processing",
		"message": "快速聊天请求已提交",
	})
}

// GetMessageStatus 获取消息处理状态
func (ctrl *AIAssistantController) GetMessageStatus(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的消息ID"})
		return
	}

	status, result, err := ctrl.aiService.GetMessageStatus(uint(messageID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取状态失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"message_id": messageID,
			"status": status,
			"result": result,
		},
	})
}

// SearchMessages 搜索消息
func (ctrl *AIAssistantController) SearchMessages(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索关键词不能为空"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 使用消息检索服务搜索
	messages, err := ctrl.messageRetrieval.SearchMessagesInConversation(uint(conversationID), userID, keyword, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换消息格式
	result := make([]models.AIMessage, len(messages))
	for i, msg := range messages {
		result[i] = *msg
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
		"total": len(result),
		"keyword": keyword,
	})
}

// GetMessageStats 获取消息统计
func (ctrl *AIAssistantController) GetMessageStats(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 解析时间参数
	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	} else {
		startDate = time.Now().AddDate(0, -1, 0) // 默认一个月前
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	} else {
		endDate = time.Now()
	}

	// 获取统计信息
	stats, err := ctrl.messagePersistence.GetMessageStats(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// DeleteMessage 删除消息
func (ctrl *AIAssistantController) DeleteMessage(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的消息ID"})
		return
	}

	// 删除消息
	if err := ctrl.messagePersistence.DeleteMessage(uint(messageID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 使相关缓存失效
	conversationID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if conversationID > 0 {
		ctrl.messageRetrieval.InvalidateCache(uint(conversationID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "消息删除成功"})
}

// StreamMessages 流式获取消息
func (ctrl *AIAssistantController) StreamMessages(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	lastMessageID, _ := strconv.ParseUint(c.DefaultQuery("last_message_id", "0"), 10, 32)

	// 获取消息流
	messageChan, err := ctrl.messageRetrieval.StreamMessages(uint(conversationID), userID, uint(lastMessageID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 发送消息流
	c.Stream(func(w io.Writer) bool {
		select {
		case message, ok := <-messageChan:
			if !ok {
				return false
			}

			// 发送SSE事件
			c.SSEvent("message", message)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// PreloadMessages 预加载消息到缓存
func (ctrl *AIAssistantController) PreloadMessages(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话ID"})
		return
	}

	// 预加载消息
	if err := ctrl.messageRetrieval.PreloadMessages(uint(conversationID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "消息预加载完成"})
}

// CloneBotConfig 克隆机器人配置
func (ctrl *AIAssistantController) CloneBotConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newConfig, err := ctrl.aiService.CloneConfig(uint(id), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("克隆配置失败: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "机器人配置克隆成功",
		"data": newConfig,
	})
}

// BatchUpdateBots 批量更新机器人配置
func (ctrl *AIAssistantController) BatchUpdateBots(c *gin.Context) {
	var req struct {
		ConfigIDs []uint                 `json:"config_ids" binding:"required"`
		Updates   map[string]interface{} `json:"updates" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.aiService.BatchUpdateConfigs(req.ConfigIDs, req.Updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("批量更新失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "批量更新成功"})
}
