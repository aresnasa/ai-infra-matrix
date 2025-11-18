# AppHub SLURM APK 构建 - 快速开始

## 前置条件

```bash
# 1. 下载 SLURM 源码包
cd src/apphub
wget https://download.schedmd.com/slurm/slurm-25.05.4.tar.bz2

# 2. 确认文件存在
ls -lh slurm-25.05.4.tar.bz2
```

## 构建 AppHub（包含 APK 包）

```bash
# 方法 1: 使用 build.sh（推荐）
./build.sh build apphub --force

# 方法 2: 使用 docker-compose
docker-compose build --no-cache apphub

# 方法 3: 使用 docker build
docker build -t ai-infra-apphub:latest \
  --build-arg SLURM_VERSION=25.05.4 \
  -f src/apphub/Dockerfile \
  src/apphub
```

## 启动 AppHub

```bash
docker-compose up -d apphub
```

## 验证 APK 包

```bash
# 检查包是否存在
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/

# 应该看到：
# slurm-client-25.05.4-alpine.tar.gz
# slurm-client-latest-alpine.tar.gz -> slurm-client-25.05.4-alpine.tar.gz

# 测试下载
curl -I http://localhost:8081/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz
```

## 重新构建 Backend

```bash
# Backend Dockerfile 已配置自动从 AppHub 下载 APK 包
./build.sh build backend --force

# 启动 Backend
docker-compose up -d --force-recreate backend

# 验证 SLURM 客户端安装
docker-compose exec backend bash -c "source /etc/profile && sinfo --version"
```

## 构建时间

- **AppHub 完整构建**: 18-52 分钟
  - Stage 1 (DEB): 5-15 分钟
  - Stage 2 (RPM): 2-5 分钟
  - Stage 3 (APK): 10-30 分钟 ← 新增
  - Stage 4 (Nginx): 1-2 分钟

## 跳过 APK 构建（如果需要）

如果不需要 APK 包，可以移除 SLURM 源码包：

```bash
# 移除源码包（Stage 3 会自动跳过）
rm src/apphub/slurm-*.tar.bz2

# 构建（跳过 APK）
docker-compose build apphub
```

## 故障排查

### 问题: 源码包不存在

```bash
# 下载 SLURM 源码
cd src/apphub
wget https://download.schedmd.com/slurm/slurm-25.05.4.tar.bz2

# 或从 GitHub
wget https://github.com/SchedMD/slurm/archive/refs/tags/slurm-25-05-4.tar.gz \
  -O slurm-25.05.4.tar.bz2
```

### 问题: 构建超时

```bash
# 增加 Docker 构建超时
export COMPOSE_HTTP_TIMEOUT=3600

# 或分阶段构建
docker build --target apk-builder -t slurm-apk-builder -f src/apphub/Dockerfile src/apphub
docker build -t ai-infra-apphub:latest -f src/apphub/Dockerfile src/apphub
```

### 问题: Backend 下载失败

```bash
# 检查 AppHub 运行状态
docker ps | grep apphub

# 检查包是否存在
docker exec ai-infra-apphub ls -l /usr/share/nginx/html/pkgs/slurm-apk/

# 测试网络连接
docker-compose exec backend ping -c 3 apphub
```

## 完整流程示例

```bash
#!/bin/bash
set -e

echo "1. 下载 SLURM 源码..."
cd src/apphub
if [ ! -f slurm-25.05.4.tar.bz2 ]; then
    wget https://download.schedmd.com/slurm/slurm-25.05.4.tar.bz2
fi
cd ../..

echo "2. 构建 AppHub（包含 APK 包）..."
./build.sh build apphub --force

echo "3. 启动 AppHub..."
docker-compose up -d apphub
sleep 5

echo "4. 验证 APK 包..."
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/

echo "5. 重新构建 Backend..."
./build.sh build backend --force

echo "6. 启动 Backend..."
docker-compose up -d --force-recreate backend
sleep 10

echo "7. 验证 SLURM 客户端..."
docker-compose exec backend bash -c "source /etc/profile && sinfo --version"

echo "✅ 完成！"
```

## 访问地址

- **AppHub**: http://localhost:8081
- **SLURM APK 包**: http://localhost:8081/pkgs/slurm-apk/
- **直接下载**: http://localhost:8081/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz

## 更多信息

- 详细文档: [docs/APPHUB_SLURM_APK_BUILD.md](./APPHUB_SLURM_APK_BUILD.md)
- Backend 集成: [docs/BACKEND_SLURM_CLIENT_SETUP.md](./BACKEND_SLURM_CLIENT_SETUP.md)
- 实现总结: [docs/BACKEND_SLURM_FROM_APPHUB_SUMMARY.md](./BACKEND_SLURM_FROM_APPHUB_SUMMARY.md)
