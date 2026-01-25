package services

import (
	"sync"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// UserSyncService 用户同步服务 - 统一管理用户在各子系统的同步
type UserSyncService interface {
	// SyncUserToAllSystems 将用户同步到所有已启用的子系统
	SyncUserToAllSystems(user models.User, password string, roles []string) error
	// SyncUserToGitea 同步用户到 Gitea
	SyncUserToGitea(user models.User) error
	// SyncUserToNightingale 同步用户到 Nightingale
	SyncUserToNightingale(user models.User, roles []string) error
	// SyncUserToKeycloak 同步用户到 Keycloak
	SyncUserToKeycloak(user models.User, password string, roles []string) error
	// DeleteUserFromAllSystems 从所有子系统删除用户
	DeleteUserFromAllSystems(username string) error
	// SyncAllUsersToAllSystems 批量同步所有用户到所有子系统
	SyncAllUsersToAllSystems() (*SyncResult, error)
}

// SyncResult 同步结果
type SyncResult struct {
	Gitea       SubSyncResult `json:"gitea"`
	Nightingale SubSyncResult `json:"nightingale"`
	Keycloak    SubSyncResult `json:"keycloak"`
}

// SubSyncResult 子系统同步结果
type SubSyncResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// userSyncServiceImpl 用户同步服务实现
type userSyncServiceImpl struct {
	cfg             *config.Config
	db              *gorm.DB
	giteaService    GiteaService
	n9eUserService  NightingaleUserService
	keycloakService KeycloakService
}

// NewUserSyncService 创建用户同步服务
func NewUserSyncService(
	cfg *config.Config,
	db *gorm.DB,
	giteaService GiteaService,
	n9eUserService NightingaleUserService,
	keycloakService KeycloakService,
) UserSyncService {
	return &userSyncServiceImpl{
		cfg:             cfg,
		db:              db,
		giteaService:    giteaService,
		n9eUserService:  n9eUserService,
		keycloakService: keycloakService,
	}
}

// SyncUserToAllSystems 将用户同步到所有已启用的子系统
func (s *userSyncServiceImpl) SyncUserToAllSystems(user models.User, password string, roles []string) error {
	logger := logrus.WithFields(logrus.Fields{
		"username": user.Username,
		"service":  "user_sync",
	})

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// 同步到 Gitea
	if s.cfg.Gitea.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.SyncUserToGitea(user); err != nil {
				logger.WithError(err).Warn("Failed to sync user to Gitea")
				errChan <- err
			}
		}()
	}

	// 同步到 Nightingale
	if s.cfg.Nightingale.Enabled && s.cfg.Nightingale.UserSyncEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.SyncUserToNightingale(user, roles); err != nil {
				logger.WithError(err).Warn("Failed to sync user to Nightingale")
				errChan <- err
			}
		}()
	}

	// 同步到 Keycloak
	if s.cfg.Keycloak.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.SyncUserToKeycloak(user, password, roles); err != nil {
				logger.WithError(err).Warn("Failed to sync user to Keycloak")
				errChan <- err
			}
		}()
	}

	// 等待所有同步完成
	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		logger.WithField("error_count", len(errors)).Warn("Some user sync operations failed")
		// 返回第一个错误
		return errors[0]
	}

	logger.Info("User synced to all systems successfully")
	return nil
}

// SyncUserToGitea 同步用户到 Gitea
func (s *userSyncServiceImpl) SyncUserToGitea(user models.User) error {
	if s.giteaService == nil {
		return nil
	}
	return s.giteaService.EnsureUser(user)
}

// SyncUserToNightingale 同步用户到 Nightingale
func (s *userSyncServiceImpl) SyncUserToNightingale(user models.User, roles []string) error {
	if s.n9eUserService == nil {
		return nil
	}
	return s.n9eUserService.EnsureUser(user, roles)
}

// SyncUserToKeycloak 同步用户到 Keycloak
func (s *userSyncServiceImpl) SyncUserToKeycloak(user models.User, password string, roles []string) error {
	if s.keycloakService == nil {
		return nil
	}
	return s.keycloakService.CreateUser(user, password, roles)
}

// DeleteUserFromAllSystems 从所有子系统删除用户
func (s *userSyncServiceImpl) DeleteUserFromAllSystems(username string) error {
	logger := logrus.WithFields(logrus.Fields{
		"username": username,
		"service":  "user_sync",
	})

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// 从 Nightingale 删除
	if s.cfg.Nightingale.Enabled && s.n9eUserService != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.n9eUserService.DeleteUser(username); err != nil {
				logger.WithError(err).Warn("Failed to delete user from Nightingale")
				errChan <- err
			}
		}()
	}

	// 从 Keycloak 删除
	if s.cfg.Keycloak.Enabled && s.keycloakService != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.keycloakService.DeleteUser(username); err != nil {
				logger.WithError(err).Warn("Failed to delete user from Keycloak")
				errChan <- err
			}
		}()
	}

	// Gitea 用户删除通常不需要自动同步（保留代码历史）
	// 如果需要，可以在这里添加

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errors[0]
	}

	logger.Info("User deleted from all systems successfully")
	return nil
}

// SyncAllUsersToAllSystems 批量同步所有用户到所有子系统
func (s *userSyncServiceImpl) SyncAllUsersToAllSystems() (*SyncResult, error) {
	logger := logrus.WithField("service", "user_sync")
	logger.Info("Starting full user sync to all systems")

	result := &SyncResult{}

	// 同步到 Gitea
	if s.cfg.Gitea.Enabled && s.giteaService != nil {
		created, updated, skipped, err := s.giteaService.SyncAllUsers()
		result.Gitea = SubSyncResult{
			Created: created,
			Updated: updated,
			Skipped: skipped,
		}
		if err != nil {
			result.Gitea.Errors = []string{err.Error()}
		}
		logger.WithFields(logrus.Fields{
			"created": created,
			"updated": updated,
			"skipped": skipped,
		}).Info("Gitea user sync completed")
	}

	// 同步到 Nightingale
	if s.cfg.Nightingale.Enabled && s.cfg.Nightingale.UserSyncEnabled && s.n9eUserService != nil {
		created, updated, skipped, err := s.n9eUserService.SyncAllUsers()
		result.Nightingale = SubSyncResult{
			Created: created,
			Updated: updated,
			Skipped: skipped,
		}
		if err != nil {
			result.Nightingale.Errors = []string{err.Error()}
		}
		logger.WithFields(logrus.Fields{
			"created": created,
			"updated": updated,
			"skipped": skipped,
		}).Info("Nightingale user sync completed")
	}

	// 同步到 Keycloak
	if s.cfg.Keycloak.Enabled && s.keycloakService != nil {
		created, updated, skipped, err := s.keycloakService.SyncAllUsers()
		result.Keycloak = SubSyncResult{
			Created: created,
			Updated: updated,
			Skipped: skipped,
		}
		if err != nil {
			result.Keycloak.Errors = []string{err.Error()}
		}
		logger.WithFields(logrus.Fields{
			"created": created,
			"updated": updated,
			"skipped": skipped,
		}).Info("Keycloak user sync completed")
	}

	logger.Info("Full user sync completed")
	return result, nil
}
