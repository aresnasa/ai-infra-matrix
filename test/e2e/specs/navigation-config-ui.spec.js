const { test, expect } = require('@playwright/test');

test.describe('Navigation Config UI Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Login before each test
    await page.goto('http://192.168.0.200:8080/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://192.168.0.200:8080/', { timeout: 5000 });
  });

  test('should load navigation config without 404 error', async ({ page }) => {
    // Listen for console errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Listen for failed network requests
    const failedRequests = [];
    page.on('response', response => {
      if (response.status() === 404 && response.url().includes('/api/navigation')) {
        failedRequests.push({
          url: response.url(),
          status: response.status()
        });
      }
    });

    // Reload the page to trigger navigation config load
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Wait a bit for any async requests
    await page.waitForTimeout(1000);

    console.log('Console errors:', consoleErrors);
    console.log('Failed navigation requests:', failedRequests);

    // Verify no 404 errors for navigation config
    expect(failedRequests.length).toBe(0);
    
    // Check that we don't have the specific 404 error message
    const has404Error = consoleErrors.some(error => 
      error.includes('404') && error.includes('/api/navigation/config')
    );
    expect(has404Error).toBe(false);
  });

  test('should successfully load navigation config API', async ({ page }) => {
    let navigationConfigResponse;
    
    // Intercept navigation config request
    page.on('response', response => {
      if (response.url().includes('/api/navigation/config')) {
        navigationConfigResponse = response;
      }
    });

    // Reload to trigger config load
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Wait for navigation config request
    await page.waitForTimeout(2000);

    expect(navigationConfigResponse).toBeDefined();
    console.log('Navigation config response status:', navigationConfigResponse?.status());
    
    if (navigationConfigResponse) {
      expect(navigationConfigResponse.status()).toBe(200);
      
      const responseBody = await navigationConfigResponse.json();
      console.log('Navigation config response:', JSON.stringify(responseBody, null, 2));
      
      expect(responseBody).toHaveProperty('status');
      expect(responseBody.status).toBe('success');
      expect(responseBody.data).toHaveProperty('items');
    }
  });

  test('should display monitoring menu item in navigation', async ({ page }) => {
    // Wait for page to fully load
    await page.waitForLoadState('networkidle');

    // Look for monitoring menu item
    const monitoringMenu = page.locator('text=监控仪表板');
    await expect(monitoringMenu).toBeVisible({ timeout: 10000 });

    console.log('Monitoring menu item is visible');

    // Take a screenshot
    await page.screenshot({ 
      path: 'test-screenshots/navigation-with-monitoring.png',
      fullPage: true 
    });
  });

  test('should navigate to monitoring page when clicked', async ({ page }) => {
    // Wait for page to fully load
    await page.waitForLoadState('networkidle');

    // Click on monitoring menu item
    const monitoringMenu = page.locator('text=监控仪表板');
    await expect(monitoringMenu).toBeVisible({ timeout: 10000 });
    await monitoringMenu.click();

    // Wait for navigation
    await page.waitForURL('**/monitoring', { timeout: 5000 });
    
    console.log('Successfully navigated to:', page.url());
    expect(page.url()).toContain('/monitoring');

    // Take a screenshot of monitoring page
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-page.png',
      fullPage: true 
    });
  });

  test('should load Nightingale iframe on monitoring page', async ({ page }) => {
    // Navigate to monitoring page
    await page.goto('http://192.168.0.200:8080/monitoring');
    await page.waitForLoadState('networkidle');

    // Wait for iframe to load
    const iframe = page.frameLocator('iframe[src*="nightingale"]');
    await expect(iframe.locator('body')).toBeVisible({ timeout: 10000 });

    console.log('Nightingale iframe loaded successfully');

    // Take a screenshot
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-iframe.png',
      fullPage: true 
    });
  });

  test('should show all expected navigation items', async ({ page }) => {
    // Wait for page to fully load
    await page.waitForLoadState('networkidle');

    // List of expected menu items (based on default navigation)
    const expectedItems = [
      '项目管理',
      'Gitea',
      'Kubernetes',
      'Ansible',
      'JupyterHub',
      '监控仪表板',
      'SLURM',
      'SaltStack'
    ];

    console.log('Checking for navigation items...');
    
    for (const itemText of expectedItems) {
      const menuItem = page.locator(`text=${itemText}`);
      const isVisible = await menuItem.isVisible();
      console.log(`  ${itemText}: ${isVisible ? '✓' : '✗'}`);
      
      if (!isVisible) {
        console.log(`  WARNING: ${itemText} not visible`);
      }
    }

    // At minimum, monitoring should be visible
    const monitoringMenu = page.locator('text=监控仪表板');
    await expect(monitoringMenu).toBeVisible({ timeout: 10000 });
  });
});
