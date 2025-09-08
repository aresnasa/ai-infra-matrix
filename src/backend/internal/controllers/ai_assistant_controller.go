package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// AIAssistantController AI助手控制器
type AIAssistantController struct {
	aiService           services.AIService
	messageQueueService services.MessageQueueService
	cacheService        services.CacheService
}

// NewAIAssistantController 创建AI助手控制器
func NewAIAssistantController() *AIAssistantController {
	return &AIAssistantController{
		aiService:           services.NewAIService(),
		messageQueueService: services.NewMessageQueueService(),
		cacheService:        services.NewCacheService(),
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
		ConfigID uint   `json:"config_id" binding:"required"`
		Title    string `json:"title"`
		Context  string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title == "" {
		req.Title = "新对话"
	}

	conversation, err := ctrl.aiService.CreateConversation(userID, req.ConfigID, req.Title, req.Context)
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

// QuickChat 快速聊天（异步版本）
func (ctrl *AIAssistantController) QuickChat(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
		Context string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 异步发送到队列（无对话ID，将自动创建新对话）
	messageID, err := ctrl.messageQueueService.SendChatRequest(
		userID, 
		nil, 
		req.Message, 
		map[string]interface{}{
			"page": req.Context,
			"type": "quick_chat",
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "快速聊天失败"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message_id": messageID,
		"status": "pending",
		"message": "正在处理您的请求",
	})
}

// GetMessageStatus 获取消息处理状态
func (ctrl *AIAssistantController) GetMessageStatus(c *gin.Context) {
	messageID := c.Param("message_id")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息ID不能为空"})
		return
	}

	status, err := ctrl.messageQueueService.GetMessageStatus(messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息状态不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
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

	// 先从缓存获取消息
	cacheKey := fmt.Sprintf("messages:%d", conversationID)
	cachedMessages := ctrl.cacheService.GetMessages(cacheKey)
	
	if cachedMessages != nil {
		c.JSON(http.StatusOK, gin.H{"data": cachedMessages, "from_cache": true})
		return
	}

	// 缓存未命中，从数据库获取
	messages, err := ctrl.aiService.GetMessages(uint(conversationID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 缓存结果
	ctrl.cacheService.SetMessages(cacheKey, messages, 24*time.Hour)

	c.JSON(http.StatusOK, gin.H{"data": messages, "from_cache": false})
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
		ConfigID uint   `json:"config_id" binding:"required"`
		Message  string `json:"message" binding:"required"`
		Context  string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 异步处理快速聊天
	messageID, err := ctrl.messageQueueService.SendQuickChatRequest(
		userID,
		req.ConfigID,
		req.Message,
		req.Context,
		map[string]interface{}{
			"page": c.GetHeader("Referer"),
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

// GetBotCategories 获取机器人分类
func (ctrl *AIAssistantController) GetBotCategories(c *gin.Context) {
	categories := []map[string]interface{}{
		{"key": "general", "name": "通用对话", "description": "适用于日常对话和一般问题"},
		{"key": "coding", "name": "代码生成", "description": "专业的编程助手"},
		{"key": "writing", "name": "写作助手", "description": "帮助写作和内容创作"},
		{"key": "analysis", "name": "数据分析", "description": "数据分析和可视化"},
		{"key": "translation", "name": "翻译助手", "description": "多语言翻译服务"},
		{"key": "custom", "name": "自定义", "description": "自定义配置的机器人"},
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
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
