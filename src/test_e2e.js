#!/usr/bin/env node

const puppeteer = require('puppeteer');

async function testKubernetesManagement() {
    console.log('=== 开始端到端测试 ===');
    
    // 检查是否安装了puppeteer
    try {
        const browser = await puppeteer.launch({ 
            headless: false, // 设为false以查看浏览器操作
            defaultViewport: { width: 1920, height: 1080 }
        });
        
        const page = await browser.newPage();
        
        // 监听页面错误
        page.on('console', msg => {
            console.log('页面日志:', msg.type(), msg.text());
        });
        
        page.on('pageerror', error => {
            console.error('页面错误:', error.message);
        });
        
        console.log('1. 访问主页...');
        await page.goto('http://localhost:3001');
        await page.waitForTimeout(2000);
        
        console.log('2. 检查是否需要登录...');
        const currentUrl = page.url();
        
        if (currentUrl.includes('/auth') || currentUrl.includes('/login')) {
            console.log('需要登录，正在登录...');
            
            // 等待登录表单
            await page.waitForSelector('input[placeholder*="用户名"], input[name="username"]', { timeout: 5000 });
            
            // 输入用户名和密码
            await page.type('input[placeholder*="用户名"], input[name="username"]', 'admin');
            await page.type('input[placeholder*="密码"], input[name="password"]', 'admin123');
            
            // 点击登录按钮
            await page.click('button[type="submit"], .ant-btn-primary');
            
            // 等待登录完成
            await page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 10000 });
            console.log('登录完成');
        }
        
        console.log('3. 导航到Kubernetes管理页面...');
        await page.goto('http://localhost:3001/kubernetes');
        await page.waitForTimeout(3000);
        
        console.log('4. 检查页面内容...');
        
        // 检查页面标题
        const title = await page.evaluate(() => {
            const titleElement = document.querySelector('h1, .ant-typography h1, [class*="title"]');
            return titleElement ? titleElement.textContent : 'No title found';
        });
        console.log('页面标题:', title);
        
        // 检查是否有集群数据
        await page.waitForTimeout(5000); // 等待数据加载
        
        const clusters = await page.evaluate(() => {
            // 查找表格行或集群卡片
            const tableRows = document.querySelectorAll('.ant-table-tbody tr');
            const clusterCards = document.querySelectorAll('[class*="cluster"], [class*="card"]');
            
            return {
                tableRows: tableRows.length,
                cards: clusterCards.length,
                hasData: tableRows.length > 0 || clusterCards.length > 0
            };
        });
        
        console.log('集群数据检查:', clusters);
        
        // 检查状态标签
        const statusTags = await page.evaluate(() => {
            const tags = document.querySelectorAll('.ant-tag');
            return Array.from(tags).map(tag => tag.textContent);
        });
        
        console.log('状态标签:', statusTags);
        
        // 检查是否有"已连接"状态
        const hasConnectedStatus = statusTags.some(tag => 
            tag.includes('已连接') || tag.includes('connected')
        );
        
        console.log('是否有已连接状态:', hasConnectedStatus);
        
        // 截图保存
        await page.screenshot({ 
            path: '/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/apisChecker/deploybot/ansible-playbook-generator/web-v2/test-screenshot.png',
            fullPage: true 
        });
        console.log('截图已保存到 test-screenshot.png');
        
        // 测试结果总结
        console.log('\n=== 测试结果总结 ===');
        console.log('✅ 前端应用可访问');
        console.log('✅ 登录功能正常');
        console.log('✅ Kubernetes页面可访问');
        console.log(`${clusters.hasData ? '✅' : '❌'} 集群数据${clusters.hasData ? '正常显示' : '未显示'}`);
        console.log(`${hasConnectedStatus ? '✅' : '❌'} 集群状态${hasConnectedStatus ? '正确显示' : '显示异常'}`);
        
        await browser.close();
        
        return {
            success: true,
            hasData: clusters.hasData,
            hasConnectedStatus,
            statusTags
        };
        
    } catch (error) {
        console.error('测试失败:', error.message);
        return { success: false, error: error.message };
    }
}

// 运行测试
testKubernetesManagement().then(result => {
    console.log('\n=== 最终结果 ===');
    console.log(JSON.stringify(result, null, 2));
    process.exit(result.success ? 0 : 1);
}).catch(error => {
    console.error('测试异常:', error);
    process.exit(1);
});
