# AI Infrastructure Matrix - E2E 测试指南

## 概述

本项目包含完整的 Playwright E2E 测试套件，用于验证所有核心功能和最近的优化修复。

## 测试覆盖范围

### 1. 完整测试套件 (`complete-e2e-test.spec.js`)

涵盖以下功能模块：

#### 1.1 用户认证
- ✅ 登录页面加载
- ✅ 管理员登录/登出
- ✅ 错误凭证处理
- ✅ 会话状态保持

#### 1.2 核心功能页面
- ✅ 项目列表
- ✅ 用户资料
- ✅ SLURM 任务管理
- ✅ SLURM Dashboard
- ✅ SaltStack 仪表板
- ✅ Kubernetes 管理
- ✅ 对象存储管理
- ✅ JupyterHub 集成
- ✅ Gitea 集成

#### 1.3 SLURM 任务管理
- ✅ 任务列表加载
- ✅ 任务筛选功能
- ✅ 统计信息页面
- ✅ 自动刷新功能
- ✅ URL 参数处理

#### 1.4 SaltStack 管理
- ✅ 仪表板加载
- ✅ Minion 列表显示
- ✅ 集成状态显示

#### 1.5 对象存储管理
- ✅ 页面加载
- ✅ 自动刷新功能
- ✅ Bucket 列表显示

#### 1.6 管理员功能
- ✅ 管理中心
- ✅ 用户管理
- ✅ 项目管理
- ✅ LDAP 配置
- ✅ 对象存储配置

#### 1.7 前端优化验证
- ✅ SLURM Tasks 刷新频率优化
- ✅ Object Storage 懒加载
- ✅ SaltStack Integration 显示
- ✅ 页面响应性能

#### 1.8 导航和路由
- ✅ 侧边栏导航
- ✅ 浏览器前进后退
- ✅ 直接 URL 访问

#### 1.9 错误处理
- ✅ 404 页面
- ✅ 网络错误处理
- ✅ 大数据量渲染

#### 1.10 集成测试
- ✅ 完整工作流测试
- ✅ 跨页面状态保持

### 2. 快速验证测试 (`quick-validation-test.spec.js`)

专注于最近修复的功能：

#### 2.1 JupyterHub 配置渲染验证
- ✅ 验证配置 URL 正确（base_url, bind_url, hub_connect_url）
- ✅ 检查没有重复路径拼接
- ✅ iframe 加载正常

#### 2.2 Gitea 静态资源路径验证
- ✅ 检查没有 `/assets/assets/` 重复路径
- ✅ 静态资源 404 监控
- ✅ STATIC_URL_PREFIX 配置正确

#### 2.3 Object Storage 自动刷新验证
- ✅ 刷新按钮功能
- ✅ 最后刷新时间显示
- ✅ 30秒自动刷新间隔
- ✅ 页面可见性检测

#### 2.4 SLURM Dashboard SaltStack 集成验证
- ✅ SaltStack 集成卡片显示
- ✅ Minion 统计信息
- ✅ 在线/离线状态
- ✅ 任务数量显示

#### 2.5 SLURM Tasks 刷新频率优化验证
- ✅ URL 参数访问测试
- ✅ 刷新间隔验证（30-60秒）
- ✅ 无循环刷新问题
- ✅ 智能间隔调整

#### 2.6 SLURM Tasks 统计信息验证
- ✅ 统计 Tab 切换
- ✅ 统计卡片加载
- ✅ 数据显示正确

#### 2.7 控制台错误检查
- ✅ 监控浏览器控制台错误
- ✅ 跨页面错误检查

#### 2.8 网络请求监控
- ✅ 4xx/5xx 错误监控
- ✅ 失败请求统计

#### 2.9 性能基准测试
- ✅ 页面加载时间测试
- ✅ 性能基准对比

## 快速开始

### 前置条件

1. **确保服务运行**：
   ```bash
   docker-compose up -d
   ```

2. **验证服务可访问**：
   ```bash
   curl http://192.168.0.200:8080
   ```

### 运行测试

#### 方式 1：使用测试脚本（推荐）

```bash
# 运行快速验证测试
./run-e2e-tests.sh --quick

# 运行完整测试套件
./run-e2e-tests.sh --full

# 显示浏览器窗口（调试）
./run-e2e-tests.sh --quick --headed

# 指定不同的 URL
./run-e2e-tests.sh --quick --url http://localhost:8080

# 查看帮助
./run-e2e-tests.sh --help
```

#### 方式 2：直接使用 npx

```bash
cd test/e2e

# 快速验证测试
BASE_URL=http://192.168.0.200:8080 \
ADMIN_USERNAME=admin \
ADMIN_PASSWORD=admin123 \
npx playwright test specs/quick-validation-test.spec.js \
  --config=playwright.config.js

# 完整测试套件
BASE_URL=http://192.168.0.200:8080 \
npx playwright test specs/complete-e2e-test.spec.js \
  --config=playwright.config.js

# 显示浏览器
npx playwright test --headed

# 调试模式
npx playwright test --debug
```

#### 方式 3：使用 package.json 脚本

```bash
cd test/e2e

# 安装依赖
npm install

# 运行快速测试
npm run test:quick

# 运行完整测试
npm run test:full

# 显示浏览器
npm run test:headed
```

## 环境变量配置

测试支持以下环境变量：

```bash
# 基础 URL（必需）
export BASE_URL=http://192.168.0.200:8080

# 管理员凭证
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD=admin123

# 测试用户凭证（可选）
export TEST_USERNAME=testuser
export TEST_PASSWORD=test123
```

## 测试报告

### 查看测试结果

测试完成后，结果保存在：
```
test/e2e/test-results/
├── test-results.json        # 测试结果 JSON
├── screenshots/              # 失败截图
└── videos/                   # 失败视频（如果启用）
```

### 生成 HTML 报告

```bash
cd test/e2e
npx playwright show-report
```

## 常见问题

### 1. 浏览器未安装

**错误**：
```
browserType.launch: Executable doesn't exist
```

**解决**：
```bash
cd test/e2e
npx playwright install chromium
```

### 2. 服务未启动

**错误**：
```
page.goto: net::ERR_CONNECTION_REFUSED
```

**解决**：
```bash
# 启动服务
docker-compose up -d

# 检查服务状态
docker-compose ps

# 查看日志
docker-compose logs -f frontend
```

### 3. 超时错误

**错误**：
```
Timeout 30000ms exceeded
```

**解决**：
- 增加超时时间（修改 `playwright.config.js`）
- 检查网络连接
- 确保服务响应正常

### 4. 登录失败

**错误**：
```
Timed out waiting for navigation to **/projects
```

**解决**：
- 检查管理员凭证是否正确
- 验证用户是否存在于系统中
- 查看后端日志排查认证问题

### 5. 元素未找到

**错误**：
```
locator.click: Target closed
```

**解决**：
- 使用 `--headed` 模式查看页面
- 增加等待时间
- 检查选择器是否正确

## 最佳实践

### 1. 测试前准备

```bash
# 清理旧数据
docker-compose down -v

# 重新启动服务
docker-compose up -d

# 等待服务就绪
sleep 30

# 运行测试
./run-e2e-tests.sh --quick
```

### 2. 调试测试

```bash
# 显示浏览器窗口
./run-e2e-tests.sh --quick --headed

# 使用调试模式
npx playwright test --debug

# 查看特定测试
npx playwright test specs/quick-validation-test.spec.js --grep "SLURM Tasks"
```

### 3. CI/CD 集成

在 CI/CD 管道中运行测试：

```yaml
# .github/workflows/e2e-test.yml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Start services
        run: docker-compose up -d
      
      - name: Wait for services
        run: sleep 30
      
      - name: Run E2E tests
        run: ./run-e2e-tests.sh --quick
        env:
          BASE_URL: http://localhost:8080
          ADMIN_USERNAME: admin
          ADMIN_PASSWORD: admin123
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v2
        with:
          name: test-results
          path: test/e2e/test-results/
```

## 测试维护

### 添加新测试

1. 在 `test/e2e/specs/` 创建新的测试文件
2. 导入必要的依赖
3. 编写测试用例
4. 运行验证

示例：

```javascript
const { test, expect } = require('@playwright/test');

test.describe('新功能测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, 'admin', 'admin123');
  });

  test('测试新功能', async ({ page }) => {
    await page.goto('/new-feature');
    await expect(page.locator('text=新功能')).toBeVisible();
  });
});
```

### 更新现有测试

1. 识别需要更新的测试
2. 修改选择器或断言
3. 运行测试验证
4. 提交更改

## 性能优化建议

### 1. 并行执行测试

修改 `playwright.config.js`：

```javascript
module.exports = {
  workers: 4, // 并行执行
  fullyParallel: true,
};
```

### 2. 减少等待时间

使用智能等待而不是固定延迟：

```javascript
// ❌ 不推荐
await page.waitForTimeout(5000);

// ✅ 推荐
await page.waitForSelector('.element', { state: 'visible' });
```

### 3. 复用登录状态

使用全局 setup 文件：

```javascript
// global-setup.js
module.exports = async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await login(page);
  await page.context().storageState({ path: 'auth.json' });
  await browser.close();
};
```

## 相关文档

- [Playwright 官方文档](https://playwright.dev/)
- [BUILD_AND_TEST_GUIDE.md](./BUILD_AND_TEST_GUIDE.md) - 构建和测试指南
- [SLURM_TASKS_REFRESH_OPTIMIZATION.md](./SLURM_TASKS_REFRESH_OPTIMIZATION.md) - 刷新优化文档
- [FRONTEND_PAGE_FIXES.md](./FRONTEND_PAGE_FIXES.md) - 前端修复汇总

## 联系方式

如有问题或建议，请提交 Issue 或联系开发团队。

---

最后更新：2025-01-12
