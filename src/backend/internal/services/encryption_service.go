package services

import (
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EncryptionModelService 模型加密服务
type EncryptionModelService struct{}

// NewEncryptionModelService 创建新的模型加密服务
func NewEncryptionModelService() *EncryptionModelService {
	return &EncryptionModelService{}
}

// EncryptKubernetesCluster 加密Kubernetes集群的敏感数据
func (s *EncryptionModelService) EncryptKubernetesCluster(cluster *models.KubernetesCluster) error {
	if database.CryptoService == nil {
		logrus.Warn("Crypto service not initialized, skipping encryption")
		return nil
	}

	if cluster.KubeConfig != "" {
		encrypted, err := database.CryptoService.Encrypt(cluster.KubeConfig)
		if err != nil {
			logrus.WithError(err).Error("Failed to encrypt KubeConfig")
			return fmt.Errorf("failed to encrypt KubeConfig: %w", err)
		}
		cluster.KubeConfig = encrypted
		logrus.Debug("KubeConfig encrypted successfully")
	}

	return nil
}

// DecryptKubernetesCluster 解密Kubernetes集群的敏感数据
func (s *EncryptionModelService) DecryptKubernetesCluster(cluster *models.KubernetesCluster) error {
	if database.CryptoService == nil {
		logrus.Warn("Crypto service not initialized, skipping decryption")
		return nil
	}

	if cluster.KubeConfig != "" {
		decrypted, err := database.CryptoService.Decrypt(cluster.KubeConfig)
		if err != nil {
			logrus.WithError(err).Error("Failed to decrypt KubeConfig")
			return fmt.Errorf("failed to decrypt KubeConfig: %w", err)
		}
		cluster.KubeConfig = decrypted
		logrus.Debug("KubeConfig decrypted successfully")
	}

	return nil
}

// EncryptUser 加密用户的敏感数据
func (s *EncryptionModelService) EncryptUser(user *models.User) error {
	if database.CryptoService == nil {
		logrus.Warn("Encryption service not initialized, skipping encryption")
		return nil
	}

	// 密码通常已经通过bcrypt哈希，这里不需要额外加密
	// 但如果有其他敏感字段可以在这里处理
	
	return nil
}

// DecryptUser 解密用户的敏感数据
func (s *EncryptionModelService) DecryptUser(user *models.User) error {
	if database.CryptoService == nil {
		logrus.Warn("Encryption service not initialized, skipping decryption")
		return nil
	}

	// 密码通常已经通过bcrypt哈希，这里不需要额外解密
	// 但如果有其他敏感字段可以在这里处理

	return nil
}

// EncryptLDAPConfig 加密LDAP配置的敏感数据
func (s *EncryptionModelService) EncryptLDAPConfig(config *models.LDAPConfig) error {
	if database.CryptoService == nil {
		logrus.Warn("Encryption service not initialized, skipping encryption")
		return nil
	}

	if config.BindPassword != "" {
		encrypted, err := database.CryptoService.EncryptIfNeeded(config.BindPassword)
		if err != nil {
			logrus.WithError(err).Error("Failed to encrypt LDAP bind password")
			return fmt.Errorf("failed to encrypt LDAP bind password: %w", err)
		}
		config.BindPassword = encrypted
		logrus.Debug("LDAP bind password encrypted successfully")
	}

	return nil
}

// DecryptLDAPConfig 解密LDAP配置的敏感数据
func (s *EncryptionModelService) DecryptLDAPConfig(config *models.LDAPConfig) error {
	if database.CryptoService == nil {
		logrus.Warn("Encryption service not initialized, skipping decryption")
		return nil
	}

	if config.BindPassword != "" {
		decrypted, err := database.CryptoService.Decrypt(config.BindPassword)
		if err != nil {
			logrus.WithError(err).Error("Failed to decrypt LDAP bind password")
			return fmt.Errorf("failed to decrypt LDAP bind password: %w", err)
		}
		config.BindPassword = decrypted
		logrus.Debug("LDAP bind password decrypted successfully")
	}

	return nil
}

// KubernetesClusterHooks GORM钩子处理器
type KubernetesClusterHooks struct {
	encryptionService *EncryptionModelService
}

// NewKubernetesClusterHooks 创建新的钩子处理器
func NewKubernetesClusterHooks() *KubernetesClusterHooks {
	return &KubernetesClusterHooks{
		encryptionService: NewEncryptionModelService(),
	}
}

// BeforeCreate 创建前钩子
func (h *KubernetesClusterHooks) BeforeCreate(tx *gorm.DB) error {
	if cluster, ok := tx.Statement.Dest.(*models.KubernetesCluster); ok {
		return h.encryptionService.EncryptKubernetesCluster(cluster)
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (h *KubernetesClusterHooks) BeforeUpdate(tx *gorm.DB) error {
	if cluster, ok := tx.Statement.Dest.(*models.KubernetesCluster); ok {
		return h.encryptionService.EncryptKubernetesCluster(cluster)
	}
	return nil
}

// AfterFind 查询后钩子
func (h *KubernetesClusterHooks) AfterFind(tx *gorm.DB) error {
	if cluster, ok := tx.Statement.Dest.(*models.KubernetesCluster); ok {
		return h.encryptionService.DecryptKubernetesCluster(cluster)
	}
	// 处理切片情况
	if clusters, ok := tx.Statement.Dest.(*[]models.KubernetesCluster); ok {
		for i := range *clusters {
			if err := h.encryptionService.DecryptKubernetesCluster(&(*clusters)[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
