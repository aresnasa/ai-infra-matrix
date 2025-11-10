#!/bin/bash

# 测试 SLURM 节点安装 API
# 使用方法: ./test-install-node-api.sh [node_name] [os_type]

BASE_URL="${BASE_URL:-http://localhost:8080}"
NODE_NAME="${1:-test-rocky01}"
OS_TYPE="${2:-rocky}"

echo "========================================="
echo "测试 SLURM 节点安装 API"
echo "========================================="
echo "BASE_URL: $BASE_URL"
echo "节点名称: $NODE_NAME"
echo "操作系统: $OS_TYPE"
echo ""

# 1. 登录获取 token
echo "1. 登录获取 JWT token..."
TOKEN=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取 token"
    exit 1
fi
echo "✅ Token: ${TOKEN:0:20}..."
echo ""

# 2. 测试单个节点安装
echo "2. 安装 SLURM 到节点: $NODE_NAME"
echo "请求参数:"
cat <<EOF
{
  "node_name": "$NODE_NAME",
  "os_type": "$OS_TYPE"
}
EOF
echo ""

RESPONSE=$(curl -s -X POST "$BASE_URL/api/slurm/nodes/install" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"node_name\": \"$NODE_NAME\",
    \"os_type\": \"$OS_TYPE\"
  }")

echo "响应:"
echo "$RESPONSE" | jq '.'
echo ""

# 检查是否成功
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
if [ "$SUCCESS" == "true" ]; then
    echo "✅ 节点安装成功"
    echo ""
    echo "安装日志:"
    echo "$RESPONSE" | jq -r '.logs'
else
    echo "❌ 节点安装失败"
    ERROR=$(echo "$RESPONSE" | jq -r '.error // .message')
    echo "错误信息: $ERROR"
    exit 1
fi

echo ""
echo "========================================="
echo "3. 验证节点状态"
echo "========================================="

# 等待几秒让服务完全启动
sleep 5

# 检查 SLURM 节点状态
echo "检查 sinfo 输出:"
docker exec ai-infra-slurm-master sinfo

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
