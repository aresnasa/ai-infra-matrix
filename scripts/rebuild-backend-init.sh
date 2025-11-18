#!/bin/bash
# ====================================================================
# 重新构建并运行 backend-init 修复脚本
# ====================================================================
# 用途: 重新构建 backend-init 镜像并重新初始化数据库
# 说明: 包含最新的 task_id 字段类型修复代码
# ====================================================================

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================================================"
echo "重新构建 backend-init 并修复数据库"
echo -e "======================================================================${NC}"

# 1. 停止并删除旧的 backend-init 容器
echo -e "\n${YELLOW}步骤 1/5: 停止并删除旧的 backend-init 容器...${NC}"
docker-compose stop backend-init 2>/dev/null || true
docker-compose rm -f backend-init 2>/dev/null || true
echo -e "${GREEN}✓ 旧容器已清理${NC}"

# 2. 重新构建 backend-init 镜像
echo -e "\n${YELLOW}步骤 2/5: 重新构建 backend-init 镜像（包含最新修复）...${NC}"
docker-compose build backend-init
echo -e "${GREEN}✓ 镜像构建完成${NC}"

# 3. 检查数据库状态
echo -e "\n${YELLOW}步骤 3/5: 检查数据库连接...${NC}"
if docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PostgreSQL 连接正常${NC}"
else
    echo -e "${RED}✗ PostgreSQL 连接失败，请确保数据库服务正在运行${NC}"
    exit 1
fi

# 4. 检查当前 task_id 字段类型
echo -e "\n${YELLOW}步骤 4/5: 检查 task_id 字段类型...${NC}"
CURRENT_TYPE=$(docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -t -c "
    SELECT data_type || COALESCE('(' || character_maximum_length || ')', '')
    FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'slurm_tasks' 
      AND column_name = 'task_id'
" 2>/dev/null | tr -d '[:space:]' || echo "table_not_exists")

if [[ "$CURRENT_TYPE" == "table_not_exists" ]]; then
    echo -e "${YELLOW}   表尚未创建，将由 backend-init 创建${NC}"
elif [[ "$CURRENT_TYPE" == "bigint" ]]; then
    echo -e "${YELLOW}   当前类型: bigint (需要修复)${NC}"
elif [[ "$CURRENT_TYPE" == "charactervarying(36)" ]]; then
    echo -e "${GREEN}   当前类型: varchar(36) (已正确)${NC}"
else
    echo -e "${YELLOW}   当前类型: $CURRENT_TYPE${NC}"
fi

# 5. 运行 backend-init
echo -e "\n${YELLOW}步骤 5/5: 运行 backend-init 进行数据库初始化...${NC}"
docker-compose up backend-init

# 检查初始化结果
if [ $? -eq 0 ]; then
    echo -e "\n${GREEN}======================================================================"
    echo "✓ backend-init 初始化成功！"
    echo -e "======================================================================${NC}"
    
    # 验证 task_id 字段类型
    echo -e "\n${YELLOW}验证修复结果...${NC}"
    FINAL_TYPE=$(docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -t -c "
        SELECT data_type || COALESCE('(' || character_maximum_length || ')', '')
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
          AND table_name = 'slurm_tasks' 
          AND column_name = 'task_id'
    " | tr -d '[:space:]')
    
    echo "   task_id 字段类型: $FINAL_TYPE"
    
    if [[ "$FINAL_TYPE" == "charactervarying(36)" ]]; then
        echo -e "${GREEN}   ✓ 字段类型正确！${NC}"
    else
        echo -e "${RED}   ✗ 字段类型不正确，期望 varchar(36)${NC}"
        exit 1
    fi
    
    # 重启 backend 服务
    echo -e "\n${YELLOW}重启 backend 服务...${NC}"
    docker-compose restart backend
    echo -e "${GREEN}✓ backend 服务已重启${NC}"
    
    echo -e "\n${GREEN}======================================================================"
    echo "✓ 修复完成！现在可以测试扩容功能了"
    echo -e "======================================================================${NC}"
    
else
    echo -e "\n${RED}======================================================================"
    echo "✗ backend-init 初始化失败"
    echo "请检查日志: docker-compose logs backend-init"
    echo -e "======================================================================${NC}"
    exit 1
fi
