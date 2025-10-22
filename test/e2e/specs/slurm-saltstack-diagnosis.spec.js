/**
 * SLURM 和 SaltStack 状态诊断测试
 * 
 * 测试目标:
 * 1. 检查 /slurm 页面的 SaltStack 状态显示
 * 2. 诊断 SLURM 集群状态同步问题
 * 3. 捕获所有相关 API 调用和错误
 */

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// 从环境变量获取基础URL，默认使用 192.168.0.200:8080
const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('SLURM SaltStack 状态诊断', () => {
  let context;
  let page;
  let apiCalls = [];
  let consoleMessages = [];
  let pageErrors = [];

  test.beforeAll(async ({ browser }) => {
    context = await browser.newContext({
      ignoreHTTPSErrors: true,
      viewport: { width: 1920, height: 1080 }
    });

    page = await context.newPage();

    // 监听所有 API 请求
    page.on('request', request => {
      const url = request.url();
      if (url.includes('/api/') || url.includes('/slurm') || url.includes('/saltstack')) {
        apiCalls.push({
          timestamp: new Date().toISOString(),
          method: request.method(),
          url: url,
          headers: request.headers(),
          postData: request.postData()
        });
      }
    });

    // 监听所有 API 响应
    page.on('response', async response => {
      const url = response.url();
      if (url.includes('/api/') || url.includes('/slurm') || url.includes('/saltstack')) {
        let responseBody = null;
        try {
          const contentType = response.headers()['content-type'] || '';
          if (contentType.includes('application/json')) {
            responseBody = await response.json();
          } else {
            responseBody = await response.text();
          }
        } catch (e) {
          responseBody = `Failed to parse: ${e.message}`;
        }

        const callIndex = apiCalls.findIndex(call => call.url === url);
        if (callIndex !== -1) {
          apiCalls[callIndex].status = response.status();
          apiCalls[callIndex].statusText = response.statusText();
          apiCalls[callIndex].responseHeaders = response.headers();
          apiCalls[callIndex].responseBody = responseBody;
        }
      }
    });

    // 监听控制台消息
    page.on('console', msg => {
      consoleMessages.push({
        timestamp: new Date().toISOString(),
        type: msg.type(),
        text: msg.text()
      });
    });

    // 监听页面错误
    page.on('pageerror', error => {
      pageErrors.push({
        timestamp: new Date().toISOString(),
        message: error.message,
        stack: error.stack
      });
    });
  });

  test.afterAll(async () => {
    // 保存诊断报告
    const reportDir = path.join(__dirname, '../../test-results/slurm-diagnosis');
    if (!fs.existsSync(reportDir)) {
      fs.mkdirSync(reportDir, { recursive: true });
    }

    const report = {
      timestamp: new Date().toISOString(),
      baseUrl: BASE_URL,
      apiCalls: apiCalls,
      consoleMessages: consoleMessages,
      pageErrors: pageErrors
    };

    fs.writeFileSync(
      path.join(reportDir, 'slurm-saltstack-diagnosis.json'),
      JSON.stringify(report, null, 2)
    );

    console.log('\n========================================');
    console.log('SLURM SaltStack 诊断报告');
    console.log('========================================\n');

    console.log('📊 API 调用统计:');
    console.log(`  总计: ${apiCalls.length} 个请求`);
    
    const failedCalls = apiCalls.filter(call => call.status >= 400);
    if (failedCalls.length > 0) {
      console.log(`  ❌ 失败: ${failedCalls.length} 个`);
      failedCalls.forEach(call => {
        console.log(`     • ${call.method} ${call.url}`);
        console.log(`       状态: ${call.status} ${call.statusText}`);
        if (call.responseBody) {
          console.log(`       响应: ${JSON.stringify(call.responseBody).substring(0, 200)}`);
        }
      });
    }

    const saltStackCalls = apiCalls.filter(call => call.url.includes('saltstack'));
    if (saltStackCalls.length > 0) {
      console.log(`\n🧂 SaltStack 相关请求: ${saltStackCalls.length} 个`);
      saltStackCalls.forEach(call => {
        console.log(`  ${call.method} ${call.url}`);
        console.log(`  状态: ${call.status} ${call.statusText}`);
        if (call.responseBody) {
          const body = typeof call.responseBody === 'string' 
            ? call.responseBody.substring(0, 300)
            : JSON.stringify(call.responseBody).substring(0, 300);
          console.log(`  响应: ${body}`);
        }
      });
    }

    const slurmCalls = apiCalls.filter(call => call.url.includes('slurm') && !call.url.includes('slurm-diagnosis'));
    if (slurmCalls.length > 0) {
      console.log(`\n🖥️  SLURM 相关请求: ${slurmCalls.length} 个`);
      slurmCalls.forEach(call => {
        console.log(`  ${call.method} ${call.url}`);
        console.log(`  状态: ${call.status} ${call.statusText}`);
        if (call.responseBody) {
          const body = typeof call.responseBody === 'string' 
            ? call.responseBody.substring(0, 300)
            : JSON.stringify(call.responseBody).substring(0, 300);
          console.log(`  响应: ${body}`);
        }
      });
    }

    if (pageErrors.length > 0) {
      console.log(`\n❌ 页面错误: ${pageErrors.length} 个`);
      pageErrors.forEach(error => {
        console.log(`  ${error.message}`);
      });
    }

    console.log('\n========================================\n');

    await context.close();
  });

  test('Step 1: 登录系统', async () => {
    console.log(`\n🔐 正在登录 ${BASE_URL}...`);
    
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    // 截图：登录页面
    await page.screenshot({ 
      path: path.join(__dirname, '../../test-screenshots/slurm-01-login.png'),
      fullPage: true 
    });

    // 填写登录表单
    await page.fill('input[name="username"], input[type="text"]', 'admin');
    await page.fill('input[name="password"], input[type="password"]', 'admin123');
    
    // 点击登录按钮
    await page.click('button[type="submit"], button:has-text("登录"), button:has-text("Login")');
    
    // 等待登录完成
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 验证登录成功
    const currentUrl = page.url();
    console.log(`✅ 登录成功，当前URL: ${currentUrl}`);
    
    // 截图：登录后页面
    await page.screenshot({ 
      path: path.join(__dirname, '../../test-screenshots/slurm-02-after-login.png'),
      fullPage: true 
    });
  });

  test('Step 2: 导航到 SLURM 页面', async () => {
    console.log('\n🔍 导航到 SLURM 页面...');
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000); // 等待页面加载
    
    // 截图：SLURM 页面
    await page.screenshot({ 
      path: path.join(__dirname, '../../test-screenshots/slurm-03-page.png'),
      fullPage: true 
    });
    
    console.log('✅ SLURM 页面加载完成');
  });

  test('Step 3: 检查 SaltStack 状态显示', async () => {
    console.log('\n🧂 检查 SaltStack 状态显示...');
    
    // 查找 SaltStack 状态元素
    const saltStackStatus = await page.locator('text=/SaltStack.*状态/i, text=/集群管理/i').first();
    
    if (await saltStackStatus.count() > 0) {
      const statusText = await saltStackStatus.textContent();
      console.log(`  找到 SaltStack 状态: ${statusText}`);
      
      // 检查是否显示"未知"或"异常"
      if (statusText.includes('未知') || statusText.includes('异常') || statusText.includes('unknown')) {
        console.log('  ⚠️  SaltStack 状态显示为未知/异常');
      } else if (statusText.includes('正常') || statusText.includes('健康') || statusText.includes('running')) {
        console.log('  ✅ SaltStack 状态正常');
      }
    } else {
      console.log('  ⚠️  未找到 SaltStack 状态显示元素');
    }
    
    // 截图：状态区域
    await page.screenshot({ 
      path: path.join(__dirname, '../../test-screenshots/slurm-04-saltstack-status.png'),
      fullPage: true 
    });
  });

  test('Step 4: 检查 SLURM 集群节点状态', async () => {
    console.log('\n🖥️  检查 SLURM 集群节点状态...');
    
    // 查找节点列表或状态表格
    const nodeTable = await page.locator('table, .node-list, [class*="node"], [class*="cluster"]');
    
    if (await nodeTable.count() > 0) {
      console.log(`  找到 ${await nodeTable.count()} 个节点相关元素`);
      
      // 获取页面文本内容
      const pageText = await page.textContent('body');
      
      // 检查是否有节点信息
      if (pageText.includes('compute') || pageText.includes('node') || pageText.includes('节点')) {
        console.log('  ✅ 找到节点信息');
        
        // 检查节点状态
        if (pageText.includes('idle') || pageText.includes('空闲')) {
          console.log('  ✅ 有空闲节点');
        }
        if (pageText.includes('down') || pageText.includes('离线')) {
          console.log('  ⚠️  有离线节点');
        }
        if (pageText.includes('drain') || pageText.includes('维护')) {
          console.log('  ⚠️  有维护中节点');
        }
      } else {
        console.log('  ⚠️  未找到节点信息');
      }
    } else {
      console.log('  ⚠️  未找到节点列表元素');
    }
    
    // 截图：节点状态
    await page.screenshot({ 
      path: path.join(__dirname, '../../test-screenshots/slurm-05-nodes.png'),
      fullPage: true 
    });
  });

  test('Step 5: 直接测试 SaltStack API', async () => {
    console.log('\n🔬 直接测试 SaltStack API...');
    
    // 测试 SaltStack 状态 API
    const saltStackStatusUrl = `${BASE_URL}/api/saltstack/status`;
    console.log(`  测试: ${saltStackStatusUrl}`);
    
    try {
      const response = await page.request.get(saltStackStatusUrl);
      console.log(`  状态码: ${response.status()}`);
      
      if (response.ok()) {
        const data = await response.json();
        console.log('  响应数据:', JSON.stringify(data, null, 2));
        
        if (data.status === 'unknown' || data.status === 'error') {
          console.log('  ❌ SaltStack API 返回未知/错误状态');
          if (data.error || data.message) {
            console.log(`  错误信息: ${data.error || data.message}`);
          }
        } else {
          console.log('  ✅ SaltStack API 响应正常');
        }
      } else {
        const text = await response.text();
        console.log(`  ❌ API 请求失败: ${text.substring(0, 500)}`);
      }
    } catch (error) {
      console.log(`  ❌ API 请求异常: ${error.message}`);
    }
  });

  test('Step 6: 直接测试 SLURM 节点 API', async () => {
    console.log('\n🔬 直接测试 SLURM 节点 API...');
    
    // 测试 SLURM 节点 API
    const slurmNodesUrl = `${BASE_URL}/api/slurm/nodes`;
    console.log(`  测试: ${slurmNodesUrl}`);
    
    try {
      const response = await page.request.get(slurmNodesUrl);
      console.log(`  状态码: ${response.status()}`);
      
      if (response.ok()) {
        const data = await response.json();
        console.log('  响应数据:', JSON.stringify(data, null, 2).substring(0, 1000));
        
        if (data.nodes && Array.isArray(data.nodes)) {
          console.log(`  ✅ 找到 ${data.nodes.length} 个节点`);
          data.nodes.forEach((node, index) => {
            if (index < 5) { // 只显示前5个
              console.log(`    节点 ${index + 1}: ${node.name || node.hostname} - 状态: ${node.state || 'unknown'}`);
            }
          });
        } else {
          console.log('  ⚠️  节点数据格式异常或为空');
        }
      } else {
        const text = await response.text();
        console.log(`  ❌ API 请求失败: ${text.substring(0, 500)}`);
      }
    } catch (error) {
      console.log(`  ❌ API 请求异常: ${error.message}`);
    }
  });

  test('Step 7: 测试 SLURM 集群信息 API', async () => {
    console.log('\n🔬 直接测试 SLURM 集群信息 API...');
    
    const slurmInfoUrl = `${BASE_URL}/api/slurm/info`;
    console.log(`  测试: ${slurmInfoUrl}`);
    
    try {
      const response = await page.request.get(slurmInfoUrl);
      console.log(`  状态码: ${response.status()}`);
      
      if (response.ok()) {
        const data = await response.json();
        console.log('  响应数据:', JSON.stringify(data, null, 2).substring(0, 1000));
      } else {
        const text = await response.text();
        console.log(`  ❌ API 请求失败: ${text.substring(0, 500)}`);
      }
    } catch (error) {
      console.log(`  ❌ API 请求异常: ${error.message}`);
    }
  });

  test('Step 8: 检查后端日志中的错误', async () => {
    console.log('\n📋 分析捕获的 API 调用和错误...');
    
    // 分析 SaltStack 相关调用
    const saltStackCalls = apiCalls.filter(call => call.url.includes('saltstack'));
    if (saltStackCalls.length > 0) {
      console.log(`\n  SaltStack API 调用 (${saltStackCalls.length} 个):`);
      saltStackCalls.forEach(call => {
        const isError = call.status >= 400;
        const icon = isError ? '❌' : '✅';
        console.log(`    ${icon} ${call.method} ${call.url}`);
        console.log(`       状态: ${call.status} ${call.statusText}`);
        
        if (call.responseBody) {
          const body = typeof call.responseBody === 'object' 
            ? JSON.stringify(call.responseBody)
            : call.responseBody;
          console.log(`       响应: ${body.substring(0, 200)}`);
        }
      });
    }
    
    // 分析 SLURM 相关调用
    const slurmCalls = apiCalls.filter(call => 
      call.url.includes('slurm') && !call.url.includes('slurm-diagnosis')
    );
    if (slurmCalls.length > 0) {
      console.log(`\n  SLURM API 调用 (${slurmCalls.length} 个):`);
      slurmCalls.forEach(call => {
        const isError = call.status >= 400;
        const icon = isError ? '❌' : '✅';
        console.log(`    ${icon} ${call.method} ${call.url}`);
        console.log(`       状态: ${call.status} ${call.statusText}`);
        
        if (call.responseBody) {
          const body = typeof call.responseBody === 'object' 
            ? JSON.stringify(call.responseBody)
            : call.responseBody;
          console.log(`       响应: ${body.substring(0, 200)}`);
        }
      });
    }
    
    // 显示控制台错误
    const errors = consoleMessages.filter(msg => msg.type === 'error');
    if (errors.length > 0) {
      console.log(`\n  控制台错误 (${errors.length} 个):`);
      errors.forEach(error => {
        console.log(`    ❌ ${error.text}`);
      });
    }
    
    // 显示页面错误
    if (pageErrors.length > 0) {
      console.log(`\n  页面错误 (${pageErrors.length} 个):`);
      pageErrors.forEach(error => {
        console.log(`    ❌ ${error.message}`);
      });
    }
  });
});
