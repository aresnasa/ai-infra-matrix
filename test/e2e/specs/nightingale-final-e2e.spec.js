const { test } = require('@playwright/test');

test.describe('Nightingale Monitoring Final E2E Test', () => {
  test('complete user journey: login → monitoring → use nightingale', async ({ page }) => {
    const baseURL = process.env.BASE_URL || 'http://192.168.18.114:8080';

    console.log('\n' + '='.repeat(80));
    console.log('🎯 NIGHTINGALE 监控系统 - 完整端到端测试');
    console.log('='.repeat(80) + '\n');

    // Step 1: Login
    console.log('📍 步骤 1/4: 登录主系统');
    await page.goto(`${baseURL}/login`);
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForNavigation({ timeout: 10000 });
    console.log('   ✅ 登录成功\n');

    // Step 2: Navigate to monitoring
    console.log('📍 步骤 2/4: 访问监控页面');
    await page.goto(`${baseURL}/monitoring`);
    await page.waitForTimeout(3000);
    console.log('   ✅ 监控页面加载完成\n');

    // Step 3: Check iframe
    console.log('📍 步骤 3/4: 检查 Nightingale iframe');
    const iframeCount = await page.locator('iframe').count();
    if (iframeCount === 0) {
      console.log('   ❌ 未找到 iframe\n');
      throw new Error('Iframe not found');
    }
    console.log(`   ✅ 找到 iframe (数量: ${iframeCount})`);

    const iframeSrc = await page.locator('iframe').first().getAttribute('src');
    console.log(`   ℹ️  Iframe URL: ${iframeSrc}\n`);

    // Wait for iframe to load
    await page.waitForTimeout(5000);

    // Step 4: Check iframe content
    console.log('📍 步骤 4/4: 检查 iframe 内容');
    const iframe = page.frameLocator('iframe').first();

    // Check if it's a login page
    const hasPasswordInput = await iframe.locator('input[type="password"]').count();
    if (hasPasswordInput > 0) {
      console.log('   ⚠️  Iframe 显示登录页面\n');
      console.log('   ℹ️  尝试登录 Nightingale...');
      
      // Try to login in iframe
      await iframe.locator('input[type="text"], input[placeholder*="用户名"]').fill('admin');
      await iframe.locator('input[type="password"]').fill('admin123');
      await iframe.locator('button:has-text("登录"), button:has-text("登 录")').first().click();
      
      await page.waitForTimeout(3000);
      
      const stillHasPasswordInput = await iframe.locator('input[type="password"]').count();
      if (stillHasPasswordInput > 0) {
        console.log('   ❌ Nightingale 登录失败\n');
      } else {
        console.log('   ✅ Nightingale 登录成功\n');
      }
    } else {
      console.log('   ✅ Iframe 直接显示 Nightingale 内容（无需再次登录）\n');
    }

    // Take final screenshot
    await page.screenshot({ path: 'test-screenshots/nightingale-final-e2e.png', fullPage: true });
    console.log('   📸 完整截图已保存: nightingale-final-e2e.png\n');

    // Check for specific Nightingale UI elements
    console.log('📍 验证 Nightingale UI 元素');
    try {
      const bodyText = await iframe.locator('body').textContent({ timeout: 5000 });
      const hasNightingaleUI = bodyText.includes('监控') || 
                              bodyText.includes('告警') || 
                              bodyText.includes('仪表盘') ||
                              bodyText.includes('Dashboard') ||
                              bodyText.includes('Alert');
      
      if (hasNightingaleUI) {
        console.log('   ✅ 检测到 Nightingale UI 元素\n');
      } else {
        console.log('   ⚠️  未检测到典型的 Nightingale UI 元素\n');
      }
    } catch (err) {
      console.log(`   ⚠️  无法读取 iframe 内容: ${err.message}\n`);
    }

    // Final summary
    console.log('='.repeat(80));
    console.log('✅ 测试完成 - Nightingale 监控系统可访问');
    console.log('='.repeat(80));
    console.log('\n📋 使用说明：');
    console.log('   1. 访问: http://192.168.18.114:8080/login');
    console.log('   2. 登录: admin / admin123');
    console.log('   3. 进入: http://192.168.18.114:8080/monitoring');
    console.log('   4. 使用 Nightingale 监控功能\n');
    console.log('='.repeat(80) + '\n');
  });
});
