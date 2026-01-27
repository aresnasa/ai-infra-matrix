package database

import (
	"fmt"
	"os"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// SeedDefaultData 初始化默认数据
func SeedDefaultData() error {
	logrus.Info("Seeding default data...")

	// 初始化默认对象存储配置
	if err := seedDefaultObjectStorageConfig(); err != nil {
		logrus.WithError(err).Error("Failed to seed default object storage config")
		return err
	}

	logrus.Info("Default data seeded successfully")
	return nil
}

// getSeaweedFSConfigFromEnv 从环境变量读取 SeaweedFS 配置
// 返回的 accessKey 和 secretKey 已加密（如果加密服务可用）
func getSeaweedFSConfigFromEnv() (endpoint, filerURL, masterURL, accessKey, secretKey string, sslEnabled bool) {
	seaweedFilerHost := os.Getenv("SEAWEEDFS_FILER_HOST")
	if seaweedFilerHost == "" {
		seaweedFilerHost = "seaweedfs-filer"
	}

	seaweedFilerPort := os.Getenv("SEAWEEDFS_FILER_PORT")
	if seaweedFilerPort == "" {
		seaweedFilerPort = "8888"
	}

	seaweedMasterHost := os.Getenv("SEAWEEDFS_MASTER_HOST")
	if seaweedMasterHost == "" {
		seaweedMasterHost = "seaweedfs-master"
	}

	seaweedMasterPort := os.Getenv("SEAWEEDFS_MASTER_PORT")
	if seaweedMasterPort == "" {
		seaweedMasterPort = "9333"
	}

	seaweedS3Port := os.Getenv("SEAWEEDFS_S3_PORT")
	if seaweedS3Port == "" {
		seaweedS3Port = "8333"
	}

	// 读取原始凭据
	rawAccessKey := os.Getenv("SEAWEEDFS_ACCESS_KEY")
	if rawAccessKey == "" {
		rawAccessKey = "seaweedfs_admin"
	}

	rawSecretKey := os.Getenv("SEAWEEDFS_SECRET_KEY")
	if rawSecretKey == "" {
		rawSecretKey = "seaweedfs_secret_key_change_me"
	}

	// 加密凭据（SECURITY: 凭据不应明文存储在数据库中）
	if CryptoService != nil {
		var err error
		accessKey, err = CryptoService.EncryptIfNeeded(rawAccessKey)
		if err != nil {
			logrus.WithError(err).Warn("Failed to encrypt access key, using raw value")
			accessKey = rawAccessKey
		}
		secretKey, err = CryptoService.EncryptIfNeeded(rawSecretKey)
		if err != nil {
			logrus.WithError(err).Warn("Failed to encrypt secret key, using raw value")
			secretKey = rawSecretKey
		}
	} else {
		logrus.Warn("CryptoService not available, credentials will be stored unencrypted")
		accessKey = rawAccessKey
		secretKey = rawSecretKey
	}

	sslEnabled = false
	if v := os.Getenv("SEAWEEDFS_USE_SSL"); v != "" {
		if v == "1" || v == "true" || v == "TRUE" || v == "True" {
			sslEnabled = true
		}
	}

	// 根据 SSL 设置确定 URL 前缀
	scheme := "http"
	if sslEnabled {
		scheme = "https"
	}

	endpoint = fmt.Sprintf("%s://%s:%s", scheme, seaweedFilerHost, seaweedS3Port)
	filerURL = fmt.Sprintf("%s://%s:%s", scheme, seaweedFilerHost, seaweedFilerPort)
	masterURL = fmt.Sprintf("%s://%s:%s", scheme, seaweedMasterHost, seaweedMasterPort)

	return
}

// seedDefaultObjectStorageConfig 初始化默认的对象存储配置
// 如果配置已存在，会同步更新凭据（AK/SK）以保持与环境变量一致
func seedDefaultObjectStorageConfig() error {
	// 从环境变量读取 SeaweedFS 配置
	endpoint, filerURL, masterURL, accessKey, secretKey, sslEnabled := getSeaweedFSConfigFromEnv()

	// 检查是否已存在默认的 SeaweedFS 配置
	var existingConfig models.ObjectStorageConfig
	err := DB.Where("type = ? AND (name LIKE ? OR name LIKE ?)",
		"seaweedfs", "%默认%", "%Default%").First(&existingConfig).Error

	if err == nil {
		// 配置已存在，同步更新凭据和连接信息
		return syncDefaultSeaweedFSConfig(&existingConfig, endpoint, filerURL, masterURL, accessKey, secretKey, sslEnabled)
	}

	// 检查是否有任何 SeaweedFS 类型的配置（可能用户手动创建的）
	var seaweedfsConfig models.ObjectStorageConfig
	err = DB.Where("type = ?", "seaweedfs").First(&seaweedfsConfig).Error
	if err == nil {
		// 存在 SeaweedFS 配置，同步凭据
		logrus.Info("Found existing SeaweedFS config, syncing credentials from environment")
		return syncDefaultSeaweedFSConfig(&seaweedfsConfig, endpoint, filerURL, masterURL, accessKey, secretKey, sslEnabled)
	}

	// 不存在任何配置，创建新的默认配置
	return createDefaultSeaweedFSConfig(endpoint, filerURL, masterURL, accessKey, secretKey, sslEnabled)
}

// syncDefaultSeaweedFSConfig 同步更新已存在的 SeaweedFS 配置
// 只更新凭据和连接信息，保护其他用户自定义的字段
// 注意：传入的 accessKey 和 secretKey 已加密
func syncDefaultSeaweedFSConfig(config *models.ObjectStorageConfig, endpoint, filerURL, masterURL, accessKey, secretKey string, sslEnabled bool) error {
	// 准备需要更新的字段（凭据已加密）
	updates := map[string]interface{}{
		"endpoint":    endpoint,
		"filer_url":   filerURL,
		"master_url":  masterURL,
		"access_key":  accessKey,
		"secret_key":  secretKey,
		"ssl_enabled": sslEnabled,
		"status":      "unknown", // 重置状态，等待健康检查更新
	}

	// 使用事务更新
	if err := DB.Model(config).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to sync SeaweedFS config credentials: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"config_id":         config.ID,
		"config_name":       config.Name,
		"endpoint":          endpoint,
		"filer_url":         filerURL,
		"master_url":        masterURL,
		"credentials_encrypted": CryptoService != nil,
	}).Info("Synced SeaweedFS config credentials from environment variables")

	return nil
}

// createDefaultSeaweedFSConfig 创建默认的 SeaweedFS 配置
func createDefaultSeaweedFSConfig(endpoint, filerURL, masterURL, accessKey, secretKey string, sslEnabled bool) error {
	defaultConfig := &models.ObjectStorageConfig{
		Name:        "SeaweedFS (Default)",
		Type:        "seaweedfs",
		Endpoint:    endpoint,
		FilerURL:    filerURL,
		MasterURL:   masterURL,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		SSLEnabled:  sslEnabled,
		IsActive:    true,
		Status:      "unknown",
		Description: "Auto-configured SeaweedFS storage",
		CreatedBy:   1,
	}

	// 创建配置
	result := DB.Create(defaultConfig)
	if result.Error != nil {
		// 如果是外键约束错误（用户不存在），尝试创建没有创建者的配置
		if isConstraintError(result.Error) {
			logrus.Warn("Admin user not found, creating object storage config without creator")
			defaultConfig.CreatedBy = 0
			result = DB.Omit("created_by").Create(defaultConfig)
		}

		if result.Error != nil {
			return fmt.Errorf("failed to create default object storage config: %w", result.Error)
		}
	}

	logrus.WithFields(logrus.Fields{
		"name":       defaultConfig.Name,
		"type":       defaultConfig.Type,
		"endpoint":   defaultConfig.Endpoint,
		"filer_url":  defaultConfig.FilerURL,
		"master_url": defaultConfig.MasterURL,
	}).Info("Default SeaweedFS object storage configuration created")

	return nil
}

// isConstraintError 检查是否为约束错误（简单实现）
func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL外键约束错误
	return contains(errStr, "violates foreign key constraint") ||
		contains(errStr, "FOREIGN KEY constraint failed") ||
		contains(errStr, "foreign key constraint")
}

// contains 检查字符串是否包含子字符串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(containsAt(s, substr, 0) || contains(s[1:], substr)))
}

// containsAt 检查字符串在指定位置是否包含子字符串
func containsAt(s, substr string, pos int) bool {
	if pos+len(substr) > len(s) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		if toLower(s[pos+i]) != toLower(substr[i]) {
			return false
		}
	}
	return true
}

// toLower 转换字符为小写
func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
