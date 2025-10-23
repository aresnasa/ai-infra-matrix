/**
 * MinIO集成完整测试 - 包含配置和访问
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('MinIO完整集成测试', async ({ page }) => {
  console.log('\n=== MinIO完整集成测试 ===\n');
  
  let token = '';
  
  // 监听失败的请求
  const failedRequests = [];
  page.on('response', async res => {
    if (res.status() >= 400) {
      failedRequests.push({ 
        url: res.url(), 
        status: res.status(),
        statusText: res.statusText() 
      });
    }
  });
  
  // 1. 登录获取token
  console.log('1. 登录系统...');
  await page.goto(`${BASE_URL}/login`);
  
  // 监听登录响应获取token
  page.on('response', async res => {
    if (res.url().includes('/api/auth/login') && res.status() === 200) {
      try {
        const data = await res.json();
        token = data.token || '';
      } catch (e) {
        // ignore
      }
    }
  });
  
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|projects)/, { timeout: 10000 });
  console.log('   ✓ 登录成功\n');
  
  // 2. 检查MinIO配置是否存在
  console.log('2. 检查MinIO配置...');
  await page.goto(`${BASE_URL}/object-storage`);
  await page.waitForTimeout(3000);
  
  const hasNoConfig = await page.locator('text=尚未配置对象存储').isVisible().catch(() => false);
  
  if (hasNoConfig) {
    console.log('   ⚠️  未找到MinIO配置，创建新配置...\n');
    
    // 3. 创建MinIO配置
    console.log('3. 创建MinIO配置...');
    await page.goto(`${BASE_URL}/admin/object-storage`);
    await page.waitForTimeout(2000);
    
    // 点击添加存储按钮
    const addBtn = page.locator('button:has-text("添加")').first();
    if (await addBtn.isVisible()) {
      await addBtn.click();
      await page.waitForTimeout(1000);
      
      // 填写MinIO配置
      await page.fill('input[placeholder*="名称"]', 'MinIO主存储');
      
      // 选择MinIO类型
      await page.click('.ant-select-selector:has-text("选择存储类型")');
      await page.click('text=MinIO');
      
      // 填写Endpoint
      await page.fill('input[placeholder*="Endpoint"]', 'minio:9000');
      
      // 填写Access Key
      await page.fill('input[placeholder*="Access Key"]', 'minioadmin');
      
      // 填写Secret Key
      await page.fill('input[placeholder*="Secret Key"]', 'minioadmin');
      
      // 填写Web URL
      await page.fill('input[placeholder*="Web"]', '/minio-console/');
      
      // 设为默认
      const defaultCheckbox = page.locator('input[type="checkbox"]:near(:text("设为默认"))');
      if (await defaultCheckbox.isVisible()) {
        await defaultCheckbox.check();
      }
      
      // 保存配置
      await page.click('button:has-text("保存")');
      await page.waitForTimeout(2000);
      
      console.log('   ✓ MinIO配置已创建\n');
    } else {
      console.log('   ⚠️  无法找到添加按钮，尝试直接访问\n');
    }
  } else {
    console.log('   ✓ 找到现有MinIO配置\n');
  }
  
  // 4. 返回对象存储页面
  console.log('4. 访问对象存储页面...');
  await page.goto(`${BASE_URL}/object-storage`);
  await page.waitForTimeout(3000);
  
  // 截图对象存储列表
  await page.screenshot({ 
    path: 'test-screenshots/minio-storage-list.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: minio-storage-list.png\n');
  
  // 5. 检查MinIO配置卡片
  console.log('5. 检查MinIO配置状态...');
  
  const minioCard = page.locator('.ant-card:has-text("MinIO")').first();
  const hasMinioCard = await minioCard.isVisible().catch(() => false);
  
  if (!hasMinioCard) {
    console.log('   ❌ 未找到MinIO配置卡片\n');
    return;
  }
  
  console.log('   ✓ 找到MinIO配置卡片');
  
  // 检查连接状态
  const connectedTag = page.locator('.ant-tag:has-text("已连接")');
  const isConnected = await connectedTag.isVisible().catch(() => false);
  console.log(`   连接状态: ${isConnected ? '✓ 已连接' : '❌ 未连接'}\n`);
  
  // 6. 点击访问按钮
  console.log('6. 访问MinIO控制台...');
  
  const accessBtn = page.locator('button:has-text("访问")').first();
  const isBtnEnabled = await accessBtn.isEnabled().catch(() => false);
  
  if (!isBtnEnabled) {
    console.log('   ❌ 访问按钮被禁用\n');
    return;
  }
  
  await accessBtn.click();
  await page.waitForTimeout(2000);
  
  // 等待跳转
  await page.waitForURL(/\/object-storage\/minio\/\d+/, { timeout: 10000 });
  console.log('   ✓ 跳转到MinIO控制台页面');
  console.log(`   当前URL: ${page.url()}\n`);
  
  // 7. 检查MinIO控制台iframe
  console.log('7. 检查MinIO控制台iframe...');
  await page.waitForTimeout(5000); // 等待iframe加载
  
  const iframes = await page.locator('iframe').count();
  console.log(`   找到 ${iframes} 个 iframe`);
  
  if (iframes > 0) {
    const iframe = page.locator('iframe').first();
    const src = await iframe.getAttribute('src');
    console.log(`   iframe src: ${src}`);
    
    // 验证iframe URL包含minio-console
    expect(src).toContain('minio-console');
    console.log('   ✓ iframe URL正确\n');
    
    // 检查iframe可见性
    const isVisible = await iframe.isVisible();
    console.log(`   iframe 可见: ${isVisible ? '✓ 是' : '❌ 否'}`);
    
    if (isVisible) {
      const bbox = await iframe.boundingBox();
      console.log(`   iframe 尺寸: ${Math.round(bbox.width)}x${Math.round(bbox.height)}`);
      
      // 验证iframe尺寸合理
      expect(bbox.width).toBeGreaterThan(500);
      expect(bbox.height).toBeGreaterThan(300);
      console.log('   ✓ iframe尺寸正常\n');
    }
  } else {
    console.log('   ❌ 未找到iframe\n');
    
    // 检查是否有错误消息
    const errorMsg = await page.locator('text=无法连接MinIO服务').isVisible().catch(() => false);
    if (errorMsg) {
      console.log('   ❌ 显示连接错误\n');
    }
  }
  
  // 8. 截图最终状态
  console.log('8. 保存最终截图...');
  await page.screenshot({ 
    path: 'test-screenshots/minio-console-final.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: minio-console-final.png\n');
  
  // 9. 测试MinIO Console直接访问
  console.log('9. 测试MinIO Console直接访问...');
  await page.goto(`${BASE_URL}/minio-console/`);
  await page.waitForTimeout(3000);
  
  // 检查是否成功加载MinIO登录页面
  const pageContent = await page.content();
  const hasMinioContent = pageContent.includes('MinIO') || 
                          pageContent.includes('minio') ||
                          pageContent.includes('Console');
  
  console.log(`   MinIO Console页面加载: ${hasMinioContent ? '✓ 成功' : '❌ 失败'}`);
  
  await page.screenshot({ 
    path: 'test-screenshots/minio-console-direct.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存: minio-console-direct.png\n');
  
  // 10. 报告失败的请求
  if (failedRequests.length > 0) {
    console.log('失败的请求:');
    failedRequests.slice(0, 10).forEach((req, idx) => {
      console.log(`   ${idx + 1}. [${req.status}] ${req.url}`);
    });
    if (failedRequests.length > 10) {
      console.log(`   ... 还有 ${failedRequests.length - 10} 个失败请求`);
    }
    console.log('');
  }
  
  console.log('=== ✅ 测试完成 ===\n');
});
