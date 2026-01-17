# =============================================================================
# 全局 ARG 声明（用于多阶段构建的 FROM 语句）
# Global ARG declarations for multi-stage build FROM statements
# 这些值将通过 build.sh 的 --build-arg 参数从 .env 文件传入
# 提供默认值以保持向后兼容性
#
# SLURM 构建策略说明：
# - cgroup 支持不在编译时硬编码启用，而是通过运行时配置管理
# - 这允许同一个 SLURM 包在 Docker 容器和物理机环境中灵活部署
# - Docker 环境：使用 proctrack/pgid 或 proctrack/linuxproc，无 cgroup
# - 物理机环境：可通过 slurm.conf 启用 proctrack/cgroup 和 task/cgroup
# - 参考：src/slurm-master/config/README.md
# =============================================================================
ARG UBUNTU_VERSION=22.04
ARG ALMALINUX_VERSION=9.3-minimal
ARG APT_MIRROR=mirrors.aliyun.com
ARG YUM_MIRROR=mirrors.aliyun.com
ARG MUNGE_VERSION=0.5.16

# =============================================================================
# Stage 1: Build SLURM deb packages (Ubuntu 22.04)
# =============================================================================
FROM ubuntu:${UBUNTU_VERSION} AS deb-builder

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# Build control flags - set to "true" to enable building specific components
# 构建控制开关 - 设置为 "true" 来启用特定组件的构建
ARG BUILD_SLURM=true
ARG BUILD_SALTSTACK=true
ARG BUILD_CATEGRAF=true
ARG BUILD_SINGULARITY=false
ARG APT_MIRROR=mirrors.aliyun.com
ARG MUNGE_VERSION

# Copy third_party directory for offline builds
COPY third_party/ /third_party/

# SLURM version configuration - update these when upgrading
# 更新 SLURM 版本时只需修改这两个变量
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2

# SaltStack version configuration
ARG SALTSTACK_VERSION=v3007.8

# Categraf version configuration
ARG CATEGRAF_VERSION=v0.4.25

# Singularity version configuration
ARG SINGULARITY_VERSION=v4.3.6

# Accept optional tarball path relative to build context root
# 默认在 src/apphub/ 目录查找 tarball（因为构建上下文是项目根目录）
ARG SLURM_TARBALL_PATH=src/apphub/${SLURM_TARBALL_NAME}

# 配置APT镜像源和安装构建依赖（分步骤避免网络问题）
RUN set -eux; \
    # 清除可能干扰的代理设置（构建容器内部无法访问宿主机代理）
    unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy no_proxy NO_PROXY || true; \
    rm -f /etc/apt/apt.conf.d/*proxy* 2>/dev/null || true; \
    echo 'Acquire::http::Proxy "false";' > /etc/apt/apt.conf.d/99no-proxy; \
    echo 'Acquire::https::Proxy "false";' >> /etc/apt/apt.conf.d/99no-proxy; \
    # 备份原始sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    # 根据架构配置镜像源（优先尝试阿里云，失败则回退到官方源）
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        sed -i "s|archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|ports.ubuntu.com/ubuntu-ports/|${APT_MIRROR}/ubuntu-ports/|g" /etc/apt/sources.list; \
    elif [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        echo "配置ARM64架构镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "配置AMD64架构镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    fi; \
    # 更新包列表并安装基础工具（带重试和回退机制）
    if ! apt-get update; then \
        echo "阿里云镜像源失败，回退到官方源..."; \
        mv /etc/apt/sources.list.backup /etc/apt/sources.list; \
        apt-get update; \
    fi; \
    apt-get install -y ca-certificates curl tzdata; \
    ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone

# Install build prerequisites (分组安装减少失败风险)
RUN apt-get install -y --no-install-recommends \
       wget git gpg \
       build-essential fakeroot devscripts equivs gdebi-core \
       pkg-config debhelper dh-autoreconf \
    && rm -rf /var/lib/apt/lists/*

# Install development libraries (单独安装避免依赖冲突)
RUN set -eux; \
    # 尝试更新包列表，失败则回退到官方源
    if ! apt-get update; then \
        echo "镜像源更新失败，尝试回退到官方源..."; \
        if [ -f /etc/apt/sources.list.backup ]; then \
            mv /etc/apt/sources.list.backup /etc/apt/sources.list; \
        fi; \
        apt-get update; \
    fi; \
    apt-get install -y --no-install-recommends \
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
        # 清除可能干扰的代理设置
        unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy no_proxy NO_PROXY || true; \
        # 清除 apt 代理配置
        rm -f /etc/apt/apt.conf.d/*proxy* 2>/dev/null || true; \
        echo 'Acquire::http::Proxy "false";' > /etc/apt/apt.conf.d/99no-proxy; \
        echo 'Acquire::https::Proxy "false";' >> /etc/apt/apt.conf.d/99no-proxy; \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        echo "📦 Updating package list..."; \
        apt-get update || { \
            echo "⚠️  Update failed, retrying in 5 seconds..."; \
            sleep 5; \
            apt-get update; \
        }; \
        echo "📦 Installing build dependencies..."; \
        mk-build-deps -i debian/control -t 'apt-get -y --no-install-recommends' --remove || { \
            echo "⚠️  mk-build-deps failed, trying with --fix-missing..."; \
            apt-get update && apt-get install -y --fix-missing --no-install-recommends -f; \
            mk-build-deps -i debian/control -t 'apt-get -y --no-install-recommends --fix-missing' --remove; \
        }; \
    fi

USER builder
RUN set -eux; \
    if grep -q "SKIP_SLURM_BUILD=1" /home/builder/build/.srcdir; then \
        echo "⚠️  Skipping SLURM package build"; \
    else \
        srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
        cd "/home/builder/build/${srcdir}"; \
        echo "📦 Building SLURM DEB packages..."; \
        echo "Note: Building without hardcoded cgroup dependency - use system defaults"; \
        echo "      cgroup features will be configured via slurm.conf, not at compile time"; \
        echo ">>> Current directory: $(pwd)"; \
        echo ">>> SLURM source directory contents:"; \
        ls -la | head -20; \
        # Use DEB_BUILD_OPTIONS to pass configuration (skip tests, minimal build)
        export DEB_BUILD_OPTIONS="nocheck parallel=$(nproc)"; \
        echo ">>> DEB_BUILD_OPTIONS: $DEB_BUILD_OPTIONS"; \
        echo ">>> Starting: dpkg-buildpackage -b -uc"; \
        build_exit_code=0; \
        if ! dpkg-buildpackage -b -uc 2>&1 | tee /tmp/dpkg-build.log; then \
            build_exit_code=$?; \
        fi; \
        echo ">>> dpkg-buildpackage exit code: $build_exit_code"; \
        if [ "$build_exit_code" -ne 0 ]; then \
            echo "❌ DEB build failed with exit code: $build_exit_code"; \
            echo ">>> Last 150 lines of build log:"; \
            tail -150 /tmp/dpkg-build.log; \
            echo ">>> Searching for error messages:"; \
            grep -iE "error|fatal|failed|cannot find|undefined reference|missing|configure" /tmp/dpkg-build.log | tail -30 || echo "No specific error patterns found"; \
            echo ">>> Checking debian/rules file:"; \
            head -50 debian/rules 2>/dev/null || echo "debian/rules not found"; \
            echo "⚠️  DEB package build failed - skipping for now"; \
            mkdir -p /home/builder/debs; \
            touch /home/builder/debs/.skip_slurm_deb; \
        else \
            echo "✓ SLURM DEB build completed"; \
            echo ">>> Checking for generated DEB packages..."; \
            deb_count=$(find /home/builder/build -name "slurm*.deb" -type f 2>/dev/null | wc -l); \
            if [ "$deb_count" -gt 0 ]; then \
                echo "✓ Found $deb_count DEB package(s)"; \
                mkdir -p /home/builder/debs; \
                find /home/builder/build -maxdepth 2 -name "slurm*.deb" -type f -exec cp {} /home/builder/debs/ \;; \
                ls -lh /home/builder/debs/; \
            else \
                echo "⚠️  Build succeeded but no .deb files found"; \
                echo ">>> Searching for any .deb files:"; \
                find /home/builder/build -name "*.deb" -type f || echo "No .deb files found"; \
            fi; \
        fi; \
    fi


# Download SaltStack packages from GitHub releases
USER root
ARG BUILD_SALTSTACK
ARG SALTSTACK_VERSION=v3007.8
ARG GITHUB_PROXY

# Use BuildKit cache mount for package caching
# This allows packages to be reused across builds without re-downloading
RUN --mount=type=cache,target=/var/cache/saltstack-deb,sharing=locked \
    set -eux; \
    if [ "${BUILD_SALTSTACK}" = "true" ]; then \
        mkdir -p /saltstack-deb; \
        cd /var/cache/saltstack-deb; \
        # ========================================
        # 包缓存优化：检查并复用缓存中的包
        # ========================================
        cached_count=$(ls -1 *.deb 2>/dev/null | wc -l || echo 0); \
        if [ "$cached_count" -gt 0 ]; then \
            echo "📦 发现缓存的 SaltStack deb 包: ${cached_count} 个"; \
            echo "✓ 验证缓存包完整性..."; \
            # 验证并复制有效的包到目标目录
            valid_count=0; \
            for pkg in *.deb; do \
                if [ -f "$pkg" ] && [ -s "$pkg" ]; then \
                    cp "$pkg" /saltstack-deb/ 2>/dev/null || true; \
                    valid_count=$((valid_count + 1)); \
                fi; \
            done; \
            echo "✓ 复制了 ${valid_count} 个有效包到构建目录"; \
        fi; \
        # ========================================
        # 网络下载（仅下载缺失的包）
        # ========================================
        echo "📦 检查 SaltStack ${SALTSTACK_VERSION} deb packages..."; \
        # 配置代理（如果提供）
        if [ -n "${GITHUB_PROXY:-}" ]; then \
            echo "🌐 Using proxy: ${GITHUB_PROXY}"; \
            export ALL_PROXY="${GITHUB_PROXY}"; \
            export HTTP_PROXY="${GITHUB_PROXY}"; \
            export HTTPS_PROXY="${GITHUB_PROXY}"; \
            export http_proxy="${GITHUB_PROXY}"; \
            export https_proxy="${GITHUB_PROXY}"; \
        fi; \
        # GitHub releases 基础 URL (修正版本号格式，确保有 v 前缀)
        VERSION_NUM="${SALTSTACK_VERSION#v}"; \
        # 确保 release tag 有 v 前缀
        RELEASE_TAG="${SALTSTACK_VERSION}"; \
        if [[ ! "$RELEASE_TAG" =~ ^v ]]; then \
            RELEASE_TAG="v${RELEASE_TAG}"; \
        fi; \
        BASE_URL="https://ghfast.top/github.com/saltstack/salt/releases/download/${RELEASE_TAG}"; \
        echo "Version: ${VERSION_NUM}"; \
        echo "Release Tag: ${RELEASE_TAG}"; \
        echo "Base URL: ${BASE_URL}"; \
        # 下载两种架构的 deb 包
        total_downloaded=0; \
        total_cached=0; \
        for ARCH_SUFFIX in amd64 arm64; do \
            echo ""; \
            echo "📥 Processing ${ARCH_SUFFIX} packages..."; \
            arch_count=0; \
            # 检查所有主要的 deb 包
            for pkg in salt-common salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do \
                PKG_FILE="${pkg}_${VERSION_NUM}_${ARCH_SUFFIX}.deb"; \
                # 检查包是否已存在（从缓存复制或已下载）
                if [ -f "${PKG_FILE}" ] && [ -s "${PKG_FILE}" ]; then \
                    echo "✓ Cached: ${PKG_FILE}"; \
                    total_cached=$((total_cached + 1)); \
                    arch_count=$((arch_count + 1)); \
                    continue; \
                fi; \
                echo "Downloading: ${PKG_FILE}"; \
                if [ -f "/third_party/saltstack/${PKG_FILE}" ]; then \
                    echo "📦 Using local file: ${PKG_FILE}"; \
                    cp "/third_party/saltstack/${PKG_FILE}" /saltstack-deb/ 2>/dev/null || true; \
                    total_downloaded=$((total_downloaded + 1)); \
                    arch_count=$((arch_count + 1)); \
                    continue; \
                fi; \
                for attempt in 1 2 3; do \
                    if wget --timeout=60 --tries=3 -nv "${BASE_URL}/${PKG_FILE}"; then \
                        echo "✓ Downloaded: ${PKG_FILE}"; \
                        # 生成 SHA256 校验文件（用于后续验证）
                        shasum -a 256 "${PKG_FILE}" > "${PKG_FILE}.sha256" 2>/dev/null || true; \
                        # 复制到构建目录
                        cp "${PKG_FILE}" /saltstack-deb/ 2>/dev/null || true; \
                        total_downloaded=$((total_downloaded + 1)); \
                        arch_count=$((arch_count + 1)); \
                        break; \
                    else \
                        echo "⚠️  Attempt ${attempt}/3 failed"; \
                        if [ $attempt -lt 3 ]; then sleep 2; fi; \
                    fi; \
                done || echo "✗ Failed to download ${PKG_FILE}"; \
            done; \
            echo "✓ ${ARCH_SUFFIX}: ${arch_count} packages available"; \
        done; \
        # 检查结果
        echo ""; \
        echo "📊 Package Summary:"; \
        echo "   Cached: ${total_cached}"; \
        echo "   Downloaded: ${total_downloaded}"; \
        total_packages=$((total_cached + total_downloaded)); \
        if [ "$total_packages" -gt 0 ]; then \
            echo "✓ Total available: ${total_packages} SaltStack deb packages"; \
            echo ""; \
            echo "AMD64 packages:"; \
            ls -lh /saltstack-deb/*_amd64.deb 2>/dev/null || echo "  (none)"; \
            echo ""; \
            echo "ARM64 packages:"; \
            ls -lh /saltstack-deb/*_arm64.deb 2>/dev/null || echo "  (none)"; \
        else \
            echo "⚠️  No SaltStack packages available"; \
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
# Stage 2: Build SLURM rpm packages (AlmaLinux 9)
# 使用 AlmaLinux 替代 Rocky Linux（Rocky 的 AppStream 模块元数据损坏）
# =============================================================================
FROM almalinux:9.3-minimal AS rpm-builder

ENV TZ=Asia/Shanghai

# Build control flags
ARG BUILD_SLURM=true
ARG BUILD_SALTSTACK=true
ARG YUM_MIRROR=mirrors.aliyun.com
ARG MUNGE_VERSION
ARG GITHUB_PROXY
ARG GITHUB_MIRROR=https://ghfast.top/

# Copy third_party directory for offline builds
COPY third_party/ /third_party/

# SLURM version configuration (same as deb builder)
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2
ARG SLURM_TARBALL_PATH=src/apphub/${SLURM_TARBALL_NAME}

# SaltStack version configuration (same as deb builder)
ARG SALTSTACK_VERSION=v3007.8

# 配置 AlmaLinux 镜像源并安装 dnf（minimal 镜像只有 microdnf）
# 注意：也可以通过 docker build --build-arg HTTP_PROXY=... 使用代理
RUN set -eux; \
    echo "尝试配置 AlmaLinux 镜像源..."; \
    # 清理可能损坏的 DNF 模块状态（解决 YAML 解析错误）
    rm -rf /etc/dnf/modules.d/* 2>/dev/null || true; \
    # 备份原始配置
    cp -r /etc/yum.repos.d /etc/yum.repos.d.backup 2>/dev/null || true; \
    # 直接写入阿里云镜像源配置（更可靠，避免 sed 替换问题）
    MIRROR_HOST="${YUM_MIRROR:-mirrors.aliyun.com}"; \
    echo "Using mirror: ${MIRROR_HOST}"; \
    printf '%s\n' \
        '[baseos]' \
        'name=AlmaLinux $releasever - BaseOS' \
        "baseurl=https://${MIRROR_HOST}/almalinux/\$releasever/BaseOS/\$basearch/os/" \
        'enabled=1' \
        'gpgcheck=1' \
        'countme=0' \
        'gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-9' \
        > /etc/yum.repos.d/almalinux-baseos.repo; \
    printf '%s\n' \
        '[appstream]' \
        'name=AlmaLinux $releasever - AppStream' \
        "baseurl=https://${MIRROR_HOST}/almalinux/\$releasever/AppStream/\$basearch/os/" \
        'enabled=1' \
        'gpgcheck=1' \
        'countme=0' \
        'gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-9' \
        > /etc/yum.repos.d/almalinux-appstream.repo; \
    printf '%s\n' \
        '[crb]' \
        'name=AlmaLinux $releasever - CRB' \
        "baseurl=https://${MIRROR_HOST}/almalinux/\$releasever/CRB/\$basearch/os/" \
        'enabled=0' \
        'gpgcheck=1' \
        'countme=0' \
        'gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-9' \
        > /etc/yum.repos.d/almalinux-crb.repo; \
    printf '%s\n' \
        '[extras]' \
        'name=AlmaLinux $releasever - Extras' \
        "baseurl=https://${MIRROR_HOST}/almalinux/\$releasever/extras/\$basearch/os/" \
        'enabled=0' \
        'gpgcheck=1' \
        'countme=0' \
        'gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-9' \
        > /etc/yum.repos.d/almalinux-extras.repo; \
    # 删除原始 repo 文件中可能残留的 mirrorlist 配置
    rm -f /etc/yum.repos.d/almalinux*.repo.bak 2>/dev/null || true; \
    rm -f /etc/yum.repos.d/almalinux-*source*.repo 2>/dev/null || true; \
    rm -f /etc/yum.repos.d/almalinux-*debug*.repo 2>/dev/null || true; \
    # 显示配置的 repo 文件以便调试
    echo "=== Configured repositories ==="; \
    cat /etc/yum.repos.d/almalinux-baseos.repo; \
    # minimal 镜像只有 microdnf，先安装 dnf
    echo "📦 Installing dnf on minimal image..."; \
    microdnf clean all; \
    microdnf install -y dnf || { echo "❌ Failed to install dnf"; exit 1; }; \
    dnf clean all; \
    dnf makecache || true; \
    # 更新系统以确保基础包是最新的（镜像源已配置为阿里云）
    echo "📦 Updating system packages..."; \
    dnf update -y --setopt=timeout=300 || echo "⚠️ dnf update failed, continuing anyway"

# Install build prerequisites and enable required repositories
RUN set -eux; \
    # Enable PowerTools/CRB repository for additional development packages
    dnf config-manager --set-enabled crb 2>/dev/null || \
    dnf config-manager --set-enabled powertools 2>/dev/null || \
    echo "PowerTools/CRB repository not available"; \
    dnf install -y epel-release || echo "EPEL repository not available"; \
    # 更新元数据缓存
    dnf makecache --refresh || dnf makecache || true; \
    # Install basic build dependencies with retry
    echo "📦 Installing RPM build tools..."; \
    for attempt in 1 2 3; do \
        echo "Attempt ${attempt}/3..."; \
        if dnf install -y --setopt=timeout=300 --setopt=retries=5 \
            rpm-build \
            rpmdevtools \
            redhat-rpm-config \
            gcc \
            make \
            wget \
            tar \
            bzip2 \
            pam-devel \
            readline-devel \
            perl-ExtUtils-MakeMaker \
            openssl-devel; then \
            echo "✓ RPM build tools installed successfully"; \
            break; \
        else \
            echo "⚠️ Attempt ${attempt} failed, cleaning cache..."; \
            dnf clean all; \
            sleep 5; \
        fi; \
        if [ "$attempt" = "3" ]; then \
            echo "❌ Failed after 3 attempts"; \
            exit 1; \
        fi; \
    done; \
    # Verify rpmdevtools installation (using command -v instead of which)
    echo "✓ Verifying rpmdevtools installation..."; \
    if ! command -v rpmdev-setuptree >/dev/null 2>&1; then \
        echo "❌ rpmdev-setuptree not found, trying to reinstall..."; \
        dnf reinstall -y rpmdevtools || dnf install -y rpmdevtools; \
    fi; \
    if command -v rpmdev-setuptree >/dev/null 2>&1; then \
        echo "✓ rpmdev-setuptree found: $(command -v rpmdev-setuptree)"; \
    else \
        echo "⚠️  rpmdev-setuptree still not available, will use manual setup"; \
    fi; \
    dnf clean all; \
    # Install SLURM build dependencies (required by rpmbuild)
    # Note: Some packages have different names or are in CRB repo
    echo "📦 Installing SLURM build dependencies..."; \
    dnf install -y \
        autoconf \
        automake \
        systemd \
        || { echo "❌ Failed to install basic dependencies"; exit 1; }; \
    # Install mariadb-devel (required by SLURM)
    dnf module reset mysql -y 2>/dev/null || true; \
    dnf install -y mysql-devel 2>/dev/null || \
    dnf install -y mariadb-connector-c-devel 2>/dev/null || \
    { echo "❌ Failed to install mariadb-devel"; exit 1; }; \
    # Try to install munge from EPEL first
    dnf install -y \
        munge-devel \
        munge-libs \
        2>/dev/null && echo "✓ munge packages installed from repository" || { \
        echo "⚠️  munge not available in repos, will build from source..."; \
        cd /tmp; \
        MUNGE_TARBALL="munge-${MUNGE_VERSION}.tar.xz"; \
        MUNGE_DIRECT_URL="https://github.com/dun/munge/releases/download/munge-${MUNGE_VERSION}/${MUNGE_TARBALL}"; \
        DOWNLOAD_SUCCESS=false; \
        # 方式0: 检查本地文件
        if [ -f "/third_party/munge/${MUNGE_TARBALL}" ]; then \
            echo "📦 [本地] Using local Munge tarball..."; \
            cp "/third_party/munge/${MUNGE_TARBALL}" munge.tar.xz; \
            DOWNLOAD_SUCCESS=true; \
        fi; \
        # 方式1: GITHUB_MIRROR
        if [ "${DOWNLOAD_SUCCESS}" = "false" ] && [ -n "${GITHUB_MIRROR:-}" ]; then \
            MIRROR_URL="${GITHUB_MIRROR}github.com/dun/munge/releases/download/munge-${MUNGE_VERSION}/${MUNGE_TARBALL}"; \
            echo "📥 [方式1] GITHUB_MIRROR: ${MIRROR_URL}"; \
            if wget --timeout=60 --tries=3 -nv "${MIRROR_URL}" -O munge.tar.xz 2>/dev/null; then \
                echo "✓ Downloaded Munge via GITHUB_MIRROR"; \
                DOWNLOAD_SUCCESS=true; \
            else \
                echo "⚠️  GITHUB_MIRROR failed"; \
            fi; \
        fi; \
        # 方式2: GITHUB_PROXY
        if [ "${DOWNLOAD_SUCCESS}" = "false" ] && [ -n "${GITHUB_PROXY:-}" ]; then \
            echo "📥 [方式2] GITHUB_PROXY: ${MUNGE_DIRECT_URL}"; \
            export http_proxy="${GITHUB_PROXY}"; \
            export https_proxy="${GITHUB_PROXY}"; \
            if wget --timeout=60 --tries=3 -nv "${MUNGE_DIRECT_URL}" -O munge.tar.xz 2>/dev/null; then \
                echo "✓ Downloaded Munge via GITHUB_PROXY"; \
                DOWNLOAD_SUCCESS=true; \
            fi; \
            unset http_proxy https_proxy; \
            if [ "${DOWNLOAD_SUCCESS}" = "false" ]; then \
                echo "⚠️  GITHUB_PROXY failed"; \
            fi; \
        fi; \
        # 方式3: 直接下载
        if [ "${DOWNLOAD_SUCCESS}" = "false" ]; then \
            echo "📥 [方式3] Direct: ${MUNGE_DIRECT_URL}"; \
            if wget --timeout=60 --tries=3 -nv "${MUNGE_DIRECT_URL}" -O munge.tar.xz; then \
                echo "✓ Downloaded Munge directly"; \
                DOWNLOAD_SUCCESS=true; \
            else \
                echo "❌ All download methods failed for Munge"; \
                exit 1; \
            fi; \
        fi; \
        tar xf munge.tar.xz; \
        cd "munge-${MUNGE_VERSION}"; \
        ./configure --prefix=/usr --sysconfdir=/etc --localstatedir=/var; \
        make -j$(nproc); \
        make install; \
        ldconfig; \
        cd /tmp && rm -rf munge*; \
        echo "✓ munge built and installed from source"; \
    }; \
    # hwloc and other optional libs
    dnf install -y \
        hwloc-devel \
        2>/dev/null || echo "⚠️  hwloc-devel not available"; \
    dnf install -y \
        json-c-devel \
        2>/dev/null || echo "⚠️  json-c-devel not available"; \
    dnf install -y \
        libyaml-devel \
        2>/dev/null || echo "⚠️  libyaml-devel not available"; \
    echo "✓ SLURM build dependencies installed (with available packages)"

# Add non-root builder user
RUN useradd -m -u 1000 builder
USER builder
WORKDIR /home/builder/build

# Setup RPM build environment
RUN set -eux; \
    echo "📁 Setting up RPM build tree..."; \
    # Check if rpmdev-setuptree is available
    if command -v rpmdev-setuptree >/dev/null 2>&1; then \
        rpmdev-setuptree; \
        echo "✓ RPM build tree created successfully"; \
    else \
        # Manual setup if rpmdev-setuptree is not available
        echo "⚠️  rpmdev-setuptree not found, creating directories manually..."; \
        mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}; \
        echo "%_topdir %(echo \$HOME)/rpmbuild" > ~/.rpmmacros; \
        echo "✓ RPM build tree created manually"; \
    fi; \
    ls -la ~/rpmbuild/ || echo "Warning: rpmbuild directory check failed"

# Copy SLURM source tarball
ARG SLURM_TARBALL_PATH
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# Build SLURM RPM packages using official rpmbuild method
# Reference: https://slurm.schedmd.com/quickstart_admin.html#rpmbuild
RUN set -eux; \
    if [ "${BUILD_SLURM}" = "true" ]; then \
        echo "📦 Building SLURM ${SLURM_VERSION} RPM packages using rpmbuild..."; \
        cd /home/builder/build; \
        # Find the SLURM tarball
        tarball=$(ls slurm-*.tar.bz2 | head -1); \
        echo "✓ Found SLURM tarball: ${tarball}"; \
        # Setup rpmbuild directories
        echo ">>> Setting up rpmbuild environment..."; \
        mkdir -p /home/builder/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}; \
        # Create .rpmmacros file for custom configurations
        # Note: Define macros that SLURM spec file will recognize
        echo '%_topdir %(echo $HOME)/rpmbuild' > ~/.rpmmacros; \
        echo '%_prefix /usr' >> ~/.rpmmacros; \
        echo '%_slurm_sysconfdir %{_prefix}/etc/slurm' >> ~/.rpmmacros; \
        # For SLURM spec file - define conditional macros
        echo '%define with_munge 1' >> ~/.rpmmacros; \
        echo '%define without_cgroup 1' >> ~/.rpmmacros; \
        echo "✓ Created ~/.rpmmacros configuration"; \
        cat ~/.rpmmacros; \
        echo "Note: cgroup support disabled at build time - use system defaults"; \
        # Build RPMs directly using rpmbuild -ta (recommended by official docs)
        # The --with-munge and --without-cgroup options are already defined in ~/.rpmmacros
        echo ">>> Building SLURM RPM packages (this may take 10-15 minutes)..."; \
        echo ">>> Using tarball-to-rpms workflow"; \
        \
        # Extract tarball to inspect spec file
        echo ">>> Extracting tarball to find spec file..."; \
        tar -xjf "${tarball}" -C /tmp; \
        extracted_dir=$(tar -tjf "${tarball}" | head -1 | cut -d'/' -f1); \
        echo "✓ Extracted to: /tmp/${extracted_dir}"; \
        \
        # Find and display spec file
        spec_file="/tmp/${extracted_dir}/contribs/slurm.spec"; \
        echo ">>> Looking for spec file: ${spec_file}"; \
        if [ -f "${spec_file}" ]; then \
            echo "✓ Found spec file"; \
            head -30 "${spec_file}"; \
        else \
            echo "⚠️  Spec file not found at expected location"; \
            find /tmp/${extracted_dir} -name "*.spec" -type f; \
        fi; \
        \
        # Copy spec file to SPECS directory
        echo ">>> Setting up spec file for rpmbuild..."; \
        spec_path=$(find "/tmp/${extracted_dir}" -name "slurm.spec" -type f | head -1); \
        if [ -z "$spec_path" ]; then \
            echo "❌ ERROR: slurm.spec not found in tarball!"; \
            echo ">>> Available spec files:"; \
            find "/tmp/${extracted_dir}" -name "*.spec" -type f || echo "No .spec files found"; \
            echo ">>> Directory structure:"; \
            ls -la "/tmp/${extracted_dir}/" | head -20; \
            echo "⚠️  Skipping SLURM RPM build - spec file not found"; \
            mkdir -p /home/builder/rpms; \
            touch /home/builder/rpms/.skip_slurm; \
        else \
            echo ">>> Found spec file at: $spec_path"; \
            cp "$spec_path" /home/builder/rpmbuild/SPECS/; \
            \
            # Copy source tarball to SOURCES directory
            cp "${tarball}" /home/builder/rpmbuild/SOURCES/; \
            \
            # Build with explicit spec file reference
            echo ">>> Building RPMs using: rpmbuild -bb SPECS/slurm.spec"; \
            rpmbuild_exit_code=0; \
            if ! rpmbuild -bb /home/builder/rpmbuild/SPECS/slurm.spec 2>&1 | tee /tmp/rpmbuild.log; then \
                rpmbuild_exit_code=$?; \
            fi; \
            echo ">>> rpmbuild exit code: $rpmbuild_exit_code"; \
            \
            # Check if RPM files were created
            echo ">>> Checking for generated RPM packages..."; \
            rpm_count=$(find /home/builder/rpmbuild/RPMS -type f -name "*.rpm" 2>/dev/null | wc -l); \
            echo "Found $rpm_count RPM file(s)"; \
            \
            if [ "$rpm_count" -gt 0 ]; then \
                echo "✓ SLURM RPM build completed successfully"; \
                echo ">>> Verifying and collecting generated RPM packages:"; \
                mkdir -p /out; \
                find /home/builder/rpmbuild/RPMS -type f -name "*.rpm" -exec cp {} /out/ \;; \
                echo "✓ Copied $rpm_count RPM packages to /out"; \
                ls -lh /out/*.rpm 2>/dev/null; \
            else \
                echo "❌ ERROR: No .rpm files were created!"; \
                echo ">>> rpmbuild exit code was: $rpmbuild_exit_code"; \
                echo ">>> Last 200 lines of build log:"; \
                tail -200 /tmp/rpmbuild.log; \
                echo ">>> Searching for error messages:"; \
                grep -iE "error|fatal|failed|cannot find|undefined reference" /tmp/rpmbuild.log | tail -30 || echo "No error patterns found"; \
                echo ">>> Checking rpmbuild directory contents:"; \
                ls -la /home/builder/rpmbuild/; \
                ls -la /home/builder/rpmbuild/BUILD/ 2>/dev/null || echo "BUILD directory empty"; \
                mkdir -p /home/builder/rpms; \
                touch /home/builder/rpms/.skip_slurm; \
            fi; \
        fi; \
    else \
        echo "🚫 Skipping SLURM RPM build (BUILD_SLURM=false)"; \
        mkdir -p /home/builder/rpms; \
        touch /home/builder/rpms/.skip_slurm; \
    fi

# Download SaltStack packages from GitHub releases
USER root
ARG BUILD_SALTSTACK
ARG SALTSTACK_VERSION=v3007.8
ARG GITHUB_PROXY

# Use BuildKit cache mount for package caching
# This allows packages to be reused across builds without re-downloading
RUN --mount=type=cache,target=/var/cache/saltstack-rpm,sharing=locked \
    set -eux; \
    if [ "${BUILD_SALTSTACK}" = "true" ]; then \
        mkdir -p /saltstack-rpm; \
        cd /var/cache/saltstack-rpm; \
        # ========================================
        # 包缓存优化：检查并复用缓存中的包
        # ========================================
        cached_count=$(ls -1 *.rpm 2>/dev/null | wc -l || echo 0); \
        if [ "$cached_count" -gt 0 ]; then \
            echo "� 发现缓存的 SaltStack rpm 包: ${cached_count} 个"; \
            echo "✓ 验证缓存包完整性..."; \
            # 验证并复制有效的包到目标目录
            valid_count=0; \
            for pkg in *.rpm; do \
                if [ -f "$pkg" ] && [ -s "$pkg" ]; then \
                    cp "$pkg" /saltstack-rpm/ 2>/dev/null || true; \
                    valid_count=$((valid_count + 1)); \
                fi; \
            done; \
            echo "✓ 复制了 ${valid_count} 个有效包到构建目录"; \
        fi; \
        # ========================================
        # 网络下载（仅下载缺失的包）
        # ========================================
        echo "📦 检查 SaltStack ${SALTSTACK_VERSION} rpm packages..."; \
        # 配置代理（如果提供）
        if [ -n "${GITHUB_PROXY:-}" ]; then \
            echo "🌐 Using proxy: ${GITHUB_PROXY}"; \
            export ALL_PROXY="${GITHUB_PROXY}"; \
            export HTTP_PROXY="${GITHUB_PROXY}"; \
            export HTTPS_PROXY="${GITHUB_PROXY}"; \
            export http_proxy="${GITHUB_PROXY}"; \
            export https_proxy="${GITHUB_PROXY}"; \
        fi; \
        # GitHub releases 基础 URL (修正版本号格式，确保有 v 前缀)
        VERSION_NUM="${SALTSTACK_VERSION#v}"; \
        # 确保 release tag 有 v 前缀
        RELEASE_TAG="${SALTSTACK_VERSION}"; \
        if [[ ! "$RELEASE_TAG" =~ ^v ]]; then \
            RELEASE_TAG="v${RELEASE_TAG}"; \
        fi; \
        BASE_URL="https://ghfast.top/github.com/saltstack/salt/releases/download/${RELEASE_TAG}"; \
        echo "Version: ${VERSION_NUM}"; \
        echo "Release Tag: ${RELEASE_TAG}"; \
        echo "Base URL: ${BASE_URL}"; \
        # 下载两种架构的 rpm 包
        total_downloaded=0; \
        total_cached=0; \
        for ARCH_SUFFIX in x86_64 aarch64; do \
            echo ""; \
            echo "📥 Processing ${ARCH_SUFFIX} packages..."; \
            arch_count=0; \
            # 检查所有主要的 rpm 包
            for pkg in salt salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do \
                PKG_FILE="${pkg}-${VERSION_NUM}-0.${ARCH_SUFFIX}.rpm"; \
                # 检查包是否已存在（从缓存复制或已下载）
                if [ -f "${PKG_FILE}" ] && [ -s "${PKG_FILE}" ]; then \
                    echo "✓ Cached: ${PKG_FILE}"; \
                    total_cached=$((total_cached + 1)); \
                    arch_count=$((arch_count + 1)); \
                    continue; \
                fi; \
                echo "Downloading: ${PKG_FILE}"; \
                if [ -f "/third_party/saltstack/${PKG_FILE}" ]; then \
                    echo "📦 Using local file: ${PKG_FILE}"; \
                    cp "/third_party/saltstack/${PKG_FILE}" /saltstack-rpm/ 2>/dev/null || true; \
                    total_downloaded=$((total_downloaded + 1)); \
                    arch_count=$((arch_count + 1)); \
                    continue; \
                fi; \
                for attempt in 1 2 3; do \
                    if wget --timeout=60 --tries=3 -nv "${BASE_URL}/${PKG_FILE}"; then \
                        echo "✓ Downloaded: ${PKG_FILE}"; \
                        # 生成 SHA256 校验文件（用于后续验证）
                        sha256sum "${PKG_FILE}" > "${PKG_FILE}.sha256" 2>/dev/null || \
                        shasum -a 256 "${PKG_FILE}" > "${PKG_FILE}.sha256" 2>/dev/null || true; \
                        # 复制到构建目录
                        cp "${PKG_FILE}" /saltstack-rpm/ 2>/dev/null || true; \
                        total_downloaded=$((total_downloaded + 1)); \
                        arch_count=$((arch_count + 1)); \
                        break; \
                    else \
                        echo "⚠️  Attempt ${attempt}/3 failed"; \
                        if [ $attempt -lt 3 ]; then sleep 2; fi; \
                    fi; \
                done || echo "✗ Failed to download ${PKG_FILE}"; \
            done; \
            echo "✓ ${ARCH_SUFFIX}: ${arch_count} packages available"; \
        done; \
        # 检查结果
        echo ""; \
        echo "📊 Package Summary:"; \
        echo "   Cached: ${total_cached}"; \
        echo "   Downloaded: ${total_downloaded}"; \
        total_packages=$((total_cached + total_downloaded)); \
        if [ "$total_packages" -gt 0 ]; then \
            echo "✓ Total available: ${total_packages} SaltStack rpm packages"; \
            echo ""; \
            echo "x86_64 packages:"; \
            ls -lh /saltstack-rpm/*.x86_64.rpm 2>/dev/null || echo "  (none)"; \
            echo ""; \
            echo "aarch64 packages:"; \
            ls -lh /saltstack-rpm/*.aarch64.rpm 2>/dev/null || echo "  (none)"; \
        else \
            echo "⚠️  No SaltStack packages available"; \
        fi; \
    else \
        echo "⏭️  Skipping SaltStack download (BUILD_SALTSTACK=${BUILD_SALTSTACK})"; \
        mkdir -p /saltstack-rpm; \
    fi

# Collect RPM artifacts
RUN set -eux; \
    mkdir -p /out; \
    echo "📦 Collecting RPM packages..."; \
    if [ ! -f /home/builder/rpms/.skip_slurm ] && [ "${BUILD_SLURM}" = "true" ]; then \
        # Find and copy built SLURM RPMs from all possible locations
        echo ">>> Looking for SLURM RPM packages..."; \
        echo ">>> Searching in common RPM build locations:"; \
        # List all possible locations
        find /home/builder -name "*.rpm" -type f 2>/dev/null | head -20 || echo "No RPMs found with find command"; \
        # Check rpmbuild directory structure (standard rpmbuild location)
        if [ -d /home/builder/rpmbuild/RPMS ]; then \
            echo "  Checking: /home/builder/rpmbuild/RPMS"; \
            find /home/builder/rpmbuild/RPMS -type f -name '*.rpm' -exec cp {} /out/ \; 2>/dev/null || true; \
        fi; \
        # Check source directory (make rpm sometimes puts RPMs here)
        if [ -d /home/builder/build ]; then \
            echo "  Checking: /home/builder/build"; \
            find /home/builder/build -type f -name '*.rpm' -exec cp {} /out/ \; 2>/dev/null || true; \
        fi; \
        # Check home directory root (backup location)
        echo "  Checking: /home/builder"; \
        find /home/builder -maxdepth 3 -type f -name '*.rpm' -exec cp {} /out/ \; 2>/dev/null || true; \
        # Remove duplicates and debug symbols if needed
        cd /out && rm -f *-debuginfo-*.rpm *-debugsource-*.rpm 2>/dev/null || true; \
        # Count collected RPMs
        rpm_count=$(ls /out/*.rpm 2>/dev/null | wc -l || echo 0); \
        if [ "$rpm_count" -gt 0 ]; then \
            echo "✓ Successfully collected ${rpm_count} SLURM RPM package(s)"; \
            echo ">>> SLURM RPM packages:"; \
            ls -lh /out/*.rpm; \
        else \
            echo "⚠️  No SLURM RPM packages were found"; \
            echo ">>> Listing /home/builder structure for debugging:"; \
            ls -laR /home/builder/ | head -100 || true; \
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
    echo "✓ Total RPM packages in /out: ${total_rpm_count}"; \
    # Generate RPM repository metadata using createrepo_c (Rocky Linux has it)
    echo "🔧 Installing createrepo_c for metadata generation..."; \
    dnf install -y createrepo_c 2>/dev/null || { \
        echo "⚠️  createrepo_c not available, trying createrepo..."; \
        dnf install -y createrepo 2>/dev/null || echo "⚠️  No createrepo tools available"; \
    }; \
    # Separate SLURM and SaltStack RPMs into subdirectories
    mkdir -p /out/slurm-rpm /out/saltstack-rpm; \
    # Remove invalid/empty RPMs
    find /out -name "*.rpm" -size 0 -delete || true; \
    # Check if there are any RPM files to organize
    if ls /out/*.rpm >/dev/null 2>&1; then \
        echo "📦 Organizing RPM packages..."; \
        mv /out/slurm-*.rpm /out/slurm-rpm/ 2>/dev/null || true; \
        mv /out/salt-*.rpm /out/saltstack-rpm/ 2>/dev/null || true; \
        # Count packages in each directory
        slurm_count=$(ls /out/slurm-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
        salt_count=$(ls /out/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
        echo "  - SLURM RPMs: ${slurm_count}"; \
        echo "  - SaltStack RPMs: ${salt_count}"; \
        # Generate metadata for SLURM repository
        if [ "$slurm_count" -gt 0 ] && command -v createrepo_c >/dev/null 2>&1; then \
            echo "🔧 Generating SLURM RPM repository metadata..."; \
            cd /out/slurm-rpm && (createrepo_c . || echo "⚠️  createrepo_c failed, continuing anyway") && \
            echo "✓ Generated SLURM repodata: $(ls -d repodata 2>/dev/null || echo 'failed')"; \
        elif [ "$slurm_count" -gt 0 ] && command -v createrepo >/dev/null 2>&1; then \
            echo "🔧 Generating SLURM RPM repository metadata (using createrepo)..."; \
            cd /out/slurm-rpm && (createrepo . || echo "⚠️  createrepo failed, continuing anyway") && \
            echo "✓ Generated SLURM repodata"; \
        fi; \
        # Generate metadata for SaltStack repository
        if [ "$salt_count" -gt 0 ] && command -v createrepo_c >/dev/null 2>&1; then \
            echo "🔧 Generating SaltStack RPM repository metadata..."; \
            cd /out/saltstack-rpm && (createrepo_c . || echo "⚠️  createrepo_c failed, continuing anyway") && \
            echo "✓ Generated SaltStack repodata: $(ls -d repodata 2>/dev/null || echo 'failed')"; \
        elif [ "$salt_count" -gt 0 ] && command -v createrepo >/dev/null 2>&1; then \
            echo "🔧 Generating SaltStack RPM repository metadata (using createrepo)..."; \
            cd /out/saltstack-rpm && (createrepo . || echo "⚠️  createrepo failed, continuing anyway") && \
            echo "✓ Generated SaltStack repodata"; \
        fi; \
    fi

# =============================================================================
# Stage 3: Extract SLURM binaries from built packages (Support multi-arch)
# 从已构建的 DEB 包中提取二进制文件，支持 x86_64 和 aarch64
# =============================================================================
FROM ubuntu:${UBUNTU_VERSION} AS binary-builder

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# Build control flags
ARG BUILD_SLURM=true
ARG APT_MIRROR=mirrors.aliyun.com
# SLURM version configuration
ARG SLURM_VERSION=25.05.4

# 配置APT镜像源并安装提取工具
RUN set -eux; \
    # 备份原始源配置
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构并配置镜像源
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        sed -i "s|archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|ports.ubuntu.com/ubuntu-ports/|${APT_MIRROR}/ubuntu-ports/|g" /etc/apt/sources.list; \
    elif [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        echo "配置ARM64架构镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "配置AMD64架构镜像源..."; \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse" >> /etc/apt/sources.list; \
    fi; \
    # 更新包列表（带回退机制）
    if ! apt-get update; then \
        echo "阿里云镜像源失败，回退到官方源..."; \
        mv /etc/apt/sources.list.backup /etc/apt/sources.list; \
        apt-get update; \
    fi; \
    # 安装提取工具
    apt-get install -y --no-install-recommends dpkg-dev binutils file && \
    rm -rf /var/lib/apt/lists/*

# 从 deb-builder 复制构建好的 DEB 包（包含 amd64 和 arm64 两种架构）
COPY --from=deb-builder /out/*.deb /packages/deb/

# 提取两种架构的 SLURM 二进制文件
RUN set -eux; \
    if [ "$BUILD_SLURM" = "true" ]; then \
        mkdir -p /out/packages; \
        cd /packages/deb; \
        echo "📦 提取 SLURM 二进制文件从 DEB 包..."; \
        # 遍历两种架构
        for ARCH in amd64 arm64; do \
            # 转换为实际的架构名称
            if [ "$ARCH" = "amd64" ]; then \
                ARCH_DIR="x86_64"; \
            else \
                ARCH_DIR="aarch64"; \
            fi; \
            echo ""; \
            echo ">>> 处理 ${ARCH} (${ARCH_DIR}) 架构..."; \
            mkdir -p /out/packages/${ARCH_DIR}/bin; \
            mkdir -p /out/packages/${ARCH_DIR}/lib; \
            mkdir -p /tmp/extract/${ARCH_DIR}; \
            # 查找该架构的 slurm 包
            SLURM_PKG=$(ls slurm_*_${ARCH}.deb 2>/dev/null | head -1 || echo ""); \
            if [ -z "$SLURM_PKG" ]; then \
                echo "⚠️  未找到 slurm_*_${ARCH}.deb，尝试查找其他 slurm 包..."; \
                SLURM_PKG=$(ls slurm-*_${ARCH}.deb 2>/dev/null | head -1 || echo ""); \
            fi; \
            if [ -n "$SLURM_PKG" ] && [ -f "$SLURM_PKG" ]; then \
                echo "  ✓ 找到包: $SLURM_PKG"; \
                # 提取 DEB 包内容
                dpkg-deb -x "$SLURM_PKG" /tmp/extract/${ARCH_DIR}/ || { \
                    echo "  ⚠️  无法提取 $SLURM_PKG"; \
                    continue; \
                }; \
                # 复制二进制文件
                for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr; do \
                    # 查找命令（可能在多个位置）
                    FOUND=0; \
                    for bindir in /tmp/extract/${ARCH_DIR}/usr/bin \
                                  /tmp/extract/${ARCH_DIR}/usr/local/bin \
                                  /tmp/extract/${ARCH_DIR}/usr/sbin \
                                  /tmp/extract/${ARCH_DIR}/bin; do \
                        if [ -f "${bindir}/${cmd}" ]; then \
                            cp "${bindir}/${cmd}" /out/packages/${ARCH_DIR}/bin/; \
                            chmod +x /out/packages/${ARCH_DIR}/bin/${cmd}; \
                            echo "    ✓ ${cmd}"; \
                            FOUND=1; \
                            break; \
                        fi; \
                    done; \
                    if [ "$FOUND" = "0" ]; then \
                        echo "    ✗ 未找到: ${cmd}"; \
                    fi; \
                done; \
                # 复制库文件
                for libdir in /tmp/extract/${ARCH_DIR}/usr/lib \
                              /tmp/extract/${ARCH_DIR}/usr/lib64 \
                              /tmp/extract/${ARCH_DIR}/usr/lib/${ARCH}-linux-gnu \
                              /tmp/extract/${ARCH_DIR}/usr/local/lib; do \
                    if [ -d "${libdir}" ]; then \
                        find "${libdir}" -name "libslurm*.so*" -type f -exec cp {} /out/packages/${ARCH_DIR}/lib/ \; 2>/dev/null || true; \
                    fi; \
                done; \
            else \
                echo "  ❌ 未找到 ${ARCH} 架构的 SLURM DEB 包"; \
            fi; \
            # 保存版本信息
            echo "${SLURM_VERSION}" > /out/packages/${ARCH_DIR}/VERSION; \
            # 显示结果
            echo "  📁 二进制文件:"; \
            ls -lh /out/packages/${ARCH_DIR}/bin/ 2>/dev/null || echo "    (无)"; \
            echo "  📚 库文件:"; \
            ls -lh /out/packages/${ARCH_DIR}/lib/ 2>/dev/null || echo "    (无)"; \
        done; \
        # 清理临时文件
        rm -rf /tmp/extract; \
        echo ""; \
        echo "✅ SLURM 二进制文件提取完成"; \
    else \
        echo "🚫 跳过 SLURM 二进制文件提取 (BUILD_SLURM=false)"; \
        mkdir -p /out/packages/empty; \
        echo "No SLURM binaries extracted" > /out/packages/empty/README.txt; \
    fi

# =============================================================================
# Stage 4: Download Categraf (Pre-built Binary from GitHub Releases)
# 直接下载预编译二进制，避免 Go 版本兼容问题
# =============================================================================
FROM ubuntu:${UBUNTU_VERSION} AS categraf-builder

# Build control flags
ARG BUILD_CATEGRAF=true

# Categraf version configuration
ARG CATEGRAF_VERSION=v0.4.25
ARG CATEGRAF_REPO=https://ghfast.top/github.com/flashcatcloud/categraf.git

# 配置 Go 代理（中国镜像加速）- 仅用于 fallback 源码构建
ARG GO_PROXY=https://goproxy.cn,direct
ENV GOPROXY=${GO_PROXY}
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ARG APT_MIRROR=mirrors.aliyun.com

# 配置 APT 镜像源并安装依赖
RUN set -eux; \
    # 清理可能损坏的 apt 缓存
    rm -rf /var/lib/apt/lists/*; \
    # 备份原始源配置
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构并配置镜像源
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        sed -i "s|archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|ports.ubuntu.com/ubuntu-ports/|${APT_MIRROR}/ubuntu-ports/|g" /etc/apt/sources.list; \
    fi; \
    # 更新包列表（带回退机制）
    if ! apt-get update; then \
        echo "镜像源更新失败，回退到官方源..."; \
        mv /etc/apt/sources.list.backup /etc/apt/sources.list; \
        apt-get update; \
    fi; \
    # 安装下载工具（不再需要 golang，直接下载预编译二进制）
    # 注意: 需要 git 用于 fallback 源码构建时的代理配置
    apt-get install -y --no-install-recommends curl tar gzip ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

# Copy third_party directory for offline builds
COPY third_party/ /third_party/

# 复制通用构建脚本和应用特定脚本（仅用于 fallback）
COPY src/apphub/scripts/build-app.sh /scripts/build-app.sh
COPY src/apphub/scripts/categraf/ /scripts/categraf/
RUN chmod +x /scripts/build-app.sh /scripts/categraf/*.sh

# 创建输出目录
RUN mkdir -p /out

# 执行构建或下载（优先下载预编译二进制）
ARG BUILD_CATEGRAF
ARG GITHUB_PROXY
ARG GITHUB_MIRROR=https://ghfast.top/
RUN set -eux; \
    if [ "${BUILD_CATEGRAF}" = "true" ]; then \
        # 检测架构
        ARCH=$(uname -m); \
        if [ "${ARCH}" = "x86_64" ]; then \
            ARCH_SUFFIX="amd64"; \
        elif [ "${ARCH}" = "aarch64" ]; then \
            ARCH_SUFFIX="arm64"; \
        else \
            ARCH_SUFFIX="${ARCH}"; \
        fi; \
        TARBALL_NAME="categraf-${CATEGRAF_VERSION}-linux-${ARCH_SUFFIX}.tar.gz"; \
        GITHUB_DIRECT_URL="https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/${TARBALL_NAME}"; \
        DOWNLOAD_SUCCESS=false; \
        # 优先检查本地预下载的包
        if [ -f "/third_party/categraf/${TARBALL_NAME}" ]; then \
            echo "📦 使用本地 Categraf 二进制包: ${TARBALL_NAME}"; \
            cp "/third_party/categraf/${TARBALL_NAME}" /out/; \
            DOWNLOAD_SUCCESS=true; \
        fi; \
        # 方式1: 尝试使用 GITHUB_MIRROR 加速下载
        if [ "${DOWNLOAD_SUCCESS}" = "false" ] && [ -n "${GITHUB_MIRROR:-}" ]; then \
            MIRROR_URL="${GITHUB_MIRROR}github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/${TARBALL_NAME}"; \
            echo "📥 [方式1] 尝试通过 GITHUB_MIRROR 下载: ${MIRROR_URL}"; \
            if curl -fsSL --connect-timeout 30 --max-time 300 -o "/out/${TARBALL_NAME}" "${MIRROR_URL}"; then \
                echo "✓ Categraf ${CATEGRAF_VERSION} downloaded via GITHUB_MIRROR"; \
                DOWNLOAD_SUCCESS=true; \
            else \
                echo "⚠️  GITHUB_MIRROR 下载失败，尝试其他方式..."; \
            fi; \
        fi; \
        # 方式2: 尝试使用 GITHUB_PROXY 代理直接从 GitHub 下载
        if [ "${DOWNLOAD_SUCCESS}" = "false" ] && [ -n "${GITHUB_PROXY:-}" ]; then \
            echo "📥 [方式2] 尝试通过 GITHUB_PROXY 代理下载: ${GITHUB_DIRECT_URL}"; \
            echo "🌐 Using proxy: ${GITHUB_PROXY}"; \
            if curl --proxy "${GITHUB_PROXY}" -fsSL --connect-timeout 30 --max-time 300 -o "/out/${TARBALL_NAME}" "${GITHUB_DIRECT_URL}"; then \
                echo "✓ Categraf ${CATEGRAF_VERSION} downloaded via GITHUB_PROXY"; \
                DOWNLOAD_SUCCESS=true; \
            else \
                echo "⚠️  GITHUB_PROXY 代理下载失败..."; \
            fi; \
        fi; \
        # 方式3: 直接从 GitHub 下载（无代理）
        if [ "${DOWNLOAD_SUCCESS}" = "false" ]; then \
            echo "📥 [方式3] 尝试直接从 GitHub 下载: ${GITHUB_DIRECT_URL}"; \
            if curl -fsSL --connect-timeout 30 --max-time 300 -o "/out/${TARBALL_NAME}" "${GITHUB_DIRECT_URL}"; then \
                echo "✓ Categraf ${CATEGRAF_VERSION} downloaded directly from GitHub"; \
                DOWNLOAD_SUCCESS=true; \
            else \
                echo "⚠️  直接下载也失败了..."; \
            fi; \
        fi; \
        # 所有方式都失败，创建占位文件
        if [ "${DOWNLOAD_SUCCESS}" = "false" ]; then \
            echo "❌ 所有下载方式均失败"; \
            echo "⚠️  Creating placeholder - Categraf will need to be installed manually or via apphub"; \
            echo "DOWNLOAD_FAILED" > "/out/${TARBALL_NAME}.failed"; \
            touch "/out/${TARBALL_NAME}"; \
        fi; \
    else \
        echo "⏭️  Skipping Categraf (BUILD_CATEGRAF=${BUILD_CATEGRAF})"; \
    fi

# =============================================================================
# Stage 4.5: Download Pre-built Singularity (Container Runtime for HPC)
# =============================================================================
FROM ubuntu:${UBUNTU_VERSION} AS singularity-builder

# Build control flags
ARG BUILD_SINGULARITY=true

# Singularity version configuration
ARG SINGULARITY_VERSION=v4.3.6
ARG GITHUB_PROXY
ARG GITHUB_MIRROR=https://ghfast.top/
ARG APT_MIRROR=mirrors.aliyun.com

# 配置 APT 镜像源并安装依赖
RUN set -eux; \
    # 备份原始源配置
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构并配置镜像源
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        sed -i "s|archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|ports.ubuntu.com/ubuntu-ports/|${APT_MIRROR}/ubuntu-ports/|g" /etc/apt/sources.list; \
    fi; \
    # 更新包列表
    apt-get update; \
    # 安装工具
    apt-get install -y --no-install-recommends curl tar gzip binutils ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy third_party directory for offline builds
COPY third_party/ /third_party/

# 创建输出目录
RUN mkdir -p /out /build

# 下载函数：支持 GITHUB_MIRROR、GITHUB_PROXY 和直接下载
# 下载并重新打包预编译的 Singularity
RUN set -eux; \
    if [ "${BUILD_SINGULARITY}" = "true" ]; then \
        echo "📦 Downloading pre-built Singularity ${SINGULARITY_VERSION}..."; \
        echo "   GITHUB_MIRROR: ${GITHUB_MIRROR:-<disabled>}"; \
        echo "   GITHUB_PROXY:  ${GITHUB_PROXY:-<disabled>}"; \
        mkdir -p /out; \
        VERSION_NUM=$(echo ${SINGULARITY_VERSION} | sed 's/^v//'); \
        GITHUB_DIRECT_URL="https://github.com/sylabs/singularity/releases/download/v${VERSION_NUM}"; \
        # 通用下载函数
        download_file() { \
            local filename=$1; \
            local output_path="/out/${filename}"; \
            local direct_url="${GITHUB_DIRECT_URL}/${filename}"; \
            local download_success=false; \
            # 方式0: 检查本地文件
            if [ -f "/third_party/singularity/${filename}" ]; then \
                echo "  📦 [本地] Using local file: ${filename}"; \
                cp "/third_party/singularity/${filename}" "${output_path}"; \
                return 0; \
            fi; \
            # 方式1: GITHUB_MIRROR
            if [ -n "${GITHUB_MIRROR:-}" ]; then \
                local mirror_url="${GITHUB_MIRROR}github.com/sylabs/singularity/releases/download/v${VERSION_NUM}/${filename}"; \
                echo "  📥 [方式1] GITHUB_MIRROR: ${mirror_url}"; \
                if curl -fsSL --connect-timeout 30 --max-time 300 -o "${output_path}" "${mirror_url}" 2>/dev/null; then \
                    echo "  ✓ Downloaded ${filename} via GITHUB_MIRROR"; \
                    return 0; \
                fi; \
                echo "  ⚠️  GITHUB_MIRROR failed"; \
            fi; \
            # 方式2: GITHUB_PROXY
            if [ -n "${GITHUB_PROXY:-}" ]; then \
                echo "  📥 [方式2] GITHUB_PROXY: ${direct_url}"; \
                if curl --proxy "${GITHUB_PROXY}" -fsSL --connect-timeout 30 --max-time 300 -o "${output_path}" "${direct_url}" 2>/dev/null; then \
                    echo "  ✓ Downloaded ${filename} via GITHUB_PROXY"; \
                    return 0; \
                fi; \
                echo "  ⚠️  GITHUB_PROXY failed"; \
            fi; \
            # 方式3: 直接下载
            echo "  📥 [方式3] Direct: ${direct_url}"; \
            if curl -fsSL --connect-timeout 30 --max-time 300 -o "${output_path}" "${direct_url}"; then \
                echo "  ✓ Downloaded ${filename} directly"; \
                return 0; \
            fi; \
            echo "  ❌ All download methods failed for ${filename}"; \
            rm -f "${output_path}"; \
            return 1; \
        }; \
        # Download Ubuntu 22.04 DEB packages (amd64 & arm64)
        for ARCH in amd64 arm64; do \
            DEB_FILE="singularity-ce_${VERSION_NUM}-1~ubuntu22.04_${ARCH}.deb"; \
            echo "Processing ${DEB_FILE}..."; \
            download_file "${DEB_FILE}" || echo "⚠️  Skipping ${DEB_FILE}"; \
        done; \
        # Download EL9 RPM packages (x86_64 & aarch64)
        for ARCH in x86_64 aarch64; do \
            RPM_FILE="singularity-ce-${VERSION_NUM}-1.el9.${ARCH}.rpm"; \
            echo "Processing ${RPM_FILE}..."; \
            download_file "${RPM_FILE}" || echo "⚠️  Skipping ${RPM_FILE}"; \
        done; \
        echo "✅ Singularity packages processed"; \
        ls -lh /out/; \
    else \
        echo "⏭️  Skipping Singularity (BUILD_SINGULARITY=${BUILD_SINGULARITY})"; \
        mkdir -p /out; \
    fi

# =============================================================================
# Stage 5: AppHub - HTTP Server with Package Management & Development Tools
# =============================================================================
FROM ubuntu:${UBUNTU_VERSION}

# 版本元数据 ARG（需要在 FROM 后重新声明）
ARG SLURM_VERSION=25.05.4
ARG SALTSTACK_VERSION=v3007.8
ARG CATEGRAF_VERSION=v0.4.25
ARG APPHUB_BASE_URL=http://localhost:8081
ARG APT_MIRROR=mirrors.aliyun.com
# 将版本保存到环境变量（可在运行时访问）
ENV SLURM_VERSION=${SLURM_VERSION}
ENV SALTSTACK_VERSION=${SALTSTACK_VERSION}
ENV CATEGRAF_VERSION=${CATEGRAF_VERSION}
ENV APPHUB_BASE_URL=${APPHUB_BASE_URL}
ENV DEBIAN_FRONTEND=noninteractive

# 配置 APT 镜像源并安装基础工具
RUN set -eux; \
    # 备份原始源配置
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构并配置镜像源
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "Using custom APT mirror: ${APT_MIRROR}"; \
        sed -i "s|archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
        sed -i "s|ports.ubuntu.com/ubuntu-ports/|${APT_MIRROR}/ubuntu-ports/|g" /etc/apt/sources.list; \
    fi; \
    # 更新包列表
    apt-get update; \
    # 安装工具
    apt-get install -y --no-install-recommends \
        nginx \
        dpkg-dev \
        zstd \
        createrepo-c \
        build-essential \
        git \
        vim \
        wget \
        curl \
        bash \
        ca-certificates \
        gzip \
        perl \
        openssh-server \
        net-tools \
        iputils-ping \
        procps \
        python3 \
        python3-pip && \
    rm -rf /var/lib/apt/lists/*

# Configure SSH server for backend access (仅配置公钥认证，无密码登录)
RUN set -eux; \
    # 创建SSH目录
    mkdir -p /root/.ssh /var/run/sshd; \
    chmod 700 /root/.ssh; \
    # 配置SSH服务器（安全配置）
    sed -i 's/#PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config; \
    sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config; \
    sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config; \
    # 生成SSH host keys
    ssh-keygen -A; \
    echo "✓ SSH server configured (public key authentication only)"

# Copy shared public key from project root (统一密钥管理)
# AppHub只需要公钥，用于接受backend的SSH连接
# Note: SSH密钥会在构建前由build.sh从项目根目录同步到此处
COPY ssh-key/id_rsa.pub /root/.ssh/authorized_keys
RUN chmod 600 /root/.ssh/authorized_keys && \
    echo "✓ SSH public key installed for backend access"

# Copy nginx config
COPY src/apphub/nginx.conf /etc/nginx/nginx.conf

# Copy custom index.html (AppHub homepage)
COPY src/apphub/html/index.html /usr/share/nginx/html/index.html

# Create directories for packages
RUN mkdir -p \
    /usr/share/nginx/html/deb \
    /usr/share/nginx/html/rpm \
    /usr/share/nginx/html/pkgs/slurm-deb \
    /usr/share/nginx/html/pkgs/slurm-rpm \
    /usr/share/nginx/html/pkgs/slurm-binaries \
    /usr/share/nginx/html/pkgs/slurm-plugins \
    /usr/share/nginx/html/pkgs/saltstack-deb \
    /usr/share/nginx/html/pkgs/saltstack-rpm \
    /usr/share/nginx/html/pkgs/categraf

# Copy all deb packages from deb-builder stage (SLURM + SaltStack)
COPY --from=deb-builder /out/ /usr/share/nginx/html/pkgs/slurm-deb/

# Copy SLURM rpm packages with metadata from rpm-builder stage
COPY --from=rpm-builder /out/slurm-rpm/ /usr/share/nginx/html/pkgs/slurm-rpm/

# Extract cgroup_v2.so plugin from DEB packages for Rocky nodes
RUN set -eux; \
    echo "📦 Extracting cgroup_v2.so plugin for Rocky nodes..."; \
    cd /usr/share/nginx/html/pkgs/slurm-deb; \
    # Prefer slurm-smd-slurmd package; fall back to base slurm-smd when needed
    DEB_FILE=$(ls -1 slurm-smd-slurmd_*.deb 2>/dev/null | head -1); \
    if [ -z "$DEB_FILE" ]; then \
        DEB_FILE=$(ls -1 slurm-*_${ARCH}.deb 2>/dev/null | head -1); \
    fi; \
    if [ -n "$DEB_FILE" ]; then \
        dpkg-deb -x "$DEB_FILE" /tmp/slurm-extract; \
        # Find and copy cgroup_v2.so
        find /tmp/slurm-extract -name "cgroup_v2.so" -exec cp {} /usr/share/nginx/html/pkgs/slurm-plugins/ \;; \
        rm -rf /tmp/slurm-extract; \
        echo "✓ Extracted cgroup_v2.so plugin"; \
        ls -lh /usr/share/nginx/html/pkgs/slurm-plugins/; \
    else \
        echo "⚠️  Warning: slurm-smd-slurmd DEB package not found"; \
    fi

# Copy SaltStack rpm packages with metadata from rpm-builder stage
COPY --from=rpm-builder /out/saltstack-rpm/ /usr/share/nginx/html/pkgs/saltstack-rpm/

# Copy SLURM binaries from binary-builder stage
COPY --from=binary-builder /out/packages/ /usr/share/nginx/html/pkgs/slurm-binaries/

# Copy scripts for installation
COPY src/apphub/scripts/install-slurm.sh.tmpl /app/scripts/install-slurm.sh.tmpl
COPY src/apphub/scripts/generate-install-script.sh /app/scripts/generate-install-script.sh
RUN chmod +x /app/scripts/generate-install-script.sh

# Generate SLURM installation script from template

# ARG needs to be redeclared here if used in RUN after COPY
ARG SLURM_VERSION=25.05.4
ARG APPHUB_BASE_URL=http://localhost:8081
ENV SLURM_VERSION=${SLURM_VERSION}
ENV APPHUB_BASE_URL=${APPHUB_BASE_URL}
RUN /app/scripts/generate-install-script.sh \
    /app/scripts/install-slurm.sh.tmpl \
    /usr/share/nginx/html/packages/install-slurm.sh

# Copy Categraf packages from categraf-builder stage
COPY --from=categraf-builder /out/ /usr/share/nginx/html/pkgs/categraf/

# Copy Singularity packages from singularity-builder stage
COPY --from=singularity-builder /out/ /usr/share/nginx/html/pkgs/singularity/

# =============================================================================
# Download/Copy Prometheus, Node Exporter, Categraf and Python dependencies
# 监控工具包 - 直接从 third_party/ 复制预下载的文件
# 使用 scripts/download_third_party.sh 预下载所有依赖
# 构建前请运行: ./build.sh download-deps 或 ./scripts/download_third_party.sh
# =============================================================================
ARG PROMETHEUS_VERSION=v3.4.1
ARG NODE_EXPORTER_VERSION=v1.8.2
ARG ALERTMANAGER_VERSION=v0.28.1
ARG CATEGRAF_VERSION=v0.4.25
ARG GITHUB_PROXY
ARG GITHUB_MIRROR=https://ghfast.top/
ARG PYPI_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/

# Copy ALL pre-downloaded packages from third_party/ directory
# 复制所有预下载的包
COPY third_party/ /third_party/

# Copy download scripts as fallback for any missing packages
COPY src/apphub/scripts/prometheus/download-prometheus.sh /tmp/download-prometheus.sh
COPY src/apphub/scripts/node_exporter/download-node-exporter.sh /tmp/download-node-exporter.sh
COPY src/apphub/scripts/categraf/download-categraf.sh /tmp/download-categraf.sh
COPY src/apphub/scripts/categraf/install-categraf.sh /scripts/categraf/install-categraf.sh
COPY src/apphub/scripts/saltstack/download-python-deps.sh /tmp/download-python-deps.sh

RUN set -eux; \
    chmod +x /tmp/download-prometheus.sh /tmp/download-node-exporter.sh /tmp/download-categraf.sh /tmp/download-python-deps.sh; \
    mkdir -p /scripts/categraf; \
    mkdir -p /usr/share/nginx/html/pkgs/prometheus; \
    mkdir -p /usr/share/nginx/html/pkgs/node_exporter; \
    mkdir -p /usr/share/nginx/html/pkgs/alertmanager; \
    mkdir -p /usr/share/nginx/html/pkgs/categraf; \
    mkdir -p /usr/share/nginx/html/pkgs/python-deps; \
    \
    # 去掉版本号前的 v 前缀 (Prometheus 文件名不带 v)
    PROMETHEUS_VER="${PROMETHEUS_VERSION#v}"; \
    NODE_EXPORTER_VER="${NODE_EXPORTER_VERSION#v}"; \
    ALERTMANAGER_VER="${ALERTMANAGER_VERSION#v}"; \
    \
    # === Prometheus ===
    echo "📦 Processing Prometheus..."; \
    prometheus_copied=0; \
    for arch in amd64 arm64; do \
        src_file="/third_party/prometheus/prometheus-${PROMETHEUS_VER}.linux-${arch}.tar.gz"; \
        dst_file="/usr/share/nginx/html/pkgs/prometheus/prometheus-${PROMETHEUS_VER}.linux-${arch}.tar.gz"; \
        if [ -f "$src_file" ] && [ -s "$src_file" ]; then \
            cp "$src_file" "$dst_file"; \
            echo "  ✓ Copied prometheus-${PROMETHEUS_VER}.linux-${arch}.tar.gz from third_party"; \
            prometheus_copied=$((prometheus_copied + 1)); \
        fi; \
    done; \
    if [ "$prometheus_copied" -lt 2 ]; then \
        echo "  ⚠ Pre-downloaded files not found or incomplete, downloading..."; \
        PROMETHEUS_VERSION=${PROMETHEUS_VERSION} \
        GITHUB_MIRROR=${GITHUB_MIRROR} \
        GITHUB_PROXY=${GITHUB_PROXY:-} \
        OUTPUT_DIR=/usr/share/nginx/html/pkgs/prometheus \
        /tmp/download-prometheus.sh || echo "⚠️ Prometheus download failed, continuing..."; \
    fi; \
    \
    # === Node Exporter ===
    echo "📦 Processing Node Exporter..."; \
    node_exporter_copied=0; \
    for arch in amd64 arm64; do \
        src_file="/third_party/node_exporter/node_exporter-${NODE_EXPORTER_VER}.linux-${arch}.tar.gz"; \
        dst_file="/usr/share/nginx/html/pkgs/node_exporter/node_exporter-${NODE_EXPORTER_VER}.linux-${arch}.tar.gz"; \
        if [ -f "$src_file" ] && [ -s "$src_file" ]; then \
            cp "$src_file" "$dst_file"; \
            echo "  ✓ Copied node_exporter-${NODE_EXPORTER_VER}.linux-${arch}.tar.gz from third_party"; \
            node_exporter_copied=$((node_exporter_copied + 1)); \
        fi; \
    done; \
    if [ "$node_exporter_copied" -lt 2 ]; then \
        echo "  ⚠ Pre-downloaded files not found or incomplete, downloading..."; \
        NODE_EXPORTER_VERSION=${NODE_EXPORTER_VERSION} \
        GITHUB_MIRROR=${GITHUB_MIRROR} \
        GITHUB_PROXY=${GITHUB_PROXY:-} \
        OUTPUT_DIR=/usr/share/nginx/html/pkgs/node_exporter \
        /tmp/download-node-exporter.sh || echo "⚠️ Node Exporter download failed, continuing..."; \
    fi; \
    \
    # === Alertmanager ===
    echo "📦 Processing Alertmanager..."; \
    alertmanager_copied=0; \
    for arch in amd64 arm64; do \
        src_file="/third_party/alertmanager/alertmanager-${ALERTMANAGER_VER}.linux-${arch}.tar.gz"; \
        dst_file="/usr/share/nginx/html/pkgs/alertmanager/alertmanager-${ALERTMANAGER_VER}.linux-${arch}.tar.gz"; \
        if [ -f "$src_file" ] && [ -s "$src_file" ]; then \
            cp "$src_file" "$dst_file"; \
            echo "  ✓ Copied alertmanager-${ALERTMANAGER_VER}.linux-${arch}.tar.gz from third_party"; \
            alertmanager_copied=$((alertmanager_copied + 1)); \
        fi; \
    done; \
    if [ "$alertmanager_copied" -lt 2 ]; then \
        echo "  ⚠ Alertmanager files not found in third_party (optional, skipping)"; \
    fi; \
    \
    # === Categraf ===
    echo "📦 Processing Categraf..."; \
    categraf_copied=0; \
    for arch in amd64 arm64; do \
        src_file="/third_party/categraf/categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz"; \
        dst_file="/usr/share/nginx/html/pkgs/categraf/categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz"; \
        if [ -f "$src_file" ] && [ -s "$src_file" ]; then \
            cp "$src_file" "$dst_file"; \
            echo "  ✓ Copied categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz from third_party"; \
            categraf_copied=$((categraf_copied + 1)); \
        fi; \
    done; \
    if [ "$categraf_copied" -lt 2 ]; then \
        echo "  ⚠ Pre-downloaded files not found or incomplete, downloading..."; \
        CATEGRAF_VERSION=${CATEGRAF_VERSION} \
        GITHUB_MIRROR=${GITHUB_MIRROR} \
        GITHUB_PROXY=${GITHUB_PROXY:-} \
        OUTPUT_DIR=/usr/share/nginx/html/pkgs/categraf \
        /tmp/download-categraf.sh || echo "⚠️ Categraf download failed, continuing..."; \
    fi; \
    \
    # === SaltStack packages (DEB) ===
    echo "📦 Processing SaltStack DEB packages..."; \
    saltstack_deb_copied=0; \
    if [ -d "/third_party/saltstack" ]; then \
        for deb_file in /third_party/saltstack/*.deb; do \
            if [ -f "$deb_file" ] && [ -s "$deb_file" ]; then \
                cp "$deb_file" /usr/share/nginx/html/pkgs/saltstack-deb/; \
                saltstack_deb_copied=$((saltstack_deb_copied + 1)); \
            fi; \
        done; \
        echo "  ✓ Copied ${saltstack_deb_copied} SaltStack DEB packages from third_party"; \
    fi; \
    \
    # === SaltStack packages (RPM) ===
    echo "📦 Processing SaltStack RPM packages..."; \
    saltstack_rpm_copied=0; \
    if [ -d "/third_party/saltstack" ]; then \
        for rpm_file in /third_party/saltstack/*.rpm; do \
            if [ -f "$rpm_file" ] && [ -s "$rpm_file" ]; then \
                cp "$rpm_file" /usr/share/nginx/html/pkgs/saltstack-rpm/; \
                saltstack_rpm_copied=$((saltstack_rpm_copied + 1)); \
            fi; \
        done; \
        echo "  ✓ Copied ${saltstack_rpm_copied} SaltStack RPM packages from third_party"; \
    fi; \
    \
    # === Singularity packages ===
    echo "📦 Processing Singularity packages..."; \
    singularity_copied=0; \
    if [ -d "/third_party/singularity" ]; then \
        mkdir -p /usr/share/nginx/html/pkgs/singularity; \
        for pkg_file in /third_party/singularity/*.deb /third_party/singularity/*.rpm; do \
            if [ -f "$pkg_file" ] && [ -s "$pkg_file" ]; then \
                cp "$pkg_file" /usr/share/nginx/html/pkgs/singularity/; \
                singularity_copied=$((singularity_copied + 1)); \
            fi; \
        done; \
        echo "  ✓ Copied ${singularity_copied} Singularity packages from third_party"; \
    fi; \
    \
    # === Munge source ===
    echo "📦 Processing Munge..."; \
    if [ -d "/third_party/munge" ]; then \
        mkdir -p /usr/share/nginx/html/pkgs/munge; \
        cp /third_party/munge/*.tar.xz /usr/share/nginx/html/pkgs/munge/ 2>/dev/null || true; \
        echo "  ✓ Copied Munge source from third_party"; \
    fi; \
    \
    # === Python dependencies for SaltStack (looseversion for Python 3.12+) ===
    echo "📦 Processing Python dependencies..."; \
    PYPI_MIRROR=${PYPI_INDEX_URL:-https://pypi.org/simple} \
    PYPI_MIRROR_CN=https://mirrors.aliyun.com/pypi/simple \
    OUTPUT_DIR=/usr/share/nginx/html/pkgs/python-deps \
    /tmp/download-python-deps.sh || echo "⚠️ Python deps download failed, continuing..."; \
    \
    # Cleanup
    rm -f /tmp/download-prometheus.sh /tmp/download-node-exporter.sh /tmp/download-categraf.sh /tmp/download-python-deps.sh; \
    rm -rf /third_party; \
    \
    # Summary
    prometheus_count=$(ls -1 /usr/share/nginx/html/pkgs/prometheus/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    node_exporter_count=$(ls -1 /usr/share/nginx/html/pkgs/node_exporter/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    alertmanager_count=$(ls -1 /usr/share/nginx/html/pkgs/alertmanager/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    categraf_count=$(ls -1 /usr/share/nginx/html/pkgs/categraf/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    python_deps_count=$(ls -1 /usr/share/nginx/html/pkgs/python-deps/*.whl /usr/share/nginx/html/pkgs/python-deps/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    saltstack_deb_final=$(ls -1 /usr/share/nginx/html/pkgs/saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0); \
    saltstack_rpm_final=$(ls -1 /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
    echo ""; \
    echo "📊 Package Summary (from third_party/):"; \
    echo "  - Prometheus packages: ${prometheus_count}"; \
    echo "  - Node Exporter packages: ${node_exporter_count}"; \
    echo "  - Alertmanager packages: ${alertmanager_count}"; \
    echo "  - Categraf packages: ${categraf_count}"; \
    echo "  - SaltStack DEB packages: ${saltstack_deb_final}"; \
    echo "  - SaltStack RPM packages: ${saltstack_rpm_final}"; \
    echo "  - Python dependencies: ${python_deps_count}"

# Organize packages and generate DEB indexes (RPM metadata already generated in rpm-builder stage)
RUN set -eux; \
    echo "📦 Organizing packages..."; \
    # Separate SLURM and SaltStack deb packages
    mkdir -p /usr/share/nginx/html/pkgs/saltstack-deb; \
    if [ -d /usr/share/nginx/html/pkgs/slurm-deb ]; then \
        cd /usr/share/nginx/html/pkgs/slurm-deb; \
        # Move SaltStack packages to separate directory
        find . -name "salt-*.deb" -exec mv {} /usr/share/nginx/html/pkgs/saltstack-deb/ \; 2>/dev/null || true; \
    fi; \
    # Count packages
    slurm_deb_count=$(ls -1 /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l || echo 0); \
    slurm_rpm_count=$(ls -1 /usr/share/nginx/html/pkgs/slurm-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
    slurm_bin_count=$(find /usr/share/nginx/html/pkgs/slurm-binaries -type f -name "s*" 2>/dev/null | wc -l || echo 0); \
    salt_deb_count=$(ls -1 /usr/share/nginx/html/pkgs/saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0); \
    salt_rpm_count=$(ls -1 /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0); \
    categraf_count=$(ls -1 /usr/share/nginx/html/pkgs/categraf/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    singularity_count=$(ls -1 /usr/share/nginx/html/pkgs/singularity/ | grep -E '\.(deb|rpm)$' | wc -l || echo 0); \
    prometheus_count=$(ls -1 /usr/share/nginx/html/pkgs/prometheus/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    node_exporter_count=$(ls -1 /usr/share/nginx/html/pkgs/node_exporter/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    alertmanager_count=$(ls -1 /usr/share/nginx/html/pkgs/alertmanager/*.tar.gz 2>/dev/null | wc -l || echo 0); \
    # Check if RPM metadata was copied from rpm-builder
    slurm_rpm_metadata=$([ -d /usr/share/nginx/html/pkgs/slurm-rpm/repodata ] && echo "yes" || echo "no"); \
    salt_rpm_metadata=$([ -d /usr/share/nginx/html/pkgs/saltstack-rpm/repodata ] && echo "yes" || echo "no"); \
    echo "📊 Package Summary:"; \
    echo "  - SLURM deb packages: ${slurm_deb_count}"; \
    echo "  - SLURM rpm packages: ${slurm_rpm_count} (metadata: ${slurm_rpm_metadata})"; \
    echo "  - SLURM binaries: ${slurm_bin_count}"; \
    echo "  - SaltStack deb packages: ${salt_deb_count}"; \
    echo "  - SaltStack rpm packages: ${salt_rpm_count} (metadata: ${salt_rpm_metadata})"; \
    echo "  - Categraf packages: ${categraf_count}"; \
    echo "  - Singularity packages: ${singularity_count}"; \
    echo "  - Prometheus packages: ${prometheus_count}"; \
    echo "  - Node Exporter packages: ${node_exporter_count}"; \
    echo "  - Alertmanager packages: ${alertmanager_count}"; \
    echo "  - Prometheus packages: ${prometheus_count}"; \
    echo "  - Node Exporter packages: ${node_exporter_count}"; \
    # Check if any packages are missing
    if [ "$slurm_deb_count" -eq 0 ] && [ "$salt_deb_count" -eq 0 ] && [ "$slurm_rpm_count" -eq 0 ] && [ "$salt_rpm_count" -eq 0 ]; then \
        echo "❌ Error: No packages found for SLURM or SaltStack"; \
        exit 1; \
    fi; \
    # Generate Debian repository metadata (both Packages and Packages.gz)
    if [ "$slurm_deb_count" -gt 0 ] && command -v dpkg-scanpackages >/dev/null 2>&1; then \
        echo "🔧 Generating SLURM DEB repository metadata..."; \
        cd /usr/share/nginx/html/pkgs/slurm-deb && \
        dpkg-scanpackages -m . > Packages && \
        gzip -k -f Packages; \
        echo "✓ Generated SLURM DEB repository metadata (Packages, Packages.gz)"; \
    elif [ "$slurm_deb_count" -gt 0 ]; then \
        echo "⚠️  dpkg-scanpackages not available, SLURM DEB packages available for direct download only"; \
    fi; \
    if [ "$salt_deb_count" -gt 0 ] && command -v dpkg-scanpackages >/dev/null 2>&1; then \
        echo "🔧 Generating SaltStack DEB repository metadata..."; \
        cd /usr/share/nginx/html/pkgs/saltstack-deb && \
        dpkg-scanpackages -m . > Packages && \
        gzip -k -f Packages; \
        echo "✓ Generated SaltStack DEB repository metadata (Packages, Packages.gz)"; \
    elif [ "$salt_deb_count" -gt 0 ]; then \
        echo "⚠️  dpkg-scanpackages not available, SaltStack DEB packages available for direct download only"; \
    fi; \
    # General deb index (if any)
    if [ -d /usr/share/nginx/html/deb ] && [ "$(ls -A /usr/share/nginx/html/deb 2>/dev/null)" ]; then \
        if command -v dpkg-scanpackages >/dev/null 2>&1; then \
            cd /usr/share/nginx/html/deb && \
            dpkg-scanpackages -m . > Packages && \
            gzip -k -f Packages; \
            echo "✓ Generated general deb package index"; \
        fi; \
    fi; \
    # RPM metadata was already generated in rpm-builder stage (Rocky Linux has createrepo)
    # Verify it was copied correctly
    if [ "$slurm_rpm_count" -gt 0 ]; then \
        if [ -d /usr/share/nginx/html/pkgs/slurm-rpm/repodata ]; then \
            echo "✓ SLURM RPM repository metadata available (repodata/)"; \
        else \
            echo "⚠️  SLURM RPM metadata missing - packages available for direct download only"; \
        fi; \
    fi; \
    if [ "$salt_rpm_count" -gt 0 ]; then \
        if [ -d /usr/share/nginx/html/pkgs/saltstack-rpm/repodata ]; then \
            echo "✓ SaltStack RPM repository metadata available (repodata/)"; \
            ls -la /usr/share/nginx/html/pkgs/saltstack-rpm/repodata/ || true; \
        else \
            echo "⚠️  SaltStack RPM metadata missing - packages available for direct download only"; \
        fi; \
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
        # Remove existing latest symlinks to avoid self-reference
        rm -f categraf-latest-linux-amd64.tar.gz categraf-latest-linux-arm64.tar.gz 2>/dev/null || true; \
        # Create latest symlink for amd64 (exclude 'latest' from search)
        latest_amd64=$(ls -t categraf-v*-linux-amd64.tar.gz 2>/dev/null | head -1); \
        if [ -n "$latest_amd64" ]; then \
            ln -sf "$latest_amd64" categraf-latest-linux-amd64.tar.gz; \
            echo "  ✓ Created symlink: categraf-latest-linux-amd64.tar.gz -> $latest_amd64"; \
        fi; \
        # Create latest symlink for arm64 (exclude 'latest' from search)
        latest_arm64=$(ls -t categraf-v*-linux-arm64.tar.gz 2>/dev/null | head -1); \
        if [ -n "$latest_arm64" ]; then \
            ln -sf "$latest_arm64" categraf-latest-linux-arm64.tar.gz; \
            echo "  ✓ Created symlink: categraf-latest-linux-arm64.tar.gz -> $latest_arm64"; \
        fi; \
    fi; \
    # Create symlinks in top-level directories for easier access
    echo "🔗 Creating top-level package directory symlinks..."; \
    # Link SLURM deb packages to /usr/share/nginx/html/deb/
    if [ "$slurm_deb_count" -gt 0 ]; then \
        ln -sf ../pkgs/slurm-deb/* /usr/share/nginx/html/deb/ 2>/dev/null || true; \
        echo "  ✓ Linked SLURM deb packages to /deb/"; \
    fi; \
    # Link SLURM rpm packages to /usr/share/nginx/html/rpm/
    if [ "$slurm_rpm_count" -gt 0 ]; then \
        ln -sf ../pkgs/slurm-rpm/* /usr/share/nginx/html/rpm/ 2>/dev/null || true; \
        echo "  ✓ Linked SLURM rpm packages to /rpm/"; \
    fi; \
    # Note about RPM metadata
    if [ "$slurm_rpm_count" -gt 0 ] || [ "$salt_rpm_count" -gt 0 ]; then \
        echo "⚠️  Note: YUM/DNF metadata generation skipped (can be enabled if createrepo is installed)"; \
        echo "⚠️  Packages can be downloaded directly via HTTP"; \
    fi

# Copy installation scripts (rendered from templates by build.sh render)
# These scripts are pre-configured with EXTERNAL_HOST and other settings
COPY scripts/install-salt-minion.sh /usr/share/nginx/html/scripts/install-salt-minion.sh
COPY scripts/install-categraf.sh /usr/share/nginx/html/scripts/install-categraf.sh
COPY scripts/install-node-exporter.sh /usr/share/nginx/html/scripts/install-node-exporter.sh
COPY scripts/install-prometheus.sh /usr/share/nginx/html/scripts/install-prometheus.sh
RUN chmod +x /usr/share/nginx/html/scripts/*.sh && \
    echo "✓ Installation scripts copied to /scripts/"

# Expose port
EXPOSE 80

# Entrypoint to regenerate indexes if needed
COPY src/apphub/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]