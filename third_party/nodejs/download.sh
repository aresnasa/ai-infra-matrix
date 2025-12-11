#!/bin/bash
# 下载 Node.js 22.x LTS 预编译二进制包
# 支持 x86_64 和 arm64 双架构

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Node.js 版本
NODE_VERSION="22.11.0"

# 下载 URL 基础路径
# 官方源: https://nodejs.org/dist
# 国内镜像: https://npmmirror.com/mirrors/node 或 https://mirrors.aliyun.com/nodejs-release
NODE_MIRROR="${NODE_MIRROR:-https://npmmirror.com/mirrors/node}"

echo "=== Downloading Node.js v${NODE_VERSION} ==="
echo "Mirror: ${NODE_MIRROR}"

# 下载函数
download_node() {
    local arch=$1
    local filename="node-v${NODE_VERSION}-linux-${arch}.tar.xz"
    local url="${NODE_MIRROR}/v${NODE_VERSION}/${filename}"
    
    if [ -f "$filename" ]; then
        echo "✓ ${filename} already exists, skipping..."
        return 0
    fi
    
    echo "Downloading ${filename}..."
    if curl -fSL -o "${filename}.tmp" "$url"; then
        mv "${filename}.tmp" "$filename"
        echo "✓ Downloaded ${filename}"
    else
        echo "❌ Failed to download from ${NODE_MIRROR}, trying official source..."
        url="https://nodejs.org/dist/v${NODE_VERSION}/${filename}"
        if curl -fSL -o "${filename}.tmp" "$url"; then
            mv "${filename}.tmp" "$filename"
            echo "✓ Downloaded ${filename} from official source"
        else
            echo "❌ Failed to download ${filename}"
            rm -f "${filename}.tmp"
            return 1
        fi
    fi
}

# 下载 x86_64 版本
download_node "x64"

# 下载 arm64 版本
download_node "arm64"

echo ""
echo "=== Download Complete ==="
ls -lh *.tar.xz 2>/dev/null || echo "No files downloaded"

# 创建 README
cat > README.md << 'EOF'
# Node.js 预编译二进制包

用于 JupyterHub 容器构建，避免在 Docker 构建时从网络下载。

## 文件说明

- `node-v22.11.0-linux-x64.tar.xz` - x86_64 架构
- `node-v22.11.0-linux-arm64.tar.xz` - ARM64 架构

## 更新方法

```bash
# 使用国内镜像下载
./download.sh

# 使用官方源下载
NODE_MIRROR=https://nodejs.org/dist ./download.sh
```

## 版本信息

- Node.js: 22.11.0 LTS
- npm: 包含在 Node.js 中
EOF

echo "✓ Created README.md"
