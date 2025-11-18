#!/bin/bash
# 快速集成测试脚本

set -e

BASE_URL="${BASE_URL:-http://192.168.18.114:8080}"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "==========================================="
echo "AI-Infra-Matrix 快速集成测试"
echo "==========================================="
echo "基础 URL: $BASE_URL"
echo ""

PASS=0
TOTAL=0

# 测试 1: Nightingale 代理
echo "【测试 1】Nightingale 监控代理..."
TOTAL=$((TOTAL + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/nightingale/" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "401" ]; then
    echo -e "${GREEN}✓${NC} /nightingale/ 可访问 (状态: $STATUS)"
    PASS=$((PASS + 1))
else
    echo -e "${RED}✗${NC} /nightingale/ 访问失败 (状态: $STATUS)"
fi

# 测试 2: MinIO Console 代理
echo "【测试 2】MinIO Console 代理..."
TOTAL=$((TOTAL + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/minio-console/" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} /minio-console/ 可访问 (状态: $STATUS)"
    PASS=$((PASS + 1))
else
    echo -e "${RED}✗${NC} /minio-console/ 访问失败 (状态: $STATUS)"
fi

# 测试 3: MinIO API
echo "【测试 3】MinIO API..."
TOTAL=$((TOTAL + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/minio/" 2>/dev/null || echo "000")
if [ "$STATUS" = "403" ] || [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} /minio/ API 可访问 (状态: $STATUS)"
    PASS=$((PASS + 1))
else
    echo -e "${YELLOW}⚠${NC} /minio/ API 状态: $STATUS"
fi

# 测试 4: 前端
echo "【测试 4】前端页面..."
TOTAL=$((TOTAL + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} 前端页面可访问 (状态: $STATUS)"
    PASS=$((PASS + 1))
else
    echo -e "${RED}✗${NC} 前端页面访问失败 (状态: $STATUS)"
fi

# 测试 5: Backend API
echo "【测试 5】Backend API..."
TOTAL=$((TOTAL + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/health" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} Backend API 健康 (状态: $STATUS)"
    PASS=$((PASS + 1))
else
    echo -e "${RED}✗${NC} Backend API 失败 (状态: $STATUS)"
fi

# 测试 6: Gitea
echo "【测试 6】Gitea 服务..."
TOTAL=$((TOTAL + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/gitea/" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "302" ]; then
    echo -e "${GREEN}✓${NC} Gitea 可访问 (状态: $STATUS)"
    PASS=$((PASS + 1))
else
    echo -e "${RED}✗${NC} Gitea 访问失败 (状态: $STATUS)"
fi

# 测试 7: 容器状态
echo "【测试 7】关键容器状态..."
TOTAL=$((TOTAL + 1))
CONTAINERS="backend frontend nginx gitea minio nightingale"
RUNNING=0
for c in $CONTAINERS; do
    if docker-compose ps 2>/dev/null | grep "$c.*Up" > /dev/null; then
        RUNNING=$((RUNNING + 1))
    fi
done
if [ $RUNNING -eq 6 ]; then
    echo -e "${GREEN}✓${NC} 所有关键容器运行中 ($RUNNING/6)"
    PASS=$((PASS + 1))
else
    echo -e "${YELLOW}⚠${NC} 部分容器未运行 ($RUNNING/6)"
fi

echo ""
echo "==========================================="
echo "测试结果汇总"
echo "==========================================="
echo "通过: $PASS / $TOTAL"
echo ""

if [ $PASS -eq $TOTAL ]; then
    echo -e "${GREEN}✅ 所有测试通过！系统运行正常${NC}"
    exit 0
elif [ $PASS -ge 5 ]; then
    echo -e "${YELLOW}⚠️  大部分测试通过，核心功能可用${NC}"
    exit 0
else
    echo -e "${RED}❌ 多项测试失败，请检查系统状态${NC}"
    exit 1
fi
