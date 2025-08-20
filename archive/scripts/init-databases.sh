#!/bin/bash
set -e

# PostgreSQL 多数据库初始化脚本
# 为不同的服务创建独立的数据库

echo "🚀 开始 PostgreSQL 数据库初始化..."
echo "👤 当前用户: $POSTGRES_USER"
echo "🗄️  默认数据库: $POSTGRES_DB"

# 等待PostgreSQL完全启动
echo "⏳ 等待 PostgreSQL 服务完全就绪..."
until pg_isready -h localhost -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -q; do
    echo "PostgreSQL 还未就绪，等待中..."
    sleep 2
done

echo "✅ PostgreSQL 服务已就绪，开始创建数据库..."

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- 创建 JupyterHub 数据库
    SELECT 'CREATE DATABASE jupyterhub_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'jupyterhub_db')\gexec
    
    -- 创建 Gitea 用户与数据库（若不存在）
    DO 
    \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'gitea') THEN
            CREATE USER gitea WITH LOGIN PASSWORD 'gitea-password';
            RAISE NOTICE '✅ 创建用户 gitea';
        ELSE
            RAISE NOTICE 'ℹ️  用户 gitea 已存在';
        END IF;
    END
    \$\$;
    
    SELECT 'CREATE DATABASE gitea OWNER gitea' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gitea')\gexec
    
    GRANT ALL PRIVILEGES ON DATABASE gitea TO gitea;
    
    -- 验证数据库创建
    SELECT datname as "已创建的数据库" FROM pg_database WHERE datname IN ('${POSTGRES_DB}', 'jupyterhub_db', 'gitea');
EOSQL

echo ""
echo "✅ 数据库初始化完成"
echo "📊 已创建数据库:"
echo "  - ${POSTGRES_DB} (后端主服务)"  
echo "  - jupyterhub_db (JupyterHub服务)"
echo "  - gitea (Gitea服务，用户: gitea)"
echo ""
