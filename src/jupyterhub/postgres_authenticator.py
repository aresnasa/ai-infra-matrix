# PostgreSQL + Redis 认证器
# 与后端系统统一的用户认证体系

import os
import hashlib
import jwt
import redis
import psycopg2
import psycopg2.extras
from datetime import datetime, timedelta
from traitlets import Unicode, Int, Bool
from jupyterhub.auth import Authenticator
from jupyterhub.handlers import BaseHandler
from tornado import gen, web
import json
import bcrypt

class PostgreSQLRedisAuthenticator(Authenticator):
    """
    PostgreSQL数据库认证器，使用Redis缓存会话
    与后端系统共享用户数据库和认证逻辑
    """
    
    # PostgreSQL配置
    db_host = Unicode('localhost', config=True, help="PostgreSQL host")
    db_port = Int(5432, config=True, help="PostgreSQL port")
    db_name = Unicode('ansible_playbook_generator', config=True, help="PostgreSQL database name")
    db_user = Unicode('postgres', config=True, help="PostgreSQL user")
    db_password = Unicode('postgres', config=True, help="PostgreSQL password")
    
    # Redis配置
    redis_host = Unicode('localhost', config=True, help="Redis host")
    redis_port = Int(6379, config=True, help="Redis port")
    redis_password = Unicode('', config=True, help="Redis password")
    redis_db = Int(0, config=True, help="Redis database number")
    
    # JWT配置
    jwt_secret = Unicode('your-secret-key-change-in-production', config=True, help="JWT secret key")
    
    # 会话缓存配置
    session_timeout = Int(3600 * 24, config=True, help="Session timeout in seconds (default: 24 hours)")
    
    # 自动创建本地用户
    create_system_users = Bool(False, config=True, help="Create system users automatically")

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self._db_pool = None
        self._redis_client = None
        
    def _get_db_connection(self):
        """获取数据库连接"""
        try:
            conn = psycopg2.connect(
                host=self.db_host,
                port=self.db_port,
                database=self.db_name,
                user=self.db_user,
                password=self.db_password,
                cursor_factory=psycopg2.extras.RealDictCursor
            )
            return conn
        except Exception as e:
            self.log.error(f"Database connection error: {e}")
            return None
    
    def _get_redis_client(self):
        """获取Redis客户端"""
        if self._redis_client is None:
            try:
                self._redis_client = redis.Redis(
                    host=self.redis_host,
                    port=self.redis_port,
                    password=self.redis_password if self.redis_password else None,
                    db=self.redis_db,
                    decode_responses=True
                )
                # 测试连接
                self._redis_client.ping()
                self.log.info("Redis connection established")
            except Exception as e:
                self.log.error(f"Redis connection error: {e}")
                self._redis_client = None
        return self._redis_client
    
    def _verify_password(self, stored_password, provided_password):
        """验证密码（支持bcrypt）"""
        try:
            # 如果是bcrypt哈希
            if stored_password.startswith('$2a$') or stored_password.startswith('$2b$') or stored_password.startswith('$2y$'):
                return bcrypt.checkpw(provided_password.encode('utf-8'), stored_password.encode('utf-8'))
            else:
                # 如果是普通字符串密码（兼容现有系统）
                return stored_password == provided_password
        except Exception as e:
            self.log.error(f"Password verification error: {e}")
            return False
    
    def _get_user_from_db(self, username):
        """从数据库获取用户信息"""
        conn = self._get_db_connection()
        if not conn:
            return None
            
        try:
            cursor = conn.cursor()
            query = """
            SELECT id, username, email, password_hash, is_admin, role, 
                   created_at, updated_at, is_active, last_login_at
            FROM users 
            WHERE username = %s AND is_active = TRUE
            """
            cursor.execute(query, (username,))
            result = cursor.fetchone()
            
            if result:
                return {
                    'id': result['id'],
                    'username': result['username'],
                    'email': result['email'],
                    'password_hash': result['password_hash'],
                    'is_admin': result['is_admin'],
                    'role': result['role'],
                    'created_at': result['created_at'],
                    'updated_at': result['updated_at'],
                    'is_active': result['is_active'],
                    'last_login_at': result['last_login_at']
                }
            return None
            
        except Exception as e:
            self.log.error(f"Database query error: {e}")
            return None
        finally:
            if conn:
                conn.close()
    
    def _create_session_cache(self, user_info, token):
        """在Redis中创建会话缓存"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return False
            
        try:
            session_data = {
                'user_id': user_info['id'],
                'username': user_info['username'],
                'email': user_info['email'],
                'role': user_info['role'],
                'is_admin': user_info['is_admin'],
                'login_time': datetime.utcnow().isoformat(),
                'last_activity': datetime.utcnow().isoformat(),
                'source': 'jupyterhub',
                'active': True
            }
            
            # 使用多个键存储会话信息
            session_key = f"session:{user_info['username']}"
            token_key = f"token:{token}"
            user_session_key = f"user_session:{user_info['id']}"
            
            # 将会话信息存储到Redis
            redis_client.setex(session_key, self.session_timeout, json.dumps(session_data))
            redis_client.setex(token_key, self.session_timeout, json.dumps(session_data))
            redis_client.setex(user_session_key, self.session_timeout, token)
            
            self.log.info(f"Session created for user {user_info['username']}")
            return True
            
        except Exception as e:
            self.log.error(f"Redis session creation error: {e}")
            return False
    
    def _validate_session_token(self, token):
        """验证会话token"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return None
            
        try:
            token_key = f"token:{token}"
            session_data = redis_client.get(token_key)
            
            if session_data:
                return json.loads(session_data)
            return None
            
        except Exception as e:
            self.log.error(f"Session validation error: {e}")
            return None
    
    def _validate_jwt_token(self, token):
        """验证JWT token"""
        try:
            payload = jwt.decode(token, self.jwt_secret, algorithms=['HS256'])
            return payload
        except jwt.ExpiredSignatureError:
            self.log.warning("JWT token expired")
            return None
        except jwt.InvalidTokenError:
            self.log.warning("Invalid JWT token")
            return None
        except Exception as e:
            self.log.error(f"JWT validation error: {e}")
            return None
    
    @gen.coroutine
    def authenticate(self, handler, data):
        """
        认证方法
        1. 支持传统用户名密码认证
        2. 支持JWT token认证（从后端跳转）
        3. 支持Redis会话验证
        """
        username = data.get('username', '').strip()
        password = data.get('password', '').strip()
        token = data.get('token', '').strip()
        
        # 从请求中获取token（如果没有在data中）
        if not token:
            auth_header = handler.request.headers.get('Authorization', '')
            if auth_header.startswith('Bearer '):
                token = auth_header[7:]
            
        # 方式1: JWT Token认证（从后端系统跳转）
        if token:
            self.log.info(f"Attempting JWT token authentication")
            
            # 首先尝试验证Redis会话
            session_data = self._validate_session_token(token)
            if session_data:
                self.log.info(f"User {session_data['username']} authenticated via Redis session")
                return {
                    'name': session_data['username'],
                    'admin': session_data['is_admin'],
                    'auth_model': session_data
                }
            
            # 如果Redis会话无效，尝试JWT验证
            jwt_payload = self._validate_jwt_token(token)
            if jwt_payload:
                username = jwt_payload.get('username')
                user_info = self._get_user_from_db(username)
                if user_info:
                    # 创建新的会话缓存
                    self._create_session_cache(user_info, token)
                    self.log.info(f"User {username} authenticated via JWT token")
                    return {
                        'name': user_info['username'],
                        'admin': user_info['is_admin'],
                        'auth_model': user_info
                    }
        
        # 方式2: 传统用户名密码认证
        if username and password:
            self.log.info(f"Attempting username/password authentication for {username}")
            
            user_info = self._get_user_from_db(username)
            if user_info and self._verify_password(user_info['password_hash'], password):
                # 生成新的JWT token
                token_payload = {
                    'user_id': user_info['id'],
                    'username': user_info['username'],
                    'email': user_info['email'],
                    'role': user_info['role'],
                    'is_admin': user_info['is_admin'],
                    'iat': datetime.utcnow().timestamp(),
                    'exp': (datetime.utcnow() + timedelta(seconds=self.session_timeout)).timestamp()
                }
                
                new_token = jwt.encode(token_payload, self.jwt_secret, algorithm='HS256')
                
                # 创建会话缓存
                self._create_session_cache(user_info, new_token)
                
                self.log.info(f"User {username} authenticated via username/password")
                return {
                    'name': user_info['username'],
                    'admin': user_info['is_admin'],
                    'auth_model': user_info
                }
        
        # 认证失败
        self.log.warning(f"Authentication failed for {username}")
        return None
    
    def get_handlers(self, app):
        """添加自定义处理器"""
        return [
            (r'/login', UnifiedLoginHandler),
            (r'/logout', UnifiedLogoutHandler),
        ]


class UnifiedLoginHandler(BaseHandler):
    """统一登录处理器 - 处理从后端系统的跳转"""
    
    @web.authenticated
    def get(self):
        """处理登录GET请求"""
        token = self.get_argument('token', None)
        next_url = self.get_argument('next', '/hub/home')
        
        if token:
            # 设置token到cookie，供后续认证使用
            self.set_secure_cookie('jupyterhub-token', token)
            
        # 重定向到指定页面
        self.redirect(next_url)
    
    async def post(self):
        """处理登录POST请求"""
        username = self.get_argument('username', '')
        password = self.get_argument('password', '')
        token = self.get_argument('token', '')
        
        if not any([username and password, token]):
            raise web.HTTPError(400, "Missing credentials")
        
        auth_data = {}
        if token:
            auth_data['token'] = token
        else:
            auth_data['username'] = username
            auth_data['password'] = password
        
        # 使用认证器进行认证
        user_info = await self.authenticator.authenticate(self, auth_data)
        
        if user_info:
            self.set_login_cookie(user_info['name'])
            self.redirect('/hub/home')
        else:
            raise web.HTTPError(401, "Authentication failed")


class UnifiedLogoutHandler(BaseHandler):
    """统一登出处理器"""
    
    def get(self):
        """处理登出请求"""
        user = self.current_user
        
        if user:
            # 清理Redis会话
            authenticator = self.authenticator
            redis_client = authenticator._get_redis_client()
            
            if redis_client:
                try:
                    session_key = f"session:{user.name}"
                    user_session_key = f"user_session:{user.id}"
                    
                    redis_client.delete(session_key)
                    redis_client.delete(user_session_key)
                    
                    self.log.info(f"Cleared session for user {user.name}")
                except Exception as e:
                    self.log.error(f"Error clearing session: {e}")
        
        # 清除登录cookie
        self.clear_login_cookie()
        self.clear_cookie('jupyterhub-token')
        
        # 重定向到主项目登录页面
        backend_url = os.getenv('AI_INFRA_BACKEND_URL', 'http://localhost:3001')
        logout_url = f"{backend_url}/login?logout=true&source=jupyterhub"
        self.redirect(logout_url)
    
    def _get_user_from_db(self, username):
        """从数据库获取用户信息"""
        conn = self._get_db_connection()
        if not conn:
            return None
            
        try:
            cursor = conn.cursor()
            query = """
            SELECT id, username, email, password_hash, is_admin, role, 
                   created_at, updated_at, is_active, last_login_at
            FROM users 
            WHERE username = %s AND is_active = TRUE
            """
            cursor.execute(query, (username,))
            result = cursor.fetchone()
            
            if result:
                return {
                    'id': result['id'],
                    'username': result['username'],
                    'email': result['email'],
                    'password_hash': result['password_hash'],
                    'is_admin': result['is_admin'],
                    'role': result['role'],
                    'created_at': result['created_at'],
                    'updated_at': result['updated_at'],
                    'is_active': result['is_active'],
                    'last_login_at': result['last_login_at']
                }
            return None
            
        except Exception as e:
            self.log.error(f"Database query error: {e}")
            return None
        finally:
            if conn:
                conn.close()
    
    def _create_session_cache(self, user_info, token):
        """在Redis中创建会话缓存"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return False
            
        try:
            session_data = {
                'user_id': user_info['id'],
                'username': user_info['username'],
                'email': user_info['email'],
                'role': user_info['role'],
                'is_admin': user_info['is_admin'],
                'login_time': datetime.utcnow().isoformat(),
                'last_activity': datetime.utcnow().isoformat(),
                'source': 'jupyterhub',
                'active': True
            }
            
            # 使用多个键存储会话信息
            session_key = f"session:{user_info['username']}"
            token_key = f"token:{token}"
            user_session_key = f"user_session:{user_info['id']}"
            
            # 将会话信息存储到Redis
            redis_client.setex(session_key, self.session_timeout, json.dumps(session_data))
            redis_client.setex(token_key, self.session_timeout, json.dumps(session_data))
            redis_client.setex(user_session_key, self.session_timeout, token)
            
            self.log.info(f"Session created for user {user_info['username']}")
            return True
            
        except Exception as e:
            self.log.error(f"Redis session creation error: {e}")
            return False
    
    def _validate_session_token(self, token):
        """验证会话token"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return None
            
        try:
            token_key = f"token:{token}"
            session_data = redis_client.get(token_key)
            
            if session_data:
                return json.loads(session_data)
            return None
            
        except Exception as e:
            self.log.error(f"Session validation error: {e}")
            return None
        conn = self._get_db_connection()
        if not conn:
            return None
            
        try:
            with conn.cursor() as cursor:
                # 查询用户信息，包括角色
                cursor.execute("""
                    SELECT u.*, 
                           COALESCE(json_agg(DISTINCT r.name) FILTER (WHERE r.name IS NOT NULL), '[]') as roles,
                           COALESCE(json_agg(DISTINCT ug.name) FILTER (WHERE ug.name IS NOT NULL), '[]') as user_groups
                    FROM users u
                    LEFT JOIN user_roles ur ON u.id = ur.user_id
                    LEFT JOIN roles r ON ur.role_id = r.id
                    LEFT JOIN user_group_memberships ugm ON u.id = ugm.user_id
                    LEFT JOIN user_groups ug ON ugm.user_group_id = ug.id
                    WHERE u.username = %s AND u.is_active = true AND u.deleted_at IS NULL
                    GROUP BY u.id
                """, (username,))
                
                user = cursor.fetchone()
                if user:
                    # 转换为字典
                    user_dict = dict(user)
                    # 解析JSON字段
                    if isinstance(user_dict['roles'], str):
                        user_dict['roles'] = json.loads(user_dict['roles'])
                    if isinstance(user_dict['user_groups'], str):
                        user_dict['user_groups'] = json.loads(user_dict['user_groups'])
                    return user_dict
                return None
                
        except Exception as e:
            self.log.error(f"Database query error: {e}")
            return None
        finally:
            conn.close()
    
    def _cache_user_session(self, username, user_data):
        """缓存用户会话到Redis"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return False
            
        try:
            session_key = f"jupyterhub:session:{username}"
            session_data = {
                'username': username,
                'user_id': user_data['id'],
                'email': user_data['email'],
                'roles': user_data.get('roles', []),
                'user_groups': user_data.get('user_groups', []),
                'is_admin': 'admin' in user_data.get('roles', []) or 'super-admin' in user_data.get('roles', []),
                'last_activity': datetime.utcnow().isoformat()
            }
            
            redis_client.setex(
                session_key,
                self.session_timeout,
                json.dumps(session_data)
            )
            
            self.log.info(f"Cached session for user: {username}")
            return True
            
        except Exception as e:
            self.log.error(f"Redis cache error: {e}")
            return False
    
    def _get_cached_session(self, username):
        """从Redis获取缓存的会话"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return None
            
        try:
            session_key = f"jupyterhub:session:{username}"
            session_data = redis_client.get(session_key)
            
            if session_data:
                return json.loads(session_data)
            return None
            
        except Exception as e:
            self.log.error(f"Redis get session error: {e}")
            return None
    
    def _update_user_activity(self, username):
        """更新用户活动时间"""
        redis_client = self._get_redis_client()
        if not redis_client:
            return
            
        try:
            session_key = f"jupyterhub:session:{username}"
            session_data = redis_client.get(session_key)
            
            if session_data:
                data = json.loads(session_data)
                data['last_activity'] = datetime.utcnow().isoformat()
                redis_client.setex(
                    session_key,
                    self.session_timeout,
                    json.dumps(data)
                )
                
        except Exception as e:
            self.log.error(f"Update activity error: {e}")

    @gen.coroutine
    def authenticate(self, handler, data):
        """认证用户"""
        username = data.get('username', '').strip()
        password = data.get('password', '').strip()
        
        if not username or not password:
            self.log.warning("Empty username or password")
            return None
        
        # 首先检查缓存
        cached_session = self._get_cached_session(username)
        if cached_session:
            self.log.info(f"Found cached session for user: {username}")
            # 验证密码（仍需要验证，因为缓存可能被攻击）
            user_data = self._get_user_from_db(username)
            if user_data and self._verify_password(user_data['password'], password):
                self._update_user_activity(username)
                return {
                    'name': username,
                    'admin': cached_session.get('is_admin', False),
                    'auth_model': {
                        'user_id': cached_session['user_id'],
                        'email': cached_session['email'],
                        'roles': cached_session['roles'],
                        'user_groups': cached_session['user_groups']
                    }
                }
        
        # 从数据库验证用户
        user_data = self._get_user_from_db(username)
        if not user_data:
            self.log.warning(f"User not found: {username}")
            return None
        
        # 验证密码
        if not self._verify_password(user_data['password'], password):
            self.log.warning(f"Invalid password for user: {username}")
            return None
        
        # 更新最后登录时间
        self._update_last_login(user_data['id'])
        
        # 缓存会话
        self._cache_user_session(username, user_data)
        
        # 检查是否为管理员
        is_admin = 'admin' in user_data.get('roles', []) or 'super-admin' in user_data.get('roles', [])
        
        self.log.info(f"Authenticated user: {username}, admin: {is_admin}")
        
        return {
            'name': username,
            'admin': is_admin,
            'auth_model': {
                'user_id': user_data['id'],
                'email': user_data['email'],
                'roles': user_data.get('roles', []),
                'user_groups': user_data.get('user_groups', [])
            }
        }
    
    def _update_last_login(self, user_id):
        """更新用户最后登录时间"""
        conn = self._get_db_connection()
        if not conn:
            return
            
        try:
            with conn.cursor() as cursor:
                cursor.execute(
                    "UPDATE users SET last_login = %s WHERE id = %s",
                    (datetime.utcnow(), user_id)
                )
                conn.commit()
        except Exception as e:
            self.log.error(f"Update last login error: {e}")
        finally:
            conn.close()

    @gen.coroutine
    def pre_spawn_start(self, user, spawner):
        """在启动spawner之前调用"""
        # 检查会话是否仍然有效
        cached_session = self._get_cached_session(user.name)
        if cached_session:
            self._update_user_activity(user.name)
            
            # 设置环境变量，传递给用户容器
            spawner.environment.update({
                'JUPYTERHUB_USER_ID': str(cached_session['user_id']),
                'JUPYTERHUB_USER_EMAIL': cached_session['email'],
                'JUPYTERHUB_USER_ROLES': ','.join(cached_session['roles']),
                'JUPYTERHUB_USER_GROUPS': ','.join(cached_session['user_groups'])
            })

    def get_handlers(self, app):
        """返回自定义处理器"""
        return [
            (r'/hub/api/user-session', UserSessionHandler),
            (r'/hub/api/logout-all', LogoutAllHandler),
        ]

class UserSessionHandler(BaseHandler):
    """用户会话信息处理器"""
    
    @web.authenticated
    def get(self):
        """获取当前用户会话信息"""
        user = self.current_user
        authenticator = self.authenticator
        
        if hasattr(authenticator, '_get_cached_session'):
            session_data = authenticator._get_cached_session(user.name)
            if session_data:
                self.write(session_data)
            else:
                self.set_status(404)
                self.write({'error': 'Session not found'})
        else:
            self.set_status(500)
            self.write({'error': 'Authenticator not supported'})

class LogoutAllHandler(BaseHandler):
    """登出所有会话处理器"""
    
    @web.authenticated
    def post(self):
        """清除Redis中的用户会话"""
        user = self.current_user
        authenticator = self.authenticator
        
        if hasattr(authenticator, '_get_redis_client'):
            redis_client = authenticator._get_redis_client()
            if redis_client:
                try:
                    session_key = f"jupyterhub:session:{user.name}"
                    redis_client.delete(session_key)
                    self.write({'message': 'All sessions cleared'})
                except Exception as e:
                    self.set_status(500)
                    self.write({'error': str(e)})
            else:
                self.set_status(500)
                self.write({'error': 'Redis not available'})
        else:
            self.set_status(500)
            self.write({'error': 'Authenticator not supported'})
