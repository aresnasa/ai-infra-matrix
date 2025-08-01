#!/usr/bin/env python3
"""
JupyterHub Absolute No-Redirect Configuration
绝对阻止任何形式的重定向问题
"""

import os
import sys
from pathlib import Path
from ai_infra_auth import PostgreSQLRedisAuthenticator

# 获取JupyterHub配置对象
c = get_config()

print("🛑 LOADING ABSOLUTE NO-REDIRECT CONFIGURATION...")

# ===== 核心反重定向设置 - 第一优先级 =====
c.JupyterHub.redirect_to_server = False
c.Authenticator.auto_login = False

# ===== 基础网络配置 =====
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.base_url = '/jupyter/'

# ===== 强制指定登录和默认URL =====
c.JupyterHub.default_url = '/jupyter/hub/home'
c.JupyterHub.login_url = '/jupyter/hub/login'

# ===== 数据配置 =====
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'
c.JupyterHub.cookie_secret_file = str(project_data_dir / 'cookie_secret')

# ===== 认证配置 =====
c.JupyterHub.authenticator_class = PostgreSQLRedisAuthenticator
c.PostgreSQLRedisAuthenticator.database_url = 'postgresql://ai_infra_user:ai_infra_password@postgres:5432/ai_infra_matrix'
c.PostgreSQLRedisAuthenticator.redis_host = 'redis'
c.PostgreSQLRedisAuthenticator.redis_port = 6379
c.PostgreSQLRedisAuthenticator.redis_db = 0

# ===== 权限配置 =====
c.Authenticator.allow_all = True
c.Authenticator.admin_users = {'admin'}

# ===== Docker Spawner =====
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
c.DockerSpawner.image = 'jupyter/datascience-notebook:latest'
c.DockerSpawner.network_name = 'ai-infra-matrix_ai-infra-network'
c.DockerSpawner.remove = True
c.DockerSpawner.name_template = "jupyter-{username}"

# ===== 服务配置 =====
c.JupyterHub.services = [
    {
        'name': 'idle-culler',
        'command': [
            sys.executable, '-m', 'jupyterhub_idle_culler',
            '--timeout=3600',
            '--cull-every=7200',
            '--remove-named-servers'
        ],
        'admin': True,
    }
]

# ===== 日志配置 =====
c.JupyterHub.log_level = 'INFO'

print("✅ ABSOLUTE NO-REDIRECT CONFIGURATION LOADED")
