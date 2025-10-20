# E2E 测试快速开始清单

## ✅ 已完成

### 1. 测试文件创建 ✓
- [x] `test/e2e/specs/complete-e2e-test.spec.js` - 完整测试套件（41个测试）
- [x] `test/e2e/specs/quick-validation-test.spec.js` - 快速验证测试（9个测试）

### 2. 运行脚本创建 ✓
- [x] `run-e2e-tests.sh` - 主测试运行脚本
- [x] 添加执行权限

### 3. 配置文件更新 ✓
- [x] `test/e2e/package.json` - 添加 npm 脚本

### 4. 文档创建 ✓
- [x] `docs/E2E_TESTING_GUIDE.md` - 完整测试指南
- [x] `docs/E2E_TEST_IMPLEMENTATION.md` - 实现总结
- [x] `test/e2e/README.md` - 快速参考

## 🚀 立即开始测试

### 步骤 1: 确保服务运行

```bash
# 启动所有服务
docker-compose up -d

# 等待服务就绪（约30秒）
sleep 30

# 验证服务可访问
curl http://192.168.0.200:8080
```

### 步骤 2: 安装测试依赖

```bash
# 进入测试目录
cd test/e2e

# 安装依赖
npm install

# 安装 Chromium 浏览器
npm run install:browsers
```

### 步骤 3: 运行测试

#### 方式 A: 使用项目根目录的脚本（推荐）

```bash
# 返回项目根目录
cd ../..

# 运行快速验证测试（3-5分钟）
./run-e2e-tests.sh --quick

# 运行完整测试套件（10-15分钟）
./run-e2e-tests.sh --full

# 显示浏览器窗口（调试用）
./run-e2e-tests.sh --quick --headed
```

#### 方式 B: 使用 npm 脚本

```bash
# 进入测试目录
cd test/e2e

# 快速验证测试
npm run test:quick

# 完整测试套件
npm run test:full

# 显示浏览器
npm run test:headed

# 调试模式
npm run test:debug
```

#### 方式 C: 直接使用 npx

```bash
cd test/e2e

# 快速测试
BASE_URL=http://192.168.0.200:8080 \
npx playwright test specs/quick-validation-test.spec.js

# 完整测试
BASE_URL=http://192.168.0.200:8080 \
npx playwright test specs/complete-e2e-test.spec.js
```

### 步骤 4: 查看测试报告

```bash
cd test/e2e

# 在浏览器中打开 HTML 报告
npm run report

# 或直接使用 npx
npx playwright show-report
```

## 📋 测试覆盖范围

### 快速验证测试（推荐先运行）
验证最近修复的功能：

1. ✅ JupyterHub 配置渲染
2. ✅ Gitea 静态资源路径
3. ✅ Object Storage 自动刷新
4. ✅ SLURM Dashboard SaltStack 集成
5. ✅ SLURM Tasks 刷新频率优化
6. ✅ SLURM Tasks 统计信息
7. ✅ 控制台错误检查
8. ✅ 网络请求监控
9. ✅ 性能基准测试

**预计时间**: 3-5 分钟

### 完整测试套件
覆盖所有核心功能：

1. 用户认证（4个测试）
2. 核心功能页面（9个测试）
3. SLURM 任务管理（5个测试）
4. SaltStack 管理（3个测试）
5. 对象存储管理（3个测试）
6. 管理员功能（5个测试）
7. 前端优化验证（4个测试）
8. 导航和路由（3个测试）
9. 错误处理（3个测试）
10. 集成测试（2个测试）

**预计时间**: 10-15 分钟

## ⚙️ 环境变量

可选的环境变量配置：

```bash
export BASE_URL=http://192.168.0.200:8080
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD=admin123
export TEST_USERNAME=testuser
export TEST_PASSWORD=test123
```

## 🔍 故障排查

### 问题 1: 浏览器未安装

```bash
cd test/e2e
npm run install:browsers
```

### 问题 2: 服务未启动

```bash
# 检查状态
docker-compose ps

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f frontend
```

### 问题 3: 测试超时

编辑 `test/e2e/playwright.config.js`，增加超时时间：

```javascript
timeout: 90_000, // 增加到 90 秒
```

### 问题 4: 端口冲突

修改 BASE_URL：

```bash
./run-e2e-tests.sh --quick --url http://localhost:8080
```

## 📚 详细文档

- **完整指南**: [docs/E2E_TESTING_GUIDE.md](../docs/E2E_TESTING_GUIDE.md)
- **实现总结**: [docs/E2E_TEST_IMPLEMENTATION.md](../docs/E2E_TEST_IMPLEMENTATION.md)
- **快速参考**: [test/e2e/README.md](README.md)

## 💡 提示

### 首次运行建议

1. 先运行快速验证测试：
   ```bash
   ./run-e2e-tests.sh --quick
   ```

2. 如果通过，再运行完整测试：
   ```bash
   ./run-e2e-tests.sh --full
   ```

3. 查看详细报告：
   ```bash
   cd test/e2e && npm run report
   ```

### 调试测试失败

1. 使用 headed 模式查看浏览器：
   ```bash
   ./run-e2e-tests.sh --quick --headed
   ```

2. 使用调试模式逐步执行：
   ```bash
   cd test/e2e
   npm run test:debug
   ```

3. 查看失败截图：
   ```bash
   ls -lh test/e2e/test-results/
   ```

### 性能优化

如果测试运行缓慢：

1. 检查 Docker 资源限制
2. 确保网络连接稳定
3. 考虑增加超时时间
4. 使用无头模式运行

## 🎯 下一步

1. ✅ 运行快速验证测试
2. ✅ 检查测试报告
3. ✅ 如有失败，查看截图和日志
4. ✅ 修复问题后重新运行
5. ✅ 运行完整测试套件
6. ✅ 将测试集成到 CI/CD 流程

## 📞 获取帮助

- 查看 [E2E_TESTING_GUIDE.md](../docs/E2E_TESTING_GUIDE.md) 的常见问题章节
- 查看 Playwright 官方文档: https://playwright.dev/
- 提交 Issue 或联系开发团队

---

**准备好了吗？运行你的第一个测试！**

```bash
./run-e2e-tests.sh --quick
```
