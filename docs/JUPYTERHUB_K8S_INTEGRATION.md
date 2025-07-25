# JupyterHub K8s GPU集成系统

## 概述

本系统实现了将JupyterHub项目中的Python脚本转换为Kubernetes GPU作业的完整解决方案。系统支持：

- 自动扫描JupyterHub项目中的Python脚本
- 智能GPU资源检测和分配
- 动态Docker容器构建和执行
- NFS分布式存储集成
- 作业监控和日志管理
- RESTful API接口

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端应用      │    │   Go后端服务    │    │  Kubernetes     │
│   (Web UI)      │◄──►│  (API Server)   │◄──►│   集群          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │                         │
                              ▼                         ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   JupyterHub    │    │   GPU节点       │
                       │   项目扫描      │    │   (RTX/Tesla)   │
                       └─────────────────┘    └─────────────────┘
                              │                         │
                              ▼                         ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   NFS存储       │    │   Docker镜像    │
                       │   (共享数据)    │    │   (GPU/CPU)     │
                       └─────────────────┘    └─────────────────┘
```

## 功能特性

### 1. GPU资源管理
- 自动检测NVIDIA/AMD GPU节点
- 实时监控GPU使用情况
- 智能节点选择和资源分配
- 支持GPU类型偏好设置

### 2. Python脚本处理
- 自动解析Python脚本依赖
- 智能估算资源需求
- 支持自定义环境变量
- 容器化执行环境

### 3. Kubernetes集成
- 动态Job创建和管理
- Pod生命周期监控
- 网络策略和安全配置
- 资源配额限制

### 4. 存储管理
- NFS分布式存储
- 作业结果持久化
- 共享数据访问
- 自动清理机制

## 快速开始

### 1. 环境准备

```bash
# 1. 确保Kubernetes集群运行正常
kubectl cluster-info

# 2. 检查GPU节点
kubectl get nodes -l accelerator=nvidia

# 3. 创建命名空间和权限
kubectl apply -f k8s/jupyterhub-namespace.yaml
```

### 2. 构建Docker镜像

```bash
# 构建GPU和CPU执行环境镜像
./scripts/build-jupyterhub-images.sh --push --test

# 验证镜像构建
docker images | grep jupyterhub
```

### 3. 配置环境变量

```bash
# 设置环境变量
export JUPYTERHUB_K8S_NAMESPACE="jupyterhub-jobs"
export NFS_SERVER="nfs-server.default.svc.cluster.local"
export NFS_PATH="/shared"
export PYTHON_GPU_IMAGE="localhost:5000/jupyterhub-python-gpu:latest"
export PYTHON_BASE_IMAGE="localhost:5000/jupyterhub-python-cpu:latest"
```

### 4. 启动服务

```bash
# 启动Go后端服务
cd src/backend
go run cmd/main.go
```

### 5. 测试API

```bash
# 检查GPU资源状态
curl http://localhost:8080/api/v1/jupyterhub/gpu/status

# 提交Python脚本作业
python examples/jupyterhub_k8s_client.py --task gpu --wait
```

## API文档

### GPU资源管理

#### 获取GPU状态
```http
GET /api/v1/jupyterhub/gpu/status
```

响应示例：
```json
{
  "total_gpus": 8,
  "available_gpus": 6,
  "used_gpus": 2,
  "gpu_nodes": [
    {
      "node_name": "gpu-node-1",
      "gpu_type": "RTX 4090",
      "gpu_count": 2,
      "available_gpus": 1,
      "schedulable": true
    }
  ],
  "last_updated": "2024-01-20T10:30:00Z"
}
```

#### 查找适合的GPU节点
```http
GET /api/v1/jupyterhub/gpu/nodes?gpu_count=2&gpu_type=rtx4090
```

### 作业管理

#### 提交Python脚本
```http
POST /api/v1/jupyterhub/jobs/submit
Content-Type: application/json

{
  "name": "数据分析任务",
  "script": "import torch\nprint(f'CUDA可用: {torch.cuda.is_available()}')",
  "requirements": ["torch", "numpy"],
  "gpu_required": true,
  "gpu_count": 1,
  "memory_mb": 4096,
  "cpu_cores": 2,
  "environment": {
    "CUDA_VISIBLE_DEVICES": "0"
  }
}
```

#### 获取作业状态
```http
GET /api/v1/jupyterhub/jobs/{jobName}/status
```

响应示例：
```json
{
  "job_id": "job-1642684200",
  "job_name": "python-job-1642684200",
  "status": "completed",
  "created_at": "2024-01-20T10:30:00Z",
  "started_at": "2024-01-20T10:30:05Z",
  "completed_at": "2024-01-20T10:32:15Z"
}
```

## 客户端示例

### Python客户端

```python
from jupyterhub_k8s_client import JupyterHubK8sClient

# 初始化客户端
client = JupyterHubK8sClient("http://localhost:8080")

# 检查GPU资源
status = client.get_gpu_status()
print(f"可用GPU: {status['available_gpus']}")

# 提交GPU作业
script = """
import torch
print(f"GPU数量: {torch.cuda.device_count()}")
for i in range(torch.cuda.device_count()):
    print(f"GPU {i}: {torch.cuda.get_device_name(i)}")
"""

result = client.submit_python_script(
    name="GPU检测",
    script=script,
    requirements=["torch"],
    gpu_required=True,
    gpu_count=1
)

print(f"作业已提交: {result['job_name']}")

# 等待完成
final_status = client.wait_for_job_completion(result['job_name'])
print(f"作业状态: {final_status['status']}")
```

### curl示例

```bash
# 提交CPU密集型任务
curl -X POST http://localhost:8080/api/v1/jupyterhub/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "name": "数据处理",
    "script": "import pandas as pd; import numpy as np; print(\"处理完成\")",
    "requirements": ["pandas", "numpy"],
    "gpu_required": false,
    "memory_mb": 2048,
    "cpu_cores": 2
  }'

# 检查作业状态
curl http://localhost:8080/api/v1/jupyterhub/jobs/python-job-1642684200/status
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `KUBE_CONFIG_PATH` | - | Kubernetes配置文件路径 |
| `JUPYTERHUB_K8S_NAMESPACE` | `jupyterhub-jobs` | K8s命名空间 |
| `NFS_SERVER` | `nfs-server.default.svc.cluster.local` | NFS服务器地址 |
| `NFS_PATH` | `/shared` | NFS共享路径 |
| `DEFAULT_GPU_LIMIT` | `1` | 默认GPU限制 |
| `DEFAULT_MEMORY_MB` | `2048` | 默认内存限制(MB) |
| `DEFAULT_CPU_CORES` | `2` | 默认CPU核心数 |
| `JOB_TIMEOUT_SECONDS` | `3600` | 作业超时时间(秒) |
| `PYTHON_BASE_IMAGE` | `python:3.9-slim` | Python基础镜像 |
| `PYTHON_GPU_IMAGE` | `nvidia/cuda:11.8-devel-ubuntu20.04` | GPU镜像 |

### Kubernetes配置

系统需要以下Kubernetes资源：

1. **命名空间**: `jupyterhub-jobs`
2. **ServiceAccount**: 具有Job管理权限
3. **RBAC**: ClusterRole和ClusterRoleBinding
4. **NFS存储**: PersistentVolume和PersistentVolumeClaim
5. **网络策略**: 安全访问控制
6. **资源配额**: 防止资源滥用

### GPU节点标签

确保GPU节点具有正确的标签：

```bash
# 标记NVIDIA GPU节点
kubectl label nodes gpu-node-1 accelerator=nvidia
kubectl label nodes gpu-node-1 gpu-type=rtx4090

# 添加GPU污点
kubectl taint nodes gpu-node-1 nvidia.com/gpu=present:NoSchedule
```

## 监控和日志

### 作业监控

```bash
# 查看命名空间中的作业
kubectl get jobs -n jupyterhub-jobs

# 查看作业详情
kubectl describe job python-job-1642684200 -n jupyterhub-jobs

# 查看Pod日志
kubectl logs -f job/python-job-1642684200 -n jupyterhub-jobs
```

### 资源监控

```bash
# 查看GPU使用情况
kubectl top nodes --selector=accelerator=nvidia

# 查看命名空间资源使用
kubectl top pods -n jupyterhub-jobs
```

### 清理操作

```bash
# 清理已完成的作业
curl -X POST http://localhost:8080/api/v1/jupyterhub/jobs/cleanup

# 手动删除作业
kubectl delete job python-job-1642684200 -n jupyterhub-jobs
```

## 故障排除

### 常见问题

1. **GPU不可用**
   - 检查GPU驱动和NVIDIA Device Plugin
   - 确认节点标签和污点配置
   - 验证GPU资源在节点上是否可见

2. **作业创建失败**
   - 检查RBAC权限配置
   - 验证Docker镜像是否可访问
   - 确认资源配额是否足够

3. **NFS挂载失败**
   - 检查NFS服务器连接
   - 验证PersistentVolume配置
   - 确认网络策略允许访问

4. **Python依赖安装失败**
   - 检查网络连接和代理设置
   - 验证pip源配置
   - 使用预构建镜像减少安装时间

### 调试命令

```bash
# 检查系统健康
curl http://localhost:8080/api/v1/jupyterhub/health

# 查看详细日志
kubectl logs -f deployment/ai-infra-matrix-backend

# 检查网络连接
kubectl exec -it test-pod -- ping nfs-server.default.svc.cluster.local

# 测试GPU功能
kubectl run gpu-test --image=nvidia/cuda:11.8-base --rm -it --restart=Never -- nvidia-smi
```

## 扩展开发

### 添加新的脚本类型支持

1. 继承`PythonScriptJob`结构
2. 实现特定的资源估算逻辑
3. 添加相应的Docker镜像
4. 扩展API端点

### 支持其他GPU厂商

1. 修改`analyzeGPUNode`函数
2. 添加特定的设备插件支持
3. 更新资源配额计算
4. 测试不同GPU类型

### 集成其他存储系统

1. 实现存储接口
2. 添加相应的Volume配置
3. 更新安全策略
4. 测试数据持久化

## 贡献指南

1. Fork项目仓库
2. 创建功能分支
3. 实现新功能或修复bug
4. 添加测试用例
5. 提交Pull Request

## 许可证

本项目采用MIT许可证，详见LICENSE文件。
