#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}确保系统用户存在${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 数据库连接信息
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ai_infra}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"

# 检查 PostgreSQL 是否可访问
echo -e "${YELLOW}步骤 1: 检查数据库连接...${NC}"
if command -v psql &> /dev/null; then
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 数据库连接成功${NC}"
    else
        echo -e "${RED}✗ 数据库连接失败${NC}"
        echo "请检查数据库配置或使用 docker exec 在容器内执行"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ 本地未安装 psql，将尝试使用 docker exec${NC}"
    DOCKER_EXEC="docker exec -i ai-infra-postgres psql -U $DB_USER -d $DB_NAME"
fi

# 检查系统用户是否存在
echo ""
echo -e "${YELLOW}步骤 2: 检查系统用户...${NC}"

if [ -z "$DOCKER_EXEC" ]; then
    USER_EXISTS=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM users WHERE id = 1;")
else
    USER_EXISTS=$(echo "SELECT COUNT(*) FROM users WHERE id = 1;" | $DOCKER_EXEC -t)
fi

USER_EXISTS=$(echo $USER_EXISTS | tr -d '[:space:]')

if [ "$USER_EXISTS" = "1" ]; then
    echo -e "${GREEN}✓ 系统用户 (ID=1) 已存在${NC}"
    
    # 显示用户信息
    if [ -z "$DOCKER_EXEC" ]; then
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT id, username, email, created_at FROM users WHERE id = 1;"
    else
        echo "SELECT id, username, email, created_at FROM users WHERE id = 1;" | $DOCKER_EXEC
    fi
else
    echo -e "${YELLOW}⚠ 系统用户不存在，正在创建...${NC}"
    
    # 创建系统用户
    SQL="INSERT INTO users (id, username, email, password, created_at, updated_at) 
         VALUES (1, 'system', 'system@ai-infra.local', 
                 '\$2a\$10\$dummyhashedpasswordforSystemUser1234567890', 
                 NOW(), NOW())
         ON CONFLICT (id) DO NOTHING;"
    
    if [ -z "$DOCKER_EXEC" ]; then
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "$SQL"
    else
        echo "$SQL" | $DOCKER_EXEC
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 系统用户创建成功${NC}"
        echo "  用户名: system"
        echo "  邮箱: system@ai-infra.local"
        echo "  ID: 1"
    else
        echo -e "${RED}✗ 系统用户创建失败${NC}"
        exit 1
    fi
fi

# 确保用户表的序列从 2 开始
echo ""
echo -e "${YELLOW}步骤 3: 调整用户ID序列...${NC}"

SQL="SELECT setval('users_id_seq', GREATEST(2, (SELECT MAX(id) FROM users) + 1), false);"

if [ -z "$DOCKER_EXEC" ]; then
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "$SQL"
else
    echo "$SQL" | $DOCKER_EXEC
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 序列调整成功${NC}"
    echo "  新用户将从 ID=2 开始"
else
    echo -e "${YELLOW}⚠ 序列调整失败（可能序列不存在）${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "系统用户信息："
echo "  ID: 1"
echo "  用户名: system"
echo "  用途: 未认证用户执行的任务将关联到此用户"
echo ""
