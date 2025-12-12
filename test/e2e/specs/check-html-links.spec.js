// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';

test('检查 Nightingale HTML 中的链接', async ({ page }) => {
  await page.goto(`${BASE_URL}/login`);
  await page.waitForLoadState('networkidle');
  await page.fill('input[placeholder*="用户名"], input[id*="username"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"], button:has-text("登")');
  await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
  
  await page.goto(`${BASE_URL}/nightingale/`);
  await page.waitForTimeout(3000);
  
  const html = await page.content();
  
  // 检查 script 和 link 标签的 src/href
  const scriptSrcs = html.match(/src="([^"]+)"/g) || [];
  const linkHrefs = html.match(/href="([^"]+\.(?:css|js))"/g) || [];
  
  console.log('\n=== Script src 属性 ===');
  scriptSrcs.forEach(s => console.log(s));
  
  console.log('\n=== Link href 属性 (CSS/JS) ===');
  linkHrefs.forEach(h => console.log(h));
  
  // 检查是否有 /nightingale 前缀
  const hasNightingalePrefix = scriptSrcs.some(s => s.includes('/nightingale/'));
  console.log('\n资源是否有 /nightingale 前缀:', hasNightingalePrefix);
  
  // 检查菜单链接
  const menuLinks = await page.locator('a[href]').all();
  console.log('\n=== 菜单链接 href ===');
  for (const link of menuLinks.slice(0, 10)) {
    const href = await link.getAttribute('href');
    console.log(href);
  }
});
