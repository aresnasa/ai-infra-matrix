#!/bin/bash
# =============================================================================
# Categraf Multi-Architecture Build Script (For Docker Build)
# 为 x86_64 和 aarch64 构建 Categraf 监控客户端
# 在 Dockerfile 中调用，不使用 heredoc
# =============================================================================

set -e

# 环境变量配置
CATEGRAF_VERSION=${CATEGRAF_VERSION:-"v0.3.90"}
CATEGRAF_REPO=${CATEGRAF_REPO:-"https://github.com/flashcatcloud/categraf.git"}
BUILD_DIR=${BUILD_DIR:-"/build"}
OUTPUT_DIR=${OUTPUT_DIR:-"/out"}
SCRIPT_DIR=${SCRIPT_DIR:-"/scripts/categraf"}

echo "📥 Cloning Categraf ${CATEGRAF_VERSION}..."
git clone --depth 1 --branch "${CATEGRAF_VERSION}" "${CATEGRAF_REPO}" ${BUILD_DIR}/categraf
cd ${BUILD_DIR}/categraf

# 获取版本信息
VERSION=$(git describe --tags --always 2>/dev/null || echo "${CATEGRAF_VERSION}")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo "  Version: ${VERSION}"
echo "  Commit: ${COMMIT}"
echo "  Build time: ${BUILD_TIME}"

# 构建 ldflags
LDFLAGS="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# 构建 AMD64
echo "🔨 Building Categraf for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/categraf-linux-amd64 ./cmd/categraf
echo "✓ Built categraf-linux-amd64"

# 构建 ARM64
echo "🔨 Building Categraf for linux/arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/categraf-linux-arm64 ./cmd/categraf
echo "✓ Built categraf-linux-arm64"

# 打包 AMD64 版本
echo "📦 Packaging Categraf for amd64..."
PKG_AMD64="${BUILD_DIR}/categraf-${VERSION}-linux-amd64"
mkdir -p "${PKG_AMD64}"/{bin,conf,logs}
cp ${BUILD_DIR}/categraf-linux-amd64 "${PKG_AMD64}/bin/categraf"
chmod +x "${PKG_AMD64}/bin/categraf"
if [ -d "conf" ]; then
    cp -r conf/* "${PKG_AMD64}/conf/"
fi

# 创建 AMD64 服务文件和脚本
cp ${SCRIPT_DIR}/systemd.service "${PKG_AMD64}/categraf.service"
cp ${SCRIPT_DIR}/install.sh "${PKG_AMD64}/install.sh"
cp ${SCRIPT_DIR}/uninstall.sh "${PKG_AMD64}/uninstall.sh"
chmod +x "${PKG_AMD64}/install.sh" "${PKG_AMD64}/uninstall.sh"

# 创建 AMD64 README
sed "s/VERSION_PLACEHOLDER/${VERSION}/g; s/ARCH_PLACEHOLDER/amd64/g" ${SCRIPT_DIR}/readme.md > "${PKG_AMD64}/README.md"

# 打包 AMD64
cd ${BUILD_DIR}
tar czf "${OUTPUT_DIR}/categraf-${VERSION}-linux-amd64.tar.gz" "categraf-${VERSION}-linux-amd64"
echo "✓ Created categraf-${VERSION}-linux-amd64.tar.gz"

# 打包 ARM64 版本
echo "📦 Packaging Categraf for arm64..."
PKG_ARM64="${BUILD_DIR}/categraf-${VERSION}-linux-arm64"
mkdir -p "${PKG_ARM64}"/{bin,conf,logs}
cp ${BUILD_DIR}/categraf-linux-arm64 "${PKG_ARM64}/bin/categraf"
chmod +x "${PKG_ARM64}/bin/categraf"
if [ -d "categraf/conf" ]; then
    cp -r categraf/conf/* "${PKG_ARM64}/conf/"
fi

# 创建 ARM64 服务文件和脚本
cp ${SCRIPT_DIR}/systemd.service "${PKG_ARM64}/categraf.service"
cp ${SCRIPT_DIR}/install.sh "${PKG_ARM64}/install.sh"
cp ${SCRIPT_DIR}/uninstall.sh "${PKG_ARM64}/uninstall.sh"
chmod +x "${PKG_ARM64}/install.sh" "${PKG_ARM64}/uninstall.sh"

# 创建 ARM64 README
sed "s/VERSION_PLACEHOLDER/${VERSION}/g; s/ARCH_PLACEHOLDER/arm64/g" ${SCRIPT_DIR}/readme.md > "${PKG_ARM64}/README.md"

# 打包 ARM64
tar czf "${OUTPUT_DIR}/categraf-${VERSION}-linux-arm64.tar.gz" "categraf-${VERSION}-linux-arm64"
echo "✓ Created categraf-${VERSION}-linux-arm64.tar.gz"

# 输出摘要
echo ""
echo "📦 Package summary:"
ls -lh ${OUTPUT_DIR}/*.tar.gz
