#!/bin/bash
# =============================================================================
# Prometheus 安装脚本模板
# 此脚本由 build.sh render 自动生成，变量已被替换为实际值
# =============================================================================
# 生成时间: 由 build.sh sync 命令渲染
#
# 变量说明:
#   localhost - AppHub 外部访问地址
#   28080 - AppHub 端口 (默认 28080)
#   v3.4.1 - Prometheus 版本
# =============================================================================

set -eo pipefail

# ===========================================
# 环境配置 (已由模板渲染替换)
# ===========================================
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-v3.4.1}"
APPHUB_URL="${APPHUB_URL:-http://localhost:28080}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/prometheus}"
DATA_DIR="${DATA_DIR:-/var/lib/prometheus}"
CONFIG_DIR="${CONFIG_DIR:-/etc/prometheus}"
SERVICE_USER="${SERVICE_USER:-prometheus}"
LISTEN_PORT="${LISTEN_PORT:-9090}"
RETENTION_TIME="${RETENTION_TIME:-30d}"

# ===========================================
# 颜色输出
# ===========================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $*"; }

# ===========================================
# 帮助信息
# ===========================================
usage() {
    cat <<EOF
Prometheus 安装脚本 (预配置模板)

此脚本已预配置以下默认值:
  Prometheus 版本: v3.4.1
  AppHub URL: http://localhost:28080

用法: $0 [OPTIONS]

选项:
    -h, --help              显示帮助信息
    --version VERSION       覆盖 Prometheus 版本
    --apphub-url URL        覆盖 AppHub URL
    --port PORT             监听端口 (默认: 9090)
    --retention TIME        数据保留时间 (默认: 30d)
    --use-official          从官方 GitHub 下载
    --config-only           仅更新配置，不重新安装

环境变量:
    PROMETHEUS_VERSION      Prometheus 版本
    APPHUB_URL              AppHub URL
    LISTEN_PORT             监听端口
    RETENTION_TIME          数据保留时间

示例:
    # 使用预配置的默认值安装
    $0

    # 指定监听端口和保留时间
    $0 --port 9091 --retention 15d

    # 使用 curl 管道安装
    curl -fsSL http://localhost:28080/scripts/install-prometheus.sh | bash
EOF
    exit 0
}

# ===========================================
# 参数解析
# ===========================================
USE_OFFICIAL="false"
CONFIG_ONLY="false"

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            --version)
                PROMETHEUS_VERSION="$2"
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
            --retention)
                RETENTION_TIME="$2"
                shift 2
                ;;
            --use-official)
                USE_OFFICIAL="true"
                shift
                ;;
            --config-only)
                CONFIG_ONLY="true"
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
    
    if [[ -z "$url" ]] || [[ "$url" == *"localhost"* ]]; then
        return 1
    fi
    
    log_info "检查 AppHub 可用性: $url"
    
    if command -v curl &> /dev/null; then
        if timeout 10 curl -fsL "${url}/pkgs/prometheus/VERSION" &>/dev/null; then
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
    local filename="prometheus-${PROMETHEUS_VERSION}.linux-${arch}.tar.gz"
    local download_url
    local temp_dir=$(mktemp -d)
    
    # 确定下载 URL
    if [[ "$USE_OFFICIAL" == "true" ]]; then
        download_url="https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${filename}"
        log_info "从官方 GitHub 下载..."
    elif check_apphub "$APPHUB_URL"; then
        download_url="${APPHUB_URL}/pkgs/prometheus/${filename}"
        log_info "从 AppHub 下载..."
    else
        download_url="https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${filename}"
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
    
    # 安装二进制文件
    local extract_dir="prometheus-${PROMETHEUS_VERSION}.linux-${arch}"
    if [[ ! -d "$extract_dir" ]]; then
        extract_dir=$(ls -d prometheus-* 2>/dev/null | head -1)
    fi
    
    # 创建目录
    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${DATA_DIR}"
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${CONFIG_DIR}/rules"
    mkdir -p "${CONFIG_DIR}/targets"
    
    # 复制二进制文件
    cp "${extract_dir}/prometheus" "${INSTALL_DIR}/prometheus"
    cp "${extract_dir}/promtool" "${INSTALL_DIR}/promtool"
    chmod +x "${INSTALL_DIR}/prometheus" "${INSTALL_DIR}/promtool"
    
    # 复制控制台文件
    cp -r "${extract_dir}/consoles" "${CONFIG_DIR}/"
    cp -r "${extract_dir}/console_libraries" "${CONFIG_DIR}/"
    
    # 清理
    cd /
    rm -rf "$temp_dir"
    
    log_info "Prometheus 已安装到: ${INSTALL_DIR}"
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
# 创建配置文件
# ===========================================
create_config() {
    log_step "创建 Prometheus 配置文件..."
    
    # 如果配置文件已存在，备份
    if [[ -f "${CONFIG_DIR}/prometheus.yml" ]]; then
        local backup_file="${CONFIG_DIR}/prometheus.yml.bak.$(date +%Y%m%d%H%M%S)"
        cp "${CONFIG_DIR}/prometheus.yml" "${backup_file}"
        log_info "已备份原配置到: ${backup_file}"
    fi
    
    cat > "${CONFIG_DIR}/prometheus.yml" << 'EOF'
# Prometheus 配置文件
# 由安装脚本自动生成

global:
  scrape_interval: 15s          # 默认抓取间隔
  evaluation_interval: 15s      # 规则评估间隔
  external_labels:
    monitor: 'ai-infra-matrix'

# 告警管理器配置 (可选)
# alerting:
#   alertmanagers:
#     - static_configs:
#         - targets:
#           - localhost:9093

# 规则文件
rule_files:
  # - "rules/*.yml"

# 抓取配置
scrape_configs:
  # Prometheus 自身监控
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
    
  # Node Exporter
  - job_name: 'node_exporter'
    file_sd_configs:
      - files:
          - '/etc/prometheus/targets/node_exporter.yml'
        refresh_interval: 30s
    static_configs:
      - targets: ['localhost:9100']
        labels:
          instance: 'localhost'
    
  # Categraf (如果启用了 Prometheus 端点)
  - job_name: 'categraf'
    file_sd_configs:
      - files:
          - '/etc/prometheus/targets/categraf.yml'
        refresh_interval: 30s

# 远程写入到 VictoriaMetrics (可选)
# remote_write:
#   - url: "http://victoriametrics:8428/api/v1/write"
#     queue_config:
#       max_samples_per_send: 10000
#       batch_send_deadline: 5s
#       capacity: 100000
EOF

    # 创建目标发现文件模板
    cat > "${CONFIG_DIR}/targets/node_exporter.yml" << 'EOF'
# Node Exporter 目标配置
# 添加更多目标：
# - targets: ['host1:9100', 'host2:9100']
#   labels:
#     group: 'production'
[]
EOF

    cat > "${CONFIG_DIR}/targets/categraf.yml" << 'EOF'
# Categraf 目标配置
# 添加更多目标：
# - targets: ['host1:9100', 'host2:9100']
#   labels:
#     group: 'production'
[]
EOF

    log_info "配置文件已创建: ${CONFIG_DIR}/prometheus.yml"
}

# ===========================================
# 创建服务
# ===========================================
create_systemd_service() {
    log_step "创建 systemd 服务..."
    
    cat > /etc/systemd/system/prometheus.service << EOF
[Unit]
Description=Prometheus Monitoring System
Documentation=https://prometheus.io/docs/
Wants=network-online.target
After=network-online.target

[Service]
User=${SERVICE_USER}
Group=${SERVICE_USER}
Type=simple
Restart=always
RestartSec=5
ExecReload=/bin/kill -HUP \$MAINPID
ExecStart=${INSTALL_DIR}/prometheus \\
    --config.file=${CONFIG_DIR}/prometheus.yml \\
    --storage.tsdb.path=${DATA_DIR} \\
    --storage.tsdb.retention.time=${RETENTION_TIME} \\
    --web.console.templates=${CONFIG_DIR}/consoles \\
    --web.console.libraries=${CONFIG_DIR}/console_libraries \\
    --web.listen-address=0.0.0.0:${LISTEN_PORT} \\
    --web.enable-lifecycle \\
    --web.enable-admin-api

LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable prometheus
    log_info "systemd 服务已创建"
}

create_openrc_service() {
    log_step "创建 OpenRC 服务..."
    
    cat > /etc/init.d/prometheus << EOF
#!/sbin/openrc-run

name="prometheus"
description="Prometheus Monitoring System"
command="${INSTALL_DIR}/prometheus"
command_args="--config.file=${CONFIG_DIR}/prometheus.yml --storage.tsdb.path=${DATA_DIR} --storage.tsdb.retention.time=${RETENTION_TIME} --web.listen-address=0.0.0.0:${LISTEN_PORT}"
command_user="${SERVICE_USER}"
command_background="yes"
pidfile="/run/\${RC_SVCNAME}.pid"

depend() {
    need net
}
EOF
    
    chmod +x /etc/init.d/prometheus
    rc-update add prometheus default 2>/dev/null || true
    log_info "OpenRC 服务已创建"
}

# ===========================================
# 启动服务
# ===========================================
start_service() {
    local init_system=$(detect_init_system)
    
    log_step "启动 Prometheus 服务..."
    
    case "$init_system" in
        systemd)
            systemctl restart prometheus
            sleep 3
            if systemctl is-active --quiet prometheus; then
                log_info "服务已启动 (systemd)"
                return 0
            fi
            ;;
        openrc)
            rc-service prometheus restart
            if rc-service prometheus status &>/dev/null; then
                log_info "服务已启动 (openrc)"
                return 0
            fi
            ;;
        *)
            log_warn "未知的 init 系统，尝试直接启动..."
            nohup ${INSTALL_DIR}/prometheus \
                --config.file=${CONFIG_DIR}/prometheus.yml \
                --storage.tsdb.path=${DATA_DIR} \
                --storage.tsdb.retention.time=${RETENTION_TIME} \
                --web.listen-address=0.0.0.0:${LISTEN_PORT} \
                > /var/log/prometheus.log 2>&1 &
            sleep 3
            if pgrep -f prometheus &>/dev/null; then
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
    log_step "验证安装..."
    
    # 检查进程
    if pgrep -f "prometheus.*--config" &>/dev/null; then
        log_info "✓ Prometheus 进程运行中"
    else
        log_error "✗ Prometheus 进程未运行"
        return 1
    fi
    
    # 检查 Web UI
    sleep 2
    if curl -s "http://localhost:${LISTEN_PORT}/-/ready" | grep -q "Prometheus Server is Ready"; then
        log_info "✓ Prometheus Web UI 可访问"
    else
        # 尝试备用检查
        if curl -s "http://localhost:${LISTEN_PORT}/api/v1/status/config" | grep -q "status"; then
            log_info "✓ Prometheus API 可访问"
        else
            log_warn "⚠ Prometheus 可能未完全就绪"
        fi
    fi
    
    # 检查配置
    if ${INSTALL_DIR}/promtool check config ${CONFIG_DIR}/prometheus.yml &>/dev/null; then
        log_info "✓ 配置文件语法正确"
    else
        log_warn "⚠ 配置文件可能有问题"
    fi
    
    return 0
}

# ===========================================
# 主函数
# ===========================================
main() {
    parse_args "$@"
    
    # 验证版本
    if [[ -z "$PROMETHEUS_VERSION" ]] || [[ "$PROMETHEUS_VERSION" == "v3.4.1" ]]; then
        log_error "Prometheus 版本未配置"
        log_error "请确保模板已正确渲染，或使用 --version 参数指定"
        exit 1
    fi
    
    log_info "=========================================="
    log_info "Prometheus 安装脚本"
    log_info "=========================================="
    log_info "版本: ${PROMETHEUS_VERSION}"
    log_info "架构: $(detect_arch)"
    log_info "监听端口: ${LISTEN_PORT}"
    log_info "数据保留: ${RETENTION_TIME}"
    log_info "AppHub: ${APPHUB_URL}"
    echo ""
    
    if [[ "$CONFIG_ONLY" == "true" ]]; then
        log_info "仅更新配置模式..."
        create_config
        start_service
        verify_installation
    else
        # 完整安装
        download_and_install
        create_user
        
        # 设置权限
        chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}" 2>/dev/null || true
        chown -R "${SERVICE_USER}:${SERVICE_USER}" "${DATA_DIR}" 2>/dev/null || true
        chown -R "${SERVICE_USER}:${SERVICE_USER}" "${CONFIG_DIR}" 2>/dev/null || true
        
        # 创建配置
        create_config
        
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
    fi
    
    # 输出结果
    echo ""
    log_info "=========================================="
    log_info "✓ Prometheus 安装完成"
    log_info "=========================================="
    log_info "版本: ${PROMETHEUS_VERSION}"
    log_info "端口: ${LISTEN_PORT}"
    log_info "Web UI: http://localhost:${LISTEN_PORT}"
    log_info "配置文件: ${CONFIG_DIR}/prometheus.yml"
    log_info "数据目录: ${DATA_DIR}"
    log_info "保留时间: ${RETENTION_TIME}"
    echo ""
    log_info "常用命令:"
    log_info "  systemctl status prometheus    # 查看状态"
    log_info "  systemctl restart prometheus   # 重启服务"
    log_info "  promtool check config ${CONFIG_DIR}/prometheus.yml  # 检查配置"
    echo ""
    log_info "添加抓取目标:"
    log_info "  编辑 ${CONFIG_DIR}/targets/node_exporter.yml"
    log_info "  或编辑 ${CONFIG_DIR}/prometheus.yml"
    log_info "=========================================="
}

main "$@"
