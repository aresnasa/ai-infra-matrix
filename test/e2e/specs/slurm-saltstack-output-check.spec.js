const { test, expect } = require('@playwright/test');

test.describe('SLURM SaltStack 命令输出检查', () => {
  test('检查 SaltStack 命令执行和输出显示', async ({ page }) => {
    console.log('========================================');
    console.log('开始测试 SaltStack 命令执行输出功能');
    console.log('========================================\n');

    // 1. 访问主页并登录（如果需要）
    await page.goto('http://192.168.3.91:8080');
    console.log('✓ 访问主页');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 2. 查找并点击 SLURM 菜单项
    console.log('\n查找 SLURM 菜单项...');
    
    // 尝试多种定位方式
    let slurmMenu = page.locator('menuitem').filter({ hasText: 'SLURM' });
    if (await slurmMenu.count() === 0) {
      slurmMenu = page.locator('.ant-menu-item').filter({ hasText: 'SLURM' });
    }
    if (await slurmMenu.count() === 0) {
      slurmMenu = page.locator('a[href*="/slurm"]');
    }
    
    const menuCount = await slurmMenu.count();
    console.log(`找到 ${menuCount} 个 SLURM 菜单项`);
    
    if (menuCount > 0) {
      await slurmMenu.first().click();
      console.log('✓ 已点击 SLURM 菜单项');
      await page.waitForTimeout(2000);
      await page.waitForLoadState('networkidle');
    } else {
      console.log('⚠ 未找到 SLURM 菜单项，直接访问 URL');
      await page.goto('http://192.168.3.91:8080/slurm');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);
    }

    // 3. 检查当前页面内容
    console.log('\n检查页面内容...');
    const pageTitle = await page.title();
    console.log(`页面标题: ${pageTitle}`);
    
    const url = page.url();
    console.log(`当前 URL: ${url}`);

    // 4. 截图：SLURM 页面初始状态
    await page.screenshot({ 
      path: 'test-screenshots/01-slurm-page-loaded.png',
      fullPage: true 
    });
    console.log('✓ 截图: 01-slurm-page-loaded.png');

    // 5. 查找所有可用的标签页
    console.log('\n查找页面上的所有标签页...');
    const allTabs = page.locator('.ant-tabs-tab');
    const tabCount = await allTabs.count();
    console.log(`找到 ${tabCount} 个标签页:`);
    
    for (let i = 0; i < tabCount; i++) {
      const tabText = await allTabs.nth(i).innerText();
      console.log(`  ${i + 1}. ${tabText}`);
    }

    // 6. 查找 SaltStack 命令执行标签
    console.log('\n查找 SaltStack 命令执行标签...');
    let saltTab = null;
    
    // 尝试多种匹配方式
    const possibleMatches = [
      /SaltStack.*命令执行/,
      /SaltStack.*命令/,
      /命令执行/,
      /SaltStack/
    ];
    
    for (const pattern of possibleMatches) {
      const tab = page.locator('.ant-tabs-tab').filter({ hasText: pattern });
      if (await tab.count() > 0) {
        saltTab = tab;
        console.log(`✓ 使用模式 ${pattern} 找到标签`);
        break;
      }
    }
    
    if (!saltTab || await saltTab.count() === 0) {
      console.log('✗ 未找到 SaltStack 命令执行标签');
      console.log('\n可能的原因:');
      console.log('1. 标签文本与预期不匹配');
      console.log('2. 页面结构已改变');
      console.log('3. 标签被隐藏或需要权限');
      
      // 列出所有 Card 标题
      console.log('\n页面上的所有 Card:');
      const allCards = page.locator('.ant-card-head-title');
      const cardCount = await allCards.count();
      for (let i = 0; i < cardCount; i++) {
        const title = await allCards.nth(i).innerText();
        console.log(`  ${i + 1}. ${title}`);
      }
      
      // 最终截图
      await page.screenshot({ 
        path: 'test-screenshots/02-no-saltstack-tab.png',
        fullPage: true 
      });
      console.log('✓ 截图: 02-no-saltstack-tab.png');
      
      throw new Error('未找到 SaltStack 命令执行标签');
    }

    // 7. 点击 SaltStack 命令执行标签
    await saltTab.first().click();
    console.log('✓ 已点击 SaltStack 命令执行标签');
    await page.waitForTimeout(1000);

    // 8. 截图：切换到 SaltStack 标签后
    await page.screenshot({ 
      path: 'test-screenshots/03-saltstack-tab-active.png',
      fullPage: true 
    });
    console.log('✓ 截图: 03-saltstack-tab-active.png');

    // 9. 查找执行表单
    console.log('\n查找 SaltStack 命令执行表单...');
    
    // 检查表单元素
    const targetLabel = page.locator('label').filter({ hasText: '目标节点' });
    const functionLabel = page.locator('label').filter({ hasText: 'Salt 函数' });
    const executeButton = page.locator('button').filter({ hasText: '执行命令' });
    
    console.log(`目标节点标签: ${await targetLabel.count()} 个`);
    console.log(`Salt 函数标签: ${await functionLabel.count()} 个`);
    console.log(`执行命令按钮: ${await executeButton.count()} 个`);

    if (await executeButton.count() === 0) {
      console.log('✗ 未找到执行命令按钮');
      throw new Error('未找到执行命令按钮');
    }

    // 10. 填写表单
    console.log('\n填写表单...');
    
    // 选择目标节点
    const targetSelect = page.locator('label:has-text("目标节点")').locator('..').locator('.ant-select').first();
    await targetSelect.click();
    console.log('✓ 点击目标节点选择器');
    await page.waitForTimeout(500);
    
    const allNodesOption = page.locator('.ant-select-item').filter({ hasText: /所有节点|\*/ }).first();
    await allNodesOption.click();
    console.log('✓ 选择：所有节点');
    await page.waitForTimeout(500);

    // 选择 Salt 函数
    const functionSelect = page.locator('label:has-text("Salt 函数")').locator('..').locator('.ant-select').first();
    await functionSelect.click();
    console.log('✓ 点击 Salt 函数选择器');
    await page.waitForTimeout(500);
    
    const testPingOption = page.locator('.ant-select-item').filter({ hasText: 'test.ping' }).first();
    await testPingOption.click();
    console.log('✓ 选择：test.ping');
    await page.waitForTimeout(500);

    // 11. 截图：表单填写完成
    await page.screenshot({ 
      path: 'test-screenshots/04-form-filled.png',
      fullPage: true 
    });
    console.log('✓ 截图: 04-form-filled.png');

    // 12. 执行命令
    console.log('\n执行命令...');
    
    // 监听 API 请求
    page.on('response', response => {
      if (response.url().includes('/api/slurm/saltstack/execute')) {
        console.log(`  API 响应: ${response.status()} ${response.url()}`);
      }
    });

    await executeButton.first().click();
    console.log('✓ 已点击执行命令按钮');
    
    // 等待 API 响应
    await page.waitForTimeout(3000);

    // 13. 截图：执行后立即截图
    await page.screenshot({ 
      path: 'test-screenshots/05-after-execute-immediate.png',
      fullPage: true 
    });
    console.log('✓ 截图: 05-after-execute-immediate.png');

    // 14. 检查是否出现 "最新执行结果" 卡片
    console.log('\n检查执行结果...');
    
    const resultCard = page.locator('.ant-card').filter({ hasText: '最新执行结果' });
    const resultCardCount = await resultCard.count();
    console.log(`找到 ${resultCardCount} 个"最新执行结果"卡片`);

    if (resultCardCount > 0) {
      console.log('✓ 找到最新执行结果卡片');
      
      // 获取卡片完整内容
      const cardText = await resultCard.first().innerText();
      console.log('\n卡片内容:');
      console.log('----------------------------------------');
      console.log(cardText);
      console.log('----------------------------------------');

      // 检查状态标签
      const statusTag = resultCard.locator('.ant-tag').first();
      if (await statusTag.count() > 0) {
        const statusText = await statusTag.innerText();
        console.log(`\n执行状态: ${statusText}`);
      }

      // 检查输出内容
      const outputPre = resultCard.locator('pre');
      const outputCount = await outputPre.count();
      console.log(`\n找到 ${outputCount} 个输出区域`);
      
      if (outputCount > 0) {
        const outputText = await outputPre.first().innerText();
        console.log('\n执行输出:');
        console.log('----------------------------------------');
        console.log(outputText);
        console.log('----------------------------------------');
      }

      // 检查按钮
      const copyButton = resultCard.locator('button').filter({ hasText: '复制输出' });
      const closeButton = resultCard.locator('button').filter({ hasText: '关闭' });
      console.log(`\n复制输出按钮: ${await copyButton.count()} 个`);
      console.log(`关闭按钮: ${await closeButton.count()} 个`);

    } else {
      console.log('✗ 未找到最新执行结果卡片');
      console.log('\n检查可能的问题...');
      
      // 检查是否有错误消息
      const errorAlert = page.locator('.ant-alert-error');
      if (await errorAlert.count() > 0) {
        const errorText = await errorAlert.innerText();
        console.log(`✗ 发现错误: ${errorText}`);
      }
      
      // 检查是否有成功消息
      const successMessage = page.locator('.ant-message-success');
      if (await successMessage.count() > 0) {
        const successText = await successMessage.innerText();
        console.log(`✓ 成功消息: ${successText}`);
      }
      
      // 列出所有 Card
      console.log('\n页面上的所有 Card:');
      const allCards = page.locator('.ant-card');
      const cardCount = await allCards.count();
      for (let i = 0; i < cardCount; i++) {
        const card = allCards.nth(i);
        const cardTitle = await card.locator('.ant-card-head-title').innerText().catch(() => '无标题');
        console.log(`  ${i + 1}. ${cardTitle}`);
      }
    }

    // 15. 检查命令执行历史
    console.log('\n检查命令执行历史表格...');
    const historyTable = page.locator('.ant-table').filter({ 
      has: page.locator('thead th').filter({ hasText: /目标|时间|函数/ }) 
    });
    
    const historyTableCount = await historyTable.count();
    console.log(`找到 ${historyTableCount} 个历史表格`);
    
    if (historyTableCount > 0) {
      const rows = historyTable.first().locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`历史记录数量: ${rowCount} 条`);
      
      if (rowCount > 0) {
        const firstRowText = await rows.first().innerText();
        console.log('\n最新记录:');
        console.log(firstRowText);
      }
    }

    // 16. 最终截图
    await page.screenshot({ 
      path: 'test-screenshots/06-final-state.png',
      fullPage: true 
    });
    console.log('\n✓ 截图: 06-final-state.png');

    // 17. 检查浏览器控制台错误
    console.log('\n检查浏览器控制台...');
    const consoleLogs = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleLogs.push(`[ERROR] ${msg.text()}`);
      }
    });
    
    if (consoleLogs.length > 0) {
      console.log('控制台错误:');
      consoleLogs.forEach(log => console.log(`  ${log}`));
    } else {
      console.log('✓ 无控制台错误');
    }

    // 总结
    console.log('\n========================================');
    console.log('测试完成');
    console.log('========================================');
    console.log('\n截图文件:');
    console.log('1. 01-slurm-page-loaded.png - SLURM 页面加载');
    console.log('2. 03-saltstack-tab-active.png - SaltStack 标签激活');
    console.log('3. 04-form-filled.png - 表单填写完成');
    console.log('4. 05-after-execute-immediate.png - 执行后立即状态');
    console.log('5. 06-final-state.png - 最终状态');
    console.log('\n请检查截图以诊断问题。');
  });
});
