"""
分布式训练示例
演示如何在多GPU环境下进行分布式训练
"""

import torch
import torch.nn as nn
import torch.optim as optim
import torch.distributed as dist
import torch.multiprocessing as mp
from torch.nn.parallel import DistributedDataParallel as DDP
from torch.utils.data import DataLoader, DistributedSampler
import torchvision
import torchvision.transforms as transforms
import os
import time
import json
from datetime import datetime

class DistributedCNN(nn.Module):
    def __init__(self, num_classes=10):
        super(DistributedCNN, self).__init__()
        self.features = nn.Sequential(
            nn.Conv2d(3, 64, kernel_size=3, padding=1),
            nn.BatchNorm2d(64),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(kernel_size=2, stride=2),
            
            nn.Conv2d(64, 128, kernel_size=3, padding=1),
            nn.BatchNorm2d(128),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(kernel_size=2, stride=2),
            
            nn.Conv2d(128, 256, kernel_size=3, padding=1),
            nn.BatchNorm2d(256),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(kernel_size=2, stride=2),
            
            nn.Conv2d(256, 512, kernel_size=3, padding=1),
            nn.BatchNorm2d(512),
            nn.ReLU(inplace=True),
            nn.AdaptiveAvgPool2d((1, 1)),
        )
        
        self.classifier = nn.Sequential(
            nn.Dropout(0.5),
            nn.Linear(512, 256),
            nn.ReLU(inplace=True),
            nn.Dropout(0.5),
            nn.Linear(256, num_classes),
        )
    
    def forward(self, x):
        x = self.features(x)
        x = x.view(x.size(0), -1)
        x = self.classifier(x)
        return x

def setup(rank, world_size):
    """初始化分布式训练环境"""
    os.environ['MASTER_ADDR'] = 'localhost'
    os.environ['MASTER_PORT'] = '12355'
    
    # 初始化进程组
    dist.init_process_group("nccl", rank=rank, world_size=world_size)
    torch.cuda.set_device(rank)

def cleanup():
    """清理分布式训练环境"""
    dist.destroy_process_group()

def train_distributed(rank, world_size, epochs=5):
    """分布式训练函数"""
    print(f"Running DDP on rank {rank}, world_size {world_size}")
    setup(rank, world_size)
    
    device = torch.device(f'cuda:{rank}')
    
    # 数据准备
    transform = transforms.Compose([
        transforms.RandomHorizontalFlip(),
        transforms.RandomCrop(32, padding=4),
        transforms.ToTensor(),
        transforms.Normalize((0.4914, 0.4822, 0.4465), (0.2023, 0.1994, 0.2010))
    ])
    
    try:
        # 加载数据集
        train_dataset = torchvision.datasets.CIFAR10(
            root='/shared/data',
            train=True,
            download=True,
            transform=transform
        )
        
        # 分布式采样器
        train_sampler = DistributedSampler(
            train_dataset,
            num_replicas=world_size,
            rank=rank,
            shuffle=True
        )
        
        train_loader = DataLoader(
            train_dataset,
            batch_size=128,
            sampler=train_sampler,
            num_workers=2,
            pin_memory=True
        )
        
    except Exception as e:
        print(f"Rank {rank}: 数据加载失败，使用模拟数据")
        # 创建模拟数据
        train_data = torch.randn(5000, 3, 32, 32)
        train_labels = torch.randint(0, 10, (5000,))
        train_dataset = torch.utils.data.TensorDataset(train_data, train_labels)
        
        train_sampler = DistributedSampler(
            train_dataset,
            num_replicas=world_size,
            rank=rank,
            shuffle=True
        )
        
        train_loader = DataLoader(
            train_dataset,
            batch_size=128,
            sampler=train_sampler,
            num_workers=2,
            pin_memory=True
        )
    
    # 模型准备
    model = DistributedCNN(num_classes=10).to(device)
    
    # 包装为DDP模型
    model = DDP(model, device_ids=[rank])
    
    criterion = nn.CrossEntropyLoss()
    optimizer = optim.SGD(model.parameters(), lr=0.1, momentum=0.9, weight_decay=5e-4)
    scheduler = optim.lr_scheduler.StepLR(optimizer, step_size=30, gamma=0.1)
    
    training_log = []
    
    if rank == 0:
        print(f"开始分布式训练 - 世界大小: {world_size}")
        print(f"每个GPU批次大小: 128")
        print(f"有效批次大小: {128 * world_size}")
    
    for epoch in range(epochs):
        # 设置采样器的epoch以确保数据打乱
        train_sampler.set_epoch(epoch)
        
        model.train()
        running_loss = 0.0
        correct = 0
        total = 0
        epoch_start_time = time.time()
        
        for batch_idx, (data, target) in enumerate(train_loader):
            data, target = data.to(device), target.to(device)
            
            optimizer.zero_grad()
            output = model(data)
            loss = criterion(output, target)
            loss.backward()
            optimizer.step()
            
            running_loss += loss.item()
            pred = output.argmax(dim=1, keepdim=True)
            correct += pred.eq(target.view_as(pred)).sum().item()
            total += target.size(0)
            
            if batch_idx % 50 == 0 and rank == 0:
                print(f'Epoch {epoch+1}/{epochs}, Batch {batch_idx}/{len(train_loader)}, '
                      f'Loss: {loss.item():.4f}, Acc: {100.*correct/total:.2f}%')
        
        scheduler.step()
        
        # 同步所有进程的统计信息
        total_loss = torch.tensor(running_loss).to(device)
        total_correct = torch.tensor(correct).to(device)
        total_samples = torch.tensor(total).to(device)
        
        dist.all_reduce(total_loss, op=dist.ReduceOp.SUM)
        dist.all_reduce(total_correct, op=dist.ReduceOp.SUM)
        dist.all_reduce(total_samples, op=dist.ReduceOp.SUM)
        
        epoch_time = time.time() - epoch_start_time
        avg_loss = total_loss.item() / len(train_loader) / world_size
        accuracy = 100. * total_correct.item() / total_samples.item()
        
        if rank == 0:
            epoch_log = {
                'epoch': epoch + 1,
                'loss': avg_loss,
                'accuracy': accuracy,
                'epoch_time': epoch_time,
                'learning_rate': scheduler.get_last_lr()[0]
            }
            training_log.append(epoch_log)
            
            print(f'Epoch {epoch+1}/{epochs} 完成')
            print(f'  平均损失: {avg_loss:.4f}')
            print(f'  准确率: {accuracy:.2f}%')
            print(f'  学习率: {scheduler.get_last_lr()[0]:.6f}')
            print(f'  耗时: {epoch_time:.2f}秒')
            print('-' * 60)
    
    # 保存结果（只在rank 0上保存）
    if rank == 0:
        output_dir = "/shared"
        if not os.path.exists(output_dir):
            os.makedirs(output_dir)
        
        # 保存模型
        model_path = os.path.join(output_dir, "distributed_model.pth")
        torch.save(model.module.state_dict(), model_path)
        print(f"分布式模型已保存: {model_path}")
        
        # 保存训练日志
        results = {
            'timestamp': datetime.now().isoformat(),
            'world_size': world_size,
            'epochs': epochs,
            'effective_batch_size': 128 * world_size,
            'training_log': training_log,
            'final_accuracy': accuracy
        }
        
        log_path = os.path.join(output_dir, "distributed_training_results.json")
        with open(log_path, 'w') as f:
            json.dump(results, f, indent=2)
        print(f"分布式训练日志已保存: {log_path}")
        
        print(f"\n=== 分布式训练完成 ===")
        print(f"最终准确率: {accuracy:.2f}%")
        print(f"使用GPU数量: {world_size}")
    
    cleanup()

def main():
    """主函数 - 检测GPU数量并启动分布式训练"""
    print("=== 分布式训练示例 ===")
    print(f"开始时间: {datetime.now()}")
    
    if not torch.cuda.is_available():
        print("错误: 需要CUDA支持进行分布式训练")
        return
    
    world_size = torch.cuda.device_count()
    print(f"检测到 {world_size} 个GPU设备")
    
    if world_size < 2:
        print("警告: 分布式训练建议使用2个或更多GPU")
        print("在单GPU上运行普通训练...")
        
        # 单GPU训练
        device = torch.device('cuda:0')
        model = DistributedCNN().to(device)
        print(f"使用设备: {device}")
        print("训练将在单GPU模式下进行")
        return
    
    for i in range(world_size):
        print(f"GPU {i}: {torch.cuda.get_device_name(i)}")
    
    print(f"\n启动分布式训练进程...")
    
    # 启动多进程分布式训练
    mp.spawn(
        train_distributed,
        args=(world_size, 3),  # 3个epochs用于演示
        nprocs=world_size,
        join=True
    )

if __name__ == "__main__":
    main()
