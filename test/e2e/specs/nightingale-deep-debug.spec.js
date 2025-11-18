/**
 * Nightingale 深度调试 - 检查 iframe 内部真实状态
 */

const { test, expect } = require('@playwright/test');

const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://192.168.0.200:8080',
  adminUser: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  },
};

test.use({
  viewport: { width: 1920, height: 1080 },
});

async function login(page, username, password) {
  await page.goto(TEST_CONFIG.baseURL + '/');
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

test('Nightingale iframe 深度检查', async ({ page }) => {
  const allRequests = [];
  const failedRequests = [];
  const consoleMessages = [];
  
  // 监听所有请求
  page.on('request', request => {
    const url = request.url();
    allRequests.push(url);
  });
  
  page.on('response', async response => {
    const url = response.url();
    const status = response.status();
    
    if (status >= 400) {
      failedRequests.push({ url, status });
      console.log(`❌ [${status}] ${url}`);
    }
    
    if (url.includes('nightingale')) {
      console.log(`[Nightingale] ${status} ${url}`);
    }
  });
  
  // 监听控制台
  page.on('console', msg => {
    const text = msg.text();
    consoleMessages.push({ type: msg.type(), text });
    
    if (msg.type() === 'error') {
      console.log(`❌ [Console Error] ${text}`);
    }
  });
  
  console.log('\n=== 登录并访问监控页面 ===\n');
  await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  
  await page.goto(TEST_CONFIG.baseURL + '/monitoring');
  await page.waitForLoadState('networkidle', { timeout: 20000 });
  
  console.log('\n=== 等待 20 秒让 Nightingale 完全加载 ===\n');
  await page.waitForTimeout(20000);
  
  // 获取 iframe
  const iframe = page.locator('iframe').first();
  const frameElement = await iframe.elementHandle();
  const frame = await frameElement.contentFrame();
  
  if (!frame) {
    console.log('❌ 无法获取 iframe frame');
    throw new Error('无法访问 iframe');
  }
  
  console.log('✅ 成功访问 iframe\n');
  
  // 检查 iframe 内的完整内容
  const frameHTML = await frame.content();
  console.log(`=== Iframe HTML 内容 ===`);
  console.log(`长度: ${frameHTML.length} 字符`);
  console.log(`\n前 2000 字符:\n${frameHTML.substring(0, 2000)}`);
  console.log(`\n后 1000 字符:\n${frameHTML.substring(frameHTML.length - 1000)}`);
  
  // 检查是否有 root div
  const rootDiv = await frame.locator('#root').count();
  console.log(`\n#root div 数量: ${rootDiv}`);
  
  if (rootDiv > 0) {
    const rootHTML = await frame.locator('#root').innerHTML();
    console.log(`#root 内容长度: ${rootHTML.length} 字符`);
    console.log(`#root 内容预览:\n${rootHTML.substring(0, 500)}`);
  }
  
  // 检查是否有任何可见文本
  const bodyText = await frame.locator('body').innerText();
  console.log(`\nIframe body 文本内容:\n${bodyText}`);
  
  // 检查是否有错误信息
  const hasError = bodyText.toLowerCase().includes('error') || 
                   bodyText.toLowerCase().includes('错误') ||
                   bodyText.toLowerCase().includes('failed');
  
  if (hasError) {
    console.log('\n⚠️  检测到错误信息！');
  }
  
  // 检查是否还在加载
  const hasLoading = bodyText.toLowerCase().includes('loading') ||
                     bodyText.toLowerCase().includes('加载') ||
                     frameHTML.includes('preloader');
  
  if (hasLoading) {
    console.log('\n⚠️  页面还在加载状态');
  }
  
  // 检查是否有实际的 Nightingale UI 元素
  const hasNightingaleUI = await frame.locator('[class*="n9e"]').count() > 0 ||
                           await frame.locator('[class*="nightingale"]').count() > 0 ||
                           await frame.locator('nav').count() > 0 ||
                           await frame.locator('[class*="menu"]').count() > 0;
  
  console.log(`\n是否有 Nightingale UI 元素: ${hasNightingaleUI ? '✓' : '✗'}`);
  
  // 列出所有可见元素
  const allElements = await frame.locator('body *').count();
  console.log(`\nIframe 内元素总数: ${allElements}`);
  
  // 检查所有 div
  const allDivs = await frame.locator('div').count();
  console.log(`div 元素数量: ${allDivs}`);
  
  // 截图 - 整个页面
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-deep-debug-full.png',
    fullPage: true 
  });
  
  // 只截 iframe
  await iframe.screenshot({
    path: 'test-screenshots/nightingale-deep-debug-iframe.png'
  });
  
  console.log('\n=== 失败请求统计 ===');
  console.log(`失败请求数: ${failedRequests.length}`);
  failedRequests.forEach(({ url, status }) => {
    console.log(`  [${status}] ${url}`);
  });
  
  console.log('\n=== Nightingale 相关请求 ===');
  const nightingaleRequests = allRequests.filter(url => url.includes('nightingale'));
  console.log(`总数: ${nightingaleRequests.length}`);
  
  console.log('\n=== 控制台错误 ===');
  const errors = consoleMessages.filter(m => m.type === 'error');
  console.log(`错误数: ${errors.length}`);
  errors.forEach(({ text }) => {
    console.log(`  ${text}`);
  });
  
  console.log('\n已保存截图:');
  console.log('  - test-screenshots/nightingale-deep-debug-full.png');
  console.log('  - test-screenshots/nightingale-deep-debug-iframe.png\n');
});
