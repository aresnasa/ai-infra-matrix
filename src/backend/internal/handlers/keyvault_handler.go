package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// KeyVaultHandler 密钥保管库 API 处理器
type KeyVaultHandler struct {
	service *services.KeyVaultService
}

// NewKeyVaultHandler 创建密钥保管库处理器
func NewKeyVaultHandler() *KeyVaultHandler {
	return &KeyVaultHandler{
		service: services.NewKeyVaultService(),
	}
}

// ==================== 请求/响应结构体 ====================

// GenerateSyncTokenRequest 生成同步令牌请求
type GenerateSyncTokenRequest struct {
	KeyNames   []string           `json:"key_names"`             // 允许访问的密钥名称（可选，留空表示不限制）
	KeyTypes   []services.KeyType `json:"key_types"`             // 允许访问的密钥类型（可选）
	AllowWrite bool               `json:"allow_write"`           // 是否允许写入
	TTLSeconds int                `json:"ttl_seconds,omitempty"` // 令牌有效期（秒），默认300
}

// StoreKeyRequest 存储密钥请求
type StoreKeyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	KeyType     services.KeyType       `json:"key_type" binding:"required"`
	KeyData     string                 `json:"key_data" binding:"required"` // Base64 编码的密钥数据
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// SyncKeyRequest 同步密钥请求（使用一次性令牌）
type SyncKeyRequest struct {
	SyncToken string `json:"sync_token" binding:"required"` // 一次性同步令牌
	KeyName   string `json:"key_name" binding:"required"`   // 要同步的密钥名称
}

// SyncMultipleKeysRequest 批量同步密钥请求
type SyncMultipleKeysRequest struct {
	SyncToken string   `json:"sync_token" binding:"required"`
	KeyNames  []string `json:"key_names" binding:"required"`
}

// StoreKeyWithTokenRequest 使用令牌存储密钥请求
type StoreKeyWithTokenRequest struct {
	SyncToken   string                 `json:"sync_token" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	KeyType     services.KeyType       `json:"key_type" binding:"required"`
	KeyData     string                 `json:"key_data" binding:"required"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ==================== API 端点处理 ====================

// GenerateSyncToken 生成密钥同步一次性令牌
// @Summary 生成密钥同步令牌
// @Description 生成一个用于安全获取密钥的一次性令牌，有效期短且只能使用一次
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param request body GenerateSyncTokenRequest true "生成令牌请求"
// @Success 200 {object} map[string]interface{} "令牌信息"
// @Router /api/keyvault/sync-token [post]
func (h *KeyVaultHandler) GenerateSyncToken(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	var req GenerateSyncTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 设置默认 TTL
	ttl := 5 * time.Minute
	if req.TTLSeconds > 0 {
		// 最大允许 30 分钟
		if req.TTLSeconds > 1800 {
			req.TTLSeconds = 1800
		}
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}

	token, err := h.service.GenerateSyncToken(
		userID,
		usernameStr,
		req.KeyNames,
		req.KeyTypes,
		req.AllowWrite,
		c.ClientIP(),
		ttl,
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate sync token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate sync token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token_id":    token.TokenID,
		"expires_at":  token.ExpiresAt,
		"ttl_seconds": int(ttl.Seconds()),
		"key_names":   token.KeyNames,
		"key_types":   token.KeyTypes,
		"allow_write": token.AllowWrite,
	})
}

// SyncKey 使用一次性令牌同步获取密钥
// @Summary 同步获取密钥
// @Description 使用一次性令牌安全地获取密钥，令牌使用后即失效
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param request body SyncKeyRequest true "同步密钥请求"
// @Success 200 {object} map[string]interface{} "密钥数据"
// @Router /api/keyvault/sync [post]
func (h *KeyVaultHandler) SyncKey(c *gin.Context) {
	var req SyncKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	keyData, entry, err := h.service.SyncKeyWithToken(req.SyncToken, req.KeyName, c.ClientIP())
	if err != nil {
		logrus.WithError(err).WithField("key_name", req.KeyName).Warn("Failed to sync key")
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 返回 Base64 编码的密钥数据
	c.JSON(http.StatusOK, gin.H{
		"key_name":    entry.Name,
		"key_type":    entry.KeyType,
		"key_data":    h.service.EncodeKeyForTransport(keyData),
		"checksum":    entry.Checksum,
		"version":     entry.Version,
		"description": entry.Description,
	})
}

// SyncMultipleKeys 批量同步获取密钥
// @Summary 批量同步获取密钥
// @Description 使用一次性令牌批量获取多个密钥
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param request body SyncMultipleKeysRequest true "批量同步请求"
// @Success 200 {object} map[string]interface{} "密钥数据"
// @Router /api/keyvault/sync/batch [post]
func (h *KeyVaultHandler) SyncMultipleKeys(c *gin.Context) {
	var req SyncMultipleKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	keysData, err := h.service.SyncMultipleKeysWithToken(req.SyncToken, req.KeyNames, c.ClientIP())
	if err != nil {
		logrus.WithError(err).Warn("Failed to sync multiple keys")
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 编码所有密钥数据
	encodedKeys := make(map[string]string)
	for name, data := range keysData {
		encodedKeys[name] = h.service.EncodeKeyForTransport(data)
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":      encodedKeys,
		"total":     len(encodedKeys),
		"requested": len(req.KeyNames),
	})
}

// StoreKey 存储密钥（需要管理员权限）
// @Summary 存储密钥
// @Description 存储新密钥或更新现有密钥
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param request body StoreKeyRequest true "存储密钥请求"
// @Success 200 {object} map[string]interface{} "存储结果"
// @Router /api/keyvault/keys [post]
func (h *KeyVaultHandler) StoreKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req StoreKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 解码 Base64 密钥数据
	keyData, err := h.service.DecodeKeyFromTransport(req.KeyData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key data encoding"})
		return
	}

	entry, err := h.service.StoreKey(req.Name, req.KeyType, keyData, req.Description, req.Metadata, userID, req.ExpiresAt)
	if err != nil {
		logrus.WithError(err).Error("Failed to store key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       entry.ID,
		"name":     entry.Name,
		"key_type": entry.KeyType,
		"version":  entry.Version,
		"checksum": entry.Checksum,
		"message":  "key stored successfully",
	})
}

// StoreKeyWithToken 使用令牌存储密钥
// @Summary 使用令牌存储密钥
// @Description 使用一次性令牌存储密钥（需要令牌具有写入权限）
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param request body StoreKeyWithTokenRequest true "存储请求"
// @Success 200 {object} map[string]interface{} "存储结果"
// @Router /api/keyvault/sync/store [post]
func (h *KeyVaultHandler) StoreKeyWithToken(c *gin.Context) {
	var req StoreKeyWithTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 解码密钥数据
	keyData, err := h.service.DecodeKeyFromTransport(req.KeyData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key data encoding"})
		return
	}

	entry, err := h.service.StoreKeyWithToken(req.SyncToken, req.Name, req.KeyType, keyData, req.Description, req.Metadata, c.ClientIP())
	if err != nil {
		logrus.WithError(err).Warn("Failed to store key with token")
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       entry.ID,
		"name":     entry.Name,
		"key_type": entry.KeyType,
		"version":  entry.Version,
		"message":  "key stored successfully",
	})
}

// GetKey 获取密钥（需要认证）
// @Summary 获取密钥
// @Description 直接获取密钥（需要认证，适用于后端内部调用）
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param name path string true "密钥名称"
// @Success 200 {object} map[string]interface{} "密钥数据"
// @Router /api/keyvault/keys/{name} [get]
func (h *KeyVaultHandler) GetKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	keyName := c.Param("name")
	if keyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key name is required"})
		return
	}

	keyData, entry, err := h.service.GetKey(keyName, userID, usernameStr, c.ClientIP(), "")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key_name":    entry.Name,
		"key_type":    entry.KeyType,
		"key_data":    h.service.EncodeKeyForTransport(keyData),
		"checksum":    entry.Checksum,
		"version":     entry.Version,
		"description": entry.Description,
	})
}

// ListKeys 列出密钥（不包含密钥数据）
// @Summary 列出密钥
// @Description 获取密钥列表，按类型过滤
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param key_type query string false "密钥类型过滤"
// @Success 200 {object} map[string]interface{} "密钥列表"
// @Router /api/keyvault/keys [get]
func (h *KeyVaultHandler) ListKeys(c *gin.Context) {
	keyType := services.KeyType(c.Query("key_type"))

	entries, err := h.service.GetKeysByType(keyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":  entries,
		"total": len(entries),
	})
}

// DeleteKey 删除密钥（需要管理员权限）
// @Summary 删除密钥
// @Description 删除指定密钥
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param name path string true "密钥名称"
// @Success 200 {object} map[string]interface{} "删除结果"
// @Router /api/keyvault/keys/{name} [delete]
func (h *KeyVaultHandler) DeleteKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	keyName := c.Param("name")
	if keyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key name is required"})
		return
	}

	if err := h.service.DeleteKey(keyName, userID, usernameStr, c.ClientIP()); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "key deleted successfully"})
}

// GetAccessLogs 获取访问日志
// @Summary 获取访问日志
// @Description 获取密钥访问日志
// @Tags KeyVault
// @Accept json
// @Produce json
// @Param key_name query string false "密钥名称过滤"
// @Param limit query int false "每页数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{} "访问日志"
// @Router /api/keyvault/logs [get]
func (h *KeyVaultHandler) GetAccessLogs(c *gin.Context) {
	keyName := c.Query("key_name")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	logs, total, err := h.service.GetAccessLogs(keyName, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get access logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HealthCheck 健康检查
// @Summary KeyVault 健康检查
// @Description 检查 KeyVault 服务健康状态
// @Tags KeyVault
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "健康状态"
// @Router /api/keyvault/health [get]
func (h *KeyVaultHandler) HealthCheck(c *gin.Context) {
	health := h.service.HealthCheck()

	status := http.StatusOK
	if health["status"] == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}

// AutoMigrate 自动迁移数据库表
func (h *KeyVaultHandler) AutoMigrate() error {
	return h.service.AutoMigrate()
}
