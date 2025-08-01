# 最小化的JupyterHub配置，专注于DummyAuthenticator
import os
from jupyterhub.auth import DummyAuthenticator
from jupyterhub.spawner import LocalProcessSpawner

print("🔧 开始加载JupyterHub配置...")

# 基本配置
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.base_url = '/jupyter/'

# 认证器配置 - 强制使用DummyAuthenticator
c.JupyterHub.authenticator_class = DummyAuthenticator

# 用户配置
c.Authenticator.allowed_users = {'admin', 'testuser'}
c.Authenticator.admin_users = {'admin'}

# Spawner配置  
c.JupyterHub.spawner_class = LocalProcessSpawner

# 数据库配置（简化为SQLite）
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

# 安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'

# 日志配置
c.JupyterHub.log_level = 'DEBUG'

print("✅ DummyAuthenticator配置已加载")
print(f"✅ 允许的用户: {list(c.Authenticator.allowed_users)}")
print(f"✅ 管理员用户: {list(c.Authenticator.admin_users)}")

# 数据库配置
DB_CONFIG = {
    'host': os.environ.get('POSTGRES_HOST', 'ai-infra-postgres'),
    'port': int(os.environ.get('POSTGRES_PORT', 5432)),
    'database': os.environ.get('POSTGRES_DB', 'jupyterhub'),
    'user': os.environ.get('POSTGRES_USER', 'postgres'),
    'password': os.environ.get('POSTGRES_PASSWORD', 'postgres')
}

# 用户数据库配置
USER_DB_CONFIG = {
    'host': os.environ.get('POSTGRES_HOST', 'ai-infra-postgres'),
    'port': int(os.environ.get('POSTGRES_PORT', 5432)),
    'database': os.environ.get('USER_DB_NAME', 'ansible_playbook_generator'),
    'user': os.environ.get('POSTGRES_USER', 'postgres'),
    'password': os.environ.get('POSTGRES_PASSWORD', 'postgres')
}

# Redis配置
REDIS_CONFIG = {
    'host': os.environ.get('REDIS_HOST', 'ai-infra-redis'),
    'port': int(os.environ.get('REDIS_PORT', 6379)),
    'password': os.environ.get('REDIS_PASSWORD', 'ansible-redis-password'),
    'db': int(os.environ.get('REDIS_DB', 1)),
    'decode_responses': True
}

def ensure_system_user(username):
    """确保系统用户存在"""
    try:
        # 检查用户是否存在
        pwd.getpwnam(username)
        print(f"✅ 系统用户 {username} 已存在")
        return True
    except KeyError:
        try:
            # 创建系统用户
            subprocess.run([
                'adduser', '-D', '-s', '/bin/bash', username
            ], check=True, capture_output=True)
            
            # 设置密码
            subprocess.run([
                'sh', '-c', f'echo "{username}:password" | chpasswd'
            ], check=True, capture_output=True)
            
            # 创建用户目录
            home_dir = f"/home/{username}"
            notebook_dir = f"/srv/jupyterhub/notebooks/{username}"
            
            for directory in [home_dir, notebook_dir]:
                os.makedirs(directory, exist_ok=True)
                subprocess.run(['chown', f'{username}:{username}', directory], check=True)
            
            print(f"✅ 创建系统用户 {username} 成功")
            return True
        except subprocess.CalledProcessError as e:
            print(f"❌ 创建系统用户 {username} 失败: {e}")
            return False

def test_database_connection():
    """测试数据库连接"""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        conn.close()
        print("✅ PostgreSQL connection successful")
        return True
    except Exception as e:
        print(f"❌ PostgreSQL connection failed: {e}")
        return False

def test_redis_connection():
    """测试Redis连接"""
    try:
        r = redis.Redis(**REDIS_CONFIG)
        r.ping()
        print("✅ Redis connection successful")
        return True
    except Exception as e:
        print(f"❌ Redis connection failed: {e}")
        return False

def sync_users_from_database():
    """从数据库同步用户到Redis缓存并确保系统用户存在"""
    try:
        # 连接用户数据库
        conn = psycopg2.connect(**USER_DB_CONFIG, cursor_factory=RealDictCursor)
        cursor = conn.cursor()
        
        # 查询活跃用户
        query = """
        SELECT u.id, u.username, u.email, u.is_active, u.created_at,
               ARRAY_AGG(r.name) as roles
        FROM users u 
        LEFT JOIN user_roles ur ON u.id = ur.user_id 
        LEFT JOIN roles r ON ur.role_id = r.id 
        WHERE u.is_active = true AND u.deleted_at IS NULL
        GROUP BY u.id, u.username, u.email, u.is_active, u.created_at
        """
        
        cursor.execute(query)
        users = cursor.fetchall()
        conn.close()
        
        if not users:
            print("⚠️  No active users found in database, using defaults")
            # 使用默认用户
            default_users = [
                {'username': 'admin', 'roles': ['admin']},
                {'username': 'testuser', 'roles': ['user']}
            ]
            users = default_users
        
        # 连接Redis
        r = redis.Redis(**REDIS_CONFIG)
        
        active_users = []
        admin_users = []
        
        for user in users:
            username = user['username']
            roles = user.get('roles', []) or []
            
            # 确保系统用户存在
            if ensure_system_user(username):
                active_users.append(username)
                
                # 检查是否是管理员
                if 'admin' in roles or username == 'admin':
                    admin_users.append(username)
                
                # 缓存用户信息
                user_key = f"jupyterhub:user:{username}"
                user_data = {
                    'username': username,
                    'roles': json.dumps(roles),
                    'last_sync': datetime.now().isoformat(),
                    'system_user_ready': 'true'
                }
                
                r.hset(user_key, mapping=user_data)
                r.expire(user_key, 3600)  # 1小时过期
        
        # 更新活跃用户列表
        if active_users:
            r.delete("jupyterhub:users:active")
            r.lpush("jupyterhub:users:active", *active_users)
            r.expire("jupyterhub:users:active", 3600)
        
        # 更新管理员用户列表  
        if admin_users:
            r.delete("jupyterhub:users:admin")
            r.lpush("jupyterhub:users:admin", *admin_users)
            r.expire("jupyterhub:users:admin", 3600)
        
        print(f"✅ 用户同步完成: {len(active_users)} 活跃用户, {len(admin_users)} 管理员")
        return active_users, admin_users
        
    except Exception as e:
        print(f"❌ 用户同步失败: {e}")
        # 返回默认用户并确保系统用户存在
        default_users = ['admin', 'testuser']
        for username in default_users:
            ensure_system_user(username)
        return default_users, ['admin']


class CustomLocalProcessSpawner(LocalProcessSpawner):
    """自定义本地进程Spawner，支持动态用户目录创建和Redis活动日志"""
    
    def user_env(self, env):
        """设置用户环境变量，避免系统用户查找"""
        # 设置虚拟HOME目录，避免pwd查找
        username = self.user.name
        user_home = f"/srv/jupyterhub/notebooks/{username}"
        
        # 创建并设置环境变量
        os.makedirs(user_home, exist_ok=True)
        os.chmod(user_home, 0o755)
        
        env['HOME'] = user_home
        env['USER'] = username
        env['LOGNAME'] = username
        env['SHELL'] = '/bin/bash'
        
        return env
    
    async def start(self):
        """启动用户的Jupyter服务器"""
        # 创建用户环境
        self.create_user_environment()
        
        # 记录用户活动到Redis
        self.log_user_activity()
        
        # 设置工作目录
        username = self.user.name
        self.notebook_dir = f"/srv/jupyterhub/notebooks/{username}"
        
        # 调用父类的start方法
        return await super().start()
        
    def create_user_environment(self):
        """为用户创建notebook目录和环境"""
        username = self.user.name
        user_dir = f"/srv/jupyterhub/notebooks/{username}"
        
        try:
            os.makedirs(user_dir, exist_ok=True)
            os.chmod(user_dir, 0o755)
            
            # 创建欢迎notebook
            welcome_nb = os.path.join(user_dir, "Welcome.ipynb")
            if not os.path.exists(welcome_nb):
                welcome_content = {
                    "cells": [
                        {
                            "cell_type": "markdown",
                            "metadata": {},
                            "source": [
                                f"# 欢迎 {username}!\\n",
                                "\\n",
                                "这是您的个人JupyterLab工作空间。\\n",
                                "\\n",
                                f"- 用户: {username}\\n",
                                f"- 工作目录: {user_dir}\\n",
                                "- 后端: PostgreSQL + Redis统一架构\\n"
                            ]
                        }
                    ],
                    "metadata": {
                        "kernelspec": {
                            "display_name": "Python 3",
                            "language": "python",
                            "name": "python3"
                        }
                    },
                    "nbformat": 4,
                    "nbformat_minor": 4
                }
                
                with open(welcome_nb, 'w') as f:
                    json.dump(welcome_content, f, indent=2)
            
            self.log.info(f"📁 Created user directory: {user_dir}")
        except Exception as e:
            self.log.error(f"❌ Failed to create user directory {user_dir}: {e}")
    
    def log_user_activity(self):
        """记录用户活动到Redis"""
        try:
            r = redis.Redis(**REDIS_CONFIG)
            username = self.user.name
            activity_key = f"jupyterhub:user_activity:{username}"
            
            # 记录用户启动时间
            activity_data = {
                'last_spawn': datetime.now().isoformat(),
                'spawn_count': str(r.incr(f"{activity_key}:count")),
                'user_dir': f"/srv/jupyterhub/notebooks/{username}"
            }
            
            r.hset(activity_key, mapping=activity_data)
            r.expire(activity_key, 86400)  # 24小时过期
            
            self.log.info(f"📊 Logged user activity for {username}")
        except Exception as e:
            self.log.error(f"❌ Failed to log user activity: {e}")

# JupyterHub配置
# c 变量由 JupyterHub 自动提供

# 基本配置
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'

# 基础URL
c.JupyterHub.base_url = '/jupyter/'

# 数据库配置 - PostgreSQL（完全替换SQLite）
c.JupyterHub.db_url = f"postgresql://{DB_CONFIG['user']}:{DB_CONFIG['password']}@{DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}"

# 认证器配置（使用DummyAuthenticator避免PAM问题）
c.JupyterHub.authenticator_class = DummyAuthenticator

# 强制设置DummyAuthenticator不需要密码
c.DummyAuthenticator.password = ""

# Spawner配置
c.JupyterHub.spawner_class = CustomLocalProcessSpawner

# 安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/cookie_secret'
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'default-token-change-me')

# 简化开发配置
c.JupyterHub.tornado_settings = {
    'headers': {
        'X-Frame-Options': 'SAMEORIGIN',
    },
}

# 用户同步
try:
    print("🔄 Starting user synchronization from database...")
    active_users, admin_users = sync_users_from_database()
    
    # 设置允许的用户
    c.Authenticator.allowed_users = set(active_users)
    
    # 设置管理员用户
    c.Authenticator.admin_users = set(admin_users)
    
except Exception as e:
    print(f"❌ User sync failed during startup: {e}")
    # 使用默认用户作为回退
    c.Authenticator.allowed_users = {'admin', 'testuser'}
    c.Authenticator.admin_users = {'admin'}

# 日志配置
c.JupyterHub.log_level = 'INFO'
c.Application.log_datefmt = '%Y-%m-%d %H:%M:%S'
c.Application.log_format = '[%(levelname)1.1s %(asctime)s.%(msecs).03d %(name)s %(module)s:%(lineno)d] %(message)s'

# 启动时验证连接
print("="*50)
print("🚀 JupyterHub统一后端启动中...")
print("="*50)
test_database_connection()
test_redis_connection()
print("="*50)
