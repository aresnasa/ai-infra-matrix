/**
 * AI Infrastructure Matrix - Nightingale Monitoring Iframe 测试
 * 
 * 测试目标：
 * 1. MonitoringPage 组件加载
 * 2. Iframe 元素存在性
 * 3. Iframe src 配置正确
 * 4. Nightingale 服务可访问性
 * 5. X-Frame-Options 和 CSP 头检查
 * 6. 控制台错误诊断
 */

const { test, expect } = require('@playwright/test');

const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://192.168.18.114:8080',
  adminUser: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  },
  nightingalePort: process.env.REACT_APP_NIGHTINGALE_PORT || '17000',
};

async function login(page, username, password) {
  await page.goto('/');
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

async function waitForPageLoad(page) {
  await page.waitForLoadState('domcontentloaded', { timeout: 15000 });
  await page.waitForTimeout(2000);
  await page.waitForSelector('.ant-spin-spinning', { state: 'detached', timeout: 10000 }).catch(() => {});
}

test.describe('Nightingale Monitoring Iframe 诊断', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  });

  test('1. MonitoringPage 路由和权限检查', async ({ page }) => {
    console.log('测试 MonitoringPage 路由...');
    
    // 导航到监控页面
    await page.goto('/monitoring');
    await waitForPageLoad(page);
    
    // 检查是否被重定向到登录页或权限错误页
    const currentURL = page.url();
    console.log(`当前 URL: ${currentURL}`);
    
    if (currentURL.includes('login')) {
      console.log('❌ 被重定向到登录页，可能是认证问题');
      throw new Error('重定向到登录页');
    }
    
    if (currentURL.includes('403') || currentURL.includes('unauthorized')) {
      console.log('❌ 权限不足，需要 SRE 团队权限');
      throw new Error('权限不足');
    }
    
    // 检查是否有权限错误提示
    const permissionError = page.locator('text=/权限不足|Unauthorized|403/');
    const hasPermissionError = await permissionError.isVisible().catch(() => false);
    
    if (hasPermissionError) {
      const errorText = await permissionError.textContent();
      console.log(`❌ 权限错误: ${errorText}`);
      throw new Error('权限错误: ' + errorText);
    }
    
    console.log('✓ MonitoringPage 路由正常，权限验证通过');
  });

  test('2. Iframe 元素检查', async ({ page }) => {
    console.log('测试 Iframe 元素...');
    
    await page.goto('/monitoring');
    await waitForPageLoad(page);
    
    // 检查 iframe 是否存在
    const iframe = page.locator('iframe');
    const iframeCount = await iframe.count();
    
    console.log(`发现 ${iframeCount} 个 iframe 元素`);
    
    if (iframeCount === 0) {
      console.log('❌ 未找到 iframe 元素');
      
      // 截图保存
      await page.screenshot({ path: 'test-screenshots/nightingale-no-iframe.png' });
      console.log('已保存截图: test-screenshots/nightingale-no-iframe.png');
      
      throw new Error('未找到 iframe 元素');
    }
    
    // 获取 iframe 属性
    const iframeSrc = await iframe.getAttribute('src');
    console.log(`Iframe src: ${iframeSrc}`);
    
    // 验证 src 包含正确的端口
    if (iframeSrc && iframeSrc.includes(TEST_CONFIG.nightingalePort)) {
      console.log(`✓ Iframe src 包含正确端口: ${TEST_CONFIG.nightingalePort}`);
    } else {
      console.log(`⚠ Iframe src 端口可能不正确，期望: ${TEST_CONFIG.nightingalePort}`);
    }
    
    // 检查 iframe 的其他属性
    const sandbox = await iframe.getAttribute('sandbox');
    console.log(`Iframe sandbox: ${sandbox}`);
    
    const style = await iframe.getAttribute('style');
    console.log(`Iframe style: ${style}`);
    
    // 检查 iframe 是否可见
    const isVisible = await iframe.isVisible();
    console.log(`Iframe visible: ${isVisible}`);
    
    if (!isVisible) {
      console.log('⚠ Iframe 存在但不可见');
      await page.screenshot({ path: 'test-screenshots/nightingale-iframe-hidden.png' });
    }
    
    console.log('✓ Iframe 元素检查完成');
  });

  test('3. Nightingale 服务直接访问测试', async ({ page }) => {
    console.log('测试 Nightingale 服务直接访问...');
    
    // 构造 Nightingale URL
    const hostname = new URL(TEST_CONFIG.baseURL).hostname;
    const nightingaleURL = `http://${hostname}:${TEST_CONFIG.nightingalePort}`;
    
    console.log(`Nightingale URL: ${nightingaleURL}`);
    
    // 监听响应
    let response;
    page.on('response', async (resp) => {
      if (resp.url().includes(`:${TEST_CONFIG.nightingalePort}`)) {
        response = resp;
        console.log(`[响应] ${resp.status()} ${resp.url()}`);
        
        // 检查关键头信息
        const headers = resp.headers();
        if (headers['x-frame-options']) {
          console.log(`X-Frame-Options: ${headers['x-frame-options']}`);
        }
        if (headers['content-security-policy']) {
          console.log(`CSP: ${headers['content-security-policy']}`);
        }
      }
    });
    
    // 尝试直接访问 Nightingale
    try {
      await page.goto(nightingaleURL, { timeout: 10000, waitUntil: 'domcontentloaded' });
      console.log('✓ Nightingale 服务可直接访问');
      
      // 截图保存
      await page.screenshot({ path: 'test-screenshots/nightingale-direct-access.png' });
      console.log('已保存截图: test-screenshots/nightingale-direct-access.png');
      
      // 检查页面内容
      const bodyText = await page.locator('body').textContent();
      if (bodyText.includes('Nightingale') || bodyText.includes('n9e') || bodyText.includes('监控')) {
        console.log('✓ Nightingale 页面内容正常');
      } else {
        console.log('⚠ Nightingale 页面内容可能异常');
      }
      
    } catch (error) {
      console.log(`❌ 无法访问 Nightingale 服务: ${error.message}`);
      await page.screenshot({ path: 'test-screenshots/nightingale-access-failed.png' });
      throw error;
    }
  });

  test('4. 网络请求和响应头检查', async ({ page }) => {
    console.log('检查网络请求和响应头...');
    
    const nightingaleRequests = [];
    const failedRequests = [];
    
    // 监听所有请求
    page.on('request', request => {
      if (request.url().includes(`:${TEST_CONFIG.nightingalePort}`)) {
        console.log(`[请求] ${request.method()} ${request.url()}`);
        nightingaleRequests.push({
          url: request.url(),
          method: request.method()
        });
      }
    });
    
    // 监听响应
    page.on('response', async response => {
      if (response.url().includes(`:${TEST_CONFIG.nightingalePort}`)) {
        const status = response.status();
        const url = response.url();
        
        console.log(`[响应] ${status} ${url}`);
        
        // 检查关键响应头
        const headers = response.headers();
        
        // X-Frame-Options 检查
        if (headers['x-frame-options']) {
          console.log(`  ⚠ X-Frame-Options: ${headers['x-frame-options']}`);
          console.log(`  ⚠ 这个头会阻止 iframe 嵌入！需要配置 Nightingale 移除或设置为 ALLOWALL`);
        }
        
        // Content-Security-Policy 检查
        if (headers['content-security-policy']) {
          const csp = headers['content-security-policy'];
          console.log(`  CSP: ${csp}`);
          
          if (csp.includes('frame-ancestors')) {
            const frameAncestors = csp.match(/frame-ancestors[^;]+/)?.[0];
            console.log(`  ⚠ CSP frame-ancestors: ${frameAncestors}`);
            console.log(`  ⚠ 这个指令限制了 iframe 嵌入！需要配置允许来源`);
          }
        }
        
        // 记录失败请求
        if (status >= 400) {
          failedRequests.push({
            url,
            status,
            headers
          });
          console.log(`  ❌ 请求失败: ${status}`);
        }
      }
    });
    
    // 访问监控页面
    await page.goto('/monitoring');
    await waitForPageLoad(page);
    await page.waitForTimeout(5000); // 等待 iframe 加载
    
    console.log(`\n总共捕获 ${nightingaleRequests.length} 个 Nightingale 请求`);
    console.log(`失败请求数: ${failedRequests.length}`);
    
    if (failedRequests.length > 0) {
      console.log('\n失败请求详情:');
      failedRequests.forEach(({ url, status }) => {
        console.log(`  [${status}] ${url}`);
      });
    }
  });

  test('5. 控制台错误诊断', async ({ page }) => {
    console.log('诊断控制台错误...');
    
    const consoleMessages = {
      errors: [],
      warnings: [],
      logs: []
    };
    
    // 监听所有控制台消息
    page.on('console', msg => {
      const text = msg.text();
      const type = msg.type();
      
      // 只记录与 Nightingale、iframe 或监控相关的消息
      if (text.toLowerCase().includes('nightingale') || 
          text.toLowerCase().includes('iframe') ||
          text.toLowerCase().includes('monitoring') ||
          text.toLowerCase().includes('frame') ||
          text.toLowerCase().includes(':17000')) {
        
        if (type === 'error') {
          console.log(`❌ [控制台错误] ${text}`);
          consoleMessages.errors.push(text);
        } else if (type === 'warning') {
          console.log(`⚠ [控制台警告] ${text}`);
          consoleMessages.warnings.push(text);
        } else {
          console.log(`ℹ [控制台日志] ${text}`);
          consoleMessages.logs.push(text);
        }
      }
    });
    
    // 监听页面错误
    page.on('pageerror', error => {
      console.log(`❌ [页面错误] ${error.message}`);
      consoleMessages.errors.push(`Page Error: ${error.message}`);
    });
    
    // 访问监控页面
    await page.goto('/monitoring');
    await waitForPageLoad(page);
    await page.waitForTimeout(5000);
    
    // 截图保存
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-iframe-state.png',
      fullPage: true 
    });
    console.log('已保存完整截图: test-screenshots/nightingale-iframe-state.png');
    
    // 总结
    console.log('\n控制台消息总结:');
    console.log(`  错误: ${consoleMessages.errors.length}`);
    console.log(`  警告: ${consoleMessages.warnings.length}`);
    console.log(`  日志: ${consoleMessages.logs.length}`);
    
    // 如果有错误，详细列出
    if (consoleMessages.errors.length > 0) {
      console.log('\n详细错误列表:');
      consoleMessages.errors.forEach((error, index) => {
        console.log(`  ${index + 1}. ${error}`);
      });
    }
  });

  test('6. Iframe 内容访问测试', async ({ page }) => {
    console.log('测试 Iframe 内容访问...');
    
    await page.goto('/monitoring');
    await waitForPageLoad(page);
    
    // 等待 iframe 加载
    const iframe = page.locator('iframe');
    await iframe.waitFor({ state: 'attached', timeout: 10000 });
    console.log('✓ Iframe 已附加到 DOM');
    
    // 等待 iframe 加载完成
    await page.waitForTimeout(5000);
    
    try {
      // 尝试获取 iframe 的 frame
      const frameElement = await iframe.elementHandle();
      const frame = await frameElement.contentFrame();
      
      if (frame) {
        console.log('✓ 成功访问 iframe frame 对象');
        
        // 尝试获取 iframe 内的内容
        try {
          const frameURL = frame.url();
          console.log(`Iframe 当前 URL: ${frameURL}`);
          
          const frameTitle = await frame.title().catch(() => '无法获取');
          console.log(`Iframe 标题: ${frameTitle}`);
          
          // 尝试获取 iframe 内的 body 文本
          const bodyLocator = frame.locator('body');
          const bodyText = await bodyLocator.textContent({ timeout: 5000 }).catch(() => '无法获取');
          
          if (bodyText && bodyText.length > 0) {
            console.log(`✓ Iframe 内容长度: ${bodyText.length} 字符`);
            console.log(`Iframe 内容预览: ${bodyText.substring(0, 200)}...`);
          } else {
            console.log('⚠ Iframe 内容为空');
          }
          
        } catch (contentError) {
          console.log(`⚠ 无法访问 iframe 内容: ${contentError.message}`);
          console.log('这可能是由于跨域策略限制');
        }
        
      } else {
        console.log('❌ 无法获取 iframe frame 对象');
        console.log('可能原因:');
        console.log('  1. Iframe 被 X-Frame-Options 阻止');
        console.log('  2. CSP frame-ancestors 限制');
        console.log('  3. Iframe src 无法加载');
      }
      
    } catch (error) {
      console.log(`❌ Iframe 访问异常: ${error.message}`);
    }
    
    // 截图保存当前状态
    await page.screenshot({ path: 'test-screenshots/nightingale-iframe-final.png' });
    console.log('已保存最终截图: test-screenshots/nightingale-iframe-final.png');
  });

  test('7. 完整诊断报告', async ({ page }) => {
    console.log('\n=== Nightingale Iframe 完整诊断 ===\n');
    
    const diagnostics = {
      route: false,
      iframe: false,
      service: false,
      headers: {},
      errors: []
    };
    
    // 监听控制台和网络
    page.on('console', msg => {
      if (msg.type() === 'error') {
        diagnostics.errors.push(msg.text());
      }
    });
    
    page.on('response', async response => {
      if (response.url().includes(`:${TEST_CONFIG.nightingalePort}`)) {
        const headers = response.headers();
        diagnostics.headers = {
          'x-frame-options': headers['x-frame-options'] || 'none',
          'content-security-policy': headers['content-security-policy'] || 'none',
          status: response.status()
        };
      }
    });
    
    // 1. 路由检查
    try {
      await page.goto('/monitoring', { timeout: 15000 });
      await waitForPageLoad(page);
      diagnostics.route = true;
      console.log('✓ 1. 路由访问成功');
    } catch (error) {
      console.log(`❌ 1. 路由访问失败: ${error.message}`);
    }
    
    // 2. Iframe 检查
    try {
      const iframe = page.locator('iframe');
      const count = await iframe.count();
      diagnostics.iframe = count > 0;
      console.log(`${count > 0 ? '✓' : '❌'} 2. Iframe 元素${count > 0 ? '存在' : '不存在'} (数量: ${count})`);
      
      if (count > 0) {
        const src = await iframe.getAttribute('src');
        console.log(`   Iframe src: ${src}`);
      }
    } catch (error) {
      console.log(`❌ 2. Iframe 检查失败: ${error.message}`);
    }
    
    // 等待网络请求
    await page.waitForTimeout(5000);
    
    // 3. 服务检查
    diagnostics.service = diagnostics.headers.status >= 200 && diagnostics.headers.status < 400;
    console.log(`${diagnostics.service ? '✓' : '❌'} 3. Nightingale 服务${diagnostics.service ? '正常' : '异常'} (状态: ${diagnostics.headers.status || '未响应'})`);
    
    // 4. 响应头检查
    console.log('\n4. 响应头分析:');
    console.log(`   X-Frame-Options: ${diagnostics.headers['x-frame-options']}`);
    console.log(`   CSP: ${diagnostics.headers['content-security-policy']}`);
    
    if (diagnostics.headers['x-frame-options'] && diagnostics.headers['x-frame-options'] !== 'none') {
      console.log('   ⚠ X-Frame-Options 会阻止 iframe 嵌入！');
    }
    
    if (diagnostics.headers['content-security-policy'] && 
        diagnostics.headers['content-security-policy'].includes('frame-ancestors')) {
      console.log('   ⚠ CSP frame-ancestors 限制了 iframe 嵌入！');
    }
    
    // 5. 错误总结
    console.log(`\n5. 控制台错误: ${diagnostics.errors.length} 个`);
    if (diagnostics.errors.length > 0) {
      diagnostics.errors.forEach((error, i) => {
        console.log(`   ${i + 1}. ${error}`);
      });
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-diagnosis-complete.png',
      fullPage: true 
    });
    
    // 最终建议
    console.log('\n=== 诊断建议 ===');
    
    if (!diagnostics.route) {
      console.log('❌ 路由无法访问，检查前端路由配置和权限设置');
    }
    
    if (!diagnostics.iframe) {
      console.log('❌ Iframe 未渲染，检查 MonitoringPage 组件代码');
    }
    
    if (!diagnostics.service) {
      console.log('❌ Nightingale 服务无响应，检查 docker-compose 服务状态');
    }
    
    if (diagnostics.headers['x-frame-options'] && diagnostics.headers['x-frame-options'] !== 'none') {
      console.log('❌ 需要配置 Nightingale 移除 X-Frame-Options 头或设置为 ALLOWALL');
      console.log('   建议在 nginx 配置中添加: proxy_hide_header X-Frame-Options;');
    }
    
    if (diagnostics.headers['content-security-policy'] && 
        diagnostics.headers['content-security-policy'].includes('frame-ancestors')) {
      console.log('❌ 需要配置 CSP 允许 iframe 嵌入');
      console.log(`   建议添加: frame-ancestors 'self' http://${new URL(TEST_CONFIG.baseURL).hostname}:8080;`);
    }
    
    if (diagnostics.route && diagnostics.iframe && diagnostics.service && 
        !diagnostics.headers['x-frame-options'] && 
        !diagnostics.headers['content-security-policy']?.includes('frame-ancestors')) {
      console.log('✓ 所有检查通过，iframe 应该可以正常显示');
    }
    
    console.log('\n=== 诊断完成 ===\n');
  });
});
