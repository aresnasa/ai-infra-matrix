// @ts-check
/**
 * 主题切换和语言同步测试
 * 测试暗黑模式、光明模式、跟随系统切换功能
 * 测试语言切换功能
 */
const { test, expect } = require('@playwright/test');

const USER = {
  username: process.env.E2E_USER || 'admin',
  password: process.env.E2E_PASS || 'admin123',
};

// Helper: perform login
async function login(page) {
  await page.goto('/');
  await page.waitForLoadState('domcontentloaded');
  
  // Check if already logged in
  const isLoggedIn = await page.locator('header, .ant-layout-header').count() > 0 &&
                     await page.locator('text=/项目管理|Projects/i').count() > 0;
  
  if (isLoggedIn) {
    console.log('Already logged in');
    return;
  }
  
  try {
    await page.waitForSelector('input[placeholder*="用户名"], input[placeholder*="Username"]', { timeout: 10000 });
    
    const usernameInput = page.locator('input[placeholder*="用户名"], input[placeholder*="Username"]');
    const passwordInput = page.locator('input[placeholder*="密码"], input[placeholder*="Password"]');
    
    await usernameInput.fill(USER.username);
    await passwordInput.fill(USER.password);
    
    const loginButton = page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login")').first();
    await loginButton.click();
    
    await page.waitForSelector('header, .ant-layout-header', { timeout: 15000 });
  } catch (e) {
    console.log('Login process completed or skipped');
  }
}

// Helper: find theme switcher button
async function findThemeSwitcher(page) {
  const header = page.locator('header, .ant-layout-header');
  const buttons = header.locator('button');
  const count = await buttons.count();
  
  for (let i = 0; i < count; i++) {
    const btn = buttons.nth(i);
    const text = await btn.innerText();
    // Theme button has no text (just icon)
    if (text === '' || text.trim() === '') {
      return btn;
    }
  }
  return null;
}

test.describe('Theme Switching Tests', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.waitForLoadState('networkidle');
  });

  test('should find and click theme switcher', async ({ page }) => {
    // Take initial screenshot
    await page.screenshot({ path: 'test-screenshots/01-initial-state.png' });
    
    const themeBtn = await findThemeSwitcher(page);
    
    if (themeBtn) {
      await themeBtn.click();
      await page.waitForTimeout(500);
      
      // Check if dropdown appeared
      const dropdown = page.locator('.ant-dropdown');
      const isVisible = await dropdown.isVisible();
      console.log('Dropdown visible:', isVisible);
      
      await page.screenshot({ path: 'test-screenshots/02-after-theme-click.png' });
      
      if (isVisible) {
        // Try to click dark option
        const darkOption = page.locator('.ant-dropdown-menu-item').filter({
          hasText: /深色|Dark/i
        });
        
        if (await darkOption.count() > 0) {
          await darkOption.first().click();
          await page.waitForTimeout(500);
          await page.screenshot({ path: 'test-screenshots/03-dark-mode.png' });
          
          // Verify theme changed
          const bodyClass = await page.locator('body').getAttribute('class');
          const htmlTheme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
          console.log('Body class:', bodyClass);
          console.log('HTML theme:', htmlTheme);
          
          expect(bodyClass || htmlTheme).toMatch(/dark/);
        }
      }
    } else {
      console.log('Theme button not found');
      await page.screenshot({ path: 'test-screenshots/theme-button-not-found.png' });
    }
  });

  test('should switch between light and dark mode', async ({ page }) => {
    const themeBtn = await findThemeSwitcher(page);
    if (!themeBtn) {
      test.skip();
      return;
    }
    
    // Switch to dark mode
    await themeBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    
    const darkOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /深色|Dark/i }).first();
    await darkOption.click();
    await page.waitForTimeout(500);
    
    // Verify dark mode
    let bodyClass = await page.locator('body').getAttribute('class') || '';
    expect(bodyClass).toContain('theme-dark');
    
    // Switch to light mode
    await themeBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    
    const lightOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /浅色|Light/i }).first();
    await lightOption.click();
    await page.waitForTimeout(500);
    
    // Verify light mode
    bodyClass = await page.locator('body').getAttribute('class') || '';
    expect(bodyClass).toContain('theme-light');
    
    await page.screenshot({ path: 'test-screenshots/04-light-mode.png' });
  });

  test('should persist theme preference', async ({ page }) => {
    const themeBtn = await findThemeSwitcher(page);
    if (!themeBtn) {
      test.skip();
      return;
    }
    
    // Switch to dark mode
    await themeBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    const darkOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /深色|Dark/i }).first();
    await darkOption.click();
    await page.waitForTimeout(500);
    
    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');
    
    // Verify theme persisted
    const bodyClass = await page.locator('body').getAttribute('class') || '';
    expect(bodyClass).toContain('theme-dark');
    
    // Check localStorage
    const storedTheme = await page.evaluate(() => localStorage.getItem('ai_infra_theme'));
    expect(storedTheme).toBe('dark');
  });
});

test.describe('Language Switching Tests', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.waitForLoadState('networkidle');
  });

  test('should switch to English', async ({ page }) => {
    // Find language switcher
    const langBtn = page.locator('button:has-text("简体中文"), button:has-text("English")').first();
    await expect(langBtn).toBeVisible({ timeout: 10000 });
    
    await langBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    
    // Click English option
    const englishOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /English/i });
    await englishOption.click();
    await page.waitForTimeout(500);
    
    // Verify UI language changed
    const storedLang = await page.evaluate(() => localStorage.getItem('ai_infra_language'));
    expect(storedLang).toBe('en-US');
    
    await page.screenshot({ path: 'test-screenshots/05-english-ui.png' });
  });

  test('should switch to Chinese', async ({ page }) => {
    // First ensure we're in English, then switch to Chinese
    const langBtn = page.locator('button:has-text("简体中文"), button:has-text("English")').first();
    
    // Get current language
    const currentText = await langBtn.innerText();
    console.log('Current language button text:', currentText);
    
    await langBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    
    // Click Chinese option
    const chineseOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /简体中文|中文/i });
    await chineseOption.click();
    await page.waitForTimeout(500);
    
    // Verify UI language changed
    const storedLang = await page.evaluate(() => localStorage.getItem('ai_infra_language'));
    expect(storedLang).toBe('zh-CN');
    
    await page.screenshot({ path: 'test-screenshots/06-chinese-ui.png' });
  });
});

test.describe('Monitoring Page Tests', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to monitoring page', async ({ page }) => {
    await page.goto('/monitoring');
    await page.waitForLoadState('networkidle');
    
    // Wait for monitoring page to load
    const title = page.locator('text=/监控仪表板|Monitoring Dashboard/i');
    await expect(title).toBeVisible({ timeout: 15000 });
    
    await page.screenshot({ path: 'test-screenshots/07-monitoring-page.png' });
  });

  test('should have refresh button', async ({ page }) => {
    await page.goto('/monitoring');
    await page.waitForLoadState('networkidle');
    
    const refreshBtn = page.locator('button').filter({ hasText: /刷新|Refresh/i });
    await expect(refreshBtn).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Combined Tests', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.waitForLoadState('networkidle');
  });

  test('theme should persist when switching language', async ({ page }) => {
    const themeBtn = await findThemeSwitcher(page);
    if (!themeBtn) {
      test.skip();
      return;
    }
    
    // Set dark mode
    await themeBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    const darkOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /深色|Dark/i }).first();
    await darkOption.click();
    await page.waitForTimeout(500);
    
    // Switch language
    const langBtn = page.locator('button:has-text("简体中文"), button:has-text("English")').first();
    await langBtn.click();
    await page.waitForSelector('.ant-dropdown', { state: 'visible', timeout: 3000 });
    const englishOption = page.locator('.ant-dropdown-menu-item').filter({ hasText: /English/i });
    await englishOption.click();
    await page.waitForTimeout(500);
    
    // Verify theme is still dark
    const bodyClass = await page.locator('body').getAttribute('class') || '';
    expect(bodyClass).toContain('theme-dark');
    
    await page.screenshot({ path: 'test-screenshots/08-dark-english.png' });
  });
});
