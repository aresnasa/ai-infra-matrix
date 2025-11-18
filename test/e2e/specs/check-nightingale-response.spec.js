const { test, expect } = require('@playwright/test');

test.describe('Check Nightingale API response details', () => {
  test('Capture actual API responses', async ({ page }) => {
    const apiResponses = [];

    // Monitor responses
    page.on('response', async response => {
      const url = response.url();
      if (url.includes('/api/n9e/')) {
        try {
          const body = await response.text().catch(() => 'Could not read body');
          apiResponses.push({
            url,
            status: response.status(),
            statusText: response.statusText(),
            headers: response.headers(),
            body: body.substring(0, 500)
          });
        } catch (e) {
          apiResponses.push({
            url,
            status: response.status(),
            error: e.message
          });
        }
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

    console.log('\n=== 检查 Nightingale API 响应详情 ===\n');

    // Navigate to Nightingale
    await page.goto('http://192.168.0.200:8080/nightingale/dashboards', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    // Report API responses
    console.log(`\n捕获到 ${apiResponses.length} 个 API 响应:\n`);
    apiResponses.slice(0, 5).forEach((res, i) => {
      console.log(`${i+1}. ${res.status} - ${res.url}`);
      if (res.body) {
        console.log(`   响应体: ${res.body}`);
      }
      if (res.error) {
        console.log(`   错误: ${res.error}`);
      }
      console.log('');
    });
  });
});
