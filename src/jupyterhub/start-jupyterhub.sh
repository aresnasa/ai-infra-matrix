#!/bin/bash
set -e

echo "JupyterHub启动脚本 - AI基础设施矩阵"
echo "项目根目录: $(pwd)"

# 确保数据目录存在
mkdir -p /srv/data/jupyterhub

# 设置数据库路径
DB_PATH="/srv/data/jupyterhub/jupyterhub.sqlite"

# 如果数据库文件存在，尝试升级
if [ -f "$DB_PATH" ]; then
    echo "找到现有数据库，尝试升级..."
    jupyterhub upgrade-db --db-url="sqlite:///$DB_PATH" || {
        echo "数据库升级失败，重新创建..."
        rm -f "$DB_PATH"
    }
fi

# 如果数据库不存在，创建新的
if [ ! -f "$DB_PATH" ]; then
    echo "初始化新数据库..."
    # 先创建数据库
    touch "$DB_PATH"
    # 运行升级以创建正确的模式
    jupyterhub upgrade-db --db-url="sqlite:///$DB_PATH"
fi

echo "启动JupyterHub..."
exec jupyterhub -f /srv/jupyterhub/config/ai_infra_jupyterhub_config.py
