package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
)

// EncryptionService 加密服务
type EncryptionService struct {
	key []byte
}

// NewEncryptionService 创建新的加密服务实例
func NewEncryptionService(encryptionKey string) (*EncryptionService, error) {
	if encryptionKey == "" {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}
	
	// 使用SHA256生成32字节的密钥
	hash := sha256.Sum256([]byte(encryptionKey))
	
	return &EncryptionService{
		key: hash[:],
	}, nil
}

// Encrypt 加密数据
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		logrus.WithError(err).Error("Failed to create AES cipher")
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		logrus.WithError(err).Error("Failed to create GCM")
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		logrus.WithError(err).Error("Failed to generate nonce")
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64编码
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	
	// 添加前缀标识这是加密数据
	return "encrypted:" + encoded, nil
}

// Decrypt 解密数据
func (e *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// 检查是否是加密数据
	if !strings.HasPrefix(ciphertext, "encrypted:") {
		// 如果没有加密前缀，直接返回原文（向后兼容）
		return ciphertext, nil
	}

	// 去除前缀
	encodedData := strings.TrimPrefix(ciphertext, "encrypted:")

	// Base64解码
	data, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		logrus.WithError(err).Error("Failed to decode base64 data")
		return "", fmt.Errorf("failed to decode data: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		logrus.WithError(err).Error("Failed to create AES cipher for decryption")
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		logrus.WithError(err).Error("Failed to create GCM for decryption")
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 提取nonce
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := aesGCM.Open(nil, nonce, cipherData, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to decrypt data")
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 检查数据是否已加密
func (e *EncryptionService) IsEncrypted(data string) bool {
	return strings.HasPrefix(data, "encrypted:")
}

// EncryptIfNeeded 如果数据未加密则进行加密
func (e *EncryptionService) EncryptIfNeeded(data string) (string, error) {
	if e.IsEncrypted(data) {
		return data, nil
	}
	return e.Encrypt(data)
}

// DecryptSafely 安全解密，如果失败则返回原文
func (e *EncryptionService) DecryptSafely(data string) string {
	decrypted, err := e.Decrypt(data)
	if err != nil {
		logrus.WithError(err).Warn("Failed to decrypt data, returning original")
		return data
	}
	return decrypted
}
