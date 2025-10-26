#!/bin/bash

# 快速测试扩容任务创建

BACKEND_URL="${BACKEND_URL:-http://localhost:3000}"

echo "=== 测试扩容任务创建 ==="
echo ""

curl -X POST "$BACKEND_URL/api/slurm/scaling/scale-up/async" \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": [
      {
        "host": "test-ssh01",
        "port": 22,
        "user": "root",
        "password": "rootpass123"
      }
    ]
  }' \
  2>&1 | tee /tmp/scale-up-response.json

echo ""
echo ""
echo "=== 响应内容 ==="
cat /tmp/scale-up-response.json | jq '.' 2>/dev/null || cat /tmp/scale-up-response.json

if grep -q "invalid input syntax for type bigint" /tmp/scale-up-response.json; then
    echo ""
    echo "❌ 错误仍然存在"
    echo ""
    echo "查看后端日志："
    echo "docker-compose logs --tail=100 backend | grep -A 10 -B 10 'invalid input'"
    exit 1
fi

if jq -e '.taskId' /tmp/scale-up-response.json > /dev/null 2>&1; then
    echo ""
    echo "✅ 任务创建成功"
    TASK_ID=$(jq -r '.taskId' /tmp/scale-up-response.json)
    echo "Task ID: $TASK_ID"
else
    echo ""
    echo "⚠️  响应格式不符合预期"
fi
