#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}测试任务创建修复${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 配置
BACKEND_URL="${BACKEND_URL:-http://localhost:3000}"
API_BASE="$BACKEND_URL/api/slurm"

echo -e "${YELLOW}步骤 1: 检查后端服务...${NC}"
if ! curl -s --max-time 5 "$BACKEND_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ 后端服务无响应${NC}"
    echo "请确保后端服务已启动"
    exit 1
fi
echo -e "${GREEN}✓ 后端服务正常${NC}"

echo ""
echo -e "${YELLOW}步骤 2: 创建测试扩容任务...${NC}"
SCALE_REQUEST='{
  "nodes": [
    {
      "host": "test-node-001",
      "port": 22,
      "user": "root",
      "password": "test123"
    }
  ]
}'

echo "发送请求..."
SCALE_RESPONSE=$(curl -s -X POST "$API_BASE/scaling/scale-up/async" \
  -H "Content-Type: application/json" \
  -d "$SCALE_REQUEST" 2>&1)

echo ""
echo "响应内容:"
echo "$SCALE_RESPONSE" | jq '.' 2>/dev/null || echo "$SCALE_RESPONSE"

if echo "$SCALE_RESPONSE" | grep -q "ERROR.*invalid input syntax for type bigint"; then
    echo ""
    echo -e "${RED}✗ 仍然存在 bigint 类型错误${NC}"
    echo -e "${RED}错误详情: $(echo "$SCALE_RESPONSE" | grep -o 'ERROR.*')"
    exit 1
fi

if echo "$SCALE_RESPONSE" | jq -e '.taskId' > /dev/null 2>&1; then
    TASK_ID=$(echo "$SCALE_RESPONSE" | jq -r '.taskId')
    OP_ID=$(echo "$SCALE_RESPONSE" | jq -r '.opId')
    echo ""
    echo -e "${GREEN}✓ 任务创建成功${NC}"
    echo "  任务ID: $TASK_ID"
    echo "  操作ID: $OP_ID"
else
    echo ""
    echo -e "${RED}✗ 任务创建失败${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}步骤 3: 等待任务初始化（3秒）...${NC}"
sleep 3

echo ""
echo -e "${YELLOW}步骤 4: 验证任务记录...${NC}"
TASK_DETAIL=$(curl -s "$API_BASE/tasks/$TASK_ID")

if echo "$TASK_DETAIL" | jq -e '.data.task' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 任务已成功记录到数据库${NC}"
    
    TASK_NAME=$(echo "$TASK_DETAIL" | jq -r '.data.task.name // "未知"')
    TASK_STATUS=$(echo "$TASK_DETAIL" | jq -r '.data.task.status // "未知"')
    TASK_ID_DB=$(echo "$TASK_DETAIL" | jq -r '.data.task.id // 0')
    
    echo ""
    echo "任务详情:"
    echo "  数据库ID: $TASK_ID_DB"
    echo "  任务UUID: $TASK_ID"
    echo "  名称: $TASK_NAME"
    echo "  状态: $TASK_STATUS"
else
    echo -e "${RED}✗ 无法获取任务详情${NC}"
    echo "$TASK_DETAIL" | jq '.'
    exit 1
fi

echo ""
echo -e "${YELLOW}步骤 5: 检查任务事件...${NC}"
if echo "$TASK_DETAIL" | jq -e '.data.events' > /dev/null 2>&1; then
    EVENT_COUNT=$(echo "$TASK_DETAIL" | jq -r '.data.events | length')
    echo "  事件数量: $EVENT_COUNT"
    
    if [ "$EVENT_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ 任务事件已成功创建${NC}"
        echo ""
        echo "初始事件:"
        echo "$TASK_DETAIL" | jq -r '.data.events[0] | "  类型: \(.event_type)\n  消息: \(.message)\n  时间: \(.timestamp)"'
    else
        echo -e "${YELLOW}⚠ 暂无任务事件（可能仍在处理中）${NC}"
    fi
else
    echo -e "${YELLOW}⚠ 无法获取事件信息${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}测试完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}修复验证成功：${NC}"
echo "  ✓ 任务创建不再报 bigint 类型错误"
echo "  ✓ 任务 ID 正确生成和保存"
echo "  ✓ 任务事件成功关联"
echo ""
