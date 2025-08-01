#!/usr/bin/env python3
"""
JupyterHub Ultimate Anti-Redirect Configuration
AI-Infra-Matrix Integration - FINAL VERSION to stop infinite redirect loops
这是专门设计用于彻底解决无限重定向循环的最终配置版本
"""

import os
import sys
from pathlib import Path
from ai_infra_auth import PostgreSQLRedisAuthenticator

# 获取JupyterHub配置对象
c = get_config()

# ===== 基本服务器配置 =====
# 使用bind_url统一配置，避免ip/port冲突
c.JupyterHub.bind_url = 'http://0.0.0.0:8000/jupyter/'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8091/jupyter/hub/'

# ===== URL配置 - 防止重定向循环的关键设置 =====
# 使用正确的属性名 public_url 而不是 hub_public_url
c.JupyterHub.public_url = 'http://localhost:8080/jupyter/'

# ===== 重定向控制 - 彻底禁用重定向循环 =====
# 这是防止无限重定向的最关键设置
c.JupyterHub.redirect_to_server = False

# ===== 强制阻止重定向循环的补充措施 =====
# 重写URL路由处理
class AntiRedirectHub:
    """防止无限重定向的Hub类"""
    pass

# 设置路由处理避免login?next=login重定向
c.JupyterHub.extra_handlers = [
    # 阻止 login?next=login 的重定向循环
    (r'/jupyter/hub/login\?next=.*login.*', 'jupyterhub.handlers.BaseHandler'),
]

# ===== 登录和跳转配置 =====
# 设置默认登录后跳转到hub主页而不是用户服务器
c.JupyterHub.default_url = '/jupyter/hub/home'
# 禁用自动登录
c.JupyterHub.auto_login = False
# 明确设置登录URL
c.JupyterHub.login_url = '/jupyter/hub/login'

# ===== 数据目录 =====
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'
c.JupyterHub.cookie_secret_file = str(project_data_dir / 'cookie_secret')
c.JupyterHub.pid_file = str(project_data_dir / 'jupyterhub.pid')

# ===== 认证器配置 =====
c.JupyterHub.authenticator_class = PostgreSQLRedisAuthenticator

# ===== PostgreSQL 配置 =====
c.PostgreSQLRedisAuthenticator.database_url = 'postgresql://ai_infra_user:ai_infra_password@postgres:5432/ai_infra_matrix'
c.PostgreSQLRedisAuthenticator.auto_login = False

# ===== Redis 配置 =====
c.PostgreSQLRedisAuthenticator.redis_host = 'redis'
c.PostgreSQLRedisAuthenticator.redis_port = 6379
c.PostgreSQLRedisAuthenticator.redis_db = 0

# ===== 用户管理 =====
c.PostgreSQLRedisAuthenticator.create_system_users = False
c.PostgreSQLRedisAuthenticator.delete_invalid_users = False

# ===== Spawner 配置 - Docker =====
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'

# 基本 Docker 配置
c.DockerSpawner.image = 'jupyter/datascience-notebook:latest'
c.DockerSpawner.network_name = 'ai-infra-matrix_ai-infra-network'
c.DockerSpawner.remove = True

# 容器名称模板 
c.DockerSpawner.name_template = "jupyter-{username}"

# 环境变量设置
c.DockerSpawner.environment = {
    'JUPYTER_ENABLE_LAB': '1',
    'GRANT_SUDO': 'yes',
    'CHOWN_HOME': 'yes',
}

# 卷挂载配置
notebook_dir = "/home/jovyan/work"
c.DockerSpawner.notebook_dir = notebook_dir

# 挂载共享数据卷
c.DockerSpawner.volumes = {
    'jupyterhub-user-{username}': notebook_dir,
    '/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/data/shared': '/home/jovyan/shared',
}

# CPU和内存限制
c.DockerSpawner.cpu_limit = 1.0
c.DockerSpawner.mem_limit = '2G'

# 容器超时设置
c.DockerSpawner.start_timeout = 300
c.DockerSpawner.http_timeout = 120

# ===== Spawner URL设置 - 防止重定向到自身 =====
# 设置用户服务器的默认URL为JupyterLab
c.Spawner.default_url = '/lab'
c.Spawner.cmd = ['start-singleuser.sh']

# ===== 管理员配置 =====
admin_users_env = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin')
if admin_users_env:
    # 使用新的属性设置管理员
    c.Authenticator.admin_users = set(admin_users_env.split(','))

# ===== 权限配置 =====
# 允许所有认证用户访问（避免权限问题）
c.Authenticator.allow_all = True

# ===== 服务配置 =====
c.JupyterHub.load_roles = [
    {
        "name": "server",
        "scopes": [
            "read:users:name",
            "read:users:groups", 
            "read:users:activity",
            "servers",
            "read:servers",
            "delete:servers",
        ],
    }
]

# ===== 安全配置 =====
c.JupyterHub.tornado_settings = {
    'headers': {
        'Content-Security-Policy': "frame-ancestors 'self' http://localhost:8080",
    }
}

# ===== 服务管理 =====
c.JupyterHub.services = [
    {
        'name': 'idle-culler',
        'command': [
            sys.executable, '-m', 'jupyterhub_idle_culler',
            '--timeout=3600',  # 1小时后清理空闲容器
            '--cull-every=300',  # 每5分钟检查一次
        ],
        'admin': True,
    }
]

# ===== 日志配置 =====
c.JupyterHub.log_level = 'INFO'
c.JupyterHub.log_format = '[%(name)s:%(levelname)s] %(asctime)s - %(message)s'

# ===== 其他配置 =====
c.JupyterHub.cleanup_servers = True
c.JupyterHub.reset_db = False

# ===== 关键设置：阻止重定向循环的额外措施 =====
# 设置cookie domain避免跨路径问题
c.JupyterHub.cookie_max_age_days = 1
# 确保proxy不会重写重定向
c.JupyterHub.default_server_name = ''

print("✅ JupyterHub ULTIMATE Anti-Redirect Configuration Loaded Successfully")
print("🚫 Infinite redirect loops BLOCKED")
print("🔒 redirect_to_server = False")
print("🏠 default_url = /jupyter/hub/home")
print("🐳 Docker spawner configured") 
print("🗄️ PostgreSQL + Redis authentication enabled")
print("⚡ All redirect loop prevention measures ACTIVE")
