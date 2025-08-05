-- =================================================
-- AI Infrastructure Matrix - JupyterHub数据库初始化脚本
-- =================================================
-- 创建时间: 2025-08-05
-- 说明: JupyterHub PostgreSQL数据库表结构初始化
-- =================================================

-- 注意：此脚本假设数据库 jupyterhub_db 已经存在
-- 通过 docker-entrypoint-initdb.d 机制自动执行

-- 连接到 JupyterHub 数据库
\c jupyterhub_db;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    admin BOOLEAN DEFAULT FALSE,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP,
    cookie_id VARCHAR(255),
    state JSONB,
    encrypted_auth_state TEXT
);

-- 服务器表
CREATE TABLE IF NOT EXISTS spawners (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    server_id INTEGER,
    state JSONB,
    name VARCHAR(255),
    started TIMESTAMP,
    last_activity TIMESTAMP,
    user_options JSONB
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
INSERT INTO users (name, admin, created) 
VALUES ('admin', TRUE, CURRENT_TIMESTAMP)
ON CONFLICT (name) DO NOTHING;

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
    user_id INTEGER UNIQUE REFERENCES users(id),
    cpu_limit DECIMAL(5,2),
    memory_limit_mb INTEGER,
    storage_limit_gb INTEGER,
    gpu_limit INTEGER DEFAULT 0,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
INSERT INTO user_groups (group_name, description) VALUES 
('administrators', '管理员组'),
('power_users', '高级用户组'),
('standard_users', '标准用户组'),
('guests', '访客用户组')
ON CONFLICT (group_name) DO NOTHING;

-- 为管理员用户添加默认权限
INSERT INTO user_group_members (user_id, group_id)
SELECT u.id, g.id FROM users u, user_groups g 
WHERE u.name = 'admin' AND g.group_name = 'administrators'
ON CONFLICT (user_id, group_id) DO NOTHING;

-- 设置默认资源配额
INSERT INTO resource_quotas (user_id, cpu_limit, memory_limit_mb, storage_limit_gb, gpu_limit)
SELECT id, 4.0, 8192, 50, 1 FROM users WHERE name = 'admin'
ON CONFLICT (user_id) DO NOTHING;

-- =================================================
-- 数据库配置完成
-- =================================================

-- 显示创建的表
\dt

