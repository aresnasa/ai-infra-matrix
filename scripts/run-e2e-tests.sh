#!/bin/bash

# AI Infrastructure Matrix - E2E 测试运行脚本
# 
# 使用方法:
#   ./run-e2e-tests.sh [选项]
#
# 选项:
#   --quick       运行快速验证测试
#   --full        运行完整测试套件
#   --headed      显示浏览器窗口（默认无头模式）
#   --debug       调试模式（保留测试痕迹和截图）
#   --url URL     指定测试的 base URL（默认: http://192.168.0.200:8080）

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"
TEST_MODE="quick"
HEADED=""
DEBUG_MODE=""
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --quick)
      TEST_MODE="quick"
      shift
      ;;
    --full)
      TEST_MODE="full"
      shift
      ;;
    --headed)
      HEADED="--headed"
      shift
      ;;
    --debug)
      DEBUG_MODE="--debug"
      shift
      ;;
    --url)
      BASE_URL="$2"
      shift 2
      ;;
    --help)
      echo "使用方法: ./run-e2e-tests.sh [选项]"
      echo ""
      echo "选项:"
      echo "  --quick       运行快速验证测试（默认）"
      echo "  --full        运行完整测试套件"
      echo "  --headed      显示浏览器窗口"
      echo "  --debug       调试模式"
      echo "  --url URL     指定 base URL"
      echo "  --help        显示帮助信息"
      exit 0
      ;;
    *)
      echo -e "${RED}错误: 未知选项 $1${NC}"
      exit 1
      ;;
  esac
done

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}AI Infrastructure Matrix - E2E 测试${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "测试模式: ${YELLOW}${TEST_MODE}${NC}"
echo -e "Base URL: ${YELLOW}${BASE_URL}${NC}"
echo -e "浏览器: ${YELLOW}$([ -n "$HEADED" ] && echo "可见" || echo "无头")${NC}"
echo ""

# 检查服务是否运行
echo -e "${YELLOW}检查服务状态...${NC}"
if curl -s "${BASE_URL}" > /dev/null 2>&1; then
  echo -e "${GREEN}✓ 服务正在运行${NC}"
else
  echo -e "${RED}✗ 无法连接到 ${BASE_URL}${NC}"
  echo -e "${YELLOW}请确保服务已启动：docker-compose up -d${NC}"
  exit 1
fi

# 进入测试目录
cd "$(dirname "$0")/test/e2e"

# 确保依赖已安装
if [ ! -d "node_modules" ]; then
  echo -e "${YELLOW}安装 Playwright 依赖...${NC}"
  npm install @playwright/test
fi

# 确保浏览器已安装
echo -e "${YELLOW}检查 Playwright 浏览器...${NC}"
npx playwright install chromium

# 运行测试
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}开始测试...${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 设置环境变量
export BASE_URL
export ADMIN_USERNAME
export ADMIN_PASSWORD

# 绕过代理（如果配置了代理）
# 确保本地 IP 不走代理
if [ -n "$http_proxy" ] || [ -n "$HTTP_PROXY" ]; then
  echo -e "${YELLOW}检测到 HTTP 代理配置，添加本地 IP 到 no_proxy...${NC}"
  export no_proxy="${no_proxy},192.168.0.200,192.168.0.*,localhost,127.0.0.1"
  export NO_PROXY="${NO_PROXY},192.168.0.200,192.168.0.*,localhost,127.0.0.1"
  echo -e "${GREEN}✓ 已配置绕过代理${NC}"
fi

# 根据测试模式运行不同的测试
if [ "$TEST_MODE" = "quick" ]; then
  echo -e "${YELLOW}运行快速验证测试...${NC}"
  npx playwright test specs/quick-validation-test.spec.js \
    --config=playwright.config.js \
    $HEADED \
    $DEBUG_MODE
elif [ "$TEST_MODE" = "full" ]; then
  echo -e "${YELLOW}运行完整测试套件...${NC}"
  npx playwright test specs/complete-e2e-test.spec.js \
    --config=playwright.config.js \
    $HEADED \
    $DEBUG_MODE
fi

TEST_EXIT_CODE=$?

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo -e "${GREEN}✓ 所有测试通过！${NC}"
else
  echo -e "${RED}✗ 部分测试失败${NC}"
  echo -e "${YELLOW}查看详细报告: test/e2e/test-results/${NC}"
fi
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 显示测试报告位置
if [ -d "test-results" ]; then
  echo -e "${YELLOW}测试结果:${NC}"
  echo -e "  - 报告目录: $(pwd)/test-results/"
  echo -e "  - 截图目录: $(pwd)/test-results/"
  echo ""
fi

exit $TEST_EXIT_CODE
