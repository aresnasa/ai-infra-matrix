# 清理浏览器缓存解决8083端口问题

## 问题分析

浏览器仍在访问 `localhost:8083/api/auth/me`，尽管我们已经修复了所有代码和配置文件。这是典型的浏览器缓存问题。

## 解决方案

### 1. 强制清理浏览器缓存（推荐）

#### Chrome/Edge：
1. 按 `F12` 打开开发者工具
2. 右键点击刷新按钮
3. 选择"清空缓存并硬性重新加载"
4. 或者按 `Ctrl+Shift+R` (Windows) / `Cmd+Shift+R` (Mac)

#### Firefox：
1. 按 `Ctrl+Shift+R` (Windows) / `Cmd+Shift+R` (Mac) 强制刷新
2. 或者按 `F12` 打开开发者工具，在网络选项卡中选择"禁用缓存"

### 2. 清理应用程序存储

#### 在开发者工具中：
1. 按 `F12` 打开开发者工具
2. 转到 "Application"（应用程序）选项卡
3. 在左侧找到 "Storage"（存储）
4. 点击 "Clear storage"（清除存储）
5. 确保勾选所有选项：
   - Local and session storage
   - IndexedDB
   - Web SQL
   - Cookies
   - Service Workers
   - Cache storage
6. 点击 "Clear site data"（清除站点数据）

### 3. 无痕模式测试

打开无痕/隐私模式窗口，访问 `http://localhost:8080` 测试是否正常工作。

### 4. Service Worker 清理

如果仍有问题，在开发者工具中：
1. 转到 "Application" > "Service Workers"
2. 如果看到任何 Service Worker，点击 "Unregister"

### 5. 检查网络请求

在开发者工具的 Network（网络）选项卡中：
1. 确保勾选 "Disable cache"（禁用缓存）
2. 刷新页面
3. 查看实际发送的API请求URL
4. 确认请求是发送到 `localhost:8080/api/` 而不是 `8083`

## 快速验证脚本

```bash
# 检查前端容器配置
docker exec ai-infra-frontend cat /etc/nginx/conf.d/default.conf | grep proxy_pass

# 检查前端构建文件中是否还有8083引用
docker exec ai-infra-frontend grep -r "8083" /usr/share/nginx/html/ || echo "✅ 没有找到8083引用"

# 验证服务端口
docker ps --format "table {{.Names}}\t{{.Ports}}" | grep ai-infra
```

## 预期结果

清理缓存后，浏览器应该：
1. 访问 `http://localhost:8080/api/auth/me`
2. 通过nginx代理转发到 `backend:8082`
3. 不再出现CORS错误
4. 能够正常登录和使用AI助手

## 如果问题仍然存在

如果清理缓存后问题仍存在，请：
1. 重启Docker容器：`docker-compose restart`
2. 重建前端镜像：`docker-compose build frontend`
3. 检查是否有其他浏览器扩展或代理设置影响请求
