# Nightingale 监控系统集成总结

## 📅 日期：2025年10月22日

## ✅ 完成的工作

### 1. build.sh 脚本优化

**修改内容**：
- 将硬编码的服务列表改为动态扫描 `src/` 目录
- `get_all_services()` 函数现在会自动识别 `src/` 下所有包含 `Dockerfile` 的子目录
- `get_service_path()` 函数优化为动态查找服务路径

**修改位置**：
- `/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/build.sh`

**效果**：
```bash
./build.sh build --help
# 现在会显示：
# 可用服务: backend frontend jupyterhub nginx saltstack singleuser 
#           gitea backend-init apphub slurm-master test-containers nightingale
```

### 2. Nightingale 项目准备

**克隆项目**：
```bash
git clone --depth 1 https://github.com/ccfos/nightingale.git src/nightingale
```

**项目信息**：
- 仓库：https://github.com/ccfos/nightingale
- 描述：开源监控告警系统（Open-Source Alerting Expert）
- 原开发者：滴滴出行
- 当前维护：中国计算机学会开放原子开源基金会 (CCF ODC)

### 3. Nightingale Dockerfile 创建

**文件路径**：`src/nightingale/Dockerfile`

**配置要点**：
- 基础镜像：`flashcatcloud/nightingale:latest`
- 暴露端口：
  - 17000：HTTP API
  - 17001：HTTPS API
  - 19000：告警引擎
- 健康检查：`http://localhost:17000/api/v1/health`
- 工作目录：`/app`
- 启动命令：`/app/n9e server`

### 4. Nightingale 配置文件调整

**配置文件**：`src/nightingale/etc/config.toml`

**关键修改**：

#### 数据库配置
```toml
[DB]
# 使用 PostgreSQL（从 ai-infra-matrix）
DSN="host=postgres port=5432 user=postgres dbname=nightingale password=your-postgres-password sslmode=disable"
DBType = "postgres"
```

**注意**：数据库名从 `ai-infra-matrix` 改为 `nightingale`，因为 PostgreSQL 不允许标识符中包含连字符。

#### Redis 配置
```toml
[Redis]
Address = "ai-infra-redis:6379"
Password = "your-redis-password"
RedisType = "standalone"
```

#### VictoriaMetrics 写入配置
```toml
[[Pushgw.Writers]]
Url = "http://victoriametrics:8428/api/v1/write"
```

### 5. Docker Compose 集成

**修改文件**：`docker-compose.yml`

**添加的服务**：
```yaml
nightingale:
  build:
    context: ./src/nightingale
    dockerfile: Dockerfile
  image: ${PRIVATE_REGISTRY}ai-infra-nightingale:${IMAGE_TAG}
  container_name: ai-infra-nightingale
  hostname: nightingale
  environment:
    GIN_MODE: release
    TZ: Asia/Shanghai
  ports:
    - "${EXTERNAL_HOST}:${NIGHTINGALE_PORT:-17000}:17000"
    - "${EXTERNAL_HOST}:${NIGHTINGALE_ALERT_PORT:-19000}:19000"
  volumes:
    - ./src/nightingale/etc:/app/etc:ro
    - nightingale_data:/app/data
    - nightingale_logs:/app/logs
  depends_on:
    postgres:
      condition: service_healthy
    redis:
      condition: service_healthy
  healthcheck:
    test: ["CMD", "wget", "-q", "--spider", "http://localhost:17000/api/v1/health"]
    interval: 30s
    timeout: 10s
    retries: 3
    start_period: 60s
  networks:
    - ai-infra-network
  restart: unless-stopped
```

**添加的数据卷**：
```yaml
volumes:
  nightingale_data:
    name: ai-infra-nightingale-data
  nightingale_logs:
    name: ai-infra-nightingale-logs
```

### 6. 环境变量配置

**文件**：`.env`

**新增配置**：
```bash
# Nightingale 监控告警系统配置
NIGHTINGALE_PORT=17000
NIGHTINGALE_ALERT_PORT=19000
```

### 7. 构建和部署

**构建命令**：
```bash
./build.sh build nightingale --force
```

**构建结果**：
- ✅ 镜像：`ai-infra-nightingale:v0.3.6-dev`
- ✅ 构建时间：约 5 秒（使用官方基础镜像）
- ✅ 镜像大小：约 280MB

**启动服务**：
```bash
docker compose up -d nightingale
```

**服务状态**：
- ✅ 容器名：`ai-infra-nightingale`
- ✅ 状态：`Up X seconds (healthy)`
- ✅ HTTP 端口：`192.168.0.200:17000`
- ✅ 告警端口：`192.168.0.200:19000`

## 🔧 遇到的问题和解决方案

### 问题 1：PostgreSQL 数据库名包含连字符

**错误信息**：
```
ERROR: syntax error at or near "-" (SQLSTATE 42601)
CREATE DATABASE ai-infra-matrix ...
```

**原因**：PostgreSQL 不允许在未加引号的标识符中使用连字符

**解决方案**：将数据库名从 `ai-infra-matrix` 改为 `nightingale`

### 问题 2：Redis 认证失败

**错误信息**：
```
failed to ping redis: NOAUTH HELLO must be called with the client already authenticated
```

**原因**：Redis 配置中未提供密码

**解决方案**：在 `config.toml` 中添加 Redis 密码配置：
```toml
Password = "your-redis-password"
```

### 问题 3：build.sh 不识别 nightingale 服务

**错误信息**：
```
[ERROR] 未知服务: nightingale
```

**原因**：`build.sh` 使用硬编码的服务列表

**解决方案**：
- 修改 `get_all_services()` 函数，动态扫描 `src/` 目录
- 修改 `get_service_path()` 函数，支持动态服务路径查找

## 📊 测试结果

### Playwright 测试结果

**测试用例**：`test/e2e/specs/slurm-saltstack-diagnosis.spec.js`

**结果**：✅ 8/8 passed (10.4s)

**关键指标**：
- ✅ SaltStack API：200 OK，状态 "connected"
- ✅ SLURM 节点 API：200 OK（演示模式，空数据）
- ✅ SLURM 作业 API：200 OK（演示模式，空数据）
- ✅ SLURM 摘要 API：200 OK（演示数据）

### Nightingale 服务验证

**Web 界面访问**：
```bash
curl http://192.168.0.200:17000/
# 返回：Nightingale HTML 页面
```

**健康检查**：
```bash
docker compose ps nightingale
# 状态：Up X seconds (healthy)
```

**日志验证**：
```
http server listening on: 0.0.0.0:17000
please view n9e at http://172.18.0.25:17000
```

## 🎯 下一步工作

### 1. 配置数据源连接

需要在 Nightingale 中配置数据源，以便收集监控数据：

- **VictoriaMetrics**（已在配置中）
  - URL: `http://victoriametrics:8428`
  - 用途：时序数据存储

- **Prometheus**（可选）
  - 需要部署 Prometheus 实例
  - 或使用 VictoriaMetrics 的 Prometheus 兼容接口

### 2. 配置告警规则

在 Nightingale Web 界面中配置告警规则：
- SLURM 节点状态监控
- 系统资源监控（CPU、内存、磁盘）
- 服务健康监控

### 3. 集成 SLURM 节点监控

**方案 A：使用 Categraf 采集器**
- 部署在 SLURM 节点上
- 收集 SLURM 指标并推送到 VictoriaMetrics

**方案 B：使用 Prometheus Exporter**
- 部署 SLURM Exporter
- Nightingale 通过 VictoriaMetrics 查询数据

### 4. 配置通知渠道

在 Nightingale 中配置告警通知：
- 邮件通知
- Webhook（可集成到 frontend）
- 钉钉/企业微信等

### 5. 前端集成监控仪表板

**选项 1：iframe 嵌入**
```javascript
// 在 frontend 中嵌入 Nightingale 页面
<iframe 
  src="http://192.168.0.200:17000/dashboard" 
  style="width: 100%; height: 100vh; border: none;"
/>
```

**选项 2：反向代理**
```nginx
location /monitoring/ {
    proxy_pass http://nightingale:17000/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## 📝 配置文件清单

1. **build.sh** - 构建脚本（已优化）
2. **src/nightingale/Dockerfile** - Nightingale 镜像定义
3. **src/nightingale/etc/config.toml** - Nightingale 主配置
4. **docker-compose.yml** - 添加了 nightingale 服务
5. **.env** - 添加了 Nightingale 端口配置

## 🔗 相关链接

- Nightingale GitHub: https://github.com/ccfos/nightingale
- Nightingale 文档: https://flashcat.cloud/docs/
- VictoriaMetrics 文档: https://docs.victoriametrics.com/

## ✨ 总结

成功完成了以下目标：

1. ✅ **build.sh 脚本优化**：动态识别 src 目录下的所有服务
2. ✅ **Nightingale 集成**：克隆、配置、构建、部署
3. ✅ **配置调整**：PostgreSQL、Redis 连接配置
4. ✅ **服务验证**：所有服务正常运行，健康检查通过
5. ✅ **测试通过**：Playwright E2E 测试全部通过

Nightingale 监控系统已成功集成到 AI Infra Matrix 项目中，可以通过 `http://192.168.0.200:17000` 访问 Web 界面进行监控配置和告警管理。
