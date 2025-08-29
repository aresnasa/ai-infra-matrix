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
from jupyterhub.utils import url_path_join
from urllib.parse import quote
from typing import cast

# 尝试导入KubeSpawner支持
try:
    from kubespawner import KubeSpawner
    KUBESPAWNER_AVAILABLE = True
    print("✅ KubeSpawner 可用")
except ImportError:
    KUBESPAWNER_AVAILABLE = False
    print("⚠️  KubeSpawner 不可用，仅支持DockerSpawner")

# 兼容编辑器/静态检查与运行时：优先使用 JupyterHub 提供的 get_config，否则退化为本地 Config
try:  # pragma: no cover - 编辑器环境
    c = get_config()  # type: ignore[name-defined]
except Exception:  # 运行在非Hub环境时兜底
    from traitlets.config import Config
    c = Config()

# 配置日志
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

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
    # 启用自动登录，/hub/login 将重定向到 login_url（/auto-login）
    auto_login = Bool(True, config=True, help="启用SSO自动登录")

    def login_url(self, base_url):
        """
        重写登录URL，始终指向自定义的自动登录处理器。
        该处理器将负责处理SSO流程，而不是JupyterHub的默认登录页。
        """
        # 注意：extra_handlers 中注册的是 '/auto-login'，不是 '/hub/auto-login'
        # 因此这里应当返回 base_url + '/auto-login'，避免 404
        return url_path_join(base_url, 'auto-login')
    
    async def authenticate(self, handler, data):  # type: ignore[override]
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
        """提取JWT Token - 优先级：Authorization header > Cookie > URL参数"""
        # 1. 从Authorization header (最高优先级)
        auth_header = handler.request.headers.get('Authorization', '')
        if auth_header.startswith('Bearer '):
            logger.info("从Authorization header提取token")
            return auth_header[7:]
        
        # 2. 从多种Cookie名称尝试获取 (支持不同的cookie名称)
        cookie_names = ['ai_infra_token', 'jwt_token', 'auth_token']
        for cookie_name in cookie_names:
            token = handler.get_cookie(cookie_name)
            if token:
                logger.info(f"从Cookie '{cookie_name}' 提取token")
                return token
        
        # 3. 从URL参数 (用于nginx认证桥接传递)
        auth_token = handler.get_argument('auth_token', None)
        if auth_token:
            logger.info("从URL参数'auth_token'提取token")
            return auth_token
        
        # 4. 从URL参数 'token' (备用)
        token = handler.get_argument('token', None)
        if token:
            logger.info("从URL参数提取token")
            return token
        
        return None
    
    async def _verify_jwt_token(self, token):
        """通过后端验证JWT Token"""
        try:
            headers = {'Authorization': f'Bearer {token}'}
            async with aiohttp.ClientSession() as session:
                async with session.get(f"{self.backend_url}/api/auth/verify", headers=headers) as resp:
                    if resp.status == 200:
                        # 后端返回200即表示有效；优先取username字段
                        result = await resp.json()
                        return result.get('username') or result.get('user', {}).get('username')
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
                # 使用已认证用户的profile接口，避免按用户名查询不存在的路由
                async with session.get(f"{self.backend_url}/api/users/profile", headers=headers) as resp:
                    if resp.status == 200:
                        user_info = await resp.json()
                        
                        # 返回用户名，JupyterHub会创建用户对象
                        # 用户权限信息通过auth_state传递
                        return {
                            'name': user_info.get('username', username),
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
        await self._proxy_to_backend()
    
    async def post(self):
        """处理POST请求"""
        await self._proxy_to_backend()
    
    # 自动登录逻辑由 AutoLoginHandler 负责
    
    async def _proxy_to_backend(self):
        """代理请求到后端"""
        try:
            # 这里可以实现请求代理逻辑
            self.write({'status': 'proxy', 'message': '后端代理功能'})
        except Exception as e:
            logger.error(f"代理请求失败: {e}")
            self.write({'status': 'error', 'message': str(e)})


class ContainerSpawner(DockerSpawner):
    """容器环境优化的Spawner
    使用 Jupyter Docker Stacks 的默认用户/环境（jovyan），避免破坏启动脚本。
    如需自定义用户，请使用 Dockerfile 定制镜像而非在此处强行覆盖。
    """
    pass


# =========================
# JupyterHub核心配置
# =========================

# 基础网络配置（使用 bind_url 配置监听地址）
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
# Hub API 绑定到 0.0.0.0，以便单用户容器通过docker网络访问
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'

# 通过环境变量决定是否通过代理访问
use_proxy = os.environ.get('JUPYTERHUB_USE_PROXY', 'true').lower() == 'true'
if use_proxy:
    # 代理模式：JupyterHub 通过 nginx /jupyter/ 前缀访问
    c.JupyterHub.base_url = '/jupyter'
    # 代理模式下，接收来自代理的Token
    c.JupyterHub.trust_user_provided_tokens = True
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

# Hub 对外(容器内)连接信息：让单用户容器能访问 Hub API（通过服务名 jupyterhub:8081）
hub_connect_host = os.environ.get('JUPYTERHUB_HUB_CONNECT_HOST', 'jupyterhub')
c.JupyterHub.hub_connect_url = f'http://{hub_connect_host}:8081'

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
# 启用单点登录（SSO）：将 /hub/login 重定向到 authenticator.login_url，即 /auto-login
_auto_login_env = os.environ.get('JUPYTERHUB_AUTO_LOGIN', 'true').lower() == 'true'
c.BackendIntegratedAuthenticator.auto_login = _auto_login_env

class AutoLoginHandler(BaseHandler):
    """自动登录处理器：验证JWT并登录用户"""

    async def get(self):
        next_url = self.get_argument('next', url_path_join(self.base_url, 'hub/'))
        try:
            auth: BackendIntegratedAuthenticator = self.authenticator  # type: ignore

            # 提取token（与认证器一致的策略）
            token = (
                self.get_cookie('ai_infra_token')
                or self.get_cookie('jwt_token')
                or self.get_cookie('auth_token')
            )
            if not token:
                auth_header = self.request.headers.get('Authorization', '')
                if auth_header.startswith('Bearer '):
                    token = auth_header[7:]
            if not token:
                # 作为兜底，支持从URL参数获取 token
                token = self.get_argument('token', None)

            if not token:
                logger.warning("AutoLogin: 未找到token，跳转到前端桥接页")
                bridge = f"/jupyterhub-auth-bridge?target_url={quote(next_url)}&from=auto_login_missing"
                self.redirect(bridge)
                return

            username = await auth._verify_jwt_token(token)
            if not username:
                logger.warning("AutoLogin: token无效，跳转到前端桥接页")
                bridge = f"/jupyterhub-auth-bridge?target_url={quote(next_url)}&from=auto_login_invalid"
                self.redirect(bridge)
                return

            user_info = await auth._get_user_info(username, token)

            # 标准化login_user参数
            if isinstance(user_info, str):
                login_data = {'name': user_info}
            else:
                login_data = user_info

            # 登录并设置Hub会话
            await self.login_user(login_data)
            logger.info(f"AutoLogin: 登录成功: {login_data.get('name', username)}")
            self.redirect(next_url)
        except Exception as e:
            logger.error(f"AutoLogin: 处理失败: {e}")
            bridge = f"/jupyterhub-auth-bridge?target_url={quote(next_url)}&from=auto_login_error"
            self.redirect(bridge)

# 额外处理器
c.JupyterHub.extra_handlers = [
    (r'/backend/(.*)', BackendProxyHandler),
    (r'/auto-login', AutoLoginHandler),
]

# =========================
# 动态Spawner配置
# =========================

# Spawner类型选择：支持docker/kubernetes两种模式
SPAWNER_TYPE = os.environ.get('JUPYTERHUB_SPAWNER', 'docker').lower()

print(f"🔧 Spawner配置模式: {SPAWNER_TYPE}")

if SPAWNER_TYPE == 'kubernetes' and KUBESPAWNER_AVAILABLE:
    print("🚀 使用KubeSpawner进行Kubernetes部署")
    
    # 加载KubeSpawner配置
    try:
        exec(open('/srv/jupyterhub/kubernetes_spawner_config.py').read())
        # 配置KubeSpawner
        configure_kubespawner(c)
        print("✅ KubeSpawner配置加载完成")
    except Exception as e:
        print(f"❌ KubeSpawner配置加载失败: {e}")
        print("🔄 回退到DockerSpawner模式")
        SPAWNER_TYPE = 'docker'

if SPAWNER_TYPE == 'docker':
    print("🐳 使用DockerSpawner进行容器部署")
    
    # Spawner配置
    c.JupyterHub.spawner_class = ContainerSpawner
    
    # Docker Spawner配置
    # 配置Docker Spawner网络
    c.ContainerSpawner.image = os.environ.get('JUPYTERHUB_IMAGE', 'jupyter/base-notebook:latest')
    c.ContainerSpawner.network_name = os.environ.get('JUPYTERHUB_NETWORK', 'ai-infra-network')
    c.ContainerSpawner.remove = True  # 删除停止的容器
    c.ContainerSpawner.debug = True
    
    # DockerSpawner 在同一 docker 网络内访问，使用容器内网IP可避免端口映射问题
    c.DockerSpawner.use_internal_ip = True
    
    # 启动/就绪超时调大，避免首次拉取镜像或慢启动导致超时
    c.Spawner.start_timeout = int(os.environ.get('JUPYTERHUB_START_TIMEOUT', '180'))
    c.Spawner.http_timeout = int(os.environ.get('JUPYTERHUB_HTTP_TIMEOUT', '120'))
    
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
    
    print("✅ DockerSpawner配置完成")

# =========================
# 通用Spawner配置
# =========================

# 安全配置（将cookie密钥保存在数据卷中，避免每次重启失效）
c.JupyterHub.cookie_secret_file = '/srv/data/jupyterhub/jupyterhub_cookie_secret'
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'default-token-change-me')

# 会话与Cookie设置：默认会话时长由 SESSION_TIMEOUT 环境变量控制（秒），默认 1 天
_session_timeout = int(os.environ.get('SESSION_TIMEOUT', '86400'))
c.JupyterHub.cookie_max_age_days = max(1, _session_timeout // 86400)
# 刷新认证，降低重复登录概率；在spawn前强制刷新
c.Authenticator.auth_refresh_age = _session_timeout
c.Authenticator.refresh_pre_spawn = True

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
