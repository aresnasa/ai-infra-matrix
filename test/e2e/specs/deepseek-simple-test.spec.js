/**
 * DeepSeek 最小化测试
 * 
 * 用于快速验证 DeepSeek API 是否可以正常工作
 * 不依赖异步消息队列，直接测试 API 调用
 */

const { test, expect } = require('@playwright/test');

test.describe('DeepSeek 最小化测试', () => {
  let baseURL;
  let authToken;
  let deepseekConfigId;

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

  test('创建会话并发送一条简单消息', async ({ request }) => {
    console.log('\n开始测试...');
    
    // 1. 创建会话
    console.log('1. 创建会话...');
    const createConvResponse = await request.post(`${baseURL}/api/ai/conversations`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: 'DeepSeek 最小化测试',
        config_id: deepseekConfigId
      }
    });
    
    expect(createConvResponse.ok()).toBeTruthy();
    const convData = await createConvResponse.json();
    const conversationId = convData.data?.id || convData.id;
    console.log(`✓ 会话已创建 (ID: ${conversationId})`);

    // 2. 发送消息
    console.log('2. 发送消息...');
    const sendMessageResponse = await request.post(
      `${baseURL}/api/ai/conversations/${conversationId}/messages`,
      {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          message: '你好'
        }
      }
    );

    expect(sendMessageResponse.ok()).toBeTruthy();
    const messageData = await sendMessageResponse.json();
    console.log('✓ 消息已发送');
    console.log(`  Message ID: ${messageData.message_id}`);
    console.log(`  Status: ${messageData.status}`);

    // 3. 等待响应 (最多 30 秒，每 2 秒检查一次)
    console.log('3. 等待 AI 响应...');
    let aiResponse = null;
    let retries = 0;
    const maxRetries = 15; // 30 秒 / 2 秒

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
        console.log(`  等待中... (${retries * 2}/${maxRetries * 2} 秒)`);
      }
    }

    // 4. 验证响应
    if (aiResponse) {
      console.log('✅ 收到 AI 响应！');
      console.log(`  内容: ${aiResponse.content.substring(0, 100)}...`);
      expect(aiResponse.content).toBeTruthy();
      expect(aiResponse.content.length).toBeGreaterThan(0);
    } else {
      console.log('⚠️  30 秒内未收到响应');
      console.log('\n诊断信息：');
      console.log('  - 检查后端日志: docker compose logs backend --tail=50');
      console.log('  - 检查 Redis Stream: docker compose exec redis redis-cli XLEN "ai:chat:requests"');
      console.log('  - 检查消息状态:', messageData);
      throw new Error('未收到 AI 响应 (30 秒超时)');
    }

    // 5. 清理
    console.log('4. 清理会话...');
    await request.delete(`${baseURL}/api/ai/conversations/${conversationId}`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    console.log('✓ 测试完成');
  });
});
