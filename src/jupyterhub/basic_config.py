# AI-Infra-Matrix 基础 JupyterHub 配置
# 用于测试和验证基本功能

import os
from pathlib import Path

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

# 使用默认认证器（本地用户）
c.JupyterHub.authenticator_class = 'jupyterhub.auth.DummyAuthenticator'
c.DummyAuthenticator.password = 'test123'

# 管理员用户
c.JupyterHub.admin_users = {'admin'}

# 日志配置
c.JupyterHub.log_level = 'INFO'

print("=== JupyterHub基础配置加载完成 ===")
print(f"数据目录: {project_data_dir}")
print(f"绑定地址: http://0.0.0.0:8000")
