ARG UBUNTU_VERSION=22.04
FROM ubuntu:${UBUNTU_VERSION}

# Build arguments for versions
ARG SALTSTACK_VERSION=v3007.8
ARG PIP_VERSION=24.2
ARG PIP_INDEX_URL
ARG APT_MIRROR=mirrors.aliyun.com
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}
ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker

# 配置APT镜像源（阿里云）
RUN set -eux; \
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        sed -i "s|http://archive.ubuntu.com/ubuntu/|http://${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|http://security.ubuntu.com/ubuntu/|http://${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|http://ports.ubuntu.com/ubuntu-ports/|http://${APT_MIRROR}/ubuntu-ports/|g" /etc/apt/sources.list; \
    else \
        sed -i 's|archive.ubuntu.com|mirrors.aliyun.com|g; s|security.ubuntu.com|mirrors.aliyun.com|g; s|ports.ubuntu.com|mirrors.aliyun.com|g' /etc/apt/sources.list; \
    fi; \
    apt-get update

# 安装系统依赖、Python环境和编译工具
RUN set -eux; \
    apt-get install -y --no-install-recommends \
        systemd \
        systemd-sysv \
        dbus \
        python3 \
        python3-pip \
        python3-dev \
        python3-venv \
        build-essential \
        libffi-dev \
        libssl-dev \
        libldap2-dev \
        libsasl2-dev \
        curl \
        wget \
        netcat-openbsd \
        openssh-client \
        ca-certificates \
        locales \
        jq \
        vim-tiny \
        git \
        tar \
        gzip \
        tini \
        lsof \
        tzdata \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 配置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 升级pip，解决旧版解析器的依赖冲突问题
RUN python3 -m pip install --no-cache-dir --upgrade "pip==${PIP_VERSION}"

# 配置本地化
RUN locale-gen zh_CN.UTF-8 en_US.UTF-8
ENV LANG=zh_CN.UTF-8
ENV LC_ALL=zh_CN.UTF-8

# 复制 third_party 目录以支持离线构建
COPY third_party/ /third_party/

# 配置PyPI镜像源与Python警告控制（屏蔽冗余的上游弃用告警）
ENV PIP_INDEX_URL="${PIP_INDEX_URL:-https://mirrors.aliyun.com/pypi/simple/}" \
    PIP_EXTRA_INDEX_URL="${PIP_INDEX_URL:-https://mirrors.aliyun.com/pypi/simple/}" \
    PIP_TRUSTED_HOST="mirrors.aliyun.com" \
    PIP_TIMEOUT=60 \
    PYTHONWARNINGS="ignore::DeprecationWarning"

# 安装SaltStack及其Python依赖包
RUN pip3 install --no-cache-dir \
        "setuptools<81" \
        packaging \
        wheel \
        six \
        pyyaml \
        jinja2 \
        markupsafe \
        msgpack \
        tornado \
        psutil \
        requests \
        cryptography \
        cffi \
        pyzmq==25.0.2 \
        pycryptodomex \
    && pip3 install --no-cache-dir --no-deps "salt==${SALTSTACK_VERSION}" \
    && pip3 install --no-cache-dir \
        cherrypy \
        python-ldap \
        pyopenssl \
        setproctitle \
        distro \
        looseversion \
        backports.ssl-match-hostname

# 创建必要目录
RUN mkdir -p /etc/salt/master.d \
    && mkdir -p /etc/salt/minion.d \
    && mkdir -p /etc/salt/pki/master \
    && mkdir -p /etc/salt/pki/minion \
    && mkdir -p /srv/salt \
    && mkdir -p /srv/pillar \
    && mkdir -p /var/cache/salt/master \
    && mkdir -p /var/cache/salt/minion \
    && mkdir -p /var/log/salt \
    && mkdir -p /var/run/salt

# 创建saltapi用户用于API认证
RUN useradd -m -s /bin/bash saltapi

# 复制配置文件和脚本
COPY salt-master.conf /etc/salt/master.d/master.conf
COPY salt-api.conf /etc/salt/master.d/api.conf
COPY salt-minion.conf /etc/salt/minion.d/minion.conf
COPY salt-users.conf /etc/salt/master.d/users.conf
COPY salt-states/ /srv/salt/
COPY salt-pillar/ /srv/pillar/
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 4505 4506 8002

# 增强的健康检查 - 检查Salt Master和API端口
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD nc -z 127.0.0.1 4505 && nc -z 127.0.0.1 4506 && nc -z 127.0.0.1 8002 || exit 1

STOPSIGNAL SIGRTMIN+3
VOLUME ["/sys/fs/cgroup"]

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["start-services"]

LABEL maintainer="AI Infrastructure Team" \
      version="${APP_VERSION}" \
      description="AI基础设施矩阵 - SaltStack管理服务" \
      features="SaltStack,SSO,API,Management"