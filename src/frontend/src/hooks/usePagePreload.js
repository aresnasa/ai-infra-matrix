import { useEffect, useRef } from 'react';

// 优化的页面预加载钩子
export const useSmartPreload = (user) => {
  const preloadedRef = useRef(new Set());
  const preloadTimeoutRef = useRef(null);

  useEffect(() => {
    if (!user) {
      // 用户未登录，清理预加载状态
      preloadedRef.current.clear();
      if (preloadTimeoutRef.current) {
        clearTimeout(preloadTimeoutRef.current);
        preloadTimeoutRef.current = null;
      }
      return;
    }

    // 延迟预加载，避免影响首次渲染
    preloadTimeoutRef.current = setTimeout(() => {
      preloadUserPages(user);
    }, 3000); // 3秒后开始预加载

    return () => {
      if (preloadTimeoutRef.current) {
        clearTimeout(preloadTimeoutRef.current);
        preloadTimeoutRef.current = null;
      }
    };
  }, [user]);

  const preloadUserPages = async (user) => {
    const pagesToPreload = [];

    // 根据用户角色确定需要预加载的页面
    if (user.role === 'admin' || user.role === 'super-admin' || 
        (user.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'))) {
      pagesToPreload.push(
        'admin-users',
        'admin-projects', 
        'admin-auth'
      );
    }

    // 所有用户都可能访问的页面
    pagesToPreload.push('user-profile', 'project-detail');

    // 批量预加载，使用requestIdleCallback优化性能
    const preloadBatch = async (pages) => {
      for (const page of pages) {
        if (preloadedRef.current.has(page)) {
          continue; // 已预加载，跳过
        }

        try {
          await preloadPage(page);
          preloadedRef.current.add(page);
          
          // 添加小延迟，避免阻塞主线程
          await new Promise(resolve => setTimeout(resolve, 100));
        } catch (error) {
          console.debug(`Failed to preload ${page}:`, error);
        }
      }
    };

    if (window.requestIdleCallback) {
      window.requestIdleCallback(() => {
        preloadBatch(pagesToPreload);
      }, { timeout: 10000 });
    } else {
      // 降级处理
      setTimeout(() => {
        preloadBatch(pagesToPreload);
      }, 5000);
    }
  };

  const preloadPage = async (page) => {
    switch (page) {
      case 'admin-users':
        return import('../pages/AdminUsers');
      case 'admin-projects':
        return import('../pages/AdminProjects');
      case 'admin-auth':
        return import('../pages/AdminAuthSettings');
      case 'admin-ldap':
        return import('../pages/AdminLDAPCenter');
      case 'admin-test':
        return import('../pages/AdminTest');
      case 'admin-trash':
        return import('../pages/AdminTrash');
      case 'project-detail':
        return import('../pages/ProjectDetail');
      case 'user-profile':
        return import('../pages/UserProfile');
      case 'jupyter-management':
        return import('../pages/JupyterHubManagement');
      case 'ai-assistant':
        return import('../pages/AIAssistantManagement');
      default:
        console.warn(`Unknown page for preload: ${page}`);
        return Promise.resolve();
    }
  };
};

// 页面预加载钩子（兼容性）
export const usePagePreload = (preloadPages = []) => {
  useEffect(() => {
    if (!preloadPages.length) return;

    const preloadPromises = preloadPages.map(async (page) => {
      try {
        // 延迟预加载
        await new Promise(resolve => setTimeout(resolve, 2000));
        
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
            await import('../pages/AdminLDAPCenter');
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
      }, { timeout: 10000 });
    } else {
      // 降级处理：延迟执行
      setTimeout(() => {
        Promise.all(preloadPromises);
      }, 3000);
    }
  }, [preloadPages]);
};
