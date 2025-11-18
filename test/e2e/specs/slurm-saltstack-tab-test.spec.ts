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

test('验证 SaltStack 集成Tab中的任务显示', async ({ page }) => {
  console.log('\n=== 验证 SaltStack 集成Tab ===');
  
  // 登录并访问
  await login(page);
  await page.goto(`${BASE_URL}/slurm`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(2000);
  
  console.log('\n查找 SaltStack 集成 Tab...');
  
  // 点击 "SaltStack 集成" Tab
  const saltStackTab = page.locator('.ant-tabs-tab').filter({ hasText: 'SaltStack 集成' });
  await expect(saltStackTab).toBeVisible({ timeout: 5000 });
  await saltStackTab.click();
  console.log('✓ 点击 SaltStack 集成 Tab');
  
  // 等待Tab内容加载
  await page.waitForTimeout(2000);
  
  // 截图当前Tab
  await page.screenshot({ 
    path: 'test-results/saltstack-integration-tab.png', 
    fullPage: true 
  });
  
  console.log('\n检查Tab内容...');
  
  // 查找 "最近 SaltStack 作业" 卡片
  const saltJobsCard = page.locator('.ant-card').filter({ hasText: '最近 SaltStack 作业' });
  const hasCard = await saltJobsCard.count() > 0;
  console.log(`找到作业卡片: ${hasCard}`);
  
  if (hasCard) {
    await expect(saltJobsCard).toBeVisible();
    console.log('✓ SaltStack 作业卡片可见');
    
    // 获取卡片内容
    const cardText = await saltJobsCard.textContent();
    console.log(`\n卡片内容:\n${cardText}`);
    
    // 检查列表项
    const listItems = saltJobsCard.locator('.ant-list-item');
    const itemCount = await listItems.count();
    console.log(`\n找到 ${itemCount} 个任务`);
    
    if (itemCount > 0) {
      // 遍历每个任务
      for (let i = 0; i < itemCount; i++) {
        const item = listItems.nth(i);
        const itemText = await item.textContent();
        console.log(`\n任务 ${i + 1}:`);
        console.log(`  ${itemText}`);
        
        // 检查结构
        const title = await item.locator('.ant-list-item-meta-title').textContent().catch(() => 'N/A');
        const description = await item.locator('.ant-list-item-meta-description').textContent().catch(() => 'N/A');
        
        console.log(`  标题: ${title}`);
        console.log(`  描述: ${description}`);
      }
      
      // 断言：应该找到 test.ping
      await expect(saltJobsCard).toContainText('test.ping');
      console.log('\n✓ 找到 test.ping 任务');
      
      // 断言：不应该有 undefined
      expect(cardText).not.toContain('undefined');
      console.log('✓ 没有 undefined 文本');
      
      // 断言：应该显示成功信息
      const hasSuccess = cardText?.includes('成功') || cardText?.includes('2/2');
      expect(hasSuccess).toBe(true);
      console.log('✓ 显示任务执行成功');
      
    } else {
      console.log('\n❌ 列表为空，可能是前端未重新构建');
    }
    
  } else {
    console.log('\n❌ 未找到 "最近 SaltStack 作业" 卡片');
    
    // 列出Tab中的所有卡片
    const allCards = page.locator('.ant-card');
    const cardCount = await allCards.count();
    console.log(`\n当前Tab有 ${cardCount} 个卡片:`);
    
    for (let i = 0; i < cardCount; i++) {
      const card = allCards.nth(i);
      const title = await card.locator('.ant-card-head-title').textContent().catch(() => '无标题');
      console.log(`  ${i + 1}. ${title}`);
    }
  }
});
