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

WORKDIR /go/src/ansible-playbook-generator-backend

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

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

# Copy the binary from builder stage
COPY --from=builder /go/src/ansible-playbook-generator/web-v2/backend/main .
COPY --from=builder /go/src/ansible-playbook-generator/web-v2/backend/init .

# Create necessary directories
RUN mkdir -p outputs uploads

# Expose port
EXPOSE 8082

# Run the binary
CMD ["./main"]
