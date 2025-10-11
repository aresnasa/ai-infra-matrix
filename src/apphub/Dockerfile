# =============================================================================
# Stage 1: Build SLURM deb packages
# =============================================================================
FROM ubuntu:22.04 AS slurm-builder

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# Accept optional tarball path relative to build context root
ARG SLURM_TARBALL_PATH=slurm-25.05.3.tar.bz2

# 配置APT镜像源和安装构建依赖（分步骤避免网络问题）
RUN set -eux; \
    # 备份原始sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    # 根据架构直接配置阿里云镜像源
    if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        echo "配置ARM64架构的阿里云HTTP镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "配置AMD64架构的阿里云HTTP镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    fi; \
    # 更新包列表并安装基础工具（带重试机制）
    apt-get update || (sleep 5 && apt-get update) || (sleep 10 && apt-get update); \
    apt-get install -y ca-certificates curl tzdata lsof; \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Install build prerequisites (分组安装减少失败风险)
RUN apt-get install -y --no-install-recommends \
       wget git gpg \
       build-essential fakeroot devscripts equivs gdebi-core \
       pkg-config debhelper dh-autoreconf \
    && rm -rf /var/lib/apt/lists/*

# Install development libraries (单独安装避免依赖冲突)
RUN apt-get update && apt-get install -y --no-install-recommends \
       libmunge-dev libmariadb-dev libpam0g-dev libcgroup-dev libhwloc-dev \
    && rm -rf /var/lib/apt/lists/*

# Add a non-root builder user to satisfy debuild
RUN useradd -m -u 1000 builder
USER builder
WORKDIR /home/builder/build

# Copy and extract the Slurm source tarball
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# Extract and enter source dir
RUN set -eux; \
    tar -xaf $(basename ${SLURM_TARBALL_PATH}); \
    srcdir=$(basename ${SLURM_TARBALL_PATH} .tar.bz2); \
    echo "SRC=${srcdir}" > /home/builder/build/.srcdir

# Install build-deps as root, then build .deb packages as unprivileged user
USER root
RUN set -eux; \
    srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
    cd "/home/builder/build/${srcdir}"; \
    apt-get update; \
    mk-build-deps -i debian/control -t 'apt-get -y --no-install-recommends' --remove

USER builder
RUN set -eux; \
    srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
    cd "/home/builder/build/${srcdir}"; \
    debuild -b -uc -us

# Collect artifacts into /out (root stage)
USER root
RUN mkdir -p /out \
    && chown -R root:root /home/builder

# Move all debuild outputs to /out
RUN set -eux; \
    find /home/builder/build -maxdepth 1 -type f -name '*.deb' -exec mv {} /out/ \; || true; \
    find /home/builder/build -maxdepth 1 -type f -name '*.ddeb' -exec mv {} /out/ \; || true; \
    find /home/builder/build -maxdepth 1 -type f -name '*.build*' -exec mv {} /out/ \; || true; \
    find /home/builder/build -maxdepth 1 -type f -name '*.changes' -exec mv {} /out/ \; || true; \
    # Verify at least one .deb file was produced
    deb_count=$(find /out -name '*.deb' -type f | wc -l); \
    if [ "$deb_count" -eq 0 ]; then \
        echo "ERROR: No .deb packages were built!"; \
        echo "Build artifacts in /home/builder:"; \
        ls -la /home/builder/ || true; \
        echo "Build artifacts in /home/builder/build:"; \
        ls -la /home/builder/build/ || true; \
        exit 1; \
    fi; \
    echo "Successfully built $deb_count .deb package(s)"

# =============================================================================
# Stage 2: AppHub - HTTP Server with Package Management & Development Tools
# =============================================================================
FROM nginx:stable

ENV DEBIAN_FRONTEND=noninteractive

# 配置APT镜像源并安装基础工具（使用阿里云镜像源）
RUN set -eux; \
    # 检查并创建sources.list（nginx镜像可能没有标准配置）
    if [ ! -f /etc/apt/sources.list ]; then \
        echo "创建默认sources.list..."; \
        echo "deb http://deb.debian.org/debian bookworm main" > /etc/apt/sources.list; \
        echo "deb http://deb.debian.org/debian bookworm-updates main" >> /etc/apt/sources.list; \
        echo "deb http://security.debian.org/debian-security bookworm-security main" >> /etc/apt/sources.list; \
    fi; \
    # 备份原始配置
    cp /etc/apt/sources.list /etc/apt/sources.list.bak || true; \
    # 更新包列表并安装基础工具
    apt-get update && apt-get install -y ca-certificates curl; \
    # 直接配置阿里云镜像源
    echo "配置阿里云HTTP镜像源..."; \
    echo "deb http://mirrors.aliyun.com/debian/ bookworm main" > /etc/apt/sources.list; \
    echo "deb http://mirrors.aliyun.com/debian/ bookworm-updates main" >> /etc/apt/sources.list; \
    echo "deb http://mirrors.aliyun.com/debian-security/ bookworm-security main" >> /etc/apt/sources.list; \
    apt-get update || (sleep 5 && apt-get update) || (sleep 10 && apt-get update)

# Install tools for repo management and general development
RUN apt-get install -y \
    # Package management tools
    apt-utils dpkg-dev createrepo-c \
    # Development tools (for testing and debugging)
    build-essential \
    git \
    vim \
    wget \
    curl \
    net-tools \
    iputils-ping \
    procps \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Create directories for packages
RUN mkdir -p \
    /usr/share/nginx/html/deb \
    /usr/share/nginx/html/rpm \
    /usr/share/nginx/html/pkgs/slurm-deb

# Copy SLURM deb packages from builder stage
COPY --from=slurm-builder /out/*.deb /usr/share/nginx/html/pkgs/slurm-deb/

# Generate package indexes
RUN set -eux; \
    # 统计SLURM deb包数量
    slurm_deb_count=$(ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l); \
    echo "Copied ${slurm_deb_count} SLURM deb packages"; \
    # 生成deb包索引（如果有deb包）
    if [ -d /usr/share/nginx/html/deb ] && [ "$(ls -A /usr/share/nginx/html/deb 2>/dev/null)" ]; then \
        cd /usr/share/nginx/html/deb && dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz; \
        echo "Generated deb package index"; \
    fi; \
    # 生成SLURM deb包索引
    if [ -d /usr/share/nginx/html/pkgs/slurm-deb ] && [ "$(ls -A /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null)" ]; then \
        cd /usr/share/nginx/html/pkgs/slurm-deb && dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz; \
        echo "Generated SLURM deb package index"; \
    fi; \
    # 生成rpm包索引（如果有rpm包）
    if [ -d /usr/share/nginx/html/rpm ] && [ "$(ls -A /usr/share/nginx/html/rpm 2>/dev/null)" ]; then \
        createrepo_c /usr/share/nginx/html/rpm; \
        echo "Generated rpm package index"; \
    fi

# Expose port
EXPOSE 80

# Entrypoint to regenerate indexes if needed
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]