#!/bin/bash

# JupyterHub K8s GPU集成系统部署脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
NAMESPACE=${NAMESPACE:-"jupyterhub-jobs"}
REGISTRY=${REGISTRY:-"localhost:5000"}
VERSION=${VERSION:-"latest"}
NFS_SERVER=${NFS_SERVER:-"nfs-server.default.svc.cluster.local"}
NFS_PATH=${NFS_PATH:-"/shared"}

echo -e "${BLUE}=== JupyterHub K8s GPU集成系统部署脚本 ===${NC}"

# 函数定义
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    print_status "检查系统依赖..."
    
    # 检查kubectl
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl未安装，请先安装kubectl"
        exit 1
    fi
    
    # 检查docker
    if ! command -v docker &> /dev/null; then
        print_error "docker未安装，请先安装docker"
        exit 1
    fi
    
    # 检查集群连接
    if ! kubectl cluster-info &> /dev/null; then
        print_error "无法连接到Kubernetes集群"
        exit 1
    fi
    
    print_status "依赖检查通过"
}

# 构建Docker镜像
build_images() {
    print_status "构建Docker镜像..."
    
    if [ -f "scripts/build-jupyterhub-images.sh" ]; then
        chmod +x scripts/build-jupyterhub-images.sh
        REGISTRY=$REGISTRY VERSION=$VERSION ./scripts/build-jupyterhub-images.sh --test
    else
        print_error "构建脚本不存在: scripts/build-jupyterhub-images.sh"
        exit 1
    fi
    
    print_status "Docker镜像构建完成"
}

# 部署Kubernetes资源
deploy_k8s_resources() {
    print_status "部署Kubernetes资源..."
    
    # 检查配置文件
    if [ ! -f "k8s/jupyterhub-namespace.yaml" ]; then
        print_error "Kubernetes配置文件不存在: k8s/jupyterhub-namespace.yaml"
        exit 1
    fi
    
    # 应用配置
    kubectl apply -f k8s/jupyterhub-namespace.yaml
    
    # 等待命名空间创建
    kubectl wait --for=condition=Active namespace/$NAMESPACE --timeout=60s
    
    print_status "Kubernetes资源部署完成"
}

# 配置GPU节点
configure_gpu_nodes() {
    print_status "配置GPU节点..."
    
    # 获取GPU节点列表
    GPU_NODES=$(kubectl get nodes -l accelerator=nvidia -o name 2>/dev/null || echo "")
    
    if [ -z "$GPU_NODES" ]; then
        print_warning "未找到标记为GPU的节点，请手动标记GPU节点："
        print_warning "kubectl label nodes <node-name> accelerator=nvidia"
        print_warning "kubectl label nodes <node-name> gpu-type=<gpu-type>"
        
        # 显示所有节点
        print_status "当前集群节点："
        kubectl get nodes -o wide
    else
        print_status "找到GPU节点："
        echo "$GPU_NODES"
    fi
}

# 验证NFS存储
verify_nfs_storage() {
    print_status "验证NFS存储配置..."
    
    # 检查PVC状态
    PVC_STATUS=$(kubectl get pvc jupyterhub-nfs-pvc -n $NAMESPACE -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
    
    if [ "$PVC_STATUS" = "Bound" ]; then
        print_status "NFS PVC已绑定"
    elif [ "$PVC_STATUS" = "Pending" ]; then
        print_warning "NFS PVC待绑定，请检查NFS服务器: $NFS_SERVER"
    else
        print_warning "NFS PVC不存在，请确保NFS服务器可访问: $NFS_SERVER"
    fi
}

# 部署应用
deploy_application() {
    print_status "部署主应用..."
    
    # 检查Go应用
    if [ -f "src/backend/cmd/main.go" ]; then
        print_status "准备构建Go应用..."
        
        # 设置环境变量
        export JUPYTERHUB_K8S_NAMESPACE=$NAMESPACE
        export NFS_SERVER=$NFS_SERVER
        export NFS_PATH=$NFS_PATH
        export PYTHON_GPU_IMAGE="$REGISTRY/jupyterhub-python-gpu:$VERSION"
        export PYTHON_BASE_IMAGE="$REGISTRY/jupyterhub-python-cpu:$VERSION"
        
        print_status "环境变量已设置："
        print_status "  NAMESPACE: $NAMESPACE"
        print_status "  NFS_SERVER: $NFS_SERVER"
        print_status "  GPU_IMAGE: $PYTHON_GPU_IMAGE"
        print_status "  CPU_IMAGE: $PYTHON_BASE_IMAGE"
    else
        print_error "Go应用源码不存在: src/backend/cmd/main.go"
        exit 1
    fi
}

# 运行测试
run_tests() {
    print_status "运行系统测试..."
    
    # 等待应用启动
    print_status "等待应用启动..."
    sleep 5
    
    # 检查API健康状态
    API_URL="http://localhost:8080"
    if curl -s "${API_URL}/api/v1/jupyterhub/health" &> /dev/null; then
        print_status "API健康检查通过"
    else
        print_warning "API健康检查失败，应用可能未启动"
    fi
    
    # 运行Python客户端测试
    if [ -f "examples/jupyterhub_k8s_client.py" ]; then
        print_status "运行Python客户端测试..."
        python3 examples/jupyterhub_k8s_client.py --url $API_URL --task status
    else
        print_warning "Python客户端示例不存在"
    fi
}

# 显示部署信息
show_deployment_info() {
    print_status "部署完成！"
    echo ""
    print_status "系统信息："
    print_status "  命名空间: $NAMESPACE"
    print_status "  Docker镜像注册表: $REGISTRY"
    print_status "  NFS服务器: $NFS_SERVER"
    print_status "  API地址: http://localhost:8080"
    
    echo ""
    print_status "可用的API端点："
    echo "  GET  /api/v1/jupyterhub/gpu/status        - 获取GPU资源状态"
    echo "  GET  /api/v1/jupyterhub/gpu/nodes         - 查找GPU节点"
    echo "  POST /api/v1/jupyterhub/jobs/submit       - 提交Python脚本"
    echo "  GET  /api/v1/jupyterhub/jobs/{name}/status - 获取作业状态"
    echo "  GET  /api/v1/jupyterhub/health             - 健康检查"
    
    echo ""
    print_status "使用示例："
    echo "  # 检查GPU状态"
    echo "  curl http://localhost:8080/api/v1/jupyterhub/gpu/status"
    echo ""
    echo "  # 运行测试任务"
    echo "  python3 examples/jupyterhub_k8s_client.py --task gpu --wait"
    
    echo ""
    print_status "监控命令："
    echo "  # 查看作业"
    echo "  kubectl get jobs -n $NAMESPACE"
    echo ""
    echo "  # 查看Pod"
    echo "  kubectl get pods -n $NAMESPACE"
    echo ""
    echo "  # 查看日志"
    echo "  kubectl logs -f job/<job-name> -n $NAMESPACE"
}

# 清理函数
cleanup() {
    if [ "$CLEANUP" = "true" ]; then
        print_status "清理部署资源..."
        kubectl delete namespace $NAMESPACE --ignore-not-found=true
        print_status "清理完成"
    fi
}

# 显示使用说明
show_usage() {
    echo "使用方法:"
    echo "  $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help           显示帮助信息"
    echo "  --skip-build         跳过Docker镜像构建"
    echo "  --skip-k8s           跳过Kubernetes资源部署"
    echo "  --skip-test          跳过测试运行"
    echo "  --cleanup            清理已部署的资源"
    echo "  --dev                开发模式（跳过镜像构建和推送）"
    echo ""
    echo "环境变量:"
    echo "  NAMESPACE            Kubernetes命名空间 (默认: jupyterhub-jobs)"
    echo "  REGISTRY             Docker镜像注册表 (默认: localhost:5000)"
    echo "  VERSION              镜像版本标签 (默认: latest)"
    echo "  NFS_SERVER           NFS服务器地址"
    echo "  NFS_PATH             NFS共享路径 (默认: /shared)"
    echo ""
    echo "示例:"
    echo "  $0                   # 完整部署"
    echo "  $0 --skip-build      # 跳过镜像构建"
    echo "  $0 --dev             # 开发模式"
    echo "  $0 --cleanup         # 清理资源"
}

# 主函数
main() {
    local SKIP_BUILD=false
    local SKIP_K8S=false
    local SKIP_TEST=false
    local CLEANUP=false
    local DEV_MODE=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-k8s)
                SKIP_K8S=true
                shift
                ;;
            --skip-test)
                SKIP_TEST=true
                shift
                ;;
            --cleanup)
                CLEANUP=true
                shift
                ;;
            --dev)
                DEV_MODE=true
                SKIP_BUILD=true
                shift
                ;;
            *)
                print_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 如果是清理模式，直接执行清理
    if [ "$CLEANUP" = "true" ]; then
        cleanup
        exit 0
    fi
    
    print_status "开始部署 JupyterHub K8s GPU集成系统..."
    print_status "配置: NAMESPACE=$NAMESPACE, REGISTRY=$REGISTRY, VERSION=$VERSION"
    
    # 执行部署步骤
    check_dependencies
    
    if [ "$SKIP_BUILD" = "false" ]; then
        build_images
    fi
    
    if [ "$SKIP_K8S" = "false" ]; then
        deploy_k8s_resources
        configure_gpu_nodes
        verify_nfs_storage
    fi
    
    deploy_application
    
    if [ "$SKIP_TEST" = "false" ]; then
        run_tests
    fi
    
    show_deployment_info
    
    print_status "部署脚本执行完成！"
}

# 设置清理陷阱
trap cleanup EXIT

# 运行主函数
main "$@"
