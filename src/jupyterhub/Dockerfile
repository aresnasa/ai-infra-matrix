# JupyterHub Backend集成 - 精炼版（修复构建问题）
FROM python:3.13-alpine

# 基础环境和构建工具
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories && \
    apk --no-cache add ca-certificates curl tzdata bash shadow nodejs npm netcat-openbsd \
    build-base python3-dev postgresql-dev postgresql-client redis git linux-headers && \
    ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /srv/jupyterhub

# Python环境
ENV PYTHONUNBUFFERED=1 \
    PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple \
    PYTHONPATH="/srv/jupyterhub"

# 升级pip并安装核心工具
RUN pip install --no-cache-dir --upgrade pip setuptools wheel

# 依赖安装（分层优化，优先二进制版本）
COPY requirements.txt .
RUN pip install --no-cache-dir --prefer-binary psutil>=5.9.0 && \
    pip install --no-cache-dir -r requirements.txt && \
    npm install -g configurable-http-proxy

# 用户和目录
RUN adduser -D -s /bin/bash admin && \
    adduser -D -s /bin/bash testuser && \
    mkdir -p /var/log/jupyterhub

# 配置时间戳（强制重建配置层）
RUN echo "Build: $(date '+%Y-%m-%d %H:%M:%S')" > /srv/jupyterhub/build_info.txt

# 配置文件（最后复制，优化缓存）
COPY jupyterhub_config.py backend_integrated_config.py ./

# 创建数据库等待脚本（简化版，只检查连接性）
RUN echo '#!/bin/bash' > /wait-for-db.sh && \
    echo 'set -e' >> /wait-for-db.sh && \
    echo 'host="$1"' >> /wait-for-db.sh && \
    echo 'shift' >> /wait-for-db.sh && \
    echo 'cmd="$@"' >> /wait-for-db.sh && \
    echo 'echo "Waiting for PostgreSQL at $host:5432..."' >> /wait-for-db.sh && \
    echo 'while ! nc -z "$host" 5432; do' >> /wait-for-db.sh && \
    echo '  echo "PostgreSQL is unavailable - sleeping"' >> /wait-for-db.sh && \
    echo '  sleep 2' >> /wait-for-db.sh && \
    echo 'done' >> /wait-for-db.sh && \
    echo 'echo "PostgreSQL is up - waiting for backend initialization..."' >> /wait-for-db.sh && \
    echo 'sleep 10' >> /wait-for-db.sh && \
    echo 'echo "Backend should be initialized - starting JupyterHub"' >> /wait-for-db.sh && \
    echo 'exec $cmd' >> /wait-for-db.sh && \
    chmod +x /wait-for-db.sh

EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/jupyter/hub/api || exit 1

CMD ["/wait-for-db.sh", "postgres", "jupyterhub", "-f", "/srv/jupyterhub/backend_integrated_config.py"]
