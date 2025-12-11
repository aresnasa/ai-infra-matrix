package services

import (
	"testing"
	"time"
)

func TestSystemMetricsCollectorService_DetectDeploymentType(t *testing.T) {
	svc := &SystemMetricsCollectorService{}
	deploymentType := svc.detectDeploymentType()

	t.Logf("检测到的部署类型: %s", deploymentType)

	// 验证返回值是有效的部署类型
	validTypes := []DeploymentType{
		DeploymentTypeContainer,
		DeploymentTypeVM,
		DeploymentTypeBareMetal,
		DeploymentTypeUnknown,
	}

	found := false
	for _, vt := range validTypes {
		if deploymentType == vt {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("无效的部署类型: %s", deploymentType)
	}
}

func TestProcCollector_IsAvailable(t *testing.T) {
	collector := NewProcCollector()
	available := collector.IsAvailable()

	t.Logf("ProcCollector 可用状态: %v", available)

	// 在 Linux 系统上应该是可用的
	if !available {
		t.Skip("ProcCollector 在当前系统上不可用 (可能是 macOS 或 Windows)")
	}
}

func TestProcCollector_Collect(t *testing.T) {
	collector := NewProcCollector()

	if !collector.IsAvailable() {
		t.Skip("ProcCollector 在当前系统上不可用")
	}

	// 第一次采集（初始化 CPU 基线）
	metrics1, err := collector.Collect()
	if err != nil {
		t.Fatalf("第一次采集失败: %v", err)
	}

	t.Logf("第一次采集结果: %+v", metrics1)

	// 等待一小段时间让系统有变化
	time.Sleep(100 * time.Millisecond)

	// 第二次采集（应该有 CPU 使用率）
	metrics2, err := collector.Collect()
	if err != nil {
		t.Fatalf("第二次采集失败: %v", err)
	}

	t.Logf("第二次采集结果:")
	t.Logf("  CPU 使用率: %.2f%%", metrics2.CPUUsagePercent)
	t.Logf("  内存使用率: %.2f%%", metrics2.MemoryUsagePercent)
	t.Logf("  内存总量: %d bytes (%.2f GB)", metrics2.MemoryTotalBytes, float64(metrics2.MemoryTotalBytes)/1024/1024/1024)
	t.Logf("  已用内存: %d bytes (%.2f GB)", metrics2.MemoryUsedBytes, float64(metrics2.MemoryUsedBytes)/1024/1024/1024)
	t.Logf("  可用内存: %d bytes (%.2f GB)", metrics2.MemoryAvailableBytes, float64(metrics2.MemoryAvailableBytes)/1024/1024/1024)
	t.Logf("  网络入流量: %d bytes", metrics2.NetworkInBytes)
	t.Logf("  网络出流量: %d bytes", metrics2.NetworkOutBytes)
	t.Logf("  入带宽: %.2f bytes/s", metrics2.NetworkBandwidthIn)
	t.Logf("  出带宽: %.2f bytes/s", metrics2.NetworkBandwidthOut)
	t.Logf("  活跃连接数: %d", metrics2.ActiveConnections)
	t.Logf("  负载: %.2f / %.2f / %.2f", metrics2.LoadAvg1, metrics2.LoadAvg5, metrics2.LoadAvg15)
	t.Logf("  运行时间: %d 秒", metrics2.UptimeSeconds)
	t.Logf("  CPU 核心数: %d", metrics2.CPUCores)

	// 验证一些基本的合理性
	if metrics2.MemoryTotalBytes <= 0 {
		t.Error("内存总量应该大于 0")
	}

	if metrics2.CPUCores <= 0 {
		t.Error("CPU 核心数应该大于 0")
	}

	if metrics2.MemoryUsagePercent < 0 || metrics2.MemoryUsagePercent > 100 {
		t.Errorf("内存使用率应该在 0-100 之间，实际: %.2f", metrics2.MemoryUsagePercent)
	}
}

func TestCgroupCollector_IsAvailable(t *testing.T) {
	collector := NewCgroupCollector()
	available := collector.IsAvailable()

	t.Logf("CgroupCollector 可用状态: %v", available)
	t.Logf("Cgroup 版本: %d", collector.cgroupVersion)

	if !available {
		t.Skip("CgroupCollector 在当前系统上不可用")
	}
}

func TestCgroupCollector_Collect(t *testing.T) {
	collector := NewCgroupCollector()

	if !collector.IsAvailable() {
		t.Skip("CgroupCollector 在当前系统上不可用")
	}

	// 第一次采集
	metrics1, err := collector.Collect()
	if err != nil {
		t.Fatalf("第一次采集失败: %v", err)
	}

	t.Logf("第一次采集结果: %+v", metrics1)

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 第二次采集
	metrics2, err := collector.Collect()
	if err != nil {
		t.Fatalf("第二次采集失败: %v", err)
	}

	t.Logf("Cgroup 采集结果:")
	t.Logf("  CPU 使用率: %.2f%%", metrics2.CPUUsagePercent)
	t.Logf("  内存使用率: %.2f%%", metrics2.MemoryUsagePercent)
	t.Logf("  内存总量: %d bytes", metrics2.MemoryTotalBytes)
	t.Logf("  已用内存: %d bytes", metrics2.MemoryUsedBytes)
}

func TestSystemMetricsCollectorService_Collect(t *testing.T) {
	// 重置单例以便测试
	systemMetricsCollectorOnce.Do(func() {}) // 空操作，确保单例初始化

	svc := NewSystemMetricsCollectorService()

	t.Logf("部署类型: %s", svc.GetDeploymentType())

	// 第一次采集
	metrics1, err := svc.Collect()
	if err != nil {
		t.Logf("第一次采集警告: %v", err)
	}

	if metrics1 != nil {
		t.Logf("第一次采集结果: %+v", metrics1)
	}

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 第二次采集
	metrics2, err := svc.CollectFresh()
	if err != nil {
		t.Logf("第二次采集警告: %v", err)
	}

	if metrics2 != nil {
		t.Logf("第二次采集结果:")
		t.Logf("  部署类型: %s", metrics2.DeploymentType)
		t.Logf("  指标来源: %s", metrics2.MetricsSource)
		t.Logf("  CPU 使用率: %.2f%%", metrics2.CPUUsagePercent)
		t.Logf("  内存使用率: %.2f%%", metrics2.MemoryUsagePercent)
		t.Logf("  活跃连接数: %d", metrics2.ActiveConnections)
		t.Logf("  采集时间: %s", metrics2.CollectedAt)
		if metrics2.Error != "" {
			t.Logf("  错误信息: %s", metrics2.Error)
		}
	}
}

func TestSystemMetricsCollectorService_Cache(t *testing.T) {
	svc := NewSystemMetricsCollectorService()
	svc.SetCacheTTL(1 * time.Second)

	// 第一次采集
	metrics1, _ := svc.CollectFresh()
	time1 := metrics1.CollectedAt

	// 立即再次采集（应该命中缓存）
	metrics2, _ := svc.Collect()
	time2 := metrics2.CollectedAt

	if !time1.Equal(time2) {
		t.Error("缓存应该生效，两次采集时间应该相同")
	}

	// 等待缓存过期
	time.Sleep(1100 * time.Millisecond)

	// 再次采集（缓存应该过期）
	metrics3, _ := svc.Collect()
	time3 := metrics3.CollectedAt

	if time1.Equal(time3) {
		t.Error("缓存应该已过期，采集时间应该不同")
	}
}

func TestCollectSystemMetrics_GlobalFunction(t *testing.T) {
	// 测试全局便捷函数
	metrics, err := CollectSystemMetrics()
	if err != nil {
		t.Logf("采集警告: %v", err)
	}

	if metrics != nil {
		t.Logf("全局函数采集结果: %+v", metrics)
	}

	// 测试部署类型检测
	deploymentType := GetDeploymentType()
	t.Logf("部署类型: %s", deploymentType)
}

// BenchmarkProcCollector_Collect 基准测试
func BenchmarkProcCollector_Collect(b *testing.B) {
	collector := NewProcCollector()

	if !collector.IsAvailable() {
		b.Skip("ProcCollector 在当前系统上不可用")
	}

	// 预热
	collector.Collect()
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.Collect()
	}
}

// BenchmarkSystemMetricsCollectorService_Collect 基准测试（带缓存）
func BenchmarkSystemMetricsCollectorService_Collect(b *testing.B) {
	svc := NewSystemMetricsCollectorService()

	// 预热
	svc.CollectFresh()
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Collect()
	}
}

// BenchmarkSystemMetricsCollectorService_CollectFresh 基准测试（不带缓存）
func BenchmarkSystemMetricsCollectorService_CollectFresh(b *testing.B) {
	svc := NewSystemMetricsCollectorService()

	// 预热
	svc.CollectFresh()
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.CollectFresh()
	}
}
