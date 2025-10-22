package handlers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	projectService *services.ProjectService
	rbacService    *services.RBACService
}

func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{
		projectService: services.NewProjectService(),
		rbacService:    services.NewRBACService(db),
	}
}

// @Summary 创建项目
// @Description 创建新的Ansible项目
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project body models.Project true "项目信息"
// @Success 201 {object} models.Project
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		logrus.WithError(err).Error("Failed to bind project data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.projectService.CreateProject(&project, userID); err != nil {
		logrus.WithError(err).Error("Failed to create project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// @Summary 获取项目列表
// @Description 获取当前用户的所有项目
// @Tags projects
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Project
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects [get]
func (h *ProjectHandler) GetProjects(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	projects, err := h.projectService.GetProjectsWithRBAC(userID, h.rbacService)
	if err != nil {
		logrus.WithError(err).Error("Failed to get projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get projects"})
		return
	}

	c.JSON(http.StatusOK, projects)
}

// @Summary 获取项目详情
// @Description 根据ID获取项目详细信息
// @Tags projects
// @Produce json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 200 {object} models.Project
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/projects/{id} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	project, err := h.projectService.GetProject(uint(id), userID, h.rbacService)
	if err != nil {
		logrus.WithError(err).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// @Summary 更新项目
// @Description 更新项目信息
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Param project body models.Project true "项目信息"
// @Success 200 {object} models.Project
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		logrus.WithError(err).Error("Failed to bind project data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.projectService.UpdateProject(uint(id), userID, &project); err != nil {
		logrus.WithError(err).Error("Failed to update project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// @Summary 删除项目
// @Description 删除指定项目
// @Tags projects
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := h.projectService.DeleteProject(uint(id), userID); err != nil {
		logrus.WithError(err).Error("Failed to delete project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// @Summary 软删除项目（移至回收站）
// @Description 将项目移至回收站而不是永久删除
// @Tags projects
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects/{id}/soft-delete [patch]
func (h *ProjectHandler) SoftDeleteProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := h.projectService.SoftDeleteProject(uint(id), userID); err != nil {
		logrus.WithError(err).Error("Failed to soft delete project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move project to trash"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project moved to trash successfully"})
}

// @Summary 获取回收站项目
// @Description 获取当前用户的已删除项目列表
// @Tags projects
// @Security BearerAuth
// @Success 200 {array} models.Project
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects/trash [get]
func (h *ProjectHandler) GetDeletedProjects(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	projects, err := h.projectService.GetDeletedProjects(userID, h.rbacService)
	if err != nil {
		logrus.WithError(err).Error("Failed to get deleted projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get deleted projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// @Summary 从回收站恢复项目
// @Description 将项目从回收站恢复到正常状态
// @Tags projects
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects/{id}/restore [patch]
func (h *ProjectHandler) RestoreProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := h.projectService.RestoreProject(uint(id), userID, h.rbacService); err != nil {
		logrus.WithError(err).Error("Failed to restore project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project restored successfully"})
}

// @Summary 永久删除项目
// @Description 永久删除回收站中的项目（仅管理员）
// @Tags projects
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/projects/{id}/force [delete]
func (h *ProjectHandler) ForceDeleteProject(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 检查是否为管理员
	if !h.rbacService.IsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can permanently delete projects"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := h.projectService.ForceDeleteProject(uint(id), userID); err != nil {
		logrus.WithError(err).Error("Failed to force delete project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to permanently delete project"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
