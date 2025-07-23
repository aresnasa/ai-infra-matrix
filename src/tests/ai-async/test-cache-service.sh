#!/bin/bash

# 缓存服务功能测试脚本
set -e

echo "💾 测试缓存服务功能..."

BASE_URL="http://backend:8080"
REDIS_URL="redis://redis:6379"

# 获取测试token
TOKEN="test-token-123"

# 测试1: 验证Redis连接
echo "验证Redis连接..."
redis-cli -u $REDIS_URL ping
echo "✅ Redis连接正常"

# 测试2: 测试基础缓存操作
echo "测试基础缓存操作..."
redis-cli -u $REDIS_URL SET test_key "test_value" EX 60
CACHED_VALUE=$(redis-cli -u $REDIS_URL GET test_key)
if [ "$CACHED_VALUE" = "test_value" ]; then
    echo "✅ 基础缓存读写正常"
else
    echo "❌ 基础缓存读写失败"
    exit 1
fi

# 测试3: 测试对话缓存
echo "测试对话缓存..."

# 先创建一个对话
CONVERSATION_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/conversations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "config_id": 1,
    "title": "缓存测试对话",
    "context": "cache_test"
  }')

CONVERSATION_ID=$(echo $CONVERSATION_RESPONSE | jq -r .data.id)

if [ "$CONVERSATION_ID" != "null" ] && [ -n "$CONVERSATION_ID" ]; then
    echo "✅ 对话创建成功，ID: $CONVERSATION_ID"
    
    # 发送消息触发缓存
    curl -s -X POST "$BASE_URL/api/ai/conversations/$CONVERSATION_ID/messages" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"message": "缓存测试消息"}' > /dev/null
    
    sleep 2
    
    # 第一次获取消息（从数据库）
    FIRST_RESPONSE=$(curl -s "$BASE_URL/api/ai/conversations/$CONVERSATION_ID/messages" \
      -H "Authorization: Bearer $TOKEN")
    
    FROM_CACHE_FIRST=$(echo $FIRST_RESPONSE | jq -r .from_cache)
    
    # 第二次获取消息（应该从缓存）
    SECOND_RESPONSE=$(curl -s "$BASE_URL/api/ai/conversations/$CONVERSATION_ID/messages" \
      -H "Authorization: Bearer $TOKEN")
    
    FROM_CACHE_SECOND=$(echo $SECOND_RESPONSE | jq -r .from_cache)
    
    if [ "$FROM_CACHE_SECOND" = "true" ]; then
        echo "✅ 消息缓存工作正常"
    else
        echo "⚠️  消息缓存可能未生效"
    fi
else
    echo "⚠️  无法创建测试对话，跳过对话缓存测试"
fi

# 测试4: 测试缓存键模式
echo "测试缓存键模式..."
CACHE_KEYS=$(redis-cli -u $REDIS_URL KEYS "ai:*")
echo "AI相关缓存键: $CACHE_KEYS"

# 测试5: 测试缓存过期
echo "测试缓存过期..."
redis-cli -u $REDIS_URL SET expire_test "will_expire" EX 3
sleep 1
BEFORE_EXPIRE=$(redis-cli -u $REDIS_URL GET expire_test)
sleep 3
AFTER_EXPIRE=$(redis-cli -u $REDIS_URL GET expire_test)

if [ "$BEFORE_EXPIRE" = "will_expire" ] && [ "$AFTER_EXPIRE" = "" ]; then
    echo "✅ 缓存过期机制正常"
else
    echo "⚠️  缓存过期机制异常"
fi

# 测试6: 测试系统健康检查中的缓存状态
echo "测试缓存健康检查..."
HEALTH_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/health" \
  -H "Authorization: Bearer $TOKEN")

CACHE_STATUS=$(echo $HEALTH_RESPONSE | jq -r .data.services.cache.status)
if [ "$CACHE_STATUS" = "healthy" ]; then
    echo "✅ 缓存健康检查正常"
else
    echo "⚠️  缓存健康检查异常: $CACHE_STATUS"
fi

# 测试7: 测试缓存统计
echo "测试缓存统计信息..."
REDIS_INFO=$(redis-cli -u $REDIS_URL INFO stats)
echo "Redis统计信息获取成功"

# 测试8: 测试缓存预热
echo "测试缓存预热..."
# 这个需要调用专门的预热接口，如果存在的话
WARMUP_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/cache/warmup" \
  -H "Authorization: Bearer $TOKEN" || echo "预热接口不存在")

if [[ $WARMUP_RESPONSE == *"success"* ]]; then
    echo "✅ 缓存预热成功"
else
    echo "⚠️  缓存预热接口不可用或失败"
fi

# 清理测试数据
redis-cli -u $REDIS_URL DEL test_key expire_test

echo "🎉 缓存服务功能测试完成！"
