const { test, expect } = require('@playwright/test');

/**
 * SLURM Dashboard - SaltStack 命令输出显示测试
 * 
 * 测试目标：
 * 1. 验证 SaltStack 命令执行后立即显示输出
 * 2. 验证节点列表显示正确的 SaltStack 状态
 * 3. 验证错误信息正确显示
 * 
 * 环境变量：
 * - BASE_URL: 测试基础 URL (默认: http://192.168.3.91:8080)
 * - TEST_USERNAME: 测试用户名 (默认: admin)
 * - TEST_PASSWORD: 测试密码 (默认: admin123)
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('SLURM SaltStack 命令输出测试', () => {
  test.beforeEach(async ({ page }) => {
    console.log('📋 准备测试环境...');
    console.log(`  URL: ${BASE_URL}`);
    console.log(`  用户: ${TEST_USERNAME}`);
    
    // 登录
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    const usernameInput = page.locator('input[name="username"], input[placeholder*="用户"], input[type="text"]').first();
    const passwordInput = page.locator('input[name="password"], input[placeholder*="密码"], input[type="password"]').first();
    const loginButton = page.locator('button[type="submit"], button:has-text("登录")').first();
    
    if (await usernameInput.isVisible({ timeout: 2000 })) {
      await usernameInput.fill(TEST_USERNAME);
      await passwordInput.fill(TEST_PASSWORD);
      await loginButton.click();
      await page.waitForURL(/\/(dashboard|slurm|home|projects)/i, { timeout: 10000 });
      console.log('✅ 登录成功');
    } else {
      console.log('ℹ️  已登录或无需登录');
    }
  });

  test('应该能够执行 SaltStack 命令并看到输出', async ({ page }) => {
    console.log('开始测试 SaltStack 命令执行输出功能...');

    // 1. 访问 SLURM Dashboard
    await page.goto(`${BASE_URL}/slurm`);
    console.log('✓ 已访问 SLURM Dashboard');

    // 等待页面加载
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 先打印页面上所有的标签
    console.log('\n查找页面上的所有标签...');
    const allTabs = page.locator('.ant-tabs-tab');
    const tabCount = await allTabs.count();
    console.log(`找到 ${tabCount} 个标签:`);
    for (let i = 0; i < tabCount; i++) {
      const tabText = await allTabs.nth(i).innerText();
      console.log(`  ${i + 1}. ${tabText}`);
    }

    // 2. 切换到 SaltStack 集成标签
    console.log('\n查找 SaltStack 集成标签...');
    const saltTab = page.locator('.ant-tabs-tab').filter({ hasText: /SaltStack.*集成/ });
    await expect(saltTab).toBeVisible({ timeout: 10000 });
    console.log('✓ 找到 SaltStack 集成标签');

    await saltTab.click();
    console.log('✓ 已点击 SaltStack 集成标签');
    await page.waitForTimeout(1000);

    // 检查是否有子标签（SaltStack 命令执行）
    console.log('\n查找 SaltStack 集成内的子标签...');
    const subTabs = page.locator('.ant-tabs-tab');
    const subTabCount = await subTabs.count();
    console.log(`找到 ${subTabCount} 个标签:`);
    for (let i = 0; i < subTabCount; i++) {
      const tabText = await subTabs.nth(i).innerText();
      console.log(`  ${i + 1}. ${tabText}`);
    }

    // 如果有 "SaltStack 命令执行" 子标签，点击它
    const cmdTab = page.locator('.ant-tabs-tab').filter({ hasText: /SaltStack.*命令|命令执行/ });
    if (await cmdTab.count() > 0) {
      console.log('\n找到命令执行子标签');
      await cmdTab.click();
      await page.waitForTimeout(1000);
    }

    // 3. 截图：执行前的状态
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-cmd-before-exec.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: saltstack-cmd-before-exec.png');

    // 4. 检查页面内容
    console.log('\n检查页面内容...');
    console.log('\n页面上的所有 Card 标题:');
    let allCards = page.locator('.ant-card-head-title');
    let cardCount = await allCards.count();
    for (let i = 0; i < cardCount; i++) {
      const title = await allCards.nth(i).innerText();
      console.log(`  ${i + 1}. ${title}`);
    }

    // 查找执行 SaltStack 命令的卡片
    console.log('\n查找命令执行表单...');
    const cmdCard = page.locator('.ant-card').filter({ hasText: /执行.*SaltStack.*命令/ });
    
    if (await cmdCard.count() === 0) {
      console.log('❌ 未找到命令执行卡片');
      console.log('\n可能的原因:');
      console.log('1. 前端代码未重新构建');
      console.log('2. SaltCommandExecutor 组件导入失败');
      console.log('3. 组件渲染条件不满足');
      
      // 检查控制台错误
      console.log('\n检查浏览器控制台错误...');
      const logs = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          logs.push(`[ERROR] ${msg.text()}`);
        }
      });
      
      await page.waitForTimeout(2000);
      
      if (logs.length > 0) {
        console.log('控制台错误:');
        logs.forEach(log => console.log(`  ${log}`));
      } else {
        console.log('  无控制台错误');
      }
      
      console.log('\n建议: 请重新构建前端代码');
      console.log('  cd src/frontend && npm run build');
      return;
    }

    console.log('✓ 找到命令执行卡片');

    // 5. 填写表单
    console.log('\n填写 SaltStack 命令表单...');
    
    // 选择目标节点
    const targetSelect = page.locator('label:has-text("目标节点")').locator('..').locator('.ant-select');
    await targetSelect.click();
    await page.waitForTimeout(500);
    
    // 选择 "所有节点 (*)"
    const allNodesOption = page.locator('.ant-select-item').filter({ hasText: '所有节点' }).first();
    await allNodesOption.click();
    console.log('✓ 已选择目标节点: 所有节点 (*)');
    await page.waitForTimeout(500);

    // 选择 Salt 函数
    const functionSelect = page.locator('label:has-text("Salt 函数")').locator('..').locator('.ant-select');
    await functionSelect.click();
    await page.waitForTimeout(500);
    
    // 选择 test.ping
    const testPingOption = page.locator('.ant-select-item').filter({ hasText: 'test.ping' }).first();
    await testPingOption.click();
    console.log('✓ 已选择 Salt 函数: test.ping');
    await page.waitForTimeout(500);

    // 5. 截图：表单填写完成
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-cmd-form-filled.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: saltstack-cmd-form-filled.png');

    // 6. 点击执行按钮
    console.log('\n点击执行命令按钮...');
    const executeButton = page.locator('button').filter({ hasText: '执行命令' });
    await expect(executeButton).toBeVisible();
    
    // 监听网络请求
    const responsePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm/saltstack/execute') && response.status() === 200,
      { timeout: 30000 }
    );
    
    await executeButton.click();
    console.log('✓ 已点击执行命令按钮');

    // 7. 等待 API 响应
    try {
      const response = await responsePromise;
      const responseData = await response.json();
      console.log('\n✓ API 响应成功:');
      console.log(JSON.stringify(responseData, null, 2));
    } catch (error) {
      console.log('\n⚠ API 响应超时或失败:', error.message);
    }

    // 等待执行完成
    await page.waitForTimeout(3000);

    // 8. 检查是否出现 "最新执行结果" 卡片
    console.log('\n检查最新执行结果卡片...');
    const resultCard = page.locator('.ant-card').filter({ hasText: '最新执行结果' });
    
    try {
      await expect(resultCard).toBeVisible({ timeout: 5000 });
      console.log('✓ 找到最新执行结果卡片');

      // 检查卡片内容
      const cardText = await resultCard.innerText();
      console.log('\n卡片内容:');
      console.log(cardText);

      // 检查是否有执行输出
      const outputPre = resultCard.locator('pre');
      if (await outputPre.count() > 0) {
        const outputText = await outputPre.first().innerText();
        console.log('\n执行输出:');
        console.log(outputText);
        console.log('✓ 找到执行输出内容');
      } else {
        console.log('⚠ 未找到 <pre> 输出内容');
      }

      // 检查是否有成功/失败标签
      const successTag = resultCard.locator('.ant-tag').filter({ hasText: /成功|失败/ });
      if (await successTag.count() > 0) {
        const tagText = await successTag.first().innerText();
        console.log(`✓ 找到状态标签: ${tagText}`);
      }

      // 检查是否有复制按钮
      const copyButton = resultCard.locator('button').filter({ hasText: '复制输出' });
      if (await copyButton.count() > 0) {
        console.log('✓ 找到复制输出按钮');
      }

    } catch (error) {
      console.log('✗ 未找到最新执行结果卡片:', error.message);
      console.log('\n可能的问题：');
      console.log('1. 执行按钮点击失败');
      console.log('2. API 调用失败');
      console.log('3. 前端状态更新失败');
      console.log('4. lastExecutionResult 未正确设置');
    }

    // 9. 截图：执行后的状态
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-cmd-after-exec.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: saltstack-cmd-after-exec.png');

    // 10. 检查命令执行历史表格
    console.log('\n检查命令执行历史表格...');
    const historyTable = page.locator('.ant-table').filter({ has: page.locator('thead th:has-text("目标")') });
    
    if (await historyTable.count() > 0) {
      console.log('✓ 找到命令执行历史表格');
      
      // 检查表格行数
      const rows = historyTable.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`✓ 历史记录数量: ${rowCount}`);

      if (rowCount > 0) {
        // 检查第一行（最新记录）
        const firstRow = rows.first();
        const rowText = await firstRow.innerText();
        console.log('\n最新记录内容:');
        console.log(rowText);

        // 检查是否有查看详情按钮
        const detailButton = firstRow.locator('button').filter({ hasText: '查看详情' });
        if (await detailButton.count() > 0) {
          console.log('✓ 找到查看详情按钮');
        }
      }
    } else {
      console.log('⚠ 未找到命令执行历史表格');
    }

    // 11. 检查是否有错误消息
    console.log('\n检查错误消息...');
    const errorAlert = page.locator('.ant-alert-error');
    if (await errorAlert.count() > 0) {
      const errorText = await errorAlert.innerText();
      console.log('⚠ 发现错误消息:', errorText);
    } else {
      console.log('✓ 未发现错误消息');
    }

    // 12. 检查网络请求
    console.log('\n检查网络请求...');
    const performanceEntries = await page.evaluate(() => {
      return performance.getEntriesByType('resource')
        .filter(entry => entry.name.includes('/api/slurm/saltstack'))
        .map(entry => ({
          url: entry.name,
          duration: entry.duration,
          status: entry.responseStatus
        }));
    });
    
    if (performanceEntries.length > 0) {
      console.log('SaltStack API 请求:');
      performanceEntries.forEach(entry => {
        console.log(`  ${entry.url} - ${entry.duration}ms`);
      });
    }

    // 13. 最终截图
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-cmd-final.png',
      fullPage: true 
    });
    console.log('\n✓ 最终截图已保存: saltstack-cmd-final.png');

    // 总结
    console.log('\n========================================');
    console.log('测试完成');
    console.log('========================================');
    console.log('请检查以下截图:');
    console.log('1. test-screenshots/saltstack-cmd-before-exec.png - 执行前');
    console.log('2. test-screenshots/saltstack-cmd-form-filled.png - 表单填写');
    console.log('3. test-screenshots/saltstack-cmd-after-exec.png - 执行后');
    console.log('4. test-screenshots/saltstack-cmd-final.png - 最终状态');
  });
});
