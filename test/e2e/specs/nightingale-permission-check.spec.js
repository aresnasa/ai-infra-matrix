/**
 * Nightingale 权限检查
 * 检查 admin 用户是否能够访问 /monitoring 页面
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
  await page.goto('/');
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

test('检查 admin 用户团队权限', async ({ page }) => {
  console.log('\n=== 登录系统 ===\n');
  await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  
  // 检查用户信息
  console.log('\n=== 检查用户信息 ===\n');
  const userInfo = await page.evaluate(() => {
    return JSON.parse(localStorage.getItem('user') || '{}');
  });
  
  console.log('用户信息:', JSON.stringify(userInfo, null, 2));
  console.log(`role: ${userInfo.role}`);
  console.log(`role_template: ${userInfo.role_template || userInfo.roleTemplate || '未设置'}`);
  console.log(`team: ${userInfo.team || '未设置'}`);
  
  console.log('\n=== 访问 Nightingale 页面 ===\n');
  await page.goto('/monitoring');
  await page.waitForLoadState('domcontentloaded', { timeout: 10000 });
  
  // 等待内容加载
  await page.waitForTimeout(3000);
  
  // 检查是否显示权限不足
  const hasPermissionError = await page.locator('text=团队权限不足').count() > 0;
  const has403 = await page.locator('text=403').count() > 0;
  const hasSRERequirement = await page.locator('text=SRE运维团队').count() > 0;
  
  console.log(`\n权限检查结果:`);
  console.log(`- 显示 "团队权限不足": ${hasPermissionError}`);
  console.log(`- 显示 "403": ${has403}`);
  console.log(`- 要求 SRE 团队: ${hasSRERequirement}`);
  
  // 检查 iframe
  const iframeCount = await page.locator('iframe').count();
  console.log(`- Iframe 数量: ${iframeCount}`);
  
  // 获取页面文本内容
  const bodyText = await page.locator('body').innerText();
  console.log(`\n页面内容预览:\n${bodyText.substring(0, 500)}`);
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-permission-check.png',
    fullPage: true 
  });
  console.log('\n已保存截图: test-screenshots/nightingale-permission-check.png');
  
  // 诊断结果
  console.log('\n=== 诊断结果 ===');
  if (hasPermissionError || has403) {
    console.log('❌ 问题: admin 用户没有访问 /monitoring 的权限');
    console.log('   原因: TeamProtectedRoute 要求用户属于 sre 团队');
    console.log('   解决方案:');
    console.log('   1. 修改 App.js，为 admin 用户添加例外');
    console.log('   2. 给 admin 用户添加 role_template="sre"');
    console.log('   3. 修改 MonitoringPage 路由，不使用 TeamProtectedRoute');
  } else if (iframeCount > 0) {
    console.log('✓ admin 用户可以访问页面，iframe 已渲染');
  } else {
    console.log('⚠️  admin 用户可以访问页面，但 iframe 未渲染（其他原因）');
  }
  console.log('\n');
});
