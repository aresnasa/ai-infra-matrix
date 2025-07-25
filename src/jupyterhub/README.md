# JupyterHub Docker部署

此目录包含基于ai-infra-matrix conda环境配置的JupyterHub Docker部署文件。

## 文件结构

```
jupyterhub/
├── Dockerfile                        # JupyterHub Docker镜像定义
├── requirements.txt                  # Python依赖列表（基于conda环境）
├── ai_infra_auth.py                 # 自定义认证器
├── ai_infra_jupyterhub_config.py    # JupyterHub配置文件
└── README.md                        # 此说明文件
```

## 版本信息

基于ai-infra-matrix conda环境的包版本：

- **Python**: 3.13.5
- **JupyterHub**: 5.3.0
- **JupyterLab**: 4.4.5
- **Notebook**: 7.4.4
- **Requests**: 2.32.4
- **Tornado**: 6.5.1
- **AIOHttp**: 3.12.14

## 构建和运行

### 使用Docker Compose

从项目根目录运行：

```bash
# 构建JupyterHub镜像
docker-compose -f src/docker-compose.yml build jupyterhub

# 启动JupyterHub服务
docker-compose -f src/docker-compose.yml --profile jupyterhub up -d

# 查看日志
docker-compose -f src/docker-compose.yml logs -f jupyterhub
```

### 直接使用Docker

```bash
# 构建镜像
cd jupyterhub
docker build -t ai-infra-jupyterhub:latest .

# 运行容器
docker run -d \
  --name ai-infra-jupyterhub \
  -p 8888:8000 \
  -e AI_INFRA_MATRIX_BACKEND_URL=http://backend:8080 \
  ai-infra-jupyterhub:latest
```

## 访问地址

启动成功后，可通过以下地址访问：

- **JupyterHub主页**: http://localhost:8888
- **管理面板**: http://localhost:8888/hub/admin
- **API文档**: http://localhost:8888/hub/api

## 认证集成

JupyterHub使用自定义认证器(`ai_infra_auth.py`)与AI-Infra-Matrix后端集成，支持：

- JWT令牌验证
- 自动用户创建
- 统一登录状态维持
- 后端API集成

## 配置文件

主要配置在`ai_infra_jupyterhub_config.py`中，包括：

- 认证器设置
- 数据库连接
- 用户权限配置
- 日志设置

## 环境变量

支持的环境变量：

- `AI_INFRA_MATRIX_BACKEND_URL`: 后端API地址
- `JWT_SECRET`: JWT密钥
- `JUPYTERHUB_API_TOKEN`: JupyterHub API令牌
- `POSTGRES_*`: PostgreSQL数据库配置

## 故障排除

### 检查服务状态

```bash
# 检查容器状态
docker ps | grep jupyterhub

# 查看详细日志
docker logs ai-infra-jupyterhub

# 进入容器调试
docker exec -it ai-infra-jupyterhub /bin/bash
```

### 常见问题

1. **认证失败**: 检查JWT_SECRET配置和后端API连接
2. **端口冲突**: 确保8888端口未被占用
3. **权限问题**: 检查文件权限和用户配置

## 开发模式

开发时可以挂载本地文件进行实时调试：

```bash
docker run -d \
  --name ai-infra-jupyterhub-dev \
  -p 8888:8000 \
  -v $(pwd):/srv/jupyterhub/custom \
  ai-infra-jupyterhub:latest
```
