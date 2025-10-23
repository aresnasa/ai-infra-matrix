/**
 * 测试登录后访问监控页面
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('登录后访问监控页面', async ({ page }) => {
  console.log(`\n=== 步骤 1: 访问登录页面 ===\n`);
  
  await page.goto(`${BASE_URL}/login`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  // 等待登录表单加载
  await page.waitForTimeout(2000);
  
  // 检查是否已经登录（如果有 token 可能会自动跳转）
  const currentUrl = page.url();
  console.log(`当前 URL: ${currentUrl}`);
  
  if (!currentUrl.includes('/login')) {
    console.log('✓ 已自动登录，跳过登录步骤\n');
  } else {
    console.log('需要手动登录\n');
    
    // 尝试填写登录表单
    try {
      await page.fill('input[type="text"], input[name="username"]', 'admin', { timeout: 5000 });
      await page.fill('input[type="password"], input[name="password"]', 'admin123', { timeout: 5000 });
      
      // 点击登录按钮
      await page.click('button[type="submit"]', { timeout: 5000 });
      
      console.log('✓ 已提交登录表单\n');
      await page.waitForTimeout(3000);
      
      // 检查是否登录成功
      const afterLoginUrl = page.url();
      console.log(`登录后 URL: ${afterLoginUrl}\n`);
      
      if (afterLoginUrl.includes('/login')) {
        console.log('⚠️ 登录可能失败，仍在登录页\n');
      } else {
        console.log('✓ 登录成功\n');
      }
    } catch (error) {
      console.log(`登录表单填写失败: ${error.message}\n`);
    }
  }
  
  console.log(`\n=== 步骤 2: 访问监控页面 ===\n`);
  
  await page.goto(`${BASE_URL}/monitoring`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  await page.waitForTimeout(3000);
  
  const finalUrl = page.url();
  console.log(`最终 URL: ${finalUrl}\n`);
  
  // 检查页面标题
  const title = await page.title();
  console.log(`页面标题: ${title}\n`);
  
  // 检查页面内容
  const bodyText = await page.locator('body').innerText();
  console.log(`页面文本长度: ${bodyText.length} 字符\n`);
  
  if (bodyText.includes('Monitoring') || bodyText.includes('监控')) {
    console.log('✓ 找到监控相关文本\n');
  }
  
  // 检查 iframe
  const iframeCount = await page.locator('iframe').count();
  console.log(`找到 ${iframeCount} 个 iframe\n`);
  
  if (iframeCount > 0) {
    const iframeSrc = await page.locator('iframe').first().getAttribute('src');
    console.log(`iframe src: ${iframeSrc}\n`);
    
    // 获取 iframe 尺寸
    const box = await page.locator('iframe').first().boundingBox();
    if (box) {
      console.log(`iframe 尺寸: ${box.width}x${box.height}\n`);
      
      if (box.height > 100) {
        console.log('✓ iframe 高度正常\n');
      } else {
        console.log('⚠️ iframe 高度过小\n');
      }
    }
    
    // 尝试读取 iframe 内容
    try {
      await page.waitForTimeout(3000);
      const iframe = page.frameLocator('iframe').first();
      const iframeBody = iframe.locator('body');
      const iframeText = await iframeBody.innerText({ timeout: 10000 });
      
      console.log(`iframe 内容长度: ${iframeText.length} 字符\n`);
      
      if (iframeText.length > 100) {
        console.log('✓ iframe 内容已加载\n');
        console.log(`内容预览: ${iframeText.substring(0, 200)}...\n`);
      }
    } catch (error) {
      console.log(`读取 iframe 内容失败: ${error.message}\n`);
    }
  } else {
    console.log('⚠️ 未找到 iframe\n');
    
    // 检查是否有错误消息
    if (bodyText.includes('403') || bodyText.includes('权限') || bodyText.includes('Forbidden')) {
      console.log('⚠️ 可能是权限问题\n');
    }
    
    if (bodyText.includes('404')) {
      console.log('⚠️ 页面未找到\n');
    }
    
    console.log(`页面内容预览:\n${bodyText.substring(0, 500)}\n`);
  }
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/monitoring-with-login.png',
    fullPage: true 
  });
  console.log('✓ 截图已保存\n');
  
  // 基本断言
  expect(bodyText.length).toBeGreaterThan(0);
});
