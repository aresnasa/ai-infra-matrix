# JupyterHub Backend集成 - 精炼版（修复构建问题）
FROM python:3.13-alpine

# Version metadata (overridable at build time)
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}
ENV TZ=Asia/Shanghai

# 基础环境和构建工具（多镜像源智能回退配置）
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
    }; \
    # 安装必需的运行时依赖
    apk add --no-cache \
        ca-certificates \
        curl \
        openssl \
        tzdata \
        bash \
        shadow \
        netcat-openbsd \
        redis \
        git \
        lsof \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /srv/jupyterhub

# Python环境
ENV PYTHONUNBUFFERED=1 \
    PYTHONPATH="/srv/jupyterhub"

# 使用阿里云 PyPI 镜像（带官方 PyPI 作为备用源）
ENV PIP_INDEX_URL="https://mirrors.aliyun.com/pypi/simple/" \
    PIP_EXTRA_INDEX_URL="https://mirrors.aliyun.com/pypi/simple/" \
    PIP_TRUSTED_HOST="mirrors.aliyun.com" \
    PIP_TIMEOUT=60

# 升级pip并安装核心工具
RUN pip install --no-cache-dir --upgrade pip setuptools wheel

# 依赖安装（分层优化，优先二进制版本）
COPY requirements.txt .
# Install Node.js and temporary build deps to compile psutil on Alpine, then remove them
RUN set -eux; \
    apk update || (sleep 5 && apk update) || (sleep 10 && apk update); \
    apk add --no-cache nodejs npm && \
    apk add --no-cache --virtual .build-deps \
        build-base \
        musl-dev \
        linux-headers \
        python3-dev \
        curl-dev \
        openssl-dev \
        zlib-dev \
    ; \
    # 配置npm使用阿里云镜像源
    npm config set registry https://registry.npmmirror.com; \
    # 设置node-gyp镜像环境变量
    # 配置pip使用阿里云镜像源
    pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/; \
    pip config set global.trusted-host mirrors.aliyun.com; \
    # 安装Python依赖
    pip install --no-cache-dir --prefer-binary psutil>=5.9.0; \
    pip install --no-cache-dir -r requirements.txt; \
    PYCURL_SSL_LIBRARY=openssl pip install --no-cache-dir --no-binary=:all: pycurl; \
    # 安装configurable-http-proxy
    npm install -g configurable-http-proxy; \
    apk del .build-deps

# 用户和目录
RUN adduser -D -s /bin/bash admin && \
    adduser -D -s /bin/bash testuser && \
    mkdir -p /var/log/jupyterhub

# 配置时间戳（强制重建配置层）
RUN echo "Build: $(date '+%Y-%m-%d %H:%M:%S')" > /srv/jupyterhub/build_info.txt

# 配置文件（最后复制，优化缓存）
COPY jupyterhub_config.py backend_integrated_config.py simple_config.py kubernetes_spawner_config.py ./

# 创建数据库等待脚本（简化版，只检查连接性）
RUN echo '#!/bin/bash' > /wait-for-db.sh && \
    echo 'set -e' >> /wait-for-db.sh && \
    echo 'host="$1"' >> /wait-for-db.sh && \
    echo 'shift' >> /wait-for-db.sh && \
    echo 'cmd="$@"' >> /wait-for-db.sh && \
    echo 'echo "Waiting for PostgreSQL at $host:5432..."' >> /wait-for-db.sh && \
    echo 'while ! nc -z "$host" 5432; do' >> /wait-for-db.sh && \
    echo '  echo "PostgreSQL is unavailable - sleeping"' >> /wait-for-db.sh && \
    echo '  sleep 2' >> /wait-for-db.sh && \
    echo 'done' >> /wait-for-db.sh && \
    echo 'echo "PostgreSQL is up - waiting for backend initialization..."' >> /wait-for-db.sh && \
    echo 'sleep 10' >> /wait-for-db.sh && \
    echo 'echo "Backend should be initialized - starting JupyterHub"' >> /wait-for-db.sh && \
    echo 'exec $cmd' >> /wait-for-db.sh && \
    chmod +x /wait-for-db.sh

EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://127.0.0.1:8000/jupyter/hub/api || exit 1

CMD ["/wait-for-db.sh", "postgres", "jupyterhub", "-f", "/srv/jupyterhub/backend_integrated_config.py"]

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-jupyterhub" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - JupyterHub with integrated backend auth"
