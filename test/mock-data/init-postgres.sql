-- Mock 数据初始化脚本
CREATE DATABASE ai_infra_test;
CREATE USER test_user WITH PASSWORD 'test_password';
GRANT ALL PRIVILEGES ON DATABASE ai_infra_test TO test_user;

\c ai_infra_test;

-- 创建测试表
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    user_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入测试数据
INSERT INTO users (username, email) VALUES 
    ('admin', 'admin@test.com'),
    ('testuser1', 'user1@test.com'),
    ('testuser2', 'user2@test.com');

INSERT INTO projects (name, description, user_id) VALUES 
    ('AI Project 1', 'Machine Learning Project', 1),
    ('AI Project 2', 'Deep Learning Project', 2),
    ('Data Analysis', 'Statistical Analysis Project', 3);
