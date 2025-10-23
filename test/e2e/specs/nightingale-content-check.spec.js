/**
 * Nightingale 页面内容检查
 * 检查 /monitoring 页面实际渲染的内容
 */

const { test, expect } = require('@playwright/test');

const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://192.168.0.200:8080',
  adminUser: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  },
};

async function login(page, username, password) {
  await page.goto(TEST_CONFIG.baseURL + '/');
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

test('检查 Nightingale 页面内容', async ({ page }) => {
  console.log('\n=== 登录系统 ===\n');
  await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  
  console.log('\n=== 访问 /monitoring 页面 ===\n');
  await page.goto(TEST_CONFIG.baseURL + '/monitoring');
  await page.waitForLoadState('networkidle', { timeout: 10000 });
  
  // 获取完整的页面 HTML
  const pageHTML = await page.content();
  console.log(`页面 HTML 长度: ${pageHTML.length} 字符`);
  
  // 检查是否有权限错误
  const bodyText = await page.locator('body').innerText();
  console.log(`\n=== 页面文本内容 ===`);
  console.log(bodyText);
  console.log(`\n=== 页面文本内容结束 ===\n`);
  
  // 检查关键元素
  console.log(`\n=== 检查关键元素 ===`);
  
  const has403 = await page.locator('text=403').count();
  console.log(`- 包含 "403": ${has403 > 0}`);
  
  const hasPermissionError = await page.locator('text=权限').count();
  console.log(`- 包含 "权限": ${hasPermissionError > 0}`);
  
  const hasTeamError = await page.locator('text=团队').count();
  console.log(`- 包含 "团队": ${hasTeamError > 0}`);
  
  const hasAdminError = await page.locator('text=管理员').count();
  console.log(`- 包含 "管理员": ${hasAdminError > 0}`);
  
  const hasMonitoring = await page.locator('text=监控').count();
  console.log(`- 包含 "监控": ${hasMonitoring > 0}`);
  
  const hasLoading = await page.locator('text=加载').count();
  console.log(`- 包含 "加载": ${hasLoading > 0}`);
  
  const iframeCount = await page.locator('iframe').count();
  console.log(`- iframe 数量: ${iframeCount}`);
  
  // 检查 div 结构
  const mainContainer = await page.locator('[class*="monitoring"]').count();
  console.log(`- 监控容器数量: ${mainContainer}`);
  
  // 获取所有可见的 div
  const allDivs = await page.locator('div').all();
  console.log(`\n总共 ${allDivs.length} 个 div 元素`);
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-content-check.png',
    fullPage: true 
  });
  console.log('\n已保存截图: test-screenshots/nightingale-content-check.png\n');
});
