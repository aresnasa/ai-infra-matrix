# 网络代理容器 - 将Docker网络中的服务代理到宿主机网络
# 使用HAProxy作为代理服务器，支持TCP代理多种数据库和服务
ARG HAPROXY_VERSION=2.9-alpine
FROM haproxy:${HAPROXY_VERSION}

# 设置维护者信息
ARG VERSION="dev"
ARG HAPROXY_VERSION=2.9-alpine
ARG ALPINE_MIRROR=mirrors.aliyun.com
LABEL maintainer="AI Infra Matrix Team"
LABEL description="Network proxy for exposing Docker internal services to host network"
LABEL version="${VERSION}"
LABEL haproxy.version="${HAPROXY_VERSION}"

# 切换到root用户以安装软件包
USER root

# 配置Alpine镜像源（使用阿里云镜像加速）
RUN set -eux; \
    # 备份原始源
    cp /etc/apk/repositories /etc/apk/repositories.bak || true; \
    # 获取Alpine版本
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2); \
    echo "检测到 Alpine 版本: ${ALPINE_VERSION}"; \
    if [ -n "${ALPINE_MIRROR:-}" ]; then \
        echo "Using custom Alpine mirror: ${ALPINE_MIRROR}"; \
        sed -i "s|dl-cdn.alpinelinux.org|${ALPINE_MIRROR}|g" /etc/apk/repositories; \
        apk update 2>/dev/null; \
    else \
        # 配置阿里云镜像源
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories; \
        # 更新索引（带回退机制）
        if ! apk update 2>/dev/null; then \
            echo "阿里云镜像源失败，回退到官方源..."; \
            mv /etc/apk/repositories.bak /etc/apk/repositories || true; \
            apk update; \
        fi; \
    fi

# 安装必要工具
RUN apk add --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制HAProxy配置文件
COPY haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg

# 创建日志目录
RUN mkdir -p /var/log/haproxy

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD haproxy -c -f /usr/local/etc/haproxy/haproxy.cfg || exit 1

# 暴露端口
# PostgreSQL: 5432
# MySQL: 3306  
# Redis: 6379
# OceanBase: 2881
EXPOSE 5432 3306 6379 2881

# 启动HAProxy
CMD ["haproxy", "-f", "/usr/local/etc/haproxy/haproxy.cfg"]
