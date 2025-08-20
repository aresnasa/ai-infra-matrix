# 使用Node.js 18作为基础镜像
FROM node:18-alpine AS build

# 配置Alpine镜像（使用阿里云镜像加速）
RUN set -eux; \
	for MIR in mirrors.aliyun.com mirrors.ustc.edu.cn mirrors.tuna.tsinghua.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY package*.json ./

# 安装依赖
RUN npm install

# 复制源代码
COPY . .

# 设置构建时环境变量
ARG REACT_APP_API_URL=http://localhost:8083/api
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

# 复制docker-desktop-ca.crt并更新系统信任
COPY docker-desktop-ca.crt /usr/local/share/ca-certificates/docker-desktop-ca.crt
RUN update-ca-certificates || true

# 暴露端口
EXPOSE 80

# 启动nginx
CMD ["nginx", "-g", "daemon off;"]

LABEL maintainer="AI Infrastructure Team" \
	org.opencontainers.image.title="ai-infra-frontend" \
	org.opencontainers.image.version="${APP_VERSION}" \
	org.opencontainers.image.description="AI Infra Matrix - Frontend"
