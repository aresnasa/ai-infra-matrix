// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';

test.describe('诊断 Monitoring 和 SaltStack 问题', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"], input[id*="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"], button:has-text("登")');
    await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
    await page.waitForLoadState('networkidle');
  });

  test('诊断 Monitoring 页面 404 问题', async ({ page }) => {
    console.log('\n=== 诊断 Monitoring 页面 ===');
    
    // 直接访问 /monitoring
    await page.goto(`${BASE_URL}/monitoring`);
    await page.waitForTimeout(3000);
    
    const currentUrl = page.url();
    console.log('当前 URL:', currentUrl);
    
    // 检查页面内容
    const pageContent = await page.content();
    const has404 = pageContent.includes('404') || pageContent.includes('不存在');
    console.log('页面是否包含 404:', has404);
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/monitoring-diagnose.png', fullPage: true });
    
    // 检查 iframe
    const iframes = await page.locator('iframe').all();
    console.log('iframe 数量:', iframes.length);
    
    for (let i = 0; i < iframes.length; i++) {
      const src = await iframes[i].getAttribute('src');
      console.log(`iframe[${i}] src:`, src);
    }
    
    // 直接访问 nightingale
    console.log('\n--- 直接访问 /nightingale/ ---');
    const n9eResponse = await page.goto(`${BASE_URL}/nightingale/`);
    console.log('Nightingale 根路径状态:', n9eResponse?.status());
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'test-screenshots/nightingale-root.png', fullPage: true });
    
    // 检查页面中的链接
    const links = await page.locator('a[href]').all();
    const linkHrefs = [];
    for (const link of links.slice(0, 10)) {
      const href = await link.getAttribute('href');
      linkHrefs.push(href);
    }
    console.log('页面中的链接 (前10个):', linkHrefs);
    
    // 检查链接是否包含 /nightingale 前缀
    const hasCorrectPrefix = linkHrefs.some(href => href && href.startsWith('/nightingale'));
    console.log('链接是否有 /nightingale 前缀:', hasCorrectPrefix);
  });

  test('诊断 SaltStack 监控数据问题', async ({ page }) => {
    console.log('\n=== 诊断 SaltStack 监控数据 ===');
    
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    // 截图当前状态
    await page.screenshot({ path: 'test-screenshots/saltstack-diagnose.png', fullPage: true });
    
    // 获取 API 数据
    const statusData = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        const res = await fetch('/api/saltstack/status', { headers });
        return await res.json();
      } catch (e) {
        return { error: e.message };
      }
    });
    
    console.log('SaltStack Status API 响应:');
    console.log(JSON.stringify(statusData, null, 2));
    
    // 检查监控字段
    if (statusData.data) {
      const data = statusData.data;
      console.log('\n监控字段检查:');
      console.log('  cpu_usage:', data.cpu_usage, '(存在:', 'cpu_usage' in data, ')');
      console.log('  memory_usage:', data.memory_usage, '(存在:', 'memory_usage' in data, ')');
      console.log('  active_connections:', data.active_connections, '(存在:', 'active_connections' in data, ')');
      console.log('  network_bandwidth:', data.network_bandwidth, '(存在:', 'network_bandwidth' in data, ')');
      console.log('  metrics_source:', data.metrics_source, '(存在:', 'metrics_source' in data, ')');
    }
    
    // 检查页面上显示的监控数据
    const pageText = await page.textContent('body');
    console.log('\n页面是否包含 CPU:', pageText?.includes('CPU'));
    console.log('页面是否包含 Memory/内存:', pageText?.includes('Memory') || pageText?.includes('内存'));
  });
});
