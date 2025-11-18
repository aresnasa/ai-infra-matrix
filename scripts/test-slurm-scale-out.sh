#!/bin/bash
# 测试SLURM集群扩容功能
# 通过API添加test-ssh节点到SLURM集群

set -e

BACKEND_URL="http://localhost:8082"

echo "========================================="
echo "  SLURM集群扩容测试"
echo "========================================="

# 登录获取token
echo ""
echo "[1] 登录获取token..."
TOKEN=$(curl -s -X POST "${BACKEND_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}' | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "❌ 登录失败"
    exit 1
fi
echo "✓ 登录成功"

# 检查或创建SLURM集群
echo ""
echo "[2] 检查SLURM集群..."
CLUSTER_ID=$(curl -s -X GET "${BACKEND_URL}/api/slurm/clusters" \
  -H "Authorization: Bearer ${TOKEN}" | jq -r '.data[0].id // empty')

if [ -z "$CLUSTER_ID" ]; then
    echo "  创建新集群..."
    CLUSTER_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/slurm/clusters" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -d '{
        "name": "default-cluster",
        "description": "Default SLURM Cluster",
        "master_host": "ai-infra-slurm-master",
        "master_port": 6817
      }')
    CLUSTER_ID=$(echo "$CLUSTER_RESPONSE" | jq -r '.data.id // empty')
    echo "  ✓ 集群已创建，ID: ${CLUSTER_ID}"
else
    echo "  ✓ 使用现有集群，ID: ${CLUSTER_ID}"
fi

# 触发扩容（添加test-ssh节点）
echo ""
echo "[3] 触发SLURM集群扩容..."
echo "  添加节点: test-ssh01, test-ssh02, test-ssh03"

SCALE_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/slurm/clusters/${CLUSTER_ID}/scale-out" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": [
      {
        "node_name": "test-ssh01",
        "host": "test-ssh01",
        "port": 22,
        "username": "root",
        "password": "rootpass123",
        "cpus": 4,
        "memory": 4096,
        "node_type": "compute"
      },
      {
        "node_name": "test-ssh02",
        "host": "test-ssh02",
        "port": 22,
        "username": "root",
        "password": "rootpass123",
        "cpus": 4,
        "memory": 4096,
        "node_type": "compute"
      },
      {
        "node_name": "test-ssh03",
        "host": "test-ssh03",
        "port": 22,
        "username": "root",
        "password": "rootpass123",
        "cpus": 4,
        "memory": 4096,
        "node_type": "compute"
      }
    ],
    "partition_name": "compute"
  }')

echo "$SCALE_RESPONSE" | jq .

TASK_ID=$(echo "$SCALE_RESPONSE" | jq -r '.data.task_id // empty')
if [ -z "$TASK_ID" ]; then
    echo "❌ 扩容任务创建失败"
    exit 1
fi

echo "✓ 扩容任务已创建，Task ID: ${TASK_ID}"

# 监控任务进度
echo ""
echo "[4] 监控扩容进度（最多等待10分钟）..."
for i in {1..120}; do
    sleep 5
    
    TASK_STATUS=$(curl -s -X GET "${BACKEND_URL}/api/slurm/tasks/${TASK_ID}" \
      -H "Authorization: Bearer ${TOKEN}" | jq -r '.data.status // empty')
    
    echo "  [${i}/120] 任务状态: ${TASK_STATUS}"
    
    if [ "$TASK_STATUS" = "completed" ]; then
        echo "✓ 扩容任务完成"
        break
    elif [ "$TASK_STATUS" = "failed" ]; then
        echo "❌ 扩容任务失败"
        curl -s -X GET "${BACKEND_URL}/api/slurm/tasks/${TASK_ID}" \
          -H "Authorization: Bearer ${TOKEN}" | jq .
        exit 1
    fi
done

# 验证节点状态
echo ""
echo "[5] 验证节点状态..."
curl -s -X GET "${BACKEND_URL}/api/slurm/clusters/${CLUSTER_ID}/nodes" \
  -H "Authorization: Bearer ${TOKEN}" | jq -r '.data[] | "\(.node_name): \(.status)"'

# 验证Salt密钥
echo ""
echo "[6] 验证Salt密钥..."
docker exec ai-infra-saltstack salt-key -L

echo ""
echo "========================================="
echo "  扩容测试完成"
echo "========================================="
