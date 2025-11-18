/**
 * MinIO测试 - 先通过API配置，再测试访问
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('MinIO集成测试（API配置+Web访问）', async ({ page, request }) => {
  console.log('\n=== MinIO集成测试 ===\n');
  
  // 1. 登录获取token
  console.log('1. 登录获取token...');
  const loginRes = await request.post(`${BASE_URL}/api/auth/login`, {
    data: {
      username: 'admin',
      password: 'admin123'
    }
  });
  
  const loginData = await loginRes.json();
  const token = loginData.token;
  console.log(`   ✓ Token获取成功: ${token.substring(0, 20)}...\n`);
  
  // 2. 检查现有配置
  console.log('2. 检查现有MinIO配置...');
  const configsRes = await request.get(`${BASE_URL}/api/object-storage/configs`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  
  let minioConfigId = null;
  if (configsRes.ok()) {
    const configsData = await configsRes.json();
    const configs = configsData.data || [];
    const minioConfig = configs.find(c => c.type === 'minio');
    
    if (minioConfig) {
      minioConfigId = minioConfig.id;
      console.log(`   ✓ 找到现有MinIO配置 (ID: ${minioConfigId})\n`);
    } else {
      console.log('   未找到MinIO配置，创建新配置...\n');
      
      // 3. 创建MinIO配置
      console.log('3. 创建MinIO配置...');
      const createRes = await request.post(`${BASE_URL}/api/object-storage/configs`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        data: {
          name: 'MinIO主存储',
          type: 'minio',
          endpoint: 'minio:9000',
          access_key: 'minioadmin',
          secret_key: 'minioadmin',
          web_url: '/minio-console/',
          use_ssl: false,
          is_active: true
        }
      });
      
      if (createRes.ok()) {
        const createData = await createRes.json();
        minioConfigId = createData.data?.id;
        console.log(`   ✓ MinIO配置创建成功 (ID: ${minioConfigId})\n`);
      } else {
        const error = await createRes.text();
        console.log(`   ❌ 创建失败: ${error}\n`);
      }
    }
  }
  
  // 4. Web登录
  console.log('4. Web登录...');
  await page.goto(`${BASE_URL}/login`);
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|projects)/, { timeout: 10000 });
  console.log('   ✓ 登录成功\n');
  
  // 5. 访问对象存储页面
  console.log('5. 访问对象存储页面...');
  await page.goto(`${BASE_URL}/object-storage`);
  await page.waitForTimeout(3000);
  
  // 检查是否有配置
  const hasNoConfig = await page.locator('text=尚未配置对象存储').isVisible().catch(() => false);
  
  if (hasNoConfig) {
    console.log('   ❌ 仍然显示"尚未配置"，可能是API配置未生效\n');
    await page.screenshot({ 
      path: 'test-screenshots/minio-no-config.png',
      fullPage: true 
    });
    return;
  }
  
  console.log('   ✓ 页面加载完成');
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/minio-list.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存\n');
  
  // 6. 检查MinIO卡片
  console.log('6. 检查MinIO配置卡片...');
  const minioText = page.locator('text=/MinIO|minio/i').first();
  const hasMinioCard = await minioText.isVisible().catch(() => false);
  
  if (!hasMinioCard) {
    console.log('   ❌ 未找到MinIO相关内容\n');
    return;
  }
  
  console.log('   ✓ 找到MinIO配置');
  
  // 检查连接状态
  const connectedTag = page.locator('.ant-tag:has-text("已连接")');
  const isConnected = await connectedTag.isVisible().catch(() => false);
  console.log(`   连接状态: ${isConnected ? '已连接' : '未连接'}`);
  
  // 检查访问按钮
  const accessBtn = page.locator('button:has-text("访问")').first();
  const isBtnVisible = await accessBtn.isVisible().catch(() => false);
  const isBtnEnabled = isBtnVisible && await accessBtn.isEnabled().catch(() => false);
  
  console.log(`   访问按钮: ${isBtnVisible ? '可见' : '不可见'} / ${isBtnEnabled ? '可用' : '禁用'}\n`);
  
  if (!isBtnEnabled) {
    console.log('   ⚠️  访问按钮不可用，可能MinIO服务未连接\n');
    return;
  }
  
  // 7. 点击访问按钮
  console.log('7. 点击访问MinIO控制台...');
  await accessBtn.click();
  await page.waitForTimeout(2000);
  
  // 等待跳转
  const currentUrl = page.url();
  console.log(`   当前URL: ${currentUrl}`);
  
  if (!currentUrl.includes('/object-storage/minio/')) {
    console.log('   ❌ 未跳转到MinIO控制台页面\n');
    return;
  }
  
  console.log('   ✓ 跳转成功\n');
  
  // 8. 检查iframe
  console.log('8. 检查MinIO控制台iframe...');
  await page.waitForTimeout(5000);
  
  const iframes = await page.locator('iframe').count();
  console.log(`   找到 ${iframes} 个 iframe`);
  
  if (iframes > 0) {
    const iframe = page.locator('iframe').first();
    const src = await iframe.getAttribute('src');
    console.log(`   iframe src: ${src}`);
    
    // 验证src包含minio-console
    const hasMinio = src.includes('minio-console') || src.includes('minio');
    console.log(`   包含MinIO路径: ${hasMinio ? '✓ 是' : '❌ 否'}`);
    
    const isVisible = await iframe.isVisible();
    console.log(`   iframe可见: ${isVisible ? '✓ 是' : '❌ 否'}`);
    
    if (isVisible) {
      const bbox = await iframe.boundingBox();
      console.log(`   iframe尺寸: ${Math.round(bbox.width)}x${Math.round(bbox.height)}`);
      
      expect(bbox.width).toBeGreaterThan(300);
      expect(bbox.height).toBeGreaterThan(200);
      console.log('   ✓ 尺寸正常\n');
    }
  } else {
    console.log('   ❌ 未找到iframe\n');
  }
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/minio-console.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存\n');
  
  // 9. 直接访问MinIO Console
  console.log('9. 直接访问MinIO Console路径...');
  await page.goto(`${BASE_URL}/minio-console/`);
  await page.waitForTimeout(3000);
  
  const title = await page.title();
  const content = await page.content();
  const hasMinioContent = content.toLowerCase().includes('minio') || 
                          content.toLowerCase().includes('console');
  
  console.log(`   页面标题: ${title}`);
  console.log(`   包含MinIO内容: ${hasMinioContent ? '✓ 是' : '❌ 否'}`);
  
  await page.screenshot({ 
    path: 'test-screenshots/minio-console-direct.png',
    fullPage: true 
  });
  console.log('   ✓ 截图已保存\n');
  
  console.log('=== ✅ 测试完成 ===\n');
});
