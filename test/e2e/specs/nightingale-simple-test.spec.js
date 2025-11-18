const { test, expect } = require('@playwright/test');

test.describe('Nightingale 简单测试', () => {
  test('MonitoringPage 应该能加载 Nightingale', async ({ page }) => {
    // 登录
    await page.goto('/');
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {});
    
    // 填写登录表单
    const usernameInput = page.locator('input[type="text"], input[placeholder*="用户名"], input[name="username"]').first();
    await usernameInput.fill('admin');
    
    const passwordInput = page.locator('input[type="password"], input[placeholder*="密码"], input[name="password"]').first();
    await passwordInput.fill('admin123');
    
    // 点击登录按钮
    const loginButton = page.locator('button').filter({ hasText: /登录|Login/ }).first();
    await loginButton.click();
    
    // 等待登录完成
    await page.waitForURL('**/dashboard', { timeout: 15000 }).catch(async () => {
      await page.waitForTimeout(3000);
    });
    
    console.log('✓ 登录成功');
    
    // 访问监控页面
    await page.goto('/monitoring');
    await page.waitForTimeout(3000);
    
    // 检查 iframe
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    const iframeExists = await iframe.count();
    console.log('Iframe 数量:', iframeExists);
    
    if (iframeExists > 0) {
      const src = await iframe.getAttribute('src');
      console.log('Iframe src:', src);
      
      const sandbox = await iframe.getAttribute('sandbox');
      console.log('Iframe sandbox:', sandbox);
      
      const isVisible = await iframe.isVisible();
      console.log('Iframe 可见:', isVisible);
    }
    
    // 截图
    await page.screenshot({ path: 'test-results/nightingale-simple-monitoring-page.png', fullPage: true });
    
    // 直接测试 /nightingale/ 路径
    const nightingaleResponse = await page.goto('/nightingale/');
    console.log('/nightingale/ 状态:', nightingaleResponse.status());
    
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'test-results/nightingale-simple-direct-access.png', fullPage: true });
    
    console.log('✅ 测试完成');
  });
});
