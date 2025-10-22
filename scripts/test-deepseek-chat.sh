#!/bin/bash

# DeepSeek 聊天集成测试运行脚本
# 用途：使用操作系统环境变量 DEEPSEEK_API_KEY 运行 Playwright 测试
# 注意：API Key 不会写入任何文件，只从操作系统环境变量读取
# 日期：2025-10-21

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}DeepSeek 聊天集成测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查操作系统环境变量中的 DEEPSEEK_API_KEY
echo -e "${CYAN}🔍 检查操作系统环境变量...${NC}"
echo ""

if [ -z "$DEEPSEEK_API_KEY" ]; then
    echo -e "${RED}❌ 错误: 未找到环境变量 DEEPSEEK_API_KEY${NC}"
    echo ""
    echo -e "${YELLOW}本测试需要从操作系统环境变量读取 DEEPSEEK_API_KEY${NC}"
    echo -e "${YELLOW}（不使用 .env 文件，以确保 API Key 安全）${NC}"
    echo ""
    echo "📝 设置方法："
    echo ""
    echo -e "${GREEN}1. 临时设置（仅本次会话有效）:${NC}"
    echo "   $ export DEEPSEEK_API_KEY=sk-your-real-api-key"
    echo "   $ ./test-deepseek-chat.sh"
    echo ""
    echo -e "${GREEN}2. 永久设置（推荐 - 添加到 shell 配置文件）:${NC}"
    echo "   $ echo 'export DEEPSEEK_API_KEY=sk-your-real-api-key' >> ~/.zshrc"
    echo "   $ source ~/.zshrc"
    echo "   $ ./test-deepseek-chat.sh"
    echo ""
    echo -e "${GREEN}3. 单次运行（推荐用于测试）:${NC}"
    echo "   $ DEEPSEEK_API_KEY=sk-your-real-api-key ./test-deepseek-chat.sh"
    echo ""
    echo "🌐 获取 API Key:"
    echo "   访问 https://platform.deepseek.com 注册并获取"
    echo ""
    exit 1
fi

# 验证 API Key 格式
if [[ ! "$DEEPSEEK_API_KEY" =~ ^sk- ]]; then
    echo -e "${RED}❌ 错误: DEEPSEEK_API_KEY 格式不正确${NC}"
    echo "   API Key 应该以 'sk-' 开头"
    echo "   当前值: $DEEPSEEK_API_KEY"
    exit 1
fi

# 检查是否是测试占位符
if [ "$DEEPSEEK_API_KEY" = "sk-test-deepseek-api-key-for-testing" ]; then
    echo -e "${YELLOW}⚠️  警告: 检测到测试占位符 API Key${NC}"
    echo ""
    echo "测试将继续运行，但可能会因为 API Key 无效而失败"
    echo ""
    echo -e "${YELLOW}按 Ctrl+C 退出，或按 Enter 继续...${NC}"
    read -r
fi

# 显示当前配置（隐藏大部分 API Key）
KEY_PREFIX="${DEEPSEEK_API_KEY:0:10}"
KEY_SUFFIX="${DEEPSEEK_API_KEY: -4}"
KEY_MASKED="${KEY_PREFIX}...${KEY_SUFFIX}"

echo -e "${GREEN}✓ 检测到有效的 DEEPSEEK_API_KEY${NC}"
echo "  API Key: $KEY_MASKED (已隐藏中间部分)"
echo "  来源: 操作系统环境变量"
echo "  安全性: ✓ 未写入任何文件"
echo ""

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}开始运行测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 设置测试 URL
export BASE_URL=${BASE_URL:-"http://192.168.0.200:8080"}

echo "测试配置:"
echo "  BASE_URL: $BASE_URL"
echo "  DEEPSEEK_API_KEY: ${DEEPSEEK_API_KEY:0:10}... (已隐藏)"
echo ""

# 检查是否指定了特定测试
if [ -n "$3" ]; then
    TEST_FILTER="$3"
    echo "运行特定测试: $TEST_FILTER"
    npx playwright test test/e2e/specs/deepseek-chat-integration.spec.js \
        --grep "$TEST_FILTER" \
        --reporter=line \
        --timeout=120000
else
    echo "运行所有测试"
    npx playwright test test/e2e/specs/deepseek-chat-integration.spec.js \
        --reporter=line \
        --timeout=120000
fi

EXIT_CODE=$?

echo ""
echo -e "${BLUE}========================================${NC}"
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ 测试完成${NC}"
else
    echo -e "${RED}❌ 测试失败${NC}"
    echo ""
    echo "常见问题排查:"
    echo "  1. API Key 是否有效？"
    echo "  2. Backend 服务是否正在运行？"
    echo "  3. 网络是否可以访问 DeepSeek API？"
    echo ""
    echo "查看日志:"
    echo "  docker logs ai-infra-backend | tail -100"
fi
echo -e "${BLUE}========================================${NC}"

exit $EXIT_CODE
