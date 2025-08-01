# -*- coding: utf-8 -*-
"""
极简JupyterHub配置文件 - 确保基本功能正常
"""
import os

# 获取JupyterHub配置对象
c = get_config()

print("✅ LOADING MINIMAL JUPYTERHUB CONFIGURATION")

# ===== 核心设置 =====
c.JupyterHub.bind_url = 'http://0.0.0.0:8000/jupyter/'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081/jupyter/'
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0' 
c.JupyterHub.hub_port = 8081

# ===== 认证配置 =====
c.JupyterHub.authenticator_class = 'jupyterhub.auth.DummyAuthenticator'
c.DummyAuthenticator.password = "password"

# 设置允许的用户
c.Authenticator.allowed_users = {'admin', 'testuser'}
c.Authenticator.admin_users = {'admin'}

# ===== Spawner配置 =====
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
    print(f"📁 Created notebook directory for user: {username}")
    return user_notebook_dir

# 设置pre-spawn hook
c.Spawner.pre_spawn_hook = create_user_environment

# ===== 服务配置 =====
c.Spawner.start_timeout = 60
c.Spawner.http_timeout = 30

# ===== 调试和日志 =====
c.JupyterHub.log_level = 'INFO'

print("🔧 Minimal configuration loaded successfully!")
print(f"🔐 Authentication: DummyAuthenticator")
print(f"👤 Allowed users: admin, testuser")
print(f"👑 Admin users: admin")
print(f"📝 Spawner: LocalProcessSpawner")
