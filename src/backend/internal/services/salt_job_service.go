package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
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
	// 检查前缀 - 使用 strings.HasPrefix 避免越界问题
	if strings.HasPrefix(function, "status.") ||
		strings.HasPrefix(function, "runner.") ||
		strings.HasPrefix(function, "saltutil.") ||
		strings.HasPrefix(function, "grains.") {
		return true
	}
	return false
}

// IsUserTask 判断是否是用户发起的任务
// 用户任务必须有 task_id，且不是监控系统任务
func IsUserTask(function string, taskID string) bool {
	// 必须有 task_id
	if taskID == "" {
		return false
	}
	// 不能是监控任务
	if IsMonitoringTask(function) {
		return false
	}
	return true
}

// SaltJobService Salt作业持久化服务
type SaltJobService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *models.SaltJobConfig
	configMu    sync.RWMutex
	cleanupOnce sync.Once
	syncOnce    sync.Once // 状态同步后台任务
	cleanupDone chan struct{}
	syncDone    chan struct{} // 同步任务停止信号
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
			syncDone:    make(chan struct{}),
		}
		// 自动迁移数据库表
		if err := db.AutoMigrate(&models.SaltJobHistory{}, &models.SaltJobConfig{}); err != nil {
			log.Printf("[SaltJobService] 自动迁移失败: %v", err)
		}
		// 加载或初始化配置
		saltJobServiceInstance.loadOrCreateConfig()
		// 启动后台清理任务
		saltJobServiceInstance.startCleanupWorker()
		// 启动后台状态同步任务
		saltJobServiceInstance.startJobStatusSyncWorker()
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
			BlacklistEnabled:    true,
		}
		// 设置默认危险命令
		config.SetDangerousCommands(models.GetDefaultDangerousCommands())
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

// GetDB 获取数据库连接（用于直接查询）
func (s *SaltJobService) GetDB() *gorm.DB {
	return s.db
}

// GetConfigWithDangerousCommands 获取配置包含危险命令列表
func (s *SaltJobService) GetConfigWithDangerousCommands() map[string]interface{} {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	if s.config == nil {
		return nil
	}

	return map[string]interface{}{
		"id":                             s.config.ID,
		"max_retention_days":             s.config.MaxRetentionDays,
		"max_records":                    s.config.MaxRecords,
		"cleanup_enabled":                s.config.CleanupEnabled,
		"cleanup_interval_hour":          s.config.CleanupIntervalHour,
		"cleanup_interval_value":         s.config.CleanupIntervalValue,
		"cleanup_interval_unit":          s.config.CleanupIntervalUnit,
		"last_cleanup_time":              s.config.LastCleanupTime,
		"blacklist_enabled":              s.config.BlacklistEnabled,
		"require_auth_for_dangerous_cmd": s.config.RequireAuthForDangerousCmd,
		"dangerous_commands":             s.config.GetDangerousCommands(),
		"created_at":                     s.config.CreatedAt,
		"updated_at":                     s.config.UpdatedAt,
	}
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

// UpdateDangerousCommands 更新危险命令列表
func (s *SaltJobService) UpdateDangerousCommands(commands []models.DangerousCommand) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	if s.config == nil {
		return fmt.Errorf("config not initialized")
	}

	if err := s.config.SetDangerousCommands(commands); err != nil {
		return err
	}

	if err := s.db.Save(s.config).Error; err != nil {
		return err
	}
	return nil
}

// CheckDangerousCommand 检查命令是否在黑名单中
// 返回: (是否危险, 匹配的规则, 错误)
func (s *SaltJobService) CheckDangerousCommand(command string) (bool, *models.DangerousCommand, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	if s.config == nil || !s.config.BlacklistEnabled {
		return false, nil, nil
	}

	commands := s.config.GetDangerousCommands()
	for _, cmd := range commands {
		if !cmd.Enabled {
			continue
		}

		matched := false
		if cmd.IsRegex {
			re, err := regexp.Compile(cmd.Pattern)
			if err != nil {
				log.Printf("[SaltJobService] 无效的正则表达式: %s, 错误: %v", cmd.Pattern, err)
				continue
			}
			matched = re.MatchString(command)
		} else {
			// 精确匹配或包含匹配
			matched = strings.Contains(command, cmd.Pattern)
		}

		if matched {
			log.Printf("[SaltJobService] 检测到危险命令: %s, 匹配规则: %s", command, cmd.Pattern)
			return true, &cmd, nil
		}
	}

	return false, nil, nil
}

// CreateJob 创建或更新作业记录
// 存储策略：
// - 用户任务（有 task_id 且不是监控函数）：存入数据库 + Redis
// - 监控任务（无 task_id 或是监控函数）：只存入 Redis 用于临时缓存
func (s *SaltJobService) CreateJob(ctx context.Context, job *models.SaltJobHistory) error {
	// 设置默认状态
	if job.Status == "" {
		job.Status = "running"
	}
	if job.StartTime.IsZero() {
		job.StartTime = time.Now()
	}

	// 判断是否是用户任务
	isUserTask := IsUserTask(job.Function, job.TaskID)

	// 监控任务只存入 Redis，不污染数据库
	if !isUserTask {
		log.Printf("[SaltJobService] 监控任务只存入 Redis: JID=%s, Function=%s", job.JID, job.Function)
		s.cacheJobToRedis(ctx, job)
		return nil
	}

	// 以下是用户任务的处理逻辑
	// 使用 Upsert (FirstOrCreate + Updates) 处理重复 JID
	var existingJob models.SaltJobHistory
	result := s.db.Where("jid = ?", job.JID).First(&existingJob)

	if result.Error == nil {
		// JID 已存在，执行更新（但保留原有的某些字段如 StartTime）
		// 只更新状态、结果和其他执行后的信息
		updateFields := map[string]interface{}{
			"status":        job.Status,
			"result":        job.Result,
			"error_message": job.ErrorMessage,
			"return_code":   job.ReturnCode,
			"success_count": job.SuccessCount,
			"failed_count":  job.FailedCount,
			"end_time":      job.EndTime,
			"duration":      job.Duration,
			"task_id":       job.TaskID,
		}

		// 如果新的字段有值，才更新
		if job.Function != "" {
			updateFields["function"] = job.Function
		}
		if job.Target != "" {
			updateFields["target"] = job.Target
		}
		if job.Arguments != "" {
			updateFields["arguments"] = job.Arguments
		}

		if err := s.db.Model(&existingJob).Updates(updateFields).Error; err != nil {
			log.Printf("[SaltJobService] 更新作业记录失败: %v", err)
			return err
		}

		*job = existingJob
		log.Printf("[SaltJobService] 作业记录已更新: JID=%s, TaskID=%s", job.JID, job.TaskID)
	} else if result.Error.Error() == "record not found" {
		// JID 不存在，创建新记录
		if err := s.db.Create(job).Error; err != nil {
			log.Printf("[SaltJobService] 创建作业记录失败: %v", err)
			return err
		}
		log.Printf("[SaltJobService] 作业记录已创建: JID=%s, TaskID=%s", job.JID, job.TaskID)
	} else {
		// 其他数据库错误
		log.Printf("[SaltJobService] 查询作业记录出错: %v", result.Error)
		return result.Error
	}

	// 同时更新 Redis 缓存（用于快速查询）
	s.cacheJobToRedis(ctx, job)

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
	// 返回码需要记录，包括 0 值（表示成功）
	// 只有在状态为 completed/failed 时才更新返回码
	if update.Status == "completed" || update.Status == "failed" {
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

	// 如果状态变为终态，设置结束时间和持续时间（毫秒，最小1000ms即1秒）
	if update.Status == "completed" || update.Status == "failed" || update.Status == "timeout" {
		now := time.Now()
		updates["end_time"] = now
		durationMs := now.Sub(job.StartTime).Milliseconds()
		// 简单命令最小记录 1 秒
		if durationMs < 1000 {
			durationMs = 1000
		}
		updates["duration"] = durationMs
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
// 从执行结果中提取返回码，返回码取所有节点中最大的非零返回码
func (s *SaltJobService) CompleteJob(ctx context.Context, jid string, result map[string]interface{}, successCount, failedCount int) error {
	// 提取返回码：遍历所有节点结果，获取最大的非零返回码
	returnCode := 0
	for _, v := range result {
		if vMap, ok := v.(map[string]interface{}); ok {
			// 尝试从结果中提取 retcode
			if retcode, ok := vMap["retcode"].(float64); ok {
				if int(retcode) != 0 && (returnCode == 0 || int(retcode) > returnCode) {
					returnCode = int(retcode)
				}
			}
		}
	}

	return s.UpdateJobStatus(ctx, jid, &models.SaltJobUpdateRequest{
		Status:       "completed",
		Result:       result,
		SuccessCount: successCount,
		FailedCount:  failedCount,
		ReturnCode:   returnCode,
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
// 默认只返回用户任务（有 task_id 的），可通过 user_only=false 参数查看所有任务
func (s *SaltJobService) ListJobs(ctx context.Context, params *models.SaltJobQueryParams) (*models.SaltJobListResponse, error) {
	query := s.db.Model(&models.SaltJobHistory{})

	// 默认只返回用户任务（有 task_id 的）
	// 用户任务的定义：1. 有 task_id  2. 不是监控函数
	if params.UserOnly {
		// 必须有 task_id（用户提交的任务都有 task_id）
		query = query.Where("task_id IS NOT NULL AND task_id != ''")
	}

	// 始终过滤监控系统的状态
	query = query.Where("status NOT IN ('monitoring', 'system')")

	// 始终使用黑名单过滤监控函数（即使 user_only=false 也过滤）
	monitoringFunctions := []string{
		"status.cpuload", "status.meminfo", "status.cpuinfo", "status.diskusage",
		"status.netstats", "status.uptime", "status.loadavg", "status.loadavg5", "status.loadavg15",
		"runner.manage.status",
		"test.ping",
		"grains.items",
		"saltutil.sync_all", "saltutil.sync_grains", "saltutil.sync_modules", "saltutil.refresh",
		"saltutil.find_job", // Salt 内部用于查询任务状态
		"cmd.run_all", "test.echo",
	}
	query = query.Where("function NOT IN ?", monitoringFunctions)
	query = query.Where("function NOT LIKE 'status.%'")
	query = query.Where("function NOT LIKE 'runner.%'")
	query = query.Where("function NOT LIKE 'saltutil.%'")
	query = query.Where("function NOT LIKE 'grains.%'")

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
	close(s.syncDone)
}

// startJobStatusSyncWorker 启动后台状态同步任务
// 定期检查 running 状态的作业，并从 Redis 同步最新状态到数据库
func (s *SaltJobService) startJobStatusSyncWorker() {
	s.syncOnce.Do(func() {
		go func() {
			// 启动时先执行一次同步
			s.syncRunningJobs()

			ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					s.syncRunningJobs()
				case <-s.syncDone:
					log.Printf("[SaltJobService] 状态同步任务已停止")
					return
				}
			}
		}()
	})
}

// syncRunningJobs 同步 running 状态的作业
func (s *SaltJobService) syncRunningJobs() {
	ctx := context.Background()

	// 查找所有 running 状态的作业
	var runningJobs []models.SaltJobHistory
	if err := s.db.Where("status = 'running'").Find(&runningJobs).Error; err != nil {
		log.Printf("[SaltJobService] 查询运行中作业失败: %v", err)
		return
	}

	if len(runningJobs) == 0 {
		return
	}

	log.Printf("[SaltJobService] 检查 %d 个运行中的作业状态", len(runningJobs))

	for _, job := range runningJobs {
		// 检查作业是否已超时（超过30分钟）
		if time.Since(job.StartTime) > 30*time.Minute {
			log.Printf("[SaltJobService] 作业超时，标记为 timeout: JID=%s, StartTime=%s", job.JID, job.StartTime)
			s.TimeoutJob(ctx, job.JID)
			continue
		}

		// 尝试从 Redis 获取最新状态
		if s.redis != nil {
			jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", job.JID)
			jobInfoJSON, err := s.redis.Get(ctx, jobDetailKey).Result()
			if err == nil && jobInfoJSON != "" {
				var cachedJob map[string]interface{}
				if err := json.Unmarshal([]byte(jobInfoJSON), &cachedJob); err == nil {
					status, _ := cachedJob["status"].(string)
					if status != "" && status != "running" {
						// Redis 中状态已更新，同步到数据库
						log.Printf("[SaltJobService] 从 Redis 同步作业状态: JID=%s, 新状态=%s", job.JID, status)

						update := &models.SaltJobUpdateRequest{
							Status: status,
						}

						// 解析成功/失败数量
						if sc, ok := cachedJob["success_count"].(float64); ok {
							update.SuccessCount = int(sc)
						}
						if fc, ok := cachedJob["failed_count"].(float64); ok {
							update.FailedCount = int(fc)
						}

						// 解析结果
						if result, ok := cachedJob["result"].(map[string]interface{}); ok {
							update.Result = result
						}

						// 解析返回码
						if rc, ok := cachedJob["return_code"].(float64); ok {
							update.ReturnCode = int(rc)
						}

						s.UpdateJobStatus(ctx, job.JID, update)
					}
				}
			}
		}
	}
}

// SyncJobFromRedis 从 Redis 同步单个作业到数据库（用于命令执行后立即同步）
func (s *SaltJobService) SyncJobFromRedis(ctx context.Context, jid string) error {
	if s.redis == nil {
		return fmt.Errorf("redis not available")
	}

	// 先检查数据库是否已存在
	var existingJob models.SaltJobHistory
	if err := s.db.Where("jid = ?", jid).First(&existingJob).Error; err == nil {
		// 作业已存在，更新状态
		jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
		jobInfoJSON, err := s.redis.Get(ctx, jobDetailKey).Result()
		if err != nil {
			return fmt.Errorf("redis get failed: %v", err)
		}

		var cachedJob map[string]interface{}
		if err := json.Unmarshal([]byte(jobInfoJSON), &cachedJob); err != nil {
			return fmt.Errorf("json unmarshal failed: %v", err)
		}

		status, _ := cachedJob["status"].(string)
		if status != "" && status != existingJob.Status {
			update := &models.SaltJobUpdateRequest{Status: status}
			if sc, ok := cachedJob["success_count"].(float64); ok {
				update.SuccessCount = int(sc)
			}
			if fc, ok := cachedJob["failed_count"].(float64); ok {
				update.FailedCount = int(fc)
			}
			if result, ok := cachedJob["result"].(map[string]interface{}); ok {
				update.Result = result
			}
			return s.UpdateJobStatus(ctx, jid, update)
		}
		return nil
	}

	// 数据库不存在，从 Redis 创建
	return s.createJobFromRedis(ctx, jid)
}

// createJobFromRedis 从 Redis 创建作业记录
func (s *SaltJobService) createJobFromRedis(ctx context.Context, jid string) error {
	if s.redis == nil {
		return fmt.Errorf("redis not available")
	}

	jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", jid)
	jobInfoJSON, err := s.redis.Get(ctx, jobDetailKey).Result()
	if err != nil {
		return fmt.Errorf("redis get failed: %v", err)
	}

	var cachedJob map[string]interface{}
	if err := json.Unmarshal([]byte(jobInfoJSON), &cachedJob); err != nil {
		return fmt.Errorf("json unmarshal failed: %v", err)
	}

	function, _ := cachedJob["function"].(string)
	target, _ := cachedJob["target"].(string)
	user, _ := cachedJob["user"].(string)
	taskID, _ := cachedJob["task_id"].(string)
	status, _ := cachedJob["status"].(string)
	if status == "" {
		status = "running"
	}

	// 解析参数
	var argsStr string
	if args, ok := cachedJob["arguments"]; ok {
		argsJSON, _ := json.Marshal(args)
		argsStr = string(argsJSON)
	}

	// 解析结果
	var resultStr string
	if result, ok := cachedJob["result"]; ok && result != nil {
		resultJSON, _ := json.Marshal(result)
		resultStr = string(resultJSON)
	}

	// 解析开始时间
	startTime := time.Now()
	if startTimeStr, ok := cachedJob["start_time"].(string); ok && startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}

	job := &models.SaltJobHistory{
		JID:       jid,
		TaskID:    taskID,
		Function:  function,
		Target:    target,
		Arguments: argsStr,
		Result:    resultStr,
		User:      user,
		Status:    status,
		StartTime: startTime,
	}

	// 解析成功/失败数量
	if sc, ok := cachedJob["success_count"].(float64); ok {
		job.SuccessCount = int(sc)
	}
	if fc, ok := cachedJob["failed_count"].(float64); ok {
		job.FailedCount = int(fc)
	}

	return s.CreateJob(ctx, job)
}
