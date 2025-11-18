#!/bin/bash
# ====================================================================
# 完整重建 backend-init 并修复数据库类型脚本
# ====================================================================
# 用途: 从源头重新构建 backend-init，确保使用最新的修复代码
# ====================================================================

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================================================"
echo "完整重建 backend-init 并修复数据库"
echo -e "======================================================================${NC}"

# 显示当前问题
echo -e "\n${YELLOW}检查当前问题...${NC}"
CURRENT_TYPE=$(docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -t -c "
    SELECT data_type 
    FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'slurm_tasks' 
      AND column_name = 'task_id'
" 2>/dev/null | tr -d '[:space:]' || echo "not_exists")

echo "   当前 task_id 类型: $CURRENT_TYPE"
if [[ "$CURRENT_TYPE" == "bigint" ]]; then
    echo -e "   ${RED}✗ 类型错误！需要修复为 varchar(36)${NC}"
else
    echo -e "   ${GREEN}✓ 类型正确或表不存在${NC}"
fi

# 步骤1: 停止相关服务
echo -e "\n${YELLOW}步骤 1/6: 停止相关服务...${NC}"
docker-compose stop backend backend-init
echo -e "${GREEN}✓ 服务已停止${NC}"

# 步骤2: 删除旧容器
echo -e "\n${YELLOW}步骤 2/6: 删除旧容器...${NC}"
docker-compose rm -f backend-init backend
echo -e "${GREEN}✓ 旧容器已删除${NC}"

# 步骤3: 删除旧镜像（强制重新构建）
echo -e "\n${YELLOW}步骤 3/6: 删除旧的 backend-init 镜像...${NC}"
docker rmi ai-infra-backend-init:v0.3.6-dev 2>/dev/null || true
echo -e "${GREEN}✓ 旧镜像已删除${NC}"

# 步骤4: 重新构建 backend-init 镜像
echo -e "\n${YELLOW}步骤 4/6: 重新构建 backend-init 镜像（使用最新代码）...${NC}"
echo "   这可能需要几分钟时间..."
docker-compose build --no-cache backend-init
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ backend-init 镜像构建成功${NC}"
else
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi

# 步骤5: 运行 backend-init 初始化
echo -e "\n${YELLOW}步骤 5/6: 运行 backend-init 初始化数据库...${NC}"
docker-compose up backend-init

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 数据库初始化成功${NC}"
else
    echo -e "${RED}✗ 数据库初始化失败${NC}"
    echo "查看日志: docker-compose logs backend-init"
    exit 1
fi

# 步骤6: 验证修复结果
echo -e "\n${YELLOW}步骤 6/6: 验证修复结果...${NC}"
sleep 2

FINAL_TYPE=$(docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -t -c "
    SELECT data_type || COALESCE('(' || character_maximum_length || ')', '')
    FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'slurm_tasks' 
      AND column_name = 'task_id'
" 2>/dev/null | tr -d '[:space:]')

echo ""
echo "   最终 task_id 类型: $FINAL_TYPE"

if [[ "$FINAL_TYPE" == "charactervarying(36)" ]]; then
    echo -e "   ${GREEN}✓ 字段类型正确！${NC}"
    
    # 显示完整的表结构
    echo -e "\n${YELLOW}完整表结构:${NC}"
    docker-compose exec -T postgres psql -U postgres -d ai_infra_matrix -c "\d slurm_tasks" | grep -A 2 -B 2 task_id
    
    # 重新构建并启动 backend
    echo -e "\n${YELLOW}重新构建并启动 backend 服务...${NC}"
    docker-compose build backend
    docker-compose up -d backend
    
    echo -e "\n${GREEN}======================================================================"
    echo "✓✓✓ 修复完成！backend-init 和 backend 都已更新"
    echo -e "======================================================================${NC}"
    
    echo -e "\n${YELLOW}测试建议:${NC}"
    echo "  1. 等待 backend 服务完全启动（约10秒）"
    echo "  2. 访问前端并尝试扩容操作"
    echo "  3. 如果仍有问题，查看日志:"
    echo "     docker-compose logs backend --tail=50"
    
elif [[ "$FINAL_TYPE" == "bigint" ]]; then
    echo -e "   ${RED}✗ 字段类型仍然是 bigint！${NC}"
    echo ""
    echo "可能的原因:"
    echo "  1. GORM 模型定义未生效"
    echo "  2. fixSlurmTasksTableSchema() 函数未执行"
    echo ""
    echo "请检查 backend-init 日志:"
    echo "  docker-compose logs backend-init | grep -i 'slurm\|task_id\|fix'"
    exit 1
else
    echo -e "   ${YELLOW}⚠ 意外的类型: $FINAL_TYPE${NC}"
    exit 1
fi

echo ""
