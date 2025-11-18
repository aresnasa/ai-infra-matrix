#!/bin/bash
# 测试 Salt API 密钥检查功能

set -e

echo "=== 测试 Salt API 密钥检查 ==="
echo

# 从环境变量获取配置
SALT_API_URL="${SALTSTACK_MASTER_URL:-http://saltstack:8002}"
SALT_USERNAME="${SALT_API_USERNAME:-saltapi}"
SALT_PASSWORD="${SALT_API_PASSWORD:-your-salt-api-password}"
SALT_EAUTH="${SALT_API_EAUTH:-file}"

echo "1. 测试 Salt API 认证..."
TOKEN_RESPONSE=$(curl -s -X POST "${SALT_API_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SALT_USERNAME}\",\"password\":\"${SALT_PASSWORD}\",\"eauth\":\"${SALT_EAUTH}\"}")

echo "认证响应: $TOKEN_RESPONSE"
echo

# 提取 token（兼容不同的 JSON 格式）
TOKEN=$(echo "$TOKEN_RESPONSE" | sed -n 's/.*"token": *"\([^"]*\)".*/\1/p')

if [ -z "$TOKEN" ]; then
  echo "❌ 认证失败，无法获取 token"
  echo "调试信息: $TOKEN_RESPONSE"
  exit 1
fi

echo "✓ 认证成功，token: ${TOKEN:0:20}..."
echo

echo "2. 获取密钥列表..."
KEYS_RESPONSE=$(curl -s -X GET "${SALT_API_URL}/keys" \
  -H "X-Auth-Token: ${TOKEN}")

echo "密钥列表响应: $KEYS_RESPONSE"
echo

echo "3. 检查特定 minion 状态..."
for minion in test-ssh01 test-ssh02 test-ssh03; do
  if echo "$KEYS_RESPONSE" | grep -q "\"$minion\""; then
    echo "✓ Minion $minion 存在于密钥列表中"
  else
    echo "✗ Minion $minion 未找到"
  fi
done

echo
echo "=== 测试完成 ==="
