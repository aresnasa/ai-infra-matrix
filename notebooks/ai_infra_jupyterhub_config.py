# JupyterHub配置文件
# 用于AI基础设施矩阵项目的K8s GPU作业集成

import os
import sys
from pathlib import Path

# 添加项目路径到Python路径
project_root = Path(__file__).parent.parent.parent
sys.path.insert(0, str(project_root / "src" / "backend"))

# JupyterHub基础配置
c = get_config()

# 服务器配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8081'

# 数据目录
c.JupyterHub.data_dir = '/srv/jupyterhub'
c.JupyterHub.cookie_dir = '/srv/jupyterhub/cookies'

# 日志配置
c.Application.log_level = 'INFO'
c.JupyterHub.log_file = '/var/log/jupyterhub.log'

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
    c.JupyterHub.spawner_class = 'jupyterhub.spawner.LocalProcessSpawner'

# 自定义服务配置
c.JupyterHub.services = [
    {
        'name': 'ai-infra-k8s-service',
        'url': 'http://localhost:8080',
        'command': [
            'python', '-m', 'ai_infra_k8s_service',
            '--port=8080'
        ],
        'environment': {
            'JUPYTERHUB_API_TOKEN': os.environ.get('JUPYTERHUB_API_TOKEN', ''),
            'JUPYTERHUB_K8S_NAMESPACE': 'jupyterhub-jobs',
        }
    }
]

# 数据库配置
c.JupyterHub.db_url = 'sqlite:///srv/jupyterhub/jupyterhub.sqlite'

# 安全配置
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/cookie_secret'
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
c.JupyterHub.api_tokens = {
    os.environ.get('JUPYTERHUB_API_TOKEN', 'default-token'): 'ai-infra-service'
}

# 启动后钩子
def post_spawn_hook(spawner, **kwargs):
    """启动后执行的钩子"""
    # 这里可以添加启动后的初始化逻辑
    pass

c.Spawner.post_spawn_hook = post_spawn_hook

# 日志轮转配置
c.JupyterHub.extra_log_file = '/var/log/jupyterhub-debug.log'

print("JupyterHub配置加载完成 - AI基础设施矩阵集成模式")
print(f"项目根目录: {project_root}")
print(f"数据目录: {c.JupyterHub.data_dir}")
print(f"绑定地址: {c.JupyterHub.bind_url}")
