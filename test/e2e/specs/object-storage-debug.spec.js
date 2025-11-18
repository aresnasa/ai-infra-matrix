/**
 * 对象存储页面调试测试
 */

const { test } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('对象存储页面调试', async ({ page }) => {
  console.log('\n=== 对象存储页面调试 ===\n');
  
  // 监听控制台消息
  page.on('console', msg => {
    console.log(`[Console ${msg.type()}] ${msg.text()}`);
  });
  
  // 监听页面错误
  page.on('pageerror', error => {
    console.log(`❌ Page Error: ${error.message}`);
  });
  
  // 监听失败的请求
  page.on('response', async res => {
    if (res.status() >= 400) {
      console.log(`❌ [${res.status()}] ${res.url()}`);
    }
  });
  
  // 1. 登录
  console.log('1. 登录系统...');
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|projects)/, { timeout: 10000 });
  console.log('   ✓ 登录成功\n');
  
  // 2. 导航到对象存储页面
  console.log('2. 访问对象存储页面...');
  await page.goto(`${BASE_URL}/object-storage`);
  await page.waitForTimeout(3000);
  console.log('   ✓ 页面加载完成\n');
  
  // 3. 检查页面内容
  console.log('3. 检查页面元素...');
  const pageContent = await page.content();
  
  // 检查是否有 iframe
  const iframes = await page.locator('iframe').count();
  console.log(`   找到 ${iframes} 个 iframe`);
  
  if (iframes > 0) {
    for (let i = 0; i < iframes; i++) {
      const iframe = page.locator('iframe').nth(i);
      const src = await iframe.getAttribute('src');
      const title = await iframe.getAttribute('title');
      console.log(`   iframe ${i + 1}:`);
      console.log(`     src: ${src}`);
      console.log(`     title: ${title}`);
    }
  }
  
  // 检查页面标题
  const title = await page.title();
  console.log(`\n   页面标题: ${title}`);
  
  // 检查是否有错误消息
  const errorMsg = await page.locator('.ant-alert-error').count();
  if (errorMsg > 0) {
    const errorText = await page.locator('.ant-alert-error').first().textContent();
    console.log(`\n   ⚠️  错误消息: ${errorText}`);
  }
  
  // 检查是否有加载状态
  const loading = await page.locator('.ant-spin').count();
  console.log(`   加载状态: ${loading > 0 ? '正在加载' : '加载完成'}`);
  
  // 4. 截图
  console.log('\n4. 保存截图...');
  await page.screenshot({ 
    path: 'test-screenshots/object-storage-debug.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存\n');
  
  // 5. 检查 MinIO 相关元素
  console.log('5. 检查 MinIO 元素...');
  const hasMinioText = pageContent.includes('MinIO') || pageContent.includes('minio');
  console.log(`   页面包含 MinIO 文本: ${hasMinioText}`);
  
  // 检查按钮
  const buttons = await page.locator('button').allTextContents();
  console.log(`\n   页面按钮: ${buttons.filter(b => b.trim()).join(', ')}`);
  
  // 6. 尝试查看 iframe 内容（如果有）
  if (iframes > 0) {
    console.log('\n6. 检查 iframe 内容...');
    const iframe = page.frameLocator('iframe').first();
    
    try {
      await page.waitForTimeout(2000);
      console.log('   ✓ iframe 可能已加载');
    } catch (e) {
      console.log(`   ⚠️  iframe 加载异常: ${e.message}`);
    }
  }
  
  console.log('\n=== 调试完成 ===\n');
});
