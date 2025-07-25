package services

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// JupyterHubK8sService JupyterHub与K8s集成服务
type JupyterHubK8sService struct {
	k8sClient   kubernetes.Interface
	namespace   string
	nfsServer   string
	nfsPath     string
	config      *JupyterHubK8sConfig
}

// JupyterHubK8sConfig 配置结构
type JupyterHubK8sConfig struct {
	KubeConfigPath    string `json:"kube_config_path"`
	Namespace         string `json:"namespace"`
	NFSServer         string `json:"nfs_server"`
	NFSPath           string `json:"nfs_path"`
	DefaultGPULimit   int    `json:"default_gpu_limit"`
	DefaultMemoryMB   int    `json:"default_memory_mb"`
	DefaultCPUCores   int    `json:"default_cpu_cores"`
	JobTimeoutSeconds int    `json:"job_timeout_seconds"`
	BaseImage         string `json:"base_image"`
}

// PythonScriptJob Python脚本作业结构
type PythonScriptJob struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Script         string            `json:"script"`
	Requirements   []string          `json:"requirements"`
	GPURequired    bool              `json:"gpu_required"`
	GPUCount       int               `json:"gpu_count"`
	GPUType        string            `json:"gpu_type"`
	MemoryMB       int               `json:"memory_mb"`
	CPUCores       int               `json:"cpu_cores"`
	Environment    map[string]string `json:"environment"`
	WorkingDir     string            `json:"working_dir"`
	OutputPath     string            `json:"output_path"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	StartedAt      *time.Time        `json:"started_at,omitempty"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
}

// GPUNodeInfo GPU节点信息
type GPUNodeInfo struct {
	NodeName      string            `json:"node_name"`
	GPUCount      int               `json:"gpu_count"`
	GPUType       string            `json:"gpu_type"`
	AvailableGPUs int               `json:"available_gpus"`
	Labels        map[string]string `json:"labels"`
	Taints        []corev1.Taint    `json:"taints"`
	Schedulable   bool              `json:"schedulable"`
}

// GPUResourceStatus GPU资源状态
type GPUResourceStatus struct {
	TotalGPUs     int           `json:"total_gpus"`
	AvailableGPUs int           `json:"available_gpus"`
	UsedGPUs      int           `json:"used_gpus"`
	GPUNodes      []GPUNodeInfo `json:"gpu_nodes"`
	LastUpdated   time.Time     `json:"last_updated"`
}

// NewJupyterHubK8sService 创建新的服务实例
func NewJupyterHubK8sService(config *JupyterHubK8sConfig) (*JupyterHubK8sService, error) {
	service := &JupyterHubK8sService{
		namespace: config.Namespace,
		nfsServer: config.NFSServer,
		nfsPath:   config.NFSPath,
		config:    config,
	}

	// 初始化Kubernetes客户端
	if err := service.initializeK8sClient(); err != nil {
		return nil, fmt.Errorf("初始化K8s客户端失败: %w", err)
	}

	// 确保命名空间存在
	if err := service.ensureNamespace(); err != nil {
		return nil, fmt.Errorf("确保命名空间存在失败: %w", err)
	}

	log.Printf("JupyterHub K8s服务初始化成功 - 命名空间: %s", config.Namespace)
	return service, nil
}

// initializeK8sClient 初始化Kubernetes客户端
func (s *JupyterHubK8sService) initializeK8sClient() error {
	var config *rest.Config
	var err error

	// 尝试集群内配置
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用本地配置文件
		if s.config.KubeConfigPath != "" {
			config, err = clientcmd.BuildConfigFromFlags("", s.config.KubeConfigPath)
		} else {
			config, err = clientcmd.BuildConfigFromFlags("", 
				filepath.Join(clientcmd.RecommendedHomeDir, ".kube", "config"))
		}
		
		if err != nil {
			return fmt.Errorf("无法加载Kubernetes配置: %w", err)
		}
	}

	s.k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 验证连接
	_, err = s.k8sClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("验证Kubernetes连接失败: %w", err)
	}

	log.Println("Kubernetes客户端连接成功")
	return nil
}

// ensureNamespace 确保命名空间存在
func (s *JupyterHubK8sService) ensureNamespace() error {
	_, err := s.k8sClient.CoreV1().Namespaces().Get(
		context.TODO(), s.namespace, metav1.GetOptions{})
	
	if err != nil {
		// 创建命名空间
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.namespace,
				Labels: map[string]string{
					"purpose": "jupyterhub-gpu-jobs",
					"created-by": "ai-infra-matrix",
				},
			},
		}
		
		_, err = s.k8sClient.CoreV1().Namespaces().Create(
			context.TODO(), namespace, metav1.CreateOptions{})
		
		if err != nil {
			return fmt.Errorf("创建命名空间失败: %w", err)
		}
		
		log.Printf("创建命名空间: %s", s.namespace)
	}
	
	return nil
}

// GetGPUResourceStatus 获取GPU资源状态
func (s *JupyterHubK8sService) GetGPUResourceStatus(ctx context.Context) (*GPUResourceStatus, error) {
	nodes, err := s.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}

	status := &GPUResourceStatus{
		GPUNodes:    make([]GPUNodeInfo, 0),
		LastUpdated: time.Now(),
	}

	for _, node := range nodes.Items {
		gpuInfo := s.analyzeGPUNode(&node)
		if gpuInfo != nil && gpuInfo.GPUCount > 0 {
			// 计算可用GPU
			availableGPUs, err := s.calculateAvailableGPUs(ctx, node.Name, gpuInfo.GPUCount)
			if err != nil {
				log.Printf("计算节点 %s 可用GPU失败: %v", node.Name, err)
				gpuInfo.AvailableGPUs = 0
			} else {
				gpuInfo.AvailableGPUs = availableGPUs
			}

			status.GPUNodes = append(status.GPUNodes, *gpuInfo)
			status.TotalGPUs += gpuInfo.GPUCount
			status.AvailableGPUs += gpuInfo.AvailableGPUs
		}
	}

	status.UsedGPUs = status.TotalGPUs - status.AvailableGPUs
	
	log.Printf("GPU资源状态: 总计%d, 可用%d, 已用%d", 
		status.TotalGPUs, status.AvailableGPUs, status.UsedGPUs)
	
	return status, nil
}

// analyzeGPUNode 分析单个节点的GPU信息
func (s *JupyterHubK8sService) analyzeGPUNode(node *corev1.Node) *GPUNodeInfo {
	capacity := node.Status.Capacity
	labels := node.Labels
	
	// 检查NVIDIA GPU
	var gpuCount int
	var gpuType string
	
	if nvidiaGPU, exists := capacity["nvidia.com/gpu"]; exists {
		gpuCount = int(nvidiaGPU.Value())
		gpuType = s.extractGPUType(labels, "nvidia")
	} else if amdGPU, exists := capacity["amd.com/gpu"]; exists {
		gpuCount = int(amdGPU.Value())
		gpuType = s.extractGPUType(labels, "amd")
	}
	
	if gpuCount == 0 {
		return nil
	}
	
	return &GPUNodeInfo{
		NodeName:    node.Name,
		GPUCount:    gpuCount,
		GPUType:     gpuType,
		Labels:      labels,
		Taints:      node.Spec.Taints,
		Schedulable: !node.Spec.Unschedulable,
	}
}

// extractGPUType 从节点标签提取GPU类型
func (s *JupyterHubK8sService) extractGPUType(labels map[string]string, vendor string) string {
	gpuTypeKeys := []string{
		vendor + ".com/gpu-type",
		vendor + ".com/gpu.product",
		"accelerator",
		"gpu-type",
		"node.kubernetes.io/instance-type",
	}
	
	for _, key := range gpuTypeKeys {
		if value, exists := labels[key]; exists {
			return value
		}
	}
	
	// 查找包含GPU型号的标签
	for key, value := range labels {
		if strings.Contains(strings.ToLower(key), "gpu") &&
		   (strings.Contains(strings.ToLower(value), "rtx") ||
			strings.Contains(strings.ToLower(value), "tesla") ||
			strings.Contains(strings.ToLower(value), "a100") ||
			strings.Contains(strings.ToLower(value), "v100")) {
			return value
		}
	}
	
	return vendor + "-gpu"
}

// calculateAvailableGPUs 计算节点可用GPU数量
func (s *JupyterHubK8sService) calculateAvailableGPUs(ctx context.Context, nodeName string, totalGPUs int) (int, error) {
	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", nodeName).String()
	
	pods, err := s.k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	
	if err != nil {
		return 0, err
	}
	
	usedGPUs := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			for _, container := range pod.Spec.Containers {
				if container.Resources.Requests != nil {
					if gpuRequest, exists := container.Resources.Requests["nvidia.com/gpu"]; exists {
						usedGPUs += int(gpuRequest.Value())
					}
				}
			}
		}
	}
	
	availableGPUs := totalGPUs - usedGPUs
	if availableGPUs < 0 {
		availableGPUs = 0
	}
	
	return availableGPUs, nil
}

// FindSuitableGPUNodes 查找适合的GPU节点
func (s *JupyterHubK8sService) FindSuitableGPUNodes(ctx context.Context, requiredGPUs int, gpuTypePreference string) ([]GPUNodeInfo, error) {
	status, err := s.GetGPUResourceStatus(ctx)
	if err != nil {
		return nil, err
	}
	
	var suitableNodes []GPUNodeInfo
	
	for _, node := range status.GPUNodes {
		if node.Schedulable && node.AvailableGPUs >= requiredGPUs {
			// 检查GPU类型偏好
			if gpuTypePreference == "" || 
			   strings.Contains(strings.ToLower(node.GPUType), strings.ToLower(gpuTypePreference)) {
				suitableNodes = append(suitableNodes, node)
			}
		}
	}
	
	log.Printf("找到 %d 个适合的GPU节点 (需要 %d GPU)", len(suitableNodes), requiredGPUs)
	return suitableNodes, nil
}

// SubmitPythonScriptJob 提交Python脚本作业
func (s *JupyterHubK8sService) SubmitPythonScriptJob(ctx context.Context, job *PythonScriptJob) (*batchv1.Job, error) {
	// 验证GPU资源
	if job.GPURequired {
		suitableNodes, err := s.FindSuitableGPUNodes(ctx, job.GPUCount, job.GPUType)
		if err != nil {
			return nil, fmt.Errorf("查找GPU节点失败: %w", err)
		}
		
		if len(suitableNodes) == 0 {
			return nil, fmt.Errorf("没有找到满足要求的GPU节点 (需要 %d GPU)", job.GPUCount)
		}
		
		log.Printf("选择GPU节点: %s (可用GPU: %d)", 
			suitableNodes[0].NodeName, suitableNodes[0].AvailableGPUs)
	}
	
	// 生成K8s Job
	k8sJob, err := s.generateK8sJob(job)
	if err != nil {
		return nil, fmt.Errorf("生成K8s Job失败: %w", err)
	}
	
	// 提交Job
	createdJob, err := s.k8sClient.BatchV1().Jobs(s.namespace).Create(
		ctx, k8sJob, metav1.CreateOptions{})
	
	if err != nil {
		return nil, fmt.Errorf("提交K8s Job失败: %w", err)
	}
	
	job.Status = "submitted"
	now := time.Now()
	job.StartedAt = &now
	
	log.Printf("成功提交Python脚本Job: %s", createdJob.Name)
	return createdJob, nil
}

// generateK8sJob 生成Kubernetes Job
func (s *JupyterHubK8sService) generateK8sJob(job *PythonScriptJob) (*batchv1.Job, error) {
	// 构建容器规格
	container := corev1.Container{
		Name:  "python-script",
		Image: s.config.BaseImage,
		Command: []string{"/bin/bash", "-c"},
		Args: []string{s.buildScriptCommand(job)},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "nfs-storage",
				MountPath: "/shared",
			},
		},
		Env: s.buildEnvironmentVars(job),
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", job.CPUCores)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", job.MemoryMB)),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", job.CPUCores)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", job.MemoryMB)),
			},
		},
	}
	
	// 如果需要GPU，添加GPU资源
	if job.GPURequired {
		container.Resources.Requests["nvidia.com/gpu"] = resource.MustParse(fmt.Sprintf("%d", job.GPUCount))
		container.Resources.Limits["nvidia.com/gpu"] = resource.MustParse(fmt.Sprintf("%d", job.GPUCount))
	}
	
	// 构建Pod规格
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers:    []corev1.Container{container},
		Volumes: []corev1.Volume{
			{
				Name: "nfs-storage",
				VolumeSource: corev1.VolumeSource{
					NFS: &corev1.NFSVolumeSource{
						Server: s.nfsServer,
						Path:   s.nfsPath,
					},
				},
			},
		},
	}
	
	// 如果需要GPU，添加节点选择器
	if job.GPURequired {
		podSpec.NodeSelector = map[string]string{
			"accelerator": "nvidia",
		}
		// 添加GPU容忍度
		podSpec.Tolerations = []corev1.Toleration{
			{
				Key:      "nvidia.com/gpu",
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			},
		}
	}
	
	// 构建Job规格
	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("python-job-%s", job.ID),
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":        "jupyterhub-python-job",
				"job-id":     job.ID,
				"created-by": "ai-infra-matrix",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            int32Ptr(int32(s.config.JobTimeoutSeconds/60)),
			ActiveDeadlineSeconds:   int64Ptr(int64(s.config.JobTimeoutSeconds)),
			TTLSecondsAfterFinished: int32Ptr(3600), // 1小时后清理
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
	
	return k8sJob, nil
}

// buildScriptCommand 构建脚本执行命令
func (s *JupyterHubK8sService) buildScriptCommand(job *PythonScriptJob) string {
	commands := []string{
		"set -e",  // 遇到错误时退出
	}
	
	// 安装依赖
	if len(job.Requirements) > 0 {
		commands = append(commands, "echo '安装Python依赖...'")
		for _, req := range job.Requirements {
			commands = append(commands, fmt.Sprintf("pip install %s", req))
		}
	}
	
	// 设置工作目录
	if job.WorkingDir != "" {
		commands = append(commands, fmt.Sprintf("mkdir -p %s && cd %s", job.WorkingDir, job.WorkingDir))
	}
	
	// 写入并执行Python脚本
	commands = append(commands, 
		"echo '开始执行Python脚本...'",
		"cat << 'EOF' > script.py",
		job.Script,
		"EOF",
		"python script.py",
	)
	
	// 保存输出
	if job.OutputPath != "" {
		commands = append(commands, 
			fmt.Sprintf("echo '保存输出到 %s'", job.OutputPath),
			fmt.Sprintf("mkdir -p %s", filepath.Dir(job.OutputPath)),
			"echo '脚本执行完成' > execution_status.txt",
		)
	}
	
	return strings.Join(commands, "\n")
}

// buildEnvironmentVars 构建环境变量
func (s *JupyterHubK8sService) buildEnvironmentVars(job *PythonScriptJob) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "JOB_ID", Value: job.ID},
		{Name: "JOB_NAME", Value: job.Name},
		{Name: "PYTHONUNBUFFERED", Value: "1"},
	}
	
	// 添加自定义环境变量
	for key, value := range job.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	
	// 如果使用GPU，添加CUDA相关环境变量
	if job.GPURequired {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "NVIDIA_VISIBLE_DEVICES", Value: "all"},
			{Name: "CUDA_VISIBLE_DEVICES", Value: "all"},
		}...)
	}
	
	return envVars
}

// MonitorJob 监控Job状态
func (s *JupyterHubK8sService) MonitorJob(ctx context.Context, jobName string) (*PythonScriptJob, error) {
	k8sJob, err := s.k8sClient.BatchV1().Jobs(s.namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取Job状态失败: %w", err)
	}
	
	job := &PythonScriptJob{
		ID:   k8sJob.Labels["job-id"],
		Name: k8sJob.Name,
	}
	
	// 分析Job状态
	if k8sJob.Status.CompletionTime != nil {
		job.Status = "completed"
		completedAt := k8sJob.Status.CompletionTime.Time
		job.CompletedAt = &completedAt
	} else if k8sJob.Status.Failed > 0 {
		job.Status = "failed"
		job.ErrorMessage = "Job执行失败"
	} else if k8sJob.Status.Active > 0 {
		job.Status = "running"
	} else {
		job.Status = "pending"
	}
	
	// 获取Job日志 (可选)
	if job.Status == "failed" || job.Status == "completed" {
		logs, err := s.getJobLogs(ctx, jobName)
		if err == nil && logs != "" {
			job.ErrorMessage = logs
		}
	}
	
	return job, nil
}

// getJobLogs 获取Job日志
func (s *JupyterHubK8sService) getJobLogs(ctx context.Context, jobName string) (string, error) {
	// 获取Job关联的Pod
	labelSelector := fmt.Sprintf("job-name=%s", jobName)
	pods, err := s.k8sClient.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	
	if err != nil || len(pods.Items) == 0 {
		return "", fmt.Errorf("未找到Job对应的Pod")
	}
	
	// 获取第一个Pod的日志
	pod := pods.Items[0]
	req := s.k8sClient.CoreV1().Pods(s.namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		TailLines: int64Ptr(100), // 只获取最后100行
	})
	
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}
	
	return string(logs), nil
}

// CleanupCompletedJobs 清理已完成的Job
func (s *JupyterHubK8sService) CleanupCompletedJobs(ctx context.Context) error {
	jobs, err := s.k8sClient.BatchV1().Jobs(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=jupyterhub-python-job",
	})
	
	if err != nil {
		return err
	}
	
	cleaned := 0
	for _, job := range jobs.Items {
		// 清理超过1小时的已完成Job
		if job.Status.CompletionTime != nil && 
		   time.Since(job.Status.CompletionTime.Time) > time.Hour {
			
			err := s.k8sClient.BatchV1().Jobs(s.namespace).Delete(
				ctx, job.Name, metav1.DeleteOptions{
					PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationBackground}[0],
				})
			
			if err != nil {
				log.Printf("清理Job %s 失败: %v", job.Name, err)
			} else {
				cleaned++
			}
		}
	}
	
	log.Printf("清理了 %d 个已完成的Job", cleaned)
	return nil
}

// 辅助函数
func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
