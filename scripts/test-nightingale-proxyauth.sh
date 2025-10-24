#!/bin/bash

echo "=== Nightingale ProxyAuth 问题诊断 ==="
echo ""

# Step 1: 登录获取 token
echo "Step 1: 登录获取 token..."
LOGIN_RESPONSE=$(curl -s -X POST http://192.168.0.200:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token // .token // empty')

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败！"
  echo "$LOGIN_RESPONSE" | jq '.'
  exit 1
fi

echo "✅ 登录成功"
echo "Token: ${TOKEN:0:30}..."
echo ""

# Step 2: 测试 /api/auth/verify
echo "Step 2: 测试 /api/auth/verify..."
VERIFY_RESPONSE=$(curl -s -i -H "Cookie: auth_token=$TOKEN" \
  http://192.168.0.200:8080/api/auth/verify 2>&1)

VERIFY_STATUS=$(echo "$VERIFY_RESPONSE" | grep "HTTP" | head -1)
echo "状态: $VERIFY_STATUS"

if echo "$VERIFY_STATUS" | grep -q "200"; then
  echo "✅ auth_request 验证成功"
  echo "$VERIFY_RESPONSE" | grep "^X-User:"
else
  echo "❌ auth_request 验证失败"
  echo "响应头:"
  echo "$VERIFY_RESPONSE" | grep "^HTTP\|^Content-Type\|^Content-Length" | head -5
fi
echo ""

# Step 3: 测试 /api/n9e/self/profile
echo "Step 3: 测试 Nightingale API..."
N9E_RESPONSE=$(curl -s -i -H "Cookie: auth_token=$TOKEN" \
  http://192.168.0.200:8080/api/n9e/self/profile 2>&1)

N9E_STATUS=$(echo "$N9E_RESPONSE" | grep "HTTP" | head -1)
echo "状态: $N9E_STATUS"

if echo "$N9E_STATUS" | grep -q "200"; then
  echo "✅ Nightingale API 成功"
elif echo "$N9E_STATUS" | grep -q "401"; then
  echo "❌ 401 - auth_request 验证失败，需要重新构建后端"
else
  echo "⚠️  非预期状态"
fi
echo ""

# Step 4: 检查 Nightingale 数据库
echo "Step 4: 检查 Nightingale 数据库..."
echo "用户列表:"
docker exec ai-infra-postgres psql -U postgres -d nightingale \
  -c "SELECT id, username, nickname, roles FROM users ORDER BY id;" 2>/dev/null

echo ""
echo "序列状态:"
docker exec ai-infra-postgres psql -U postgres -d nightingale \
  -c "SELECT last_value FROM users_id_seq;" 2>/dev/null

echo ""
echo "=== 诊断完成 ==="
