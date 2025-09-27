# Build Slurm deb packages using Ubuntu 22.04 and the local slurm source tarball
# Reference steps adapted from USTC guide: https://scc.ustc.edu.cn/hmli/doc/linux/slurm-install/slurm-install.html

FROM ubuntu:22.04 AS build
ENV DEBIAN_FRONTEND=noninteractive

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
    apt-get install -y ca-certificates curl

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

# Add a non-root builder user to satisfy debuild (avoids building as root)
RUN useradd -m -u 1000 builder
USER builder
WORKDIR /home/builder/build

# Copy and extract the Slurm source tarball
# Note: The tarball is expected to exist at ${SLURM_TARBALL_PATH} in the build context
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

# Move all debuild outputs to /out (POSIX sh compatible)
RUN set -eux; \
    find /home/builder -maxdepth 1 -type f -name '*.deb' -exec mv {} /out/ \; || true; \
    find /home/builder -maxdepth 1 -type f -name '*.ddeb' -exec mv {} /out/ \; || true; \
    find /home/builder -maxdepth 1 -type f -name '*.build*' -exec mv {} /out/ \; || true; \
    find /home/builder -maxdepth 1 -type f -name '*.changes' -exec mv {} /out/ \; || true

# Default command lists produced artifacts
CMD ["bash", "-lc", "ls -alh /out && echo '\nTip: docker cp <container>:/out . to retrieve artifacts'" ]
