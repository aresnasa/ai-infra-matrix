#!/bin/bash
#
# SLURM REST API 扩缩容功能测试脚本
# 测试新的基于REST API的扩缩容端点
#

set -e

# 配置参数
BACKEND_URL="http://192.168.0.200:8080"
TEST_NODE_HOST="test-compute-01"

echo "=========================================="
echo "  SLURM REST API 扩缩容测试"
echo "  Backend URL: ${BACKEND_URL}"
echo "=========================================="
echo ""

# 1. 获取认证Token (模拟登录)
echo "[1/6] 获取认证Token..."
LOGIN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "${BACKEND_URL}/api/auth/login" 2>/dev/null)

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token // .token // empty' 2>/dev/null)
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "  ⚠️  无法获取Token，尝试无认证方式"
    AUTH_HEADER=""
else
    echo "  ✅ Token获取成功: ${TOKEN:0:20}..."
    AUTH_HEADER="Authorization: Bearer $TOKEN"
fi

# 2. 检查SLURM服务状态
echo ""
echo "[2/6] 检查SLURM服务状态..."
SUMMARY_RESPONSE=$(curl -s -H "$AUTH_HEADER" "${BACKEND_URL}/api/slurm/summary" 2>/dev/null)
if echo "$SUMMARY_RESPONSE" | jq . &>/dev/null; then
    echo "  ✅ SLURM服务响应正常"
    echo "$SUMMARY_RESPONSE" | jq -C '.' | head -10 | sed 's/^/    /'
else
    echo "  ⚠️  SLURM服务响应异常"
    echo "$SUMMARY_RESPONSE" | head -3 | sed 's/^/    /'
fi

# 3. 测试配置重新加载
echo ""
echo "[3/6] 测试SLURM配置重新加载..."
RELOAD_RESPONSE=$(curl -s -X POST -H "$AUTH_HEADER" \
    "${BACKEND_URL}/api/slurm/reload-config" 2>/dev/null)

if echo "$RELOAD_RESPONSE" | jq -r '.success' 2>/dev/null | grep -q true; then
    echo "  ✅ 配置重新加载成功"
    echo "$RELOAD_RESPONSE" | jq -C '.' | sed 's/^/    /'
else
    echo "  ⚠️  配置重新加载失败"
    echo "$RELOAD_RESPONSE" | head -3 | sed 's/^/    /'
fi

# 4. 测试REST API扩容
echo ""
echo "[4/6] 测试REST API扩容..."
SCALE_UP_DATA=$(cat <<EOF
{
    "nodes": [
        {
            "host": "${TEST_NODE_HOST}",
            "port": 22,
            "user": "root",
            "password": "test123",
            "minion_id": "${TEST_NODE_HOST}"
        }
    ]
}
EOF
)

SCALE_UP_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "$SCALE_UP_DATA" \
    "${BACKEND_URL}/api/slurm/scaling/scale-up-api" 2>/dev/null)

if echo "$SCALE_UP_RESPONSE" | jq -r '.data.Success' 2>/dev/null | grep -q true; then
    echo "  ✅ REST API扩容请求成功"
    echo "$SCALE_UP_RESPONSE" | jq -C '.data' | sed 's/^/    /'
else
    echo "  ⚠️  REST API扩容请求失败"
    echo "$SCALE_UP_RESPONSE" | head -5 | sed 's/^/    /'
fi

# 5. 检查节点状态
echo ""
echo "[5/6] 检查节点状态..."
NODES_RESPONSE=$(curl -s -H "$AUTH_HEADER" "${BACKEND_URL}/api/slurm/nodes" 2>/dev/null)
if echo "$NODES_RESPONSE" | jq . &>/dev/null; then
    echo "  ✅ 节点列表获取成功"
    echo "$NODES_RESPONSE" | jq -C '.data[] | select(.name | contains("'$TEST_NODE_HOST'")) // empty' 2>/dev/null | sed 's/^/    /' || \
    echo "    未找到测试节点 $TEST_NODE_HOST"
else
    echo "  ⚠️  节点列表获取失败"
    echo "$NODES_RESPONSE" | head -3 | sed 's/^/    /'
fi

# 6. 测试REST API缩容
echo ""
echo "[6/6] 测试REST API缩容..."
SCALE_DOWN_DATA=$(cat <<EOF
{
    "node_ids": ["${TEST_NODE_HOST}"]
}
EOF
)

SCALE_DOWN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "$SCALE_DOWN_DATA" \
    "${BACKEND_URL}/api/slurm/scaling/scale-down-api" 2>/dev/null)

if echo "$SCALE_DOWN_RESPONSE" | jq -r '.data.Success' 2>/dev/null | grep -q true; then
    echo "  ✅ REST API缩容请求成功"
    echo "$SCALE_DOWN_RESPONSE" | jq -C '.data' | sed 's/^/    /'
else
    echo "  ⚠️  REST API缩容请求失败"
    echo "$SCALE_DOWN_RESPONSE" | head -5 | sed 's/^/    /'
fi

echo ""
echo "=========================================="
echo "  测试完成"
echo "=========================================="
echo ""
echo "📝 手动测试命令:"
echo "  # 获取Token"
echo "  TOKEN=\$(curl -s -X POST -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"admin123\"}' \\"
echo "    '${BACKEND_URL}/api/auth/login' | jq -r '.data.token')"
echo ""
echo "  # 测试扩容"
echo "  curl -X POST -H 'Content-Type: application/json' \\"
echo "    -H \"Authorization: Bearer \$TOKEN\" \\"
echo "    -d '{\"nodes\":[{\"host\":\"test-node\",\"port\":22,\"user\":\"root\",\"password\":\"test\"}]}' \\"
echo "    '${BACKEND_URL}/api/slurm/scaling/scale-up-api'"
echo ""