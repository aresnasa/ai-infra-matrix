const { test, expect } = require('@playwright/test');

/**
 * SLURM 左上角任务框测试
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';

const TEST_USER = {
  username: 'admin',
  password: 'admin123'
};

test.describe('SLURM 任务框位置检查', () => {
  test('检查左上角固定任务框', async ({ page }) => {
    console.log('========================================');
    console.log('步骤 1: 登录');
    console.log('========================================');
    
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
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
    await page.waitForTimeout(2000);
    
    console.log('========================================');
    console.log('步骤 3: 检查固定任务框');
    console.log('========================================');
    
    // 查找包含"任务栏"文本的元素
    const taskBarLocator = page.locator('text=任务栏').first();
    const taskBarVisible = await taskBarLocator.isVisible({ timeout: 5000 }).catch(() => false);
    
    console.log('任务栏是否可见:', taskBarVisible);
    
    if (!taskBarVisible) {
      console.log('✗ 未找到任务栏！');
      await page.screenshot({ path: 'test-screenshots/slurm-taskbar-missing.png', fullPage: true });
      throw new Error('任务栏不可见');
    }
    
    // 获取任务框的位置和样式
    const taskBarContainer = taskBarLocator.locator('..').locator('..');
    const boundingBox = await taskBarContainer.boundingBox();
    
    console.log('任务框位置信息:');
    if (boundingBox) {
      console.log(`  X: ${Math.round(boundingBox.x)}`);
      console.log(`  Y: ${Math.round(boundingBox.y)}`);
      console.log(`  宽度: ${Math.round(boundingBox.width)}`);
      console.log(`  高度: ${Math.round(boundingBox.height)}`);
    }
    
    // 检查是否是固定定位
    const position = await page.evaluate(() => {
      const taskBar = document.querySelector('div[style*="position: fixed"]');
      if (taskBar) {
        const styles = window.getComputedStyle(taskBar);
        return {
          position: styles.position,
          top: styles.top,
          right: styles.right,
          zIndex: styles.zIndex,
          hasBoxShadow: styles.boxShadow !== 'none'
        };
      }
      return null;
    });
    
    console.log('CSS 样式信息:');
    if (position) {
      console.log(`  position: ${position.position}`);
      console.log(`  top: ${position.top}`);
      console.log(`  right: ${position.right}`);
      console.log(`  z-index: ${position.zIndex}`);
      console.log(`  有阴影: ${position.hasBoxShadow}`);
      
      expect(position.position).toBe('fixed');
      console.log('✓ 任务框使用 fixed 定位');
    } else {
      console.log('✗ 未找到 fixed 定位的任务框');
    }
    
    console.log('========================================');
    console.log('步骤 4: 测试任务框功能');
    console.log('========================================');
    
    // 查找显示任务数量的按钮
    const taskButton = page.locator('button:has-text("个任务")').first();
    const taskButtonVisible = await taskButton.isVisible().catch(() => false);
    
    if (taskButtonVisible) {
      console.log('✓ 找到任务数量按钮');
      const taskText = await taskButton.textContent();
      console.log(`  任务数量: ${taskText}`);
      
      // 点击打开任务列表
      await taskButton.click();
      await page.waitForTimeout(500);
      
      // 检查是否弹出 Popover
      const popover = page.locator('.ant-popover').first();
      const popoverVisible = await popover.isVisible().catch(() => false);
      
      if (popoverVisible) {
        console.log('✓ 任务列表弹窗已打开');
        
        // 检查刷新和查看全部按钮
        const refreshBtn = popover.locator('button:has-text("刷新")');
        const viewAllBtn = popover.locator('button:has-text("查看全部")');
        
        console.log('  刷新按钮:', await refreshBtn.isVisible() ? '✓ 可见' : '✗ 不可见');
        console.log('  查看全部按钮:', await viewAllBtn.isVisible() ? '✓ 可见' : '✗ 不可见');
      } else {
        console.log('⚠ 任务列表弹窗未打开');
      }
      
      // 关闭弹窗
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    } else {
      console.log('⚠ 未找到任务数量按钮');
    }
    
    console.log('========================================');
    console.log('步骤 5: 滚动测试（验证 fixed 定位）');
    console.log('========================================');
    
    // 滚动页面
    await page.evaluate(() => window.scrollTo(0, 500));
    await page.waitForTimeout(500);
    
    const taskBarStillVisible = await taskBarLocator.isVisible();
    console.log('滚动后任务栏仍可见:', taskBarStillVisible ? '✓ 是' : '✗ 否');
    
    if (taskBarStillVisible) {
      const newBoundingBox = await taskBarContainer.boundingBox();
      if (newBoundingBox && boundingBox) {
        // fixed 定位的元素滚动后Y坐标应该不变
        const yDiff = Math.abs(newBoundingBox.y - boundingBox.y);
        console.log(`Y 坐标变化: ${yDiff}px`);
        
        if (yDiff < 5) {
          console.log('✓ 任务框保持固定位置（fixed 定位正常）');
        } else {
          console.log('⚠ 任务框位置发生变化，可能不是 fixed 定位');
        }
      }
    }
    
    // 滚回顶部
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(300);
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/slurm-taskbar-position.png', fullPage: true });
    console.log('✓ 已保存截图: test-screenshots/slurm-taskbar-position.png');
    
    console.log('========================================');
    console.log('测试完成 ✓');
    console.log('========================================');
    
    expect(taskBarVisible).toBeTruthy();
  });
});
