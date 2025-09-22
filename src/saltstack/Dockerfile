FROM python:3.13-alpine

ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 基础环境和构建工具（多镜像源智能回退配置）
RUN set -eux; \
    # 备份原始repositories文件
    cp /etc/apk/repositories /etc/apk/repositories.bak; \
    # 获取Alpine版本
    ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2); \
    echo "Detected Alpine version: ${ALPINE_VERSION}"; \
    # 尝试阿里云镜像源
    { \
        echo "尝试阿里云镜像源..."; \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用阿里云镜像源"; \
    } || { \
        echo "阿里云镜像源失败，尝试清华源..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用清华源"; \
    } || { \
        echo "清华源失败，尝试中科大源..."; \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/main" > /etc/apk/repositories && \
        echo "https://mirrors.ustc.edu.cn/alpine/v${ALPINE_VERSION}/community" >> /etc/apk/repositories && \
        apk update --no-cache && \
        echo "成功使用中科大源"; \
    } || { \
        echo "所有国内源都失败，使用官方源..."; \
        cp /etc/apk/repositories.bak /etc/apk/repositories && \
        apk update && \
        echo "使用官方源"; \
    }

# 更新包索引并安装基础依赖
RUN apk update && apk add --no-cache \
    ca-certificates \
    curl \
    openssl \
    bash \
    wget \
    gcc \
    python3-dev \
    musl-dev \
    libffi-dev \
    openssl-dev \
    build-base \
    linux-headers

# 安装网络和系统工具
RUN apk add --no-cache \
    openssh-client \
    netcat-openbsd \
    shadow

# 安装LDAP相关依赖
RUN apk add --no-cache \
    openldap-dev \
    cyrus-sasl-dev || echo "LDAP packages skipped"

# 配置pip使用中国镜像源
RUN pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip config set global.trusted-host mirrors.aliyun.com && \
    pip config set install.trusted-host mirrors.aliyun.com

# 安装SaltStack及其依赖（确保所有必需依赖）
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

EXPOSE 4505 4506 8000

# 增强的健康检查 - 检查Salt Master和API端口
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD nc -z 127.0.0.1 4505 && nc -z 127.0.0.1 4506 && nc -z 127.0.0.1 8000 || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
# 默认启动Salt Master和API（由entrypoint脚本管理）
CMD ["salt-master", "--log-level=info"]

LABEL maintainer="AI Infrastructure Team" \
      version="${APP_VERSION}" \
      description="AI基础设施矩阵 - SaltStack管理服务" \
      features="SaltStack,SSO,API,Management"