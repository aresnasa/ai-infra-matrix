#!/bin/bash
# Nightingale 入口点脚本
# 同时启动 Nightingale 服务和脚本服务器

set -e

# 脚本服务端口
SCRIPT_SERVER_PORT="${SCRIPT_SERVER_PORT:-17002}"

# 配置目录
# 原始配置目录 (只读挂载)
N9E_CONFIGS_SRC="${N9E_CONFIGS_SRC:-/app/etc}"
# 运行时配置目录 (可写)
N9E_CONFIGS="${N9E_CONFIGS:-/app/etc-runtime}"

echo "Starting Nightingale entrypoint..."
echo "Source config directory: ${N9E_CONFIGS_SRC}"
echo "Runtime config directory: ${N9E_CONFIGS}"

# 修复数据库中的 JSON 字段（防止 "unexpected end of JSON input" 错误）
fix_json_fields() {
    local pg_host="${POSTGRES_HOST:-postgres}"
    local pg_port="${POSTGRES_PORT:-5432}"
    local pg_user="${POSTGRES_USER:-postgres}"
    local pg_pass="${POSTGRES_PASSWORD:-}"
    local pg_db="${N9E_DB_NAME:-nightingale}"
    
    echo "Checking and fixing JSON fields in database..."
    
    # 等待 PostgreSQL 可用
    local max_retries=30
    local retry=0
    while [ $retry -lt $max_retries ]; do
        if PGPASSWORD="${pg_pass}" psql -h "${pg_host}" -p "${pg_port}" -U "${pg_user}" -d "${pg_db}" -c '\q' 2>/dev/null; then
            break
        fi
        retry=$((retry + 1))
        echo "Waiting for PostgreSQL... ($retry/$max_retries)"
        sleep 2
    done
    
    if [ $retry -eq $max_retries ]; then
        echo "WARNING: PostgreSQL not available, skipping JSON field fix"
        return 0
    fi
    
    # 修复 users 表的 contacts 字段（将 NULL 或空字符串转换为空 JSON 对象）
    PGPASSWORD="${pg_pass}" psql -h "${pg_host}" -p "${pg_port}" -U "${pg_user}" -d "${pg_db}" <<-EOSQL 2>/dev/null || true
        UPDATE users SET contacts = '{}' WHERE contacts IS NULL OR contacts = '';
EOSQL
    
    echo "JSON fields check complete"
}

# 复制配置文件到可写目录并替换环境变量
prepare_config() {
    # 创建运行时配置目录
    mkdir -p "${N9E_CONFIGS}"
    
    # 复制所有配置文件
    cp -r "${N9E_CONFIGS_SRC}"/* "${N9E_CONFIGS}/"
    
    local config_file="${N9E_CONFIGS}/config.toml"
    
    if [ ! -f "$config_file" ]; then
        echo "ERROR: Config file not found: $config_file"
        return 1
    fi
    
    echo "Updating config with environment variables..."
    
    # PostgreSQL 配置
    local pg_host="${POSTGRES_HOST:-postgres}"
    local pg_port="${POSTGRES_PORT:-5432}"
    local pg_user="${POSTGRES_USER:-postgres}"
    local pg_pass="${POSTGRES_PASSWORD:-}"
    local pg_db="${N9E_DB_NAME:-nightingale}"
    
    # Redis 配置
    local redis_host="${REDIS_HOST:-redis}"
    local redis_port="${REDIS_PORT:-6379}"
    local redis_pass="${REDIS_PASSWORD:-}"
    
    # 构建 DSN
    local pg_dsn="host=${pg_host} port=${pg_port} user=${pg_user} dbname=${pg_db} password=${pg_pass} sslmode=disable"
    local redis_addr="${redis_host}:${redis_port}"
    
    # 使用 sed 替换配置
    sed -i "s|DSN=\"host=postgres port=5432 user=postgres dbname=nightingale password=your-postgres-password sslmode=disable\"|DSN=\"${pg_dsn}\"|g" "$config_file"
    
    # 替换 Redis 配置
    sed -i "s|Address = \"redis:6379\"|Address = \"${redis_addr}\"|g" "$config_file"
    sed -i "s|Password = \"your-redis-password\"|Password = \"${redis_pass}\"|g" "$config_file"
    
    echo "Config updated successfully"
    echo "  PostgreSQL: ${pg_host}:${pg_port}/${pg_db} (user: ${pg_user})"
    echo "  Redis: ${redis_addr}"
}

# 启动简单的 HTTP 服务器提供脚本下载 (使用 Python 内置 HTTP 服务器)
# 脚本位于 /app/scripts/ 目录
start_script_server() {
    echo "Starting script server on port ${SCRIPT_SERVER_PORT}..."
    cd /app/scripts
    python3 -m http.server ${SCRIPT_SERVER_PORT} &
    SCRIPT_SERVER_PID=$!
    echo "Script server started with PID ${SCRIPT_SERVER_PID}"
}

# 清理函数
cleanup() {
    echo "Stopping services..."
    if [ -n "${SCRIPT_SERVER_PID}" ]; then
        kill ${SCRIPT_SERVER_PID} 2>/dev/null || true
    fi
    exit 0
}

trap cleanup SIGTERM SIGINT

# 准备配置文件 (复制到可写目录并替换变量)
prepare_config

# 启动脚本服务器
start_script_server

# 启动 Nightingale (前台运行)
# 使用 -configs 参数指定运行时配置目录
echo "Starting Nightingale with configs from ${N9E_CONFIGS}..."
exec /app/n9e -configs "${N9E_CONFIGS}" "$@"
