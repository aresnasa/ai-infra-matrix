#!/bin/bash
# =============================================================================
# Categraf Installation Script for AI-Infra-Matrix
# 安装 Categraf 监控客户端并配置上报到 Nightingale
#
# Required Environment Variables:
#   HOSTNAME      - Target hostname
#   HOST_IP       - Target host IP  
#   N9E_HOST      - Nightingale server host
#   N9E_PORT      - Nightingale server port
#
# Optional Environment Variables:
#   APPHUB_URL       - AppHub URL for downloading packages
#   GITHUB_MIRROR    - GitHub mirror URL
#   CATEGRAF_VERSION - Categraf version (default from config)
#   INSTALL_DIR      - Installation directory (default: /usr/local/categraf)
# =============================================================================

set -e

# Validate required environment variables
: "${HOSTNAME:?ERROR: HOSTNAME environment variable is required}"
: "${HOST_IP:?ERROR: HOST_IP environment variable is required}"
: "${N9E_HOST:?ERROR: N9E_HOST environment variable is required}"
: "${N9E_PORT:?ERROR: N9E_PORT environment variable is required}"

# Configuration from environment
CATEGRAF_VERSION="${CATEGRAF_VERSION:-v0.4.23}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/categraf}"
USER="${CATEGRAF_USER:-root}"
GROUP="${CATEGRAF_GROUP:-root}"

# 日志函数
log_info() {
    echo "[INFO] $1"
}

log_error() {
    echo "[ERROR] $1" >&2
}

log_success() {
    echo "[SUCCESS] ✓ $1"
}

# 检测架构
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# 检测操作系统
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    elif [ -f /etc/redhat-release ]; then
        echo "rhel"
    else
        echo "unknown"
    fi
}

# 打印帮助
print_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
  --n9e-host HOST       Nightingale server host (required)
  --n9e-port PORT       Nightingale server port (default: 17000)
  --apphub-url URL      AppHub URL for downloading packages
  --hostname NAME       Hostname for monitoring (default: system hostname)
  --host-ip IP          Host IP address (default: auto-detect)
  --version VERSION     Categraf version (default: v0.4.23)
  --help                Show this help message

Example:
  $0 --n9e-host 192.168.0.200 --n9e-port 17000

Environment Variables:
  N9E_HOST              Nightingale server host
  N9E_PORT              Nightingale server port
  APPHUB_URL            AppHub URL
  GITHUB_MIRROR         GitHub mirror URL (e.g., https://ghfast.top)
EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --n9e-host)
                N9E_HOST="$2"
                shift 2
                ;;
            --n9e-port)
                N9E_PORT="$2"
                shift 2
                ;;
            --apphub-url)
                APPHUB_URL="$2"
                shift 2
                ;;
            --hostname)
                HOSTNAME="$2"
                shift 2
                ;;
            --host-ip)
                HOST_IP="$2"
                shift 2
                ;;
            --version)
                CATEGRAF_VERSION="$2"
                shift 2
                ;;
            --help)
                print_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                print_help
                exit 1
                ;;
        esac
    done
}

# 验证必需参数
validate_params() {
    if [ -z "$N9E_HOST" ]; then
        log_error "Nightingale host is required. Use --n9e-host or set N9E_HOST environment variable."
        print_help
        exit 1
    fi
}

# 下载 Categraf
download_categraf() {
    local arch=$(detect_arch)
    local package_name="categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz"
    local download_url=""
    
    # 优先从 AppHub 下载
    if [ -n "$APPHUB_URL" ]; then
        download_url="${APPHUB_URL}/pkgs/categraf/${package_name}"
        log_info "Downloading from AppHub: ${download_url}"
    else
        # 从 GitHub 下载
        if [ -n "$GITHUB_MIRROR" ]; then
            download_url="${GITHUB_MIRROR}/https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/${package_name}"
        else
            download_url="https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/${package_name}"
        fi
        log_info "Downloading from GitHub: ${download_url}"
    fi
    
    cd /tmp
    if ! curl -fsSL -o "${package_name}" "${download_url}"; then
        log_error "Failed to download Categraf"
        exit 1
    fi
    
    log_success "Downloaded ${package_name}"
    
    # 解压
    tar xzf "${package_name}"
    cd "categraf-${CATEGRAF_VERSION}-linux-${arch}" 2>/dev/null || cd categraf-*-linux-${arch} 2>/dev/null || cd categraf
}

# 安装 Categraf
install_categraf() {
    log_info "Installing Categraf to ${INSTALL_DIR}..."
    
    # 创建目录
    mkdir -p ${INSTALL_DIR}/{bin,conf,logs}
    
    # 复制二进制
    if [ -f bin/categraf ]; then
        cp bin/categraf ${INSTALL_DIR}/bin/
    elif [ -f categraf ]; then
        cp categraf ${INSTALL_DIR}/bin/
    fi
    chmod +x ${INSTALL_DIR}/bin/categraf
    
    # 复制配置文件
    if [ -d conf ]; then
        cp -r conf/* ${INSTALL_DIR}/conf/
    fi
    
    log_success "Categraf binaries installed"
}

# 配置 Categraf
configure_categraf() {
    log_info "Configuring Categraf..."
    
    # 创建主配置文件
    cat > ${INSTALL_DIR}/conf/config.toml << EOF
# Categraf Configuration
# Generated by AI-Infra-Matrix installer

[global]
# 打印运行配置
print_configs = false
# 主机名，默认使用 hostname -s 的结果
hostname = "${HOSTNAME}"
# 是否忽略运行配置中的 hostname
omit_hostname = false
# 采集周期
interval = 15
# 标签配置
[global.labels]
# 添加自定义标签
# region = "cn-beijing"
# env = "production"
ip = "${HOST_IP}"

# 心跳配置
[heartbeat]
enable = true
# 向服务端发送心跳的间隔时间
interval = 10
# 心跳携带的额外标签
# [heartbeat.labels]
# region = "cn-beijing"

# 日志配置
[log]
# 日志文件名
file_name = "${INSTALL_DIR}/logs/categraf.log"
# 日志级别 debug info warn error
level = "info"
# 单个日志文件大小，单位 MB
max_size = 50
# 日志文件保留数量
max_backups = 5
# 日志文件保留天数
max_age = 7
# 是否压缩归档文件
compress = true

# 数据推送配置
[[writers]]
# Nightingale 地址
url = "http://${N9E_HOST}:${N9E_PORT}/prometheus/v1/write"
# 基本认证
basic_auth_user = ""
basic_auth_pass = ""
# 超时设置
timeout = 10
dial_timeout = 3
max_idle_conns_per_host = 100
EOF

    log_success "Configuration created at ${INSTALL_DIR}/conf/config.toml"
    
    # 启用默认采集器
    enable_default_plugins
}

# 启用默认采集插件
enable_default_plugins() {
    log_info "Enabling default collection plugins..."
    
    local input_dir="${INSTALL_DIR}/conf/input"
    mkdir -p ${input_dir}
    
    # CPU 采集配置
    cat > ${input_dir}/cpu/cpu.toml << 'EOF'
# CPU 监控配置
[cpu]
# 采集所有 CPU
collect_per_cpu = true
# 报告活跃 CPU 时间
report_active = true
EOF

    # 内存采集配置
    cat > ${input_dir}/mem/mem.toml << 'EOF'
# 内存监控配置
[mem]
# 采集平台特定指标
collect_platform_fields = true
EOF

    # 磁盘采集配置
    cat > ${input_dir}/disk/disk.toml << 'EOF'
# 磁盘监控配置
[disk]
# 忽略的文件系统类型
ignore_fs = ["tmpfs", "devtmpfs", "devfs", "iso9660", "overlay", "aufs", "squashfs"]
EOF

    # 磁盘 IO 配置
    cat > ${input_dir}/diskio/diskio.toml << 'EOF'
# 磁盘 IO 监控配置
[diskio]
# 采集所有设备
# devices = ["sda", "vda"]
EOF

    # 网络采集配置
    cat > ${input_dir}/net/net.toml << 'EOF'
# 网络监控配置
[net]
# 忽略的网卡
ignore_protocol_stats = false
EOF

    # 系统采集配置
    cat > ${input_dir}/system/system.toml << 'EOF'
# 系统监控配置
[system]
# 采集系统信息
collect_sys_load = true
EOF

    # 进程采集配置
    cat > ${input_dir}/procstat/procstat.toml << 'EOF'
# 进程监控配置
[[instances]]
# 按名称匹配进程
# search_cmdline_string = "salt-minion"
# pattern = "salt.*"
EOF

    log_success "Default plugins enabled"
}

# 创建 systemd 服务
create_systemd_service() {
    log_info "Creating systemd service..."
    
    cat > /etc/systemd/system/categraf.service << EOF
[Unit]
Description=Categraf Monitoring Agent
Documentation=https://github.com/flashcatcloud/categraf
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=${USER}
Group=${GROUP}
ExecStart=${INSTALL_DIR}/bin/categraf --config ${INSTALL_DIR}/conf/config.toml
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5

# 工作目录
WorkingDirectory=${INSTALL_DIR}

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=categraf

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable categraf
    
    log_success "Systemd service created and enabled"
}

# 启动服务
start_service() {
    log_info "Starting Categraf service..."
    systemctl start categraf
    
    # 等待服务启动
    sleep 2
    
    if systemctl is-active categraf > /dev/null 2>&1; then
        log_success "Categraf is running"
    else
        log_error "Categraf failed to start. Check logs: journalctl -u categraf"
        exit 1
    fi
}

# 清理
cleanup() {
    log_info "Cleaning up..."
    cd /tmp
    rm -rf categraf-* *.tar.gz
}

# 打印安装摘要
print_summary() {
    echo ""
    echo "=========================================="
    echo "Categraf Installation Complete"
    echo "=========================================="
    echo ""
    echo "Installation Directory: ${INSTALL_DIR}"
    echo "Configuration: ${INSTALL_DIR}/conf/config.toml"
    echo "Logs: ${INSTALL_DIR}/logs/"
    echo ""
    echo "Nightingale Server: ${N9E_HOST}:${N9E_PORT}"
    echo "Hostname: ${HOSTNAME}"
    echo "Host IP: ${HOST_IP}"
    echo ""
    echo "Commands:"
    echo "  systemctl status categraf   # 查看状态"
    echo "  systemctl restart categraf  # 重启服务"
    echo "  journalctl -u categraf -f   # 查看日志"
    echo ""
    echo "Verify data in Nightingale:"
    echo "  http://${N9E_HOST}:${N9E_PORT}"
    echo ""
}

# 主函数
main() {
    parse_args "$@"
    validate_params
    
    echo "=========================================="
    echo "Installing Categraf ${CATEGRAF_VERSION}"
    echo "=========================================="
    echo ""
    
    download_categraf
    install_categraf
    configure_categraf
    create_systemd_service
    start_service
    cleanup
    print_summary
}

# 执行主函数
main "$@"
