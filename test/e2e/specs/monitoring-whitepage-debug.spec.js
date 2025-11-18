const { test, expect } = require('@playwright/test');

test.describe('Monitoring White Page Debug', () => {
  test('should diagnose white page issue', async ({ page }) => {
    console.log('=== 诊断监控页面白屏问题 ===\n');

    // 监听控制台错误
    const consoleErrors = [];
    const networkErrors = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
        console.log('Console Error:', msg.text());
      }
    });
    
    page.on('pageerror', error => {
      consoleErrors.push(error.message);
      console.log('Page Error:', error.message);
    });
    
    page.on('requestfailed', request => {
      networkErrors.push(`${request.url()} - ${request.failure().errorText}`);
      console.log('Network Error:', request.url(), '-', request.failure().errorText);
    });

    // Step 1: 登录
    console.log('Step 1: 登录系统...');
    await page.goto('http://192.168.18.154:8080/login');
    await page.waitForLoadState('networkidle');
    
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    await page.click('button[type="submit"]');
    
    await page.waitForURL('http://192.168.18.154:8080/projects', { timeout: 10000 });
    console.log('✓ 登录成功\n');

    // Step 2: 访问监控页面
    console.log('Step 2: 访问 /monitoring...');
    await page.goto('http://192.168.18.154:8080/monitoring');
    
    // 等待一段时间让页面加载
    await page.waitForTimeout(3000);
    
    const currentUrl = page.url();
    console.log('当前 URL:', currentUrl);
    
    // 检查页面内容
    const bodyHTML = await page.content();
    console.log('\n页面 HTML 长度:', bodyHTML.length);
    
    const bodyText = await page.textContent('body');
    console.log('页面文本内容 (前200字符):', bodyText.substring(0, 200));
    
    // 检查是否有 React root
    const hasReactRoot = await page.locator('#root').count();
    console.log('\n#root 元素存在:', hasReactRoot > 0);
    
    if (hasReactRoot > 0) {
      const rootContent = await page.locator('#root').innerHTML();
      console.log('#root 内容长度:', rootContent.length);
      console.log('#root 内容 (前200字符):', rootContent.substring(0, 200));
    }
    
    // 检查是否加载了 React 应用
    const hasReactApp = await page.evaluate(() => {
      return window.React !== undefined || document.querySelector('[data-reactroot]') !== null;
    });
    console.log('React 应用已加载:', hasReactApp);
    
    // 检查路由信息
    const routeInfo = await page.evaluate(() => {
      return {
        pathname: window.location.pathname,
        hash: window.location.hash,
        search: window.location.search
      };
    });
    console.log('\n路由信息:', routeInfo);
    
    // 检查 localStorage
    const storage = await page.evaluate(() => {
      return {
        token: localStorage.getItem('token') !== null,
        user: localStorage.getItem('user') !== null
      };
    });
    console.log('localStorage:', storage);
    
    // 检查控制台错误
    console.log('\n控制台错误数量:', consoleErrors.length);
    if (consoleErrors.length > 0) {
      console.log('错误详情:');
      consoleErrors.forEach((err, i) => console.log(`  ${i + 1}. ${err}`));
    }
    
    // 检查网络错误
    console.log('\n网络错误数量:', networkErrors.length);
    if (networkErrors.length > 0) {
      console.log('错误详情:');
      networkErrors.forEach((err, i) => console.log(`  ${i + 1}. ${err}`));
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-whitepage.png', 
      fullPage: true 
    });
    console.log('\n✓ 截图已保存到 test-screenshots/monitoring-whitepage.png');
    
    console.log('\n=== 诊断完成 ===');
  });
});
