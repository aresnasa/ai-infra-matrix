#!/bin/bash

# 异步API功能测试脚本
set -e

echo "🔌 测试异步API功能..."

BASE_URL="http://backend:8080"
TOKEN="test-token-123"

# 测试1: 快速聊天API
echo "测试快速聊天API..."

QUICK_CHAT_START=$(date +%s)
QUICK_CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "解释什么是Kubernetes",
    "context": "api_test"
  }')

QUICK_CHAT_END=$(date +%s)
QUICK_CHAT_TIME=$((QUICK_CHAT_END - QUICK_CHAT_START))

MESSAGE_ID=$(echo $QUICK_CHAT_RESPONSE | jq -r .message_id)
QUICK_STATUS=$(echo $QUICK_CHAT_RESPONSE | jq -r .status)

if [ "$QUICK_STATUS" = "pending" ] && [ "$MESSAGE_ID" != "null" ]; then
    echo "✅ 快速聊天API响应正常 (${QUICK_CHAT_TIME}s)"
    echo "  消息ID: $MESSAGE_ID"
else
    echo "❌ 快速聊天API失败"
    echo "Response: $QUICK_CHAT_RESPONSE"
    exit 1
fi

# 测试2: 消息状态轮询
echo "测试消息状态轮询..."

POLL_COUNT=0
MAX_POLLS=10
STATUS="pending"

while [ "$STATUS" = "pending" ] && [ $POLL_COUNT -lt $MAX_POLLS ]; do
    sleep 2
    POLL_COUNT=$((POLL_COUNT + 1))
    
    STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/messages/$MESSAGE_ID/status" \
      -H "Authorization: Bearer $TOKEN")
    
    STATUS=$(echo $STATUS_RESPONSE | jq -r .data.status)
    echo "  轮询 $POLL_COUNT: 状态 = $STATUS"
done

if [ "$STATUS" != "pending" ]; then
    echo "✅ 消息状态轮询成功，最终状态: $STATUS"
else
    echo "⚠️  消息处理超时，状态仍为: $STATUS"
fi

# 测试3: 对话消息API
echo "测试对话消息API..."

# 先创建对话
CONVERSATION_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/conversations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "config_id": 1,
    "title": "API测试对话",
    "context": "async_api_test"
  }')

CONVERSATION_ID=$(echo $CONVERSATION_RESPONSE | jq -r .data.id)

if [ "$CONVERSATION_ID" != "null" ] && [ -n "$CONVERSATION_ID" ]; then
    echo "✅ 对话创建成功，ID: $CONVERSATION_ID"
    
    # 发送对话消息
    CONVERSATION_MSG_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/conversations/$CONVERSATION_ID/messages" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "message": "这是一个异步对话消息测试"
      }')
    
    CONV_MESSAGE_ID=$(echo $CONVERSATION_MSG_RESPONSE | jq -r .message_id)
    CONV_STATUS=$(echo $CONVERSATION_MSG_RESPONSE | jq -r .status)
    
    if [ "$CONV_STATUS" = "pending" ] && [ "$CONV_MESSAGE_ID" != "null" ]; then
        echo "✅ 对话消息发送成功，ID: $CONV_MESSAGE_ID"
    else
        echo "⚠️  对话消息发送可能失败"
    fi
else
    echo "⚠️  无法创建测试对话，跳过对话消息测试"
fi

# 测试4: 集群操作API
echo "测试集群操作API..."

CLUSTER_OP_START=$(date +%s)
CLUSTER_OP_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "describe_deployment",
    "parameters": {
      "name": "test-deployment",
      "namespace": "default"
    },
    "cluster_id": 1,
    "description": "API测试描述部署"
  }')

CLUSTER_OP_END=$(date +%s)
CLUSTER_OP_TIME=$((CLUSTER_OP_END - CLUSTER_OP_START))

OPERATION_ID=$(echo $CLUSTER_OP_RESPONSE | jq -r .operation_id)
OP_STATUS=$(echo $CLUSTER_OP_RESPONSE | jq -r .status)

if [ "$OP_STATUS" = "pending" ] && [ "$OPERATION_ID" != "null" ]; then
    echo "✅ 集群操作API响应正常 (${CLUSTER_OP_TIME}s)"
    echo "  操作ID: $OPERATION_ID"
else
    echo "❌ 集群操作API失败"
    echo "Response: $CLUSTER_OP_RESPONSE"
fi

# 测试5: 批量操作测试
echo "测试批量操作..."

BATCH_START=$(date +%s)
BATCH_IDS=()

# 批量发送消息
for i in {1..3}; do
    BATCH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"message\": \"批量测试消息 $i\", \"context\": \"batch_test\"}")
    
    BATCH_ID=$(echo $BATCH_RESPONSE | jq -r .message_id)
    BATCH_IDS+=("$BATCH_ID")
    echo "  批量消息 $i ID: $BATCH_ID"
done

BATCH_END=$(date +%s)
BATCH_TIME=$((BATCH_END - BATCH_START))
echo "✅ 批量操作完成 (${BATCH_TIME}s)"

# 测试6: 批量状态查询
echo "测试批量状态查询..."

for ID in "${BATCH_IDS[@]}"; do
    BATCH_STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/messages/$ID/status" \
      -H "Authorization: Bearer $TOKEN")
    
    BATCH_STATUS=$(echo $BATCH_STATUS_RESPONSE | jq -r .data.status)
    echo "  消息 $ID 状态: $BATCH_STATUS"
done

# 测试7: API响应时间测试
echo "测试API响应时间..."

RESPONSE_TIMES=()

for i in {1..5}; do
    START_TIME=$(date +%s%N)
    
    curl -s "$BASE_URL/api/ai/async/health" \
      -H "Authorization: Bearer $TOKEN" > /dev/null
    
    END_TIME=$(date +%s%N)
    RESPONSE_TIME=$(((END_TIME - START_TIME) / 1000000))  # 转换为毫秒
    RESPONSE_TIMES+=("$RESPONSE_TIME")
    echo "  请求 $i: ${RESPONSE_TIME}ms"
done

# 计算平均响应时间
TOTAL_TIME=0
for TIME in "${RESPONSE_TIMES[@]}"; do
    TOTAL_TIME=$((TOTAL_TIME + TIME))
done
AVG_TIME=$((TOTAL_TIME / ${#RESPONSE_TIMES[@]}))
echo "✅ 平均响应时间: ${AVG_TIME}ms"

# 测试8: 错误处理和边界情况
echo "测试错误处理和边界情况..."

# 无效的消息内容
INVALID_MSG_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": ""}')

INVALID_ERROR=$(echo $INVALID_MSG_RESPONSE | jq -r .error)
if [ "$INVALID_ERROR" != "null" ]; then
    echo "✅ 空消息错误处理正常"
else
    echo "⚠️  空消息错误处理可能不完善"
fi

# 无效的JSON
INVALID_JSON_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{invalid json}')

if echo $INVALID_JSON_RESPONSE | grep -q "error"; then
    echo "✅ 无效JSON错误处理正常"
else
    echo "⚠️  无效JSON错误处理可能不完善"
fi

# 无授权访问
UNAUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Content-Type: application/json" \
  -d '{"message": "unauthorized test"}')

if echo $UNAUTH_RESPONSE | grep -q "error\|未授权\|unauthorized"; then
    echo "✅ 未授权访问错误处理正常"
else
    echo "⚠️  未授权访问错误处理可能不完善"
fi

# 测试9: 并发API调用
echo "测试并发API调用..."

CONCURRENT_START=$(date +%s)

# 并发调用API
for i in {1..10}; do
    curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"message\": \"并发API测试 $i\", \"context\": \"concurrent_api_test\"}" &
done

wait
CONCURRENT_END=$(date +%s)
CONCURRENT_TIME=$((CONCURRENT_END - CONCURRENT_START))
echo "✅ 并发API调用完成 (${CONCURRENT_TIME}s)"

# 测试10: 长时间运行测试
echo "测试长时间运行..."

LONG_RUNNING_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "full_cluster_scan",
    "parameters": {
      "scan_depth": "deep",
      "include_logs": true
    },
    "description": "长时间运行测试"
  }')

LONG_OP_ID=$(echo $LONG_RUNNING_RESPONSE | jq -r .operation_id)
if [ "$LONG_OP_ID" != "null" ]; then
    echo "✅ 长时间运行操作提交成功，ID: $LONG_OP_ID"
    
    # 监控状态变化
    for i in {1..5}; do
        sleep 3
        LONG_STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/operations/$LONG_OP_ID/status" \
          -H "Authorization: Bearer $TOKEN")
        LONG_STATUS=$(echo $LONG_STATUS_RESPONSE | jq -r .data.status)
        echo "  长时间操作状态检查 $i: $LONG_STATUS"
    done
else
    echo "⚠️  长时间运行操作提交失败"
fi

echo "🎉 异步API功能测试完成！"
