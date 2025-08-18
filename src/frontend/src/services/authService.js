/**
 * 增强的认证服务 - 支持SSO单点登录
 */

class AuthService {
    constructor() {
        this.baseURL = window.location.origin;
        this.tokenKey = 'token';
        this.expiresKey = 'token_expires';
        this.userKey = 'user';
    }

    /**
     * 登录
     */
    async login(credentials) {
        try {
            const response = await fetch('/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(credentials)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || '登录失败');
            }

            const data = await response.json();
            
            // 存储认证信息
            this.setAuthData(data.token, data.expires_at, data.user);
            
            // 设置SSO cookie
            await this.setupSSOCookies(data.token, data.user);
            
            return data;
        } catch (error) {
            console.error('Login failed:', error);
            throw error;
        }
    }

    /**
     * 登出
     */
    async logout() {
        try {
            // 清除localStorage
            this.clearAuthData();
            
            // 清除SSO cookies
            this.clearSSOCookies();
            
            // 通知后端登出
            const token = this.getToken();
            if (token) {
                try {
                    await fetch('/api/auth/logout', {
                        method: 'POST',
                        headers: {
                            'Authorization': `Bearer ${token}`
                        }
                    });
                } catch (e) {
                    console.warn('后端登出请求失败:', e);
                }
            }

            // 统一清理Gitea侧会话，避免iframe内JS解析非JSON导致的错误
            try {
                await fetch('/gitea/_logout', { method: 'GET', credentials: 'include' });
            } catch (e) {
                console.warn('Gitea 登出清理失败（可忽略）:', e);
            }
            
            // 重定向到登录页
            window.location.href = '/';
        } catch (error) {
            console.error('Logout failed:', error);
            throw error;
        }
    }

    /**
     * 获取用户资料
     */
    async getProfile() {
        const token = this.getToken();
        if (!token) {
            throw new Error('未找到认证token');
        }

        const response = await fetch('/api/users/profile', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                // Token可能过期，尝试刷新
                await this.refreshToken();
                return this.getProfile(); // 递归重试
            }
            throw new Error('获取用户资料失败');
        }

        return response.json();
    }

    /**
     * 刷新token
     */
    async refreshToken() {
        const token = this.getToken();
        if (!token) {
            throw new Error('未找到认证token');
        }

        const response = await fetch('/api/auth/refresh-token', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) {
            this.clearAuthData();
            throw new Error('Token刷新失败');
        }

        const data = await response.json();
        
        // 更新存储的认证信息
        this.setAuthData(data.token, data.expires_at);
        
        // 更新SSO cookies
        await this.setupSSOCookies(data.token);
        
        return data;
    }

    /**
     * 验证token
     */
    async verifyToken(token = null) {
        const authToken = token || this.getToken();
        if (!authToken) {
            return { valid: false, error: '未找到token' };
        }

        try {
            const response = await fetch('/api/auth/verify', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${authToken}`
                }
            });

            if (response.ok) {
                const data = await response.json();
                return { valid: true, user: data };
            } else {
                return { valid: false, error: '验证失败' };
            }
        } catch (error) {
            return { valid: false, error: error.message };
        }
    }

    /**
     * 设置认证数据
     */
    setAuthData(token, expiresAt, user = null) {
        localStorage.setItem(this.tokenKey, token);
        if (expiresAt) {
            localStorage.setItem(this.expiresKey, expiresAt);
        }
        if (user) {
            localStorage.setItem(this.userKey, JSON.stringify(user));
        }
    }

    /**
     * 获取token
     */
    getToken() {
        return localStorage.getItem(this.tokenKey);
    }

    /**
     * 获取用户信息
     */
    getUser() {
        const userStr = localStorage.getItem(this.userKey);
        try {
            return userStr ? JSON.parse(userStr) : null;
        } catch (e) {
            return null;
        }
    }

    /**
     * 检查token是否有效
     */
    isTokenValid() {
        const token = this.getToken();
        const expiresAt = localStorage.getItem(this.expiresKey);

        if (!token || !expiresAt) {
            return false;
        }

        const expiryTime = new Date(expiresAt).getTime();
        const currentTime = new Date().getTime();
        const bufferTime = 5 * 60 * 1000; // 5分钟缓冲

        return currentTime + bufferTime < expiryTime;
    }

    /**
     * 清除认证数据
     */
    clearAuthData() {
        localStorage.removeItem(this.tokenKey);
        localStorage.removeItem(this.expiresKey);
        localStorage.removeItem(this.userKey);
    }

    /**
     * 设置SSO cookies
     */
    async setupSSOCookies(token, user = null) {
        try {
            console.log('设置SSO cookies...');
            
            // Cookie过期时间（1小时）
            const maxAge = 3600;
            const cookieOptions = `path=/; max-age=${maxAge}; SameSite=Lax`;
            
            // 设置主要的认证cookies
            const cookies = [
                `ai_infra_token=${token}; ${cookieOptions}`,
                `jwt_token=${token}; ${cookieOptions}`,
                `auth_token=${token}; ${cookieOptions}`
            ];
            
            cookies.forEach(cookieStr => {
                document.cookie = cookieStr;
            });
            
            // 设置用户信息cookie
            if (user) {
                const userInfo = {
                    username: user.username,
                    roles: user.roles || [],
                    permissions: user.permissions || []
                };
                
                document.cookie = `user_info=${encodeURIComponent(JSON.stringify(userInfo))}; ${cookieOptions}`;
            }
            
            console.log('SSO cookies设置完成');
            
            // 通知JupyterHub有新的认证状态
            this.notifyJupyterHubAuth(token);
            
        } catch (error) {
            console.error('设置SSO cookies失败:', error);
        }
    }

    /**
     * 清除SSO cookies
     */
    clearSSOCookies() {
        const cookieNames = ['ai_infra_token', 'jwt_token', 'auth_token', 'user_info'];
        
        cookieNames.forEach(name => {
            document.cookie = `${name}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
        });
        
        console.log('SSO cookies已清除');
    }

    /**
     * 通知JupyterHub有新的认证状态
     */
    notifyJupyterHubAuth(token) {
        try {
            // 如果当前页面有JupyterHub iframe或窗口，通知它们
            const iframes = document.querySelectorAll('iframe[src*="jupyter"]');
            iframes.forEach(iframe => {
                iframe.contentWindow?.postMessage({
                    type: 'auth_update',
                    token: token
                }, '*');
            });
            
            // 设置一个标记，告诉JupyterHub页面认证状态已更新
            localStorage.setItem('auth_updated', Date.now().toString());
            
        } catch (error) {
            console.warn('通知JupyterHub认证状态失败:', error);
        }
    }

    /**
     * 创建JupyterHub访问链接 - 使用认证桥接
     */
    createJupyterHubLink(path = '/') {
        const token = this.getToken();
        if (!token) {
            // 没有token，跳转到SSO登录
            try {
                const { resolveSSOTarget } = require('../utils/ssoTarget');
                const target = resolveSSOTarget();
                return '/sso/?next=' + encodeURIComponent(target.nextPath);
            } catch (_) {
                return '/sso/?next=' + encodeURIComponent('/jupyterhub');
            }
        }
        
        // 有token，使用认证桥接页面
        try {
            const { resolveSSOTarget } = require('../utils/ssoTarget');
            const target = resolveSSOTarget();
            return target.key === 'gitea' ? '/gitea/' : '/jupyterhub';
        } catch (_) {
            return '/jupyterhub';
        }
    }

    /**
     * 跳转到JupyterHub（确保SSO状态）- 使用认证桥接机制
     */
    async goToJupyterHub(path = '/') {
        try {
            const token = this.getToken();
            if (!token) {
                // 没有token，跳转到SSO登录
                try {
                    const { resolveSSOTarget } = require('../utils/ssoTarget');
                    const target = resolveSSOTarget();
                    window.location.href = '/sso/?next=' + encodeURIComponent(target.nextPath);
                } catch (_) {
                    window.location.href = '/sso/?next=' + encodeURIComponent('/jupyterhub');
                }
                return;
            }
            
            // 验证token有效性
            try {
                const verification = await this.verifyToken(token);
                if (!verification.valid) {
                    // Token无效，尝试刷新
                    await this.refreshToken();
                }
            } catch (error) {
                console.warn('Token验证失败，将通过认证桥接处理:', error);
            }
            
            // 跳转到认证桥接页面，它会自动处理认证传递
            try {
                const { resolveSSOTarget } = require('../utils/ssoTarget');
                const target = resolveSSOTarget();
                window.location.href = target.key === 'gitea' ? '/gitea/' : '/jupyterhub';
            } catch (_) {
                window.location.href = '/jupyterhub';
            }
            
        } catch (error) {
            console.error('跳转到JupyterHub失败:', error);
            // 降级方案：跳转到SSO登录
            try {
                const { resolveSSOTarget } = require('../utils/ssoTarget');
                const target = resolveSSOTarget();
                window.location.href = '/sso/?next=' + encodeURIComponent(target.nextPath);
            } catch (_) {
                window.location.href = '/sso/?next=' + encodeURIComponent('/jupyterhub');
            }
        }
    }

    /**
     * 初始化认证状态
     */
    async initializeAuth() {
        try {
            const token = this.getToken();
            if (!token) {
                return { authenticated: false };
            }
            
            if (!this.isTokenValid()) {
                try {
                    await this.refreshToken();
                } catch (error) {
                    this.clearAuthData();
                    this.clearSSOCookies();
                    return { authenticated: false };
                }
            }
            
            // 设置SSO cookies
            await this.setupSSOCookies(this.getToken(), this.getUser());
            
            return { authenticated: true, user: this.getUser() };
            
        } catch (error) {
            console.error('初始化认证状态失败:', error);
            this.clearAuthData();
            this.clearSSOCookies();
            return { authenticated: false };
        }
    }
}

// 创建全局实例
const authService = new AuthService();

// 兼容现有代码的authAPI对象
const authAPI = {
    login: (credentials) => authService.login(credentials),
    logout: () => authService.logout(),
    getProfile: () => authService.getProfile(),
    refreshToken: () => authService.refreshToken(),
    verifyToken: (token) => authService.verifyToken(token)
};

// 导出
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { authService, authAPI };
} else {
    window.authService = authService;
    window.authAPI = authAPI;
}

// 页面加载时自动初始化
document.addEventListener('DOMContentLoaded', () => {
    authService.initializeAuth();
});
