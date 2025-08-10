# JupyterHub 简化配置 - 修复重定向循环问题
# 版本: 2025-08-10

import os
from jupyterhub.auth import DummyAuthenticator

# 基础网络配置 - 关键修复
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'
c.JupyterHub.base_url = '/'  # 使用根路径，避免重定向循环

# 使用简单的DummyAuthenticator - 避免复杂的自动登录逻辑
c.JupyterHub.authenticator_class = DummyAuthenticator
c.DummyAuthenticator.password = "admin123"

# 用户配置
c.Authenticator.admin_access = True
c.Authenticator.allowed_users = {'admin', 'user', 'test'}
c.Authenticator.admin_users = {'admin'}

# Spawner配置
c.JupyterHub.spawner_class = 'jupyterhub.spawner.SimpleLocalProcessSpawner'

# 日志配置
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

# 数据存储
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

# 安全配置
c.JupyterHub.cookie_max_age_days = 1

# 禁用不必要的功能
c.JupyterHub.cleanup_servers = False

print("✅ JupyterHub 简化配置加载完成")
print("🔧 配置详情:")
print(f"   - 监听地址: {c.JupyterHub.bind_url}")
print(f"   - 基础路径: {c.JupyterHub.base_url}")
print(f"   - 认证器: DummyAuthenticator")
print(f"   - 默认密码: admin123")
