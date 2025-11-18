/**
 * MinIO Console直接访问测试
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('MinIO Console直接访问测试', async ({ page }) => {
  console.log('\n=== MinIO Console直接访问测试 ===\n');
  
  // 1. 直接访问MinIO Console
  console.log('1. 访问 /minio-console/ ...');
  await page.goto(`${BASE_URL}/minio-console/`);
  await page.waitForTimeout(5000);
  
  console.log(`   当前URL: ${page.url()}`);
  
  // 2. 检查页面标题
  const title = await page.title();
  console.log(`   页面标题: ${title}`);
  
  // 3. 检查页面内容
  const content = await page.content();
  const hasMinioText = content.toLowerCase().includes('minio');
  const hasConsoleText = content.toLowerCase().includes('console');
  const hasLoginText = content.toLowerCase().includes('login');
  
  console.log(`   包含"MinIO"文本: ${hasMinioText ? '✓ 是' : '❌ 否'}`);
  console.log(`   包含"Console"文本: ${hasConsoleText ? '✓ 是' : '❌ 否'}`);
  console.log(`   包含"Login"文本: ${hasLoginText ? '✓ 是' : '❌ 否'}`);
  
  // 4. 查找登录表单元素
  console.log('\n2. 检查登录表单...');
  
  const usernameInput = page.locator('input[type="text"], input[name*="user"], input[placeholder*="user"]').first();
  const passwordInput = page.locator('input[type="password"]').first();
  const loginButton = page.locator('button:has-text(/login|登录/i)').first();
  
  const hasUsername = await usernameInput.isVisible().catch(() => false);
  const hasPassword = await passwordInput.isVisible().catch(() => false);
  const hasLoginBtn = await loginButton.isVisible().catch(() => false);
  
  console.log(`   用户名输入框: ${hasUsername ? '✓ 可见' : '❌ 不可见'}`);
  console.log(`   密码输入框: ${hasPassword ? '✓ 可见' : '❌ 不可见'}`);
  console.log(`   登录按钮: ${hasLoginBtn ? '✓ 可见' : '❌ 不可见'}`);
  
  // 5. 截图
  console.log('\n3. 保存截图...');
  await page.screenshot({ 
    path: 'test-screenshots/minio-console-login.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: minio-console-login.png');
  
  // 6. 尝试登录
  if (hasUsername && hasPassword && hasLoginBtn) {
    console.log('\n4. 尝试登录MinIO Console...');
    
    await usernameInput.fill('minioadmin');
    await passwordInput.fill('minioadmin');
    await loginButton.click();
    
    await page.waitForTimeout(3000);
    
    const afterLoginUrl = page.url();
    console.log(`   登录后URL: ${afterLoginUrl}`);
    
    // 检查是否登录成功（URL变化或页面内容变化）
    const isStillOnLogin = afterLoginUrl.includes('/login') || 
                           await page.locator('input[type="password"]').isVisible().catch(() => false);
    
    console.log(`   登录状态: ${isStillOnLogin ? '❌ 失败' : '✓ 成功'}`);
    
    if (!isStillOnLogin) {
      console.log('\n5. 检查MinIO控制台主界面...');
      
      await page.screenshot({ 
        path: 'test-screenshots/minio-console-dashboard.png',
        fullPage: true 
      });
      console.log('   ✓ 截图已保存: minio-console-dashboard.png');
      
      // 检查是否有MinIO控制台特有的元素
      const hasBuckets = await page.locator('text=/bucket|存储桶/i').isVisible().catch(() => false);
      const hasObjects = await page.locator('text=/object|对象/i').isVisible().catch(() => false);
      
      console.log(`   显示存储桶: ${hasBuckets ? '✓ 是' : '❌ 否'}`);
      console.log(`   显示对象: ${hasObjects ? '✓ 是' : '❌ 否'}`);
    }
  }
  
  console.log('\n=== ✅ 测试完成 ===\n');
});
