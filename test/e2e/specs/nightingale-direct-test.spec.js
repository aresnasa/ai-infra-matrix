/**
 * 直接打开 Nightingale URL 测试（不通过前端iframe）
 */

const { test } = require('@playwright/test');

test('直接访问 Nightingale 测试', async ({ page }) => {
  const allErrors = [];
  const allRequests = [];
  
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`❌ Console Error: ${msg.text()}`);
      allErrors.push(msg.text());
    }
  });
  
  page.on('request', req => {
    allRequests.push(req.url());
  });
  
  page.on('response', async res => {
    const url = res.url();
    const status = res.status();
    
    if (status >= 400) {
      console.log(`❌ [${status}] ${url}`);
    } else if (url.includes('nightingale')) {
      console.log(`✓ [${status}] ${url}`);
    }
  });
  
  console.log('\n=== 直接访问 Nightingale ===\n');
  await page.goto('http://192.168.0.200:8080/nightingale/');
  
  console.log('\n=== 等待 30 秒 ===\n');
  await page.waitForTimeout(30000);
  
  // 检查 #root
  const rootHTML = await page.locator('#root').innerHTML();
  console.log(`\n#root HTML 长度: ${rootHTML.length} 字符`);
  
  if (rootHTML.length > 0) {
    console.log(`#root 内容预览:\n${rootHTML.substring(0, 500)}`);
  } else {
    console.log('#root 是空的 - React应用没有挂载！');
  }
  
  // 检查body文本
  const bodyText = await page.locator('body').innerText();
  console.log(`\nBody 文本:\n${bodyText}`);
  
  console.log(`\n总请求数: ${allRequests.length}`);
  console.log(`总错误数: ${allErrors.length}\n`);
  
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-direct-access.png',
    fullPage: true 
  });
  console.log('截图已保存\n');
});
