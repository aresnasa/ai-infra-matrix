import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';

async function login(page: any) {
  await page.goto(`${BASE_URL}`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(1000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  if (!hasLoginForm) {
    console.log('✓ 已登录');
    return;
  }
  
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  console.log('✓ 登录完成');
}

test.describe('SLURM SaltStack 任务验证', () => {
  
  test('验证 SaltStack 任务列表正确显示', async ({ page }) => {
    console.log('\n=== 验证 SaltStack 任务显示 ===');
    
    // 登录
    await login(page);
    
    // 访问 SLURM 页面
    await page.goto(`${BASE_URL}/slurm`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    // 等待页面完全加载
    await page.waitForTimeout(3000);
    
    console.log('\n检查页面内容...');
    
    // 查找 "最近 SaltStack 作业" 卡片
    const saltJobsCard = page.locator('div.ant-card').filter({ hasText: '最近 SaltStack 作业' });
    await expect(saltJobsCard).toBeVisible({ timeout: 10000 });
    console.log('✓ 找到 SaltStack 作业卡片');
    
    // 截图卡片
    await saltJobsCard.screenshot({ 
      path: 'test-results/saltstack-jobs-card-detail.png' 
    });
    
    // 获取卡片内的文本内容
    const cardText = await saltJobsCard.textContent();
    console.log('\n卡片内容:');
    console.log(cardText);
    
    // 检查是否有任务列表
    const listItems = saltJobsCard.locator('.ant-list-item');
    const itemCount = await listItems.count();
    console.log(`\n找到 ${itemCount} 个列表项`);
    
    if (itemCount > 0) {
      console.log('\n列表项内容:');
      for (let i = 0; i < itemCount; i++) {
        const item = listItems.nth(i);
        const itemText = await item.textContent();
        console.log(`  ${i + 1}. ${itemText}`);
        
        // 检查每个项目的结构
        const meta = item.locator('.ant-list-item-meta');
        const title = await meta.locator('.ant-list-item-meta-title').textContent();
        const description = await meta.locator('.ant-list-item-meta-description').textContent();
        
        console.log(`    标题: ${title}`);
        console.log(`    描述: ${description}`);
      }
      
      // 验证：应该找到 test.ping 任务
      await expect(saltJobsCard).toContainText('test.ping');
      console.log('\n✓ 找到 test.ping 任务');
      
      // 验证：应该显示目标 "*"
      await expect(saltJobsCard).toContainText('*');
      console.log('✓ 找到目标 "*"');
      
      // 验证：应该显示成功状态
      await expect(saltJobsCard).toContainText('成功');
      console.log('✓ 找到成功状态');
      
      // 验证：不应该包含 "undefined"
      expect(cardText).not.toContain('undefined');
      console.log('✓ 没有 undefined 文本');
      
    } else {
      console.log('\n❌ 没有找到任何列表项');
      console.log('可能原因:');
      console.log('1. 前端代码未重新构建');
      console.log('2. API 返回为空');
      console.log('3. 渲染逻辑有问题');
    }
    
    // 完整页面截图
    await page.screenshot({ 
      path: 'test-results/slurm-page-saltstack-tasks.png', 
      fullPage: true 
    });
  });
});
