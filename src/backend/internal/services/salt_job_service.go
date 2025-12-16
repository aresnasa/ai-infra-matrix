package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SaltJobService Salt作业持久化服务
type SaltJobService struct {
	db    *gorm.DB
	cache *redis.Client
}

// NewSaltJobService 创建Salt作业服务
func NewSaltJobService(db *gorm.DB, cache *redis.Client) *SaltJobService {
	return &SaltJobService{
		db:    db,
		cache: cache,
	}
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
	if s.cache != nil && job.TaskID != "" {
		ttl := 7 * 24 * time.Hour // 7天过期

		// JID -> TaskID 映射
		jidToTaskKey := fmt.Sprintf("saltstack:jid_to_task:%s", job.JID)
		s.cache.Set(ctx, jidToTaskKey, job.TaskID, ttl)

		// TaskID -> JID 映射
		taskToJidKey := fmt.Sprintf("saltstack:task_to_jid:%s", job.TaskID)
		s.cache.Set(ctx, taskToJidKey, job.JID, ttl)

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

		log.Printf("[SaltJobService] 作业已保存并缓存: JID=%s, TaskID=%s", job.JID, job.TaskID)
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
