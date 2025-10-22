package services

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

type HostService struct{}

func NewHostService() *HostService {
	return &HostService{}
}

func (s *HostService) CreateHost(host *models.Host) error {
	if err := database.DB.Create(host).Error; err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.HostsKey(host.ProjectID))
	cache.Delete(cache.ProjectKey(host.ProjectID))

	return nil
}

func (s *HostService) GetHosts(projectID uint) ([]models.Host, error) {
	// 先从缓存获取
	var hosts []models.Host
	cacheKey := cache.HostsKey(projectID)

	if err := cache.Get(cacheKey, &hosts); err == nil {
		return hosts, nil
	}

	// 缓存未命中，从数据库获取
	if err := database.DB.Where("project_id = ?", projectID).Find(&hosts).Error; err != nil {
		return nil, fmt.Errorf("failed to get hosts: %w", err)
	}

	// 存入缓存
	cache.Set(cacheKey, hosts, 30*time.Minute)

	return hosts, nil
}

func (s *HostService) UpdateHost(id uint, host *models.Host) error {
	host.ID = id
	if err := database.DB.Save(host).Error; err != nil {
		return fmt.Errorf("failed to update host: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.HostsKey(host.ProjectID))
	cache.Delete(cache.ProjectKey(host.ProjectID))

	return nil
}

func (s *HostService) DeleteHost(id uint) error {
	var host models.Host
	if err := database.DB.First(&host, id).Error; err != nil {
		return fmt.Errorf("host not found: %w", err)
	}

	if err := database.DB.Delete(&host).Error; err != nil {
		return fmt.Errorf("failed to delete host: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.HostsKey(host.ProjectID))
	cache.Delete(cache.ProjectKey(host.ProjectID))

	return nil
}
