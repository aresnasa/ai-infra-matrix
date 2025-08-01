#!/bin/bash
# JupyterHub PostgreSQL 迁移脚本

echo "开始 JupyterHub 数据库迁移..."

# 1. 停止当前 JupyterHub 服务
echo "停止 JupyterHub 服务..."
docker stop ai-infra-jupyterhub || true

# 2. 备份现有数据库（如果存在）
echo "备份现有数据库..."
if [ -f "/data/jupyterhub/jupyterhub.sqlite" ]; then
    cp /data/jupyterhub/jupyterhub.sqlite /data/jupyterhub/jupyterhub.sqlite.backup.$(date +%Y%m%d_%H%M%S)
    echo "✅ SQLite 数据库已备份"
fi

# 3. 确保 PostgreSQL 和 Redis 运行
echo "启动数据库服务..."
docker-compose up -d ai-infra-postgres ai-infra-redis

# 等待服务启动
echo "等待数据库服务就绪..."
sleep 10

# 4. 创建 JupyterHub 数据库
echo "创建 JupyterHub 数据库..."
docker exec ai-infra-postgres psql -U ai_user -d ai_infra -c "CREATE DATABASE jupyterhub;" || true

# 5. 使用新镜像重建 JupyterHub
echo "重建 JupyterHub 容器..."
docker-compose up -d ai-infra-jupyterhub

echo "✅ 数据库迁移完成！"
echo "JupyterHub 现在使用 PostgreSQL + Redis 统一后端"
