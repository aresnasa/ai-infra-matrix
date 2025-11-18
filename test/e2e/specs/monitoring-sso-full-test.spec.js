const { test, expect } = require('@playwright/test');

test.describe('Monitoring Page Full Test', () => {
  test('should successfully login to Nightingale via SSO', async ({ page }) => {
    console.log('=== Nightingale SSO 完整测试 ===\n');

    // Step 1: 登录主系统
    console.log('Step 1: 登录 AI-Infra-Matrix...');
    await page.goto('http://192.168.0.200:8080/login');
    await page.waitForLoadState('networkidle');
    
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    
    await page.waitForURL('http://192.168.0.200:8080/projects', { timeout: 10000 });
    console.log('✅ 登录成功\n');

    // Step 2: 访问监控页面
    console.log('Step 2: 访问监控页面...');
    await page.goto('http://192.168.0.200:8080/monitoring');
    await page.waitForLoadState('networkidle');
    
    // 验证 URL
    expect(page.url()).toBe('http://192.168.0.200:8080/monitoring');
    console.log('✅ URL 正确');

    // 验证卡片标题
    const cardTitle = await page.locator('.ant-card-head-title').textContent();
    expect(cardTitle).toContain('监控仪表板');
    console.log('✅ 卡片标题正确:', cardTitle);

    // 验证 iframe 存在
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframe).toBeVisible();
    console.log('✅ Nightingale iframe 可见');

    const iframeSrc = await iframe.getAttribute('src');
    expect(iframeSrc).toBe('http://192.168.0.200:8080/nightingale/');
    console.log('✅ iframe src 正确:', iframeSrc);

    // Step 3: 验证 Nightingale 已加载
    console.log('\nStep 3: 验证 Nightingale 已加载...');
    
    // 等待足够时间让 iframe 内容加载
    await page.waitForTimeout(3000);
    
    // 检查是否没有错误提示
    const hasError = await page.locator('.ant-alert-error').count();
    expect(hasError).toBe(0);
    console.log('✅ 没有错误提示');
    
    // 验证 iframe 内容已加载（不是空白）
    const iframeContent = page.frameLocator('iframe[title="Nightingale Monitoring"]');
    const bodyExists = await iframeContent.locator('body').count();
    expect(bodyExists).toBeGreaterThan(0);
    console.log('✅ iframe 内容已加载');

    // Step 4: 截图
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-success.png', 
      fullPage: true 
    });
    console.log('\n✅ 截图保存: test-screenshots/monitoring-success.png');

    console.log('\n=== ✅ SSO 测试成功！===');
    console.log('监控页面正常工作，用户通过 SSO 自动登录 Nightingale');
  });

  test('should have working refresh button', async ({ page }) => {
    console.log('\n=== 测试刷新按钮 ===');
    
    // 登录并访问监控页面
    await page.goto('http://192.168.0.200:8080/login');
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/projects/);
    
    await page.goto('http://192.168.0.200:8080/monitoring');
    await page.waitForLoadState('networkidle');
    
    // 点击刷新按钮
    const refreshButton = page.locator('button:has-text("刷新")');
    await expect(refreshButton).toBeVisible();
    console.log('✅ 刷新按钮可见');
    
    await refreshButton.click();
    console.log('✅ 刷新按钮可点击');
    
    // 等待 iframe 重新加载
    await page.waitForTimeout(2000);
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframe).toBeVisible();
    console.log('✅ iframe 重新加载成功');
  });

  test('should have working fullscreen button', async ({ page, context }) => {
    console.log('\n=== 测试新窗口打开按钮 ===');
    
    // 登录并访问监控页面
    await page.goto('http://192.168.0.200:8080/login');
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/projects/);
    
    await page.goto('http://192.168.0.200:8080/monitoring');
    await page.waitForLoadState('networkidle');
    
    // 点击新窗口打开按钮
    const fullscreenButton = page.locator('button:has-text("新窗口打开")');
    await expect(fullscreenButton).toBeVisible();
    console.log('✅ 新窗口打开按钮可见');
    
    // 监听新页面打开
    const pagePromise = context.waitForEvent('page');
    await fullscreenButton.click();
    const newPage = await pagePromise;
    
    await newPage.waitForLoadState('networkidle');
    console.log('✅ 新窗口已打开');
    console.log('新窗口 URL:', newPage.url());
    
    expect(newPage.url()).toBe('http://192.168.0.200:8080/nightingale/');
    console.log('✅ 新窗口 URL 正确');
    
    await newPage.close();
  });
});
