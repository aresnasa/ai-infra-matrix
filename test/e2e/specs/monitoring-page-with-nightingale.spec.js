/**
 * 测试前端 MonitoringPage 页面中的 Nightingale iframe
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('通过前端 /monitoring 页面访问 Nightingale', async ({ page }) => {
  console.log(`\n=== 先访问首页检查登录状态 ===\n`);
  
  // 先访问首页
  await page.goto(`${BASE_URL}/`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  await page.waitForTimeout(2000);
  
  // 检查当前 URL，看是否被重定向到登录页
  const currentUrl = page.url();
  console.log(`当前 URL: ${currentUrl}`);
  
  if (currentUrl.includes('/login')) {
    console.log('⚠️  未登录，需要先登录');
    console.log('提示：这个测试需要已登录的管理员账户\n');
  }
  
  console.log(`\n=== 访问前端监控页面: ${BASE_URL}/monitoring ===\n`);
  
  // 访问前端的 monitoring 页面
  await page.goto(`${BASE_URL}/monitoring`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  console.log('✓ 前端页面已加载\n');
  
  // 等待页面加载
  await page.waitForTimeout(3000);
  
  // 检查是否有 iframe
  const iframeCount = await page.locator('iframe').count();
  console.log(`找到 ${iframeCount} 个 iframe`);
  
  if (iframeCount > 0) {
    // 获取 iframe 的 src
    const iframeSrc = await page.locator('iframe').first().getAttribute('src');
    console.log(`iframe src: ${iframeSrc}`);
    
    // 获取 iframe
    const iframe = page.frameLocator('iframe').first();
    
    // 等待 iframe 内容加载
    console.log('\n等待 iframe 内容加载...\n');
    await page.waitForTimeout(5000);
    
    // 检查 iframe 的尺寸
    const iframeElement = await page.locator('iframe').first();
    const box = await iframeElement.boundingBox();
    if (box) {
      console.log(`iframe 尺寸: ${box.width}x${box.height}`);
    }
    
    // 尝试在 iframe 中查找 Nightingale 的元素
    try {
      const iframeBody = iframe.locator('body');
      const iframeText = await iframeBody.innerText({ timeout: 5000 });
      console.log(`\niframe body 文本长度: ${iframeText.length} 字符`);
      
      if (iframeText.length > 100) {
        console.log('✓ iframe 内容已加载');
        console.log(`文本预览: ${iframeText.substring(0, 200)}...`);
      } else {
        console.log('⚠️  iframe 内容较少，可能未完全加载');
      }
    } catch (error) {
      console.log(`⚠️  无法读取 iframe 内容: ${error.message}`);
    }
  } else {
    console.log('⚠️  未找到 iframe，页面可能未正确加载');
  }
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/monitoring-page-with-iframe.png',
    fullPage: true 
  });
  console.log('\n✓ 截图已保存\n');
  
  // 验证至少有一个 iframe
  expect(iframeCount).toBeGreaterThan(0);
});
