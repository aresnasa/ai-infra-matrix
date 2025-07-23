package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AIGatewayService AI网关服务
type AIGatewayService struct {
	messageQueueService MessageQueueService
	aiMessageProcessor  *AIMessageProcessor
	cacheService        CacheService
	
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.RWMutex
}

// NewAIGatewayService 创建AI网关服务
func NewAIGatewayService() *AIGatewayService {
	ctx, cancel := context.WithCancel(context.Background())
	
	messageQueueService := NewMessageQueueService()
	aiService := NewAIService()
	cacheService := NewCacheService()
	
	aiMessageProcessor := NewAIMessageProcessor(aiService, messageQueueService)
	
	return &AIGatewayService{
		messageQueueService: messageQueueService,
		aiMessageProcessor:  aiMessageProcessor,
		cacheService:        cacheService,
		ctx:                 ctx,
		cancel:              cancel,
		isRunning:           false,
	}
}

// Start 启动AI网关服务
func (g *AIGatewayService) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.isRunning {
		return fmt.Errorf("AI gateway service is already running")
	}
	
	logrus.Info("Starting AI Gateway Service...")
	
	// 1. 预热缓存
	if err := g.cacheService.WarmupCache(); err != nil {
		logrus.Warnf("Cache warmup failed: %v", err)
	}
	
	// 2. 启动消息处理器
	if err := g.aiMessageProcessor.Start(); err != nil {
		return fmt.Errorf("failed to start message processor: %v", err)
	}
	
	// 3. 启动健康检查协程
	g.wg.Add(1)
	go g.healthCheckWorker()
	
	// 4. 启动缓存清理协程
	g.wg.Add(1)
	go g.cacheCleanupWorker()
	
	// 5. 启动性能监控协程
	g.wg.Add(1)
	go g.performanceMonitorWorker()
	
	g.isRunning = true
	logrus.Info("AI Gateway Service started successfully")
	
	return nil
}

// Stop 停止AI网关服务
func (g *AIGatewayService) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if !g.isRunning {
		return fmt.Errorf("AI gateway service is not running")
	}
	
	logrus.Info("Stopping AI Gateway Service...")
	
	// 取消上下文，停止所有协程
	g.cancel()
	
	// 等待所有协程结束
	g.wg.Wait()
	
	g.isRunning = false
	logrus.Info("AI Gateway Service stopped successfully")
	
	return nil
}

// IsRunning 检查服务是否运行中
func (g *AIGatewayService) IsRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isRunning
}

// GetStatus 获取服务状态
func (g *AIGatewayService) GetStatus() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	status := map[string]interface{}{
		"is_running": g.isRunning,
		"timestamp": time.Now(),
		"components": map[string]interface{}{},
	}
	
	// 检查各组件状态
	if g.messageQueueService != nil {
		if err := g.messageQueueService.HealthCheck(); err != nil {
			status["components"].(map[string]interface{})["message_queue"] = map[string]interface{}{
				"status": "unhealthy",
				"error": err.Error(),
			}
		} else {
			status["components"].(map[string]interface{})["message_queue"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}
	
	if g.cacheService != nil {
		if err := g.cacheService.HealthCheck(); err != nil {
			status["components"].(map[string]interface{})["cache"] = map[string]interface{}{
				"status": "unhealthy",
				"error": err.Error(),
			}
		} else {
			status["components"].(map[string]interface{})["cache"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}
	
	return status
}

// healthCheckWorker 健康检查工作协程
func (g *AIGatewayService) healthCheckWorker() {
	defer g.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-g.ctx.Done():
			logrus.Info("Health check worker stopped")
			return
		case <-ticker.C:
			g.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (g *AIGatewayService) performHealthCheck() {
	// 检查消息队列健康状态
	if err := g.messageQueueService.HealthCheck(); err != nil {
		logrus.Errorf("Message queue health check failed: %v", err)
		// 这里可以添加告警逻辑
	}
	
	// 检查缓存健康状态
	if err := g.cacheService.HealthCheck(); err != nil {
		logrus.Errorf("Cache health check failed: %v", err)
		// 这里可以添加告警逻辑
	}
	
	logrus.Debug("Health check completed")
}

// cacheCleanupWorker 缓存清理工作协程
func (g *AIGatewayService) cacheCleanupWorker() {
	defer g.wg.Done()
	
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-g.ctx.Done():
			logrus.Info("Cache cleanup worker stopped")
			return
		case <-ticker.C:
			g.performCacheCleanup()
		}
	}
}

// performCacheCleanup 执行缓存清理
func (g *AIGatewayService) performCacheCleanup() {
	logrus.Info("Starting cache cleanup...")
	
	// 这里可以实现缓存清理逻辑
	// 例如：清理过期的消息状态、临时会话等
	
	logrus.Info("Cache cleanup completed")
}

// performanceMonitorWorker 性能监控工作协程
func (g *AIGatewayService) performanceMonitorWorker() {
	defer g.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-g.ctx.Done():
			logrus.Info("Performance monitor worker stopped")
			return
		case <-ticker.C:
			g.collectPerformanceMetrics()
		}
	}
}

// collectPerformanceMetrics 收集性能指标
func (g *AIGatewayService) collectPerformanceMetrics() {
	// 收集缓存统计信息
	if cacheImpl, ok := g.cacheService.(*cacheServiceImpl); ok {
		stats, err := cacheImpl.GetCacheStats()
		if err != nil {
			logrus.Errorf("Failed to get cache stats: %v", err)
		} else {
			logrus.Infof("Cache Stats - Hit Rate: %.2f%%, Memory Usage: %d bytes", 
				stats.HitRate*100, stats.MemoryUsage)
		}
	}
	
	// 这里可以添加更多性能指标收集
	// 例如：消息队列长度、处理延迟、错误率等
}

// GetMessageQueueService 获取消息队列服务
func (g *AIGatewayService) GetMessageQueueService() MessageQueueService {
	return g.messageQueueService
}

// GetCacheService 获取缓存服务
func (g *AIGatewayService) GetCacheService() CacheService {
	return g.cacheService
}

// 全局AI网关服务实例
var globalAIGateway *AIGatewayService
var gatewayOnce sync.Once

// GetGlobalAIGateway 获取全局AI网关服务实例（单例）
func GetGlobalAIGateway() *AIGatewayService {
	gatewayOnce.Do(func() {
		globalAIGateway = NewAIGatewayService()
	})
	return globalAIGateway
}

// InitializeAIGateway 初始化AI网关服务
func InitializeAIGateway() error {
	gateway := GetGlobalAIGateway()
	return gateway.Start()
}

// ShutdownAIGateway 关闭AI网关服务
func ShutdownAIGateway() error {
	if globalAIGateway != nil {
		return globalAIGateway.Stop()
	}
	return nil
}
