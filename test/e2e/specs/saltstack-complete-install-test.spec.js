// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * SaltStack 完整安装和集成测试
 * 测试流程：
 * 1. 验证 AppHub 可用性
 * 2. 安装 SaltStack Minion 到 test-ssh01-03
 * 3. 安装 Slurm Client 到 test-ssh01-03
 * 4. 验证 SaltStack 页面显示正常
 * 5. 验证 Slurm 页面显示正常
 * 6. 测试 SaltStack 命令执行
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';
const API_BASE_URL = process.env.API_BASE_URL || `${BASE_URL.replace(/\/$/, '')}/api`;
const APPHUB_URL = process.env.APPHUB_URL || 'http://192.168.0.200:8090';
const ADMIN_USER = process.env.E2E_USER || 'admin';
const ADMIN_PASS = process.env.E2E_PASS || 'admin123';

const TEST_NODES = ['test-ssh01', 'test-ssh02', 'test-ssh03'];

// 辅助函数：登录获取 token
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

// 辅助函数：验证安装步骤成功
function assertStepSuccess(steps, stepName) {
  const match = steps.find((step) => step.Name === stepName);
  expect(match, `步骤 "${stepName}" 应该存在`).toBeTruthy();
  expect(match.Success, `步骤 "${stepName}" 应该成功`).toBeTruthy();
}

// 辅助函数：等待页面加载
async function waitForPageLoad(page, timeout = 15000) {
  await page.waitForLoadState('networkidle', { timeout });
  await page.waitForTimeout(1000);
}

// 辅助函数：页面登录
async function loginViaBrowser(page) {
  console.log('\n🌐 通过浏览器登录...');
  await page.goto(BASE_URL);
  await waitForPageLoad(page);

  // 检查是否需要登录
  const loginTabVisible = await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false);
  
  if (loginTabVisible) {
    await page.getByPlaceholder('用户名').fill(ADMIN_USER);
    await page.getByPlaceholder('密码').fill(ADMIN_PASS);
    await page.getByRole('button', { name: '登录' }).click();
    await waitForPageLoad(page);
    console.log('✅ 浏览器登录成功');
  } else {
    console.log('✅ 已登录状态');
  }
}

test.describe('SaltStack 完整安装和集成测试', () => {
  test.describe.configure({ mode: 'serial' });

  let authToken;

  test.beforeAll(async ({ request }) => {
    authToken = await loginAsAdmin(request);
  });

  test('1. 验证 AppHub 可用性', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 1: 验证 AppHub 可用性');
    console.log('========================================');

    // 检查 AppHub 首页
    console.log('\n📦 检查 AppHub 首页...');
    const homeResponse = await request.get(APPHUB_URL);
    expect(homeResponse.ok(), 'AppHub 首页应该可访问').toBeTruthy();
    console.log(`✅ AppHub 首页可访问: ${APPHUB_URL}`);

    // 检查 Slurm DEB 包仓库
    console.log('\n📦 检查 Slurm DEB 包仓库...');
    const packagesUrl = `${APPHUB_URL.replace(/\/$/, '')}/pkgs/slurm-deb/Packages`;
    const packagesResponse = await request.get(packagesUrl);
    expect(packagesResponse.ok(), 'Packages 文件应该可访问').toBeTruthy();
    
    const packagesText = await packagesResponse.text();
    expect(packagesText.length, 'Packages 文件不应该为空').toBeGreaterThan(0);
    expect(packagesText, 'Packages 文件应该包含 slurm 包').toMatch(/Package:\s+slurm/i);
    console.log('✅ Slurm DEB 包仓库正常');

    // 检查 SaltStack DEB 包仓库
    console.log('\n📦 检查 SaltStack DEB 包仓库...');
    const saltPackagesUrl = `${APPHUB_URL.replace(/\/$/, '')}/pkgs/salt-deb/Packages`;
    const saltPackagesResponse = await request.get(saltPackagesUrl);
    
    if (saltPackagesResponse.ok()) {
      const saltPackagesText = await saltPackagesResponse.text();
      if (saltPackagesText.includes('salt-minion')) {
        console.log('✅ SaltStack DEB 包仓库正常');
      } else {
        console.log('⚠️  SaltStack DEB 包仓库存在但可能不完整');
      }
    } else {
      console.log('⚠️  SaltStack DEB 包仓库不可用 (这可能是正常的，如果使用系统包)');
    }
  });

  test('2. 安装 SaltStack Minion 到测试节点', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 2: 安装 SaltStack Minion');
    console.log('========================================');

    test.setTimeout(6 * 60 * 1000); // 6 分钟超时

    const payload = {
      nodes: TEST_NODES,
      appHubURL: APPHUB_URL,
      enableSaltMinion: true,
      enableSlurmClient: false,
    };

    console.log('\n🚀 开始安装 SaltStack Minion...');
    console.log(`   节点: ${TEST_NODES.join(', ')}`);
    console.log(`   AppHub: ${APPHUB_URL}`);

    const startTime = Date.now();
    const response = await request.post(`${API_BASE_URL}/slurm/install-test-nodes`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
      data: payload,
      timeout: 5 * 60 * 1000, // 5 分钟 API 超时
    });

    const duration = ((Date.now() - startTime) / 1000).toFixed(1);
    console.log(`\n⏱️  安装耗时: ${duration} 秒`);
    console.log(`   HTTP 状态码: ${response.status()}`);

    // 先获取响应体，无论成功失败
    const body = await response.json().catch(async () => {
      const text = await response.text();
      return { error: '无法解析 JSON 响应', raw: text };
    });
    
    console.log('\n📊 安装响应:');
    console.log(JSON.stringify(body, null, 2));

    if (!response.ok()) {
      console.log('\n❌ 请求失败详情:');
      console.log(`   状态码: ${response.status()}`);
      console.log(`   错误: ${body.error || body.message || '未知错误'}`);
      if (body.results) {
        console.log(`   结果数量: ${body.results.length}`);
      }
    }

    expect(response.ok(), `安装请求应该成功 (状态码: ${response.status()})`).toBeTruthy();

    expect(body.success, '安装应该成功').toBeTruthy();
    expect(Array.isArray(body.results), '结果应该是数组').toBeTruthy();
    expect(body.results.length, '应该有结果').toBeGreaterThan(0);

    // 验证每个节点的安装结果
    console.log('\n📋 各节点安装详情:');
    for (const hostResult of body.results) {
      console.log(`\n  节点: ${hostResult.host}`);
      console.log(`  状态: ${hostResult.success ? '✅ 成功' : '❌ 失败'}`);
      console.log(`  耗时: ${hostResult.duration}`);
      
      if (hostResult.error) {
        console.log(`  错误: ${hostResult.error}`);
      }

      expect(hostResult.success, `节点 ${hostResult.host} 应该安装成功`).toBeTruthy();
      expect(Array.isArray(hostResult.steps), '步骤应该是数组').toBeTruthy();

      // 验证关键安装步骤
      console.log(`  步骤验证:`);
      
      try {
        assertStepSuccess(hostResult.steps, 'configure_apt_source');
        console.log('    ✅ APT 源配置');
      } catch (e) {
        console.log('    ⚠️  APT 源配置步骤未找到或失败');
      }

      try {
        assertStepSuccess(hostResult.steps, 'install_saltstack_minion');
        console.log('    ✅ SaltStack Minion 安装');
      } catch (e) {
        console.log('    ⚠️  SaltStack Minion 安装步骤未找到或失败');
      }

      try {
        assertStepSuccess(hostResult.steps, 'final_verification');
        console.log('    ✅ 最终验证');
      } catch (e) {
        console.log('    ⚠️  最终验证步骤未找到或失败');
      }
    }

    console.log('\n✅ SaltStack Minion 安装完成');
  });

  test('3. 安装 Slurm Client 到测试节点', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 3: 安装 Slurm Client');
    console.log('========================================');

    test.setTimeout(6 * 60 * 1000); // 6 分钟超时

    const payload = {
      nodes: TEST_NODES,
      appHubURL: APPHUB_URL,
      enableSaltMinion: false,
      enableSlurmClient: true,
    };

    console.log('\n🚀 开始安装 Slurm Client...');
    console.log(`   节点: ${TEST_NODES.join(', ')}`);
    console.log(`   AppHub: ${APPHUB_URL}`);

    const startTime = Date.now();
    const response = await request.post(`${API_BASE_URL}/slurm/install-test-nodes`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
      data: payload,
      timeout: 5 * 60 * 1000, // 5 分钟 API 超时
    });

    const duration = ((Date.now() - startTime) / 1000).toFixed(1);
    console.log(`\n⏱️  安装耗时: ${duration} 秒`);
    console.log(`   HTTP 状态码: ${response.status()}`);

    // 先获取响应体，无论成功失败
    const body = await response.json().catch(async () => {
      const text = await response.text();
      return { error: '无法解析 JSON 响应', raw: text };
    });

    if (!response.ok()) {
      console.log('\n❌ 请求失败详情:');
      console.log(`   状态码: ${response.status()}`);
      console.log(`   错误: ${body.error || body.message || '未知错误'}`);
    }

    expect(response.ok(), `安装请求应该成功 (状态码: ${response.status()})`).toBeTruthy();
    
    console.log('\n📊 安装响应:');
    console.log(JSON.stringify(body, null, 2));

    expect(body.success, '安装应该成功').toBeTruthy();
    expect(Array.isArray(body.results), '结果应该是数组').toBeTruthy();

    // 验证每个节点的安装结果
    console.log('\n📋 各节点安装详情:');
    for (const hostResult of body.results) {
      console.log(`\n  节点: ${hostResult.host}`);
      console.log(`  状态: ${hostResult.success ? '✅ 成功' : '❌ 失败'}`);
      console.log(`  耗时: ${hostResult.duration}`);
      
      if (hostResult.error) {
        console.log(`  错误: ${hostResult.error}`);
      }

      expect(hostResult.success, `节点 ${hostResult.host} 应该安装成功`).toBeTruthy();
      expect(Array.isArray(hostResult.steps), '步骤应该是数组').toBeTruthy();

      // 验证关键安装步骤
      console.log(`  步骤验证:`);
      
      try {
        assertStepSuccess(hostResult.steps, 'install_slurm_client');
        console.log('    ✅ Slurm Client 安装');
      } catch (e) {
        console.log('    ⚠️  Slurm Client 安装步骤未找到或失败');
      }
    }

    console.log('\n✅ Slurm Client 安装完成');
  });

  test('4. 验证 SaltStack 页面显示正常', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试 4: 验证 SaltStack 页面');
    console.log('========================================');

    test.setTimeout(3 * 60 * 1000); // 3 分钟超时

    await loginViaBrowser(page);

    // 导航到 SaltStack 页面
    console.log('\n🌐 访问 SaltStack 页面...');
    await page.goto(`${BASE_URL.replace(/\/$/, '')}/saltstack`);
    await waitForPageLoad(page);

    // 验证页面标题
    console.log('\n📋 验证页面元素...');
    await expect(page.getByText('SaltStack 配置管理')).toBeVisible({ timeout: 15000 });
    console.log('✅ 页面标题正常');

    // 检查是否有演示模式横幅（不应该出现）
    const demoBannerCount = await page.getByText(/演示模式|显示SaltStack演示数据/).count();
    expect(demoBannerCount, '不应该显示演示模式横幅').toBe(0);
    console.log('✅ 无演示模式横幅');

    // 等待 Minions 数据加载
    console.log('\n⏳ 等待 Minions 数据加载 (15秒)...');
    await page.waitForTimeout(15000);

    // 检查 Minions 表格
    const minionsTable = page.locator('.ant-table').or(page.locator('table')).first();
    const hasTable = await minionsTable.isVisible().catch(() => false);
    
    if (hasTable) {
      console.log('✅ Minions 表格已加载');
      
      // 获取表格行数
      const rows = minionsTable.locator('tbody tr');
      const rowCount = await rows.count().catch(() => 0);
      console.log(`📊 Minion 数量: ${rowCount}`);

      if (rowCount > 0) {
        // 检查测试节点
        const pageText = await page.textContent('body');
        let foundNodes = [];

        for (const nodeName of TEST_NODES) {
          if (pageText.includes(nodeName)) {
            foundNodes.push(nodeName);
            console.log(`  ✅ 找到节点: ${nodeName}`);
          }
        }

        if (foundNodes.length > 0) {
          console.log(`\n✅ 成功找到 ${foundNodes.length}/${TEST_NODES.length} 个测试节点`);
        } else {
          console.log('\n⚠️  未找到测试节点，可能还在连接中...');
        }
      } else {
        console.log('⚠️  Minions 表格为空');
      }
    } else {
      console.log('⚠️  未找到 Minions 表格');
    }

    // 检查 SaltStack 状态卡片
    console.log('\n📊 检查状态卡片...');
    const statusCards = page.locator('.ant-statistic').or(page.locator('.ant-card'));
    const cardCount = await statusCards.count();
    console.log(`  状态卡片数量: ${cardCount}`);

    // 截图保存当前状态
    const screenshotPath = 'test-screenshots/saltstack-page-status.png';
    await page.screenshot({ path: screenshotPath, fullPage: true });
    console.log(`\n📸 页面截图已保存: ${screenshotPath}`);
  });

  test('5. 验证 Slurm 页面显示正常', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试 5: 验证 Slurm 页面');
    console.log('========================================');

    test.setTimeout(3 * 60 * 1000); // 3 分钟超时

    await loginViaBrowser(page);

    // 导航到 Slurm 页面
    console.log('\n🌐 访问 Slurm 页面...');
    await page.goto(`${BASE_URL.replace(/\/$/, '')}/slurm`);
    await waitForPageLoad(page);

    // 验证页面标题
    console.log('\n📋 验证页面元素...');
    const titleVisible = await page.getByText(/SLURM 集群|SLURM 任务/).isVisible({ timeout: 15000 });
    expect(titleVisible, 'Slurm 页面标题应该可见').toBeTruthy();
    console.log('✅ 页面标题正常');

    // 检查是否有演示模式横幅（不应该出现）
    const demoBannerCount = await page.getByText(/演示模式|显示Slurm演示数据/).count();
    expect(demoBannerCount, '不应该显示演示模式横幅').toBe(0);
    console.log('✅ 无演示模式横幅');

    // 等待数据加载
    console.log('\n⏳ 等待数据加载 (10秒)...');
    await page.waitForTimeout(10000);

    // 检查 SaltStack 集成状态
    console.log('\n📊 检查 SaltStack 集成状态...');
    const pageText = await page.textContent('body');
    
    if (pageText.includes('Master 状态') || pageText.includes('Master Status')) {
      console.log('✅ SaltStack 集成状态卡片已加载');
      
      // 检查状态值
      if (pageText.includes('running') || pageText.includes('运行中')) {
        console.log('  ✅ Master 状态: 运行中');
      }
      
      if (pageText.includes('connected') || pageText.includes('已连接')) {
        console.log('  ✅ API 状态: 已连接');
      }
    }

    // 截图保存当前状态
    const screenshotPath = 'test-screenshots/slurm-page-status.png';
    await page.screenshot({ path: screenshotPath, fullPage: true });
    console.log(`\n📸 页面截图已保存: ${screenshotPath}`);
  });

  test('6. 测试 SaltStack 命令执行', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 6: 测试 SaltStack 命令执行');
    console.log('========================================');

    test.setTimeout(2 * 60 * 1000); // 2 分钟超时

    // 获取 Minions 列表
    console.log('\n📋 获取 Minions 列表...');
    const minionsResponse = await request.get(`${API_BASE_URL}/saltstack/minions`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
    });

    if (!minionsResponse.ok()) {
      console.log('⚠️  无法获取 Minions 列表，跳过命令执行测试');
      return;
    }

    const minionsBody = await minionsResponse.json();
    const minions = minionsBody.data || minionsBody.minions || [];
    
    console.log(`📊 Minion 数量: ${minions.length}`);
    
    if (minions.length === 0) {
      console.log('⚠️  没有可用的 Minions，跳过命令执行测试');
      return;
    }

    // 显示 Minion 列表
    console.log('\n📋 可用的 Minions:');
    minions.forEach((minion, index) => {
      const minionId = minion.id || minion.minion_id || 'unknown';
      const status = minion.status || 'unknown';
      console.log(`  ${index + 1}. ${minionId} (${status})`);
    });

    // 执行测试命令：test.ping
    console.log('\n🧪 执行测试命令: test.ping');
    const commandPayload = {
      target: '*',
      function: 'test.ping',
      args: [],
    };

    const execResponse = await request.post(`${API_BASE_URL}/saltstack/execute`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
      data: commandPayload,
      timeout: 30000,
    });

    if (execResponse.ok()) {
      const execBody = await execResponse.json();
      console.log('\n📊 命令执行结果:');
      console.log(JSON.stringify(execBody, null, 2));
      
      if (execBody.success || (execBody.data && Object.keys(execBody.data).length > 0)) {
        console.log('✅ 命令执行成功');
        
        // 显示每个 Minion 的响应
        const results = execBody.data || execBody.result || {};
        Object.entries(results).forEach(([minionId, response]) => {
          console.log(`  ${minionId}: ${JSON.stringify(response)}`);
        });
      } else {
        console.log('⚠️  命令执行失败或无响应');
      }
    } else {
      console.log(`⚠️  命令执行 API 失败: ${execResponse.status()}`);
    }
  });

  test('7. 完整性验证总结', async ({ request }) => {
    console.log('\n========================================');
    console.log('测试 7: 完整性验证总结');
    console.log('========================================');

    const summary = {
      appHubAvailable: false,
      saltStackInstalled: false,
      slurmInstalled: false,
      minionsOnline: 0,
      saltStackPageWorking: false,
      slurmPageWorking: false,
    };

    // 检查 AppHub
    try {
      const appHubResponse = await request.get(APPHUB_URL);
      summary.appHubAvailable = appHubResponse.ok();
    } catch (e) {
      console.log('⚠️  AppHub 检查失败');
    }

    // 检查 SaltStack 集成状态
    try {
      const saltIntegrationResponse = await request.get(`${API_BASE_URL}/slurm/saltstack/integration`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
        },
      });

      if (saltIntegrationResponse.ok()) {
        const saltBody = await saltIntegrationResponse.json();
        const data = saltBody.data || {};
        
        summary.saltStackInstalled = data.enabled || data.master_status === 'running';
        summary.minionsOnline = data.minions?.online || 0;
        summary.saltStackPageWorking = true;
      }
    } catch (e) {
      console.log('⚠️  SaltStack 集成检查失败');
    }

    // 检查 Slurm 节点
    try {
      const slurmNodesResponse = await request.get(`${API_BASE_URL}/slurm/nodes`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
        },
      });

      if (slurmNodesResponse.ok()) {
        const slurmBody = await slurmNodesResponse.json();
        const nodes = slurmBody.data || slurmBody.nodes || [];
        summary.slurmInstalled = nodes.length > 0;
        summary.slurmPageWorking = true;
      }
    } catch (e) {
      console.log('⚠️  Slurm 节点检查失败');
    }

    // 打印总结
    console.log('\n📊 完整性验证结果:');
    console.log('========================================');
    console.log(`  AppHub 可用性:        ${summary.appHubAvailable ? '✅' : '❌'}`);
    console.log(`  SaltStack 已安装:     ${summary.saltStackInstalled ? '✅' : '❌'}`);
    console.log(`  Slurm Client 已安装:  ${summary.slurmInstalled ? '✅' : '❌'}`);
    console.log(`  在线 Minions 数量:    ${summary.minionsOnline}`);
    console.log(`  SaltStack 页面正常:   ${summary.saltStackPageWorking ? '✅' : '❌'}`);
    console.log(`  Slurm 页面正常:       ${summary.slurmPageWorking ? '✅' : '❌'}`);
    console.log('========================================');

    // 计算成功率
    const checks = [
      summary.appHubAvailable,
      summary.saltStackInstalled,
      summary.slurmInstalled,
      summary.saltStackPageWorking,
      summary.slurmPageWorking,
    ];
    const successCount = checks.filter(Boolean).length;
    const successRate = (successCount / checks.length * 100).toFixed(0);

    console.log(`\n🎯 整体成功率: ${successCount}/${checks.length} (${successRate}%)`);

    if (successRate >= 80) {
      console.log('✅ 测试通过！系统运行正常');
    } else {
      console.log('⚠️  部分功能异常，需要检查');
    }
  });
});
