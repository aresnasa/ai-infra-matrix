#!/bin/bash
# 测试 SLURM 扩容功能

set -e

BACKEND_URL="${BACKEND_URL:-http://192.168.0.200:8082}"

echo "=== 测试 SLURM 扩容功能 ==="
echo "Backend URL: $BACKEND_URL"
echo

# 0. 登录获取token
echo "0. 登录获取认证token..."
LOGIN_RESPONSE=$(curl -s -X POST "$BACKEND_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "✗ 登录失败: $LOGIN_RESPONSE"
  exit 1
fi

echo "✓ 登录成功，token: ${TOKEN:0:30}..."
echo

# 1. 使用异步API（推荐）
echo "1. 使用异步扩容API..."
echo "POST $BACKEND_URL/api/slurm/scaling/scale-up/async"
echo

ASYNC_RESPONSE=$(curl -s -X POST "$BACKEND_URL/api/slurm/scaling/scale-up/async" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "nodes": [
      {
        "host": "test-ssh01",
        "port": 22,
        "user": "root",
        "password": "test",
        "minion_id": "test-ssh01"
      },
      {
        "host": "test-ssh02",
        "port": 22,
        "user": "root",
        "password": "test",
        "minion_id": "test-ssh02"
      },
      {
        "host": "test-ssh03",
        "port": 22,
        "user": "root",
        "password": "test",
        "minion_id": "test-ssh03"
      }
    ]
  }')

echo "异步响应: $ASYNC_RESPONSE"
echo

# 提取任务ID
TASK_ID=$(echo "$ASYNC_RESPONSE" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -n "$TASK_ID" ]; then
  echo "✓ 扩容任务已创建，任务ID: $TASK_ID"
  echo
  echo "2. 查询任务状态..."
  sleep 3
  
  TASK_STATUS=$(curl -s "$BACKEND_URL/api/slurm/tasks/$TASK_ID")
  echo "任务状态: $TASK_STATUS"
  echo
  
  echo "3. 持续监控任务进度（30秒）..."
  for i in {1..6}; do
    sleep 5
    STATUS=$(curl -s "$BACKEND_URL/api/slurm/tasks/$TASK_ID" | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
    PROGRESS=$(curl -s "$BACKEND_URL/api/slurm/tasks/$TASK_ID" | grep -o '"progress":[0-9.]*' | cut -d':' -f2 || echo "0")
    echo "  [$i/6] 状态: $STATUS, 进度: $PROGRESS%"
    
    if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
      echo "  任务已结束"
      break
    fi
  done
else
  echo "✗ 未能创建扩容任务"
  echo "详细响应: $ASYNC_RESPONSE"
fi

echo
echo "=== 测试完成 ==="
echo
echo "提示："
echo "- 查看所有任务: curl $BACKEND_URL/api/slurm/tasks"
echo "- 查看Backend日志: docker logs -f ai-infra-backend --tail 100"
echo "- 查看节点状态: docker exec ai-infra-saltstack salt-key -L"
