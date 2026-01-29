# AI Infrastructure Matrix - SLURM Master Service
# 基于Ubuntu构建SLURM控制节点
ARG UBUNTU_VERSION={{UBUNTU_VERSION}}
FROM ubuntu:${UBUNTU_VERSION}

# Build arguments for versions
ARG SLURM_VERSION={{SLURM_VERSION}}
ARG VERSION=v0.3.8
ARG APT_MIRROR={{APT_MIRROR}}
LABEL maintainer="AI Infrastructure Team"
LABEL version="${VERSION}"
LABEL description="AI Infrastructure Matrix SLURM Master Service"
LABEL slurm.version="${SLURM_VERSION}"

# 避免交互式安装提示
ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker
ENV TZ=Asia/Shanghai

# 立即配置阿里云镜像源（在第一次 apt-get 之前）
RUN set -eux; \
    # 备份原始sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.backup; \
    # 检测架构
    ARCH=$(dpkg --print-architecture); \
    echo "🔍 检测到系统架构: ${ARCH}"; \
    # 根据架构配置镜像源（多镜像源智能回退）
    if [ -n "${APT_MIRROR:-}" ]; then \
        echo "⚙️  使用自定义镜像源: ${APT_MIRROR}"; \
        if [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
            { \
                echo "deb http://${APT_MIRROR}/ubuntu-ports/ jammy main restricted universe multiverse"; \
                echo "deb http://${APT_MIRROR}/ubuntu-ports/ jammy-security main restricted universe multiverse"; \
                echo "deb http://${APT_MIRROR}/ubuntu-ports/ jammy-updates main restricted universe multiverse"; \
                echo "deb http://${APT_MIRROR}/ubuntu-ports/ jammy-backports main restricted universe multiverse"; \
            } > /etc/apt/sources.list; \
        else \
            { \
                echo "deb http://${APT_MIRROR}/ubuntu/ jammy main restricted universe multiverse"; \
                echo "deb http://${APT_MIRROR}/ubuntu/ jammy-security main restricted universe multiverse"; \
                echo "deb http://${APT_MIRROR}/ubuntu/ jammy-updates main restricted universe multiverse"; \
                echo "deb http://${APT_MIRROR}/ubuntu/ jammy-backports main restricted universe multiverse"; \
            } > /etc/apt/sources.list; \
        fi; \
        apt-get update; \
    elif [ "${ARCH}" = "arm64" ] || [ "${ARCH}" = "aarch64" ]; then \
        echo "⚙️  配置 ARM64 镜像源（带智能回退）..."; \
        # 尝试阿里云源（静默失败检测）
        { \
            echo "# 阿里云 Ubuntu Ports 镜像源 (ARM64)"; \
            echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy main restricted universe multiverse"; \
            echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-security main restricted universe multiverse"; \
            echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-updates main restricted universe multiverse"; \
            echo "deb http://mirrors.aliyun.com/ubuntu-ports/ jammy-backports main restricted universe multiverse"; \
        } > /etc/apt/sources.list; \
        if apt-get update 2>/dev/null; then \
            echo "✅ 成功使用阿里云源"; \
        else \
            echo "⚠️  阿里云源失败，尝试清华源..."; \
            cp /etc/apt/sources.list.backup /etc/apt/sources.list; \
            { \
                echo "# 清华 Ubuntu Ports 镜像源 (ARM64)"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/ jammy main restricted universe multiverse"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/ jammy-security main restricted universe multiverse"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/ jammy-updates main restricted universe multiverse"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/ jammy-backports main restricted universe multiverse"; \
            } > /etc/apt/sources.list; \
            if apt-get update 2>/dev/null; then \
                echo "✅ 成功使用清华源"; \
            else \
                echo "⚠️  清华源失败，尝试中科大源..."; \
                cp /etc/apt/sources.list.backup /etc/apt/sources.list; \
                { \
                    echo "# 中科大 Ubuntu Ports 镜像源 (ARM64)"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu-ports/ jammy main restricted universe multiverse"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu-ports/ jammy-security main restricted universe multiverse"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu-ports/ jammy-updates main restricted universe multiverse"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu-ports/ jammy-backports main restricted universe multiverse"; \
                } > /etc/apt/sources.list; \
                if apt-get update 2>/dev/null; then \
                    echo "✅ 成功使用中科大源"; \
                else \
                    echo "⚠️  所有国内源都失败，使用官方源..."; \
                    cp /etc/apt/sources.list.backup /etc/apt/sources.list; \
                    apt-get update; \
                fi; \
            fi; \
        fi; \
    else \
        echo "⚙️  配置 AMD64 镜像源（带智能回退）..."; \
        # 尝试阿里云源（静默失败检测）
        { \
            echo "# 阿里云 Ubuntu 镜像源 (AMD64)"; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse"; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse"; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse"; \
            echo "deb http://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse"; \
        } > /etc/apt/sources.list; \
        if apt-get update 2>/dev/null; then \
            echo "✅ 成功使用阿里云源"; \
        else \
            echo "⚠️  阿里云源失败，尝试清华源..."; \
            cp /etc/apt/sources.list.backup /etc/apt/sources.list; \
            { \
                echo "# 清华 Ubuntu 镜像源 (AMD64)"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu/ jammy main restricted universe multiverse"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu/ jammy-security main restricted universe multiverse"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu/ jammy-updates main restricted universe multiverse"; \
                echo "deb http://mirrors.tuna.tsinghua.edu.cn/ubuntu/ jammy-backports main restricted universe multiverse"; \
            } > /etc/apt/sources.list; \
            if apt-get update 2>/dev/null; then \
                echo "✅ 成功使用清华源"; \
            else \
                echo "⚠️  清华源失败，尝试中科大源..."; \
                cp /etc/apt/sources.list.backup /etc/apt/sources.list; \
                { \
                    echo "# 中科大 Ubuntu 镜像源 (AMD64)"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu/ jammy main restricted universe multiverse"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu/ jammy-security main restricted universe multiverse"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu/ jammy-updates main restricted universe multiverse"; \
                    echo "deb http://mirrors.ustc.edu.cn/ubuntu/ jammy-backports main restricted universe multiverse"; \
                } > /etc/apt/sources.list; \
                if apt-get update 2>/dev/null; then \
                    echo "✅ 成功使用中科大源"; \
                else \
                    echo "⚠️  所有国内源都失败，使用官方源..."; \
                    cp /etc/apt/sources.list.backup /etc/apt/sources.list; \
                    apt-get update; \
                fi; \
            fi; \
        fi; \
    fi; \
    # 显示最终使用的源
    echo "📋 最终使用的APT源:"; \
    cat /etc/apt/sources.list; \
    # 【关键】在换源成功后立即安装 systemd（SLURM Master 必需的核心依赖）
    echo "🔧 【关键步骤】安装 systemd + 基础工具..."; \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        systemd \
        systemd-sysv && \
    # 立即验证 systemd 安装成功
    echo "🔍 验证 systemd 安装..."; \
    if [ -x /lib/systemd/systemd ] && [ -e /sbin/init ]; then \
        echo "✅ systemd 核心已安装: /lib/systemd/systemd"; \
        echo "✅ /sbin/init 符号链接存在"; \
        ls -la /lib/systemd/systemd /sbin/init; \
        /lib/systemd/systemd --version; \
    else \
        echo "❌ systemd 安装验证失败！"; \
        echo "   /lib/systemd/systemd 存在: $([ -x /lib/systemd/systemd ] && echo '是' || echo '否')"; \
        echo "   /sbin/init 存在: $([ -e /sbin/init ] && echo '是' || echo '否')"; \
        exit 1; \
    fi

# 安装其他基础依赖（ca-certificates和curl已在上面安装）
RUN set -eux; \
    # Refresh index right before install to avoid stale caches across layers
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=3 update && break || (echo "apt-get update failed (attempt $i), retrying..." && sleep 5); \
    done; \
    # Install with --fix-missing and retries to improve robustness on flaky mirrors
    apt-get -o Acquire::Retries=3 install -y --no-install-recommends --fix-missing \
        # networking and diagnostics \
        curl \
        wget \
        telnet \
        openssh-client \
        openssh-server \
        sshpass \
        mtr-tiny \
        netcat-openbsd \
        mysql-client \
        default-mysql-client \
        lsof \
        jq \
        # utilities \
        vim \
        tree \
        procps \
        gettext-base \
        tzdata \
        # slurm prerequisites \
        munge \
        libmunge2 \
        libmunge-dev \
        postgresql-client \
        default-mysql-client \
        # SLURM build and runtime dependencies \
        make \
        hwloc \
        libhwloc-dev \
        liblua5.3-0 \
        libfreeipmi17 \
        libjwt0 \
        libb64-0d \
        libipmimonitoring6 \
        libpmix2 \
        libpmix-dev \
        librdkafka1 \
        freeipmi-common \
        pmix \
        libmysqlclient-dev && \
    # Install optional HDF5 and MPI packages (may not be available on all architectures)
    echo "📦 尝试安装可选的 HDF5 和 MPI 包..."; \
    apt-get -o Acquire::Retries=3 install -y --no-install-recommends \
        libhdf5-dev \
        libhdf5-mpich-dev \
        libhdf5-openmpi-dev \
        mpich \
        libmpich-dev \
        openmpi-bin \
        libopenmpi-dev 2>/dev/null || \
    echo "⚠️  部分 HDF5/MPI 包未安装（可能在当前架构不可用）"; \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# 配置时区
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 在安装任何包之前先创建用户，使用固定的 UID/GID 以确保跨节点一致性
# 这样可以避免不同节点上自动分配的 UID/GID 不一致的问题
# slurm: UID=999, GID=999
# munge: UID=998, GID=998
RUN groupadd -g 998 munge && useradd -u 998 -g munge -d /var/lib/munge -s /sbin/nologin munge && \
    groupadd -g 999 slurm && useradd -u 999 -g slurm -d /var/lib/slurm -s /bin/bash slurm

# 创建必要的目录
RUN mkdir -p /etc/slurm \
    /var/spool/slurm/slurmctld \
    /var/spool/slurm/slurmdbd \
    /var/log/slurm \
    /var/lib/slurm \
    /var/run/slurm \
    /etc/munge \
    /var/lib/munge \
    /var/log/munge \
    /var/run/munge \
    /srv/shared

# 复制 third_party 目录以支持离线构建
COPY third_party/ /third_party/

# =============================================================================
# SLURM 安装策略
# =============================================================================
# 1. 首选：从 AppHub 安装（确保版本一致性）
# 2. 回退：如果 ALLOW_SYSTEM_SLURM=true，则从 Ubuntu 官方仓库安装
# 
# 跨架构构建说明：
# - AppHub 镜像包含的 SLURM deb 包是架构相关的
# - 当构建目标架构与 AppHub 架构不匹配时，需要启用系统回退
# - Mac 上通过 Rosetta/QEMU 可以切换 AppHub 架构
# - Linux 上建议使用系统回退 (ALLOW_SYSTEM_SLURM=true)
# =============================================================================
ARG APPHUB_URL=http://apphub:80
ARG ALLOW_SYSTEM_SLURM=false

RUN set -eux; \
    SLURM_INSTALLED=false; \
    SLURM_SOURCE="none"; \
    echo "🔍 尝试从AppHub安装SLURM ${SLURM_VERSION} 包..."; \
    # 添加AppHub源
    echo "deb [trusted=yes] ${APPHUB_URL}/pkgs/slurm-deb ./" > /etc/apt/sources.list.d/ai-infra-slurm.list; \
    echo "📋 AppHub源配置:"; \
    cat /etc/apt/sources.list.d/ai-infra-slurm.list; \
    \
    # 测试AppHub连接
    echo "🌐 测试AppHub连接..."; \
    if curl -sf --max-time 10 ${APPHUB_URL}/pkgs/slurm-deb/Packages > /dev/null; then \
        echo "✅ AppHub连接正常"; \
        \
        # 动态发现AppHub中所有可用的SLURM包
        echo "📦 发现AppHub中的SLURM包..."; \
        AVAILABLE_PACKAGES=$(curl -sL ${APPHUB_URL}/pkgs/slurm-deb/Packages 2>/dev/null | \
            grep "^Package:" | \
            awk '{print $2}' | \
            grep -E "^slurm" | \
            sort -u | \
            tr '\n' ' '); \
        \
        if [ -n "$AVAILABLE_PACKAGES" ]; then \
            echo "✓ 发现的SLURM包:"; \
            echo "$AVAILABLE_PACKAGES" | tr ' ' '\n' | sed 's/^/  - /'; \
            PACKAGE_COUNT=$(echo "$AVAILABLE_PACKAGES" | wc -w); \
            echo "✓ 总计: $PACKAGE_COUNT 个包"; \
            \
            # 定义核心包和可选包
            CORE_PACKAGES="slurm-smd slurm-smd-client slurm-smd-slurmctld slurm-smd-slurmdbd slurm-smd-slurmrestd"; \
            OPTIONAL_PACKAGES=""; \
            \
            # 从可用包中筛选出非核心的可选包
            for pkg in $AVAILABLE_PACKAGES; do \
                is_core=0; \
                for core in $CORE_PACKAGES; do \
                    if [ "$pkg" = "$core" ]; then \
                        is_core=1; \
                        break; \
                    fi; \
                done; \
                if [ $is_core -eq 0 ]; then \
                    OPTIONAL_PACKAGES="$OPTIONAL_PACKAGES $pkg"; \
                fi; \
            done; \
            \
            # 更新包列表
            if timeout 120 apt-get update; then \
                # 安装核心包
                echo "📦 安装核心SLURM包..."; \
                if apt-get install -y --no-install-recommends $CORE_PACKAGES; then \
                    echo "✅ 核心SLURM包安装成功"; \
                    SLURM_INSTALLED=true; \
                    SLURM_SOURCE="AppHub-Core"; \
                    \
                    # 尝试安装可选包（失败不影响）
                    if [ -n "$OPTIONAL_PACKAGES" ]; then \
                        echo "📦 尝试安装可选SLURM包..."; \
                        apt-get install -y --no-install-recommends $OPTIONAL_PACKAGES 2>/dev/null && \
                            echo "✅ 可选包安装成功" || \
                            echo "⚠️  部分可选包安装失败（不影响核心功能）"; \
                    fi; \
                else \
                    echo "❌ 核心包安装失败"; \
                    SLURM_INSTALLED=false; \
                    SLURM_SOURCE="AppHub-Failed"; \
                fi; \
            else \
                echo "❌ apt-get update 失败"; \
                SLURM_INSTALLED=false; \
                SLURM_SOURCE="AppHub-UpdateFailed"; \
            fi; \
        else \
            echo "❌ 未能从AppHub获取包列表"; \
            SLURM_INSTALLED=false; \
            SLURM_SOURCE="AppHub-NoPackages"; \
        fi; \
        \
        # 如果AppHub安装失败，检查是否允许回退到系统仓库
        if [ "$SLURM_INSTALLED" != "true" ]; then \
            if [ "${ALLOW_SYSTEM_SLURM}" = "true" ]; then \
                echo "⚠️  AppHub安装失败，尝试从Ubuntu官方仓库安装..."; \
            else \
                echo "❌ AppHub安装失败，构建终止"; \
                echo "💡 提示: 确保docker-compose构建时AppHub服务可用"; \
                echo "💡 解决方案: 先启动AppHub服务，然后再构建slurm-master"; \
                echo "💡 或者: 设置 --build-arg ALLOW_SYSTEM_SLURM=true 允许从系统仓库安装"; \
                exit 1; \
            fi; \
        fi; \
    else \
        echo "⚠️  AppHub SLURM包不可用（返回404或连接失败）"; \
        if [ "${ALLOW_SYSTEM_SLURM}" = "true" ]; then \
            echo "📦 允许系统回退，将尝试从Ubuntu官方仓库安装..."; \
        else \
            echo "❌ AppHub连接失败，构建终止"; \
            echo "💡 提示: SLURM master必须从AppHub安装以确保版本一致性"; \
            echo "💡 AppHub URL: ${APPHUB_URL}"; \
            echo ""; \
            echo "📋 故障排查:"; \
            echo "   1. 确保AppHub服务正在运行:"; \
            echo "      docker ps | grep apphub"; \
            echo ""; \
            echo "   2. 检查AppHub端口映射:"; \
            echo "      docker port ai-infra-apphub"; \
            echo ""; \
            echo "   3. 测试AppHub连接:"; \
            echo "      curl http://\${EXTERNAL_HOST}:\${APPHUB_PORT}/pkgs/slurm-deb/Packages"; \
            echo ""; \
            echo "   4. 允许系统回退（跨架构构建时）:"; \
            echo "      --build-arg ALLOW_SYSTEM_SLURM=true"; \
            echo ""; \
            exit 1; \
        fi; \
    fi; \
    \
    # ==========================================================================
    # 系统仓库回退：如果AppHub不可用且允许回退
    # ==========================================================================
    if [ "$SLURM_INSTALLED" != "true" ] && [ "${ALLOW_SYSTEM_SLURM}" = "true" ]; then \
        echo ""; \
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
        echo "📦 从Ubuntu官方仓库安装SLURM（系统回退模式）"; \
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
        echo "⚠️  警告: 系统仓库的SLURM版本可能与AppHub版本不同"; \
        echo ""; \
        # 删除AppHub源，使用系统源
        rm -f /etc/apt/sources.list.d/ai-infra-slurm.list; \
        apt-get update; \
        # 安装SLURM核心包（Ubuntu包名与SchedMD不同）
        if apt-get install -y --no-install-recommends \
            slurm-wlm \
            slurm-client \
            slurmctld \
            slurmdbd \
            slurmrestd 2>/dev/null || \
           apt-get install -y --no-install-recommends \
            slurm-wlm \
            slurm-client \
            slurmctld \
            slurmdbd; then \
            echo "✅ SLURM从系统仓库安装成功"; \
            SLURM_INSTALLED=true; \
            SLURM_SOURCE="System-Ubuntu"; \
            # 显示安装的版本
            echo "📋 系统SLURM版本:"; \
            dpkg -l | grep -i slurm | head -10 || true; \
        else \
            echo "❌ 系统仓库安装也失败了"; \
            SLURM_INSTALLED=false; \
            SLURM_SOURCE="System-Failed"; \
        fi; \
    fi; \
    \
    # 最终检查
    if [ "$SLURM_INSTALLED" != "true" ]; then \
        echo "❌ SLURM安装失败（所有方法都失败了）"; \
        exit 1; \
    fi; \
    \
    # 确保关键工具包已安装（bootstrap脚本依赖）
    echo "📦 确保关键工具包已安装..."; \
    apt-get update && apt-get install -y --no-install-recommends \
        netcat-openbsd \
        mysql-client \
        default-mysql-client \
        wget \
        telnet \
        gettext-base 2>/dev/null || \
    echo "⚠️  部分工具包安装失败"; \
    \
    # 【重要】删除构建时使用的 APT 源配置
    # 避免将构建机器的 IP 地址写入镜像
    echo "🧹 清理构建时的 APT 源配置..."; \
    rm -f /etc/apt/sources.list.d/ai-infra-slurm.list; \
    \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    \
    # 创建标记文件和路径检查
    touch /opt/slurm-installed; \
    echo "$SLURM_SOURCE" > /opt/slurm-source; \
    # 动态检查实际的二进制文件位置
    echo "🔍 检查SLURM二进制文件位置..."; \
    SLURMCTLD_PATH=$(which slurmctld 2>/dev/null || find /usr -name "slurmctld" -type f -executable 2>/dev/null | head -1 || echo ""); \
    SLURMDBD_PATH=$(which slurmdbd 2>/dev/null || find /usr -name "slurmdbd" -type f -executable 2>/dev/null | head -1 || echo ""); \
    \
    if [ -n "$SLURMCTLD_PATH" ] && [ -x "$SLURMCTLD_PATH" ]; then \
        echo "$SLURMCTLD_PATH" > /opt/slurmctld-path; \
        echo "✅ slurmctld: $SLURMCTLD_PATH"; \
    else \
        echo "❌ slurmctld二进制文件未找到"; \
        exit 1; \
    fi; \
    \
    if [ -n "$SLURMDBD_PATH" ] && [ -x "$SLURMDBD_PATH" ]; then \
        echo "$SLURMDBD_PATH" > /opt/slurmdbd-path; \
        echo "✅ slurmdbd: $SLURMDBD_PATH"; \
    else \
        echo "❌ slurmdbd二进制文件未找到"; \
        exit 1; \
    fi; \
    \
    echo "📦 SLURM安装摘要:"; \
    echo "  来源: $SLURM_SOURCE"; \
    echo "  版本: $(slurmctld -V)"; \
    echo "  slurmctld: $(cat /opt/slurmctld-path)"; \
    echo "  slurmdbd: $(cat /opt/slurmdbd-path)"

# 配置用户组和权限（在SLURM安装后）
RUN usermod -a -G munge slurm && \
    chown -R slurm:slurm /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm /etc/slurm && \
    chown -R munge:munge /etc/munge /var/lib/munge /var/log/munge /var/run/munge && \
    chmod 755 /var/spool/slurm /var/log/slurm /var/lib/slurm /var/run/slurm && \
    chmod 700 /etc/munge /var/lib/munge && \
    chmod 755 /var/log/munge /var/run/munge && \
    chmod 640 /etc/munge/munge.key || true

# 注意：slurmrestd 现在通过上面的动态包发现自动安装，不再需要单独的安装脚本

# 复制配置与systemd脚本
COPY src/slurm-master/config/ /etc/slurm-templates/
COPY src/slurm-master/entrypoint.sh /usr/local/bin/slurm-bootstrap.sh
COPY src/slurm-master/systemd-entrypoint.sh /usr/local/bin/systemd-entrypoint.sh
COPY src/slurm-master/healthcheck.sh /usr/local/bin/healthcheck.sh
COPY src/slurm-master/systemd/ /etc/systemd/system/
RUN chmod +x /usr/local/bin/slurm-bootstrap.sh /usr/local/bin/systemd-entrypoint.sh /usr/local/bin/healthcheck.sh && \
    ln -sf /etc/systemd/system/slurm-bootstrap.service /etc/systemd/system/multi-user.target.wants/slurm-bootstrap.service && \
    ln -sf /etc/systemd/system/slurmctld.service /etc/systemd/system/multi-user.target.wants/slurmctld.service && \
    ln -sf /etc/systemd/system/slurmdbd.service /etc/systemd/system/multi-user.target.wants/slurmdbd.service && \
    ln -sf /lib/systemd/system/munge.service /etc/systemd/system/multi-user.target.wants/munge.service

# 配置SSH服务（使用统一密钥管理）
RUN mkdir -p /var/run/sshd /root/.ssh && \
    chmod 700 /root/.ssh && \
    # 确保已经安装并生成 sshd 配置
    if [ ! -f /etc/ssh/sshd_config ]; then \
        apt-get update && apt-get install -y --no-install-recommends openssh-server; \
    fi && \
    # 配置SSH服务器（仅允许公钥认证，安全配置）
    sed -i 's/#PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    # 启用SSH服务（构建阶段可能没有systemctl，手动创建WantedBy链接）
    if command -v systemctl >/dev/null 2>&1; then \
        systemctl enable ssh; \
    elif [ -d /etc/systemd/system/multi-user.target.wants ]; then \
        ln -sf /lib/systemd/system/ssh.service /etc/systemd/system/multi-user.target.wants/ssh.service; \
    fi

# 复制统一的SSH公钥（允许backend访问）
# Note: SSH密钥会在构建前由build.sh从项目根目录同步到此处
COPY ssh-key/id_rsa.pub /root/.ssh/authorized_keys
RUN chmod 600 /root/.ssh/authorized_keys && \
    cp /root/.ssh/authorized_keys /root/.ssh/authorized_keys.bootstrap && \
    echo "✓ SSH public key installed for backend access"

# 复制并启用运行时公钥刷新脚本（支持从共享目录热更新）
COPY src/slurm-master/scripts/bootstrap-authorized-keys.sh /usr/local/bin/bootstrap-authorized-keys.sh
RUN chmod +x /usr/local/bin/bootstrap-authorized-keys.sh

# 暴露端口
EXPOSE 6817 6818 22

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=90s --retries=3 \
    CMD /usr/local/bin/healthcheck.sh

# 设置工作目录
WORKDIR /etc/slurm

# 启动脚本
STOPSIGNAL SIGRTMIN+3

VOLUME ["/sys/fs/cgroup"]

ENTRYPOINT ["/usr/local/bin/systemd-entrypoint.sh"]
CMD ["/sbin/init"]