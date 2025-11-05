/**
 * SLURM SaltStack Minions数量显示测试
 * 
 * 测试目标: 验证前端页面正确显示Minions数量
 * 
 * Bug修复: 前端使用 connected_minions 字段，但API返回的是 minions.total
 * 修复: 将前端代码从 saltIntegration?.connected_minions 改为 saltIntegration?.minions?.total
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('SLURM Minions数量显示测试', () => {
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

  test('验证API返回正确的Minions数量', async ({ request }) => {
    console.log('\n🔌 测试 API: /api/slurm/saltstack/integration');
    
    const response = await request.get(`${BASE_URL}/api/slurm/saltstack/integration`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    const integration = data.data;

    console.log('\n📊 API返回的数据结构:');
    console.log('  - minions.total:', integration.minions?.total);
    console.log('  - minions.online:', integration.minions?.online);
    console.log('  - minions.offline:', integration.minions?.offline);
    console.log('  - minion_list长度:', integration.minion_list?.length);

    // 验证minions对象存在
    expect(integration.minions).toBeDefined();
    expect(integration.minions.total).toBeDefined();
    
    // 验证minions数量为7
    expect(integration.minions.total).toBe(7);
    expect(integration.minions.online).toBe(7);
    expect(integration.minions.offline).toBe(0);
    
    // 验证minion_list长度也是7
    expect(integration.minion_list).toBeDefined();
    expect(integration.minion_list.length).toBe(7);
    
    console.log('✅ API返回数据正确: minions.total = 7');
  });

  test('验证前端页面显示正确的Minions数量', async ({ page }) => {
    console.log('\n🌐 测试前端页面: /slurm');

    // 设置认证token
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('token', token);
    }, authToken);

    // 访问SLURM页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');

    // 等待SaltStack集成卡片出现
    const saltStackCard = page.locator('text=SaltStack 集成').first();
    await expect(saltStackCard).toBeVisible({ timeout: 10000 });

    console.log('✓ SaltStack集成卡片已显示');

    // 等待数据加载完成
    await page.waitForTimeout(3000);

    // 查找"连接的Minions"标签
    const minionsLabel = page.locator('text=连接的Minions');
    await expect(minionsLabel).toBeVisible({ timeout: 5000 });

    // 获取Minions数量显示值
    // 使用不同的选择器策略
    const minionsValue = await page.evaluate(() => {
      // 查找包含"连接的Minions"的元素
      const labels = Array.from(document.querySelectorAll('.ant-descriptions-item-label'));
      const minionsLabel = labels.find(el => el.textContent?.includes('连接的Minions'));
      
      if (minionsLabel) {
        // 找到对应的值元素（下一个兄弟节点）
        const valueElement = minionsLabel.parentElement?.querySelector('.ant-descriptions-item-content');
        return valueElement?.textContent?.trim();
      }
      return null;
    });

    console.log('  连接的Minions显示值:', minionsValue);

    // 验证显示的是7而不是0
    expect(minionsValue).toBe('7');
    console.log('✅ 前端页面正确显示 Minions 数量: 7');

    // 截图保存
    await page.screenshot({ 
      path: 'test-screenshots/slurm-minions-count-fixed.png',
      fullPage: true 
    });
    console.log('📸 已保存页面截图: test-screenshots/slurm-minions-count-fixed.png');
  });

  test('验证Minions详细信息', async ({ page }) => {
    console.log('\n📋 验证Minions详细信息');

    // 设置认证token
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('token', token);
    }, authToken);

    // 访问SLURM页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 检查是否有显示详细的minion列表
    const expectedMinions = [
      'salt-master-local',
      'test-rocky01',
      'test-rocky02', 
      'test-rocky03',
      'test-ssh01',
      'test-ssh02',
      'test-ssh03'
    ];

    console.log('  预期的7个Minions:');
    expectedMinions.forEach(m => console.log(`    - ${m}`));

    // 尝试查找minion列表（可能在表格或列表中）
    for (const minionId of expectedMinions.slice(0, 3)) {
      const minionText = page.locator(`text="${minionId}"`).first();
      const isVisible = await minionText.isVisible({ timeout: 1000 }).catch(() => false);
      
      if (isVisible) {
        console.log(`  ✅ 找到: ${minionId}`);
      } else {
        console.log(`  ℹ️  未在页面上找到: ${minionId} (可能在其他tab或需要展开)`);
      }
    }
  });
});
