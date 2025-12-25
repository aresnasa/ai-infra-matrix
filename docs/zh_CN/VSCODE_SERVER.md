# VS Code Server 集成说明

本文档介绍如何在 JupyterHub 环境中使用 VS Code Server (code-server)。

## 功能概述

AI Infra Matrix 的 JupyterHub 单用户容器现已集成 VS Code Server 支持，用户可以通过以下方式使用 VS Code：

1. **Web 浏览器访问** - 通过 JupyterLab 界面直接启动 VS Code
2. **VS Code 客户端连接** - 使用本地 VS Code 客户端连接远程环境

## 使用方式

### 方式一：通过 JupyterLab Launcher 启动

1. 登录 JupyterHub
2. 启动您的 Jupyter 服务器
3. 在 JupyterLab 界面，点击 Launcher 中的 "VS Code" 图标
4. VS Code Server 将在新标签页中打开

### 方式二：直接 URL 访问

登录 JupyterHub 后，可以直接访问：

```
https://<your-jupyterhub-domain>/user/<username>/vscode/
```

例如：
```
https://jupyterhub.example.com/user/john/vscode/
```

### 方式三：使用 VS Code Remote 扩展

您也可以使用本地 VS Code 客户端通过 Remote - Tunnels 或 Remote - SSH 连接：

#### 使用 Remote - Tunnels（推荐）

1. 在本地安装 VS Code
2. 安装 "Remote - Tunnels" 扩展
3. 在容器内运行：
   ```bash
   code tunnel
   ```
4. 按提示完成 GitHub 认证
5. 在本地 VS Code 中连接到 tunnel

#### 使用 Remote - SSH

如果容器配置了 SSH 访问：

1. 安装 "Remote - SSH" 扩展
2. 配置 SSH 连接到容器
3. 在 VS Code 中通过 SSH 连接

## 预装扩展

默认预装的 VS Code 扩展：

- **Python** (`ms-python.python`) - Python 语言支持
- **Jupyter** (`ms-toolsai.jupyter`) - Jupyter Notebook 支持

## 环境变量配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `CODE_SERVER_PORT` | `8443` | VS Code Server 监听端口 |
| `CODE_SERVER_BIND_ADDR` | `0.0.0.0:8443` | VS Code Server 绑定地址 |
| `WORKSPACE_DIR` | `/home/jovyan/work` | 默认工作目录 |
| `START_CODE_SERVER` | `false` | 是否独立启动 code-server |

## 技术架构

```
┌─────────────────────────────────────────────────────────┐
│                      用户浏览器                          │
│                          │                              │
│         ┌────────────────┼────────────────┐             │
│         │                │                │             │
│         ▼                ▼                ▼             │
│   JupyterLab      VS Code UI       其他服务            │
└─────────┬─────────────────────────────────┬─────────────┘
          │                                 │
          │            nginx/代理            │
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
│  │   (端口 8888)           (端口 8443)              │   │
│  │                                                  │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## 安全说明

- VS Code Server 通过 `jupyter-server-proxy` 代理访问，复用 JupyterHub 的认证机制
- 无需单独的密码认证（由 JupyterHub 处理）
- 所有流量通过 JupyterHub 的 HTTPS 加密传输

## 故障排除

### VS Code 无法启动

1. 检查 code-server 是否正确安装：
   ```bash
   code-server --version
   ```

2. 检查 jupyter-server-proxy 是否安装：
   ```bash
   pip show jupyter-server-proxy
   ```

3. 查看 Jupyter Server 日志：
   ```bash
   cat ~/.jupyter/jupyter_server.log
   ```

### 扩展无法安装

在容器内手动安装扩展：
```bash
code-server --install-extension <extension-id>
```

### 性能问题

如果 VS Code 响应缓慢：

1. 检查容器资源限制（CPU/内存）
2. 关闭不必要的扩展
3. 增加容器资源配额

## 相关链接

- [code-server 官方文档](https://coder.com/docs/code-server/latest)
- [jupyter-server-proxy 文档](https://jupyter-server-proxy.readthedocs.io/)
- [VS Code Remote Development](https://code.visualstudio.com/docs/remote/remote-overview)
