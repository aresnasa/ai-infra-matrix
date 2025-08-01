#!/usr/bin/env python3
"""
JupyterHub Clean Configuration
彻底解决无限重定向问题的干净配置
"""

import os
import sys
from pathlib import Path
from ai_infra_auth import PostgreSQLRedisAuthenticator

# 获取JupyterHub配置对象
c = get_config()

print("🚫 LOADING CLEAN NO-REDIRECT CONFIGURATION...")

# ===== 基础网络配置 - 避免冲突 =====
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.base_url = '/jupyter/'

# ===== 核心反重定向设置 =====
c.JupyterHub.redirect_to_server = False
c.JupyterHub.default_url = '/jupyter/hub/home'

# ===== 数据库配置 =====
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

# ===== 日志配置 =====
c.JupyterHub.log_level = 'INFO'

print("✅ CLEAN Configuration Loaded")
print("🚫 redirect_to_server = False")
print("🏠 default_url = /jupyter/hub/home")
