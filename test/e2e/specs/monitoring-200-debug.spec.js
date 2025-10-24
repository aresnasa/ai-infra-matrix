const { test, expect } = require('@playwright/test');

test.describe('Monitoring Page Debug (192.168.0.200)', () => {
  test('should diagnose monitoring page issue', async ({ page }) => {
    console.log('=== 诊断监控页面问题 (192.168.0.200:8080) ===\n');

    // 监听所有错误
    const consoleErrors = [];
    const networkErrors = [];
    const networkRequests = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
        console.log('❌ Console Error:', msg.text());
      } else if (msg.type() === 'log') {
        console.log('📝 Console Log:', msg.text());
      }
    });
    
    page.on('pageerror', error => {
      consoleErrors.push(error.message);
      console.log('❌ Page Error:', error.message);
      console.log('Stack:', error.stack);
    });
    
    page.on('requestfailed', request => {
      networkErrors.push(`${request.url()} - ${request.failure().errorText}`);
      console.log('❌ Network Failed:', request.url(), '-', request.failure().errorText);
    });

    page.on('response', response => {
      const url = response.url();
      const status = response.status();
      if (status >= 400 || url.includes('/api/') || url.includes('/nightingale/')) {
        networkRequests.push({ url, status });
        console.log(`📡 ${status} ${url}`);
      }
    });

    // Step 1: 登录
    console.log('\nStep 1: 登录系统...');
    await page.goto('http://192.168.0.200:8080/login');
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // 等待登录完成
    await page.waitForTimeout(2000);
    const loginUrl = page.url();
    console.log('登录后 URL:', loginUrl);
    
    // 检查是否登录成功
    const token = await page.evaluate(() => localStorage.getItem('token'));
    if (token) {
      console.log('✅ 登录成功，Token 存在');
    } else {
      console.log('❌ 登录失败，Token 不存在');
      await page.screenshot({ path: 'test-screenshots/login-failed.png' });
      return;
    }

    // Step 2: 访问监控页面
    console.log('\nStep 2: 访问 /monitoring...');
    await page.goto('http://192.168.0.200:8080/monitoring');
    
    // 等待页面加载
    await page.waitForTimeout(5000);
    
    const currentUrl = page.url();
    console.log('当前 URL:', currentUrl);
    
    // 检查是否被重定向
    if (currentUrl !== 'http://192.168.0.200:8080/monitoring') {
      console.log('⚠️  页面被重定向到:', currentUrl);
    }
    
    // 检查页面内容
    const bodyText = await page.textContent('body');
    console.log('\n页面内容 (前300字符):', bodyText.substring(0, 300).replace(/\s+/g, ' '));
    
    // 检查是否有 React root
    const hasReactRoot = await page.locator('#root').count();
    console.log('\n#root 元素:', hasReactRoot > 0 ? '✅ 存在' : '❌ 不存在');
    
    if (hasReactRoot > 0) {
      const rootContent = await page.locator('#root').innerHTML();
      console.log('#root 内容长度:', rootContent.length);
      
      if (rootContent.length < 100) {
        console.log('⚠️  #root 内容太少，可能是白屏');
        console.log('#root 内容:', rootContent);
      }
    }
    
    // 检查是否有错误提示
    const hasErrorAlert = await page.locator('.ant-alert-error').count();
    if (hasErrorAlert > 0) {
      const errorText = await page.locator('.ant-alert-error').textContent();
      console.log('❌ 错误提示:', errorText);
    }
    
    // 检查是否有 iframe
    const iframeCount = await page.locator('iframe').count();
    console.log('\niframe 数量:', iframeCount);
    
    if (iframeCount > 0) {
      const iframeSrc = await page.locator('iframe').first().getAttribute('src');
      console.log('iframe src:', iframeSrc);
      
      // 检查 iframe 是否加载
      const iframeVisible = await page.locator('iframe').first().isVisible();
      console.log('iframe 可见:', iframeVisible);
    } else {
      console.log('❌ 未找到 iframe，Nightingale 未加载');
    }
    
    // 检查监控卡片
    const cardTitle = await page.locator('.ant-card-head-title').textContent().catch(() => '');
    if (cardTitle) {
      console.log('\n卡片标题:', cardTitle);
    }
    
    // 汇总错误
    console.log('\n=== 错误汇总 ===');
    console.log('控制台错误数:', consoleErrors.length);
    if (consoleErrors.length > 0) {
      console.log('前5个错误:');
      consoleErrors.slice(0, 5).forEach((err, i) => {
        console.log(`  ${i + 1}. ${err.substring(0, 150)}`);
      });
    }
    
    console.log('\n网络错误数:', networkErrors.length);
    if (networkErrors.length > 0) {
      console.log('所有网络错误:');
      networkErrors.forEach((err, i) => {
        console.log(`  ${i + 1}. ${err}`);
      });
    }
    
    console.log('\n关键请求状态:');
    const importantUrls = networkRequests.filter(r => 
      r.url.includes('/navigation/config') || 
      r.url.includes('/nightingale/') ||
      r.url.includes('/auth/verify')
    );
    importantUrls.forEach(r => {
      const status = r.status >= 200 && r.status < 300 ? '✅' : 
                     r.status === 404 ? '⚠️' : '❌';
      console.log(`  ${status} ${r.status} ${r.url}`);
    });
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-200-debug.png', 
      fullPage: true 
    });
    console.log('\n✅ 截图保存: test-screenshots/monitoring-200-debug.png');
    
    // 如果有 iframe，尝试访问 iframe 内容
    if (iframeCount > 0) {
      console.log('\n=== 检查 iframe 内容 ===');
      try {
        const iframe = page.frameLocator('iframe').first();
        await page.waitForTimeout(2000);
        
        // 检查 iframe 是否显示登录页面
        const iframeBodyText = await iframe.locator('body').textContent().catch(() => '');
        if (iframeBodyText.includes('登录') || iframeBodyText.includes('Sign in')) {
          console.log('❌ iframe 显示登录页面，SSO 未生效');
        } else if (iframeBodyText.includes('告警') || iframeBodyText.includes('仪表板')) {
          console.log('✅ iframe 显示 Nightingale 仪表板');
        } else {
          console.log('⚠️  iframe 内容未知:', iframeBodyText.substring(0, 100));
        }
      } catch (error) {
        console.log('❌ 无法访问 iframe 内容:', error.message);
      }
    }
    
    console.log('\n=== 诊断完成 ===');
  });
});
