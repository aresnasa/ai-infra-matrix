# JupyterHub Backend集成 - Ubuntu 版本
ARG UBUNTU_VERSION=22.04
FROM ubuntu:${UBUNTU_VERSION}

# Build arguments for versions
ARG PIP_VERSION={{PIP_VERSION}}
ARG PYPI_INDEX_URL={{PYPI_INDEX_URL}}
ARG NPM_REGISTRY={{NPM_REGISTRY}}
ARG APT_MIRROR={{APT_MIRROR}}
# Version metadata (overridable at build time)
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive

# 基础环境配置（APT 镜像源智能回退）
RUN set -eux; \
    # 备份原始 sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.bak 2>/dev/null || true; \
    # 获取 Ubuntu 版本代号
    . /etc/os-release && CODENAME=${VERSION_CODENAME:-jammy}; \
    echo "Detected Ubuntu codename: ${CODENAME}"; \
    # 尝试配置镜像源
    if [ -n "${APT_MIRROR}" ]; then \
        echo "尝试自定义镜像源: ${APT_MIRROR}..."; \
        echo "deb http://${APT_MIRROR}/ubuntu/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
        echo "deb http://${APT_MIRROR}/ubuntu/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
        echo "deb http://${APT_MIRROR}/ubuntu/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
        if apt-get update 2>/dev/null; then \
            echo "✓ 成功使用自定义镜像源"; \
        else \
            echo "❌ 自定义镜像源失败，使用官方源..."; \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list 2>/dev/null || true; \
            apt-get update; \
        fi; \
    else \
        # 尝试阿里云镜像源
        { \
            echo "尝试阿里云镜像源..."; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update && echo "✓ 成功使用阿里云镜像源"; \
        } || { \
            echo "❌ 阿里云镜像源失败，尝试清华源..."; \
            echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update && echo "✓ 成功使用清华源"; \
        } || { \
            echo "❌ 清华源失败，尝试中科大源..."; \
            echo "deb https://mirrors.ustc.edu.cn/ubuntu/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb https://mirrors.ustc.edu.cn/ubuntu/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb https://mirrors.ustc.edu.cn/ubuntu/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update && echo "✓ 成功使用中科大源"; \
        } || { \
            echo "❌ 所有国内源都失败，使用官方源..."; \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list 2>/dev/null || true; \
            apt-get update; \
        }; \
    fi; \
    # 安装必需的运行时依赖
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        openssl \
        tzdata \
        bash \
        netcat-openbsd \
        redis-tools \
        git \
        lsof \
        python3 \
        python3-pip \
        python3-venv \
        python3-dev \
        build-essential \
        libcurl4-openssl-dev \
        libssl-dev \
        nodejs \
        npm \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

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
RUN pip3 install --no-cache-dir --upgrade pip==${PIP_VERSION} setuptools wheel --break-system-packages

# 依赖安装
COPY src/jupyterhub/requirements.txt .
# 复制 third_party 目录以支持离线构建
COPY third_party/ /third_party/

# 安装 Python 依赖和 configurable-http-proxy
RUN set -eux; \
    # 步骤1: 配置 npm 镜像源（多重降级）
    echo "配置 npm 镜像源: ${NPM_REGISTRY}"; \
    npm config set registry "${NPM_REGISTRY}" || \
    npm config set registry "https://registry.npmmirror.com" || \
    npm config set registry "https://registry.npm.taobao.org" || \
    npm config set registry "https://registry.npmjs.org"; \
    npm config get registry; \
    \
    # 步骤2: 配置 pip 镜像源
    pip3 config set global.index-url ${PYPI_INDEX_URL} --break-system-packages || true; \
    pip3 config set global.trusted-host mirrors.aliyun.com --break-system-packages || true; \
    \
    # 步骤3: 安装 Python 依赖（带重试）
    pip3 install --no-cache-dir --prefer-binary psutil>=5.9.0 --break-system-packages || \
        (pip3 config set global.index-url https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/ && \
         pip3 config set global.trusted-host mirrors.tuna.tsinghua.edu.cn && \
         pip3 install --no-cache-dir --prefer-binary psutil>=5.9.0 --break-system-packages) || \
        (pip3 config set global.index-url https://pypi.org/simple/ && \
         pip3 config unset global.trusted-host && \
         pip3 install --no-cache-dir --prefer-binary psutil>=5.9.0 --break-system-packages); \
    \
    pip3 install --no-cache-dir -r requirements.txt --break-system-packages || \
        (pip3 config set global.index-url https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/ && \
         pip3 install --no-cache-dir -r requirements.txt --break-system-packages) || \
        (pip3 config set global.index-url https://pypi.org/simple/ && \
         pip3 install --no-cache-dir -r requirements.txt --break-system-packages); \
    \
    # 步骤4: 安装 pycurl（编译安装，带多重降级）
    echo "🔨 开始编译安装 pycurl..."; \
    if [ -f "/third_party/python/pycurl-7.45.3.tar.gz" ]; then \
        echo "📦 使用本地 PyCurl 源码..."; \
        PYCURL_SSL_LIBRARY=openssl pip3 install --no-cache-dir --no-binary=:all: /third_party/python/pycurl-7.45.3.tar.gz --break-system-packages; \
    else \
        PYCURL_SSL_LIBRARY=openssl pip3 install --no-cache-dir --no-binary=:all: pycurl --break-system-packages || \
            (echo "❌ 阿里云源安装失败，尝试清华源..."; \
             pip3 config set global.index-url https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/ && \
             pip3 config set global.trusted-host mirrors.tuna.tsinghua.edu.cn && \
             PYCURL_SSL_LIBRARY=openssl pip3 install --no-cache-dir --no-binary=:all: pycurl --break-system-packages) || \
            (echo "❌ 清华源安装失败，尝试官方 PyPI..."; \
             pip3 config set global.index-url https://pypi.org/simple/ && \
             pip3 config unset global.trusted-host && \
             PYCURL_SSL_LIBRARY=openssl pip3 install --no-cache-dir --no-binary=:all: pycurl --break-system-packages); \
    fi; \
    \
    # 步骤5: 安装 configurable-http-proxy
    npm install -g configurable-http-proxy

# 用户和目录
RUN useradd -m -s /bin/bash admin && \
    useradd -m -s /bin/bash testuser && \
    mkdir -p /var/log/jupyterhub

# 配置时间戳（强制重建配置层）
RUN echo "Build: $(date '+%Y-%m-%d %H:%M:%S')" > /srv/jupyterhub/build_info.txt

# 配置文件（最后复制，优化缓存）
COPY src/jupyterhub/jupyterhub_config.py src/jupyterhub/backend_integrated_config.py src/jupyterhub/simple_config.py src/jupyterhub/kubernetes_spawner_config.py ./

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
