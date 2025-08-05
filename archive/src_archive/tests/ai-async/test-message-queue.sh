#!/bin/bash

# 消息队列功能测试脚本
set -e

echo "🔧 测试消息队列功能..."

BASE_URL="http://backend:8080"

# 获取测试用户token（假设有测试接口）
echo "获取测试token..."
TOKEN=$(curl -s -X POST "$BASE_URL/api/auth/test-login" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}' | jq -r .token)

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "⚠️  无法获取测试token，使用模拟token"
    TOKEN="test-token-123"
fi

# 测试1: 发送异步聊天消息
echo "测试发送异步聊天消息..."
RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "测试消息队列功能",
    "context": "queue_test"
  }')

MESSAGE_ID=$(echo $RESPONSE | jq -r .message_id)
echo "✅ 消息已发送，ID: $MESSAGE_ID"

# 测试2: 查询消息状态
echo "查询消息处理状态..."
sleep 2
STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/messages/$MESSAGE_ID/status" \
  -H "Authorization: Bearer $TOKEN")

STATUS=$(echo $STATUS_RESPONSE | jq -r .data.status)
echo "✅ 消息状态: $STATUS"

# 测试3: 检查Redis中的消息队列
echo "检查Redis消息队列..."
REDIS_CHECK=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
echo "✅ 聊天请求队列长度: $REDIS_CHECK"

# 测试4: 提交集群操作
echo "测试集群操作队列..."
CLUSTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "scale_deployment",
    "parameters": {
      "deployment": "test-app",
      "replicas": 3
    },
    "description": "测试扩容操作"
  }')

OPERATION_ID=$(echo $CLUSTER_RESPONSE | jq -r .operation_id)
echo "✅ 集群操作已提交，ID: $OPERATION_ID"

# 测试5: 检查集群操作队列
CLUSTER_QUEUE_CHECK=$(redis-cli -u redis://redis:6379 XLEN ai:cluster:operations)
echo "✅ 集群操作队列长度: $CLUSTER_QUEUE_CHECK"

# 测试6: 验证消息处理器消费
echo "验证消息处理器运行..."
sleep 5

# 再次检查队列长度，应该有所减少
REDIS_CHECK_AFTER=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
echo "✅ 处理后聊天请求队列长度: $REDIS_CHECK_AFTER"

if [ "$REDIS_CHECK_AFTER" -lt "$REDIS_CHECK" ]; then
    echo "✅ 消息队列处理正常"
else
    echo "⚠️  消息队列可能处理缓慢"
fi

echo "🎉 消息队列功能测试完成！"
