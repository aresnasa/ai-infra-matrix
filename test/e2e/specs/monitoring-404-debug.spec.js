const { test, expect } = require('@playwright/test');

test.describe('Monitoring 404 Debug on 192.168.0.200:8080', () => {
  test('Check all network requests and find 404 errors', async ({ page }) => {
    const failed404Requests = [];
    const allRequests = [];

    // Listen to all requests
    page.on('request', request => {
      allRequests.push({
        url: request.url(),
        method: request.method(),
        resourceType: request.resourceType()
      });
    });

    // Listen to all responses
    page.on('response', response => {
      if (response.status() === 404) {
        failed404Requests.push({
          url: response.url(),
          status: response.status(),
          statusText: response.statusText()
        });
      }
    });

    // Listen to console messages
    const consoleLogs = [];
    page.on('console', msg => {
      consoleLogs.push({
        type: msg.type(),
        text: msg.text()
      });
    });

    // Navigate to login page first
    console.log('\n=== 步骤 1: 访问登录页面 ===');
    await page.goto('http://192.168.0.200:8080/login', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    // Wait for page to be ready
    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });

    // Login
    console.log('\n=== 步骤 2: 执行登录 ===');
    await page.locator('input[type="text"], input[name="username"]').first().fill('admin');
    await page.locator('input[type="password"], input[name="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').click();
    
    // Wait for navigation after login
    await page.waitForURL(/\/(projects|monitoring)/, { timeout: 10000 });
    console.log('登录后跳转到:', page.url());

    // Clear previous 404s
    failed404Requests.length = 0;

    // Navigate to monitoring page
    console.log('\n=== 步骤 3: 访问监控页面 ===');
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    // Wait a bit for all resources to load
    await page.waitForTimeout(3000);

    // Print all 404 errors
    console.log('\n=== 404 错误列表 ===');
    if (failed404Requests.length === 0) {
      console.log('✅ 没有发现 404 错误');
    } else {
      console.log(`❌ 发现 ${failed404Requests.length} 个 404 错误:`);
      failed404Requests.forEach((req, index) => {
        console.log(`${index + 1}. ${req.url}`);
        console.log(`   状态: ${req.status} ${req.statusText}`);
      });
    }

    // Print console errors
    console.log('\n=== 控制台错误 ===');
    const errors = consoleLogs.filter(log => log.type === 'error');
    if (errors.length === 0) {
      console.log('✅ 没有控制台错误');
    } else {
      console.log(`❌ 发现 ${errors.length} 个控制台错误:`);
      errors.forEach((err, index) => {
        console.log(`${index + 1}. ${err.text}`);
      });
    }

    // Take screenshot for debugging
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-404-debug.png',
      fullPage: true 
    });
    console.log('\n📸 截图已保存到: test-screenshots/monitoring-404-debug.png');

    // Check if iframe loaded
    const iframes = page.frames();
    console.log(`\n=== iframe 信息 ===`);
    console.log(`发现 ${iframes.length} 个 frame(s):`);
    iframes.forEach((frame, index) => {
      console.log(`${index + 1}. ${frame.url()}`);
    });

    // Get page content
    const content = await page.content();
    console.log('\n=== 页面标题 ===');
    console.log(await page.title());

    // Check if monitoring iframe exists
    const monitoringIframe = page.frameLocator('iframe[title*="监控"], iframe[src*="nightingale"], iframe[src*="n9e"]');
    try {
      const iframeVisible = await monitoringIframe.locator('body').isVisible({ timeout: 5000 });
      console.log('\n✅ 监控 iframe 已加载');
    } catch (e) {
      console.log('\n❌ 监控 iframe 未找到或加载失败');
    }

    // Analyze 404 patterns
    console.log('\n=== 404 分析 ===');
    const apiErrors = failed404Requests.filter(r => r.url.includes('/api/'));
    const staticErrors = failed404Requests.filter(r => 
      r.url.match(/\.(js|css|png|jpg|svg|woff|ttf)$/)
    );
    
    if (apiErrors.length > 0) {
      console.log(`API 404 错误 (${apiErrors.length}):`);
      apiErrors.forEach(err => console.log(`  - ${err.url}`));
    }
    
    if (staticErrors.length > 0) {
      console.log(`静态资源 404 错误 (${staticErrors.length}):`);
      staticErrors.forEach(err => console.log(`  - ${err.url}`));
    }

    // The test should fail if there are 404s so we can see the report
    expect(failed404Requests.length, `发现 ${failed404Requests.length} 个 404 错误`).toBe(0);
  });
});
