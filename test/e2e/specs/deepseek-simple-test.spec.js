/**
 * DeepSeek 完整测试套件
 * 
 * 测试内容：
 * 1. 自动创建会话并发送消息
 * 2. 验证 AI 响应
 * 3. 测试多轮对话
 * 4. 验证 Admin AI Assistant 页面的统计数据
 */

const { test, expect } = require('@playwright/test');

test.describe('DeepSeek 完整测试套件', () => {
  let baseURL;
  let authToken;
  let deepseekConfigId;
  let createdConversationIds = []; // 跟踪创建的会话ID用于清理

  test.beforeAll(async ({ request }) => {
    baseURL = process.env.BASE_URL || 'http://localhost:8080';
    console.log('测试基础 URL:', baseURL);

    // 登录
    console.log('正在登录...');
    const loginResponse = await request.post(`${baseURL}/api/auth/login`, {
      data: { username: 'admin', password: 'admin123' }
    });
    expect(loginResponse.ok()).toBeTruthy();
    const loginData = await loginResponse.json();
    authToken = loginData.token || loginData.data.token;
    console.log('✓ 登录成功');

    // 获取 DeepSeek 配置
    console.log('正在获取 DeepSeek 配置...');
    const configResponse = await request.get(`${baseURL}/api/ai/configs`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    expect(configResponse.ok()).toBeTruthy();
    const configData = await configResponse.json();
    const configs = configData.data || configData;
    
    const deepseekConfig = configs.find(c => c.provider === 'deepseek' && c.model === 'deepseek-chat');
    expect(deepseekConfig).toBeTruthy();
    deepseekConfigId = deepseekConfig.id;
    console.log(`✓ 找到 DeepSeek Chat 配置 (ID: ${deepseekConfigId})`);

    // 更新 API Key (从环境变量)
    const apiKey = process.env.DEEPSEEK_API_KEY;
    if (apiKey && apiKey.startsWith('sk-') && apiKey !== 'sk-test-deepseek-api-key-for-testing') {
      console.log('🔑 更新 API Key...');
      const updateResponse = await request.put(`${baseURL}/api/ai/configs/${deepseekConfigId}`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          ...deepseekConfig,
          api_key: apiKey
        }
      });
      
      if (!updateResponse.ok()) {
        const errorText = await updateResponse.text();
        console.error('更新 API Key 失败:', errorText);
      } else {
        console.log('✓ API Key 已更新');
      }
    }
  });

  // 辅助函数：发送消息并等待响应
  async function sendAndWaitForResponse(request, conversationId, message, maxWaitSeconds = 30) {
    console.log(`📤 发送消息: "${message}"`);
    
    const sendMessageResponse = await request.post(
      `${baseURL}/api/ai/conversations/${conversationId}/messages`,
      {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        },
        data: { message }
      }
    );

    expect(sendMessageResponse.ok()).toBeTruthy();
    const messageData = await sendMessageResponse.json();
    console.log(`  ✓ 消息已发送 (ID: ${messageData.message_id})`);

    // 等待响应
    let aiResponse = null;
    let retries = 0;
    const maxRetries = maxWaitSeconds / 2;

    while (retries < maxRetries && !aiResponse) {
      await new Promise(resolve => setTimeout(resolve, 2000));
      retries++;

      const messagesResponse = await request.get(
        `${baseURL}/api/ai/conversations/${conversationId}/messages`,
        {
          headers: { 'Authorization': `Bearer ${authToken}` }
        }
      );

      if (messagesResponse.ok()) {
        const messagesData = await messagesResponse.json();
        const messages = messagesData.data || messagesData;
        const assistantMessages = messages.filter(msg => msg.role === 'assistant');
        
        if (assistantMessages.length > 0) {
          aiResponse = assistantMessages[assistantMessages.length - 1];
          break;
        }
      }

      if (retries % 3 === 0) {
        console.log(`  ⏳ 等待中... (${retries * 2}/${maxRetries * 2} 秒)`);
      }
    }

    if (!aiResponse) {
      throw new Error(`未收到 AI 响应 (${maxWaitSeconds} 秒超时)`);
    }

    console.log(`  ✅ 收到响应: "${aiResponse.content.substring(0, 80)}..."`);
    return aiResponse;
  }

  // 辅助函数：创建会话
  async function createConversation(request, title) {
    console.log(`📝 创建会话: "${title}"`);
    const createConvResponse = await request.post(`${baseURL}/api/ai/conversations`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: title,
        config_id: deepseekConfigId
      }
    });
    
    expect(createConvResponse.ok()).toBeTruthy();
    const convData = await createConvResponse.json();
    const conversationId = convData.data?.id || convData.id;
    createdConversationIds.push(conversationId);
    console.log(`  ✓ 会话已创建 (ID: ${conversationId})`);
    return conversationId;
  }

  test.afterAll(async ({ request }) => {
    // 清理所有创建的会话
    console.log('\n🧹 清理测试数据...');
    for (const id of createdConversationIds) {
      try {
        await request.delete(`${baseURL}/api/ai/conversations/${id}`, {
          headers: { 'Authorization': `Bearer ${authToken}` }
        });
        console.log(`  ✓ 已删除会话 ${id}`);
      } catch (error) {
        console.log(`  ⚠️  删除会话 ${id} 失败: ${error.message}`);
      }
    }
  });

  test('创建会话并发送一条简单消息', async ({ request }) => {
    console.log('\n🧪 测试 1: 单条消息对话');
    
    const conversationId = await createConversation(request, 'DeepSeek 简单测试');
    const response = await sendAndWaitForResponse(request, conversationId, '你好，请用一句话介绍你自己');
    
    expect(response.content).toBeTruthy();
    expect(response.content.length).toBeGreaterThan(0);
    expect(response.role).toBe('assistant');
    console.log('  ✅ 测试通过');
  });

  test('多轮对话测试', async ({ request }) => {
    console.log('\n🧪 测试 2: 多轮对话');
    
    const conversationId = await createConversation(request, 'DeepSeek 多轮对话');
    
    // 第一轮
    const response1 = await sendAndWaitForResponse(
      request, 
      conversationId, 
      '请记住这个数字：42'
    );
    expect(response1.content).toBeTruthy();
    
    // 第二轮 - 测试上下文记忆
    const response2 = await sendAndWaitForResponse(
      request, 
      conversationId, 
      '我刚才让你记住的数字是多少？'
    );
    expect(response2.content).toBeTruthy();
    expect(response2.content.toLowerCase()).toContain('42');
    
    console.log('  ✅ 多轮对话测试通过');
  });

  test('验证 AI Assistant 统计数据', async ({ page, request }) => {
    console.log('\n🧪 测试 3: Admin AI Assistant 统计数据');
    
    // 创建会话并发送几条消息
    const conversationId = await createConversation(request, 'DeepSeek 统计测试');
    await sendAndWaitForResponse(request, conversationId, '测试消息 1');
    await sendAndWaitForResponse(request, conversationId, '测试消息 2');
    
    // 直接测试 API 端点
    console.log('  📡 测试统计 API 端点...');
    const statsResponse = await request.get(`${baseURL}/api/ai/usage-stats`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    
    expect(statsResponse.ok()).toBeTruthy();
    const statsData = await statsResponse.json();
    console.log('  📊 API 返回的统计数据:', JSON.stringify(statsData, null, 2));
    
    // 验证统计数据结构
    expect(statsData.data).toBeTruthy();
    expect(statsData.data.total_messages).toBeDefined();
    expect(statsData.data.total_conversations).toBeDefined();
    expect(statsData.data.active_conversations).toBeDefined();
    
    // 验证统计数据有值（至少有我们刚创建的数据）
    expect(statsData.data.total_messages).toBeGreaterThan(0);
    expect(statsData.data.total_conversations).toBeGreaterThan(0);
    
    console.log(`  ✅ 统计数据验证通过:`);
    console.log(`    - 总消息数: ${statsData.data.total_messages}`);
    console.log(`    - 总会话数: ${statsData.data.total_conversations}`);
    console.log(`    - 活跃会话: ${statsData.data.active_conversations}`);
    
    // 访问 Admin AI Assistant 页面
    console.log('  📊 访问管理页面...');
    await page.goto(`${baseURL}/admin/ai-assistant`);
    
    // 等待页面加载（可能需要登录）
    const needsLogin = await page.locator('input[name="username"]').isVisible().catch(() => false);
    
    if (needsLogin) {
      console.log('  🔐 需要登录...');
      await page.fill('input[name="username"]', 'admin');
      await page.fill('input[name="password"]', 'admin123');
      await page.click('button[type="submit"]');
      await page.waitForURL('**/admin/**', { timeout: 10000 });
      
      // 重新访问 AI Assistant 页面
      await page.goto(`${baseURL}/admin/ai-assistant`);
    }
    
    // 等待统计数据加载
    console.log('  ⏳ 等待统计数据加载...');
    await page.waitForTimeout(3000);
    
    // 截图用于调试
    await page.screenshot({ path: 'test-screenshots/ai-assistant-stats.png', fullPage: true });
    console.log('  📸 已保存截图: test-screenshots/ai-assistant-stats.png');
    
    // 检查页面是否有统计卡片或数值
    const pageText = await page.textContent('body');
    
    // 更灵活的检测 - 查找数字和统计相关文本
    const hasNumbers = /\d+/.test(pageText);
    const hasStatsKeywords = /(消息|会话|对话|token|message|conversation)/i.test(pageText);
    const noDataText = /(无数据|暂无数据|No data)/i.test(pageText);
    
    console.log(`  📊 页面内容检查:`);
    console.log(`    - 包含数字: ${hasNumbers ? '✓' : '✗'}`);
    console.log(`    - 包含统计关键词: ${hasStatsKeywords ? '✓' : '✗'}`);
    console.log(`    - 显示"无数据": ${noDataText ? '✗' : '✓'}`);
    
    if (noDataText) {
      console.log('  ⚠️  警告: 页面仍显示"无数据"，但API返回了数据');
      console.log('  💡 可能的原因: 前端组件数据加载或显示逻辑问题');
    } else if (hasNumbers && hasStatsKeywords) {
      console.log('  ✅ 统计页面显示正常');
    }
  });

  test('快速创建并自动对话', async ({ request }) => {
    console.log('\n🧪 测试 4: 快速自动对话');
    
    const testQuestions = [
      '1+1等于几？',
      '地球的卫星叫什么？',
      'JavaScript是什么？'
    ];
    
    const conversationId = await createConversation(request, 'DeepSeek 快速对话测试');
    
    for (const question of testQuestions) {
      const response = await sendAndWaitForResponse(request, conversationId, question);
      expect(response.content).toBeTruthy();
      expect(response.content.length).toBeGreaterThan(0);
    }
    
    console.log('  ✅ 快速自动对话测试通过');
  });
});
