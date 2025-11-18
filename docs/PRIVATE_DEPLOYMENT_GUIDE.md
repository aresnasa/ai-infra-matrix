# AI-Infra-Matrix 私有环境部署指南

## 🎯 概述

本指南说明如何在私有环境中部署AI-Infra-Matrix，包括配置管理、镜像仓库设置和服务启动。

## 📋 快速开始

### 1. 验证当前配置
```bash
# 检查环境配置文件
./scripts/verify-config.sh .env.prod

# 检查开发环境配置
./scripts/verify-config.sh .env
```

### 2. 生成生产环境强密码（推荐）
```bash
# 自动生成并应用强密码
./scripts/generate-prod-passwords.sh

# 重新验证配置
./scripts/verify-config.sh .env.prod
```

### 3. 配置私有镜像仓库
```bash
# 修改docker-compose.yml使用私有仓库
./build.sh registry harbor.company.com/ai-infra

# 或使用阿里云ACR
./build.sh registry xxx.aliyuncs.com/ai-infra-matrix

# 或使用其他Docker Registry
./build.sh registry registry.company.com/ai-infra
```

### 4. 启动服务
```bash
# 启动所有服务（使用.env.prod配置）
./build.sh start harbor.company.com/ai-infra

# 或分步执行
./build.sh registry harbor.company.com/ai-infra
./build.sh start
```

### 5. 验证部署
```bash
# 检查服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f
```

## 🔧 详细配置

### 环境文件说明

#### .env (开发环境)
- 用于本地开发
- 包含开发友好的默认值
- 密码相对简单

#### .env.prod (生产环境)
- 用于生产部署
- 需要配置强密码
- 支持私有仓库配置

### 关键配置项

#### 数据库配置
```bash
POSTGRES_DB=ai-infra-matrix
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<强密码>
```

#### Redis配置
```bash
REDIS_PASSWORD=<强密码>
```

#### JupyterHub配置
```bash
JUPYTERHUB_ADMIN_USERS=admin
JUPYTERHUB_CRYPT_KEY=<64字符hex密钥>
JUPYTERHUB_MEM_LIMIT=2G
JUPYTERHUB_CPU_LIMIT=1.0
```

#### Gitea配置
```bash
GITEA_ADMIN_USER=admin
GITEA_ADMIN_PASSWORD=<强密码>
GITEA_BASE_URL=http://gitea:3000
GITEA_DB_PASSWD=<强密码>
```

#### LDAP配置
```bash
LDAP_ADMIN_PASSWORD=<强密码>
LDAP_CONFIG_PASSWORD=<强密码>
```

## 🏗️ build.sh 命令参考

### 基本命令
```bash
# 显示帮助
./build.sh help

# 修改镜像仓库
./build.sh registry <registry_url> [tag]

# 启动服务
./build.sh start [registry_url] [tag]

# 停止服务
./build.sh stop

# 恢复原始配置
./build.sh restore
```

### 镜像管理
```bash
# 拉取所有镜像
./build.sh pull harbor.company.com/ai-infra

# 推送所有镜像
./build.sh push harbor.company.com/ai-infra

# 查看镜像映射
./build.sh images harbor.company.com/ai-infra
```

### 支持的仓库格式
- **Harbor**: `harbor.company.com/ai-infra`
- **阿里云ACR**: `xxx.aliyuncs.com/ai-infra-matrix`
- **Docker Registry**: `registry.company.com/project`
- **Docker Hub**: `docker.io/username`

## 🔍 故障排除

### 常见问题

#### 1. 环境变量未设置
```bash
# 错误: The "IMAGE_TAG" variable is not set
# 解决: 检查环境文件配置
./scripts/verify-config.sh .env.prod
```

#### 2. 密码过于简单
```bash
# 解决: 生成强密码
./scripts/generate-prod-passwords.sh
```

#### 3. 镜像拉取失败
```bash
# 检查镜像是否存在
docker pull harbor.company.com/ai-infra/postgres:15-alpine

# 检查Docker登录状态
docker login harbor.company.com
```

#### 4. 服务启动失败
```bash
# 查看详细日志
docker-compose logs <service_name>

# 检查网络连接
docker network ls
```

### 调试命令
```bash
# 验证Docker Compose配置
docker-compose config

# 检查特定服务
docker-compose ps <service_name>

# 查看实时日志
docker-compose logs -f <service_name>
```

## 🚀 生产环境部署最佳实践

### 1. 安全配置
- ✅ 使用强密码（运行`./scripts/generate-prod-passwords.sh`）
- ✅ 配置HTTPS（如需要）
- ✅ 限制网络访问
- ✅ 定期备份数据

### 2. 监控配置
- ✅ 配置日志收集
- ✅ 设置健康检查
- ✅ 监控资源使用

### 3. 备份策略
- ✅ 数据库定期备份
- ✅ 配置文件备份
- ✅ 镜像版本管理

## 📁 相关文件

- `build.sh` - 主构建和部署脚本
- `.env` - 开发环境配置
- `.env.prod` - 生产环境配置
- `scripts/verify-config.sh` - 配置验证脚本
- `scripts/generate-prod-passwords.sh` - 密码生成脚本
- `docker-compose.yml` - 服务编排配置

## 🆘 获取帮助

如果遇到问题，请：
1. 运行配置验证脚本
2. 检查服务日志
3. 确认网络连接
4. 验证镜像可用性

```bash
# 一键诊断
./scripts/verify-config.sh .env.prod
docker-compose config --quiet
docker-compose ps
```
