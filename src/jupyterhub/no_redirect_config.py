#!/usr/bin/env python3
"""
JupyterHub NO-REDIRECT Configuration
专门设计用于阻止无限重定向循环的最简化配置
"""

import os
import sys
from pathlib import Path
from ai_infra_auth import PostgreSQLRedisAuthenticator

# 获取JupyterHub配置对象
c = get_config()

print("🚫 LOADING NO-REDIRECT CONFIGURATION...")

# ===== 简化服务器配置 =====
# 直接使用端口绑定，避免URL冲突
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.hub_port = 8091

# ===== 基础URL设置 =====
c.JupyterHub.base_url = '/jupyter/'
c.JupyterHub.public_url = 'http://localhost:8080/jupyter/'

# ===== 阻止重定向的核心设置 =====
# 最关键：完全禁用服务器重定向
c.JupyterHub.redirect_to_server = False

# 设置默认着陆页面到hub主页
c.JupyterHub.default_url = '/jupyter/hub/home'

# 禁用自动登录避免重定向
c.JupyterHub.auto_login = False

# ===== 数据目录 =====
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'
c.JupyterHub.cookie_secret_file = str(project_data_dir / 'cookie_secret')

# ===== 认证器配置 =====
c.JupyterHub.authenticator_class = PostgreSQLRedisAuthenticator
c.PostgreSQLRedisAuthenticator.database_url = 'postgresql://ai_infra_user:ai_infra_password@postgres:5432/ai_infra_matrix'
c.PostgreSQLRedisAuthenticator.redis_host = 'redis'
c.PostgreSQLRedisAuthenticator.redis_port = 6379
c.PostgreSQLRedisAuthenticator.redis_db = 0
c.PostgreSQLRedisAuthenticator.auto_login = False

# ===== 权限设置 =====
c.Authenticator.allow_all = True
admin_users_env = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin')
if admin_users_env:
    c.Authenticator.admin_users = set(admin_users_env.split(','))

# ===== Docker Spawner 基础配置 =====
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
c.DockerSpawner.image = 'jupyter/datascience-notebook:latest'
c.DockerSpawner.network_name = 'ai-infra-matrix_ai-infra-network'
c.DockerSpawner.remove = True
c.DockerSpawner.name_template = "jupyter-{username}"

# Spawner重定向设置
c.Spawner.default_url = '/lab'

# ===== 服务 - 简化版idle-culler =====
c.JupyterHub.services = [
    {
        'name': 'idle-culler',
        'command': [
            sys.executable, '-m', 'jupyterhub_idle_culler',
            '--timeout=3600',
            '--cull-every=7200',
            '--remove-named-servers'
        ],
    }
]

# ===== 日志设置 =====
c.JupyterHub.log_level = 'INFO'

print("✅ NO-REDIRECT Configuration Loaded")
print("🚫 redirect_to_server = False")
print("🏠 default_url = /jupyter/hub/home") 
print("🔒 auto_login = False")
print("📝 Simplified configuration to prevent redirect loops")
