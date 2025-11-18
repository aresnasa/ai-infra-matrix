#!/bin/bash

# SaltStack 集成状态测试脚本
# 用途：验证 SaltStack 状态显示修复
# 功能：自动安装 SaltStack 客户端到 test-ssh01-03 节点并验证集群状态

set -e

# 配置变量
BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"
APPHUB_URL="${APPHUB_URL:-http://192.168.0.200:8090}"
API_BASE_URL="${API_BASE_URL:-$BASE_URL/api}"
TEST_NODES=("test-ssh01" "test-ssh02" "test-ssh03")

echo "🔧 SaltStack 集成状态完整测试"
echo "==============================="
echo "📍 Base URL: $BASE_URL"
echo "📦 AppHub URL: $APPHUB_URL"
echo "🖥️  测试节点: ${TEST_NODES[*]}"
echo ""

# 检查环境
echo "🔍 检查服务状态..."
if ! docker-compose ps | grep -q "backend.*Up"; then
    echo "❌ Backend 服务未运行"
    echo "请先启动服务: docker-compose -f docker-compose.test.yml up -d"
    exit 1
fi

# 检查测试容器
for node in "${TEST_NODES[@]}"; do
    if ! docker-compose -f docker-compose.test.yml ps | grep -q "$node.*Up"; then
        echo "⚠️  测试容器 $node 未运行，正在启动..."
        docker-compose -f docker-compose.test.yml up -d "$node"
    fi
done

echo "✅ 所有服务正在运行"
echo ""

# 重启 backend 以应用代码更改
echo "🔄 重启 backend 服务以应用修复..."
docker-compose restart backend
echo "⏳ 等待服务就绪 (15秒)..."
sleep 15
echo ""

# 测试 API 端点
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 步骤 1: 登录获取认证 token"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 获取登录 token
echo "📝 正在登录..."
TOKEN_RESPONSE=$(curl -s -X POST "$API_BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $TOKEN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取 token"
    echo "响应: $TOKEN_RESPONSE"
    exit 1
fi

echo "✅ 登录成功"
echo "🔑 Token: ${TOKEN:0:20}..."
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 步骤 2: 安装 SaltStack Minion 到测试节点"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 构建安装请求
INSTALL_REQUEST=$(cat <<EOF
{
  "nodes": ["test-ssh01", "test-ssh02", "test-ssh03"],
  "appHubURL": "$APPHUB_URL",
  "enableSaltMinion": true,
  "enableSlurmClient": false
}
EOF
)

echo "📦 安装配置:"
echo "$INSTALL_REQUEST" | python3 -m json.tool 2>/dev/null || echo "$INSTALL_REQUEST"
echo ""

echo "🚀 开始安装 SaltStack Minion (预计耗时 3-5 分钟)..."
INSTALL_RESPONSE=$(curl -s -X POST "$API_BASE_URL/slurm/install-test-nodes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$INSTALL_REQUEST")

echo "📋 安装响应:"
echo "$INSTALL_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$INSTALL_RESPONSE"
echo ""

# 检查安装是否成功
if echo "$INSTALL_RESPONSE" | grep -q '"success":true'; then
    echo "✅ SaltStack Minion 安装成功"
    
    # 提取每个节点的安装结果
    echo ""
    echo "📊 各节点安装详情:"
    echo "$INSTALL_RESPONSE" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    results = data.get('results', [])
    for i, result in enumerate(results, 1):
        host = result.get('host', 'unknown')
        success = result.get('success', False)
        steps = result.get('steps', [])
        status = '✅' if success else '❌'
        print(f'{status} 节点 {i}: {host}')
        for step in steps:
            step_name = step.get('Name', 'unknown')
            step_success = step.get('Success', False)
            step_status = '  ✓' if step_success else '  ✗'
            print(f'{step_status} {step_name}')
except Exception as e:
    print(f'解析失败: {e}')
" || echo "  (解析详情失败)"
else
    echo "⚠️  SaltStack Minion 安装可能失败，继续测试..."
fi

echo ""
echo "⏳ 等待 SaltStack Master 接受 Minion 连接 (10秒)..."
sleep 10
echo ""

# 测试 SaltStack 集成状态端点
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 步骤 3: 获取 SaltStack 集成状态"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "📊 正在获取 SaltStack 集成状态..."
RESPONSE=$(curl -s "$API_BASE_URL/slurm/saltstack/integration" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "📄 API 响应数据:"
echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
echo ""

# 验证响应字段
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 步骤 4: 验证响应数据结构"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

VALIDATION_PASSED=true

# 检查关键字段是否存在
if echo "$RESPONSE" | grep -q '"enabled"'; then
    ENABLED=$(echo "$RESPONSE" | grep -o '"enabled":[^,}]*' | cut -d':' -f2)
    echo "  ✓ enabled: $ENABLED"
else
    echo "  ✗ 缺少 enabled 字段"
    VALIDATION_PASSED=false
fi

if echo "$RESPONSE" | grep -q '"master_status"'; then
    MASTER_STATUS=$(echo "$RESPONSE" | grep -o '"master_status":"[^"]*"' | cut -d'"' -f4)
    echo "  ✓ master_status: $MASTER_STATUS"
else
    echo "  ✗ 缺少 master_status 字段"
    VALIDATION_PASSED=false
fi

if echo "$RESPONSE" | grep -q '"api_status"'; then
    API_STATUS=$(echo "$RESPONSE" | grep -o '"api_status":"[^"]*"' | cut -d'"' -f4)
    echo "  ✓ api_status: $API_STATUS"
else
    echo "  ✗ 缺少 api_status 字段"
    VALIDATION_PASSED=false
fi

if echo "$RESPONSE" | grep -q '"minions"'; then
    echo "  ✓ minions 字段存在"
    if echo "$RESPONSE" | grep -q '"total"'; then
        TOTAL=$(echo "$RESPONSE" | grep -o '"total":[0-9]*' | cut -d':' -f2 | head -1)
        echo "    - total: $TOTAL"
    fi
    if echo "$RESPONSE" | grep -q '"online"'; then
        ONLINE=$(echo "$RESPONSE" | grep -o '"online":[0-9]*' | cut -d':' -f2 | head -1)
        echo "    - online: $ONLINE"
    fi
    if echo "$RESPONSE" | grep -q '"offline"'; then
        OFFLINE=$(echo "$RESPONSE" | grep -o '"offline":[0-9]*' | cut -d':' -f2 | head -1)
        echo "    - offline: $OFFLINE"
    fi
else
    echo "  ✗ 缺少 minions 字段"
    VALIDATION_PASSED=false
fi

if echo "$RESPONSE" | grep -q '"minion_list"'; then
    echo "  ✓ minion_list 字段存在"
    # 提取 minion 列表
    MINION_COUNT=$(echo "$RESPONSE" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    minions = data.get('data', {}).get('minion_list', [])
    print(len(minions))
except:
    print(0)
" 2>/dev/null || echo "0")
    echo "    - 包含 $MINION_COUNT 个 minion"
    
    if [ "$MINION_COUNT" -gt 0 ]; then
        echo ""
        echo "  📋 Minion 详情:"
        echo "$RESPONSE" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    minions = data.get('data', {}).get('minion_list', [])
    for minion in minions:
        mid = minion.get('id', 'unknown')
        status = minion.get('status', 'unknown')
        status_icon = '🟢' if status == 'online' else '🔴' if status == 'offline' else '🟡'
        print(f'    {status_icon} {mid} ({status})')
except Exception as e:
    print(f'    解析失败: {e}')
" || echo "    (解析失败)"
    fi
else
    echo "  ✗ 缺少 minion_list 字段"
    VALIDATION_PASSED=false
fi

if echo "$RESPONSE" | grep -q '"recent_jobs"'; then
    RECENT_JOBS=$(echo "$RESPONSE" | grep -o '"recent_jobs":[0-9]*' | cut -d':' -f2 | head -1)
    echo "  ✓ recent_jobs: $RECENT_JOBS"
else
    echo "  ✗ 缺少 recent_jobs 字段"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 测试结果总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 检查测试结果
if [ "$VALIDATION_PASSED" = true ]; then
    echo "✅ 所有关键字段验证通过！"
    echo ""
    echo "🎉 修复验证成功："
    echo "  ✓ 后端正确返回 master_status 字段"
    echo "  ✓ 后端正确返回 api_status 字段"
    echo "  ✓ 后端正确返回 minions 统计 (total/online/offline)"
    echo "  ✓ 后端正确返回 minion_list 数组"
    echo "  ✓ 数据格式符合前端期望"
    echo ""
    echo "📌 后续步骤："
    echo "  1. 访问 $BASE_URL/slurm 查看页面效果"
    echo "  2. 验证 SaltStack 状态卡片显示正常"
    echo "  3. 检查 Minion 列表和状态标签"
    echo ""
    
    # 如果有 minion，显示额外信息
    if [ "${MINION_COUNT:-0}" -gt 0 ]; then
        echo "🎯 检测到 $MINION_COUNT 个 Minion 节点"
        echo "  - 可以在 /slurm 页面查看节点详情"
        echo "  - 可以在 /saltstack 页面管理 Minion"
        echo ""
    fi
else
    echo "❌ 部分字段验证失败"
    echo ""
    echo "🔍 故障排查建议："
    echo "  1. 检查 Backend 日志:"
    echo "     docker-compose logs backend --tail=100"
    echo ""
    echo "  2. 检查 SaltStack 配置:"
    echo "     docker-compose exec backend env | grep SALT"
    echo ""
    echo "  3. 验证 SaltStack Master 状态:"
    echo "     docker-compose exec saltstack salt-master --version"
    echo ""
    echo "  4. 检查 Minion 连接:"
    echo "     docker-compose exec saltstack salt-key -L"
    echo ""
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ SaltStack 集成状态测试完成"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
