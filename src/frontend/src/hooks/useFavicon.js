import { useEffect, useCallback } from 'react';
import { useLocation } from 'react-router-dom';

/**
 * 动态Favicon Hook
 * 在React组件中使用，自动根据路由更新favicon
 */
export const useFavicon = () => {
  const location = useLocation();

  // 更新favicon
  const updateFavicon = useCallback(() => {
    if (window.faviconManager) {
      window.faviconManager.updateFavicon();
    }
  }, []);

  // 设置特定页面图标
  const setPageIcon = useCallback((pageType) => {
    if (window.faviconManager) {
      window.faviconManager.setPageIcon(pageType);
    }
  }, []);

  // 添加动态效果
  const addEffect = useCallback((effect) => {
    if (window.faviconManager) {
      window.faviconManager.addEffect(effect);
    }
  }, []);

  // 恢复默认图标
  const resetToDefault = useCallback(() => {
    if (window.faviconManager) {
      window.faviconManager.resetToDefault();
    }
  }, []);

  // 监听路由变化
  useEffect(() => {
    updateFavicon();
  }, [location.pathname, updateFavicon]);

  return {
    updateFavicon,
    setPageIcon,
    addEffect,
    resetToDefault
  };
};

/**
 * 页面标题和Favicon Hook
 * 同时管理页面标题和favicon
 */
export const usePageMeta = (title, pageType = null) => {
  const { setPageIcon, resetToDefault } = useFavicon();

  useEffect(() => {
    // 设置页面标题
    const originalTitle = document.title;
    document.title = title ? `${title} - AI-Infra-Matrix` : 'AI-Infra-Matrix';

    // 设置favicon
    if (pageType) {
      setPageIcon(pageType);
    }

    // 清理函数
    return () => {
      document.title = originalTitle;
      if (pageType) {
        resetToDefault();
      }
    };
  }, [title, pageType, setPageIcon, resetToDefault]);
};

/**
 * 加载状态Favicon Hook
 * 在异步操作时显示加载效果
 */
export const useLoadingFavicon = (isLoading) => {
  const { addEffect, updateFavicon } = useFavicon();

  useEffect(() => {
    if (isLoading) {
      addEffect('loading');
    } else {
      // 加载完成后恢复正常图标
      setTimeout(updateFavicon, 100);
    }
  }, [isLoading, addEffect, updateFavicon]);
};

/**
 * 状态Favicon Hook
 * 根据操作状态显示不同效果
 */
export const useStatusFavicon = () => {
  const { addEffect } = useFavicon();

  const showSuccess = useCallback(() => {
    addEffect('success');
  }, [addEffect]);

  const showError = useCallback(() => {
    addEffect('error');
  }, [addEffect]);

  const showLoading = useCallback(() => {
    addEffect('loading');
  }, [addEffect]);

  return {
    showSuccess,
    showError,
    showLoading
  };
};

export default useFavicon;
