# AI Infrastructure Matrix - 离线环境部署指南

> 🔒 **完全离线部署解决方案** - 在无互联网环境中快速部署AI Infrastructure Matrix

## 📋 目录

- [概述](#概述)
- [系统要求](#系统要求)
- [准备工作](#准备工作)
- [离线部署流程](#离线部署流程)
- [服务管理](#服务管理)
- [故障排除](#故障排除)
- [配置说明](#配置说明)

## 🎯 概述

AI Infrastructure Matrix离线环境部署方案允许您在完全断网的环境中部署和运行完整的AI基础设施平台。该方案包含：

- 🐳 **镜像打包系统** - 自动导出所有必需Docker镜像
- 🚀 **一键部署脚本** - 全自动化离线环境部署
- 📊 **服务监控** - 完整的健康检查和状态监控
- 🔧 **配置管理** - 离线环境优化配置

## 💻 系统要求

### 硬件要求

| 组件 | 最低要求 | 推荐配置 |
|------|----------|----------|
| CPU | 4核 | 8核+ |
| 内存 | 8GB | 16GB+ |
| 存储 | 50GB | 100GB+ |
| 网络 | 无需外网 | 局域网连通 |

### 软件要求

- **操作系统**: Linux (Ubuntu 18.04+, CentOS 7+), macOS 10.15+
- **Docker**: 20.10+
- **Docker Compose**: 2.0+ (或 docker-compose 1.28+)
- **Bash**: 4.0+
- **工具**: curl, lsof, gzip (通常系统自带)

## 🛠️ 准备工作

### 第一步：在有网络环境中准备离线包

在有互联网访问的机器上执行以下步骤：

```bash
# 1. 克隆或获取项目代码
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix

# 2. 构建AI-Infra镜像 (如果未构建)
./build.sh prod --version v0.3.6-dev

# 3. 导出所有必需镜像到离线包
./scripts/export-offline-images.sh
```

导出完成后，你将得到：
```
offline-images/
├── ai-infra-third-party-*.tar.gz     # 第三方依赖镜像
├── ai-infra-ai-infra-*.tar.gz        # AI-Infra组件镜像  
├── ai-infra-matrix-complete-*.tar.gz # 完整镜像包
├── image-manifest.txt                # 镜像清单
└── import-images.sh                  # 镜像导入脚本
```

### 第二步：传输到目标环境

将整个项目目录(包括offline-images)复制到目标离线环境：

```bash
# 方法1: 使用scp (如果有网络连通)
scp -r ai-infra-matrix/ user@target-server:/path/to/

# 方法2: 使用U盘/移动硬盘
# 直接复制整个 ai-infra-matrix 目录

# 方法3: 打包传输
tar -czf ai-infra-matrix-offline.tar.gz ai-infra-matrix/
# 在目标环境解压
tar -xzf ai-infra-matrix-offline.tar.gz
```

## 🚀 离线部署流程

### 快速部署 (推荐)

```bash
cd ai-infra-matrix
./offline-start.sh
```

该脚本将自动执行：
1. ✅ 检查系统依赖和端口占用
2. ✅ 导入Docker镜像
3. ✅ 创建环境配置文件
4. ✅ 创建必要的数据目录
5. ✅ 分阶段启动所有服务
6. ✅ 执行健康检查

### 手动部署流程

如果需要更精细的控制，可以手动执行各个步骤：

#### 1. 导入镜像

```bash
cd offline-images
./import-images.sh
```

#### 2. 配置环境

```bash
# 复制环境配置模板
cp .env.prod.example .env.prod

# 编辑配置文件 (可选)
vim .env.prod
```

#### 3. 启动服务

```bash
# 使用Docker Compose启动
docker compose up -d

# 或使用传统命令
docker-compose up -d
```

## 🎛️ 服务管理

### 基本操作

```bash
# 启动所有服务
./offline-start.sh start

# 停止所有服务  
./offline-start.sh stop

# 重启服务
./offline-start.sh restart

# 查看服务状态
./offline-start.sh status

# 健康检查
./offline-start.sh health

# 查看日志
./offline-start.sh logs [服务名]
```

### 单独管理服务

```bash
# 启动特定服务
docker compose up -d postgres redis

# 重启特定服务
docker compose restart nginx

# 查看特定服务日志
docker compose logs -f backend

# 查看所有服务状态
docker compose ps
```

## 🌐 服务访问

部署成功后，可通过以下地址访问各项服务：

| 服务 | 访问地址 | 说明 |
|------|----------|------|
| 🏠 **主页面** | http://localhost:8080 | AI-Infra主界面 |
| 🔐 **SSO登录** | http://localhost:8080/sso/ | 单点登录系统 |
| 📊 **JupyterHub** | http://localhost:8080/jupyter | Jupyter笔记本环境 |
| 🔧 **Gitea** | http://localhost:8080/gitea/ | Git代码仓库 |
| 📈 **Kafka UI** | http://localhost:9095 | 消息队列管理 |
| 👥 **LDAP管理** | http://localhost:8080/phpldapadmin/ | 用户目录管理 |
| 🗄️ **Redis监控** | http://localhost:8001 | Redis数据库监控 |

### 默认账号

- **管理员账号**: `admin` / `admin123`
- **LDAP管理**: `cn=admin,dc=ai-infra,dc=com` / `ldap_admin_2024`

## 🔧 配置说明

### 环境变量配置

主要配置文件：`.env.prod`

```bash
# 基础配置
COMPOSE_PROJECT_NAME=ai-infra-matrix-offline
IMAGE_TAG=v0.3.6-dev
BUILD_ENV=production
DEBUG_MODE=false

# 网络配置
EXTERNAL_HOST=localhost
EXTERNAL_PORT=8080
EXTERNAL_SCHEME=http

# 数据库配置
POSTGRES_DB=ai_infra
POSTGRES_USER=ai_infra_user
POSTGRES_PASSWORD=ai_infra_password_2024

# Redis配置  
REDIS_PASSWORD=redis_password_2024

# LDAP配置
LDAP_ADMIN_PASSWORD=ldap_admin_2024

# 离线模式配置
OFFLINE_MODE=true
DISABLE_EXTERNAL_APIS=true
```

### 端口配置

| 服务 | 内部端口 | 外部端口 | 可修改 |
|------|----------|----------|---------|
| Nginx | 80 | 8080 | ✅ |
| PostgreSQL | 5432 | - | ❌ |
| Redis | 6379 | - | ❌ |
| Kafka | 9092 | 9094 | ✅ |
| LDAP | 389 | - | ❌ |

### 数据持久化

数据将保存在以下目录：

```
data/
├── postgres/          # PostgreSQL数据
├── redis/             # Redis数据  
├── kafka/             # Kafka数据
├── ldap/              # LDAP数据
├── gitea/             # Gitea数据
├── jupyter/           # JupyterHub数据
└── minio/             # 文件存储数据
```

## 🔍 故障排除

### 常见问题

#### 1. 端口被占用

```bash
# 查看端口占用
lsof -i :8080

# 停止占用进程或修改端口配置
vim .env.prod  # 修改EXTERNAL_PORT
```

#### 2. 镜像导入失败

```bash
# 检查Docker服务状态
sudo systemctl status docker

# 手动导入单个镜像
docker load -i offline-images/ai-infra-matrix-complete-*.tar.gz

# 查看已导入镜像
docker images | grep ai-infra
```

#### 3. 服务启动失败

```bash
# 查看容器状态
docker compose ps

# 查看失败容器日志
docker compose logs <服务名>

# 检查配置文件语法
docker compose config
```

#### 4. 健康检查失败

```bash
# 检查网络连通性
curl -I http://localhost:8080

# 检查服务进程
docker compose exec nginx ps aux

# 重启相关服务
docker compose restart nginx backend
```

### 日志查看

```bash
# 查看所有服务日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f nginx
docker compose logs -f backend  
docker compose logs -f postgres

# 查看系统资源使用
docker stats
```

### 性能调优

```bash
# 清理Docker缓存
docker system prune -f

# 查看磁盘使用
df -h
du -sh data/*

# 调整服务资源限制
vim docker-compose.yml  # 修改resources配置
```

## 🔄 升级和维护

### 版本升级

```bash
# 1. 停止服务
./offline-start.sh stop

# 2. 备份数据
tar -czf backup-$(date +%Y%m%d).tar.gz data/

# 3. 更新镜像
./scripts/export-offline-images.sh  # 在有网络环境中
cd offline-images && ./import-images.sh

# 4. 启动服务
./offline-start.sh start
```

### 数据备份

```bash
# 完整备份
./offline-start.sh stop
tar -czf ai-infra-backup-$(date +%Y%m%d).tar.gz data/ .env.prod

# 数据库备份
docker compose exec postgres pg_dump -U ai_infra_user ai_infra > backup.sql
```

### 系统清理

```bash
# 清理所有服务和数据 (谨慎使用)
./offline-start.sh clean

# 清理Docker缓存
docker system prune -a -f

# 清理日志文件
find logs/ -name "*.log" -mtime +30 -delete
```

## 📚 高级功能

### 自定义配置

1. **修改端口映射**：编辑 `.env.prod` 中的端口配置
2. **调整资源限制**：编辑 `docker-compose.yml` 中的resources部分
3. **配置HTTPS**：添加SSL证书和nginx配置
4. **集成内部DNS**：配置服务发现和域名解析

### 扩展部署

- **多节点部署**：使用Docker Swarm或Kubernetes
- **高可用配置**：配置数据库主从、Redis集群
- **监控告警**：集成Prometheus + Grafana
- **日志收集**：配置ELK Stack或Fluentd

### 安全加固

- 修改默认密码和密钥
- 配置防火墙规则
- 启用访问日志审计
- 定期安全更新

## 📞 技术支持

如需技术支持，请检查：

1. 📖 **项目文档**: [README.md](README.md)
2. 🐛 **问题反馈**: [GitHub Issues](https://github.com/aresnasa/ai-infra-matrix/issues)
3. 💬 **社区讨论**: 项目Discussion区
4. 📧 **邮件支持**: admin@example.com

---

## 🎉 部署成功！

恭喜！您已成功在离线环境中部署AI Infrastructure Matrix。

现在可以访问 **http://localhost:8080** 开始使用您的AI基础设施平台！

---

*📝 本文档最后更新: $(date)*
*🔖 适用版本: v0.3.6-dev*