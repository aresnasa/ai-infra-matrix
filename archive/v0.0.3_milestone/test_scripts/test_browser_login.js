// 测试浏览器登录流程
const axios = require('axios');

async function testBrowserLogin() {
    console.log('=== 测试浏览器登录流程 ===');
    
    // 步骤1: 登录
    try {
        console.log('1. 发送登录请求...');
        const loginResponse = await axios.post('http://localhost:8080/api/auth/login', {
            username: 'admin',
            password: 'admin123'
        }, {
            headers: {
                'Content-Type': 'application/json',
                'Origin': 'http://localhost:8080'
            }
        });
        
        console.log('✅ 登录成功');
        console.log('Token:', loginResponse.data.token ? '已获取' : '未获取');
        console.log('User:', loginResponse.data.user?.username);
        
        const token = loginResponse.data.token;
        
        // 步骤2: 使用token获取用户信息
        console.log('\n2. 使用token获取用户信息...');
        const meResponse = await axios.get('http://localhost:8080/api/auth/me', {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Origin': 'http://localhost:8080'
            }
        });
        
        console.log('✅ 获取用户信息成功');
        console.log('用户角色:', meResponse.data.roles?.map(r => r.name));
        
    } catch (error) {
        console.error('❌ 登录流程失败:', error.response?.data || error.message);
        
        // 如果是401错误，尝试refresh token
        if (error.response?.status === 401) {
            console.log('\n3. 尝试刷新token...');
            try {
                const refreshResponse = await axios.post('http://localhost:8080/api/auth/refresh', {}, {
                    headers: {
                        'Authorization': `Bearer ${token}`,
                        'Origin': 'http://localhost:8080'
                    }
                });
                console.log('✅ Token刷新成功');
            } catch (refreshError) {
                console.error('❌ Token刷新失败:', refreshError.response?.data || refreshError.message);
            }
        }
    }
}

testBrowserLogin().catch(console.error);
