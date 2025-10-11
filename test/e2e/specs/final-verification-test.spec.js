// @ts-nocheck
/* eslint-disable */
// SaltStack 执行完成状态修复验证测试 - 基于成功的 debug-saltstack 模式
const { test } = require('@playwright/test');

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

test('SaltStack 执行完成状态修复验证', async ({ page }) => {
  console.log('\n========================================');
  console.log('SaltStack 执行完成状态修复验证测试');
  console.log('========================================\n');
  
  // 1. 登录
  console.log('[1/8] 登录系统...');
  await loginIfNeeded(page);
  console.log('✓ 登录成功\n');
  
  // 2. 导航到 SaltStack 页面
  console.log('[2/8] 打开 SaltStack 页面...');
  await page.goto(BASE + '/saltstack', { waitUntil: 'load' });
  
  // 等待加载状态消失
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 15000 });
  } catch (e) {
    // 可能加载很快
  }
  
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'verification-01-page-loaded.png', fullPage: true });
  
  // 验证页面加载
  const hasSaltStackTitle = await page.locator('text=/SaltStack 配置管理/').isVisible().catch(() => false);
  if (!hasSaltStackTitle) {
    console.log('❌ 错误: SaltStack 页面未正确加载');
    throw new Error('页面加载失败');
  }
  console.log('✓ SaltStack 页面加载成功\n');
  
  // 3. 查找并点击"执行命令"按钮
  console.log('[3/8] 查找"执行命令"按钮...');
  const execButtons = await page.locator('button:has-text("执行命令")').all();
  console.log(`找到 ${execButtons.length} 个"执行命令"按钮`);
  
  if (execButtons.length === 0) {
    console.log('❌ 错误: 未找到"执行命令"按钮');
    await page.screenshot({ path: 'verification-error-no-button.png', fullPage: true });
    throw new Error('未找到执行命令按钮');
  }
  
  const execCmdButton = execButtons[0];
  await execCmdButton.click();
  console.log('✓ 点击"执行命令"按钮\n');
  
  // 4. 等待对话框打开
  console.log('[4/8] 等待执行对话框打开...');
  await page.waitForSelector('text=执行自定义命令', { timeout: 5000 });
  await page.screenshot({ path: 'verification-02-dialog-opened.png' });
  console.log('✓ 执行对话框打开成功\n');
  
  // 5. 输入测试命令
  console.log('[5/8] 输入测试命令...');
  const codeTextarea = page.getByLabel('代码');
  await codeTextarea.clear();
  await codeTextarea.fill('echo "Verification Test" && sleep 2 && date');
  console.log('✓ 命令输入完成\n');
  
  // 6. 执行命令
  console.log('[6/8] 执行命令...');
  const executeButtons = await page.locator('button:has-text("执 行")').all();
  if (executeButtons.length === 0) {
    console.log('❌ 错误: 未找到"执 行"按钮');
    throw new Error('未找到执行按钮');
  }
  
  const executeButton = executeButtons[0];
  await executeButton.click();
  console.log('✓ 已点击执行按钮\n');
  
  // 7. 验证按钮禁用状态
  console.log('[7/8] 验证执行期间按钮状态...');
  await page.waitForTimeout(500);
  const isDisabled = await executeButton.isDisabled();
  if (!isDisabled) {
    console.log('⚠️ 警告: 按钮未进入禁用状态');
  } else {
    console.log('✓ 按钮正确进入禁用状态\n');
  }
  
  // 8. 核心测试: 等待按钮恢复可用状态
  console.log('[8/8] **核心测试**: 等待执行完成并验证按钮恢复...');
  console.log('这是修复的关键: 执行完成后按钮应该恢复可用状态');
  console.log('(修复前会一直保持禁用/转圈状态)\n');
  
  console.log('等待最多 45 秒...');
  
  let buttonEnabled = false;
  const startTime = Date.now();
  const timeout = 45000; // 45 秒
  
  while (Date.now() - startTime < timeout) {
    const enabled = await executeButton.isEnabled();
    if (enabled) {
      buttonEnabled = true;
      const elapsedTime = ((Date.now() - startTime) / 1000).toFixed(1);
      console.log(`✅ 成功! 按钮在 ${elapsedTime} 秒后恢复可用状态`);
      break;
    }
    await page.waitForTimeout(500);
  }
  
  await page.screenshot({ path: 'verification-03-after-execution.png', fullPage: true });
  
  if (!buttonEnabled) {
    console.log('\n========================================');
    console.log('❌ 测试失败!');
    console.log('========================================');
    console.log('执行完成后按钮未能恢复可用状态');
    console.log('这说明"一直转圈"的问题仍然存在');
    console.log('========================================\n');
    throw new Error('按钮未能恢复可用状态 - 修复可能未生效');
  }
  
  console.log('\n========================================');
  console.log('✅✅✅ 测试通过! ✅✅✅');
  console.log('========================================');
  console.log('执行完成后按钮状态正确恢复!');
  console.log('这证明"一直转圈"的问题已经修复!');
  console.log('========================================\n');
  
  // 额外验证: 第二次执行
  console.log('额外验证: 尝试第二次执行...');
  await codeTextarea.clear();
  await codeTextarea.fill('echo "Second test"');
  await executeButton.click();
  
  await page.waitForTimeout(500);
  console.log('第二次执行已开始, 等待完成...');
  
  buttonEnabled = false;
  const startTime2 = Date.now();
  while (Date.now() - startTime2 < timeout) {
    const enabled = await executeButton.isEnabled();
    if (enabled) {
      buttonEnabled = true;
      const elapsedTime = ((Date.now() - startTime2) / 1000).toFixed(1);
      console.log(`✅ 第二次执行也成功! (耗时 ${elapsedTime} 秒)`);
      break;
    }
    await page.waitForTimeout(500);
  }
  
  await page.screenshot({ path: 'verification-04-second-execution.png', fullPage: true });
  
  if (buttonEnabled) {
    console.log('\n========================================');
    console.log('🎉 完美! 状态管理完全正常!');
    console.log('========================================\n');
  }
});
