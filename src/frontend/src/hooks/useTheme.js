/**
 * 主题切换 Hook
 * 支持暗黑模式、光明模式和跟随系统切换
 */

import React, { createContext, useContext, useState, useCallback, useMemo, useEffect } from 'react';

// 主题存储 Key
const THEME_STORAGE_KEY = 'ai_infra_theme';

// 主题模式枚举
export const ThemeMode = {
  LIGHT: 'light',
  DARK: 'dark',
  SYSTEM: 'system',
};

// 创建上下文
const ThemeContext = createContext(null);

/**
 * 获取系统主题偏好
 * @returns {'light' | 'dark'}
 */
const getSystemTheme = () => {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'light';
};

/**
 * Theme Provider 组件
 */
export const ThemeProvider = ({ children }) => {
  // 从 localStorage 获取保存的主题模式，默认跟随系统
  const [themeMode, setThemeModeState] = useState(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem(THEME_STORAGE_KEY);
      if (saved && Object.values(ThemeMode).includes(saved)) {
        return saved;
      }
    }
    return ThemeMode.SYSTEM;
  });

  // 系统主题
  const [systemTheme, setSystemTheme] = useState(getSystemTheme);

  // 实际应用的主题
  const actualTheme = useMemo(() => {
    if (themeMode === ThemeMode.SYSTEM) {
      return systemTheme;
    }
    return themeMode;
  }, [themeMode, systemTheme]);

  // 是否是暗黑模式
  const isDark = actualTheme === ThemeMode.DARK;

  // 监听系统主题变化
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = (e) => {
      setSystemTheme(e.matches ? 'dark' : 'light');
    };

    // 添加监听
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', handleChange);
    } else {
      // 兼容旧浏览器
      mediaQuery.addListener(handleChange);
    }

    return () => {
      if (mediaQuery.removeEventListener) {
        mediaQuery.removeEventListener('change', handleChange);
      } else {
        mediaQuery.removeListener(handleChange);
      }
    };
  }, []);

  // 切换主题模式
  const setThemeMode = useCallback((mode) => {
    if (Object.values(ThemeMode).includes(mode)) {
      setThemeModeState(mode);
      if (typeof window !== 'undefined') {
        localStorage.setItem(THEME_STORAGE_KEY, mode);
      }
    }
  }, []);

  // 快捷切换到下一个主题
  const toggleTheme = useCallback(() => {
    const modes = [ThemeMode.LIGHT, ThemeMode.DARK, ThemeMode.SYSTEM];
    const currentIndex = modes.indexOf(themeMode);
    const nextIndex = (currentIndex + 1) % modes.length;
    setThemeMode(modes[nextIndex]);
  }, [themeMode, setThemeMode]);

  // 应用主题到 DOM
  useEffect(() => {
    if (typeof document !== 'undefined') {
      // 更新 HTML 属性
      document.documentElement.setAttribute('data-theme', actualTheme);
      
      // 更新 body 类名
      document.body.classList.remove('theme-light', 'theme-dark');
      document.body.classList.add(`theme-${actualTheme}`);
      
      // 更新 meta theme-color
      const metaThemeColor = document.querySelector('meta[name="theme-color"]');
      if (metaThemeColor) {
        metaThemeColor.setAttribute('content', isDark ? '#141414' : '#ffffff');
      }
    }
  }, [actualTheme, isDark]);

  const value = useMemo(() => ({
    themeMode,        // 用户选择的模式：light/dark/system
    setThemeMode,     // 设置主题模式
    toggleTheme,      // 切换到下一个主题
    actualTheme,      // 实际应用的主题：light/dark
    isDark,           // 是否暗黑模式
    systemTheme,      // 系统主题
    isLight: !isDark,
    isSystem: themeMode === ThemeMode.SYSTEM,
  }), [themeMode, setThemeMode, toggleTheme, actualTheme, isDark, systemTheme]);

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
};

/**
 * 使用主题 Hook
 * @returns {object}
 */
export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

/**
 * 高阶组件：为类组件提供主题支持
 */
export const withTheme = (Component) => {
  return function WithThemeComponent(props) {
    const theme = useTheme();
    return <Component {...props} theme={theme} />;
  };
};

export default useTheme;
