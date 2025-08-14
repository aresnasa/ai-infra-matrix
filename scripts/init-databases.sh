#!/bin/bash
set -e

# PostgreSQL 多数据库初始化脚本
# 为不同的服务创建独立的数据库

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- 创建 JupyterHub 数据库
    SELECT 'CREATE DATABASE jupyterhub_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'jupyterhub_db')\gexec
    
    -- 创建 Gitea 用户与数据库（若不存在）
    DO 
    $$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'gitea') THEN
            CREATE USER gitea WITH LOGIN PASSWORD 'gitea-password';
        END IF;
    END
    $$;
    
    SELECT 'CREATE DATABASE gitea OWNER gitea' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gitea')\gexec
    
    GRANT ALL PRIVILEGES ON DATABASE gitea TO gitea;
    
    -- 创建后端数据库（如果需要单独的数据库）
    -- ansible_playbook_generator 已经是默认数据库，由 POSTGRES_DB 环境变量指定
    
    -- 显示创建的数据库
    \l
EOSQL

echo "✅ 数据库初始化完成"
echo "📊 已创建数据库:"
echo "  - ansible_playbook_generator (后端服务)"  
echo "  - jupyterhub_db (JupyterHub服务)"
echo "  - gitea (Gitea服务)"
