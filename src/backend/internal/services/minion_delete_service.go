package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 缓存键
const saltMinionsCacheKey = "saltstack:minions"

// MinionDeleteService Minion 删除服务
// 实现软删除 + 后台异步真实删除 + 并发处理
type MinionDeleteService struct {
	db                  *gorm.DB
	saltService         *SaltStackService
	batchInstallService *BatchInstallService
	workerOnce          sync.Once
	stopChan            chan struct{}
	taskChan            chan *models.MinionDeleteTask
	workerCount         int // 并发 worker 数量
}

var (
	minionDeleteService     *MinionDeleteService
	minionDeleteServiceOnce sync.Once
)

// GetMinionDeleteService 获取单例服务
func GetMinionDeleteService() *MinionDeleteService {
	minionDeleteServiceOnce.Do(func() {
		minionDeleteService = &MinionDeleteService{
			db:                  database.DB,
			saltService:         NewSaltStackService(),
			batchInstallService: NewBatchInstallService(),
			stopChan:            make(chan struct{}),
			taskChan:            make(chan *models.MinionDeleteTask, 100),
			workerCount:         5, // 默认 5 个并发 worker
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

// startWorker 启动后台删除工作器（支持并发）
func (s *MinionDeleteService) startWorker() {
	s.workerOnce.Do(func() {
		// 启动多个并发 worker
		for i := 0; i < s.workerCount; i++ {
			go s.deleteWorker(i)
		}
		go s.retryWorker()
		logrus.WithField("worker_count", s.workerCount).Info("[MinionDeleteService] 后台删除工作器已启动（并发模式）")
	})
}

// deleteWorker 处理删除任务的工作器
func (s *MinionDeleteService) deleteWorker(workerID int) {
	logger := logrus.WithField("worker_id", workerID)
	logger.Info("[MinionDeleteService] 删除 worker 启动")

	for {
		select {
		case <-s.stopChan:
			logger.Info("[MinionDeleteService] 删除 worker 已停止")
			return
		case task := <-s.taskChan:
			s.processDeleteTask(task, workerID)
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
func (s *MinionDeleteService) processDeleteTask(task *models.MinionDeleteTask, workerID int) {
	logger := logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"minion_id": task.MinionID,
		"retry":     task.RetryCount,
		"worker_id": workerID,
	})

	logger.Info("[MinionDeleteService] 开始处理删除任务")

	// 更新状态为删除中
	task.MarkAsDeleting()
	if err := s.db.Save(task).Error; err != nil {
		logger.WithError(err).Error("更新任务状态失败")
		return
	}

	// 执行真实删除
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. 如果设置了 Uninstall 和 SSH 信息，先尝试通过 SSH 卸载远程节点上的 salt-minion
	if task.Uninstall && task.SSHHost != "" && task.SSHUsername != "" {
		logger.WithField("host", task.SSHHost).Info("[MinionDeleteService] 尝试通过 SSH 卸载远程 salt-minion")

		config := HostInstallConfig{
			Host:     task.SSHHost,
			Port:     task.SSHPort,
			Username: task.SSHUsername,
			Password: task.SSHPassword,
			KeyPath:  task.SSHKeyPath,
			UseSudo:  task.UseSudo,
			SudoPass: task.SSHPassword, // 通常 sudo 密码与登录密码相同
		}

		if config.Port == 0 {
			config.Port = 22
		}

		uninstallErr := s.batchInstallService.UninstallSaltMinion(ctx, config)
		if uninstallErr != nil {
			logger.WithError(uninstallErr).Warn("[MinionDeleteService] SSH 卸载失败，继续删除 Salt Master 密钥")
		} else {
			logger.Info("[MinionDeleteService] SSH 卸载 salt-minion 成功")
		}
	}

	// 2. 从 Salt Master 删除密钥
	err := s.saltService.DeleteMinionWithForce(ctx, task.MinionID, task.Force)
	if err != nil {
		task.MarkAsFailed(err.Error())
		s.db.Save(task)
		logger.WithError(err).Warn("从 Salt Master 删除 Minion 密钥失败")
		return
	}

	// 删除成功
	task.MarkAsCompleted()
	if err := s.db.Save(task).Error; err != nil {
		logger.WithError(err).Error("更新任务完成状态失败")
		return
	}

	// 3. 清除 Minions 缓存，确保前端获取最新列表
	s.clearMinionsCache()

	logger.Info("[MinionDeleteService] Minion 删除成功")
}

// clearMinionsCache 清除 Minions 缓存
func (s *MinionDeleteService) clearMinionsCache() {
	if cache.RDB != nil {
		if err := cache.RDB.Del(context.Background(), saltMinionsCacheKey).Err(); err != nil {
			logrus.WithError(err).Warn("[MinionDeleteService] 清除 Minions 缓存失败")
		} else {
			logrus.Debug("[MinionDeleteService] Minions 缓存已清除")
		}
	}
}

// SoftDeleteOptions 软删除选项
type SoftDeleteOptions struct {
	Force       bool   // 强制删除（包括在线节点）
	Uninstall   bool   // 是否执行远程卸载
	SSHHost     string // SSH 主机地址
	SSHPort     int    // SSH 端口
	SSHUsername string // SSH 用户名
	SSHPassword string // SSH 密码
	SSHKeyPath  string // SSH 私钥路径
	UseSudo     bool   // 是否使用 sudo
}

// SoftDelete 软删除 Minion（立即返回，后台执行真实删除）
func (s *MinionDeleteService) SoftDelete(ctx context.Context, minionID string, force bool, createdBy string) (*models.MinionDeleteTask, error) {
	return s.SoftDeleteWithOptions(ctx, minionID, SoftDeleteOptions{Force: force}, createdBy)
}

// SoftDeleteWithOptions 使用选项软删除 Minion
func (s *MinionDeleteService) SoftDeleteWithOptions(ctx context.Context, minionID string, opts SoftDeleteOptions, createdBy string) (*models.MinionDeleteTask, error) {
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

	// 设置默认 SSH 端口
	sshPort := opts.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	// 创建新的删除任务
	task := &models.MinionDeleteTask{
		MinionID:    minionID,
		Status:      models.MinionDeleteStatusPending,
		Force:       opts.Force,
		MaxRetries:  3,
		CreatedBy:   createdBy,
		Uninstall:   opts.Uninstall,
		SSHHost:     opts.SSHHost,
		SSHPort:     sshPort,
		SSHUsername: opts.SSHUsername,
		SSHPassword: opts.SSHPassword,
		SSHKeyPath:  opts.SSHKeyPath,
		UseSudo:     opts.UseSudo,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建删除任务失败: %w", err)
	}

	// 异步发送到工作队列
	go func() {
		select {
		case s.taskChan <- task:
			logrus.WithFields(logrus.Fields{
				"task_id":   task.ID,
				"minion_id": minionID,
			}).Debug("[MinionDeleteService] 删除任务已发送到队列")
		case <-time.After(5 * time.Second):
			logrus.WithField("minion_id", minionID).Warn("发送删除任务到队列超时")
		}
	}()

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"minion_id": minionID,
		"force":     opts.Force,
		"uninstall": opts.Uninstall,
	}).Info("[MinionDeleteService] Minion 软删除任务已创建")

	return task, nil
}

// SoftDeleteBatch 批量软删除
func (s *MinionDeleteService) SoftDeleteBatch(ctx context.Context, minionIDs []string, force bool, createdBy string) ([]*models.MinionDeleteTask, error) {
	return s.SoftDeleteBatchWithOptions(ctx, minionIDs, SoftDeleteOptions{Force: force}, createdBy)
}

// SoftDeleteBatchWithOptions 使用选项批量软删除
func (s *MinionDeleteService) SoftDeleteBatchWithOptions(ctx context.Context, minionIDs []string, opts SoftDeleteOptions, createdBy string) ([]*models.MinionDeleteTask, error) {
	tasks := make([]*models.MinionDeleteTask, 0, len(minionIDs))

	for _, minionID := range minionIDs {
		task, err := s.SoftDeleteWithOptions(ctx, minionID, opts, createdBy)
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

// GetRecentlyCompletedMinionIDs 获取最近一段时间内完成删除的 Minion ID 列表
// 用于过滤 Salt Master 可能仍然返回的已删除节点残留数据
func (s *MinionDeleteService) GetRecentlyCompletedMinionIDs(duration time.Duration) ([]string, error) {
	var tasks []models.MinionDeleteTask
	cutoff := time.Now().Add(-duration)
	if err := s.db.Select("minion_id").Where(
		"status = ? AND completed_at > ?",
		models.MinionDeleteStatusCompleted,
		cutoff,
	).Find(&tasks).Error; err != nil {
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
