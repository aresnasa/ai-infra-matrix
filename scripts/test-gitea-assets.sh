#!/usr/bin/env bash
# Gitea 静态资源配置验证脚本
# 用于验证 Gitea 静态资源路径配置是否正确

echo "=================================="
echo "Gitea 静态资源配置验证脚本"
echo "=================================="
echo ""

# 定义颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 验证结果
PASSED=0
FAILED=0

echo "1. 检查 .env 文件配置"
echo "=================================="

# 检查 STATIC_URL_PREFIX
STATIC_URL_PREFIX=$(grep "^STATIC_URL_PREFIX=" .env 2>/dev/null | cut -d'=' -f2)
ROOT_URL=$(grep "^ROOT_URL=" .env 2>/dev/null | cut -d'=' -f2)
SUBURL=$(grep "^SUBURL=" .env 2>/dev/null | cut -d'=' -f2)

echo "STATIC_URL_PREFIX = $STATIC_URL_PREFIX"
echo "ROOT_URL = $ROOT_URL"
echo "SUBURL = $SUBURL"
echo ""

# 验证 STATIC_URL_PREFIX
if [ "$STATIC_URL_PREFIX" = "/gitea" ]; then
    echo -e "${GREEN}✓${NC} STATIC_URL_PREFIX 配置正确: /gitea"
    ((PASSED++))
elif [ "$STATIC_URL_PREFIX" = "/assets" ]; then
    echo -e "${RED}✗${NC} STATIC_URL_PREFIX 配置错误: /assets (会导致重复路径 /assets/assets/)"
    echo "  应该设置为: /gitea"
    ((FAILED++))
else
    echo -e "${YELLOW}⚠${NC} STATIC_URL_PREFIX 配置异常: $STATIC_URL_PREFIX"
    echo "  推荐设置为: /gitea"
    ((FAILED++))
fi
echo ""

# 验证 SUBURL 和 STATIC_URL_PREFIX 是否一致
if [ "$STATIC_URL_PREFIX" = "$SUBURL" ]; then
    echo -e "${GREEN}✓${NC} STATIC_URL_PREFIX 与 SUBURL 一致"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} STATIC_URL_PREFIX 与 SUBURL 不一致"
    echo "  SUBURL = $SUBURL"
    echo "  STATIC_URL_PREFIX = $STATIC_URL_PREFIX"
    echo "  建议将 STATIC_URL_PREFIX 设置为与 SUBURL 相同的值"
    ((FAILED++))
fi
echo ""

echo "2. 检查 Gitea 容器配置（如果运行中）"
echo "=================================="

# 检查 Gitea 容器是否运行
if docker compose ps gitea 2>/dev/null | grep -q "Up"; then
    echo -e "${GREEN}✓${NC} Gitea 容器正在运行"
    
    # 尝试读取容器内的 app.ini 配置
    echo ""
    echo "检查容器内 app.ini 配置..."
    
    CONTAINER_STATIC_URL=$(docker compose exec -T gitea grep "^STATIC_URL_PREFIX" /data/gitea/conf/app.ini 2>/dev/null | cut -d'=' -f2 | xargs)
    CONTAINER_ROOT_URL=$(docker compose exec -T gitea grep "^ROOT_URL" /data/gitea/conf/app.ini 2>/dev/null | cut -d'=' -f2 | xargs)
    CONTAINER_SUBURL=$(docker compose exec -T gitea grep "^SUBURL" /data/gitea/conf/app.ini 2>/dev/null | cut -d'=' -f2 | xargs)
    
    if [ -n "$CONTAINER_STATIC_URL" ]; then
        echo "容器内 STATIC_URL_PREFIX = $CONTAINER_STATIC_URL"
        
        if [ "$CONTAINER_STATIC_URL" = "/gitea" ]; then
            echo -e "${GREEN}✓${NC} 容器内 STATIC_URL_PREFIX 配置正确"
            ((PASSED++))
        else
            echo -e "${RED}✗${NC} 容器内 STATIC_URL_PREFIX 配置错误: $CONTAINER_STATIC_URL"
            echo "  建议重启 Gitea 容器以应用新配置"
            ((FAILED++))
        fi
    else
        echo -e "${YELLOW}⚠${NC} 无法读取容器内 STATIC_URL_PREFIX 配置"
    fi
    
    if [ -n "$CONTAINER_ROOT_URL" ]; then
        echo "容器内 ROOT_URL = $CONTAINER_ROOT_URL"
    fi
    
    if [ -n "$CONTAINER_SUBURL" ]; then
        echo "容器内 SUBURL = $CONTAINER_SUBURL"
    fi
else
    echo -e "${YELLOW}⚠${NC} Gitea 容器未运行，跳过容器配置检查"
fi
echo ""

echo "3. Gitea 静态资源路径说明"
echo "=================================="
echo "正确的配置方式："
echo "  SUBURL=/gitea"
echo "  STATIC_URL_PREFIX=/gitea"
echo ""
echo "这样 Gitea 会生成如下静态资源路径："
echo "  → /gitea/assets/js/index.js"
echo "  → /gitea/assets/css/index.css"
echo "  → /gitea/assets/img/logo.svg"
echo ""
echo "错误的配置方式："
echo "  SUBURL=/gitea"
echo "  STATIC_URL_PREFIX=/assets  ❌"
echo ""
echo "这会导致 Gitea 生成错误的路径："
echo "  → /assets/assets/js/index.js  ❌ (重复的 /assets/)"
echo ""

echo "=================================="
echo "验证总结"
echo "=================================="
echo -e "通过: ${GREEN}$PASSED${NC}"
echo -e "失败: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ Gitea 静态资源配置正确！${NC}"
    echo ""
    echo "如果仍然遇到资源加载问题，请："
    echo "1. 重启 Gitea 容器: docker compose restart gitea"
    echo "2. 清理浏览器缓存"
    echo "3. 检查 Nginx 日志: docker compose logs nginx | grep gitea"
    exit 0
else
    echo -e "${RED}✗ Gitea 静态资源配置有问题${NC}"
    echo ""
    echo "修复步骤："
    echo "1. 修改 .env 文件，将 STATIC_URL_PREFIX=/assets 改为 STATIC_URL_PREFIX=/gitea"
    echo "2. 重启 Gitea 容器: docker compose restart gitea"
    echo "3. 清理浏览器缓存并重新访问"
    echo ""
    echo "或者运行自动修复："
    echo "  sed -i 's|^STATIC_URL_PREFIX=.*|STATIC_URL_PREFIX=/gitea|' .env"
    echo "  docker compose restart gitea"
    exit 1
fi
