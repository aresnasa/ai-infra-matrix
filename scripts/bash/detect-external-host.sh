#!/bin/bash

# 自动检测外部访问IP地址的脚本

# 检测本机可访问的IP地址
detect_host() {
    # 优先检测局域网IP
    local lan_ip=$(ip route get 8.8.8.8 2>/dev/null | grep -oP 'src \K[^ ]+' 2>/dev/null)
    if [[ -n "$lan_ip" && "$lan_ip" != "127.0.0.1" ]]; then
        echo "$lan_ip"
        return 0
    fi
    
    # macOS 环境检测
    if command -v route >/dev/null 2>&1; then
        local mac_ip=$(route get default 2>/dev/null | grep interface | awk '{print $2}' | xargs ifconfig 2>/dev/null | grep -E 'inet [0-9]' | grep -v '127.0.0.1' | head -1 | awk '{print $2}')
        if [[ -n "$mac_ip" ]]; then
            echo "$mac_ip"
            return 0
        fi
    fi
    
    # 降级到localhost
    echo "localhost"
}

# 检测主机
DETECTED_HOST=$(detect_host)
echo "检测到的主机地址: $DETECTED_HOST"

# 设置环境变量
export EXTERNAL_HOST="$DETECTED_HOST"
export JUPYTERHUB_PUBLIC_HOST="${DETECTED_HOST}:8080"

echo "设置 EXTERNAL_HOST=$EXTERNAL_HOST"
echo "设置 JUPYTERHUB_PUBLIC_HOST=$JUPYTERHUB_PUBLIC_HOST"
