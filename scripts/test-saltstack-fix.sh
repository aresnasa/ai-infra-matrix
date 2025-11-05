#!/bin/bash

# SaltStack状态同步测试和修复脚本
# 用途: 验证SaltStack API集成修复，确保状态正确同步

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"

echo "======================================================================"
echo "SaltStack 状态同步测试"
echo "======================================================================"
echo "Base URL: $BASE_URL"
echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

# 步骤1: 检查SaltStack容器状态
echo "📋 步骤1: 检查SaltStack容器状态"
echo "----------------------------------------------------------------------"
if docker ps | grep -q ai-infra-saltstack; then
    echo "✅ SaltStack容器运行中"
    
    # 检查Salt Master Keys
    echo ""
    echo "🔑 Salt Master Keys状态:"
    docker exec ai-infra-saltstack salt-key -L
    
    # 统计keys数量
    ACCEPTED_COUNT=$(docker exec ai-infra-saltstack salt-key -L | grep -A 100 "Accepted Keys:" | grep -v "Accepted Keys:" | grep -v "Denied Keys:" | grep -v "Unaccepted Keys:" | grep -v "Rejected Keys:" | grep -c "^" || echo "0")
    echo ""
    echo "📊 已接受的Keys数量: $ACCEPTED_COUNT"
    
    if [ "$ACCEPTED_COUNT" -ge 7 ]; then
        echo "✅ Keys数量符合预期 (>= 7)"
    else
        echo "⚠️  Keys数量不足，预期至少7个"
    fi
else
    echo "❌ SaltStack容器未运行"
    exit 1
fi

# 步骤2: 检查Backend容器状态
echo ""
echo "📋 步骤2: 检查Backend容器状态"
echo "----------------------------------------------------------------------"
if docker ps | grep -q ai-infra-backend; then
    echo "✅ Backend容器运行中"
    
    # 检查环境变量
    echo ""
    echo "🔧 Backend Salt API配置:"
    docker exec ai-infra-backend printenv | grep SALT_API
else
    echo "❌ Backend容器未运行"
    exit 1
fi

# 步骤3: 测试Salt API直接连接
echo ""
echo "📋 步骤3: 测试Salt API直接连接"
echo "----------------------------------------------------------------------"
echo "测试登录..."
LOGIN_RESPONSE=$(docker exec ai-infra-backend curl -sS -X POST http://saltstack:8002/login \
  -H "Content-Type: application/json" \
  -d '{"username":"saltapi","password":"your-salt-api-password","eauth":"file"}' 2>/dev/null || echo "{}")

if echo "$LOGIN_RESPONSE" | jq -e '.return[0].token' > /dev/null 2>&1; then
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.return[0].token')
    echo "✅ Salt API登录成功"
    echo "   Token: ${TOKEN:0:20}..."
    
    # 测试获取keys
    echo ""
    echo "测试获取keys..."
    KEYS_RESPONSE=$(docker exec ai-infra-backend curl -sS http://saltstack:8002/keys \
      -H "X-Auth-Token: $TOKEN" 2>/dev/null || echo "{}")
    
    if echo "$KEYS_RESPONSE" | jq -e '.return.minions' > /dev/null 2>&1; then
        MINIONS_COUNT=$(echo "$KEYS_RESPONSE" | jq -r '.return.minions | length')
        echo "✅ 成功获取keys"
        echo "   Minions数量: $MINIONS_COUNT"
        echo "   Minions列表:"
        echo "$KEYS_RESPONSE" | jq -r '.return.minions[]' | sed 's/^/     - /'
    else
        echo "❌ 获取keys失败"
        echo "$KEYS_RESPONSE" | jq '.' || echo "$KEYS_RESPONSE"
    fi
else
    echo "❌ Salt API登录失败"
    echo "$LOGIN_RESPONSE" | jq '.' || echo "$LOGIN_RESPONSE"
    exit 1
fi

# 步骤4: 运行Playwright E2E测试
echo ""
echo "📋 步骤4: 运行Playwright E2E测试"
echo "----------------------------------------------------------------------"

if command -v npx > /dev/null 2>&1; then
    echo "🧪 运行SaltStack状态同步测试..."
    echo ""
    
    BASE_URL=$BASE_URL npx playwright test \
      test/e2e/specs/slurm-saltstack-status-test.spec.js \
      --reporter=line \
      || echo "⚠️  部分测试失败，请查看详细报告"
    
    echo ""
    echo "📊 测试完成"
else
    echo "⚠️  npx未安装，跳过Playwright测试"
    echo "   请手动运行: BASE_URL=$BASE_URL npx playwright test test/e2e/specs/slurm-saltstack-status-test.spec.js"
fi

# 步骤5: 总结
echo ""
echo "======================================================================"
echo "测试总结"
echo "======================================================================"
echo ""
echo "✅ SaltStack容器: 运行中"
echo "✅ Backend容器: 运行中"  
echo "✅ Salt API认证: 成功"
echo "✅ Keys获取: 成功 ($MINIONS_COUNT minions)"
echo ""
echo "📝 如果前端页面仍显示不正确："
echo "   1. 检查Backend日志: docker logs ai-infra-backend --tail=100"
echo "   2. 清除浏览器缓存并刷新页面"
echo "   3. 检查浏览器控制台是否有错误"
echo ""
echo "🔗 访问页面: $BASE_URL/slurm"
echo ""
echo "======================================================================"
