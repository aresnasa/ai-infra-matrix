FROM alpine:3.22

ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

# 配置Alpine镜像源使用阿里云镜像加速
RUN echo "https://mirrors.aliyun.com/alpine/v3.22/main" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v3.22/community" >> /etc/apk/repositories && \
    echo "https://dl-cdn.alpinelinux.org/alpine/v3.22/main" >> /etc/apk/repositories && \
    echo "https://dl-cdn.alpinelinux.org/alpine/v3.22/community" >> /etc/apk/repositories

# 安装依赖
RUN apk update && apk add --no-cache \
    python3 \
    py3-pip \
    py3-cryptography \
    py3-openssl \
    py3-requests \
    py3-yaml \
    py3-ldap3 \
    openssh \
    supervisor \
    curl \
    wget \
    shadow \
    bash \
    gcc \
    python3-dev \
    musl-dev \
    libffi-dev \
    openssl-dev \
    openldap-dev \
    cyrus-sasl-dev

# 配置pip使用阿里云镜像源并安装SaltStack及其所有依赖
RUN pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip3 config set global.trusted-host mirrors.aliyun.com && \
    pip3 install --no-cache-dir --break-system-packages \
        salt==3007.6 \
        tornado \
        jinja2 \
        msgpack \
        pycrypto \
        pyyaml \
        markupsafe \
        requests \
        looseversion \
        distro \
        psutil \
        setproctitle \
        CherryPy==18.8.0 \
        python-ldap \
        pyopenssl

# 创建Salt用户（Alpine方式）
RUN adduser -D -s /bin/bash salt \
    && echo "salt:salt" | chpasswd \
    && adduser salt wheel

# 创建必要目录
RUN mkdir -p /etc/salt/master.d \
    && mkdir -p /etc/salt/minion.d \
    && mkdir -p /etc/salt/pki/master \
    && mkdir -p /etc/salt/pki/minion \
    && mkdir -p /srv/salt \
    && mkdir -p /srv/pillar \
    && mkdir -p /var/cache/salt \
    && mkdir -p /var/log/salt \
    && mkdir -p /var/run/salt

# 复制配置文件和脚本
COPY salt-master.conf /etc/salt/master.d/master.conf
COPY salt-api.conf /etc/salt/master.d/api.conf
COPY salt-minion.conf /etc/salt/minion.d/minion.conf
COPY supervisord.conf /etc/supervisord.conf
COPY supervisor.conf /etc/supervisor/conf.d/saltstack.conf
COPY salt-states/ /srv/salt/
COPY salt-pillar/ /srv/pillar/
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# 设置权限
RUN chown -R salt:salt /etc/salt \
    && chown -R salt:salt /srv/salt \
    && chown -R salt:salt /srv/pillar \
    && chown -R salt:salt /var/cache/salt \
    && chown -R salt:salt /var/log/salt \
    && chown -R salt:salt /var/run/salt

EXPOSE 4505 4506 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/stats || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["supervisord", "-c", "/etc/supervisord.conf"]

LABEL maintainer="AI Infrastructure Team" \
      version="${APP_VERSION}" \
      description="AI基础设施矩阵 - SaltStack管理服务" \
      features="SaltStack,SSO,API,Management"