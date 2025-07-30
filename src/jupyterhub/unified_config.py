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
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8091'

# 设置正确的base URL用于反向代理
c.JupyterHub.base_url = '/jupyter/'

# 不设置 hub_public_url，让 JupyterHub 自动处理重定向
# 这样可以避免重定向循环问题
# c.JupyterHub.hub_public_url = ''

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

# 会话和安全配置
c.PostgreSQLRedisAuthenticator.jwt_secret = os.environ.get('JWT_SECRET', 'your-secret-key-change-in-production')

# 会话超时 (24小时)
c.PostgreSQLRedisAuthenticator.session_timeout = int(os.environ.get('SESSION_TIMEOUT', str(3600 * 24)))  # 24小时

# ===== Spawner 配置 =====

# 环境检测和spawner选择
spawner_mode = os.environ.get('JUPYTERHUB_SPAWNER', 'local')

if spawner_mode == 'docker':
    # Docker Spawner配置
    from dockerspawner import DockerSpawner
    c.JupyterHub.spawner_class = spawner_class = DockerSpawner
    
    # Docker 配置
    c.DockerSpawner.image = os.environ.get('JUPYTERHUB_NOTEBOOK_IMAGE', 'jupyter/base-notebook:latest')
    
    # 网络配置
    c.DockerSpawner.network_name = os.environ.get('JUPYTERHUB_NETWORK', 'ai-infra-matrix_default')
    
    # 卷挂载配置
    c.DockerSpawner.volumes = {
        '/home/{username}': '/home/jovyan/work',
        '/srv/data/shared': '/srv/shared'
    }
    
    # 资源限制
    c.DockerSpawner.cpu_limit = float(os.environ.get('JUPYTERHUB_CPU_LIMIT', '1.0'))
    c.DockerSpawner.mem_limit = os.environ.get('JUPYTERHUB_MEM_LIMIT', '2G')
    
    # 环境变量
    c.DockerSpawner.environment = {
        'GRANT_SUDO': 'yes',
        'CHOWN_HOME': 'yes',
        'CHOWN_HOME_OPTS': '-R'
    }
    
    # 清理容器
    c.DockerSpawner.remove = True
    
    # 开发模式的额外配置
    if os.environ.get('JUPYTERHUB_DEBUG', '').lower() == 'true':
        c.DockerSpawner.debug = True

else:
    # 本地进程spawner（开发环境）
    from jupyterhub.spawner import LocalProcessSpawner
    c.JupyterHub.spawner_class = LocalProcessSpawner
    c.LocalProcessSpawner.create_system_users = False

# Spawner通用配置
# Notebook工作目录
notebook_dir = Path(os.environ.get('JUPYTERHUB_NOTEBOOK_DIR', '/srv/data/shared/notebooks'))
notebook_dir.mkdir(parents=True, exist_ok=True)
c.Spawner.notebook_dir = str(notebook_dir)

# 默认启动页面
c.Spawner.default_url = '/lab'

# ===== 管理员配置 =====

# 管理员用户
admin_users_env = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin')
if admin_users_env:
    c.JupyterHub.admin_users = set(admin_users_env.split(','))

# ===== 服务配置 =====

# 内置服务
services = []

# 可选的idle culler服务 - 暂时禁用以避免模块错误
# if os.environ.get('JUPYTERHUB_IDLE_CULLER_ENABLED', 'false').lower() == 'true':
#     idle_timeout = int(os.environ.get('JUPYTERHUB_IDLE_TIMEOUT', '3600'))  # 1小时
#     cull_interval = int(os.environ.get('JUPYTERHUB_CULL_INTERVAL', '7200'))  # 2小时
#     
#     services.append({
#         'name': 'idle-culler',
#         'command': [
#             'python3', '-m', 'jupyterhub_idle_culler',
#             f'--timeout={idle_timeout}',
#             f'--cull-every={cull_interval}',
#             '--remove-named-servers'
#         ]
#     })

c.JupyterHub.services = services

# ===== 开发和调试配置 =====

# 日志配置
if os.environ.get('JUPYTERHUB_DEBUG', '').lower() == 'true':
    c.JupyterHub.log_level = 'DEBUG'
    c.Application.log_level = 'DEBUG'

# SSL配置（生产环境）
ssl_key = os.environ.get('JUPYTERHUB_SSL_KEY')
ssl_cert = os.environ.get('JUPYTERHUB_SSL_CERT')
if ssl_key and ssl_cert:
    c.JupyterHub.ssl_key = ssl_key
    c.JupyterHub.ssl_cert = ssl_cert

print("=== JupyterHub 配置加载完成 ===")
print(f"Base URL: {c.JupyterHub.base_url}")
print(f"Bind URL: {c.JupyterHub.bind_url}")
print(f"Hub Bind URL: {c.JupyterHub.hub_bind_url}")
print(f"Spawner模式: {spawner_mode}")
print(f"认证器: PostgreSQL + Redis")
print("==================================")
