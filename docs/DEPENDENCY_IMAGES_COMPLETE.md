# AI-Infra-Matrix 依赖镜像完整列表

## 📋 依赖镜像配置总结

### ✅ 已完成的增强功能

经过分析整个项目的Dockerfile和Helm Chart配置，已将所有必要的依赖镜像补充到build.sh中，确保了完整的镜像推送支持。

### 🗂️ 完整依赖镜像分类

#### 1. 数据库和存储服务
| 镜像 | 版本 | 用途 | 配置位置 |
|------|------|------|----------|
| `postgres:15-alpine` | 15-alpine | PostgreSQL数据库 | docker-compose.yml, Helm Chart |
| `redis:7-alpine` | 7-alpine | Redis缓存 | docker-compose.yml, Helm Chart |
| `minio/minio:latest` | latest | 对象存储服务 | docker-compose.yml, Helm Chart |

#### 2. 认证和管理服务
| 镜像 | 版本 | 用途 | 配置位置 |
|------|------|------|----------|
| `osixia/openldap:stable` | stable | LDAP认证服务 | docker-compose.yml, Helm Chart |
| `osixia/phpldapadmin:stable` | stable | LDAP管理界面 | docker-compose.yml, Helm Chart |
| `gitea/gitea:1.25.1` | 1.24.5 | Git仓库基础镜像 | src/gitea/Dockerfile |

#### 3. 构建时依赖镜像
| 镜像 | 版本 | 用途 | Dockerfile位置 |
|------|------|------|----------------|
| `node:22-alpine` | 22-alpine | 前端构建环境 | src/frontend/Dockerfile |
| `golang:1.25-alpine` | 1.25-alpine | 后端构建环境 | src/backend/Dockerfile |
| `python:3.13-alpine` | 3.13-alpine | JupyterHub和SaltStack构建 | src/jupyterhub/Dockerfile, src/saltstack/Dockerfile |
| `jupyter/base-notebook:latest` | latest | SingleUser基础镜像 | src/singleuser/Dockerfile |

#### 4. 运行时和代理服务
| 镜像 | 版本 | 用途 | 配置位置 |
|------|------|------|----------|
| `nginx:1.27-alpine` | 1.27-alpine | 通用Nginx服务 | docker-compose.yml |
| `nginx:stable-alpine-perl` | stable-alpine-perl | 前端运行时, 自定义Nginx | src/frontend/Dockerfile, src/nginx/Dockerfile |
| `tecnativa/tcp-proxy` | latest | TCP代理服务 | docker-compose.yml |

#### 5. 开发和测试工具
| 镜像 | 版本 | 用途 | 配置位置 |
|------|------|------|----------|
| `redislabs/redisinsight:latest` | latest | Redis可视化管理工具 | docker-compose.yml |

### 🚀 构建脚本增强功能

#### 新增的推送函数

1. **`push_dependencies()`** - 推送所有依赖镜像
   ```bash
   ./build.sh deps-push <registry> [tag]
   ```

2. **`push_production_dependencies()`** - 推送生产环境依赖（排除开发工具）
   ```bash
   ./build.sh prod-deps-push <registry> [tag]
   ```

3. **`push_build_dependencies()`** - 推送构建依赖镜像
   ```bash
   ./build.sh build-deps-push <registry> [tag]
   ```

#### 依赖镜像管理命令

```bash
# 推送所有依赖镜像到Docker Hub
./build.sh deps-push docker.io/youruser v0.3.5

# 推送所有依赖镜像到阿里云ACR
./build.sh deps-push xxx.aliyuncs.com/ai-infra-matrix v0.3.5

# 推送生产环境依赖（不含开发工具）
./build.sh prod-deps-push your-registry.com/ai-infra v1.0.0

# 推送构建依赖镜像
./build.sh build-deps-push your-registry.com/ai-infra v1.0.0

# 拉取、标记并推送所有依赖（一键操作）
./build.sh deps-all your-registry.com/ai-infra v1.0.0
```

### 🔍 验证和测试

#### 1. 验证依赖镜像列表
```bash
# 查看所有依赖镜像
source build.sh && get_all_dependencies

# 查看生产环境依赖
source build.sh && get_production_dependencies
```

#### 2. 验证镜像推送配置
```bash
# 测试镜像映射（不实际推送）
./build.sh verify your-registry.com/ai-infra v1.0.0
```

### 📊 推送统计

- **总计**: 14个依赖镜像
- **数据库/存储**: 3个镜像 (PostgreSQL, Redis, MinIO)
- **认证/管理**: 3个镜像 (OpenLDAP, phpLDAPadmin, Gitea)
- **构建依赖**: 4个镜像 (Node.js, Go, Python, Jupyter)
- **运行时服务**: 3个镜像 (Nginx变体, TCP代理)
- **开发工具**: 1个镜像 (RedisInsight)

### 🎯 使用场景

#### CI/CD流水线推送
```bash
# 步骤1: 构建所有AI-Infra服务
./build.sh build-all v1.2.0

# 步骤2: 推送AI-Infra自研镜像  
./build.sh push-all harbor.example.com/ai-infra v1.2.0

# 步骤3: 推送所有依赖镜像
./build.sh deps-push harbor.example.com/ai-infra v1.2.0
```

#### 生产环境镜像准备
```bash
# 推送生产环境必需的依赖镜像（排除开发工具）
./build.sh prod-deps-push your-production-registry.com/ai-infra v1.0.0
```

#### 离线环境镜像迁移
```bash
# 准备所有镜像到内网仓库
./build.sh deps-all internal-harbor.company.com/ai-infra v1.0.0
```

### ⚠️ 重要说明

1. **镜像兼容性**: 所有依赖镜像版本已与Helm Chart保持一致
2. **构建优化**: 构建依赖镜像支持多阶段构建优化
3. **注册表适配**: 支持Docker Hub、Harbor、阿里云ACR等多种注册表
4. **版本管理**: 所有依赖镜像支持统一标签管理

---

**✅ 依赖镜像补充完成 - PostgreSQL、Redis及所有必要依赖已全面支持！**

生成时间: 2025-08-28
状态: 已验证并可用于生产环境 🚀
