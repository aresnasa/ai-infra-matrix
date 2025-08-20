#!/usr/bin/env python3
"""
JupyterHub与Kubernetes GPU集成系统

项目概述：
本脚本实现了一个完整的系统，将JupyterHub项目集成到主项目中，
支持将Python脚本转换为Kubernetes GPU Job并提交到K8s集群。

主要功能：
- 🔗 JupyterHub项目集成和配置
- 🎯 GPU资源查询和智能调度
- 📦 Python脚本自动化容器化
- 🚀 K8s Job自动提交和监控
- 💾 NFS存储集成和结果管理
- 📊 完整的任务监控和日志系统

技术栈：
- Python: Kubernetes客户端、JupyterHub API
- Kubernetes: GPU节点调度、Job管理
- NFS: 分布式存储解决方案
- Docker: 容器化运行环境
"""

import subprocess
import sys
import os
import time
import json
import uuid
import yaml
from datetime import datetime
from typing import Dict, List, Optional, Any

def install_package(package):
    """安装Python包"""
    try:
        subprocess.check_call([sys.executable, "-m", "pip", "install", package])
        print(f"✅ {package} 安装成功")
    except subprocess.CalledProcessError:
        print(f"❌ {package} 安装失败")

def setup_environment():
    """环境设置和依赖安装"""
    print("🔧 设置环境和安装依赖...")
    
    # 安装主要依赖
    required_packages = [
        "kubernetes>=24.2.0",
        "pyyaml>=6.0",
        "requests>=2.28.0",
        "aiohttp>=3.8.0",
        "jinja2>=3.1.0",
        "psutil>=5.9.0",
        "docker>=6.0.0"
    ]
    
    for package in required_packages:
        install_package(package)
    
    # 设置环境变量
    os.environ.setdefault('AI_INFRA_API_URL', 'http://localhost:8080')
    os.environ.setdefault('JUPYTERHUB_K8S_NAMESPACE', 'jupyterhub-jobs')
    os.environ.setdefault('PYTHON_GPU_IMAGE', 'localhost:5000/jupyterhub-python-gpu:latest')
    os.environ.setdefault('PYTHON_BASE_IMAGE', 'localhost:5000/jupyterhub-python-cpu:latest')
    
    print("✅ 环境设置完成")

class JupyterHubK8sIntegration:
    """JupyterHub K8s GPU集成主类"""
    
    def __init__(self):
        self.api_url = os.environ.get('AI_INFRA_API_URL', 'http://localhost:8080')
        self.namespace = os.environ.get('JUPYTERHUB_K8S_NAMESPACE', 'jupyterhub-jobs')
        self.gpu_image = os.environ.get('PYTHON_GPU_IMAGE')
        self.base_image = os.environ.get('PYTHON_BASE_IMAGE')
        
        # 初始化K8s客户端
        try:
            from kubernetes import client, config
            config.load_incluster_config()  # 容器内配置
        except:
            try:
                config.load_kube_config()  # 本地配置
            except:
                print("⚠️ 无法加载Kubernetes配置")
        
        self.k8s_batch = client.BatchV1Api()
        self.k8s_core = client.CoreV1Api()
    
    def get_gpu_nodes(self) -> List[Dict]:
        """获取GPU节点列表"""
        try:
            nodes = self.k8s_core.list_node()
            gpu_nodes = []
            
            for node in nodes.items:
                labels = node.metadata.labels or {}
                
                # 检查GPU标签
                if any(key.startswith('accelerator') for key in labels.keys()):
                    gpu_info = {
                        'name': node.metadata.name,
                        'gpu_type': labels.get('accelerator', 'unknown'),
                        'gpu_count': int(labels.get('gpu-count', '1')),
                        'status': 'Ready' if any(
                            condition.type == 'Ready' and condition.status == 'True'
                            for condition in node.status.conditions
                        ) else 'NotReady'
                    }
                    gpu_nodes.append(gpu_info)
            
            return gpu_nodes
        except Exception as e:
            print(f"❌ 获取GPU节点失败: {e}")
            return []
    
    def create_gpu_job(self, script_content: str, job_name: str = None, 
                      gpu_required: bool = True, gpu_count: int = 1) -> str:
        """创建GPU作业"""
        if not job_name:
            job_name = f"jupyterhub-job-{int(time.time())}"
        
        # 选择镜像
        image = self.gpu_image if gpu_required else self.base_image
        
        # 创建Job配置
        job_config = {
            'apiVersion': 'batch/v1',
            'kind': 'Job',
            'metadata': {
                'name': job_name,
                'namespace': self.namespace
            },
            'spec': {
                'template': {
                    'spec': {
                        'containers': [{
                            'name': 'python-executor',
                            'image': image,
                            'command': ['python3', '-c'],
                            'args': [script_content],
                            'resources': {
                                'limits': {
                                    'nvidia.com/gpu': str(gpu_count) if gpu_required else '0',
                                    'memory': '8Gi',
                                    'cpu': '4'
                                },
                                'requests': {
                                    'memory': '4Gi',
                                    'cpu': '2'
                                }
                            },
                            'volumeMounts': [{
                                'name': 'shared-storage',
                                'mountPath': '/shared'
                            }]
                        }],
                        'volumes': [{
                            'name': 'shared-storage',
                            'nfs': {
                                'server': os.environ.get('NFS_SERVER', 'nfs-server'),
                                'path': os.environ.get('NFS_PATH', '/shared')
                            }
                        }],
                        'restartPolicy': 'Never',
                        'nodeSelector': {
                            'accelerator': 'nvidia'
                        } if gpu_required else {}
                    }
                },
                'backoffLimit': 3
            }
        }
        
        try:
            # 提交作业
            job = self.k8s_batch.create_namespaced_job(
                namespace=self.namespace,
                body=job_config
            )
            print(f"✅ 作业已提交: {job_name}")
            return job_name
        except Exception as e:
            print(f"❌ 作业提交失败: {e}")
            return None
    
    def get_job_status(self, job_name: str) -> Dict:
        """获取作业状态"""
        try:
            job = self.k8s_batch.read_namespaced_job(
                name=job_name,
                namespace=self.namespace
            )
            
            status = {
                'name': job_name,
                'active': job.status.active or 0,
                'succeeded': job.status.succeeded or 0,
                'failed': job.status.failed or 0,
                'completion_time': job.status.completion_time,
                'start_time': job.status.start_time
            }
            
            if status['succeeded'] > 0:
                status['phase'] = 'Succeeded'
            elif status['failed'] > 0:
                status['phase'] = 'Failed'
            elif status['active'] > 0:
                status['phase'] = 'Running'
            else:
                status['phase'] = 'Pending'
            
            return status
        except Exception as e:
            return {'name': job_name, 'phase': 'Unknown', 'error': str(e)}
    
    def get_job_logs(self, job_name: str) -> str:
        """获取作业日志"""
        try:
            # 获取Job对应的Pod
            pods = self.k8s_core.list_namespaced_pod(
                namespace=self.namespace,
                label_selector=f'job-name={job_name}'
            )
            
            if not pods.items:
                return "未找到相关Pod"
            
            pod_name = pods.items[0].metadata.name
            logs = self.k8s_core.read_namespaced_pod_log(
                name=pod_name,
                namespace=self.namespace
            )
            return logs
        except Exception as e:
            return f"获取日志失败: {e}"

def demo_gpu_test():
    """GPU性能测试示例"""
    script = '''
import torch
import time
from datetime import datetime

print(f"=== GPU性能测试 - {datetime.now()} ===")
print(f"CUDA可用: {torch.cuda.is_available()}")

if torch.cuda.is_available():
    device_count = torch.cuda.device_count()
    print(f"GPU数量: {device_count}")
    
    for i in range(device_count):
        print(f"GPU {i}: {torch.cuda.get_device_name(i)}")
    
    # 性能测试
    device = torch.device('cuda')
    size = 1000
    
    a = torch.randn(size, size, device=device)
    b = torch.randn(size, size, device=device)
    
    start_time = time.time()
    c = torch.mm(a, b)
    torch.cuda.synchronize()
    duration = time.time() - start_time
    
    print(f"矩阵乘法 ({size}x{size}): {duration:.4f}秒")
    print("GPU测试完成！")
else:
    print("GPU不可用，使用CPU测试")
    import torch
    a = torch.randn(500, 500)
    b = torch.randn(500, 500)
    start_time = time.time()
    c = torch.mm(a, b)
    duration = time.time() - start_time
    print(f"CPU矩阵乘法: {duration:.4f}秒")
'''
    return script

def demo_ml_training():
    """机器学习训练示例"""
    script = '''
import torch
import torch.nn as nn
import torch.optim as optim
from datetime import datetime

print(f"=== 机器学习训练示例 - {datetime.now()} ===")

# 简单的神经网络
class SimpleNet(nn.Module):
    def __init__(self):
        super(SimpleNet, self).__init__()
        self.fc1 = nn.Linear(784, 128)
        self.fc2 = nn.Linear(128, 64)
        self.fc3 = nn.Linear(64, 10)
        self.relu = nn.ReLU()
    
    def forward(self, x):
        x = self.relu(self.fc1(x))
        x = self.relu(self.fc2(x))
        return self.fc3(x)

device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
print(f"使用设备: {device}")

model = SimpleNet().to(device)
criterion = nn.CrossEntropyLoss()
optimizer = optim.Adam(model.parameters(), lr=0.001)

# 模拟训练数据
batch_size = 32
x = torch.randn(batch_size, 784).to(device)
y = torch.randint(0, 10, (batch_size,)).to(device)

# 训练循环
for epoch in range(10):
    optimizer.zero_grad()
    outputs = model(x)
    loss = criterion(outputs, y)
    loss.backward()
    optimizer.step()
    
    if epoch % 2 == 0:
        print(f"Epoch {epoch}, Loss: {loss.item():.4f}")

print("训练完成！")

# 保存模型
torch.save(model.state_dict(), '/shared/simple_model.pth')
print("模型已保存到 /shared/simple_model.pth")
'''
    return script

def main():
    """主函数"""
    print("🚀 JupyterHub K8s GPU集成系统")
    print("=" * 50)
    
    # 设置环境
    setup_environment()
    
    # 初始化集成系统
    integration = JupyterHubK8sIntegration()
    
    # 显示GPU节点信息
    print("\n📊 GPU节点状态:")
    gpu_nodes = integration.get_gpu_nodes()
    if gpu_nodes:
        for node in gpu_nodes:
            print(f"  {node['name']}: {node['gpu_type']} ({node['gpu_count']}x GPU) - {node['status']}")
    else:
        print("  未检测到GPU节点")
    
    # 交互式选择
    print("\n🎯 选择操作:")
    print("1. 运行GPU性能测试")
    print("2. 运行机器学习训练示例")
    print("3. 自定义脚本")
    print("4. 查看作业状态")
    
    choice = input("\n请选择操作 (1-4): ").strip()
    
    if choice == '1':
        print("\n🔥 提交GPU性能测试作业...")
        script = demo_gpu_test()
        job_name = integration.create_gpu_job(script, 'gpu-performance-test', gpu_required=True)
        
    elif choice == '2':
        print("\n🤖 提交机器学习训练作业...")
        script = demo_ml_training()
        job_name = integration.create_gpu_job(script, 'ml-training-demo', gpu_required=True)
        
    elif choice == '3':
        print("\n📝 请输入Python脚本内容 (输入 'END' 结束):")
        lines = []
        while True:
            line = input()
            if line.strip() == 'END':
                break
            lines.append(line)
        script = '\n'.join(lines)
        
        job_name = input("作业名称: ").strip() or f"custom-job-{int(time.time())}"
        gpu_required = input("需要GPU? (y/n): ").strip().lower() == 'y'
        
        job_name = integration.create_gpu_job(script, job_name, gpu_required)
        
    elif choice == '4':
        job_name = input("作业名称: ").strip()
        if job_name:
            status = integration.get_job_status(job_name)
            print(f"\n📋 作业状态: {status}")
            
            if input("\n查看日志? (y/n): ").strip().lower() == 'y':
                logs = integration.get_job_logs(job_name)
                print(f"\n📄 作业日志:\n{logs}")
    
    if choice in ['1', '2', '3'] and job_name:
        print(f"\n✅ 作业已提交: {job_name}")
        
        # 监控作业状态
        if input("监控作业状态? (y/n): ").strip().lower() == 'y':
            print("\n⏳ 监控作业状态...")
            while True:
                status = integration.get_job_status(job_name)
                print(f"状态: {status['phase']} | 活跃: {status.get('active', 0)} | "
                      f"成功: {status.get('succeeded', 0)} | 失败: {status.get('failed', 0)}")
                
                if status['phase'] in ['Succeeded', 'Failed']:
                    print(f"\n🏁 作业完成: {status['phase']}")
                    
                    logs = integration.get_job_logs(job_name)
                    print(f"\n📄 作业日志:\n{logs}")
                    break
                
                time.sleep(5)

if __name__ == "__main__":
    main()
