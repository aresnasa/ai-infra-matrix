#!/bin/bash

# 测试节点 API 的 SaltStack 状态

BASE_URL="http://192.168.3.91:8080"

echo "=== 1. 登录获取 token ==="
TOKEN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.data.token // .token // empty')

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败"
  echo "$TOKEN_RESPONSE" | jq '.'
  exit 1
fi

echo "✅ 登录成功，获取到 token"

echo ""
echo "=== 2. 获取节点列表 ==="
NODES_RESPONSE=$(curl -s -X GET "${BASE_URL}/api/slurm/nodes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "$NODES_RESPONSE" | jq -r '.data[0:3] | .[] | "节点: \(.name)\n  SLURM状态: \(.state)\n  Salt状态: \(.salt_status // "undefined")\n  Minion ID: \(.salt_minion_id // "undefined")\n  Salt启用: \(.salt_enabled // "undefined")\n  错误信息: \(.salt_status_error // "none")\n"'

echo ""
echo "=== 3. 统计 Salt 状态 ==="
echo "$NODES_RESPONSE" | jq -r '.data | group_by(.salt_status) | .[] | "\(.[0].salt_status): \(length) 个节点"'

echo ""
echo "=== 4. SaltStack集成状态 ==="
SALT_RESPONSE=$(curl -s -X GET "${BASE_URL}/api/slurm/saltstack/integration" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "$SALT_RESPONSE" | jq '{
  master_status: .data.master_status,
  api_status: .data.api_status,
  total_minions: .data.minions.total,
  online_minions: .data.minions.online,
  minion_ids: .data.minion_list | map(.id)
}'

echo ""
echo "=== 5. 分析问题 ==="
UNKNOWN_COUNT=$(echo "$NODES_RESPONSE" | jq '.data | map(select(.salt_status == "unknown")) | length')
TOTAL_COUNT=$(echo "$NODES_RESPONSE" | jq '.data | length')

echo "节点总数: $TOTAL_COUNT"
echo "Unknown状态节点: $UNKNOWN_COUNT"

if [ "$UNKNOWN_COUNT" -gt 0 ]; then
  echo ""
  echo "⚠️  有 $UNKNOWN_COUNT 个节点状态为 unknown"
  echo ""
  echo "Unknown 状态的节点:"
  echo "$NODES_RESPONSE" | jq -r '.data | map(select(.salt_status == "unknown")) | .[] | "  - \(.name) (minion_id: \(.salt_minion_id // "无"))"'
fi
