#!/bin/bash
# =============================================================================
# Prometheus Installation Script
# 安装 Prometheus 到目标主机
# =============================================================================

set -e

# 默认配置
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-3.7.3}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/prometheus}"
DATA_DIR="${DATA_DIR:-/var/lib/prometheus}"
CONFIG_DIR="${CONFIG_DIR:-/etc/prometheus}"
USER="${PROMETHEUS_USER:-prometheus}"
GROUP="${PROMETHEUS_GROUP:-prometheus}"
APPHUB_URL="${APPHUB_URL:-}"

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
            echo "Unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

ARCH=$(detect_arch)
PACKAGE_NAME="prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}.tar.gz"

echo "==================================="
echo "Installing Prometheus ${PROMETHEUS_VERSION}"
echo "Architecture: ${ARCH}"
echo "==================================="

# 创建用户和组
if ! getent group ${GROUP} > /dev/null 2>&1; then
    groupadd --system ${GROUP}
    echo "✓ Created group: ${GROUP}"
fi

if ! getent passwd ${USER} > /dev/null 2>&1; then
    useradd --system --no-create-home --gid ${GROUP} ${USER}
    echo "✓ Created user: ${USER}"
fi

# 创建目录
mkdir -p ${INSTALL_DIR}
mkdir -p ${DATA_DIR}
mkdir -p ${CONFIG_DIR}
mkdir -p ${CONFIG_DIR}/rules

# 下载 Prometheus
DOWNLOAD_URL=""
if [ -n "${APPHUB_URL}" ]; then
    DOWNLOAD_URL="${APPHUB_URL}/pkgs/prometheus/${PACKAGE_NAME}"
else
    DOWNLOAD_URL="https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${PACKAGE_NAME}"
fi

echo "Downloading from: ${DOWNLOAD_URL}"
cd /tmp
curl -fsSL -o "${PACKAGE_NAME}" "${DOWNLOAD_URL}"

# 解压
tar xzf "${PACKAGE_NAME}"
cd "prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}"

# 安装二进制文件
cp prometheus promtool ${INSTALL_DIR}/
chmod +x ${INSTALL_DIR}/prometheus ${INSTALL_DIR}/promtool

# 安装控制台文件
cp -r consoles console_libraries ${CONFIG_DIR}/

# 创建默认配置（如果不存在）
if [ ! -f ${CONFIG_DIR}/prometheus.yml ]; then
    cat > ${CONFIG_DIR}/prometheus.yml << 'EOF'
# Prometheus 配置文件
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
EOF
fi

# 设置权限
chown -R ${USER}:${GROUP} ${INSTALL_DIR}
chown -R ${USER}:${GROUP} ${DATA_DIR}
chown -R ${USER}:${GROUP} ${CONFIG_DIR}

# 创建 systemd 服务
cat > /etc/systemd/system/prometheus.service << EOF
[Unit]
Description=Prometheus Monitoring System
Documentation=https://prometheus.io/docs/introduction/overview/
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=${USER}
Group=${GROUP}
ExecReload=/bin/kill -HUP \$MAINPID
ExecStart=${INSTALL_DIR}/prometheus \\
    --config.file=${CONFIG_DIR}/prometheus.yml \\
    --storage.tsdb.path=${DATA_DIR} \\
    --storage.tsdb.retention.time=30d \\
    --web.console.templates=${CONFIG_DIR}/consoles \\
    --web.console.libraries=${CONFIG_DIR}/console_libraries \\
    --web.enable-lifecycle \\
    --web.listen-address=0.0.0.0:9090

SyslogIdentifier=prometheus
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 重载 systemd
systemctl daemon-reload
systemctl enable prometheus

# 清理
cd /tmp
rm -rf "prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}" "${PACKAGE_NAME}"

echo ""
echo "✓ Prometheus ${PROMETHEUS_VERSION} installed successfully"
echo ""
echo "Commands:"
echo "  systemctl start prometheus    # 启动服务"
echo "  systemctl status prometheus   # 查看状态"
echo "  systemctl restart prometheus  # 重启服务"
echo ""
echo "Configuration: ${CONFIG_DIR}/prometheus.yml"
echo "Data: ${DATA_DIR}"
echo "Web UI: http://localhost:9090"
