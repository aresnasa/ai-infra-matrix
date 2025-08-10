# AI基础设施矩阵 - 自定义Nginx镜像
# 支持分布式部署和JupyterHub upstream访问

FROM nginx:alpine

# 安装必要的工具
RUN apk add --no-cache \
    curl \
    bash \
    tzdata

# 设置时区
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建必要的目录结构
RUN mkdir -p /usr/share/nginx/html/sso \
    && mkdir -p /usr/share/nginx/html/jupyterhub \
    && mkdir -p /usr/share/nginx/html/debug \
    && mkdir -p /var/log/nginx \
    && mkdir -p /etc/nginx/conf.d

# 复制项目静态文件
COPY src/shared/sso/ /usr/share/nginx/html/sso/
COPY src/shared/jupyterhub/ /usr/share/nginx/html/jupyterhub/
COPY browser_debug.html /usr/share/nginx/html/debug.html

# 复制自定义Nginx配置
COPY src/nginx/nginx.conf /etc/nginx/nginx.conf

# 创建JupyterHub wrapper页面的符号链接，支持多种访问方式
RUN ln -sf /usr/share/nginx/html/jupyterhub/jupyterhub_wrapper_upstream.html /usr/share/nginx/html/jupyterhub_wrapper.html

# 设置权限
RUN chown -R nginx:nginx /usr/share/nginx/html \
    && chmod -R 755 /usr/share/nginx/html \
    && chown -R nginx:nginx /var/log/nginx \
    && chmod -R 755 /var/log/nginx

# 复制启动脚本
COPY src/nginx/docker-entrypoint.sh /docker-entrypoint.sh

# 设置启动脚本权限
RUN chmod +x /docker-entrypoint.sh

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# 暴露端口
EXPOSE 80 443

# 设置工作目录
WORKDIR /usr/share/nginx/html

# 启动命令
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]

# 元数据标签
LABEL maintainer="AI Infrastructure Team" \
      version="1.0.0" \
      description="AI基础设施矩阵 - 分布式Nginx代理服务" \
      features="SSO,JupyterHub,Distributed,Upstream"
