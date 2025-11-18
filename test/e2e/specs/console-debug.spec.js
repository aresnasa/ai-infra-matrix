// @ts-nocheck
/* eslint-disable */
// 调试控制台错误
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
      
      await page.waitForTimeout(2000);
      await page.waitForLoadState('load');
    }
  }
}

test('检查控制台错误', async ({ page }) => {
  const messages = [];
  
  page.on('console', msg => {
    messages.push(`[${msg.type()}] ${msg.text()}`);
  });
  
  page.on('pageerror', err => {
    console.log('PAGE ERROR:', err.message);
  });
  
  page.on('requestfailed', request => {
    console.log('FAILED REQUEST:', request.url(), request.failure().errorText);
  });
  
  await loginIfNeeded(page);
  
  console.log('=== After login ===');
  messages.forEach(m => console.log(m));
  messages.length = 0;
  
  await page.goto(BASE + '/saltstack');
  await page.waitForLoadState('load');
  await page.waitForTimeout(5000);
  
  console.log('=== After saltstack navigation ===');
  messages.forEach(m => console.log(m));
});
