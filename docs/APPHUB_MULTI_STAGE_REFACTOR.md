# AppHub 多阶段构建重构报告

## 需求背景

**需求编号**: 86  
**日期**: 2025年10月20日  
**目标**: 重构 AppHub Dockerfile，使用多阶段构建同时支持 SLURM 和 SaltStack 的 deb/rpm 包

## 架构设计

### 三阶段构建流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    Stage 1: deb-builder                          │
│                    (Ubuntu 22.04)                                │
│                                                                  │
│  1. 构建 SLURM deb 包（从源码编译）                              │
│  2. 下载 SaltStack deb 包（从官方仓库）                          │
│  3. 输出到 /out 目录                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Stage 2: rpm-builder                          │
│                    (Rocky Linux 9)                               │
│                                                                  │
│  1. 构建 SLURM rpm 包（从源码编译）                              │
│  2. 下载 SaltStack rpm 包（从官方仓库）                          │
│  3. 输出到 /out 目录                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Stage 3: AppHub                               │
│                    (nginx:alpine)                                │
│                                                                  │
│  1. 复制 deb 包到 /usr/share/nginx/html/pkgs/                   │
│  2. 复制 rpm 包到 /usr/share/nginx/html/pkgs/                   │
│  3. 分离 SLURM 和 SaltStack 包到不同目录                         │
│  4. 生成 deb 包索引（Packages.gz）                               │
│  5. 提供 HTTP 服务                                               │
└─────────────────────────────────────────────────────────────────┘
```

## 详细实现

### Stage 1: deb-builder (Ubuntu 22.04)

**功能**:
1. 从 `slurm-25.05.4.tar.bz2` 源码构建 SLURM deb 包
2. 从 SaltStack 官方仓库下载 deb 包

**关键配置**:
```dockerfile
FROM ubuntu:22.04 AS deb-builder

# SLURM 版本配置
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_PATH=slurm-${SLURM_VERSION}.tar.bz2

# 使用阿里云镜像源加速
# 根据架构自动选择 amd64 或 arm64 源
```

**SLURM 构建流程**:
1. 安装构建依赖：`build-essential`, `devscripts`, `debhelper` 等
2. 安装开发库：`libmunge-dev`, `libmariadb-dev`, `libpam0g-dev` 等
3. 以非 root 用户执行 `dpkg-buildpackage -b -uc`
4. 收集生成的 `.deb` 包

**SaltStack 下载**:
```bash
SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/ubuntu/22.04/amd64/latest"
# 下载的包:
- salt-master_3007.1-1_amd64.deb
- salt-minion_3007.1-1_amd64.deb
- salt-common_3007.1-1_amd64.deb
- salt-api_3007.1-1_amd64.deb
- salt-ssh_3007.1-1_amd64.deb
- salt-syndic_3007.1-1_amd64.deb
```

**输出**: `/out/` 目录包含所有 deb 包

### Stage 2: rpm-builder (Rocky Linux 9)

**功能**:
1. 从 `slurm-25.05.4.tar.bz2` 源码构建 SLURM rpm 包
2. 从 SaltStack 官方仓库下载 rpm 包

**关键配置**:
```dockerfile
FROM rockylinux:9 AS rpm-builder

# SLURM 版本配置（与 deb-builder 一致）
ARG SLURM_VERSION=25.05.4
ARG SLURM_TARBALL_PATH=slurm-${SLURM_VERSION}.tar.bz2

# 使用阿里云 Rocky Linux 镜像源
```

**SLURM 构建流程**:
1. 安装构建依赖：`rpm-build`, `rpmdevtools`, `gcc`, `make` 等
2. 安装开发库：`munge-devel`, `mariadb-devel`, `pam-devel` 等
3. 设置 RPM 构建环境：`rpmdev-setuptree`
4. 执行 `rpmbuild -ba slurm.spec`
5. 收集生成的 `.rpm` 包

**SaltStack 下载**:
```bash
SALTSTACK_REPO="https://repo.saltproject.io/salt/py3/redhat/9/x86_64/latest"
# 下载的包:
- salt-master-3007.1-1.el9.x86_64.rpm
- salt-minion-3007.1-1.el9.x86_64.rpm
- salt-3007.1-1.el9.x86_64.rpm
- salt-api-3007.1-1.el9.x86_64.rpm
- salt-ssh-3007.1-1.el9.x86_64.rpm
- salt-syndic-3007.1-1.el9.x86_64.rpm
```

**输出**: `/out/` 目录包含所有 rpm 包

### Stage 3: AppHub (nginx:alpine)

**功能**:
1. 提供轻量级的 nginx HTTP 服务器
2. 复制并组织所有 deb/rpm 包
3. 生成 APT 仓库索引
4. 提供包下载服务

**目录结构**:
```
/usr/share/nginx/html/
├── deb/                    # 通用 deb 包目录
├── rpm/                    # 通用 rpm 包目录
└── pkgs/
    ├── slurm-deb/         # SLURM deb 包 + Packages.gz
    ├── slurm-rpm/         # SLURM rpm 包
    ├── saltstack-deb/     # SaltStack deb 包 + Packages.gz
    └── saltstack-rpm/     # SaltStack rpm 包
```

**包分离逻辑**:
```bash
# 从混合目录中分离 SaltStack 包
find . -name "salt-*.deb" -exec mv {} /saltstack-deb/ \;
find . -name "salt-*.rpm" -exec mv {} /saltstack-rpm/ \;
```

**索引生成**:
```bash
# 为每个 deb 目录生成 APT 索引
cd /usr/share/nginx/html/pkgs/slurm-deb
dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz

cd /usr/share/nginx/html/pkgs/saltstack-deb
dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz
```

**镜像源配置**:
- 主源：阿里云 Alpine 镜像 (HTTP)
- 备份1：清华大学镜像
- 备份2：Alpine 官方 CDN

## 包版本信息

### SLURM
- **版本**: 25.05.4
- **源码**: `slurm-25.05.4.tar.bz2`
- **构建类型**: 源码编译
- **包数量**: ~15-20 个（取决于配置）

### SaltStack
- **版本**: 3007.1
- **获取方式**: 官方仓库下载
- **包数量**: 6 个核心包

## 使用方法

### 构建镜像

```bash
# 构建 AppHub 镜像
./build.sh build apphub

# 强制重新构建
./build.sh build apphub --force
```

### 访问包仓库

构建完成后，可以通过 HTTP 访问包：

```bash
# SLURM deb 包
http://apphub-host/pkgs/slurm-deb/

# SLURM rpm 包
http://apphub-host/pkgs/slurm-rpm/

# SaltStack deb 包
http://apphub-host/pkgs/saltstack-deb/

# SaltStack rpm 包
http://apphub-host/pkgs/saltstack-rpm/
```

### 配置客户端使用仓库

**Ubuntu/Debian 客户端**:
```bash
# 添加 SLURM 仓库
echo "deb [trusted=yes] http://apphub-host/pkgs/slurm-deb ./" > /etc/apt/sources.list.d/slurm.list

# 添加 SaltStack 仓库
echo "deb [trusted=yes] http://apphub-host/pkgs/saltstack-deb ./" > /etc/apt/sources.list.d/saltstack.list

# 更新并安装
apt-get update
apt-get install slurm-client salt-minion
```

**RHEL/Rocky 客户端**:
```bash
# 创建 SLURM 仓库配置
cat > /etc/yum.repos.d/slurm.repo <<EOF
[slurm]
name=SLURM
baseurl=http://apphub-host/pkgs/slurm-rpm/
enabled=1
gpgcheck=0
EOF

# 创建 SaltStack 仓库配置
cat > /etc/yum.repos.d/saltstack.repo <<EOF
[saltstack]
name=SaltStack
baseurl=http://apphub-host/pkgs/saltstack-rpm/
enabled=1
gpgcheck=0
EOF

# 安装
dnf install slurm-client salt-minion
```

## 版本升级指南

### 升级 SLURM 版本

1. 下载新版本源码：
   ```bash
   cd src/apphub
   wget https://download.schedmd.com/slurm/slurm-25.05.5.tar.bz2
   ```

2. 更新 Dockerfile ARG：
   ```dockerfile
   ARG SLURM_VERSION=25.05.5
   ```

3. 重新构建：
   ```bash
   ./build.sh build apphub --force
   ```

### 升级 SaltStack 版本

1. 更新 Dockerfile 中的下载 URL：
   ```dockerfile
   # Stage 1 (deb)
   wget -nv "${SALTSTACK_REPO}/salt-master_3007.2-1_amd64.deb"
   
   # Stage 2 (rpm)
   wget -nv "${SALTSTACK_REPO}/salt-master-3007.2-1.el9.x86_64.rpm"
   ```

2. 重新构建：
   ```bash
   ./build.sh build apphub --force
   ```

## 技术特点

### 优势

1. **多平台支持**: 同时提供 deb 和 rpm 包
2. **版本管理**: 通过 ARG 变量统一管理版本
3. **镜像源优化**: 使用国内镜像源加速构建
4. **容错机制**: 
   - 镜像源自动切换
   - SLURM 构建失败不影响 SaltStack
   - 下载失败会显示警告但不中断
5. **轻量化**: 最终镜像基于 nginx:alpine，体积小
6. **模块化**: 各个阶段独立，易于维护

### 局限性

1. **RPM 元数据**: Alpine 中没有 `createrepo`，无法生成 YUM/DNF 元数据
   - 影响：无法使用 `yum search` 或依赖解析
   - 解决方案：包仍可直接下载安装
   
2. **构建时间**: 从源码编译 SLURM 较耗时（~10-20分钟）
   
3. **架构限制**: 
   - SaltStack 下载仅支持 amd64/x86_64
   - ARM 架构需要单独处理

## 故障排查

### 常见问题

**Q: SLURM 构建失败**
```
ERROR: No .deb packages were built!
```
**A**: 检查：
1. 源码 tarball 是否存在
2. 构建依赖是否完整安装
3. 查看详细构建日志

**Q: SaltStack 下载失败**
```
⚠️  Failed to download salt-master
```
**A**: 
1. 检查网络连接
2. 验证 SaltStack 仓库 URL 是否有效
3. 尝试手动下载测试

**Q: Alpine 镜像源超时**
```
ERROR: unable to select packages
```
**A**: 
- 自动切换到备用镜像源
- 检查 Docker 网络配置

## 性能优化建议

1. **使用构建缓存**:
   ```bash
   # 利用 Docker BuildKit 缓存
   DOCKER_BUILDKIT=1 docker build --cache-from=ai-infra-apphub:latest .
   ```

2. **并行构建**: Stage 1 和 Stage 2 可以并行（Docker 自动优化）

3. **预下载包**: 将 SaltStack 包预下载到本地，避免每次构建时下载

4. **多架构支持**:
   ```bash
   docker buildx build --platform linux/amd64,linux/arm64 -t apphub:multi .
   ```

## 测试验证

### 验证构建成功

```bash
# 检查镜像
docker images | grep apphub

# 运行容器
docker run -d -p 8080:80 ai-infra-apphub:v0.3.6-dev

# 测试访问
curl http://localhost:8080/pkgs/slurm-deb/
curl http://localhost:8080/pkgs/saltstack-deb/
```

### 验证包完整性

```bash
# 进入容器
docker exec -it <container-id> sh

# 检查包数量
ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb | wc -l
ls /usr/share/nginx/html/pkgs/saltstack-deb/*.deb | wc -l
ls /usr/share/nginx/html/pkgs/slurm-rpm/*.rpm | wc -l
ls /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm | wc -l

# 检查索引文件
zcat /usr/share/nginx/html/pkgs/slurm-deb/Packages.gz | head -20
zcat /usr/share/nginx/html/pkgs/saltstack-deb/Packages.gz | head -20
```

## 相关文档

- [SLURM 官方文档](https://slurm.schedmd.com/)
- [SaltStack 文档](https://docs.saltproject.io/)
- [Docker 多阶段构建](https://docs.docker.com/build/building/multi-stage/)
- [APT 仓库格式](https://wiki.debian.org/DebianRepository/Format)

## 更新记录

| 日期 | 版本 | 说明 |
|------|------|------|
| 2025-10-20 | 1.0 | 初始版本，重构为三阶段构建 |
| 2025-10-20 | 1.1 | 添加 SaltStack 下载支持 |
| 2025-10-20 | 1.2 | 实现包分离和多仓库索引 |

## 总结

本次重构实现了 AppHub 的完整多阶段构建架构，支持：
- ✅ Ubuntu/Debian (deb) 包管理
- ✅ RHEL/Rocky (rpm) 包管理
- ✅ SLURM 和 SaltStack 同时支持
- ✅ 自动化构建和索引生成
- ✅ 轻量级最终镜像
- ✅ 容错和自动恢复机制

该架构为后续添加更多软件包（如监控工具、编译器等）提供了良好的扩展基础。
