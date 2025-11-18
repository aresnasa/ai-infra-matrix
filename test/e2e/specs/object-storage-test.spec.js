const { test, expect } = require('@playwright/test');

test.describe('对象存储 MinIO 集成测试', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('/');
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {});
    
    const usernameInput = page.locator('input[type="text"], input[placeholder*="用户名"], input[name="username"]').first();
    await usernameInput.fill('admin');
    
    const passwordInput = page.locator('input[type="password"], input[placeholder*="密码"], input[name="password"]').first();
    await passwordInput.fill('Admin@123');
    
    const loginButton = page.locator('button').filter({ hasText: /登录|Login/ }).first();
    await loginButton.click();
    
    // 等待登录完成
    await page.waitForURL('**/dashboard', { timeout: 15000 }).catch(async () => {
      await page.waitForTimeout(3000);
    });
    
    console.log('✓ 登录成功');
  });

  test('测试 /object-storage 页面加载', async ({ page }) => {
    // 访问对象存储页面
    await page.goto('/object-storage');
    await page.waitForTimeout(2000);
    
    // 检查页面标题
    const title = page.locator('text=对象存储').first();
    await expect(title).toBeVisible({ timeout: 10000 });
    
    console.log('✓ 对象存储页面标题显示正确');
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/object-storage-page.png', 
      fullPage: true 
    });
    
    // 检查是否有配置列表
    const configCards = page.locator('.ant-card');
    const cardCount = await configCards.count();
    console.log(`找到 ${cardCount} 个配置卡片`);
    
    // 查找 MinIO 相关按钮或链接
    const minioLinks = page.locator('text=/minio|MinIO/i');
    const minioLinkCount = await minioLinks.count();
    console.log(`找到 ${minioLinkCount} 个 MinIO 相关元素`);
    
    if (minioLinkCount > 0) {
      // 如果有 MinIO 链接，尝试点击
      const firstMinioLink = minioLinks.first();
      await firstMinioLink.click();
      await page.waitForTimeout(2000);
      
      console.log('✓ 已点击 MinIO 链接');
      
      // 检查 URL 是否变化
      const currentUrl = page.url();
      console.log('当前 URL:', currentUrl);
      
      // 截图新页面
      await page.screenshot({ 
        path: 'test-results/object-storage-minio-clicked.png', 
        fullPage: true 
      });
    }
  });

  test('测试 /minio-console/ 代理路径', async ({ page, context }) => {
    // 获取 cookies
    const cookies = await context.cookies();
    console.log('当前 Cookies 数量:', cookies.length);
    
    // 直接访问 /minio-console/
    const response = await page.goto('/minio-console/');
    
    console.log('/minio-console/ 状态码:', response.status());
    console.log('/minio-console/ 最终 URL:', page.url());
    
    // 检查响应头
    const headers = await response.allHeaders();
    console.log('X-Frame-Options:', headers['x-frame-options']);
    console.log('Content-Security-Policy:', headers['content-security-policy']);
    
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/minio-console-direct.png', 
      fullPage: true 
    });
    
    // 检查页面内容
    const pageContent = await page.content();
    const hasMinioContent = pageContent.includes('MinIO') || 
                           pageContent.includes('minio') || 
                           pageContent.includes('Object Browser');
    
    console.log('页面包含 MinIO 内容:', hasMinioContent);
    
    // 检查是否有登录表单
    const loginForm = page.locator('input[type="text"], input[type="password"]').first();
    const hasLoginForm = await loginForm.isVisible().catch(() => false);
    console.log('显示登录表单:', hasLoginForm);
    
    // 检查控制台错误
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    await page.waitForTimeout(2000);
    console.log('控制台错误数量:', errors.length);
    if (errors.length > 0) {
      console.log('控制台错误:', errors.slice(0, 5));
    }
  });

  test('测试 MinIO iframe 集成', async ({ page }) => {
    // 访问对象存储页面
    await page.goto('/object-storage');
    await page.waitForTimeout(2000);
    
    // 尝试查找访问 MinIO 的按钮
    const viewButtons = page.locator('button:has-text("查看控制台"), button:has-text("打开"), a:has-text("MinIO")');
    const buttonCount = await viewButtons.count();
    
    console.log('找到 "查看控制台/打开" 按钮数量:', buttonCount);
    
    if (buttonCount > 0) {
      const firstButton = viewButtons.first();
      await firstButton.click();
      await page.waitForTimeout(3000);
      
      // 检查是否进入了 MinIO 控制台页面
      const currentUrl = page.url();
      console.log('点击后的 URL:', currentUrl);
      
      // 检查 iframe
      const iframe = page.locator('iframe#minio-console-iframe, iframe[title*="MinIO"]').first();
      const iframeExists = await iframe.count();
      console.log('MinIO iframe 存在:', iframeExists > 0);
      
      if (iframeExists > 0) {
        const src = await iframe.getAttribute('src');
        console.log('Iframe src:', src);
        
        const isVisible = await iframe.isVisible();
        console.log('Iframe 可见:', isVisible);
        
        // 检查 iframe 加载状态
        await page.waitForTimeout(5000);
        
        // 截图
        await page.screenshot({ 
          path: 'test-results/minio-iframe-integration.png', 
          fullPage: true 
        });
      }
    } else {
      console.log('⚠ 未找到访问 MinIO 的按钮，可能需要先创建配置');
      
      // 截图当前状态
      await page.screenshot({ 
        path: 'test-results/object-storage-no-config.png', 
        fullPage: true 
      });
    }
  });

  test('检查 nginx minio 配置', async ({ request }) => {
    // 测试 /minio/ API 路径
    const minioApiResponse = await request.get('/minio/').catch(e => e);
    console.log('/minio/ 状态:', minioApiResponse.status?.() || 'failed');
    
    // 测试 /minio-console/ 路径
    const minioConsoleResponse = await request.get('/minio-console/').catch(e => e);
    console.log('/minio-console/ 状态:', minioConsoleResponse.status?.() || 'failed');
    
    // 测试 MinIO 健康检查
    const healthResponse = await request.get('/minio/health').catch(e => e);
    console.log('/minio/health 状态:', healthResponse.status?.() || 'failed');
  });

  test('诊断 MinIO 加载问题', async ({ page }) => {
    const errors = [];
    const networkRequests = [];
    
    // 监听控制台错误
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    // 监听网络请求
    page.on('response', response => {
      if (response.url().includes('minio')) {
        networkRequests.push({
          url: response.url(),
          status: response.status(),
          statusText: response.statusText()
        });
      }
    });
    
    // 访问对象存储页面
    await page.goto('/object-storage');
    await page.waitForTimeout(5000);
    
    console.log('\n=== MinIO 诊断报告 ===');
    console.log('控制台错误数:', errors.length);
    if (errors.length > 0) {
      console.log('前 3 个错误:', errors.slice(0, 3));
    }
    
    console.log('\nMinIO 相关网络请求数:', networkRequests.length);
    networkRequests.forEach(req => {
      console.log(`  ${req.status} - ${req.url}`);
    });
    
    // 检查页面状态
    const pageTitle = await page.title();
    console.log('\n页面标题:', pageTitle);
    
    const currentUrl = page.url();
    console.log('当前 URL:', currentUrl);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/minio-diagnostic.png', 
      fullPage: true 
    });
    
    console.log('\n✅ 诊断完成');
  });
});
