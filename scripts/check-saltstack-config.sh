#!/bin/bash
# ====================================================================
# SaltStack 配置验证和修复脚本
# ====================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}SaltStack 配置诊断${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""

# 检查 .env.example
echo -e "${BLUE}[1] 检查 .env.example${NC}"
if [ -f .env.example ]; then
    EXAMPLE_PORT=$(grep "^SALT_API_PORT=" .env.example | cut -d'=' -f2)
    EXAMPLE_URL=$(grep "^SALTSTACK_MASTER_URL=" .env.example | cut -d'=' -f2)
    
    echo "  SALT_API_PORT: $EXAMPLE_PORT"
    echo "  SALTSTACK_MASTER_URL: $EXAMPLE_URL"
    
    if [ "$EXAMPLE_PORT" = "8000" ]; then
        echo -e "  ${GREEN}✓ .env.example 配置正确（端口 8000）${NC}"
    else
        echo -e "  ${RED}✗ .env.example 配置错误（端口 $EXAMPLE_PORT，应为 8000）${NC}"
    fi
else
    echo -e "  ${RED}✗ .env.example 不存在${NC}"
fi

echo ""

# 检查 .env
echo -e "${BLUE}[2] 检查 .env${NC}"
if [ -f .env ]; then
    ENV_PORT=$(grep "^SALT_API_PORT=" .env | cut -d'=' -f2)
    ENV_URL=$(grep "^SALTSTACK_MASTER_URL=" .env | cut -d'=' -f2)
    
    echo "  SALT_API_PORT: $ENV_PORT"
    echo "  SALTSTACK_MASTER_URL: $ENV_URL"
    
    # 检查端口是否与 example 一致
    if [ "$ENV_PORT" = "$EXAMPLE_PORT" ]; then
        echo -e "  ${GREEN}✓ .env 配置与 .env.example 一致（端口 $ENV_PORT）${NC}"
    else
        echo -e "  ${YELLOW}⚠ .env 配置与 .env.example 不一致${NC}"
        echo -e "  ${YELLOW}  当前端口: $ENV_PORT，推荐端口: $EXAMPLE_PORT${NC}"
        echo -e "  ${YELLOW}  如果 SaltStack 无法连接，建议检查实际端口配置${NC}"
    fi
else
    echo -e "  ${RED}✗ .env 不存在${NC}"
fi

echo ""

# 检查 Salt Master 连接性
echo -e "${BLUE}[3] 检查 Salt Master 连接性${NC}"

# 从配置中提取主机和端口
SALT_HOST=$(echo "$ENV_URL" | sed 's|http://||' | sed 's|https://||' | cut -d':' -f1)
SALT_PORT=$(echo "$ENV_URL" | sed 's|.*:||' | sed 's|/.*||')

echo "  检测目标: $SALT_HOST:$SALT_PORT"

# 尝试配置的端口
if timeout 2 bash -c "echo > /dev/tcp/$SALT_HOST/$SALT_PORT" 2>/dev/null; then
    echo -e "  ${GREEN}✓ Salt Master $SALT_HOST:$SALT_PORT 可连接${NC}"
elif timeout 2 bash -c "echo > /dev/tcp/192.168.0.200/$SALT_PORT" 2>/dev/null; then
    echo -e "  ${GREEN}✓ Salt Master (IP: 192.168.0.200:$SALT_PORT) 可连接${NC}"
else
    echo -e "  ${YELLOW}⚠ Salt Master $SALT_HOST:$SALT_PORT 无法连接（可能容器未启动）${NC}"
fi

echo ""
echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}建议操作${NC}"
echo -e "${BLUE}=====================================${NC}"

if [ "$ENV_PORT" != "$EXAMPLE_PORT" ]; then
    echo ""
    echo -e "${YELLOW}您的 .env 配置与 .env.example 不一致${NC}"
    echo ""
    echo "推荐操作："
    echo "  1. 手动编辑 .env 文件："
    echo -e "     ${BLUE}vim .env${NC}"
    echo ""
    echo "  2. 修改以下配置项为推荐值："
    echo -e "     ${GREEN}SALT_API_PORT=$EXAMPLE_PORT${NC}"
    echo -e "     ${GREEN}SALTSTACK_MASTER_URL=$EXAMPLE_URL${NC}"
    echo ""
    echo "  3. 或者运行同步命令："
    echo -e "     ${BLUE}./build.sh build-all${NC}"
    echo "     (会自动同步 .env.example 的新配置到 .env)"
    echo ""
    echo "  4. 重启后端服务："
    echo -e "     ${BLUE}docker-compose restart backend${NC}"
    echo ""
else
    echo -e "${GREEN}✓ 配置与推荐值一致！${NC}"
    echo ""
    echo "如果仍无法连接 SaltStack，请检查："
    echo "  1. Salt Master 容器是否正常运行: docker-compose ps saltstack"
    echo "  2. 后端服务日志: docker-compose logs backend"
    echo "  3. Salt Master 实际监听端口: docker-compose exec saltstack netstat -tlnp | grep 8000"
    echo "  4. 网络连接: docker-compose exec backend ping saltstack"
fi
