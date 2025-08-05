"""
GPU检测和性能测试脚本
用于验证GPU环境和执行基准测试
"""

import torch
import time
import numpy as np
from datetime import datetime
import json
import os

def main():
    print(f"=== GPU检测和性能测试 - {datetime.now()} ===")
    
    results = {
        'timestamp': datetime.now().isoformat(),
        'pytorch_version': torch.__version__,
        'cuda_available': torch.cuda.is_available(),
        'tests': []
    }
    
    # 基础信息检测
    print(f"PyTorch版本: {torch.__version__}")
    print(f"CUDA可用: {torch.cuda.is_available()}")
    
    if torch.cuda.is_available():
        device_count = torch.cuda.device_count()
        print(f"GPU设备数量: {device_count}")
        
        results['gpu_count'] = device_count
        results['gpu_devices'] = []
        
        for i in range(device_count):
            name = torch.cuda.get_device_name(i)
            props = torch.cuda.get_device_properties(i)
            total_memory = props.total_memory / 1e9
            
            print(f"GPU {i}: {name}")
            print(f"  总内存: {total_memory:.1f} GB")
            print(f"  计算能力: {props.major}.{props.minor}")
            
            results['gpu_devices'].append({
                'id': i,
                'name': name,
                'memory_gb': total_memory,
                'compute_capability': f"{props.major}.{props.minor}"
            })
        
        # 性能测试
        print("\n=== 性能测试 ===")
        device = torch.device('cuda')
        
        # 测试1: 矩阵乘法
        print("测试1: 矩阵乘法性能")
        sizes = [1000, 2000, 4000]
        
        for size in sizes:
            print(f"  矩阵大小: {size}x{size}")
            
            a = torch.randn(size, size, device=device)
            b = torch.randn(size, size, device=device)
            
            # 预热
            torch.mm(a, b)
            torch.cuda.synchronize()
            
            # 测试
            start_time = time.time()
            for _ in range(5):
                c = torch.mm(a, b)
            torch.cuda.synchronize()
            avg_time = (time.time() - start_time) / 5
            
            gflops = (2 * size**3) / (avg_time * 1e9)
            print(f"    平均耗时: {avg_time:.4f}秒")
            print(f"    性能: {gflops:.2f} GFLOPS")
            
            results['tests'].append({
                'test': 'matrix_multiplication',
                'size': size,
                'avg_time': avg_time,
                'gflops': gflops
            })
        
        # 测试2: 内存带宽
        print("\n测试2: 内存带宽")
        size = 100_000_000  # 100M elements
        
        a = torch.randn(size, device=device)
        b = torch.randn(size, device=device)
        
        torch.cuda.synchronize()
        start_time = time.time()
        
        for _ in range(10):
            c = a + b
        
        torch.cuda.synchronize()
        total_time = time.time() - start_time
        
        # 计算带宽 (GB/s)
        data_size = size * 4 * 3 * 10 / 1e9  # 4 bytes per float, 3 arrays, 10 iterations
        bandwidth = data_size / total_time
        
        print(f"  内存带宽: {bandwidth:.2f} GB/s")
        
        results['tests'].append({
            'test': 'memory_bandwidth',
            'bandwidth_gbs': bandwidth
        })
        
        # 测试3: 神经网络推理
        print("\n测试3: 神经网络推理性能")
        
        model = torch.nn.Sequential(
            torch.nn.Linear(1000, 512),
            torch.nn.ReLU(),
            torch.nn.Linear(512, 256),
            torch.nn.ReLU(),
            torch.nn.Linear(256, 10)
        ).to(device)
        
        batch_size = 1000
        x = torch.randn(batch_size, 1000, device=device)
        
        # 预热
        with torch.no_grad():
            model(x)
        torch.cuda.synchronize()
        
        # 测试
        start_time = time.time()
        with torch.no_grad():
            for _ in range(100):
                y = model(x)
        torch.cuda.synchronize()
        total_time = time.time() - start_time
        
        throughput = (batch_size * 100) / total_time
        print(f"  推理吞吐量: {throughput:.2f} samples/s")
        
        results['tests'].append({
            'test': 'neural_network_inference',
            'throughput_samples_per_sec': throughput
        })
        
    else:
        print("GPU不可用，使用CPU进行基础测试")
        results['gpu_count'] = 0
        
        # CPU测试
        print("\n=== CPU性能测试 ===")
        a = torch.randn(1000, 1000)
        b = torch.randn(1000, 1000)
        
        start_time = time.time()
        c = torch.mm(a, b)
        cpu_time = time.time() - start_time
        
        print(f"CPU矩阵乘法耗时: {cpu_time:.4f}秒")
        
        results['tests'].append({
            'test': 'cpu_matrix_multiplication',
            'time': cpu_time
        })
    
    # 保存结果
    output_dir = "/shared"
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
    
    output_file = os.path.join(output_dir, "gpu_test_results.json")
    with open(output_file, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"\n=== 测试完成 ===")
    print(f"结果已保存到: {output_file}")
    print(f"测试时间: {datetime.now()}")

if __name__ == "__main__":
    main()
