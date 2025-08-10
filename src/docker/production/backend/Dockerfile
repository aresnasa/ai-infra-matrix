# Build stage
FROM golang:1.24-alpine AS builder

# 配置Alpine镜像（多镜像自动回退）
RUN set -eux; \
	for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

# Use the same layout as the main backend Dockerfile
WORKDIR /go/src/github.com/aresnasa/ai-infra-matrix/src/backend

# Copy go mod files first for better caching
COPY ../../backend/go.mod ../../backend/go.sum ./
RUN go mod download || (sleep 5 && go mod download) || (export GOPROXY=direct && go mod download)

# Copy backend source
COPY ../../backend/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o init cmd/init/main.go

# Final stage
FROM alpine:latest

# 配置Alpine镜像（多镜像自动回退）
RUN set -eux; \
	for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done

RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

# Copy the binaries from builder stage
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/main .
COPY --from=builder /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/init .

# Create necessary directories
RUN mkdir -p outputs uploads

# Expose port
EXPOSE 8082

# Run the binary
CMD ["./main"]
