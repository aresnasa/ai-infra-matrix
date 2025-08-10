import os
import jwt
from datetime import datetime
from jupyterhub.auth import Authenticator
from traitlets import Unicode

class SimplePasswordAuthenticator(Authenticator):
    """简单的密码认证器，避免重定向循环"""
    
    # 禁用自动登录以避免重定向循环
    auto_login = False
    
    async def authenticate(self, handler, data):
        """基础密码认证逻辑"""
        username = data.get('username', '')
        password = data.get('password', '')
        
        # 简单的用户名密码验证
        if username == 'admin' and password == 'admin123':
            self.log.info(f"认证成功: {username}")
            return username
        elif username == 'user' and password == 'user123':
            self.log.info(f"认证成功: {username}")
            return username
        else:
            self.log.error(f"认证失败: {username}")
            return None

# JupyterHub配置
c = get_config()

# 使用简单的密码认证器
c.JupyterHub.authenticator_class = SimplePasswordAuthenticator

# 基本配置
c.Authenticator.admin_access = True
c.Authenticator.allowed_users = {'admin', 'user', 'test'}
c.Authenticator.admin_users = {'admin'}

# 网络配置 - 修复重定向循环问题
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'
c.JupyterHub.base_url = '/jupyter/'  # 设置正确的base_url与nginx代理路径匹配

# Spawner配置
c.JupyterHub.spawner_class = 'jupyterhub.spawner.SimpleLocalProcessSpawner'

# 日志
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

# 数据存储
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

# 修复iframe加载问题 - 允许在iframe中嵌入
c.JupyterHub.tornado_settings = {
    'headers': {
        'Content-Security-Policy': "frame-ancestors 'self' http://localhost:8080 https://localhost:8443"
    }
}

print("简单密码认证器配置完成 - 已禁用自动登录以避免重定向循环")
