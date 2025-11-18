# Harbor项目创建指南

## 概述

在开始镜像迁移之前，需要在Harbor私有仓库中创建必要的项目。本指南说明如何创建所需的项目结构。

## 必需的Harbor项目

根据AI Infrastructure Matrix的镜像需求，需要创建以下项目：

### 1. 主项目：aihpc
- **用途**: 存储AI Infrastructure Matrix的源码镜像
- **包含镜像**: 
  - `ai-infra-backend:v0.3.5`
  - `ai-infra-frontend:v0.3.5`
  - `ai-infra-jupyterhub:v0.3.5`
  - `ai-infra-singleuser:v0.3.5`
  - `ai-infra-saltstack:v0.3.5`
  - `ai-infra-nginx:v0.3.5`
  - `ai-infra-gitea:v0.3.5`
  - `ai-infra-backend-init:v0.3.5`

### 2. 基础镜像项目：library
- **用途**: 存储官方Docker Hub基础镜像
- **包含镜像**:
  - `postgres:15-alpine`
  - `redis:7-alpine`
  - `nginx:1.27-alpine`

### 3. 第三方项目：tecnativa
- **用途**: 存储tecnativa组织的镜像
- **包含镜像**:
  - `tcp-proxy:latest`

### 4. 第三方项目：redislabs
- **用途**: 存储redislabs组织的镜像
- **包含镜像**:
  - `redisinsight:latest`

### 5. 第三方项目：minio
- **用途**: 存储minio组织的镜像
- **包含镜像**:
  - `minio:latest`

## 创建步骤

### 通过Harbor Web UI创建

1. **登录Harbor管理界面**
   ```
   https://aiharbor.msxf.local
   ```

2. **创建项目**
   - 点击左侧导航栏的"项目"
   - 点击"新建项目"按钮
   - 按照下表创建项目：

   | 项目名称 | 访问级别 | 描述 |
   |---------|---------|------|
   | aihpc | 私有 | AI Infrastructure Matrix主项目 |
   | library | 私有 | 官方基础镜像 |
   | tecnativa | 私有 | Tecnativa组织镜像 |
   | redislabs | 私有 | RedisLabs组织镜像 |
   | minio | 私有 | MinIO组织镜像 |

### 通过Harbor API创建（可选）

如果有API访问权限，可以使用脚本批量创建：

```bash
#!/bin/bash
# Harbor项目批量创建脚本

HARBOR_URL="https://aiharbor.msxf.local"
HARBOR_USER="admin"
HARBOR_PASSWORD="your_password"

# 项目列表
projects=("aihpc" "library" "tecnativa" "redislabs" "minio")

echo "创建Harbor项目..."

for project in "${projects[@]}"; do
    echo "创建项目: $project"
    
    curl -X POST "$HARBOR_URL/api/v2.0/projects" \
        -H "Content-Type: application/json" \
        -u "$HARBOR_USER:$HARBOR_PASSWORD" \
        -d "{
            \"project_name\": \"$project\",
            \"public\": false,
            \"metadata\": {
                \"auto_scan\": \"true\",
                \"enable_content_trust\": \"false\",
                \"prevent_vul\": \"false\",
                \"severity\": \"low\"
            }
        }"
    
    if [ $? -eq 0 ]; then
        echo "✓ 项目 $project 创建成功"
    else
        echo "✗ 项目 $project 创建失败"
    fi
    echo
done

echo "所有项目创建完成！"
```

## 验证项目创建

创建完成后，可以在Harbor Web UI中验证：

1. 登录Harbor管理界面
2. 点击左侧"项目"
3. 确认看到以下5个项目：
   - ✅ aihpc
   - ✅ library
   - ✅ tecnativa
   - ✅ redislabs
   - ✅ minio

## 权限配置

确保用于推送镜像的用户账号具有以下权限：

- **所有项目**: 推送镜像权限（Guest以上级别）
- **建议**: 为CI/CD用户分配"开发者"角色

## 故障排除

### 常见问题

1. **项目已存在错误**
   ```
   Error: project name aihpc already exists
   ```
   **解决方案**: 项目已存在，可以直接使用。

2. **权限不足错误**
   ```
   Error: insufficient privilege
   ```
   **解决方案**: 使用管理员账号创建项目，或联系Harbor管理员。

3. **网络连接错误**
   ```
   Error: failed to connect to Harbor
   ```
   **解决方案**: 检查Harbor服务状态和网络连接。

## 下一步

项目创建完成后，可以继续执行镜像迁移：

```bash
# 1. 迁移基础镜像
./scripts/migrate-base-images.sh aiharbor.msxf.local/aihpc

# 2. 构建源码镜像
./build.sh build-push aiharbor.msxf.local/aihpc v0.3.5

# 3. 验证镜像
./scripts/verify-private-images.sh aiharbor.msxf.local/aihpc v0.3.5
```
