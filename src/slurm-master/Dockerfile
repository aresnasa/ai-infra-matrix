# AI Infrastructure Matrix - SLURM Master Service
# 基于Ubuntu构建SLURM控制节点
FROM ubuntu:22.04

ARG VERSION=v0.3.6-dev
LABEL maintainer="AI Infrastructure Team"
LABEL version="${VERSION}"
LABEL description="AI Infrastructure Matrix SLURM Master Service"

# 避免交互式安装提示
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# 首先备份原始sources.list并配置阿里云源（针对ARM64架构）
RUN cp /etc/apt/sources.list /etc/apt/sources.list.backup && \
    echo "# 阿里云Ubuntu 22.04 LTS ARM64源" > /etc/apt/sources.list && \
    echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse" >> /etc/apt/sources.list && \
    echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
    echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
    echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list && \
    apt-get update

# 安装基础依赖
RUN apt-get install -y --no-install-recommends \
    curl \
    wget \
    ca-certificates \
    netcat-openbsd \
    procps \
    gettext-base \
    munge \
    libmunge2 \
    supervisor \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
            
# 创建SLURM用户和组
RUN groupadd -r slurm && useradd -r -g slurm -d /var/lib/slurm -s /bin/bash slurm

# 创建必要的目录
RUN mkdir -p /etc/slurm \
    /var/spool/slurm/slurmctld \
    /var/spool/slurm/slurmdbd \
    /var/log/slurm \
    /var/lib/slurm \
    /var/run/slurm \
    /etc/munge \
    /var/lib/munge \
    /var/log/munge \
    /var/run/munge \
    /srv/shared \
    /etc/supervisor/conf.d

# 设置目录权限
RUN chown -R slurm:slurm /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm /etc/slurm && \
    chown -R munge:munge /etc/munge /var/lib/munge /var/log/munge /var/run/munge && \
    chmod 755 /var/spool/slurm /var/log/slurm /var/lib/slurm && \
    chmod 700 /etc/munge /var/lib/munge && \
    chmod 755 /var/log/munge /var/run/munge

# 从AppHub安装SLURM包
ARG APPHUB_URL=http://192.168.0.199:8090
RUN echo "deb [trusted=yes] ${APPHUB_URL}/pkgs/slurm-deb ./" > /etc/apt/sources.list.d/ai-infra-slurm.list \
    && apt-get update && apt-get install -y \
    slurm-smd \
    slurm-smd-client \
    slurm-smd-slurmctld \
    slurm-smd-slurmdbd \
    && rm -rf /var/lib/apt/lists/*

# 复制配置文件模板
COPY config/ /etc/slurm-templates/
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY healthcheck.sh /usr/local/bin/healthcheck.sh
COPY supervisord.conf /etc/supervisor/conf.d/slurm.conf

# 设置脚本权限
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/healthcheck.sh

# 暴露端口
EXPOSE 6817 6818

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=90s --retries=3 \
    CMD /usr/local/bin/healthcheck.sh

# 设置工作目录
WORKDIR /etc/slurm

# 启动脚本
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["start-services"]