const { test, expect } = require('@playwright/test');

test.describe('Check Nightingale API requests headers', () => {
  test('Inspect request headers to Nightingale API', async ({ page }) => {
    const apiRequests = [];

    // Monitor all requests
    page.on('request', request => {
      const url = request.url();
      if (url.includes('/api/n9e/') || url.includes('nightingale')) {
        apiRequests.push({
          url,
          method: request.method(),
          headers: request.headers()
        });
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

    console.log('\n=== 检查 Nightingale API 请求的 Headers ===\n');

    // Clear previous requests
    apiRequests.length = 0;

    // Navigate to Nightingale
    await page.goto('http://192.168.0.200:8080/nightingale/dashboards', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    // Report API requests
    console.log(`\n捕获到 ${apiRequests.length} 个 API 请求:\n`);
    apiRequests.forEach((req, i) => {
      console.log(`${i+1}. ${req.method} ${req.url}`);
      console.log(`   Cookie: ${req.headers.cookie ? '✓ 有' : '✗ 无'}`);
      console.log(`   Authorization: ${req.headers.authorization ? '✓ 有' : '✗ 无'}`);
      console.log(`   X-User-Name: ${req.headers['x-user-name'] || '✗ 无'}`);
      console.log('');
    });
  });
});
