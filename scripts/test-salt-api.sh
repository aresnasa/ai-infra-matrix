#!/bin/bash

# 测试 Salt API 执行命令
# 这个脚本从容器内部直接调用 Salt API 来验证功能

echo "========================================="
echo "测试 1: 获取 Salt API token"
echo "========================================="

TOKEN_RESPONSE=$(docker exec ai-infra-backend curl -s -X POST http://saltstack:8002/login \
  -H "Content-Type: application/json" \
  -d '{"username": "saltapi", "password": "your-salt-api-password", "eauth": "file"}')

echo "$TOKEN_RESPONSE" | jq .

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.return[0].token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ 无法获取 token"
    exit 1
fi

echo "✅ Token: $TOKEN"
echo ""

echo "========================================="
echo "测试 2: 执行 test.ping 命令"
echo "========================================="

RESULT=$(docker exec ai-infra-backend curl -s -X POST http://saltstack:8002/ \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: $TOKEN" \
  -d '[{"client": "local", "tgt": "*", "fun": "test.ping"}]')

echo "$RESULT" | jq .

echo ""

echo "========================================="
echo "测试 3: 执行 cmd.run 命令"
echo "========================================="

RESULT=$(docker exec ai-infra-backend curl -s -X POST http://saltstack:8002/ \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: $TOKEN" \
  -d '[{"client": "local", "tgt": "*", "fun": "cmd.run", "arg": ["uptime"]}]')

echo "$RESULT" | jq .

echo ""
echo "✅ 所有测试完成！"
