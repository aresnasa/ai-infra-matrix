const { test } = require('@playwright/test');

test.describe('Nightingale 401 Error Debug', () => {
  test('capture all network requests and find 401 errors', async ({ page }) => {
    console.log('\n🔍 Debugging 401 error in /monitoring...\n');

    // Track all network requests
    const requests = [];
    page.on('request', request => {
      requests.push({
        url: request.url(),
        method: request.method()
      });
    });

    // Track all network responses
    const failedRequests = [];
    page.on('response', async response => {
      if (response.status() === 401) {
        failedRequests.push({
          url: response.url(),
          status: response.status(),
          statusText: response.statusText(),
          headers: response.headers()
        });
      }
    });

    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('📍 Step 1: Logging in...');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    // Wait for navigation after login (could be /projects or /dashboard)
    await page.waitForNavigation({ timeout: 10000 });
    console.log('   ✅ Login successful\n');

    console.log('📍 Step 2: Navigating to /monitoring...');
    await page.goto(`${baseURL}/monitoring`);
    
    // Wait for iframe to load
    await page.waitForSelector('iframe', { timeout: 10000 });
    
    // Wait a bit more for all resources to load
    await page.waitForTimeout(5000);

    console.log('\n📊 All requests made:');
    requests.forEach((req, i) => {
      console.log(`   ${i + 1}. [${req.method}] ${req.url}`);
    });

    console.log('\n❌ Failed requests (401):');
    if (failedRequests.length === 0) {
      console.log('   ✅ No 401 errors found!');
    } else {
      failedRequests.forEach((req, i) => {
        console.log(`   ${i + 1}. ${req.url}`);
        console.log(`      Status: ${req.status} ${req.statusText}`);
        console.log(`      Headers:`, JSON.stringify(req.headers, null, 2));
      });
    }

    // Check console errors with details
    const consoleLogs = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleLogs.push(msg.text());
      }
    });

    await page.waitForTimeout(2000);

    console.log('\n📝 Console errors:');
    if (consoleLogs.length === 0) {
      console.log('   ✅ No console errors!');
    } else {
      consoleLogs.forEach((log, i) => {
        console.log(`   ${i + 1}. ${log}`);
      });
    }

    // Check iframe content
    console.log('\n📍 Step 3: Checking iframe content...');
    const iframe = page.frameLocator('iframe').first();
    
    try {
      const iframeBody = await iframe.locator('body').innerText({ timeout: 5000 });
      console.log('   Iframe body content preview:', iframeBody.substring(0, 200));
      
      // Check if login page is shown in iframe
      const hasLoginForm = await iframe.locator('input[type="password"]').count();
      if (hasLoginForm > 0) {
        console.log('   ⚠️  Iframe is showing LOGIN page (requires authentication)');
      } else {
        console.log('   ✅ Iframe is showing Nightingale content');
      }
    } catch (error) {
      console.log('   ❌ Cannot access iframe content:', error.message);
    }
  });
});
