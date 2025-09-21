#!/bin/bash

# KubeSpawner配置测试脚本
# 验证JupyterHub多节点部署是否正常工作

set -e

# 脚本配置
NAMESPACE="ai-infra-matrix"
USER_NAMESPACE="ai-infra-users"
RELEASE_NAME="ai-infra-matrix"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 检查Kubernetes连接
check_k8s_connection() {
    log_info "检查Kubernetes连接..."
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到Kubernetes集群"
        exit 1
    fi
    
    log_success "Kubernetes连接正常"
}

# 检查命名空间
check_namespaces() {
    log_info "检查命名空间..."
    
    # 检查主命名空间
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_success "主命名空间存在: $NAMESPACE"
    else
        log_error "主命名空间不存在: $NAMESPACE"
        return 1
    fi
    
    # 检查用户命名空间
    if kubectl get namespace "$USER_NAMESPACE" &> /dev/null; then
        log_success "用户命名空间存在: $USER_NAMESPACE"
    else
        log_warning "用户命名空间不存在: $USER_NAMESPACE"
    fi
}

# 检查部署状态
check_deployment_status() {
    log_info "检查部署状态..."
    
    # 检查JupyterHub Pod
    local jupyterhub_pod
    jupyterhub_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=jupyterhub -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [[ -n "$jupyterhub_pod" ]]; then
        local status
        status=$(kubectl get pod -n "$NAMESPACE" "$jupyterhub_pod" -o jsonpath='{.status.phase}')
        if [[ "$status" == "Running" ]]; then
            log_success "JupyterHub Pod运行正常: $jupyterhub_pod"
        else
            log_error "JupyterHub Pod状态异常: $status"
            return 1
        fi
    else
        log_error "未找到JupyterHub Pod"
        return 1
    fi
}

# 检查服务状态
check_services() {
    log_info "检查服务状态..."
    
    # 检查JupyterHub服务
    if kubectl get svc -n "$NAMESPACE" "$RELEASE_NAME-jupyterhub" &> /dev/null; then
        log_success "JupyterHub服务存在"
    else
        log_error "JupyterHub服务不存在"
        return 1
    fi
    
    # 检查Nginx服务
    if kubectl get svc -n "$NAMESPACE" "$RELEASE_NAME-nginx" &> /dev/null; then
        local nodeport
        nodeport=$(kubectl get svc -n "$NAMESPACE" "$RELEASE_NAME-nginx" -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo "")
        if [[ -n "$nodeport" ]]; then
            log_success "Nginx服务存在，NodePort: $nodeport"
        else
            log_warning "Nginx服务存在但没有NodePort"
        fi
    else
        log_error "Nginx服务不存在"
        return 1
    fi
}

# 检查RBAC配置
check_rbac() {
    log_info "检查RBAC配置..."
    
    # 检查ServiceAccount
    if kubectl get sa -n "$NAMESPACE" "$RELEASE_NAME-jupyterhub" &> /dev/null; then
        log_success "JupyterHub ServiceAccount存在"
    else
        log_error "JupyterHub ServiceAccount不存在"
        return 1
    fi
    
    # 检查Role
    if kubectl get role -n "$USER_NAMESPACE" "$RELEASE_NAME-jupyterhub-user-pods" &> /dev/null; then
        log_success "用户Pod管理Role存在"
    else
        log_warning "用户Pod管理Role不存在"
    fi
    
    # 检查RoleBinding
    if kubectl get rolebinding -n "$USER_NAMESPACE" "$RELEASE_NAME-jupyterhub-user-pods" &> /dev/null; then
        log_success "用户Pod管理RoleBinding存在"
    else
        log_warning "用户Pod管理RoleBinding不存在"
    fi
}

# 检查环境变量配置
check_env_config() {
    log_info "检查JupyterHub环境变量配置..."
    
    local jupyterhub_pod
    jupyterhub_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=jupyterhub -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [[ -n "$jupyterhub_pod" ]]; then
        # 检查Spawner配置
        local spawner_type
        spawner_type=$(kubectl exec -n "$NAMESPACE" "$jupyterhub_pod" -- env | grep "JUPYTERHUB_SPAWNER=" | cut -d'=' -f2 2>/dev/null || echo "")
        
        if [[ "$spawner_type" == "kubernetes" ]]; then
            log_success "Spawner类型配置正确: kubernetes"
        else
            log_warning "Spawner类型: $spawner_type (期望: kubernetes)"
        fi
        
        # 检查用户命名空间配置
        local k8s_namespace
        k8s_namespace=$(kubectl exec -n "$NAMESPACE" "$jupyterhub_pod" -- env | grep "KUBERNETES_NAMESPACE=" | cut -d'=' -f2 2>/dev/null || echo "")
        
        if [[ "$k8s_namespace" == "$USER_NAMESPACE" ]]; then
            log_success "用户命名空间配置正确: $k8s_namespace"
        else
            log_warning "用户命名空间配置: $k8s_namespace (期望: $USER_NAMESPACE)"
        fi
        
        # 检查公共主机配置
        local public_host
        public_host=$(kubectl exec -n "$NAMESPACE" "$jupyterhub_pod" -- env | grep "JUPYTERHUB_PUBLIC_HOST=" | cut -d'=' -f2 2>/dev/null || echo "")
        
        if [[ "$public_host" =~ ^192\.168\.0\.199 ]]; then
            log_success "公共主机配置正确: $public_host"
        else
            log_warning "公共主机配置: $public_host (期望包含192.168.0.199)"
        fi
    else
        log_error "无法检查环境变量：JupyterHub Pod不存在"
        return 1
    fi
}

# 测试网络访问
test_network_access() {
    log_info "测试网络访问..."
    
    # 获取访问地址
    local nodeport
    nodeport=$(kubectl get svc -n "$NAMESPACE" "$RELEASE_NAME-nginx" -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo "")
    
    local node_ip
    node_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "")
    
    if [[ -n "$nodeport" && -n "$node_ip" ]]; then
        log_info "测试访问地址: http://$node_ip:$nodeport"
        
        # 测试主页面
        if curl -s -o /dev/null -w "%{http_code}" "http://$node_ip:$nodeport" | grep -q "200\|302"; then
            log_success "主页面访问正常"
        else
            log_warning "主页面访问异常"
        fi
        
        # 测试JupyterHub页面
        if curl -s -o /dev/null -w "%{http_code}" "http://$node_ip:$nodeport/jupyter" | grep -q "200\|302"; then
            log_success "JupyterHub页面访问正常"
        else
            log_warning "JupyterHub页面访问异常"
        fi
    else
        log_error "无法获取访问地址信息"
        return 1
    fi
}

# 检查存储配置
check_storage_config() {
    log_info "检查存储配置..."
    
    # 检查StorageClass
    if kubectl get storageclass local-path &> /dev/null; then
        log_success "本地存储类存在: local-path"
    else
        log_warning "本地存储类不存在: local-path"
    fi
    
    # 检查共享存储PVC
    if kubectl get pvc -n "$NAMESPACE" "$RELEASE_NAME-shared-notebooks" &> /dev/null; then
        log_success "共享存储PVC存在"
    else
        log_warning "共享存储PVC不存在"
    fi
    
    # 检查JupyterHub数据PVC
    if kubectl get pvc -n "$NAMESPACE" "$RELEASE_NAME-jupyterhub-data" &> /dev/null; then
        log_success "JupyterHub数据PVC存在"
    else
        log_warning "JupyterHub数据PVC不存在"
    fi
}

# 检查日志
check_logs() {
    log_info "检查JupyterHub启动日志..."
    
    local jupyterhub_pod
    jupyterhub_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=jupyterhub -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [[ -n "$jupyterhub_pod" ]]; then
        # 检查是否有错误日志
        local error_count
        error_count=$(kubectl logs -n "$NAMESPACE" "$jupyterhub_pod" --tail=100 | grep -i "error\|exception\|failed" | wc -l || echo "0")
        
        if [[ "$error_count" -eq 0 ]]; then
            log_success "JupyterHub日志无明显错误"
        else
            log_warning "JupyterHub日志中发现 $error_count 个潜在错误"
            echo "最近的错误信息："
            kubectl logs -n "$NAMESPACE" "$jupyterhub_pod" --tail=50 | grep -i "error\|exception\|failed" | head -5
        fi
        
        # 检查KubeSpawner相关日志
        if kubectl logs -n "$NAMESPACE" "$jupyterhub_pod" --tail=100 | grep -q "KubeSpawner\|kubernetes"; then
            log_success "发现KubeSpawner相关日志"
        else
            log_warning "未发现KubeSpawner相关日志"
        fi
    else
        log_error "无法检查日志：JupyterHub Pod不存在"
        return 1
    fi
}

# 显示访问信息
show_access_info() {
    echo ""
    log_info "部署访问信息："
    
    local nodeport
    nodeport=$(kubectl get svc -n "$NAMESPACE" "$RELEASE_NAME-nginx" -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo "")
    
    local node_ip
    node_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "")
    
    if [[ -n "$nodeport" && -n "$node_ip" ]]; then
        echo "  主页面: http://$node_ip:$nodeport"
        echo "  JupyterHub: http://$node_ip:$nodeport/jupyter"
        echo "  管理员账户: admin / demo-password"
    else
        echo "  无法获取访问信息"
    fi
    
    echo ""
    echo "常用检查命令："
    echo "  kubectl get pods -n $NAMESPACE"
    echo "  kubectl get pods -n $USER_NAMESPACE"
    echo "  kubectl logs -n $NAMESPACE deployment/$RELEASE_NAME-jupyterhub"
}

# 主测试函数
run_tests() {
    echo "========================================"
    echo "AI-Infra Matrix KubeSpawner 配置测试"
    echo "========================================"
    echo ""
    
    local failed_tests=0
    
    # 执行各项检查
    check_k8s_connection || ((failed_tests++))
    echo ""
    
    check_namespaces || ((failed_tests++))
    echo ""
    
    check_deployment_status || ((failed_tests++))
    echo ""
    
    check_services || ((failed_tests++))
    echo ""
    
    check_rbac || ((failed_tests++))
    echo ""
    
    check_env_config || ((failed_tests++))
    echo ""
    
    check_storage_config || ((failed_tests++))
    echo ""
    
    test_network_access || ((failed_tests++))
    echo ""
    
    check_logs || ((failed_tests++))
    echo ""
    
    # 显示测试结果
    echo "========================================"
    if [[ $failed_tests -eq 0 ]]; then
        log_success "所有测试通过！KubeSpawner配置正常"
    else
        log_warning "$failed_tests 个测试未完全通过，请检查警告信息"
    fi
    echo "========================================"
    
    show_access_info
    
    return $failed_tests
}

# 显示帮助信息
show_help() {
    cat << EOF
KubeSpawner配置测试脚本

用法: $0 [选项]

选项:
  -n, --namespace      主命名空间 (默认: ai-infra-matrix)
  -u, --user-namespace 用户命名空间 (默认: ai-infra-users)
  -r, --release        Release名称 (默认: ai-infra-matrix)
  -h, --help           显示此帮助信息
  --logs-only          仅检查日志
  --network-only       仅测试网络访问

示例:
  $0                                  # 运行完整测试
  $0 --logs-only                      # 仅检查日志
  $0 -n my-namespace                  # 指定命名空间测试

EOF
}

# 主函数
main() {
    local logs_only=false
    local network_only=false
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -u|--user-namespace)
                USER_NAMESPACE="$2"
                shift 2
                ;;
            -r|--release)
                RELEASE_NAME="$2"
                shift 2
                ;;
            --logs-only)
                logs_only=true
                shift
                ;;
            --network-only)
                network_only=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 执行对应测试
    if [[ "$logs_only" == "true" ]]; then
        check_logs
    elif [[ "$network_only" == "true" ]]; then
        test_network_access
    else
        run_tests
    fi
}

# 执行主函数
main "$@"
