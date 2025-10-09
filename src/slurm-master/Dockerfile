# AI Infrastructure Matrix - SLURM Master Service
# 基于Ubuntu构建SLURM控制节点
FROM ubuntu:22.04

ARG VERSION=v0.3.6-dev
LABEL maintainer="AI Infrastructure Team"
LABEL version="${VERSION}"
LABEL description="AI Infrastructure Matrix SLURM Master Service"

# 避免交互式安装提示
ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker
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
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "# 官方备用源" >> /etc/apt/sources.list && \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "配置AMD64阿里云镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "# 官方备用源" >> /etc/apt/sources.list && \
        echo "deb http://archive.ubuntu.com/ubuntu jammy main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://archive.ubuntu.com/ubuntu jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://archive.ubuntu.com/ubuntu jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://archive.ubuntu.com/ubuntu jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    fi; \
    # APT 重试与超时配置
    echo 'Acquire::Retries "5"; Acquire::http::Timeout "30"; Acquire::https::Timeout "30";' > /etc/apt/apt.conf.d/99retries; \
    # 更新包列表（带重试机制）
    apt-get update || (sleep 5 && apt-get update) || (sleep 10 && apt-get update)

# 安装基础依赖（ca-certificates和curl已在上面安装）
RUN set -eux; \
    # Refresh index right before install to avoid stale caches across layers
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=3 update && break || (echo "apt-get update failed (attempt $i), retrying..." && sleep 5); \
    done; \
    # Install with --fix-missing and retries to improve robustness on flaky mirrors
    apt-get -o Acquire::Retries=3 install -y --no-install-recommends --fix-missing \
        systemd \
        systemd-sysv \
        # networking and diagnostics \
        curl \
        wget \
        telnet \
        openssh-client \
        mtr-tiny \
        netcat-openbsd \
        # utilities \
        vim \
        tree \
        procps \
        gettext-base \
        # slurm prerequisites \
        munge \
        libmunge2 \
        postgresql-client \
        default-mysql-client && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
            
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
    /srv/shared

# 尝试从AppHub安装SLURM 25.05.3版本包（优先级：AppHub > 系统仓库）
ARG APPHUB_URL=http://apphub:80
RUN set -eux; \
    echo "🔍 尝试从AppHub安装SLURM包..."; \
    # 添加AppHub源
    echo "deb [trusted=yes] ${APPHUB_URL}/pkgs/slurm-deb ./" > /etc/apt/sources.list.d/ai-infra-slurm.list; \
    echo "📋 AppHub源配置:"; \
    cat /etc/apt/sources.list.d/ai-infra-slurm.list; \
    \
    # 测试AppHub连接
    echo "🌐 测试AppHub连接..."; \
    if timeout 10 wget -q --spider ${APPHUB_URL}/pkgs/slurm-deb/Packages; then \
        echo "✅ AppHub连接正常"; \
        # 更新包列表并安装
        if timeout 60 apt-get update && \
           apt-get install -y --no-install-recommends \
               slurm-smd \
               slurm-smd-client \
               slurm-smd-slurmctld \
               slurm-smd-slurmdbd; then \
            echo "✅ 成功从AppHub安装SLURM 25.05.3包"; \
            SLURM_INSTALLED=true; \
            SLURM_SOURCE="AppHub"; \
        else \
            echo "❌ AppHub包安装失败，尝试系统仓库..."; \
            rm -f /etc/apt/sources.list.d/ai-infra-slurm.list; \
            apt-get update; \
            if apt-get install -y slurm-wlm slurm-wlm-basic-plugins slurmctld slurmdbd slurm-client; then \
                echo "✅ 成功从系统仓库安装SLURM"; \
                SLURM_INSTALLED=true; \
                SLURM_SOURCE="SystemRepo"; \
            else \
                echo "❌ 系统仓库安装也失败"; \
                SLURM_INSTALLED=false; \
                SLURM_SOURCE="None"; \
            fi; \
        fi; \
    else \
        echo "❌ AppHub连接失败，直接使用系统仓库"; \
        rm -f /etc/apt/sources.list.d/ai-infra-slurm.list; \
        apt-get update; \
        if apt-get install -y slurm-wlm slurm-wlm-basic-plugins slurmctld slurmdbd slurm-client; then \
            echo "✅ 成功从系统仓库安装SLURM"; \
            SLURM_INSTALLED=true; \
            SLURM_SOURCE="SystemRepo"; \
        else \
            echo "❌ SLURM安装完全失败"; \
            SLURM_INSTALLED=false; \
            SLURM_SOURCE="None"; \
        fi; \
    fi; \
    \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    \
    # 创建标记文件和路径检查
    if [ "$SLURM_INSTALLED" = "true" ]; then \
        touch /opt/slurm-installed; \
        echo "$SLURM_SOURCE" > /opt/slurm-source; \
        # 动态检查实际的二进制文件位置
        echo "🔍 检查SLURM二进制文件位置..."; \
        SLURMCTLD_PATH=$(which slurmctld 2>/dev/null || find /usr -name "slurmctld" -type f -executable 2>/dev/null | head -1 || echo ""); \
        SLURMDBD_PATH=$(which slurmdbd 2>/dev/null || find /usr -name "slurmdbd" -type f -executable 2>/dev/null | head -1 || echo ""); \
        \
        if [ -n "$SLURMCTLD_PATH" ] && [ -x "$SLURMCTLD_PATH" ]; then \
            echo "$SLURMCTLD_PATH" > /opt/slurmctld-path; \
            echo "✅ slurmctld: $SLURMCTLD_PATH"; \
        else \
            echo "/usr/sbin/slurmctld" > /opt/slurmctld-path; \
            echo "⚠️  slurmctld路径未找到，使用默认路径"; \
        fi; \
        \
        if [ -n "$SLURMDBD_PATH" ] && [ -x "$SLURMDBD_PATH" ]; then \
            echo "$SLURMDBD_PATH" > /opt/slurmdbd-path; \
            echo "✅ slurmdbd: $SLURMDBD_PATH"; \
        else \
            echo "/usr/sbin/slurmdbd" > /opt/slurmdbd-path; \
            echo "⚠️  slurmdbd路径未找到，使用默认路径"; \
        fi; \
        \
        echo "📦 SLURM安装摘要:"; \
        echo "  来源: $SLURM_SOURCE"; \
        echo "  slurmctld: $(cat /opt/slurmctld-path)"; \
        echo "  slurmdbd: $(cat /opt/slurmdbd-path)"; \
    else \
        echo "SLURM未安装，将运行演示模式" > /opt/slurm-demo-mode; \
        echo "❌ 容器将以演示模式运行"; \
    fi

# 配置用户组和权限（在SLURM安装后）
RUN usermod -a -G munge slurm && \
    chown -R slurm:slurm /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm /etc/slurm && \
    chown -R munge:munge /etc/munge /var/lib/munge /var/log/munge /var/run/munge && \
    chmod 755 /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm && \
    chmod 700 /etc/munge /var/lib/munge && \
    chmod 755 /var/log/munge /var/run/munge && \
    chmod 640 /etc/munge/munge.key || true

# 复制配置与systemd脚本
COPY config/ /etc/slurm-templates/
COPY entrypoint.sh /usr/local/bin/slurm-bootstrap.sh
COPY systemd-entrypoint.sh /usr/local/bin/systemd-entrypoint.sh
COPY healthcheck.sh /usr/local/bin/healthcheck.sh
COPY systemd/ /etc/systemd/system/
RUN chmod +x /usr/local/bin/slurm-bootstrap.sh /usr/local/bin/systemd-entrypoint.sh /usr/local/bin/healthcheck.sh && \
    ln -sf /etc/systemd/system/slurm-bootstrap.service /etc/systemd/system/multi-user.target.wants/slurm-bootstrap.service && \
    ln -sf /etc/systemd/system/slurmctld.service /etc/systemd/system/multi-user.target.wants/slurmctld.service && \
    ln -sf /etc/systemd/system/slurmdbd.service /etc/systemd/system/multi-user.target.wants/slurmdbd.service && \
    ln -sf /lib/systemd/system/munge.service /etc/systemd/system/multi-user.target.wants/munge.service

# 暴露端口
EXPOSE 6817 6818

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=90s --retries=3 \
    CMD /usr/local/bin/healthcheck.sh

# 设置工作目录
WORKDIR /etc/slurm

# 启动脚本
STOPSIGNAL SIGRTMIN+3

VOLUME ["/sys/fs/cgroup"]

ENTRYPOINT ["/usr/local/bin/systemd-entrypoint.sh"]
CMD ["/sbin/init"]