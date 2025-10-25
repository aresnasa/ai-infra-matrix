import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';

/**
 * 登录为管理员或有权限的用户
 */
async function login(page: any) {
  console.log('开始登录...');
  
  await page.goto(`${BASE_URL}`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  await page.waitForTimeout(1000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  
  if (!hasLoginForm) {
    console.log('✓ 已自动登录');
    return;
  }
  
  console.log('填写登录凭据...');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  
  await page.waitForTimeout(2000);
  console.log('✓ 登录完成');
}

test.describe('SLURM 页面 SaltStack 任务调试', () => {
  
  test('检查最近 SaltStack 作业显示问题', async ({ page }) => {
    console.log('\n=== 调试 SLURM 页面 SaltStack 任务 ===');
    
    // 登录
    await login(page);
    
    // 访问 SLURM 页面
    console.log('\n访问 /slurm 页面...');
    await page.goto(`${BASE_URL}/slurm`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    // 等待页面加载
    await page.waitForTimeout(3000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/slurm-page-full.png', 
      fullPage: true 
    });
    
    console.log('\n=== 页面内容检查 ===');
    const bodyText = await page.locator('body').textContent();
    
    // 检查是否有 "最近 SaltStack 作业" 文本
    const hasSaltStackSection = bodyText?.includes('最近 SaltStack 作业') || 
                                bodyText?.includes('SaltStack');
    console.log(`找到 SaltStack 作业区域: ${hasSaltStackSection}`);
    
    // 查找包含 "test.ping" 的元素
    const testPingElements = page.locator('text=test.ping');
    const testPingCount = await testPingElements.count();
    console.log(`找到 ${testPingCount} 个 test.ping 元素`);
    
    // 查找显示 "undefined" 的元素
    const undefinedElements = page.locator('text=undefined');
    const undefinedCount = await undefinedElements.count();
    console.log(`找到 ${undefinedCount} 个 undefined 文本`);
    
    if (undefinedCount > 0) {
      console.log('\n❌ 页面显示 undefined');
      for (let i = 0; i < Math.min(undefinedCount, 3); i++) {
        const element = undefinedElements.nth(i);
        const parentText = await element.locator('..').textContent();
        console.log(`  ${i + 1}. 上下文: ${parentText?.substring(0, 200)}`);
      }
    }
    
    // 查找 SaltStack 相关的卡片或列表
    console.log('\n=== 查找 SaltStack 作业列表 ===');
    const cards = page.locator('.ant-card');
    const cardCount = await cards.count();
    console.log(`找到 ${cardCount} 个卡片组件`);
    
    // 查找包含 "SaltStack" 的卡片
    for (let i = 0; i < cardCount; i++) {
      const card = cards.nth(i);
      const cardText = await card.textContent();
      if (cardText?.includes('SaltStack') || cardText?.includes('作业')) {
        console.log(`\n找到 SaltStack 作业卡片 [${i}]:`);
        console.log(`内容预览: ${cardText?.substring(0, 300)}`);
        
        // 截图该卡片
        await card.screenshot({ 
          path: `test-results/saltstack-jobs-card-${i}.png` 
        });
      }
    }
    
    // 检查网络请求
    console.log('\n=== 监听 API 请求 ===');
    
    // 重新加载页面，监听 API 调用
    const apiRequests: string[] = [];
    page.on('request', request => {
      const url = request.url();
      if (url.includes('/api/')) {
        apiRequests.push(url);
        console.log(`API 请求: ${url}`);
      }
    });
    
    page.on('response', async response => {
      const url = response.url();
      if (url.includes('saltstack') || url.includes('jobs') || url.includes('tasks')) {
        console.log(`\nAPI 响应: ${url}`);
        console.log(`状态: ${response.status()}`);
        
        try {
          const contentType = response.headers()['content-type'];
          if (contentType?.includes('application/json')) {
            const data = await response.json();
            console.log(`响应数据:`, JSON.stringify(data, null, 2).substring(0, 500));
          }
        } catch (error) {
          console.log(`无法解析响应数据: ${error}`);
        }
      }
    });
    
    console.log('\n刷新页面以捕获 API 请求...');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    
    console.log(`\n捕获到 ${apiRequests.length} 个 API 请求`);
    
    // 检查控制台错误
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
        console.log(`控制台错误: ${msg.text()}`);
      }
    });
    
    if (consoleErrors.length > 0) {
      console.log(`\n检测到 ${consoleErrors.length} 个控制台错误`);
    }
    
    // 最终截图
    await page.screenshot({ 
      path: 'test-results/slurm-page-after-reload.png', 
      fullPage: true 
    });
  });
});
