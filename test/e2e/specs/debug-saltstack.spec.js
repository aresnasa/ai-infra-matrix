// @ts-nocheck
/* eslint-disable */
// 调试 SaltStack 页面
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

test('调试 SaltStack 页面', async ({ page }) => {
  await loginIfNeeded(page);
  
  // 导航到 SaltStack 页面
  await page.goto(BASE + '/saltstack', { waitUntil: 'load' });
  
  // 等待加载状态消失或数据出现
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 10000 });
  } catch (e) {
    console.log('Loading text still visible after 10 seconds');
  }
  
  await page.waitForTimeout(2000); // 再等待 2 秒让数据渲染
  
  // 截图
  await page.screenshot({ path: 'saltstack-page-debug.png', fullPage: true });
  
  // 查找所有标题
  const h1s = await page.locator('h1, h2, h3, h4').all();
  console.log('找到的标题数量:', h1s.length);
  for (let i = 0; i < h1s.length; i++) {
    const text = await h1s[i].textContent();
    console.log(`标题 ${i}: "${text}"`);
  }
  
  // 查找所有按钮
  const buttons = await page.locator('button').all();
  console.log('\n找到的按钮数量:', buttons.length);
  for (let i = 0; i < buttons.length; i++) {
    const text = await buttons[i].textContent();
    const visible = await buttons[i].isVisible();
    console.log(`按钮 ${i}: "${text}" (visible: ${visible})`);
  }
  
  // 查找所有包含 "Salt" 的文本
  const saltTexts = await page.locator('text=/Salt/i').all();
  console.log('\n找到的包含 Salt 的文本数量:', saltTexts.length);
  for (let i = 0; i < Math.min(saltTexts.length, 10); i++) {
    const text = await saltTexts[i].textContent();
    console.log(`Salt 文本 ${i}: "${text}"`);
  }
  
  await page.waitForTimeout(1000);
});
