# 镜像别名双向创建 - 快速参考

## 问题与解决

### ❌ 问题
```bash
$ docker images | grep redis
localhost/redis    7-alpine    bb186d083732    3 months ago    61.4MB
# 缺少 redis:7-alpine（原始名称）
```

### ✅ 解决
```bash
$ ./build.sh tag-localhost redis:7-alpine

# 或者运行完整构建
$ ./build.sh build-all

# 结果
$ docker images | grep redis
redis               7-alpine    bb186d083732    3 months ago    61.4MB  ✓
localhost/redis     7-alpine    bb186d083732    3 months ago    61.4MB  ✓
```

## 快速修复

### 手动创建单个镜像别名

```bash
# 如果有 localhost/image:tag，创建 image:tag
docker tag localhost/redis:7-alpine redis:7-alpine

# 如果有 image:tag，创建 localhost/image:tag  
docker tag redis:7-alpine localhost/redis:7-alpine
```

### 批量修复所有镜像

```bash
# 扫描并创建所有缺失的别名
./build.sh tag-localhost

# 或者指定具体镜像
./build.sh tag-localhost redis:7-alpine nginx:stable golang:1.25-alpine
```

### 集成到完整构建流程

```bash
# 推荐：一键构建，自动处理所有别名
./build.sh build-all

# Step 2 会自动创建所有基础镜像的双向别名
```

## 工作原理

### 公网环境

```
检测逻辑：
1. 有 redis:7-alpine → 创建 localhost/redis:7-alpine
2. 有 localhost/redis:7-alpine → 创建 redis:7-alpine
3. 两者都有 → 跳过
4. 两者都无 → 警告
```

### 内网环境

```
降级策略：
1. 优先：Harbor 镜像存在
   - aiharbor/redis:7-alpine → redis:7-alpine
   - aiharbor/redis:7-alpine → localhost/redis:7-alpine

2. 降级：localhost/ 镜像存在（Harbor 不可用）
   - localhost/redis:7-alpine → redis:7-alpine

3. 再降级：原始镜像存在
   - redis:7-alpine → localhost/redis:7-alpine
```

## 常见场景

### 场景 1: 只有 localhost/ 镜像

```bash
# 现状
$ docker images | grep redis
localhost/redis    7-alpine    bb186d083732

# 修复
$ ./build.sh tag-localhost redis:7-alpine

# 结果
$ docker images | grep redis
redis               7-alpine    bb186d083732  ✓
localhost/redis     7-alpine    bb186d083732  ✓
```

### 场景 2: 只有原始镜像

```bash
# 现状
$ docker images | grep nginx
nginx    stable    abc123

# 修复
$ ./build.sh tag-localhost nginx:stable

# 结果
$ docker images | grep nginx
nginx               stable    abc123  ✓
localhost/nginx     stable    abc123  ✓
```

### 场景 3: 两个版本都有

```bash
# 现状
$ docker images | grep golang
golang              1.25-alpine    def456
localhost/golang    1.25-alpine    def456

# 修复
$ ./build.sh tag-localhost golang:1.25-alpine

# 结果
[INFO] 处理镜像: golang:1.25-alpine
[INFO]   ✓ 原始镜像已存在: golang:1.25-alpine
[INFO]   ✓ localhost 镜像已存在: localhost/golang:1.25-alpine
# 跳过，无需创建
```

## 验证检查

### 检查所有基础镜像

```bash
# 查看当前 localhost/ 镜像
docker images | grep "^localhost/"

# 对应检查原始镜像
for img in $(docker images | grep "^localhost/" | awk '{print $1":"$2}' | sed 's/localhost\///'); do
  if docker image inspect "$img" >/dev/null 2>&1; then
    echo "✓ $img"
  else
    echo "✗ $img (缺少)"
  fi
done
```

### 检查特定镜像

```bash
# 检查 redis 镜像
docker images | grep -E "^redis|^localhost/redis"

# 预期输出（两行）
redis               7-alpine    bb186d083732
localhost/redis     7-alpine    bb186d083732
```

## 环境变量配置

### 检查当前网络环境

```bash
$ ./build.sh detect-network
[INFO] 当前网络环境: external  # 或 internal
```

### 临时切换网络环境

```bash
# 强制使用公网模式
AI_INFRA_NETWORK_ENV=external ./build.sh tag-localhost redis:7-alpine

# 强制使用内网模式
AI_INFRA_NETWORK_ENV=internal ./build.sh tag-localhost redis:7-alpine
```

### 永久修改网络环境

```bash
# 编辑 .env 文件
vim .env

# 设置网络环境
AI_INFRA_NETWORK_ENV=external  # 或 internal

# 重新运行
./build.sh tag-localhost
```

## 故障排查

### Q1: 为什么没有创建别名？

**检查网络环境**：
```bash
$ ./build.sh detect-network
[INFO] 当前网络环境: internal

# 如果是 internal，确保镜像存在
$ docker images | grep redis
localhost/redis    7-alpine    xxx  # ✓ 存在

# 重新运行
$ ./build.sh tag-localhost redis:7-alpine
```

### Q2: Harbor 镜像不存在怎么办？

**降级到本地镜像**：
```bash
# 内网模式会自动降级
$ ./build.sh tag-localhost redis:7-alpine
[INFO]   🏢 内网环境：检查镜像来源
[INFO]     💡 Harbor 不可用，使用本地 localhost/ 镜像
[SUCCESS]  ✓ 已创建别名: localhost/redis:7-alpine → redis:7-alpine
```

### Q3: 所有方式都失败？

**手动拉取镜像**：
```bash
# 公网环境
docker pull redis:7-alpine

# 内网环境（Harbor）
docker pull aiharbor.msxf.local/aihpc/redis:7-alpine

# 然后重新运行
./build.sh tag-localhost redis:7-alpine
```

## 最佳实践

### 推荐工作流

```bash
# 1. 初始构建
./build.sh build-all

# 2. 验证镜像
docker images | grep -E "^(redis|nginx|golang)"

# 3. 检查别名
./build.sh check-status

# 4. 如有问题，单独修复
./build.sh tag-localhost <镜像名称>
```

### 自动化脚本

```bash
#!/bin/bash
# 自动修复所有镜像别名

# 扫描所有 localhost/ 镜像
for img in $(docker images | grep "^localhost/" | awk '{print $1":"$2}' | sed 's/localhost\///'); do
  # 检查原始镜像是否存在
  if ! docker image inspect "$img" >/dev/null 2>&1; then
    echo "修复: $img"
    ./build.sh tag-localhost "$img"
  fi
done
```

## 相关文档

- [完整修复报告](./IMAGE_ALIAS_BIDIRECTIONAL_FIX.md)
- [智能镜像 Tag 管理指南](./IMAGE_TAG_SMART_GUIDE.md)
- [Build-All 集成说明](./BUILD_ALL_SMART_TAG_INTEGRATION.md)

---

**更新日期**: 2025年10月11日  
**适用版本**: v0.3.7+
