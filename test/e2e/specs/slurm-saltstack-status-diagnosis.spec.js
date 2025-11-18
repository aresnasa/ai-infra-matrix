/**
 * SLURM SaltStack 状态同步问题诊断测试
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('SLURM SaltStack 状态同步问题诊断', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto(`${BASE_URL}/login`);
    
    // 等待页面加载完成
    await page.waitForLoadState('domcontentloaded');
    
    // 等待登录表单加载并可见（使用 placeholder 选择器）
    await page.waitForSelector('input[placeholder="用户名"]', { state: 'visible', timeout: 30000 });
    
    // 填写登录信息
    await page.fill('input[placeholder="用户名"]', 'admin');
    await page.fill('input[placeholder="密码"]', 'admin123');
    
    // 点击登录按钮
    await page.click('button:has-text("登 录")');
    
    // 等待导航完成（可能会跳转到 /dashboard 或 /projects）
    await page.waitForNavigation({ timeout: 10000 });
    
    // 验证登录成功（URL 应该不再是 /login）
    expect(page.url()).not.toContain('/login');
  });

  test('检查 SLURM 页面 SaltStack 状态显示', async ({ page }) => {
    console.log('\n=== 访问 SLURM 页面 ===');
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    
    // 等待页面加载
    await page.waitForTimeout(2000);
    
    // 截图保存当前状态
    await page.screenshot({ path: 'test-screenshots/slurm-page-overview.png', fullPage: true });
    console.log('✅ 截图已保存: test-screenshots/slurm-page-overview.png');
    
    // 检查 SaltStack 状态卡片
    console.log('\n=== 检查 SaltStack 状态卡片 ===');
    const saltStackCard = page.locator('.ant-card:has-text("SaltStack 集成状态")');
    const hasSaltStackCard = await saltStackCard.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (hasSaltStackCard) {
      console.log('✅ 找到 SaltStack 集成状态卡片');
      
      // 检查状态列
      const statusColumn = page.locator('text=/状态/i');
      const hasStatusColumn = await statusColumn.isVisible({ timeout: 2000 }).catch(() => false);
      console.log(`状态列显示: ${hasStatusColumn ? '✅' : '❌'}`);
      
      // 检查 minions 列表
      const minionsList = page.locator('.ant-tag').filter({ hasText: /test-|slurm-/ });
      const minionsCount = await minionsList.count();
      console.log(`Minions 数量: ${minionsCount}`);
      
      if (minionsCount > 0) {
        for (let i = 0; i < Math.min(minionsCount, 5); i++) {
          const minionText = await minionsList.nth(i).textContent();
          console.log(`  - Minion ${i + 1}: ${minionText}`);
        }
      }
    } else {
      console.log('⚠️ 未找到 SaltStack 集成状态卡片');
    }
    
    // 检查节点列表表格
    console.log('\n=== 检查节点列表表格 ===');
    const nodeTable = page.locator('.ant-table:has-text("节点列表")').or(page.locator('table:has(thead:has-text("节点名"))'));
    const hasNodeTable = await nodeTable.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (hasNodeTable) {
      console.log('✅ 找到节点列表表格');
      
      // 检查表头
      const headers = await page.locator('thead th').allTextContents();
      console.log('表头列: ', headers);
      
      // 检查是否有状态列
      const hasStatusHeader = headers.some(h => h.includes('状态') || h.toLowerCase().includes('status'));
      console.log(`状态列存在: ${hasStatusHeader ? '✅' : '❌'}`);
      
      // 检查数据行
      const rows = page.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`节点数量: ${rowCount}`);
      
      if (rowCount > 0) {
        console.log('\n前 3 个节点数据:');
        for (let i = 0; i < Math.min(rowCount, 3); i++) {
          const rowData = await rows.nth(i).locator('td').allTextContents();
          console.log(`  行 ${i + 1}: ${JSON.stringify(rowData)}`);
        }
      }
    } else {
      console.log('⚠️ 未找到节点列表表格');
    }
    
    // 检查节点操作按钮
    console.log('\n=== 检查节点管理按钮 ===');
    const operationButtons = [
      { name: '添加节点', selector: 'button:has-text("添加节点")' },
      { name: '节点操作', selector: 'button:has-text("节点操作")' },
      { name: '扩容', selector: 'button:has-text("扩容")' },
      { name: '删除', selector: 'button:has-text("删除")' },
      { name: 'RESUME', selector: 'button:has-text("RESUME")' },
      { name: 'DRAIN', selector: 'button:has-text("DRAIN")' }
    ];
    
    for (const btn of operationButtons) {
      const button = page.locator(btn.selector);
      const isVisible = await button.isVisible({ timeout: 1000 }).catch(() => false);
      console.log(`${btn.name} 按钮: ${isVisible ? '✅ 可见' : '❌ 不可见'}`);
    }
  });

  test('检查 SaltStack API 响应', async ({ page }) => {
    console.log('\n=== 检查 SaltStack API ===');
    
    // 监听 API 请求
    const apiRequests = [];
    page.on('request', request => {
      if (request.url().includes('/api/slurm/saltstack')) {
        apiRequests.push({
          url: request.url(),
          method: request.method()
        });
      }
    });
    
    const apiResponses = [];
    page.on('response', async response => {
      if (response.url().includes('/api/slurm/saltstack')) {
        const data = await response.json().catch(() => null);
        apiResponses.push({
          url: response.url(),
          status: response.status(),
          data: data
        });
      }
    });
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    console.log(`\nAPI 请求数量: ${apiRequests.length}`);
    apiRequests.forEach((req, i) => {
      console.log(`  ${i + 1}. ${req.method} ${req.url}`);
    });
    
    console.log(`\nAPI 响应数量: ${apiResponses.length}`);
    apiResponses.forEach((res, i) => {
      console.log(`  ${i + 1}. ${res.status} ${res.url}`);
      if (res.data) {
        console.log(`     数据: ${JSON.stringify(res.data).substring(0, 200)}...`);
      }
    });
  });

  test('测试节点管理功能', async ({ page }) => {
    console.log('\n=== 测试节点管理功能 ===');
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 查找添加节点按钮
    const addNodeButton = page.locator('button:has-text("添加节点")');
    const hasAddButton = await addNodeButton.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (hasAddButton) {
      console.log('✅ 找到"添加节点"按钮');
      
      // 点击添加节点按钮
      await addNodeButton.click();
      await page.waitForTimeout(1000);
      
      // 检查弹窗
      const modal = page.locator('.ant-modal:visible');
      const hasModal = await modal.isVisible({ timeout: 2000 }).catch(() => false);
      
      if (hasModal) {
        console.log('✅ 添加节点弹窗已打开');
        await page.screenshot({ path: 'test-screenshots/slurm-add-node-modal.png' });
        
        // 检查表单字段
        const formFields = await modal.locator('.ant-form-item-label').allTextContents();
        console.log('表单字段: ', formFields);
        
        // 关闭弹窗 - 使用正确的按钮文本（含空格）
        const cancelButton = modal.locator('button:has-text("取 消")');
        await cancelButton.click();
        await page.waitForTimeout(500);
      } else {
        console.log('❌ 添加节点弹窗未打开');
      }
    } else {
      console.log('❌ 未找到"添加节点"按钮');
    }
    
    // 测试节点选择和操作
    console.log('\n=== 测试节点选择和批量操作 ===');
    
    // 选择第一个节点
    const firstCheckbox = page.locator('tbody tr').first().locator('.ant-checkbox-input');
    const hasCheckbox = await firstCheckbox.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (hasCheckbox) {
      console.log('✅ 找到节点选择框');
      await firstCheckbox.click();
      await page.waitForTimeout(500);
      
      // 检查节点操作下拉菜单
      const operationDropdown = page.locator('button:has-text("节点操作")');
      const hasDropdown = await operationDropdown.isVisible({ timeout: 2000 }).catch(() => false);
      
      if (hasDropdown) {
        console.log('✅ 找到"节点操作"下拉菜单');
        await operationDropdown.click();
        await page.waitForTimeout(500);
        
        // 检查下拉菜单项
        const menuItems = await page.locator('.ant-dropdown-menu-item').allTextContents();
        console.log('操作菜单项: ', menuItems);
        
        await page.screenshot({ path: 'test-screenshots/slurm-node-operations.png' });
        
        // 关闭菜单
        await page.keyboard.press('Escape');
      } else {
        console.log('❌ 未找到"节点操作"下拉菜单');
      }
    } else {
      console.log('⚠️ 节点列表为空或无选择框');
    }
  });

  test('生成诊断报告', async ({ page }) => {
    console.log('\n=== SLURM SaltStack 状态同步问题诊断报告 ===\n');
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    const issues = [];
    const recommendations = [];
    
    // 检查 1: SaltStack 状态卡片
    const saltStackCard = page.locator('.ant-card:has-text("SaltStack 集成状态")');
    const hasSaltStackCard = await saltStackCard.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (!hasSaltStackCard) {
      issues.push('❌ SaltStack 集成状态卡片不可见');
      recommendations.push('检查 SlurmDashboard.js 中是否正确加载 SaltStack 数据');
    }
    
    // 检查 2: 状态列
    const statusColumn = page.locator('th:has-text("状态")');
    const hasStatusColumn = await statusColumn.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (!hasStatusColumn) {
      issues.push('❌ 节点列表缺少"状态"列');
      recommendations.push('在 columnsNodes 中添加状态列定义');
    }
    
    // 检查 3: 管理按钮
    const addButton = page.locator('button:has-text("添加节点")');
    const hasAddButton = await addButton.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (!hasAddButton) {
      issues.push('❌ 缺少"添加节点"按钮');
      recommendations.push('检查 SlurmDashboard.js 中按钮渲染逻辑');
    }
    
    // 输出报告
    console.log('发现的问题:');
    if (issues.length === 0) {
      console.log('  ✅ 未发现明显问题');
    } else {
      issues.forEach((issue, i) => {
        console.log(`  ${i + 1}. ${issue}`);
      });
    }
    
    console.log('\n修复建议:');
    if (recommendations.length === 0) {
      console.log('  ✅ 无需修复');
    } else {
      recommendations.forEach((rec, i) => {
        console.log(`  ${i + 1}. ${rec}`);
      });
    }
    
    console.log('\n需要检查的文件:');
    console.log('  1. src/frontend/src/pages/SlurmDashboard.js');
    console.log('  2. src/frontend/src/services/api.js');
    console.log('  3. src/backend/internal/controllers/slurm_controller.go');
    
    console.log('\n=== 诊断完成 ===\n');
  });
});
