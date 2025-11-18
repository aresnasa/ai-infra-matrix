#!/bin/bash

echo "=== 测试 Nightingale SSO 认证流程 ==="

# Step 1: 登录获取token
echo -e "\nStep 1: 登录系统获取 JWT token..."
LOGIN_RESPONSE=$(curl -s -X POST http://192.168.18.154:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  -c /tmp/cookies.txt)

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token // .token // empty')

if [ -z "$TOKEN" ]; then
  echo "登录失败！响应:"
  echo "$LOGIN_RESPONSE" | jq '.'
  exit 1
fi

echo "✓ 登录成功"
echo "Token (前20字符): ${TOKEN:0:20}..."

# Step 2: 测试 /api/auth/verify 端点
echo -e "\nStep 2: 测试 /api/auth/verify 端点..."
VERIFY_RESPONSE=$(curl -s -i -H "Cookie: auth_token=$TOKEN" http://192.168.18.154:8080/api/auth/verify 2>&1)
VERIFY_STATUS=$(echo "$VERIFY_RESPONSE" | grep "HTTP" | head -1)

echo "$VERIFY_STATUS"
echo "$VERIFY_RESPONSE" | grep -E "^(X-User|X-WEBAUTH|HTTP)" | head -10

# Step 3: 测试访问 /nightingale/
echo -e "\nStep 3: 使用 Cookie 访问 /nightingale/..."
NIGHTINGALE_RESPONSE=$(curl -s -i -H "Cookie: auth_token=$TOKEN" http://192.168.18.154:8080/nightingale/ 2>&1 | head -30)

NIGHTINGALE_STATUS=$(echo "$NIGHTINGALE_RESPONSE" | grep "HTTP" | head -1)
echo "$NIGHTINGALE_STATUS"

if echo "$NIGHTINGALE_STATUS" | grep -q "200"; then
  echo "✓ Nightingale SSO 认证成功！"
  echo -e "\n响应头:"
  echo "$NIGHTINGALE_RESPONSE" | grep -E "^(Location|Set-Cookie|X-)" | head -10
elif echo "$NIGHTINGALE_STATUS" | grep -q "401"; then
  echo "✗ 401 Unauthorized - auth_request 验证失败"
  echo -e "\n可能的原因:"
  echo "1. auth_request 未被触发"
  echo "2. /__auth/verify 返回非 2xx"
  echo "3. X-User 头未正确提取"
elif echo "$NIGHTINGALE_STATUS" | grep -q "302\|301"; then
  echo "⚠ 重定向 - 可能被Nightingale的JWT认证拦截"
  LOCATION=$(echo "$NIGHTINGALE_RESPONSE" | grep -i "^Location:" | cut -d' ' -f2)
  echo "重定向到: $LOCATION"
else
  echo "✗ 非预期状态: $NIGHTINGALE_STATUS"
fi

echo -e "\n=== 测试完成 ==="
