const { test, expect } = require('@playwright/test');

test.describe('Find Nightingale real SPA routes', () => {
  test('Navigate to Nightingale and check available routes', async ({ page }) => {
    // Login first
    await page.goto('http://192.168.0.200:8080/login', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });
    await page.locator('input[type="text"], input[name="username"]').first().fill('admin');
    await page.locator('input[type="password"], input[name="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').click();
    
    await page.waitForURL(/\/(projects|monitoring)/, { timeout: 10000 });

    console.log('\n=== 导航到 Nightingale 并检查可用路由 ===\n');

    // Load the main page
    await page.goto('http://192.168.0.200:8080/nightingale/', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    // Wait for page to render
    await page.waitForTimeout(3000);

    // Get current URL after SPA loads
    const currentUrl = page.url();
    console.log(`当前 URL: ${currentUrl}`);

    // Check for navigation menus
    const bodyText = await page.locator('body').innerText();
    console.log(`\n页面内容包含:`);
    console.log(`  - 是否有 "404": ${bodyText.includes('404')}`);
    console.log(`  - 是否有 "页面不存在": ${bodyText.includes('页面不存在')}`);
    console.log(`  - 是否有 "仪表板": ${bodyText.includes('仪表板')}`);
    console.log(`  - 是否有 "告警": ${bodyText.includes('告警')}`);
    console.log(`  - 是否有 "监控": ${bodyText.includes('监控')}`);

    // Try to find navigation links
    const links = await page.locator('a[href]').all();
    console.log(`\n找到 ${links.length} 个链接，检查相关路径:`);
    
    const relevantLinks = [];
    for (const link of links.slice(0, 50)) { // Check first 50 links
      try {
        const href = await link.getAttribute('href');
        if (href && (href.startsWith('/') || href.includes('nightingale'))) {
          const text = await link.innerText().catch(() => '');
          relevantLinks.push({ href, text: text.trim() });
        }
      } catch (e) {
        // Skip
      }
    }

    relevantLinks.forEach(({ href, text }) => {
      console.log(`  ${href} - "${text}"`);
    });

    // Take a screenshot
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-main-page.png',
      fullPage: true 
    });
    console.log(`\n截图已保存到: test-screenshots/nightingale-main-page.png`);

    // Get console logs
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log(`浏览器控制台错误: ${msg.text()}`);
      }
    });
  });
});
