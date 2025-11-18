# Nightingale 监控页面 404 错误修复指南

## 问题描述

访问 http://192.168.0.200:8080/monitoring 时，页面虽然能加载，但 Nightingale iframe 中显示：

```
404
The page you visited does not exist!
Back to home
```

## 根本原因

### 问题分析

1. **iframe 加载的路径**: `/nightingale/` (根路径)
2. **Nightingale 行为**: 根路径没有默认页面，显示 404 错误页面
3. **用户体验**: 用户看到 404 错误，不知道如何使用监控系统

### 架构说明

```
浏览器访问: http://192.168.0.200:8080/monitoring
    ↓
AI-Infra-Matrix Frontend (React)
    ↓
渲染 MonitoringPage 组件
    ↓
iframe src="/nightingale/" ← 这里是问题所在
    ↓
Nginx 代理到 Nightingale
    ↓
Nightingale 根路径 → 404 页面
```

## 解决方案

### 修复方法

将 iframe 的默认路径从 `/nightingale/` 改为 `/nightingale/metrics`，直接显示 Nightingale 的指标浏览器页面。

### 可选的默认页面

Nightingale 支持多个有效的起始页面：

1. `/nightingale/metrics` - **推荐** 指标浏览器
2. `/nightingale/explorer` - 探索器页面
3. `/nightingale/targets` - 监控目标列表
4. `/nightingale/alert-rules` - 告警规则
5. `/nightingale/infrastructure` - 基础设施视图
6. `/nightingale/busi-groups` - 业务组管理

本次修复选择 `/metrics` 作为默认页面，因为：
- 这是监控系统最常用的功能
- 直观展示系统指标数据
- 用户可以从这里导航到其他页面

## 修改的文件

### 1. src/frontend/src/pages/MonitoringPage.js

**修改位置**: `getNightingaleUrl()` 函数

**修改前:**
```javascript
// 默认使用 nginx 代理路径（推荐，支持 ProxyAuth SSO）
const currentPort = window.location.port ? `:${window.location.port}` : '';
return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/`;
```

**修改后:**
```javascript
// 默认使用 nginx 代理路径（推荐，支持 ProxyAuth SSO）
// 使用 /metrics 作为默认页面，显示指标浏览器
const currentPort = window.location.port ? `:${window.location.port}` : '';
return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/metrics`;
```

### 完整的 getNightingaleUrl() 函数

```javascript
const getNightingaleUrl = () => {
  // 优先使用完整的 URL 配置
  if (process.env.REACT_APP_NIGHTINGALE_URL) {
    return process.env.REACT_APP_NIGHTINGALE_URL;
  }
  
  // 如果配置了端口，使用直接端口访问
  if (process.env.REACT_APP_NIGHTINGALE_PORT) {
    const port = process.env.REACT_APP_NIGHTINGALE_PORT;
    return `${window.location.protocol}//${window.location.hostname}:${port}/metrics`;
  }
  
  // 默认使用 nginx 代理路径（推荐，支持 ProxyAuth SSO）
  // 使用 /metrics 作为默认页面，显示指标浏览器
  const currentPort = window.location.port ? `:${window.location.port}` : '';
  return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/metrics`;
};
```

## 部署步骤

### 方式 1: 使用修复脚本（推荐）

```bash
# 运行修复脚本
./fix-monitoring-404.sh
```

脚本会自动：
1. 重新构建 frontend 镜像
2. 重启 frontend 容器
3. 验证容器状态

### 方式 2: 手动部署

```bash
# 1. 重新构建 frontend
docker-compose build frontend

# 2. 重启 frontend 容器
docker-compose up -d frontend

# 3. 检查容器状态
docker-compose ps frontend

# 4. 查看日志（可选）
docker-compose logs -f frontend
```

## 验证测试

### 1. 手动测试

1. **清除浏览器缓存**
   - Chrome: Ctrl/Cmd + Shift + Delete
   - 选择"缓存的图像和文件"
   - 时间范围选"全部"

2. **访问监控页面**
   ```
   http://192.168.0.200:8080/monitoring
   ```

3. **预期结果**
   - ✅ 不再显示 "404 - The page you visited does not exist!"
   - ✅ 直接显示 Nightingale 指标浏览器页面
   - ✅ 可以看到左侧导航菜单
   - ✅ 可以看到指标选择器和查询界面

### 2. 自动化测试（如果 Node.js 可用）

```bash
# 运行 Playwright E2E 测试
npx playwright test test/e2e/specs/monitoring-check-404-message.spec.js --config=test/e2e/playwright.config.js
```

**预期输出:**
```
=== iframe 检查 ===
Frame 2: http://192.168.0.200:8080/nightingale/metrics
✅ 此 frame 未发现 404 信息
```

## 故障排查

### 问题 1: 仍然显示 404

**可能原因:**
- 浏览器缓存未清除
- Frontend 容器未正确重启

**解决方法:**
```bash
# 1. 强制清除浏览器缓存
# 2. 完全重启 frontend
docker-compose stop frontend
docker-compose rm -f frontend
docker-compose up -d frontend
```

### 问题 2: Iframe 无法加载

**可能原因:**
- Nginx 配置问题
- Nightingale 服务未启动
- SSO 认证失败

**排查步骤:**
```bash
# 1. 检查 Nightingale 容器状态
docker-compose ps nightingale

# 2. 检查 Nightingale 日志
docker-compose logs nightingale --tail=50

# 3. 检查 Nginx 配置
docker-compose exec nginx nginx -t

# 4. 测试直接访问
curl -H "Cookie: auth_token=YOUR_TOKEN" http://192.168.0.200:8080/nightingale/metrics
```

### 问题 3: 权限问题

**症状:** 可以访问页面但看不到数据

**解决方法:**
1. 确认用户已登录
2. 检查 Nightingale ProxyAuth 配置
3. 验证用户在 Nightingale 中有正确的角色（Admin）

```bash
# 检查 Nightingale 数据库中的用户
docker-compose exec postgres psql -U nightingale -d nightingale -c "SELECT username, roles FROM users;"
```

## 自定义配置

### 更改默认页面

如果希望使用其他页面作为默认页面，编辑 `src/frontend/src/pages/MonitoringPage.js`:

```javascript
// 例如：使用告警规则页面作为默认
return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/alert-rules`;

// 或者：使用基础设施视图
return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/infrastructure`;
```

然后重新构建和部署。

### 使用环境变量

在 `.env` 文件中配置：

```env
# 方式 1: 完整 URL
REACT_APP_NIGHTINGALE_URL=http://192.168.0.200:8080/nightingale/metrics

# 方式 2: 仅端口（如果直接访问 Nightingale）
REACT_APP_NIGHTINGALE_PORT=17000
```

## 技术细节

### Nightingale 路由说明

Nightingale 是一个单页应用（SPA），使用前端路由。当访问根路径时：

1. Nightingale 前端加载
2. 检查当前路由
3. 如果路由为 `/`，显示 404 页面（因为没有对应的组件）
4. 用户需要手动导航到有效页面

### 为什么不在 Nginx 中重定向？

虽然可以在 Nginx 中配置重定向：

```nginx
location = /nightingale/ {
    return 302 /nightingale/metrics;
}
```

但这种方法有缺点：
- 增加一次 HTTP 重定向
- 浏览器 URL 会变化
- iframe 中的重定向可能被浏览器阻止

在前端直接配置正确的 URL 更简洁、更可靠。

## 相关文档

- [Nightingale 官方文档](https://n9e.github.io/)
- [AI-Infra-Matrix 监控集成文档](./NGINX_TEMPLATE_UPDATE.md)
- [SSO 集成文档](./MONITORING_404_FIX_REPORT.md)

## 总结

本次修复通过将 Nightingale iframe 的默认路径从根路径 `/nightingale/` 改为指标浏览器 `/nightingale/metrics`，解决了监控页面显示 404 错误的问题。

修复后：
- ✅ 用户访问监控页面立即看到有用的内容
- ✅ 无需手动导航到其他页面
- ✅ 保持 SSO 集成正常工作
- ✅ 不影响其他功能

---

**修复日期**: 2025年10月24日  
**影响范围**: Frontend (监控页面)  
**向后兼容**: 是  
**需要重启**: Frontend 容器
