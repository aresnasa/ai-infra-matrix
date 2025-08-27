# Build stage
FROM golang:1.25-alpine AS builder

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

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
        echo "成功使用阿里云镜像源"; \
    } || { \
        echo "阿里云镜像源失败，尝试清华源..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用清华源"; \
    } || { \
        echo "清华源失败，尝试中科大源..."; \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用中科大源"; \
    } || { \
        echo "所有国内源都失败，使用官方源..."; \
        cp /etc/apk/repositories.bak /etc/apk/repositories && \
        apk update && \
        echo "使用官方源"; \
    }

ENV GO111MODULE=on

WORKDIR /go/src/github.com/aresnasa/ai-infra-matrix/src/backend

# Copy go mod files
COPY src/backend/go.mod src/backend/go.sum ./

# Download dependencies with retry and fallback
RUN go mod download || (sleep 5 && go mod download) || \
    (export GOPROXY=direct && go mod download)

# Copy source code
COPY src/backend/ .

# Build the application with JupyterHub integration
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init cmd/init/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o test-k8s cmd/test-k8s/main.go

# Final stage - Backend Service (Default)
FROM golang:1.25-alpine AS backend

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

# Keep version metadata available in the final image, too
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（使用阿里云镜像加速）
RUN set -eux; \
    for MIR in mirrors.aliyun.com mirrors.ustc.edu.cn mirrors.tuna.tsinghua.edu.cn dl-cdn.alpinelinux.org; do \
        sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
        apk update && break || true; \
    done

# 更新apk索引并设置时区为上海，添加必要工具
RUN apk update && apk --no-cache add ca-certificates tzdata netcat-openbsd bash curl || \
    apk --no-cache add ca-certificates tzdata busybox-extras bash curl
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/main .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/init .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/test-k8s .

# Copy environment configuration file (if exists)
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/.env* ./

# Create necessary directories
RUN mkdir -p outputs uploads

# 创建等待数据库的脚本
RUN echo '#!/bin/bash' > /wait-for-postgres.sh && \
    echo 'set -e' >> /wait-for-postgres.sh && \
    echo 'echo "Waiting for PostgreSQL to be ready..."' >> /wait-for-postgres.sh && \
    echo 'until nc -z postgres 5432; do' >> /wait-for-postgres.sh && \
    echo '  echo "Database is unavailable - sleeping"' >> /wait-for-postgres.sh && \
    echo '  sleep 1' >> /wait-for-postgres.sh && \
    echo 'done' >> /wait-for-postgres.sh && \
    echo 'echo "Database is ready - starting application"' >> /wait-for-postgres.sh && \
    echo 'exec "$@"' >> /wait-for-postgres.sh && \
    chmod +x /wait-for-postgres.sh

# Expose port
EXPOSE 8082

# Run with database wait
CMD ["bash", "-c", "echo 'Waiting for database...' && while ! nc -z postgres 5432; do sleep 1; done && echo 'Database ready!' && ./main"]

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
        echo "成功使用阿里云镜像源"; \
    } || { \
        echo "阿里云镜像源失败，尝试清华源..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用清华源"; \
    } || { \
        echo "清华源失败，尝试中科大源..."; \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用中科大源"; \
    } || { \
        echo "所有国内源都失败，使用官方源..."; \
        cp /etc/apk/repositories.bak /etc/apk/repositories && \
        apk update && \
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

# Create wait script for database initialization
RUN echo '#!/bin/bash' > /wait-for-postgres-init.sh && \
    echo 'set -e' >> /wait-for-postgres-init.sh && \
    echo 'echo "Waiting for PostgreSQL to be ready for initialization..."' >> /wait-for-postgres-init.sh && \
    echo 'until nc -z postgres 5432; do' >> /wait-for-postgres-init.sh && \
    echo '  echo "Database is unavailable - sleeping"' >> /wait-for-postgres-init.sh && \
    echo '  sleep 1' >> /wait-for-postgres-init.sh && \
    echo 'done' >> /wait-for-postgres-init.sh && \
    echo 'echo "Database is ready - running initialization"' >> /wait-for-postgres-init.sh && \
    echo './init' >> /wait-for-postgres-init.sh && \
    chmod +x /wait-for-postgres-init.sh

# Run initialization
CMD ["bash", "/wait-for-postgres-init.sh"]

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-backend-init" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Backend initialization service"
