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

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies with retry and fallback
RUN go mod download || (sleep 5 && go mod download) || \
    (export GOPROXY=direct && go mod download)

# Copy source code
COPY . .

    # Add required dependencies for Kafka
    RUN go get github.com/IBM/sarama@latest || \
        (sleep 3 && go get github.com/IBM/sarama@latest) || \
        (export GOPROXY=direct && go get github.com/IBM/sarama@latest)# Update go.mod and go.sum
RUN go mod tidy

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
