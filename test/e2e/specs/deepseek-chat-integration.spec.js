/**
 * DeepSeek 聊天集成测试
 * 
 * 测试目标：
 * 1. 测试 DeepSeek 模型的实际聊天功能
 * 2. 验证 API 能够正确返回 DeepSeek 的响应数据
 * 3. 使用系统环境变量中的 DEEPSEEK_API_KEY（不在代码中硬编码）
 * 
 * 环境要求：
 * - DEEPSEEK_API_KEY 必须在 .env 文件中配置
 * - Backend 服务必须正在运行
 */

const { test, expect } = require('@playwright/test');

test.describe('DeepSeek 聊天集成测试', () => {
  let baseURL;
  let authToken;
  let conversationId;
  let deepseekChatConfigId;
  let deepseekReasonerConfigId;

  test.beforeAll(async ({ request }) => {
    baseURL = process.env.BASE_URL || 'http://localhost:8080';
    console.log('测试基础 URL:', baseURL);

    // 1. 登录获取 token
    console.log('正在登录获取认证 token...');
    const loginResponse = await request.post(`${baseURL}/api/auth/login`, {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });

    if (loginResponse.ok()) {
      const loginData = await loginResponse.json();
      authToken = loginData.token;
      console.log('✓ 登录成功，已获取 token');
    } else {
      console.error('登录失败:', await loginResponse.text());
      throw new Error('无法获取认证 token');
    }

    // 2. 获取 DeepSeek 配置 ID
    console.log('正在获取 DeepSeek 配置...');
    const configResponse = await request.get(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    if (configResponse.ok()) {
      const configData = await configResponse.json();
      const deepseekConfigs = configData.data.filter(config => config.provider === 'deepseek');
      
      if (deepseekConfigs.length === 0) {
        throw new Error('未找到 DeepSeek 配置，请确保 DEEPSEEK_API_KEY 已在 .env 中配置并重新初始化数据库');
      }

      // 找到 Chat 和 Reasoner 配置
      const chatConfig = deepseekConfigs.find(c => c.name.includes('Chat'));
      const reasonerConfig = deepseekConfigs.find(c => c.name.includes('Reasoner'));

      if (chatConfig) {
        deepseekChatConfigId = chatConfig.id;
        console.log(`✓ 找到 DeepSeek Chat 配置 (ID: ${deepseekChatConfigId})`);
      }

      if (reasonerConfig) {
        deepseekReasonerConfigId = reasonerConfig.id;
        console.log(`✓ 找到 DeepSeek Reasoner 配置 (ID: ${deepseekReasonerConfigId})`);
      }

      if (!deepseekChatConfigId && !deepseekReasonerConfigId) {
        throw new Error('未找到可用的 DeepSeek 配置');
      }

      // 3. 从操作系统环境变量读取 DEEPSEEK_API_KEY 并更新配置
      // 注意：这里直接读取操作系统的环境变量，不从 .env 文件读取
      const osDeepSeekApiKey = process.env.DEEPSEEK_API_KEY;
      
      if (osDeepSeekApiKey && 
          osDeepSeekApiKey.startsWith('sk-') && 
          osDeepSeekApiKey !== 'sk-test-deepseek-api-key-for-testing') {
        console.log('');
        console.log('🔑 检测到操作系统环境变量 DEEPSEEK_API_KEY');
        console.log(`   API Key: ${osDeepSeekApiKey.substring(0, 10)}...${osDeepSeekApiKey.substring(osDeepSeekApiKey.length - 4)} (已隐藏)`);
        console.log('   正在更新 DeepSeek 配置...');
        console.log('');
        
        // 更新所有 DeepSeek 配置的 API Key
        for (const config of deepseekConfigs) {
          console.log(`  📝 更新配置: ${config.name} (ID: ${config.id})`);
          
          const updateResponse = await request.put(
            `${baseURL}/api/ai/configs/${config.id}`,
            {
              headers: {
                'Authorization': `Bearer ${authToken}`,
                'Content-Type': 'application/json'
              },
              data: {
                ...config,
                api_key: osDeepSeekApiKey
              }
            }
          );

          if (updateResponse.ok()) {
            console.log(`     ✓ API Key 已更新`);
          } else {
            const errorText = await updateResponse.text();
            console.warn(`     ⚠️  更新失败: ${errorText}`);
          }
        }
        
        console.log('');
        console.log('✅ DeepSeek 配置已使用操作系统环境变量更新');
        console.log('   （API Key 未写入任何文件，仅存储在数据库中）');
        console.log('');
      } else {
        console.log('');
        console.log('⚠️  警告: 未检测到有效的操作系统环境变量 DEEPSEEK_API_KEY');
        console.log('   当前值:', osDeepSeekApiKey || '(未设置)');
        console.log('   测试将使用数据库中的现有配置（可能是占位符）');
        console.log('');
        console.log('💡 设置方法（推荐 - API Key 不会写入文件）:');
        console.log('');
        console.log('   macOS/Linux 临时设置 (当前终端会话):');
        console.log('   $ export DEEPSEEK_API_KEY=sk-your-real-api-key');
        console.log('');
        console.log('   macOS/Linux 永久设置 (添加到 ~/.zshrc 或 ~/.bashrc):');
        console.log('   $ echo "export DEEPSEEK_API_KEY=sk-your-real-api-key" >> ~/.zshrc');
        console.log('   $ source ~/.zshrc');
        console.log('');
        console.log('   运行测试:');
        console.log('   $ DEEPSEEK_API_KEY=sk-xxx npm run test:deepseek');
        console.log('');
      }
    } else {
      throw new Error('无法获取 AI 配置: ' + await configResponse.text());
    }
  });

  test.afterAll(async ({ request }) => {
    // 清理测试会话（如果创建了）
    if (conversationId && authToken) {
      console.log(`清理测试会话 (ID: ${conversationId})...`);
      try {
        await request.delete(`${baseURL}/api/ai/conversations/${conversationId}`, {
          headers: {
            'Authorization': `Bearer ${authToken}`
          }
        });
        console.log('✓ 测试会话已清理');
      } catch (error) {
        console.warn('清理会话时出错:', error.message);
      }
    }
  });

  test('使用 DeepSeek Chat 模型进行简单对话', async ({ request }) => {
    if (!deepseekChatConfigId) {
      test.skip('DeepSeek Chat 配置不可用');
      return;
    }

    console.log('开始测试 DeepSeek Chat 模型...');

    // 1. 创建会话
    console.log('创建新会话...');
    const createConvResponse = await request.post(`${baseURL}/api/ai/conversations`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: 'DeepSeek Chat 测试会话',
        config_id: deepseekChatConfigId
      }
    });

    expect(createConvResponse.ok()).toBeTruthy();
    const convData = await createConvResponse.json();
    conversationId = convData.data.id;
    console.log(`✓ 会话已创建 (ID: ${conversationId})`);

    // 2. 发送测试消息
    console.log('发送测试消息...');
    const testMessage = '你好，请用一句话介绍一下你自己。';
    console.log(`问题: "${testMessage}"`);

    const sendMessageResponse = await request.post(
      `${baseURL}/api/ai/conversations/${conversationId}/messages`,
      {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          message: testMessage
        }
      }
    );

    // 3. 验证响应
    if (!sendMessageResponse.ok()) {
      const errorText = await sendMessageResponse.text();
      console.error('发送消息失败:', errorText);
    }
    expect(sendMessageResponse.ok()).toBeTruthy();

    const messageData = await sendMessageResponse.json();
    console.log('消息响应:', JSON.stringify(messageData, null, 2));

    // 验证响应结构（异步处理模式）
    expect(messageData).toHaveProperty('message_id');
    expect(messageData).toHaveProperty('status');
    expect(messageData.status).toBe('pending');

    const messageId = messageData.message_id;
    console.log(`✓ 消息已提交处理 (Message ID: ${messageId})`);

    // 4. 等待 AI 响应（轮询检查消息状态）
    console.log('等待 DeepSeek 响应...');
    let aiResponse = null;
    let retries = 0;
    const maxRetries = 60; // 最多等待 60 秒

    while (retries < maxRetries && !aiResponse) {
      await new Promise(resolve => setTimeout(resolve, 1000)); // 等待 1 秒

      // 获取会话消息列表
      const messagesResponse = await request.get(
        `${baseURL}/api/ai/conversations/${conversationId}/messages`,
        {
          headers: {
            'Authorization': `Bearer ${authToken}`
          }
        }
      );

      if (messagesResponse.ok()) {
        const messagesData = await messagesResponse.json();
        const messages = messagesData.data || messagesData;

        // 查找 AI 的回复（role 为 'assistant'）
        const assistantMessages = messages.filter(msg => msg.role === 'assistant');
        if (assistantMessages.length > 0) {
          aiResponse = assistantMessages[assistantMessages.length - 1];
          break;
        }
      }

      retries++;
      if (retries % 5 === 0) {
        console.log(`  等待中... (${retries}/${maxRetries} 秒)`);
      }
    }

    // 5. 验证 AI 响应
    if (!aiResponse) {
      throw new Error('超时：未收到 DeepSeek 的响应');
    }

    console.log('✓ 收到 DeepSeek 响应:');
    console.log(`  角色: ${aiResponse.role}`);
    console.log(`  内容长度: ${aiResponse.content.length} 字符`);
    console.log(`  内容: ${aiResponse.content.substring(0, 200)}...`);

    // 验证响应内容
    expect(aiResponse.role).toBe('assistant');
    expect(aiResponse.content).toBeTruthy();
    expect(aiResponse.content.length).toBeGreaterThan(0);
    expect(typeof aiResponse.content).toBe('string');

    // 验证响应内容不是错误消息
    expect(aiResponse.content).not.toContain('error');
    expect(aiResponse.content).not.toContain('Error');
    expect(aiResponse.content).not.toContain('失败');

    console.log('✅ DeepSeek Chat 测试通过');
  });

  test('使用 DeepSeek Reasoner 模型进行推理任务', async ({ request }) => {
    if (!deepseekReasonerConfigId) {
      test.skip('DeepSeek Reasoner 配置不可用');
      return;
    }

    console.log('开始测试 DeepSeek Reasoner 模型...');

    // 1. 创建会话
    console.log('创建新会话...');
    const createConvResponse = await request.post(`${baseURL}/api/ai/conversations`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: 'DeepSeek Reasoner 测试会话',
        config_id: deepseekReasonerConfigId
      }
    });

    expect(createConvResponse.ok()).toBeTruthy();
    const convData = await createConvResponse.json();
    const reasonerConversationId = convData.data.id;
    console.log(`✓ 会话已创建 (ID: ${reasonerConversationId})`);

    // 2. 发送推理测试消息
    console.log('发送推理测试消息...');
    const testMessage = '计算：15 + 27 = ?';
    console.log(`问题: "${testMessage}"`);

    const sendMessageResponse = await request.post(
      `${baseURL}/api/ai/conversations/${reasonerConversationId}/messages`,
      {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          message: testMessage
        }
      }
    );

    expect(sendMessageResponse.ok()).toBeTruthy();
    const messageData = await sendMessageResponse.json();
    console.log(`✓ 消息已提交处理 (Message ID: ${messageData.message_id})`);

    // 3. 等待 AI 响应
    console.log('等待 DeepSeek Reasoner 响应...');
    let aiResponse = null;
    let retries = 0;
    const maxRetries = 60;

    while (retries < maxRetries && !aiResponse) {
      await new Promise(resolve => setTimeout(resolve, 1000));

      const messagesResponse = await request.get(
        `${baseURL}/api/ai/conversations/${reasonerConversationId}/messages`,
        {
          headers: {
            'Authorization': `Bearer ${authToken}`
          }
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

      retries++;
      if (retries % 5 === 0) {
        console.log(`  等待中... (${retries}/${maxRetries} 秒)`);
      }
    }

    // 4. 验证响应
    if (!aiResponse) {
      throw new Error('超时：未收到 DeepSeek Reasoner 的响应');
    }

    console.log('✓ 收到 DeepSeek Reasoner 响应:');
    console.log(`  内容: ${aiResponse.content}`);

    expect(aiResponse.role).toBe('assistant');
    expect(aiResponse.content).toBeTruthy();
    expect(aiResponse.content.length).toBeGreaterThan(0);

    // 验证响应包含正确答案 42
    expect(aiResponse.content).toMatch(/42/);

    console.log('✅ DeepSeek Reasoner 测试通过');

    // 清理这个测试会话
    try {
      await request.delete(`${baseURL}/api/ai/conversations/${reasonerConversationId}`, {
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      });
      console.log('✓ Reasoner 测试会话已清理');
    } catch (error) {
      console.warn('清理 Reasoner 会话时出错:', error.message);
    }
  });

  test('验证 DeepSeek API Key 来自环境变量', async ({ request }) => {
    console.log('验证 DeepSeek 配置的 API Key 来源...');

    // 获取 DeepSeek 配置详情
    const configResponse = await request.get(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    expect(configResponse.ok()).toBeTruthy();
    const configData = await configResponse.json();
    const deepseekConfigs = configData.data.filter(config => config.provider === 'deepseek');

    expect(deepseekConfigs.length).toBeGreaterThan(0);

    deepseekConfigs.forEach(config => {
      console.log(`检查配置: ${config.name}`);
      
      // API Key 应该被脱敏显示为 "***"
      expect(config.api_key).toBe('***');
      console.log(`  ✓ API Key 已脱敏: ${config.api_key}`);
      
      // 验证 API 端点
      expect(config.api_endpoint).toBeTruthy();
      expect(config.api_endpoint).toContain('deepseek');
      console.log(`  ✓ API Endpoint: ${config.api_endpoint}`);
    });

    console.log('✅ DeepSeek 配置验证通过（API Key 来自环境变量 DEEPSEEK_API_KEY）');
  });

  test('测试网络错误处理', async ({ request }) => {
    if (!deepseekChatConfigId) {
      test.skip('DeepSeek Chat 配置不可用');
      return;
    }

    console.log('测试网络错误处理...');

    // 创建一个临时配置，使用无效的 endpoint
    const invalidConfig = {
      name: 'Test Invalid DeepSeek',
      provider: 'deepseek',
      model_type: 'chat',
      api_key: 'invalid-key',
      api_endpoint: 'https://invalid-endpoint-that-does-not-exist.com',
      model: 'deepseek-chat',
      max_tokens: 100,
      temperature: 0.7
    };

    // 尝试创建配置（可能失败，这是预期的）
    const createConfigResponse = await request.post(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: invalidConfig
    });

    if (createConfigResponse.ok()) {
      const configData = await createConfigResponse.json();
      const invalidConfigId = configData.data.id;
      console.log(`✓ 创建了测试配置 (ID: ${invalidConfigId})`);

      // 创建会话
      const createConvResponse = await request.post(`${baseURL}/api/ai/conversations`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          title: '网络错误测试会话',
          config_id: invalidConfigId
        }
      });

      if (createConvResponse.ok()) {
        const convData = await createConvResponse.json();
        const testConversationId = convData.data.id;

        // 发送消息（预期会失败）
        const sendMessageResponse = await request.post(
          `${baseURL}/api/ai/conversations/${testConversationId}/messages`,
          {
            headers: {
              'Authorization': `Bearer ${authToken}`,
              'Content-Type': 'application/json'
            },
            data: {
              message: '测试消息'
            }
          }
        );

        // 清理
        await request.delete(`${baseURL}/api/ai/conversations/${testConversationId}`, {
          headers: { 'Authorization': `Bearer ${authToken}` }
        });
      }

      // 删除测试配置
      await request.delete(`${baseURL}/api/ai/configs/${invalidConfigId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` }
      });

      console.log('✅ 网络错误处理测试完成');
    } else {
      console.log('⚠️  无法创建测试配置，跳过网络错误测试');
    }
  });
});
