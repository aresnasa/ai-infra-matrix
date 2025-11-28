/**
 * 国际化 Hook
 * 提供语言切换和文本翻译功能
 */

import React, { createContext, useContext, useState, useCallback, useMemo, useEffect } from 'react';
import { locales, defaultLocale, languageNames } from '../locales';

// 语言存储 Key
const LANGUAGE_STORAGE_KEY = 'ai_infra_language';

// 创建上下文
const I18nContext = createContext(null);

/**
 * 获取嵌套对象的值
 * @param {object} obj - 对象
 * @param {string} path - 路径，如 'common.loading'
 * @returns {string}
 */
const getNestedValue = (obj, path) => {
  return path.split('.').reduce((current, key) => {
    return current && current[key] !== undefined ? current[key] : null;
  }, obj);
};

/**
 * 模板字符串替换
 * @param {string} template - 模板字符串，如 '共 {count} 条'
 * @param {object} params - 参数对象，如 { count: 10 }
 * @returns {string}
 */
const interpolate = (template, params = {}) => {
  if (!template || typeof template !== 'string') return template;
  
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    return params[key] !== undefined ? params[key] : match;
  });
};

/**
 * I18n Provider 组件
 */
export const I18nProvider = ({ children, initialLocale }) => {
  // 从 localStorage 获取保存的语言，默认使用中文
  const [locale, setLocaleState] = useState(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem(LANGUAGE_STORAGE_KEY);
      if (saved && locales[saved]) {
        return saved;
      }
    }
    return initialLocale || defaultLocale;
  });

  // 当前语言包
  const messages = useMemo(() => locales[locale] || locales[defaultLocale], [locale]);

  // 切换语言
  const setLocale = useCallback((newLocale) => {
    if (locales[newLocale]) {
      setLocaleState(newLocale);
      if (typeof window !== 'undefined') {
        localStorage.setItem(LANGUAGE_STORAGE_KEY, newLocale);
        // 更新 HTML lang 属性
        document.documentElement.lang = newLocale;
      }
    }
  }, []);

  // 翻译函数
  const t = useCallback((key, params) => {
    const value = getNestedValue(messages, key);
    if (value === null) {
      console.warn(`[i18n] Missing translation for key: ${key}`);
      return key;
    }
    return interpolate(value, params);
  }, [messages]);

  // 检查是否有翻译
  const hasTranslation = useCallback((key) => {
    return getNestedValue(messages, key) !== null;
  }, [messages]);

  // 获取所有可用语言
  const availableLocales = useMemo(() => {
    return Object.keys(locales).map(key => ({
      key,
      name: languageNames[key] || key,
    }));
  }, []);

  // 初始化时设置 HTML lang 属性
  useEffect(() => {
    if (typeof window !== 'undefined') {
      document.documentElement.lang = locale;
    }
  }, [locale]);

  const value = useMemo(() => ({
    locale,
    setLocale,
    t,
    hasTranslation,
    messages,
    availableLocales,
    isZhCN: locale === 'zh-CN',
    isEnUS: locale === 'en-US',
  }), [locale, setLocale, t, hasTranslation, messages, availableLocales]);

  return (
    <I18nContext.Provider value={value}>
      {children}
    </I18nContext.Provider>
  );
};

/**
 * 使用国际化 Hook
 * @returns {object}
 */
export const useI18n = () => {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error('useI18n must be used within an I18nProvider');
  }
  return context;
};

/**
 * 高阶组件：为类组件提供国际化支持
 */
export const withI18n = (Component) => {
  return function WithI18nComponent(props) {
    const i18n = useI18n();
    return <Component {...props} i18n={i18n} />;
  };
};

/**
 * 翻译组件：用于在 JSX 中直接翻译
 * @example <Trans i18nKey="common.loading" />
 * @example <Trans i18nKey="table.total" params={{ total: 100 }} />
 */
export const Trans = ({ i18nKey, params, children }) => {
  const { t } = useI18n();
  return <>{t(i18nKey, params) || children}</>;
};

export default useI18n;
