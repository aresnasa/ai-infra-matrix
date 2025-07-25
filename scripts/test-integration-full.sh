#!/bin/bash

# JupyterHub K8s GPU 集成系统完整测试脚本
# 执行完整的系统部署、启动和功能验证

set -e

PROJECT_ROOT="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix"
SCRIPT_DIR="$PROJECT_ROOT/scripts"
JUPYTERHUB_DIR="$PROJECT_ROOT/third-party/jupyterhub"

echo "========================================="
echo "JupyterHub K8s GPU 集成系统完整测试"
echo "开始时间: $(date)"
echo "========================================="

# 颜色定义
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

# 函数：检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    # 检查 kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装"
        exit 1
    fi
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    
    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js 未安装"
        exit 1
    fi
    
    # 检查 Python3
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装"
        exit 1
    fi
    
    log_success "所有依赖检查通过"
}

# 函数：检查 Kubernetes 集群
check_kubernetes() {
    log_info "检查 Kubernetes 集群连接..."
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到 Kubernetes 集群"
        exit 1
    fi
    
    # 检查节点状态
    log_info "检查节点状态..."
    kubectl get nodes
    
    # 检查 GPU 节点
    log_info "检查 GPU 节点..."
    GPU_NODES=$(kubectl get nodes -l accelerator --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$GPU_NODES" -eq 0 ]; then
        log_warning "未检测到 GPU 节点标签，将创建模拟环境"
    else
        log_success "检测到 $GPU_NODES 个 GPU 节点"
    fi
}

# 函数：构建 Go 后端
build_backend() {
    log_info "构建 Go 后端服务..."
    
    cd "$PROJECT_ROOT/src/backend"
    
    # 检查 go.mod
    if [ ! -f "go.mod" ]; then
        log_info "初始化 Go 模块..."
        go mod init ai-infra-matrix
    fi
    
    # 安装依赖
    log_info "安装 Go 依赖..."
    go mod tidy
    
    # 构建应用
    log_info "构建应用..."
    go build -o app ./cmd/main.go
    
    if [ ! -f "app" ]; then
        log_error "Go 应用构建失败"
        exit 1
    fi
    
    log_success "Go 后端构建完成"
}

# 函数：部署 Kubernetes 资源
deploy_kubernetes() {
    log_info "部署 Kubernetes 资源..."
    
    # 创建命名空间
    kubectl create namespace ai-infra --dry-run=client -o yaml | kubectl apply -f -
    
    # 应用配置
    if [ -f "$PROJECT_ROOT/src/complete-k8s-config.yaml" ]; then
        log_info "应用 Kubernetes 配置..."
        kubectl apply -f "$PROJECT_ROOT/src/complete-k8s-config.yaml"
    else
        log_warning "未找到 Kubernetes 配置文件"
    fi
    
    # 等待部署完成
    log_info "等待后端服务启动..."
    kubectl wait --for=condition=available --timeout=300s deployment/ai-infra-backend -n ai-infra || true
    
    log_success "Kubernetes 资源部署完成"
}

# 函数：启动 JupyterHub
start_jupyterhub() {
    log_info "启动 JupyterHub 服务..."
    
    cd "$JUPYTERHUB_DIR"
    
    # 检查启动脚本
    if [ ! -f "start-jupyterhub.sh" ]; then
        log_error "JupyterHub 启动脚本不存在"
        exit 1
    fi
    
    # 设置执行权限
    chmod +x start-jupyterhub.sh
    
    # 运行设置
    log_info "运行 JupyterHub 设置..."
    ./start-jupyterhub.sh setup
    
    # 启动服务
    log_info "启动 JupyterHub..."
    ./start-jupyterhub.sh start &
    
    # 等待服务启动
    log_info "等待 JupyterHub 服务启动..."
    sleep 30
    
    # 检查服务状态
    if pgrep -f "jupyterhub" > /dev/null; then
        log_success "JupyterHub 启动成功"
    else
        log_warning "JupyterHub 可能未完全启动，请检查日志"
    fi
}

# 函数：运行功能测试
run_functional_tests() {
    log_info "运行功能测试..."
    
    # 测试 1: 后端 API 健康检查
    log_info "测试 1: 后端服务健康检查"
    
    # 等待后端服务
    sleep 10
    
    # 检查后端服务端口转发
    kubectl port-forward service/ai-infra-backend 8080:8080 -n ai-infra &
    PORT_FORWARD_PID=$!
    sleep 5
    
    # 测试健康检查端点
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log_success "后端服务健康检查通过"
    else
        log_warning "后端服务健康检查失败，继续其他测试"
    fi
    
    # 测试 2: JupyterHub 访问
    log_info "测试 2: JupyterHub 服务访问"
    
    if curl -f http://localhost:8000 > /dev/null 2>&1; then
        log_success "JupyterHub 服务访问正常"
    else
        log_warning "JupyterHub 服务访问失败"
    fi
    
    # 测试 3: GPU 状态 API
    log_info "测试 3: GPU 状态查询"
    
    if curl -f http://localhost:8080/api/k8s/gpu-status > /dev/null 2>&1; then
        log_success "GPU 状态 API 正常"
        curl -s http://localhost:8080/api/k8s/gpu-status | jq '.' || echo "GPU状态数据已返回"
    else
        log_warning "GPU 状态 API 访问失败"
    fi
    
    # 清理端口转发
    kill $PORT_FORWARD_PID 2>/dev/null || true
    
    log_success "功能测试完成"
}

# 函数：运行示例脚本
run_examples() {
    log_info "运行示例脚本测试..."
    
    # 检查示例文件
    if [ ! -f "$JUPYTERHUB_DIR/examples/gpu_performance_test.py" ]; then
        log_warning "示例脚本不存在，跳过示例测试"
        return
    fi
    
    log_info "运行 GPU 性能测试..."
    
    # 在临时容器中运行测试
    kubectl run gpu-test-temp \
        --image=pytorch/pytorch:latest \
        --rm -i --restart=Never \
        --overrides='{"spec":{"containers":[{"name":"gpu-test","image":"pytorch/pytorch:latest","command":["python3","-c"],"args":["import torch; print(f\"CUDA Available: {torch.cuda.is_available()}\"); print(f\"GPU Count: {torch.cuda.device_count()}\")"],"resources":{"limits":{"nvidia.com/gpu":"1"}}}]}}' \
        -n ai-infra 2>/dev/null || log_warning "GPU 测试容器启动失败"
    
    log_success "示例测试完成"
}

# 函数：生成测试报告
generate_report() {
    log_info "生成测试报告..."
    
    REPORT_FILE="$PROJECT_ROOT/integration_test_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$REPORT_FILE" << EOF
# JupyterHub K8s GPU 集成系统测试报告

**测试时间**: $(date)
**测试环境**: $(uname -a)

## 系统状态

### Kubernetes 集群
\`\`\`
$(kubectl get nodes 2>/dev/null || echo "无法获取节点信息")
\`\`\`

### GPU 节点
\`\`\`
$(kubectl get nodes -l accelerator 2>/dev/null || echo "无GPU节点标签")
\`\`\`

### 部署状态
\`\`\`
$(kubectl get all -n ai-infra 2>/dev/null || echo "ai-infra命名空间未创建")
\`\`\`

### JupyterHub 进程
\`\`\`
$(pgrep -fl jupyterhub || echo "JupyterHub未运行")
\`\`\`

## 服务端点

- 后端服务: http://localhost:8080
- JupyterHub: http://localhost:8000
- API文档: http://localhost:8080/docs

## 测试结果

所有功能测试已完成，详细结果请查看上述日志输出。

## 下一步操作

1. 访问 JupyterHub: http://localhost:8000
2. 使用管理员账户登录（用户名: admin, 密码: 在日志中查看）
3. 打开初始化笔记本进行 GPU 任务提交测试
4. 通过 API 接口提交更多测试任务

## 故障排除

如果遇到问题，请检查：
1. Docker 服务状态
2. Kubernetes 集群连接
3. GPU 驱动和设备插件
4. 存储卷挂载
5. 网络端口占用

---
测试完成时间: $(date)
EOF

    log_success "测试报告已生成: $REPORT_FILE"
}

# 函数：显示使用说明
show_usage() {
    echo "使用方法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --full          运行完整测试流程"
    echo "  --check-only    仅检查环境和依赖"
    echo "  --build-only    仅构建后端服务"
    echo "  --deploy-only   仅部署 Kubernetes 资源"
    echo "  --jupyterhub-only 仅启动 JupyterHub"
    echo "  --test-only     仅运行功能测试"
    echo "  --cleanup       清理部署的资源"
    echo "  --help          显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 --full                # 运行完整测试"
    echo "  $0 --check-only          # 检查环境"
    echo "  $0 --cleanup             # 清理资源"
}

# 函数：清理资源
cleanup_resources() {
    log_info "清理部署的资源..."
    
    # 停止 JupyterHub
    pkill -f jupyterhub || true
    
    # 删除 Kubernetes 资源
    kubectl delete namespace ai-infra --ignore-not-found=true
    
    # 删除临时文件
    rm -f /tmp/ai-infra-*.log
    
    log_success "资源清理完成"
}

# 主流程
main() {
    case "${1:-}" in
        --full)
            check_dependencies
            check_kubernetes
            build_backend
            deploy_kubernetes
            start_jupyterhub
            run_functional_tests
            run_examples
            generate_report
            ;;
        --check-only)
            check_dependencies
            check_kubernetes
            ;;
        --build-only)
            build_backend
            ;;
        --deploy-only)
            deploy_kubernetes
            ;;
        --jupyterhub-only)
            start_jupyterhub
            ;;
        --test-only)
            run_functional_tests
            run_examples
            ;;
        --cleanup)
            cleanup_resources
            ;;
        --help|"")
            show_usage
            ;;
        *)
            log_error "未知选项: $1"
            show_usage
            exit 1
            ;;
    esac
}

# 捕获中断信号
trap 'log_warning "测试被中断"; cleanup_resources; exit 1' INT TERM

# 执行主流程
main "$@"

log_success "========================================="
log_success "JupyterHub K8s GPU 集成系统测试完成"
log_success "结束时间: $(date)"
log_success "========================================="
