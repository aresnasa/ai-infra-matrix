package controllers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// JobTemplateController 作业模板控制器
type JobTemplateController struct {
	templateService *services.JobTemplateService
}

// NewJobTemplateController 创建作业模板控制器
func NewJobTemplateController(db *gorm.DB) *JobTemplateController {
	return &JobTemplateController{
		templateService: services.NewJobTemplateService(db),
	}
}

// CreateTemplate 创建作业模板
// @Summary 创建作业模板
// @Description 创建新的作业模板
// @Tags 作业模板
// @Accept json
// @Produce json
// @Param request body models.CreateJobTemplateRequest true "创建模板请求"
// @Success 200 {object} models.Response{data=models.JobTemplate}
// @Router /api/job-templates [post]
func (jtc *JobTemplateController) CreateTemplate(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	var req models.CreateJobTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	template, err := jtc.templateService.CreateTemplate(c.Request.Context(), uint(userID), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "创建模板失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    template,
	})
}

// GetTemplate 获取单个作业模板
// @Summary 获取作业模板详情
// @Description 获取指定作业模板的详细信息
// @Tags 作业模板
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} models.Response{data=models.JobTemplate}
// @Router /api/job-templates/{id} [get]
func (jtc *JobTemplateController) GetTemplate(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的模板ID",
		})
		return
	}

	template, err := jtc.templateService.GetTemplate(c.Request.Context(), uint(id), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取模板失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    template,
	})
}

// ListTemplates 获取作业模板列表
// @Summary 获取作业模板列表
// @Description 获取用户可访问的作业模板列表
// @Tags 作业模板
// @Accept json
// @Produce json
// @Param category query string false "分类筛选"
// @Param is_public query bool false "是否只显示公开模板"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} models.Response{data=models.JobTemplateListResponse}
// @Router /api/job-templates [get]
func (jtc *JobTemplateController) ListTemplates(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	// 解析查询参数
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var isPublic *bool
	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		if isPublicValue, err := strconv.ParseBool(isPublicStr); err == nil {
			isPublic = &isPublicValue
		}
	}

	templates, total, err := jtc.templateService.ListTemplates(c.Request.Context(), uint(userID), category, isPublic, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取模板列表失败: " + err.Error(),
		})
		return
	}

	response := models.JobTemplateListResponse{
		Templates: templates,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    response,
	})
}

// UpdateTemplate 更新作业模板
// @Summary 更新作业模板
// @Description 更新指定的作业模板
// @Tags 作业模板
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param request body models.UpdateJobTemplateRequest true "更新模板请求"
// @Success 200 {object} models.Response{data=models.JobTemplate}
// @Router /api/job-templates/{id} [put]
func (jtc *JobTemplateController) UpdateTemplate(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的模板ID",
		})
		return
	}

	var req models.UpdateJobTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	template, err := jtc.templateService.UpdateTemplate(c.Request.Context(), uint(id), uint(userID), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "更新模板失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    template,
	})
}

// DeleteTemplate 删除作业模板
// @Summary 删除作业模板
// @Description 删除指定的作业模板
// @Tags 作业模板
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} models.Response
// @Router /api/job-templates/{id} [delete]
func (jtc *JobTemplateController) DeleteTemplate(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的模板ID",
		})
		return
	}

	if err := jtc.templateService.DeleteTemplate(c.Request.Context(), uint(id), uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除模板失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
	})
}

// GetTemplateCategories 获取模板分类
// @Summary 获取模板分类列表
// @Description 获取用户可访问的所有模板分类
// @Tags 作业模板
// @Accept json
// @Produce json
// @Success 200 {object} models.Response{data=[]string}
// @Router /api/job-templates/categories [get]
func (jtc *JobTemplateController) GetTemplateCategories(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	categories, err := jtc.templateService.GetTemplateCategories(c.Request.Context(), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取分类失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    categories,
	})
}

// RegisterRoutes 注册路由
func (jtc *JobTemplateController) RegisterRoutes(r *gin.RouterGroup) {
	templates := r.Group("/job-templates")
	// 作业模板管理需要认证
	templates.Use(middleware.AuthMiddlewareWithSession())
	{
		templates.POST("", jtc.CreateTemplate)
		templates.GET("", jtc.ListTemplates)
		templates.GET("/categories", jtc.GetTemplateCategories)
		templates.GET("/:id", jtc.GetTemplate)
		templates.PUT("/:id", jtc.UpdateTemplate)
		templates.DELETE("/:id", jtc.DeleteTemplate)
	}
}
