-- 
-- Ansible Playbook Generator Database Initialization Script
-- 包含表结构和基础RBAC数据
-- 

-- 创建权限表和基础权限数据
CREATE TABLE IF NOT EXISTS permissions (
    id bigint NOT NULL,
    resource character varying(100) NOT NULL,
    verb character varying(50) NOT NULL,
    scope character varying(100) DEFAULT '*'::character varying,
    description character varying(500),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE permissions_id_seq OWNED BY permissions.id;
ALTER TABLE ONLY permissions ALTER COLUMN id SET DEFAULT nextval('permissions_id_seq'::regclass);

-- 创建角色表
CREATE TABLE IF NOT EXISTS roles (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    description character varying(500),
    is_system boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE roles_id_seq OWNED BY roles.id;
ALTER TABLE ONLY roles ALTER COLUMN id SET DEFAULT nextval('roles_id_seq'::regclass);

-- 创建角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    id bigint NOT NULL,
    role_id bigint NOT NULL,
    permission_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS role_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE role_permissions_id_seq OWNED BY role_permissions.id;
ALTER TABLE ONLY role_permissions ALTER COLUMN id SET DEFAULT nextval('role_permissions_id_seq'::regclass);

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id bigint NOT NULL,
    username character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    is_active boolean DEFAULT true,
    last_login timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE users_id_seq OWNED BY users.id;
ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

-- 创建用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    role_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS user_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_roles_id_seq OWNED BY user_roles.id;
ALTER TABLE ONLY user_roles ALTER COLUMN id SET DEFAULT nextval('user_roles_id_seq'::regclass);

-- 创建用户组表
CREATE TABLE IF NOT EXISTS user_groups (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS user_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_groups_id_seq OWNED BY user_groups.id;
ALTER TABLE ONLY user_groups ALTER COLUMN id SET DEFAULT nextval('user_groups_id_seq'::regclass);

-- 创建用户组成员关联表
CREATE TABLE IF NOT EXISTS user_group_memberships (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    user_group_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS user_group_memberships_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_group_memberships_id_seq OWNED BY user_group_memberships.id;
ALTER TABLE ONLY user_group_memberships ALTER COLUMN id SET DEFAULT nextval('user_group_memberships_id_seq'::regclass);

-- 创建用户组角色关联表
CREATE TABLE IF NOT EXISTS user_group_roles (
    id bigint NOT NULL,
    user_group_id bigint NOT NULL,
    role_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS user_group_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_group_roles_id_seq OWNED BY user_group_roles.id;
ALTER TABLE ONLY user_group_roles ALTER COLUMN id SET DEFAULT nextval('user_group_roles_id_seq'::regclass);

-- 创建角色绑定表
CREATE TABLE IF NOT EXISTS role_bindings (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    namespace character varying(255) DEFAULT 'default'::character varying,
    subject_type character varying(50) NOT NULL,
    subject_id bigint NOT NULL,
    role_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS role_bindings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE role_bindings_id_seq OWNED BY role_bindings.id;
ALTER TABLE ONLY role_bindings ALTER COLUMN id SET DEFAULT nextval('role_bindings_id_seq'::regclass);

-- 创建项目表
CREATE TABLE IF NOT EXISTS projects (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS projects_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE projects_id_seq OWNED BY projects.id;
ALTER TABLE ONLY projects ALTER COLUMN id SET DEFAULT nextval('projects_id_seq'::regclass);

-- 创建主机表
CREATE TABLE IF NOT EXISTS hosts (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    ip character varying(45) NOT NULL,
    port bigint DEFAULT 22,
    "user" character varying(100) NOT NULL,
    "group" character varying(100),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS hosts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE hosts_id_seq OWNED BY hosts.id;
ALTER TABLE ONLY hosts ALTER COLUMN id SET DEFAULT nextval('hosts_id_seq'::regclass);

-- 创建任务表
CREATE TABLE IF NOT EXISTS tasks (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    module character varying(255) NOT NULL,
    params text,
    "when" character varying(255),
    tags text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE tasks_id_seq OWNED BY tasks.id;
ALTER TABLE ONLY tasks ALTER COLUMN id SET DEFAULT nextval('tasks_id_seq'::regclass);

-- 创建变量表
CREATE TABLE IF NOT EXISTS variables (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    value text,
    type character varying(50) DEFAULT 'string'::character varying,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS variables_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE variables_id_seq OWNED BY variables.id;
ALTER TABLE ONLY variables ALTER COLUMN id SET DEFAULT nextval('variables_id_seq'::regclass);

-- 创建playbook生成记录表
CREATE TABLE IF NOT EXISTS playbook_generations (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    user_id bigint NOT NULL,
    filename character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying,
    error_message text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS playbook_generations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE playbook_generations_id_seq OWNED BY playbook_generations.id;
ALTER TABLE ONLY playbook_generations ALTER COLUMN id SET DEFAULT nextval('playbook_generations_id_seq'::regclass);

-- 添加主键约束
ALTER TABLE ONLY permissions ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY roles ADD CONSTRAINT roles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY role_permissions ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE ONLY user_roles ADD CONSTRAINT user_roles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY user_groups ADD CONSTRAINT user_groups_pkey PRIMARY KEY (id);
ALTER TABLE ONLY user_group_memberships ADD CONSTRAINT user_group_memberships_pkey PRIMARY KEY (id);
ALTER TABLE ONLY user_group_roles ADD CONSTRAINT user_group_roles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY role_bindings ADD CONSTRAINT role_bindings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY projects ADD CONSTRAINT projects_pkey PRIMARY KEY (id);
ALTER TABLE ONLY hosts ADD CONSTRAINT hosts_pkey PRIMARY KEY (id);
ALTER TABLE ONLY tasks ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY variables ADD CONSTRAINT variables_pkey PRIMARY KEY (id);
ALTER TABLE ONLY playbook_generations ADD CONSTRAINT playbook_generations_pkey PRIMARY KEY (id);

-- 添加唯一约束
ALTER TABLE ONLY users ADD CONSTRAINT users_username_key UNIQUE (username);
ALTER TABLE ONLY users ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE ONLY roles ADD CONSTRAINT roles_name_key UNIQUE (name);
ALTER TABLE ONLY user_groups ADD CONSTRAINT user_groups_name_key UNIQUE (name);

-- 添加外键约束
ALTER TABLE ONLY role_permissions ADD CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id);
ALTER TABLE ONLY role_permissions ADD CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE ONLY user_roles ADD CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE ONLY user_roles ADD CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE ONLY user_group_memberships ADD CONSTRAINT fk_user_group_memberships_user FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE ONLY user_group_memberships ADD CONSTRAINT fk_user_group_memberships_user_group FOREIGN KEY (user_group_id) REFERENCES user_groups(id);
ALTER TABLE ONLY user_group_roles ADD CONSTRAINT fk_user_group_roles_role FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE ONLY user_group_roles ADD CONSTRAINT fk_user_group_roles_user_group FOREIGN KEY (user_group_id) REFERENCES user_groups(id);
ALTER TABLE ONLY role_bindings ADD CONSTRAINT fk_role_bindings_role FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE ONLY projects ADD CONSTRAINT fk_projects_user FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE ONLY hosts ADD CONSTRAINT fk_hosts_project FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE ONLY tasks ADD CONSTRAINT fk_tasks_project FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE ONLY variables ADD CONSTRAINT fk_variables_project FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE ONLY playbook_generations ADD CONSTRAINT fk_playbook_generations_project FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE ONLY playbook_generations ADD CONSTRAINT fk_playbook_generations_user FOREIGN KEY (user_id) REFERENCES users(id);

-- 插入基础权限数据
INSERT INTO permissions (id, resource, verb, scope, description) VALUES 
(1, '*', '*', '*', '超级管理员权限'),
(2, 'projects', 'create', '*', '创建项目权限'),
(3, 'projects', 'read', '*', '查看项目权限'),
(4, 'projects', 'update', '*', '更新项目权限'),
(5, 'projects', 'delete', '*', '删除项目权限'),
(6, 'projects', 'list', '*', '列出项目权限'),
(7, 'users', 'create', '*', '创建用户权限'),
(8, 'users', 'read', '*', '查看用户权限'),
(9, 'users', 'update', '*', '更新用户权限'),
(10, 'users', 'delete', '*', '删除用户权限'),
(11, 'users', 'list', '*', '列出用户权限'),
(12, 'roles', '*', '*', '角色管理权限'),
(13, 'groups', '*', '*', '用户组管理权限')
ON CONFLICT (id) DO NOTHING;

-- 插入基础角色数据
INSERT INTO roles (id, name, description, is_system) VALUES 
(1, 'super-admin', '超级管理员角色', true),
(2, 'user', '普通用户角色', true)
ON CONFLICT (id) DO NOTHING;

-- 插入角色权限关联数据
INSERT INTO role_permissions (id, role_id, permission_id) VALUES 
(1, 1, 1),  -- super-admin 拥有超级管理员权限
(2, 2, 2),  -- user 拥有创建项目权限
(3, 2, 3),  -- user 拥有查看项目权限
(4, 2, 4),  -- user 拥有更新项目权限
(5, 2, 5),  -- user 拥有删除项目权限
(6, 2, 6)   -- user 拥有列出项目权限
ON CONFLICT (id) DO NOTHING;

-- 更新序列到正确的值
SELECT setval('permissions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM permissions));
SELECT setval('roles_id_seq', (SELECT COALESCE(MAX(id), 1) FROM roles));
SELECT setval('role_permissions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM role_permissions));
SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users));
SELECT setval('user_roles_id_seq', (SELECT COALESCE(MAX(id), 1) FROM user_roles));
SELECT setval('user_groups_id_seq', (SELECT COALESCE(MAX(id), 1) FROM user_groups));
SELECT setval('user_group_memberships_id_seq', (SELECT COALESCE(MAX(id), 1) FROM user_group_memberships));
SELECT setval('user_group_roles_id_seq', (SELECT COALESCE(MAX(id), 1) FROM user_group_roles));
SELECT setval('role_bindings_id_seq', (SELECT COALESCE(MAX(id), 1) FROM role_bindings));
SELECT setval('projects_id_seq', (SELECT COALESCE(MAX(id), 1) FROM projects));
SELECT setval('hosts_id_seq', (SELECT COALESCE(MAX(id), 1) FROM hosts));
SELECT setval('tasks_id_seq', (SELECT COALESCE(MAX(id), 1) FROM tasks));
SELECT setval('variables_id_seq', (SELECT COALESCE(MAX(id), 1) FROM variables));
SELECT setval('playbook_generations_id_seq', (SELECT COALESCE(MAX(id), 1) FROM playbook_generations));
