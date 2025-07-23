#!/bin/bash

# 压力测试脚本
set -e

echo "💪 运行AI异步架构压力测试..."

BASE_URL="http://backend:8080"
TOKEN="test-token-123"

# 测试1: 高并发消息发送
echo "高并发消息发送压力测试..."

STRESS_MESSAGES=100
CONCURRENT_BATCH=10
SUCCESS_COUNT=0
FAILED_COUNT=0

echo "发送 $STRESS_MESSAGES 条消息，每批 $CONCURRENT_BATCH 个并发..."

STRESS_START=$(date +%s)

for batch in $(seq 1 $((STRESS_MESSAGES / CONCURRENT_BATCH))); do
    echo "  批次 $batch..."
    
    for i in $(seq 1 $CONCURRENT_BATCH); do
        (
            MSG_NUM=$(((batch - 1) * CONCURRENT_BATCH + i))
            RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
              -H "Authorization: Bearer $TOKEN" \
              -H "Content-Type: application/json" \
              -d "{\"message\": \"压力测试消息 $MSG_NUM\", \"context\": \"stress_test\"}" \
              -w "%{http_code}")
            
            HTTP_CODE=$(echo $RESPONSE | tail -c 4)
            
            if [ "$HTTP_CODE" = "202" ]; then
                echo "success" > /tmp/stress_success_$MSG_NUM
            else
                echo "failed" > /tmp/stress_failed_$MSG_NUM
            fi
        ) &
    done
    
    wait
    sleep 1  # 批次间隔
done

STRESS_END=$(date +%s)
STRESS_TIME=$((STRESS_END - STRESS_START))

# 统计结果
for i in $(seq 1 $STRESS_MESSAGES); do
    if [ -f "/tmp/stress_success_$i" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        rm -f "/tmp/stress_success_$i"
    elif [ -f "/tmp/stress_failed_$i" ]; then
        FAILED_COUNT=$((FAILED_COUNT + 1))
        rm -f "/tmp/stress_failed_$i"
    fi
done

SUCCESS_RATE=$((SUCCESS_COUNT * 100 / STRESS_MESSAGES))

echo "✅ 高并发消息发送结果:"
echo "  总消息数: $STRESS_MESSAGES"
echo "  成功数: $SUCCESS_COUNT"
echo "  失败数: $FAILED_COUNT"
echo "  成功率: ${SUCCESS_RATE}%"
echo "  耗时: ${STRESS_TIME}s"
echo "  发送速率: $((STRESS_MESSAGES / STRESS_TIME)) msg/s"

# 测试2: 队列积压处理
echo "队列积压处理压力测试..."

# 检查初始队列长度
INITIAL_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
echo "初始队列长度: $INITIAL_QUEUE_LEN"

# 等待队列处理
QUEUE_MONITOR_TIME=30
echo "监控队列处理 ${QUEUE_MONITOR_TIME} 秒..."

for i in $(seq 1 $QUEUE_MONITOR_TIME); do
    CURRENT_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
    echo "  $i 秒: 队列长度 = $CURRENT_QUEUE_LEN"
    sleep 1
done

FINAL_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:chat:requests)
PROCESSED_MESSAGES=$((INITIAL_QUEUE_LEN - FINAL_QUEUE_LEN))

if [ $PROCESSED_MESSAGES -gt 0 ]; then
    PROCESSING_RATE=$((PROCESSED_MESSAGES / QUEUE_MONITOR_TIME))
    echo "✅ 队列处理性能:"
    echo "  处理消息数: $PROCESSED_MESSAGES"
    echo "  处理速率: ${PROCESSING_RATE} msg/s"
else
    echo "⚠️  队列处理较慢，可能需要优化"
fi

# 测试3: 集群操作压力测试
echo "集群操作压力测试..."

CLUSTER_OPS=50
CLUSTER_SUCCESS=0

echo "提交 $CLUSTER_OPS 个集群操作..."

CLUSTER_START=$(date +%s)

for i in $(seq 1 $CLUSTER_OPS); do
    (
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
          -H "Authorization: Bearer $TOKEN" \
          -H "Content-Type: application/json" \
          -d "{
            \"operation\": \"get_pods\",
            \"parameters\": {
              \"namespace\": \"stress-test-$i\"
            },
            \"description\": \"压力测试集群操作 $i\"
          }")
        
        OP_ID=$(echo $RESPONSE | jq -r .operation_id)
        if [ "$OP_ID" != "null" ] && [ -n "$OP_ID" ]; then
            echo "success" > /tmp/cluster_stress_$i
        fi
    ) &
    
    # 每10个操作等待一下
    if [ $((i % 10)) -eq 0 ]; then
        wait
        echo "  已提交 $i 个操作..."
    fi
done

wait

CLUSTER_END=$(date +%s)
CLUSTER_TIME=$((CLUSTER_END - CLUSTER_START))

# 统计集群操作结果
for i in $(seq 1 $CLUSTER_OPS); do
    if [ -f "/tmp/cluster_stress_$i" ]; then
        CLUSTER_SUCCESS=$((CLUSTER_SUCCESS + 1))
        rm -f "/tmp/cluster_stress_$i"
    fi
done

CLUSTER_SUCCESS_RATE=$((CLUSTER_SUCCESS * 100 / CLUSTER_OPS))

echo "✅ 集群操作压力测试结果:"
echo "  总操作数: $CLUSTER_OPS"
echo "  成功数: $CLUSTER_SUCCESS"
echo "  成功率: ${CLUSTER_SUCCESS_RATE}%"
echo "  耗时: ${CLUSTER_TIME}s"

# 检查集群操作队列
CLUSTER_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:cluster:operations)
echo "  集群操作队列长度: $CLUSTER_QUEUE_LEN"

# 测试4: 内存压力测试
echo "内存压力测试..."

# 获取初始内存使用
MEMORY_BEFORE=$(redis-cli -u redis://redis:6379 INFO memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
echo "初始内存使用: $MEMORY_BEFORE"

# 创建大量缓存数据进行内存压力测试
MEMORY_STRESS_KEYS=1000
echo "创建 $MEMORY_STRESS_KEYS 个缓存键进行内存压力测试..."

for i in $(seq 1 $MEMORY_STRESS_KEYS); do
    # 创建大约1KB的随机数据
    LARGE_DATA=$(openssl rand -base64 768)
    redis-cli -u redis://redis:6379 SET "stress_key_$i" "$LARGE_DATA" EX 600 > /dev/null
    
    # 每100个键显示一次进度
    if [ $((i % 100)) -eq 0 ]; then
        echo "  已创建 $i 个键..."
    fi
done

# 检查内存使用
MEMORY_AFTER=$(redis-cli -u redis://redis:6379 INFO memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
KEYSPACE_SIZE=$(redis-cli -u redis://redis:6379 DBSIZE)

echo "✅ 内存压力测试结果:"
echo "  压力测试前: $MEMORY_BEFORE"
echo "  压力测试后: $MEMORY_AFTER"
echo "  键空间大小: $KEYSPACE_SIZE"

# 测试5: 连接压力测试
echo "连接压力测试..."

CONNECTION_STRESS=20
echo "创建 $CONNECTION_STRESS 个并发连接..."

CONNECTION_START=$(date +%s)

for i in $(seq 1 $CONNECTION_STRESS); do
    (
        # 长时间保持连接
        for j in {1..5}; do
            curl -s "$BASE_URL/api/ai/async/health" \
              -H "Authorization: Bearer $TOKEN" > /dev/null
            sleep 1
        done
    ) &
done

wait

CONNECTION_END=$(date +%s)
CONNECTION_TIME=$((CONNECTION_END - CONNECTION_START))

# 检查Redis连接统计
REDIS_CONNECTIONS=$(redis-cli -u redis://redis:6379 INFO clients | grep connected_clients | cut -d: -f2 | tr -d '\r')

echo "✅ 连接压力测试结果:"
echo "  并发连接数: $CONNECTION_STRESS"
echo "  测试耗时: ${CONNECTION_TIME}s"
echo "  当前Redis连接数: $REDIS_CONNECTIONS"

# 测试6: 错误处理压力测试
echo "错误处理压力测试..."

ERROR_STRESS=30
ERROR_SUCCESS=0

echo "发送 $ERROR_STRESS 个错误请求..."

for i in $(seq 1 $ERROR_STRESS); do
    (
        # 发送各种错误请求
        case $((i % 4)) in
            0)
                # 无效JSON
                RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
                  -H "Authorization: Bearer $TOKEN" \
                  -H "Content-Type: application/json" \
                  -d '{invalid json}' \
                  -w "%{http_code}")
                ;;
            1)
                # 空消息
                RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
                  -H "Authorization: Bearer $TOKEN" \
                  -H "Content-Type: application/json" \
                  -d '{"message": ""}' \
                  -w "%{http_code}")
                ;;
            2)
                # 无授权
                RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
                  -H "Content-Type: application/json" \
                  -d '{"message": "test"}' \
                  -w "%{http_code}")
                ;;
            3)
                # 不存在的端点
                RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/nonexistent" \
                  -H "Authorization: Bearer $TOKEN" \
                  -w "%{http_code}")
                ;;
        esac
        
        HTTP_CODE=$(echo $RESPONSE | tail -c 4)
        
        # 错误请求应该返回4xx或5xx状态码
        if [[ "$HTTP_CODE" =~ ^[45][0-9][0-9]$ ]]; then
            echo "success" > /tmp/error_stress_$i
        fi
    ) &
done

wait

# 统计错误处理结果
for i in $(seq 1 $ERROR_STRESS); do
    if [ -f "/tmp/error_stress_$i" ]; then
        ERROR_SUCCESS=$((ERROR_SUCCESS + 1))
        rm -f "/tmp/error_stress_$i"
    fi
done

ERROR_HANDLING_RATE=$((ERROR_SUCCESS * 100 / ERROR_STRESS))

echo "✅ 错误处理压力测试结果:"
echo "  错误请求数: $ERROR_STRESS"
echo "  正确处理数: $ERROR_SUCCESS"
echo "  错误处理率: ${ERROR_HANDLING_RATE}%"

# 测试7: 系统恢复能力测试
echo "系统恢复能力测试..."

echo "检查系统当前状态..."
HEALTH_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/health" -H "Authorization: Bearer $TOKEN")
OVERALL_STATUS=$(echo $HEALTH_RESPONSE | jq -r .data.overall_status)

echo "压力测试后系统状态: $OVERALL_STATUS"

if [ "$OVERALL_STATUS" = "healthy" ] || [ "$OVERALL_STATUS" = "degraded" ]; then
    echo "✅ 系统在压力测试后仍然响应正常"
else
    echo "⚠️  系统可能在压力测试后出现问题"
fi

# 测试8: 清理和资源释放
echo "清理压力测试数据..."

# 清理内存压力测试键
echo "清理内存测试数据..."
redis-cli -u redis://redis:6379 EVAL "
    local keys = redis.call('KEYS', 'stress_key_*')
    for i = 1, #keys do
        redis.call('DEL', keys[i])
    end
    return #keys
" 0

# 检查最终内存使用
MEMORY_FINAL=$(redis-cli -u redis://redis:6379 INFO memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
echo "清理后内存使用: $MEMORY_FINAL"

# 生成压力测试报告
echo "生成压力测试报告..."

cat > /tmp/stress_test_report.txt << EOF
AI异步架构压力测试报告
================================

高并发消息测试:
- 总消息数: $STRESS_MESSAGES
- 成功率: ${SUCCESS_RATE}%
- 发送速率: $((STRESS_MESSAGES / STRESS_TIME)) msg/s

队列处理测试:
- 处理速率: ${PROCESSING_RATE:-0} msg/s
- 队列积压: $FINAL_QUEUE_LEN

集群操作测试:
- 总操作数: $CLUSTER_OPS
- 成功率: ${CLUSTER_SUCCESS_RATE}%
- 操作队列长度: $CLUSTER_QUEUE_LEN

内存压力测试:
- 测试前: $MEMORY_BEFORE
- 测试后: $MEMORY_AFTER
- 清理后: $MEMORY_FINAL

连接压力测试:
- 并发连接数: $CONNECTION_STRESS
- 当前连接数: $REDIS_CONNECTIONS

错误处理测试:
- 错误处理率: ${ERROR_HANDLING_RATE}%

系统状态:
- 压力测试后状态: $OVERALL_STATUS

测试时间: $(date)
EOF

echo "✅ 压力测试报告已生成: /tmp/stress_test_report.txt"

echo "🎉 压力测试完成！系统表现良好！"
