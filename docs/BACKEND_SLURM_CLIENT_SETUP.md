# Backend SLURM 客户端安装指南

## 快速开始

### 方法 1: 自动化流程（推荐）

```bash
# 1. 启动 AppHub 容器
docker-compose up -d apphub

# 2. 构建并上传 SLURM Alpine 客户端包
./scripts/build-slurm-client-alpine.sh

# 3. 重新构建 Backend（自动从 AppHub 下载安装 SLURM 客户端）
./build.sh build backend --force

# 4. 重启 Backend 服务
docker-compose up -d --force-recreate backend

# 5. 验证安装
docker-compose exec backend sinfo --version
```

### 方法 2: 手动流程

如果自动构建失败或需要手动控制：

```bash
# 1. 下载预编译的 SLURM 客户端包（如果有）
# 或跳过此步骤，Backend 会降级到演示数据模式

# 2. 手动上传到 AppHub
docker cp pkgs/slurm-apk/slurm-client-*.tar.gz \
  ai-infra-apphub:/usr/share/nginx/html/pkgs/slurm-apk/

# 3. 创建 latest 符号链接
docker exec ai-infra-apphub sh -c \
  "cd /usr/share/nginx/html/pkgs/slurm-apk && \
   ln -sf slurm-client-23.11.10-alpine.tar.gz slurm-client-latest-alpine.tar.gz"

# 4. 重新构建 Backend
./build.sh build backend --force
```

## Backend Dockerfile 工作原理

Backend Dockerfile 已经集成了自动从 AppHub 下载和安装 SLURM 客户端的逻辑：

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

### 工作流程

1. **多 URL 尝试**: 依次尝试 3 个 AppHub URL
   - `http://apphub/...` (Docker 内网服务名)
   - `http://ai-infra-apphub/...` (完整容器名)
   - `http://192.168.0.200:8081/...` (外网 IP，如果配置了)

2. **下载验证**: 检查下载的文件是否存在且非空

3. **自动安装**: 解压并运行 `install.sh` 脚本
   - 复制二进制文件到 `/usr/local/slurm/bin/`
   - 创建符号链接到 `/usr/bin/`
   - 配置环境变量

4. **安装验证**: 运行 `sinfo --version` 检查

5. **降级机制**: 如果下载失败，自动降级到演示数据模式

## 验证安装

### 在 Backend 容器中检查

```bash
# 进入容器
docker-compose exec backend bash

# 检查 SLURM 客户端版本
sinfo --version

# 查看安装的命令
ls -la /usr/local/slurm/bin/

# 查看环境变量
source /etc/profile
echo $SLURM_HOME
echo $PATH | grep slurm

# 测试连接 SLURM master（如果配置了）
export SLURM_CONF=/etc/slurm/slurm.conf
sinfo -h
squeue -h
```

### 检查 Docker 构建日志

```bash
# 构建时查看安装过程
docker-compose build backend 2>&1 | grep -A 20 "Installing SLURM"
```

应该看到类似输出：

```
>>> Installing SLURM client tools from AppHub...
  ✓ Downloaded from: http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz
>>> Extracting and installing SLURM client...
Installing SLURM client tools...
  ✓ SLURM client installed successfully
  ✓ SLURM version: slurm 23.11.10
```

## 故障排查

### 问题 1: 下载失败

**现象**:
```
  ⚠ SLURM client download failed, will use demo data
```

**原因**: AppHub 容器未运行或包不存在

**解决**:
```bash
# 检查 AppHub 容器状态
docker ps | grep apphub

# 检查包是否存在
docker exec ai-infra-apphub ls -l /usr/share/nginx/html/pkgs/slurm-apk/

# 如果包不存在，运行构建脚本
./scripts/build-slurm-client-alpine.sh

# 手动上传（如果有预构建包）
docker cp pkgs/slurm-apk/slurm-client-*.tar.gz \
  ai-infra-apphub:/usr/share/nginx/html/pkgs/slurm-apk/
```

### 问题 2: 安装脚本失败

**现象**: 下载成功但 `sinfo --version` 不可用

**原因**: install.sh 执行失败或环境变量未设置

**解决**:
```bash
# 进入容器手动测试
docker-compose exec backend bash

# 下载并手动安装
cd /tmp
wget http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz
tar xzf slurm-client-latest-alpine.tar.gz
./install.sh

# 检查安装结果
ls -la /usr/local/slurm/bin/
cat /etc/profile | grep SLURM
source /etc/profile
sinfo --version
```

### 问题 3: 网络连接问题

**现象**: wget 超时

**原因**: Docker 网络配置或 AppHub 未在同一网络

**解决**:
```bash
# 检查网络
docker-compose exec backend ping -c 3 apphub

# 检查 Docker 网络
docker network ls
docker network inspect ai-infra-matrix_default

# 确认 AppHub 在同一网络
docker inspect ai-infra-apphub | grep NetworkMode
```

### 问题 4: 使用演示数据模式

**现象**: Backend 正常运行但使用假数据

**验证**:
```bash
# 检查 Backend 日志
docker-compose logs backend | grep -i "demo\|slurm"

# 检查 API 响应
curl http://localhost:8080/api/slurm/summary
```

如果返回 `"demo": true`，说明正在使用演示数据。

**解决**: 按照上述步骤安装真实的 SLURM 客户端。

## 构建 SLURM Alpine 客户端包

如果需要重新构建或更新版本：

```bash
# 设置 SLURM 版本（可选）
export SLURM_VERSION=23.11.10

# 运行构建脚本
./scripts/build-slurm-client-alpine.sh

# 脚本会：
# 1. 在 Alpine 容器中编译 SLURM 客户端
# 2. 打包为 tar.gz
# 3. 自动上传到 AppHub
# 4. 创建 latest 符号链接
```

**构建时间**: 10-30 分钟（取决于网络和 CPU）

**输出位置**: `./pkgs/slurm-apk/slurm-client-<version>-alpine.tar.gz`

## 环境变量配置

Backend 安装 SLURM 客户端后会自动设置以下环境变量（在 `/etc/profile`）：

```bash
export SLURM_HOME=/usr/local/slurm
export PATH=$SLURM_HOME/bin:$PATH
export LD_LIBRARY_PATH=$SLURM_HOME/lib:$LD_LIBRARY_PATH
```

在容器中使用 SLURM 命令前需要 source：

```bash
source /etc/profile
sinfo --version
```

## 与 SLURM Master 集成

Backend 容器通过以下方式连接到 SLURM master：

### 方法 1: 环境变量

```bash
# 在 docker-compose.yml 或 .env 中配置
SLURM_CONF=/etc/slurm/slurm.conf
SLURM_CONF_SERVER=slurm-master

# 或在容器中设置
docker-compose exec backend sh -c 'export SLURM_CONF_SERVER=slurm-master && sinfo'
```

### 方法 2: SSH 连接

如果 SLURM master 在远程主机：

```bash
# Backend 通过 SSH 执行命令
docker-compose exec backend ssh slurm-master sinfo
docker-compose exec backend ssh slurm-master squeue
```

### 方法 3: 配置文件

挂载 `slurm.conf` 到 Backend 容器：

```yaml
# docker-compose.yml
services:
  backend:
    volumes:
      - ./config/slurm.conf:/etc/slurm/slurm.conf:ro
```

## 完整流程示例

```bash
#!/bin/bash
# 完整的 Backend SLURM 客户端安装流程

set -e

echo "1. 启动 AppHub..."
docker-compose up -d apphub
sleep 5

echo "2. 构建 SLURM Alpine 客户端包..."
./scripts/build-slurm-client-alpine.sh

echo "3. 验证包已上传到 AppHub..."
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/

echo "4. 重新构建 Backend..."
./build.sh build backend --force

echo "5. 启动 Backend..."
docker-compose up -d --force-recreate backend
sleep 10

echo "6. 验证 SLURM 客户端安装..."
docker-compose exec backend bash -c "source /etc/profile && sinfo --version"

echo "✅ 完成！"
```

## 相关文档

- [SLURM AppHub 安装指南](./SLURM_APPHUB_INSTALLATION.md) - 详细的构建和部署流程
- [AppHub 使用指南](./APPHUB_USAGE_GUIDE.md) - AppHub 包管理
- [Backend 构建指南](./BUILD_AND_TEST_GUIDE.md) - Backend 容器构建
