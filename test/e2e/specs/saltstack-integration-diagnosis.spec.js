// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * SaltStack 集成状态诊断测试
 * 诊断 /slurm 页面中 SaltStack 集成状态获取失败的问题
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';
const API_BASE_URL = process.env.API_BASE_URL || `${BASE_URL.replace(/\/$/, '')}/api`;
const ADMIN_USER = process.env.E2E_USER || 'admin';
const ADMIN_PASS = process.env.E2E_PASS || 'admin123';

// 登录获取 token
async function loginAsAdmin(request) {
  console.log('\n🔐 正在登录...');
  const response = await request.post(`${API_BASE_URL}/auth/login`, {
    data: {
      username: ADMIN_USER,
      password: ADMIN_PASS,
    },
  });

  expect(response.ok(), '登录应该成功').toBeTruthy();
  const body = await response.json();
  expect(body, '响应应该包含 token').toHaveProperty('token');
  console.log(`✅ 登录成功，Token: ${body.token.substring(0, 20)}...`);
  return body.token;
}

test.describe('SaltStack 集成状态诊断', () => {
  let authToken;

  test.beforeAll(async ({ request }) => {
    authToken = await loginAsAdmin(request);
  });

  test('1. 诊断 SaltStack 集成 API', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 1: 诊断 SaltStack 集成 API');
    console.log('========================================\n');

    // 测试 SaltStack 集成状态 API
    console.log('📊 调用 /api/slurm/saltstack/integration...');
    const response = await request.get(`${API_BASE_URL}/slurm/saltstack/integration`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
    });

    console.log(`   HTTP 状态码: ${response.status()}`);
    
    const body = await response.json().catch(async () => {
      const text = await response.text();
      return { error: '无法解析 JSON', raw: text };
    });

    console.log('\n📄 API 响应:');
    console.log(JSON.stringify(body, null, 2));

    if (!response.ok()) {
      console.log('\n❌ API 请求失败');
      return;
    }

    const data = body.data || body;
    console.log('\n🔍 响应数据分析:');
    console.log(`   enabled: ${data.enabled}`);
    console.log(`   master_status: ${data.master_status}`);
    console.log(`   api_status: ${data.api_status}`);
    console.log(`   demo: ${data.demo}`);
    
    if (data.minions) {
      console.log(`   minions.total: ${data.minions.total}`);
      console.log(`   minions.online: ${data.minions.online}`);
      console.log(`   minions.offline: ${data.minions.offline}`);
    }

    if (data.minion_list && Array.isArray(data.minion_list)) {
      console.log(`   minion_list 数量: ${data.minion_list.length}`);
      data.minion_list.forEach((minion, idx) => {
        console.log(`     ${idx + 1}. ${minion.id} (${minion.status})`);
      });
    }

    if (data.services) {
      console.log('\n📊 服务状态:');
      Object.entries(data.services).forEach(([service, status]) => {
        console.log(`   ${service}: ${status}`);
      });
    }

    // 检查是否是演示模式
    if (data.demo) {
      console.log('\n⚠️  当前处于演示模式，未连接真实 SaltStack');
    }
  });

  test('2. 测试 SaltStack 原始 API', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 2: 测试 SaltStack 原始 API');
    console.log('========================================\n');

    // 测试 /api/saltstack/status
    console.log('📊 调用 /api/saltstack/status...');
    const statusResponse = await request.get(`${API_BASE_URL}/saltstack/status`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
    });

    console.log(`   HTTP 状态码: ${statusResponse.status()}`);
    
    if (statusResponse.ok()) {
      const statusBody = await statusResponse.json();
      console.log('\n📄 SaltStack 状态:');
      console.log(JSON.stringify(statusBody, null, 2));
    }

    // 测试 /api/saltstack/minions
    console.log('\n📊 调用 /api/saltstack/minions...');
    const minionsResponse = await request.get(`${API_BASE_URL}/saltstack/minions`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
    });

    console.log(`   HTTP 状态码: ${minionsResponse.status()}`);
    
    if (minionsResponse.ok()) {
      const minionsBody = await minionsResponse.json();
      const minions = minionsBody.data || minionsBody.minions || [];
      console.log(`\n📄 Minions 数量: ${minions.length}`);
      minions.slice(0, 5).forEach((minion, idx) => {
        const id = minion.id || minion.minion_id || 'unknown';
        const status = minion.status || 'unknown';
        console.log(`   ${idx + 1}. ${id} (${status})`);
      });
    }
  });

  test('3. 检查 /slurm 页面显示', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试 3: 检查 /slurm 页面显示');
    console.log('========================================\n');

    // 登录
    console.log('🔐 通过浏览器登录...');
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle', { timeout: 15000 });

    const loginTabVisible = await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false);
    
    if (loginTabVisible) {
      await page.getByPlaceholder('用户名').fill(ADMIN_USER);
      await page.getByPlaceholder('密码').fill(ADMIN_PASS);
      await page.getByRole('button', { name: '登录' }).click();
      await page.waitForLoadState('networkidle', { timeout: 15000 });
      console.log('✅ 登录成功');
    }

    // 访问 /slurm 页面
    console.log('\n🌐 访问 /slurm 页面...');
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    await page.waitForTimeout(3000);

    // 截图
    const screenshotPath = 'test-screenshots/slurm-saltstack-integration.png';
    await page.screenshot({ path: screenshotPath, fullPage: true });
    console.log(`📸 页面截图: ${screenshotPath}`);

    // 检查页面内容
    const pageText = await page.textContent('body');
    
    console.log('\n🔍 页面内容检查:');
    
    // 检查 SaltStack 相关文本
    const keywords = [
      'SaltStack',
      'Master 状态',
      'API 状态',
      'Minion',
      '未知',
      'unavailable',
      'api_unavailable',
      '演示模式',
    ];

    keywords.forEach(keyword => {
      if (pageText.includes(keyword)) {
        console.log(`   ✓ 找到关键字: "${keyword}"`);
      }
    });

    // 检查网络请求
    console.log('\n📡 监听网络请求...');
    const requests = [];
    page.on('request', request => {
      if (request.url().includes('/api/')) {
        requests.push({
          url: request.url(),
          method: request.method(),
        });
      }
    });

    // 刷新页面重新加载
    await page.reload();
    await page.waitForTimeout(5000);

    console.log('\n📊 API 请求记录:');
    const saltStackRequests = requests.filter(r => 
      r.url.includes('saltstack') || r.url.includes('slurm')
    );
    
    saltStackRequests.forEach((req, idx) => {
      console.log(`   ${idx + 1}. ${req.method} ${req.url}`);
    });

    if (saltStackRequests.length === 0) {
      console.log('   ⚠️  未发现 SaltStack 相关 API 请求');
    }
  });

  test('4. 检查后端日志', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 4: 分析问题根因');
    console.log('========================================\n');

    console.log('🔍 问题分析：');
    console.log('');
    console.log('可能的原因：');
    console.log('1. SaltStack 服务未启动或配置错误');
    console.log('2. Backend 无法连接到 SaltStack API');
    console.log('3. 前端 API 调用失败或数据格式不匹配');
    console.log('4. 认证或权限问题');
    console.log('');
    console.log('建议检查：');
    console.log('1. docker-compose logs saltstack --tail=50');
    console.log('2. docker-compose logs backend --tail=50 | grep -i salt');
    console.log('3. docker-compose exec saltstack salt-master --version');
    console.log('4. docker-compose exec saltstack salt-key -L');
  });
});
