package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ClusterPermissionService 集群权限服务
type ClusterPermissionService struct {
	db  *gorm.DB
	log *logrus.Logger
}

// NewClusterPermissionService 创建集群权限服务
func NewClusterPermissionService(db *gorm.DB) *ClusterPermissionService {
	return &ClusterPermissionService{
		db:  db,
		log: logrus.StandardLogger(),
	}
}

// ========================================
// SLURM 集群权限管理
// ========================================

// GrantSlurmPermission 授予SLURM集群权限
func (s *ClusterPermissionService) GrantSlurmPermission(ctx context.Context, input *models.GrantSlurmPermissionInput, grantedBy uint, ipAddress string) (*models.SlurmClusterPermission, error) {
	// 检查集群是否存在
	var cluster models.SlurmCluster
	if err := s.db.First(&cluster, input.ClusterID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("SLURM cluster not found: %d", input.ClusterID)
		}
		return nil, err
	}

	// 检查用户是否存在
	var user models.User
	if err := s.db.First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %d", input.UserID)
		}
		return nil, err
	}

	// 检查是否已有相同的权限记录
	var existingPerm models.SlurmClusterPermission
	err := s.db.Where("user_id = ? AND cluster_id = ? AND is_active = ?", input.UserID, input.ClusterID, true).First(&existingPerm).Error
	if err == nil {
		return nil, fmt.Errorf("user already has active permission for this cluster")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 计算过期时间
	var expiresAt *time.Time
	if input.ValidDays > 0 {
		t := time.Now().AddDate(0, 0, input.ValidDays)
		expiresAt = &t
	}

	// 创建权限记录
	perm := &models.SlurmClusterPermission{
		UserID:        input.UserID,
		ClusterID:     input.ClusterID,
		Verbs:         input.Verbs,
		AllPartitions: input.AllPartitions,
		Partitions:    input.Partitions,
		MaxJobs:       input.MaxJobs,
		MaxCPUs:       input.MaxCPUs,
		MaxGPUs:       input.MaxGPUs,
		MaxMemoryGB:   input.MaxMemoryGB,
		MaxWalltime:   input.MaxWalltime,
		Priority:      input.Priority,
		QOS:           input.QOS,
		Account:       input.Account,
		GrantedBy:     grantedBy,
		GrantReason:   input.Reason,
		ExpiresAt:     expiresAt,
		IsActive:      true,
	}

	if err := s.db.Create(perm).Error; err != nil {
		return nil, err
	}

	// 记录日志
	s.logPermissionChange(ctx, models.PermissionTypeSlurmCluster, perm.ID, input.UserID, grantedBy, "grant", nil, perm, input.Reason, ipAddress)

	// 加载关联数据
	s.db.Preload("User").Preload("Cluster").Preload("Grantor").First(perm, perm.ID)

	return perm, nil
}

// UpdateSlurmPermission 更新SLURM权限
func (s *ClusterPermissionService) UpdateSlurmPermission(ctx context.Context, permID uint, input *models.UpdateSlurmPermissionInput, operatorID uint, ipAddress string) (*models.SlurmClusterPermission, error) {
	var perm models.SlurmClusterPermission
	if err := s.db.First(&perm, permID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("permission not found: %d", permID)
		}
		return nil, err
	}

	// 保存旧值
	oldPerm := perm

	// 更新字段
	if input.Verbs != nil {
		perm.Verbs = input.Verbs
	}
	if input.AllPartitions != nil {
		perm.AllPartitions = *input.AllPartitions
	}
	if input.Partitions != nil {
		perm.Partitions = input.Partitions
	}
	if input.MaxJobs != nil {
		perm.MaxJobs = *input.MaxJobs
	}
	if input.MaxCPUs != nil {
		perm.MaxCPUs = *input.MaxCPUs
	}
	if input.MaxGPUs != nil {
		perm.MaxGPUs = *input.MaxGPUs
	}
	if input.MaxMemoryGB != nil {
		perm.MaxMemoryGB = *input.MaxMemoryGB
	}
	if input.MaxWalltime != nil {
		perm.MaxWalltime = *input.MaxWalltime
	}
	if input.Priority != nil {
		perm.Priority = *input.Priority
	}
	if input.QOS != nil {
		perm.QOS = *input.QOS
	}
	if input.Account != nil {
		perm.Account = *input.Account
	}
	if input.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *input.ExpiresAt)
		if err == nil {
			perm.ExpiresAt = &t
		}
	}

	if err := s.db.Save(&perm).Error; err != nil {
		return nil, err
	}

	// 记录日志
	s.logPermissionChange(ctx, models.PermissionTypeSlurmCluster, perm.ID, perm.UserID, operatorID, "modify", &oldPerm, &perm, input.Reason, ipAddress)

	// 加载关联数据
	s.db.Preload("User").Preload("Cluster").Preload("Grantor").First(&perm, perm.ID)

	return &perm, nil
}

// RevokeSlurmPermission 撤销SLURM权限
func (s *ClusterPermissionService) RevokeSlurmPermission(ctx context.Context, permID uint, reason string, operatorID uint, ipAddress string) error {
	var perm models.SlurmClusterPermission
	if err := s.db.First(&perm, permID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("permission not found: %d", permID)
		}
		return err
	}

	now := time.Now()
	perm.IsActive = false
	perm.RevokedAt = &now
	perm.RevokedBy = &operatorID
	perm.RevokeReason = reason

	if err := s.db.Save(&perm).Error; err != nil {
		return err
	}

	// 记录日志
	s.logPermissionChange(ctx, models.PermissionTypeSlurmCluster, perm.ID, perm.UserID, operatorID, "revoke", &perm, nil, reason, ipAddress)

	return nil
}

// GetSlurmPermission 获取单个SLURM权限
func (s *ClusterPermissionService) GetSlurmPermission(ctx context.Context, permID uint) (*models.SlurmClusterPermission, error) {
	var perm models.SlurmClusterPermission
	if err := s.db.Preload("User").Preload("Cluster").Preload("Grantor").First(&perm, permID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("permission not found: %d", permID)
		}
		return nil, err
	}
	return &perm, nil
}

// ListSlurmPermissions 列出SLURM权限
func (s *ClusterPermissionService) ListSlurmPermissions(ctx context.Context, query *models.ClusterPermissionQuery) (*models.SlurmPermissionListResponse, error) {
	db := s.db.Model(&models.SlurmClusterPermission{})

	// 应用过滤条件
	if query.UserID > 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.ClusterID > 0 {
		db = db.Where("cluster_id = ?", query.ClusterID)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}
	if !query.IncludeExpired {
		db = db.Where("expires_at IS NULL OR expires_at > ?", time.Now())
	}

	// 计数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var items []models.SlurmClusterPermission
	if err := db.Preload("User").Preload("Cluster").Preload("Grantor").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return &models.SlurmPermissionListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}, nil
}

// GetUserSlurmPermissions 获取用户的所有SLURM权限
func (s *ClusterPermissionService) GetUserSlurmPermissions(ctx context.Context, userID uint) ([]models.SlurmClusterPermission, error) {
	var perms []models.SlurmClusterPermission
	if err := s.db.Preload("Cluster").
		Where("user_id = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)", userID, true, time.Now()).
		Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// CheckSlurmAccess 检查用户对SLURM集群的访问权限
func (s *ClusterPermissionService) CheckSlurmAccess(ctx context.Context, userID, clusterID uint, requiredVerb models.ClusterPermissionVerb, partition string) (*models.VerifyPermissionResult, error) {
	var perm models.SlurmClusterPermission
	err := s.db.Where("user_id = ? AND cluster_id = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)",
		userID, clusterID, true, time.Now()).First(&perm).Error

	result := &models.VerifyPermissionResult{
		Allowed: false,
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		result.Reason = "No permission record found for this cluster"
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	// 检查权限动作
	if !perm.HasVerb(requiredVerb) {
		result.Reason = fmt.Sprintf("Missing required verb: %s", requiredVerb)
		result.MissingVerbs = []string{string(requiredVerb)}
		return result, nil
	}

	// 检查分区权限(如果指定了分区)
	if partition != "" && !perm.HasPartitionAccess(partition) {
		result.Reason = fmt.Sprintf("No access to partition: %s", partition)
		return result, nil
	}

	result.Allowed = true
	result.MatchedVerbs = perm.Verbs
	result.ResourceLimits = &models.ResourceLimits{
		MaxJobs:     perm.MaxJobs,
		MaxCPUs:     perm.MaxCPUs,
		MaxGPUs:     perm.MaxGPUs,
		MaxMemoryGB: perm.MaxMemoryGB,
		MaxWalltime: perm.MaxWalltime,
	}

	return result, nil
}

// ========================================
// SaltStack 集群权限管理
// ========================================

// GrantSaltstackPermission 授予SaltStack权限
func (s *ClusterPermissionService) GrantSaltstackPermission(ctx context.Context, input *models.GrantSaltstackPermissionInput, grantedBy uint, ipAddress string) (*models.SaltstackClusterPermission, error) {
	// 检查用户是否存在
	var user models.User
	if err := s.db.First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %d", input.UserID)
		}
		return nil, err
	}

	// 检查是否已有相同的权限记录
	var existingPerm models.SaltstackClusterPermission
	err := s.db.Where("user_id = ? AND master_id = ? AND is_active = ?", input.UserID, input.MasterID, true).First(&existingPerm).Error
	if err == nil {
		return nil, fmt.Errorf("user already has active permission for this Salt master")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 计算过期时间
	var expiresAt *time.Time
	if input.ValidDays > 0 {
		t := time.Now().AddDate(0, 0, input.ValidDays)
		expiresAt = &t
	}

	// 设置默认值
	maxConcurrent := input.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	rateLimit := input.RateLimit
	if rateLimit <= 0 {
		rateLimit = 60
	}

	// 创建权限记录
	perm := &models.SaltstackClusterPermission{
		UserID:           input.UserID,
		MasterID:         input.MasterID,
		MasterAddress:    input.MasterAddress,
		Verbs:            input.Verbs,
		AllMinions:       input.AllMinions,
		MinionGroups:     input.MinionGroups,
		MinionPatterns:   input.MinionPatterns,
		AllowedFunctions: input.AllowedFunctions,
		DeniedFunctions:  input.DeniedFunctions,
		AllowDangerous:   input.AllowDangerous,
		MaxConcurrent:    maxConcurrent,
		RateLimit:        rateLimit,
		GrantedBy:        grantedBy,
		GrantReason:      input.Reason,
		ExpiresAt:        expiresAt,
		IsActive:         true,
	}

	if err := s.db.Create(perm).Error; err != nil {
		return nil, err
	}

	// 记录日志
	s.logPermissionChange(ctx, models.PermissionTypeSaltstackCluster, perm.ID, input.UserID, grantedBy, "grant", nil, perm, input.Reason, ipAddress)

	// 加载关联数据
	s.db.Preload("User").Preload("Grantor").First(perm, perm.ID)

	return perm, nil
}

// UpdateSaltstackPermission 更新SaltStack权限
func (s *ClusterPermissionService) UpdateSaltstackPermission(ctx context.Context, permID uint, input *models.UpdateSaltstackPermissionInput, operatorID uint, ipAddress string) (*models.SaltstackClusterPermission, error) {
	var perm models.SaltstackClusterPermission
	if err := s.db.First(&perm, permID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("permission not found: %d", permID)
		}
		return nil, err
	}

	// 保存旧值
	oldPerm := perm

	// 更新字段
	if input.Verbs != nil {
		perm.Verbs = input.Verbs
	}
	if input.AllMinions != nil {
		perm.AllMinions = *input.AllMinions
	}
	if input.MinionGroups != nil {
		perm.MinionGroups = input.MinionGroups
	}
	if input.MinionPatterns != nil {
		perm.MinionPatterns = input.MinionPatterns
	}
	if input.AllowedFunctions != nil {
		perm.AllowedFunctions = input.AllowedFunctions
	}
	if input.DeniedFunctions != nil {
		perm.DeniedFunctions = input.DeniedFunctions
	}
	if input.AllowDangerous != nil {
		perm.AllowDangerous = *input.AllowDangerous
	}
	if input.MaxConcurrent != nil {
		perm.MaxConcurrent = *input.MaxConcurrent
	}
	if input.RateLimit != nil {
		perm.RateLimit = *input.RateLimit
	}
	if input.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *input.ExpiresAt)
		if err == nil {
			perm.ExpiresAt = &t
		}
	}

	if err := s.db.Save(&perm).Error; err != nil {
		return nil, err
	}

	// 记录日志
	s.logPermissionChange(ctx, models.PermissionTypeSaltstackCluster, perm.ID, perm.UserID, operatorID, "modify", &oldPerm, &perm, input.Reason, ipAddress)

	// 加载关联数据
	s.db.Preload("User").Preload("Grantor").First(&perm, perm.ID)

	return &perm, nil
}

// RevokeSaltstackPermission 撤销SaltStack权限
func (s *ClusterPermissionService) RevokeSaltstackPermission(ctx context.Context, permID uint, reason string, operatorID uint, ipAddress string) error {
	var perm models.SaltstackClusterPermission
	if err := s.db.First(&perm, permID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("permission not found: %d", permID)
		}
		return err
	}

	now := time.Now()
	perm.IsActive = false
	perm.RevokedAt = &now
	perm.RevokedBy = &operatorID
	perm.RevokeReason = reason

	if err := s.db.Save(&perm).Error; err != nil {
		return err
	}

	// 记录日志
	s.logPermissionChange(ctx, models.PermissionTypeSaltstackCluster, perm.ID, perm.UserID, operatorID, "revoke", &perm, nil, reason, ipAddress)

	return nil
}

// GetSaltstackPermission 获取单个SaltStack权限
func (s *ClusterPermissionService) GetSaltstackPermission(ctx context.Context, permID uint) (*models.SaltstackClusterPermission, error) {
	var perm models.SaltstackClusterPermission
	if err := s.db.Preload("User").Preload("Grantor").First(&perm, permID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("permission not found: %d", permID)
		}
		return nil, err
	}
	return &perm, nil
}

// ListSaltstackPermissions 列出SaltStack权限
func (s *ClusterPermissionService) ListSaltstackPermissions(ctx context.Context, query *models.ClusterPermissionQuery) (*models.SaltstackPermissionListResponse, error) {
	db := s.db.Model(&models.SaltstackClusterPermission{})

	// 应用过滤条件
	if query.UserID > 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.MasterID != "" {
		db = db.Where("master_id = ?", query.MasterID)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}
	if !query.IncludeExpired {
		db = db.Where("expires_at IS NULL OR expires_at > ?", time.Now())
	}

	// 计数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var items []models.SaltstackClusterPermission
	if err := db.Preload("User").Preload("Grantor").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return &models.SaltstackPermissionListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}, nil
}

// GetUserSaltstackPermissions 获取用户的所有SaltStack权限
func (s *ClusterPermissionService) GetUserSaltstackPermissions(ctx context.Context, userID uint) ([]models.SaltstackClusterPermission, error) {
	var perms []models.SaltstackClusterPermission
	if err := s.db.Where("user_id = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)", userID, true, time.Now()).
		Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// CheckSaltstackAccess 检查用户对SaltStack集群的访问权限
func (s *ClusterPermissionService) CheckSaltstackAccess(ctx context.Context, userID uint, masterID string, requiredVerb models.ClusterPermissionVerb, minionID string, function string) (*models.VerifyPermissionResult, error) {
	var perm models.SaltstackClusterPermission
	err := s.db.Where("user_id = ? AND master_id = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)",
		userID, masterID, true, time.Now()).First(&perm).Error

	result := &models.VerifyPermissionResult{
		Allowed: false,
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		result.Reason = "No permission record found for this Salt master"
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	// 检查权限动作
	if !perm.HasVerb(requiredVerb) {
		result.Reason = fmt.Sprintf("Missing required verb: %s", requiredVerb)
		result.MissingVerbs = []string{string(requiredVerb)}
		return result, nil
	}

	// 检查Minion访问权限(如果指定了minionID)
	if minionID != "" && !perm.HasMinionAccess(minionID) {
		result.Reason = fmt.Sprintf("No access to minion: %s", minionID)
		return result, nil
	}

	// 检查函数权限(如果指定了function)
	if function != "" && !perm.IsFunctionAllowed(function) {
		result.Reason = fmt.Sprintf("Function not allowed: %s", function)
		return result, nil
	}

	result.Allowed = true
	result.MatchedVerbs = perm.Verbs

	return result, nil
}

// ========================================
// 用户权限汇总
// ========================================

// GetUserClusterPermissions 获取用户的所有集群权限
func (s *ClusterPermissionService) GetUserClusterPermissions(ctx context.Context, userID uint) (*models.UserClusterPermissions, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %d", userID)
		}
		return nil, err
	}

	slurmPerms, err := s.GetUserSlurmPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	saltPerms, err := s.GetUserSaltstackPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.UserClusterPermissions{
		UserID:    userID,
		Username:  user.Username,
		Slurm:     slurmPerms,
		Saltstack: saltPerms,
	}, nil
}

// GetClusterAccessList 获取用户可访问的集群列表
func (s *ClusterPermissionService) GetClusterAccessList(ctx context.Context, userID uint) ([]models.ClusterAccessInfo, error) {
	var result []models.ClusterAccessInfo

	// 获取SLURM集群权限
	slurmPerms, err := s.GetUserSlurmPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, perm := range slurmPerms {
		result = append(result, models.ClusterAccessInfo{
			ClusterID:   perm.ClusterID,
			ClusterName: perm.Cluster.Name,
			ClusterType: "slurm",
			HasAccess:   perm.IsValid(),
			Verbs:       perm.Verbs,
			Partitions:  perm.Partitions,
			ResourceLimits: &models.ResourceLimits{
				MaxJobs:     perm.MaxJobs,
				MaxCPUs:     perm.MaxCPUs,
				MaxGPUs:     perm.MaxGPUs,
				MaxMemoryGB: perm.MaxMemoryGB,
				MaxWalltime: perm.MaxWalltime,
			},
		})
	}

	// 获取SaltStack集群权限
	saltPerms, err := s.GetUserSaltstackPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, perm := range saltPerms {
		result = append(result, models.ClusterAccessInfo{
			ClusterID:    0, // SaltStack没有数字ID
			ClusterName:  perm.MasterID,
			ClusterType:  "saltstack",
			HasAccess:    perm.IsValid(),
			Verbs:        perm.Verbs,
			MinionGroups: perm.MinionGroups,
		})
	}

	return result, nil
}

// ========================================
// 权限日志
// ========================================

// logPermissionChange 记录权限变更日志
func (s *ClusterPermissionService) logPermissionChange(ctx context.Context, permType models.ClusterPermissionType, permID, userID, operatorID uint, action string, oldValue, newValue interface{}, reason, ipAddress string) {
	var oldJSON, newJSON string

	if oldValue != nil {
		if b, err := json.Marshal(oldValue); err == nil {
			oldJSON = string(b)
		}
	}
	if newValue != nil {
		if b, err := json.Marshal(newValue); err == nil {
			newJSON = string(b)
		}
	}

	log := &models.ClusterPermissionLog{
		PermissionType: permType,
		PermissionID:   permID,
		UserID:         userID,
		OperatorID:     operatorID,
		Action:         action,
		OldValue:       oldJSON,
		NewValue:       newJSON,
		Reason:         reason,
		IPAddress:      ipAddress,
	}

	if err := s.db.Create(log).Error; err != nil {
		s.log.WithError(err).Error("Failed to create permission log")
	}
}

// GetPermissionLogs 获取权限变更日志
func (s *ClusterPermissionService) GetPermissionLogs(ctx context.Context, permType models.ClusterPermissionType, permID uint, page, pageSize int) ([]models.ClusterPermissionLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	db := s.db.Model(&models.ClusterPermissionLog{})
	if permType != "" {
		db = db.Where("permission_type = ?", permType)
	}
	if permID > 0 {
		db = db.Where("permission_id = ?", permID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.ClusterPermissionLog
	if err := db.Preload("User").Preload("Operator").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ========================================
// 权限过期检查
// ========================================

// ExpirePermissions 检查并过期超时的权限
func (s *ClusterPermissionService) ExpirePermissions(ctx context.Context) (int64, error) {
	now := time.Now()
	var count int64

	// 过期SLURM权限
	result := s.db.Model(&models.SlurmClusterPermission{}).
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at < ?", true, now).
		Update("is_active", false)
	if result.Error != nil {
		return 0, result.Error
	}
	count += result.RowsAffected

	// 过期SaltStack权限
	result = s.db.Model(&models.SaltstackClusterPermission{}).
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at < ?", true, now).
		Update("is_active", false)
	if result.Error != nil {
		return 0, result.Error
	}
	count += result.RowsAffected

	if count > 0 {
		s.log.WithField("count", count).Info("Expired cluster permissions")
	}

	return count, nil
}

// GetExpiringPermissions 获取即将过期的权限(用于通知)
func (s *ClusterPermissionService) GetExpiringPermissions(ctx context.Context, withinDays int) ([]models.SlurmClusterPermission, []models.SaltstackClusterPermission, error) {
	deadline := time.Now().AddDate(0, 0, withinDays)

	var slurmPerms []models.SlurmClusterPermission
	if err := s.db.Preload("User").Preload("Cluster").
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at <= ? AND expires_at > ?", true, deadline, time.Now()).
		Find(&slurmPerms).Error; err != nil {
		return nil, nil, err
	}

	var saltPerms []models.SaltstackClusterPermission
	if err := s.db.Preload("User").
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at <= ? AND expires_at > ?", true, deadline, time.Now()).
		Find(&saltPerms).Error; err != nil {
		return nil, nil, err
	}

	return slurmPerms, saltPerms, nil
}
