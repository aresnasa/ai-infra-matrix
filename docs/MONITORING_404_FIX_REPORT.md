# Monitoring Page 404 Error Fix Report

## 问题描述

在访问 http://192.168.0.200:8080/monitoring 时，Nightingale 监控系统的多个静态资源返回 404 错误，导致页面无法正常加载。

### 受影响的资源

1. `/font/iconfont.js` - 字体图标文件
2. `/js/widget.js` - JavaScript 组件
3. `/image/logo-light.png` - Logo 图片（浅色主题）
4. `/image/logo-l.png` - Logo 图片（另一个版本）
5. `/image/logo.png` - Logo 图片（默认）

## 根本原因分析

### 问题根源

Nginx 配置中的静态文件 location 匹配规则存在优先级问题：

```nginx
# 问题配置：此规则会匹配所有静态文件扩展名
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
    proxy_pass http://frontend;
    expires 1y;
    add_header Cache-Control "public, immutable";
}
```

当 Nightingale 监控系统尝试加载其静态资源时（如 `/font/iconfont.js`、`/js/widget.js`、`/image/logo.png`），这些请求被上述规则拦截并转发到 frontend 服务，而 frontend 中并不包含这些 Nightingale 专用的资源，因此返回 404。

### 为什么 `/nightingale/` 路径下的资源正常？

在 `nightingale.conf` 中定义的 `/nightingale/` location 使用了 `^~` 修饰符：

```nginx
location ^~ /nightingale/ {
    proxy_pass http://nightingale:17000/;
    # ... 正确代理到 Nightingale 服务
}
```

`^~` 修饰符告诉 Nginx 停止正则匹配，所以 `/nightingale/` 下的资源能正确代理。但是，Nightingale 的某些资源（特别是 iconfont.js、widget.js 和 logo 图片）被设计为从根路径加载（`/font/`、`/js/`、`/image/`），而不是 `/nightingale/` 子路径。

## 解决方案

### 修改的文件

1. **src/nginx/conf.d/server-main.conf**
2. **src/nginx/templates/conf.d/server-main.conf.tpl**

### 实施的修复

在通用静态文件 location 规则**之前**添加了 Nightingale 专用的静态资源路由：

```nginx
# Nightingale static assets - must come before general static file location
# These paths are used by Nightingale monitoring system
location ~ ^/(font|js|image)/ {
    proxy_pass http://nightingale_console;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    expires 1d;
    add_header Cache-Control "public";
}

# 前端静态资源与入口（保持不变）
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
    proxy_pass http://frontend;
    expires 1y;
    add_header Cache-Control "public, immutable";
}
```

### 为什么这个方案有效？

1. **Nginx location 匹配优先级**：正则 location（`~`）按照在配置文件中出现的顺序进行匹配，第一个匹配的规则生效
2. **更具体的匹配优先**：新添加的 `location ~ ^/(font|js|image)/` 规则在更通用的静态文件规则之前，所以会优先匹配
3. **不影响其他路径**：只有 `/font/`、`/js/`、`/image/` 开头的路径才会被代理到 Nightingale，其他静态资源仍然正常代理到 frontend

## 部署步骤

```bash
# 1. 重建 Nginx 镜像
docker-compose build nginx

# 2. 重启 Nginx 容器
docker-compose up -d nginx

# 3. 验证服务状态
docker-compose ps nginx
```

## 测试验证

### 自动化测试

创建了两个 Playwright E2E 测试套件：

#### 1. monitoring-404-debug.spec.js
- **目的**：诊断并列出所有 404 错误
- **结果**：✅ 没有发现 404 错误

#### 2. monitoring-complete-test.spec.js
- **包含 6 个测试用例**：
  1. ✅ No 404 errors on monitoring page
  2. ✅ Monitoring iframe loads successfully
  3. ✅ All Nightingale static assets load correctly
     - Font: iconfont.js (200 OK)
     - JS: widget.js (200 OK)
     - Images: logo-light.png, logo-l.png, logo.png (200 OK)
  4. ✅ Page renders without JavaScript errors
  5. ✅ Monitoring page SSO integration works
  6. ✅ Screenshot comparison

### 测试结果

```
Running 6 tests using 1 worker

  ✓  1. No 404 errors on monitoring page (5.1s)
  ✓  2. Monitoring iframe loads successfully (2.8s)
  ✓  3. All Nightingale static assets load correctly (4.1s)
  ✓  4. Page renders without JavaScript errors (3.8s)
  ✓  5. Monitoring page SSO integration works (5.9s)
  ✓  6. Screenshot comparison - monitoring page (4.0s)

  6 passed (26.9s)
```

### 静态资源加载验证

所有 Nightingale 静态资源现在都返回 200 OK：

```
字体文件 (1):
  ✅ 200 - http://192.168.0.200:8080/font/iconfont.js

JS 文件 (3):
  ✅ 200 - http://192.168.0.200:8080/js/widget.js
  ✅ 200 - http://192.168.0.200:8080/nightingale/js/node-sql-parser@4.10.0_umd_mysql.umd.js
  ✅ 200 - http://192.168.0.200:8080/nightingale/js/placement.min.js

图片文件 (3):
  ✅ 200 - http://192.168.0.200:8080/image/logo-light.png
  ✅ 200 - http://192.168.0.200:8080/image/logo-l.png
  ✅ 200 - http://192.168.0.200:8080/image/logo.png
```

## 运行测试

### 运行 404 诊断测试
```bash
npx playwright test test/e2e/specs/monitoring-404-debug.spec.js --config=test/e2e/playwright.config.js
```

### 运行完整功能测试
```bash
npx playwright test test/e2e/specs/monitoring-complete-test.spec.js --config=test/e2e/playwright.config.js
```

### 运行所有监控相关测试
```bash
npx playwright test test/e2e/specs/monitoring-*.spec.js --config=test/e2e/playwright.config.js
```

## 技术要点

### Nginx Location 匹配优先级

1. **精确匹配** `location = /path` - 最高优先级
2. **前缀匹配（停止搜索）** `location ^~ /path` - 匹配成功后停止正则匹配
3. **正则匹配** `location ~ /path` 或 `location ~* /path` - 按配置顺序匹配
4. **普通前缀匹配** `location /path` - 最低优先级

### 本次修复的关键点

- 使用正则 location `location ~ ^/(font|js|image)/` 
- 放置在更通用的静态文件规则**之前**
- 正则匹配按照配置文件中的顺序，第一个匹配的生效

## 影响范围

### 修改的服务
- ✅ Nginx (已重建并重启)

### 不受影响的服务
- Frontend
- Backend  
- Nightingale
- 其他微服务

### 兼容性
- ✅ 向后兼容：不影响现有的前端静态资源加载
- ✅ SSO 集成：保持现有的 ProxyAuth SSO 功能正常
- ✅ iframe 嵌入：Nightingale 继续在 iframe 中正常工作

## 后续建议

### 监控
- 定期运行 E2E 测试以确保 404 问题不会复现
- 监控 Nginx 错误日志：`docker-compose logs nginx | grep 404`

### 文档
- 更新运维文档，说明 Nightingale 静态资源的特殊路由配置
- 在添加新的静态资源 location 规则时，注意不要影响 `/font/`、`/js/`、`/image/` 路径

### CI/CD
- 将 monitoring-404-debug.spec.js 添加到 CI pipeline 中
- 在每次 Nginx 配置变更后自动运行测试

## 总结

通过添加 Nightingale 专用的静态资源路由规则，成功解决了监控页面的 5 个 404 错误。修复方案：

1. ✅ 识别问题：静态文件规则拦截了 Nightingale 的资源请求
2. ✅ 实施修复：添加优先级更高的 Nightingale 专用规则
3. ✅ 验证效果：所有自动化测试通过（6/6）
4. ✅ 保持兼容：不影响现有功能

修复后，http://192.168.0.200:8080/monitoring 页面完全正常，所有模块都能正确加载。
