package services

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/gorm"
)

// AccessControlService IP访问控制服务
type AccessControlService struct {
	db              *gorm.DB
	policyCache     map[string]*models.AccessControlPolicy
	policyMu        sync.RWMutex
	cacheTTL        time.Duration
	lastCacheUpdate time.Time
}

var (
	accessControlServiceInstance *AccessControlService
	accessControlServiceOnce     sync.Once
)

// NewAccessControlService 创建访问控制服务（单例）
func NewAccessControlService(db *gorm.DB) *AccessControlService {
	accessControlServiceOnce.Do(func() {
		accessControlServiceInstance = &AccessControlService{
			db:          db,
			policyCache: make(map[string]*models.AccessControlPolicy),
			cacheTTL:    5 * time.Minute,
		}
		// 自动迁移数据库表
		if err := db.AutoMigrate(
			&models.AccessControlPolicy{},
			&models.AccessControlLog{},
		); err != nil {
			log.Printf("[AccessControlService] 自动迁移失败: %v", err)
		}
		// 初始化加载策略缓存
		accessControlServiceInstance.reloadPolicies()
	})
	return accessControlServiceInstance
}

// GetAccessControlService 获取访问控制服务实例
func GetAccessControlService() *AccessControlService {
	return accessControlServiceInstance
}

// reloadPolicies 重新加载所有启用的策略到缓存
func (s *AccessControlService) reloadPolicies() {
	s.policyMu.Lock()
	defer s.policyMu.Unlock()

	s.policyCache = make(map[string]*models.AccessControlPolicy)

	var policies []models.AccessControlPolicy
	if err := s.db.Where("enabled = ?", true).Order("priority ASC").Find(&policies).Error; err != nil {
		log.Printf("[AccessControlService] 加载策略失败: %v", err)
		return
	}

	for i := range policies {
		s.policyCache[policies[i].PolicyName] = &policies[i]
	}
	s.lastCacheUpdate = time.Now()
	log.Printf("[AccessControlService] 已加载 %d 个访问控制策略到缓存", len(policies))
}

// IsIPAllowed 检查IP是否被允许访问（考虑所有启用的策略）
// 返回: (是否允许, 拒绝原因)
func (s *AccessControlService) IsIPAllowed(ctx context.Context, ipAddress string, includeAuthentication bool) (bool, string) {
	s.policyMu.RLock()
	policies := make([]*models.AccessControlPolicy, 0, len(s.policyCache))
	for _, p := range s.policyCache {
		if !includeAuthentication && !p.IncludeAuthentication {
			continue
		}
		policies = append(policies, p)
	}
	s.policyMu.RUnlock()

	// 如果没有策略，允许访问
	if len(policies) == 0 {
		return true, ""
	}

	// 检查所有白名单策略
	for _, policy := range policies {
		if policy.PolicyType == "whitelist" {
			if s.isIPInList(ipAddress, policy.IPList) {
				return true, ""
			}
		}
	}

	// 检查所有黑名单策略
	for _, policy := range policies {
		if policy.PolicyType == "blacklist" {
			if s.isIPInList(ipAddress, policy.IPList) {
				return false, "IP address is in blacklist: " + policy.PolicyName
			}
		}
	}

	// 默认：如果有白名单策略但IP不在其中，拒绝
	hasWhitelist := false
	for _, policy := range policies {
		if policy.PolicyType == "whitelist" {
			hasWhitelist = true
			break
		}
	}
	if hasWhitelist {
		return false, "IP address not in whitelist"
	}

	// 没有白名单策略或IP通过所有检查
	return true, ""
}

// isIPInList 检查IP是否在IP列表中（支持CIDR范围和精确匹配）
func (s *AccessControlService) isIPInList(ipAddress string, ipListJSON interface{}) bool {
	var ips []string

	// 处理 json.RawMessage 类型
	var jsonData []byte
	if raw, ok := ipListJSON.(json.RawMessage); ok {
		jsonData = []byte(raw)
	} else if raw, ok := ipListJSON.([]byte); ok {
		jsonData = raw
	} else {
		return false
	}

	if err := json.Unmarshal(jsonData, &ips); err != nil {
		log.Printf("[AccessControlService] 无法解析IP列表: %v", err)
		return false
	}

	clientIP := net.ParseIP(ipAddress)
	if clientIP == nil {
		return false
	}

	for _, item := range ips {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		// 尝试作为CIDR范围
		if _, network, err := net.ParseCIDR(item); err == nil {
			if network.Contains(clientIP) {
				return true
			}
		} else if item == ipAddress {
			// 精确匹配
			return true
		}
	}

	return false
}

// CheckUserIPAccess 检查用户是否可以从指定IP登录
// 返回: (是否允许, 拒绝原因)
func (s *AccessControlService) CheckUserIPAccess(ctx context.Context, username string, ipAddress string) (bool, string) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		// 用户不存在时，不进行IP检查
		return true, ""
	}

	// 检查用户是否配置了IP限制
	var allowedIPs []string
	if len(user.AllowedIPs) > 0 {
		if err := json.Unmarshal(user.AllowedIPs, &allowedIPs); err == nil && len(allowedIPs) > 0 {
			// 用户配置了IP限制
			if !s.isIPInList(ipAddress, user.AllowedIPs) {
				return false, "User login from this IP is not allowed"
			}
		}
	}

	// 检查全局访问控制策略
	return s.IsIPAllowed(ctx, ipAddress, true)
}

// LogAccessAttempt 记录访问尝试
func (s *AccessControlService) LogAccessAttempt(ctx context.Context, log *models.AccessControlLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return s.db.Create(log).Error
}

// GetAccessLogs 获取访问日志
func (s *AccessControlService) GetAccessLogs(ctx context.Context, username string, limit int) ([]models.AccessControlLog, error) {
	var logs []models.AccessControlLog
	query := s.db.Where("username = ?", username)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// CreatePolicy 创建访问控制策略
func (s *AccessControlService) CreatePolicy(ctx context.Context, policy *models.AccessControlPolicy) error {
	if err := s.db.Create(policy).Error; err != nil {
		return err
	}
	// 重新加载策略缓存
	s.reloadPolicies()
	return nil
}

// UpdatePolicy 更新访问控制策略
func (s *AccessControlService) UpdatePolicy(ctx context.Context, policy *models.AccessControlPolicy) error {
	if err := s.db.Save(policy).Error; err != nil {
		return err
	}
	// 重新加载策略缓存
	s.reloadPolicies()
	return nil
}

// DeletePolicy 删除访问控制策略
func (s *AccessControlService) DeletePolicy(ctx context.Context, policyID uint) error {
	if err := s.db.Delete(&models.AccessControlPolicy{}, policyID).Error; err != nil {
		return err
	}
	// 重新加载策略缓存
	s.reloadPolicies()
	return nil
}

// GetPolicies 获取所有访问控制策略
func (s *AccessControlService) GetPolicies(ctx context.Context) ([]models.AccessControlPolicy, error) {
	var policies []models.AccessControlPolicy
	if err := s.db.Order("priority ASC").Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// RefreshPolicyCache 强制刷新策略缓存
func (s *AccessControlService) RefreshPolicyCache(ctx context.Context) {
	s.reloadPolicies()
}
