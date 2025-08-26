# 生产环境 Backend-Init 问题修复指南

## 问题描述

在生产环境中，`ai-infra-backend-init` 服务可能遇到以下错误：

```log
time="2025-08-27T00:20:44+08:00" level=warning msg="No .env file found"
failed to connect to `host=postgres user=postgres database=postgres`: failed SASL auth (FATAL: password authentication failed for user "postgres")
```

## 根本原因

1. **环境文件挂载缺失**: backend-init 容器内部无法访问 `.env` 文件
2. **PostgreSQL 密码认证**: 数据库密码配置不一致
3. **重启策略不当**: backend-init 应该是一次性服务，不应该重启

## 解决方案

### 自动修复（推荐）

使用最新的 `build.sh` 脚本自动生成正确的生产配置：

```bash
# 1. 重新生成生产配置
./build.sh prod-generate <registry> <tag>

# 2. 验证配置
./scripts/fix-production-deployment.sh

# 3. 启动生产环境
./build.sh --force prod-up <registry> <tag>
```

### 修复内容

最新的 `build.sh` 脚本会自动应用以下修复：

1. **环境文件挂载修复**:

   ```yaml
   backend-init:
     volumes:
       - ./.env.prod:/app/.env:ro
     environment:
       - ENV_FILE=/app/.env
   ```

2. **重启策略修复**:

   ```yaml
   backend-init:
     restart: 'no'
   ```

3. **环境变量引用修复**:

   ```yaml
   backend-init:
     env_file:
       - .env.prod
   ```

### 手动修复（如需要）

如果无法使用自动修复，可以手动修改 `docker-compose.prod.yml`：

```yaml
services:
  backend-init:
    # ... 其他配置 ...
    env_file:
      - .env.prod
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${POSTGRES_USER:-postgres}
      - DB_PASSWORD=${POSTGRES_PASSWORD:-postgres}
      - REDIS_HOST=redis
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - TZ=Asia/Shanghai
      - ENV_FILE=/app/.env  # 添加这行
    volumes:
      - ./.env.prod:/app/.env:ro  # 添加这行
    restart: 'no'  # 确保是 'no' 而不是 'unless-stopped'
    # ... 其他配置 ...
```

## 验证修复

### 1. 检查配置文件

```bash
# 检查 backend-init 服务配置
grep -A 30 "backend-init:" docker-compose.prod.yml

# 应该看到:
# - volumes: - ./.env.prod:/app/.env:ro
# - restart: 'no'
# - ENV_FILE=/app/.env 环境变量
```

### 2. 检查环境文件

```bash
# 确保 .env.prod 文件存在且包含正确的密码
ls -la .env.prod
grep POSTGRES_PASSWORD .env.prod
```

### 3. 启动并验证

```bash
# 启动生产环境
./build.sh --force prod-up <registry> <tag>

# 检查 backend-init 状态（应该是 Exited 0）
docker ps -a | grep backend-init

# 检查 backend-init 日志（应该显示 "Initialization completed successfully!"）
docker logs ai-infra-backend-init --tail 10

# 检查 backend 服务状态（应该是 Up healthy）
docker ps | grep backend
```

## 预期结果

修复后，backend-init 服务应该：

1. ✅ **成功读取环境文件**: 不再显示 "No .env file found"
2. ✅ **连接数据库成功**: 使用正确的 PostgreSQL 密码
3. ✅ **完成初始化**: 显示 "Initialization completed successfully!"
4. ✅ **正常退出**: 容器状态为 "Exited (0)"，不会重启
5. ✅ **允许后端启动**: backend 服务能够检测到 backend-init 完成并启动

## 故障排除

### 如果仍然出现 "No .env file found"

```bash
# 检查挂载配置
docker inspect ai-infra-backend-init | grep -A 10 Mounts

# 检查容器内部文件
docker exec ai-infra-backend-init ls -la /app/.env
```

### 如果仍然出现密码认证失败

```bash
# 检查环境变量是否正确传递
docker exec ai-infra-backend-init env | grep POSTGRES

# 检查数据库服务状态
docker logs ai-infra-postgres
```

### 如果 backend-init 仍然重启

```bash
# 检查重启策略
docker inspect ai-infra-backend-init | grep RestartPolicy

# 确保 docker-compose.prod.yml 中 restart: 'no'
grep -A 30 "backend-init:" docker-compose.prod.yml | grep restart
```

## 相关文件

- `build.sh`: 主构建脚本，包含生产配置生成逻辑
- `fix_backend_init_restart.py`: backend-init 重启策略修复脚本
- `fix_backend_init_volumes.py`: backend-init 环境文件挂载修复脚本
- `scripts/fix-production-deployment.sh`: 生产环境部署验证脚本

## 版本要求

此修复适用于 AI-Infra-Matrix v0.3.5 及以上版本。如果使用较早版本，请升级 `build.sh` 脚本。
