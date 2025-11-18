#!/bin/bash
# ====================================================================
# 环境变量归一化验证脚本
# ====================================================================
# 功能: 检查项目是否正确归一化所有环境变量到根目录 .env
# 作者: AI Infrastructure Team
# 版本: v1.0.0
# ====================================================================

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}环境变量归一化验证工具${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 计数器
ISSUES_FOUND=0
WARNINGS_FOUND=0

# ====================================================================
# 1. 检查 src/ 下是否还有 .env 文件
# ====================================================================
echo -e "${BLUE}[1/6] 检查 src/ 目录下的 .env 文件...${NC}"
ENV_FILES=$(find src -name ".env" -type f 2>/dev/null || true)
if [ -n "$ENV_FILES" ]; then
    echo -e "${RED}✗ 发现组件级 .env 文件（应删除）:${NC}"
    echo "$ENV_FILES" | while read -r file; do
        echo -e "  ${RED}- $file${NC}"
        ((ISSUES_FOUND++))
    done
else
    echo -e "${GREEN}✓ 无组件级 .env 文件${NC}"
fi
echo ""

# ====================================================================
# 2. 检查 Dockerfile 是否有 COPY .env
# ====================================================================
echo -e "${BLUE}[2/6] 检查 Dockerfile 中的 .env 复制操作...${NC}"
DOCKERFILE_COPY=$(grep -rn "COPY.*\.env" src/*/Dockerfile 2>/dev/null | grep -v "^#" || true)
if [ -n "$DOCKERFILE_COPY" ]; then
    echo -e "${RED}✗ 发现 Dockerfile 复制 .env 文件:${NC}"
    echo "$DOCKERFILE_COPY" | while read -r line; do
        echo -e "  ${RED}- $line${NC}"
        ((ISSUES_FOUND++))
    done
else
    echo -e "${GREEN}✓ 所有 Dockerfile 不复制 .env 文件${NC}"
fi
echo ""

# ====================================================================
# 3. 检查 build.sh 是否创建组件级 .env
# ====================================================================
echo -e "${BLUE}[3/6] 检查 build.sh 是否创建组件级 .env...${NC}"
BUILD_CREATE_ENV=$(grep -n "src/.*\.env" build.sh 2>/dev/null | grep -v "^#" | grep -v "读取" | grep -v "不再" || true)
if [ -n "$BUILD_CREATE_ENV" ]; then
    echo -e "${YELLOW}⚠ build.sh 中可能存在组件级 .env 引用:${NC}"
    echo "$BUILD_CREATE_ENV" | while read -r line; do
        echo -e "  ${YELLOW}- $line${NC}"
        ((WARNINGS_FOUND++))
    done
else
    echo -e "${GREEN}✓ build.sh 不创建组件级 .env${NC}"
fi
echo ""

# ====================================================================
# 4. 检查 docker-compose.yml 配置
# ====================================================================
echo -e "${BLUE}[4/6] 检查 docker-compose.yml 配置...${NC}"

# 检查根级 env_file
ROOT_ENV_FILE=$(grep -c "env_file:" docker-compose.yml 2>/dev/null || echo "0")
if [ "$ROOT_ENV_FILE" -gt 0 ]; then
    echo -e "${GREEN}✓ docker-compose.yml 配置了 env_file${NC}"
    
    # 显示哪些服务使用了 env_file
    SERVICES_WITH_ENV=$(awk '/^  [a-z-]+:$/{ service=$1 } /env_file:/{ if(service) print "  - " substr(service, 1, length(service)-1) }' docker-compose.yml)
    if [ -n "$SERVICES_WITH_ENV" ]; then
        echo -e "${BLUE}  使用 env_file 的服务:${NC}"
        echo "$SERVICES_WITH_ENV"
    fi
else
    echo -e "${YELLOW}⚠ docker-compose.yml 未配置 env_file${NC}"
    ((WARNINGS_FOUND++))
fi
echo ""

# ====================================================================
# 5. 检查必要的环境变量示例文件
# ====================================================================
echo -e "${BLUE}[5/6] 检查环境变量示例文件...${NC}"

if [ -f ".env.example" ]; then
    echo -e "${GREEN}✓ .env.example 存在${NC}"
    
    # 检查关键配置项
    CRITICAL_VARS=(
        "POSTGRES_PASSWORD"
        "JUPYTERHUB_ADMIN_USERS"
        "SALTSTACK_MASTER_URL"
        "SALT_API_PORT"
        "BACKEND_PORT"
        "FRONTEND_PORT"
        "REACT_APP_API_URL"
    )
    
    for var in "${CRITICAL_VARS[@]}"; do
        if grep -q "^${var}=" .env.example; then
            echo -e "  ${GREEN}✓ $var${NC}"
        else
            echo -e "  ${YELLOW}⚠ $var 未配置${NC}"
            ((WARNINGS_FOUND++))
        fi
    done
else
    echo -e "${RED}✗ .env.example 不存在${NC}"
    ((ISSUES_FOUND++))
fi
echo ""

# ====================================================================
# 6. 检查废弃的 .env.example 文件
# ====================================================================
echo -e "${BLUE}[6/6] 检查组件级 .env.example 废弃标记...${NC}"

COMPONENT_ENV_EXAMPLES=$(find src -name ".env.example" -type f 2>/dev/null || true)
if [ -n "$COMPONENT_ENV_EXAMPLES" ]; then
    echo -e "${YELLOW}⚠ 发现组件级 .env.example 文件:${NC}"
    echo "$COMPONENT_ENV_EXAMPLES" | while read -r file; do
        # 检查是否有废弃警告
        if grep -q "废弃\|deprecated\|DO NOT USE" "$file" 2>/dev/null; then
            echo -e "  ${GREEN}✓ $file (已标记废弃)${NC}"
        else
            echo -e "  ${YELLOW}⚠ $file (未标记废弃)${NC}"
            ((WARNINGS_FOUND++))
        fi
    done
else
    echo -e "${GREEN}✓ 无组件级 .env.example 文件${NC}"
fi
echo ""

# ====================================================================
# 总结报告
# ====================================================================
echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}验证报告${NC}"
echo -e "${BLUE}======================================${NC}"

if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓ 严重问题: 0${NC}"
else
    echo -e "${RED}✗ 严重问题: $ISSUES_FOUND${NC}"
fi

if [ $WARNINGS_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓ 警告: 0${NC}"
else
    echo -e "${YELLOW}⚠ 警告: $WARNINGS_FOUND${NC}"
fi

echo ""

# ====================================================================
# 退出状态
# ====================================================================
if [ $ISSUES_FOUND -gt 0 ]; then
    echo -e "${RED}环境变量归一化未完成，存在严重问题！${NC}"
    echo -e "${YELLOW}建议：${NC}"
    echo -e "  1. 删除所有 src/*/\.env 文件"
    echo -e "  2. 移除 Dockerfile 中的 COPY .env 操作"
    echo -e "  3. 确保 docker-compose.yml 使用 env_file: - .env"
    echo -e "  4. 所有配置统一在项目根目录 .env 文件中管理"
    exit 1
elif [ $WARNINGS_FOUND -gt 0 ]; then
    echo -e "${YELLOW}环境变量归一化基本完成，但存在警告项${NC}"
    echo -e "${BLUE}建议查看上述警告并根据需要修复${NC}"
    exit 0
else
    echo -e "${GREEN}✓ 环境变量归一化验证通过！${NC}"
    echo -e "${GREEN}所有环境变量已正确归一化到项目根目录 .env${NC}"
    exit 0
fi
