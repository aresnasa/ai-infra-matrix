package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// CryptoService 提供数据库敏感数据的加密解密服务
type CryptoService struct {
	key []byte
}

// NewCryptoService 创建加密服务实例
func NewCryptoService(secretKey string) *CryptoService {
	// 使用SHA256生成32字节的密钥
	hash := sha256.Sum256([]byte(secretKey))
	return &CryptoService{
		key: hash[:],
	}
}

// Encrypt 加密数据
func (c *CryptoService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建AES cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func (c *CryptoService) Decrypt(encryptedData string) (string, error) {
	if encryptedData == "" {
		return "", nil
	}

	// base64解码
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("加密数据长度不足")
	}

	// 分离nonce和密文
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 检查数据是否已加密（通过尝试base64解码来判断）
func (c *CryptoService) IsEncrypted(data string) bool {
	if data == "" {
		return false
	}
	
	// 尝试base64解码
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false
	}
	
	// 检查长度是否符合加密数据的特征
	if len(decoded) < 12 { // GCM nonce是12字节
		return false
	}
	
	return true
}

// EncryptIfNeeded 如果数据未加密则加密
func (c *CryptoService) EncryptIfNeeded(data string) (string, error) {
	if c.IsEncrypted(data) {
		return data, nil
	}
	return c.Encrypt(data)
}

// DecryptSafely 安全解密，如果解密失败返回原数据
func (c *CryptoService) DecryptSafely(data string) string {
	if !c.IsEncrypted(data) {
		return data
	}
	
	decrypted, err := c.Decrypt(data)
	if err != nil {
		// 如果解密失败，返回原数据
		return data
	}
	
	return decrypted
}
