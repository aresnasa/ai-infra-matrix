# SaltStack 配置修复 - 快速参考

## 问题症状

```bash
# ❌ 修复前
$ docker exec ai-infra-backend env | grep SALTSTACK_MASTER_URL
SALTSTACK_MASTER_URL=  # 空值，无法连接
```

## 修复结果

```bash
# ✅ 修复后
$ docker exec ai-infra-backend env | grep SALTSTACK_MASTER_URL
SALTSTACK_MASTER_URL=http://saltstack:8002  # 正确的完整URL

# ✅ 连接测试成功
$ docker exec ai-infra-backend sh -c 'curl -s ${SALTSTACK_MASTER_URL}/'
{"return": "Welcome", "clients": ["local", "local_async", "local_batch", ...]}
```

## 修改的文件

1. **`.env.example`** - 添加默认 URL 值
   ```bash
   SALTSTACK_MASTER_URL=http://saltstack:8002
   ```

2. **`docker-compose.yml`** - 设置默认值
   ```yaml
   SALTSTACK_MASTER_URL: "${SALTSTACK_MASTER_URL:-http://saltstack:8002}"
   ```

3. **`build.sh`** - 增强渲染和初始化逻辑
   - `render_env_template_enhanced()`: 自动拼装 URL
   - `setup_saltstack_defaults()`: 自动填充空值

## 如何应用修复

### 方法 1: 重新构建（推荐）

```bash
# 重新构建并启动所有服务
./build.sh build-all

# 或者只重启后端
docker compose up -d backend
```

### 方法 2: 手动更新环境变量

```bash
# 1. 更新 .env 文件
sed -i.bak 's|^SALTSTACK_MASTER_URL=$|SALTSTACK_MASTER_URL=http://saltstack:8002|' .env

# 2. 重启后端容器
docker compose up -d backend

# 3. 验证
docker exec ai-infra-backend env | grep SALTSTACK_MASTER_URL
```

## 验证步骤

```bash
# 1. 检查环境变量
docker exec ai-infra-backend env | grep -E "SALT|saltstack" | sort

# 2. 测试 API 连接
docker exec ai-infra-backend sh -c 'curl -s ${SALTSTACK_MASTER_URL}/'

# 3. 检查后端日志
docker logs ai-infra-backend 2>&1 | grep -i salt
```

## 配置说明

### 完整 URL 配置（推荐）

```bash
# .env 文件
SALTSTACK_MASTER_URL=http://saltstack:8002
SALT_API_SCHEME=http
SALT_MASTER_HOST=saltstack
SALT_API_PORT=8002
```

### 分离组件配置（降级方案）

```bash
# .env 文件
SALTSTACK_MASTER_URL=  # 留空，后端自动拼装
SALT_API_SCHEME=http
SALT_MASTER_HOST=saltstack
SALT_API_PORT=8002
```

## 常见问题

### Q1: 为什么需要 SALTSTACK_MASTER_URL？

**A**: 后端代码优先使用完整 URL，避免拼装错误，提高配置清晰度。

### Q2: 如果改变 SaltStack 端口怎么办？

**A**: 同时更新两处配置：

```bash
# .env 文件
SALT_API_PORT=9999
SALTSTACK_MASTER_URL=http://saltstack:9999
```

### Q3: HTTPS 环境如何配置？

**A**: 使用 HTTPS 协议和对应端口：

```bash
# .env 文件
SALT_API_SCHEME=https
SALT_API_PORT=8443
SALTSTACK_MASTER_URL=https://saltstack:8443
```

## 详细文档

查看完整修复报告: [SALTSTACK_URL_FIX.md](./SALTSTACK_URL_FIX.md)

---

**修复日期**: 2025年10月11日  
**影响版本**: v0.3.7+  
**相关服务**: backend, saltstack
