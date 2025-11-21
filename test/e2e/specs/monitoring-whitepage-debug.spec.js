const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.199:8080';

test('Debug: Monitoring page white screen analysis', async ({ page }) => {
  test.setTimeout(60000);

  // Capture console logs
  page.on('console', msg => console.log(`BROWSER CONSOLE: ${msg.type()}: ${msg.text()}`));
  
  // Capture page errors
  page.on('pageerror', exception => console.log(`BROWSER PAGE ERROR: ${exception}`));

  // Capture network failures
  page.on('requestfailed', request => {
    console.log(`NETWORK FAIL: ${request.url()} - ${request.failure().errorText}`);
  });

  // Login
  console.log('Navigating to login...');
  await page.goto(`${BASE_URL}/login`);
  
  // Handle login if redirected to login page
  if (page.url().includes('/login')) {
      const usernameInput = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
      await usernameInput.fill('admin');
      
      const passwordInput = await page.locator('input[type="password"]').first();
      await passwordInput.fill('admin123');
      
      const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
      await loginButton.click();
      
      await page.waitForURL('**/projects');
  }

  // Go to monitoring
  console.log('Navigating to monitoring...');
  await page.goto(`${BASE_URL}/monitoring`);
  
  // Wait a bit
  await page.waitForTimeout(5000);

  // Check iframe
  const iframeElement = await page.locator('iframe').first();
  const iframeCount = await page.locator('iframe').count();
  console.log(`Found ${iframeCount} iframes.`);

  if (iframeCount > 0) {
      const src = await iframeElement.getAttribute('src');
      console.log(`Iframe SRC: ${src}`);
      
      // Try to access the iframe src directly to see if it loads
      console.log(`Attempting to fetch iframe URL directly: ${src}`);
      try {
        const response = await page.request.get(src);
        console.log(`Direct fetch status: ${response.status()}`);
      } catch (e) {
        console.log(`Direct fetch failed: ${e.message}`);
      }
  }

  // Screenshot
  await page.screenshot({ path: 'test-screenshots/monitoring-whitepage-debug.png', fullPage: true });
});
