/**
 * Nightingale iframe 内容检查
 * 检查 iframe 是否成功加载 Nightingale
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

test('检查 Nightingale iframe 内容', async ({ page }) => {
  const allRequests = [];
  
  page.on('request', request => {
    const url = request.url();
    allRequests.push(url);
    if (url.includes('nightingale')) {
      console.log(`[请求] ${url}`);
    }
  });
  
  page.on('response', async response => {
    const url = response.url();
    const status = response.status();
    if (url.includes('nightingale')) {
      console.log(`[响应] ${status} ${url}`);
    }
  });
  
  console.log('\n=== 登录系统 ===\n');
  await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  
  console.log('\n=== 访问 /monitoring 页面 ===\n');
  await page.goto(TEST_CONFIG.baseURL + '/monitoring');
  
  // 等待网络空闲
  await page.waitForLoadState('networkidle', { timeout: 15000 });
  
  console.log('\n=== 检查 iframe ===\n');
  const iframe = page.locator('iframe');
  const iframeCount = await iframe.count();
  console.log(`iframe 数量: ${iframeCount}`);
  
  if (iframeCount > 0) {
    const iframeSrc = await iframe.getAttribute('src');
    console.log(`iframe src: ${iframeSrc}`);
    
    const iframeStyle = await iframe.getAttribute('style');
    console.log(`iframe style: ${iframeStyle}`);
    
    const iframeWidth = await iframe.evaluate(el => el.offsetWidth);
    const iframeHeight = await iframe.evaluate(el => el.offsetHeight);
    console.log(`iframe 尺寸: ${iframeWidth}x${iframeHeight}`);
    
    const isVisible = await iframe.isVisible();
    console.log(`iframe 可见: ${isVisible}`);
    
    // 等待更长时间让 iframe 加载
    console.log('\n等待 10 秒让 iframe 内容加载...\n');
    await page.waitForTimeout(10000);
    
    // 尝试访问 iframe 内容
    try {
      const frameElement = await iframe.elementHandle();
      const frame = await frameElement.contentFrame();
      
      if (frame) {
        console.log('✅ 成功获取 iframe frame');
        
        const frameURL = frame.url();
        console.log(`iframe URL: ${frameURL}`);
        
        // 检查 iframe 是否有内容
        const frameContent = await frame.content();
        console.log(`iframe HTML 长度: ${frameContent.length} 字符`);
        
        if (frameContent.length > 1000) {
          console.log(`iframe HTML 预览:\n${frameContent.substring(0, 800)}...`);
        } else {
          console.log(`iframe HTML 完整内容:\n${frameContent}`);
        }
        
        // 检查特定元素
        const frameBody = await frame.locator('body').innerHTML();
        console.log(`\niframe body 长度: ${frameBody.length} 字符`);
        
        if (frameBody.includes('nightingale') || frameBody.includes('n9e')) {
          console.log('✅ iframe 包含 Nightingale 内容');
        } else {
          console.log('⚠️  iframe 不包含 Nightingale 特征内容');
        }
      } else {
        console.log('❌ 无法获取 iframe frame');
      }
    } catch (error) {
      console.log(`❌ 访问 iframe 内容失败: ${error.message}`);
    }
  } else {
    console.log('❌ 没有找到 iframe');
  }
  
  // 统计 Nightingale 请求
  const nightingaleRequests = allRequests.filter(url => url.includes('nightingale'));
  console.log(`\n=== Nightingale 请求统计 ===`);
  console.log(`总共 ${nightingaleRequests.length} 个请求`);
  nightingaleRequests.forEach((url, index) => {
    console.log(`  ${index + 1}. ${url}`);
  });
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-iframe-content.png',
    fullPage: true 
  });
  console.log('\n已保存截图: test-screenshots/nightingale-iframe-content.png\n');
});
