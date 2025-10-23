/**
 * 简单测试监控仪表板集成
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('监控仪表板集成验证', async ({ page }) => {
  console.log('\n=== 开始监控仪表板集成测试 ===\n');
  
  // 1. 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|projects)/, { timeout: 10000 });
  console.log('✓ 登录成功');
  
  // 2. 等待页面加载完成
  await page.waitForTimeout(2000);
  
  // 3. 截图当前页面，查看导航栏
  await page.screenshot({ 
    path: 'test-screenshots/monitoring-navbar.png',
    fullPage: false 
  });
  console.log('✓ 截图已保存: monitoring-navbar.png');
  
  // 4. 打印所有导航菜单项
  const menuItems = await page.locator('.ant-menu-item, .ant-menu-submenu').allTextContents();
  console.log('\n导航菜单项:');
  menuItems.forEach((item, idx) => {
    console.log(`  ${idx + 1}. ${item.trim()}`);
  });
  
  // 5. 尝试查找监控相关的菜单项
  const hasMonitoring = menuItems.some(item => 
    item.includes('监控') || item.includes('Monitoring') || item.includes('Nightingale')
  );
  
  if (hasMonitoring) {
    console.log('\n✓ 找到监控相关菜单项');
    
    // 尝试点击导航到监控页面
    const monitoringLink = page.locator('text=/监控|Monitoring|Nightingale/i').first();
    if (await monitoringLink.isVisible()) {
      await monitoringLink.click();
      await page.waitForTimeout(2000);
      
      console.log('✓ 点击监控菜单项成功');
      console.log('当前 URL:', page.url());
      
      // 截图监控页面
      await page.screenshot({ 
        path: 'test-screenshots/monitoring-page.png',
        fullPage: true 
      });
      console.log('✓ 监控页面截图已保存');
    }
  } else {
    console.log('\n⚠️  未找到监控菜单项，请检查前端配置');
    
    // 直接导航到监控页面测试
    await page.goto(`${BASE_URL}/monitoring`);
    await page.waitForTimeout(2000);
    
    console.log('直接访问 /monitoring 页面');
    console.log('当前 URL:', page.url());
    
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-direct.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存');
  }
  
  console.log('\n=== 测试完成 ===\n');
});
