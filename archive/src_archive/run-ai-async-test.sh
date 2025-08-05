#!/bin/bash

# Docker Compose AI异步架构测试脚本
set -e

echo "🐳 启动AI异步架构Docker Compose测试环境..."

# 设置颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目目录
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# 检查Docker和Docker Compose
echo -e "${BLUE}检查Docker环境...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker未安装${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker环境检查通过${NC}"

# 清理现有容器
echo -e "${BLUE}清理现有测试环境...${NC}"
docker-compose --profile ai-test --profile monitoring down --volumes --remove-orphans 2>/dev/null || true

# 构建和启动基础服务
echo -e "${BLUE}启动基础服务...${NC}"
docker-compose up -d postgres redis openldap

# 等待基础服务启动
echo -e "${YELLOW}等待基础服务启动...${NC}"
sleep 20

# 启动后端服务
echo -e "${BLUE}启动后端服务...${NC}"
docker-compose up -d backend

# 等待后端服务健康检查通过
echo -e "${YELLOW}等待后端服务健康检查...${NC}"
RETRY_COUNT=0
MAX_RETRIES=30

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker-compose ps backend | grep -q "healthy"; then
        echo -e "${GREEN}✅ 后端服务健康检查通过${NC}"
        break
    fi
    
    echo -e "${YELLOW}⏳ 等待后端服务启动... ($((RETRY_COUNT + 1))/$MAX_RETRIES)${NC}"
    sleep 10
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}❌ 后端服务启动超时${NC}"
    docker-compose logs backend
    exit 1
fi

# 启动前端服务
echo -e "${BLUE}启动前端服务...${NC}"
docker-compose up -d frontend

# 启动监控服务（可选）
echo -e "${BLUE}启动监控服务...${NC}"
docker-compose --profile monitoring up -d redis-insight

# 运行AI异步测试
echo -e "${BLUE}运行AI异步架构测试...${NC}"
docker-compose --profile ai-test up --build ai-async-test

# 获取测试结果
echo -e "${BLUE}收集测试结果...${NC}"

# 检查测试容器退出状态
TEST_EXIT_CODE=$(docker-compose ps -q ai-async-test | xargs docker inspect --format='{{.State.ExitCode}}')

if [ "$TEST_EXIT_CODE" = "0" ]; then
    echo -e "${GREEN}🎉 AI异步架构测试全部通过！${NC}"
else
    echo -e "${RED}❌ AI异步架构测试失败，退出码: $TEST_EXIT_CODE${NC}"
    echo -e "${YELLOW}查看测试日志:${NC}"
    docker-compose logs ai-async-test
fi

# 复制测试报告到本地
echo -e "${BLUE}复制测试报告到本地...${NC}"
REPORT_DIR="./test-reports/ai-async-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$REPORT_DIR"

# 从容器中复制报告
if docker-compose ps -q ai-async-test | head -1 | xargs -I {} docker cp {}:/tmp/ai-async-test-reports/. "$REPORT_DIR" 2>/dev/null; then
    echo -e "${GREEN}✅ 测试报告已复制到: $REPORT_DIR${NC}"
    
    # 显示报告摘要
    if [ -f "$REPORT_DIR/status_summary.txt" ]; then
        echo -e "${BLUE}测试结果摘要:${NC}"
        cat "$REPORT_DIR/status_summary.txt"
    fi
else
    echo -e "${YELLOW}⚠️  无法复制测试报告，可能测试未完成${NC}"
fi

# 显示服务状态
echo -e "${BLUE}当前服务状态:${NC}"
docker-compose ps

# 显示访问信息
echo -e "${BLUE}服务访问信息:${NC}"
echo -e "${GREEN}📊 Redis Insight: http://localhost:8001${NC}"
echo -e "${GREEN}🌐 前端应用: http://localhost:3001${NC}"
echo -e "${GREEN}🔗 后端API: http://localhost:8082${NC}"

# 选择是否保持服务运行
echo -e "${YELLOW}是否保持服务运行以便进一步测试？(y/N)${NC}"
read -r KEEP_RUNNING

if [[ $KEEP_RUNNING =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}服务将继续运行...${NC}"
    echo -e "${BLUE}停止服务请运行: docker-compose --profile ai-test --profile monitoring down${NC}"
else
    echo -e "${BLUE}停止测试环境...${NC}"
    docker-compose --profile ai-test --profile monitoring down
    echo -e "${GREEN}✅ 测试环境已清理${NC}"
fi

# 返回测试结果
exit $TEST_EXIT_CODE
