#!/bin/bash

# 批量测试 SLURM 节点安装 API
# 使用方法: ./test-batch-install-nodes.sh

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "========================================="
echo "批量安装 SLURM 节点测试"
echo "========================================="
echo "BASE_URL: $BASE_URL"
echo ""

# 1. 登录获取 token
echo "1. 登录获取 JWT token..."
TOKEN=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取 token"
    exit 1
fi
echo "✅ Token 获取成功"
echo ""

# 2. 批量安装节点
echo "2. 批量安装 SLURM 节点..."
echo "安装节点列表:"
echo "  - test-rocky01 (Rocky Linux)"
echo "  - test-rocky02 (Rocky Linux)"
echo "  - test-rocky03 (Rocky Linux)"
echo "  - test-ssh01 (Ubuntu)"
echo "  - test-ssh02 (Ubuntu)"
echo "  - test-ssh03 (Ubuntu)"
echo ""

REQUEST_BODY=$(cat <<EOF
{
  "nodes": [
    {"node_name": "test-rocky01", "os_type": "rocky"},
    {"node_name": "test-rocky02", "os_type": "rocky"},
    {"node_name": "test-rocky03", "os_type": "rocky"},
    {"node_name": "test-ssh01", "os_type": "ubuntu"},
    {"node_name": "test-ssh02", "os_type": "ubuntu"},
    {"node_name": "test-ssh03", "os_type": "ubuntu"}
  ]
}
EOF
)

echo "开始安装（这可能需要几分钟）..."
RESPONSE=$(curl -s -X POST "$BASE_URL/api/slurm/nodes/batch-install" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "$REQUEST_BODY")

echo ""
echo "========================================="
echo "安装结果"
echo "========================================="
echo "$RESPONSE" | jq '.'

# 提取统计信息
TOTAL=$(echo "$RESPONSE" | jq -r '.total')
SUCCESS_COUNT=$(echo "$RESPONSE" | jq -r '.success_count')
FAILURE_COUNT=$(echo "$RESPONSE" | jq -r '.failure_count')

echo ""
echo "========================================="
echo "统计信息"
echo "========================================="
echo "总计: $TOTAL"
echo "成功: $SUCCESS_COUNT"
echo "失败: $FAILURE_COUNT"

# 显示每个节点的详细结果
echo ""
echo "========================================="
echo "详细结果"
echo "========================================="
echo "$RESPONSE" | jq -r '.results | to_entries[] | "\(.key): \(if .value.success then "✅ 成功" else "❌ 失败" end) - \(.value.message)"'

echo ""
echo "========================================="
echo "3. 验证节点状态"
echo "========================================="
sleep 5

echo "检查 sinfo 输出:"
docker exec ai-infra-slurm-master sinfo

echo ""
echo "检查详细节点状态:"
docker exec ai-infra-slurm-master scontrol show nodes | grep -E "NodeName=|State="

echo ""
echo "========================================="
echo "4. 恢复节点（如果节点显示 DOWN）"
echo "========================================="
echo "如果节点显示 DOWN，执行以下命令恢复:"
echo "  docker exec ai-infra-slurm-master scontrol update nodename=test-rocky01,test-rocky02,test-rocky03,test-ssh01,test-ssh02,test-ssh03 state=idle"

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
