# Backend SLURM 客户端安装指南（APK 方式）

## 概述

Backend 容器现在通过 AppHub 的 APK 仓库直接安装 SLURM 客户端，无需编译源码。

## 架构

```
AppHub (APK Repository)
    ↓
    apks/alpine/
    ├── slurm-client-24.05.5-r0.apk
    ├── APKINDEX.tar.gz
    └── (签名文件)
    ↓
Backend Dockerfile
    ↓
    apk add slurm-client
    ↓
SLURM 客户端工具就绪
```

## 快速开始

### 1. 构建 SLURM APK 包

```bash
cd src/apphub
./build-slurm-apk.sh
```

这会：
- ✓ 在 Alpine 容器中构建 SLURM APK 包
- ✓ 生成 APK 索引和签名
- ✓ 上传到 AppHub 容器的 APK 仓库

### 2. 重新构建 Backend 镜像

```bash
docker-compose build backend
```

Backend Dockerfile 会自动：
- ✓ 添加 AppHub APK 仓库
- ✓ 安装 `slurm-client` 包
- ✓ 验证 SLURM 命令可用

### 3. 启动并验证

```bash
# 启动 backend 容器
docker-compose up -d backend

# 验证 SLURM 客户端
docker exec ai-infra-backend sh -c 'sinfo --version'
```

## 使用 SLURM 命令

### 基本命令格式

```bash
# 正确的格式（使用 sh -c）
docker exec ai-infra-backend sh -c 'sinfo'
docker exec ai-infra-backend sh -c 'squeue'
docker exec ai-infra-backend sh -c 'sbatch /path/to/job.sh'

# 错误的格式（不要这样用）
❌ docker exec ai-infra-backend "source /etc/profile ;sinfo"
❌ docker exec ai-infra-backend sinfo && squeue
```

### 常用命令

```bash
# 查看集群状态
docker exec ai-infra-backend sh -c 'sinfo'

# 查看节点详情
docker exec ai-infra-backend sh -c 'sinfo -N -l'

# 查看作业队列
docker exec ai-infra-backend sh -c 'squeue'

# 提交作业
docker exec ai-infra-backend sh -c 'sbatch /path/to/script.sh'

# 取消作业
docker exec ai-infra-backend sh -c 'scancel <job_id>'
```

## AppHub APK 仓库结构

```
AppHub 容器: /usr/share/nginx/html/apks/alpine/
├── slurm-client-24.05.5-r0.apk      # SLURM 客户端包
├── APKINDEX.tar.gz                  # APK 索引
└── <签名密钥>                       # APK 签名

访问 URL: http://apphub/apks/alpine/
```

## SLURM APK 包内容

安装的 SLURM 客户端工具包括：

- `sinfo` - 查看集群信息
- `squeue` - 查看作业队列
- `sbatch` - 提交批处理作业
- `salloc` - 分配资源
- `srun` - 运行作业步骤
- `scancel` - 取消作业
- `scontrol` - 集群控制命令
- `sacct` - 作业统计
- `sacctmgr` - 账户管理

## 故障排查

### 问题 1: SLURM 客户端未安装

**症状**:
```
sh: sinfo: not found
```

**解决方案**:

1. 检查 AppHub 是否有 APK 包:
```bash
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/apks/alpine/
```

2. 如果没有，构建 APK:
```bash
cd src/apphub
./build-slurm-apk.sh
```

3. 重新构建 backend:
```bash
docker-compose build backend
docker-compose up -d backend
```

### 问题 2: AppHub APK 仓库不可访问

**症状**:
```
WARNING: Ignoring http://apphub/apks/alpine: No such file or directory
```

**解决方案**:

1. 确认 AppHub 容器运行:
```bash
docker ps | grep apphub
```

2. 检查网络连接:
```bash
docker exec ai-infra-backend sh -c 'wget -O- http://apphub/apks/alpine/APKINDEX.tar.gz'
```

3. 检查 AppHub 容器中的文件:
```bash
docker exec ai-infra-apphub ls -la /usr/share/nginx/html/apks/alpine/
```

### 问题 3: APK 构建失败

**症状**:
```
make: *** [Makefile:982: config_info.lo] Error 1
```

**解决方案**:

这通常是因为编译依赖问题。使用修改后的构建脚本会：
- 使用预编译的二进制文件
- 或调整编译选项以适配 Alpine musl libc

### 问题 4: SLURM 连接失败

**症状**:
```
sinfo: error: Unable to contact slurm controller
```

**解决方案**:

1. 确认 slurm-master 运行:
```bash
docker ps | grep slurm-master
```

2. 检查配置文件:
```bash
docker exec ai-infra-backend sh -c 'cat /etc/slurm/slurm.conf'
```

3. 测试网络连接:
```bash
docker exec ai-infra-backend sh -c 'nc -zv slurm-master 6817'
```

## 测试工具

### 自动测试脚本

```bash
# 完整测试
./scripts/test-slurm-client.sh

# 快速修复
./scripts/fix-backend-slurm.sh
```

### 手动测试步骤

```bash
# 1. 检查 SLURM 命令存在
docker exec ai-infra-backend sh -c 'command -v sinfo'

# 2. 检查版本
docker exec ai-infra-backend sh -c 'sinfo --version'

# 3. 测试连接
docker exec ai-infra-backend sh -c 'sinfo'

# 4. 进入容器交互式调试
docker exec -it ai-infra-backend bash
```

## 高级配置

### 自定义 SLURM 版本

编辑 `src/apphub/build-slurm-apk.sh`:

```bash
SLURM_VERSION="24.05.5"  # 修改为需要的版本
```

然后重新构建：

```bash
cd src/apphub
./build-slurm-apk.sh
docker-compose build backend
docker-compose up -d backend
```

### 添加额外的 SLURM 插件

修改 APK 构建脚本中的 `package()` 函数，添加更多二进制文件或库。

### 使用本地 APK 缓存

将构建好的 APK 包保存到本地：

```bash
cp src/apphub/apks/alpine/*.apk /path/to/cache/
```

在 Dockerfile 中添加本地源：

```dockerfile
COPY cache/*.apk /tmp/
RUN apk add --allow-untrusted /tmp/slurm-client-*.apk
```

## 环境变量

Backend 容器中的 SLURM 相关环境变量：

```bash
# SLURM 配置文件路径
SLURM_CONF=/etc/slurm/slurm.conf

# SLURM 控制节点
SLURM_CONTROL_HOST=slurm-master
SLURM_CONTROL_PORT=6817

# MUNGE 认证（如果需要）
MUNGE_SOCKET=/var/run/munge/munge.socket.2
```

## 性能优化

### 减小 APK 包大小

只包含必要的工具：

```bash
# 在 APKBUILD 中只安装核心命令
install -m 755 src/sinfo/sinfo "$pkgdir/usr/bin/"
install -m 755 src/squeue/squeue "$pkgdir/usr/bin/"
# 其他工具可选
```

### 加速构建

使用构建缓存：

```bash
# 启用 Docker 构建缓存
docker-compose build --parallel backend
```

## 相关文档

- [SLURM 官方文档](https://slurm.schedmd.com/)
- [Alpine APK 工具文档](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper)
- [Backend SLURM 修复指南](./BACKEND_SLURM_FIX.md)
- [SLURM 客户端故障排除](./SLURM_CLIENT_TROUBLESHOOTING.md)

## 常见问题

**Q: 为什么不直接从 Alpine 官方仓库安装 SLURM？**

A: Alpine 官方仓库中的 SLURM 版本可能较旧，且不包含所有需要的组件。自建 APK 可以：
- 使用最新版本
- 自定义编译选项
- 只安装需要的客户端工具
- 减小镜像大小

**Q: APK 包需要签名吗？**

A: 建议签名以确保包的完整性，但在内部环境中可以使用 `--allow-untrusted` 跳过验证。

**Q: 如何更新 SLURM 版本？**

A: 修改构建脚本中的版本号，重新运行 `build-slurm-apk.sh`，然后重新构建 backend 镜像。

**Q: Backend 容器能否在没有 AppHub 的情况下运行？**

A: 可以。Dockerfile 中的安装逻辑会优雅降级，在无法获取 SLURM 客户端时会使用演示模式。
