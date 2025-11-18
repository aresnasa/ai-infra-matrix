const { test, expect } = require('@playwright/test');

test.describe('Nightingale Monitoring Page with Login', () => {
  const baseUrl = 'http://192.168.18.114:8080';
  const adminUsername = 'admin';
  const adminPassword = 'admin123';

  test('access /monitoring after login', async ({ page }) => {
    console.log('\n🔐 Testing /monitoring access with admin login...\n');
    
    // Step 1: Login first
    console.log('📍 Step 1: Logging in to main system...');
    await page.goto(`${baseUrl}/login`);
    await page.waitForTimeout(2000);
    
    // Fill login form
    const usernameInput = await page.locator('input[type="text"], input[name="username"], input[placeholder*="用户名"]').first();
    await usernameInput.fill(adminUsername);
    
    const passwordInput = await page.locator('input[type="password"]').first();
    await passwordInput.fill(adminPassword);
    
    const loginButton = await page.locator('button[type="submit"]').first();
    await loginButton.click();
    
    console.log('   ⏳ Waiting for login to complete...');
    await page.waitForTimeout(3000);
    
    const loginSuccess = !page.url().includes('/login');
    console.log(`   ${loginSuccess ? '✅' : '❌'} Login: ${loginSuccess ? 'Success' : 'Failed'}`);
    
    if (!loginSuccess) {
      await page.screenshot({ path: 'test-screenshots/login-failed.png', fullPage: true });
      console.log('   📸 Login failed screenshot saved');
      return;
    }
    
    // Step 2: Navigate to /monitoring
    console.log('\n📍 Step 2: Navigating to /monitoring...');
    await page.goto(`${baseUrl}/monitoring`);
    await page.waitForTimeout(3000);
    
    await page.screenshot({ path: 'test-screenshots/monitoring-after-login.png', fullPage: true });
    console.log('   📸 Screenshot saved: monitoring-after-login.png');
    
    const finalUrl = page.url();
    console.log(`   Current URL: ${finalUrl}`);
    
    // Step 3: Check if iframe is present
    console.log('\n📍 Step 3: Checking for Nightingale iframe...');
    
    const iframeSelector = 'iframe[title*="Nightingale"], iframe[src*="nightingale"]';
    const iframeCount = await page.locator(iframeSelector).count();
    console.log(`   Nightingale iframe count: ${iframeCount}`);
    
    if (iframeCount > 0) {
      const iframeSrc = await page.locator(iframeSelector).first().getAttribute('src');
      console.log(`   ✅ Iframe found! src: ${iframeSrc}`);
      
      // Wait for iframe to load
      await page.waitForTimeout(2000);
      
      // Try to access iframe content (if same-origin)
      try {
        const iframe = page.frameLocator(iframeSelector).first();
        const iframeBody = iframe.locator('body');
        const hasContent = await iframeBody.count() > 0;
        console.log(`   Iframe loaded: ${hasContent ? '✅ Yes' : '⚠️  Unknown (may be cross-origin)'}`);
      } catch (error) {
        console.log(`   Iframe content: ⚠️  Cannot access (${error.message})`);
      }
      
    } else {
      console.log('   ❌ No Nightingale iframe found!');
      
      // Check page content for clues
      const pageText = await page.locator('body').textContent();
      const hasMonitoring = pageText.includes('监控') || pageText.includes('Monitoring');
      const hasError = pageText.includes('错误') || pageText.includes('Error') || pageText.includes('无权限');
      
      console.log(`\n   Page analysis:`);
      console.log(`   - Contains "监控/Monitoring": ${hasMonitoring ? '✓' : '✗'}`);
      console.log(`   - Contains error message: ${hasError ? '⚠️  Yes' : '✗'}`);
      
      if (hasError) {
        console.log(`\n   ⚠️  There might be a permission or error issue`);
      }
    }
    
    // Step 4: Check console logs
    console.log('\n📍 Step 4: Checking browser console...');
    
    const consoleLogs = [];
    const consoleErrors = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (msg.type() === 'error') {
        consoleErrors.push(text);
      } else {
        consoleLogs.push(text);
      }
    });
    
    // Reload to capture console logs
    await page.reload();
    await page.waitForTimeout(3000);
    
    if (consoleErrors.length > 0) {
      console.log(`   ❌ Console errors (${consoleErrors.length}):`);
      consoleErrors.slice(0, 5).forEach(err => {
        console.log(`      - ${err.substring(0, 100)}`);
      });
    } else {
      console.log(`   ✅ No console errors`);
    }
    
    // Step 5: Check network requests
    console.log('\n📍 Step 5: Checking network requests...');
    
    const requests = [];
    page.on('request', request => {
      if (request.url().includes('nightingale') || request.url().includes('monitoring')) {
        requests.push({
          url: request.url(),
          method: request.method()
        });
      }
    });
    
    await page.reload();
    await page.waitForTimeout(2000);
    
    if (requests.length > 0) {
      console.log(`   Network requests to nightingale/monitoring:`);
      requests.slice(0, 5).forEach(req => {
        console.log(`      ${req.method} ${req.url}`);
      });
    } else {
      console.log(`   ⚠️  No requests to nightingale/monitoring found`);
    }
    
    await page.screenshot({ path: 'test-screenshots/monitoring-final.png', fullPage: true });
  });
});
