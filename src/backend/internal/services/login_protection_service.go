package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// LoginProtectionService 登录保护服务
// 提供账号锁定、IP封禁、登录统计等功能
type LoginProtectionService struct {
	db             *gorm.DB
	mu             sync.RWMutex
	config         *models.SecurityConfig
	configLoadedAt time.Time
	configCacheTTL time.Duration
}

// NewLoginProtectionService 创建登录保护服务实例
func NewLoginProtectionService() *LoginProtectionService {
	return &LoginProtectionService{
		db:             database.DB,
		configCacheTTL: 5 * time.Minute,
	}
}

// getConfig 获取安全配置（带缓存）
func (s *LoginProtectionService) getConfig() (*models.SecurityConfig, error) {
	s.mu.RLock()
	if s.config != nil && time.Since(s.configLoadedAt) < s.configCacheTTL {
		config := s.config
		s.mu.RUnlock()
		return config, nil
	}
	s.mu.RUnlock()

	// 加载配置
	s.mu.Lock()
	defer s.mu.Unlock()

	var config models.SecurityConfig
	if err := s.db.First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回默认配置
			config = models.SecurityConfig{
				MaxLoginAttempts:     5,
				LoginLockoutDuration: 30,
				AutoBlockEnabled:     true,
				AutoBlockThreshold:   10,
				AutoBlockDuration:    1440, // 24小时
			}
		} else {
			return nil, err
		}
	}

	s.config = &config
	s.configLoadedAt = time.Now()
	return s.config, nil
}

// CheckAccountLocked 检查账号是否被锁定
// 返回：是否锁定、剩余锁定时间（秒）、错误
func (s *LoginProtectionService) CheckAccountLocked(username string) (bool, int, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, nil // 用户不存在时不报错，让后续认证处理
		}
		return false, 0, err
	}

	if user.IsLocked() {
		return true, user.GetLockRemainingSeconds(), nil
	}

	return false, 0, nil
}

// CheckIPBlocked 检查IP是否被封禁
// 返回：是否封禁、封禁原因、剩余封禁时间（秒）、错误
func (s *LoginProtectionService) CheckIPBlocked(ip string) (bool, string, int, error) {
	// 首先检查 IP 黑名单
	var blacklist models.IPBlacklist
	if err := s.db.Where("ip = ? AND enabled = ?", ip, true).First(&blacklist).Error; err == nil {
		// 检查是否过期
		if blacklist.ExpireAt == nil || blacklist.ExpireAt.After(time.Now()) {
			remainingSeconds := 0
			if blacklist.ExpireAt != nil {
				remainingSeconds = int(time.Until(*blacklist.ExpireAt).Seconds())
			}
			return true, blacklist.Reason, remainingSeconds, nil
		} else {
			// 已过期，标记为非活跃
			s.db.Model(&blacklist).Update("enabled", false)
		}
	}

	// 检查 IP 登录统计中的封禁状态
	var stats models.IPLoginStats
	if err := s.db.Where("ip = ?", ip).First(&stats).Error; err == nil {
		if stats.IsBlocked() {
			remaining := 0
			if stats.BlockedUntil != nil {
				remaining = int(time.Until(*stats.BlockedUntil).Seconds())
			}
			return true, "登录失败次数过多", remaining, nil
		}
	}

	return false, "", 0, nil
}

// RecordLoginAttempt 记录登录尝试
// success: 是否登录成功
// failureType: 失败类型（成功时传空）
func (s *LoginProtectionService) RecordLoginAttempt(
	ip string,
	username string,
	userAgent string,
	success bool,
	failureType string,
	requestID string,
) error {
	config, err := s.getConfig()
	if err != nil {
		logrus.Warnf("获取安全配置失败: %v", err)
		// 继续使用默认值
	}

	// 查找用户ID
	var userID uint
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err == nil {
		userID = user.ID
	}

	// 创建登录尝试记录
	attempt := &models.LoginAttempt{
		IP:        ip,
		Username:  username,
		UserID:    userID,
		UserAgent: userAgent,
		Success:   success,
		RequestID: requestID,
	}
	if !success {
		attempt.FailureType = failureType
	}

	if err := s.db.Create(attempt).Error; err != nil {
		logrus.Errorf("记录登录尝试失败: %v", err)
		return err
	}

	// 更新用户的失败计数
	if userID > 0 {
		if success {
			// 登录成功，重置失败计数
			if err := s.ResetUserFailedCount(userID); err != nil {
				logrus.Errorf("重置用户失败计数错误: %v", err)
			}
		} else {
			// 登录失败，增加失败计数
			if err := s.incrementUserFailedCount(&user, ip, config); err != nil {
				logrus.Errorf("增加用户失败计数错误: %v", err)
			}
		}
	}

	// 更新IP登录统计
	if err := s.updateIPStats(ip, success, config); err != nil {
		logrus.Errorf("更新IP统计错误: %v", err)
	}

	return nil
}

// incrementUserFailedCount 增加用户失败计数并检查是否需要锁定
func (s *LoginProtectionService) incrementUserFailedCount(user *models.User, ip string, config *models.SecurityConfig) error {
	now := time.Now()
	maxAttempts := 5
	lockoutMinutes := 30

	if config != nil {
		maxAttempts = config.MaxLoginAttempts
		lockoutMinutes = config.LoginLockoutDuration
	}

	// 检查是否需要重置计数（距离上次失败超过锁定时间）
	if user.LastFailedLoginAt != nil {
		timeSinceLastFail := time.Since(*user.LastFailedLoginAt)
		if timeSinceLastFail > time.Duration(lockoutMinutes)*time.Minute {
			user.FailedLoginCount = 0
		}
	}

	user.FailedLoginCount++
	user.LastFailedLoginAt = &now
	user.LastFailedLoginIP = ip

	// 检查是否需要锁定账号
	if user.FailedLoginCount >= maxAttempts {
		lockUntil := now.Add(time.Duration(lockoutMinutes) * time.Minute)
		user.LockedUntil = &lockUntil
		logrus.Warnf("账号 %s 因连续 %d 次登录失败已被锁定至 %v",
			user.Username, user.FailedLoginCount, lockUntil)
	}

	return s.db.Model(user).Updates(map[string]interface{}{
		"failed_login_count":   user.FailedLoginCount,
		"last_failed_login_at": user.LastFailedLoginAt,
		"last_failed_login_ip": user.LastFailedLoginIP,
		"locked_until":         user.LockedUntil,
	}).Error
}

// updateIPStats 更新IP登录统计
func (s *LoginProtectionService) updateIPStats(ip string, success bool, config *models.SecurityConfig) error {
	var stats models.IPLoginStats
	err := s.db.Where("ip = ?", ip).First(&stats).Error

	now := time.Now()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新记录
		stats = models.IPLoginStats{
			IP:            ip,
			FirstSeenAt:   now,
			LastAttemptAt: &now,
			TotalAttempts: 1,
		}
		if success {
			stats.SuccessCount = 1
			stats.ConsecutiveFails = 0
			stats.LastSuccessAt = &now
		} else {
			stats.FailureCount = 1
			stats.ConsecutiveFails = 1
			stats.LastFailureAt = &now
		}
		return s.db.Create(&stats).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	stats.TotalAttempts++
	stats.LastAttemptAt = &now

	if success {
		stats.SuccessCount++
		stats.ConsecutiveFails = 0
		stats.LastSuccessAt = &now
	} else {
		stats.FailureCount++
		stats.ConsecutiveFails++
		stats.LastFailureAt = &now

		// 检查是否需要自动封禁
		if config != nil && config.AutoBlockEnabled {
			if stats.ConsecutiveFails >= config.AutoBlockThreshold {
				blockUntil := now.Add(time.Duration(config.AutoBlockDuration) * time.Minute)
				stats.BlockedUntil = &blockUntil
				stats.BlockCount++
				stats.RiskScore = 100 // 高风险
				logrus.Warnf("IP %s 因连续 %d 次登录失败已被自动封禁至 %v",
					ip, stats.ConsecutiveFails, blockUntil)
			}
		}
	}

	// 计算风险分数
	stats.RiskScore = s.calculateRiskScore(&stats)

	return s.db.Save(&stats).Error
}

// calculateRiskScore 计算IP风险分数 (0-100)
func (s *LoginProtectionService) calculateRiskScore(stats *models.IPLoginStats) int {
	if stats.TotalAttempts == 0 {
		return 0
	}

	score := 0

	// 失败率因素 (最高40分)
	failureRate := float64(stats.FailureCount) / float64(stats.TotalAttempts)
	score += int(failureRate * 40)

	// 连续失败因素 (最高30分)
	if stats.ConsecutiveFails >= 10 {
		score += 30
	} else if stats.ConsecutiveFails >= 5 {
		score += 20
	} else if stats.ConsecutiveFails >= 3 {
		score += 10
	}

	// 高频率因素 (最高30分) - 基于总尝试次数
	if stats.TotalAttempts >= 100 {
		score += 30
	} else if stats.TotalAttempts >= 50 {
		score += 20
	} else if stats.TotalAttempts >= 20 {
		score += 10
	}

	if score > 100 {
		score = 100
	}

	return score
}

// ResetUserFailedCount 重置用户失败计数
func (s *LoginProtectionService) ResetUserFailedCount(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"failed_login_count":   0,
		"last_failed_login_at": nil,
		"last_failed_login_ip": "",
		"locked_until":         nil,
	}).Error
}

// UnlockAccount 手动解锁账号
func (s *LoginProtectionService) UnlockAccount(username string, operatorID uint) error {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	if err := s.ResetUserFailedCount(user.ID); err != nil {
		return err
	}

	// 记录审计日志
	logrus.Infof("管理员 (ID: %d) 手动解锁了账号: %s", operatorID, username)

	return nil
}

// BlockIP 手动封禁IP
func (s *LoginProtectionService) BlockIP(ip string, reason string, durationMinutes int, operatorID uint) error {
	var expireAt *time.Time
	if durationMinutes > 0 {
		t := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
		expireAt = &t
	}

	blacklist := &models.IPBlacklist{
		IP:        ip,
		Reason:    reason,
		BlockType: "temporary",
		ExpireAt:  expireAt,
		CreatedBy: fmt.Sprintf("user:%d", operatorID),
		Enabled:   true,
	}

	if durationMinutes == 0 {
		blacklist.BlockType = "permanent"
	}

	// 使用 upsert
	return s.db.Where(models.IPBlacklist{IP: ip}).
		Assign(blacklist).
		FirstOrCreate(blacklist).Error
}

// UnblockIP 解除IP封禁
func (s *LoginProtectionService) UnblockIP(ip string, operatorID uint) error {
	// 从黑名单中移除
	if err := s.db.Model(&models.IPBlacklist{}).
		Where("ip = ?", ip).
		Update("enabled", false).Error; err != nil {
		return err
	}

	// 重置IP统计中的封禁状态
	if err := s.db.Model(&models.IPLoginStats{}).
		Where("ip = ?", ip).
		Updates(map[string]interface{}{
			"blocked_until":     nil,
			"consecutive_fails": 0,
			"risk_score":        0,
		}).Error; err != nil {
		return err
	}

	logrus.Infof("管理员 (ID: %d) 解除了IP封禁: %s", operatorID, ip)
	return nil
}

// GetIPStats 获取IP统计信息
func (s *LoginProtectionService) GetIPStats(ip string) (*models.IPLoginStats, error) {
	var stats models.IPLoginStats
	if err := s.db.Where("ip = ?", ip).First(&stats).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

// GetBlockedIPs 获取被封禁的IP列表
func (s *LoginProtectionService) GetBlockedIPs(page, pageSize int) ([]models.IPBlacklist, int64, error) {
	var items []models.IPBlacklist
	var total int64

	query := s.db.Model(&models.IPBlacklist{}).Where("enabled = ?", true)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// GetLockedAccounts 获取被锁定的账号列表
func (s *LoginProtectionService) GetLockedAccounts(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := s.db.Model(&models.User{}).Where("locked_until IS NOT NULL AND locked_until > ?", time.Now())

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select("id, username, email, locked_until, failed_login_count, last_failed_login_at, last_failed_login_ip").
		Order("locked_until DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetLoginAttempts 获取登录尝试记录
func (s *LoginProtectionService) GetLoginAttempts(filter LoginAttemptFilter, page, pageSize int) ([]models.LoginAttempt, int64, error) {
	var attempts []models.LoginAttempt
	var total int64

	query := s.db.Model(&models.LoginAttempt{})

	if filter.IP != "" {
		query = query.Where("ip = ?", filter.IP)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.Success != nil {
		query = query.Where("success = ?", *filter.Success)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&attempts).Error; err != nil {
		return nil, 0, err
	}

	return attempts, total, nil
}

// LoginAttemptFilter 登录尝试查询过滤器
type LoginAttemptFilter struct {
	IP        string
	Username  string
	Success   *bool
	StartTime time.Time
	EndTime   time.Time
}

// GetIPLoginStatsList 获取IP登录统计列表
func (s *LoginProtectionService) GetIPLoginStatsList(filter IPStatsFilter, page, pageSize int) ([]models.IPLoginStats, int64, error) {
	var stats []models.IPLoginStats
	var total int64

	query := s.db.Model(&models.IPLoginStats{})

	if filter.OnlyBlocked {
		query = query.Where("blocked_until IS NOT NULL AND blocked_until > ?", time.Now())
	}
	if filter.MinRiskScore > 0 {
		query = query.Where("risk_score >= ?", filter.MinRiskScore)
	}
	if filter.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filter.IP+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	orderBy := "last_attempt_at DESC"
	if filter.OrderByRisk {
		orderBy = "risk_score DESC"
	}

	if err := query.Order(orderBy).Offset(offset).Limit(pageSize).Find(&stats).Error; err != nil {
		return nil, 0, err
	}

	return stats, total, nil
}

// IPStatsFilter IP统计查询过滤器
type IPStatsFilter struct {
	IP           string
	OnlyBlocked  bool
	MinRiskScore int
	OrderByRisk  bool
}

// GetLoginStatsSummary 获取登录统计摘要
func (s *LoginProtectionService) GetLoginStatsSummary(hours int) (*LoginStatsSummary, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var summary LoginStatsSummary

	// 总登录尝试次数
	s.db.Model(&models.LoginAttempt{}).Where("created_at >= ?", since).Count(&summary.TotalAttempts)

	// 成功登录次数
	s.db.Model(&models.LoginAttempt{}).Where("created_at >= ? AND success = ?", since, true).Count(&summary.SuccessfulLogins)

	// 失败登录次数
	s.db.Model(&models.LoginAttempt{}).Where("created_at >= ? AND success = ?", since, false).Count(&summary.FailedLogins)

	// 当前锁定账号数
	s.db.Model(&models.User{}).Where("locked_until IS NOT NULL AND locked_until > ?", time.Now()).Count(&summary.LockedAccounts)

	// 当前封禁IP数
	s.db.Model(&models.IPBlacklist{}).Where("enabled = ? AND (expire_at IS NULL OR expire_at > ?)", true, time.Now()).Count(&summary.BlockedIPs)

	// 自动封禁IP数
	s.db.Model(&models.IPLoginStats{}).Where("blocked_until IS NOT NULL AND blocked_until > ?", time.Now()).Count(&summary.AutoBlockedIPs)

	// 高风险IP数
	s.db.Model(&models.IPLoginStats{}).Where("risk_score >= ?", 70).Count(&summary.HighRiskIPs)

	// 独立IP数
	s.db.Model(&models.LoginAttempt{}).Where("created_at >= ?", since).Distinct("ip").Count(&summary.UniqueIPs)

	// 独立用户数
	s.db.Model(&models.LoginAttempt{}).Where("created_at >= ?", since).Distinct("username").Count(&summary.UniqueUsers)

	return &summary, nil
}

// LoginStatsSummary 登录统计摘要
type LoginStatsSummary struct {
	TotalAttempts    int64 `json:"total_attempts"`
	SuccessfulLogins int64 `json:"successful_logins"`
	FailedLogins     int64 `json:"failed_logins"`
	LockedAccounts   int64 `json:"locked_accounts"`
	BlockedIPs       int64 `json:"blocked_ips"`
	AutoBlockedIPs   int64 `json:"auto_blocked_ips"`
	HighRiskIPs      int64 `json:"high_risk_ips"`
	UniqueIPs        int64 `json:"unique_ips"`
	UniqueUsers      int64 `json:"unique_users"`
}

// CleanupExpiredRecords 清理过期记录
func (s *LoginProtectionService) CleanupExpiredRecords(retentionDays int) error {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	// 清理过期的登录尝试记录
	if err := s.db.Where("created_at < ?", cutoff).Delete(&models.LoginAttempt{}).Error; err != nil {
		return fmt.Errorf("清理登录尝试记录失败: %w", err)
	}

	// 清理过期的IP黑名单
	if err := s.db.Where("expire_at IS NOT NULL AND expire_at < ?", time.Now()).Delete(&models.IPBlacklist{}).Error; err != nil {
		return fmt.Errorf("清理IP黑名单失败: %w", err)
	}

	logrus.Infof("已清理 %d 天前的登录记录", retentionDays)
	return nil
}

// GetSuspiciousIPs 获取疑似攻击IP列表
func (s *LoginProtectionService) GetSuspiciousIPs(page, pageSize int, minRiskScore int) ([]models.IPLoginStats, int64, error) {
	var stats []models.IPLoginStats
	var total int64

	if minRiskScore <= 0 {
		minRiskScore = 50 // 默认风险分数阈值
	}

	query := s.db.Model(&models.IPLoginStats{}).Where("risk_score >= ?", minRiskScore)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("risk_score DESC, last_attempt_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&stats).Error; err != nil {
		return nil, 0, err
	}

	return stats, total, nil
}

// GetFailureTypeStats 获取登录失败类型统计
func (s *LoginProtectionService) GetFailureTypeStats(hours int) (map[string]int64, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	type Result struct {
		FailureType string
		Count       int64
	}

	var results []Result
	if err := s.db.Model(&models.LoginAttempt{}).
		Select("failure_type, count(*) as count").
		Where("created_at >= ? AND success = ? AND failure_type != ''", since, false).
		Group("failure_type").
		Find(&results).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		failureType := r.FailureType
		if failureType == "" {
			failureType = "unknown"
		}
		stats[failureType] = r.Count
	}

	return stats, nil
}

// IsIPInWhitelist 检查IP是否在白名单中
func (s *LoginProtectionService) IsIPInWhitelist(ip string) (bool, error) {
	var whitelist models.IPWhitelist
	if err := s.db.Where("ip = ? AND enabled = ?", ip, true).First(&whitelist).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetRecentLoginsByUser 获取用户最近的登录记录
func (s *LoginProtectionService) GetRecentLoginsByUser(username string, limit int) ([]models.LoginAttempt, error) {
	var attempts []models.LoginAttempt
	if err := s.db.Where("username = ?", username).
		Order("created_at DESC").
		Limit(limit).
		Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

// GetRecentLoginsByIP 获取IP最近的登录记录
func (s *LoginProtectionService) GetRecentLoginsByIP(ip string, limit int) ([]models.LoginAttempt, error) {
	var attempts []models.LoginAttempt
	if err := s.db.Where("ip = ?", ip).
		Order("created_at DESC").
		Limit(limit).
		Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

// DetectBruteForcePattern 检测暴力破解模式
func (s *LoginProtectionService) DetectBruteForcePattern(ip string, windowMinutes int) (bool, string, error) {
	since := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	// 统计时间窗口内的失败次数
	var failCount int64
	if err := s.db.Model(&models.LoginAttempt{}).
		Where("ip = ? AND created_at >= ? AND success = ?", ip, since, false).
		Count(&failCount).Error; err != nil {
		return false, "", err
	}

	// 统计尝试的不同用户名数量
	var uniqueUsernames int64
	if err := s.db.Model(&models.LoginAttempt{}).
		Where("ip = ? AND created_at >= ?", ip, since).
		Distinct("username").
		Count(&uniqueUsernames).Error; err != nil {
		return false, "", err
	}

	// 检测模式
	var patterns []string

	// 模式1: 高频失败
	if failCount >= 10 {
		patterns = append(patterns, fmt.Sprintf("高频失败尝试(%d次/%d分钟)", failCount, windowMinutes))
	}

	// 模式2: 多用户名尝试（可能是用户名枚举）
	if uniqueUsernames >= 5 {
		patterns = append(patterns, fmt.Sprintf("多用户名尝试(%d个不同用户名)", uniqueUsernames))
	}

	if len(patterns) > 0 {
		return true, strings.Join(patterns, "; "), nil
	}

	return false, "", nil
}
