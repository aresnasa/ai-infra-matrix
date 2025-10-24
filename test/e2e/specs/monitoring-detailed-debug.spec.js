const { test, expect } = require('@playwright/test');

test.describe('Monitoring Page Detailed Debug', () => {
  test('debug monitoring page content in detail', async ({ page }) => {
    console.log('=== 开始详细调试监控页面 ===\n');

    // Step 1: 登录
    console.log('Step 1: 登录...');
    await page.goto('http://192.168.18.154:8080/login');
    await page.waitForLoadState('networkidle');
    
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    
    await page.waitForURL('http://192.168.18.154:8080/projects', { timeout: 10000 });
    console.log('✓ 登录成功\n');

    // Step 2: 导航到 monitoring
    console.log('Step 2: 导航到 /monitoring...');
    await page.goto('http://192.168.18.154:8080/monitoring');
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    const currentUrl = page.url();
    console.log('当前 URL:', currentUrl);
    
    // Step 3: 截图
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-detailed-debug.png', 
      fullPage: true 
    });
    console.log('✓ 已保存截图到 test-screenshots/monitoring-detailed-debug.png\n');

    // Step 4: 检查页面标题
    const pageTitle = await page.textContent('h3, h2, h1, .ant-card-head-title').catch(() => '未找到标题');
    console.log('页面标题:', pageTitle);

    // Step 5: 检查是否有 iframe
    const iframeCount = await page.locator('iframe').count();
    console.log('iframe 数量:', iframeCount);
    
    if (iframeCount > 0) {
      const iframeSrc = await page.locator('iframe').first().getAttribute('src');
      const iframeTitle = await page.locator('iframe').first().getAttribute('title');
      console.log('iframe src:', iframeSrc);
      console.log('iframe title:', iframeTitle);
    }

    // Step 6: 检查页面上的所有主要文本内容
    const bodyText = await page.locator('body').textContent();
    console.log('\n页面主要内容关键词检查:');
    console.log('- 包含"监控":', bodyText.includes('监控'));
    console.log('- 包含"Nightingale":', bodyText.includes('Nightingale'));
    console.log('- 包含"项目":', bodyText.includes('项目'));
    console.log('- 包含"403":', bodyText.includes('403'));
    console.log('- 包含"权限":', bodyText.includes('权限'));
    console.log('- 包含"管理员":', bodyText.includes('管理员'));

    // Step 7: 检查是否显示了错误结果页
    const has403 = await page.locator('text=403').count() > 0;
    const hasPermissionDenied = await page.locator('text=权限不足').count() > 0;
    const hasAccessDenied = await page.locator('text=访问被拒绝').count() > 0;
    
    console.log('\n错误页面检查:');
    console.log('- 403 错误:', has403);
    console.log('- 权限不足:', hasPermissionDenied);
    console.log('- 访问被拒绝:', hasAccessDenied);

    // Step 8: 检查主要 DOM 结构
    const hasMonitoringCard = await page.locator('.ant-card').locator('text=监控仪表板').count() > 0;
    const hasProjectCard = await page.locator('.ant-card').locator('text=项目').count() > 0;
    
    console.log('\nDOM 结构检查:');
    console.log('- 有监控仪表板卡片:', hasMonitoringCard);
    console.log('- 有项目卡片:', hasProjectCard);

    // Step 9: 获取所有卡片标题
    const cardTitles = await page.locator('.ant-card-head-title').allTextContents();
    console.log('\n所有卡片标题:', cardTitles);

    // Step 10: 检查 localStorage
    const localStorageData = await page.evaluate(() => {
      return {
        token: localStorage.getItem('token') ? '存在' : '不存在',
        user: localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')) : null,
      };
    });
    
    console.log('\nlocalStorage 数据:');
    console.log('- token:', localStorageData.token);
    console.log('- user:', JSON.stringify(localStorageData.user, null, 2));

    // Step 11: 输出最终诊断
    console.log('\n=== 诊断总结 ===');
    if (currentUrl.includes('/monitoring')) {
      console.log('✓ URL 正确 (/monitoring)');
    } else {
      console.log('✗ URL 错误:', currentUrl);
    }
    
    if (iframeCount > 0) {
      console.log('✓ 找到 iframe，Nightingale 应该已加载');
    } else {
      console.log('✗ 未找到 iframe，这是问题所在！');
      
      if (has403 || hasPermissionDenied || hasAccessDenied) {
        console.log('  原因: 权限不足 (403)');
        console.log('  用户角色:', localStorageData.user?.role);
        console.log('  用户权限组:', localStorageData.user?.roles);
      } else if (hasProjectCard) {
        console.log('  原因: 显示了项目页面而不是监控页面');
      } else {
        console.log('  原因: 未知 - 请查看截图');
      }
    }
    
    console.log('\n=== 调试完成 ===');
  });
});
