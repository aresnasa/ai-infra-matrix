package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ComponentRegistrationService 组件注册服务接口
type ComponentRegistrationService interface {
	// RegisterUserToComponents 注册用户到指定组件
	RegisterUserToComponents(user models.User, password string, components []models.ComponentPermissionConfig) error
	// RegisterUserToAllEnabledComponents 注册用户到所有已启用的组件（根据角色模板）
	RegisterUserToAllEnabledComponents(user models.User, password string, roleTemplate string) error
	// SyncUserToComponent 同步用户到单个组件
	SyncUserToComponent(user models.User, password string, component string, permission models.UserComponentPermission) error
	// UpdateUserComponentPermission 更新用户的组件权限
	UpdateUserComponentPermission(userID uint, component string, req models.UpdateComponentPermissionRequest) error
	// GetUserComponentPermissions 获取用户的所有组件权限
	GetUserComponentPermissions(userID uint) ([]models.UserComponentPermission, error)
	// GetComponentSyncStatus 获取组件同步状态
	GetComponentSyncStatus(userID uint) ([]models.ComponentSyncStatus, error)
	// DeleteUserFromComponent 从组件中删除用户
	DeleteUserFromComponent(userID uint, component string) error
	// DeleteUserFromAllComponents 从所有组件中删除用户
	DeleteUserFromAllComponents(userID uint) error
	// GetRoleTemplateComponentPermissions 获取角色模板的组件权限配置
	GetRoleTemplateComponentPermissions(roleTemplate string) ([]models.RoleTemplateComponentPermission, error)
	// SetRoleTemplateComponentPermissions 设置角色模板的组件权限配置
	SetRoleTemplateComponentPermissions(roleTemplateID uint, permissions []models.RoleTemplateComponentPermission) error
	// RetryFailedSyncs 重试失败的同步任务
	RetryFailedSyncs() error
}

// componentRegistrationServiceImpl 组件注册服务实现
type componentRegistrationServiceImpl struct {
	cfg                   *config.Config
	db                    *gorm.DB
	giteaService          GiteaService
	n9eUserService        NightingaleUserService
	keycloakService       KeycloakService
	seaweedFSUserService  SeaweedFSUserService
	jupyterHubUserService JupyterHubUserService
	slurmUserService      SlurmUserService
}

// NewComponentRegistrationService 创建组件注册服务
func NewComponentRegistrationService(
	cfg *config.Config,
	db *gorm.DB,
	giteaService GiteaService,
	n9eUserService NightingaleUserService,
	keycloakService KeycloakService,
	seaweedFSUserService SeaweedFSUserService,
	jupyterHubUserService JupyterHubUserService,
	slurmUserService SlurmUserService,
) ComponentRegistrationService {
	return &componentRegistrationServiceImpl{
		cfg:                   cfg,
		db:                    db,
		giteaService:          giteaService,
		n9eUserService:        n9eUserService,
		keycloakService:       keycloakService,
		seaweedFSUserService:  seaweedFSUserService,
		jupyterHubUserService: jupyterHubUserService,
		slurmUserService:      slurmUserService,
	}
}

// RegisterUserToAllEnabledComponents 根据角色模板注册用户到所有启用的组件
func (s *componentRegistrationServiceImpl) RegisterUserToAllEnabledComponents(user models.User, password string, roleTemplate string) error {
	logger := logrus.WithFields(logrus.Fields{
		"username":      user.Username,
		"role_template": roleTemplate,
		"service":       "component_registration",
	})
	logger.Info("Starting user registration to all enabled components")

	// 获取角色模板的组件权限配置
	templatePermissions, err := s.GetRoleTemplateComponentPermissions(roleTemplate)
	if err != nil {
		logger.WithError(err).Warn("Failed to get role template component permissions, using defaults")
		templatePermissions = s.getDefaultComponentPermissions()
	}

	// 转换为 ComponentPermissionConfig
	var components []models.ComponentPermissionConfig
	for _, tp := range templatePermissions {
		if tp.Enabled {
			components = append(components, models.ComponentPermissionConfig{
				Component:       tp.Component,
				PermissionLevel: tp.PermissionLevel,
				Enabled:         tp.Enabled,
				Metadata:        tp.DefaultMetadata,
			})
		}
	}

	if len(components) == 0 {
		logger.Info("No components enabled for this role template")
		return nil
	}

	return s.RegisterUserToComponents(user, password, components)
}

// RegisterUserToComponents 注册用户到指定组件
func (s *componentRegistrationServiceImpl) RegisterUserToComponents(user models.User, password string, components []models.ComponentPermissionConfig) error {
	logger := logrus.WithFields(logrus.Fields{
		"username":        user.Username,
		"component_count": len(components),
		"service":         "component_registration",
	})
	logger.Info("Starting user registration to components")

	var wg sync.WaitGroup
	errChan := make(chan error, len(components))
	resultChan := make(chan models.UserComponentPermission, len(components))

	for _, comp := range components {
		wg.Add(1)
		go func(c models.ComponentPermissionConfig) {
			defer wg.Done()

			// 创建或更新组件权限记录
			permission := models.UserComponentPermission{
				UserID:          user.ID,
				Component:       c.Component,
				PermissionLevel: c.PermissionLevel,
				Enabled:         c.Enabled,
				Metadata:        c.Metadata,
				SyncStatus:      "pending",
			}

			// 先保存到数据库
			if err := s.saveComponentPermission(&permission); err != nil {
				logger.WithError(err).WithField("component", c.Component).Error("Failed to save component permission")
				errChan <- fmt.Errorf("save permission for %s: %w", c.Component, err)
				return
			}

			// 同步到组件
			if err := s.SyncUserToComponent(user, password, c.Component, permission); err != nil {
				logger.WithError(err).WithField("component", c.Component).Warn("Failed to sync user to component")
				permission.SyncStatus = "failed"
				permission.SyncError = err.Error()
			} else {
				now := time.Now()
				permission.SyncStatus = "synced"
				permission.LastSyncAt = &now
			}

			// 更新同步状态
			s.db.Save(&permission)
			resultChan <- permission
		}(comp)
	}

	// 等待所有同步完成
	wg.Wait()
	close(errChan)
	close(resultChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	// 记录同步任务
	s.recordSyncTask(user.ID, components, errors)

	if len(errors) > 0 {
		logger.WithField("error_count", len(errors)).Warn("Some component registrations failed")
		return fmt.Errorf("partial failure: %d of %d components failed", len(errors), len(components))
	}

	logger.Info("User registered to all components successfully")
	return nil
}

// SyncUserToComponent 同步用户到单个组件
func (s *componentRegistrationServiceImpl) SyncUserToComponent(user models.User, password string, component string, permission models.UserComponentPermission) error {
	if !permission.Enabled {
		return nil
	}

	logger := logrus.WithFields(logrus.Fields{
		"username":  user.Username,
		"component": component,
		"level":     permission.PermissionLevel,
	})

	switch component {
	case models.ComponentNightingale:
		return s.syncToNightingale(user, permission, logger)
	case models.ComponentGitea:
		return s.syncToGitea(user, permission, logger)
	case models.ComponentSeaweedFS:
		return s.syncToSeaweedFS(user, password, permission, logger)
	case models.ComponentJupyterHub:
		return s.syncToJupyterHub(user, password, permission, logger)
	case models.ComponentSlurm:
		return s.syncToSlurm(user, permission, logger)
	case models.ComponentKeycloak:
		return s.syncToKeycloak(user, password, permission, logger)
	default:
		return fmt.Errorf("unknown component: %s", component)
	}
}

// syncToNightingale 同步用户到 Nightingale
func (s *componentRegistrationServiceImpl) syncToNightingale(user models.User, permission models.UserComponentPermission, logger *logrus.Entry) error {
	if s.n9eUserService == nil || !s.cfg.Nightingale.Enabled {
		logger.Debug("Nightingale service not available, skipping")
		return nil
	}

	roles := mapPermissionLevelToN9eRoles(permission.PermissionLevel)
	return s.n9eUserService.EnsureUser(user, roles)
}

// syncToGitea 同步用户到 Gitea
func (s *componentRegistrationServiceImpl) syncToGitea(user models.User, permission models.UserComponentPermission, logger *logrus.Entry) error {
	if s.giteaService == nil || !s.cfg.Gitea.Enabled {
		logger.Debug("Gitea service not available, skipping")
		return nil
	}

	return s.giteaService.EnsureUser(user)
}

// syncToSeaweedFS 同步用户到 SeaweedFS
func (s *componentRegistrationServiceImpl) syncToSeaweedFS(user models.User, password string, permission models.UserComponentPermission, logger *logrus.Entry) error {
	if s.seaweedFSUserService == nil {
		logger.Debug("SeaweedFS user service not available, skipping")
		return nil
	}

	// 解析 metadata 获取存储桶配置
	var metadata SeaweedFSUserMetadata
	if permission.Metadata != nil {
		if err := json.Unmarshal(permission.Metadata, &metadata); err != nil {
			logger.WithError(err).Warn("Failed to parse SeaweedFS metadata, using defaults")
		}
	}

	return s.seaweedFSUserService.EnsureUser(user, permission.PermissionLevel, metadata)
}

// syncToJupyterHub 同步用户到 JupyterHub
func (s *componentRegistrationServiceImpl) syncToJupyterHub(user models.User, password string, permission models.UserComponentPermission, logger *logrus.Entry) error {
	if s.jupyterHubUserService == nil {
		logger.Debug("JupyterHub user service not available, skipping")
		return nil
	}

	// 解析 metadata 获取资源配额
	var metadata JupyterHubUserMetadata
	if permission.Metadata != nil {
		if err := json.Unmarshal(permission.Metadata, &metadata); err != nil {
			logger.WithError(err).Warn("Failed to parse JupyterHub metadata, using defaults")
		}
	}

	return s.jupyterHubUserService.EnsureUser(user, password, permission.PermissionLevel, metadata)
}

// syncToSlurm 同步用户到 SLURM
func (s *componentRegistrationServiceImpl) syncToSlurm(user models.User, permission models.UserComponentPermission, logger *logrus.Entry) error {
	if s.slurmUserService == nil {
		logger.Debug("SLURM user service not available, skipping")
		return nil
	}

	// 解析 metadata 获取分区和配额配置
	var metadata SlurmUserMetadata
	if permission.Metadata != nil {
		if err := json.Unmarshal(permission.Metadata, &metadata); err != nil {
			logger.WithError(err).Warn("Failed to parse SLURM metadata, using defaults")
		}
	}

	return s.slurmUserService.EnsureUser(user, permission.PermissionLevel, metadata)
}

// syncToKeycloak 同步用户到 Keycloak
func (s *componentRegistrationServiceImpl) syncToKeycloak(user models.User, password string, permission models.UserComponentPermission, logger *logrus.Entry) error {
	if s.keycloakService == nil || !s.cfg.Keycloak.Enabled {
		logger.Debug("Keycloak service not available, skipping")
		return nil
	}

	roles := mapPermissionLevelToKeycloakRoles(permission.PermissionLevel)
	return s.keycloakService.CreateUser(user, password, roles)
}

// UpdateUserComponentPermission 更新用户的组件权限
func (s *componentRegistrationServiceImpl) UpdateUserComponentPermission(userID uint, component string, req models.UpdateComponentPermissionRequest) error {
	var permission models.UserComponentPermission
	err := s.db.Where("user_id = ? AND component = ?", userID, component).First(&permission).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新记录
			permission = models.UserComponentPermission{
				UserID:    userID,
				Component: component,
			}
		} else {
			return err
		}
	}

	// 更新字段
	if req.PermissionLevel != "" {
		permission.PermissionLevel = req.PermissionLevel
	}
	if req.Enabled != nil {
		permission.Enabled = *req.Enabled
	}
	if req.Metadata != nil {
		permission.Metadata = req.Metadata
	}

	// 标记需要重新同步
	permission.SyncStatus = "pending"

	return s.db.Save(&permission).Error
}

// GetUserComponentPermissions 获取用户的所有组件权限
func (s *componentRegistrationServiceImpl) GetUserComponentPermissions(userID uint) ([]models.UserComponentPermission, error) {
	var permissions []models.UserComponentPermission
	err := s.db.Where("user_id = ?", userID).Find(&permissions).Error
	return permissions, err
}

// GetComponentSyncStatus 获取组件同步状态
func (s *componentRegistrationServiceImpl) GetComponentSyncStatus(userID uint) ([]models.ComponentSyncStatus, error) {
	permissions, err := s.GetUserComponentPermissions(userID)
	if err != nil {
		return nil, err
	}

	var statuses []models.ComponentSyncStatus
	for _, p := range permissions {
		statuses = append(statuses, models.ComponentSyncStatus{
			Component:    p.Component,
			Status:       p.SyncStatus,
			LastSyncAt:   p.LastSyncAt,
			Error:        p.SyncError,
			ExternalID:   p.ExternalUserID,
			ExternalUser: p.ExternalUsername,
		})
	}
	return statuses, nil
}

// DeleteUserFromComponent 从组件中删除用户
func (s *componentRegistrationServiceImpl) DeleteUserFromComponent(userID uint, component string) error {
	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	logger := logrus.WithFields(logrus.Fields{
		"user_id":   userID,
		"username":  user.Username,
		"component": component,
	})

	var err error
	switch component {
	case models.ComponentNightingale:
		if s.n9eUserService != nil {
			err = s.n9eUserService.DeleteUser(user.Username)
		}
	case models.ComponentGitea:
		// Gitea 用户通常不删除，保留代码历史
		logger.Info("Gitea user deletion skipped to preserve code history")
	case models.ComponentSeaweedFS:
		if s.seaweedFSUserService != nil {
			err = s.seaweedFSUserService.DeleteUser(user.Username)
		}
	case models.ComponentJupyterHub:
		if s.jupyterHubUserService != nil {
			err = s.jupyterHubUserService.DeleteUser(user.Username)
		}
	case models.ComponentSlurm:
		if s.slurmUserService != nil {
			err = s.slurmUserService.DeleteUser(user.Username)
		}
	case models.ComponentKeycloak:
		if s.keycloakService != nil {
			err = s.keycloakService.DeleteUser(user.Username)
		}
	}

	if err != nil {
		logger.WithError(err).Warn("Failed to delete user from component")
	}

	// 删除权限记录
	return s.db.Where("user_id = ? AND component = ?", userID, component).Delete(&models.UserComponentPermission{}).Error
}

// DeleteUserFromAllComponents 从所有组件中删除用户
func (s *componentRegistrationServiceImpl) DeleteUserFromAllComponents(userID uint) error {
	permissions, err := s.GetUserComponentPermissions(userID)
	if err != nil {
		return err
	}

	var errors []error
	for _, p := range permissions {
		if err := s.DeleteUserFromComponent(userID, p.Component); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete user from %d components", len(errors))
	}
	return nil
}

// GetRoleTemplateComponentPermissions 获取角色模板的组件权限配置
func (s *componentRegistrationServiceImpl) GetRoleTemplateComponentPermissions(roleTemplate string) ([]models.RoleTemplateComponentPermission, error) {
	// 先获取角色模板ID
	var template models.RoleTemplate
	if err := s.db.Where("name = ?", roleTemplate).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.getDefaultComponentPermissions(), nil
		}
		return nil, err
	}

	var permissions []models.RoleTemplateComponentPermission
	err := s.db.Where("role_template_id = ?", template.ID).Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	// 如果没有配置，返回默认值
	if len(permissions) == 0 {
		return s.getDefaultComponentPermissions(), nil
	}

	return permissions, nil
}

// SetRoleTemplateComponentPermissions 设置角色模板的组件权限配置
func (s *componentRegistrationServiceImpl) SetRoleTemplateComponentPermissions(roleTemplateID uint, permissions []models.RoleTemplateComponentPermission) error {
	// 删除旧的配置
	if err := s.db.Where("role_template_id = ?", roleTemplateID).Delete(&models.RoleTemplateComponentPermission{}).Error; err != nil {
		return err
	}

	// 创建新的配置
	for _, p := range permissions {
		p.RoleTemplateID = roleTemplateID
		if err := s.db.Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

// RetryFailedSyncs 重试失败的同步任务
func (s *componentRegistrationServiceImpl) RetryFailedSyncs() error {
	var failedPermissions []models.UserComponentPermission
	if err := s.db.Where("sync_status = ? AND enabled = ?", "failed", true).
		Preload("User").
		Find(&failedPermissions).Error; err != nil {
		return err
	}

	logger := logrus.WithField("count", len(failedPermissions))
	logger.Info("Retrying failed component syncs")

	for _, p := range failedPermissions {
		if err := s.SyncUserToComponent(p.User, "", p.Component, p); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"user_id":   p.UserID,
				"component": p.Component,
			}).Warn("Retry sync failed")
			p.SyncError = err.Error()
		} else {
			now := time.Now()
			p.SyncStatus = "synced"
			p.SyncError = ""
			p.LastSyncAt = &now
		}
		s.db.Save(&p)
	}

	return nil
}

// saveComponentPermission 保存组件权限记录
func (s *componentRegistrationServiceImpl) saveComponentPermission(permission *models.UserComponentPermission) error {
	var existing models.UserComponentPermission
	err := s.db.Where("user_id = ? AND component = ?", permission.UserID, permission.Component).First(&existing).Error
	if err == nil {
		// 更新现有记录
		permission.ID = existing.ID
		return s.db.Save(permission).Error
	}
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(permission).Error
	}
	return err
}

// recordSyncTask 记录同步任务
func (s *componentRegistrationServiceImpl) recordSyncTask(userID uint, components []models.ComponentPermissionConfig, errors []error) {
	for i, comp := range components {
		task := models.ComponentSyncTask{
			UserID:    userID,
			Component: comp.Component,
			Action:    "create",
			Status:    "completed",
		}
		now := time.Now()
		task.StartedAt = &now
		task.CompletedAt = &now

		if i < len(errors) && errors[i] != nil {
			task.Status = "failed"
			task.ErrorMessage = errors[i].Error()
		}

		s.db.Create(&task)
	}
}

// getDefaultComponentPermissions 获取默认的组件权限配置
func (s *componentRegistrationServiceImpl) getDefaultComponentPermissions() []models.RoleTemplateComponentPermission {
	// 默认配置：基于系统启用的组件
	var permissions []models.RoleTemplateComponentPermission

	if s.cfg.Gitea.Enabled {
		permissions = append(permissions, models.RoleTemplateComponentPermission{
			Component:       models.ComponentGitea,
			PermissionLevel: models.PermissionLevelUser,
			Enabled:         true,
		})
	}

	if s.cfg.Nightingale.Enabled {
		permissions = append(permissions, models.RoleTemplateComponentPermission{
			Component:       models.ComponentNightingale,
			PermissionLevel: models.PermissionLevelReadonly,
			Enabled:         true,
		})
	}

	// SeaweedFS、JupyterHub、SLURM 默认不启用，需要管理员手动分配
	return permissions
}

// Helper functions

// mapPermissionLevelToN9eRoles 映射权限级别到 Nightingale 角色
func mapPermissionLevelToN9eRoles(level string) []string {
	switch level {
	case models.PermissionLevelAdmin:
		return []string{"Admin"}
	case models.PermissionLevelUser:
		return []string{"Standard"}
	case models.PermissionLevelReadonly:
		return []string{"Guest"}
	default:
		return []string{"Guest"}
	}
}

// mapPermissionLevelToKeycloakRoles 映射权限级别到 Keycloak 角色
func mapPermissionLevelToKeycloakRoles(level string) []string {
	switch level {
	case models.PermissionLevelAdmin:
		return []string{"admin", "manage-users"}
	case models.PermissionLevelUser:
		return []string{"user"}
	case models.PermissionLevelReadonly:
		return []string{"viewer"}
	default:
		return []string{"viewer"}
	}
}

// SeaweedFSUserMetadata SeaweedFS 用户元数据
type SeaweedFSUserMetadata struct {
	BucketName   string `json:"bucket_name,omitempty"`
	QuotaBytes   int64  `json:"quota_bytes,omitempty"`
	AllowPublic  bool   `json:"allow_public,omitempty"`
	CustomPolicy string `json:"custom_policy,omitempty"`
}

// JupyterHubUserMetadata JupyterHub 用户元数据
type JupyterHubUserMetadata struct {
	CPULimit       float64  `json:"cpu_limit,omitempty"`
	MemoryLimitMB  int      `json:"memory_limit_mb,omitempty"`
	GPULimit       int      `json:"gpu_limit,omitempty"`
	StorageLimitGB int      `json:"storage_limit_gb,omitempty"`
	ImageName      string   `json:"image_name,omitempty"`
	AllowedGroups  []string `json:"allowed_groups,omitempty"`
}

// SlurmUserMetadata SLURM 用户元数据
type SlurmUserMetadata struct {
	DefaultAccount    string   `json:"default_account,omitempty"`
	AllowedPartitions []string `json:"allowed_partitions,omitempty"`
	MaxJobsPerUser    int      `json:"max_jobs_per_user,omitempty"`
	MaxCPUsPerUser    int      `json:"max_cpus_per_user,omitempty"`
	MaxMemoryMB       int      `json:"max_memory_mb,omitempty"`
	MaxWallTime       string   `json:"max_wall_time,omitempty"`
	DefaultQOS        string   `json:"default_qos,omitempty"`
}
