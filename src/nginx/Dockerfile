# AI基础设施矩阵 - 自定义Nginx镜像 (Ubuntu版)
# 支持分布式部署和JupyterHub upstream访问
# 支持开发模式和生产模式

ARG UBUNTU_VERSION=22.04
FROM ubuntu:${UBUNTU_VERSION}

# Version metadata (can be overridden at build time)
ARG VERSION="dev"
ARG APT_MIRROR=
ENV APP_VERSION=${VERSION}
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive

# 构建参数 - 控制是否启用调试模式
ARG DEBUG_MODE=false
ARG BUILD_ENV=production

# 配置 APT 镜像源并安装 nginx
# 注意: ARM 架构 (aarch64) 需要使用 ubuntu-ports 镜像源
RUN set -eux; \
    # 备份原始 sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.bak 2>/dev/null || true; \
    # 获取 Ubuntu 版本代号
    . /etc/os-release && CODENAME=${VERSION_CODENAME:-jammy}; \
    echo "Detected Ubuntu codename: ${CODENAME}"; \
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
        echo "Using custom APT mirror: ${APT_MIRROR}/${UBUNTU_PATH}"; \
        echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
        echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
        echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
        if ! apt-get update 2>/dev/null; then \
            echo "Custom mirror failed, falling back to defaults..."; \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list 2>/dev/null || true; \
            apt-get update; \
        fi; \
    else \
        { \
            echo "尝试阿里云镜像源..."; \
            echo "deb http://mirrors.aliyun.com/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb http://mirrors.aliyun.com/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb http://mirrors.aliyun.com/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update; \
        } || { \
            echo "阿里云镜像失败，尝试清华镜像..."; \
            echo "deb https://mirrors.tuna.tsinghua.edu.cn/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb https://mirrors.tuna.tsinghua.edu.cn/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb https://mirrors.tuna.tsinghua.edu.cn/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update; \
        } || { \
            echo "清华镜像失败，尝试中科大镜像..."; \
            echo "deb https://mirrors.ustc.edu.cn/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
            echo "deb https://mirrors.ustc.edu.cn/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
            echo "deb https://mirrors.ustc.edu.cn/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
            apt-get update; \
        } || { \
            echo "国内镜像均失败，恢复官方源..."; \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list 2>/dev/null || true; \
            apt-get update; \
        }; \
    fi; \
    # 安装必要的工具 (必须在同一个 RUN 中，确保使用最新的 apt 缓存)
    # nginx-full 包含 stream 模块，用于 TCP/UDP 负载均衡
    apt-get install -y --no-install-recommends \
        nginx-full \
        libnginx-mod-stream \
        curl \
        bash \
        tzdata \
        lsof \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建必要的目录结构
RUN mkdir -p /usr/share/nginx/html/sso \
    && mkdir -p /usr/share/nginx/html/jupyterhub \
    && mkdir -p /usr/share/nginx/html/debug \
    && mkdir -p /var/log/nginx \
    && mkdir -p /etc/nginx/conf.d

# 复制项目静态文件
COPY src/shared/sso/ /usr/share/nginx/html/sso/
COPY src/shared/jupyterhub/ /usr/share/nginx/html/jupyterhub/
# 复制 third_party 目录以支持离线构建
COPY third_party/ /third_party/
# Copy any additional nginx static html (like debug_auth.html)
COPY src/nginx/html/ /usr/share/nginx/html/
# browser_debug.html was archived; serve debug index from existing debug bundle when DEBUG_MODE=true
# Provide a lightweight default debug page otherwise
RUN echo "<html><body><h3>Debug entry</h3><p>See /debug/ when DEBUG_MODE=true.</p></body></html>" > /usr/share/nginx/html/debug.html

# 调试文件处理 - 根据模式决定是否复制完整调试工具
RUN if [ "$DEBUG_MODE" = "true" ]; then \
        echo "🔧 启用调试模式，将复制完整调试工具..."; \
    else \
        echo "🚀 生产模式，将创建简单调试页面"; \
        echo "<h1>Debug tools are disabled in production mode</h1>" > /usr/share/nginx/html/debug/index.html; \
    fi

# 条件复制：仅在调试模式下且debug目录存在时复制调试文件夹
# 先检查源目录是否有内容，然后决定是否复制
    # 复制调试工具
COPY src/shared/debug/ /tmp/debug/
RUN if [ "$DEBUG_MODE" = "true" ]; then \
        echo "🔧 调试模式启用，检查调试文件..."; \
        if [ "$(ls -A /tmp/debug 2>/dev/null)" ]; then \
            echo "📂 复制调试文件到目标目录..."; \
            cp -r /tmp/debug/* /usr/share/nginx/html/debug/ 2>/dev/null || echo "⚠️  调试文件复制失败，但继续构建"; \
            echo "✅ 调试文件已复制到 /usr/share/nginx/html/debug/"; \
            ls -la /usr/share/nginx/html/debug/ | head -10; \
        else \
            echo "📝 调试目录为空，创建默认调试页面"; \
            echo "<h1>Debug Mode Enabled</h1><p>Debug tools directory is empty. Please add debug tools to src/shared/debug/</p>" > /usr/share/nginx/html/debug/index.html; \
        fi; \
    else \
        echo "🚀 生产模式，创建生产调试页面"; \
        echo "<h1>Debug tools are disabled in production mode</h1>" > /usr/share/nginx/html/debug/index.html; \
    fi && \
    rm -rf /tmp/debug

# 复制Nginx主配置与片段到容器
COPY src/nginx/nginx.conf /etc/nginx/nginx.conf
COPY src/nginx/conf.d/ /etc/nginx/conf.d/

# 复制 SSL 证书 (如果存在)
# 证书由 ./build.sh ssl-setup 生成到 src/nginx/ssl/
# 如果不存在则创建空目录
RUN mkdir -p /etc/nginx/ssl
COPY src/nginx/ssl/ /etc/nginx/ssl/

# 创建JupyterHub wrapper页面的符号链接，支持多种访问方式
RUN ln -sf /usr/share/nginx/html/jupyterhub/jupyterhub_wrapper_upstream.html /usr/share/nginx/html/jupyterhub_wrapper.html

# 设置权限 (Ubuntu nginx 使用 www-data 用户)
RUN chown -R www-data:www-data /usr/share/nginx/html \
    && chmod -R 755 /usr/share/nginx/html \
    && chown -R www-data:www-data /var/log/nginx \
    && chmod -R 755 /var/log/nginx

# 复制启动脚本
COPY src/nginx/docker-entrypoint.sh /docker-entrypoint.sh

# 设置启动脚本权限
RUN chmod +x /docker-entrypoint.sh

# 设置环境变量
ENV DEBUG_MODE=${DEBUG_MODE}
ENV BUILD_ENV=${BUILD_ENV}

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD curl -f http://127.0.0.1/health || exit 1

# 暴露端口
EXPOSE 80 443

# 设置工作目录
WORKDIR /usr/share/nginx/html

# 启动命令
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]

# 元数据标签
LABEL maintainer="AI Infrastructure Team" \
    version="${APP_VERSION}" \
      description="AI基础设施矩阵 - 分布式Nginx代理服务" \
    features="SSO,JupyterHub,Distributed,Upstream" \
    org.opencontainers.image.title="ai-infra-nginx" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix custom nginx gateway"
