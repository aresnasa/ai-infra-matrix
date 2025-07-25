#!/bin/bash

# JupyterHub快速测试部署脚本

set -e

echo "=== JupyterHub 快速测试部署 ==="

# 清理现有的JupyterHub卷
echo "清理现有数据..."
docker volume rm src_jupyterhub_data src_jupyterhub_logs src_jupyterhub_notebooks 2>/dev/null || true

# 使用基础配置运行JupyterHub测试
echo "启动JupyterHub测试容器..."
docker run -d \
  --name jupyterhub-test \
  -p 8088:8000 \
  -v $(pwd)/jupyterhub:/srv/config:ro \
  -e JUPYTERHUB_ADMIN_USERS=admin \
  -e CONFIGPROXY_AUTH_TOKEN=test-token \
  ai-infra-jupyterhub-integrated:latest \
  bash -c "
    mkdir -p /srv/data/jupyterhub
    echo 'Starting JupyterHub with basic config...'
    jupyterhub -f /srv/config/basic_config.py
  "

echo "等待启动..."
sleep 10

echo "检查容器状态..."
docker ps | grep jupyterhub-test

echo "检查日志..."
docker logs jupyterhub-test | tail -20

echo ""
echo "=== JupyterHub 测试部署完成 ==="
echo "访问地址: http://localhost:8088"
echo "用户名: admin (或任意用户名)"
echo "密码: test123"
