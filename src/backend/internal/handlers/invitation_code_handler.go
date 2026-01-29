package handlers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// InvitationCodeHandler 邀请码处理器
type InvitationCodeHandler struct {
	invitationService *services.InvitationCodeService
}

// NewInvitationCodeHandler 创建邀请码处理器实例
func NewInvitationCodeHandler() *InvitationCodeHandler {
	return &InvitationCodeHandler{
		invitationService: services.NewInvitationCodeService(),
	}
}

// CreateInvitationCode 创建邀请码
// @Summary 创建邀请码
// @Description 管理员创建新的邀请码
// @Tags 邀请码管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateInvitationCodeRequest true "创建邀请码请求"
// @Success 201 {object} models.InvitationCode
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /admin/invitation-codes [post]
func (h *InvitationCodeHandler) CreateInvitationCode(c *gin.Context) {
	var req models.CreateInvitationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 检查是否批量创建
	if req.Count > 1 {
		codes, err := h.invitationService.BatchCreateInvitationCodes(&req, userID.(uint))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": "邀请码创建成功",
			"count":   len(codes),
			"codes":   codes,
		})
		return
	}

	code, err := h.invitationService.CreateInvitationCode(&req, userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "邀请码创建成功",
		"code":    code,
	})
}

// ListInvitationCodes 获取邀请码列表
// @Summary 获取邀请码列表
// @Description 管理员获取邀请码列表
// @Tags 邀请码管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param include_expired query bool false "包含过期的" default(false)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /admin/invitation-codes [get]
func (h *InvitationCodeHandler) ListInvitationCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	includeExpired := c.Query("include_expired") == "true"

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	codes, total, err := h.invitationService.ListInvitationCodes(page, pageSize, includeExpired)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      codes,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetInvitationCode 获取邀请码详情
// @Summary 获取邀请码详情
// @Description 管理员获取邀请码详情
// @Tags 邀请码管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "邀请码ID"
// @Success 200 {object} models.InvitationCode
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /admin/invitation-codes/{id} [get]
func (h *InvitationCodeHandler) GetInvitationCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	code, err := h.invitationService.GetInvitationCode(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 获取使用记录
	usages, _ := h.invitationService.GetInvitationCodeUsages(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"code":   code,
		"usages": usages,
	})
}

// DisableInvitationCode 禁用邀请码
// @Summary 禁用邀请码
// @Description 管理员禁用邀请码
// @Tags 邀请码管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "邀请码ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /admin/invitation-codes/{id}/disable [post]
func (h *InvitationCodeHandler) DisableInvitationCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.invitationService.DisableInvitationCode(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "邀请码已禁用"})
}

// EnableInvitationCode 启用邀请码
// @Summary 启用邀请码
// @Description 管理员启用邀请码
// @Tags 邀请码管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "邀请码ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /admin/invitation-codes/{id}/enable [post]
func (h *InvitationCodeHandler) EnableInvitationCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.invitationService.EnableInvitationCode(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "邀请码已启用"})
}

// DeleteInvitationCode 删除邀请码
// @Summary 删除邀请码
// @Description 管理员删除邀请码
// @Tags 邀请码管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "邀请码ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /admin/invitation-codes/{id} [delete]
func (h *InvitationCodeHandler) DeleteInvitationCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.invitationService.DeleteInvitationCode(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "邀请码已删除"})
}

// GetInvitationCodeStatistics 获取邀请码统计
// @Summary 获取邀请码统计
// @Description 管理员获取邀请码统计信息
// @Tags 邀请码管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /admin/invitation-codes/statistics [get]
func (h *InvitationCodeHandler) GetInvitationCodeStatistics(c *gin.Context) {
	stats, err := h.invitationService.GetStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ValidateInvitationCode 验证邀请码（公开API，用于前端验证）
// @Summary 验证邀请码
// @Description 验证邀请码是否有效（无需登录）
// @Tags 邀请码管理
// @Accept json
// @Produce json
// @Param code query string true "邀请码"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /auth/validate-invitation-code [get]
func (h *InvitationCodeHandler) ValidateInvitationCode(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": "邀请码不能为空"})
		return
	}

	invitation, err := h.invitationService.ValidateCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":         true,
		"role_template": invitation.RoleTemplate,
		"description":   invitation.Description,
	})
}
