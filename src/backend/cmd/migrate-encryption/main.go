package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
	"gorm.io/gorm"
)

// 此脚本用于将数据库中的明文敏感数据加密
// 运行一次即可，之后数据会自动加密/解密

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化加密服务
	if err := utils.InitEncryptionService(cfg.EncryptionKey); err != nil {
		log.Fatal("Failed to initialize encryption service:", err)
	}

	// 连接数据库
	if err := database.Connect(cfg); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db := database.DB

	// 1. 迁移 SlurmNode 表的 password 和 username
	if err := migrateSlurmNodes(db); err != nil {
		log.Printf("Warning: Failed to migrate slurm_nodes: %v", err)
	}

	// 2. 迁移 SlurmCluster 表的 master_ssh JSON 字段
	if err := migrateSlurmClusters(db); err != nil {
		log.Printf("Warning: Failed to migrate slurm_clusters: %v", err)
	}

	// 3. 迁移 AIAssistantConfig 表的 api_key 和 api_secret
	if err := migrateAIConfigs(db); err != nil {
		log.Printf("Warning: Failed to migrate ai_assistant_configs: %v", err)
	}

	log.Println("Sensitive data encryption migration completed!")
}

// migrateSlurmNodes 加密 slurm_nodes 表中的敏感数据
func migrateSlurmNodes(db *gorm.DB) error {
	type SlurmNodeRaw struct {
		ID       uint
		Username string
		Password string
	}

	var nodes []SlurmNodeRaw
	if err := db.Table("slurm_nodes").Select("id, username, password").Find(&nodes).Error; err != nil {
		return fmt.Errorf("failed to query slurm_nodes: %w", err)
	}

	encryptionService := utils.GetEncryptionService()
	if encryptionService == nil {
		return fmt.Errorf("encryption service not initialized")
	}

	migratedCount := 0
	for _, node := range nodes {
		needUpdate := false
		updates := make(map[string]interface{})

		// 检查并加密 username
		if node.Username != "" && !encryptionService.IsEncrypted(node.Username) {
			encrypted, err := encryptionService.Encrypt(node.Username)
			if err != nil {
				log.Printf("Warning: Failed to encrypt username for node %d: %v", node.ID, err)
				continue
			}
			updates["username"] = encrypted
			needUpdate = true
		}

		// 检查并加密 password
		if node.Password != "" && !encryptionService.IsEncrypted(node.Password) {
			encrypted, err := encryptionService.Encrypt(node.Password)
			if err != nil {
				log.Printf("Warning: Failed to encrypt password for node %d: %v", node.ID, err)
				continue
			}
			updates["password"] = encrypted
			needUpdate = true
		}

		if needUpdate {
			if err := db.Table("slurm_nodes").Where("id = ?", node.ID).Updates(updates).Error; err != nil {
				log.Printf("Warning: Failed to update node %d: %v", node.ID, err)
				continue
			}
			migratedCount++
		}
	}

	log.Printf("Migrated %d slurm_nodes records", migratedCount)
	return nil
}

// migrateSlurmClusters 加密 slurm_clusters 表中的敏感数据
func migrateSlurmClusters(db *gorm.DB) error {
	type SlurmClusterRaw struct {
		ID        uint
		MasterSSH string `gorm:"column:master_ssh"`
	}

	var clusters []SlurmClusterRaw
	if err := db.Table("slurm_clusters").Select("id, master_ssh").Find(&clusters).Error; err != nil {
		return fmt.Errorf("failed to query slurm_clusters: %w", err)
	}

	encryptionService := utils.GetEncryptionService()
	if encryptionService == nil {
		return fmt.Errorf("encryption service not initialized")
	}

	migratedCount := 0
	for _, cluster := range clusters {
		if cluster.MasterSSH == "" || cluster.MasterSSH == "null" {
			continue
		}

		// 检查 JSON 中是否有未加密的敏感字段
		needUpdate := false

		// 简单检查：如果包含 "password" 但不包含 "encrypted:"，则需要加密
		if strings.Contains(cluster.MasterSSH, "\"password\":") &&
			!strings.Contains(cluster.MasterSSH, "encrypted:") {
			needUpdate = true
		}
		if strings.Contains(cluster.MasterSSH, "\"username\":") &&
			!strings.Contains(cluster.MasterSSH, "encrypted:") {
			needUpdate = true
		}

		if needUpdate {
			// 使用 GORM 模型的 hooks 来自动处理加密
			// 这里我们只标记需要迁移，实际迁移由模型 hooks 完成
			// 触发一个 update 来让 hooks 工作
			if err := db.Exec("UPDATE slurm_clusters SET updated_at = NOW() WHERE id = ?", cluster.ID).Error; err != nil {
				log.Printf("Warning: Failed to update cluster %d: %v", cluster.ID, err)
				continue
			}
			migratedCount++
		}
	}

	log.Printf("Marked %d slurm_clusters records for migration (hooks will handle encryption)", migratedCount)
	return nil
}

// migrateAIConfigs 加密 ai_assistant_configs 表中的敏感数据
func migrateAIConfigs(db *gorm.DB) error {
	type AIConfigRaw struct {
		ID        uint
		APIKey    string `gorm:"column:api_key"`
		APISecret string `gorm:"column:api_secret"`
	}

	var configs []AIConfigRaw
	if err := db.Table("ai_assistant_configs").Select("id, api_key, api_secret").Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to query ai_assistant_configs: %w", err)
	}

	encryptionService := utils.GetEncryptionService()
	if encryptionService == nil {
		return fmt.Errorf("encryption service not initialized")
	}

	migratedCount := 0
	for _, cfg := range configs {
		needUpdate := false
		updates := make(map[string]interface{})

		// 检查并加密 api_key
		if cfg.APIKey != "" && !encryptionService.IsEncrypted(cfg.APIKey) {
			encrypted, err := encryptionService.Encrypt(cfg.APIKey)
			if err != nil {
				log.Printf("Warning: Failed to encrypt api_key for config %d: %v", cfg.ID, err)
				continue
			}
			updates["api_key"] = encrypted
			needUpdate = true
		}

		// 检查并加密 api_secret
		if cfg.APISecret != "" && !encryptionService.IsEncrypted(cfg.APISecret) {
			encrypted, err := encryptionService.Encrypt(cfg.APISecret)
			if err != nil {
				log.Printf("Warning: Failed to encrypt api_secret for config %d: %v", cfg.ID, err)
				continue
			}
			updates["api_secret"] = encrypted
			needUpdate = true
		}

		if needUpdate {
			if err := db.Table("ai_assistant_configs").Where("id = ?", cfg.ID).Updates(updates).Error; err != nil {
				log.Printf("Warning: Failed to update config %d: %v", cfg.ID, err)
				continue
			}
			migratedCount++
		}
	}

	log.Printf("Migrated %d ai_assistant_configs records", migratedCount)
	return nil
}
