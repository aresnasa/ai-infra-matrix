# AI-Infra-Matrix JupyterHub Integration Dockerfile
# 基于python:3.13-alpine优化构建，参考backend Dockerfile
FROM python:3.13-alpine

# 更换alpine下载源为清华大学镜像
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories

# 设置时区为上海
RUN apk --no-cache add ca-certificates curl tzdata
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 设置工作目录
WORKDIR /srv/jupyterhub

# 设置环境变量
ENV PYTHONUNBUFFERED=1
ENV PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple
ENV PIP_TRUSTED_HOST=pypi.tuna.tsinghua.edu.cn

# 安装系统依赖 - alpine版本
RUN apk --no-cache add \
    git \
    build-base \
    linux-headers \
    nodejs \
    npm \
    postgresql-dev \
    pkgconfig \
    libffi-dev \
    openssl-dev \
    && rm -rf /var/cache/apk/*

# 升级pip并安装与conda环境匹配的包版本
RUN pip install --no-cache-dir --upgrade pip && \
    npm install -g configurable-http-proxy && \
    pip install --no-cache-dir \
    jupyterhub==5.3.0 \
    jupyterlab==4.4.5 \
    notebook==7.4.4 \
    requests==2.32.4 \
    tornado==6.5.1 \
    aiohttp==3.12.14 \
    ipython==9.4.0 \
    jupyter-client==8.6.3 \
    jupyter-core==5.8.1 \
    jupyter-server==2.16.0 \
    jupyter-lsp==2.2.6 \
    jupyterlab-server==2.27.3 \
    notebook-shim==0.2.4 \
    traitlets \
    PyJWT \
    psycopg2-binary \
    redis \
    jupyterhub-nativeauthenticator \
    dockerspawner

# 安装最新版本的configurable-http-proxy
RUN npm install -g configurable-http-proxy

# 创建必要的目录
RUN mkdir -p /srv/jupyterhub/config \
    /srv/jupyterhub/data \
    /srv/jupyterhub/notebooks \
    /srv/jupyterhub/templates \
    /srv/jupyterhub/cookies \
    /var/log/jupyterhub \
    /home/jovyan

# 复制自定义认证器和配置文件
COPY ai_infra_auth.py /srv/jupyterhub/
COPY postgres_authenticator.py /srv/jupyterhub/
# 复制配置文件
COPY unified_config.py /srv/jupyterhub/jupyterhub_config.py
COPY requirements.txt /srv/jupyterhub/requirements.txt
COPY init-db.sh /srv/jupyterhub/init-db.sh

# 创建默认环境配置
RUN echo "AI_INFRA_BACKEND_URL=http://backend:8082" > /srv/jupyterhub/.env && \
    echo "JWT_SECRET=your-jwt-secret-key-change-in-production" >> /srv/jupyterhub/.env && \
    echo "JUPYTERHUB_API_TOKEN=your-api-token" >> /srv/jupyterhub/.env

# 设置权限
RUN chown -R root:root /srv/jupyterhub && \
    chmod 755 /srv/jupyterhub/ai_infra_auth.py && \
    chmod +x /srv/jupyterhub/init-db.sh && \
    chmod 755 /home/jovyan

# 设置Python路径
ENV PYTHONPATH="/srv/jupyterhub:$PYTHONPATH"

# 暴露端口
EXPOSE 8000

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/hub/health || exit 1

# 创建启动脚本
RUN echo '#!/bin/bash' > /srv/jupyterhub/start.sh && \
    echo 'set -e' >> /srv/jupyterhub/start.sh && \
    echo 'echo "=== 启动JupyterHub AI-Infra-Matrix集成版本 ==="' >> /srv/jupyterhub/start.sh && \
    echo '' >> /srv/jupyterhub/start.sh && \
    echo 'mkdir -p /srv/data/jupyterhub' >> /srv/jupyterhub/start.sh && \
    echo '' >> /srv/jupyterhub/start.sh && \
    echo 'DB_PATH="/srv/data/jupyterhub/jupyterhub.sqlite"' >> /srv/jupyterhub/start.sh && \
    echo '' >> /srv/jupyterhub/start.sh && \
    echo 'if [ -f "$DB_PATH" ]; then' >> /srv/jupyterhub/start.sh && \
    echo '    echo "删除现有数据库以避免版本冲突..."' >> /srv/jupyterhub/start.sh && \
    echo '    rm -f "$DB_PATH"' >> /srv/jupyterhub/start.sh && \
    echo 'fi' >> /srv/jupyterhub/start.sh && \
    echo '' >> /srv/jupyterhub/start.sh && \
    echo 'echo "初始化新数据库..."' >> /srv/jupyterhub/start.sh && \
    echo 'jupyterhub upgrade-db --config=/srv/jupyterhub/jupyterhub_config.py' >> /srv/jupyterhub/start.sh && \
    echo '' >> /srv/jupyterhub/start.sh && \
    echo 'echo "启动JupyterHub..."' >> /srv/jupyterhub/start.sh && \
    echo 'exec jupyterhub -f /srv/jupyterhub/jupyterhub_config.py' >> /srv/jupyterhub/start.sh && \
    chmod +x /srv/jupyterhub/start.sh

# 启动JupyterHub
CMD ["jupyterhub", "-f", "/srv/jupyterhub/jupyterhub_config.py"]
