# SLURM 客户端从 AppHub 安装指南

## 概述

本文档说明如何为 Alpine Linux 容器构建 SLURM 客户端，并通过 AppHub 分发安装。

## 背景

- **问题**：Alpine Linux 官方仓库不提供 SLURM 包
- **需求**：Backend 容器需要 SLURM 客户端工具（sinfo、squeue、scontrol 等）
- **解决方案**：从源码编译 SLURM 客户端，打包为 tar.gz，通过 AppHub 分发

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│  构建流程                                                     │
├─────────────────────────────────────────────────────────────┤
│  1. 运行构建脚本                                              │
│     ./scripts/build-slurm-client-alpine.sh                   │
│                                                              │
│  2. 在 Alpine 容器中编译 SLURM                                │
│     - 下载 SLURM 源码                                         │
│     - 配置编译选项（禁用不需要的特性）                          │
│     - 编译客户端工具                                          │
│                                                              │
│  3. 打包客户端                                                │
│     - 二进制文件：sinfo、squeue、scontrol、scancel、sbatch 等  │
│     - 动态库：libslurm.so                                     │
│     - 安装脚本：install.sh、uninstall.sh                      │
│     - 文档：README.md、VERSION                                │
│                                                              │
│  4. 上传到 AppHub                                             │
│     - 目标路径：/usr/share/nginx/html/pkgs/slurm-apk/        │
│     - 文件名：slurm-client-23.11.10-alpine.tar.gz            │
│     - 符号链接：slurm-client-latest-alpine.tar.gz            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  安装流程（Backend Dockerfile）                               │
├─────────────────────────────────────────────────────────────┤
│  1. 从 AppHub 下载包                                          │
│     wget http://apphub/pkgs/slurm-apk/slurm-client-latest... │
│                                                              │
│  2. 解压并运行安装脚本                                         │
│     tar xzf slurm.tar.gz && ./install.sh                     │
│                                                              │
│  3. 安装脚本执行：                                            │
│     - 复制文件到 /usr/local/slurm/                            │
│     - 创建符号链接到 /usr/bin/                                │
│     - 配置 LD_LIBRARY_PATH                                    │
│     - 设置环境变量                                            │
│                                                              │
│  4. 验证安装                                                  │
│     sinfo --version                                          │
└─────────────────────────────────────────────────────────────┘
```

## 使用步骤

### 1. 构建 SLURM Alpine 客户端包

```bash
# 进入项目目录
cd /path/to/ai-infra-matrix

# 运行构建脚本（需要 Docker）
./scripts/build-slurm-client-alpine.sh
```

**构建过程**：
- ⏱️ 预计时间：10-30 分钟（取决于网络和 CPU）
- 📦 输出位置：`./pkgs/slurm-apk/slurm-client-23.11.10-alpine.tar.gz`
- 📊 包大小：约 5-10 MB

**构建输出示例**：
```
[INFO] 开始构建 SLURM Alpine 客户端 v23.11.10...
[INFO] 创建 Alpine 构建容器...
>>> 安装构建依赖...
>>> 下载 SLURM 源码...
>>> 配置编译选项...
>>> 编译 SLURM 客户端工具...
>>> 安装客户端工具...
  ✓ Installed: sinfo
  ✓ Installed: squeue
  ✓ Installed: scontrol
  ✓ Installed: scancel
  ✓ Installed: sbatch
  ✓ Installed: srun
  ✓ Installed: salloc
  ✓ Installed: sacct
>>> 复制依赖库...
>>> 打包客户端工具...
[SUCCESS] SLURM Alpine 客户端包构建完成
[INFO] 上传到 AppHub...
[SUCCESS] 已上传到 AppHub: /usr/share/nginx/html/pkgs/slurm-apk/
[INFO] 下载 URL: http://localhost:8081/pkgs/slurm-apk/slurm-client-23.11.10-alpine.tar.gz
```

### 2. 验证 AppHub 中的包

```bash
# 检查包是否存在
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/

# 查看包内容
docker exec ai-infra-apphub tar tzf /usr/share/nginx/html/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz | head -20
```

### 3. 重新构建 Backend 容器

```bash
# 使用 build.sh 脚本
./build.sh build backend --force

# 或使用 docker-compose
docker-compose build --no-cache backend
```

**Backend Dockerfile 会自动**：
1. 从 AppHub 下载 SLURM 客户端包
2. 解压并运行安装脚本
3. 验证安装成功

### 4. 验证 Backend 容器中的 SLURM 客户端

```bash
# 进入 backend 容器
docker-compose exec backend bash

# 检查 SLURM 版本
sinfo --version

# 查看安装的命令
ls -la /usr/local/slurm/bin/

# 测试连接 SLURM master（如果可达）
sinfo -h
squeue -h
```

## 包结构

```
slurm-client-23.11.10-alpine.tar.gz
├── usr/
│   └── local/
│       └── slurm/
│           ├── bin/
│           │   ├── sinfo
│           │   ├── squeue
│           │   ├── scontrol
│           │   ├── scancel
│           │   ├── sbatch
│           │   ├── srun
│           │   ├── salloc
│           │   └── sacct
│           ├── lib/
│           │   └── libslurm.so*
│           └── VERSION
├── etc/
│   └── slurm/
├── install.sh      # 安装脚本
├── uninstall.sh    # 卸载脚本
└── README.md       # 使用说明
```

## Dockerfile 集成示例

### Backend Dockerfile（已集成）

```dockerfile
# 从 AppHub 安装预编译的 SLURM 客户端工具
RUN set -eux; \
    echo ">>> Installing SLURM client tools from AppHub..."; \
    # 尝试从 AppHub 下载（支持多种 URL 格式）
    for APPHUB_URL in http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz \
                      http://ai-infra-apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz \
                      http://192.168.0.200:8081/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz; do \
        if wget -q --timeout=10 --tries=2 "$APPHUB_URL" -O /tmp/slurm.tar.gz 2>/dev/null; then \
            echo "  ✓ Downloaded from: $APPHUB_URL"; \
            break; \
        fi; \
    done; \
    # 如果下载成功，安装 SLURM 客户端
    if [ -f /tmp/slurm.tar.gz ] && [ -s /tmp/slurm.tar.gz ]; then \
        echo ">>> Extracting and installing SLURM client..."; \
        cd /tmp; \
        tar xzf slurm.tar.gz; \
        if [ -f install.sh ]; then \
            chmod +x install.sh; \
            ./install.sh; \
            echo "  ✓ SLURM client installed successfully"; \
            # 验证安装
            if command -v sinfo >/dev/null 2>&1; then \
                echo "  ✓ SLURM version: $(sinfo --version 2>&1 | head -1)"; \
            fi; \
        fi; \
        rm -rf /tmp/slurm.tar.gz /tmp/install.sh; \
    else \
        echo "  ⚠ SLURM client download failed, will use demo data"; \
    fi
```

### 其他 Alpine 容器集成示例

```dockerfile
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    openssl \
    readline \
    ncurses \
    json-c \
    yaml \
    libevent \
    wget \
    ca-certificates

# 从 AppHub 安装 SLURM 客户端
RUN wget -q http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz -O /tmp/slurm.tar.gz && \
    cd /tmp && \
    tar xzf slurm.tar.gz && \
    ./install.sh && \
    rm -rf /tmp/slurm.tar.gz

# 验证安装
RUN sinfo --version
```

## 环境变量

安装后，`/etc/profile` 会包含：

```bash
# SLURM Client Environment
export SLURM_HOME=/usr/local/slurm
export PATH=$SLURM_HOME/bin:$PATH
export LD_LIBRARY_PATH=$SLURM_HOME/lib:$LD_LIBRARY_PATH
```

在容器中使用时可以 source：

```bash
source /etc/profile
sinfo --version
```

## 客户端工具说明

| 命令 | 功能 | 示例 |
|------|------|------|
| `sinfo` | 查看集群/节点信息 | `sinfo` |
| `squeue` | 查看作业队列 | `squeue` |
| `scontrol` | 集群管理工具 | `scontrol show config` |
| `scancel` | 取消作业 | `scancel <job_id>` |
| `sbatch` | 提交批处理作业 | `sbatch script.sh` |
| `srun` | 运行并行作业 | `srun -N 2 ./program` |
| `salloc` | 分配资源 | `salloc -N 2` |
| `sacct` | 作业统计 | `sacct -u username` |

## 连接到 SLURM Master

Backend 容器通过 Docker 网络连接到 SLURM master：

```bash
# 在 backend 容器中
export SLURM_CONF=/etc/slurm/slurm.conf

# 或者通过 SSH（如果配置了 SSH）
ssh slurm-master sinfo
ssh slurm-master squeue
```

## 故障排查

### 1. 构建失败

**问题**：`./scripts/build-slurm-client-alpine.sh` 失败

**检查**：
```bash
# 查看构建日志
cat /tmp/slurm-build.log

# 检查 Docker 是否运行
docker ps

# 检查网络连接
wget -q https://download.schedmd.com/slurm/ -O /dev/null && echo "OK"
```

### 2. 下载失败

**问题**：Backend 构建时无法从 AppHub 下载

**检查**：
```bash
# 检查 AppHub 容器是否运行
docker ps | grep apphub

# 检查包是否存在
docker exec ai-infra-apphub ls -l /usr/share/nginx/html/pkgs/slurm-apk/

# 测试下载
docker run --rm --network ai-infra-matrix_default alpine:latest \
  sh -c "apk add --no-cache wget && wget -q http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz -O /tmp/test.tar.gz && ls -lh /tmp/test.tar.gz"
```

### 3. 安装失败

**问题**：install.sh 执行失败

**检查**：
```bash
# 手动测试安装
docker run --rm --network ai-infra-matrix_default -v $(pwd)/pkgs/slurm-apk:/pkgs alpine:latest sh -c "
  cd /tmp
  cp /pkgs/slurm-client-latest-alpine.tar.gz .
  tar xzf slurm-client-latest-alpine.tar.gz
  ./install.sh
  sinfo --version
"
```

### 4. SLURM 命令不可用

**问题**：`sinfo: command not found`

**解决**：
```bash
# 检查是否安装
ls -la /usr/local/slurm/bin/

# Source 环境变量
source /etc/profile

# 或手动设置 PATH
export PATH=/usr/local/slurm/bin:$PATH
```

## 自定义配置

### 修改 SLURM 版本

编辑 `scripts/build-slurm-client-alpine.sh`：

```bash
# 修改版本号
SLURM_VERSION="${SLURM_VERSION:-24.05.0}"  # 改为你需要的版本
```

### 添加额外的客户端工具

编辑 `scripts/build-slurm-client-alpine.sh`，在打包部分添加：

```bash
for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sstat sprio; do
    if [ -f "src/${cmd}/${cmd}" ]; then
        cp -f "src/${cmd}/${cmd}" /tmp/slurm-install/usr/local/slurm/bin/
    fi
done
```

## 性能优化

### 1. 使用本地缓存

如果频繁构建，可以缓存 SLURM 源码：

```bash
# 下载一次
mkdir -p ~/.cache/slurm
wget https://download.schedmd.com/slurm/slurm-23.11.10.tar.bz2 \
  -O ~/.cache/slurm/slurm-23.11.10.tar.bz2

# 修改构建脚本使用本地缓存
# ...（在脚本中添加 -v ~/.cache/slurm:/cache 挂载）
```

### 2. 多架构构建

如果需要支持 x86_64 和 arm64：

```bash
# 使用 docker buildx
docker buildx build --platform linux/amd64,linux/arm64 ...
```

## 安全考虑

1. **包验证**：考虑添加 checksum 验证
2. **最小权限**：SLURM 客户端不需要 root 权限运行
3. **网络隔离**：AppHub 只在内网可访问
4. **版本锁定**：使用特定版本号而非 `latest`

## 参考资料

- [SLURM 官方文档](https://slurm.schedmd.com/)
- [SLURM 下载页面](https://download.schedmd.com/slurm/)
- [Alpine Linux 包管理](https://wiki.alpinelinux.org/wiki/Alpine_Linux_package_management)
- [AppHub 使用指南](./APPHUB_USAGE_GUIDE.md)

## 维护

### 更新 SLURM 版本

```bash
# 1. 修改版本号
export SLURM_VERSION=24.05.0

# 2. 重新构建
./scripts/build-slurm-client-alpine.sh

# 3. 重新构建 backend
./build.sh build backend --force
```

### 清理旧版本

```bash
# 清理 AppHub 中的旧版本
docker exec ai-infra-apphub sh -c "
  cd /usr/share/nginx/html/pkgs/slurm-apk/
  ls -lt | grep slurm-client- | tail -n +6 | awk '{print \$9}' | xargs rm -f
"
```

## 总结

通过这个方案，我们实现了：

- ✅ Alpine Linux 的 SLURM 客户端支持
- ✅ 通过 AppHub 统一分发
- ✅ 自动化构建和安装
- ✅ 版本管理和回退
- ✅ 最小化容器体积

Backend 容器现在可以使用完整的 SLURM 客户端工具，无需依赖演示数据。
