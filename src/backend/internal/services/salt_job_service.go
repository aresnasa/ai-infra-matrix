package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 监控系统任务的黑名单 - 这些任务自动被过滤
var MonitoringFunctionBlacklist = map[string]bool{
	"status.cpuload":        true,
	"status.meminfo":        true,
	"status.cpuinfo":        true,
	"status.diskusage":      true,
	"status.netstats":       true,
	"status.uptime":         true,
	"status.loadavg":        true,
	"status.loadavg5":       true,
	"status.loadavg15":      true,
	"runner.manage.status":  true,
	"test.ping":             true,
	"grains.items":          true,
	"saltutil.sync_all":     true,
	"saltutil.sync_grains":  true,
	"saltutil.sync_modules": true,
	"saltutil.refresh":      true,
	"cmd.run_all":           true,
	"test.echo":             true,
}

// IsMonitoringTask 判断是否是监控系统的任务
func IsMonitoringTask(function string) bool {
	if MonitoringFunctionBlacklist[function] {
		return true
	}
	// 检查前缀
	if len(function) > 0 {
		if function[0:len("status.")] == "status." {
			return true
		}
		if function[0:len("runner.")] == "runner." {
			return true
		}
	}
	return false
}

// SaltJobService Salt作业持久化服务
type SaltJobService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *models.SaltJobConfig
	configMu    sync.RWMutex
	cleanupOnce sync.Once
	cleanupDone chan struct{}
}

var (
	saltJobServiceInstance *SaltJobService
	saltJobServiceOnce     sync.Once
)

// NewSaltJobService 创建Salt作业服务（单例）
func NewSaltJobService(db *gorm.DB, redisClient *redis.Client) *SaltJobService {
	saltJobServiceOnce.Do(func() {
		saltJobServiceInstance = &SaltJobService{
			db:          db,
			redis:       redisClient,
			cleanupDone: make(chan struct{}),
		}
		// 自动迁移数据库表
		if err := db.AutoMigrate(&models.SaltJobHistory{}, &models.SaltJobConfig{}); err != nil {
			log.Printf("[SaltJobService] 自动迁移失败: %v", err)
		}
		// 加载或初始化配置
		saltJobServiceInstance.loadOrCreateConfig()
		// 启动后台清理任务
		saltJobServiceInstance.startCleanupWorker()
	})
	return saltJobServiceInstance
}

// GetSaltJobService 获取Salt作业服务实例
func GetSaltJobService() *SaltJobService {
	return saltJobServiceInstance
}

// loadOrCreateConfig 加载或创建默认配置
func (s *SaltJobService) loadOrCreateConfig() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	var config models.SaltJobConfig
	result := s.db.First(&config)
	if result.Error == gorm.ErrRecordNotFound {
		// 创建默认配置
		config = models.SaltJobConfig{
			MaxRetentionDays:    30,
			MaxRecords:          10000,
			CleanupEnabled:      true,
			CleanupIntervalHour: 24,
		}
		s.db.Create(&config)
	}
	s.config = &config
}

// GetConfig 获取当前配置
func (s *SaltJobService) GetConfig() *models.SaltJobConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

// UpdateConfig 更新配置
func (s *SaltJobService) UpdateConfig(config *models.SaltJobConfig) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	if err := s.db.Save(config).Error; err != nil {
		return err
	}
	s.config = config
	return nil
}

// CreateJob 创建作业记录
func (s *SaltJobService) CreateJob(ctx context.Context, job *models.SaltJobHistory) error {
	// 设置默认状态
	if job.Status == "" {
		job.Status = "running"
	}
	if job.StartTime.IsZero() {
		job.StartTime = time.Now()
	}

	// 保存到数据库
	if err := s.db.Create(job).Error; err != nil {
		log.Printf("[SaltJobService] 创建作业记录失败: %v", err)
		return err
	}

	// 同时更新 Redis 缓存（用于快速查询）
	s.cacheJobToRedis(ctx, job)

	log.Printf("[SaltJobService] 作业记录已创建: JID=%s, TaskID=%s", job.JID, job.TaskID)
	return nil
}

// UpdateJobStatus 更新作业状态（状态机触发）
func (s *SaltJobService) UpdateJobStatus(ctx context.Context, jid string, update *models.SaltJobUpdateRequest) error {
	var job models.SaltJobHistory
	if err := s.db.Where("jid = ?", jid).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[SaltJobService] 作业不存在，尝试从 Redis 恢复: JID=%s", jid)
			// 尝试从 Redis 获取并创建
			if cachedJob := s.getJobFromRedis(ctx, jid); cachedJob != nil {
				job = *cachedJob
				s.db.Create(&job)
			} else {
				return fmt.Errorf("job not found: %s", jid)
			}
		} else {
			return err
		}
	}

	// 状态机校验：只允许 running -> completed/failed/timeout
	if job.Status != "running" && update.Status != "" {
		log.Printf("[SaltJobService] 警告：尝试更新非运行中的作业状态: JID=%s, 当前状态=%s", jid, job.Status)
		// 允许重复更新，但记录日志
	}

	// 更新字段
	updates := map[string]interface{}{}
	if update.Status != "" {
		updates["status"] = update.Status
	}
	if update.ReturnCode != 0 {
		updates["return_code"] = update.ReturnCode
	}
	if update.SuccessCount > 0 {
		updates["success_count"] = update.SuccessCount
	}
	if update.FailedCount > 0 {
		updates["failed_count"] = update.FailedCount
	}
	if update.ErrorMessage != "" {
		updates["error_message"] = update.ErrorMessage
	}
	if update.Result != nil {
		resultJSON, _ := json.Marshal(update.Result)
		updates["result"] = string(resultJSON)
	}

	// 如果状态变为终态，设置结束时间和持续时间
	if update.Status == "completed" || update.Status == "failed" || update.Status == "timeout" {
		now := time.Now()
		updates["end_time"] = now
		updates["duration"] = now.Sub(job.StartTime).Milliseconds()
	}

	if err := s.db.Model(&job).Updates(updates).Error; err != nil {
		return err
	}

	// 更新 Redis 缓存
	job.Status = update.Status
	if endTime, ok := updates["end_time"].(time.Time); ok {
		job.EndTime = &endTime
	}
	if duration, ok := updates["duration"].(int64); ok {
		job.Duration = duration
	}
	s.cacheJobToRedis(ctx, &job)

	log.Printf("[SaltJobService] 作业状态已更新: JID=%s, Status=%s, Duration=%dms", jid, update.Status, job.Duration)
	return nil
}

// CompleteJob 完成作业（便捷方法）
func (s *SaltJobService) CompleteJob(ctx context.Context, jid string, result map[string]interface{}, successCount, failedCount int) error {
	return s.UpdateJobStatus(ctx, jid, &models.SaltJobUpdateRequest{
		Status:       "completed",
		Result:       result,
		SuccessCount: successCount,
		FailedCount:  failedCount,
	})
}

// FailJob 标记作业失败
func (s *SaltJobService) FailJob(ctx context.Context, jid string, errorMessage string) error {
	return s.UpdateJobStatus(ctx, jid, &models.SaltJobUpdateRequest{
		Status:       "failed",
		ErrorMessage: errorMessage,
	})
}

// TimeoutJob 标记作业超时
func (s *SaltJobService) TimeoutJob(ctx context.Context, jid string) error {
	return s.UpdateJobStatus(ctx, jid, &models.SaltJobUpdateRequest{
		Status: "timeout",
	})
}

// GetJobByJID 通过 JID 获取作业
func (s *SaltJobService) GetJobByJID(ctx context.Context, jid string) (*models.SaltJobHistory, error) {
	// 先从 Redis 缓存查询
	if job := s.getJobFromRedis(ctx, jid); job != nil {
		return job, nil
	}

	// 从数据库查询
	var job models.SaltJobHistory
	if err := s.db.Where("jid = ?", jid).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// GetJobByTaskID 通过 TaskID 获取作业
func (s *SaltJobService) GetJobByTaskID(ctx context.Context, taskID string) (*models.SaltJobHistory, error) {
	// 先从 Redis 获取 TaskID -> JID 映射
	jidKey := fmt.Sprintf("saltstack:task_to_jid:%s", taskID)
	if jid, err := s.redis.Get(ctx, jidKey).Result(); err == nil && jid != "" {
		return s.GetJobByJID(ctx, jid)
	}

	// 从数据库查询
	var job models.SaltJobHistory
	if err := s.db.Where("task_id = ?", taskID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ListJobs 分页查询作业列表
func (s *SaltJobService) ListJobs(ctx context.Context, params *models.SaltJobQueryParams) (*models.SaltJobListResponse, error) {
	query := s.db.Model(&models.SaltJobHistory{})

	// 默认过滤掉监控相关的任务，只展示用户发起的作业
	query = query.Where("status NOT IN ('monitoring', 'system')")

	// 使用黑名单过滤监控函数
	monitoringFunctions := []string{
		"status.cpuload", "status.meminfo", "status.cpuinfo", "status.diskusage",
		"status.netstats", "status.uptime", "status.loadavg", "status.loadavg5", "status.loadavg15",
		"runner.manage.status",
		"test.ping",
		"grains.items",
		"saltutil.sync_all", "saltutil.sync_grains", "saltutil.sync_modules", "saltutil.refresh",
		"cmd.run_all", "test.echo",
	}
	query = query.Where("function NOT IN ?", monitoringFunctions)
	query = query.Where("function NOT LIKE 'status.%'")
	query = query.Where("function NOT LIKE 'runner.%'")

	// 应用过滤条件
	if params.TaskID != "" {
		query = query.Where("task_id = ?", params.TaskID)
	}
	if params.JID != "" {
		query = query.Where("jid = ?", params.JID)
	}
	if params.Function != "" {
		query = query.Where("function LIKE ?", "%"+params.Function+"%")
	}
	if params.Target != "" {
		query = query.Where("target LIKE ?", "%"+params.Target+"%")
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.User != "" {
		query = query.Where("user = ?", params.User)
	}

	// 统计总数
	var total int64
	query.Count(&total)

	// 排序
	orderBy := params.SortBy
	if params.SortDesc {
		orderBy += " DESC"
	}
	query = query.Order(orderBy)

	// 分页
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize
	query = query.Offset(offset).Limit(params.PageSize)

	var jobs []models.SaltJobHistory
	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}

	return &models.SaltJobListResponse{
		Total: total,
		Page:  params.Page,
		Size:  params.PageSize,
		Data:  jobs,
	}, nil
}

// cacheJobToRedis 将作业缓存到 Redis
func (s *SaltJobService) cacheJobToRedis(ctx context.Context, job *models.SaltJobHistory) {
	if s.redis == nil {
		return
	}

	// 序列化作业信息
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return
	}

	// 保存作业详情（7天过期）
	jobKey := fmt.Sprintf("saltstack:job_detail:%s", job.JID)
	s.redis.Set(ctx, jobKey, string(jobJSON), 7*24*time.Hour)

	// 保存 JID -> TaskID 映射
	if job.TaskID != "" {
		jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", job.JID)
		s.redis.Set(ctx, jidToTaskKey, job.TaskID, 7*24*time.Hour)

		// 保存 TaskID -> JID 映射
		taskToJidKey := fmt.Sprintf("saltstack:task_to_jid:%s", job.TaskID)
		s.redis.Set(ctx, taskToJidKey, job.JID, 7*24*time.Hour)
	}

	// 添加到最近作业列表
	s.redis.LPush(ctx, "saltstack:recent_jobs", job.JID)
	s.redis.LTrim(ctx, "saltstack:recent_jobs", 0, 99)
}

// getJobFromRedis 从 Redis 获取作业
func (s *SaltJobService) getJobFromRedis(ctx context.Context, jid string) *models.SaltJobHistory {
	if s.redis == nil {
		return nil
	}

	jobKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
	jobJSON, err := s.redis.Get(ctx, jobKey).Result()
	if err != nil {
		return nil
	}

	var job models.SaltJobHistory
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		return nil
	}
	return &job
}

// startCleanupWorker 启动后台清理任务
func (s *SaltJobService) startCleanupWorker() {
	s.cleanupOnce.Do(func() {
		go func() {
			// 启动时先执行一次清理
			s.runCleanup()

			ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					s.configMu.RLock()
					enabled := s.config != nil && s.config.CleanupEnabled
					interval := 24
					if s.config != nil {
						interval = s.config.CleanupIntervalHour
					}
					lastCleanup := time.Time{}
					if s.config != nil {
						lastCleanup = s.config.LastCleanupTime
					}
					s.configMu.RUnlock()

					if enabled && time.Since(lastCleanup) >= time.Duration(interval)*time.Hour {
						s.runCleanup()
					}
				case <-s.cleanupDone:
					return
				}
			}
		}()
	})
}

// runCleanup 执行清理
func (s *SaltJobService) runCleanup() {
	s.configMu.RLock()
	maxDays := 30
	maxRecords := 10000
	if s.config != nil {
		maxDays = s.config.MaxRetentionDays
		maxRecords = s.config.MaxRecords
	}
	s.configMu.RUnlock()

	log.Printf("[SaltJobService] 开始清理历史作业记录: 保留天数=%d, 最大记录数=%d", maxDays, maxRecords)

	// 按时间清理
	cutoffTime := time.Now().AddDate(0, 0, -maxDays)
	result := s.db.Where("created_at < ? AND status IN ('completed', 'failed', 'timeout')", cutoffTime).Delete(&models.SaltJobHistory{})
	if result.Error == nil && result.RowsAffected > 0 {
		log.Printf("[SaltJobService] 按时间清理了 %d 条记录", result.RowsAffected)
	}

	// 按数量清理（保留最新的 maxRecords 条）
	var count int64
	s.db.Model(&models.SaltJobHistory{}).Count(&count)
	if count > int64(maxRecords) {
		// 获取第 maxRecords 条记录的 ID
		var job models.SaltJobHistory
		s.db.Model(&models.SaltJobHistory{}).Order("start_time DESC").Offset(maxRecords).First(&job)
		if job.ID > 0 {
			result := s.db.Where("id < ? AND status IN ('completed', 'failed', 'timeout')", job.ID).Delete(&models.SaltJobHistory{})
			if result.Error == nil && result.RowsAffected > 0 {
				log.Printf("[SaltJobService] 按数量清理了 %d 条记录", result.RowsAffected)
			}
		}
	}

	// 更新最后清理时间
	s.configMu.Lock()
	if s.config != nil {
		s.config.LastCleanupTime = time.Now()
		s.db.Save(s.config)
	}
	s.configMu.Unlock()

	log.Printf("[SaltJobService] 清理完成")
}

// CheckAndUpdateStaleJobs 检查并更新过期的运行中作业
func (s *SaltJobService) CheckAndUpdateStaleJobs(ctx context.Context, maxRunningMinutes int) {
	if maxRunningMinutes <= 0 {
		maxRunningMinutes = 30 // 默认30分钟
	}

	cutoffTime := time.Now().Add(-time.Duration(maxRunningMinutes) * time.Minute)
	var staleJobs []models.SaltJobHistory
	s.db.Where("status = 'running' AND start_time < ?", cutoffTime).Find(&staleJobs)

	for _, job := range staleJobs {
		log.Printf("[SaltJobService] 标记过期作业为超时: JID=%s, StartTime=%s", job.JID, job.StartTime)
		s.TimeoutJob(ctx, job.JID)
	}
}

// Stop 停止服务
func (s *SaltJobService) Stop() {
	close(s.cleanupDone)
}
