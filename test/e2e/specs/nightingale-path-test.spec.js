const { test } = require('@playwright/test');

test.describe('Nightingale Path Test', () => {
  test('test /nightingale/ access and errors', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔍 Testing /nightingale/ access...\n');

    // Capture console errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Capture network responses
    const responses = [];
    page.on('response', async res => {
      responses.push({
        url: res.url(),
        status: res.status(),
        statusText: res.statusText()
      });
    });

    console.log('📍 Step 1: Navigate to /nightingale/');
    try {
      await page.goto(`${baseURL}/nightingale/`, { waitUntil: 'networkidle', timeout: 10000 });
      const currentUrl = page.url();
      console.log(`   Current URL: ${currentUrl}`);
      console.log(`   ✅ Page loaded\n`);
    } catch (err) {
      console.log(`   ❌ Failed to load: ${err.message}\n`);
    }

    // Take screenshot
    await page.screenshot({ path: 'test-screenshots/nightingale-direct-access.png' });
    console.log('   📸 Screenshot saved: nightingale-direct-access.png\n');

    // Check page content
    console.log('📍 Step 2: Check page content');
    const pageTitle = await page.title();
    console.log(`   Page title: ${pageTitle}`);

    const bodyText = await page.locator('body').textContent();
    const bodyPreview = bodyText.substring(0, 200).replace(/\s+/g, ' ');
    console.log(`   Body preview: ${bodyPreview}\n`);

    // Check for specific elements
    const hasPasswordInput = await page.locator('input[type="password"]').count();
    const hasLoginButton = await page.locator('button:has-text("登录"), button:has-text("登 录")').count();
    const hasErrorMessage = bodyText.includes('错误') || bodyText.includes('Error') || bodyText.includes('error');

    console.log(`   Password input: ${hasPasswordInput > 0 ? '✅ Found (Login page)' : '❌ Not found'}`);
    console.log(`   Login button: ${hasLoginButton > 0 ? '✅ Found' : '❌ Not found'}`);
    console.log(`   Error message: ${hasErrorMessage ? '⚠️  Yes' : '✅ No'}\n`);

    // Check network responses
    console.log('📍 Step 3: Network responses');
    const failedRequests = responses.filter(r => r.status >= 400);
    if (failedRequests.length === 0) {
      console.log('   ✅ All requests successful\n');
    } else {
      console.log(`   ❌ Failed requests (${failedRequests.length}):`);
      failedRequests.forEach((req, i) => {
        console.log(`      ${i + 1}. [${req.status}] ${req.url}`);
      });
      console.log('');
    }

    // Check console errors
    console.log('📍 Step 4: Console errors');
    if (consoleErrors.length === 0) {
      console.log('   ✅ No console errors\n');
    } else {
      console.log(`   ❌ Console errors (${consoleErrors.length}):`);
      consoleErrors.slice(0, 10).forEach((err, i) => {
        console.log(`      ${i + 1}. ${err}`);
      });
      console.log('');
    }

    console.log('='.repeat(80));
    console.log('SUMMARY');
    console.log('='.repeat(80));
    console.log(`Page loaded: ${page.url() === `${baseURL}/nightingale/` ? '✅' : '⚠️  Redirected'}`);
    console.log(`Shows login page: ${hasPasswordInput > 0 ? '✅' : '❌'}`);
    console.log(`Has errors: ${hasErrorMessage || consoleErrors.length > 0 ? '❌' : '✅'}`);
    console.log(`Failed requests: ${failedRequests.length}`);
    console.log('='.repeat(80) + '\n');
  });
});
