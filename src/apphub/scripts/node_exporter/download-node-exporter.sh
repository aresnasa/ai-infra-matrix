#!/bin/bash
# =============================================================================
# Node Exporter Download Script for AppHub
# 下载 Node Exporter 预编译二进制到 AppHub
# =============================================================================
set -e

# 版本配置 (从环境变量或使用默认值)
NODE_EXPORTER_VERSION="${NODE_EXPORTER_VERSION:-1.8.2}"
# 去掉版本号前的 v 前缀 (如果有)
NODE_EXPORTER_VERSION="${NODE_EXPORTER_VERSION#v}"
OUTPUT_DIR="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/node_exporter}"
GITHUB_MIRROR="${GITHUB_MIRROR:-}"

echo "📦 Downloading Node Exporter ${NODE_EXPORTER_VERSION}..."

mkdir -p "$OUTPUT_DIR"

download_node_exporter() {
    local arch="$1"
    local filename="node_exporter-${NODE_EXPORTER_VERSION}.linux-${arch}.tar.gz"
    local url="https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/${filename}"
    
    # 使用 GitHub 镜像加速 (如果配置了)
    if [[ -n "$GITHUB_MIRROR" ]]; then
        url="${GITHUB_MIRROR}${url}"
    fi
    
    echo "  📥 Downloading ${arch}..."
    echo "     URL: $url"
    
    if command -v wget &> /dev/null; then
        wget -q --show-progress -O "${OUTPUT_DIR}/${filename}" "$url" || {
            echo "  ⚠️  wget failed, trying curl..."
            curl -fsSL -o "${OUTPUT_DIR}/${filename}" "$url"
        }
    elif command -v curl &> /dev/null; then
        curl -fsSL -o "${OUTPUT_DIR}/${filename}" "$url"
    else
        echo "  ❌ Neither wget nor curl available"
        return 1
    fi
    
    # 验证下载
    if [[ -f "${OUTPUT_DIR}/${filename}" ]]; then
        local size=$(stat -f%z "${OUTPUT_DIR}/${filename}" 2>/dev/null || stat -c%s "${OUTPUT_DIR}/${filename}" 2>/dev/null)
        echo "  ✓ Downloaded: ${filename} (${size} bytes)"
    else
        echo "  ❌ Download failed: ${filename}"
        return 1
    fi
}

# 下载 amd64 和 arm64 版本
download_node_exporter "amd64"
download_node_exporter "arm64"

# 创建版本文件
echo "${NODE_EXPORTER_VERSION}" > "${OUTPUT_DIR}/VERSION"

# 创建安装脚本
cat > "${OUTPUT_DIR}/install.sh" << 'INSTALL_SCRIPT'
#!/bin/bash
# Node Exporter 快速安装脚本
set -e

VERSION="${NODE_EXPORTER_VERSION:-1.8.2}"
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

APPHUB_URL="${APPHUB_URL:-http://apphub:8081}"
FILENAME="node_exporter-${VERSION}.linux-${ARCH}.tar.gz"

echo "Installing Node Exporter ${VERSION} (${ARCH})..."

# 下载并解压
curl -fsSL "${APPHUB_URL}/pkgs/node_exporter/${FILENAME}" | tar xzf - -C /tmp

# 安装二进制
mv "/tmp/node_exporter-${VERSION}.linux-${ARCH}/node_exporter" /usr/local/bin/
chmod +x /usr/local/bin/node_exporter

# 创建用户
useradd --no-create-home --shell /bin/false node_exporter 2>/dev/null || true

# 创建 systemd 服务
cat > /etc/systemd/system/node_exporter.service << EOF
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter \\
    --web.listen-address=:9100 \\
    --collector.textfile.directory=/var/lib/node_exporter/textfile_collector

[Install]
WantedBy=multi-user.target
EOF

# 创建 textfile collector 目录
mkdir -p /var/lib/node_exporter/textfile_collector
chown node_exporter:node_exporter /var/lib/node_exporter/textfile_collector

# 启动服务
systemctl daemon-reload
systemctl enable node_exporter
systemctl start node_exporter

echo "✓ Node Exporter installed and started on port 9100"
INSTALL_SCRIPT

chmod +x "${OUTPUT_DIR}/install.sh"

echo ""
echo "✅ Node Exporter ${NODE_EXPORTER_VERSION} download complete!"
echo "   Output: ${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}/"
