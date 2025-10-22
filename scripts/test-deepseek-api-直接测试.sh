#!/bin/bash

# DeepSeek API 直接测试脚本
# 用于验证 API Key 是否有效以及网络连接是否正常

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}DeepSeek API 直接测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查 API Key
if [ -z "$DEEPSEEK_API_KEY" ]; then
    echo -e "${RED}❌ 错误: 未找到环境变量 DEEPSEEK_API_KEY${NC}"
    echo ""
    echo "请先设置环境变量："
    echo "  export DEEPSEEK_API_KEY=sk-your-real-api-key"
    exit 1
fi

# 显示 API Key (脱敏)
KEY_PREFIX="${DEEPSEEK_API_KEY:0:10}"
KEY_SUFFIX="${DEEPSEEK_API_KEY: -4}"
KEY_MASKED="${KEY_PREFIX}...${KEY_SUFFIX}"

echo -e "${GREEN}✓ API Key: $KEY_MASKED${NC}"
echo ""

# 测试 1: 测试 DeepSeek Chat API
echo -e "${CYAN}测试 1: DeepSeek Chat API${NC}"
echo "API Endpoint: https://api.deepseek.com/v1/chat/completions"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "https://api.deepseek.com/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${DEEPSEEK_API_KEY}" \
  -d '{
    "model": "deepseek-chat",
    "messages": [
      {
        "role": "system",
        "content": "你是DeepSeek助手"
      },
      {
        "role": "user",
        "content": "你好，请用一句话介绍你自己"
      }
    ],
    "max_tokens": 100,
    "temperature": 0.7
  }' --max-time 30)

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP 状态码: $HTTP_CODE"
echo ""

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ DeepSeek Chat API 测试成功！${NC}"
    echo ""
    echo "响应内容："
    echo "$BODY" | jq '.'
    echo ""
    
    # 提取回复内容
    REPLY=$(echo "$BODY" | jq -r '.choices[0].message.content' 2>/dev/null || echo "无法解析回复")
    echo -e "${CYAN}AI 回复：${NC}"
    echo "$REPLY"
else
    echo -e "${RED}❌ DeepSeek Chat API 测试失败${NC}"
    echo ""
    echo "错误响应："
    echo "$BODY"
    echo ""
    
    # 解析错误信息
    ERROR_MSG=$(echo "$BODY" | jq -r '.error.message' 2>/dev/null || echo "未知错误")
    echo -e "${YELLOW}错误详情: $ERROR_MSG${NC}"
    
    if [ "$HTTP_CODE" = "401" ]; then
        echo ""
        echo -e "${RED}认证失败！请检查 API Key 是否正确。${NC}"
    elif [ "$HTTP_CODE" = "429" ]; then
        echo ""
        echo -e "${YELLOW}请求过于频繁，请稍后再试。${NC}"
    elif [ "$HTTP_CODE" = "000" ]; then
        echo ""
        echo -e "${RED}网络连接失败或超时！${NC}"
        echo "可能原因："
        echo "  1. 网络无法访问 DeepSeek API"
        echo "  2. 防火墙阻止了连接"
        echo "  3. API 服务暂时不可用"
    fi
    
    exit 1
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ 所有测试通过！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "下一步："
echo "  1. 运行完整的 Playwright 测试："
echo "     BASE_URL=http://192.168.0.200:8080 ./test-deepseek-chat.sh"
echo ""
echo "  2. 检查后端日志："
echo "     docker compose logs backend --tail=100 -f | grep -i deepseek"
echo ""
