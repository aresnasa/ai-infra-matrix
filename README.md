# JupyterHub K8s GPU 集成示例

这个目录包含了在JupyterHub环境中使用Kubernetes GPU集群的完整示例。

## 目录结构

```
examples/
├── README.md                           # 本文件
├── gpu_performance_test.py            # GPU性能测试脚本
├── ml_training_example.py             # 机器学习训练示例
└── distributed_training_example.py    # 分布式训练示例
```

## 使用方法

### 1. GPU 性能测试

测试GPU环境和性能基准：

```python
# 在JupyterLab中运行
exec(open('/shared/examples/gpu_performance_test.py').read())
```

或者通过API提交为K8s Job：

```python
import requests

# 提交GPU性能测试任务
response = requests.post('http://localhost:8080/api/k8s/submit-job', json={
    "name": "gpu-performance-test",
    "script_path": "/shared/examples/gpu_performance_test.py",
    "gpu_required": True,
    "gpu_type": "any",
    "cpu_limit": "2",
    "memory_limit": "4Gi",
    "description": "GPU性能基准测试"
})

job_id = response.json()['job_id']
print(f"任务已提交，ID: {job_id}")
```

### 2. 机器学习训练

完整的CNN训练示例：

```python
# 直接运行
exec(open('/shared/examples/ml_training_example.py').read())
```

通过API提交：

```python
# 提交训练任务
response = requests.post('http://localhost:8080/api/k8s/submit-job', json={
    "name": "cnn-training",
    "script_path": "/shared/examples/ml_training_example.py",
    "gpu_required": True,
    "gpu_type": "any",
    "cpu_limit": "4",
    "memory_limit": "8Gi",
    "description": "CIFAR-10 CNN训练"
})
```

### 3. 分布式训练

多GPU分布式训练：

```python
# 提交分布式训练任务
response = requests.post('http://localhost:8080/api/k8s/submit-job', json={
    "name": "distributed-training",
    "script_path": "/shared/examples/distributed_training_example.py",
    "gpu_required": True,
    "gpu_count": 2,  # 请求2个GPU
    "gpu_type": "any",
    "cpu_limit": "8",
    "memory_limit": "16Gi",
    "description": "分布式CNN训练"
})
```

## 脚本说明

### gpu_performance_test.py

**功能**: 
- 检测GPU环境和配置
- 执行矩阵乘法性能测试
- 测试内存带宽
- 神经网络推理性能测试

**输出**: 
- 控制台性能报告
- `/shared/gpu_test_results.json` - 详细测试结果

**适用场景**:
- 验证GPU环境配置
- 性能基准测试
- 环境调试

### ml_training_example.py

**功能**:
- 完整的CNN模型训练流程
- CIFAR-10数据集训练
- 自动模型保存和性能记录

**输出**:
- `/shared/trained_model.pth` - 训练好的模型
- `/shared/training_results.json` - 训练日志和结果

**特点**:
- 自动GPU/CPU检测
- 数据加载失败时使用模拟数据
- 详细的训练过程监控

### distributed_training_example.py

**功能**:
- 多GPU分布式训练
- 使用PyTorch DDP (DistributedDataParallel)
- 自动GPU数量检测和进程管理

**输出**:
- `/shared/distributed_model.pth` - 分布式训练模型
- `/shared/distributed_training_results.json` - 分布式训练日志

**特点**:
- 支持任意数量GPU
- 自动负载均衡
- 同步训练状态

## 监控和调试

### 1. 查看任务状态

```python
# 获取任务状态
response = requests.get(f'http://localhost:8080/api/k8s/job/{job_id}/status')
status = response.json()
print(f"任务状态: {status['phase']}")
```

### 2. 获取任务日志

```python
# 获取任务输出
response = requests.get(f'http://localhost:8080/api/k8s/job/{job_id}/logs')
logs = response.json()
print("任务日志:")
print(logs['logs'])
```

### 3. 监控GPU使用

```python
# 获取集群GPU状态
response = requests.get('http://localhost:8080/api/k8s/gpu-status')
gpu_status = response.json()

for node in gpu_status['nodes']:
    print(f"节点: {node['name']}")
    for gpu in node['gpus']:
        print(f"  GPU {gpu['index']}: {gpu['type']} - {gpu['status']}")
```

## 环境要求

### Python包依赖

```bash
torch>=1.9.0
torchvision>=0.10.0
numpy
pandas
matplotlib
requests
ipywidgets
```

### Kubernetes资源

- GPU节点标签: `accelerator=nvidia-tesla-*` 或类似
- 存储: 共享存储卷 `/shared`
- RBAC: Job创建和监控权限

## 常见问题

### Q1: GPU检测失败
**A**: 检查CUDA驱动和PyTorch GPU支持：
```python
import torch
print(f"CUDA可用: {torch.cuda.is_available()}")
print(f"CUDA版本: {torch.version.cuda}")
```

### Q2: 内存不足错误
**A**: 减少批次大小或增加内存限制：
```python
# 在任务提交时增加内存
"memory_limit": "16Gi"  # 根据需要调整
```

### Q3: 分布式训练失败
**A**: 确保：
- 多GPU节点可用
- 网络通信正常
- NCCL后端支持

### Q4: 数据加载慢
**A**: 使用共享存储和数据预处理：
```python
# 预先下载数据到共享存储
import torchvision
torchvision.datasets.CIFAR10(root='/shared/data', download=True)
```

## 扩展示例

可以基于这些示例创建更复杂的场景：

1. **超参数调优**: 使用不同参数运行多个训练任务
2. **模型比较**: 并行训练不同模型架构
3. **数据管道**: 集成数据预处理和增强
4. **模型部署**: 训练完成后自动部署推理服务

## 技术支持

遇到问题时，请检查：
1. JupyterHub日志: `kubectl logs -f deployment/jupyterhub`
2. GPU节点状态: `kubectl describe nodes`
3. 任务状态: `kubectl get jobs -n ai-infra`
4. 存储挂载: `kubectl get pv,pvc`

---

更多信息请参考项目文档或联系运维团队。
