# Build stage
ARG GOLANG_ALPINE_VERSION=1.25-alpine
FROM golang:${GOLANG_ALPINE_VERSION} AS builder

# Build arguments for versions
ARG GOLANG_VERSION=1.25
ARG GO_PROXY=https://goproxy.cn,https://proxy.golang.org,direct

ENV GO111MODULE=on
ENV GOPROXY=${GO_PROXY}

# Version metadata (overridable at build time)
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（多镜像源智能回退配置）
RUN set -eux; \
    # 备份原始repositories文件
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 获取Alpine版本
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2); \
    echo "Detected Alpine version: ${ALPINE_VERSION}"; \
    # 尝试阿里云镜像源
    { \
        echo "尝试阿里云镜像源..."; \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        apk add --no-cache git ca-certificates && \
        echo "成功使用阿里云镜像源"; \
    } || { \
        echo "阿里云镜像源失败，尝试清华源..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        apk add --no-cache git ca-certificates && \
        echo "成功使用清华源"; \
    } || { \
        echo "清华源失败，尝试中科大源..."; \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        apk add --no-cache git ca-certificates && \
        echo "成功使用中科大源"; \
    } || { \
        echo "所有国内源都失败，使用官方源..."; \
        cp /etc/apk/repositories.bak /etc/apk/repositories && \
        apk update && \
        apk add --no-cache git ca-certificates && \
        echo "使用官方源"; \
    }

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOSUMDB=off
ENV CGO_ENABLED=0

WORKDIR /go/src/github.com/aresnasa/ai-infra-matrix/src/backend

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies with retry and fallback
RUN set -eux; \
    echo "Starting Go module download..."; \
    # 第一次尝试：使用默认代理
    if go mod download; then \
        echo "Go modules downloaded successfully"; \
    else \
        echo "First attempt failed, trying with different proxy..."; \
        # 第二次尝试：只使用官方代理
        export GOPROXY=https://proxy.golang.org,direct && \
        if go mod download; then \
            echo "Go modules downloaded with official proxy"; \
        else \
            echo "Second attempt failed, trying direct..."; \
            # 第三次尝试：直接连接
            export GOPROXY=direct && \
            if go mod download; then \
                echo "Go modules downloaded directly"; \
            else \
                echo "All download attempts failed, trying with increased timeout..."; \
                # 第四次尝试：增加超时时间
                export GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct && \
                export GOSUMDB=off && \
                timeout 300 go mod download || \
                (echo "Final attempt with all proxies..." && \
                 export GOPROXY=https://goproxy.cn,https://proxy.golang.org,https://goproxy.io,direct && \
                 go mod download); \
            fi; \
        fi; \
    fi

# Copy source code
COPY . .

# Update go.mod and go.sum
RUN go mod tidy

# Build the application with JupyterHub integration
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init cmd/init/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o test-k8s cmd/test-k8s/main.go

# Final stage - Backend Service (Default)
ARG GOLANG_ALPINE_VERSION=1.25-alpine
FROM golang:${GOLANG_ALPINE_VERSION} AS backend

# Build arguments for versions
ARG GO_PROXY=https://goproxy.cn,https://proxy.golang.org,direct

ENV GO111MODULE=on
ENV GOPROXY=${GO_PROXY}
ENV GOSUMDB=off

# Keep version metadata available in the final image, too
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（多镜像源智能回退配置）
RUN set -eux; \
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 尝试阿里云镜像
    sed -i 's#://[^/]\+/alpine#://mirrors.aliyun.com/alpine#g' /etc/apk/repositories && \
    (apk update 2>/dev/null || \
    # 失败则尝试清华镜像
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     sed -i 's#://[^/]\+/alpine#://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories && \
     apk update 2>/dev/null) || \
    # 再失败则尝试中科大镜像
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     sed -i 's#://[^/]\+/alpine#://mirrors.ustc.edu.cn/alpine#g' /etc/apk/repositories && \
     apk update 2>/dev/null) || \
    # 最后恢复官方源
    (cp /etc/apk/repositories.bak /etc/apk/repositories && apk update))

# 更新apk索引并设置时区为上海，添加必要工具（包括docker CLI）
RUN apk update && apk --no-cache add ca-certificates tzdata netcat-openbsd bash curl openssh-keygen lsof wget docker-cli || \
    apk --no-cache add ca-certificates tzdata busybox-extras bash curl openssh-keygen lsof wget docker-cli

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制 SLURM 运行时安装脚本（在容器启动时执行）
COPY install-slurm-runtime.sh /install-slurm-runtime.sh
RUN chmod +x /install-slurm-runtime.sh

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/main .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/init .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/test-k8s .

# Copy installation scripts for remote execution
COPY scripts/ /root/scripts/
RUN find /root/scripts -type f -name "*.sh" -exec chmod +x {} \;

# 注意：不再从 src/backend/.env 读取配置
# 所有环境变量统一从项目根目录 .env 文件读取，通过 docker-compose.yml 的 env_file 传递
# 或者在 Kubernetes 中通过 ConfigMap/Secret 注入

# Create necessary directories including SSH
RUN mkdir -p outputs uploads ~/.ssh

# Copy shared SSH key pair from build context (统一密钥管理)
# 私钥仅存在于backend，用于远程SSH连接
# Note: SSH密钥会在构建前由build.sh从项目根目录同步到此处
COPY ssh-key/id_rsa /root/.ssh/id_rsa
COPY ssh-key/id_rsa.pub /root/.ssh/id_rsa.pub

# Set proper permissions for SSH directory and keys
RUN ls -la ~/.ssh/ && \
    chmod 700 ~/.ssh && \
    chmod 600 ~/.ssh/id_rsa && \
    chmod 644 ~/.ssh/id_rsa.pub

# Disable strict host key checking for container environments
RUN echo 'Host *' > ~/.ssh/config && \
    echo '  StrictHostKeyChecking no' >> ~/.ssh/config && \
    echo '  UserKnownHostsFile=/dev/null' >> ~/.ssh/config && \
    chmod 600 ~/.ssh/config

# 创建等待数据库的脚本（集成脚本同步到AppHub）
RUN echo '#!/bin/bash' > /wait-for-db.sh && \
    echo 'set -e' >> /wait-for-db.sh && \
    echo '# Sync installation scripts to AppHub' >> /wait-for-db.sh && \
    echo 'if [ -f /root/scripts/sync-scripts-to-apphub.sh ]; then' >> /wait-for-db.sh && \
    echo '  echo ">>> Syncing scripts to AppHub..."' >> /wait-for-db.sh && \
    echo '  /root/scripts/sync-scripts-to-apphub.sh || echo "⚠️  Script sync skipped"' >> /wait-for-db.sh && \
    echo 'fi' >> /wait-for-db.sh && \
    echo '# Install SLURM client if not already installed' >> /wait-for-db.sh && \
    echo 'if [ -f /install-slurm-runtime.sh ] && ! command -v sinfo >/dev/null 2>&1; then' >> /wait-for-db.sh && \
    echo '  echo ">>> Installing SLURM client tools..."' >> /wait-for-db.sh && \
    echo '  /install-slurm-runtime.sh || echo "⚠️  SLURM installation skipped"' >> /wait-for-db.sh && \
    echo 'fi' >> /wait-for-db.sh && \
    echo 'if [ "${OB_ENABLED}" = "true" ]; then' >> /wait-for-db.sh && \
    echo '  echo "Waiting for OceanBase (${OB_HOST:-oceanbase}:${OB_PORT:-2881})..."' >> /wait-for-db.sh && \
    echo '  until nc -z "${OB_HOST:-oceanbase}" "${OB_PORT:-2881}"; do echo "OceanBase unavailable - sleeping"; sleep 1; done' >> /wait-for-db.sh && \
    echo 'else' >> /wait-for-db.sh && \
    echo '  echo "Waiting for PostgreSQL (${DB_HOST:-postgres}:${DB_PORT:-5432})..."' >> /wait-for-db.sh && \
    echo '  until nc -z "${DB_HOST:-postgres}" "${DB_PORT:-5432}"; do echo "PostgreSQL unavailable - sleeping"; sleep 1; done' >> /wait-for-db.sh && \
    echo 'fi' >> /wait-for-db.sh && \
    echo 'echo "Database is ready - starting application"' >> /wait-for-db.sh && \
    echo 'exec "$@"' >> /wait-for-db.sh && \
    chmod +x /wait-for-db.sh

# Expose port
EXPOSE 8082

# Run with database wait
CMD ["bash", "-c", "/wait-for-db.sh ./main"]

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-backend" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Backend service"

# Backend Init Stage - for database initialization
ARG GOLANG_ALPINE_VERSION=1.25-alpine
FROM golang:${GOLANG_ALPINE_VERSION} AS backend-init

# Version metadata (overridable at build time)
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（多镜像源智能回退配置）
RUN set -eux; \
    # 备份原始repositories文件
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 获取Alpine版本
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2); \
    echo "Detected Alpine version: ${ALPINE_VERSION}"; \
    # 尝试阿里云镜像源
    { \
        echo "尝试阿里云镜像源..."; \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        apk add --no-cache git ca-certificates && \
        echo "成功使用阿里云镜像源"; \
    } || { \
        echo "阿里云镜像源失败，尝试清华源..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        apk add --no-cache git ca-certificates && \
        echo "成功使用清华源"; \
    } || { \
        echo "清华源失败，尝试中科大源..."; \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        apk add --no-cache git ca-certificates && \
        echo "成功使用中科大源"; \
    } || { \
        echo "所有国内源都失败，使用官方源..."; \
        cp /etc/apk/repositories.bak /etc/apk/repositories && \
        apk update && \
        apk add --no-cache git ca-certificates && \
        echo "使用官方源"; \
    }

# 更新apk索引并设置时区为上海，添加必要工具
RUN apk update && apk --no-cache add ca-certificates tzdata netcat-openbsd bash curl postgresql-client || \
    apk --no-cache add ca-certificates tzdata busybox-extras bash curl postgresql-client
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /root/

# Copy the init binary from builder stage
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/init .

# 注意：不再从 src/backend/.env 读取配置
# 所有环境变量统一从项目根目录 .env 文件读取，通过 docker-compose.yml 的 env_file 传递
# 或者在 Kubernetes 中通过 ConfigMap/Secret 注入

# Create necessary directories
RUN mkdir -p outputs uploads

# Create wait script for database initialization (OceanBase or PostgreSQL)
RUN echo '#!/bin/bash' > /wait-for-db-init.sh && \
    echo 'set -e' >> /wait-for-db-init.sh && \
    echo 'if [ "${OB_ENABLED}" = "true" ]; then' >> /wait-for-db-init.sh && \
    echo '  echo "Waiting for OceanBase (${OB_HOST:-oceanbase}:${OB_PORT:-2881}) for initialization..."' >> /wait-for-db-init.sh && \
    echo '  until nc -z "${OB_HOST:-oceanbase}" "${OB_PORT:-2881}"; do echo "OceanBase unavailable - sleeping"; sleep 1; done' >> /wait-for-db-init.sh && \
    echo 'else' >> /wait-for-db-init.sh && \
    echo '  echo "Waiting for PostgreSQL (${DB_HOST:-postgres}:${DB_PORT:-5432}) for initialization..."' >> /wait-for-db-init.sh && \
    echo '  until nc -z "${DB_HOST:-postgres}" "${DB_PORT:-5432}"; do echo "PostgreSQL unavailable - sleeping"; sleep 1; done' >> /wait-for-db-init.sh && \
    echo 'fi' >> /wait-for-db-init.sh && \
    echo 'echo "Database is ready - running initialization"' >> /wait-for-db-init.sh && \
    echo './init' >> /wait-for-db-init.sh && \
    chmod +x /wait-for-db-init.sh

# Run initialization
CMD ["bash", "/wait-for-db-init.sh"]

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-backend-init" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Backend initialization service"
