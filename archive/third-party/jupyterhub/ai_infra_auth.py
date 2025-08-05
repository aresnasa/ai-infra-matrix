"""
AI基础设施矩阵统一认证器
集成JupyterHub与ai-infra-matrix后端系统的用户认证和授权
"""

import json
import requests
import os
from urllib.parse import urljoin
from tornado import web
from tornado.httputil import url_concat
from tornado.log import app_log

from jupyterhub.auth import Authenticator
from jupyterhub.handlers import BaseHandler
from jupyterhub.utils import url_path_join
from traitlets import Unicode, Bool, Int, default


class AIInfraMatrixAuth(BaseHandler):
    """AI基础设施矩阵认证处理器"""
    
    async def get(self):
        """处理认证GET请求 - 显示登录表单"""
        # 检查是否已经有JWT token参数
        jwt_token = self.get_argument('token', None)
        if jwt_token:
            # 有token，尝试验证并登录
            user = await self._verify_jwt_token(jwt_token)
            if user:
                # token有效，设置用户并重定向
                self.set_login_cookie(user)
                next_url = self.get_argument('next', None) or self.hub.base_url
                self.redirect(next_url)
                return
        
        # 没有token或token无效，显示登录页面
        # 使用标准的JupyterHub登录表单而不是重定向
        html = self.render_template('login.html',
            next=self.get_argument('next', ''),
            username=self.get_argument('username', ''),
            login_error=self.get_argument('error', ''),
            custom_html=self.authenticator.custom_html,
            login_url=self.hub.base_url + 'login',
            authenticator_login_url=url_path_join(
                self.hub.base_url, 'ai-infra-login'
            ),
        )
        self.finish(html)
    
    async def _verify_jwt_token(self, token):
        """验证JWT token并返回用户信息"""
        try:
            # 调用后端验证token
            authenticator = self.settings['authenticator']
            response = await authenticator._api_request('POST', '/api/auth/verify-token', {
                'token': token
            })
            
            if response and response.get('success'):
                user_info = response.get('user', {})
                return user_info.get('username')
            
            return None
        except Exception as e:
            self.log.error(f"JWT token verification error: {e}")
            return None


class AIInfraMatrixAuthenticator(Authenticator):
    """
    AI基础设施矩阵统一认证器
    与后端API进行集成，实现统一的用户认证和token管理
    """
    
    # 后端API配置
    backend_api_url = Unicode(
        config=True,
        help="""
        AI基础设施矩阵后端API地址
        默认: http://localhost:8080
        """
    )
    
    @default('backend_api_url')
    def _default_backend_api_url(self):
        return os.environ.get('AI_INFRA_BACKEND_URL', 'http://localhost:8080')
    
    backend_api_token = Unicode(
        config=True,
        help="""
        访问后端API的认证token
        """
    )
    
    @default('backend_api_token')
    def _default_backend_api_token(self):
        return os.environ.get('AI_INFRA_API_TOKEN', '')
    
    # 认证配置
    enable_auth_state = Bool(
        True,
        config=True,
        help="""
        启用认证状态存储，用于保存用户JWT token等信息
        """
    )
    
    auto_login = Bool(
        False,
        config=True,
        help="""
        是否启用自动登录（基于JWT token）
        """
    )
    
    token_refresh_threshold = Int(
        300,  # 5分钟
        config=True,
        help="""
        Token刷新阈值（秒），当token即将过期时自动刷新
        """
    )
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'User-Agent': 'JupyterHub-AIInfraMatrix/1.0'
        })
        if self.backend_api_token:
            self.session.headers.update({
                'Authorization': f'Bearer {self.backend_api_token}'
            })
    
    async def authenticate(self, handler, data):
        """
        主要认证方法
        支持用户名密码认证和JWT token认证
        """
        username = data.get('username', '').strip()
        password = data.get('password', '').strip()
        jwt_token = data.get('jwt_token', '').strip()
        
        # 优先使用JWT token认证
        if jwt_token:
            return await self._authenticate_with_jwt(jwt_token)
        
        # 用户名密码认证
        if username and password:
            return await self._authenticate_with_password(username, password)
        
        self.log.error("No valid authentication credentials provided")
        return None
    
    async def _authenticate_with_jwt(self, jwt_token):
        """使用JWT token进行认证"""
        try:
            # 调用后端API验证JWT token
            response = await self._api_request('POST', '/api/auth/verify-token', {
                'token': jwt_token
            })
            
            if response and response.get('valid'):
                user_info = response.get('user', {})
                username = user_info.get('username')
                
                if username:
                    # 构建认证结果
                    auth_model = {
                        'name': username,
                        'auth_state': {
                            'jwt_token': jwt_token,
                            'user_info': user_info,
                            'token_expires_at': response.get('expires_at'),
                            'auth_method': 'jwt'
                        }
                    }
                    
                    self.log.info(f"JWT authentication successful for user: {username}")
                    return auth_model
            
            self.log.error("JWT token validation failed")
            return None
            
        except Exception as e:
            self.log.error(f"JWT authentication error: {e}")
            return None
    
    async def _authenticate_with_password(self, username, password):
        """使用用户名密码进行认证"""
        try:
            # 调用后端API进行用户认证
            response = await self._api_request('POST', '/api/auth/login', {
                'username': username,
                'password': password
            })
            
            if response and response.get('success'):
                user_info = response.get('user', {})
                jwt_token = response.get('token')
                
                # 构建认证结果
                auth_model = {
                    'name': username,
                    'auth_state': {
                        'jwt_token': jwt_token,
                        'user_info': user_info,
                        'token_expires_at': response.get('expires_at'),
                        'auth_method': 'password'
                    }
                }
                
                self.log.info(f"Password authentication successful for user: {username}")
                return auth_model
            
            self.log.error(f"Password authentication failed for user: {username}")
            return None
            
        except Exception as e:
            self.log.error(f"Password authentication error: {e}")
            return None
    
    async def pre_spawn_start(self, user, spawner):
        """
        在启动用户服务器前的钩子
        设置环境变量和配置
        """
        auth_state = await user.get_auth_state()
        if auth_state:
            jwt_token = auth_state.get('jwt_token')
            user_info = auth_state.get('user_info', {})
            
            # 设置环境变量
            if jwt_token:
                spawner.environment['AI_INFRA_JWT_TOKEN'] = jwt_token
                spawner.environment['AI_INFRA_USER_ID'] = str(user_info.get('id', ''))
                spawner.environment['AI_INFRA_USERNAME'] = user_info.get('username', '')
                spawner.environment['AI_INFRA_EMAIL'] = user_info.get('email', '')
                spawner.environment['AI_INFRA_BACKEND_URL'] = self.backend_api_url
            
            # 检查token是否即将过期，如果是则刷新
            await self._refresh_token_if_needed(user, auth_state)
    
    async def _refresh_token_if_needed(self, user, auth_state):
        """检查并刷新即将过期的token"""
        try:
            expires_at = auth_state.get('token_expires_at')
            if not expires_at:
                return
            
            # 检查是否即将过期
            import datetime
            expire_time = datetime.datetime.fromisoformat(expires_at.replace('Z', '+00:00'))
            now = datetime.datetime.now(datetime.timezone.utc)
            time_left = (expire_time - now).total_seconds()
            
            if time_left < self.token_refresh_threshold:
                # 刷新token
                jwt_token = auth_state.get('jwt_token')
                response = await self._api_request('POST', '/api/auth/refresh-token', {
                    'token': jwt_token
                })
                
                if response and response.get('success'):
                    # 更新认证状态
                    auth_state['jwt_token'] = response.get('token')
                    auth_state['token_expires_at'] = response.get('expires_at')
                    await user.save_auth_state(auth_state)
                    
                    self.log.info(f"Token refreshed for user: {user.name}")
                
        except Exception as e:
            self.log.error(f"Token refresh error: {e}")
    
    async def _api_request(self, method, endpoint, data=None):
        """向后端API发送请求"""
        try:
            url = urljoin(self.backend_api_url, endpoint)
            
            if method.upper() == 'GET':
                response = self.session.get(url, params=data)
            else:
                response = self.session.request(method, url, json=data)
            
            response.raise_for_status()
            return response.json()
            
        except requests.exceptions.RequestException as e:
            self.log.error(f"API request failed: {e}")
            return None
        except json.JSONDecodeError as e:
            self.log.error(f"API response JSON decode error: {e}")
            return None
    
    def get_handlers(self, app):
        """获取认证处理器"""
        return [
            (r'/ai-infra-login', AIInfraMatrixAuth),
        ]
    
    def login_url(self, base_url):
        """生成登录URL - 使用标准的JupyterHub登录"""
        return url_path_join(base_url, 'login')


# 便捷的配置函数
def configure_ai_infra_auth(c, backend_url=None, api_token=None):
    """
    配置AI基础设施矩阵认证器的便捷函数
    
    参数:
        c: JupyterHub配置对象
        backend_url: 后端API地址
        api_token: API访问token
    """
    # 设置认证器
    c.JupyterHub.authenticator_class = AIInfraMatrixAuthenticator
    
    # 配置后端连接
    if backend_url:
        c.AIInfraMatrixAuthenticator.backend_api_url = backend_url
    if api_token:
        c.AIInfraMatrixAuthenticator.backend_api_token = api_token
    
    # 启用认证状态
    c.AIInfraMatrixAuthenticator.enable_auth_state = True
    
    # 配置admin用户（从环境变量读取）
    admin_users = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin').split(',')
    c.Authenticator.admin_users = {user.strip() for user in admin_users if user.strip()}
    
    print("AI基础设施矩阵统一认证配置完成")
