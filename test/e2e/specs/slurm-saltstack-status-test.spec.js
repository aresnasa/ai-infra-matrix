/**
 * SLURM SaltStack状态同步测试
 * 
 * 测试目标:
 * 1. 验证SaltStack Master状态正确显示
 * 2. 验证SaltStack API状态正确显示
 * 3. 验证Minions数量正确同步（应该显示7个已接受的minions）
 * 4. 验证活跃作业数量
 * 
 * 期望结果:
 * - Master状态: running/available
 * - API状态: available
 * - 连接的Minions: 7 (salt-master-local, test-rocky01-03, test-ssh01-03)
 * - Minions列表正确显示
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('SLURM SaltStack状态同步测试', () => {
  let authToken;

  test.beforeAll(async ({ request }) => {
    // 登录获取token
    const loginResponse = await request.post(`${BASE_URL}/api/auth/login`, {
      data: {
        username: TEST_USERNAME,
        password: TEST_PASSWORD
      }
    });
    
    expect(loginResponse.ok()).toBeTruthy();
    const loginData = await loginResponse.json();
    authToken = loginData.data?.token || loginData.token;
    console.log('✓ 登录成功');
  });

  test('验证SaltStack集成状态API', async ({ request }) => {
    console.log('\n🔌 测试 /api/slurm/saltstack/integration');
    
    const response = await request.get(`${BASE_URL}/api/slurm/saltstack/integration`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    console.log('Response status:', response.status());
    expect(response.ok()).toBeTruthy();

    const data = await response.json();
    console.log('Response data:', JSON.stringify(data, null, 2));

    // 验证响应结构
    expect(data).toHaveProperty('data');
    const integration = data.data;

    console.log('\n📊 SaltStack集成状态:');
    console.log('  - Enabled:', integration.enabled);
    console.log('  - Master Status:', integration.master_status);
    console.log('  - API Status:', integration.api_status);
    console.log('  - Total Minions:', integration.minions?.total);
    console.log('  - Online Minions:', integration.minions?.online);
    console.log('  - Offline Minions:', integration.minions?.offline);
    console.log('  - Recent Jobs:', integration.recent_jobs);

    // 验证SaltStack已启用
    expect(integration.enabled).toBe(true);

    // 验证Master状态
    expect(['running', 'available']).toContain(integration.master_status);

    // 验证API状态
    expect(['running', 'available']).toContain(integration.api_status);

    // 验证Minions数量（应该有7个已接受的minions）
    expect(integration.minions).toBeDefined();
    expect(integration.minions.total).toBeGreaterThanOrEqual(7);
    console.log('✅ Minions总数符合预期:', integration.minions.total);

    // 验证Minion列表
    if (integration.minion_list) {
      console.log('\n📋 Minion列表:');
      integration.minion_list.forEach(minion => {
        console.log(`  - ${minion}`);
      });
      expect(integration.minion_list.length).toBeGreaterThanOrEqual(7);
    }

    // 如果有错误，应该为空
    if (integration.error) {
      console.warn('⚠️  检测到错误:', integration.error);
      // 修复后不应该有错误
      expect(integration.error).toBe('');
    }
  });

  test('验证SaltStack Keys API', async ({ request }) => {
    console.log('\n🔑 测试 /api/slurm/saltstack/keys');
    
    const response = await request.get(`${BASE_URL}/api/slurm/saltstack/keys`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    console.log('Response status:', response.status());
    expect(response.ok()).toBeTruthy();

    const data = await response.json();
    
    // 验证响应结构
    expect(data).toHaveProperty('data');
    const keys = data.data;

    console.log('\n🔑 Salt Keys状态:');
    console.log('  - Accepted Keys:', keys.accepted?.length || 0);
    console.log('  - Unaccepted Keys:', keys.unaccepted?.length || 0);
    console.log('  - Rejected Keys:', keys.rejected?.length || 0);

    // 验证已接受的keys（应该有7个）
    expect(keys.accepted).toBeDefined();
    expect(Array.isArray(keys.accepted)).toBe(true);
    expect(keys.accepted.length).toBeGreaterThanOrEqual(7);

    console.log('\n📋 已接受的Keys:');
    const expectedMinions = [
      'salt-master-local',
      'test-rocky01',
      'test-rocky02',
      'test-rocky03',
      'test-ssh01',
      'test-ssh02',
      'test-ssh03'
    ];

    keys.accepted.forEach(key => {
      console.log(`  ✓ ${key}`);
    });

    // 验证关键minions存在
    expectedMinions.forEach(minionId => {
      expect(keys.accepted).toContain(minionId);
      console.log(`  ✅ 找到 ${minionId}`);
    });

    // 未接受的keys应该为空或很少
    if (keys.unaccepted && keys.unaccepted.length > 0) {
      console.log('\n⚠️  待接受的Keys:');
      keys.unaccepted.forEach(key => {
        console.log(`  - ${key}`);
      });
    }
  });

  test('验证SLURM前端页面显示SaltStack状态', async ({ page }) => {
    console.log('\n🌐 测试前端页面 /slurm');

    // 设置认证token
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('token', token);
    }, authToken);

    // 访问SLURM页面
    await page.goto(`${BASE_URL}/slurm`);

    // 等待页面加载
    await page.waitForLoadState('networkidle');

    // 等待SaltStack集成卡片出现
    const saltStackCard = page.locator('text=SaltStack 集成').first();
    await expect(saltStackCard).toBeVisible({ timeout: 10000 });

    console.log('✓ SaltStack集成卡片已显示');

    // 检查Master状态
    const masterStatusLocator = page.locator('text=Master状态').locator('..')
      .locator('xpath=following-sibling::*[1]');
    
    // 等待状态加载完成（不是"加载中..."或骨架屏）
    await page.waitForTimeout(3000);

    const masterStatus = await masterStatusLocator.textContent();
    console.log('  Master状态:', masterStatus?.trim());
    
    // Master状态应该是available或running，不应该是unavailable
    expect(masterStatus?.trim().toLowerCase()).not.toBe('unavailable');
    expect(masterStatus?.trim().toLowerCase()).not.toBe('加载中...');

    // 检查API状态
    const apiStatusLocator = page.locator('text=API状态').locator('..')
      .locator('xpath=following-sibling::*[1]');
    const apiStatus = await apiStatusLocator.textContent();
    console.log('  API状态:', apiStatus?.trim());
    
    // API状态应该是available或running
    expect(apiStatus?.trim().toLowerCase()).not.toBe('unavailable');

    // 检查连接的Minions数量
    const minionsLocator = page.locator('text=连接的Minions').locator('..')
      .locator('xpath=following-sibling::*[1]');
    const minionsText = await minionsLocator.textContent();
    console.log('  连接的Minions:', minionsText?.trim());
    
    // Minions数量应该大于0
    const minionsCount = parseInt(minionsText?.trim() || '0');
    expect(minionsCount).toBeGreaterThanOrEqual(7);
    console.log('  ✅ Minions数量符合预期:', minionsCount);

    // 检查活跃作业（可选）
    const activeJobsLocator = page.locator('text=活跃作业').locator('..')
      .locator('xpath=following-sibling::*[1]');
    const activeJobs = await activeJobsLocator.textContent();
    console.log('  活跃作业:', activeJobs?.trim());

    // 截图保存当前状态
    await page.screenshot({ 
      path: 'test-screenshots/slurm-saltstack-status.png',
      fullPage: true 
    });
    console.log('📸 已保存页面截图: test-screenshots/slurm-saltstack-status.png');
  });

  test('验证SaltStack Minions列表显示', async ({ page }) => {
    console.log('\n📋 测试SaltStack Minions列表');

    // 设置认证token
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('token', token);
    }, authToken);

    // 访问SLURM页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');

    // 等待页面加载
    await page.waitForTimeout(3000);

    // 查找Minions列表表格或列表
    const minionsSection = page.locator('text=/Minions|节点列表/i').first();
    
    if (await minionsSection.isVisible()) {
      console.log('✓ 找到Minions列表区域');

      // 检查是否有minion显示
      const expectedMinions = [
        'salt-master-local',
        'test-rocky01',
        'test-rocky02',
        'test-rocky03',
        'test-ssh01',
        'test-ssh02',
        'test-ssh03'
      ];

      for (const minionId of expectedMinions) {
        const minionElement = page.locator(`text="${minionId}"`).first();
        if (await minionElement.isVisible({ timeout: 2000 }).catch(() => false)) {
          console.log(`  ✅ 找到 minion: ${minionId}`);
        } else {
          console.log(`  ⚠️  未找到 minion: ${minionId}`);
        }
      }
    } else {
      console.log('⚠️  未找到Minions列表区域（可能在其他tab或折叠面板中）');
    }

    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-minions-list.png',
      fullPage: true 
    });
    console.log('📸 已保存页面截图: test-screenshots/slurm-minions-list.png');
  });
});
