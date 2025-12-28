package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// AuditHandler 审计日志处理器
type AuditHandler struct {
	auditService *services.AuditService
}

// NewAuditHandler 创建审计日志处理器
func NewAuditHandler() *AuditHandler {
	return &AuditHandler{
		auditService: services.GetAuditService(),
	}
}

// ListAuditLogs godoc
// @Summary 查询审计日志列表
// @Description 分页查询审计日志，支持多种过滤条件
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category query string false "审计类别: ansible, slurm, saltstack, role_template, kubernetes, monitor, admin, security"
// @Param action query string false "操作动作"
// @Param status query string false "状态: success, failed, pending"
// @Param severity query string false "严重程度: info, warning, critical, alert"
// @Param user_id query int false "用户ID"
// @Param username query string false "用户名（模糊匹配）"
// @Param resource_type query string false "资源类型"
// @Param resource_id query string false "资源ID"
// @Param client_ip query string false "客户端IP"
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Param keywords query string false "关键词搜索"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param sort_by query string false "排序字段" default(created_at)
// @Param sort_order query string false "排序方向: asc, desc" default(desc)
// @Success 200 {object} models.AuditLogResponse
// @Failure 400 {object} models.Response
// @Failure 401 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var req models.AuditLogQueryRequest

	// 解析查询参数
	req.Category = c.Query("category")
	req.Action = c.Query("action")
	req.Status = c.Query("status")
	req.Severity = c.Query("severity")
	req.Username = c.Query("username")
	req.ResourceType = c.Query("resource_type")
	req.ResourceID = c.Query("resource_id")
	req.ClientIP = c.Query("client_ip")
	req.Keywords = c.Query("keywords")
	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")

	if userID := c.Query("user_id"); userID != "" {
		if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
			req.UserID = uint(id)
		}
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			req.StartDate = t
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			req.EndDate = t
		}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = ps
		}
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	result, err := h.auditService.QueryAuditLogs(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询审计日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}

// GetAuditLog godoc
// @Summary 获取单条审计日志详情
// @Description 根据ID获取审计日志详情，包括变更明细
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "审计日志ID"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 404 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/logs/{id} [get]
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的ID",
		})
		return
	}

	log, details, err := h.auditService.GetAuditLogByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "审计日志不存在",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"log":     log,
			"details": details,
		},
	})
}

// GetAuditStatistics godoc
// @Summary 获取审计统计信息
// @Description 获取审计日志的统计信息，包括分类统计、趋势等
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} models.AuditStatisticsResponse
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/statistics [get]
func (h *AuditHandler) GetAuditStatistics(c *gin.Context) {
	var startDate, endDate time.Time

	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = t
		}
	}

	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endDate = t
		}
	}

	stats, err := h.auditService.GetAuditStatistics(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取统计信息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    stats,
	})
}

// GetAuditCategories godoc
// @Summary 获取审计类别列表
// @Description 获取所有支持的审计类别
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Response
// @Router /api/v1/audit/categories [get]
func (h *AuditHandler) GetAuditCategories(c *gin.Context) {
	categories := []gin.H{
		{"value": string(models.AuditCategoryAnsible), "label": "Ansible", "description": "Ansible 自动化操作"},
		{"value": string(models.AuditCategorySlurm), "label": "SLURM", "description": "SLURM 集群操作"},
		{"value": string(models.AuditCategorySaltstack), "label": "SaltStack", "description": "SaltStack 配置管理"},
		{"value": string(models.AuditCategoryRoleTemplate), "label": "角色模板", "description": "角色和权限管理"},
		{"value": string(models.AuditCategoryKubernetes), "label": "Kubernetes", "description": "Kubernetes 资源操作"},
		{"value": string(models.AuditCategoryMonitor), "label": "监控", "description": "监控配置操作"},
		{"value": string(models.AuditCategoryAdmin), "label": "管理员操作", "description": "系统管理操作"},
		{"value": string(models.AuditCategorySecurity), "label": "安全", "description": "安全相关操作"},
		{"value": string(models.AuditCategorySystem), "label": "系统", "description": "系统级操作"},
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    categories,
	})
}

// GetAuditActions godoc
// @Summary 获取审计动作列表
// @Description 获取指定类别下支持的审计动作
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category query string false "审计类别"
// @Success 200 {object} models.Response
// @Router /api/v1/audit/actions [get]
func (h *AuditHandler) GetAuditActions(c *gin.Context) {
	category := c.Query("category")

	// 通用动作
	commonActions := []gin.H{
		{"value": string(models.AuditActionCreate), "label": "创建"},
		{"value": string(models.AuditActionUpdate), "label": "更新"},
		{"value": string(models.AuditActionDelete), "label": "删除"},
		{"value": string(models.AuditActionRead), "label": "查看"},
		{"value": string(models.AuditActionExecute), "label": "执行"},
		{"value": string(models.AuditActionEnable), "label": "启用"},
		{"value": string(models.AuditActionDisable), "label": "禁用"},
	}

	// 类别特定动作
	categoryActions := map[string][]gin.H{
		string(models.AuditCategoryAnsible): {
			{"value": string(models.AuditActionPlaybookRun), "label": "运行 Playbook"},
			{"value": string(models.AuditActionPlaybookDryRun), "label": "Playbook 预演"},
			{"value": string(models.AuditActionInventoryUpdate), "label": "更新 Inventory"},
		},
		string(models.AuditCategorySaltstack): {
			{"value": string(models.AuditActionSaltExecute), "label": "Salt 命令执行"},
			{"value": string(models.AuditActionSaltStateApply), "label": "Salt 状态应用"},
			{"value": string(models.AuditActionSaltPillarUpdate), "label": "Salt Pillar 更新"},
			{"value": string(models.AuditActionSaltKeyAccept), "label": "接受 Salt Key"},
			{"value": string(models.AuditActionSaltKeyReject), "label": "拒绝 Salt Key"},
			{"value": string(models.AuditActionSaltKeyDelete), "label": "删除 Salt Key"},
		},
		string(models.AuditCategorySlurm): {
			{"value": string(models.AuditActionJobSubmit), "label": "提交作业"},
			{"value": string(models.AuditActionJobCancel), "label": "取消作业"},
			{"value": string(models.AuditActionNodeAdd), "label": "添加节点"},
			{"value": string(models.AuditActionNodeRemove), "label": "移除节点"},
			{"value": string(models.AuditActionNodeDrain), "label": "节点排空"},
			{"value": string(models.AuditActionNodeResume), "label": "恢复节点"},
			{"value": string(models.AuditActionClusterDeploy), "label": "集群部署"},
		},
		string(models.AuditCategoryKubernetes): {
			{"value": string(models.AuditActionK8sResourceCreate), "label": "创建 K8s 资源"},
			{"value": string(models.AuditActionK8sResourceUpdate), "label": "更新 K8s 资源"},
			{"value": string(models.AuditActionK8sResourceDelete), "label": "删除 K8s 资源"},
			{"value": string(models.AuditActionK8sHelmInstall), "label": "Helm 安装"},
			{"value": string(models.AuditActionK8sHelmUpgrade), "label": "Helm 升级"},
			{"value": string(models.AuditActionK8sHelmUninstall), "label": "Helm 卸载"},
			{"value": string(models.AuditActionScale), "label": "扩缩容"},
			{"value": string(models.AuditActionRestart), "label": "重启"},
		},
		string(models.AuditCategoryRoleTemplate): {
			{"value": string(models.AuditActionRoleAssign), "label": "分配角色"},
			{"value": string(models.AuditActionRoleRevoke), "label": "撤销角色"},
			{"value": string(models.AuditActionPermissionGrant), "label": "授予权限"},
			{"value": string(models.AuditActionPermissionRevoke), "label": "撤销权限"},
		},
		string(models.AuditCategoryMonitor): {
			{"value": string(models.AuditActionAlertCreate), "label": "创建告警"},
			{"value": string(models.AuditActionAlertAck), "label": "确认告警"},
			{"value": string(models.AuditActionAlertResolve), "label": "解决告警"},
			{"value": string(models.AuditActionDashboardCreate), "label": "创建仪表板"},
			{"value": string(models.AuditActionDashboardUpdate), "label": "更新仪表板"},
		},
		string(models.AuditCategoryAdmin): {
			{"value": string(models.AuditActionUserCreate), "label": "创建用户"},
			{"value": string(models.AuditActionUserUpdate), "label": "更新用户"},
			{"value": string(models.AuditActionUserDelete), "label": "删除用户"},
			{"value": string(models.AuditActionUserLock), "label": "锁定用户"},
			{"value": string(models.AuditActionUserUnlock), "label": "解锁用户"},
			{"value": string(models.AuditActionPasswordReset), "label": "重置密码"},
			{"value": string(models.AuditActionConfigUpdate), "label": "更新配置"},
			{"value": string(models.AuditActionBackupCreate), "label": "创建备份"},
			{"value": string(models.AuditActionBackupRestore), "label": "恢复备份"},
		},
	}

	result := commonActions
	if category != "" {
		if actions, ok := categoryActions[category]; ok {
			result = append(result, actions...)
		}
	} else {
		// 返回所有动作
		for _, actions := range categoryActions {
			result = append(result, actions...)
		}
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}

// ListAuditConfigs godoc
// @Summary 获取审计配置列表
// @Description 获取所有审计类别的配置
// @Tags 审计配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/configs [get]
func (h *AuditHandler) ListAuditConfigs(c *gin.Context) {
	configs, err := h.auditService.GetAllAuditConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    configs,
	})
}

// GetAuditConfig godoc
// @Summary 获取指定类别的审计配置
// @Description 获取指定审计类别的配置详情
// @Tags 审计配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category path string true "审计类别"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.Response
// @Router /api/v1/audit/configs/{category} [get]
func (h *AuditHandler) GetAuditConfig(c *gin.Context) {
	category := models.AuditCategory(c.Param("category"))

	config, err := h.auditService.GetAuditConfig(category)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "配置不存在",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// UpdateAuditConfig godoc
// @Summary 更新审计配置
// @Description 更新指定类别的审计配置
// @Tags 审计配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category path string true "审计类别"
// @Param config body models.AuditConfig true "配置内容"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/configs/{category} [put]
func (h *AuditHandler) UpdateAuditConfig(c *gin.Context) {
	category := models.AuditCategory(c.Param("category"))

	var config models.AuditConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	config.Category = category

	if err := h.auditService.UpdateAuditConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "更新配置失败: " + err.Error(),
		})
		return
	}

	// 记录配置变更
	h.auditService.LogAdminOperation(c, models.AuditActionConfigUpdate, "audit_config", string(category), string(category), nil, &config, nil)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "配置更新成功",
	})
}

// ExportAuditLogs godoc
// @Summary 导出审计日志
// @Description 创建审计日志导出请求
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.AuditExportRequestCreate true "导出请求"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/export [post]
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	var req models.AuditExportRequestCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	// TODO: 实现异步导出逻辑
	// 1. 创建导出请求记录
	// 2. 启动异步任务生成导出文件
	// 3. 返回请求ID

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "导出请求已创建，请稍后查看导出结果",
		Data: gin.H{
			"request_id": 0, // TODO: 返回实际的请求ID
		},
	})
}

// GetUserAuditLogs godoc
// @Summary 获取当前用户的操作记录
// @Description 获取当前登录用户的审计日志
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} models.AuditLogResponse
// @Failure 401 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/my-logs [get]
func (h *AuditHandler) GetUserAuditLogs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	req := &models.AuditLogQueryRequest{
		UserID: userID.(uint),
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = ps
		}
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	result, err := h.auditService.QueryAuditLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}

// GetResourceAuditLogs godoc
// @Summary 获取资源的操作历史
// @Description 获取指定资源的审计日志历史
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resource_type path string true "资源类型"
// @Param resource_id path string true "资源ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} models.AuditLogResponse
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/resources/{resource_type}/{resource_id} [get]
func (h *AuditHandler) GetResourceAuditLogs(c *gin.Context) {
	resourceType := c.Param("resource_type")
	resourceID := c.Param("resource_id")

	if resourceType == "" || resourceID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "资源类型和ID不能为空",
		})
		return
	}

	req := &models.AuditLogQueryRequest{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = ps
		}
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	result, err := h.auditService.QueryAuditLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}

// InitializeAuditConfigs godoc
// @Summary 初始化审计配置
// @Description 初始化默认的审计配置（仅管理员）
// @Tags 审计配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/configs/init [post]
func (h *AuditHandler) InitializeAuditConfigs(c *gin.Context) {
	if err := h.auditService.InitializeDefaultConfigs(); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "初始化配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "配置初始化成功",
	})
}

// CleanupAuditLogs godoc
// @Summary 清理过期审计日志
// @Description 根据配置的保留天数清理过期的审计日志（仅管理员）
// @Tags 审计配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /api/v1/audit/cleanup [post]
func (h *AuditHandler) CleanupAuditLogs(c *gin.Context) {
	if err := h.auditService.CleanupOldLogs(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "清理失败: " + err.Error(),
		})
		return
	}

	// 记录清理操作
	h.auditService.LogAdminOperation(c, models.AuditActionDelete, "audit_logs", "expired", "过期审计日志", nil, nil, nil)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "清理完成",
	})
}
