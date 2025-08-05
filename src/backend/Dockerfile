# Build stage
FROM golang:1.24-alpine AS builder

# 更换alpine下载源为清华大学镜像
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

WORKDIR /go/src/github.com/aresnasa/ai-infra-matrix/src/backend

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with JupyterHub integration
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init cmd/init/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o test-k8s cmd/test-k8s/main.go

# Final stage
FROM golang:1.24-alpine

# 更换alpine下载源为清华大学镜像
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories

# 设置时区为上海，添加netcat用于数据库等待
RUN apk --no-cache add ca-certificates curl tzdata netcat-openbsd
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
CMD ["/wait-for-postgres.sh", "./main"]
