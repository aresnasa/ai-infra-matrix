const { test, expect } = require('@playwright/test');

test.describe('Nightingale Admin User Verification', () => {
  const nightingaleUrl = 'http://192.168.18.114:8080/monitoring';
  const adminUsername = 'admin';
  const adminPassword = 'admin123';

  test('should check if ProxyAuth is disabled', async ({ page }) => {
    console.log('🔍 Checking if ProxyAuth is disabled...');
    
    // Visit the page
    await page.goto(nightingaleUrl, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // Take screenshot
    await page.screenshot({ path: 'test-screenshots/nightingale-initial-page.png', fullPage: true });
    console.log('📸 Screenshot saved: nightingale-initial-page.png');
    
    // Check if we see a login page or are auto-logged in
    const hasLoginForm = await page.locator('input[type="password"]').count() > 0;
    const hasUsername = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').count() > 0;
    
    console.log(`📊 Login form present: ${hasLoginForm ? '✓ Yes' : '✗ No (auto-logged in)'}`);
    console.log(`📊 Username field present: ${hasUsername ? '✓ Yes' : '✗ No'}`);
    
    if (hasLoginForm && hasUsername) {
      console.log('✅ ProxyAuth is disabled - login form is shown');
    } else {
      console.log('⚠️  ProxyAuth may still be enabled - no login form (auto-logged in)');
    }
    
    expect(hasLoginForm).toBeTruthy();
    expect(hasUsername).toBeTruthy();
  });

  test('should login with admin credentials', async ({ page }) => {
    console.log('🔐 Logging in with admin credentials...');
    
    await page.goto(nightingaleUrl, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // Fill in admin credentials
    const usernameInput = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
    await usernameInput.fill(adminUsername);
    
    const passwordInput = await page.locator('input[type="password"]').first();
    await passwordInput.fill(adminPassword);
    
    console.log('✏️  Filled in credentials: admin/admin123');
    
    // Click login
    const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
    await loginButton.click();
    
    console.log('🖱️  Clicked login button');
    await page.waitForTimeout(3000);
    
    // Take screenshot after login
    await page.screenshot({ path: 'test-screenshots/nightingale-after-admin-login.png', fullPage: true });
    console.log('📸 Screenshot saved: nightingale-after-admin-login.png');
    
    // Check if login was successful
    const currentUrl = page.url();
    console.log('📍 Current URL:', currentUrl);
    
    expect(currentUrl).not.toContain('/login');
    console.log('✅ Admin login successful');
  });

  test('should verify logged in as admin user', async ({ page }) => {
    console.log('👤 Verifying logged in user is admin...');
    
    // Login first
    await page.goto(nightingaleUrl, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    const usernameInput = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
    await usernameInput.fill(adminUsername);
    
    const passwordInput = await page.locator('input[type="password"]').first();
    await passwordInput.fill(adminPassword);
    
    const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
    await loginButton.click();
    
    await page.waitForTimeout(3000);
    
    // Check page content for user info
    const pageContent = await page.content();
    
    // Look for admin user indicators
    const hasAdminText = pageContent.includes('admin') || pageContent.includes('Administrator');
    const hasAnonymousText = pageContent.includes('anonymous');
    
    console.log('📊 User verification:');
    console.log(`   - "admin" in page: ${hasAdminText ? '✓' : '✗'}`);
    console.log(`   - "anonymous" in page: ${hasAnonymousText ? '⚠️  Yes (should not be)' : '✓ No'}`);
    
    // Try to find user display name in header/menu
    const userMenuText = await page.locator('[class*="user"], [class*="User"], [class*="avatar"], [class*="Avatar"]').allTextContents();
    console.log('👤 User menu text:', userMenuText);
    
    // Click on user menu to see dropdown
    const userMenu = await page.locator('[class*="avatar"]').first();
    if (await userMenu.count() > 0) {
      await userMenu.click();
      await page.waitForTimeout(1000);
      
      // Take screenshot of user dropdown
      await page.screenshot({ path: 'test-screenshots/nightingale-user-dropdown.png', fullPage: true });
      console.log('📸 Screenshot saved: nightingale-user-dropdown.png');
      
      // Get dropdown content
      const dropdownText = await page.locator('body').textContent();
      console.log('📄 Dropdown content includes "admin":', dropdownText.includes('admin'));
      console.log('📄 Dropdown content includes "anonymous":', dropdownText.includes('anonymous'));
      
      if (dropdownText.includes('admin')) {
        console.log('✅ Logged in as admin user');
      } else if (dropdownText.includes('anonymous')) {
        console.log('❌ Still logged in as anonymous user!');
      }
    }
    
    // Check API calls for user info
    const responses = [];
    page.on('response', response => {
      if (response.url().includes('/api/') && response.url().includes('user')) {
        responses.push({
          url: response.url(),
          status: response.status()
        });
      }
    });
    
    await page.reload();
    await page.waitForTimeout(2000);
    
    if (responses.length > 0) {
      console.log('📡 User-related API calls:');
      responses.forEach(r => console.log(`   - ${r.url} [${r.status}]`));
    }
  });

  test('should check database for current sessions', async ({ page }) => {
    console.log('🗄️  Checking database for active sessions...');
    
    const { exec } = require('child_process');
    const { promisify } = require('util');
    const execAsync = promisify(exec);
    
    try {
      // Check all users
      const { stdout: usersOutput } = await execAsync(
        `docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT id, username, nickname, roles FROM users;"`
      );
      
      console.log('📊 All users in Nightingale database:');
      console.log(usersOutput);
      
      // Check if there are any JWT tokens in Redis
      const { stdout: redisOutput } = await execAsync(
        `docker exec ai-infra-redis redis-cli --no-auth-warning -a your-redis-password KEYS "/jwt/*" 2>/dev/null || echo "No JWT keys found"`
      );
      
      console.log('🔑 JWT tokens in Redis:');
      console.log(redisOutput || '(none)');
      
    } catch (error) {
      console.error('❌ Error checking database:', error.message);
    }
  });

  test('should verify ProxyAuth is disabled in config', async ({ page }) => {
    console.log('⚙️  Verifying ProxyAuth configuration...');
    
    const { exec } = require('child_process');
    const { promisify } = require('util');
    const execAsync = promisify(exec);
    
    try {
      // Check if ProxyAuth is disabled in config
      const { stdout: configOutput } = await execAsync(
        `docker exec ai-infra-nightingale cat /app/etc/config.toml | grep -A3 "\\[HTTP.ProxyAuth\\]"`
      );
      
      console.log('📋 ProxyAuth configuration:');
      console.log(configOutput);
      
      const isDisabled = configOutput.includes('Enable = false');
      
      if (isDisabled) {
        console.log('✅ ProxyAuth is disabled in config');
      } else {
        console.log('❌ ProxyAuth is still enabled in config!');
      }
      
      expect(isDisabled).toBeTruthy();
      
    } catch (error) {
      console.error('❌ Error checking config:', error.message);
      throw error;
    }
  });
});
