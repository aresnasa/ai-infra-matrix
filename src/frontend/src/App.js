import React, { useState, useEffect, Suspense } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Spin, message, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import Layout from './components/Layout';
import { ThemeProvider, useTheme } from './hooks/useTheme';
import ErrorBoundary from './components/ErrorBoundary';
import LoadingFallback, { AdminLoadingFallback, ProjectLoadingFallback } from './components/LoadingFallback';
import EnhancedLoading from './components/EnhancedLoading';
import withLazyLoading from './components/withLazyLoading';
import AIAssistantFloat from './components/AIAssistantFloat';
import { useSmartPreload } from './hooks/usePagePreload';
import { useAPIHealth } from './hooks/useAPIHealth'; // 使用优化版本
import { usePerformanceMonitor } from './hooks/usePerformanceMonitor';
import { I18nProvider, useI18n } from './hooks/useI18n';
import AuthPage from './pages/AuthPage';
import { authAPI } from './services/api';
import { authCache } from './utils/authCache';
import { apiRequestManager } from './utils/apiRequestManager';
import { hasRoutePermission } from './utils/permissions';
import ProtectedRoute, { AdminProtectedRoute, TeamProtectedRoute } from './components/ProtectedRoute';
import './App.css';

// 懒加载组件 - 使用增强的懒加载包装
const ProjectList = withLazyLoading(React.lazy(() => import('./pages/ProjectList')), {
  loadingText: '正在加载项目列表...'
});
const ProjectDetail = withLazyLoading(React.lazy(() => import('./pages/ProjectDetail')), {
  loadingText: '正在加载项目详情...'
});
const UserProfile = withLazyLoading(React.lazy(() => import('./pages/UserProfile')), {
  loadingText: '正在加载用户资料...'
});

// 管理员页面懒加载
const AdminCenter = withLazyLoading(React.lazy(() => import('./pages/AdminCenter')), {
  loadingText: '正在加载管理中心...'
});
const AdminUsers = withLazyLoading(React.lazy(() => import('./pages/AdminUsers')), {
  loadingText: '正在加载用户管理...'
});
const AdminProjects = withLazyLoading(React.lazy(() => import('./pages/AdminProjects')), {
  loadingText: '正在加载项目管理...'
});
const AdminLDAP = withLazyLoading(React.lazy(() => import('./pages/AdminLDAP')), {
  loadingText: '正在加载LDAP配置...'
});
const AdminLDAPCenter = withLazyLoading(React.lazy(() => import('./pages/AdminLDAPCenter')), {
  loadingText: '正在加载LDAP管理中心...'
});
const AdminAuthSettings = withLazyLoading(React.lazy(() => import('./pages/AdminAuthSettings')), {
  loadingText: '正在加载认证设置...'
});
const AdminTrash = withLazyLoading(React.lazy(() => import('./pages/AdminTrash')), {
  loadingText: '正在加载回收站...'
});
const AdminTest = withLazyLoading(React.lazy(() => import('./pages/AdminTest')), {
  loadingText: '正在加载系统测试...'
});
const KafkaUIPage = withLazyLoading(React.lazy(() => import('./pages/KafkaUIPage')), {
  loadingText: '正在加载Kafka UI...'
});

// Kubernetes和Ansible管理页面懒加载
const KubernetesManagement = withLazyLoading(React.lazy(() => import('./pages/KubernetesManagement')), {
  loadingText: '正在加载Kubernetes管理...'
});
const EnhancedKubernetesManagement = withLazyLoading(React.lazy(() => import('./pages/EnhancedKubernetesManagement')), {
  loadingText: '正在加载增强Kubernetes管理...'
});
const AnsibleManagement = withLazyLoading(React.lazy(() => import('./pages/AnsibleManagement')), {
  loadingText: '正在加载Ansible管理...'
});

// AI助手管理页面懒加载
const AIAssistantManagement = withLazyLoading(React.lazy(() => import('./pages/AIAssistantManagement')), {
  loadingText: '正在加载AI助手管理...'
});

// AI助手聊天页面懒加载 (OpenAI风格)
const AIAssistantChat = withLazyLoading(React.lazy(() => import('./pages/AIAssistantChat')), {
  loadingText: '正在加载AI助手...'
});

// 调试页面懒加载
const DebugPage = withLazyLoading(React.lazy(() => import('./pages/DebugPage')), {
  loadingText: '正在加载调试页面...'
});

// JupyterHub管理页面懒加载
const JupyterHubManagement = withLazyLoading(React.lazy(() => import('./pages/JupyterHubManagement')), {
  loadingText: '正在加载JupyterHub管理...'
});

// JupyterHub主页面懒加载
const JupyterHubPage = withLazyLoading(React.lazy(() => import('./pages/JupyterHubPage')), {
  loadingText: '正在加载JupyterHub...'
});
const EmbeddedJupyter = withLazyLoading(React.lazy(() => import('./pages/EmbeddedJupyter')), {
  loadingText: '正在加载Jupyter环境...'
});
const SaltStackDashboard = withLazyLoading(React.lazy(() => import('./pages/SaltStackDashboard')), {
  loadingText: '正在加载SaltStack仪表板...'
});
const SlurmScalingPage = withLazyLoading(React.lazy(() => import('./pages/SlurmScalingPage')), {
  loadingText: '正在加载SLURM页面...'
});
const SlurmTasksPage = withLazyLoading(React.lazy(() => import('./pages/SlurmTasksPage')), {
  loadingText: '正在加载任务管理页面...'
});
const MonitoringPage = withLazyLoading(React.lazy(() => import('./pages/MonitoringPage')), {
  loadingText: '正在加载监控仪表板...'
});
const JobManagement = withLazyLoading(React.lazy(() => import('./pages/JobManagement')), {
  loadingText: '正在加载作业管理...'
});
const JobTemplateManagement = withLazyLoading(React.lazy(() => import('./pages/JobTemplateManagement')), {
  loadingText: '正在加载模板管理...'
});
const SSHConnectionTest = withLazyLoading(React.lazy(() => import('./pages/SSHConnectionTest')), {
  loadingText: '正在加载SSH测试工具...'
});
const GiteaEmbed = withLazyLoading(React.lazy(() => import('./pages/GiteaEmbed')), {
  loadingText: '正在加载Gitea...'
});

// 对象存储相关页面
const ObjectStoragePage = withLazyLoading(React.lazy(() => import('./pages/ObjectStoragePage')), {
  loadingText: '正在加载对象存储管理...'
});
const StorageConsolePage = withLazyLoading(React.lazy(() => import('./pages/StorageConsolePage')), {
  loadingText: '正在加载存储控制台...'
});
const ObjectStorageConfigPage = withLazyLoading(React.lazy(() => import('./pages/admin/ObjectStorageConfigPage')), {
  loadingText: '正在加载对象存储配置...'
});

// 安全管理页面
const SecuritySettings = withLazyLoading(React.lazy(() => import('./pages/admin/SecuritySettings')), {
  loadingText: '正在加载安全管理...'
});

// 新增功能页面懒加载
const EnhancedUserManagement = withLazyLoading(React.lazy(() => import('./pages/EnhancedUserManagement')), {
  loadingText: '正在加载增强用户管理...'
});
const RoleTemplateManagement = withLazyLoading(React.lazy(() => import('./pages/RoleTemplateManagement')), {
  loadingText: '正在加载角色模板管理...'
});
const PermissionApprovalPage = withLazyLoading(React.lazy(() => import('./pages/PermissionApprovalPage')), {
  loadingText: '正在加载权限审批管理...'
});
const FileBrowser = withLazyLoading(React.lazy(() => import('./pages/FileBrowser')),
  { loadingText: '正在加载文件浏览器...' }
);

function App() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [authChecked, setAuthChecked] = useState(false);
  const [permissionsLoaded, setPermissionsLoaded] = useState(false);

  // API健康监控 - 使用优化配置
  const { apiHealth, isHealthy, isDegraded, isDown } = useAPIHealth({
    checkInterval: 5 * 60 * 1000, // 5分钟检查一次
    enableAutoCheck: true,
    showNotifications: false, // 关闭通知，减少干扰
    onlyCheckOnFocus: true // 只在窗口获得焦点时检查
  });

  // 智能预加载用户可能访问的页面
  useSmartPreload(user);

  // 性能监控 - 仅在开发环境或特定条件下启用
  const performanceTools = usePerformanceMonitor(
    process.env.NODE_ENV === 'development' || 
    localStorage.getItem('enable_performance_monitor') === 'true'
  );

  useEffect(() => {
    initializeAuth();
    
    // 初始化favicon管理器
    const initializeFavicon = () => {
      if (window.faviconManager) {
        window.faviconManager.updateFavicon();
      }
    };
    
    initializeFavicon();
    
    // 清理过期缓存
    const cleanupInterval = setInterval(() => {
      apiRequestManager.cleanExpiredCache();
    }, 10 * 60 * 1000); // 每10分钟清理一次
    
    return () => {
      clearInterval(cleanupInterval);
    };
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

  // 初始化认证状态 - 优化版本，减少API调用
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
    if (authCache.isTokenNearExpiry()) {
      console.log('Token即将过期，尝试刷新...');
      
      try {
        // 使用去重机制避免重复刷新请求
        await authCache.withDeduplication('token-refresh', async () => {
          const response = await authAPI.refreshToken();
          const { token: newToken, expires_at } = response.data;
          
          localStorage.setItem('token', newToken);
          localStorage.setItem('token_expires', expires_at);
          console.log('Token刷新成功');
          
          return response;
        });
        
        // 刷新后验证用户信息
        await verifyUserWithBackend();
      } catch (error) {
        console.log('Token刷新失败，需要重新登录:', error.message);
        clearUserState();
      }
    } else {
      // Token有效，检查是否已有缓存用户信息
      const cachedUser = authCache.getCachedUser();
      if (cachedUser && !authCache.shouldRefreshAuth()) {
        // 使用缓存的用户信息
        setUser(cachedUser);
        setPermissionsLoaded(true);
        console.log('使用缓存的用户信息');
      } else {
        // 缓存过期或不存在，从后端获取
        await verifyUserWithBackend();
      }
    }
    
    setAuthChecked(true);
    setLoading(false);
    console.log('=== 认证状态初始化完成 ===');
  };

  // 从后端验证用户并获取完整权限信息 - 优化版本
  const verifyUserWithBackend = async (retryCount = 0) => {
    try {
      console.log('正在验证token并获取用户权限...');
      
      // 使用去重机制避免重复请求
      const userData = await authCache.withDeduplication('user-profile', async () => {
        const response = await authAPI.getProfile();
        return response.data;
      });
      
      console.log('后端返回用户数据:', userData);
      console.log('用户角色:', userData.role);
      console.log('用户权限组:', userData.roles);
      
      // 设置用户状态
      setUser(userData);
      setPermissionsLoaded(true);
      
      // 缓存用户信息
      authCache.cacheUser(userData);
      
      console.log('✅ 用户权限验证成功');
      
    } catch (error) {
      console.log('❌ Token验证失败:', error.message);
      
      // 判断是否是token过期错误且允许重试
      if (error.response?.status === 401 && retryCount < 1) {
        console.log('认证失败，尝试刷新token...');
        try {
          await authCache.withDeduplication('token-refresh-retry', async () => {
            const refreshResponse = await authAPI.refreshToken();
            const { token: newToken, expires_at } = refreshResponse.data;
            
            localStorage.setItem('token', newToken);
            localStorage.setItem('token_expires', expires_at);
            console.log('Token刷新成功，重新验证用户信息');
            
            return refreshResponse;
          });
          
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

  // 清除用户状态 - 优化版本
  const clearUserState = () => {
    setUser(null);
    setPermissionsLoaded(false);
    
    // 清理所有缓存
    authCache.clearAll();
    apiRequestManager.clearCache();
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
    
    // 立即清除本地状态
    clearUserState();
    
    // 调用authAPI登出，这会清除token并重定向
    authAPI.logout();
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

  // 懒加载的加载组件 - 包含API状态监控
  const LazyLoadingSpinner = ({ loadingText = '正在加载页面...', ...props }) => (
    <EnhancedLoading
      loading={true}
      apiHealth={apiHealth}
      showAPIStatus={true}
      loadingText={loadingText}
      {...props}
    />
  );

  console.log('=== App渲染状态 ===');
  console.log('用户已认证:', !!user);
  console.log('权限已加载:', permissionsLoaded);
  console.log('用户角色:', user?.role);
  console.log('用户权限组:', user?.roles);
  console.log('==================');

  return (
    <ThemeProvider>
      <I18nProvider>
        <AppContent 
          user={user}
          handleLogin={handleLogin}
          handleLogout={handleLogout}
          apiHealth={apiHealth}
          LazyLoadingSpinner={LazyLoadingSpinner}
        />
      </I18nProvider>
    </ThemeProvider>
  );
}

// 内部组件：使用 I18n 和 Theme 上下文来切换 Ant Design 语言和主题
function AppContent({ user, handleLogin, handleLogout, apiHealth, LazyLoadingSpinner }) {
  const { locale } = useI18n();
  const { isDark } = useTheme();
  
  // Ant Design 语言包映射
  const antdLocale = locale === 'en-US' ? enUS : zhCN;
  
  // Ant Design 主题配置
  const themeConfig = {
    algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      colorPrimary: '#1890ff',
      borderRadius: 6,
    },
  };

  return (
    <ConfigProvider locale={antdLocale} theme={themeConfig}>
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
                    
                    {/* AI助手聊天页面 (OpenAI风格) - 允许所有登录用户 */}
                    <Route
                      path="/ai-chat"
                      element={
                        <Suspense fallback={<LazyLoadingSpinner />}>
                          <AIAssistantChat />
                        </Suspense>
                      }
                    />
                    
                    {/* Kubernetes和Ansible管理页面 - 只允许SRE团队 */}
                    <Route
                      path="/kubernetes"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <KubernetesManagement />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />
                    <Route
                      path="/kubernetes/resources"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <EnhancedKubernetesManagement />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />
                    <Route
                      path="/ansible"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <AnsibleManagement />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* Embedded Jupyter page - 允许数据开发和SRE团队 */}
                    <Route
                      path="/jupyter"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <EmbeddedJupyter />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* JupyterHub - 允许数据开发和SRE团队 */}
                    <Route
                      path="/jupyterhub"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <JupyterHubPage />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* Slurm dashboard page - 允许数据开发和SRE团队 */}
                    <Route
                      path="/slurm"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <SlurmScalingPage />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* SLURM自动化页面 - 重定向到主SLURM页面 */}
                    <Route
                      path="/slurm-scaling"
                      element={<Navigate to="/slurm" replace />}
                    />

                    {/* SLURM任务管理页面 */}
                    <Route
                      path="/slurm-tasks"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <SlurmTasksPage />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* Job management page - 允许数据开发和SRE团队 */}
                    <Route
                      path="/jobs"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <JobManagement />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* 对象存储管理页面 - 允许数据开发和SRE团队 */}
                    <Route
                      path="/object-storage"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <ObjectStoragePage />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* 存储控制台页面 - 支持 SeaweedFS/MinIO，允许数据开发和SRE团队 */}
                    <Route
                      path="/object-storage/console/:configId"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <StorageConsolePage />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* Job template management page - 允许数据开发和SRE团队 */}
                    <Route
                      path="/job-templates"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <JobTemplateManagement />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* SSH Connection Test - 允许数据开发和SRE团队 */}
                    <Route
                      path="/ssh-test"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <SSHConnectionTest />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* File browser - 允许数据开发和SRE团队 */}
                    <Route
                      path="/files"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['data-developer', 'sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <FileBrowser />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* SaltStack dashboard page - 只允许SRE团队 */}
                    <Route
                      path="/saltstack"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['sre']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <SaltStackDashboard />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />

                    {/* Monitoring dashboard page (Nightingale) - 允许管理员访问 */}
                    <Route
                      path="/monitoring"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <MonitoringPage />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />

                    {/* Kafka UI management page - 只允许审计团队 */}
                    <Route
                      path="/kafka-ui"
                      element={
                        <TeamProtectedRoute user={user} allowedTeams={['audit']}>
                          <Suspense fallback={<LazyLoadingSpinner />}>
                            <KafkaUIPage />
                          </Suspense>
                        </TeamProtectedRoute>
                      }
                    />
                    
                    {/* 管理员路由 - 只允许管理员访问 */}
                    <Route
                      path="/admin"
                      element={
                        <AdminProtectedRoute user={user}>
                          <AdminCenter />
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/test"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AdminTest />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/users"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AdminUsers />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/role-templates"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <RoleTemplateManagement />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/permission-approvals"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <PermissionApprovalPage />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/projects"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AdminProjects />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/auth"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AdminAuthSettings />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/ldap"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AdminLDAPCenter />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/trash"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AdminTrash />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/debug"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <DebugPage />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/ai-assistant"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <AIAssistantManagement />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/jupyterhub"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <JupyterHubManagement />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/object-storage"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <ObjectStorageConfigPage />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    <Route
                      path="/admin/security"
                      element={
                        <AdminProtectedRoute user={user}>
                          <Suspense fallback={<AdminLoadingFallback />}>
                            <SecuritySettings />
                          </Suspense>
                        </AdminProtectedRoute>
                      }
                    />
                    {/* 移除自动重定向 - 让 AuthPage 处理登录后的导航 */}
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