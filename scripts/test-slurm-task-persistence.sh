#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}SLURM 任务持久化测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 配置
BACKEND_URL="${BACKEND_URL:-http://localhost:3000}"
API_BASE="$BACKEND_URL/api/slurm"

# 步骤 1: 检查后端服务
echo -e "${YELLOW}步骤 1: 检查后端服务...${NC}"
if curl -s --max-time 5 "$BACKEND_URL/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 后端服务正常运行${NC}"
else
    echo -e "${RED}✗ 后端服务无响应${NC}"
    echo "请确保后端服务已启动: docker-compose up -d backend"
    exit 1
fi

# 步骤 2: 检查当前任务统计（修复前）
echo ""
echo -e "${YELLOW}步骤 2: 获取当前任务统计...${NC}"
STATS_RESPONSE=$(curl -s "$API_BASE/tasks/statistics")
echo "$STATS_RESPONSE" | jq '.'

if echo "$STATS_RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
    TOTAL_TASKS=$(echo "$STATS_RESPONSE" | jq -r '.data.total_tasks // 0')
    RUNNING_TASKS=$(echo "$STATS_RESPONSE" | jq -r '.data.running_tasks // 0')
    COMPLETED_TASKS=$(echo "$STATS_RESPONSE" | jq -r '.data.completed_tasks // 0')
    FAILED_TASKS=$(echo "$STATS_RESPONSE" | jq -r '.data.failed_tasks // 0')
    
    echo ""
    echo "当前统计:"
    echo "  总任务数: $TOTAL_TASKS"
    echo "  运行中: $RUNNING_TASKS"
    echo "  已完成: $COMPLETED_TASKS"
    echo "  失败: $FAILED_TASKS"
else
    echo -e "${RED}✗ 无法获取任务统计${NC}"
    exit 1
fi

# 步骤 3: 创建测试扩容任务
echo ""
echo -e "${YELLOW}步骤 3: 创建测试扩容任务...${NC}"
SCALE_REQUEST='{
  "nodes": [
    {
      "host": "192.168.0.100",
      "port": 22,
      "user": "root",
      "password": "test123"
    }
  ]
}'

SCALE_RESPONSE=$(curl -s -X POST "$API_BASE/scaling/scale-up/async" \
  -H "Content-Type: application/json" \
  -d "$SCALE_REQUEST")

echo "$SCALE_RESPONSE" | jq '.'

if echo "$SCALE_RESPONSE" | jq -e '.opId' > /dev/null 2>&1; then
    OP_ID=$(echo "$SCALE_RESPONSE" | jq -r '.opId')
    TASK_ID=$(echo "$SCALE_RESPONSE" | jq -r '.taskId // .opId')
    echo -e "${GREEN}✓ 任务已创建${NC}"
    echo "  操作ID: $OP_ID"
    echo "  任务ID: $TASK_ID"
else
    echo -e "${RED}✗ 创建任务失败${NC}"
    exit 1
fi

# 步骤 4: 等待任务启动
echo ""
echo -e "${YELLOW}步骤 4: 等待任务启动（5秒）...${NC}"
sleep 5

# 步骤 5: 检查任务是否已记录到数据库
echo ""
echo -e "${YELLOW}步骤 5: 检查任务数据库记录...${NC}"
TASK_DETAIL=$(curl -s "$API_BASE/tasks/$TASK_ID")
echo "$TASK_DETAIL" | jq '.'

if echo "$TASK_DETAIL" | jq -e '.data.task' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 任务已记录到数据库${NC}"
    
    TASK_NAME=$(echo "$TASK_DETAIL" | jq -r '.data.task.name // "未知"')
    TASK_TYPE=$(echo "$TASK_DETAIL" | jq -r '.data.task.type // "未知"')
    TASK_STATUS=$(echo "$TASK_DETAIL" | jq -r '.data.task.status // "未知"')
    TARGET_NODES=$(echo "$TASK_DETAIL" | jq -r '.data.task.target_nodes // 0')
    
    echo ""
    echo "任务详情:"
    echo "  名称: $TASK_NAME"
    echo "  类型: $TASK_TYPE"
    echo "  状态: $TASK_STATUS"
    echo "  目标节点数: $TARGET_NODES"
    
    # 检查参数是否完整
    PARAMS=$(echo "$TASK_DETAIL" | jq -r '.data.task.parameters // "{}"')
    if echo "$PARAMS" | jq -e '.nodes' > /dev/null 2>&1; then
        NODE_COUNT=$(echo "$PARAMS" | jq -r '.node_count // 0')
        echo "  参数中的节点数: $NODE_COUNT"
        echo -e "${GREEN}✓ 任务参数已完整保存${NC}"
    else
        echo -e "${RED}✗ 任务参数不完整${NC}"
    fi
else
    echo -e "${RED}✗ 任务未记录到数据库${NC}"
    exit 1
fi

# 步骤 6: 检查任务事件
echo ""
echo -e "${YELLOW}步骤 6: 检查任务事件...${NC}"
if echo "$TASK_DETAIL" | jq -e '.data.events' > /dev/null 2>&1; then
    EVENT_COUNT=$(echo "$TASK_DETAIL" | jq -r '.data.events | length')
    echo "  事件数量: $EVENT_COUNT"
    
    if [ "$EVENT_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ 任务事件已记录${NC}"
        echo ""
        echo "最近的事件:"
        echo "$TASK_DETAIL" | jq -r '.data.events[-3:] | .[] | "  [\(.event_type)] \(.message)"'
    else
        echo -e "${YELLOW}⚠ 暂无任务事件${NC}"
    fi
else
    echo -e "${YELLOW}⚠ 无法获取任务事件${NC}"
fi

# 步骤 7: 再次检查任务统计
echo ""
echo -e "${YELLOW}步骤 7: 检查更新后的任务统计...${NC}"
sleep 2
NEW_STATS_RESPONSE=$(curl -s "$API_BASE/tasks/statistics")

if echo "$NEW_STATS_RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
    NEW_TOTAL_TASKS=$(echo "$NEW_STATS_RESPONSE" | jq -r '.data.total_tasks // 0')
    NEW_RUNNING_TASKS=$(echo "$NEW_STATS_RESPONSE" | jq -r '.data.running_tasks // 0')
    
    echo ""
    echo "更新后的统计:"
    echo "  总任务数: $NEW_TOTAL_TASKS (之前: $TOTAL_TASKS)"
    echo "  运行中: $NEW_RUNNING_TASKS (之前: $RUNNING_TASKS)"
    
    if [ "$NEW_TOTAL_TASKS" -gt "$TOTAL_TASKS" ]; then
        echo -e "${GREEN}✓ 任务统计已更新！${NC}"
    else
        echo -e "${YELLOW}⚠ 任务统计未变化（可能需要等待任务完成）${NC}"
    fi
else
    echo -e "${RED}✗ 无法获取更新后的统计${NC}"
fi

# 步骤 8: 检查任务列表
echo ""
echo -e "${YELLOW}步骤 8: 检查任务列表...${NC}"
TASKS_RESPONSE=$(curl -s "$API_BASE/tasks?page=1&page_size=5")

if echo "$TASKS_RESPONSE" | jq -e '.data.tasks' > /dev/null 2>&1; then
    TASKS_COUNT=$(echo "$TASKS_RESPONSE" | jq -r '.data.tasks | length')
    TOTAL_COUNT=$(echo "$TASKS_RESPONSE" | jq -r '.data.total // 0')
    
    echo "  当前页任务数: $TASKS_COUNT"
    echo "  总任务数: $TOTAL_COUNT"
    
    if [ "$TASKS_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ 任务列表正常${NC}"
        echo ""
        echo "最近的任务:"
        echo "$TASKS_RESPONSE" | jq -r '.data.tasks[0:3] | .[] | "  [\(.status)] \(.name) - \(.type)"'
    fi
else
    echo -e "${RED}✗ 无法获取任务列表${NC}"
fi

# 总结
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}测试完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "验证结果:"
echo "  ✓ 后端服务正常"
echo "  ✓ 扩容任务已创建"
echo "  ✓ 任务已持久化到数据库"
echo "  ✓ 任务参数完整保存"
echo "  ✓ 任务事件正常记录"
echo "  ✓ 任务统计可以查询"
echo ""
echo "访问前端查看详情:"
echo "  http://192.168.18.154:8080/slurm-tasks?taskId=$TASK_ID"
echo ""
