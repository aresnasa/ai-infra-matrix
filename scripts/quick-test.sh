#!/bin/bash

# 快速测试脚本 - 修复代理问题
# 使用方法: ./quick-test.sh

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "AI Infrastructure Matrix - 快速测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 配置
BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

echo "Base URL: $BASE_URL"
echo ""

# 关键：绕过代理
echo "配置网络代理绕过..."
export no_proxy="192.168.0.200,192.168.0.*,localhost,127.0.0.1,10.*.*.*,172.0.*.*"
export NO_PROXY="192.168.0.200,192.168.0.*,localhost,127.0.0.1,10.*.*.*,172.0.*.*"
echo "✓ 代理配置完成"
echo ""

# 检查服务
echo "检查服务状态..."
if curl --noproxy "*" -s -o /dev/null -w "%{http_code}" "$BASE_URL" | grep -q "200\|302\|301"; then
  echo "✓ 服务正在运行"
else
  echo "✗ 无法连接到 $BASE_URL"
  echo "尝试启动服务: docker-compose up -d"
  exit 1
fi
echo ""

# 进入测试目录
cd "$(dirname "$0")/test/e2e"

# 确保依赖已安装
if [ ! -d "node_modules" ]; then
  echo "安装依赖..."
  npm install @playwright/test
fi

# 确保浏览器已安装
echo "检查 Playwright 浏览器..."
npx playwright install chromium --with-deps

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "开始测试..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 运行测试（传递环境变量）
BASE_URL="$BASE_URL" \
ADMIN_USERNAME="$ADMIN_USERNAME" \
ADMIN_PASSWORD="$ADMIN_PASSWORD" \
NO_PROXY="*" \
no_proxy="*" \
npx playwright test specs/quick-validation-test.spec.js \
  --config=playwright.config.js

TEST_EXIT_CODE=$?

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo "✓ 所有测试通过！"
else
  echo "✗ 部分测试失败"
  echo "查看详细报告: cd test/e2e && npx playwright show-report"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

exit $TEST_EXIT_CODE
