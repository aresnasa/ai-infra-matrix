package handlers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type HostHandler struct {
	hostService *services.HostService
}

func NewHostHandler() *HostHandler {
	return &HostHandler{
		hostService: services.NewHostService(),
	}
}

// @Summary 创建主机
// @Description 为项目添加新主机
// @Tags hosts
// @Accept json
// @Produce json
// @Param host body models.Host true "主机信息"
// @Success 201 {object} models.Host
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/hosts [post]
func (h *HostHandler) CreateHost(c *gin.Context) {
	var host models.Host
	if err := c.ShouldBindJSON(&host); err != nil {
		logrus.WithError(err).Error("Failed to bind host data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.hostService.CreateHost(&host); err != nil {
		logrus.WithError(err).Error("Failed to create host")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create host"})
		return
	}

	c.JSON(http.StatusCreated, host)
}

// @Summary 获取项目主机列表
// @Description 获取指定项目的所有主机
// @Tags hosts
// @Produce json
// @Param project_id query int true "项目ID"
// @Success 200 {array} models.Host
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/hosts [get]
func (h *HostHandler) GetHosts(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	hosts, err := h.hostService.GetHosts(uint(projectID))
	if err != nil {
		logrus.WithError(err).Error("Failed to get hosts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get hosts"})
		return
	}

	c.JSON(http.StatusOK, hosts)
}

// @Summary 更新主机
// @Description 更新主机信息
// @Tags hosts
// @Accept json
// @Produce json
// @Param id path int true "主机ID"
// @Param host body models.Host true "主机信息"
// @Success 200 {object} models.Host
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/hosts/{id} [put]
func (h *HostHandler) UpdateHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid host ID"})
		return
	}

	var host models.Host
	if err := c.ShouldBindJSON(&host); err != nil {
		logrus.WithError(err).Error("Failed to bind host data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.hostService.UpdateHost(uint(id), &host); err != nil {
		logrus.WithError(err).Error("Failed to update host")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update host"})
		return
	}

	c.JSON(http.StatusOK, host)
}

// @Summary 删除主机
// @Description 删除指定主机
// @Tags hosts
// @Param id path int true "主机ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/hosts/{id} [delete]
func (h *HostHandler) DeleteHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid host ID"})
		return
	}

	if err := h.hostService.DeleteHost(uint(id)); err != nil {
		logrus.WithError(err).Error("Failed to delete host")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete host"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
