#!/bin/bash
# =============================================================================
# 快速测试主机文件解析调试接口
# =============================================================================
# 用法: ./test-debug-parse.sh [API_URL] [USERNAME] [PASSWORD]
# =============================================================================

API_URL="${1:-http://localhost:8080}"
USERNAME="${2:-admin}"
PASSWORD="${3:-admin123}"

echo "=== 测试调试解析接口 ==="
echo "API: ${API_URL}/api/saltstack/hosts/parse/debug"
echo "用户: ${USERNAME}"
echo ""

# 登录获取 Token
echo "正在登录..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // .data.token // empty')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
  echo "❌ 登录失败"
  echo "响应: $LOGIN_RESPONSE"
  exit 1
fi

echo "✓ 登录成功"
echo ""

# CSV 测试内容
CSV_CONTENT='host,port,username,password,use_sudo,minion_id,group
192.168.1.10,22,root,password123,false,minion-01,webservers
192.168.1.11,22,admin,password456,true,minion-02,databases
192.168.1.12,2222,deploy,password789,true,minion-03,webservers'

echo "发送 CSV 内容..."
echo ""

# 发送请求
curl -s -X POST "${API_URL}/api/saltstack/hosts/parse/debug" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"content\": $(echo "$CSV_CONTENT" | jq -Rs '.'), \"filename\": \"test.csv\"}" | jq '.'

echo ""
echo "=== 测试完成 ==="
