# AI Infrastructure Matrix - 快速开始指南

## 🎯 5分钟快速部署

本指南帮助您在5分钟内快速部署并运行AI Infrastructure Matrix。

## 📋 前置检查

在开始之前，请确保您的系统满足以下要求：

```bash
# 检查Docker版本
docker --version
# 应该显示 20.10+ 版本

# 检查Docker Compose版本  
docker compose version
# 应该显示 2.0+ 版本

# 检查可用内存
free -h
# 至少需要4GB可用内存

# 检查磁盘空间
df -h
# 至少需要10GB可用空间
```

## ⚡ 一键部署

### 方法1：完全自动化部署

```bash
# 1. 克隆项目
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix

# 2. 一键部署（开发环境）
./scripts/build.sh dev --up --test
```

### 方法2：分步部署

```bash
# 1. 克隆项目
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix

# 2. 环境配置
cp .env.example .env

# 3. 构建镜像
./scripts/build.sh dev

# 4. 启动服务
docker compose up -d

# 5. 等待服务启动（约30秒）
sleep 30

# 6. 健康检查
./scripts/test-health.sh
```

## 🌐 访问服务

部署完成后，打开浏览器访问以下地址：

| 服务 | 地址 | 说明 |
|------|------|------|
| **主页** | <http://localhost:8080> | 项目主页和导航 |
| **SSO登录** | <http://localhost:8080/sso/> | 统一身份认证 |
| **JupyterHub** | <http://localhost:8080/jupyter> | 机器学习平台 |
| **Gitea** | <http://localhost:8080/gitea/> | 代码仓库管理 |
| **管理后台** | <http://localhost:8080/admin> | 系统管理界面 |

### 默认账号

| 服务 | 用户名 | 密码 | 权限 |
|------|--------|------|------|
| **系统管理员** | `admin` | `admin123` | 最高权限 |
| **普通用户** | `user` | `user123` | 基础权限 |

## ✅ 验证部署

### 1. 检查服务状态

```bash
# 查看所有服务状态
docker compose ps

# 应该看到所有服务都是 "Up" 状态
```

### 2. 运行健康检查

```bash
# 运行完整健康检查
./scripts/test-health.sh

# 预期输出：
# ✅ Nginx 服务正常
# ✅ Backend API 正常  
# ✅ Frontend 正常
# ✅ JupyterHub 正常
# ✅ Gitea 正常
# ✅ PostgreSQL 正常
# ✅ Redis 正常
```

### 3. 测试核心功能

```bash
# 测试API访问
curl http://localhost:8080/api/health

# 测试前端访问
curl -I http://localhost:8080

# 测试JupyterHub
curl -I http://localhost:8080/jupyter

# 测试Gitea
curl -I http://localhost:8080/gitea
```

## 🔐 首次登录设置

### 1. 管理员登录

1. 访问 <http://localhost:8080/sso/>
2. 使用管理员账号登录：`admin` / `admin123`
3. 完成首次登录配置

### 2. JupyterHub设置

1. 访问 <http://localhost:8080/jupyter>
2. 使用管理员账号登录
3. 创建第一个Notebook测试环境

### 3. Gitea设置

1. 访问 <http://localhost:8080/gitea/>
2. 使用管理员账号登录
3. 创建第一个代码仓库

## 🚀 开始使用

### 创建您的第一个项目

1. **在Gitea中创建代码仓库**

   ```bash
   # 或者通过命令行
   cd /tmp
   git clone http://localhost:8080/gitea/admin/my-first-project.git
   cd my-first-project
   echo "# My First AI Project" > README.md
   git add README.md
   git commit -m "Initial commit"
   git push origin main
   ```

2. **在JupyterHub中开始机器学习**

   - 访问 <http://localhost:8080/jupyter>
   - 启动Notebook服务器
   - 创建新的Python Notebook
   - 开始您的ML/AI项目

3. **使用统一认证**

   - 所有服务使用相同的账号密码
   - 在一个服务登录后，其他服务自动登录
   - 集中的用户和权限管理

## 🔧 基础配置

### 修改默认密码

```bash
# 编辑环境配置文件
vi .env

# 修改以下配置项
ADMIN_PASSWORD=your_secure_password
POSTGRES_PASSWORD=your_db_password  
REDIS_PASSWORD=your_redis_password
JWT_SECRET_KEY=your_jwt_secret

# 重启服务应用新配置
docker compose down
docker compose up -d
```

### 添加新用户

1. **通过管理界面**
   - 访问 <http://localhost:8080/admin>
   - 进入用户管理
   - 添加新用户

2. **通过API**

   ```bash
   curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -d '{
       "username": "newuser",
       "email": "newuser@example.com", 
       "password": "userpassword"
     }'
   ```

### 配置GPU支持（可选）

```bash
# 安装NVIDIA Container Toolkit
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://nvidia.github.io/libnvidia-container/stable/deb/$(ARCH) /" | sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
sudo apt-get update && sudo apt-get install -y nvidia-container-toolkit

# 重启Docker
sudo systemctl restart docker

# 重新部署启用GPU
ENABLE_GPU=true docker compose up -d
```

## 📊 监控和管理

### 查看服务日志

```bash
# 查看所有服务日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f jupyterhub
docker compose logs -f nginx
```

### 监控资源使用

```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
docker system df

# 查看网络连接
docker network ls
```

### 备份重要数据

```bash
# 备份数据库
docker exec ai-infra-postgres pg_dump -U ai_infra_user ai_infra_db > backup.sql

# 备份用户数据
docker run --rm -v ai-infra-matrix_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_backup.tar.gz -C /data .

# 备份JupyterHub用户文件
docker run --rm -v ai-infra-matrix_jupyterhub_data:/data -v $(pwd):/backup alpine tar czf /backup/jupyterhub_backup.tar.gz -C /data .
```

## 🔧 常见问题解决

### 端口占用问题

```bash
# 检查端口占用
lsof -i :8080
lsof -i :5432

# 停止占用端口的进程
sudo kill -9 <PID>

# 或者修改端口配置
vi docker-compose.yml
# 修改 ports: "8080:80" 为其他端口
```

### 内存不足

```bash
# 检查系统内存
free -h

# 增加Docker内存限制
# 在docker-compose.yml中添加
services:
  backend:
    mem_limit: 512m
  frontend:
    mem_limit: 256m
```

### 服务启动超时

```bash
# 增加健康检查超时时间
# 在docker-compose.yml中修改
healthcheck:
  interval: 30s
  timeout: 10s
  retries: 10
  start_period: 60s
```

### 数据库连接失败

```bash
# 检查数据库状态
docker compose logs postgres

# 重置数据库
docker compose down -v
docker compose up postgres -d
sleep 30
docker compose up -d
```

## 🎯 下一步

恭喜！您已经成功部署了AI Infrastructure Matrix。接下来您可以：

1. **阅读用户手册** - 了解详细功能
2. **查看API文档** - 集成其他系统
3. **参与开发** - 贡献代码和功能
4. **部署到生产** - 配置生产环境

## 📚 相关文档

- [用户操作手册](USER_GUIDE.md)
- [开发环境搭建](DEVELOPMENT_SETUP.md)
- [生产部署指南](PRODUCTION_DEPLOYMENT.md)
- [API接口文档](API_REFERENCE.md)
- [故障排除指南](TROUBLESHOOTING.md)

## 💬 获取帮助

如果遇到问题，可以通过以下方式获取帮助：

- 📖 查看[完整文档](README.md)
- 🐛 提交[问题报告](https://github.com/aresnasa/ai-infra-matrix/issues)
- 💬 加入[社区讨论](https://github.com/aresnasa/ai-infra-matrix/discussions)
- 📧 发送邮件：support@example.com

---

**部署时间**: 约5分钟  
**最后更新**: 2025年8月20日  
**适用版本**: v0.0.3.3+
