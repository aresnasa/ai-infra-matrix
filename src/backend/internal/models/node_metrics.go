package models

import (
	"time"

	"gorm.io/gorm"
)

// NodeMetrics 节点指标表 - 存储从 Salt Minion 定期采集的 GPU/IB 等硬件信息
type NodeMetrics struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	MinionID  string    `json:"minion_id" gorm:"index;not null;size:255"` // Salt Minion ID
	Timestamp time.Time `json:"timestamp" gorm:"index;not null"`          // 采集时间戳

	// GPU 信息
	GPUDriverVersion string `json:"gpu_driver_version" gorm:"size:50"` // NVIDIA 驱动版本
	CUDAVersion      string `json:"cuda_version" gorm:"size:50"`       // CUDA 版本
	GPUCount         int    `json:"gpu_count" gorm:"default:0"`        // GPU 数量
	GPUModel         string `json:"gpu_model" gorm:"size:200"`         // GPU 型号
	GPUMemoryTotal   string `json:"gpu_memory_total" gorm:"size:50"`   // GPU 显存总量
	GPUInfo          string `json:"gpu_info" gorm:"type:text"`         // GPU 详细信息 JSON

	// InfiniBand 信息
	IBActiveCount int    `json:"ib_active_count" gorm:"default:0"` // 活跃 IB 端口数量
	IBPortsInfo   string `json:"ib_ports_info" gorm:"type:text"`   // IB 端口详细信息 JSON

	// 系统信息（可选）
	KernelVersion string `json:"kernel_version" gorm:"size:100"` // 内核版本
	OSVersion     string `json:"os_version" gorm:"size:100"`     // 操作系统版本

	// 原始数据
	RawData string `json:"raw_data" gorm:"type:text"` // 原始 JSON 数据

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (NodeMetrics) TableName() string {
	return "node_metrics"
}

// NodeMetricsLatest 节点最新指标视图（或表）
// 用于快速查询每个节点的最新指标，避免每次都要聚合查询
type NodeMetricsLatest struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	MinionID  string    `json:"minion_id" gorm:"uniqueIndex;not null;size:255"` // Salt Minion ID
	Timestamp time.Time `json:"timestamp" gorm:"index;not null"`                // 最近采集时间戳

	// GPU 信息
	GPUDriverVersion string `json:"gpu_driver_version" gorm:"size:50"`
	CUDAVersion      string `json:"cuda_version" gorm:"size:50"`
	GPUCount         int    `json:"gpu_count" gorm:"default:0"`
	GPUModel         string `json:"gpu_model" gorm:"size:200"`
	GPUMemoryTotal   string `json:"gpu_memory_total" gorm:"size:50"`
	GPUInfo          string `json:"gpu_info" gorm:"type:text"`

	// InfiniBand 信息
	IBActiveCount int    `json:"ib_active_count" gorm:"default:0"`
	IBPortsInfo   string `json:"ib_ports_info" gorm:"type:text"`

	// 系统信息
	KernelVersion string `json:"kernel_version" gorm:"size:100"`
	OSVersion     string `json:"os_version" gorm:"size:100"`

	// 原始数据
	RawData string `json:"raw_data" gorm:"type:text"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (NodeMetricsLatest) TableName() string {
	return "node_metrics_latest"
}

// NodeMetricsCallbackRequest 节点指标回调请求
type NodeMetricsCallbackRequest struct {
	MinionID  string             `json:"minion_id" binding:"required"`
	Timestamp string             `json:"timestamp"`
	GPU       *NodeGPUMetrics    `json:"gpu,omitempty"`
	IB        *NodeIBMetrics     `json:"ib,omitempty"`
	System    *NodeSystemMetrics `json:"system,omitempty"`
}

// NodeGPUMetrics GPU 指标
type NodeGPUMetrics struct {
	DriverVersion string              `json:"driver_version"`
	CUDAVersion   string              `json:"cuda_version"`
	Count         int                 `json:"count"`
	Model         string              `json:"model"`
	MemoryTotal   string              `json:"memory_total"`
	GPUs          []NodeGPUDetailInfo `json:"gpus,omitempty"` // 各 GPU 详细信息
}

// NodeGPUDetailInfo 单个 GPU 详细信息
type NodeGPUDetailInfo struct {
	Index       int    `json:"index"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	MemoryTotal string `json:"memory_total"`
	MemoryUsed  string `json:"memory_used"`
	MemoryFree  string `json:"memory_free"`
	Temperature int    `json:"temperature"`
	PowerDraw   string `json:"power_draw"`
	PowerLimit  string `json:"power_limit"`
	Utilization int    `json:"utilization"`
}

// NodeIBMetrics InfiniBand 指标
type NodeIBMetrics struct {
	ActiveCount int              `json:"active_count"`
	Ports       []NodeIBPortInfo `json:"ports,omitempty"`
}

// NodeIBPortInfo IB 端口信息
type NodeIBPortInfo struct {
	Name     string `json:"name"`     // 端口名称，如 mlx5_0
	State    string `json:"state"`    // 状态：Active, Down
	Rate     string `json:"rate"`     // 速率，如 400 Gb/sec
	Firmware string `json:"firmware"` // 固件版本
	GUID     string `json:"guid"`     // 端口 GUID
}

// NodeSystemMetrics 系统指标
type NodeSystemMetrics struct {
	KernelVersion string `json:"kernel_version"`
	OSVersion     string `json:"os_version"`
	Hostname      string `json:"hostname"`
	Uptime        string `json:"uptime"`
}

// NodeMetricsResponse 节点指标响应
type NodeMetricsResponse struct {
	MinionID    string             `json:"minion_id"`
	GPU         *NodeGPUMetrics    `json:"gpu,omitempty"`
	IB          *NodeIBMetrics     `json:"ib,omitempty"`
	System      *NodeSystemMetrics `json:"system,omitempty"`
	CollectedAt time.Time          `json:"collected_at"`
}
