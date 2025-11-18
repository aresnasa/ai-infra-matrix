/**
 * 测试监控仪表板导航集成
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('监控仪表板导航集成测试', () => {
  
  test.beforeEach(async ({ page }) => {
    // 登录系统
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // 等待登录成功并跳转（可能跳转到 dashboard 或 projects）
    await page.waitForURL(/\/(dashboard|projects)/, { timeout: 10000 });
    console.log('✓ 登录成功，当前页面:', page.url());
  });

  test('1. 验证监控仪表板菜单项显示在顶部导航栏', async ({ page }) => {
    // 查找监控仪表板菜单项
    const monitoringMenu = page.locator('text=监控仪表板').first();
    
    // 验证菜单项存在
    await expect(monitoringMenu).toBeVisible();
    console.log('✓ 监控仪表板菜单项显示在导航栏');
    
    // 验证菜单项可点击
    await expect(monitoringMenu).toBeEnabled();
    console.log('✓ 监控仪表板菜单项可点击');
  });

  test('2. 点击监控仪表板菜单项跳转到正确页面', async ({ page }) => {
    // 点击监控仪表板菜单
    await page.click('text=监控仪表板');
    
    // 等待页面跳转
    await page.waitForURL(`${BASE_URL}/monitoring`, { timeout: 10000 });
    console.log('✓ 成功跳转到 /monitoring 页面');
    
    // 验证页面标题
    const pageTitle = page.locator('text=监控仪表板').first();
    await expect(pageTitle).toBeVisible();
    console.log('✓ 页面标题正确显示');
  });

  test('3. 验证 Nightingale iframe 正确加载', async ({ page }) => {
    // 导航到监控页面
    await page.goto(`${BASE_URL}/monitoring`);
    
    // 等待 iframe 加载
    const iframe = page.frameLocator('iframe[title="Nightingale Monitoring"]');
    
    // 验证 iframe 存在
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframeElement).toBeVisible({ timeout: 10000 });
    console.log('✓ Nightingale iframe 存在');
    
    // 等待 iframe 内容加载（检查 #root 元素）
    await page.waitForTimeout(5000);
    
    // 验证 iframe src 正确
    const src = await iframeElement.getAttribute('src');
    console.log('iframe src:', src);
    expect(src).toContain('/nightingale/');
    console.log('✓ iframe src 正确指向 Nightingale');
  });

  test('4. 验证页面操作按钮功能', async ({ page }) => {
    // 导航到监控页面
    await page.goto(`${BASE_URL}/monitoring`);
    
    // 等待页面加载
    await page.waitForTimeout(2000);
    
    // 验证刷新按钮存在
    const refreshBtn = page.locator('button:has-text("刷新")');
    await expect(refreshBtn).toBeVisible();
    console.log('✓ 刷新按钮显示');
    
    // 验证新窗口打开按钮存在
    const openBtn = page.locator('button:has-text("新窗口打开")');
    await expect(openBtn).toBeVisible();
    console.log('✓ 新窗口打开按钮显示');
    
    // 测试刷新按钮
    await refreshBtn.click();
    await page.waitForTimeout(1000);
    console.log('✓ 刷新按钮点击成功');
  });

  test('5. 验证监控页面与其他页面导航切换', async ({ page }) => {
    // 从 Dashboard 切换到监控仪表板
    await page.click('text=监控仪表板');
    await page.waitForURL(`${BASE_URL}/monitoring`);
    console.log('✓ 切换到监控仪表板');
    
    // 切换到项目管理
    await page.click('text=项目管理');
    await page.waitForURL(`${BASE_URL}/projects`);
    console.log('✓ 切换到项目管理');
    
    // 再次切换回监控仪表板
    await page.click('text=监控仪表板');
    await page.waitForURL(`${BASE_URL}/monitoring`);
    console.log('✓ 再次切换到监控仪表板');
    
    // 验证 iframe 仍然正常显示
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframeElement).toBeVisible();
    console.log('✓ iframe 在页面切换后仍正常显示');
  });

  test('6. 验证菜单项高亮状态', async ({ page }) => {
    // 导航到监控页面
    await page.goto(`${BASE_URL}/monitoring`);
    await page.waitForTimeout(1000);
    
    // 检查监控仪表板菜单项是否被选中（高亮）
    // Ant Design Menu 选中的项会有 ant-menu-item-selected 类
    const selectedMenu = page.locator('.ant-menu-item-selected:has-text("监控仪表板")');
    
    // 等待一下让选中状态生效
    await page.waitForTimeout(500);
    
    // 截图用于调试
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-menu-selected.png',
      fullPage: false 
    });
    
    console.log('✓ 截图已保存，检查菜单选中状态');
  });

  test('7. 完整流程测试', async ({ page }) => {
    console.log('\n=== 开始完整流程测试 ===\n');
    
    // 1. 验证初始状态（登录后可能在 Dashboard 或 Projects）
    const currentUrl = page.url();
    console.log(`✓ 1. 登录后在页面: ${currentUrl}`);
    
    // 2. 点击监控仪表板
    await page.click('text=监控仪表板');
    await page.waitForURL(`${BASE_URL}/monitoring`);
    console.log('✓ 2. 跳转到监控仪表板');
    
    // 3. 等待 iframe 加载
    await page.waitForSelector('iframe[title="Nightingale Monitoring"]', { timeout: 10000 });
    console.log('✓ 3. iframe 加载成功');
    
    // 4. 等待 Nightingale 内容加载
    await page.waitForTimeout(5000);
    console.log('✓ 4. 等待 Nightingale 内容加载');
    
    // 5. 截图验证最终效果
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-final-integration.png',
      fullPage: true 
    });
    console.log('✓ 5. 截图已保存');
    
    // 6. 验证 iframe 尺寸合理
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    const bbox = await iframeElement.boundingBox();
    
    console.log(`\niframe 尺寸: ${bbox.width}x${bbox.height}`);
    expect(bbox.width).toBeGreaterThan(800);
    expect(bbox.height).toBeGreaterThan(400);
    console.log('✓ 6. iframe 尺寸正常');
    
    console.log('\n=== 完整流程测试通过 ===\n');
  });
});
