const { test, expect } = require('@playwright/test');

test.describe('Monitoring Page Redirect Loop Fix', () => {
  test.beforeEach(async ({ page }) => {
    // Enable request/response logging
    page.on('request', request => {
      if (request.url().includes('/monitoring') || request.url().includes('/projects') || request.url().includes('/nightingale')) {
        console.log(`>> ${request.method()} ${request.url()}`);
      }
    });
    
    page.on('response', response => {
      if (response.url().includes('/monitoring') || response.url().includes('/projects') || response.url().includes('/nightingale')) {
        console.log(`<< ${response.status()} ${response.url()}`);
      }
    });
  });

  test('should login as admin and access /monitoring without redirect loop', async ({ page }) => {
    // Track all navigations
    const navigations = [];
    page.on('framenavigated', frame => {
      if (frame === page.mainFrame()) {
        navigations.push(frame.url());
        console.log('Navigated to:', frame.url());
      }
    });

    // Step 1: Navigate to login page
    console.log('Step 1: Going to login page');
    await page.goto('http://192.168.18.154:8080/login');
    await page.waitForLoadState('networkidle');
    
    // Step 2: Login as admin (Ant Design Form.Item uses different structure)
    console.log('Step 2: Logging in as admin');
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // Wait for login to complete
    await page.waitForURL(/\/(projects|dashboard|monitoring)/, { timeout: 10000 });
    console.log('Logged in, current URL:', page.url());
    
    
    // Step 3: Wait for login to complete
    console.log('Step 3: Waiting for login to complete');
    await page.waitForURL('http://192.168.18.154:8080/projects', { timeout: 10000 });
    console.log('Login successful, redirected to:', page.url());
    
    // Step 4: Navigate to /monitoring
    console.log('Step 4: Navigating to /monitoring');
    await page.goto('http://192.168.18.154:8080/monitoring');
    
    // Step 5: Wait for page to load and check URL
    console.log('Step 5: Waiting for /monitoring page to load');
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    const currentUrl = page.url();
    console.log('Current URL after navigation:', currentUrl);
    console.log('All navigations:', navigations);
    
    // Take a screenshot for debugging
    await page.screenshot({ path: 'test-screenshots/monitoring-page.png', fullPage: true });
    
    // Check if we stayed on /monitoring
    expect(currentUrl).toBe('http://192.168.18.154:8080/monitoring');
    
    // Check if Nightingale iframe is present
    const iframeExists = await page.locator('iframe[title="Nightingale Monitoring"]').count() > 0;
    console.log('Nightingale iframe exists:', iframeExists);
    
    if (!iframeExists) {
      // Check for error messages or 403 page
      const pageContent = await page.content();
      console.log('Page content sample:', pageContent.substring(0, 500));
      
      // Check for specific error indicators
      const has403 = await page.locator('text=403').count() > 0;
      const hasPermissionError = await page.locator('text=权限').count() > 0;
      
      console.log('Has 403 error:', has403);
      console.log('Has permission error:', hasPermissionError);
    }
    
    expect(iframeExists).toBe(true);    // Track redirects
    const redirects = [];
    let redirectCount = 0;
    const maxRedirects = 10;
    
    page.on('framenavigated', frame => {
      if (frame === page.mainFrame()) {
        const url = frame.url();
        redirects.push(url);
        redirectCount++;
        console.log(`Navigation ${redirectCount}: ${url}`);
        
        if (redirectCount > maxRedirects) {
          console.error('Detected redirect loop!');
        }
      }
    });
    
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    const finalUrl = page.url();
    console.log('\n=== Final URL ===');
    console.log(finalUrl);
    console.log('\n=== All redirects ===');
    console.log(redirects);
    
    // Step 4: Verify no redirect loop occurred
    expect(redirectCount).toBeLessThan(5); // Should not have excessive redirects
    
    // Step 5: Check if we landed on /monitoring or /projects
    const currentPath = new URL(finalUrl).pathname;
    console.log('Current path:', currentPath);
    
    if (currentPath.includes('/projects')) {
      console.error('❌ ISSUE: Redirected to /projects instead of /monitoring');
      
      // Capture screenshot for debugging
      await page.screenshot({ path: 'test-screenshots/monitoring-redirect-to-projects.png', fullPage: true });
    } else if (currentPath.includes('/monitoring')) {
      console.log('✅ SUCCESS: Stayed on /monitoring');
      
      // Check if Nightingale iframe is present
      const iframe = await page.locator('iframe[title*="Nightingale"], iframe[src*="nightingale"]').first();
      const iframeExists = await iframe.count() > 0;
      
      if (iframeExists) {
        console.log('✅ Nightingale iframe found');
        await page.screenshot({ path: 'test-screenshots/monitoring-page-with-iframe.png', fullPage: true });
      } else {
        console.log('❌ Nightingale iframe NOT found');
        await page.screenshot({ path: 'test-screenshots/monitoring-page-no-iframe.png', fullPage: true });
      }
      
      expect(iframeExists).toBeTruthy();
    }
    
    // Step 6: Verify admin has access to monitoring
    const pageContent = await page.content();
    const hasMonitoringTitle = pageContent.includes('监控') || pageContent.includes('Monitoring') || pageContent.includes('Nightingale');
    
    console.log('Page has monitoring title:', hasMonitoringTitle);
    
    // Final assertion
    expect(currentPath).toContain('/monitoring');
  });

  test('should check JWT token and SSO capability', async ({ page }) => {
    // Login
    await page.goto('http://192.168.18.154:8080/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(projects|dashboard|monitoring)/, { timeout: 10000 });
    
    // Check for JWT token in localStorage or cookies
    const token = await page.evaluate(() => {
      return localStorage.getItem('token') || localStorage.getItem('jwt_token');
    });
    
    console.log('JWT token present:', !!token);
    console.log('Token length:', token ? token.length : 0);
    
    // Check cookies for SSO
    const cookies = await page.context().cookies();
    const ssoCookies = cookies.filter(c => 
      c.name.includes('token') || 
      c.name.includes('jwt') || 
      c.name.includes('session') ||
      c.name.includes('ai_infra')
    );
    
    console.log('SSO-related cookies:', ssoCookies.map(c => c.name));
    
    expect(token).toBeTruthy();
    expect(ssoCookies.length).toBeGreaterThan(0);
  });

  test('should verify Nightingale service accessibility', async ({ page }) => {
    // Login first
    await page.goto('http://192.168.18.154:8080/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(projects|dashboard|monitoring)/, { timeout: 10000 });
    
    // Try to access Nightingale directly
    console.log('\n=== Testing direct Nightingale access ===');
    
    const nightingaleResponse = await page.goto('http://192.168.18.154:8080/nightingale/', { 
      waitUntil: 'networkidle',
      timeout: 15000 
    });
    
    console.log('Nightingale response status:', nightingaleResponse.status());
    console.log('Nightingale final URL:', page.url());
    
    // Check if we can access Nightingale API
    const apiResponse = await page.request.get('http://192.168.18.154:8080/api/n9e/self/profile', {
      headers: {
        'Cookie': await page.context().cookies().then(cookies => 
          cookies.map(c => `${c.name}=${c.value}`).join('; ')
        )
      }
    });
    
    console.log('Nightingale API response status:', apiResponse.status());
    
    if (apiResponse.ok()) {
      const apiData = await apiResponse.json();
      console.log('Nightingale API data:', apiData);
    }
    
    await page.screenshot({ path: 'test-screenshots/nightingale-direct-access.png', fullPage: true });
  });
});
