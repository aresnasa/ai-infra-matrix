#!/bin/bash
# 测试脚本：手动调用后端 API 在节点上安装 SLURM

set -e

BACKEND_URL="${BACKEND_URL:-http://localhost:8081}"
NODE_NAME="${1:-test-rocky02}"

echo "=========================================="
echo "测试在节点 $NODE_NAME 上安装 SLURM"
echo "后端 URL: $BACKEND_URL"
echo "=========================================="

# 1. 检查节点当前状态
echo ""
echo "[1] 检查节点当前状态..."
docker exec $NODE_NAME bash -c "ps aux | egrep 'slurm|munge' | grep -v grep" || echo "未发现 SLURM/Munge 进程"

# 2. 调用后端 API 添加节点（这会触发安装）
echo ""
echo "[2] 调用后端 API 添加节点..."
curl -X POST "$BACKEND_URL/api/slurm/nodes" \
  -H "Content-Type: application/json" \
  -d "{
    \"node_name\": \"$NODE_NAME\",
    \"host\": \"$NODE_NAME\",
    \"port\": 22,
    \"username\": \"root\",
    \"password\": \"rootpass123\",
    \"cpus\": 2,
    \"real_memory\": 2048,
    \"state\": \"IDLE\"
  }" || echo "API 调用可能失败或节点已存在"

# 3. 等待安装完成
echo ""
echo "[3] 等待10秒让安装完成..."
sleep 10

# 4. 检查安装后的状态
echo ""
echo "[4] 检查安装后的状态..."
echo "--- Munge 进程 ---"
docker exec $NODE_NAME bash -c "ps aux | grep munge | grep -v grep" || echo "未发现 Munge 进程"

echo ""
echo "--- SLURMD 进程 ---"
docker exec $NODE_NAME bash -c "ps aux | grep slurmd | grep -v grep" || echo "未发现 SLURMD 进程"

echo ""
echo "--- 检查配置文件 ---"
docker exec $NODE_NAME bash -c "ls -lh /etc/munge/munge.key" 2>/dev/null || echo "munge.key 不存在"
docker exec $NODE_NAME bash -c "ls -lh /etc/slurm/slurm.conf" 2>/dev/null || echo "slurm.conf 不存在"

# 5. 验证 Munge
echo ""
echo "[5] 验证 Munge..."
docker exec $NODE_NAME bash -c "munge -n | unmunge" 2>/dev/null && echo "✓ Munge 验证成功" || echo "✗ Munge 验证失败"

# 6. 检查 SLURM Master 是否能看到节点
echo ""
echo "[6] 检查 SLURM Master 节点列表..."
docker exec ai-infra-slurm-master sinfo

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
