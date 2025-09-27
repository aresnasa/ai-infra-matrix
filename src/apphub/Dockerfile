FROM nginx:stable

# 配置APT镜像源并安装基础工具（简化版，适配nginx镜像）
RUN set -eux; \
    # 检查并创建sources.list（nginx镜像可能没有标准配置）
    if [ ! -f /etc/apt/sources.list ]; then \
        echo "创建默认sources.list..."; \
        echo "deb http://deb.debian.org/debian bookworm main" > /etc/apt/sources.list; \
        echo "deb http://deb.debian.org/debian bookworm-updates main" >> /etc/apt/sources.list; \
        echo "deb http://security.debian.org/debian-security bookworm-security main" >> /etc/apt/sources.list; \
    fi; \
    # 备份原始配置
    cp /etc/apt/sources.list /etc/apt/sources.list.bak || true; \
    # 更新包列表并安装基础工具
    apt-get update && apt-get install -y ca-certificates curl; \
    # 尝试配置阿里云镜像源
    if curl -s --connect-timeout 5 http://mirrors.aliyun.com/debian/ >/dev/null 2>&1; then \
        echo "配置阿里云HTTP镜像源..."; \
        echo "deb http://mirrors.aliyun.com/debian/ bookworm main" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/debian/ bookworm-updates main" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/debian-security/ bookworm-security main" >> /etc/apt/sources.list; \
        apt-get update || { \
            echo "阿里云源失败，回退到官方源"; \
            cp /etc/apt/sources.list.bak /etc/apt/sources.list; \
            apt-get update; \
        }; \
    else \
        echo "网络测试失败，使用官方源"; \
    fi

# Install tools for repo management (deb packages only)  
RUN apt-get install -y apt-utils dpkg-dev createrepo-c && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Create directories for packages
RUN mkdir -p /usr/share/nginx/html/deb /usr/share/nginx/html/rpm

# Expose port
EXPOSE 80

# Entrypoint to regenerate indexes if needed
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]