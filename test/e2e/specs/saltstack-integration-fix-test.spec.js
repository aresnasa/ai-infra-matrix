/**
 * SaltStack 集成修复验证测试
 * 
 * 目的：验证 /api/slurm/saltstack/integration API 修复
 * 
 * 修复内容：
 * 1. GetSaltStackIntegration 现在直接调用 SaltStack handler
 * 2. 避免使用有问题的 saltSvc.GetStatus（返回 demo 数据）
 * 3. 确保返回真实的 SaltStack 集群状态
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';
const API_BASE = BASE_URL.replace(':8080', ':8082');

test.describe('SaltStack Integration Fix Verification', () => {
  let adminToken;

  test.beforeAll(async ({ request }) => {
    console.log('\n🔧 获取管理员 token...');
    const loginRes = await request.post(`${API_BASE}/api/auth/login`, {
      data: { username: 'admin', password: 'admin123' }
    });
    const loginData = await loginRes.json();
    adminToken = loginData.token; // token 在顶层，不在 data 中
    expect(adminToken).toBeTruthy();
    console.log('✅ 管理员认证成功');
  });

  test('验证 /api/slurm/saltstack/integration 返回真实数据', async ({ request }) => {
    console.log('\n📊 测试 1: 验证 SaltStack 集成 API 返回真实数据');
    
    const response = await request.get(`${API_BASE}/api/slurm/saltstack/integration`, {
      headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    
    expect(response.status()).toBe(200);
    const result = await response.json();
    console.log('Integration API Response:', JSON.stringify(result, null, 2));
    
    const data = result.data;
    
    // 关键验证：不应该是演示模式
    console.log(`\n🎯 关键检查:`);
    console.log(`   demo: ${data.demo}`);
    console.log(`   master_status: ${data.master_status}`);
    console.log(`   api_status: ${data.api_status}`);
    console.log(`   minions.total: ${data.minions.total}`);
    console.log(`   minions.online: ${data.minions.online}`);
    
    // 修复验证：demo 应该是 false
    expect(data.demo).toBe(false);
    
    // 应该有真实的 minion 数据
    expect(data.minions.total).toBeGreaterThan(0);
    expect(data.minions.online).toBeGreaterThan(0);
    
    // Master 状态应该是 connected 或 running
    expect(['connected', 'running']).toContain(data.master_status);
    
    // API 状态应该是 connected
    expect(data.api_status).toBe('connected');
    
    // 应该有 minion 列表
    expect(data.minion_list).toBeDefined();
    expect(data.minion_list.length).toBeGreaterThan(0);
    
    console.log(`\n✅ 修复验证通过！`);
    console.log(`   - 不是演示模式 (demo=false)`);
    console.log(`   - 找到 ${data.minions.total} 个 minions`);
    console.log(`   - ${data.minions.online} 个 minions 在线`);
    console.log(`   - Master 状态: ${data.master_status}`);
  });

  test('对比两个 API 的数据一致性', async ({ request }) => {
    console.log('\n🔍 测试 2: 对比两个 API 的数据一致性');
    
    // 获取 /api/saltstack/status
    const statusResponse = await request.get(`${API_BASE}/api/saltstack/status`, {
      headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    const statusResult = await statusResponse.json();
    const statusData = statusResult.data;
    
    // 获取 /api/slurm/saltstack/integration
    const integrationResponse = await request.get(`${API_BASE}/api/slurm/saltstack/integration`, {
      headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    const integrationResult = await integrationResponse.json();
    const integrationData = integrationResult.data;
    
    console.log('\n📊 数据对比:');
    console.log(`/api/saltstack/status:`);
    console.log(`   connected_minions: ${statusData.connected_minions}`);
    console.log(`   status: ${statusData.status}`);
    console.log(`   accepted_keys: ${statusData.accepted_keys?.length || 0} keys`);
    
    console.log(`\n/api/slurm/saltstack/integration:`);
    console.log(`   minions.online: ${integrationData.minions.online}`);
    console.log(`   master_status: ${integrationData.master_status}`);
    console.log(`   minion_list: ${integrationData.minion_list?.length || 0} items`);
    console.log(`   demo: ${integrationData.demo}`);
    
    // 验证数据一致性
    // 1. minion 数量应该一致
    expect(integrationData.minions.online).toBe(statusData.connected_minions);
    
    // 2. 状态映射应该正确
    if (statusData.status === 'connected') {
      expect(['connected', 'running']).toContain(integrationData.master_status);
    }
    
    // 3. minion 列表数量应该匹配
    const statusKeysCount = statusData.accepted_keys?.length || 0;
    const integrationMinionCount = integrationData.minion_list?.length || 0;
    expect(integrationMinionCount).toBe(statusKeysCount);
    
    console.log(`\n✅ 数据一致性验证通过！`);
    console.log(`   - Minion 数量匹配`);
    console.log(`   - 状态映射正确`);
    console.log(`   - 不再返回演示数据`);
  });

  test('验证 minion 列表详细信息', async ({ request }) => {
    console.log('\n📋 测试 3: 验证 minion 列表详细信息');
    
    const response = await request.get(`${API_BASE}/api/slurm/saltstack/integration`, {
      headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    const result = await response.json();
    const data = result.data;
    
    console.log(`\n找到 ${data.minion_list.length} 个 minions:`);
    
    data.minion_list.forEach((minion, index) => {
      console.log(`\n${index + 1}. ${minion.name}`);
      console.log(`   ID: ${minion.id}`);
      console.log(`   状态: ${minion.status}`);
      
      // 验证每个 minion 的必要字段
      expect(minion.id).toBeTruthy();
      expect(minion.name).toBeTruthy();
      expect(minion.status).toBeTruthy();
      expect(['online', 'offline', 'pending']).toContain(minion.status);
    });
    
    // 验证包含我们的测试节点
    const minionIds = data.minion_list.map(m => m.id);
    console.log(`\nMinion IDs: ${minionIds.join(', ')}`);
    
    // 应该包含 salt-master-local 或测试节点
    const hasTestMinions = minionIds.some(id => 
      id.includes('test-ssh') || id.includes('salt-master')
    );
    expect(hasTestMinions).toBe(true);
    
    console.log(`\n✅ Minion 列表验证通过！`);
  });

  test('验证服务状态信息', async ({ request }) => {
    console.log('\n🔧 测试 4: 验证服务状态信息');
    
    const response = await request.get(`${API_BASE}/api/slurm/saltstack/integration`, {
      headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    const result = await response.json();
    const data = result.data;
    
    console.log('\n服务状态:');
    console.log(`   enabled: ${data.enabled}`);
    console.log(`   master_status: ${data.master_status}`);
    console.log(`   api_status: ${data.api_status}`);
    
    if (data.services) {
      console.log('\n服务详情:');
      Object.entries(data.services).forEach(([service, status]) => {
        console.log(`   ${service}: ${status}`);
      });
    }
    
    // 验证服务状态
    expect(data.enabled).toBe(true); // 应该启用
    expect(['connected', 'running']).toContain(data.master_status);
    expect(data.api_status).toBe('connected');
    
    console.log(`\n✅ 服务状态验证通过！`);
  });

  test('性能测试：API 响应时间', async ({ request }) => {
    console.log('\n⚡ 测试 5: API 响应时间性能测试');
    
    const iterations = 5;
    const times = [];
    
    for (let i = 0; i < iterations; i++) {
      const start = Date.now();
      const response = await request.get(`${API_BASE}/api/slurm/saltstack/integration`, {
        headers: { 'Authorization': `Bearer ${adminToken}` }
      });
      const elapsed = Date.now() - start;
      times.push(elapsed);
      
      expect(response.status()).toBe(200);
      console.log(`   请求 ${i + 1}: ${elapsed}ms`);
    }
    
    const avgTime = times.reduce((a, b) => a + b, 0) / times.length;
    const maxTime = Math.max(...times);
    const minTime = Math.min(...times);
    
    console.log(`\n性能统计:`);
    console.log(`   平均响应时间: ${avgTime.toFixed(0)}ms`);
    console.log(`   最快: ${minTime}ms`);
    console.log(`   最慢: ${maxTime}ms`);
    
    // 响应时间应该在合理范围内（<3秒）
    expect(avgTime).toBeLessThan(3000);
    
    console.log(`\n✅ 性能测试通过！`);
  });
});

test.describe('前端页面显示验证', () => {
  test('验证 /slurm 页面 SaltStack 集成卡片显示', async ({ page }) => {
    console.log('\n🌐 测试 6: 验证前端页面显示');
    
    // 登录
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(`${BASE_URL}/`);
    console.log('✅ 登录成功');
    
    // 访问 /slurm 页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000); // 等待数据加载
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-integration-fixed.png',
      fullPage: true 
    });
    console.log('✅ 页面截图已保存');
    
    // 检查 SaltStack 集成卡片
    const saltStackCard = page.locator('text=SaltStack 集成').first();
    await expect(saltStackCard).toBeVisible();
    console.log('✅ 找到 SaltStack 集成卡片');
    
    // 检查状态显示
    const pageContent = await page.content();
    
    // 不应该显示 "演示模式" 或 "API 不可用"
    expect(pageContent).not.toContain('演示模式');
    expect(pageContent).not.toContain('API 不可用');
    
    console.log('✅ 页面不再显示演示模式');
    
    // 应该显示真实的 minion 数量
    const hasRealData = 
      pageContent.includes('在线') || 
      pageContent.includes('Minion') ||
      pageContent.includes('test-ssh');
    
    expect(hasRealData).toBe(true);
    console.log('✅ 页面显示真实的 minion 数据');
    
    console.log(`\n✅ 前端页面验证通过！`);
  });
});
