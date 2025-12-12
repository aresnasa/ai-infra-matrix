// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';

test.describe('Nightingale 页面加载诊断', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"], input[id*="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"], button:has-text("登")');
    await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
  });

  test('诊断 Nightingale iframe 加载问题', async ({ page }) => {
    console.log('\n=== 诊断 Nightingale iframe 加载 ===');
    
    await page.goto(`${BASE_URL}/monitoring`);
    await page.waitForTimeout(5000);
    
    // 截图主页面
    await page.screenshot({ path: 'test-screenshots/monitoring-main.png', fullPage: true });
    
    // 检查 iframe
    const iframes = await page.locator('iframe').all();
    console.log('iframe 数量:', iframes.length);
    
    if (iframes.length > 0) {
      const iframeSrc = await iframes[0].getAttribute('src');
      console.log('iframe src:', iframeSrc);
      
      // 进入 iframe 内部
      const frame = page.frameLocator('iframe').first();
      
      // 等待 iframe 加载
      await page.waitForTimeout(3000);
      
      // 检查 iframe 内部是否有 loading/logo 元素
      try {
        const loadingElement = frame.locator('.ant-spin, .loading, [class*="loading"], [class*="spin"]');
        const loadingCount = await loadingElement.count();
        console.log('Loading 元素数量:', loadingCount);
        
        // 检查是否有 logo
        const logoElement = frame.locator('img[src*="logo"], [class*="logo"]');
        const logoCount = await logoElement.count();
        console.log('Logo 元素数量:', logoCount);
        
        // 获取 iframe 内容的文本
        const bodyText = await frame.locator('body').textContent({ timeout: 5000 });
        console.log('iframe body 文本 (前500字符):', bodyText?.substring(0, 500));
        
        // 检查是否有错误信息
        const hasError = bodyText?.includes('error') || bodyText?.includes('Error') || bodyText?.includes('failed');
        console.log('是否包含错误信息:', hasError);
        
      } catch (e) {
        console.log('访问 iframe 内容失败:', e.message);
      }
    }
    
    // 直接访问 nightingale 检查
    console.log('\n--- 直接访问 Nightingale 页面 ---');
    const n9ePage = await page.context().newPage();
    await n9ePage.goto(`${BASE_URL}/nightingale/`);
    await n9ePage.waitForTimeout(5000);
    
    // 截图 nightingale 页面
    await n9ePage.screenshot({ path: 'test-screenshots/nightingale-direct.png', fullPage: true });
    
    // 检查页面内容
    const n9eContent = await n9ePage.content();
    console.log('Nightingale 页面 HTML 长度:', n9eContent.length);
    
    // 检查是否有 JS/CSS 资源加载
    const scripts = await n9ePage.locator('script[src]').all();
    const stylesheets = await n9ePage.locator('link[rel="stylesheet"]').all();
    console.log('script 标签数量:', scripts.length);
    console.log('stylesheet 标签数量:', stylesheets.length);
    
    // 检查资源路径
    for (const script of scripts.slice(0, 5)) {
      const src = await script.getAttribute('src');
      console.log('script src:', src);
    }
    
    // 检查控制台错误
    const consoleMessages = [];
    n9ePage.on('console', msg => {
      if (msg.type() === 'error') {
        consoleMessages.push(msg.text());
      }
    });
    
    await n9ePage.reload();
    await n9ePage.waitForTimeout(3000);
    
    console.log('\n控制台错误:', consoleMessages.slice(0, 10));
    
    await n9ePage.close();
  });

  test('检查 Nightingale 静态资源路径', async ({ page }) => {
    console.log('\n=== 检查 Nightingale 静态资源 ===');
    
    // 监听网络请求
    const failedRequests = [];
    page.on('requestfailed', request => {
      failedRequests.push({
        url: request.url(),
        failure: request.failure()?.errorText
      });
    });
    
    await page.goto(`${BASE_URL}/nightingale/`);
    await page.waitForTimeout(5000);
    
    console.log('失败的请求:');
    for (const req of failedRequests) {
      console.log(`  ${req.url} - ${req.failure}`);
    }
    
    // 检查页面是否正确渲染
    const bodyContent = await page.textContent('body');
    console.log('\n页面文本内容 (前300字符):', bodyContent?.substring(0, 300));
    
    // 检查是否只显示 logo (加载中状态)
    const hasOnlyLogo = bodyContent?.trim().length < 100;
    console.log('是否只显示 logo (内容少):', hasOnlyLogo);
    
    // 查看 HTML 结构
    const html = await page.content();
    
    // 检查 base 标签
    const baseMatch = html.match(/<base[^>]*>/);
    console.log('base 标签:', baseMatch ? baseMatch[0] : '无');
    
    // 检查资源引用
    const assetPaths = html.match(/src="([^"]+)"|href="([^"]+\.css)"/g);
    console.log('\n资源路径 (前10个):');
    if (assetPaths) {
      assetPaths.slice(0, 10).forEach(p => console.log('  ', p));
    }
  });
});
