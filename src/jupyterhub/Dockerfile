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

# 安装系统依赖
RUN apk update && apk add --no-cache \
    bash \
    shadow \
    su-exec \
    tini \
    nodejs \
    npm \
    build-base \
    linux-headers \
    linux-pam \
    linux-pam-dev \
    python3-dev \
    postgresql-dev \
    postgresql-client \
    redis \
    curl \
    git

# 创建系统用户（解决getpwnam错误）
RUN addgroup -g 1000 jupyterhub && \
    adduser -D -u 1000 -G jupyterhub jupyterhub && \
    adduser -D -u 1001 -s /bin/bash admin && \
    adduser -D -u 1002 -s /bin/bash testuser && \
    echo "admin:password" | chpasswd && \
    echo "testuser:password" | chpasswd

# 先复制requirements.txt并安装依赖（提高缓存效率）
COPY requirements.txt /srv/jupyterhub/requirements.txt
# 优先安装预编译的psutil以避免编译问题
RUN pip install --no-cache-dir psutil --only-binary=psutil || pip install --no-cache-dir psutil
RUN pip install --no-cache-dir -r /srv/jupyterhub/requirements.txt

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

# 创建必要的目录和用户目录
RUN mkdir -p /srv/jupyterhub/config \
    /srv/jupyterhub/data \
    /srv/jupyterhub/notebooks \
    /srv/jupyterhub/templates \
    /srv/jupyterhub/cookies \
    /var/log/jupyterhub \
    /home/jovyan \
    /home/admin \
    /home/testuser \
    /srv/jupyterhub/notebooks/admin \
    /srv/jupyterhub/notebooks/testuser

# 设置用户目录权限
RUN chown admin:admin /home/admin && \
    chown testuser:testuser /home/testuser && \
    chown admin:admin /srv/jupyterhub/notebooks/admin && \
    chown testuser:testuser /srv/jupyterhub/notebooks/testuser

# 复制配置文件（放在最后以提高缓存效率）
# 复制配置文件
COPY jupyterhub_config.py /srv/jupyterhub/jupyterhub_config.py

# 设置权限和环境变量
RUN chown -R root:root /srv/jupyterhub && \
    chmod 755 /home/jovyan

# 设置Python路径
ENV PYTHONPATH="/srv/jupyterhub"

# 暴露端口
EXPOSE 8000

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/hub/health || exit 1

# 启动JupyterHub（使用PostgreSQL，无需SQLite初始化）
CMD ["jupyterhub", "-f", "/srv/jupyterhub/jupyterhub_config.py"]
