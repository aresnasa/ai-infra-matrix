const { test, expect } = require('@playwright/test');

test.describe('Nightingale 监控 - 最终测试', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('/');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button:has-text("登录")');
    await page.waitForURL('**/dashboard');
  });

  test('MonitoringPage iframe 应该正确加载', async ({ page, context }) => {
    // 访问监控页面
    await page.goto('/monitoring');
    await page.waitForTimeout(2000);

    // 检查 iframe 是否存在
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframe).toBeVisible({ timeout: 10000 });
    
    // 检查 iframe src
    const src = await iframe.getAttribute('src');
    console.log('Iframe src:', src);
    expect(src).toContain('/nightingale/');
    
    // 检查 iframe 是否有 sandbox 属性（应该没有）
    const sandbox = await iframe.getAttribute('sandbox');
    console.log('Iframe sandbox:', sandbox);
    expect(sandbox).toBeNull();
    
    // 等待 iframe 加载完成
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ path: 'test-results/monitoring-page-final.png', fullPage: true });
    
    console.log('✅ MonitoringPage iframe 测试通过');
  });

  test('直接访问 /nightingale/ 应该返回内容', async ({ page, context }) => {
    // 先登录获取 cookies
    await page.goto('/monitoring');
    await page.waitForTimeout(2000);
    
    // 获取所有 cookies
    const cookies = await context.cookies();
    console.log('Cookies:', cookies.map(c => `${c.name}=${c.value.substring(0, 20)}...`));
    
    // 直接访问 /nightingale/
    const response = await page.goto('/nightingale/');
    
    console.log('Response status:', response.status());
    console.log('Response headers:', await response.allHeaders());
    
    // 应该返回 200 而不是 401
    expect(response.status()).toBe(200);
    
    // 截图
    await page.screenshot({ path: 'test-results/nightingale-direct-access.png', fullPage: true });
    
    console.log('✅ 直接访问 /nightingale/ 测试通过');
  });

  test('检查 MonitoringPage 是否有加载错误', async ({ page }) => {
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    await page.goto('/monitoring');
    await page.waitForTimeout(5000);
    
    console.log('Console errors:', errors);
    
    // 检查是否有 Loading 状态
    const loadingText = page.locator('text=正在加载监控系统');
    const isLoading = await loadingText.isVisible().catch(() => false);
    console.log('是否在加载中:', isLoading);
    
    // 检查是否有错误提示
    const errorAlert = page.locator('.ant-alert-error');
    const hasError = await errorAlert.isVisible().catch(() => false);
    console.log('是否有错误提示:', hasError);
    
    if (hasError) {
      const errorText = await errorAlert.textContent();
      console.log('错误内容:', errorText);
    }
    
    console.log('✅ 错误检查完成');
  });
});
