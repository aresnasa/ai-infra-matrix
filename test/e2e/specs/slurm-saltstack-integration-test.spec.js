// @ts-nocheck
/* eslint-disable */
/**
 * SLURM SaltStack 集成测试
 * 
 * 测试内容:
 * 1. 验证SaltStack API连接
 * 2. 检查SaltStack集成状态
 * 3. 验证Minions状态同步
 * 4. 测试前端页面显示
 */

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('SLURM SaltStack集成测试', () => {
  
  test('测试SaltStack集成API', async ({ request }) => {
    // 登录获取token
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    console.log('✓ 登录成功');
    const token = loginData.token;
    
    // 测试SaltStack集成API
    console.log('\n🔌 测试 /api/slurm/saltstack/integration');
    const integrationResponse = await request.get(BASE + '/api/slurm/saltstack/integration', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    console.log('Response status:', integrationResponse.status());
    
    const integrationData = await integrationResponse.json();
    console.log('Response data:', JSON.stringify(integrationData, null, 2));
    
    // 检查响应状态
    if (integrationResponse.status() === 503) {
      console.log('\n⚠️  SaltStack服务不可用');
      console.log('错误信息:', integrationData.error);
      console.log('\n诊断信息:');
      if (integrationData.data) {
        console.log('  - Enabled:', integrationData.data.enabled);
        console.log('  - Master Status:', integrationData.data.master_status);
        console.log('  - API Status:', integrationData.data.api_status);
        console.log('  - Minions Total:', integrationData.data.minions?.total);
      }
    } else if (integrationResponse.ok()) {
      console.log('\n✅ SaltStack服务可用');
      console.log('集成状态:', integrationData.data);
      
      if (integrationData.data) {
        console.log('\n📊 详细信息:');
        console.log('  - Enabled:', integrationData.data.enabled);
        console.log('  - Master Status:', integrationData.data.master_status);
        console.log('  - API Status:', integrationData.data.api_status);
        console.log('  - Minions Total:', integrationData.data.minions?.total);
        console.log('  - Minions Online:', integrationData.data.minions?.online);
        console.log('  - Minions Offline:', integrationData.data.minions?.offline);
        console.log('  - Recent Jobs:', integrationData.data.recent_jobs);
      }
    }
  });

  test('测试SaltStack原始状态API', async ({ request }) => {
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 测试原始SaltStack状态API
    console.log('\n🔍 测试 /api/saltstack/status');
    const statusResponse = await request.get(BASE + '/api/saltstack/status', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    console.log('Response status:', statusResponse.status());
    
    const statusData = await statusResponse.json();
    console.log('Response data:', JSON.stringify(statusData, null, 2));
    
    if (statusData.data) {
      console.log('\n📊 SaltStack状态:');
      console.log('  - Status:', statusData.data.status);
      console.log('  - Demo Mode:', statusData.data.demo);
      console.log('  - Connected Minions:', statusData.data.connected_minions);
      console.log('  - Accepted Keys:', statusData.data.accepted_keys?.length);
      console.log('  - Services:', statusData.data.services);
    }
  });

  test('测试SaltStack Minions API', async ({ request }) => {
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 测试Minions列表
    console.log('\n📋 测试 /api/saltstack/minions');
    const minionsResponse = await request.get(BASE + '/api/saltstack/minions', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    console.log('Response status:', minionsResponse.status());
    
    const minionsData = await minionsResponse.json();
    console.log('Response data:', JSON.stringify(minionsData, null, 2));
    
    if (minionsData.data && Array.isArray(minionsData.data)) {
      console.log(`\n找到 ${minionsData.data.length} 个 Minions:`);
      minionsData.data.forEach((minion, index) => {
        console.log(`  ${index + 1}. ${minion.id} - Status: ${minion.status}`);
      });
    }
  });

  test('检查SaltStack环境变量配置', async ({ request }) => {
    // 这个测试用于诊断配置问题
    console.log('\n🔧 检查SaltStack配置要求:');
    console.log('─'.repeat(80));
    console.log('Backend需要的环境变量:');
    console.log('  - SALT_API_URL: SaltStack API地址 (例如: http://salt-master:8000)');
    console.log('  - SALT_API_USERNAME: SaltStack API用户名 (默认: saltapi)');
    console.log('  - SALT_API_PASSWORD: SaltStack API密码 (默认: saltapi123)');
    console.log('  - SALT_API_EAUTH: 认证方式 (默认: file)');
    console.log('  - SALT_API_TIMEOUT: 超时时间 (默认: 8s)');
    console.log('─'.repeat(80));
    
    console.log('\n可能的问题:');
    console.log('  1. SaltStack Master未启动');
    console.log('  2. salt-api服务未运行');
    console.log('  3. 环境变量配置错误');
    console.log('  4. 网络连接问题');
    console.log('  5. 认证失败');
  });

  test('测试前端SaltStack标签页加载', async ({ page }) => {
    // 登录
    await page.goto(BASE + '/');
    await page.waitForLoadState('domcontentloaded');
    
    const loginTab = await page.locator('text=登录').first().isVisible({ timeout: 3000 }).catch(() => false);
    if (loginTab) {
      console.log('需要登录...');
      await page.fill('input[placeholder*="用户名"]', 'admin');
      await page.fill('input[placeholder*="密码"]', 'admin123');
      await page.click('button:has-text("登录")');
      await page.waitForTimeout(2000);
    }
    
    // 访问SLURM页面
    console.log('\n📄 测试SLURM页面 - SaltStack标签');
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    // 查找SaltStack标签
    const saltStackTab = page.locator('text=SaltStack 集成').first();
    const isVisible = await saltStackTab.isVisible();
    
    console.log(`SaltStack标签可见: ${isVisible ? '✓' : '✗'}`);
    
    if (isVisible) {
      // 点击SaltStack标签
      await saltStackTab.click();
      await page.waitForTimeout(2000);
      
      // 检查SaltStack状态卡片
      const statusCard = page.locator('text=SaltStack 状态');
      const cardVisible = await statusCard.isVisible();
      console.log(`SaltStack状态卡片可见: ${cardVisible ? '✓' : '✗'}`);
      
      // 截图
      await page.screenshot({ 
        path: 'test-screenshots/slurm-saltstack-tab.png',
        fullPage: true 
      });
      console.log('✓ SaltStack标签截图已保存: test-screenshots/slurm-saltstack-tab.png');
      
      // 检查状态信息
      const pageContent = await page.content();
      
      // 检查是否有错误信息
      if (pageContent.includes('unavailable') || pageContent.includes('不可用')) {
        console.log('\n⚠️  检测到SaltStack服务不可用');
      } else if (pageContent.includes('success') || pageContent.includes('running')) {
        console.log('\n✅ SaltStack服务正常运行');
      }
      
      // 检查Minions信息
      const minionsList = page.locator('text=SaltStack Minions');
      if (await minionsList.isVisible()) {
        console.log('✓ Minions列表区域可见');
      }
    } else {
      console.log('⚠️  SaltStack标签未找到');
    }
  });

  test('诊断SaltStack API直连测试', async ({ request }) => {
    console.log('\n🔍 直接测试SaltStack API连接');
    console.log('─'.repeat(80));
    
    // 尝试常见的SaltStack API地址
    const possibleURLs = [
      'http://salt-master:8000',
      'http://192.168.0.200:8000',
      'http://localhost:8000',
    ];
    
    for (const url of possibleURLs) {
      console.log(`\n测试URL: ${url}`);
      
      try {
        // 尝试访问登录端点
        const response = await request.post(url + '/login', {
          data: {
            username: 'saltapi',
            password: 'saltapi123',
            eauth: 'file'
          },
          timeout: 5000
        }).catch(e => {
          console.log(`  ✗ 连接失败: ${e.message}`);
          return null;
        });
        
        if (response) {
          console.log(`  Status: ${response.status()}`);
          if (response.status() === 200) {
            const data = await response.json();
            console.log(`  ✓ 认证成功`);
            console.log(`  Response:`, JSON.stringify(data, null, 2));
          } else if (response.status() === 401) {
            console.log(`  ⚠️  认证失败 - 用户名或密码错误`);
          } else {
            console.log(`  ⚠️  未预期的响应`);
          }
        }
      } catch (error) {
        console.log(`  ✗ 异常: ${error.message}`);
      }
    }
    
    console.log('─'.repeat(80));
  });

  test('生成SaltStack问题诊断报告', async ({ request }) => {
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    console.log('\n📋 SaltStack问题诊断报告');
    console.log('═'.repeat(80));
    
    // 1. 测试集成API
    console.log('\n1️⃣  测试集成API (/api/slurm/saltstack/integration)');
    const integrationResp = await request.get(BASE + '/api/slurm/saltstack/integration', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    const integrationStatus = integrationResp.status();
    const integrationData = await integrationResp.json();
    
    console.log(`   Status Code: ${integrationStatus}`);
    if (integrationStatus === 503) {
      console.log(`   ❌ 服务不可用`);
      console.log(`   错误: ${integrationData.error}`);
    } else if (integrationStatus === 200) {
      console.log(`   ✅ 服务可用`);
      console.log(`   API Status: ${integrationData.data?.api_status}`);
      console.log(`   Master Status: ${integrationData.data?.master_status}`);
    }
    
    // 2. 测试原始状态API
    console.log('\n2️⃣  测试原始状态API (/api/saltstack/status)');
    const statusResp = await request.get(BASE + '/api/saltstack/status', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    const statusData = await statusResp.json();
    console.log(`   Status Code: ${statusResp.status()}`);
    console.log(`   Demo Mode: ${statusData.data?.demo}`);
    console.log(`   Status: ${statusData.data?.status}`);
    console.log(`   Connected Minions: ${statusData.data?.connected_minions}`);
    
    // 3. 测试Minions API
    console.log('\n3️⃣  测试Minions API (/api/saltstack/minions)');
    const minionsResp = await request.get(BASE + '/api/saltstack/minions', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    const minionsData = await minionsResp.json();
    console.log(`   Status Code: ${minionsResp.status()}`);
    console.log(`   Minions Count: ${minionsData.data?.length || 0}`);
    
    // 4. 总结
    console.log('\n📊 诊断总结');
    console.log('─'.repeat(80));
    
    if (integrationStatus === 503) {
      console.log('❌ 问题确认: SaltStack服务不可用');
      console.log('\n可能原因:');
      console.log('  1. SaltStack Master未启动或未配置');
      console.log('  2. salt-api服务未运行');
      console.log('  3. Backend无法连接到SaltStack API');
      console.log('  4. 环境变量SALT_API_URL配置错误');
      console.log('\n建议解决方案:');
      console.log('  1. 检查docker-compose.test.yml中salt-master服务');
      console.log('  2. 验证SALT_API_URL环境变量');
      console.log('  3. 确认salt-api服务在端口8000上运行');
      console.log('  4. 检查Backend日志: docker-compose logs backend');
    } else if (statusData.data?.demo === true) {
      console.log('⚠️  问题确认: 使用Demo数据模式');
      console.log('\n原因:');
      console.log('  - Backend正在返回演示数据而不是真实SaltStack状态');
      console.log('\n建议解决方案:');
      console.log('  1. 配置真实的SaltStack Master连接');
      console.log('  2. 确保SALT_API_URL环境变量正确设置');
      console.log('  3. 重启Backend服务以应用配置');
    } else if (integrationStatus === 200 && minionsData.data?.length === 0) {
      console.log('⚠️  问题确认: SaltStack连接正常但无Minions');
      console.log('\n建议:');
      console.log('  1. 部署Salt Minions到计算节点');
      console.log('  2. 在Master上接受Minion密钥');
      console.log('  3. 验证Minions与Master的连接');
    } else {
      console.log('✅ SaltStack集成正常工作');
    }
    
    console.log('═'.repeat(80));
  });
});
