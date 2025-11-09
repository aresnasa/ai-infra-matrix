const { test, expect } = require('@playwright/test');

/**
 * SaltStack 命令执行、历史记录和作业显示测试
 * 测试目标：
 * 1. 执行 SaltStack 命令（test.ping, cmd.run 等）
 * 2. 验证命令执行历史正确显示
 * 3. 验证最近的 SaltStack 作业列表
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('SaltStack 命令执行和历史记录', () => {
  let page;
  let context;

  test.beforeAll(async ({ browser }) => {
    context = await browser.newContext();
    page = await context.newPage();
    
    // 登录
    console.log('\n=== 登录系统 ===');
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    const usernameInput = page.locator('input[name="username"]').or(page.locator('input[placeholder*="用户名"]'));
    await usernameInput.waitFor({ timeout: 5000 });
    await usernameInput.fill(TEST_USERNAME);
    
    const passwordInput = page.locator('input[name="password"]').or(page.locator('input[type="password"]'));
    await passwordInput.fill(TEST_PASSWORD);
    
    const loginButton = page.locator('button[type="submit"]').or(page.locator('button:has-text("登录")'));
    await loginButton.click();
    await page.waitForTimeout(2000);
    
    console.log('✅ 登录成功');
  });

  test.afterAll(async () => {
    await context.close();
  });

  test('完整测试：命令执行 → 历史记录 → 作业列表', async () => {
    console.log('\n=== 步骤 1: 访问 SaltStack 集成页面 ===');
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 点击 SaltStack 集成标签
    const saltStackTab = page.locator('.ant-tabs-tab').filter({ hasText: /SaltStack.*集成/ });
    await saltStackTab.waitFor({ timeout: 5000 });
    console.log('✅ 找到 SaltStack 集成标签');
    await saltStackTab.click();
    await page.waitForTimeout(2000);
    
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-integration-page.png',
      fullPage: true 
    });
    console.log('📸 截图: saltstack-integration-page.png');

    console.log('\n=== 步骤 2: 测试命令执行功能 ===');
    
    // 查找命令执行区域
    const commandSection = page.locator('text=命令执行').or(page.locator('text=SaltStack 命令执行'));
    const commandSectionVisible = await commandSection.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (!commandSectionVisible) {
      console.log('⚠️  未找到命令执行区域，检查页面结构...');
      const pageText = await page.locator('body').innerText();
      console.log('页面包含的关键字:');
      if (pageText.includes('SaltStack')) console.log('  - SaltStack ✓');
      if (pageText.includes('命令')) console.log('  - 命令 ✓');
      if (pageText.includes('执行')) console.log('  - 执行 ✓');
      if (pageText.includes('历史')) console.log('  - 历史 ✓');
    }

    // 测试 test.ping 命令
    console.log('\n--- 测试 1: test.ping 命令 ---');
    await executeCommand(page, 'test.ping', '*', '测试所有 minions 连接');
    
    // 测试 cmd.run 命令
    console.log('\n--- 测试 2: cmd.run 命令 ---');
    await executeCommand(page, 'cmd.run', '*', '执行 hostname 命令', 'hostname');
    
    // 测试 grains.item 命令
    console.log('\n--- 测试 3: grains.item 命令 ---');
    await executeCommand(page, 'grains.item', '*', '获取系统信息', 'os,osrelease');

    console.log('\n=== 步骤 3: 验证命令执行历史 ===');
    await page.waitForTimeout(2000);
    
    // 关闭最新执行结果卡片以便查看历史记录
    const closeButton = page.locator('button:has-text("关闭")');
    if (await closeButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await closeButton.click();
      await page.waitForTimeout(500);
      console.log('✅ 关闭最新执行结果卡片');
    }
    
    // 查找命令执行历史区域（通过标题和图标查找）
    const historyCard = page.locator('.ant-card').filter({ 
      has: page.locator('span:has-text("命令执行历史")') 
    });
    const historyCardVisible = await historyCard.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (historyCardVisible) {
      console.log('✅ 找到命令执行历史卡片');
      
      // 检查历史记录数量标签
      const recordTag = historyCard.locator('.ant-tag').filter({ hasText: /\d+.*条记录/ });
      if (await recordTag.isVisible({ timeout: 2000 }).catch(() => false)) {
        const tagText = await recordTag.innerText();
        console.log(`📊 ${tagText}`);
      }
      
      // 检查历史记录表格
      const historyTable = historyCard.locator('.ant-table');
      const historyTableVisible = await historyTable.isVisible({ timeout: 3000 }).catch(() => false);
      
      if (historyTableVisible) {
        const rows = historyTable.locator('tbody tr');
        const rowCount = await rows.count();
        console.log(`📊 历史记录表格: ${rowCount} 行`);
        
        if (rowCount > 0) {
          console.log(`\n最近 ${Math.min(rowCount, 3)} 条历史记录:`);
          
          for (let i = 0; i < Math.min(rowCount, 3); i++) {
            const row = rows.nth(i);
            const cells = row.locator('td');
            const cellCount = await cells.count();
            
            // 提取关键信息：时间、目标、函数、参数、耗时、状态
            const rowInfo = [];
            for (let j = 0; j < Math.min(cellCount, 6); j++) {
              const cellText = await cells.nth(j).innerText();
              rowInfo.push(cellText.trim());
            }
            
            console.log(`  ${i + 1}. ${rowInfo.slice(0, 4).join(' | ')}`); // 显示前4列
          }
          
          // 验证历史记录包含我们执行的命令
          const tableText = await historyTable.innerText();
          const hasTestPing = tableText.includes('test.ping');
          const hasCmdRun = tableText.includes('cmd.run');
          const hasGrains = tableText.includes('grains');
          
          console.log('\n✅ 验证命令历史:');
          console.log(`  ${hasTestPing ? '✅' : '❌'} test.ping 命令记录`);
          console.log(`  ${hasCmdRun ? '✅' : '❌'} cmd.run 命令记录`);
          console.log(`  ${hasGrains ? '✅' : '❌'} grains 命令记录`);
          
          if (hasTestPing && hasCmdRun && hasGrains) {
            console.log('\n✅✅✅ 所有执行的命令都已记录在历史中');
          } else {
            console.log('\n⚠️  部分命令未在历史中找到');
          }
        } else {
          console.log('⚠️  历史记录表格为空');
        }
      } else {
        // 检查是否显示"暂无执行记录"的提示
        const noRecordAlert = historyCard.locator('.ant-alert').filter({ hasText: /暂无执行记录/ });
        if (await noRecordAlert.isVisible({ timeout: 2000 }).catch(() => false)) {
          console.log('⚠️  显示"暂无执行记录"提示');
        } else {
          console.log('⚠️  未找到历史记录表格或提示信息');
        }
      }
      
      await page.screenshot({ 
        path: 'test-screenshots/saltstack-command-history.png',
        fullPage: true 
      });
      console.log('📸 截图: saltstack-command-history.png');
    } else {
      console.log('❌ 未找到命令执行历史卡片');
    }

    console.log('\n=== 步骤 4: 验证最近的 SaltStack 作业 ===');
    
    // 查找作业列表卡片
    const jobsCard = page.locator('.ant-card').filter({ 
      has: page.locator('span:has-text("最近的 SaltStack 作业")') 
    });
    const jobsCardVisible = await jobsCard.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (jobsCardVisible) {
      console.log('✅ 找到 SaltStack 作业卡片');
      
      // 检查作业数量标签
      const jobTag = jobsCard.locator('.ant-tag').filter({ hasText: /\d+.*个作业/ });
      if (await jobTag.isVisible({ timeout: 2000 }).catch(() => false)) {
        const tagText = await jobTag.innerText();
        console.log(`📊 ${tagText}`);
      }
      
      // 检查是否正在加载
      const loadingSpin = jobsCard.locator('.ant-spin');
      if (await loadingSpin.isVisible({ timeout: 2000 }).catch(() => false)) {
        console.log('⏳ 作业列表加载中...');
        await page.waitForTimeout(3000);
      }
      
      // 检查作业列表（折叠面板格式）
      const jobsCollapse = jobsCard.locator('.ant-collapse');
      const jobsCollapseVisible = await jobsCollapse.isVisible({ timeout: 3000 }).catch(() => false);
      
      if (jobsCollapseVisible) {
        const jobPanels = jobsCollapse.locator('.ant-collapse-item');
        const panelCount = await jobPanels.count();
        console.log(`📊 作业列表: ${panelCount} 个作业`);
        
        if (panelCount > 0) {
          console.log(`\n最近 ${Math.min(panelCount, 3)} 个作业:`);
          
          for (let i = 0; i < Math.min(panelCount, 3); i++) {
            const panel = jobPanels.nth(i);
            const header = panel.locator('.ant-collapse-header');
            const headerText = await header.innerText();
            console.log(`  ${i + 1}. ${headerText.replace(/\n/g, ' ')}`);
            
            // 展开面板查看详情
            if (i === 0) {
              await header.click();
              await page.waitForTimeout(500);
              
              const content = panel.locator('.ant-collapse-content');
              if (await content.isVisible({ timeout: 2000 }).catch(() => false)) {
                const contentText = await content.innerText();
                const lines = contentText.split('\n').slice(0, 6); // 只显示前6行
                console.log(`    详情:`);
                lines.forEach(line => console.log(`      ${line}`));
              }
            }
          }
          
          console.log('\n✅ SaltStack 作业列表正确显示');
        } else {
          console.log('⚠️  作业列表为空');
        }
      } else {
        // 检查是否显示"暂无作业记录"的提示
        const noJobAlert = jobsCard.locator('.ant-alert').filter({ hasText: /暂无作业记录/ });
        if (await noJobAlert.isVisible({ timeout: 2000 }).catch(() => false)) {
          console.log('ℹ️  显示"暂无作业记录"提示');
          const alertText = await noJobAlert.innerText();
          console.log(`   ${alertText.replace(/\n/g, ' ')}`);
        } else {
          console.log('⚠️  未找到作业列表或提示信息');
        }
      }
      
      await page.screenshot({ 
        path: 'test-screenshots/saltstack-jobs-list.png',
        fullPage: true 
      });
      console.log('📸 截图: saltstack-jobs-list.png');
    } else {
      console.log('❌ 未找到 SaltStack 作业卡片');
    }

    console.log('\n=== 步骤 5: 通过 API 直接验证 ===');
    
    // 获取 token
    const cookies = await context.cookies();
    const tokenCookie = cookies.find(c => c.name === 'token');
    const token = tokenCookie?.value;
    
    if (token) {
      // 验证命令执行历史 API
      try {
        const historyResponse = await page.request.get(`${BASE_URL}/api/slurm/saltstack/history`, {
          headers: { 'Authorization': `Bearer ${token}` }
        });
        
        if (historyResponse.ok()) {
          const historyData = await historyResponse.json();
          const history = historyData.data || historyData;
          
          if (Array.isArray(history)) {
            console.log(`\n✅ API 返回 ${history.length} 条命令历史`);
            
            if (history.length > 0) {
              console.log('\nAPI 返回的最近 3 条历史:');
              history.slice(0, 3).forEach((item, index) => {
                console.log(`  ${index + 1}. ${item.function || item.fun || 'N/A'} - ${item.target || item.tgt || 'N/A'} - ${item.status || 'N/A'}`);
              });
            }
          } else {
            console.log('⚠️  API 返回的历史格式异常');
          }
        } else {
          console.log(`⚠️  命令历史 API 返回错误: ${historyResponse.status()}`);
        }
      } catch (error) {
        console.log(`⚠️  无法访问命令历史 API: ${error.message}`);
      }
      
      // 验证作业列表 API
      try {
        const jobsResponse = await page.request.get(`${BASE_URL}/api/slurm/saltstack/jobs`, {
          headers: { 'Authorization': `Bearer ${token}` }
        });
        
        if (jobsResponse.ok()) {
          const jobsData = await jobsResponse.json();
          const jobs = jobsData.data || jobsData;
          
          if (Array.isArray(jobs)) {
            console.log(`\n✅ API 返回 ${jobs.length} 个作业`);
            
            if (jobs.length > 0) {
              console.log('\nAPI 返回的最近 3 个作业:');
              jobs.slice(0, 3).forEach((job, index) => {
                console.log(`  ${index + 1}. JID: ${job.jid || job.id || 'N/A'} - ${job.function || job.fun || 'N/A'}`);
              });
            }
          } else if (typeof jobs === 'object') {
            const jobKeys = Object.keys(jobs);
            console.log(`\n✅ API 返回 ${jobKeys.length} 个作业 (对象格式)`);
            
            if (jobKeys.length > 0) {
              console.log('\nAPI 返回的最近 3 个作业:');
              jobKeys.slice(0, 3).forEach((jid, index) => {
                const job = jobs[jid];
                console.log(`  ${index + 1}. JID: ${jid} - ${job.Function || job.fun || 'N/A'}`);
              });
            }
          } else {
            console.log('⚠️  API 返回的作业格式异常');
          }
        } else {
          console.log(`⚠️  作业列表 API 返回错误: ${jobsResponse.status()}`);
        }
      } catch (error) {
        console.log(`⚠️  无法访问作业列表 API: ${error.message}`);
      }
    } else {
      console.log('⚠️  未找到认证 token，跳过 API 验证');
    }

    console.log('\n=== 测试总结 ===');
    console.log('✅ 测试完成，请查看截图和日志验证结果');
  });
});

/**
 * 执行 SaltStack 命令的辅助函数
 */
async function executeCommand(page, functionName, target, description, args = '') {
  console.log(`\n执行命令: ${functionName} on ${target}`);
  console.log(`描述: ${description}`);
  
  try {
    // 等待命令执行区域加载
    await page.waitForTimeout(1000);
    
    // 1. 选择目标节点（使用下拉选择）
    const targetSelector = page.locator('.ant-select').filter({ has: page.locator('input[id*="target"]') });
    const targetSelectorVisible = await targetSelector.isVisible({ timeout: 5000 }).catch(() => false);
    
    if (targetSelectorVisible) {
      await targetSelector.click();
      await page.waitForTimeout(500);
      
      // 根据 target 参数选择对应选项
      let targetOption;
      if (target === '*') {
        targetOption = page.locator('.ant-select-item').filter({ hasText: '所有节点' });
      } else if (target.includes('ssh')) {
        targetOption = page.locator('.ant-select-item').filter({ hasText: '测试节点' });
      } else if (target.includes('rocky')) {
        targetOption = page.locator('.ant-select-item').filter({ hasText: 'Rocky节点' });
      } else {
        targetOption = page.locator('.ant-select-item').first();
      }
      
      await targetOption.click();
      console.log(`  ✓ 选择目标: ${target}`);
    } else {
      console.log('  ⚠️  未找到目标节点选择器');
    }
    
    // 2. 选择 Salt 函数（使用下拉选择）
    const functionSelector = page.locator('.ant-select').filter({ has: page.locator('input[id*="function"]') });
    const functionSelectorVisible = await functionSelector.isVisible({ timeout: 5000 }).catch(() => false);
    
    if (functionSelectorVisible) {
      await functionSelector.click();
      await page.waitForTimeout(500);
      
      // 在下拉列表中查找对应的函数
      const functionOption = page.locator('.ant-select-item').filter({ hasText: functionName });
      const optionExists = await functionOption.count() > 0;
      
      if (optionExists) {
        await functionOption.first().click();
        console.log(`  ✓ 选择函数: ${functionName}`);
      } else {
        // 如果没有预定义选项，尝试手动输入
        const functionInput = page.locator('input[id*="function"]');
        await functionInput.fill(functionName);
        await page.keyboard.press('Enter');
        console.log(`  ✓ 手动输入函数: ${functionName}`);
      }
    } else {
      console.log('  ⚠️  未找到函数选择器');
      return;
    }
    
    // 3. 填写参数（如果有）
    if (args) {
      const argsTextarea = page.locator('textarea[id*="arguments"]');
      const argsTextareaVisible = await argsTextarea.isVisible({ timeout: 3000 }).catch(() => false);
      
      if (argsTextareaVisible) {
        await argsTextarea.fill(args);
        console.log(`  ✓ 填写参数: ${args}`);
      }
    }
    
    // 4. 点击执行按钮
    const executeButton = page.locator('button:has-text("执行命令")');
    await executeButton.click();
    console.log('  ✓ 点击执行按钮');
    
    // 5. 等待命令执行完成（查找成功消息）
    const successMessage = page.locator('.ant-message-success');
    const messageAppeared = await successMessage.isVisible({ timeout: 10000 }).catch(() => false);
    
    if (messageAppeared) {
      console.log('  ✅ 命令执行成功');
    } else {
      console.log('  ⚠️  未检测到成功消息');
    }
    
    // 等待结果显示
    await page.waitForTimeout(2000);
    
    // 6. 查找最新执行结果卡片
    const resultCard = page.locator('.ant-card').filter({ hasText: /最新执行结果|执行结果/ });
    const resultCardVisible = await resultCard.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (resultCardVisible) {
      const resultText = await resultCard.innerText();
      const resultLines = resultText.split('\n').slice(0, 10); // 只显示前10行
      console.log(`  ✅ 执行结果预览:`);
      resultLines.forEach(line => console.log(`    ${line}`));
    } else {
      console.log('  ℹ️  未找到执行结果卡片（可能还未渲染）');
    }
    
    await page.screenshot({ 
      path: `test-screenshots/command-${functionName.replace(/\./g, '-')}.png`,
      fullPage: true 
    });
    console.log(`  📸 截图: command-${functionName.replace(/\./g, '-')}.png`);
    
  } catch (error) {
    console.log(`  ❌ 执行失败: ${error.message}`);
    await page.screenshot({ 
      path: `test-screenshots/command-${functionName.replace(/\./g, '-')}-error.png`,
      fullPage: true 
    });
  }
}
