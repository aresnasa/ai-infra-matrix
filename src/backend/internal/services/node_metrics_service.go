package services

import (
	"encoding/json"
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
		MinionID:  metrics.MinionID,
		Timestamp: metrics.Timestamp,
		// CPU
		CPUCores:        metrics.CPUCores,
		CPUModel:        metrics.CPUModel,
		CPUUsagePercent: metrics.CPUUsagePercent,
		CPULoadAvg:      metrics.CPULoadAvg,
		// Memory
		MemoryTotalGB:      metrics.MemoryTotalGB,
		MemoryUsedGB:       metrics.MemoryUsedGB,
		MemoryAvailableGB:  metrics.MemoryAvailableGB,
		MemoryUsagePercent: metrics.MemoryUsagePercent,
		// Network
		NetworkInfo:       metrics.NetworkInfo,
		ActiveConnections: metrics.ActiveConnections,
		// GPU
		GPUDriverVersion:      metrics.GPUDriverVersion,
		CUDAVersion:           metrics.CUDAVersion,
		GPUCount:              metrics.GPUCount,
		GPUModel:              metrics.GPUModel,
		GPUMemoryTotal:        metrics.GPUMemoryTotal,
		GPUAvgUtilization:     metrics.GPUAvgUtilization,
		GPUMemoryUsedMB:       metrics.GPUMemoryUsedMB,
		GPUMemoryTotalMB:      metrics.GPUMemoryTotalMB,
		GPUMemoryUsagePercent: metrics.GPUMemoryUsagePercent,
		GPUInfo:               metrics.GPUInfo,
		// IB
		IBActiveCount: metrics.IBActiveCount,
		IBDownCount:   metrics.IBDownCount,
		IBTotalCount:  metrics.IBTotalCount,
		IBPortsInfo:   metrics.IBPortsInfo,
		// RoCE
		RoCEInfo: metrics.RoCEInfo,
		// System
		KernelVersion: metrics.KernelVersion,
		OSVersion:     metrics.OSVersion,
		UptimeSeconds: metrics.UptimeSeconds,
		RawData:       metrics.RawData,
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
			"timestamp":                latest.Timestamp,
			"cpu_cores":                latest.CPUCores,
			"cpu_model":                latest.CPUModel,
			"cpu_usage_percent":        latest.CPUUsagePercent,
			"cpu_load_avg":             latest.CPULoadAvg,
			"memory_total_gb":          latest.MemoryTotalGB,
			"memory_used_gb":           latest.MemoryUsedGB,
			"memory_available_gb":      latest.MemoryAvailableGB,
			"memory_usage_percent":     latest.MemoryUsagePercent,
			"network_info":             latest.NetworkInfo,
			"active_connections":       latest.ActiveConnections,
			"gpu_driver_version":       latest.GPUDriverVersion,
			"cuda_version":             latest.CUDAVersion,
			"gpu_count":                latest.GPUCount,
			"gpu_model":                latest.GPUModel,
			"gpu_memory_total":         latest.GPUMemoryTotal,
			"gpu_avg_utilization":      latest.GPUAvgUtilization,
			"gpu_memory_used_mb":       latest.GPUMemoryUsedMB,
			"gpu_memory_total_mb":      latest.GPUMemoryTotalMB,
			"gpu_memory_usage_percent": latest.GPUMemoryUsagePercent,
			"gpu_info":                 latest.GPUInfo,
			"ib_active_count":          latest.IBActiveCount,
			"ib_down_count":            latest.IBDownCount,
			"ib_total_count":           latest.IBTotalCount,
			"ib_ports_info":            latest.IBPortsInfo,
			"roce_info":                latest.RoCEInfo,
			"kernel_version":           latest.KernelVersion,
			"os_version":               latest.OSVersion,
			"uptime_seconds":           latest.UptimeSeconds,
			"raw_data":                 latest.RawData,
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
	var nodesWithIBDown int64
	var totalGPUs int64
	var totalIBPorts int64
	var totalIBDown int64

	// 统计节点总数
	s.db.Model(&models.NodeMetricsLatest{}).Count(&totalNodes)

	// 统计有 GPU 的节点数
	s.db.Model(&models.NodeMetricsLatest{}).Where("gpu_count > 0").Count(&nodesWithGPU)

	// 统计有活跃 IB 端口的节点数
	s.db.Model(&models.NodeMetricsLatest{}).Where("ib_active_count > 0").Count(&nodesWithIB)

	// 统计有 Down 状态 IB 端口的节点数
	s.db.Model(&models.NodeMetricsLatest{}).Where("ib_down_count > 0").Count(&nodesWithIBDown)

	// 统计 GPU 总数
	s.db.Model(&models.NodeMetricsLatest{}).Select("COALESCE(SUM(gpu_count), 0)").Scan(&totalGPUs)

	// 统计 IB 端口总数
	s.db.Model(&models.NodeMetricsLatest{}).Select("COALESCE(SUM(ib_total_count), 0)").Scan(&totalIBPorts)

	// 统计 Down 状态 IB 端口总数
	s.db.Model(&models.NodeMetricsLatest{}).Select("COALESCE(SUM(ib_down_count), 0)").Scan(&totalIBDown)

	return map[string]interface{}{
		"total_nodes":         totalNodes,
		"nodes_with_gpu":      nodesWithGPU,
		"nodes_with_ib":       nodesWithIB,
		"nodes_with_ib_down":  nodesWithIBDown,
		"total_gpus":          totalGPUs,
		"total_ib_ports":      totalIBPorts,
		"total_ib_down_ports": totalIBDown,
	}, nil
}

// ========== IB 端口忽略功能 ==========

// AddIBPortIgnore 添加 IB 端口忽略
func (s *NodeMetricsService) AddIBPortIgnore(minionID, portName string, portNum int, reason, createdBy string) error {
	ignore := &models.IBPortIgnore{
		MinionID:  minionID,
		PortName:  portName,
		PortNum:   portNum,
		Reason:    reason,
		CreatedBy: createdBy,
	}
	return s.db.Create(ignore).Error
}

// RemoveIBPortIgnore 移除 IB 端口忽略
// portNum 为 0 时删除所有匹配的端口号
func (s *NodeMetricsService) RemoveIBPortIgnore(minionID, portName string, portNum int) error {
	query := s.db.Where("minion_id = ? AND port_name = ?", minionID, portName)
	if portNum > 0 {
		query = query.Where("port_num = ?", portNum)
	}
	return query.Delete(&models.IBPortIgnore{}).Error
}

// GetIBPortIgnores 获取节点的 IB 端口忽略列表
func (s *NodeMetricsService) GetIBPortIgnores(minionID string) ([]models.IBPortIgnore, error) {
	var ignores []models.IBPortIgnore
	query := s.db.Model(&models.IBPortIgnore{})
	if minionID != "" {
		query = query.Where("minion_id = ?", minionID)
	}
	if err := query.Find(&ignores).Error; err != nil {
		return nil, err
	}
	return ignores, nil
}

// IsIBPortIgnored 检查 IB 端口是否被忽略
func (s *NodeMetricsService) IsIBPortIgnored(minionID, portName string, portNum int) (bool, string, error) {
	var ignore models.IBPortIgnore
	err := s.db.Where("minion_id = ? AND port_name = ? AND port_num = ?", minionID, portName, portNum).
		First(&ignore).Error
	if err == gorm.ErrRecordNotFound {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, ignore.Reason, nil
}

// GetIBPortAlerts 获取 IB 端口告警列表（Down 状态的端口）
func (s *NodeMetricsService) GetIBPortAlerts() ([]models.IBPortAlert, error) {
	// 获取所有有 Down 状态端口的节点
	var latestMetrics []models.NodeMetricsLatest
	if err := s.db.Where("ib_down_count > 0").Find(&latestMetrics).Error; err != nil {
		return nil, err
	}

	// 获取所有忽略规则
	ignores, err := s.GetIBPortIgnores("")
	if err != nil {
		return nil, err
	}

	// 构建忽略规则映射
	ignoreMap := make(map[string]string) // key: minionID|portName|portNum, value: reason
	for _, ig := range ignores {
		key := ig.MinionID + "|" + ig.PortName + "|" + string(rune(ig.PortNum+'0'))
		ignoreMap[key] = ig.Reason
	}

	var alerts []models.IBPortAlert
	for _, m := range latestMetrics {
		if m.IBPortsInfo == "" {
			continue
		}

		var ports []models.NodeIBPortInfo
		if err := json.Unmarshal([]byte(m.IBPortsInfo), &ports); err != nil {
			continue
		}

		for _, port := range ports {
			if port.State != "Active" {
				key := m.MinionID + "|" + port.Name + "|" + string(rune(port.Port+'0'))
				reason, isIgnored := ignoreMap[key]

				alerts = append(alerts, models.IBPortAlert{
					MinionID:      m.MinionID,
					PortName:      port.Name,
					PortNum:       port.Port,
					State:         port.State,
					PhysicalState: port.PhysicalState,
					Rate:          port.Rate,
					IsIgnored:     isIgnored,
					IgnoreReason:  reason,
				})
			}
		}
	}

	return alerts, nil
}
