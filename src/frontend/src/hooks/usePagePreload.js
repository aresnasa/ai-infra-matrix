import { useEffect } from 'react';

// 页面预加载钩子
export const usePagePreload = (preloadPages = []) => {
  useEffect(() => {
    const preloadPromises = preloadPages.map(async (page) => {
      try {
        // 根据页面类型预加载对应的组件
        switch (page) {
          case 'admin-users':
            await import('../pages/AdminUsers');
            break;
          case 'admin-projects':
            await import('../pages/AdminProjects');
            break;
          case 'admin-auth':
            await import('../pages/AdminAuthSettings');
            break;
          case 'admin-ldap':
            await import('../pages/AdminLDAP');
            break;
          case 'admin-test':
            await import('../pages/AdminTest');
            break;
          case 'admin-trash':
            await import('../pages/AdminTrash');
            break;
          case 'project-detail':
            await import('../pages/ProjectDetail');
            break;
          case 'user-profile':
            await import('../pages/UserProfile');
            break;
          default:
            console.warn(`Unknown page for preload: ${page}`);
        }
      } catch (error) {
        // 静默处理预加载失败，不影响主要功能
        console.debug(`Failed to preload ${page}:`, error);
      }
    });

    // 在空闲时间执行预加载
    if (window.requestIdleCallback) {
      window.requestIdleCallback(() => {
        Promise.all(preloadPromises);
      });
    } else {
      // 降级处理：延迟执行
      setTimeout(() => {
        Promise.all(preloadPromises);
      }, 1000);
    }
  }, [preloadPages]);
};

// 根据用户角色智能预加载
export const useSmartPreload = (user) => {
  const isAdmin = user?.role === 'admin' || user?.role === 'super-admin' || 
    (user?.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));

  const preloadPages = [
    'user-profile', // 所有用户都可能访问
    'project-detail', // 项目详情页面
    ...(isAdmin ? [
      'admin-users', // 管理员最常用的功能
      'admin-auth', // LDAP设置
      'admin-test' // 系统测试
    ] : [])
  ];

  usePagePreload(preloadPages);
};

export default usePagePreload;
