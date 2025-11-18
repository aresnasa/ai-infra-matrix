// SaltStack Minions 数据获取修复验证测试
// 修复: 删除无效 SSH minion keys + 调整超时(90s->10s)

const { test, expect } = require('@playwright/test');

// 测试配置
const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';
const TEST_USERNAME = 'admin';
const TEST_PASSWORD = 'admin123';

test.describe('SaltStack Minions 数据获取修复验证', () => {
  
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[placeholder="用户名"]', TEST_USERNAME);
    await page.fill('input[placeholder="密码"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    
    // 等待登录完成
    await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
  });

  test('验证 SaltStack 页面能正确加载 minions 数据', async ({ page }) => {
    console.log('✅ 测试1: 验证 SaltStack 页面数据加载');
    
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    
    // 等待页面加载(最多10秒,之前会超时30秒)
    await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
    
    // 验证统计卡片显示正确
    const stats = await page.locator('.ant-statistic').allTextContents();
    console.log('📊 统计数据:', stats.join(', '));
    
    // 验证至少有在线 minions
    const onlineMinions = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
    console.log(`🟢 在线 Minions: ${onlineMinions}`);
    expect(parseInt(onlineMinions)).toBeGreaterThan(0);
    
    // 验证离线 minions 为 0 (因为已删除 SSH minion keys)
    const offlineMinions = await page.locator('text=离线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
    console.log(`⚪ 离线 Minions: ${offlineMinions}`);
    expect(parseInt(offlineMinions)).toBe(0);
    
    // 验证 Master 状态
    const masterStatus = await page.locator('text=Master状态').locator('..').locator('.ant-statistic-content').textContent();
    console.log(`⚙️  Master 状态: ${masterStatus}`);
    expect(masterStatus).toContain('running');
    
    // 验证 API 状态
    const apiStatus = await page.locator('text=API状态').locator('..').locator('.ant-statistic-content').textContent();
    console.log(`🔌 API 状态: ${apiStatus}`);
    expect(apiStatus).toContain('running');
    
    console.log('✅ 测试1通过: SaltStack 页面数据加载正常');
  });

  test('验证 Minions管理 标签能显示详细信息', async ({ page }) => {
    console.log('✅ 测试2: 验证 Minions 详细信息显示');
    
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
    
    // 点击 Minions管理 标签
    await page.click('text=Minions管理');
    await page.waitForTimeout(1000);
    
    // 验证至少有一个 minion 信息卡片
    const minionCards = await page.locator('.ant-card').count();
    console.log(`📦 Minion 卡片数量: ${minionCards}`);
    expect(minionCards).toBeGreaterThan(0);
    
    // 验证 minion 详细信息
    const minionInfo = await page.locator('.ant-card').first();
    const hasOS = await minionInfo.locator('text=操作系统').count();
    const hasArch = await minionInfo.locator('text=架构').count();
    const hasVersion = await minionInfo.locator('text=Salt版本').count();
    
    console.log(`ℹ️  包含操作系统信息: ${hasOS > 0}`);
    console.log(`ℹ️  包含架构信息: ${hasArch > 0}`);
    console.log(`ℹ️  包含版本信息: ${hasVersion > 0}`);
    
    expect(hasOS).toBeGreaterThan(0);
    expect(hasArch).toBeGreaterThan(0);
    expect(hasVersion).toBeGreaterThan(0);
    
    console.log('✅ 测试2通过: Minions 详细信息显示正常');
  });

  test('验证 SaltStack 页面不会出现超时错误', async ({ page }) => {
    console.log('✅ 测试3: 验证无超时错误');
    
    // 监听console错误
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
    
    // 等待数据加载
    await page.waitForTimeout(3000);
    
    // 检查是否有超时错误
    const hasTimeoutError = errors.some(err => 
      err.includes('timeout') || 
      err.includes('exceeded') ||
      err.includes('Network error')
    );
    
    console.log(`🔍 发现的错误数量: ${errors.length}`);
    if (errors.length > 0) {
      console.log(`⚠️  错误列表: ${errors.slice(0, 3).join(', ')}...`);
    }
    
    expect(hasTimeoutError).toBe(false);
    console.log('✅ 测试3通过: 无超时错误');
  });

  test('验证刷新数据功能工作正常', async ({ page }) => {
    console.log('✅ 测试4: 验证刷新数据功能');
    
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
    
    // 获取初始的 minions 数量
    const initialCount = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
    console.log(`📊 初始在线 Minions: ${initialCount}`);
    
    // 点击刷新按钮
    await page.click('button:has-text("刷新数据")');
    
    // 等待加载图标出现(可选)
    await page.waitForTimeout(1000);
    
    // 等待数据更新完成
    await page.waitForTimeout(2000);
    
    // 验证数据仍然正确
    const refreshedCount = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
    console.log(`📊 刷新后在线 Minions: ${refreshedCount}`);
    
    expect(refreshedCount).toBe(initialCount);
    console.log('✅ 测试4通过: 刷新数据功能正常');
  });

  test('验证页面响应时间符合预期', async ({ page }) => {
    console.log('✅ 测试5: 验证页面响应时间');
    
    const startTime = Date.now();
    
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
    
    // 等待在线 minions 数据显示
    await page.waitForSelector('text=在线Minions');
    const onlineMinions = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
    
    const loadTime = Date.now() - startTime;
    console.log(`⏱️  页面加载时间: ${loadTime}ms`);
    console.log(`📊 在线 Minions: ${onlineMinions}`);
    
    // 验证加载时间在合理范围内 (应该远小于30秒超时)
    expect(loadTime).toBeLessThan(10000); // 10秒内完成加载
    expect(parseInt(onlineMinions)).toBeGreaterThan(0);
    
    console.log('✅ 测试5通过: 页面响应时间符合预期');
  });
});

test.describe('SLURM SaltStack 集成验证', () => {
  
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[placeholder="用户名"]', TEST_USERNAME);
    await page.fill('input[placeholder="密码"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    
    // 等待登录完成
    await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
  });

  test('验证 SLURM 页面能访问 SaltStack 数据', async ({ page }) => {
    console.log('✅ 测试6: 验证 SLURM SaltStack 集成');
    
    // 导航到 SLURM 页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForSelector('text=SLURM 弹性扩缩容管理', { timeout: 10000 });
    
    // 等待数据加载
    await page.waitForTimeout(3000);
    
    // 验证 SaltStack Minions 统计卡片存在
    const hasSaltStackCard = await page.locator('text=SaltStack Minions').count();
    console.log(`🔍 SaltStack Minions 卡片存在: ${hasSaltStackCard > 0}`);
    expect(hasSaltStackCard).toBeGreaterThan(0);
    
    // 注意: SLURM 页面的 SaltStack 集成可能显示 0,因为它需要专门配置
    // 主要验证是页面不会因为 SaltStack API 超时而挂起
    const saltStackMinions = await page.locator('text=SaltStack Minions').locator('..').locator('.ant-statistic-content-value').textContent();
    console.log(`📊 SLURM 页面 SaltStack Minions: ${saltStackMinions}`);
    
    // 主要验证: 数字是有效的,不是空或错误状态
    expect(saltStackMinions).toMatch(/^\d+$/);
    
    console.log('✅ 测试6通过: SLURM SaltStack 集成工作正常(无超时)');
  });
});
