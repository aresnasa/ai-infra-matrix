# 使用Node.js 22作为基础镜像
FROM node:22-alpine AS build

# 配置Alpine镜像（多镜像源智能回退配置 - 优化版）
RUN set -eux; \
    # 备份原始repositories文件
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 获取Alpine版本
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2); \
    echo "Detected Alpine version: ${ALPINE_VERSION}"; \
    # 尝试阿里云镜像源（静默失败检测）
    (sed -i "s#https\?://[^/]\+/alpine#https://mirrors.aliyun.com/alpine#g" /etc/apk/repositories && \
     apk update 2>/dev/null) || \
    # 清华源回退
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     sed -i "s#https\?://[^/]\+/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g" /etc/apk/repositories && \
     apk update 2>/dev/null) || \
    # 中科大源回退
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     sed -i "s#https\?://[^/]\+/alpine#https://mirrors.ustc.edu.cn/alpine#g" /etc/apk/repositories && \
     apk update 2>/dev/null) || \
    # 官方源回退（最终方案）
    (cp /etc/apk/repositories.bak /etc/apk/repositories && apk update)

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY package*.json ./

# 配置npm镜像源为中国镜像
RUN npm config set registry https://registry.npmmirror.com

# 安装依赖
RUN npm install --verbose

# 复制源代码
COPY . .

# 设置构建时环境变量
ARG REACT_APP_API_URL=/api
ARG REACT_APP_JUPYTERHUB_URL=/jupyter
ARG VERSION="dev"
ENV REACT_APP_API_URL=$REACT_APP_API_URL
ENV REACT_APP_JUPYTERHUB_URL=$REACT_APP_JUPYTERHUB_URL
ENV APP_VERSION=${VERSION}

# 构建应用
RUN npm run build

# 使用nginx作为生产环境镜像
FROM nginx:stable-alpine-perl

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（多镜像源智能回退 - 与build阶段一致）
RUN set -eux; \
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 尝试阿里云镜像源（静默失败检测）
    (sed -i "s#https\?://[^/]\+/alpine#https://mirrors.aliyun.com/alpine#g" /etc/apk/repositories && \
     apk update 2>/dev/null && apk add --no-cache tzdata) || \
    # 清华源回退
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     sed -i "s#https\?://[^/]\+/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g" /etc/apk/repositories && \
     apk update 2>/dev/null && apk add --no-cache tzdata) || \
    # 中科大源回退
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     sed -i "s#https\?://[^/]\+/alpine#https://mirrors.ustc.edu.cn/alpine#g" /etc/apk/repositories && \
     apk update 2>/dev/null && apk add --no-cache tzdata) || \
    # 官方源回退（最终方案）
    (cp /etc/apk/repositories.bak /etc/apk/repositories && \
     apk update && apk add --no-cache tzdata) || \
    echo "tzdata install skipped"

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制构建的应用到nginx目录
COPY --from=build /app/build /usr/share/nginx/html

# 复制nginx配置文件
COPY nginx.conf /etc/nginx/conf.d/default.conf

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
