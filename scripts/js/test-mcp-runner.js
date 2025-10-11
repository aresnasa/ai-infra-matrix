#!/usr/bin/env node

/**
 * AI Infrastructure Matrix - Playwright MCP测试运行器
 * 演示如何使用playwright-mcp进行UI测试
 */

const { chromium } = require('playwright');

// 测试配置
const config = {
    baseUrl: 'http://localhost:8080',
    testPageUrl: 'http://localhost:8080/test-page.html',
    headless: process.env.HEADLESS !== 'false',
    slowMo: Number(process.env.SLOWMO || 0),
};

// 测试结果
const results = {
    total: 0,
    passed: 0,
    failed: 0,
    tests: []
};

// 记录测试结果
function recordTest(name, passed, message) {
    results.total++;
    if (passed) {
        results.passed++;
        console.log(`✅ ${name}: ${message}`);
    } else {
        results.failed++;
        console.log(`❌ ${name}: ${message}`);
    }
    results.tests.push({ name, passed, message });
}

// 测试套件
async function runTests() {
    let browser;
    let page;
    
    try {
        console.log('🚀 启动Playwright MCP测试...');
        console.log('='.repeat(60));
        
        // 启动浏览器
        browser = await chromium.launch({
            headless: config.headless,
            slowMo: config.slowMo
        });
        
        const context = await browser.newContext({
            viewport: { width: 1280, height: 720 }
        });
        
        page = await context.newPage();
        
        // 测试1: 页面加载
        console.log('\n📋 测试1: 页面加载');
        try {
            await page.goto(config.testPageUrl, { waitUntil: 'networkidle' });
            const title = await page.title();
            recordTest('page_load', title.includes('AI Infrastructure Matrix'), 
                `页面标题: ${title}`);
        } catch (error) {
            recordTest('page_load', false, error.message);
        }
        
        // 测试2: 表单元素存在
        console.log('\n📋 测试2: 表单元素检查');
        try {
            const usernameInput = await page.locator('input[name="username"]').count();
            const passwordInput = await page.locator('input[name="password"]').count();
            const loginButton = await page.locator('button:has-text("登录")').count();
            
            const allElementsPresent = usernameInput > 0 && passwordInput > 0 && loginButton > 0;
            recordTest('form_elements', allElementsPresent, 
                `找到 ${usernameInput} 个用户名输入框, ${passwordInput} 个密码输入框, ${loginButton} 个登录按钮`);
        } catch (error) {
            recordTest('form_elements', false, error.message);
        }
        
        // 测试3: 执行登录
        console.log('\n📋 测试3: 登录功能');
        try {
            await page.fill('input[name="username"]', 'admin');
            await page.fill('input[name="password"]', 'admin123');
            await page.click('button:has-text("登录")');
            
            // 等待登录完成
            await page.waitForTimeout(1500);
            
            // 检查是否显示成功消息
            const successMessage = await page.locator('text=✅ 登录成功').isVisible();
            recordTest('login_action', successMessage, '登录功能正常，显示成功消息');
        } catch (error) {
            recordTest('login_action', false, error.message);
        }
        
        // 测试4: 验证localStorage
        console.log('\n📋 测试4: localStorage验证');
        try {
            const storageData = await page.evaluate(() => {
                return {
                    token: localStorage.getItem('token'),
                    username: localStorage.getItem('username'),
                    expires: localStorage.getItem('token_expires')
                };
            });
            
            const hasValidData = !!storageData.token && !!storageData.username;
            recordTest('localstorage_check', hasValidData, 
                `Token: ${storageData.token ? '存在' : '不存在'}, 用户名: ${storageData.username}`);
        } catch (error) {
            recordTest('localstorage_check', false, error.message);
        }
        
        // 测试5: 用户信息显示
        console.log('\n📋 测试5: 用户信息显示');
        try {
            const userInfoVisible = await page.locator('#userInfo').isVisible();
            const usernameDisplayed = await page.locator('#displayUsername').textContent();
            
            recordTest('user_info_display', userInfoVisible && usernameDisplayed === 'admin',
                `用户信息可见，显示用户名: ${usernameDisplayed}`);
        } catch (error) {
            recordTest('user_info_display', false, error.message);
        }
        
        // 测试6: iframe元素存在
        console.log('\n📋 测试6: iframe元素检查');
        try {
            const iframeCount = await page.locator('iframe#jupyterhub-frame').count();
            const iframeVisible = iframeCount > 0 ? 
                await page.locator('iframe#jupyterhub-frame').isVisible() : false;
            
            recordTest('iframe_check', iframeCount > 0 && iframeVisible,
                `找到 ${iframeCount} 个iframe，可见性: ${iframeVisible}`);
        } catch (error) {
            recordTest('iframe_check', false, error.message);
        }
        
        // 测试7: 认证检查功能
        console.log('\n📋 测试7: 认证检查功能');
        try {
            await page.click('button:has-text("检查认证")');
            await page.waitForTimeout(500);
            
            const authMessage = await page.locator('text=认证状态有效').isVisible();
            recordTest('auth_check', authMessage, '认证检查功能正常');
        } catch (error) {
            recordTest('auth_check', false, error.message);
        }
        
        // 测试8: 登出功能
        console.log('\n📋 测试8: 登出功能');
        try {
            await page.click('button:has-text("登出")');
            await page.waitForTimeout(500);
            
            const logoutMessage = await page.locator('text=已登出').isVisible();
            
            // 验证localStorage已清空
            const storageEmpty = await page.evaluate(() => {
                return !localStorage.getItem('token');
            });
            
            recordTest('logout', logoutMessage && storageEmpty, 
                '登出成功，localStorage已清空');
        } catch (error) {
            recordTest('logout', false, error.message);
        }
        
        // 测试9: 控制台错误检查
        console.log('\n📋 测试9: JavaScript控制台错误');
        const consoleErrors = [];
        page.on('console', msg => {
            if (msg.type() === 'error') {
                consoleErrors.push(msg.text());
            }
        });
        
        await page.reload({ waitUntil: 'networkidle' });
        await page.waitForTimeout(2000);
        
        recordTest('console_errors', consoleErrors.length === 0,
            consoleErrors.length === 0 ? '无控制台错误' : `发现 ${consoleErrors.length} 个错误`);
        
        console.log('\n' + '='.repeat(60));
        
    } catch (error) {
        console.error('❌ 测试执行失败:', error.message);
        results.failed++;
    } finally {
        if (browser) {
            await browser.close();
        }
    }
}

// 生成测试报告
function generateReport() {
    console.log('\n' + '='.repeat(60));
    console.log('📊 测试报告');
    console.log('='.repeat(60));
    console.log(`总测试数: ${results.total}`);
    console.log(`通过: ${results.passed} ✅`);
    console.log(`失败: ${results.failed} ❌`);
    
    if (results.total > 0) {
        const passRate = (results.passed / results.total * 100).toFixed(2);
        console.log(`通过率: ${passRate}%`);
    }
    
    console.log('\n详细结果:');
    results.tests.forEach((test, index) => {
        const icon = test.passed ? '✅' : '❌';
        console.log(`  ${index + 1}. ${icon} ${test.name}: ${test.message}`);
    });
    
    console.log('='.repeat(60));
}

// 主函数
async function main() {
    console.log('🧪 AI Infrastructure Matrix - Playwright MCP测试');
    console.log(`测试URL: ${config.testPageUrl}`);
    console.log(`无头模式: ${config.headless}`);
    console.log('');
    
    await runTests();
    generateReport();
    
    // 退出代码
    process.exit(results.failed > 0 ? 1 : 0);
}

// 执行测试
if (require.main === module) {
    main().catch(error => {
        console.error('❌ 测试执行异常:', error);
        process.exit(1);
    });
}

module.exports = { runTests, generateReport };
