/**
 * 统计API测试脚本
 * 使用 Playwright 测试 AI 使用统计 API
 * 
 * 运行方式：
 * BASE_URL=http://192.168.0.200:8080 node test/e2e/test-usage-stats.js
 */

const { chromium } = require('@playwright/test');

async function testUsageStatsAPI() {
  const baseURL = process.env.BASE_URL || 'http://localhost:8080';
  console.log('🌐 测试基础 URL:', baseURL);

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // 1. 登录获取 Token
    console.log('\n🔐 登录获取 Token...');
    const loginResponse = await page.request.post(`${baseURL}/api/auth/login`, {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });

    if (!loginResponse.ok()) {
      console.error('❌ 登录失败:', loginResponse.status());
      const errorText = await loginResponse.text();
      console.error(errorText);
      return;
    }

    const loginData = await loginResponse.json();
    const authToken = loginData.token || loginData.data?.token;
    
    if (!authToken) {
      console.error('❌ 未获取到 Token');
      console.log('登录响应:', loginData);
      return;
    }

    console.log('✅ 登录成功');
    console.log('   Token:', authToken.substring(0, 20) + '...');

    // 2. 测试路由 1: /api/ai/usage-stats
    console.log('\n📊 测试统计 API (路由 1: /api/ai/usage-stats)...');
    const stats1Response = await page.request.get(`${baseURL}/api/ai/usage-stats`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    console.log('   状态码:', stats1Response.status());
    
    if (stats1Response.ok()) {
      const stats1Data = await stats1Response.json();
      console.log('   ✅ API 响应成功');
      console.log('   数据结构:', JSON.stringify(stats1Data, null, 2));
      
      if (stats1Data.data) {
        console.log('\n   📈 统计数据详情:');
        console.log('      - 总消息数:', stats1Data.data.total_messages || 0);
        console.log('      - 总会话数:', stats1Data.data.total_conversations || 0);
        console.log('      - 活跃会话:', stats1Data.data.active_conversations || 0);
        console.log('      - Token 使用:', stats1Data.data.total_tokens_used || 0);
        console.log('      - 时间范围:', stats1Data.data.start_date, '至', stats1Data.data.end_date);
        
        if (stats1Data.data.provider_stats && stats1Data.data.provider_stats.length > 0) {
          console.log('      - 提供商统计:');
          stats1Data.data.provider_stats.forEach(stat => {
            console.log(`        * ${stat.provider || stat.Provider}: ${stat.count || stat.Count} 条消息`);
          });
        }
      }
    } else {
      console.log('   ❌ API 请求失败');
      const errorText = await stats1Response.text();
      console.log('   错误信息:', errorText);
    }

    // 3. 测试路由 2: /api/ai/system/usage
    console.log('\n📊 测试统计 API (路由 2: /api/ai/system/usage)...');
    const stats2Response = await page.request.get(`${baseURL}/api/ai/system/usage`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    console.log('   状态码:', stats2Response.status());
    
    if (stats2Response.ok()) {
      const stats2Data = await stats2Response.json();
      console.log('   ✅ API 响应成功');
      console.log('   数据结构:', JSON.stringify(stats2Data, null, 2));
    } else {
      console.log('   ❌ API 请求失败');
      const errorText = await stats2Response.text();
      console.log('   错误信息:', errorText);
    }

    // 4. 测试带时间范围参数的查询
    console.log('\n📊 测试带时间范围的统计查询...');
    const today = new Date().toISOString().split('T')[0];
    const sevenDaysAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];
    
    const stats3Response = await page.request.get(
      `${baseURL}/api/ai/usage-stats?start_date=${sevenDaysAgo}&end_date=${today}`,
      {
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      }
    );

    console.log('   状态码:', stats3Response.status());
    
    if (stats3Response.ok()) {
      const stats3Data = await stats3Response.json();
      console.log('   ✅ 带参数查询成功');
      console.log('   查询时间范围:', sevenDaysAgo, '至', today);
      if (stats3Data.data) {
        console.log('   该时间段统计:');
        console.log('      - 消息数:', stats3Data.data.total_messages || 0);
        console.log('      - 会话数:', stats3Data.data.total_conversations || 0);
      }
    } else {
      console.log('   ❌ 带参数查询失败');
    }

    // 5. 获取配置列表以查看数据
    console.log('\n🔧 获取 AI 配置列表...');
    const configsResponse = await page.request.get(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    if (configsResponse.ok()) {
      const configsData = await configsResponse.json();
      const configs = configsData.data || configsData;
      console.log('   ✅ 找到', configs.length, '个 AI 配置');
      configs.forEach(config => {
        console.log(`      - ${config.name} (${config.provider}/${config.model})`);
      });
    }

    // 6. 获取会话列表
    console.log('\n💬 获取会话列表...');
    const conversationsResponse = await page.request.get(`${baseURL}/api/ai/conversations`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    if (conversationsResponse.ok()) {
      const conversationsData = await conversationsResponse.json();
      const conversations = conversationsData.data || conversationsData;
      console.log('   ✅ 找到', conversations.length, '个会话');
      if (conversations.length > 0) {
        console.log('   最近的会话:');
        conversations.slice(0, 3).forEach(conv => {
          console.log(`      - ID: ${conv.id}, 标题: ${conv.title || '(无标题)'}`);
        });
      }
    }

    console.log('\n✅ 所有测试完成');

  } catch (error) {
    console.error('\n❌ 测试过程中出错:', error.message);
    console.error(error.stack);
  } finally {
    await browser.close();
  }
}

// 运行测试
testUsageStatsAPI().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
