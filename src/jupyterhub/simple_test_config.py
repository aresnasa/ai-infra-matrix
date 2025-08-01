#!/usr/bin/env python3
"""
JupyterHub Production Configuration
支持PostgreSQL用户同步和LDAP集成的配置
"""
import os
import subprocess
import pwd
import grp

# 获取JupyterHub配置对象
c = get_config()

print("✅ LOADING PRODUCTION CONFIGURATION WITH DB SYNC")

# ===== 核心设置 =====
c.JupyterHub.bind_url = 'http://0.0.0.0:8000/jupyter/'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081/jupyter/'
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0' 
c.JupyterHub.hub_port = 8081

# JupyterHub内部使用SQLite数据库
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/data/jupyterhub.sqlite'

# ===== 认证配置 - 使用自定义认证器 =====
# 暂时使用DummyAuthenticator进行测试，稍后切换到数据库认证
c.JupyterHub.authenticator_class = 'jupyterhub.auth.DummyAuthenticator'
c.DummyAuthenticator.password = "password"

# ===== Spawner配置 - 支持动态用户创建 =====
c.JupyterHub.spawner_class = 'jupyterhub.spawner.LocalProcessSpawner'

# 用户环境设置
c.Spawner.notebook_dir = '/srv/jupyterhub/notebooks/{username}'
c.Spawner.default_url = '/lab'

# 动态创建用户目录的函数
def create_user_environment(spawner):
    """为用户创建必要的目录和环境"""
    username = spawner.user.name
    user_notebook_dir = f'/srv/jupyterhub/notebooks/{username}'
    
    # 创建用户notebook目录
    os.makedirs(user_notebook_dir, exist_ok=True)
    
    # 设置目录权限（虽然在容器中，但保持一致性）
    try:
        os.chmod(user_notebook_dir, 0o755)
        print(f"📁 Created notebook directory for user: {username}")
    except Exception as e:
        print(f"⚠️  Warning: Could not set permissions for {username}: {e}")
    
    return user_notebook_dir

# 用户环境初始化钩子
def pre_spawn_hook(spawner):
    """在启动notebook服务器之前执行的钩子"""
    username = spawner.user.name
    
    # 创建用户环境
    notebook_dir = create_user_environment(spawner)
    
    # 更新spawner的notebook目录
    spawner.notebook_dir = notebook_dir
    
    # 设置用户特定的环境变量
    spawner.environment.update({
        'JUPYTER_ENABLE_LAB': '1',
        'USER': username,
        'HOME': f'/home/{username}',
        'JUPYTER_CONFIG_DIR': f'/srv/jupyterhub/notebooks/{username}/.jupyter',
        'JUPYTER_DATA_DIR': f'/srv/jupyterhub/notebooks/{username}/.local/share/jupyter',
        'PATH': '/usr/local/bin:/usr/bin:/bin'
    })
    
    print(f"🚀 Pre-spawn setup completed for user: {username}")

c.Spawner.pre_spawn_hook = pre_spawn_hook

# 基础环境变量
c.Spawner.environment = {
    'JUPYTER_ENABLE_LAB': '1',
    'PATH': '/usr/local/bin:/usr/bin:/bin'
}

# ===== 用户管理和权限 =====
c.JupyterHub.admin_access = True

# 从环境变量或数据库获取管理员用户
admin_users = set()
if os.environ.get('JUPYTERHUB_ADMIN_USERS'):
    admin_users.update(os.environ.get('JUPYTERHUB_ADMIN_USERS').split(','))
else:
    # 默认管理员用户
    admin_users.update(['admin', 'testuser'])

c.Authenticator.admin_users = admin_users

# 用户白名单（允许登录的用户）
allowed_users = set()
if os.environ.get('JUPYTERHUB_ALLOWED_USERS'):
    allowed_users.update(os.environ.get('JUPYTERHUB_ALLOWED_USERS').split(','))
else:
    # 开发环境允许所有用户
    c.Authenticator.allow_all = True

if allowed_users:
    c.Authenticator.allowed_users = allowed_users

# ===== 数据库用户同步配置 =====
# 数据库连接配置
DB_CONFIG = {
    'host': os.environ.get('DB_HOST', 'ai-infra-postgres'),
    'port': os.environ.get('DB_PORT', '5432'),
    'database': os.environ.get('DB_NAME', 'ansible_playbook_generator'),
    'user': os.environ.get('DB_USER', 'postgres'),
    'password': os.environ.get('DB_PASSWORD', 'postgres')
}

def sync_users_from_database():
    """从数据库同步用户信息"""
    try:
        import psycopg2
        
        conn = psycopg2.connect(**DB_CONFIG)
        cursor = conn.cursor()
        
        # 查询活跃用户及其角色
        cursor.execute("""
            SELECT DISTINCT u.username, u.email, u.is_active,
                   CASE WHEN r.name LIKE '%admin%' THEN true ELSE false END as is_admin
            FROM users u 
            LEFT JOIN user_roles ur ON u.id = ur.user_id 
            LEFT JOIN roles r ON ur.role_id = r.id
            WHERE u.is_active = true AND u.deleted_at IS NULL
        """)
        
        db_users = cursor.fetchall()
        
        active_users = set()
        admin_users = set()
        
        for username, email, is_admin, is_active in db_users:
            if is_active:
                active_users.add(username)
                if is_admin:
                    admin_users.add(username)
        
        cursor.close()
        conn.close()
        
        print(f"📊 Synced {len(active_users)} users from database")
        print(f"👑 Admin users: {admin_users}")
        
        return active_users, admin_users
        
    except Exception as e:
        print(f"⚠️  Database sync failed: {e}")
        # 返回默认用户
        return {'admin', 'testuser'}, {'admin', 'testuser'}

# 尝试从数据库同步用户
try:
    synced_users, synced_admins = sync_users_from_database()
    if synced_users:
        c.Authenticator.allowed_users = synced_users
        c.Authenticator.admin_users = synced_admins
        c.Authenticator.allow_all = False
except Exception as e:
    print(f"⚠️  Using default users due to sync error: {e}")

# ===== LDAP支持配置（未来使用） =====
# 这里为LDAP集成预留配置空间
LDAP_CONFIG = {
    'enabled': os.environ.get('LDAP_ENABLED', 'false').lower() == 'true',
    'server': os.environ.get('LDAP_SERVER', 'ai-infra-openldap'),
    'port': int(os.environ.get('LDAP_PORT', '389')),
    'base_dn': os.environ.get('LDAP_BASE_DN', 'dc=aiinfra,dc=local'),
    'bind_dn': os.environ.get('LDAP_BIND_DN', 'cn=admin,dc=aiinfra,dc=local'),
    'bind_password': os.environ.get('LDAP_BIND_PASSWORD', 'admin_password')
}

if LDAP_CONFIG['enabled']:
    print("🔗 LDAP integration enabled")
    # 这里可以添加LDAP用户同步逻辑

# ===== 加密密钥配置 =====
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/data/jupyterhub_cookie_secret'

# ===== 数据库配置 =====
# JupyterHub内部使用SQLite，避免与应用数据库冲突
# c.JupyterHub.db_url = f"postgresql://{DB_CONFIG['user']}:{DB_CONFIG['password']}@{DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}"

# ===== 调试和日志 =====
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

# ===== 服务配置 =====
# 允许命名服务器（用户可以启动多个notebook服务器）
c.JupyterHub.allow_named_servers = True
c.JupyterHub.named_server_limit_per_user = 3

# 服务器超时设置
c.Spawner.start_timeout = 60
c.Spawner.http_timeout = 30

print("🔧 Production configuration loaded successfully!")
print(f"�️  Database: {DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}")
print(f"📝 Spawner: LocalProcessSpawner with dynamic user directories")
print(f"� Admin users: {c.Authenticator.admin_users}")
print(f"🔐 Authentication: DummyAuthenticator (testing mode)")
