import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';

async function login(page: any) {
  await page.goto(`${BASE_URL}`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(1000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  if (!hasLoginForm) {
    console.log('✓ 已登录');
    return;
  }
  
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  console.log('✓ 登录完成');
}

test('调试 SLURM 页面结构', async ({ page }) => {
  await login(page);
  
  // 访问 SLURM 页面
  await page.goto(`${BASE_URL}/slurm`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  await page.waitForTimeout(2000);
  
  // 截图初始状态
  await page.screenshot({ 
    path: 'test-results/slurm-page-initial.png', 
    fullPage: true 
  });
  console.log('✓ 截图: slurm-page-initial.png');
  
  // 查找所有 Tabs
  const tabs = await page.locator('.ant-tabs-tab').allTextContents();
  console.log('\n可用的 Tabs:', tabs);
  
  // 切换到节点管理
  const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: '节点管理' });
  if (await nodesTab.count() > 0) {
    await nodesTab.click();
    await page.waitForTimeout(1000);
    console.log('✓ 切换到节点管理 Tab');
    
    await page.screenshot({ 
      path: 'test-results/slurm-page-nodes-tab.png', 
      fullPage: true 
    });
    console.log('✓ 截图: slurm-page-nodes-tab.png');
  }
  
  // 查找所有按钮
  const buttons = await page.locator('button').allTextContents();
  console.log('\n可用的按钮:', buttons);
  
  // 查找添加节点按钮
  const addNodeButton = page.locator('button').filter({ hasText: '添加节点' });
  const addNodeCount = await addNodeButton.count();
  console.log(`\n添加节点按钮数量: ${addNodeCount}`);
  
  if (addNodeCount > 0) {
    await addNodeButton.click();
    await page.waitForTimeout(2000);
    console.log('✓ 点击添加节点按钮');
    
    await page.screenshot({ 
      path: 'test-results/slurm-page-modal-opened.png', 
      fullPage: true 
    });
    console.log('✓ 截图: slurm-page-modal-opened.png');
    
    // 检查 Modal
    const modals = await page.locator('.ant-modal').count();
    console.log(`\nModal 数量: ${modals}`);
    
    if (modals > 0) {
      const modalTitle = await page.locator('.ant-modal-title').textContent();
      console.log(`Modal 标题: ${modalTitle}`);
      
      const modalContent = await page.locator('.ant-modal-body').textContent();
      console.log(`Modal 内容长度: ${modalContent?.length} 字符`);
      
      // 查找输入框
      const inputs = await page.locator('.ant-modal input').count();
      console.log(`Modal 中的输入框数量: ${inputs}`);
      
      // 查找所有输入框的 placeholder
      const placeholders = await page.locator('.ant-modal input').evaluateAll((els: any[]) => 
        els.map(el => ({ 
          placeholder: el.placeholder, 
          type: el.type,
          id: el.id
        }))
      );
      console.log('输入框详情:', JSON.stringify(placeholders, null, 2));
      
      // 查找所有按钮
      const modalButtons = await page.locator('.ant-modal button').allTextContents();
      console.log('\nModal 中的按钮:', modalButtons);
      
      // 查找主机名输入
      const hostnameInput = page.locator('.ant-modal textarea[placeholder*="节点"]');
      const hasHostnameInput = await hostnameInput.count() > 0;
      console.log(`\n主机名 textarea: ${hasHostnameInput}`);
      
      if (hasHostnameInput) {
        await hostnameInput.fill('test-ssh01,test-ssh02,test-ssh03');
        console.log('✓ 填入主机列表');
        
        // 填入密码
        const pwdInput = page.locator('.ant-modal input[type="password"]');
        if (await pwdInput.count() > 0) {
          await pwdInput.fill('root');
          console.log('✓ 填入密码');
        }
        
        // 等待一下让组件更新
        await page.waitForTimeout(1000);
        
        // 再次截图
        await page.screenshot({ 
          path: 'test-results/slurm-page-modal-filled.png', 
          fullPage: true 
        });
        console.log('✓ 截图: slurm-page-modal-filled.png');
        
        // 查找测试按钮
        const testButton = page.locator('.ant-modal button').filter({ hasText: /测试.*SSH/ });
        const testButtonCount = await testButton.count();
        console.log(`\n测试SSH连接按钮数量: ${testButtonCount}`);
        
        if (testButtonCount > 0) {
          await testButton.click();
          console.log('✓ 点击测试按钮');
          
          await page.waitForTimeout(5000);
          
          await page.screenshot({ 
            path: 'test-results/slurm-page-test-result.png', 
            fullPage: true 
          });
          console.log('✓ 截图: slurm-page-test-result.png');
        }
      }
    }
  }
});
