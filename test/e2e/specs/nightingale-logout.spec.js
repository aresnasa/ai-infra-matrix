const { test, expect } = require('@playwright/test');

test.describe('Nightingale Logout Functionality Tests', () => {
  const nightingaleUrl = 'http://192.168.18.114:8080/monitoring';
  const adminUsername = 'admin';
  const adminPassword = 'admin123';

  test('should check logout button availability', async ({ page }) => {
    console.log('🔐 Testing Nightingale logout functionality...');
    
    // Login first
    await page.goto(nightingaleUrl, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // Fill in credentials
    const usernameInput = await page.locator('input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
    await usernameInput.fill(adminUsername);
    
    const passwordInput = await page.locator('input[type="password"]').first();
    await passwordInput.fill(adminPassword);
    
    const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
    await loginButton.click();
    
    console.log('✅ Logged in successfully');
    await page.waitForTimeout(3000);
    
    // Take screenshot after login
    await page.screenshot({ path: 'test-screenshots/nightingale-logout-before.png', fullPage: true });
    console.log('📸 Screenshot saved: nightingale-logout-before.png');
    
    // Search for logout button or user menu
    const pageContent = await page.content();
    
    // Check for common logout elements
    const logoutButtons = await page.locator('button:has-text("退出"), button:has-text("登出"), button:has-text("Logout"), a:has-text("退出"), a:has-text("登出"), a:has-text("Logout")').count();
    console.log(`🔍 Found ${logoutButtons} logout button(s)`);
    
    // Check for user menu/dropdown
    const userMenus = await page.locator('[class*="user"], [class*="User"], [class*="avatar"], [class*="Avatar"], [class*="profile"], [class*="Profile"]').count();
    console.log(`🔍 Found ${userMenus} user menu element(s)`);
    
    // Try to find and click user menu if exists
    if (userMenus > 0) {
      console.log('🖱️  Attempting to click user menu...');
      const userMenu = await page.locator('[class*="user"], [class*="User"], [class*="avatar"], [class*="Avatar"]').first();
      
      try {
        await userMenu.click({ timeout: 5000 });
        await page.waitForTimeout(1000);
        
        // Take screenshot after clicking user menu
        await page.screenshot({ path: 'test-screenshots/nightingale-logout-menu-open.png', fullPage: true });
        console.log('📸 Screenshot saved: nightingale-logout-menu-open.png');
        
        // Check for logout button in dropdown
        const dropdownLogoutButtons = await page.locator('button:has-text("退出"), button:has-text("登出"), button:has-text("Logout"), a:has-text("退出"), a:has-text("登出"), a:has-text("Logout")').count();
        console.log(`🔍 Found ${dropdownLogoutButtons} logout button(s) in dropdown`);
        
        if (dropdownLogoutButtons > 0) {
          console.log('✅ Logout button found in user menu');
        } else {
          console.log('⚠️  No logout button found in user menu');
        }
      } catch (error) {
        console.log('⚠️  Could not open user menu:', error.message);
      }
    }
    
    // Check page HTML for logout-related elements
    const hasLogoutInHTML = pageContent.includes('logout') || pageContent.includes('登出') || pageContent.includes('退出');
    console.log(`🔍 Logout keyword in HTML: ${hasLogoutInHTML ? '✓' : '✗'}`);
    
    // Check for proxy auth warning
    const hasProxyAuthWarning = pageContent.includes('proxy auth') || pageContent.includes('Proxy Auth');
    console.log(`🔍 Proxy auth warning: ${hasProxyAuthWarning ? '✓ Found' : '✗ Not found'}`);
    
    // Log all visible text content
    const bodyText = await page.locator('body').textContent();
    if (bodyText.includes('proxy auth') || bodyText.includes('Proxy Auth')) {
      console.log('⚠️  WARNING: Proxy auth is enabled, logout may not be supported');
    }
    
    // Take final screenshot
    await page.screenshot({ path: 'test-screenshots/nightingale-logout-final.png', fullPage: true });
    console.log('📸 Screenshot saved: nightingale-logout-final.png');
    
    // Report findings
    console.log('\n📊 Logout Functionality Report:');
    console.log(`   - Logout buttons visible: ${logoutButtons > 0 ? '✓' : '✗'}`);
    console.log(`   - User menu available: ${userMenus > 0 ? '✓' : '✗'}`);
    console.log(`   - Logout in HTML: ${hasLogoutInHTML ? '✓' : '✗'}`);
    console.log(`   - Proxy auth warning: ${hasProxyAuthWarning ? '✓' : '✗'}`);
  });

  test('should attempt to logout if button exists', async ({ page }) => {
    console.log('🚪 Attempting to logout from Nightingale...');
    
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
    console.log('✅ Logged in');
    
    // Try multiple strategies to find logout
    let logoutSuccess = false;
    
    // Strategy 1: Direct logout button
    const directLogout = page.locator('button:has-text("退出"), button:has-text("登出"), button:has-text("Logout"), a:has-text("退出"), a:has-text("登出"), a:has-text("Logout")').first();
    if (await directLogout.count() > 0) {
      console.log('🖱️  Strategy 1: Clicking direct logout button...');
      try {
        await directLogout.click({ timeout: 5000 });
        await page.waitForTimeout(2000);
        logoutSuccess = true;
        console.log('✅ Clicked logout button');
      } catch (error) {
        console.log('❌ Failed to click logout button:', error.message);
      }
    }
    
    // Strategy 2: User menu then logout
    if (!logoutSuccess) {
      console.log('🖱️  Strategy 2: Looking for user menu...');
      const userMenuSelectors = [
        '[class*="user-menu"]',
        '[class*="userMenu"]',
        '[class*="user-dropdown"]',
        '[class*="avatar"]',
        '[class*="header-user"]',
        '.ant-dropdown-trigger',
        '[role="button"]'
      ];
      
      for (const selector of userMenuSelectors) {
        const menu = page.locator(selector);
        const count = await menu.count();
        
        if (count > 0) {
          console.log(`   Found ${count} element(s) with selector: ${selector}`);
          try {
            await menu.first().click({ timeout: 3000 });
            await page.waitForTimeout(1000);
            
            // Now look for logout in dropdown
            const dropdownLogout = page.locator('li:has-text("退出"), li:has-text("登出"), li:has-text("Logout"), [role="menuitem"]:has-text("退出"), [role="menuitem"]:has-text("登出"), [role="menuitem"]:has-text("Logout")');
            
            if (await dropdownLogout.count() > 0) {
              await dropdownLogout.first().click();
              await page.waitForTimeout(2000);
              logoutSuccess = true;
              console.log('✅ Clicked logout in dropdown menu');
              break;
            }
          } catch (error) {
            console.log(`   Could not interact with ${selector}:`, error.message);
          }
        }
      }
    }
    
    // Take screenshot after logout attempt
    await page.screenshot({ path: 'test-screenshots/nightingale-after-logout-attempt.png', fullPage: true });
    console.log('📸 Screenshot saved: nightingale-after-logout-attempt.png');
    
    // Check if we're back at login page
    const currentUrl = page.url();
    const isLoginPage = currentUrl.includes('login') || await page.locator('input[type="password"]').count() > 0;
    
    console.log(`\n📊 Logout Attempt Result:`);
    console.log(`   - Logout action performed: ${logoutSuccess ? '✓' : '✗'}`);
    console.log(`   - Current URL: ${currentUrl}`);
    console.log(`   - At login page: ${isLoginPage ? '✓' : '✗'}`);
    
    if (!logoutSuccess) {
      console.log('\n⚠️  WARNING: Could not find logout button');
      console.log('   This may indicate proxy auth is enabled and logout is disabled');
    }
  });

  test('should check Nightingale configuration for proxy auth', async ({ page }) => {
    console.log('🔧 Checking Nightingale configuration...');
    
    const { exec } = require('child_process');
    const { promisify } = require('util');
    const execAsync = promisify(exec);
    
    try {
      // Check if Nightingale has proxy auth enabled in configs table
      const { stdout: configOutput } = await execAsync(
        `docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT ckey, cval FROM configs WHERE ckey LIKE '%auth%' OR ckey LIKE '%proxy%';"`
      );
      
      console.log('📊 Nightingale auth configuration:');
      console.log(configOutput || '(no auth-related configs found)');
      
      // Check sso_config table
      const { stdout: ssoOutput } = await execAsync(
        `docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT id, name, enabled FROM sso_config;"`
      );
      
      console.log('\n📊 SSO Configuration:');
      console.log(ssoOutput || '(no SSO configs found)');
      
      // Check if there's a proxy auth environment variable
      const { stdout: envOutput } = await execAsync(
        `docker exec ai-infra-nightingale env | grep -i auth || echo "No auth-related env vars"`
      );
      
      console.log('\n📊 Nightingale container environment:');
      console.log(envOutput);
      
    } catch (error) {
      console.error('❌ Error checking configuration:', error.message);
    }
  });

  test('should analyze Nightingale docker-compose configuration', async ({ page }) => {
    console.log('🐳 Analyzing Nightingale docker-compose configuration...');
    
    const fs = require('fs');
    const path = require('path');
    
    try {
      const dockerComposePath = path.join('/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix', 'docker-compose.yml');
      const dockerComposeContent = fs.readFileSync(dockerComposePath, 'utf8');
      
      // Find Nightingale service configuration
      const nightingaleSection = dockerComposeContent.split('nightingale:')[1]?.split(/^\s{2}[a-z]/m)[0] || '';
      
      console.log('📋 Nightingale service configuration:');
      
      // Check for auth-related environment variables
      const authEnvVars = nightingaleSection.match(/.*AUTH.*/gi) || [];
      const proxyEnvVars = nightingaleSection.match(/.*PROXY.*/gi) || [];
      
      if (authEnvVars.length > 0) {
        console.log('\n🔑 Auth-related environment variables:');
        authEnvVars.forEach(env => console.log('   -', env.trim()));
      } else {
        console.log('\n✓ No auth-related environment variables found in docker-compose');
      }
      
      if (proxyEnvVars.length > 0) {
        console.log('\n🔐 Proxy-related environment variables:');
        proxyEnvVars.forEach(env => console.log('   -', env.trim()));
      } else {
        console.log('\n✓ No proxy-related environment variables found in docker-compose');
      }
      
      // Check for nginx proxy configuration
      const nginxSection = dockerComposeContent.split('nginx:')[1]?.split(/^\s{2}[a-z]/m)[0] || '';
      const hasNginxAuth = nginxSection.includes('auth') || nginxSection.includes('AUTH');
      
      console.log(`\n🌐 Nginx proxy auth: ${hasNginxAuth ? '⚠️  Enabled' : '✓ Not detected'}`);
      
    } catch (error) {
      console.error('❌ Error reading docker-compose.yml:', error.message);
    }
  });
});
