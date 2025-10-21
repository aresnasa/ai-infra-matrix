#!/bin/bash

# 高级功能 E2E 测试运行脚本
# 包括 SLURM 节点扩容测试和聊天机器人测试

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"
DB_HOST="${DB_HOST:-localhost}"  # 改为 localhost，假设通过端口转发访问
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ai_infra}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres123}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

# 测试选项
TEST_MODE="all"  # all, slurm, chatbot
HEADED=""
DEBUG_MODE=""

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --slurm)
      TEST_MODE="slurm"
      shift
      ;;
    --chatbot)
      TEST_MODE="chatbot"
      shift
      ;;
    --all)
      TEST_MODE="all"
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
    --db-host)
      DB_HOST="$2"
      shift 2
      ;;
    --help)
      echo "使用方法: ./run-advanced-e2e-tests.sh [选项]"
      echo ""
      echo "选项:"
      echo "  --all         运行所有高级测试（默认）"
      echo "  --slurm       只运行 SLURM 节点扩容测试"
      echo "  --chatbot     只运行聊天机器人测试"
      echo "  --headed      显示浏览器窗口"
      echo "  --debug       调试模式"
      echo "  --url URL     指定 base URL"
      echo "  --db-host HOST 指定数据库主机"
      echo "  --help        显示帮助信息"
      exit 0
      ;;
    *)
      echo -e "${RED}错误: 未知选项 $1${NC}"
      exit 1
      ;;
  esac
done

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}AI Infrastructure Matrix - 高级功能 E2E 测试${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "测试模式: ${YELLOW}${TEST_MODE}${NC}"
echo -e "Base URL: ${YELLOW}${BASE_URL}${NC}"
echo -e "数据库: ${YELLOW}${DB_HOST}:${DB_PORT}/${DB_NAME}${NC}"
echo -e "浏览器: ${YELLOW}$([ -n "$HEADED" ] && echo "可见" || echo "无头")${NC}"
echo ""

# 检查服务状态
echo -e "${YELLOW}检查服务状态...${NC}"
if curl -s "${BASE_URL}" > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Web 服务正在运行${NC}"
else
  echo -e "${RED}✗ 无法连接到 ${BASE_URL}${NC}"
  echo -e "${YELLOW}请确保服务已启动：docker-compose up -d${NC}"
  exit 1
fi

# 检查数据库连接
echo -e "${YELLOW}检查数据库连接...${NC}"
if command -v psql &> /dev/null; then
  if PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -U "${DB_USER}" -d "${DB_NAME}" -c '\q' 2>/dev/null; then
    echo -e "${GREEN}✓ 数据库连接正常${NC}"
  else
    echo -e "${YELLOW}⚠ 数据库连接失败（聊天机器人测试可能受影响）${NC}"
  fi
else
  echo -e "${YELLOW}⚠ psql 未安装，跳过数据库检查${NC}"
fi

# 进入测试目录
cd "$(dirname "$0")/test/e2e"

# 确保依赖已安装
if [ ! -d "node_modules" ]; then
  echo -e "${YELLOW}安装测试依赖...${NC}"
  npm install
fi

# 确保浏览器已安装
echo -e "${YELLOW}检查 Playwright 浏览器...${NC}"
npx playwright install chromium

# 设置环境变量
export BASE_URL
export DB_HOST
export DB_PORT
export DB_NAME
export DB_USER
export DB_PASSWORD
export ADMIN_USERNAME
export ADMIN_PASSWORD

# 绕过代理
if [ -n "$http_proxy" ] || [ -n "$HTTP_PROXY" ]; then
  echo -e "${YELLOW}配置代理绕过...${NC}"
  LOCAL_IP=$(echo "${BASE_URL}" | sed -E 's|https?://([^:/]+).*|\1|')
  export no_proxy="${no_proxy},${LOCAL_IP},localhost,127.0.0.1"
  export NO_PROXY="${NO_PROXY},${LOCAL_IP},localhost,127.0.0.1"
  echo -e "${GREEN}✓ 已配置绕过代理${NC}"
fi

# 运行测试
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}开始测试...${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

TEST_EXIT_CODE=0

# SLURM 节点扩容测试
if [ "$TEST_MODE" = "all" ] || [ "$TEST_MODE" = "slurm" ]; then
  echo -e "${GREEN}━━━ SLURM 节点扩容测试 ━━━${NC}"
  echo ""
  
  npx playwright test specs/slurm-node-expansion-test.spec.js \
    --config=playwright.config.js \
    --reporter=list \
    $HEADED \
    $DEBUG_MODE || TEST_EXIT_CODE=$?
  
  echo ""
fi

# 聊天机器人测试
if [ "$TEST_MODE" = "all" ] || [ "$TEST_MODE" = "chatbot" ]; then
  echo -e "${GREEN}━━━ 聊天机器人 Kafka 测试 ━━━${NC}"
  echo ""
  
  npx playwright test specs/chatbot-kafka-test.spec.js \
    --config=playwright.config.js \
    --reporter=list \
    $HEADED \
    $DEBUG_MODE || TEST_EXIT_CODE=$?
  
  echo ""
fi

# 测试结果
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo -e "${GREEN}✓ 所有测试通过！${NC}"
else
  echo -e "${RED}✗ 部分测试失败${NC}"
  echo -e "${YELLOW}查看详细报告: test/e2e/test-results/${NC}"
fi
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 显示测试报告位置
if [ -d "test-results" ]; then
  echo -e "${YELLOW}测试结果:${NC}"
  echo -e "  - 报告目录: $(pwd)/test-results/"
  echo -e "  - 截图目录: $(pwd)/test-results/"
  echo ""
  
  # 统计测试文件
  SLURM_RESULTS=$(find test-results -name "*slurm-node-expansion*" -type d 2>/dev/null | wc -l)
  CHATBOT_RESULTS=$(find test-results -name "*chatbot-kafka*" -type d 2>/dev/null | wc -l)
  
  if [ $SLURM_RESULTS -gt 0 ]; then
    echo -e "${GREEN}  ✓ SLURM 测试结果: ${SLURM_RESULTS} 个${NC}"
  fi
  
  if [ $CHATBOT_RESULTS -gt 0 ]; then
    echo -e "${GREEN}  ✓ 聊天机器人测试结果: ${CHATBOT_RESULTS} 个${NC}"
  fi
  
  echo ""
fi

exit $TEST_EXIT_CODE
