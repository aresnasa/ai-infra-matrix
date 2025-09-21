#!/bin/bash

# Docker 容器数据库初始化脚本
# 此脚本会在PostgreSQL容器首次启动时自动运行

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- 检查是否已初始化
    DO \$\$
    BEGIN
        -- 如果permissions表不存在，则执行初始化
        IF NOT EXISTS (SELECT FROM pg_tables WHERE tablename = 'permissions') THEN
            RAISE NOTICE '开始初始化数据库...';
            
            -- 创建权限表和基础权限数据
            CREATE TABLE permissions (
                id bigint NOT NULL,
                resource character varying(100) NOT NULL,
                verb character varying(50) NOT NULL,
                scope character varying(100) DEFAULT '*'::character varying,
                description character varying(500),
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE permissions_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE permissions_id_seq OWNED BY permissions.id;
            ALTER TABLE ONLY permissions ALTER COLUMN id SET DEFAULT nextval('permissions_id_seq'::regclass);

            -- 创建角色表
            CREATE TABLE roles (
                id bigint NOT NULL,
                name character varying(100) NOT NULL,
                description character varying(500),
                is_system boolean DEFAULT false,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE roles_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE roles_id_seq OWNED BY roles.id;
            ALTER TABLE ONLY roles ALTER COLUMN id SET DEFAULT nextval('roles_id_seq'::regclass);

            -- 创建角色权限关联表
            CREATE TABLE role_permissions (
                id bigint NOT NULL,
                role_id bigint NOT NULL,
                permission_id bigint NOT NULL,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE role_permissions_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE role_permissions_id_seq OWNED BY role_permissions.id;
            ALTER TABLE ONLY role_permissions ALTER COLUMN id SET DEFAULT nextval('role_permissions_id_seq'::regclass);

            -- 创建用户表
            CREATE TABLE users (
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

            CREATE SEQUENCE users_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE users_id_seq OWNED BY users.id;
            ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

            -- 创建用户角色关联表
            CREATE TABLE user_roles (
                id bigint NOT NULL,
                user_id bigint NOT NULL,
                role_id bigint NOT NULL,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE user_roles_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE user_roles_id_seq OWNED BY user_roles.id;
            ALTER TABLE ONLY user_roles ALTER COLUMN id SET DEFAULT nextval('user_roles_id_seq'::regclass);

            -- 创建用户组表
            CREATE TABLE user_groups (
                id bigint NOT NULL,
                name character varying(255) NOT NULL,
                description text,
                created_at timestamp with time zone,
                updated_at timestamp with time zone,
                deleted_at timestamp with time zone
            );

            CREATE SEQUENCE user_groups_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE user_groups_id_seq OWNED BY user_groups.id;
            ALTER TABLE ONLY user_groups ALTER COLUMN id SET DEFAULT nextval('user_groups_id_seq'::regclass);

            -- 创建用户组成员关联表
            CREATE TABLE user_group_memberships (
                id bigint NOT NULL,
                user_id bigint NOT NULL,
                user_group_id bigint NOT NULL,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE user_group_memberships_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE user_group_memberships_id_seq OWNED BY user_group_memberships.id;
            ALTER TABLE ONLY user_group_memberships ALTER COLUMN id SET DEFAULT nextval('user_group_memberships_id_seq'::regclass);

            -- 创建用户组角色关联表
            CREATE TABLE user_group_roles (
                id bigint NOT NULL,
                user_group_id bigint NOT NULL,
                role_id bigint NOT NULL,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE user_group_roles_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE user_group_roles_id_seq OWNED BY user_group_roles.id;
            ALTER TABLE ONLY user_group_roles ALTER COLUMN id SET DEFAULT nextval('user_group_roles_id_seq'::regclass);

            -- 创建角色绑定表
            CREATE TABLE role_bindings (
                id bigint NOT NULL,
                name character varying(255) NOT NULL,
                namespace character varying(255) DEFAULT 'default'::character varying,
                subject_type character varying(50) NOT NULL,
                subject_id bigint NOT NULL,
                role_id bigint NOT NULL,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE role_bindings_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE role_bindings_id_seq OWNED BY role_bindings.id;
            ALTER TABLE ONLY role_bindings ALTER COLUMN id SET DEFAULT nextval('role_bindings_id_seq'::regclass);

            -- 创建项目表
            CREATE TABLE projects (
                id bigint NOT NULL,
                user_id bigint NOT NULL,
                name character varying(255) NOT NULL,
                description text,
                created_at timestamp with time zone,
                updated_at timestamp with time zone,
                deleted_at timestamp with time zone
            );

            CREATE SEQUENCE projects_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE projects_id_seq OWNED BY projects.id;
            ALTER TABLE ONLY projects ALTER COLUMN id SET DEFAULT nextval('projects_id_seq'::regclass);

            -- 创建主机表
            CREATE TABLE hosts (
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

            CREATE SEQUENCE hosts_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE hosts_id_seq OWNED BY hosts.id;
            ALTER TABLE ONLY hosts ALTER COLUMN id SET DEFAULT nextval('hosts_id_seq'::regclass);

            -- 创建任务表
            CREATE TABLE tasks (
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

            CREATE SEQUENCE tasks_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE tasks_id_seq OWNED BY tasks.id;
            ALTER TABLE ONLY tasks ALTER COLUMN id SET DEFAULT nextval('tasks_id_seq'::regclass);

            -- 创建变量表
            CREATE TABLE variables (
                id bigint NOT NULL,
                project_id bigint NOT NULL,
                name character varying(255) NOT NULL,
                value text,
                type character varying(50) DEFAULT 'string'::character varying,
                created_at timestamp with time zone,
                updated_at timestamp with time zone,
                deleted_at timestamp with time zone
            );

            CREATE SEQUENCE variables_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE variables_id_seq OWNED BY variables.id;
            ALTER TABLE ONLY variables ALTER COLUMN id SET DEFAULT nextval('variables_id_seq'::regclass);

            -- 创建playbook生成记录表
            CREATE TABLE playbook_generations (
                id bigint NOT NULL,
                project_id bigint NOT NULL,
                user_id bigint NOT NULL,
                filename character varying(255) NOT NULL,
                status character varying(50) DEFAULT 'pending'::character varying,
                error_message text,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE playbook_generations_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE playbook_generations_id_seq OWNED BY playbook_generations.id;
            ALTER TABLE ONLY playbook_generations ALTER COLUMN id SET DEFAULT nextval('playbook_generations_id_seq'::regclass);

            -- 创建AI助手配置表
            CREATE TABLE ai_assistant_configs (
                id bigint NOT NULL,
                name character varying(100) NOT NULL,
                provider character varying(50) NOT NULL,
                api_key text,
                api_endpoint character varying(500),
                model character varying(100),
                max_tokens integer DEFAULT 4096,
                temperature real DEFAULT 0.7,
                system_prompt text,
                is_enabled boolean DEFAULT true,
                is_default boolean DEFAULT false,
                mcp_config text,
                rate_limit_per_hour integer DEFAULT 100,
                created_at timestamp with time zone,
                updated_at timestamp with time zone,
                deleted_at timestamp with time zone
            );

            CREATE SEQUENCE ai_assistant_configs_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE ai_assistant_configs_id_seq OWNED BY ai_assistant_configs.id;
            ALTER TABLE ONLY ai_assistant_configs ALTER COLUMN id SET DEFAULT nextval('ai_assistant_configs_id_seq'::regclass);

            -- 创建AI对话表
            CREATE TABLE ai_conversations (
                id bigint NOT NULL,
                user_id bigint NOT NULL,
                title character varying(200),
                config_id bigint NOT NULL,
                context text,
                is_active boolean DEFAULT true,
                tokens_used integer DEFAULT 0,
                created_at timestamp with time zone,
                updated_at timestamp with time zone,
                deleted_at timestamp with time zone
            );

            CREATE SEQUENCE ai_conversations_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE ai_conversations_id_seq OWNED BY ai_conversations.id;
            ALTER TABLE ONLY ai_conversations ALTER COLUMN id SET DEFAULT nextval('ai_conversations_id_seq'::regclass);

            -- 创建AI消息表
            CREATE TABLE ai_messages (
                id bigint NOT NULL,
                conversation_id bigint NOT NULL,
                role character varying(50) NOT NULL,
                content text NOT NULL,
                metadata text,
                tokens_used integer DEFAULT 0,
                response_time integer,
                created_at timestamp with time zone,
                updated_at timestamp with time zone,
                deleted_at timestamp with time zone
            );

            CREATE SEQUENCE ai_messages_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE ai_messages_id_seq OWNED BY ai_messages.id;
            ALTER TABLE ONLY ai_messages ALTER COLUMN id SET DEFAULT nextval('ai_messages_id_seq'::regclass);

            -- 创建AI使用统计表
            CREATE TABLE ai_usage_stats (
                id bigint NOT NULL,
                user_id bigint NOT NULL,
                config_id bigint NOT NULL,
                date timestamp with time zone NOT NULL,
                request_count integer DEFAULT 0,
                tokens_used integer DEFAULT 0,
                success_count integer DEFAULT 0,
                error_count integer DEFAULT 0,
                average_response integer DEFAULT 0,
                created_at timestamp with time zone,
                updated_at timestamp with time zone
            );

            CREATE SEQUENCE ai_usage_stats_id_seq
                START WITH 1
                INCREMENT BY 1
                NO MINVALUE
                NO MAXVALUE
                CACHE 1;

            ALTER SEQUENCE ai_usage_stats_id_seq OWNED BY ai_usage_stats.id;
            ALTER TABLE ONLY ai_usage_stats ALTER COLUMN id SET DEFAULT nextval('ai_usage_stats_id_seq'::regclass);

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
            ALTER TABLE ONLY ai_assistant_configs ADD CONSTRAINT ai_assistant_configs_pkey PRIMARY KEY (id);
            ALTER TABLE ONLY ai_conversations ADD CONSTRAINT ai_conversations_pkey PRIMARY KEY (id);
            ALTER TABLE ONLY ai_messages ADD CONSTRAINT ai_messages_pkey PRIMARY KEY (id);
            ALTER TABLE ONLY ai_usage_stats ADD CONSTRAINT ai_usage_stats_pkey PRIMARY KEY (id);

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

            -- 添加AI助手相关外键约束
            ALTER TABLE ONLY ai_conversations ADD CONSTRAINT fk_ai_conversations_user FOREIGN KEY (user_id) REFERENCES users(id);
            ALTER TABLE ONLY ai_conversations ADD CONSTRAINT fk_ai_conversations_config FOREIGN KEY (config_id) REFERENCES ai_assistant_configs(id);
            ALTER TABLE ONLY ai_messages ADD CONSTRAINT fk_ai_messages_conversation FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id);
            ALTER TABLE ONLY ai_usage_stats ADD CONSTRAINT fk_ai_usage_stats_user FOREIGN KEY (user_id) REFERENCES users(id);
            ALTER TABLE ONLY ai_usage_stats ADD CONSTRAINT fk_ai_usage_stats_config FOREIGN KEY (config_id) REFERENCES ai_assistant_configs(id);

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
            (13, 'groups', '*', '*', '用户组管理权限');

            -- 插入基础角色数据
            INSERT INTO roles (id, name, description, is_system) VALUES 
            (1, 'super-admin', '超级管理员角色', true),
            (2, 'user', '普通用户角色', true);

            -- 插入角色权限关联数据
            INSERT INTO role_permissions (id, role_id, permission_id) VALUES 
            (1, 1, 1),  -- super-admin 拥有超级管理员权限
            (2, 2, 2),  -- user 拥有创建项目权限
            (3, 2, 3),  -- user 拥有查看项目权限
            (4, 2, 4),  -- user 拥有更新项目权限
            (5, 2, 5),  -- user 拥有删除项目权限
            (6, 2, 6);  -- user 拥有列出项目权限

            -- 更新序列到正确的值
            SELECT setval('permissions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM permissions));
            SELECT setval('roles_id_seq', (SELECT COALESCE(MAX(id), 1) FROM roles));
            SELECT setval('role_permissions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM role_permissions));
            SELECT setval('users_id_seq', 1);
            SELECT setval('user_roles_id_seq', 1);
            SELECT setval('user_groups_id_seq', 1);
            SELECT setval('user_group_memberships_id_seq', 1);
            SELECT setval('user_group_roles_id_seq', 1);
            SELECT setval('role_bindings_id_seq', 1);
            SELECT setval('projects_id_seq', 1);
            SELECT setval('hosts_id_seq', 1);
            SELECT setval('tasks_id_seq', 1);
            SELECT setval('variables_id_seq', 1);
            SELECT setval('playbook_generations_id_seq', 1);
            SELECT setval('ai_assistant_configs_id_seq', 1);
            SELECT setval('ai_conversations_id_seq', 1);
            SELECT setval('ai_messages_id_seq', 1);
            SELECT setval('ai_usage_stats_id_seq', 1);

            RAISE NOTICE '数据库初始化完成！';
        ELSE
            RAISE NOTICE '数据库已经初始化，跳过初始化步骤';
        END IF;
    END
    \$\$;
EOSQL
