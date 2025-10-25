import { test } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';

async function login(page: any) {
  await page.goto(`${BASE_URL}`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(1000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  if (!hasLoginForm) return;
  
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
}

test('探索 SLURM 页面结构', async ({ page }) => {
  console.log('\n=== 探索 SLURM 页面 ===');
  
  await login(page);
  await page.goto(`${BASE_URL}/slurm`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(3000);
  
  // 查找所有Tab
  const tabs = page.locator('.ant-tabs-tab');
  const tabCount = await tabs.count();
  console.log(`\n找到 ${tabCount} 个Tab:`);
  
  for (let i = 0; i < tabCount; i++) {
    const tab = tabs.nth(i);
    const tabText = await tab.textContent();
    console.log(`  ${i + 1}. ${tabText}`);
  }
  
  // 查找所有卡片
  const cards = page.locator('.ant-card');
  const cardCount = await cards.count();
  console.log(`\n找到 ${cardCount} 个卡片组件`);
  
  for (let i = 0; i < cardCount; i++) {
    const card = cards.nth(i);
    const cardTitle = await card.locator('.ant-card-head-title').textContent().catch(() => '无标题');
    const cardText = await card.textContent();
    console.log(`\n卡片 ${i + 1}: ${cardTitle}`);
    console.log(`  内容预览: ${cardText?.substring(0, 150)}...`);
  }
  
  // 查找包含 "SaltStack" 的所有元素
  console.log('\n=== 搜索 SaltStack 相关内容 ===');
  const saltElements = page.locator('text=/SaltStack/i');
  const saltCount = await saltElements.count();
  console.log(`找到 ${saltCount} 个包含 "SaltStack" 的元素`);
  
  for (let i = 0; i < Math.min(saltCount, 5); i++) {
    const element = saltElements.nth(i);
    const text = await element.textContent();
    console.log(`  ${i + 1}. ${text}`);
  }
  
  // 完整页面截图
  await page.screenshot({ 
    path: 'test-results/slurm-page-structure.png', 
    fullPage: true 
  });
  
  console.log('\n✓ 页面结构探索完成');
});
