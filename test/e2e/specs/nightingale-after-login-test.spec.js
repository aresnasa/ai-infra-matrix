const { test } = require('@playwright/test');

test.describe('Nightingale Access After Login', () => {
  test('access /nightingale/ after logging into main system', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔍 Testing /nightingale/ access after main system login...\n');

    // Capture console
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Step 1: Login to main system
    console.log('📍 Step 1: Login to main system');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForNavigation({ timeout: 10000 });
    console.log(`   ✅ Login successful, redirected to: ${page.url()}\n`);

    // Step 2: Now try to access /nightingale/
    console.log('📍 Step 2: Navigate to /nightingale/ after login');
    await page.goto(`${baseURL}/nightingale/`, { waitUntil: 'networkidle', timeout: 10000 });
    const currentUrl = page.url();
    console.log(`   Current URL: ${currentUrl}`);
    
    await page.screenshot({ path: 'test-screenshots/nightingale-after-login.png' });
    console.log('   📸 Screenshot saved: nightingale-after-login.png\n');

    // Check page content
    console.log('📍 Step 3: Check page content');
    const pageTitle = await page.title();
    console.log(`   Page title: ${pageTitle}`);

    const hasPasswordInput = await page.locator('input[type="password"]').count();
    const hasNightingaleContent = await page.locator('body').textContent().then(text => 
      text.includes('Nightingale') || text.includes('夜莺') || text.includes('监控')
    );

    console.log(`   Password input (login page): ${hasPasswordInput > 0 ? '❌ Yes (still on login)' : '✅ No'}`);
    console.log(`   Nightingale content: ${hasNightingaleContent ? '✅ Yes' : '❌ No'}\n`);

    // Check if still redirected
    console.log('📍 Step 4: Check redirection');
    const isRedirected = currentUrl !== `${baseURL}/nightingale/`;
    console.log(`   Redirected: ${isRedirected ? '❌ Yes' : '✅ No'}`);
    if (isRedirected) {
      console.log(`   Redirect target: ${currentUrl}\n`);
    } else {
      console.log('   ✅ No redirection\n');
    }

    // Check console errors
    console.log('📍 Step 5: Console errors');
    if (consoleErrors.length === 0) {
      console.log('   ✅ No console errors\n');
    } else {
      console.log(`   ❌ Console errors (${consoleErrors.length}):`);
      consoleErrors.slice(0, 5).forEach((err, i) => {
        console.log(`      ${i + 1}. ${err}`);
      });
      console.log('');
    }

    console.log('='.repeat(80));
    console.log('SUMMARY');
    console.log('='.repeat(80));
    console.log(`Main system login: ✅`);
    console.log(`Can access /nightingale/: ${!isRedirected ? '✅' : '❌'}`);
    console.log(`Shows Nightingale content: ${hasNightingaleContent ? '✅' : '❌'}`);
    console.log('='.repeat(80) + '\n');
  });

  test('access /monitoring page with full flow', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔍 Testing /monitoring page full flow...\n');

    // Login first
    console.log('📍 Step 1: Login to main system');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForNavigation({ timeout: 10000 });
    console.log('   ✅ Login successful\n');

    // Navigate to monitoring
    console.log('📍 Step 2: Navigate to /monitoring');
    await page.goto(`${baseURL}/monitoring`);
    await page.waitForTimeout(3000);

    await page.screenshot({ path: 'test-screenshots/monitoring-full-flow.png' });
    console.log('   📸 Screenshot saved: monitoring-full-flow.png\n');

    // Check iframe
    console.log('📍 Step 3: Check iframe');
    const iframeCount = await page.locator('iframe').count();
    console.log(`   Iframe count: ${iframeCount}`);

    if (iframeCount > 0) {
      const iframeSrc = await page.locator('iframe').first().getAttribute('src');
      console.log(`   Iframe src: ${iframeSrc}`);

      // Check iframe content
      await page.waitForTimeout(3000);
      const iframe = page.frameLocator('iframe').first();
      
      const hasPasswordInput = await iframe.locator('input[type="password"]').count();
      console.log(`   Iframe shows login: ${hasPasswordInput > 0 ? '⚠️  Yes' : '✅ No'}`);

      if (hasPasswordInput > 0) {
        console.log('   ℹ️  Need to login in iframe\n');
      } else {
        console.log('   ✅ Iframe shows content directly\n');
      }
    } else {
      console.log('   ❌ No iframe found\n');
    }

    console.log('='.repeat(80));
    console.log('SUMMARY');
    console.log('='.repeat(80));
    console.log(`Iframe present: ${iframeCount > 0 ? '✅' : '❌'}`);
    console.log('='.repeat(80) + '\n');
  });
});
