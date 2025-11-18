# Nightingale Iframe 集成修复指南

## 问题总结

1. **MonitoringPage iframe 仍指向端口 17000** - 未使用 `/nightingale/` 代理路径
2. **SLURM Dashboard 指向 Grafana** - 应该指向 Nightingale
3. **Nginx 配置被禁用** - nightingale.conf 显示为 disabled
4. **401 认证错误** - ProxyAuth 未正确传递用户信息

## 已修复内容

### 1. Nginx 配置 ✅
- **文件**: `src/nginx/conf.d/includes/nightingale.conf`
- **修改**: 启用 Nightingale 代理配置，添加 ProxyAuth 支持

### 2. SLURM Dashboard ✅
- **文件**: `src/frontend/src/pages/SlurmScalingPage.js`
- **修改**: 
  - 将 Grafana URL (`port 3000`) 改为 Nightingale 代理路径
  - 移除 Grafana 相关的错误处理
  - 更新说明文字

### 3. 后端 API ✅  
- **文件**: `src/backend/internal/handlers/user_handler.go`
- **修改**: `GetProfile()` 添加响应头 `X-User-Name`, `X-User-Email`, `X-User-ID`

### 4. Gitea 配置 ✅
- **文件**: `.env`, `.env.example`
- **修改**: `GITEA_BASE_URL` 从 `http://192.168.0.200:8080/gitea/` 改为 `http://gitea:3000/`

### 5. 测试脚本 ✅
- **文件**: `test/e2e/specs/nightingale-integration-test.spec.js`
- **功能**: 完整的 iframe 集成测试

## 部署步骤

### 方案 A: 完整重新构建（推荐）

```bash
# 1. 重新构建 frontend (包含新的 iframe URL)
docker-compose build frontend

# 2. 重新构建 backend (包含新的 API 响应头)
docker-compose build backend

# 3. 重启所有相关服务
docker-compose restart nginx backend frontend nightingale

# 4. 等待服务启动
sleep 10

# 5. 验证 nginx 配置
docker-compose exec nginx nginx -t

# 6. 查看 nginx 配置
docker-compose exec nginx cat /etc/nginx/conf.d/includes/nightingale.conf
```

### 方案 B: 热更新（仅限测试）

```bash
# 1. 更新 nginx 配置（已修改本地文件）
docker-compose restart nginx

# 2. 重启后端（新增响应头）
docker-compose restart backend

# 3. 重新构建前端（修改了 iframe URL）
docker-compose up -d --build frontend

# 4. 等待启动
sleep 5
```

## 验证步骤

### 1. 检查 Nginx 配置

```bash
# 查看 nightingale.conf 内容
docker-compose exec nginx cat /etc/nginx/conf.d/includes/nightingale.conf

# 应该看到完整的配置，而不是 "Nightingale is disabled"
# 确认包含: location /nightingale/, auth_request, proxy_pass http://nightingale:17000
```

### 2. 测试后端 API

```bash
# 获取 JWT token
TOKEN=$(curl -s -X POST http://192.168.18.114:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.token')

# 测试 /api/auth/me 返回的响应头
curl -v -H "Authorization: Bearer $TOKEN" \
  http://192.168.18.114:8080/api/auth/me \
  2>&1 | grep -i "x-user"

# 应该看到:
# < X-User-Name: admin
# < X-User-Email: admin@example.com
# < X-User-ID: 1
```

### 3. 测试 Nightingale 代理路径

```bash
# 直接访问 Nightingale 代理（带认证）
curl -v -H "Authorization: Bearer $TOKEN" \
  http://192.168.18.114:8080/nightingale/ \
  2>&1 | head -20

# 应该返回 200 OK 和 HTML 内容
```

### 4. 运行 Playwright 测试

```bash
# 运行完整集成测试
BASE_URL=http://192.168.18.114:8080 \
npx playwright test test/e2e/specs/nightingale-integration-test.spec.js \
  --config=test/e2e/playwright.config.js \
  --reporter=line

# 检查截图
ls -lh test-screenshots/*nightingale*.png
```

### 5. 浏览器手动测试

1. **访问 MonitoringPage**: http://192.168.18.114:8080/monitoring
   - 检查 iframe src 是否为 `/nightingale/`
   - 确认 Nightingale UI 正常显示
   - 验证用户已自动登录（右上角显示用户名）

2. **访问 SLURM Dashboard**: http://192.168.18.114:8080/slurm
   - 切换到"监控仪表板" Tab
   - 检查 iframe src 是否为 `/nightingale/`
   - 确认显示 Nightingale 而不是 Grafana

## 预期结果

✅ **MonitoringPage**:
- Iframe src: `http://192.168.18.114:8080/nightingale/`
- 显示 Nightingale UI
- 用户自动登录（通过 ProxyAuth）

✅ **SLURM Dashboard**:
- 监控仪表板 Tab 存在
- Iframe src: `http://192.168.18.114:8080/nightingale/`
- 显示 Nightingale 监控数据

✅ **认证流程**:
```
浏览器 → /nightingale/ 
  → Nginx auth_request → Backend /api/auth/me (验证 JWT)
  → 返回 X-User-Name 头
  → Nginx 注入 X-User-Name → Nightingale
  → ProxyAuth 识别用户 → 自动登录
```

## 故障排查

### 问题 1: Iframe 仍显示 401 错误

**原因**: 后端未返回 X-User-Name 头

**解决**:
```bash
# 1. 确认后端已重新构建
docker-compose logs backend | grep "X-User-Name"

# 2. 重启后端
docker-compose restart backend

# 3. 测试 API
curl -v -H "Authorization: Bearer $TOKEN" \
  http://192.168.18.114:8080/api/auth/me 2>&1 | grep "X-User"
```

### 问题 2: Iframe 仍指向端口 17000

**原因**: Frontend 未重新构建

**解决**:
```bash
# 1. 确认环境变量未设置端口
docker-compose exec frontend env | grep REACT_APP_NIGHTINGALE_PORT

# 2. 重新构建 frontend
docker-compose build frontend
docker-compose up -d frontend

# 3. 清除浏览器缓存并重新加载
```

### 问题 3: Nightingale 配置文件为空

**原因**: build.sh 未执行或 NIGHTINGALE_ENABLED=false

**解决**:
```bash
# 1. 检查环境变量
grep NIGHTINGALE_ENABLED .env

# 2. 手动复制配置（已在上面完成）
cat src/nginx/conf.d/includes/nightingale.conf

# 3. 重启 nginx
docker-compose restart nginx
```

### 问题 4: SLURM Dashboard 仍显示 Grafana

**原因**: Frontend 代码未更新

**解决**:
```bash
# 1. 确认代码已修改
grep "nightingale" src/frontend/src/pages/SlurmScalingPage.js

# 2. 重新构建
docker-compose build frontend
docker-compose up -d frontend
```

## 环境变量配置

### .env 关键配置

```bash
# Nightingale 启用
NIGHTINGALE_ENABLED=true
NIGHTINGALE_HOST=nightingale
NIGHTINGALE_PORT=17000

# Frontend 不设置端口（使用代理）
# REACT_APP_NIGHTINGALE_PORT=
REACT_APP_NIGHTINGALE_URL=

# Gitea 内部地址（后端直接访问）
GITEA_HOST=gitea
GITEA_PORT=3000
GITEA_BASE_URL=http://gitea:3000/
```

## 总结

本次修复解决了以下问题：

1. ✅ **统一认证**: Nightingale 通过 ProxyAuth 使用 JWT 认证，无需重复登录
2. ✅ **代理路径**: 所有 iframe 使用 `/nightingale/` 代理路径，而非直接端口访问
3. ✅ **SLURM 集成**: SLURM Dashboard 正确显示 Nightingale 监控
4. ✅ **安全性**: 通过 nginx auth_request 验证，确保只有已登录用户可访问
5. ✅ **性能**: 后端直接访问内部服务，避免不必要的代理跳转

下一步建议：
- 在 Nightingale 中配置 SLURM 集群监控指标
- 添加自定义仪表板
- 配置告警规则
