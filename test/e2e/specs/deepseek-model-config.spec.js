/**
 * DeepSeek 模型配置测试
 * 
 * 测试目标：
 * 1. 验证 DeepSeek 模型配置 API 返回正确数据
 * 2. 确保所有 DeepSeek 模型都有 model_type 字段
 * 3. 验证两个 DeepSeek 模型都存在且配置正确
 */

const { test, expect } = require('@playwright/test');

test.describe('DeepSeek 模型配置测试', () => {
  let baseURL;
  let authToken;

  test.beforeAll(async ({ request }) => {
    baseURL = process.env.BASE_URL || 'http://localhost:8080';
    console.log('测试基础 URL:', baseURL);

    // 登录获取 token
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
  });

  test('验证 DeepSeek API 返回正确的模型配置', async ({ request }) => {
    console.log('开始测试 DeepSeek 模型配置...');

    // 1. 调用 AI 配置 API (带认证)
    const response = await request.get(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });
    
    // 2. 验证响应状态
    if (!response.ok()) {
      const errorText = await response.text();
      console.error('API 错误响应:', errorText);
    }
    expect(response.ok()).toBeTruthy();
    expect(response.status()).toBe(200);

    // 3. 解析响应数据
    const responseData = await response.json();
    console.log('API 响应数据:', JSON.stringify(responseData, null, 2));

    // 4. 验证响应结构
    expect(responseData).toHaveProperty('data');
    expect(Array.isArray(responseData.data)).toBeTruthy();

    // 5. 过滤出 DeepSeek 模型
    const deepseekModels = responseData.data.filter(
      config => config.provider === 'deepseek'
    );

    console.log(`找到 ${deepseekModels.length} 个 DeepSeek 模型:`, 
      JSON.stringify(deepseekModels, null, 2));

    // 6. 验证至少有 2 个 DeepSeek 模型
    expect(deepseekModels.length).toBeGreaterThanOrEqual(2);

    // 7. 验证每个 DeepSeek 模型的配置
    deepseekModels.forEach((model, index) => {
      console.log(`验证 DeepSeek 模型 ${index + 1}:`, model.name);

      // 验证基本字段存在
      expect(model).toHaveProperty('id');
      expect(model).toHaveProperty('name');
      expect(model).toHaveProperty('provider');
      expect(model).toHaveProperty('model_type');
      expect(model).toHaveProperty('api_key');

      // 验证 provider 值
      expect(model.provider).toBe('deepseek');

      // 验证 model_type 不为空
      expect(model.model_type).not.toBe('');
      expect(model.model_type).not.toBeNull();
      expect(model.model_type).not.toBeUndefined();

      // 验证 model_type 是 'chat'
      expect(model.model_type).toBe('chat');

      console.log(`✓ ${model.name} - model_type: ${model.model_type}`);
    });

    // 8. 验证具体的两个模型
    const chatModel = deepseekModels.find(m => m.name.includes('Chat'));
    const reasonerModel = deepseekModels.find(m => m.name.includes('Reasoner'));

    expect(chatModel).toBeDefined();
    expect(reasonerModel).toBeDefined();

    console.log('✓ DeepSeek Chat 模型:', chatModel?.name);
    console.log('  - model_type:', chatModel?.model_type);
    console.log('  - API endpoint:', chatModel?.api_endpoint || chatModel?.base_url);

    console.log('✓ DeepSeek Reasoner 模型:', reasonerModel?.name);
    console.log('  - model_type:', reasonerModel?.model_type);
    console.log('  - API endpoint:', reasonerModel?.api_endpoint || reasonerModel?.base_url);
  });

  test('验证 DeepSeek 模型的详细配置', async ({ request }) => {
    console.log('验证 DeepSeek 模型详细配置...');

    // 1. 获取所有配置
    const response = await request.get(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });
    const responseData = await response.json();

    // 2. 找到 DeepSeek 模型
    const deepseekModels = responseData.data.filter(
      config => config.provider === 'deepseek'
    );

    // 3. 验证每个模型的详细配置
    for (const model of deepseekModels) {
      console.log(`验证模型: ${model.name}`);

      // 验证字段类型
      expect(typeof model.id).toBe('number');
      expect(typeof model.name).toBe('string');
      expect(typeof model.provider).toBe('string');
      expect(typeof model.model_type).toBe('string');

      // 验证 API 端点
      if (model.api_endpoint || model.base_url) {
        const endpoint = model.api_endpoint || model.base_url;
        expect(endpoint).toContain('deepseek');
        console.log(`  ✓ API endpoint: ${endpoint}`);
      }

      // 验证模型名称
      if (model.model) {
        expect(model.model).toMatch(/deepseek/i);
        console.log(`  ✓ Model: ${model.model}`);
      }

      // 验证状态
      console.log(`  ✓ Enabled: ${model.is_enabled}`);
      console.log(`  ✓ Default: ${model.is_default}`);
    }
  });

  test('验证 DeepSeek 模型 ID 和 model_type 的映射', async ({ request }) => {
    console.log('验证 model_type 字段映射...');

    const response = await request.get(`${baseURL}/api/ai/configs`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });
    const responseData = await response.json();

    const deepseekModels = responseData.data.filter(
      config => config.provider === 'deepseek'
    );

    // 创建 ID 到 model_type 的映射表
    const idToModelType = {};
    deepseekModels.forEach(model => {
      idToModelType[model.id] = {
        name: model.name,
        model_type: model.model_type
      };
    });

    console.log('DeepSeek 模型 ID 映射表:');
    console.log(JSON.stringify(idToModelType, null, 2));

    // 验证所有 model_type 都不为空
    Object.entries(idToModelType).forEach(([id, info]) => {
      console.log(`ID ${id}: ${info.name} -> model_type: "${info.model_type}"`);
      expect(info.model_type).not.toBe('');
      expect(info.model_type).toBe('chat');
    });
  });
});
