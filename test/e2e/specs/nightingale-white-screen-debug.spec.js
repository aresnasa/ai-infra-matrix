/**
 * Nightingale 白屏调试 - 详细版本
 * 捕获所有控制台消息、网络请求和页面错误
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

test('Nightingale 白屏完整调试', async ({ page }) => {
  const allRequests = [];
  const failedRequests = [];
  const consoleMessages = [];
  const pageErrors = [];
  
  // 监听所有请求
  page.on('request', request => {
    const url = request.url();
    const method = request.method();
    allRequests.push({ url, method, timestamp: Date.now() });
    console.log(`[请求] ${method} ${url}`);
  });
  
  // 监听所有响应
  page.on('response', async response => {
    const status = response.status();
    const url = response.url();
    const contentType = response.headers()['content-type'] || '';
    
    if (status >= 400) {
      console.log(`❌ [${status}] ${url}`);
      failedRequests.push({ url, status, contentType });
    } else if (url.includes('nightingale')) {
      console.log(`✓ [${status}] ${url}`);
    }
  });
  
  // 监听所有控制台消息
  page.on('console', msg => {
    const text = msg.text();
    const type = msg.type();
    const location = msg.location();
    
    consoleMessages.push({ type, text, location });
    
    if (type === 'error') {
      console.log(`❌ [Console Error] ${text}`);
      console.log(`   Location: ${location.url}:${location.lineNumber}`);
    } else if (type === 'warning') {
      console.log(`⚠️  [Console Warning] ${text}`);
    } else if (text.includes('nightingale') || text.includes('Nightingale')) {
      console.log(`ℹ️  [Console] ${text}`);
    }
  });
  
  // 监听页面错误
  page.on('pageerror', error => {
    console.log(`❌ [Page Error] ${error.message}`);
    console.log(`   Stack: ${error.stack}`);
    pageErrors.push({ message: error.message, stack: error.stack });
  });
  
  // 登录
  console.log('\n=== 登录系统 ===\n');
  await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  
  console.log('\n=== 访问 Nightingale 页面 ===\n');
  await page.goto(TEST_CONFIG.baseURL + '/monitoring');
  await page.waitForLoadState('domcontentloaded', { timeout: 15000 });
  
  // 检查 iframe 是否存在
  const iframe = page.locator('iframe');
  const iframeCount = await iframe.count();
  console.log(`\nIframe 数量: ${iframeCount}`);
  
  if (iframeCount > 0) {
    const iframeSrc = await iframe.getAttribute('src');
    console.log(`Iframe src: ${iframeSrc}`);
    
    // 等待 iframe 加载
    console.log('\n等待 15 秒让 iframe 加载...\n');
    await page.waitForTimeout(15000);
    
    // 尝试访问 iframe 内容
    try {
      const frameElement = await iframe.elementHandle();
      const frame = await frameElement.contentFrame();
      
      if (frame) {
        console.log('✓ 成功获取 iframe frame 对象');
        
        const frameURL = frame.url();
        console.log(`Iframe URL: ${frameURL}`);
        
        // 检查 iframe 内的 body
        const bodyLocator = frame.locator('body');
        const bodyHTML = await bodyLocator.innerHTML({ timeout: 5000 }).catch(() => null);
        
        if (bodyHTML) {
          console.log(`\nIframe body HTML 长度: ${bodyHTML.length} 字符`);
          console.log(`Iframe body HTML 预览:\n${bodyHTML.substring(0, 500)}...`);
          
          // 检查是否包含 Nightingale 特征内容
          if (bodyHTML.includes('nightingale') || bodyHTML.includes('n9e') || bodyHTML.includes('登录')) {
            console.log('✓ Iframe 包含 Nightingale 内容');
          } else {
            console.log('⚠️  Iframe 内容可能异常');
          }
        } else {
          console.log('⚠️  无法获取 iframe body HTML');
        }
      } else {
        console.log('❌ 无法获取 iframe frame 对象');
      }
    } catch (error) {
      console.log(`❌ 访问 iframe 异常: ${error.message}`);
    }
  }
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-white-screen-debug.png',
    fullPage: true 
  });
  console.log('\n已保存截图: test-screenshots/nightingale-white-screen-debug.png');
  
  // 输出统计
  console.log('\n=== 统计信息 ===');
  console.log(`总请求数: ${allRequests.length}`);
  console.log(`失败请求数: ${failedRequests.length}`);
  console.log(`控制台错误数: ${consoleMessages.filter(m => m.type === 'error').length}`);
  console.log(`页面错误数: ${pageErrors.length}`);
  
  if (failedRequests.length > 0) {
    console.log('\n失败的请求:');
    failedRequests.forEach(({ url, status }, index) => {
      console.log(`  ${index + 1}. [${status}] ${url}`);
    });
  }
  
  if (pageErrors.length > 0) {
    console.log('\n页面错误详情:');
    pageErrors.forEach((error, index) => {
      console.log(`  ${index + 1}. ${error.message}`);
      if (error.stack) {
        console.log(`     Stack: ${error.stack.split('\n')[0]}`);
      }
    });
  }
  
  // 检查特定的 Nightingale 请求
  const nightingaleRequests = allRequests.filter(r => r.url.includes('/nightingale/'));
  console.log(`\nNightingale 相关请求数: ${nightingaleRequests.length}`);
  
  console.log('\n=== 调试完成 ===\n');
});
