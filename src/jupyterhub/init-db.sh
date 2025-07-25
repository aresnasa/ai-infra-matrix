#!/bin/bash

echo "=== JupyterHub 数据库初始化脚本 ==="

# 确保数据目录存在
mkdir -p /srv/data/jupyterhub

# 备份现有数据库（如果存在）
if [ -f "/srv/data/jupyterhub/jupyterhub.sqlite" ]; then
    echo "备份现有数据库..."
    cp /srv/data/jupyterhub/jupyterhub.sqlite /srv/data/jupyterhub/jupyterhub.sqlite.backup.$(date +%Y%m%d_%H%M%S)
fi

# 升级数据库模式
echo "升级JupyterHub数据库模式..."
jupyterhub upgrade-db --config=/srv/jupyterhub/config/ai_infra_jupyterhub_config.py

if [ $? -eq 0 ]; then
    echo "✅ 数据库升级成功"
else
    echo "❌ 数据库升级失败"
    exit 1
fi

echo "=== 数据库初始化完成 ==="
