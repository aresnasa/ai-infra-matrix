# AI Infrastructure Matrix - E2E Tests

端到端测试套件，使用 Playwright 进行自动化测试。

## 快速开始

### 1. 安装依赖

```bash
npm install
npm run install:browsers
```

### 2. 启动服务

```bash
# 返回项目根目录
cd ../..

# 启动所有服务
docker-compose up -d

# 等待服务就绪
sleep 30
```

### 3. 运行测试

```bash
# 返回测试目录
cd test/e2e

# 运行快速验证测试（推荐）⭐
npm run test:quick

# 运行完整测试套件
npm run test:full

# 显示浏览器窗口
npm run test:headed

# 调试模式
npm run test:debug

# 使用 UI 模式
npm run test:ui
```

## 测试文件说明

### 新增测试套件 🆕

#### `specs/quick-validation-test.spec.js` ⭐ **推荐**

快速验证测试，专注于最近修复的功能：
- ✅ JupyterHub 配置渲染验证
- ✅ Gitea 静态资源路径验证
- ✅ Object Storage 自动刷新验证
- ✅ SLURM Dashboard SaltStack 集成验证
- ✅ SLURM Tasks 刷新频率优化验证
- ✅ SLURM Tasks 统计信息验证
- ✅ 控制台错误检查
- ✅ 网络请求监控
- ✅ 性能基准测试

**运行命令**:
```bash
npm run test:quick
```

**预计运行时间**: 约 3-5 分钟

#### `specs/complete-e2e-test.spec.js`

完整的 E2E 测试套件，覆盖所有核心功能：
- 用户认证流程（登录、登出、错误处理）
- 核心功能页面访问（项目、SLURM、SaltStack、K8s 等）
- SLURM 任务管理（列表、筛选、统计、刷新）
- SaltStack 管理（仪表板、Minion、集成状态）
- 对象存储管理（页面加载、刷新、Bucket 列表）
- 管理员功能（用户管理、项目管理、LDAP、配置）
- 前端优化验证（刷新频率、懒加载、集成显示）
- 导航和路由测试
- 错误处理和边界测试
- 集成测试（完整工作流）

**运行命令**:
```bash
npm run test:full
```

**预计运行时间**: 约 10-15 分钟

### 现有测试文件

#### `specs/final-verification-test.spec.js`
- **用途**: 验证 SaltStack 执行完成状态修复
- **测试内容**:
  1. 登录系统
  2. 打开 SaltStack 页面
  3. 执行命令
  4. 验证执行完成后按钮状态正确恢复
  5. 验证可重复执行

#### `specs/debug-saltstack.spec.js`
- 调试 SaltStack 页面加载
- 输出页面元素信息
- 生成截图

#### `specs/saltstack-exec.spec.js`
- 完整的 SaltStack 执行测试套件
- 包含 11 个全面的测试用例

## 环境变量

在运行测试前，可以设置以下环境变量：

```bash
export BASE_URL=http://192.168.0.200:8080
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD=admin123
export TEST_USERNAME=testuser
export TEST_PASSWORD=test123
```

## 查看测试报告

```bash
npm run report
```

这将在浏览器中打开 HTML 测试报告，包含：
- 测试结果摘要
- 失败截图
- 测试执行视频（如果启用）
- 详细的测试步骤

## 使用项目根目录的测试脚本

从项目根目录运行测试：

```bash
# 快速验证测试
./run-e2e-tests.sh --quick

# 完整测试套件
./run-e2e-tests.sh --full

# 显示浏览器窗口
./run-e2e-tests.sh --quick --headed

# 指定不同的 URL
./run-e2e-tests.sh --quick --url http://localhost:8080
```

## 故障排查

### 浏览器未安装

```bash
npm run install:browsers
```

### 服务未启动

```bash
# 检查服务状态
docker-compose ps

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f frontend
```

### 超时错误

增加超时时间，修改 `playwright.config.js`:

```javascript
timeout: 60_000, // 增加到 60 秒
```

### 找不到"执行命令"按钮

页面可能还在加载，使用适当的等待逻辑：

```javascript
await page.waitForSelector('button:has-text("执行命令")', { 
  state: 'visible',
  timeout: 10000 
});
```

## 项目结构

```
test/e2e/
├── specs/                              # 测试规范文件
│   ├── complete-e2e-test.spec.js      # 🆕 完整测试套件
│   ├── quick-validation-test.spec.js  # 🆕 快速验证测试
│   ├── final-verification-test.spec.js
│   ├── debug-saltstack.spec.js
│   ├── saltstack-exec.spec.js
│   └── ...                            # 其他测试文件
├── test-results/                       # 测试结果输出
├── playwright.config.js                # Playwright 配置
├── package.json                        # NPM 配置
└── README.md                           # 本文件
```

## 详细文档

查看以下文档获取更多信息：

- [E2E_TESTING_GUIDE.md](../../docs/E2E_TESTING_GUIDE.md) - 完整的测试指南
- [SLURM_TASKS_REFRESH_OPTIMIZATION.md](../../docs/SLURM_TASKS_REFRESH_OPTIMIZATION.md) - 刷新优化文档
- [FRONTEND_PAGE_FIXES.md](../../docs/FRONTEND_PAGE_FIXES.md) - 前端修复汇总
- [SALTSTACK_FIX_TEST_SUMMARY.md](../../docs/SALTSTACK_FIX_TEST_SUMMARY.md) - SaltStack 修复总结

## 测试最佳实践

1. **运行前确认服务可访问**
   ```bash
   curl http://192.168.0.200:8080
   ```

2. **清理旧的测试结果**
   ```bash
   rm -rf test-results/
   ```

3. **查看详细日志**
   ```bash
   npm run test:quick -- --reporter=list
   ```

4. **只运行特定测试**
   ```bash
   # 使用 --grep 过滤
   npm run test:quick -- --grep "SLURM Tasks"
   ```

## 贡献指南

### 添加新测试

1. 在 `specs/` 目录创建新的 `.spec.js` 文件
2. 参考 `complete-e2e-test.spec.js` 的结构
3. 使用辅助函数（`login`, `logout`, `waitForPageLoad`）
4. 添加适当的等待和错误处理
5. 包含截图便于调试

### 测试模板

```javascript
const { test, expect } = require('@playwright/test');

const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://localhost:8080',
  adminUser: {
    username: 'admin',
    password: 'admin123',
  },
};

async function login(page, username, password) {
  await page.goto('/');
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

test.describe('我的测试套件', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  });

  test('测试功能', async ({ page }) => {
    await page.goto('/my-page');
    await expect(page.locator('text=标题')).toBeVisible();
  });
});
```

## 相关链接

- [Playwright 文档](https://playwright.dev/)
- [项目主 README](../../README.md)
- [构建和测试指南](../../docs/BUILD_AND_TEST_GUIDE.md)

---
最后更新: 2025-01-12
