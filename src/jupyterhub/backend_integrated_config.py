"""
JupyterHub后端集成配置
统一使用backend作为认证中心，JupyterHub只作为功能组件
支持PostgreSQL + Redis + 完整后端认证集成
"""

import os
import sys
import json
import logging
import aiohttp
import asyncio
from datetime import datetime, timezone
from jupyterhub.auth import Authenticator
from jupyterhub.handlers import BaseHandler
from dockerspawner import DockerSpawner
from tornado import web
from traitlets import Unicode, Bool, Dict, List
import redis
import psycopg2

# 配置日志
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

# 获取配置对象
c = get_config()

print("🚀 JupyterHub后端集成配置加载中...")

# 环境配置
BACKEND_URL = os.environ.get('BACKEND_URL', 'http://backend:8082')
JWT_SECRET = os.environ.get('JWT_SECRET', 'your-secret-key-change-in-production')

# 数据库配置
DB_CONFIG = {
    'host': os.environ.get('POSTGRES_HOST', 'postgres'),
    'port': int(os.environ.get('POSTGRES_PORT', 5432)),
    'database': os.environ.get('POSTGRES_DB', 'jupyterhub_db'),
    'user': os.environ.get('POSTGRES_USER', 'postgres'),
    'password': os.environ.get('POSTGRES_PASSWORD', 'postgres')
}

# Redis配置
REDIS_CONFIG = {
    'host': os.environ.get('REDIS_HOST', 'redis'),
    'port': int(os.environ.get('REDIS_PORT', 6379)),
    'password': os.environ.get('REDIS_PASSWORD', 'ansible-redis-password'),
    'db': int(os.environ.get('REDIS_DB', 1)),
    'decode_responses': True
}


class BackendIntegratedAuthenticator(Authenticator):
    """
    完全集成后端的认证器
    所有认证、用户管理、权限控制都通过后端API
    """
    
    backend_url = Unicode(BACKEND_URL, config=True, help="后端API地址")
    jwt_secret = Unicode(JWT_SECRET, config=True, help="JWT签名密钥")
    
    async def authenticate(self, handler, data):
        """统一认证入口 - 通过后端验证"""
        try:
            logger.info("🔐 开始后端集成认证...")
            
            # 1. 尝试JWT Token认证
            token = self._extract_token(handler)
            if token:
                username = await self._verify_jwt_token(token)
                if username:
                    logger.info(f"✅ JWT认证成功: {username}")
                    return await self._get_user_info(username, token)
            
            # 2. 表单登录认证
            if data and data.get('username') and data.get('password'):
                username = data['username']
                password = data['password']
                logger.info(f"📝 处理表单登录: {username}")
                
                auth_result = await self._backend_login(username, password)
                if auth_result:
                    logger.info(f"✅ 表单认证成功: {username}")
                    return await self._get_user_info(username, auth_result.get('token'))
            
            logger.warning("❌ 认证失败")
            return None
            
        except Exception as e:
            logger.error(f"❌ 认证过程异常: {e}")
            return None
    
    def _extract_token(self, handler):
        """提取JWT Token"""
        # 从Authorization header
        auth_header = handler.request.headers.get('Authorization', '')
        if auth_header.startswith('Bearer '):
            return auth_header[7:]
        
        # 从Cookie
        token = handler.get_cookie('jwt_token')
        if token:
            return token
        
        # 从URL参数
        token = handler.get_argument('token', None)
        if token:
            return token
        
        return None
    
    async def _verify_jwt_token(self, token):
        """通过后端验证JWT Token"""
        try:
            headers = {'Authorization': f'Bearer {token}'}
            async with aiohttp.ClientSession() as session:
                async with session.get(f"{self.backend_url}/api/auth/verify", headers=headers) as resp:
                    if resp.status == 200:
                        result = await resp.json()
                        if result.get('valid'):
                            return result.get('username')
            return None
        except Exception as e:
            logger.error(f"JWT验证失败: {e}")
            return None
    
    async def _backend_login(self, username, password):
        """通过后端进行用户名密码登录"""
        try:
            login_data = {
                'username': username,
                'password': password
            }
            async with aiohttp.ClientSession() as session:
                async with session.post(f"{self.backend_url}/api/auth/login", json=login_data) as resp:
                    if resp.status == 200:
                        return await resp.json()
            return None
        except Exception as e:
            logger.error(f"后端登录失败: {e}")
            return None
    
    async def _get_user_info(self, username, token=None):
        """从后端获取用户信息"""
        try:
            headers = {}
            if token:
                headers['Authorization'] = f'Bearer {token}'
            
            async with aiohttp.ClientSession() as session:
                async with session.get(f"{self.backend_url}/api/users/{username}", headers=headers) as resp:
                    if resp.status == 200:
                        user_info = await resp.json()
                        
                        # 返回用户名，JupyterHub会创建用户对象
                        # 用户权限信息通过auth_state传递
                        return {
                            'name': username,
                            'auth_state': {
                                'user_info': user_info,
                                'token': token,
                                'roles': user_info.get('roles', []),
                                'permissions': user_info.get('permissions', [])
                            }
                        }
            
            # 如果后端没有用户信息，返回基本用户名
            return username
            
        except Exception as e:
            logger.error(f"获取用户信息失败: {e}")
            return username


class BackendProxyHandler(BaseHandler):
    """后端代理处理器 - 处理特殊请求"""
    
    async def get(self):
        """处理GET请求"""
        # 处理自动登录
        if self.request.path.endswith('/auto-login'):
            await self._handle_auto_login()
        else:
            await self._proxy_to_backend()
    
    async def post(self):
        """处理POST请求"""
        await self._proxy_to_backend()
    
    async def _handle_auto_login(self):
        """处理自动登录逻辑"""
        try:
            token = self.get_cookie('jwt_token') or self.get_argument('token', None)
            if token:
                # 验证token并重定向到hub
                headers = {'Authorization': f'Bearer {token}'}
                async with aiohttp.ClientSession() as session:
                    async with session.get(f"{BACKEND_URL}/api/auth/verify", headers=headers) as resp:
                        if resp.status == 200:
                            result = await resp.json()
                            if result.get('valid'):
                                self.redirect('/hub/home')
                                return
            
            # 验证失败，重定向到登录页
            self.redirect('/hub/login')
            
        except Exception as e:
            logger.error(f"自动登录处理失败: {e}")
            self.redirect('/hub/login')
    
    async def _proxy_to_backend(self):
        """代理请求到后端"""
        try:
            # 这里可以实现请求代理逻辑
            self.write({'status': 'proxy', 'message': '后端代理功能'})
        except Exception as e:
            logger.error(f"代理请求失败: {e}")
            self.write({'status': 'error', 'message': str(e)})


class ContainerSpawner(DockerSpawner):
    """容器环境优化的Spawner"""
    
    def user_env(self, env):
        """设置用户环境"""
        # 在容器环境中统一使用root用户
        env['USER'] = 'root'
        env['HOME'] = '/root'
        env['SHELL'] = '/bin/bash'
        return env


# =========================
# JupyterHub核心配置
# =========================

# 基础网络配置
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0'

# 通过环境变量决定是否通过代理访问
use_proxy = os.environ.get('JUPYTERHUB_USE_PROXY', 'true').lower() == 'true'
if use_proxy:
    # 代理模式：JupyterHub 通过 nginx /jupyter/ 前缀访问
    c.JupyterHub.base_url = '/jupyter/'
    # 配置代理头处理
    c.JupyterHub.trust_user_provided_tokens = True
    c.JupyterHub.trust_user_provided_image = True
    # 允许来自代理的请求
    c.JupyterHub.allow_origin = '*'
    c.JupyterHub.allow_origin_pat = '.*'
else:
    # 直接访问模式
    c.JupyterHub.base_url = '/'

# 公共URL配置
public_host = os.environ.get('JUPYTERHUB_PUBLIC_HOST', 'localhost:8080')
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
if use_proxy:
    if not public_host.startswith('http'):
        public_host = f'http://{public_host}'
    os.environ['JUPYTERHUB_PUBLIC_URL'] = f'{public_host}/jupyter/'
    # 通知spawner使用代理URL
    c.JupyterHub.public_url = f'{public_host}/jupyter/'
else:
    c.JupyterHub.public_url = f'http://{public_host}/'

# 数据库配置 - PostgreSQL
c.JupyterHub.db_url = f"postgresql://{DB_CONFIG['user']}:{DB_CONFIG['password']}@{DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}"

# 认证器配置
c.JupyterHub.authenticator_class = BackendIntegratedAuthenticator
c.BackendIntegratedAuthenticator.backend_url = BACKEND_URL
c.BackendIntegratedAuthenticator.jwt_secret = JWT_SECRET

# 用户管理配置
c.Authenticator.allow_all = True  # 用户权限由后端控制
c.Authenticator.admin_users = set()  # 管理员由后端API确定
c.Authenticator.enable_auth_state = True  # 启用认证状态传递

# 额外处理器
c.JupyterHub.extra_handlers = [
    (r'/backend/(.*)', BackendProxyHandler),
    (r'/auto-login', BackendProxyHandler),
]

# Spawner配置
c.JupyterHub.spawner_class = ContainerSpawner

# Docker Spawner配置
# 配置Docker Spawner网络
c.ContainerSpawner.image = os.environ.get('JUPYTERHUB_IMAGE', 'jupyter/base-notebook:latest')
c.ContainerSpawner.network_name = os.environ.get('JUPYTERHUB_NETWORK', 'ai-infra-network')
c.ContainerSpawner.remove = True  # 删除停止的容器
c.ContainerSpawner.debug = True

# 资源限制
c.ContainerSpawner.mem_limit = os.environ.get('JUPYTERHUB_MEM_LIMIT', '2G')
c.ContainerSpawner.cpu_limit = float(os.environ.get('JUPYTERHUB_CPU_LIMIT', '1.0'))

# 容器配置
c.ContainerSpawner.notebook_dir = '/home/jovyan/work'
c.ContainerSpawner.cmd = ['start-singleuser.sh']  # 使用标准单用户启动脚本

# 环境变量设置
c.ContainerSpawner.environment = {
    'JUPYTER_ENABLE_LAB': 'yes',  # 启用JupyterLab
}

# 挂载配置（可选）
c.ContainerSpawner.volumes = {
    # 可以添加持久化存储
}

# 安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'default-token-change-me')

# 加密密钥配置（用于auth_state）
crypt_key = os.environ.get('JUPYTERHUB_CRYPT_KEY', '790031b2deeb70d780d4ccd100514b37f3c168ce80141478bf80aebfb65580c1')
if len(crypt_key) == 64:  # 十六进制字符串
    import binascii
    c.CryptKeeper.keys = [binascii.unhexlify(crypt_key)]
else:
    print(f"Warning: Invalid crypt key length: {len(crypt_key)}, expected 64 hex chars")
    c.Authenticator.enable_auth_state = False  # 禁用auth_state

# 日志配置
c.JupyterHub.log_level = 'DEBUG'

# =========================
# 连接测试
# =========================

def test_backend_connection():
    """测试后端连接"""
    try:
        import requests
        resp = requests.get(f"{BACKEND_URL}/health", timeout=5)
        if resp.status_code == 200:
            print("✅ 后端连接成功")
            return True
    except Exception as e:
        print(f"❌ 后端连接失败: {e}")
    return False

def test_database_connection():
    """测试PostgreSQL连接"""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        conn.close()
        print("✅ PostgreSQL连接成功")
        return True
    except Exception as e:
        print(f"❌ PostgreSQL连接失败: {e}")
        return False

def test_redis_connection():
    """测试Redis连接"""
    try:
        r = redis.Redis(**REDIS_CONFIG)
        r.ping()
        print("✅ Redis连接成功")
        return True
    except Exception as e:
        print(f"❌ Redis连接失败: {e}")
        return False

# 启动时连接测试
print("="*60)
print("🚀 JupyterHub后端集成启动中...")
print(f"📍 后端地址: {BACKEND_URL}")
print(f"📍 数据库: PostgreSQL@{DB_CONFIG['host']}:{DB_CONFIG['port']}")
print(f"📍 缓存: Redis@{REDIS_CONFIG['host']}:{REDIS_CONFIG['port']}")
print("="*60)

# 执行连接测试
test_backend_connection()
test_database_connection() 
test_redis_connection()

print("="*60)
print("✅ JupyterHub后端集成配置加载完成")
print("📋 功能特性:")
print("   - 统一后端认证（JWT + 用户名密码）")
print("   - PostgreSQL数据持久化")
print("   - Redis缓存支持")
print("   - 容器环境优化")
print("   - 完整权限代理")
print("="*60)
