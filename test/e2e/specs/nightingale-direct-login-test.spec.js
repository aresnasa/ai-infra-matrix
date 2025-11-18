const { test } = require('@playwright/test');

test.describe('Nightingale Direct Login Test', () => {
  test('test login directly in Nightingale UI', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔐 Testing direct Nightingale login...\n');

    // Navigate directly to Nightingale
    console.log('📍 Step 1: Navigate to Nightingale');
    await page.goto(`${baseURL}/nightingale/`);
    await page.waitForTimeout(3000);

    const currentUrl = page.url();
    console.log(`   Current URL: ${currentUrl}`);

    // Take screenshot
    await page.screenshot({ path: 'test-screenshots/nightingale-direct.png' });
    console.log('   📸 Screenshot saved: nightingale-direct.png\n');

    // Check if it shows login form
    console.log('📍 Step 2: Check for login form');
    const hasUsernameInput = await page.locator('input[placeholder*="用户名"], input[name="username"]').count();
    const hasPasswordInput = await page.locator('input[type="password"]').count();
    
    console.log(`   Username input: ${hasUsernameInput > 0 ? '✅ Found' : '❌ Not found'}`);
    console.log(`   Password input: ${hasPasswordInput > 0 ? '✅ Found' : '❌ Not found'}`);

    if (hasUsernameInput > 0 && hasPasswordInput > 0) {
      console.log('\n📍 Step 3: Try to login');
      
      // Fill login form
      try {
        await page.fill('input[placeholder*="用户名"], input[name="username"]', 'admin');
        await page.fill('input[type="password"]', 'admin123');
        console.log('   ✅ Filled username and password');

        // Find and click login button
        const loginButton = page.locator('button:has-text("登录"), button:has-text("登 录"), button[type="submit"]').first();
        await loginButton.click();
        console.log('   ✅ Clicked login button');

        // Wait for navigation or response
        await page.waitForTimeout(3000);

        const newUrl = page.url();
        console.log(`   New URL: ${newUrl}`);

        // Check if login was successful
        const stillHasLoginForm = await page.locator('input[type="password"]').count();
        if (stillHasLoginForm > 0) {
          console.log('   ❌ Still on login page - login failed');
          
          // Check for error message
          const errorMsg = await page.locator('.ant-message-error, .ant-alert-error, [class*="error"]').textContent().catch(() => '');
          if (errorMsg) {
            console.log(`   Error message: ${errorMsg}`);
          }

          // Take screenshot of error
          await page.screenshot({ path: 'test-screenshots/nightingale-login-failed.png' });
          console.log('   📸 Screenshot saved: nightingale-login-failed.png');
        } else {
          console.log('   ✅ Login successful - no longer on login page');
          
          // Take screenshot of success
          await page.screenshot({ path: 'test-screenshots/nightingale-login-success.png' });
          console.log('   📸 Screenshot saved: nightingale-login-success.png');
        }
      } catch (err) {
        console.log(`   ❌ Error during login: ${err.message}`);
      }
    } else {
      console.log('   ⚠️  Not a login page or different UI');
    }
  });
});
