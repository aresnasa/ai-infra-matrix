import React, { useState, useEffect, useRef } from 'react';
import { Card, Spin, Alert, Button, message } from 'antd';
import { ReloadOutlined, FullscreenOutlined } from '@ant-design/icons';
import { useI18n, onLanguageChange } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';
import '../App.css';

/**
 * MonitoringPage - 监控仪表板页面
 * 使用 iframe 嵌入 Nightingale 监控系统
 * 支持语言自动同步
 */
const MonitoringPage = () => {
  const { t, locale } = useI18n();
  const { isDark } = useTheme();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [iframeKey, setIframeKey] = useState(0);
  const iframeRef = useRef(null);

  // Nightingale 服务地址 - 使用环境变量或动态构建
  // 默认使用 nginx 代理路径 /nightingale/ 访问，支持 ProxyAuth SSO
  // 注意：Nightingale 根路径会显示 404，需要指定具体页面
  const getNightingaleUrl = () => {
    // 默认落地页：指标查询页面（metric/explorer）
    // Nightingale 的根路径 "/" 没有默认页面，会显示 404
    const defaultPath = process.env.REACT_APP_NIGHTINGALE_DEFAULT_PATH || 'metric/explorer';
    
    let baseUrl = '';
    
    // 优先使用完整的 URL 配置
    if (process.env.REACT_APP_NIGHTINGALE_URL) {
      baseUrl = process.env.REACT_APP_NIGHTINGALE_URL;
    } else {
      // 默认使用 nginx 代理路径（同域，避免跨域问题，支持 SSO）
      const currentPort = window.location.port ? `:${window.location.port}` : '';
      baseUrl = `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/${defaultPath}`;
    }
    
    // 添加语言参数，Nightingale 支持 lang=en 或 lang=zh
    // 添加主题参数，Nightingale 支持 themeMode=dark 或 themeMode=light
    const n9eLang = locale === 'en-US' ? 'en' : 'zh';
    const n9eTheme = isDark ? 'dark' : 'light';
    const separator = baseUrl.includes('?') ? '&' : '?';
    return `${baseUrl}${separator}lang=${n9eLang}&themeMode=${n9eTheme}`;
  };

  // 使用 useMemo 确保 URL 在 locale 或 isDark 变化时更新
  const nightingaleUrl = React.useMemo(() => {
    return getNightingaleUrl();
  }, [locale, isDark]);

  // 监听全局语言变化事件，自动刷新 Nightingale iframe
  useEffect(() => {
    const unsubscribe = onLanguageChange(({ newLocale, n9eLang }) => {
      console.log('[MonitoringPage] Language changed, refreshing Nightingale with lang:', n9eLang);
      setLoading(true);
      setIframeKey(prev => prev + 1);
      message.info(t('monitoring.languageSyncing'));
    });
    
    return unsubscribe;
  }, [t]);

  // 监听语言变化，刷新 iframe (保留原有逻辑作为备份)
  useEffect(() => {
    setIframeKey(prev => prev + 1);
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
    <div style={{ height: 'calc(100vh - 112px)', display: 'flex', flexDirection: 'column' }}>
      <Card 
        title={t('monitoring.title')}
        style={{ flex: 1, display: 'flex', flexDirection: 'column', height: '100%' }}
        bodyStyle={{ flex: 1, padding: 0, display: 'flex', flexDirection: 'column', height: '100%' }}
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
          key={iframeKey}
          src={nightingaleUrl}
          title="Nightingale Monitoring"
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          style={{
            width: '100%',
            height: '100%',
            border: 'none',
            display: loading ? 'none' : 'block'
          }}
          allow="fullscreen"
        />
      </Card>
    </div>
  );
};

export default MonitoringPage;
