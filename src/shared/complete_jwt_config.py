
# 完整的JWT认证器配置
import os
import jwt
from datetime import datetime
from jupyterhub.auth import Authenticator
from jupyterhub.handlers import BaseHandler
from tornado import web
from traitlets import Unicode, Bool

class JWTAuthenticator(Authenticator):
    """
    JWT Token认证器 - 支持多种token传递方式
    """

    jwt_secret = Unicode(
        config=True,
        help="JWT签名密钥"
    ).tag(config=True)

    jwt_algorithm = Unicode(
        default_value='HS256',
        config=True,
        help="JWT算法"
    ).tag(config=True)

    cookie_name = Unicode(
        default_value='jupyterhub-jwt-token',
        config=True,
        help="JWT token的cookie名称"
    ).tag(config=True)

    def _get_token_from_request(self, handler):
        """从请求中提取JWT token"""
        # 1. 从URL参数获取
        token = handler.get_argument('token', None)
        if token:
            self.log.info("从URL参数获取到token")
            return token

        # 2. 从Authorization header获取
        auth_header = handler.request.headers.get('Authorization', '')
        if auth_header.startswith('Bearer '):
            token = auth_header[7:]
            self.log.info("从Authorization header获取到token")
            return token

        # 3. 从cookie获取
        token = handler.get_cookie(self.cookie_name)
        if token:
            self.log.info("从cookie获取到token")
            return token

        # 4. 从自定义header获取
        token = handler.request.headers.get('X-JWT-Token')
        if token:
            self.log.info("从X-JWT-Token header获取到token")
            return token

        return None

    async def authenticate(self, handler, data):
        """认证用户"""
        try:
            # 获取token
            token = self._get_token_from_request(handler)

            if not token:
                self.log.warning("未找到JWT token")
                return None

            # 验证token
            try:
                payload = jwt.decode(
                    token, 
                    self.jwt_secret, 
                    algorithms=[self.jwt_algorithm]
                )

                username = payload.get('username')
                if not username:
                    self.log.error("JWT token中缺少username")
                    return None

                # 检查token是否过期
                exp = payload.get('exp')
                if exp and datetime.utcnow().timestamp() > exp:
                    self.log.error("JWT token已过期")
                    return None

                self.log.info(f"JWT认证成功，用户: {username}")

                # 设置cookie以保持登录状态
                handler.set_cookie(
                    self.cookie_name,
                    token,
                    max_age=3600,  # 1小时
                    secure=False,  # 开发环境可以设置为False
                    httponly=True,
                    path='/jupyter/'
                )

                return {
                    'name': username,
                    'auth_model': {
                        'username': username,
                        'roles': payload.get('roles', []),
                        'user_id': payload.get('user_id')
                    }
                }

            except jwt.InvalidTokenError as e:
                self.log.error(f"JWT token验证失败: {e}")
                return None

        except Exception as e:
            self.log.error(f"认证过程出错: {e}")
            return None

# JupyterHub配置
c = get_config()

# 使用JWT认证器
c.JupyterHub.authenticator_class = JWTAuthenticator
c.JWTAuthenticator.jwt_secret = "ai-infra-matrix-jwt-secret-2024"

# 管理员配置
c.Authenticator.admin_access = True
c.Authenticator.allowed_users = {'admin', 'user', 'test'}
c.Authenticator.admin_users = {'admin'}

# 网络配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'
c.JupyterHub.base_url = '/jupyter/'

# 日志配置
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

# Spawner配置
c.JupyterHub.spawner_class = 'jupyterhub.spawner.SimpleLocalProcessSpawner'
c.Spawner.start_timeout = 60
c.Spawner.http_timeout = 60

# Cookie和安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

print("JWT认证器配置完成")
