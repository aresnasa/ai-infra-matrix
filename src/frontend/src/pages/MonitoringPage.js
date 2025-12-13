import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Card, Spin, Alert, Button, message } from 'antd';
import { ReloadOutlined, FullscreenOutlined } from '@ant-design/icons';
import { useI18n, onLanguageChange } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';
import '../App.css';

/**
 * MonitoringPage - 监控仪表板页面
 * 使用 iframe 嵌入 Nightingale 监控系统
 * 支持语言和主题自动同步
 */
const MonitoringPage = () => {
  const { t, locale } = useI18n();
  const { isDark } = useTheme();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [iframeKey, setIframeKey] = useState(0);
  const [iframeLanguage, setIframeLanguage] = useState(null); // 检测到的 iframe 语言状态
  const iframeRef = useRef(null);

  // 获取 Nightingale 语言代码
  const getN9eLang = useCallback(() => {
    return locale === 'en-US' ? 'en' : 'zh';
  }, [locale]);

  // 检测 iframe 中的语言状态
  const detectIframeLanguage = useCallback(() => {
    try {
      const iframe = iframeRef.current;
      if (!iframe || !iframe.contentWindow) {
        console.log('[MonitoringPage] iframe not ready for language detection');
        return null;
      }

      // 尝试通过 localStorage 检测 Nightingale 的语言设置
      // Nightingale 将语言存储在 localStorage 的 'language' 或 'locale' 键中
      try {
        const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
        
        // 检测页面中的语言标识（例如 html lang 属性或特定文本）
        const htmlLang = iframeDoc.documentElement.lang;
        if (htmlLang) {
          const detectedLang = htmlLang.startsWith('zh') ? 'zh' : 'en';
          console.log('[MonitoringPage] Detected iframe language from html lang:', detectedLang);
          setIframeLanguage(detectedLang);
          return detectedLang;
        }

        // 检测特定的中文或英文文本来判断语言
        const bodyText = iframeDoc.body?.innerText || '';
        if (bodyText.includes('仪表盘') || bodyText.includes('告警') || bodyText.includes('监控')) {
          console.log('[MonitoringPage] Detected Chinese content in iframe');
          setIframeLanguage('zh');
          return 'zh';
        } else if (bodyText.includes('Dashboard') || bodyText.includes('Alert') || bodyText.includes('Monitor')) {
          console.log('[MonitoringPage] Detected English content in iframe');
          setIframeLanguage('en');
          return 'en';
        }
      } catch (e) {
        // 跨域限制，无法访问 iframe 内容
        console.log('[MonitoringPage] Cannot access iframe content due to cross-origin restrictions');
      }

      return null;
    } catch (e) {
      console.error('[MonitoringPage] Error detecting iframe language:', e);
      return null;
    }
  }, []);

  // 尝试通过 postMessage 设置 iframe 中的语言
  const setIframeLanguageViaPostMessage = useCallback((lang) => {
    try {
      const iframe = iframeRef.current;
      if (!iframe || !iframe.contentWindow) {
        return;
      }

      // 发送语言切换消息到 iframe
      // Nightingale 前端需要监听这个消息来切换语言
      iframe.contentWindow.postMessage({
        type: 'SET_LANGUAGE',
        language: lang,
        source: 'ai-infra-matrix'
      }, '*');

      console.log('[MonitoringPage] Sent language change message to iframe:', lang);
    } catch (e) {
      console.error('[MonitoringPage] Error sending postMessage to iframe:', e);
    }
  }, []);

  // 使用 useMemo 确保 URL 在 locale 或 isDark 变化时更新
  const nightingaleUrl = React.useMemo(() => {
    const defaultPath = process.env.REACT_APP_NIGHTINGALE_DEFAULT_PATH || 'metric/explorer';
    
    let baseUrl = '';
    
    if (process.env.REACT_APP_NIGHTINGALE_URL) {
      baseUrl = process.env.REACT_APP_NIGHTINGALE_URL;
    } else {
      const currentPort = window.location.port ? `:${window.location.port}` : '';
      baseUrl = `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/${defaultPath}`;
    }
    
    // Nightingale 使用 en_US/zh_CN 格式的语言代码
    const n9eLang = locale === 'en-US' ? 'en' : 'zh';
    const n9eTheme = isDark ? 'dark' : 'light';
    const separator = baseUrl.includes('?') ? '&' : '?';
    
    console.log('[MonitoringPage] Generating URL with locale:', locale, 'isDark:', isDark, 'n9eLang:', n9eLang, 'n9eTheme:', n9eTheme);
    
    // 注意：nginx 配置中的 sub_filter 会注入脚本来读取这些参数并同步到 localStorage
    return `${baseUrl}${separator}lang=${n9eLang}&themeMode=${n9eTheme}`;
  }, [locale, isDark]);

  // 监听全局语言变化事件，自动刷新 Nightingale iframe
  useEffect(() => {
    const unsubscribe = onLanguageChange(({ newLocale, n9eLang }) => {
      console.log('[MonitoringPage] Language change event received:', newLocale, 'n9eLang:', n9eLang);
      // 首先尝试通过 postMessage 切换语言
      setIframeLanguageViaPostMessage(n9eLang);
      
      // 使用 setTimeout 确保 locale 状态已更新后再刷新 iframe
      setTimeout(() => {
        console.log('[MonitoringPage] Triggering iframe refresh after language change');
        setLoading(true);
        setIframeKey(prev => prev + 1);
        message.info(t('monitoring.languageSyncing'));
      }, 50);
    });
    
    return unsubscribe;
  }, [t, setIframeLanguageViaPostMessage]);

  // 监听语言变化，刷新 iframe - 这是主要的刷新逻辑
  useEffect(() => {
    console.log('[MonitoringPage] locale changed to:', locale, '- refreshing iframe');
    // 使用 setTimeout 确保 useMemo 已重新计算 URL
    const timer = setTimeout(() => {
      setLoading(true);
      setIframeKey(prev => prev + 1);
    }, 100);
    return () => clearTimeout(timer);
  }, [locale]);

  // 监听主题变化，刷新 iframe 以同步暗色模式
  useEffect(() => {
    console.log('[MonitoringPage] Theme changed, refreshing Nightingale with themeMode:', isDark ? 'dark' : 'light');
    setLoading(true);
    setIframeKey(prev => prev + 1);
  }, [isDark]);

  useEffect(() => {
    // 组件挂载时的初始化
    console.log('MonitoringPage mounted, Nightingale URL:', nightingaleUrl);
    
    // 设置超时检测iframe加载
    const timeout = setTimeout(() => {
      if (loading) {
        setLoading(false);
        setError(t('monitoring.loadTimeout'));
      }
    }, 15000); // 15秒超时

    return () => clearTimeout(timeout);
  }, [nightingaleUrl, loading, t]);

  // iframe 加载完成处理
  const handleIframeLoad = () => {
    console.log('Nightingale iframe loaded successfully');
    setLoading(false);
    setError(null);
    
    // 延迟检测 iframe 语言状态
    setTimeout(() => {
      const detectedLang = detectIframeLanguage();
      const expectedLang = getN9eLang();
      
      if (detectedLang && detectedLang !== expectedLang) {
        console.log('[MonitoringPage] Language mismatch detected! Expected:', expectedLang, 'Got:', detectedLang);
        // 尝试通过 postMessage 修正语言
        setIframeLanguageViaPostMessage(expectedLang);
      }
    }, 1000);
  };

  // iframe 加载错误处理
  const handleIframeError = () => {
    console.error('Failed to load Nightingale iframe');
    setLoading(false);
    setError(t('monitoring.loadFailed'));
  };

  // 刷新 iframe
  const handleRefresh = () => {
    setLoading(true);
    setError(null);
    setIframeKey(prev => prev + 1);
  };

  // 全屏打开
  const handleFullscreen = () => {
    window.open(nightingaleUrl, '_blank');
  };

  return (
    <div style={{ 
      height: 'calc(100vh - 112px)', 
      display: 'flex', 
      flexDirection: 'column',
      backgroundColor: isDark ? '#001529' : 'transparent', // 暗色模式深蓝色背景
    }}>
      <Card 
        title={t('monitoring.title')}
        style={{ 
          flex: 1, 
          display: 'flex', 
          flexDirection: 'column', 
          height: '100%',
          backgroundColor: isDark ? '#001529' : '#fff', // 暗色模式深蓝色背景
          borderColor: isDark ? '#1d39c4' : undefined, // 暗色模式边框颜色
        }}
        bodyStyle={{ 
          flex: 1, 
          padding: 0, 
          display: 'flex', 
          flexDirection: 'column', 
          height: '100%',
          backgroundColor: isDark ? '#001529' : 'transparent', // 暗色模式深蓝色背景
        }}
        headStyle={{
          backgroundColor: isDark ? '#001529' : undefined, // 暗色模式深蓝色背景
          borderBottomColor: isDark ? '#1d39c4' : undefined, // 暗色模式边框颜色
          color: isDark ? '#fff' : undefined,
        }}
        extra={
          <div>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={handleRefresh}
              style={{ marginRight: 8 }}
            >
              {t('monitoring.refresh')}
            </Button>
            <Button 
              icon={<FullscreenOutlined />} 
              onClick={handleFullscreen}
            >
              {t('monitoring.openNewWindow')}
            </Button>
          </div>
        }
      >
        {error && (
          <Alert
            message={t('monitoring.loadError')}
            description={error}
            type="error"
            showIcon
            closable
            onClose={() => setError(null)}
            style={{ margin: '16px' }}
          />
        )}

        {loading && (
          <div style={{ 
            display: 'flex', 
            justifyContent: 'center', 
            alignItems: 'center', 
            height: '100%',
            flexDirection: 'column'
          }}>
            <Spin size="large" />
            <div style={{ marginTop: 16, color: '#999' }}>
              {t('monitoring.loadingSystem')}
            </div>
          </div>
        )}

        <iframe
          ref={iframeRef}
          key={iframeKey}
          src={nightingaleUrl}
          title="Nightingale Monitoring"
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          style={{
            width: '100%',
            height: '100%',
            border: 'none',
            display: loading ? 'none' : 'block',
            backgroundColor: isDark ? '#001529' : '#ffffff', // 暗色模式使用深蓝色背景
          }}
          allow="fullscreen"
        />
      </Card>
    </div>
  );
};

export default MonitoringPage;
