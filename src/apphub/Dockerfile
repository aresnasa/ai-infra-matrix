# =============================================================================
# Stage 1: Build SLURM deb packages (Ubuntu 22.04)
# =============================================================================
FROM ubuntu:22.04 AS deb-builder

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# SLURM version configuration - update these when upgrading
# 更新 SLURM 版本时只需修改这两个变量
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2

# Accept optional tarball path relative to build context root
# 默认在当前目录查找 tarball
ARG SLURM_TARBALL_PATH=${SLURM_TARBALL_NAME}

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

# Copy SLURM source tarball (use wildcard to make it optional-ish)
# Docker will fail if no matching file, but we handle it gracefully in build script
ARG SLURM_TARBALL_PATH
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# Extract SLURM source
# 解压 tarball 并记录源码目录名称
RUN set -eux; \
    if ls slurm-*.tar.bz2 >/dev/null 2>&1; then \
        tarball=$(ls slurm-*.tar.bz2 | head -1); \
        echo "✓ Found SLURM tarball: ${tarball}"; \
        tar -xaf "${tarball}"; \
        srcdir=$(basename "${tarball}" .tar.bz2); \
        echo "SRC=${srcdir}" > /home/builder/build/.srcdir; \
        echo "✓ SLURM source extracted: ${srcdir}"; \
        echo "  Version: $(echo ${srcdir} | grep -oP '\\d+\\.\\d+\\.\\d+' || echo 'unknown')"; \
    else \
        echo "SKIP_SLURM_BUILD=1" > /home/builder/build/.srcdir; \
        echo "⚠️  No SLURM tarball found - will skip SLURM package build"; \
    fi

# Install build-deps as root, then build .deb packages as unprivileged user (skip if no SLURM source)
USER root
RUN set -eux; \
    if grep -q "SKIP_SLURM_BUILD=1" /home/builder/build/.srcdir; then \
        echo "⚠️  Skipping SLURM build - no source tarball available"; \
        mkdir -p /out; \
        touch /out/.skip_slurm; \
    else \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        apt-get update; \
        mk-build-deps -i debian/control -t 'apt-get -y --no-install-recommends' --remove; \
    fi

USER builder
RUN set -eux; \
    if grep -q "SKIP_SLURM_BUILD=1" /home/builder/build/.srcdir; then \
        echo "⚠️  Skipping SLURM package build"; \
    else \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        dpkg-buildpackage -b -uc; \
    fi

# Download SaltStack packages for Ubuntu 22.04
USER root
RUN set -eux; \
    mkdir -p /saltstack-deb; \
    cd /saltstack-deb; \
    echo "📦 Downloading SaltStack deb packages for Ubuntu 22.04..."; \
    # SaltStack repository URL for Ubuntu 22.04 (Jammy)
    SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/ubuntu/22.04/amd64/latest"; \
    # Download key packages
    wget -nv "${SALTSTACK_REPO}/salt-master_3007.1-1_amd64.deb" || echo "⚠️  Failed to download salt-master"; \
    wget -nv "${SALTSTACK_REPO}/salt-minion_3007.1-1_amd64.deb" || echo "⚠️  Failed to download salt-minion"; \
    wget -nv "${SALTSTACK_REPO}/salt-common_3007.1-1_amd64.deb" || echo "⚠️  Failed to download salt-common"; \
    wget -nv "${SALTSTACK_REPO}/salt-api_3007.1-1_amd64.deb" || echo "⚠️  Failed to download salt-api"; \
    wget -nv "${SALTSTACK_REPO}/salt-ssh_3007.1-1_amd64.deb" || echo "⚠️  Failed to download salt-ssh"; \
    wget -nv "${SALTSTACK_REPO}/salt-syndic_3007.1-1_amd64.deb" || echo "⚠️  Failed to download salt-syndic"; \
    salt_count=$(ls *.deb 2>/dev/null | wc -l || echo 0); \
    if [ "$salt_count" -gt 0 ]; then \
        echo "✓ Downloaded ${salt_count} SaltStack deb packages"; \
    else \
        echo "⚠️  No SaltStack packages downloaded"; \
    fi

# Collect artifacts into /out (root stage)
RUN mkdir -p /out \
    && chown -R root:root /home/builder

# Move all debuild outputs to /out (skip verification if SLURM build was skipped)
RUN set -eux; \
    if [ ! -f /out/.skip_slurm ]; then \
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
        echo "✓ Successfully built $deb_count SLURM .deb package(s)"; \
    else \
        echo "⚠️  SLURM build was skipped - no packages to collect"; \
    fi; \
    # Copy SaltStack packages
    if [ -d /saltstack-deb ] && [ "$(ls -A /saltstack-deb/*.deb 2>/dev/null)" ]; then \
        cp /saltstack-deb/*.deb /out/ || true; \
        salt_count=$(ls /saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0); \
        echo "✓ Added ${salt_count} SaltStack deb packages"; \
    fi

# =============================================================================
# Stage 2: Build SLURM rpm packages (Rocky Linux 9)
# =============================================================================
FROM rockylinux:9 AS rpm-builder

ENV TZ=Asia/Shanghai

# SLURM version configuration (same as deb builder)
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2
ARG SLURM_TARBALL_PATH=${SLURM_TARBALL_NAME}

# 配置 Rocky Linux 镜像源（使用阿里云）
RUN set -eux; \
    # 备份原始配置
    cp -r /etc/yum.repos.d /etc/yum.repos.d.backup || true; \
    # 配置阿里云镜像源
    echo "配置阿里云 Rocky Linux 镜像源..."; \
    sed -e 's|^mirrorlist=|#mirrorlist=|g' \
        -e 's|^#baseurl=http://dl.rockylinux.org/$contentdir|baseurl=https://mirrors.aliyun.com/rockylinux|g' \
        -i.bak \
        /etc/yum.repos.d/rocky-*.repo || true; \
    # 清理缓存并更新
    dnf clean all; \
    dnf makecache || (sleep 5 && dnf makecache)

# Install build prerequisites
RUN dnf install -y \
    rpm-build rpmdevtools \
    gcc make wget tar bzip2 \
    pam-devel readline-devel perl-ExtUtils-MakeMaker \
    && dnf clean all

# Note: munge-devel and mariadb-devel require EPEL/PowerTools repos
# For simplified build, we'll download pre-built SaltStack RPMs instead of building SLURM from source

# Add non-root builder user
RUN useradd -m -u 1000 builder
USER builder
WORKDIR /home/builder/build

# Setup RPM build environment
RUN rpmdev-setuptree

# Copy SLURM source tarball
ARG SLURM_TARBALL_PATH
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# Note: Skipping SLURM RPM build due to missing dependencies in Rocky Linux 9 base repos
# SLURM RPM build requires munge-devel, mariadb-devel from EPEL/PowerTools
# For production, consider enabling these repos or using pre-built SLURM RPMs
RUN set -eux; \
    mkdir -p /home/builder/rpms; \
    touch /home/builder/rpms/.skip_slurm; \
    echo "⚠️  SLURM RPM build skipped (requires EPEL/PowerTools repos for dependencies)"

# Download SaltStack packages for Rocky Linux 9
USER root
RUN set -eux; \
    mkdir -p /saltstack-rpm; \
    cd /saltstack-rpm; \
    echo "📦 Downloading SaltStack rpm packages for Rocky Linux 9..."; \
    # SaltStack repository URL for RHEL 9
    SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/redhat/9/x86_64/latest"; \
    # Download key packages
    wget -nv "${SALTSTACK_REPO}/salt-master-3007.1-1.el9.x86_64.rpm" || echo "⚠️  Failed to download salt-master"; \
    wget -nv "${SALTSTACK_REPO}/salt-minion-3007.1-1.el9.x86_64.rpm" || echo "⚠️  Failed to download salt-minion"; \
    wget -nv "${SALTSTACK_REPO}/salt-3007.1-1.el9.x86_64.rpm" || echo "⚠️  Failed to download salt"; \
    wget -nv "${SALTSTACK_REPO}/salt-api-3007.1-1.el9.x86_64.rpm" || echo "⚠️  Failed to download salt-api"; \
    wget -nv "${SALTSTACK_REPO}/salt-ssh-3007.1-1.el9.x86_64.rpm" || echo "⚠️  Failed to download salt-ssh"; \
    wget -nv "${SALTSTACK_REPO}/salt-syndic-3007.1-1.el9.x86_64.rpm" || echo "⚠️  Failed to download salt-syndic"; \
    salt_count=$(ls *.rpm 2>/dev/null | wc -l || echo 0); \
    if [ "$salt_count" -gt 0 ]; then \
        echo "✓ Downloaded ${salt_count} SaltStack rpm packages"; \
    else \
        echo "⚠️  No SaltStack packages downloaded"; \
    fi

# Collect RPM artifacts
RUN set -eux; \
    mkdir -p /out; \
    if [ ! -f /home/builder/rpms/.skip_slurm ]; then \
        # Copy built RPMs to /out
        find /home/builder/rpmbuild/RPMS -type f -name '*.rpm' -exec cp {} /out/ \; 2>/dev/null || true; \
        rpm_count=$(ls /out/*.rpm 2>/dev/null | wc -l || echo 0); \
        if [ "$rpm_count" -gt 0 ]; then \
            echo "✓ Successfully built ${rpm_count} SLURM RPM package(s)"; \
        else \
            echo "⚠️  No SLURM RPM packages were built"; \
            touch /out/.skip_slurm; \
        fi; \
    else \
        echo "⚠️  SLURM RPM build was skipped"; \
        touch /out/.skip_slurm; \
    fi; \
    # Copy SaltStack packages
    if [ -d /saltstack-rpm ] && [ "$(ls -A /saltstack-rpm/*.rpm 2>/dev/null)" ]; then \
        cp /saltstack-rpm/*.rpm /out/ || true; \
        salt_count=$(ls /saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
        echo "✓ Added ${salt_count} SaltStack rpm packages"; \
    fi

# =============================================================================
# Stage 3: Build SLURM apk packages (Alpine Linux)
# =============================================================================
FROM alpine:latest AS apk-builder

ENV TZ=Asia/Shanghai

# SLURM version configuration (same as deb/rpm builders)
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2
ARG SLURM_TARBALL_PATH=${SLURM_TARBALL_NAME}

# 配置 Alpine 镜像源（多镜像回退）
RUN set -eux; \
    cp /etc/apk/repositories /etc/apk/repositories.bak || true; \
    ARCH=$(uname -m); \
    echo "Detected architecture: ${ARCH}"; \
    for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do \
        sed -i "s#://[^/]\+/alpine#://$MIR/alpine#g" /etc/apk/repositories || true; \
        apk update && break || true; \
    done; \
    apk add --no-cache ca-certificates tzdata; \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 安装构建依赖（核心依赖）
RUN apk add --no-cache \
    build-base \
    linux-headers \
    openssl-dev \
    readline-dev \
    curl \
    wget \
    perl \
    python3 \
    mariadb-dev \
    ncurses-dev \
    json-c-dev \
    yaml-dev \
    libevent-dev \
    lz4-dev \
    zlib-dev \
    bzip2-dev \
    tar

# 尝试安装可选依赖（可能不可用）
RUN apk add --no-cache http-parser-dev || echo '⚠ http-parser-dev not available (optional)'
RUN apk add --no-cache numactl-dev || echo '⚠ numactl-dev not available (optional)'
RUN apk add --no-cache hwloc-dev || echo '⚠ hwloc-dev not available (optional)'

# 创建构建用户
RUN adduser -D -u 1000 builder
USER builder
WORKDIR /home/builder/build

# 复制 SLURM 源码包
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# 解压 SLURM 源码 - 跳过 Alpine 构建
# 注意: Alpine SLURM 构建比较复杂，暂时跳过
# Backend 将使用演示数据模式或从 DEB 包中提取
RUN set -eux; \
    mkdir -p /home/builder/apk-output; \
    echo "SKIP_SLURM_BUILD=1" > /home/builder/build/.srcdir; \
    touch /home/builder/apk-output/.skip_slurm; \
    echo "⚠️  Alpine SLURM build skipped (complex dependencies)"; \
    echo "💡 Backend will use demo mode or extract binaries from deb packages"

# 安装到临时目录并打包
USER root
RUN set -eux; \
    if grep -q "SKIP_SLURM_BUILD=1" /home/builder/build/.srcdir; then \
        echo "⚠️  Skipping SLURM installation"; \
    else \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        mkdir -p /tmp/slurm-install/usr/local/slurm/bin; \
        mkdir -p /tmp/slurm-install/usr/local/slurm/lib; \
        mkdir -p /tmp/slurm-install/etc/slurm; \
        echo ">>> Installing SLURM client tools..."; \
        make install DESTDIR=/tmp/slurm-install || true; \
        # 手动复制客户端工具（确保存在）
        for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct; do \
            if [ -f "src/${cmd}/${cmd}" ]; then \
                cp -f "src/${cmd}/${cmd}" /tmp/slurm-install/usr/local/slurm/bin/; \
                echo "  ✓ Installed: ${cmd}"; \
            fi; \
        done; \
        # 复制必要的库
        cp -f src/common/.libs/libslurm.so* /tmp/slurm-install/usr/local/slurm/lib/ 2>/dev/null || true; \
        # 创建版本信息
        echo "25.05.4" > /tmp/slurm-install/usr/local/slurm/VERSION; \
    fi

# 复制安装脚本到容器
COPY --chown=root:root scripts/slurm-install.sh /tmp/slurm-install/install.sh
COPY --chown=root:root scripts/slurm-uninstall.sh /tmp/slurm-install/uninstall.sh
COPY --chown=root:root scripts/slurm-README.md /tmp/slurm-install/README.md

# 设置脚本权限并打包
RUN set -eux; \
    if [ ! -f /home/builder/apk-output/.skip_slurm ]; then \
        chmod +x /tmp/slurm-install/install.sh /tmp/slurm-install/uninstall.sh; \
        cd /tmp/slurm-install; \
        mkdir -p /home/builder/apk-output; \
        tar czf /home/builder/apk-output/slurm-client-25.05.4-alpine.tar.gz .; \
        echo "✓ SLURM Alpine package created"; \
        ls -lh /home/builder/apk-output/; \
    fi

# 收集 APK 包到 /out
RUN set -eux; \
    mkdir -p /out; \
    if [ -f /home/builder/apk-output/.skip_slurm ]; then \
        echo "⚠️  SLURM APK build was skipped"; \
        touch /out/.skip_slurm; \
    else \
        cp /home/builder/apk-output/*.tar.gz /out/ 2>/dev/null || true; \
        apk_count=$(ls /out/*.tar.gz 2>/dev/null | wc -l || echo 0); \
        if [ "$apk_count" -gt 0 ]; then \
            echo "✓ Successfully built ${apk_count} SLURM Alpine package(s)"; \
        else \
            echo "⚠️  No SLURM Alpine packages were built"; \
            touch /out/.skip_slurm; \
        fi; \
    fi

# =============================================================================
# Stage 4: AppHub - HTTP Server with Package Management & Development Tools
# =============================================================================
FROM nginx:alpine

# 配置 Alpine 镜像源并安装基础工具（使用国内镜像源）
RUN set -eux; \
    # 备份原始配置
    cp /etc/apk/repositories /etc/apk/repositories.bak || true; \
    # 检测架构
    ARCH=$(uname -m); \
    echo "Detected architecture: ${ARCH}"; \
    # 根据架构配置镜像源
    if [ "${ARCH}" = "aarch64" ]; then \
        echo "配置ARM64架构的清华Alpine镜像源..."; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.21/community" >> /etc/apk/repositories; \
    else \
        echo "配置AMD64架构的阿里云Alpine镜像源..."; \
        echo "https://mirrors.aliyun.com/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v3.21/community" >> /etc/apk/repositories; \
    fi; \
    # 更新包索引（带重试和备用源）
    apk update || { \
        echo "主镜像源失败，切换到官方镜像源..."; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories; \
        apk update; \
    } || { \
        echo "尝试使用HTTP协议..."; \
        echo "http://dl-cdn.alpinelinux.org/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "http://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories; \
        apk update; \
    }

# Install tools for repo management and general development
RUN apk add --no-cache \
    # Package management tools (dpkg for deb package index generation)
    dpkg dpkg-dev \
    # Development tools (for testing and debugging)
    build-base \
    git \
    vim \
    wget \
    curl \
    net-tools \
    iputils \
    procps \
    bash \
    ca-certificates \
    gzip \
    perl

# Copy nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Create directories for packages
RUN mkdir -p \
    /usr/share/nginx/html/deb \
    /usr/share/nginx/html/rpm \
    /usr/share/nginx/html/pkgs/slurm-deb \
    /usr/share/nginx/html/pkgs/slurm-rpm \
    /usr/share/nginx/html/pkgs/slurm-apk \
    /usr/share/nginx/html/pkgs/saltstack-deb \
    /usr/share/nginx/html/pkgs/saltstack-rpm

# Copy all deb packages from deb-builder stage (SLURM + SaltStack)
COPY --from=deb-builder /out/ /usr/share/nginx/html/pkgs/slurm-deb/

# Copy all rpm packages from rpm-builder stage (SLURM + SaltStack)
COPY --from=rpm-builder /out/ /usr/share/nginx/html/pkgs/slurm-rpm/

# Copy all apk packages from apk-builder stage (SLURM client tools)
COPY --from=apk-builder /out/ /usr/share/nginx/html/pkgs/slurm-apk/

# Separate packages into subdirectories and generate indexes
RUN set -eux; \
    echo "📦 Organizing packages..."; \
    # Separate SLURM and SaltStack deb packages
    mkdir -p /usr/share/nginx/html/pkgs/saltstack-deb; \
    if [ -d /usr/share/nginx/html/pkgs/slurm-deb ]; then \
        cd /usr/share/nginx/html/pkgs/slurm-deb; \
        # Move SaltStack packages to separate directory
        find . -name "salt-*.deb" -exec mv {} /usr/share/nginx/html/pkgs/saltstack-deb/ \; 2>/dev/null || true; \
    fi; \
    # Separate SLURM and SaltStack rpm packages
    mkdir -p /usr/share/nginx/html/pkgs/saltstack-rpm; \
    if [ -d /usr/share/nginx/html/pkgs/slurm-rpm ]; then \
        cd /usr/share/nginx/html/pkgs/slurm-rpm; \
        # Move SaltStack packages to separate directory
        find . -name "salt-*.rpm" -exec mv {} /usr/share/nginx/html/pkgs/saltstack-rpm/ \; 2>/dev/null || true; \
    fi; \
    # Count packages
    slurm_deb_count=$(ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l || echo 0); \
    slurm_rpm_count=$(ls /usr/share/nginx/html/pkgs/slurm-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
    slurm_apk_count=$(ls /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    salt_deb_count=$(ls /usr/share/nginx/html/pkgs/saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0); \
    salt_rpm_count=$(ls /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
    echo "📊 Package Summary:"; \
    echo "  - SLURM deb packages: ${slurm_deb_count}"; \
    echo "  - SLURM rpm packages: ${slurm_rpm_count}"; \
    echo "  - SLURM apk packages: ${slurm_apk_count}"; \
    echo "  - SaltStack deb packages: ${salt_deb_count}"; \
    echo "  - SaltStack rpm packages: ${salt_rpm_count}"; \
    # Generate deb package indexes
    if [ "$slurm_deb_count" -gt 0 ]; then \
        cd /usr/share/nginx/html/pkgs/slurm-deb && dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz; \
        echo "✓ Generated SLURM deb package index"; \
    fi; \
    if [ "$salt_deb_count" -gt 0 ]; then \
        cd /usr/share/nginx/html/pkgs/saltstack-deb && dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz; \
        echo "✓ Generated SaltStack deb package index"; \
    fi; \
    # General deb index (if any)
    if [ -d /usr/share/nginx/html/deb ] && [ "$(ls -A /usr/share/nginx/html/deb 2>/dev/null)" ]; then \
        cd /usr/share/nginx/html/deb && dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz; \
        echo "✓ Generated general deb package index"; \
    fi; \
    # RPM packages (available for direct download, no metadata)
    if [ "$slurm_rpm_count" -gt 0 ]; then \
        echo "✓ SLURM RPM packages available at /pkgs/slurm-rpm/"; \
    fi; \
    if [ "$salt_rpm_count" -gt 0 ]; then \
        echo "✓ SaltStack RPM packages available at /pkgs/saltstack-rpm/"; \
    fi; \
    # APK packages (create latest symlink)
    if [ "$slurm_apk_count" -gt 0 ]; then \
        cd /usr/share/nginx/html/pkgs/slurm-apk; \
        latest_apk=$(ls -t slurm-client-*.tar.gz 2>/dev/null | head -1); \
        if [ -n "$latest_apk" ]; then \
            ln -sf "$latest_apk" slurm-client-latest-alpine.tar.gz; \
            echo "✓ SLURM Alpine packages available at /pkgs/slurm-apk/"; \
            echo "  ✓ Created symlink: slurm-client-latest-alpine.tar.gz -> $latest_apk"; \
        fi; \
    fi; \
    # Note about RPM metadata
    if [ "$slurm_rpm_count" -gt 0 ] || [ "$salt_rpm_count" -gt 0 ]; then \
        echo "⚠️  Note: YUM/DNF metadata not generated (createrepo not available in Alpine)"; \
        echo "⚠️  Packages can be downloaded directly via HTTP"; \
    fi

# Expose port
EXPOSE 80

# Entrypoint to regenerate indexes if needed
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]