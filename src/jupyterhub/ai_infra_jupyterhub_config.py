# JupyterHub配置文件
# 用于AI基础设施矩阵项目的K8s GPU作业集成

import os
import sys
from pathlib import Path

# 添加项目路径到Python路径
project_root = Path(__file__).parent.parent.parent
sys.path.insert(0, str(project_root / "src" / "backend"))

# 添加当前目录到Python路径以便导入ai_infra_auth
current_dir = Path(__file__).parent
sys.path.insert(0, str(current_dir))

# 导入自定义认证器
from ai_infra_auth import AIInfraMatrixAuthenticator

# JupyterHub基础配置
c = get_config()  #type:ignore

# 认证器配置
c.JupyterHub.authenticator_class = AIInfraMatrixAuthenticator

# AI-Infra-Matrix认证器配置
backend_url = os.environ.get('AI_INFRA_BACKEND_URL', 'http://localhost:8082')  # 修正后端端口
api_token = os.environ.get('AI_INFRA_API_TOKEN', '')

c.AIInfraMatrixAuthenticator.backend_api_url = backend_url
c.AIInfraMatrixAuthenticator.backend_api_token = api_token
c.AIInfraMatrixAuthenticator.enable_auth_state = False
c.AIInfraMatrixAuthenticator.auto_login = os.environ.get('JUPYTERHUB_AUTO_LOGIN', 'true').lower() == 'true'
c.AIInfraMatrixAuthenticator.token_refresh_threshold = int(os.environ.get('JUPYTERHUB_TOKEN_REFRESH_THRESHOLD', '300'))  # 5分钟

# 允许通过URL参数进行token认证
c.AIInfraMatrixAuthenticator.allow_token_in_url = True

# 服务器配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8091'

# 数据目录 - 修正配置项名称
project_data_dir = Path("/srv/data/jupyterhub")
project_data_dir.mkdir(parents=True, exist_ok=True)

# 修正配置项名称
c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'
c.JupyterHub.cookie_secret_file = str(project_data_dir / "cookie_secret")

# 代理配置 - 使用新的配置项
c.ConfigurableHTTPProxy.auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'ai-infra-proxy-token')

# 日志配置
c.Application.log_level = 'INFO'

# 用户配置
c.Authenticator.admin_users = {'admin', 'jupyter-admin'}
c.Authenticator.allowed_users = set()  # 允许所有用户

# Spawner配置 - 使用SystemdSpawner或DockerSpawner
try:
    from dockerspawner import DockerSpawner
    c.JupyterHub.spawner_class = DockerSpawner
    
    # Docker配置
    c.DockerSpawner.image = 'jupyter/datascience-notebook:latest'
    c.DockerSpawner.network_name = 'jupyterhub-network'
    c.DockerSpawner.remove = True
    c.DockerSpawner.use_internal_ip = True
    
    # 挂载卷配置
    c.DockerSpawner.volumes = {
        '/shared': '/home/jovyan/shared',  # NFS共享目录
        '/srv/jupyterhub/notebooks': '/home/jovyan/work',  # 用户工作目录
    }
    
    # 环境变量
    c.DockerSpawner.environment = {
        'JUPYTER_ENABLE_LAB': '1',
        'AI_INFRA_API_URL': 'http://host.docker.internal:8080',
        'JUPYTERHUB_K8S_NAMESPACE': 'jupyterhub-jobs',
    }
    
except ImportError:
    # 如果没有DockerSpawner，使用默认Spawner
    print("DockerSpawner 不可用，使用本地进程启动器")
    c.JupyterHub.spawner_class = 'jupyterhub.spawner.LocalProcessSpawner'

# 自定义服务配置 - 暂时禁用直到服务准备好
# c.JupyterHub.services = [
#     {
#         'name': 'ai-infra-k8s-service',
#         'url': 'http://localhost:8080',
#         'command': [
#             'python', '-m', 'ai_infra_k8s_service',
#             '--port=8080'
#         ],
#         'environment': {
#             'JUPYTERHUB_API_TOKEN': os.environ.get('JUPYTERHUB_API_TOKEN', ''),
#             'JUPYTERHUB_K8S_NAMESPACE': 'jupyterhub-jobs',
#         }
#     }
# ]

# 数据库配置 - 支持PostgreSQL
postgres_host = os.environ.get('POSTGRES_HOST', 'localhost')
postgres_port = os.environ.get('POSTGRES_PORT', '5432')
postgres_db = os.environ.get('POSTGRES_DB', 'jupyter_hub')
postgres_user = os.environ.get('POSTGRES_USER', 'postgres')
postgres_password = os.environ.get('POSTGRES_PASSWORD', 'postgres')

# 在Docker环境中使用PostgreSQL，本地开发使用SQLite
if postgres_host != 'localhost':
    c.JupyterHub.db_url = f'postgresql://{postgres_user}:{postgres_password}@{postgres_host}:{postgres_port}/{postgres_db}'
else:
    c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'

# 安全配置
c.JupyterHub.cookie_secret_file = str(project_data_dir / "cookie_secret")
c.JupyterHub.proxy_auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', '')

# 启动钩子 - 初始化K8s GPU集成环境
def pre_spawn_hook(spawner):
    """在启动用户环境前执行的钩子"""
    username = spawner.user.name
    
    # 创建用户工作目录
    user_dir = Path(f'/srv/jupyterhub/notebooks/{username}')
    user_dir.mkdir(parents=True, exist_ok=True)
    
    # 复制初始化notebook到用户目录
    init_notebook_src = Path(__file__).parent / 'notebooks' / 'k8s-gpu-integration-init.ipynb'
    init_notebook_dst = user_dir / 'k8s-gpu-integration-init.ipynb'
    
    if init_notebook_src.exists() and not init_notebook_dst.exists():
        import shutil
        shutil.copy2(init_notebook_src, init_notebook_dst)
        
    # 创建示例脚本目录
    examples_dir = user_dir / 'examples'
    examples_dir.mkdir(exist_ok=True)
    
    # 复制示例脚本
    examples_src = Path(__file__).parent / 'examples'
    if examples_src.exists():
        import shutil
        for script_file in examples_src.glob('*.py'):
            dst_file = examples_dir / script_file.name
            if not dst_file.exists():
                shutil.copy2(script_file, dst_file)

c.Spawner.pre_spawn_hook = pre_spawn_hook

# 自定义模板
c.JupyterHub.template_paths = ['/srv/jupyterhub/templates']

# API配置
api_token = os.environ.get('JUPYTERHUB_API_TOKEN', 'ai-infra-hub-api-token')
c.JupyterHub.api_tokens = {
    api_token: 'ai-infra-service'
}

# Configurable HTTP Proxy配置
proxy_auth_token = os.environ.get('CONFIGPROXY_AUTH_TOKEN', 'ai-infra-proxy-token')
c.JupyterHub.proxy_auth_token = proxy_auth_token

print("JupyterHub配置加载完成 - AI基础设施矩阵集成模式")
print(f"项目根目录: {project_root}")
print(f"数据目录: {c.JupyterHub.data_dir}")
print(f"绑定地址: {c.JupyterHub.bind_url}")
