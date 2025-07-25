"""
机器学习训练示例
在Kubernetes集群上运行GPU训练任务的完整示例
"""

import torch
import torch.nn as nn
import torch.optim as optim
import torchvision
import torchvision.transforms as transforms
from torch.utils.data import DataLoader
import time
import json
import os
from datetime import datetime

class SimpleCNN(nn.Module):
    def __init__(self, num_classes=10):
        super(SimpleCNN, self).__init__()
        self.features = nn.Sequential(
            nn.Conv2d(3, 32, kernel_size=3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(kernel_size=2, stride=2),
            nn.Conv2d(32, 64, kernel_size=3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(kernel_size=2, stride=2),
            nn.Conv2d(64, 128, kernel_size=3, padding=1),
            nn.ReLU(inplace=True),
            nn.MaxPool2d(kernel_size=2, stride=2),
        )
        self.classifier = nn.Sequential(
            nn.Dropout(0.5),
            nn.Linear(128 * 4 * 4, 512),
            nn.ReLU(inplace=True),
            nn.Dropout(0.5),
            nn.Linear(512, num_classes),
        )
    
    def forward(self, x):
        x = self.features(x)
        x = x.view(x.size(0), -1)
        x = self.classifier(x)
        return x

def train_model():
    print("=== 机器学习训练示例 ===")
    print(f"开始时间: {datetime.now()}")
    
    # 设备检测
    device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    print(f"使用设备: {device}")
    
    if torch.cuda.is_available():
        print(f"GPU: {torch.cuda.get_device_name(0)}")
        print(f"显存: {torch.cuda.get_device_properties(0).total_memory / 1e9:.1f} GB")
    
    # 数据准备
    print("\n=== 数据准备 ===")
    transform = transforms.Compose([
        transforms.ToTensor(),
        transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))
    ])
    
    # 使用CIFAR-10数据集
    try:
        # 尝试加载数据集
        trainset = torchvision.datasets.CIFAR10(
            root='/shared/data', 
            train=True,
            download=True, 
            transform=transform
        )
        trainloader = DataLoader(
            trainset, 
            batch_size=128, 
            shuffle=True, 
            num_workers=2
        )
        
        testset = torchvision.datasets.CIFAR10(
            root='/shared/data', 
            train=False,
            download=True, 
            transform=transform
        )
        testloader = DataLoader(
            testset, 
            batch_size=128, 
            shuffle=False, 
            num_workers=2
        )
        
        print(f"训练样本数: {len(trainset)}")
        print(f"测试样本数: {len(testset)}")
        
    except Exception as e:
        print(f"数据加载失败: {e}")
        print("创建模拟数据...")
        
        # 创建模拟数据
        train_data = torch.randn(1000, 3, 32, 32)
        train_labels = torch.randint(0, 10, (1000,))
        train_dataset = torch.utils.data.TensorDataset(train_data, train_labels)
        trainloader = DataLoader(train_dataset, batch_size=128, shuffle=True)
        
        test_data = torch.randn(200, 3, 32, 32)
        test_labels = torch.randint(0, 10, (200,))
        test_dataset = torch.utils.data.TensorDataset(test_data, test_labels)
        testloader = DataLoader(test_dataset, batch_size=128, shuffle=False)
    
    # 模型准备
    print("\n=== 模型初始化 ===")
    model = SimpleCNN(num_classes=10).to(device)
    criterion = nn.CrossEntropyLoss()
    optimizer = optim.Adam(model.parameters(), lr=0.001)
    
    # 计算模型参数数量
    total_params = sum(p.numel() for p in model.parameters())
    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)
    print(f"总参数数: {total_params:,}")
    print(f"可训练参数数: {trainable_params:,}")
    
    # 训练
    print("\n=== 开始训练 ===")
    num_epochs = 5
    training_log = []
    
    for epoch in range(num_epochs):
        model.train()
        running_loss = 0.0
        correct = 0
        total = 0
        epoch_start_time = time.time()
        
        for i, (inputs, labels) in enumerate(trainloader):
            inputs, labels = inputs.to(device), labels.to(device)
            
            optimizer.zero_grad()
            outputs = model(inputs)
            loss = criterion(outputs, labels)
            loss.backward()
            optimizer.step()
            
            running_loss += loss.item()
            _, predicted = outputs.max(1)
            total += labels.size(0)
            correct += predicted.eq(labels).sum().item()
            
            if i % 20 == 19:  # 每20个批次打印一次
                print(f'Epoch [{epoch+1}/{num_epochs}], Step [{i+1}/{len(trainloader)}], '
                      f'Loss: {running_loss/20:.4f}, Acc: {100.*correct/total:.2f}%')
                running_loss = 0.0
        
        epoch_time = time.time() - epoch_start_time
        epoch_acc = 100. * correct / total
        
        # 验证
        model.eval()
        test_correct = 0
        test_total = 0
        test_loss = 0.0
        
        with torch.no_grad():
            for inputs, labels in testloader:
                inputs, labels = inputs.to(device), labels.to(device)
                outputs = model(inputs)
                loss = criterion(outputs, labels)
                test_loss += loss.item()
                _, predicted = outputs.max(1)
                test_total += labels.size(0)
                test_correct += predicted.eq(labels).sum().item()
        
        test_acc = 100. * test_correct / test_total
        
        epoch_log = {
            'epoch': epoch + 1,
            'train_acc': epoch_acc,
            'test_acc': test_acc,
            'train_loss': running_loss,
            'test_loss': test_loss / len(testloader),
            'epoch_time': epoch_time
        }
        training_log.append(epoch_log)
        
        print(f'Epoch [{epoch+1}/{num_epochs}] 完成')
        print(f'  训练精度: {epoch_acc:.2f}%')
        print(f'  测试精度: {test_acc:.2f}%')
        print(f'  耗时: {epoch_time:.2f}秒')
        print('-' * 50)
    
    # 保存模型和结果
    print("\n=== 保存结果 ===")
    output_dir = "/shared"
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
    
    # 保存模型
    model_path = os.path.join(output_dir, "trained_model.pth")
    torch.save(model.state_dict(), model_path)
    print(f"模型已保存: {model_path}")
    
    # 保存训练日志
    results = {
        'timestamp': datetime.now().isoformat(),
        'device': str(device),
        'model_params': total_params,
        'num_epochs': num_epochs,
        'training_log': training_log,
        'final_test_accuracy': test_acc
    }
    
    log_path = os.path.join(output_dir, "training_results.json")
    with open(log_path, 'w') as f:
        json.dump(results, f, indent=2)
    print(f"训练日志已保存: {log_path}")
    
    print(f"\n=== 训练完成 ===")
    print(f"最终测试精度: {test_acc:.2f}%")
    print(f"结束时间: {datetime.now()}")

if __name__ == "__main__":
    train_model()
