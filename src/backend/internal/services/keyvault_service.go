package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// KeyType 密钥类型
type KeyType string

const (
	KeyTypeSaltMaster KeyType = "salt_master"  // Salt Master 密钥
	KeyTypeSaltMinion KeyType = "salt_minion"  // Salt Minion 密钥
	KeyTypeMunge      KeyType = "munge"        // Munge 密钥
	KeyTypeSSHPublic  KeyType = "ssh_public"   // SSH 公钥
	KeyTypeSSHPrivate KeyType = "ssh_private"  // SSH 私钥
	KeyTypeSSHHostKey KeyType = "ssh_host_key" // SSH Host Key
	KeyTypeTLSCert    KeyType = "tls_cert"     // TLS 证书
	KeyTypeTLSKey     KeyType = "tls_key"      // TLS 私钥
	KeyTypeAPIKey     KeyType = "api_key"      // API 密钥
	KeyTypeEncryption KeyType = "encryption"   // 加密密钥
	KeyTypeCustom     KeyType = "custom"       // 自定义密钥
)

// KeyVaultEntry 密钥保管库条目
type KeyVaultEntry struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex;size:255;not null"`
	KeyType     KeyType    `json:"key_type" gorm:"size:50;not null;index"`
	Description string     `json:"description" gorm:"size:500"`
	KeyData     string     `json:"-" gorm:"type:text;not null"`       // 加密存储的密钥数据
	Metadata    string     `json:"metadata" gorm:"type:text"`         // JSON 格式的元数据
	Checksum    string     `json:"checksum" gorm:"size:64"`           // SHA256 校验和
	Version     int        `json:"version" gorm:"default:1"`          // 版本号
	ExpiresAt   *time.Time `json:"expires_at,omitempty" gorm:"index"` // 过期时间
	CreatedBy   uint       `json:"created_by" gorm:"not null"`        // 创建者用户ID
	UpdatedBy   uint       `json:"updated_by"`                        // 最后更新者用户ID
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (KeyVaultEntry) TableName() string {
	return "key_vault_entries"
}

// KeyVaultAccessLog 密钥访问日志
type KeyVaultAccessLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	KeyID       uint      `json:"key_id" gorm:"index;not null"`
	KeyName     string    `json:"key_name" gorm:"size:255;not null"`
	UserID      uint      `json:"user_id" gorm:"index;not null"`
	Username    string    `json:"username" gorm:"size:100"`
	Action      string    `json:"action" gorm:"size:50;not null"` // read, write, delete, sync
	IPAddress   string    `json:"ip_address" gorm:"size:45"`
	UserAgent   string    `json:"user_agent" gorm:"size:500"`
	Success     bool      `json:"success" gorm:"not null"`
	ErrorMsg    string    `json:"error_msg,omitempty" gorm:"size:500"`
	AccessToken string    `json:"access_token,omitempty" gorm:"size:64"` // 一次性 Token ID (截断)
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName 表名
func (KeyVaultAccessLog) TableName() string {
	return "key_vault_access_logs"
}

// KeySyncToken 密钥同步一次性令牌
type KeySyncToken struct {
	TokenID    string     `json:"token_id"`
	UserID     uint       `json:"user_id"`
	Username   string     `json:"username"`
	KeyNames   []string   `json:"key_names"`   // 允许访问的密钥名称
	KeyTypes   []KeyType  `json:"key_types"`   // 允许访问的密钥类型
	AllowWrite bool       `json:"allow_write"` // 是否允许写入
	IPAddress  string     `json:"ip_address"`  // 请求来源 IP
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	Used       bool       `json:"used"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
}

// KeyVaultService 密钥保管库服务
type KeyVaultService struct {
	db                *gorm.DB
	encryptionService *utils.EncryptionService
}

// NewKeyVaultService 创建密钥保管库服务
func NewKeyVaultService() *KeyVaultService {
	return &KeyVaultService{
		db:                database.DB,
		encryptionService: utils.GetEncryptionService(),
	}
}

// AutoMigrate 自动迁移数据库表
func (s *KeyVaultService) AutoMigrate() error {
	return s.db.AutoMigrate(&KeyVaultEntry{}, &KeyVaultAccessLog{})
}

// ==================== 一次性令牌管理 ====================

// GenerateSyncToken 生成密钥同步一次性令牌
// 该令牌用于安全地获取或同步密钥，有效期短且只能使用一次
func (s *KeyVaultService) GenerateSyncToken(userID uint, username string, keyNames []string, keyTypes []KeyType, allowWrite bool, ipAddress string, ttl time.Duration) (*KeySyncToken, error) {
	// 生成随机 Token ID
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	tokenID := hex.EncodeToString(tokenBytes)

	// 默认 TTL 为 5 分钟
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	token := &KeySyncToken{
		TokenID:    tokenID,
		UserID:     userID,
		Username:   username,
		KeyNames:   keyNames,
		KeyTypes:   keyTypes,
		AllowWrite: allowWrite,
		IPAddress:  ipAddress,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
		Used:       false,
	}

	// 存储到 Redis
	tokenKey := fmt.Sprintf("keyvault:sync_token:%s", tokenID)
	tokenData, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token: %w", err)
	}

	ctx := context.Background()
	if err := cache.RDB.Set(ctx, tokenKey, tokenData, ttl).Err(); err != nil {
		return nil, fmt.Errorf("failed to store sync token: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":     userID,
		"username":    username,
		"key_names":   keyNames,
		"key_types":   keyTypes,
		"allow_write": allowWrite,
		"ttl":         ttl,
	}).Info("KeyVault sync token generated")

	return token, nil
}

// ValidateSyncToken 验证并消费一次性令牌
func (s *KeyVaultService) ValidateSyncToken(tokenID string, ipAddress string) (*KeySyncToken, error) {
	ctx := context.Background()
	tokenKey := fmt.Sprintf("keyvault:sync_token:%s", tokenID)

	// 获取令牌
	tokenData, err := cache.RDB.Get(ctx, tokenKey).Result()
	if err != nil {
		return nil, fmt.Errorf("invalid or expired sync token")
	}

	var token KeySyncToken
	if err := json.Unmarshal([]byte(tokenData), &token); err != nil {
		return nil, fmt.Errorf("failed to parse sync token: %w", err)
	}

	// 检查是否已使用
	if token.Used {
		return nil, fmt.Errorf("sync token has already been used")
	}

	// 检查是否过期
	if time.Now().After(token.ExpiresAt) {
		cache.RDB.Del(ctx, tokenKey)
		return nil, fmt.Errorf("sync token has expired")
	}

	// 可选：验证 IP 地址一致性（增强安全性）
	// 如果创建时指定了 IP，则验证请求 IP 是否匹配
	if token.IPAddress != "" && token.IPAddress != ipAddress {
		logrus.WithFields(logrus.Fields{
			"expected_ip": token.IPAddress,
			"actual_ip":   ipAddress,
			"token_id":    tokenID[:16] + "...",
		}).Warn("KeyVault sync token IP mismatch")
		// 注意：这里可以选择拒绝或仅记录警告
		// return nil, fmt.Errorf("IP address mismatch")
	}

	// 标记令牌为已使用
	now := time.Now()
	token.Used = true
	token.UsedAt = &now

	updatedData, _ := json.Marshal(token)
	cache.RDB.Set(ctx, tokenKey, updatedData, time.Minute) // 保留1分钟用于审计

	logrus.WithFields(logrus.Fields{
		"user_id":  token.UserID,
		"username": token.Username,
		"ip":       ipAddress,
	}).Info("KeyVault sync token validated and consumed")

	return &token, nil
}

// ==================== 密钥管理 ====================

// StoreKey 存储密钥
func (s *KeyVaultService) StoreKey(name string, keyType KeyType, keyData string, description string, metadata map[string]interface{}, userID uint, expiresAt *time.Time) (*KeyVaultEntry, error) {
	if s.encryptionService == nil {
		return nil, fmt.Errorf("encryption service not initialized")
	}

	// 加密密钥数据
	encryptedData, err := s.encryptionService.Encrypt(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key data: %w", err)
	}

	// 计算原始数据的校验和
	checksum := sha256.Sum256([]byte(keyData))
	checksumHex := hex.EncodeToString(checksum[:])

	// 序列化元数据
	var metadataJSON string
	if metadata != nil {
		metadataBytes, _ := json.Marshal(metadata)
		metadataJSON = string(metadataBytes)
	}

	// 查找是否已存在
	var existing KeyVaultEntry
	err = s.db.Where("name = ?", name).First(&existing).Error

	if err == nil {
		// 更新现有记录
		existing.KeyType = keyType
		existing.Description = description
		existing.KeyData = encryptedData
		existing.Metadata = metadataJSON
		existing.Checksum = checksumHex
		existing.Version++
		existing.ExpiresAt = expiresAt
		existing.UpdatedBy = userID

		if err := s.db.Save(&existing).Error; err != nil {
			return nil, fmt.Errorf("failed to update key: %w", err)
		}

		s.logAccess(existing.ID, name, userID, "", "write", "", "", true, "")
		return &existing, nil
	}

	// 创建新记录
	entry := &KeyVaultEntry{
		Name:        name,
		KeyType:     keyType,
		Description: description,
		KeyData:     encryptedData,
		Metadata:    metadataJSON,
		Checksum:    checksumHex,
		Version:     1,
		ExpiresAt:   expiresAt,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := s.db.Create(entry).Error; err != nil {
		return nil, fmt.Errorf("failed to store key: %w", err)
	}

	s.logAccess(entry.ID, name, userID, "", "write", "", "", true, "")
	return entry, nil
}

// GetKey 获取密钥
func (s *KeyVaultService) GetKey(name string, userID uint, username, ipAddress, tokenID string) (string, *KeyVaultEntry, error) {
	if s.encryptionService == nil {
		return "", nil, fmt.Errorf("encryption service not initialized")
	}

	var entry KeyVaultEntry
	if err := s.db.Where("name = ?", name).First(&entry).Error; err != nil {
		s.logAccess(0, name, userID, username, "read", ipAddress, "", false, "key not found")
		return "", nil, fmt.Errorf("key not found: %s", name)
	}

	// 检查是否过期
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		s.logAccess(entry.ID, name, userID, username, "read", ipAddress, tokenID, false, "key expired")
		return "", nil, fmt.Errorf("key has expired: %s", name)
	}

	// 解密密钥数据
	decryptedData, err := s.encryptionService.Decrypt(entry.KeyData)
	if err != nil {
		s.logAccess(entry.ID, name, userID, username, "read", ipAddress, tokenID, false, "decryption failed")
		return "", nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	// 验证校验和
	checksum := sha256.Sum256([]byte(decryptedData))
	if hex.EncodeToString(checksum[:]) != entry.Checksum {
		s.logAccess(entry.ID, name, userID, username, "read", ipAddress, tokenID, false, "checksum mismatch")
		return "", nil, fmt.Errorf("key data integrity check failed")
	}

	s.logAccess(entry.ID, name, userID, username, "read", ipAddress, tokenID, true, "")
	return decryptedData, &entry, nil
}

// GetKeysByType 按类型获取密钥列表（不包含密钥数据）
func (s *KeyVaultService) GetKeysByType(keyType KeyType) ([]KeyVaultEntry, error) {
	var entries []KeyVaultEntry
	query := s.db.Select("id, name, key_type, description, metadata, checksum, version, expires_at, created_by, updated_by, created_at, updated_at")

	if keyType != "" {
		query = query.Where("key_type = ?", keyType)
	}

	if err := query.Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	return entries, nil
}

// DeleteKey 删除密钥
func (s *KeyVaultService) DeleteKey(name string, userID uint, username, ipAddress string) error {
	var entry KeyVaultEntry
	if err := s.db.Where("name = ?", name).First(&entry).Error; err != nil {
		s.logAccess(0, name, userID, username, "delete", ipAddress, "", false, "key not found")
		return fmt.Errorf("key not found: %s", name)
	}

	if err := s.db.Delete(&entry).Error; err != nil {
		s.logAccess(entry.ID, name, userID, username, "delete", ipAddress, "", false, err.Error())
		return fmt.Errorf("failed to delete key: %w", err)
	}

	s.logAccess(entry.ID, name, userID, username, "delete", ipAddress, "", true, "")
	return nil
}

// ==================== 密钥同步（使用一次性令牌）====================

// SyncKeyWithToken 使用一次性令牌同步获取密钥
func (s *KeyVaultService) SyncKeyWithToken(tokenID, keyName, ipAddress string) (string, *KeyVaultEntry, error) {
	// 验证令牌
	token, err := s.ValidateSyncToken(tokenID, ipAddress)
	if err != nil {
		return "", nil, err
	}

	// 检查是否有权限访问该密钥
	if !s.isKeyAccessAllowed(token, keyName) {
		s.logAccess(0, keyName, token.UserID, token.Username, "sync", ipAddress, tokenID[:16], false, "access denied by token")
		return "", nil, fmt.Errorf("access to key '%s' not allowed by this token", keyName)
	}

	// 获取密钥
	return s.GetKey(keyName, token.UserID, token.Username, ipAddress, tokenID[:16])
}

// SyncMultipleKeysWithToken 使用一次性令牌同步获取多个密钥
func (s *KeyVaultService) SyncMultipleKeysWithToken(tokenID string, keyNames []string, ipAddress string) (map[string]string, error) {
	// 验证令牌
	token, err := s.ValidateSyncToken(tokenID, ipAddress)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, keyName := range keyNames {
		if !s.isKeyAccessAllowed(token, keyName) {
			logrus.WithFields(logrus.Fields{
				"key_name": keyName,
				"user_id":  token.UserID,
			}).Warn("KeyVault: access denied for key in batch sync")
			continue
		}

		keyData, _, err := s.GetKey(keyName, token.UserID, token.Username, ipAddress, tokenID[:16])
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"key_name": keyName,
				"error":    err.Error(),
			}).Warn("KeyVault: failed to get key in batch sync")
			continue
		}
		result[keyName] = keyData
	}

	return result, nil
}

// StoreKeyWithToken 使用一次性令牌存储密钥
func (s *KeyVaultService) StoreKeyWithToken(tokenID, keyName string, keyType KeyType, keyData, description string, metadata map[string]interface{}, ipAddress string) (*KeyVaultEntry, error) {
	// 验证令牌
	token, err := s.ValidateSyncToken(tokenID, ipAddress)
	if err != nil {
		return nil, err
	}

	// 检查是否允许写入
	if !token.AllowWrite {
		s.logAccess(0, keyName, token.UserID, token.Username, "sync_write", ipAddress, tokenID[:16], false, "write not allowed")
		return nil, fmt.Errorf("write operation not allowed by this token")
	}

	// 检查是否有权限操作该密钥类型
	if !s.isKeyTypeAllowed(token, keyType) {
		s.logAccess(0, keyName, token.UserID, token.Username, "sync_write", ipAddress, tokenID[:16], false, "key type not allowed")
		return nil, fmt.Errorf("key type '%s' not allowed by this token", keyType)
	}

	return s.StoreKey(keyName, keyType, keyData, description, metadata, token.UserID, nil)
}

// isKeyAccessAllowed 检查令牌是否允许访问指定密钥
func (s *KeyVaultService) isKeyAccessAllowed(token *KeySyncToken, keyName string) bool {
	// 如果指定了特定密钥名称，检查是否在允许列表中
	if len(token.KeyNames) > 0 {
		for _, allowedName := range token.KeyNames {
			if allowedName == keyName || allowedName == "*" {
				return true
			}
			// 支持前缀匹配，如 "salt_*" 匹配 "salt_master_key"
			if strings.HasSuffix(allowedName, "*") {
				prefix := strings.TrimSuffix(allowedName, "*")
				if strings.HasPrefix(keyName, prefix) {
					return true
				}
			}
		}
		return false
	}

	// 如果指定了密钥类型，需要查询密钥确认类型匹配
	if len(token.KeyTypes) > 0 {
		var entry KeyVaultEntry
		if err := s.db.Select("key_type").Where("name = ?", keyName).First(&entry).Error; err != nil {
			return false
		}
		return s.isKeyTypeAllowed(token, entry.KeyType)
	}

	// 没有限制，允许访问
	return true
}

// isKeyTypeAllowed 检查令牌是否允许访问指定类型的密钥
func (s *KeyVaultService) isKeyTypeAllowed(token *KeySyncToken, keyType KeyType) bool {
	if len(token.KeyTypes) == 0 {
		return true
	}
	for _, allowedType := range token.KeyTypes {
		if allowedType == keyType {
			return true
		}
	}
	return false
}

// ==================== 审计日志 ====================

// logAccess 记录密钥访问日志
func (s *KeyVaultService) logAccess(keyID uint, keyName string, userID uint, username, action, ipAddress, tokenID string, success bool, errorMsg string) {
	log := &KeyVaultAccessLog{
		KeyID:       keyID,
		KeyName:     keyName,
		UserID:      userID,
		Username:    username,
		Action:      action,
		IPAddress:   ipAddress,
		AccessToken: tokenID,
		Success:     success,
		ErrorMsg:    errorMsg,
	}

	if err := s.db.Create(log).Error; err != nil {
		logrus.WithError(err).Error("Failed to create key vault access log")
	}
}

// GetAccessLogs 获取访问日志
func (s *KeyVaultService) GetAccessLogs(keyName string, limit int, offset int) ([]KeyVaultAccessLog, int64, error) {
	var logs []KeyVaultAccessLog
	var total int64

	query := s.db.Model(&KeyVaultAccessLog{})
	if keyName != "" {
		query = query.Where("key_name = ?", keyName)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ==================== 便捷方法 ====================

// StoreSaltMasterKey 存储 Salt Master 密钥
func (s *KeyVaultService) StoreSaltMasterKey(publicKey, privateKey string, userID uint) error {
	if publicKey != "" {
		_, err := s.StoreKey("salt_master_public", KeyTypeSaltMaster, publicKey, "Salt Master Public Key", nil, userID, nil)
		if err != nil {
			return err
		}
	}
	if privateKey != "" {
		_, err := s.StoreKey("salt_master_private", KeyTypeSaltMaster, privateKey, "Salt Master Private Key", nil, userID, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// StoreMungeKey 存储 Munge 密钥
func (s *KeyVaultService) StoreMungeKey(mungeKey string, clusterName string, userID uint) error {
	keyName := fmt.Sprintf("munge_key_%s", clusterName)
	_, err := s.StoreKey(keyName, KeyTypeMunge, mungeKey, fmt.Sprintf("Munge Key for cluster: %s", clusterName),
		map[string]interface{}{"cluster": clusterName}, userID, nil)
	return err
}

// StoreSSHKeyPair 存储 SSH 密钥对
func (s *KeyVaultService) StoreSSHKeyPair(name, publicKey, privateKey string, userID uint) error {
	if publicKey != "" {
		_, err := s.StoreKey(name+"_public", KeyTypeSSHPublic, publicKey, fmt.Sprintf("SSH Public Key: %s", name), nil, userID, nil)
		if err != nil {
			return err
		}
	}
	if privateKey != "" {
		_, err := s.StoreKey(name+"_private", KeyTypeSSHPrivate, privateKey, fmt.Sprintf("SSH Private Key: %s", name), nil, userID, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// EncodeKeyForTransport 将密钥编码为安全的传输格式
func (s *KeyVaultService) EncodeKeyForTransport(keyData string) string {
	return base64.StdEncoding.EncodeToString([]byte(keyData))
}

// DecodeKeyFromTransport 从传输格式解码密钥
func (s *KeyVaultService) DecodeKeyFromTransport(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %w", err)
	}
	return string(decoded), nil
}

// ==================== 密钥轮换 ====================

// RotateKey 轮换密钥（创建新版本）
func (s *KeyVaultService) RotateKey(name string, newKeyData string, userID uint) (*KeyVaultEntry, error) {
	var entry KeyVaultEntry
	if err := s.db.Where("name = ?", name).First(&entry).Error; err != nil {
		return nil, fmt.Errorf("key not found: %s", name)
	}

	// 备份旧版本
	backupName := fmt.Sprintf("%s_v%d_%s", name, entry.Version, time.Now().Format("20060102150405"))
	oldData, _, err := s.GetKey(name, userID, "", "", "")
	if err == nil {
		s.StoreKey(backupName, entry.KeyType, oldData, fmt.Sprintf("Backup of %s version %d", name, entry.Version),
			map[string]interface{}{"original_name": name, "original_version": entry.Version}, userID, nil)
	}

	// 存储新版本
	return s.StoreKey(name, entry.KeyType, newKeyData, entry.Description, nil, userID, entry.ExpiresAt)
}

// ==================== 健康检查 ====================

// HealthCheck 服务健康检查
func (s *KeyVaultService) HealthCheck() map[string]interface{} {
	result := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 检查数据库连接
	var count int64
	if err := s.db.Model(&KeyVaultEntry{}).Count(&count).Error; err != nil {
		result["status"] = "unhealthy"
		result["database_error"] = err.Error()
	} else {
		result["total_keys"] = count
	}

	// 检查加密服务
	if s.encryptionService == nil {
		result["status"] = "unhealthy"
		result["encryption_service"] = "not initialized"
	} else {
		result["encryption_service"] = "ready"
	}

	// 检查 Redis 连接
	ctx := context.Background()
	if err := cache.RDB.Ping(ctx).Err(); err != nil {
		result["status"] = "degraded"
		result["redis_error"] = err.Error()
	} else {
		result["redis"] = "connected"
	}

	return result
}
