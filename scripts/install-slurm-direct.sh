#!/bin/bash
# 直接通过 docker exec 安装 SLURM（不依赖 Salt API 和后端 API）

set -e

echo "=========================================="
echo "直接安装 SLURM 到所有节点"
echo "=========================================="
echo

echo "步骤 1: 在 Rocky Linux 节点上安装 SLURM 包"
echo "----------------------------------------"
for node in test-rocky01 test-rocky02 test-rocky03; do
    echo "安装 $node..."
    docker exec $node bash -c "
        dnf install -y epel-release && \
        dnf install -y slurm slurm-slurmd munge && \
        mkdir -p /var/spool/slurm/slurmd /var/log/slurm /var/run/slurm /etc/munge && \
        chmod 755 /var/spool/slurm/slurmd
    " || echo "  警告: $node 安装失败"
done

echo
echo "步骤 2: 在 Ubuntu 节点上安装 SLURM 包"
echo "----------------------------------------"
for node in test-ssh01 test-ssh02 test-ssh03; do
    echo "安装 $node..."
    docker exec $node bash -c "
        export DEBIAN_FRONTEND=noninteractive && \
        apt-get update -qq && \
        apt-get install -y slurm-client slurmd munge && \
        mkdir -p /var/spool/slurm/slurmd /var/log/slurm /var/run/slurm /etc/munge /etc/slurm-llnl && \
        chmod 755 /var/spool/slurm/slurmd
    " || echo "  警告: $node 安装失败"
done

echo
echo "步骤 3: 从 slurm-master 复制配置文件"
echo "----------------------------------------"

# 获取配置文件
echo "获取 slurm.conf..."
docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf > /tmp/slurm.conf

echo "获取 munge.key..."
docker exec ai-infra-slurm-master cat /etc/munge/munge.key > /tmp/munge.key

# 复制到 Rocky 节点
echo "配置 Rocky 节点..."
for node in test-rocky01 test-rocky02 test-rocky03; do
    echo "  配置 $node..."
    docker cp /tmp/slurm.conf $node:/etc/slurm/slurm.conf
    docker cp /tmp/munge.key $node:/etc/munge/munge.key
    docker exec $node bash -c "chown munge:munge /etc/munge/munge.key && chmod 400 /etc/munge/munge.key"
done

# 复制到 Ubuntu 节点
echo "配置 Ubuntu 节点..."
for node in test-ssh01 test-ssh02 test-ssh03; do
    echo "  配置 $node..."
    docker cp /tmp/slurm.conf $node:/etc/slurm-llnl/slurm.conf
    docker cp /tmp/munge.key $node:/etc/munge/munge.key
    docker exec $node bash -c "chown munge:munge /etc/munge/munge.key && chmod 400 /etc/munge/munge.key"
done

rm -f /tmp/slurm.conf /tmp/munge.key

echo
echo "步骤 4: 启动服务"
echo "----------------------------------------"

# 读取启动脚本
START_SCRIPT=$(cat src/backend/scripts/start-slurmd.sh)

echo "在所有节点上启动 slurmd..."
for node in test-rocky01 test-rocky02 test-rocky03 test-ssh01 test-ssh02 test-ssh03; do
    echo "  启动 $node..."
    docker exec $node bash -c "$START_SCRIPT" 2>&1 | sed 's/^/    /'
done

echo
echo "步骤 5: 验证安装"
echo "=========================================="
sleep 3

for node in test-rocky01 test-rocky02 test-rocky03 test-ssh01 test-ssh02 test-ssh03; do
    echo "=== $node ==="
    docker exec $node bash -c "
        if pgrep -x munged >/dev/null; then
            echo '  ✓ munged 运行中'
        else
            echo '  ✗ munged 未运行'
        fi
        
        if pgrep -x slurmd >/dev/null; then
            echo '  ✓ slurmd 运行中'
            ps aux | grep '[s]lurmd' | head -1
        else
            echo '  ✗ slurmd 未运行'
            if [ -f /var/log/slurm/slurmd.log ]; then
                echo '  最后错误:'
                tail -5 /var/log/slurm/slurmd.log | sed 's/^/    /'
            fi
        fi
    "
    echo
done

echo "=========================================="
echo "步骤 6: 更新 SLURM Master 配置"
echo "=========================================="
docker exec ai-infra-slurm-master scontrol reconfigure

echo
echo "步骤 7: 检查 SLURM 集群状态"
echo "=========================================="
docker exec ai-infra-slurm-master sinfo
echo
docker exec ai-infra-slurm-master scontrol show nodes | grep -E "NodeName|State"

echo
echo "=========================================="
echo "安装完成！"
echo "=========================================="
