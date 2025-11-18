#!/bin/bash

# 简易测试运行脚本
# 使用方法: cd test/e2e && ./run-test.sh

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "AI Infrastructure Matrix - E2E 测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 完全禁用代理
unset http_proxy
unset https_proxy
unset HTTP_PROXY
unset HTTPS_PROXY
export no_proxy="*"
export NO_PROXY="*"

BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

echo "Base URL: $BASE_URL"
echo "代理: 已禁用"
echo ""

# 检查服务
echo "检查服务状态..."
if curl -s -o /dev/null -w "%{http_code}" "$BASE_URL" | grep -q "200\|302"; then
  echo "✓ 服务正在运行"
else
  echo "✗ 服务未响应"
  echo ""
  echo "请先启动服务:"
  echo "  cd ../.. && docker-compose up -d"
  exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "运行快速验证测试..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 运行测试
BASE_URL="$BASE_URL" \
ADMIN_USERNAME="$ADMIN_USERNAME" \
ADMIN_PASSWORD="$ADMIN_PASSWORD" \
npx playwright test specs/quick-validation-test.spec.js \
  --config=playwright.config.js

EXIT_CODE=$?

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $EXIT_CODE -eq 0 ]; then
  echo "✓ 测试通过！"
else
  echo "✗ 测试失败"
  echo ""
  echo "查看详细报告:"
  echo "  npx playwright show-report"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit $EXIT_CODE
