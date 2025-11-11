#!/bin/bash
# 测试多集群管理功能

set -e

BASE_URL="http://192.168.3.91:8082"
API_URL="$BASE_URL/api"

echo "====== 多集群管理功能测试 ======"
echo ""

# 1. 登录获取 token
echo "1. 登录系统..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo "❌ 登录失败"
  echo "$LOGIN_RESPONSE" | jq '.'
  exit 1
fi

echo "✅ 登录成功，token: ${TOKEN:0:20}..."
echo ""

# 2. 获取现有集群列表
echo "2. 获取现有集群列表..."
CLUSTERS=$(curl -s -X GET "$API_URL/slurm/clusters" \
  -H "Authorization: Bearer $TOKEN")

echo "$CLUSTERS" | jq '.'
echo ""

# 3. 测试连接外部集群 API（使用当前的 slurm-master 作为"外部"集群测试）
echo "3. 测试连接外部集群..."
CONNECT_RESPONSE=$(curl -s -X POST "$API_URL/slurm/clusters/connect" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试外部集群",
    "description": "通过SSH连接的测试集群",
    "master_host": "slurm-master",
    "master_ssh": {
      "host": "slurm-master",
      "port": 22,
      "username": "root",
      "auth_type": "password",
      "password": "aiinfra2024"
    },
    "config": {
      "max_nodes": 10
    }
  }')

echo "$CONNECT_RESPONSE" | jq '.'
echo ""

# 检查连接是否成功
if echo "$CONNECT_RESPONSE" | jq -e '.success' > /dev/null; then
  CLUSTER_ID=$(echo "$CONNECT_RESPONSE" | jq -r '.data.id')
  echo "✅ 外部集群连接成功，集群ID: $CLUSTER_ID"
  echo ""
  
  # 4. 等待节点发现完成
  echo "4. 等待节点自动发现（10秒）..."
  sleep 10
  
  # 5. 获取集群详细信息
  echo "5. 获取集群详细信息..."
  CLUSTER_INFO=$(curl -s -X GET "$API_URL/slurm/clusters/$CLUSTER_ID/info" \
    -H "Authorization: Bearer $TOKEN")
  
  echo "$CLUSTER_INFO" | jq '.'
  echo ""
  
  # 6. 再次获取集群列表，查看新添加的集群
  echo "6. 获取更新后的集群列表..."
  UPDATED_CLUSTERS=$(curl -s -X GET "$API_URL/slurm/clusters" \
    -H "Authorization: Bearer $TOKEN")
  
  echo "$UPDATED_CLUSTERS" | jq '.'
  echo ""
  
  # 7. 验证集群类型
  CLUSTER_TYPE=$(echo "$UPDATED_CLUSTERS" | jq -r ".data[] | select(.id==$CLUSTER_ID) | .cluster_type")
  if [ "$CLUSTER_TYPE" = "external" ]; then
    echo "✅ 集群类型验证成功: external"
  else
    echo "❌ 集群类型错误: $CLUSTER_TYPE (expected: external)"
  fi
  
else
  echo "❌ 外部集群连接失败"
  echo "$CONNECT_RESPONSE" | jq '.'
fi

echo ""
echo "====== 测试完成 ======"
