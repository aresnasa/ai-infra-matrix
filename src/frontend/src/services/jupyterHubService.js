/**
 * JupyterHub集成服务
 * 提供完整的SSO体验和JupyterHub交互功能
 */

import { message } from 'antd';
import api from './api';

class JupyterHubService {
    constructor() {
        this.baseURL = window.location.origin;
        this.jupyterHubURL = `${this.baseURL}/jupyter/`;
        this.ssoLoginURL = `${this.baseURL}/sso/?redirect_uri=/jupyterhub-authenticated`;
    }

    /**
     * 一键登录JupyterHub - 使用后端API处理
     */
    async loginWithSSO() {
        try {
            // 调用后端API处理JupyterHub访问
            const response = await fetch('/api/jupyter/access', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
                },
                body: JSON.stringify({
                    redirect_uri: '/jupyterhub-authenticated',
                    source: 'jupyterhub_service'
                })
            });

            const data = await response.json();
            
            if (data.success && data.action === 'authenticated') {
                // 认证成功，直接跳转
                message.success('认证成功，正在跳转到JupyterHub...');
                window.location.href = data.redirect_url;
                return { success: true, username: data.username };
                
            } else if (data.action === 'redirect') {
                // 需要重定向到SSO登录
                message.info('需要登录，正在跳转到SSO...');
                window.location.href = data.redirect_url;
                return;
                
            } else {
                throw new Error(data.message || '访问失败');
            }

        } catch (error) {
            console.error('JupyterHub SSO登录失败:', error);
            
            // 出错时跳转到SSO登录
            message.warning('登录遇到问题，正在跳转到SSO...');
            setTimeout(() => {
                window.location.href = this.ssoLoginURL;
            }, 1500);
            
            throw error;
        }
    }

    /**
     * 设置认证cookie
     */
    async setAuthCookies(token, username) {
        return new Promise((resolve) => {
            // 创建隐藏iframe设置cookie
            const iframe = document.createElement('iframe');
            iframe.style.display = 'none';
            iframe.style.width = '1px';
            iframe.style.height = '1px';

            iframe.onload = () => {
                setTimeout(() => {
                    if (iframe.parentNode) {
                        iframe.parentNode.removeChild(iframe);
                    }
                    resolve();
                }, 200);
            };

            // 设置cookie的HTML内容
            const cookieScript = `
                document.cookie = 'ai_infra_token=${token}; path=/; max-age=3600; SameSite=Lax';
                document.cookie = 'ai_infra_username=${username}; path=/; max-age=3600; SameSite=Lax';
                document.cookie = 'jupyterhub_sso=true; path=/; max-age=3600; SameSite=Lax';
            `;

            iframe.src = `data:text/html,<html><body><script>${cookieScript}</script></body></html>`;
            document.body.appendChild(iframe);
        });
    }

    /**
     * 构建JupyterHub登录URL
     */
    buildLoginUrl(token, username) {
        const params = new URLSearchParams({
            token: token,
            username: username,
            next: `${this.jupyterHubURL}hub/`,
            auto_login: 'true'
        });

        return `${this.jupyterHubURL}hub/login?${params.toString()}`;
    }

    /**
     * 打开JupyterHub窗口
     */
    openJupyterHubWindow(loginUrl) {
        const screenWidth = window.screen ? window.screen.width : 1920;
        const screenHeight = window.screen ? window.screen.height : 1080;
        
        const windowFeatures = [
            'width=1400',
            'height=900',
            'scrollbars=yes',
            'resizable=yes',
            'status=yes',
            'location=yes',
            'menubar=no',
            'toolbar=no',
            'left=' + (screenWidth / 2 - 700),
            'top=' + (screenHeight / 2 - 450)
        ].join(',');

        const jupyterWindow = window.open(loginUrl, 'jupyterhub_sso', windowFeatures);

        if (!jupyterWindow) {
            message.error('弹窗被阻止，请允许弹窗后重试');
            throw new Error('弹窗被阻止');
        }

        return jupyterWindow;
    }

    /**
     * 监听JupyterHub窗口状态
     */
    monitorJupyterHubWindow(jupyterWindow) {
        let loginSuccess = false;
        
        const checkWindow = setInterval(() => {
            try {
                if (jupyterWindow.closed) {
                    clearInterval(checkWindow);
                    if (!loginSuccess) {
                        message.info('JupyterHub窗口已关闭');
                    }
                    return;
                }

                // 尝试检查URL变化来判断登录状态
                try {
                    const url = jupyterWindow.location.href;
                    if (url.includes('/hub/spawn') || url.includes('/user/') || url.includes('/lab')) {
                        loginSuccess = true;
                        clearInterval(checkWindow);
                        message.success('已成功登录JupyterHub！');
                        
                        // 可选：聚焦到JupyterHub窗口
                        jupyterWindow.focus();
                    }
                } catch (e) {
                    // 跨域限制是正常的，忽略这个错误
                }
                
            } catch (e) {
                clearInterval(checkWindow);
            }
        }, 2000);

        // 15秒后停止检查
        setTimeout(() => {
            clearInterval(checkWindow);
            if (!loginSuccess && !jupyterWindow.closed) {
                message.info('JupyterHub正在加载中...');
            }
        }, 15000);
    }

    /**
     * 降级登录 - 使用传统方式
     */
    fallbackLogin() {
        message.warning('使用传统方式登录JupyterHub');
        
        const screenWidth = window.screen ? window.screen.width : 1920;
        const screenHeight = window.screen ? window.screen.height : 1080;
        
        const fallbackUrl = `${this.jupyterHubURL}hub/login`;
        const windowFeatures = [
            'width=1400',
            'height=900',
            'scrollbars=yes',
            'resizable=yes',
            'status=yes',
            'location=yes',
            'menubar=no',
            'toolbar=no',
            'left=' + (screenWidth / 2 - 700),
            'top=' + (screenHeight / 2 - 450)
        ].join(',');

        window.open(fallbackUrl, 'jupyterhub_fallback', windowFeatures);
    }

    /**
     * 检查JupyterHub状态
     */
    async checkStatus() {
        try {
            const response = await api.get('/jupyterhub/status');
            return response.data;
        } catch (error) {
            console.error('检查JupyterHub状态失败:', error);
            return null;
        }
    }

    /**
     * 获取用户的JupyterHub任务
     */
    async getUserTasks() {
        try {
            const response = await api.get('/jupyterhub/user-tasks');
            return response.data;
        } catch (error) {
            console.error('获取用户任务失败:', error);
            return [];
        }
    }

    /**
     * 验证当前用户的JupyterHub会话
     */
    async verifySession() {
        try {
            const response = await api.get('/auth/verify-jupyterhub-session');
            return response.data;
        } catch (error) {
            console.error('验证JupyterHub会话失败:', error);
            return { valid: false };
        }
    }

    /**
     * 刷新JupyterHub令牌
     */
    async refreshToken() {
        try {
            const response = await api.post('/auth/refresh-jupyterhub-token');
            return response.data;
        } catch (error) {
            console.error('刷新JupyterHub令牌失败:', error);
            throw error;
        }
    }
}

// 创建单例实例
const jupyterHubService = new JupyterHubService();

export default jupyterHubService;
