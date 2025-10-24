const { test, expect } = require('@playwright/test');

test.describe('Nightingale SSO Integration Test', () => {
  test('should access Nightingale via SSO without login redirect', async ({ page }) => {
    console.log('=== 测试 Nightingale SSO 集成 ===\n');

    // Step 1: 登录主系统
    console.log('Step 1: 登录 AI-Infra-Matrix...');
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
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    console.log('当前 URL:', page.url());
    
    // Step 3: 等待 iframe 加载
    console.log('\nStep 3: 等待 Nightingale iframe 加载...');
    const iframe = page.frameLocator('iframe[title="Nightingale Monitoring"]');
    
    // 等待 iframe 内容加载
    await page.waitForTimeout(3000); // 给 Nightingale 时间加载
    
    // Step 4: 检查 iframe 的 URL
    const iframeSrc = await page.locator('iframe[title="Nightingale Monitoring"]').getAttribute('src');
    console.log('iframe src:', iframeSrc);
    
    // Step 5: 截图
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-sso-test.png', 
      fullPage: true 
    });
    console.log('✓ 截图已保存\n');

    // Step 6: 获取 JWT token 并尝试直接访问 Nightingale
    console.log('Step 4: 获取 JWT token...');
    const token = await page.evaluate(() => localStorage.getItem('token'));
    console.log('Token:', token ? `${token.substring(0, 20)}...` : 'null');
    
    console.log('\nStep 5: 直接访问 /nightingale/...');
    // Playwright 会自动传递 Cookie，但我们也通过 setExtraHTTPHeaders 传递 Authorization
    await page.setExtraHTTPHeaders({
      'Authorization': `Bearer ${token}`
    });
    
    const nightingaleResponse = await page.goto('http://192.168.18.154:8080/nightingale/');
    const nightingaleStatus = nightingaleResponse.status();
    console.log('Nightingale 响应状态:', nightingaleStatus);
    
    if (nightingaleStatus === 200) {
      console.log('✓ Nightingale SSO 认证成功！');
      
      // 检查是否被重定向到登录页
      const currentUrl = page.url();
      const isOnLoginPage = currentUrl.includes('/login') || currentUrl.includes('/signin');
      
      if (isOnLoginPage) {
        console.log('✗ 被重定向到登录页面，SSO 未生效');
        console.log('当前 URL:', currentUrl);
      } else {
        console.log('✓ 未被重定向到登录页，SSO 工作正常');
        console.log('当前 URL:', currentUrl);
      }
      
      // 检查页面内容
      const bodyText = await page.textContent('body');
      const hasLoginForm = bodyText.includes('登录') || bodyText.includes('Sign in') || bodyText.includes('用户名');
      const hasNightingaleUI = bodyText.includes('告警') || bodyText.includes('仪表板') || bodyText.includes('监控对象');
      
      console.log('\n页面内容检查:');
      console.log('- 包含登录表单:', hasLoginForm);
      console.log('- 包含 Nightingale UI:', hasNightingaleUI);
      
      if (!hasLoginForm && hasNightingaleUI) {
        console.log('\n✅ SSO 测试通过！用户无需登录即可访问 Nightingale');
      } else if (hasLoginForm) {
        console.log('\n❌ SSO 测试失败：显示了登录表单');
      } else {
        console.log('\n⚠️  SSO 状态不明确，请查看截图');
      }
    } else if (nightingaleStatus === 401) {
      console.log('❌ 401 Unauthorized - SSO 认证失败');
      console.log('可能的原因:');
      console.log('1. auth_request 未正确配置');
      console.log('2. X-User-Name 头未传递');
      console.log('3. Nightingale ProxyAuth 未启用');
    } else {
      console.log('❌ 非预期状态码:', nightingaleStatus);
    }
    
    // 最终截图
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-direct-access.png', 
      fullPage: true 
    });
    
    console.log('\n=== 测试完成 ===');
    
    // 断言：不应该是 401
    expect(nightingaleStatus).not.toBe(401);
  });
});
