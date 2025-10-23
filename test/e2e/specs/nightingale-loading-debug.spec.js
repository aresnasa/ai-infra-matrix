/**
 * 调试 Nightingale 加载问题
 */

const { test } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test('调试 Nightingale 加载卡住问题', async ({ page }) => {
  const errors = [];
  const failedRequests = [];
  
  // 监听所有控制台消息
  page.on('console', msg => {
    const text = msg.text();
    console.log(`[Console ${msg.type()}] ${text}`);
    if (msg.type() === 'error') {
      errors.push(text);
    }
  });
  
  // 监听所有失败的请求
  page.on('response', async res => {
    const url = res.url();
    const status = res.status();
    
    if (status >= 400) {
      console.log(`❌ [${status}] ${url}`);
      failedRequests.push({ url, status });
      
      // 如果是 API 请求，打印响应内容
      if (url.includes('/api/')) {
        try {
          const body = await res.text();
          console.log(`   Response body: ${body.substring(0, 200)}`);
        } catch (e) {
          console.log(`   Could not read response body`);
        }
      }
    }
  });
  
  // 监听页面错误
  page.on('pageerror', error => {
    console.log(`❌ Page Error: ${error.message}`);
    errors.push(error.message);
  });
  
  console.log(`\n=== 访问 Nightingale ===\n`);
  await page.goto(`${BASE_URL}/nightingale/`, { 
    waitUntil: 'networkidle',
    timeout: 30000 
  });
  
  console.log('\n=== 等待 10 秒观察 ===\n');
  await page.waitForTimeout(10000);
  
  // 检查预加载动画是否还在
  const preloaderVisible = await page.locator('.preloader').isVisible().catch(() => false);
  console.log(`\n预加载动画是否可见: ${preloaderVisible}`);
  
  // 检查 #root 内容
  const rootHTML = await page.locator('#root').innerHTML();
  console.log(`\n#root 内容长度: ${rootHTML.length} 字符`);
  
  if (rootHTML.length > 0) {
    console.log(`#root 内容预览:\n${rootHTML.substring(0, 300)}...`);
  } else {
    console.log('⚠️ #root 为空，React 应用未挂载');
  }
  
  // 检查是否有 .App 类
  const appDiv = await page.locator('.App').count();
  console.log(`\n找到 .App 元素: ${appDiv} 个`);
  
  // 获取所有加载的 script 标签
  const scripts = await page.locator('script[src]').count();
  console.log(`\n加载的 script 标签数量: ${scripts}`);
  
  // 打印所有脚本 src
  for (let i = 0; i < scripts; i++) {
    const src = await page.locator('script[src]').nth(i).getAttribute('src');
    console.log(`  Script ${i + 1}: ${src}`);
  }
  
  // 检查是否有 JavaScript 运行时错误
  console.log(`\n=== 错误统计 ===`);
  console.log(`控制台错误数量: ${errors.length}`);
  console.log(`失败请求数量: ${failedRequests.length}`);
  
  if (errors.length > 0) {
    console.log('\n控制台错误列表:');
    errors.forEach((err, idx) => {
      console.log(`  ${idx + 1}. ${err}`);
    });
  }
  
  if (failedRequests.length > 0) {
    console.log('\n失败请求列表:');
    failedRequests.forEach((req, idx) => {
      console.log(`  ${idx + 1}. [${req.status}] ${req.url}`);
    });
  }
  
  // 尝试执行 JavaScript 检查全局变量
  const hasReact = await page.evaluate(() => typeof window.React !== 'undefined');
  const hasReactDOM = await page.evaluate(() => typeof window.ReactDOM !== 'undefined');
  console.log(`\nReact 已加载: ${hasReact}`);
  console.log(`ReactDOM 已加载: ${hasReactDOM}`);
  
  // 检查是否有挂载点
  const rootExists = await page.evaluate(() => document.getElementById('root') !== null);
  console.log(`#root 元素存在: ${rootExists}`);
  
  // 截图
  await page.screenshot({ 
    path: 'test-screenshots/nightingale-loading-stuck.png',
    fullPage: true 
  });
  console.log('\n✓ 截图已保存\n');
});
