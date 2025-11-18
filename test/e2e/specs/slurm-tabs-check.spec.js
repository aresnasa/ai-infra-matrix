const { test, expect } = require('@playwright/test');

/**
 * SLURM 任务栏可见性测试
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';

const TEST_USER = {
  username: 'admin',
  password: 'admin123'
};

test.describe('SLURM 任务栏检查', () => {
  test('检查所有任务栏是否可见', async ({ page }) => {
    console.log('========================================');
    console.log('步骤 1: 登录');
    console.log('========================================');
    
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    // 检查是否需要登录
    if (page.url().includes('/login')) {
      await page.fill('input[type="text"], input[name="username"]', TEST_USER.username);
      await page.fill('input[type="password"], input[name="password"]', TEST_USER.password);
      await page.click('button[type="submit"], button:has-text("登录")');
      await page.waitForURL(/\/(dashboard|home|slurm|projects)/, { timeout: 10000 });
      console.log('✓ 登录成功');
    }
    
    console.log('========================================');
    console.log('步骤 2: 访问 SLURM 页面');
    console.log('========================================');
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000); // 等待页面完全渲染
    
    console.log('页面 URL:', page.url());
    console.log('页面标题:', await page.title());
    
    console.log('========================================');
    console.log('步骤 3: 检查 Tabs 组件');
    console.log('========================================');
    
    // 查找 Tabs 容器
    const tabsContainer = await page.locator('.ant-tabs').first();
    const tabsVisible = await tabsContainer.isVisible({ timeout: 5000 }).catch(() => false);
    console.log('Tabs 容器是否可见:', tabsVisible);
    
    if (!tabsVisible) {
      console.log('✗ Tabs 容器不可见！');
      // 截图保存
      await page.screenshot({ path: 'test-screenshots/slurm-tabs-hidden.png', fullPage: true });
      throw new Error('Tabs 容器不可见');
    }
    
    console.log('========================================');
    console.log('步骤 4: 检查所有任务栏标签');
    console.log('========================================');
    
    const expectedTabs = [
      { key: 'nodes', name: '节点管理', icon: 'DesktopOutlined' },
      { key: 'jobs', name: '作业队列', icon: 'PlayCircleOutlined' },
      { key: 'saltstack', name: 'SaltStack 集成', icon: 'ThunderboltOutlined' },
      { key: 'templates', name: '节点模板', icon: 'SettingOutlined' },
      { key: 'dashboard', name: '监控仪表板', icon: 'BarChartOutlined' },
      { key: 'external-clusters', name: '外部集群管理', icon: 'ClusterOutlined' },
      { key: 'tasks', name: '任务监控', icon: 'UnorderedListOutlined' }
    ];
    
    const foundTabs = [];
    const missingTabs = [];
    
    for (const tab of expectedTabs) {
      // 查找包含标签文本的 tab
      const tabLocator = page.locator('.ant-tabs-tab').filter({ hasText: tab.name });
      const isVisible = await tabLocator.isVisible().catch(() => false);
      
      if (isVisible) {
        foundTabs.push(tab.name);
        console.log(`✓ 找到标签: ${tab.name}`);
        
        // 检查是否可点击
        const boundingBox = await tabLocator.boundingBox();
        if (boundingBox) {
          console.log(`  位置: x=${Math.round(boundingBox.x)}, y=${Math.round(boundingBox.y)}, width=${Math.round(boundingBox.width)}, height=${Math.round(boundingBox.height)}`);
        }
      } else {
        missingTabs.push(tab.name);
        console.log(`✗ 未找到标签: ${tab.name}`);
      }
    }
    
    console.log('========================================');
    console.log('检查结果汇总');
    console.log('========================================');
    console.log(`找到的标签 (${foundTabs.length}/${expectedTabs.length}):`);
    foundTabs.forEach(name => console.log(`  ✓ ${name}`));
    
    if (missingTabs.length > 0) {
      console.log(`缺失的标签 (${missingTabs.length}):`);
      missingTabs.forEach(name => console.log(`  ✗ ${name}`));
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/slurm-tabs-check.png', fullPage: true });
    console.log('✓ 已保存截图: test-screenshots/slurm-tabs-check.png');
    
    // 验证所有标签都可见
    expect(foundTabs.length).toBe(expectedTabs.length);
    expect(missingTabs.length).toBe(0);
    
    console.log('========================================');
    console.log('步骤 5: 测试标签切换');
    console.log('========================================');
    
    // 点击每个标签
    for (const tab of expectedTabs.slice(0, 3)) { // 只测试前3个
      const tabLocator = page.locator('.ant-tabs-tab').filter({ hasText: tab.name });
      if (await tabLocator.isVisible()) {
        console.log(`点击标签: ${tab.name}`);
        await tabLocator.click();
        await page.waitForTimeout(500);
        
        // 检查是否激活
        const isActive = await tabLocator.locator('.ant-tabs-tab-btn').evaluate(el => 
          el.closest('.ant-tabs-tab').classList.contains('ant-tabs-tab-active')
        );
        console.log(`  ${tab.name} ${isActive ? '✓ 已激活' : '✗ 未激活'}`);
      }
    }
    
    console.log('========================================');
    console.log('测试完成 ✓');
    console.log('========================================');
  });
  
  test('检查 DOM 结构', async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    
    if (page.url().includes('/login')) {
      await page.fill('input[type="text"]', TEST_USER.username);
      await page.fill('input[type="password"]', TEST_USER.password);
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/(dashboard|home|slurm|projects)/, { timeout: 10000 });
    }
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    console.log('========================================');
    console.log('DOM 结构分析');
    console.log('========================================');
    
    // 检查页面结构
    const structure = await page.evaluate(() => {
      const tabs = document.querySelector('.ant-tabs');
      const tabNav = document.querySelector('.ant-tabs-nav');
      const tabContent = document.querySelector('.ant-tabs-content');
      
      return {
        hasTabs: !!tabs,
        hasTabNav: !!tabNav,
        hasTabContent: !!tabContent,
        tabsDisplay: tabs ? window.getComputedStyle(tabs).display : null,
        tabNavDisplay: tabNav ? window.getComputedStyle(tabNav).display : null,
        tabsVisibility: tabs ? window.getComputedStyle(tabs).visibility : null,
        tabsOpacity: tabs ? window.getComputedStyle(tabs).opacity : null,
        tabCount: tabNav ? tabNav.querySelectorAll('.ant-tabs-tab').length : 0
      };
    });
    
    console.log('页面结构信息:');
    console.log(JSON.stringify(structure, null, 2));
    
    expect(structure.hasTabs).toBeTruthy();
    expect(structure.hasTabNav).toBeTruthy();
    expect(structure.tabCount).toBeGreaterThan(0);
  });
});
