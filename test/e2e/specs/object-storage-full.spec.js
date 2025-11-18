/**
 * 对象存储完整流程测试
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('对象存储完整流程测试', async ({ page }) => {
  console.log('\n=== 对象存储完整流程测试 ===\n');
  
  // 监听所有请求
  const failedRequests = [];
  page.on('response', async res => {
    if (res.status() >= 400) {
      failedRequests.push({ url: res.url(), status: res.status() });
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
  
  // 3. 检查页面状态
  console.log('3. 检查页面内容...');
  
  // 检查是否有配置列表
  const hasNoConfig = await page.locator('text=尚未配置对象存储').isVisible().catch(() => false);
  
  if (hasNoConfig) {
    console.log('   ⚠️  尚未配置对象存储');
    console.log('   需要先配置MinIO服务\n');
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/object-storage-no-config.png',
      fullPage: true 
    });
    
    return;
  }
  
  // 4. 查找存储配置
  console.log('4. 查找MinIO配置...');
  const configCards = await page.locator('.ant-card').count();
  console.log(`   找到 ${configCards} 个配置卡片`);
  
  // 查找MinIO相关的访问按钮
  const accessButtons = page.locator('button:has-text("访问")');
  const accessButtonCount = await accessButtons.count();
  console.log(`   找到 ${accessButtonCount} 个访问按钮\n`);
  
  if (accessButtonCount === 0) {
    console.log('   ⚠️  没有可访问的存储配置');
    await page.screenshot({ 
      path: 'test-screenshots/object-storage-no-access.png',
      fullPage: true 
    });
    return;
  }
  
  // 5. 截图配置列表
  await page.screenshot({ 
    path: 'test-screenshots/object-storage-list.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: object-storage-list.png\n');
  
  // 6. 点击第一个访问按钮
  console.log('5. 点击访问MinIO控制台...');
  const firstAccessBtn = accessButtons.first();
  
  // 检查按钮是否可用
  const isEnabled = await firstAccessBtn.isEnabled();
  console.log(`   访问按钮状态: ${isEnabled ? '可用' : '禁用'}`);
  
  if (!isEnabled) {
    console.log('   ⚠️  访问按钮被禁用，MinIO可能未连接\n');
    await page.screenshot({ 
      path: 'test-screenshots/object-storage-disabled.png',
      fullPage: true 
    });
    return;
  }
  
  await firstAccessBtn.click();
  await page.waitForTimeout(2000);
  
  // 等待跳转到MinIO控制台页面
  await page.waitForURL(/\/object-storage\/minio\/\d+/, { timeout: 10000 });
  console.log('   ✓ 跳转到MinIO控制台页面');
  console.log(`   当前URL: ${page.url()}\n`);
  
  // 7. 检查MinIO控制台页面
  console.log('6. 检查MinIO控制台...');
  await page.waitForTimeout(3000);
  
  // 检查是否有iframe
  const iframes = await page.locator('iframe').count();
  console.log(`   找到 ${iframes} 个 iframe`);
  
  if (iframes > 0) {
    const iframe = page.locator('iframe').first();
    const src = await iframe.getAttribute('src');
    const title = await iframe.getAttribute('title');
    console.log(`   iframe src: ${src}`);
    console.log(`   iframe title: ${title}`);
    
    // 检查iframe是否可见
    const isVisible = await iframe.isVisible();
    console.log(`   iframe 可见: ${isVisible}`);
    
    if (isVisible) {
      const bbox = await iframe.boundingBox();
      console.log(`   iframe 尺寸: ${Math.round(bbox.width)}x${Math.round(bbox.height)}`);
    }
  } else {
    console.log('   ⚠️  没有找到iframe，可能有连接错误');
    
    // 检查错误消息
    const errorCard = await page.locator('text=无法连接MinIO服务').isVisible().catch(() => false);
    if (errorCard) {
      console.log('   ❌ 显示连接错误');
    }
  }
  
  // 8. 截图最终状态
  console.log('\n7. 保存截图...');
  await page.screenshot({ 
    path: 'test-screenshots/object-storage-minio-console.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: object-storage-minio-console.png\n');
  
  // 9. 报告失败的请求
  if (failedRequests.length > 0) {
    console.log('失败的请求:');
    failedRequests.forEach((req, idx) => {
      console.log(`   ${idx + 1}. [${req.status}] ${req.url}`);
    });
    console.log('');
  }
  
  console.log('=== 测试完成 ===\n');
});
