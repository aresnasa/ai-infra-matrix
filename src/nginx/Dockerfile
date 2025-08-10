# AI基础设施矩阵 - 自定义Nginx镜像
# 支持分布式部署和JupyterHub upstream访问
# 支持开发模式和生产模式

FROM nginx:alpine

# 构建参数 - 控制是否启用调试模式
ARG DEBUG_MODE=false
ARG BUILD_ENV=production

# 配置Alpine镜像（多镜像自动回退）
RUN set -eux; \
	for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
		sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
		apk update && break || true; \
	done

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
COPY src/shared/jupyterhub/browser_debug.html /usr/share/nginx/html/debug.html

# 调试文件处理 - 根据模式决定是否复制完整调试工具
RUN if [ "$DEBUG_MODE" = "true" ]; then \
        echo "🔧 启用调试模式，将复制完整调试工具..."; \
    else \
        echo "🚀 生产模式，将创建简单调试页面"; \
        echo "<h1>Debug tools are disabled in production mode</h1>" > /usr/share/nginx/html/debug/index.html; \
    fi

# 条件复制：仅在调试模式下复制完整调试文件夹
COPY --chown=nginx:nginx src/shared/debug/ /tmp/debug/
RUN if [ "$DEBUG_MODE" = "true" ]; then \
        echo "� 复制调试文件到目标目录..."; \
        cp -r /tmp/debug/* /usr/share/nginx/html/debug/ || echo "⚠️  调试文件复制失败，但继续构建"; \
        echo "✅ 调试文件已复制到 /usr/share/nginx/html/debug/"; \
        ls -la /usr/share/nginx/html/debug/ | head -10; \
    fi && \
    rm -rf /tmp/debug

# 复制环境配置模板
COPY src/nginx/nginx.conf /tmp/nginx.conf.template

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

# 设置环境变量
ENV DEBUG_MODE=${DEBUG_MODE}
ENV BUILD_ENV=${BUILD_ENV}

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
