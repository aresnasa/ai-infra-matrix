const { test, expect } = require('@playwright/test');

test.describe('Check where the 404 text comes from', () => {
  test('Find the exact location of 404 text', async ({ page }) => {
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

    console.log('\n=== 检查 404 文本的确切位置 ===\n');

    // Navigate to different Nightingale paths
    const pathsToCheck = [
      '',
      'targets',
      'metric/explorer',
      'dashboards'
    ];

    for (const path of pathsToCheck) {
      const testUrl = `http://192.168.0.200:8080/nightingale/${path}`;
      console.log(`\n测试路径: ${testUrl}`);
      
      await page.goto(testUrl, { 
        waitUntil: 'networkidle',
        timeout: 30000 
      });

      await page.waitForTimeout(2000);

      // Find elements containing "404"
      const elements404 = await page.locator('text=404').all();
      console.log(`  找到包含 "404" 的元素数量: ${elements404.length}`);
      
      for (let i = 0; i < Math.min(elements404.length, 3); i++) {
        try {
          const el = elements404[i];
          const tagName = await el.evaluate(node => node.tagName);
          const className = await el.evaluate(node => node.className).catch(() => '');
          const visible = await el.isVisible();
          const text = await el.innerText().catch(() => '');
          
          console.log(`  元素 ${i+1}:`);
          console.log(`    标签: ${tagName}`);
          console.log(`    类名: ${className}`);
          console.log(`    可见: ${visible}`);
          console.log(`    文本: ${text.substring(0, 100)}`);
        } catch (e) {
          console.log(`  元素 ${i+1}: 获取信息失败 - ${e.message}`);
        }
      }

      // Check if there's a visible error message or 404 page
      const visibleBodyText = await page.locator('body').evaluate(body => {
        // Get only visible text
        const range = document.createRange();
        range.selectNodeContents(body);
        return range.toString().substring(0, 500);
      });
      
      console.log(`  可见文本预览: ${visibleBodyText}`);
    }

    // Take screenshots of each page
    await page.goto('http://192.168.0.200:8080/nightingale/', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    await page.waitForTimeout(2000);
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-root.png',
      fullPage: false
    });

    await page.goto('http://192.168.0.200:8080/nightingale/dashboards', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    await page.waitForTimeout(2000);
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-dashboards.png',
      fullPage: false 
    });

    console.log(`\n截图已保存`);
  });
});
