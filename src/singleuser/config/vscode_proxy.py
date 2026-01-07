"""
jupyter-vscode-proxy configuration for JupyterHub
This module provides VS Code Server integration via jupyter-server-proxy

Users can access VS Code through:
1. JupyterLab launcher icon
2. Direct URL: /user/{username}/vscode/

Reference: https://github.com/betatim/jupyter-vscode-proxy
"""

import os
import shutil


def setup_vscode():
    """
    Setup function for jupyter-server-proxy to launch VS Code Server (code-server)
    
    Returns:
        dict: Configuration for jupyter-server-proxy
    """
    
    # Find code-server binary
    code_server_path = shutil.which('code-server')
    
    if not code_server_path:
        # Check common locations
        possible_paths = [
            '/usr/bin/code-server',
            '/usr/local/bin/code-server',
            os.path.expanduser('~/.local/bin/code-server'),
        ]
        for path in possible_paths:
            if os.path.exists(path):
                code_server_path = path
                break
    
    if not code_server_path:
        raise FileNotFoundError("code-server not found. Please install code-server first.")
    
    # Get environment configurations
    workspace_dir = os.environ.get('WORKSPACE_DIR', '/home/jovyan/work')
    
    return {
        'command': [
            code_server_path,
            '--bind-addr', '0.0.0.0:{port}',
            '--auth', 'none',
            '--disable-telemetry',
            '--disable-update-check',
            workspace_dir
        ],
        'timeout': 30,
        'launcher_entry': {
            'title': 'VS Code',
            'icon_path': os.path.join(
                os.path.dirname(os.path.abspath(__file__)),
                'icons',
                'vscode.svg'
            ) if os.path.exists(os.path.join(
                os.path.dirname(os.path.abspath(__file__)),
                'icons',
                'vscode.svg'
            )) else None,
            'enabled': True,
            'category': 'IDE'
        },
        'new_browser_tab': True,
        'absolute_url': False,
    }


# Entry point for jupyter_server_proxy
def _jupyter_server_extension_paths():
    """Define the Jupyter server extension paths"""
    return [{'module': 'vscode_proxy'}]


# Provide setup function for jupyter-server-proxy discovery
def _load_jupyter_server_extension(nbapp):
    """Load the Jupyter server extension"""
    pass
