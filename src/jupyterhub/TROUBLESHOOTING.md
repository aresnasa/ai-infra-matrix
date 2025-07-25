# JupyterHub Docker构建故障排除指南

## 🚨 常见网络问题及解决方案

### 问题1: Docker Hub连接超时
```
ERROR: failed to solve: DeadlineExceeded: failed to fetch oauth token
```

### 解决方案：

#### 方案A: 使用简化版Dockerfile
```bash
# 使用最小化依赖的版本
docker build -f Dockerfile.minimal -t ai-infra-jupyterhub:minimal .
```

#### 方案B: 使用中国镜像源
```bash
# 使用国内镜像源版本
docker build -f Dockerfile.china -t ai-infra-jupyterhub:china .
```

#### 方案C: 使用构建脚本(带重试)
```bash
# 使用自动重试脚本
./build.sh --max-retries 5 --retry-delay 30
```

#### 方案D: 手动配置Docker镜像源
创建 `/etc/docker/daemon.json`:
```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
```

然后重启Docker:
```bash
sudo systemctl restart docker
```

## 🔧 快速构建命令

### 1. 基本构建
```bash
cd jupyterhub
docker build -t ai-infra-jupyterhub:latest .
```

### 2. 无缓存构建
```bash
docker build --no-cache -t ai-infra-jupyterhub:latest .
```

### 3. 使用特定网络模式
```bash
docker build --network=host -t ai-infra-jupyterhub:latest .
```

### 4. 设置构建超时
```bash
timeout 1800 docker build -t ai-infra-jupyterhub:latest .
```

## 📋 可用的Dockerfile版本

| 文件名 | 特点 | 适用场景 |
|--------|------|----------|
| `Dockerfile` | 完整功能版本 | 网络良好时使用 |
| `Dockerfile.china` | 使用国内镜像源 | 国内网络环境 |
| `Dockerfile.minimal` | 最小化依赖 | 快速测试/网络不稳定 |

## 🚀 构建后测试

### 验证镜像
```bash
# 查看镜像
docker images ai-infra-jupyterhub

# 快速测试
docker run --rm ai-infra-jupyterhub:latest python --version
docker run --rm ai-infra-jupyterhub:latest jupyterhub --version
```

### 运行容器
```bash
# 基本运行
docker run -d --name test-jupyterhub -p 8888:8000 ai-infra-jupyterhub:latest

# 查看日志
docker logs test-jupyterhub

# 清理测试容器
docker stop test-jupyterhub && docker rm test-jupyterhub
```

## 🔍 调试工具

### 进入容器调试
```bash
docker run -it --entrypoint /bin/bash ai-infra-jupyterhub:latest
```

### 检查网络连接
```bash
# 测试Docker Hub连接
curl -I https://registry-1.docker.io/v2/

# 测试PyPI连接
curl -I https://pypi.org/simple/

# 检查DNS解析
nslookup docker.io
```

### 清理Docker缓存
```bash
# 清理构建缓存
docker builder prune -f

# 清理未使用的镜像
docker image prune -f

# 完全清理
docker system prune -af
```

## ⚡ 应急方案

如果所有构建方案都失败，可以：

1. **使用已有的Python环境**：
   ```bash
   # 直接在本地运行JupyterHub
   conda activate ai-infra-matrix
   jupyterhub -f ai_infra_jupyterhub_config.py
   ```

2. **使用官方JupyterHub镜像**：
   ```bash
   # 使用官方镜像并挂载配置
   docker run -d \
     --name jupyterhub \
     -p 8888:8000 \
     -v $(pwd):/srv/jupyterhub/custom \
     jupyterhub/jupyterhub:latest
   ```

3. **使用docker-compose**：
   ```bash
   # 在src目录下使用已配置的compose
   cd ../src
   docker-compose --profile jupyterhub up --build
   ```

## 📞 获取帮助

如果问题仍然存在：
1. 检查Docker版本: `docker --version`
2. 检查网络状态: `docker info`
3. 查看详细错误: `docker build . 2>&1 | tee build.log`
4. 尝试其他镜像源或VPN连接
