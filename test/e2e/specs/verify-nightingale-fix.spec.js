const { test, expect } = require('@playwright/test');

test.describe('Final Nightingale 404 Fix Verification', () => {
  test('Verify Nightingale monitoring page loads without 404', async ({ page }) => {
    console.log('\n=== 验证监控页面 404 已修复 ===\n');

    // Login
    await page.goto('http://192.168.0.200:8080/login', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });
    await page.locator('input[type="text"], input[name="username"]').first().fill('admin');
    await page.locator('input[type="password"], input[name="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').click();
    
    await page.waitForURL(/\/(projects|monitoring)/, { timeout: 10000 });
    console.log('✓ 登录成功');

    // Navigate to monitoring page
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });
    console.log('✓ 导航到监控页面');

    // Wait for iframe to load
    await page.waitForSelector('iframe[title="Nightingale Monitoring"]', { timeout: 10000 });
    console.log('✓ Nightingale iframe 已加载');

    await page.waitForTimeout(5000);

    // Get iframe
    const iframe = page.frameLocator('iframe[title="Nightingale Monitoring"]');

    // Check for 404 in iframe
    const iframeBody = iframe.locator('body');
    const bodyText = await iframeBody.innerText({ timeout: 10000 }).catch(() => '');
    
    const has404 = bodyText.includes('404') || bodyText.toLowerCase().includes('page not found') || bodyText.includes('页面不存在');
    
    console.log(`\niframe 内容检查:`);
    console.log(`  - 包含 "404": ${bodyText.includes('404')}`);
    console.log(`  - 包含 "page not found": ${bodyText.toLowerCase().includes('page not found')}`);
    console.log(`  - 包含 "页面不存在": ${bodyText.includes('页面不存在')}`);

    // Check if the 404 element is visible in the center
    const result404 = iframe.locator('.ant-result-title:has-text("404")');
    const has404Element = await result404.count() > 0;
    
    if (has404Element) {
      const isVisible = await result404.isVisible().catch(() => false);
      console.log(`  - .ant-result-title 404 元素存在: ${has404Element}, 可见: ${isVisible}`);
      
      if (isVisible) {
        console.log('\n❌ 错误：监控页面仍然显示 404 错误页面');
        expect(isVisible).toBe(false);
      }
    } else {
      console.log(`  - .ant-result-title 404 元素: 不存在 ✓`);
    }

    // Check for successful Nightingale content
    const hasTargets = bodyText.toLowerCase().includes('target') || bodyText.includes('业务组');
    const hasMetrics = bodyText.toLowerCase().includes('metric') || bodyText.includes('指标');
    const hasDashboard = bodyText.toLowerCase().includes('dashboard') || bodyText.includes('仪表板');
    const hasMonitoring = bodyText.toLowerCase().includes('monitoring') || bodyText.includes('监控');

    console.log(`\nNightingale 内容检查:`);
    console.log(`  - Targets/业务组: ${hasTargets ? '✓' : '✗'}`);
    console.log(`  - Metrics/指标: ${hasMetrics ? '✓' : '✗'}`);
    console.log(`  - Dashboard/仪表板: ${hasDashboard ? '✓' : '✗'}`);
    console.log(`  - Monitoring/监控: ${hasMonitoring ? '✓' : '✗'}`);

    // Take screenshots
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-page-final-verification.png',
      fullPage: false
    });
    console.log(`\n截图已保存: test-screenshots/monitoring-page-final-verification.png`);

    // Final assertion
    const contentLoaded = hasTargets || hasMetrics || hasDashboard || hasMonitoring;
    console.log(`\n最终结果: ${contentLoaded && !has404Element ? '✅ 修复成功' : '❌ 仍有问题'}`);
    
    expect(contentLoaded).toBe(true);
    if (has404Element) {
      const isVisible = await result404.isVisible().catch(() => false);
      expect(isVisible).toBe(false);
    }
  });
});
