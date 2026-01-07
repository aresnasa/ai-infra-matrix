# VS Code Server Integration

This document describes how to use VS Code Server (code-server) in the JupyterHub environment.

## Overview

The AI Infra Matrix JupyterHub single-user containers now include VS Code Server support. Users can access VS Code through:

1. **Web Browser Access** - Launch VS Code directly from JupyterLab interface
2. **VS Code Client Connection** - Connect using local VS Code client to remote environment

## Usage

### Method 1: Launch via JupyterLab Launcher

1. Log in to JupyterHub
2. Start your Jupyter server
3. In JupyterLab interface, click the "VS Code" icon in the Launcher
4. VS Code Server will open in a new browser tab

### Method 2: Direct URL Access

After logging into JupyterHub, access directly:

```
https://<your-jupyterhub-domain>/user/<username>/vscode/
```

Example:
```
https://jupyterhub.example.com/user/john/vscode/
```

### Method 3: VS Code Remote Extension

You can also use local VS Code client via Remote - Tunnels or Remote - SSH:

#### Using Remote - Tunnels (Recommended)

1. Install VS Code locally
2. Install "Remote - Tunnels" extension
3. In the container, run:
   ```bash
   code tunnel
   ```
4. Complete GitHub authentication as prompted
5. Connect to the tunnel from local VS Code

#### Using Remote - SSH

If the container has SSH access configured:

1. Install "Remote - SSH" extension
2. Configure SSH connection to the container
3. Connect via SSH in VS Code

## Pre-installed Extensions

Default pre-installed VS Code extensions:

- **Python** (`ms-python.python`) - Python language support
- **Jupyter** (`ms-toolsai.jupyter`) - Jupyter Notebook support

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CODE_SERVER_PORT` | `8443` | VS Code Server listen port |
| `CODE_SERVER_BIND_ADDR` | `0.0.0.0:8443` | VS Code Server bind address |
| `WORKSPACE_DIR` | `/home/jovyan/work` | Default workspace directory |
| `START_CODE_SERVER` | `false` | Whether to start code-server standalone |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    User Browser                          │
│                          │                              │
│         ┌────────────────┼────────────────┐             │
│         │                │                │             │
│         ▼                ▼                ▼             │
│   JupyterLab      VS Code UI      Other Services       │
└─────────┬─────────────────────────────────┬─────────────┘
          │                                 │
          │           nginx/proxy           │
          │                │                │
          ▼                ▼                │
┌─────────────────────────────────────────────────────────┐
│                    JupyterHub                            │
│                          │                              │
│         ┌────────────────┼────────────────┐             │
│         │                │                │             │
│         ▼                ▼                ▼             │
│   Hub Server    User Spawner      Proxy Server         │
└─────────┬─────────────────────────────────┬─────────────┘
          │                                 │
          │                                 │
          ▼                                 ▼
┌─────────────────────────────────────────────────────────┐
│              Singleuser Container                        │
│  ┌──────────────────────────────────────────────────┐   │
│  │                                                  │   │
│  │   Jupyter Server ◄──── jupyter-server-proxy ────┤   │
│  │        │                      │                  │   │
│  │        ▼                      ▼                  │   │
│  │   JupyterLab            code-server             │   │
│  │   (port 8888)           (port 8443)             │   │
│  │                                                  │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Security

- VS Code Server is accessed through `jupyter-server-proxy`, reusing JupyterHub's authentication
- No separate password authentication required (handled by JupyterHub)
- All traffic encrypted via JupyterHub's HTTPS

## Troubleshooting

### VS Code Won't Start

1. Check if code-server is installed correctly:
   ```bash
   code-server --version
   ```

2. Check if jupyter-server-proxy is installed:
   ```bash
   pip show jupyter-server-proxy
   ```

3. View Jupyter Server logs:
   ```bash
   cat ~/.jupyter/jupyter_server.log
   ```

### Extensions Won't Install

Manually install extensions in the container:
```bash
code-server --install-extension <extension-id>
```

### Performance Issues

If VS Code is slow:

1. Check container resource limits (CPU/memory)
2. Disable unnecessary extensions
3. Increase container resource quota

## Related Links

- [code-server Official Documentation](https://coder.com/docs/code-server/latest)
- [jupyter-server-proxy Documentation](https://jupyter-server-proxy.readthedocs.io/)
- [VS Code Remote Development](https://code.visualstudio.com/docs/remote/remote-overview)
