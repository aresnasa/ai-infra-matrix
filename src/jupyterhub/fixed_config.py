# AI-Infra-Matrix JupyterHub 配置 - 修正版
# 解决Docker Spawner连接问题

import os
from pathlib import Path

print("=== JupyterHub配置加载中 - AI基础设施矩阵 ===")

# 基本服务器配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8091'

# 数据目录
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)

# 数据库配置
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'
c.JupyterHub.cookie_secret_file = str(project_data_dir / "cookie_secret")

# 代理配置
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'ai-infra-proxy-token')

# 认证配置
USE_CUSTOM_AUTH = os.environ.get('USE_CUSTOM_AUTH', 'false').lower() == 'true'

if USE_CUSTOM_AUTH:
    print("使用自定义 AI-Infra-Matrix 认证器")
    # 添加Python路径
    import sys
    sys.path.insert(0, '/srv/jupyterhub')
    
    try:
        from ai_infra_auth import AIInfraMatrixAuthenticator
        
        # AI-Infra-Matrix 后端集成
        backend_url = os.environ.get('AI_INFRA_BACKEND_URL', 'http://backend:8082')
        api_token = os.environ.get('AI_INFRA_API_TOKEN', 'ai-infra-hub-token')
        
        # 配置自定义认证器
        c.JupyterHub.authenticator_class = AIInfraMatrixAuthenticator
        c.AIInfraMatrixAuthenticator.backend_api_url = backend_url
        c.AIInfraMatrixAuthenticator.backend_api_token = api_token
        c.AIInfraMatrixAuthenticator.enable_auth_state = True
        c.AIInfraMatrixAuthenticator.auto_login = os.environ.get('JUPYTERHUB_AUTO_LOGIN', 'true').lower() == 'true'
        c.AIInfraMatrixAuthenticator.allow_token_in_url = True
        print(f"自定义认证器配置完成，后端URL: {backend_url}")
    except ImportError as e:
        print(f"自定义认证器导入失败，使用默认认证: {e}")
        USE_CUSTOM_AUTH = False

if not USE_CUSTOM_AUTH:
    print("使用Dummy认证器（开发/测试模式）")
    c.JupyterHub.authenticator_class = 'jupyterhub.auth.DummyAuthenticator'
    c.DummyAuthenticator.password = 'test123'

# 管理员用户
admin_users = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin,jupyter-admin').split(',')
c.Authenticator.admin_users = set(admin_users)

# Spawner配置 - 解决Docker连接问题
DOCKER_AVAILABLE = os.path.exists('/var/run/docker.sock') or os.environ.get('DOCKER_HOST')

if DOCKER_AVAILABLE and os.environ.get('USE_DOCKER_SPAWNER', 'false').lower() == 'true':
    print("使用Docker Spawner")
    try:
        c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
        c.DockerSpawner.image = 'jupyter/scipy-notebook:latest'
        c.DockerSpawner.remove = True
        c.DockerSpawner.network_name = 'ansible-network'
        # Docker套接字配置
        if os.path.exists('/var/run/docker.sock'):
            c.DockerSpawner.docker_kwargs = {'base_url': 'unix://var/run/docker.sock'}
        print("Docker Spawner 配置成功")
    except Exception as e:
        print(f"Docker Spawner 配置失败，切换到本地进程: {e}")
        DOCKER_AVAILABLE = False

if not DOCKER_AVAILABLE:
    print("使用本地进程 Spawner")
    c.JupyterHub.spawner_class = 'jupyterhub.spawner.LocalProcessSpawner'
    
    # 本地进程配置
    c.LocalProcessSpawner.create_system_users = True
    c.LocalProcessSpawner.start_timeout = 60
    c.LocalProcessSpawner.http_timeout = 60
    
    # 为本地进程spawner设置notebook启动命令
    c.Spawner.cmd = ['jupyter-labhub']
    c.Spawner.default_url = '/lab'

# 服务器配置
c.JupyterHub.tornado_settings = {
    'slow_spawn_timeout': 60,
    'slow_stop_timeout': 60,
}

# 日志配置
c.JupyterHub.log_level = 'INFO'

# 安全配置
c.JupyterHub.allow_named_servers = False
c.JupyterHub.concurrent_spawn_limit = 5

print("=== JupyterHub配置加载完成 ===")
print(f"项目根目录: /srv")
print(f"数据目录: {project_data_dir}")
print(f"绑定地址: http://0.0.0.0:8000")
print(f"自定义认证: {USE_CUSTOM_AUTH}")
print(f"管理员用户: {admin_users}")
print(f"Docker可用: {DOCKER_AVAILABLE}")
print(f"Spawner类型: {'Docker' if DOCKER_AVAILABLE else 'LocalProcess'}")
