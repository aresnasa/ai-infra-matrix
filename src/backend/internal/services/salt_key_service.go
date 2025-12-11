package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SaltKeyService Salt Master 公钥安全分发服务
// 该服务提供一次性令牌机制，用于安全地将 Salt Master 公钥分发到 Minion 节点
type SaltKeyService struct {
	secretKey   []byte
	usedNonces  sync.Map
	nonceExpiry time.Duration
}

// InstallToken 安装令牌信息
type InstallToken struct {
	Token     string     `json:"token"`
	MinionID  string     `json:"minion_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	Used      bool       `json:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	IPAddress string     `json:"ip_address,omitempty"`
}

// 全局安装令牌存储
var installTokenStore sync.Map

var (
	saltKeyServiceInstance *SaltKeyService
	saltKeyServiceOnce     sync.Once
)

// GetSaltKeyService 获取 SaltKeyService 单例
func GetSaltKeyService() *SaltKeyService {
	saltKeyServiceOnce.Do(func() {
		secretKey := os.Getenv("SALT_KEY_SECRET")
		if secretKey == "" {
			// 自动生成安全密钥
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				logrus.WithError(err).Fatal("Failed to generate SALT_KEY_SECRET")
			}
			secretKey = hex.EncodeToString(key)
			logrus.Warn("SALT_KEY_SECRET not set, generated random key (set env var for persistence)")
		}

		saltKeyServiceInstance = &SaltKeyService{
			secretKey:   []byte(secretKey),
			nonceExpiry: 5 * time.Minute,
		}

		// 启动清理任务
		go saltKeyServiceInstance.startCleanupTask()

		logrus.Info("[SaltKeyService] Initialized")
	})
	return saltKeyServiceInstance
}

// GetSaltKeyHandler 获取 SaltKeyService（别名，兼容 handler 层）
func GetSaltKeyHandler() *SaltKeyService {
	return GetSaltKeyService()
}

// GenerateInstallTokenForBatch 为批量安装生成一次性令牌
// 返回: token, masterPubURL, error
func (s *SaltKeyService) GenerateInstallTokenForBatch(minionID string, ttlSeconds int) (string, string, error) {
	if ttlSeconds < 60 {
		ttlSeconds = 300 // 最少 5 分钟
	}
	if ttlSeconds > 3600 {
		ttlSeconds = 3600 // 最长 1 小时
	}

	// 生成随机令牌
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	tokenInfo := &InstallToken{
		Token:     token,
		MinionID:  minionID,
		ExpiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
		Used:      false,
		CreatedAt: time.Now(),
	}
	installTokenStore.Store(token, tokenInfo)

	// 构建 URL
	apiURL := s.getAPIURL()
	masterPubURL := fmt.Sprintf("%s/api/salt-key/master-pub/simple?token=%s", apiURL, token)

	logrus.WithFields(logrus.Fields{
		"minion_id":   minionID,
		"ttl_seconds": ttlSeconds,
		"token":       token[:8] + "...",
	}).Info("Generated install token for batch install")

	return token, masterPubURL, nil
}

// ValidateToken 验证令牌并标记为已使用
func (s *SaltKeyService) ValidateToken(token, clientIP string) (*InstallToken, error) {
	value, ok := installTokenStore.Load(token)
	if !ok {
		return nil, fmt.Errorf("invalid or expired token")
	}

	tokenInfo := value.(*InstallToken)

	// 检查是否过期
	if time.Now().After(tokenInfo.ExpiresAt) {
		installTokenStore.Delete(token)
		return nil, fmt.Errorf("token expired")
	}

	// 检查是否已使用
	if tokenInfo.Used {
		return nil, fmt.Errorf("token already used")
	}

	// 标记为已使用
	now := time.Now()
	tokenInfo.Used = true
	tokenInfo.UsedAt = &now
	tokenInfo.IPAddress = clientIP
	installTokenStore.Store(token, tokenInfo)

	return tokenInfo, nil
}

// GetToken 获取令牌信息（不标记为已使用）
func (s *SaltKeyService) GetToken(token string) (*InstallToken, bool) {
	value, ok := installTokenStore.Load(token)
	if !ok {
		return nil, false
	}
	return value.(*InstallToken), true
}

// ListTokens 列出所有有效令牌
func (s *SaltKeyService) ListTokens() []*InstallToken {
	var tokens []*InstallToken
	now := time.Now()

	installTokenStore.Range(func(key, value interface{}) bool {
		info := value.(*InstallToken)
		if now.Before(info.ExpiresAt) {
			tokens = append(tokens, info)
		} else {
			// 清理过期令牌
			installTokenStore.Delete(key)
		}
		return true
	})

	return tokens
}

// RevokeToken 撤销令牌
func (s *SaltKeyService) RevokeToken(token string) bool {
	_, ok := installTokenStore.LoadAndDelete(token)
	return ok
}

// ComputeSignature 计算 HMAC 签名
func (s *SaltKeyService) ComputeSignature(minionID string, timestamp int64, nonce string) string {
	data := fmt.Sprintf("%s|%d|%s", minionID, timestamp, nonce)
	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateSignature 验证签名
func (s *SaltKeyService) ValidateSignature(minionID string, timestamp int64, nonce, signature string) bool {
	// 验证时间戳（允许 5 分钟误差）
	now := time.Now().Unix()
	if abs64(now-timestamp) > 300 {
		return false
	}

	// 验证 nonce（防重放攻击）
	if _, loaded := s.usedNonces.LoadOrStore(nonce, time.Now()); loaded {
		return false
	}

	// 验证签名
	expectedSig := s.ComputeSignature(minionID, timestamp, nonce)
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// GetSecretKey 获取密钥（供生成签名使用）
func (s *SaltKeyService) GetSecretKey() string {
	return string(s.secretKey)
}

// ReadMasterPubKey 读取 Master 公钥
// 优先从 KeyVault 读取，如果不存在则尝试从文件系统读取
func (s *SaltKeyService) ReadMasterPubKey() ([]byte, error) {
	// 1. 优先从 KeyVault 读取 (适用于容器化部署)
	kvService := NewKeyVaultService()
	if kvService != nil {
		keyData, _, err := kvService.GetKey("salt_master_public", 0, "system", "", "salt-key-service")
		if err == nil && keyData != "" {
			logrus.Debug("[SaltKeyService] Master public key loaded from KeyVault")
			return []byte(keyData), nil
		}
		// KeyVault 中没找到，继续尝试文件系统
		logrus.WithError(err).Debug("[SaltKeyService] KeyVault lookup failed, trying filesystem")
	}

	// 2. 从文件系统读取 (适用于非容器化部署或挂载了 pki 目录的场景)
	paths := []string{
		"/etc/salt/pki/master/master.pub",
		"/data/saltstack/pki/master/master.pub",
		os.Getenv("SALT_MASTER_PUB_KEY_PATH"),
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if data, err := os.ReadFile(path); err == nil {
			logrus.WithField("path", path).Debug("[SaltKeyService] Master public key loaded from filesystem")
			return data, nil
		}
	}

	return nil, fmt.Errorf("master public key not found in KeyVault or filesystem locations")
}

// getAPIURL 获取 API URL（用于外部访问）
func (s *SaltKeyService) getAPIURL() string {
	// 优先使用显式配置的 API_URL
	apiURL := os.Getenv("API_URL")
	if apiURL != "" {
		return strings.TrimSuffix(apiURL, "/")
	}

	// 否则使用 EXTERNAL_HOST + EXTERNAL_PORT（nginx 端口）构建 URL
	externalHost := os.Getenv("EXTERNAL_HOST")
	if externalHost == "" {
		externalHost = "localhost"
	}

	// 使用 EXTERNAL_PORT（nginx 端口），而不是内部 API_PORT
	// 因为外部机器需要通过 nginx 代理访问 /api 路径
	externalPort := os.Getenv("EXTERNAL_PORT")
	if externalPort == "" {
		externalPort = os.Getenv("NGINX_PORT")
	}
	if externalPort == "" {
		externalPort = "8080"
	}

	externalScheme := os.Getenv("EXTERNAL_SCHEME")
	if externalScheme == "" {
		externalScheme = "http"
	}

	return fmt.Sprintf("%s://%s:%s", externalScheme, externalHost, externalPort)
}

// startCleanupTask 启动清理任务
func (s *SaltKeyService) startCleanupTask() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupExpiredNonces()
		s.cleanupExpiredTokens()
	}
}

// cleanupExpiredNonces 清理过期的 nonce
func (s *SaltKeyService) cleanupExpiredNonces() {
	now := time.Now()
	s.usedNonces.Range(func(key, value interface{}) bool {
		if usedTime, ok := value.(time.Time); ok {
			if now.Sub(usedTime) > s.nonceExpiry {
				s.usedNonces.Delete(key)
			}
		}
		return true
	})
}

// cleanupExpiredTokens 清理过期的令牌
func (s *SaltKeyService) cleanupExpiredTokens() {
	now := time.Now()
	installTokenStore.Range(func(key, value interface{}) bool {
		info := value.(*InstallToken)
		if now.After(info.ExpiresAt) {
			installTokenStore.Delete(key)
		}
		return true
	})
}

// abs64 返回 int64 绝对值
func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// GenerateNonce 生成随机 nonce
func GenerateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
