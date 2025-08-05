-- 测试数据库初始化脚本
-- 创建测试用户和初始数据

-- 创建管理员用户
INSERT INTO users (username, email, password_hash, role, is_active, created_at, updated_at) 
VALUES 
  ('admin', 'admin@test.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin', true, NOW(), NOW()),
  ('testuser', 'test@test.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'user', true, NOW(), NOW()),
  ('inactive_user', 'inactive@test.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'user', false, NOW(), NOW());

-- 创建测试项目
INSERT INTO projects (name, description, user_id, status, created_at, updated_at)
VALUES 
  ('Test Project 1', 'First test project', 2, 'active', NOW(), NOW()),
  ('Test Project 2', 'Second test project', 2, 'active', NOW(), NOW()),
  ('Admin Project', 'Admin test project', 1, 'active', NOW(), NOW());

-- 创建测试主机
INSERT INTO hosts (project_id, name, ip_address, ssh_user, ssh_port, variables, created_at, updated_at)
VALUES 
  (1, 'test-web-01', '192.168.1.10', 'ubuntu', 22, '{"env": "test", "role": "web"}', NOW(), NOW()),
  (1, 'test-db-01', '192.168.1.11', 'ubuntu', 22, '{"env": "test", "role": "database"}', NOW(), NOW()),
  (2, 'prod-web-01', '10.0.1.10', 'centos', 22, '{"env": "prod", "role": "web"}', NOW(), NOW());

-- 创建测试变量
INSERT INTO variables (project_id, name, value, description, is_secret, created_at, updated_at)
VALUES 
  (1, 'app_name', 'test-app', 'Application name', false, NOW(), NOW()),
  (1, 'db_password', 'encrypted_password', 'Database password', true, NOW(), NOW()),
  (2, 'domain_name', 'example.com', 'Domain name', false, NOW(), NOW());

-- 创建测试任务
INSERT INTO tasks (project_id, name, playbook_content, variables, tags, created_at, updated_at)
VALUES 
  (1, 'Deploy Application', 'tasks:\n  - name: Install nginx\n    apt:\n      name: nginx\n      state: present', '{"port": 80}', 'deploy,web', NOW(), NOW()),
  (2, 'Update System', 'tasks:\n  - name: Update packages\n    yum:\n      name: "*"\n      state: latest', '{}', 'update,system', NOW(), NOW());

-- 创建测试角色和权限（如果RBAC表存在）
-- 这些语句会在RBAC初始化后执行
