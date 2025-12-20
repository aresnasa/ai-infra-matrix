package handlers

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

// SecurityHandler 安全管理处理器
type SecurityHandler struct {
	db *gorm.DB
}

// NewSecurityHandler 创建安全管理处理器
func NewSecurityHandler() *SecurityHandler {
	return &SecurityHandler{
		db: database.DB,
	}
}

// AutoMigrate 自动迁移安全相关数据库表
func (h *SecurityHandler) AutoMigrate() error {
	return h.db.AutoMigrate(
		&models.IPBlacklist{},
		&models.IPWhitelist{},
		&models.LoginAttempt{},
		&models.TwoFactorConfig{},
		&models.TwoFactorGlobalConfig{},
		&models.OAuthProvider{},
		&models.UserOAuthBinding{},
		&models.SecurityAuditLog{},
		&models.SecurityConfig{},
	)
}

// === IP 黑名单管理 ===

// ListIPBlacklist 获取IP黑名单列表
func (h *SecurityHandler) ListIPBlacklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")

	query := h.db.Model(&models.IPBlacklist{})

	if search != "" {
		query = query.Where("ip LIKE ? OR reason LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var blacklist []models.IPBlacklist
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&blacklist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    blacklist,
		"total":   total,
		"page":    page,
		"size":    pageSize,
	})
}

// AddIPBlacklist 添加IP到黑名单
func (h *SecurityHandler) AddIPBlacklist(c *gin.Context) {
	var req struct {
		IP          string     `json:"ip" binding:"required"`
		Reason      string     `json:"reason"`
		BlockType   string     `json:"block_type"`
		ExpireAt    *time.Time `json:"expire_at,omitempty"`
		Description string     `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	// 验证IP格式
	if !isValidIP(req.IP) && !isValidCIDR(req.IP) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的IP地址或CIDR格式"})
		return
	}

	// 获取当前用户
	username, _ := c.Get("username")

	blockType := req.BlockType
	if blockType == "" {
		blockType = "permanent"
	}

	blacklist := models.IPBlacklist{
		IP:          req.IP,
		Reason:      req.Reason,
		BlockType:   blockType,
		ExpireAt:    req.ExpireAt,
		CreatedBy:   fmt.Sprintf("%v", username),
		Enabled:     true,
		Description: req.Description,
	}

	if err := h.db.Create(&blacklist).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "该IP已在黑名单中"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 记录审计日志
	h.logSecurityAudit(c, "ip_blacklist_add", "ip_blacklist", fmt.Sprint(blacklist.ID), "success", map[string]interface{}{
		"ip":     req.IP,
		"reason": req.Reason,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "data": blacklist})
}

// UpdateIPBlacklist 更新IP黑名单
func (h *SecurityHandler) UpdateIPBlacklist(c *gin.Context) {
	id := c.Param("id")

	var blacklist models.IPBlacklist
	if err := h.db.First(&blacklist, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "记录不存在"})
		return
	}

	var req struct {
		Reason      string     `json:"reason"`
		BlockType   string     `json:"block_type"`
		ExpireAt    *time.Time `json:"expire_at,omitempty"`
		Enabled     *bool      `json:"enabled"`
		Description string     `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	updates := map[string]interface{}{}
	if req.Reason != "" {
		updates["reason"] = req.Reason
	}
	if req.BlockType != "" {
		updates["block_type"] = req.BlockType
	}
	if req.ExpireAt != nil {
		updates["expire_at"] = req.ExpireAt
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := h.db.Model(&blacklist).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "ip_blacklist_update", "ip_blacklist", id, "success", updates)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": blacklist})
}

// DeleteIPBlacklist 删除IP黑名单
func (h *SecurityHandler) DeleteIPBlacklist(c *gin.Context) {
	id := c.Param("id")

	var blacklist models.IPBlacklist
	if err := h.db.First(&blacklist, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "记录不存在"})
		return
	}

	if err := h.db.Delete(&blacklist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "ip_blacklist_delete", "ip_blacklist", id, "success", map[string]interface{}{
		"ip": blacklist.IP,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

// BatchDeleteIPBlacklist 批量删除IP黑名单
func (h *SecurityHandler) BatchDeleteIPBlacklist(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	if err := h.db.Delete(&models.IPBlacklist{}, req.IDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "ip_blacklist_batch_delete", "ip_blacklist", "", "success", map[string]interface{}{
		"ids": req.IDs,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功删除 %d 条记录", len(req.IDs))})
}

// CheckIP 检查IP是否在黑名单中
func (h *SecurityHandler) CheckIP(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		ip = c.ClientIP()
	}

	blocked, reason := h.isIPBlocked(ip)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"ip":      ip,
		"blocked": blocked,
		"reason":  reason,
	})
}

// === IP 白名单管理 ===

// ListIPWhitelist 获取IP白名单列表
func (h *SecurityHandler) ListIPWhitelist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var total int64
	h.db.Model(&models.IPWhitelist{}).Count(&total)

	var whitelist []models.IPWhitelist
	offset := (page - 1) * pageSize
	if err := h.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&whitelist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    whitelist,
		"total":   total,
		"page":    page,
		"size":    pageSize,
	})
}

// AddIPWhitelist 添加IP到白名单
func (h *SecurityHandler) AddIPWhitelist(c *gin.Context) {
	var req struct {
		IP          string `json:"ip" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	if !isValidIP(req.IP) && !isValidCIDR(req.IP) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的IP地址或CIDR格式"})
		return
	}

	username, _ := c.Get("username")

	whitelist := models.IPWhitelist{
		IP:          req.IP,
		Description: req.Description,
		CreatedBy:   fmt.Sprintf("%v", username),
		Enabled:     true,
	}

	if err := h.db.Create(&whitelist).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "该IP已在白名单中"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "ip_whitelist_add", "ip_whitelist", fmt.Sprint(whitelist.ID), "success", map[string]interface{}{
		"ip": req.IP,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "data": whitelist})
}

// DeleteIPWhitelist 删除IP白名单
func (h *SecurityHandler) DeleteIPWhitelist(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.IPWhitelist{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "ip_whitelist_delete", "ip_whitelist", id, "success", nil)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

// === 二次认证（2FA）管理 ===

// Get2FAStatus 获取用户2FA状态
func (h *SecurityHandler) Get2FAStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var config models.TwoFactorConfig
	err := h.db.Where("user_id = ?", userID).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": map[string]interface{}{
				"enabled":         false,
				"type":            "",
				"has_recovery":    false,
				"recovery_used":   0,
				"last_verified":   nil,
				"can_setup_totp":  true,
				"can_setup_sms":   false,
				"can_setup_email": false,
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 获取全局配置
	globalConfig := h.get2FAGlobalConfig()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"enabled":         config.Enabled,
			"type":            config.Type,
			"has_recovery":    config.RecoveryCodes != "",
			"recovery_used":   config.RecoveryUsedCount,
			"last_verified":   config.LastVerifiedAt,
			"can_setup_totp":  true,
			"can_setup_sms":   globalConfig.SMSEnabled,
			"can_setup_email": globalConfig.EmailEnabled,
		},
	})
}

// Setup2FA 设置2FA
func (h *SecurityHandler) Setup2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	username, _ := c.Get("username")

	var req struct {
		Type string `json:"type"` // totp, sms, email
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Type = "totp"
	}

	// 目前只支持 TOTP
	if req.Type != "totp" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "目前只支持 TOTP（Google Authenticator）"})
		return
	}

	// 生成TOTP密钥
	globalConfig := h.get2FAGlobalConfig()
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      globalConfig.TOTPIssuer,
		AccountName: fmt.Sprintf("%v", username),
		Period:      uint(globalConfig.TOTPPeriod),
		Digits:      6,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "生成密钥失败"})
		return
	}

	// 生成恢复码
	recoveryCodes := generateRecoveryCodes(globalConfig.RecoveryCodeCount)
	recoveryCodesJSON, _ := json.Marshal(recoveryCodes)

	// 创建或更新配置
	config := models.TwoFactorConfig{
		UserID:        userID.(uint),
		Enabled:       false, // 设置时还未启用，需要验证后才启用
		Type:          "totp",
		Secret:        key.Secret(),
		RecoveryCodes: string(recoveryCodesJSON),
	}

	if err := h.db.Where("user_id = ?", userID).Assign(config).FirstOrCreate(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"secret":          key.Secret(),
			"qr_code":         key.URL(),
			"issuer":          globalConfig.TOTPIssuer,
			"account":         username,
			"recovery_codes":  recoveryCodes,
			"period":          globalConfig.TOTPPeriod,
			"digits":          globalConfig.TOTPDigits,
			"setup_completed": false,
		},
	})
}

// Enable2FA 启用2FA（验证后启用）
func (h *SecurityHandler) Enable2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请提供验证码"})
		return
	}

	var config models.TwoFactorConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请先设置2FA"})
		return
	}

	// 验证TOTP码
	globalConfig := h.get2FAGlobalConfig()
	valid := totp.Validate(req.Code, config.Secret)

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "验证码无效"})
		return
	}

	// 启用2FA
	now := time.Now()
	if err := h.db.Model(&config).Updates(map[string]interface{}{
		"enabled":          true,
		"last_verified_at": now,
		"verify_count":     gorm.Expr("verify_count + 1"),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "2fa_enable", "user", fmt.Sprint(userID), "success", map[string]interface{}{
		"type":   "totp",
		"issuer": globalConfig.TOTPIssuer,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "二次认证已启用"})
}

// Disable2FA 禁用2FA
func (h *SecurityHandler) Disable2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req struct {
		Code     string `json:"code"`     // TOTP码或恢复码
		Password string `json:"password"` // 当前密码（额外验证）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	var config models.TwoFactorConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "2FA未启用"})
		return
	}

	// 验证码验证
	valid := false
	if req.Code != "" {
		// 首先尝试TOTP验证
		valid = totp.Validate(req.Code, config.Secret)
		// 如果TOTP验证失败，尝试恢复码
		if !valid {
			valid = h.validateRecoveryCode(&config, req.Code)
		}
	}

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "验证码或恢复码无效"})
		return
	}

	// 禁用2FA
	if err := h.db.Model(&config).Updates(map[string]interface{}{
		"enabled":        false,
		"secret":         "",
		"recovery_codes": "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "2fa_disable", "user", fmt.Sprint(userID), "success", nil)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "二次认证已禁用"})
}

// Verify2FA 验证2FA码
func (h *SecurityHandler) Verify2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请提供验证码"})
		return
	}

	var config models.TwoFactorConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "2FA未启用"})
		return
	}

	// 检查是否被锁定
	if config.LockedUntil != nil && config.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success":      false,
			"error":        "验证已被锁定，请稍后再试",
			"locked_until": config.LockedUntil,
		})
		return
	}

	valid := totp.Validate(req.Code, config.Secret)

	if !valid {
		// 尝试恢复码
		valid = h.validateRecoveryCode(&config, req.Code)
	}

	if !valid {
		// 增加失败计数
		globalConfig := h.get2FAGlobalConfig()
		newFailedCount := config.FailedCount + 1

		updates := map[string]interface{}{
			"failed_count": newFailedCount,
		}

		// 检查是否需要锁定
		if newFailedCount >= globalConfig.MaxFailedAttempts {
			lockUntil := time.Now().Add(time.Duration(globalConfig.LockoutDuration) * time.Second)
			updates["locked_until"] = lockUntil
			updates["failed_count"] = 0
		}

		h.db.Model(&config).Updates(updates)

		c.JSON(http.StatusBadRequest, gin.H{
			"success":       false,
			"error":         "验证码无效",
			"attempts_left": globalConfig.MaxFailedAttempts - newFailedCount,
		})
		return
	}

	// 验证成功，重置失败计数
	now := time.Now()
	h.db.Model(&config).Updates(map[string]interface{}{
		"failed_count":     0,
		"last_verified_at": now,
		"verify_count":     gorm.Expr("verify_count + 1"),
		"locked_until":     nil,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "验证成功"})
}

// RegenerateRecoveryCodes 重新生成恢复码
func (h *SecurityHandler) RegenerateRecoveryCodes(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"` // 需要当前2FA验证
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请提供当前2FA验证码"})
		return
	}

	var config models.TwoFactorConfig
	if err := h.db.Where("user_id = ? AND enabled = ?", userID, true).First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "2FA未启用"})
		return
	}

	// 验证当前2FA码
	if !totp.Validate(req.Code, config.Secret) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "验证码无效"})
		return
	}

	// 生成新的恢复码
	globalConfig := h.get2FAGlobalConfig()
	recoveryCodes := generateRecoveryCodes(globalConfig.RecoveryCodeCount)
	recoveryCodesJSON, _ := json.Marshal(recoveryCodes)

	if err := h.db.Model(&config).Updates(map[string]interface{}{
		"recovery_codes":      string(recoveryCodesJSON),
		"recovery_used_count": 0,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	h.logSecurityAudit(c, "2fa_recovery_regenerate", "user", fmt.Sprint(userID), "success", nil)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"recovery_codes": recoveryCodes,
		"message":        "恢复码已重新生成，请妥善保管",
	})
}

// === 全局2FA配置 ===

// Get2FAGlobalConfig 获取全局2FA配置
func (h *SecurityHandler) Get2FAGlobalConfig(c *gin.Context) {
	config := h.get2FAGlobalConfig()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// Update2FAGlobalConfig 更新全局2FA配置
func (h *SecurityHandler) Update2FAGlobalConfig(c *gin.Context) {
	var req models.TwoFactorGlobalConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	var config models.TwoFactorGlobalConfig
	h.db.First(&config)

	if config.ID == 0 {
		config = req
		if err := h.db.Create(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
	} else {
		if err := h.db.Model(&config).Updates(req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
	}

	h.logSecurityAudit(c, "2fa_global_config_update", "config", "", "success", nil)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}

// === OAuth 提供商管理 ===

// ListOAuthProviders 获取OAuth提供商列表
func (h *SecurityHandler) ListOAuthProviders(c *gin.Context) {
	var providers []models.OAuthProvider
	if err := h.db.Order("sort_order ASC").Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 如果没有配置，返回默认模板
	if len(providers) == 0 {
		providers = models.GetDefaultOAuthProviders()
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": providers})
}

// GetOAuthProvider 获取单个OAuth提供商
func (h *SecurityHandler) GetOAuthProvider(c *gin.Context) {
	name := c.Param("name")

	var provider models.OAuthProvider
	if err := h.db.Where("name = ?", name).First(&provider).Error; err != nil {
		// 返回默认模板
		for _, p := range models.GetDefaultOAuthProviders() {
			if p.Name == name {
				c.JSON(http.StatusOK, gin.H{"success": true, "data": p, "is_default": true})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "提供商不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": provider})
}

// UpdateOAuthProvider 更新OAuth提供商配置
func (h *SecurityHandler) UpdateOAuthProvider(c *gin.Context) {
	name := c.Param("name")

	var req models.OAuthProvider
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	req.Name = name

	var provider models.OAuthProvider
	result := h.db.Where("name = ?", name).First(&provider)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		if err := h.db.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		provider = req
	} else {
		// 更新现有记录
		if err := h.db.Model(&provider).Updates(req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
	}

	h.logSecurityAudit(c, "oauth_provider_update", "oauth_provider", name, "success", map[string]interface{}{
		"enabled": req.Enabled,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "data": provider})
}

// === 安全配置 ===

// GetSecurityConfig 获取安全配置
func (h *SecurityHandler) GetSecurityConfig(c *gin.Context) {
	var config models.SecurityConfig
	if err := h.db.First(&config).Error; err != nil {
		// 返回默认配置
		config = models.SecurityConfig{
			IPBlacklistEnabled:       true,
			IPWhitelistEnabled:       false,
			MaxLoginAttempts:         5,
			LoginLockoutDuration:     900,
			AutoBlockEnabled:         true,
			AutoBlockThreshold:       10,
			AutoBlockDuration:        3600,
			SessionTimeout:           3600,
			MaxConcurrentSessions:    5,
			PasswordMinLength:        8,
			PasswordRequireUppercase: true,
			PasswordRequireLowercase: true,
			PasswordRequireNumber:    true,
			PasswordRequireSpecial:   false,
			PasswordExpireDays:       0,
			AuditLogEnabled:          true,
			AuditLogRetentionDays:    90,
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}

// UpdateSecurityConfig 更新安全配置
func (h *SecurityHandler) UpdateSecurityConfig(c *gin.Context) {
	var req models.SecurityConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数"})
		return
	}

	var config models.SecurityConfig
	h.db.First(&config)

	if config.ID == 0 {
		if err := h.db.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		config = req
	} else {
		if err := h.db.Model(&config).Updates(req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
	}

	h.logSecurityAudit(c, "security_config_update", "config", "", "success", nil)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}

// === 审计日志 ===

// ListAuditLogs 获取审计日志列表
func (h *SecurityHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	action := c.Query("action")
	username := c.Query("username")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := h.db.Model(&models.SecurityAuditLog{})

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate)
	}

	var total int64
	query.Count(&total)

	var logs []models.SecurityAuditLog
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
		"total":   total,
		"page":    page,
		"size":    pageSize,
	})
}

// === 辅助方法 ===

func (h *SecurityHandler) isIPBlocked(ip string) (bool, string) {
	// 检查白名单（优先）
	var whitelistCount int64
	h.db.Model(&models.IPWhitelist{}).Where("ip = ? AND enabled = ?", ip, true).Count(&whitelistCount)
	if whitelistCount > 0 {
		return false, ""
	}

	// 检查黑名单
	var blacklist models.IPBlacklist
	err := h.db.Where("ip = ? AND enabled = ?", ip, true).First(&blacklist).Error
	if err == nil {
		// 检查临时封禁是否过期
		if blacklist.BlockType == "temporary" && blacklist.ExpireAt != nil && blacklist.ExpireAt.Before(time.Now()) {
			return false, ""
		}
		// 更新命中计数
		now := time.Now()
		h.db.Model(&blacklist).Updates(map[string]interface{}{
			"hit_count":   gorm.Expr("hit_count + 1"),
			"last_hit_at": now,
		})
		return true, blacklist.Reason
	}

	// TODO: 检查CIDR匹配

	return false, ""
}

func (h *SecurityHandler) get2FAGlobalConfig() models.TwoFactorGlobalConfig {
	var config models.TwoFactorGlobalConfig
	if err := h.db.First(&config).Error; err != nil {
		return models.TwoFactorGlobalConfig{
			Enabled:           false,
			EnforceForAdmin:   true,
			EnforceForAll:     false,
			AllowedTypes:      "totp",
			TOTPIssuer:        "AI-Infra-Matrix",
			TOTPDigits:        6,
			TOTPPeriod:        30,
			RecoveryCodeCount: 10,
			MaxFailedAttempts: 5,
			LockoutDuration:   300,
		}
	}
	return config
}

func (h *SecurityHandler) validateRecoveryCode(config *models.TwoFactorConfig, code string) bool {
	if config.RecoveryCodes == "" {
		return false
	}

	var codes []string
	if err := json.Unmarshal([]byte(config.RecoveryCodes), &codes); err != nil {
		return false
	}

	for i, c := range codes {
		if c == code {
			// 移除已使用的恢复码
			codes = append(codes[:i], codes[i+1:]...)
			newCodesJSON, _ := json.Marshal(codes)
			h.db.Model(config).Updates(map[string]interface{}{
				"recovery_codes":      string(newCodesJSON),
				"recovery_used_count": gorm.Expr("recovery_used_count + 1"),
			})
			return true
		}
	}
	return false
}

func (h *SecurityHandler) logSecurityAudit(c *gin.Context, action, resource, resourceID, status string, details map[string]interface{}) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	detailsJSON := ""
	if details != nil {
		if data, err := json.Marshal(details); err == nil {
			detailsJSON = string(data)
		}
	}

	auditLog := models.SecurityAuditLog{
		UserID:     userID.(uint),
		Username:   fmt.Sprintf("%v", username),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Status:     status,
		Details:    detailsJSON,
		CreatedAt:  time.Now(),
	}

	if err := h.db.Create(&auditLog).Error; err != nil {
		// 静默处理审计日志错误
		log.Printf("[SecurityAudit] Failed to create audit log: %v", err)
	}
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func generateRecoveryCodes(count int) []string {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		codes[i] = generateRecoveryCode()
	}
	return codes
}

func generateRecoveryCode() string {
	bytes := make([]byte, 5)
	rand.Read(bytes)
	code := base32.StdEncoding.EncodeToString(bytes)
	return code[:8] // 8字符的恢复码
}

// === 管理员2FA管理 ===

// AdminGet2FAStatus 管理员获取指定用户的2FA状态
func (h *SecurityHandler) AdminGet2FAStatus(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的用户ID"})
		return
	}

	var config models.TwoFactorConfig
	err = h.db.Where("user_id = ?", userID).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": map[string]interface{}{
				"enabled":     false,
				"type":        "",
				"setup_at":    nil,
				"user_id":     userID,
				"can_disable": false,
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"enabled":          config.Enabled,
			"type":             config.Type,
			"setup_at":         config.CreatedAt,
			"last_verified_at": config.LastVerifiedAt,
			"user_id":          userID,
			"can_disable":      true,
		},
	})
}

// AdminEnable2FA 管理员为指定用户强制启用2FA
func (h *SecurityHandler) AdminEnable2FA(c *gin.Context) {
	userIDParam := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的用户ID"})
		return
	}

	// 获取目标用户信息
	var targetUser models.User
	if err := h.db.First(&targetUser, targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
		return
	}

	// 检查是否已有2FA配置
	var existingConfig models.TwoFactorConfig
	err = h.db.Where("user_id = ?", targetUserID).First(&existingConfig).Error
	if err == nil && existingConfig.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "该用户已启用2FA"})
		return
	}

	// 生成TOTP密钥
	globalConfig := h.get2FAGlobalConfig()
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      globalConfig.TOTPIssuer,
		AccountName: targetUser.Username,
		Period:      uint(globalConfig.TOTPPeriod),
		Digits:      6,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "生成密钥失败"})
		return
	}

	// 生成恢复码
	recoveryCodes := generateRecoveryCodes(globalConfig.RecoveryCodeCount)
	recoveryCodesJSON, _ := json.Marshal(recoveryCodes)

	// 创建或更新配置
	config := models.TwoFactorConfig{
		UserID:        uint(targetUserID),
		Enabled:       true, // 管理员强制启用
		Type:          "totp",
		Secret:        key.Secret(),
		RecoveryCodes: string(recoveryCodesJSON),
	}

	if err := h.db.Where("user_id = ?", targetUserID).Assign(config).FirstOrCreate(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 获取管理员信息
	adminUserID, _ := c.Get("user_id")
	h.logSecurityAudit(c, "admin_2fa_enable", "user", userIDParam, "success", map[string]interface{}{
		"target_user":   targetUser.Username,
		"admin_user_id": adminUserID,
		"type":          "totp",
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已为用户 %s 启用2FA", targetUser.Username),
		"data": map[string]interface{}{
			"secret":         key.Secret(),
			"qr_code":        key.URL(),
			"issuer":         globalConfig.TOTPIssuer,
			"account":        targetUser.Username,
			"recovery_codes": recoveryCodes,
		},
	})
}

// AdminDisable2FA 管理员为指定用户禁用2FA
func (h *SecurityHandler) AdminDisable2FA(c *gin.Context) {
	userIDParam := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的用户ID"})
		return
	}

	// 获取目标用户信息
	var targetUser models.User
	if err := h.db.First(&targetUser, targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
		return
	}

	// 检查2FA状态
	var config models.TwoFactorConfig
	if err := h.db.Where("user_id = ?", targetUserID).First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "该用户未启用2FA"})
		return
	}

	// 禁用2FA
	if err := h.db.Model(&config).Updates(map[string]interface{}{
		"enabled":        false,
		"secret":         "",
		"recovery_codes": "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 获取管理员信息
	adminUserID, _ := c.Get("user_id")
	h.logSecurityAudit(c, "admin_2fa_disable", "user", userIDParam, "success", map[string]interface{}{
		"target_user":   targetUser.Username,
		"admin_user_id": adminUserID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已为用户 %s 禁用2FA", targetUser.Username),
	})
}
