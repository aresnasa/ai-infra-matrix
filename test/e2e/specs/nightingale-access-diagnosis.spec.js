const { test } = require('@playwright/test');

test.describe('Nightingale Access Diagnosis', () => {
  test('diagnose /monitoring page and iframe issues', async ({ page, context }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n🔍 Diagnosing /monitoring access issue...\n');

    // Track network requests
    const requests = [];
    page.on('request', req => {
      if (req.url().includes('n9e') || req.url().includes('nightingale')) {
        requests.push({ method: req.method(), url: req.url() });
      }
    });

    // Track responses
    const responses = [];
    page.on('response', async res => {
      if (res.url().includes('n9e') || res.url().includes('nightingale')) {
        responses.push({ 
          url: res.url(), 
          status: res.status(),
          statusText: res.statusText()
        });
      }
    });

    // Track console
    const consoleLogs = [];
    page.on('console', msg => {
      consoleLogs.push({ type: msg.type(), text: msg.text() });
    });

    console.log('📍 Step 1: Login to main system');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    try {
      await page.waitForNavigation({ timeout: 10000 });
      console.log('   ✅ Login successful\n');
    } catch (err) {
      console.log('   ❌ Login failed:', err.message);
      
      // Take screenshot
      await page.screenshot({ path: 'test-screenshots/login-failed.png' });
      console.log('   📸 Screenshot saved: login-failed.png\n');
      
      // Check if there's an error message
      const errorMsg = await page.locator('.ant-message-error, .ant-alert-error').textContent().catch(() => null);
      if (errorMsg) {
        console.log('   ❌ Error message:', errorMsg);
      }
      
      return; // Stop test if login fails
    }

    console.log('📍 Step 2: Navigate to /monitoring');
    await page.goto(`${baseURL}/monitoring`);
    await page.waitForTimeout(3000);

    const currentUrl = page.url();
    console.log(`   Current URL: ${currentUrl}`);

    // Take screenshot
    await page.screenshot({ path: 'test-screenshots/monitoring-page.png' });
    console.log('   📸 Screenshot saved: monitoring-page.png\n');

    console.log('📍 Step 3: Check page content');
    
    // Check for loading indicator
    const loadingCount = await page.locator('.ant-spin').count();
    console.log(`   Loading spinner count: ${loadingCount}`);

    // Check for error message
    const errorCount = await page.locator('.ant-alert-error').count();
    if (errorCount > 0) {
      const errorText = await page.locator('.ant-alert-error').textContent();
      console.log(`   ❌ Error alert: ${errorText}`);
    }

    // Check for iframe
    const iframeCount = await page.locator('iframe').count();
    console.log(`   Iframe count: ${iframeCount}`);

    if (iframeCount > 0) {
      const iframeSrc = await page.locator('iframe').first().getAttribute('src');
      console.log(`   Iframe src: ${iframeSrc}`);

      // Wait for iframe to load
      await page.waitForTimeout(5000);

      // Check iframe content
      try {
        const iframe = page.frameLocator('iframe').first();
        const iframeUrl = page.url(); // Can't directly get iframe URL
        
        // Check if iframe shows login page
        const hasPasswordInput = await iframe.locator('input[type="password"]').count();
        const hasUsernameInput = await iframe.locator('input[type="text"], input[name="username"]').count();
        
        if (hasPasswordInput > 0 && hasUsernameInput > 0) {
          console.log('   ⚠️  Iframe is showing LOGIN page (authentication issue)');
        } else {
          console.log('   ✅ Iframe is showing content (not login page)');
        }

        // Try to get iframe body text
        const bodyText = await iframe.locator('body').textContent({ timeout: 5000 }).catch(() => 'Cannot read iframe content');
        console.log(`   Iframe content preview: ${bodyText.substring(0, 200)}`);
      } catch (err) {
        console.log(`   ❌ Cannot access iframe: ${err.message}`);
      }
    } else {
      console.log('   ❌ No iframe found on page');
    }

    console.log('\n📍 Step 4: Network requests to Nightingale');
    if (requests.length === 0) {
      console.log('   ⚠️  No requests to Nightingale/n9e endpoints');
    } else {
      requests.forEach((req, i) => {
        console.log(`   ${i + 1}. [${req.method}] ${req.url}`);
      });
    }

    console.log('\n📍 Step 5: Network responses from Nightingale');
    if (responses.length === 0) {
      console.log('   ⚠️  No responses from Nightingale/n9e endpoints');
    } else {
      responses.forEach((res, i) => {
        const status = res.status >= 200 && res.status < 300 ? '✅' : '❌';
        console.log(`   ${status} ${i + 1}. ${res.status} ${res.statusText} - ${res.url}`);
      });
    }

    console.log('\n📍 Step 6: Console logs');
    const importantLogs = consoleLogs.filter(log => 
      log.text.includes('Nightingale') || 
      log.text.includes('n9e') ||
      log.text.includes('login') ||
      log.type === 'error'
    );
    
    if (importantLogs.length === 0) {
      console.log('   ✅ No relevant console logs');
    } else {
      importantLogs.forEach((log, i) => {
        const icon = log.type === 'error' ? '❌' : 'ℹ️';
        console.log(`   ${icon} [${log.type}] ${log.text}`);
      });
    }

    console.log('\n' + '='.repeat(80));
    console.log('DIAGNOSIS SUMMARY');
    console.log('='.repeat(80));
    console.log(`Iframe present: ${iframeCount > 0 ? '✅ YES' : '❌ NO'}`);
    console.log(`Nightingale API calls: ${requests.length > 0 ? `✅ YES (${requests.length})` : '❌ NO'}`);
    console.log(`Errors detected: ${errorCount > 0 ? `❌ YES` : '✅ NO'}`);
    console.log('='.repeat(80) + '\n');
  });
});
