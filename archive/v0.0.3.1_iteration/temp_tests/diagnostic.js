// 前端诊断脚本
console.log('=== Frontend Diagnostic Script ===');
console.log('User Agent:', navigator.userAgent);
console.log('Current URL:', window.location.href);
console.log('Document Ready State:', document.readyState);

// 测试API连接
async function testAPIConnection() {
    console.log('=== Testing API Connection ===');
    
    try {
        console.log('Testing /api/health...');
        const healthResponse = await fetch('/api/health');
        const healthData = await healthResponse.json();
        console.log('Health API Response:', healthData);
        
        if (healthResponse.ok) {
            console.log('✅ Backend API is accessible');
        } else {
            console.log('❌ Backend API error:', healthResponse.status);
        }
    } catch (error) {
        console.log('❌ API Connection Error:', error);
    }
}

// 测试React DOM
function testReactDOM() {
    console.log('=== Testing React DOM ===');
    
    const rootElement = document.getElementById('root');
    if (rootElement) {
        console.log('✅ Root element found');
        console.log('Root element children:', rootElement.children.length);
        console.log('Root element content:', rootElement.innerHTML.substring(0, 100));
    } else {
        console.log('❌ Root element not found');
    }
}

// 测试外部资源加载
function testResourceLoading() {
    console.log('=== Testing Resource Loading ===');
    
    const scripts = document.querySelectorAll('script[src]');
    const links = document.querySelectorAll('link[href]');
    
    console.log('Scripts found:', scripts.length);
    scripts.forEach((script, index) => {
        console.log(`Script ${index + 1}:`, script.src);
    });
    
    console.log('CSS links found:', links.length);
    links.forEach((link, index) => {
        console.log(`Link ${index + 1}:`, link.href);
    });
}

// 监听DOMContentLoaded事件
document.addEventListener('DOMContentLoaded', function() {
    console.log('✅ DOM Content Loaded');
    testReactDOM();
    testResourceLoading();
    testAPIConnection();
});

// 监听window load事件
window.addEventListener('load', function() {
    console.log('✅ Window Loaded');
    
    // 给React应用一些时间渲染
    setTimeout(() => {
        console.log('=== Post-Load DOM Check ===');
        testReactDOM();
        
        // 检查是否有React错误
        if (window.React) {
            console.log('✅ React library detected');
        } else {
            console.log('⚠️ React library not detected');
        }
        
        // 检查是否有错误事件
        if (window.onerror) {
            console.log('⚠️ Error handler detected');
        }
    }, 2000);
});

// 全局错误处理
window.addEventListener('error', function(event) {
    console.log('🚨 JavaScript Error:', event.error);
    console.log('Error message:', event.message);
    console.log('Error filename:', event.filename);
    console.log('Error line:', event.lineno);
});

// 监听未处理的Promise拒绝
window.addEventListener('unhandledrejection', function(event) {
    console.log('🚨 Unhandled Promise Rejection:', event.reason);
});

console.log('=== Diagnostic script loaded ===');
