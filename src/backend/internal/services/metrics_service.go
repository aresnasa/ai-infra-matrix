package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// MetricsService 监控指标查询服务
// 从 VictoriaMetrics 或 Prometheus 查询监控数据
type MetricsService struct {
	vmHost string
	vmPort string
	vmURL  string
	client *http.Client
}

// MetricsQueryResult Prometheus/VictoriaMetrics 查询结果
type MetricsQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`  // [timestamp, value]
			Values [][]interface{}   `json:"values"` // for range queries
		} `json:"result"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// HostMetrics 主机监控指标
type HostMetrics struct {
	Hostname          string  `json:"hostname"`
	CPUUsage          float64 `json:"cpu_usage"`          // CPU 使用率 (%)
	MemoryUsage       float64 `json:"memory_usage"`       // 内存使用率 (%)
	DiskUsage         float64 `json:"disk_usage"`         // 磁盘使用率 (%)
	NetworkIn         float64 `json:"network_in"`         // 网络入流量 (bytes/s)
	NetworkOut        float64 `json:"network_out"`        // 网络出流量 (bytes/s)
	Load1             float64 `json:"load1"`              // 1分钟负载
	Load5             float64 `json:"load5"`              // 5分钟负载
	Load15            float64 `json:"load15"`             // 15分钟负载
	Uptime            float64 `json:"uptime"`             // 运行时间 (秒)
	ActiveConnections int     `json:"active_connections"` // 活跃连接数
	LastUpdate        int64   `json:"last_update"`        // 最后更新时间
}

// ClusterMetrics 集群聚合指标
type ClusterMetrics struct {
	TotalHosts        int     `json:"total_hosts"`
	OnlineHosts       int     `json:"online_hosts"`
	AvgCPUUsage       float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage    float64 `json:"avg_memory_usage"`
	TotalDiskUsage    float64 `json:"total_disk_usage"`
	ActiveConnections int     `json:"active_connections"`
	LastUpdate        int64   `json:"last_update"`
}

// NewMetricsService 创建监控指标服务
func NewMetricsService() *MetricsService {
	vmHost := os.Getenv("VICTORIAMETRICS_HOST")
	if vmHost == "" {
		vmHost = "victoriametrics"
	}
	vmPort := os.Getenv("VICTORIAMETRICS_PORT")
	if vmPort == "" {
		vmPort = "8428"
	}

	vmURL := fmt.Sprintf("http://%s:%s", vmHost, vmPort)

	return &MetricsService{
		vmHost: vmHost,
		vmPort: vmPort,
		vmURL:  vmURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Query 执行 PromQL 查询
func (s *MetricsService) Query(promQL string) (*MetricsQueryResult, error) {
	queryURL := fmt.Sprintf("%s/api/v1/query", s.vmURL)

	params := url.Values{}
	params.Add("query", promQL)

	resp, err := s.client.Get(queryURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to query VictoriaMetrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VictoriaMetrics returned status %d", resp.StatusCode)
	}

	var result MetricsQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", result.Error)
	}

	return &result, nil
}

// GetHostMetrics 获取单个主机的监控指标
func (s *MetricsService) GetHostMetrics(hostname string) (*HostMetrics, error) {
	metrics := &HostMetrics{
		Hostname:   hostname,
		LastUpdate: time.Now().Unix(),
	}

	// CPU 使用率
	cpuQuery := fmt.Sprintf(`100 - avg(cpu_usage_idle{ident="%s"})`, hostname)
	if result, err := s.Query(cpuQuery); err == nil {
		metrics.CPUUsage = s.extractValue(result)
	}

	// 内存使用率
	memQuery := fmt.Sprintf(`100 - mem_available_percent{ident="%s"}`, hostname)
	if result, err := s.Query(memQuery); err == nil {
		metrics.MemoryUsage = s.extractValue(result)
	}

	// 磁盘使用率 (根分区)
	diskQuery := fmt.Sprintf(`disk_used_percent{ident="%s",path="/"}`, hostname)
	if result, err := s.Query(diskQuery); err == nil {
		metrics.DiskUsage = s.extractValue(result)
	}

	// 系统负载
	loadQuery := fmt.Sprintf(`system_load1{ident="%s"}`, hostname)
	if result, err := s.Query(loadQuery); err == nil {
		metrics.Load1 = s.extractValue(result)
	}

	loadQuery5 := fmt.Sprintf(`system_load5{ident="%s"}`, hostname)
	if result, err := s.Query(loadQuery5); err == nil {
		metrics.Load5 = s.extractValue(result)
	}

	loadQuery15 := fmt.Sprintf(`system_load15{ident="%s"}`, hostname)
	if result, err := s.Query(loadQuery15); err == nil {
		metrics.Load15 = s.extractValue(result)
	}

	// 运行时间
	uptimeQuery := fmt.Sprintf(`system_uptime{ident="%s"}`, hostname)
	if result, err := s.Query(uptimeQuery); err == nil {
		metrics.Uptime = s.extractValue(result)
	}

	return metrics, nil
}

// GetClusterMetrics 获取集群聚合指标
func (s *MetricsService) GetClusterMetrics() (*ClusterMetrics, error) {
	metrics := &ClusterMetrics{
		LastUpdate: time.Now().Unix(),
	}

	// 在线主机数
	hostsQuery := `count(up{job=~".*categraf.*"} == 1)`
	if result, err := s.Query(hostsQuery); err == nil {
		metrics.OnlineHosts = int(s.extractValue(result))
	}

	// 总主机数 (包括离线)
	totalQuery := `count(up{job=~".*categraf.*"})`
	if result, err := s.Query(totalQuery); err == nil {
		metrics.TotalHosts = int(s.extractValue(result))
	}

	// 平均 CPU 使用率
	avgCPUQuery := `avg(100 - cpu_usage_idle)`
	if result, err := s.Query(avgCPUQuery); err == nil {
		metrics.AvgCPUUsage = s.extractValue(result)
	}

	// 平均内存使用率
	avgMemQuery := `avg(100 - mem_available_percent)`
	if result, err := s.Query(avgMemQuery); err == nil {
		metrics.AvgMemoryUsage = s.extractValue(result)
	}

	return metrics, nil
}

// GetSaltStackMetrics 获取 SaltStack 相关指标
func (s *MetricsService) GetSaltStackMetrics() (cpuUsage, memoryUsage, activeConnections int, err error) {
	// 尝试获取 Salt Master 主机的指标
	// 首先尝试从 VictoriaMetrics 查询
	// 匹配多种可能的主机标识：salt-master, saltstack, 或包含 salt 的主机名

	// Salt Master CPU 使用率
	// 使用更宽泛的匹配模式，包括容器名称
	cpuQuery := `avg(100 - cpu_usage_idle{ident=~".*salt.*|saltstack|salt-master.*"})`
	if result, queryErr := s.Query(cpuQuery); queryErr == nil {
		cpuUsage = int(s.extractValue(result))
	}

	// 如果没有数据，尝试用 host 标签匹配
	if cpuUsage == 0 {
		cpuQuery = `avg(100 - cpu_usage_idle{host=~".*salt.*|saltstack|salt-master.*"})`
		if result, queryErr := s.Query(cpuQuery); queryErr == nil {
			cpuUsage = int(s.extractValue(result))
		}
	}

	// Salt Master 内存使用率
	memQuery := `avg(100 - mem_available_percent{ident=~".*salt.*|saltstack|salt-master.*"})`
	if result, queryErr := s.Query(memQuery); queryErr == nil {
		memoryUsage = int(s.extractValue(result))
	}

	// 如果没有数据，尝试用 host 标签匹配
	if memoryUsage == 0 {
		memQuery = `avg(100 - mem_available_percent{host=~".*salt.*|saltstack|salt-master.*"})`
		if result, queryErr := s.Query(memQuery); queryErr == nil {
			memoryUsage = int(s.extractValue(result))
		}
	}

	// 活跃连接数 - 需要 netstat 类型的指标
	connQuery := `sum(netstat_tcp_established{ident=~".*salt.*|saltstack|salt-master.*"})`
	if result, queryErr := s.Query(connQuery); queryErr == nil {
		activeConnections = int(s.extractValue(result))
	}

	return cpuUsage, memoryUsage, activeConnections, nil
}

// CheckHealth 检查 VictoriaMetrics 健康状态
func (s *MetricsService) CheckHealth() bool {
	healthURL := fmt.Sprintf("%s/health", s.vmURL)
	resp, err := s.client.Get(healthURL)
	if err != nil {
		logrus.WithError(err).Warn("VictoriaMetrics health check failed")
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// extractValue 从查询结果中提取单个值
func (s *MetricsService) extractValue(result *MetricsQueryResult) float64 {
	if result == nil || len(result.Data.Result) == 0 {
		return 0
	}

	values := result.Data.Result[0].Value
	if len(values) < 2 {
		return 0
	}

	// Value 格式: [timestamp, "value_string"]
	switch v := values[1].(type) {
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	case float64:
		return v
	default:
		return 0
	}
}
