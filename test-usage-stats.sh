#!/bin/bash

BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"

echo "🔐 登录获取 Token..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // .data.token // empty')

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败"
  echo "$LOGIN_RESPONSE" | jq
  exit 1
fi

echo "✅ Token: ${TOKEN:0:20}..."

echo ""
echo "📊 测试统计 API (路由 1: /api/ai/usage-stats)..."
STATS_RESPONSE_1=$(curl -s "$BASE_URL/api/ai/usage-stats" \
  -H "Authorization: Bearer $TOKEN")

echo "$STATS_RESPONSE_1" | jq

echo ""
echo "📊 测试统计 API (路由 2: /api/ai/system/usage)..."
STATS_RESPONSE_2=$(curl -s "$BASE_URL/api/ai/system/usage" \
  -H "Authorization: Bearer $TOKEN")

echo "$STATS_RESPONSE_2" | jq

echo ""
echo "✅ 测试完成"
