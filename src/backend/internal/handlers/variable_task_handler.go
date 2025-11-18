package handlers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type VariableHandler struct {
	variableService *services.VariableService
}

func NewVariableHandler() *VariableHandler {
	return &VariableHandler{
		variableService: services.NewVariableService(),
	}
}

// @Summary 创建变量
// @Description 为项目添加新变量
// @Tags variables
// @Accept json
// @Produce json
// @Param variable body models.Variable true "变量信息"
// @Success 201 {object} models.Variable
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/variables [post]
func (h *VariableHandler) CreateVariable(c *gin.Context) {
	var variable models.Variable
	if err := c.ShouldBindJSON(&variable); err != nil {
		logrus.WithError(err).Error("Failed to bind variable data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.variableService.CreateVariable(&variable); err != nil {
		logrus.WithError(err).Error("Failed to create variable")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create variable"})
		return
	}

	c.JSON(http.StatusCreated, variable)
}

// @Summary 获取项目变量列表
// @Description 获取指定项目的所有变量
// @Tags variables
// @Produce json
// @Param project_id query int true "项目ID"
// @Success 200 {array} models.Variable
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/variables [get]
func (h *VariableHandler) GetVariables(c *gin.Context) {
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

	variables, err := h.variableService.GetVariables(uint(projectID))
	if err != nil {
		logrus.WithError(err).Error("Failed to get variables")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get variables"})
		return
	}

	c.JSON(http.StatusOK, variables)
}

// @Summary 更新变量
// @Description 更新变量信息
// @Tags variables
// @Accept json
// @Produce json
// @Param id path int true "变量ID"
// @Param variable body models.Variable true "变量信息"
// @Success 200 {object} models.Variable
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/variables/{id} [put]
func (h *VariableHandler) UpdateVariable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variable ID"})
		return
	}

	var variable models.Variable
	if err := c.ShouldBindJSON(&variable); err != nil {
		logrus.WithError(err).Error("Failed to bind variable data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.variableService.UpdateVariable(uint(id), &variable); err != nil {
		logrus.WithError(err).Error("Failed to update variable")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update variable"})
		return
	}

	c.JSON(http.StatusOK, variable)
}

// @Summary 删除变量
// @Description 删除指定变量
// @Tags variables
// @Param id path int true "变量ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/variables/{id} [delete]
func (h *VariableHandler) DeleteVariable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variable ID"})
		return
	}

	if err := h.variableService.DeleteVariable(uint(id)); err != nil {
		logrus.WithError(err).Error("Failed to delete variable")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete variable"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

type TaskHandler struct {
	taskService *services.TaskService
}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{
		taskService: services.NewTaskService(),
	}
}

// @Summary 创建任务
// @Description 为项目添加新任务
// @Tags tasks
// @Accept json
// @Produce json
// @Param task body models.Task true "任务信息"
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		logrus.WithError(err).Error("Failed to bind task data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.taskService.CreateTask(&task); err != nil {
		logrus.WithError(err).Error("Failed to create task")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// @Summary 获取项目任务列表
// @Description 获取指定项目的所有任务
// @Tags tasks
// @Produce json
// @Param project_id query int true "项目ID"
// @Success 200 {array} models.Task
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tasks [get]
func (h *TaskHandler) GetTasks(c *gin.Context) {
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

	tasks, err := h.taskService.GetTasks(uint(projectID))
	if err != nil {
		logrus.WithError(err).Error("Failed to get tasks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// @Summary 更新任务
// @Description 更新任务信息
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Param task body models.Task true "任务信息"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		logrus.WithError(err).Error("Failed to bind task data")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := h.taskService.UpdateTask(uint(id), &task); err != nil {
		logrus.WithError(err).Error("Failed to update task")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// @Summary 删除任务
// @Description 删除指定任务
// @Tags tasks
// @Param id path int true "任务ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	if err := h.taskService.DeleteTask(uint(id)); err != nil {
		logrus.WithError(err).Error("Failed to delete task")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
