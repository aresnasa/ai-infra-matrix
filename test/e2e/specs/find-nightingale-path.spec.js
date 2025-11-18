const { test, expect } = require('@playwright/test');

test.describe('Find working Nightingale default path', () => {
  test('Test different Nightingale paths', async ({ page }) => {
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

    // Test paths
    const pathsToTest = [
      '',
      'metrics',
      'explorer',
      'alert-rules',
      'dashboard-built-in',
      'targets',
      'infrastructure',
      'busi-groups'
    ];

    console.log('\n=== 测试 Nightingale 各个路径 ===\n');

    for (const path of pathsToTest) {
      const testUrl = `http://192.168.0.200:8080/nightingale/${path}`;
      console.log(`\n测试: ${testUrl}`);
      
      try {
        await page.goto(testUrl, { waitUntil: 'load', timeout: 10000 });
        await page.waitForTimeout(1500);

        const bodyText = await page.locator('body').innerText();
        const has404 = bodyText.includes('404') || bodyText.includes('页面不存在');
        
        if (has404) {
          console.log(`  ❌ 显示 404`);
        } else {
          console.log(`  ✅ 正常加载 - 这个路径可以使用!`);
          console.log(`  页面内容预览: ${bodyText.substring(0, 200)}`);
        }
      } catch (e) {
        console.log(`  ⚠️  加载出错: ${e.message}`);
      }
    }
  });
});
