#!/bin/bash
# 完整的Rocky节点部署测试脚本

set -e

BACKEND_URL="http://localhost:8080"

echo "========================================="
echo "  Rocky节点完整部署流程测试"
echo "========================================="

# 步骤1：检查AppHub元数据
echo ""
echo "[步骤1] 验证AppHub RPM元数据..."
if docker exec ai-infra-apphub test -d /usr/share/nginx/html/pkgs/saltstack-rpm/repodata; then
    echo "  ✓ AppHub RPM元数据存在"
    docker exec ai-infra-apphub ls -la /usr/share/nginx/html/pkgs/saltstack-rpm/repodata/ 2>&1 | head -5
else
    echo "  ❌ AppHub RPM元数据缺失！"
    echo "  需要重新构建AppHub镜像"
    exit 1
fi

# 步骤2：检查或创建集群
echo ""
echo "[步骤2] 检查SLURM集群..."
CLUSTER_COUNT=$(docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -t -c "SELECT COUNT(*) FROM slurm_clusters;")
echo "  集群数量: ${CLUSTER_COUNT}"

if [ "${CLUSTER_COUNT}" -eq 0 ]; then
    echo "  创建默认SLURM集群..."
    CLUSTER_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/v1/slurm/clusters" \
      -H "Content-Type: application/json" \
      -d '{
        "name": "default-cluster",
        "description": "Default SLURM Cluster",
        "master_host": "ai-infra-slurm-master",
        "master_port": 6817
      }')
    
    echo "  Response: ${CLUSTER_RESPONSE}"
    CLUSTER_ID=$(echo "${CLUSTER_RESPONSE}" | jq -r '.data.id // 1')
    echo "  ✓ 集群已创建，ID: ${CLUSTER_ID}"
else
    CLUSTER_ID=$(docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -t -c "SELECT id FROM slurm_clusters LIMIT 1;" | tr -d ' ')
    echo "  ✓ 使用现有集群，ID: ${CLUSTER_ID}"
fi

# 步骤3：添加test-rocky01节点
echo ""
echo "[步骤3] 添加test-rocky01节点..."
NODE_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/v1/slurm/nodes" \
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
  }' 2>&1)

echo "  Response: ${NODE_RESPONSE}"

NODE_ID=$(echo "${NODE_RESPONSE}" | jq -r '.data.id // empty')
if [ -z "${NODE_ID}" ]; then
    echo "  ❌ 添加节点失败"
    echo "  可能节点已存在，尝试查找现有节点..."
    NODE_ID=$(docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -t -c "SELECT id FROM slurm_nodes WHERE node_name='test-rocky01' AND deleted_at IS NULL LIMIT 1;" | tr -d ' ')
    if [ -z "${NODE_ID}" ]; then
        echo "  ❌ 无法获取节点ID"
        exit 1
    fi
    echo "  ✓ 使用现有节点，ID: ${NODE_ID}"
else
    echo "  ✓ 节点已添加，ID: ${NODE_ID}"
fi

# 步骤4：检查SSH连接
echo ""
echo "[步骤4] 验证SSH连接..."
if docker exec ai-infra-backend sh -c "sshpass -p 'root123' ssh -o StrictHostKeyChecking=no root@test-rocky01 'echo OK'" 2>&1 | grep -q "OK"; then
    echo "  ✓ SSH连接正常"
else
    echo "  ❌ SSH连接失败"
    exit 1
fi

# 步骤5：触发节点部署
echo ""
echo "[步骤5] 触发节点部署..."
DEPLOY_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/v1/slurm/nodes/${NODE_ID}/deploy" \
  -H "Content-Type: application/json" 2>&1)

echo "  Response: ${DEPLOY_RESPONSE}"

TASK_ID=$(echo "${DEPLOY_RESPONSE}" | jq -r '.data.task_id // empty')
if [ -z "${TASK_ID}" ]; then
    echo "  ❌ 触发部署失败"
    exit 1
fi

echo "  ✓ 部署任务已创建，Task ID: ${TASK_ID}"

# 步骤6：监控部署进度
echo ""
echo "[步骤6] 监控部署进度（最多60秒）..."
for i in {1..30}; do
    sleep 2
    
    # 查询节点状态
    NODE_STATUS=$(docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -t -c "SELECT status FROM slurm_nodes WHERE id=${NODE_ID};" | tr -d ' ')
    
    # 查询任务状态
    TASK_STATUS=$(docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -t -c "SELECT status FROM node_install_tasks WHERE task_id='${TASK_ID}';" | tr -d ' ')
    
    # 查询Salt Minion ID
    MINION_ID=$(docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -t -c "SELECT salt_minion_id FROM slurm_nodes WHERE id=${NODE_ID};" | tr -d ' ')
    
    echo "  [${i}/30] 节点状态: ${NODE_STATUS:-pending}, 任务状态: ${TASK_STATUS:-pending}, Minion ID: ${MINION_ID:-none}"
    
    if [ "${NODE_STATUS}" = "active" ]; then
        echo "  ✓ 节点部署成功！"
        break
    elif [ "${NODE_STATUS}" = "failed" ] || [ "${TASK_STATUS}" = "failed" ]; then
        echo "  ❌ 节点部署失败"
        echo ""
        echo "  查看错误信息:"
        docker exec ai-infra-postgres psql -U postgres -d ai_infra_matrix -c "SELECT error_message FROM node_install_tasks WHERE task_id='${TASK_ID}';"
        exit 1
    fi
done

# 步骤7：验证Salt密钥
echo ""
echo "[步骤7] 验证Salt密钥状态..."
docker exec ai-infra-saltstack salt-key -L

# 步骤8：测试Salt连接
echo ""
echo "[步骤8] 测试Salt连接..."
if docker exec ai-infra-saltstack salt 'test-rocky01' test.ping --timeout=5 2>&1 | grep -q "True"; then
    echo "  ✓ Salt连接正常"
else
    echo "  ⚠️  Salt连接失败（可能还在初始化）"
fi

echo ""
echo "========================================="
echo "  部署流程完成"
echo "========================================="
