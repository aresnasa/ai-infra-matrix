#!/bin/bash
# 通过 Salt API 批量安装 SLURM

set -e

echo "=========================================="
echo "通过 Salt API 批量安装 SLURM"
echo "=========================================="
echo

# Rocky Linux 节点
ROCKY_NODES="test-rocky01,test-rocky02,test-rocky03"
# Ubuntu 节点
UBUNTU_NODES="test-ssh01,test-ssh02,test-ssh03"

SALT_API="http://localhost:8002"

echo "步骤 1: 在 Rocky Linux 节点上安装 SLURM"
echo "----------------------------------------"
curl -sSk "$SALT_API/run" \
  -H "Content-Type: application/json" \
  -d '{
    "client": "local",
    "tgt": "'"$ROCKY_NODES"'",
    "tgt_type": "list",
    "fun": "cmd.run",
    "arg": ["dnf install -y epel-release && dnf install -y slurm slurm-slurmd munge && mkdir -p /var/spool/slurm/slurmd /var/log/slurm /var/run/slurm /etc/munge && chmod 755 /var/spool/slurm/slurmd"]
  }' | jq '.'

echo
echo "步骤 2: 在 Ubuntu 节点上安装 SLURM"
echo "----------------------------------------"
curl -sSk "$SALT_API/run" \
  -H "Content-Type: application/json" \
  -d '{
    "client": "local",
    "tgt": "'"$UBUNTU_NODES"'",
    "tgt_type": "list",
    "fun": "cmd.run",
    "arg": ["export DEBIAN_FRONTEND=noninteractive && apt-get update -qq && apt-get install -y slurm-client slurmd munge && mkdir -p /var/spool/slurm/slurmd /var/log/slurm /var/run/slurm /etc/munge /etc/slurm-llnl && chmod 755 /var/spool/slurm/slurmd"]
  }' | jq '.'

echo
echo "步骤 3: 从 slurm-master 复制配置文件"
echo "----------------------------------------"

# 获取 slurm.conf
echo "获取 slurm.conf..."
docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf > /tmp/slurm.conf

# 获取 munge.key
echo "获取 munge.key..."
docker exec ai-infra-slurm-master cat /etc/munge/munge.key > /tmp/munge.key

# 复制到 Rocky 节点
echo "复制配置到 Rocky 节点..."
for node in test-rocky01 test-rocky02 test-rocky03; do
    docker cp /tmp/slurm.conf $node:/etc/slurm/slurm.conf
    docker cp /tmp/munge.key $node:/etc/munge/munge.key
    docker exec $node chown munge:munge /etc/munge/munge.key
    docker exec $node chmod 400 /etc/munge/munge.key
done

# 复制到 Ubuntu 节点
echo "复制配置到 Ubuntu 节点..."
for node in test-ssh01 test-ssh02 test-ssh03; do
    docker cp /tmp/slurm.conf $node:/etc/slurm-llnl/slurm.conf
    docker cp /tmp/munge.key $node:/etc/munge/munge.key
    docker exec $node chown munge:munge /etc/munge/munge.key
    docker exec $node chmod 400 /etc/munge/munge.key
done

rm -f /tmp/slurm.conf /tmp/munge.key

echo
echo "步骤 4: 启动服务"
echo "----------------------------------------"

# 读取启动脚本
START_SCRIPT=$(cat src/backend/scripts/start-slurmd.sh)

echo "在所有节点上启动 slurmd..."
for node in test-rocky01 test-rocky02 test-rocky03 test-ssh01 test-ssh02 test-ssh03; do
    echo "启动 $node..."
    docker exec $node bash -c "$START_SCRIPT" || echo "  警告: $node 启动可能失败"
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
            ps aux | grep '[s]lurmd'
        else
            echo '  ✗ slurmd 未运行'
            if [ -f /var/log/slurm/slurmd.log ]; then
                echo '  日志:'
                tail -10 /var/log/slurm/slurmd.log | sed 's/^/    /'
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
docker exec ai-infra-slurm-master scontrol show nodes

echo
echo "=========================================="
echo "安装完成！"
echo "=========================================="
