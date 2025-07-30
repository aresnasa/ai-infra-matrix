# AI-Infra-Matrix 统一用户体系 JupyterHub 配置
# 使用 PostgreSQL + Redis 的认证系统

import os
import sys
from pathlib import Path

# 添加自定义认证器到Python路径
sys.path.insert(0, '/srv/jupyterhub')

# 导入自定义认证器
from postgres_authenticator import PostgreSQLRedisAuthenticator

# 基本服务器配置
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.hub_port = 8091

# 数据目录
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)

# 不再使用SQLite，JupyterHub仍需要自己的数据库来存储spawner状态等
# 但用户认证完全依赖PostgreSQL
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub_internal.sqlite'
c.JupyterHub.cookie_secret_file = str(project_data_dir / "cookie_secret")

# 代理配置
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'ai-infra-proxy-token')

# ===== 统一认证配置 =====

# 使用自定义的PostgreSQL Redis认证器
c.JupyterHub.authenticator_class = PostgreSQLRedisAuthenticator

# PostgreSQL配置 - 从环境变量读取
c.PostgreSQLRedisAuthenticator.db_host = os.environ.get('DB_HOST', 'localhost')
c.PostgreSQLRedisAuthenticator.db_port = int(os.environ.get('DB_PORT', '5432'))
c.PostgreSQLRedisAuthenticator.db_name = os.environ.get('DB_NAME', 'ansible_playbook_generator')
c.PostgreSQLRedisAuthenticator.db_user = os.environ.get('DB_USER', 'postgres')
c.PostgreSQLRedisAuthenticator.db_password = os.environ.get('DB_PASSWORD', 'postgres')

# Redis配置 - 从环境变量读取
c.PostgreSQLRedisAuthenticator.redis_host = os.environ.get('REDIS_HOST', 'localhost')
c.PostgreSQLRedisAuthenticator.redis_port = int(os.environ.get('REDIS_PORT', '6379'))
c.PostgreSQLRedisAuthenticator.redis_password = os.environ.get('REDIS_PASSWORD', '')
c.PostgreSQLRedisAuthenticator.redis_db = int(os.environ.get('REDIS_DB', '0'))

# JWT配置
c.PostgreSQLRedisAuthenticator.jwt_secret = os.environ.get('JWT_SECRET', 'your-secret-key-change-in-production')

# 会话配置
c.PostgreSQLRedisAuthenticator.session_timeout = int(os.environ.get('SESSION_TIMEOUT', str(3600 * 24)))  # 24小时

# ===== Spawner配置 =====

# 使用LocalProcessSpawner（简化版本）
from jupyterhub.spawner import LocalProcessSpawner
c.JupyterHub.spawner_class = LocalProcessSpawner

# 设置用户工作目录
c.LocalProcessSpawner.create_system_users = False

# 设置notebook目录
notebook_dir = Path("/srv/jupyterhub/notebooks")
notebook_dir.mkdir(parents=True, exist_ok=True)
c.Spawner.notebook_dir = str(notebook_dir)

# 设置默认URL
c.Spawner.default_url = '/lab'

# ===== 管理员配置 =====

# 管理员用户从环境变量读取
admin_users_env = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin')
if admin_users_env:
    c.JupyterHub.admin_users = set(admin_users_env.split(','))

# ===== 安全配置 =====

# CORS配置
c.JupyterHub.tornado_settings = {
    'headers': {
        'Access-Control-Allow-Origin': os.environ.get('JUPYTERHUB_CORS_ORIGIN', '*'),
        'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    }
}

# ===== 日志配置 =====

# 日志级别
log_level = os.environ.get('JUPYTERHUB_LOG_LEVEL', 'INFO').upper()
c.JupyterHub.log_level = log_level

# 访问日志
if os.environ.get('JUPYTERHUB_ACCESS_LOG', 'true').lower() == 'true':
    c.JupyterHub.extra_log_file = f'{project_data_dir}/access.log'

print("=== AI-Infra-Matrix JupyterHub 统一认证系统配置加载完成 ===")
print(f"数据目录: {project_data_dir}")
print(f"绑定地址: http://0.0.0.0:8000")
print(f"数据库: {os.environ.get('DB_HOST', 'localhost')}:{os.environ.get('DB_PORT', '5432')}")
print(f"Redis: {os.environ.get('REDIS_HOST', 'localhost')}:{os.environ.get('REDIS_PORT', '6379')}")
print(f"管理员用户: {admin_users_env}")
print("=== 配置加载完成 ===")
