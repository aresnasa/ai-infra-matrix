import React, { useState, useEffect, Suspense } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Spin, message } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import Layout from './components/Layout';
import ErrorBoundary from './components/ErrorBoundary';
import LoadingFallback, { AdminLoadingFallback, ProjectLoadingFallback } from './components/LoadingFallback';
import AIAssistantFloat from './components/AIAssistantFloat';
import { useSmartPreload } from './hooks/usePagePreload';
import AuthPage from './pages/AuthPage';
import { authAPI } from './services/api';
import './App.css';

// 懒加载组件
const ProjectList = React.lazy(() => import('./pages/ProjectList'));
const ProjectDetail = React.lazy(() => import('./pages/ProjectDetail'));
const UserProfile = React.lazy(() => import('./pages/UserProfile'));

// 管理员页面懒加载
const AdminCenter = React.lazy(() => import('./pages/AdminCenter'));
const AdminUsers = React.lazy(() => import('./pages/AdminUsers'));
const AdminProjects = React.lazy(() => import('./pages/AdminProjects'));
const AdminLDAP = React.lazy(() => import('./pages/AdminLDAP'));
const AdminAuthSettings = React.lazy(() => import('./pages/AdminAuthSettings'));
const AdminTrash = React.lazy(() => import('./pages/AdminTrash'));
const AdminTest = React.lazy(() => import('./pages/AdminTest'));

// Kubernetes和Ansible管理页面懒加载
const KubernetesManagement = React.lazy(() => import('./pages/KubernetesManagement'));
const AnsibleManagement = React.lazy(() => import('./pages/AnsibleManagement'));

// AI助手管理页面懒加载
const AIAssistantManagement = React.lazy(() => import('./pages/AIAssistantManagement'));

// JupyterHub管理页面懒加载
const JupyterHubManagement = React.lazy(() => import('./pages/JupyterHubManagement'));

// JupyterHub主页面懒加载
const JupyterHubPage = React.lazy(() => import('./pages/JupyterHubPage'));
const EmbeddedJupyter = React.lazy(() => import('./pages/EmbeddedJupyter'));
const SlurmDashboard = React.lazy(() => import('./pages/SlurmDashboard'));
const GiteaEmbed = React.lazy(() => import('./pages/GiteaEmbed'));

function App() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [authChecked, setAuthChecked] = useState(false);
  const [permissionsLoaded, setPermissionsLoaded] = useState(false);

  // 智能预加载用户可能访问的页面
  useSmartPreload(user);

  useEffect(() => {
    initializeAuth();
    
    // 初始化favicon管理器
    const initializeFavicon = () => {
      if (window.faviconManager) {
        window.faviconManager.updateFavicon();
      }
    };
    
    initializeFavicon();
  }, []);

  // 监听用户状态变化，更新favicon
  useEffect(() => {
    if (window.faviconManager) {
      if (user) {
        // 用户已登录，显示正常状态
        window.faviconManager.resetToDefault();
        
        // 根据用户角色设置不同的favicon效果
        if (user.roles && user.roles.some(role => role.name === 'admin')) {
          window.faviconManager.addEffect('admin');
        }
      } else {
        // 用户未登录，可以设置特殊状态
        window.faviconManager.resetToDefault();
      }
    }
  }, [user]);

  // 检查token是否有效（本地检查，避免频繁请求后端）
  const isTokenValid = () => {
    const token = localStorage.getItem('token');
    const expires_at = localStorage.getItem('token_expires');
    
    if (!token || !expires_at) {
      console.log('Token或过期时间不存在');
      return false;
    }
    
    const expiryTime = new Date(expires_at).getTime();
    const currentTime = new Date().getTime();
    const bufferTime = 5 * 60 * 1000; // 5分钟缓冲时间
    
    if (currentTime + bufferTime >= expiryTime) {
      console.log('Token即将过期或已过期');
      return false;
    }
    
    return true;
  };

  // 初始化认证状态 - 确保完整的权限验证流程
  const initializeAuth = async () => {
    console.log('=== 开始初始化认证状态 ===');
    
    const token = localStorage.getItem('token');
    console.log('检查token:', token ? '存在' : '不存在');
    
    if (!token) {
      console.log('无token，用户未认证');
      clearUserState();
      setAuthChecked(true);
      setLoading(false);
      return;
    }
    
    // 先检查token本地有效性
    if (!isTokenValid()) {
      console.log('Token已过期，尝试刷新...');
      
      try {
        // 尝试刷新token
        const response = await authAPI.refreshToken();
        const { token: newToken, expires_at } = response.data;
        
        localStorage.setItem('token', newToken);
        localStorage.setItem('token_expires', expires_at);
        console.log('Token刷新成功');
        
        // 刷新后验证用户信息
        await verifyUserWithBackend();
      } catch (error) {
        console.log('Token刷新失败，需要重新登录:', error.message);
        clearUserState();
      }
    } else {
      // Token有效，检查是否已有用户信息
      const savedUser = localStorage.getItem('user');
      if (savedUser) {
        try {
          const userData = JSON.parse(savedUser);
          setUser(userData);
          setPermissionsLoaded(true);
          console.log('使用缓存的用户信息');
        } catch (error) {
          console.log('缓存用户信息解析失败，重新获取');
          await verifyUserWithBackend();
        }
      } else {
        // 没有缓存用户信息，从后端获取
        await verifyUserWithBackend();
      }
    }
    
    setAuthChecked(true);
    setLoading(false);
    console.log('=== 认证状态初始化完成 ===');
  };

  // 从后端验证用户并获取完整权限信息
  const verifyUserWithBackend = async (retryCount = 0) => {
    try {
      console.log('正在验证token并获取用户权限...');
      
      // 确保 localStorage 的写入已经完成
      await new Promise(resolve => setTimeout(resolve, 50));
      
      const response = await authAPI.getProfile();
      const userData = response.data;
      
      console.log('后端返回用户数据:', userData);
      console.log('用户角色:', userData.role);
      console.log('用户权限组:', userData.roles);
      
      // 设置用户状态
      setUser(userData);
      setPermissionsLoaded(true);
      
      // 更新本地存储
      localStorage.setItem('user', JSON.stringify(userData));
      
      console.log('✅ 用户权限验证成功');
      
    } catch (error) {
      console.log('❌ Token验证失败:', error.message);
      
      // 判断是否是token过期错误且允许重试
      if (error.response?.status === 401 && retryCount < 1) {
        console.log('认证失败，尝试刷新token...');
        try {
          const refreshResponse = await authAPI.refreshToken();
          const { token: newToken, expires_at } = refreshResponse.data;
          
          localStorage.setItem('token', newToken);
          localStorage.setItem('token_expires', expires_at);
          console.log('Token刷新成功，重新验证用户信息');
          
          // 递归调用，但限制重试次数
          await verifyUserWithBackend(retryCount + 1);
          return;
        } catch (refreshError) {
          console.log('Token刷新失败:', refreshError.message);
        }
      }
      
      console.log('清除认证状态...');
      clearUserState();
    }
  };

  // 清除用户状态
  const clearUserState = () => {
    setUser(null);
    setPermissionsLoaded(false);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('token_expires');
  };

  // 处理登录成功 - 重新验证权限并设置SSO
  const handleLogin = async (userData) => {
    console.log('=== 处理登录成功 ===');
    console.log('登录返回的用户数据:', userData);
    
    // 重置加载状态
    setLoading(true);
    setAuthChecked(false);
    setPermissionsLoaded(false);
    
    try {
      // 使用增强的认证服务设置认证状态
      if (window.authService) {
        await window.authService.setAuthData(
          userData.token, 
          userData.expires_at, 
          userData.user || userData
        );
        
        // 设置SSO cookies
        await window.authService.setupSSOCookies(
          userData.token, 
          userData.user || userData
        );
        
        console.log('✅ SSO状态设置完成');
      } else {
        // 降级处理：手动设置
        localStorage.setItem('token', userData.token);
        if (userData.expires_at) {
          localStorage.setItem('token_expires', userData.expires_at);
        }
        if (userData.user) {
          localStorage.setItem('user', JSON.stringify(userData.user));
        }
        
        // 手动设置SSO cookie
        const maxAge = 3600;
        const cookieOptions = `path=/; max-age=${maxAge}; SameSite=Lax`;
        document.cookie = `ai_infra_token=${userData.token}; ${cookieOptions}`;
        document.cookie = `jwt_token=${userData.token}; ${cookieOptions}`;
        
        console.log('✅ 降级SSO状态设置完成');
      }
      
      // 重新初始化认证状态
      await initializeAuth();
      
      // 立即从后端获取最新的权限信息
      await verifyUserWithBackend();
      
    } catch (error) {
      console.error('设置登录状态失败:', error);
      message.error('登录状态设置失败');
      
      // 如果获取最新权限失败，使用登录返回的数据作为备用
      setUser(userData);
      setPermissionsLoaded(true);
      localStorage.setItem('user', JSON.stringify(userData));
    }
    
    setAuthChecked(true);
    setLoading(false);
    console.log('=== 登录处理完成 ===');
  };

  const handleLogout = () => {
    console.log('=== 处理登出 ===');
    authAPI.logout();
    clearUserState();
    setAuthChecked(false);
  };

  // 渲染加载状态
  if (loading || !authChecked || (user && !permissionsLoaded)) {
    return (
      <div style={{ 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        height: '100vh',
        flexDirection: 'column',
        gap: '16px',
        backgroundColor: '#f5f5f5'
      }}>
        <Spin size="large" />
        <div style={{ color: '#666', fontSize: '14px', textAlign: 'center' }}>
          {loading ? '正在验证身份...' : 
           !authChecked ? '正在检查认证状态...' : 
           '正在加载权限信息...'}
        </div>
        <div style={{ color: '#999', fontSize: '12px' }}>
          请稍候，系统正在确保您获得正确的访问权限
        </div>
      </div>
    );
  }

  // 懒加载的加载组件
  const LazyLoadingSpinner = () => (
    <div style={{ 
      display: 'flex', 
      justifyContent: 'center', 
      alignItems: 'center', 
      minHeight: '200px' 
    }}>
      <Spin size="large" tip="正在加载页面..." />
    </div>
  );

  console.log('=== App渲染状态 ===');
  console.log('用户已认证:', !!user);
  console.log('权限已加载:', permissionsLoaded);
  console.log('用户角色:', user?.role);
  console.log('用户权限组:', user?.roles);
  console.log('==================');

  return (
    <ConfigProvider locale={zhCN}>
      <ErrorBoundary>
        <Router>
          {!user ? (
            <AuthPage onLogin={handleLogin} />
          ) : (
            <Layout user={user} onLogout={handleLogout}>
              <ErrorBoundary>
                <Suspense fallback={<LoadingFallback />}>
                  <Routes>
                    <Route path="/" element={<Navigate to="/projects" replace />} />
                    {/* Legacy redirects for Slurm paths to keep old links working */}
                    <Route path="/jupyter/with-slurm" element={<Navigate to="/slurm" replace />} />
                    <Route path="/jupyter/slurm" element={<Navigate to="/slurm" replace />} />
                    <Route 
                      path="/projects" 
                      element={
                        <Suspense fallback={<ProjectLoadingFallback />}>
                          <ProjectList />
                        </Suspense>
                      } 
                    />
                    <Route 
                      path="/gitea" 
                      element={
                        <Suspense fallback={<LazyLoadingSpinner />}>
                          <GiteaEmbed />
                        </Suspense>
                      } 
                    />
                    <Route 
                      path="/projects/:id" 
                      element={
                        <Suspense fallback={<ProjectLoadingFallback />}>
                          <ProjectDetail />
                        </Suspense>
                      } 
                    />
                    <Route path="/profile" element={<UserProfile />} />
                    
                    {/* Kubernetes和Ansible管理页面 */}
                    <Route 
                      path="/kubernetes" 
                      element={
                        <Suspense fallback={<LazyLoadingSpinner />}>
                          <KubernetesManagement />
                        </Suspense>
                      } 
                    />
                    <Route 
                      path="/ansible" 
                      element={
                        <Suspense fallback={<LazyLoadingSpinner />}>
                          <AnsibleManagement />
                        </Suspense>
                      } 
                    />
                    
                    {/* Embedded Jupyter page (same-origin iframe) */}
                    <Route 
                      path="/jupyter" 
                      element={
                        <Suspense fallback={<LazyLoadingSpinner />}>
                          <EmbeddedJupyter />
                        </Suspense>
                      } 
                    />

                    {/* Slurm dashboard page */}
                    <Route 
                      path="/slurm" 
                      element={
                        <Suspense fallback={<LazyLoadingSpinner />}>
                          <SlurmDashboard />
                        </Suspense>
                      } 
                    />
                    
                    {/* 管理员路由 - 支持 admin 和 super-admin 角色 */}
                    {(user?.role === 'admin' || user?.role === 'super-admin' || (user?.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'))) && (
                      <>
                        <Route path="/admin" element={<AdminCenter />} />
                        <Route 
                          path="/admin/test" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AdminTest />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/users" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AdminUsers />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/projects" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AdminProjects />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/auth" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AdminAuthSettings />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/ldap" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AdminLDAP />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/trash" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AdminTrash />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/ai-assistant" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <AIAssistantManagement />
                            </Suspense>
                          } 
                        />
                        <Route 
                          path="/admin/jupyterhub" 
                          element={
                            <Suspense fallback={<AdminLoadingFallback />}>
                              <JupyterHubManagement />
                            </Suspense>
                          } 
                        />
                      </>
                    )}
                    <Route path="/login" element={<Navigate to="/projects" replace />} />
                  </Routes>
                </Suspense>
              </ErrorBoundary>
            </Layout>
          )}
          {/* AI助手悬浮组件 - 只在用户登录后显示 */}
          {user && <AIAssistantFloat />}
        </Router>
      </ErrorBoundary>
    </ConfigProvider>
  );
}

export default App;
