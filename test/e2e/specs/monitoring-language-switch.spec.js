/**
 * 监控页面 iframe 语言切换测试
 * 测试 Nightingale iframe 能否正确响应语言切换
 */
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.TEST_BASE_URL || 'http://192.168.3.101:8080';

test.describe('监控页面 iframe 语言切换测试', () => {
  test.setTimeout(120000);
  
  test.beforeEach(async ({ page }) => {
    // 先访问项目页面确保已登录
    await page.goto(`${BASE_URL}/project`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(2000);
    
    // 如果跳转到登录页，执行登录
    if (page.url().includes('/login')) {
      console.log('[Test] Logging in...');
      await page.fill('input[id="username"]', 'admin');
      await page.fill('input[id="password"]', 'admin123');
      await page.click('button[type="submit"]');
      await page.waitForURL('**/project**', { timeout: 15000 });
      console.log('[Test] Login successful');
    }
    
    // 确保登录完成
    await page.waitForTimeout(2000);
  });

  test('访问监控页面并检查 iframe URL 参数', async ({ page }) => {
    // 导航到监控页面
    console.log('[Test] Navigating to monitoring page...');
    await page.goto(`${BASE_URL}/monitoring`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    
    // 等待更长时间让 React 渲染完成
    await page.waitForTimeout(8000);
    
    // 截图当前状态
    await page.screenshot({ path: 'test-screenshots/monitoring-language-01-initial.png', fullPage: true });
    
    // 打印页面内容以调试
    const pageTitle = await page.title();
    console.log('[Test] Page title:', pageTitle);
    
    const pageUrl = page.url();
    console.log('[Test] Current URL:', pageUrl);
    
    // 检查页面是否有错误提示
    const errorAlert = page.locator('.ant-alert-error');
    if (await errorAlert.isVisible()) {
      const errorText = await errorAlert.textContent();
      console.log('[Test] Error alert:', errorText);
    }
    
    // 检查加载状态
    const spinner = page.locator('.ant-spin');
    const spinnerVisible = await spinner.isVisible();
    console.log('[Test] Spinner visible:', spinnerVisible);
    
    // 等待 iframe 出现
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    const iframeCount = await iframe.count();
    console.log('[Test] iframe count:', iframeCount);
    
    if (iframeCount > 0) {
      await expect(iframe).toBeVisible({ timeout: 30000 });
      
      // 获取 iframe 的 src 属性
      const iframeSrc = await iframe.getAttribute('src');
      console.log('[Test] iframe src:', iframeSrc);
      
      // 检查 URL 是否包含语言参数
      expect(iframeSrc).toContain('lang=');
      expect(iframeSrc).toContain('themeMode=');
      
      // 解析参数
      const url = new URL(iframeSrc);
      const langParam = url.searchParams.get('lang');
      const themeParam = url.searchParams.get('themeMode');
      console.log('[Test] lang:', langParam, 'themeMode:', themeParam);
    } else {
      console.log('[Test] No iframe found on the page');
      
      // 检查页面 HTML 内容
      const pageContent = await page.content();
      console.log('[Test] Page contains "iframe":', pageContent.includes('iframe'));
      console.log('[Test] Page contains "Nightingale":', pageContent.includes('Nightingale'));
      console.log('[Test] Page contains "monitoring":', pageContent.includes('monitoring'));
      
      // 检查 React 渲染的卡片
      const card = page.locator('.ant-card');
      const cardCount = await card.count();
      console.log('[Test] Card count:', cardCount);
      
      if (cardCount > 0) {
        const cardTitle = await card.first().locator('.ant-card-head-title').textContent();
        console.log('[Test] Card title:', cardTitle);
      }
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/monitoring-language-02-iframe-loaded.png', fullPage: true });
  });

  test('检测 iframe 内容和背景样式', async ({ page }) => {
    // 导航到监控页面
    console.log('[Test] Navigating to monitoring page...');
    await page.goto(`${BASE_URL}/monitoring`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(10000);
    
    // 获取 iframe
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    const iframeCount = await iframe.count();
    console.log('[Test] iframe count:', iframeCount);
    
    if (iframeCount > 0) {
      const iframeSrc = await iframe.getAttribute('src');
      console.log('[Test] iframe src:', iframeSrc);
      
      // 获取 iframe 样式
      const iframeStyle = await iframe.getAttribute('style');
      console.log('[Test] iframe style:', iframeStyle);
      
      // 检查 Card 组件样式
      const card = page.locator('.ant-card').first();
      if (await card.isVisible()) {
        const cardBg = await card.evaluate(el => window.getComputedStyle(el).backgroundColor);
        console.log('[Test] Card background:', cardBg);
      }
    } else {
      // 打印更多调试信息
      const allIframes = page.locator('iframe');
      const allIframeCount = await allIframes.count();
      console.log('[Test] Total iframes on page:', allIframeCount);
      
      for (let i = 0; i < allIframeCount; i++) {
        const iframeSrc = await allIframes.nth(i).getAttribute('src');
        const iframeTitle = await allIframes.nth(i).getAttribute('title');
        console.log(`[Test] iframe ${i}: title="${iframeTitle}", src="${iframeSrc}"`);
      }
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/monitoring-language-03-styles.png', fullPage: true });
  });
});
