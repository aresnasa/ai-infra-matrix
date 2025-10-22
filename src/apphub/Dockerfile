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
    # 检测架构
    ARCH=$(dpkg --print-architecture); \
    echo "Architecture: ${ARCH}"; \
    # SaltStack repository URL for Ubuntu 22.04 (Jammy)
    if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/ubuntu/22.04/arm64/latest"; \
        ARCH_SUFFIX="arm64"; \
    else \
        SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/ubuntu/22.04/amd64/latest"; \
        ARCH_SUFFIX="amd64"; \
    fi; \
    echo "Using SaltStack repo: ${SALTSTACK_REPO}"; \
    # Download key packages with retry
    for pkg in salt-master salt-minion salt-common salt-api salt-ssh salt-syndic; do \
        for attempt in 1 2 3; do \
            if wget --timeout=30 --tries=3 -nv "${SALTSTACK_REPO}/${pkg}_3007.1-1_${ARCH_SUFFIX}.deb" 2>/dev/null; then \
                echo "✓ Downloaded: ${pkg}"; \
                break; \
            else \
                echo "⚠️  Attempt ${attempt}/3 failed for ${pkg}"; \
                sleep 2; \
            fi; \
        done || echo "✗ Failed to download ${pkg} after 3 attempts"; \
    done; \
    salt_count=$(ls *.deb 2>/dev/null | wc -l || echo 0); \
    if [ "$salt_count" -gt 0 ]; then \
        echo "✓ Downloaded ${salt_count} SaltStack deb packages"; \
        ls -lh *.deb; \
    else \
        echo "⚠️  No SaltStack packages downloaded - will continue without them"; \
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

# Download SaltStack packages for Rocky Linux 9
USER root
RUN set -eux; \
    mkdir -p /saltstack-rpm; \
    cd /saltstack-rpm; \
    echo "📦 Downloading SaltStack rpm packages for Rocky Linux 9..."; \
    # 检测架构
    ARCH=$(uname -m); \
    echo "Architecture: ${ARCH}"; \
    # SaltStack repository URL for RHEL 9
    if [ "${ARCH}" = "aarch64" ]; then \
        SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/redhat/9/aarch64/latest"; \
    else \
        SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/redhat/9/x86_64/latest"; \
    fi; \
    echo "Using SaltStack repo: ${SALTSTACK_REPO}"; \
    # Download key packages with retry and better error handling
    SUCCESS=0; \
    for pkg in salt-master salt-minion salt salt-api salt-ssh salt-syndic; do \
        pkg_file="${pkg}-3007.1-1.el9.${ARCH}.rpm"; \
        echo "Downloading ${pkg_file}..."; \
        for attempt in 1 2 3; do \
            if wget --timeout=30 --tries=1 -nv "${SALTSTACK_REPO}/${pkg_file}" 2>&1; then \
                if [ -f "${pkg_file}" ]; then \
                    file_size=$(stat -f%z "${pkg_file}" 2>/dev/null || stat -c%s "${pkg_file}" 2>/dev/null || echo 0); \
                    if [ "$file_size" -gt 1000 ]; then \
                        echo "  ✓ Downloaded: ${pkg_file} (${file_size} bytes)"; \
                        SUCCESS=1; \
                        break; \
                    else \
                        echo "  ⚠️  File too small: ${file_size} bytes, retrying..."; \
                        rm -f "${pkg_file}"; \
                    fi; \
                fi; \
            fi; \
            echo "  ⚠️  Attempt ${attempt}/3 failed for ${pkg_file}"; \
            [ $attempt -lt 3 ] && sleep 2; \
        done; \
        [ ! -f "${pkg_file}" ] && echo "  ✗ Failed to download ${pkg_file} after 3 attempts"; \
    done; \
    # Verify downloads
    salt_count=$(ls *.rpm 2>/dev/null | wc -l || echo 0); \
    if [ "$salt_count" -gt 0 ]; then \
        echo "✓ Downloaded ${salt_count} SaltStack rpm packages"; \
        ls -lh *.rpm; \
    else \
        echo "⚠️  WARNING: No SaltStack packages downloaded (network or DNS issue)"; \
        echo "⚠️  saltstack-rpm directory will be empty"; \
        echo "⚠️  This is not fatal - continuing build..."; \
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

# 解压并编译 SLURM 客户端工具
RUN set -eux; \
    if ls slurm-*.tar.bz2 >/dev/null 2>&1; then \
        tarball=$(ls slurm-*.tar.bz2 | head -1); \
        echo "✓ Found SLURM tarball: ${tarball}"; \
        tar -xaf "${tarball}"; \
        srcdir=$(basename "${tarball}" .tar.bz2); \
        echo "SRC=${srcdir}" > /home/builder/build/.srcdir; \
        echo "✓ SLURM source extracted: ${srcdir}"; \
        # 配置并编译
        cd "${srcdir}"; \
        echo ">>> Configuring SLURM..."; \
        ./configure \
            --prefix=/usr/local/slurm \
            --sysconfdir=/etc/slurm \
            --disable-debug \
            --enable-pam \
            --without-rpath \
            --with-ssl=/usr || { \
            echo "⚠️  Configure failed, marking for skip"; \
            echo "SKIP_SLURM_BUILD=1" > /home/builder/build/.srcdir; \
        }; \
    else \
        echo "SKIP_SLURM_BUILD=1" > /home/builder/build/.srcdir; \
        echo "⚠️  No SLURM tarball found - will skip SLURM build"; \
    fi

# 编译 SLURM
RUN set -eux; \
    if grep -q "SKIP_SLURM_BUILD=1" /home/builder/build/.srcdir; then \
        echo "⚠️  Skipping SLURM build"; \
    else \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        echo ">>> Building SLURM client tools..."; \
        make -j$(nproc) || { \
            echo "⚠️  Full build failed, trying client tools only"; \
            for tool in sinfo squeue scontrol scancel sbatch srun salloc sacct; do \
                make -C "src/${tool}" || echo "⚠️  Failed to build ${tool}"; \
            done; \
        }; \
    fi

# 安装到临时目录并打包
USER root
RUN set -eux; \
    mkdir -p /home/builder/apk-output; \
    if grep -q "SKIP_SLURM_BUILD=1" /home/builder/build/.srcdir; then \
        echo "⚠️  Skipping SLURM installation"; \
        touch /home/builder/apk-output/.skip_slurm; \
    else \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        # 创建安装目录
        mkdir -p /tmp/slurm-install/usr/local/slurm/bin; \
        mkdir -p /tmp/slurm-install/usr/local/slurm/lib; \
        mkdir -p /tmp/slurm-install/etc/slurm; \
        echo ">>> Installing SLURM client tools..."; \
        # 复制客户端工具
        tool_count=0; \
        for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct; do \
            # 尝试从不同位置查找编译好的工具
            if [ -f "src/${cmd}/${cmd}" ]; then \
                cp -f "src/${cmd}/${cmd}" /tmp/slurm-install/usr/local/slurm/bin/; \
                chmod +x /tmp/slurm-install/usr/local/slurm/bin/${cmd}; \
                tool_count=$((tool_count + 1)); \
                echo "  ✓ Installed: ${cmd}"; \
            elif [ -f "src/${cmd}/.libs/${cmd}" ]; then \
                cp -f "src/${cmd}/.libs/${cmd}" /tmp/slurm-install/usr/local/slurm/bin/; \
                chmod +x /tmp/slurm-install/usr/local/slurm/bin/${cmd}; \
                tool_count=$((tool_count + 1)); \
                echo "  ✓ Installed: ${cmd} (from .libs)"; \
            else \
                echo "  ✗ Not found: ${cmd}"; \
            fi; \
        done; \
        # 复制共享库
        if [ -d "src/common/.libs" ]; then \
            find src/common/.libs -name "libslurm*.so*" -exec cp -f {} /tmp/slurm-install/usr/local/slurm/lib/ \; 2>/dev/null || true; \
            echo "  ✓ Copied SLURM libraries"; \
        fi; \
        # 创建版本信息和脚本
        echo "${SLURM_VERSION}" > /tmp/slurm-install/usr/local/slurm/VERSION; \
        echo "  ✓ Installed ${tool_count} client tools"; \
        if [ "$tool_count" -eq 0 ]; then \
            echo "  ⚠️  No tools installed, marking as skipped"; \
            touch /home/builder/apk-output/.skip_slurm; \
        fi; \
    fi

    # 创建安装脚本（如果不存在）
    RUN set -eux; \
        echo "📝 Setting up installation scripts..."; \
        mkdir -p /tmp/slurm-install; \
        if [ ! -f /tmp/slurm-install/install.sh ]; then \
            echo '#!/bin/sh' > /tmp/slurm-install/install.sh; \
            echo 'echo "Installing SLURM client tools..."' >> /tmp/slurm-install/install.sh; \
            echo 'cp -r usr/local/slurm /usr/local/' >> /tmp/slurm-install/install.sh; \
            echo 'echo "SLURM client tools installed to /usr/local/slurm"' >> /tmp/slurm-install/install.sh; \
            chmod +x /tmp/slurm-install/install.sh; \
            echo "✓ Created install.sh"; \
        fi; \
        if [ ! -f /tmp/slurm-install/uninstall.sh ]; then \
            echo '#!/bin/sh' > /tmp/slurm-install/uninstall.sh; \
            echo 'rm -rf /usr/local/slurm' >> /tmp/slurm-install/uninstall.sh; \
            chmod +x /tmp/slurm-install/uninstall.sh; \
            echo "✓ Created uninstall.sh"; \
        fi; \
        if [ ! -f /tmp/slurm-install/README.md ]; then \
            echo '# SLURM Client Tools' > /tmp/slurm-install/README.md; \
            echo 'Run ./install.sh to install' >> /tmp/slurm-install/README.md; \
            echo "✓ Created README.md"; \
        fi
    
    # 设置脚本权限并打包
    RUN set -eux; \
    echo "📦 Creating SLURM Alpine package..."; \
    if [ -f /home/builder/apk-output/.skip_slurm ]; then \
        echo "⚠️  SLURM build skipped, no package to create"; \
    else \
        # 检查是否有安装的工具
        if [ ! -d /tmp/slurm-install/usr/local/slurm/bin ] || [ -z "$(ls -A /tmp/slurm-install/usr/local/slurm/bin 2>/dev/null)" ]; then \
            echo "❌ No SLURM tools found in /tmp/slurm-install/usr/local/slurm/bin"; \
            echo "⚠️  Marking build as skipped"; \
            touch /home/builder/apk-output/.skip_slurm; \
        else \
            echo "✓ Found SLURM tools:"; \
            ls -lh /tmp/slurm-install/usr/local/slurm/bin/; \
            # 复制安装/卸载脚本
            if [ -f /tmp/slurm-install/install.sh ]; then \
                chmod +x /tmp/slurm-install/install.sh /tmp/slurm-install/uninstall.sh; \
                echo "✓ Set script permissions"; \
            else \
                echo "⚠️  Install scripts not found, but continuing with packaging"; \
            fi; \
            # 打包
            cd /tmp/slurm-install; \
            tar czf /home/builder/apk-output/slurm-client-${SLURM_VERSION}-alpine.tar.gz . || { \
                echo "❌ Failed to create tar.gz package"; \
                touch /home/builder/apk-output/.skip_slurm; \
                exit 0; \
            }; \
            if [ -f /home/builder/apk-output/slurm-client-${SLURM_VERSION}-alpine.tar.gz ]; then \
                echo "✓ SLURM Alpine package created:"; \
                ls -lh /home/builder/apk-output/*.tar.gz; \
                # 验证包内容
                echo "📦 Package contents (first 20 files):"; \
                tar tzf /home/builder/apk-output/slurm-client-${SLURM_VERSION}-alpine.tar.gz | head -20; \
            else \
                echo "❌ Package file not created"; \
                touch /home/builder/apk-output/.skip_slurm; \
            fi; \
        fi; \
    fi

# 收集 APK 包到 /out
RUN set -eux; \
    mkdir -p /out; \
    echo "📦 Collecting APK packages..."; \
    if [ -f /home/builder/apk-output/.skip_slurm ]; then \
        echo "⚠️  SLURM APK build was skipped"; \
        touch /out/.skip_slurm; \
    else \
        apk_count=$(ls /home/builder/apk-output/*.tar.gz 2>/dev/null | wc -l || echo 0); \
        echo "Found ${apk_count} APK files in /home/builder/apk-output"; \
        if [ "$apk_count" -gt 0 ]; then \
            ls -lh /home/builder/apk-output/*.tar.gz; \
            cp /home/builder/apk-output/*.tar.gz /out/ || { \
                echo "❌ Failed to copy APK packages"; \
                exit 1; \
            }; \
            echo "✓ Successfully copied ${apk_count} SLURM Alpine package(s)"; \
            ls -lh /out/*.tar.gz; \
        else \
            echo "⚠️  No SLURM Alpine packages found"; \
            touch /out/.skip_slurm; \
        fi; \
    fi; \
    # Final verification
    echo "📊 Final /out contents:"; \
    ls -lh /out/ || echo "⚠️  /out is empty"

#================================================
# Stage 4: 最终镜像 - 使用 nginx 提供下载服务
#================================================

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