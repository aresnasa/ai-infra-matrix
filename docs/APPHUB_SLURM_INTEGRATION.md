# AppHub SLURM 集成方案

## 概述

本文档描述了如何确保 SLURM 客户端工具只从 AppHub 安装，涵盖 RPM、DEB、APK 三种包格式。

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      AppHub 构建流程                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Stage 1: deb-builder (Ubuntu 22.04)                         │
│    ├─ 解压 slurm-25.05.4.tar.bz2                             │
│    ├─ 构建 SLURM DEB 包                                      │
│    └─ 输出到 /out/*.deb                                       │
│                                                               │
│  Stage 2: rpm-builder (Rocky Linux 9)                        │
│    ├─ 解压 slurm-25.05.4.tar.bz2                             │
│    ├─ 构建 SLURM RPM 包（需要 EPEL）                         │
│    └─ 输出到 /out/*.rpm                                       │
│                                                               │
│  Stage 3: apk-builder (Alpine Linux)                         │
│    ├─ 解压 slurm-25.05.4.tar.bz2                             │
│    ├─ 编译 SLURM 客户端工具                                  │
│    ├─ 创建 tar.gz 包（包含 install.sh）                      │
│    └─ 输出到 /out/slurm-client-25.05.4-alpine.tar.gz         │
│                                                               │
│  Stage 4: Final (nginx:alpine)                               │
│    ├─ COPY --from=deb-builder /out/ → slurm-deb/            │
│    ├─ COPY --from=rpm-builder /out/ → slurm-rpm/            │
│    ├─ COPY --from=apk-builder /out/ → slurm-apk/            │
│    └─ 生成索引文件和符号链接                                 │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP 下载
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Backend 容器构建                           │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 检测 AppHub 可用性                                        │
│     - 探测 http://apphub/pkgs/slurm-apk/                     │
│                                                               │
│  2. 下载 SLURM 包                                            │
│     - wget http://apphub/pkgs/slurm-apk/slurm-client-*       │
│                                                               │
│  3. 解压并安装                                                │
│     - tar xzf slurm-client-*.tar.gz                          │
│     - ./install.sh                                           │
│                                                               │
│  4. 验证安装                                                  │
│     - sinfo --version                                        │
│                                                               │
│  ❌ 如果失败 → 停止构建并报错                                 │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## 当前问题

1. **APK 包构建可能失败**：`/usr/share/nginx/html/pkgs/slurm-apk/` 为空
2. **Backend 有备用源**：会从 Alpine edge 仓库安装，不符合要求
3. **缺少强制检查**：AppHub 不可用时不应继续构建

## 修复方案

### 1. 检查并修复 AppHub APK 构建

需要确保 `apk-builder` 阶段正确输出 `.tar.gz` 包：

```dockerfile
# Stage 3: apk-builder
# 输出格式：slurm-client-25.05.4-alpine.tar.gz
# 包含：
#   - usr/local/slurm/bin/{sinfo,squeue,scontrol,...}
#   - usr/local/slurm/lib/libslurm*.so*
#   - install.sh
#   - uninstall.sh
```

### 2. 修改 Backend Dockerfile

移除所有备用源，只从 AppHub 安装：

```dockerfile
# ❌ 移除 Alpine edge 仓库备用方案
# ✅ 只从 AppHub 下载安装
# ✅ 失败则报错停止构建
```

### 3. 构建验证流程

```bash
# 步骤 1: 重建 AppHub（确保 APK 包生成）
docker-compose build --no-cache apphub

# 步骤 2: 验证 APK 包存在
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/

# 步骤 3: 重建 Backend
docker-compose build --no-cache backend

# 步骤 4: 验证 SLURM 安装
docker exec ai-infra-backend sinfo --version
```

## 包格式规范

### APK 包结构（tar.gz）

```
slurm-client-25.05.4-alpine.tar.gz
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
│           │   └── libslurm*.so*
│           └── VERSION
├── etc/
│   └── slurm/
├── install.sh
├── uninstall.sh
└── README.md
```

### 安装脚本 (install.sh)

```bash
#!/bin/sh
set -e

# 复制文件
cp -r usr/local/slurm /usr/local/
cp -r etc/slurm /etc/ 2>/dev/null || mkdir -p /etc/slurm

# 设置权限
chmod +x /usr/local/slurm/bin/*

# 创建符号链接
for cmd in /usr/local/slurm/bin/*; do
    ln -sf "$cmd" /usr/bin/$(basename "$cmd")
done

# 配置动态库
echo "/usr/local/slurm/lib" > /etc/ld.so.conf.d/slurm.conf
ldconfig 2>/dev/null || true
```

## 故障排查

### 问题 1: APK 包为空

**症状**：`/usr/share/nginx/html/pkgs/slurm-apk/` 目录为空

**排查**：
```bash
# 检查构建日志
docker-compose logs apphub | grep -A 20 "Building SLURM"

# 检查 apk-builder 阶段是否跳过
docker-compose logs apphub | grep "SKIP_SLURM"
```

**可能原因**：
1. SLURM 源码包未复制到容器
2. 编译失败但未报错
3. 打包路径错误

### 问题 2: Backend 从官方源安装

**症状**：Backend 日志显示 "installing from Alpine edge"

**排查**：
```bash
docker logs ai-infra-backend 2>&1 | grep -i "slurm\|apphub"
```

**修复**：移除 Dockerfile 中的备用源逻辑

### 问题 3: 网络连接失败

**症状**：`wget: can't connect to apphub`

**排查**：
```bash
# 检查 Docker 网络
docker network ls
docker network inspect ai-infra-matrix_default

# 测试连通性
docker exec ai-infra-backend wget -O- http://apphub/
```

## 最佳实践

1. **使用固定版本**：`SLURM_VERSION=25.05.4`
2. **验证构建输出**：每个 stage 都检查文件存在
3. **创建符号链接**：`slurm-client-latest-alpine.tar.gz`
4. **强制依赖**：Backend 构建必须依赖 AppHub
5. **详细日志**：输出每个步骤的状态

## 相关文件

- `src/apphub/Dockerfile`: AppHub 构建文件
- `src/apphub/slurm-25.05.4.tar.bz2`: SLURM 源码包
- `src/apphub/scripts/slurm/install.sh`: 安装脚本模板
- `src/backend/Dockerfile`: Backend 构建文件

## 更新日志

- 2025-10-26: 创建文档，设计 AppHub 强制依赖方案
