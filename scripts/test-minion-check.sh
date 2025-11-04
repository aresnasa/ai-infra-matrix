#!/bin/bash
# 测试 Salt API 密钥检查功能

set -e

SALT_API_URL="${SALT_API_URL:-http://saltstack:8002}"
SALT_USERNAME="${SALT_API_USERNAME:-saltapi}"
SALT_PASSWORD="${SALT_API_PASSWORD:-your-salt-api-password}"
SALT_EAUTH="${SALT_API_EAUTH:-file}"

echo "=== 测试 Salt API 密钥检查 ==="
echo "Salt API URL: $SALT_API_URL"

# 1. 认证获取 token
echo -e "\n[1] 认证获取 token..."
AUTH_RESPONSE=$(curl -s -X POST "$SALT_API_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$SALT_USERNAME\",\"password\":\"$SALT_PASSWORD\",\"eauth\":\"$SALT_EAUTH\"}")

TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.return[0].token // empty')

if [ -z "$TOKEN" ]; then
  echo "❌ 认证失败"
  echo "响应: $AUTH_RESPONSE"
  exit 1
fi

echo "✓ 认证成功，token: ${TOKEN:0:20}..."

# 2. 获取密钥列表
echo -e "\n[2] 获取密钥列表..."
KEYS_RESPONSE=$(curl -s -X GET "$SALT_API_URL/keys" \
  -H "X-Auth-Token: $TOKEN")

echo "密钥列表响应:"
echo "$KEYS_RESPONSE" | jq .

# 解析已接受的 minions
ACCEPTED_MINIONS=$(echo "$KEYS_RESPONSE" | jq -r '.return[0].minions[]? // empty')
PENDING_MINIONS=$(echo "$KEYS_RESPONSE" | jq -r '.return[0].minions_pre[]? // empty')

echo -e "\n已接受的 Minions:"
echo "$ACCEPTED_MINIONS"

echo -e "\n等待接受的 Minions:"
if [ -z "$PENDING_MINIONS" ]; then
  echo "(无)"
else
  echo "$PENDING_MINIONS"
fi

# 3. 测试检查特定 minion
TEST_MINIONS=("test-ssh01" "test-ssh02" "test-ssh03")

echo -e "\n[3] 检查测试节点状态..."
for MINION in "${TEST_MINIONS[@]}"; do
  if echo "$ACCEPTED_MINIONS" | grep -q "^$MINION$"; then
    echo "✓ $MINION: 已接受"
  elif echo "$PENDING_MINIONS" | grep -q "^$MINION$"; then
    echo "⏳ $MINION: 等待接受"
  else
    echo "✗ $MINION: 未找到"
  fi
done

echo -e "\n=== 测试完成 ==="
