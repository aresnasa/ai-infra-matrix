// AI-Infra-Matrix 前端SSO集成脚本
// 支持统一的JWT单点登录

class AIInfraSSO {
    constructor(config) {
        this.config = {
            backendUrl: config.backendUrl || 'http://localhost:8082',
            jupyterhubUrl: config.jupyterhubUrl || 'http://localhost:8000',
            tokenKey: config.tokenKey || 'ai_infra_token',
            refreshInterval: config.refreshInterval || 300000, // 5分钟
            ...config
        };

        this.token = null;
        this.userInfo = null;
        this.refreshTimer = null;

        this.init();
    }

    init() {
        console.log('🔐 AI-Infra SSO系统初始化...');

        // 从存储中恢复token
        this.loadToken();

        // 检查URL中的token参数
        this.checkUrlToken();

        // 验证当前token
        if (this.token) {
            this.verifyToken();
        }

        // 设置自动刷新
        this.startTokenRefresh();
    }

    loadToken() {
        // 从localStorage读取token
        this.token = localStorage.getItem(this.config.tokenKey);

        // 从cookie读取token
        if (!this.token) {
            this.token = this.getCookie(this.config.tokenKey);
        }

        if (this.token) {
            console.log('📍 从存储中恢复token');
        }
    }

    checkUrlToken() {
        const urlParams = new URLSearchParams(window.location.search);
        const urlToken = urlParams.get('token');

        if (urlToken) {
            console.log('📍 从URL参数获取token');
            this.setToken(urlToken);

            // 清理URL中的token参数
            urlParams.delete('token');
            const newUrl = window.location.pathname + 
                          (urlParams.toString() ? '?' + urlParams.toString() : '');
            window.history.replaceState({}, '', newUrl);
        }
    }

    async verifyToken() {
        if (!this.token) return false;

        try {
            const response = await fetch(`${this.config.backendUrl}/api/auth/verify-token`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.token}`
                },
                body: JSON.stringify({ token: this.token })
            });

            const result = await response.json();

            if (result.valid) {
                this.userInfo = result.user_info;
                console.log('✅ Token验证成功:', this.userInfo);
                this.onLoginSuccess();
                return true;
            } else {
                console.log('❌ Token验证失败');
                this.clearToken();
                return false;
            }

        } catch (error) {
            console.error('❌ Token验证异常:', error);
            return false;
        }
    }

    async login(username, password) {
        try {
            const response = await fetch(`${this.config.backendUrl}/api/auth/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password })
            });

            const result = await response.json();

            if (result.success && result.token) {
                this.setToken(result.token);
                this.userInfo = result.user_info;
                console.log('✅ 登录成功:', this.userInfo);
                this.onLoginSuccess();
                return true;
            } else {
                console.log('❌ 登录失败:', result.message);
                return false;
            }

        } catch (error) {
            console.error('❌ 登录异常:', error);
            return false;
        }
    }

    async logout() {
        try {
            if (this.token) {
                await fetch(`${this.config.backendUrl}/api/auth/logout`, {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${this.token}`
                    }
                });
            }
        } catch (error) {
            console.error('登出请求失败:', error);
        }

        this.clearToken();
        this.onLogout();
    }

    setToken(token) {
        this.token = token;

        // 保存到localStorage
        localStorage.setItem(this.config.tokenKey, token);

        // 保存到cookie
        this.setCookie(this.config.tokenKey, token, 1); // 1天过期

        console.log('💾 Token已保存');
    }

    clearToken() {
        this.token = null;
        this.userInfo = null;

        // 清理localStorage
        localStorage.removeItem(this.config.tokenKey);

        // 清理cookie
        this.setCookie(this.config.tokenKey, '', -1);

        console.log('🗑️ Token已清理');
    }

    startTokenRefresh() {
        this.refreshTimer = setInterval(() => {
            if (this.token) {
                this.verifyToken();
            }
        }, this.config.refreshInterval);
    }

    stopTokenRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }

    // 跳转到JupyterHub
    openJupyterHub() {
        if (this.token) {
            const jupyterUrl = `${this.config.jupyterhubUrl}?token=${this.token}`;
            window.open(jupyterUrl, '_blank');
        } else {
            console.warn('无有效token，无法打开JupyterHub');
        }
    }

    // 工具方法
    getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
        return null;
    }

    setCookie(name, value, days) {
        const expires = new Date();
        expires.setTime(expires.getTime() + (days * 24 * 60 * 60 * 1000));
        document.cookie = `${name}=${value};expires=${expires.toUTCString()};path=/`;
    }

    // 事件回调
    onLoginSuccess() {
        console.log('🎉 SSO登录成功回调');
        // 触发自定义事件
        window.dispatchEvent(new CustomEvent('sso:login', {
            detail: { userInfo: this.userInfo }
        }));
    }

    onLogout() {
        console.log('👋 SSO登出回调');
        // 触发自定义事件
        window.dispatchEvent(new CustomEvent('sso:logout'));
    }

    // 获取当前状态
    isLoggedIn() {
        return !!this.token && !!this.userInfo;
    }

    getToken() {
        return this.token;
    }

    getUserInfo() {
        return this.userInfo;
    }
}

// 全局SSO实例
window.AIInfraSSO = AIInfraSSO;

// 使用示例
/*
const sso = new AIInfraSSO({
    backendUrl: 'http://localhost:8082',
    jupyterhubUrl: 'http://localhost:8000'
});

// 监听登录成功事件
window.addEventListener('sso:login', (event) => {
    console.log('用户已登录:', event.detail.userInfo);
});

// 监听登出事件
window.addEventListener('sso:logout', () => {
    console.log('用户已登出');
});
*/
