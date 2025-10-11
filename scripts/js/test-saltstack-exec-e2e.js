#!/usr/bin/env node

/**
 * SaltStack 命令执行功能测试
 * 
 * 基于 Playwright 的自动化测试脚本
 * 测试 SaltStack 页面的命令执行功能，特别是执行完成后状态的正确更新
 * 
 * 使用方法:
 * BASE_URL=http://192.168.0.200:8080 E2E_USER=admin E2E_PASS=admin123 node scripts/js/test-saltstack-exec-e2e.js
 */

const { chromium } = require('playwright');

const CONFIG = {
    baseURL: process.env.BASE_URL || 'http://192.168.0.200:8080',
    username: process.env.E2E_USER || 'admin',
    password: process.env.E2E_PASS || 'admin123',
    headless: process.env.HEADLESS !== 'false',
    timeout: 30000,
};

function log(message, type = 'info') {
    const timestamp = new Date().toLocaleTimeString();
    const prefix = {
        info: '📋',
        success: '✅',
        error: '❌',
        warning: '⚠️ ',
        test: '🧪'
    }[type] || 'ℹ️';
    console.log(`[${timestamp}] ${prefix} ${message}`);
}

async function loginIfNeeded(page) {
    log('检查登录状态...');
    await page.goto(CONFIG.baseURL + '/');
    
    const isLoggedIn = await page.locator('text=admin').isVisible().catch(() => false);
    
    if (!isLoggedIn) {
        log('需要登录，开始登录流程...', 'warning');
        const hasLoginTab = await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false);
        
        if (hasLoginTab) {
            await page.getByPlaceholder('用户名').fill(CONFIG.username);
            await page.getByPlaceholder('密码').fill(CONFIG.password);
            await page.getByRole('button', { name: '登录' }).click();
            
            await page.waitForURL(/\/(projects|dashboard|saltstack)?$/, { timeout: 10000 });
            log('登录成功', 'success');
        }
    } else {
        log('已经登录', 'success');
    }
}

async function testSaltStackPage(page) {
    log('测试 SaltStack 页面加载...', 'test');
    
    await page.goto(CONFIG.baseURL + '/saltstack');
    
    // 验证页面标题
    const titleVisible = await page.getByText('SaltStack 配置管理').isVisible();
    if (titleVisible) {
        log('页面标题显示正常', 'success');
    } else {
        log('页面标题未找到', 'error');
        return false;
    }
    
    // 验证关键元素
    const masterStatus = await page.getByText('Master状态').isVisible();
    const minionsStatus = await page.getByText('在线Minions').isVisible();
    const apiStatus = await page.getByText('API状态').isVisible();
    
    if (masterStatus && minionsStatus && apiStatus) {
        log('所有关键状态元素显示正常', 'success');
    } else {
        log('部分状态元素未找到', 'warning');
    }
    
    return true;
}

async function testCommandExecution(page) {
    log('测试命令执行功能...', 'test');
    
    // 打开执行命令对话框
    log('打开执行命令对话框...');
    await page.getByRole('button', { name: /执行命令/ }).click();
    await page.waitForTimeout(500);
    
    const dialogVisible = await page.getByText('执行自定义命令').isVisible();
    if (!dialogVisible) {
        log('执行命令对话框未打开', 'error');
        return false;
    }
    log('执行命令对话框已打开', 'success');
    
    // 输入测试命令
    log('输入测试命令...');
    const codeTextarea = page.getByLabel('代码');
    await codeTextarea.clear();
    await codeTextarea.fill('echo "Test from E2E Playwright"\nhostname\ndate');
    
    const targetInput = page.getByLabel('目标节点');
    await targetInput.clear();
    await targetInput.fill('*');
    
    log('开始执行命令...');
    const executeButton = page.getByRole('button', { name: /执 行/ });
    await executeButton.click();
    
    // 验证按钮进入 loading 状态
    await page.waitForTimeout(500);
    const isDisabled = await executeButton.isDisabled();
    if (isDisabled) {
        log('执行按钮正确进入 loading 状态', 'success');
    } else {
        log('执行按钮未进入 loading 状态', 'warning');
    }
    
    // 等待执行日志出现
    log('等待执行进度日志...');
    try {
        await page.locator('text=/step-log|step-done|complete/').waitFor({ timeout: 30000 });
        log('看到执行进度日志', 'success');
    } catch (e) {
        log('未看到执行进度日志', 'warning');
    }
    
    // 关键测试：等待执行完成后按钮恢复可用
    log('等待执行完成（按钮恢复可用）...这是修复的关键点');
    try {
        await executeButton.waitFor({ state: 'enabled', timeout: 35000 });
        log('✨ 执行完成！按钮已恢复可用状态 - 修复成功！', 'success');
    } catch (e) {
        log('执行超时或按钮未恢复可用 - 可能仍存在问题', 'error');
        return false;
    }
    
    // 验证完成消息
    const hasCompleteMessage = await page.locator('text=/执行完成|complete/').isVisible();
    if (hasCompleteMessage) {
        log('看到执行完成消息', 'success');
    }
    
    // 截图保存结果
    await page.screenshot({ 
        path: './test-results/saltstack-exec-completed.png',
        fullPage: true 
    });
    log('已保存执行完成状态截图', 'success');
    
    return true;
}

async function testMultipleExecutions(page) {
    log('测试连续多次执行...', 'test');
    
    const executeButton = page.getByRole('button', { name: /执 行/ });
    const codeTextarea = page.getByLabel('代码');
    
    // 第二次执行
    log('开始第二次执行...');
    await codeTextarea.clear();
    await codeTextarea.fill('echo "Second execution test"\nuptime');
    await executeButton.click();
    
    try {
        await executeButton.waitFor({ state: 'enabled', timeout: 35000 });
        log('第二次执行完成', 'success');
    } catch (e) {
        log('第二次执行失败', 'error');
        return false;
    }
    
    return true;
}

async function main() {
    log('========================================');
    log('SaltStack 命令执行 E2E 测试');
    log('========================================');
    log(`测试环境: ${CONFIG.baseURL}`);
    log(`测试用户: ${CONFIG.username}`);
    log(`Headless: ${CONFIG.headless}`);
    log('========================================\n');
    
    const browser = await chromium.launch({
        headless: CONFIG.headless,
        slowMo: 100,
    });
    
    const context = await browser.newContext({
        viewport: { width: 1920, height: 1080 },
    });
    
    const page = await context.newPage();
    page.setDefaultTimeout(CONFIG.timeout);
    
    let allTestsPassed = true;
    
    try {
        // 1. 登录
        await loginIfNeeded(page);
        
        // 2. 测试 SaltStack 页面
        const pageTestPassed = await testSaltStackPage(page);
        if (!pageTestPassed) {
            allTestsPassed = false;
        }
        
        // 3. 测试命令执行（主要测试）
        const execTestPassed = await testCommandExecution(page);
        if (!execTestPassed) {
            allTestsPassed = false;
        }
        
        // 4. 测试连续执行
        const multiExecPassed = await testMultipleExecutions(page);
        if (!multiExecPassed) {
            allTestsPassed = false;
        }
        
        // 关闭对话框
        await page.getByRole('button', { name: /关闭/ }).first().click();
        
    } catch (error) {
        log(`测试过程发生错误: ${error.message}`, 'error');
        allTestsPassed = false;
        
        // 保存错误截图
        await page.screenshot({ 
            path: './test-results/saltstack-exec-error.png',
            fullPage: true 
        });
    } finally {
        await context.close();
        await browser.close();
    }
    
    log('\n========================================');
    if (allTestsPassed) {
        log('所有测试通过！🎉', 'success');
        log('========================================\n');
        process.exit(0);
    } else {
        log('部分测试失败', 'error');
        log('========================================\n');
        process.exit(1);
    }
}

// 运行测试
if (require.main === module) {
    main().catch(err => {
        log(`致命错误: ${err.message}`, 'error');
        process.exit(1);
    });
}

module.exports = { CONFIG, loginIfNeeded, testSaltStackPage, testCommandExecution };
