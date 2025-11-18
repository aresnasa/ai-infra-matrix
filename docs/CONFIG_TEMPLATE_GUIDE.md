# AI Infrastructure Matrix 配置模板系统

## 概述

该配置模板系统允许通过 `build.sh` 统一管理 nginx 和 JupyterHub 的配置文件，支持不同环境和认证方式的配置生成。

## 目录结构

```
src/
├── nginx/
│   ├── templates/              # nginx配置模板
│   │   ├── server-main.conf.template
│   │   ├── jupyterhub.conf.template
│   │   ├── gitea.conf.template
│   │   └── production-security.conf.template
│   └── conf.d/                 # 生成的nginx配置文件
│       ├── server-main.conf
│       └── includes/
│           ├── jupyterhub.conf
│           └── gitea.conf
└── jupyterhub/
    ├── templates/              # JupyterHub配置模板
    │   ├── jupyterhub_config.py.template
    │   ├── spawner-docker.py.template
    │   ├── spawner-kubernetes.py.template
    │   ├── auth-local.py.template
    │   ├── auth-ldap.py.template
    │   ├── auth-oauth.py.template
    │   └── production-config.py.template
    └── jupyterhub_config_*.py  # 生成的配置文件
```

## 命令用法

### 1. 生成所有配置文件

```bash
./build.sh generate-configs [environment] [domain] [auth_type]
```

**参数说明:**
- `environment`: 环境类型 (development, production, test) - 默认: development
- `domain`: 域名 - 默认: localhost  
- `auth_type`: 认证类型 (local, ldap, oauth) - 默认: local

**示例:**
```bash
# 开发环境，本地认证
./build.sh generate-configs development ai-infra.local local

# 生产环境，LDAP认证
./build.sh generate-configs production ai-infra.company.com ldap

# 测试环境，OAuth认证
./build.sh generate-configs test test.ai-infra.com oauth
```

### 2. 单独生成 nginx 配置

```bash
./build.sh generate-nginx [environment] [domain]
```

**示例:**
```bash
./build.sh generate-nginx production ai-infra.example.com
```

### 3. 单独生成 JupyterHub 配置

```bash
./build.sh generate-jupyterhub [environment] [auth_type]
```

**示例:**
```bash
# Docker环境，LDAP认证
./build.sh generate-jupyterhub development ldap

# Kubernetes环境，OAuth认证  
./build.sh generate-jupyterhub kubernetes oauth
```

## 配置选项

### 环境类型 (Environment)

- **development**: 开发环境，基础配置
- **production**: 生产环境，包含安全头、压缩等优化配置
- **test**: 测试环境，与开发环境类似
- **kubernetes**: Kubernetes部署环境，使用KubeSpawner

### 认证类型 (Authentication)

- **local**: 本地认证，使用JupyterHub内置用户管理
- **ldap**: LDAP认证，连接到OpenLDAP服务器
- **oauth**: OAuth认证，支持第三方OAuth提供商

### 域名配置

可以指定自定义域名，生成的nginx配置会相应调整 `server_name` 指令。

## 模板变量

模板文件使用 `{{变量名}}` 格式的占位符，支持以下变量：

### nginx模板变量:
- `{{ENVIRONMENT}}`: 环境类型
- `{{DOMAIN}}`: 域名
- `{{PRODUCTION_SECURITY}}`: 生产环境安全配置

### JupyterHub模板变量:
- `{{ENVIRONMENT}}`: 环境类型
- `{{AUTH_TYPE}}`: 认证类型
- `{{GENERATION_TIME}}`: 生成时间
- `{{SPAWNER_CONFIG}}`: Spawner配置
- `{{AUTH_CONFIG}}`: 认证配置
- `{{PRODUCTION_CONFIG}}`: 生产环境配置

## 自定义模板

如需自定义配置，可以直接编辑 `src/nginx/templates/` 和 `src/jupyterhub/templates/` 目录下的模板文件。

模板文件遵循原始配置文件的语法，使用 `{{变量名}}` 标记需要替换的内容。

## 最佳实践

1. **开发环境**: 使用默认配置进行快速开发
   ```bash
   ./build.sh generate-configs
   ```

2. **生产环境**: 指定域名和认证方式
   ```bash
   ./build.sh generate-configs production your-domain.com ldap
   ./build.sh generate-passwords .env.prod --force
   ```

3. **Kubernetes部署**: 使用kubernetes环境类型
   ```bash
   ./build.sh generate-jupyterhub kubernetes ldap
   ```

4. **版本控制**: 建议将模板文件纳入版本控制，生成的配置文件可根据需要选择是否纳入版本控制。

## 故障排除

1. **模板文件不存在**: 确保 `src/nginx/templates/` 和 `src/jupyterhub/templates/` 目录及其模板文件存在
2. **Python依赖**: 配置生成依赖Python3，确保系统已安装
3. **权限问题**: 确保对目标目录有写权限

## 集成到构建流程

配置生成可以集成到CI/CD流程中：

```bash
# 1. 生成配置
./build.sh generate-configs production your-domain.com ldap

# 2. 生成密码
./build.sh generate-passwords .env.prod --force

# 3. 构建镜像
./build.sh build-all v1.0.0

# 4. 推送到仓库
./build.sh push-all your-registry.com/ai-infra v1.0.0
```
