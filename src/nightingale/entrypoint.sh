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

# 启动脚本服务器
start_script_server

# 启动 Nightingale (前台运行)
# 使用 -configs 参数指定配置目录
echo "Starting Nightingale with configs from ${N9E_CONFIGS}..."
exec /app/n9e -configs "${N9E_CONFIGS}" "$@"
