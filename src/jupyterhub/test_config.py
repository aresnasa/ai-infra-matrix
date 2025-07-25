# JupyterHub配置文件 - 简化版本
# 用于测试基本功能

import os
from pathlib import Path

# JupyterHub基础配置
c = get_config()  #type:ignore

# 使用默认认证器进行测试
c.JupyterHub.authenticator_class = 'jupyterhub.auth.DummyAuthenticator'

# 服务器配置
c.JupyterHub.bind_url = 'http://0.0.0.0:8000'
c.JupyterHub.hub_bind_url = 'http://0.0.0.0:8091'

# 数据目录 - 使用项目内路径
project_root = Path(__file__).parent.parent.parent
project_data_dir = project_root / "data" / "jupyterhub"
project_data_dir.mkdir(parents=True, exist_ok=True)

c.JupyterHub.db_url = f'sqlite:///{project_data_dir}/jupyterhub.sqlite'

# 简化配置
c.JupyterHub.cookie_secret_file = str(project_data_dir / "cookie_secret")

# 日志配置
c.Application.log_level = 'INFO'

# 管理员用户
c.Authenticator.admin_users = {'admin', 'jupyter-admin'}

# 允许任何用户登录（测试用）
c.DummyAuthenticator.password = "test123"

# Spawner配置
c.JupyterHub.spawner_class = 'jupyterhub.spawner.SimpleLocalProcessSpawner'

print("JupyterHub测试配置加载完成 - 使用DummyAuthenticator")
print(f"项目根目录: {project_root}")
print(f"数据目录: {project_data_dir}")
print("绑定地址: http://0.0.0.0:8000")
