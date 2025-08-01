# 简化的JupyterHub配置文件，专注于修复DummyAuthenticator问题
import os
from jupyterhub.auth import DummyAuthenticator
from jupyterhub.spawner import LocalProcessSpawner

# 基本配置
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.base_url = '/jupyter/'

# 强制使用DummyAuthenticator
c.JupyterHub.authenticator_class = DummyAuthenticator

# DummyAuthenticator配置 - 允许所有用户
c.DummyAuthenticator.password = ""
c.Authenticator.allowed_users = {'admin', 'testuser', 'user1', 'user2'}
c.Authenticator.admin_users = {'admin'}

# 或者允许所有用户
# c.Authenticator.allow_all = True

# Spawner配置
c.JupyterHub.spawner_class = LocalProcessSpawner

# 数据库配置
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

# 安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/cookie_secret'

# 日志配置
c.JupyterHub.log_level = 'DEBUG'

print("🔧 简化配置已加载")
print(f"✅ 认证器: DummyAuthenticator")
print(f"✅ 允许的用户: {c.Authenticator.allowed_users}")
print(f"✅ 管理员用户: {c.Authenticator.admin_users}")
