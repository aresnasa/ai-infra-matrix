# Nightingale 数据源配置修复

## 问题描述

访问 Nightingale 监控页面时，浏览器控制台报错：
```
GET http://192.168.0.200:8080/api/n9e/proxy/undefined/api/v1/metadata
Error: cannot convert undefined to int64
```

## 根本原因

Nightingale 是一个监控系统的**告警引擎和前端界面**，它本身不存储时序数据，需要连接到一个数据源（如 Prometheus）来查询监控指标。

当 Nightingale 数据库中没有配置任何数据源时：
1. 前端尝试获取数据源列表 `/api/n9e/datasource/brief` 返回空（`null`）
2. JavaScript 代码获取不到数据源 ID，变成 `undefined`
3. 构造 API 请求时使用 `undefined` 作为数据源 ID: `/api/n9e/proxy/undefined/api/v1/metadata`
4. 后端尝试将 `undefined` 字符串转换为 int64 失败，报错

## 解决方案

### 添加默认数据源

在 Nightingale 数据库的 `datasource` 表中插入一个默认的 Prometheus 数据源：

```sql
INSERT INTO datasource (
  id, 
  name, 
  description, 
  category, 
  plugin_id, 
  plugin_type, 
  plugin_type_name, 
  cluster_name, 
  settings, 
  status, 
  http, 
  auth, 
  is_default, 
  created_at, 
  created_by, 
  updated_at, 
  updated_by, 
  identifier
)
VALUES (
  1,
  'Default Prometheus',
  'Built-in Prometheus data source for AI-Infra-Matrix monitoring',
  'prometheus',
  0,
  'prometheus',
  'Prometheus',
  'Default',
  '{}',
  'enabled',
  '{"url": "http://nightingale:17000", "timeout": 30, "dial_timeout": 3}',
  '{}',
  true,
  extract(epoch from now())::bigint,
  'system',
  extract(epoch from now())::bigint,
  'system',
  'default-prometheus'
);
```

### 自动初始化脚本

创建了初始化脚本 `scripts/init-nightingale-datasource.sh`，可以在部署时自动检查并创建数据源：

```bash
# 运行初始化脚本
./scripts/init-nightingale-datasource.sh
```

脚本功能：
- ✅ 检查数据库连接
- ✅ 检查数据源是否存在
- ✅ 如果不存在则自动创建默认数据源
- ✅ 如果存在则跳过（幂等操作）
- ✅ 验证并显示当前数据源列表

## 验证修复

### 手动验证

1. **检查数据源 API**：
```bash
TOKEN=$(curl -s -X POST 'http://192.168.0.200:8080/api/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

curl -s "http://192.168.0.200:8080/api/n9e/datasource/brief" \
  -H "Cookie: auth_token=$TOKEN" | jq .
```

预期输出：
```json
{
  "dat": [
    {
      "id": 1,
      "name": "Default Prometheus",
      "identifier": "default-prometheus",
      "plugin_type": "prometheus",
      "status": "enabled",
      "is_default": true,
      ...
    }
  ],
  "err": ""
}
```

2. **测试 metadata API**（之前报错的接口）：
```bash
curl -s "http://192.168.0.200:8080/api/n9e/proxy/1/api/v1/metadata" \
  -H "Cookie: auth_token=$TOKEN" | head -20
```

应该返回 HTML 响应，而不是错误信息。

### 自动化测试

运行 Playwright E2E 测试：

```bash
BASE_URL=http://192.168.0.200:8080 npx playwright test \
  test/e2e/specs/verify-datasource-fix.spec.js \
  --config=test/e2e/playwright.config.js
```

测试验证：
- ✅ 无 API 错误（状态码 >= 400）
- ✅ 无 URL 中包含 `undefined` 的请求
- ✅ 无浏览器控制台 undefined 错误
- ✅ 监控页面正常加载

## 数据源配置说明

### 当前配置

- **数据源类型**: Prometheus
- **数据源 URL**: `http://nightingale:17000`
- **用途**: Nightingale 自身的 metrics 端点

### 注意事项

1. **内置 metrics 端点**：
   - Nightingale 在 `http://nightingale:17000/metrics` 提供 Prometheus 格式的指标
   - 这是 Nightingale 服务自身的运行指标（Go runtime metrics）
   - **不包含业务监控数据**

2. **添加外部 Prometheus**：
   如果需要监控其他服务，可以在 Nightingale UI 中添加外部 Prometheus 数据源：
   - 登录 Nightingale
   - 导航到 "Integrations" -> "Data sources"
   - 点击 "Add data source"
   - 选择 "Prometheus"
   - 配置 URL 和认证信息

3. **多数据源支持**：
   - Nightingale 支持配置多个数据源
   - 可以设置默认数据源（`is_default = true`）
   - 在创建仪表板和告警规则时选择数据源

## 相关文件

- 初始化脚本: `scripts/init-nightingale-datasource.sh`
- E2E 测试: `test/e2e/specs/verify-datasource-fix.spec.js`
- Nightingale 配置: `src/nightingale/etc/config.toml`
- 数据库: PostgreSQL `nightingale` database, `datasource` table

## 后续改进

### 可选：部署独立 Prometheus

如果需要完整的监控功能，建议部署独立的 Prometheus 实例：

```yaml
# docker-compose.yml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: ai-infra-prometheus
    volumes:
      - ./config/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    networks:
      - ai-infra-network

volumes:
  prometheus-data:
```

然后在 Nightingale 中配置数据源 URL 为 `http://prometheus:9090`。

## 总结

✅ **问题已解决**：
- 添加了默认 Prometheus 数据源（ID=1）
- API 不再出现 `undefined` 错误
- Nightingale 前端可以正常查询数据源列表
- 创建了自动化初始化脚本和验证测试

⚠️ **限制**：
- 当前数据源指向 Nightingale 自身的 metrics 端点
- 只包含 Nightingale 服务的运行指标，不包含业务监控数据
- 如需完整监控功能，建议部署独立 Prometheus 并添加采集配置
