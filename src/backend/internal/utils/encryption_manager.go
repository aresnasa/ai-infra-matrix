package utils

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// 全局加密服务实例
var (
	globalEncryptionService *EncryptionService
	encryptionOnce          sync.Once
	encryptionMutex         sync.RWMutex
)

// InitEncryptionService 初始化全局加密服务
func InitEncryptionService(encryptionKey string) error {
	var initErr error
	encryptionOnce.Do(func() {
		service, err := NewEncryptionService(encryptionKey)
		if err != nil {
			logrus.WithError(err).Error("Failed to initialize encryption service")
			initErr = err
			return
		}
		encryptionMutex.Lock()
		globalEncryptionService = service
		encryptionMutex.Unlock()
		logrus.Info("Encryption service initialized successfully")
	})
	return initErr
}

// GetEncryptionService 获取全局加密服务实例
func GetEncryptionService() *EncryptionService {
	encryptionMutex.RLock()
	defer encryptionMutex.RUnlock()
	return globalEncryptionService
}

// EncryptSensitiveField 加密敏感字段（全局便捷方法）
func EncryptSensitiveField(plaintext string) string {
	if plaintext == "" {
		return ""
	}

	service := GetEncryptionService()
	if service == nil {
		logrus.Warn("Encryption service not initialized, returning plaintext")
		return plaintext
	}

	encrypted, err := service.EncryptIfNeeded(plaintext)
	if err != nil {
		logrus.WithError(err).Error("Failed to encrypt sensitive field")
		return plaintext
	}
	return encrypted
}

// DecryptSensitiveField 解密敏感字段（全局便捷方法）
func DecryptSensitiveField(ciphertext string) string {
	if ciphertext == "" {
		return ""
	}

	service := GetEncryptionService()
	if service == nil {
		logrus.Warn("Encryption service not initialized, returning ciphertext")
		return ciphertext
	}

	return service.DecryptSafely(ciphertext)
}

// IsSensitiveFieldEncrypted 检查敏感字段是否已加密
func IsSensitiveFieldEncrypted(data string) bool {
	service := GetEncryptionService()
	if service == nil {
		return false
	}
	return service.IsEncrypted(data)
}
