#!/bin/bash

# Monitoring Page 404 Fix - Deployment Script
# 修复 Nightingale 监控页面显示 "404 - The page you visited does not exist!" 的问题

set -e

echo "=========================================="
echo "修复监控页面 404 问题"
echo "=========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查 docker-compose
if ! command -v docker-compose &> /dev/null; then
    echo "${RED}❌ docker-compose 命令未找到${NC}"
    exit 1
fi

echo "${YELLOW}问题原因:${NC}"
echo "Nightingale iframe 默认加载 /nightingale/ 根路径"
echo "该路径没有默认页面，显示 404 错误"
echo ""

echo "${YELLOW}解决方案:${NC}"
echo "将默认路径改为 /nightingale/metrics (指标浏览器页面)"
echo ""

echo "${YELLOW}步骤 1: 重新构建 frontend 容器${NC}"
echo "--------------------------------------"
docker-compose build frontend

echo ""
echo "${GREEN}✅ Frontend 构建完成${NC}"
echo ""

echo "${YELLOW}步骤 2: 重启 frontend 容器${NC}"
echo "--------------------------------------"
docker-compose up -d frontend

echo ""
echo "${GREEN}✅ Frontend 重启完成${NC}"
echo ""

# 等待容器启动
echo "等待 frontend 容器完全启动..."
sleep 5

# 检查容器状态
if docker-compose ps frontend | grep -q "Up"; then
    echo "${GREEN}✅ Frontend 容器运行正常${NC}"
else
    echo "${RED}❌ Frontend 容器启动失败${NC}"
    echo "请检查容器日志: docker-compose logs frontend"
    exit 1
fi

echo ""
echo "=========================================="
echo "${GREEN}修复完成!${NC}"
echo "=========================================="
echo ""
echo "请访问以下地址测试:"
echo "  http://192.168.0.200:8080/monitoring"
echo ""
echo "预期结果:"
echo "  ✅ 不再显示 '404 - The page you visited does not exist!'"
echo "  ✅ 直接显示 Nightingale 指标浏览器 (Metrics Explorer)"
echo ""
echo "如果仍有问题，请清除浏览器缓存后重试"
echo ""
