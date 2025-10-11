// @ts-nocheck
/* eslint-disable */
// SaltStack 执行命令完成状态修复 - 验证测试
// 这个测试验证 SaltStack 命令执行完成后,按钮状态能正确恢复(不再一直转圈)

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

async function loginIfNeeded(page) {
  await page.goto(BASE + '/');
  
  const isLoggedIn = await page.locator('text=admin').isVisible().catch(() => false);
  
  if (!isLoggedIn) {
    const hasLoginTab = await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false);
    
    if (hasLoginTab) {
      const user = process.env.E2E_USER || 'admin';
      const pass = process.env.E2E_PASS || 'admin123';
      
      await page.getByPlaceholder('用户名').fill(user);
      await page.getByPlaceholder('密码').fill(pass);
      await page.getByRole('button', { name: /登\s*录/ }).click();
      
      await page.waitForLoadState('networkidle');
    }
  }
}

test('验证 SaltStack 命令执行完成后按钮状态正确恢复', async ({ page }) => {
  console.log('========================================');
  console.log('测试: SaltStack 执行完成状态修复验证');
  console.log('========================================\n');
  
  // 1. 登录
  console.log('步骤 1: 登录系统...');
  await loginIfNeeded(page);
  console.log('✓ 登录成功\n');
  
  // 2. 导航到 SaltStack 页面
  console.log('步骤 2: 打开 SaltStack 页面...');
  await page.goto(BASE + '/saltstack', { waitUntil: 'load' });
  
  // 等待加载状态消失
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 15000 });
  } catch (e) {
    // 可能加载很快
  }
  
  await page.waitForTimeout(2000);
  
  // 验证页面加载成功
  const pageTitle = await page.locator('text=/SaltStack/').first().isVisible();
  if (!pageTitle) {
    console.log('❌ 页面加载失败: 未找到 SaltStack 标题');
    await page.screenshot({ path: 'test-failed-page-not-loaded.png', fullPage: true });
    throw new Error('SaltStack 页面未正确加载');
  }
  
  await page.screenshot({ path: 'test-01-saltstack-page-loaded.png', fullPage: true });
  console.log('✓ SaltStack 页面加载成功\n');
  
  // 3. 点击"执行命令"按钮
  console.log('步骤 3: 打开执行命令对话框...');
  
  // 等待一下确保按钮已渲染
  await page.waitForTimeout(1000);
  
  // 使用更精确的选择器
  const execCmdButton = page.locator('button:has-text("执行命令")').first();
  const isButtonVisible = await execCmdButton.isVisible().catch(() => false);
  
  if (!isButtonVisible) {
    console.log('❌ 未找到"执行命令"按钮');
    const buttons = await page.locator('button').all();
    console.log('页面上的所有按钮:', buttons.length);
    for (let i = 0; i < Math.min(buttons.length, 10); i++) {
      const text = await buttons[i].textContent();
      console.log(`  按钮 ${i}: "${text}"`);
    }
    await page.screenshot({ path: 'test-failed-no-exec-button.png', fullPage: true });
    throw new Error('未找到执行命令按钮');
  }
  
  await execCmdButton.click();
  
  // 等待对话框出现
  await expect(page.getByText('执行自定义命令')).toBeVisible({ timeout: 5000 });
  await page.screenshot({ path: 'test-02-execution-dialog-opened.png' });
  console.log('✓ 执行命令对话框打开成功\n');
  
  // 4. 输入测试命令
  console.log('步骤 4: 输入测试命令...');
  const codeTextarea = page.getByLabel('代码');
  await codeTextarea.clear();
  await codeTextarea.fill('echo "E2E Test - Verifying execution completion" && sleep 1 && date');
  console.log('✓ 测试命令输入完成\n');
  
  // 5. 执行命令
  console.log('步骤 5: 执行命令...');
  const executeButton = page.getByRole('button', { name: /执 行/ });
  await executeButton.click();
  console.log('✓ 执行按钮已点击\n');
  
  // 6. 验证按钮进入禁用状态
  console.log('步骤 6: 验证执行期间按钮状态...');
  await expect(executeButton).toBeDisabled({ timeout: 2000 });
  console.log('✓ 执行期间按钮正确进入禁用状态\n');
  
  // 7. 等待执行完成 - 核心测试点!
  console.log('步骤 7: 等待执行完成(最多 40 秒)...');
  console.log('这是核心测试:验证修复后按钮能在执行完成后恢复可用状态');
  console.log('如果修复前,按钮会一直保持禁用(转圈)状态');
  console.log('如果修复后,按钮应该在执行完成后变为可用\n');
  
  try {
    // 等待按钮恢复为可用状态
    await expect(executeButton).toBeEnabled({ timeout: 40000 });
    
    console.log('========================================');
    console.log('✅ 测试通过!');
    console.log('========================================');
    console.log('执行完成后按钮状态正确恢复为可用状态');
    console.log('这证明之前的"一直转圈"问题已经修复!');
    console.log('========================================\n');
    
    await page.screenshot({ path: 'test-03-execution-completed-button-enabled.png', fullPage: true });
    
  } catch (error) {
    console.log('========================================');
    console.log('❌ 测试失败!');
    console.log('========================================');
    console.log('执行完成后按钮未能恢复为可用状态');
    console.log('这说明"一直转圈"问题仍然存在');
    console.log('========================================\n');
    
    await page.screenshot({ path: 'test-failed-button-still-disabled.png', fullPage: true });
    throw error;
  }
  
  // 8. 额外验证:尝试第二次执行
  console.log('步骤 8: 额外验证 - 尝试第二次执行...');
  await codeTextarea.clear();
  await codeTextarea.fill('echo "Second execution test"');
  await executeButton.click();
  
  // 验证第二次执行也能正常完成
  await expect(executeButton).toBeDisabled({ timeout: 2000 });
  await expect(executeButton).toBeEnabled({ timeout: 40000 });
  
  console.log('✅ 第二次执行也成功完成!');
  console.log('确认状态管理完全正常!\n');
  
  await page.screenshot({ path: 'test-04-second-execution-completed.png', fullPage: true });
  
  console.log('========================================');
  console.log('所有测试通过!修复有效!');
  console.log('========================================');
});
