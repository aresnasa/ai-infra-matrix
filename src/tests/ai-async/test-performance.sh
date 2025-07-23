#!/bin/bash

# 性能测试脚本
set -e

echo "🚄 运行AI异步架构性能测试..."

BASE_URL="http://backend:8080"
TOKEN="test-token-123"

# 测试1: 响应时间测试
echo "测试API响应时间..."

RESPONSE_TIMES=()
ENDPOINTS=(
    "/api/ai/async/health"
    "/api/ai/async/quick-chat"
    "/api/ai/async/cluster-operations"
)

for ENDPOINT in "${ENDPOINTS[@]}"; do
    echo "  测试端点: $ENDPOINT"
    
    TOTAL_TIME=0
    REQUESTS=10
    
    for i in $(seq 1 $REQUESTS); do
        START_TIME=$(date +%s%N)
        
        if [ "$ENDPOINT" = "/api/ai/async/health" ]; then
            curl -s "$BASE_URL$ENDPOINT" \
              -H "Authorization: Bearer $TOKEN" > /dev/null
        elif [ "$ENDPOINT" = "/api/ai/async/quick-chat" ]; then
            curl -s -X POST "$BASE_URL$ENDPOINT" \
              -H "Authorization: Bearer $TOKEN" \
              -H "Content-Type: application/json" \
              -d "{\"message\": \"性能测试消息 $i\", \"context\": \"perf_test\"}" > /dev/null
        else
            curl -s -X POST "$BASE_URL$ENDPOINT" \
              -H "Authorization: Bearer $TOKEN" \
              -H "Content-Type: application/json" \
              -d "{\"operation\": \"get_pods\", \"parameters\": {\"namespace\": \"default\"}, \"description\": \"性能测试 $i\"}" > /dev/null
        fi
        
        END_TIME=$(date +%s%N)
        RESPONSE_TIME=$(((END_TIME - START_TIME) / 1000000))  # 转换为毫秒
        TOTAL_TIME=$((TOTAL_TIME + RESPONSE_TIME))
    done
    
    AVG_TIME=$((TOTAL_TIME / REQUESTS))
    echo "    平均响应时间: ${AVG_TIME}ms"
    RESPONSE_TIMES+=("$AVG_TIME")
done

# 测试2: 吞吐量测试
echo "测试系统吞吐量..."

THROUGHPUT_START=$(date +%s)
CONCURRENT_REQUESTS=50
SUCCESS_COUNT=0

# 并发发送请求
for i in $(seq 1 $CONCURRENT_REQUESTS); do
    (
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
          -H "Authorization: Bearer $TOKEN" \
          -H "Content-Type: application/json" \
          -d "{\"message\": \"吞吐量测试 $i\", \"context\": \"throughput_test\"}")
        
        if echo $RESPONSE | jq -e .message_id > /dev/null 2>&1; then
            echo "success" > /tmp/throughput_$i
        fi
    ) &
done

wait

# 统计成功请求
for i in $(seq 1 $CONCURRENT_REQUESTS); do
    if [ -f "/tmp/throughput_$i" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        rm -f "/tmp/throughput_$i"
    fi
done

THROUGHPUT_END=$(date +%s)
THROUGHPUT_TIME=$((THROUGHPUT_END - THROUGHPUT_START))
THROUGHPUT_RATE=$((SUCCESS_COUNT / THROUGHPUT_TIME))

echo "✅ 吞吐量测试结果:"
echo "  总请求数: $CONCURRENT_REQUESTS"
echo "  成功请求数: $SUCCESS_COUNT"
echo "  耗时: ${THROUGHPUT_TIME}s"
echo "  吞吐量: ${THROUGHPUT_RATE} req/s"

# 测试3: 内存使用监控
echo "监控系统内存使用..."

# 监控Redis内存使用
REDIS_MEMORY_BEFORE=$(redis-cli -u redis://redis:6379 INFO memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
echo "Redis内存使用（测试前）: $REDIS_MEMORY_BEFORE"

# 创建大量缓存数据
echo "创建缓存压力..."
for i in {1..100}; do
    redis-cli -u redis://redis:6379 SET "perf_test_key_$i" "$(openssl rand -base64 1024)" EX 300 > /dev/null
done

REDIS_MEMORY_AFTER=$(redis-cli -u redis://redis:6379 INFO memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
echo "Redis内存使用（测试后）: $REDIS_MEMORY_AFTER"

# 测试4: 队列处理性能
echo "测试队列处理性能..."

# 清空现有队列
redis-cli -u redis://redis:6379 DEL ai:chat:requests > /dev/null
redis-cli -u redis://redis:6379 DEL ai:cluster:operations > /dev/null

QUEUE_START=$(date +%s)
QUEUE_MESSAGES=30

# 批量发送消息到队列
echo "发送 $QUEUE_MESSAGES 条消息到队列..."
for i in $(seq 1 $QUEUE_MESSAGES); do
    curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"message\": \"队列性能测试 $i\", \"context\": \"queue_perf_test\"}" > /dev/null &
    
    # 每5个请求暂停一下，避免过载
    if [ $((i % 5)) -eq 0 ]; then
        wait
    fi
done

wait

QUEUE_END=$(date +%s)
QUEUE_TIME=$((QUEUE_END - QUEUE_START))

# 检查队列长度
QUEUE_LENGTH=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
echo "✅ 队列填充性能:"
echo "  发送消息数: $QUEUE_MESSAGES"
echo "  耗时: ${QUEUE_TIME}s"
echo "  队列长度: $QUEUE_LENGTH"
echo "  发送速率: $((QUEUE_MESSAGES / QUEUE_TIME)) msg/s"

# 监控队列消费速度
echo "监控队列消费速度..."
CONSUME_START=$(date +%s)
INITIAL_LENGTH=$QUEUE_LENGTH

sleep 10  # 等待消费

FINAL_LENGTH=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
CONSUME_END=$(date +%s)
CONSUME_TIME=$((CONSUME_END - CONSUME_START))
CONSUMED_MESSAGES=$((INITIAL_LENGTH - FINAL_LENGTH))

if [ $CONSUMED_MESSAGES -gt 0 ]; then
    CONSUME_RATE=$((CONSUMED_MESSAGES / CONSUME_TIME))
    echo "✅ 队列消费性能:"
    echo "  消费消息数: $CONSUMED_MESSAGES"
    echo "  耗时: ${CONSUME_TIME}s"
    echo "  消费速率: ${CONSUME_RATE} msg/s"
else
    echo "⚠️  队列消费速度较慢或处理器未运行"
fi

# 测试5: 数据库连接池性能
echo "测试数据库连接性能..."

DB_START=$(date +%s)
DB_REQUESTS=20

# 并发数据库操作（通过API）
for i in $(seq 1 $DB_REQUESTS); do
    (
        curl -s "$BASE_URL/api/ai/configs" \
          -H "Authorization: Bearer $TOKEN" > /dev/null
    ) &
done

wait

DB_END=$(date +%s)
DB_TIME=$((DB_END - DB_START))
DB_RATE=$((DB_REQUESTS / DB_TIME))

echo "✅ 数据库连接性能:"
echo "  并发请求数: $DB_REQUESTS"
echo "  耗时: ${DB_TIME}s"
echo "  请求速率: ${DB_RATE} req/s"

# 测试6: 缓存命中率测试
echo "测试缓存命中率..."

# 清空相关缓存
redis-cli -u redis://redis:6379 DEL "messages:*" > /dev/null

CACHE_REQUESTS=20
CACHE_HITS=0

# 首次请求（应该缓存未命中）
for i in $(seq 1 5); do
    RESPONSE=$(curl -s "$BASE_URL/api/ai/conversations/1/messages" \
      -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo '{"from_cache":false}')
    
    FROM_CACHE=$(echo $RESPONSE | jq -r .from_cache 2>/dev/null || echo "false")
    if [ "$FROM_CACHE" = "true" ]; then
        CACHE_HITS=$((CACHE_HITS + 1))
    fi
done

# 重复请求（应该缓存命中）
for i in $(seq 6 $CACHE_REQUESTS); do
    RESPONSE=$(curl -s "$BASE_URL/api/ai/conversations/1/messages" \
      -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo '{"from_cache":false}')
    
    FROM_CACHE=$(echo $RESPONSE | jq -r .from_cache 2>/dev/null || echo "false")
    if [ "$FROM_CACHE" = "true" ]; then
        CACHE_HITS=$((CACHE_HITS + 1))
    fi
done

CACHE_HIT_RATE=$((CACHE_HITS * 100 / CACHE_REQUESTS))
echo "✅ 缓存性能:"
echo "  总请求数: $CACHE_REQUESTS"
echo "  缓存命中数: $CACHE_HITS"
echo "  命中率: ${CACHE_HIT_RATE}%"

# 测试7: 系统资源监控
echo "系统资源使用情况..."

# Redis统计信息
REDIS_STATS=$(redis-cli -u redis://redis:6379 INFO stats)
REDIS_COMMANDS=$(echo "$REDIS_STATS" | grep total_commands_processed | cut -d: -f2 | tr -d '\r')
REDIS_CONNECTIONS=$(echo "$REDIS_STATS" | grep total_connections_received | cut -d: -f2 | tr -d '\r')

echo "✅ Redis统计:"
echo "  总命令数: $REDIS_COMMANDS"
echo "  总连接数: $REDIS_CONNECTIONS"

# 测试8: 性能基准
echo "生成性能基准报告..."

cat > /tmp/performance_report.txt << EOF
AI异步架构性能测试报告
================================

响应时间性能:
- 健康检查: ${RESPONSE_TIMES[0]}ms
- 快速聊天: ${RESPONSE_TIMES[1]}ms  
- 集群操作: ${RESPONSE_TIMES[2]}ms

吞吐量性能:
- 并发请求数: $CONCURRENT_REQUESTS
- 成功率: $((SUCCESS_COUNT * 100 / CONCURRENT_REQUESTS))%
- 吞吐量: ${THROUGHPUT_RATE} req/s

队列性能:
- 发送速率: $((QUEUE_MESSAGES / QUEUE_TIME)) msg/s
- 消费速率: ${CONSUME_RATE:-0} msg/s

数据库性能:
- 连接速率: ${DB_RATE} req/s

缓存性能:
- 命中率: ${CACHE_HIT_RATE}%

资源使用:
- Redis内存: $REDIS_MEMORY_AFTER
- 处理命令数: $REDIS_COMMANDS
- 总连接数: $REDIS_CONNECTIONS

测试时间: $(date)
EOF

echo "✅ 性能报告已生成: /tmp/performance_report.txt"

# 清理测试数据
echo "清理性能测试数据..."
redis-cli -u redis://redis:6379 EVAL "
    for i, key in ipairs(redis.call('KEYS', 'perf_test_key_*')) do
        redis.call('DEL', key)
    end
    return 'OK'
" 0

echo "🎉 性能测试完成！"
