# build.sh 硬编码版本清理总结

## 执行时间
2025年1月

## 清理目标
清理 `build.sh` 中所有与版本相关的硬编码函数，确保所有版本都通过 `.env` 文件进行统一管理。

## 发现的硬编码位置

通过 `grep` 搜索发现了 50+ 处硬编码版本引用，主要分布在：

### 1. 核心函数
- ✅ **Line 191-220**: `get_dependency_images_list()` - 新增的依赖镜像列表生成函数
- ✅ **Line 1012**: `get_all_dependencies()` - 现在调用 `get_dependency_images_list()`
- ✅ **Line 1030-1036**: `get_production_dependencies()` - 生产环境依赖镜像
- ✅ **Line 1118-1124**: `collect_dependency_images()` - 依赖镜像收集函数

### 2. Mock 测试配置
- ✅ **Lines 1231-1232**: Mock 数据测试镜像变量
  ```bash
  MOCK_POSTGRES_IMAGE="postgres:${POSTGRES_VERSION:-15-alpine}"
  MOCK_REDIS_IMAGE="redis:${REDIS_VERSION:-7-alpine}"
  ```

### 3. 镜像推送和拉取函数
- ✅ **Line 7152-7158**: `push_build_dependencies()` - 构建依赖推送数组
- ✅ **Line 7278**: 依赖镜像回退列表
- ✅ **Line 7792-7802**: 镜像完整性检查的依赖数组
- ✅ **Line 7890-7900**: Compose 文件镜像替换的依赖数组

### 4. 私有仓库映射
- ✅ **Lines 8096-8121**: `replace_images_in_compose_file()` - Docker Compose 镜像替换映射
  - 包含 Kafka, Postgres, Redis, Nginx, Gitea, Jupyter 等所有镜像的版本映射

### 5. 统一标记函数
- ✅ **Lines 8458-8473**: `unified_tag_images()` - 依赖镜像标记数组

### 6. 验证和导出函数
- ✅ **Lines 8905-8912**: `verify_images()` - 镜像验证的基础镜像列表
- ✅ **Lines 9008-9010**: `quick_verify_images()` - 快速验证关键镜像
- ✅ **Lines 9916-9927**: `export_offline_images()` - 离线导出基础依赖
- ✅ **Lines 10191-10206**: `push_to_internal_registry()` - 推送到内部仓库

### 7. 帮助文档
- ✅ **Lines 9551-9553**: AppHub 默认版本说明
  ```bash
  --slurm-version=<ver>    - 指定 SLURM 版本 (默认值见 .env)
  --salt-version=<ver>     - 指定 SaltStack 版本 (默认值见 .env)
  --categraf-version=<ver> - 指定 Categraf 版本 (默认值见 .env)
  ```
- ✅ **Line 9557**: 版本示例从具体数字改为占位符

## 修复模式

### 统一替换模式
所有硬编码版本都使用以下模式替换：

```bash
# 修复前
"postgres:15-alpine"
"redis:7-alpine"
"node:22-alpine"
"golang:1.25-alpine"
"python:3.13-alpine"
"gitea/gitea:1.25.1"

# 修复后
"postgres:${POSTGRES_VERSION:-15-alpine}"
"redis:${REDIS_VERSION:-7-alpine}"
"node:${NODE_ALPINE_VERSION:-22-alpine}"
"golang:${GOLANG_ALPINE_VERSION:-1.25-alpine}"
"python:${PYTHON_ALPINE_VERSION:-3.13-alpine}"
"gitea/gitea:${GITEA_VERSION:-1.25.1}"
```

### 函数中添加环境加载
所有使用版本变量的函数都添加了 `load_env_file` 调用：

```bash
function_name() {
    load_env_file  # 确保环境变量已加载
    local images=(
        "postgres:${POSTGRES_VERSION:-15-alpine}"
        ...
    )
}
```

## 涉及的环境变量

### 基础镜像版本（60+ 个）
详见 `.env` 和 `.env.example` 文件，主要包括：

**基础镜像**
- `GOLANG_ALPINE_VERSION=1.25-alpine`
- `NODE_ALPINE_VERSION=22-alpine`
- `PYTHON_ALPINE_VERSION=3.14-alpine`
- `UBUNTU_VERSION=22.04`
- `ROCKYLINUX_VERSION=9`
- `NGINX_VERSION=stable-alpine-perl`
- `NGINX_ALPINE_VERSION=1.27-alpine`

**基础设施服务**
- `POSTGRES_VERSION=15-alpine`
- `REDIS_VERSION=7-alpine`
- `MINIO_VERSION=latest`
- `OPENLDAP_VERSION=stable`
- `PHPLDAPADMIN_VERSION=stable`
- `REDISINSIGHT_VERSION=latest`
- `TCP_PROXY_VERSION=latest`
- `KAFKA_VERSION=7.5.0`
- `KAFKAUI_VERSION=latest`

**应用版本**
- `GITEA_VERSION=1.25.1`
- `SALTSTACK_VERSION=3007.8`
- `SLURM_VERSION=25.05.4`
- `CATEGRAF_VERSION=v0.4.22`
- `SINGULARITY_VERSION=v4.3.4`
- `JUPYTER_BASE_NOTEBOOK_VERSION=latest`

## 验证结果

### 硬编码清除验证
```bash
# 最终验证（排除注释和文档）
grep -E '"(postgres:15|redis:7|nginx:1\.27|node:22|golang:1\.25|python:3\.1[34]|gitea/gitea:1\.25)' build.sh
# 结果: No matches found ✅
```

### 剩余硬编码
仅剩注释中的示例（不影响功能）：
- Line 4764: 注释示例 `# 简单镜像名 (如 redis:7-alpine, postgres:15-alpine)`
- Line 5139: 注释示例 `# 例如：aiharbor.msxf.local/aihpc/redis:7-alpine → redis:7-alpine`

这些是文档性质的注释，不影响实际构建。

## 影响的函数列表

### 直接修复的函数（14 个）
1. `get_dependency_images_list()` - ✅ 新增
2. `get_all_dependencies()` - ✅ 修复
3. `get_production_dependencies()` - ✅ 修复
4. `collect_dependency_images()` - ✅ 修复
5. `push_build_dependencies()` - ✅ 修复
6. `pull_from_harbor()` - ✅ 修复（依赖列表回退）
7. `list_all_images()` - ✅ 修复（依赖数组）
8. `replace_images_in_compose_file()` - ✅ 修复（映射数组）
9. `unified_tag_images()` - ✅ 修复（依赖数组）
10. `verify_images()` - ✅ 修复（基础镜像数组）
11. `quick_verify_images()` - ✅ 修复（关键镜像）
12. `export_offline_images()` - ✅ 修复（依赖数组 + Kafka）
13. `push_to_internal_registry()` - ✅ 修复（依赖数组 + Kafka）
14. `show_help()` - ✅ 修复（版本说明和示例）

### 间接受益的函数
所有调用以上函数的其他函数都自动受益于版本统一管理。

## 修复统计

| 类型 | 数量 | 状态 |
|------|------|------|
| 核心依赖函数 | 4 | ✅ 完成 |
| 镜像推送/拉取函数 | 3 | ✅ 完成 |
| 私有仓库映射 | 1 | ✅ 完成 |
| 统一标记函数 | 1 | ✅ 完成 |
| 验证和导出函数 | 4 | ✅ 完成 |
| Mock 测试配置 | 1 | ✅ 完成 |
| 帮助文档 | 1 | ✅ 完成 |
| **总计** | **15** | **✅ 全部完成** |

## 向后兼容性

### 默认值保护
所有环境变量都使用 `${VAR:-default}` 语法，确保：
- ✅ 如果 `.env` 不存在，使用默认值
- ✅ 如果变量未设置，使用默认值
- ✅ 旧的构建脚本无需修改即可继续工作

### 环境文件优先级
```bash
1. 用户设置的环境变量（最高优先级）
2. .env 文件中的值
3. ${VAR:-default} 中的默认值（最低优先级）
```

## 测试建议

### 1. 基本功能测试
```bash
# 测试环境变量加载
source build.sh
load_env_file
echo $POSTGRES_VERSION  # 应输出: 15-alpine

# 测试依赖列表生成
get_dependency_images_list

# 测试版本参数生成
get_version_build_args backend
```

### 2. 构建测试
```bash
# 测试单个服务构建（使用 .env 版本）
./build.sh build backend

# 测试版本覆盖
POSTGRES_VERSION=16-alpine ./build.sh build backend

# 测试完整构建
./build.sh build-all
```

### 3. 镜像操作测试
```bash
# 测试依赖镜像拉取
./build.sh deps-pull

# 测试镜像验证
./build.sh verify <registry> <tag>

# 测试离线导出
./build.sh export-offline /tmp/offline-images
```

## 后续维护建议

### 1. 新增组件时
当添加新的第三方依赖镜像时：
1. 在 `.env.example` 中添加对应的版本变量
2. 更新 `get_dependency_images_list()` 函数
3. 如果需要私有仓库映射，更新 `replace_images_in_compose_file()` 中的映射数组
4. 测试构建和推送流程

### 2. 版本升级时
升级第三方依赖版本：
1. 编辑 `.env` 文件中的版本变量
2. 运行 `./build.sh clean` 清理旧镜像（可选）
3. 运行 `./build.sh build-all` 重新构建
4. 验证所有服务正常工作

### 3. 文档同步
- 保持 `.env.example` 与实际使用的版本同步
- 更新 `VERSION_MANAGEMENT_GUIDE.md` 文档
- 在 `show_help()` 中保持准确的版本说明

## 相关文档

- `VERSION_MANAGEMENT_GUIDE.md` - 版本管理使用指南
- `VERSION_MANAGEMENT_IMPLEMENTATION_SUMMARY.md` - 实现总结
- `.env.example` - 完整的版本配置模板
- `test-version-args.sh` - 版本参数测试脚本

## 总结

✅ **完成状态**: 所有 50+ 处硬编码版本已全部清理完成  
✅ **向后兼容**: 使用默认值确保兼容性  
✅ **统一管理**: 所有版本通过 .env 文件集中配置  
✅ **灵活性**: 支持环境变量覆盖和命令行参数  
✅ **可维护性**: 版本升级只需修改 .env 文件  

现在 `build.sh` 中所有与版本相关的配置都实现了动态化，真正做到了"统一版本管理"（unified version management）。
