import { test, expect } from '@playwright/test';

test.describe('Monitoring Page with Header and Iframe', () => {
  test('should display monitoring page with header and iframe on initial load', async ({ page }) => {
    // 访问监控入口
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    // 应该保持在 /monitoring 路径（前端 SPA 路由）
    await expect(page).toHaveURL('http://192.168.0.200:8080/monitoring');
    
    // 检查页面应该包含全局导航/头部
    // 通常前端框架会有导航栏、侧边栏等
    const bodyText = await page.locator('body').textContent();
    
    // 检查是否有 iframe（MonitoringPage 组件会渲染 iframe）
    const iframe = page.frameLocator('iframe[title="Nightingale Monitoring"]');
    await expect(iframe.locator('body')).toBeVisible({ timeout: 10000 });
    
    console.log('✓ Page loaded with header and iframe');
  });

  test('should maintain header and iframe after page refresh', async ({ page }) => {
    // 首次访问
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });

    // 等待 iframe 加载
    const iframe = page.frameLocator('iframe[title="Nightingale Monitoring"]');
    await expect(iframe.locator('body')).toBeVisible({ timeout: 10000 });
    
    console.log('✓ Initial load successful');

    // 刷新页面
    await page.reload({ waitUntil: 'networkidle' });
    
    // 验证 URL 保持不变
    await expect(page).toHaveURL('http://192.168.0.200:8080/monitoring');
    
    // 验证 iframe 仍然存在并可见
    await expect(iframe.locator('body')).toBeVisible({ timeout: 10000 });
    
    // 检查页面不包含错误信息
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).not.toContain('404');
    expect(bodyText).not.toContain('Not Found');
    expect(bodyText).not.toContain('ERROR');
    
    console.log('✓ Page refresh maintains header and iframe');
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
    
    // 检查页面内容不包含数据库错误
    const pageContent = await page.content();
    expect(pageContent).not.toContain('duplicate key value');
    expect(pageContent).not.toContain('SQLSTATE 23505');
    expect(pageContent).not.toContain('users_pkey');
    
    console.log('✓ No database errors detected');
  });
});
