// 检查浏览器登录问题的详细调试脚本
console.log('=== 详细调试浏览器登录问题 ===');

// 清理localStorage
localStorage.clear();
console.log('已清理localStorage');

// 执行登录
async function testLoginFlow() {
    try {
        console.log('1. 开始登录请求...');
        
        const loginResponse = await fetch('/api/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                username: 'admin',
                password: 'admin123'
            })
        });
        
        console.log('2. 登录响应状态:', loginResponse.status);
        
        if (!loginResponse.ok) {
            console.error('登录失败');
            return;
        }
        
        const loginData = await loginResponse.json();
        console.log('3. 登录响应数据:', loginData);
        
        const token = loginData.token;
        
        if (!token) {
            console.error('未获得token');
            return;
        }
        
        console.log('4. 获得token:', token.substring(0, 50) + '...');
        
        // 存储token
        localStorage.setItem('token', token);
        console.log('5. Token已存储到localStorage');
        
        // 验证token
        console.log('6. 开始验证token...');
        
        const verifyResponse = await fetch('/api/auth/me', {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });
        
        console.log('7. 验证响应状态:', verifyResponse.status);
        console.log('8. 验证请求headers:', {
            'Authorization': `Bearer ${token.substring(0, 20)}...`,
            'Content-Type': 'application/json'
        });
        
        if (verifyResponse.ok) {
            const userData = await verifyResponse.json();
            console.log('9. ✅ Token验证成功:', userData);
        } else {
            const errorText = await verifyResponse.text();
            console.error('9. ❌ Token验证失败:', errorText);
            
            // 检查token是否确实存在
            console.log('检查localStorage中的token:', localStorage.getItem('token'));
        }
        
    } catch (error) {
        console.error('测试过程出错:', error);
    }
}

// 执行测试
testLoginFlow();
