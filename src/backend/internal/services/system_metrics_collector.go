package services

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DeploymentType 部署类型
type DeploymentType string

const (
	DeploymentTypeContainer DeploymentType = "container"  // 容器环境 (Docker, Kubernetes)
	DeploymentTypeVM        DeploymentType = "vm"         // 虚拟机
	DeploymentTypeBareMetal DeploymentType = "bare_metal" // 物理机
	DeploymentTypeUnknown   DeploymentType = "unknown"    // 未知环境
)

// MetricsSource 指标来源
type MetricsSource string

const (
	MetricsSourceLocal      MetricsSource = "local"      // 本地采集 (/proc, /sys)
	MetricsSourceDocker     MetricsSource = "docker"     // Docker API
	MetricsSourceCgroup     MetricsSource = "cgroup"     // cgroup 文件系统
	MetricsSourceSalt       MetricsSource = "salt"       // Salt API
	MetricsSourcePrometheus MetricsSource = "prometheus" // Prometheus/VictoriaMetrics
	MetricsSourceExternal   MetricsSource = "external"   // 外部采集器 (Categraf等)
)

// SystemMetrics 系统指标结构
type SystemMetrics struct {
	CPUUsagePercent      float64        `json:"cpu_usage_percent"`      // CPU 使用率 (%)
	MemoryUsagePercent   float64        `json:"memory_usage_percent"`   // 内存使用率 (%)
	MemoryTotalBytes     int64          `json:"memory_total_bytes"`     // 内存总量 (bytes)
	MemoryUsedBytes      int64          `json:"memory_used_bytes"`      // 已用内存 (bytes)
	MemoryAvailableBytes int64          `json:"memory_available_bytes"` // 可用内存 (bytes)
	NetworkInBytes       int64          `json:"network_in_bytes"`       // 网络入流量 (bytes)
	NetworkOutBytes      int64          `json:"network_out_bytes"`      // 网络出流量 (bytes)
	NetworkBandwidthIn   float64        `json:"network_bandwidth_in"`   // 网络入带宽 (bytes/s)
	NetworkBandwidthOut  float64        `json:"network_bandwidth_out"`  // 网络出带宽 (bytes/s)
	ActiveConnections    int            `json:"active_connections"`     // 活跃连接数
	LoadAvg1             float64        `json:"load_avg_1"`             // 1分钟负载
	LoadAvg5             float64        `json:"load_avg_5"`             // 5分钟负载
	LoadAvg15            float64        `json:"load_avg_15"`            // 15分钟负载
	UptimeSeconds        int64          `json:"uptime_seconds"`         // 运行时间 (秒)
	CPUCores             int            `json:"cpu_cores"`              // CPU 核心数
	DeploymentType       DeploymentType `json:"deployment_type"`        // 部署类型
	MetricsSource        MetricsSource  `json:"metrics_source"`         // 指标来源
	CollectedAt          time.Time      `json:"collected_at"`           // 采集时间
	Error                string         `json:"error,omitempty"`        // 错误信息
}

// MetricsCollector 指标采集器接口
type MetricsCollector interface {
	// Collect 采集系统指标
	Collect() (*SystemMetrics, error)
	// Source 返回采集器来源类型
	Source() MetricsSource
	// IsAvailable 检查采集器是否可用
	IsAvailable() bool
}

// SystemMetricsCollectorService 系统指标采集服务
// 自动检测环境并选择合适的采集方式
type SystemMetricsCollectorService struct {
	deploymentType   DeploymentType
	collectors       []MetricsCollector
	primaryCollector MetricsCollector
	mu               sync.RWMutex
	lastMetrics      *SystemMetrics
	lastCollectTime  time.Time
	cacheTTL         time.Duration
}

var (
	systemMetricsCollector     *SystemMetricsCollectorService
	systemMetricsCollectorOnce sync.Once
)

// NewSystemMetricsCollectorService 创建系统指标采集服务
func NewSystemMetricsCollectorService() *SystemMetricsCollectorService {
	systemMetricsCollectorOnce.Do(func() {
		svc := &SystemMetricsCollectorService{
			cacheTTL: 5 * time.Second, // 默认5秒缓存
		}
		svc.init()
		systemMetricsCollector = svc
	})
	return systemMetricsCollector
}

// init 初始化采集服务
func (s *SystemMetricsCollectorService) init() {
	// 检测部署类型
	s.deploymentType = s.detectDeploymentType()
	logrus.WithField("deployment_type", s.deploymentType).Info("[SystemMetricsCollector] 检测到部署环境类型")

	// 根据部署类型注册采集器（按优先级排序）
	s.collectors = make([]MetricsCollector, 0)

	switch s.deploymentType {
	case DeploymentTypeContainer:
		// 容器环境：优先使用 cgroup，然后 /proc
		s.collectors = append(s.collectors, NewCgroupCollector())
		s.collectors = append(s.collectors, NewProcCollector())
	case DeploymentTypeVM, DeploymentTypeBareMetal:
		// VM/物理机：优先使用 /proc
		s.collectors = append(s.collectors, NewProcCollector())
	default:
		// 未知环境：尝试所有方式
		s.collectors = append(s.collectors, NewProcCollector())
		s.collectors = append(s.collectors, NewCgroupCollector())
	}

	// 选择第一个可用的采集器作为主采集器
	for _, collector := range s.collectors {
		if collector.IsAvailable() {
			s.primaryCollector = collector
			logrus.WithFields(logrus.Fields{
				"source":          collector.Source(),
				"deployment_type": s.deploymentType,
			}).Info("[SystemMetricsCollector] 选择主采集器")
			break
		}
	}

	if s.primaryCollector == nil {
		logrus.Warn("[SystemMetricsCollector] 未找到可用的采集器")
	}
}

// detectDeploymentType 检测部署类型
func (s *SystemMetricsCollectorService) detectDeploymentType() DeploymentType {
	// 检查是否在容器中运行

	// 方法1: 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		logrus.Debug("[SystemMetricsCollector] 检测到 /.dockerenv，运行在 Docker 容器中")
		return DeploymentTypeContainer
	}

	// 方法2: 检查 /proc/1/cgroup 中是否包含 docker 或 kubepods
	if content, err := ioutil.ReadFile("/proc/1/cgroup"); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "docker") ||
			strings.Contains(contentStr, "kubepods") ||
			strings.Contains(contentStr, "containerd") {
			logrus.Debug("[SystemMetricsCollector] 检测到 cgroup 包含容器标识，运行在容器中")
			return DeploymentTypeContainer
		}
	}

	// 方法3: 检查 /proc/self/cgroup
	if content, err := ioutil.ReadFile("/proc/self/cgroup"); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "docker") ||
			strings.Contains(contentStr, "kubepods") ||
			strings.Contains(contentStr, "containerd") {
			return DeploymentTypeContainer
		}
	}

	// 方法4: 检查环境变量
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return DeploymentTypeContainer
	}

	// 检查是否是虚拟机
	if s.isVirtualMachine() {
		return DeploymentTypeVM
	}

	// 默认认为是物理机
	return DeploymentTypeBareMetal
}

// isVirtualMachine 检测是否是虚拟机
func (s *SystemMetricsCollectorService) isVirtualMachine() bool {
	// 检查 /sys/class/dmi/id/product_name
	if content, err := ioutil.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		productName := strings.ToLower(string(content))
		vmIndicators := []string{"virtual", "vmware", "kvm", "qemu", "xen", "hyper-v", "virtualbox"}
		for _, indicator := range vmIndicators {
			if strings.Contains(productName, indicator) {
				return true
			}
		}
	}

	// 检查 systemd-detect-virt 命令
	if output, err := exec.Command("systemd-detect-virt").Output(); err == nil {
		result := strings.TrimSpace(string(output))
		if result != "none" && result != "" {
			return true
		}
	}

	return false
}

// Collect 采集系统指标（带缓存）
func (s *SystemMetricsCollectorService) Collect() (*SystemMetrics, error) {
	s.mu.RLock()
	if s.lastMetrics != nil && time.Since(s.lastCollectTime) < s.cacheTTL {
		metrics := s.lastMetrics
		s.mu.RUnlock()
		return metrics, nil
	}
	s.mu.RUnlock()

	return s.CollectFresh()
}

// CollectFresh 强制重新采集（不使用缓存）
func (s *SystemMetricsCollectorService) CollectFresh() (*SystemMetrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.primaryCollector == nil {
		return &SystemMetrics{
			DeploymentType: s.deploymentType,
			CollectedAt:    time.Now(),
			Error:          "no available collector",
		}, fmt.Errorf("no available collector")
	}

	metrics, err := s.primaryCollector.Collect()
	if err != nil {
		// 尝试备用采集器
		for _, collector := range s.collectors {
			if collector != s.primaryCollector && collector.IsAvailable() {
				metrics, err = collector.Collect()
				if err == nil {
					break
				}
			}
		}
	}

	if metrics != nil {
		metrics.DeploymentType = s.deploymentType
		metrics.CollectedAt = time.Now()
		s.lastMetrics = metrics
		s.lastCollectTime = time.Now()
	}

	return metrics, err
}

// GetDeploymentType 获取部署类型
func (s *SystemMetricsCollectorService) GetDeploymentType() DeploymentType {
	return s.deploymentType
}

// SetCacheTTL 设置缓存过期时间
func (s *SystemMetricsCollectorService) SetCacheTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
}

// ============================================================================
// ProcCollector - 从 /proc 文件系统采集指标
// ============================================================================

// ProcCollector 从 /proc 文件系统采集系统指标
type ProcCollector struct {
	lastCPUStats     cpuStats
	lastCPUTime      time.Time
	lastNetworkStats networkStats
	lastNetworkTime  time.Time
	mu               sync.Mutex
}

type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	total   uint64
}

type networkStats struct {
	rxBytes uint64
	txBytes uint64
}

// NewProcCollector 创建 /proc 采集器
func NewProcCollector() *ProcCollector {
	return &ProcCollector{}
}

// Source 返回采集器来源
func (c *ProcCollector) Source() MetricsSource {
	return MetricsSourceLocal
}

// IsAvailable 检查采集器是否可用
func (c *ProcCollector) IsAvailable() bool {
	_, err := os.Stat("/proc/stat")
	return err == nil
}

// Collect 采集系统指标
func (c *ProcCollector) Collect() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		MetricsSource: MetricsSourceLocal,
		CPUCores:      runtime.NumCPU(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]string, 0)

	// 并行采集各项指标
	wg.Add(5)

	// CPU 使用率
	go func() {
		defer wg.Done()
		if cpuUsage, err := c.collectCPUUsage(); err == nil {
			mu.Lock()
			metrics.CPUUsagePercent = cpuUsage
			mu.Unlock()
		} else {
			mu.Lock()
			errors = append(errors, fmt.Sprintf("cpu: %v", err))
			mu.Unlock()
		}
	}()

	// 内存使用率
	go func() {
		defer wg.Done()
		if memInfo, err := c.collectMemoryInfo(); err == nil {
			mu.Lock()
			metrics.MemoryTotalBytes = memInfo.total
			metrics.MemoryUsedBytes = memInfo.used
			metrics.MemoryAvailableBytes = memInfo.available
			if memInfo.total > 0 {
				metrics.MemoryUsagePercent = float64(memInfo.used) / float64(memInfo.total) * 100
			}
			mu.Unlock()
		} else {
			mu.Lock()
			errors = append(errors, fmt.Sprintf("memory: %v", err))
			mu.Unlock()
		}
	}()

	// 网络流量
	go func() {
		defer wg.Done()
		if netStats, bwIn, bwOut, err := c.collectNetworkStats(); err == nil {
			mu.Lock()
			metrics.NetworkInBytes = int64(netStats.rxBytes)
			metrics.NetworkOutBytes = int64(netStats.txBytes)
			metrics.NetworkBandwidthIn = bwIn
			metrics.NetworkBandwidthOut = bwOut
			mu.Unlock()
		} else {
			mu.Lock()
			errors = append(errors, fmt.Sprintf("network: %v", err))
			mu.Unlock()
		}
	}()

	// 负载平均值
	go func() {
		defer wg.Done()
		if load1, load5, load15, err := c.collectLoadAvg(); err == nil {
			mu.Lock()
			metrics.LoadAvg1 = load1
			metrics.LoadAvg5 = load5
			metrics.LoadAvg15 = load15
			mu.Unlock()
		} else {
			mu.Lock()
			errors = append(errors, fmt.Sprintf("loadavg: %v", err))
			mu.Unlock()
		}
	}()

	// 活跃连接数和运行时间
	go func() {
		defer wg.Done()
		if conns, err := c.collectActiveConnections(); err == nil {
			mu.Lock()
			metrics.ActiveConnections = conns
			mu.Unlock()
		}
		if uptime, err := c.collectUptime(); err == nil {
			mu.Lock()
			metrics.UptimeSeconds = uptime
			mu.Unlock()
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		metrics.Error = strings.Join(errors, "; ")
	}

	return metrics, nil
}

// collectCPUUsage 计算 CPU 使用率
func (c *ProcCollector) collectCPUUsage() (float64, error) {
	content, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return 0, fmt.Errorf("invalid /proc/stat format")
			}

			user, _ := strconv.ParseUint(fields[1], 10, 64)
			nice, _ := strconv.ParseUint(fields[2], 10, 64)
			system, _ := strconv.ParseUint(fields[3], 10, 64)
			idle, _ := strconv.ParseUint(fields[4], 10, 64)
			iowait, _ := strconv.ParseUint(fields[5], 10, 64)
			irq, _ := strconv.ParseUint(fields[6], 10, 64)
			softirq, _ := strconv.ParseUint(fields[7], 10, 64)

			total := user + nice + system + idle + iowait + irq + softirq

			current := cpuStats{
				user:    user,
				nice:    nice,
				system:  system,
				idle:    idle,
				iowait:  iowait,
				irq:     irq,
				softirq: softirq,
				total:   total,
			}

			c.mu.Lock()
			defer c.mu.Unlock()

			// 第一次采集，存储并返回 0
			if c.lastCPUTime.IsZero() {
				c.lastCPUStats = current
				c.lastCPUTime = time.Now()
				return 0, nil
			}

			// 计算增量
			totalDelta := current.total - c.lastCPUStats.total
			idleDelta := current.idle - c.lastCPUStats.idle + current.iowait - c.lastCPUStats.iowait

			c.lastCPUStats = current
			c.lastCPUTime = time.Now()

			if totalDelta == 0 {
				return 0, nil
			}

			cpuUsage := 100 * (1 - float64(idleDelta)/float64(totalDelta))
			if cpuUsage < 0 {
				cpuUsage = 0
			}
			if cpuUsage > 100 {
				cpuUsage = 100
			}

			return cpuUsage, nil
		}
	}

	return 0, fmt.Errorf("cpu line not found in /proc/stat")
}

type memInfo struct {
	total     int64
	used      int64
	available int64
}

// collectMemoryInfo 采集内存信息
func (c *ProcCollector) collectMemoryInfo() (*memInfo, error) {
	content, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	info := &memInfo{}
	var free, buffers, cached int64

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseInt(fields[1], 10, 64)
		value *= 1024 // 转换为 bytes (meminfo 单位是 kB)

		switch key {
		case "MemTotal":
			info.total = value
		case "MemFree":
			free = value
		case "MemAvailable":
			info.available = value
		case "Buffers":
			buffers = value
		case "Cached":
			cached = value
		}
	}

	// 如果没有 MemAvailable (老内核)，计算估算值
	if info.available == 0 {
		info.available = free + buffers + cached
	}

	info.used = info.total - info.available

	return info, nil
}

// collectNetworkStats 采集网络流量统计
func (c *ProcCollector) collectNetworkStats() (networkStats, float64, float64, error) {
	content, err := ioutil.ReadFile("/proc/net/dev")
	if err != nil {
		return networkStats{}, 0, 0, err
	}

	var stats networkStats
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			continue
		}

		// 跳过 lo 接口
		if strings.HasPrefix(line, "lo:") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 10 {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)

		stats.rxBytes += rxBytes
		stats.txBytes += txBytes
	}

	// 计算带宽
	c.mu.Lock()
	defer c.mu.Unlock()

	var bwIn, bwOut float64

	if !c.lastNetworkTime.IsZero() {
		elapsed := time.Since(c.lastNetworkTime).Seconds()
		if elapsed > 0 {
			bwIn = float64(stats.rxBytes-c.lastNetworkStats.rxBytes) / elapsed
			bwOut = float64(stats.txBytes-c.lastNetworkStats.txBytes) / elapsed
		}
	}

	c.lastNetworkStats = stats
	c.lastNetworkTime = time.Now()

	return stats, bwIn, bwOut, nil
}

// collectLoadAvg 采集负载平均值
func (c *ProcCollector) collectLoadAvg() (float64, float64, float64, error) {
	content, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid /proc/loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return load1, load5, load15, nil
}

// collectActiveConnections 采集活跃连接数
func (c *ProcCollector) collectActiveConnections() (int, error) {
	// 统计 TCP ESTABLISHED 连接数
	count := 0

	// TCP IPv4
	if content, err := ioutil.ReadFile("/proc/net/tcp"); err == nil {
		count += c.countEstablishedConnections(string(content))
	}

	// TCP IPv6
	if content, err := ioutil.ReadFile("/proc/net/tcp6"); err == nil {
		count += c.countEstablishedConnections(string(content))
	}

	return count, nil
}

// countEstablishedConnections 统计 ESTABLISHED 状态的连接数
func (c *ProcCollector) countEstablishedConnections(content string) int {
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Scan() // 跳过标题行

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		// 状态字段是第 4 个，01 表示 ESTABLISHED
		if fields[3] == "01" {
			count++
		}
	}

	return count
}

// collectUptime 采集系统运行时间
func (c *ProcCollector) collectUptime() (int64, error) {
	content, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid /proc/uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return int64(uptime), nil
}

// ============================================================================
// CgroupCollector - 从 cgroup 文件系统采集指标（容器环境）
// ============================================================================

// CgroupCollector 从 cgroup 采集容器指标
type CgroupCollector struct {
	cgroupVersion int // 1 或 2
	cgroupPath    string
	procCollector *ProcCollector // 用于采集 cgroup 无法提供的指标
	lastCPUUsage  uint64
	lastCPUTime   time.Time
	mu            sync.Mutex
}

// NewCgroupCollector 创建 cgroup 采集器
func NewCgroupCollector() *CgroupCollector {
	c := &CgroupCollector{
		procCollector: NewProcCollector(),
	}
	c.detectCgroupVersion()
	return c
}

// detectCgroupVersion 检测 cgroup 版本
func (c *CgroupCollector) detectCgroupVersion() {
	// 检查 cgroup v2
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		c.cgroupVersion = 2
		c.cgroupPath = "/sys/fs/cgroup"
		logrus.Debug("[CgroupCollector] 检测到 cgroup v2")
		return
	}

	// 检查 cgroup v1
	if _, err := os.Stat("/sys/fs/cgroup/cpu/cpuacct.usage"); err == nil {
		c.cgroupVersion = 1
		c.cgroupPath = "/sys/fs/cgroup"
		logrus.Debug("[CgroupCollector] 检测到 cgroup v1")
		return
	}

	// 检查容器内的 cgroup 路径
	if _, err := os.Stat("/sys/fs/cgroup/cpu.stat"); err == nil {
		c.cgroupVersion = 2
		c.cgroupPath = "/sys/fs/cgroup"
		return
	}

	c.cgroupVersion = 0
}

// Source 返回采集器来源
func (c *CgroupCollector) Source() MetricsSource {
	return MetricsSourceCgroup
}

// IsAvailable 检查采集器是否可用
func (c *CgroupCollector) IsAvailable() bool {
	return c.cgroupVersion > 0
}

// Collect 采集容器指标
func (c *CgroupCollector) Collect() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		MetricsSource: MetricsSourceCgroup,
		CPUCores:      runtime.NumCPU(),
	}

	var errors []string

	// CPU 使用率
	if cpuUsage, err := c.collectCPUUsage(); err == nil {
		metrics.CPUUsagePercent = cpuUsage
	} else {
		errors = append(errors, fmt.Sprintf("cpu: %v", err))
		// 回退到 /proc
		if cpuUsage, err := c.procCollector.collectCPUUsage(); err == nil {
			metrics.CPUUsagePercent = cpuUsage
		}
	}

	// 内存使用率
	if memInfo, err := c.collectMemoryUsage(); err == nil {
		metrics.MemoryTotalBytes = memInfo.total
		metrics.MemoryUsedBytes = memInfo.used
		metrics.MemoryAvailableBytes = memInfo.available
		if memInfo.total > 0 {
			metrics.MemoryUsagePercent = float64(memInfo.used) / float64(memInfo.total) * 100
		}
	} else {
		errors = append(errors, fmt.Sprintf("memory: %v", err))
		// 回退到 /proc
		if memInfo, err := c.procCollector.collectMemoryInfo(); err == nil {
			metrics.MemoryTotalBytes = memInfo.total
			metrics.MemoryUsedBytes = memInfo.used
			metrics.MemoryAvailableBytes = memInfo.available
			if memInfo.total > 0 {
				metrics.MemoryUsagePercent = float64(memInfo.used) / float64(memInfo.total) * 100
			}
		}
	}

	// 网络和负载从 /proc 采集 (cgroup 不提供这些)
	if netStats, bwIn, bwOut, err := c.procCollector.collectNetworkStats(); err == nil {
		metrics.NetworkInBytes = int64(netStats.rxBytes)
		metrics.NetworkOutBytes = int64(netStats.txBytes)
		metrics.NetworkBandwidthIn = bwIn
		metrics.NetworkBandwidthOut = bwOut
	}

	if load1, load5, load15, err := c.procCollector.collectLoadAvg(); err == nil {
		metrics.LoadAvg1 = load1
		metrics.LoadAvg5 = load5
		metrics.LoadAvg15 = load15
	}

	if conns, err := c.procCollector.collectActiveConnections(); err == nil {
		metrics.ActiveConnections = conns
	}

	if uptime, err := c.procCollector.collectUptime(); err == nil {
		metrics.UptimeSeconds = uptime
	}

	if len(errors) > 0 {
		metrics.Error = strings.Join(errors, "; ")
	}

	return metrics, nil
}

// collectCPUUsage 从 cgroup 采集 CPU 使用率
func (c *CgroupCollector) collectCPUUsage() (float64, error) {
	var usage uint64
	var err error

	if c.cgroupVersion == 2 {
		usage, err = c.readCgroupV2CPUUsage()
	} else {
		usage, err = c.readCgroupV1CPUUsage()
	}

	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 第一次采集
	if c.lastCPUTime.IsZero() {
		c.lastCPUUsage = usage
		c.lastCPUTime = time.Now()
		return 0, nil
	}

	elapsed := time.Since(c.lastCPUTime).Nanoseconds()
	usageDelta := usage - c.lastCPUUsage

	c.lastCPUUsage = usage
	c.lastCPUTime = time.Now()

	if elapsed == 0 {
		return 0, nil
	}

	// CPU 使用率 = (CPU 时间增量 / 实际时间增量) / CPU 核心数 * 100
	cpuUsage := float64(usageDelta) / float64(elapsed) * 100
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	return cpuUsage, nil
}

// readCgroupV2CPUUsage 读取 cgroup v2 CPU 使用时间
func (c *CgroupCollector) readCgroupV2CPUUsage() (uint64, error) {
	content, err := ioutil.ReadFile(c.cgroupPath + "/cpu.stat")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "usage_usec" {
			usage, err := strconv.ParseUint(fields[1], 10, 64)
			return usage * 1000, err // 转换为纳秒
		}
	}

	return 0, fmt.Errorf("usage_usec not found in cpu.stat")
}

// readCgroupV1CPUUsage 读取 cgroup v1 CPU 使用时间
func (c *CgroupCollector) readCgroupV1CPUUsage() (uint64, error) {
	content, err := ioutil.ReadFile(c.cgroupPath + "/cpu/cpuacct.usage")
	if err != nil {
		// 尝试其他路径
		content, err = ioutil.ReadFile("/sys/fs/cgroup/cpuacct/cpuacct.usage")
		if err != nil {
			return 0, err
		}
	}

	usage, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	return usage, err
}

// collectMemoryUsage 从 cgroup 采集内存使用
func (c *CgroupCollector) collectMemoryUsage() (*memInfo, error) {
	if c.cgroupVersion == 2 {
		return c.readCgroupV2Memory()
	}
	return c.readCgroupV1Memory()
}

// readCgroupV2Memory 读取 cgroup v2 内存使用
func (c *CgroupCollector) readCgroupV2Memory() (*memInfo, error) {
	info := &memInfo{}

	// 读取内存限制
	if content, err := ioutil.ReadFile(c.cgroupPath + "/memory.max"); err == nil {
		limitStr := strings.TrimSpace(string(content))
		if limitStr != "max" {
			info.total, _ = strconv.ParseInt(limitStr, 10, 64)
		}
	}

	// 如果没有限制或是 max，使用系统总内存
	if info.total == 0 {
		if procMemInfo, err := c.procCollector.collectMemoryInfo(); err == nil {
			info.total = procMemInfo.total
		}
	}

	// 读取当前使用
	if content, err := ioutil.ReadFile(c.cgroupPath + "/memory.current"); err == nil {
		info.used, _ = strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
	}

	info.available = info.total - info.used
	if info.available < 0 {
		info.available = 0
	}

	return info, nil
}

// readCgroupV1Memory 读取 cgroup v1 内存使用
func (c *CgroupCollector) readCgroupV1Memory() (*memInfo, error) {
	info := &memInfo{}

	// 读取内存限制
	paths := []string{
		c.cgroupPath + "/memory/memory.limit_in_bytes",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	}

	for _, path := range paths {
		if content, err := ioutil.ReadFile(path); err == nil {
			info.total, _ = strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
			break
		}
	}

	// 读取当前使用
	usagePaths := []string{
		c.cgroupPath + "/memory/memory.usage_in_bytes",
		"/sys/fs/cgroup/memory/memory.usage_in_bytes",
	}

	for _, path := range usagePaths {
		if content, err := ioutil.ReadFile(path); err == nil {
			info.used, _ = strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
			break
		}
	}

	// 如果限制值很大（通常是 9223372036854771712），使用系统总内存
	if info.total > 1e15 {
		if procMemInfo, err := c.procCollector.collectMemoryInfo(); err == nil {
			info.total = procMemInfo.total
		}
	}

	info.available = info.total - info.used
	if info.available < 0 {
		info.available = 0
	}

	return info, nil
}

// ============================================================================
// ExternalMetricsCollector - 从外部系统（VictoriaMetrics/Prometheus）采集
// ============================================================================

// ExternalMetricsCollector 从 VictoriaMetrics/Prometheus 采集指标
type ExternalMetricsCollector struct {
	endpoint string
	client   *http.Client
	hostname string
}

// NewExternalMetricsCollector 创建外部指标采集器
func NewExternalMetricsCollector(endpoint, hostname string) *ExternalMetricsCollector {
	return &ExternalMetricsCollector{
		endpoint: endpoint,
		hostname: hostname,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Source 返回采集器来源
func (c *ExternalMetricsCollector) Source() MetricsSource {
	return MetricsSourcePrometheus
}

// IsAvailable 检查采集器是否可用
func (c *ExternalMetricsCollector) IsAvailable() bool {
	resp, err := c.client.Get(c.endpoint + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Collect 从外部系统采集指标
func (c *ExternalMetricsCollector) Collect() (*SystemMetrics, error) {
	// 使用 MetricsService 查询
	metricsService := NewMetricsService()
	hostMetrics, err := metricsService.GetHostMetrics(c.hostname)
	if err != nil {
		return nil, err
	}

	return &SystemMetrics{
		MetricsSource:       MetricsSourcePrometheus,
		CPUUsagePercent:     hostMetrics.CPUUsage,
		MemoryUsagePercent:  hostMetrics.MemoryUsage,
		NetworkBandwidthIn:  hostMetrics.NetworkIn,
		NetworkBandwidthOut: hostMetrics.NetworkOut,
		LoadAvg1:            hostMetrics.Load1,
		LoadAvg5:            hostMetrics.Load5,
		LoadAvg15:           hostMetrics.Load15,
		UptimeSeconds:       int64(hostMetrics.Uptime),
		ActiveConnections:   hostMetrics.ActiveConnections,
	}, nil
}

// ============================================================================
// 辅助函数：统一的指标采集入口
// ============================================================================

// CollectSystemMetrics 采集当前系统的指标（全局便捷函数）
func CollectSystemMetrics() (*SystemMetrics, error) {
	return NewSystemMetricsCollectorService().Collect()
}

// CollectSystemMetricsFresh 强制重新采集指标（不使用缓存）
func CollectSystemMetricsFresh() (*SystemMetrics, error) {
	return NewSystemMetricsCollectorService().CollectFresh()
}

// GetDeploymentType 获取当前部署类型
func GetDeploymentType() DeploymentType {
	return NewSystemMetricsCollectorService().GetDeploymentType()
}

// ============================================================================
// DockerMetricsClient - 通过 Docker API 获取容器指标
// ============================================================================

// ContainerMetrics 容器指标
type ContainerMetrics struct {
	ContainerID        string  `json:"container_id"`
	ContainerName      string  `json:"container_name"`
	CPUPercent         float64 `json:"cpu_percent"`
	MemoryUsed         int64   `json:"memory_used"`
	MemoryLimit        int64   `json:"memory_limit"`
	MemoryPercent      float64 `json:"memory_percent"`
	NetworkRxBytes     int64   `json:"network_rx_bytes"`
	NetworkTxBytes     int64   `json:"network_tx_bytes"`
	NetworkConnections int     `json:"network_connections"`
}

// DockerMetricsClient Docker 指标采集客户端
type DockerMetricsClient struct {
	socketPath string
}

// NewDockerMetricsClient 创建 Docker 指标客户端
func NewDockerMetricsClient() (*DockerMetricsClient, error) {
	socketPath := "/var/run/docker.sock"

	// 检查 Docker socket 是否存在
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		// 尝试检查 docker 命令是否可用
		if _, err := exec.LookPath("docker"); err != nil {
			return nil, fmt.Errorf("Docker not available: socket not found and docker command not in PATH")
		}
	}

	return &DockerMetricsClient{
		socketPath: socketPath,
	}, nil
}

// Close 关闭客户端
func (c *DockerMetricsClient) Close() error {
	return nil
}

// GetContainerMetrics 获取指定容器的指标
func (c *DockerMetricsClient) GetContainerMetrics(containerName string) (*ContainerMetrics, error) {
	// 使用 docker stats 命令获取容器指标（更可靠的方式）
	return c.getContainerMetricsViaExec(containerName)
}

// getContainerMetricsViaExec 通过执行 docker 命令获取容器指标
func (c *DockerMetricsClient) getContainerMetricsViaExec(containerName string) (*ContainerMetrics, error) {
	// 使用 docker stats --no-stream 获取一次性的容器指标
	// 格式: {{.Container}} {{.CPUPerc}} {{.MemUsage}} {{.MemPerc}} {{.NetIO}}
	cmd := exec.Command("docker", "stats", "--no-stream", "--format",
		"{{.Container}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}", containerName)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats failed for %s: %v", containerName, err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return nil, fmt.Errorf("container %s not found or not running", containerName)
	}

	// 解析输出: container_id|5.50%|100MiB / 1GiB|10.00%|1.5MB / 500KB
	parts := strings.Split(outputStr, "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected docker stats output format: %s", outputStr)
	}

	metrics := &ContainerMetrics{
		ContainerName: containerName,
	}

	// 解析 CPU 百分比 (去掉 % 符号)
	cpuStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), "%")
	if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
		metrics.CPUPercent = cpu
	}

	// 解析内存百分比 (去掉 % 符号)
	memPercStr := strings.TrimSuffix(strings.TrimSpace(parts[3]), "%")
	if memPerc, err := strconv.ParseFloat(memPercStr, 64); err == nil {
		metrics.MemoryPercent = memPerc
	}

	// 解析内存使用量 (格式: "100MiB / 1GiB")
	memParts := strings.Split(parts[2], "/")
	if len(memParts) >= 2 {
		metrics.MemoryUsed = parseMemorySize(strings.TrimSpace(memParts[0]))
		metrics.MemoryLimit = parseMemorySize(strings.TrimSpace(memParts[1]))
	}

	// 解析网络 IO (格式: "1.5MB / 500KB")
	netParts := strings.Split(parts[4], "/")
	if len(netParts) >= 2 {
		metrics.NetworkRxBytes = parseMemorySize(strings.TrimSpace(netParts[0]))
		metrics.NetworkTxBytes = parseMemorySize(strings.TrimSpace(netParts[1]))
	}

	return metrics, nil
}

// parseMemorySize 解析内存大小字符串 (如 "100MiB", "1GiB", "500KB")
func parseMemorySize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	if sizeStr == "" {
		return 0
	}

	var multiplier int64 = 1
	var numStr string

	switch {
	case strings.HasSuffix(sizeStr, "GIB"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "GIB")
	case strings.HasSuffix(sizeStr, "GB"):
		multiplier = 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(sizeStr, "GB")
	case strings.HasSuffix(sizeStr, "MIB"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "MIB")
	case strings.HasSuffix(sizeStr, "MB"):
		multiplier = 1000 * 1000
		numStr = strings.TrimSuffix(sizeStr, "MB")
	case strings.HasSuffix(sizeStr, "KIB"):
		multiplier = 1024
		numStr = strings.TrimSuffix(sizeStr, "KIB")
	case strings.HasSuffix(sizeStr, "KB"):
		multiplier = 1000
		numStr = strings.TrimSuffix(sizeStr, "KB")
	case strings.HasSuffix(sizeStr, "B"):
		multiplier = 1
		numStr = strings.TrimSuffix(sizeStr, "B")
	default:
		numStr = sizeStr
	}

	numStr = strings.TrimSpace(numStr)
	if num, err := strconv.ParseFloat(numStr, 64); err == nil {
		return int64(num * float64(multiplier))
	}

	return 0
}
