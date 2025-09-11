package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// CacheService 缓存服务接口
type CacheService interface {
	// 对话缓存
	SetConversation(key string, conversation *models.AIConversation, duration time.Duration) error
	GetConversation(key string) *models.AIConversation
	DeleteConversation(key string) error

	// 消息缓存
	SetMessages(key string, messages []models.AIMessage, duration time.Duration) error
	GetMessages(key string) []models.AIMessage
	AppendMessage(key string, message *models.AIMessage) error
	DeleteMessages(key string) error
	
	// 单个消息缓存
	SetMessage(key string, message *models.AIMessage, duration time.Duration) error
	GetMessage(key string) *models.AIMessage
	Delete(key string) error

	// 配置缓存
	SetConfigs(configs []models.AIAssistantConfig, duration time.Duration) error
	GetConfigs() []models.AIAssistantConfig
	SetDefaultConfig(config *models.AIAssistantConfig, duration time.Duration) error
	GetDefaultConfig() *models.AIAssistantConfig

	// 用户会话缓存
	SetUserSession(userID uint, sessionData map[string]interface{}, duration time.Duration) error
	GetUserSession(userID uint) map[string]interface{}

	// 请求去重
	SetRequestDeduplication(userID uint, messageHash string, duration time.Duration) error
	IsRequestDuplicated(userID uint, messageHash string) bool

	// 热点数据预热
	WarmupCache() error
	
	// 健康检查
	HealthCheck() error
}

// cacheServiceImpl 缓存服务实现
type cacheServiceImpl struct {
	redis *redis.Client
	ctx   context.Context
}

// NewCacheService 创建缓存服务
func NewCacheService() CacheService {
	return &cacheServiceImpl{
		redis: cache.RDB,
		ctx:   context.Background(),
	}
}

// SetConversation 设置对话缓存
func (c *cacheServiceImpl) SetConversation(key string, conversation *models.AIConversation, duration time.Duration) error {
	data, err := json.Marshal(conversation)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %v", err)
	}

	return c.redis.Set(c.ctx, key, data, duration).Err()
}

// GetConversation 获取对话缓存
func (c *cacheServiceImpl) GetConversation(key string) *models.AIConversation {
	data, err := c.redis.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Errorf("Failed to get conversation from cache: %v", err)
		}
		return nil
	}

	var conversation models.AIConversation
	if err := json.Unmarshal([]byte(data), &conversation); err != nil {
		logrus.Errorf("Failed to unmarshal conversation: %v", err)
		return nil
	}

	return &conversation
}

// DeleteConversation 删除对话缓存
func (c *cacheServiceImpl) DeleteConversation(key string) error {
	return c.redis.Del(c.ctx, key).Err()
}

// SetMessages 设置消息缓存
func (c *cacheServiceImpl) SetMessages(key string, messages []models.AIMessage, duration time.Duration) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %v", err)
	}

	return c.redis.Set(c.ctx, key, data, duration).Err()
}

// GetMessages 获取消息缓存
func (c *cacheServiceImpl) GetMessages(key string) []models.AIMessage {
	data, err := c.redis.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Errorf("Failed to get messages from cache: %v", err)
		}
		return nil
	}

	var messages []models.AIMessage
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		logrus.Errorf("Failed to unmarshal messages: %v", err)
		return nil
	}

	return messages
}

// AppendMessage 追加消息到缓存
func (c *cacheServiceImpl) AppendMessage(key string, message *models.AIMessage) error {
	// 获取现有消息
	messages := c.GetMessages(key)
	if messages == nil {
		messages = []models.AIMessage{}
	}

	// 追加新消息
	messages = append(messages, *message)

	// 限制消息数量（保留最近100条）
	if len(messages) > 100 {
		messages = messages[len(messages)-100:]
	}

	// 重新设置缓存
	return c.SetMessages(key, messages, 24*time.Hour)
}

// DeleteMessages 删除消息缓存
func (c *cacheServiceImpl) DeleteMessages(key string) error {
	return c.redis.Del(c.ctx, key).Err()
}

// SetConfigs 设置配置缓存
func (c *cacheServiceImpl) SetConfigs(configs []models.AIAssistantConfig, duration time.Duration) error {
	data, err := json.Marshal(configs)
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %v", err)
	}

	return c.redis.Set(c.ctx, "ai:configs", data, duration).Err()
}

// GetConfigs 获取配置缓存
func (c *cacheServiceImpl) GetConfigs() []models.AIAssistantConfig {
	data, err := c.redis.Get(c.ctx, "ai:configs").Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Errorf("Failed to get configs from cache: %v", err)
		}
		return nil
	}

	var configs []models.AIAssistantConfig
	if err := json.Unmarshal([]byte(data), &configs); err != nil {
		logrus.Errorf("Failed to unmarshal configs: %v", err)
		return nil
	}

	return configs
}

// SetDefaultConfig 设置默认配置缓存
func (c *cacheServiceImpl) SetDefaultConfig(config *models.AIAssistantConfig, duration time.Duration) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %v", err)
	}

	return c.redis.Set(c.ctx, "ai:default_config", data, duration).Err()
}

// GetDefaultConfig 获取默认配置缓存
func (c *cacheServiceImpl) GetDefaultConfig() *models.AIAssistantConfig {
	data, err := c.redis.Get(c.ctx, "ai:default_config").Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Errorf("Failed to get default config from cache: %v", err)
		}
		return nil
	}

	var config models.AIAssistantConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		logrus.Errorf("Failed to unmarshal default config: %v", err)
		return nil
	}

	return &config
}

// SetUserSession 设置用户会话缓存
func (c *cacheServiceImpl) SetUserSession(userID uint, sessionData map[string]interface{}, duration time.Duration) error {
	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %v", err)
	}

	key := fmt.Sprintf("user_session:%d", userID)
	return c.redis.Set(c.ctx, key, data, duration).Err()
}

// GetUserSession 获取用户会话缓存
func (c *cacheServiceImpl) GetUserSession(userID uint) map[string]interface{} {
	key := fmt.Sprintf("user_session:%d", userID)
	data, err := c.redis.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Errorf("Failed to get user session from cache: %v", err)
		}
		return nil
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
		logrus.Errorf("Failed to unmarshal session data: %v", err)
		return nil
	}

	return sessionData
}

// SetRequestDeduplication 设置请求去重标记
func (c *cacheServiceImpl) SetRequestDeduplication(userID uint, messageHash string, duration time.Duration) error {
	key := fmt.Sprintf("dedup:%d:%s", userID, messageHash)
	return c.redis.Set(c.ctx, key, "1", duration).Err()
}

// IsRequestDuplicated 检查请求是否重复
func (c *cacheServiceImpl) IsRequestDuplicated(userID uint, messageHash string) bool {
	key := fmt.Sprintf("dedup:%d:%s", userID, messageHash)
	_, err := c.redis.Get(c.ctx, key).Result()
	return err != redis.Nil
}

// WarmupCache 预热缓存
func (c *cacheServiceImpl) WarmupCache() error {
	logrus.Info("Starting cache warmup...")

	// 预热AI配置
	aiService := NewAIService()
	configs, err := aiService.ListConfigs()
	if err != nil {
		logrus.Errorf("Failed to load configs for warmup: %v", err)
	} else {
		c.SetConfigs(configs, time.Hour)
		
		// 查找并缓存默认配置
		for _, config := range configs {
			if config.IsDefault && config.IsEnabled {
				c.SetDefaultConfig(&config, time.Hour)
				break
			}
		}
	}

	// 预热热点对话数据（最近活跃的对话）
	// 这里可以根据实际需求实现更复杂的预热逻辑

	logrus.Info("Cache warmup completed")
	return nil
}

// HealthCheck 健康检查
func (c *cacheServiceImpl) HealthCheck() error {
	// 测试Redis连接
	_, err := c.redis.Ping(c.ctx).Result()
	if err != nil {
		return fmt.Errorf("redis connection failed: %v", err)
	}

	// 测试基本读写操作
	testKey := "health_check_test"
	testValue := "ok"
	
	err = c.redis.Set(c.ctx, testKey, testValue, time.Minute).Err()
	if err != nil {
		return fmt.Errorf("redis write test failed: %v", err)
	}

	result, err := c.redis.Get(c.ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("redis read test failed: %v", err)
	}

	if result != testValue {
		return fmt.Errorf("redis read/write test failed: expected %s, got %s", testValue, result)
	}

	// 清理测试数据
	c.redis.Del(c.ctx, testKey)

	return nil
}

// 缓存统计信息
type CacheStats struct {
	HitRate           float64 `json:"hit_rate"`
	MissRate          float64 `json:"miss_rate"`
	TotalRequests     int64   `json:"total_requests"`
	CacheHits         int64   `json:"cache_hits"`
	CacheMisses       int64   `json:"cache_misses"`
	MemoryUsage       int64   `json:"memory_usage"`
	KeyCount          int64   `json:"key_count"`
	ExpiredKeyCount   int64   `json:"expired_key_count"`
}

// GetCacheStats 获取缓存统计信息
func (c *cacheServiceImpl) GetCacheStats() (*CacheStats, error) {
	info, err := c.redis.Info(c.ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis stats: %v", err)
	}

	memory, err := c.redis.Info(c.ctx, "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis memory info: %v", err)
	}

	// 解析Redis INFO输出（简化版本）
	stats := &CacheStats{
		// 这里需要解析Redis INFO的输出，提取相关统计信息
		// 为了简化，这里使用模拟数据
		HitRate:         0.85,
		MissRate:        0.15,
		TotalRequests:   1000,
		CacheHits:       850,
		CacheMisses:     150,
		MemoryUsage:     1024 * 1024, // 1MB
		KeyCount:        100,
		ExpiredKeyCount: 10,
	}

	logrus.Debugf("Redis Info - Stats: %s", info)
	logrus.Debugf("Redis Info - Memory: %s", memory)

	return stats, nil
}

// SetMessage 设置单个消息缓存
func (c *cacheServiceImpl) SetMessage(key string, message *models.AIMessage, duration time.Duration) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return c.redis.Set(c.ctx, key, data, duration).Err()
}

// GetMessage 获取单个消息缓存
func (c *cacheServiceImpl) GetMessage(key string) *models.AIMessage {
	data, err := c.redis.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Errorf("Failed to get message from cache: %v", err)
		}
		return nil
	}

	var message models.AIMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		logrus.Errorf("Failed to unmarshal message: %v", err)
		return nil
	}

	return &message
}

// Delete 删除缓存键
func (c *cacheServiceImpl) Delete(key string) error {
	return c.redis.Del(c.ctx, key).Err()
}
