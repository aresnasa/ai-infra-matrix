// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';

test.describe('Monitoring 页面验证', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"], input[id*="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"], button:has-text("登")');
    await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
  });

  test('验证 Monitoring 页面正常加载', async ({ page }) => {
    console.log('\n=== 验证 Monitoring 页面 ===');
    
    await page.goto(`${BASE_URL}/monitoring`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/monitoring-final.png', fullPage: true });
    
    // 检查 iframe 是否正确加载
    const iframes = await page.locator('iframe').all();
    console.log('iframe 数量:', iframes.length);
    
    if (iframes.length > 0) {
      const frame = page.frameLocator('iframe').first();
      
      // 等待 iframe 内容加载
      await page.waitForTimeout(3000);
      
      // 检查 iframe 内是否有 Nightingale 的内容
      try {
        // 检查是否有菜单项
        const menuItems = frame.locator('a[href*="/"], .ant-menu-item, [role="menuitem"]');
        const menuCount = await menuItems.count();
        console.log('菜单项数量:', menuCount);
        
        // 检查是否有 loading spinner
        const loadingSpinner = frame.locator('.ant-spin-spinning, .loading-spinner');
        const isLoading = await loadingSpinner.count() > 0;
        console.log('是否正在加载:', isLoading);
        
        // 获取主要内容区域文本
        const mainContent = await frame.locator('.main-content, [class*="content"], main').first().textContent({ timeout: 5000 }).catch(() => '');
        console.log('主要内容区域文本 (前200字符):', mainContent.substring(0, 200));
        
        // 检查页面标题
        const title = await frame.locator('h1, h2, .page-title, .ant-page-header-heading-title').first().textContent({ timeout: 5000 }).catch(() => '');
        console.log('页面标题:', title);
        
      } catch (e) {
        console.log('访问 iframe 内容时出错:', String(e));
      }
    }
    
    // 验证页面是否正常工作
    const pageContent = await page.content();
    const has404 = pageContent.includes('404') && pageContent.includes('不存在');
    expect(has404).toBe(false);
    
    console.log('\n✅ Monitoring 页面加载验证完成');
  });

  test('验证 Nightingale 直接访问', async ({ page }) => {
    console.log('\n=== 验证 Nightingale 直接访问 ===');
    
    // 直接访问 Nightingale
    await page.goto(`${BASE_URL}/nightingale/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/nightingale-final.png', fullPage: true });
    
    // 检查是否加载了完整的 UI
    const bodyText = await page.textContent('body');
    console.log('页面文本长度:', bodyText?.length);
    
    // 应该有丰富的内容（不只是 logo）
    expect(bodyText?.length).toBeGreaterThan(100);
    
    // 检查是否有关键菜单项
    const hasMetricExplorer = bodyText?.includes('指标') || bodyText?.includes('metric');
    const hasDashboard = bodyText?.includes('仪表盘') || bodyText?.includes('dashboard');
    const hasAlert = bodyText?.includes('告警') || bodyText?.includes('alert');
    
    console.log('包含指标菜单:', hasMetricExplorer);
    console.log('包含仪表盘菜单:', hasDashboard);
    console.log('包含告警菜单:', hasAlert);
    
    // 至少应该有一些关键功能菜单
    expect(hasMetricExplorer || hasDashboard || hasAlert).toBe(true);
    
    console.log('\n✅ Nightingale 直接访问验证完成');
  });
});
