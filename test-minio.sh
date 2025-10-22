#!/bin/bash
# MinIO 对象存储快速测试脚本

set -e

echo "==========================================="
echo "MinIO 对象存储集成测试"
echo "==========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 基础 URL
BASE_URL="${BASE_URL:-http://192.168.18.114:8080}"

echo "测试环境: $BASE_URL"
echo ""

# 测试 1: MinIO Console 代理路径
echo "【测试 1】检查 /minio-console/ 代理路径..."
MINIO_CONSOLE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -L "$BASE_URL/minio-console/" 2>/dev/null || echo "000")

if [ "$MINIO_CONSOLE_STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} /minio-console/ 可访问 (状态: $MINIO_CONSOLE_STATUS)"
elif [ "$MINIO_CONSOLE_STATUS" = "302" ] || [ "$MINIO_CONSOLE_STATUS" = "301" ]; then
    echo -e "${YELLOW}⚠${NC} /minio-console/ 重定向 (状态: $MINIO_CONSOLE_STATUS)"
else
    echo -e "${RED}✗${NC} /minio-console/ 访问失败 (状态: $MINIO_CONSOLE_STATUS)"
fi
echo ""

# 测试 2: MinIO API 路径
echo "【测试 2】检查 /minio/ API 路径..."
MINIO_API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/minio/" 2>/dev/null || echo "000")

if [ "$MINIO_API_STATUS" = "403" ] || [ "$MINIO_API_STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} /minio/ API 可访问 (状态: $MINIO_API_STATUS)"
else
    echo -e "${YELLOW}⚠${NC} /minio/ API 状态: $MINIO_API_STATUS"
fi
echo ""

# 测试 3: MinIO 健康检查
echo "【测试 3】检查 MinIO 健康状态..."
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/minio/health" 2>/dev/null || echo "000")

if [ "$HEALTH_STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} MinIO 健康检查通过 (状态: $HEALTH_STATUS)"
else
    echo -e "${RED}✗${NC} MinIO 健康检查失败 (状态: $HEALTH_STATUS)"
fi
echo ""

# 测试 4: 检查 MinIO 容器
echo "【测试 4】检查 MinIO 容器状态..."
if docker-compose ps 2>/dev/null | grep -q "minio.*Up"; then
    echo -e "${GREEN}✓${NC} MinIO 容器正在运行"
    MINIO_CONTAINER=$(docker-compose ps | grep minio | awk '{print $1}')
    echo "   容器名: $MINIO_CONTAINER"
else
    echo -e "${RED}✗${NC} MinIO 容器未运行"
fi
echo ""

# 测试 5: 检查 Nginx 配置
echo "【测试 5】检查 Nginx 配置..."
if docker-compose exec -T nginx nginx -t 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Nginx 配置有效"
else
    echo -e "${RED}✗${NC} Nginx 配置错误"
fi
echo ""

# 测试 6: 检查响应头
echo "【测试 6】检查 /minio-console/ 响应头..."
echo "执行: curl -I $BASE_URL/minio-console/"
HEADERS=$(curl -sI "$BASE_URL/minio-console/" 2>/dev/null)

X_FRAME=$(echo "$HEADERS" | grep -i "x-frame-options" | cut -d: -f2- | xargs || echo "未设置")
CSP=$(echo "$HEADERS" | grep -i "content-security-policy" | cut -d: -f2- | xargs || echo "未设置")

echo "   X-Frame-Options: $X_FRAME"
echo "   Content-Security-Policy: $CSP"

if [[ "$X_FRAME" == *"SAMEORIGIN"* ]] || [[ "$X_FRAME" == "未设置" ]]; then
    echo -e "${GREEN}✓${NC} X-Frame-Options 允许 iframe 嵌入"
else
    echo -e "${RED}✗${NC} X-Frame-Options 可能阻止 iframe 嵌入"
fi
echo ""

# 汇总
echo "==========================================="
echo "测试总结"
echo "==========================================="

PASS_COUNT=0
TOTAL_COUNT=6

[ "$MINIO_CONSOLE_STATUS" = "200" ] && PASS_COUNT=$((PASS_COUNT + 1))
[ "$MINIO_API_STATUS" = "403" ] || [ "$MINIO_API_STATUS" = "200" ] && PASS_COUNT=$((PASS_COUNT + 1))
[ "$HEALTH_STATUS" = "200" ] && PASS_COUNT=$((PASS_COUNT + 1))
docker-compose ps 2>/dev/null | grep -q "minio.*Up" && PASS_COUNT=$((PASS_COUNT + 1))
docker-compose exec -T nginx nginx -t &>/dev/null && PASS_COUNT=$((PASS_COUNT + 1))
[[ "$X_FRAME" == *"SAMEORIGIN"* ]] || [[ "$X_FRAME" == "未设置" ]] && PASS_COUNT=$((PASS_COUNT + 1))

echo "通过: $PASS_COUNT / $TOTAL_COUNT"
echo ""

if [ $PASS_COUNT -eq $TOTAL_COUNT ]; then
    echo -e "${GREEN}✅ 所有测试通过！MinIO 集成正常${NC}"
    exit 0
elif [ $PASS_COUNT -ge 4 ]; then
    echo -e "${YELLOW}⚠️  部分测试失败，但核心功能可用${NC}"
    exit 0
else
    echo -e "${RED}❌ 多项测试失败，MinIO 集成存在问题${NC}"
    exit 1
fi
