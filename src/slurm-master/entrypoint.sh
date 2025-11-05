#!/bin/bash
#!/bin/bash
set -euo pipefail

log() {
    local level="$1"; shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*"
}

log "INFO" "🚀 启动 AI Infrastructure Matrix SLURM Master 引导任务..."

# 默认环境变量
export SLURM_CLUSTER_NAME=${SLURM_CLUSTER_NAME:-ai-infra-cluster}
export SLURM_CONTROLLER_HOST=${SLURM_CONTROLLER_HOST:-slurm-master}
export SLURM_CONTROLLER_PORT=${SLURM_CONTROLLER_PORT:-6817}
export SLURM_SLURMDBD_HOST=${SLURM_SLURMDBD_HOST:-slurm-master}
export SLURM_SLURMDBD_PORT=${SLURM_SLURMDBD_PORT:-6818}

# 数据库配置 (MySQL 默认)
export SLURM_DB_HOST=${SLURM_DB_HOST:-mysql}
export SLURM_DB_PORT=${SLURM_DB_PORT:-3306}
export SLURM_DB_NAME=${SLURM_DB_NAME:-slurm_acct_db}
export SLURM_DB_USER=${SLURM_DB_USER:-slurm}
export SLURM_DB_PASSWORD=${SLURM_DB_PASSWORD:-slurm123}
export MYSQL_ROOT_USER=${MYSQL_ROOT_USER:-root}
export MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-}

# 认证与节点配置
export SLURM_AUTH_TYPE=${SLURM_AUTH_TYPE:-auth/munge}
export SLURM_MUNGE_KEY=${SLURM_MUNGE_KEY:-ai-infra-slurm-munge-key-dev}
export SLURM_PARTITION_NAME=${SLURM_PARTITION_NAME:-compute}
export SLURM_DEFAULT_PARTITION=${SLURM_DEFAULT_PARTITION:-compute}
export SLURM_NODE_PREFIX=${SLURM_NODE_PREFIX:-compute}
export SLURM_NODE_COUNT=${SLURM_NODE_COUNT:-3}
export SLURM_TEST_NODES=${SLURM_TEST_NODES:-}
export SLURM_TEST_NODE_CPUS=${SLURM_TEST_NODE_CPUS:-4}
export SLURM_TEST_NODE_MEMORY=${SLURM_TEST_NODE_MEMORY:-8192}
export SLURM_MAX_JOB_COUNT=${SLURM_MAX_JOB_COUNT:-10000}
export SLURM_MAX_ARRAY_SIZE=${SLURM_MAX_ARRAY_SIZE:-1000}
export SLURM_DEFAULT_TIME_LIMIT=${SLURM_DEFAULT_TIME_LIMIT:-01:00:00}
export SLURM_MAX_TIME_LIMIT=${SLURM_MAX_TIME_LIMIT:-24:00:00}

detect_slurm_mode() {
    if [ -f /opt/slurm-installed ]; then
        SLURM_MODE="full"
        log "INFO" "✅ 检测到完整SLURM安装"
        return
    fi

    if [ -f /opt/slurm-demo-mode ]; then
        SLURM_MODE="demo"
        log "WARN" "⚠️ 检测到演示模式"
        return
    fi

    if command -v slurmctld >/dev/null 2>&1 && command -v slurmdbd >/dev/null 2>&1; then
        SLURM_MODE="full"
        touch /opt/slurm-installed
        log "INFO" "✅ SLURM组件可用，使用完整模式"
    else
        SLURM_MODE="demo"
        touch /opt/slurm-demo-mode
        log "WARN" "⚠️ 未检测到完整SLURM安装，启用演示模式"
    fi
}

set_plugin_dir() {
    local arch_dpkg arch_gnu
    arch_dpkg=$(dpkg --print-architecture)

    case "$arch_dpkg" in
        amd64) arch_gnu="x86_64-linux-gnu" ;;
        arm64) arch_gnu="aarch64-linux-gnu" ;;
        armhf) arch_gnu="arm-linux-gnueabihf" ;;
        *) arch_gnu="${arch_dpkg}-linux-gnu" ;;
    esac

    local candidates=(
        "/usr/lib/$arch_gnu/slurm-wlm"
        "/usr/lib/$arch_gnu/slurm"
        "/usr/lib/$arch_dpkg/slurm-wlm"
        "/usr/lib/$arch_dpkg/slurm"
        "/usr/lib64/slurm-wlm"
        "/usr/lib64/slurm"
        "/usr/lib/slurm-wlm"
        "/usr/lib/slurm"
    )

    for dir in "${candidates[@]}"; do
        if [ -d "$dir" ] && [ -n "$(ls -A "$dir" 2>/dev/null)" ]; then
            export SLURM_PLUGIN_DIR="$dir"
            log "INFO" "✅ 使用SLURM插件目录: $dir"
            return
        fi
    done

    export SLURM_PLUGIN_DIR="/usr/lib/${arch_dpkg}/slurm"
    log "WARN" "⚠️ 未找到特定插件目录，回退至 ${SLURM_PLUGIN_DIR}"
}

print_configuration() {
    log "INFO" "📋 SLURM 配置摘要"
    log "INFO" "  运行模式: ${SLURM_MODE}"
    log "INFO" "  集群名称: ${SLURM_CLUSTER_NAME}"
    log "INFO" "  控制器: ${SLURM_CONTROLLER_HOST}:${SLURM_CONTROLLER_PORT}"
    log "INFO" "  数据库: ${SLURM_DB_HOST}:${SLURM_DB_PORT}/${SLURM_DB_NAME}"
    log "INFO" "  插件目录: ${SLURM_PLUGIN_DIR}"
    log "INFO" "  测试节点: ${SLURM_TEST_NODES}"
}

wait_for_database() {
    if [ "${SLURM_MODE}" = "demo" ]; then
        log "WARN" "演示模式跳过数据库等待"
        return 0
    fi

    local host=${SLURM_DB_HOST}
    local port=${SLURM_DB_PORT}
    local user=${SLURM_DB_USER}
    local pass=${SLURM_DB_PASSWORD}
    local max_attempts=30

    log "INFO" "⏳ 等待 MySQL 数据库 ${host}:${port} 就绪..."
    for attempt in $(seq 1 ${max_attempts}); do
        if mysqladmin ping -h "${host}" -P "${port}" -u "${user}" --password="${pass}" --connect-timeout=5 >/dev/null 2>&1; then
            log "INFO" "✅ MySQL 数据库已就绪"
            return 0
        fi
        if nc -z "${host}" "${port}" >/dev/null 2>&1; then
            log "INFO" "✅ MySQL 端口已开放"
            return 0
        fi
        log "WARN" "等待数据库中 (第 ${attempt}/${max_attempts} 次)..."
        sleep 5
    done

    log "ERROR" "MySQL 数据库在规定时间内未就绪"
    return 1
}

mysql_exec() {
    local user="$1"; shift
    local passwd="$1"; shift
    MYSQL_PWD="$passwd" mysql --protocol=TCP -h "${SLURM_DB_HOST}" -P "${SLURM_DB_PORT}" -u "$user" "$@"
}

init_database() {
    if [ "${SLURM_MODE}" = "demo" ]; then
        log "WARN" "演示模式跳过数据库初始化"
        return 0
    fi

    log "INFO" "🛠️ 初始化SLURM数据库..."

    local admin_user=${MYSQL_ROOT_USER}
    local admin_pass=${MYSQL_ROOT_PASSWORD}
    local db=${SLURM_DB_NAME}
    local slurm_user=${SLURM_DB_USER}
    local slurm_pass=${SLURM_DB_PASSWORD}

    if [ -n "${admin_pass}" ]; then
        log "INFO" "使用管理员用户 ${admin_user} 初始化数据库"
        if ! mysql_exec "${admin_user}" "${admin_pass}" -e "CREATE DATABASE IF NOT EXISTS \\`${db}\\`;" >/dev/null 2>&1; then
            log "WARN" "创建数据库 ${db} 失败（可能已存在或权限不足）"
        else
            log "INFO" "数据库 ${db} 可用"
        fi

        mysql_exec "${admin_user}" "${admin_pass}" <<SQL || log "WARN" "授予权限时出现问题"
CREATE USER IF NOT EXISTS '${slurm_user}'@'%' IDENTIFIED BY '${slurm_pass}';
GRANT ALL PRIVILEGES ON \\`${db}\\`.* TO '${slurm_user}'@'%';
FLUSH PRIVILEGES;
SQL
    else
        log "WARN" "未提供管理员凭据，尝试使用 ${slurm_user} 用户验证访问"
        if mysql_exec "${slurm_user}" "${slurm_pass}" -e "SELECT 1;" >/dev/null 2>&1; then
            log "INFO" "验证 slurm 用户访问成功"
        else
            log "ERROR" "无法验证 slurm 用户访问数据库，请检查凭据"
            return 1
        fi
    fi

    log "INFO" "✅ 数据库初始化逻辑完成"
}

generate_configs() {
    log "INFO" "📝 生成 SLURM 配置文件..."
    mkdir -p /etc/slurm

    envsubst < /etc/slurm-templates/slurm.conf.template > /etc/slurm/slurm.conf
    envsubst < /etc/slurm-templates/slurmdbd.conf.template > /etc/slurm/slurmdbd.conf
    envsubst < /etc/slurm-templates/cgroup.conf.template > /etc/slurm/cgroup.conf

    # 如果没有配置测试节点，移除空的 NodeName 和 PartitionName 行
    if [ -z "${SLURM_TEST_NODES}" ]; then
        log "INFO" "未配置测试节点，移除空的节点配置行"
        sed -i '/^NodeName= /d' /etc/slurm/slurm.conf
        sed -i '/^PartitionName=.*Nodes= /d' /etc/slurm/slurm.conf
    fi

    chown slurm:slurm /etc/slurm/slurm.conf /etc/slurm/cgroup.conf /etc/slurm/slurmdbd.conf
    chmod 644 /etc/slurm/slurm.conf /etc/slurm/cgroup.conf
    chmod 600 /etc/slurm/slurmdbd.conf

    # Fix SLURM log directory permissions
    log "INFO" "🔧 修复 SLURM 日志目录权限..."
    mkdir -p /var/log/slurm
    chown -R slurm:slurm /var/log/slurm
    chmod 755 /var/log/slurm
    # Remove any existing log files created by root and let slurm recreate them
    rm -f /var/log/slurm/*.log

    log "INFO" "✅ 配置文件生成完成"
}

setup_munge() {
    log "INFO" "🔐 配置 Munge 认证..."
    if [ ! -f /etc/munge/munge.key ]; then
        printf "%s" "${SLURM_MUNGE_KEY}" > /etc/munge/munge.key
        log "INFO" "生成 Munge 密钥"
    else
        log "INFO" "使用现有 Munge 密钥"
    fi

    # Create required directories
    mkdir -p /run/munge /var/log/munge /var/lib/munge

    # Fix ownership of munge directories and files
    chown -R munge:munge /etc/munge /var/lib/munge /var/log/munge /run/munge
    chmod 700 /etc/munge /var/lib/munge /var/log/munge
    chmod 755 /run/munge
    chmod 400 /etc/munge/munge.key

    log "INFO" "✅ Munge 配置完成"
}

bootstrap() {
    detect_slurm_mode
    set_plugin_dir
    print_configuration

    if [ "${SLURM_MODE}" = "full" ]; then
        wait_for_database
        init_database
    else
        log "WARN" "演示模式将仅生成基础配置"
    fi

    generate_configs
    setup_munge

    log "INFO" "✨ SLURM 引导任务完成"
}

case "${1:-bootstrap}" in
    bootstrap)
        bootstrap
        ;;
    generate-config)
        detect_slurm_mode
        set_plugin_dir
        generate_configs
        log "INFO" "配置文件生成完成"
        ;;
    test-connection)
        detect_slurm_mode
        wait_for_database
        log "INFO" "数据库连接检测通过"
        ;;
    *)
        log "ERROR" "用法: $0 {bootstrap|generate-config|test-connection}"
        exit 1
        ;;
esac

exit 0