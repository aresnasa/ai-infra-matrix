package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// JupyterLabTemplateController JupyterLab模板控制器
type JupyterLabTemplateController struct {
	templateService services.JupyterLabTemplateService
}

// NewJupyterLabTemplateController 创建控制器实例
func NewJupyterLabTemplateController() *JupyterLabTemplateController {
	return &JupyterLabTemplateController{
		templateService: services.NewJupyterLabTemplateService(),
	}
}

// CreateTemplate 创建模板
func (ctrl *JupyterLabTemplateController) CreateTemplate(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req struct {
		Name            string                          `json:"name" binding:"required"`
		Description     string                          `json:"description"`
		PythonVersion   string                          `json:"python_version"`
		CondaVersion    string                          `json:"conda_version"`
		BaseImage       string                          `json:"base_image"`
		Requirements    []string                        `json:"requirements"`
		CondaPackages   []string                        `json:"conda_packages"`
		SystemPackages  []string                        `json:"system_packages"`
		EnvironmentVars []models.EnvironmentVariable    `json:"environment_vars"`
		StartupScript   string                          `json:"startup_script"`
		ResourceQuota   *models.JupyterLabResourceQuota `json:"resource_quota"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建模板
	template := &models.JupyterLabTemplate{
		Name:          req.Name,
		Description:   req.Description,
		PythonVersion: req.PythonVersion,
		CondaVersion:  req.CondaVersion,
		BaseImage:     req.BaseImage,
		StartupScript: req.StartupScript,
		IsActive:      true,
		CreatedBy:     userID,
	}

	// 设置默认值
	if template.PythonVersion == "" {
		template.PythonVersion = "3.11"
	}
	if template.CondaVersion == "" {
		template.CondaVersion = "23.7.0"
	}
	if template.BaseImage == "" {
		template.BaseImage = "jupyter/scipy-notebook:latest"
	}

	// 序列化列表字段
	if err := ctrl.setTemplateFields(template, req.Requirements, req.CondaPackages, req.SystemPackages, req.EnvironmentVars); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "序列化字段失败: " + err.Error()})
		return
	}

	if err := ctrl.templateService.CreateTemplate(template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建模板失败: " + err.Error()})
		return
	}

	// 创建资源配额
	if req.ResourceQuota != nil {
		req.ResourceQuota.TemplateID = template.ID
		if err := ctrl.templateService.CreateResourceQuota(req.ResourceQuota); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建资源配额失败: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// GetTemplate 获取模板详情
func (ctrl *JupyterLabTemplateController) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
		return
	}

	template, err := ctrl.templateService.GetTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// ListTemplates 获取模板列表
func (ctrl *JupyterLabTemplateController) ListTemplates(c *gin.Context) {
	userID, _ := middleware.GetCurrentUserID(c)
	includeInactive := c.Query("include_inactive") == "true"

	templates, err := ctrl.templateService.ListTemplates(userID, includeInactive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模板列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// UpdateTemplate 更新模板
func (ctrl *JupyterLabTemplateController) UpdateTemplate(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
		return
	}

	// 检查模板是否存在和权限
	template, err := ctrl.templateService.GetTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	// 只有创建者或管理员可以修改
	if template.CreatedBy != userID && template.CreatedBy != 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限修改此模板"})
		return
	}

	var req struct {
		Name            string                          `json:"name"`
		Description     string                          `json:"description"`
		PythonVersion   string                          `json:"python_version"`
		CondaVersion    string                          `json:"conda_version"`
		BaseImage       string                          `json:"base_image"`
		Requirements    []string                        `json:"requirements"`
		CondaPackages   []string                        `json:"conda_packages"`
		SystemPackages  []string                        `json:"system_packages"`
		EnvironmentVars []models.EnvironmentVariable    `json:"environment_vars"`
		StartupScript   string                          `json:"startup_script"`
		IsActive        *bool                           `json:"is_active"`
		ResourceQuota   *models.JupyterLabResourceQuota `json:"resource_quota"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新字段
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.PythonVersion != "" {
		template.PythonVersion = req.PythonVersion
	}
	if req.CondaVersion != "" {
		template.CondaVersion = req.CondaVersion
	}
	if req.BaseImage != "" {
		template.BaseImage = req.BaseImage
	}
	if req.StartupScript != "" {
		template.StartupScript = req.StartupScript
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	// 序列化列表字段
	if err := ctrl.setTemplateFields(template, req.Requirements, req.CondaPackages, req.SystemPackages, req.EnvironmentVars); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "序列化字段失败: " + err.Error()})
		return
	}

	if err := ctrl.templateService.UpdateTemplate(template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新模板失败: " + err.Error()})
		return
	}

	// 更新资源配额
	if req.ResourceQuota != nil {
		req.ResourceQuota.TemplateID = template.ID
		if template.ResourceQuota != nil {
			// 更新现有配额
			req.ResourceQuota.ID = template.ResourceQuota.ID
			if err := ctrl.templateService.UpdateResourceQuota(req.ResourceQuota); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "更新资源配额失败: " + err.Error()})
				return
			}
		} else {
			// 创建新配额
			if err := ctrl.templateService.CreateResourceQuota(req.ResourceQuota); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建资源配额失败: " + err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// DeleteTemplate 删除模板
func (ctrl *JupyterLabTemplateController) DeleteTemplate(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
		return
	}

	// 检查模板是否存在和权限
	template, err := ctrl.templateService.GetTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	// 只有创建者可以删除，系统模板不能删除
	if template.CreatedBy == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统模板不能删除"})
		return
	}
	if template.CreatedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限删除此模板"})
		return
	}

	if err := ctrl.templateService.DeleteTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除模板失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// CloneTemplate 克隆模板
func (ctrl *JupyterLabTemplateController) CloneTemplate(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cloned, err := ctrl.templateService.CloneTemplate(uint(id), req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "克隆模板失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": cloned})
}

// SetDefaultTemplate 设置默认模板
func (ctrl *JupyterLabTemplateController) SetDefaultTemplate(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
		return
	}

	// 检查模板是否存在
	template, err := ctrl.templateService.GetTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	// 只有创建者可以设置为默认
	if template.CreatedBy != userID && template.CreatedBy != 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限设置此模板为默认"})
		return
	}

	if err := ctrl.templateService.SetDefaultTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置默认模板失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "设置成功"})
}

// ExportTemplate 导出模板
func (ctrl *JupyterLabTemplateController) ExportTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
		return
	}

	export, err := ctrl.templateService.ExportTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出模板失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": export})
}

// ImportTemplate 导入模板
func (ctrl *JupyterLabTemplateController) ImportTemplate(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := ctrl.templateService.ImportTemplate(req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导入模板失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// CreatePredefinedTemplates 创建预定义模板
func (ctrl *JupyterLabTemplateController) CreatePredefinedTemplates(c *gin.Context) {
	if err := ctrl.templateService.CreatePredefinedTemplates(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建预定义模板失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "预定义模板创建成功"})
}

// 实例管理相关方法

// CreateInstance 创建实例
func (ctrl *JupyterLabTemplateController) CreateInstance(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req struct {
		TemplateID uint   `json:"template_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查模板是否存在
	template, err := ctrl.templateService.GetTemplate(req.TemplateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	if !template.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "模板未激活"})
		return
	}

	instance := &models.JupyterLabInstance{
		UserID:     userID,
		TemplateID: req.TemplateID,
		Name:       req.Name,
		Status:     "pending",
		Namespace:  "jupyterhub",
	}

	if err := ctrl.templateService.CreateInstance(instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建实例失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": instance})
}

// ListInstances 获取用户实例列表
func (ctrl *JupyterLabTemplateController) ListInstances(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	instances, err := ctrl.templateService.ListUserInstances(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取实例列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": instances})
}

// GetInstance 获取实例详情
func (ctrl *JupyterLabTemplateController) GetInstance(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的实例ID"})
		return
	}

	instance, err := ctrl.templateService.GetInstance(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "实例不存在"})
		return
	}

	// 检查权限
	if instance.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问此实例"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": instance})
}

// DeleteInstance 删除实例
func (ctrl *JupyterLabTemplateController) DeleteInstance(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的实例ID"})
		return
	}

	// 检查实例是否存在和权限
	instance, err := ctrl.templateService.GetInstance(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "实例不存在"})
		return
	}

	if instance.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限删除此实例"})
		return
	}

	if err := ctrl.templateService.DeleteInstance(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除实例失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 辅助方法
func (ctrl *JupyterLabTemplateController) setTemplateFields(template *models.JupyterLabTemplate, requirements, condaPackages, systemPackages []string, envVars []models.EnvironmentVariable) error {
	if requirements != nil {
		if err := ctrl.setJSONField(&template.Requirements, requirements); err != nil {
			return err
		}
	}
	if condaPackages != nil {
		if err := ctrl.setJSONField(&template.CondaPackages, condaPackages); err != nil {
			return err
		}
	}
	if systemPackages != nil {
		if err := ctrl.setJSONField(&template.SystemPackages, systemPackages); err != nil {
			return err
		}
	}
	if envVars != nil {
		if err := template.SetEnvironmentVars(envVars); err != nil {
			return err
		}
	}
	return nil
}

func (ctrl *JupyterLabTemplateController) setJSONField(field *string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	*field = string(jsonData)
	return nil
}
