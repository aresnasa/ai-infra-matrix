"""
JupyterHub简化配置 - 避免重定向循环
使用基本的密码认证，禁用自动登录
"""
import os
from jupyterhub.auth import Authenticator
from dockerspawner import DockerSpawner
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

print("🚀 简化JupyterHub配置加载中...")

# 使用简单的密码认证器
c.JupyterHub.authenticator_class = SimplePasswordAuthenticator

# 基本配置
c.Authenticator.admin_access = True
c.Authenticator.allowed_users = {'admin', 'user', 'test'}
c.Authenticator.admin_users = {'admin'}

# 网络配置 - 与nginx代理路径匹配
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'
c.JupyterHub.base_url = '/jupyter/'  # 必须与nginx代理路径匹配

# Spawner配置 - 使用DockerSpawner
c.JupyterHub.spawner_class = DockerSpawner
c.DockerSpawner.image = 'jupyter/minimal-notebook:latest'
c.DockerSpawner.network_name = 'ai-infra-matrix_default'
c.DockerSpawner.remove = True

# Docker配置
c.DockerSpawner.extra_host_config = {
    'network_mode': 'ai-infra-matrix_default'
}

# 数据库配置（基础PostgreSQL）
c.JupyterHub.db_url = f"postgresql://postgres:postgres@postgres:5432/jupyterhub_db"

# 日志
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

print("✅ 简化JupyterHub配置加载完成 - 无重定向循环")
