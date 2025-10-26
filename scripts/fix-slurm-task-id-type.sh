#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复 SLURM Task ID 字段类型${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 从环境变量读取数据库配置
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-ai-infra-matrix}"

echo -e "${YELLOW}数据库配置:${NC}"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  User: $DB_USER"
echo "  Database: $DB_NAME"
echo ""

# 检查 PostgreSQL 连接
echo -e "${YELLOW}步骤 1: 检查数据库连接...${NC}"
if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
    echo -e "${RED}✗ 无法连接到数据库${NC}"
    echo "请确保数据库服务正在运行并且配置正确"
    exit 1
fi
echo -e "${GREEN}✓ 数据库连接成功${NC}"
echo ""

# 检查当前 task_id 字段类型
echo -e "${YELLOW}步骤 2: 检查 task_id 字段类型...${NC}"
CURRENT_TYPE=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
SELECT data_type 
FROM information_schema.columns 
WHERE table_name = 'slurm_tasks' 
AND column_name = 'task_id';
" 2>/dev/null | tr -d '[:space:]')

if [ -z "$CURRENT_TYPE" ]; then
    echo -e "${YELLOW}⚠ 表 slurm_tasks 不存在或 task_id 字段不存在${NC}"
    echo "这是正常的，backend-init 会创建正确的表结构"
    exit 0
fi

echo "  当前类型: $CURRENT_TYPE"
echo ""

if [ "$CURRENT_TYPE" = "charactervarying" ] || [ "$CURRENT_TYPE" = "varchar" ] || [ "$CURRENT_TYPE" = "text" ]; then
    echo -e "${GREEN}✓ task_id 字段类型已经是字符串类型，无需修复${NC}"
    exit 0
fi

if [ "$CURRENT_TYPE" != "bigint" ]; then
    echo -e "${YELLOW}⚠ task_id 字段类型是 $CURRENT_TYPE，预期是 bigint 或 varchar${NC}"
    echo "请手动检查表结构"
    exit 1
fi

# 备份表
echo -e "${YELLOW}步骤 3: 备份 slurm_tasks 表...${NC}"
BACKUP_TABLE="slurm_tasks_backup_$(date +%Y%m%d_%H%M%S)"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
CREATE TABLE $BACKUP_TABLE AS SELECT * FROM slurm_tasks;
" 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 备份表已创建: $BACKUP_TABLE${NC}"
else
    echo -e "${RED}✗ 备份表创建失败${NC}"
    exit 1
fi
echo ""

# 修复字段类型
echo -e "${YELLOW}步骤 4: 修复 task_id 字段类型...${NC}"
echo "  将 task_id 从 bigint 改为 varchar(36)"
echo ""

PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'EOF'
-- 开始事务
BEGIN;

-- 1. 删除依赖 task_id 的外键约束（如果存在）
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN 
        SELECT conname, conrelid::regclass AS table_name
        FROM pg_constraint
        WHERE confrelid = 'slurm_tasks'::regclass
          AND contype = 'f'
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s', r.table_name, r.conname);
        RAISE NOTICE 'Dropped foreign key: % from %', r.conname, r.table_name;
    END LOOP;
END $$;

-- 2. 删除 task_id 上的索引（如果存在）
DROP INDEX IF EXISTS idx_slurm_tasks_task_id;

-- 3. 修改 task_id 字段类型
-- 如果字段中有数据，需要清空或转换
-- 这里我们假设旧数据无效，直接清空表
TRUNCATE TABLE slurm_tasks CASCADE;
TRUNCATE TABLE slurm_task_events CASCADE;

-- 4. 修改字段类型
ALTER TABLE slurm_tasks 
ALTER COLUMN task_id TYPE VARCHAR(36);

-- 5. 重新创建唯一索引
CREATE UNIQUE INDEX idx_slurm_tasks_task_id ON slurm_tasks(task_id);

-- 6. 修复 slurm_task_events.task_id 外键（如果存在的话，先检查该字段）
-- slurm_task_events.task_id 应该引用 slurm_tasks.id (uint)，而不是 task_id (uuid)
-- 这是正确的，不需要修改

-- 提交事务
COMMIT;

-- 显示结果
\d slurm_tasks
EOF

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ task_id 字段类型修复成功${NC}"
else
    echo -e "${RED}✗ 字段类型修复失败${NC}"
    exit 1
fi
echo ""

# 验证修复
echo -e "${YELLOW}步骤 5: 验证修复结果...${NC}"
NEW_TYPE=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
SELECT data_type 
FROM information_schema.columns 
WHERE table_name = 'slurm_tasks' 
AND column_name = 'task_id';
" | tr -d '[:space:]')

echo "  修复后类型: $NEW_TYPE"
echo ""

if [ "$NEW_TYPE" = "charactervarying" ] || [ "$NEW_TYPE" = "varchar" ]; then
    echo -e "${GREEN}✓ 验证成功！task_id 字段现在是字符串类型${NC}"
else
    echo -e "${RED}✗ 验证失败，字段类型仍然是: $NEW_TYPE${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}修复完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}修复总结:${NC}"
echo "  ✓ task_id 字段类型已从 bigint 改为 varchar(36)"
echo "  ✓ 唯一索引已重新创建"
echo "  ✓ 旧数据已备份到: $BACKUP_TABLE"
echo ""
echo -e "${YELLOW}注意:${NC}"
echo "  - slurm_tasks 和 slurm_task_events 表已被清空"
echo "  - 如需恢复旧数据，请从备份表恢复: $BACKUP_TABLE"
echo ""
echo "现在可以重新启动 backend 服务测试扩容功能："
echo "  docker-compose restart backend"
echo ""
