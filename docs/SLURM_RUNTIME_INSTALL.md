# SLURM 运行时安装说明

## 概述

为了解决 Docker 构建时无法访问 AppHub 的问题，我们将 SLURM 客户端的安装改为**运行时安装**模式。

## 工作流程

### 1. 构建阶段（Dockerfile）
- **AppHub**: 编译 SLURM 二进制文件并打包
- **Backend**: 仅复制 `install-slurm-runtime.sh` 脚本，不安装 SLURM

### 2. 运行阶段（容器启动）
- Backend 容器启动时，`wait-for-db.sh` 会自动调用 `install-slurm-runtime.sh`
- 脚本会探测 AppHub 地址并下载安装 SLURM 客户端
- 如果 AppHub 不可用，会跳过安装并允许容器正常启动

## 文件说明

### src/backend/install-slurm-runtime.sh
运行时安装脚本，负责：
- 检查 SLURM 是否已安装
- 探测可用的 AppHub URL
- 从 AppHub 下载并执行安装脚本
- 失败时优雅降级（允许容器启动）

### src/backend/Dockerfile
修改点：
- 移除构建时的 SLURM 安装逻辑（旧代码 136-196 行）
- 添加运行时安装脚本复制（新代码 136-138 行）
- 修改 `wait-for-db.sh` 集成 SLURM 安装（新代码 165-168 行）

### src/apphub/packages/install-slurm.sh
SLURM 安装脚本（由 AppHub 提供）：
- 检测系统架构（x86_64/arm64）
- 从 AppHub 下载对应架构的 SLURM 二进制文件
- 安装到 `/usr/local/slurm/`
- 配置环境变量和动态库路径

## 使用方法

### 正常启动流程
```bash
# 1. 构建 AppHub（包含 SLURM 二进制）
./build.sh build apphub --force

# 2. 启动 AppHub
docker-compose up -d apphub

# 3. 构建 Backend（无需 AppHub 运行）
./build.sh build backend

# 4. 启动 Backend（自动安装 SLURM）
docker-compose up -d backend
```

### 手动安装 SLURM
如果容器启动时 AppHub 不可用，可以后续手动安装：

```bash
# 确保 AppHub 运行
docker-compose up -d apphub

# 在 Backend 容器内手动安装
docker exec ai-infra-backend /install-slurm-runtime.sh
```

### 验证安装
```bash
# 检查 SLURM 命令是否可用
docker exec ai-infra-backend sinfo --version

# 查看已安装的 SLURM 工具
docker exec ai-infra-backend ls -lh /usr/local/slurm/bin/
```

## 优势

1. **解耦构建依赖**: Backend 构建不依赖 AppHub 运行
2. **优雅降级**: AppHub 不可用时容器仍可启动
3. **灵活性**: 可随时手动安装或更新 SLURM
4. **调试友好**: 安装失败不影响服务启动

## 故障排除

### SLURM 未安装
**症状**: `sinfo: command not found`

**解决**:
```bash
# 检查 AppHub 是否运行
docker ps | grep apphub

# 检查 AppHub 是否有 SLURM 二进制
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-binaries/

# 手动安装
docker exec ai-infra-backend /install-slurm-runtime.sh
```

### AppHub 连接失败
**症状**: `AppHub is not accessible`

**解决**:
```bash
# 检查网络连接
docker exec ai-infra-backend ping -c 3 apphub

# 检查 AppHub 服务
curl http://localhost:8081/packages/

# 重启 AppHub
docker-compose restart apphub
```

### 架构不匹配
**症状**: `No binaries found for architecture`

**解决**:
```bash
# 检查当前架构
docker exec ai-infra-backend uname -m

# 检查 AppHub 支持的架构
docker exec ai-infra-apphub ls /usr/share/nginx/html/pkgs/slurm-binaries/

# 重新构建 AppHub（确保编译了对应架构）
./build.sh build apphub --force
```

## 技术细节

### 探测逻辑
脚本按顺序尝试以下 URL：
1. `http://apphub` - Docker Compose 服务名
2. `http://ai-infra-apphub` - 完整容器名
3. `http://localhost:8081` - 本地映射端口
4. `http://192.168.0.200:8081` - 自定义地址

### 安装位置
- 二进制文件: `/usr/local/slurm/bin/`
- 库文件: `/usr/local/slurm/lib/`
- 配置文件: `/etc/slurm/`
- 符号链接: `/usr/local/bin/s*` → `/usr/local/slurm/bin/s*`

### 环境变量
安装后会在 `/etc/profile` 添加：
```bash
export SLURM_HOME=/usr/local/slurm
export PATH=$SLURM_HOME/bin:$PATH
export LD_LIBRARY_PATH=$SLURM_HOME/lib:$LD_LIBRARY_PATH
```

## 版本信息

- SLURM 版本: 25.05.4
- 支持架构: x86_64, arm64
- 安装模式: 二进制直接安装（无需包管理器）
