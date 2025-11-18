# 代理问题修复说明

## 问题描述

运行 E2E 测试时出现 `net::ERR_SOCKET_NOT_CONNECTED` 错误，原因是系统配置了 HTTP 代理，导致 Playwright 无法访问本地服务 `http://192.168.0.200:8080`。

## 错误信息

```
Error: page.goto: net::ERR_SOCKET_NOT_CONNECTED at http://192.168.0.200:8080/
```

## 根本原因

系统环境变量中配置了代理：
```bash
http_proxy=http://127.0.0.1:7890
```

但本地 IP `192.168.0.200` 没有正确添加到 `no_proxy` 列表中，导致请求被错误地发送到代理服务器。

## 解决方案

### 方案 1: 使用更新后的 npm 脚本（推荐）

npm 脚本已更新，自动禁用代理：

```bash
cd test/e2e

# 运行快速测试
npm run test:quick

# 运行完整测试
npm run test:full

# 显示浏览器
npm run test:headed
```

### 方案 2: 使用测试目录的运行脚本

```bash
cd test/e2e
./run-test.sh
```

这个脚本会自动：
1. 禁用所有代理
2. 检查服务状态
3. 运行测试

### 方案 3: 手动设置环境变量

```bash
cd test/e2e

# 完全禁用代理
unset http_proxy
unset https_proxy
unset HTTP_PROXY
unset HTTPS_PROXY
export NO_PROXY="*"
export no_proxy="*"

# 运行测试
BASE_URL=http://192.168.0.200:8080 \
npx playwright test specs/quick-validation-test.spec.js
```

### 方案 4: 使用项目根目录的快速测试脚本

```bash
# 从项目根目录运行
./quick-test.sh
```

## 验证修复

运行以下命令验证可以正常访问：

```bash
# 测试不使用代理访问
NO_PROXY="*" no_proxy="*" curl http://192.168.0.200:8080

# 应该返回 HTML 或 302 重定向
```

## 更新的文件

1. **`test/e2e/package.json`** - 添加 `NO_PROXY='*'` 到所有测试脚本
2. **`test/e2e/run-test.sh`** - 新增简易测试运行脚本
3. **`quick-test.sh`** - 新增项目根目录快速测试脚本
4. **`run-e2e-tests.sh`** - 更新主测试脚本，自动检测并配置代理绕过

## 测试步骤

### 1. 确保服务运行

```bash
docker-compose up -d
docker-compose ps  # 确认所有服务状态为 Up
```

### 2. 运行测试

**推荐方式 - 使用 npm**：
```bash
cd test/e2e
npm run test:quick
```

**或使用脚本**：
```bash
cd test/e2e
./run-test.sh
```

**或从项目根目录**：
```bash
./quick-test.sh
```

### 3. 查看结果

```bash
cd test/e2e
npm run report
```

## 预期结果

所有 9 个快速验证测试应该通过：

```
✓ 1. JupyterHub 配置渲染验证
✓ 2. Gitea 静态资源路径验证
✓ 3. Object Storage 自动刷新功能验证
✓ 4. SLURM Dashboard SaltStack 集成显示验证
✓ 5. SLURM Tasks 刷新频率优化验证
✓ 6. SLURM Tasks 统计信息加载验证
✓ 7. 控制台错误检查
✓ 8. 网络请求监控
✓ 9. 页面加载性能
```

## 常见问题

### Q: 为什么要禁用代理？

A: 本地 Docker 服务运行在 `192.168.0.200:8080`，这是一个本地网络地址，不需要通过代理访问。代理会导致连接失败。

### Q: 禁用代理会影响其他请求吗？

A: 不会。我们只在测试运行时临时禁用代理，不影响系统全局配置。

### Q: 如果服务在不同的 IP 或端口？

A: 修改 `BASE_URL` 环境变量：

```bash
BASE_URL=http://localhost:8080 npm run test:quick
```

### Q: 代理配置文件在哪里？

A: 通常在：
- `~/.zshrc` 或 `~/.bashrc` - shell 配置
- `/etc/environment` - 系统全局配置

## 进一步调试

如果问题仍然存在：

1. **检查代理配置**：
   ```bash
   echo $http_proxy
   echo $HTTP_PROXY
   echo $no_proxy
   echo $NO_PROXY
   ```

2. **测试直接访问**：
   ```bash
   curl -v http://192.168.0.200:8080
   ```

3. **检查 Docker 网络**：
   ```bash
   docker network ls
   docker network inspect ai-infra-matrix_default
   ```

4. **查看 nginx 日志**：
   ```bash
   docker-compose logs -f nginx
   ```

## 相关文档

- [E2E_TESTING_GUIDE.md](../docs/E2E_TESTING_GUIDE.md) - 完整测试指南
- [E2E_QUICKSTART.md](../E2E_QUICKSTART.md) - 快速开始指南

---

**更新日期**: 2025-01-12  
**问题类型**: 网络/代理配置  
**影响范围**: E2E 测试  
**解决状态**: ✅ 已修复
