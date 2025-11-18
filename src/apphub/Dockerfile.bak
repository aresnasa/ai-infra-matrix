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
ARG ROCKYLINUX_VERSION=9
ARG GOLANG_ALPINE_VERSION=1.25-alpine
ARG NGINX_ALPINE_VERSION=1.27-alpine

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

# Package cache optimization - specify a previous image to copy packages from
# 包缓存优化 - 指定一个先前的镜像来复制包，避免重复下载
ARG CACHE_IMAGE=""

# SLURM version configuration - update these when upgrading
# 更新 SLURM 版本时只需修改这两个变量
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_NAME=slurm-${SLURM_VERSION}.tar.bz2

# SaltStack version configuration
ARG SALTSTACK_VERSION=v3007.8

# Categraf version configuration
ARG CATEGRAF_VERSION=v0.4.25

# Singularity version configuration
ARG SINGULARITY_VERSION=v4.3.4

# Accept optional tarball path relative to build context root
# 默认在当前目录查找 tarball
ARG SLURM_TARBALL_PATH=${SLURM_TARBALL_NAME}

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
    if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
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
        # Use DEB_BUILD_OPTIONS to pass configuration (skip tests, minimal build)
        # Note: SLURM's debian/rules may not honor --without-cgroup, but we document the intent
        export DEB_BUILD_OPTIONS="nocheck parallel=$(nproc)"; \
        dpkg-buildpackage -b -uc; \
    fi

# Download SaltStack packages from GitHub releases
USER root
ARG BUILD_SALTSTACK
ARG SALTSTACK_VERSION
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
        BASE_URL="https://github.com/saltstack/salt/releases/download/${RELEASE_TAG}"; \
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
# Stage 2: Build SLURM rpm packages (Rocky Linux 9)
# =============================================================================
FROM rockylinux:${ROCKYLINUX_VERSION} AS rpm-builder

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

# 配置 Rocky Linux 镜像源（可选，如果网络不好则跳过）
# 注意：也可以通过 docker build --build-arg HTTP_PROXY=... 使用代理
RUN set -eux; \
    echo "尝试配置 Rocky Linux 阿里云镜像源（可选）..."; \
    # 备份原始配置
    cp -r /etc/yum.repos.d /etc/yum.repos.d.backup 2>/dev/null || true; \
    # 尝试配置阿里云镜像源
    ( \
        sed -e 's|^mirrorlist=|#mirrorlist=|g' \
            -e 's|^#baseurl=http://dl.rockylinux.org/\$contentdir|baseurl=http://mirrors.aliyun.com/rockylinux|g' \
            -i.bak \
            /etc/yum.repos.d/rocky*.repo 2>/dev/null && \
        dnf clean all 2>/dev/null && \
        dnf makecache 2>/dev/null && \
        echo "✓ 成功配置阿里云镜像源" \
    ) || { \
        echo "⚠️ 镜像源配置失败，使用默认配置"; \
        cp -r /etc/yum.repos.d.backup/* /etc/yum.repos.d/ 2>/dev/null || true; \
        dnf clean all 2>/dev/null || true; \
    }

# Install build prerequisites and enable required repositories
RUN set -eux; \
    # Enable PowerTools/CRB repository for additional development packages
    dnf config-manager --set-enabled crb 2>/dev/null || \
    dnf config-manager --set-enabled powertools 2>/dev/null || \
    echo "PowerTools/CRB repository not available"; \
    dnf install -y epel-release || echo "EPEL repository not available"; \
    # 只更新元数据缓存，不更新所有包（避免网络问题）
    dnf makecache --refresh || dnf makecache || true; \
    # Install basic build dependencies first
    echo "📦 Installing RPM build tools..."; \
    dnf install -y \
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
        openssl-devel; \
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
        MUNGE_VERSION="0.5.16"; \
        wget "https://github.com/dun/munge/releases/download/munge-${MUNGE_VERSION}/munge-${MUNGE_VERSION}.tar.xz" -O munge.tar.xz; \
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
        # Note: Disable cgroup by default - let SLURM configuration (not compile-time) manage it
        echo '%_topdir %(echo $HOME)/rpmbuild' > ~/.rpmmacros; \
        echo '%_prefix /usr' >> ~/.rpmmacros; \
        echo '%_slurm_sysconfdir %{_prefix}/etc/slurm' >> ~/.rpmmacros; \
        echo '%with_munge --with-munge' >> ~/.rpmmacros; \
        echo '%without_cgroup --without-cgroup' >> ~/.rpmmacros; \
        echo "✓ Created ~/.rpmmacros configuration"; \
        cat ~/.rpmmacros; \
        echo "Note: cgroup support disabled at build time - use system defaults"; \
        # Build RPMs directly using rpmbuild -ta (recommended by official docs)
        echo ">>> Building SLURM RPM packages using 'rpmbuild -ta' (this may take 10-15 minutes)..."; \
        echo ">>> Command: rpmbuild -ta --nodeps ${tarball}"; \
        echo ">>> Note: Using --nodeps because we installed munge from source (not RPM package)"; \
        if ! rpmbuild -ta --nodeps "${tarball}" 2>&1 | tee /tmp/rpmbuild.log; then \
            echo "❌ RPM build failed! Last 100 lines of output:"; \
            tail -100 /tmp/rpmbuild.log; \
            echo ""; \
            echo ">>> Checking rpmbuild directory structure:"; \
            ls -laR ~/rpmbuild/ | head -50 || true; \
            exit 1; \
        fi; \
        echo "✓ SLURM RPM build completed successfully"; \
        echo ">>> Listing generated RPM packages:"; \
        find ~/rpmbuild/RPMS -name "*.rpm" -type f 2>/dev/null || echo "No RPMs found"; \
    else \
        echo "🚫 Skipping SLURM RPM build (BUILD_SLURM=false)"; \
        mkdir -p /home/builder/rpms; \
        touch /home/builder/rpms/.skip_slurm; \
    fi

# Download SaltStack packages from GitHub releases
USER root
ARG BUILD_SALTSTACK
ARG SALTSTACK_VERSION
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
        BASE_URL="https://github.com/saltstack/salt/releases/download/${RELEASE_TAG}"; \
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
            cd /out/slurm-rpm && createrepo_c . && \
            echo "✓ Generated SLURM repodata: $(ls -d repodata 2>/dev/null || echo 'failed')"; \
        elif [ "$slurm_count" -gt 0 ] && command -v createrepo >/dev/null 2>&1; then \
            echo "🔧 Generating SLURM RPM repository metadata (using createrepo)..."; \
            cd /out/slurm-rpm && createrepo . && \
            echo "✓ Generated SLURM repodata"; \
        fi; \
        # Generate metadata for SaltStack repository
        if [ "$salt_count" -gt 0 ] && command -v createrepo_c >/dev/null 2>&1; then \
            echo "🔧 Generating SaltStack RPM repository metadata..."; \
            cd /out/saltstack-rpm && createrepo_c . && \
            echo "✓ Generated SaltStack repodata: $(ls -d repodata 2>/dev/null || echo 'failed')"; \
        elif [ "$salt_count" -gt 0 ] && command -v createrepo >/dev/null 2>&1; then \
            echo "🔧 Generating SaltStack RPM repository metadata (using createrepo)..."; \
            cd /out/saltstack-rpm && createrepo . && \
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

# SLURM version configuration
ARG SLURM_VERSION=25.05.4

# 配置APT镜像源并安装提取工具
RUN set -eux; \
    # 备份原始源配置
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构并配置镜像源
    ARCH=$(dpkg --print-architecture); \
    echo "Detected architecture: ${ARCH}"; \
    if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
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
# Stage 4: Build Categraf (Multi-Architecture Go Binary)
# 泛化的应用构建阶段 - 只需复制 scripts/categraf/ 目录即可
# =============================================================================
FROM golang:${GOLANG_ALPINE_VERSION} AS categraf-builder

# Build control flags
ARG BUILD_CATEGRAF=true

# Categraf version configuration
ARG CATEGRAF_VERSION=v0.4.25
ARG CATEGRAF_REPO=https://github.com/flashcatcloud/categraf.git

# 配置 Go 代理（中国镜像加速）
ARG GO_PROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GO_PROXY}
ENV GO111MODULE=on
ENV CGO_ENABLED=0

# 配置 Alpine 镜像源并安装依赖（合并到一个RUN减少层数）
RUN set -eux; \
    # 备份并配置镜像源
    cp /etc/apk/repositories /etc/apk/repositories.bak || true; \
    ARCH=$(uname -m); \
    echo "Detected architecture: ${ARCH}"; \
    # 根据架构配置镜像源
    if [ "${ARCH}" = "aarch64" ]; then \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.20/community" >> /etc/apk/repositories; \
    else \
        echo "https://mirrors.aliyun.com/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v3.20/community" >> /etc/apk/repositories; \
    fi; \
    # 更新包索引（带重试和备用源）
    apk update || { \
        echo "主镜像源失败，切换到官方源..."; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/community" >> /etc/apk/repositories; \
        apk update; \
    } || { \
        echo "尝试使用HTTP协议..."; \
        echo "http://dl-cdn.alpinelinux.org/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "http://dl-cdn.alpinelinux.org/alpine/v3.20/community" >> /etc/apk/repositories; \
        apk update; \
    }; \
    # 安装构建依赖
    apk add --no-cache git make bash tar gzip sed coreutils

# 复制通用构建脚本和应用特定脚本
COPY scripts/build-app.sh /scripts/build-app.sh
COPY scripts/categraf/ /scripts/categraf/
RUN chmod +x /scripts/build-app.sh /scripts/categraf/*.sh

# 创建输出目录
RUN mkdir -p /out

# 执行构建（使用通用构建脚本）
ARG BUILD_CATEGRAF
ARG GITHUB_PROXY
RUN set -eux; \
    if [ "${BUILD_CATEGRAF}" = "true" ]; then \
        # 配置代理（如果提供）
        if [ -n "${GITHUB_PROXY:-}" ]; then \
            echo "🌐 Using proxy for Categraf build: ${GITHUB_PROXY}"; \
            export ALL_PROXY="${GITHUB_PROXY}"; \
            export HTTP_PROXY="${GITHUB_PROXY}"; \
            export HTTPS_PROXY="${GITHUB_PROXY}"; \
            export http_proxy="${GITHUB_PROXY}"; \
            export https_proxy="${GITHUB_PROXY}"; \
            # 配置 git 使用代理和 SSL
            git config --global http.proxy "${GITHUB_PROXY}"; \
            git config --global https.proxy "${GITHUB_PROXY}"; \
            git config --global http.sslVerify false; \
            git config --global http.version HTTP/1.1; \
            echo "✓ Git proxy configured"; \
        else \
            echo "⚠️  No GITHUB_PROXY provided, using direct connection"; \
        fi; \
        echo "Starting Categraf build..."; \
        CATEGRAF_VERSION=${CATEGRAF_VERSION} \
        CATEGRAF_REPO=${CATEGRAF_REPO} \
        BUILD_DIR=/build \
        OUTPUT_DIR=/out \
        /scripts/build-app.sh categraf; \
    else \
        echo "⏭️  Skipping Categraf build (BUILD_CATEGRAF=${BUILD_CATEGRAF})"; \
    fi

# =============================================================================
# Stage 4.5: Download Pre-built Singularity (Container Runtime for HPC)
# =============================================================================
FROM alpine:3.20 AS singularity-builder

# Build control flags - 暂时禁用（deb 包下载问题）
ARG BUILD_SINGULARITY=false

# Singularity version configuration
ARG SINGULARITY_VERSION=v4.2.1
ARG GITHUB_PROXY

# 配置 Alpine 镜像源并安装依赖
RUN set -eux; \
    # 配置镜像源
    ARCH=$(uname -m); \
    if [ "${ARCH}" = "aarch64" ]; then \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.20/community" >> /etc/apk/repositories; \
    else \
        echo "https://mirrors.aliyun.com/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v3.20/community" >> /etc/apk/repositories; \
    fi; \
    apk update || { \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/community" >> /etc/apk/repositories; \
        apk update; \
    }; \
    # 安装工具
    apk add --no-cache curl tar gzip binutils

# 创建输出目录
RUN mkdir -p /out /build

# 下载并重新打包预编译的 Singularity
RUN set -eux; \
    if [ "${BUILD_SINGULARITY}" = "true" ]; then \
        echo "📦 Downloading pre-built Singularity ${SINGULARITY_VERSION}..."; \
        ARCH=$(uname -m); \
        RELEASE_URL="https://github.com/sylabs/singularity/releases/download/${SINGULARITY_VERSION}"; \
        VERSION_NUM=$(echo ${SINGULARITY_VERSION} | sed 's/^v//'); \
        CURL_OPTS=""; \
        if [ -n "${GITHUB_PROXY:-}" ]; then \
            echo "🌐 Using proxy: ${GITHUB_PROXY}"; \
            CURL_OPTS="--proxy ${GITHUB_PROXY}"; \
        fi; \
        cd /build; \
        if [ "${ARCH}" = "x86_64" ]; then \
            DEB_FILE="singularity-ce_${VERSION_NUM}-1~ubuntu22.04_amd64.deb"; \
        elif [ "${ARCH}" = "aarch64" ]; then \
            DEB_FILE="singularity-ce_${VERSION_NUM}-1~ubuntu22.04_arm64.deb"; \
        else \
            echo "❌ Unsupported architecture: ${ARCH}"; \
            exit 1; \
        fi; \
        echo "Downloading ${DEB_FILE}..."; \
        curl ${CURL_OPTS} -fsSL -o singularity.deb "${RELEASE_URL}/${DEB_FILE}"; \
        ar x singularity.deb; \
        tar xf data.tar.xz; \
        tar czf /out/singularity-${SINGULARITY_VERSION}-linux-${ARCH}.tar.gz usr/; \
        echo "Package: singularity" > /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "Version: ${SINGULARITY_VERSION}" >> /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "Architecture: ${ARCH}" >> /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "Build-Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "Description: Singularity Container Runtime for HPC (Pre-built)" >> /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "Homepage: https://github.com/sylabs/singularity" >> /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "License: BSD-3-Clause" >> /out/singularity-${SINGULARITY_VERSION}.info; \
        echo "✅ Singularity download and repackage completed"; \
    else \
        echo "⏭️  Skipping Singularity (BUILD_SINGULARITY=${BUILD_SINGULARITY})"; \
    fi

# =============================================================================
# Stage 5: AppHub - HTTP Server with Package Management & Development Tools
# =============================================================================
ARG NGINX_ALPINE_VERSION=1.27-alpine
FROM nginx:${NGINX_ALPINE_VERSION}

# 版本元数据 ARG（需要在 FROM 后重新声明）
ARG SLURM_VERSION=25.05.4
ARG SALTSTACK_VERSION=v3007.8
ARG CATEGRAF_VERSION=v0.4.25
ARG APPHUB_BASE_URL=http://localhost:8081

# 将版本保存到环境变量（可在运行时访问）
ENV SLURM_VERSION=${SLURM_VERSION}
ENV SALTSTACK_VERSION=${SALTSTACK_VERSION}
ENV CATEGRAF_VERSION=${CATEGRAF_VERSION}
ENV APPHUB_BASE_URL=${APPHUB_BASE_URL}

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
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.20/community" >> /etc/apk/repositories; \
    else \
        echo "配置AMD64架构的阿里云Alpine镜像源..."; \
        echo "https://mirrors.aliyun.com/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://mirrors.aliyun.com/alpine/v3.20/community" >> /etc/apk/repositories; \
    fi; \
    # 更新包索引（带重试和备用源）
    apk update || { \
        echo "主镜像源失败，切换到官方镜像源..."; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "https://dl-cdn.alpinelinux.org/alpine/v3.20/community" >> /etc/apk/repositories; \
        apk update; \
    } || { \
        echo "尝试使用HTTP协议..."; \
        echo "http://dl-cdn.alpinelinux.org/alpine/v3.20/main" > /etc/apk/repositories; \
        echo "http://dl-cdn.alpinelinux.org/alpine/v3.20/community" >> /etc/apk/repositories; \
        apk update; \
    }

# Install tools for repo management and general development
# Note: Some packages may not be available in Alpine, install what we can
RUN set -eux; \
    # Install dpkg tools first (critical for deb package indexing)
    # Also install zstd for decompressing modern deb packages (Ubuntu 22.04+ uses zstd compression)
    apk add --no-cache dpkg dpkg-dev zstd || { \
        echo "⚠️  dpkg packages not available, will skip deb indexing"; \
    }; \
    # Install createrepo_c for RPM repository metadata generation
    apk add --no-cache createrepo_c || { \
        echo "⚠️  createrepo_c not available, will skip RPM indexing"; \
    }; \
    # Install core development and network tools (including SSH server)
    apk add --no-cache \
        build-base \
        git \
        vim \
        wget \
        curl \
        bash \
        ca-certificates \
        gzip \
        perl \
        openssh-server \
        || echo "⚠️  Some packages failed to install"; \
    # Try to install optional network tools (may not be available)
    apk add --no-cache net-tools 2>/dev/null || echo "⚠️  net-tools not available"; \
    apk add --no-cache iputils 2>/dev/null || echo "⚠️  iputils not available"; \
    apk add --no-cache procps 2>/dev/null || echo "⚠️  procps not available"

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
COPY nginx.conf /etc/nginx/nginx.conf

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
        DEB_FILE=$(ls -1 slurm-smd_*.deb 2>/dev/null | head -1); \
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
COPY scripts/install-slurm.sh.tmpl /app/scripts/install-slurm.sh.tmpl
COPY scripts/generate-install-script.sh /app/scripts/generate-install-script.sh
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
    singularity_count=$(ls -1 /usr/share/nginx/html/pkgs/singularity/*.tar.gz 2>/dev/null | wc -l || echo 0); \
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
        echo "⚠️  Note: YUM/DNF metadata not generated (createrepo not available in Alpine)"; \
        echo "⚠️  Packages can be downloaded directly via HTTP"; \
    fi

# Expose port
EXPOSE 80

# Entrypoint to regenerate indexes if needed
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]