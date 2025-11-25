# 全局 ARG 声明（用于多个构建阶段）
ARG NODE_ALPINE_VERSION=22-alpine
ARG NGINX_VERSION=stable-alpine-perl

# 使用Node.js作为基础镜像
FROM node:${NODE_ALPINE_VERSION} AS build

# Build arguments for versions
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG ALPINE_MIRROR=mirrors.aliyun.com
# 配置Alpine镜像（简化版 - 优先使用默认源，失败则切换阿里云）
RUN set -eux; \
    apk add --no-cache tzdata || \
    (sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
     apk add --no-cache tzdata)

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY src/frontend/package*.json ./

# 配置npm镜像源
RUN npm config set registry ${NPM_REGISTRY}

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
# Stage 2: Production nginx 镜像
# ========================================
FROM nginx:${NGINX_VERSION}

# Version metadata (需要在 FROM 之后重新声明ARG，因为跨越了构建阶段)
ARG VERSION="dev"
ARG ALPINE_MIRROR=mirrors.aliyun.com
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（简化版 - 优先使用默认源，失败则切换阿里云）
RUN set -eux; \
    apk add --no-cache tzdata || \
    (sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
     apk add --no-cache tzdata)

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制构建的应用到nginx目录
COPY --from=build /app/build /usr/share/nginx/html

# 复制nginx配置文件
COPY src/frontend/nginx.conf /etc/nginx/conf.d/default.conf

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
