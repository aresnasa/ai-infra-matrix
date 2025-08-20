# JupyterHub Backend集成 - 精炼版（修复构建问题）
FROM python:3.13-alpine

# 基础环境和构建工具（多镜像自动回退）
RUN set -eux; \
    for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
        sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
        apk update && break || true; \
    done; \
    apk --no-cache add ca-certificates curl tzdata bash shadow nodejs npm \
    build-base python3-dev postgresql-dev redis git linux-headers && \
    ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /srv/jupyterhub

# Python环境
ENV PYTHONUNBUFFERED=1 \
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

EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/hub/health || exit 1

CMD ["jupyterhub", "-f", "/srv/jupyterhub/jupyterhub_config.py"]
