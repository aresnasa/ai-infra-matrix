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

ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG N9E_FE_VERSION=v8.4.1
ARG GITHUB_MIRROR=https://ghfast.top/
ARG GITHUB_PROXY={{GITHUB_PROXY}}
# Set VITE_PREFIX to match Nightingale's BasePath configuration
# This ensures React Router and all asset paths use the correct prefix
# Backend config.toml must have: BasePath = "/nightingale"
# Nginx must NOT strip the /nightingale prefix (no trailing slash in proxy_pass)
ARG VITE_PREFIX=/nightingale

WORKDIR /fe

# Download frontend source code from GitHub
# The n9e-fe-src directory may not be included in Docker build context
# because it's a nested git repository (not a submodule)
# Uses 3-way fallback: GITHUB_MIRROR → GITHUB_PROXY → Direct
RUN set -eux; \
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
RUN set -eux; \
    if [ -n "${NPM_REGISTRY}" ]; then \
        npm config set registry ${NPM_REGISTRY}; \
    fi; \
    npm install --legacy-peer-deps; \
    echo "Building frontend with VITE_PREFIX=${VITE_PREFIX}"; \
    VITE_PREFIX=${VITE_PREFIX} npm run build; \
    # The build output is in /fe/pub directory
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

# Build
ENV GOPROXY=${GO_PROXY}

# Step 1: Embed front-end files into Go binary using statik
RUN set -eux; \
    # Install statik tool
    go install github.com/rakyll/statik@latest; \
    # Embed front-end files
    $(go env GOPATH)/bin/statik -src=./pub -dest=./front

# Step 2: Download Go dependencies
RUN go mod download

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
