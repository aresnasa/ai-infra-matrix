# Nightingale Dockerfile for AI Infra Matrix
# Build from source code in third_party/nightingale
# Supports custom VITE_PREFIX for sub-path deployment (e.g., /nightingale)

# 全局 ARG 声明 - 必须在所有 FROM 之前声明才能在 FROM 中使用
ARG GOLANG_IMAGE=golang:1.25-bookworm
ARG UBUNTU_IMAGE=ubuntu:22.04
ARG NODE_IMAGE=node:22-bookworm

# ============================================================
# Stage 0: Build Frontend with custom VITE_PREFIX
# ============================================================
FROM ${NODE_IMAGE} AS fe-builder

ARG APT_MIRROR=mirrors.aliyun.com
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG N9E_FE_VERSION=v8.4.1
ARG GITHUB_MIRROR=https://ghfast.top/
ARG GITHUB_PROXY=
# Set VITE_PREFIX to match Nightingale's BasePath configuration
# This ensures React Router and all asset paths use the correct prefix
# Backend config.toml must have: BasePath = "/nightingale"
# Nginx must NOT strip the /nightingale prefix (no trailing slash in proxy_pass)
ARG VITE_PREFIX=/nightingale

ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /fe

# Configure APT mirror for Debian-based Node image (加速 apt-get)
# Download frontend source code from GitHub
# The n9e-fe-src directory may not be included in Docker build context
# because it's a nested git repository (not a submodule)
# Uses 3-way fallback: GITHUB_MIRROR → GITHUB_PROXY → Direct
RUN set -eux; \
    if [ -n "${APT_MIRROR}" ]; then \
        echo "Configuring APT mirror: ${APT_MIRROR}"; \
        if [ -f /etc/apt/sources.list ]; then \
            sed -i "s|deb.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list; \
            sed -i "s|security.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list; \
        elif [ -f /etc/apt/sources.list.d/debian.sources ]; then \
            sed -i "s|deb.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list.d/debian.sources; \
            sed -i "s|security.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list.d/debian.sources; \
        fi; \
    fi; \
    apt-get update && apt-get install -y --no-install-recommends git ca-certificates curl && rm -rf /var/lib/apt/lists/*; \
    echo "Downloading n9e-fe ${N9E_FE_VERSION}..."; \
    GITHUB_URL="https://github.com/n9e/fe.git"; \
    DOWNLOADED=false; \
    # Method 1: Try GITHUB_MIRROR (URL prefix acceleration)
    if [ "$DOWNLOADED" = "false" ] && [ -n "${GITHUB_MIRROR}" ]; then \
        echo "  → Trying GITHUB_MIRROR: ${GITHUB_MIRROR}${GITHUB_URL}"; \
        if git clone --depth 1 --branch "${N9E_FE_VERSION}" "${GITHUB_MIRROR}${GITHUB_URL}" . 2>/dev/null; then \
            echo "  ✓ Downloaded via GITHUB_MIRROR"; \
            DOWNLOADED=true; \
        else \
            echo "  ✗ GITHUB_MIRROR failed"; \
        fi; \
    fi; \
    # Method 2: Try GITHUB_PROXY (HTTP proxy)
    if [ "$DOWNLOADED" = "false" ] && [ -n "${GITHUB_PROXY}" ]; then \
        echo "  → Trying GITHUB_PROXY: ${GITHUB_PROXY}"; \
        export http_proxy="${GITHUB_PROXY}"; \
        export https_proxy="${GITHUB_PROXY}"; \
        if git clone --depth 1 --branch "${N9E_FE_VERSION}" "${GITHUB_URL}" . 2>/dev/null; then \
            echo "  ✓ Downloaded via GITHUB_PROXY"; \
            DOWNLOADED=true; \
        else \
            echo "  ✗ GITHUB_PROXY failed"; \
        fi; \
        unset http_proxy https_proxy; \
    fi; \
    # Method 3: Direct download (fallback)
    if [ "$DOWNLOADED" = "false" ]; then \
        echo "  → Trying direct download: ${GITHUB_URL}"; \
        if git clone --depth 1 --branch "${N9E_FE_VERSION}" "${GITHUB_URL}" .; then \
            echo "  ✓ Downloaded directly"; \
            DOWNLOADED=true; \
        else \
            echo "  ✗ Direct download failed"; \
        fi; \
    fi; \
    if [ "$DOWNLOADED" = "false" ]; then \
        echo "ERROR: Failed to download n9e-fe ${N9E_FE_VERSION}"; \
        exit 1; \
    fi; \
    ls -la

# Install dependencies and build with custom prefix
# Note: esbuild crashes under QEMU emulation (ARM64 host building AMD64)
# due to Go runtime "lfstack.push" error in emulated environment.
# Strategy:
#   1. Try normal npm build first
#   2. If build fails (QEMU), download pre-built frontend from n9e/fe releases
#   3. Pre-built frontend works but lacks custom VITE_PREFIX (default /)
# Memory optimization: package.json sets NODE_OPTIONS=--max-old-space-size=6096
# which requires 6GB+ RAM. We modify package.json for low-memory environments.
RUN set -eux; \
    if [ -n "${NPM_REGISTRY}" ]; then \
        npm config set registry ${NPM_REGISTRY}; \
    fi; \
    BUILD_ARCH=$(uname -m); \
    echo "Build architecture: ${BUILD_ARCH}"; \
    # Detect available memory (in MB) and adjust NODE_OPTIONS accordingly
    TOTAL_MEM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}'); \
    TOTAL_MEM_MB=$((TOTAL_MEM_KB / 1024)); \
    if [ "$TOTAL_MEM_MB" -lt 6000 ]; then \
        NODE_MEM=3072; \
        echo "⚠️  Low memory detected (${TOTAL_MEM_MB}MB total)"; \
        echo "   Reducing Node.js heap size to ${NODE_MEM}MB"; \
        # Modify package.json to use lower memory setting
        sed -i "s/max-old-space-size=6096/max-old-space-size=${NODE_MEM}/g" package.json; \
    else \
        NODE_MEM=4096; \
        echo "✓ Sufficient memory (${TOTAL_MEM_MB}MB), using default settings"; \
    fi; \
    # Try to build frontend with custom VITE_PREFIX
    BUILD_SUCCESS=false; \
    npm install --legacy-peer-deps && \
    echo "Building frontend with VITE_PREFIX=${VITE_PREFIX}" && \
    NODE_OPTIONS="--max-old-space-size=${NODE_MEM}" VITE_PREFIX=${VITE_PREFIX} npm run build && \
    BUILD_SUCCESS=true || true; \
    # If build failed (likely QEMU), download pre-built frontend from n9e/fe releases
    if [ "$BUILD_SUCCESS" = "false" ]; then \
        echo "⚠️  Frontend build failed (likely QEMU emulation)"; \
        echo "📦 Downloading pre-built frontend from n9e/fe releases..."; \
        # n9e/fe releases contain pre-built pub/ directory
        FE_RELEASE_URL="https://github.com/n9e/fe/releases/download/${N9E_FE_VERSION}/n9e-fe-${N9E_FE_VERSION}.tar.gz"; \
        # Try with GITHUB_MIRROR first
        if [ -n "${GITHUB_MIRROR}" ]; then \
            FE_RELEASE_URL="${GITHUB_MIRROR}${FE_RELEASE_URL}"; \
        fi; \
        echo "Downloading from: ${FE_RELEASE_URL}"; \
        curl -fsSL "${FE_RELEASE_URL}" | tar -xz || \
        curl -fsSL "https://github.com/n9e/fe/releases/download/${N9E_FE_VERSION}/n9e-fe-${N9E_FE_VERSION}.tar.gz" | tar -xz; \
        # Verify pub directory exists
        if [ -d pub ]; then \
            echo "✓ Using pre-built frontend from n9e/fe releases"; \
            echo "⚠️  Note: Pre-built frontend uses default BasePath (/)"; \
            echo "    Custom VITE_PREFIX=${VITE_PREFIX} was not applied"; \
            echo "    For custom path, build on native platform or use nginx rewrite"; \
        else \
            echo "ERROR: pub directory not found in release"; \
            exit 1; \
        fi; \
    else \
        echo "✓ Frontend built successfully with VITE_PREFIX=${VITE_PREFIX}"; \
    fi; \
    # Verify pub directory exists
    ls -la pub/

# ============================================================
# Stage 1: Build Go Backend
# ============================================================
FROM ${GOLANG_IMAGE} AS builder

ARG APT_MIRROR=mirrors.aliyun.com
ARG GO_PROXY=https://goproxy.cn,direct
ARG GITHUB_MIRROR=https://ghfast.top/
ARG INTERNAL_FILE_SERVER=http://files.example.com
ARG N9E_FE_VERSION=v8.4.1

ENV DEBIAN_FRONTEND=noninteractive

# Configure APT mirror (支持 x86 和 ARM64 双架构，Debian bookworm)
RUN set -eux; \
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ -n "${APT_MIRROR}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        if [ -f /etc/apt/sources.list ]; then \
            sed -i "s|deb.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list; \
            sed -i "s|security.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list; \
        elif [ -f /etc/apt/sources.list.d/debian.sources ]; then \
            sed -i "s|deb.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list.d/debian.sources; \
            sed -i "s|security.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list.d/debian.sources; \
        fi; \
    fi

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends git make bash curl && rm -rf /var/lib/apt/lists/*

# Copy source code
COPY third_party/nightingale/ .

# Copy frontend build output from fe-builder stage
COPY --from=fe-builder /fe/pub/ ./pub/

# Build - 使用多个 Go 代理加速下载
# goproxy.cn 是主要代理，goproxy.io 作为备用
ENV GOPROXY=${GO_PROXY}
ENV GOSUMDB=sum.golang.google.cn

# Step 1: Embed front-end files into Go binary using statik
# 使用多重代理策略，确保 Go 模块可以快速下载
RUN set -eux; \
    echo "Installing statik with GOPROXY=${GOPROXY}..."; \
    # 尝试使用配置的代理安装 statik
    if ! go install github.com/rakyll/statik@latest; then \
        echo "Primary GOPROXY failed, trying goproxy.io..."; \
        GOPROXY=https://goproxy.io,direct go install github.com/rakyll/statik@latest; \
    fi; \
    # Embed front-end files
    $(go env GOPATH)/bin/statik -src=./pub -dest=./front

# Step 2: Tidy and download Go dependencies with fallback
RUN set -eux; \
    go mod tidy && \
    echo "Downloading Go dependencies with GOPROXY=${GOPROXY}..."; \
    if ! go mod download; then \
        echo "Primary GOPROXY failed, trying goproxy.io..."; \
        GOPROXY=https://goproxy.io,direct go mod download; \
    fi

# Step 3: Build the binary
RUN go build -ldflags "-w -s" -o n9e ./cmd/center/main.go

# Stage 2: Runtime
# UBUNTU_IMAGE 已在文件顶部全局声明
FROM ${UBUNTU_IMAGE}

ARG APT_MIRROR=mirrors.aliyun.com
ENV DEBIAN_FRONTEND=noninteractive

# Configure APT mirror (支持 x86 和 ARM64 双架构)
RUN set -eux; \
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ "${ARCH}" = "amd64" ]; then \
        UBUNTU_PATH="ubuntu"; \
    else \
        UBUNTU_PATH="ubuntu-ports"; \
    fi; \
    if [ -n "${APT_MIRROR}" ]; then \
        sed -i "s|archive.ubuntu.com/ubuntu/|${APT_MIRROR}/${UBUNTU_PATH}/|g" /etc/apt/sources.list; \
        sed -i "s|security.ubuntu.com/ubuntu/|${APT_MIRROR}/${UBUNTU_PATH}/|g" /etc/apt/sources.list; \
        sed -i "s|ports.ubuntu.com/ubuntu-ports/|${APT_MIRROR}/${UBUNTU_PATH}/|g" /etc/apt/sources.list; \
    fi

WORKDIR /app

# Install runtime dependencies (including python3 for script server)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata curl wget python3 \
    && rm -rf /var/lib/apt/lists/*

# Set timezone
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Copy binary
COPY --from=builder /app/n9e /app/n9e

# Copy custom configuration from src/nightingale/etc
COPY src/nightingale/etc/ /app/etc/

# Copy agent installation scripts
COPY src/nightingale/scripts/ /app/scripts/
RUN chmod +x /app/scripts/*.sh

# Copy entrypoint script
COPY src/nightingale/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose ports
# 17000 - HTTP API
# 17001 - HTTP API (mirror)
# 17002 - Script server (agent installation scripts)
# 19000 - Prometheus metrics
EXPOSE 17000 17001 17002 19000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget -q --spider http://localhost:17000/api/v1/health || exit 1

# Use entrypoint script to start both n9e and script server
ENTRYPOINT ["/app/entrypoint.sh"]
CMD []
