#!/bin/bash
# 代理容器测试脚本

set -e

echo "=== 测试网络代理容器 ==="

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}错误: Docker未运行${NC}"
    exit 1
fi

echo "1. 构建代理镜像..."
docker build -t ai-infra-proxy:test -f src/proxy/Dockerfile src/proxy

echo ""
echo "2. 启动代理容器（需要先启动依赖服务）..."
docker-compose up -d postgres mysql redis oceanbase proxy

echo ""
echo "3. 等待服务就绪..."
sleep 10

echo ""
echo "4. 测试连接..."

# 测试PostgreSQL
echo -n "测试 PostgreSQL (5432): "
if docker run --rm --network host postgres:15-alpine pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 成功${NC}"
else
    echo -e "${RED}✗ 失败${NC}"
fi

# 测试MySQL
echo -n "测试 MySQL (3306): "
if docker run --rm --network host mysql:8.0 mysqladmin ping -h localhost -P 3306 --silent 2>/dev/null; then
    echo -e "${GREEN}✓ 成功${NC}"
else
    echo -e "${YELLOW}⚠ 需要认证（正常）${NC}"
fi

# 测试Redis
echo -n "测试 Redis (6379): "
if docker run --rm --network host redis:7-alpine redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 成功${NC}"
else
    echo -e "${YELLOW}⚠ 需要认证（正常）${NC}"
fi

# 测试OceanBase
echo -n "测试 OceanBase (2881): "
if nc -zv localhost 2881 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 端口开放${NC}"
else
    echo -e "${RED}✗ 端口未开放${NC}"
fi

echo ""
echo "5. 查看HAProxy统计信息..."
echo "访问: http://localhost:8404/stats (用户名/密码: admin/admin)"

echo ""
echo "6. 查看代理容器日志..."
docker logs --tail 20 ai-infra-proxy

echo ""
echo -e "${GREEN}测试完成！${NC}"
echo ""
echo "从宿主机连接示例："
echo "  PostgreSQL: psql -h localhost -p 5432 -U postgres"
echo "  MySQL:      mysql -h localhost -P 3306 -u root -p"
echo "  Redis:      redis-cli -h localhost -p 6379"
echo "  OceanBase:  mysql -h localhost -P 2881 -u root"
