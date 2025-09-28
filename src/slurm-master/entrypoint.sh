#!/bin/bash
set -e

# AI Infrastructure Matrix SLURM Master Entrypoint
echo "🚀 启动 AI Infrastructure Matrix SLURM Master..."

# 检查SLURM安装状态
if [ -f /opt/slurm-installed ]; then
    echo "✅ SLURM已安装，使用完整功能模式"
    SLURM_MODE="full"
elif [ -f /opt/slurm-demo-mode ]; then
    echo "⚠️ SLURM未安装，使用演示模式"
    SLURM_MODE="demo"
else
    echo "🔍 检测SLURM安装状态..."
    if command -v slurmctld >/dev/null 2>&1 && command -v slurmdbd >/dev/null 2>&1; then
        echo "✅ SLURM已安装"
        SLURM_MODE="full"
        touch /opt/slurm-installed
    else
        echo "⚠️ SLURM未安装，使用演示模式"
        SLURM_MODE="demo"
        touch /opt/slurm-demo-mode
    fi
fi

# 环境变量默认值
export SLURM_CLUSTER_NAME=${SLURM_CLUSTER_NAME:-ai-infra-cluster}
export SLURM_CONTROLLER_HOST=${SLURM_CONTROLLER_HOST:-slurm-master}
export SLURM_CONTROLLER_PORT=${SLURM_CONTROLLER_PORT:-6817}
export SLURM_SLURMDBD_HOST=${SLURM_SLURMDBD_HOST:-slurm-master}
export SLURM_SLURMDBD_PORT=${SLURM_SLURMDBD_PORT:-6818}

# 数据库配置 (使用MySQL作为默认，从环境变量读取)
export SLURM_DB_HOST=${SLURM_DB_HOST:-mysql}
export SLURM_DB_PORT=${SLURM_DB_PORT:-3306}
export SLURM_DB_NAME=${SLURM_DB_NAME:-slurm_acct_db}
export SLURM_DB_USER=${SLURM_DB_USER:-slurm}
export SLURM_DB_PASSWORD=${SLURM_DB_PASSWORD:-slurm123}

# 认证配置
export SLURM_AUTH_TYPE=${SLURM_AUTH_TYPE:-auth/munge}
export SLURM_MUNGE_KEY=${SLURM_MUNGE_KEY:-ai-infra-slurm-munge-key-dev}

# 节点配置
export SLURM_PARTITION_NAME=${SLURM_PARTITION_NAME:-compute}
export SLURM_DEFAULT_PARTITION=${SLURM_DEFAULT_PARTITION:-compute}
export SLURM_TEST_NODES=${SLURM_TEST_NODES:-test-ssh01,test-ssh02,test-ssh03}
export SLURM_TEST_NODE_CPUS=${SLURM_TEST_NODE_CPUS:-4}
export SLURM_TEST_NODE_MEMORY=${SLURM_TEST_NODE_MEMORY:-8192}

# 作业配置
export SLURM_MAX_JOB_COUNT=${SLURM_MAX_JOB_COUNT:-10000}
export SLURM_MAX_ARRAY_SIZE=${SLURM_MAX_ARRAY_SIZE:-1000}
export SLURM_DEFAULT_TIME_LIMIT=${SLURM_DEFAULT_TIME_LIMIT:-01:00:00}
export SLURM_MAX_TIME_LIMIT=${SLURM_MAX_TIME_LIMIT:-24:00:00}

# 动态检测插件目录
if [ "$SLURM_MODE" = "full" ]; then
    echo "🔍 检测SLURM插件目录..."
    
    # 检测可能的插件目录位置（优先检查slurm-wlm目录）
    # 支持不同的架构命名约定
    arch_dpkg=$(dpkg --print-architecture)
    arch_gnu=""
    case "$arch_dpkg" in
        "amd64") arch_gnu="x86_64-linux-gnu" ;;
        "arm64") arch_gnu="aarch64-linux-gnu" ;;
        "armhf") arch_gnu="arm-linux-gnueabihf" ;;
        *) arch_gnu="$arch_dpkg-linux-gnu" ;;
    esac
    
    POSSIBLE_DIRS=(
        "/usr/lib/$arch_gnu/slurm-wlm"
        "/usr/lib/$arch_gnu/slurm"
        "/usr/lib/$arch_dpkg/slurm-wlm"
        "/usr/lib/$arch_dpkg/slurm"
        "/usr/lib64/slurm-wlm"
        "/usr/lib64/slurm"
        "/usr/lib/slurm-wlm"
        "/usr/lib/slurm"
    )
    
    PLUGIN_DIR_FOUND=""
    for dir in "${POSSIBLE_DIRS[@]}"; do
        if [ -d "$dir" ] && [ -n "$(ls -A "$dir" 2>/dev/null)" ]; then
            PLUGIN_DIR_FOUND="$dir"
            echo "✅ 找到插件目录: $dir"
            break
        fi
    done
    
    if [ -n "$PLUGIN_DIR_FOUND" ]; then
        export SLURM_PLUGIN_DIR="$PLUGIN_DIR_FOUND"
    else
        echo "⚠️  未找到SLURM插件目录，使用默认路径"
        export SLURM_PLUGIN_DIR="/usr/lib/$(dpkg --print-architecture)/slurm"
    fi
else
    # 演示模式使用默认路径（优先使用slurm-wlm）
    arch_dpkg=$(dpkg --print-architecture)
    arch_gnu=""
    case "$arch_dpkg" in
        "amd64") arch_gnu="x86_64-linux-gnu" ;;
        "arm64") arch_gnu="aarch64-linux-gnu" ;;
        "armhf") arch_gnu="arm-linux-gnueabihf" ;;
        *) arch_gnu="$arch_dpkg-linux-gnu" ;;
    esac
    
    if [ -d "/usr/lib/$arch_gnu/slurm-wlm" ]; then
        export SLURM_PLUGIN_DIR="/usr/lib/$arch_gnu/slurm-wlm"
    elif [ -d "/usr/lib/$arch_gnu/slurm" ]; then
        export SLURM_PLUGIN_DIR="/usr/lib/$arch_gnu/slurm"
    elif [ -d "/usr/lib/$arch_dpkg/slurm-wlm" ]; then
        export SLURM_PLUGIN_DIR="/usr/lib/$arch_dpkg/slurm-wlm"
    else
        export SLURM_PLUGIN_DIR="/usr/lib/$arch_dpkg/slurm"
    fi
fi

echo "📋 SLURM配置摘要："
echo "  运行模式: $SLURM_MODE"
echo "  集群名称: $SLURM_CLUSTER_NAME"
echo "  控制器: $SLURM_CONTROLLER_HOST:$SLURM_CONTROLLER_PORT"
echo "  数据库: $SLURM_DB_HOST:$SLURM_DB_PORT/$SLURM_DB_NAME"
echo "  测试节点: $SLURM_TEST_NODES"
echo ""

# 函数：等待数据库服务可用
wait_for_database() {
    echo "⏳ 等待数据库服务可用..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        # 检查数据库类型并使用相应的检查方法
        if [ "$SLURM_DB_PORT" = "3306" ]; then
            # MySQL检查
            if nc -z "$SLURM_DB_HOST" "$SLURM_DB_PORT" >/dev/null 2>&1; then
                echo "✅ 数据库服务已可用"
                return 0
            fi
        else
            # PostgreSQL检查
            if pg_isready -h "$SLURM_DB_HOST" -p "$SLURM_DB_PORT" -U "$SLURM_DB_USER" >/dev/null 2>&1; then
                echo "✅ 数据库服务已可用"
                return 0
            fi
        fi
        
        echo "  尝试 $attempt/$max_attempts: 数据库未就绪，等待 5 秒..."
        sleep 5
        attempt=$((attempt + 1))
    done
    
    echo "❌ 数据库服务超时未响应"
    exit 1
}

# 函数：初始化数据库
init_database() {
    echo "🗄️ 初始化SLURM数据库..."
    
    if [ "$SLURM_DB_PORT" = "3306" ]; then
        # MySQL数据库初始化
        echo "  使用MySQL数据库初始化"
        # MySQL数据库和用户已在Docker启动时创建，这里只需验证连接
        # 通过调用后端初始化服务来创建SLURM数据库表
        if command -v mysql >/dev/null 2>&1; then
            mysql -h "$SLURM_DB_HOST" -P "$SLURM_DB_PORT" -u "$SLURM_DB_USER" -p"$SLURM_DB_PASSWORD" -e "SELECT 1;" >/dev/null 2>&1 || {
                echo "❌ MySQL连接失败"
                exit 1
            }
        fi
    else
        # PostgreSQL数据库初始化
        echo "  使用PostgreSQL数据库初始化"
        # 创建数据库（如果不存在）
        PGPASSWORD="$POSTGRES_PASSWORD" createdb -h "$SLURM_DB_HOST" -p "$SLURM_DB_PORT" -U "$POSTGRES_USER" "$SLURM_DB_NAME" 2>/dev/null || true
        
        # 创建SLURM数据库用户（如果不存在）
        PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$SLURM_DB_HOST" -p "$SLURM_DB_PORT" -U "$POSTGRES_USER" -d "$SLURM_DB_NAME" -c "
            DO \$\$
            BEGIN
                IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$SLURM_DB_USER') THEN
                    CREATE USER $SLURM_DB_USER WITH PASSWORD '$SLURM_DB_PASSWORD';
                END IF;
            END
            \$\$;
            GRANT ALL PRIVILEGES ON DATABASE $SLURM_DB_NAME TO $SLURM_DB_USER;
        " 2>/dev/null || echo "  数据库用户可能已存在"
    fi
    
    echo "✅ 数据库初始化完成"
}

# 函数：生成配置文件
generate_configs() {
    echo "📝 生成SLURM配置文件..."
    
    # 创建配置目录
    mkdir -p /etc/slurm
    
    # 生成slurm.conf
    envsubst < /etc/slurm-templates/slurm.conf.template > /etc/slurm/slurm.conf
    echo "  生成 slurm.conf"
    
    # 生成slurmdbd.conf
    envsubst < /etc/slurm-templates/slurmdbd.conf.template > /etc/slurm/slurmdbd.conf
    echo "  生成 slurmdbd.conf"
    
    # 生成cgroup.conf
    envsubst < /etc/slurm-templates/cgroup.conf.template > /etc/slurm/cgroup.conf
    echo "  生成 cgroup.conf"
    
    # 设置配置文件权限
    chown slurm:slurm /etc/slurm/slurm.conf /etc/slurm/cgroup.conf
    chown slurm:slurm /etc/slurm/slurmdbd.conf
    chmod 644 /etc/slurm/slurm.conf /etc/slurm/cgroup.conf
    chmod 600 /etc/slurm/slurmdbd.conf
    
    echo "✅ 配置文件生成完成"
}

# 函数：配置Munge认证
setup_munge() {
    echo "🔐 配置Munge认证服务..."
    
    # 生成或使用现有的Munge密钥
    if [ ! -f /etc/munge/munge.key ]; then
        echo "  生成新的Munge密钥..."
        echo -n "$SLURM_MUNGE_KEY" > /etc/munge/munge.key
    else
        echo "  使用现有的Munge密钥"
    fi
    
    # 设置密钥权限
    chown munge:munge /etc/munge/munge.key
    chmod 400 /etc/munge/munge.key
    
    echo "✅ Munge认证服务配置完成"
}

# 函数：信号处理
handle_signal() {
    echo "� 收到停止信号，正在关闭SLURM服务..."
    
    # supervisor会处理子进程的关闭
    supervisorctl -c /etc/supervisor/conf.d/slurm.conf shutdown
    
    echo "✅ SLURM服务已关闭"
    exit 0
}

# 设置信号处理
trap 'handle_signal' TERM INT

# 主函数
main() {
    case "${1:-start-services}" in
        start-services)
            if [ "$SLURM_MODE" = "full" ]; then
                wait_for_database
                init_database
                generate_configs
                setup_munge
                
                echo "🎯 启动SLURM服务 (使用supervisor)..."
                exec supervisord -c /etc/supervisor/conf.d/slurm.conf
            else
                echo "🎭 启动演示模式..."
                setup_munge
                
                echo "🎯 启动演示服务 (使用supervisor)..."
                exec supervisord -c /etc/supervisor/conf.d/slurm.conf
            fi
            ;;
        supervisord)
            # 旧版本兼容性，重定向到start-services
            wait_for_database
            init_database
            generate_configs
            setup_munge
            
            echo "🎯 启动SLURM服务 (使用supervisor)..."
            exec supervisord -c /etc/supervisor/conf.d/slurm.conf
            ;;
        generate-config)
            generate_configs
            echo "配置文件已生成完成"
            ;;
        test-connection)
            echo "测试数据库连接..."
            wait_for_database
            echo "数据库连接正常"
            ;;
        *)
            echo "用法: $0 {start-services|supervisord|generate-config|test-connection}"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"