# AI-Infra Matrix 生产环境测试部署指南

## 概述

本指南介绍如何使用重新标记的 `aiharbor.msxf.local/aihpc` 镜像进行生产环境测试部署。

## 文件结构

### 新增文件

1. **`scripts/retag-images-for-prod.sh`** - 镜像重新标记脚本
2. **`docker-compose.prod-test.yml`** - 生产测试环境配置
3. **`.env.prod-test`** - 生产测试环境变量
4. **`scripts/start-prod-test.sh`** - 快速启动脚本
5. **`docs/PROD_TEST_DEPLOYMENT_GUIDE.md`** - 本文档

### 修改功能

- 基于 `docker-compose.prod.yml` 结构重新设计
- 移除 LDAP 相关服务（简化测试环境）
- 统一镜像标签为 `aiharbor.msxf.local/aihpc` 格式
- 优化健康检查和依赖关系

## 快速开始

### 1. 镜像重新标记

```bash
# 查看当前镜像
./scripts/retag-images-for-prod.sh --list

# 预览重新标记操作
./scripts/retag-images-for-prod.sh --dry-run

# 执行镜像重新标记（主要镜像）
./scripts/retag-images-for-prod.sh

# 包含依赖镜像的重新标记
./scripts/retag-images-for-prod.sh --deps
```

### 2. 启动生产测试环境

```bash
# 使用快速启动脚本
./scripts/start-prod-test.sh start

# 或者使用 docker-compose 直接启动
docker-compose -f docker-compose.prod-test.yml --env-file .env.prod-test up -d
```

### 3. 查看服务状态

```bash
# 查看服务状态
./scripts/start-prod-test.sh status

# 检查健康状态
./scripts/start-prod-test.sh health

# 显示访问地址
./scripts/start-prod-test.sh urls
```

## 服务访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 主界面 (Nginx) | http://localhost:8080 | 统一入口 |
| 后端API | http://localhost:8082 | 后端服务 |
| 前端 (直接) | http://localhost:3000 | 前端服务 |
| JupyterHub | http://localhost:8088 | Jupyter服务 |
| Gitea | http://localhost:3010 | 代码仓库 |
| Gitea (调试) | http://localhost:3011 | 调试代理 |
| MinIO控制台 | http://localhost:9001 | 对象存储 |
| Redis Insight | http://localhost:8001 | Redis管理 |

## 默认凭据

### 管理员账户
- **用户名**: `admin`
- **密码**: `admin123prod`

### MinIO
- **Access Key**: `minioadmin_prod`
- **Secret Key**: `minioadmin_prod_2024_secure`

### 数据库
- **数据库**: `ai_infra_prod_test`
- **用户**: `postgres`
- **密码**: `ai_infra_prod_pass_2024`

## 镜像映射

### 主要服务镜像

| 原镜像 | 目标镜像 |
|-------|----------|
| ai-infra-backend:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-backend:v0.3.5 |
| ai-infra-frontend:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-frontend:v0.3.5 |
| ai-infra-nginx:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-nginx:v0.3.5 |
| ai-infra-jupyterhub:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-jupyterhub:v0.3.5 |
| ai-infra-gitea:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-gitea:v0.3.5 |
| ai-infra-singleuser:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-singleuser:v0.3.5 |
| ai-infra-saltstack:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-saltstack:v0.3.5 |
| ai-infra-backend-init:v0.3.5 | aiharbor.msxf.local/aihpc/ai-infra-backend-init:v0.3.5 |

### 依赖镜像

| 原镜像 | 目标镜像 |
|-------|----------|
| postgres:15-alpine | aiharbor.msxf.local/library/postgres:v0.3.5 |
| redis:7-alpine | aiharbor.msxf.local/library/redis:v0.3.5 |
| nginx:1.27-alpine | aiharbor.msxf.local/library/nginx:v0.3.5 |
| minio/minio:latest | aiharbor.msxf.local/minio/minio:v0.3.5 |
| tecnativa/tcp-proxy:latest | aiharbor.msxf.local/aihpc/tcp-proxy:v0.3.5 |
| redislabs/redisinsight:latest | aiharbor.msxf.local/aihpc/redisinsight:v0.3.5 |

## 脚本使用指南

### 重新标记脚本

```bash
# 帮助信息
./scripts/retag-images-for-prod.sh --help

# 自定义标签
./scripts/retag-images-for-prod.sh --tag v0.4.0

# 自定义仓库
./scripts/retag-images-for-prod.sh --registry my-registry.com --namespace myproject
```

### 启动脚本

```bash
# 查看帮助
./scripts/start-prod-test.sh --help

# 启动服务
./scripts/start-prod-test.sh start

# 停止服务
./scripts/start-prod-test.sh stop

# 重启服务
./scripts/start-prod-test.sh restart

# 查看日志（跟随）
./scripts/start-prod-test.sh logs -f

# 查看特定服务日志
./scripts/start-prod-test.sh logs -s backend

# 清理环境
./scripts/start-prod-test.sh clean

# 重新构建（包含重新标记）
./scripts/start-prod-test.sh rebuild --retag
```

## 环境变量配置

### 关键配置项

- `POSTGRES_PASSWORD`: 数据库密码
- `REDIS_PASSWORD`: Redis密码
- `JWT_SECRET`: JWT密钥
- `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY`: MinIO凭据
- `GITEA_ADMIN_TOKEN`: Gitea管理令牌

### 特性开关

- `ENABLE_LDAP_AUTH=false`: 禁用LDAP认证
- `ENABLE_LOCAL_AUTH=true`: 启用本地认证
- `ENABLE_REGISTRATION=false`: 禁用自注册
- `DEBUG_MODE=false`: 生产模式

## 与原版配置的主要差异

### 移除的服务
- LDAP (openldap)
- LDAP Admin (phpldapadmin)

### 修改的配置
- 容器名称添加 `-prod-test` 后缀
- 网络名称更改为 `ai-infra-network-prod-test`
- 数据卷名称添加 `-prod-test` 后缀
- 端口映射调整避免冲突

### 新增功能
- 完整的健康检查配置
- 统一的环境变量管理
- 自动化脚本支持

## 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 检查端口占用
   netstat -tulpn | grep :8080
   
   # 修改 .env.prod-test 中的端口配置
   ```

2. **镜像不存在**
   ```bash
   # 重新标记镜像
   ./scripts/retag-images-for-prod.sh --deps
   ```

3. **服务启动失败**
   ```bash
   # 查看详细日志
   ./scripts/start-prod-test.sh logs -s <service-name>
   
   # 检查健康状态
   ./scripts/start-prod-test.sh health
   ```

4. **数据卷问题**
   ```bash
   # 清理并重新启动
   ./scripts/start-prod-test.sh clean
   ./scripts/start-prod-test.sh start
   ```

### 日志位置

- Nginx日志: `./logs/nginx/`
- 后端输出: `./src/backend/outputs/`
- 后端上传: `./src/backend/uploads/`
- 共享数据: `./shared/`

## 生产部署注意事项

1. **安全配置**
   - 修改所有默认密码
   - 启用 HTTPS
   - 配置防火墙规则

2. **性能优化**
   - 调整资源限制
   - 配置负载均衡
   - 优化数据库连接池

3. **监控告警**
   - 配置健康检查
   - 设置监控指标
   - 建立告警机制

4. **备份策略**
   - 数据库定期备份
   - 配置文件版本控制
   - 镜像版本管理

## 下一步计划

1. **集成测试**: 验证所有服务功能
2. **性能测试**: 压力测试和性能调优
3. **安全加固**: 安全配置和漏洞扫描
4. **文档完善**: 操作手册和故障排除
5. **自动化部署**: CI/CD 流水线集成
