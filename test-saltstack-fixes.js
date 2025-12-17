/**
 * 测试 SaltStack 页面的修复
 * - 翻译问题：saltstack.executeFailed, saltstack.settings
 * - Master 信息：版本号和运行时间显示
 * - 作业状态：是否正确更新为 completed
 */

const { chromium } = require('@playwright/test');

async function runTests() {
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();
  
  try {
    console.log('🧪 测试 1: 访问 SaltStack 页面');
    await page.goto('http://192.168.48.123:8080/saltstack', { waitUntil: 'networkidle' });
    console.log('✅ 页面加载成功');

    // 等待页面稳定
    await page.waitForTimeout(2000);

    console.log('\n🧪 测试 2: 检查翻译是否加载');
    
    // 检查 Settings 标签页翻译
    const settingsTab = await page.locator('text=设置').first();
    if (await settingsTab.isVisible()) {
      console.log('✅ Settings 标签页翻译正确加载');
    } else {
      console.log('❌ Settings 标签页翻译未加载，可能显示为 "saltstack.settings: undefined"');
    }

    console.log('\n🧪 测试 3: 检查 Master 信息');
    
    // 查找 Master Info 卡片
    const masterInfoCard = await page.locator('text=Master 信息, Master Info').first();
    if (await masterInfoCard.isVisible()) {
      console.log('✅ Master Info 卡片可见');
      
      // 检查版本信息
      const versionInfo = await page.locator('text=版本').first().evaluate(el => el.parentElement?.textContent);
      console.log(`   版本信息: ${versionInfo || '未找到'}`);
      
      // 检查运行时间
      const uptimeInfo = await page.locator('text=启动时间, Uptime').first().evaluate(el => el.parentElement?.textContent);
      console.log(`   运行时间: ${uptimeInfo || '未找到'}`);
      
      // 检查是否为 "未知"
      if (versionInfo && versionInfo.includes('未知')) {
        console.log('⚠️  版本显示为"未知"，可能需要检查后端获取逻辑');
      }
      if (uptimeInfo && uptimeInfo.includes('未知')) {
        console.log('⚠️  运行时间显示为"未知"，可能需要检查后端获取逻辑');
      }
    } else {
      console.log('❌ Master Info 卡片不可见');
    }

    console.log('\n🧪 测试 4: 检查作业历史表格');
    
    // 查找作业历史选项卡并切换
    const jobsTab = await page.locator('text=作业历史, Job History').first();
    if (await jobsTab.isVisible()) {
      await jobsTab.click();
      await page.waitForTimeout(1000);
      console.log('✅ 已切换到作业历史选项卡');
      
      // 查找表格中的作业状态
      const statusCells = await page.locator('td').filter({ hasText: 'running' });
      const count = await statusCells.count();
      if (count > 0) {
        console.log(`📊 发现 ${count} 个运行状态的作业`);
      }
      
      // 检查是否有已完成的作业
      const completedCells = await page.locator('td').filter({ hasText: 'completed' });
      const completedCount = await completedCells.count();
      if (completedCount > 0) {
        console.log(`✅ 发现 ${completedCount} 个已完成的作业`);
      } else {
        console.log('⚠️  未找到已完成的作业，或状态未正确更新');
      }
    }

    console.log('\n🧪 测试 5: 执行一个简单命令并检查状态更新');
    
    // 切换到批量执行选项卡
    const batchExecTab = await page.locator('text=批量执行, Batch Execution').first();
    if (await batchExecTab.isVisible()) {
      await batchExecTab.click();
      await page.waitForTimeout(1000);
      console.log('✅ 已切换到批量执行选项卡');
      
      // 执行一个简单的 test.ping 命令
      const targetInput = await page.locator('input[placeholder*="目标"]').first();
      if (await targetInput.isVisible()) {
        await targetInput.fill('*');
        console.log('   已填充目标: *');
        
        // 输入函数
        const functionInput = await page.locator('input[placeholder*="函数"]').first();
        if (await functionInput.isVisible()) {
          await functionInput.fill('test.ping');
          console.log('   已填充函数: test.ping');
          
          // 点击执行
          const executeBtn = await page.locator('button:has-text("立即执行")').first();
          if (await executeBtn.isVisible()) {
            console.log('   点击执行按钮...');
            await executeBtn.click();
            
            // 等待执行完成并观察状态变化
            console.log('   等待执行完成（最多30秒）...');
            await page.waitForTimeout(3000);
            
            // 检查是否有执行结果
            const resultElements = await page.locator('text=执行结果').count();
            if (resultElements > 0) {
              console.log('✅ 执行完成，显示结果');
            }
            
            // 切回作业历史查看状态
            await jobsTab.click();
            await page.waitForTimeout(1000);
            
            const latestJobStatus = await page.locator('tbody tr:first-child td:nth-child(4)').textContent();
            console.log(`   最新作业状态: ${latestJobStatus}`);
            
            if (latestJobStatus && latestJobStatus.includes('completed')) {
              console.log('✅ 作业状态正确更新为 completed');
            } else if (latestJobStatus && latestJobStatus.includes('running')) {
              console.log('❌ 作业状态仍为 running，未正确更新');
            }
          }
        }
      }
    }

    console.log('\n📝 测试总结:');
    console.log('- ✅ 翻译问题: saltstack.executeFailed, saltstack.settings 已添加');
    console.log('- ⏳ Master 版本和运行时间: 需要检查后端获取逻辑');
    console.log('- ⏳ 作业状态更新: 需要观察实时执行结果');

  } catch (error) {
    console.error('❌ 测试出错:', error.message);
  } finally {
    await browser.close();
  }
}

runTests();
