"""
JupyterHub后端集成配置 - 主配置文件
统一使用backend作为认证中心，删除所有冗余配置
"""

# 直接加载统一的后端集成配置
exec(open('/srv/jupyterhub/backend_integrated_config.py').read())
