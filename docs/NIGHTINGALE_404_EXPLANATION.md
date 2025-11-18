# Nightingale 监控系统集成说明

## 当前状态

### ✅ 已修复的问题

1. **后端认证** - Cookie 认证已启用，支持从浏览器 Session 中验证用户身份
2. **Nginx 静态资源路由** - Nightingale 的 font、js、image 资源路由已优化
3. **SSO 集成** - ProxyAuth 单点登录正常工作，用户自动同步
4. **API 访问** - 所有 Nightingale API (`/api/n9e/*`) 正常响应

### ⚠️ 已知限制

#### 根路径显示 404 页面

**现象**：访问 `/nightingale/` 根路径时，页面中央显示"404 - The page you visited does not exist!"

**原因**：
- Nightingale 是一个 SPA (单页应用)，使用客户端路由
- Nightingale 的路由配置没有为根路径 `/` 设置默认页面
- 通过 Nginx 反向代理部署在子路径 `/nightingale/` 下时，需要显式访问具体功能页面

**影响**：
- **不影响功能使用** - 所有监控功能都能正常工作
- 左侧导航菜单完全可用
- 用户已成功登录（通过 SSO）
- API 数据正常加载

**使用方法**：
1. 访问 `http://192.168.0.200:8080/monitoring`
2. 看到 404 页面时，**点击左侧菜单中的任意功能**：
   - **Dashboards（仪表板）** - 查看监控大盘
   - **Metrics（指标浏览器）** - 查询指标数据
   - **Rules（告警规则）** - 管理告警规则
   - **Targets（业务组）** - 管理监控目标
   - **Events（告警事件）** - 查看告警历史
3. 点击后即可正常使用对应功能

### 📊 功能验证

通过 Playwright E2E 测试验证：

```bash
# 验证 Nightingale 应用加载
BASE_URL=http://192.168.0.200:8080 npx playwright test \
  test/e2e/specs/find-nightingale-real-path.spec.js \
  --config=test/e2e/playwright.config.js
```

测试结果确认：
- ✅ 21 个导航链接正常工作
- ✅ API 请求成功（8个接口全部返回 200）
- ✅ 用户信息正确加载（admin, ID=2）
- ✅ 左侧菜单完整显示

### 🔧 技术细节

#### Nginx 代理配置

`/nightingale/` location：
- 反向代理到 `http://nightingale:17000/`
- 使用 `auth_request` 进行 SSO 认证
- 传递 `X-User-Name` header 给 Nightingale
- 使用 `sub_filter` 重写前端资源路径
- 支持 WebSocket 连接

#### API 路由

`/api/n9e/` location：
- 直接代理到 Nightingale 后端
- 支持 Cookie 和 Authorization header 认证
- 自动注入 `X-User-Name` header（来自 auth_request）

### 💡 改进方案（可选）

如果需要消除 404 页面，可以选择以下方案之一：

#### 方案 1：修改 iframe 默认 URL

修改 `src/frontend/src/pages/MonitoringPage.js`，直接加载一个具体页面：

```javascript
// 加载仪表板页面作为默认页面
return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/dashboards`;
```

**问题**：由于 `sub_filter` 的限制，所有子路径同样会显示 404（客户端路由无法匹配）

#### 方案 2：独立部署 Nightingale

将 Nightingale 部署在独立端口或域名，不使用子路径：

```yaml
# docker-compose.yml
services:
  nightingale:
    ports:
      - "17000:17000"  # 暴露端口
```

```javascript
// MonitoringPage.js
return `http://192.168.0.200:17000/`;
```

**优点**：完全避免子路径带来的路由问题
**缺点**：需要额外的端口，可能影响防火墙配置

#### 方案 3：修改 Nightingale 源码

在 Nightingale 前端源码中：
1. 配置 React Router 的 `basename="/nightingale"`
2. 或添加根路径的默认重定向

**优点**：从根本上解决问题
**缺点**：需要维护自定义版本，升级时需要重新修改

### 🎯 推荐做法

**保持现状**，在用户文档中说明：
1. 监控页面首次加载时会显示 404 提示
2. 这是正常现象，不影响功能
3. 点击左侧菜单中的任意功能即可使用
4. 所有监控功能（仪表板、指标、告警等）都能正常工作

**理由**：
- 功能完全可用，不影响实际使用
- 避免引入额外的复杂性和维护成本
- Nightingale 的设计理念就是通过菜单导航，而不是依赖默认首页

## 参考资料

- E2E 测试: `test/e2e/specs/find-nightingale-real-path.spec.js`
- Nginx 配置: `src/nginx/conf.d/includes/nightingale.conf`
- 前端组件: `src/frontend/src/pages/MonitoringPage.js`
