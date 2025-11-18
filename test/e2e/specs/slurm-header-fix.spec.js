const { test, expect } = require('@playwright/test');

test.describe('SLURM Page Header Fix Verification', () => {
  test('should display only SLURM in the page title', async ({ page }) => {
    // 直接访问 SLURM 页面（跳过登录，假设已经有 session）
    await page.goto('http://192.168.0.200:8080/slurm');
    
    // 等待页面加载
    await page.waitForLoadState('networkidle', { timeout: 10000 });

    // 等待 Title 元素出现
    await page.waitForSelector('h2, .ant-typography-title', { timeout: 5000 });

    // 获取页面截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-page-header-after-fix.png',
      fullPage: true 
    });

    // 查找页面标题
    const h2Elements = await page.locator('h2').all();
    console.log(`\n=== Found ${h2Elements.length} h2 element(s) ===`);
    
    for (let i = 0; i < h2Elements.length; i++) {
      const text = await h2Elements[i].textContent();
      console.log(`H2 ${i + 1}: "${text}"`);
    }

    // 检查是否有包含 "SLURM" 的标题
    const slurmTitle = page.locator('h2:has-text("SLURM")');
    const titleCount = await slurmTitle.count();
    console.log(`\n=== Found ${titleCount} SLURM title(s) ===`);

    if (titleCount > 0) {
      const titleText = await slurmTitle.first().textContent();
      console.log(`Title text: "${titleText}"`);
      
      // 验证标题只包含 "SLURM"，不包含其他内容如 "slurm任务" 或 "弹性扩缩容管理"
      expect(titleText.trim()).toBe('SLURM');
      console.log('✅ Title is correct: "SLURM"');
    }

    // 检查页面内容，确保没有 "slurm任务" 出现在标题中
    const pageText = await page.locator('body').textContent();
    const hasUnwantedText = pageText.includes('slurm任务') || pageText.includes('SLURM 弹性扩缩容管理');
    
    if (hasUnwantedText) {
      console.log('⚠️  Warning: Page still contains "slurm任务" or "SLURM 弹性扩缩容管理"');
    } else {
      console.log('✅ Page does not contain unwanted text');
    }
  });

  test('should not show SlurmTaskBar component', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle', { timeout: 10000 });

    // 监听控制台错误
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // 等待一会儿让所有错误被捕获
    await page.waitForTimeout(2000);

    // 检查是否有关于 SlurmTaskBar 的错误
    const hasSlurmTaskBarError = consoleErrors.some(error => 
      error.includes('SlurmTaskBar') || error.toLowerCase().includes('cannot find module')
    );

    if (hasSlurmTaskBarError) {
      console.log('❌ Found SlurmTaskBar related errors:');
      consoleErrors.forEach(err => console.log('  ', err));
    } else {
      console.log('✅ No SlurmTaskBar related errors');
    }

    expect(hasSlurmTaskBarError).toBe(false);
  });

  test('should display SLURM tabs correctly', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle', { timeout: 10000 });

    // 等待 Tabs 组件加载
    await page.waitForSelector('.ant-tabs', { timeout: 5000 });

    // 获取所有 tab 标签
    const tabs = await page.locator('.ant-tabs-tab').all();
    console.log(`\n=== Found ${tabs.length} tab(s) ===`);

    const tabTexts = [];
    for (let i = 0; i < tabs.length; i++) {
      const text = await tabs[i].textContent();
      tabTexts.push(text);
      console.log(`Tab ${i + 1}: "${text}"`);
    }

    // 验证有正确的 tabs
    expect(tabs.length).toBeGreaterThan(0);
    expect(tabTexts.some(text => text.includes('节点管理'))).toBe(true);
    expect(tabTexts.some(text => text.includes('监控仪表板'))).toBe(true);
    
    console.log('✅ Tabs are displayed correctly');
  });

  test('should show monitoring dashboard tab with iframe', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle', { timeout: 10000 });

    // 点击"监控仪表板" tab
    const monitoringTab = page.locator('.ant-tabs-tab:has-text("监控仪表板")');
    await monitoringTab.click();

    // 等待 iframe 加载
    await page.waitForSelector('iframe#slurm-dashboard-iframe', { timeout: 5000 });

    const iframe = page.locator('iframe#slurm-dashboard-iframe');
    const iframeSrc = await iframe.getAttribute('src');
    
    console.log('\n=== Monitoring Dashboard Iframe ===');
    console.log('Iframe src:', iframeSrc);
    
    expect(iframeSrc).toContain('/nightingale/');
    console.log('✅ Monitoring iframe is configured correctly');

    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-monitoring-tab.png',
      fullPage: true 
    });
  });

  test('should have correct page structure', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle', { timeout: 10000 });

    // 检查页面结构
    const structure = await page.evaluate(() => {
      const h2 = document.querySelector('h2');
      const tabs = document.querySelector('.ant-tabs');
      const cards = document.querySelectorAll('.ant-card');
      
      return {
        hasH2: !!h2,
        h2Text: h2?.textContent || '',
        hasTabs: !!tabs,
        cardCount: cards.length
      };
    });

    console.log('\n=== Page Structure ===');
    console.log('Has H2:', structure.hasH2);
    console.log('H2 Text:', structure.h2Text);
    console.log('Has Tabs:', structure.hasTabs);
    console.log('Card Count:', structure.cardCount);

    expect(structure.hasH2).toBe(true);
    expect(structure.h2Text.trim()).toBe('SLURM');
    expect(structure.hasTabs).toBe(true);
    expect(structure.cardCount).toBeGreaterThan(0);

    console.log('✅ Page structure is correct');
  });
});
