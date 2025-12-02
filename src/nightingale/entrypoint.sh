#!/bin/bash
# Nightingale 入口点脚本
# 同时启动 Nightingale 服务和脚本服务器

set -e

# 脚本服务端口
SCRIPT_SERVER_PORT="${SCRIPT_SERVER_PORT:-17002}"

# 配置目录 (可通过 N9E_CONFIGS 环境变量覆盖)
N9E_CONFIGS="${N9E_CONFIGS:-/app/etc}"

echo "Starting Nightingale entrypoint..."
echo "Config directory: ${N9E_CONFIGS}"

# 动态替换配置文件中的环境变量
# 支持的环境变量:
#   POSTGRES_PASSWORD - PostgreSQL 密码
#   POSTGRES_USER - PostgreSQL 用户名 (默认: postgres)
#   POSTGRES_HOST - PostgreSQL 主机 (默认: postgres)
#   POSTGRES_PORT - PostgreSQL 端口 (默认: 5432)
#   POSTGRES_DB - 数据库名 (默认: nightingale)
#   REDIS_PASSWORD - Redis 密码
#   REDIS_HOST - Redis 主机 (默认: redis)
#   REDIS_PORT - Redis 端口 (默认: 6379)
update_config() {
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
    
    # 使用 sed 替换配置 (创建临时文件避免权限问题)
    local tmp_config="/tmp/config.toml"
    cp "$config_file" "$tmp_config"
    
    # 替换 PostgreSQL DSN
    sed -i "s|DSN=\"host=postgres port=5432 user=postgres dbname=nightingale password=your-postgres-password sslmode=disable\"|DSN=\"${pg_dsn}\"|g" "$tmp_config"
    
    # 替换 Redis 配置
    sed -i "s|Address = \"redis:6379\"|Address = \"${redis_addr}\"|g" "$tmp_config"
    sed -i "s|Password = \"your-redis-password\"|Password = \"${redis_pass}\"|g" "$tmp_config"
    
    # 复制回原位置
    cp "$tmp_config" "$config_file"
    rm -f "$tmp_config"
    
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

# 更新配置文件
update_config

# 启动脚本服务器
start_script_server

# 启动 Nightingale (前台运行)
# 使用 -configs 参数指定配置目录
echo "Starting Nightingale with configs from ${N9E_CONFIGS}..."
exec /app/n9e -configs "${N9E_CONFIGS}" "$@"
