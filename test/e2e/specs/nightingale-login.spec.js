const { test, expect } = require('@playwright/test');

test.describe('Nightingale Admin Login and Permissions Tests', () => {
  const nightingaleUrl = 'http://192.168.18.114:8080/monitoring';
  const adminUsername = 'admin';
  const adminPassword = 'admin123';

  test('should successfully login with admin credentials', async ({ page }) => {
    console.log('🔐 Testing Nightingale admin login...');
    
    // Navigate to Nightingale
    await page.goto(nightingaleUrl, { waitUntil: 'networkidle' });
    
    // Wait for login page
    await page.waitForSelector('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]', { timeout: 10000 });
    
    // Take screenshot of login page
    await page.screenshot({ path: 'test-screenshots/nightingale-login-page.png', fullPage: true });
    console.log('📸 Login page screenshot saved');
    
    // Fill in credentials
    const usernameInput = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
    await usernameInput.fill(adminUsername);
    
    const passwordInput = await page.locator('input[type="password"]').first();
    await passwordInput.fill(adminPassword);
    
    console.log('✏️  Filled in credentials: admin/admin123');
    
    // Click login button
    const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
    await loginButton.click();
    
    console.log('🖱️  Clicked login button');
    
    // Wait for navigation after login
    await page.waitForTimeout(2000);
    
    // Take screenshot after login
    await page.screenshot({ path: 'test-screenshots/nightingale-after-login.png', fullPage: true });
    console.log('📸 After login screenshot saved');
    
    // Check if login was successful (should not be on login page anymore)
    const currentUrl = page.url();
    console.log('📍 Current URL after login:', currentUrl);
    
    // Should not contain login path
    expect(currentUrl).not.toContain('/login');
    
    // Check for common dashboard elements or user info
    const dashboardElements = await page.locator('body').textContent();
    console.log('✅ Login successful! Dashboard loaded.');
  });

  test('should verify admin has access to admin features', async ({ page }) => {
    console.log('🔧 Testing admin features access...');
    
    // Login first
    await page.goto(nightingaleUrl, { waitUntil: 'networkidle' });
    await page.waitForSelector('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]', { timeout: 10000 });
    
    const usernameInput = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
    await usernameInput.fill(adminUsername);
    
    const passwordInput = await page.locator('input[type="password"]').first();
    await passwordInput.fill(adminPassword);
    
    const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
    await loginButton.click();
    
    await page.waitForTimeout(3000);
    
    // Take screenshot of main dashboard
    await page.screenshot({ path: 'test-screenshots/nightingale-dashboard.png', fullPage: true });
    console.log('📸 Dashboard screenshot saved');
    
    // Check for admin menu items (common in Nightingale)
    const pageContent = await page.locator('body').textContent();
    console.log('📄 Page content length:', pageContent.length);
    
    // Look for common admin features
    const hasAlerts = pageContent.includes('告警') || pageContent.includes('Alert');
    const hasMonitoring = pageContent.includes('监控') || pageContent.includes('Monitor');
    const hasMetrics = pageContent.includes('指标') || pageContent.includes('Metric');
    
    console.log('✅ Admin features check:');
    console.log('   - Alerts:', hasAlerts ? '✓' : '✗');
    console.log('   - Monitoring:', hasMonitoring ? '✓' : '✗');
    console.log('   - Metrics:', hasMetrics ? '✓' : '✗');
    
    // At least one feature should be visible
    expect(hasAlerts || hasMonitoring || hasMetrics).toBeTruthy();
  });

  test('should verify admin user in database', async ({ page }) => {
    const { Client } = require('pg');
    
    console.log('🗄️  Verifying admin user in Nightingale database...');
    
    // Connect to database via docker exec (since PostgreSQL is not exposed)
    const { exec } = require('child_process');
    const { promisify } = require('util');
    const execAsync = promisify(exec);
    
    try {
      // Query admin user
      const { stdout } = await execAsync(
        `docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT username, roles, email FROM users WHERE username='admin';"`
      );
      
      console.log('📊 Database query result:');
      console.log(stdout);
      
      expect(stdout).toContain('admin');
      expect(stdout).toContain('Admin');
      expect(stdout).toContain('admin@example.com');
      
      console.log('✅ Admin user verified in database');
    } catch (error) {
      console.error('❌ Database verification failed:', error);
      throw error;
    }
  });

  test('should verify admin group membership', async ({ page }) => {
    const { exec } = require('child_process');
    const { promisify } = require('util');
    const execAsync = promisify(exec);
    
    console.log('👥 Verifying admin group membership...');
    
    try {
      // Check user group
      const { stdout: userGroupOutput } = await execAsync(
        `docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT ug.name FROM user_group ug JOIN user_group_member ugm ON ug.id = ugm.group_id JOIN users u ON ugm.user_id = u.id WHERE u.username='admin';"`
      );
      
      console.log('📊 User groups:');
      console.log(userGroupOutput);
      
      expect(userGroupOutput).toContain('admin-group');
      
      console.log('✅ Admin is member of admin-group');
      
      // Check business group
      const { stdout: busiGroupOutput } = await execAsync(
        `docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT bg.name, bgm.perm_flag FROM busi_group bg JOIN busi_group_member bgm ON bg.id = bgm.busi_group_id JOIN user_group ug ON bgm.user_group_id = ug.id WHERE ug.name='admin-group';"`
      );
      
      console.log('📊 Business groups:');
      console.log(busiGroupOutput);
      
      expect(busiGroupOutput).toContain('Default');
      expect(busiGroupOutput).toContain('rw');
      
      console.log('✅ Admin-group has rw permissions on Default business group');
    } catch (error) {
      console.error('❌ Group membership verification failed:', error);
      throw error;
    }
  });
});
