const { test, expect } = require('@playwright/test');

test.describe('Nightingale Iframe Fix Verification', () => {
  test('verify MonitoringPage shows Nightingale after auto-login', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔧 Testing MonitoringPage with auto-login fix...\n');

    // Intercept Nightingale login API
    let nightingaleLoginCalled = false;
    await page.route('**/api/n9e/auth/login', async (route) => {
      console.log('   🎯 Intercepted Nightingale login API call');
      nightingaleLoginCalled = true;
      await route.continue();
    });

    console.log('📍 Step 1: Login to main system');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForNavigation({ timeout: 10000 });
    console.log('   ✅ Main system login successful\n');

    console.log('📍 Step 2: Navigate to /monitoring');
    await page.goto(`${baseURL}/monitoring`);
    await page.waitForTimeout(3000); // Wait for component to mount

    console.log('\n📍 Step 3: Checking if Nightingale login was called');
    console.log(`   Nightingale login API called: ${nightingaleLoginCalled ? '✅ Yes' : '❌ No'}`);

    console.log('\n📍 Step 4: Waiting for iframe to appear...');
    try {
      await page.waitForSelector('iframe', { timeout: 10000 });
      const iframeCount = await page.locator('iframe').count();
      console.log(`   Iframe count: ${iframeCount}`);

      if (iframeCount > 0) {
        const iframeSrc = await page.locator('iframe').first().getAttribute('src');
        console.log(`   ✅ Iframe found with src: ${iframeSrc}`);

        // Check iframe content
        await page.waitForTimeout(5000); // Wait for iframe to load

        const iframe = page.frameLocator('iframe').first();
        const iframeBody = await iframe.locator('body').innerHTML();
        
        // Check if it's showing Nightingale content (not login page)
        const hasNightingaleTitle = iframeBody.includes('Nightingale') || 
                                    iframeBody.includes('夜莺') ||
                                    iframeBody.includes('监控');
        const hasLoginForm = iframeBody.includes('type="password"') && 
                            iframeBody.includes('用户名');

        console.log(`   Nightingale content detected: ${hasNightingaleTitle ? '✅ Yes' : '❌ No'}`);
        console.log(`   Login form detected: ${hasLoginForm ? '❌ Yes (BAD)' : '✅ No (GOOD)'}`);

        if (hasLoginForm) {
          console.log('\n   ⚠️  ISSUE: Iframe is still showing login page!');
          console.log('   This means auto-login did not work properly.');
        } else if (hasNightingaleTitle) {
          console.log('\n   🎉 SUCCESS: Iframe is showing Nightingale content!');
        } else {
          console.log('\n   ⚠️  UNKNOWN: Cannot determine iframe content');
          console.log('   Iframe body preview:', iframeBody.substring(0, 500));
        }
      } else {
        console.log('   ❌ No iframe found');
      }
    } catch (error) {
      console.log(`   ❌ Error waiting for iframe: ${error.message}`);
    }

    // Check console for errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    await page.waitForTimeout(2000);

    console.log('\n📍 Step 5: Console errors');
    if (consoleErrors.length === 0) {
      console.log('   ✅ No console errors');
    } else {
      console.log(`   ❌ Console errors (${consoleErrors.length}):`);
      consoleErrors.forEach((err, i) => {
        console.log(`      ${i + 1}. ${err}`);
      });
    }

    console.log('\n' + '='.repeat(60));
    console.log('SUMMARY:');
    console.log('='.repeat(60));
    console.log(`Auto-login API called: ${nightingaleLoginCalled ? '✅' : '❌'}`);
    console.log(`Iframe present: ${await page.locator('iframe').count() > 0 ? '✅' : '❌'}`);
    console.log('='.repeat(60) + '\n');
  });
});
