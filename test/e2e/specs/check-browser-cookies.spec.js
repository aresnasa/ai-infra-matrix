const { test, expect } = require('@playwright/test');

test.describe('Check browser cookies', () => {
  test('Verify cookies are set after login', async ({ page, context }) => {
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

    // Check cookies
    const cookies = await context.cookies('http://192.168.0.200:8080');
    console.log('\n=== 登录后的 Cookies ===\n');
    cookies.forEach(cookie => {
      console.log(`${cookie.name}: ${cookie.value.substring(0, 50)}...`);
      console.log(`  Domain: ${cookie.domain}`);
      console.log(`  Path: ${cookie.path}`);
      console.log(`  HttpOnly: ${cookie.httpOnly}`);
      console.log('');
    });

    // Navigate to Nightingale and check cookies again
    await page.goto('http://192.168.0.200:8080/nightingale/dashboards', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    const cookiesAfter = await context.cookies('http://192.168.0.200:8080');
    console.log(`\n导航到 Nightingale 后的 Cookies (${cookiesAfter.length}):\n`);
    cookiesAfter.forEach(cookie => {
      console.log(`  - ${cookie.name}`);
    });

    // Check if token cookie exists
    const tokenCookie = cookies.find(c => c.name === 'token');
    console.log(`\ntoken Cookie: ${tokenCookie ? '✓ 存在' : '✗ 不存在'}`);
    if (tokenCookie) {
      console.log(`  值: ${tokenCookie.value.substring(0, 100)}...`);
    }
  });
});
