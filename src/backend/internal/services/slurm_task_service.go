package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SlurmTaskService SLURM任务服务
type SlurmTaskService struct {
	db *gorm.DB
}

// NewSlurmTaskService 创建SLURM任务服务
func NewSlurmTaskService(db *gorm.DB) *SlurmTaskService {
	return &SlurmTaskService{db: db}
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	UserID      uint                   `json:"user_id" binding:"required"`
	ClusterID   *uint                  `json:"cluster_id"`
	Parameters  map[string]interface{} `json:"parameters"`
	TargetNodes []string               `json:"target_nodes"`
	Tags        []string               `json:"tags"`
	Priority    int                    `json:"priority"`
	MaxRetries  int                    `json:"max_retries"`
}

// TaskListRequest 任务列表查询请求
type TaskListRequest struct {
	UserID    *uint      `form:"user_id"`
	Status    string     `form:"status"`
	Type      string     `form:"type"`
	StartDate *time.Time `form:"start_date"`
	EndDate   *time.Time `form:"end_date"`
	Tags      []string   `form:"tags"`
	Page      int        `form:"page" binding:"min=1"`
	PageSize  int        `form:"page_size" binding:"min=1,max=100"`
	OrderBy   string     `form:"order_by"`
	OrderDesc bool       `form:"order_desc"`
}

// TaskDetailResponse 任务详情响应
type TaskDetailResponse struct {
	*models.SlurmTask
	Events     []models.SlurmTaskEvent `json:"events"`
	Statistics map[string]interface{}  `json:"statistics"`
	CanRetry   bool                    `json:"can_retry"`
	CanCancel  bool                    `json:"can_cancel"`
}

// CreateTask 创建新任务
func (s *SlurmTaskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*models.SlurmTask, error) {
	// 生成唯一任务ID
	taskID := uuid.New().String()

	logrus.WithFields(logrus.Fields{
		"task_id":      taskID,
		"user_id":      req.UserID,
		"cluster_id":   req.ClusterID,
		"target_nodes": req.TargetNodes,
		"name":         req.Name,
		"type":         req.Type,
	}).Debug("CreateTask: 开始创建任务")

	// 序列化参数
	var parametersJSON models.JSON
	if req.Parameters != nil {
		paramBytes, err := json.Marshal(req.Parameters)
		if err != nil {
			return nil, fmt.Errorf("序列化任务参数失败: %w", err)
		}
		parametersJSON = models.JSON(paramBytes)
	}

	// 创建任务记录
	task := &models.SlurmTask{
		TaskID:      taskID,
		Name:        req.Name,
		Type:        req.Type,
		Status:      "pending",
		UserID:      req.UserID,
		ClusterID:   req.ClusterID,
		Parameters:  parametersJSON,
		TargetNodes: req.TargetNodes,
		Tags:        req.Tags,
		Priority:    req.Priority,
		MaxRetries:  req.MaxRetries,
	}

	logrus.WithFields(logrus.Fields{
		"task_uuid":    task.TaskID,
		"user_id":      task.UserID,
		"cluster_id":   task.ClusterID,
		"target_nodes": len(task.TargetNodes),
	}).Debug("CreateTask: 准备插入数据库")

	if err := s.db.Create(task).Error; err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"task_uuid": task.TaskID,
			"user_id":   task.UserID,
		}).Error("CreateTask: 数据库插入失败")
		return nil, fmt.Errorf("创建任务记录失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"task_db_id": task.ID,
		"task_uuid":  task.TaskID,
	}).Debug("CreateTask: 任务已插入数据库")

	// 确保任务ID已经被数据库填充
	if task.ID == 0 {
		logrus.Warn("CreateTask: task.ID 为 0，重新查询")
		// 重新查询以获取ID
		if err := s.db.Where("task_id = ?", taskID).First(task).Error; err != nil {
			return nil, fmt.Errorf("查询新创建的任务失败: %w", err)
		}
		logrus.WithField("task_db_id", task.ID).Debug("CreateTask: 重新查询获得ID")
	}

	logrus.WithFields(logrus.Fields{
		"task_db_id": task.ID,
		"task_uuid":  task.TaskID,
	}).Debug("CreateTask: 准备添加初始事件")

	// 添加初始事件
	if err := task.AddEvent(s.db, "created", "initialize", "任务已创建", "", 0, req.Parameters); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"task_db_id": task.ID,
			"task_uuid":  task.TaskID,
		}).Error("CreateTask: 添加初始事件失败")
		return nil, fmt.Errorf("添加初始事件失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"task_db_id": task.ID,
		"task_uuid":  task.TaskID,
	}).Info("CreateTask: 任务创建成功")

	return task, nil
}

// GetTask 获取任务详情
func (s *SlurmTaskService) GetTask(ctx context.Context, taskID string) (*TaskDetailResponse, error) {
	var task models.SlurmTask

	// 尝试按 task_id (UUID) 查询
	err := s.db.Preload("User").Preload("Cluster").
		Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		// 如果失败，尝试按 id (主键) 查询
		if err := s.db.Preload("User").Preload("Cluster").
			Where("id = ?", taskID).First(&task).Error; err != nil {
			return nil, fmt.Errorf("查询任务失败: %w", err)
		}
	}

	// 获取任务事件
	var events []models.SlurmTaskEvent
	if err := s.db.Where("task_id = ?", task.ID).
		Order("timestamp ASC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("查询任务事件失败: %w", err)
	}

	// 计算统计信息
	statistics := s.calculateStatistics(&task, events)

	response := &TaskDetailResponse{
		SlurmTask:  &task,
		Events:     events,
		Statistics: statistics,
		CanRetry:   s.canRetry(&task),
		CanCancel:  s.canCancel(&task),
	}

	return response, nil
}

// GetTaskByID 根据数据库ID获取任务
func (s *SlurmTaskService) GetTaskByID(ctx context.Context, id uint) (*models.SlurmTask, error) {
	var task models.SlurmTask
	err := s.db.Preload("User").Preload("Cluster").
		Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return &task, nil
}

// ListTasks 获取任务列表
func (s *SlurmTaskService) ListTasks(ctx context.Context, req TaskListRequest) ([]models.SlurmTask, int64, error) {
	query := s.db.Model(&models.SlurmTask{}).Preload("User")

	// 应用过滤条件
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	if req.StartDate != nil {
		query = query.Where("created_at >= ?", *req.StartDate)
	}

	if req.EndDate != nil {
		query = query.Where("created_at <= ?", *req.EndDate)
	}

	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			query = query.Where("tags::text LIKE ?", "%"+tag+"%")
		}
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("计算任务总数失败: %w", err)
	}

	// 应用排序
	orderBy := "created_at"
	if req.OrderBy != "" {
		orderBy = req.OrderBy
	}
	if req.OrderDesc {
		orderBy += " DESC"
	} else {
		orderBy += " ASC"
	}
	query = query.Order(orderBy)

	// 应用分页
	offset := (req.Page - 1) * req.PageSize
	query = query.Offset(offset).Limit(req.PageSize)

	var tasks []models.SlurmTask
	if err := query.Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败: %w", err)
	}

	return tasks, total, nil
}

// UpdateTaskStatus 更新任务状态
func (s *SlurmTaskService) UpdateTaskStatus(ctx context.Context, taskID string, status string, errorMsg ...string) error {
	var task models.SlurmTask
	if err := s.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return fmt.Errorf("查询任务失败: %w", err)
	}

	return task.Complete(s.db, status, errorMsg...)
}

// UpdateTaskProgress 更新任务进度
func (s *SlurmTaskService) UpdateTaskProgress(ctx context.Context, taskID string, progress float64, currentStep string) error {
	var task models.SlurmTask
	if err := s.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return fmt.Errorf("查询任务失败: %w", err)
	}

	return task.UpdateProgress(s.db, progress, currentStep)
}

// UpdateTaskNodes 更新任务的节点统计
func (s *SlurmTaskService) UpdateTaskNodes(ctx context.Context, taskID string, total, success, failed int) error {
	err := s.db.Model(&models.SlurmTask{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"nodes_total":   total,
			"nodes_success": success,
			"nodes_failed":  failed,
		}).Error

	if err != nil {
		return fmt.Errorf("更新任务节点统计失败: %w", err)
	}

	return nil
}

// AddTaskEvent 添加任务事件
func (s *SlurmTaskService) AddTaskEvent(ctx context.Context, taskID string, eventType, step, message, host string, progress float64, data interface{}) error {
	var task models.SlurmTask
	if err := s.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return fmt.Errorf("查询任务失败: %w", err)
	}

	return task.AddEvent(s.db, eventType, step, message, host, progress, data)
}

// StartTask 启动任务
func (s *SlurmTaskService) StartTask(ctx context.Context, taskID string) error {
	now := time.Now()
	err := s.db.Model(&models.SlurmTask{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":     "running",
			"started_at": now,
		}).Error

	if err != nil {
		return fmt.Errorf("启动任务失败: %w", err)
	}

	// 添加启动事件
	return s.AddTaskEvent(ctx, taskID, "start", "start", "任务开始执行", "", 0, nil)
}

// CancelTask 取消任务
func (s *SlurmTaskService) CancelTask(ctx context.Context, taskID string, reason string) error {
	var task models.SlurmTask

	// 尝试按 task_id (UUID) 查询
	err := s.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		// 如果失败，尝试按 id (主键) 查询
		if err := s.db.Where("id = ?", taskID).First(&task).Error; err != nil {
			return fmt.Errorf("查询任务失败: %w", err)
		}
	}

	if !s.canCancel(&task) {
		return fmt.Errorf("任务当前状态不支持取消操作")
	}

	if err := task.Complete(s.db, "cancelled", reason); err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}

	// 添加取消事件（使用 task.TaskID 确保是 UUID）
	return s.AddTaskEvent(ctx, task.TaskID, "cancelled", "cancel", "任务已取消: "+reason, "", task.Progress, nil)
}

// RetryTask 重试任务
func (s *SlurmTaskService) RetryTask(ctx context.Context, taskID string) (*models.SlurmTask, error) {
	var originalTask models.SlurmTask

	// 尝试按 task_id (UUID) 查询
	err := s.db.Where("task_id = ?", taskID).First(&originalTask).Error
	if err != nil {
		// 如果失败，尝试按 id (主键) 查询
		if err := s.db.Where("id = ?", taskID).First(&originalTask).Error; err != nil {
			return nil, fmt.Errorf("查询原任务失败: %w", err)
		}
	}

	if !s.canRetry(&originalTask) {
		return nil, fmt.Errorf("任务当前状态不支持重试操作")
	}

	// 解析原任务参数
	var parameters map[string]interface{}
	if originalTask.Parameters != nil {
		if err := json.Unmarshal([]byte(originalTask.Parameters), &parameters); err != nil {
			return nil, fmt.Errorf("解析原任务参数失败: %w", err)
		}
	}

	// 创建重试任务
	retryReq := CreateTaskRequest{
		Name:        originalTask.Name + " (重试)",
		Type:        originalTask.Type,
		UserID:      originalTask.UserID,
		ClusterID:   originalTask.ClusterID,
		Parameters:  parameters,
		TargetNodes: originalTask.TargetNodes,
		Tags:        append(originalTask.Tags, "retry"),
		Priority:    originalTask.Priority,
		MaxRetries:  originalTask.MaxRetries,
	}

	newTask, err := s.CreateTask(ctx, retryReq)
	if err != nil {
		return nil, fmt.Errorf("创建重试任务失败: %w", err)
	}

	// 更新原任务重试次数
	s.db.Model(&originalTask).Update("retry_count", originalTask.RetryCount+1)

	return newTask, nil
}

// GetTaskStatistics 获取任务统计信息
func (s *SlurmTaskService) GetTaskStatistics(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总任务数 - 使用独立查询
	var totalTasks int64
	s.db.Model(&models.SlurmTask{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Count(&totalTasks)
	stats["total_tasks"] = totalTasks

	// 按状态统计 - 使用新的查询
	statusStats := make(map[string]int64)
	var statusResults []struct {
		Status string
		Count  int64
	}
	s.db.Model(&models.SlurmTask{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusResults)

	for _, result := range statusResults {
		statusStats[result.Status] = result.Count
	}
	stats["status_stats"] = statusStats

	// 为前端提供单独的字段（方便直接使用）
	stats["running_tasks"] = statusStats["running"] + statusStats["pending"] + statusStats["in_progress"]
	stats["completed_tasks"] = statusStats["completed"] + statusStats["success"]
	stats["failed_tasks"] = statusStats["failed"] + statusStats["error"] + statusStats["cancelled"]

	// 按类型统计 - 使用新的查询
	typeStats := make(map[string]int64)
	var typeResults []struct {
		Type  string
		Count int64
	}
	s.db.Model(&models.SlurmTask{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Select("type, count(*) as count").
		Group("type").
		Scan(&typeResults)

	for _, result := range typeResults {
		typeStats[result.Type] = result.Count
	}
	stats["type_stats"] = typeStats

	// 平均执行时间
	var avgDuration float64
	s.db.Model(&models.SlurmTask{}).
		Where("created_at >= ? AND created_at <= ? AND duration > 0", startDate, endDate).
		Select("AVG(duration)").Scan(&avgDuration)
	stats["avg_duration"] = avgDuration

	// 成功率
	var successCount int64
	s.db.Model(&models.SlurmTask{}).
		Where("created_at >= ? AND created_at <= ? AND status IN ?",
			startDate, endDate, []string{"completed", "success"}).
		Count(&successCount)

	successRate := float64(0)
	if totalTasks > 0 {
		successRate = float64(successCount) / float64(totalTasks) * 100
	}
	stats["success_rate"] = successRate

	// 活跃用户数（在时间范围内有任务的用户）
	var activeUsers int64
	s.db.Model(&models.SlurmTask{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Distinct("user_id").
		Count(&activeUsers)
	stats["active_users"] = activeUsers

	return stats, nil
}

// DeleteTask 删除任务
func (s *SlurmTaskService) DeleteTask(ctx context.Context, taskID string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 查找任务
	var task models.SlurmTask
	if err := tx.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询任务失败: %w", err)
	}

	// 删除相关事件
	if err := tx.Where("task_id = ?", task.ID).Delete(&models.SlurmTaskEvent{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除任务事件失败: %w", err)
	}

	// 删除任务
	if err := tx.Delete(&task).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除任务失败: %w", err)
	}

	return tx.Commit().Error
}

// 辅助方法

// calculateStatistics 计算任务统计信息
func (s *SlurmTaskService) calculateStatistics(task *models.SlurmTask, events []models.SlurmTaskEvent) map[string]interface{} {
	stats := make(map[string]interface{})

	stats["total_events"] = len(events)
	stats["success_rate"] = task.GetSuccessRate()
	stats["formatted_duration"] = task.GetFormattedDuration()

	// 事件类型统计
	eventTypeStats := make(map[string]int)
	for _, event := range events {
		eventTypeStats[event.EventType]++
	}
	stats["event_type_stats"] = eventTypeStats

	// 步骤统计
	stepStats := make(map[string]int)
	for _, event := range events {
		if event.Step != "" {
			stepStats[event.Step]++
		}
	}
	stats["step_stats"] = stepStats

	return stats
}

// canRetry 检查任务是否可以重试
func (s *SlurmTaskService) canRetry(task *models.SlurmTask) bool {
	return task.Status == "failed" && task.RetryCount < task.MaxRetries
}

// canCancel 检查任务是否可以取消
func (s *SlurmTaskService) canCancel(task *models.SlurmTask) bool {
	return task.Status == "pending" || task.Status == "running"
}
