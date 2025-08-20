#!/bin/bash

# JupyterHub 目录整理脚本
echo "🔧 整理 JupyterHub 目录..."

# 创建归档目录
mkdir -p archive/jupyterhub_archive/{configs,dockerfiles,scripts,sql}

echo "📁 创建 JupyterHub 归档目录结构完成"

# 进入 src/jupyterhub 目录
cd src/jupyterhub

echo "📋 当前 src/jupyterhub 目录内容:"
ls -la

# 1. 移动各种实验性配置文件到归档
echo "📦 归档实验性配置文件..."
mv absolute_no_redirect_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv anti_redirect_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv clean_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv minimal_fix_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv no_redirect_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv simple_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv ultimate_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv unified_config.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true
mv unified_config_simple.py ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true

# 2. 移动 Dockerfile 到归档
echo "🐳 归档 Dockerfile..."
mv Dockerfile ../../archive/jupyterhub_archive/dockerfiles/ 2>/dev/null || true
mv Dockerfile.unified ../../archive/jupyterhub_archive/dockerfiles/ 2>/dev/null || true

# 3. 移动脚本到归档
echo "📜 归档脚本文件..."
mv start-unified.sh ../../archive/jupyterhub_archive/scripts/ 2>/dev/null || true

# 4. 处理 SQL 文件 - 创建统一的数据库初始化文件
echo "🗄️  统一SQL文件..."
cat > database_init.sql << 'EOF'
-- =================================================
-- AI Infrastructure Matrix - 数据库初始化脚本
-- =================================================
-- 创建时间: 2025-08-05
-- 说明: 统一的数据库初始化和配置脚本
-- =================================================

-- JupyterHub 数据库初始化
-- 基于原始的 init-jupyterhub-db.sql

-- 创建 JupyterHub 数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS jupyterhub_db;
CREATE USER IF NOT EXISTS 'jupyterhub'@'%' IDENTIFIED BY 'jupyterhub_password';
GRANT ALL PRIVILEGES ON jupyterhub_db.* TO 'jupyterhub'@'%';

-- 切换到 JupyterHub 数据库
USE jupyterhub_db;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    admin BOOLEAN DEFAULT FALSE,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP,
    cookie_id VARCHAR(255),
    state JSON
);

-- 服务器表
CREATE TABLE IF NOT EXISTS spawners (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    server_id INTEGER,
    state JSON,
    name VARCHAR(255),
    started TIMESTAMP,
    last_activity TIMESTAMP,
    user_options JSON
);

-- API 令牌表
CREATE TABLE IF NOT EXISTS api_tokens (
    id SERIAL PRIMARY KEY,
    hashed_token VARCHAR(255) UNIQUE,
    prefix VARCHAR(16),
    user_id INTEGER REFERENCES users(id),
    service_id INTEGER,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP,
    expires_at TIMESTAMP,
    note TEXT
);

-- OAuth 代码表
CREATE TABLE IF NOT EXISTS oauth_codes (
    id SERIAL PRIMARY KEY,
    client_id VARCHAR(255),
    code VARCHAR(255) UNIQUE,
    expires_at TIMESTAMP,
    redirect_uri VARCHAR(1024),
    user_id INTEGER REFERENCES users(id),
    session_id VARCHAR(255)
);

-- 服务表
CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE,
    admin BOOLEAN DEFAULT FALSE,
    api_token VARCHAR(255)
);

-- 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
CREATE INDEX IF NOT EXISTS idx_users_cookie_id ON users(cookie_id);
CREATE INDEX IF NOT EXISTS idx_users_last_activity ON users(last_activity);
CREATE INDEX IF NOT EXISTS idx_spawners_user_id ON spawners(user_id);
CREATE INDEX IF NOT EXISTS idx_spawners_name ON spawners(name);
CREATE INDEX IF NOT EXISTS idx_api_tokens_hashed_token ON api_tokens(hashed_token);
CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_codes_code ON oauth_codes(code);
CREATE INDEX IF NOT EXISTS idx_oauth_codes_client_id ON oauth_codes(client_id);

-- 插入默认管理员用户（可选）
INSERT IGNORE INTO users (name, admin, created) 
VALUES ('admin', TRUE, CURRENT_TIMESTAMP);

-- =================================================
-- 扩展配置区域
-- =================================================

-- 后端集成相关表（如果需要）
-- 这里可以添加与后端 API 集成相关的表结构

-- 用户权限扩展表
CREATE TABLE IF NOT EXISTS user_permissions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    permission_name VARCHAR(255),
    resource_type VARCHAR(255),
    resource_id VARCHAR(255),
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

-- 用户组表
CREATE TABLE IF NOT EXISTS user_groups (
    id SERIAL PRIMARY KEY,
    group_name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 用户组成员表
CREATE TABLE IF NOT EXISTS user_group_members (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    group_id INTEGER REFERENCES user_groups(id),
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, group_id)
);

-- 资源配额表
CREATE TABLE IF NOT EXISTS resource_quotas (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    cpu_limit DECIMAL(5,2),
    memory_limit_mb INTEGER,
    storage_limit_gb INTEGER,
    gpu_limit INTEGER DEFAULT 0,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 使用统计表
CREATE TABLE IF NOT EXISTS usage_stats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    session_start TIMESTAMP,
    session_end TIMESTAMP,
    cpu_hours DECIMAL(10,4),
    memory_mb_hours DECIMAL(15,4),
    storage_gb_hours DECIMAL(15,4),
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建扩展表的索引
CREATE INDEX IF NOT EXISTS idx_user_permissions_user_id ON user_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_resource ON user_permissions(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_user_group_members_user_id ON user_group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_user_group_members_group_id ON user_group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_resource_quotas_user_id ON resource_quotas(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_stats_user_id ON usage_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_stats_session_start ON usage_stats(session_start);

-- =================================================
-- 初始化数据
-- =================================================

-- 创建默认用户组
INSERT IGNORE INTO user_groups (group_name, description) VALUES 
('administrators', '管理员组'),
('power_users', '高级用户组'),
('standard_users', '标准用户组'),
('guests', '访客用户组');

-- 为管理员用户添加默认权限
INSERT IGNORE INTO user_group_members (user_id, group_id)
SELECT u.id, g.id FROM users u, user_groups g 
WHERE u.name = 'admin' AND g.group_name = 'administrators';

-- 设置默认资源配额
INSERT IGNORE INTO resource_quotas (user_id, cpu_limit, memory_limit_mb, storage_limit_gb, gpu_limit)
SELECT id, 4.0, 8192, 50, 1 FROM users WHERE name = 'admin';

-- =================================================
-- 数据库配置完成
-- =================================================

FLUSH PRIVILEGES;

-- 显示创建的表
SHOW TABLES;

EOF

# 移动原始 SQL 文件到归档
mv init-jupyterhub-db.sql ../../archive/jupyterhub_archive/sql/ 2>/dev/null || true

# 5. 整理requirements文件
echo "📋 归档多余的requirements文件..."
mv requirements-unified.txt ../../archive/jupyterhub_archive/configs/ 2>/dev/null || true

# 返回项目根目录
cd ../..

echo ""
echo "✨ JupyterHub 目录整理完成！"
echo ""
echo "📁 保留的核心文件:"
echo "src/jupyterhub/"
echo "├── backend_integrated_config.py    # ✨ 主要配置文件"
echo "├── postgres_authenticator.py       # ✨ PostgreSQL 认证器"
echo "├── ai_infra_jupyterhub_config.py   # ✨ AI基础设施配置"
echo "├── jupyterhub_config.py            # ✨ 标准配置文件"
echo "├── requirements.txt                # ✨ Python依赖"
echo "├── cookie_secret                   # ✨ 会话密钥"
echo "└── database_init.sql               # ✨ 统一数据库初始化脚本"
echo ""
echo "🗃️ 归档内容:"
echo "archive/jupyterhub_archive/"
echo "├── configs/                        # 实验性配置文件"
echo "├── dockerfiles/                    # Docker构建文件"
echo "├── scripts/                        # 部署脚本"
echo "└── sql/                           # 原始SQL文件"
echo ""
echo "🎯 现在 JupyterHub 配置更加简洁和专业！"
