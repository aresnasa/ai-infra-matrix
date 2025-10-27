# SLURM 二进制构建方案

## 概述

由于 Alpine Linux 使用 musl libc，不支持 SLURM 编译所需的 `cpu_set_t` 类型（glibc 特有），我们改用 Ubuntu 22.04 编译 SLURM 二进制文件，然后在 AppHub 中按架构提供。

## 架构

```
┌─────────────────────────────────────────────┐
│  AppHub (Nginx)                             │
│  ┌─────────────────────────────────────┐   │
│  │ /pkgs/slurm-binaries/               │   │
│  │   ├── x86_64/                       │   │
│  │   │   ├── bin/ (sinfo, squeue...)   │   │
│  │   │   ├── lib/ (libslurm*.so)       │   │
│  │   │   └── VERSION                   │   │
│  │   └── arm64/                        │   │
│  │       ├── bin/                      │   │
│  │       ├── lib/                      │   │
│  │       └── VERSION                   │   │
│  └─────────────────────────────────────┘   │
│  /packages/install-slurm.sh                 │
└─────────────────────────────────────────────┘
                      ↓
         HTTP 下载二进制文件
                      ↓
┌─────────────────────────────────────────────┐
│  Backend (Alpine + 运行时依赖)              │
│  /usr/local/bin/ → SLURM 客户端工具         │
│  /usr/local/slurm/lib/ → SLURM 库文件       │
└─────────────────────────────────────────────┘
```

## 构建流程

### 1. AppHub 构建 (Dockerfile)

#### Stage 3: binary-builder (Ubuntu 22.04)
- 使用 Ubuntu 22.04 (glibc) 编译 SLURM
- 检测架构 (x86_64/arm64)
- 编译完整的 SLURM
- 收集二进制文件到 `/out/packages/{arch}/bin/`
- 收集库文件到 `/out/packages/{arch}/lib/`
- 创建版本文件 `VERSION`

#### Stage 5: AppHub 最终镜像
- 从 binary-builder 复制 `/out/packages/` → `/usr/share/nginx/html/pkgs/slurm-binaries/`
- 复制 `packages/install-slurm.sh` → `/usr/share/nginx/html/packages/`
- 通过 Nginx 提供 HTTP 访问

### 2. Backend 安装

#### 安装脚本 (install-slurm.sh)
1. 检测架构 (x86_64/arm64)
2. 从 AppHub 下载对应架构的二进制文件
3. 安装到 `/usr/local/slurm/bin/`
4. 安装库到 `/usr/local/slurm/lib/`
5. 创建符号链接到 `/usr/local/bin/`
6. 配置环境变量

## 使用方法

### 构建 AppHub
```bash
./build.sh build apphub --force
```

### 启动 AppHub
```bash
docker-compose up -d apphub
```

### 验证 SLURM 二进制文件
```bash
# 查看可用架构
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-binaries/

# 查看特定架构的二进制文件
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-binaries/x86_64/bin/
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-binaries/x86_64/lib/

# 通过 HTTP 访问
curl http://localhost:8081/pkgs/slurm-binaries/x86_64/bin/
curl http://localhost:8081/packages/install-slurm.sh
```

### 构建 Backend
```bash
# 确保 AppHub 正在运行
docker-compose up -d apphub

# 构建 Backend（会自动从 AppHub 安装 SLURM）
./build.sh build backend --force
```

### 验证 Backend 中的 SLURM
```bash
docker exec ai-infra-backend sinfo --version
docker exec ai-infra-backend which sinfo
docker exec ai-infra-backend ls -l /usr/local/slurm/bin/
```

## 优势

1. **架构无关**: 同时支持 x86_64 和 arm64
2. **简化构建**: 不需要复杂的包管理器配置
3. **兼容性**: 使用 glibc 编译，避免 musl libc 的兼容性问题
4. **灵活部署**: 二进制文件可以直接复制，无需安装包
5. **易于调试**: 构建和安装过程清晰可见

## 文件列表

- `src/apphub/Dockerfile` - AppHub 构建文件（包含 binary-builder stage）
- `src/apphub/packages/install-slurm.sh` - SLURM 安装脚本
- `src/backend/Dockerfile` - Backend 构建文件（使用安装脚本）

## 故障排查

### 问题: Backend 构建失败，提示 AppHub 不可用
```bash
# 确保 AppHub 正在运行
docker ps | grep apphub

# 如果未运行，启动 AppHub
docker-compose up -d apphub

# 等待几秒让 AppHub 启动
sleep 5

# 验证 AppHub 可访问
curl http://localhost:8081/packages/install-slurm.sh
```

### 问题: SLURM 二进制文件不存在
```bash
# 检查 AppHub 中的文件
docker exec ai-infra-apphub ls -lR /usr/share/nginx/html/pkgs/slurm-binaries/

# 如果文件不存在，重新构建 AppHub
docker-compose build --no-cache apphub
```

### 问题: 架构不匹配
```bash
# 检查当前架构
uname -m

# 确保 AppHub 中有对应架构的二进制文件
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-binaries/$(uname -m)/bin/
```

## 版本管理

SLURM 版本在 `src/apphub/Dockerfile` 中定义：

```dockerfile
ARG SLURM_VERSION=25.05.4
```

更新版本时：
1. 修改 `SLURM_VERSION`
2. 确保 `slurm-25.05.4.tar.bz2` 存在于 `src/apphub/`
3. 重新构建 AppHub
