#!/bin/bash

# AI-Infra Matrix Kubernetes多节点部署脚本
# 支持KubeSpawner的分布式JupyterHub部署

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HELM_CHART_DIR="$PROJECT_ROOT/helm/ai-infra-matrix"

# 默认配置
NAMESPACE="ai-infra-matrix"
RELEASE_NAME="ai-infra-matrix"
VALUES_FILE="values-k8s-prod.yaml"
USER_NAMESPACE="ai-infra-users"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 显示帮助信息
show_help() {
    cat << EOF
AI-Infra Matrix Kubernetes部署脚本

用法: $0 [选项] [命令]

命令:
  deploy          部署完整的AI-Infra Matrix系统 (默认)
  upgrade         升级现有部署
  uninstall       卸载部署
  status          查看部署状态
  logs            查看JupyterHub日志
  test            测试部署连接
  clean           清理资源

选项:
  -n, --namespace      Kubernetes命名空间 (默认: ai-infra-matrix)
  -r, --release        Helm Release名称 (默认: ai-infra-matrix)
  -f, --values         Values文件路径 (默认: values-k8s-prod.yaml)
  -h, --help           显示此帮助信息
  --dry-run            干运行模式，仅显示将要执行的操作
  --debug              启用调试模式

示例:
  $0 deploy                                    # 部署系统
  $0 deploy --dry-run                          # 预览部署操作
  $0 upgrade -f values-custom.yaml             # 使用自定义配置升级
  $0 uninstall -n my-namespace                 # 从指定命名空间卸载
  $0 status                                    # 查看状态
  $0 logs                                      # 查看日志

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖工具..."
    
    # 检查kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装，请先安装kubectl"
        exit 1
    fi
    
    # 检查helm
    if ! command -v helm &> /dev/null; then
        log_error "helm 未安装，请先安装Helm 3.x"
        exit 1
    fi
    
    # 检查集群连接
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到Kubernetes集群，请检查kubeconfig配置"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 检查/创建命名空间
ensure_namespaces() {
    log_info "检查命名空间..."
    
    # 创建主命名空间
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "创建命名空间: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    fi
    
    # 创建用户命名空间
    if ! kubectl get namespace "$USER_NAMESPACE" &> /dev/null; then
        log_info "创建用户命名空间: $USER_NAMESPACE"
        kubectl create namespace "$USER_NAMESPACE"
        kubectl label namespace "$USER_NAMESPACE" purpose=single-user-pods
    fi
    
    log_success "命名空间检查完成"
}

# 安装/更新CRDs
install_crds() {
    log_info "安装/更新CRDs..."
    
    # 如果有自定义CRDs，在这里安装
    # kubectl apply -f "$PROJECT_ROOT/k8s/crds/"
    
    log_success "CRDs安装完成"
}

# 部署应用
deploy_app() {
    local dry_run_flag=""
    if [[ "$DRY_RUN" == "true" ]]; then
        dry_run_flag="--dry-run"
    fi
    
    log_info "开始部署AI-Infra Matrix..."
    log_info "命名空间: $NAMESPACE"
    log_info "Release名称: $RELEASE_NAME"
    log_info "Values文件: $VALUES_FILE"
    
    # 检查Values文件是否存在
    if [[ ! -f "$HELM_CHART_DIR/$VALUES_FILE" ]]; then
        log_error "Values文件不存在: $HELM_CHART_DIR/$VALUES_FILE"
        exit 1
    fi
    
    # 执行Helm部署
    cd "$HELM_CHART_DIR"
    
    if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
        log_info "Release已存在，执行升级..."
        helm upgrade "$RELEASE_NAME" . \
            --namespace "$NAMESPACE" \
            --values "$VALUES_FILE" \
            --wait \
            --timeout 10m \
            $dry_run_flag
    else
        log_info "执行首次安装..."
        helm install "$RELEASE_NAME" . \
            --namespace "$NAMESPACE" \
            --values "$VALUES_FILE" \
            --create-namespace \
            --wait \
            --timeout 10m \
            $dry_run_flag
    fi
    
    if [[ "$DRY_RUN" != "true" ]]; then
        log_success "部署完成！"
        show_access_info
    else
        log_info "干运行完成"
    fi
}

# 升级应用
upgrade_app() {
    log_info "升级AI-Infra Matrix..."
    
    cd "$HELM_CHART_DIR"
    helm upgrade "$RELEASE_NAME" . \
        --namespace "$NAMESPACE" \
        --values "$VALUES_FILE" \
        --wait \
        --timeout 10m
    
    log_success "升级完成！"
    show_access_info
}

# 卸载应用
uninstall_app() {
    log_warning "准备卸载AI-Infra Matrix..."
    read -p "确认卸载? (y/N): " confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        log_info "卸载Release: $RELEASE_NAME"
        helm uninstall "$RELEASE_NAME" --namespace "$NAMESPACE"
        
        log_info "清理PVCs..."
        kubectl delete pvc --all -n "$NAMESPACE" || true
        kubectl delete pvc --all -n "$USER_NAMESPACE" || true
        
        log_success "卸载完成"
    else
        log_info "取消卸载"
    fi
}

# 显示访问信息
show_access_info() {
    log_info "获取访问信息..."
    
    # 获取NodePort
    local nodeport
    nodeport=$(kubectl get svc -n "$NAMESPACE" "$RELEASE_NAME-nginx" -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo "")
    
    # 获取节点IP
    local node_ip
    node_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "")
    
    echo ""
    log_success "部署信息:"
    echo "  命名空间: $NAMESPACE"
    echo "  Release: $RELEASE_NAME"
    echo "  用户命名空间: $USER_NAMESPACE"
    
    if [[ -n "$nodeport" && -n "$node_ip" ]]; then
        echo ""
        log_success "访问地址:"
        echo "  主页面: http://$node_ip:$nodeport"
        echo "  JupyterHub: http://$node_ip:$nodeport/jupyter"
    fi
    
    echo ""
    log_info "查看Pod状态:"
    echo "  kubectl get pods -n $NAMESPACE"
    echo ""
    log_info "查看服务状态:"
    echo "  kubectl get svc -n $NAMESPACE"
}

# 查看状态
show_status() {
    log_info "AI-Infra Matrix 部署状态:"
    echo ""
    
    # Helm状态
    echo "=== Helm Release 状态 ==="
    helm list -n "$NAMESPACE"
    echo ""
    
    # Pod状态
    echo "=== Pod 状态 ==="
    kubectl get pods -n "$NAMESPACE" -o wide
    echo ""
    
    # 服务状态
    echo "=== Service 状态 ==="
    kubectl get svc -n "$NAMESPACE"
    echo ""
    
    # PVC状态
    echo "=== PVC 状态 ==="
    kubectl get pvc -n "$NAMESPACE"
    echo ""
    
    # 用户命名空间状态
    echo "=== 用户Pod状态 ==="
    kubectl get pods -n "$USER_NAMESPACE" 2>/dev/null || echo "用户命名空间为空或不存在"
    
    show_access_info
}

# 查看日志
show_logs() {
    log_info "显示JupyterHub日志..."
    
    local pod_name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=jupyterhub -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [[ -n "$pod_name" ]]; then
        kubectl logs -n "$NAMESPACE" "$pod_name" -f
    else
        log_error "未找到JupyterHub Pod"
        exit 1
    fi
}

# 测试连接
test_deployment() {
    log_info "测试部署连接..."
    
    # 测试数据库连接
    log_info "测试PostgreSQL连接..."
    kubectl run --rm -it --restart=Never postgres-test \
        --image=postgres:13 \
        --namespace="$NAMESPACE" \
        -- psql -h "$RELEASE_NAME-postgresql" -U postgres -c "SELECT 1;" || true
    
    # 测试Redis连接
    log_info "测试Redis连接..."
    kubectl run --rm -it --restart=Never redis-test \
        --image=redis:6-alpine \
        --namespace="$NAMESPACE" \
        -- redis-cli -h "$RELEASE_NAME-redis-master" ping || true
    
    log_success "连接测试完成"
}

# 清理资源
clean_resources() {
    log_warning "清理所有资源..."
    read -p "确认清理所有资源? 这将删除数据! (y/N): " confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        log_info "清理命名空间资源..."
        kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
        kubectl delete namespace "$USER_NAMESPACE" --ignore-not-found=true
        
        log_success "资源清理完成"
    else
        log_info "取消清理"
    fi
}

# 主函数
main() {
    local command="deploy"
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -r|--release)
                RELEASE_NAME="$2"
                shift 2
                ;;
            -f|--values)
                VALUES_FILE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --debug)
                set -x
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            deploy|upgrade|uninstall|status|logs|test|clean)
                command="$1"
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 执行命令
    case "$command" in
        deploy)
            check_dependencies
            ensure_namespaces
            install_crds
            deploy_app
            ;;
        upgrade)
            check_dependencies
            upgrade_app
            ;;
        uninstall)
            check_dependencies
            uninstall_app
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        test)
            test_deployment
            ;;
        clean)
            clean_resources
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
