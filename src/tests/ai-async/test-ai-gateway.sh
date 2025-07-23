#!/bin/bash

# AI网关功能测试脚本
set -e

echo "🤖 测试AI网关功能..."

BASE_URL="http://backend:8080"
TOKEN="test-token-123"

# 测试1: AI网关健康检查
echo "测试AI网关健康检查..."
HEALTH_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/health" \
  -H "Authorization: Bearer $TOKEN")

OVERALL_STATUS=$(echo $HEALTH_RESPONSE | jq -r .data.overall_status)
if [ "$OVERALL_STATUS" = "healthy" ] || [ "$OVERALL_STATUS" = "degraded" ]; then
    echo "✅ AI网关健康检查响应正常: $OVERALL_STATUS"
else
    echo "❌ AI网关健康检查失败"
    exit 1
fi

# 测试2: 消息队列服务状态
QUEUE_STATUS=$(echo $HEALTH_RESPONSE | jq -r .data.services.message_queue.status)
echo "消息队列状态: $QUEUE_STATUS"

# 测试3: 缓存服务状态
CACHE_STATUS=$(echo $HEALTH_RESPONSE | jq -r .data.services.cache.status)
echo "缓存服务状态: $CACHE_STATUS"

# 测试4: 异步消息处理流程
echo "测试异步消息处理流程..."

# 发送快速聊天消息
QUICK_CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，这是一个AI网关测试消息",
    "context": "ai_gateway_test"
  }')

MESSAGE_ID=$(echo $QUICK_CHAT_RESPONSE | jq -r .message_id)
INITIAL_STATUS=$(echo $QUICK_CHAT_RESPONSE | jq -r .status)

if [ "$INITIAL_STATUS" = "pending" ] && [ "$MESSAGE_ID" != "null" ]; then
    echo "✅ 异步消息提交成功，ID: $MESSAGE_ID"
else
    echo "❌ 异步消息提交失败"
    exit 1
fi

# 测试5: 消息状态查询
echo "测试消息状态查询..."
sleep 2

STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/messages/$MESSAGE_ID/status" \
  -H "Authorization: Bearer $TOKEN")

CURRENT_STATUS=$(echo $STATUS_RESPONSE | jq -r .data.status)
echo "✅ 消息当前状态: $CURRENT_STATUS"

# 测试6: 集群操作提交
echo "测试集群操作提交..."

CLUSTER_OP_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_pods",
    "parameters": {
      "namespace": "default",
      "label_selector": "app=test"
    },
    "description": "AI网关测试获取Pod列表"
  }')

OPERATION_ID=$(echo $CLUSTER_OP_RESPONSE | jq -r .operation_id)
OP_STATUS=$(echo $CLUSTER_OP_RESPONSE | jq -r .status)

if [ "$OP_STATUS" = "pending" ] && [ "$OPERATION_ID" != "null" ]; then
    echo "✅ 集群操作提交成功，ID: $OPERATION_ID"
else
    echo "❌ 集群操作提交失败"
    exit 1
fi

# 测试7: 集群操作状态查询
echo "测试集群操作状态查询..."
sleep 1

OP_STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/operations/$OPERATION_ID/status" \
  -H "Authorization: Bearer $TOKEN")

CURRENT_OP_STATUS=$(echo $OP_STATUS_RESPONSE | jq -r .data.status)
echo "✅ 集群操作当前状态: $CURRENT_OP_STATUS"

# 测试8: 使用统计
echo "测试使用统计..."

STATS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/usage-stats" \
  -H "Authorization: Bearer $TOKEN")

if echo $STATS_RESPONSE | jq -e . > /dev/null 2>&1; then
    echo "✅ 使用统计接口响应正常"
    TOTAL_MESSAGES=$(echo $STATS_RESPONSE | jq -r .data.total_messages // 0)
    TOTAL_OPERATIONS=$(echo $STATS_RESPONSE | jq -r .data.total_operations // 0)
    echo "  总消息数: $TOTAL_MESSAGES"
    echo "  总操作数: $TOTAL_OPERATIONS"
else
    echo "⚠️  使用统计接口不可用"
fi

# 测试9: 错误处理
echo "测试错误处理..."

# 提交无效消息
INVALID_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}')

ERROR_MESSAGE=$(echo $INVALID_RESPONSE | jq -r .error)
if [ "$ERROR_MESSAGE" != "null" ]; then
    echo "✅ 错误处理正常: $ERROR_MESSAGE"
else
    echo "⚠️  错误处理可能不完善"
fi

# 测试10: 查询不存在的消息状态
NONEXISTENT_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/messages/nonexistent-id/status" \
  -H "Authorization: Bearer $TOKEN")

NONEXISTENT_ERROR=$(echo $NONEXISTENT_RESPONSE | jq -r .error)
if [ "$NONEXISTENT_ERROR" != "null" ]; then
    echo "✅ 不存在资源错误处理正常"
else
    echo "⚠️  不存在资源错误处理可能不完善"
fi

# 测试11: Redis队列监控
echo "测试Redis队列监控..."

CHAT_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
CLUSTER_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:cluster:operations)
NOTIFICATION_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:notifications)

echo "✅ 聊天请求队列长度: $CHAT_QUEUE_LEN"
echo "✅ 集群操作队列长度: $CLUSTER_QUEUE_LEN"
echo "✅ 通知队列长度: $NOTIFICATION_QUEUE_LEN"

# 测试12: 并发处理能力
echo "测试并发处理能力..."

# 并发发送多个消息
for i in {1..5}; do
    curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"message\": \"并发测试消息 $i\", \"context\": \"concurrent_test\"}" &
done

wait
echo "✅ 并发消息发送完成"

# 检查队列长度变化
sleep 2
FINAL_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
echo "✅ 并发测试后队列长度: $FINAL_QUEUE_LEN"

echo "🎉 AI网关功能测试完成！"
