import { test } from '@playwright/test';

test('debug monitoring page', async ({ page }) => {
  console.log('Navigating to /monitoring...');
  await page.goto('http://192.168.0.200:8080/monitoring', { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });

  // 等待一段时间让页面完全渲染
  await page.waitForTimeout(5000);
  
  // 截图
  await page.screenshot({ path: 'test-results/monitoring-debug.png', fullPage: true });
  
  // 打印 HTML 结构
  const html = await page.content();
  console.log('=== Page HTML (first 2000 chars) ===');
  console.log(html.substring(0, 2000));
  
  // 打印所有 iframe
  const iframes = await page.$$('iframe');
  console.log(`\n=== Found ${iframes.length} iframe(s) ===`);
  for (let i = 0; i < iframes.length; i++) {
    const title = await iframes[i].getAttribute('title');
    const src = await iframes[i].getAttribute('src');
    console.log(`Iframe ${i}: title="${title}", src="${src}"`);
  }
  
  // 打印所有包含 "monitoring" 的元素
  const monitoringElements = await page.$$('[class*="monitoring"], [id*="monitoring"]');
  console.log(`\n=== Found ${monitoringElements.length} elements with 'monitoring' ===`);
  
  // 检查是否有 ant design 元素
  const antElements = await page.$$('.ant-card, .ant-layout');
  console.log(`\n=== Found ${antElements.length} Ant Design elements ===`);
  
  // 检查 React root
  const root = await page.$('#root');
  const rootHTML = await root?.innerHTML();
  console.log('\n=== Root element HTML (first 1000 chars) ===');
  console.log(rootHTML?.substring(0, 1000));
});
