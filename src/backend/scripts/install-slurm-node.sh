#!/bin/bash
#
# install-slurm-node.sh - Install SLURM node components using apphub RPM repository
#
# This script is executed remotely via SaltStack to install SLURM on compute nodes
#
# Usage: 
#   ./install-slurm-node.sh <apphub_url> <node_type>
#
# Arguments:
#   apphub_url  - URL to apphub server (e.g., http://ai-infra-apphub:8080)
#   node_type   - Node type: compute (default) or login
#
# Environment Variables:
#   SLURM_VERSION - SLURM version to install (default: auto-detect from repo)
#

set -euo pipefail

# 配置变量
APPHUB_URL="${1:-http://ai-infra-apphub:8080}"
NODE_TYPE="${2:-compute}"
SLURM_VERSION="${SLURM_VERSION:-}"

# 日志函数
log_info() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [INFO] $*"
}

log_error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [ERROR] $*" >&2
}

log_warn() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [WARN] $*"
}

# 检测操作系统类型
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="$ID"
        OS_VERSION="$VERSION_ID"
        log_info "Detected OS: $OS_ID $OS_VERSION"
        
        # 标准化 OS 类型
        case "$OS_ID" in
            rocky|centos|rhel|fedora)
                OS_TYPE="rpm"
                ;;
            ubuntu|debian)
                OS_TYPE="deb"
                ;;
            *)
                log_error "Unsupported OS: $OS_ID"
                exit 1
                ;;
        esac
    else
        log_error "Cannot detect OS type"
        exit 1
    fi
}

# 配置 SLURM RPM 仓库
configure_slurm_repo() {
    if [ "$OS_TYPE" = "rpm" ]; then
        log_info "Configuring SLURM RPM repository from apphub..."
        
        # 检测包管理器
        if command -v dnf &>/dev/null; then
            PKG_MANAGER="dnf"
        elif command -v yum &>/dev/null; then
            PKG_MANAGER="yum"
        else
            log_error "No supported package manager found (dnf/yum)"
            exit 1
        fi
        
        log_info "Using package manager: $PKG_MANAGER"
        
        # 创建 SLURM 仓库配置
        cat > /etc/yum.repos.d/slurm-apphub.repo <<EOF
[slurm-apphub]
name=SLURM from AI-Infra AppHub
baseurl=${APPHUB_URL}/pkgs/slurm-rpm/
enabled=1
gpgcheck=0
priority=1
EOF

        log_info "Created repository configuration: /etc/yum.repos.d/slurm-apphub.repo"
        
        # 清理并重建缓存
        log_info "Refreshing package manager cache..."
        $PKG_MANAGER clean all
        $PKG_MANAGER makecache
        
        # 验证仓库可用性
        if $PKG_MANAGER repolist | grep -q "slurm-apphub"; then
            log_info "✓ SLURM repository configured successfully"
        else
            log_error "Failed to configure SLURM repository"
            exit 1
        fi
        
    elif [ "$OS_TYPE" = "deb" ]; then
        log_info "Configuring SLURM DEB repository from apphub..."
        
        # 安装必要的工具
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        apt-get install -y wget gnupg2 ca-certificates
        
        # 配置 APT 仓库
        cat > /etc/apt/sources.list.d/slurm-apphub.list <<EOF
deb [trusted=yes] ${APPHUB_URL}/pkgs/slurm-deb/ ./
EOF

        log_info "Created repository configuration: /etc/apt/sources.list.d/slurm-apphub.list"
        
        # 更新包缓存
        log_info "Refreshing package manager cache..."
        apt-get update -qq
        
        # 验证仓库可用性
        if apt-cache search slurm-smd &>/dev/null; then
            log_info "✓ SLURM repository configured successfully"
        else
            log_error "Failed to configure SLURM repository"
            exit 1
        fi
    fi
}

# 安装 munge（认证服务）
install_munge() {
    log_info "Installing munge authentication service..."
    
    if [ "$OS_TYPE" = "rpm" ]; then
        # RPM-based systems
        if $PKG_MANAGER install -y munge munge-libs 2>/dev/null; then
            log_info "✓ Installed munge from system repository"
        else
            log_warn "munge not available in system repo, checking if already installed..."
            if command -v munged &>/dev/null; then
                log_info "✓ munge already installed"
            else
                log_error "Failed to install munge - SLURM authentication will not work"
            fi
        fi
    elif [ "$OS_TYPE" = "deb" ]; then
        # Debian-based systems
        export DEBIAN_FRONTEND=noninteractive
        if apt-get install -y munge 2>/dev/null; then
            log_info "✓ Installed munge from system repository"
        else
            log_error "Failed to install munge"
            exit 1
        fi
    fi
    
    # 确保 munge 用户和组存在
    if ! getent group munge &>/dev/null; then
        groupadd -r munge || log_warn "Failed to create munge group"
    fi
    
    if ! getent passwd munge &>/dev/null; then
        useradd -r -g munge -d /var/lib/munge -s /sbin/nologin munge || log_warn "Failed to create munge user"
    fi
    
    # 创建必要的目录并设置正确的权限
    mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge
    
    # 关键：正确的权限配置
    # /var/log/munge 和 /var/lib/munge 需要 root 所有（munged 以 root 启动然后降权）
    # /etc/munge 和 /run/munge 需要 munge 所有
    chown -R root:root /var/log/munge /var/lib/munge
    chown -R munge:munge /etc/munge /run/munge
    chmod 700 /etc/munge /var/lib/munge /var/log/munge
    chmod 755 /run/munge
    
    log_info "✓ Munge directories and permissions configured"
}

# 安装 SLURM 组件
install_slurm_packages() {
    log_info "Installing SLURM packages for node type: $NODE_TYPE..."
    
    if [ "$OS_TYPE" = "rpm" ]; then
        # RPM-based systems (Rocky/CentOS/RHEL)
        PACKAGES="slurm slurm-slurmd"
        
        # 根据节点类型添加额外包
        case "$NODE_TYPE" in
            compute)
                PACKAGES="$PACKAGES slurm-libpmi"
                log_info "Installing compute node packages: $PACKAGES"
                ;;
            login)
                PACKAGES="$PACKAGES slurm-contribs"
                log_info "Installing login node packages: $PACKAGES"
                ;;
            *)
                log_warn "Unknown node type: $NODE_TYPE, using default (compute)"
                ;;
        esac
        
        # 安装依赖（cgroup v2 需要 dbus-libs）
        log_info "Installing dependencies for cgroup v2 support..."
        $PKG_MANAGER install -y dbus-libs || log_warn "Failed to install dbus-libs"
        
        # 安装 SLURM 包
        log_info "Running: $PKG_MANAGER install -y $PACKAGES"
        if $PKG_MANAGER install -y $PACKAGES; then
            log_info "✓ SLURM packages installed successfully"
        else
            log_error "Failed to install SLURM packages"
            exit 1
        fi
        
        # Rocky RPM 包需要额外配置
        log_info "Configuring Rocky-specific SLURM settings..."
        
        # 检查 cgroup v2 插件是否存在
        if [ ! -f /usr/lib64/slurm/cgroup_v2.so ]; then
            log_warn "cgroup_v2.so not found in RPM package"
            log_warn "This is a known issue with Rocky RPM builds"
            log_warn "Trying to download from AppHub Ubuntu package..."
            
            # 尝试从 AppHub 获取 cgroup_v2.so
            if wget -q -O /tmp/cgroup_v2.so "${APPHUB_URL}/slurm-plugins/cgroup_v2.so" 2>/dev/null; then
                cp /tmp/cgroup_v2.so /usr/lib64/slurm/
                chmod 755 /usr/lib64/slurm/cgroup_v2.so
                log_info "✓ Downloaded and installed cgroup_v2.so from AppHub"
            else
                log_error "Failed to download cgroup_v2.so"
                log_error "Node will not work with cgroup v2 (Rocky 9.x default)"
            fi
        else
            log_info "✓ cgroup_v2.so plugin found"
        fi
        
        # 创建 /etc/slurm 符号链接（Rocky RPM 使用 /usr/etc/slurm）
        if [ ! -L /etc/slurm ] && [ -d /usr/etc/slurm ]; then
            log_info "Creating symlink: /etc/slurm -> /usr/etc/slurm"
            ln -sf /usr/etc/slurm /etc/slurm
        fi
        
    elif [ "$OS_TYPE" = "deb" ]; then
        # Debian-based systems (Ubuntu/Debian) - 使用 AppHub 的 DEB 包
        log_info "Installing SLURM from AppHub (DEB packages)"
        export DEBIAN_FRONTEND=noninteractive
        
        # 使用 slurm-smd 包（从 AppHub 构建的包）
        PACKAGES="slurm-smd-client slurm-smd-slurmd"
        
        # 根据节点类型添加额外包
        case "$NODE_TYPE" in
            compute)
                PACKAGES="$PACKAGES slurm-smd-libpmi0 slurm-smd-libslurm-perl"
                ;;
            login)
                PACKAGES="$PACKAGES slurm-smd-doc"
                ;;
        esac
        
        log_info "Installing packages: $PACKAGES"
        if apt-get install -y $PACKAGES; then
            log_info "✓ SLURM packages installed successfully"
        else
            log_error "Failed to install SLURM packages"
            exit 1
        fi
        
        # Ubuntu DEB 包包含完整的 cgroup v2 支持
        log_info "✓ Ubuntu package includes cgroup v2 support"
    fi
    
    # 验证安装
    if command -v slurmd &>/dev/null; then
        INSTALLED_VERSION=$(slurmd -V | head -1)
        log_info "✓ Installed: $INSTALLED_VERSION"
    else
        log_error "slurmd command not found after installation"
        exit 1
    fi
}

# 创建 SLURM 用户和组
create_slurm_user() {
    log_info "Creating SLURM user and group..."
    
    # 统一使用 UID/GID 1999（避免与其他节点冲突）
    SLURM_UID=1999
    SLURM_GID=1999
    
    if ! getent group slurm &>/dev/null; then
        groupadd -g $SLURM_GID slurm
        log_info "✓ Created slurm group (GID: $SLURM_GID)"
    else
        # 检查现有 GID
        EXISTING_GID=$(getent group slurm | cut -d: -f3)
        if [ "$EXISTING_GID" != "$SLURM_GID" ]; then
            log_warn "slurm group exists with different GID: $EXISTING_GID, changing to $SLURM_GID"
            groupmod -g $SLURM_GID slurm
        else
            log_info "slurm group already exists (GID: $SLURM_GID)"
        fi
    fi
    
    if ! getent passwd slurm &>/dev/null; then
        useradd -u $SLURM_UID -g slurm -d /var/spool/slurm -s /sbin/nologin slurm
        log_info "✓ Created slurm user (UID: $SLURM_UID)"
    else
        # 检查现有 UID
        EXISTING_UID=$(id -u slurm)
        if [ "$EXISTING_UID" != "$SLURM_UID" ]; then
            log_warn "slurm user exists with different UID: $EXISTING_UID, changing to $SLURM_UID"
            usermod -u $SLURM_UID slurm
            # 更新文件所有权
            find /var/spool/slurm /var/log/slurm /run/slurm -user $EXISTING_UID -exec chown slurm:slurm {} \; 2>/dev/null || true
        else
            log_info "slurm user already exists (UID: $SLURM_UID)"
        fi
    fi
}

# 创建必要的目录结构
create_directories() {
    log_info "Creating SLURM directory structure..."
    
    # 为 Rocky RPM 创建 /usr/etc/slurm 目录
    if [ "$OS_TYPE" = "rpm" ] && [ ! -d /usr/etc/slurm ]; then
        mkdir -p /usr/etc/slurm
        log_info "Created /usr/etc/slurm (Rocky RPM location)"
    fi
    
    # 创建标准目录
    mkdir -p /etc/slurm \
             /var/spool/slurm/d \
             /var/spool/slurm/ctld \
             /var/log/slurm \
             /run/slurm
    
    # 设置权限
    chown -R slurm:slurm /var/spool/slurm /var/log/slurm /run/slurm
    chmod 755 /etc/slurm
    chmod 755 /var/spool/slurm
    chmod 755 /var/log/slurm
    
    # Rocky: 如果 /usr/etc/slurm 存在且 /etc/slurm 不是符号链接，创建链接
    if [ "$OS_TYPE" = "rpm" ] && [ -d /usr/etc/slurm ] && [ ! -L /etc/slurm ]; then
        if [ -d /etc/slurm ] && [ ! -L /etc/slurm ]; then
            # 如果 /etc/slurm 是真实目录，先备份
            if [ "$(ls -A /etc/slurm 2>/dev/null)" ]; then
                mv /etc/slurm /etc/slurm.backup.$(date +%s)
                log_info "Backed up existing /etc/slurm"
            else
                rmdir /etc/slurm
            fi
        fi
        ln -sf /usr/etc/slurm /etc/slurm
        log_info "Created symlink: /etc/slurm -> /usr/etc/slurm"
    fi
    
    log_info "✓ Directory structure created"
}

# 配置 systemd 服务
configure_systemd() {
    log_info "Configuring systemd service for slurmd..."
    
    # 重新加载 systemd
    systemctl daemon-reload
    
    # 启用但不启动服务（需要配置文件）
    systemctl enable slurmd || log_warn "Failed to enable slurmd service"
    
    log_info "✓ systemd configured (service enabled but not started)"
    log_info "Note: Start service after deploying slurm.conf with: systemctl start slurmd"
}

# 直接启动 slurmd（不使用 systemd，避免超时问题）
start_slurmd_direct() {
    log_info "Starting slurmd daemon directly..."
    
    # 停止已有的 slurmd 进程
    pkill -9 slurmd 2>/dev/null || true
    
    # 直接启动 slurmd（后台运行）
    /usr/sbin/slurmd
    
    # 等待进程启动
    sleep 2
    
    # 验证进程是否运行
    if pgrep -x slurmd > /dev/null; then
        log_info "✓ slurmd daemon started successfully"
    else
        log_error "Failed to start slurmd daemon"
        return 1
    fi
}

# 主函数
main() {
    log_info "=========================================="
    log_info "SLURM Node Installation Script"
    log_info "=========================================="
    log_info "AppHub URL: $APPHUB_URL"
    log_info "Node Type: $NODE_TYPE"
    log_info ""
    
    # 检测操作系统
    detect_os
    
    # 配置仓库
    configure_slurm_repo
    
    # 安装组件
    install_munge
    create_slurm_user
    install_slurm_packages
    create_directories
    configure_systemd
    
    log_info ""
    log_info "=========================================="
    log_info "✓ SLURM node installation completed successfully"
    log_info "=========================================="
    log_info ""
    log_info "Next steps:"
    log_info "  1. Deploy munge key: /etc/munge/munge.key"
    log_info "  2. Deploy slurm config: /etc/slurm/slurm.conf"
    log_info "  3. Start munge: systemctl start munge"
    log_info "  4. Start slurmd: systemctl start slurmd"
    log_info ""
}

# 执行主函数
main "$@"
