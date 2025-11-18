/**
 * 监控仪表板完整集成测试
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('监控仪表板完整功能测试', async ({ page }) => {
  console.log('\n=== 监控仪表板完整功能测试 ===\n');
  
  // 1. 登录
  console.log('1. 登录系统...');
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|projects)/, { timeout: 10000 });
  console.log('   ✓ 登录成功\n');
  
  // 2. 验证导航菜单
  console.log('2. 验证导航菜单...');
  const monitoringMenu = page.locator('text=监控仪表板').first();
  await expect(monitoringMenu).toBeVisible();
  console.log('   ✓ 监控仪表板菜单项可见\n');
  
  // 3. 点击导航到监控页面
  console.log('3. 导航到监控页面...');
  await monitoringMenu.click();
  await page.waitForURL(`${BASE_URL}/monitoring`, { timeout: 10000 });
  console.log('   ✓ 成功跳转到 /monitoring\n');
  
  // 4. 验证页面标题
  console.log('4. 验证页面元素...');
  const pageTitle = page.locator('.ant-card-head-title:has-text("监控仪表板")');
  await expect(pageTitle).toBeVisible();
  console.log('   ✓ 页面标题显示正确');
  
  // 5. 验证操作按钮
  const refreshBtn = page.locator('button:has-text("刷新")');
  const openBtn = page.locator('button:has-text("新窗口打开")');
  await expect(refreshBtn).toBeVisible();
  await expect(openBtn).toBeVisible();
  console.log('   ✓ 刷新和新窗口按钮显示\n');
  
  // 6. 验证 iframe 加载
  console.log('5. 验证 Nightingale iframe...');
  const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
  await expect(iframe).toBeVisible({ timeout: 15000 });
  console.log('   ✓ iframe 元素存在');
  
  const src = await iframe.getAttribute('src');
  console.log(`   iframe src: ${src}`);
  expect(src).toContain('/nightingale/');
  console.log('   ✓ iframe src 正确\n');
  
  // 7. 等待 iframe 内容加载
  console.log('6. 等待 iframe 内容加载...');
  await page.waitForTimeout(5000);
  
  const bbox = await iframe.boundingBox();
  console.log(`   iframe 尺寸: ${Math.round(bbox.width)}x${Math.round(bbox.height)}`);
  expect(bbox.width).toBeGreaterThan(500);
  expect(bbox.height).toBeGreaterThan(300);
  console.log('   ✓ iframe 尺寸正常\n');
  
  // 8. 截图最终效果
  console.log('7. 保存截图...');
  await page.screenshot({ 
    path: 'test-screenshots/monitoring-complete.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: monitoring-complete.png\n');
  
  // 9. 测试页面切换
  console.log('8. 测试页面导航切换...');
  await page.click('text=项目管理');
  await page.waitForURL(`${BASE_URL}/projects`);
  console.log('   ✓ 切换到项目管理');
  
  await page.click('text=监控仪表板');
  await page.waitForURL(`${BASE_URL}/monitoring`);
  console.log('   ✓ 切换回监控仪表板');
  
  await expect(iframe).toBeVisible();
  console.log('   ✓ iframe 仍然正常显示\n');
  
  console.log('=== ✅ 所有测试通过！===\n');
});
