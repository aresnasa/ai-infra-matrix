# Nightingale Dockerfile for AI Infra Matrix
# Build from source code in third_party/nightingale

# 全局 ARG 声明 - 必须在所有 FROM 之前声明才能在 FROM 中使用
ARG GOLANG_IMAGE={{GOLANG_IMAGE}}
ARG UBUNTU_IMAGE={{UBUNTU_IMAGE}}

# Stage 1: Build
FROM ${GOLANG_IMAGE} AS builder

ARG APT_MIRROR={{APT_MIRROR}}
ARG GO_PROXY=https://goproxy.cn,direct
ARG GITHUB_MIRROR={{GITHUB_MIRROR}}
ARG INTERNAL_FILE_SERVER={{INTERNAL_FILE_SERVER}}
ARG N9E_FE_VERSION={{N9E_FE_VERSION}}

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

# Copy local n9e-fe package if exists (优先使用本地文件)
COPY third_party/n9e-fe/ /tmp/n9e-fe/

# Build
ENV GOPROXY=${GO_PROXY}

# Step 1: Get front-end release and embed using statik
# This creates the front/statik package required by the build
# 
# 支持四种下载模式（按优先级）:
# 1. 本地文件: third_party/n9e-fe/n9e-fe-${VERSION}.tar.gz (优先)
# 2. 内网环境 (INTERNAL_FILE_SERVER 非默认值): 从内部文件服务器下载
# 3. 互联网环境 + 指定版本 (N9E_FE_VERSION 非空): 使用 GitHub 镜像下载指定版本
# 4. 互联网环境 + 自动版本: 从 GitHub API 获取最新版本并下载
RUN set -eux; \
    # Install statik tool
    go install github.com/rakyll/statik@latest; \
    \
    TAG="${N9E_FE_VERSION:-v8.4.1}"; \
    \
    # 优先使用本地文件
    if [ -f "/tmp/n9e-fe/n9e-fe-${TAG}.tar.gz" ]; then \
        echo "Local mode: Using local n9e-fe ${TAG} from third_party/n9e-fe/"; \
        cp /tmp/n9e-fe/n9e-fe-${TAG}.tar.gz .; \
    # 检查是否配置了真实的内网文件服务器（非默认占位值）
    elif [ -n "${INTERNAL_FILE_SERVER}" ] && \
         [ "${INTERNAL_FILE_SERVER}" != "http://files.example.com" ] && \
         [ "${INTERNAL_FILE_SERVER}" != "" ]; then \
        # 内网模式: 从内部文件服务器下载
        echo "Intranet mode: Downloading n9e-fe ${TAG} from internal server"; \
        curl -fsSL -o n9e-fe-${TAG}.tar.gz "${INTERNAL_FILE_SERVER}/nightingale/n9e-fe-${TAG}.tar.gz"; \
    elif [ -n "${N9E_FE_VERSION}" ]; then \
        # 指定版本模式: 使用 GitHub 镜像下载
        echo "Fixed version mode: Downloading n9e-fe ${TAG}"; \
        curl -fsSL -o n9e-fe-${TAG}.tar.gz "${GITHUB_MIRROR}github.com/n9e/fe/releases/download/${TAG}/n9e-fe-${TAG}.tar.gz"; \
    else \
        # 自动版本模式: 从 GitHub API 获取最新版本
        echo "Auto version mode: Fetching latest version from GitHub API"; \
        # 使用 GitHub 镜像代理 API（如果可用）
        if [ -n "${GITHUB_MIRROR}" ]; then \
            TAG=$(curl -sX GET "${GITHUB_MIRROR}https://api.github.com/repos/n9e/fe/releases/latest" 2>/dev/null | grep '"tag_name"' | head -1 | awk -F'"' '{print $4}'); \
        fi; \
        # 如果镜像代理失败，尝试直连
        if [ -z "${TAG}" ]; then \
            TAG=$(curl -sX GET https://api.github.com/repos/n9e/fe/releases/latest | grep '"tag_name"' | head -1 | awk -F'"' '{print $4}'); \
        fi; \
        if [ -z "${TAG}" ]; then \
            echo "ERROR: Failed to get version from GitHub API. Please set N9E_FE_VERSION or INTERNAL_FILE_SERVER"; \
            exit 1; \
        fi; \
        echo "Downloading n9e-fe version: ${TAG}"; \
        curl -fsSL -o n9e-fe-${TAG}.tar.gz "${GITHUB_MIRROR}github.com/n9e/fe/releases/download/${TAG}/n9e-fe-${TAG}.tar.gz"; \
    fi; \
    \
    # Extract to pub directory
    tar zxf n9e-fe-${TAG}.tar.gz; \
    # Embed front-end files into Go binary using statik
    $(go env GOPATH)/bin/statik -src=./pub -dest=./front; \
    # Cleanup
    rm -rf n9e-fe-${TAG}.tar.gz /tmp/n9e-fe

# Step 2: Download Go dependencies
RUN go mod download

# Step 3: Build the binary
RUN go build -ldflags "-w -s" -o n9e ./cmd/center/main.go

# Stage 2: Runtime
# UBUNTU_IMAGE 已在文件顶部全局声明
FROM ${UBUNTU_IMAGE}

ARG APT_MIRROR={{APT_MIRROR}}
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
