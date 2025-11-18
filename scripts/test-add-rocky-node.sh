#!/bin/bash
# 测试脚本：添加test-rocky01节点到SLURM集群

set -e

BACKEND_URL="http://localhost:8080"

echo "=== 测试添加test-rocky01节点 ==="

# 1. 获取cluster ID
echo "1. 获取cluster ID..."
CLUSTER_ID=$(curl -s "${BACKEND_URL}/api/v1/slurm/clusters" | jq -r '.data[0].id // 1')
echo "   Cluster ID: ${CLUSTER_ID}"

# 2. 添加节点
echo ""
echo "2. 添加test-rocky01节点..."
RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/v1/slurm/nodes" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": '${CLUSTER_ID}',
    "node_name": "test-rocky01",
    "node_type": "compute",
    "host": "test-rocky01",
    "port": 22,
    "username": "root",
    "auth_type": "password",
    "password": "root123",
    "cpus": 4,
    "memory": 4096,
    "storage": 50
  }')

echo "   Response: ${RESPONSE}"
NODE_ID=$(echo "${RESPONSE}" | jq -r '.data.id // empty')

if [ -z "${NODE_ID}" ]; then
    echo "   ❌ 添加节点失败"
    exit 1
fi

echo "   ✓ 节点已添加，ID: ${NODE_ID}"

# 3. 触发部署
echo ""
echo "3. 触发节点部署..."
DEPLOY_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/v1/slurm/nodes/${NODE_ID}/deploy" \
  -H "Content-Type: application/json")

echo "   Response: ${DEPLOY_RESPONSE}"

TASK_ID=$(echo "${DEPLOY_RESPONSE}" | jq -r '.data.task_id // empty')

if [ -z "${TASK_ID}" ]; then
    echo "   ❌ 触发部署失败"
    exit 1
fi

echo "   ✓ 部署任务已创建，Task ID: ${TASK_ID}"

# 4. 监控部署状态
echo ""
echo "4. 监控部署状态..."
for i in {1..30}; do
    sleep 2
    STATUS_RESPONSE=$(curl -s "${BACKEND_URL}/api/v1/slurm/nodes/${NODE_ID}")
    STATUS=$(echo "${STATUS_RESPONSE}" | jq -r '.data.status // "unknown"')
    MINION_ID=$(echo "${STATUS_RESPONSE}" | jq -r '.data.salt_minion_id // ""')
    
    echo "   [${i}/30] Status: ${STATUS}, Minion ID: ${MINION_ID}"
    
    if [ "${STATUS}" = "active" ]; then
        echo "   ✓ 节点部署成功！"
        break
    elif [ "${STATUS}" = "failed" ]; then
        echo "   ❌ 节点部署失败"
        exit 1
    fi
done

echo ""
echo "=== 测试完成 ==="
