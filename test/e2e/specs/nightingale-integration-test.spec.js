#!/usr/bin/env node
/**
 * Nightingale Iframe 集成测试
 * 测试 MonitoringPage 和 SLURM Dashboard 中的 Nightingale iframe
 */

const { test, expect } = require('@playwright/test');

const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://192.168.18.114:8080',
  adminUser: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  },
};

async function login(page, username, password) {
  await page.goto('/');
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

test.describe('Nightingale Iframe 完整测试', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  });

  test('1. MonitoringPage - 验证 iframe 使用 /nightingale/ 代理路径', async ({ page }) => {
    console.log('\n=== 测试 MonitoringPage ===');
    
    // 导航到监控页面
    await page.goto('/monitoring');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    
    // 检查 iframe
    const iframe = page.locator('iframe');
    const iframeCount = await iframe.count();
    console.log(`发现 ${iframeCount} 个 iframe`);
    expect(iframeCount).toBe(1);
    
    // 检查 iframe src
    const iframeSrc = await iframe.getAttribute('src');
    console.log(`Iframe src: ${iframeSrc}`);
    
    // 验证使用代理路径
    expect(iframeSrc).toContain('/nightingale/');
    expect(iframeSrc).not.toContain(':17000');
    
    console.log('✓ MonitoringPage 使用正确的代理路径');
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-page-nightingale.png',
      fullPage: true 
    });
  });

  test('2. SLURM Dashboard - 验证监控仪表板使用 Nightingale', async ({ page }) => {
    console.log('\n=== 测试 SLURM Dashboard 监控仪表板 ===');
    
    // 导航到 SLURM 页面
    await page.goto('/slurm');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    
    // 切换到监控仪表板 tab
    const monitoringTab = page.locator('text=监控仪表板');
    if (await monitoringTab.isVisible()) {
      console.log('✓ 找到监控仪表板 Tab');
      await monitoringTab.click();
      await page.waitForTimeout(2000);
      
      // 检查 iframe
      const iframe = page.locator('iframe#slurm-dashboard-iframe');
      const isVisible = await iframe.isVisible();
      console.log(`Iframe 可见: ${isVisible}`);
      
      if (isVisible) {
        const iframeSrc = await iframe.getAttribute('src');
        console.log(`Iframe src: ${iframeSrc}`);
        
        // 验证使用 Nightingale 代理路径
        expect(iframeSrc).toContain('/nightingale/');
        expect(iframeSrc).not.toContain(':3000'); // 不应该是 Grafana
        
        console.log('✓ SLURM Dashboard 使用 Nightingale');
      } else {
        console.log('❌ Iframe 不可见');
      }
      
      // 截图
      await page.screenshot({ 
        path: 'test-screenshots/slurm-dashboard-nightingale.png',
        fullPage: true 
      });
    } else {
      console.log('⚠ 未找到监控仪表板 Tab');
    }
  });

  test('3. 测试 Nightingale 代理路径是否可访问', async ({ page }) => {
    console.log('\n=== 测试 Nightingale 代理路径 ===');
    
    const consoleMessages = {
      errors: [],
      warnings: []
    };
    
    // 监听控制台消息
    page.on('console', msg => {
      const text = msg.text();
      if (msg.type() === 'error') {
        consoleMessages.errors.push(text);
      } else if (msg.type() === 'warning') {
        consoleMessages.warnings.push(text);
      }
    });
    
    // 监听网络请求
    const nightingaleRequests = [];
    page.on('response', response => {
      if (response.url().includes('/nightingale/')) {
        nightingaleRequests.push({
          url: response.url(),
          status: response.status(),
          headers: response.headers()
        });
        console.log(`[响应] ${response.status()} ${response.url()}`);
      }
    });
    
    // 访问监控页面
    await page.goto('/monitoring');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(5000); // 等待 iframe 加载
    
    console.log(`\n捕获到 ${nightingaleRequests.length} 个 Nightingale 请求`);
    
    // 检查主要请求
    const mainRequest = nightingaleRequests.find(r => r.url.endsWith('/nightingale/'));
    if (mainRequest) {
      console.log(`主页面状态: ${mainRequest.status}`);
      expect(mainRequest.status).toBeLessThan(400);
    }
    
    // 检查控制台错误
    console.log(`\n控制台错误: ${consoleMessages.errors.length} 个`);
    if (consoleMessages.errors.length > 0) {
      console.log('错误列表:');
      consoleMessages.errors.slice(0, 5).forEach((error, i) => {
        console.log(`  ${i + 1}. ${error}`);
      });
    }
    
    // 401 错误检查
    const has401 = consoleMessages.errors.some(e => e.includes('401') || e.includes('Unauthorized'));
    if (has401) {
      console.log('❌ 发现 401 认证错误 - ProxyAuth 可能未正确配置');
    } else {
      console.log('✓ 没有 401 认证错误');
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-proxy-test.png',
      fullPage: true 
    });
  });

  test('4. 综合检查 - 完整诊断', async ({ page }) => {
    console.log('\n=== 综合诊断 ===\n');
    
    const diagnostics = {
      monitoringPage: { route: false, iframe: false, correctUrl: false },
      slurmDashboard: { tab: false, iframe: false, correctUrl: false },
      proxyPath: { accessible: false, authenticated: false },
      errors: []
    };
    
    // 监听错误
    page.on('console', msg => {
      if (msg.type() === 'error') {
        diagnostics.errors.push(msg.text());
      }
    });
    
    // 1. 测试 MonitoringPage
    try {
      await page.goto('/monitoring', { timeout: 10000 });
      diagnostics.monitoringPage.route = true;
      
      const iframe = page.locator('iframe');
      diagnostics.monitoringPage.iframe = await iframe.count() > 0;
      
      if (diagnostics.monitoringPage.iframe) {
        const src = await iframe.getAttribute('src');
        diagnostics.monitoringPage.correctUrl = src.includes('/nightingale/') && !src.includes(':17000');
      }
    } catch (e) {
      console.log(`MonitoringPage 测试失败: ${e.message}`);
    }
    
    // 2. 测试 SLURM Dashboard
    try {
      await page.goto('/slurm', { timeout: 10000 });
      await page.waitForTimeout(1000);
      
      const tab = page.locator('text=监控仪表板');
      diagnostics.slurmDashboard.tab = await tab.isVisible();
      
      if (diagnostics.slurmDashboard.tab) {
        await tab.click();
        await page.waitForTimeout(1000);
        
        const iframe = page.locator('iframe#slurm-dashboard-iframe');
        diagnostics.slurmDashboard.iframe = await iframe.isVisible();
        
        if (diagnostics.slurmDashboard.iframe) {
          const src = await iframe.getAttribute('src');
          diagnostics.slurmDashboard.correctUrl = src.includes('/nightingale/') && !src.includes(':3000');
        }
      }
    } catch (e) {
      console.log(`SLURM Dashboard 测试失败: ${e.message}`);
    }
    
    // 3. 测试代理路径
    try {
      let responseStatus = 0;
      page.on('response', response => {
        if (response.url().includes('/nightingale/') && response.url().split('/nightingale/')[1] === '') {
          responseStatus = response.status();
        }
      });
      
      await page.goto('/monitoring', { timeout: 10000 });
      await page.waitForTimeout(3000);
      
      diagnostics.proxyPath.accessible = responseStatus === 200;
      diagnostics.proxyPath.authenticated = !diagnostics.errors.some(e => 
        e.includes('401') || e.includes('Unauthorized')
      );
    } catch (e) {
      console.log(`代理路径测试失败: ${e.message}`);
    }
    
    // 输出诊断结果
    console.log('诊断结果:');
    console.log('==========================================');
    console.log('1. MonitoringPage');
    console.log(`   - 路由可访问: ${diagnostics.monitoringPage.route ? '✓' : '✗'}`);
    console.log(`   - Iframe 存在: ${diagnostics.monitoringPage.iframe ? '✓' : '✗'}`);
    console.log(`   - 使用代理路径: ${diagnostics.monitoringPage.correctUrl ? '✓' : '✗'}`);
    
    console.log('\n2. SLURM Dashboard');
    console.log(`   - 监控Tab存在: ${diagnostics.slurmDashboard.tab ? '✓' : '✗'}`);
    console.log(`   - Iframe 存在: ${diagnostics.slurmDashboard.iframe ? '✓' : '✗'}`);
    console.log(`   - 使用 Nightingale: ${diagnostics.slurmDashboard.correctUrl ? '✓' : '✗'}`);
    
    console.log('\n3. Nightingale 代理');
    console.log(`   - 路径可访问: ${diagnostics.proxyPath.accessible ? '✓' : '✗'}`);
    console.log(`   - 已认证: ${diagnostics.proxyPath.authenticated ? '✓' : '✗'}`);
    
    console.log(`\n4. 错误数量: ${diagnostics.errors.length}`);
    if (diagnostics.errors.length > 0) {
      console.log('   前5个错误:');
      diagnostics.errors.slice(0, 5).forEach((e, i) => {
        console.log(`   ${i + 1}. ${e.substring(0, 100)}`);
      });
    }
    
    console.log('\n==========================================');
    
    // 保存最终截图
    await page.screenshot({ 
      path: 'test-screenshots/nightingale-final-diagnostic.png',
      fullPage: true 
    });
    
    // 验证关键功能
    expect(diagnostics.monitoringPage.route).toBe(true);
    expect(diagnostics.monitoringPage.iframe).toBe(true);
    expect(diagnostics.monitoringPage.correctUrl).toBe(true);
  });
});
