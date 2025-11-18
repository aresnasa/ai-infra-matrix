// @ts-nocheck
/* eslint-disable */
// 简单的 SaltStack 测试 - 用于调试问题
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
      
      await expect(page).toHaveURL(/\/(projects|dashboard|saltstack)?$/, { timeout: 10000 });
      await page.waitForLoadState('load');
    }
  }
}

test('简单测试 SaltStack 页面加载', async ({ page }) => {
  await loginIfNeeded(page);
  
  console.log('Login complete, current URL:', page.url());
  
  await page.goto(BASE + '/saltstack');
  
  console.log('After goto, URL:', page.url());
  await page.waitForLoadState('load');
  await page.waitForLoadState('domcontentloaded');
  
  console.log('Waiting for React to render...');
  await page.waitForTimeout(2000);
  
  console.log('Taking screenshot...');
  await page.screenshot({ path: 'simple-saltstack-test.png', fullPage: true });
  
  // 查找页面上的文本
  const html = await page.content();
  console.log('HTML length:', html.length);
  console.log('Has SaltStack in HTML:', html.includes('SaltStack'));
  console.log('Has "配置管理" in HTML:', html.includes('配置管理'));
  
  // 检查是否有React root
  const hasReactRoot = await page.locator('#root').count();
  console.log('Has #root:', hasReactRoot);
  
  const titles = await page.locator('h1, h2, h3').all();
  console.log('Found titles:', titles.length);
  for (const title of titles) {
    console.log('Title:', await title.textContent());
  }
  
  // 等待加载状态消失
  console.log('Waiting for loading text to disappear...');
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 15000 });
    console.log('Loading text hidden');
  } catch (e) {
    console.log('Loading text did not appear or disappear');
  }
  
  await page.waitForTimeout(3000);
  
  await page.screenshot({ path: 'simple-saltstack-test-after-wait.png', fullPage: true });
  
  const titles2 = await page.locator('h1, h2, h3').all();
  console.log('Found titles after wait:', titles2.length);
  for (const title of titles2) {
    console.log('Title after wait:', await title.textContent());
  }
  
  // 检查是否有错误提示
  const errors = await page.locator('text=/error|错误|失败/i').all();
  console.log('Found error messages:', errors.length);
  for (const error of errors.slice(0, 3)) {
    console.log('Error:', await error.textContent());
  }
});
