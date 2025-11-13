# Ubuntu 22.04 SSH测试容器 - 用于SaltStack和SLURM客户端安装测试
FROM ubuntu:22.04

ARG VERSION="dev"
ENV APP_VERSION=${VERSION}
ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker
ENV TZ=Asia/Shanghai

# 使用阿里云源加速下载
RUN sed -i 's|archive.ubuntu.com|mirrors.aliyun.com|g; s|security.ubuntu.com|mirrors.aliyun.com|g; s|ports.ubuntu.com|mirrors.aliyun.com|g' /etc/apt/sources.list

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

# 创建测试用户
RUN useradd -m -s /bin/bash testuser
RUN usermod -aG sudo testuser

# 允许sudo免密码
RUN echo 'testuser ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# 配置pip使用阿里云源
RUN pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip3 config set global.trusted-host mirrors.aliyun.com

# 创建工作目录
RUN mkdir -p /opt/saltstack /opt/slurm
WORKDIR /root

# 复制启动脚本
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
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
