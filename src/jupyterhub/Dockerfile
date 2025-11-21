# JupyterHub Backend集成 - 精炼版（修复构建问题）
ARG PYTHON_ALPINE_VERSION=3.14-alpine
FROM python:${PYTHON_ALPINE_VERSION}

# Build arguments for versions
ARG PIP_VERSION=24.2
ARG PYPI_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG ALPINE_MIRROR

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
    # 尝试自定义镜像源
    if [ -n "${ALPINE_MIRROR}" ]; then \
        echo "尝试自定义镜像源: ${ALPINE_MIRROR}..."; \
        echo "https://${ALPINE_MIRROR}/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://${ALPINE_MIRROR}/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update && \
        echo "成功使用自定义镜像源"; \
    else \
        # 尝试阿里云镜像源
        { \
            echo "尝试阿里云镜像源..."; \
            echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
            echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
            apk update && \
            echo "成功使用阿里云镜像源"; \
        } || { \
            echo "阿里云镜像源失败，尝试清华源..."; \
            echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
            echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
            apk update && \
            echo "成功使用清华源"; \
        } || { \
            echo "清华源失败，尝试中科大源..."; \
            echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
            echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
            apk update && \
            echo "成功使用中科大源"; \
        } || { \
            echo "所有国内源都失败，使用官方源..."; \
            cp /etc/apk/repositories.bak /etc/apk/repositories && \
            apk update && \
            echo "使用官方源"; \
        }; \
    fi; \
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

# 使用配置的 PyPI 镜像（带官方 PyPI 作为备用源）
ENV PIP_INDEX_URL="${PYPI_INDEX_URL}" \
    PIP_EXTRA_INDEX_URL="${PYPI_INDEX_URL}" \
    PIP_TRUSTED_HOST="mirrors.aliyun.com" \
    PIP_TIMEOUT=60

# 升级pip并安装核心工具
RUN pip install --no-cache-dir --upgrade pip==${PIP_VERSION} setuptools wheel

# 依赖安装（分层优化，优先二进制版本）
COPY requirements.txt .
# 复制 third_party 目录以支持离线构建
COPY third_party/ /third_party/

# Install Node.js and temporary build deps to compile psutil on Alpine, then remove them
RUN set -eux; \
    # 步骤1: 配置 Alpine 镜像源（多重降级）
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d. -f1,2); \
    if [ -n "${ALPINE_MIRROR}" ]; then \
        echo "https://${ALPINE_MIRROR}/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories; \
        echo "https://${ALPINE_MIRROR}/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories; \
    else \
        # 尝试阿里云镜像
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories; \
    fi; \
    if ! apk update 2>/dev/null; then \
        echo "❌ 阿里云镜像失败，尝试清华镜像..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories; \
        if ! apk update 2>/dev/null; then \
            echo "❌ 清华镜像失败，尝试中科大镜像..."; \
            echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories; \
            echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories; \
            if ! apk update 2>/dev/null; then \
                echo "❌ 中科大镜像失败，使用官方镜像源..."; \
                mv /etc/apk/repositories.bak /etc/apk/repositories; \
                apk update || (sleep 5 && apk update) || (sleep 10 && apk update); \
            fi; \
        fi; \
    fi; \
    \
    # 步骤2: 安装系统依赖
    apk add --no-cache nodejs npm && \
    apk add --no-cache --virtual .build-deps \
        build-base \
        musl-dev \
        linux-headers \
        python3-dev \
        curl-dev \
        openssl-dev \
        zlib-dev; \
    \
    # 步骤3: 配置 npm 镜像源（多重降级）
    npm config set registry ${NPM_REGISTRY} || \
    npm config set registry https://registry.npm.taobao.org || \
    npm config set registry https://registry.npmjs.org; \
    \
    # 步骤4: 配置 pip 镜像源（多重降级）
    pip config set global.index-url ${PYPI_INDEX_URL} || true; \
    pip config set global.trusted-host mirrors.aliyun.com || true; \
    \
    # 步骤5: 安装 Python 依赖（带重试）
    pip install --no-cache-dir --prefer-binary psutil>=5.9.0 || \
        (pip config set global.index-url https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/ && \
         pip config set global.trusted-host mirrors.tuna.tsinghua.edu.cn && \
         pip install --no-cache-dir --prefer-binary psutil>=5.9.0) || \
        (pip config set global.index-url https://pypi.org/simple/ && \
         pip config unset global.trusted-host && \
         pip install --no-cache-dir --prefer-binary psutil>=5.9.0); \
    \
    pip install --no-cache-dir -r requirements.txt || \
        (pip config set global.index-url https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/ && \
         pip install --no-cache-dir -r requirements.txt) || \
        (pip config set global.index-url https://pypi.org/simple/ && \
         pip install --no-cache-dir -r requirements.txt); \
    \
    # 步骤6: 安装 pycurl（编译安装，带多重降级）
    echo "🔨 开始编译安装 pycurl..."; \
    if [ -f "/third_party/python/pycurl-7.45.3.tar.gz" ]; then \
        echo "📦 使用本地 PyCurl 源码..."; \
        PYCURL_SSL_LIBRARY=openssl pip install --no-cache-dir --no-binary=:all: /third_party/python/pycurl-7.45.3.tar.gz; \
    else \
        PYCURL_SSL_LIBRARY=openssl pip install --no-cache-dir --no-binary=:all: pycurl || \
            (echo "❌ 阿里云源安装失败，尝试清华源..."; \
             pip config set global.index-url https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/ && \
             pip config set global.trusted-host mirrors.tuna.tsinghua.edu.cn && \
             PYCURL_SSL_LIBRARY=openssl pip install --no-cache-dir --no-binary=:all: pycurl) || \
            (echo "❌ 清华源安装失败，尝试官方 PyPI..."; \
             pip config set global.index-url https://pypi.org/simple/ && \
             pip config unset global.trusted-host && \
             PYCURL_SSL_LIBRARY=openssl pip install --no-cache-dir --no-binary=:all: pycurl); \
    fi; \
    \
    # 步骤7: 安装 configurable-http-proxy
    npm install -g configurable-http-proxy; \
    \
    # 步骤8: 清理构建依赖
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
