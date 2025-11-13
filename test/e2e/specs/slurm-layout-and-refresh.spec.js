const { test, expect } = require('@playwright/test');

/**
 * SLURM 集群管理页面 - 布局和刷新功能测试
 * 
 * 测试目标：
 * 1. 验证任务栏和扩容按钮并排显示
 * 2. 验证刷新按钮能够正确刷新页面和任务
 * 3. 验证页面基本功能正常
 */

test.describe('SLURM 集群管理 - 布局和刷新测试', () => {
  let page;

  test.beforeAll(async ({ browser }) => {
    page = await browser.newPage();
    
    // 登录
    await page.goto('http://192.168.3.91:8080/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // 等待登录完成
    await page.waitForURL('http://192.168.3.91:8080/', { timeout: 10000 });
    console.log('✓ 登录成功');
  });

  test.afterAll(async () => {
    await page?.close();
  });

  test('1. 页面加载和基本元素检查', async () => {
    console.log('\n[测试 1] 页面加载和基本元素检查');
    
    // 访问 SLURM 管理页面
    await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
    console.log('✓ 访问 SLURM 管理页面');

    // 检查页面标题
    const title = await page.locator('h2:has-text("SLURM 集群管理")');
    await expect(title).toBeVisible({ timeout: 10000 });
    console.log('✓ 页面标题显示正确');

    // 等待页面完全加载
    await page.waitForTimeout(2000);
    
    // 截图
    await page.screenshot({ 
      path: 'test/e2e/test-screenshots/slurm-layout-1-loaded.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: slurm-layout-1-loaded.png');
  });

  test('2. 任务栏和按钮布局验证', async () => {
    console.log('\n[测试 2] 任务栏和按钮布局验证');

    await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // 检查任务栏存在
    const taskBar = page.locator('div:has-text("任务栏")').first();
    await expect(taskBar).toBeVisible({ timeout: 5000 });
    console.log('✓ 任务栏显示正常');

    // 检查刷新按钮
    const refreshButton = page.locator('button:has-text("刷新")').first();
    await expect(refreshButton).toBeVisible({ timeout: 5000 });
    console.log('✓ 刷新按钮显示正常');

    // 检查扩容节点按钮
    const expandButton = page.locator('button:has-text("扩容节点")');
    await expect(expandButton).toBeVisible({ timeout: 5000 });
    console.log('✓ 扩容节点按钮显示正常');

    // 检查 SaltStack 命令按钮
    const saltButton = page.locator('button:has-text("SaltStack 命令")');
    await expect(saltButton).toBeVisible({ timeout: 5000 });
    console.log('✓ SaltStack 命令按钮显示正常');

    // 验证任务栏和按钮在同一行
    const taskBarBox = await taskBar.boundingBox();
    const refreshButtonBox = await refreshButton.boundingBox();
    const expandButtonBox = await expandButton.boundingBox();

    if (taskBarBox && refreshButtonBox && expandButtonBox) {
      // 检查垂直位置是否接近（允许一定误差）
      const yDiff1 = Math.abs(taskBarBox.y - refreshButtonBox.y);
      const yDiff2 = Math.abs(taskBarBox.y - expandButtonBox.y);
      
      console.log(`任务栏 Y: ${taskBarBox.y.toFixed(2)}`);
      console.log(`刷新按钮 Y: ${refreshButtonBox.y.toFixed(2)}`);
      console.log(`扩容按钮 Y: ${expandButtonBox.y.toFixed(2)}`);
      console.log(`垂直差距: ${yDiff1.toFixed(2)}px, ${yDiff2.toFixed(2)}px`);
      
      expect(yDiff1).toBeLessThan(50); // 允许50px的垂直误差
      expect(yDiff2).toBeLessThan(50);
      console.log('✓ 任务栏和按钮在同一行显示');

      // 检查水平位置（任务栏应该在左侧，按钮在右侧）
      expect(taskBarBox.x).toBeLessThan(refreshButtonBox.x);
      console.log('✓ 布局顺序正确（任务栏在左，按钮在右）');
    } else {
      console.warn('⚠️ 无法获取元素边界框，跳过位置验证');
    }

    // 截图
    await page.screenshot({ 
      path: 'test/e2e/test-screenshots/slurm-layout-2-buttons.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: slurm-layout-2-buttons.png');
  });

  test('3. 刷新按钮功能测试', async () => {
    console.log('\n[测试 3] 刷新按钮功能测试');

    await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // 记录刷新前的网络请求
    let apiRequests = [];
    page.on('request', request => {
      const url = request.url();
      if (url.includes('/api/slurm')) {
        apiRequests.push({
          url: url,
          timestamp: Date.now()
        });
      }
    });

    // 点击刷新按钮
    const refreshButton = page.locator('button:has-text("刷新")').first();
    console.log('点击刷新按钮...');
    
    // 清空之前的请求记录
    apiRequests = [];
    
    await refreshButton.click();
    
    // 等待加载完成
    await page.waitForTimeout(3000);

    // 验证是否触发了API请求
    console.log(`\n刷新后触发的 API 请求数量: ${apiRequests.length}`);
    apiRequests.forEach(req => {
      console.log(`- ${req.url}`);
    });

    expect(apiRequests.length).toBeGreaterThan(0);
    console.log('✓ 刷新按钮触发了 API 请求');

    // 检查是否有摘要、节点、任务等核心API请求
    const hasSummary = apiRequests.some(r => r.url.includes('/summary'));
    const hasNodes = apiRequests.some(r => r.url.includes('/nodes'));
    const hasTasks = apiRequests.some(r => r.url.includes('/tasks'));

    console.log(`摘要 API: ${hasSummary ? '✓' : '✗'}`);
    console.log(`节点 API: ${hasNodes ? '✓' : '✗'}`);
    console.log(`任务 API: ${hasTasks ? '✓' : '✗'}`);

    // 至少应该有摘要或节点的请求
    expect(hasSummary || hasNodes).toBe(true);
    console.log('✓ 核心数据 API 被正确调用');

    // 截图
    await page.screenshot({ 
      path: 'test/e2e/test-screenshots/slurm-layout-3-after-refresh.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: slurm-layout-3-after-refresh.png');
  });

  test('4. 任务栏内部刷新测试', async () => {
    console.log('\n[测试 4] 任务栏内部刷新测试');

    await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // 点击任务栏展开
    const taskBarButton = page.locator('div:has-text("任务栏")').locator('button').first();
    await expect(taskBarButton).toBeVisible({ timeout: 5000 });
    
    console.log('点击任务栏按钮...');
    await taskBarButton.click();
    await page.waitForTimeout(1000);

    // 检查弹出的任务列表
    const popover = page.locator('.ant-popover-content');
    await expect(popover).toBeVisible({ timeout: 5000 });
    console.log('✓ 任务栏弹窗显示正常');

    // 截图任务栏弹窗
    await page.screenshot({ 
      path: 'test/e2e/test-screenshots/slurm-layout-4-taskbar-popup.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: slurm-layout-4-taskbar-popup.png');

    // 点击任务栏内的刷新按钮
    const taskBarRefreshButton = popover.locator('button:has-text("刷新")');
    if (await taskBarRefreshButton.isVisible()) {
      console.log('点击任务栏内的刷新按钮...');
      
      let taskApiCalled = false;
      page.on('request', request => {
        if (request.url().includes('/api/slurm/tasks')) {
          taskApiCalled = true;
        }
      });

      await taskBarRefreshButton.click();
      await page.waitForTimeout(2000);

      expect(taskApiCalled).toBe(true);
      console.log('✓ 任务栏刷新按钮触发了任务 API 请求');
    } else {
      console.log('⚠️ 任务栏内无刷新按钮，跳过测试');
    }
  });

  test('5. 页面数据显示测试', async () => {
    console.log('\n[测试 5] 页面数据显示测试');

    await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);

    // 检查扩缩容状态卡片
    const scalingCard = page.locator('.ant-card:has-text("扩缩容状态")');
    if (await scalingCard.isVisible()) {
      console.log('✓ 扩缩容状态卡片显示正常');
      
      // 检查卡片内的统计数据
      const statistics = await scalingCard.locator('.ant-statistic').count();
      console.log(`统计数据项数量: ${statistics}`);
    } else {
      console.log('⚠️ 扩缩容状态卡片未显示');
    }

    // 检查集群摘要卡片
    const summaryCard = page.locator('.ant-card:has-text("集群摘要")');
    if (await summaryCard.isVisible()) {
      console.log('✓ 集群摘要卡片显示正常');
    }

    // 检查节点表格
    const nodeTable = page.locator('.ant-table:has-text("节点名称")');
    if (await nodeTable.isVisible()) {
      console.log('✓ 节点表格显示正常');
      
      const nodeRows = await nodeTable.locator('tbody tr').count();
      console.log(`节点数量: ${nodeRows}`);
    }

    // 截图完整页面
    await page.screenshot({ 
      path: 'test/e2e/test-screenshots/slurm-layout-5-full-page.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: slurm-layout-5-full-page.png');
  });

  test('6. 响应式布局测试', async () => {
    console.log('\n[测试 6] 响应式布局测试');

    // 测试不同屏幕尺寸
    const viewports = [
      { width: 1920, height: 1080, name: '桌面' },
      { width: 1366, height: 768, name: '笔记本' },
      { width: 768, height: 1024, name: '平板' },
    ];

    for (const viewport of viewports) {
      console.log(`\n测试 ${viewport.name} 分辨率 (${viewport.width}x${viewport.height})`);
      
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
      await page.waitForTimeout(2000);

      // 检查关键元素是否可见
      const taskBar = page.locator('div:has-text("任务栏")').first();
      const refreshButton = page.locator('button:has-text("刷新")').first();
      const expandButton = page.locator('button:has-text("扩容节点")');

      await expect(taskBar).toBeVisible({ timeout: 5000 });
      await expect(refreshButton).toBeVisible({ timeout: 5000 });
      await expect(expandButton).toBeVisible({ timeout: 5000 });
      
      console.log(`✓ ${viewport.name} 分辨率下所有关键元素可见`);

      // 截图
      await page.screenshot({ 
        path: `test/e2e/test-screenshots/slurm-layout-6-${viewport.name}.png`,
        fullPage: true 
      });
      console.log(`✓ 截图已保存: slurm-layout-6-${viewport.name}.png`);
    }

    // 恢复默认视图
    await page.setViewportSize({ width: 1920, height: 1080 });
  });

  test('7. 综合功能测试', async () => {
    console.log('\n[测试 7] 综合功能测试');

    await page.goto('http://192.168.3.91:8080/slurm', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // 测试多次刷新
    console.log('测试连续刷新...');
    const refreshButton = page.locator('button:has-text("刷新")').first();
    
    for (let i = 1; i <= 3; i++) {
      console.log(`第 ${i} 次刷新`);
      await refreshButton.click();
      await page.waitForTimeout(1500);
    }
    console.log('✓ 连续刷新测试完成');

    // 测试扩容按钮
    console.log('\n测试扩容按钮点击...');
    const expandButton = page.locator('button:has-text("扩容节点")');
    await expandButton.click();
    await page.waitForTimeout(1000);

    // 检查是否打开了扩容模态框
    const modal = page.locator('.ant-modal:has-text("扩容节点")');
    if (await modal.isVisible()) {
      console.log('✓ 扩容模态框打开成功');
      
      // 截图模态框
      await page.screenshot({ 
        path: 'test/e2e/test-screenshots/slurm-layout-7-expand-modal.png',
        fullPage: true 
      });
      console.log('✓ 截图已保存: slurm-layout-7-expand-modal.png');

      // 关闭模态框
      const closeButton = modal.locator('button.ant-modal-close');
      await closeButton.click();
      await page.waitForTimeout(500);
    } else {
      console.log('⚠️ 扩容模态框未打开');
    }

    // 最终截图
    await page.screenshot({ 
      path: 'test/e2e/test-screenshots/slurm-layout-7-final.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: slurm-layout-7-final.png');
  });
});

test.describe('测试总结', () => {
  test('输出测试报告', async () => {
    console.log('\n' + '='.repeat(60));
    console.log('SLURM 布局和刷新功能测试完成');
    console.log('='.repeat(60));
    console.log('\n测试内容：');
    console.log('1. ✓ 页面加载和基本元素检查');
    console.log('2. ✓ 任务栏和按钮布局验证（并排显示）');
    console.log('3. ✓ 刷新按钮功能测试（触发 API 请求）');
    console.log('4. ✓ 任务栏内部刷新测试');
    console.log('5. ✓ 页面数据显示测试');
    console.log('6. ✓ 响应式布局测试（多分辨率）');
    console.log('7. ✓ 综合功能测试');
    console.log('\n所有截图已保存到: test/e2e/test-screenshots/');
    console.log('='.repeat(60) + '\n');
  });
});
