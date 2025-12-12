// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';

test.describe('Nightingale 前端路由诊断', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"], input[id*="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"], button:has-text("登")');
    await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
  });

  test('检查 Nightingale 各个路由的响应', async ({ page }) => {
    console.log('\n=== 检查 Nightingale 路由 ===');
    
    // 测试不同的路由
    const routes = [
      '/nightingale/',                  // 根路径
      '/nightingale/metric/explorer',   // 指标查询
      '/nightingale/dashboards',        // 仪表盘
      '/nightingale/targets',           // 监控目标
    ];
    
    for (const route of routes) {
      const fullUrl = `${BASE_URL}${route}`;
      console.log(`\n测试路由: ${route}`);
      
      await page.goto(fullUrl);
      await page.waitForTimeout(3000);
      
      const bodyText = await page.textContent('body');
      const has404 = bodyText?.includes('404') || bodyText?.includes('不存在');
      const hasContent = (bodyText?.length || 0) > 200;
      
      console.log(`  - 页面文本长度: ${bodyText?.length}`);
      console.log(`  - 包含 404: ${has404}`);
      console.log(`  - 有实际内容: ${hasContent}`);
      
      // 截图
      const screenshotName = route.replace(/\//g, '_').replace(/^_/, '');
      await page.screenshot({ path: `test-screenshots/n9e_${screenshotName}.png` });
    }
  });

  test('检查 Nightingale 根路径的菜单链接', async ({ page }) => {
    console.log('\n=== 检查 Nightingale 菜单链接 ===');
    
    await page.goto(`${BASE_URL}/nightingale/`);
    await page.waitForTimeout(3000);
    
    // 获取所有链接
    const links = await page.locator('a[href]').all();
    const hrefs = new Set();
    
    for (const link of links) {
      const href = await link.getAttribute('href');
      if (href) hrefs.add(href);
    }
    
    console.log('页面中的链接:');
    [...hrefs].sort().forEach(href => console.log(`  ${href}`));
    
    // 点击第一个看起来有效的链接
    const targetLink = await page.locator('a[href="/metric/explorer"], a[href="/nightingale/metric/explorer"]').first();
    if (await targetLink.count() > 0) {
      console.log('\n点击指标查询链接...');
      await targetLink.click();
      await page.waitForTimeout(3000);
      
      console.log('点击后的 URL:', page.url());
      
      const bodyText = await page.textContent('body');
      console.log('点击后页面文本长度:', bodyText?.length);
      console.log('点击后是否包含 404:', bodyText?.includes('404'));
      
      await page.screenshot({ path: 'test-screenshots/n9e_after_click_metric.png' });
    }
  });
});
