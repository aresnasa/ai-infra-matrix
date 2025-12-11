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

// seedDefaultObjectStorageConfig 初始化默认的对象存储配置
func seedDefaultObjectStorageConfig() error {
	// 检查是否已存在对象存储配置
	var count int64
	if err := DB.Model(&models.ObjectStorageConfig{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count object storage configs: %w", err)
	}

	// 如果已存在配置，跳过初始化
	if count > 0 {
		logrus.Debug("Object storage configurations already exist, skipping initialization")
		return nil
	}

	// 从环境变量读取 SeaweedFS 配置
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

	accessKey := os.Getenv("SEAWEEDFS_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "seaweedfs_admin"
	}

	secretKey := os.Getenv("SEAWEEDFS_SECRET_KEY")
	if secretKey == "" {
		secretKey = "seaweedfs_secret_key_change_me"
	}

	sslEnabled := false
	if v := os.Getenv("SEAWEEDFS_USE_SSL"); v != "" {
		if v == "1" || v == "true" || v == "TRUE" || v == "True" {
			sslEnabled = true
		}
	}

	// 创建默认 SeaweedFS 配置
	defaultConfig := &models.ObjectStorageConfig{
		Name:        "默认SeaweedFS存储",
		Type:        "seaweedfs",
		Endpoint:    fmt.Sprintf("%s:%s", seaweedFilerHost, seaweedS3Port), // S3 API 端点
		FilerURL:    fmt.Sprintf("%s:%s", seaweedFilerHost, seaweedFilerPort),
		MasterURL:   fmt.Sprintf("%s:%s", seaweedMasterHost, seaweedMasterPort),
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		SSLEnabled:  sslEnabled,
		IsActive:    true,
		Status:      "unknown",
		Description: "系统默认的SeaweedFS对象存储配置",
		CreatedBy:   1, // 假设admin用户ID为1，如果不存在会在外键约束中处理
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
