/**
 * 以有头浏览器模式运行，观察真实加载过程
 */

const { test } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('有头浏览器观察 Nightingale 加载过程', async ({ page }) => {
  console.log('\n⏳ 打开浏览器窗口，观察加载过程...\n');
  console.log('   URL: ' + BASE_URL + '/nightingale/\n');
  
  await page.goto(`${BASE_URL}/nightingale/`);
  
  // 每 2 秒检查一次状态，持续 30 秒
  for (let i = 0; i < 15; i++) {
    const preloaderVisible = await page.locator('.preloader').isVisible().catch(() => false);
    const rootLength = await page.locator('#root').innerHTML().then(html => html.length);
    const appCount = await page.locator('.App').count();
    
    console.log(`[${i * 2}秒] 预加载: ${preloaderVisible ? '✓ 显示' : '✗ 隐藏'} | #root: ${rootLength}字符 | .App: ${appCount}个`);
    
    if (!preloaderVisible && rootLength > 1000 && appCount > 0) {
      console.log('\n✅ 页面加载完成！\n');
      break;
    }
    
    await page.waitForTimeout(2000);
  }
  
  // 保持浏览器打开 10 秒让用户观察
  console.log('浏览器将保持打开 10 秒，请观察页面...\n');
  await page.waitForTimeout(10000);
});
