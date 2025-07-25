package config

import (
	"os"
	"strconv"
)

// JupyterHubK8sConfig JupyterHub K8s集成配置
type JupyterHubK8sConfig struct {
	// Kubernetes配置
	KubeConfigPath    string `json:"kube_config_path" yaml:"kube_config_path"`
	Namespace         string `json:"namespace" yaml:"namespace"`
	
	// NFS存储配置
	NFSServer         string `json:"nfs_server" yaml:"nfs_server"`
	NFSPath           string `json:"nfs_path" yaml:"nfs_path"`
	
	// 默认资源配置
	DefaultGPULimit   int    `json:"default_gpu_limit" yaml:"default_gpu_limit"`
	DefaultMemoryMB   int    `json:"default_memory_mb" yaml:"default_memory_mb"`
	DefaultCPUCores   int    `json:"default_cpu_cores" yaml:"default_cpu_cores"`
	JobTimeoutSeconds int    `json:"job_timeout_seconds" yaml:"job_timeout_seconds"`
	
	// Docker镜像配置
	BaseImage         string `json:"base_image" yaml:"base_image"`
	GPUImage          string `json:"gpu_image" yaml:"gpu_image"`
	
	// JupyterHub配置
	JupyterHubProjectPath string `json:"jupyterhub_project_path" yaml:"jupyterhub_project_path"`
	
	// 监控配置
	MetricsEnabled    bool   `json:"metrics_enabled" yaml:"metrics_enabled"`
	LogLevel          string `json:"log_level" yaml:"log_level"`
}

// LoadJupyterHubK8sConfig 加载JupyterHub K8s配置
func LoadJupyterHubK8sConfig() *JupyterHubK8sConfig {
	config := &JupyterHubK8sConfig{
		// 默认值
		KubeConfigPath:        getEnv("KUBE_CONFIG_PATH", ""),
		Namespace:             getEnv("JUPYTERHUB_K8S_NAMESPACE", "jupyterhub-jobs"),
		NFSServer:             getEnv("NFS_SERVER", "nfs-server.default.svc.cluster.local"),
		NFSPath:               getEnv("NFS_PATH", "/shared"),
		DefaultGPULimit:       getEnvInt("DEFAULT_GPU_LIMIT", 1),
		DefaultMemoryMB:       getEnvInt("DEFAULT_MEMORY_MB", 2048),
		DefaultCPUCores:       getEnvInt("DEFAULT_CPU_CORES", 2),
		JobTimeoutSeconds:     getEnvInt("JOB_TIMEOUT_SECONDS", 3600),
		BaseImage:             getEnv("PYTHON_BASE_IMAGE", "python:3.9-slim"),
		GPUImage:              getEnv("PYTHON_GPU_IMAGE", "nvidia/cuda:11.8-devel-ubuntu20.04"),
		JupyterHubProjectPath: getEnv("JUPYTERHUB_PROJECT_PATH", "/workspace/third-party/jupyterhub"),
		MetricsEnabled:        getEnvBool("METRICS_ENABLED", true),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
	}
	
	return config
}

// getEnvInt 获取整数环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
