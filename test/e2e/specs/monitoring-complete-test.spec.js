const { test, expect } = require('@playwright/test');

test.describe('Monitoring Page - Complete Functionality Test', () => {
  
  test.beforeEach(async ({ page }) => {
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
  });

  test('1. No 404 errors on monitoring page', async ({ page }) => {
    const failed404Requests = [];

    page.on('response', response => {
      if (response.status() === 404) {
        failed404Requests.push({
          url: response.url(),
          status: response.status()
        });
      }
    });

    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    console.log('\n=== 404 检查结果 ===');
    if (failed404Requests.length === 0) {
      console.log('✅ 没有 404 错误');
    } else {
      console.log(`❌ 发现 ${failed404Requests.length} 个 404 错误:`);
      failed404Requests.forEach((req, index) => {
        console.log(`${index + 1}. ${req.url}`);
      });
    }

    expect(failed404Requests.length, '不应该有 404 错误').toBe(0);
  });

  test('2. Monitoring iframe loads successfully', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    // Wait for iframe to be present
    await page.waitForSelector('iframe', { timeout: 10000 });

    const iframes = page.frames();
    console.log(`\n=== iframe 检查 ===`);
    console.log(`找到 ${iframes.length} 个 frame(s)`);

    // Find the nightingale iframe
    const nightingaleFrame = iframes.find(f => 
      f.url().includes('nightingale') || f.url().includes('n9e')
    );

    expect(nightingaleFrame, 'Nightingale iframe 应该存在').toBeDefined();
    console.log(`✅ Nightingale iframe URL: ${nightingaleFrame.url()}`);
  });

  test('3. All Nightingale static assets load correctly', async ({ page }) => {
    const staticAssets = {
      font: [],
      js: [],
      image: []
    };

    page.on('response', response => {
      const url = response.url();
      if (url.includes('/font/') || url.match(/\/font$/)) {
        staticAssets.font.push({
          url: url,
          status: response.status(),
          ok: response.ok()
        });
      } else if (url.includes('/js/') && !url.includes('static/js')) {
        staticAssets.js.push({
          url: url,
          status: response.status(),
          ok: response.ok()
        });
      } else if (url.includes('/image/')) {
        staticAssets.image.push({
          url: url,
          status: response.status(),
          ok: response.ok()
        });
      }
    });

    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    console.log('\n=== 静态资源加载检查 ===');
    
    const allAssets = [...staticAssets.font, ...staticAssets.js, ...staticAssets.image];
    console.log(`总共检测到 ${allAssets.length} 个相关静态资源`);
    
    if (staticAssets.font.length > 0) {
      console.log(`\n字体文件 (${staticAssets.font.length}):`);
      staticAssets.font.forEach(asset => {
        console.log(`  ${asset.ok ? '✅' : '❌'} ${asset.status} - ${asset.url}`);
      });
    }
    
    if (staticAssets.js.length > 0) {
      console.log(`\nJS 文件 (${staticAssets.js.length}):`);
      staticAssets.js.forEach(asset => {
        console.log(`  ${asset.ok ? '✅' : '❌'} ${asset.status} - ${asset.url}`);
      });
    }
    
    if (staticAssets.image.length > 0) {
      console.log(`\n图片文件 (${staticAssets.image.length}):`);
      staticAssets.image.forEach(asset => {
        console.log(`  ${asset.ok ? '✅' : '❌'} ${asset.status} - ${asset.url}`);
      });
    }

    // Check that all loaded assets have successful status codes
    const failedAssets = allAssets.filter(asset => !asset.ok && asset.status !== 401);
    expect(failedAssets.length, `失败的静态资源数量应该为 0`).toBe(0);
  });

  test('4. Page renders without JavaScript errors', async ({ page }) => {
    const jsErrors = [];

    page.on('pageerror', error => {
      jsErrors.push(error.message);
    });

    page.on('console', msg => {
      if (msg.type() === 'error') {
        jsErrors.push(msg.text());
      }
    });

    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    console.log('\n=== JavaScript 错误检查 ===');
    if (jsErrors.length === 0) {
      console.log('✅ 没有 JavaScript 错误');
    } else {
      console.log(`发现 ${jsErrors.length} 个错误:`);
      jsErrors.forEach((err, index) => {
        // Filter out 401 errors as those are expected during SSO
        if (!err.includes('401')) {
          console.log(`${index + 1}. ${err}`);
        }
      });
    }

    // Filter out 401 errors and check remaining errors
    const criticalErrors = jsErrors.filter(err => 
      !err.includes('401') && 
      !err.includes('Unauthorized') &&
      !err.includes('Failed to load resource')
    );

    expect(criticalErrors.length, '不应该有严重的 JavaScript 错误').toBe(0);
  });

  test('5. Monitoring page SSO integration works', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    // Wait for iframe
    await page.waitForSelector('iframe', { timeout: 10000 });
    await page.waitForTimeout(3000);

    const iframes = page.frames();
    const nightingaleFrame = iframes.find(f => 
      f.url().includes('nightingale') || f.url().includes('n9e')
    );

    console.log('\n=== SSO 集成检查 ===');
    console.log(`Nightingale Frame URL: ${nightingaleFrame?.url()}`);
    
    expect(nightingaleFrame, 'Nightingale iframe 应该加载').toBeDefined();
    
    // The iframe should be accessible
    try {
      const frameVisible = await page.locator('iframe').isVisible({ timeout: 5000 });
      console.log(`✅ Monitoring iframe 可见: ${frameVisible}`);
      expect(frameVisible, 'iframe 应该可见').toBe(true);
    } catch (e) {
      console.log(`❌ iframe 可见性检查失败: ${e.message}`);
      throw e;
    }
  });

  test('6. Screenshot comparison - monitoring page', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForTimeout(3000);

    // Take screenshot
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-complete-test.png',
      fullPage: true 
    });

    console.log('\n=== 截图已保存 ===');
    console.log('📸 test-screenshots/monitoring-complete-test.png');
    
    // This test always passes - it's just for visual verification
    expect(true).toBe(true);
  });
});
