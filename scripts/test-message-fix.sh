#!/bin/bash

# 消息响应修复测试脚本
# 用途：验证消息不再错乱，每个问题都能得到正确对应的答案

set -e

echo "🔧 消息响应修复测试"
echo "===================="
echo ""

# 检查环境
if ! docker-compose ps | grep -q "backend.*Up"; then
    echo "❌ Backend 服务未运行"
    echo "请先启动服务: docker-compose up -d backend"
    exit 1
fi

echo "✅ Backend 服务正在运行"
echo ""

# 重启 backend 以应用代码更改
echo "🔄 重启 backend 服务以应用修复..."
docker-compose restart backend
echo "⏳ 等待服务就绪 (10秒)..."
sleep 10
echo ""

# 运行 Playwright 测试
echo "🧪 运行 Playwright E2E 测试..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

BASE_URL=http://192.168.0.200:8080 npx playwright test \
  test/e2e/specs/deepseek-simple-test.spec.js \
  --reporter=line

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 检查测试结果
if [ $? -eq 0 ]; then
    echo "✅ 所有测试通过！"
    echo ""
    echo "修复验证成功："
    echo "  ✓ 消息响应精确匹配"
    echo "  ✓ 缓存禁用确保实时性"
    echo "  ✓ 快速连续对话正常工作"
    echo ""
else
    echo "❌ 测试失败"
    echo ""
    echo "请检查："
    echo "  1. Backend 日志: docker-compose logs backend --tail=100"
    echo "  2. Redis 状态: docker-compose exec redis redis-cli PING"
    echo "  3. PostgreSQL 状态: docker-compose exec postgres pg_isready"
    echo ""
    exit 1
fi
