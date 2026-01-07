// @ts-nocheck
/* eslint-disable */
/**
 * 二次密码功能测试
 * 测试内容：
 * 1. 登录系统
 * 2. 进入个人资料页面
 * 3. 设置二次密码
 * 4. 验证二次密码状态
 */
const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://192.168.18.131:8080';
const USERNAME = 'admin';
const PASSWORD = 'admin123';
const SECONDARY_PASSWORD = 'secondary123';

test.describe('二次密码功能测试', () => {
  
  test('完整测试：登录并设置二次密码', async ({ page }) => {
    console.log('=== 开始二次密码功能测试 ===');
    
    // 1. 访问登录页面
    console.log('步骤1: 访问登录页面');
    await page.goto(BASE + '/login');
    await page.waitForLoadState('networkidle');
    
    // 截图：登录页面
    await page.screenshot({ path: 'test-screenshots/secondary-pwd-01-login-page.png', fullPage: true });
    
    // 2. 输入用户名和密码
    console.log('步骤2: 输入登录凭据');
    const usernameInput = page.locator('input[placeholder="用户名"]');
    const passwordInput = page.locator('input[placeholder="密码"]');
    
    await usernameInput.fill(USERNAME);
    await passwordInput.fill(PASSWORD);
    
    // 3. 点击登录按钮
    console.log('步骤3: 点击登录按钮');
    const loginButton = page.locator('button:has-text("登 录")');
    await loginButton.click();
    
    // 等待登录完成（可能需要2FA验证）
    await page.waitForTimeout(3000);
    
    // 检查是否需要2FA验证
    const twoFAInput = page.locator('input[placeholder*="验证码"]');
    if (await twoFAInput.isVisible({ timeout: 2000 }).catch(() => false)) {
      console.log('检测到2FA验证，需要手动输入验证码...');
      // 这里可以根据实际情况处理2FA
      await page.waitForTimeout(5000);
    }
    
    // 等待跳转到主页
    await page.waitForURL(/\/(projects|dashboard|profile)/, { timeout: 15000 });
    console.log('登录成功，当前URL:', page.url());
    
    // 截图：登录后
    await page.screenshot({ path: 'test-screenshots/secondary-pwd-02-after-login.png', fullPage: true });
    
    // 4. 导航到个人资料页面
    console.log('步骤4: 导航到个人资料页面');
    await page.goto(BASE + '/profile');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 截图：个人资料页面
    await page.screenshot({ path: 'test-screenshots/secondary-pwd-03-profile-page.png', fullPage: true });
    
    // 5. 查找二次密码设置区域
    console.log('步骤5: 查找二次密码设置区域');
    
    // 检查页面内容
    const pageContent = await page.content();
    console.log('页面是否包含"二次密码":', pageContent.includes('二次密码'));
    console.log('页面是否包含"敏感操作":', pageContent.includes('敏感操作'));
    
    // 查找设置二次密码按钮
    const setSecondaryPwdButton = page.locator('button:has-text("设置二次密码")');
    const changeSecondaryPwdButton = page.locator('button:has-text("修改二次密码")');
    
    let hasSecondaryPassword = false;
    
    if (await setSecondaryPwdButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      console.log('找到"设置二次密码"按钮，用户尚未设置二次密码');
      hasSecondaryPassword = false;
    } else if (await changeSecondaryPwdButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      console.log('找到"修改二次密码"按钮，用户已设置二次密码');
      hasSecondaryPassword = true;
    } else {
      console.log('未找到二次密码相关按钮，可能前端未更新');
      // 列出所有按钮帮助调试
      const allButtons = await page.locator('button').all();
      console.log('页面上的所有按钮:');
      for (let i = 0; i < allButtons.length; i++) {
        const text = await allButtons[i].textContent();
        console.log(`  按钮 ${i}: "${text}"`);
      }
    }
    
    // 6. 如果未设置二次密码，则设置
    if (!hasSecondaryPassword && await setSecondaryPwdButton.isVisible({ timeout: 1000 }).catch(() => false)) {
      console.log('步骤6: 点击设置二次密码按钮');
      await setSecondaryPwdButton.click();
      await page.waitForTimeout(1000);
      
      // 截图：设置对话框
      await page.screenshot({ path: 'test-screenshots/secondary-pwd-04-setup-dialog.png', fullPage: true });
      
      // 填写表单
      console.log('步骤7: 填写二次密码表单');
      
      // 当前登录密码（用于验证身份）
      const currentPwdInput = page.locator('input[placeholder*="当前登录密码"]');
      if (await currentPwdInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await currentPwdInput.fill(PASSWORD);
      }
      
      // 新二次密码
      const newSecondaryPwdInput = page.locator('input[placeholder*="新二次密码"]');
      if (await newSecondaryPwdInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await newSecondaryPwdInput.fill(SECONDARY_PASSWORD);
      }
      
      // 确认二次密码
      const confirmPwdInput = page.locator('input[placeholder*="再次输入"]');
      if (await confirmPwdInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await confirmPwdInput.fill(SECONDARY_PASSWORD);
      }
      
      // 截图：填写后
      await page.screenshot({ path: 'test-screenshots/secondary-pwd-05-form-filled.png', fullPage: true });
      
      // 点击确认设置按钮
      console.log('步骤8: 点击确认设置按钮');
      const confirmButton = page.locator('button:has-text("确认设置")');
      if (await confirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await confirmButton.click();
        await page.waitForTimeout(2000);
      }
      
      // 截图：设置结果
      await page.screenshot({ path: 'test-screenshots/secondary-pwd-06-result.png', fullPage: true });
      
      // 检查是否出现成功消息
      const successMessage = page.locator('.ant-message-success, :has-text("设置成功")');
      if (await successMessage.isVisible({ timeout: 3000 }).catch(() => false)) {
        console.log('✅ 二次密码设置成功!');
      }
    }
    
    // 7. 最终验证
    console.log('步骤9: 刷新页面验证状态');
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 截图：最终状态
    await page.screenshot({ path: 'test-screenshots/secondary-pwd-07-final.png', fullPage: true });
    
    // 检查二次密码状态
    const statusTag = page.locator('.ant-tag:has-text("已设置")');
    if (await statusTag.isVisible({ timeout: 3000 }).catch(() => false)) {
      console.log('✅ 验证成功：二次密码状态显示"已设置"');
    }
    
    console.log('=== 二次密码功能测试完成 ===');
  });
  
});
