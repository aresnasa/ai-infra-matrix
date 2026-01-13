/**
 * Test test1 user login and component permissions
 * Run with: BASE_URL=https://192.168.216.57:8443 npx playwright test test/e2e/specs/test1-user-permissions.spec.js --config=test/e2e/playwright.config.js
 */

const { chromium } = require('playwright');

// Test configuration
const TEST_USER = {
  username: 'test1',
  password: 'test123',
};

const BASE_URL = process.env.BASE_URL || 'https://192.168.216.57:8443';

async function runTest() {
  const browser = await chromium.launch({ 
    headless: true,
  });
  
  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
  });
  
  const page = await context.newPage();
  
  try {
    console.log('=== Test1 User Permissions Test ===');
    console.log(`Base URL: ${BASE_URL}`);
    
    // Navigate to login page
    console.log('\n1. Navigating to login page...');
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    // Take screenshot
    await page.screenshot({ path: 'test-screenshots/test1-01-login-page.png' });
    console.log('Screenshot saved: test1-01-login-page.png');
    
    // Fill login form
    console.log('\n2. Filling login form...');
    await page.fill('input[id="username"], input[name="username"], input[placeholder*="用户名"], input[placeholder*="username"]', TEST_USER.username);
    await page.fill('input[type="password"]', TEST_USER.password);
    
    await page.screenshot({ path: 'test-screenshots/test1-02-login-filled.png' });
    
    // Click login button
    console.log('\n3. Clicking login button...');
    await page.click('button[type="submit"], button:has-text("登录"), button:has-text("Login")');
    
    // Wait for response
    await page.waitForTimeout(3000);
    
    await page.screenshot({ path: 'test-screenshots/test1-03-after-login.png' });
    
    const currentUrl = page.url();
    console.log(`Current URL after login: ${currentUrl}`);
    
    // Check for errors
    const errorMessage = await page.$('.ant-message-error, .ant-alert-error');
    if (errorMessage) {
      const errorText = await errorMessage.textContent();
      console.log(`ERROR: Login failed - ${errorText}`);
      await page.screenshot({ path: 'test-screenshots/test1-04-login-error.png' });
    }
    
    if (!currentUrl.includes('/login')) {
      console.log('\n4. Login successful! Checking permissions...');
      
      // Get user info from localStorage/cookies
      const authInfo = await page.evaluate(() => {
        return {
          token: localStorage.getItem('token'),
          user: localStorage.getItem('user'),
          cookies: document.cookie
        };
      });
      console.log('Auth info:', JSON.stringify(authInfo, null, 2).substring(0, 500));
      
      // Check menu visibility
      console.log('\n5. Checking menu visibility...');
      await page.screenshot({ path: 'test-screenshots/test1-05-dashboard.png', fullPage: true });
      
      // Get all visible menu items
      const menuItems = await page.$$eval('.ant-menu-item, .ant-menu-submenu-title', items => 
        items.map(item => ({
          text: item.textContent?.trim(),
          visible: item.offsetParent !== null
        }))
      );
      console.log('Menu items found:', JSON.stringify(menuItems, null, 2));
      
      // Try accessing different pages
      console.log('\n6. Testing page access...');
      
      const pagesToTest = [
        { path: '/saltstack', name: 'SaltStack' },
        { path: '/kubernetes', name: 'Kubernetes' },
        { path: '/ansible', name: 'Ansible' },
        { path: '/monitoring', name: 'Monitoring' },
        { path: '/admin/users', name: 'Admin Users' },
        { path: '/slurm', name: 'SLURM' },
      ];
      
      for (const testPage of pagesToTest) {
        await page.goto(`${BASE_URL}${testPage.path}`);
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(1000);
        
        const finalUrl = page.url();
        const has403 = await page.$('.ant-result-403, [class*="forbidden"]');
        const hasError = await page.$('.ant-result-error, .ant-alert-error');
        
        const status = has403 ? '403 FORBIDDEN' : (hasError ? 'ERROR' : 'OK');
        console.log(`  ${testPage.name}: ${status} (URL: ${finalUrl})`);
        
        await page.screenshot({ 
          path: `test-screenshots/test1-page-${testPage.name.toLowerCase().replace(/\s+/g, '-')}.png` 
        });
      }
      
      // Check API permissions
      console.log('\n7. Testing API access...');
      const apiResults = await page.evaluate(async (baseUrl) => {
        const results = {};
        const endpoints = [
          '/api/auth/me',
          '/api/saltstack/minions',
          '/api/kubernetes/clusters',
          '/api/admin/users',
        ];
        
        for (const endpoint of endpoints) {
          try {
            const resp = await fetch(`${baseUrl}${endpoint}`, {
              credentials: 'include'
            });
            results[endpoint] = { status: resp.status, ok: resp.ok };
          } catch (e) {
            results[endpoint] = { error: e.message };
          }
        }
        return results;
      }, BASE_URL);
      
      console.log('API Results:', JSON.stringify(apiResults, null, 2));
      
    } else {
      console.log('ERROR: Login failed, still on login page');
    }
    
  } catch (error) {
    console.error('Test error:', error);
    await page.screenshot({ path: 'test-screenshots/test1-error.png' });
  } finally {
    await browser.close();
    console.log('\n=== Test Complete ===');
  }
}

runTest().catch(console.error);
