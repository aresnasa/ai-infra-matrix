#!/bin/bash

# JupyterHub 启动脚本 - AI基础设施矩阵统一认证版
# 用法: ./start-jupyterhub.sh [setup|start|daemon|stop|status|logs|restart]

set -e

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
JUPYTERHUB_DIR="$PROJECT_ROOT/src/jupyterhub"
DATA_DIR="$PROJECT_ROOT/data"
LOG_DIR="$PROJECT_ROOT/log"
CONFIG_FILE="$JUPYTERHUB_DIR/ai_infra_jupyterhub_config.py"
CONDA_ENV="ai-infra-matrix"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 加载环境配置
ENV_FILE="$PROJECT_ROOT/.env.jupyterhub"
if [ -f "$ENV_FILE" ]; then
    print_info "加载环境配置: $ENV_FILE"
    export $(grep -v '^#' "$ENV_FILE" | xargs)
else
    print_warning "未找到环境配置文件，使用默认配置"
fi

# 环境变量配置（默认值）
export AI_INFRA_BACKEND_URL="${AI_INFRA_BACKEND_URL:-http://localhost:8080}"
export AI_INFRA_API_TOKEN="${AI_INFRA_API_TOKEN:-ai-infra-default-token}"
export JUPYTERHUB_ADMIN_USERS="${JUPYTERHUB_ADMIN_USERS:-admin,jupyter-admin}"
export JUPYTERHUB_API_TOKEN="${JUPYTERHUB_API_TOKEN:-ai-infra-hub-token}"
export JUPYTERHUB_PORT="${JUPYTERHUB_PORT:-8090}"
export JUPYTERHUB_LOG_LEVEL="${JUPYTERHUB_LOG_LEVEL:-INFO}"

# 生成随机token（如果未设置）
if [ -z "$CONFIGPROXY_AUTH_TOKEN" ]; then
    export CONFIGPROXY_AUTH_TOKEN="$(openssl rand -hex 32)"
fi

# 生成加密密钥（如果未设置）
if [ -z "$JUPYTERHUB_CRYPT_KEY" ]; then
    export JUPYTERHUB_CRYPT_KEY="$(openssl rand -hex 32)"
fi

# 激活 conda 环境
activate_conda() {
    print_info "激活 conda 环境: $CONDA_ENV"
    
    # 初始化 conda
    if [ -f "$HOME/miniforge3/etc/profile.d/conda.sh" ]; then
        source "$HOME/miniforge3/etc/profile.d/conda.sh"
    elif [ -f "$HOME/miniconda3/etc/profile.d/conda.sh" ]; then
        source "$HOME/miniconda3/etc/profile.d/conda.sh"
    elif [ -f "$HOME/anaconda3/etc/profile.d/conda.sh" ]; then
        source "$HOME/anaconda3/etc/profile.d/conda.sh"
    else
        print_error "找不到 conda 安装"
        exit 1
    fi
    
    # 激活环境
    conda activate "$CONDA_ENV" || {
        print_error "无法激活 conda 环境: $CONDA_ENV"
        print_info "请确保环境存在: conda env list"
        exit 1
    }
    
    print_success "conda 环境激活成功"
}

# 检查依赖
check_dependencies() {
    print_info "检查依赖..."
    
    # 检查 conda
    if ! command -v conda &> /dev/null; then
        print_error "conda 未安装"
        exit 1
    fi
    
    # 激活环境
    activate_conda
    
    # 检查 Python
    if ! command -v python &> /dev/null; then
        print_error "Python 未在 conda 环境中找到"
        exit 1
    fi
    
    print_success "依赖检查通过"
}

# 安装 JupyterHub 和统一认证组件
install_jupyterhub() {
    print_info "在 conda 环境中安装 JupyterHub 和统一认证组件..."
    
    activate_conda
    
    # 检查是否已安装
    if conda list jupyterhub | grep -q jupyterhub; then
        print_success "JupyterHub 已安装"
    else
        # 使用 conda 安装 JupyterHub 和相关组件
        print_info "使用 conda 安装 JupyterHub..."
        conda install -y -c conda-forge jupyterhub notebook jupyterlab nodejs
        
        # 安装 configurable-http-proxy
        print_info "安装 configurable-http-proxy..."
        npm install -g configurable-http-proxy
    fi
    
    # 安装统一认证所需的Python包
    print_info "安装统一认证依赖..."
    pip install requests tornado
    
    print_success "JupyterHub 和统一认证组件安装完成"
}

# 设置目录
setup_directories() {
    print_info "设置目录结构..."
    
    # 创建必要的目录
    mkdir -p "$DATA_DIR/jupyterhub"
    mkdir -p "$LOG_DIR"
    mkdir -p "$PROJECT_ROOT/notebooks"
    
    print_success "目录结构设置完成"
}

# 生成 cookie secret
generate_secrets() {
    local cookie_secret_file="$DATA_DIR/jupyterhub/cookie_secret"
    
    if [ ! -f "$cookie_secret_file" ]; then
        print_info "生成 cookie secret..."
        openssl rand -hex 32 > "$cookie_secret_file"
        chmod 600 "$cookie_secret_file"
        print_success "cookie secret 生成完成"
    fi
}

# 启动 JupyterHub
start_jupyterhub() {
    local mode="${1:-foreground}"
    
    print_info "启动 JupyterHub..."
    
    # 激活 conda 环境
    activate_conda
    
    # 设置环境变量
    export JUPYTERHUB_DATA_DIR="$DATA_DIR/jupyterhub"
    export JUPYTERHUB_LOG_DIR="$LOG_DIR"
    
    cd "$JUPYTERHUB_DIR"
    
    if [ "$mode" = "daemon" ]; then
        # 后台启动
        print_info "后台启动 JupyterHub..."
        nohup jupyterhub --config="$CONFIG_FILE" \
            --log-file="$LOG_DIR/jupyterhub.log" \
            --log-level=INFO \
            > "$LOG_DIR/jupyterhub.out" 2>&1 &
        
        local pid=$!
        echo $pid > "$DATA_DIR/jupyterhub/jupyterhub.pid"
        
        print_success "JupyterHub 已在后台启动 (PID: $pid)"
        print_info "日志文件: $LOG_DIR/jupyterhub.log"
        print_info "访问地址: http://localhost:8090"
    else
        # 前台启动
        print_info "启动 JupyterHub (前台模式)..."
        print_info "访问地址: http://localhost:8090"
        print_info "按 Ctrl+C 停止服务"
        
        jupyterhub --config="$CONFIG_FILE" \
            --log-file="$LOG_DIR/jupyterhub.log" \
            --log-level=INFO
    fi
}

# 停止 JupyterHub
stop_jupyterhub() {
    print_info "停止 JupyterHub..."
    
    local pid_file="$DATA_DIR/jupyterhub/jupyterhub.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "$pid_file"
            print_success "JupyterHub 已停止 (PID: $pid)"
        else
            print_warning "JupyterHub 进程不存在"
            rm -f "$pid_file"
        fi
    else
        # 尝试通过进程名称停止
        local pids=$(pgrep -f "jupyterhub" || true)
        if [ -n "$pids" ]; then
            echo "$pids" | xargs kill
            print_success "JupyterHub 进程已停止"
        else
            print_warning "未找到运行中的 JupyterHub 进程"
        fi
    fi
}

# 查看状态
show_status() {
    print_info "检查 JupyterHub 状态..."
    
    local pid_file="$DATA_DIR/jupyterhub/jupyterhub.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            print_success "JupyterHub 正在运行 (PID: $pid)"
            print_info "访问地址: http://localhost:8090"
        else
            print_warning "PID 文件存在但进程未运行"
            rm -f "$pid_file"
        fi
    else
        local pids=$(pgrep -f "jupyterhub" || true)
        if [ -n "$pids" ]; then
            print_warning "JupyterHub 进程运行中但无 PID 文件:"
            echo "$pids"
        else
            print_info "JupyterHub 未运行"
        fi
    fi
}

# 查看日志
show_logs() {
    local log_file="$LOG_DIR/jupyterhub.log"
    
    if [ -f "$log_file" ]; then
        print_info "JupyterHub 日志 (最近 50 行):"
        echo "===================="
        tail -50 "$log_file"
        echo "===================="
        print_info "完整日志文件: $log_file"
    else
        print_warning "日志文件不存在: $log_file"
    fi
}

# 显示帮助
show_help() {
    echo "JupyterHub 管理脚本 (使用 conda 环境: $CONDA_ENV)"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  setup     - 安装依赖和设置环境"
    echo "  start     - 启动 JupyterHub (前台)"
    echo "  daemon    - 启动 JupyterHub (后台)"
    echo "  stop      - 停止 JupyterHub"
    echo "  status    - 查看运行状态"
    echo "  logs      - 查看日志"
    echo "  restart   - 重启 JupyterHub"
    echo ""
    echo "示例:"
    echo "  $0 setup     # 首次使用时运行"
    echo "  $0 daemon    # 后台启动服务"
    echo "  $0 status    # 检查运行状态"
    echo "  $0 logs      # 查看日志"
    echo "  $0 stop      # 停止服务"
    echo ""
    echo "注意: 确保已激活 conda 环境 '$CONDA_ENV'"
}

# 主程序
case "${1:-help}" in
    setup)
        check_dependencies
        install_jupyterhub
        setup_directories
        generate_secrets
        print_success "环境设置完成！"
        print_info "运行 '$0 daemon' 启动服务"
        ;;
    start)
        setup_directories
        generate_secrets
        start_jupyterhub
        ;;
    daemon)
        setup_directories
        generate_secrets
        start_jupyterhub daemon
        ;;
    stop)
        stop_jupyterhub
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    restart)
        stop_jupyterhub
        sleep 2
        setup_directories
        generate_secrets
        start_jupyterhub daemon
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "未知命令: $1"
        show_help
        exit 1
        ;;
esac
