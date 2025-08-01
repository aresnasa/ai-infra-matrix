# AI-Infra-Matrix 简化JupyterHub配置 - 修复重定向循环问题
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

# 关键：设置public URL并禁用重定向循环
c.JupyterHub.public_url = 'http://localhost:8080/jupyter/'

# 禁用自动重定向到login页面，避免循环
c.JupyterHub.default_url = '/jupyter/hub/home'
c.JupyterHub.redirect_to_server = False

# 数据目录
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)

# 数据库配置
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub_internal.sqlite'

# 代理配置
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'ai-infra-proxy-token')

# ===== 认证器配置 =====

# 使用自定义PostgreSQL+Redis认证器
c.JupyterHub.authenticator_class = PostgreSQLRedisAuthenticator

# PostgreSQL配置
c.PostgreSQLRedisAuthenticator.db_host = os.environ.get('DB_HOST', 'postgres')
c.PostgreSQLRedisAuthenticator.db_port = int(os.environ.get('DB_PORT', '5432'))
c.PostgreSQLRedisAuthenticator.db_name = os.environ.get('DB_NAME', 'ansible_playbook_generator')
c.PostgreSQLRedisAuthenticator.db_user = os.environ.get('DB_USER', 'postgres')
c.PostgreSQLRedisAuthenticator.db_password = os.environ.get('DB_PASSWORD', 'postgres')

# Redis配置
c.PostgreSQLRedisAuthenticator.redis_host = os.environ.get('REDIS_HOST', 'redis')
c.PostgreSQLRedisAuthenticator.redis_port = int(os.environ.get('REDIS_PORT', '6379'))
c.PostgreSQLRedisAuthenticator.redis_password = os.environ.get('REDIS_PASSWORD', '')

# JWT配置
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
    c.DockerSpawner.mem_limit = os.environ.get('JUPYTERHUB_MEM_LIMIT', '2G')
    c.DockerSpawner.cpu_limit = float(os.environ.get('JUPYTERHUB_CPU_LIMIT', '1.0'))
    
    # 环境变量
    c.DockerSpawner.environment = {
        'JUPYTER_ENABLE_LAB': '1',
        'GRANT_SUDO': 'yes',
        'CHOWN_HOME': 'yes'
    }
    
    # 容器清理
    c.DockerSpawner.remove = True
    
    # 调试模式
    if os.environ.get('DEBUG_MODE', 'false').lower() == 'true':
        c.DockerSpawner.debug = True

else:
    # 本地进程spawner
    from jupyterhub.spawner import LocalProcessSpawner
    c.JupyterHub.spawner_class = LocalProcessSpawner
    c.LocalProcessSpawner.create_system_users = False

# ===== 通用spawner配置 =====

# 设置默认启动Jupyter Lab而不是Notebook
c.Spawner.default_url = '/lab'

# ===== 管理配置 =====

# 管理员用户（可选）
admin_users_env = os.environ.get('JUPYTERHUB_ADMIN_USERS', '')
if admin_users_env:
    c.JupyterHub.admin_users = set(admin_users_env.split(','))

# ===== 服务配置 =====

# 空闲清理服务
services = []
idle_timeout = int(os.environ.get('JUPYTERHUB_IDLE_TIMEOUT', '3600'))  # 1小时
if idle_timeout > 0:
    services.append({
        'name': 'idle-culler',
        'admin': True,
        'command': [
            sys.executable, '-m', 'jupyterhub_idle_culler',
            f'--timeout={idle_timeout}',
            '--cull-every=600',
            '--concurrency=10',
            '--max-age=86400',  # 24小时最大年龄
        ],
    })

c.JupyterHub.services = services

# ===== 调试配置 =====

# 日志级别
if os.environ.get('DEBUG_MODE', 'false').lower() == 'true':
    c.JupyterHub.log_level = 'DEBUG'
    c.Application.log_level = 'DEBUG'
else:
    c.JupyterHub.log_level = 'INFO'
    c.Application.log_level = 'INFO'

# ===== 安全配置 =====

# 允许命名服务器（可选）
c.JupyterHub.allow_named_servers = False

# 内部SSL（生产环境建议启用）
c.JupyterHub.internal_ssl = False

# CORS配置
c.JupyterHub.tornado_settings = {
    'headers': {
        'Content-Security-Policy': "frame-ancestors 'self' http://localhost:8080"
    }
}

print("JupyterHub配置加载完成 - 简化版本，修复重定向循环问题")
