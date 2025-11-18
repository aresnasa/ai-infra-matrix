// SaltStack Minions 数据获取修复验证 - 简化版本
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';

test('SaltStack 页面加载并显示 minions 数据', async ({ page }) => {
  console.log('\n✅ 测试: SaltStack 页面数据加载');
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
  
  // 导航到 SaltStack 页面
  await page.goto(`${BASE_URL}/saltstack`);
  await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
  
  // 验证在线 minions
  const onlineMinions = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
  console.log(`🟢 在线 Minions: ${onlineMinions}`);
  expect(parseInt(onlineMinions)).toBeGreaterThan(0);
  
  // 验证离线 minions 为 0
  const offlineMinions = await page.locator('text=离线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
  console.log(`⚪ 离线 Minions: ${offlineMinions}`);
  expect(parseInt(offlineMinions)).toBe(0);
  
  // 验证 Master 状态
  const masterStatus = await page.locator('text=Master状态').locator('..').locator('.ant-statistic-content').textContent();
  console.log(`⚙️  Master 状态: ${masterStatus}`);
  expect(masterStatus).toContain('running');
  
  console.log('✅ 测试通过: SaltStack 页面数据加载正常\n');
});

test('Minions管理标签显示详细信息', async ({ page }) => {
  console.log('\n✅ 测试: Minions 详细信息显示');
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
  
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
  
  console.log('✅ 测试通过: Minions 详细信息显示正常\n');
});

test('SaltStack 页面无超时错误', async ({ page }) => {
  console.log('\n✅ 测试: 验证无超时错误');
  
  // 监听console错误
  const errors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
  
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
  expect(hasTimeoutError).toBe(false);
  console.log('✅ 测试通过: 无超时错误\n');
});

test('页面响应时间符合预期', async ({ page }) => {
  console.log('\n✅ 测试: 验证页面响应时间');
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
  
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
  
  console.log('✅ 测试通过: 页面响应时间符合预期\n');
});
