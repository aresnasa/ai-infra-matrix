const { test, expect } = require('@playwright/test');

/**
 * SaltStack 状态页面完整诊断（含登录）
 * 测试目标: http://192.168.0.200:8080/saltstack
 */

test.describe('SaltStack 状态完整诊断', () => {
  test('登录后访问 SaltStack 页面并检查状态', async ({ page }) => {
    test.setTimeout(120000);
    
    // 监听控制台和错误
    page.on('console', msg => {
      if (msg.type() === 'error' || msg.type() === 'warning') {
        console.log(`[${msg.type()}]:`, msg.text());
      }
    });
    
    console.log('====================================');
    console.log('步骤 1: 登录系统');
    console.log('====================================');
    
    // 访问登录页
    await page.goto('http://192.168.0.200:8080/login', {
      waitUntil: 'networkidle'
    });
    
    // 填写登录表单
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    
    console.log('✓ 填写登录信息: admin / admin123');
    
    // 点击登录按钮
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    
    console.log('✓ 已点击登录按钮');
    
    // 检查是否登录成功（URL 改变或有 token）
    const currentUrl = page.url();
    console.log(`当前 URL: ${currentUrl}`);
    
    // ========================================
    // 步骤 2: 访问 SaltStack 页面
    // ========================================
    console.log('\n====================================');
    console.log('步骤 2: 访问 SaltStack 页面');
    console.log('====================================');
    
    const apiRequests = [];
    const apiResponses = [];
    
    // 监听 API 请求
    page.on('request', request => {
      if (request.url().includes('/api/')) {
        apiRequests.push({
          url: request.url(),
          method: request.method()
        });
        console.log(`  → ${request.method()} ${request.url()}`);
      }
    });
    
    page.on('response', async response => {
      if (response.url().includes('/api/')) {
        const status = response.status();
        let body = null;
        try {
          const contentType = response.headers()['content-type'] || '';
          if (contentType.includes('application/json')) {
            body = await response.json();
          } else {
            body = await response.text();
          }
        } catch (e) {
          body = '[无法读取]';
        }
        
        apiResponses.push({
          url: response.url(),
          status: status,
          body: body
        });
        
        console.log(`  ← ${status} ${response.url()}`);
        if (body && typeof body === 'object') {
          console.log(`     ${JSON.stringify(body).substring(0, 200)}`);
        } else if (body && body.length < 200) {
          console.log(`     ${body}`);
        }
      }
    });
    
    await page.goto('http://192.168.0.200:8080/saltstack', {
      waitUntil: 'networkidle'
    });
    
    console.log('✓ SaltStack 页面加载完成');
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-logged-in.png',
      fullPage: true 
    });
    console.log('✓ 截图: saltstack-logged-in.png');
    
    // ========================================
    // 步骤 3: 检查页面内容
    // ========================================
    console.log('\n====================================');
    console.log('步骤 3: 检查页面内容');
    console.log('====================================');
    
    // 检查标题
    const hasSaltStackTitle = await page.locator('text=SaltStack').first().isVisible({ timeout: 2000 }).catch(() => false);
    console.log(`SaltStack 标题: ${hasSaltStackTitle ? '✓ 可见' : '✗ 不可见'}`);
    
    // 检查关键元素
    const keywords = [
      'Master 状态',
      'Minion',
      '在线',
      '离线',
      '集群',
      '状态',
      '未知',
      'Unknown',
      'Running',
      'Connected'
    ];
    
    console.log('\n查找关键词:');
    for (const keyword of keywords) {
      const found = await page.locator(`text=${keyword}`).first().isVisible({ timeout: 1000 }).catch(() => false);
      console.log(`  ${found ? '✓' : '✗'} "${keyword}"`);
    }
    
    // ========================================
    // 步骤 4: 分析 API 调用
    // ========================================
    console.log('\n====================================');
    console.log('步骤 4: API 调用分析');
    console.log('====================================');
    
    console.log(`\n总请求数: ${apiRequests.length}`);
    console.log(`总响应数: ${apiResponses.length}`);
    
    // 查找 SaltStack 相关的 API
    const saltAPIs = apiResponses.filter(res => 
      res.url.includes('salt') || res.url.includes('slurm')
    );
    
    console.log(`\nSaltStack/Slurm 相关 API (${saltAPIs.length} 个):`);
    saltAPIs.forEach(api => {
      console.log(`\n  ${api.status} ${api.url}`);
      if (api.body && typeof api.body === 'object') {
        console.log(`  响应: ${JSON.stringify(api.body, null, 2)}`);
      } else {
        console.log(`  响应: ${api.body}`);
      }
    });
    
    // ========================================
    // 步骤 5: 直接测试 API
    // ========================================
    console.log('\n====================================');
    console.log('步骤 5: 直接测试后端 API');
    console.log('====================================');
    
    const testAPIs = [
      '/api/slurm/saltstack/status',
      '/api/slurm/status',
      '/api/saltstack/status',
      '/api/saltstack/minions'
    ];
    
    for (const apiPath of testAPIs) {
      const fullUrl = `http://192.168.0.200:8080${apiPath}`;
      try {
        console.log(`\n测试: ${apiPath}`);
        const response = await page.request.get(fullUrl);
        const status = response.status();
        
        let body;
        try {
          body = await response.json();
        } catch {
          body = await response.text();
        }
        
        console.log(`  状态: ${status}`);
        if (typeof body === 'object') {
          console.log(`  响应: ${JSON.stringify(body, null, 2)}`);
        } else {
          console.log(`  响应: ${body}`);
        }
      } catch (error) {
        console.log(`  ✗ 失败: ${error.message}`);
      }
    }
    
    // ========================================
    // 步骤 6: 检查路由定义
    // ========================================
    console.log('\n====================================');
    console.log('步骤 6: 查看页面路由信息');
    console.log('====================================');
    
    // 获取页面中的所有链接
    const links = await page.locator('a[href*="salt"]').all();
    console.log(`\n找到 ${links.length} 个包含 'salt' 的链接:`);
    for (const link of links) {
      const href = await link.getAttribute('href');
      const text = await link.textContent();
      console.log(`  - ${href}: ${text?.substring(0, 50)}`);
    }
    
    // ========================================
    // 总结
    // ========================================
    console.log('\n====================================');
    console.log('诊断总结');
    console.log('====================================');
    
    console.log('\n【问题分析】');
    
    // 分析 404 错误
    const has404 = apiResponses.some(res => res.status === 404);
    if (has404) {
      console.log('✗ 发现 404 错误 - API 路由可能不存在');
      const errors404 = apiResponses.filter(res => res.status === 404);
      errors404.forEach(err => {
        console.log(`  - ${err.url}`);
      });
    }
    
    // 分析 401 错误
    const has401 = apiResponses.some(res => res.status === 401);
    if (has401) {
      console.log('✗ 发现 401 错误 - 认证问题');
    }
    
    // 分析成功的请求
    const success = apiResponses.filter(res => res.status === 200);
    if (success.length > 0) {
      console.log(`✓ ${success.length} 个成功请求`);
    }
    
    console.log('\n【建议】');
    console.log('1. 检查后端路由配置，确认 SaltStack API 路径');
    console.log('2. 检查 Salt Master 服务是否正常运行');
    console.log('3. 验证环境变量配置（SALTSTACK_MASTER_URL 等）');
    console.log('4. 查看后端日志获取详细错误信息');
  });
});
