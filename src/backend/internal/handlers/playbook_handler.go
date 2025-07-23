package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PlaybookHandler struct {
	playbookService *services.PlaybookService
	previewService  *services.PlaybookPreviewService
	validationService *services.PlaybookValidationService
}

func NewPlaybookHandler() *PlaybookHandler {
	return &PlaybookHandler{
		playbookService:   services.NewPlaybookService(),
		previewService:    services.NewPlaybookPreviewService(),
		validationService: services.NewPlaybookValidationService(),
	}
}

type GenerateRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
}

// @Summary 生成Playbook
// @Description 根据项目配置生成Ansible Playbook
// @Tags playbook
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "生成请求"
// @Success 200 {object} models.PlaybookGeneration
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/playbook/generate [post]
func (h *PlaybookHandler) GeneratePlaybook(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind generate request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	generation, err := h.playbookService.GeneratePlaybook(req.ProjectID, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to generate playbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate playbook"})
		return
	}

	c.JSON(http.StatusOK, generation)
}

// @Summary 下载Playbook文件
// @Description 下载生成的Playbook文件
// @Tags playbook
// @Param id path int true "生成记录ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/playbook/download/{id} [get]
func (h *PlaybookHandler) DownloadPlaybook(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logrus.WithError(err).WithField("id_param", idStr).Error("Invalid generation ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid generation ID"})
		return
	}

	filePath, fileName, err := h.playbookService.GetPlaybookFile(uint(id))
	if err != nil {
		logrus.WithError(err).WithField("generation_id", id).Error("Failed to get playbook file")
		c.JSON(http.StatusNotFound, gin.H{"error": "Playbook file not found"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"generation_id": id,
		"file_path": filePath,
		"file_name": fileName,
	}).Info("Starting file download")

	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

// @Summary 预览Playbook内容
// @Description 生成Playbook内容预览，包括YAML、Inventory和README
// @Tags playbook
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "预览请求"
// @Success 200 {object} services.PreviewContent
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/playbook/preview [post]
func (h *PlaybookHandler) PreviewPlaybook(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind preview request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	preview, err := h.previewService.GeneratePreview(req.ProjectID, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to generate preview")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate preview"})
		return
	}

	c.JSON(http.StatusOK, preview)
}

// @Summary 校验Playbook
// @Description 校验Playbook的语法、结构和最佳实践
// @Tags playbook
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "校验请求"
// @Success 200 {object} services.ValidationResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/playbook/validate [post]
func (h *PlaybookHandler) ValidatePlaybook(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind validation request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	validation, err := h.previewService.ValidatePreview(req.ProjectID, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to validate playbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate playbook"})
		return
	}

	c.JSON(http.StatusOK, validation)
}

// CompatibilityRequest 兼容性检查请求
type CompatibilityRequest struct {
	ProjectID       uint     `json:"project_id" binding:"required"`
	TargetVersions  []string `json:"target_versions" binding:"required"`
}

// @Summary 检查Ansible版本兼容性
// @Description 检查Playbook与不同Ansible版本的兼容性
// @Tags playbook
// @Accept json
// @Produce json
// @Param request body CompatibilityRequest true "兼容性检查请求"
// @Success 200 {object} map[string]services.AnsibleVersionCompatibility
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/playbook/compatibility [post]
func (h *PlaybookHandler) CheckCompatibility(c *gin.Context) {
	var req CompatibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind compatibility request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 生成预览内容以获取YAML
	preview, err := h.previewService.GeneratePreview(req.ProjectID, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to generate preview for compatibility check")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate preview"})
		return
	}

	// 检查兼容性
	compatibility, err := h.validationService.ValidateCompatibility([]byte(preview.PlaybookYAML), req.TargetVersions)
	if err != nil {
		logrus.WithError(err).Error("Failed to check compatibility")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check compatibility"})
		return
	}

	c.JSON(http.StatusOK, compatibility)
}

// @Summary 生成下载包
// @Description 生成包含Playbook、Inventory、README等文件的ZIP下载包
// @Tags playbook
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "下载包生成请求"
// @Success 200 {object} services.DownloadPackage
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/playbook/package [post]
func (h *PlaybookHandler) GeneratePackage(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Failed to bind package request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	downloadPackage, err := h.previewService.GenerateDownloadPackage(req.ProjectID, userID.(uint))
	if err != nil {
		logrus.WithError(err).Error("Failed to generate download package")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download package"})
		return
	}

	c.JSON(http.StatusOK, downloadPackage)
}

// @Summary 下载ZIP包
// @Description 下载生成的ZIP包文件
// @Tags playbook
// @Param path path string true "ZIP文件路径"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/playbook/download-zip/{path} [get]
func (h *PlaybookHandler) DownloadPackage(c *gin.Context) {
	zipPath := c.Param("path")
	if zipPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ZIP path"})
		return
	}

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Error("Failed to get current directory")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 安全检查：确保路径在outputs目录下，使用绝对路径
	fullPath := filepath.Join(currentDir, "outputs", "packages", zipPath)
	
	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logrus.WithFields(logrus.Fields{
			"zip_path": zipPath,
			"full_path": fullPath,
		}).Error("ZIP file not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "ZIP file not found"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"zip_path": zipPath,
		"full_path": fullPath,
	}).Info("Starting ZIP file download")

	// 设置下载头
	fileName := filepath.Base(fullPath)
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/zip")
	c.File(fullPath)
}
