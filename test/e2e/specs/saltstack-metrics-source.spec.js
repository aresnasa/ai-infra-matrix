// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';

test.describe('SaltStack 监控数据来源诊断', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"], input[id*="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"], button:has-text("登")');
    await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
  });

  test('检查 VictoriaMetrics 连接状态', async ({ page }) => {
    console.log('\n=== 检查 VictoriaMetrics 状态 ===');
    
    // 检查健康检查接口
    const healthData = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        const res = await fetch('/api/health', { headers });
        return await res.json();
      } catch (e) {
        return { error: String(e) };
      }
    });
    
    console.log('健康检查响应:', JSON.stringify(healthData, null, 2));
    
    // 检查 VictoriaMetrics 是否可达
    const vmTestData = await page.evaluate(async () => {
      try {
        // 尝试直接访问 VictoriaMetrics
        const res = await fetch('/api/proxy/victoriametrics/api/v1/query?query=up');
        return await res.json();
      } catch (e) {
        return { error: String(e) };
      }
    });
    
    console.log('VictoriaMetrics 查询结果:', JSON.stringify(vmTestData, null, 2));
  });

  test('检查 Docker API 和 Salt 容器状态', async ({ page }) => {
    console.log('\n=== 检查 Docker 和 Salt 容器状态 ===');
    
    // 检查后端是否能获取 Docker 信息
    const dockerData = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        // 尝试调用系统信息接口
        const res = await fetch('/api/system/info', { headers });
        return await res.json();
      } catch (e) {
        return { error: String(e) };
      }
    });
    
    console.log('系统信息:', JSON.stringify(dockerData, null, 2));
  });

  test('刷新 SaltStack 状态（绕过缓存）', async ({ page }) => {
    console.log('\n=== 刷新 SaltStack 状态 ===');
    
    // 先获取当前缓存状态
    const cachedData = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        const res = await fetch('/api/saltstack/status', { headers });
        return await res.json();
      } catch (e) {
        return { error: String(e) };
      }
    });
    
    console.log('缓存状态:', JSON.stringify(cachedData, null, 2));
    console.log('是否使用缓存:', cachedData.cached);
    
    // 等待缓存过期（如果 TTL 较短）
    console.log('\n等待 3 秒后重新请求...');
    await page.waitForTimeout(3000);
    
    // 尝试刷新请求
    const freshData = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 
        'Authorization': `Bearer ${token}`,
        'Cache-Control': 'no-cache'
      } : {
        'Cache-Control': 'no-cache'
      };
      
      try {
        const res = await fetch('/api/saltstack/status?refresh=true', { headers });
        return await res.json();
      } catch (e) {
        return { error: String(e) };
      }
    });
    
    console.log('刷新后状态:', JSON.stringify(freshData, null, 2));
    console.log('metrics_source:', freshData?.data?.metrics_source);
  });

  test('检查监控页面数据显示', async ({ page }) => {
    console.log('\n=== 检查 SaltStack 页面显示 ===');
    
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/saltstack-monitoring-display.png', fullPage: true });
    
    // 查找监控数据卡片
    const cpuElement = await page.locator('text=/CPU.*?\\d+%?/i').first();
    const memElement = await page.locator('text=/内存|Memory.*?\\d+%?/i').first();
    
    if (await cpuElement.count() > 0) {
      console.log('找到 CPU 显示:', await cpuElement.textContent());
    } else {
      console.log('未找到 CPU 显示元素');
    }
    
    if (await memElement.count() > 0) {
      console.log('找到 Memory 显示:', await memElement.textContent());
    } else {
      console.log('未找到 Memory 显示元素');
    }
    
    // 检查是否有 "数据来源" 或 "metrics_source" 显示
    const sourceElement = await page.locator('text=/数据来源|metrics.*source/i').first();
    if (await sourceElement.count() > 0) {
      console.log('数据来源显示:', await sourceElement.textContent());
    }
    
    // 打印页面上所有包含数字百分比的文本
    const percentages = await page.locator('text=/\\d+%/').all();
    console.log('\n页面上显示的百分比值:');
    for (const p of percentages.slice(0, 10)) {
      const text = await p.textContent();
      console.log('  -', text);
    }
  });
});
