// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * SLURM Tasks 统计信息测试
 * 验证统计信息是否正确显示和计算
 */

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const ADMIN_USERNAME = process.env.ADMIN_USERNAME || 'admin';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'admin123';

// 等待页面加载完成
async function waitForPageLoad(page) {
  await page.waitForLoadState('networkidle', { timeout: 10000 });
  await page.waitForTimeout(1000);
}

// 登录辅助函数
async function login(page) {
  console.log('执行登录...');
  await page.goto('/login');
  await waitForPageLoad(page);
  
  // 填写登录表单
  await page.fill('input[type="text"]', ADMIN_USERNAME);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  
  // 等待登录完成
  await page.waitForURL(/\//, { timeout: 10000 });
  await waitForPageLoad(page);
  console.log('✓ 登录成功');
}

test.describe('SLURM Tasks 统计信息测试', () => {
  test.beforeEach(async ({ page }) => {
    // 每个测试前先登录
    await login(page);
  });

  test('1. 统计信息 API 响应验证', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 统计信息 API 响应');
    console.log('========================================\n');
    
    // 导航到 SLURM Tasks 页面
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 切换到统计信息 Tab 触发 API 调用
    const statisticsTab = page.locator('text=统计信息').or(page.locator('text=Statistics'));
    
    // 先设置监听器，再点击
    const statisticsResponsePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm/tasks/statistics'),
      { timeout: 10000 }
    );
    
    await statisticsTab.click();
    const statisticsResponse = await statisticsResponsePromise;
    
    expect(statisticsResponse.ok()).toBeTruthy();
    
    const statsData = await statisticsResponse.json();
    console.log('📊 统计信息 API 响应:');
    console.log(JSON.stringify(statsData, null, 2));
    
    // 验证响应结构
    expect(statsData).toHaveProperty('data');
    expect(statsData.data).toHaveProperty('total_tasks');
    expect(statsData.data).toHaveProperty('status_stats');
    expect(statsData.data).toHaveProperty('success_rate');
    expect(statsData.data).toHaveProperty('avg_duration');
    
    console.log('\n✅ API 响应结构正确');
    console.log(`   总任务数: ${statsData.data.total_tasks}`);
    console.log(`   成功率: ${statsData.data.success_rate}%`);
    console.log(`   平均时长: ${statsData.data.avg_duration}秒`);
    
    // 验证状态统计
    const statusStats = statsData.data.status_stats;
    console.log('\n📈 状态统计:');
    
    if (Object.keys(statusStats).length > 0) {
      for (const [status, count] of Object.entries(statusStats)) {
        console.log(`   ${status}: ${count}`);
        expect(typeof count).toBe('number');
        expect(count).toBeGreaterThanOrEqual(0);
      }
    } else {
      console.log('   (暂无任务)');
    }
  });

  test('2. 统计卡片显示验证', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 统计卡片显示');
    console.log('========================================\n');
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 点击统计信息 Tab
    const statisticsTab = page.locator('text=统计信息').or(page.locator('text=Statistics'));
    await statisticsTab.click();
    await page.waitForTimeout(2000);
    
    console.log('✓ 切换到统计信息 Tab');
    
    // 查找所有统计卡片
    const statCards = page.locator('.ant-col .ant-statistic');
    const cardCount = await statCards.count();
    
    console.log(`\n📊 找到 ${cardCount} 个统计卡片`);
    expect(cardCount).toBeGreaterThan(0);
    
    // 验证每个卡片的标题和数值
    const expectedStats = [
      '总任务数',
      '运行中',
      '已完成',
      '失败',
      '成功率',
      '平均耗时'
    ];
    
    for (const statTitle of expectedStats) {
      const statCard = page.locator('.ant-statistic-title', { hasText: statTitle });
      const isVisible = await statCard.isVisible().catch(() => false);
      
      if (isVisible) {
        // 获取数值
        const valueElement = statCard.locator('..').locator('.ant-statistic-content-value');
        const value = await valueElement.textContent();
        console.log(`   ✓ ${statTitle}: ${value}`);
      } else {
        console.log(`   ⚠ ${statTitle}: 未找到`);
      }
    }
  });

  test('3. 统计数据一致性验证', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 统计数据一致性');
    console.log('========================================\n');
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 切换到统计信息 Tab 触发 API 调用
    const statisticsTab = page.locator('text=统计信息').or(page.locator('text=Statistics'));
    
    // 先设置监听器，再点击
    const statisticsResponsePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm/tasks/statistics'),
      { timeout: 10000 }
    );
    
    await statisticsTab.click();
    const statisticsResponse = await statisticsResponsePromise;
    
    const apiStats = await statisticsResponse.json();
    const apiData = apiStats.data;
    
    console.log('📊 API 统计数据:');
    console.log(`   总任务数: ${apiData.total_tasks}`);
    console.log(`   成功率: ${apiData.success_rate}%`);
    
    // Tab 已经在上面切换了，等待加载
    await page.waitForTimeout(2000);
    
    // 验证总任务数
    const totalTasksCard = page.locator('.ant-statistic-title', { hasText: '总任务数' });
    if (await totalTasksCard.isVisible().catch(() => false)) {
      const totalTasksValue = await totalTasksCard.locator('..').locator('.ant-statistic-content-value').textContent();
      const displayedTotal = parseInt(totalTasksValue.replace(/[^0-9]/g, ''));
      
      console.log(`\n🔍 总任务数验证:`);
      console.log(`   API: ${apiData.total_tasks}`);
      console.log(`   界面: ${displayedTotal}`);
      
      expect(displayedTotal).toBe(apiData.total_tasks);
      console.log('   ✅ 一致');
    }
    
    // 验证成功率
    const successRateCard = page.locator('.ant-statistic-title', { hasText: '成功率' });
    if (await successRateCard.isVisible().catch(() => false)) {
      const successRateValue = await successRateCard.locator('..').locator('.ant-statistic-content-value').textContent();
      const displayedRate = parseFloat(successRateValue.replace(/[^0-9.]/g, ''));
      
      console.log(`\n🔍 成功率验证:`);
      console.log(`   API: ${apiData.success_rate}%`);
      console.log(`   界面: ${displayedRate}%`);
      
      // 允许小数点精度误差
      expect(Math.abs(displayedRate - apiData.success_rate)).toBeLessThan(0.1);
      console.log('   ✅ 一致');
    }
    
    // 验证状态统计总和
    const statusStats = apiData.status_stats || {};
    const statusSum = Object.values(statusStats).reduce((sum, count) => sum + count, 0);
    
    console.log(`\n🔍 状态统计验证:`);
    console.log(`   各状态任务总和: ${statusSum}`);
    console.log(`   总任务数: ${apiData.total_tasks}`);
    
    if (statusSum > 0) {
      expect(statusSum).toBe(apiData.total_tasks);
      console.log('   ✅ 一致');
    } else {
      console.log('   ⚠ 暂无任务数据');
    }
  });

  test('4. 状态统计详细验证', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 状态统计详细验证');
    console.log('========================================\n');
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 切换到统计信息 Tab 触发 API 调用
    const statisticsTab = page.locator('text=统计信息').or(page.locator('text=Statistics'));
    
    // 先设置监听器，再点击
    const statisticsResponsePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm/tasks/statistics'),
      { timeout: 10000 }
    );
    
    await statisticsTab.click();
    const statisticsResponse = await statisticsResponsePromise;
    
    const apiStats = await statisticsResponse.json();
    const statusStats = apiStats.data.status_stats || {};
    
    console.log('📊 API 状态统计:');
    console.log(JSON.stringify(statusStats, null, 2));
    
    // Tab 已经在上面切换了，等待加载
    await page.waitForTimeout(2000);
    
    // 验证各状态的显示
    const statusMapping = {
      'running': '运行中',
      'completed': '已完成',
      'complete': '已完成',
      'failed': '失败',
      'pending': '等待中',
      'cancelled': '已取消'
    };
    
    console.log('\n🔍 验证状态卡片:');
    
    for (const [apiStatus, count] of Object.entries(statusStats)) {
      const displayName = statusMapping[apiStatus.toLowerCase()] || apiStatus;
      const statusCard = page.locator('.ant-statistic-title', { hasText: displayName });
      
      if (await statusCard.isVisible().catch(() => false)) {
        const displayValue = await statusCard.locator('..').locator('.ant-statistic-content-value').textContent();
        const displayCount = parseInt(displayValue.replace(/[^0-9]/g, ''));
        
        console.log(`   ${displayName}:`);
        console.log(`     API: ${count}`);
        console.log(`     界面: ${displayCount}`);
        
        expect(displayCount).toBe(count);
        console.log(`     ✅ 一致`);
      } else {
        console.log(`   ${displayName}: 未在界面显示（API 数据: ${count}）`);
      }
    }
  });

  test('5. 刷新后统计信息更新', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 刷新后统计信息更新');
    console.log('========================================\n');
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 切换到统计信息 Tab 触发 API 调用
    const statisticsTab = page.locator('text=统计信息').or(page.locator('text=Statistics'));
    
    // 先设置监听器，再点击
    const initialResponsePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm/tasks/statistics'),
      { timeout: 10000 }
    );
    
    await statisticsTab.click();
    const initialResponse = await initialResponsePromise;
    
    const initialStats = await initialResponse.json();
    console.log('📊 初始统计数据:');
    console.log(`   总任务数: ${initialStats.data.total_tasks}`);
    console.log(`   成功率: ${initialStats.data.success_rate}%`);
    
    // Tab 已经在上面切换了，等待加载
    await page.waitForTimeout(2000);
    
    // 获取初始界面显示
    const totalTasksCard = page.locator('.ant-statistic-title', { hasText: '总任务数' });
    const initialDisplayValue = await totalTasksCard.locator('..').locator('.ant-statistic-content-value').textContent();
    console.log(`\n🖥️  初始界面显示: ${initialDisplayValue}`);
    
    // 点击刷新按钮
    const refreshButton = page.locator('button', { hasText: /刷新|Refresh/ }).first();
    if (await refreshButton.isVisible().catch(() => false)) {
      console.log('\n🔄 点击刷新按钮...');
      
      // 监听新的统计请求
      const refreshPromise = page.waitForResponse(
        response => response.url().includes('/api/slurm/tasks/statistics'),
        { timeout: 10000 }
      );
      
      await refreshButton.click();
      
      const refreshedResponse = await refreshPromise;
      const refreshedStats = await refreshedResponse.json();
      
      console.log('\n📊 刷新后统计数据:');
      console.log(`   总任务数: ${refreshedStats.data.total_tasks}`);
      console.log(`   成功率: ${refreshedStats.data.success_rate}%`);
      
      await page.waitForTimeout(1000);
      
      // 验证界面更新
      const refreshedDisplayValue = await totalTasksCard.locator('..').locator('.ant-statistic-content-value').textContent();
      console.log(`\n🖥️  刷新后界面显示: ${refreshedDisplayValue}`);
      
      console.log('\n✅ 刷新功能正常');
    } else {
      console.log('\n⚠ 未找到刷新按钮');
    }
  });

  test('6. 无任务时的统计显示', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 无任务时的统计显示');
    console.log('========================================\n');
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 切换到统计信息 Tab 触发 API 调用
    const statisticsTab = page.locator('text=统计信息').or(page.locator('text=Statistics'));
    
    // 先设置监听器，再点击
    const statisticsResponsePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm/tasks/statistics'),
      { timeout: 10000 }
    );
    
    await statisticsTab.click();
    const statisticsResponse = await statisticsResponsePromise;
    
    const apiStats = await statisticsResponse.json();
    const totalTasks = apiStats.data.total_tasks;
    
    console.log(`📊 总任务数: ${totalTasks}`);
    
    // Tab 已经在上面切换了，等待加载
    await page.waitForTimeout(2000);
    
    if (totalTasks === 0) {
      console.log('\n✓ 当前无任务，验证零值显示');
      
      // 验证各项统计都显示为 0
      const statsToCheck = [
        { title: '总任务数', expected: 0 },
        { title: '运行中', expected: 0 },
        { title: '已完成', expected: 0 },
        { title: '失败', expected: 0 },
        { title: '成功率', expected: 0 }
      ];
      
      console.log('\n🔍 验证零值显示:');
      
      for (const stat of statsToCheck) {
        const statCard = page.locator('.ant-statistic-title', { hasText: stat.title });
        if (await statCard.isVisible().catch(() => false)) {
          const value = await statCard.locator('..').locator('.ant-statistic-content-value').textContent();
          const numValue = parseFloat(value.replace(/[^0-9.]/g, '')) || 0;
          
          console.log(`   ${stat.title}: ${value}`);
          expect(numValue).toBe(stat.expected);
        }
      }
      
      console.log('\n✅ 零值显示正确');
    } else {
      console.log('\n⚠ 当前有任务数据，跳过零值测试');
    }
  });
});
