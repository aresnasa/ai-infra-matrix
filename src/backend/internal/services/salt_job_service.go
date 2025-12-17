package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SaltJobService Salt作业持久化服务
type SaltJobService struct {
	db          *gorm.DB
	cache       *redis.Client
	config      *models.SaltJobConfig
	configMu    sync.RWMutex
	cleanupStop chan struct{}
	cleanupOnce sync.Once
}

// NewSaltJobService 创建Salt作业服务
func NewSaltJobService(db *gorm.DB, cache *redis.Client) *SaltJobService {
	s := &SaltJobService{
		db:          db,
		cache:       cache,
		cleanupStop: make(chan struct{}),
	}
	// 加载或初始化配置
	s.loadOrCreateConfig()
	return s
}

// loadOrCreateConfig 加载或创建默认配置
func (s *SaltJobService) loadOrCreateConfig() {
	var config models.SaltJobConfig
	result := s.db.First(&config)
	if result.Error != nil {
		// 创建默认配置
		config = models.SaltJobConfig{
			RetentionDays:        30,
			AutoCleanupEnabled:   true,
			CleanupIntervalHours: 24,
			MaxJobsCount:         10000,
			RedisCacheDays:       7,
		}
		if err := s.db.Create(&config).Error; err != nil {
			log.Printf("[SaltJobService] 创建默认配置失败: %v", err)
		} else {
			log.Printf("[SaltJobService] 已创建默认配置: 保留%d天, 自动清理=%v", config.RetentionDays, config.AutoCleanupEnabled)
		}
	}
	s.configMu.Lock()
	s.config = &config
	s.configMu.Unlock()
}

// GetConfig 获取当前配置
func (s *SaltJobService) GetConfig() *models.SaltJobConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

// UpdateConfig 更新配置
func (s *SaltJobService) UpdateConfig(ctx context.Context, req models.SaltJobConfigRequest) (*models.SaltJobConfig, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	if s.config == nil {
		return nil, fmt.Errorf("配置未初始化")
	}

	// 更新配置字段
	if req.RetentionDays != nil && *req.RetentionDays > 0 {
		s.config.RetentionDays = *req.RetentionDays
	}
	if req.AutoCleanupEnabled != nil {
		s.config.AutoCleanupEnabled = *req.AutoCleanupEnabled
	}
	if req.CleanupIntervalHours != nil && *req.CleanupIntervalHours > 0 {
		s.config.CleanupIntervalHours = *req.CleanupIntervalHours
	}
	if req.MaxJobsCount != nil && *req.MaxJobsCount > 0 {
		s.config.MaxJobsCount = *req.MaxJobsCount
	}
	if req.RedisCacheDays != nil && *req.RedisCacheDays > 0 {
		s.config.RedisCacheDays = *req.RedisCacheDays
	}

	// 保存到数据库
	if err := s.db.WithContext(ctx).Save(s.config).Error; err != nil {
		return nil, err
	}

	log.Printf("[SaltJobService] 配置已更新: 保留%d天, 自动清理=%v, 间隔%d小时",
		s.config.RetentionDays, s.config.AutoCleanupEnabled, s.config.CleanupIntervalHours)

	return s.config, nil
}

// StartCleanupScheduler 启动定时清理任务
func (s *SaltJobService) StartCleanupScheduler() {
	s.cleanupOnce.Do(func() {
		go s.cleanupLoop()
		log.Printf("[SaltJobService] 定时清理任务已启动")
	})
}

// StopCleanupScheduler 停止定时清理任务
func (s *SaltJobService) StopCleanupScheduler() {
	select {
	case <-s.cleanupStop:
		// 已经关闭
	default:
		close(s.cleanupStop)
	}
}

// cleanupLoop 清理循环
func (s *SaltJobService) cleanupLoop() {
	// 首次启动时检查是否需要清理
	s.checkAndCleanup()

	for {
		config := s.GetConfig()
		interval := time.Duration(config.CleanupIntervalHours) * time.Hour
		if interval < time.Hour {
			interval = time.Hour // 最小间隔1小时
		}

		select {
		case <-time.After(interval):
			s.checkAndCleanup()
		case <-s.cleanupStop:
			log.Printf("[SaltJobService] 定时清理任务已停止")
			return
		}
	}
}

// checkAndCleanup 检查并执行清理
func (s *SaltJobService) checkAndCleanup() {
	config := s.GetConfig()
	if !config.AutoCleanupEnabled {
		return
	}

	ctx := context.Background()

	// 检查是否需要清理
	var count int64
	s.db.Model(&models.SaltJob{}).Count(&count)

	shouldCleanup := false
	reason := ""

	// 条件1: 作业数量超过最大限制
	if count > int64(config.MaxJobsCount) {
		shouldCleanup = true
		reason = fmt.Sprintf("作业数量(%d)超过最大限制(%d)", count, config.MaxJobsCount)
	}

	// 条件2: 存在超过保留天数的作业
	cutoffTime := time.Now().AddDate(0, 0, -config.RetentionDays)
	var oldCount int64
	s.db.Model(&models.SaltJob{}).Where("start_time < ?", cutoffTime).Count(&oldCount)
	if oldCount > 0 {
		shouldCleanup = true
		reason = fmt.Sprintf("存在%d条超过%d天的旧作业", oldCount, config.RetentionDays)
	}

	if shouldCleanup {
		log.Printf("[SaltJobService] 开始清理: %s", reason)
		cleaned, err := s.CleanupOldJobs(ctx, config.RetentionDays)
		if err != nil {
			log.Printf("[SaltJobService] 清理失败: %v", err)
		} else {
			// 更新统计
			s.configMu.Lock()
			now := time.Now()
			s.config.LastCleanupAt = &now
			s.config.CleanedCount += cleaned
			s.db.Save(s.config)
			s.configMu.Unlock()
		}
	}
}

// TriggerCleanup 手动触发清理
func (s *SaltJobService) TriggerCleanup(ctx context.Context) (int64, error) {
	config := s.GetConfig()
	cleaned, err := s.CleanupOldJobs(ctx, config.RetentionDays)
	if err != nil {
		return 0, err
	}

	// 更新统计
	s.configMu.Lock()
	now := time.Now()
	s.config.LastCleanupAt = &now
	s.config.CleanedCount += cleaned
	s.db.Save(s.config)
	s.configMu.Unlock()

	return cleaned, nil
}

// GetStats 获取作业统计信息
func (s *SaltJobService) GetStats(ctx context.Context) (*models.SaltJobStats, error) {
	stats := &models.SaltJobStats{}

	// 总数
	s.db.WithContext(ctx).Model(&models.SaltJob{}).Count(&stats.TotalJobs)

	// 各状态数量
	s.db.WithContext(ctx).Model(&models.SaltJob{}).Where("status = ?", "running").Count(&stats.RunningJobs)
	s.db.WithContext(ctx).Model(&models.SaltJob{}).Where("status = ?", "completed").Count(&stats.CompletedJobs)
	s.db.WithContext(ctx).Model(&models.SaltJob{}).Where("status = ?", "failed").Count(&stats.FailedJobs)

	// 最早和最新作业时间
	var oldest, newest models.SaltJob
	if err := s.db.WithContext(ctx).Order("start_time ASC").First(&oldest).Error; err == nil {
		stats.OldestJobTime = &oldest.StartTime
	}
	if err := s.db.WithContext(ctx).Order("start_time DESC").First(&newest).Error; err == nil {
		stats.NewestJobTime = &newest.StartTime
	}

	// 预估存储大小（简单估算）
	avgRowSize := int64(2048) // 假设每条记录约2KB
	estimatedBytes := stats.TotalJobs * avgRowSize
	if estimatedBytes < 1024 {
		stats.StorageEstimate = fmt.Sprintf("%d B", estimatedBytes)
	} else if estimatedBytes < 1024*1024 {
		stats.StorageEstimate = fmt.Sprintf("%.2f KB", float64(estimatedBytes)/1024)
	} else if estimatedBytes < 1024*1024*1024 {
		stats.StorageEstimate = fmt.Sprintf("%.2f MB", float64(estimatedBytes)/(1024*1024))
	} else {
		stats.StorageEstimate = fmt.Sprintf("%.2f GB", float64(estimatedBytes)/(1024*1024*1024))
	}

	return stats, nil
}

// SaveJob 保存作业到数据库和Redis
func (s *SaltJobService) SaveJob(ctx context.Context, job *models.SaltJob) error {
	// 保存到数据库
	result := s.db.WithContext(ctx).Create(job)
	if result.Error != nil {
		// 如果是唯一索引冲突，尝试更新
		if strings.Contains(result.Error.Error(), "duplicate") || strings.Contains(result.Error.Error(), "UNIQUE") {
			result = s.db.WithContext(ctx).Where("jid = ?", job.JID).Updates(job)
			if result.Error != nil {
				log.Printf("[SaltJobService] 更新作业失败: %v", result.Error)
				return result.Error
			}
		} else {
			log.Printf("[SaltJobService] 保存作业失败: %v", result.Error)
			return result.Error
		}
	}

	// 同时保存到Redis（用于快速查询和JID-TaskID映射）
	if s.cache != nil {
		config := s.GetConfig()
		ttl := time.Duration(config.RedisCacheDays) * 24 * time.Hour
		if ttl < 24*time.Hour {
			ttl = 7 * 24 * time.Hour // 默认7天
		}

		// JID -> TaskID 映射（无论是否有TaskID都保存）
		if job.TaskID != "" {
			jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", job.JID)
			s.cache.Set(ctx, jidToTaskKey, job.TaskID, ttl)

			// TaskID -> JID 映射
			taskToJidKey := fmt.Sprintf("saltstack:task_to_jid:%s", job.TaskID)
			s.cache.Set(ctx, taskToJidKey, job.JID, ttl)
		}

		// 作业详情缓存
		jobJSON, _ := json.Marshal(map[string]interface{}{
			"jid":        job.JID,
			"task_id":    job.TaskID,
			"function":   job.Function,
			"target":     job.Target,
			"arguments":  job.Arguments,
			"user":       job.User,
			"status":     job.Status,
			"start_time": job.StartTime.Format(time.RFC3339),
		})
		jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", job.JID)
		s.cache.Set(ctx, jobDetailKey, string(jobJSON), ttl)

		// 添加到最近作业列表
		recentJobsKey := "saltstack:recent_jobs"
		s.cache.LPush(ctx, recentJobsKey, job.JID)
		s.cache.LTrim(ctx, recentJobsKey, 0, 99) // 保留最近100条

		log.Printf("[SaltJobService] 作业已保存并缓存: JID=%s, TaskID=%s, TTL=%v", job.JID, job.TaskID, ttl)
	}

	return nil
}

// UpdateJobStatus 更新作业状态
func (s *SaltJobService) UpdateJobStatus(ctx context.Context, jid string, status string, result map[string]interface{}) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if result != nil {
		resultJSON, _ := json.Marshal(result)
		updates["result"] = resultJSON
	}

	if status == "completed" || status == "failed" || status == "timeout" {
		now := time.Now()
		updates["end_time"] = now
	}

	return s.db.WithContext(ctx).Model(&models.SaltJob{}).Where("jid = ?", jid).Updates(updates).Error
}

// UpdateJobResult 更新作业执行结果（完成回调）
func (s *SaltJobService) UpdateJobResult(ctx context.Context, jid string, status string, result map[string]interface{}, successCount, failedCount int, durationMs int64, endTime *time.Time) error {
	updates := map[string]interface{}{
		"status":        status,
		"success_count": successCount,
		"failed_count":  failedCount,
		"duration":      durationMs,
		"updated_at":    time.Now(),
	}

	if result != nil {
		resultJSON, _ := json.Marshal(result)
		updates["result"] = resultJSON
	}

	if endTime != nil {
		updates["end_time"] = *endTime
	}

	// 更新数据库
	if err := s.db.WithContext(ctx).Model(&models.SaltJob{}).Where("jid = ?", jid).Updates(updates).Error; err != nil {
		return err
	}

	// 同时更新 Redis 缓存中的状态
	if s.cache != nil {
		jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
		// 获取现有缓存
		if jobInfoJSON, err := s.cache.Get(ctx, jobDetailKey).Result(); err == nil {
			var jobInfo map[string]interface{}
			if json.Unmarshal([]byte(jobInfoJSON), &jobInfo) == nil {
				jobInfo["status"] = status
				jobInfo["success_count"] = successCount
				jobInfo["failed_count"] = failedCount
				jobInfo["duration_ms"] = durationMs
				if endTime != nil {
					jobInfo["end_time"] = endTime.Format(time.RFC3339)
				}
				// 更新缓存
				if newJSON, err := json.Marshal(jobInfo); err == nil {
					config := s.GetConfig()
					ttl := time.Duration(config.RedisCacheDays) * 24 * time.Hour
					s.cache.Set(ctx, jobDetailKey, string(newJSON), ttl)
				}
			}
		}
	}

	log.Printf("[SaltJobService] 作业结果已更新: JID=%s, 状态=%s, 成功=%d, 失败=%d, 时长=%dms",
		jid, status, successCount, failedCount, durationMs)
	return nil
}

// GetJobByJID 通过JID获取作业
func (s *SaltJobService) GetJobByJID(ctx context.Context, jid string) (*models.SaltJob, error) {
	var job models.SaltJob
	result := s.db.WithContext(ctx).Where("jid = ?", jid).First(&job)
	if result.Error != nil {
		return nil, result.Error
	}
	return &job, nil
}

// GetJobByTaskID 通过TaskID获取作业
func (s *SaltJobService) GetJobByTaskID(ctx context.Context, taskID string) (*models.SaltJob, error) {
	// 先尝试从Redis获取JID
	if s.cache != nil {
		taskToJidKey := fmt.Sprintf("saltstack:task_to_jid:%s", taskID)
		if jid, err := s.cache.Get(ctx, taskToJidKey).Result(); err == nil && jid != "" {
			return s.GetJobByJID(ctx, jid)
		}
	}

	// 从数据库查询
	var job models.SaltJob
	result := s.db.WithContext(ctx).Where("task_id = ?", taskID).First(&job)
	if result.Error != nil {
		return nil, result.Error
	}

	// 写入Redis缓存
	if s.cache != nil {
		ttl := 7 * 24 * time.Hour
		taskToJidKey := fmt.Sprintf("saltstack:task_to_jid:%s", taskID)
		s.cache.Set(ctx, taskToJidKey, job.JID, ttl)
		jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", job.JID)
		s.cache.Set(ctx, jidToTaskKey, taskID, ttl)
	}

	return &job, nil
}

// ListJobs 查询作业列表
func (s *SaltJobService) ListJobs(ctx context.Context, req models.SaltJobListRequest) (*models.SaltJobListResponse, error) {
	// 默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	query := s.db.WithContext(ctx).Model(&models.SaltJob{})

	// 应用过滤条件
	if req.TaskID != "" {
		query = query.Where("task_id = ?", req.TaskID)
	}
	if req.JID != "" {
		query = query.Where("jid = ?", req.JID)
	}
	if req.Function != "" {
		query = query.Where("function LIKE ?", "%"+req.Function+"%")
	}
	if req.Target != "" {
		query = query.Where("target LIKE ?", "%"+req.Target+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.User != "" {
		query = query.Where("user = ?", req.User)
	}
	if req.StartFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.StartFrom); err == nil {
			query = query.Where("start_time >= ?", t)
		}
	}
	if req.StartTo != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTo); err == nil {
			query = query.Where("start_time <= ?", t)
		}
	}
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("task_id LIKE ? OR jid LIKE ? OR function LIKE ?", keyword, keyword, keyword)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var jobs []models.SaltJob
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("start_time DESC").Offset(offset).Limit(req.PageSize).Find(&jobs).Error; err != nil {
		return nil, err
	}

	return &models.SaltJobListResponse{
		Total: total,
		Page:  req.Page,
		Size:  req.PageSize,
		Data:  jobs,
	}, nil
}

// GetRecentJobs 获取最近的作业（优先从Redis，回退到数据库）
func (s *SaltJobService) GetRecentJobs(ctx context.Context, limit int) ([]models.SaltJob, error) {
	if limit <= 0 {
		limit = 20
	}

	// 从数据库获取最近的作业
	var jobs []models.SaltJob
	if err := s.db.WithContext(ctx).Order("start_time DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetTaskIDByJID 通过JID获取TaskID（优先Redis，回退数据库）
func (s *SaltJobService) GetTaskIDByJID(ctx context.Context, jid string) (string, error) {
	// 优先从Redis获取
	if s.cache != nil {
		jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", jid)
		if taskID, err := s.cache.Get(ctx, jidToTaskKey).Result(); err == nil && taskID != "" {
			return taskID, nil
		}
	}

	// 从数据库查询
	var job models.SaltJob
	result := s.db.WithContext(ctx).Select("task_id").Where("jid = ?", jid).First(&job)
	if result.Error != nil {
		return "", result.Error
	}

	// 写入Redis缓存
	if s.cache != nil && job.TaskID != "" {
		ttl := 7 * 24 * time.Hour
		jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", jid)
		s.cache.Set(ctx, jidToTaskKey, job.TaskID, ttl)
	}

	return job.TaskID, nil
}

// CleanupOldJobs 清理旧作业记录（保留指定天数）
func (s *SaltJobService) CleanupOldJobs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90 // 默认保留90天
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.WithContext(ctx).Where("start_time < ?", cutoffTime).Delete(&models.SaltJob{})
	if result.Error != nil {
		return 0, result.Error
	}

	log.Printf("[SaltJobService] 清理了 %d 条超过 %d 天的旧作业记录", result.RowsAffected, retentionDays)
	return result.RowsAffected, nil
}
