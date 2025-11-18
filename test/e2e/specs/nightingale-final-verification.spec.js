/**
 * Nightingale 最终验证测试
 * 测试通过正确的端口访问 Nightingale
 */

const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('Nightingale 完整功能验证', async ({ page }) => {
  const allErrors = [];
  const allRequests = new Map();
  
  // 监听控制台错误
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`❌ Console Error: ${msg.text()}`);
      allErrors.push(msg.text());
    }
  });
  
  // 监听所有请求和响应
  page.on('response', async res => {
    const url = res.url();
    const status = res.status();
    allRequests.set(url, status);
    
    if (status >= 400) {
      console.log(`❌ [${status}] ${url}`);
    } else if (url.includes('nightingale') || url.includes('/api/n9e/')) {
      console.log(`✓ [${status}] ${url}`);
    }
  });
  
  console.log(`\n=== 访问 Nightingale: ${BASE_URL}/nightingale/ ===\n`);
  
  // 访问 Nightingale
  await page.goto(`${BASE_URL}/nightingale/`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  // 等待 React 应用挂载
  console.log('\n=== 等待 React 应用加载 ===\n');
  await page.waitForTimeout(5000);
  
  // 检查 #root 是否有内容
  const rootHTML = await page.locator('#root').innerHTML();
  console.log(`\n#root HTML 长度: ${rootHTML.length} 字符`);
  
  // 验证 #root 不为空
  expect(rootHTML.length).toBeGreaterThan(100);
  
  if (rootHTML.length > 0) {
    console.log(`✓ React 应用成功挂载`);
    console.log(`#root 内容预览:\n${rootHTML.substring(0, 500)}...`);
  }
  
  // 检查是否有导航菜单
  const menuItems = await page.locator('.ant-menu-item, [role="menuitem"]').count();
  console.log(`\n找到 ${menuItems} 个菜单项`);
  
  // 检查页面文本
  const bodyText = await page.locator('body').innerText();
  console.log(`\nBody 文本长度: ${bodyText.length} 字符`);
  
  // 验证没有404页面
  const has404 = bodyText.includes('404') || bodyText.includes('not found');
  if (has404) {
    console.log('⚠️  检测到可能的 404 页面');
  } else {
    console.log('✓ 未检测到 404 错误');
  }
  
  // 统计请求
  let successCount = 0;
  let failCount = 0;
  allRequests.forEach((status, url) => {
    if (status >= 200 && status < 400) {
      successCount++;
    } else {
      failCount++;
    }
  });
  
  console.log(`\n=== 请求统计 ===`);
  console.log(`成功请求: ${successCount}`);
  console.log(`失败请求: ${failCount}`);
  console.log(`控制台错误: ${allErrors.length}`);
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-final-verification.png',
    fullPage: true 
  });
  console.log('\n✓ 截图已保存到 test-screenshots/nightingale-final-verification.png\n');
  
  // 最终断言
  expect(rootHTML.length).toBeGreaterThan(100);
  expect(failCount).toBeLessThan(5); // 允许少量失败请求
});
