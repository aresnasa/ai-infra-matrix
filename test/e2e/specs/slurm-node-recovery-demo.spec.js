const { test, expect } = require('@playwright/test');

/**
 * SLURM 节点恢复演示
 * 通过 Web 界面将 down 状态的节点恢复为正常状态
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('SLURM 节点恢复演示', () => {
  test('演示：通过 Web 界面恢复 down 节点', async ({ page }) => {
    console.log('\n🎬 开始演示 SLURM 节点恢复流程...\n');
    
    // 步骤 1: 登录系统
    console.log('步骤 1: 登录系统');
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    const usernameInput = page.locator('input[type="text"]').first();
    const passwordInput = page.locator('input[type="password"]').first();
    const loginButton = page.locator('button[type="submit"]').first();
    
    await usernameInput.fill('admin');
    await passwordInput.fill('admin123');
    await loginButton.click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    console.log('  ✅ 登录成功\n');
    
    // 步骤 2: 访问 SLURM 页面
    console.log('步骤 2: 访问 SLURM 管理页面');
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    console.log('  ✅ 页面加载完成\n');
    
    // 截图 1: 初始状态
    await page.screenshot({ 
      path: 'test-screenshots/demo-01-initial-state.png',
      fullPage: true 
    });
    console.log('  📸 截图保存: demo-01-initial-state.png\n');
    
    // 步骤 3: 查看节点列表
    console.log('步骤 3: 查看节点列表');
    const nodeTable = page.locator('table').filter({ 
      has: page.locator('th').filter({ hasText: /节点|Node/ })
    }).first();
    
    const rows = nodeTable.locator('tbody tr');
    const rowCount = await rows.count();
    console.log(`  📊 当前节点总数: ${rowCount}`);
    
    // 统计节点状态
    let downCount = 0;
    let otherCount = 0;
    for (let i = 0; i < rowCount; i++) {
      const stateCell = rows.nth(i).locator('td').nth(2);
      const stateText = await stateCell.textContent();
      if (stateText && stateText.toLowerCase().includes('down')) {
        downCount++;
      } else {
        otherCount++;
      }
    }
    console.log(`  ⚠️  DOWN 状态节点: ${downCount}`);
    console.log(`  ✅ 其他状态节点: ${otherCount}\n`);
    
    if (downCount === 0) {
      console.log('🎉 所有节点已处于正常状态，无需恢复！\n');
      return;
    }
    
    // 步骤 4: 选择所有 down 状态的节点
    console.log('步骤 4: 选择 DOWN 状态的节点');
    console.log('  💡 提示: 您可以单独选择节点，或使用"全选"功能\n');
    
    // 方法 A: 选择所有节点（全选）
    const selectAllCheckbox = nodeTable.locator('thead input[type="checkbox"]').first();
    if (await selectAllCheckbox.isVisible({ timeout: 3000 })) {
      console.log('  🔲 使用"全选"功能选择所有节点...');
      await selectAllCheckbox.check();
      await page.waitForTimeout(1000);
      console.log('  ✅ 已选择所有节点\n');
    } else {
      // 方法 B: 手动选择第一个 down 节点
      console.log('  🔲 手动选择第一个 DOWN 状态的节点...');
      for (let i = 0; i < rowCount; i++) {
        const row = rows.nth(i);
        const stateCell = row.locator('td').nth(2);
        const stateText = await stateCell.textContent();
        
        if (stateText && stateText.toLowerCase().includes('down')) {
          const checkbox = row.locator('input[type="checkbox"]').first();
          await checkbox.check();
          console.log(`  ✅ 已选择节点\n`);
          break;
        }
      }
    }
    
    // 截图 2: 选中节点后
    await page.screenshot({ 
      path: 'test-screenshots/demo-02-nodes-selected.png',
      fullPage: true 
    });
    console.log('  📸 截图保存: demo-02-nodes-selected.png\n');
    
    // 步骤 5: 点击"节点操作"按钮
    console.log('步骤 5: 点击"节点操作"按钮');
    console.log('  💡 提示: 此按钮仅在选中节点后显示\n');
    
    const actionButton = page.locator('button').filter({ 
      hasText: /节点操作/i 
    }).first();
    
    await expect(actionButton).toBeVisible({ timeout: 5000 });
    console.log('  ✅ 找到"节点操作"按钮');
    
    await actionButton.click();
    await page.waitForTimeout(500);
    console.log('  ✅ 已点击"节点操作"按钮\n');
    
    // 截图 3: 操作菜单展开
    await page.screenshot({ 
      path: 'test-screenshots/demo-03-operation-menu.png',
      fullPage: true 
    });
    console.log('  📸 截图保存: demo-03-operation-menu.png\n');
    
    // 步骤 6: 选择"恢复 (RESUME)"操作
    console.log('步骤 6: 选择"恢复 (RESUME)"操作');
    
    const dropdownMenu = page.locator('.ant-dropdown:visible');
    await expect(dropdownMenu).toBeVisible({ timeout: 3000 });
    
    const resumeOption = dropdownMenu.locator('.ant-dropdown-menu-item').filter({ 
      hasText: /恢复|RESUME/i 
    }).first();
    
    await expect(resumeOption).toBeVisible({ timeout: 3000 });
    console.log('  ✅ 找到"恢复 (RESUME)"选项');
    
    await resumeOption.click();
    await page.waitForTimeout(500);
    console.log('  ✅ 已选择"恢复 (RESUME)"操作\n');
    
    // 步骤 7: 确认操作
    console.log('步骤 7: 确认操作');
    
    const confirmModal = page.locator('.ant-modal:visible');
    await expect(confirmModal).toBeVisible({ timeout: 3000 });
    console.log('  📋 确认对话框已显示');
    
    // 截图 4: 确认对话框
    await page.screenshot({ 
      path: 'test-screenshots/demo-04-confirm-dialog.png',
      fullPage: true 
    });
    console.log('  📸 截图保存: demo-04-confirm-dialog.png\n');
    
    const confirmButton = confirmModal.locator('button').filter({ 
      hasText: /确定|确认/i 
    }).first();
    
    await confirmButton.click();
    console.log('  ✅ 已确认操作\n');
    
    // 步骤 8: 等待操作结果
    console.log('步骤 8: 等待操作结果');
    console.log('  ⏳ 正在执行节点恢复操作...\n');
    
    // 等待成功消息
    const successMessage = page.locator('.ant-message-success, .ant-notification-success');
    
    try {
      await expect(successMessage).toBeVisible({ timeout: 10000 });
      console.log('  ✅ 操作成功！\n');
      
      // 截图 5: 成功消息
      await page.screenshot({ 
        path: 'test-screenshots/demo-05-success.png',
        fullPage: true 
    });
      console.log('  📸 截图保存: demo-05-success.png\n');
    } catch (e) {
      console.log('  ⚠️  未检测到成功消息（可能操作仍在进行中）\n');
    }
    
    // 步骤 9: 刷新页面查看最新状态
    console.log('步骤 9: 刷新页面查看最新状态');
    await page.waitForTimeout(3000);
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    console.log('  ✅ 页面已刷新\n');
    
    // 截图 6: 最终状态
    await page.screenshot({ 
      path: 'test-screenshots/demo-06-final-state.png',
      fullPage: true 
    });
    console.log('  📸 截图保存: demo-06-final-state.png\n');
    
    // 统计最终节点状态
    console.log('步骤 10: 验证节点状态');
    const finalRows = nodeTable.locator('tbody tr');
    const finalRowCount = await finalRows.count();
    
    let finalDownCount = 0;
    let finalIdleCount = 0;
    let finalOtherCount = 0;
    
    for (let i = 0; i < finalRowCount; i++) {
      const stateCell = finalRows.nth(i).locator('td').nth(2);
      const stateText = await stateCell.textContent();
      const lowerState = stateText ? stateText.toLowerCase() : '';
      
      if (lowerState.includes('down')) {
        finalDownCount++;
      } else if (lowerState.includes('idle')) {
        finalIdleCount++;
      } else {
        finalOtherCount++;
      }
    }
    
    console.log(`  📊 最终节点状态统计:`);
    console.log(`    - DOWN: ${finalDownCount}`);
    console.log(`    - IDLE: ${finalIdleCount}`);
    console.log(`    - 其他: ${finalOtherCount}\n`);
    
    // 总结
    console.log('═'.repeat(60));
    console.log('🎉 演示完成！');
    console.log('═'.repeat(60));
    console.log('\n操作摘要:');
    console.log(`  - 初始 DOWN 节点: ${downCount}`);
    console.log(`  - 最终 DOWN 节点: ${finalDownCount}`);
    console.log(`  - 恢复成功节点: ${downCount - finalDownCount}`);
    console.log('\n截图保存位置:');
    console.log('  - test-screenshots/demo-01-initial-state.png');
    console.log('  - test-screenshots/demo-02-nodes-selected.png');
    console.log('  - test-screenshots/demo-03-operation-menu.png');
    console.log('  - test-screenshots/demo-04-confirm-dialog.png');
    console.log('  - test-screenshots/demo-05-success.png');
    console.log('  - test-screenshots/demo-06-final-state.png');
    console.log('\n💡 提示:');
    console.log('  如果节点仍显示 DOWN 状态，可能需要检查：');
    console.log('  1. SLURM 服务是否正常运行');
    console.log('  2. 节点网络连接是否正常');
    console.log('  3. 后端日志是否有错误信息');
    console.log('═'.repeat(60) + '\n');
  });
});
