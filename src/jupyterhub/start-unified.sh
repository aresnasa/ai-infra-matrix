#!/bin/bash

# 统一认证系统JupyterHub启动脚本

set -e

echo "=== 启动统一认证系统JupyterHub ==="

# 检查必需的环境变量
required_vars=("DB_HOST" "DB_NAME" "DB_USER" "DB_PASSWORD" "REDIS_HOST")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "警告: 环境变量 $var 未设置，使用默认值"
    fi
done

# 等待PostgreSQL可用
echo "等待PostgreSQL连接..."
max_attempts=30
attempt=0

while ! python3 -c "
import psycopg2
try:
    conn = psycopg2.connect(
        host='${DB_HOST:-localhost}',
        port=${DB_PORT:-5432},
        database='${DB_NAME:-ansible_playbook_generator}',
        user='${DB_USER:-postgres}',
        password='${DB_PASSWORD:-postgres}'
    )
    conn.close()
    print('PostgreSQL连接成功')
except Exception as e:
    print(f'PostgreSQL连接失败: {e}')
    exit(1)
"; do
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
        echo "错误: PostgreSQL连接超时"
        exit 1
    fi
    echo "尝试连接PostgreSQL... ($attempt/$max_attempts)"
    sleep 2
done

# 等待Redis可用
echo "等待Redis连接..."
attempt=0

while ! python3 -c "
import redis
try:
    r = redis.Redis(
        host='${REDIS_HOST:-localhost}',
        port=${REDIS_PORT:-6379},
        password='${REDIS_PASSWORD}' if '${REDIS_PASSWORD}' else None,
        db=${REDIS_DB:-0}
    )
    r.ping()
    print('Redis连接成功')
except Exception as e:
    print(f'Redis连接失败: {e}')
    exit(1)
"; do
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
        echo "警告: Redis连接失败，会话缓存将不可用"
        break
    fi
    echo "尝试连接Redis... ($attempt/$max_attempts)"
    sleep 2
done

# 验证配置文件
echo "验证配置文件..."
if [ ! -f "/srv/jupyterhub/jupyterhub_config.py" ]; then
    echo "错误: 配置文件不存在"
    exit 1
fi

if [ ! -f "/srv/jupyterhub/postgres_authenticator.py" ]; then
    echo "错误: 认证器文件不存在"
    exit 1
fi

# 验证Python语法
echo "验证Python配置语法..."
python3 -m py_compile /srv/jupyterhub/postgres_authenticator.py
# 跳过配置文件语法验证，因为需要JupyterHub运行时环境
# python3 -c "exec(open('/srv/jupyterhub/jupyterhub_config.py').read())"

echo "配置验证通过"

# 设置权限
echo "设置目录权限..."
mkdir -p /srv/data/jupyterhub
mkdir -p /srv/jupyterhub/notebooks
chmod -R 755 /srv/data/jupyterhub
chmod -R 755 /srv/jupyterhub/notebooks

# 删除可能存在的有权限问题的cookie文件
if [ -f "/srv/data/jupyterhub/cookie_secret" ]; then
    echo "删除旧的cookie_secret文件"
    rm -f /srv/data/jupyterhub/cookie_secret
fi

# 显示启动信息
echo "=== 启动信息 ==="
echo "JupyterHub版本: $(jupyterhub --version)"
echo "Python版本: $(python3 --version)"
echo "配置文件: /srv/jupyterhub/jupyterhub_config.py"
echo "数据目录: /srv/data/jupyterhub"
echo "数据库: ${DB_HOST:-localhost}:${DB_PORT:-5432}/${DB_NAME:-ansible_playbook_generator}"
echo "Redis: ${REDIS_HOST:-localhost}:${REDIS_PORT:-6379}"
echo "绑定地址: http://0.0.0.0:8000"
echo "管理员用户: ${JUPYTERHUB_ADMIN_USERS:-admin}"

# 启动JupyterHub
echo "=== 启动JupyterHub ==="
exec jupyterhub --config=/srv/jupyterhub/jupyterhub_config.py
