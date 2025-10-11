// @ts-nocheck
/* eslint-disable */
// 调试登录页面
const { test } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

test('调试登录页面', async ({ page }) => {
  await page.goto(BASE + '/');
  
  // 等待页面加载
  await page.waitForLoadState('networkidle');
  
  // 截图
  await page.screenshot({ path: 'login-debug.png', fullPage: true });
  
  // 输出页面的 HTML
  const html = await page.content();
  console.log('页面 HTML 长度:', html.length);
  
  // 查找所有按钮
  const buttons = await page.locator('button').all();
  console.log('找到的按钮数量:', buttons.length);
  for (let i = 0; i < buttons.length; i++) {
    const text = await buttons[i].textContent();
    console.log(`按钮 ${i}: "${text}"`);
  }
  
  // 查找所有输入框
  const inputs = await page.locator('input').all();
  console.log('找到的输入框数量:', inputs.length);
  for (let i = 0; i < inputs.length; i++) {
    const placeholder = await inputs[i].getAttribute('placeholder');
    const type = await inputs[i].getAttribute('type');
    console.log(`输入框 ${i}: type=${type}, placeholder="${placeholder}"`);
  }
  
  // 等待一下看结果
  await page.waitForTimeout(1000);
});
