#!/bin/bash

# 临时后端服务脚本 - 跳过JupyterHub相关功能
# 这样可以让基础的AI-Infra-Matrix服务先运行起来

echo "Starting AI-Infra-Matrix Backend (without JupyterHub integration)"
echo "Port: 8080"
echo "Database: PostgreSQL"
echo "Redis: Enabled"

# 模拟后端服务运行
while true; do
    echo "$(date): Backend service running..."
    sleep 30
done
