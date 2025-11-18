#!/bin/bash

# 在所有计算节点上安装 SLURM 客户端和 slurmd
# 这个脚本会：
# 1. 在 Rocky Linux 节点上安装 SLURM
# 2. 在 Ubuntu 节点上安装 SLURM  
# 3. 复制配置文件
# 4. 启动 slurmd 服务

set -e

echo "========================================="
echo "安装 SLURM 到所有计算节点"
echo "========================================="

# 获取 slurm master 的 slurm.conf
echo "1. 从 slurm-master 获取配置文件..."
docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf > /tmp/slurm.conf
docker exec ai-infra-slurm-master cat /etc/munge/munge.key > /tmp/munge.key 2>/dev/null || echo "Warning: No munge.key found"

# Rocky Linux 节点（test-rocky01-03）
ROCKY_NODES="test-rocky01 test-rocky02 test-rocky03"

for node in $ROCKY_NODES; do
    echo ""
    echo "========================================="
    echo "处理节点: $node (Rocky Linux 9)"
    echo "========================================="
    
    # 安装 EPEL 和 SLURM
    echo "安装 SLURM..."
    docker exec $node bash -c "
        # 安装 EPEL
        dnf install -y epel-release
        
        # 安装 SLURM
        dnf install -y slurm slurm-slurmd munge
        
        # 创建必要的目录
        mkdir -p /var/spool/slurm/slurmd /var/log/slurm /var/run/slurm /etc/munge
        
        # 设置权限
        chown -R slurm:slurm /var/spool/slurm /var/log/slurm /var/run/slurm
        chmod 755 /var/spool/slurm/slurmd
    " || { echo "警告: $node 安装失败，继续下一个..."; continue; }
    
    # 复制配置文件
    echo "复制配置文件..."
    docker cp /tmp/slurm.conf $node:/etc/slurm/slurm.conf
    
    # 如果有 munge.key，也复制
    if [ -f /tmp/munge.key ]; then
        docker cp /tmp/munge.key $node:/etc/munge/munge.key
        docker exec $node chown munge:munge /etc/munge/munge.key
        docker exec $node chmod 400 /etc/munge/munge.key
    fi
    
    echo "✅ $node 安装完成"
done

# Ubuntu 节点（test-ssh01-03）
UBUNTU_NODES="test-ssh01 test-ssh02 test-ssh03"

for node in $UBUNTU_NODES; do
    echo ""
    echo "========================================="
    echo "处理节点: $node (Ubuntu 22.04)"
    echo "========================================="
    
    # 安装 SLURM
    echo "安装 SLURM..."
    docker exec $node bash -c "
        export DEBIAN_FRONTEND=noninteractive
        
        # 更新包列表
        apt-get update -qq
        
        # 安装 SLURM
        apt-get install -y slurm-client slurmd munge
        
        # 创建必要的目录
        mkdir -p /var/spool/slurm/slurmd /var/log/slurm /var/run/slurm /etc/munge
        
        # 设置权限
        chown -R slurm:slurm /var/spool/slurm /var/log/slurm /var/run/slurm
        chmod 755 /var/spool/slurm/slurmd
    " || { echo "警告: $node 安装失败，继续下一个..."; continue; }
    
    # 复制配置文件
    echo "复制配置文件..."
    
    # 确保目录存在
    docker exec $node mkdir -p /etc/slurm-llnl /etc/munge
    
    # 复制 slurm.conf
    docker cp /tmp/slurm.conf $node:/etc/slurm-llnl/slurm.conf
    
    # 如果有 munge.key，也复制
    if [ -f /tmp/munge.key ]; then
        docker cp /tmp/munge.key $node:/etc/munge/munge.key
        docker exec $node chown munge:munge /etc/munge/munge.key
        docker exec $node chmod 400 /etc/munge/munge.key
    fi
    
    echo "✅ $node 安装完成"
done

echo ""
echo "========================================="
echo "启动所有节点的服务"
echo "========================================="

# 启动 Rocky Linux 节点的服务
for node in $ROCKY_NODES; do
    echo "启动 $node 的 munge 和 slurmd..."
    docker exec $node bash -c "
        # 启动 munge
        systemctl enable --now munge 2>/dev/null || /usr/sbin/munged || true
        sleep 1
        
        # 启动 slurmd
        systemctl enable --now slurmd 2>/dev/null || /usr/sbin/slurmd || true
        sleep 1
    " || echo "警告: $node 服务启动可能有问题"
done

# 启动 Ubuntu 节点的服务
for node in $UBUNTU_NODES; do
    echo "启动 $node 的 munge 和 slurmd..."
    docker exec $node bash -c "
        # 启动 munge
        systemctl enable --now munge 2>/dev/null || /usr/sbin/munged || true
        sleep 1
        
        # 启动 slurmd
        systemctl enable --now slurmd 2>/dev/null || /usr/sbin/slurmd -D & || true
        sleep 1
    " || echo "警告: $node 服务启动可能有问题"
done

echo ""
echo "========================================="
echo "等待节点注册到 SLURM..."
echo "========================================="
sleep 5

# 检查节点状态
echo ""
echo "检查 SLURM 节点状态:"
docker exec ai-infra-slurm-master sinfo

echo ""
echo "========================================="
echo "安装完成！"
echo "========================================="
echo ""
echo "如果节点仍然显示 DOWN，尝试在 slurm-master 上运行:"
echo "  docker exec ai-infra-slurm-master scontrol update nodename=test-rocky01,test-rocky02,test-rocky03,test-ssh01,test-ssh02,test-ssh03 state=idle"
echo ""
