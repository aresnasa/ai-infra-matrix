#!/usr/bin/env node

/**
 * AI Infrastructure Matrix - Playwright登录和认证测试
 * 使用playwright-mcp测试登录流程、JupyterHub wrapper和iframe功能
 */

const { chromium, firefox, webkit } = require('playwright');
const fs = require('fs');
const path = require('path');

// 配置项（支持通过环境变量覆盖）
const config = {
    baseUrl: process.env.FRONTEND_URL || 'http://localhost:8080',
    timeout: Number(process.env.TIMEOUT || 30000),
    screenshot: (process.env.SCREENSHOT || 'true').toLowerCase() !== 'false',
    screenshotPath: process.env.SCREENSHOT_PATH || './test-screenshots',
    browsers: (process.env.BROWSERS ? process.env.BROWSERS.split(',').map(b => b.trim()) : ['chromium']),
    headless: (process.env.HEADLESS || process.env.PW_HEADLESS || 'true').toLowerCase() === 'true',
    slowMo: Number(process.env.SLOWMO || 0),
    username: process.env.TEST_USERNAME || 'admin',
    password: process.env.TEST_PASSWORD || 'admin123'
};

// 测试结果收集
const testResults = {
    passed: 0,
    failed: 0,
    details: []
};

// 日志函数
function log(level, message) {
    const timestamp = new Date().toISOString();
    const prefix = `[${timestamp}] [${level.toUpperCase()}]`;
    console.log(`${prefix} ${message}`);
}

// 创建截图目录
function createScreenshotDir() {
    if (config.screenshot && !fs.existsSync(config.screenshotPath)) {
        fs.mkdirSync(config.screenshotPath, { recursive: true });
        log('info', `创建截图目录: ${config.screenshotPath}`);
    }
}

// 截图函数
async function takeScreenshot(page, name, step = '') {
    if (!config.screenshot) return;
    
    try {
        const filename = `${name}${step ? '_' + step : ''}_${Date.now()}.png`;
        const filepath = path.join(config.screenshotPath, filename);
        await page.screenshot({ 
            path: filepath, 
            fullPage: true,
            type: 'png'
        });
        log('info', `截图保存: ${filepath}`);
        return filepath;
    } catch (error) {
        log('error', `截图失败: ${error.message}`);
    }
}

// 记录测试结果
function recordTestResult(testName, passed, message, url = null) {
    if (passed) {
        testResults.passed++;
        testResults.details.push({
            test: testName,
            status: 'PASSED',
            message: message,
            url: url
        });
        log('info', `✅ ${testName}: ${message}`);
    } else {
        testResults.failed++;
        testResults.details.push({
            test: testName,
            status: 'FAILED',
            message: message,
            url: url
        });
        log('error', `❌ ${testName}: ${message}`);
    }
}

// 测试1: 访问主页
async function testHomepage(page) {
    const testName = 'homepage_access';
    try {
        log('info', `测试主页访问: ${config.baseUrl}`);
        
        const response = await page.goto(config.baseUrl, { 
            waitUntil: 'networkidle',
            timeout: config.timeout 
        });
        
        if (!response.ok()) {
            throw new Error(`页面加载失败，HTTP状态: ${response.status()}`);
        }
        
        await page.waitForLoadState('domcontentloaded');
        await takeScreenshot(page, testName, 'loaded');
        
        const title = await page.title();
        log('info', `页面标题: ${title}`);
        
        recordTestResult(testName, true, `主页加载成功 (HTTP ${response.status()})`, config.baseUrl);
        return true;
        
    } catch (error) {
        await takeScreenshot(page, testName, 'error');
        recordTestResult(testName, false, error.message, config.baseUrl);
        return false;
    }
}

// 测试2: 用户登录
async function testLogin(page) {
    const testName = 'user_login';
    try {
        log('info', `测试用户登录: ${config.username}`);
        
        // 导航到登录页面
        const loginUrl = `${config.baseUrl}/login`;
        await page.goto(loginUrl, { waitUntil: 'networkidle' });
        
        await takeScreenshot(page, testName, 'login_page');
        
        // 查找并填写登录表单
        const usernameInput = await page.locator('input[name="username"], input[type="text"], input[placeholder*="用户名"], input[placeholder*="username"]').first();
        const passwordInput = await page.locator('input[name="password"], input[type="password"], input[placeholder*="密码"], input[placeholder*="password"]').first();
        
        await usernameInput.waitFor({ state: 'visible', timeout: 10000 });
        await passwordInput.waitFor({ state: 'visible', timeout: 10000 });
        
        await usernameInput.fill(config.username);
        await passwordInput.fill(config.password);
        
        await takeScreenshot(page, testName, 'form_filled');
        
        // 查找并点击登录按钮
        const loginButton = await page.locator('button[type="submit"], button:has-text("登录"), button:has-text("Login"), button:has-text("Sign in")').first();
        await loginButton.click();
        
        // 等待登录完成 - 检查token是否存在于localStorage
        await page.waitForTimeout(2000);
        
        const token = await page.evaluate(() => localStorage.getItem('token'));
        
        if (!token) {
            throw new Error('登录后未找到token');
        }
        
        await takeScreenshot(page, testName, 'logged_in');
        
        recordTestResult(testName, true, '用户登录成功，token已设置', loginUrl);
        return true;
        
    } catch (error) {
        await takeScreenshot(page, testName, 'error');
        recordTestResult(testName, false, error.message);
        return false;
    }
}

// 测试3: 使用API登录
async function testAPILogin(page) {
    const testName = 'api_login';
    try {
        log('info', '测试API登录');
        
        const response = await page.request.post(`${config.baseUrl}/api/auth/login`, {
            data: { 
                username: config.username, 
                password: config.password 
            },
            timeout: config.timeout
        });
        
        const status = response.status();
        if (status !== 200) {
            throw new Error(`登录API返回状态: ${status}`);
        }
        
        const data = await response.json();
        const token = data?.token;
        const expires_at = data?.expires_at || new Date(Date.now() + 3600_000).toISOString();
        
        if (!token) {
            throw new Error('登录返回未包含token');
        }
        
        // 将token写入localStorage
        await page.evaluate(({ t, exp }) => {
            localStorage.setItem('token', t);
            localStorage.setItem('token_expires', exp);
        }, { t: token, exp: expires_at });
        
        log('info', `API登录成功，token: ${token.substring(0, 20)}...`);
        
        recordTestResult(testName, true, 'API登录成功，token已设置');
        return { token, expires_at };
        
    } catch (error) {
        recordTestResult(testName, false, error.message);
        return null;
    }
}

// 测试4: 检查认证状态
async function testAuthStatus(page) {
    const testName = 'auth_status_check';
    try {
        log('info', '测试认证状态检查');
        
        const response = await page.request.get(`${config.baseUrl}/api/auth/me`);
        const status = response.status();
        
        if (status !== 200) {
            throw new Error(`认证检查返回状态: ${status}`);
        }
        
        const userData = await response.json();
        log('info', `当前用户: ${JSON.stringify(userData)}`);
        
        recordTestResult(testName, true, `认证状态有效，用户: ${userData.username || 'unknown'}`);
        return true;
        
    } catch (error) {
        recordTestResult(testName, false, error.message);
        return false;
    }
}

// 测试5: 访问JupyterHub wrapper页面
async function testJupyterHubWrapper(page) {
    const testName = 'jupyterhub_wrapper';
    try {
        log('info', '测试JupyterHub wrapper页面');
        
        const url = `${config.baseUrl}/jupyterhub`;
        const response = await page.goto(url, { 
            waitUntil: 'networkidle',
            timeout: config.timeout 
        });
        
        if (!response.ok()) {
            throw new Error(`JupyterHub wrapper加载失败，HTTP状态: ${response.status()}`);
        }
        
        await page.waitForLoadState('domcontentloaded');
        await page.waitForTimeout(3000); // 等待JavaScript执行
        
        await takeScreenshot(page, testName, 'loaded');
        
        // 检查页面内容
        const pageContent = await page.content();
        const hasJupyterHub = pageContent.toLowerCase().includes('jupyter');
        const hasIframe = pageContent.includes('iframe');
        
        log('info', `页面包含JupyterHub相关内容: ${hasJupyterHub}`);
        log('info', `页面包含iframe元素: ${hasIframe}`);
        
        if (!hasJupyterHub && !hasIframe) {
            throw new Error('JupyterHub wrapper页面缺少关键内容');
        }
        
        recordTestResult(testName, true, 'JupyterHub wrapper页面加载成功', url);
        return true;
        
    } catch (error) {
        await takeScreenshot(page, testName, 'error');
        recordTestResult(testName, false, error.message, `${config.baseUrl}/jupyterhub`);
        return false;
    }
}

// 测试6: 检查iframe元素
async function testIframeElement(page) {
    const testName = 'iframe_element_check';
    try {
        log('info', '测试iframe元素检查');
        
        // 查找iframe元素
        const iframeLocator = page.locator('iframe#jupyterhub-frame, iframe[title*="JupyterHub"], iframe');
        const iframeCount = await iframeLocator.count();
        
        if (iframeCount === 0) {
            throw new Error('未找到iframe元素');
        }
        
        log('info', `找到iframe元素数量: ${iframeCount}`);
        
        // 获取iframe的src属性
        const iframeSrc = await iframeLocator.first().getAttribute('src');
        log('info', `iframe src: ${iframeSrc}`);
        
        // 检查iframe是否可见
        const isVisible = await iframeLocator.first().isVisible();
        log('info', `iframe可见性: ${isVisible}`);
        
        await takeScreenshot(page, testName, 'iframe_found');
        
        if (!isVisible) {
            throw new Error('iframe存在但不可见');
        }
        
        recordTestResult(testName, true, `找到${iframeCount}个iframe元素，src: ${iframeSrc}`);
        return true;
        
    } catch (error) {
        await takeScreenshot(page, testName, 'error');
        recordTestResult(testName, false, error.message);
        return false;
    }
}

// 测试7: 验证localStorage中的token
async function testTokenInStorage(page) {
    const testName = 'token_storage_check';
    try {
        log('info', '测试localStorage中的token');
        
        const token = await page.evaluate(() => localStorage.getItem('token'));
        const tokenExpires = await page.evaluate(() => localStorage.getItem('token_expires'));
        
        if (!token) {
            throw new Error('localStorage中未找到token');
        }
        
        log('info', `Token长度: ${token.length}`);
        log('info', `Token过期时间: ${tokenExpires}`);
        
        // 验证token格式（JWT通常有3个部分）
        const tokenParts = token.split('.');
        if (tokenParts.length !== 3) {
            throw new Error(`Token格式不正确，部分数量: ${tokenParts.length}`);
        }
        
        recordTestResult(testName, true, `Token存在于localStorage，长度: ${token.length}`);
        return true;
        
    } catch (error) {
        recordTestResult(testName, false, error.message);
        return false;
    }
}

// 测试8: 检查API健康状态
async function testAPIHealth(page) {
    const testName = 'api_health_check';
    try {
        log('info', '测试API健康状态');
        
        const response = await page.request.get(`${config.baseUrl}/api/health`);
        const status = response.status();
        
        if (status !== 200) {
            throw new Error(`健康检查返回状态: ${status}`);
        }
        
        const data = await response.json();
        log('info', `健康检查响应: ${JSON.stringify(data)}`);
        
        recordTestResult(testName, true, `API健康状态正常 (${status})`);
        return true;
        
    } catch (error) {
        recordTestResult(testName, false, error.message);
        return false;
    }
}

// 测试9: 访问Projects页面（需要登录）
async function testProjectsPage(page) {
    const testName = 'projects_page_access';
    try {
        log('info', '测试Projects页面访问');
        
        const url = `${config.baseUrl}/projects`;
        const response = await page.goto(url, { 
            waitUntil: 'networkidle',
            timeout: config.timeout 
        });
        
        if (!response.ok()) {
            throw new Error(`Projects页面加载失败，HTTP状态: ${response.status()}`);
        }
        
        await page.waitForLoadState('domcontentloaded');
        await page.waitForTimeout(2000);
        
        await takeScreenshot(page, testName, 'loaded');
        
        // 检查是否被重定向到登录页面
        const currentUrl = page.url();
        if (currentUrl.includes('/login')) {
            throw new Error('被重定向到登录页面，认证可能失败');
        }
        
        recordTestResult(testName, true, 'Projects页面访问成功', url);
        return true;
        
    } catch (error) {
        await takeScreenshot(page, testName, 'error');
        recordTestResult(testName, false, error.message, `${config.baseUrl}/projects`);
        return false;
    }
}

// 测试10: JavaScript控制台错误检查
async function testConsoleErrors(page) {
    const testName = 'console_errors_check';
    const consoleErrors = [];
    
    // 监听控制台消息
    page.on('console', msg => {
        if (msg.type() === 'error') {
            consoleErrors.push(msg.text());
        }
    });
    
    try {
        log('info', '测试JavaScript控制台错误');
        
        // 访问主要页面并检查错误
        await page.goto(config.baseUrl, { waitUntil: 'networkidle' });
        await page.waitForTimeout(3000);
        
        if (consoleErrors.length > 0) {
            log('warn', `发现${consoleErrors.length}个控制台错误:`);
            consoleErrors.forEach(err => log('warn', `  - ${err}`));
            
            // 如果有严重错误则失败，否则通过但记录警告
            const hasCriticalError = consoleErrors.some(err => 
                err.includes('Failed to load') || 
                err.includes('404') || 
                err.includes('ERR_')
            );
            
            if (hasCriticalError) {
                throw new Error(`发现${consoleErrors.length}个控制台错误，包括严重错误`);
            } else {
                recordTestResult(testName, true, `发现${consoleErrors.length}个非严重控制台错误`);
            }
        } else {
            recordTestResult(testName, true, '未发现控制台错误');
        }
        
        return true;
        
    } catch (error) {
        recordTestResult(testName, false, error.message);
        return false;
    }
}

// 主测试流程
async function runTestSuite(browserType = 'chromium') {
    let browser;
    let context;
    let page;
    
    try {
        log('info', `启动浏览器: ${browserType}`);
        
        // 启动浏览器
        const browsers = { chromium, firefox, webkit };
        browser = await browsers[browserType].launch({
            headless: config.headless,
            slowMo: config.slowMo
        });
        
        // 创建上下文
        context = await browser.newContext({
            viewport: { width: 1280, height: 720 },
            userAgent: 'AI-Infra-Matrix-Playwright-Test/1.0'
        });
        
        // 创建页面
        page = await context.newPage();
        
        // 设置默认超时
        page.setDefaultTimeout(config.timeout);
        
        log('info', '开始执行测试套件...');
        log('info', '='.repeat(60));
        
        // 执行测试序列
        await testHomepage(page);
        await testAPIHealth(page);
        await testAPILogin(page);
        await testAuthStatus(page);
        await testTokenInStorage(page);
        await testJupyterHubWrapper(page);
        await testIframeElement(page);
        await testProjectsPage(page);
        await testConsoleErrors(page);
        
        log('info', '='.repeat(60));
        log('info', '测试套件执行完成');
        
    } catch (error) {
        log('error', `测试执行失败: ${error.message}`);
        recordTestResult('test_execution', false, error.message);
        
    } finally {
        // 清理资源
        if (page) await page.close();
        if (context) await context.close();
        if (browser) await browser.close();
    }
}

// 生成测试报告
function generateReport() {
    const total = testResults.passed + testResults.failed;
    const passRate = total > 0 ? (testResults.passed / total * 100).toFixed(2) : 0;
    
    console.log('\n' + '='.repeat(70));
    console.log('🧪 AI Infrastructure Matrix - 登录和认证测试报告');
    console.log('='.repeat(70));
    console.log(`📊 测试统计:`);
    console.log(`   总计: ${total}`);
    console.log(`   通过: ${testResults.passed} ✅`);
    console.log(`   失败: ${testResults.failed} ❌`);
    console.log(`   通过率: ${passRate}%`);
    console.log('');
    
    console.log('📋 详细结果:');
    testResults.details.forEach((detail, index) => {
        const icon = detail.status === 'PASSED' ? '✅' : '❌';
        const urlInfo = detail.url ? ` [${detail.url}]` : '';
        console.log(`   ${index + 1}. ${icon} ${detail.test}: ${detail.message}${urlInfo}`);
    });
    
    console.log('');
    console.log(`🖼️  截图保存路径: ${config.screenshotPath}`);
    console.log(`🌐 测试基础URL: ${config.baseUrl}`);
    console.log(`👤 测试账户: ${config.username}`);
    console.log('='.repeat(70));
}

// 主入口函数
async function main() {
    try {
        log('info', 'AI Infrastructure Matrix - 登录和认证Playwright测试启动');
        log('info', `测试配置: ${JSON.stringify(config, null, 2)}`);
        
        // 创建截图目录
        createScreenshotDir();
        
        // 运行测试
        for (const browserType of config.browsers) {
            log('info', `开始 ${browserType} 浏览器测试...`);
            await runTestSuite(browserType);
        }
        
        // 生成报告
        generateReport();
        
        // 退出代码
        process.exit(testResults.failed > 0 ? 1 : 0);
        
    } catch (error) {
        log('error', `测试执行失败: ${error.message}`);
        console.error(error);
        process.exit(1);
    }
}

// 启动测试
if (require.main === module) {
    main();
}

module.exports = {
    testHomepage,
    testLogin,
    testAPILogin,
    testAuthStatus,
    testJupyterHubWrapper,
    testIframeElement,
    testTokenInStorage,
    testAPIHealth,
    testProjectsPage,
    testConsoleErrors
};
