package scripts

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed salt/*.sh
var saltScripts embed.FS

// GetSaltScript 获取 Salt 脚本内容
func GetSaltScript(name string) (string, error) {
	// 构建完整路径
	path := fmt.Sprintf("salt/%s", name)
	if !strings.HasSuffix(path, ".sh") {
		path = path + ".sh"
	}

	content, err := saltScripts.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read script %s: %v", name, err)
	}

	return string(content), nil
}

// GetCPUMemoryScript 获取 CPU/内存收集脚本
func GetCPUMemoryScript() (string, error) {
	return GetSaltScript("get_cpu_memory")
}

// GetCPUMemoryLoadAvgScript 获取 CPU/内存/负载收集脚本
func GetCPUMemoryLoadAvgScript() (string, error) {
	return GetSaltScript("get_cpu_memory_loadavg")
}

// GetGPUInfoScript 获取 GPU 信息收集脚本
func GetGPUInfoScript() (string, error) {
	return GetSaltScript("get_gpu_info")
}

// GetIBInfoScript 获取 InfiniBand 信息收集脚本
func GetIBInfoScript() (string, error) {
	return GetSaltScript("get_ib_info")
}

// GetFullMetricsScript 获取完整系统指标收集脚本（CPU/内存/网络/连接数）
func GetFullMetricsScript() (string, error) {
	return GetSaltScript("get_full_metrics")
}

// GetNPUInfoScript 获取 NPU（华为昇腾/寒武纪等）信息收集脚本
func GetNPUInfoScript() (string, error) {
	return GetSaltScript("get_npu_info")
}

// GetTPUInfoScript 获取 TPU（Google TPU/Habana Gaudi 等）信息收集脚本
func GetTPUInfoScript() (string, error) {
	return GetSaltScript("get_tpu_info")
}

// GetAllAcceleratorInfoScript 获取所有 AI 加速器（GPU/NPU/TPU）信息的综合脚本
func GetAllAcceleratorInfoScript() (string, error) {
	gpuScript, err := GetGPUInfoScript()
	if err != nil {
		gpuScript = ""
	}
	npuScript, err := GetNPUInfoScript()
	if err != nil {
		npuScript = ""
	}
	tpuScript, err := GetTPUInfoScript()
	if err != nil {
		tpuScript = ""
	}

	// 组合脚本
	combined := `#!/bin/bash
# Combined AI Accelerator Info Script
echo "=== GPU INFO ==="
` + gpuScript + `
echo "=== NPU INFO ==="
` + npuScript + `
echo "=== TPU INFO ==="
` + tpuScript

	return combined, nil
}
