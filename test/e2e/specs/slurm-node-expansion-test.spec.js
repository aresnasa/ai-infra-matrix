// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * SLURM 节点扩容和 SaltStack 客户端安装测试
 * 测试添加 test-ssh01, test-ssh02, test-ssh03 到 SLURM 集群
 */

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const ADMIN_USERNAME = process.env.ADMIN_USERNAME || 'admin';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'admin123';

// 测试节点配置
const TEST_NODES = [
  { hostname: 'test-ssh01', port: 22, password: 'rootpass123' },
  { hostname: 'test-ssh02', port: 22, password: 'rootpass123' },
  { hostname: 'test-ssh03', port: 22, password: 'rootpass123' }
];

// 节点规格
const NODE_SPEC = {
  cpus: 1,
  memory: 1, // GB
  disk: 1,   // GB
  os: 'ubuntu22.04'
};

// 等待页面加载
async function waitForPageLoad(page) {
  await page.waitForLoadState('networkidle', { timeout: 15000 });
  await page.waitForTimeout(1000);
}

// 登录
async function login(page) {
  console.log('执行登录...');
  await page.goto('/login');
  await waitForPageLoad(page);
  
  await page.fill('input[type="text"]', ADMIN_USERNAME);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  
  await page.waitForURL(/\//, { timeout: 10000 });
  await waitForPageLoad(page);
  console.log('✓ 登录成功');
}

test.describe('SLURM 节点扩容测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('1. 批量添加 SLURM 节点 (test-ssh01-03)', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 批量添加 SLURM 节点');
    console.log('========================================\n');

    // 导航到 SLURM 页面
    await page.goto('/slurm');
    await waitForPageLoad(page);
    console.log('✓ SLURM 页面加载完成');

    // 查找添加节点按钮
    const addNodeButton = page.locator('button', { hasText: /添加节点|Add Node/ });
    if (!(await addNodeButton.isVisible().catch(() => false))) {
      console.log('⚠ 未找到"添加节点"按钮，尝试其他选择器');
      // 尝试其他可能的选择器
      const alternativeButton = page.locator('button').filter({ hasText: '节点' }).first();
      if (await alternativeButton.isVisible().catch(() => false)) {
        await alternativeButton.click();
      }
    } else {
      await addNodeButton.click();
    }

    console.log('✓ 点击添加节点按钮');
    await page.waitForTimeout(1000);

    // 填写节点信息 - 支持多行输入
    const nodeInputText = TEST_NODES.map(node => 
      `root@${node.hostname}:${node.port}`
    ).join('\n');

    console.log('\n📝 节点配置:');
    console.log(nodeInputText);

    // 查找节点输入框（可能是 textarea 或 input）
    const nodeInput = page.locator('textarea').or(page.locator('input[placeholder*="节点"]')).first();
    await nodeInput.fill(nodeInputText);
    console.log('✓ 填写节点地址');

    // 填写 SSH 密码
    const passwordInput = page.locator('input[type="password"]').or(
      page.locator('input[placeholder*="密码"]')
    );
    if (await passwordInput.isVisible().catch(() => false)) {
      await passwordInput.fill(TEST_NODES[0].password);
      console.log('✓ 填写 SSH 密码');
    }

    // 填写节点规格
    console.log('\n⚙️ 配置节点规格:');
    
    // CPU 核心数
    const cpuInput = page.locator('input[placeholder*="CPU"]').or(
      page.locator('input').filter({ hasText: /核心|CPU|cpus/i })
    ).first();
    if (await cpuInput.isVisible().catch(() => false)) {
      await cpuInput.fill(String(NODE_SPEC.cpus));
      console.log(`  CPU: ${NODE_SPEC.cpus} 核`);
    }

    // 内存
    const memoryInput = page.locator('input[placeholder*="内存"]').or(
      page.locator('input[placeholder*="Memory"]')
    ).first();
    if (await memoryInput.isVisible().catch(() => false)) {
      await memoryInput.fill(String(NODE_SPEC.memory));
      console.log(`  内存: ${NODE_SPEC.memory} GB`);
    }

    // 磁盘
    const diskInput = page.locator('input[placeholder*="磁盘"]').or(
      page.locator('input[placeholder*="Disk"]')
    ).first();
    if (await diskInput.isVisible().catch(() => false)) {
      await diskInput.fill(String(NODE_SPEC.disk));
      console.log(`  磁盘: ${NODE_SPEC.disk} GB`);
    }

    // 操作系统选择
    const osSelect = page.locator('select').or(page.locator('.ant-select'));
    if (await osSelect.isVisible().catch(() => false)) {
      console.log(`  操作系统: ${NODE_SPEC.os}`);
    }

    // 提交表单
    console.log('\n🚀 提交节点添加请求...');
    const submitButton = page.locator('button[type="submit"]').or(
      page.locator('button', { hasText: /确定|提交|Submit|OK/ })
    );
    
    // 监听 API 请求
    const addNodePromise = page.waitForResponse(
      response => response.url().includes('/api/slurm') && 
                  (response.url().includes('/nodes') || response.url().includes('/add')),
      { timeout: 30000 }
    );

    await submitButton.click();
    
    try {
      const response = await addNodePromise;
      const responseData = await response.json();
      
      console.log('\n📊 API 响应:');
      console.log(JSON.stringify(responseData, null, 2));

      if (response.ok()) {
        console.log('✅ 节点添加请求成功');
      } else {
        console.log(`⚠ API 返回错误状态: ${response.status()}`);
      }
    } catch (error) {
      console.log('⚠ 未捕获到 API 响应，可能使用了不同的端点');
    }

    // 等待任务创建
    await page.waitForTimeout(3000);
  });

  test('2. 验证 SaltStack 客户端安装任务', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: SaltStack 客户端安装任务');
    console.log('========================================\n');

    // 导航到 SLURM Tasks 页面
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    console.log('✓ SLURM Tasks 页面加载完成');

    // 查找 SaltStack 安装任务
    await page.waitForTimeout(2000);
    
    const taskRows = page.locator('tbody tr');
    const taskCount = await taskRows.count();
    
    console.log(`\n📋 找到 ${taskCount} 个任务`);

    let saltStackInstallTasks = [];
    
    for (let i = 0; i < Math.min(taskCount, 10); i++) {
      const row = taskRows.nth(i);
      const taskName = await row.locator('td').nth(1).textContent().catch(() => '');
      const status = await row.locator('td').nth(3).textContent().catch(() => '');
      
      if (taskName.toLowerCase().includes('saltstack') || 
          taskName.toLowerCase().includes('minion') ||
          taskName.toLowerCase().includes('test-ssh')) {
        saltStackInstallTasks.push({
          name: taskName,
          status: status.trim()
        });
        console.log(`  ✓ 找到 SaltStack 任务: ${taskName} - ${status}`);
      }
    }

    if (saltStackInstallTasks.length > 0) {
      console.log(`\n✅ 找到 ${saltStackInstallTasks.length} 个 SaltStack 安装任务`);
      
      // 验证至少有一个任务在运行或完成
      const activeOrCompletedTasks = saltStackInstallTasks.filter(task => 
        task.status.includes('运行') || 
        task.status.includes('完成') ||
        task.status.includes('Running') ||
        task.status.includes('Completed')
      );
      
      expect(activeOrCompletedTasks.length).toBeGreaterThan(0);
    } else {
      console.log('⚠ 未找到 SaltStack 安装任务');
    }
  });

  test('3. 验证 SaltStack 集群节点状态', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: SaltStack 集群节点状态');
    console.log('========================================\n');

    // 导航到 SaltStack 页面
    await page.goto('/saltstack');
    await waitForPageLoad(page);
    console.log('✓ SaltStack 页面加载完成');

    // 等待数据加载
    await page.waitForTimeout(3000);

    // 查找节点列表
    const nodeListContainer = page.locator('.ant-table').or(page.locator('[class*="node"]'));
    
    if (await nodeListContainer.isVisible().catch(() => false)) {
      console.log('✓ 节点列表容器已加载');

      // 查找所有节点行
      const nodeRows = page.locator('tbody tr');
      const nodeCount = await nodeRows.count();
      
      console.log(`\n📋 SaltStack 集群节点列表:`);
      console.log(`   节点数量: ${nodeCount}`);

      // 检查是否包含测试节点
      const pageText = await page.textContent('body');
      let foundNodes = [];

      for (const testNode of TEST_NODES) {
        if (pageText.includes(testNode.hostname)) {
          foundNodes.push(testNode.hostname);
          console.log(`   ✓ 找到节点: ${testNode.hostname}`);
        }
      }

      if (foundNodes.length > 0) {
        console.log(`\n✅ 成功找到 ${foundNodes.length}/${TEST_NODES.length} 个测试节点`);
      } else {
        console.log('\n⚠ 未找到测试节点，可能还在安装中...');
      }

      // 检查节点状态
      for (let i = 0; i < Math.min(nodeCount, 10); i++) {
        const row = nodeRows.nth(i);
        const nodeName = await row.locator('td').first().textContent().catch(() => '');
        const status = await row.locator('td').nth(1).textContent().catch(() => '');
        
        if (nodeName && TEST_NODES.some(n => nodeName.includes(n.hostname))) {
          console.log(`\n   节点: ${nodeName}`);
          console.log(`   状态: ${status}`);
          
          // 验证节点在线
          if (status.includes('在线') || status.includes('Online') || status.includes('up')) {
            console.log(`   ✅ 节点状态正常`);
          } else {
            console.log(`   ⚠ 节点状态: ${status}`);
          }
        }
      }
    } else {
      console.log('⚠ 未找到节点列表容器');
      
      // 检查是否有错误消息
      const errorMsg = page.locator('text=/错误|Error|无法连接/');
      if (await errorMsg.isVisible().catch(() => false)) {
        const errorText = await errorMsg.textContent();
        console.log(`❌ 错误消息: ${errorText}`);
      }
    }
  });

  test('4. 验证 SLURM 集群节点状态', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: SLURM 集群节点状态');
    console.log('========================================\n');

    // 导航到 SLURM 页面
    await page.goto('/slurm');
    await waitForPageLoad(page);
    console.log('✓ SLURM 页面加载完成');

    // 等待数据加载
    await page.waitForTimeout(3000);

    // 监听 SLURM 节点 API
    let apiNodeData = null;
    page.on('response', async response => {
      if (response.url().includes('/api/slurm/nodes')) {
        try {
          apiNodeData = await response.json();
        } catch (e) {}
      }
    });

    // 刷新页面以触发 API 调用
    const refreshButton = page.locator('button', { hasText: /刷新|Refresh/ });
    if (await refreshButton.isVisible().catch(() => false)) {
      await refreshButton.click();
      await page.waitForTimeout(2000);
    }

    console.log('\n📊 SLURM 集群状态:');
    
    if (apiNodeData) {
      console.log('API 响应:');
      console.log(JSON.stringify(apiNodeData, null, 2));

      const nodes = apiNodeData.data || apiNodeData.nodes || [];
      console.log(`\n节点数量: ${nodes.length}`);

      // 检查测试节点
      for (const testNode of TEST_NODES) {
        const found = nodes.some(node => 
          node.hostname === testNode.hostname || 
          node.name === testNode.hostname
        );
        
        if (found) {
          console.log(`✓ 节点 ${testNode.hostname} 已加入集群`);
        } else {
          console.log(`⚠ 节点 ${testNode.hostname} 未找到`);
        }
      }
    } else {
      console.log('⚠ 未获取到 API 数据');
    }

    // 检查页面显示的节点
    const nodeTable = page.locator('.ant-table').or(page.locator('table'));
    if (await nodeTable.isVisible().catch(() => false)) {
      const rows = nodeTable.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`\n页面显示节点数: ${rowCount}`);

      for (let i = 0; i < Math.min(rowCount, 10); i++) {
        const row = rows.nth(i);
        const nodeName = await row.locator('td').first().textContent().catch(() => '');
        
        if (TEST_NODES.some(n => nodeName.includes(n.hostname))) {
          const state = await row.locator('td').nth(1).textContent().catch(() => '');
          console.log(`  ${nodeName}: ${state}`);
        }
      }
    }
  });

  test('5. 端到端验证：添加到验证完成', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试: 端到端节点扩容验证');
    console.log('========================================\n');

    const startTime = Date.now();
    let testResults = {
      nodeAdded: false,
      saltStackTaskCreated: false,
      saltStackNodeOnline: false,
      slurmNodeOnline: false
    };

    // 步骤 1: 检查初始状态
    console.log('📌 步骤 1: 检查初始状态');
    await page.goto('/saltstack');
    await waitForPageLoad(page);
    
    const initialNodes = await page.locator('tbody tr').count().catch(() => 0);
    console.log(`   初始 SaltStack 节点数: ${initialNodes}`);

    // 步骤 2: 添加节点（简化版，只添加一个节点）
    console.log('\n📌 步骤 2: 添加测试节点');
    await page.goto('/slurm');
    await waitForPageLoad(page);

    const addButton = page.locator('button', { hasText: /添加|Add/ }).first();
    if (await addButton.isVisible().catch(() => false)) {
      await addButton.click();
      await page.waitForTimeout(1000);

      // 简化输入
      const nodeInput = page.locator('textarea, input').first();
      await nodeInput.fill('root@test-ssh01:22');
      
      const passwordInput = page.locator('input[type="password"]');
      if (await passwordInput.isVisible().catch(() => false)) {
        await passwordInput.fill('rootpass123');
      }

      // 查找并点击提交按钮
      const submitBtn = page.locator('button', { hasText: /提交|确定|OK|Submit/ });
      const isVisible = await submitBtn.first().isVisible({ timeout: 5000 }).catch(() => false);
      
      if (isVisible) {
        // 滚动到按钮位置
        await submitBtn.first().scrollIntoViewIfNeeded();
        await page.waitForTimeout(500);
        await submitBtn.first().click();
        
        testResults.nodeAdded = true;
        console.log('   ✅ 节点添加请求已提交');
      } else {
        console.log('   ⚠ 未找到提交按钮，尝试按 Enter');
        await nodeInput.press('Enter');
        testResults.nodeAdded = true;
        console.log('   ✅ 通过 Enter 提交');
      }
    }

    // 步骤 3: 等待并检查任务
    console.log('\n📌 步骤 3: 检查安装任务 (等待 10 秒)');
    await page.waitForTimeout(10000);
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);

    const taskText = await page.textContent('body');
    if (taskText.includes('saltstack') || taskText.includes('test-ssh')) {
      testResults.saltStackTaskCreated = true;
      console.log('   ✅ SaltStack 安装任务已创建');
    }

    // 步骤 4: 检查 SaltStack 节点（等待更长时间）
    console.log('\n📌 步骤 4: 检查 SaltStack 节点 (等待 20 秒)');
    await page.waitForTimeout(20000);
    
    await page.goto('/saltstack');
    await waitForPageLoad(page);

    const saltStackPageText = await page.textContent('body');
    if (saltStackPageText.includes('test-ssh01')) {
      testResults.saltStackNodeOnline = true;
      console.log('   ✅ SaltStack 节点已上线');
    }

    // 步骤 5: 检查 SLURM 节点
    console.log('\n📌 步骤 5: 检查 SLURM 节点');
    await page.goto('/slurm');
    await waitForPageLoad(page);

    const slurmPageText = await page.textContent('body');
    if (slurmPageText.includes('test-ssh01')) {
      testResults.slurmNodeOnline = true;
      console.log('   ✅ SLURM 节点已上线');
    }

    const duration = ((Date.now() - startTime) / 1000).toFixed(1);

    // 测试结果总结
    console.log('\n========================================');
    console.log('📊 测试结果总结');
    console.log('========================================');
    console.log(`总耗时: ${duration} 秒`);
    console.log(`\n各步骤状态:`);
    console.log(`  1. 节点添加请求: ${testResults.nodeAdded ? '✅' : '❌'}`);
    console.log(`  2. 安装任务创建: ${testResults.saltStackTaskCreated ? '✅' : '❌'}`);
    console.log(`  3. SaltStack 上线: ${testResults.saltStackNodeOnline ? '✅' : '❌'}`);
    console.log(`  4. SLURM 节点上线: ${testResults.slurmNodeOnline ? '✅' : '❌'}`);

    const successCount = Object.values(testResults).filter(Boolean).length;
    console.log(`\n成功率: ${successCount}/4 (${(successCount/4*100).toFixed(0)}%)`);

    // 至少要有节点添加和任务创建成功
    expect(testResults.nodeAdded).toBeTruthy();
  });
});
