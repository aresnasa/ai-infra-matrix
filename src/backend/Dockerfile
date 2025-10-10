# Build stage
FROM golang:1.25-alpine AS builder

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct

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
FROM golang:1.25-alpine AS backend

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOSUMDB=off

# Keep version metadata available in the final image, too
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（直接使用阿里云镜像源）
RUN set -eux; \
    sed -i 's#://[^/]\+/alpine#://mirrors.aliyun.com/alpine#g' /etc/apk/repositories; \
    apk update || (sleep 5 && apk update) || (sleep 10 && apk update)

# 更新apk索引并设置时区为上海，添加必要工具
RUN apk update && apk --no-cache add ca-certificates tzdata netcat-openbsd bash curl openssh-keygen lsof || \
    apk --no-cache add ca-certificates tzdata busybox-extras bash curl openssh-keygen lsof

# 安装SLURM客户端工具（如果可用）
RUN apk add --no-cache slurm || echo "SLURM not available in Alpine, will use demo data"

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/main .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/init .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/test-k8s .

# Copy environment configuration file (if exists)
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/.env* ./

# Create necessary directories including SSH
RUN mkdir -p outputs uploads ~/.ssh

# Generate SSH key pair for SLURM node management
RUN ssh-keygen -t rsa -b 2048 -f ~/.ssh/id_rsa -N "" -C "ai-infra-backend@container"

# Set proper permissions for SSH directory and keys
RUN chmod 700 ~/.ssh && chmod 600 ~/.ssh/id_rsa && chmod 644 ~/.ssh/id_rsa.pub

# Copy private key to a location that can be accessed after build
RUN cp ~/.ssh/id_rsa /tmp/backend_id_rsa && cp ~/.ssh/id_rsa.pub /tmp/backend_id_rsa.pub

# 创建等待数据库的脚本
RUN echo '#!/bin/bash' > /wait-for-db.sh && \
    echo 'set -e' >> /wait-for-db.sh && \
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
FROM golang:1.25-alpine AS backend-init

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

# Copy environment configuration file (if exists)
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/.env* ./

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
