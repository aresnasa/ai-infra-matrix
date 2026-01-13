package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ObjectStorageHealthChecker 对象存储健康检查器
type ObjectStorageHealthChecker struct {
	service  *ObjectStorageService
	stopChan chan struct{}
	stopped  bool
	mu       sync.Mutex
}

var (
	objectStorageHealthChecker *ObjectStorageHealthChecker
	objectStorageHealthOnce    sync.Once
)

// ObjectStorageService 对象存储服务
type ObjectStorageService struct {
	db *gorm.DB
}

// NewObjectStorageService 创建对象存储服务实例
func NewObjectStorageService(db *gorm.DB) *ObjectStorageService {
	return &ObjectStorageService{
		db: db,
	}
}

// GetConfigs 获取所有存储配置
func (s *ObjectStorageService) GetConfigs(userID uint) ([]models.ObjectStorageConfig, error) {
	var configs []models.ObjectStorageConfig

	err := s.db.Preload("Creator").Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("获取存储配置失败: %v", err)
	}

	// 更新连接状态
	go s.updateConfigsStatus(configs)

	return configs, nil
}

// GetConfig 获取单个存储配置
func (s *ObjectStorageService) GetConfig(id uint) (*models.ObjectStorageConfig, error) {
	var config models.ObjectStorageConfig

	err := s.db.Preload("Creator").First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("配置不存在")
		}
		return nil, fmt.Errorf("获取配置失败: %v", err)
	}

	return &config, nil
}

// CreateConfig 创建存储配置
func (s *ObjectStorageService) CreateConfig(config *models.ObjectStorageConfig) error {
	// 验证配置
	if err := s.validateConfig(config); err != nil {
		return err
	}

	// 如果是第一个配置，自动设为激活
	var count int64
	s.db.Model(&models.ObjectStorageConfig{}).Count(&count)
	if count == 0 {
		config.IsActive = true
	}

	err := s.db.Create(config).Error
	if err != nil {
		return fmt.Errorf("创建配置失败: %v", err)
	}

	// 异步测试连接
	go s.testAndUpdateStatus(config)

	return nil
}

// UpdateConfig 更新存储配置
func (s *ObjectStorageService) UpdateConfig(id uint, updates *models.ObjectStorageConfig) error {
	var config models.ObjectStorageConfig

	err := s.db.First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("配置不存在")
		}
		return fmt.Errorf("获取配置失败: %v", err)
	}

	// 验证更新的配置
	if err := s.validateConfig(updates); err != nil {
		return err
	}

	err = s.db.Save(updates).Error
	if err != nil {
		return fmt.Errorf("更新配置失败: %v", err)
	}

	// 异步测试连接
	go s.testAndUpdateStatus(updates)

	return nil
}

// DeleteConfig 删除存储配置
func (s *ObjectStorageService) DeleteConfig(id uint) error {
	var config models.ObjectStorageConfig

	err := s.db.First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("配置不存在")
		}
		return fmt.Errorf("获取配置失败: %v", err)
	}

	// 如果是激活配置，不允许删除
	if config.IsActive {
		return fmt.Errorf("不能删除激活的配置")
	}

	err = s.db.Delete(&config).Error
	if err != nil {
		return fmt.Errorf("删除配置失败: %v", err)
	}

	return nil
}

// SetActiveConfig 设置激活配置
func (s *ObjectStorageService) SetActiveConfig(id uint) error {
	var config models.ObjectStorageConfig

	err := s.db.First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("配置不存在")
		}
		return fmt.Errorf("获取配置失败: %v", err)
	}

	// 使用事务确保原子性
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 将所有配置设为非激活（使用 Where("1=1") 满足 GORM 的 WHERE 条件要求）
		if err := tx.Model(&models.ObjectStorageConfig{}).Where("1 = 1").Update("is_active", false).Error; err != nil {
			return err
		}

		// 设置指定配置为激活
		config.IsActive = true
		return tx.Save(&config).Error
	})
}

// TestConnection 测试连接
func (s *ObjectStorageService) TestConnection(config *models.ObjectStorageConfig) error {
	switch config.Type {
	case models.StorageTypeSeaweedFS:
		return s.testSeaweedFSConnection(config)
	case models.StorageTypeMinIO, models.StorageTypeAWSS3:
		// MinIO 和 AWS S3 使用相同的 S3 兼容测试
		return s.testS3CompatibleConnection(config)
	default:
		return fmt.Errorf("不支持的存储类型: %s", config.Type)
	}
}

// CheckConnectionStatus 检查连接状态
func (s *ObjectStorageService) CheckConnectionStatus(id uint) (string, error) {
	config, err := s.GetConfig(id)
	if err != nil {
		return "", err
	}

	// 测试连接并返回状态
	err = s.TestConnection(config)
	if err != nil {
		// 更新数据库状态
		s.db.Model(config).Updates(map[string]interface{}{
			"status":      "error",
			"last_tested": time.Now(),
		})
		return "error", nil
	}

	// 更新数据库状态
	s.db.Model(config).Updates(map[string]interface{}{
		"status":      "connected",
		"last_tested": time.Now(),
	})
	return "connected", nil
}

// GetStatistics 获取存储统计信息
func (s *ObjectStorageService) GetStatistics(id uint) (*models.ObjectStorageStatistics, error) {
	config, err := s.GetConfig(id)
	if err != nil {
		return nil, err
	}

	switch config.Type {
	case models.StorageTypeSeaweedFS:
		return s.getSeaweedFSStatistics(config)
	case models.StorageTypeMinIO, models.StorageTypeAWSS3:
		// MinIO 和 AWS S3 使用相同的 S3 兼容统计
		return s.getS3CompatibleStatistics(config)
	default:
		// 返回空统计信息
		return &models.ObjectStorageStatistics{
			ConfigID: id,
		}, nil
	}
}

// validateConfig 验证配置
func (s *ObjectStorageService) validateConfig(config *models.ObjectStorageConfig) error {
	if config.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}

	if config.Type == "" {
		return fmt.Errorf("存储类型不能为空")
	}

	// SeaweedFS 支持多种端点配置
	if config.Type == models.StorageTypeSeaweedFS {
		// SeaweedFS 至少需要配置 Master 或 Filer 或 S3 端点
		if config.Endpoint == "" && config.MasterURL == "" && config.FilerURL == "" {
			return fmt.Errorf("SeaweedFS 需要配置 S3 端点、Master 地址或 Filer 地址")
		}
		// 如果配置了 S3 端点，需要 AccessKey 和 SecretKey
		if config.Endpoint != "" {
			if config.AccessKey == "" || config.SecretKey == "" {
				return fmt.Errorf("SeaweedFS S3 端点需要配置 AccessKey 和 SecretKey")
			}
		}
		return nil
	}

	// 其他存储类型必须配置端点
	if config.Endpoint == "" {
		return fmt.Errorf("服务端点不能为空")
	}

	if config.AccessKey == "" {
		return fmt.Errorf("访问密钥不能为空")
	}

	if config.SecretKey == "" {
		return fmt.Errorf("访问密钥Secret不能为空")
	}

	return nil
}

// testSeaweedFSConnection 测试 SeaweedFS 连接
func (s *ObjectStorageService) testSeaweedFSConnection(config *models.ObjectStorageConfig) error {
	service, err := NewSeaweedFSService(config)
	if err != nil {
		return fmt.Errorf("创建 SeaweedFS 服务失败: %v", err)
	}

	return service.TestConnection()
}

// getSeaweedFSStatistics 获取 SeaweedFS 统计信息
func (s *ObjectStorageService) getSeaweedFSStatistics(config *models.ObjectStorageConfig) (*models.ObjectStorageStatistics, error) {
	service, err := NewSeaweedFSService(config)
	if err != nil {
		return nil, fmt.Errorf("创建 SeaweedFS 服务失败: %v", err)
	}

	return service.GetStatistics()
}

// testS3CompatibleConnection 测试 S3 兼容存储连接 (MinIO/AWS S3 等)
func (s *ObjectStorageService) testS3CompatibleConnection(config *models.ObjectStorageConfig) error {
	// 创建 S3 兼容客户端
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.SSLEnabled,
	})
	if err != nil {
		return fmt.Errorf("创建 S3 客户端失败: %v", err)
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	// 测试连接 - 尝试列出存储桶
	_, err = client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("S3 连接测试失败: %v", err)
	}

	return nil
}

// testS3Connection 测试S3连接 (已废弃，使用 testS3CompatibleConnection)
func (s *ObjectStorageService) testS3Connection(config *models.ObjectStorageConfig) error {
	return s.testS3CompatibleConnection(config)
}

// testAndUpdateStatus 测试连接并更新状态
func (s *ObjectStorageService) testAndUpdateStatus(config *models.ObjectStorageConfig) {
	err := s.TestConnection(config)

	status := "connected"
	if err != nil {
		status = "error"
	}

	// 更新状态
	s.db.Model(config).Updates(map[string]interface{}{
		"status":      status,
		"last_tested": time.Now(),
	})
}

// updateConfigsStatus 更新所有配置的连接状态
func (s *ObjectStorageService) updateConfigsStatus(configs []models.ObjectStorageConfig) {
	for _, config := range configs {
		// 如果最近没有测试过，或者状态未知，进行测试
		if config.LastTested == nil ||
			time.Since(*config.LastTested) > 5*time.Minute ||
			config.Status == "" {
			s.testAndUpdateStatus(&config)
		}
	}
}

// getS3CompatibleStatistics 获取 S3 兼容存储统计信息 (MinIO/AWS S3 等)
func (s *ObjectStorageService) getS3CompatibleStatistics(config *models.ObjectStorageConfig) (*models.ObjectStorageStatistics, error) {
	// 创建 S3 兼容客户端
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.SSLEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 S3 客户端失败: %v", err)
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	// 获取存储桶列表
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取存储桶列表失败: %v", err)
	}

	stats := &models.ObjectStorageStatistics{
		ConfigID:    config.ID,
		BucketCount: int64(len(buckets)),
		TotalSpace:  "N/A", // S3 API 不直接提供总容量信息
		UsedSpace:   "N/A",
	}

	// 计算对象总数（简单统计，可能较慢）
	var totalObjects int64
	for _, bucket := range buckets {
		objectCh := client.ListObjects(ctx, bucket.Name, minio.ListObjectsOptions{})
		for range objectCh {
			totalObjects++
		}
	}
	stats.ObjectCount = totalObjects

	return stats, nil
}

// StartObjectStorageHealthCheck 启动对象存储健康检查服务
func StartObjectStorageHealthCheck(db *gorm.DB) {
	objectStorageHealthOnce.Do(func() {
		service := NewObjectStorageService(db)
		objectStorageHealthChecker = &ObjectStorageHealthChecker{
			service:  service,
			stopChan: make(chan struct{}),
		}
		go objectStorageHealthChecker.run()
		logrus.Info("[ObjectStorageHealthCheck] Health check service started")
	})
}

// StopObjectStorageHealthCheck 停止对象存储健康检查服务
func StopObjectStorageHealthCheck() {
	if objectStorageHealthChecker != nil {
		objectStorageHealthChecker.mu.Lock()
		defer objectStorageHealthChecker.mu.Unlock()

		if !objectStorageHealthChecker.stopped {
			close(objectStorageHealthChecker.stopChan)
			objectStorageHealthChecker.stopped = true
			logrus.Info("[ObjectStorageHealthCheck] Health check service stopped")
		}
	}
}

// run 运行健康检查循环
func (h *ObjectStorageHealthChecker) run() {
	// 启动后立即进行一次检查
	h.checkAllConfigs()

	// 每 60 秒检查一次
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.checkAllConfigs()
		}
	}
}

// checkAllConfigs 检查所有配置的连接状态
func (h *ObjectStorageHealthChecker) checkAllConfigs() {
	var configs []models.ObjectStorageConfig
	if err := h.service.db.Find(&configs).Error; err != nil {
		logrus.WithError(err).Error("[ObjectStorageHealthCheck] Failed to load configs")
		return
	}

	for _, config := range configs {
		h.checkAndUpdateConfig(&config)
	}
}

// checkAndUpdateConfig 检查并更新单个配置的连接状态
func (h *ObjectStorageHealthChecker) checkAndUpdateConfig(config *models.ObjectStorageConfig) {
	err := h.service.TestConnection(config)

	oldStatus := config.Status
	newStatus := "connected"
	if err != nil {
		newStatus = "disconnected"
	}

	// 只有状态变化时才更新并记录日志
	if oldStatus != newStatus {
		now := time.Now()
		if err := h.service.db.Model(config).Updates(map[string]interface{}{
			"status":      newStatus,
			"last_tested": now,
		}).Error; err != nil {
			logrus.WithError(err).WithField("config_id", config.ID).Error("[ObjectStorageHealthCheck] Failed to update status")
			return
		}

		logFields := logrus.Fields{
			"config_id":   config.ID,
			"config_name": config.Name,
			"old_status":  oldStatus,
			"new_status":  newStatus,
		}
		if newStatus == "connected" {
			logrus.WithFields(logFields).Info("[ObjectStorageHealthCheck] Connection restored")
		} else {
			logrus.WithFields(logFields).WithError(err).Warn("[ObjectStorageHealthCheck] Connection lost")
		}
	} else {
		// 状态未变化，只更新测试时间
		now := time.Now()
		h.service.db.Model(config).Update("last_tested", now)
	}
}
