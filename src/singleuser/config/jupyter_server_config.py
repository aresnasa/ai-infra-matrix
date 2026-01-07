# Jupyter Server configuration for VS Code Server integration
# Place this file in ~/.jupyter/jupyter_server_config.py

import os
import shutil

c = get_config()  # noqa: F821

# ========================================
# VS Code Server Proxy Configuration
# ========================================

def get_vscode_command(port):
    """Generate VS Code Server command with dynamic port"""
    code_server = shutil.which('code-server')
    if not code_server:
        for path in ['/usr/bin/code-server', '/usr/local/bin/code-server']:
            if os.path.exists(path):
                code_server = path
                break
    
    if not code_server:
        return None
    
    workspace_dir = os.environ.get('WORKSPACE_DIR', '/home/jovyan/work')
    
    return [
        code_server,
        '--bind-addr', f'0.0.0.0:{port}',
        '--auth', 'none',
        '--disable-telemetry',
        '--disable-update-check',
        workspace_dir
    ]

# ServerProxy configuration
c.ServerProxy.servers = {
    'vscode': {
        'command': get_vscode_command,
        'timeout': 60,
        'launcher_entry': {
            'title': 'VS Code',
            'enabled': True,
            'category': 'IDE',
        },
        'new_browser_tab': True,
        'absolute_url': False,
    },
}

# ========================================
# Security Settings
# ========================================

# Allow connections from any origin (required for proxy)
c.ServerApp.allow_origin = '*'
c.ServerApp.allow_credentials = True

# Disable token authentication when running behind JupyterHub
# JupyterHub handles authentication
if os.environ.get('JUPYTERHUB_API_TOKEN'):
    c.ServerApp.token = ''

# ========================================
# Resource Limits
# ========================================

# Set resource limits for the server
c.ServerApp.max_body_size = 100 * 1024 * 1024  # 100MB
c.ServerApp.max_buffer_size = 100 * 1024 * 1024  # 100MB

# ========================================
# Logging
# ========================================

c.ServerApp.log_level = 'INFO'
