import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

/**
 * 登录为管理员
 */
async function loginAsAdmin(page) {
  console.log('开始管理员登录...');
  
  await page.goto(`${BASE_URL}`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  await page.waitForTimeout(1000);
  
  // 检查是否已经登录（通过检查是否有登录表单）
  const currentUrl = page.url();
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  
  if (!hasLoginForm) {
    console.log('✓ 已自动登录，跳过登录步骤');
    return;
  }
  
  // 填写登录表单
  console.log('填写管理员登录凭据...');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  
  // 点击登录按钮
  await page.click('button[type="submit"]');
  console.log('✓ 已提交登录表单');
  
  // 等待登录完成（检查URL变化或登录表单消失）
  await page.waitForTimeout(2000);
  
  // 验证登录成功
  const afterLoginUrl = page.url();
  const stillHasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  
  if (!stillHasLoginForm && !afterLoginUrl.includes('/login')) {
    console.log('✓ 登录成功');
  } else {
    console.log('⚠️ 登录验证: 可能仍在登录页面');
    // 截图调试
    await page.screenshot({ path: 'test-results/login-status.png', fullPage: true });
  }
}

test.describe('监控页面认证测试', () => {
  
  test('管理员登录后应该能访问监控页面并显示iframe', async ({ page }) => {
    console.log('\n=== 测试: 管理员访问监控页面 ===');
    
    // 第一步: 登录
    await loginAsAdmin(page);
    
    // 第二步: 访问监控页面
    console.log('访问监控页面...');
    await page.goto(`${BASE_URL}/monitoring`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    // 等待React应用加载
    await page.waitForTimeout(2000);
    
    // 截图查看页面状态
    await page.screenshot({ 
      path: 'test-results/monitoring-with-admin.png', 
      fullPage: true 
    });
    
    // 验证URL
    const currentUrl = page.url();
    console.log(`当前URL: ${currentUrl}`);
    expect(currentUrl).toContain('/monitoring');
    
    // 检查是否还在登录页面（如果是说明没有正确登录或权限问题）
    const bodyText = await page.locator('body').textContent();
    
    // 不应该显示登录表单的特定文本
    expect(bodyText).not.toContain('智能基础设施管理平台');
    
    // 应该显示监控页面特定内容
    expect(bodyText).toContain('监控仪表板');
    
    // 检查iframe是否存在
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    const iframeCount = await iframeElement.count();
    console.log(`找到 ${iframeCount} 个iframe`);
    
    // 应该找到iframe
    expect(iframeCount).toBeGreaterThan(0);
    
    // 等待iframe可见
    await expect(iframeElement).toBeVisible({ timeout: 15000 });
    
    const iframeSrc = await iframeElement.getAttribute('src');
    console.log(`✓ iframe src: ${iframeSrc}`);
    
    // iframe src应该指向Nightingale服务
    expect(iframeSrc).toContain('/nightingale/');
    
    console.log('✓ 测试通过: iframe显示正常');
  });

  test('刷新页面后应该保持登录状态并显示iframe', async ({ page }) => {
    console.log('\n=== 测试: 刷新后保持状态 ===');
    
    // 登录
    await loginAsAdmin(page);
    
    // 访问监控页面
    await page.goto(`${BASE_URL}/monitoring`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    await page.waitForTimeout(2000);
    
    // 验证iframe存在
    const iframeElement = page.locator('iframe[title="Nightingale Monitoring"]');
    await expect(iframeElement).toBeVisible({ timeout: 15000 });
    console.log('✓ 初次加载成功');
    
    // 刷新页面
    console.log('刷新页面...');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/monitoring-after-refresh.png', 
      fullPage: true 
    });
    
    // 验证仍然在监控页面
    expect(page.url()).toContain('/monitoring');
    
    // 验证没有跳转到登录页
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).not.toContain('登录');
    
    // 验证iframe仍然存在
    await expect(iframeElement).toBeVisible({ timeout: 15000 });
    
    console.log('✓ 测试通过: 刷新后状态保持正常');
  });

  test('不应显示数据库重复键错误', async ({ page }) => {
    console.log('\n=== 测试: 无数据库错误 ===');
    
    // 监听控制台错误
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    // 登录
    await loginAsAdmin(page);
    
    // 访问监控页面
    await page.goto(`${BASE_URL}/monitoring`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    await page.waitForTimeout(3000);
    
    // 检查页面内容不包含数据库错误
    const pageContent = await page.content();
    expect(pageContent).not.toContain('duplicate key value');
    expect(pageContent).not.toContain('SQLSTATE 23505');
    expect(pageContent).not.toContain('users_pkey');
    
    console.log('✓ 测试通过: 无数据库错误');
  });
});
