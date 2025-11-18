// @ts-nocheck
/* eslint-disable */
/**
 * SaltStack 命令执行功能 E2E 测试 (简化版)
 * 
 * 说明：
 * 由于完整的测试套件在当前环境下存在页面加载问题,
 * 这个简化版本测试核心功能 - 确认 SaltStack 执行命令的完成状态能正确重置。
 * 
 * 主要测试点:
 * 1. 页面加载
 * 2. 执行命令对话框打开
 * 3. 命令执行
 * 4. **关键**: 执行完成后按钮状态正确恢复(修复的核心问题)
 */

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

// 登录辅助函数
async function login(page) {
  await page.goto(BASE + '/');
  await page.waitForLoadState('load');
  
  const user = process.env.E2E_USER || 'admin';
  const pass = process.env.E2E_PASS || 'admin123';
  
  const isLoggedIn = await page.locator('text=admin').isVisible().catch(() => false);
  
  if (!isLoggedIn) {
    await page.getByPlaceholder('用户名').fill(user);
    await page.getByPlaceholder('密码').fill(pass);
    await page.getByRole('button', { name: /登\s*录/ }).click();
    await page.waitForTimeout(2000);
  }
}

// 等待 SaltStack 页面加载
async function waitForSaltStackPage(page) {
  await page.goto(BASE + '/saltstack', { waitUntil: 'load' });
  
  // 等待加载状态消失
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 15000 });
  } catch (e) {
    // 可能加载很快,没有显示 loading 状态
  }
  
  // 确保页面标题出现
  await page.waitForSelector('h2:has-text("SaltStack")', { timeout: 15000 });
  await page.waitForTimeout(1000); // 给 React 一点时间完成渲染
}

test.describe('SaltStack 执行命令功能测试', () => {
  
  test('核心功能: 命令执行完成后按钮状态正确恢复', async ({ page }) => {
    // 1. 登录
    await login(page);
    console.log('✓ 登录成功');
    
    // 2. 打开 SaltStack 页面
    await waitForSaltStackPage(page);
    await page.screenshot({ path: 'test-saltstack-01-page-loaded.png' });
    console.log('✓ SaltStack 页面加载成功');
    
    // 3. 点击"执行命令"按钮
    const execButton = page.getByRole('button', { name: /执行命令/ });
    await expect(execButton).toBeVisible({ timeout: 10000 });
    await execButton.click();
    console.log('✓ 执行命令按钮点击成功');
    
    // 4. 等待对话框出现
    await expect(page.getByText('执行自定义命令')).toBeVisible({ timeout: 5000 });
    await page.screenshot({ path: 'test-saltstack-02-dialog-opened.png' });
    console.log('✓ 执行命令对话框打开成功');
    
    // 5. 输入简单的测试命令
    const codeTextarea = page.getByLabel('代码');
    await codeTextarea.clear();
    await codeTextarea.fill('echo "E2E Test" && date');
    console.log('✓ 命令输入完成');
    
    // 6. 执行命令
    const executeButton = page.getByRole('button', { name: /执 行/ });
    await executeButton.click();
    console.log('✓ 点击执行按钮');
    
    // 7. 验证按钮进入 loading 状态(禁用)
    await expect(executeButton).toBeDisabled({ timeout: 2000 });
    console.log('✓ 执行按钮正确进入禁用状态');
    
    // 8. 等待执行完成 (这是测试的关键 - 验证修复后按钮能正确恢复)
    // 如果修复有效,按钮应该在执行完成后恢复为可用状态
    await expect(executeButton).toBeEnabled({ timeout: 40000 });
    console.log('✅ 关键测试通过: 执行完成后按钮状态正确恢复为可用!');
    
    // 9. 截图保存最终状态
    await page.screenshot({ path: 'test-saltstack-03-execution-complete.png', fullPage: true });
    
    // 10. 可选:验证能够再次执行(证明状态真的恢复了)
    await codeTextarea.clear();
    await codeTextarea.fill('echo "Second execution"');
    await executeButton.click();
    
    // 再次验证
    await expect(executeButton).toBeDisabled({ timeout: 2000 });
    await expect(executeButton).toBeEnabled({ timeout: 40000 });
    console.log('✅ 第二次执行也成功: 确认状态管理完全正常!');
    
    await page.screenshot({ path: 'test-saltstack-04-second-execution.png', fullPage: true });
  });
  
});
