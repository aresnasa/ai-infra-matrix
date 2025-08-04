import os
import jwt
from datetime import datetime
from jupyterhub.auth import Authenticator
from traitlets import Unicode

class SimpleJWTAuthenticator(Authenticator):
    """简单有效的JWT认证器，支持自动登录"""

    jwt_secret = Unicode(
        "ai-infra-matrix-jwt-secret-2024",
        config=True,
        help="JWT签名密钥"
    )
    
    auto_login = True  # 启用自动登录

    async def authenticate(self, handler, data):
        """认证逻辑"""
        # 获取token - 多种方式
        token = None

        # 1. 从URL参数获取
        token = handler.get_argument('token', None)
        if token:
            self.log.info(f"从URL获取token: {token[:20]}...")

        # 2. 从cookie获取
        if not token:
            token = handler.get_cookie('jwt-token')
            if token:
                self.log.info("从cookie获取token")

        # 3. 从header获取
        if not token:
            auth_header = handler.request.headers.get('Authorization', '')
            if auth_header.startswith('Bearer '):
                token = auth_header[7:]
                self.log.info("从header获取token")

        if not token:
            self.log.warning("未找到JWT token，回退到密码认证")
            # 允许传统密码认证
            username = data.get('username', '')
            password = data.get('password', '')

            if username == 'admin' and password == 'admin123':
                self.log.info(f"密码认证成功: {username}")
                return username
            else:
                self.log.error("密码认证失败")
                return None

        # 验证JWT token
        try:
            payload = jwt.decode(token, self.jwt_secret, algorithms=['HS256'])
            username = payload.get('username')

            if not username:
                self.log.error("JWT中缺少用户名")
                return None

            # 检查过期
            exp = payload.get('exp', 0)
            if exp and datetime.utcnow().timestamp() > exp:
                self.log.error("JWT token已过期")
                return None

            self.log.info(f"JWT认证成功: {username}")

            # 设置认证cookie以保持会话
            handler.set_cookie(
                'jwt-token', 
                token,
                max_age=3600,
                path='/jupyter/',
                httponly=True
            )

            return username

        except Exception as e:
            self.log.error(f"JWT验证失败: {e}")
            return None

    async def pre_spawn_start(self, user, spawner):
        """在spawn之前的处理"""
        pass

# JupyterHub配置
c = get_config()

# 使用简化的JWT认证器
c.JupyterHub.authenticator_class = SimpleJWTAuthenticator

# 启用自动登录
c.Authenticator.auto_login = True

# 基本配置
c.Authenticator.admin_access = True
c.Authenticator.allowed_users = {'admin', 'user', 'test'}
c.Authenticator.admin_users = {'admin'}

# 网络配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'
c.JupyterHub.base_url = '/jupyter/'

# Spawner配置
c.JupyterHub.spawner_class = 'jupyterhub.spawner.SimpleLocalProcessSpawner'

# 日志
c.JupyterHub.log_level = 'DEBUG'  # 增加日志级别
c.Application.log_level = 'DEBUG'

# 数据存储
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

print("简化JWT认证器配置完成 - 启用自动登录")

# 使用简化的JWT认证器
c.JupyterHub.authenticator_class = SimpleJWTAuthenticator

# 基本配置
c.Authenticator.admin_access = True
c.Authenticator.allowed_users = {'admin', 'user', 'test'}
c.Authenticator.admin_users = {'admin'}

# 网络配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'
c.JupyterHub.base_url = '/jupyter/'

# Spawner配置
c.JupyterHub.spawner_class = 'jupyterhub.spawner.SimpleLocalProcessSpawner'

# 日志
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

# 数据存储
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

print("简化JWT认证器配置完成")
