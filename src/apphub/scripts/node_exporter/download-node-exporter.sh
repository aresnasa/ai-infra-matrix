#!/bin/bash
# =============================================================================
# Node Exporter Download Script for AppHub
# 下载 Node Exporter 预编译二进制到 AppHub
#
# 注意: 此脚本已被整合到统一下载脚本:
#   - 项目根目录: scripts/download_third_party.sh
#   - AppHub 目录: scripts/download-github-release.sh
# 建议使用统一脚本进行下载，此脚本保留用于 AppHub 独立使用场景
# =============================================================================
set -e

# 版本配置 (从环境变量或使用默认值)
NODE_EXPORTER_VERSION="${NODE_EXPORTER_VERSION:-1.8.2}"
# 去掉版本号前的 v 前缀 (如果有)
NODE_EXPORTER_VERSION="${NODE_EXPORTER_VERSION#v}"
OUTPUT_DIR="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/node_exporter}"
GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"
GITHUB_PROXY="${GITHUB_PROXY:-}"

echo "📦 Downloading Node Exporter ${NODE_EXPORTER_VERSION}..."
echo "   GITHUB_MIRROR: ${GITHUB_MIRROR:-<disabled>}"
echo "   GITHUB_PROXY:  ${GITHUB_PROXY:-<disabled>}"

mkdir -p "$OUTPUT_DIR"

download_node_exporter() {
    local arch="$1"
    local filename="node_exporter-${NODE_EXPORTER_VERSION}.linux-${arch}.tar.gz"
    local base_url="https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/${filename}"
    # 移除 https:// 前缀避免重复
    local base_url_without_scheme="${base_url#https://}"
    local mirror_url="${GITHUB_MIRROR}${base_url_without_scheme}"
    
    if [[ -f "${OUTPUT_DIR}/${filename}" ]]; then
        echo "  ✓ ${filename} already exists, skipping"
        return 0
    fi
    
    echo "  📥 Downloading ${arch}..."
    
    # 方式1: 尝试镜像
    if [[ -n "$GITHUB_MIRROR" ]]; then
        echo "     [方式1] GITHUB_MIRROR: ${mirror_url}"
        if curl -fsSL --connect-timeout 30 --max-time 300 --retry 3 -o "${OUTPUT_DIR}/${filename}" "$mirror_url" 2>/dev/null; then
            echo "  ✓ Downloaded ${filename} via GITHUB_MIRROR"
            return 0
        fi
        echo "  ⚠️  GITHUB_MIRROR failed, trying next method..."
    fi
    
    # 方式2: 尝试代理
    if [[ -n "$GITHUB_PROXY" ]]; then
        echo "     [方式2] GITHUB_PROXY: ${base_url}"
        echo "     Using proxy: ${GITHUB_PROXY}"
        if curl --proxy "$GITHUB_PROXY" -fsSL --connect-timeout 30 --max-time 300 --retry 3 -o "${OUTPUT_DIR}/${filename}" "$base_url" 2>/dev/null; then
            echo "  ✓ Downloaded ${filename} via GITHUB_PROXY"
            return 0
        fi
        echo "  ⚠️  GITHUB_PROXY failed, trying direct download..."
    fi
    
    # 方式3: 直接下载
    echo "     [方式3] Direct: $base_url"
    if curl -fsSL --connect-timeout 30 --max-time 300 --retry 3 -o "${OUTPUT_DIR}/${filename}" "$base_url"; then
        echo "  ✓ Downloaded ${filename} directly"
        return 0
    else
        echo "  ❌ All download methods failed: ${filename}"
        rm -f "${OUTPUT_DIR}/${filename}"
        return 1
    fi
}

# 下载 amd64 和 arm64 版本
download_node_exporter "amd64"
download_node_exporter "arm64"

# 创建版本文件
cat > "${OUTPUT_DIR}/version.json" << EOF
{
    "name": "node_exporter",
    "version": "${NODE_EXPORTER_VERSION}",
    "files": [
        {
            "filename": "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz",
            "arch": "amd64",
            "os": "linux"
        },
        {
            "filename": "node_exporter-${NODE_EXPORTER_VERSION}.linux-arm64.tar.gz",
            "arch": "arm64",
            "os": "linux"
        }
    ],
    "updated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

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
