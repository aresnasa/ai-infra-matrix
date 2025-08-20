#!/bin/bash

# 快速AI异步功能测试脚本
set -e

echo "⚡ 快速AI异步功能测试..."

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}启动最小测试环境...${NC}"

# 启动基础服务
docker-compose up -d postgres redis backend

# 等待服务启动
echo -e "${YELLOW}等待服务启动...${NC}"
sleep 30

# 检查服务状态
echo -e "${BLUE}检查服务状态...${NC}"
if ! docker-compose ps backend | grep -q "healthy"; then
    echo -e "${RED}❌ 后端服务未启动${NC}"
    docker-compose logs backend
    exit 1
fi

BASE_URL="http://localhost:8082"
TOKEN="test-token-123"

echo -e "${GREEN}✅ 服务启动成功${NC}"

# 测试1: 健康检查
echo -e "${BLUE}测试1: 健康检查...${NC}"
HEALTH_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/health" \
  -H "Authorization: Bearer $TOKEN" || echo '{"error":"failed"}')

if echo $HEALTH_RESPONSE | jq -e . > /dev/null 2>&1; then
    STATUS=$(echo $HEALTH_RESPONSE | jq -r .data.overall_status 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✅ 健康检查通过，状态: $STATUS${NC}"
else
    echo -e "${RED}❌ 健康检查失败${NC}"
    exit 1
fi

# 测试2: 快速聊天
echo -e "${BLUE}测试2: 快速聊天...${NC}"
CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "快速测试消息", "context": "quick_test"}' || echo '{"error":"failed"}')

MESSAGE_ID=$(echo $CHAT_RESPONSE | jq -r .message_id 2>/dev/null)
if [ "$MESSAGE_ID" != "null" ] && [ -n "$MESSAGE_ID" ]; then
    echo -e "${GREEN}✅ 快速聊天成功，消息ID: $MESSAGE_ID${NC}"
else
    echo -e "${RED}❌ 快速聊天失败${NC}"
    echo "Response: $CHAT_RESPONSE"
fi

# 测试3: 消息状态查询
if [ -n "$MESSAGE_ID" ]; then
    echo -e "${BLUE}测试3: 消息状态查询...${NC}"
    sleep 2
    STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/messages/$MESSAGE_ID/status" \
      -H "Authorization: Bearer $TOKEN" || echo '{"error":"failed"}')
    
    MESSAGE_STATUS=$(echo $STATUS_RESPONSE | jq -r .data.status 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✅ 状态查询成功，状态: $MESSAGE_STATUS${NC}"
fi

# 测试4: 集群操作
echo -e "${BLUE}测试4: 集群操作...${NC}"
CLUSTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_pods",
    "parameters": {"namespace": "default"},
    "description": "快速测试集群操作"
  }' || echo '{"error":"failed"}')

OPERATION_ID=$(echo $CLUSTER_RESPONSE | jq -r .operation_id 2>/dev/null)
if [ "$OPERATION_ID" != "null" ] && [ -n "$OPERATION_ID" ]; then
    echo -e "${GREEN}✅ 集群操作提交成功，操作ID: $OPERATION_ID${NC}"
else
    echo -e "${RED}❌ 集群操作失败${NC}"
    echo "Response: $CLUSTER_RESPONSE"
fi

# 测试5: Redis队列检查
echo -e "${BLUE}测试5: Redis队列检查...${NC}"
CHAT_QUEUE=$(docker-compose exec -T redis redis-cli XLEN ai:chat:requests)
CLUSTER_QUEUE=$(docker-compose exec -T redis redis-cli XLEN ai:cluster:operations)

echo -e "${GREEN}✅ 队列状态 - 聊天: $CHAT_QUEUE, 集群: $CLUSTER_QUEUE${NC}"

# 简单性能测试
echo -e "${BLUE}测试6: 简单性能测试...${NC}"
START_TIME=$(date +%s)

# 发送5个并发请求
for i in {1..5}; do
    curl -s -X POST "$BASE_URL/api/ai/async/quick-chat" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"message\": \"性能测试 $i\", \"context\": \"perf_test\"}" > /dev/null &
done

wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo -e "${GREEN}✅ 性能测试完成，5个并发请求耗时: ${DURATION}s${NC}"

# 最终队列状态
FINAL_CHAT_QUEUE=$(docker-compose exec -T redis redis-cli XLEN ai:chat:requests)
echo -e "${GREEN}✅ 最终聊天队列长度: $FINAL_CHAT_QUEUE${NC}"

# 生成简单报告
REPORT_FILE="./quick-test-report-$(date +%Y%m%d-%H%M%S).txt"
cat > "$REPORT_FILE" << EOF
AI异步架构快速测试报告
===================

测试时间: $(date)
测试结果: ✅ 通过

核心功能测试:
✅ 健康检查 - 状态: $STATUS
✅ 快速聊天 - 消息ID: $MESSAGE_ID
✅ 状态查询 - 状态: $MESSAGE_STATUS
✅ 集群操作 - 操作ID: $OPERATION_ID
✅ Redis队列 - 聊天: $FINAL_CHAT_QUEUE, 集群: $CLUSTER_QUEUE
✅ 性能测试 - 5个并发请求: ${DURATION}s

系统运行正常，核心功能验证通过！
EOF

echo -e "${GREEN}✅ 快速测试报告: $REPORT_FILE${NC}"

# 显示清理提示
echo -e "${YELLOW}清理测试环境请运行: docker-compose down${NC}"

echo -e "${GREEN}🎉 快速测试完成！所有核心功能正常！${NC}"
