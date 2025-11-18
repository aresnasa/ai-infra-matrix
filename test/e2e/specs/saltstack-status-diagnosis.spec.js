const { test, expect } = require('@playwright/test');

/**
 * SaltStack 状态诊断测试
 * 测试目标: http://192.168.0.200:8080/saltstack
 * 诊断问题: 无法获取 SaltStack 集群状态
 */

test.describe('SaltStack 状态页面诊断', () => {
  test.beforeEach(async ({ page }) => {
    // 设置较长的超时时间
    test.setTimeout(120000);
    
    // 监听控制台消息
    page.on('console', msg => {
      console.log(`[浏览器控制台 ${msg.type()}]:`, msg.text());
    });
    
    // 监听页面错误
    page.on('pageerror', error => {
      console.error('[页面错误]:', error.message);
    });
    
    // 监听网络请求失败
    page.on('requestfailed', request => {
      console.error('[请求失败]:', request.url(), request.failure().errorText);
    });
  });

  test('访问 SaltStack 页面并诊断状态显示问题', async ({ page }) => {
    console.log('====================================');
    console.log('开始访问 SaltStack 页面...');
    console.log('====================================');
    
    // 访问页面
    await page.goto('http://192.168.0.200:8080/saltstack', {
      waitUntil: 'networkidle',
      timeout: 30000
    });
    
    console.log('\n✓ 页面加载完成');
    
    // 等待页面渲染
    await page.waitForTimeout(2000);
    
    // 截图1：页面加载后
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-status-01-loaded.png',
      fullPage: true 
    });
    console.log('✓ 截图保存: saltstack-status-01-loaded.png');
    
    // ========================================
    // 1. 检查页面标题和基本结构
    // ========================================
    console.log('\n====================================');
    console.log('1. 检查页面基本结构');
    console.log('====================================');
    
    const pageTitle = await page.title();
    console.log(`页面标题: ${pageTitle}`);
    
    // 检查 SaltStack 标题是否存在
    const titleExists = await page.locator('text=SaltStack').first().isVisible({ timeout: 5000 }).catch(() => false);
    console.log(`SaltStack 标题可见: ${titleExists}`);
    
    // ========================================
    // 2. 检查状态卡片
    // ========================================
    console.log('\n====================================');
    console.log('2. 检查 SaltStack 状态卡片');
    console.log('====================================');
    
    // 查找状态相关的文本
    const statusTexts = [
      'Master 状态',
      'Minion 总数',
      '在线 Minions',
      '离线 Minions',
      '集群状态',
      'Salt Master',
      '状态',
      '未知'
    ];
    
    for (const text of statusTexts) {
      const exists = await page.locator(`text=${text}`).first().isVisible({ timeout: 2000 }).catch(() => false);
      console.log(`  - "${text}": ${exists ? '✓ 存在' : '✗ 不存在'}`);
      
      if (exists) {
        const element = page.locator(`text=${text}`).first();
        const textContent = await element.textContent();
        console.log(`    内容: "${textContent}"`);
      }
    }
    
    // ========================================
    // 3. 检查 API 请求
    // ========================================
    console.log('\n====================================');
    console.log('3. 监听 API 请求');
    console.log('====================================');
    
    const apiRequests = [];
    const apiResponses = [];
    
    page.on('request', request => {
      if (request.url().includes('/api/')) {
        apiRequests.push({
          url: request.url(),
          method: request.method(),
          time: new Date().toISOString()
        });
        console.log(`  → API请求: ${request.method()} ${request.url()}`);
      }
    });
    
    page.on('response', async response => {
      if (response.url().includes('/api/')) {
        const status = response.status();
        let body = null;
        try {
          body = await response.text();
        } catch (e) {
          body = '[无法读取响应体]';
        }
        
        apiResponses.push({
          url: response.url(),
          status: status,
          body: body,
          time: new Date().toISOString()
        });
        
        console.log(`  ← API响应: ${status} ${response.url()}`);
        if (body && body.length < 500) {
          console.log(`     响应体: ${body}`);
        } else if (body) {
          console.log(`     响应体: ${body.substring(0, 200)}... (已截断)`);
        }
      }
    });
    
    // 刷新页面以捕获 API 请求
    console.log('\n刷新页面以捕获 API 请求...');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    
    // ========================================
    // 4. 检查特定的 SaltStack API
    // ========================================
    console.log('\n====================================');
    console.log('4. 检查 SaltStack 相关 API');
    console.log('====================================');
    
    const saltStackAPIs = [
      '/api/slurm/saltstack/status',
      '/api/saltstack/status',
      '/api/slurm/status',
      '/api/salt/status'
    ];
    
    for (const apiPath of saltStackAPIs) {
      const found = apiRequests.some(req => req.url.includes(apiPath));
      console.log(`  ${found ? '✓' : '✗'} ${apiPath}`);
      
      if (found) {
        const responses = apiResponses.filter(res => res.url.includes(apiPath));
        responses.forEach(res => {
          console.log(`    状态码: ${res.status}`);
          if (res.body && res.body.length < 1000) {
            console.log(`    响应: ${res.body}`);
          }
        });
      }
    }
    
    // ========================================
    // 5. 直接调用 API 测试
    // ========================================
    console.log('\n====================================');
    console.log('5. 直接调用 SaltStack API');
    console.log('====================================');
    
    const testAPIs = [
      'http://192.168.0.200:8080/api/slurm/saltstack/status',
      'http://192.168.0.200:8080/api/saltstack/status',
      'http://192.168.0.200:8080/api/slurm/status'
    ];
    
    for (const apiUrl of testAPIs) {
      try {
        console.log(`\n测试 API: ${apiUrl}`);
        const response = await page.request.get(apiUrl);
        const status = response.status();
        const body = await response.text();
        
        console.log(`  状态码: ${status}`);
        console.log(`  响应长度: ${body.length} 字节`);
        
        if (status === 200) {
          try {
            const json = JSON.parse(body);
            console.log('  响应 JSON:');
            console.log(JSON.stringify(json, null, 2));
          } catch (e) {
            console.log('  响应体 (非JSON):');
            console.log(body.substring(0, 500));
          }
        } else {
          console.log(`  错误响应: ${body}`);
        }
      } catch (error) {
        console.error(`  ✗ API 调用失败: ${error.message}`);
      }
    }
    
    // ========================================
    // 6. 检查页面 DOM 结构
    // ========================================
    console.log('\n====================================');
    console.log('6. 检查页面 DOM 结构');
    console.log('====================================');
    
    // 查找所有状态相关的元素
    const statusElements = await page.locator('[class*="status"], [class*="card"], [class*="stat"]').all();
    console.log(`找到 ${statusElements.length} 个状态相关元素`);
    
    for (let i = 0; i < Math.min(statusElements.length, 10); i++) {
      const element = statusElements[i];
      const text = await element.textContent();
      const className = await element.getAttribute('class');
      console.log(`  [${i}] class="${className}"`);
      console.log(`      text="${text?.substring(0, 100)}"`);
    }
    
    // ========================================
    // 7. 截图最终状态
    // ========================================
    console.log('\n====================================');
    console.log('7. 保存最终截图');
    console.log('====================================');
    
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-status-02-final.png',
      fullPage: true 
    });
    console.log('✓ 截图保存: saltstack-status-02-final.png');
    
    // ========================================
    // 8. 总结
    // ========================================
    console.log('\n====================================');
    console.log('诊断总结');
    console.log('====================================');
    console.log(`总 API 请求数: ${apiRequests.length}`);
    console.log(`总 API 响应数: ${apiResponses.length}`);
    console.log('\nAPI 请求列表:');
    apiRequests.forEach((req, idx) => {
      console.log(`  ${idx + 1}. ${req.method} ${req.url}`);
    });
    
    console.log('\nAPI 响应状态:');
    apiResponses.forEach((res, idx) => {
      console.log(`  ${idx + 1}. ${res.status} ${res.url}`);
    });
    
    console.log('\n====================================');
    console.log('诊断完成');
    console.log('====================================');
  });

  test('检查 SaltStack Master 连接性', async ({ page }) => {
    console.log('\n====================================');
    console.log('测试 Salt Master API 直接连接');
    console.log('====================================');
    
    // 测试 Salt Master API
    const saltMasterUrls = [
      'http://192.168.0.200:8000',
      'http://192.168.0.200:8000/login',
      'http://salt-master:8000'
    ];
    
    for (const url of saltMasterUrls) {
      try {
        console.log(`\n测试连接: ${url}`);
        const response = await page.request.get(url, { timeout: 5000 });
        console.log(`  ✓ 状态码: ${response.status()}`);
      } catch (error) {
        console.log(`  ✗ 连接失败: ${error.message}`);
      }
    }
  });
});
