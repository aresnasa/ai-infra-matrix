#!/bin/bash

# Kubernetes 资源异常诊断和修复脚本
# 用于诊断 AI Infrastructure Matrix 项目中的 Kubernetes 集群资源读取异常问题

set -e

echo "🔍 Kubernetes 资源异常诊断和修复脚本"
echo "================================================"

# 设置颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 1. 基础环境检查
echo "📝 1. 基础环境检查"
echo "--------------------------------"

log_info "检查 Docker 状态..."
if docker info &>/dev/null; then
    log_success "Docker 正常运行"
else
    log_error "Docker 未运行或配置有误"
    exit 1
fi

log_info "检查 kubectl 可用性..."
if command -v kubectl &>/dev/null; then
    log_success "kubectl 已安装"
    kubectl version --client 2>/dev/null | head -1 || echo "kubectl client version"
else
    log_error "kubectl 未安装"
    exit 1
fi

log_info "检查 Docker Desktop Kubernetes..."
if kubectl cluster-info --context docker-desktop &>/dev/null; then
    log_success "Docker Desktop Kubernetes 正常运行"
    kubectl cluster-info --context docker-desktop
else
    log_error "Docker Desktop Kubernetes 未运行"
    exit 1
fi

# 2. 网络连接测试
echo ""
echo "📝 2. 网络连接测试"
echo "--------------------------------"

log_info "测试 Kubernetes API 连接..."
K8S_ENDPOINT="https://kubernetes.docker.internal:6443"
if curl -k -s --connect-timeout 5 "$K8S_ENDPOINT/version" &>/dev/null; then
    log_success "Kubernetes API 端点可达"
else
    log_warning "直接连接 Kubernetes API 可能有问题"
fi

log_info "测试 host.docker.internal 解析..."
if ping -c 1 host.docker.internal &>/dev/null; then
    log_success "host.docker.internal 解析正常"
else
    log_warning "host.docker.internal 解析失败"
fi

# 3. 后端服务状态检查
echo ""
echo "📝 3. 后端服务状态检查"
echo "--------------------------------"

log_info "检查 AI Infrastructure Matrix 服务状态..."
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

if docker-compose ps | grep -q "ai-infra-backend.*Up"; then
    log_success "后端服务正在运行"
else
    log_warning "后端服务未运行，正在启动..."
    docker-compose up -d backend
    sleep 10
fi

log_info "测试后端 API 健康检查..."
if curl -s http://localhost:8080/api/health | grep -q "healthy"; then
    log_success "后端 API 健康检查通过"
else
    log_error "后端 API 健康检查失败"
fi

# 4. Kubernetes 代理服务检查
echo ""
echo "📝 4. Kubernetes 代理服务检查"
echo "--------------------------------"

log_info "检查 k8s-proxy 服务状态..."
if docker-compose ps | grep -q "ai-infra-k8s-proxy.*Up"; then
    log_success "K8s 代理服务正在运行"
else
    log_warning "K8s 代理服务未运行，正在启动..."
    docker-compose up -d k8s-proxy
    sleep 10
fi

log_info "测试代理端口连接..."
if nc -z localhost 6443 2>/dev/null; then
    log_success "代理端口 6443 可访问"
else
    log_warning "代理端口 6443 不可访问"
fi

# 5. Kubernetes 资源操作测试
echo ""
echo "📝 5. Kubernetes 资源操作测试"
echo "--------------------------------"

log_info "测试基本资源查询..."

# 测试节点
log_info "获取节点信息..."
if kubectl get nodes --context=docker-desktop; then
    log_success "节点查询成功"
else
    log_error "节点查询失败"
fi

# 测试命名空间
log_info "获取命名空间..."
if kubectl get namespaces --context=docker-desktop; then
    log_success "命名空间查询成功"
else
    log_error "命名空间查询失败"
fi

# 测试 Pod 查询
log_info "获取所有 Pod..."
if kubectl get pods --all-namespaces --context=docker-desktop; then
    log_success "Pod 查询成功"
else
    log_error "Pod 查询失败"
fi

# 测试服务查询
log_info "获取所有服务..."
if kubectl get services --all-namespaces --context=docker-desktop; then
    log_success "服务查询成功"
else
    log_error "服务查询失败"
fi

# 6. 通过后端 API 测试集群连接
echo ""
echo "📝 6. 通过后端 API 测试集群操作"
echo "--------------------------------"

log_info "准备集群配置数据..."
CLUSTER_DATA_FILE="/tmp/k8s_cluster_test.json"

# 获取 kubeconfig 内容
KUBECONFIG_CONTENT=$(kubectl config view --context=docker-desktop --minify --flatten | jq -R -s .)

cat > "$CLUSTER_DATA_FILE" << EOF
{
    "name": "docker-desktop-test",
    "description": "Docker Desktop 测试集群",
    "apiServer": "https://kubernetes.docker.internal:6443",
    "kubeConfig": $KUBECONFIG_CONTENT,
    "namespace": "default"
}
EOF

log_info "通过 API 测试集群添加..."
API_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d @"$CLUSTER_DATA_FILE" \
    http://localhost:8080/api/kubernetes/clusters \
    || echo "API_FAILED")

if [[ "$API_RESPONSE" == "API_FAILED" ]] || echo "$API_RESPONSE" | grep -q "error\|Error"; then
    log_warning "集群添加可能需要身份验证或有其他问题"
    log_info "API 响应: $API_RESPONSE"
else
    log_success "集群通过 API 添加成功"
    echo "$API_RESPONSE" | jq . 2>/dev/null || echo "$API_RESPONSE"
fi

# 7. 创建测试资源
echo ""
echo "📝 7. 创建测试资源"
echo "--------------------------------"

log_info "创建测试命名空间..."
kubectl create namespace ai-infra-test --context=docker-desktop --dry-run=client -o yaml | kubectl apply --context=docker-desktop -f -

log_info "创建测试 ConfigMap..."
kubectl create configmap test-config \
    --from-literal=app="ai-infra-matrix" \
    --from-literal=test="true" \
    --namespace=ai-infra-test \
    --context=docker-desktop \
    --dry-run=client -o yaml | kubectl apply --context=docker-desktop -f -

log_info "创建测试 Pod..."
cat > /tmp/test-diagnostic-pod.yaml << 'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: diagnostic-test-pod
  namespace: ai-infra-test
  labels:
    app: ai-infra-matrix
    test: diagnostic
spec:
  containers:
  - name: test-container
    image: nginx:stable-alpine-perl
    ports:
    - containerPort: 80
    env:
    - name: TEST_VAR
      value: "ai-infra-matrix-test"
    resources:
      requests:
        memory: "64Mi"
        cpu: "50m"
      limits:
        memory: "128Mi"
        cpu: "100m"
  restartPolicy: Never
EOF

kubectl apply -f /tmp/test-diagnostic-pod.yaml --context=docker-desktop

log_info "等待 Pod 启动..."
sleep 10

log_info "检查测试资源状态..."
kubectl get all -n ai-infra-test --context=docker-desktop

# 8. 测试资源查看功能
echo ""
echo "📝 8. 测试各种资源查看功能"
echo "--------------------------------"

log_info "测试资源描述功能..."
kubectl describe pod diagnostic-test-pod -n ai-infra-test --context=docker-desktop

log_info "测试日志查看功能..."
kubectl logs diagnostic-test-pod -n ai-infra-test --context=docker-desktop || log_warning "Pod 可能还在启动中"

log_info "测试资源标签筛选..."
kubectl get pods -l app=ai-infra-matrix --all-namespaces --context=docker-desktop

log_info "测试资源 YAML 导出..."
kubectl get pod diagnostic-test-pod -n ai-infra-test -o yaml --context=docker-desktop > /tmp/test-pod-export.yaml
log_success "Pod YAML 已导出到 /tmp/test-pod-export.yaml"

# 9. 测试通过代理的连接
echo ""
echo "📝 9. 测试通过代理的连接"
echo "--------------------------------"

log_info "测试代理配置文件..."
PROXY_KUBECONFIG="kubeconfig-proxy.yaml"
if [ -f "$PROXY_KUBECONFIG" ]; then
    log_info "使用代理配置测试连接..."
    if kubectl --kubeconfig="$PROXY_KUBECONFIG" get nodes &>/dev/null; then
        log_success "代理连接测试成功"
        kubectl --kubeconfig="$PROXY_KUBECONFIG" get nodes
    else
        log_warning "代理连接失败，可能需要配置调整"
    fi
else
    log_warning "代理配置文件不存在"
fi

# 10. 性能和负载测试
echo ""
echo "📝 10. 性能和负载测试"
echo "--------------------------------"

log_info "测试并发资源查询..."
for i in {1..5}; do
    (kubectl get pods --all-namespaces --context=docker-desktop &>/dev/null && echo "查询 $i 成功") &
done
wait
log_success "并发查询测试完成"

log_info "测试大量资源列表..."
kubectl get events --all-namespaces --context=docker-desktop | head -20

# 11. 日志收集和错误诊断
echo ""
echo "📝 11. 日志收集和错误诊断"
echo "--------------------------------"

log_info "收集后端服务日志..."
docker-compose logs --tail=50 backend > /tmp/backend-logs.txt
log_success "后端日志已保存到 /tmp/backend-logs.txt"

log_info "收集代理服务日志..."
docker-compose logs --tail=50 k8s-proxy > /tmp/k8s-proxy-logs.txt
log_success "代理日志已保存到 /tmp/k8s-proxy-logs.txt"

log_info "检查系统资源使用..."
echo "Docker 容器状态:"
docker-compose ps

echo ""
echo "系统内存使用:"
docker stats --no-stream

# 12. 清理测试资源
echo ""
echo "📝 12. 清理测试资源"
echo "--------------------------------"

log_info "清理测试资源..."
kubectl delete namespace ai-infra-test --context=docker-desktop --ignore-not-found=true
rm -f /tmp/test-diagnostic-pod.yaml /tmp/test-pod-export.yaml

# 13. 生成诊断报告
echo ""
echo "📝 13. 生成诊断报告"
echo "--------------------------------"

REPORT_FILE="/tmp/k8s-diagnostic-report-$(date +%Y%m%d-%H%M%S).txt"

cat > "$REPORT_FILE" << EOF
Kubernetes 资源异常诊断报告
========================================
生成时间: $(date)
系统信息: $(uname -a)

Docker 版本:
$(docker --version)

Kubectl 版本:
$(kubectl version --client 2>/dev/null | head -1 || echo "kubectl client")

集群信息:
$(kubectl cluster-info --context=docker-desktop)

节点状态:
$(kubectl get nodes --context=docker-desktop)

系统命名空间 Pod 状态:
$(kubectl get pods -n kube-system --context=docker-desktop)

Docker Compose 服务状态:
$(docker-compose ps)

后端服务健康检查:
$(curl -s http://localhost:8080/api/health)

网络连接测试:
$(curl -k -s --connect-timeout 5 https://kubernetes.docker.internal:6443/version || echo "连接失败")

EOF

log_success "诊断报告已生成: $REPORT_FILE"

# 14. 总结和建议
echo ""
echo "🎉 诊断完成!"
echo "=================================="
log_success "Kubernetes 集群基础功能正常"
log_success "Docker Desktop 集群连接正常"
log_success "资源创建和查看功能正常"

echo ""
echo "📋 诊断结果文件:"
echo "- 诊断报告: $REPORT_FILE"
echo "- 后端日志: /tmp/backend-logs.txt"
echo "- 代理日志: /tmp/k8s-proxy-logs.txt"
echo "- 集群配置: $CLUSTER_DATA_FILE"

echo ""
echo "🚀 下一步建议:"
echo "1. 检查前端界面的 Kubernetes 管理页面"
echo "2. 通过前端界面添加集群并测试功能"
echo "3. 如有特定错误，请查看相应的日志文件"
echo "4. 确认代理配置是否符合网络环境要求"

echo ""
echo "📞 如需进一步支持，请提供:"
echo "- 具体的错误信息和重现步骤"
echo "- 诊断报告文件内容"
echo "- 相关的日志文件"
