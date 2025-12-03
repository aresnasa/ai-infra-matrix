package services

import (
	"log"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NodeMetricsService 节点指标服务
type NodeMetricsService struct {
	db *gorm.DB
}

var (
	nodeMetricsService     *NodeMetricsService
	nodeMetricsServiceOnce sync.Once
	schedulerStarted       bool
	schedulerMu            sync.Mutex
)

// NewNodeMetricsService 创建节点指标服务实例
func NewNodeMetricsService() *NodeMetricsService {
	nodeMetricsServiceOnce.Do(func() {
		nodeMetricsService = &NodeMetricsService{
			db: database.DB,
		}
		// 启动后台调度器
		nodeMetricsService.startScheduler()
	})
	return nodeMetricsService
}

// startScheduler 启动后台定时调度器
func (s *NodeMetricsService) startScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if schedulerStarted {
		return
	}
	schedulerStarted = true

	go func() {
		// 每 6 小时执行一次清理任务
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		// 启动后立即执行一次清理
		s.runCleanupTask()

		for range ticker.C {
			s.runCleanupTask()
		}
	}()

	logrus.Info("[NodeMetricsService] 后台调度器已启动，每 6 小时清理一次过期指标数据")
}

// runCleanupTask 执行清理任务
func (s *NodeMetricsService) runCleanupTask() {
	// 保留最近 7 天的数据
	deleted, err := s.CleanupOldMetrics(7)
	if err != nil {
		logrus.WithError(err).Error("[NodeMetricsService] 清理过期指标数据失败")
	} else if deleted > 0 {
		logrus.WithField("deleted_count", deleted).Info("[NodeMetricsService] 清理过期指标数据完成")
	}
}

// SaveNodeMetrics 保存节点指标
func (s *NodeMetricsService) SaveNodeMetrics(metrics *models.NodeMetrics) error {
	// 保存到历史记录表
	if err := s.db.Create(metrics).Error; err != nil {
		log.Printf("[NodeMetricsService] Failed to save metrics history: %v", err)
		return err
	}

	// 更新或创建最新指标记录
	return s.updateLatestMetrics(metrics)
}

// updateLatestMetrics 更新最新指标记录
func (s *NodeMetricsService) updateLatestMetrics(metrics *models.NodeMetrics) error {
	latest := &models.NodeMetricsLatest{
		MinionID:         metrics.MinionID,
		Timestamp:        metrics.Timestamp,
		GPUDriverVersion: metrics.GPUDriverVersion,
		CUDAVersion:      metrics.CUDAVersion,
		GPUCount:         metrics.GPUCount,
		GPUModel:         metrics.GPUModel,
		GPUMemoryTotal:   metrics.GPUMemoryTotal,
		GPUInfo:          metrics.GPUInfo,
		IBActiveCount:    metrics.IBActiveCount,
		IBPortsInfo:      metrics.IBPortsInfo,
		KernelVersion:    metrics.KernelVersion,
		OSVersion:        metrics.OSVersion,
		RawData:          metrics.RawData,
	}

	// 使用 upsert（存在则更新，不存在则创建）
	result := s.db.Where("minion_id = ?", metrics.MinionID).First(&models.NodeMetricsLatest{})
	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		return s.db.Create(latest).Error
	}

	// 更新已存在的记录
	return s.db.Model(&models.NodeMetricsLatest{}).
		Where("minion_id = ?", metrics.MinionID).
		Updates(map[string]interface{}{
			"timestamp":          latest.Timestamp,
			"gpu_driver_version": latest.GPUDriverVersion,
			"cuda_version":       latest.CUDAVersion,
			"gpu_count":          latest.GPUCount,
			"gpu_model":          latest.GPUModel,
			"gpu_memory_total":   latest.GPUMemoryTotal,
			"gpu_info":           latest.GPUInfo,
			"ib_active_count":    latest.IBActiveCount,
			"ib_ports_info":      latest.IBPortsInfo,
			"kernel_version":     latest.KernelVersion,
			"os_version":         latest.OSVersion,
			"raw_data":           latest.RawData,
		}).Error
}

// GetLatestMetrics 获取指定节点的最新指标
func (s *NodeMetricsService) GetLatestMetrics(minionID string) (*models.NodeMetricsLatest, error) {
	var metrics models.NodeMetricsLatest
	if err := s.db.Where("minion_id = ?", minionID).First(&metrics).Error; err != nil {
		return nil, err
	}
	return &metrics, nil
}

// GetAllLatestMetrics 获取所有节点的最新指标
func (s *NodeMetricsService) GetAllLatestMetrics() ([]models.NodeMetricsLatest, error) {
	var metrics []models.NodeMetricsLatest
	if err := s.db.Find(&metrics).Error; err != nil {
		return nil, err
	}
	return metrics, nil
}

// GetMetricsHistory 获取指定节点的指标历史
func (s *NodeMetricsService) GetMetricsHistory(minionID string, limit int) ([]models.NodeMetrics, error) {
	if limit <= 0 {
		limit = 100
	}
	var metrics []models.NodeMetrics
	if err := s.db.Where("minion_id = ?", minionID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&metrics).Error; err != nil {
		return nil, err
	}
	return metrics, nil
}

// CleanupOldMetrics 清理旧的指标数据（保留最近 N 天）
func (s *NodeMetricsService) CleanupOldMetrics(keepDays int) (int64, error) {
	if keepDays <= 0 {
		keepDays = 7
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -keepDays)

	// 使用 GORM 的方式删除，兼容多种数据库
	result := s.db.Where("timestamp < ?", cutoffTime).Delete(&models.NodeMetrics{})

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("[NodeMetricsService] Cleaned up %d old metrics records (older than %d days)",
			result.RowsAffected, keepDays)
	}

	return result.RowsAffected, nil
}

// GetMetricsSummary 获取指标汇总统计
func (s *NodeMetricsService) GetMetricsSummary() (map[string]interface{}, error) {
	var totalNodes int64
	var nodesWithGPU int64
	var nodesWithIB int64
	var totalGPUs int64
	var totalIBPorts int64

	// 统计节点总数
	s.db.Model(&models.NodeMetricsLatest{}).Count(&totalNodes)

	// 统计有 GPU 的节点数
	s.db.Model(&models.NodeMetricsLatest{}).Where("gpu_count > 0").Count(&nodesWithGPU)

	// 统计有活跃 IB 端口的节点数
	s.db.Model(&models.NodeMetricsLatest{}).Where("ib_active_count > 0").Count(&nodesWithIB)

	// 统计 GPU 总数
	s.db.Model(&models.NodeMetricsLatest{}).Select("COALESCE(SUM(gpu_count), 0)").Scan(&totalGPUs)

	// 统计 IB 端口总数
	s.db.Model(&models.NodeMetricsLatest{}).Select("COALESCE(SUM(ib_active_count), 0)").Scan(&totalIBPorts)

	return map[string]interface{}{
		"total_nodes":    totalNodes,
		"nodes_with_gpu": nodesWithGPU,
		"nodes_with_ib":  nodesWithIB,
		"total_gpus":     totalGPUs,
		"total_ib_ports": totalIBPorts,
	}, nil
}
