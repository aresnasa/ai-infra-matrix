# 全局 ARG 声明（用于多个构建阶段）
ARG NODE_IMAGE_VERSION=22.21-bookworm
ARG UBUNTU_VERSION=22.04

# Stage 1: 构建阶段 - 使用 Node.js Debian 镜像
FROM node:${NODE_IMAGE_VERSION} AS build

# Build arguments for versions
ARG NPM_REGISTRY={{NPM_REGISTRY}}
ARG APT_MIRROR={{APT_MIRROR}}
ENV DEBIAN_FRONTEND=noninteractive

# 配置 APT 镜像源（Debian bookworm）
RUN set -eux; \
    apt-get update || \
    (sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources && \
     apt-get update)

# 安装时区工具
RUN apt-get install -y --no-install-recommends tzdata && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY src/frontend/package*.json ./

# 配置npm镜像源（支持多种格式）
RUN set -eux; \
    echo "配置 npm 镜像源: ${NPM_REGISTRY}"; \
    npm config set registry "${NPM_REGISTRY}" || \
    npm config set registry "https://registry.npmmirror.com" || \
    npm config set registry "https://registry.npm.taobao.org" || \
    npm config set registry "https://registry.npmjs.org"; \
    npm config get registry

# 安装依赖
RUN npm install --verbose

# 复制源代码
COPY src/frontend/ .

# 设置构建时环境变量
ARG REACT_APP_API_URL=/api
ARG REACT_APP_JUPYTERHUB_URL=/jupyter
ARG VERSION="dev"
ENV REACT_APP_API_URL=$REACT_APP_API_URL
ENV REACT_APP_JUPYTERHUB_URL=$REACT_APP_JUPYTERHUB_URL
ENV APP_VERSION=${VERSION}

# 构建应用
RUN npm run build

# ========================================
# Stage 2: Production nginx 镜像 (Ubuntu)
# ========================================
FROM ubuntu:${UBUNTU_VERSION}

# Version metadata
ARG VERSION="dev"
ARG APT_MIRROR={{APT_MIRROR}}
ENV APP_VERSION=${VERSION}
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive

# 配置 APT 镜像源并安装 nginx（支持 x86 和 ARM64 双架构）
RUN set -eux; \
    # 备份原始 sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.bak 2>/dev/null || true; \
    # 获取 Ubuntu 版本代号
    . /etc/os-release && CODENAME=${VERSION_CODENAME:-jammy}; \
    # 检测架构: x86_64 使用 ubuntu, arm64/aarch64 使用 ubuntu-ports
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ "${ARCH}" = "amd64" ]; then \
        UBUNTU_PATH="ubuntu"; \
    else \
        UBUNTU_PATH="ubuntu-ports"; \
    fi; \
    echo "Using mirror path: ${UBUNTU_PATH}"; \
    # 尝试配置镜像源
    if [ -n "${APT_MIRROR}" ]; then \
        echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
        echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
        echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
        if ! apt-get update 2>/dev/null; then \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list 2>/dev/null || true; \
            apt-get update; \
        fi; \
    else \
        { \
            echo "deb http://mirrors.aliyun.com/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb http://mirrors.aliyun.com/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb http://mirrors.aliyun.com/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update; \
        } || { \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list 2>/dev/null || true; \
            apt-get update; \
        }; \
    fi; \
    # 安装 nginx 和必要工具
    apt-get install -y --no-install-recommends \
        nginx \
        tzdata \
        curl \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 复制构建的应用到nginx目录
COPY --from=build /app/build /usr/share/nginx/html

# 复制nginx配置文件
COPY src/frontend/nginx.conf /etc/nginx/sites-available/default

# 设置工作目录
WORKDIR /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 启动nginx
CMD ["nginx", "-g", "daemon off;"]

LABEL maintainer="AI Infrastructure Team" \
	org.opencontainers.image.title="ai-infra-frontend" \
	org.opencontainers.image.version="${APP_VERSION}" \
	org.opencontainers.image.description="AI Infra Matrix - Frontend"
