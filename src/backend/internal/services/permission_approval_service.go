package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// PermissionApprovalService 权限审批服务
type PermissionApprovalService struct {
	db          *gorm.DB
	rbacService *RBACService
}

// NewPermissionApprovalService 创建权限审批服务实例
func NewPermissionApprovalService(db *gorm.DB, rbacService *RBACService) *PermissionApprovalService {
	return &PermissionApprovalService{
		db:          db,
		rbacService: rbacService,
	}
}

// ====================
// 权限模块定义
// ====================

// GetAvailableModules 获取可用的权限模块列表
func (s *PermissionApprovalService) GetAvailableModules() []models.PermissionModuleInfo {
	return []models.PermissionModuleInfo{
		{
			Name:        "saltstack",
			DisplayName: "SaltStack 配置管理",
			Description: "SaltStack 远程执行和配置管理",
			Category:    "infrastructure",
			Icon:        "tool",
			Verbs:       []string{"read", "create", "update", "delete", "execute"},
		},
		{
			Name:        "ansible",
			DisplayName: "Ansible 自动化",
			Description: "Ansible Playbook 执行和管理",
			Category:    "infrastructure",
			Icon:        "code",
			Verbs:       []string{"read", "create", "update", "delete", "execute"},
		},
		{
			Name:        "kubernetes",
			DisplayName: "Kubernetes 集群",
			Description: "Kubernetes 集群管理和操作",
			Category:    "infrastructure",
			Icon:        "cluster",
			Verbs:       []string{"read", "create", "update", "delete", "execute"},
		},
		{
			Name:        "jupyterhub",
			DisplayName: "JupyterHub 机器学习",
			Description: "JupyterHub 笔记本和环境管理",
			Category:    "compute",
			Icon:        "experiment",
			Verbs:       []string{"read", "create", "update", "delete"},
		},
		{
			Name:        "slurm",
			DisplayName: "Slurm HPC 调度",
			Description: "Slurm 作业调度和集群管理",
			Category:    "compute",
			Icon:        "schedule",
			Verbs:       []string{"read", "create", "update", "delete", "submit"},
		},
		{
			Name:        "gitea",
			DisplayName: "Gitea 代码仓库",
			Description: "Git 代码仓库管理",
			Category:    "devops",
			Icon:        "branch",
			Verbs:       []string{"read", "create", "update", "delete"},
		},
		{
			Name:        "kafka-ui",
			DisplayName: "Kafka 消息队列",
			Description: "Kafka 消息队列管理",
			Category:    "data",
			Icon:        "message",
			Verbs:       []string{"read", "create", "update", "delete"},
		},
		{
			Name:        "nightingale",
			DisplayName: "Nightingale 监控",
			Description: "监控和告警系统",
			Category:    "monitoring",
			Icon:        "monitor",
			Verbs:       []string{"read", "create", "update", "delete"},
		},
		{
			Name:        "projects",
			DisplayName: "项目管理",
			Description: "项目创建和管理",
			Category:    "management",
			Icon:        "project",
			Verbs:       []string{"read", "create", "update", "delete", "list"},
		},
		{
			Name:        "hosts",
			DisplayName: "主机管理",
			Description: "主机和节点管理",
			Category:    "infrastructure",
			Icon:        "desktop",
			Verbs:       []string{"read", "create", "update", "delete", "list"},
		},
		{
			Name:        "users",
			DisplayName: "用户管理",
			Description: "用户账号管理",
			Category:    "admin",
			Icon:        "user",
			Verbs:       []string{"read", "create", "update", "delete", "list"},
		},
		{
			Name:        "roles",
			DisplayName: "角色权限",
			Description: "角色和权限配置",
			Category:    "admin",
			Icon:        "lock",
			Verbs:       []string{"read", "create", "update", "delete", "list"},
		},
		{
			Name:        "object-storage",
			DisplayName: "对象存储",
			Description: "SeaweedFS 对象存储管理",
			Category:    "storage",
			Icon:        "database",
			Verbs:       []string{"read", "create", "update", "delete"},
		},
		{
			Name:        "audit-logs",
			DisplayName: "审计日志",
			Description: "系统审计日志查看",
			Category:    "admin",
			Icon:        "file-text",
			Verbs:       []string{"read", "export"},
		},
		{
			Name:        "ai-chat",
			DisplayName: "AI 助手",
			Description: "AI 对话助手功能",
			Category:    "tools",
			Icon:        "robot",
			Verbs:       []string{"read", "use"},
		},
	}
}

// GetAvailableVerbs 获取可用的操作权限列表
func (s *PermissionApprovalService) GetAvailableVerbs() []string {
	return []string{
		"read",    // 读取
		"create",  // 创建
		"update",  // 更新
		"delete",  // 删除
		"list",    // 列表
		"execute", // 执行
		"submit",  // 提交
		"export",  // 导出
		"use",     // 使用
		"*",       // 所有权限
	}
}

// ====================
// 权限申请管理
// ====================

// CreatePermissionRequest 创建权限申请
func (s *PermissionApprovalService) CreatePermissionRequest(requesterID uint, input *models.CreatePermissionRequestInput) (*models.PermissionRequest, error) {
	// 验证申请人是否存在
	var requester models.User
	if err := s.db.First(&requester, requesterID).Error; err != nil {
		return nil, fmt.Errorf("申请人不存在: %v", err)
	}

	// 确定目标用户
	targetUserID := requesterID
	if input.TargetUserID > 0 {
		// 如果为他人申请，需要检查申请人是否有权限
		if !s.rbacService.CheckPermission(requesterID, "users", "update", "*", "") {
			return nil, errors.New("您没有权限为他人申请权限")
		}
		var targetUser models.User
		if err := s.db.First(&targetUser, input.TargetUserID).Error; err != nil {
			return nil, fmt.Errorf("目标用户不存在: %v", err)
		}
		targetUserID = input.TargetUserID
	}

	// 验证申请的模块是否有效
	availableModules := s.GetAvailableModules()
	moduleMap := make(map[string]bool)
	for _, m := range availableModules {
		moduleMap[m.Name] = true
	}
	for _, module := range input.RequestedModules {
		if !moduleMap[module] {
			return nil, fmt.Errorf("无效的模块: %s", module)
		}
	}

	// 验证申请的操作权限是否有效
	availableVerbs := s.GetAvailableVerbs()
	verbMap := make(map[string]bool)
	for _, v := range availableVerbs {
		verbMap[v] = true
	}
	for _, verb := range input.RequestedVerbs {
		if !verbMap[verb] {
			return nil, fmt.Errorf("无效的操作权限: %s", verb)
		}
	}

	// 检查是否有重复的待审批申请
	var existingRequest models.PermissionRequest
	modulesJSON, _ := json.Marshal(input.RequestedModules)
	verbsJSON, _ := json.Marshal(input.RequestedVerbs)
	
	err := s.db.Where(
		"requester_id = ? AND target_user_id = ? AND status = ? AND requested_modules = ? AND requested_verbs = ?",
		requesterID, targetUserID, models.ApprovalStatusPending, modulesJSON, verbsJSON,
	).First(&existingRequest).Error
	
	if err == nil {
		return nil, errors.New("已存在相同的待审批申请，请等待审批结果")
	}

	// 创建权限申请记录
	request := &models.PermissionRequest{
		RequesterID:      requesterID,
		TargetUserID:     targetUserID,
		RoleTemplateName: input.RoleTemplateName,
		Reason:           input.Reason,
		Status:           models.ApprovalStatusPending,
		Priority:         input.Priority,
		ValidDays:        input.ValidDays,
		NotifyEmail:      input.NotifyEmail,
		NotifyDingTalk:   input.NotifyDingTalk,
		RelatedTicket:    input.RelatedTicket,
	}

	if err := request.SetRequestedModules(input.RequestedModules); err != nil {
		return nil, fmt.Errorf("设置申请模块失败: %v", err)
	}
	if err := request.SetRequestedVerbs(input.RequestedVerbs); err != nil {
		return nil, fmt.Errorf("设置操作权限失败: %v", err)
	}

	// 检查是否符合自动审批规则
	autoApprove, rule := s.checkAutoApprovalRule(request)
	if autoApprove && rule != nil {
		request.AutoApprove = true
		// 设置有效期限制
		if rule.MaxValidDays > 0 && (request.ValidDays == 0 || request.ValidDays > rule.MaxValidDays) {
			request.ValidDays = rule.MaxValidDays
		}
	}

	// 保存申请记录
	if err := s.db.Create(request).Error; err != nil {
		return nil, fmt.Errorf("创建申请记录失败: %v", err)
	}

	// 记录审批日志
	s.createApprovalLog(request.ID, requesterID, "submit", "", string(models.ApprovalStatusPending), "提交权限申请", "", "")

	// 如果符合自动审批条件，自动批准
	if autoApprove {
		logrus.WithField("request_id", request.ID).Info("符合自动审批规则，自动批准")
		if err := s.autoApproveRequest(request); err != nil {
			logrus.WithError(err).Warn("自动审批失败，转为人工审批")
		} else {
			// 重新加载请求状态
			s.db.First(request, request.ID)
		}
	}

	// 加载关联数据
	s.db.Preload("Requester").Preload("TargetUser").First(request, request.ID)

	return request, nil
}

// GetPermissionRequest 获取权限申请详情
func (s *PermissionApprovalService) GetPermissionRequest(requestID uint, userID uint) (*models.PermissionRequest, error) {
	var request models.PermissionRequest
	err := s.db.Preload("Requester").Preload("TargetUser").Preload("Approver").First(&request, requestID).Error
	if err != nil {
		return nil, fmt.Errorf("申请记录不存在: %v", err)
	}

	// 检查权限：只有申请人、管理员或审批人可以查看详情
	isAdmin := s.rbacService.IsAdmin(userID)
	isRequester := request.RequesterID == userID
	isApprover := request.ApproverID != nil && *request.ApproverID == userID

	if !isAdmin && !isRequester && !isApprover {
		return nil, errors.New("您没有权限查看此申请")
	}

	return &request, nil
}

// ListPermissionRequests 获取权限申请列表
func (s *PermissionApprovalService) ListPermissionRequests(userID uint, query *models.PermissionRequestQuery) (*models.PermissionRequestListResponse, error) {
	isAdmin := s.rbacService.IsAdmin(userID)

	db := s.db.Model(&models.PermissionRequest{}).Preload("Requester").Preload("TargetUser").Preload("Approver")

	// 非管理员只能查看自己的申请或待审批的申请（如果是审批人）
	if !isAdmin {
		db = db.Where("requester_id = ? OR target_user_id = ?", userID, userID)
	}

	// 状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OnlyPending {
		db = db.Where("status = ?", models.ApprovalStatusPending)
	}

	// 申请人筛选
	if query.RequesterID > 0 {
		db = db.Where("requester_id = ?", query.RequesterID)
	}

	// 审批人筛选
	if query.ApproverID > 0 {
		db = db.Where("approver_id = ?", query.ApproverID)
	}

	// 模块筛选
	if query.Module != "" {
		db = db.Where("requested_modules @> ?", fmt.Sprintf(`["%s"]`, query.Module))
	}

	// 优先级筛选
	if query.Priority > 0 {
		db = db.Where("priority = ?", query.Priority)
	}

	// 日期筛选
	if query.StartDate != "" {
		db = db.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		db = db.Where("created_at <= ?", query.EndDate+" 23:59:59")
	}

	// 统计总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询总数失败: %v", err)
	}

	// 排序
	sortBy := "created_at"
	sortOrder := "desc"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	if query.SortOrder != "" {
		sortOrder = query.SortOrder
	}
	db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// 分页
	page := 1
	pageSize := 20
	if query.Page > 0 {
		page = query.Page
	}
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	offset := (page - 1) * pageSize
	db = db.Offset(offset).Limit(pageSize)

	// 查询数据
	var requests []models.PermissionRequest
	if err := db.Find(&requests).Error; err != nil {
		return nil, fmt.Errorf("查询申请列表失败: %v", err)
	}

	return &models.PermissionRequestListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    requests,
	}, nil
}

// ApprovePermissionRequest 审批权限申请
func (s *PermissionApprovalService) ApprovePermissionRequest(requestID uint, approverID uint, input *models.ApprovePermissionRequestInput, ipAddress, userAgent string) error {
	// 检查审批人是否是管理员
	if !s.rbacService.IsAdmin(approverID) {
		return errors.New("只有管理员才能审批权限申请")
	}

	// 获取申请记录
	var request models.PermissionRequest
	if err := s.db.First(&request, requestID).Error; err != nil {
		return fmt.Errorf("申请记录不存在: %v", err)
	}

	// 检查是否可以审批
	if !request.CanBeApproved() {
		return fmt.Errorf("该申请状态为 %s，无法审批", request.Status)
	}

	// 不能审批自己的申请
	if request.RequesterID == approverID {
		return errors.New("不能审批自己的申请")
	}

	oldStatus := string(request.Status)
	var newStatus models.PermissionApprovalStatus
	var action string

	if input.Approved {
		newStatus = models.ApprovalStatusApproved
		action = "approve"
		
		// 批准时授予权限
		if err := s.grantPermissions(&request, approverID); err != nil {
			return fmt.Errorf("授予权限失败: %v", err)
		}
	} else {
		newStatus = models.ApprovalStatusRejected
		action = "reject"
	}

	// 更新申请状态
	now := time.Now()
	updates := map[string]interface{}{
		"status":          newStatus,
		"approver_id":     approverID,
		"approve_comment": input.Comment,
		"approved_at":     now,
	}

	if err := s.db.Model(&request).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新申请状态失败: %v", err)
	}

	// 记录审批日志
	s.createApprovalLog(requestID, approverID, action, oldStatus, string(newStatus), input.Comment, ipAddress, userAgent)

	// TODO: 发送通知（邮件/钉钉）
	if request.NotifyEmail {
		// s.sendEmailNotification(&request, input.Approved, input.Comment)
	}
	if request.NotifyDingTalk {
		// s.sendDingTalkNotification(&request, input.Approved, input.Comment)
	}

	logrus.WithFields(logrus.Fields{
		"request_id":  requestID,
		"approver_id": approverID,
		"approved":    input.Approved,
	}).Info("权限申请已审批")

	return nil
}

// CancelPermissionRequest 取消权限申请
func (s *PermissionApprovalService) CancelPermissionRequest(requestID uint, userID uint, reason string) error {
	var request models.PermissionRequest
	if err := s.db.First(&request, requestID).Error; err != nil {
		return fmt.Errorf("申请记录不存在: %v", err)
	}

	// 只有申请人可以取消自己的申请
	if request.RequesterID != userID {
		return errors.New("只能取消自己的申请")
	}

	// 只能取消待审批的申请
	if request.Status != models.ApprovalStatusPending {
		return errors.New("只能取消待审批状态的申请")
	}

	oldStatus := string(request.Status)
	if err := s.db.Model(&request).Update("status", models.ApprovalStatusCanceled).Error; err != nil {
		return fmt.Errorf("取消申请失败: %v", err)
	}

	s.createApprovalLog(requestID, userID, "cancel", oldStatus, string(models.ApprovalStatusCanceled), reason, "", "")

	return nil
}

// ====================
// 权限授予管理
// ====================

// grantPermissions 授予权限（内部方法）
func (s *PermissionApprovalService) grantPermissions(request *models.PermissionRequest, grantedBy uint) error {
	modules := request.GetRequestedModules()
	verbs := request.GetRequestedVerbs()

	var expiresAt *time.Time
	if request.ValidDays > 0 {
		t := time.Now().AddDate(0, 0, request.ValidDays)
		expiresAt = &t
	}

	// 为每个模块和操作创建授权记录
	for _, module := range modules {
		for _, verb := range verbs {
			grant := &models.PermissionGrant{
				UserID:              request.TargetUserID,
				PermissionRequestID: &request.ID,
				Module:              module,
				Resource:            module,
				Verb:                verb,
				Scope:               "*",
				GrantType:           "approval",
				GrantedBy:           grantedBy,
				Reason:              request.Reason,
				ExpiresAt:           expiresAt,
				IsActive:            true,
			}

			// 检查是否已存在相同的授权
			var existing models.PermissionGrant
			err := s.db.Where(
				"user_id = ? AND module = ? AND verb = ? AND is_active = ?",
				grant.UserID, grant.Module, grant.Verb, true,
			).First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				// 创建新授权
				if err := s.db.Create(grant).Error; err != nil {
					return fmt.Errorf("创建授权记录失败: %v", err)
				}
			} else if err == nil {
				// 更新现有授权的过期时间
				if expiresAt != nil && (existing.ExpiresAt == nil || expiresAt.After(*existing.ExpiresAt)) {
					s.db.Model(&existing).Update("expires_at", expiresAt)
				}
			}
		}
	}

	// 同步到 RBAC 系统
	if err := s.syncPermissionsToRBAC(request.TargetUserID, modules, verbs); err != nil {
		logrus.WithError(err).Warn("同步权限到RBAC系统失败")
	}

	return nil
}

// GrantPermission 手动授予权限
func (s *PermissionApprovalService) GrantPermission(grantedBy uint, input *models.GrantPermissionInput) error {
	// 检查授权人是否是管理员
	if !s.rbacService.IsAdmin(grantedBy) {
		return errors.New("只有管理员才能手动授权")
	}

	// 验证目标用户
	var targetUser models.User
	if err := s.db.First(&targetUser, input.UserID).Error; err != nil {
		return fmt.Errorf("目标用户不存在: %v", err)
	}

	var expiresAt *time.Time
	if input.ValidDays > 0 {
		t := time.Now().AddDate(0, 0, input.ValidDays)
		expiresAt = &t
	}

	// 为每个模块和操作创建授权记录
	for _, module := range input.Modules {
		for _, verb := range input.Verbs {
			grant := &models.PermissionGrant{
				UserID:    input.UserID,
				Module:    module,
				Resource:  module,
				Verb:      verb,
				Scope:     "*",
				GrantType: "manual",
				GrantedBy: grantedBy,
				Reason:    input.Reason,
				ExpiresAt: expiresAt,
				IsActive:  true,
			}

			// 检查是否已存在
			var existing models.PermissionGrant
			err := s.db.Where(
				"user_id = ? AND module = ? AND verb = ? AND is_active = ?",
				grant.UserID, grant.Module, grant.Verb, true,
			).First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				if err := s.db.Create(grant).Error; err != nil {
					return fmt.Errorf("创建授权记录失败: %v", err)
				}
			} else if err == nil {
				// 更新现有授权
				updates := map[string]interface{}{
					"expires_at": expiresAt,
					"granted_by": grantedBy,
					"reason":     input.Reason,
				}
				s.db.Model(&existing).Updates(updates)
			}
		}
	}

	// 同步到 RBAC 系统
	if err := s.syncPermissionsToRBAC(input.UserID, input.Modules, input.Verbs); err != nil {
		logrus.WithError(err).Warn("同步权限到RBAC系统失败")
	}

	logrus.WithFields(logrus.Fields{
		"granted_by": grantedBy,
		"user_id":    input.UserID,
		"modules":    input.Modules,
		"verbs":      input.Verbs,
	}).Info("手动授予权限")

	return nil
}

// RevokePermission 撤销权限
func (s *PermissionApprovalService) RevokePermission(revokedBy uint, input *models.RevokePermissionInput) error {
	// 检查撤销人是否是管理员
	if !s.rbacService.IsAdmin(revokedBy) {
		return errors.New("只有管理员才能撤销权限")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_active":     false,
		"revoked_at":    now,
		"revoked_by":    revokedBy,
		"revoke_reason": input.Reason,
	}

	result := s.db.Model(&models.PermissionGrant{}).
		Where("user_id = ? AND module = ? AND is_active = ?", input.UserID, input.Module, true).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("撤销权限失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("未找到可撤销的权限记录")
	}

	logrus.WithFields(logrus.Fields{
		"revoked_by": revokedBy,
		"user_id":    input.UserID,
		"module":     input.Module,
	}).Info("撤销权限")

	return nil
}

// GetUserGrants 获取用户的权限授权列表
func (s *PermissionApprovalService) GetUserGrants(userID uint, onlyActive bool) ([]models.PermissionGrant, error) {
	var grants []models.PermissionGrant
	db := s.db.Where("user_id = ?", userID).Preload("Grantor")
	
	if onlyActive {
		db = db.Where("is_active = ?", true)
	}
	
	if err := db.Order("module, verb").Find(&grants).Error; err != nil {
		return nil, fmt.Errorf("查询授权记录失败: %v", err)
	}

	return grants, nil
}

// ====================
// 审批规则管理
// ====================

// CreateApprovalRule 创建审批规则
func (s *PermissionApprovalService) CreateApprovalRule(createdBy uint, input *models.CreateApprovalRuleInput) (*models.PermissionApprovalRule, error) {
	if !s.rbacService.IsAdmin(createdBy) {
		return nil, errors.New("只有管理员才能创建审批规则")
	}

	conditionValue, _ := json.Marshal(input.ConditionValue)
	requiredApprovers, _ := json.Marshal(input.RequiredApprovers)
	allowedModules, _ := json.Marshal(input.AllowedModules)

	rule := &models.PermissionApprovalRule{
		Name:              input.Name,
		Description:       input.Description,
		IsActive:          input.IsActive,
		Priority:          input.Priority,
		ConditionType:     input.ConditionType,
		ConditionValue:    conditionValue,
		AutoApprove:       input.AutoApprove,
		RequiredApprovers: requiredApprovers,
		MinApprovals:      input.MinApprovals,
		MaxValidDays:      input.MaxValidDays,
		AllowedModules:    allowedModules,
		NotifyAdmins:      input.NotifyAdmins,
		CreatedBy:         createdBy,
	}

	if err := s.db.Create(rule).Error; err != nil {
		return nil, fmt.Errorf("创建审批规则失败: %v", err)
	}

	return rule, nil
}

// GetApprovalRules 获取审批规则列表
func (s *PermissionApprovalService) GetApprovalRules(onlyActive bool) ([]models.PermissionApprovalRule, error) {
	var rules []models.PermissionApprovalRule
	db := s.db.Preload("Creator").Order("priority DESC, created_at ASC")
	
	if onlyActive {
		db = db.Where("is_active = ?", true)
	}
	
	if err := db.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("查询审批规则失败: %v", err)
	}

	return rules, nil
}

// UpdateApprovalRule 更新审批规则
func (s *PermissionApprovalService) UpdateApprovalRule(ruleID uint, userID uint, updates map[string]interface{}) error {
	if !s.rbacService.IsAdmin(userID) {
		return errors.New("只有管理员才能更新审批规则")
	}

	if err := s.db.Model(&models.PermissionApprovalRule{}).Where("id = ?", ruleID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新审批规则失败: %v", err)
	}

	return nil
}

// DeleteApprovalRule 删除审批规则
func (s *PermissionApprovalService) DeleteApprovalRule(ruleID uint, userID uint) error {
	if !s.rbacService.IsAdmin(userID) {
		return errors.New("只有管理员才能删除审批规则")
	}

	if err := s.db.Delete(&models.PermissionApprovalRule{}, ruleID).Error; err != nil {
		return fmt.Errorf("删除审批规则失败: %v", err)
	}

	return nil
}

// checkAutoApprovalRule 检查是否符合自动审批规则
func (s *PermissionApprovalService) checkAutoApprovalRule(request *models.PermissionRequest) (bool, *models.PermissionApprovalRule) {
	var rules []models.PermissionApprovalRule
	if err := s.db.Where("is_active = ? AND auto_approve = ?", true, true).
		Order("priority DESC").Find(&rules).Error; err != nil {
		return false, nil
	}

	for _, rule := range rules {
		if s.matchApprovalRule(&rule, request) {
			return true, &rule
		}
	}

	return false, nil
}

// matchApprovalRule 检查申请是否匹配规则
func (s *PermissionApprovalService) matchApprovalRule(rule *models.PermissionApprovalRule, request *models.PermissionRequest) bool {
	conditionValues := rule.GetConditionValue()
	requestedModules := request.GetRequestedModules()

	switch rule.ConditionType {
	case "role_template":
		// 检查角色模板是否匹配
		for _, v := range conditionValues {
			if v == request.RoleTemplateName {
				return true
			}
		}
	case "module":
		// 检查申请的模块是否在允许的模块列表中
		allowedModules := rule.GetAllowedModules()
		for _, module := range requestedModules {
			allowed := false
			for _, am := range allowedModules {
				if am == module || am == "*" {
					allowed = true
					break
				}
			}
			if !allowed {
				return false
			}
		}
		return true
	case "user_group":
		// 检查申请人是否在指定用户组中
		var memberships []models.UserGroupMembership
		s.db.Where("user_id = ?", request.RequesterID).Find(&memberships)
		for _, m := range memberships {
			var group models.UserGroup
			if s.db.First(&group, m.UserGroupID).Error == nil {
				for _, v := range conditionValues {
					if v == group.Name {
						return true
					}
				}
			}
		}
	}

	return false
}

// autoApproveRequest 自动审批申请
func (s *PermissionApprovalService) autoApproveRequest(request *models.PermissionRequest) error {
	// 使用系统账户作为审批人
	systemApproverID := uint(1) // 假设系统管理员ID为1

	now := time.Now()
	updates := map[string]interface{}{
		"status":          models.ApprovalStatusApproved,
		"approver_id":     systemApproverID,
		"approve_comment": "系统自动审批",
		"approved_at":     now,
	}

	if err := s.db.Model(request).Updates(updates).Error; err != nil {
		return err
	}

	// 授予权限
	if err := s.grantPermissions(request, systemApproverID); err != nil {
		return err
	}

	s.createApprovalLog(request.ID, systemApproverID, "auto_approve", string(models.ApprovalStatusPending), string(models.ApprovalStatusApproved), "符合自动审批规则，系统自动批准", "", "")

	return nil
}

// ====================
// 辅助方法
// ====================

// createApprovalLog 创建审批日志
func (s *PermissionApprovalService) createApprovalLog(requestID uint, operatorID uint, action, oldStatus, newStatus, comment, ipAddress, userAgent string) {
	log := &models.PermissionApprovalLog{
		PermissionRequestID: requestID,
		OperatorID:          operatorID,
		Action:              action,
		OldStatus:           oldStatus,
		NewStatus:           newStatus,
		Comment:             comment,
		IPAddress:           ipAddress,
		UserAgent:           userAgent,
	}

	if err := s.db.Create(log).Error; err != nil {
		logrus.WithError(err).Error("创建审批日志失败")
	}
}

// GetApprovalLogs 获取审批日志
func (s *PermissionApprovalService) GetApprovalLogs(requestID uint) ([]models.PermissionApprovalLog, error) {
	var logs []models.PermissionApprovalLog
	if err := s.db.Where("permission_request_id = ?", requestID).
		Preload("Operator").
		Order("created_at ASC").
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("查询审批日志失败: %v", err)
	}

	return logs, nil
}

// syncPermissionsToRBAC 同步权限到 RBAC 系统
func (s *PermissionApprovalService) syncPermissionsToRBAC(userID uint, modules []string, verbs []string) error {
	// 为每个模块和操作创建对应的 Permission 并关联到用户角色
	for _, module := range modules {
		for _, verb := range verbs {
			// 查找或创建权限
			var permission models.Permission
			err := s.db.Where("resource = ? AND verb = ?", module, verb).First(&permission).Error
			if err == gorm.ErrRecordNotFound {
				permission = models.Permission{
					Resource:    module,
					Verb:        verb,
					Scope:       "*",
					Description: fmt.Sprintf("%s %s 权限", module, verb),
				}
				if err := s.db.Create(&permission).Error; err != nil {
					continue
				}
			}

			// 获取或创建用户的个人角色
			var userRole models.Role
			roleName := fmt.Sprintf("user_%d_custom", userID)
			err = s.db.Where("name = ?", roleName).First(&userRole).Error
			if err == gorm.ErrRecordNotFound {
				userRole = models.Role{
					Name:        roleName,
					Description: fmt.Sprintf("用户 %d 的自定义权限角色", userID),
					IsSystem:    false,
				}
				if err := s.db.Create(&userRole).Error; err != nil {
					continue
				}

				// 关联用户和角色
				userRoleAssoc := models.UserRole{
					UserID: userID,
					RoleID: userRole.ID,
				}
				s.db.Create(&userRoleAssoc)
			}

			// 关联角色和权限
			var existing models.RolePermission
			err = s.db.Where("role_id = ? AND permission_id = ?", userRole.ID, permission.ID).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				rolePermission := models.RolePermission{
					RoleID:       userRole.ID,
					PermissionID: permission.ID,
				}
				s.db.Create(&rolePermission)
			}
		}
	}

	return nil
}

// GetStats 获取权限审批统计信息
func (s *PermissionApprovalService) GetStats() (*models.PermissionStats, error) {
	stats := &models.PermissionStats{}

	// 申请统计
	s.db.Model(&models.PermissionRequest{}).Count(&stats.TotalRequests)
	s.db.Model(&models.PermissionRequest{}).Where("status = ?", models.ApprovalStatusPending).Count(&stats.PendingRequests)
	s.db.Model(&models.PermissionRequest{}).Where("status = ?", models.ApprovalStatusApproved).Count(&stats.ApprovedRequests)
	s.db.Model(&models.PermissionRequest{}).Where("status = ?", models.ApprovalStatusRejected).Count(&stats.RejectedRequests)

	// 授权统计
	s.db.Model(&models.PermissionGrant{}).Count(&stats.TotalGrants)
	s.db.Model(&models.PermissionGrant{}).Where("is_active = ?", true).Count(&stats.ActiveGrants)
	s.db.Model(&models.PermissionGrant{}).Where("is_active = ? AND expires_at < ?", true, time.Now()).Count(&stats.ExpiredGrants)

	return stats, nil
}

// ExpireGrants 过期处理（定时任务调用）
func (s *PermissionApprovalService) ExpireGrants() (int64, error) {
	result := s.db.Model(&models.PermissionGrant{}).
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at < ?", true, time.Now()).
		Update("is_active", false)

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		logrus.WithField("count", result.RowsAffected).Info("已过期的权限授权已自动停用")
	}

	return result.RowsAffected, nil
}

// CheckUserModulePermission 检查用户是否有模块权限
func (s *PermissionApprovalService) CheckUserModulePermission(userID uint, module string, verb string) bool {
	// 首先检查 RBAC 权限
	if s.rbacService.CheckPermission(userID, module, verb, "*", "") {
		return true
	}

	// 然后检查授权记录
	var grant models.PermissionGrant
	err := s.db.Where(
		"user_id = ? AND module = ? AND (verb = ? OR verb = '*') AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)",
		userID, module, verb, true, time.Now(),
	).First(&grant).Error

	return err == nil
}
