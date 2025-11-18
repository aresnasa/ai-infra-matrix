const { test, expect } = require('@playwright/test');

test.describe('Analyze Nightingale 404 page structure', () => {
  test('Get full HTML structure around 404 element', async ({ page }) => {
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

    console.log('\n=== 分析 404 元素的 HTML 结构 ===\n');

    // Navigate to Nightingale root
    await page.goto('http://192.168.0.200:8080/nightingale/', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    // Find the 404 element and its parent structure
    const element404 = page.locator('.ant-result-title:has-text("404")').first();
    
    if (await element404.count() > 0) {
      // Get parent structure
      const parentHTML = await element404.evaluate(el => {
        let parent = el.parentElement;
        for (let i = 0; i < 3 && parent; i++) {
          parent = parent.parentElement;
        }
        return parent ? parent.outerHTML.substring(0, 2000) : 'No parent';
      });

      console.log('404 元素的父级HTML结构:');
      console.log(parentHTML);
      console.log('\n');

      // Check if the 404 is visible
      const isVisible = await element404.isVisible();
      const boundingBox = await element404.boundingBox().catch(() => null);
      
      console.log(`404 元素是否可见: ${isVisible}`);
      console.log(`404 元素位置: ${boundingBox ? `x=${boundingBox.x}, y=${boundingBox.y}, w=${boundingBox.width}, h=${boundingBox.height}` : '无'}`);

      // Check viewport size
      const viewportSize = page.viewportSize();
      console.log(`视口大小: ${viewportSize.width}x${viewportSize.height}`);

      // Check if there's content below/above
      const bodyHeight = await page.evaluate(() => document.body.scrollHeight);
      console.log(`页面总高度: ${bodyHeight}px`);
      
      // Take a full page screenshot
      await page.screenshot({ 
        path: 'test-screenshots/nightingale-404-analysis.png',
        fullPage: true 
      });
      console.log('\n完整页面截图: test-screenshots/nightingale-404-analysis.png');
    } else {
      console.log('未找到 404 元素');
    }
  });
});
