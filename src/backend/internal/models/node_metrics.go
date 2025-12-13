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

	// CPU 信息
	CPUCores        int     `json:"cpu_cores" gorm:"default:0"`         // CPU 核心数
	CPUModel        string  `json:"cpu_model" gorm:"size:200"`          // CPU 型号
	CPUUsagePercent float64 `json:"cpu_usage_percent" gorm:"default:0"` // CPU 使用率
	CPULoadAvg      string  `json:"cpu_load_avg" gorm:"size:50"`        // CPU 负载

	// 内存信息
	MemoryTotalGB      float64 `json:"memory_total_gb" gorm:"default:0"`      // 内存总量 (GB)
	MemoryUsedGB       float64 `json:"memory_used_gb" gorm:"default:0"`       // 已用内存 (GB)
	MemoryAvailableGB  float64 `json:"memory_available_gb" gorm:"default:0"`  // 可用内存 (GB)
	MemoryUsagePercent float64 `json:"memory_usage_percent" gorm:"default:0"` // 内存使用率

	// 网络信息
	NetworkInfo       string `json:"network_info" gorm:"type:text"`       // 网络接口详情 JSON
	ActiveConnections int    `json:"active_connections" gorm:"default:0"` // 活跃连接数

	// GPU 信息
	GPUDriverVersion      string  `json:"gpu_driver_version" gorm:"size:50"`         // NVIDIA 驱动版本
	CUDAVersion           string  `json:"cuda_version" gorm:"size:50"`               // CUDA 版本
	GPUCount              int     `json:"gpu_count" gorm:"default:0"`                // GPU 数量
	GPUModel              string  `json:"gpu_model" gorm:"size:200"`                 // GPU 型号
	GPUMemoryTotal        string  `json:"gpu_memory_total" gorm:"size:50"`           // GPU 显存总量
	GPUAvgUtilization     float64 `json:"gpu_avg_utilization" gorm:"default:0"`      // GPU 平均利用率
	GPUMemoryUsedMB       int     `json:"gpu_memory_used_mb" gorm:"default:0"`       // GPU 已用显存 (MB)
	GPUMemoryTotalMB      int     `json:"gpu_memory_total_mb" gorm:"default:0"`      // GPU 总显存 (MB)
	GPUMemoryUsagePercent float64 `json:"gpu_memory_usage_percent" gorm:"default:0"` // GPU 显存使用率
	GPUInfo               string  `json:"gpu_info" gorm:"type:text"`                 // GPU 详细信息 JSON

	// NPU 信息 (华为昇腾、寒武纪等)
	NPUVendor         string  `json:"npu_vendor" gorm:"size:50"`            // NPU 厂商
	NPUVersion        string  `json:"npu_version" gorm:"size:50"`           // NPU 驱动/SMI 版本
	NPUCount          int     `json:"npu_count" gorm:"default:0"`           // NPU 数量
	NPUModel          string  `json:"npu_model" gorm:"size:200"`            // NPU 型号
	NPUAvgUtilization float64 `json:"npu_avg_utilization" gorm:"default:0"` // NPU 平均利用率
	NPUMemoryUsedMB   int     `json:"npu_memory_used_mb" gorm:"default:0"`  // NPU 已用显存 (MB)
	NPUMemoryTotalMB  int     `json:"npu_memory_total_mb" gorm:"default:0"` // NPU 总显存 (MB)
	NPUInfo           string  `json:"npu_info" gorm:"type:text"`            // NPU 详细信息 JSON

	// TPU 信息
	TPUVendor  string `json:"tpu_vendor" gorm:"size:50"`  // TPU 厂商
	TPUVersion string `json:"tpu_version" gorm:"size:50"` // TPU 驱动版本
	TPUCount   int    `json:"tpu_count" gorm:"default:0"` // TPU 数量
	TPUModel   string `json:"tpu_model" gorm:"size:200"`  // TPU 型号
	TPUInfo    string `json:"tpu_info" gorm:"type:text"`  // TPU 详细信息 JSON

	// InfiniBand 信息
	IBActiveCount int    `json:"ib_active_count" gorm:"default:0"` // 活跃 IB 端口数量
	IBDownCount   int    `json:"ib_down_count" gorm:"default:0"`   // Down 状态 IB 端口数量
	IBTotalCount  int    `json:"ib_total_count" gorm:"default:0"`  // IB 端口总数
	IBPortsInfo   string `json:"ib_ports_info" gorm:"type:text"`   // IB 端口详细信息 JSON

	// RoCE 信息
	RoCEInfo string `json:"roce_info" gorm:"type:text"` // RoCE 设备信息 JSON

	// 系统信息（可选）
	KernelVersion string `json:"kernel_version" gorm:"size:100"`  // 内核版本
	OSVersion     string `json:"os_version" gorm:"size:100"`      // 操作系统版本
	UptimeSeconds int    `json:"uptime_seconds" gorm:"default:0"` // 系统运行时长 (秒)

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

	// CPU 信息
	CPUCores        int     `json:"cpu_cores" gorm:"default:0"`
	CPUModel        string  `json:"cpu_model" gorm:"size:200"`
	CPUUsagePercent float64 `json:"cpu_usage_percent" gorm:"default:0"`
	CPULoadAvg      string  `json:"cpu_load_avg" gorm:"size:50"`

	// 内存信息
	MemoryTotalGB      float64 `json:"memory_total_gb" gorm:"default:0"`
	MemoryUsedGB       float64 `json:"memory_used_gb" gorm:"default:0"`
	MemoryAvailableGB  float64 `json:"memory_available_gb" gorm:"default:0"`
	MemoryUsagePercent float64 `json:"memory_usage_percent" gorm:"default:0"`

	// 网络信息
	NetworkInfo       string `json:"network_info" gorm:"type:text"`
	ActiveConnections int    `json:"active_connections" gorm:"default:0"`

	// GPU 信息
	GPUDriverVersion      string  `json:"gpu_driver_version" gorm:"size:50"`
	CUDAVersion           string  `json:"cuda_version" gorm:"size:50"`
	GPUCount              int     `json:"gpu_count" gorm:"default:0"`
	GPUModel              string  `json:"gpu_model" gorm:"size:200"`
	GPUMemoryTotal        string  `json:"gpu_memory_total" gorm:"size:50"`
	GPUAvgUtilization     float64 `json:"gpu_avg_utilization" gorm:"default:0"`
	GPUMemoryUsedMB       int     `json:"gpu_memory_used_mb" gorm:"default:0"`
	GPUMemoryTotalMB      int     `json:"gpu_memory_total_mb" gorm:"default:0"`
	GPUMemoryUsagePercent float64 `json:"gpu_memory_usage_percent" gorm:"default:0"`
	GPUInfo               string  `json:"gpu_info" gorm:"type:text"`

	// NPU 信息 (华为昇腾、寒武纪等)
	NPUVendor         string  `json:"npu_vendor" gorm:"size:50"`
	NPUVersion        string  `json:"npu_version" gorm:"size:50"`
	NPUCount          int     `json:"npu_count" gorm:"default:0"`
	NPUModel          string  `json:"npu_model" gorm:"size:200"`
	NPUAvgUtilization float64 `json:"npu_avg_utilization" gorm:"default:0"`
	NPUMemoryUsedMB   int     `json:"npu_memory_used_mb" gorm:"default:0"`
	NPUMemoryTotalMB  int     `json:"npu_memory_total_mb" gorm:"default:0"`
	NPUInfo           string  `json:"npu_info" gorm:"type:text"`

	// TPU 信息
	TPUVendor  string `json:"tpu_vendor" gorm:"size:50"`
	TPUVersion string `json:"tpu_version" gorm:"size:50"`
	TPUCount   int    `json:"tpu_count" gorm:"default:0"`
	TPUModel   string `json:"tpu_model" gorm:"size:200"`
	TPUInfo    string `json:"tpu_info" gorm:"type:text"`

	// InfiniBand 信息
	IBActiveCount int    `json:"ib_active_count" gorm:"default:0"`
	IBDownCount   int    `json:"ib_down_count" gorm:"default:0"`
	IBTotalCount  int    `json:"ib_total_count" gorm:"default:0"`
	IBPortsInfo   string `json:"ib_ports_info" gorm:"type:text"`

	// RoCE 信息
	RoCEInfo string `json:"roce_info" gorm:"type:text"`

	// 系统信息
	KernelVersion string `json:"kernel_version" gorm:"size:100"`
	OSVersion     string `json:"os_version" gorm:"size:100"`
	UptimeSeconds int    `json:"uptime_seconds" gorm:"default:0"`

	// 原始数据
	RawData string `json:"raw_data" gorm:"type:text"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (NodeMetricsLatest) TableName() string {
	return "node_metrics_latest"
}

// IBPortIgnore IB 端口忽略表 - 用于标记不需要告警的 IB 端口
type IBPortIgnore struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	MinionID  string    `json:"minion_id" gorm:"index;not null;size:255"` // Salt Minion ID
	PortName  string    `json:"port_name" gorm:"not null;size:100"`       // 端口名称，如 mlx5_0
	PortNum   int       `json:"port_num" gorm:"default:1"`                // 端口号
	Reason    string    `json:"reason" gorm:"size:500"`                   // 忽略原因
	CreatedBy string    `json:"created_by" gorm:"size:100"`               // 创建者
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (IBPortIgnore) TableName() string {
	return "ib_port_ignore"
}

// IBPortAlert IB 端口告警信息
type IBPortAlert struct {
	MinionID      string `json:"minion_id"`
	PortName      string `json:"port_name"`
	PortNum       int    `json:"port_num"`
	State         string `json:"state"`
	PhysicalState string `json:"physical_state"`
	Rate          string `json:"rate"`
	IsIgnored     bool   `json:"is_ignored"`
	IgnoreReason  string `json:"ignore_reason,omitempty"`
}

// NodeMetricsCallbackRequest 节点指标回调请求
type NodeMetricsCallbackRequest struct {
	MinionID  string              `json:"minion_id" binding:"required"`
	Timestamp string              `json:"timestamp"`
	CPU       *NodeCPUMetrics     `json:"cpu,omitempty"`
	Memory    *NodeMemoryMetrics  `json:"memory,omitempty"`
	Network   *NodeNetworkMetrics `json:"network,omitempty"`
	GPU       *NodeGPUMetrics     `json:"gpu,omitempty"`
	NPU       *NodeNPUMetrics     `json:"npu,omitempty"`
	TPU       *NodeTPUMetrics     `json:"tpu,omitempty"`
	IB        *NodeIBMetrics      `json:"ib,omitempty"`
	RoCE      *NodeRoCEMetrics    `json:"roce,omitempty"`
	System    *NodeSystemMetrics  `json:"system,omitempty"`
}

// NodeCPUMetrics CPU 指标
type NodeCPUMetrics struct {
	Cores        int     `json:"cores"`
	Model        string  `json:"model"`
	UsagePercent float64 `json:"usage_percent"`
	Usage        float64 `json:"usage"` // 别名，与 UsagePercent 相同值，兼容前端
	LoadAvg      string  `json:"load_avg"`
}

// NodeMemoryMetrics 内存指标
type NodeMemoryMetrics struct {
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	AvailableGB  float64 `json:"available_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

// NodeNetworkMetrics 网络指标
type NodeNetworkMetrics struct {
	Interfaces        []NodeNetworkInterface `json:"interfaces,omitempty"`
	ActiveConnections int                    `json:"active_connections"`
}

// NodeNetworkInterface 网络接口信息
type NodeNetworkInterface struct {
	Name          string `json:"name"`
	State         string `json:"state"`
	IP            string `json:"ip"`
	SpeedMbps     int    `json:"speed_mbps"`
	RxBytesPerSec int64  `json:"rx_bytes_per_sec"`
	TxBytesPerSec int64  `json:"tx_bytes_per_sec"`
}

// NodeGPUMetrics GPU 指标
type NodeGPUMetrics struct {
	DriverVersion      string              `json:"driver_version"`
	CUDAVersion        string              `json:"cuda_version"`
	Count              int                 `json:"count"`
	Model              string              `json:"model"`
	MemoryTotal        string              `json:"memory_total"`
	AvgUtilization     float64             `json:"avg_utilization"`
	MemoryUsedMB       int                 `json:"memory_used_mb"`
	MemoryTotalMB      int                 `json:"memory_total_mb"`
	MemoryUsagePercent float64             `json:"memory_usage_percent"`
	GPUs               []NodeGPUDetailInfo `json:"gpus,omitempty"` // 各 GPU 详细信息
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

// NodeNPUMetrics NPU 指标 (华为昇腾、寒武纪等)
type NodeNPUMetrics struct {
	Vendor         string              `json:"vendor"`          // 厂商: huawei, cambricon, iluvatar
	Version        string              `json:"version"`         // 驱动/SMI 版本
	Count          int                 `json:"count"`           // NPU 数量
	Model          string              `json:"model"`           // NPU 型号
	AvgUtilization float64             `json:"avg_utilization"` // 平均利用率
	MemoryUsedMB   int                 `json:"memory_used_mb"`  // 已用显存 (MB)
	MemoryTotalMB  int                 `json:"memory_total_mb"` // 总显存 (MB)
	NPUs           []NodeNPUDetailInfo `json:"npus,omitempty"`  // 各 NPU 详细信息
}

// NodeNPUDetailInfo 单个 NPU 详细信息
type NodeNPUDetailInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Health      string `json:"health"`       // OK, Warning, Error
	Power       string `json:"power"`        // 功耗 (W)
	Temperature int    `json:"temperature"`  // 温度 (C)
	Utilization int    `json:"utilization"`  // 利用率 %
	MemoryUsed  string `json:"memory_used"`  // HBM 已用
	MemoryTotal string `json:"memory_total"` // HBM 总量
	BusID       string `json:"bus_id"`       // 总线 ID
}

// NodeTPUMetrics TPU 指标 (Google TPU 或其他 AI 加速器)
type NodeTPUMetrics struct {
	Vendor  string              `json:"vendor"`         // 厂商
	Version string              `json:"version"`        // 驱动版本
	Count   int                 `json:"count"`          // TPU 数量
	Model   string              `json:"model"`          // TPU 型号
	TPUs    []NodeTPUDetailInfo `json:"tpus,omitempty"` // 各 TPU 详细信息
}

// NodeTPUDetailInfo 单个 TPU 详细信息
type NodeTPUDetailInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Health      string `json:"health"`
	Utilization int    `json:"utilization"`
}

// NodeIBMetrics InfiniBand 指标
type NodeIBMetrics struct {
	ActiveCount int              `json:"active_count"`
	DownCount   int              `json:"down_count"`
	TotalCount  int              `json:"total_count"`
	Ports       []NodeIBPortInfo `json:"ports,omitempty"`
}

// NodeIBPortInfo IB 端口信息
type NodeIBPortInfo struct {
	Name          string `json:"name"`           // CA 名称，如 mlx5_0
	Port          int    `json:"port"`           // 端口号
	CAType        string `json:"ca_type"`        // CA 类型，如 MT4129
	Firmware      string `json:"firmware"`       // 固件版本
	State         string `json:"state"`          // 状态：Active, Down
	PhysicalState string `json:"physical_state"` // 物理状态：LinkUp, Polling
	Rate          string `json:"rate"`           // 速率，如 400
	GUID          string `json:"guid"`           // 端口 GUID
}

// NodeRoCEMetrics RoCE 网络指标
type NodeRoCEMetrics struct {
	Devices []NodeRoCEDevice `json:"devices,omitempty"`
}

// NodeRoCEDevice RoCE 设备信息
type NodeRoCEDevice struct {
	Name     string `json:"name"`
	NodeType string `json:"node_type"`
	NodeGUID string `json:"node_guid"`
	IsRoCE   bool   `json:"is_roce"`
	State    string `json:"state"`
	GID      string `json:"gid"`
}

// NodeSystemMetrics 系统指标
type NodeSystemMetrics struct {
	KernelVersion string `json:"kernel_version"`
	OSVersion     string `json:"os_version"`
	Hostname      string `json:"hostname"`
	Uptime        string `json:"uptime"`
	UptimeSeconds int    `json:"uptime_seconds"`
}

// NodeMetricsResponse 节点指标响应
type NodeMetricsResponse struct {
	MinionID    string              `json:"minion_id"`
	CPU         *NodeCPUMetrics     `json:"cpu,omitempty"`
	Memory      *NodeMemoryMetrics  `json:"memory,omitempty"`
	Network     *NodeNetworkMetrics `json:"network,omitempty"`
	GPU         *NodeGPUMetrics     `json:"gpu,omitempty"`
	NPU         *NodeNPUMetrics     `json:"npu,omitempty"`
	TPU         *NodeTPUMetrics     `json:"tpu,omitempty"`
	IB          *NodeIBMetrics      `json:"ib,omitempty"`
	RoCE        *NodeRoCEMetrics    `json:"roce,omitempty"`
	System      *NodeSystemMetrics  `json:"system,omitempty"`
	CollectedAt time.Time           `json:"collected_at"`
}
