#!/usr/bin/env python3
"""
JupyterHub K8s GPU集成系统 - 示例客户端
演示如何使用API提交Python脚本到K8s集群执行
"""

import json
import time
import requests
import argparse
from typing import Optional, Dict, Any

class JupyterHubK8sClient:
    """JupyterHub K8s API客户端"""
    
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip('/')
        self.api_base = f"{self.base_url}/api/v1/jupyterhub"
    
    def get_gpu_status(self) -> Dict[str, Any]:
        """获取GPU资源状态"""
        response = requests.get(f"{self.api_base}/gpu/status")
        response.raise_for_status()
        return response.json()
    
    def find_gpu_nodes(self, gpu_count: int = 1, gpu_type: str = "") -> Dict[str, Any]:
        """查找适合的GPU节点"""
        params = {"gpu_count": gpu_count}
        if gpu_type:
            params["gpu_type"] = gpu_type
        
        response = requests.get(f"{self.api_base}/gpu/nodes", params=params)
        response.raise_for_status()
        return response.json()
    
    def submit_python_script(self, 
                           name: str,
                           script: str,
                           requirements: Optional[list] = None,
                           gpu_required: bool = False,
                           gpu_count: int = 1,
                           gpu_type: str = "",
                           memory_mb: int = 1024,
                           cpu_cores: int = 1,
                           environment: Optional[Dict[str, str]] = None,
                           working_dir: str = "/workspace",
                           output_path: str = "/shared/output") -> Dict[str, Any]:
        """提交Python脚本作业"""
        
        data = {
            "name": name,
            "script": script,
            "requirements": requirements or [],
            "gpu_required": gpu_required,
            "gpu_count": gpu_count,
            "gpu_type": gpu_type,
            "memory_mb": memory_mb,
            "cpu_cores": cpu_cores,
            "environment": environment or {},
            "working_dir": working_dir,
            "output_path": output_path
        }
        
        response = requests.post(f"{self.api_base}/jobs/submit", json=data)
        response.raise_for_status()
        return response.json()
    
    def get_job_status(self, job_name: str) -> Dict[str, Any]:
        """获取作业状态"""
        response = requests.get(f"{self.api_base}/jobs/{job_name}/status")
        response.raise_for_status()
        return response.json()
    
    def wait_for_job_completion(self, job_name: str, timeout: int = 3600) -> Dict[str, Any]:
        """等待作业完成"""
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            status = self.get_job_status(job_name)
            
            if status["status"] in ["completed", "failed"]:
                return status
            
            print(f"作业状态: {status['status']}, 等待中...")
            time.sleep(10)
        
        raise TimeoutError(f"作业 {job_name} 在 {timeout} 秒内未完成")

def create_sample_scripts():
    """创建示例脚本"""
    
    # CPU密集型脚本
    cpu_script = """
import time
import numpy as np
import pandas as pd
from datetime import datetime

print(f"开始执行CPU密集型任务 - {datetime.now()}")

# 生成大量数据
print("生成随机数据...")
data = np.random.randn(10000, 100)
df = pd.DataFrame(data)

print("执行数据处理...")
# 计算统计信息
stats = {
    'mean': df.mean().mean(),
    'std': df.std().mean(),
    'max': df.max().max(),
    'min': df.min().min()
}

print("计算结果:")
for key, value in stats.items():
    print(f"  {key}: {value:.4f}")

# 模拟复杂计算
print("执行矩阵运算...")
result = np.dot(data, data.T)
eigenvalues = np.linalg.eigvals(result[:100, :100])

print(f"特征值范围: {eigenvalues.min():.2f} - {eigenvalues.max():.2f}")
print(f"任务完成 - {datetime.now()}")

# 保存结果
output = {
    'task_type': 'cpu_intensive',
    'data_shape': data.shape,
    'statistics': stats,
    'eigenvalue_count': len(eigenvalues),
    'completion_time': datetime.now().isoformat()
}

import json
with open('/shared/cpu_task_result.json', 'w') as f:
    json.dump(output, f, indent=2)

print("结果已保存到 /shared/cpu_task_result.json")
"""
    
    # GPU计算脚本
    gpu_script = """
import time
import numpy as np
from datetime import datetime

print(f"开始执行GPU计算任务 - {datetime.now()}")

try:
    import torch
    print(f"PyTorch版本: {torch.__version__}")
    print(f"CUDA可用: {torch.cuda.is_available()}")
    
    if torch.cuda.is_available():
        device = torch.device('cuda')
        print(f"GPU设备: {torch.cuda.get_device_name()}")
        print(f"GPU内存: {torch.cuda.get_device_properties(0).total_memory / 1e9:.1f} GB")
    else:
        device = torch.device('cpu')
        print("使用CPU计算")
    
    # 创建大型张量
    print("创建张量...")
    a = torch.randn(5000, 5000, device=device)
    b = torch.randn(5000, 5000, device=device)
    
    # GPU矩阵乘法
    print("执行矩阵乘法...")
    start_time = time.time()
    c = torch.mm(a, b)
    compute_time = time.time() - start_time
    
    print(f"计算时间: {compute_time:.2f}秒")
    print(f"结果张量形状: {c.shape}")
    print(f"结果统计: mean={c.mean().item():.4f}, std={c.std().item():.4f}")
    
    # 神经网络示例
    print("创建简单神经网络...")
    model = torch.nn.Sequential(
        torch.nn.Linear(5000, 1000),
        torch.nn.ReLU(),
        torch.nn.Linear(1000, 100),
        torch.nn.ReLU(),
        torch.nn.Linear(100, 1)
    ).to(device)
    
    # 前向传播
    print("执行前向传播...")
    x = torch.randn(100, 5000, device=device)
    y = model(x)
    
    print(f"网络输出形状: {y.shape}")
    print(f"网络输出范围: {y.min().item():.4f} - {y.max().item():.4f}")
    
    # 保存结果
    output = {
        'task_type': 'gpu_compute',
        'pytorch_version': torch.__version__,
        'cuda_available': torch.cuda.is_available(),
        'device': str(device),
        'compute_time': compute_time,
        'tensor_shape': list(c.shape),
        'network_output_shape': list(y.shape),
        'completion_time': datetime.now().isoformat()
    }
    
    if torch.cuda.is_available():
        output['gpu_name'] = torch.cuda.get_device_name()
        output['gpu_memory_gb'] = torch.cuda.get_device_properties(0).total_memory / 1e9

except ImportError as e:
    print(f"导入错误: {e}")
    output = {
        'task_type': 'gpu_compute',
        'error': str(e),
        'completion_time': datetime.now().isoformat()
    }

import json
with open('/shared/gpu_task_result.json', 'w') as f:
    json.dump(output, f, indent=2)

print("结果已保存到 /shared/gpu_task_result.json")
print(f"任务完成 - {datetime.now()}")
"""
    
    # 数据科学脚本
    datascience_script = """
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from sklearn.datasets import make_classification
from sklearn.model_selection import train_test_split
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import classification_report, confusion_matrix
from datetime import datetime
import json

print(f"开始执行数据科学任务 - {datetime.now()}")

# 生成示例数据集
print("生成分类数据集...")
X, y = make_classification(
    n_samples=10000,
    n_features=20,
    n_informative=15,
    n_redundant=5,
    n_classes=3,
    random_state=42
)

# 创建DataFrame
feature_names = [f'feature_{i}' for i in range(X.shape[1])]
df = pd.DataFrame(X, columns=feature_names)
df['target'] = y

print(f"数据集形状: {df.shape}")
print(f"目标分布:\\n{df['target'].value_counts()}")

# 数据分析
print("执行数据分析...")
correlation_matrix = df.corr()
high_corr_features = []
for i in range(len(correlation_matrix.columns)):
    for j in range(i+1, len(correlation_matrix.columns)):
        if abs(correlation_matrix.iloc[i, j]) > 0.8:
            high_corr_features.append((
                correlation_matrix.columns[i],
                correlation_matrix.columns[j],
                correlation_matrix.iloc[i, j]
            ))

print(f"高相关性特征对数量: {len(high_corr_features)}")

# 机器学习模型
print("训练随机森林模型...")
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)

rf_model = RandomForestClassifier(
    n_estimators=100,
    max_depth=10,
    random_state=42,
    n_jobs=-1
)

rf_model.fit(X_train, y_train)
y_pred = rf_model.predict(X_test)

# 模型评估
from sklearn.metrics import accuracy_score, f1_score
accuracy = accuracy_score(y_test, y_pred)
f1 = f1_score(y_test, y_pred, average='weighted')

print(f"模型准确率: {accuracy:.4f}")
print(f"F1分数: {f1:.4f}")

# 特征重要性
feature_importance = pd.DataFrame({
    'feature': feature_names,
    'importance': rf_model.feature_importances_
}).sort_values('importance', ascending=False)

print("\\n前5个重要特征:")
print(feature_importance.head())

# 保存结果
results = {
    'task_type': 'data_science',
    'dataset_shape': df.shape,
    'target_distribution': df['target'].value_counts().to_dict(),
    'high_correlation_pairs': len(high_corr_features),
    'model_performance': {
        'accuracy': accuracy,
        'f1_score': f1
    },
    'top_features': feature_importance.head(5).to_dict('records'),
    'completion_time': datetime.now().isoformat()
}

with open('/shared/datascience_task_result.json', 'w') as f:
    json.dump(results, f, indent=2)

# 保存特征重要性图
plt.figure(figsize=(10, 6))
sns.barplot(data=feature_importance.head(10), x='importance', y='feature')
plt.title('Top 10 Feature Importance')
plt.tight_layout()
plt.savefig('/shared/feature_importance.png', dpi=300, bbox_inches='tight')

print("结果已保存到 /shared/datascience_task_result.json")
print("特征重要性图已保存到 /shared/feature_importance.png")
print(f"任务完成 - {datetime.now()}")
"""
    
    return {
        "cpu_intensive": cpu_script,
        "gpu_compute": gpu_script,
        "data_science": datascience_script
    }

def main():
    parser = argparse.ArgumentParser(description="JupyterHub K8s GPU集成系统示例客户端")
    parser.add_argument("--url", default="http://localhost:8080", help="API服务器URL")
    parser.add_argument("--task", choices=["cpu", "gpu", "datascience", "status"], 
                       default="status", help="要执行的任务类型")
    parser.add_argument("--wait", action="store_true", help="等待作业完成")
    
    args = parser.parse_args()
    
    client = JupyterHubK8sClient(args.url)
    
    if args.task == "status":
        # 显示GPU资源状态
        try:
            print("=== GPU资源状态 ===")
            status = client.get_gpu_status()
            print(f"总GPU数: {status['total_gpus']}")
            print(f"可用GPU数: {status['available_gpus']}")
            print(f"已用GPU数: {status['used_gpus']}")
            print(f"上次更新: {status['last_updated']}")
            
            print("\\n=== GPU节点信息 ===")
            for node in status['gpu_nodes']:
                print(f"节点: {node['node_name']}")
                print(f"  GPU类型: {node['gpu_type']}")
                print(f"  GPU总数: {node['gpu_count']}")
                print(f"  可用GPU: {node['available_gpus']}")
                print(f"  可调度: {node['schedulable']}")
                
        except requests.RequestException as e:
            print(f"无法连接到API服务器: {e}")
            print("请确保服务器正在运行")
            return
    
    else:
        # 提交作业
        scripts = create_sample_scripts()
        
        if args.task == "cpu":
            script_config = {
                "name": "cpu-intensive-task",
                "script": scripts["cpu_intensive"],
                "requirements": ["numpy", "pandas"],
                "gpu_required": False,
                "memory_mb": 2048,
                "cpu_cores": 2
            }
        
        elif args.task == "gpu":
            script_config = {
                "name": "gpu-compute-task", 
                "script": scripts["gpu_compute"],
                "requirements": ["torch", "torchvision"],
                "gpu_required": True,
                "gpu_count": 1,
                "memory_mb": 4096,
                "cpu_cores": 2
            }
        
        elif args.task == "datascience":
            script_config = {
                "name": "datascience-task",
                "script": scripts["data_science"],
                "requirements": ["pandas", "numpy", "matplotlib", "seaborn", "scikit-learn"],
                "gpu_required": False,
                "memory_mb": 3072,
                "cpu_cores": 4
            }
        
        try:
            print(f"=== 提交{args.task}任务 ===")
            result = client.submit_python_script(**script_config)
            
            print(f"作业ID: {result['job_id']}")
            print(f"作业名称: {result['job_name']}")
            print(f"状态: {result['status']}")
            print(f"创建时间: {result['created_at']}")
            
            if args.wait:
                print("\\n等待作业完成...")
                final_status = client.wait_for_job_completion(result['job_name'])
                print(f"\\n最终状态: {final_status['status']}")
                if final_status.get('error_message'):
                    print(f"错误信息: {final_status['error_message']}")
            
        except requests.RequestException as e:
            print(f"提交作业失败: {e}")

if __name__ == "__main__":
    main()
