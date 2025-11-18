const { test, expect } = require('@playwright/test');

test.describe('Verify Nightingale datasource fix', () => {
  test('Check that datasource is configured and no undefined errors', async ({ page }) => {
    const apiErrors = [];

    // Monitor for API errors
    page.on('response', async response => {
      const url = response.url();
      if (url.includes('/api/n9e/') && response.status() >= 400) {
        try {
          const body = await response.text().catch(() => 'Could not read body');
          apiErrors.push({
            url,
            status: response.status(),
            body: body.substring(0, 200)
          });
        } catch (e) {
          apiErrors.push({
            url,
            status: response.status(),
            error: e.message
          });
        }
      }
    });

    // Monitor console for undefined errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error' && msg.text().includes('undefined')) {
        consoleErrors.push(msg.text());
      }
    });

    console.log('\n=== 验证数据源配置和 undefined 错误修复 ===\n');

    // Login
    await page.goto('http://192.168.0.200:8080/login', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });
    await page.locator('input[type="text"], input[name="username"]').first().fill('admin');
    await page.locator('input[type="password"], input[name="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').click();
    
    await page.waitForURL(/\/(projects|monitoring)/, { timeout: 10000 });
    console.log('✓ 登录成功');

    // Navigate to monitoring page
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });
    console.log('✓ 导航到监控页面');

    // Wait for iframe to load
    await page.waitForSelector('iframe[title="Nightingale Monitoring"]', { timeout: 10000 });
    console.log('✓ Nightingale iframe 已加载');

    // Wait for network activity to settle
    await page.waitForTimeout(5000);

    // Check for undefined errors in API URLs
    const undefinedUrlErrors = apiErrors.filter(err => 
      err.url.includes('/undefined/') || err.url.includes('undefined')
    );

    console.log(`\nAPI 错误检查:`);
    console.log(`  - 总 API 错误数: ${apiErrors.length}`);
    console.log(`  - 包含 undefined 的错误: ${undefinedUrlErrors.length}`);

    if (undefinedUrlErrors.length > 0) {
      console.log(`\n❌ 发现 undefined 错误:`);
      undefinedUrlErrors.forEach(err => {
        console.log(`  ${err.status} - ${err.url}`);
        if (err.body) {
          console.log(`  响应: ${err.body}`);
        }
      });
    }

    console.log(`\n控制台 undefined 错误: ${consoleErrors.length}`);
    if (consoleErrors.length > 0) {
      consoleErrors.forEach((err, i) => {
        console.log(`  ${i+1}. ${err.substring(0, 200)}`);
      });
    }

    // Take screenshot
    await page.screenshot({ 
      path: 'test-screenshots/datasource-fix-verification.png',
      fullPage: false
    });
    console.log(`\n截图已保存: test-screenshots/datasource-fix-verification.png`);

    // Final assertions
    console.log(`\n最终结果: ${undefinedUrlErrors.length === 0 ? '✅ 修复成功' : '❌ 仍有问题'}`);
    expect(undefinedUrlErrors.length).toBe(0);
  });
});
