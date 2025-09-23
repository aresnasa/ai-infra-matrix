#!/bin/bash
# AI Infrastructure Matrix - SLURM数据库初始化脚本

set -e

# 环境变量配置
DB_HOST=${SLURM_DB_HOST:-postgres}
DB_PORT=${SLURM_DB_PORT:-5432}
DB_NAME=${SLURM_DB_NAME:-slurm_acct_db}
DB_USER=${SLURM_DB_USER:-slurm}
DB_PASSWORD=${SLURM_DB_PASSWORD:-slurm123}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-postgres123}
CLUSTER_NAME=${SLURM_CLUSTER_NAME:-ai-infra-cluster}

echo "🗄️ SLURM数据库初始化脚本"
echo "数据库主机: $DB_HOST:$DB_PORT"
echo "数据库名称: $DB_NAME"
echo "SLURM用户: $DB_USER"
echo "集群名称: $CLUSTER_NAME"
echo ""

# 函数：等待数据库可用
wait_for_postgres() {
    echo "⏳ 等待PostgreSQL服务可用..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if pg_isready -h "$DB_HOST" -p "$DB_PORT" >/dev/null 2>&1; then
            echo "✅ PostgreSQL服务已可用"
            return 0
        fi
        
        echo "  尝试 $attempt/$max_attempts: PostgreSQL未就绪，等待 5 秒..."
        sleep 5
        attempt=$((attempt + 1))
    done
    
    echo "❌ PostgreSQL服务超时未响应"
    exit 1
}

# 函数：创建数据库和用户
create_database_and_user() {
    echo "🔧 创建SLURM数据库和用户..."
    
    # 创建数据库
    echo "  创建数据库: $DB_NAME"
    PGPASSWORD="$POSTGRES_PASSWORD" createdb \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$POSTGRES_USER" \
        "$DB_NAME" \
        --encoding=UTF8 \
        --lc-collate=C \
        --lc-ctype=C \
        --template=template0 \
        2>/dev/null || echo "    数据库可能已存在"
    
    # 创建SLURM用户
    echo "  创建用户: $DB_USER"
    PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$POSTGRES_USER" \
        -d postgres \
        -c "DO \$\$
            BEGIN
                IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER') THEN
                    CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
                    RAISE NOTICE 'User $DB_USER created';
                ELSE
                    RAISE NOTICE 'User $DB_USER already exists';
                END IF;
            END
            \$\$;" 2>/dev/null || echo "    用户创建操作可能失败"
    
    # 授予权限
    echo "  授予数据库权限"
    PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$POSTGRES_USER" \
        -d "$DB_NAME" \
        -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
            GRANT ALL PRIVILEGES ON SCHEMA public TO $DB_USER;
            GRANT CREATE ON SCHEMA public TO $DB_USER;" 2>/dev/null || echo "    权限授予可能失败"
    
    echo "✅ 数据库和用户创建完成"
}

# 函数：测试数据库连接
test_connection() {
    echo "🔍 测试SLURM数据库连接..."
    
    # 使用SLURM用户连接数据库
    if PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -c "SELECT version();" >/dev/null 2>&1; then
        echo "✅ SLURM数据库连接测试成功"
    else
        echo "❌ SLURM数据库连接测试失败"
        exit 1
    fi
}

# 函数：初始化SLURM表结构
init_slurm_tables() {
    echo "📊 初始化SLURM表结构..."
    
    # 检查slurmdbd是否运行
    local slurmdbd_ready=false
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ] && [ "$slurmdbd_ready" = "false" ]; do
        if pgrep -f slurmdbd >/dev/null 2>&1; then
            # 等待slurmdbd完全启动
            sleep 5
            if timeout 10 sacctmgr -i list cluster 2>/dev/null | grep -q "$CLUSTER_NAME"; then
                echo "  集群 $CLUSTER_NAME 已存在于数据库中"
                slurmdbd_ready=true
            elif timeout 10 sacctmgr -i add cluster "$CLUSTER_NAME" 2>/dev/null; then
                echo "  集群 $CLUSTER_NAME 已添加到数据库"
                slurmdbd_ready=true
            else
                echo "  尝试 $attempt/$max_attempts: 等待slurmdbd服务就绪..."
                sleep 5
                attempt=$((attempt + 1))
            fi
        else
            echo "  尝试 $attempt/$max_attempts: slurmdbd未运行，等待启动..."
            sleep 5
            attempt=$((attempt + 1))
        fi
    done
    
    if [ "$slurmdbd_ready" = "false" ]; then
        echo "⚠️  无法通过sacctmgr初始化，但数据库基础结构已准备就绪"
        echo "    slurmdbd启动后会自动创建必要的表结构"
    else
        echo "✅ SLURM表结构初始化完成"
    fi
}

# 函数：创建基础账户结构
setup_accounts() {
    echo "👥 设置基础账户结构..."
    
    # 等待sacctmgr可用
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if timeout 10 sacctmgr -i list cluster 2>/dev/null | grep -q "$CLUSTER_NAME"; then
            break
        fi
        echo "  尝试 $attempt/$max_attempts: 等待sacctmgr可用..."
        sleep 5
        attempt=$((attempt + 1))
    done
    
    if [ $attempt -gt $max_attempts ]; then
        echo "⚠️  sacctmgr暂时不可用，跳过账户设置"
        echo "    可以稍后手动运行账户初始化"
        return 0
    fi
    
    # 创建默认账户
    echo "  创建默认账户..."
    timeout 10 sacctmgr -i add account ai-infra description="AI Infrastructure Default Account" 2>/dev/null || echo "    账户可能已存在"
    
    # 创建默认用户
    echo "  创建默认用户..."
    timeout 10 sacctmgr -i create user root defaultaccount=ai-infra 2>/dev/null || echo "    用户可能已存在"
    timeout 10 sacctmgr -i create user admin defaultaccount=ai-infra adminlevel=admin 2>/dev/null || echo "    管理员用户可能已存在"
    
    echo "✅ 基础账户结构设置完成"
}

# 函数：显示数据库状态
show_status() {
    echo ""
    echo "📈 SLURM数据库状态报告"
    echo "========================"
    
    # 数据库连接状态
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\q" >/dev/null 2>&1; then
        echo "数据库连接: ✅ 正常"
        
        # 表统计
        local table_count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | xargs)
        echo "数据库表数: ${table_count:-未知}"
        
    else
        echo "数据库连接: ❌ 异常"
    fi
    
    # SLURM集群状态
    if timeout 5 sacctmgr -n -P list cluster 2>/dev/null; then
        echo "SLURM集群: ✅ 已配置"
        echo "集群列表:"
        timeout 5 sacctmgr -n -P list cluster 2>/dev/null | while read line; do
            echo "  - $line"
        done
    else
        echo "SLURM集群: ⚠️  配置待完成"
    fi
    
    echo "========================"
}

# 主执行函数
main() {
    case "${1:-init}" in
        init)
            wait_for_postgres
            create_database_and_user
            test_connection
            echo ""
            echo "✅ SLURM数据库初始化完成！"
            echo ""
            echo "📝 后续步骤:"
            echo "  1. 启动slurmdbd服务"
            echo "  2. 运行: $0 setup-tables"
            echo "  3. 运行: $0 setup-accounts"
            ;;
        setup-tables)
            wait_for_postgres
            test_connection
            init_slurm_tables
            ;;
        setup-accounts)
            wait_for_postgres
            test_connection
            setup_accounts
            ;;
        full-setup)
            wait_for_postgres
            create_database_and_user
            test_connection
            init_slurm_tables
            setup_accounts
            show_status
            echo ""
            echo "🎉 SLURM数据库完整设置完成！"
            ;;
        test)
            wait_for_postgres
            test_connection
            echo "✅ 数据库连接测试通过"
            ;;
        status)
            show_status
            ;;
        *)
            echo "用法: $0 {init|setup-tables|setup-accounts|full-setup|test|status}"
            echo ""
            echo "  init           - 初始化数据库和用户"
            echo "  setup-tables   - 设置SLURM表结构（需要slurmdbd运行）"
            echo "  setup-accounts - 设置基础账户结构"
            echo "  full-setup     - 完整设置（包含以上所有步骤）"
            echo "  test          - 测试数据库连接"
            echo "  status        - 显示数据库状态"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"