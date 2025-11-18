const { test, expect } = require('@playwright/test');

test.describe('Check Nightingale network requests and errors', () => {
  test('Monitor network and console for issues', async ({ page }) => {
    const failedRequests = [];
    const consoleErrors = [];

    // Monitor network
    page.on('response', response => {
      if (response.status() >= 400) {
        failedRequests.push({
          url: response.url(),
          status: response.status(),
          statusText: response.statusText()
        });
      }
    });

    // Monitor console
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Login first
    await page.goto('http://192.168.0.200:8080/login', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });
    await page.locator('input[type="text"], input[name="username"]').first().fill('admin');
    await page.locator('input[type="password"], input[name="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').click();
    
    await page.waitForURL(/\/(projects|monitoring)/, { timeout: 10000 });

    console.log('\n=== 监控 Nightingale 网络请求和控制台错误 ===\n');

    // Navigate to Nightingale
    await page.goto('http://192.168.0.200:8080/nightingale/dashboards', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    // Report findings
    console.log(`\n失败的请求 (${failedRequests.length}):`);
    failedRequests.forEach(req => {
      console.log(`  ${req.status} - ${req.url}`);
    });

    console.log(`\n控制台错误 (${consoleErrors.length}):`);
    consoleErrors.forEach((err, i) => {
      console.log(`  ${i+1}. ${err.substring(0, 200)}`);
    });

    // Check if user info is loaded
    const bodyText = await page.locator('body').innerText();
    console.log(`\n页面是否包含用户信息:`);
    console.log(`  - "admin": ${bodyText.includes('admin')}`);
    console.log(`  - "logout": ${bodyText.toLowerCase().includes('logout')}`);
    console.log(`  - "退出": ${bodyText.includes('退出')}`);
  });
});
