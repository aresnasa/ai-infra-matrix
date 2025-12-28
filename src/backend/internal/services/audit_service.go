package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// AuditService 审计服务
type AuditService struct {
	db              *gorm.DB
	hostname        string
	environment     string
	asyncBuffer     chan *models.InfraAuditLog
	configs         map[models.AuditCategory]*models.AuditConfig
	configMutex     sync.RWMutex
	sensitiveFields []string
}

// auditServiceInstance 单例
var (
	auditServiceInstance *AuditService
	auditServiceOnce     sync.Once
)

// NewAuditService 创建审计服务
func NewAuditService() *AuditService {
	auditServiceOnce.Do(func() {
		hostname, _ := os.Hostname()
		env := os.Getenv("APP_ENV")
		if env == "" {
			env = "development"
		}

		auditServiceInstance = &AuditService{
			db:          database.GetDB(),
			hostname:    hostname,
			environment: env,
			asyncBuffer: make(chan *models.InfraAuditLog, 1000), // 异步缓冲区
			configs:     make(map[models.AuditCategory]*models.AuditConfig),
			sensitiveFields: []string{
				"password", "secret", "token", "key", "credential",
				"api_key", "access_token", "refresh_token", "private_key",
				"kube_config", "ssh_key", "passphrase",
			},
		}

		// 加载配置
		auditServiceInstance.loadConfigs()

		// 启动异步写入协程
		go auditServiceInstance.asyncWriter()
	})
	return auditServiceInstance
}

// GetAuditService 获取审计服务实例
func GetAuditService() *AuditService {
	if auditServiceInstance == nil {
		return NewAuditService()
	}
	return auditServiceInstance
}

// loadConfigs 加载审计配置
func (s *AuditService) loadConfigs() {
	var configs []models.AuditConfig
	if err := s.db.Find(&configs).Error; err != nil {
		logrus.WithError(err).Warn("Failed to load audit configs, using defaults")
		// 使用默认配置
		for _, cfg := range models.GetDefaultAuditConfigs() {
			s.configs[cfg.Category] = &cfg
		}
		return
	}

	s.configMutex.Lock()
	defer s.configMutex.Unlock()

	for i := range configs {
		s.configs[configs[i].Category] = &configs[i]
	}
}

// ReloadConfigs 重新加载配置
func (s *AuditService) ReloadConfigs() {
	s.loadConfigs()
}

// asyncWriter 异步写入协程
func (s *AuditService) asyncWriter() {
	batch := make([]*models.InfraAuditLog, 0, 100)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log := <-s.asyncBuffer:
			batch = append(batch, log)
			// 批量写入
			if len(batch) >= 100 {
				s.writeBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			// 定时写入
			if len(batch) > 0 {
				s.writeBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// writeBatch 批量写入
func (s *AuditService) writeBatch(logs []*models.InfraAuditLog) {
	if len(logs) == 0 {
		return
	}

	if err := s.db.Create(&logs).Error; err != nil {
		logrus.WithError(err).Error("Failed to write audit logs batch")
		// 单独写入失败的记录
		for _, log := range logs {
			if err := s.db.Create(log).Error; err != nil {
				logrus.WithError(err).WithField("trace_id", log.TraceID).Error("Failed to write audit log")
			}
		}
	}
}

// ==================== 核心记录方法 ====================

// AuditEntry 审计条目构建器
type AuditEntry struct {
	service       *AuditService
	log           *models.InfraAuditLog
	changeDetails []models.AuditChangeDetail
}

// NewAuditEntry 创建新的审计条目
func (s *AuditService) NewAuditEntry(category models.AuditCategory, action models.AuditAction) *AuditEntry {
	return &AuditEntry{
		service: s,
		log: &models.InfraAuditLog{
			TraceID:     uuid.New().String(),
			Category:    category,
			Action:      action,
			Status:      models.AuditStatusSuccess,
			Severity:    models.AuditSeverityInfo,
			Environment: s.environment,
			HostName:    s.hostname,
		},
		changeDetails: make([]models.AuditChangeDetail, 0),
	}
}

// WithUser 设置用户信息
func (e *AuditEntry) WithUser(userID uint, username, role string) *AuditEntry {
	e.log.UserID = userID
	e.log.Username = username
	e.log.UserRole = role
	return e
}

// WithUserFromContext 从 Gin Context 中获取用户信息
func (e *AuditEntry) WithUserFromContext(c *gin.Context) *AuditEntry {
	if userID, exists := c.Get("user_id"); exists {
		e.log.UserID = userID.(uint)
	}
	if username, exists := c.Get("username"); exists {
		e.log.Username = username.(string)
	}
	if role, exists := c.Get("user_role"); exists {
		e.log.UserRole = role.(string)
	}
	return e
}

// WithResource 设置资源信息
func (e *AuditEntry) WithResource(resourceType, resourceID, resourceName string) *AuditEntry {
	e.log.ResourceType = resourceType
	e.log.ResourceID = resourceID
	e.log.ResourceName = resourceName
	return e
}

// WithRequest 设置请求信息
func (e *AuditEntry) WithRequest(method, path string, params interface{}) *AuditEntry {
	e.log.RequestMethod = method
	e.log.RequestPath = path
	if params != nil {
		e.log.RequestParams = e.service.maskSensitiveData(models.ToJSON(params))
	}
	return e
}

// WithRequestFromContext 从 Gin Context 中获取请求信息
func (e *AuditEntry) WithRequestFromContext(c *gin.Context) *AuditEntry {
	e.log.RequestMethod = c.Request.Method
	e.log.RequestPath = c.Request.URL.Path
	e.log.ClientIP = c.ClientIP()
	e.log.UserAgent = c.Request.UserAgent()
	if sessionID, exists := c.Get("session_id"); exists {
		e.log.SessionID = sessionID.(string)
	}
	return e
}

// WithClient 设置客户端信息
func (e *AuditEntry) WithClient(ip, userAgent, sessionID string) *AuditEntry {
	e.log.ClientIP = ip
	e.log.UserAgent = userAgent
	e.log.SessionID = sessionID
	return e
}

// WithChange 设置变更信息
func (e *AuditEntry) WithChange(oldValue, newValue interface{}, summary string) *AuditEntry {
	e.log.OldValue = e.service.maskSensitiveData(models.ToJSON(oldValue))
	e.log.NewValue = e.service.maskSensitiveData(models.ToJSON(newValue))
	e.log.ChangeSummary = summary

	// 自动计算详细变更
	if oldValue != nil && newValue != nil {
		e.changeDetails = e.service.computeChanges(oldValue, newValue)
	}
	return e
}

// WithStatus 设置状态
func (e *AuditEntry) WithStatus(status models.AuditStatus) *AuditEntry {
	e.log.Status = status
	return e
}

// WithSeverity 设置严重程度
func (e *AuditEntry) WithSeverity(severity models.AuditSeverity) *AuditEntry {
	e.log.Severity = severity
	return e
}

// WithError 设置错误信息
func (e *AuditEntry) WithError(err error) *AuditEntry {
	if err != nil {
		e.log.Status = models.AuditStatusFailed
		e.log.ErrorMessage = err.Error()
	}
	return e
}

// WithErrorAndStack 设置错误信息和堆栈
func (e *AuditEntry) WithErrorAndStack(err error, stack string) *AuditEntry {
	if err != nil {
		e.log.Status = models.AuditStatusFailed
		e.log.ErrorMessage = err.Error()
		e.log.StackTrace = stack
	}
	return e
}

// WithExecutionTime 设置执行时间
func (e *AuditEntry) WithExecutionTime(duration time.Duration) *AuditEntry {
	e.log.ExecutionTime = duration.Milliseconds()
	return e
}

// WithMetadata 设置元数据
func (e *AuditEntry) WithMetadata(metadata interface{}) *AuditEntry {
	e.log.Metadata = models.ToJSON(metadata)
	return e
}

// WithTags 设置标签
func (e *AuditEntry) WithTags(tags ...string) *AuditEntry {
	e.log.Tags = strings.Join(tags, ",")
	return e
}

// WithNotes 设置备注
func (e *AuditEntry) WithNotes(notes string) *AuditEntry {
	e.log.Notes = notes
	return e
}

// WithRequestParams 设置请求参数
func (e *AuditEntry) WithRequestParams(params string) *AuditEntry {
	e.log.RequestParams = params
	return e
}

// WithErrorMessage 设置错误信息（不改变状态）
func (e *AuditEntry) WithErrorMessage(message string) *AuditEntry {
	e.log.ErrorMessage = message
	return e
}

// WithTraceID 设置追踪ID（用于关联多个审计记录）
func (e *AuditEntry) WithTraceID(traceID string) *AuditEntry {
	e.log.TraceID = traceID
	return e
}

// Save 保存审计日志（同步）
func (e *AuditEntry) Save() error {
	// 检查是否启用
	if !e.service.isEnabled(e.log.Category) {
		return nil
	}

	// 保存主记录
	if err := e.service.db.Create(e.log).Error; err != nil {
		logrus.WithError(err).Error("Failed to save audit log")
		return err
	}

	// 保存变更明细
	if len(e.changeDetails) > 0 {
		for i := range e.changeDetails {
			e.changeDetails[i].AuditLogID = e.log.ID
		}
		if err := e.service.db.Create(&e.changeDetails).Error; err != nil {
			logrus.WithError(err).Warn("Failed to save audit change details")
		}
	}

	// 触发通知
	go e.service.triggerNotification(e.log)

	// 同时写入日志文件
	e.service.logToFile(e.log)

	return nil
}

// SaveAsync 异步保存审计日志
func (e *AuditEntry) SaveAsync() {
	// 检查是否启用
	if !e.service.isEnabled(e.log.Category) {
		return
	}

	// 写入日志文件
	e.service.logToFile(e.log)

	// 发送到异步缓冲区
	select {
	case e.service.asyncBuffer <- e.log:
	default:
		// 缓冲区满时同步写入
		logrus.Warn("Audit async buffer full, writing synchronously")
		e.Save()
	}
}

// ==================== 便捷记录方法 ====================

// LogAnsibleExecution 记录 Ansible 执行
func (s *AuditService) LogAnsibleExecution(c *gin.Context, action models.AuditAction, projectID uint, projectName string, executionType string, status models.AuditStatus, err error) {
	entry := s.NewAuditEntry(models.AuditCategoryAnsible, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource("ansible_execution", fmt.Sprintf("%d", projectID), projectName).
		WithStatus(status).
		WithMetadata(map[string]interface{}{
			"execution_type": executionType,
		})

	if err != nil {
		entry.WithError(err)
	}

	entry.SaveAsync()
}

// LogSlurmOperation 记录 SLURM 操作
func (s *AuditService) LogSlurmOperation(c *gin.Context, action models.AuditAction, resourceType, resourceID, resourceName string, oldValue, newValue interface{}, err error) {
	entry := s.NewAuditEntry(models.AuditCategorySlurm, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource(resourceType, resourceID, resourceName).
		WithChange(oldValue, newValue, "")

	if err != nil {
		entry.WithError(err)
	}

	entry.SaveAsync()
}

// LogSaltstackOperation 记录 SaltStack 操作
func (s *AuditService) LogSaltstackOperation(c *gin.Context, action models.AuditAction, target, function string, args interface{}, result interface{}, err error) {
	entry := s.NewAuditEntry(models.AuditCategorySaltstack, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource("salt_command", target, function).
		WithMetadata(map[string]interface{}{
			"target":   target,
			"function": function,
			"args":     args,
			"result":   result,
		})

	if err != nil {
		entry.WithError(err)
	}

	// SaltStack 操作通常比较重要，使用同步保存
	entry.Save()
}

// LogRoleTemplateChange 记录角色模板变更
func (s *AuditService) LogRoleTemplateChange(c *gin.Context, action models.AuditAction, templateID uint, templateName string, oldValue, newValue interface{}) {
	entry := s.NewAuditEntry(models.AuditCategoryRoleTemplate, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource("role_template", fmt.Sprintf("%d", templateID), templateName).
		WithChange(oldValue, newValue, "").
		WithSeverity(models.AuditSeverityCritical) // 角色变更通常很重要

	entry.Save() // 同步保存
}

// LogRoleAssignment 记录角色分配
func (s *AuditService) LogRoleAssignment(c *gin.Context, userID uint, username string, roleID uint, roleName string, isAssign bool) {
	action := models.AuditActionRoleAssign
	if !isAssign {
		action = models.AuditActionRoleRevoke
	}

	s.NewAuditEntry(models.AuditCategoryRoleTemplate, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource("role", fmt.Sprintf("%d", roleID), roleName).
		WithMetadata(map[string]interface{}{
			"target_user_id":  userID,
			"target_username": username,
			"role_id":         roleID,
			"role_name":       roleName,
		}).
		WithSeverity(models.AuditSeverityCritical).
		Save()
}

// LogKubernetesOperation 记录 Kubernetes 操作
func (s *AuditService) LogKubernetesOperation(c *gin.Context, action models.AuditAction, clusterID uint, clusterName string, resourceType, resourceName, namespace string, oldValue, newValue interface{}, err error) {
	entry := s.NewAuditEntry(models.AuditCategoryKubernetes, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource(resourceType, resourceName, fmt.Sprintf("%s/%s", namespace, resourceName)).
		WithChange(oldValue, newValue, "").
		WithMetadata(map[string]interface{}{
			"cluster_id":   clusterID,
			"cluster_name": clusterName,
			"namespace":    namespace,
		})

	if err != nil {
		entry.WithError(err)
	}

	entry.SaveAsync()
}

// LogMonitorOperation 记录监控操作
func (s *AuditService) LogMonitorOperation(c *gin.Context, action models.AuditAction, resourceType, resourceID, resourceName string, oldValue, newValue interface{}, err error) {
	entry := s.NewAuditEntry(models.AuditCategoryMonitor, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource(resourceType, resourceID, resourceName).
		WithChange(oldValue, newValue, "")

	if err != nil {
		entry.WithError(err)
	}

	entry.SaveAsync()
}

// LogAdminOperation 记录管理员操作
func (s *AuditService) LogAdminOperation(c *gin.Context, action models.AuditAction, resourceType, resourceID, resourceName string, oldValue, newValue interface{}, err error) {
	entry := s.NewAuditEntry(models.AuditCategoryAdmin, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource(resourceType, resourceID, resourceName).
		WithChange(oldValue, newValue, "").
		WithSeverity(models.AuditSeverityCritical)

	if err != nil {
		entry.WithError(err)
	}

	entry.Save() // 管理员操作同步保存
}

// LogUserOperation 记录用户管理操作
func (s *AuditService) LogUserOperation(c *gin.Context, action models.AuditAction, targetUserID uint, targetUsername string, oldValue, newValue interface{}, err error) {
	entry := s.NewAuditEntry(models.AuditCategoryAdmin, action).
		WithUserFromContext(c).
		WithRequestFromContext(c).
		WithResource("user", fmt.Sprintf("%d", targetUserID), targetUsername).
		WithChange(oldValue, newValue, "").
		WithSeverity(models.AuditSeverityCritical)

	if err != nil {
		entry.WithError(err)
	}

	entry.Save()
}

// LogSecurityEvent 记录安全事件
func (s *AuditService) LogSecurityEvent(c *gin.Context, action models.AuditAction, resourceType, resourceID string, details interface{}, severity models.AuditSeverity, err error) {
	entry := s.NewAuditEntry(models.AuditCategorySecurity, action).
		WithRequestFromContext(c).
		WithResource(resourceType, resourceID, "").
		WithMetadata(details).
		WithSeverity(severity)

	if c != nil {
		entry.WithUserFromContext(c)
	}

	if err != nil {
		entry.WithError(err)
	}

	entry.Save() // 安全事件同步保存
}

// ==================== 查询方法 ====================

// QueryAuditLogs 查询审计日志
func (s *AuditService) QueryAuditLogs(ctx context.Context, req *models.AuditLogQueryRequest) (*models.AuditLogResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.InfraAuditLog{})

	// 应用过滤条件
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Severity != "" {
		query = query.Where("severity = ?", req.Severity)
	}
	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Username != "" {
		query = query.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.ResourceID != "" {
		query = query.Where("resource_id = ?", req.ResourceID)
	}
	if req.ClientIP != "" {
		query = query.Where("client_ip = ?", req.ClientIP)
	}
	if !req.StartDate.IsZero() {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if !req.EndDate.IsZero() {
		query = query.Where("created_at <= ?", req.EndDate.Add(24*time.Hour))
	}
	if req.Keywords != "" {
		keywords := "%" + req.Keywords + "%"
		query = query.Where(
			"resource_name LIKE ? OR change_summary LIKE ? OR notes LIKE ? OR error_message LIKE ?",
			keywords, keywords, keywords, keywords,
		)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// 排序
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" {
		sortOrder = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// 查询数据
	var logs []models.InfraAuditLog
	if err := query.Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.AuditLogResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       logs,
	}, nil
}

// GetAuditLogByID 根据ID获取审计日志
func (s *AuditService) GetAuditLogByID(ctx context.Context, id uint) (*models.InfraAuditLog, []models.AuditChangeDetail, error) {
	var log models.InfraAuditLog
	if err := s.db.WithContext(ctx).First(&log, id).Error; err != nil {
		return nil, nil, err
	}

	var details []models.AuditChangeDetail
	s.db.WithContext(ctx).Where("audit_log_id = ?", id).Find(&details)

	return &log, details, nil
}

// GetAuditStatistics 获取审计统计
func (s *AuditService) GetAuditStatistics(ctx context.Context, startDate, endDate time.Time) (*models.AuditStatisticsResponse, error) {
	stats := &models.AuditStatisticsResponse{}

	// 总数
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).Count(&stats.TotalLogs)

	// 今日数量
	today := time.Now().Truncate(24 * time.Hour)
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).Where("created_at >= ?", today).Count(&stats.TodayLogs)

	// 成功/失败数量
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).Where("status = ?", models.AuditStatusSuccess).Count(&stats.SuccessCount)
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).Where("status = ?", models.AuditStatusFailed).Count(&stats.FailedCount)

	// 按类别统计
	var categoryStats []models.CategoryStatItem
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&categoryStats)
	stats.CategoryStats = categoryStats

	// 按动作统计（取前10）
	var actionStats []models.ActionStatItem
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Order("count DESC").
		Limit(10).
		Scan(&actionStats)
	stats.ActionStats = actionStats

	// 按用户统计（取前10）
	var userStats []models.UserStatItem
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).
		Select("user_id, username, COUNT(*) as count").
		Where("user_id > 0").
		Group("user_id, username").
		Order("count DESC").
		Limit(10).
		Scan(&userStats)
	stats.UserStats = userStats

	// 趋势数据（最近30天）
	var trendData []models.TrendDataItem
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	s.db.WithContext(ctx).Model(&models.InfraAuditLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", thirtyDaysAgo).
		Group("DATE(created_at)").
		Order("date").
		Scan(&trendData)
	stats.TrendData = trendData

	return stats, nil
}

// ==================== 内部方法 ====================

// isEnabled 检查类别是否启用
func (s *AuditService) isEnabled(category models.AuditCategory) bool {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()

	if cfg, ok := s.configs[category]; ok {
		return cfg.Enabled
	}
	return true // 默认启用
}

// maskSensitiveData 掩码敏感数据
func (s *AuditService) maskSensitiveData(data string) string {
	if data == "" {
		return data
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return data
	}

	masked := s.maskObject(obj)
	result, _ := json.Marshal(masked)
	return string(result)
}

// maskObject 递归掩码对象中的敏感字段
func (s *AuditService) maskObject(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			isSensitive := false
			for _, sf := range s.sensitiveFields {
				if strings.Contains(lowerKey, sf) {
					isSensitive = true
					break
				}
			}
			if isSensitive {
				v[key] = "******"
			} else {
				v[key] = s.maskObject(value)
			}
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = s.maskObject(item)
		}
		return v
	default:
		return obj
	}
}

// computeChanges 计算变更明细
func (s *AuditService) computeChanges(oldValue, newValue interface{}) []models.AuditChangeDetail {
	var changes []models.AuditChangeDetail

	oldMap := s.toMap(oldValue)
	newMap := s.toMap(newValue)

	if oldMap == nil || newMap == nil {
		return changes
	}

	// 查找修改和删除的字段
	for key, oldVal := range oldMap {
		if newVal, ok := newMap[key]; ok {
			if !reflect.DeepEqual(oldVal, newVal) {
				changes = append(changes, models.AuditChangeDetail{
					FieldName:  key,
					OldValue:   fmt.Sprintf("%v", oldVal),
					NewValue:   fmt.Sprintf("%v", newVal),
					ChangeType: "modify",
				})
			}
		} else {
			changes = append(changes, models.AuditChangeDetail{
				FieldName:  key,
				OldValue:   fmt.Sprintf("%v", oldVal),
				NewValue:   "",
				ChangeType: "delete",
			})
		}
	}

	// 查找新增的字段
	for key, newVal := range newMap {
		if _, ok := oldMap[key]; !ok {
			changes = append(changes, models.AuditChangeDetail{
				FieldName:  key,
				OldValue:   "",
				NewValue:   fmt.Sprintf("%v", newVal),
				ChangeType: "add",
			})
		}
	}

	return changes
}

// toMap 将对象转换为 map
func (s *AuditService) toMap(obj interface{}) map[string]interface{} {
	if obj == nil {
		return nil
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		return v
	default:
		data, err := json.Marshal(obj)
		if err != nil {
			return nil
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil
		}
		return result
	}
}

// logToFile 写入日志文件
func (s *AuditService) logToFile(log *models.InfraAuditLog) {
	fields := logrus.Fields{
		"audit":         true,
		"trace_id":      log.TraceID,
		"category":      log.Category,
		"action":        log.Action,
		"status":        log.Status,
		"severity":      log.Severity,
		"user_id":       log.UserID,
		"username":      log.Username,
		"resource_type": log.ResourceType,
		"resource_id":   log.ResourceID,
		"resource_name": log.ResourceName,
		"client_ip":     log.ClientIP,
	}

	if log.ErrorMessage != "" {
		fields["error"] = log.ErrorMessage
	}

	switch log.Severity {
	case models.AuditSeverityAlert:
		logrus.WithFields(fields).Error("AUDIT ALERT")
	case models.AuditSeverityCritical:
		logrus.WithFields(fields).Warn("AUDIT CRITICAL")
	case models.AuditSeverityWarning:
		logrus.WithFields(fields).Warn("AUDIT WARNING")
	default:
		logrus.WithFields(fields).Info("AUDIT")
	}
}

// triggerNotification 触发通知
func (s *AuditService) triggerNotification(log *models.InfraAuditLog) {
	s.configMutex.RLock()
	cfg, ok := s.configs[log.Category]
	s.configMutex.RUnlock()

	if !ok || !cfg.NotifyEnabled {
		return
	}

	// 检查是否需要通知
	if cfg.NotifyOn != "*" {
		actions := strings.Split(cfg.NotifyOn, ",")
		shouldNotify := false
		for _, a := range actions {
			if strings.TrimSpace(a) == string(log.Action) {
				shouldNotify = true
				break
			}
		}
		if !shouldNotify {
			return
		}
	}

	// TODO: 实现实际的通知逻辑（邮件、Webhook、Slack等）
	logrus.WithFields(logrus.Fields{
		"category": log.Category,
		"action":   log.Action,
		"channels": cfg.NotifyChannels,
	}).Debug("Audit notification triggered")
}

// ==================== 配置管理方法 ====================

// GetAuditConfig 获取审计配置
func (s *AuditService) GetAuditConfig(category models.AuditCategory) (*models.AuditConfig, error) {
	var config models.AuditConfig
	if err := s.db.Where("category = ?", category).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// GetAllAuditConfigs 获取所有审计配置
func (s *AuditService) GetAllAuditConfigs() ([]models.AuditConfig, error) {
	var configs []models.AuditConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// UpdateAuditConfig 更新审计配置
func (s *AuditService) UpdateAuditConfig(config *models.AuditConfig) error {
	if err := s.db.Save(config).Error; err != nil {
		return err
	}
	s.ReloadConfigs()
	return nil
}

// InitializeDefaultConfigs 初始化默认配置
func (s *AuditService) InitializeDefaultConfigs() error {
	defaults := models.GetDefaultAuditConfigs()
	for _, cfg := range defaults {
		var existing models.AuditConfig
		if err := s.db.Where("category = ?", cfg.Category).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := s.db.Create(&cfg).Error; err != nil {
					logrus.WithError(err).WithField("category", cfg.Category).Error("Failed to create default audit config")
				}
			}
		}
	}
	s.ReloadConfigs()
	return nil
}

// ==================== 归档和清理方法 ====================

// ArchiveOldLogs 归档旧日志
func (s *AuditService) ArchiveOldLogs(ctx context.Context, category models.AuditCategory, beforeDate time.Time) (*models.AuditArchive, error) {
	// TODO: 实现归档逻辑
	// 1. 查询需要归档的日志
	// 2. 导出到文件
	// 3. 创建归档记录
	// 4. 删除原始记录
	return nil, fmt.Errorf("not implemented")
}

// CleanupExpiredArchives 清理过期归档
func (s *AuditService) CleanupExpiredArchives(ctx context.Context) error {
	// TODO: 实现清理逻辑
	return nil
}

// CleanupOldLogs 清理旧日志（根据配置的保留天数）
func (s *AuditService) CleanupOldLogs(ctx context.Context) error {
	configs, err := s.GetAllAuditConfigs()
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if cfg.RetentionDays > 0 {
			cutoff := time.Now().AddDate(0, 0, -cfg.RetentionDays)
			result := s.db.WithContext(ctx).
				Where("category = ? AND created_at < ?", cfg.Category, cutoff).
				Delete(&models.InfraAuditLog{})
			if result.Error != nil {
				logrus.WithError(result.Error).WithField("category", cfg.Category).Error("Failed to cleanup old audit logs")
			} else if result.RowsAffected > 0 {
				logrus.WithFields(logrus.Fields{
					"category": cfg.Category,
					"deleted":  result.RowsAffected,
				}).Info("Cleaned up old audit logs")
			}
		}
	}

	return nil
}
