const { test, expect } = require('@playwright/test');

test.describe('SLURM Page - Top Tabs Check', () => {
  test('check what tabs exist at the top of the page', async ({ page }) => {
    console.log('\n=== Checking SLURM Page Top Tabs ===\n');

    // 访问 SLURM 页面
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 截图整个页面
    await page.screenshot({ 
      path: 'test-screenshots/slurm-page-top-tabs.png',
      fullPage: true 
    });

    // 获取页面 HTML
    const pageContent = await page.content();
    
    // 检查是否有重复的标签
    console.log('Page title:', await page.title());
    console.log('Page URL:', page.url());
    
    // 查找所有可能的标签元素
    const allTabs = await page.locator('.ant-tabs-tab, [role="tab"], .ant-menu-item').allTextContents();
    console.log('\nAll tabs found on page:');
    allTabs.forEach((tab, index) => {
      console.log(`  ${index + 1}. "${tab}"`);
    });

    // 查找包含 "Slurm" 或 "SLURM" 的文本
    const slurmTexts = await page.locator('text=/Slurm|SLURM/i').allTextContents();
    console.log('\nAll "Slurm" related texts:');
    slurmTexts.forEach((text, index) => {
      console.log(`  ${index + 1}. "${text}"`);
    });

    // 检查页面内是否有 "Slurm任务" 的标签
    const hasSlurmTaskTab = pageContent.includes('Slurm任务') || pageContent.includes('SLURM任务');
    console.log('\nHas "Slurm任务" tab:', hasSlurmTaskTab ? '❌ YES (should remove)' : '✅ NO');

    // 查找顶部导航
    const topNavItems = await page.locator('nav .ant-menu-item, header .ant-menu-item, .ant-layout-header .ant-menu-item').allTextContents();
    console.log('\nTop navigation items:');
    topNavItems.forEach((item, index) => {
      console.log(`  ${index + 1}. "${item}"`);
    });

    console.log('\n');
  });

  test('capture page structure', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle');

    // 获取页面的主要结构
    const headerHTML = await page.locator('header, .ant-layout-header').innerHTML().catch(() => 'Not found');
    const navHTML = await page.locator('nav, .ant-menu').first().innerHTML().catch(() => 'Not found');

    console.log('\n=== Page Structure ===\n');
    console.log('Header exists:', headerHTML !== 'Not found');
    console.log('Navigation exists:', navHTML !== 'Not found');

    // 检查是否有面包屑
    const breadcrumbs = await page.locator('.ant-breadcrumb-link').allTextContents();
    if (breadcrumbs.length > 0) {
      console.log('\nBreadcrumbs:');
      breadcrumbs.forEach((crumb, index) => {
        console.log(`  ${index + 1}. "${crumb}"`);
      });
    }

    // 检查页面标题区域
    const titles = await page.locator('h1, h2, h3, .ant-typography-title').allTextContents();
    console.log('\nPage titles/headings:');
    titles.forEach((title, index) => {
      console.log(`  ${index + 1}. "${title}"`);
    });
  });
});
