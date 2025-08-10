# Build stage
FROM golang:1.24-alpine AS builder

# 配置Alpine镜像（多镜像自动回退）
RUN set -eux; \
    for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
        sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
        apk update && break || true; \
    done

ENV GO111MODULE=on

WORKDIR /go/src/github.com/aresnasa/ai-infra-matrix/src/backend

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with retry and fallback
RUN go mod download || (sleep 5 && go mod download) || \
    (export GOPROXY=direct && go mod download)

# Copy source code
COPY . .

# Build the application with JupyterHub integration
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init cmd/init/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o test-k8s cmd/test-k8s/main.go

# Final stage
FROM golang:1.24-alpine

# 配置Alpine镜像（多镜像自动回退）
RUN set -eux; \
    for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
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

# Copy docker-desktop-ca.crt and update certificates
COPY docker-desktop-ca.crt /usr/local/share/ca-certificates/docker-desktop-ca.crt
RUN update-ca-certificates

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
