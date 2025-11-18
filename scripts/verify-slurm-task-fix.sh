#!/bin/bash
# ====================================================================
# 验证 SLURM Task 表结构修复脚本
# ====================================================================

set -e

echo "======================================================================"
echo "SLURM Tasks 表结构验证"
echo "======================================================================"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 task_id 字段类型
echo -e "\n${YELLOW}1. 检查 task_id 字段类型...${NC}"
TASK_ID_TYPE=$(docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -t -c "
    SELECT data_type || '(' || character_maximum_length || ')' as type
    FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'slurm_tasks' 
      AND column_name = 'task_id'
" | tr -d '[:space:]')

echo "   当前类型: $TASK_ID_TYPE"

if [[ "$TASK_ID_TYPE" == "charactervarying(36)" ]]; then
    echo -e "   ${GREEN}✓ task_id 类型正确 (VARCHAR(36))${NC}"
else
    echo -e "   ${RED}✗ task_id 类型错误，期望 VARCHAR(36)，实际 $TASK_ID_TYPE${NC}"
    exit 1
fi

# 检查唯一索引
echo -e "\n${YELLOW}2. 检查唯一索引...${NC}"
INDEX_EXISTS=$(docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -t -c "
    SELECT COUNT(*) 
    FROM pg_indexes 
    WHERE schemaname = 'public' 
      AND tablename = 'slurm_tasks' 
      AND indexname = 'idx_slurm_tasks_task_id'
" | tr -d '[:space:]')

if [[ "$INDEX_EXISTS" == "1" ]]; then
    echo -e "   ${GREEN}✓ 唯一索引存在${NC}"
else
    echo -e "   ${RED}✗ 唯一索引不存在${NC}"
    exit 1
fi

# 检查后端服务日志
echo -e "\n${YELLOW}3. 检查后端服务日志...${NC}"
echo "   最近的迁移日志:"
docker-compose logs backend 2>/dev/null | grep -i "migration\|slurm_task" | tail -n 5 || echo "   无相关日志"

# 测试创建任务
echo -e "\n${YELLOW}4. 测试创建 UUID 任务记录...${NC}"
TEST_UUID=$(uuidgen | tr '[:upper:]' '[:lower:]')
echo "   测试 UUID: $TEST_UUID"

docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix <<EOF
BEGIN;
INSERT INTO slurm_tasks (task_id, name, type, status, user_id, created_at, updated_at)
VALUES ('$TEST_UUID', 'Test Task', 'test', 'pending', 1, NOW(), NOW());

SELECT 
    task_id, 
    name, 
    type, 
    status,
    pg_typeof(task_id) as task_id_type
FROM slurm_tasks 
WHERE task_id = '$TEST_UUID';

ROLLBACK;
EOF

if [ $? -eq 0 ]; then
    echo -e "   ${GREEN}✓ UUID 插入测试成功${NC}"
else
    echo -e "   ${RED}✗ UUID 插入测试失败${NC}"
    exit 1
fi

echo -e "\n${GREEN}======================================================================"
echo "✓ 所有验证通过！"
echo "======================================================================${NC}"
