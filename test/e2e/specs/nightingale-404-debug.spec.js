/**
 * Nightingale 404 错误调试
 * 捕获所有失败的请求 URL
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

test('捕获所有 404 请求 URL', async ({ page }) => {
  const failedRequests = [];
  const allRequests = [];
  
  // 监听所有请求
  page.on('request', request => {
    allRequests.push({
      url: request.url(),
      method: request.method()
    });
  });
  
  // 监听所有响应
  page.on('response', async response => {
    const status = response.status();
    const url = response.url();
    
    if (status === 404) {
      console.log(`❌ [404] ${url}`);
      failedRequests.push(url);
    } else if (status >= 400) {
      console.log(`⚠️  [${status}] ${url}`);
    }
  });
  
  // 监听控制台错误
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`❌ [Console Error] ${msg.text()}`);
    }
  });
  
  // 登录
  await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  
  console.log('\n=== 访问 Nightingale 页面 ===\n');
  
  // 访问监控页面
  await page.goto('/monitoring');
  await page.waitForLoadState('domcontentloaded', { timeout: 15000 });
  
  // 等待 iframe 和资源加载
  console.log('等待 10 秒让所有资源加载...');
  await page.waitForTimeout(10000);
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-404-debug.png',
    fullPage: true 
  });
  console.log('已保存截图: test-screenshots/nightingale-404-debug.png');
  
  // 输出结果
  console.log('\n=== 404 请求汇总 ===');
  console.log(`总请求数: ${allRequests.length}`);
  console.log(`404 错误数: ${failedRequests.length}\n`);
  
  if (failedRequests.length > 0) {
    console.log('404 URLs:');
    failedRequests.forEach((url, index) => {
      console.log(`  ${index + 1}. ${url}`);
    });
  } else {
    console.log('✅ 没有 404 错误！');
  }
  
  console.log('\n=== 调试完成 ===\n');
});
