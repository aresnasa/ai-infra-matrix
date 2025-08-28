# 使用Node.js 18作为基础镜像
FROM node:22-alpine AS build

# 配置Alpine镜像（多镜像源智能回退配置）
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
    }

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY package*.json ./

# 安装依赖
RUN npm install

# 复制源代码
COPY . .

# 设置构建时环境变量
ARG REACT_APP_API_URL=/api
ENV REACT_APP_API_URL=$REACT_APP_API_URL

# 设置阿里云npm镜像并构建应用
RUN npm config set registry https://registry.npmmirror.com
RUN npm run build

# 使用nginx作为生产环境镜像
FROM nginx:stable-alpine-perl

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（使用阿里云镜像加速），并设置时区为上海
RUN set -eux; \
	for MIR in mirrors.aliyun.com mirrors.ustc.edu.cn mirrors.tuna.tsinghua.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done; \
	apk add --no-cache tzdata
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
