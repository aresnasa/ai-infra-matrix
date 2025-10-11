/**
 * SaltStack 4 Minions 验证测试
 * 
 * 目的: 验证重启 test-ssh 容器后,SaltStack 正确显示 4 个 minions
 * 
 * 预期结果:
 * - 在线 Minions: 4 (salt-master-local + test-ssh01/02/03)
 * - 离线 Minions: 0
 * - 页面加载时间 < 10秒
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('SaltStack 应该显示 4 个在线 minions', async ({ page }) => {
  console.log('\n✅ 测试: 验证 4 个在线 minions');
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000); // 等待登录完成
  
  const startTime = Date.now();
  
  // 访问 SaltStack 页面
  await page.goto(`${BASE_URL}/saltstack`);
  await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
  
  const loadTime = Date.now() - startTime;
  console.log(`⏱️  页面加载时间: ${loadTime}ms`);
  
  // 验证加载时间 < 10秒
  expect(loadTime).toBeLessThan(10000);
  
  // 等待数据加载
  await page.waitForTimeout(2000);
  
  // 验证在线 Minions 数量
  const onlineMinionsText = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
  console.log(`🟢 在线 Minions: ${onlineMinionsText}`);
  expect(onlineMinionsText).toBe('4');
  
  // 验证离线 Minions 数量
  const offlineMinionsText = await page.locator('text=离线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
  console.log(`⚪ 离线 Minions: ${offlineMinionsText}`);
  expect(offlineMinionsText).toBe('0');
  
  // 验证 Master 状态
  const masterStatusText = await page.locator('text=Master状态').locator('..').locator('.ant-statistic-content').textContent();
  console.log(`⚙️  Master 状态: ${masterStatusText}`);
  expect(masterStatusText).toContain('running');
  
  // 验证 API 状态
  const apiStatusText = await page.locator('text=API状态').locator('..').locator('.ant-statistic-content').textContent();
  console.log(`🔌 API 状态: ${apiStatusText}`);
  expect(apiStatusText).toContain('running');
  
  console.log('✅ 测试通过: 4 个 minions 在线\n');
});

test('执行命令应该在所有 4 个 minions 上成功', async ({ page }) => {
  console.log('\n✅ 测试: 在所有 minions 上执行命令');
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000); // 等待登录完成
  
  await page.goto(`${BASE_URL}/saltstack`);
  await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
  await page.waitForTimeout(2000);
  
  // 点击执行命令按钮
  await page.click('text=执行命令');
  await page.waitForTimeout(1000);
  
  // 填写命令
  await page.fill('textarea[placeholder*="粘贴脚本"]', 'hostname');
  
  // 执行
  await page.click('button:has-text("执 行")');
  
  // 等待执行完成
  await page.waitForSelector('text=命令执行完成', { timeout: 30000 });
  
  // 验证所有 4 个 minions 都有响应
  // 日志区域在 Modal 内的 Card 中,使用更精确的定位器
  const logContainer = page.locator('.ant-modal-body').locator('.ant-card-body').last();
  const progressText = await logContainer.textContent();
  console.log('📝 执行日志:\n', progressText);
  
  // 检查是否包含所有 minions
  expect(progressText).toContain('salt-master-local');
  expect(progressText).toContain('test-ssh01');
  expect(progressText).toContain('test-ssh02');
  expect(progressText).toContain('test-ssh03');
  
  console.log('✅ 测试通过: 所有 4 个 minions 执行成功\n');
});

test('刷新数据应该保持 4 个 minions', async ({ page }) => {
  console.log('\n✅ 测试: 刷新数据验证');
  
  // 登录
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[placeholder="用户名"]', 'admin');
  await page.fill('input[placeholder="密码"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000); // 等待登录完成
  
  await page.goto(`${BASE_URL}/saltstack`);
  await page.waitForSelector('text=SaltStack 配置管理', { timeout: 10000 });
  await page.waitForTimeout(2000);
  
  // 第一次检查
  let onlineMinionsText = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
  console.log(`🔍 第一次检查 - 在线 Minions: ${onlineMinionsText}`);
  expect(onlineMinionsText).toBe('4');
  
  // 点击刷新
  await page.click('text=刷新数据');
  await page.waitForTimeout(3000);
  
  // 第二次检查
  onlineMinionsText = await page.locator('text=在线Minions').locator('..').locator('.ant-statistic-content-value').textContent();
  console.log(`🔄 刷新后 - 在线 Minions: ${onlineMinionsText}`);
  expect(onlineMinionsText).toBe('4');
  
  console.log('✅ 测试通过: 刷新后保持 4 个 minions\n');
});
