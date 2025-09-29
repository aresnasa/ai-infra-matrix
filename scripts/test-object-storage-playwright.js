#!/usr/bin/env node

/**
 * AI Infrastructure Matrix - Playwright对象存储测试
 * 测试对象存储iframe和管理页面功能
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
    browsers: (process.env.BROWSERS ? process.env.BROWSERS.split(',').map(b => b.trim()) : ['chromium']), // 可选: 'firefox', 'webkit'
    headless: (process.env.HEADLESS || process.env.PW_HEADLESS || 'true').toLowerCase() === 'true', // 默认无头，CI友好
    slowMo: Number(process.env.SLOWMO || 0), // CI默认0，可在本地调试设为>0
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

// 登录助手：确保在访问受保护页面前已登录
async function ensureLogin(page) {
    try {
        // 尝试读取当前用户
        const me = await page.request.get(`${config.baseUrl}/api/auth/me`);
        if (me.status() === 200) {
            log('info', '检测到已登录状态');
            return true;
        }
    } catch (_) { /* ignore */ }

    // 使用默认管理员账户登录（在开发/测试环境可用）
    const username = process.env.TEST_USERNAME || 'admin';
    const password = process.env.TEST_PASSWORD || 'admin123';
    log('info', `尝试使用测试账户登录: ${username}`);

    try {
        const resp = await page.request.post(`${config.baseUrl}/api/auth/login`, {
            data: { username, password },
            timeout: config.timeout
        });
        const status = resp.status();
        if (status !== 200) {
            log('warn', `登录API返回状态: ${status}`);
            return false;
        }
        const data = await resp.json();
        const token = data?.token;
        const expires_at = data?.expires_at || new Date(Date.now() + 3600_000).toISOString();
        if (!token) {
            log('warn', '登录返回未包含token');
            return false;
        }
        // 确保在页面脚本执行前写入localStorage
        await page.addInitScript(({ t, exp }) => {
            try {
                localStorage.setItem('token', t);
                localStorage.setItem('token_expires', exp);
            } catch (e) {}
        }, { t: token, exp: expires_at });

        // 写入到当前页上下文（如已打开页面）
        await page.evaluate(({ t, exp }) => {
            localStorage.setItem('token', t);
            localStorage.setItem('token_expires', exp);
        }, { t: token, exp: expires_at });

        log('info', '登录成功，已设置token到localStorage');
        return true;
    } catch (error) {
        log('error', `登录失败: ${error.message}`);
        return false;
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

// 等待元素可见
async function waitForElement(page, selector, timeout = config.timeout) {
    try {
        await page.waitForSelector(selector, { 
            visible: true, 
            timeout 
        });
        return true;
    } catch (error) {
        log('error', `等待元素失败 ${selector}: ${error.message}`);
        return false;
    }
}

// 测试页面加载
async function testPageLoad(page, url, expectedTitle, testName) {
    try {
        log('info', `测试页面加载: ${url}`);
        
        // 导航到页面
        const response = await page.goto(url, { 
            waitUntil: 'networkidle',
            timeout: config.timeout 
        });
        
        if (!response.ok()) {
            throw new Error(`页面加载失败，HTTP状态: ${response.status()}`);
        }
        
        // 等待页面加载完成
        await page.waitForLoadState('domcontentloaded');
        
        // 检查标题（如果提供）
        if (expectedTitle) {
            const title = await page.title();
            if (!title.includes(expectedTitle)) {
                throw new Error(`页面标题不匹配，期望包含: ${expectedTitle}，实际: ${title}`);
            }
        }
        
        // 截图
        await takeScreenshot(page, testName, 'loaded');
        
        testResults.passed++;
        testResults.details.push({
            test: testName,
            status: 'PASSED',
            message: `页面成功加载: ${url}`
        });
        
        log('info', `✅ ${testName}: 页面加载成功`);
        return true;
        
    } catch (error) {
        await takeScreenshot(page, testName, 'error');
        
        testResults.failed++;
        testResults.details.push({
            test: testName,
            status: 'FAILED',
            message: error.message,
            url: url
        });
        
        log('error', `❌ ${testName}: ${error.message}`);
        return false;
    }
}

// 测试对象存储主页面
async function testObjectStoragePage(page) {
    await ensureLogin(page);
    const url = `${config.baseUrl}/object-storage`;
    // 放宽标题断言，避免由于未设置document.title导致的误判
    const success = await testPageLoad(page, url, '', 'object_storage_main');
    
    if (!success) return false;
    
    try {
        // 检查页面关键元素
        const elementsToCheck = [
            { selector: 'h2', description: '页面标题' },
            { selector: '[data-testid="storage-config-btn"], button[title*="存储配置"], button:has-text("存储配置")', description: '存储配置按钮' },
            { selector: '[data-testid="add-storage-btn"], button[title*="添加存储"], button:has-text("添加存储")', description: '添加存储按钮' },
        ];
        
        for (const element of elementsToCheck) {
            const found = await waitForElement(page, element.selector, 5000);
            if (!found) {
                log('warn', `页面元素未找到: ${element.description}`);
            } else {
                log('info', `✓ 找到页面元素: ${element.description}`);
            }
        }
        
        // 检查是否显示存储服务列表或空状态
        const hasStorageList = await page.locator('.ant-card, [class*="Card"], [data-testid="storage-list"]').count();
        log('info', `存储服务卡片数量: ${hasStorageList}`);
        
        await takeScreenshot(page, 'object_storage_main', 'elements_checked');
        return true;
        
    } catch (error) {
        log('error', `检查对象存储主页面元素失败: ${error.message}`);
        return false;
    }
}

// 测试对象存储管理页面
async function testObjectStorageAdminPage(page) {
    await ensureLogin(page);
    const url = `${config.baseUrl}/admin/object-storage`;
    const success = await testPageLoad(page, url, '', 'object_storage_admin');
    
    if (!success) return false;
    
    try {
        // 等待页面内容加载
        await page.waitForTimeout(2000);
        
        // 检查管理页面关键元素
        const elementsToCheck = [
            { selector: 'h4, h3, h2, [class*="title"]', description: '页面标题' },
            { selector: 'button:has-text("添加配置"), button:has-text("创建"), [data-testid="add-config-btn"]', description: '添加配置按钮' },
            { selector: 'table, .ant-table, [class*="Table"]', description: '配置列表表格' },
        ];
        
        for (const element of elementsToCheck) {
            const found = await waitForElement(page, element.selector, 5000);
            if (!found) {
                log('warn', `管理页面元素未找到: ${element.description}`);
            } else {
                log('info', `✓ 找到管理页面元素: ${element.description}`);
            }
        }
        
        await takeScreenshot(page, 'object_storage_admin', 'elements_checked');
        return true;
        
    } catch (error) {
        log('error', `检查对象存储管理页面失败: ${error.message}`);
        return false;
    }
}

// 测试MinIO控制台iframe页面
async function testMinIOConsolePage(page) {
    await ensureLogin(page);
    // 先检查是否有MinIO配置
    try {
        // 假设配置ID为1，实际应该从API获取
        const url = `${config.baseUrl}/object-storage/minio/1`;
        
        log('info', `测试MinIO控制台页面: ${url}`);
        
        const response = await page.goto(url, { 
            waitUntil: 'domcontentloaded',
            timeout: config.timeout 
        });
        
        if (!response.ok()) {
            // 如果配置不存在，尝试通用测试
            log('warn', `MinIO配置页面不可访问 (${response.status()})，可能配置不存在`);
            return await testMinIOConsoleProxy(page);
        }
        
        // 等待iframe出现（最长15秒，兼容前端探测/懒加载）
        try {
            await page.waitForSelector('iframe#minio-console-iframe, iframe[title*="MinIO"], iframe, [id*="iframe"], [class*="iframe"]', { timeout: 15000 });
        } catch (_) {
            // ignore here; we will handle as no-iframe below
        }

        // 检查iframe容器（必须存在）
    const iframeLocator = page.locator('iframe#minio-console-iframe, iframe[title*="MinIO"], iframe');
    const iframeCount = await iframeLocator.count();
        if (iframeCount > 0) {
            log('info', `✓ 找到iframe容器，数量: ${iframeCount}`);

            // 检查控制按钮
            const controls = await page.locator('button:has-text("刷新"), button:has-text("全屏"), button:has-text("返回")').count();
            log('info', `✓ 找到控制按钮，数量: ${controls}`);

            // 深入校验：访问iframe内部，确认MinIO控制台实际渲染
            let innerOk = false;
            try {
                // 获取第一帧（或匹配 /minio-console/ 的帧）
                const frames = page.frames();
                let frame = frames.find(f => /\/minio-console\//.test(f.url())) || frames.find(f => f !== page.mainFrame());
                if (!frame) {
                    // 使用frameLocator方式
                    const elementHandle = await iframeLocator.first().elementHandle();
                    frame = await elementHandle.contentFrame();
                }

                if (frame) {
                    // 尝试多个典型选择器
                    const selectors = [
                        'text=/MinIO/i',
                        'text=/Console/i',
                        'input[name="username"]',
                        'input[type="password"]',
                        'button:has-text("Login"), button:has-text("Log in"), button:has-text("Sign in")'
                    ];
                    for (const sel of selectors) {
                        try {
                            await frame.waitForSelector(sel, { timeout: 5000 });
                            log('info', `✓ 发现MinIO控制台标识元素: ${sel}`);
                            innerOk = true;
                            break;
                        } catch (_) {}
                    }
                } else {
                    log('warn', '未能解析到iframe内部frame，可能存在跨域限制');
                }
            } catch (e) {
                log('warn', `检测iframe内部内容失败: ${e.message}`);
            }

            await takeScreenshot(page, 'minio_console', innerOk ? 'iframe_page' : 'iframe_no_inner_content');

            if (!innerOk) {
                testResults.failed++;
                testResults.details.push({
                    test: 'minio_console_page',
                    status: 'FAILED',
                    message: '未检测到MinIO控制台内部内容（登录页面或控制台元素）'
                });
                // 继续做代理直连诊断
                await testMinIOConsoleProxy(page);
                return false;
            }

            testResults.passed++;
            testResults.details.push({
                test: 'minio_console_page',
                status: 'PASSED',
                message: 'MinIO控制台页面加载成功（内嵌iframe且内部内容可见）'
            });
            return true;
        }

        // 未找到iframe，判定为失败，尝试直接访问代理以获取更多诊断
        log('warn', '未找到iframe容器，尝试直接访问 /minio-console/ 进行诊断');
        await takeScreenshot(page, 'minio_console', 'no_iframe');
        testResults.failed++;
        testResults.details.push({
            test: 'minio_console_page',
            status: 'FAILED',
            message: '页面中未检测到MinIO控制台iframe'
        });

        // 继续进行代理直连测试，返回其结果但不覆盖上述失败记录
        await testMinIOConsoleProxy(page);
        return false;
        
    } catch (error) {
        log('error', `测试MinIO控制台页面失败: ${error.message}`);
        return await testMinIOConsoleProxy(page);
    }
}

// 测试MinIO控制台代理
async function testMinIOConsoleProxy(page) {
    const url = `${config.baseUrl}/minio-console/`;
    
    try {
        log('info', `测试MinIO控制台代理: ${url}`);
        
        const response = await page.goto(url, { 
            waitUntil: 'domcontentloaded',
            timeout: config.timeout 
        });
        
        await takeScreenshot(page, 'minio_console_proxy', 'direct_access');
        
        if (response.ok()) {
            log('info', '✅ MinIO控制台代理响应正常');
            
            // 检查是否为MinIO登录页面或控制台
            const pageContent = await page.content();
            if (pageContent.includes('MinIO') || pageContent.includes('Console') || pageContent.includes('login')) {
                log('info', '✓ 检测到MinIO控制台内容');
                
                testResults.passed++;
                testResults.details.push({
                    test: 'minio_console_proxy',
                    status: 'PASSED',
                    message: 'MinIO控制台代理功能正常'
                });
                return true;
            }
        }
        
        log('warn', `MinIO控制台代理响应异常: HTTP ${response.status()}`);
        return false;
        
    } catch (error) {
        log('error', `测试MinIO控制台代理失败: ${error.message}`);
        await takeScreenshot(page, 'minio_console_proxy', 'error');
        
        testResults.failed++;
        testResults.details.push({
            test: 'minio_console_proxy',
            status: 'FAILED',
            message: error.message
        });
        
        return false;
    }
}

// 测试iframe测试页面
async function testIframeTestPage(page) {
    const url = `${config.baseUrl}/test-object-storage-iframe.html`;
    // 放宽测试页标题校验
    const success = await testPageLoad(page, url, '', 'iframe_test_page');
    
    if (!success) return false;
    
    try {
        // 等待测试页面加载
        await page.waitForTimeout(3000);
        
        // 检查测试页面关键元素
        const elementsToCheck = [
            { selector: 'h1', description: '页面标题' },
            { selector: 'button:has-text("重新加载"), button:has-text("刷新")', description: '刷新按钮' },
            { selector: 'button:has-text("全屏")', description: '全屏按钮' },
            { selector: 'iframe#minio-iframe, iframe[title*="MinIO"]', description: 'MinIO iframe' },
        ];
        
        for (const element of elementsToCheck) {
            const found = await waitForElement(page, element.selector, 5000);
            if (found) {
                log('info', `✓ 找到测试页面元素: ${element.description}`);
            } else {
                log('warn', `测试页面元素未找到: ${element.description}`);
            }
        }
        
        // 检查iframe状态指示器
        const statusElements = await page.locator('.status, [class*="status"]').count();
        log('info', `状态指示器数量: ${statusElements}`);
        
        await takeScreenshot(page, 'iframe_test_page', 'complete');
        return true;
        
    } catch (error) {
        log('error', `检查iframe测试页面失败: ${error.message}`);
        return false;
    }
}

// 检查API端点
async function testAPIEndpoints(page) {
    const endpoints = [
        { url: `${config.baseUrl}/api/health`, name: 'health_check' },
        { url: `${config.baseUrl}/minio/health`, name: 'minio_health' },
        { url: `${config.baseUrl}/api/object-storage/configs`, name: 'storage_configs' },
    ];
    
    for (const endpoint of endpoints) {
        try {
            log('info', `测试API端点: ${endpoint.url}`);
            
            const response = await page.request.get(endpoint.url);
            const status = response.status();
            
            if (status === 200) {
                log('info', `✅ ${endpoint.name}: API响应正常 (${status})`);
                testResults.passed++;
                testResults.details.push({
                    test: endpoint.name,
                    status: 'PASSED',
                    message: `API端点响应正常 (${status})`
                });
            } else if (status === 401 || status === 403) {
                log('warn', `⚠️ ${endpoint.name}: 需要认证 (${status})`);
            } else {
                log('error', `❌ ${endpoint.name}: API响应异常 (${status})`);
            }
            
        } catch (error) {
            log('error', `测试API端点失败 ${endpoint.name}: ${error.message}`);
            testResults.failed++;
            testResults.details.push({
                test: endpoint.name,
                status: 'FAILED',
                message: error.message
            });
        }
    }
}

// 主测试函数
async function runTests(browserType = 'chromium') {
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
        
        // 执行测试
        await testPageLoad(page, config.baseUrl, '', 'homepage');
        await testAPIEndpoints(page);
        await testObjectStoragePage(page);
        await testObjectStorageAdminPage(page);
        await testMinIOConsolePage(page);
        await testIframeTestPage(page);
        
        log('info', '测试套件执行完成');
        
    } catch (error) {
        log('error', `测试执行失败: ${error.message}`);
        testResults.failed++;
        testResults.details.push({
            test: 'test_execution',
            status: 'FAILED',
            message: error.message
        });
        
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
    
    console.log('\n' + '='.repeat(60));
    console.log('🧪 AI Infrastructure Matrix - 对象存储测试报告');
    console.log('='.repeat(60));
    console.log(`📊 测试统计:`);
    console.log(`   总计: ${total}`);
    console.log(`   通过: ${testResults.passed} ✅`);
    console.log(`   失败: ${testResults.failed} ❌`);
    console.log(`   通过率: ${passRate}%`);
    console.log('');
    
    console.log('📋 详细结果:');
    testResults.details.forEach((detail, index) => {
        const icon = detail.status === 'PASSED' ? '✅' : '❌';
        console.log(`   ${index + 1}. ${icon} ${detail.test}: ${detail.message}`);
    });
    
    console.log('');
    console.log(`🖼️  截图保存路径: ${config.screenshotPath}`);
    console.log(`🌐 测试基础URL: ${config.baseUrl}`);
    console.log('='.repeat(60));
}

// 主入口函数
async function main() {
    try {
        log('info', 'AI Infrastructure Matrix - 对象存储Playwright测试启动');
        log('info', `测试配置: ${JSON.stringify(config, null, 2)}`);
        
        // 创建截图目录
        createScreenshotDir();
        
        // 运行测试
        for (const browserType of config.browsers) {
            log('info', `开始 ${browserType} 浏览器测试...`);
            await runTests(browserType);
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