#!/bin/bash
# =============================================================================
# Categraf Installation Script
# 从 AppHub 安装 Categraf 监控代理并配置上报到 Nightingale
# =============================================================================
set -e

# 配置参数 (从模板变量)
APPHUB_HOST="${APPHUB_HOST:-192.168.249.202}"
APPHUB_PORT="${APPHUB_PORT:-28080}"
NIGHTINGALE_HOST="${NIGHTINGALE_HOST:-192.168.249.202}"
NIGHTINGALE_PORT="${NIGHTINGALE_PORT:-17000}"
CATEGRAF_VERSION="${CATEGRAF_VERSION:-v0.4.25}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检测系统架构
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) log_error "Unsupported architecture: $arch"; exit 1 ;;
    esac
}

# 检测系统类型
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    elif [ -f /etc/centos-release ]; then
        echo "centos"
    else
        echo "unknown"
    fi
}

ARCH=$(detect_arch)
OS=$(detect_os)

log_info "===== Categraf Installation Script ====="
log_info "System: ${OS} (${ARCH})"
log_info "AppHub: http://${APPHUB_HOST}:${APPHUB_PORT}"
log_info "Nightingale: http://${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}"
log_info "Version: ${CATEGRAF_VERSION}"

# Step 1: 下载 Categraf
log_info "Step 1: Downloading Categraf..."

CATEGRAF_FILE="categraf-${CATEGRAF_VERSION}-linux-${ARCH}.tar.gz"
CATEGRAF_URL="http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/categraf/${CATEGRAF_FILE}"

# 也支持 latest 版本
if [ "$CATEGRAF_VERSION" = "latest" ]; then
    CATEGRAF_FILE="categraf-latest-linux-${ARCH}.tar.gz"
    CATEGRAF_URL="http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/categraf/${CATEGRAF_FILE}"
fi

cd /tmp
if curl -fsSL -o "${CATEGRAF_FILE}" "${CATEGRAF_URL}"; then
    log_success "Downloaded ${CATEGRAF_FILE}"
else
    log_error "Failed to download from ${CATEGRAF_URL}"
    exit 1
fi

# Step 2: 解压安装
log_info "Step 2: Installing Categraf..."

INSTALL_DIR="/opt/categraf"
mkdir -p "${INSTALL_DIR}"

tar -xzf "${CATEGRAF_FILE}" -C "${INSTALL_DIR}" --strip-components=1

# 确保权限
chmod +x "${INSTALL_DIR}/categraf"

# 创建 categraf 用户
if ! id categraf >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /bin/false categraf
    log_info "Created categraf user"
fi

# Step 3: 配置 Categraf
log_info "Step 3: Configuring Categraf for Nightingale..."

# 备份原配置
if [ -f "${INSTALL_DIR}/conf/config.toml" ]; then
    cp "${INSTALL_DIR}/conf/config.toml" "${INSTALL_DIR}/conf/config.toml.bak"
fi

# 创建配置目录
mkdir -p "${INSTALL_DIR}/conf/input.cpu"
mkdir -p "${INSTALL_DIR}/conf/input.mem"
mkdir -p "${INSTALL_DIR}/conf/input.disk"
mkdir -p "${INSTALL_DIR}/conf/input.diskio"
mkdir -p "${INSTALL_DIR}/conf/input.net"
mkdir -p "${INSTALL_DIR}/conf/input.system"
mkdir -p "${INSTALL_DIR}/conf/input.netstat"
mkdir -p "${INSTALL_DIR}/conf/input.kernel"
mkdir -p "${INSTALL_DIR}/conf/input.processes"
mkdir -p "${INSTALL_DIR}/conf/input.conntrack"

# 主配置文件
cat > "${INSTALL_DIR}/conf/config.toml" << EOF
[global]
# 主机名，默认使用系统主机名
hostname = ""
# 是否忽略主机名标签
omit_hostname = false
# 采集间隔，单位秒
interval = 15
# 标签
[global.labels]
# 自定义标签
# region = "cn-hangzhou"
# environment = "production"

# Writer 配置 - 发送数据到 Nightingale
[[writers]]
url = "http://${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}/prometheus/v1/write"
# 基础认证（如果需要）
# basic_auth_user = ""
# basic_auth_pass = ""
# 超时配置
timeout = "5s"
dial_timeout = "2500ms"
max_idle_conns_per_host = 100

# Heartbeat 配置 - 上报心跳到 Nightingale
[heartbeat]
enable = true
# n9e server 地址
url = "http://${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}/v1/n9e/heartbeat"
# 上报间隔
interval = "10s"
# 基础认证（如果需要）
# basic_auth_user = ""
# basic_auth_pass = ""

# iBex 远程执行配置（可选）
# [ibex]
# enable = false
# servers = ["${NIGHTINGALE_HOST}:20090"]

# Prometheus Exporter 配置（可选）
# [prometheus]
# enable = false
# # listen_addr = ":9101"

# HTTP API 配置
[http]
enable = true
address = ":9100"
print_access = false

# 日志配置
[log]
file_name = "/var/log/categraf/categraf.log"
max_size = 100
max_age = 1
max_backups = 3
local_time = true
compress = false
# debug info warn error
level = "info"
EOF

# CPU 采集配置
cat > "${INSTALL_DIR}/conf/input.cpu/cpu.toml" << EOF
# CPU 采集配置
[cpu]
collect_per_cpu = false
report_active = true
EOF

# Memory 采集配置
cat > "${INSTALL_DIR}/conf/input.mem/mem.toml" << EOF
# 内存采集配置
[mem]
collect_platform_fields = true
EOF

# Disk 采集配置
cat > "${INSTALL_DIR}/conf/input.disk/disk.toml" << EOF
# 磁盘采集配置
[disk]
# 忽略的文件系统类型
ignore_fs = ["tmpfs", "devtmpfs", "devfs", "iso9660", "overlay", "aufs", "squashfs"]
EOF

# DiskIO 采集配置
cat > "${INSTALL_DIR}/conf/input.diskio/diskio.toml" << EOF
# 磁盘 IO 采集配置
[diskio]
# 采集的设备
# devices = ["sda", "sdb"]
EOF

# Network 采集配置
cat > "${INSTALL_DIR}/conf/input.net/net.toml" << EOF
# 网络采集配置
[net]
# 忽略的接口
ignore_protocol_stats = false
# interfaces = ["eth*", "en*"]
EOF

# System 采集配置
cat > "${INSTALL_DIR}/conf/input.system/system.toml" << EOF
# 系统采集配置
[system]
collect_user_number = true
EOF

# Netstat 采集配置
cat > "${INSTALL_DIR}/conf/input.netstat/netstat.toml" << EOF
# 网络状态采集配置
[netstat]
EOF

# Kernel 采集配置
cat > "${INSTALL_DIR}/conf/input.kernel/kernel.toml" << EOF
# 内核采集配置
[kernel]
EOF

# Processes 采集配置
cat > "${INSTALL_DIR}/conf/input.processes/processes.toml" << EOF
# 进程采集配置
[processes]
EOF

# Conntrack 采集配置
cat > "${INSTALL_DIR}/conf/input.conntrack/conntrack.toml" << EOF
# 连接追踪采集配置
[conntrack]
files = [
    "/proc/sys/net/netfilter/nf_conntrack_count",
    "/proc/sys/net/netfilter/nf_conntrack_max"
]
dirs = []
EOF

# 创建日志目录
mkdir -p /var/log/categraf
chown -R categraf:categraf /var/log/categraf

# Step 4: 创建 Systemd 服务
log_info "Step 4: Creating systemd service..."

cat > /etc/systemd/system/categraf.service << EOF
[Unit]
Description=Categraf Monitoring Agent
Documentation=https://flashcat.cloud/product/categraf/
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=categraf
Group=categraf
ExecStart=${INSTALL_DIR}/categraf
WorkingDirectory=${INSTALL_DIR}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=65536

# 安全选项
NoNewPrivileges=false
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
EOF

# 设置目录权限
chown -R categraf:categraf "${INSTALL_DIR}"

# Step 5: 启动服务
log_info "Step 5: Starting Categraf service..."

systemctl daemon-reload
systemctl enable categraf
systemctl start categraf

# Step 6: 验证
log_info "Step 6: Verifying installation..."

sleep 3

if systemctl is-active --quiet categraf; then
    log_success "Categraf is running"
    
    # 检查 HTTP 端口
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:9100/metrics | grep -q "200"; then
        log_success "Categraf metrics endpoint is accessible at :9100/metrics"
    fi
    
    # 显示服务状态
    echo ""
    echo "===== Service Status ====="
    systemctl status categraf --no-pager -l || true
else
    log_error "Categraf failed to start"
    journalctl -u categraf --no-pager -n 20
    exit 1
fi

# 清理临时文件
rm -f "/tmp/${CATEGRAF_FILE}"

echo ""
log_success "===== Categraf Installation Complete ====="
echo ""
echo "Installation Summary:"
echo "  - Install directory: ${INSTALL_DIR}"
echo "  - Config file: ${INSTALL_DIR}/conf/config.toml"
echo "  - Log file: /var/log/categraf/categraf.log"
echo "  - Metrics endpoint: http://$(hostname -I | awk '{print $1}'):9100/metrics"
echo "  - Reporting to: http://${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}"
echo ""
echo "Useful commands:"
echo "  sudo systemctl status categraf   # Check status"
echo "  sudo systemctl restart categraf  # Restart"
echo "  sudo journalctl -u categraf -f   # View logs"
echo "  curl localhost:9100/metrics      # Check metrics"
echo ""
