import { test, expect } from '@playwright/test';

test.describe('Monitoring Page with Header and Iframe', () => {
  test('should display monitoring page with header and iframe on initial load', async ({ page }) => {
    // 访问监控入口
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    // 等待 React 应用加载
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000); // 给 React 路由时间加载

    // 截图查看实际页面
    await page.screenshot({ path: 'test-results/monitoring-initial.png', fullPage: true });

    // 应该保持在 /monitoring 路径（前端 SPA 路由）
    await expect(page).toHaveURL('http://192.168.0.200:8080/monitoring');
    
    // 检查 React 应用的根元素已加载
    const root = page.locator('#root');
    await expect(root).toBeVisible({ timeout: 5000 });
    
    // 检查是否有内容加载（Ant Design Card 或其他组件）
    const hasContent = page.locator('.ant-card, .ant-layout, main, article');
    await expect(hasContent.first()).toBeVisible({ timeout: 10000 });
    
    // 等待 iframe 元素出现（MonitoringPage 可能需要时间加载）
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframeElement).toBeVisible({ timeout: 15000 });
    
    // 获取 iframe 的 src
    const iframeSrc = await iframeElement.getAttribute('src');
    console.log('✓ Iframe src:', iframeSrc);
    console.log('✓ Page loaded with iframe');
  });

  test('should maintain header and iframe after page refresh', async ({ page }) => {
    // 首次访问
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    // 等待 React 应用和 iframe 加载
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframeElement).toBeVisible({ timeout: 15000 });
    
    console.log('✓ Initial load successful');

    // 刷新页面
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // 截图查看刷新后的页面
    await page.screenshot({ path: 'test-results/monitoring-after-refresh.png', fullPage: true });
    
    // 验证 URL 保持不变
    await expect(page).toHaveURL('http://192.168.0.200:8080/monitoring');
    
    // 检查内容仍然存在
    const hasContent = page.locator('.ant-card, .ant-layout, main');
    await expect(hasContent.first()).toBeVisible({ timeout: 10000 });
    
    // 验证 iframe 仍然存在并可见
    await expect(iframeElement).toBeVisible({ timeout: 15000 });
    
    // 检查页面不包含错误信息
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).not.toContain('404');
    expect(bodyText).not.toContain('Not Found');
    
    console.log('✓ Page refresh maintains iframe');
  });

  test('should not display database errors', async ({ page }) => {
    // 监听控制台错误
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    // 等待页面完全加载
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);
    
    // 检查页面内容不包含数据库错误
    const pageContent = await page.content();
    expect(pageContent).not.toContain('duplicate key value');
    expect(pageContent).not.toContain('SQLSTATE 23505');
    expect(pageContent).not.toContain('users_pkey');
    
    console.log('✓ No database errors detected');
  });
});
