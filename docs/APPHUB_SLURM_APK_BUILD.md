# AppHub SLURM APK 构建集成

## 概述

AppHub Dockerfile 现已集成 SLURM Alpine APK 包的构建，与现有的 DEB 和 RPM 构建流程保持一致。

## 构建架构

```
AppHub Dockerfile (Multi-Stage Build)
├─ Stage 1: deb-builder (Ubuntu 22.04)
│  └─ 构建 SLURM DEB 包 + 下载 SaltStack DEB
│
├─ Stage 2: rpm-builder (Rocky Linux 9)
│  └─ 下载 SaltStack RPM（SLURM RPM 需要 EPEL）
│
├─ Stage 3: apk-builder (Alpine Latest) ✨ 新增
│  └─ 从源码编译 SLURM 客户端 APK 包
│
└─ Stage 4: nginx:alpine (最终镜像)
   └─ 复制所有包并提供 HTTP 服务
```

## Stage 3: APK Builder 详细说明

### 构建流程

1. **基础镜像**: `alpine:latest`
2. **配置镜像源**: 多镜像回退（清华、阿里云、中科大、官方）
3. **安装依赖**: 
   - 核心: build-base, linux-headers, openssl-dev, readline-dev
   - 数据库: mariadb-dev
   - 工具: json-c-dev, yaml-dev, libevent-dev
   - 可选: http-parser-dev, numactl-dev, hwloc-dev
4. **解压源码**: 从构建上下文复制 SLURM tarball
5. **配置编译**:
   ```bash
   ./configure \
       --prefix=/usr/local/slurm \
       --sysconfdir=/etc/slurm \
       --without-munge \      # Alpine 不可用
       --without-pam \        # Alpine 不可用
       --without-rpath \
       --disable-debug \
       --without-gtk2 \
       --without-hdf5 \
       --without-numa \       # 可选
       --without-hwloc        # 可选
   ```
6. **编译**: `make -j$(nproc)`
7. **安装**: 手动复制客户端工具到临时目录
8. **打包**: 创建 `slurm-client-25.05.4-alpine.tar.gz`

### 包内容

```
slurm-client-25.05.4-alpine.tar.gz
├── usr/local/slurm/
│   ├── bin/
│   │   ├── sinfo
│   │   ├── squeue
│   │   ├── scontrol
│   │   ├── scancel
│   │   ├── sbatch
│   │   ├── srun
│   │   ├── salloc
│   │   └── sacct
│   ├── lib/
│   │   └── libslurm.so*
│   └── VERSION
├── etc/slurm/
├── install.sh     # 自动安装脚本
├── uninstall.sh   # 卸载脚本
└── README.md      # 使用说明
```

### install.sh 功能

自动化安装脚本包含：
- 复制文件到 `/usr/local/slurm/`
- 创建符号链接到 `/usr/bin/`
- 配置 LD_LIBRARY_PATH
- 设置环境变量到 `/etc/profile`

### 编译选项说明

| 选项 | 说明 | 原因 |
|------|------|------|
| `--without-munge` | 禁用 Munge 认证 | Alpine 不提供 munge-dev |
| `--without-pam` | 禁用 PAM 支持 | Alpine 不提供 pam-dev |
| `--without-numa` | 禁用 NUMA 支持 | 可选，减少依赖 |
| `--without-hwloc` | 禁用硬件拓扑 | 可选，减少依赖 |

## Stage 4: 最终镜像集成

### 复制包

```dockerfile
# Copy all deb packages from deb-builder stage
COPY --from=deb-builder /out/ /usr/share/nginx/html/pkgs/slurm-deb/

# Copy all rpm packages from rpm-builder stage
COPY --from=rpm-builder /out/ /usr/share/nginx/html/pkgs/slurm-rpm/

# Copy all apk packages from apk-builder stage ✨ 新增
COPY --from=apk-builder /out/ /usr/share/nginx/html/pkgs/slurm-apk/
```

### 包统计

构建完成后会显示：

```
📊 Package Summary:
  - SLURM deb packages: 12
  - SLURM rpm packages: 0 (需要 EPEL)
  - SLURM apk packages: 1  ✨ 新增
  - SaltStack deb packages: 6
  - SaltStack rpm packages: 6
```

### 符号链接

自动创建 `latest` 链接：

```bash
slurm-client-latest-alpine.tar.gz -> slurm-client-25.05.4-alpine.tar.gz
```

## 使用方法

### 构建 AppHub

```bash
# 确保 SLURM 源码包存在
ls src/apphub/slurm-25.05.4.tar.bz2

# 构建 AppHub（多阶段构建会自动编译所有包）
./build.sh build apphub --force

# 或使用 docker-compose
docker-compose build --no-cache apphub
```

### 访问包

启动 AppHub 后，包可通过以下 URL 访问：

```bash
# DEB 包
http://apphub/pkgs/slurm-deb/
http://apphub/pkgs/saltstack-deb/

# RPM 包
http://apphub/pkgs/slurm-rpm/
http://apphub/pkgs/saltstack-rpm/

# APK 包 ✨ 新增
http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz
http://apphub/pkgs/slurm-apk/slurm-client-25.05.4-alpine.tar.gz
```

### Backend 自动下载

Backend Dockerfile 已配置从 AppHub 自动下载：

```dockerfile
RUN set -eux; \
    for APPHUB_URL in http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz \
                      http://ai-infra-apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz; do \
        if wget -q "$APPHUB_URL" -O /tmp/slurm.tar.gz 2>/dev/null; then \
            break; \
        fi; \
    done; \
    if [ -f /tmp/slurm.tar.gz ]; then \
        cd /tmp && tar xzf slurm.tar.gz && ./install.sh; \
    fi
```

## 构建时间和资源

### 各阶段构建时间（预估）

| 阶段 | 时间 | 说明 |
|------|------|------|
| Stage 1 (DEB) | 5-15分钟 | 下载依赖 + 编译 SLURM DEB |
| Stage 2 (RPM) | 2-5分钟 | 仅下载 SaltStack RPM |
| Stage 3 (APK) | 10-30分钟 | 编译 SLURM 客户端 ✨ |
| Stage 4 (Nginx) | 1-2分钟 | 复制包 + 生成索引 |
| **总计** | **18-52分钟** | 取决于网络和 CPU |

### 镜像大小

- **构建镜像**: ~2-3 GB（包含所有构建工具）
- **最终镜像**: ~200-300 MB（仅 Nginx + 包）

## 验证

### 检查包是否存在

```bash
# 进入 AppHub 容器
docker exec -it ai-infra-apphub sh

# 查看所有包
ls -lh /usr/share/nginx/html/pkgs/slurm-apk/

# 应该看到：
# slurm-client-25.05.4-alpine.tar.gz
# slurm-client-latest-alpine.tar.gz -> slurm-client-25.05.4-alpine.tar.gz
```

### 测试下载

```bash
# 从 Host 测试
curl -I http://localhost:8081/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz

# 从 Backend 容器测试
docker-compose exec backend wget -q http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz -O /tmp/test.tar.gz
docker-compose exec backend tar tzf /tmp/test.tar.gz | head -20
```

### 验证安装脚本

```bash
# 下载并测试安装
docker run --rm --network ai-infra-matrix_default alpine:latest sh -c "
  apk add --no-cache wget
  wget -q http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz
  tar xzf slurm-client-latest-alpine.tar.gz
  ./install.sh
  sinfo --version
"
```

## 故障排查

### 问题 1: 构建失败 - 源码包不存在

**错误**:
```
ERROR: failed to compute cache key: failed to calculate checksum of ref...
```

**解决**:
```bash
# 下载 SLURM 源码到 AppHub 目录
cd src/apphub
wget https://download.schedmd.com/slurm/slurm-25.05.4.tar.bz2

# 重新构建
docker-compose build apphub
```

### 问题 2: 编译失败 - 依赖缺失

**错误**:
```
configure: error: Cannot find required library
```

**解决**:
检查 Dockerfile 中的依赖安装是否完整，可能需要添加额外的 `-dev` 包。

### 问题 3: APK 包为空

**检查构建日志**:
```bash
docker-compose build apphub 2>&1 | grep -A 20 "Stage 3"
```

**验证**:
```bash
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/
```

### 问题 4: Backend 下载失败

**检查网络**:
```bash
docker-compose exec backend ping -c 3 apphub
docker-compose exec backend wget -S http://apphub/pkgs/slurm-apk/
```

**检查 URL**:
```bash
# 在 Backend 构建日志中查找
docker-compose build backend 2>&1 | grep "Installing SLURM"
```

## 与独立构建脚本的对比

### 独立脚本方式 (`build-slurm-client-alpine.sh`)

**优点**:
- 灵活，可单独运行
- 不影响 AppHub 构建时间
- 便于测试和调试

**缺点**:
- 需要手动上传到 AppHub
- 构建环境不一致
- 需要额外维护

### AppHub 集成方式（当前实现）

**优点**:
- ✅ 一键构建所有包（DEB + RPM + APK）
- ✅ 构建环境统一
- ✅ 自动化部署
- ✅ 版本一致性保证

**缺点**:
- 增加 AppHub 构建时间（10-30分钟）
- 需要 SLURM 源码包在构建上下文中

## 建议

### 开发环境

使用独立脚本快速迭代：

```bash
./scripts/build-slurm-client-alpine.sh
docker cp pkgs/slurm-apk/* ai-infra-apphub:/usr/share/nginx/html/pkgs/slurm-apk/
```

### 生产环境

使用 AppHub 集成构建确保一致性：

```bash
./build.sh build apphub --force
```

## 更新 SLURM 版本

1. 下载新版本源码包：
   ```bash
   cd src/apphub
   wget https://download.schedmd.com/slurm/slurm-XX.XX.XX.tar.bz2
   ```

2. 更新 Dockerfile ARG：
   ```dockerfile
   ARG SLURM_VERSION=XX.XX.XX
   ```

3. 重新构建：
   ```bash
   ./build.sh build apphub --force
   ```

## 总结

AppHub 现在支持三种包格式的完整构建：

| 包格式 | 构建阶段 | 基础镜像 | 输出 |
|--------|----------|----------|------|
| DEB | Stage 1 | Ubuntu 22.04 | 12+ SLURM DEB 包 |
| RPM | Stage 2 | Rocky Linux 9 | SaltStack RPM（SLURM 需 EPEL） |
| APK | Stage 3 | Alpine Latest | 1 个 SLURM 客户端包 ✨ |

所有包在 Stage 4（Nginx）中统一提供 HTTP 下载服务，Backend 和其他组件可以自动下载安装。

## 相关文档

- [Backend SLURM 客户端安装](./BACKEND_SLURM_CLIENT_SETUP.md)
- [SLURM AppHub 安装指南](./SLURM_APPHUB_INSTALLATION.md)
- [AppHub 使用指南](./APPHUB_USAGE_GUIDE.md)
