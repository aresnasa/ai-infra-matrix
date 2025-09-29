FROM python:3.13-alpine

ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像（多镜像源智能回退配置）
RUN set -eux; \
    # 备份原始repositories文件
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 获取Alpine版本
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2); \
    echo "Detected Alpine version: ${ALPINE_VERSION}"; \
    # 直接配置阿里云镜像源
    echo "配置阿里云镜像源..."; \
    echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
    apk update --no-cache || (sleep 5 && apk update --no-cache) || (sleep 10 && apk update --no-cache)

# 安装基础依赖和编译工具
RUN apk add --no-cache \
        bash \
        curl \
        openssh-client \
        netcat-openbsd \
        gcc \
        python3-dev \
        musl-dev \
        libffi-dev \
        openssl-dev \
        build-base \
        openldap-dev \
        cyrus-sasl-dev \
        linux-headers

# 配置PyPI镜像源
ENV PIP_INDEX_URL="https://mirrors.aliyun.com/pypi/simple/" \
    PIP_EXTRA_INDEX_URL="https://mirrors.aliyun.com/pypi/simple/" \
    PIP_TRUSTED_HOST="mirrors.aliyun.com" \
    PIP_TIMEOUT=60

# 安装SaltStack及其Python依赖包
RUN pip install --no-cache-dir \
        packaging \
        wheel \
        setuptools \
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
        pyzmq \
        && \
    pip install --no-cache-dir \
        salt \
        cherrypy \
        python-ldap \
        pyopenssl \
        setproctitle \
        distro \
        looseversion

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
RUN adduser -D -s /bin/sh saltapi

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

ENTRYPOINT ["/docker-entrypoint.sh"]
# 默认启动Salt Master和API（由entrypoint脚本管理）
CMD ["salt-master", "--log-level=info"]

LABEL maintainer="AI Infrastructure Team" \
      version="${APP_VERSION}" \
      description="AI基础设施矩阵 - SaltStack管理服务" \
      features="SaltStack,SSO,API,Management"