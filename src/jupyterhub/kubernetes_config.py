"""
JupyterHub Configuration for Kubernetes Deployment
"""

import os
import sys
import asyncio
from jupyterhub.auth import Authenticator
from jupyterhub.handlers import BaseHandler
from jupyterhub.spawner import Spawner
from jupyterhub.utils import url_path_join
from kubernetes import client, config
from kubespawner import KubeSpawner
from traitlets import Bool, Unicode, Int, Dict, List
from tornado import web
import json
import jwt
import aiohttp
import logging

# Configure logging
c.JupyterHub.log_level = 'DEBUG'

# Database configuration
postgres_host = os.environ.get('POSTGRES_HOST', 'ai-infra-matrix-postgresql')
postgres_port = os.environ.get('POSTGRES_PORT', '5432')
postgres_db = os.environ.get('POSTGRES_DB', 'ai_infra_db')
postgres_user = os.environ.get('POSTGRES_USER', 'ai_infra_user')
postgres_password = os.environ.get('POSTGRES_PASSWORD', '')

c.JupyterHub.db_url = f'postgresql://{postgres_user}:{postgres_password}@{postgres_host}:{postgres_port}/{postgres_db}'

# Network configuration
c.JupyterHub.base_url = '/jupyter/'
c.JupyterHub.bind_url = 'http://0.0.0.0:8000/jupyter/'

# Hub configuration for Kubernetes
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081/jupyter/'
c.JupyterHub.hub_connect_url = f"http://{os.environ.get('JUPYTERHUB_HOST', 'ai-infra-matrix-jupyterhub')}:8081/jupyter/"

# Public URL configuration
public_host = os.environ.get('JUPYTERHUB_PUBLIC_HOST', 'localhost:8080')
c.JupyterHub.public_url = f'http://{public_host}/jupyter/'

# Session and cookie configuration
c.JupyterHub.cookie_secret_file = '/srv/data/jupyterhub/jupyterhub_cookie_secret'

# Session timeout from environment (in minutes, default 8 hours = 480 minutes)
session_timeout_minutes = int(os.environ.get('SESSION_TIMEOUT', '480'))
c.JupyterHub.cookie_max_age_days = session_timeout_minutes / (24 * 60)  # Convert to days

# Enable auto login and session refresh
c.Authenticator.auto_login = True
c.Authenticator.auth_refresh_age = session_timeout_minutes * 60  # Convert to seconds
c.Authenticator.refresh_pre_spawn = True

# Kubernetes spawner configuration
c.JupyterHub.spawner_class = KubeSpawner

# Kubernetes namespace for single-user pods
k8s_namespace = os.environ.get('KUBERNETES_NAMESPACE', 'ai-infra-users')
c.KubeSpawner.namespace = k8s_namespace

# Service account for single-user pods
k8s_service_account = os.environ.get('KUBERNETES_SERVICE_ACCOUNT', 'ai-infra-matrix-jupyterhub')
c.KubeSpawner.service_account = k8s_service_account

# Single-user image configuration
jupyterhub_image = os.environ.get('JUPYTERHUB_IMAGE', 'ai-infra-singleuser:latest')
c.KubeSpawner.image = jupyterhub_image

# Resource limits
mem_limit = os.environ.get('JUPYTERHUB_MEM_LIMIT', '3G')
cpu_limit = float(os.environ.get('JUPYTERHUB_CPU_LIMIT', '2.0'))

c.KubeSpawner.mem_limit = mem_limit
c.KubeSpawner.cpu_limit = cpu_limit
c.KubeSpawner.mem_guarantee = '512M'
c.KubeSpawner.cpu_guarantee = 0.1

# Storage configuration
jupyterhub_storage_class = os.environ.get('JUPYTERHUB_STORAGE_CLASS', 'default')
shared_storage_class = os.environ.get('SHARED_STORAGE_CLASS', 'default')

c.KubeSpawner.storage_class = jupyterhub_storage_class
c.KubeSpawner.storage_capacity = '10Gi'

# Persistent volumes
c.KubeSpawner.volumes = [
    {
        'name': 'home',
        'persistentVolumeClaim': {
            'claimName': 'home-{username}'
        }
    },
    {
        'name': 'shared-notebooks',
        'persistentVolumeClaim': {
            'claimName': 'ai-infra-matrix-shared-notebooks'
        }
    }
]

c.KubeSpawner.volume_mounts = [
    {
        'name': 'home',
        'mountPath': '/home/jovyan'
    },
    {
        'name': 'shared-notebooks',
        'mountPath': '/home/jovyan/shared'
    }
]

# Networking
c.KubeSpawner.start_timeout = 300
c.KubeSpawner.http_timeout = 120

# Security context
c.KubeSpawner.fs_gid = 100
c.KubeSpawner.supplemental_gids = [100]

# Environment variables for single-user containers
c.KubeSpawner.environment = {
    'JUPYTER_ENABLE_LAB': 'yes',
    'GRANT_SUDO': 'yes',
    'CHOWN_HOME': 'yes',
    'CHOWN_HOME_OPTS': '-R'
}

# Command and arguments
c.KubeSpawner.cmd = ['start-singleuser.sh']

# Labels for single-user pods
c.KubeSpawner.common_labels = {
    'app.kubernetes.io/name': 'jupyterhub-singleuser',
    'app.kubernetes.io/component': 'singleuser',
    'app.kubernetes.io/managed-by': 'jupyterhub'
}

# Node selector and tolerations (optional)
# c.KubeSpawner.node_selector = {'node-type': 'compute'}
# c.KubeSpawner.tolerations = [
#     {
#         'key': 'dedicated',
#         'operator': 'Equal',
#         'value': 'user',
#         'effect': 'NoSchedule'
#     }
# ]

class BackendIntegratedAuthenticator(Authenticator):
    """
    Custom authenticator that integrates with the backend API
    and supports both JWT tokens and form-based authentication.
    """
    
    auto_login = Bool(True, config=True, help="Enable automatic login for SSO")
    
    backend_url = Unicode(
        os.environ.get('BACKEND_URL', 'http://ai-infra-matrix-backend:3000'),
        config=True,
        help="Backend API URL for authentication"
    )
    
    jwt_secret = Unicode(
        os.environ.get('JWT_SECRET', 'your-secret-key'),
        config=True,
        help="JWT secret for token verification"
    )
    
    def login_url(self, base_url):
        """
        Custom login URL that redirects to the auth bridge to break redirect loops
        """
        return url_path_join(base_url, '/jupyterhub-auth-bridge?target_url=%2Fjupyter%2Fhub%2F')
    
    async def authenticate(self, handler, data):
        """
        Authenticate user with JWT token or form data
        """
        self.log.debug("authenticate called")
        
        # Check for JWT token in Authorization header
        auth_header = handler.request.headers.get('Authorization', '')
        if auth_header.startswith('Bearer '):
            token = auth_header[7:]  # Remove 'Bearer ' prefix
            user_info = await self.verify_jwt_token(token)
            if user_info:
                self.log.info(f"Authenticated user via JWT header: {user_info['username']}")
                return user_info
        
        # Check for JWT token in cookies
        token = handler.get_cookie('jwt_token') or handler.get_cookie('access_token')
        if token:
            user_info = await self.verify_jwt_token(token)
            if user_info:
                self.log.info(f"Authenticated user via cookie: {user_info['username']}")
                return user_info
        
        # Check for JWT token in URL parameters
        token = handler.get_argument('token', None)
        if token:
            user_info = await self.verify_jwt_token(token)
            if user_info:
                self.log.info(f"Authenticated user via URL parameter: {user_info['username']}")
                return user_info
        
        # Form-based authentication
        if data and 'username' in data and 'password' in data:
            username = data['username']
            password = data['password']
            
            self.log.debug(f"Attempting form authentication for user: {username}")
            
            # Verify credentials with backend
            user_info = await self.verify_credentials(username, password)
            if user_info:
                self.log.info(f"Authenticated user via form: {username}")
                return user_info
            else:
                self.log.warning(f"Form authentication failed for user: {username}")
                return None
        
        self.log.debug("No valid authentication method found")
        return None
    
    async def verify_jwt_token(self, token):
        """
        Verify JWT token and extract user information
        """
        try:
            # Decode JWT token
            payload = jwt.decode(token, self.jwt_secret, algorithms=['HS256'])
            
            username = payload.get('username') or payload.get('sub')
            if not username:
                self.log.warning("JWT token missing username")
                return None
            
            # Verify token with backend
            async with aiohttp.ClientSession() as session:
                headers = {'Authorization': f'Bearer {token}'}
                async with session.get(f"{self.backend_url}/api/auth/verify", headers=headers) as response:
                    if response.status == 200:
                        user_data = await response.json()
                        return {
                            'name': username,
                            'username': username,
                            'email': user_data.get('email'),
                            'admin': user_data.get('is_admin', False)
                        }
                    else:
                        self.log.warning(f"Backend token verification failed: {response.status}")
                        return None
        
        except jwt.ExpiredSignatureError:
            self.log.warning("JWT token expired")
            return None
        except jwt.InvalidTokenError as e:
            self.log.warning(f"Invalid JWT token: {e}")
            return None
        except Exception as e:
            self.log.error(f"Error verifying JWT token: {e}")
            return None
    
    async def verify_credentials(self, username, password):
        """
        Verify username and password with backend
        """
        try:
            async with aiohttp.ClientSession() as session:
                login_data = {
                    'username': username,
                    'password': password
                }
                async with session.post(f"{self.backend_url}/api/auth/login", json=login_data) as response:
                    if response.status == 200:
                        user_data = await response.json()
                        return {
                            'name': username,
                            'username': username,
                            'email': user_data.get('email'),
                            'admin': user_data.get('is_admin', False)
                        }
                    else:
                        self.log.warning(f"Backend credential verification failed: {response.status}")
                        return None
        
        except Exception as e:
            self.log.error(f"Error verifying credentials: {e}")
            return None

class AutoLoginHandler(BaseHandler):
    """
    Handler for automatic login with token verification
    """
    
    async def get(self):
        """
        Handle GET request for auto-login
        """
        self.log.debug("AutoLoginHandler.get() called")
        
        # Extract token from various sources
        token = (
            self.get_cookie('jwt_token') or 
            self.get_cookie('access_token') or
            self.request.headers.get('Authorization', '').replace('Bearer ', '') or
            self.get_argument('token', None)
        )
        
        if not token:
            self.log.debug("No token found, redirecting to auth bridge")
            next_url = self.get_argument('next', '/jupyter/hub/')
            redirect_url = f'/jupyterhub-auth-bridge?target_url={next_url}'
            self.redirect(redirect_url)
            return
        
        # Verify token
        authenticator = self.authenticator
        user_info = await authenticator.verify_jwt_token(token)
        
        if user_info:
            # Log in the user
            username = user_info['username']
            user = self.find_user(username)
            if not user:
                user = await self.hub.add_user(username)
            
            # Set login cookie
            self.set_login_cookie(user)
            
            # Redirect to next URL
            next_url = self.get_argument('next', '/jupyter/hub/')
            self.redirect(next_url)
        else:
            self.log.warning("Token verification failed, redirecting to auth bridge")
            next_url = self.get_argument('next', '/jupyter/hub/')
            redirect_url = f'/jupyterhub-auth-bridge?target_url={next_url}'
            self.redirect(redirect_url)

# Configure the custom authenticator
c.JupyterHub.authenticator_class = BackendIntegratedAuthenticator

# Add custom handlers
c.JupyterHub.extra_handlers = [
    (r'/auto-login', AutoLoginHandler),
]

# Admin users
c.Authenticator.admin_users = {'admin'}

# Allow all users to spawn servers
c.JupyterHub.allow_named_servers = True

# Shutdown settings
c.JupyterHub.shutdown_on_logout = False
c.JupyterHub.cleanup_servers = False

# Configure proxy
c.ConfigurableHTTPProxy.should_start = True
c.ConfigurableHTTPProxy.api_url = 'http://0.0.0.0:8001'

print("JupyterHub Kubernetes configuration loaded successfully!")
