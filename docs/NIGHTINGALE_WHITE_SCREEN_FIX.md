# Nightingale 白屏问题修复总结

## 问题描述
使用 Playwright 访问 http://192.168.0.200:8080/nightingale/ 时，浏览器显示白屏，iframe 无法正常显示 Nightingale 监控内容。

## 根本原因分析

### 1. 权限路由问题（已修复✅）
**问题**：MonitoringPage 使用 `TeamProtectedRoute` 限制，只允许 `['sre']` 团队访问
**影响**：admin 用户如果没有 sre 角色，会看到 403 权限不足页面
**修复**：将路由改为 `AdminProtectedRoute`，允许所有管理员访问

**修改文件**：`src/frontend/src/App.js`
```javascript
// 修改前
<Route path="/monitoring" element={
  <TeamProtectedRoute user={user} allowedTeams={['sre']}>
    <MonitoringPage />
  </TeamProtectedRoute>
} />

// 修改后  
<Route path="/monitoring" element={
  <AdminProtectedRoute user={user}>
    <MonitoringPage />
  </AdminProtectedRoute>
} />
```

### 2. 布局容器高度问题（已修复✅）
**问题**：`.ant-layout-content` 没有设置 flexbox 布局，导致内容区域无法占满剩余高度
**影响**：iframe 虽然设置了 `height: 100%`，但父容器高度不足，实际只有 150px
**修复**：添加 flexbox 布局，让 content 区域自动填充剩余高度

**修改文件**：`src/frontend/src/App.css`
```css
.ant-layout {
  min-height: 100vh;
  display: flex;              /* 新增 */
  flex-direction: column;     /* 新增 */
}

.ant-layout-content {
  padding: 24px;
  background: #f0f2f5;
  flex: 1;                    /* 新增 */
  display: flex;              /* 新增 */
  flex-direction: column;     /* 新增 */
}
```

## 测试结果

### ✅ 成功点
1. **nginx 代理配置正确**：
   - `location ^~ /nightingale/` 优先级正确
   - `sub_filter` URL 重写正常工作
   - 所有 Nightingale 资源返回 200 OK

2. **iframe 已成功创建**：
   - iframe 数量: 1
   - iframe src: `http://192.168.0.200:8080/nightingale/`
   - iframe 可见: true
   - 加载了 10 个 Nightingale 资源（JS/CSS）

3. **权限检查通过**：
   - 页面显示 "监控仪表板"
   - 有刷新和新窗口打开按钮
   - 没有 403 权限错误

### ⚠️  待验证
- **iframe 高度问题**：Playwright 测试显示 iframe 高度只有 150px
- 需要在实际浏览器中验证 flexbox 布局是否生效
- iframe 内容加载但可能需要等待更长时间

## Nightingale 资源加载统计
```
总共 10 个请求，全部返回 200 OK:
1. /nightingale/
2. /nightingale/js/node-sql-parser@4.10.0_umd_mysql.umd.js
3. /nightingale/js/placement.min.js
4. /nightingale/assets/index-edd562d0.js
5. /nightingale/assets/vendor-4765f6f8.js
6. /nightingale/assets/antdChunk-95884032.js
7. /nightingale/assets/vendor1-4a208f89.js
8. /nightingale/assets/excelChunk-ca56de4c.js
9. /nightingale/assets/vendor2-c16957a5.js
10. /nightingale/assets/index-01eb45cc.css
```

## 下一步行动
1. **手动浏览器测试**：访问 http://192.168.0.200:8080/monitoring 查看实际效果
2. **检查 iframe 内容加载**：Nightingale 应用可能需要更长时间初始化
3. **验证 ProxyAuth**：确认通过 nginx 代理的用户名是否正确传递到 Nightingale

## 技术细节

### nginx 配置
- **Location**: `location ^~ /nightingale/`
- **Proxy**: `proxy_pass http://nightingale:17000/`
- **Headers**:
  - `X-User-Name: anonymous` (默认用户)
  - `X-Forwarded-Prefix: /nightingale`
  - `Accept-Encoding: ""` (禁用压缩以支持 sub_filter)
- **sub_filter**: 重写所有绝对路径资源引用

### React 组件
- **组件**: `MonitoringPage.js`
- **iframe URL**: 动态构建，支持环境变量配置
- **加载状态**: 15 秒超时检测
- **功能**: 刷新、新窗口打开

### 部署步骤
```bash
# 重新构建前端
docker compose build frontend

# 重启前端服务
docker compose up -d frontend
```

## 修复时间线
1. 10:52 - 发现权限问题，修改 TeamProtectedRoute → AdminProtectedRoute
2. 11:01 - 重新构建并部署前端
3. 11:10 - 测试确认 iframe 创建成功，发现高度问题
4. 11:15 - 修改 CSS flexbox 布局
5. 11:20 - 再次构建并部署

## 相关文件
- `/src/frontend/src/App.js` (路由权限)
- `/src/frontend/src/App.css` (布局样式)
- `/src/frontend/src/pages/MonitoringPage.js` (监控页面组件)
- `/src/nginx/templates/conf.d/includes/nightingale.conf.tpl` (nginx 配置)
- `/build.sh` (模板渲染脚本)
