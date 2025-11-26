# Nightingale Dockerfile for AI Infra Matrix
# Build from source code in third_party/nightingale

# Stage 1: Build
ARG GOLANG_IMAGE_VERSION={{GOLANG_IMAGE_VERSION}}
FROM golang:${GOLANG_IMAGE_VERSION} AS builder

ARG APT_MIRROR={{APT_MIRROR}}
ARG GO_PROXY=https://goproxy.cn,direct
ARG GITHUB_MIRROR={{GITHUB_MIRROR}}

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

# Build
ENV GOPROXY=${GO_PROXY}

# Step 1: Download front-end release and embed using statik
# This creates the front/statik package required by the build
RUN set -eux; \
    # Install statik tool
    go install github.com/rakyll/statik@latest; \
    # Get latest fe release tag from GitHub API (with mirror support)
    TAG=$(curl -sX GET ${GITHUB_MIRROR}api.github.com/repos/n9e/fe/releases/latest | grep '"tag_name"' | head -1 | awk -F'"' '{print $4}'); \
    echo "Downloading n9e-fe version: ${TAG}"; \
    # Download front-end release
    curl -fsSL -o n9e-fe-${TAG}.tar.gz "${GITHUB_MIRROR}github.com/n9e/fe/releases/download/${TAG}/n9e-fe-${TAG}.tar.gz"; \
    # Extract to pub directory
    tar zxf n9e-fe-${TAG}.tar.gz; \
    # Embed front-end files into Go binary using statik
    $(go env GOPATH)/bin/statik -src=./pub -dest=./front; \
    # Cleanup
    rm -rf n9e-fe-${TAG}.tar.gz

# Step 2: Download Go dependencies
RUN go mod download

# Step 3: Build the binary
RUN go build -ldflags "-w -s" -o n9e ./cmd/center/main.go

# Stage 2: Runtime
FROM ubuntu:22.04

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
ENV N9E_CONFIGS=/app/etc
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
