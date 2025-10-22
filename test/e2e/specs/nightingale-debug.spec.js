/**
 * Nightingale 调试测试
 * 详细检查 /nightingale/ 路径返回的内容
 */

const { test, expect } = require('@playwright/test');

test.describe('Nightingale 路径调试', () => {
  test('检查 /nightingale/ 返回的内容', async ({ page }) => {
    // 先登录
    await page.goto('/');
    
    // 等待登录表单
    await page.waitForSelector('input[type="text"]', { timeout: 5000 });
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'Admin@123');
    await page.click('button[type="submit"]');
    
    // 等待登录完成
    await page.waitForURL(/\/dashboard|\/project/, { timeout: 10000 });
    
    console.log('\n=== 访问 /nightingale/ ===');
    
    // 访问 /nightingale/
    const response = await page.goto('/nightingale/');
    console.log(`状态码: ${response.status()}`);
    console.log(`Content-Type: ${response.headers()['content-type']}`);
    
    // 获取页面内容
    const content = await page.content();
    console.log(`\n页面内容长度: ${content.length} 字符`);
    
    // 检查是否是 HTML
    const isHTML = content.includes('<html') || content.includes('<!DOCTYPE');
    console.log(`是否是 HTML: ${isHTML}`);
    
    // 检查是否包含前端应用的特征
    const hasFrontendApp = content.includes('AI-Infra-Matrix') && content.includes('React');
    console.log(`包含前端应用特征: ${hasFrontendApp}`);
    
    // 检查是否包含 Nightingale 的特征
    const hasNightingale = content.includes('nightingale') || content.includes('Nightingale');
    console.log(`包含 Nightingale 关键字: ${hasNightingale}`);
    
    // 检查页面标题
    const title = await page.title();
    console.log(`页面标题: ${title}`);
    
    // 检查是否有 iframe
    const iframes = await page.locator('iframe').count();
    console.log(`Iframe 数量: ${iframes}`);
    
    // 检查控制台错误
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.waitForTimeout(2000);
    
    if (consoleErrors.length > 0) {
      console.log(`\n控制台错误 (${consoleErrors.length} 个):`);
      consoleErrors.slice(0, 5).forEach((err, i) => {
        console.log(`  ${i + 1}. ${err}`);
      });
    }
    
    // 获取页面快照
    console.log('\n=== 页面结构快照 ===');
    const bodyText = await page.locator('body').textContent();
    console.log(`Body 文本内容前 200 字符:\n${bodyText.substring(0, 200)}`);
    
    // 检查是否有导航栏
    const hasNav = await page.locator('nav, [role="navigation"], .ant-menu').count();
    console.log(`\n导航元素数量: ${hasNav}`);
    
    // 检查关键元素
    const hasHeader = await page.locator('header, .ant-layout-header').count();
    console.log(`Header 元素数量: ${hasHeader}`);
    
    const hasMain = await page.locator('main, .ant-layout-content').count();
    console.log(`Main 元素数量: ${hasMain}`);
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-debug.png',
      fullPage: true 
    });
    console.log('\n截图已保存: test-screenshots/nightingale-debug.png');
  });
  
  test('直接请求 Nightingale 服务', async ({ request }) => {
    console.log('\n=== 测试直接访问 Nightingale 服务 ===');
    
    // 尝试直接访问 nightingale 服务（通过 nginx 代理）
    try {
      const response = await request.get('http://192.168.18.114:8080/nightingale/', {
        headers: {
          'Cookie': 'your-session-cookie-here' // 需要实际的 session cookie
        }
      });
      
      console.log(`状态码: ${response.status()}`);
      console.log(`Headers: ${JSON.stringify(response.headers(), null, 2)}`);
      
      const body = await response.text();
      console.log(`响应长度: ${body.length} 字符`);
      console.log(`前 500 字符:\n${body.substring(0, 500)}`);
    } catch (error) {
      console.error(`请求失败: ${error.message}`);
    }
  });
});
