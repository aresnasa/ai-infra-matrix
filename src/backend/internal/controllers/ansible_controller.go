package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AnsibleController 提供Ansible执行API
type AnsibleController struct {
	service *services.AnsibleService
	logger  *logrus.Logger
}

// NewAnsibleController 创建新的Ansible控制器
func NewAnsibleController() *AnsibleController {
	return &AnsibleController{
		service: services.NewAnsibleService(),
		logger:  logrus.New(),
	}
}

// ExecutePlaybook 执行Ansible playbook
// @Summary 执行Ansible playbook
// @Description 根据项目配置执行Ansible playbook
// @Tags ansible
// @Accept json
// @Produce json
// @Param request body models.AnsibleExecutionRequest true "执行请求"
// @Success 200 {object} models.AnsibleExecutionResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ansible/execute [post]
func (ctl *AnsibleController) ExecutePlaybook(c *gin.Context) {
	var req models.AnsibleExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	db := database.DB

	// 获取项目信息
	var project models.Project
	if err := db.Preload("Hosts").Preload("Variables").Preload("Tasks").First(&project, req.ProjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	// 检查用户权限（项目所有者或有执行权限）
	if project.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限操作此项目"})
		return
	}

	// 生成playbook内容
	playbookContent, err := ctl.generatePlaybookFromProject(project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成playbook失败: " + err.Error()})
		return
	}

	// 生成inventory内容
	var inventoryContent string
	if req.Inventory != "" {
		inventoryContent = req.Inventory
	} else {
		inventoryContent = ctl.service.GenerateInventoryFromProject(project)
	}

	// 创建执行记录
	execution := models.AnsibleExecution{
		ProjectID:     req.ProjectID,
		UserID:        userID.(uint),
		ExecutionType: req.ExecutionType,
		Environment:   req.Environment,
		ExtraVars:     req.ExtraVars,
		Status:        "pending",
	}

	if err := db.Create(&execution).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建执行记录失败: " + err.Error()})
		return
	}

	// 异步执行playbook
	go func() {
		ctx := context.Background()
		if err := ctl.service.ExecutePlaybook(ctx, &execution, playbookContent, inventoryContent); err != nil {
			ctl.logger.WithFields(logrus.Fields{
				"execution_id": execution.ID,
				"error":        err,
			}).Error("Ansible执行失败")
		}

		// 更新数据库中的执行结果
		db.Save(&execution)
	}()

	// 返回响应
	response := models.AnsibleExecutionResponse{
		ID:           execution.ID,
		Status:       execution.Status,
		Message:      "执行已开始",
		ExecutionURL: "/api/ansible/execution/" + strconv.Itoa(int(execution.ID)),
	}

	c.JSON(http.StatusOK, response)
}

// DryRunPlaybook 执行Ansible playbook dry-run
// @Summary 执行Ansible playbook dry-run
// @Description 以dry-run模式执行Ansible playbook，不会实际修改目标系统
// @Tags ansible
// @Accept json
// @Produce json
// @Param request body models.AnsibleExecutionRequest true "执行请求"
// @Success 200 {object} models.AnsibleExecutionResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ansible/dry-run [post]
func (ctl *AnsibleController) DryRunPlaybook(c *gin.Context) {
	var req models.AnsibleExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 强制设置为dry-run模式
	req.ExecutionType = "dry-run"

	// 重用ExecutePlaybook的逻辑
	ctl.ExecutePlaybook(c)
}

// GetExecutionStatus 获取执行状态
// @Summary 获取Ansible执行状态
// @Description 获取指定执行ID的状态信息
// @Tags ansible
// @Produce json
// @Param id path int true "执行ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /ansible/execution/{id}/status [get]
func (ctl *AnsibleController) GetExecutionStatus(c *gin.Context) {
	id := c.Param("id")
	db := database.DB

	var execution models.AnsibleExecution
	if err := db.Preload("Project").Preload("User").First(&execution, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "执行记录不存在"})
		return
	}

	// 检查权限
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	if execution.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限查看此执行记录"})
		return
	}

	// 获取实时状态
	currentStatus := ctl.service.GetExecutionStatus(&execution)
	if currentStatus != execution.Status {
		execution.Status = currentStatus
		db.Save(&execution)
	}

	response := map[string]interface{}{
		"id":             execution.ID,
		"status":         execution.Status,
		"execution_type": execution.ExecutionType,
		"environment":    execution.Environment,
		"start_time":     execution.StartTime,
		"end_time":       execution.EndTime,
		"duration":       execution.Duration,
		"exit_code":      execution.ExitCode,
		"project":        execution.Project,
	}

	c.JSON(http.StatusOK, response)
}

// GetExecutionLogs 获取执行日志
// @Summary 获取Ansible执行日志
// @Description 获取指定执行ID的详细日志
// @Tags ansible
// @Produce json
// @Param id path int true "执行ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /ansible/execution/{id}/logs [get]
func (ctl *AnsibleController) GetExecutionLogs(c *gin.Context) {
	id := c.Param("id")
	db := database.DB

	var execution models.AnsibleExecution
	if err := db.First(&execution, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "执行记录不存在"})
		return
	}

	// 检查权限
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	if execution.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限查看此执行记录"})
		return
	}

	// 格式化日志
	logs := ctl.service.FormatExecutionLogs(&execution)
	c.JSON(http.StatusOK, logs)
}

// ListExecutions 获取执行历史列表
// @Summary 获取Ansible执行历史列表
// @Description 获取当前用户的Ansible执行历史列表
// @Tags ansible
// @Produce json
// @Param project_id query int false "项目ID过滤"
// @Param status query string false "状态过滤"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页大小" default(10)
// @Success 200 {object} map[string]interface{}
// @Router /ansible/executions [get]
func (ctl *AnsibleController) ListExecutions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	db := database.DB
	query := db.Where("user_id = ?", userID.(uint))

	// 项目ID过滤
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := strconv.ParseUint(projectIDStr, 10, 32); err == nil {
			query = query.Where("project_id = ?", projectID)
		}
	}

	// 状态过滤
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 计算总数
	var total int64
	query.Model(&models.AnsibleExecution{}).Count(&total)

	// 获取数据
	var executions []models.AnsibleExecution
	err := query.Preload("Project").Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&executions).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询执行历史失败: " + err.Error()})
		return
	}

	response := map[string]interface{}{
		"executions": executions,
		"pagination": map[string]interface{}{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	}

	c.JSON(http.StatusOK, response)
}

// CancelExecution 取消执行
// @Summary 取消Ansible执行
// @Description 取消正在运行的Ansible执行
// @Tags ansible
// @Produce json
// @Param id path int true "执行ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /ansible/execution/{id}/cancel [post]
func (ctl *AnsibleController) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	db := database.DB

	var execution models.AnsibleExecution
	if err := db.First(&execution, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "执行记录不存在"})
		return
	}

	// 检查权限
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	if execution.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限操作此执行记录"})
		return
	}

	// 检查状态
	if execution.Status != "running" && execution.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能取消运行中或待执行的任务"})
		return
	}

	// 取消执行
	if err := ctl.service.CancelExecution(&execution); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消执行失败: " + err.Error()})
		return
	}

	// 更新数据库
	db.Save(&execution)

	c.JSON(http.StatusOK, gin.H{"message": "执行已取消", "status": execution.Status})
}

// generatePlaybookFromProject 从项目生成playbook内容
func (ctl *AnsibleController) generatePlaybookFromProject(project models.Project) (string, error) {
	// 这里使用简单的模板生成playbook
	// 在实际应用中，可能需要更复杂的模板引擎

	playbook := "---\n"
	playbook += "- name: " + project.Name + " Playbook\n"
	playbook += "  hosts: " + project.Name + "\n"
	playbook += "  become: true\n"
	playbook += "  gather_facts: true\n"

	if len(project.Variables) > 0 {
		playbook += "  vars:\n"
		for _, variable := range project.Variables {
			playbook += "    " + variable.Name + ": " + variable.Value + "\n"
		}
	}

	playbook += "  tasks:\n"

	if len(project.Tasks) == 0 {
		// 默认任务
		playbook += "    - name: Ping all hosts\n"
		playbook += "      ping:\n"
	} else {
		for _, task := range project.Tasks {
			playbook += "    - name: " + task.Name + "\n"
			playbook += "      " + task.Module + ":\n"

			// 解析任务参数
			if task.Args != "" {
				// 这里应该解析JSON格式的参数
				// 简化处理，直接输出
				playbook += "        " + task.Args + "\n"
			}
		}
	}

	return playbook, nil
}
