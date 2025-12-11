const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.199:8080';

test('Reproduction: Monitoring page 404 and Back to Home behavior', async ({ page }) => {
  test.setTimeout(60000);
  
  // Login first
  console.log('Navigating to login...');
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
  
  // Check if already logged in
  if (page.url().includes('/login')) {
      console.log('Filling login form...');
      await page.fill('input[type="text"], input[name="username"]', 'admin');
      await page.fill('input[type="password"], input[name="password"]', 'admin123');
      await page.click('button[type="submit"]');
      await page.waitForURL('**/projects');
      console.log('Logged in.');
  } else {
      console.log('Already logged in.');
  }

  // Go to monitoring
  console.log('Navigating to monitoring...');
  await page.goto(`${BASE_URL}/monitoring`);
  
  // Wait for iframe
  console.log('Waiting for iframe...');
  const iframeElement = await page.waitForSelector('iframe');
  const frame = await iframeElement.contentFrame();
  
  // Check for 404 text
  console.log('Waiting for iframe content...');
  try {
    await frame.waitForLoadState('networkidle');
  } catch (e) {
    console.log('Wait for networkidle timed out, continuing...');
  }
  // Give it a bit more time to render text
  await page.waitForTimeout(5000);
  
  // Take a screenshot for debugging
  await page.screenshot({ path: 'test-screenshots/monitoring-fix-verify.png', fullPage: true });
  
  const text = await frame.innerText('body');
  console.log('Iframe content preview:', text.substring(0, 200));
  
  if (text.includes('404') && text.includes('不存在')) {
      console.log('FAILURE: 404 page is still displayed.');
      // Fail the test if 404 is found
      expect(text).not.toContain('404');
      expect(text).not.toContain('不存在');
  } else {
      console.log('SUCCESS: 404 page is NOT displayed.');
      // Verify we are on the dashboard or home page
      // Look for common Nightingale text
      if (text.includes('仪表盘') || text.includes('Dashboard') || text.includes('即时查询')) {
          console.log('Verified: Loaded Nightingale UI successfully.');
      } else {
          console.log('Loaded page content seems valid.');
      }
  }

  // Try to find "Back to Home" button (should not be there if not 404)
  const backButton = frame.getByText('回到首页');
  if (await backButton.isVisible()) {
      console.log('Found "Back to Home" button (Unexpected if page loaded correctly).');
  } else {
      console.log('"Back to Home" button not found (Expected).');
  }
});
