const { test, expect } = require('@playwright/test');

/**
 * SLURM Tasks 统计信息调试测试
 * 用于诊断统计信息未能正确统计的问题
 */

// 等待页面加载完成
async function waitForPageLoad(page) {
  await page.waitForLoadState('networkidle', { timeout: 10000 });
  await page.waitForTimeout(500);
}

test.describe('SLURM Tasks 统计信息调试', () => {
  test('诊断统计信息加载问题', async ({ page }) => {
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('开始诊断 SLURM Tasks 统计信息问题');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

    // 登录步骤
    console.log('步骤 0: 执行登录');
    await page.goto('/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    console.log('✓ 登录完成\n');

    // 监听控制台消息
    const consoleMessages = [];
    page.on('console', msg => {
      const text = msg.text();
      consoleMessages.push({ type: msg.type(), text });
      console.log(`[Browser ${msg.type().toUpperCase()}] ${text}`);
    });

    // 监听 API 请求和响应
    const apiRequests = [];
    page.on('request', request => {
      if (request.url().includes('/api/slurm')) {
        apiRequests.push({
          method: request.method(),
          url: request.url(),
          timestamp: new Date().toISOString()
        });
        console.log(`\n[API 请求] ${request.method()} ${request.url()}`);
      }
    });

    page.on('response', async response => {
      if (response.url().includes('/api/slurm')) {
        const status = response.status();
        console.log(`[API 响应] ${status} ${response.url()}`);
        
        if (response.url().includes('/statistics')) {
          try {
            const body = await response.json();
            console.log('[统计 API 响应体]:', JSON.stringify(body, null, 2));
          } catch (e) {
            console.log('[统计 API 响应] 无法解析 JSON');
          }
        }
      }
    });

    // 导航到 SLURM Tasks 页面
    console.log('\n步骤 1: 导航到 /slurm-tasks');
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    console.log('✓ 页面加载完成');

    // 等待任务列表加载
    console.log('\n步骤 2: 等待任务列表加载');
    await page.waitForTimeout(2000);

    // 检查页面元素
    console.log('\n步骤 3: 检查页面元素');
    const pageTitle = await page.locator('h2').first().textContent();
    console.log(`页面标题: ${pageTitle}`);

    // 查找统计信息 Tab
    console.log('\n步骤 4: 查找统计信息 Tab');
    const statisticsTab = page.locator('text=统计信息');
    const hasStatisticsTab = await statisticsTab.isVisible().catch(() => false);
    console.log(`统计信息 Tab 可见: ${hasStatisticsTab}`);

    if (hasStatisticsTab) {
      // 点击统计信息 Tab
      console.log('\n步骤 5: 点击统计信息 Tab');
      await statisticsTab.click();
      await page.waitForTimeout(3000); // 等待统计数据加载

      // 检查加载状态
      console.log('\n步骤 6: 检查加载状态');
      const spinElement = page.locator('.ant-spin');
      const isLoading = await spinElement.isVisible().catch(() => false);
      console.log(`正在加载: ${isLoading}`);

      // 等待加载完成
      if (isLoading) {
        console.log('等待加载完成...');
        await page.waitForTimeout(5000);
      }

      // 检查统计卡片
      console.log('\n步骤 7: 检查统计卡片');
      const cards = page.locator('.ant-card');
      const cardCount = await cards.count();
      console.log(`找到 ${cardCount} 个卡片`);

      // 获取每个卡片的内容
      for (let i = 0; i < Math.min(cardCount, 10); i++) {
        try {
          const card = cards.nth(i);
          const cardText = await card.textContent();
          console.log(`卡片 ${i + 1}: ${cardText.substring(0, 100)}`);
        } catch (e) {
          console.log(`无法读取卡片 ${i + 1}`);
        }
      }

      // 检查是否有错误消息
      console.log('\n步骤 8: 检查错误消息');
      const errorMessage = page.locator('text=/加载失败|Error|错误/');
      const hasError = await errorMessage.isVisible().catch(() => false);
      console.log(`有错误消息: ${hasError}`);

      if (hasError) {
        const errorText = await errorMessage.textContent();
        console.log(`错误内容: ${errorText}`);
      }

      // 检查是否有空状态
      console.log('\n步骤 9: 检查空状态');
      const emptyState = page.locator('.ant-empty');
      const isEmpty = await emptyState.isVisible().catch(() => false);
      console.log(`显示空状态: ${isEmpty}`);

      if (isEmpty) {
        const emptyText = await emptyState.textContent();
        console.log(`空状态内容: ${emptyText}`);
      }

      // 检查 Statistic 组件
      console.log('\n步骤 10: 检查 Statistic 组件');
      const statistics = page.locator('.ant-statistic');
      const statisticCount = await statistics.count();
      console.log(`找到 ${statisticCount} 个统计数字组件`);

      for (let i = 0; i < Math.min(statisticCount, 15); i++) {
        try {
          const stat = statistics.nth(i);
          const title = await stat.locator('.ant-statistic-title').textContent();
          const value = await stat.locator('.ant-statistic-content-value').textContent();
          console.log(`统计 ${i + 1}: ${title} = ${value}`);
        } catch (e) {
          console.log(`无法读取统计 ${i + 1}`);
        }
      }

      // 截图保存
      console.log('\n步骤 11: 保存截图');
      await page.screenshot({ 
        path: 'test-results/slurm-tasks-statistics-debug.png',
        fullPage: true 
      });
      console.log('✓ 截图已保存');
    } else {
      console.log('❌ 未找到统计信息 Tab');
    }

    // 总结
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('诊断总结');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`控制台消息数: ${consoleMessages.length}`);
    console.log(`API 请求数: ${apiRequests.length}`);
    console.log('\nAPI 请求列表:');
    apiRequests.forEach((req, idx) => {
      console.log(`  ${idx + 1}. ${req.method} ${req.url}`);
    });
  });
});
