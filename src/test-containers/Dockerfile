# Ubuntu 22.04 SSH测试容器 - 用于SaltStack和SLURM客户端安装测试
FROM ubuntu:22.04

ARG VERSION="dev"
ARG APT_MIRROR
ARG PIP_INDEX_URL
ENV APP_VERSION=${VERSION}
ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker
ENV TZ=Asia/Shanghai

# 使用阿里云源加速下载
RUN if [ -n "${APT_MIRROR:-}" ]; then \
        sed -i "s|archive.ubuntu.com|${APT_MIRROR}|g; s|security.ubuntu.com|${APT_MIRROR}|g; s|ports.ubuntu.com|${APT_MIRROR}|g" /etc/apt/sources.list; \
    else \
        sed -i 's|archive.ubuntu.com|mirrors.aliyun.com|g; s|security.ubuntu.com|mirrors.aliyun.com|g; s|ports.ubuntu.com|mirrors.aliyun.com|g' /etc/apt/sources.list; \
    fi

# 安装SSH服务器和基本工具
RUN apt-get update && apt-get install -y \
    systemd \
    systemd-sysv \
    openssh-server \
    sudo \
    curl \
    wget \
    python3 \
    python3-pip \
    dnsutils \
    iputils-ping \
    net-tools \
    lsof \
    tzdata \
    ca-certificates \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# 配置时区
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建SSH运行目录
RUN mkdir -p /var/run/sshd /run/sshd

# 配置SSH（使用密码认证，便于测试）
RUN sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config

# 生成SSH主机密钥
RUN ssh-keygen -A

# 预先创建 SLURM 相关用户（使用固定 UID/GID 确保一致性）
# 这样在后续安装 SLURM 时不会自动分配不同的 UID/GID
# munge: UID=998, GID=998 (SLURM 认证服务)
# slurm: UID=999, GID=999 (SLURM 主用户)
RUN groupadd -g 998 munge && useradd -u 998 -g munge -d /var/lib/munge -s /sbin/nologin munge && \
    groupadd -g 999 slurm && useradd -u 999 -g slurm -d /var/spool/slurm -s /sbin/nologin slurm

# 创建测试用户
RUN useradd -m -s /bin/bash testuser
RUN usermod -aG sudo testuser

# 允许sudo免密码
RUN echo 'testuser ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# 配置pip使用阿里云源
RUN if [ -n "${PIP_INDEX_URL:-}" ]; then \
        pip3 config set global.index-url ${PIP_INDEX_URL} && \
        pip3 config set global.trusted-host $(echo ${PIP_INDEX_URL} | awk -F/ '{print $3}'); \
    else \
        pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
        pip3 config set global.trusted-host mirrors.aliyun.com; \
    fi

# 创建工作目录
RUN mkdir -p /opt/saltstack /opt/slurm
WORKDIR /root

# 复制SSH密钥（用于连接slurm-master）
RUN mkdir -p /root/.ssh && chmod 700 /root/.ssh
COPY ssh-key/id_rsa /root/.ssh/id_rsa
RUN chmod 600 /root/.ssh/id_rsa && \
    ssh-keyscan slurm-master > /root/.ssh/known_hosts 2>/dev/null || true

# 复制启动脚本
COPY src/test-containers/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# 暴露SSH端口
EXPOSE 22

# 预启用systemd管理的SSH服务
RUN systemctl enable ssh

# 添加标签
LABEL maintainer="AI Infrastructure Team"
LABEL version="${VERSION}"
LABEL description="AI Infrastructure Matrix Test Container - Ubuntu 22.04"
LABEL os="ubuntu"
LABEL os.version="22.04"

STOPSIGNAL SIGRTMIN+3

VOLUME [ "/sys/fs/cgroup" ]

# 使用entrypoint脚本启动（会设置密码并启动systemd）
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
