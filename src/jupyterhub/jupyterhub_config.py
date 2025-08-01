# JupyterHub配置 - 使用PostgreSQL + Redis + DummyAuthenticator
import os
import redis
import psycopg2
from jupyterhub.auth import DummyAuthenticator
from jupyterhub.spawner import LocalProcessSpawner

print("🔧 开始加载JupyterHub配置（PostgreSQL + Redis + DummyAuthenticator）...")

# 数据库配置
DB_CONFIG = {
    'host': os.environ.get('POSTGRES_HOST', 'ai-infra-postgres'),
    'port': int(os.environ.get('POSTGRES_PORT', 5432)),
    'database': os.environ.get('POSTGRES_DB', 'jupyterhub'),
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

# 基本配置
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.base_url = '/jupyter/'

# 数据库配置 - PostgreSQL（替换SQLite）
c.JupyterHub.db_url = f"postgresql://{DB_CONFIG['user']}:{DB_CONFIG['password']}@{DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}"

# 认证器配置 - 强制使用DummyAuthenticator
c.JupyterHub.authenticator_class = DummyAuthenticator

# 用户配置
c.Authenticator.allowed_users = {'admin', 'testuser'}
c.Authenticator.admin_users = {'admin'}

# Spawner配置  
c.JupyterHub.spawner_class = LocalProcessSpawner

# 安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'default-token-change-me')

# 日志配置
c.JupyterHub.log_level = 'DEBUG'

# 连接测试函数
def test_database_connection():
    """测试PostgreSQL连接"""
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

# 启动时验证连接
print("="*50)
print("🚀 JupyterHub启动中（PostgreSQL + Redis + DummyAuthenticator）...")
print("="*50)
test_database_connection()
test_redis_connection()
print("="*50)

print("✅ DummyAuthenticator配置已加载")
print(f"✅ 允许的用户: {list(c.Authenticator.allowed_users)}")
print(f"✅ 管理员用户: {list(c.Authenticator.admin_users)}")
print(f"✅ 数据库: PostgreSQL ({DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']})")
print(f"✅ 缓存: Redis ({REDIS_CONFIG['host']}:{REDIS_CONFIG['port']}/{REDIS_CONFIG['db']})")
