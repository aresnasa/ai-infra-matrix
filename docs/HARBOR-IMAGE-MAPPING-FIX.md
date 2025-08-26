# ✅ Harbor镜像映射修复完成

## 🎯 问题描述

原始的镜像映射配置将基础镜像映射到了不同的Harbor项目：
- ❌ PostgreSQL -> `aiharbor.msxf.local/library/postgres:v0.3.5`
- ❌ Redis -> `aiharbor.msxf.local/library/redis:v0.3.5`  
- ❌ MinIO -> `aiharbor.msxf.local/minio/minio:v0.3.5`

这导致镜像拉取失败，因为Harbor中只有 `aihpc` 项目。

## 🔧 修复内容

更新了 `config/image-mapping.conf` 文件，将所有镜像统一映射到 `aihpc` 项目：

### 修复前后对比

| 镜像 | 修复前 | 修复后 |
|------|--------|---------|
| PostgreSQL | `aiharbor.msxf.local/library/postgres:v0.3.5` | `aiharbor.msxf.local/aihpc/postgres:v0.3.5` |
| Redis | `aiharbor.msxf.local/library/redis:v0.3.5` | `aiharbor.msxf.local/aihpc/redis:v0.3.5` |
| MinIO | `aiharbor.msxf.local/minio/minio:v0.3.5` | `aiharbor.msxf.local/aihpc/minio:v0.3.5` |
| Nginx | `aiharbor.msxf.local/aihpc/nginx:v0.3.5` | ✅ 保持不变 |

### 修改的配置

```diff
# === 基础镜像映射 ===
- postgres:15-alpine|library|v0.3.5
- postgres:latest|library|v0.3.5
+ postgres:15-alpine|aihpc|v0.3.5
+ postgres:latest|aihpc|v0.3.5

- redis:7-alpine|library|v0.3.5
- redis:latest|library|v0.3.5
+ redis:7-alpine|aihpc|v0.3.5
+ redis:latest|aihpc|v0.3.5

- quay.io/minio/minio:latest|minio|v0.3.5
- minio/minio:latest|minio|v0.3.5
+ quay.io/minio/minio:latest|aihpc|v0.3.5
+ minio/minio:latest|aihpc|v0.3.5
```

## ✅ 验证结果

重新生成生产配置后，所有镜像现在都正确映射到 `aiharbor.msxf.local/aihpc/` 命名空间：

```yaml
# 基础服务镜像
postgres: aiharbor.msxf.local/aihpc/postgres:v0.3.5
redis: aiharbor.msxf.local/aihpc/redis:v0.3.5
minio: aiharbor.msxf.local/aihpc/minio:v0.3.5
nginx: aiharbor.msxf.local/aihpc/nginx:v0.3.5

# 项目镜像
ai-infra-backend: aiharbor.msxf.local/aihpc/ai-infra-backend:v0.3.5
ai-infra-frontend: aiharbor.msxf.local/aihpc/ai-infra-frontend:v0.3.5
ai-infra-jupyterhub: aiharbor.msxf.local/aihpc/ai-infra-jupyterhub:v0.3.5
# ... 等等
```

## 🚀 下一步

现在可以正常使用生产环境部署命令：

```bash
# 确保所有镜像已推送到Harbor
./build.sh deps-all aiharbor.msxf.local/aihpc v0.3.5
./build.sh build-push aiharbor.msxf.local/aihpc v0.3.5

# 启动生产环境
./build.sh prod-up aiharbor.msxf.local/aihpc v0.3.5
```

## 📋 修复的文件

- ✅ `config/image-mapping.conf` - 更新镜像映射配置
- ✅ `docker-compose.prod.yml` - 重新生成（包含正确的镜像映射）

现在所有镜像都符合预期的Harbor项目结构！
