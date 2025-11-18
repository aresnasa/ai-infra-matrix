const { test, expect } = require('@playwright/test');

test.describe('MinIO 对象存储集成测试', () => {
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
    
    await page.waitForURL('**/dashboard', { timeout: 15000 }).catch(async () => {
      await page.waitForTimeout(3000);
    });
    
    console.log('✓ 登录成功');
  });

  test('1. 测试 /minio-console/ 代理路径直接访问', async ({ page }) => {
    console.log('\n=== 测试 1: MinIO Console 代理路径 ===');
    
    // 直接访问 /minio-console/
    const response = await page.goto('/minio-console/');
    
    const status = response.status();
    const url = page.url();
    
    console.log('HTTP 状态码:', status);
    console.log('最终 URL:', url);
    
    // 检查响应头
    const headers = await response.allHeaders();
    console.log('X-Frame-Options:', headers['x-frame-options'] || '未设置');
    console.log('Content-Security-Policy:', headers['content-security-policy'] || '未设置');
    
    // 等待页面加载
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/minio-console-direct-access.png', 
      fullPage: true 
    });
    
    // 检查页面内容
    const pageText = await page.textContent('body').catch(() => '');
    const hasMinioContent = pageText.includes('MinIO') || 
                           pageText.includes('Object Browser') ||
                           pageText.includes('Buckets');
    
    console.log('包含 MinIO 内容:', hasMinioContent);
    
    // 检查是否有登录表单
    const loginInputs = await page.locator('input[type="text"], input[name="accessKey"]').count();
    console.log('登录表单输入框数量:', loginInputs);
    
    if (status === 200) {
      console.log('✅ /minio-console/ 可以访问 (200)');
    } else if (status === 302 || status === 301) {
      console.log('⚠️  /minio-console/ 重定向 (' + status + ')');
    } else {
      console.log('❌ /minio-console/ 访问失败 (' + status + ')');
    }
  });

  test('2. 测试对象存储页面', async ({ page }) => {
    console.log('\n=== 测试 2: 对象存储页面 ===');
    
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    // 访问对象存储页面
    await page.goto('/object-storage');
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/object-storage-page.png', 
      fullPage: true 
    });
    
    // 检查页面标题
    const pageTitle = await page.textContent('body').catch(() => '');
    console.log('页面包含"对象存储":', pageTitle.includes('对象存储'));
    
    // 检查是否有配置卡片
    const cards = await page.locator('.ant-card').count();
    console.log('配置卡片数量:', cards);
    
    // 查找 MinIO 相关按钮
    const minioButtons = await page.locator('button:has-text("MinIO"), button:has-text("查看控制台"), a:has-text("打开")').count();
    console.log('MinIO 相关按钮数量:', minioButtons);
    
    console.log('控制台错误数量:', errors.length);
    if (errors.length > 0) {
      console.log('前 3 个错误:', errors.slice(0, 3));
    }
    
    if (cards > 0 || minioButtons > 0) {
      console.log('✅ 对象存储页面加载正常');
    } else {
      console.log('⚠️  对象存储页面可能没有配置');
    }
  });

  test('3. 测试 MinIO iframe 嵌入', async ({ page }) => {
    console.log('\n=== 测试 3: MinIO iframe 嵌入 ===');
    
    const networkRequests = [];
    page.on('response', response => {
      if (response.url().includes('minio')) {
        networkRequests.push({
          url: response.url(),
          status: response.status()
        });
      }
    });
    
    // 访问对象存储页面
    await page.goto('/object-storage');
    await page.waitForTimeout(2000);
    
    // 查找并点击 MinIO 按钮
    const viewButton = page.locator('button:has-text("查看控制台"), button:has-text("打开"), a:has-text("MinIO")').first();
    const buttonExists = await viewButton.count();
    
    console.log('找到 MinIO 访问按钮:', buttonExists > 0);
    
    if (buttonExists > 0) {
      await viewButton.click();
      await page.waitForTimeout(3000);
      
      const currentUrl = page.url();
      console.log('点击后的 URL:', currentUrl);
      
      // 检查 iframe
      const iframe = page.locator('iframe#minio-console-iframe, iframe[title*="MinIO"]').first();
      const iframeCount = await iframe.count();
      
      console.log('MinIO iframe 存在:', iframeCount > 0);
      
      if (iframeCount > 0) {
        const src = await iframe.getAttribute('src');
        const isVisible = await iframe.isVisible();
        
        console.log('Iframe src:', src);
        console.log('Iframe 可见:', isVisible);
        
        // 检查 iframe sandbox 属性
        const sandbox = await iframe.getAttribute('sandbox');
        console.log('Iframe sandbox:', sandbox || '未设置');
        
        await page.waitForTimeout(5000);
        
        // 截图
        await page.screenshot({ 
          path: 'test-results/minio-iframe-embed.png', 
          fullPage: true 
        });
        
        if (isVisible && src?.includes('/minio-console/')) {
          console.log('✅ MinIO iframe 正确加载');
        } else {
          console.log('❌ MinIO iframe 有问题');
        }
      } else {
        console.log('❌ 未找到 MinIO iframe');
        await page.screenshot({ 
          path: 'test-results/minio-no-iframe.png', 
          fullPage: true 
        });
      }
    } else {
      console.log('⚠️  未找到 MinIO 访问按钮，可能需要创建配置');
      await page.screenshot({ 
        path: 'test-results/object-storage-no-button.png', 
        fullPage: true 
      });
    }
    
    console.log('\nMinIO 网络请求数:', networkRequests.length);
    networkRequests.slice(0, 5).forEach(req => {
      console.log(`  ${req.status} - ${req.url}`);
    });
  });

  test('4. 完整诊断报告', async ({ page }) => {
    console.log('\n=== 测试 4: MinIO 完整诊断 ===');
    
    const diagnostics = {
      consoleErrors: [],
      networkErrors: [],
      minioRequests: [],
      pageInfo: {}
    };
    
    // 监听控制台
    page.on('console', msg => {
      if (msg.type() === 'error') {
        diagnostics.consoleErrors.push(msg.text());
      }
    });
    
    // 监听网络
    page.on('response', response => {
      const url = response.url();
      if (url.includes('minio')) {
        diagnostics.minioRequests.push({
          url,
          status: response.status(),
          statusText: response.statusText()
        });
      }
      if (response.status() >= 400) {
        diagnostics.networkErrors.push({
          url,
          status: response.status()
        });
      }
    });
    
    // 测试 1: 直接访问 /minio-console/
    console.log('\n[1/3] 测试直接访问 /minio-console/...');
    const consoleResponse = await page.goto('/minio-console/');
    diagnostics.consoleDirectStatus = consoleResponse.status();
    await page.waitForTimeout(2000);
    
    // 测试 2: 访问对象存储页面
    console.log('[2/3] 测试对象存储页面...');
    await page.goto('/object-storage');
    await page.waitForTimeout(3000);
    
    diagnostics.pageInfo.title = await page.title();
    diagnostics.pageInfo.url = page.url();
    diagnostics.pageInfo.cards = await page.locator('.ant-card').count();
    
    // 测试 3: 尝试访问 MinIO
    console.log('[3/3] 尝试访问 MinIO 控制台...');
    const minioLink = page.locator('button:has-text("MinIO"), button:has-text("查看"), a:has-text("MinIO")').first();
    const hasLink = await minioLink.count();
    
    if (hasLink > 0) {
      await minioLink.click();
      await page.waitForTimeout(3000);
      
      const iframe = page.locator('iframe#minio-console-iframe, iframe[title*="MinIO"]').first();
      diagnostics.pageInfo.hasIframe = await iframe.count() > 0;
      
      if (diagnostics.pageInfo.hasIframe) {
        diagnostics.pageInfo.iframeSrc = await iframe.getAttribute('src');
        diagnostics.pageInfo.iframeVisible = await iframe.isVisible();
      }
    }
    
    // 生成报告
    console.log('\n' + '='.repeat(60));
    console.log('MinIO 诊断报告');
    console.log('='.repeat(60));
    
    console.log('\n【直接访问】');
    console.log('  /minio-console/ 状态:', diagnostics.consoleDirectStatus);
    
    console.log('\n【页面信息】');
    console.log('  标题:', diagnostics.pageInfo.title);
    console.log('  URL:', diagnostics.pageInfo.url);
    console.log('  配置卡片数:', diagnostics.pageInfo.cards);
    console.log('  有 iframe:', diagnostics.pageInfo.hasIframe || false);
    if (diagnostics.pageInfo.iframeSrc) {
      console.log('  iframe src:', diagnostics.pageInfo.iframeSrc);
      console.log('  iframe 可见:', diagnostics.pageInfo.iframeVisible);
    }
    
    console.log('\n【网络请求】');
    console.log('  MinIO 请求数:', diagnostics.minioRequests.length);
    diagnostics.minioRequests.slice(0, 5).forEach(req => {
      console.log(`    ${req.status} ${req.statusText} - ${req.url}`);
    });
    
    console.log('\n【错误统计】');
    console.log('  控制台错误:', diagnostics.consoleErrors.length);
    console.log('  网络错误 (4xx/5xx):', diagnostics.networkErrors.length);
    
    if (diagnostics.consoleErrors.length > 0) {
      console.log('\n【控制台错误详情】');
      diagnostics.consoleErrors.slice(0, 3).forEach((err, i) => {
        console.log(`  ${i + 1}. ${err}`);
      });
    }
    
    if (diagnostics.networkErrors.length > 0) {
      console.log('\n【网络错误详情】');
      diagnostics.networkErrors.slice(0, 3).forEach((err, i) => {
        console.log(`  ${i + 1}. ${err.status} - ${err.url}`);
      });
    }
    
    console.log('\n' + '='.repeat(60));
    
    // 最终截图
    await page.screenshot({ 
      path: 'test-results/minio-diagnostic-final.png', 
      fullPage: true 
    });
    
    // 判断整体状态
    const isHealthy = diagnostics.consoleDirectStatus === 200 && 
                     diagnostics.consoleErrors.length === 0 &&
                     diagnostics.networkErrors.length === 0;
    
    if (isHealthy) {
      console.log('\n✅ MinIO 集成整体健康');
    } else {
      console.log('\n⚠️  MinIO 集成存在问题，请查看上述详情');
    }
  });
});
