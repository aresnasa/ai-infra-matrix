# =============================================================================
# Stage 1: Build SLURM deb packages (Ubuntu 22.04)
# =============================================================================
FROM ubuntu:22.04 AS deb-builder

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# Build control flags - set to "true" to enable building specific components
# 构建控制开关 - 设置为 "true" 来启用特定组件的构建
ARG BUILD_SLURM=true
ARG BUILD_SALTSTACK=true
ARG BUILD_CATEGRAF=true
ARG BUILD_SINGULARITY=false

# SLURM version configuration - update these when upgrading
# 更新 SLURM 版本时只需修改这两个变量
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2

# SaltStack version configuration
ARG SALTSTACK_VERSION=v3007.8

# Categraf version configuration
ARG CATEGRAF_VERSION=v0.4.22

# Singularity version configuration
ARG SINGULARITY_VERSION=v4.3.4

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

# Download SaltStack packages from GitHub releases
USER root
ARG BUILD_SALTSTACK
ARG SALTSTACK_VERSION
RUN set -eux; \
    if [ "${BUILD_SALTSTACK}" = "true" ]; then \
        mkdir -p /saltstack-deb; \
        cd /saltstack-deb; \
        echo "📦 Downloading SaltStack ${SALTSTACK_VERSION} deb packages from GitHub releases..."; \
        # 检测架构
        ARCH=$(dpkg --print-architecture); \
        echo "Architecture: ${ARCH}"; \
        # 根据架构确定包后缀
        if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
            ARCH_SUFFIX="arm64"; \
        else \
            ARCH_SUFFIX="amd64"; \
        fi; \
        # GitHub releases 基础 URL
        BASE_URL="https://github.com/saltstack/salt/releases/download/${SALTSTACK_VERSION}"; \
        echo "Downloading from: ${BASE_URL}"; \
        # 下载所有主要的 deb 包
        for pkg in salt-common salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do \
            PKG_FILE="${pkg}_${SALTSTACK_VERSION#v}_${ARCH_SUFFIX}.deb"; \
            for attempt in 1 2 3; do \
                if wget --timeout=60 --tries=3 -nv "${BASE_URL}/${PKG_FILE}" 2>/dev/null; then \
                    echo "✓ Downloaded: ${PKG_FILE}"; \
                    break; \
                else \
                    echo "⚠️  Attempt ${attempt}/3 failed for ${PKG_FILE}"; \
                    sleep 2; \
                fi; \
            done || echo "✗ Failed to download ${PKG_FILE} after 3 attempts"; \
        done; \
        # 检查下载结果
        salt_count=$(ls *.deb 2>/dev/null | wc -l || echo 0); \
        if [ "$salt_count" -gt 0 ]; then \
            echo "✓ Downloaded ${salt_count} SaltStack deb packages"; \
            ls -lh *.deb; \
        else \
            echo "⚠️  No SaltStack packages downloaded - will continue without them"; \
        fi; \
    else \
        echo "⏭️  Skipping SaltStack download (BUILD_SALTSTACK=${BUILD_SALTSTACK})"; \
        mkdir -p /saltstack-deb; \
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
        ls -lh /out/salt*.deb 2>/dev/null || true; \
    fi

# =============================================================================
# Stage 2: Build SLURM rpm packages (Rocky Linux 9)
# =============================================================================
FROM rockylinux:9 AS rpm-builder

ENV TZ=Asia/Shanghai

# Build control flags
ARG BUILD_SLURM=true
ARG BUILD_SALTSTACK=true

# SLURM version configuration (same as deb builder)
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2
ARG SLURM_TARBALL_PATH=${SLURM_TARBALL_NAME}

# SaltStack version configuration (same as deb builder)
ARG SALTSTACK_VERSION=v3007.8

# 配置 Rocky Linux 镜像源（多镜像源回退策略）
RUN set -eux; \
    # 备份原始配置
    cp -r /etc/yum.repos.d /etc/yum.repos.d.backup || true; \
    # 尝试多个中国镜像源（按优先级顺序）
    MIRRORS="mirrors.aliyun.com mirrors.tuna.tsinghua.edu.cn mirrors.ustc.edu.cn mirror.sjtu.edu.cn mirrors.163.com"; \
    SUCCESS=0; \
    for MIRROR in $MIRRORS; do \
        echo "尝试配置 Rocky Linux 镜像源: ${MIRROR}..."; \
        # 禁用 mirrorlist，使用指定的 baseurl
        sed -e 's|^mirrorlist=|#mirrorlist=|g' \
            -e "s|^#baseurl=http://dl.rockylinux.org/\$contentdir|baseurl=https://${MIRROR}/rockylinux|g" \
            -i.bak \
            /etc/yum.repos.d/rocky-*.repo || true; \
        # 清理缓存
        dnf clean all >/dev/null 2>&1 || true; \
        # 尝试更新缓存
        if dnf makecache --timer 2>/dev/null || dnf makecache 2>/dev/null; then \
            echo "✓ 成功配置镜像源: ${MIRROR}"; \
            SUCCESS=1; \
            break; \
        else \
            echo "✗ 镜像源 ${MIRROR} 失败，尝试下一个..."; \
            # 恢复备份
            cp -r /etc/yum.repos.d.backup/* /etc/yum.repos.d/ 2>/dev/null || true; \
        fi; \
    done; \
    # 如果所有镜像源都失败，尝试使用原始源（HTTP）
    if [ "$SUCCESS" -eq 0 ]; then \
        echo "⚠️ 所有中国镜像源失败，尝试使用官方源（HTTP）..."; \
        sed -e 's|^mirrorlist=|#mirrorlist=|g' \
            -e 's|^#baseurl=https://dl.rockylinux.org/$contentdir|baseurl=http://dl.rockylinux.org/$contentdir|g' \
            -i.bak \
            /etc/yum.repos.d/rocky-*.repo || true; \
        dnf clean all; \
        dnf makecache --timer || dnf makecache || { \
            echo "ERROR: 无法配置任何可用的镜像源"; \
            exit 1; \
        }; \
    fi

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

# Download SaltStack packages from GitHub releases
USER root
ARG BUILD_SALTSTACK
ARG SALTSTACK_VERSION
RUN set -eux; \
    if [ "${BUILD_SALTSTACK}" = "true" ]; then \
        mkdir -p /saltstack-rpm; \
        cd /saltstack-rpm; \
        echo "📦 Downloading SaltStack ${SALTSTACK_VERSION} rpm packages from GitHub releases..."; \
        # 检测架构
        ARCH=$(uname -m); \
        echo "Architecture: ${ARCH}"; \
        # 根据架构确定包后缀
        if [ "${ARCH}" = "aarch64" ]; then \
            ARCH_SUFFIX="aarch64"; \
        else \
            ARCH_SUFFIX="x86_64"; \
        fi; \
        # GitHub releases 基础 URL
        BASE_URL="https://github.com/saltstack/salt/releases/download/${SALTSTACK_VERSION}"; \
        echo "Downloading from: ${BASE_URL}"; \
        # 下载所有主要的 rpm 包
        for pkg in salt salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do \
            PKG_FILE="${pkg}-${SALTSTACK_VERSION#v}-0.${ARCH_SUFFIX}.rpm"; \
            for attempt in 1 2 3; do \
                if wget --timeout=60 --tries=3 -nv "${BASE_URL}/${PKG_FILE}" 2>/dev/null; then \
                    echo "✓ Downloaded: ${PKG_FILE}"; \
                    break; \
                else \
                    echo "⚠️  Attempt ${attempt}/3 failed for ${PKG_FILE}"; \
                    sleep 2; \
                fi; \
            done || echo "✗ Failed to download ${PKG_FILE} after 3 attempts"; \
        done; \
        # 检查下载结果
        salt_count=$(ls *.rpm 2>/dev/null | wc -l || echo 0); \
        if [ "$salt_count" -gt 0 ]; then \
            echo "✓ Downloaded ${salt_count} SaltStack rpm packages"; \
            ls -lh *.rpm; \
        else \
            echo "⚠️  No SaltStack packages downloaded - will continue without them"; \
        fi; \
    else \
        echo "⏭️  Skipping SaltStack download (BUILD_SALTSTACK=${BUILD_SALTSTACK})"; \
        mkdir -p /saltstack-rpm; \
    fi

# Collect RPM artifacts
RUN set -eux; \
    mkdir -p /out; \
    echo "📦 Collecting RPM packages..."; \
    if [ ! -f /home/builder/rpms/.skip_slurm ]; then \
        # Copy built RPMs to /out
        find /home/builder/rpmbuild/RPMS -type f -name '*.rpm' -exec cp {} /out/ \; 2>/dev/null || true; \
        rpm_count=$(ls /out/*.rpm 2>/dev/null | wc -l || echo 0); \
        if [ "$rpm_count" -gt 0 ]; then \
            echo "✓ Successfully built ${rpm_count} SLURM RPM package(s)"; \
            ls -lh /out/*.rpm; \
        else \
            echo "⚠️  No SLURM RPM packages were built"; \
            touch /out/.skip_slurm; \
        fi; \
    else \
        echo "⚠️  SLURM RPM build was skipped"; \
        touch /out/.skip_slurm; \
    fi; \
    # Copy SaltStack packages (CRITICAL: ensure they exist before copying)
    echo "📦 Checking SaltStack packages..."; \
    if [ -d /saltstack-rpm ]; then \
        salt_rpm_count=$(ls /saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
        echo "Found ${salt_rpm_count} SaltStack rpm files in /saltstack-rpm"; \
        if [ "$salt_rpm_count" -gt 0 ]; then \
            ls -lh /saltstack-rpm/*.rpm; \
            cp /saltstack-rpm/*.rpm /out/ || { \
                echo "❌ Failed to copy SaltStack RPMs"; \
                exit 1; \
            }; \
            echo "✓ Copied ${salt_rpm_count} SaltStack rpm packages to /out"; \
            ls -lh /out/salt*.rpm 2>/dev/null || echo "⚠️  No salt*.rpm in /out"; \
        else \
            echo "⚠️  No SaltStack RPM files found in /saltstack-rpm"; \
        fi; \
    else \
        echo "❌ /saltstack-rpm directory does not exist"; \
    fi; \
    # Final verification
    echo "📊 Final /out contents:"; \
    ls -lh /out/ || echo "⚠️  /out is empty"; \
    total_rpm_count=$(ls /out/*.rpm 2>/dev/null | wc -l || echo 0); \
    echo "✓ Total RPM packages in /out: ${total_rpm_count}"

# =============================================================================
# Stage 3: Build SLURM binaries (Ubuntu 22.04 with glibc)
# Alpine uses musl libc which doesn't support cpu_set_t, so we use Ubuntu
# =============================================================================
FROM ubuntu:22.04 AS binary-builder

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# SLURM version configuration
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2
ARG SLURM_TARBALL_PATH=${SLURM_TARBALL_NAME}

# 配置 Ubuntu 镜像源（阿里云）并安装所有依赖
RUN set -eux; \
    # 检测架构
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    # 备份原始源
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 根据架构配置阿里云镜像源
    if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        echo "配置ARM64架构的阿里云镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse" > /etc/apt/sources.list; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "配置AMD64架构的阿里云镜像源..."; \
        sed -i 's|http://archive.ubuntu.com|http://mirrors.aliyun.com|g' /etc/apt/sources.list; \
        sed -i 's|http://security.ubuntu.com|http://mirrors.aliyun.com|g' /etc/apt/sources.list; \
    fi; \
    cat /etc/apt/sources.list; \
    # 更新包列表（带重试）
    apt-get update || (sleep 5 && apt-get update) || (sleep 10 && apt-get update); \
    # 安装所有依赖
    apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
        build-essential \
        wget \
        tar \
        bzip2 \
        python3 \
        python3-pip \
        libmunge-dev \
        libmariadb-dev \
        libpam0g-dev \
        libhwloc-dev \
        libjson-c-dev \
        libyaml-dev \
        libssl-dev \
        libreadline-dev; \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone; \
    rm -rf /var/lib/apt/lists/*

# 创建构建用户
RUN useradd -m -u 1000 builder
USER builder
WORKDIR /home/builder/build

# 复制 SLURM 源码包
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# 解压并编译 SLURM
RUN set -eux; \
    tarball=$(ls slurm-*.tar.bz2 | head -1); \
    echo "✓ Found SLURM tarball: ${tarball}"; \
    tar -xaf "${tarball}"; \
    srcdir=$(basename "${tarball}" .tar.bz2); \
    cd "${srcdir}"; \
    echo ">>> Configuring SLURM..."; \
    ./configure \
        --prefix=/usr/local/slurm \
        --sysconfdir=/etc/slurm \
        --disable-debug \
        --without-rpath 2>&1 | tee /tmp/configure.log || { \
        echo "❌ Configure failed! Last 50 lines of output:"; \
        tail -50 /tmp/configure.log; \
        exit 1; \
    }; \
    echo ">>> Building SLURM (this may take 5-10 minutes)..."; \
    make -j$(nproc) 2>&1 | tee /tmp/make.log || { \
        echo "❌ Make failed! Last 100 lines of output:"; \
        tail -100 /tmp/make.log; \
        exit 1; \
    }; \
    echo "✓ SLURM build completed successfully"

# 安装到临时目录并收集二进制文件
USER root
RUN set -eux; \
    cd /home/builder/build/slurm-*; \
    ARCH=$(uname -m); \
    echo "Architecture: ${ARCH}"; \
    mkdir -p /out/packages/${ARCH}; \
    mkdir -p /out/packages/${ARCH}/bin; \
    mkdir -p /out/packages/${ARCH}/lib; \
    echo ">>> Collecting SLURM binaries..."; \
    for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr; do \
        if [ -f "src/${cmd}/${cmd}" ]; then \
            cp -f "src/${cmd}/${cmd}" /out/packages/${ARCH}/bin/; \
            chmod +x /out/packages/${ARCH}/bin/${cmd}; \
            echo "  ✓ Collected: ${cmd}"; \
        elif [ -f "src/${cmd}/.libs/${cmd}" ]; then \
            cp -f "src/${cmd}/.libs/${cmd}" /out/packages/${ARCH}/bin/; \
            chmod +x /out/packages/${ARCH}/bin/${cmd}; \
            echo "  ✓ Collected: ${cmd} (from .libs)"; \
        else \
            echo "  ✗ Not found: ${cmd}"; \
        fi; \
    done; \
    echo ">>> Collecting SLURM libraries..."; \
    if [ -d "src/api/.libs" ]; then \
        find src/api/.libs -name "libslurm*.so*" -type f -exec cp {} /out/packages/${ARCH}/lib/ \; || true; \
    fi; \
    if [ -d "src/common/.libs" ]; then \
        find src/common/.libs -name "libslurm*.so*" -type f -exec cp {} /out/packages/${ARCH}/lib/ \; || true; \
    fi; \
    echo "${SLURM_VERSION}" > /out/packages/${ARCH}/VERSION; \
    echo ""; \
    echo "📦 SLURM binaries for ${ARCH}:"; \
    ls -lh /out/packages/${ARCH}/bin/ || true; \
    echo ""; \
    echo "� SLURM libraries for ${ARCH}:"; \
    ls -lh /out/packages/${ARCH}/lib/ || true

# =============================================================================
# Stage 4: Build Categraf (Multi-Architecture Go Binary)
# 泛化的应用构建阶段 - 只需复制 scripts/categraf/ 目录即可
# =============================================================================
FROM golang:1.25-alpine AS categraf-builder

# Build control flags
ARG BUILD_CATEGRAF=true

# Categraf version configuration
ARG CATEGRAF_VERSION=v0.4.22
ARG CATEGRAF_REPO=https://github.com/flashcatcloud/categraf.git

# 配置 Go 代理（中国镜像加速）
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GO111MODULE=on
ENV CGO_ENABLED=0

# 配置 Alpine 镜像源（通用配置）
RUN set -eux; \
    ARCH=$(uname -m); \
    echo "Detected architecture: ${ARCH}"; \
    if [ "${ARCH}" = "aarch64" ]; then \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.21/community" >> /etc/apk/repositories; \
    else \
        echo "https://mirrors.aliyun.com/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v3.21/community" >> /etc/apk/repositories; \
    fi; \
    apk update || { \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/main" > /etc/apk/repositories; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories; \
        apk update; \
    }

# Install build dependencies（通用依赖）
RUN apk add --no-cache git make bash tar gzip sed coreutils

# 复制通用构建脚本和应用特定脚本
COPY scripts/build-app.sh /scripts/build-app.sh
COPY scripts/categraf/ /scripts/categraf/
RUN chmod +x /scripts/build-app.sh /scripts/categraf/*.sh

# 创建输出目录
RUN mkdir -p /out

# 执行构建（使用通用构建脚本）
ARG BUILD_CATEGRAF
RUN set -eux; \
    if [ "${BUILD_CATEGRAF}" = "true" ]; then \
        CATEGRAF_VERSION=${CATEGRAF_VERSION} \
        CATEGRAF_REPO=${CATEGRAF_REPO} \
        BUILD_DIR=/build \
        OUTPUT_DIR=/out \
        /scripts/build-app.sh categraf; \
    else \
        echo "⏭️  Skipping Categraf build (BUILD_CATEGRAF=${BUILD_CATEGRAF})"; \
    fi

#================================================
# Stage 5: 最终镜像 - 使用 nginx 提供下载服务
#================================================

# =============================================================================
# Stage 5: AppHub - HTTP Server with Package Management & Development Tools
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
    /usr/share/nginx/html/pkgs/slurm-binaries \
    /usr/share/nginx/html/pkgs/saltstack-deb \
    /usr/share/nginx/html/pkgs/saltstack-rpm \
    /usr/share/nginx/html/pkgs/categraf

# Copy all deb packages from deb-builder stage (SLURM + SaltStack)
COPY --from=deb-builder /out/ /usr/share/nginx/html/pkgs/slurm-deb/

# Copy all rpm packages from rpm-builder stage (SLURM + SaltStack)
COPY --from=rpm-builder /out/ /usr/share/nginx/html/pkgs/slurm-rpm/

# Copy SLURM binaries from binary-builder stage
COPY --from=binary-builder /out/packages/ /usr/share/nginx/html/pkgs/slurm-binaries/

# Copy SLURM installation script
COPY packages/install-slurm.sh /usr/share/nginx/html/packages/install-slurm.sh
RUN chmod +x /usr/share/nginx/html/packages/install-slurm.sh

# Copy Categraf packages from categraf-builder stage
COPY --from=categraf-builder /out/ /usr/share/nginx/html/pkgs/categraf/

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
    slurm_bin_count=$(find /usr/share/nginx/html/pkgs/slurm-binaries -type f -name "s*" 2>/dev/null | wc -l || echo 0); \
    salt_deb_count=$(ls /usr/share/nginx/html/pkgs/saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0); \
    salt_rpm_count=$(ls /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
    categraf_count=$(ls /usr/share/nginx/html/pkgs/categraf/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    echo "📊 Package Summary:"; \
    echo "  - SLURM deb packages: ${slurm_deb_count}"; \
    echo "  - SLURM rpm packages: ${slurm_rpm_count}"; \
    echo "  - SLURM binaries: ${slurm_bin_count}"; \
    echo "  - SaltStack deb packages: ${salt_deb_count}"; \
    echo "  - SaltStack rpm packages: ${salt_rpm_count}"; \
    echo "  - Categraf packages: ${categraf_count}"; \
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
    # SLURM binaries (list architecture directories)
    if [ "$slurm_bin_count" -gt 0 ]; then \
        echo "✓ SLURM binaries available at /pkgs/slurm-binaries/"; \
        for arch_dir in /usr/share/nginx/html/pkgs/slurm-binaries/*; do \
            if [ -d "$arch_dir" ]; then \
                arch=$(basename "$arch_dir"); \
                bin_count=$(ls "$arch_dir/bin/"* 2>/dev/null | wc -l || echo 0); \
                echo "  ✓ ${arch}: ${bin_count} binaries"; \
            fi; \
        done; \
    fi; \
    # Categraf packages (create latest symlinks)
    if [ "$categraf_count" -gt 0 ]; then \
        cd /usr/share/nginx/html/pkgs/categraf; \
        echo "✓ Categraf packages available at /pkgs/categraf/"; \
        # Create latest symlink for amd64
        latest_amd64=$(ls -t categraf-*-linux-amd64.tar.gz 2>/dev/null | head -1); \
        if [ -n "$latest_amd64" ]; then \
            ln -sf "$latest_amd64" categraf-latest-linux-amd64.tar.gz; \
            echo "  ✓ Created symlink: categraf-latest-linux-amd64.tar.gz -> $latest_amd64"; \
        fi; \
        # Create latest symlink for arm64
        latest_arm64=$(ls -t categraf-*-linux-arm64.tar.gz 2>/dev/null | head -1); \
        if [ -n "$latest_arm64" ]; then \
            ln -sf "$latest_arm64" categraf-latest-linux-arm64.tar.gz; \
            echo "  ✓ Created symlink: categraf-latest-linux-arm64.tar.gz -> $latest_arm64"; \
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