package controllers

import (
	"net/http"
	"strconv"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PermissionApprovalController 权限审批控制器
type PermissionApprovalController struct {
	approvalService *services.PermissionApprovalService
}

// NewPermissionApprovalController 创建权限审批控制器实例
func NewPermissionApprovalController(approvalService *services.PermissionApprovalService) *PermissionApprovalController {
	return &PermissionApprovalController{
		approvalService: approvalService,
	}
}

// ====================
// 模块和权限信息
// ====================

// GetAvailableModules 获取可用模块列表
// @Summary 获取可用模块列表
// @Description 获取系统中所有可申请的权限模块
// @Tags 权限审批
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/modules [get]
func (c *PermissionApprovalController) GetAvailableModules(ctx *gin.Context) {
	modules := c.approvalService.GetAvailableModules()
	
	// 按类别分组
	grouped := make(map[string][]models.PermissionModuleInfo)
	for _, m := range modules {
		grouped[m.Category] = append(grouped[m.Category], m)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"modules": modules,
			"grouped": grouped,
		},
	})
}

// GetAvailableVerbs 获取可用操作权限列表
// @Summary 获取可用操作权限列表
// @Description 获取系统中所有可申请的操作权限
// @Tags 权限审批
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/verbs [get]
func (c *PermissionApprovalController) GetAvailableVerbs(ctx *gin.Context) {
	verbs := c.approvalService.GetAvailableVerbs()
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    verbs,
	})
}

// ====================
// 权限申请管理
// ====================

// CreatePermissionRequest 创建权限申请
// @Summary 创建权限申请
// @Description 用户提交权限申请，等待管理员审批
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param request body models.CreatePermissionRequestInput true "申请信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/requests [post]
func (c *PermissionApprovalController) CreatePermissionRequest(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	if userID == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var input models.CreatePermissionRequestInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证必填字段
	if len(input.RequestedModules) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请至少选择一个模块",
		})
		return
	}
	if len(input.RequestedVerbs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请至少选择一个操作权限",
		})
		return
	}
	if input.Reason == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请填写申请原因",
		})
		return
	}

	request, err := c.approvalService.CreatePermissionRequest(userID, &input)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	logrus.WithFields(logrus.Fields{
		"request_id": request.ID,
		"user_id":    userID,
		"modules":    input.RequestedModules,
	}).Info("权限申请已创建")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "权限申请已提交",
		"data":    request,
	})
}

// GetPermissionRequest 获取权限申请详情
// @Summary 获取权限申请详情
// @Description 获取单个权限申请的详细信息
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param id path int true "申请ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/approvals/requests/{id} [get]
func (c *PermissionApprovalController) GetPermissionRequest(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	requestID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的申请ID",
		})
		return
	}

	request, err := c.approvalService.GetPermissionRequest(uint(requestID), userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 获取审批日志
	logs, _ := c.approvalService.GetApprovalLogs(uint(requestID))

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"request": request,
			"logs":    logs,
		},
	})
}

// ListPermissionRequests 获取权限申请列表
// @Summary 获取权限申请列表
// @Description 获取权限申请列表，支持分页和筛选
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param status query string false "状态筛选"
// @Param only_pending query bool false "仅显示待审批"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/requests [get]
func (c *PermissionApprovalController) ListPermissionRequests(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	query := &models.PermissionRequestQuery{}
	
	// 解析查询参数
	query.Status = ctx.Query("status")
	query.OnlyPending = ctx.Query("only_pending") == "true"
	query.Module = ctx.Query("module")
	query.StartDate = ctx.Query("start_date")
	query.EndDate = ctx.Query("end_date")
	query.SortBy = ctx.Query("sort_by")
	query.SortOrder = ctx.Query("sort_order")

	if page, err := strconv.Atoi(ctx.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(ctx.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}
	if requesterID, err := strconv.ParseUint(ctx.Query("requester_id"), 10, 32); err == nil {
		query.RequesterID = uint(requesterID)
	}
	if approverID, err := strconv.ParseUint(ctx.Query("approver_id"), 10, 32); err == nil {
		query.ApproverID = uint(approverID)
	}
	if priority, err := strconv.Atoi(ctx.Query("priority")); err == nil {
		query.Priority = priority
	}

	result, err := c.approvalService.ListPermissionRequests(userID, query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ApprovePermissionRequest 审批权限申请
// @Summary 审批权限申请
// @Description 管理员审批权限申请（批准或拒绝）
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param id path int true "申请ID"
// @Param request body models.ApprovePermissionRequestInput true "审批信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/requests/{id}/approve [post]
func (c *PermissionApprovalController) ApprovePermissionRequest(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	requestID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的申请ID",
		})
		return
	}

	var input models.ApprovePermissionRequestInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	if err := c.approvalService.ApprovePermissionRequest(uint(requestID), userID, &input, ipAddress, userAgent); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	action := "拒绝"
	if input.Approved {
		action = "批准"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "申请已" + action,
	})
}

// CancelPermissionRequest 取消权限申请
// @Summary 取消权限申请
// @Description 申请人取消自己的待审批申请
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param id path int true "申请ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/requests/{id}/cancel [post]
func (c *PermissionApprovalController) CancelPermissionRequest(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	requestID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的申请ID",
		})
		return
	}

	var input struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&input)

	if err := c.approvalService.CancelPermissionRequest(uint(requestID), userID, input.Reason); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "申请已取消",
	})
}

// ====================
// 权限授权管理
// ====================

// GrantPermission 手动授予权限
// @Summary 手动授予权限
// @Description 管理员直接授予用户权限
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param request body models.GrantPermissionInput true "授权信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/grants [post]
func (c *PermissionApprovalController) GrantPermission(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var input models.GrantPermissionInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := c.approvalService.GrantPermission(userID, &input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "权限已授予",
	})
}

// RevokePermission 撤销权限
// @Summary 撤销权限
// @Description 管理员撤销用户的权限
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param request body models.RevokePermissionInput true "撤销信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/grants/revoke [post]
func (c *PermissionApprovalController) RevokePermission(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var input models.RevokePermissionInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := c.approvalService.RevokePermission(userID, &input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "权限已撤销",
	})
}

// GetUserGrants 获取用户权限授权列表
// @Summary 获取用户权限授权列表
// @Description 获取指定用户的所有权限授权记录
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param user_id query int true "用户ID"
// @Param only_active query bool false "仅显示有效授权"
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/grants [get]
func (c *PermissionApprovalController) GetUserGrants(ctx *gin.Context) {
	userIDParam, err := strconv.ParseUint(ctx.Query("user_id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	onlyActive := ctx.Query("only_active") != "false"

	grants, err := c.approvalService.GetUserGrants(uint(userIDParam), onlyActive)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    grants,
	})
}

// GetMyGrants 获取当前用户的权限授权
// @Summary 获取当前用户的权限授权
// @Description 获取当前登录用户的所有权限授权记录
// @Tags 权限审批
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/my-grants [get]
func (c *PermissionApprovalController) GetMyGrants(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	grants, err := c.approvalService.GetUserGrants(userID, true)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 按模块分组
	grouped := make(map[string][]models.PermissionGrant)
	for _, g := range grants {
		grouped[g.Module] = append(grouped[g.Module], g)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"grants":  grants,
			"grouped": grouped,
		},
	})
}

// ====================
// 审批规则管理
// ====================

// CreateApprovalRule 创建审批规则
// @Summary 创建审批规则
// @Description 管理员创建自动审批规则
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param request body models.CreateApprovalRuleInput true "规则信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/rules [post]
func (c *PermissionApprovalController) CreateApprovalRule(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var input models.CreateApprovalRuleInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	rule, err := c.approvalService.CreateApprovalRule(userID, &input)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "审批规则已创建",
		"data":    rule,
	})
}

// GetApprovalRules 获取审批规则列表
// @Summary 获取审批规则列表
// @Description 获取所有审批规则
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param only_active query bool false "仅显示启用的规则"
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/rules [get]
func (c *PermissionApprovalController) GetApprovalRules(ctx *gin.Context) {
	onlyActive := ctx.Query("only_active") == "true"

	rules, err := c.approvalService.GetApprovalRules(onlyActive)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rules,
	})
}

// UpdateApprovalRule 更新审批规则
// @Summary 更新审批规则
// @Description 管理员更新审批规则
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param request body map[string]interface{} true "更新字段"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/rules/{id} [put]
func (c *PermissionApprovalController) UpdateApprovalRule(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	ruleID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的规则ID",
		})
		return
	}

	var updates map[string]interface{}
	if err := ctx.ShouldBindJSON(&updates); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := c.approvalService.UpdateApprovalRule(uint(ruleID), userID, updates); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "审批规则已更新",
	})
}

// DeleteApprovalRule 删除审批规则
// @Summary 删除审批规则
// @Description 管理员删除审批规则
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/approvals/rules/{id} [delete]
func (c *PermissionApprovalController) DeleteApprovalRule(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	ruleID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的规则ID",
		})
		return
	}

	if err := c.approvalService.DeleteApprovalRule(uint(ruleID), userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "审批规则已删除",
	})
}

// ====================
// 统计和其他
// ====================

// GetStats 获取权限审批统计
// @Summary 获取权限审批统计
// @Description 获取权限审批相关统计数据
// @Tags 权限审批
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/stats [get]
func (c *PermissionApprovalController) GetStats(ctx *gin.Context) {
	stats, err := c.approvalService.GetStats()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// CheckModulePermission 检查用户模块权限
// @Summary 检查用户模块权限
// @Description 检查用户是否有指定模块的权限
// @Tags 权限审批
// @Accept json
// @Produce json
// @Param module query string true "模块名"
// @Param verb query string true "操作权限"
// @Success 200 {object} map[string]interface{}
// @Router /api/approvals/check [get]
func (c *PermissionApprovalController) CheckModulePermission(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	module := ctx.Query("module")
	verb := ctx.Query("verb")

	if module == "" || verb == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请提供模块名和操作权限",
		})
		return
	}

	hasPermission := c.approvalService.CheckUserModulePermission(userID, module, verb)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"module":         module,
			"verb":           verb,
			"has_permission": hasPermission,
		},
	})
}
