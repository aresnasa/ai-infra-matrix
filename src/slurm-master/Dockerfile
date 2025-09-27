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

# 使用官方源先安装基础工具，然后配置镜像源
RUN set -eux; \
    # 备份原始sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 重试机制安装基础包
    for i in 1 2 3; do \
        apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && break; \
        echo "尝试第 $i 次安装失败，等待重试..."; \
        sleep 5; \
    done

# 配置阿里云镜像源（直接使用，不做验证）
RUN set -eux; \
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    # 根据架构直接配置阿里云镜像源
    if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        echo "配置ARM64阿里云镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "配置AMD64阿里云镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    fi; \
    # 更新包列表（带重试机制）
    apt-get update || (sleep 5 && apt-get update) || (sleep 10 && apt-get update)

# 安装基础依赖（ca-certificates和curl已在上面安装）
RUN apt-get install -y --no-install-recommends \
    wget \
    netcat \
    procps \
    gettext-base \
    munge \
    libmunge2 \
    supervisor \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
            
# 安装SLURM包前先创建用户，这样安装过程会自动配置权限
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

# 尝试从AppHub安装SLURM 25.05.3版本包（可选，允许失败）
ARG APPHUB_URL=http://apphub:80
RUN set -eux; \
    # 首先尝试从AppHub安装
    echo "deb [trusted=yes] ${APPHUB_URL}/pkgs/slurm-deb ./" > /etc/apt/sources.list.d/ai-infra-slurm.list; \
    if timeout 30 apt-get update && apt-get install -y \
        slurm-smd \
        slurm-smd-client \
        slurm-smd-slurmctld \
        slurm-smd-slurmdbd; then \
        echo "✅ 成功从AppHub安装SLURM包"; \
        SLURM_INSTALLED=true; \
    else \
        echo "⚠️  AppHub不可用，尝试从系统仓库安装SLURM..."; \
        rm -f /etc/apt/sources.list.d/ai-infra-slurm.list; \
        apt-get update; \
        if apt-get install -y slurm-wlm slurm-wlm-basic-plugins slurm-client; then \
            echo "✅ 成功从系统仓库安装SLURM"; \
            SLURM_INSTALLED=true; \
        else \
            echo "❌ SLURM安装失败，容器将以演示模式运行"; \
            SLURM_INSTALLED=false; \
        fi; \
    fi; \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    # 创建标记文件
    if [ "$SLURM_INSTALLED" = "true" ]; then \
        touch /opt/slurm-installed; \
        # 检查实际的二进制文件位置
        which slurmctld > /opt/slurmctld-path || echo "/usr/sbin/slurmctld" > /opt/slurmctld-path; \
        which slurmdbd > /opt/slurmdbd-path || echo "/usr/sbin/slurmdbd" > /opt/slurmdbd-path; \
        echo "SLURM二进制文件路径:"; \
        echo "slurmctld: $(cat /opt/slurmctld-path)"; \
        echo "slurmdbd: $(cat /opt/slurmdbd-path)"; \
    else \
        echo "SLURM未安装，将运行演示模式" > /opt/slurm-demo-mode; \
    fi

# 配置用户组和权限（在SLURM安装后）
RUN usermod -a -G munge slurm && \
    chown -R slurm:slurm /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm /etc/slurm && \
    chown -R munge:munge /etc/munge /var/lib/munge /var/log/munge /var/run/munge && \
    chmod 755 /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm && \
    chmod 700 /etc/munge /var/lib/munge && \
    chmod 755 /var/log/munge /var/run/munge && \
    chmod 640 /etc/munge/munge.key || true

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