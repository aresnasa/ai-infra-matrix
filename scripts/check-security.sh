#!/bin/bash

# 安全检查脚本 - 检查代码中是否存在硬编码的敏感信息
# 使用方法: ./check-security.sh

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 问题计数
ISSUES_FOUND=0

echo "============================================================"
echo "🔒 开始安全检查..."
echo "============================================================"
echo ""

# 检查 1: 硬编码的 API Key
echo "1. 检查硬编码的 API Key..."
API_KEY_PATTERN="sk-[a-zA-Z0-9]{30,}"
API_KEYS=$(grep -r -E "$API_KEY_PATTERN" . \
    --exclude-dir=node_modules \
    --exclude-dir=.git \
    --exclude-dir=test-results \
    --exclude="*.example" \
    --exclude="*.md" \
    --exclude="*.log" 2>/dev/null || true)

if [ -n "$API_KEY_ISSUES" ]; then
    echo -e "${RED}❌ 发现硬编码的 API Key:${NC}"
    echo "$API_KEY_ISSUES"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✅ 未发现硬编码的 API Key${NC}"
fi
echo ""

# 2. 检查硬编码的密码
echo "2. 检查硬编码的密码..."
PASSWORD_PATTERNS=(
    "password\s*=\s*['\"][^'\"]{6,}['\"]"
    "PASSWORD\s*=\s*['\"][^'\"]{6,}['\"]"
    "pwd\s*=\s*['\"][^'\"]{6,}['\"]"
)

for pattern in "${PASSWORD_PATTERNS[@]}"; do
    PWD_ISSUES=$(grep -rni --exclude-dir={node_modules,.git,test-results,test-screenshots} \
                 --exclude="*.{md,example,log,png,jpg,svg}" \
                 -E "$pattern" . 2>/dev/null || true)
    if [ -n "$PWD_ISSUES" ]; then
        echo -e "${RED}❌ 发现硬编码的密码:${NC}"
        echo "$PWD_ISSUES"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
        break
    fi
done

if [ $ISSUES_FOUND -eq 0 ] || [ -z "$PWD_ISSUES" ]; then
    echo -e "${GREEN}✅ 密码检查完成${NC}"
fi
echo ""

# 3. 检查 .gitignore 配置
echo "3. 检查 .gitignore 配置..."
REQUIRED_IGNORES=(".env.local" ".env.test")
MISSING_IGNORES=()

for ignore in "${REQUIRED_IGNORES[@]}"; do
    if ! grep -q "^$ignore$" .gitignore 2>/dev/null; then
        MISSING_IGNORES+=("$ignore")
    fi
done

if [ ${#MISSING_IGNORES[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠️  .gitignore 中缺少以下配置:${NC}"
    printf '%s\n' "${MISSING_IGNORES[@]}"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✅ .gitignore 配置正确${NC}"
fi
echo ""

# 4. 检查是否存在不应提交的配置文件
echo "4. 检查是否存在不应提交的配置文件..."
SENSITIVE_FILES=(".env.local" ".env.test")
FOUND_SENSITIVE=()

for file in "${SENSITIVE_FILES[@]}"; do
    if [ -f "$file" ] && git ls-files --error-unmatch "$file" 2>/dev/null; then
        FOUND_SENSITIVE+=("$file")
    fi
done

if [ ${#FOUND_SENSITIVE[@]} -gt 0 ]; then
    echo -e "${RED}❌ 发现已被 Git 跟踪的敏感文件:${NC}"
    printf '%s\n' "${FOUND_SENSITIVE[@]}"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✅ 敏感文件检查完成${NC}"
fi
echo ""

# 5. 检查环境变量使用
echo "5. 检查环境变量使用..."
ENV_ISSUES=$(grep -rn --exclude-dir={node_modules,.git,test-results,test-screenshots} \
             --exclude="*.{md,example,log,png,jpg,svg}" \
             -E "process\.env\.[A-Z_]+|os\.getenv" . 2>/dev/null | \
             grep -v "process.env.NODE_ENV" | \
             grep -v "process.env.BASE_URL" | \
             head -5 || true)

if [ -n "$ENV_ISSUES" ]; then
    echo -e "${YELLOW}ℹ️  发现环境变量使用 (前5个):${NC}"
    echo "$ENV_ISSUES" | head -5
fi
echo -e "${GREEN}✅ 环境变量检查完成${NC}"

# 总结
echo ""
echo "============================================================"
if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}🎉 安全检查通过！未发现问题。${NC}"
    exit 0
else
    echo -e "${RED}⚠️  发现 $ISSUES_FOUND 个安全问题，请修复后再提交。${NC}"
    exit 1
fi
echo "============================================================"
