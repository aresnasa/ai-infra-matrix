/**
 * AI Infrastructure Matrix - SLURM & SaltStack 页面自动化测试
 * 
 * 测试目标：
 * 1. 登录系统
 * 2. 访问 SLURM 页面，检查是否有"部分数据加载失败"错误
 * 3. 访问 SaltStack 页面，测试脚本执行是否正常完成
 * 
 * 使用方法：
 * node scripts/test-slurm-saltstack.js
 */

const { chromium } = require('playwright');

// 配置
const CONFIG = {
    baseURL: 'http://192.168.0.200:8080',
    username: 'admin',
    password: 'admin123',
    timeout: 30000,
    headless: false, // 设置为 false 可以看到浏览器操作
};

async function login(page) {
    console.log('🔐 开始登录...');
    
    // 访问登录页面
    await page.goto(`${CONFIG.baseURL}/login`);
    await page.waitForLoadState('networkidle');
    
    // 输入用户名和密码
    await page.fill('input[name="username"], input[type="text"]', CONFIG.username);
    await page.fill('input[name="password"], input[type="password"]', CONFIG.password);
    
    // 点击登录按钮
    await page.click('button[type="submit"], button:has-text("登录")');
    
    // 等待登录成功（跳转到首页或仪表板）
    await page.waitForURL(/\/(dashboard|home)?$/, { timeout: CONFIG.timeout });
    
    console.log('✅ 登录成功');
}

async function testSlurmPage(page) {
    console.log('\n📊 测试 SLURM 页面...');
    
    try {
        // 访问 SLURM 页面
        await page.goto(`${CONFIG.baseURL}/slurm`);
        await page.waitForLoadState('networkidle');
        
        // 等待页面加载
        await page.waitForTimeout(2000);
        
        // 检查是否有错误消息
        const errorMessages = await page.locator('text=/部分数据加载失败|加载失败|错误|Error/i').all();
        
        if (errorMessages.length > 0) {
            console.log('❌ SLURM 页面发现错误消息：');
            for (const error of errorMessages) {
                const text = await error.textContent();
                console.log(`   - ${text}`);
            }
            
            // 截图保存错误状态
            await page.screenshot({ 
                path: 'slurm-error.png',
                fullPage: true 
            });
            console.log('📸 错误截图已保存: slurm-error.png');
        } else {
            console.log('✅ SLURM 页面未发现错误消息');
        }
        
        // 检查页面内容
        const pageContent = await page.content();
        
        // 检查是否有 SLURM 相关元素
        const hasSlurmContent = pageContent.includes('slurm') || 
                                pageContent.includes('SLURM') ||
                                pageContent.includes('作业') ||
                                pageContent.includes('队列');
        
        if (hasSlurmContent) {
            console.log('✅ SLURM 页面包含相关内容');
        } else {
            console.log('⚠️  SLURM 页面可能缺少内容');
        }
        
        // 检查控制台错误
        const consoleErrors = [];
        page.on('console', msg => {
            if (msg.type() === 'error') {
                consoleErrors.push(msg.text());
            }
        });
        
        if (consoleErrors.length > 0) {
            console.log('⚠️  浏览器控制台错误：');
            consoleErrors.forEach(err => console.log(`   - ${err}`));
        }
        
        // 检查网络请求失败
        const failedRequests = [];
        page.on('requestfailed', request => {
            failedRequests.push({
                url: request.url(),
                failure: request.failure()
            });
        });
        
        if (failedRequests.length > 0) {
            console.log('⚠️  网络请求失败：');
            failedRequests.forEach(req => {
                console.log(`   - ${req.url}: ${req.failure?.errorText}`);
            });
        }
        
    } catch (error) {
        console.log('❌ SLURM 页面测试失败:', error.message);
        await page.screenshot({ 
            path: 'slurm-test-error.png',
            fullPage: true 
        });
    }
}

async function testSaltStackPage(page) {
    console.log('\n🧂 测试 SaltStack 页面...');
    
    try {
        // 访问 SaltStack 页面
        await page.goto(`${CONFIG.baseURL}/saltstack`);
        await page.waitForLoadState('networkidle');
        
        // 等待页面加载
        await page.waitForTimeout(2000);
        
        console.log('✅ SaltStack 页面已加载');
        
        // 查找脚本执行区域
        const scriptArea = page.locator('textarea, input[type="text"]').first();
        
        if (await scriptArea.count() > 0) {
            console.log('📝 找到脚本输入框，准备测试脚本执行...');
            
            // 输入测试脚本
            await scriptArea.fill('test.ping');
            
            // 查找并点击执行按钮
            const executeButton = page.locator('button:has-text("执行"), button:has-text("运行"), button:has-text("Execute")').first();
            
            if (await executeButton.count() > 0) {
                console.log('🚀 点击执行按钮...');
                await executeButton.click();
                
                // 监控加载状态
                const startTime = Date.now();
                let isLoading = true;
                let loadingDuration = 0;
                
                // 等待执行完成或超时（最多 70 秒，因为 SaltStack 可能需要 60 秒）
                while (isLoading && loadingDuration < 70000) {
                    await page.waitForTimeout(1000);
                    
                    // 检查是否还在加载（转圈）
                    const spinner = await page.locator('.loading, .spinner, [class*="spin"]').count();
                    isLoading = spinner > 0;
                    
                    loadingDuration = Date.now() - startTime;
                    
                    if (loadingDuration % 5000 === 0) {
                        console.log(`⏳ 等待执行完成... (${Math.round(loadingDuration / 1000)}s)`);
                    }
                }
                
                if (isLoading) {
                    console.log('❌ 脚本执行超时，前端一直在转圈');
                    await page.screenshot({ 
                        path: 'saltstack-loading.png',
                        fullPage: true 
                    });
                } else {
                    console.log(`✅ 脚本执行完成 (耗时: ${Math.round(loadingDuration / 1000)}s)`);
                    
                    // 检查执行结果
                    await page.waitForTimeout(1000);
                    const result = await page.locator('.result, .output, pre').first().textContent();
                    
                    if (result) {
                        console.log('📄 执行结果:', result.substring(0, 200));
                    }
                    
                    await page.screenshot({ 
                        path: 'saltstack-result.png',
                        fullPage: true 
                    });
                }
            } else {
                console.log('⚠️  未找到执行按钮');
            }
        } else {
            console.log('⚠️  未找到脚本输入框');
        }
        
        // 截图保存当前状态
        await page.screenshot({ 
            path: 'saltstack-page.png',
            fullPage: true 
        });
        console.log('📸 SaltStack 页面截图已保存');
        
    } catch (error) {
        console.log('❌ SaltStack 页面测试失败:', error.message);
        await page.screenshot({ 
            path: 'saltstack-test-error.png',
            fullPage: true 
        });
    }
}

async function main() {
    console.log('🎭 启动 Playwright 测试...\n');
    
    const browser = await chromium.launch({
        headless: CONFIG.headless,
        slowMo: 100, // 慢速执行，方便观察
    });
    
    const context = await browser.newContext({
        viewport: { width: 1920, height: 1080 },
        recordVideo: {
            dir: './test-videos/',
            size: { width: 1920, height: 1080 }
        }
    });
    
    const page = await context.newPage();
    
    // 设置默认超时
    page.setDefaultTimeout(CONFIG.timeout);
    
    try {
        // 1. 登录
        await login(page);
        
        // 2. 测试 SLURM 页面
        await testSlurmPage(page);
        
        // 3. 测试 SaltStack 页面
        await testSaltStackPage(page);
        
        console.log('\n✅ 所有测试完成！');
        
    } catch (error) {
        console.error('\n❌ 测试过程中发生错误:', error);
        
        // 保存最终截图
        await page.screenshot({ 
            path: 'final-error.png',
            fullPage: true 
        });
        
    } finally {
        await context.close();
        await browser.close();
        
        console.log('\n📊 测试报告：');
        console.log('   - 截图保存在当前目录');
        console.log('   - 视频保存在 ./test-videos/ 目录');
    }
}

// 运行测试
main().catch(console.error);
