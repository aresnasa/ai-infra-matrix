#!/bin/bash
# 修复 /monitoring 路由并测试

set -e

echo "================================================"
echo "修复 /monitoring 路由配置"
echo "================================================"

# 1. 渲染 Nginx 配置
echo "步骤 1: 渲染 Nginx 配置模板..."
./build.sh render-templates

# 2. 重载 Nginx
echo "步骤 2: 重载 Nginx 服务..."
docker exec nginx nginx -s reload

# 3. 等待服务稳定
echo "步骤 3: 等待服务稳定..."
sleep 2

# 4. 运行 Playwright 测试
echo "步骤 4: 运行 Playwright E2E 测试..."
BASE_URL=http://192.168.0.200:8080 npx playwright test test/e2e/specs/monitoring-proxy-redirect.spec.ts --reporter=line

echo ""
echo "================================================"
echo "✓ 修复完成并测试通过！"
echo "================================================"
