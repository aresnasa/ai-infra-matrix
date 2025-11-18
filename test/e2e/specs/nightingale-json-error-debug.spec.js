const { test } = require('@playwright/test');

test.describe('Nightingale JSON Error Debug', () => {
  test('capture the exact JSON parse error', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔍 Debugging JSON parse error in /monitoring...\n');

    // Track all console messages with details
    const consoleMessages = [];
    page.on('console', msg => {
      const text = msg.text();
      const type = msg.type();
      consoleMessages.push({ type, text });
      
      // Print error messages immediately
      if (type === 'error' && text.includes('JSON')) {
        console.log(`❌ [CONSOLE ERROR] ${text}`);
      }
    });

    // Track all failed network requests
    const failedRequests = [];
    page.on('requestfailed', request => {
      failedRequests.push({
        url: request.url(),
        method: request.method(),
        failure: request.failure()
      });
    });

    // Track all responses
    const responses = [];
    page.on('response', async response => {
      const url = response.url();
      const status = response.status();
      const contentType = response.headers()['content-type'] || '';
      
      // Track all Nightingale API responses
      if (url.includes('/api/n9e/') || url.includes('/nightingale/')) {
        responses.push({ url, status, contentType });
        
        // If it's an API call expecting JSON but got HTML
        if (url.includes('/api/n9e/') && contentType.includes('text/html')) {
          console.log(`\n⚠️  API返回HTML而非JSON: ${url}`);
          console.log(`   Status: ${status}`);
          console.log(`   Content-Type: ${contentType}`);
          
          try {
            const body = await response.text();
            console.log(`   Body preview: ${body.substring(0, 200)}`);
          } catch (err) {
            console.log(`   Cannot read body: ${err.message}`);
          }
        }
      }
    });

    // Step 1: Login
    console.log('📍 Step 1: Login to main system');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForNavigation({ timeout: 10000 });
    console.log('   ✅ Login successful\n');

    // Step 2: Navigate to monitoring
    console.log('📍 Step 2: Navigate to /monitoring');
    await page.goto(`${baseURL}/monitoring`);
    
    // Wait for potential errors to occur
    await page.waitForTimeout(10000);

    // Step 3: Analyze console errors
    console.log('\n📍 Step 3: Analyze console errors');
    const jsonErrors = consoleMessages.filter(msg => 
      msg.type === 'error' && (msg.text.includes('JSON') || msg.text.includes('Unexpected token'))
    );

    if (jsonErrors.length > 0) {
      console.log(`   Found ${jsonErrors.length} JSON-related errors:`);
      jsonErrors.forEach((err, i) => {
        console.log(`   ${i + 1}. ${err.text}`);
      });
    } else {
      console.log('   ✅ No JSON parse errors found');
    }

    // Step 4: Check failed requests
    console.log('\n📍 Step 4: Failed network requests');
    if (failedRequests.length === 0) {
      console.log('   ✅ No failed requests');
    } else {
      failedRequests.forEach((req, i) => {
        console.log(`   ${i + 1}. ${req.method} ${req.url}`);
        console.log(`      Failure: ${req.failure?.errorText || 'Unknown'}`);
      });
    }

    // Step 5: Check responses with wrong content-type
    console.log('\n📍 Step 5: API responses analysis');
    const apiResponses = responses.filter(r => r.url.includes('/api/n9e/'));
    const htmlApiResponses = apiResponses.filter(r => r.contentType.includes('text/html'));
    
    if (htmlApiResponses.length > 0) {
      console.log(`   ⚠️  ${htmlApiResponses.length} API(s) returned HTML instead of JSON:`);
      htmlApiResponses.forEach((res, i) => {
        console.log(`   ${i + 1}. ${res.url}`);
        console.log(`      Status: ${res.status}, Content-Type: ${res.contentType}`);
      });
    } else {
      console.log('   ✅ All APIs returned correct content type');
    }

    // Take screenshot
    await page.screenshot({ path: 'test-screenshots/json-error-debug.png' });
    console.log('\n📸 Screenshot saved: json-error-debug.png');

    // Summary
    console.log('\n' + '='.repeat(80));
    console.log('SUMMARY');
    console.log('='.repeat(80));
    console.log(`JSON errors: ${jsonErrors.length > 0 ? '❌ YES' : '✅ NO'}`);
    console.log(`Failed requests: ${failedRequests.length}`);
    console.log(`APIs returning HTML: ${htmlApiResponses.length}`);
    console.log('='.repeat(80) + '\n');
  });
});
