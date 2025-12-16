#!/bin/bash
# =============================================================================
# Node Exporter 安装脚本模板
# 此脚本由 build.sh render 自动生成，变量已被替换为实际值
# =============================================================================
# 生成时间: 由 build.sh sync 命令渲染
#
# 变量说明:
#   192.168.48.123 - AppHub 外部访问地址
#   28080 - AppHub 端口 (默认 28080)
#   v1.8.2 - Node Exporter 版本
#   nightingale - Nightingale 服务地址
#   17000 - Nightingale 端口
# =============================================================================

set -eo pipefail

# ===========================================
# 环境配置 (已由模板渲染替换)
# ===========================================
NODE_EXPORTER_VERSION="${NODE_EXPORTER_VERSION:-v1.8.2}"
APPHUB_URL="${APPHUB_URL:-http://192.168.48.123:28080}"
NIGHTINGALE_HOST="${NIGHTINGALE_HOST:-192.168.48.123}"
NIGHTINGALE_PORT="${NIGHTINGALE_PORT:-17000}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
SERVICE_USER="${SERVICE_USER:-node_exporter}"
TEXTFILE_DIR="${TEXTFILE_DIR:-/var/lib/node_exporter/textfile_collector}"
LISTEN_PORT="${LISTEN_PORT:-9100}"

# ===========================================
# 颜色输出
# ===========================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ===========================================
# 帮助信息
# ===========================================
usage() {
    cat <<EOF
Node Exporter 安装脚本 (预配置模板)

此脚本已预配置以下默认值:
  Node Exporter 版本: v1.8.2
  AppHub URL: http://192.168.48.123:28080
  Nightingale: 192.168.48.123:17000

用法: $0 [OPTIONS]

选项:
    -h, --help              显示帮助信息
    --version VERSION       覆盖 Node Exporter 版本
    --apphub-url URL        覆盖 AppHub URL
    --port PORT             监听端口 (默认: 9100)
    --use-official          从官方 GitHub 下载

环境变量:
    NODE_EXPORTER_VERSION   Node Exporter 版本
    APPHUB_URL              AppHub URL
    LISTEN_PORT             监听端口

示例:
    # 使用预配置的默认值安装
    $0

    # 指定监听端口
    $0 --port 9101

    # 使用 curl 管道安装
    curl -fsSL http://192.168.48.123:28080/packages/install-node-exporter.sh | bash
EOF
    exit 0
}

# ===========================================
# 参数解析
# ===========================================
USE_OFFICIAL="false"

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            --version)
                NODE_EXPORTER_VERSION="$2"
                shift 2
                ;;
            --apphub-url)
                APPHUB_URL="$2"
                shift 2
                ;;
            --port)
                LISTEN_PORT="$2"
                shift 2
                ;;
            --use-official)
                USE_OFFICIAL="true"
                shift
                ;;
            -*)
                log_error "未知选项: $1"
                usage
                ;;
            *)
                shift
                ;;
        esac
    done
}

# ===========================================
# 系统检测
# ===========================================
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) 
            log_error "不支持的架构: $arch"
            exit 1
            ;;
    esac
}

detect_init_system() {
    if command -v systemctl &> /dev/null && pidof systemd &> /dev/null; then
        echo "systemd"
    elif command -v rc-service &> /dev/null; then
        echo "openrc"
    elif [[ -f /etc/init.d/functions ]]; then
        echo "sysvinit"
    else
        echo "unknown"
    fi
}

# ===========================================
# 检查 AppHub 可用性
# ===========================================
check_apphub() {
    local url="$1"
    
    if [[ -z "$url" ]] || [[ "$url" == *"192.168.48.123"* ]]; then
        return 1
    fi
    
    log_info "检查 AppHub 可用性: $url"
    
    if command -v curl &> /dev/null; then
        if timeout 10 curl -fsL "${url}/pkgs/node_exporter/VERSION" &>/dev/null; then
            return 0
        fi
    fi
    
    return 1
}

# ===========================================
# 下载并安装
# ===========================================
download_and_install() {
    local arch=$(detect_arch)
    local filename="node_exporter-${NODE_EXPORTER_VERSION}.linux-${arch}.tar.gz"
    local download_url
    local temp_dir=$(mktemp -d)
    
    # 确定下载 URL
    if [[ "$USE_OFFICIAL" == "true" ]]; then
        download_url="https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/${filename}"
        log_info "从官方 GitHub 下载..."
    elif check_apphub "$APPHUB_URL"; then
        download_url="${APPHUB_URL}/pkgs/node_exporter/${filename}"
        log_info "从 AppHub 下载..."
    else
        download_url="https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/${filename}"
        log_warn "AppHub 不可用，使用官方 GitHub..."
    fi
    
    log_info "下载 URL: $download_url"
    
    # 下载
    cd "$temp_dir"
    if command -v wget &> /dev/null; then
        wget -q --show-progress -O "$filename" "$download_url" || curl -fsSL -o "$filename" "$download_url"
    else
        curl -fsSL -o "$filename" "$download_url"
    fi
    
    # 解压
    tar xzf "$filename"
    
    # 安装二进制
    local extract_dir="node_exporter-${NODE_EXPORTER_VERSION}.linux-${arch}"
    if [[ ! -d "$extract_dir" ]]; then
        # 尝试其他可能的目录名
        extract_dir=$(ls -d node_exporter-* 2>/dev/null | head -1)
    fi
    
    cp "${extract_dir}/node_exporter" "${INSTALL_DIR}/node_exporter"
    chmod +x "${INSTALL_DIR}/node_exporter"
    
    # 清理
    cd /
    rm -rf "$temp_dir"
    
    log_info "Node Exporter 已安装到: ${INSTALL_DIR}/node_exporter"
}

# ===========================================
# 创建用户
# ===========================================
create_user() {
    if ! id "$SERVICE_USER" &>/dev/null; then
        log_info "创建用户: $SERVICE_USER"
        useradd --no-create-home --shell /bin/false "$SERVICE_USER" 2>/dev/null || \
        adduser -S -D -H -s /sbin/nologin "$SERVICE_USER" 2>/dev/null || true
    else
        log_info "用户 $SERVICE_USER 已存在"
    fi
}

# ===========================================
# 创建服务
# ===========================================
create_systemd_service() {
    log_info "创建 systemd 服务..."
    
    cat > /etc/systemd/system/node_exporter.service << EOF
[Unit]
Description=Node Exporter - Prometheus exporter for hardware and OS metrics
Documentation=https://github.com/prometheus/node_exporter
Wants=network-online.target
After=network-online.target

[Service]
User=${SERVICE_USER}
Group=${SERVICE_USER}
Type=simple
Restart=always
RestartSec=5
ExecStart=${INSTALL_DIR}/node_exporter \\
    --web.listen-address=:${LISTEN_PORT} \\
    --collector.textfile.directory=${TEXTFILE_DIR}

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable node_exporter
}

create_openrc_service() {
    log_info "创建 OpenRC 服务..."
    
    cat > /etc/init.d/node_exporter << EOF
#!/sbin/openrc-run

name="node_exporter"
description="Prometheus Node Exporter"
command="${INSTALL_DIR}/node_exporter"
command_args="--web.listen-address=:${LISTEN_PORT} --collector.textfile.directory=${TEXTFILE_DIR}"
command_user="${SERVICE_USER}"
command_background="yes"
pidfile="/run/\${RC_SVCNAME}.pid"

depend() {
    need net
}
EOF
    
    chmod +x /etc/init.d/node_exporter
    rc-update add node_exporter default 2>/dev/null || true
}

# ===========================================
# 启动服务
# ===========================================
start_service() {
    local init_system=$(detect_init_system)
    
    log_info "启动 Node Exporter 服务..."
    
    case "$init_system" in
        systemd)
            systemctl restart node_exporter
            sleep 2
            if systemctl is-active --quiet node_exporter; then
                log_info "服务已启动 (systemd)"
                return 0
            fi
            ;;
        openrc)
            rc-service node_exporter restart
            if rc-service node_exporter status &>/dev/null; then
                log_info "服务已启动 (openrc)"
                return 0
            fi
            ;;
        *)
            log_warn "未知的 init 系统，尝试直接启动..."
            nohup ${INSTALL_DIR}/node_exporter \
                --web.listen-address=:${LISTEN_PORT} \
                --collector.textfile.directory=${TEXTFILE_DIR} \
                > /var/log/node_exporter.log 2>&1 &
            sleep 2
            if pgrep -f node_exporter &>/dev/null; then
                log_info "服务已启动 (后台进程)"
                return 0
            fi
            ;;
    esac
    
    log_warn "服务可能未正常启动"
    return 1
}

# ===========================================
# 验证安装
# ===========================================
verify_installation() {
    log_info "验证安装..."
    
    # 检查进程
    if pgrep -f node_exporter &>/dev/null; then
        log_info "✓ Node Exporter 进程运行中"
    else
        log_error "✗ Node Exporter 进程未运行"
        return 1
    fi
    
    # 检查端口
    sleep 2
    if curl -s "http://localhost:${LISTEN_PORT}/metrics" | head -5 &>/dev/null; then
        log_info "✓ Metrics 端点可访问"
    else
        log_warn "⚠ Metrics 端点可能未就绪"
    fi
    
    return 0
}

# ===========================================
# 主函数
# ===========================================
main() {
    parse_args "$@"
    
    # 验证版本
    if [[ -z "$NODE_EXPORTER_VERSION" ]] || [[ "$NODE_EXPORTER_VERSION" == "v1.8.2" ]]; then
        log_error "Node Exporter 版本未配置"
        log_error "请确保模板已正确渲染，或使用 --version 参数指定"
        exit 1
    fi
    
    log_info "=========================================="
    log_info "Node Exporter 安装"
    log_info "=========================================="
    log_info "版本: ${NODE_EXPORTER_VERSION}"
    log_info "架构: $(detect_arch)"
    log_info "监听端口: ${LISTEN_PORT}"
    log_info "AppHub: ${APPHUB_URL}"
    echo ""
    
    # 创建必要目录
    mkdir -p "${TEXTFILE_DIR}"
    
    # 安装步骤
    download_and_install
    create_user
    
    # 设置权限
    chown -R "${SERVICE_USER}:${SERVICE_USER}" "${TEXTFILE_DIR}" 2>/dev/null || true
    
    # 创建服务
    local init_system=$(detect_init_system)
    case "$init_system" in
        systemd) create_systemd_service ;;
        openrc) create_openrc_service ;;
        *) log_warn "跳过服务创建 (未知 init 系统)" ;;
    esac
    
    # 启动服务
    start_service
    
    # 验证
    verify_installation
    
    # 输出结果
    echo ""
    log_info "=========================================="
    log_info "✓ Node Exporter 安装完成"
    log_info "=========================================="
    log_info "版本: ${NODE_EXPORTER_VERSION}"
    log_info "端口: ${LISTEN_PORT}"
    log_info "Metrics: http://localhost:${LISTEN_PORT}/metrics"
    echo ""
    log_info "配置 Prometheus/Nightingale 抓取:"
    log_info "  - job_name: 'node_exporter'"
    log_info "    static_configs:"
    log_info "      - targets: ['<host>:${LISTEN_PORT}']"
    log_info "=========================================="
}

main "$@"
