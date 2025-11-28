package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SaltKeyHandler 处理 Salt Master 公钥安全分发
type SaltKeyHandler struct {
	service *services.SaltKeyService
}

// MasterKeyRequest 获取 Master 公钥请求
type MasterKeyRequest struct {
	MinionID  string `json:"minion_id" binding:"required"` // Minion ID
	Timestamp int64  `json:"timestamp" binding:"required"` // 请求时间戳
	Nonce     string `json:"nonce" binding:"required"`     // 随机数（防重放）
	Signature string `json:"signature" binding:"required"` // HMAC 签名
}

// MasterKeyResponse 获取 Master 公钥响应
type MasterKeyResponse struct {
	MasterPub string `json:"master_pub"` // Base64 编码的 Master 公钥
	Checksum  string `json:"checksum"`   // SHA256 校验和
	Timestamp int64  `json:"timestamp"`  // 响应时间戳
}

// NewSaltKeyHandler 创建新的 Salt Key 处理器
func NewSaltKeyHandler() *SaltKeyHandler {
	return &SaltKeyHandler{
		service: services.GetSaltKeyService(),
	}
}

// RegisterRoutes 注册路由
func (h *SaltKeyHandler) RegisterRoutes(r *gin.RouterGroup) {
	saltKey := r.Group("/salt-key")
	{
		// 无需认证的端点（使用 HMAC 签名验证或一次性令牌）
		saltKey.POST("/master-pub", h.GetMasterPubKey)
		saltKey.GET("/master-pub/simple", h.GetMasterPubKeySimple)

		// 需要认证的端点
		saltKey.POST("/install-token", h.GenerateInstallToken)
		saltKey.GET("/install-tokens", h.ListInstallTokens)
		saltKey.DELETE("/install-token/:token", h.RevokeInstallToken)
	}
}

// GetMasterPubKey 获取 Master 公钥（需要 HMAC 签名验证）
// @Summary 获取 Salt Master 公钥
// @Description 使用 HMAC 签名验证获取 Salt Master 公钥
// @Tags SaltKey
// @Accept json
// @Produce json
// @Param request body MasterKeyRequest true "获取公钥请求"
// @Success 200 {object} MasterKeyResponse "公钥数据"
// @Router /api/salt-key/master-pub [post]
func (h *SaltKeyHandler) GetMasterPubKey(c *gin.Context) {
	var req MasterKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 验证签名
	if !h.service.ValidateSignature(req.MinionID, req.Timestamp, req.Nonce, req.Signature) {
		logrus.WithFields(logrus.Fields{
			"minion_id": req.MinionID,
			"ip":        c.ClientIP(),
		}).Warn("Invalid signature for master pub key request")
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature or timestamp"})
		return
	}

	// 读取 Master 公钥
	masterPub, err := h.service.ReadMasterPubKey()
	if err != nil {
		logrus.WithError(err).Error("Failed to read master public key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read master public key"})
		return
	}

	// 计算校验和
	checksum := sha256.Sum256(masterPub)

	logrus.WithFields(logrus.Fields{
		"minion_id": req.MinionID,
		"ip":        c.ClientIP(),
	}).Info("Master public key distributed via signature")

	c.JSON(http.StatusOK, MasterKeyResponse{
		MasterPub: base64.StdEncoding.EncodeToString(masterPub),
		Checksum:  hex.EncodeToString(checksum[:]),
		Timestamp: time.Now().Unix(),
	})
}

// GetMasterPubKeySimple 使用一次性令牌获取 Master 公钥（简化版，供脚本使用）
// @Summary 使用令牌获取 Salt Master 公钥
// @Description 使用预先生成的一次性令牌获取 Salt Master 公钥
// @Tags SaltKey
// @Accept json
// @Produce text/plain
// @Param token query string true "安装令牌"
// @Success 200 {string} string "Master 公钥内容"
// @Router /api/salt-key/master-pub/simple [get]
func (h *SaltKeyHandler) GetMasterPubKeySimple(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// 验证令牌
	tokenInfo, err := h.service.ValidateToken(token, c.ClientIP())
	if err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}

	// 读取 Master 公钥
	masterPub, err := h.service.ReadMasterPubKey()
	if err != nil {
		logrus.WithError(err).Error("Failed to read master public key")
		c.String(http.StatusInternalServerError, "failed to read master public key")
		return
	}

	logrus.WithFields(logrus.Fields{
		"minion_id": tokenInfo.MinionID,
		"ip":        c.ClientIP(),
		"token":     token[:8] + "...",
	}).Info("Master public key distributed via token")

	// 返回纯文本格式的公钥
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, string(masterPub))
}

// GenerateInstallToken 生成安装令牌（需要认证）
// @Summary 生成安装令牌
// @Description 生成用于安装 Salt Minion 的一次性令牌
// @Tags SaltKey
// @Accept json
// @Produce json
// @Param minion_id query string true "Minion ID"
// @Param ttl_seconds query int false "令牌有效期（秒），默认300"
// @Success 200 {object} map[string]interface{} "令牌信息"
// @Router /api/salt-key/install-token [post]
func (h *SaltKeyHandler) GenerateInstallToken(c *gin.Context) {
	minionID := c.Query("minion_id")
	if minionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minion_id is required"})
		return
	}

	ttlSeconds, _ := strconv.Atoi(c.DefaultQuery("ttl_seconds", "300"))

	token, masterPubURL, err := h.service.GenerateInstallTokenForBatch(minionID, ttlSeconds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// 获取令牌信息
	tokenInfo, _ := h.service.GetToken(token)

	c.JSON(http.StatusOK, gin.H{
		"token":          token,
		"minion_id":      minionID,
		"expires_at":     tokenInfo.ExpiresAt,
		"ttl_seconds":    ttlSeconds,
		"master_pub_url": masterPubURL,
	})
}

// ListInstallTokens 列出所有安装令牌（需要认证）
// @Summary 列出安装令牌
// @Description 列出所有有效的安装令牌
// @Tags SaltKey
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "令牌列表"
// @Router /api/salt-key/install-tokens [get]
func (h *SaltKeyHandler) ListInstallTokens(c *gin.Context) {
	tokens := h.service.ListTokens()

	var tokenList []map[string]interface{}
	for _, info := range tokens {
		tokenList = append(tokenList, map[string]interface{}{
			"token":      info.Token[:8] + "...", // 只显示前 8 位
			"minion_id":  info.MinionID,
			"expires_at": info.ExpiresAt,
			"used":       info.Used,
			"created_at": info.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokenList,
		"total":  len(tokenList),
	})
}

// RevokeInstallToken 撤销安装令牌（需要认证）
// @Summary 撤销安装令牌
// @Description 撤销指定的安装令牌
// @Tags SaltKey
// @Accept json
// @Produce json
// @Param token path string true "令牌"
// @Success 200 {object} map[string]interface{} "撤销结果"
// @Router /api/salt-key/install-token/{token} [delete]
func (h *SaltKeyHandler) RevokeInstallToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if !h.service.RevokeToken(token) {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	logrus.WithField("token", token[:8]+"...").Info("Install token revoked")

	c.JSON(http.StatusOK, gin.H{"message": "token revoked"})
}
