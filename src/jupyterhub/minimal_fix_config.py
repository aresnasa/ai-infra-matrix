#!/usr/bin/env python3
"""
JupyterHub Minimal Fix Configuration
最精简的修复无限重定向问题的配置
"""
import os
from ai_infra_auth import AIInfraMatrixAuthenticator

# 获取JupyterHub配置对象
c = get_config()

print("✅ LOADING MINIMAL FIX CONFIGURATION - STOPPING INFINITE REDIRECTS")

# ===== 核心反重定向设置 =====
c.JupyterHub.default_url = '/jupyter/hub/login'
c.JupyterHub.redirect_to_server = False
c.JupyterHub.auto_login = False

# ===== XSRF设置 =====  
# 使用默认cookie路径避免路径冲突
c.JupyterHub.xsrf_cookie_kwargs = {
    'max_age': 3600,
    'secure': False,
    'httponly': True
}

# ===== 自定义模板配置 =====
c.JupyterHub.template_paths = ['/srv/jupyterhub/templates']

# ===== 认证配置 =====
c.JupyterHub.authenticator_class = AIInfraMatrixAuthenticator

# 后端API配置
c.AIInfraMatrixAuthenticator.backend_api_url = os.environ.get('AI_INFRA_BACKEND_URL', 'http://backend:8082')
c.AIInfraMatrixAuthenticator.backend_api_token = os.environ.get('AI_INFRA_API_TOKEN', '')

# ===== 加密密钥配置 =====
# JupyterHub需要一个随机密钥用于加密cookie和其他敏感数据
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/data/jupyterhub_cookie_secret'

# ===== 服务器配置 =====
c.JupyterHub.bind_url = 'http://0.0.0.0:8000/jupyter/'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081/jupyter/'
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.hub_ip = '0.0.0.0' 
c.JupyterHub.hub_port = 8081

# ===== 用户管理 =====
c.JupyterHub.admin_access = True
c.Authenticator.admin_users = {'admin', 'testuser'}
c.Authenticator.allow_all = True

# ===== 调试和日志 =====
c.JupyterHub.log_level = 'INFO'
c.Application.log_level = 'INFO'

print("🔧 Configuration loaded successfully!")
print(f"📍 Backend API URL: {c.AIInfraMatrixAuthenticator.backend_api_url}")
print(f"🔐 Authenticator: {c.JupyterHub.authenticator_class}")
print(f"📂 Template paths: {c.JupyterHub.template_paths}")
