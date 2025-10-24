const { test, expect } = require('@playwright/test');

test.describe('Check iframe src attribute', () => {
  test('Verify iframe URL is correct', async ({ page }) => {
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

    // Navigate to monitoring page
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    // Get iframe element
    const iframe = await page.locator('iframe[title="Nightingale Monitoring"]');
    const src = await iframe.getAttribute('src');
    
    console.log(`\niframe src 属性: ${src}`);
    console.log(`\n是否包含 /targets: ${src.includes('/targets')}`);
    console.log(`是否包含 /metrics: ${src.includes('/metrics')}`);
    console.log(`是否以 /nightingale/ 结尾: ${src.endsWith('/nightingale/')}`);
  });
});
