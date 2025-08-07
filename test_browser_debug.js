// 浏览器调试脚本 - 在开发者控制台运行
console.log('=== AI-Infra-Matrix 登录调试脚本 ===');

// 1. 清理现有状态
console.log('1. 清理localStorage和sessionStorage...');
localStorage.clear();
sessionStorage.clear();
console.log('存储已清理');

// 2. 检查当前页面状态
console.log('2. 当前页面状态:');
console.log('URL:', window.location.href);
console.log('Title:', document.title);

// 3. 监听所有网络请求
console.log('3. 设置网络请求监听...');
const originalFetch = window.fetch;
window.fetch = function(...args) {
    console.log('📡 Fetch请求:', args[0], args[1]);
    return originalFetch.apply(this, arguments)
        .then(response => {
            console.log('📡 Fetch响应:', response.status, response.statusText, args[0]);
            return response;
        })
        .catch(error => {
            console.error('📡 Fetch错误:', error, args[0]);
            throw error;
        });
};

// 4. 监听axios请求（如果存在）
if (window.axios) {
    console.log('4. 设置axios拦截器...');
    window.axios.interceptors.request.use(
        config => {
            console.log('🔗 Axios请求:', config.method?.toUpperCase(), config.url, config);
            return config;
        },
        error => {
            console.error('🔗 Axios请求错误:', error);
            return Promise.reject(error);
        }
    );
    
    window.axios.interceptors.response.use(
        response => {
            console.log('🔗 Axios响应:', response.status, response.config.url, response.data);
            return response;
        },
        error => {
            console.error('🔗 Axios响应错误:', error.response?.status, error.config?.url, error.response?.data);
            return Promise.reject(error);
        }
    );
}

// 5. 监听localStorage变化
const originalSetItem = localStorage.setItem;
localStorage.setItem = function(key, value) {
    console.log('💾 localStorage设置:', key, value);
    return originalSetItem.apply(this, arguments);
};

const originalGetItem = localStorage.getItem;
localStorage.getItem = function(key) {
    const value = originalGetItem.apply(this, arguments);
    console.log('💾 localStorage获取:', key, '=', value);
    return value;
};

// 6. 自动填写并提交登录表单
console.log('5. 等待3秒后自动登录...');
setTimeout(() => {
    console.log('6. 开始自动登录...');
    
    // 查找用户名和密码输入框
    const usernameInput = document.querySelector('input[placeholder*="用户名"], input[type="text"], input[name="username"]');
    const passwordInput = document.querySelector('input[placeholder*="密码"], input[type="password"], input[name="password"]');
    const submitButton = document.querySelector('button[type="submit"], button:contains("登录"), .ant-btn-primary');
    
    console.log('表单元素:', {
        username: usernameInput,
        password: passwordInput,
        submit: submitButton
    });
    
    if (usernameInput && passwordInput) {
        console.log('7. 填写登录信息...');
        usernameInput.value = 'admin';
        usernameInput.dispatchEvent(new Event('input', { bubbles: true }));
        usernameInput.dispatchEvent(new Event('change', { bubbles: true }));
        
        passwordInput.value = 'admin123';
        passwordInput.dispatchEvent(new Event('input', { bubbles: true }));
        passwordInput.dispatchEvent(new Event('change', { bubbles: true }));
        
        console.log('8. 提交登录表单...');
        if (submitButton) {
            submitButton.click();
        } else {
            // 尝试找到表单并提交
            const form = document.querySelector('form');
            if (form) {
                form.dispatchEvent(new Event('submit', { bubbles: true }));
            }
        }
    } else {
        console.error('❌ 未找到登录表单元素');
        console.log('页面所有input元素:', document.querySelectorAll('input'));
        console.log('页面所有button元素:', document.querySelectorAll('button'));
    }
}, 3000);

// 7. 监听页面变化
console.log('7. 设置页面变化监听...');
const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
        if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
            console.log('📄 页面内容变化:', mutation.addedNodes);
        }
    });
});

observer.observe(document.body, {
    childList: true,
    subtree: true
});

// 8. 定期检查状态
setInterval(() => {
    const token = localStorage.getItem('token');
    const currentPath = window.location.pathname;
    console.log('🔄 状态检查:', {
        path: currentPath,
        hasToken: !!token,
        token: token ? token.substring(0, 20) + '...' : null
    });
}, 5000);

console.log('调试脚本初始化完成，请观察控制台输出...');
