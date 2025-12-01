package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MinionDeleteService Minion 删除服务
// 实现软删除 + 后台异步真实删除
type MinionDeleteService struct {
	db          *gorm.DB
	saltService *SaltStackService
	workerOnce  sync.Once
	stopChan    chan struct{}
	taskChan    chan *models.MinionDeleteTask
}

var (
	minionDeleteService     *MinionDeleteService
	minionDeleteServiceOnce sync.Once
)

// GetMinionDeleteService 获取单例服务
func GetMinionDeleteService() *MinionDeleteService {
	minionDeleteServiceOnce.Do(func() {
		minionDeleteService = &MinionDeleteService{
			db:          database.DB,
			saltService: NewSaltStackService(),
			stopChan:    make(chan struct{}),
			taskChan:    make(chan *models.MinionDeleteTask, 100),
		}
		// 启动后台工作器
		minionDeleteService.startWorker()
	})
	return minionDeleteService
}

// NewMinionDeleteService 创建删除服务（用于依赖注入）
func NewMinionDeleteService() *MinionDeleteService {
	return GetMinionDeleteService()
}

// startWorker 启动后台删除工作器
func (s *MinionDeleteService) startWorker() {
	s.workerOnce.Do(func() {
		go s.deleteWorker()
		go s.retryWorker()
		logrus.Info("[MinionDeleteService] 后台删除工作器已启动")
	})
}

// deleteWorker 处理删除任务的工作器
func (s *MinionDeleteService) deleteWorker() {
	for {
		select {
		case <-s.stopChan:
			logrus.Info("[MinionDeleteService] 删除工作器已停止")
			return
		case task := <-s.taskChan:
			s.processDeleteTask(task)
		}
	}
}

// retryWorker 定期检查失败的任务并重试
func (s *MinionDeleteService) retryWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.processPendingTasks()
		}
	}
}

// processPendingTasks 处理所有待处理的任务
func (s *MinionDeleteService) processPendingTasks() {
	var tasks []models.MinionDeleteTask
	result := s.db.Where("status IN ?", []string{
		models.MinionDeleteStatusPending,
		models.MinionDeleteStatusFailed,
	}).Where("retry_count < max_retries").Find(&tasks)

	if result.Error != nil {
		logrus.WithError(result.Error).Error("[MinionDeleteService] 查询待处理任务失败")
		return
	}

	for i := range tasks {
		task := &tasks[i]
		// 检查是否可以重试失败的任务
		if task.Status == models.MinionDeleteStatusFailed && !task.CanRetry() {
			continue
		}
		// 非阻塞发送到任务通道
		select {
		case s.taskChan <- task:
		default:
			// 通道已满，跳过
		}
	}
}

// processDeleteTask 执行实际的删除操作
func (s *MinionDeleteService) processDeleteTask(task *models.MinionDeleteTask) {
	logger := logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"minion_id": task.MinionID,
		"retry":     task.RetryCount,
	})

	// 更新状态为删除中
	task.MarkAsDeleting()
	if err := s.db.Save(task).Error; err != nil {
		logger.WithError(err).Error("更新任务状态失败")
		return
	}

	// 执行真实删除
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := s.saltService.DeleteMinionWithForce(ctx, task.MinionID, task.Force)
	if err != nil {
		task.MarkAsFailed(err.Error())
		s.db.Save(task)
		logger.WithError(err).Warn("删除 Minion 失败")
		return
	}

	// 删除成功
	task.MarkAsCompleted()
	if err := s.db.Save(task).Error; err != nil {
		logger.WithError(err).Error("更新任务完成状态失败")
		return
	}

	logger.Info("Minion 删除成功")
}

// SoftDelete 软删除 Minion（立即返回，后台执行真实删除）
func (s *MinionDeleteService) SoftDelete(ctx context.Context, minionID string, force bool, createdBy string) (*models.MinionDeleteTask, error) {
	// 检查是否已有待处理的删除任务
	var existingTask models.MinionDeleteTask
	result := s.db.Where("minion_id = ? AND status IN ?", minionID, []string{
		models.MinionDeleteStatusPending,
		models.MinionDeleteStatusDeleting,
	}).First(&existingTask)

	if result.Error == nil {
		// 已有待处理任务
		return &existingTask, nil
	}

	// 创建新的删除任务
	task := &models.MinionDeleteTask{
		MinionID:   minionID,
		Status:     models.MinionDeleteStatusPending,
		Force:      force,
		MaxRetries: 3,
		CreatedBy:  createdBy,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建删除任务失败: %w", err)
	}

	// 异步发送到工作队列
	go func() {
		select {
		case s.taskChan <- task:
		case <-time.After(5 * time.Second):
			logrus.WithField("minion_id", minionID).Warn("发送删除任务到队列超时")
		}
	}()

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"minion_id": minionID,
		"force":     force,
	}).Info("Minion 软删除任务已创建")

	return task, nil
}

// SoftDeleteBatch 批量软删除
func (s *MinionDeleteService) SoftDeleteBatch(ctx context.Context, minionIDs []string, force bool, createdBy string) ([]*models.MinionDeleteTask, error) {
	tasks := make([]*models.MinionDeleteTask, 0, len(minionIDs))

	for _, minionID := range minionIDs {
		task, err := s.SoftDelete(ctx, minionID, force, createdBy)
		if err != nil {
			logrus.WithError(err).WithField("minion_id", minionID).Warn("创建批量删除任务失败")
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetDeleteTask 获取删除任务
func (s *MinionDeleteService) GetDeleteTask(taskID uint) (*models.MinionDeleteTask, error) {
	var task models.MinionDeleteTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetDeleteTaskByMinionID 根据 Minion ID 获取最新删除任务
func (s *MinionDeleteService) GetDeleteTaskByMinionID(minionID string) (*models.MinionDeleteTask, error) {
	var task models.MinionDeleteTask
	if err := s.db.Where("minion_id = ?", minionID).Order("created_at DESC").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetPendingDeleteMinionIDs 获取所有待删除的 Minion ID 列表
func (s *MinionDeleteService) GetPendingDeleteMinionIDs() ([]string, error) {
	var tasks []models.MinionDeleteTask
	if err := s.db.Select("minion_id").Where("status IN ?", []string{
		models.MinionDeleteStatusPending,
		models.MinionDeleteStatusDeleting,
	}).Find(&tasks).Error; err != nil {
		return nil, err
	}

	minionIDs := make([]string, len(tasks))
	for i, task := range tasks {
		minionIDs[i] = task.MinionID
	}
	return minionIDs, nil
}

// IsMinionPendingDelete 检查 Minion 是否待删除
func (s *MinionDeleteService) IsMinionPendingDelete(minionID string) bool {
	var count int64
	s.db.Model(&models.MinionDeleteTask{}).Where("minion_id = ? AND status IN ?", minionID, []string{
		models.MinionDeleteStatusPending,
		models.MinionDeleteStatusDeleting,
	}).Count(&count)
	return count > 0
}

// CancelDelete 取消删除任务
func (s *MinionDeleteService) CancelDelete(minionID string) error {
	result := s.db.Model(&models.MinionDeleteTask{}).
		Where("minion_id = ? AND status IN ?", minionID, []string{
			models.MinionDeleteStatusPending,
			models.MinionDeleteStatusFailed,
		}).
		Update("status", models.MinionDeleteStatusCancelled)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("没有可取消的删除任务")
	}
	return nil
}

// RetryDelete 重试删除任务
func (s *MinionDeleteService) RetryDelete(minionID string) error {
	var task models.MinionDeleteTask
	if err := s.db.Where("minion_id = ? AND status = ?", minionID, models.MinionDeleteStatusFailed).
		Order("created_at DESC").First(&task).Error; err != nil {
		return fmt.Errorf("没有找到失败的删除任务: %w", err)
	}

	if !task.CanRetry() {
		return fmt.Errorf("任务已达到最大重试次数")
	}

	// 重置状态为待处理
	task.Status = models.MinionDeleteStatusPending
	if err := s.db.Save(&task).Error; err != nil {
		return err
	}

	// 发送到工作队列
	select {
	case s.taskChan <- &task:
	default:
	}

	return nil
}

// ListDeleteTasks 列出删除任务
func (s *MinionDeleteService) ListDeleteTasks(status string, limit int) ([]models.MinionDeleteTask, error) {
	var tasks []models.MinionDeleteTask
	query := s.db.Order("created_at DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// CleanupCompletedTasks 清理已完成的任务（保留最近N天）
func (s *MinionDeleteService) CleanupCompletedTasks(retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.Where("status = ? AND completed_at < ?", models.MinionDeleteStatusCompleted, cutoff).
		Delete(&models.MinionDeleteTask{})
	return result.RowsAffected, result.Error
}

// Stop 停止服务
func (s *MinionDeleteService) Stop() {
	close(s.stopChan)
}
