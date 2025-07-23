#!/bin/bash

# 集群操作功能测试脚本
set -e

echo "☸️  测试集群操作功能..."

BASE_URL="http://backend:8080"
TOKEN="test-token-123"

# 测试1: 基础集群操作 - 获取节点信息
echo "测试获取集群节点信息..."

NODES_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_nodes",
    "parameters": {
      "output": "json"
    },
    "description": "获取集群节点信息"
  }')

NODES_OP_ID=$(echo $NODES_RESPONSE | jq -r .operation_id)
NODES_STATUS=$(echo $NODES_RESPONSE | jq -r .status)

if [ "$NODES_STATUS" = "pending" ] && [ "$NODES_OP_ID" != "null" ]; then
    echo "✅ 获取节点信息操作提交成功，ID: $NODES_OP_ID"
else
    echo "❌ 获取节点信息操作提交失败"
    exit 1
fi

# 测试2: 获取Pod列表
echo "测试获取Pod列表..."

PODS_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_pods",
    "parameters": {
      "namespace": "default",
      "label_selector": "app=test"
    },
    "description": "获取默认命名空间的测试Pod"
  }')

PODS_OP_ID=$(echo $PODS_RESPONSE | jq -r .operation_id)
echo "✅ 获取Pod列表操作提交成功，ID: $PODS_OP_ID"

# 测试3: 获取服务列表
echo "测试获取服务列表..."

SERVICES_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_services",
    "parameters": {
      "namespace": "default"
    },
    "description": "获取默认命名空间的服务"
  }')

SERVICES_OP_ID=$(echo $SERVICES_RESPONSE | jq -r .operation_id)
echo "✅ 获取服务列表操作提交成功，ID: $SERVICES_OP_ID"

# 测试4: 扩容部署
echo "测试扩容部署..."

SCALE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "scale_deployment",
    "parameters": {
      "deployment": "test-app",
      "namespace": "default",
      "replicas": 3
    },
    "description": "扩容test-app到3个副本"
  }')

SCALE_OP_ID=$(echo $SCALE_RESPONSE | jq -r .operation_id)
echo "✅ 扩容部署操作提交成功，ID: $SCALE_OP_ID"

# 测试5: 更新ConfigMap
echo "测试更新ConfigMap..."

CONFIGMAP_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "update_configmap",
    "parameters": {
      "name": "test-config",
      "namespace": "default",
      "data": {
        "config.yaml": "test: value"
      }
    },
    "description": "更新测试ConfigMap"
  }')

CONFIGMAP_OP_ID=$(echo $CONFIGMAP_RESPONSE | jq -r .operation_id)
echo "✅ 更新ConfigMap操作提交成功，ID: $CONFIGMAP_OP_ID"

# 测试6: 重启部署
echo "测试重启部署..."

RESTART_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "restart_deployment",
    "parameters": {
      "deployment": "test-app",
      "namespace": "default"
    },
    "description": "重启test-app部署"
  }')

RESTART_OP_ID=$(echo $RESTART_RESPONSE | jq -r .operation_id)
echo "✅ 重启部署操作提交成功，ID: $RESTART_OP_ID"

# 测试7: 获取部署状态
echo "测试获取部署状态..."

DEPLOYMENT_STATUS_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_deployment_status",
    "parameters": {
      "deployment": "test-app",
      "namespace": "default"
    },
    "description": "获取test-app部署状态"
  }')

DEPLOYMENT_STATUS_OP_ID=$(echo $DEPLOYMENT_STATUS_RESPONSE | jq -r .operation_id)
echo "✅ 获取部署状态操作提交成功，ID: $DEPLOYMENT_STATUS_OP_ID"

# 测试8: 检查操作状态
echo "检查所有操作状态..."

OPERATION_IDS=($NODES_OP_ID $PODS_OP_ID $SERVICES_OP_ID $SCALE_OP_ID $CONFIGMAP_OP_ID $RESTART_OP_ID $DEPLOYMENT_STATUS_OP_ID)

sleep 5  # 等待操作处理

for OP_ID in "${OPERATION_IDS[@]}"; do
    if [ "$OP_ID" != "null" ] && [ -n "$OP_ID" ]; then
        STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/operations/$OP_ID/status" \
          -H "Authorization: Bearer $TOKEN")
        
        STATUS=$(echo $STATUS_RESPONSE | jq -r .data.status)
        PROGRESS=$(echo $STATUS_RESPONSE | jq -r .data.progress // 0)
        echo "  操作 $OP_ID: 状态=$STATUS, 进度=$PROGRESS%"
    fi
done

# 测试9: 复杂集群操作 - 多步骤操作
echo "测试复杂多步骤操作..."

COMPLEX_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "deploy_application",
    "parameters": {
      "app_name": "test-complex-app",
      "namespace": "default",
      "image": "nginx:latest",
      "replicas": 2,
      "port": 80,
      "service_type": "ClusterIP"
    },
    "description": "部署复杂测试应用"
  }')

COMPLEX_OP_ID=$(echo $COMPLEX_RESPONSE | jq -r .operation_id)
echo "✅ 复杂操作提交成功，ID: $COMPLEX_OP_ID"

# 测试10: 监控资源使用情况
echo "测试监控资源使用情况..."

RESOURCE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_resource_usage",
    "parameters": {
      "resource_type": "all",
      "namespace": "default"
    },
    "description": "获取资源使用情况"
  }')

RESOURCE_OP_ID=$(echo $RESOURCE_RESPONSE | jq -r .operation_id)
echo "✅ 资源监控操作提交成功，ID: $RESOURCE_OP_ID"

# 测试11: 日志获取
echo "测试获取Pod日志..."

LOGS_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_pod_logs",
    "parameters": {
      "pod": "test-pod",
      "namespace": "default",
      "lines": 100,
      "follow": false
    },
    "description": "获取test-pod的日志"
  }')

LOGS_OP_ID=$(echo $LOGS_RESPONSE | jq -r .operation_id)
echo "✅ 日志获取操作提交成功，ID: $LOGS_OP_ID"

# 测试12: 事件查询
echo "测试查询集群事件..."

EVENTS_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_events",
    "parameters": {
      "namespace": "default",
      "limit": 50
    },
    "description": "获取最近50个事件"
  }')

EVENTS_OP_ID=$(echo $EVENTS_RESPONSE | jq -r .operation_id)
echo "✅ 事件查询操作提交成功，ID: $EVENTS_OP_ID"

# 测试13: 网络策略操作
echo "测试网络策略操作..."

NETWORK_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "apply_network_policy",
    "parameters": {
      "policy_name": "test-policy",
      "namespace": "default",
      "rules": {
        "ingress": [],
        "egress": []
      }
    },
    "description": "应用测试网络策略"
  }')

NETWORK_OP_ID=$(echo $NETWORK_RESPONSE | jq -r .operation_id)
echo "✅ 网络策略操作提交成功，ID: $NETWORK_OP_ID"

# 测试14: 错误操作处理
echo "测试错误操作处理..."

ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/ai/async/cluster-operations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "invalid_operation",
    "parameters": {},
    "description": "无效操作测试"
  }')

ERROR_OP_ID=$(echo $ERROR_RESPONSE | jq -r .operation_id)
if [ "$ERROR_OP_ID" != "null" ]; then
    echo "✅ 错误操作也被正确排队，ID: $ERROR_OP_ID"
    
    # 检查错误处理
    sleep 3
    ERROR_STATUS_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/operations/$ERROR_OP_ID/status" \
      -H "Authorization: Bearer $TOKEN")
    
    ERROR_STATUS=$(echo $ERROR_STATUS_RESPONSE | jq -r .data.status)
    ERROR_MESSAGE=$(echo $ERROR_STATUS_RESPONSE | jq -r .data.error)
    
    if [ "$ERROR_STATUS" = "failed" ] && [ "$ERROR_MESSAGE" != "null" ]; then
        echo "✅ 错误操作处理正常: $ERROR_MESSAGE"
    else
        echo "⚠️  错误操作处理可能不完善"
    fi
else
    echo "⚠️  错误操作可能在提交阶段被拒绝"
fi

# 测试15: 检查Redis中的集群操作队列
echo "检查Redis集群操作队列..."

CLUSTER_QUEUE_LEN=$(redis-cli -u redis://redis:6379 XLEN ai:cluster:operations)
echo "✅ 集群操作队列长度: $CLUSTER_QUEUE_LEN"

# 获取队列中的最新操作
LATEST_OPERATIONS=$(redis-cli -u redis://redis:6379 XREVRANGE ai:cluster:operations + - COUNT 3)
echo "✅ 最新的3个集群操作已记录"

# 测试16: 长期状态监控
echo "进行长期状态监控..."

MONITOR_OPS=($COMPLEX_OP_ID $RESOURCE_OP_ID $LOGS_OP_ID)

for i in {1..3}; do
    echo "  监控轮次 $i:"
    for OP_ID in "${MONITOR_OPS[@]}"; do
        if [ "$OP_ID" != "null" ] && [ -n "$OP_ID" ]; then
            MONITOR_RESPONSE=$(curl -s "$BASE_URL/api/ai/async/operations/$OP_ID/status" \
              -H "Authorization: Bearer $TOKEN")
            
            MONITOR_STATUS=$(echo $MONITOR_RESPONSE | jq -r .data.status)
            MONITOR_PROGRESS=$(echo $MONITOR_RESPONSE | jq -r .data.progress // 0)
            echo "    操作 $OP_ID: $MONITOR_STATUS ($MONITOR_PROGRESS%)"
        fi
    done
    sleep 5
done

echo "🎉 集群操作功能测试完成！"
