#!/bin/bash
# Categraf 监控代理安装脚本
# 由 Nightingale 提供，支持 x86_64 和 ARM64 双架构
# 
# 环境变量说明：
#   CATEGRAF_VERSION  - Categraf 版本 (默认: v0.3.80)
#   N9E_HOST         - Nightingale 服务地址 (默认: nightingale)
#   N9E_PORT         - Nightingale HTTP 端口 (默认: 17000)
#   HOSTNAME         - 主机名 (默认: $(hostname))
#   HOST_IP          - 主机 IP (默认: 自动检测)
#   GITHUB_MIRROR    - GitHub 镜像地址 (可选, 如: https://ghfast.top)
#   APPHUB_URL       - AppHub 服务地址 (可选, 如: http://apphub:80)
#
# 使用示例:
#   curl -fsSL http://nightingale:17000/api/v1/scripts/install-categraf.sh | bash
#   N9E_HOST=192.168.1.100 N9E_PORT=17000 bash install-categraf.sh

set -e

echo "=============================================="
echo "  Nightingale Monitoring Agent Installer"
echo "  (Categraf)"
echo "=============================================="

# ===================== 配置参数 =====================

# Categraf 版本
CATEGRAF_VERSION="${CATEGRAF_VERSION:-v0.3.80}"

# Nightingale 服务配置
N9E_HOST="${N9E_HOST:-nightingale}"
N9E_PORT="${N9E_PORT:-17000}"

# 主机标识
HOSTNAME="${HOSTNAME:-$(hostname)}"
HOST_IP="${HOST_IP:-$(hostname -I 2>/dev/null | awk '{print $1}' || echo "unknown")}"

# 下载源配置
GITHUB_MIRROR="${GITHUB_MIRROR:-}"
APPHUB_URL="${APPHUB_URL:-}"

# 安装目录
INSTALL_DIR="/opt/categraf"

# ===================== 架构检测 =====================

detect_arch() {
    local arch=$(uname -m)
    case "${arch}" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            echo "Unsupported architecture: ${arch}" >&2
            exit 1
            ;;
    esac
}

ARCH_SUFFIX=$(detect_arch)
echo "Detected architecture: ${ARCH_SUFFIX}"

# ===================== 下载 URL 构建 =====================

build_download_url() {
    local version=$1
    local arch=$2
    local filename="categraf-${version}-linux-${arch}.tar.gz"
    
    # 优先使用 AppHub
    if [ -n "${APPHUB_URL}" ]; then
        echo "${APPHUB_URL}/packages/categraf/${filename}"
        return
    fi
    
    # 使用 GitHub (可选镜像)
    local github_base="https://github.com"
    if [ -n "${GITHUB_MIRROR}" ]; then
        # 处理镜像格式 (支持 ghfast.top 和完整 URL)
        if [[ "${GITHUB_MIRROR}" == http* ]]; then
            github_base="${GITHUB_MIRROR}"
        else
            github_base="https://${GITHUB_MIRROR}"
        fi
    fi
    
    echo "${github_base}/flashcatcloud/categraf/releases/download/${version}/${filename}"
}

DOWNLOAD_URL=$(build_download_url "${CATEGRAF_VERSION}" "${ARCH_SUFFIX}")
echo "Download URL: ${DOWNLOAD_URL}"

# ===================== 安装过程 =====================

echo ""
echo "[1/6] Creating installation directory..."
sudo mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

echo "[2/6] Downloading Categraf ${CATEGRAF_VERSION}..."
if command -v wget &> /dev/null; then
    sudo wget -q --show-progress "${DOWNLOAD_URL}" -O categraf.tar.gz
elif command -v curl &> /dev/null; then
    sudo curl -fSL "${DOWNLOAD_URL}" -o categraf.tar.gz
else
    echo "Error: wget or curl is required" >&2
    exit 1
fi

echo "[3/6] Extracting..."
sudo tar -xzf categraf.tar.gz --strip-components=1
sudo rm -f categraf.tar.gz

echo "[4/6] Configuring Categraf..."
sudo mkdir -p "${INSTALL_DIR}/conf"
sudo tee "${INSTALL_DIR}/conf/config.toml" > /dev/null <<EOF
[global]
hostname = "${HOSTNAME}"
labels = { ip="${HOST_IP}" }

[heartbeat]
enable = true
url = "http://${N9E_HOST}:${N9E_PORT}/v1/n9e/heartbeat"
interval = 10

[writer_opt]
batch = 2000
chan_size = 10000

[[writers]]
url = "http://${N9E_HOST}:${N9E_PORT}/prometheus/v1/write"
EOF

# 配置系统监控插件
echo "[5/6] Enabling system collectors..."

# CPU 监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.cpu"
sudo tee "${INSTALL_DIR}/conf/input.cpu/cpu.toml" > /dev/null <<EOF
[[instances]]
collect_per_cpu = true
report_active = true
EOF

# 内存监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.mem"
sudo tee "${INSTALL_DIR}/conf/input.mem/mem.toml" > /dev/null <<EOF
[[instances]]
# collect memory stats
EOF

# 磁盘监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.disk"
sudo tee "${INSTALL_DIR}/conf/input.disk/disk.toml" > /dev/null <<EOF
[[instances]]
mount_points = ["/"]
ignore_fs = ["tmpfs", "devtmpfs", "devfs", "iso9660", "overlay", "aufs", "squashfs"]
EOF

# 网络监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.net"
sudo tee "${INSTALL_DIR}/conf/input.net/net.toml" > /dev/null <<EOF
[[instances]]
interfaces = ["eth*", "en*", "ens*", "eno*"]
EOF

# 网络状态监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.netstat"
sudo tee "${INSTALL_DIR}/conf/input.netstat/netstat.toml" > /dev/null <<EOF
[[instances]]
# collect netstat
EOF

# 系统负载监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.system"
sudo tee "${INSTALL_DIR}/conf/input.system/system.toml" > /dev/null <<EOF
[[instances]]
# collect system load
EOF

# 进程监控
sudo mkdir -p "${INSTALL_DIR}/conf/input.procstat"
sudo tee "${INSTALL_DIR}/conf/input.procstat/procstat.toml" > /dev/null <<EOF
[[instances]]
# Monitor specific processes if needed
# exe = "nginx"
EOF

echo "[6/6] Creating systemd service..."
sudo tee /etc/systemd/system/categraf.service > /dev/null <<'EOF'
[Unit]
Description=Categraf Monitoring Agent for Nightingale
Documentation=https://github.com/flashcatcloud/categraf
After=network.target

[Service]
Type=simple
User=root
ExecStart=/opt/categraf/categraf --configs /opt/categraf/conf
Restart=always
RestartSec=10
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
echo ""
echo "Starting Categraf service..."
sudo systemctl daemon-reload
sudo systemctl enable categraf
sudo systemctl start categraf

# 等待启动
sleep 2

# 检查状态
echo ""
echo "Checking service status..."
if sudo systemctl is-active --quiet categraf; then
    echo ""
    echo "=============================================="
    echo "✓ Categraf installed successfully!"
    echo "=============================================="
    echo ""
    echo "  Hostname:     ${HOSTNAME}"
    echo "  IP:           ${HOST_IP}"
    echo "  Architecture: ${ARCH_SUFFIX}"
    echo "  Version:      ${CATEGRAF_VERSION}"
    echo "  Reporting to: http://${N9E_HOST}:${N9E_PORT}"
    echo ""
    echo "Useful commands:"
    echo "  sudo systemctl status categraf  - Check status"
    echo "  sudo systemctl restart categraf - Restart agent"
    echo "  sudo journalctl -u categraf -f  - View logs"
    echo ""
else
    echo "✗ Service failed to start. Check logs:"
    echo "  sudo journalctl -u categraf -n 50"
    exit 1
fi
