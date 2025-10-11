# Playwright E2E 测试使用指南

## 快速开始

### 1. 安装依赖(如果尚未安装)
```bash
cd test/e2e
npm install @playwright/test
npx playwright install chromium
```

### 2. 运行核心验证测试
```bash
BASE_URL=http://192.168.0.200:8080 npx playwright test specs/final-verification-test.spec.js
```

### 3. 运行调试测试
```bash
# 页面加载测试
BASE_URL=http://192.168.0.200:8080 npx playwright test specs/debug-saltstack.spec.js

# 登录测试
BASE_URL=http://192.168.0.200:8080 npx playwright test specs/debug-login.spec.js
```

### 4. 运行所有测试
```bash
BASE_URL=http://192.168.0.200:8080 npx playwright test specs/
```

## 测试文件说明

### 推荐使用的测试

#### `specs/final-verification-test.spec.js` ⭐ **推荐**
- **用途**: 验证 SaltStack 执行完成状态修复
- **测试内容**:
  1. 登录系统
  2. 打开 SaltStack 页面
  3. 执行命令
  4. **验证执行完成后按钮状态正确恢复**(核心修复点)
  5. 验证可重复执行
- **运行命令**:
  ```bash
  BASE_URL=http://192.168.0.200:8080 npx playwright test specs/final-verification-test.spec.js
  ```
- **预期结果**: 测试通过,显示"✅✅✅ 测试通过! ✅✅✅"

#### `specs/debug-saltstack.spec.js`
- **用途**: 调试 SaltStack 页面加载
- **测试内容**:
  1. 登录
  2. 加载 SaltStack 页面
  3. 输出页面元素信息(标题、按钮等)
  4. 生成截图
- **运行命令**:
  ```bash
  BASE_URL=http://192.168.0.200:8080 npx playwright test specs/debug-saltstack.spec.js
  ```

### 调试辅助测试

#### `specs/debug-login.spec.js`
- 调试登录页面元素
- 输出按钮和输入框信息

#### `specs/console-debug.spec.js`
- 查看浏览器控制台输出
- 检测 JavaScript 错误

#### `specs/simple-saltstack-test.spec.js`
- 简单的页面加载测试
- HTML 内容检查

### 完整测试套件

#### `specs/saltstack-exec.spec.js`
- **状态**: 需要配置调整才能完全运行
- **包含**: 11个全面的测试用例
- **内容**:
  - 页面加载测试
  - 对话框打开测试
  - 命令执行测试
  - 错误处理测试
  - 表单验证测试
  - 实时进度测试
  - 重复执行测试
  - 页面状态显示测试

## 配置说明

### 环境变量
- `BASE_URL`: 测试目标URL (默认: http://localhost:8080)
- `E2E_USER`: 登录用户名 (默认: admin)
- `E2E_PASS`: 登录密码 (默认: admin123)

### Playwright 配置
配置文件: `playwright.config.js`
```javascript
{
  testDir: './specs',
  timeout: 60000,
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8080',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  }
}
```

## 常见问题

### Q: 测试失败显示"test.describe() not expected"
**A**: 将测试文件移到 `specs/` 目录,或使用不包含 `test.describe()` 的简化测试。

### Q: 找不到"执行命令"按钮
**A**: 页面可能还在加载,建议使用 `final-verification-test.spec.js`,它包含了正确的等待逻辑。

### Q: 如何查看测试截图?
**A**: 截图保存在 `test/e2e/` 目录下:
- 成功: `verification-01-page-loaded.png` 等
- 失败: `test-failed-*.png`

### Q: 如何以 UI 模式运行测试?
**A**: 
```bash
BASE_URL=http://192.168.0.200:8080 npx playwright test --ui
```

### Q: 如何以调试模式运行?
**A**:
```bash
BASE_URL=http://192.168.0.200:8080 npx playwright test --debug
```

## 测试最佳实践

1. **运行前确认服务可访问**
   ```bash
   curl http://192.168.0.200:8080
   ```

2. **清理旧的截图**
   ```bash
   rm -f *.png
   rm -rf test-results/
   ```

3. **查看详细日志**
   ```bash
   BASE_URL=http://192.168.0.200:8080 npx playwright test --reporter=list
   ```

4. **只运行特定测试**
   ```bash
   # 使用 --grep 过滤
   npx playwright test --grep "核心功能"
   ```

## 贡献指南

### 添加新测试
1. 在 `specs/` 目录创建新的 `.spec.js` 文件
2. 参考 `final-verification-test.spec.js` 的结构
3. 使用 `loginIfNeeded()` 辅助函数处理登录
4. 添加适当的等待和错误处理
5. 包含截图便于调试

### 测试模板
```javascript
const { test } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

test('我的测试', async ({ page }) => {
  // 登录
  await page.goto(BASE + '/');
  // ... 登录逻辑
  
  // 导航
  await page.goto(BASE + '/your-page');
  await page.waitForLoadState('load');
  
  // 测试逻辑
  // ...
  
  // 截图
  await page.screenshot({ path: 'my-test.png' });
});
```

## 相关文档
- [SALTSTACK_FIX_TEST_SUMMARY.md](../docs/SALTSTACK_FIX_TEST_SUMMARY.md) - 修复和测试总结
- [scripts/js/SALTSTACK-EXEC-FIX.md](../scripts/js/SALTSTACK-EXEC-FIX.md) - 详细修复说明
- [Playwright 官方文档](https://playwright.dev/)

---
最后更新: 2024-10-11
