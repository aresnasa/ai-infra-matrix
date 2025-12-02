#!/bin/bash
# =============================================================================
# 快速测试主机文件解析调试接口
# =============================================================================
# 用法: ./test-debug-parse.sh [API_URL]
# =============================================================================

API_URL="${1:-http://localhost:8080}"

echo "=== 测试调试解析接口 ==="
echo "API: ${API_URL}/api/saltstack/hosts/parse/debug"
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
  -d "{\"content\": $(echo "$CSV_CONTENT" | jq -Rs '.'), \"filename\": \"test.csv\"}" | jq '.'

echo ""
echo "=== 测试完成 ==="
