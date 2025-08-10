# AI异步测试Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 安装必要的工具
RUN set -eux; \
	for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done; \
	apk add --no-cache git curl jq

# 复制go模块文件
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制测试脚本
COPY tests/ai-async/ ./tests/
COPY test-ai-async.sh ./

# 安装测试依赖
RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest
RUN go install github.com/onsi/gomega/...@latest

# 最终镜像
FROM alpine:latest

# 配置Alpine镜像（多镜像自动回退）
RUN set -eux; \
	for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done

RUN apk add --no-cache ca-certificates curl jq bash

WORKDIR /app

# 从构建阶段复制测试文件
COPY --from=builder /app/tests/ ./tests/
COPY --from=builder /app/test-ai-async.sh ./
COPY --from=builder /go/bin/ /usr/local/bin/

# 添加测试入口脚本
COPY tests/ai-async/run-tests.sh ./

RUN chmod +x ./test-ai-async.sh ./run-tests.sh

CMD ["./run-tests.sh"]
