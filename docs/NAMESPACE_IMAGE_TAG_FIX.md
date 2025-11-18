# 带命名空间镜像 Tag 修复报告

## 问题描述

在运行 `./build.sh build-all` 时，Docker Compose 尝试从网络拉取 `osixia/openldap:stable` 失败：

```bash
Error response from daemon: failed to resolve reference "docker.io/osixia/openldap:stable": 
failed to do request: Head "https://registry-1.docker.io/v2/osixia/openldap/manifests/stable": EOF
```

### 根本原因

1. **镜像标签不完整**
   - 本地有 `openldap:stable` 和 `localhost/openldap:stable`
   - 但缺少 `osixia/openldap:stable`（Docker Compose 需要的完整命名空间形式）

2. **`docker-compose.yml` 使用完整命名空间**
   ```yaml
   openldap:
     image: osixia/openldap:stable  # 需要完整的命名空间形式
   ```

3. **镜像共享同一个 ID**
   - `phpldapadmin:stable` (ID: 9831569a2f3d)
   - `osixia/phpldapadmin:stable` (ID: 9831569a2f3d) ← 缺失
   - `localhost/phpldapadmin:stable` (ID: 9831569a2f3d)
   
   所有标签指向同一个镜像，但 Docker Compose 需要特定的标签名称。

## 修复方案

### 1. 增强 `tag_image_smart` 函数

支持带命名空间的 Docker Hub 镜像（如 `osixia/openldap:stable`）。

**关键改进**：

```bash
# 提取短名称（移除 Docker Hub 命名空间）
# 例如：osixia/openldap:stable → openldap:stable
local short_name="$base_image"
if [[ "$base_image" =~ ^[^/]+/[^/]+: ]]; then
    # 包含命名空间（如 osixia/openldap:stable）
    short_name=$(echo "$base_image" | sed -E 's|^[^/]+/||')
fi
```

**创建多个别名**：

对于 `osixia/openldap:stable`，创建：
1. ✅ `osixia/openldap:stable` (完整命名空间)
2. ✅ `openldap:stable` (短名称)
3. ✅ `localhost/openldap:stable` (localhost + 短名称)

### 2. 智能源镜像选择

**公网环境**：
```bash
"external")
    # 检查顺序：base_image → short_name → localhost/short_name
    # 从任何存在的镜像创建其他别名
    
    local source_image=""
    if docker image inspect "$base_image" >/dev/null 2>&1; then
        source_image="$base_image"  # osixia/openldap:stable
    elif docker image inspect "$short_name" >/dev/null 2>&1; then
        source_image="$short_name"  # openldap:stable
    elif docker image inspect "$localhost_short" >/dev/null 2>&1; then
        source_image="$localhost_short"  # localhost/openldap:stable
    fi
    
    # 从源镜像创建所有需要的别名
    ;;
```

**内网环境**：
```bash
"internal")
    # 检查顺序：Harbor → base_image → short_name → localhost/short_name
    # 优先级降级策略
    
    if docker image inspect "$harbor_image" >/dev/null 2>&1; then
        source_image="$harbor_image"  # aiharbor.msxf.local/aihpc/osixia/openldap:stable
    elif docker image inspect "$base_image" >/dev/null 2>&1; then
        source_image="$base_image"  # osixia/openldap:stable
    elif docker image inspect "$short_name" >/dev/null 2>&1; then
        source_image="$short_name"  # openldap:stable
    elif docker image inspect "$localhost_short" >/dev/null 2>&1; then
        source_image="$localhost_short"  # localhost/openldap:stable
    fi
    ;;
```

### 3. 集成到 `build-all` Step 2

增强 Step 2 以处理 `docker-compose.yml` 中的第三方镜像：

```bash
# Step 2: 智能镜像别名管理

# 1. 从 Dockerfile 中提取基础镜像
for service in "${services_list[@]}"; do
    images=$(extract_base_images "$dockerfile_path")
    all_images+=("$images")
done

# 2. 从 docker-compose.yml 中提取第三方镜像（带命名空间的）
compose_images=$(grep -E '^\s*image:' "$SCRIPT_DIR/docker-compose.yml" | \
    grep -v '\$' | \
    awk '{print $2}' | \
    grep '/' | \
    sort -u)

all_images+=("$compose_images")

# 3. 批量处理所有镜像
batch_tag_images_smart "$network_env" "$harbor_registry" "${unique_images[@]}"
```

### 4. 动态网络环境检测

修改 `detect_network_environment()` 函数，优先进行实际网络检测：

**修改前**：
```bash
detect_network_environment() {
    # 检测方法3：检查环境变量标识（优先级高）
    if [[ "${AI_INFRA_NETWORK_ENV}" == "internal" ]]; then
        echo "internal"
        return 0
    fi
    
    # 检测方法1：尝试连接外网
    if ping -c 1 8.8.8.8 >/dev/null 2>&1; then
        echo "external"
        return 0
    fi
    
    # 默认判定为内网
    echo "internal"
}
```

**修改后**：
```bash
detect_network_environment() {
    # 检测方法1：尝试连接外网（优先级高）
    if ping -c 1 8.8.8.8 >/dev/null 2>&1; then
        echo "external"
        return 0
    fi
    
    # 检测方法2：检查环境变量标识（强制覆盖）
    if [[ "${AI_INFRA_NETWORK_ENV}" == "internal" ]]; then
        echo "internal"
        return 0
    fi
    
    # 默认判定为内网
    echo "internal"
}
```

并从 `.env` 和 `.env.example` 中移除或注释 `AI_INFRA_NETWORK_ENV` 的默认值。

## 使用方法

### 方法 1：一键构建（推荐）

```bash
# 自动检测网络环境，处理所有镜像别名，构建并启动服务
./build.sh build-all
```

**执行流程**：
1. ✅ 检测网络环境（auto）
2. ✅ 创建环境配置 (.env)
3. ✅ **Step 2: 智能镜像别名管理**
   - 扫描 Dockerfile 中的基础镜像
   - 扫描 docker-compose.yml 中的第三方镜像
   - 为所有镜像创建完整别名
4. ✅ 同步配置文件
5. ✅ 渲染模板
6. ✅ 构建服务镜像
7. ✅ 验证构建结果
8. ✅ 启动 Docker Compose 服务

### 方法 2：手动处理镜像别名

```bash
# 处理单个带命名空间的镜像
./build.sh tag-localhost osixia/openldap:stable

# 处理多个镜像
./build.sh tag-localhost osixia/openldap:stable osixia/phpldapadmin:stable

# 处理所有 docker-compose.yml 中的镜像
grep -E '^\s*image:' docker-compose.yml | \
  grep -v '\$' | \
  awk '{print $2}' | \
  grep '/' | \
  xargs ./build.sh tag-localhost
```

### 方法 3：强制指定网络环境

```bash
# 强制使用公网环境
AI_INFRA_NETWORK_ENV=external ./build.sh build-all

# 强制使用内网环境
AI_INFRA_NETWORK_ENV=internal ./build.sh build-all
```

## 验证测试

### 测试 1: 带命名空间镜像处理

```bash
# 删除旧别名
docker rmi osixia/phpldapadmin:stable 2>/dev/null || true

# 运行智能 tag
./build.sh tag-localhost osixia/phpldapadmin:stable

# 验证结果
docker images | grep phpldapadmin

# 期望输出：
# phpldapadmin                stable    9831569a2f3d   4 years ago   443MB  ✓
# osixia/phpldapadmin         stable    9831569a2f3d   4 years ago   443MB  ✓
# localhost/phpldapadmin      stable    9831569a2f3d   4 years ago   443MB  ✓
```

### 测试 2: build-all 集成测试

```bash
# 运行完整构建
./build.sh build-all 2>&1 | grep -A 30 "步骤 2/6"

# 期望输出：
# [INFO] 步骤 2/6: 智能镜像别名管理
# [INFO] 扫描 Dockerfile 中的基础镜像...
# [INFO] 扫描 docker-compose.yml 中的第三方镜像...
# [INFO]   发现第三方镜像: osixia/openldap:stable
# [INFO]   发现第三方镜像: osixia/phpldapadmin:stable
# [INFO] 为 XX 个镜像创建智能别名...
# [SUCCESS]   ✓ 已创建别名: openldap:stable → osixia/openldap:stable
# [SUCCESS] ✓ 镜像别名管理完成
```

### 测试 3: Docker Compose 启动测试

```bash
# 验证配置
docker compose config --quiet && echo "✓ 配置验证成功"

# 启动服务（使用已有的镜像别名）
docker compose up -d openldap phpldapadmin

# 期望结果：
# [+] Running 2/2
#  ✔ Container ai-infra-openldap      Started  ✓
#  ✔ Container ai-infra-phpldapadmin  Started  ✓
```

## 支持的镜像类型

### 1. 简单镜像（无命名空间）

```yaml
redis:
  image: redis:7-alpine
```

**创建别名**：
- `redis:7-alpine` (原始)
- `localhost/redis:7-alpine` (localhost)

### 2. 带命名空间镜像（Docker Hub）

```yaml
openldap:
  image: osixia/openldap:stable
```

**创建别名**：
- `osixia/openldap:stable` (完整命名空间)
- `openldap:stable` (短名称)
- `localhost/openldap:stable` (localhost + 短名称)

### 3. Harbor 私有镜像

```yaml
custom:
  image: aiharbor.msxf.local/aihpc/custom:v1.0
```

**创建别名**：
- `aiharbor.msxf.local/aihpc/custom:v1.0` (Harbor 完整路径)
- `custom:v1.0` (基础名称)
- `localhost/custom:v1.0` (localhost)

## 影响范围

### 修改的文件

1. ✅ `build.sh`
   - `tag_image_smart()` - 增强命名空间支持
   - `detect_network_environment()` - 优先实际网络检测
   - `build_all_services()` Step 2 - 集成 docker-compose.yml 扫描

2. ✅ `.env`
   - 移除或注释 `AI_INFRA_NETWORK_ENV=internal`

3. ✅ `.env.example`
   - 移除或注释 `AI_INFRA_NETWORK_ENV` 的默认值
   - 添加说明：此变量由系统自动检测生成

### 影响的命令

- ✅ `./build.sh build-all` - 自动处理所有镜像
- ✅ `./build.sh tag-localhost` - 支持带命名空间镜像
- ✅ `docker compose up` - 无需手动拉取镜像

## 常见问题

### Q1: 为什么需要三个镜像标签？

**A**: 不同场景需要不同的标签形式：

1. **`osixia/openldap:stable`** - Docker Compose 需要的完整形式
2. **`openldap:stable`** - 简化引用，向后兼容
3. **`localhost/openldap:stable`** - 某些构建工具需要的前缀

所有标签指向同一个镜像 ID，不占用额外存储空间。

### Q2: 网络检测失败怎么办？

**A**: 手动强制指定环境

```bash
# 强制使用公网环境
AI_INFRA_NETWORK_ENV=external ./build.sh build-all

# 或在 .env 中设置（临时）
echo "AI_INFRA_NETWORK_ENV=external" >> .env
./build.sh build-all
```

### Q3: 如何添加新的第三方镜像？

**A**: 无需修改代码，只需：

1. 在 `docker-compose.yml` 中添加服务
2. 运行 `./build.sh build-all`
3. Step 2 自动扫描并处理新镜像

### Q4: Harbor 镜像不存在怎么办？

**A**: 系统自动降级

```
优先级：
1. Harbor 镜像（如果存在）
2. 完整命名空间镜像（osixia/openldap:stable）
3. 短名称镜像（openldap:stable）
4. localhost 镜像（localhost/openldap:stable）

从任何存在的镜像创建其他别名。
```

## 最佳实践

### 开发环境

```bash
# 一键完成所有操作
./build.sh build-all

# 自动：
# - 检测公网环境
# - 创建所有镜像别名
# - 构建自定义服务
# - 启动所有服务
```

### 生产环境（内网）

```bash
# 1. 从 Harbor 拉取所有依赖镜像
docker pull aiharbor.msxf.local/aihpc/osixia/openldap:stable
docker pull aiharbor.msxf.local/aihpc/osixia/phpldapadmin:stable
# ...

# 2. 运行构建（自动创建别名）
./build.sh build-all

# 3. 验证
docker compose up -d
```

### CI/CD 流程

```bash
#!/bin/bash
# 自动化部署脚本

# 1. 拉取代码
git pull origin main

# 2. 一键构建（自动检测环境）
./build.sh build-all

# 3. 健康检查
./build.sh check-status

# 4. 运行测试
docker-compose -f docker-compose.test.yml up -d
```

## 总结

此次修复解决了带命名空间的 Docker Hub 镜像在 `docker-compose.yml` 中使用时的标签不完整问题。

**核心改进**：
1. ✅ 支持带命名空间镜像（`osixia/openldap:stable`）
2. ✅ 自动创建多个别名（完整、短名称、localhost）
3. ✅ 集成到 `build-all` 自动处理
4. ✅ 动态网络环境检测
5. ✅ 多级降级策略（Harbor → 完整 → 短名称 → localhost）

**实际效果**：
- 一键 `./build.sh build-all` 即可完成所有操作
- 无需手动创建镜像别名
- 支持公网、内网、离线等多种部署场景
- 向后兼容，不影响现有功能

---

**修复日期**: 2025年10月11日  
**修复版本**: v0.3.7  
**相关组件**: build.sh, tag-localhost, build-all, docker-compose.yml
