package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

// sync-seaweedfs 工具用于从环境变量读取 SeaweedFS 凭据，写入数据库
// 凭据加密由 GORM 钩子统一处理，不在此工具中手动加密
// 使用方式: ./sync-seaweedfs
// 环境变量:
//   - SEAWEEDFS_ACCESS_KEY: SeaweedFS S3 Access Key
//   - SEAWEEDFS_SECRET_KEY: SeaweedFS S3 Secret Key
//   - SEAWEEDFS_FILER_HOST: Filer 主机名 (默认: seaweedfs-filer)
//   - SEAWEEDFS_FILER_PORT: Filer 端口 (默认: 8888)
//   - SEAWEEDFS_S3_PORT: S3 端口 (默认: 8333)
//   - SEAWEEDFS_MASTER_HOST: Master 主机名 (默认: seaweedfs-master)
//   - SEAWEEDFS_MASTER_PORT: Master 端口 (默认: 9333)

func main() {
	log.Println("🔐 SeaweedFS Credentials Sync Tool")
	log.Println("===================================")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("❌ Failed to load config:", err)
	}

	// 连接数据库（会自动初始化加密服务和注册 GORM 钩子）
	if err := database.Connect(cfg); err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	log.Println("✅ Database connected")

	// 检查加密服务是否可用
	if database.CryptoService != nil {
		log.Println("✅ Encryption service initialized (GORM hooks will handle encryption)")
	} else {
		log.Println("⚠️  Encryption service not available, credentials will be stored unencrypted")
	}

	// 从环境变量读取 SeaweedFS 配置
	accessKey := os.Getenv("SEAWEEDFS_ACCESS_KEY")
	secretKey := os.Getenv("SEAWEEDFS_SECRET_KEY")
	filerHost := getEnvOrDefault("SEAWEEDFS_FILER_HOST", "seaweedfs-filer")
	filerPort := getEnvOrDefault("SEAWEEDFS_FILER_PORT", "8888")
	s3Port := getEnvOrDefault("SEAWEEDFS_S3_PORT", "8333")
	masterHost := getEnvOrDefault("SEAWEEDFS_MASTER_HOST", "seaweedfs-master")
	masterPort := getEnvOrDefault("SEAWEEDFS_MASTER_PORT", "9333")
	region := getEnvOrDefault("SEAWEEDFS_REGION", "us-east-1")

	// 检查必要的凭据
	if accessKey == "" || secretKey == "" {
		log.Fatal("❌ SEAWEEDFS_ACCESS_KEY and SEAWEEDFS_SECRET_KEY must be set")
	}

	log.Printf("📋 SeaweedFS Configuration:")
	log.Printf("   Access Key: %s...", accessKey[:min(8, len(accessKey))])
	log.Printf("   Filer Host: %s:%s", filerHost, filerPort)
	log.Printf("   S3 Port: %s", s3Port)
	log.Printf("   Master: %s:%s", masterHost, masterPort)

	// 构建 URLs
	s3Endpoint := fmt.Sprintf("http://%s:%s", filerHost, s3Port)
	filerURL := fmt.Sprintf("http://%s:%s", filerHost, filerPort)
	masterURL := fmt.Sprintf("http://%s:%s", masterHost, masterPort)

	// 查找现有的 SeaweedFS 配置
	var existingConfig models.ObjectStorageConfig
	err = database.DB.Where("type = ? AND deleted_at IS NULL", "seaweedfs").First(&existingConfig).Error

	if err == nil {
		// 配置已存在，更新
		log.Printf("📝 Updating existing SeaweedFS configuration (ID: %d)", existingConfig.ID)

		// 更新配置字段（明文凭据，GORM 钩子会自动加密）
		now := time.Now()
		existingConfig.Endpoint = s3Endpoint
		existingConfig.FilerURL = filerURL
		existingConfig.MasterURL = masterURL
		existingConfig.Region = region
		existingConfig.AccessKey = accessKey
		existingConfig.SecretKey = secretKey
		existingConfig.Status = "unknown"
		existingConfig.LastTested = &now

		// 使用 Save 触发 GORM 钩子进行加密
		if err := database.DB.Save(&existingConfig).Error; err != nil {
			log.Fatal("❌ Failed to update configuration:", err)
		}
		log.Printf("✅ SeaweedFS configuration updated (ID: %d)", existingConfig.ID)

	} else {
		// 配置不存在，创建新配置
		log.Println("📝 Creating new SeaweedFS configuration...")

		// 创建配置（明文凭据，GORM 钩子会自动加密）
		newConfig := &models.ObjectStorageConfig{
			Name:        "SeaweedFS (Default)",
			Type:        "seaweedfs",
			Endpoint:    s3Endpoint,
			FilerURL:    filerURL,
			MasterURL:   masterURL,
			Region:      region,
			AccessKey:   accessKey,
			SecretKey:   secretKey,
			SSLEnabled:  false,
			Timeout:     30,
			IsActive:    true,
			Status:      "unknown",
			Description: "Auto-configured SeaweedFS storage",
			CreatedBy:   1, // admin user
		}

		if err := database.DB.Create(newConfig).Error; err != nil {
			log.Fatal("❌ Failed to create configuration:", err)
		}
		log.Printf("✅ SeaweedFS configuration created (ID: %d)", newConfig.ID)
	}

	log.Println("")
	log.Println("===================================")
	log.Println("✅ SeaweedFS credentials sync completed!")
	log.Println("   Credentials are encrypted by GORM hooks and stored securely.")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
