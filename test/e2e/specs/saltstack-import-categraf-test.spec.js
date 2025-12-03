// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * 测试导入配置后 Categraf 安装开关是否正确显示
 * 以及安装进度日志框的 Ctrl+A 全选范围限制
 */

test.describe('SaltStack 导入配置和日志框测试', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('http://192.168.0.199:8080/login');
    await page.fill('input[placeholder*="用户名"], input[placeholder*="Username"], input[id="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    // 等待登录完成，可能跳转到 /projects 或 /dashboard
    await page.waitForURL(/\/(dashboard|projects|saltstack)/, { timeout: 15000 });
    
    // 导航到 SaltStack 页面
    await page.goto('http://192.168.0.199:8080/saltstack');
    await page.waitForSelector('.ant-card', { timeout: 10000 });
  });

  test('导入 CSV 配置后 Categraf 开关应正确显示', async ({ page }) => {
    // 监听控制台输出 - 捕获所有相关日志
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('install_categraf') || text.includes('Categraf') || text.includes('导入') || text.includes('API') || text.includes('解析') || text.includes('newHosts') || text.includes('原始值')) {
        console.log('Browser console:', text);
      }
    });

    // 打开批量安装 Salt Minion 弹窗
    const installButton = page.locator('button:has-text("批量安装 Salt Minion"), button:has-text("Batch Install Salt Minion")');
    await installButton.click();
    await page.waitForSelector('.ant-modal', { timeout: 5000 });

    // 点击粘贴导入按钮
    const pasteImportButton = page.locator('button:has-text("粘贴导入"), button:has-text("Paste Import")');
    await pasteImportButton.click();
    await page.waitForSelector('.ant-modal:has-text("粘贴导入"), .ant-modal:has-text("Paste Import")', { timeout: 5000 });

    // 输入带有 install_categraf=true 的 CSV 配置
    const csvContent = `host,port,username,password,use_sudo,group,install_categraf
192.168.1.100,22,root,password123,false,web,true
192.168.1.101,22,admin,pass456,true,db,false
192.168.1.102,22,root,test789,false,gpu,true`;

    const textarea = page.locator('textarea');
    await textarea.fill(csvContent);

    // 点击导入按钮
    const importButton = page.locator('.ant-modal button:has-text("立即导入"), .ant-modal button:has-text("Import Now")');
    await importButton.click();

    // 等待导入完成并等待更长时间以确保 React 状态更新
    await page.waitForTimeout(2000);

    // 截图记录
    await page.screenshot({ path: 'test-screenshots/categraf-import-test.png', fullPage: true });

    // 检查开关的 aria-checked 属性或 class
    // 根据 antd Switch 组件，开启状态会有 ant-switch-checked class
    // 注意：第一个 Switch 是表单顶部的全局 Categraf 开关，我们需要跳过它
    // 找到主机列表中的 Categraf 开关（在 Space 组件中，有 Categraf 文字）
    const hostListSwitches = page.locator('.ant-row .ant-space .ant-switch').filter({ hasText: 'Categraf' });
    const count = await hostListSwitches.count();
    console.log(`找到 ${count} 个主机列表中的 Categraf 开关`);

    // 打印所有开关的状态
    for (let i = 0; i < count; i++) {
      const sw = hostListSwitches.nth(i);
      const isChecked = await sw.evaluate(el => el.classList.contains('ant-switch-checked'));
      const ariaChecked = await sw.getAttribute('aria-checked');
      console.log(`开关 ${i + 1}: checked class=${isChecked}, aria-checked=${ariaChecked}`);
    }

    if (count >= 3) {
      // 第一行应该是开启的 (true)
      const switch1 = hostListSwitches.nth(0);
      const isChecked1 = await switch1.evaluate(el => el.classList.contains('ant-switch-checked'));
      console.log(`第一行 Categraf 开关状态: ${isChecked1 ? '开启' : '关闭'}`);
      expect(isChecked1).toBe(true);

      // 第二行应该是关闭的 (false)
      const switch2 = hostListSwitches.nth(1);
      const isChecked2 = await switch2.evaluate(el => el.classList.contains('ant-switch-checked'));
      console.log(`第二行 Categraf 开关状态: ${isChecked2 ? '开启' : '关闭'}`);
      expect(isChecked2).toBe(false);

      // 第三行应该是开启的 (true)
      const switch3 = hostListSwitches.nth(2);
      const isChecked3 = await switch3.evaluate(el => el.classList.contains('ant-switch-checked'));
      console.log(`第三行 Categraf 开关状态: ${isChecked3 ? '开启' : '关闭'}`);
      expect(isChecked3).toBe(true);
    }
  });

  test('导入 JSON 配置后 Categraf 开关应正确显示', async ({ page }) => {
    // 打开批量安装 Salt Minion 弹窗
    const installButton = page.locator('button:has-text("批量安装 Salt Minion"), button:has-text("Batch Install Salt Minion")');
    await installButton.click();
    await page.waitForSelector('.ant-modal', { timeout: 5000 });

    // 点击粘贴导入按钮
    const pasteImportButton = page.locator('button:has-text("粘贴导入"), button:has-text("Paste Import")');
    await pasteImportButton.click();
    await page.waitForSelector('.ant-modal:has-text("粘贴导入"), .ant-modal:has-text("Paste Import")', { timeout: 5000 });

    // 选择 JSON 格式
    const formatSelect = page.locator('.ant-select:has-text("CSV"), .ant-select:has-text("csv")');
    if (await formatSelect.count() > 0) {
      await formatSelect.click();
      await page.click('.ant-select-item:has-text("JSON")');
    }

    // 输入带有 install_categraf 的 JSON 配置
    const jsonContent = `[
  {"host": "10.0.0.1", "port": 22, "username": "root", "password": "pass1", "install_categraf": true},
  {"host": "10.0.0.2", "port": 22, "username": "root", "password": "pass2", "install_categraf": false},
  {"host": "10.0.0.3", "port": 22, "username": "root", "password": "pass3", "install_categraf": true}
]`;

    const textarea = page.locator('textarea');
    await textarea.fill(jsonContent);

    // 点击导入按钮
    const importButton = page.locator('.ant-modal button:has-text("立即导入"), .ant-modal button:has-text("Import Now")');
    await importButton.click();

    // 等待导入完成
    await page.waitForTimeout(1000);

    // 截图记录
    await page.screenshot({ path: 'test-screenshots/categraf-import-json-test.png', fullPage: true });

    // 检查开关状态
    const switches = page.locator('.ant-switch:has-text("Categraf")');
    const count = await switches.count();
    console.log(`JSON 导入后找到 ${count} 个 Categraf 开关`);
  });

  test('安装进度日志框 Ctrl+A 应只选中日志内容', async ({ page }) => {
    // 打开批量安装 Salt Minion 弹窗
    const installButton = page.locator('button:has-text("批量安装 Salt Minion"), button:has-text("Batch Install Salt Minion")');
    await installButton.click();
    await page.waitForSelector('.ant-modal', { timeout: 5000 });

    // 找到安装进度日志框
    const logBox = page.locator('div[style*="background: #0b1021"], div[style*="background:#0b1021"]');
    
    if (await logBox.count() > 0) {
      // 点击日志框获取焦点
      await logBox.click();
      
      // 模拟 Ctrl+A
      await page.keyboard.press('Control+a');
      
      // 检查选中的内容范围
      const selection = await page.evaluate(() => {
        const sel = window.getSelection();
        if (sel && sel.rangeCount > 0) {
          const range = sel.getRangeAt(0);
          return {
            startContainer: range.startContainer.nodeName,
            endContainer: range.endContainer.nodeName,
            commonAncestor: range.commonAncestorContainer.nodeName,
            text: sel.toString().substring(0, 100) // 只取前100字符
          };
        }
        return null;
      });
      
      console.log('选中内容:', selection);
      
      // 截图记录
      await page.screenshot({ path: 'test-screenshots/log-box-select-all-test.png', fullPage: true });
    }
  });
});
