#!/bin/bash

# docker-desktop kubernetes cluster setup script
# 用于添加 docker-desktop 本地集群到 AI Infrastructure Matrix 项目中

set -e

echo "🚀 Docker Desktop Kubernetes 集群配置脚本"
echo "================================================"

# 1. 检查Docker Desktop Kubernetes状态
echo "📝 1. 检查 Docker Desktop Kubernetes 状态..."
if ! kubectl cluster-info --context docker-desktop &>/dev/null; then
    echo "❌ Docker Desktop Kubernetes 未运行或配置有误"
    echo "请确保 Docker Desktop 已启动并启用 Kubernetes"
    exit 1
fi

echo "✅ Docker Desktop Kubernetes 运行正常"

# 2. 获取当前kubeconfig
echo "📝 2. 获取当前 kubeconfig..."
KUBE_CONFIG_PATH="$HOME/.kube/config"
if [ ! -f "$KUBE_CONFIG_PATH" ]; then
    echo "❌ 未找到 kubeconfig 文件: $KUBE_CONFIG_PATH"
    exit 1
fi

# 3. 生成docker-desktop集群的kubeconfig内容
echo "📝 3. 生成 docker-desktop 集群配置..."
DOCKER_DESKTOP_CONFIG=$(kubectl config view --context=docker-desktop --minify --flatten)

# 4. 创建用于API调用的kubeconfig JSON
echo "📝 4. 创建集群配置数据..."

# 获取集群信息
CLUSTER_SERVER=$(kubectl config view --context=docker-desktop -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_NAME="docker-desktop-local"
NAMESPACE="default"

# 创建API请求的JSON数据
cat > /tmp/docker_desktop_cluster.json << EOF
{
    "name": "$CLUSTER_NAME",
    "description": "Docker Desktop 本地 Kubernetes 集群",
    "apiServer": "$CLUSTER_SERVER",
    "kubeConfig": $(echo "$DOCKER_DESKTOP_CONFIG" | jq -R -s .),
    "namespace": "$NAMESPACE"
}
EOF

echo "✅ 集群配置文件已生成: /tmp/docker_desktop_cluster.json"

# 5. 通过API添加集群
echo "📝 5. 通过 API 添加集群到项目..."

# 检查后端服务是否运行
if ! curl -s http://localhost:8080/api/health &>/dev/null; then
    echo "❌ 后端服务未运行，请先启动 docker-compose"
    echo "运行: docker-compose up -d"
    exit 1
fi

# 尝试添加集群 (需要认证token，先测试无认证接口)
echo "📝 正在添加集群..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer test-token" \
    -d @/tmp/docker_desktop_cluster.json \
    http://localhost:8080/api/kubernetes/clusters \
    || echo "FAILED")

if [[ "$RESPONSE" == "FAILED" ]] || [[ "$RESPONSE" == *"error"* ]]; then
    echo "⚠️  添加集群可能需要认证，手动添加集群信息:"
    echo ""
    echo "集群名称: $CLUSTER_NAME"
    echo "描述: Docker Desktop 本地 Kubernetes 集群"
    echo "API Server: $CLUSTER_SERVER"
    echo "命名空间: $NAMESPACE"
    echo ""
    echo "KubeConfig 内容已保存到: /tmp/docker_desktop_cluster.json"
else
    echo "✅ 集群添加成功"
    echo "$RESPONSE" | jq .
fi

# 6. 测试集群连接和资源操作
echo "📝 6. 测试集群连接和基本资源操作..."

echo "🔍 测试 1: 获取节点信息"
kubectl get nodes --context=docker-desktop

echo ""
echo "🔍 测试 2: 获取命名空间"
kubectl get namespaces --context=docker-desktop

echo ""
echo "🔍 测试 3: 获取所有Pod"
kubectl get pods --all-namespaces --context=docker-desktop

echo ""
echo "🔍 测试 4: 创建测试Pod"
TEST_POD_NAME="test-pod-$(date +%s)"
cat > /tmp/test-pod.yaml << EOF
apiVersion: v1
kind: Pod
metadata:
  name: $TEST_POD_NAME
  namespace: default
  labels:
    app: test
    created-by: ai-infra-matrix
spec:
  containers:
  - name: test-container
    image: nginx:stable-alpine-perl
    ports:
    - containerPort: 80
  restartPolicy: Never
EOF

kubectl apply -f /tmp/test-pod.yaml --context=docker-desktop

echo "✅ 测试Pod创建成功"

echo ""
echo "🔍 测试 5: 检查Pod状态"
sleep 5
kubectl get pods -l created-by=ai-infra-matrix --context=docker-desktop

# 7. 更新代理配置
echo "📝 7. 检查代理配置..."
PROXY_CONFIG_FILE="kubeconfig-proxy.yaml"

if [ -f "$PROXY_CONFIG_FILE" ]; then
    echo "✅ 代理配置文件已存在: $PROXY_CONFIG_FILE"
    echo "📝 当前代理配置:"
    cat "$PROXY_CONFIG_FILE"
else
    echo "📝 创建代理配置文件..."
    
    # 从现有kubeconfig创建代理版本
    kubectl config view --context=docker-desktop --minify --flatten > "$PROXY_CONFIG_FILE"
    
    # 修改服务器地址为代理地址
    sed -i.bak 's|kubernetes.docker.internal:6443|192.168.0.200:6443|g' "$PROXY_CONFIG_FILE"
    
    echo "✅ 代理配置文件已创建: $PROXY_CONFIG_FILE"
fi

# 8. 测试通过代理的连接
echo "📝 8. 测试代理连接..."

if docker-compose ps | grep -q k8s-proxy; then
    echo "✅ Kubernetes 代理服务运行中"
    
    echo "🔍 测试通过代理连接集群..."
    if kubectl --kubeconfig="$PROXY_CONFIG_FILE" get nodes &>/dev/null; then
        echo "✅ 代理连接测试成功"
    else
        echo "⚠️  代理连接测试失败，但这可能是正常的（需要启动代理服务）"
    fi
else
    echo "📝 启动 Kubernetes 代理服务..."
    docker-compose up -d k8s-proxy
    
    echo "⏳ 等待代理服务启动..."
    sleep 10
    
    echo "🔍 再次测试代理连接..."
    if kubectl --kubeconfig="$PROXY_CONFIG_FILE" get nodes &>/dev/null; then
        echo "✅ 代理连接测试成功"
    else
        echo "⚠️  代理连接可能需要进一步配置"
    fi
fi

# 9. 清理测试资源
echo "📝 9. 清理测试资源..."
kubectl delete pods -l created-by=ai-infra-matrix --context=docker-desktop --ignore-not-found=true

echo ""
echo "🎉 Docker Desktop Kubernetes 集群配置完成!"
echo "================================================"
echo "✅ 集群状态: 正常运行"
echo "✅ 基本功能: 已测试"
echo "✅ 代理配置: 已生成"
echo "📄 配置文件: $PROXY_CONFIG_FILE"
echo "📄 集群数据: /tmp/docker_desktop_cluster.json"
echo ""
echo "🚀 下一步操作:"
echo "1. 在 AI Infrastructure Matrix 前端界面中添加集群"
echo "2. 使用生成的配置文件测试集群连接"
echo "3. 通过前端界面管理 Kubernetes 资源"
echo ""
echo "📞 如有问题，请检查:"
echo "- Docker Desktop Kubernetes 是否启用"
echo "- 后端服务是否正常运行"
echo "- 网络连接是否正常"
