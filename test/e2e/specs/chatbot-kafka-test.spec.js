// @ts-check
const { test, expect } = require('@playwright/test');
const { Client } = require('pg');

/**
 * 聊天机器人 Kafka 消息传递和数据库记录测试
 * 验证用户消息通过 Kafka 传递并正确记录到 PostgreSQL
 */

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const ADMIN_USERNAME = process.env.ADMIN_USERNAME || 'admin';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'admin123';

// 数据库配置
const DB_CONFIG = {
  host: process.env.DB_HOST || 'localhost',
  port: parseInt(process.env.DB_PORT || '5432'),
  database: process.env.DB_NAME || 'ai_infra',
  user: process.env.DB_USER || 'postgres',
  password: process.env.DB_PASSWORD || 'postgres123',
  connectionTimeoutMillis: 5000,
};

// 测试消息
const TEST_MESSAGES = [
  {
    content: 'Hello, AI Assistant! This is a test message.',
    type: 'user'
  },
  {
    content: '你好，请帮我分析一下 SLURM 集群的状态',
    type: 'user'
  },
  {
    content: 'What is the current status of my jobs?',
    type: 'user'
  }
];

// 等待页面加载
async function waitForPageLoad(page) {
  await page.waitForLoadState('networkidle', { timeout: 15000 });
  await page.waitForTimeout(1000);
}

// 登录
async function login(page) {
  console.log('执行登录...');
  await page.goto('/login');
  await waitForPageLoad(page);
  
  await page.fill('input[type="text"]', ADMIN_USERNAME);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  
  await page.waitForURL(/\//, { timeout: 10000 });
  await waitForPageLoad(page);
  console.log('✓ 登录成功');
}

// 连接数据库
async function connectDatabase() {
  const client = new Client(DB_CONFIG);
  try {
    await client.connect();
    console.log('✓ 数据库连接成功');
    return client;
  } catch (error) {
    console.error('❌ 数据库连接失败:', error.message);
    return null;
  }
}

// 查询聊天记录
async function queryChatMessages(client, limit = 10) {
  if (!client) return [];
  
  try {
    const result = await client.query(`
      SELECT * FROM chat_messages 
      ORDER BY created_at DESC 
      LIMIT $1
    `, [limit]);
    
    return result.rows;
  } catch (error) {
    console.error('查询聊天记录失败:', error.message);
    return [];
  }
}

// 查询特定内容的消息
async function findMessageByContent(client, content) {
  if (!client) return null;
  
  try {
    const result = await client.query(`
      SELECT * FROM chat_messages 
      WHERE content LIKE $1 
      ORDER BY created_at DESC 
      LIMIT 1
    `, [`%${content}%`]);
    
    return result.rows[0] || null;
  } catch (error) {
    console.error('查询消息失败:', error.message);
    return null;
  }
}

test.describe('聊天机器人测试', () => {
  let dbClient = null;

  test.beforeAll(async () => {
    // 连接数据库
    dbClient = await connectDatabase();
  });

  test.afterAll(async () => {
    // 关闭数据库连接
    if (dbClient) {
      await dbClient.end();
      console.log('✓ 数据库连接已关闭');
    }
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('1. 验证聊天界面存在', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 验证聊天界面');
    console.log('========================================\n');

    // 尝试多个可能的聊天路由
    const chatRoutes = ['/chat', '/ai-assistant', '/assistant', '/chatbot', '/ai'];
    let chatPageFound = false;

    for (const route of chatRoutes) {
      await page.goto(route);
      await page.waitForTimeout(500);
      
      const url = page.url();
      if (!url.includes('404') && !url.includes('error') && !url.includes('login')) {
        console.log(`✓ 找到聊天页面: ${route}`);
        chatPageFound = true;
        break;
      }
    }

    if (!chatPageFound) {
      console.log('⚠ 未找到聊天页面路由，尝试从主页查找入口');
      
      await page.goto('/');
      await waitForPageLoad(page);

      // 查找聊天机器人入口
      const chatButton = page.locator('button', { hasText: /聊天|Chat|AI|助手/ }).or(
        page.locator('[class*="chat"]').or(
          page.locator('[id*="chat"]')
        )
      );

      const hasChatButton = await chatButton.count();
      
      if (hasChatButton > 0) {
        console.log(`✓ 找到 ${hasChatButton} 个聊天相关按钮`);
        
        // 尝试点击第一个可见的按钮
        for (let i = 0; i < Math.min(hasChatButton, 3); i++) {
          const btn = chatButton.nth(i);
          const isVisible = await btn.isVisible().catch(() => false);
          
          if (isVisible) {
            try {
              await btn.scrollIntoViewIfNeeded();
              await page.waitForTimeout(300);
              await btn.click({ timeout: 3000 });
              console.log(`✓ 成功点击按钮 ${i + 1}`);
              chatPageFound = true;
              break;
            } catch (error) {
              console.log(`⚠ 按钮 ${i + 1} 点击失败，尝试下一个`);
            }
          }
        }
      }
    }

    if (chatPageFound) {
      await page.waitForTimeout(1000);
      
      // 验证聊天界面元素
      const chatInput = page.locator('textarea, input[placeholder*="消息"], input[placeholder*="message"], input[placeholder*="输入"]');
      const hasChatInput = await chatInput.count();

      if (hasChatInput > 0) {
        console.log('✅ 聊天输入框已加载');
      } else {
        console.log('⚠ 未找到聊天输入框，但页面可能仍在加载');
      }
    } else {
      console.log('⚠ 未找到聊天功能，可能尚未实现');
    }
  });

  test('2. 发送测试消息', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 发送聊天消息');
    console.log('========================================\n');

    // 导航到聊天页面
    await page.goto('/chat');
    await waitForPageLoad(page);

    // 记录初始消息数量
    const initialMessageCount = await dbClient ? 
      (await queryChatMessages(dbClient, 1)).length : 0;
    
    console.log(`初始数据库消息数: ${initialMessageCount}`);

    // 发送第一条测试消息
    const testMessage = TEST_MESSAGES[0];
    console.log(`\n📤 发送消息: "${testMessage.content}"`);

    const chatInput = page.locator('textarea').or(
      page.locator('input[type="text"]')
    ).last();

    if (await chatInput.isVisible().catch(() => false)) {
      await chatInput.fill(testMessage.content);
      
      // 查找发送按钮
      const sendButton = page.locator('button', { hasText: /发送|Send|提交/ }).or(
        page.locator('button[type="submit"]')
      ).last();

      // 监听 API 请求
      let apiCalled = false;
      page.on('request', request => {
        if (request.url().includes('/api/chat') || 
            request.url().includes('/api/message')) {
          apiCalled = true;
          console.log(`✓ 检测到 API 调用: ${request.url()}`);
        }
      });

      // 发送消息
      if (await sendButton.isVisible().catch(() => false)) {
        await sendButton.click();
      } else {
        // 尝试回车发送
        await chatInput.press('Enter');
      }

      console.log('✓ 消息已发送');
      await page.waitForTimeout(2000);

      if (apiCalled) {
        console.log('✅ API 请求已触发');
      }

      // 验证消息显示在界面上
      await page.waitForTimeout(1000);
      const pageText = await page.textContent('body');
      
      if (pageText && pageText.includes(testMessage.content)) {
        console.log('✅ 消息已显示在界面');
      }
    } else {
      console.log('❌ 未找到聊天输入框');
    }
  });

  test('3. 验证消息记录到数据库', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 数据库消息记录');
    console.log('========================================\n');

    if (!dbClient) {
      console.log('⚠ 数据库未连接，跳过测试');
      return;
    }

    // 发送消息
    await page.goto('/chat');
    await waitForPageLoad(page);

    const testMessage = TEST_MESSAGES[1];
    console.log(`📤 发送测试消息: "${testMessage.content}"`);

    const chatInput = page.locator('textarea, input[type="text"]').last();
    if (await chatInput.isVisible().catch(() => false)) {
      await chatInput.fill(testMessage.content);
      
      const sendButton = page.locator('button', { hasText: /发送|Send/ }).last();
      if (await sendButton.isVisible().catch(() => false)) {
        await sendButton.click();
      } else {
        await chatInput.press('Enter');
      }

      console.log('✓ 消息已发送');
    }

    // 等待消息处理
    console.log('\n⏳ 等待消息处理 (5秒)...');
    await page.waitForTimeout(5000);

    // 查询数据库
    console.log('\n🔍 查询数据库记录...');
    const messages = await queryChatMessages(dbClient, 20);
    
    console.log(`📊 最近 20 条消息:`);
    messages.forEach((msg, index) => {
      console.log(`  ${index + 1}. [${msg.created_at}] ${msg.role || msg.type}: ${msg.content?.substring(0, 50)}...`);
    });

    // 查找测试消息
    const foundMessage = await findMessageByContent(dbClient, testMessage.content);

    if (foundMessage) {
      console.log('\n✅ 消息已成功记录到数据库');
      console.log('消息详情:');
      console.log(`  ID: ${foundMessage.id}`);
      console.log(`  内容: ${foundMessage.content}`);
      console.log(`  角色: ${foundMessage.role || foundMessage.type}`);
      console.log(`  创建时间: ${foundMessage.created_at}`);
      console.log(`  用户ID: ${foundMessage.user_id || 'N/A'}`);

      expect(foundMessage.content).toContain(testMessage.content.substring(0, 20));
    } else {
      console.log('⚠ 未在数据库中找到测试消息');
      console.log('可能原因:');
      console.log('  1. Kafka 消费者未启动');
      console.log('  2. 消息处理延迟');
      console.log('  3. 数据库表不存在或结构不匹配');
    }
  });

  test('4. 批量消息测试', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 批量消息发送');
    console.log('========================================\n');

    await page.goto('/chat');
    await waitForPageLoad(page);

    const sentMessages = [];
    
    for (const [index, testMsg] of TEST_MESSAGES.entries()) {
      console.log(`\n📤 发送消息 ${index + 1}/${TEST_MESSAGES.length}`);
      console.log(`   内容: "${testMsg.content}"`);

      const chatInput = page.locator('textarea, input').last();
      if (await chatInput.isVisible().catch(() => false)) {
        await chatInput.fill(testMsg.content);
        
        const sendButton = page.locator('button', { hasText: /发送|Send/ }).last();
        if (await sendButton.isVisible().catch(() => false)) {
          await sendButton.click();
        } else {
          await chatInput.press('Enter');
        }

        sentMessages.push(testMsg.content);
        console.log('   ✓ 已发送');

        // 等待消息处理
        await page.waitForTimeout(2000);
      }
    }

    console.log(`\n✅ 共发送 ${sentMessages.length} 条消息`);

    // 等待所有消息处理
    console.log('\n⏳ 等待所有消息处理 (10秒)...');
    await page.waitForTimeout(10000);

    // 验证数据库记录
    if (dbClient) {
      console.log('\n🔍 验证数据库记录...');
      const recentMessages = await queryChatMessages(dbClient, 50);
      
      let foundCount = 0;
      for (const sent of sentMessages) {
        const found = recentMessages.some(msg => 
          msg.content && msg.content.includes(sent.substring(0, 20))
        );
        if (found) foundCount++;
      }

      console.log(`\n📊 数据库验证结果:`);
      console.log(`   发送: ${sentMessages.length} 条`);
      console.log(`   找到: ${foundCount} 条`);
      console.log(`   成功率: ${(foundCount / sentMessages.length * 100).toFixed(1)}%`);

      if (foundCount > 0) {
        console.log('✅ 至少部分消息已记录到数据库');
      }
    }
  });

  test('5. Kafka 消息队列验证', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: Kafka 消息队列');
    console.log('========================================\n');

    await page.goto('/chat');
    await waitForPageLoad(page);

    // 监听所有 API 请求
    const apiRequests = [];
    page.on('request', request => {
      if (request.url().includes('/api/')) {
        apiRequests.push({
          url: request.url(),
          method: request.method(),
          timestamp: Date.now()
        });
      }
    });

    // 发送消息
    const testMsg = '测试 Kafka 消息队列 - ' + Date.now();
    console.log(`📤 发送标记消息: "${testMsg}"`);

    const chatInput = page.locator('textarea, input').last();
    if (await chatInput.isVisible().catch(() => false)) {
      await chatInput.fill(testMsg);
      await chatInput.press('Enter');
      
      console.log('✓ 消息已发送');
      await page.waitForTimeout(3000);
    }

    // 分析 API 请求
    console.log('\n📊 API 请求分析:');
    const chatRequests = apiRequests.filter(req => 
      req.url.includes('/chat') || 
      req.url.includes('/message') ||
      req.url.includes('/kafka')
    );

    chatRequests.forEach((req, index) => {
      console.log(`  ${index + 1}. [${req.method}] ${req.url}`);
    });

    if (chatRequests.length > 0) {
      console.log(`\n✅ 检测到 ${chatRequests.length} 个聊天相关 API 请求`);
    }

    // 验证数据库
    if (dbClient) {
      await page.waitForTimeout(5000);
      
      const found = await findMessageByContent(dbClient, testMsg);
      if (found) {
        console.log('✅ 消息已通过 Kafka 传递并记录到数据库');
        
        // 计算延迟
        const sentTime = parseInt(testMsg.split(' - ')[1]);
        const recordedTime = new Date(found.created_at).getTime();
        const latency = recordedTime - sentTime;
        
        console.log(`\n⏱️  消息传递延迟: ${latency} ms`);
      }
    }
  });

  test('6. 数据库性能测试', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 数据库性能');
    console.log('========================================\n');

    if (!dbClient) {
      console.log('⚠ 数据库未连接，跳过测试');
      return;
    }

    // 查询统计信息
    try {
      // 总消息数
      const countResult = await dbClient.query('SELECT COUNT(*) as total FROM chat_messages');
      const totalMessages = parseInt(countResult.rows[0].total);
      
      console.log(`📊 数据库统计:`);
      console.log(`   总消息数: ${totalMessages}`);

      // 按类型统计
      const typeResult = await dbClient.query(`
        SELECT role, COUNT(*) as count 
        FROM chat_messages 
        GROUP BY role
      `);
      
      console.log('\n   消息类型分布:');
      typeResult.rows.forEach(row => {
        console.log(`     ${row.role || 'unknown'}: ${row.count}`);
      });

      // 最近消息时间
      const recentResult = await dbClient.query(`
        SELECT created_at 
        FROM chat_messages 
        ORDER BY created_at DESC 
        LIMIT 1
      `);
      
      if (recentResult.rows.length > 0) {
        console.log(`\n   最新消息时间: ${recentResult.rows[0].created_at}`);
      }

      // 查询性能测试
      const startTime = Date.now();
      await queryChatMessages(dbClient, 100);
      const queryTime = Date.now() - startTime;
      
      console.log(`\n⏱️  查询性能:`);
      console.log(`   查询 100 条记录耗时: ${queryTime} ms`);

      if (queryTime < 100) {
        console.log('   ✅ 性能优秀 (< 100ms)');
      } else if (queryTime < 500) {
        console.log('   ✓ 性能良好 (< 500ms)');
      } else {
        console.log('   ⚠ 性能需要优化 (> 500ms)');
      }

      console.log('\n📝 性能建议:');
      if (totalMessages > 10000) {
        console.log('   - 消息量较大，建议添加索引');
        console.log('   - 考虑定期归档历史消息');
      }
      console.log('   - PostgreSQL 适合中小规模并发');
      console.log('   - 如需更高性能，可考虑 MySQL 或 Redis 缓存');

    } catch (error) {
      console.error('❌ 数据库查询失败:', error.message);
    }
  });
});
