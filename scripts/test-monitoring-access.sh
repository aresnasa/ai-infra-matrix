#!/bin/bash

echo "======================================="
echo "监控页面访问测试脚本"
echo "======================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="http://192.168.18.154:8080"

echo "1. 测试 /monitoring 页面是否返回 HTML..."
MONITORING_RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/monitoring")
HTTP_CODE=$(echo "$MONITORING_RESPONSE" | tail -n 1)
CONTENT=$(echo "$MONITORING_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓${NC} HTTP 200 OK"
    
    # 检查是否包含 React 应用
    if echo "$CONTENT" | grep -q "root"; then
        echo -e "${GREEN}✓${NC} 返回了 React 应用 HTML"
    else
        echo -e "${RED}✗${NC} 未找到 React root 元素"
    fi
else
    echo -e "${RED}✗${NC} HTTP $HTTP_CODE"
fi

echo ""
echo "2. 测试 Nightingale 服务是否可访问..."
NIGHTINGALE_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/nightingale/")
if [ "$NIGHTINGALE_CODE" = "200" ]; then
    echo -e "${GREEN}✓${NC} Nightingale 服务正常 (HTTP 200)"
else
    echo -e "${RED}✗${NC} Nightingale 服务异常 (HTTP $NIGHTINGALE_CODE)"
fi

echo ""
echo "3. 检查前端容器状态..."
FRONTEND_STATUS=$(docker inspect ai-infra-frontend --format='{{.State.Status}}' 2>/dev/null)
if [ "$FRONTEND_STATUS" = "running" ]; then
    echo -e "${GREEN}✓${NC} 前端容器运行中"
    
    # 获取容器启动时间
    START_TIME=$(docker inspect ai-infra-frontend --format='{{.State.StartedAt}}')
    echo "   启动时间: $START_TIME"
    
    # 检查构建时间
    BUILD_TIME=$(docker inspect ai-infra-frontend --format='{{.Created}}')
    echo "   构建时间: $BUILD_TIME"
else
    echo -e "${RED}✗${NC} 前端容器状态: $FRONTEND_STATUS"
fi

echo ""
echo "4. 检查 Nginx 容器状态..."
NGINX_STATUS=$(docker inspect ai-infra-nginx --format='{{.State.Status}}' 2>/dev/null)
if [ "$NGINX_STATUS" = "running" ]; then
    echo -e "${GREEN}✓${NC} Nginx 容器运行中"
    
    # 检查 Nginx 配置
    echo ""
    echo "   检查 Nginx absolute_redirect 配置..."
    if docker exec ai-infra-nginx grep -q "absolute_redirect off" /etc/nginx/conf.d/server-main.conf 2>/dev/null; then
        echo -e "   ${GREEN}✓${NC} absolute_redirect off 已配置"
    else
        echo -e "   ${YELLOW}!${NC} absolute_redirect off 未找到"
    fi
else
    echo -e "${RED}✗${NC} Nginx 容器状态: $NGINX_STATUS"
fi

echo ""
echo "======================================="
echo "浏览器访问建议："
echo "======================================="
echo ""
echo "如果上述测试都通过，但浏览器仍无法访问，请尝试："
echo ""
echo "1. 清除浏览器缓存："
echo -e "   ${YELLOW}Chrome/Edge:${NC} F12 -> Network -> 右键刷新按钮 -> 清空缓存并硬性重新加载"
echo -e "   ${YELLOW}Firefox:${NC} F12 -> 设置 -> 停用缓存"
echo ""
echo "2. 清除 localStorage："
echo "   F12 -> Console -> 执行: localStorage.clear()"
echo ""
echo "3. 使用无痕/隐私模式重新测试"
echo ""
echo "4. 确认使用管理员账号登录："
echo "   用户名: admin"
echo "   密码: admin123"
echo ""
echo "5. 直接访问链接："
echo "   $BASE_URL/monitoring"
echo ""
