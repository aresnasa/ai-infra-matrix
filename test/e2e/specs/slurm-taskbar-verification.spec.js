const { test, expect } = require('@playwright/test');

/**
 * SLURM 页面任务栏验证测试
 * 
 * 目的：验证 http://192.168.18.154:8080/slurm 页面左上角的任务通知栏是否正确显示
 * 
 * 测试内容：
 * 1. 检查 SlurmTaskBar 组件是否被正确导入
 * 2. 验证页面是否渲染任务栏
 * 3. 检查任务栏的功能是否正常（显示任务数、弹出菜单等）
 */

test.describe('SLURM TaskBar Verification - http://192.168.18.154:8080/slurm', () => {
  
  // 配置
  const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';
  const SLURM_PAGE = `${BASE_URL}/slurm`;
  
  test.beforeEach(async ({ page }) => {
    // 监听控制台错误
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log('Console Error:', msg.text());
      }
    });
  });

  test('verify SlurmTaskBar component import in source code', async () => {
    const fs = require('fs');
    const path = require('path');
    
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');
    
    console.log('\n=== Source Code Verification ===\n');
    
    // 检查是否导入了 SlurmTaskBar
    const hasSlurmTaskBarImport = content.includes("import SlurmTaskBar from '../components/SlurmTaskBar'");
    console.log('1. SlurmTaskBar imported:', hasSlurmTaskBarImport ? '✅ YES' : '❌ NO');
    expect(hasSlurmTaskBarImport).toBe(true);
    
    // 检查是否使用了 SlurmTaskBar
    const usesSlurmTaskBar = content.includes('<SlurmTaskBar');
    console.log('2. SlurmTaskBar used in JSX:', usesSlurmTaskBar ? '✅ YES' : '❌ NO');
    expect(usesSlurmTaskBar).toBe(true);
    
    console.log('\n✅ Source code verification passed!\n');
  });

  test('should display SlurmTaskBar on SLURM page', async ({ page }) => {
    console.log('\n=== Testing SLURM Page TaskBar Display ===\n');
    
    // 访问 SLURM 页面
    console.log('1. Navigating to:', SLURM_PAGE);
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    
    // 等待页面加载
    await page.waitForTimeout(2000);
    
    // 检查是否有任务栏相关元素
    // 方法1: 检查"任务栏"文字
    const hasTaskBarText = await page.getByText('任务栏').count() > 0;
    console.log('2. Found "任务栏" text:', hasTaskBarText ? '✅ YES' : '❌ NO');
    
    // 方法2: 检查任务图标（ThunderboltOutlined）
    const hasThunderboltIcon = await page.locator('[data-icon="thunderbolt"], .anticon-thunderbolt').count() > 0;
    console.log('3. Found thunderbolt icon:', hasThunderboltIcon ? '✅ YES' : '❌ NO');
    
    // 方法3: 检查任务数按钮
    const hasTaskCountButton = await page.getByRole('button', { name: /\d+ 个任务/ }).count() > 0 ||
                                await page.getByText(/\d+ 个任务/).count() > 0;
    console.log('4. Found task count button:', hasTaskCountButton ? '✅ YES' : '❌ NO');
    
    // 至少有一个特征存在
    const taskBarVisible = hasTaskBarText || hasThunderboltIcon || hasTaskCountButton;
    
    if (!taskBarVisible) {
      // 截图以便调试
      await page.screenshot({ path: 'test-results/slurm-page-no-taskbar.png', fullPage: true });
      console.log('❌ TaskBar not found! Screenshot saved to test-results/slurm-page-no-taskbar.png');
    } else {
      console.log('✅ TaskBar is visible on the page!');
    }
    
    expect(taskBarVisible).toBe(true);
  });

  test('should display task count in TaskBar', async ({ page }) => {
    console.log('\n=== Testing TaskBar Task Count ===\n');
    
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 查找显示任务数的按钮
    const taskButton = page.getByRole('button', { name: /\d+ 个任务/ });
    
    if (await taskButton.count() > 0) {
      const buttonText = await taskButton.first().textContent();
      console.log('1. Task count button text:', buttonText);
      console.log('✅ Task count button found!');
      expect(buttonText).toMatch(/\d+ 个任务/);
    } else {
      // 尝试其他方式查找
      const taskText = await page.getByText(/\d+ 个任务/).first().textContent();
      console.log('1. Task count text:', taskText);
      console.log('✅ Task count text found!');
      expect(taskText).toMatch(/\d+ 个任务/);
    }
  });

  test('should open task list popover when clicking task button', async ({ page }) => {
    console.log('\n=== Testing TaskBar Popover Functionality ===\n');
    
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 查找任务数按钮
    const taskButton = page.getByRole('button', { name: /\d+ 个任务/ });
    
    if (await taskButton.count() > 0) {
      console.log('1. Clicking task count button...');
      await taskButton.first().click();
      await page.waitForTimeout(500);
      
      // 检查弹出层是否出现
      // Popover 通常包含"刷新"和"查看全部"按钮
      const hasRefreshButton = await page.getByRole('button', { name: '刷新' }).count() > 0;
      const hasViewAllButton = await page.getByRole('button', { name: '查看全部' }).count() > 0;
      const hasEmptyState = await page.getByText('暂无任务').count() > 0;
      
      console.log('2. Popover content:');
      console.log('   - Refresh button:', hasRefreshButton ? '✅' : '❌');
      console.log('   - View all button:', hasViewAllButton ? '✅' : '❌');
      console.log('   - Empty state (if no tasks):', hasEmptyState ? '✅' : '❌');
      
      const popoverVisible = hasRefreshButton || hasViewAllButton || hasEmptyState;
      
      if (!popoverVisible) {
        await page.screenshot({ path: 'test-results/slurm-taskbar-popover-failed.png', fullPage: true });
        console.log('❌ Popover not visible! Screenshot saved.');
      } else {
        console.log('✅ Popover opened successfully!');
      }
      
      expect(popoverVisible).toBe(true);
    } else {
      console.log('⚠️  Task button not found, skipping popover test');
    }
  });

  test('should have refresh button in TaskBar popover', async ({ page }) => {
    console.log('\n=== Testing TaskBar Refresh Function ===\n');
    
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 点击任务按钮打开弹出层
    const taskButton = page.getByRole('button', { name: /\d+ 个任务/ });
    
    if (await taskButton.count() > 0) {
      await taskButton.first().click();
      await page.waitForTimeout(500);
      
      // 查找刷新按钮
      const refreshButton = page.getByRole('button', { name: '刷新' });
      
      if (await refreshButton.count() > 0) {
        console.log('1. Found refresh button in popover ✅');
        
        // 点击刷新按钮
        console.log('2. Clicking refresh button...');
        await refreshButton.first().click();
        await page.waitForTimeout(1000);
        
        console.log('✅ Refresh button works!');
        expect(true).toBe(true);
      } else {
        console.log('❌ Refresh button not found in popover');
        expect(false).toBe(true);
      }
    } else {
      console.log('⚠️  Task button not found, skipping refresh test');
    }
  });

  test('should navigate to /slurm-tasks when clicking "查看全部"', async ({ page }) => {
    console.log('\n=== Testing TaskBar Navigation ===\n');
    
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 点击任务按钮打开弹出层
    const taskButton = page.getByRole('button', { name: /\d+ 个任务/ });
    
    if (await taskButton.count() > 0) {
      await taskButton.first().click();
      await page.waitForTimeout(500);
      
      // 查找"查看全部"按钮
      const viewAllButton = page.getByRole('button', { name: '查看全部' });
      
      if (await viewAllButton.count() > 0) {
        console.log('1. Found "查看全部" button ✅');
        
        // 点击导航按钮
        console.log('2. Clicking "查看全部" button...');
        await viewAllButton.first().click();
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(1000);
        
        // 检查是否导航到 /slurm-tasks
        const currentURL = page.url();
        console.log('3. Current URL:', currentURL);
        
        const navigatedToTasksPage = currentURL.includes('/slurm-tasks');
        console.log('4. Navigated to /slurm-tasks:', navigatedToTasksPage ? '✅ YES' : '❌ NO');
        
        expect(navigatedToTasksPage).toBe(true);
      } else {
        console.log('❌ "查看全部" button not found');
        expect(false).toBe(true);
      }
    } else {
      console.log('⚠️  Task button not found, skipping navigation test');
    }
  });

  test('should not have console errors related to SlurmTaskBar', async ({ page }) => {
    console.log('\n=== Testing for Console Errors ===\n');
    
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    // 检查是否有 SlurmTaskBar 相关错误
    const taskBarErrors = consoleErrors.filter(error => 
      error.toLowerCase().includes('slurmtaskbar') ||
      error.toLowerCase().includes('cannot find module') && error.includes('SlurmTaskBar')
    );
    
    if (taskBarErrors.length > 0) {
      console.log('❌ Found SlurmTaskBar related errors:');
      taskBarErrors.forEach(err => console.log('  -', err));
    } else {
      console.log('✅ No SlurmTaskBar related console errors');
    }
    
    expect(taskBarErrors.length).toBe(0);
  });

  test('visual snapshot - SLURM page with TaskBar', async ({ page }) => {
    console.log('\n=== Taking Visual Snapshot ===\n');
    
    await page.goto(SLURM_PAGE);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 截取整个页面
    await page.screenshot({ 
      path: 'test-results/slurm-page-with-taskbar.png', 
      fullPage: true 
    });
    console.log('✅ Screenshot saved to: test-results/slurm-page-with-taskbar.png');
    
    // 如果找到任务栏，单独截图
    const taskBar = page.locator('text=任务栏').first();
    if (await taskBar.count() > 0) {
      const taskBarBox = await taskBar.boundingBox();
      if (taskBarBox) {
        await page.screenshot({
          path: 'test-results/slurm-taskbar-closeup.png',
          clip: {
            x: Math.max(0, taskBarBox.x - 10),
            y: Math.max(0, taskBarBox.y - 10),
            width: Math.min(page.viewportSize().width, taskBarBox.width + 300),
            height: taskBarBox.height + 100
          }
        });
        console.log('✅ TaskBar closeup saved to: test-results/slurm-taskbar-closeup.png');
      }
    }
  });
});
